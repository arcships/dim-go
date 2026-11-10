package account

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestProvidersRequiresSource 确认没有 provider 读取能力时明确不可用。
func TestProvidersRequiresSource(t *testing.T) {
	account := newTestAccount(t, Config{})
	if _, err := account.Providers(context.Background()); !errors.Is(err, ErrProvidersUnavailable) {
		t.Fatalf("应返回缺少 provider 声明：%v", err)
	}
}

// TestProvidersRejectsMalformed 确认 provider 声明结构不合法时整份拒绝。
func TestProvidersRejectsMalformed(t *testing.T) {
	cases := []struct {
		name      string
		providers []Provider
		want      error
	}{
		{name: "空列表", providers: nil, want: ErrProvidersUnavailable},
		{name: "缺标识", providers: []Provider{{LoginMethods: []LoginMethod{LoginMethodBrowser}}}, want: ErrMalformedProviders},
		{name: "标识重复", providers: []Provider{{ID: "p"}, {ID: "p"}}, want: ErrMalformedProviders},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			source := &stubProviderSource{providers: testCase.providers}
			account := newTestAccount(t, Config{Providers: source})
			if _, err := account.Providers(context.Background()); !errors.Is(err, testCase.want) {
				t.Fatalf("错误分类不符：%v", err)
			}
		})
	}
}

// TestProvidersReadFailure 确认 provider 读取失败原样报错，不发起登录。
func TestProvidersReadFailure(t *testing.T) {
	source := &stubProviderSource{err: errors.New("连接中断")}
	account := newTestAccount(t, Config{Providers: source, Auth: callbackAuth("", "", ""), Browser: &stubBrowser{}})
	if _, err := account.Providers(context.Background()); !errors.Is(err, ErrProvidersRead) {
		t.Fatalf("应返回 provider 读取失败：%v", err)
	}
	if _, err := account.Login(context.Background(), LoginRequest{}); !errors.Is(err, ErrProvidersRead) {
		t.Fatalf("登录应因 provider 读取失败而停止：%v", err)
	}
}

// TestLoginRequiresProviderSelectionWhenMultiple 确认多个 provider 且未指定时拒绝，
// 不在候选之间猜。
func TestLoginRequiresProviderSelectionWhenMultiple(t *testing.T) {
	source := &stubProviderSource{providers: []Provider{
		{ID: "p-browser", LoginMethods: []LoginMethod{LoginMethodBrowser}},
		{ID: "p-device", LoginMethods: []LoginMethod{LoginMethodDevice}},
	}}
	browserAuth := callbackAuth("code-1", "state-1", "state-1")
	deviceAuth := &stubDeviceAuth{}
	account := newTestAccount(t, Config{
		Providers:  source,
		Auth:       browserAuth,
		DeviceAuth: deviceAuth,
		Browser:    &stubBrowser{},
	})
	if _, err := account.Login(context.Background(), LoginRequest{}); !errors.Is(err, ErrProviderSelectionRequired) {
		t.Fatalf("多个 provider 未指定时应拒绝：%v", err)
	}
	if browserAuth.beginCount() != 0 || len(deviceAuth.beginList()) != 0 {
		t.Fatal("未指定 provider 时不应发起任何登录")
	}
}

// TestLoginUnknownProvider 确认指定不存在的 provider 时明确报错。
func TestLoginUnknownProvider(t *testing.T) {
	account := newTestAccount(t, Config{Providers: browserProviderSource(), Auth: callbackAuth("", "", "")})
	if _, err := account.Login(context.Background(), LoginRequest{ProviderID: "missing"}); !errors.Is(err, ErrProviderUnknown) {
		t.Fatalf("应返回 provider 不存在：%v", err)
	}
}

// TestLoginExplicitMethodMustBeDeclared 确认显式指定的方式必须被 provider 声明。
func TestLoginExplicitMethodMustBeDeclared(t *testing.T) {
	deviceAuth := &stubDeviceAuth{}
	account := newTestAccount(t, Config{
		Providers:  browserProviderSource(),
		Auth:       callbackAuth("code-1", "state-1", "state-1"),
		DeviceAuth: deviceAuth,
		Browser:    &stubBrowser{},
	})
	_, err := account.Login(context.Background(), LoginRequest{Method: LoginMethodDevice})
	if !errors.Is(err, ErrLoginMethodUnavailable) {
		t.Fatalf("未声明的方式应拒绝：%v", err)
	}
	if !strings.Contains(err.Error(), "browser") {
		t.Fatalf("拒绝原因应列出已声明方式：%v", err)
	}
	if len(deviceAuth.beginList()) != 0 {
		t.Fatal("被拒绝的登录不应发起设备码请求")
	}
}

// TestLoginPrefersBrowserWhenBothDeclared 确认同时声明 browser 与 device 时默认走
// browser，并记录在登录尝试状态里。
func TestLoginPrefersBrowserWhenBothDeclared(t *testing.T) {
	status := &stubStatusReader{fact: signedInFact()}
	auth := callbackAuth("code-1", "state-1", "state-1")
	deviceAuth := &stubDeviceAuth{}
	browser := &httpBrowser{}
	account := newTestAccount(t, Config{
		Status:       status,
		Providers:    bothProviderSource(),
		Auth:         auth,
		DeviceAuth:   deviceAuth,
		Browser:      browser,
		LoginTimeout: 2 * time.Second,
	})

	if _, err := account.Login(context.Background(), LoginRequest{}); err != nil {
		t.Fatalf("登录: %v", err)
	}
	state := account.LoginState()
	if state.ProviderID != "provider-both" || state.Method != LoginMethodBrowser || state.Phase != LoginCompleted {
		t.Fatalf("默认应走 browser：%+v", state)
	}
	if len(deviceAuth.beginList()) != 0 {
		t.Fatal("默认 browser 时不应发起设备码请求")
	}
	if auth.beginCount() != 1 {
		t.Fatalf("浏览器授权发起次数不符：%d", auth.beginCount())
	}
}

// TestLoginExplicitDeviceWhenBothDeclared 确认同时声明两种方式时，显式指定 device
// 会走设备码流程。
func TestLoginExplicitDeviceWhenBothDeclared(t *testing.T) {
	status := &stubStatusReader{fact: signedInFact()}
	deviceAuth := &stubDeviceAuth{
		device: DeviceAuthorization{
			Attempt:         "device-attempt",
			VerificationURI: "https://issuer.example/device",
			UserCode:        "ABCD-EFGH",
			Interval:        time.Millisecond,
		},
		states: []DeviceLoginState{{Status: DeviceLoginCompleted}},
	}
	browserAuth := callbackAuth("code-1", "state-1", "state-1")
	account := newTestAccount(t, Config{
		Status:       status,
		Providers:    bothProviderSource(),
		Auth:         browserAuth,
		DeviceAuth:   deviceAuth,
		Browser:      &stubBrowser{},
		LoginTimeout: 2 * time.Second,
	})
	if _, err := account.Login(context.Background(), LoginRequest{Method: LoginMethodDevice}); err != nil {
		t.Fatalf("显式 device 登录: %v", err)
	}
	if browserAuth.beginCount() != 0 {
		t.Fatal("显式 device 时不应发起浏览器登录")
	}
	if state := account.LoginState(); state.Method != LoginMethodDevice || state.Phase != LoginCompleted {
		t.Fatalf("登录方式不符：%+v", state)
	}
}

// TestLoginRejectsProviderWithoutUsableMethod 确认 provider 只声明未知方式时拒绝，
// 不猜成 browser 或 device。
func TestLoginRejectsProviderWithoutUsableMethod(t *testing.T) {
	source := &stubProviderSource{providers: []Provider{{
		ID:           "provider-sso",
		LoginMethods: []LoginMethod{LoginMethod("sso")},
	}}}
	account := newTestAccount(t, Config{
		Providers:  source,
		Auth:       callbackAuth("code-1", "state-1", "state-1"),
		DeviceAuth: &stubDeviceAuth{},
		Browser:    &stubBrowser{},
	})
	_, err := account.Login(context.Background(), LoginRequest{})
	if !errors.Is(err, ErrLoginMethodUnavailable) {
		t.Fatalf("没有可用方式时应拒绝：%v", err)
	}
	if !strings.Contains(err.Error(), "sso") {
		t.Fatalf("拒绝原因应列出声明：%v", err)
	}
}

// TestLoginDeviceWhenOnlyDevice 确认 provider 只声明 device 时走设备码流程，返回
// 验证信息并轮询到完成。
func TestLoginDeviceWhenOnlyDevice(t *testing.T) {
	status := &stubStatusReader{fact: signedInFact()}
	deviceAuth := &stubDeviceAuth{
		device: DeviceAuthorization{
			Attempt:                 "device-attempt",
			VerificationURI:         "https://issuer.example/device",
			VerificationURIComplete: "https://issuer.example/device?code=ABCD",
			UserCode:                "ABCD-EFGH",
			Interval:                time.Millisecond,
		},
		states: []DeviceLoginState{
			{Status: DeviceLoginPending},
			{Status: DeviceLoginCompleted},
		},
	}
	account := newTestAccount(t, Config{
		Status:       status,
		Providers:    deviceProviderSource(),
		DeviceAuth:   deviceAuth,
		Browser:      &stubBrowser{},
		LoginTimeout: 2 * time.Second,
	})

	start, err := account.BeginLogin(context.Background(), LoginRequest{})
	if err != nil {
		t.Fatalf("发起设备码登录: %v", err)
	}
	if start.Method != LoginMethodDevice || start.ProviderID != "provider-device" {
		t.Fatalf("登录方式不符：%+v", start)
	}
	if start.Attempt != "device-attempt" || start.UserCode != "ABCD-EFGH" ||
		start.VerificationURI != "https://issuer.example/device" ||
		start.VerificationURIComplete != "https://issuer.example/device?code=ABCD" {
		t.Fatalf("设备码提示不符：%+v", start)
	}
	snapshot, err := account.WaitLogin(context.Background())
	if err != nil {
		t.Fatalf("等待设备码登录: %v", err)
	}
	if snapshot.Status != StatusSignedIn {
		t.Fatalf("设备码登录后的账号状态不符：%+v", snapshot)
	}
	if state := account.LoginState(); state.Phase != LoginCompleted || state.Method != LoginMethodDevice {
		t.Fatalf("设备码登录尝试状态不符：%+v", state)
	}
	if len(deviceAuth.beginList()) != 1 || deviceAuth.beginList()[0].ProviderID != "provider-device" {
		t.Fatalf("设备码发起参数不符：%+v", deviceAuth.beginList())
	}
	if deviceAuth.polls < 2 {
		t.Fatalf("应轮询到完成：%d", deviceAuth.polls)
	}
}

// TestDeviceLoginFailedKeepsReason 确认设备码失败保留服务端原因，并交回服务端释放。
func TestDeviceLoginFailedKeepsReason(t *testing.T) {
	deviceAuth := &stubDeviceAuth{
		device: DeviceAuthorization{Attempt: "device-attempt", VerificationURI: "https://issuer.example/device", UserCode: "ABCD"},
		states: []DeviceLoginState{{Status: DeviceLoginFailed, Reason: "用户拒绝授权"}},
	}
	account := newTestAccount(t, Config{
		Status:     &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:  deviceProviderSource(),
		DeviceAuth: deviceAuth,
		Browser:    &stubBrowser{},
	})
	_, err := account.Login(context.Background(), LoginRequest{})
	if !errors.Is(err, ErrDeviceLogin) || !strings.Contains(err.Error(), "用户拒绝授权") {
		t.Fatalf("应保留失败原因：%v", err)
	}
	state := account.LoginState()
	if state.Phase != LoginFailed || !strings.Contains(state.Reason, "用户拒绝授权") {
		t.Fatalf("登录尝试状态不符：%+v", state)
	}
	if cancels := deviceAuth.cancelList(); len(cancels) != 1 || cancels[0] != "device-attempt" {
		t.Fatalf("失败后应交回服务端释放：%v", cancels)
	}
}

// TestDeviceLoginExpired 确认服务端报告过期时按过期结束。
func TestDeviceLoginExpired(t *testing.T) {
	deviceAuth := &stubDeviceAuth{
		device: DeviceAuthorization{Attempt: "device-attempt", VerificationURI: "https://issuer.example/device", UserCode: "ABCD"},
		states: []DeviceLoginState{{Status: DeviceLoginExpired}},
	}
	account := newTestAccount(t, Config{
		Status:     &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:  deviceProviderSource(),
		DeviceAuth: deviceAuth,
		Browser:    &stubBrowser{},
	})
	if _, err := account.Login(context.Background(), LoginRequest{}); !errors.Is(err, ErrLoginExpired) {
		t.Fatalf("应返回设备码过期：%v", err)
	}
	if state := account.LoginState(); state.Phase != LoginTimedOut {
		t.Fatalf("过期后的登录尝试状态不符：%+v", state)
	}
	if cancels := deviceAuth.cancelList(); len(cancels) != 1 {
		t.Fatalf("过期后应交回服务端释放：%v", cancels)
	}
}

// TestDeviceLoginCancel 确认取消设备码等待会结束轮询并交回服务端释放。
func TestDeviceLoginCancel(t *testing.T) {
	deviceAuth := &stubDeviceAuth{
		device: DeviceAuthorization{
			Attempt:         "device-attempt",
			VerificationURI: "https://issuer.example/device",
			UserCode:        "ABCD",
			Interval:        time.Millisecond,
		},
		states: []DeviceLoginState{{Status: DeviceLoginPending}},
	}
	account := newTestAccount(t, Config{
		Status:       &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:    deviceProviderSource(),
		DeviceAuth:   deviceAuth,
		Browser:      &stubBrowser{},
		LoginTimeout: 5 * time.Second,
	})

	done := make(chan error, 1)
	go func() {
		_, err := account.Login(context.Background(), LoginRequest{})
		done <- err
	}()
	waitForPolls(t, deviceAuth, 1)
	account.CancelLogin()

	select {
	case err := <-done:
		if !errors.Is(err, ErrLoginCanceled) {
			t.Fatalf("应返回登录已取消：%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("取消后设备码登录没有结束")
	}
	if state := account.LoginState(); state.Phase != LoginCanceled {
		t.Fatalf("登录尝试状态不符：%+v", state)
	}
	if cancels := deviceAuth.cancelList(); len(cancels) != 1 || cancels[0] != "device-attempt" {
		t.Fatalf("取消后应交回服务端释放：%v", cancels)
	}
}

// TestDeviceBeginThenCancelReleases 确认只发起不等待时，取消也会交回服务端释放。
func TestDeviceBeginThenCancelReleases(t *testing.T) {
	deviceAuth := &stubDeviceAuth{
		device: DeviceAuthorization{Attempt: "device-attempt", VerificationURI: "https://issuer.example/device", UserCode: "ABCD"},
	}
	status := &stubStatusReader{fact: Fact{Status: StatusSignedOut}}
	account := newTestAccount(t, Config{
		Status:     status,
		Providers:  deviceProviderSource(),
		DeviceAuth: deviceAuth,
		Browser:    &stubBrowser{},
	})
	if _, err := account.Refresh(context.Background()); err != nil {
		t.Fatalf("读取账号状态: %v", err)
	}
	if _, err := account.BeginLogin(context.Background(), LoginRequest{}); err != nil {
		t.Fatalf("发起设备码登录: %v", err)
	}
	account.CancelLogin()
	if state := account.LoginState(); state.Phase != LoginCanceled {
		t.Fatalf("取消后的登录尝试状态不符：%+v", state)
	}
	if cancels := deviceAuth.cancelList(); len(cancels) != 1 {
		t.Fatalf("发起后取消应交回服务端释放：%v", cancels)
	}
	if account.Status() != StatusSignedOut {
		t.Fatalf("取消不应改变账号状态：%q", account.Status())
	}
}

// TestDeviceLoginMissingPromptIsRejected 确认服务端没有给验证地址或用户码时明确
// 失败，不展示半份提示。
func TestDeviceLoginMissingPromptIsRejected(t *testing.T) {
	deviceAuth := &stubDeviceAuth{
		device: DeviceAuthorization{Attempt: "device-attempt", VerificationURI: "https://issuer.example/device"},
	}
	account := newTestAccount(t, Config{
		Providers:  deviceProviderSource(),
		DeviceAuth: deviceAuth,
		Browser:    &stubBrowser{},
	})
	if _, err := account.BeginLogin(context.Background(), LoginRequest{}); !errors.Is(err, ErrLoginRequest) {
		t.Fatalf("缺少用户码应失败：%v", err)
	}

	missingAttempt := &stubDeviceAuth{
		device: DeviceAuthorization{VerificationURI: "https://issuer.example/device", UserCode: "ABCD"},
	}
	account = newTestAccount(t, Config{
		Providers:  deviceProviderSource(),
		DeviceAuth: missingAttempt,
		Browser:    &stubBrowser{},
	})
	if _, err := account.BeginLogin(context.Background(), LoginRequest{}); !errors.Is(err, ErrLoginRequest) {
		t.Fatalf("缺少尝试标识应失败：%v", err)
	}
	if polls := deviceAuth.polls; polls != 0 {
		t.Fatalf("缺少提示时不应开始轮询：%d", polls)
	}
	if state := account.LoginState(); state.Phase != LoginFailed {
		t.Fatalf("登录尝试状态不符：%+v", state)
	}
}

// TestDevicePollErrorFails 确认轮询错误明确失败并保留原因。
func TestDevicePollErrorFails(t *testing.T) {
	deviceAuth := &stubDeviceAuth{
		device:  DeviceAuthorization{Attempt: "device-attempt", VerificationURI: "https://issuer.example/device", UserCode: "ABCD"},
		pollErr: errors.New("连接中断"),
	}
	account := newTestAccount(t, Config{
		Status:     &stubStatusReader{fact: Fact{Status: StatusSignedOut}},
		Providers:  deviceProviderSource(),
		DeviceAuth: deviceAuth,
		Browser:    &stubBrowser{},
	})
	_, err := account.Login(context.Background(), LoginRequest{})
	if !errors.Is(err, ErrDeviceLogin) || !strings.Contains(err.Error(), "连接中断") {
		t.Fatalf("轮询错误应明确失败：%v", err)
	}
	if state := account.LoginState(); state.Phase != LoginFailed {
		t.Fatalf("登录尝试状态不符：%+v", state)
	}
}

// TestWaitLoginWithoutAttempt 确认没有发起尝试时等待会明确报错。
func TestWaitLoginWithoutAttempt(t *testing.T) {
	account := newTestAccount(t, Config{})
	if _, err := account.WaitLogin(context.Background()); !errors.Is(err, ErrNoLoginAttempt) {
		t.Fatalf("应返回没有登录尝试：%v", err)
	}
}

// waitForPolls 等待设备码轮询次数达到 want。
func waitForPolls(t *testing.T, deviceAuth *stubDeviceAuth, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		deviceAuth.mu.Lock()
		polls := deviceAuth.polls
		deviceAuth.mu.Unlock()
		if polls >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("等待设备码轮询次数 %d 超时", want)
}
