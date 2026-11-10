package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ChangeCoverage 表示变更清单的覆盖级别。
type ChangeCoverage string

const (
	// CoverageToolReports 表示清单只来自工具结果声明的路径。
	// 这是工具声明级的部分覆盖，不是权威的 run 级改动清单。
	CoverageToolReports ChangeCoverage = "tool_reports"
)

// MaxChangePaths 是单次变更读取返回的路径上限。
const MaxChangePaths = 256

// ToolResultDeclaration 是投影给出的一个工具结果的路径声明。
// 同一个工具结果可以声明多个路径；不声明路径时 Paths 为空。
type ToolResultDeclaration struct {
	ToolCallID string
	ToolName   string
	Paths      []string
}

// ChangeSource 是集成层注入的最小接口：按会话与 run 返回工具结果的路径声明。
// 适配器负责从投影的工具结果中取出 structuredContent.affectedPaths、
// write/edit 声明的 path 与 fileWritten effect；本包不导入投影内部。
type ChangeSource interface {
	ToolResultDeclarations(ctx context.Context, sessionID, runID string) ([]ToolResultDeclaration, error)
}

// ChangeScope 是路径相对工作区的位置。
type ChangeScope string

const (
	// ChangeWorkspace 表示路径在工作区内。
	ChangeWorkspace ChangeScope = "workspace"
	// ChangeExternal 表示路径在工作区外。
	ChangeExternal ChangeScope = "external"
)

// RunChange 是一条声明路径的规范化结果。
type RunChange struct {
	// Path 在工作区内是相对路径（"/" 分隔），在工作区外是绝对路径。
	Path string
	// AbsPath 是解析符号链接后的绝对路径，可直接交给外部程序或界面使用。
	AbsPath string
	Scope   ChangeScope
	// Exists 表示当前磁盘上能否 Stat 到该路径；声明删除的文件为 false。
	Exists bool
}

// RunChanges 是一次 run 的变更读取结果。
type RunChanges struct {
	SessionID string
	RunID     string
	// Root 是工作区根目录的绝对路径。
	Root  string
	Paths []RunChange
	// Total 是去重后的路径总数（含被上限截掉的部分）。
	Total     int
	Truncated bool
	Coverage  ChangeCoverage
	// Declarations 是本次读取到的工具结果声明条数。
	Declarations int
	// Note 说明截断原因；为空表示没有需要说明的情况。
	Note string
}

// ReadRunChanges 读取指定 run 的工具结果声明，输出规范化后的路径清单。
//
// 纪律：
//   - 只消费 source 给出的声明，不扫描工作区、不推断 diff、不读 git 历史；
//   - 相对路径按工作区根规范化，符号链接解析后重新判断工作区内/外；
//   - 按 scope 去重，工作区内在前、同类按路径字节序排序；
//   - 不落盘、不缓存，一次调用一次结果。
//
// source 为 nil 或没有声明时返回空清单，不触发任何目录扫描。
func (w *Workspace) ReadRunChanges(ctx context.Context, source ChangeSource, sessionID, runID string) (RunChanges, error) {
	if w == nil || w.root == "" {
		return RunChanges{}, errWorkspaceNotReady
	}
	if err := ctx.Err(); err != nil {
		return RunChanges{}, err
	}
	result := RunChanges{
		SessionID: sessionID,
		RunID:     runID,
		Root:      w.root,
		Paths:     make([]RunChange, 0),
		Coverage:  CoverageToolReports,
	}
	if source == nil {
		return result, nil
	}
	declarations, err := source.ToolResultDeclarations(ctx, sessionID, runID)
	if err != nil {
		return RunChanges{}, fmt.Errorf("读取工具结果声明: %w", err)
	}
	result.Declarations = len(declarations)

	seen := make(map[string]struct{})
	for _, declaration := range declarations {
		for _, rawPath := range declaration.Paths {
			if err := ctx.Err(); err != nil {
				return RunChanges{}, err
			}
			change, ok := w.classifyChangePath(rawPath)
			if !ok {
				continue
			}
			key := string(change.Scope) + "\x00" + change.Path
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			result.Paths = append(result.Paths, change)
		}
	}

	sort.Slice(result.Paths, func(i, j int) bool {
		left, right := result.Paths[i], result.Paths[j]
		if left.Scope != right.Scope {
			return left.Scope == ChangeWorkspace
		}
		return left.Path < right.Path
	})
	result.Total = len(result.Paths)
	if result.Total > MaxChangePaths {
		result.Paths = result.Paths[:MaxChangePaths]
		result.Truncated = true
		result.Note = fmt.Sprintf("已声明 %d 条路径，按上限保留前 %d 条（已截断）", result.Total, MaxChangePaths)
	}
	return result, nil
}

// classifyChangePath 规范化一条声明路径并判定它在工作区内还是外。
// 空路径与工作区根目录本身不算改动。
func (w *Workspace) classifyChangePath(rawPath string) (RunChange, bool) {
	value := strings.TrimSpace(rawPath)
	if value == "" {
		return RunChange{}, false
	}
	value = expandChangeHome(value)
	p := value
	if !filepath.IsAbs(p) {
		p = filepath.Join(w.root, p)
	}
	p = filepath.Clean(p)
	resolved, err := resolveExistingPrefix(p)
	if err != nil {
		// 解析失败时退回字面绝对路径，仍然给出可判断的清单项。
		resolved = p
	}
	if resolved == w.root {
		return RunChange{}, false
	}
	change := RunChange{AbsPath: resolved, Exists: pathExists(resolved)}
	if within(w.root, resolved) {
		rel, relErr := filepath.Rel(w.root, resolved)
		if relErr != nil {
			return RunChange{}, false
		}
		change.Scope = ChangeWorkspace
		change.Path = filepath.ToSlash(rel)
		return change, true
	}
	change.Scope = ChangeExternal
	change.Path = resolved
	return change, true
}

// expandChangeHome 展开 ~/ 与 ~\ 前缀，与产品基线一致；其他形式保持原样。
func expandChangeHome(p string) string {
	if !strings.HasPrefix(p, "~/") && !strings.HasPrefix(p, `~\`) {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	return filepath.Join(home, p[2:])
}

// pathExists 表示当前磁盘上能否 Stat 到路径；权限不足按不存在处理。
func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
