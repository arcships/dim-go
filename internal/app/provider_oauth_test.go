package app

import (
	"encoding/json"
	"testing"
)

// TestParseProviderAccount 核对 provider.management 的两种 account 形状：
// 托管帐号给 account 对象，原生 OAuth 给 authenticated 一组字段；没有 account
// 字段的静态 provider 不产生账号形态。
func TestParseProviderAccount(t *testing.T) {
	managed := parseProviderAccount(json.RawMessage(`{
		"namespace": "account",
		"loginMethods": ["browser", "device"],
		"state": {"account": {"issuer": "https://dim.test", "subject": "u-1",
			"email": "ada@example.test", "name": "Ada"}}
	}`))
	if managed == nil || !managed.Authenticated || managed.AccountLabel != "Ada" ||
		managed.AccountPreview != "u-1" || len(managed.LoginMethods) != 2 ||
		managed.Namespace != "account" {
		t.Fatalf("托管帐号解析 = %+v", managed)
	}

	native := parseProviderAccount(json.RawMessage(`{
		"namespace": "account.codex",
		"loginMethods": ["browser"],
		"state": {"authenticated": false, "recoveryRequired": true,
			"accountIdPreview": "acct_9"}
	}`))
	if native == nil || native.Authenticated || !native.RecoveryRequired ||
		native.AccountPreview != "acct_9" || native.State != "signed_out" {
		t.Fatalf("原生 OAuth 解析 = %+v", native)
	}

	if got := parseProviderAccount(nil); got != nil {
		t.Fatalf("没有 account 字段时不该造账号形态：%+v", got)
	}
	if got := parseProviderAccount(json.RawMessage(`{"loginMethods": ["browser"]}`)); got != nil {
		t.Fatalf("缺 namespace 时不该造账号形态：%+v", got)
	}
}

// TestProviderOAuthParams 断言托管帐号带 providerId，原生 OAuth 不带。
func TestProviderOAuthParams(t *testing.T) {
	if params := providerOAuthParams("account", "p1"); params["providerId"] != "p1" {
		t.Fatalf("托管帐号参数 = %v", params)
	}
	if params := providerOAuthParams("account.codex", "p1"); len(params) != 0 {
		t.Fatalf("原生 OAuth 不该带参数：%v", params)
	}
}

// TestProviderOAuthReason 断言服务端错误码换成可读原因，认不出的保留原文。
func TestProviderOAuthReason(t *testing.T) {
	cases := map[string]string{
		"":                     "",
		"oauth_failed":         "授权服务端拒绝了这次登录",
		"account_changed":      "账号已变化，请重新登录",
		"login_attempt_closed": "这次登录已经结束，请重新发起",
		"something_else":       "something_else",
	}
	for code, want := range cases {
		if got := providerOAuthReason(code); got != want {
			t.Fatalf("%q → %q，想要 %q", code, got, want)
		}
	}
}
