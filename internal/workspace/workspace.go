// Package workspace 提供工作区内本机文件与 git 的只读能力。
//
// 职责边界：
//   - 只读：目录列表、文件预览、关键词搜索、git 状态、外部打开。
//   - 不写工作区文件，不建索引，不缓存；每次调用的结果用完即弃。
//   - 不承载界面、连接、会话与附件暂存；不提供 commit、checkout、stage 等写操作。
//
// 所有入口先规范化路径，再判断是否位于工作区内；符号链接解析完成后按新路径重新校验。
package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

var (
	// ErrOutsideWorkspace 表示规范化后的路径落在工作区之外。
	ErrOutsideWorkspace = errors.New("路径越出工作区")
	// ErrPathNotExist 表示路径在磁盘上不存在。
	ErrPathNotExist = errors.New("路径不存在")
	// ErrNotDirectory 表示路径存在但不是目录。
	ErrNotDirectory = errors.New("路径不是目录")
	// ErrNotRegularFile 表示路径存在但不是普通文件。
	ErrNotRegularFile = errors.New("路径不是普通文件")
	// ErrIsDirectory 表示该操作要求文件但拿到的是目录。
	ErrIsDirectory = errors.New("路径是目录")
	// ErrEmptyPath 表示调用方给了空的根目录。
	ErrEmptyPath = errors.New("路径为空")
	// ErrGitUnavailable 表示本机没有可执行的 git。
	ErrGitUnavailable = errors.New("本机没有可用的 git")
	// ErrUnsupportedPlatform 表示当前平台没有登记的外部打开方式。
	ErrUnsupportedPlatform = errors.New("当前平台不支持外部打开")
	// ErrOpenFailed 表示外部打开的子进程返回了失败。
	ErrOpenFailed = errors.New("外部打开失败")
	// ErrEmptyQuery 表示搜索关键词为空。
	ErrEmptyQuery = errors.New("搜索关键词为空")
	// ErrPreviewLimitTooLarge 表示请求的预览上限超过硬上限。
	ErrPreviewLimitTooLarge = errors.New("预览上限超过允许的最大值")

	errWorkspaceNotReady = errors.New("工作区未初始化")
)

// Workspace 表示一个已规范化的只读工作区根目录。
// 根目录在创建时解析符号链接，之后的路径判断都以该真实路径为基准。
type Workspace struct {
	root string
}

// NewWorkspace 创建只读工作区。root 必须存在且是目录。
func NewWorkspace(root string) (*Workspace, error) {
	if strings.TrimSpace(root) == "" {
		return nil, ErrEmptyPath
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("规范化工作区路径: %w", err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrPathNotExist, root)
		}
		return nil, fmt.Errorf("解析工作区路径: %w", err)
	}
	info, err := os.Stat(real)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrPathNotExist, root)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotDirectory, root)
	}
	return &Workspace{root: real}, nil
}

// Root 返回解析符号链接后的工作区绝对路径。
func (w *Workspace) Root() string {
	if w == nil {
		return ""
	}
	return w.root
}

// Resolve 把调用方给的路径规范化成工作区内的绝对路径。
// rel 可以为相对路径、绝对路径或空（空等同根目录）。
// 越界路径返回 ErrOutsideWorkspace；符号链接解析后越界同样被拒绝。
func (w *Workspace) Resolve(rel string) (string, error) {
	if w == nil || w.root == "" {
		return "", errWorkspaceNotReady
	}
	p := rel
	if p == "" {
		p = "."
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(w.root, p)
	}
	p = filepath.Clean(p)
	resolved, err := resolveExistingPrefix(p)
	if err != nil {
		return "", err
	}
	if !within(w.root, resolved) {
		return "", fmt.Errorf("%w: %s", ErrOutsideWorkspace, rel)
	}
	return resolved, nil
}

// Relative 返回规范化后的路径相对工作区根目录的表示；根目录返回 "."。
func (w *Workspace) Relative(p string) (string, error) {
	abs, err := w.Resolve(p)
	if err != nil {
		return "", err
	}
	return w.displayPath(abs), nil
}

// displayPath 把已确认位于工作区内的绝对路径转成相对路径。
func (w *Workspace) displayPath(abs string) string {
	rel, err := filepath.Rel(w.root, abs)
	if err != nil || rel == "." {
		return "."
	}
	return rel
}

// resolveExistingPrefix 解析路径中已存在前缀的符号链接，再把不存在的尾段接回去。
// 这样不存在的目标也能得到稳定路径，越界判断在解析结果上完成。
func resolveExistingPrefix(p string) (string, error) {
	current := p
	var missing []string
	for depth := 0; depth < 64; depth++ {
		real, err := filepath.EvalSymlinks(current)
		if err == nil {
			if len(missing) == 0 {
				return real, nil
			}
			info, statErr := os.Stat(real)
			if statErr != nil || !info.IsDir() {
				return "", fmt.Errorf("%w: %s", ErrNotDirectory, p)
			}
			parts := make([]string, 0, len(missing)+1)
			parts = append(parts, real)
			for i := len(missing) - 1; i >= 0; i-- {
				parts = append(parts, missing[i])
			}
			return filepath.Join(parts...), nil
		}
		if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
			return "", fmt.Errorf("解析路径符号链接: %w", err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("%w: %s", ErrPathNotExist, p)
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
	return "", fmt.Errorf("解析路径符号链接层数过多: %s", p)
}

// within 判断 p 是否等于 root 或位于 root 之下，按路径分隔符边界比较。
func within(root, p string) bool {
	if p == root {
		return true
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return !filepath.IsAbs(rel)
}
