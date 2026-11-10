package settings

import (
	"context"
	"encoding/json"
	"fmt"
)

// Source 是服务端声明的一个可配置来源。
type Source struct {
	// Name 是来源标识，必填。
	Name string
	// Scopes 是该来源声明的作用域；没有声明的作用域不可读写。
	Scopes []ScopeKind
	// UserOnly 是只能写入用户作用域的键；项目作用域写入时客户端先行拒绝。
	UserOnly []string
	// WriteOnly 表示服务端不通过该来源返回可读文档。
	WriteOnly bool
	// Schema 是服务端给出的结构描述；空表示服务端没有提供。
	Schema json.RawMessage
}

// Allows 判断该来源是否声明了某个作用域。
func (s Source) Allows(kind ScopeKind) bool { return containsScope(s.Scopes, kind) }

// Sources 是服务端声明的设置来源清单。
type Sources struct {
	// Dialect 是设置文档的方言；空表示服务端没有声明。
	Dialect string
	// Sources 是来源清单，顺序与服务端一致。
	Sources []Source
}

// Find 按来源名查找声明。
func (s Sources) Find(name string) (Source, bool) {
	for _, source := range s.Sources {
		if source.Name == name {
			return source, true
		}
	}
	return Source{}, false
}

// SourcesReader 读取服务端声明的设置来源。
type SourcesReader interface {
	ReadSettingsSources(ctx context.Context) (Sources, error)
}

// RefreshSources 读取设置来源清单并替换缓存。能力缺失或事实不合法时失败，
// 保留上一次确认值。
func (m *Manager) RefreshSources(ctx context.Context) (Sources, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := m.requireFeature(FeatureSettingsSources); err != nil {
		return Sources{}, err
	}
	if m.cfg.Sources == nil {
		return Sources{}, missing(FeatureSettingsSources, "未提供设置来源读取接口")
	}
	sources, err := m.cfg.Sources.ReadSettingsSources(ctx)
	if err != nil {
		return Sources{}, readFailure(FeatureSettingsSources, ErrSourcesRead, err)
	}
	if err := validateSources(sources); err != nil {
		return Sources{}, err
	}
	m.mu.Lock()
	m.sources = cloneSources(sources)
	m.sourcesOK = true
	m.mu.Unlock()
	return cloneSources(sources), nil
}

// Sources 返回最近一次确认的设置来源清单；第二个返回值表示是否读过。
func (m *Manager) Sources() (Sources, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.sourcesOK {
		return Sources{}, false
	}
	return cloneSources(m.sources), true
}

// declaredSource 返回已声明的来源；来源清单尚未读取时返回 ErrSourcesNotLoaded。
func (m *Manager) declaredSource(name string) (Source, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.sourcesOK {
		return Source{}, ErrSourcesNotLoaded
	}
	source, ok := m.sources.Find(name)
	if !ok {
		return Source{}, fmt.Errorf("%w: 来源 %q 未声明", ErrInvalidSource, name)
	}
	return source, nil
}

// validateSources 校验来源清单的结构。它不解释 Schema 的业务含义，只要求合法 JSON。
func validateSources(sources Sources) error {
	seen := make(map[string]bool, len(sources.Sources))
	for index, source := range sources.Sources {
		name, ok := trimNonEmpty(source.Name)
		if !ok {
			return fmt.Errorf("%w: 第 %d 个来源缺少名称", ErrMalformedSources, index)
		}
		if seen[name] {
			return fmt.Errorf("%w: 来源名重复 %q", ErrMalformedSources, name)
		}
		seen[name] = true
		scopes := make(map[ScopeKind]bool, len(source.Scopes))
		for _, raw := range source.Scopes {
			kind, ok := sourceScopeKind(string(raw))
			if !ok {
				return fmt.Errorf("%w: 来源 %q 的作用域 %q 未知", ErrMalformedSources, name, raw)
			}
			if scopes[kind] {
				return fmt.Errorf("%w: 来源 %q 的作用域 %q 重复", ErrMalformedSources, name, raw)
			}
			scopes[kind] = true
		}
		userOnly := make(map[string]bool, len(source.UserOnly))
		for _, key := range source.UserOnly {
			trimmed, ok := trimNonEmpty(key)
			if !ok {
				return fmt.Errorf("%w: 来源 %q 的只允许用户作用域键为空", ErrMalformedSources, name)
			}
			if userOnly[trimmed] {
				return fmt.Errorf("%w: 来源 %q 的只允许用户作用域键重复 %q", ErrMalformedSources, name, trimmed)
			}
			userOnly[trimmed] = true
		}
		if !validJSON(source.Schema) {
			return fmt.Errorf("%w: 来源 %q 的结构描述不是合法 JSON", ErrMalformedSources, name)
		}
	}
	return nil
}

// cloneSources 深拷贝来源清单，避免调用方改到缓存。
func cloneSources(sources Sources) Sources {
	cloned := Sources{Dialect: sources.Dialect, Sources: make([]Source, 0, len(sources.Sources))}
	for _, source := range sources.Sources {
		cloned.Sources = append(cloned.Sources, Source{
			Name:      source.Name,
			Scopes:    append([]ScopeKind(nil), source.Scopes...),
			UserOnly:  append([]string(nil), source.UserOnly...),
			WriteOnly: source.WriteOnly,
			Schema:    cloneRaw(source.Schema),
		})
	}
	return cloned
}
