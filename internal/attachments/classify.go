package attachments

import (
	"bytes"
)

// imageSignatures 列出按文件头即可确认的图片格式。判定图片优先于判定文本，
// 因为这些字节序列在正常文本里不会出现；识别不出的格式一律不按图片处理。
var imageSignatures = []struct {
	prefix []byte
	label  string
}{
	{[]byte("\x89PNG\r\n\x1a\n"), "PNG"},
	{[]byte("\xff\xd8\xff"), "JPEG"},
	{[]byte("GIF87a"), "GIF"},
	{[]byte("GIF89a"), "GIF"},
	{[]byte("II*\x00"), "TIFF"},
	{[]byte("MM\x00*"), "TIFF"},
}

// binarySignatures 列出已知的非文本、非图片容器格式。这些格式按内容判定为二进制，
// 本领域只支持文本与图片两类，因此直接拒绝，避免把二进制伪装成文本上传。
var binarySignatures = []struct {
	prefix []byte
	label  string
}{
	{[]byte("%PDF-"), "PDF"},
	{[]byte("PK\x03\x04"), "ZIP"},
	{[]byte("PK\x05\x06"), "空 ZIP"},
	{[]byte("PK\x07\x08"), "分卷 ZIP"},
	{[]byte("\x1f\x8b"), "GZIP"},
	{[]byte("\xfd7zXZ\x00"), "XZ"},
	{[]byte("BZh"), "BZIP2"},
	{[]byte("7z\xbc\xaf\x27\x1c"), "7Z"},
	{[]byte("Rar!\x1a\x07"), "RAR"},
	{[]byte("\x7fELF"), "ELF"},
	{[]byte("MZ"), "PE"},
	{[]byte("\xcf\xfa\xed\xfe"), "Mach-O"},
	{[]byte("\xce\xfa\xed\xfe"), "Mach-O"},
	{[]byte("\xca\xfe\xba\xbe"), "Mach-O 或 Java 类文件"},
	{[]byte("\xd0\xcf\x11\xe0"), "OLE"},
	{[]byte("SQLite format 3\x00"), "SQLite"},
}

// isoBrand 是 ISO 基础媒体文件格式里按图片接受的主品牌。
var isoBrand = map[string]bool{
	"heic": true, "heix": true, "heim": true, "heis": true,
	"hevc": true, "hevx": true, "mif1": true, "msf1": true,
	"avif": true, "avis": true,
}

// imageFormat 按文件头判断图片格式。
func imageFormat(data []byte) (string, bool) {
	for _, sig := range imageSignatures {
		if bytes.HasPrefix(data, sig.prefix) {
			return sig.label, true
		}
	}
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "WebP", true
	}
	if len(data) >= 12 && string(data[4:8]) == "ftyp" && isoBrand[string(data[8:12])] {
		return "HEIF/AVIF", true
	}
	return "", false
}

// binaryFormat 按文件头判断已知的二进制容器格式。
func binaryFormat(data []byte) (string, bool) {
	for _, sig := range binarySignatures {
		if bytes.HasPrefix(data, sig.prefix) {
			return sig.label, true
		}
	}
	return "", false
}

// classifyHeader 用于在读取全量内容前决定适用哪一档单文件上限。
// 图片与已知二进制格式只看文件头即可确定；其余一律先按文本处理，
// 由 classifyContent 在全量内容上复核，复核不通过即拒绝。
func classifyHeader(data []byte) (Category, string, bool) {
	if _, ok := imageFormat(data); ok {
		return CategoryImage, "", true
	}
	if label, ok := binaryFormat(data); ok {
		return "", "文件头识别为 " + label + "，不是文本也不是图片", false
	}
	return CategoryText, "", true
}

// classifyContent 按完整内容判定类别。判定只依据内容，不依据扩展名。
func classifyContent(data []byte) (Category, string, bool) {
	if _, ok := imageFormat(data); ok {
		return CategoryImage, "", true
	}
	if label, ok := binaryFormat(data); ok {
		return "", "文件头识别为 " + label + "，不是文本也不是图片", false
	}
	if reason, ok := textLike(data); !ok {
		return "", reason, false
	}
	return CategoryText, "", true
}

// textLike 判断内容是否可当作文本处理。
// 规则：出现 NUL 字节即视为二进制；除常见空白与控制字符外的控制字节占比超过 1% 视为二进制；
// 其余按文本处理，因此 UTF-8 与常见单字节编码都能通过。
func textLike(data []byte) (string, bool) {
	if len(data) == 0 {
		return "", true
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return "内容含 NUL 字节，按二进制处理", false
	}
	controls := 0
	for _, b := range data {
		if b >= 0x20 && b != 0x7f {
			continue
		}
		switch b {
		case '\t', '\n', '\v', '\f', '\r', 0x1b:
			// 制表、换行、换页与 ANSI 转义属于文本常见的控制字符。
		default:
			controls++
		}
	}
	if controls > 0 && controls*100 > len(data) {
		return "控制字节占比过高，按二进制处理", false
	}
	return "", true
}
