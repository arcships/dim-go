package settings

import (
	"fmt"
	"strings"
)

// ScopeKind 是设置作用域的种类。
type ScopeKind string

const (
	// ScopeUser 是用户作用域。
	ScopeUser ScopeKind = "user"
	// ScopeProject 是项目作用域；项目以服务端给出的项目根编号标识。
	ScopeProject ScopeKind = "project"
)

// Scope 是一次设置读取或写入的作用域。
type Scope struct {
	Kind ScopeKind `json:"kind"`
	// Project 是服务端 `settings.projects` 返回的项目根编号；只在 ScopeProject 时使用。
	Project int `json:"project,omitempty"`
}

// UserScope 返回用户作用域。
func UserScope() Scope { return Scope{Kind: ScopeUser} }

// ProjectScope 返回一个项目作用域。root 必须来自已授权项目根清单。
func ProjectScope(root int) Scope { return Scope{Kind: ScopeProject, Project: root} }

// Validate 校验作用域字段的组合是否合法。
func (s Scope) Validate() error {
	switch s.Kind {
	case ScopeUser:
		if s.Project != 0 {
			return fmt.Errorf("%w: 用户作用域不能带项目根", ErrInvalidScope)
		}
		return nil
	case ScopeProject:
		if s.Project <= 0 {
			return fmt.Errorf("%w: 项目作用域缺少项目根", ErrInvalidScope)
		}
		return nil
	default:
		return fmt.Errorf("%w: 未知作用域 %q", ErrInvalidScope, s.Kind)
	}
}

// String 给出稳定的作用域描述，用于缓存键与错误信息。
func (s Scope) String() string {
	switch s.Kind {
	case ScopeUser:
		return "user"
	case ScopeProject:
		return fmt.Sprintf("project:%d", s.Project)
	default:
		return "invalid:" + string(s.Kind)
	}
}

func (s Scope) cacheKey(source string) string {
	return source + "\x00" + s.String()
}

// normalizeSourceName 去掉来源名两侧空白并拒绝空值。
func normalizeSourceName(source string) (string, error) {
	trimmed, ok := trimNonEmpty(source)
	if !ok {
		return "", fmt.Errorf("%w: 来源名为空", ErrInvalidSource)
	}
	return trimmed, nil
}

// sourceScope 判断一个声明的作用域是否等于目标种类。
func sourceScopeKind(raw string) (ScopeKind, bool) {
	switch ScopeKind(raw) {
	case ScopeUser:
		return ScopeUser, true
	case ScopeProject:
		return ScopeProject, true
	default:
		return "", false
	}
}

// containsScope 判断声明的作用域集合里是否有目标种类。
func containsScope(scopes []ScopeKind, kind ScopeKind) bool {
	for _, scope := range scopes {
		if scope == kind {
			return true
		}
	}
	return false
}

// containsKey 判断列表里是否已有某个键（区分大小写，比较前去掉空白）。
func containsKey(keys []string, key string) bool {
	for _, candidate := range keys {
		if strings.TrimSpace(candidate) == key {
			return true
		}
	}
	return false
}
