package settings

import (
	"context"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// RuntimeSource 是运行组件的来源。
type RuntimeSource string

const (
	// RuntimeSourceDownload 是下载获取。
	RuntimeSourceDownload RuntimeSource = "download"
	// RuntimeSourceSystem 是使用系统安装。
	RuntimeSourceSystem RuntimeSource = "system"
)

// RuntimeStatus 是运行组件状态。
type RuntimeStatus string

const (
	RuntimeStatusOff       RuntimeStatus = "off"
	RuntimeStatusAcquiring RuntimeStatus = "acquiring"
	RuntimeStatusFailed    RuntimeStatus = "failed"
	RuntimeStatusReady     RuntimeStatus = "ready"
	RuntimeStatusStarting  RuntimeStatus = "starting"
	RuntimeStatusDown      RuntimeStatus = "down"
)

// RuntimeEntry 是一个运行组件的状态。字段全部来自服务端；空的可选字段表示
// 服务端没有给出，不由客户端补齐。
type RuntimeEntry struct {
	Name    string
	Source  RuntimeSource
	Auto    bool
	Version string
	Path    string
	Status  RuntimeStatus
	Reason  string
}

// RuntimeReader 读取运行组件清单。实现方只读，不自行下载或安装。
type RuntimeReader interface {
	ReadRuntimes(ctx context.Context) ([]RuntimeEntry, error)
}

// runtimeInstallParams 是交给写登记的透明参数形状。
type runtimeInstallParams struct {
	Op   string `json:"op"`
	Name string `json:"name"`
}

// RefreshRuntimes 读取运行组件清单并替换缓存。能力缺失时返回分类错误。
func (m *Manager) RefreshRuntimes(ctx context.Context) ([]RuntimeEntry, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := m.requireFeature(FeatureRuntimeList); err != nil {
		return nil, err
	}
	if m.cfg.Runtimes == nil {
		return nil, missing(FeatureRuntimeList, "未提供运行组件读取接口")
	}
	entries, err := m.cfg.Runtimes.ReadRuntimes(ctx)
	if err != nil {
		return nil, readFailure(FeatureRuntimeList, ErrRuntimesRead, err)
	}
	if err := validateRuntimes(entries); err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.runtimes = cloneRuntimes(entries)
	m.runtimesOK = true
	m.mu.Unlock()
	return cloneRuntimes(entries), nil
}

// Runtimes 返回最近一次确认的运行组件清单；第二个返回值表示是否读过。
func (m *Manager) Runtimes() ([]RuntimeEntry, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.runtimesOK {
		return nil, false
	}
	return cloneRuntimes(m.runtimes), true
}

// Runtime 返回单个运行组件的最近一次确认状态。
func (m *Manager) Runtime(name string) (RuntimeEntry, bool) {
	trimmed, ok := trimNonEmpty(name)
	if !ok {
		return RuntimeEntry{}, false
	}
	entries, ok := m.Runtimes()
	if !ok {
		return RuntimeEntry{}, false
	}
	for _, entry := range entries {
		if entry.Name == trimmed {
			return entry, true
		}
	}
	return RuntimeEntry{}, false
}

// InstallRuntime 把一次安装转达给服务端：经写登记登记一次
// KindManagementWrite，等待终态，成功后重读清单取该项目。
//
// 本包不下载、不校验、不写本地组件文件；安装失败或结果未知时返回分类错误，
// 不自行重试。成功但重读失败时同样返回错误：状态未确认就不冒充已就绪。
func (m *Manager) InstallRuntime(ctx context.Context, name string) (RuntimeEntry, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	trimmed, ok := trimNonEmpty(name)
	if !ok {
		return RuntimeEntry{}, ErrInvalidRuntimeName
	}
	if err := m.requireFeature(FeatureRuntimeInstall); err != nil {
		return RuntimeEntry{}, err
	}
	if m.writer() == nil {
		return RuntimeEntry{}, fmt.Errorf("%w: %s", ErrWriteUnavailable, "未提供写登记")
	}
	view, err := m.registerWrite(ctx, runtimeInstallParams{Op: "runtime.install", Name: trimmed})
	if err != nil {
		return RuntimeEntry{}, err
	}
	if view.Status != write.StatusSucceeded {
		reason := view.Reason
		if reason == "" {
			reason = string(view.Status)
		}
		return RuntimeEntry{}, fmt.Errorf("%w: %s", ErrInstallFailed, reason)
	}
	entries, err := m.RefreshRuntimes(ctx)
	if err != nil {
		return RuntimeEntry{}, fmt.Errorf("安装已登记成功，但重读运行组件失败: %w", err)
	}
	for _, entry := range entries {
		if entry.Name == trimmed {
			return entry, nil
		}
	}
	return RuntimeEntry{}, fmt.Errorf("%w: %s", ErrRuntimeEntryMissing, trimmed)
}

// validateRuntimes 校验清单结构：名称非空且唯一，来源与状态都是已知取值。
func validateRuntimes(entries []RuntimeEntry) error {
	seen := make(map[string]bool, len(entries))
	for index, entry := range entries {
		name, ok := trimNonEmpty(entry.Name)
		if !ok {
			return fmt.Errorf("%w: 第 %d 个运行组件缺少名称", ErrMalformedRuntimes, index)
		}
		if seen[name] {
			return fmt.Errorf("%w: 运行组件名重复 %q", ErrMalformedRuntimes, name)
		}
		seen[name] = true
		switch entry.Source {
		case RuntimeSourceDownload, RuntimeSourceSystem:
		default:
			return fmt.Errorf("%w: 运行组件 %q 的来源 %q 未知", ErrMalformedRuntimes, name, entry.Source)
		}
		switch entry.Status {
		case RuntimeStatusOff, RuntimeStatusAcquiring, RuntimeStatusFailed,
			RuntimeStatusReady, RuntimeStatusStarting, RuntimeStatusDown:
		default:
			return fmt.Errorf("%w: 运行组件 %q 的状态 %q 未知", ErrMalformedRuntimes, name, entry.Status)
		}
	}
	return nil
}

func cloneRuntimes(entries []RuntimeEntry) []RuntimeEntry {
	return append([]RuntimeEntry(nil), entries...)
}
