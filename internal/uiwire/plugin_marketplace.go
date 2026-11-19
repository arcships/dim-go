package uiwire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/arcships/dimagent-mygo/internal/ui"
)

// 本文件把插件市场的上游清单接给设置 › 插件设置：事实走
// ui.PluginMarketplaceFacts，动作走 ui.PluginMarketplaceIntents。
//
// 插件市场不是 App Server 的事实：源地址是本地偏好（preferences.json 的
// pluginFeedURL），清单由本端按该地址拉取。未配置源地址时界面整块禁用并写
// 「未配置插件源」，不画样例条目；配置了才拉取。
//
// 清单只用来展示：协议没有市场安装来源，所以条目的「安装」入口置灰并写原因。
//
// 源文件形状（本端定义，最小集合）：
//
//	{"plugins": [{"name": "...", "version": "1.2.0",
//	              "description": "...", "source": "owner/repo"}]}
//
// 顶层也可以直接是数组。未知字段忽略；没有 name（或 id）的条目被丢掉。

const (
	// pluginFeedTimeout 是一次上游拉取的超时。
	pluginFeedTimeout = 20 * time.Second
	// pluginFeedMaxBytes 是上游响应的读取上限。
	pluginFeedMaxBytes = 4 << 20
)

// pluginFeedFile 是上游源的顶层形状。
type pluginFeedFile struct {
	Plugins []pluginFeedEntry `json:"plugins"`
}

// pluginFeedEntry 是上游源的一个条目。
type pluginFeedEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

// PluginMarketplace 返回插件市场的只读快照（实现 ui.PluginMarketplaceFacts）。
func (w *App) PluginMarketplace() ui.PluginMarketplaceSnapshot {
	raw := strings.TrimSpace(w.A.Preferences().PluginFeedURL)
	canonical, err := normalizePluginFeed(raw)
	snapshot := ui.PluginMarketplaceSnapshot{Configured: raw != ""}
	if err != nil {
		snapshot.ConfigError = err.Error()
		return snapshot
	}
	snapshot.FeedURL = canonical
	w.marketMu.Lock()
	defer w.marketMu.Unlock()
	if canonical == "" || w.marketFeed != canonical {
		return snapshot
	}
	snapshot.Loading = w.marketLoading
	snapshot.Entries = append([]ui.PluginMarketEntrySnapshot(nil), w.marketEntries...)
	snapshot.Err = w.marketErr
	return snapshot
}

// RefreshPluginMarketplace 在后台拉取一次上游清单（实现
// ui.PluginMarketplaceIntents）。点击回调不等网络。
func (w *App) RefreshPluginMarketplace(ctx context.Context) error {
	canonical, err := normalizePluginFeed(w.A.Preferences().PluginFeedURL)
	if err != nil {
		return err
	}
	if canonical == "" {
		return errors.New("未配置插件源")
	}
	w.marketMu.Lock()
	if w.marketLoading && w.marketFeed == canonical {
		w.marketMu.Unlock()
		return nil
	}
	w.marketFeed = canonical
	w.marketLoading = true
	w.marketErr = ""
	w.marketMu.Unlock()
	w.changed()
	go func() {
		entries, fetchErr := fetchPluginFeed(canonical)
		w.marketMu.Lock()
		w.marketLoading = false
		if fetchErr != nil {
			w.marketErr = fetchErr.Error()
		} else {
			w.marketEntries = entries
			w.marketErr = ""
		}
		w.marketMu.Unlock()
		w.changed()
	}()
	return nil
}

// SetPluginFeed 保存插件源地址（空串表示未配置）。地址先规范化再落盘；不合法
// 时不写偏好，把原因交回界面。
func (w *App) SetPluginFeed(ctx context.Context, feedURL string) error {
	canonical, err := normalizePluginFeed(feedURL)
	if err != nil {
		return err
	}
	prefs := w.A.Preferences()
	if prefs.PluginFeedURL == canonical {
		return nil
	}
	prefs.PluginFeedURL = canonical
	if err := w.A.UpdatePreferences(prefs); err != nil {
		return err
	}
	w.marketMu.Lock()
	w.marketFeed, w.marketLoading, w.marketEntries, w.marketErr = canonical, false, nil, ""
	w.marketMu.Unlock()
	w.changed()
	if canonical != "" {
		_ = w.RefreshPluginMarketplace(context.Background())
	}
	return nil
}

// normalizePluginFeed 规范化插件源地址：空串表示未配置，其余必须是 http/https。
func normalizePluginFeed(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("插件源地址不可解析：%v", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("插件源地址必须是 http 或 https")
	}
	if parsed.Host == "" {
		return "", errors.New("插件源地址缺少主机名")
	}
	return parsed.String(), nil
}

// fetchPluginFeed 拉取并解析上游清单。它只在后台协程里跑。
func fetchPluginFeed(feedURL string) ([]ui.PluginMarketEntrySnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pluginFeedTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("插件源返回 HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, pluginFeedMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > pluginFeedMaxBytes {
		return nil, errors.New("插件源响应过大")
	}
	return parsePluginFeed(body)
}

// parsePluginFeed 解析上游清单：接受 {"plugins":[...]} 或顶层数组。
func parsePluginFeed(body []byte) ([]ui.PluginMarketEntrySnapshot, error) {
	var entries []pluginFeedEntry
	if trimmed := strings.TrimSpace(string(body)); strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal(body, &entries); err != nil {
			return nil, fmt.Errorf("插件源清单无法解析：%v", err)
		}
	} else {
		var file pluginFeedFile
		if err := json.Unmarshal(body, &file); err != nil {
			return nil, fmt.Errorf("插件源清单无法解析：%v", err)
		}
		entries = file.Plugins
	}
	out := make([]ui.PluginMarketEntrySnapshot, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			name = strings.TrimSpace(entry.ID)
		}
		if name == "" {
			continue
		}
		out = append(out, ui.PluginMarketEntrySnapshot{
			Name:        name,
			Version:     entry.Version,
			Description: entry.Description,
			Source:      entry.Source,
		})
	}
	return out, nil
}
