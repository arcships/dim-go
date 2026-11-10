package workspace

import (
	"encoding/binary"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
)

func TestPreviewUTF8(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("你好\nworld"))
	preview, err := ws.Preview("a.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("预览失败: %v", err)
	}
	if preview.Kind != PreviewText || preview.Encoding != "utf-8" {
		t.Fatalf("预览类别 = %s/%s", preview.Kind, preview.Encoding)
	}
	if preview.Content != "你好\nworld" || preview.Truncated || int64(preview.ReadBytes) != preview.Size {
		t.Fatalf("预览结果 = %+v", preview)
	}
	if preview.Note != "" {
		t.Fatalf("未截断时不应有说明，得到 %q", preview.Note)
	}
}

func TestPreviewTruncated(t *testing.T) {
	ws, root := newTestWorkspace(t)
	content := strings.Repeat("a", 100)
	writeFile(t, filepath.Join(root, "a.txt"), []byte(content))
	preview, err := ws.Preview("a.txt", PreviewOptions{MaxBytes: 10})
	if err != nil {
		t.Fatalf("预览失败: %v", err)
	}
	if !preview.Truncated || preview.ReadBytes != 10 || preview.Content != content[:10] {
		t.Fatalf("截断结果 = %+v", preview)
	}
	if !strings.Contains(preview.Note, "截断") || !strings.Contains(preview.Note, "100") {
		t.Fatalf("截断说明 = %q", preview.Note)
	}
}

func TestPreviewTruncatedMultibyte(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "cn.txt"), []byte(strings.Repeat("你", 10)))
	preview, err := ws.Preview("cn.txt", PreviewOptions{MaxBytes: 7})
	if err != nil {
		t.Fatalf("预览失败: %v", err)
	}
	if !preview.Truncated {
		t.Fatal("应标记截断")
	}
	if !utf8.ValidString(preview.Content) {
		t.Fatalf("截断后内容不是合法 UTF-8: %q", preview.Content)
	}
	if preview.Content != "你你" {
		t.Fatalf("截断内容 = %q，期望 %q", preview.Content, "你你")
	}
}

func TestPreviewExactlyAtLimit(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("12345"))
	preview, err := ws.Preview("a.txt", PreviewOptions{MaxBytes: 5})
	if err != nil {
		t.Fatalf("预览失败: %v", err)
	}
	if preview.Truncated || preview.ReadBytes != 5 {
		t.Fatalf("恰好等于上限不应截断: %+v", preview)
	}
}

func TestPreviewEncodings(t *testing.T) {
	ws, root := newTestWorkspace(t)
	text := "你好abc"

	le := []byte{0xFF, 0xFE}
	for _, unit := range utf16.Encode([]rune(text)) {
		var buf [2]byte
		binary.LittleEndian.PutUint16(buf[:], unit)
		le = append(le, buf[:]...)
	}
	writeFile(t, filepath.Join(root, "le.txt"), le)
	preview, err := ws.Preview("le.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("预览 UTF-16LE 失败: %v", err)
	}
	if preview.Kind != PreviewText || preview.Encoding != "utf-16le" || preview.Content != text {
		t.Fatalf("UTF-16LE 结果 = %+v", preview)
	}

	be := []byte{0xFE, 0xFF}
	for _, unit := range utf16.Encode([]rune(text)) {
		var buf [2]byte
		binary.BigEndian.PutUint16(buf[:], unit)
		be = append(be, buf[:]...)
	}
	writeFile(t, filepath.Join(root, "be.txt"), be)
	preview, err = ws.Preview("be.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("预览 UTF-16BE 失败: %v", err)
	}
	if preview.Kind != PreviewText || preview.Encoding != "utf-16be" || preview.Content != text {
		t.Fatalf("UTF-16BE 结果 = %+v", preview)
	}

	writeFile(t, filepath.Join(root, "bom.txt"), append([]byte{0xEF, 0xBB, 0xBF}, []byte("hello")...))
	preview, err = ws.Preview("bom.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("预览 UTF-8 BOM 失败: %v", err)
	}
	if preview.Kind != PreviewText || preview.Encoding != "utf-8" || preview.Content != "hello" {
		t.Fatalf("UTF-8 BOM 结果 = %+v", preview)
	}
}

func TestPreviewBinary(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "bin.dat"), []byte{'a', 0, 'b'})
	preview, err := ws.Preview("bin.dat", PreviewOptions{})
	if err != nil {
		t.Fatalf("预览失败: %v", err)
	}
	if preview.Kind != PreviewBinary || preview.Content != "" {
		t.Fatalf("二进制结果 = %+v", preview)
	}
	if !strings.Contains(preview.Note, "NUL") {
		t.Fatalf("二进制说明 = %q", preview.Note)
	}

	// 非 UTF-8、无 BOM、无 NUL 的文本无法识别编码，按二进制说明，不猜测解码。
	writeFile(t, filepath.Join(root, "gbk.txt"), []byte{0xC4, 0xE3, 0xBA, 0xC3})
	preview, err = ws.Preview("gbk.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("预览失败: %v", err)
	}
	if preview.Kind != PreviewBinary || !strings.Contains(preview.Note, "未识别编码") {
		t.Fatalf("未识别编码结果 = %+v", preview)
	}

	// UTF-32 暂不支持，明确说明而不是按 UTF-16 误解码。
	writeFile(t, filepath.Join(root, "utf32.txt"), []byte{0x00, 0x00, 0xFE, 0xFF, 0, 0, 0, 'a'})
	preview, err = ws.Preview("utf32.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("预览失败: %v", err)
	}
	if preview.Kind != PreviewBinary || !strings.Contains(preview.Note, "UTF-32") {
		t.Fatalf("UTF-32 结果 = %+v", preview)
	}
}

func TestPreviewEmptyAndErrors(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "empty.txt"), nil)
	preview, err := ws.Preview("empty.txt", PreviewOptions{})
	if err != nil {
		t.Fatalf("预览空文件失败: %v", err)
	}
	if preview.Kind != PreviewEmpty || preview.Size != 0 || !strings.Contains(preview.Note, "空文件") {
		t.Fatalf("空文件结果 = %+v", preview)
	}

	if _, err := ws.Preview("", PreviewOptions{}); !errors.Is(err, ErrIsDirectory) {
		t.Fatalf("目录应返回 ErrIsDirectory，得到 %v", err)
	}
	if _, err := ws.Preview("missing.txt", PreviewOptions{}); !errors.Is(err, ErrPathNotExist) {
		t.Fatalf("缺失文件应返回 ErrPathNotExist，得到 %v", err)
	}
	if _, err := ws.Preview("../x", PreviewOptions{}); !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("越界应返回 ErrOutsideWorkspace，得到 %v", err)
	}
	if _, err := ws.Preview("empty.txt", PreviewOptions{MaxBytes: MaxPreviewBytes + 1}); !errors.Is(err, ErrPreviewLimitTooLarge) {
		t.Fatalf("超大上限应返回 ErrPreviewLimitTooLarge，得到 %v", err)
	}
}
