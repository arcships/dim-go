package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 模型供应」详情里按提供商登录的四种形态：
//
//	OpenAI Codex    浏览器登录：授权地址 → 回调地址 → 完成交换（S-configuration-29/30/31/32/33/34/36）
//	Grok Build      设备码登录：验证地址 + 用户码 → 轮询 → 完成（S-configuration-37/38/39/40）
//	托管帐号         浏览器 / 设备码二选一（S-configuration-41/42/43）
//	Dim 与账户托管   只读提示条 + 模型块（既有分支，不在这里）
//
// 形态由服务端 provider.management 的 account.namespace 决定，登录方式由
// account.loginMethods 决定；本屏不按名字猜。
//
// 状态：未登录 / 等待浏览器回调 / 等待设备码 / 已连接 / 出错。发起、轮询、完成、
// 取消与登出都走 Intents（account.codex.* / account.grok.* / account.*）；等待中
// 的尝试只保存在这个文件的可丢状态里，不落盘，也不缓存令牌。
//
// 文案逐字取参考产品的 zh-CN（i18n 的 configurationSettings.*）。

// 账号命名空间：与服务端的 account.namespace 原文一致。
const (
	configNamespaceAccount = "account"
	configNamespaceCodex   = "account.codex"
	configNamespaceGrok    = "account.grok"
)

// 登录方式：与服务端声明的 account.loginMethods 原文一致。
const (
	configLoginBrowser = "browser"
	configLoginDevice  = "device"
)

// 本屏文案：逐字取自参考产品 zh-CN。
const (
	configCopyCodexSignedOut      = "登录 OpenAI 后可在 DimAgent 中使用 Codex 模型。"
	configCopyCodexSignIn         = "使用 OpenAI 登录"
	configCopyCodexSignedIn       = "令牌已生效。模型开关决定聊天中可用的 Codex 模型。"
	configCopyCodexWaiting        = "请将浏览器地址栏中的 callback URL 复制到 DimAgent 完成登录。"
	configCopyGrokSignedOut       = "登录 SuperGrok 或 Premium+ 后，即可在 DimAgent 中使用 Grok Build 模型。"
	configCopyGrokSignIn          = "使用 SuperGrok / Premium+ 登录"
	configCopyGrokSignedIn        = "OAuth 令牌已生效。模型开关决定聊天中可用的 Grok Build 模型。"
	configCopyGrokWaiting         = "正在等待授权..."
	configCopyGrokNoModels        = "登录后加载 Grok Build 模型。"
	configCopyGrokDeviceCode      = "设备码"
	configCopyGrokOpenVerify      = "打开验证页面"
	configCopyManagedSignedIn     = "已登录。模型开关决定聊天中显示哪些模型。"
	configCopyManagedSignIn       = "登录 %s 以加载其模型。"
	configCopyManagedDeviceLogin  = "使用设备代码登录"
	configCopyManagedBrowserLogin = "使用浏览器登录"
	configCopyManagedRedirect     = "回调地址"
	configCopyAutoModels          = "登录成功后将自动拉取可用模型。"
	configCopyOpenURL             = "打开 URL"
	configCopyCopy                = "复制"
	configCopyCopied              = "已复制"
	configCopyCallbackURL         = "Callback URL"
	configCopyCallbackHint        = "粘贴完整 callback URL 或 query string"
	configCopyConfirm             = "确认"
	configCopyCancel              = "取消"
	configCopyRefresh             = "刷新"
	configCopySignOut             = "退出登录"
	configCopyConnected           = "已连接"
	configCopyCallbackRequired    = "Callback URL 不能为空"
	configCopyRedirectRequired    = "回调地址不能为空"
	configCopyChecking            = "正在查询登录状态…"
)

// providerOAuthFlow 是一次按提供商登录的可丢状态。它只在内存里，切换屏、
// 取消或完成后清空；不保存令牌、授权码或回调原文。
type providerOAuthFlow struct {
	// 归属：只对发起它的 provider 生效；切到别的 provider 时不再显示等待态。
	providerKey string
	providerID  string
	namespace   string
	method      string
	attempt     string

	authorizationURL        string
	verificationURI         string
	verificationURIComplete string
	userCode                string
	// redirectURI 是托管帐号浏览器登录的回调地址输入（参考的默认值）。
	redirectURI string
	// callbackURL 是用户粘贴的回调地址。
	callbackURL string
	// pending 表示一次发起 / 完成请求还在路上；此时入口禁用。
	pending bool
	// interval / nextPoll / polling 是设备码轮询的节奏：下一次查询时刻由渲染线程
	// 检查，不在定时器协程里读写界面状态。
	interval time.Duration
	nextPoll time.Time
	polling  bool
	// err 是这次尝试的失败原因；空表示没有错误。
	err string
	// note 是一句结果回执（登录成功 / 已退出登录）。
	note string
}

var configOAuthFlow = providerOAuthFlow{
	redirectURI: "http://127.0.0.1:53682/oauth/callback",
}

// configOAuthReset 清掉登录尝试的界面状态（切换 provider、完成与测试用）。
func configOAuthReset() {
	configOAuthFlow = providerOAuthFlow{
		redirectURI: "http://127.0.0.1:53682/oauth/callback",
	}
}

// configOAuthDeclares 判断 provider 是否声明了某种登录方式。
func configOAuthDeclares(account *ProviderAccountSnapshot, method string) bool {
	if account == nil {
		return false
	}
	for _, declared := range account.LoginMethods {
		if declared == method {
			return true
		}
	}
	return false
}

// configOAuthBody 是账号形态 provider 的详情：状态区 + 基础 URL + 模型块 + 底部
// 动作（S-configuration-29 … 34、36 … 43）。
func (a *App) configOAuthBody(c *gui.Context, data configView) {
	gui.Column(c).Grow(1).MinHeight(0).Gap(configDetailGap).Children(func() {
		gui.Column(c).Shrink(0).Gap(configDetailGap).Children(func() {
			a.configOAuthStatus(c, data)
			if data.selected.baseURL != "" {
				configBaseURL(c, data.selected.baseURL)
			}
		})
		a.configModelsPanel(c, data)
		a.configDetailFooter(c, data)
	})
}

// configOAuthStatus 按形态与状态画一块：等待中的尝试优先于读数，其次已连接，
// 再次未登录。
func (a *App) configOAuthStatus(c *gui.Context, data configView) {
	provider := data.selected
	account := provider.providerAccount
	if account == nil {
		return
	}
	a.configOAuthMaybePoll(provider)
	flow := configOAuthFlow
	mine := flow.providerKey == configKey(provider) && flow.attempt != ""
	switch {
	case mine && flow.method == configLoginBrowser:
		a.configOAuthWaitingBrowser(c, provider)
	case mine && flow.method == configLoginDevice:
		a.configOAuthWaitingDevice(c, provider)
	case account.Authenticated:
		a.configOAuthAuthenticated(c, provider, account)
	default:
		a.configOAuthSignedOut(c, provider, account)
	}
	if flow.providerKey == configKey(provider) {
		a.configOAuthMessages(c, provider, account)
	}
}

// configOAuthSignedOut 是未登录态：说明 + 声明的登录入口（S-29/30、37、41/42）。
func (a *App) configOAuthSignedOut(c *gui.Context, provider configProvider, account *ProviderAccountSnapshot) {
	t := c.Theme()
	description := ""
	switch account.Namespace {
	case configNamespaceCodex:
		description = configCopyCodexSignedOut
	case configNamespaceGrok:
		description = configCopyGrokSignedOut
	default:
		description = fmt.Sprintf(configCopyManagedSignIn, provider.name)
	}
	gui.Column(c).Shrink(0).Gap(configGapRow).Children(func() {
		if description != "" {
			// S-configuration-29 / 37 / 41 未登录说明
			configRowStyle.style(t, gui.Text(c, description)).TextColor(t.TextMuted).
				Selectable().MaxLines(2)
		}
		gui.Row(c).Wrap().Gap(configGapRow).Children(func() {
			// S-configuration-30 / 38 / 42 登录入口（按服务端声明的方式摆）
			if configOAuthDeclares(account, configLoginBrowser) {
				label := configCopyCodexSignIn
				if account.Namespace == configNamespaceAccount {
					label = configCopyManagedBrowserLogin
				}
				if configButton(c, label, label, configOAuthFlow.pending, configCopyChecking) {
					a.configOAuthBegin(provider, configLoginBrowser)
				}
			}
			if configOAuthDeclares(account, configLoginDevice) {
				label := configCopyManagedDeviceLogin
				if account.Namespace == configNamespaceGrok {
					label = configCopyGrokSignIn
				}
				if configButton(c, label, label, configOAuthFlow.pending, configCopyChecking) {
					a.configOAuthBegin(provider, configLoginDevice)
				}
			}
		})
		if account.Namespace == configNamespaceAccount &&
			configOAuthDeclares(account, configLoginBrowser) {
			// S-configuration-41 回调地址：浏览器登录必须先有它。
			gui.Column(c).Shrink(0).Gap(configGapIcon).Children(func() {
				configMicroStyle.style(t, gui.Text(c, configCopyManagedRedirect)).
					TextColor(t.TextMuted).Label(configCopyManagedRedirect).SingleLine()
				gui.TextInput(c, &configOAuthFlow.redirectURI).
					Label(configCopyManagedRedirect).Height(configFormInputH).
					Padding(configFormPadV, configFormPadH).Background(fieldFill(t))
			})
		}
	})
}

// configOAuthAuthenticated 是已连接卡（S-33/34、37、42）：账号标识 + 刷新 + 退出
// 登录 + 一句说明；有套餐或账户预览时逐行写。
func (a *App) configOAuthAuthenticated(c *gui.Context, provider configProvider, account *ProviderAccountSnapshot) {
	t := c.Theme()
	ink := colorAccentInk.in(t)
	label := account.AccountLabel
	if label == "" {
		label = configCopyConnected
	}
	description := configCopyManagedSignedIn
	switch account.Namespace {
	case configNamespaceCodex:
		description = configCopyCodexSignedIn
	case configNamespaceGrok:
		description = configCopyGrokSignedIn
	}
	box := gui.Column(c).Shrink(0).Padding(configBannerPadV, configBannerPadH).
		Radius(screenRadiusCtl).Background(t.Accent.Alpha(configBannerFill)).
		Label("已连接的账号")
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Accent.Alpha(configBannerStroke), screenRadiusCtl, screenStroke)
	})
	box.Children(func() {
		gui.Column(c).Gap(configGapIcon).Children(func() {
			gui.Row(c).Shrink(0).AlignItems(gui.Start).Justify(gui.SpaceBetween).
				Gap(configGapRow).Children(func() {
				gui.Column(c).Grow(1).MinWidth(0).Gap(configGapItems).Children(func() {
					// S-configuration-33 / 42 账号标识
					configStrongStyle.style(t, gui.Text(c, label)).TextColor(ink).
						SingleLine().Tooltip(label)
					if account.AccountPreview != "" && account.AccountPreview != label {
						// S-configuration-33 账户预览
						configMicroStyle.style(t, gui.Text(c, "账户："+account.AccountPreview)).
							TextColor(t.TextMuted).SingleLine()
					}
					if account.RecoveryRequired {
						configMicroStyle.style(t, gui.Text(c, "登录已失效，需要重新授权")).
							TextColor(t.Danger).Role(gui.RoleAlert).MaxLines(2)
					}
				})
				gui.Row(c).Shrink(0).Gap(configGapRow).Children(func() {
					// S-configuration-33 刷新
					if configButton(c, configCopyRefresh, "刷新账号", configOAuthFlow.pending,
						configCopyChecking) {
						a.configOAuthRefresh()
					}
					// S-configuration-33 退出登录
					if configButton(c, configCopySignOut, "退出登录", configOAuthFlow.pending,
						configCopyChecking) {
						a.configOAuthLogout(provider)
					}
				})
			})
			// S-configuration-34 / 42 已登录说明
			configMicroStyle.style(t, gui.Text(c, description)).TextColor(t.TextMuted).
				Selectable().MaxLines(2)
		})
	})
}

// configOAuthWaitingBrowser 是等待浏览器回调（S-31/32）：旋转指示 + 取消、
// 授权地址的打开 / 复制、回调地址输入与确认。
func (a *App) configOAuthWaitingBrowser(c *gui.Context, provider configProvider) {
	t := c.Theme()
	box := gui.Column(c).Shrink(0).Padding(configBannerPadV, configBannerPadH).
		Radius(screenRadiusCtl).Background(t.Background).Label("等待浏览器授权")
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	box.Children(func() {
		gui.Column(c).Gap(configGapRow).Children(func() {
			gui.Row(c).Shrink(0).AlignItems(gui.Start).Justify(gui.SpaceBetween).
				Gap(configGapRow).Children(func() {
				gui.Row(c).Grow(1).MinWidth(0).AlignItems(gui.Start).
					Gap(configGapIcon).Children(func() {
					gui.Spinner(c).Size(configIconBig, configIconBig).Label("等待授权")
					configMicroStyle.style(t, gui.Text(c, configCopyCodexWaiting)).
						TextColor(t.TextMuted).Selectable().MaxLines(3)
				})
				// S-configuration-32 取消
				if configButton(c, configCopyCancel, "取消登录", configOAuthFlow.pending, "") {
					a.configOAuthCancel(provider)
				}
			})
			gui.Row(c).Wrap().Gap(configGapRow).Children(func() {
				url := configOAuthFlow.authorizationURL
				// S-configuration-31 打开 URL
				if configButton(c, configCopyOpenURL, "打开授权地址", url == "", "没有授权地址") {
					a.configOAuthOpenURL(url)
				}
				// S-configuration-31 复制
				if configButton(c, configCopyCopy, "复制授权地址", url == "", "没有授权地址") {
					a.configOAuthCopy(c, url)
				}
			})
			// S-configuration-31 回调地址 · 输入 · 确认
			gui.Column(c).Shrink(0).Gap(configGapIcon).Children(func() {
				configMicroStyle.style(t, gui.Text(c, configCopyCallbackURL)).
					TextColor(t.TextMuted).Label(configCopyCallbackURL).SingleLine()
				gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(configGapRow).Children(func() {
					gui.TextInput(c, &configOAuthFlow.callbackURL).
						Label(configCopyCallbackURL).Placeholder(configCopyCallbackHint).
						Height(configFormInputH).Padding(configFormPadV, configFormPadH).
						Grow(1).MinWidth(0).Background(fieldFill(t))
					if configButton(c, configCopyConfirm, "确认登录",
						configOAuthFlow.pending || strings.TrimSpace(configOAuthFlow.callbackURL) == "",
						configCopyCallbackHint) {
						a.configOAuthCompleteBrowser(provider)
					}
				})
				if flow := configOAuthFlow; flow.redirectURI != "" &&
					provider.providerAccount != nil && provider.providerAccount.Namespace == configNamespaceAccount {
					configMicroStyle.style(t, gui.Text(c, flow.redirectURI)).
						TextColor(t.TextMuted).SingleLine().Selectable()
				}
			})
		})
	})
}

// configOAuthWaitingDevice 是等待设备码授权（S-32/38）：旋转指示 + 取消 +
// 设备码 + 打开验证页面 / 复制 + 验证地址。
func (a *App) configOAuthWaitingDevice(c *gui.Context, provider configProvider) {
	t := c.Theme()
	box := gui.Column(c).Shrink(0).Padding(configBannerPadV, configBannerPadH).
		Radius(screenRadiusCtl).Background(t.Background).Label("等待设备码授权")
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	box.Children(func() {
		gui.Column(c).Gap(configGapRow).Children(func() {
			gui.Row(c).Shrink(0).AlignItems(gui.Start).Justify(gui.SpaceBetween).
				Gap(configGapRow).Children(func() {
				gui.Row(c).Grow(1).MinWidth(0).AlignItems(gui.Start).
					Gap(configGapIcon).Children(func() {
					gui.Spinner(c).Size(configIconBig, configIconBig).Label("等待授权")
					configMicroStyle.style(t, gui.Text(c, configCopyGrokWaiting)).
						TextColor(t.TextMuted).MaxLines(2)
				})
				if configButton(c, configCopyCancel, "取消登录", configOAuthFlow.pending, "") {
					a.configOAuthCancel(provider)
				}
			})
			// S-configuration-38 设备码 · userCode
			gui.Column(c).Shrink(0).Gap(configGapIcon).Children(func() {
				configMicroStyle.style(t, gui.Text(c, configCopyGrokDeviceCode)).
					TextColor(t.TextMuted).Label(configCopyGrokDeviceCode).SingleLine()
				gui.Row(c).Shrink(0).Height(configCodeH).AlignItems(gui.Center).
					Padding(configCodePadV, configCodePadH).Radius(screenRadiusCtl).
					Background(t.SurfaceHover).Label("设备码取值").Children(func() {
					configCodeStyle.style(t, gui.Text(c, configOAuthFlow.userCode)).
						TextColor(t.Text).Selectable().SingleLine()
				})
			})
			gui.Row(c).Wrap().Gap(configGapRow).Children(func() {
				url := configOAuthFlow.verificationURI
				if configOAuthFlow.verificationURIComplete != "" {
					url = configOAuthFlow.verificationURIComplete
				}
				// S-configuration-38 打开验证页面
				if configButton(c, configCopyGrokOpenVerify, "打开验证页面", url == "",
					"没有验证地址") {
					a.configOAuthOpenURL(url)
				}
				// S-configuration-38 复制
				if configButton(c, configCopyCopy, "复制验证地址", url == "", "没有验证地址") {
					a.configOAuthCopy(c, url)
				}
			})
			if uri := configOAuthFlow.verificationURI; uri != "" {
				configMicroStyle.style(t, gui.Text(c, uri)).TextColor(t.TextMuted).
					SingleLine().Selectable()
			}
		})
	})
}

// configOAuthMessages 是状态区下方的一行回执与一行错误（S-36/39/43）。
func (a *App) configOAuthMessages(c *gui.Context, provider configProvider, account *ProviderAccountSnapshot) {
	t := c.Theme()
	flow := configOAuthFlow
	if flow.providerKey != configKey(provider) {
		// 这次尝试不属于当前 provider：只剩服务端状态里的一条「需要重新授权」。
		if account.RecoveryRequired {
			configMicroStyle.style(t, gui.Text(c, "登录已失效，需要重新授权")).
				TextColor(t.Danger).Role(gui.RoleAlert).MaxLines(2)
		}
		return
	}
	switch {
	case flow.err != "":
		// S-configuration-36 / 39 / 43 可见错误
		configMicroStyle.style(t, gui.Text(c, flow.err)).TextColor(t.Danger).
			Role(gui.RoleAlert).MaxLines(3)
	case account.RecoveryRequired:
		// 已保存的凭据无法续期：把重新登录当作唯一出路，原话不摊内部错误码。
		configMicroStyle.style(t, gui.Text(c, "登录已失效，需要重新授权")).
			TextColor(t.Danger).Role(gui.RoleAlert).MaxLines(2)
	case flow.note != "":
		configMicroStyle.style(t, gui.Text(c, flow.note)).TextColor(t.TextMuted).
			Role(gui.RoleStatus).MaxLines(1)
	}
}

// configOAuthBegin 发起一次登录：浏览器方式记下授权地址，设备码方式开始轮询。
func (a *App) configOAuthBegin(provider configProvider, method string) {
	account := provider.providerAccount
	if account == nil || a.intents == nil || configOAuthFlow.pending {
		return
	}
	key := configKey(provider)
	namespace := account.Namespace
	providerID := provider.id
	redirectURI := ""
	if namespace == configNamespaceAccount && method == configLoginBrowser {
		redirectURI = strings.TrimSpace(configOAuthFlow.redirectURI)
		if redirectURI == "" {
			// 托管帐号的浏览器登录必须先有回调地址：空着就不发请求，就地写原因。
			configOAuthFlow.err = configCopyRedirectRequired
			return
		}
	}
	configOAuthFlow = providerOAuthFlow{
		providerKey: key, providerID: providerID, namespace: namespace,
		method: method, pending: true,
		redirectURI: configOAuthFlow.redirectURI,
	}
	var start ProviderOAuthStartSnapshot
	a.dispatchQuiet("发起登录", func(ctx context.Context) error {
		begin, err := a.intents.BeginProviderOAuth(ctx, providerID, namespace, method, redirectURI)
		start = begin
		return err
	}, func() {
		if configOAuthFlow.providerKey != key || configOAuthFlow.method != method {
			return
		}
		configOAuthFlow.pending = false
		configOAuthFlow.attempt = start.Attempt
		configOAuthFlow.authorizationURL = start.AuthorizationURL
		configOAuthFlow.verificationURI = start.VerificationURI
		configOAuthFlow.verificationURIComplete = start.VerificationURIComplete
		configOAuthFlow.userCode = start.UserCode
		if start.RedirectURI != "" {
			configOAuthFlow.redirectURI = start.RedirectURI
		}
		if method == configLoginDevice {
			interval := time.Duration(start.Interval) * time.Second
			if interval <= 0 {
				interval = 2 * time.Second
			}
			configOAuthFlow.interval = interval
			configOAuthFlow.nextPoll = time.Now().Add(interval)
		} else if start.AuthorizationURL != "" {
			// 授权页在发起后直接打开；失败也不影响下面的复制与手动打开。
			_ = a.intents.OpenProviderOAuthURL(context.Background(), start.AuthorizationURL)
		}
	}, func(reason string) {
		if configOAuthFlow.providerKey != key {
			return
		}
		configOAuthFlow.pending = false
		configOAuthFlow.err = strings.TrimPrefix(reason, "发起登录：")
	})
}

// configOAuthMaybePoll 在渲染线程上判断是否该查一次尝试状态：间隔不到、已有
// 查询在路上或换了 provider 时都不动。它不建定时器，下一次查询由下一帧触发。
func (a *App) configOAuthMaybePoll(provider configProvider) {
	flow := configOAuthFlow
	if flow.providerKey != configKey(provider) || flow.attempt == "" || flow.polling {
		return
	}
	if flow.nextPoll.IsZero() || time.Now().Before(flow.nextPoll) {
		return
	}
	a.configOAuthPoll(flow.providerKey, flow.providerID, flow.namespace, flow.attempt,
		flow.interval)
}

// configOAuthPoll 查一次尝试状态；pending 时把下一次查询时刻推后，结果由
// configOAuthPollResult 在渲染线程上结算。尝试被取消或换 provider 后结果丢弃。
func (a *App) configOAuthPoll(providerKey, providerID, namespace, attempt string, interval time.Duration) {
	if a.intents == nil || attempt == "" {
		return
	}
	if configOAuthFlow.attempt != attempt || configOAuthFlow.providerKey != providerKey {
		return
	}
	configOAuthFlow.polling = true
	var progress ProviderOAuthProgressSnapshot
	a.dispatchQuiet("查询登录状态", func(ctx context.Context) error {
		next, err := a.intents.PollProviderOAuth(ctx, providerID, namespace, attempt)
		progress = next
		return err
	}, func() {
		a.configOAuthPollResult(providerKey, providerID, namespace, attempt, progress, interval)
	}, func(reason string) {
		if configOAuthFlow.attempt != attempt || configOAuthFlow.providerKey != providerKey {
			return
		}
		configOAuthFlow.attempt = ""
		configOAuthFlow.pending = false
		configOAuthFlow.polling = false
		configOAuthFlow.err = strings.TrimPrefix(reason, "查询登录状态：")
	})
}

// configOAuthPollResult 处理一次轮询回执：完成 / 失败 / 取消 / 过期结束等待，
// pending 继续按同一间隔查。
func (a *App) configOAuthPollResult(
	providerKey, providerID, namespace, attempt string,
	progress ProviderOAuthProgressSnapshot, interval time.Duration,
) {
	if configOAuthFlow.attempt != attempt || configOAuthFlow.providerKey != providerKey {
		return
	}
	switch progress.Stage {
	case "pending":
		configOAuthFlow.polling = false
		next := interval
		if next <= 0 {
			next = 2 * time.Second
		}
		configOAuthFlow.nextPoll = time.Now().Add(next)
		return
	case "completed":
		configOAuthFlow = providerOAuthFlow{redirectURI: configOAuthFlow.redirectURI,
			providerKey: providerKey, note: "登录成功"}
		return
	case "failed":
		reason := progress.Reason
		if reason == "" {
			reason = "授权服务端拒绝了这次登录"
		}
		configOAuthFlow.attempt = ""
		configOAuthFlow.pending = false
		configOAuthFlow.err = reason
	case "cancelled":
		configOAuthFlow.attempt = ""
		configOAuthFlow.pending = false
		configOAuthFlow.note = "登录已取消"
	case "expired":
		configOAuthFlow.attempt = ""
		configOAuthFlow.pending = false
		configOAuthFlow.err = "登录已过期，请重新发起"
	}
}

// configOAuthCompleteBrowser 用粘贴的回调地址完成浏览器登录。
func (a *App) configOAuthCompleteBrowser(provider configProvider) {
	if a.intents == nil || configOAuthFlow.pending {
		return
	}
	callback := strings.TrimSpace(configOAuthFlow.callbackURL)
	if callback == "" {
		configOAuthFlow.err = configCopyCallbackRequired
		return
	}
	key := configKey(provider)
	attempt := configOAuthFlow.attempt
	namespace := configOAuthFlow.namespace
	providerID := configOAuthFlow.providerID
	configOAuthFlow.pending = true
	configOAuthFlow.err = ""
	a.dispatchQuiet("完成登录", func(ctx context.Context) error {
		return a.intents.CompleteProviderOAuth(ctx, providerID, namespace, attempt, callback)
	}, func() {
		if configOAuthFlow.providerKey != key {
			return
		}
		configOAuthFlow = providerOAuthFlow{
			redirectURI: configOAuthFlow.redirectURI, providerKey: key, note: "登录成功",
		}
	}, func(reason string) {
		if configOAuthFlow.providerKey != key {
			return
		}
		configOAuthFlow.pending = false
		configOAuthFlow.err = strings.TrimPrefix(reason, "完成登录：")
	})
}

// configOAuthCancel 取消进行中的尝试：本地等待立刻结束，服务端尝试按标识释放。
func (a *App) configOAuthCancel(provider configProvider) {
	if a.intents == nil {
		configOAuthReset()
		return
	}
	attempt := configOAuthFlow.attempt
	namespace := configOAuthFlow.namespace
	providerID := configOAuthFlow.providerID
	configOAuthFlow.attempt = ""
	configOAuthFlow.pending = true
	a.dispatchQuiet("取消登录", func(ctx context.Context) error {
		return a.intents.CancelProviderOAuth(ctx, providerID, namespace, attempt)
	}, func() {
		configOAuthFlow.pending = false
		configOAuthFlow.note = "登录已取消"
	}, func(reason string) {
		configOAuthFlow.pending = false
		configOAuthFlow.err = strings.TrimPrefix(reason, "取消登录：")
	})
	_ = provider
}

// configOAuthRefresh 重读 provider 清单（登录状态与模型都随读数回来）。
func (a *App) configOAuthRefresh() {
	if a.intents == nil || configOAuthFlow.pending {
		return
	}
	configOAuthFlow.pending = true
	configOAuthFlow.providerKey = configKey(a.configView().selected)
	configOAuthFlow.err = ""
	a.dispatchQuiet("刷新账号", func(ctx context.Context) error {
		return a.intents.RefreshProviders(ctx)
	}, func() {
		configOAuthFlow.pending = false
		configOAuthFlow.note = "已刷新"
	}, func(reason string) {
		configOAuthFlow.pending = false
		configOAuthFlow.err = strings.TrimPrefix(reason, "刷新账号：")
	})
}

// configOAuthLogout 登出并重读读数。
func (a *App) configOAuthLogout(provider configProvider) {
	if a.intents == nil || provider.providerAccount == nil || configOAuthFlow.pending {
		return
	}
	key := configKey(provider)
	namespace := provider.providerAccount.Namespace
	providerID := provider.id
	configOAuthFlow.pending = true
	configOAuthFlow.providerKey = key
	configOAuthFlow.err = ""
	a.dispatchQuiet("退出登录", func(ctx context.Context) error {
		return a.intents.LogoutProviderOAuth(ctx, providerID, namespace)
	}, func() {
		configOAuthFlow = providerOAuthFlow{
			redirectURI: configOAuthFlow.redirectURI, providerKey: key, note: "已退出登录",
		}
	}, func(reason string) {
		if configOAuthFlow.providerKey != key {
			return
		}
		configOAuthFlow.pending = false
		configOAuthFlow.err = strings.TrimPrefix(reason, "退出登录：")
	})
}

// configOAuthOpenURL 用系统浏览器打开授权 / 验证地址。
func (a *App) configOAuthOpenURL(url string) {
	if a.intents == nil || url == "" {
		return
	}
	a.dispatchQuiet("打开授权地址", func(ctx context.Context) error {
		return a.intents.OpenProviderOAuthURL(ctx, url)
	}, nil, func(reason string) {
		configOAuthFlow.err = strings.TrimPrefix(reason, "打开授权地址：")
	})
}

// configOAuthCopy 复制授权 / 验证地址并给回执。
func (a *App) configOAuthCopy(c *gui.Context, value string) {
	if value == "" {
		return
	}
	c.Services().WriteClipboard(value)
	c.Toast(configCopyCopied)
}
