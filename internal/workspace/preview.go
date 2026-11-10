package workspace

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	// DefaultPreviewBytes 是文件预览的默认读取上限（256 KiB）。
	DefaultPreviewBytes = 256 << 10
	// MaxPreviewBytes 是调用方可请求的预览硬上限（16 MiB），防止误传极大值。
	MaxPreviewBytes = 16 << 20
)

// PreviewKind 是预览内容的类别。
type PreviewKind string

const (
	// PreviewEmpty 表示空文件。
	PreviewEmpty PreviewKind = "empty"
	// PreviewText 表示已按识别出的编码解码为文本。
	PreviewText PreviewKind = "text"
	// PreviewBinary 表示二进制或未识别编码，不返回内容。
	PreviewBinary PreviewKind = "binary"
)

// PreviewOptions 控制预览读取。
type PreviewOptions struct {
	// MaxBytes 为 0 时用 DefaultPreviewBytes；超过 MaxPreviewBytes 直接报错。
	MaxBytes int
}

// Preview 是一次文件预览的结果。
type Preview struct {
	// Path 是相对工作区的路径。
	Path string
	Kind PreviewKind
	// Encoding 是识别出的编码，如 "utf-8"、"utf-16le"、"utf-16be"；二进制为空。
	Encoding string
	Size     int64
	// ReadBytes 是从磁盘实际读取的字节数。
	ReadBytes int
	// Content 只在 Kind 为 text 时有值。
	Content string
	// Truncated 表示文件大于读取上限，Content 只是前缀。
	Truncated bool
	// Note 说明截断、二进制或未识别编码的原因；为空表示没有需要说明的情况。
	Note string
}

// Preview 读取工作区内一个文件的前缀内容。
// 不跟随到工作区外的符号链接；目录返回 ErrIsDirectory。
func (w *Workspace) Preview(rel string, opts PreviewOptions) (Preview, error) {
	abs, err := w.Resolve(rel)
	if err != nil {
		return Preview{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Preview{}, fmt.Errorf("%w: %s", ErrPathNotExist, rel)
		}
		return Preview{}, fmt.Errorf("读取文件 %s: %w", rel, err)
	}
	if info.IsDir() {
		return Preview{}, fmt.Errorf("%w: %s", ErrIsDirectory, rel)
	}
	if !info.Mode().IsRegular() {
		return Preview{}, fmt.Errorf("%w: %s", ErrNotRegularFile, rel)
	}

	limit := opts.MaxBytes
	if limit == 0 {
		limit = DefaultPreviewBytes
	}
	if limit < 0 {
		return Preview{}, fmt.Errorf("%w: %d", ErrPreviewLimitTooLarge, limit)
	}
	if limit > MaxPreviewBytes {
		return Preview{}, fmt.Errorf("%w: %d > %d", ErrPreviewLimitTooLarge, limit, MaxPreviewBytes)
	}

	size := info.Size()
	truncated := size > int64(limit)
	readLen := int64(limit)
	if size < readLen {
		readLen = size
	}
	f, err := os.Open(abs)
	if err != nil {
		return Preview{}, fmt.Errorf("打开文件 %s: %w", rel, err)
	}
	defer f.Close()

	buf := make([]byte, readLen)
	n, readErr := io.ReadFull(f, buf)
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		return Preview{}, fmt.Errorf("读取文件 %s: %w", rel, readErr)
	}
	raw := buf[:n]

	preview := Preview{
		Path:      w.displayPath(abs),
		Size:      size,
		ReadBytes: n,
		Truncated: truncated,
	}
	var notes []string
	if truncated {
		notes = append(notes, fmt.Sprintf("文件共 %d 字节，按上限读取前 %d 字节（已截断）", size, n))
	}

	decodePreview(raw, truncated, &preview, &notes)
	preview.Note = joinNotes(notes...)
	return preview, nil
}

// decodePreview 判定编码与内容类别，并填充 preview。
func decodePreview(raw []byte, truncated bool, preview *Preview, notes *[]string) {
	switch {
	case len(raw) == 0 && preview.Size == 0:
		preview.Kind = PreviewEmpty
		preview.Encoding = "utf-8"
		*notes = append(*notes, "空文件")
		return
	case hasUTF32BOM(raw):
		preview.Kind = PreviewBinary
		*notes = append(*notes, "UTF-32 编码暂不支持，未解码")
		return
	case bytes.HasPrefix(raw, []byte{0xEF, 0xBB, 0xBF}):
		text := raw[3:]
		if truncated {
			text = trimPartialUTF8(text)
		}
		if !utf8.Valid(text) {
			preview.Kind = PreviewBinary
			*notes = append(*notes, "UTF-8 内容无效，按二进制处理")
			return
		}
		preview.Kind = PreviewText
		preview.Encoding = "utf-8"
		preview.Content = string(text)
		return
	case bytes.HasPrefix(raw, []byte{0xFF, 0xFE}):
		preview.Kind = PreviewText
		preview.Encoding = "utf-16le"
		preview.Content = decodeUTF16(raw[2:], false)
		return
	case bytes.HasPrefix(raw, []byte{0xFE, 0xFF}):
		preview.Kind = PreviewText
		preview.Encoding = "utf-16be"
		preview.Content = decodeUTF16(raw[2:], true)
		return
	case bytes.IndexByte(raw, 0) >= 0:
		preview.Kind = PreviewBinary
		*notes = append(*notes, "内容含 NUL 字节，按二进制处理")
		return
	}

	text := raw
	if truncated {
		text = trimPartialUTF8(text)
	}
	if !utf8.Valid(text) {
		preview.Kind = PreviewBinary
		*notes = append(*notes, "不是 UTF-8 文本，未识别编码，未解码")
		return
	}
	preview.Kind = PreviewText
	preview.Encoding = "utf-8"
	preview.Content = string(text)
}

// hasUTF32BOM 识别 UTF-32 的 BOM；UTF-32 不在本领域支持范围。
func hasUTF32BOM(raw []byte) bool {
	if len(raw) < 4 {
		return false
	}
	if raw[0] == 0x00 && raw[1] == 0x00 && raw[2] == 0xFE && raw[3] == 0xFF {
		return true
	}
	return raw[0] == 0xFF && raw[1] == 0xFE && raw[2] == 0x00 && raw[3] == 0x00
}

// decodeUTF16 解码 UTF-16 字节流，截断产生的半个码元被丢弃。
func decodeUTF16(data []byte, bigEndian bool) string {
	if len(data)%2 == 1 {
		data = data[:len(data)-1]
	}
	units := make([]uint16, len(data)/2)
	for i := range units {
		if bigEndian {
			units[i] = binary.BigEndian.Uint16(data[i*2:])
		} else {
			units[i] = binary.LittleEndian.Uint16(data[i*2:])
		}
	}
	return string(utf16.Decode(units))
}

// trimPartialUTF8 只用于截断场景：去掉末尾被切断的半个 UTF-8 字符。
// 最多去掉 UTFMax-1 个字节；若仍无效，调用方按二进制处理。
func trimPartialUTF8(b []byte) []byte {
	for i := 0; i < utf8.UTFMax && len(b) > 0 && !utf8.Valid(b); i++ {
		b = b[:len(b)-1]
	}
	return b
}

// joinNotes 用中文分号拼接说明，空串被忽略。
func joinNotes(parts ...string) string {
	notes := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			notes = append(notes, part)
		}
	}
	return strings.Join(notes, "；")
}
