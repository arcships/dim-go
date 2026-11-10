package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	// DefaultMaxMatches 是单次搜索返回的默认条数上限。
	DefaultMaxMatches = 100
	// DefaultMaxFiles 是单次搜索检查的默认文件数上限。
	DefaultMaxFiles = 20000
	// DefaultMaxFileBytes 是参与搜索的单个文件的默认大小上限（2 MiB）。
	DefaultMaxFileBytes = 2 << 20
	// DefaultMaxScanBytes 是单次搜索累计读取的默认字节上限（32 MiB）。
	DefaultMaxScanBytes = 32 << 20
	// DefaultSnippetRunes 是匹配片段保留的默认字符数。
	DefaultSnippetRunes = 200
)

// 搜索停止原因。
const (
	// StopNone 表示扫描完整结束。
	StopNone = ""
	// StopMaxMatches 表示达到匹配条数上限。
	StopMaxMatches = "max-matches"
	// StopMaxFiles 表示达到检查文件数上限。
	StopMaxFiles = "max-files"
	// StopMaxScanBytes 表示达到累计读取字节上限。
	StopMaxScanBytes = "max-scan-bytes"
)

// SearchOptions 控制工作区搜索。零值字段使用默认上限。
type SearchOptions struct {
	// Query 是关键词；大小写不敏感，按行匹配，一行最多返回一条。
	Query string
	// MaxMatches 为 0 时用 DefaultMaxMatches，为负时不限制。
	MaxMatches int
	// MaxFiles 为 0 时用 DefaultMaxFiles，为负时不限制。
	MaxFiles int
	// MaxFileBytes 为 0 时用 DefaultMaxFileBytes，为负时不限制。
	MaxFileBytes int64
	// MaxScanBytes 为 0 时用 DefaultMaxScanBytes，为负时不限制。
	MaxScanBytes int64
	// SnippetRunes 为 0 时用 DefaultSnippetRunes。
	SnippetRunes int
}

// SearchMatch 是一处命中。
type SearchMatch struct {
	// Path 是相对工作区根目录的路径，统一使用 "/" 分隔。
	Path string
	// Line 从 1 开始。
	Line int
	// Column 是命中处在该行的字符序号，从 1 开始；片段被裁掉行首空白后列号仍按原行计算。
	Column int
	// Snippet 是该行去掉首尾空白后的内容，超过上限时截断并以 "…" 结尾。
	Snippet string
}

// SearchResult 是一次搜索的结果与边界说明。
type SearchResult struct {
	Matches []SearchMatch
	// FilesScanned 是实际读取内容的普通文件数。
	FilesScanned int
	// BytesScanned 是实际读取的字节数。
	BytesScanned int64
	// Skipped 是因二进制、超限、符号链接或读失败而未参与匹配的条目数。
	Skipped int
	// StopReason 说明扫描为何提前结束；StopNone 表示完整扫描。
	StopReason string
	// Truncated 表示结果不完整，原因见 StopReason。
	Truncated bool
}

// Search 在工作区内一次性扫描关键词，不建索引、不缓存、不跟随符号链接。
// 跳过 .git 目录、非普通文件、二进制文件与超过单文件上限的文件。
func (w *Workspace) Search(rel string, opts SearchOptions) (SearchResult, error) {
	if strings.TrimSpace(opts.Query) == "" {
		return SearchResult{}, ErrEmptyQuery
	}
	root, err := w.Resolve(rel)
	if err != nil {
		return SearchResult{}, err
	}
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return SearchResult{}, fmt.Errorf("%w: %s", ErrPathNotExist, rel)
		}
		return SearchResult{}, fmt.Errorf("读取搜索起点 %s: %w", rel, err)
	}
	if !info.IsDir() {
		return SearchResult{}, fmt.Errorf("%w: %s", ErrNotDirectory, rel)
	}

	opts = opts.withDefaults()
	queryRunes := []rune(opts.Query)
	result := SearchResult{Matches: make([]SearchMatch, 0, opts.matchCapacity())}

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == root {
				return walkErr
			}
			result.Skipped++
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if result.StopReason != StopNone {
			return fs.SkipAll
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			result.Skipped++
			return nil
		}
		if !d.Type().IsRegular() {
			result.Skipped++
			return nil
		}
		if opts.MaxFiles >= 0 && result.FilesScanned >= opts.MaxFiles {
			result.stop(StopMaxFiles)
			return fs.SkipAll
		}
		fileInfo, infoErr := d.Info()
		if infoErr != nil {
			result.Skipped++
			return nil
		}
		if opts.MaxFileBytes >= 0 && fileInfo.Size() > opts.MaxFileBytes {
			result.Skipped++
			return nil
		}
		if opts.MaxScanBytes >= 0 && result.BytesScanned > 0 &&
			result.BytesScanned+fileInfo.Size() > opts.MaxScanBytes {
			result.stop(StopMaxScanBytes)
			return fs.SkipAll
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			result.Skipped++
			return nil
		}
		result.FilesScanned++
		result.BytesScanned += int64(len(data))
		if bytes.IndexByte(data, 0) >= 0 {
			result.Skipped++
			return nil
		}
		relPath := path
		if relToRoot, relErr := filepath.Rel(w.root, path); relErr == nil {
			relPath = filepath.ToSlash(relToRoot)
		}
		for lineNo, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSuffix(line, "\r")
			idx, matched := foldIndex(line, queryRunes)
			if !matched {
				continue
			}
			result.Matches = append(result.Matches, SearchMatch{
				Path:    relPath,
				Line:    lineNo + 1,
				Column:  utf8.RuneCountInString(line[:idx]) + 1,
				Snippet: snippetOf(line, opts.SnippetRunes),
			})
			if opts.MaxMatches >= 0 && len(result.Matches) >= opts.MaxMatches {
				result.stop(StopMaxMatches)
				return fs.SkipAll
			}
		}
		return nil
	})
	if walkErr != nil {
		return SearchResult{}, fmt.Errorf("搜索工作区: %w", walkErr)
	}
	return result, nil
}

// stop 记录停止原因。
func (r *SearchResult) stop(reason string) {
	r.StopReason = reason
	r.Truncated = true
}

// withDefaults 把零值字段补成默认上限，负值保留为"不限制"。
func (o SearchOptions) withDefaults() SearchOptions {
	if o.MaxMatches == 0 {
		o.MaxMatches = DefaultMaxMatches
	}
	if o.MaxFiles == 0 {
		o.MaxFiles = DefaultMaxFiles
	}
	if o.MaxFileBytes == 0 {
		o.MaxFileBytes = DefaultMaxFileBytes
	}
	if o.MaxScanBytes == 0 {
		o.MaxScanBytes = DefaultMaxScanBytes
	}
	if o.SnippetRunes <= 0 {
		o.SnippetRunes = DefaultSnippetRunes
	}
	return o
}

// matchCapacity 给结果切片一个合理的初始容量。
func (o SearchOptions) matchCapacity() int {
	switch {
	case o.MaxMatches < 0:
		return 16
	case o.MaxMatches > 1024:
		return 1024
	default:
		return o.MaxMatches
	}
}

// foldIndex 在行内做大小写不敏感匹配，返回命中的原始字节位置。
// 按字符比较而不是先整体小写，避免个别字符小写后字节数变化导致位置错位。
func foldIndex(line string, query []rune) (int, bool) {
	if len(query) == 0 {
		return 0, false
	}
	for i := 0; i < len(line); {
		end := i
		ok := true
		for range query {
			if end >= len(line) {
				ok = false
				break
			}
			_, size := utf8.DecodeRuneInString(line[end:])
			if size <= 0 {
				ok = false
				break
			}
			end += size
		}
		if ok && strings.EqualFold(line[i:end], string(query)) {
			return i, true
		}
		_, size := utf8.DecodeRuneInString(line[i:])
		if size <= 0 {
			break
		}
		i += size
	}
	return 0, false
}

// snippetOf 去掉行首尾空白，按字符数截断片段。
func snippetOf(line string, maxRunes int) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "…"
	}
	return trimmed
}
