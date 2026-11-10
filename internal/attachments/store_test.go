package attachments

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// newTestStore 创建独立暂存区并返回它。
func newTestStore(t *testing.T, limits Limits) *Store {
	t.Helper()
	store, err := Open(t.TempDir(), limits)
	if err != nil {
		t.Fatalf("打开暂存区失败：%v", err)
	}
	return store
}

// writeSource 写一个来源文件并返回路径。
func writeSource(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("写入来源文件失败：%v", err)
	}
	return path
}

// blobEntries 返回内容目录下的文件列表。
func blobEntries(t *testing.T, store *Store) []os.DirEntry {
	t.Helper()
	items, err := os.ReadDir(filepath.Join(store.Dir(), blobDirName))
	if err != nil {
		t.Fatalf("读取内容目录失败：%v", err)
	}
	return items
}

// writeIndexFile 直接覆盖索引文件，用于构造损坏或异常索引。
func writeIndexFile(t *testing.T, store *Store, file indexFile) {
	t.Helper()
	payload, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("序列化索引失败：%v", err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir(), indexFileName), payload, 0o600); err != nil {
		t.Fatalf("写入索引失败：%v", err)
	}
}

func TestStageTextAttachments(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	content := []byte("你好，附件。\nsecond line\n")
	src := writeSource(t, "notes.txt", content)

	entry, err := store.Stage(src)
	if err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	if entry.ID == "" {
		t.Fatal("暂存标识为空")
	}
	if entry.Name != "notes.txt" {
		t.Fatalf("文件名 = %q，期望 notes.txt", entry.Name)
	}
	if entry.Category != CategoryText {
		t.Fatalf("类别 = %q，期望 text", entry.Category)
	}
	if entry.Size != int64(len(content)) {
		t.Fatalf("字节数 = %d，期望 %d", entry.Size, len(content))
	}
	sum := sha256.Sum256(content)
	if entry.Digest != hex.EncodeToString(sum[:]) {
		t.Fatalf("摘要 = %q，与内容 SHA-256 不一致", entry.Digest)
	}
	if entry.Refs != 1 {
		t.Fatalf("引用计数 = %d，期望 1", entry.Refs)
	}

	got, err := store.Read(entry.ID)
	if err != nil {
		t.Fatalf("读取暂存内容失败：%v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatal("暂存内容与来源内容不一致")
	}

	entries, err := store.List()
	if err != nil {
		t.Fatalf("列出暂存项失败：%v", err)
	}
	if len(entries) != 1 || entries[0].ID != entry.ID {
		t.Fatalf("暂存项列表 = %#v，期望只含 %s", entries, entry.ID)
	}

	checks, err := store.Validate()
	if err != nil {
		t.Fatalf("校验失败：%v", err)
	}
	if len(checks) != 1 || !checks[0].OK() {
		t.Fatalf("校验结果 = %#v，期望有效", checks)
	}
}

func TestStageCopiesContent(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	content := []byte("原始内容")
	src := writeSource(t, "note.txt", content)

	entry, err := store.Stage(src)
	if err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	if err := os.WriteFile(src, []byte("来源已被改写的更长的内容"), 0o600); err != nil {
		t.Fatalf("改写来源文件失败：%v", err)
	}
	got, err := store.Read(entry.ID)
	if err != nil {
		t.Fatalf("读取暂存内容失败：%v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("暂存内容 = %q，期望保持原始内容", got)
	}
}

func TestStageRejectsOversizeFile(t *testing.T) {
	limits := DefaultLimits()
	limits.Text = 32
	store := newTestStore(t, limits)
	src := writeSource(t, "big.txt", bytes.Repeat([]byte("a"), 33))

	_, err := store.Stage(src)
	var limitErr *LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("错误 = %v，期望 *LimitError", err)
	}
	if limitErr.Kind != LimitKindFile {
		t.Fatalf("上限类别 = %q，期望 file", limitErr.Kind)
	}
	if limitErr.Category != CategoryText {
		t.Fatalf("附件类别 = %q，期望 text", limitErr.Category)
	}
	if limitErr.Size != 33 || limitErr.Limit != 32 {
		t.Fatalf("字节数/上限 = %d/%d，期望 33/32", limitErr.Size, limitErr.Limit)
	}
	msg := err.Error()
	if !strings.Contains(msg, "33") || !strings.Contains(msg, "32") {
		t.Fatalf("错误信息 %q 未同时给出字节数与上限", msg)
	}

	entries, _ := store.List()
	if len(entries) != 0 {
		t.Fatalf("超限后仍有 %d 条暂存项", len(entries))
	}
	if items := blobEntries(t, store); len(items) != 0 {
		t.Fatalf("超限后内容目录仍有 %d 个文件", len(items))
	}
}

func TestStageAcceptsExactlyAtLimit(t *testing.T) {
	limits := DefaultLimits()
	limits.Text = 10
	store := newTestStore(t, limits)
	content := []byte("0123456789")
	src := writeSource(t, "edge.txt", content)

	entry, err := store.Stage(src)
	if err != nil {
		t.Fatalf("恰好达到上限时暂存失败：%v", err)
	}
	if entry.Size != int64(len(content)) {
		t.Fatalf("字节数 = %d，期望 %d", entry.Size, len(content))
	}
	got, err := store.Read(entry.ID)
	if err != nil {
		t.Fatalf("读取暂存内容失败：%v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatal("恰好达到上限时内容被截断")
	}
}

func TestStageRejectsOversizeTotal(t *testing.T) {
	limits := DefaultLimits()
	limits.Text = 100
	limits.Total = 150
	store := newTestStore(t, limits)

	first := writeSource(t, "a.txt", bytes.Repeat([]byte("a"), 100))
	if _, err := store.Stage(first); err != nil {
		t.Fatalf("首次暂存失败：%v", err)
	}
	second := writeSource(t, "b.txt", bytes.Repeat([]byte("b"), 100))

	_, err := store.Stage(second)
	var limitErr *LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("错误 = %v，期望 *LimitError", err)
	}
	if limitErr.Kind != LimitKindTotal {
		t.Fatalf("上限类别 = %q，期望 total", limitErr.Kind)
	}
	if limitErr.Size != 100 || limitErr.Current != 100 || limitErr.Limit != 150 {
		t.Fatalf("本次/已占用/上限 = %d/%d/%d，期望 100/100/150",
			limitErr.Size, limitErr.Current, limitErr.Limit)
	}
	msg := err.Error()
	if !strings.Contains(msg, "200") || !strings.Contains(msg, "150") {
		t.Fatalf("错误信息 %q 未同时给出合计字节数与上限", msg)
	}
	entries, _ := store.List()
	if len(entries) != 1 {
		t.Fatalf("暂存项数 = %d，期望 1（首次暂存保留）", len(entries))
	}
}

func TestStageOversizeImageUsesImageLimit(t *testing.T) {
	limits := DefaultLimits()
	limits.Text = 1000
	limits.Image = 16
	store := newTestStore(t, limits)
	content := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 32)...)
	src := writeSource(t, "shot.png", content)

	_, err := store.Stage(src)
	var limitErr *LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("错误 = %v，期望 *LimitError", err)
	}
	if limitErr.Kind != LimitKindFile || limitErr.Category != CategoryImage {
		t.Fatalf("上限类别/附件类别 = %q/%q，期望 file/image", limitErr.Kind, limitErr.Category)
	}
	if limitErr.Size != int64(len(content)) || limitErr.Limit != 16 {
		t.Fatalf("字节数/上限 = %d/%d，期望 %d/16", limitErr.Size, limitErr.Limit, len(content))
	}
	if !strings.Contains(err.Error(), "图片") {
		t.Fatalf("错误信息 %q 未说明图片类别", err.Error())
	}
}

func TestStageImageCategory(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	content := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0x01}, 64)...)
	entry, err := store.Stage(writeSource(t, "icon.png", content))
	if err != nil {
		t.Fatalf("暂存图片失败：%v", err)
	}
	if entry.Category != CategoryImage {
		t.Fatalf("类别 = %q，期望 image", entry.Category)
	}
}

func TestStageDeduplicatesByDigest(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	content := []byte("同一份内容")
	srcA := writeSource(t, "a.txt", content)
	srcB := writeSource(t, "b.txt", content)

	first, err := store.Stage(srcA)
	if err != nil {
		t.Fatalf("首次暂存失败：%v", err)
	}
	second, err := store.Stage(srcB)
	if err != nil {
		t.Fatalf("重复暂存失败：%v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("重复挂载得到不同标识：%s / %s", first.ID, second.ID)
	}
	if second.Refs != 2 {
		t.Fatalf("引用计数 = %d，期望 2", second.Refs)
	}
	if second.Name != "a.txt" {
		t.Fatalf("重复挂载返回的名称 = %q，期望保留首次暂存的 a.txt", second.Name)
	}
	if items := blobEntries(t, store); len(items) != 1 {
		t.Fatalf("内容目录文件数 = %d，期望 1（同一内容只存一份）", len(items))
	}
	entries, _ := store.List()
	if len(entries) != 1 {
		t.Fatalf("暂存项数 = %d，期望 1", len(entries))
	}

	if err := store.Release(first.ID); err != nil {
		t.Fatalf("释放失败：%v", err)
	}
	still, err := store.Get(first.ID)
	if err != nil || still.Refs != 1 {
		t.Fatalf("释放一次后 = %#v, %v，期望引用计数 1", still, err)
	}
	if err := store.Release(first.ID); err != nil {
		t.Fatalf("释放失败：%v", err)
	}
	released, err := store.Get(first.ID)
	if err != nil || released.Refs != 0 {
		t.Fatalf("释放两次后 = %#v, %v，期望引用计数 0", released, err)
	}

	// 无引用项在再次暂存同一内容时被重新使用。
	again, err := store.Stage(srcA)
	if err != nil {
		t.Fatalf("重新暂存失败：%v", err)
	}
	if again.ID != first.ID || again.Refs != 1 {
		t.Fatalf("重新暂存 = %#v，期望复用 %s 且引用计数 1", again, first.ID)
	}
}

func TestStageRepairsCorruptBlobOnDuplicate(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	content := []byte("需要修复的内容")
	src := writeSource(t, "a.txt", content)
	entry, err := store.Stage(src)
	if err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	blob := filepath.Join(store.Dir(), blobDirName, entry.Digest)
	if err := os.WriteFile(blob, []byte("坏掉的同长度内容"), 0o600); err != nil {
		t.Fatalf("破坏内容文件失败：%v", err)
	}
	checks, err := store.Validate()
	if err != nil {
		t.Fatalf("校验失败：%v", err)
	}
	if len(checks) != 1 || checks[0].Problem != ProblemSizeMismatch && checks[0].Problem != ProblemDigestMismatch {
		t.Fatalf("校验结果 = %#v，期望发现内容不一致", checks)
	}

	if _, err := store.Stage(src); err != nil {
		t.Fatalf("重复暂存修复失败：%v", err)
	}
	checks, err = store.Validate()
	if err != nil {
		t.Fatalf("校验失败：%v", err)
	}
	if len(checks) != 1 || !checks[0].OK() {
		t.Fatalf("修复后校验结果 = %#v，期望有效", checks)
	}
}

func TestStageClassifiesByFullContent(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	// 前 512 字节看起来是文本，NUL 出现在文件头之后，只有读取全量内容才能发现。
	content := append(bytes.Repeat([]byte("a"), 600), 0x00)
	_, err := store.Stage(writeSource(t, "mixed.txt", content))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("错误 = %v，期望按完整内容判为 ErrUnsupported", err)
	}
	entries, _ := store.List()
	if len(entries) != 0 {
		t.Fatalf("被拒绝的内容仍留下 %d 条暂存项", len(entries))
	}
}

func TestStageRepairsMissingBlobOnDuplicate(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	content := []byte("内容文件会被删除再修复")
	src := writeSource(t, "a.txt", content)
	entry, err := store.Stage(src)
	if err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	blob := filepath.Join(store.Dir(), blobDirName, entry.Digest)
	if err := os.Remove(blob); err != nil {
		t.Fatalf("删除内容文件失败：%v", err)
	}
	again, err := store.Stage(src)
	if err != nil {
		t.Fatalf("重复暂存修复失败：%v", err)
	}
	if again.ID != entry.ID {
		t.Fatalf("修复后标识 = %s，期望 %s", again.ID, entry.ID)
	}
	checks, err := store.Validate()
	if err != nil {
		t.Fatalf("校验失败：%v", err)
	}
	if len(checks) != 1 || !checks[0].OK() {
		t.Fatalf("修复后校验 = %#v，期望有效", checks)
	}
	got, err := store.Read(entry.ID)
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("修复后内容 = %q, %v，期望原始内容", got, err)
	}
}

func TestStageDeduplicatesConcurrently(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	src := writeSource(t, "a.txt", []byte("并发暂存的同一内容"))

	const n = 8
	var wg sync.WaitGroup
	ids := make([]string, n)
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			entry, err := store.Stage(src)
			ids[i], errs[i] = entry.ID, err
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("第 %d 次暂存失败：%v", i, err)
		}
	}
	for i := 1; i < n; i++ {
		if ids[i] != ids[0] {
			t.Fatalf("并发暂存得到不同标识：%q / %q", ids[0], ids[i])
		}
	}
	entries, err := store.List()
	if err != nil {
		t.Fatalf("列出暂存项失败：%v", err)
	}
	if len(entries) != 1 || entries[0].Refs != n {
		t.Fatalf("暂存项 = %#v，期望 1 条且引用计数 %d", entries, n)
	}
	if items := blobEntries(t, store); len(items) != 1 {
		t.Fatalf("内容目录文件数 = %d，期望 1", len(items))
	}
}

func TestStageRejectsUnsupportedContent(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	cases := []struct {
		name    string
		content []byte
	}{
		{"binary.bin", []byte("abc\x00def")},
		{"doc.pdf", []byte("%PDF-1.7\n1 0 obj\n")},
		{"bundle.zip", []byte("PK\x03\x04rest-of-archive")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := store.Stage(writeSource(t, tc.name, tc.content))
			if !errors.Is(err, ErrUnsupported) {
				t.Fatalf("错误 = %v，期望 ErrUnsupported", err)
			}
			var unsupported *UnsupportedError
			if !errors.As(err, &unsupported) {
				t.Fatalf("错误 %v 未携带 *UnsupportedError", err)
			}
			if unsupported.Name != tc.name || unsupported.Size != int64(len(tc.content)) {
				t.Fatalf("错误详情 = %#v，期望名称 %q、字节数 %d", unsupported, tc.name, len(tc.content))
			}
		})
	}
	entries, _ := store.List()
	if len(entries) != 0 {
		t.Fatalf("被拒绝的内容仍留下 %d 条暂存项", len(entries))
	}
}

func TestStageRejectsNonRegularFile(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	if _, err := store.Stage(t.TempDir()); !errors.Is(err, ErrNotRegular) {
		t.Fatalf("目录暂存的错误 = %v，期望 ErrNotRegular", err)
	}
	if _, err := store.Stage(""); err == nil {
		t.Fatal("空路径未被拒绝")
	}
}

func TestValidateDetectsInvalidEntries(t *testing.T) {
	newStaged := func(t *testing.T) (*Store, Entry, string) {
		t.Helper()
		store := newTestStore(t, DefaultLimits())
		entry, err := store.Stage(writeSource(t, "note.txt", []byte("内容")))
		if err != nil {
			t.Fatalf("暂存失败：%v", err)
		}
		return store, entry, filepath.Join(store.Dir(), blobDirName, entry.Digest)
	}

	t.Run("缺失", func(t *testing.T) {
		store, _, blob := newStaged(t)
		if err := os.Remove(blob); err != nil {
			t.Fatalf("删除内容文件失败：%v", err)
		}
		checks, err := store.Validate()
		if err != nil {
			t.Fatalf("校验失败：%v", err)
		}
		if len(checks) != 1 || checks[0].Problem != ProblemBlobMissing {
			t.Fatalf("校验结果 = %#v，期望 blob_missing", checks)
		}
	})

	t.Run("字节数不一致", func(t *testing.T) {
		store, _, blob := newStaged(t)
		if err := os.WriteFile(blob, []byte("短"), 0o600); err != nil {
			t.Fatalf("改写内容文件失败：%v", err)
		}
		checks, err := store.Validate()
		if err != nil {
			t.Fatalf("校验失败：%v", err)
		}
		if len(checks) != 1 || checks[0].Problem != ProblemSizeMismatch {
			t.Fatalf("校验结果 = %#v，期望 size_mismatch", checks)
		}
	})

	t.Run("摘要不一致", func(t *testing.T) {
		store, entry, blob := newStaged(t)
		if err := os.WriteFile(blob, bytes.Repeat([]byte("x"), int(entry.Size)), 0o600); err != nil {
			t.Fatalf("改写内容文件失败：%v", err)
		}
		checks, err := store.Validate()
		if err != nil {
			t.Fatalf("校验失败：%v", err)
		}
		if len(checks) != 1 || checks[0].Problem != ProblemDigestMismatch {
			t.Fatalf("校验结果 = %#v，期望 digest_mismatch", checks)
		}
	})

	t.Run("不是普通文件", func(t *testing.T) {
		store, _, blob := newStaged(t)
		if err := os.Remove(blob); err != nil {
			t.Fatalf("删除内容文件失败：%v", err)
		}
		if err := os.Mkdir(blob, 0o700); err != nil {
			t.Fatalf("创建同名目录失败：%v", err)
		}
		checks, err := store.Validate()
		if err != nil {
			t.Fatalf("校验失败：%v", err)
		}
		if len(checks) != 1 || checks[0].Problem != ProblemBlobNotRegular {
			t.Fatalf("校验结果 = %#v，期望 blob_not_regular", checks)
		}
	})

	t.Run("类别不一致", func(t *testing.T) {
		store, entry, _ := newStaged(t)
		entry.Category = CategoryImage
		writeIndexFile(t, store, indexFile{Version: indexVersion, Entries: []Entry{entry}})

		reopened, err := Open(filepath.Dir(store.Dir()), DefaultLimits())
		if err != nil {
			t.Fatalf("重新打开暂存区失败：%v", err)
		}
		checks, err := reopened.Validate()
		if err != nil {
			t.Fatalf("校验失败：%v", err)
		}
		if len(checks) != 1 || checks[0].Problem != ProblemCategoryMismatch {
			t.Fatalf("校验结果 = %#v，期望 category_mismatch", checks)
		}
	})
}

func TestValidateDetectsInvalidRecord(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	entry, err := store.Stage(writeSource(t, "note.txt", []byte("内容")))
	if err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	entry.Name = ""
	writeIndexFile(t, store, indexFile{Version: indexVersion, Entries: []Entry{entry}})

	_, err = Open(filepath.Dir(store.Dir()), DefaultLimits())
	if !errors.Is(err, ErrIndexCorrupt) {
		t.Fatalf("错误 = %v，期望 ErrIndexCorrupt", err)
	}
}

func TestCleanupRemovesInvalidUnreferenced(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	entry, err := store.Stage(writeSource(t, "note.txt", []byte("内容")))
	if err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	if err := store.Release(entry.ID); err != nil {
		t.Fatalf("释放失败：%v", err)
	}
	blob := filepath.Join(store.Dir(), blobDirName, entry.Digest)
	if err := os.Remove(blob); err != nil {
		t.Fatalf("删除内容文件失败：%v", err)
	}

	report, err := store.Cleanup()
	if err != nil {
		t.Fatalf("清理失败：%v", err)
	}
	if len(report.RemovedInvalid) != 1 || report.RemovedInvalid[0] != entry.ID {
		t.Fatalf("清理结果 = %#v，期望删除失效项 %s", report, entry.ID)
	}
	if len(report.KeptReferenced) != 0 {
		t.Fatalf("清理保留了 %d 个引用项，期望 0", len(report.KeptReferenced))
	}
	entries, _ := store.List()
	if len(entries) != 0 {
		t.Fatalf("清理后仍有 %d 条暂存项", len(entries))
	}

	reopened, err := Open(filepath.Dir(store.Dir()), DefaultLimits())
	if err != nil {
		t.Fatalf("重新打开暂存区失败：%v", err)
	}
	entries, _ = reopened.List()
	if len(entries) != 0 {
		t.Fatalf("重启后仍有 %d 条已清理的暂存项", len(entries))
	}
}

func TestCleanupKeepsReferencedInvalid(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	entry, err := store.Stage(writeSource(t, "note.txt", []byte("仍在使用的暂存内容")))
	if err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	blob := filepath.Join(store.Dir(), blobDirName, entry.Digest)
	if err := os.Remove(blob); err != nil {
		t.Fatalf("删除内容文件失败：%v", err)
	}

	report, err := store.Cleanup()
	if err != nil {
		t.Fatalf("清理失败：%v", err)
	}
	if len(report.RemovedInvalid) != 0 {
		t.Fatalf("清理删除了被引用的失效项：%#v", report.RemovedInvalid)
	}
	if len(report.KeptReferenced) != 1 || report.KeptReferenced[0].Problem != ProblemBlobMissing {
		t.Fatalf("保留项 = %#v，期望 1 条 blob_missing", report.KeptReferenced)
	}
	entries, _ := store.List()
	if len(entries) != 1 || entries[0].ID != entry.ID {
		t.Fatalf("暂存项 = %#v，期望保留 %s", entries, entry.ID)
	}

	if err := store.Release(entry.ID); err != nil {
		t.Fatalf("释放失败：%v", err)
	}
	report, err = store.Cleanup()
	if err != nil {
		t.Fatalf("二次清理失败：%v", err)
	}
	if len(report.RemovedInvalid) != 1 || report.RemovedInvalid[0] != entry.ID {
		t.Fatalf("二次清理结果 = %#v，期望删除 %s", report, entry.ID)
	}
	entries, _ = store.List()
	if len(entries) != 0 {
		t.Fatalf("二次清理后仍有 %d 条暂存项", len(entries))
	}
}

func TestCleanupRemovesUnreferencedAndOrphans(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	first, err := store.Stage(writeSource(t, "a.txt", []byte("第一份内容")))
	if err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	kept, err := store.Stage(writeSource(t, "b.txt", []byte("第二份内容")))
	if err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	if err := store.Release(first.ID); err != nil {
		t.Fatalf("释放失败：%v", err)
	}
	orphan := filepath.Join(store.Dir(), blobDirName, "orphan-file")
	if err := os.WriteFile(orphan, []byte("没有索引记录的内容"), 0o600); err != nil {
		t.Fatalf("写入孤儿文件失败：%v", err)
	}

	report, err := store.Cleanup()
	if err != nil {
		t.Fatalf("清理失败：%v", err)
	}
	if len(report.RemovedUnreferenced) != 1 || report.RemovedUnreferenced[0] != first.ID {
		t.Fatalf("清理结果 = %#v，期望回收无引用项 %s", report, first.ID)
	}
	if len(report.RemovedOrphans) != 1 || report.RemovedOrphans[0] != "orphan-file" {
		t.Fatalf("孤儿文件 = %#v，期望 orphan-file", report.RemovedOrphans)
	}
	if report.BytesFreed <= 0 {
		t.Fatalf("释放字节数 = %d，期望大于 0", report.BytesFreed)
	}

	entries, _ := store.List()
	if len(entries) != 1 || entries[0].ID != kept.ID {
		t.Fatalf("清理后暂存项 = %#v，期望只保留 %s", entries, kept.ID)
	}
	checks, err := store.Validate()
	if err != nil {
		t.Fatalf("校验失败：%v", err)
	}
	if len(checks) != 1 || !checks[0].OK() {
		t.Fatalf("清理后校验 = %#v，期望有效", checks)
	}
	if _, err := os.Stat(orphan); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("孤儿文件仍存在：%v", err)
	}
}

func TestStageReclaimsReleasedEntries(t *testing.T) {
	limits := DefaultLimits()
	limits.Text = 100
	limits.Total = 100
	store := newTestStore(t, limits)

	old, err := store.Stage(writeSource(t, "old.txt", bytes.Repeat([]byte("o"), 60)))
	if err != nil {
		t.Fatalf("首次暂存失败：%v", err)
	}
	if err := store.Release(old.ID); err != nil {
		t.Fatalf("释放失败：%v", err)
	}
	if _, err := store.Stage(writeSource(t, "new.txt", bytes.Repeat([]byte("n"), 60))); err != nil {
		t.Fatalf("回收无引用项后暂存失败：%v", err)
	}
	entries, _ := store.List()
	if len(entries) != 1 {
		t.Fatalf("暂存项数 = %d，期望 1（无引用项已回收）", len(entries))
	}
}

func TestTotalLimitIgnoresDuplicateContent(t *testing.T) {
	limits := DefaultLimits()
	limits.Text = 100
	limits.Total = 100
	store := newTestStore(t, limits)
	content := bytes.Repeat([]byte("a"), 100)
	src := writeSource(t, "a.txt", content)

	if _, err := store.Stage(src); err != nil {
		t.Fatalf("首次暂存失败：%v", err)
	}
	// 同一内容再次挂载不增加占用，即使暂存区已到总上限也应成功。
	if _, err := store.Stage(src); err != nil {
		t.Fatalf("重复暂存失败：%v", err)
	}
}

func TestRestartKeepsEntriesAndReferences(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	src := writeSource(t, "note.txt", []byte("重启后仍可识别"))
	entry, err := store.Stage(src)
	if err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	if _, err := store.Stage(src); err != nil {
		t.Fatalf("重复暂存失败：%v", err)
	}

	reopened, err := Open(filepath.Dir(store.Dir()), DefaultLimits())
	if err != nil {
		t.Fatalf("重新打开暂存区失败：%v", err)
	}
	entries, err := reopened.List()
	if err != nil {
		t.Fatalf("列出暂存项失败：%v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("重启后暂存项数 = %d，期望 1", len(entries))
	}
	if entries[0].ID != entry.ID || entries[0].Refs != 2 {
		t.Fatalf("重启后暂存项 = %#v，期望 %s 且引用计数 2", entries[0], entry.ID)
	}
	checks, err := reopened.Validate()
	if err != nil {
		t.Fatalf("校验失败：%v", err)
	}
	if len(checks) != 1 || !checks[0].OK() {
		t.Fatalf("重启后校验 = %#v，期望有效", checks)
	}
}

func TestOpenRejectsUnsupportedIndex(t *testing.T) {
	t.Run("版本过高", func(t *testing.T) {
		store := newTestStore(t, DefaultLimits())
		writeIndexFile(t, store, indexFile{Version: indexVersion + 1, Entries: []Entry{}})
		_, err := Open(filepath.Dir(store.Dir()), DefaultLimits())
		if !errors.Is(err, ErrIndexVersion) {
			t.Fatalf("错误 = %v，期望 ErrIndexVersion", err)
		}
	})

	t.Run("无法解析", func(t *testing.T) {
		store := newTestStore(t, DefaultLimits())
		path := filepath.Join(store.Dir(), indexFileName)
		if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
			t.Fatalf("写入损坏索引失败：%v", err)
		}
		_, err := Open(filepath.Dir(store.Dir()), DefaultLimits())
		if !errors.Is(err, ErrIndexCorrupt) {
			t.Fatalf("错误 = %v，期望 ErrIndexCorrupt", err)
		}
	})

	t.Run("标识重复", func(t *testing.T) {
		store := newTestStore(t, DefaultLimits())
		entry, err := store.Stage(writeSource(t, "a.txt", []byte("内容")))
		if err != nil {
			t.Fatalf("暂存失败：%v", err)
		}
		duplicate := entry
		duplicate.Digest = strings.Repeat("0", 64)
		writeIndexFile(t, store, indexFile{Version: indexVersion, Entries: []Entry{entry, duplicate}})
		_, err = Open(filepath.Dir(store.Dir()), DefaultLimits())
		if !errors.Is(err, ErrIndexCorrupt) {
			t.Fatalf("错误 = %v，期望 ErrIndexCorrupt", err)
		}
	})
}

func TestPermissions(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	entry, err := store.Stage(writeSource(t, "note.txt", []byte("权限检查")))
	if err != nil {
		t.Fatalf("暂存失败：%v", err)
	}

	root := store.Dir()
	assertPerm(t, root, 0o700)
	assertPerm(t, filepath.Join(root, blobDirName), 0o700)
	assertPerm(t, filepath.Join(root, indexFileName), 0o600)
	assertPerm(t, filepath.Join(root, blobDirName, entry.Digest), 0o600)

	// 重启后索引权限被重新收紧。
	indexPath := filepath.Join(root, indexFileName)
	if err := os.Chmod(indexPath, 0o644); err != nil {
		t.Fatalf("放宽索引权限失败：%v", err)
	}
	if _, err := Open(filepath.Dir(root), DefaultLimits()); err != nil {
		t.Fatalf("重新打开暂存区失败：%v", err)
	}
	assertPerm(t, indexPath, 0o600)
}

func assertPerm(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("读取权限失败：%v", err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s 权限 = %o，期望 %o", filepath.Base(path), got, want)
	}
}

func TestReset(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	if _, err := store.Stage(writeSource(t, "a.txt", []byte("内容甲"))); err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	if _, err := store.Stage(writeSource(t, "b.txt", []byte("内容乙"))); err != nil {
		t.Fatalf("暂存失败：%v", err)
	}
	if err := store.Reset(); err != nil {
		t.Fatalf("重置失败：%v", err)
	}
	entries, _ := store.List()
	if len(entries) != 0 {
		t.Fatalf("重置后仍有 %d 条暂存项", len(entries))
	}
	if items := blobEntries(t, store); len(items) != 0 {
		t.Fatalf("重置后内容目录仍有 %d 个文件", len(items))
	}
	if _, err := store.Stage(writeSource(t, "c.txt", []byte("内容丙"))); err != nil {
		t.Fatalf("重置后再次暂存失败：%v", err)
	}
}

func TestReleaseUnknown(t *testing.T) {
	store := newTestStore(t, DefaultLimits())
	if err := store.Release("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("错误 = %v，期望 ErrNotFound", err)
	}
	if _, err := store.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("错误 = %v，期望 ErrNotFound", err)
	}
	if _, err := store.Read("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("错误 = %v，期望 ErrNotFound", err)
	}
}

func TestOpenRejectsInvalidLimits(t *testing.T) {
	cases := []Limits{
		{Text: 0, Image: 1, Total: 1},
		{Text: 1, Image: 0, Total: 1},
		{Text: 1, Image: 1, Total: 0},
	}
	for _, limits := range cases {
		if _, err := Open(t.TempDir(), limits); err == nil {
			t.Fatalf("上限 %#v 未被拒绝", limits)
		}
	}
}
