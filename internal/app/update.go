package app

import (
	"context"
	"net/http"

	"github.com/arcships/dimagent-mygo/internal/host"
)

// UpdateOptions 是应用更新的装配参数。
//
// 版本源默认取现有客户端的渠道约定（基址 https://dimcode.echooai.com、
// 通道 updates/stable，见 D10 §3.7）；单文件目标就是本进程的可执行文件。
type UpdateOptions struct {
	// CurrentVersion 是当前版本，必须形如 x.y.z；ReadVersion 优先。
	CurrentVersion string
	// ReadVersion 可选：每次读取安装产物里的版本。
	ReadVersion func() (string, error)
	// ManifestURL 是完整清单地址；为空时按 BaseURL/ChannelPath 推导。
	ManifestURL string
	// BaseURL 是更新源基址；空时用现有客户端的默认值。
	BaseURL string
	// ChannelPath 是更新通道；空时用 updates/stable。
	ChannelPath string
	// TempDir 是下载落点；空时由更新编排自建并回收。
	TempDir string
	// TargetPath 是要替换的单文件产物；切换时必填。
	TargetPath string
	// HTTPClient 可选，测试注入本地版本源。
	HTTPClient *http.Client
	// Restart 是替换后启动新进程的步骤；Switch 时必填。
	Restart func(context.Context) error
	// Settle 在切换前尝试收尾会话；为 nil 时直接推迟。
	Settle func(context.Context) error
}

// EnableUpdate 按参数装配更新编排。
//
// 会话是否空闲由本实例的打开会话数回答；退出五阶段只在接到宿主外壳后补上
// （见 bindUpdateHost）。诊断沿用宿主外壳的白名单记录。
func (a *App) EnableUpdate(options UpdateOptions) (*host.Updater, error) {
	cfg := host.UpdateConfig{
		CurrentVersion: options.CurrentVersion,
		ReadVersion:    options.ReadVersion,
		ManifestURL:    options.ManifestURL,
		BaseURL:        options.BaseURL,
		ChannelPath:    options.ChannelPath,
		HTTPClient:     options.HTTPClient,
		TempDir:        options.TempDir,
		TargetPath:     options.TargetPath,
		SessionsIdle:   a.SessionsIdle,
		Settle:         options.Settle,
		Restart:        options.Restart,
		Diag:           a.updateDiag,
	}
	updater, err := host.NewUpdater(cfg)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	previous := a.updater
	a.updater = updater
	a.updateOptions = options
	a.mu.Unlock()
	if previous != nil {
		_ = previous.Close()
	}
	return updater, nil
}

// Updater 返回已装配的更新编排；未装配时为 nil。
func (a *App) Updater() *host.Updater {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.updater
}

// SessionsIdle 回答当前是否有打开中的会话；更新切换用它决定是否推迟。
func (a *App) SessionsIdle() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.sessions) == 0
}

// bindUpdateHost 在宿主外壳创建后补上更新编排需要的两段：退出五阶段与诊断。
//
// 只改更新编排的接线，不复制外壳的阶段顺序：Shutdown 直接调用外壳的退出编排，
// 诊断沿用同一个白名单记录。
func (a *App) bindUpdateHost(shell *host.Shell, diag *host.Recorder) error {
	a.mu.Lock()
	options := a.updateOptions
	enabled := a.updater != nil
	a.mu.Unlock()
	if !enabled || shell == nil {
		return nil
	}
	cfg := host.UpdateConfig{
		CurrentVersion: options.CurrentVersion,
		ReadVersion:    options.ReadVersion,
		ManifestURL:    options.ManifestURL,
		BaseURL:        options.BaseURL,
		ChannelPath:    options.ChannelPath,
		HTTPClient:     options.HTTPClient,
		TempDir:        options.TempDir,
		TargetPath:     options.TargetPath,
		SessionsIdle:   a.SessionsIdle,
		Settle:         options.Settle,
		Restart:        options.Restart,
		Diag:           diag,
		Shutdown: func(ctx context.Context) error {
			_, err := shell.Shutdown(ctx)
			return err
		},
	}
	updater, err := host.NewUpdater(cfg)
	if err != nil {
		return err
	}
	a.mu.Lock()
	previous := a.updater
	a.updater = updater
	a.mu.Unlock()
	if previous != nil {
		_ = previous.Close()
	}
	return nil
}

// CloseUpdate 回收更新编排的临时产物；未装配时什么也不做。
func (a *App) CloseUpdate() error {
	a.mu.Lock()
	updater := a.updater
	a.updater = nil
	a.mu.Unlock()
	if updater == nil {
		return nil
	}
	return updater.Close()
}

func (a *App) closeUpdate() error { return a.CloseUpdate() }
