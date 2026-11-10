package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// 归档隐藏列表的落盘格式版本。读取到其他版本按损坏处理：重建为空。
const archiveVersion = 1

// archiveRecord 是一条隐藏记录。会话标识由服务端给出；该会话不存在时这条记录
// 不产生任何可见影响，也不会阻塞删除。
type archiveRecord struct {
	SessionID string    `json:"sessionId"`
	HiddenAt  time.Time `json:"hiddenAt"`
}

// archiveFile 是落盘结构。
type archiveFile struct {
	Version int             `json:"version"`
	Hidden  []archiveRecord `json:"hidden"`
}

// Archive 是本地归档隐藏列表：只影响目录列表的整洁，不写回服务端，也不改变任何
// 会话事实。写入是原子替换；文件损坏或版本不认识时按空列表处理。
type Archive struct {
	path string

	mu      sync.Mutex
	entries map[string]archiveRecord
	rebuilt bool
}

// OpenArchive 打开一个归档隐藏列表文件；文件不存在时得到空列表。
// 内容损坏或版本不认识时重建为空且不报错，只记下 rebuilt，供诊断使用。
func OpenArchive(path string) (*Archive, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("归档隐藏列表缺少文件路径")
	}
	archive := &Archive{path: path, entries: make(map[string]archiveRecord)}

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return archive, nil
		}
		return nil, fmt.Errorf("读取归档隐藏列表: %w", err)
	}

	var file archiveFile
	if err := json.Unmarshal(raw, &file); err != nil {
		archive.rebuilt = true
		return archive, nil
	}
	if file.Version != archiveVersion {
		archive.rebuilt = true
		return archive, nil
	}
	for _, record := range file.Hidden {
		if record.SessionID == "" {
			continue
		}
		if _, duplicate := archive.entries[record.SessionID]; duplicate {
			continue
		}
		archive.entries[record.SessionID] = record
	}
	return archive, nil
}

// Path 返回归档隐藏列表的文件路径。
func (a *Archive) Path() string { return a.path }

// Rebuilt 表示打开时文件损坏或版本不认识，列表已被重建为空。
func (a *Archive) Rebuilt() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.rebuilt
}

// Hide 把一个会话加入隐藏列表并原子落盘。重复隐藏不改变文件。
func (a *Archive) Hide(sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("%w: 归档缺少会话标识", ErrInvalidRequest)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.entries[sessionID]; exists {
		return nil
	}
	next := a.cloneLocked()
	next[sessionID] = archiveRecord{SessionID: sessionID, HiddenAt: time.Now().UTC()}
	if err := a.persistLocked(next); err != nil {
		return err
	}
	a.entries = next
	a.rebuilt = false
	return nil
}

// Unhide 把一个会话移出隐藏列表并原子落盘。未知标识不改变文件。
func (a *Archive) Unhide(sessionID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.entries[sessionID]; !exists {
		return nil
	}
	next := a.cloneLocked()
	delete(next, sessionID)
	if err := a.persistLocked(next); err != nil {
		return err
	}
	a.entries = next
	a.rebuilt = false
	return nil
}

// Hidden 报告一个会话是否被隐藏。
func (a *Archive) Hidden(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_, exists := a.entries[sessionID]
	return exists
}

// HiddenIDs 返回全部隐藏的会话标识，按标识排序，供诊断与测试使用。
func (a *Archive) HiddenIDs() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := make([]string, 0, len(a.entries))
	for id := range a.entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (a *Archive) cloneLocked() map[string]archiveRecord {
	next := make(map[string]archiveRecord, len(a.entries)+1)
	for id, record := range a.entries {
		next[id] = record
	}
	return next
}

// persistLocked 把下一版状态原子写盘：同目录临时文件 → 同步 → 替换。
// 任一步失败都不会动原文件。
func (a *Archive) persistLocked(entries map[string]archiveRecord) error {
	file := archiveFile{Version: archiveVersion, Hidden: make([]archiveRecord, 0, len(entries))}
	ids := make([]string, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		file.Hidden = append(file.Hidden, entries[id])
	}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("编码归档隐藏列表: %w", err)
	}
	raw = append(raw, '\n')
	if err := writeFileAtomic(a.path, raw); err != nil {
		return fmt.Errorf("写入归档隐藏列表: %w", err)
	}
	return nil
}

// writeFileAtomic 用同目录临时文件加替换完成一次原子写，文件权限 0600。
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	discard := func() {
		_ = tmp.Close()
		_ = os.Remove(name)
	}
	if err := tmp.Chmod(0o600); err != nil {
		discard()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		discard()
		return err
	}
	if err := tmp.Sync(); err != nil {
		discard()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	// 同目录 rename 是原子替换；不额外同步目录，失败时原文件保持完整。
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}
