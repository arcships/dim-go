package extensions

import (
	"context"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// Plugin 是一个插件条目。未安装的可选插件也出现在清单里：Optional 为真、
// Installed 为假，不因为未安装而被过滤掉。
type Plugin struct {
	ID        string
	Name      string
	Version   string
	Source    Source
	SourceRef string
	Installed bool
	Enabled   bool
	// Optional 表示随包提供、可由用户选择安装的可选插件。
	Optional bool
	// Use 是服务端声明的使用方式（例如 builtin）；空表示未启用。
	Use string
	// Locked 表示服务端锁定该插件，用户不能启停。
	Locked bool
	// Status 与 Error 是服务端声明的运行状态与原因。
	Status string
	Error  string
	// UpdateAvailable 与 LatestVersion 由服务端声明，本端不比较版本号。
	UpdateAvailable bool
	LatestVersion   string
	Description     string
}

// PluginReader 读取插件清单。只读，不安装、不探测。
type PluginReader interface {
	ReadPlugins(ctx context.Context) ([]Plugin, error)
}

// InstallSource 是一次安装的来源。Kind 取 local、git 或 bundled；Ref 是本地路径、
// Git 地址或随包插件标识。
type InstallSource struct {
	Kind Source `json:"kind"`
	Ref  string `json:"ref"`
}

func (s InstallSource) validate() error {
	if !s.Kind.installSourceKind() {
		return invalidRequest("安装来源 %q 不支持", s.Kind)
	}
	return requireText("安装来源引用", s.Ref)
}

// PluginInstallPayload 是 plugin.install 意图的载荷。
type PluginInstallPayload struct {
	Source InstallSource `json:"source"`
}

// PluginEnabledPayload 是 plugin.set_enabled 意图的载荷。
type PluginEnabledPayload struct {
	PluginID string `json:"pluginId"`
	Enabled  bool   `json:"enabled"`
}

// PluginUpdatePayload 是 plugin.update 意图的载荷。
type PluginUpdatePayload struct {
	PluginID string `json:"pluginId"`
}

// ListPlugins 读取插件清单并更新镜像读数。缺能力时返回已经确认的旧读数与分类错误。
func (s *Service) ListPlugins(ctx context.Context) (Snapshot[Plugin], error) {
	if s.cfg.Plugins == nil {
		return missingReader(s, "plugin.list", &s.plugins)
	}
	return readInto(s, ctx, "plugin.list", &s.plugins, s.cfg.Plugins.ReadPlugins)
}

// PluginSnapshot 返回最近一次插件读数，不发起请求。
func (s *Service) PluginSnapshot() Snapshot[Plugin] {
	return snapshotOf(s, &s.plugins)
}

// InstallPlugin 从给定来源安装插件。写入经写登记，只发一次。
func (s *Service) InstallPlugin(ctx context.Context, source InstallSource) (write.IntentView, error) {
	if err := source.validate(); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpPluginInstall, PluginInstallPayload{Source: source}, false)
}

// SetPluginEnabled 写入插件的启用状态。集成层按 Enabled 映射到真实方法
// plugin.enable 或 plugin.disable；相同目标状态的重复调用只保留一条意图。
func (s *Service) SetPluginEnabled(ctx context.Context, pluginID string, enabled bool) (write.IntentView, error) {
	if err := requireText("插件标识", pluginID); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpPluginSetEnabled, PluginEnabledPayload{PluginID: pluginID, Enabled: enabled}, false)
}

// UpdatePlugin 触发一次插件更新。每次显式调用都是一次新的检查，成功后再次调用
// 会登记新意图；重复点击在第一次未结项时仍只有一条。
func (s *Service) UpdatePlugin(ctx context.Context, pluginID string) (write.IntentView, error) {
	if err := requireText("插件标识", pluginID); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpPluginUpdate, PluginUpdatePayload{PluginID: pluginID}, true)
}
