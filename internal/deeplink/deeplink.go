// Package deeplink 解析与生成客户端自己的深链（dimagent://…）。
//
// 两种目标：
//
//	dimagent://session/<会话标识>     打开（或复用）一个显示该会话的窗口
//	dimagent://settings/<分节>        打开设置屏的某一节；分节省略时进第一节
//
// 这里只做语法与目标判定：分节是否存在、会话是否还在目录里由持有那份事实的
// 一方判断（见 internal/ui 与入口的窗口管理）。解析失败一律给可读原因，原因
// 直接给用户看，不出现内部术语。
package deeplink

import (
	"fmt"
	"net/url"
	"strings"
)

// Scheme 是客户端深链的协议名。打包时它同时写进 Info.plist 的
// CFBundleURLTypes（见 scripts/release.sh），系统据此把 dimagent:// 交给本应用。
const Scheme = "dimagent"

// 链接目标的种类。
const (
	// KindSession 是 dimagent://session/<会话标识>。
	KindSession = "session"
	// KindSettings 是 dimagent://settings/<分节>。
	KindSettings = "settings"
)

// Target 是一条解析后的深链。Kind 是目标种类；会话链接用 SessionID，
// 设置链接用 Section（空表示第一节）。
type Target struct {
	Kind      string
	SessionID string
	Section   string
}

// String 是给人看的描述，用于日志。
func (t Target) String() string {
	switch t.Kind {
	case KindSession:
		return "会话 " + t.SessionID
	case KindSettings:
		if t.Section == "" {
			return "设置"
		}
		return "设置 › " + t.Section
	default:
		return "未知目标"
	}
}

// Parse 解析一条深链。空串、别的协议、未知目标与缺标识都返回可读原因。
//
// 同时接受 dimagent://session/<id> 与 dimagent:/session/<id>（少一个斜杠的形态
// 在手工拼链接时常见），两者的段位含义一致。
func Parse(raw string) (Target, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Target{}, fmt.Errorf("空链接：需要 %s://session/<会话> 或 %s://settings/<分节>", Scheme, Scheme)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Target{}, fmt.Errorf("无法解析链接 %q：%v", raw, err)
	}
	if !strings.EqualFold(u.Scheme, Scheme) {
		if u.Scheme == "" {
			return Target{}, fmt.Errorf("链接缺少 %s:// 前缀：%q", Scheme, raw)
		}
		return Target{}, fmt.Errorf("不支持的协议 %q：只接受 %s://", u.Scheme, Scheme)
	}

	kind := strings.ToLower(u.Host)
	rest := strings.Trim(u.Path, "/")
	// dimagent:/session/<id>：目标落在路径的第一段。
	if kind == "" {
		kind, rest = splitFirst(rest)
	}

	switch kind {
	case KindSession:
		if rest == "" {
			return Target{}, fmt.Errorf("会话链接缺少标识：%s://session/<会话>", Scheme)
		}
		// 会话标识可能带斜杠（百分比编码的标识也在这里还原）：不截断，整段当标识。
		return Target{Kind: KindSession, SessionID: rest}, nil
	case KindSettings:
		if rest == "" {
			return Target{Kind: KindSettings}, nil
		}
		if strings.Contains(rest, "/") {
			return Target{}, fmt.Errorf("设置链接只接受一节：%s://settings/<分节>", Scheme)
		}
		return Target{Kind: KindSettings, Section: strings.ToLower(rest)}, nil
	default:
		return Target{}, fmt.Errorf("未知的链接目标 %q：可用 %s://session/<会话> 或 %s://settings/<分节>",
			kind, Scheme, Scheme)
	}
}

// splitFirst 把路径切成第一段与其余部分。
func splitFirst(path string) (first, rest string) {
	if i := strings.Index(path, "/"); i >= 0 {
		return path[:i], path[i+1:]
	}
	return path, ""
}

// SessionURL 生成一个会话深链；标识按 URL 规则转义。
func SessionURL(sessionID string) string {
	return (&url.URL{Scheme: Scheme, Host: KindSession, Path: "/" + sessionID}).String()
}

// SettingsURL 生成一个设置深链；分节为空时是不带分节的形态。
func SettingsURL(section string) string {
	section = strings.TrimSpace(section)
	if section == "" {
		return Scheme + "://" + KindSettings
	}
	return (&url.URL{Scheme: Scheme, Host: KindSettings, Path: "/" + section}).String()
}
