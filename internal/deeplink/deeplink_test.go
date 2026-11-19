package deeplink

import (
	"strings"
	"testing"
)

// TestParseSession 核对会话链接：标识原样取出，前后多余的空格与斜杠都不算标识。
func TestParseSession(t *testing.T) {
	for _, raw := range []string{
		"dimagent://session/abc-123",
		"  dimagent://session/abc-123  ",
		"dimagent://session/abc-123/",
		"dimagent:/session/abc-123",
	} {
		target, err := Parse(raw)
		if err != nil {
			t.Fatalf("%q 应能解析：%v", raw, err)
		}
		if target.Kind != KindSession || target.SessionID != "abc-123" {
			t.Fatalf("%q 解析成 %+v，想要会话 abc-123", raw, target)
		}
	}
}

// TestParseSessionEncodedID 核对百分比编码：标识按 URL 规则还原后再用。
func TestParseSessionEncodedID(t *testing.T) {
	target, err := Parse("dimagent://session/" + "a%2Fb")
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if target.SessionID != "a/b" {
		t.Fatalf("标识 = %q，想要 %q", target.SessionID, "a/b")
	}
}

// TestParseSettings 核对设置链接：带分节与不带分节两种形态。
func TestParseSettings(t *testing.T) {
	target, err := Parse("dimagent://settings/Appearance")
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if target.Kind != KindSettings || target.Section != "appearance" {
		t.Fatalf("解析成 %+v，想要设置 › appearance", target)
	}

	target, err = Parse("dimagent://settings")
	if err != nil {
		t.Fatalf("不带分节的设置链接应能解析：%v", err)
	}
	if target.Kind != KindSettings || target.Section != "" {
		t.Fatalf("解析成 %+v，想要设置（第一节）", target)
	}
}

// TestParseUnknownTargetReadable 核对未知目标给的是可读原因：说清可用目标，
// 不是「解析失败」四个字。
func TestParseUnknownTargetReadable(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"dimagent://nope/x", "未知的链接目标"},
		{"https://example.com/session/x", "不支持的协议"},
		{"session/abc", "缺少 dimagent:// 前缀"},
		{"dimagent://session/", "会话链接缺少标识"},
		{"", "空链接"},
		{"dimagent://settings/a/b", "设置链接只接受一节"},
	}
	for _, c := range cases {
		_, err := Parse(c.raw)
		if err == nil {
			t.Fatalf("%q 应解析失败", c.raw)
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%q 的原因 = %q，想要包含 %q", c.raw, err.Error(), c.want)
		}
		if !strings.Contains(err.Error(), "dimagent://") && c.raw != "https://example.com/session/x" {
			t.Fatalf("%q 的原因应写清可用的链接形态：%q", c.raw, err.Error())
		}
	}
}

// TestURLRoundTrip 核对生成与解析互逆：复制出去的会话链接能被自己打开。
func TestURLRoundTrip(t *testing.T) {
	link := SessionURL("0b7c-session")
	if link != "dimagent://session/0b7c-session" {
		t.Fatalf("会话链接 = %q", link)
	}
	target, err := Parse(link)
	if err != nil || target.Kind != KindSession || target.SessionID != "0b7c-session" {
		t.Fatalf("会话链接不可解析：%v %+v", err, target)
	}

	if got := SettingsURL(""); got != "dimagent://settings" {
		t.Fatalf("不带分节的设置链接 = %q", got)
	}
	if got := SettingsURL("git"); got != "dimagent://settings/git" {
		t.Fatalf("设置链接 = %q", got)
	}
	if target, err := Parse(SettingsURL("git")); err != nil || target.Section != "git" {
		t.Fatalf("设置链接不可解析：%v %+v", err, target)
	}
}
