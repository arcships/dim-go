package settings

import (
	"context"
	"fmt"
)

// Project 是服务端授权的一个项目根。Root 是进程内编号，设置作用域用它定位
// 项目，而不是用路径。
type Project struct {
	// Root 是进程内项目根编号，必须大于 0。
	Root int
	// CWD 是服务端给出的项目目录；只用于展示与按目录解析，不用于拼路径。
	CWD string
}

// ProjectsReader 读取服务端授权的项目根。
type ProjectsReader interface {
	ReadSettingsProjects(ctx context.Context) ([]Project, error)
}

// RefreshProjects 读取已授权项目根并替换缓存。
func (m *Manager) RefreshProjects(ctx context.Context) ([]Project, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := m.requireFeature(FeatureSettingsProjects); err != nil {
		return nil, err
	}
	if m.cfg.Projects == nil {
		return nil, missing(FeatureSettingsProjects, "未提供项目根读取接口")
	}
	projects, err := m.cfg.Projects.ReadSettingsProjects(ctx)
	if err != nil {
		return nil, readFailure(FeatureSettingsProjects, ErrProjectsRead, err)
	}
	if err := validateProjects(projects); err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.projects = cloneProjects(projects)
	m.projectsOK = true
	m.mu.Unlock()
	return cloneProjects(projects), nil
}

// Projects 返回最近一次确认的项目根；第二个返回值表示是否读过。
func (m *Manager) Projects() ([]Project, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.projectsOK {
		return nil, false
	}
	return cloneProjects(m.projects), true
}

// ResolveProject 把项目目录解析成服务端授权的项目根。尚未读取项目根时先读一次。
// 目录必须与服务端给出的 CWD 完全一致；不做路径归一化猜测。
func (m *Manager) ResolveProject(ctx context.Context, cwd string) (Project, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	trimmed, ok := trimNonEmpty(cwd)
	if !ok {
		return Project{}, fmt.Errorf("%w: 项目目录为空", ErrProjectNotAuthorized)
	}
	if _, loaded := m.Projects(); !loaded {
		if _, err := m.RefreshProjects(ctx); err != nil {
			return Project{}, err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, project := range m.projects {
		if project.CWD == trimmed {
			return project, nil
		}
	}
	return Project{}, fmt.Errorf("%w: 目录 %q 不在授权清单中", ErrProjectNotAuthorized, trimmed)
}

// authorizeProject 要求项目根已经读取且确实在授权清单中。
func (m *Manager) authorizeProject(root int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.projectsOK {
		return ErrProjectsNotLoaded
	}
	for _, project := range m.projects {
		if project.Root == root {
			return nil
		}
	}
	return fmt.Errorf("%w: 项目根 %d 不在授权清单中", ErrProjectNotAuthorized, root)
}

// validateProjects 校验项目根的结构：编号大于 0 且不重复，目录非空且不重复。
func validateProjects(projects []Project) error {
	roots := make(map[int]bool, len(projects))
	dirs := make(map[string]bool, len(projects))
	for index, project := range projects {
		if project.Root <= 0 {
			return fmt.Errorf("%w: 第 %d 个项目的根编号不合法", ErrMalformedProjects, index)
		}
		if roots[project.Root] {
			return fmt.Errorf("%w: 项目根编号重复 %d", ErrMalformedProjects, project.Root)
		}
		roots[project.Root] = true
		cwd, ok := trimNonEmpty(project.CWD)
		if !ok {
			return fmt.Errorf("%w: 项目根 %d 缺少目录", ErrMalformedProjects, project.Root)
		}
		if dirs[cwd] {
			return fmt.Errorf("%w: 项目目录重复 %q", ErrMalformedProjects, cwd)
		}
		dirs[cwd] = true
	}
	return nil
}

func cloneProjects(projects []Project) []Project {
	return append([]Project(nil), projects...)
}
