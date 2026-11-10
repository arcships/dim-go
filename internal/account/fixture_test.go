package account

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// 真实账号夹具验证需要两个环境变量：
//
//	DIMAGENT_ACCOUNT_FIXTURE   已复制到临时目录的产品 home（含 auth.json）
//	DIMAGENT_APP_SERVER_BIN    真实 App Server 可执行文件
//
// 来源产品 home 只作为只读复制源，测试不读取也不写回；本包代码里没有任何来源
// 路径。未设置变量时整组测试跳过，普通 go test 不受影响。
const (
	fixtureProviderID = "dimcode-api-oauth"
	fixtureDirectory  = "https://dimagent.cn"
	fixtureEnvHome    = "DIMAGENT_ACCOUNT_FIXTURE"
	fixtureEnvBin     = "DIMAGENT_APP_SERVER_BIN"
)

func fixturePaths(t *testing.T) (string, string) {
	t.Helper()
	home := os.Getenv(fixtureEnvHome)
	bin := os.Getenv(fixtureEnvBin)
	if home == "" || bin == "" {
		t.Skipf("未设置 %s / %s，跳过真实账号夹具验证", fixtureEnvHome, fixtureEnvBin)
	}
	if _, err := os.Stat(filepath.Join(home, "auth.json")); err != nil {
		t.Fatalf("夹具缺少 auth.json（不读取内容）: %v", err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("App Server 可执行文件不可用: %v", err)
	}
	return home, bin
}

// openFixture 在夹具 home 上拉起真实 App Server 控制连接。测试结束后只回收自己
// 拉起的实例。
func openFixture(t *testing.T, home, bin string) *daemon.Connection {
	t.Helper()
	supervisor, err := daemon.New(daemon.Config{
		DataDir:    home,
		ServerPath: bin,
		// 统一二进制 dimcode-server 需要 app-server 子命令；独立 App Server
		// 二进制可以留空。
		ServerArgs:     []string{"app-server"},
		RequestTimeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建连接监督者: %v", err)
	}
	connection, err := supervisor.Connect(context.Background())
	if err != nil {
		supervisor.Close(nil)
		t.Fatalf("连接真实 App Server: %v", err)
	}
	t.Cleanup(func() {
		if result, shutdownErr := supervisor.Shutdown(context.Background()); shutdownErr == nil {
			t.Logf("真实服务回收：stopped=%v reason=%s", result.Stopped, result.Reason)
		}
		supervisor.Close(nil)
	})
	return connection
}

// fixtureReader 用真实服务端的 account 命名空间实现本领域的三个读取接口。
// 它只映射字段，不解释业务含义；响应里的令牌从不经过这里。
type fixtureReader struct {
	client     *protocol.Client
	providerID string
}

func (r *fixtureReader) ReadAccount(ctx context.Context) (Fact, error) {
	var reply struct {
		Account *struct {
			Issuer  string `json:"issuer"`
			Subject string `json:"subject"`
			Email   string `json:"email"`
			Name    string `json:"name"`
		} `json:"account"`
	}
	if err := r.client.CallInto(ctx, "account.status", map[string]any{"providerId": r.providerID}, &reply); err != nil {
		return Fact{}, err
	}
	if reply.Account == nil {
		return Fact{Status: StatusSignedOut}, nil
	}
	return Fact{
		Status: StatusSignedIn,
		Identity: Identity{
			AccountID:   reply.Account.Subject,
			Email:       reply.Account.Email,
			DisplayName: reply.Account.Name,
		},
	}, nil
}

func (r *fixtureReader) ReadUsage(ctx context.Context) (Usage, error) {
	var reply struct {
		Credits *struct {
			UsedUnits  *float64 `json:"usedUnits"`
			TotalUnits *float64 `json:"totalUnits"`
			Unlimited  *bool    `json:"unlimited"`
			ResetsAt   *string  `json:"resetsAt"`
		} `json:"credits"`
		Windows []struct {
			DurationHours float64 `json:"durationHours"`
			TokenUsed     float64 `json:"tokenUsed"`
			TokenCap      float64 `json:"tokenCap"`
			ExpiresAt     *string `json:"expiresAt"`
		} `json:"windows"`
		Features []struct {
			Key       string   `json:"key"`
			Used      *float64 `json:"used"`
			Allowance *float64 `json:"allowance"`
			Unit      *string  `json:"unit"`
			PeriodEnd *string  `json:"periodEnd"`
		} `json:"features"`
	}
	if err := r.client.CallInto(ctx, "account.usage", map[string]any{"providerId": r.providerID}, &reply); err != nil {
		return Usage{}, err
	}

	usage := Usage{}
	for _, feature := range reply.Features {
		line := UsageLine{Key: feature.Key, Label: feature.Key}
		if feature.Used != nil {
			line.Used = *feature.Used
		}
		if feature.Allowance != nil {
			line.Limit = *feature.Allowance
		}
		if feature.Unit != nil {
			line.Unit = *feature.Unit
		}
		line.ResetsAt = fixtureTime(feature.PeriodEnd)
		usage.Lines = append(usage.Lines, line)
	}
	for _, window := range reply.Windows {
		line := UsageLine{
			Key:      fmt.Sprintf("window-%gh", window.DurationHours),
			Label:    fmt.Sprintf("最近 %g 小时", window.DurationHours),
			Used:     window.TokenUsed,
			Limit:    window.TokenCap,
			Unit:     "tokens",
			Window:   fmt.Sprintf("%gh", window.DurationHours),
			ResetsAt: fixtureTime(window.ExpiresAt),
		}
		usage.Lines = append(usage.Lines, line)
	}
	if reply.Credits != nil && reply.Credits.Unlimited != nil && *reply.Credits.Unlimited {
		usage.Aggregate = AggregateUsage{Reason: "服务端声明无限额度，没有上限值"}
	} else if reply.Credits != nil && reply.Credits.UsedUnits != nil && reply.Credits.TotalUnits != nil {
		usage.Aggregate = AggregateUsage{Available: true, Used: *reply.Credits.UsedUnits, Limit: *reply.Credits.TotalUnits, Unit: "units"}
	}
	// 服务端没有给聚合项时保持不可用，交给本包补原因。
	return usage, nil
}

func (r *fixtureReader) ReadModels(ctx context.Context) ([]Model, error) {
	var reply struct {
		Catalog *struct {
			Models []struct {
				ID                string  `json:"id"`
				Name              string  `json:"name"`
				UnavailableReason *string `json:"unavailableReason"`
			} `json:"models"`
		} `json:"catalog"`
	}
	if err := r.client.CallInto(ctx, "provider.catalog", map[string]any{"providerId": r.providerID}, &reply); err != nil {
		return nil, err
	}
	if reply.Catalog == nil {
		return nil, errors.New("服务端没有该账号的模型目录")
	}
	models := make([]Model, 0, len(reply.Catalog.Models))
	for _, entry := range reply.Catalog.Models {
		models = append(models, Model{
			ID:              entry.ID,
			DisplayName:     entry.Name,
			RequiresAccount: true,
			Unavailable:     entry.UnavailableReason != nil,
		})
	}
	return models, nil
}

// fixtureAuth 是真实浏览器登录的服务端适配：发起时保存服务端尝试标识，完成与
// 取消都按尝试标识交回。它不打开浏览器，也不记录任何令牌。
type fixtureAuth struct {
	client     *protocol.Client
	providerID string
	attempt    string
	expiresAt  int64
	cancels    int
}

func (a *fixtureAuth) BeginAuthorization(ctx context.Context, redirectURI string) (AuthorizationRequest, error) {
	var reply struct {
		AttemptID        string `json:"attemptId"`
		AuthorizationURL string `json:"authorizationUrl"`
		ExpiresAt        int64  `json:"expiresAt"`
	}
	err := a.client.CallInto(ctx, "account.login.browser.begin", map[string]any{
		"providerId":  a.providerID,
		"directory":   fixtureDirectory,
		"redirectUri": redirectURI,
	}, &reply)
	if err != nil {
		return AuthorizationRequest{}, err
	}
	a.attempt = reply.AttemptID
	a.expiresAt = reply.ExpiresAt
	return AuthorizationRequest{
		URL:     reply.AuthorizationURL,
		Attempt: reply.AttemptID,
	}, nil
}

func (a *fixtureAuth) CompleteAuthorization(ctx context.Context, attempt string, result AuthorizationResult) error {
	return errors.New("夹具验证不完成真实登录")
}

func (a *fixtureAuth) CancelAuthorization(ctx context.Context, attempt string) error {
	a.cancels++
	_, err := a.client.Call(ctx, "account.login.cancel", map[string]any{
		"providerId": a.providerID,
		"attemptId":  attempt,
	})
	return err
}

func fixtureTime(raw *string) time.Time {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// TestRealAccountFixtureReads 用真实 App Server 与临时夹具覆盖三条读取：账号状态、
// 逐项用量、模型清单。证据只输出条数与有无，不输出授权码或令牌。
func TestRealAccountFixtureReads(t *testing.T) {
	home, bin := fixturePaths(t)
	connection := openFixture(t, home, bin)
	client := connection.Client
	reader := &fixtureReader{client: client, providerID: fixtureProviderID}

	features := make(map[string]bool, len(connection.Hello.Features))
	for _, feature := range connection.Hello.Features {
		features[feature] = true
	}
	t.Logf("服务端版本=%s 能力：account.status=%v account.usage=%v provider.catalog=%v",
		connection.Hello.ServerVersion, features["account.status"], features["account.usage"], features["provider.catalog"])

	var listed []struct {
		ProviderID       string   `json:"providerId"`
		DisplayName      string   `json:"displayName"`
		LoginMethods     []string `json:"loginMethods"`
		DefaultDirectory *string  `json:"defaultDirectory"`
	}
	if err := client.CallInto(context.Background(), "account.providers", nil, &listed); err != nil {
		t.Fatalf("读取账号提供者: %v", err)
	}
	var found bool
	for _, provider := range listed {
		if provider.ProviderID == fixtureProviderID {
			found = true
			t.Logf("账号提供者：displayName=%q loginMethods=%v directoryConfigured=%v",
				provider.DisplayName, provider.LoginMethods, provider.DefaultDirectory != nil)
		}
	}
	if !found {
		t.Fatalf("真实服务端没有 %s 账号提供者", fixtureProviderID)
	}

	account := newTestAccount(t, Config{Status: reader, Usage: reader, Models: reader})

	snapshot, err := account.Refresh(context.Background())
	if err != nil {
		t.Fatalf("真实账号状态读取失败: %v", err)
	}
	if snapshot.Status != StatusSignedIn {
		t.Fatalf("夹具账号应为已登录：%q", snapshot.Status)
	}
	if snapshot.Identity.AccountID == "" {
		t.Fatal("已登录但没有账号标识")
	}
	t.Logf("账号状态：status=%s identity=%v usageReady=%v", snapshot.Status,
		identityShape(snapshot.Identity), snapshot.Usage.Ready)

	usage, err := account.RefreshUsage(context.Background())
	if err != nil {
		t.Fatalf("真实用量读取失败: %v", err)
	}
	if !usage.Ready {
		t.Fatal("真实用量读取后应为可用")
	}
	t.Logf("用量：lines=%d aggregateAvailable=%v aggregateReason=%q observedAtSet=%v",
		len(usage.Lines), usage.Aggregate.Available, usage.Aggregate.Reason, !usage.ObservedAt.IsZero())
	for index, line := range usage.Lines {
		if index >= 3 {
			break
		}
		t.Logf("用量项[%d]：key=%q unit=%q window=%q limitSet=%v resetSet=%v",
			index, line.Key, line.Unit, line.Window, line.Limit > 0, !line.ResetsAt.IsZero())
	}

	models, err := account.RefreshModels(context.Background())
	if err != nil {
		t.Fatalf("真实模型清单读取失败: %v", err)
	}
	if !models.Ready || len(models.Models) == 0 {
		t.Fatalf("已登录应有可用模型：%+v", models)
	}
	names := make([]string, 0, 3)
	for _, model := range models.Models {
		if len(names) >= 3 {
			break
		}
		names = append(names, model.ID)
	}
	t.Logf("模型清单：available=%d first=%v", len(models.Models), names)
}

// TestRealAccountFixtureLoginBeginReleases 用真实服务端验证登录发起、授权地址与
// 环回回调地址，并在超时后按尝试标识取消。完成交换需要真人在浏览器登录，见文档
// 的 blocked 记录。
func TestRealAccountFixtureLoginBeginReleases(t *testing.T) {
	home, bin := fixturePaths(t)
	connection := openFixture(t, home, bin)
	client := connection.Client
	reader := &fixtureReader{client: client, providerID: fixtureProviderID}
	auth := &fixtureAuth{client: client, providerID: fixtureProviderID}
	browser := &stubBrowser{}
	account := newTestAccount(t, Config{
		Status:       reader,
		Auth:         auth,
		Browser:      browser,
		LoginTimeout: 2 * time.Second,
	})

	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("真实账号状态读取失败: %v", err)
	}
	if _, err := account.Login(context.Background()); !errors.Is(err, ErrLoginTimeout) {
		t.Fatalf("登录应以等待回调超时结束：%v", err)
	}
	if auth.attempt == "" {
		t.Fatal("服务端没有给出登录尝试标识")
	}
	if cancels := auth.cancels; cancels != 1 {
		t.Fatalf("超时后应取消一次服务端尝试：%d", cancels)
	}
	opened := browser.openedURLs()
	if len(opened) != 1 {
		t.Fatalf("应打开一次授权地址：%d", len(opened))
	}
	authorizationURL, err := url.Parse(opened[0])
	if err != nil {
		t.Fatalf("授权地址无法解析: %v", err)
	}
	if authorizationURL.Hostname() != "dimagent.cn" {
		t.Fatalf("授权地址不是账号目录：%q", authorizationURL.Hostname())
	}
	redirectURI := authorizationURL.Query().Get("redirect_uri")
	parsedRedirect, err := url.Parse(redirectURI)
	if err != nil || parsedRedirect.Hostname() != "127.0.0.1" || parsedRedirect.Path != callbackPath {
		t.Fatalf("授权地址没有带环回回调：%q", redirectURI)
	}
	if account.Status() != StatusSignedIn {
		t.Fatalf("登录超时不应改变已确认的账号状态：%q", account.Status())
	}
	t.Logf("登录发起：authorizationHost=%s redirectLoopback=%v attemptIssued=%v expiresSet=%v cancelled=%d",
		authorizationURL.Hostname(), true, true, auth.expiresAt > 0, auth.cancels)
}

// identityShape 只输出身份字段的有无，不输出邮箱等个人信息。
func identityShape(identity Identity) string {
	return fmt.Sprintf("accountIDSet=%v emailSet=%v nameSet=%v planSet=%v",
		identity.AccountID != "", identity.Email != "", identity.DisplayName != "", identity.Plan != "")
}
