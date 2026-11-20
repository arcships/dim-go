package uiwire

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// 本文件是插件市场源的最小回归：地址门槛、两种清单形状与上游拉取。

func TestNormalizePluginFeed(t *testing.T) {
	if got, err := normalizePluginFeed("  "); err != nil || got != "" {
		t.Fatalf("空地址应视为未配置：%q %v", got, err)
	}
	if _, err := normalizePluginFeed("file:///tmp/plugins.json"); err == nil {
		t.Fatal("非 http/https 地址没有被拒")
	}
	if _, err := normalizePluginFeed("https://"); err == nil {
		t.Fatal("缺主机名的地址没有被拒")
	}
	got, err := normalizePluginFeed("https://feed.test/plugins.json")
	if err != nil || got != "https://feed.test/plugins.json" {
		t.Fatalf("合法地址被改坏：%q %v", got, err)
	}
}

func TestParsePluginFeedAcceptsObjectAndArray(t *testing.T) {
	object, err := parsePluginFeed([]byte(`{"plugins":[{"name":"alpha","version":"1.0.0","description":"说明"}]}`))
	if err != nil || len(object) != 1 || object[0].Name != "alpha" || object[0].Version != "1.0.0" {
		t.Fatalf("对象形状解析不对：%+v %v", object, err)
	}
	array, err := parsePluginFeed([]byte(`[{"id":"beta","version":"2.0.0"}]`))
	if err != nil || len(array) != 1 || array[0].Name != "beta" {
		t.Fatalf("数组形状解析不对：%+v %v", array, err)
	}
	if _, err := parsePluginFeed([]byte(`{`)); err == nil {
		t.Fatal("坏 JSON 没有被拒")
	}
}

// TestParsePluginFeedCarriesDetailFields：条目声明的作者、安装位置、Git 来源与
// 技能清单都进快照；skills 缺省与空数组分开保留（界面按两种文案画）。
func TestParsePluginFeedCarriesDetailFields(t *testing.T) {
	entries, err := parsePluginFeed([]byte(`{"plugins":[
		{"name":"alpha","version":"1.0.0","author":"Ada","location":"~/.dim/plugins/alpha",
		 "git":"owner/alpha","gitRef":"main",
		 "skills":[{"name":"one","description":"第一条"},{"name":"  "}]},
		{"name":"beta","skills":[]}
	]}`))
	if err != nil {
		t.Fatalf("解析：%v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("条目数不对：%+v", entries)
	}
	alpha := entries[0]
	if alpha.Author != "Ada" || alpha.Location != "~/.dim/plugins/alpha" ||
		alpha.Git != "owner/alpha" || alpha.GitRef != "main" {
		t.Fatalf("条目字段没有透传：%+v", alpha)
	}
	if len(alpha.Skills) != 1 || alpha.Skills[0].Name != "one" ||
		alpha.Skills[0].Description != "第一条" {
		t.Fatalf("技能清单不对（空名应当被丢掉）：%+v", alpha.Skills)
	}
	if entries[1].Skills == nil {
		t.Fatal("skills: [] 被当成了「源没给清单」")
	}
	if len(entries[1].Skills) != 0 {
		t.Fatalf("空技能数组解析出了条目：%+v", entries[1].Skills)
	}
	if entries[1].Git != "" {
		t.Fatalf("没给 Git 地址的条目却有来源：%+v", entries[1])
	}
}

func TestFetchPluginFeedReadsUpstream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plugins":[{"name":"remote","version":"3.0.0"}]}`))
	}))
	defer server.Close()

	entries, err := fetchPluginFeed(server.URL)
	if err != nil {
		t.Fatalf("拉取：%v", err)
	}
	if len(entries) != 1 || entries[0].Name != "remote" || entries[0].Version != "3.0.0" {
		t.Fatalf("上游条目不对：%+v", entries)
	}
}

func TestFetchPluginFeedReportsHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	if _, err := fetchPluginFeed(server.URL); err == nil {
		t.Fatal("非 2xx 没有被当作失败")
	}
}
