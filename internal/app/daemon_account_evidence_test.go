package app

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/account"
)

// TestRealAccountLoginBeginEvidence 是账号按 provider 分发的真实发起证据。
//
// 只有设置 DIMAGENT_ACCOUNT_EVIDENCE=1 时运行；夹具 home 从 /tmp 只读复制到临时
// 目录。只验证发起、轮询与取消，不完成真实登录；证据只留状态、标识与主机名。
func TestRealAccountLoginBeginEvidence(t *testing.T) {
	if os.Getenv("DIMAGENT_ACCOUNT_EVIDENCE") != "1" {
		t.Skip("未设置 DIMAGENT_ACCOUNT_EVIDENCE=1，跳过真实账号发起证据")
	}
	fixtureHome := os.Getenv("DIMAGENT_ACCOUNT_FIXTURE")
	if fixtureHome == "" {
		fixtureHome = "/private/tmp/dimagent-real-model/home"
	}
	serverPath := os.Getenv("DIMAGENT_APP_SERVER_BIN")
	if serverPath == "" {
		serverPath = filepath.Join(os.Getenv("HOME"), ".dimcode-alpha", "rust", "dimcode-server")
	}
	if _, err := os.Stat(filepath.Join(fixtureHome, "auth.json")); err != nil {
		t.Fatalf("夹具缺少 auth.json（不读取内容）: %v", err)
	}
	dataDir, err := os.MkdirTemp("/tmp", "d12a-")
	if err != nil {
		t.Fatalf("创建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	if err := copyFixtureTree(fixtureHome, dataDir); err != nil {
		t.Fatalf("复制夹具 home: %v", err)
	}
	workspaceRoot := filepath.Join(dataDir, "w")
	if err := os.MkdirAll(workspaceRoot, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}

	base, err := New(Config{
		DataDir:       dataDir,
		ServerPath:    serverPath,
		ClientName:    "dimagent-mygo-account-evidence",
		WorkspaceRoot: workspaceRoot,
		Version:       testVersion,
		SendTimeout:   30 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	if err := base.Connect(ctx); err != nil {
		t.Fatalf("连接服务端: %v", err)
	}
	defer base.CloseConnection(ctx)

	providers, err := (&accountProviderSource{wire: base.accountWire}).ReadProviders(ctx)
	if err != nil {
		t.Fatalf("读取账号提供者: %v", err)
	}
	if len(providers) == 0 {
		t.Fatalf("服务端没有账号 provider")
	}
	provider := providers[0]
	methods := make([]string, 0, len(provider.LoginMethods))
	for _, method := range provider.LoginMethods {
		methods = append(methods, string(method))
	}
	t.Logf("account.providers: count=%d methods=%v directorySet=%t", len(providers), methods, provider.Directory != "")

	auth := &accountAuthSource{wire: base.accountWire}
	request, err := auth.BeginProviderAuthorization(ctx, account.ProviderAuthorizationBegin{
		ProviderID:  provider.ID,
		Directory:   provider.Directory,
		RedirectURI: "http://127.0.0.1:45871/auth/callback",
	})
	if err != nil {
		t.Fatalf("发起浏览器登录: %v", err)
	}
	t.Logf("browser.begin: attemptSet=%t urlHost=%s", request.Attempt != "", hostOf(request.URL))
	if err := auth.CancelAuthorization(ctx, request.Attempt); err != nil {
		t.Logf("browser.cancel: 取消失败（按 blocked 记录）: %v", err)
	} else {
		t.Log("browser.cancel: ok")
	}

	device := base.deviceAuth
	deviceAuth, err := device.BeginDeviceAuthorization(ctx, provider.ID, provider.Directory)
	if err != nil {
		t.Fatalf("发起设备码登录: %v", err)
	}
	t.Logf("device.begin: attemptSet=%t userCodeSet=%t verificationHost=%s expiresSet=%t interval=%s",
		deviceAuth.Attempt != "", deviceAuth.UserCode != "", hostOf(deviceAuth.VerificationURI),
		!deviceAuth.ExpiresAt.IsZero(), deviceAuth.Interval)
	state, err := device.PollDeviceAuthorization(ctx, deviceAuth.Attempt)
	if err != nil {
		t.Logf("device.status: 轮询失败（按 blocked 记录）: %v", err)
	} else {
		t.Logf("device.status: status=%s reasonSet=%t", state.Status, state.Reason != "")
	}
	if err := auth.CancelAuthorization(ctx, deviceAuth.Attempt); err != nil {
		t.Logf("device.cancel: 取消失败（按 blocked 记录）: %v", err)
	} else {
		t.Log("device.cancel: ok")
	}
}

// hostOf 只返回 URL 的主机名，丢弃路径与查询串。
func hostOf(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "unparsable"
	}
	return parsed.Host
}

// copyFixtureTree 只复制夹具的浅层文件（auth.json/config.json）与 rust 目录；
// 不读来源、不打印内容。
func copyFixtureTree(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		from := filepath.Join(src, entry.Name())
		to := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(to, 0o700); err != nil {
				return err
			}
			if err := copyFixtureTree(from, to); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(from)
		if err != nil {
			return err
		}
		if err := os.WriteFile(to, data, 0o600); err != nil {
			return err
		}
	}
	return nil
}
