package attachments

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Store 是暂存区的唯一入口。同一 Store 可被并发调用；跨进程并发不在本版范围内。
type Store struct {
	root    string
	blobDir string
	limits  Limits

	mu      sync.Mutex
	entries []Entry
}

// indexFile 是索引文件的落盘结构。版本号决定格式与类别语义的兼容范围。
type indexFile struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

// Open 打开应用数据目录下的暂存区，不存在时创建。暂存区固定位于 dataDir 下的
// attachment-staging 子目录；打开时会把暂存区目录收紧为 0700。
func Open(dataDir string, limits Limits) (*Store, error) {
	if strings.TrimSpace(dataDir) == "" {
		return nil, errors.New("应用数据目录为空")
	}
	if err := limits.validate(); err != nil {
		return nil, err
	}
	root := filepath.Join(dataDir, DirName)
	if err := ensureDir(root); err != nil {
		return nil, err
	}
	blobDir := filepath.Join(root, blobDirName)
	if err := ensureDir(blobDir); err != nil {
		return nil, err
	}
	s := &Store{root: root, blobDir: blobDir, limits: limits}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadIndexLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

// ensureDir 创建目录并收紧权限到 0700。
func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("创建暂存目录失败：%w", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("设置暂存目录权限失败：%w", err)
	}
	return nil
}

// Dir 返回暂存区根目录，供调用方诊断与测试使用。
func (s *Store) Dir() string { return s.root }

// Limits 返回当前使用的上限配置。
func (s *Store) Limits() Limits { return s.limits }

// Stage 把本地文件复制进暂存区并返回暂存项。同一内容（SHA-256 摘要相同）只保留一份，
// 重复暂存返回同一标识并把引用计数加一；重名的来源保留首次暂存时的文件名。
// 上限在暂存时判定，超限直接拒绝，不做截断。
func (s *Store) Stage(path string) (Entry, error) {
	content, err := s.readSource(path)
	if err != nil {
		return Entry{}, err
	}

	sum := sha256.Sum256(content.data)
	digest := hex.EncodeToString(sum[:])

	s.mu.Lock()
	defer s.mu.Unlock()

	if i := s.indexOfDigestLocked(digest); i >= 0 {
		e := s.entries[i]
		// 内容文件可能缺失或被外部改写，重复挂载时以本次读到的内容为准修复。
		if err := s.writeBlobLocked(digest, content.data); err != nil {
			return Entry{}, err
		}
		e.Size = int64(len(content.data))
		e.Category = content.category
		e.Refs++
		s.entries[i] = e
		if err := s.saveIndexLocked(); err != nil {
			e.Refs--
			s.entries[i] = e
			return Entry{}, err
		}
		return e, nil
	}

	// 先回收无引用的旧项，避免已释放的内容继续占用总大小额度。
	if err := s.reclaimLocked(); err != nil {
		return Entry{}, err
	}

	used := s.totalSizeLocked()
	if used+int64(len(content.data)) > s.limits.Total {
		return Entry{}, &LimitError{
			Kind:     LimitKindTotal,
			Category: content.category,
			Size:     int64(len(content.data)),
			Current:  used,
			Limit:    s.limits.Total,
		}
	}

	id, err := newID()
	if err != nil {
		return Entry{}, err
	}
	if err := s.writeBlobLocked(digest, content.data); err != nil {
		return Entry{}, err
	}
	e := Entry{
		ID:       id,
		Digest:   digest,
		Name:     content.name,
		Category: content.category,
		Size:     int64(len(content.data)),
		Refs:     1,
		StagedAt: time.Now().UTC(),
	}
	s.entries = append(s.entries, e)
	if err := s.saveIndexLocked(); err != nil {
		s.entries = s.entries[:len(s.entries)-1]
		return Entry{}, err
	}
	return e, nil
}

// stagedContent 是一次读取来源文件的结果。
type stagedContent struct {
	name     string
	category Category
	data     []byte
}

// readSource 读取并判定来源文件，在读取全量内容前完成上限判定。
func (s *Store) readSource(path string) (stagedContent, error) {
	if strings.TrimSpace(path) == "" {
		return stagedContent{}, errors.New("附件路径为空")
	}
	f, err := os.Open(path)
	if err != nil {
		return stagedContent{}, fmt.Errorf("读取附件失败：%w", err)
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return stagedContent{}, fmt.Errorf("读取附件失败：%w", err)
	}
	if !st.Mode().IsRegular() {
		return stagedContent{}, fmt.Errorf("%w：%s", ErrNotRegular, path)
	}
	name := filepath.Base(path)

	header := make([]byte, sniffLen)
	n, err := io.ReadFull(f, header)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return stagedContent{}, fmt.Errorf("读取附件失败：%w", err)
	}
	header = header[:n]

	// 先按文件头决定适用哪一档单文件上限，超限时不必读入全量内容。
	headerCategory, reason, ok := classifyHeader(header)
	if !ok {
		return stagedContent{}, &UnsupportedError{Name: name, Size: st.Size(), Reason: reason}
	}
	limit := s.limits.perFile(headerCategory)
	if st.Size() > limit {
		return stagedContent{}, &LimitError{
			Kind:     LimitKindFile,
			Category: headerCategory,
			Size:     st.Size(),
			Limit:    limit,
		}
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return stagedContent{}, fmt.Errorf("读取附件失败：%w", err)
	}
	// 多读一个字节用于识别「读取期间文件变大」的越界情况。
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return stagedContent{}, fmt.Errorf("读取附件失败：%w", err)
	}
	if int64(len(data)) > limit {
		size := int64(len(data))
		// 读取期间文件可能继续变大，错误里尽量给出实际字节数而不是探测上限。
		if after, err := f.Stat(); err == nil && after.Size() > size {
			size = after.Size()
		}
		return stagedContent{}, &LimitError{
			Kind:     LimitKindFile,
			Category: headerCategory,
			Size:     size,
			Limit:    limit,
		}
	}

	// 类别以完整内容为准；文件头与完整内容的判定不一致时，按类别复核上限。
	category, reason, ok := classifyContent(data)
	if !ok {
		return stagedContent{}, &UnsupportedError{Name: name, Size: int64(len(data)), Reason: reason}
	}
	if category != headerCategory {
		limit = s.limits.perFile(category)
		if int64(len(data)) > limit {
			return stagedContent{}, &LimitError{
				Kind:     LimitKindFile,
				Category: category,
				Size:     int64(len(data)),
				Limit:    limit,
			}
		}
	}

	return stagedContent{name: name, category: category, data: data}, nil
}

// Get 按标识取暂存项元数据。
func (s *Store) Get(id string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.indexOfIDLocked(id)
	if i < 0 {
		return Entry{}, ErrNotFound
	}
	return s.entries[i], nil
}

// List 返回全部暂存项元数据的副本，包括已失效但尚未清理的项。
func (s *Store) List() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out, nil
}

// Read 读取暂存项内容。返回的是暂存区副本，不受来源文件后续变化影响。
func (s *Store) Read(id string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.indexOfIDLocked(id)
	if i < 0 {
		return nil, ErrNotFound
	}
	data, err := os.ReadFile(s.blobPath(s.entries[i].Digest))
	if err != nil {
		return nil, fmt.Errorf("读取暂存内容失败：%w", err)
	}
	return data, nil
}

// Release 释放一次挂载引用。引用计数归零后暂存项仍然可读，等待清理回收；
// 重复释放不改变状态。
func (s *Store) Release(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.indexOfIDLocked(id)
	if i < 0 {
		return ErrNotFound
	}
	e := s.entries[i]
	if e.Refs == 0 {
		return nil
	}
	e.Refs--
	s.entries[i] = e
	return s.saveIndexLocked()
}

// Validate 校验全部暂存项，逐项给出是否失效与失效原因。重启后可用它标注失效项。
func (s *Store) Validate() ([]Check, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	checks := make([]Check, 0, len(s.entries))
	for _, e := range s.entries {
		checks = append(checks, s.checkLocked(e))
	}
	return checks, nil
}

// Cleanup 清理暂存区：删除无引用的暂存项（有效或失效）与无索引记录的内容文件。
// 仍被引用的失效项只报告、不删除。清理先落索引再删文件，中途失败留下的内容文件
// 会在下一次清理中按孤儿文件回收。
func (s *Store) Cleanup() (CleanupReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var report CleanupReport
	kept := make([]Entry, 0, len(s.entries))
	keptDigests := make(map[string]bool, len(s.entries))
	removeDigests := make(map[string]bool)
	changed := false

	for _, e := range s.entries {
		if e.Refs > 0 {
			kept = append(kept, e)
			keptDigests[e.Digest] = true
			if check := s.checkLocked(e); !check.OK() {
				report.KeptReferenced = append(report.KeptReferenced, check)
			}
			continue
		}
		changed = true
		if check := s.checkLocked(e); check.OK() {
			report.RemovedUnreferenced = append(report.RemovedUnreferenced, e.ID)
		} else {
			report.RemovedInvalid = append(report.RemovedInvalid, e.ID)
		}
		removeDigests[e.Digest] = true
	}

	if changed {
		s.entries = kept
		if err := s.saveIndexLocked(); err != nil {
			return report, err
		}
	}

	for digest := range removeDigests {
		if keptDigests[digest] {
			continue
		}
		path := s.blobPath(digest)
		if st, err := os.Stat(path); err == nil {
			report.BytesFreed += st.Size()
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return report, fmt.Errorf("删除暂存内容失败：%w", err)
		}
	}

	items, err := os.ReadDir(s.blobDir)
	if err != nil {
		return report, fmt.Errorf("扫描暂存内容目录失败：%w", err)
	}
	for _, item := range items {
		name := item.Name()
		if item.IsDir() || keptDigests[name] || removeDigests[name] {
			continue
		}
		path := filepath.Join(s.blobDir, name)
		if st, err := os.Stat(path); err == nil {
			report.BytesFreed += st.Size()
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			continue
		}
		report.RemovedOrphans = append(report.RemovedOrphans, name)
	}
	return report, nil
}

// Reset 清空暂存区：删除全部索引记录与内容文件，保留目录与权限。
func (s *Store) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = nil
	if err := s.saveIndexLocked(); err != nil {
		return err
	}
	items, err := os.ReadDir(s.blobDir)
	if err != nil {
		return fmt.Errorf("扫描暂存内容目录失败：%w", err)
	}
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(s.blobDir, item.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("删除暂存内容失败：%w", err)
		}
	}
	return nil
}

// checkLocked 校验单条记录。调用方必须持有锁。
func (s *Store) checkLocked(e Entry) Check {
	if err := validateEntry(e); err != nil {
		return Check{Entry: e, Problem: ProblemRecordInvalid, Detail: err.Error()}
	}
	path := s.blobPath(e.Digest)
	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Check{Entry: e, Problem: ProblemBlobMissing, Detail: "内容文件缺失"}
		}
		return Check{Entry: e, Problem: ProblemBlobUnreadable, Detail: err.Error()}
	}
	if !st.Mode().IsRegular() {
		return Check{Entry: e, Problem: ProblemBlobNotRegular, Detail: "内容路径不是普通文件"}
	}
	if st.Size() != e.Size {
		return Check{
			Entry:   e,
			Problem: ProblemSizeMismatch,
			Detail:  fmt.Sprintf("内容文件 %d 字节，记录 %d 字节", st.Size(), e.Size),
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Check{Entry: e, Problem: ProblemBlobUnreadable, Detail: err.Error()}
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != e.Digest {
		return Check{Entry: e, Problem: ProblemDigestMismatch, Detail: "内容摘要与记录不一致"}
	}
	category, _, ok := classifyContent(data)
	if !ok || category != e.Category {
		return Check{Entry: e, Problem: ProblemCategoryMismatch, Detail: "内容类别与记录不一致"}
	}
	return Check{Entry: e, Problem: ProblemNone}
}

// reclaimLocked 删除引用计数为零的记录及其内容文件。调用方必须持有锁。
func (s *Store) reclaimLocked() error {
	hasZero := false
	for _, e := range s.entries {
		if e.Refs == 0 {
			hasZero = true
			break
		}
	}
	if !hasZero {
		return nil
	}
	kept := make([]Entry, 0, len(s.entries))
	keptDigests := make(map[string]bool, len(s.entries))
	removed := make([]string, 0)
	for _, e := range s.entries {
		if e.Refs > 0 {
			kept = append(kept, e)
			keptDigests[e.Digest] = true
			continue
		}
		removed = append(removed, e.Digest)
	}
	s.entries = kept
	if err := s.saveIndexLocked(); err != nil {
		return err
	}
	for _, digest := range removed {
		if keptDigests[digest] {
			continue
		}
		// 删除失败的内容文件会作为孤儿由清理回收。
		_ = os.Remove(s.blobPath(digest))
	}
	return nil
}

func (s *Store) totalSizeLocked() int64 {
	var total int64
	for _, e := range s.entries {
		total += e.Size
	}
	return total
}

func (s *Store) blobPath(digest string) string {
	return filepath.Join(s.blobDir, digest)
}

func (s *Store) indexOfIDLocked(id string) int {
	for i, e := range s.entries {
		if e.ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) indexOfDigestLocked(digest string) int {
	for i, e := range s.entries {
		if e.Digest == digest {
			return i
		}
	}
	return -1
}

// writeBlobLocked 把内容写到以摘要命名的内容文件。已存在且内容一致时不重写；
// 长度一致但内容不一致的旧文件会被覆盖。调用方必须持有锁。
func (s *Store) writeBlobLocked(digest string, data []byte) error {
	path := s.blobPath(digest)
	if st, err := os.Stat(path); err == nil && st.Mode().IsRegular() && st.Size() == int64(len(data)) {
		if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, data) {
			return nil
		}
	}

	tmp, err := os.CreateTemp(s.blobDir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("写入暂存内容失败：%w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}
	defer cleanup()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("设置暂存内容权限失败：%w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入暂存内容失败：%w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入暂存内容失败：%w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("写入暂存内容失败：%w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("写入暂存内容失败：%w", err)
	}
	tmpName = ""
	syncDirBestEffort(s.blobDir)
	return nil
}

// loadIndexLocked 读取并校验索引。调用方必须持有锁。
func (s *Store) loadIndexLocked() error {
	path := filepath.Join(s.root, indexFileName)
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.entries = nil
			return nil
		}
		return fmt.Errorf("读取暂存索引失败：%w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("设置暂存索引权限失败：%w", err)
	}

	var file indexFile
	if err := json.Unmarshal(payload, &file); err != nil {
		return fmt.Errorf("%w：%v", ErrIndexCorrupt, err)
	}
	if file.Version != indexVersion {
		return fmt.Errorf("%w：索引版本 %d，当前支持 %d", ErrIndexVersion, file.Version, indexVersion)
	}
	seenID := make(map[string]bool, len(file.Entries))
	seenDigest := make(map[string]bool, len(file.Entries))
	for _, e := range file.Entries {
		if err := validateEntry(e); err != nil {
			return fmt.Errorf("%w：%v", ErrIndexCorrupt, err)
		}
		if seenID[e.ID] {
			return fmt.Errorf("%w：暂存标识 %s 重复", ErrIndexCorrupt, e.ID)
		}
		if seenDigest[e.Digest] {
			return fmt.Errorf("%w：摘要 %s 重复", ErrIndexCorrupt, e.Digest)
		}
		seenID[e.ID] = true
		seenDigest[e.Digest] = true
	}
	s.entries = file.Entries
	return nil
}

// saveIndexLocked 原子写索引：先写临时文件并同步，再改名覆盖。调用方必须持有锁。
func (s *Store) saveIndexLocked() error {
	entries := s.entries
	if entries == nil {
		entries = []Entry{}
	}
	payload, err := json.MarshalIndent(indexFile{Version: indexVersion, Entries: entries}, "", "  ")
	if err != nil {
		return fmt.Errorf("写入暂存索引失败：%w", err)
	}
	payload = append(payload, '\n')

	tmp, err := os.CreateTemp(s.root, ".index-*")
	if err != nil {
		return fmt.Errorf("写入暂存索引失败：%w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("设置暂存索引权限失败：%w", err)
	}
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入暂存索引失败：%w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入暂存索引失败：%w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("写入暂存索引失败：%w", err)
	}
	if err := os.Rename(tmpName, filepath.Join(s.root, indexFileName)); err != nil {
		return fmt.Errorf("写入暂存索引失败：%w", err)
	}
	tmpName = ""
	syncDirBestEffort(s.root)
	return nil
}

// validateEntry 做索引记录的结构校验；内容层面的一致性由 checkLocked 负责。
func validateEntry(e Entry) error {
	if e.ID == "" {
		return errors.New("暂存项缺少标识")
	}
	if !validDigest(e.Digest) {
		return fmt.Errorf("暂存项 %s 的摘要不合法", e.ID)
	}
	if !e.Category.valid() {
		return fmt.Errorf("暂存项 %s 的类别不合法", e.ID)
	}
	if e.Size < 0 {
		return fmt.Errorf("暂存项 %s 的字节数为负", e.ID)
	}
	if e.Refs < 0 {
		return fmt.Errorf("暂存项 %s 的引用计数为负", e.ID)
	}
	if e.Name == "" {
		return fmt.Errorf("暂存项 %s 缺少文件名", e.ID)
	}
	if e.StagedAt.IsZero() {
		return fmt.Errorf("暂存项 %s 缺少暂存时间", e.ID)
	}
	return nil
}

// validDigest 检查是否为 64 位十六进制小写摘要。
func validDigest(digest string) bool {
	if len(digest) != sha256.Size*2 {
		return false
	}
	if strings.ToLower(digest) != digest {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil
}

// newID 生成 128 位随机暂存标识。
func newID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成暂存标识失败：%w", err)
	}
	return hex.EncodeToString(buf), nil
}

// syncDirBestEffort 尽力同步目录项；部分文件系统不支持目录同步，失败不影响结果。
func syncDirBestEffort(dir string) {
	f, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = f.Sync()
	_ = f.Close()
}
