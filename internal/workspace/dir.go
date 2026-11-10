package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// EntryKind 是目录条目的文件类型。
type EntryKind string

const (
	// EntryFile 表示普通文件。
	EntryFile EntryKind = "file"
	// EntryDir 表示目录。
	EntryDir EntryKind = "dir"
	// EntrySymlink 表示符号链接；列目录不跟随链接。
	EntrySymlink EntryKind = "symlink"
	// EntryOther 表示设备、管道、套接字等非普通条目。
	EntryOther EntryKind = "other"
)

// DefaultMaxDirEntries 是单次列目录的默认条目上限，避免一次列出几十万条。
const DefaultMaxDirEntries = 5000

// Entry 是目录列表中的一条。
type Entry struct {
	Name    string
	Kind    EntryKind
	Size    int64
	ModTime time.Time
	// Git 是该条目相对工作区的 git 状态；工作区不在仓库内或未启用时为 GitNone。
	// 目录的状态由它下面任意条目的状态聚合而来。
	Git GitState
}

// DirListing 是一次目录读取的结果。
type DirListing struct {
	// Path 是相对工作区的目录路径，根目录为 "."。
	Path    string
	Entries []Entry
	// Total 是磁盘上的实际条目数（含被上限截掉的部分）。
	Total     int
	Truncated bool
	// GitAvailable 表示本次读取拿到了 git 状态。
	GitAvailable bool
	// Note 说明截断或 git 不可用的原因；为空表示没有需要说明的情况。
	Note string
}

// ListOptions 控制目录读取。
type ListOptions struct {
	// MaxEntries 为 0 时用 DefaultMaxDirEntries，为负时不限制。
	MaxEntries int
	// SkipGit 为 true 时不读取 git 状态。
	SkipGit bool
}

// ListDir 读取工作区内某个目录的条目，包含隐藏文件，按名称升序。
// git 状态读取失败不影响列表本身，只在 Note 里说明。
func (w *Workspace) ListDir(rel string, opts ListOptions) (DirListing, error) {
	dirAbs, err := w.Resolve(rel)
	if err != nil {
		return DirListing{}, err
	}
	info, err := os.Stat(dirAbs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return DirListing{}, fmt.Errorf("%w: %s", ErrPathNotExist, rel)
		}
		return DirListing{}, fmt.Errorf("读取目录 %s: %w", rel, err)
	}
	if !info.IsDir() {
		return DirListing{}, fmt.Errorf("%w: %s", ErrNotDirectory, rel)
	}
	des, err := os.ReadDir(dirAbs)
	if err != nil {
		return DirListing{}, fmt.Errorf("读取目录 %s: %w", rel, err)
	}

	limit := opts.MaxEntries
	if limit == 0 {
		limit = DefaultMaxDirEntries
	}
	snap := gitSnapshot{reason: "未读取 git 状态"}
	if !opts.SkipGit {
		snap = w.gitSnapshot()
	}

	listing := DirListing{
		Path:         w.displayPath(dirAbs),
		Total:        len(des),
		GitAvailable: snap.available,
	}
	for i, de := range des {
		if limit > 0 && i >= limit {
			listing.Truncated = true
			break
		}
		entry := Entry{Name: de.Name(), Kind: kindOf(de), Git: GitNone}
		if entryInfo, infoErr := de.Info(); infoErr == nil {
			entry.Size = entryInfo.Size()
			entry.ModTime = entryInfo.ModTime()
		}
		if snap.available {
			absEntry := filepath.Join(dirAbs, de.Name())
			if relEntry, relErr := filepath.Rel(w.root, absEntry); relErr == nil {
				entry.Git = snap.status(filepath.Clean(relEntry), entry.Kind)
			}
		}
		listing.Entries = append(listing.Entries, entry)
	}
	if listing.Truncated {
		listing.Note = fmt.Sprintf("已显示前 %d 条，共 %d 条（已截断）", len(listing.Entries), listing.Total)
	}
	if !opts.SkipGit && !snap.available {
		listing.Note = joinNotes(listing.Note, snap.reason)
	}
	return listing, nil
}

// kindOf 按条目类型分类，不跟随符号链接。
func kindOf(de fs.DirEntry) EntryKind {
	mode := de.Type()
	switch {
	case mode&fs.ModeSymlink != 0:
		return EntrySymlink
	case mode.IsDir():
		return EntryDir
	case mode.IsRegular():
		return EntryFile
	default:
		return EntryOther
	}
}
