package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestArchivePersistsAcrossRestart 确认隐藏列表跨实例（模拟重启）保持，且文件带版本号。
func TestArchivePersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session-archive.json")
	archive, err := OpenArchive(path)
	if err != nil {
		t.Fatalf("打开归档隐藏列表: %v", err)
	}
	if archive.Rebuilt() {
		t.Fatal("新文件不应标记为重建")
	}
	if err := archive.Hide("s1"); err != nil {
		t.Fatalf("隐藏 s1: %v", err)
	}
	if err := archive.Hide("s2"); err != nil {
		t.Fatalf("隐藏 s2: %v", err)
	}
	// 重复隐藏不改变结果。
	if err := archive.Hide("s1"); err != nil {
		t.Fatalf("重复隐藏 s1: %v", err)
	}
	if err := archive.Unhide("s1"); err != nil {
		t.Fatalf("取消隐藏 s1: %v", err)
	}
	// 未知标识的取消隐藏是空操作。
	if err := archive.Unhide("nope"); err != nil {
		t.Fatalf("取消隐藏未知标识: %v", err)
	}

	reopened, err := OpenArchive(path)
	if err != nil {
		t.Fatalf("重新打开归档隐藏列表: %v", err)
	}
	if !reopened.Hidden("s2") {
		t.Fatal("隐藏状态没有跨重启保持")
	}
	if reopened.Hidden("s1") {
		t.Fatal("已取消隐藏的会话仍然隐藏")
	}
	mustEqualStrings(t, "隐藏标识", reopened.HiddenIDs(), []string{"s2"})

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取落盘文件: %v", err)
	}
	var file archiveFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("落盘文件不是合法 JSON: %v", err)
	}
	if file.Version != archiveVersion {
		t.Fatalf("格式版本不符：got %d，want %d", file.Version, archiveVersion)
	}
	if len(file.Hidden) != 1 || file.Hidden[0].SessionID != "s2" {
		t.Fatalf("落盘内容不符：%+v", file.Hidden)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("查看文件属性: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("文件权限不符：got %o，want 600", perm)
	}
}

// TestArchiveCorruptRebuildsEmpty 确认文件损坏时重建为空，且下一次用户操作写出合法文件。
func TestArchiveCorruptRebuildsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session-archive.json")
	if err := os.WriteFile(path, []byte("{不是 JSON"), 0o600); err != nil {
		t.Fatalf("写入损坏文件: %v", err)
	}
	archive, err := OpenArchive(path)
	if err != nil {
		t.Fatalf("打开损坏文件不应报错: %v", err)
	}
	if !archive.Rebuilt() {
		t.Fatal("损坏文件应标记为重建")
	}
	if ids := archive.HiddenIDs(); len(ids) != 0 {
		t.Fatalf("损坏文件应重建为空：%v", ids)
	}

	// 只在用户显式操作时才写盘。
	if err := archive.Hide("s1"); err != nil {
		t.Fatalf("隐藏 s1: %v", err)
	}
	reopened, err := OpenArchive(path)
	if err != nil {
		t.Fatalf("重新打开: %v", err)
	}
	mustEqualStrings(t, "重建后的隐藏标识", reopened.HiddenIDs(), []string{"s1"})
}

// TestArchiveUnknownVersionRebuildsEmpty 确认版本不认识时按损坏处理。
func TestArchiveUnknownVersionRebuildsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session-archive.json")
	raw := []byte(`{"version":99,"hidden":[{"sessionId":"s1"}]}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("写入未知版本文件: %v", err)
	}
	archive, err := OpenArchive(path)
	if err != nil {
		t.Fatalf("打开未知版本文件不应报错: %v", err)
	}
	if !archive.Rebuilt() {
		t.Fatal("未知版本应标记为重建")
	}
	if archive.Hidden("s1") {
		t.Fatal("未知版本的内容不应被采用")
	}
}

// TestArchiveWriteIsAtomic 确认写入不留临时文件，并发隐藏后文件仍是合法且完整的 JSON。
func TestArchiveWriteIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session-archive.json")
	archive, err := OpenArchive(path)
	if err != nil {
		t.Fatalf("打开归档隐藏列表: %v", err)
	}

	var wg sync.WaitGroup
	for _, id := range []string{"s1", "s2", "s3", "s4", "s5", "s6"} {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := archive.Hide(id); err != nil {
				t.Errorf("隐藏 %s: %v", id, err)
			}
		}()
	}
	wg.Wait()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取目录: %v", err)
	}
	for _, item := range entries {
		if item.Name() != filepath.Base(path) {
			t.Fatalf("留下临时文件: %s", item.Name())
		}
	}

	reopened, err := OpenArchive(path)
	if err != nil {
		t.Fatalf("重新打开: %v", err)
	}
	if len(reopened.HiddenIDs()) != 6 {
		t.Fatalf("并发写入丢失记录：%v", reopened.HiddenIDs())
	}
}

// TestArchiveRejectsEmptySessionID 确认空标识在落盘前被拒绝。
func TestArchiveRejectsEmptySessionID(t *testing.T) {
	archive := newArchive(t)
	if err := archive.Hide("  "); err == nil {
		t.Fatal("空标识应被拒绝")
	}
	if _, err := OpenArchive("  "); err == nil {
		t.Fatal("空路径应被拒绝")
	}
}
