package attachments

import (
	"bytes"
	"testing"
)

func TestClassifyContent(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 16)...)
	jpeg := append([]byte("\xff\xd8\xff\xe0"), make([]byte, 16)...)
	webp := append([]byte("RIFF\x10\x00\x00\x00WEBP"), make([]byte, 8)...)
	heic := append([]byte("\x00\x00\x00\x18ftypheic"), make([]byte, 8)...)
	avif := append([]byte("\x00\x00\x00\x18ftypavif"), make([]byte, 8)...)

	cases := []struct {
		name     string
		data     []byte
		want     Category
		wantOK   bool
		wantHint string
	}{
		{"空文件", nil, CategoryText, true, ""},
		{"ASCII 文本", []byte("hello attachment\n"), CategoryText, true, ""},
		{"中文 UTF-8 文本", []byte("你好，附件。\n"), CategoryText, true, ""},
		{"带 BOM 的 UTF-8", append([]byte("\xef\xbb\xbf"), []byte("text")...), CategoryText, true, ""},
		{"GBK 文本", []byte{0xc4, 0xe3, 0xba, 0xc3, 0xa3, 0xac}, CategoryText, true, ""},
		{"带 ANSI 转义的日志", []byte("\x1b[31merror\x1b[0m\n"), CategoryText, true, ""},
		{"PNG", png, CategoryImage, true, ""},
		{"JPEG", jpeg, CategoryImage, true, ""},
		{"GIF87a", append([]byte("GIF87a"), make([]byte, 8)...), CategoryImage, true, ""},
		{"GIF89a", append([]byte("GIF89a"), make([]byte, 8)...), CategoryImage, true, ""},
		{"TIFF", append([]byte("II*\x00"), make([]byte, 8)...), CategoryImage, true, ""},
		{"WebP", webp, CategoryImage, true, ""},
		{"HEIC", heic, CategoryImage, true, ""},
		{"AVIF", avif, CategoryImage, true, ""},
		{"PDF", []byte("%PDF-1.7\n1 0 obj\n"), "", false, "PDF"},
		{"ZIP", []byte("PK\x03\x04rest"), "", false, "ZIP"},
		{"GZIP", []byte("\x1f\x8b\x08\x00"), "", false, "GZIP"},
		{"ELF", []byte("\x7fELF\x02\x01\x01"), "", false, "ELF"},
		{"含 NUL 的内容", []byte("abc\x00def"), "", false, "NUL"},
		{"控制字节占比过高", []byte("\x01\x02\x03\x04\x05\x06\x07\x08"), "", false, "控制字节"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason, ok := classifyContent(tc.data)
			if ok != tc.wantOK {
				t.Fatalf("可判定 = %v，期望 %v（原因 %q）", ok, tc.wantOK, reason)
			}
			if ok && got != tc.want {
				t.Fatalf("类别 = %q，期望 %q", got, tc.want)
			}
			if !ok && tc.wantHint != "" && !bytes.Contains([]byte(reason), []byte(tc.wantHint)) {
				t.Fatalf("原因 %q 未说明 %q", reason, tc.wantHint)
			}
		})
	}
}

func TestClassifyHeader(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 16)...)
	category, reason, ok := classifyHeader(png)
	if !ok || category != CategoryImage {
		t.Fatalf("PNG 文件头 = %q/%v（%q），期望 image", category, ok, reason)
	}

	category, reason, ok = classifyHeader([]byte("plain text header"))
	if !ok || category != CategoryText {
		t.Fatalf("文本文件头 = %q/%v（%q），期望先按 text 处理", category, ok, reason)
	}

	_, reason, ok = classifyHeader([]byte("%PDF-1.7\n"))
	if ok {
		t.Fatal("PDF 文件头未被拒绝")
	}
	if reason == "" {
		t.Fatal("PDF 文件头被拒绝但没有原因")
	}
}

func TestTextLikeRequiresNULFreeContent(t *testing.T) {
	if _, ok := textLike([]byte("abc\x00")); ok {
		t.Fatal("含 NUL 的内容被当作文本")
	}
	if _, ok := textLike(nil); !ok {
		t.Fatal("空内容未被当作文本")
	}
}
