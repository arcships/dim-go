package host

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// latest.json 是「关于 › 软件更新」读取的版本源形状：
//
//	{
//	  "version": "1.2.3",
//	  "notes": "这一版改了什么",
//	  "url": "https://…/DimAgent-1.2.3.zip",
//	  "sha256": "<64 位十六进制>",
//	  "minVersion": "1.0.0"
//	}
//
// 它是宿主本端的更新源，不是 App Server 的协议：地址由用户在设置 › 关于里
// 填写，缺省关闭（不假装已是最新）。
const (
	// FeedFileName 是更新源地址的缺省文件名。
	FeedFileName = "latest.json"
	// maxFeedBytes 是 latest.json 的读取上限。
	maxFeedBytes int64 = 1 << 20
)

// Feed 是 latest.json 的内容。
//
// URL 与 SHA256 是下载与校验的依据；Notes 只用于展示；MinVersion 表示低于该
// 版本需要重新安装，不能走增量更新。
type Feed struct {
	Version    string
	Notes      string
	URL        string
	SHA256     string
	MinVersion string
}

// NormalizeFeedURL 把设置里填的地址规范成 latest.json 的完整地址：
// 空串表示未配置（返回空串、无错误）；基址自动补 `/latest.json`。
func NormalizeFeedURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("更新源必须是 http 或 https 地址")
	}
	if !strings.HasSuffix(strings.ToLower(parsed.Path), ".json") {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + FeedFileName
	}
	return parsed.String(), nil
}

// ParseFeed 解析并校验 latest.json。缺版本号、缺产物地址或摘要不合法一律失败：
// 不完整的事实不进下载。
func ParseFeed(feedURL string, data []byte) (Feed, error) {
	base, err := url.Parse(feedURL)
	if err != nil || !base.IsAbs() ||
		(base.Scheme != "http" && base.Scheme != "https") {
		return Feed{}, fmt.Errorf("更新源地址不可解析: %q", feedURL)
	}
	var raw struct {
		Version    string `json:"version"`
		Notes      string `json:"notes"`
		URL        string `json:"url"`
		SHA256     string `json:"sha256"`
		MinVersion string `json:"minVersion"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Feed{}, fmt.Errorf("更新源不是合法的 JSON: %w", err)
	}
	feed := Feed{
		Version:    strings.TrimSpace(raw.Version),
		Notes:      strings.TrimSpace(raw.Notes),
		URL:        strings.TrimSpace(raw.URL),
		SHA256:     normalizeSHA256(raw.SHA256),
		MinVersion: strings.TrimSpace(raw.MinVersion),
	}
	if feed.Version == "" {
		return Feed{}, fmt.Errorf("更新源缺少 version")
	}
	if _, err := parseSemver(feed.Version); err != nil {
		return Feed{}, fmt.Errorf("更新源的 version 不合法: %w", err)
	}
	if feed.MinVersion != "" {
		if _, err := parseSemver(feed.MinVersion); err != nil {
			return Feed{}, fmt.Errorf("更新源的 minVersion 不合法: %w", err)
		}
		if compared, err := compareVersions(feed.MinVersion, feed.Version); err == nil && compared > 0 {
			return Feed{}, fmt.Errorf("更新源的 minVersion %s 高于 version %s", feed.MinVersion, feed.Version)
		}
	}
	if feed.URL == "" {
		return Feed{}, fmt.Errorf("更新源缺少产物地址")
	}
	resolved, err := resolveArtifactURL(base, feed.URL)
	if err != nil {
		return Feed{}, err
	}
	feed.URL = resolved
	if feed.SHA256 == "" {
		return Feed{}, fmt.Errorf("更新源缺少 sha256 摘要")
	}
	if _, err := decodeSHA256(feed.SHA256); err != nil {
		return Feed{}, fmt.Errorf("更新源的 sha256 摘要不合法: %w", err)
	}
	return feed, nil
}

// normalizeSHA256 去掉摘要里常见的前缀与空白，统一成小写十六进制。
func normalizeSHA256(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "sha256:")
	return strings.ToLower(strings.TrimSpace(trimmed))
}

// decodeSHA256 把十六进制摘要解成 32 字节。
func decodeSHA256(value string) ([]byte, error) {
	trimmed := normalizeSHA256(value)
	if len(trimmed) != 64 {
		return nil, fmt.Errorf("摘要不是 64 位十六进制")
	}
	decoded := make([]byte, 32)
	for i := 0; i < 32; i++ {
		high, ok := hexNibble(trimmed[i*2])
		if !ok {
			return nil, fmt.Errorf("摘要不是十六进制")
		}
		low, ok := hexNibble(trimmed[i*2+1])
		if !ok {
			return nil, fmt.Errorf("摘要不是十六进制")
		}
		decoded[i] = high<<4 | low
	}
	return decoded, nil
}

func hexNibble(r byte) (byte, bool) {
	switch {
	case r >= '0' && r <= '9':
		return r - '0', true
	case r >= 'a' && r <= 'f':
		return r - 'a' + 10, true
	}
	return 0, false
}

// CompareVersions 按 semver 比较两个版本：a 小于 b 返回 -1，相等 0，大于 1。
// 预发布版低于同号的正式版，预发布段按 semver 的标识规则比较。
func CompareVersions(a, b string) (int, error) {
	return compareVersions(a, b)
}
