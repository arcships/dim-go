package uiwire

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/arcships/dimagent-mygo/internal/host"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// 本文件把宿主本端的自动更新接给设置 › 关于：事实走 ui.UpdateFacts，动作走
// ui.UpdateIntents。它不是 App Server 的事实，所以不占 Source 的必选方法。
//
// 更新源来自本地偏好（preferences.json 的 updateFeedURL）：空表示未配置，
// 界面显示「未配置更新源」并禁用入口；填基址时按 `<基址>/latest.json` 读取。
//
// 下载落在 `<数据目录>/updates/`。安装需要宿主能力（解包替换 .app），本版没有
// 接，所以 Install 传 nil、原因取 ui.NoteUpdateInstall：入口保留但置灰，不写
// 假回执。

const (
	// updateCheckTimeout 是手动检查的超时。
	updateCheckTimeout = 30 * time.Second
	// updateDownloadTimeout 是下载与校验的超时。
	updateDownloadTimeout = 12 * time.Minute
	// updateAutoCheckDelay 是启动后延迟一次的检查。
	updateAutoCheckDelay = 8 * time.Second
)

// Update 返回更新相位的只读快照（实现 ui.UpdateFacts）。
func (w *App) Update() ui.UpdateSnapshot {
	updater, configErr := w.updateUpdater()
	snapshot := ui.UpdateSnapshot{
		Configured:  strings.TrimSpace(w.A.Preferences().UpdateFeedURL) != "",
		ConfigError: configErr,
	}
	if updater == nil {
		if configErr == "" {
			snapshot.Phase = string(host.FeedUpdateIdle)
		} else {
			snapshot.Phase = string(host.FeedUpdateError)
			snapshot.Err = configErr
		}
		return snapshot
	}
	status := updater.Status()
	snapshot.Configured = true
	snapshot.FeedURL = updater.FeedURL()
	snapshot.Phase = string(status.State)
	snapshot.CurrentVersion = status.CurrentVersion
	snapshot.AvailableVersion = status.AvailableVersion
	snapshot.Notes = status.Notes
	snapshot.Percent = status.Percent
	snapshot.Installable = status.Installable
	snapshot.InstallReason = status.InstallReason
	if status.LastError != nil {
		snapshot.Err = status.LastError.Error()
	}
	return snapshot
}

// CheckUpdate 在后台检查更新源（实现 ui.UpdateIntents）。
//
// 点击回调不阻塞：网络与失败原因都写回相位，界面下一帧读。
func (w *App) CheckUpdate(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	updater, configErr := w.updateUpdater()
	if configErr != "" {
		return errors.New(configErr)
	}
	if updater == nil {
		return host.ErrFeedNotConfigured
	}
	go func() {
		checkCtx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
		defer cancel()
		_, _ = updater.Check(checkCtx)
		w.changed()
	}()
	return nil
}

// DownloadUpdate 在后台下载并校验产物（实现 ui.UpdateIntents）。
func (w *App) DownloadUpdate(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	updater, configErr := w.updateUpdater()
	if configErr != "" {
		return errors.New(configErr)
	}
	if updater == nil {
		return host.ErrFeedNotConfigured
	}
	go func() {
		downloadCtx, cancel := context.WithTimeout(context.Background(), updateDownloadTimeout)
		defer cancel()
		_, _ = updater.Download(downloadCtx)
		w.changed()
	}()
	return nil
}

// InstallUpdate 走宿主安装能力（实现 ui.UpdateIntents）。没有能力时立刻返回
// ui.NoteUpdateInstall，不写假回执。
func (w *App) InstallUpdate(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	updater, configErr := w.updateUpdater()
	if configErr != "" {
		return errors.New(configErr)
	}
	if updater == nil {
		return host.ErrFeedNotConfigured
	}
	status := updater.Status()
	if !status.Installable {
		reason := status.InstallReason
		if strings.TrimSpace(reason) == "" {
			reason = ui.NoteUpdateInstall
		}
		return errors.New(reason)
	}
	go func() {
		_ = updater.Install(ctx)
		w.changed()
	}()
	return nil
}

// SetUpdateFeed 保存更新源地址（实现 ui.UpdateIntents）：空串表示关闭自动更新。
// 地址先规范化再落盘；规范化失败不写偏好，把原因交回界面。
func (w *App) SetUpdateFeed(ctx context.Context, feedURL string) error {
	canonical, err := host.NormalizeFeedURL(feedURL)
	if err != nil {
		return err
	}
	prefs := w.A.Preferences()
	if prefs.UpdateFeedURL == canonical {
		return nil
	}
	prefs.UpdateFeedURL = canonical
	if err := w.A.UpdatePreferences(prefs); err != nil {
		return err
	}
	w.resetUpdate()
	w.changed()
	return nil
}

// StartUpdateAutoCheck 在连接成功后安排一次延迟检查；没有配置更新源时什么也不做。
// 只跑一次：启动、重连都不重复检查。
func (w *App) StartUpdateAutoCheck() {
	if w == nil {
		return
	}
	w.updateAutoOnce.Do(func() {
		go func() {
			timer := time.NewTimer(updateAutoCheckDelay)
			defer timer.Stop()
			<-timer.C
			if strings.TrimSpace(w.A.Preferences().UpdateFeedURL) == "" {
				return
			}
			_ = w.CheckUpdate(context.Background())
		}()
	})
}

// updateUpdater 返回与当前偏好对应的更新编排。未配置时返回 (nil, "")；
// 配置不合法时返回 (nil, 原因)。同一个更新源复用同一个编排，界面每帧读不重建。
func (w *App) updateUpdater() (*host.FeedUpdater, string) {
	raw := strings.TrimSpace(w.A.Preferences().UpdateFeedURL)
	canonical, err := host.NormalizeFeedURL(raw)
	if err != nil {
		w.resetUpdate()
		return nil, err.Error()
	}
	if canonical == "" {
		w.resetUpdate()
		return nil, ""
	}
	w.updateMu.Lock()
	defer w.updateMu.Unlock()
	if w.update != nil && w.updateFeed == canonical && w.updateConfig == "" {
		return w.update, ""
	}
	if w.update != nil {
		previous := w.update
		w.update, w.updateFeed, w.updateConfig = nil, "", ""
		go func() { _ = previous.Close() }()
	}
	updater, err := host.NewFeedUpdater(host.FeedUpdateConfig{
		CurrentVersion: w.A.Version(),
		FeedURL:        canonical,
		Dir:            filepath.Join(w.A.DataDir(), "updates"),
		InstallReason:  ui.NoteUpdateInstall,
	})
	if err != nil {
		w.updateConfig = err.Error()
		return nil, err.Error()
	}
	w.update, w.updateFeed = updater, canonical
	return updater, ""
}

// resetUpdate 丢掉旧编排；关闭在后台做，避免界面帧等下载收尾。
func (w *App) resetUpdate() {
	w.updateMu.Lock()
	previous := w.update
	w.update, w.updateFeed, w.updateConfig = nil, "", ""
	w.updateMu.Unlock()
	if previous != nil {
		go func() { _ = previous.Close() }()
	}
}
