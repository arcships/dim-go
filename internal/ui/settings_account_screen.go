package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 账户（首屏）的重建实现：结构、文案、控件、间距与配色按
// docs/domains/requirements/rebuild/account.md（下称"规格"）逐元素对齐，覆盖
// S-account-01 … S-account-112（112 个元素）与 B'.2 的邮箱登录对话框。
//
// 只读：事实全部经 a.liveAccount()/a.liveUsage()（= Source）读，屏内不缓存事实；
// 只保留可丢的界面状态（邮箱对话框开合与草稿、重置对话框的当前类型、刷新起点，
// 见 app.go 的 App 字段）。写事实只走既有 Intents（登录、退出登录）。
//
// # 与规格的差异（对应 §3「无对应控件清单」与 §5「依赖与未定项」）
//
// 缺口（§3.1）的落地方式：
//
//	G-01 头像：不用 ui.Avatar（尺寸固定 FontSize×2.25、底色按名字哈希取 HSL，
//	     都覆盖不了），改为自绘正圆 + 首字母（规格 G-01 的方案 ②）。
//	G-02 进度条：Box + Draw 自绘（轨道 + 填充 + 全圆角），fill 按
//	     normal / warning / critical 三档取 accent-ink / warning / danger；
//	     无障碍用 Role(RoleProgress).Range(0,100,pct).Label(…)。
//	G-03 重置确认框：ui.Modal + 自绘面板（AlertDialog 的正文只有一串文字，
//	     放不下列表与三支状态）；Escape 由 Modal 内建，主按钮 AutoFocus。
//	G-04 批次加载骨架：没有 skeleton 控件，且数据层未给「加载中」支，
//	     条件不成立就不渲染 —— 登记为未实现项。
//	G-05 按容器宽断点：Context 没有宽度查询，用 c.Size() 的窗口宽判断
//	     （>= accSplitW 才左右分栏）。
//	G-06 半透明语义色：token@α = token.Alpha(α)，档位见 accAlpha*；进度条轨道取
//	     border@40（DS.meter-track 未定）。
//	G-07 ghost / outline / icon-only：ButtonBase 自绘，集中在 accButton / accIconButton。
//	G-08 text-box 光学裁切与 tracking-tight：不实现；图标盒固定 12/14/16，
//	     按钮内一律 Row(Center) 对齐（偏差登记为可接受）。
//	G-09 滚动条宽：沿用 Theme.ScrollbarWidth（没有改成参考的 10；D-04 是全局决定）。
//	G-10 图标：mygo 不带图标集；本屏 11 个形状用 Heroicons v2 outline 的内联路径
//	     （MIT，与 icons.go 同一套 24 网格），不新增素材文件。品牌标识（BrandMark）
//	     没有资产，用占位圆环（D-03）。
//
// 依赖未定项（§5）在本文件里的取值：
//
//	D-01 design-baseline.md 尚未生成：几何取参考 Tailwind 的确定值（见 acc* 常量），
//	     不按清单的包围盒量测值（采集时视口在 0.875–0.9 缩放）。
//	D-02 卡片圆角：取现役 U2 §8.3「控件与卡片同值」= Theme.Radius（6）；参考
//	     rounded-lg = 8，差 2（裁决未出）。
//	D-06 显示名为空时回落「Dim 账户」（参考实现）；需求 R-ACC-109 期望「未命名」（待裁决）。
//	D-07 日期与金额格式：数据层没给本地化字符串，本屏用 accDate / accAmount 的占位
//	     实现（「2026年12月13日」「10.8万 / 30万」的口径），契约落地后替换。
//	D-08 间距半档：按参考写 2 / 6 / 10 / 14 / 20（accSpace*），不吸附到七档标尺
//	     （吸附会整体改变本屏密度，规格 §3.4）。
//
// 未接动作（没有对应 Intents 方法，按 D14「未接动作」不提供写入）：
//
//	企业邮箱登录提交、用量刷新、重置额度「使用」、购买页外开（没有地址）。
//	控件照渲染、状态照禁用或可用，点击不写事实。
//
// 与规格控件映射表不同的三处替换：
//
//	状态徽标（S-account-39）用 ui.Badge 覆盖内边距/圆角/底色/文字色，但它的字号在
//	     框架里固定为 Theme.FontSize−2（macOS 上 11），不能覆盖 —— 结果与参考一致。
//	身份块（S-account-10/11/13）可选中：卡内 .Selectable()。
//	计数朵（S-account-45）常挂 1px 透明描边（C-05），换色时尺寸不跳。

// 本屏几何：值取参考产品的 Tailwind 确定值（规格 §5 的 DS.* 表）。
// docs/domains/requirements/design-baseline.md 落地后整体替换成 DS.*。
const (
	accCardHeaderH  = 56   // min-h-14
	accBtnHSm       = 32   // h-8
	accChipBtnH     = 28   // h-7
	accAvatar       = 48   // h-12 w-12
	accAvatarRadius = 24   // 正圆
	accAvatarInk    = 19   // 0.4 × accAvatar：首字母字号
	accIdentityMin  = 192  // min-w-[12rem]
	accDialogW      = 448  // sm:max-w-md
	accMeterH       = 6    // h-1.5
	accMeterHThin   = 4    // h-1
	accMeterAmountW = 152  // 9.5rem（四条用量行共用，C-04）
	accMeterPctW    = 80   // 5rem（同上）
	accWindowLabelW = 96   // w-24
	accMetricRowH   = 44   // min-h-11
	accMetricColW   = 176  // lg:min-w-[11rem]
	accChipRadius   = 4    // rounded（UID 芯片）
	accChipRadiusLg = 8    // rounded-lg（重置段容器）
	accPillRadius   = 999  // 全圆
	accBrandMark    = 36   // 套餐标识边长
	accBrandRadius  = 8
	accCountW       = 18 // 计数朵最小宽
	accHitMin       = hitMin
	accCopyHit      = hitMin // 复制 UID 的命中区（参考 18，抬到 hit-min，L13）
	accIcoBlock     = 16
	accIcoInline    = 14
	accIcoMicro     = 12
	accNoticeBar    = 3 // 耗尽提示左侧条（C-02）
	accRelaxed      = 1.625
	accSplitW       = 1024 // DS.bp-split（lg）
	accSpinMin      = 600 * time.Millisecond

	// 间距半档（D-08 待裁决，规格 §3.4）：不吸附到七档标尺。
	accSpaceHalf = 2  // space-0.5
	accSpace15   = 6  // space-1.5
	accSpace25   = 10 // space-2.5
	accSpace35   = 14 // space-3.5
	accSpace5    = 20 // space-5
)

// 半透明档位（G-06）：token.Alpha(accAlphaN)。
const (
	accAlpha0  = 0
	accAlpha05 = 0.05
	accAlpha10 = 0.10
	accAlpha25 = 0.25
	accAlpha40 = 0.40
	accAlpha50 = 0.50
	accAlpha70 = 0.70
	accAlpha80 = 0.80
)

// 本屏文案：逐字取自参考产品的 zh-CN（规格 §2.2 的「逐字文案」列）。
const (
	accCopyTitle = "账户"

	accCopyExpiredTitle  = "套餐已到期，Dim 托管模型暂不可用"
	accCopyExpiredDetail = "Dim 托管模型权益已停止，订阅套餐后即可恢复使用。"
	accCopyBuyPlan       = "选购套餐"

	accCopyUID     = "UID"
	accCopyCopyUID = "复制 UID"
	accCopyCopied  = "已复制"
	accCopyCopy    = "复制"

	accCopyViewPlan = "查看套餐"
	accCopySignOut  = "退出登录"

	accCopyLoginTitle  = "登录 Dim"
	accCopyLoginDetail = "登录后可解锁 Dim 托管模型。未登录时仍可在本地使用 DimAgent。"
	accCopySigningIn   = "正在登录…"
	accCopySignIn      = "登录"
	accCopyEnterprise  = "登录企业账号"
	accCopyEmailDetail = "请输入您的企业邮箱账号"
	accCopyEmailLabel  = "企业邮箱"
	accCopyEmailSample = "name@company.com"

	accCopyPlanUsage    = "套餐用量"
	accCopyRefresh      = "刷新"
	accCopyLoadingUsage = "正在加载用量…"
	accCopyQuotaUsage   = "额度用量"
	accCopyResetUsage   = "重置当月用量"
	accCopyResetTitle   = "重置当月用量？"
	accCopyResetDetail  = "将消耗 1 次重置次数。"
	accCopyNoBatches    = "暂无可用"
	accCopyBatchFailed  = "无法加载额度明细"
	accCopyCancel       = "取消"
	accCopyApply        = "使用"

	accCopyDepletedTitle  = "套餐额度已用完"
	accCopyDepletedDetail = "当前套餐额度已全部使用，Dim 托管模型暂时不可用。"

	accCopyUnlimited       = "不限量"
	accCopyUnlimitedDetail = "当前套餐不限用量，无额度上限。"

	accCopyMonthlyQuota = "月度额度"
	accCopyWindowLimit  = "周期限制"
	accCopyWindowDetail = "以下窗口包含在月度额度内"

	accCopyUnsubscribed       = "尚未订阅套餐"
	accCopyUnsubscribedDetail = "订阅套餐后即可解锁 Dim 托管模型。"
	accCopyNoUsage            = "暂无用量数据"
	accCopyRenew              = "续订 / 升级"

	accCopyReferral     = "推广邀请"
	accCopyInviteCode   = "邀请码"
	accCopyInviteURL    = "邀请网址"
	accCopyInvited      = "已邀请"
	accCopyPaid         = "已付费"
	accCopyReward       = "累计奖励"
	accCopyBrand        = "DimAgent"
	accCopyFallbackName = "Dim 账户" // D-06：参考实现的 fallbackName
	accCopyActive       = "生效中"
	accCopyExpired      = "已到期"
)

// 本屏图标：Heroicons v2 outline（24 网格、1.5 描边、MIT）的内联路径，
// 与 icons.go 同一套约定：只放形状，颜色取调用处给的色。
var (
	accIconTriangleAlert = mustSVG(`<path d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z"/>`)
	accIconCart          = mustSVG(`<path d="M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 0 0-3 3h15.75m-12.75-3h11.218c1.121-2.3 2.1-4.684 2.924-7.138a60.114 60.114 0 0 0-16.536-1.84M7.5 14.25 5.106 5.272M6 20.25a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0Zm12.75 0a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0Z"/>`)
	accIconCopy          = mustSVG(`<path d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 0 1-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 0 1 1.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 0 0-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 0 1-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 0 0-3.375-3.375h-1.5a1.125 1.125 0 0 1-1.125-1.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H9.75"/>`)
	accIconExternal      = mustSVG(`<path d="M13.5 6H5.25A2.25 2.25 0 0 0 3 8.25v10.5A2.25 2.25 0 0 0 5.25 21h10.5A2.25 2.25 0 0 0 18 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25"/>`)
	accIconRefresh       = mustSVG(`<path d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99"/>`)
	accIconRotateCcw     = mustSVG(`<path d="M9 15 3 9m0 0 6-6M3 9h12a6 6 0 0 1 0 12h-3"/>`)
	accIconCalendar      = mustSVG(`<path d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 0 1 2.25-2.25h13.5A2.25 2.25 0 0 1 21 7.5v11.25m-18 0A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75m-18 0v-7.5A2.25 2.25 0 0 1 5.25 9h13.5A2.25 2.25 0 0 1 21 11.25v7.5"/>`)
	accIconGlobe         = mustSVG(`<path d="M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Zm0 0a8.949 8.949 0 0 0 4.951-1.488A3.987 3.987 0 0 0 13 16h-2a3.987 3.987 0 0 0-3.951 3.512A8.949 8.949 0 0 0 12 21Zm3-11.25a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"/>`)
	accIconGift          = mustSVG(`<path d="M21 11.25v8.25a1.5 1.5 0 0 1-1.5 1.5H4.5a1.5 1.5 0 0 1-1.5-1.5v-8.25M12 4.875A2.625 2.625 0 1 0 9.375 7.5H12m0-2.625V7.5m0-2.625A2.625 2.625 0 1 1 14.625 7.5H12m0 0V21m-8.625-9.75h18c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125h-18c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125Z"/>`)
	accIconUpRight       = mustSVG(`<path d="M4.5 19.5 19.5 4.5m0 0H8.25m11.25 0v11.25"/>`)
	// accIconBrandMark 是套餐标识的占位形状：品牌标识（D-03）没有资产，用一道
	// 圆环加内弧表示「产品标记」，不冒充任何真实 logo。
	accIconBrandMark = mustSVG(`<path d="M20.25 12a8.25 8.25 0 1 1-8.25-8.25"/><path d="M12 7.5a4.5 4.5 0 1 0 4.5 4.5"/>`)
)

// accMeterLevel 是进度条的三个配色档（G-02：参考的 normal / warning / critical）。
type accMeterLevel int

const (
	accMeterNormal accMeterLevel = iota
	accMeterWarning
	accMeterCritical
)

// accButtonVariant 是自绘按钮的两个变体（G-07）；主按钮仍用框架的 PrimaryButton。
type accButtonVariant int

const (
	accButtonOutline accButtonVariant = iota
	accButtonGhost
)

// accountScreenData 是本屏要显示的数据：接上 Source 时由事实填（缺项留空，
// 对应条件就不渲染）；-demo 时用 accountSampleData。这里只有值，没有文案。
type accountScreenData struct {
	SignedIn  bool
	SigningIn bool
	// Name/Email/UID 是身份；AccountError 是账号错误行的原文。
	Name         string
	Email        string
	UID          string
	AccountError string

	// PlanKnown 为假表示服务端没给套餐：状态徽标与到期行都不渲染。
	PlanKnown    bool
	PlanExpired  bool
	PlanSubtitle string
	ModelCount   int
	RenewalAt    time.Time
	RenewalDays  int
	// PurchaseURL 为空表示购买页地址没从数据层拿到：按钮照渲染，点击不外开。
	PurchaseURL string

	UsageLoading bool
	UsageError   string

	ResetsVisible bool
	ResetRows     []accountResetRow
	Depleted      bool
	Unlimited     bool
	// BatchesFailed/BatchLabels 是重置对话框的列表三支（§C.4）。
	BatchesFailed bool
	BatchLabels   []string

	CreditsKnown bool
	Credits      accountMeter
	Windows      []accountWindow
	Features     []accountMeter

	Unsubscribed bool
	Referral     accountReferral
}

// accountResetRow 是一种可重置的额度（参考按类型一行一颗胶囊）。
type accountResetRow struct {
	Type  string
	Text  string
	Hint  string
	Count int
}

// accountMeter 是一条额度读数（月度额度与功能额度共用同一条布局，C-04）。
type accountMeter struct {
	Label     string
	Feature   bool // 功能额度用 Globe，月度额度用 CalendarDays
	Amount    string
	Percent   float32
	ResetAt   time.Time
	Unlimited bool
}

// accountWindow 是「周期限制」里的一行窗口。
type accountWindow struct {
	Label   string
	Amount  string
	Percent float32
	ResetAt time.Time
}

// accountReferral 是推广邀请卡的数据；Eligible 为假整卡不渲染。
type accountReferral struct {
	Eligible    bool
	InviterRule string
	InviteeRule string
	CampaignOff string
	Code        string
	URL         string
	Invited     int
	Paid        int
	Reward      string
}

// accountScreenData 取本屏的数据：接上 Source 时只读事实，否则用样例。
func (a *App) accountScreenData() accountScreenData {
	account, ok := a.liveAccount()
	if !ok {
		return accountSampleData()
	}
	usage, hasUsage := a.liveUsage()
	return accountDataFrom(account, usage, hasUsage)
}

// accountDataFrom 把账号与用量的快照摊成本屏的数据；没有的字段留空。
func accountDataFrom(account AccountSnapshot, usage UsageSnapshot, hasUsage bool) accountScreenData {
	data := accountScreenData{
		SignedIn:  account.Status == "signed_in",
		SigningIn: account.LoginPhase == "waiting",
		Name:      account.Name,
		Email:     account.Email,
	}
	if data.SignedIn && data.Name == "" {
		data.Name = account.Initial
	}
	if account.LoginPhase == "failed" {
		data.AccountError = account.LoginReason
	}
	if data.SignedIn {
		data.PlanKnown = account.Plan != ""
		data.PlanSubtitle = account.Plan
		data.Unsubscribed = !data.PlanKnown
	}
	if hasUsage && usage.Available && usage.Limit > 0 {
		data.CreditsKnown = true
		data.Credits = accountMeter{
			Label:   accCopyMonthlyQuota,
			Amount:  accAmount(usage.Used, usage.Limit, usage.Unit),
			Percent: float32(usage.Used / usage.Limit * 100),
			ResetAt: usage.ResetsAt,
		}
		data.Depleted = usage.Used >= usage.Limit
	} else if hasUsage && usage.Reason != "" {
		data.UsageError = usage.Reason
	}
	return data
}

// accountSampleData 是 -demo 的样例：数值与内容取自参考产品的账户屏，
// 用来并排走查（不是事实；接上 Source 后一条都不上屏）。
func accountSampleData() accountScreenData {
	renewal := time.Date(2026, time.December, 13, 0, 0, 0, 0, time.Local)
	return accountScreenData{
		SignedIn:      true,
		Name:          "eric",
		Email:         "chenghan@fazhitech.cn",
		UID:           "u_9560e10671368346",
		PlanKnown:     true,
		PlanSubtitle:  "法至内部使用 · ¥0.10 / 月",
		ModelCount:    32,
		RenewalAt:     renewal,
		RenewalDays:   26,
		ResetsVisible: true,
		ResetRows: []accountResetRow{
			{Type: "monthly", Text: accCopyResetUsage, Hint: "本月已用量清零，额度回到满额"},
		},
		CreditsKnown: true,
		Credits: accountMeter{
			Label:   accCopyMonthlyQuota,
			Amount:  "10.8万 / 30万",
			Percent: 36,
			ResetAt: renewal,
		},
		Features: []accountMeter{{
			Label:   "Web Search",
			Feature: true,
			Amount:  "334 / 1,500 次",
			Percent: 22,
			ResetAt: renewal,
		}},
		Referral: accountReferral{
			Eligible:    true,
			InviterRule: "好友完成首次订阅付费后，你获得其所购套餐额度 20% 的 Credits。",
			InviteeRule: "好友首次订阅可获得所购套餐额度 20% 的 Credits。",
			Code:        "LYIr",
			URL:         "https://dimagent.cn/register?aff=LYIr",
			Invited:     11,
			Paid:        6,
			Reward:      "9,125 Credits",
		},
	}
}

// settingsAccountScreen 是设置 › 账户这一屏（设置壳的一个分节）。
//
// L0 屏幕：Column（上下滚动）+ Gap space-5；页首带 A、已登录带 B / 未登录带 B'、
// 套餐用量卡 C、推广邀请卡 D 依次排下来，两个对话框挂在屏末尾。
func (a *App) settingsAccountScreen(c *gui.Context) {
	data := a.accountScreenData()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(accSpace5).Children(func() {
				accountPageTitle(c)
				if data.SignedIn {
					a.accountSignedIn(c, data)
				} else {
					a.accountSignedOut(c, data)
				}
				if data.SignedIn {
					a.accountPlanCard(c, data)
				}
				if data.Referral.Eligible {
					a.accountReferralCard(c, data.Referral)
				}
			})
		})
	})
	a.accountEmailDialog(c)
	a.accountResetDialog(c, data)
}

// accountPageTitle 是 A.1：页面标题「账户」（S-account-01）。
func accountPageTitle(c *gui.Context) {
	t := c.Theme()
	typeH2.style(t, gui.Text(c, accCopyTitle)).TextColor(t.Text).
		Role(gui.RoleHeading).Level(2)
}

// accountSignedIn 是 B 已登录带：到期横幅 + 身份卡 + 账号错误行。
func (a *App) accountSignedIn(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	gui.Column(c).Gap(accSpace5).Children(func() {
		if d.PlanExpired {
			a.accountExpiryBanner(c, d)
		}
		a.accountIdentityCard(c, d)
		if d.AccountError != "" {
			typeSecondary.style(t, gui.Text(c, d.AccountError)).TextColor(t.Danger).
				Margin(space3, 0, 0, 0).Label("账号错误")
		}
	})
}

// accountExpiryBanner 是 B.1 到期横幅（S-account-02–07）。
func (a *App) accountExpiryBanner(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	banner := gui.Row(c).Wrap().Shrink(0).
		AlignItems(gui.Center).GapX(space3).GapY(space3).
		Padding(space3, space4).Radius(radius).
		Background(t.Danger.Alpha(accAlpha05)).
		Role(gui.RoleStatus).Label("套餐到期提示")
	banner.BorderWidth(strokeHairline).BorderColor(t.Danger.Alpha(accAlpha25))
	banner.Children(func() {
		accIcon(c, accIconTriangleAlert, accIcoBlock, t.Danger, "triangle-alert")
		gui.Column(c).Grow(1).MinWidth(0).Gap(accSpaceHalf).Children(func() {
			typeBodyStrong.style(t, gui.Text(c, accCopyExpiredTitle)).TextColor(t.Danger).
				SingleLine()
			typeSecondary.style(t, gui.Text(c, accCopyExpiredDetail)).
				TextColor(t.Danger.Alpha(accAlpha80)).Selectable()
		})
		if accButton(c, accButtonSpec{
			variant: accButtonOutline,
			height:  accBtnHSm,
			label:   accCopyBuyPlan,
			children: func() {
				gui.Row(c).Gap(accSpace15).AlignItems(gui.Center).Children(func() {
					accIcon(c, accIconCart, accIcoInline, t.Text, "shopping-cart")
					typeControl.style(t, gui.Text(c, accCopyBuyPlan)).TextColor(t.Text).SingleLine()
				})
			},
		}) {
			a.accountOpenPurchase(c, d)
		}
	})
}

// accountOpenPurchase 打开购买页：地址来自数据层（D-07），没有地址就不外开。
func (a *App) accountOpenPurchase(c *gui.Context, d accountScreenData) {
	if d.PurchaseURL == "" {
		return
	}
	c.Services().OpenURL(d.PurchaseURL)
}

// accountIdentityCard 是 B.2 身份卡（S-account-08–19）：头像 + 名称/邮箱/UID，
// 右侧是「查看套餐」与「退出登录」。
func (a *App) accountIdentityCard(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	card := accCardFrame(c, gui.Row(c).Wrap().Shrink(0).
		AlignItems(gui.Center).GapX(space4).GapY(space3).Padding(accSpace5)).
		Role(gui.RoleGroup).Label("账户身份").Selectable()
	card.Children(func() {
		gui.Row(c).Grow(1).MinWidth(accIdentityMin).Gap(space4).
			AlignItems(gui.Center).Children(func() {
			accountAvatar(c, d.Name)
			gui.Column(c).Grow(1).MinWidth(0).Children(func() {
				name := d.Name
				if name == "" {
					name = accCopyFallbackName
				}
				typeBodyStrong.style(t, gui.Text(c, name)).TextColor(t.Text).SingleLine()
				if d.Email != "" {
					typeSecondary.style(t, gui.Text(c, d.Email)).TextColor(t.TextMuted).
						SingleLine().Unselectable()
				}
				if d.UID != "" {
					accountUIDRow(c, d.UID)
				}
			})
		})
		gui.Row(c).Shrink(0).Wrap().GapX(space3).GapY(space2).
			AlignItems(gui.Center).Children(func() {
			a.accountViewPlanButton(c, d)
			a.accountSignOutButton(c, d)
		})
	})
}

// accountAvatar 是头像（S-account-09）：48 正圆、强调底、反白首字母。
// 自绘而非 ui.Avatar：它的尺寸与配色都不可覆盖（G-01）。
func accountAvatar(c *gui.Context, name string) {
	t := c.Theme()
	initials := accInitials(name)
	box := gui.Box(c).Size(accAvatar, accAvatar).Radius(accAvatarRadius).Shrink(0).
		Background(t.Accent).Center().Role(gui.RoleImage)
	if name != "" {
		box = box.Label(name)
	} else {
		box = box.Label("账户头像")
	}
	box.Children(func() {
		if initials == "" {
			accIcon(c, accIconBrandMark, accAvatarInk, t.AccentText, "")
			return
		}
		gui.Text(c, initials).FontSize(accAvatarInk).FontWeight(600).
			TextColor(t.AccentText).TextAlign(gui.Center).SingleLine()
	})
}

// accInitials 是头像里的首字母：一个词取前两个拉丁字母（eric → ER），
// 多个词取首末两词的首字母（Ada Lovelace → AL），中文取首字。
func accInitials(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return ""
	}
	first := []rune(fields[0])
	if len(fields) > 1 {
		last := []rune(fields[len(fields)-1])
		return strings.ToUpper(string(first[0]) + string(last[0]))
	}
	if len(first) == 1 || first[0] > 0x7f {
		return strings.ToUpper(string(first[:1]))
	}
	return strings.ToUpper(string(first[:2]))
}

// accountUIDRow 是 UID 行（S-account-12–15）：「UID」+ 等宽芯片 + 复制入口。
func accountUIDRow(c *gui.Context, uid string) {
	t := c.Theme()
	gui.Row(c).Gap(accSpace15).AlignItems(gui.Center).
		Margin(accSpace15, 0, 0, 0).Children(func() {
		typeMicro.style(t, gui.Text(c, accCopyUID)).TextColor(t.TextMuted).Unselectable()
		accFieldFrame(c, gui.Box(c).Shrink(0).
			Padding(accSpaceHalf, accSpace15).Radius(accChipRadius)).
			Children(func() {
				gui.Text(c, uid).Font(codeFont).FontSize(typeCode.sizeIn(t)).
					TextColor(t.Text).SingleLine()
			})
		if accIconButton(c, accIconSpec{
			icon: accIconCopy, size: accIcoMicro, hit: accCopyHit,
			label: accCopyCopyUID, tip: accCopyCopyUID,
		}) {
			accountCopy(c, uid)
		}
	})
}

// accountViewPlanButton 是「查看套餐 · 32 个模型」（S-account-16–18）：
// 按钮只接受单串，所以用 Children 拼图标 + 两段文字（C-03）。
func (a *App) accountViewPlanButton(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	label := accCopyViewPlan
	if d.ModelCount > 0 {
		label = fmt.Sprintf("%s，%d 个模型", accCopyViewPlan, d.ModelCount)
	}
	if accButton(c, accButtonSpec{
		variant: accButtonOutline,
		height:  accBtnHSm,
		label:   label,
		children: func() {
			gui.Row(c).Gap(accSpace15).AlignItems(gui.Center).Children(func() {
				accIcon(c, accIconExternal, accIcoInline, t.Text, "external-link")
				typeControl.style(t, gui.Text(c, accCopyViewPlan)).TextColor(t.Text).SingleLine()
				if d.ModelCount > 0 {
					typeSecondary.style(t, gui.Text(c, "·")).TextColor(t.TextMuted).
						AccessHidden(true)
					typeSecondary.style(t, gui.Text(c, fmt.Sprintf("%d 个模型", d.ModelCount))).
						TextColor(t.TextMuted).SingleLine()
				}
			})
		},
	}) {
		a.accountOpenPurchase(c, d)
	}
}

// accountSignOutButton 是「退出登录」（S-account-19）：ghost，刷新中禁用。
func (a *App) accountSignOutButton(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	_ = d
	if accButton(c, accButtonSpec{
		variant:  accButtonGhost,
		height:   accBtnHSm,
		label:    accCopySignOut,
		disabled: a.accountRefreshing(),
		children: func() {
			typeControl.style(t, gui.Text(c, accCopySignOut)).TextColor(t.Text).SingleLine()
		},
	}) {
		a.dispatch(accCopySignOut, func(ctx context.Context) error {
			return a.intents.Logout(ctx)
		}, nil)
	}
}

// accountRefreshing 报告用量刷新是否仍在进行（C-06：最短旋转时长在屏内保证，
// 不在 view 里用一次性延时）。视图外的状态只有刷新起点。
func (a *App) accountRefreshing() bool {
	if a.accountRefreshAt.IsZero() {
		return false
	}
	if time.Since(a.accountRefreshAt) < accSpinMin {
		return true
	}
	a.accountRefreshAt = time.Time{}
	return false
}

// accountSignedOut 是 B' 未登录带：未登录卡（S-account-21–28）。
func (a *App) accountSignedOut(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	card := accCardFrame(c, gui.Column(c).Shrink(0).
		AlignItems(gui.Start).Gap(space4).Padding(space6)).
		Role(gui.RoleGroup).Label(accCopyLoginTitle)
	card.Children(func() {
		gui.Column(c).Gap(space2).Children(func() {
			typeH3.style(t, gui.Text(c, accCopyLoginTitle)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(3)
			typeBody.style(t, gui.Text(c, accCopyLoginDetail)).TextColor(t.TextMuted)
		})
		gui.Row(c).Wrap().Gap(space3).AlignItems(gui.Center).Children(func() {
			if d.SigningIn {
				// S-account-24/25：登录中——旋转指示 + 文案，整钮禁用。
				signing := gui.PrimaryButton(c, "").Disabled(true).Label(accCopySigningIn)
				signing.Children(func() {
					gui.Row(c).Gap(accSpace15).AlignItems(gui.Center).Children(func() {
						gui.Spinner(c).Size(accIcoBlock, accIcoBlock).Label(accCopySignIn)
						gui.Text(c, accCopySigningIn).SingleLine()
					})
				})
			} else {
				if gui.PrimaryButton(c, accCopySignIn).Label(accCopySignIn).Clicked() {
					a.dispatch(accCopySignIn, func(ctx context.Context) error {
						_, err := a.intents.BeginLogin(ctx, "browser")
						return err
					}, nil)
				}
				if accButton(c, accButtonSpec{
					variant: accButtonOutline, height: accBtnHSm, label: accCopyEnterprise,
					children: func() {
						typeControl.style(t, gui.Text(c, accCopyEnterprise)).
							TextColor(t.Text).SingleLine()
					},
				}) {
					a.accountEmailOpen = true
					a.accountEmailDraft = ""
				}
			}
			if d.AccountError != "" {
				typeSecondary.style(t, gui.Text(c, d.AccountError)).TextColor(t.Danger).
					Label("账号错误")
			}
		})
	})
}

// accountEmailDialog 是 B'.2 邮箱登录对话框（S-account-29）：Modal + 自绘面板。
func (a *App) accountEmailDialog(c *gui.Context) {
	if !a.accountEmailOpen {
		return
	}
	t := c.Theme()
	open := true
	m := gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space3).Children(func() {
			typeH3.style(t, gui.Text(c, accCopyEnterprise)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(3)
			typeBody.style(t, gui.Text(c, accCopyEmailDetail)).TextColor(t.TextMuted)
			typeBodyStrong.style(t, gui.Text(c, accCopyEmailLabel)).TextColor(t.Text)
			gui.TextInput(c, &a.accountEmailDraft).Label(accCopyEmailLabel).
				Placeholder(accCopyEmailSample)
			gui.Row(c).Justify(gui.End).Children(func() {
				// 企业邮箱登录没有对应的 Intents 方法（未接动作）：提交不写事实。
				gui.PrimaryButton(c, accCopyEnterprise).Label(accCopyEnterprise)
			})
		})
		panel.Width(accDialogWidth(c))
	})
	if !open || m.Dismissed() {
		a.accountEmailOpen = false
	}
}

// accDialogWidth 是对话框面板宽：参考 sm:max-w-md = 448；窗口更窄时收进窗口内。
func accDialogWidth(c *gui.Context) float32 {
	width, _ := c.Size()
	limit := width - 2*space8
	if limit < accDialogW {
		if limit < space12*4 {
			return space12 * 2
		}
		return limit
	}
	return accDialogW
}

// accountPlanCard 是 C 套餐用量卡（S-account-30–95）。
func (a *App) accountPlanCard(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	card := accCardFrame(c, gui.Column(c).Shrink(0)).Clip().
		Role(gui.RoleGroup).Label(accCopyPlanUsage)
	card.Children(func() {
		a.accountPlanHeader(c)
		if d.UsageError != "" {
			typeSecondary.style(t, gui.Text(c, d.UsageError)).TextColor(t.Danger).
				Padding(space4, accSpace5).Label("用量错误")
		}
		if d.UsageLoading {
			typeSecondary.style(t, gui.Text(c, accCopyLoadingUsage)).TextColor(t.TextMuted).
				Padding(space4, accSpace5)
		}
		gui.Column(c).Gap(space4).Padding(accSpace5, accSpace5).Children(func() {
			a.accountServiceRow(c, d)
			if d.ResetsVisible {
				a.accountResetSection(c, d)
			}
			if d.Depleted {
				accountDepletedNotice(c)
			}
			if d.Unlimited {
				accountUnlimitedBox(c)
			}
			if d.CreditsKnown {
				a.accountCreditsBox(c, d)
			} else if len(d.Windows) > 0 {
				accountWindowBox(c, d.Windows)
			}
			for i := range d.Features {
				a.accountFeatureBox(c, d.Features[i])
			}
			if d.Unsubscribed {
				a.accountUnsubscribedRow(c, d)
			}
			if !d.CreditsKnown && len(d.Windows) == 0 && len(d.Features) == 0 {
				typeSecondary.style(t, gui.Text(c, accCopyNoUsage)).TextColor(t.TextMuted)
			}
			if d.PlanKnown {
				a.accountRenewRow(c, d)
			}
		})
	})
}

// accountPlanHeader 是 C.1 卡头（S-account-30–33）：「套餐用量」+ 刷新。
func (a *App) accountPlanHeader(c *gui.Context) {
	t := c.Theme()
	head := gui.Row(c).Shrink(0).Justify(gui.SpaceBetween).AlignItems(gui.Center).
		Gap(space3).MinHeight(accCardHeaderH).Padding(space3, accSpace5)
	head.BorderWidth(0, 0, strokeHairline, 0).BorderColor(t.Border.Alpha(accAlpha70))
	head.Children(func() {
		typeBodyStrong.style(t, gui.Text(c, accCopyPlanUsage)).TextColor(t.Text).
			Role(gui.RoleHeading).Level(3)
		if accIconButton(c, accIconSpec{
			icon: accIconRefresh, size: accIcoBlock, hit: rowRegular,
			label: accCopyRefresh, tip: accCopyRefresh,
			disabled: a.accountRefreshing(), spin: a.accountRefreshing(),
		}) {
			a.accountRefreshAt = time.Now()
		}
	})
}

// accountServiceRow 是 C.3 服务行（S-account-36–41）：品牌标 + 名称 + 副标题 +
// 状态徽标 + 到期文案。
func (a *App) accountServiceRow(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	gui.Row(c).Wrap().Shrink(0).AlignItems(gui.Center).
		GapX(accSpace25).GapY(space2).Children(func() {
		accountBrandMark(c, d.PlanExpired)
		typeBodyStrong.style(t, gui.Text(c, accCopyBrand)).TextColor(t.Text)
		if d.PlanSubtitle != "" {
			typeSecondary.style(t, gui.Text(c, d.PlanSubtitle)).TextColor(t.TextMuted).
				SingleLine().MinWidth(0)
		}
		if d.PlanKnown {
			label := accCopyActive
			if d.PlanExpired {
				label = accCopyExpired
			}
			accBadge(c, label, d.PlanExpired)
		}
		if d.PlanExpired {
			accIcon(c, iconWarn, accIcoMicro, t.Danger, "circle-alert")
		}
		if renewal := accRenewalText(d); renewal != "" {
			typeSecondary.style(t, gui.Text(c, renewal)).TextColor(t.TextMuted).
				MarginStart(gui.Auto).SingleLine()
		}
	})
}

// accountBrandMark 是套餐标识（S-account-36）：强调底的方块，到期时换中性面。
func accountBrandMark(c *gui.Context, expired bool) {
	t := c.Theme()
	box := gui.Box(c).Size(accBrandMark, accBrandMark).Radius(accBrandRadius).
		Shrink(0).Center().Label(accCopyBrand)
	color := colorAccentInk.in(t)
	if expired {
		box = box.Background(t.SurfaceHover)
		color = t.TextMuted
	} else {
		box = box.Background(t.Accent.Alpha(accAlpha10))
	}
	box.Children(func() {
		accIcon(c, accIconBrandMark, accBrandMark/2, color, "brand-mark")
	})
}

// accRenewalText 是「到期 2026年12月13日 · 还剩 26 天」/「已于 … 到期」（S-account-41）。
func accRenewalText(d accountScreenData) string {
	if d.RenewalAt.IsZero() {
		return ""
	}
	date := accDate(d.RenewalAt)
	switch {
	case d.PlanExpired || d.RenewalDays < 0:
		return "已于 " + date + " 到期"
	case d.RenewalDays == 0:
		return "到期 " + date + " · 今天结束"
	}
	return fmt.Sprintf("到期 %s · 还剩 %d 天", date, d.RenewalDays)
}

// accountResetSection 是 C.4 重置段：一颗「额度用量」选中芯片 + 每种重置类型一颗胶囊。
func (a *App) accountResetSection(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	gui.Row(c).Wrap().Shrink(0).AlignSelf(gui.Start).AlignItems(gui.Center).Gap(space1).
		Padding(space1).Radius(accChipRadiusLg).
		Background(t.SurfaceHover).Label("额度重置").Children(func() {
		// S-account-42：「额度用量」是静态选中项样式，不是可切换控件。
		gui.Box(c).Shrink(0).Padding(space1, accSpace25).Radius(radius).
			Background(t.Surface).Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, accCopyQuotaUsage)).TextColor(t.Text).
				SingleLine()
		})
		for _, row := range d.ResetRows {
			a.accountResetPill(c, row)
		}
	})
}

// accountResetPill 是 S-account-43–45：「重置当月用量」+ 剩余次数。
func (a *App) accountResetPill(c *gui.Context, row accountResetRow) {
	t := c.Theme()
	disabled := row.Count == 0
	color := t.Text
	if disabled {
		color = t.TextMuted
	}
	if accButton(c, accButtonSpec{
		variant: accButtonGhost, height: accChipBtnH,
		label: row.Text, tip: row.Hint, disabled: disabled,
		paddingX: accSpace25, paddingY: space1,
		children: func() {
			gui.Row(c).Gap(accSpace15).AlignItems(gui.Center).Children(func() {
				accIcon(c, accIconRotateCcw, accIcoMicro, color, "rotate-ccw")
				typeSecondary.style(t, gui.Text(c, row.Text)).TextColor(color).SingleLine()
				accCountBubble(c, row.Count)
			})
		},
	}) {
		a.accountResetType = row.Type
	}
}

// accCountBubble 是 S-account-45 的计数朵：>0 强调底，=0 边框底；常挂 1px
// 透明描边（C-05），换色时尺寸不跳。
func accCountBubble(c *gui.Context, count int) {
	t := c.Theme()
	bg := t.Border.Alpha(accAlpha40)
	ink := t.TextMuted
	if count > 0 {
		bg = t.Accent.Alpha(accAlpha10)
		ink = colorAccentInk.in(t)
	}
	bubble := gui.Box(c).Shrink(0).MinWidth(accCountW).Padding(0, accSpace15).
		Radius(accPillRadius).Background(bg).Center().
		Label(fmt.Sprintf("剩余 %d 次", count))
	bubble.BorderWidth(strokeHairline).BorderColor(t.Border.Alpha(accAlpha0))
	bubble.Children(func() {
		typeMicro.style(t, gui.Text(c, strconv.Itoa(count))).TextColor(ink).
			FontWeight(600).FontFeatures("tnum").SingleLine()
	})
}

// accountDepletedNotice 是 C.5 耗尽提示（S-account-54–57），左侧 3px 条见 C-02。
func accountDepletedNotice(c *gui.Context) {
	t := c.Theme()
	notice := gui.Row(c).Shrink(0).Gap(space3).AlignItems(gui.Start).
		Padding(space3, space4).Radius(radius).Clip().
		Background(t.Danger.Alpha(accAlpha05)).
		Role(gui.RoleStatus).Label(accCopyDepletedTitle)
	notice.BorderWidth(strokeHairline).BorderColor(t.Danger.Alpha(accAlpha25))
	notice.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Fill(gui.Rect{X: r.X, Y: r.Y, W: accNoticeBar, H: r.H}, t.Danger, accNoticeBar)
	})
	notice.Children(func() {
		accIcon(c, iconWarn, accIcoBlock, t.Danger, "circle-alert")
		gui.Column(c).Grow(1).MinWidth(0).Gap(accSpaceHalf).Children(func() {
			typeBodyStrong.style(t, gui.Text(c, accCopyDepletedTitle)).TextColor(t.Danger)
			typeSecondary.style(t, gui.Text(c, accCopyDepletedDetail)).
				TextColor(t.Danger.Alpha(accAlpha80)).LineHeight(accRelaxed)
		})
	})
}

// accMeterHead 是额度框的头行：图标 + 标题（S-account-63/64、83）。
func accMeterHead(c *gui.Context, label string, feature bool) {
	t := c.Theme()
	icon, mark := accIconCalendar, "calendar-days"
	if feature {
		icon, mark = accIconGlobe, "globe"
	}
	gui.Row(c).Shrink(0).Gap(accSpace15).AlignItems(gui.Center).Children(func() {
		accIcon(c, icon, accIcoInline, t.TextMuted, mark)
		typeSecondaryStrong.style(t, gui.Text(c, label)).TextColor(t.Text).SingleLine()
	})
}

// accountUnlimitedBox 是 C.6 不限量支（S-account-58–61）。
func accountUnlimitedBox(c *gui.Context) {
	t := c.Theme()
	box := gui.Column(c).Shrink(0).Padding(accSpace35, space4).Radius(radius).
		Label(accCopyQuotaUsage)
	box.BorderWidth(strokeHairline).BorderColor(t.Border.Alpha(accAlpha70))
	box.Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Children(func() {
			accMeterHead(c, accCopyQuotaUsage, false)
			typeSecondaryStrong.style(t, gui.Text(c, accCopyUnlimited)).
				TextColor(colorAccentInk.in(t)).MarginStart(gui.Auto).SingleLine()
		})
		typeSecondary.style(t, gui.Text(c, accCopyUnlimitedDetail)).TextColor(t.TextMuted).
			Margin(accSpace15, 0, 0, 0)
	})
}

// accountCreditsBox 是 C.7 月度额度（S-account-62–75）。
func (a *App) accountCreditsBox(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	box := gui.Column(c).Shrink(0).Padding(accSpace35, space4).Radius(radius).
		Label(accCopyMonthlyQuota)
	box.BorderWidth(strokeHairline).BorderColor(t.Border.Alpha(accAlpha70))
	box.Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Baseline).
			Justify(gui.SpaceBetween).Gap(space3).Children(func() {
			accMeterHead(c, accCopyMonthlyQuota, false)
			if !d.Credits.ResetAt.IsZero() {
				typeMicro.style(t, gui.Text(c, accResetLabel(d.Credits.ResetAt))).
					TextColor(t.TextMuted).SingleLine()
			}
		})
		accountMeterRow(c, d.Credits)
		if len(d.Windows) > 0 {
			accountWindowSection(c, d.Windows, true)
		}
	})
}

// accountMeterRow 是 C.7/C.9 的用量行（S-account-66–68、85–87）：金额列与
// 百分比列定宽（C-04），进度条吃掉剩下的宽度。
func accountMeterRow(c *gui.Context, meter accountMeter) {
	t := c.Theme()
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(space3).
		Margin(accSpace25, 0, 0, 0).Children(func() {
		typeBodyStrong.style(t, gui.Text(c, meter.Amount)).TextColor(t.Text).
			FontFeatures("tnum").Width(accMeterAmountW).SingleLine()
		if meter.Unlimited {
			return
		}
		accMeter(c, meter.Percent, accMeterH, accUsedLabel(meter.Percent))
		typeSecondaryStrong.style(t, gui.Text(c, accUsedLabel(meter.Percent))).
			TextColor(accMeterInk(t, meter.Percent)).FontFeatures("tnum").
			Width(accMeterPctW).TextAlign(gui.End).SingleLine()
	})
}

// accountWindowSection 是 C.8 周期限制（S-account-69–80）。inCredits 为真表示
// 这些窗口包含在月度额度里：进度条用细的一档，并多一行说明。
func accountWindowSection(c *gui.Context, rows []accountWindow, inCredits bool) {
	t := c.Theme()
	section := gui.Column(c).Shrink(0).Margin(space3, 0, 0, 0)
	section.BorderWidth(strokeHairline, 0, 0, 0).BorderColor(t.Border.Alpha(accAlpha70))
	section.Children(func() {
		typeMicro.style(t, gui.Text(c, accCopyWindowLimit)).TextColor(t.Text).
			FontWeight(600).Padding(accSpace25, 0, 0, 0)
		if inCredits {
			typeMicro.style(t, gui.Text(c, accCopyWindowDetail)).TextColor(t.TextMuted)
		}
		for _, row := range rows {
			accountWindowRow(c, row, inCredits)
		}
	})
}

// accountWindowRow 是一个窗口行：标签列 96（S-account-71–75）。
func accountWindowRow(c *gui.Context, row accountWindow, thin bool) {
	t := c.Theme()
	barHeight := float32(accMeterH)
	if thin {
		barHeight = accMeterHThin
	}
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(accSpace25).
		Margin(space2, 0, 0, 0).Children(func() {
		typeSecondaryStrong.style(t, gui.Text(c, row.Label)).TextColor(t.Text).
			Width(accWindowLabelW).SingleLine()
		if row.Amount != "" {
			typeSecondary.style(t, gui.Text(c, row.Amount)).TextColor(t.TextMuted).
				FontFeatures("tnum").SingleLine()
		}
		accMeter(c, row.Percent, barHeight, row.Label)
		typeMicro.style(t, gui.Text(c, accUsedLabel(row.Percent))).TextColor(t.Text).
			FontFeatures("tnum").TextAlign(gui.End).SingleLine()
		if !row.ResetAt.IsZero() {
			typeMicro.style(t, gui.Text(c, accResetLabel(row.ResetAt))).TextColor(t.TextMuted).
				SingleLine()
		}
	})
}

// accountWindowBox 是 C.8 的「无月度额度」支（S-account-76–80）：没有月度额度
// 时，周期限制自己成框，进度条用标准高度。
func accountWindowBox(c *gui.Context, rows []accountWindow) {
	t := c.Theme()
	box := gui.Column(c).Shrink(0).Padding(accSpace35, space4).Radius(radius).
		Label(accCopyWindowLimit)
	box.BorderWidth(strokeHairline).BorderColor(t.Border.Alpha(accAlpha70))
	box.Children(func() {
		for _, row := range rows {
			accountWindowRow(c, row, false)
		}
	})
}

// accountFeatureBox 是 C.9 功能额度（S-account-81–87）：每个 meter 一框。
func (a *App) accountFeatureBox(c *gui.Context, meter accountMeter) {
	t := c.Theme()
	box := gui.Column(c).Shrink(0).Padding(accSpace35, space4).Radius(radius).
		Label(meter.Label)
	box.BorderWidth(strokeHairline).BorderColor(t.Border.Alpha(accAlpha70))
	box.Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Baseline).
			Justify(gui.SpaceBetween).Gap(space3).Children(func() {
			accMeterHead(c, meter.Label, true)
			if !meter.ResetAt.IsZero() {
				typeMicro.style(t, gui.Text(c, accResetLabel(meter.ResetAt))).
					TextColor(t.TextMuted).SingleLine()
			}
		})
		accountMeterRow(c, meter)
	})
}

// accountUnsubscribedRow 是 C.10 未订阅支（S-account-88–91）。
func (a *App) accountUnsubscribedRow(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(accSpace15).AlignItems(gui.Start).Children(func() {
		typeBodyStrong.style(t, gui.Text(c, accCopyUnsubscribed)).TextColor(t.Text)
		typeSecondary.style(t, gui.Text(c, accCopyUnsubscribedDetail)).TextColor(t.TextMuted)
		if accButton(c, accButtonSpec{
			variant: accButtonOutline, height: accBtnHSm, label: accCopyViewPlan,
			children: func() {
				gui.Row(c).Gap(accSpace15).AlignItems(gui.Center).Children(func() {
					accIcon(c, accIconExternal, accIcoInline, t.Text, "external-link")
					typeControl.style(t, gui.Text(c, accCopyViewPlan)).TextColor(t.Text).SingleLine()
				})
			},
		}) {
			a.accountOpenPurchase(c, d)
		}
	})
}

// accountRenewRow 是 C.11 续订 CTA（S-account-93–95）。
func (a *App) accountRenewRow(c *gui.Context, d accountScreenData) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Justify(gui.End).Padding(space1, 0, 0, 0).Children(func() {
		renew := gui.PrimaryButton(c, "").Label(accCopyRenew)
		renew.Children(func() {
			gui.Row(c).Gap(accSpace15).AlignItems(gui.Center).Children(func() {
				icon, mark := accIconUpRight, "arrow-up-right"
				if d.PlanExpired {
					icon, mark = accIconCart, "shopping-cart"
				}
				accIcon(c, icon, accIcoInline, t.AccentText, mark)
				gui.Text(c, accCopyRenew).SingleLine()
			})
		})
		if renew.Clicked() {
			a.accountOpenPurchase(c, d)
		}
	})
}

// accountReferralCard 是 D 推广邀请卡（S-account-96–112）。
func (a *App) accountReferralCard(c *gui.Context, r accountReferral) {
	t := c.Theme()
	card := accCardFrame(c, gui.Column(c).Shrink(0)).Clip().
		Role(gui.RoleGroup).Label(accCopyReferral)
	card.Children(func() {
		head := gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(space2).
			MinHeight(accCardHeaderH).Padding(space3, accSpace5)
		head.BorderWidth(0, 0, strokeHairline, 0).BorderColor(t.Border.Alpha(accAlpha70))
		head.Children(func() {
			accIcon(c, accIconGift, accIcoBlock, t.TextMuted, "gift")
			typeBodyStrong.style(t, gui.Text(c, accCopyReferral)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(3)
		})
		width, _ := c.Size()
		if width >= accSplitW {
			body := gui.Row(c).Shrink(0).AlignItems(gui.Start).Padding(accSpace5, accSpace5)
			body.Children(func() {
				accountReferralFields(c, r).Grow(1)
				accountReferralMetrics(c, r, true)
			})
			return
		}
		body := gui.Column(c).Shrink(0).AlignItems(gui.Stretch).Padding(accSpace5, accSpace5)
		body.Children(func() {
			accountReferralFields(c, r)
			accountReferralMetrics(c, r, false)
		})
	})
}

// accountReferralFields 是 D.2–D.4：规则说明 + 邀请码 + 邀请网址。
func accountReferralFields(c *gui.Context, r accountReferral) gui.Element {
	t := c.Theme()
	fields := gui.Column(c).Shrink(0).Gap(accSpace25)
	fields.Children(func() {
		for _, rule := range []string{r.InviterRule, r.InviteeRule, r.CampaignOff} {
			if rule == "" {
				continue
			}
			typeSecondary.style(t, gui.Text(c, rule)).TextColor(t.TextMuted).Unselectable()
		}
		if r.Code != "" {
			accountCopyField(c, accCopyInviteCode, r.Code, true)
		}
		if r.URL != "" {
			accountCopyField(c, accCopyInviteURL, r.URL, false)
		}
	})
	return fields
}

// accountCopyField 是一段「标签 + 可复制的码行」：邀请码加粗、邀请网址不加粗。
func accountCopyField(c *gui.Context, label, value string, strong bool) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(accSpace25).Children(func() {
		typeMicro.style(t, gui.Text(c, label)).TextColor(t.TextMuted).
			Margin(0, 0, space1, 0)
		row := accFieldFrame(c, gui.Row(c).Shrink(0).AlignItems(gui.Center).
			Gap(space2).Padding(accSpace15, accSpace15)).
			PaddingStart(space3).Selectable()
		row.Children(func() {
			code := gui.Text(c, value).Font(codeFont).FontSize(typeCode.sizeIn(t)).
				TextColor(t.Text).SingleLine().Grow(1)
			if strong {
				code = code.FontWeight(600)
			}
			if accButton(c, accButtonSpec{
				variant: accButtonGhost, height: accChipBtnH,
				label: accCopyCopy + label, tip: accCopyCopy,
				paddingX: space2, paddingY: space1,
				children: func() {
					gui.Row(c).Gap(accSpace15).AlignItems(gui.Center).Children(func() {
						accIcon(c, accIconCopy, accIcoMicro, t.Text, "copy")
						typeMicro.style(t, gui.Text(c, accCopyCopy)).TextColor(t.Text).SingleLine()
					})
				},
			}) {
				accountCopy(c, value)
			}
		})
	})
}

// accountReferralMetrics 是 D.5 指标组：宽窗是右侧一列（左边线），窄窗是三列。
func accountReferralMetrics(c *gui.Context, r accountReferral, wide bool) {
	t := c.Theme()
	metrics := accMetrics(r)
	if !wide {
		group := gui.Row(c).Shrink(0).Grow(1).GapX(space4).
			Margin(accSpace35, 0, 0, 0).Label(accCopyReferral + "指标")
		group.BorderWidth(strokeHairline, 0, 0, 0).BorderColor(t.Border.Alpha(accAlpha50))
		group.Padding(space3, 0, 0, 0).Children(func() {
			for _, m := range metrics {
				gui.Column(c).Grow(1).Shrink(1).Gap(accSpaceHalf).Children(func() {
					typeMicro.style(t, gui.Text(c, m.label)).TextColor(t.TextMuted)
					accMetricValue(c, m.value)
				})
			}
		})
		return
	}
	group := gui.Column(c).Shrink(0).Width(accMetricColW).Grow(1).
		MarginStart(accSpace5).Label(accCopyReferral + "指标")
	group.BorderWidth(0, 0, 0, strokeHairline).BorderColor(t.Border.Alpha(accAlpha50))
	group.PaddingStart(accSpace5).Children(func() {
		for i, m := range metrics {
			item := gui.Row(c).Shrink(0).Justify(gui.SpaceBetween).
				AlignItems(gui.Center).MinHeight(accMetricRowH).
				Label(m.label + "：" + m.value)
			if i < len(metrics)-1 {
				item.BorderWidth(0, 0, strokeHairline, 0).BorderColor(t.Border.Alpha(accAlpha50))
			}
			item.Children(func() {
				typeMicro.style(t, gui.Text(c, m.label)).TextColor(t.TextMuted)
				accMetricValue(c, m.value)
			})
		}
	})
}

// accMetric 是一条推广指标（标签 + 值）。
type accMetric struct {
	label string
	value string
}

// accMetrics 是 D.5 的三条指标：已邀请 / 已付费 / 累计奖励。
func accMetrics(r accountReferral) []accMetric {
	return []accMetric{
		{accCopyInvited, fmt.Sprintf("%d 人", r.Invited)},
		{accCopyPaid, fmt.Sprintf("%d 人", r.Paid)},
		{accCopyReward, dashIfEmpty(r.Reward)},
	}
}

// accMetricValue 是指标值：正文强档 + 等宽数字（S-account-110–112）。
func accMetricValue(c *gui.Context, value string) {
	t := c.Theme()
	typeBodyStrong.style(t, gui.Text(c, value)).TextColor(t.Text).
		FontFeatures("tnum").SingleLine()
}

// accountResetDialog 是 C.4 的重置确认框（S-account-46–53）：Modal + 自绘面板。
func (a *App) accountResetDialog(c *gui.Context, d accountScreenData) {
	if a.accountResetType == "" {
		return
	}
	t := c.Theme()
	open := true
	m := gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space3).Children(func() {
			typeH2.style(t, gui.Text(c, accCopyResetTitle)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(2)
			typeBody.style(t, gui.Text(c, accCopyResetDetail)).TextColor(t.TextMuted)
			gui.Column(c).Gap(space1).Children(func() {
				switch {
				case d.BatchesFailed:
					typeSecondary.style(t, gui.Text(c, accCopyBatchFailed)).TextColor(t.Danger)
				case len(d.BatchLabels) == 0:
					typeSecondary.style(t, gui.Text(c, accCopyNoBatches)).TextColor(t.TextMuted)
				default:
					for _, label := range d.BatchLabels {
						typeSecondary.style(t, gui.Text(c, label)).TextColor(t.Text)
					}
				}
			})
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				if gui.Button(c, accCopyCancel).Label(accCopyCancel).Clicked() {
					a.accountResetType = ""
					open = false
				}
				// 「使用」要写额度重置（没有对应的 Intents 方法）：先禁用，
				// 等接口落地再打开（未接动作）。
				gui.PrimaryButton(c, accCopyApply).Label(accCopyApply).
					Disabled(true).AutoFocus()
			})
		})
		panel.Width(accDialogWidth(c))
	})
	if !open || m.Dismissed() {
		a.accountResetType = ""
	}
}

// ---- 组件：按钮、图标入口、进度条、徽标、外框 ----

// accButtonSpec 是一个自绘按钮：ghost / outline 两变体（G-07）。
type accButtonSpec struct {
	variant  accButtonVariant
	label    string
	tip      string
	height   float32
	paddingX float32
	paddingY float32
	disabled bool
	children func()
}

// accButton 画一个自绘按钮并返回它是否被按下。五态：默认 / 悬停 / 按下 /
// 聚焦（框架焦点环）/ 禁用（整棵子树 0.5，与框架一致）。
func accButton(c *gui.Context, spec accButtonSpec) bool {
	t := c.Theme()
	padX, padY := spec.paddingX, spec.paddingY
	if padX == 0 {
		padX = controlPadH
	}
	if padY == 0 {
		padY = controlPadV
	}
	height := spec.height
	if height == 0 {
		height = rowRegular
	}
	b := gui.ButtonBase(c).Shrink(0).Height(height).
		Padding(padY, padX).Radius(radius).AlignItems(gui.Center).
		Label(spec.label).Disabled(spec.disabled)
	if spec.tip != "" {
		b = b.Tooltip(spec.tip)
	}
	switch {
	case spec.disabled:
		b = b.Opacity(accAlpha50)
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	if spec.variant == accButtonOutline {
		b = b.Border(strokeHairline, t.Border)
	}
	if spec.children != nil {
		b.Children(spec.children)
	}
	return b.Clicked()
}

// accIconSpec 是一个只有图标的入口（G-07 icon-only）。
type accIconSpec struct {
	icon     *gui.SVG
	label    string
	tip      string
	size     float32
	hit      float32
	disabled bool
	// spin 为真时图标持续旋转（刷新进行中）。
	spin bool
}

// accIconButton 画一个图标入口，返回它是否被按下。命中区不小于 hit-min（L13），
// 图标本身固定 12/14/16 三档。
func accIconButton(c *gui.Context, spec accIconSpec) bool {
	t := c.Theme()
	hit := spec.hit
	if hit < accHitMin {
		hit = accHitMin
	}
	b := gui.ButtonBase(c).Shrink(0).Size(hit, hit).Radius(radius).
		Center().Label(spec.label).Disabled(spec.disabled)
	if spec.tip != "" {
		b = b.Tooltip(spec.tip)
	}
	ink := t.TextMuted
	rotate := float32(0)
	if spec.spin {
		rotate = b.Loop("spin", time.Second, gui.Linear) * 360
	}
	switch {
	case spec.disabled:
		b = b.Opacity(accAlpha50)
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
		ink = t.Text
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
		ink = t.Text
	}
	b.Children(func() {
		icon := gui.Box(c).Width(spec.size).Height(spec.size).Shrink(0).
			Label("图标：" + spec.label).AccessHidden(true)
		if rotate != 0 {
			icon = icon.Rotate(rotate)
		}
		icon.DrawOver(func(p *gui.Painter, r gui.Rect) {
			p.Icon(spec.icon, r, ink)
		})
	})
	return b.Clicked()
}

// accIcon 画一个固定尺寸的装饰图标：尺寸不随字号变化（U2 §6.4）。
// mark 非空时带一个给检查器用的标记（对辅助技术隐藏）。
func accIcon(c *gui.Context, icon *gui.SVG, size float32, color gui.Color, mark string) {
	box := gui.Box(c).Width(size).Height(size).Shrink(0)
	if mark != "" {
		box = box.Label("图标：" + mark).AccessHidden(true)
	}
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		p.Icon(icon, r, color)
	})
}

// accMeter 是自绘进度条（G-02）：轨道 + 按档位的填充 + 全圆角，无障碍给
// RoleProgress 与 0–100 的范围。
func accMeter(c *gui.Context, percent, height float32, label string) {
	t := c.Theme()
	track := t.Border.Alpha(accAlpha40)
	fill := accMeterInk(t, percent)
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	bar := gui.Box(c).Shrink(1).Grow(1).MinWidth(0).Height(height).
		Role(gui.RoleProgress).Range(0, 100, float64(percent)).Label(label)
	bar.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Fill(r, track, height/2)
		if percent <= 0 {
			return
		}
		p.Fill(gui.Rect{X: r.X, Y: r.Y, W: r.W * percent / 100, H: r.H}, fill, height/2)
	})
}

// accMeterInk 是进度条与其百分比文字的颜色：普通档 accent-ink，接近上限
// warning，用尽 danger（参考 normal / warning / critical）。
func accMeterInk(t *gui.Theme, percent float32) gui.Color {
	switch accLevel(percent) {
	case accMeterCritical:
		return t.Danger
	case accMeterWarning:
		return t.Warning
	}
	return colorAccentInk.in(t)
}

// accLevel 把已用百分比换成档位：>=100 用尽，>= 上限警告档（contextNearPercent）
// 接近上限，其余普通。
func accLevel(percent float32) accMeterLevel {
	switch {
	case percent >= 100:
		return accMeterCritical
	case percent >= float32(contextNearPercent):
		return accMeterWarning
	}
	return accMeterNormal
}

// accUsedLabel 是「已使用 N%」（S-account-68/74/87）。
func accUsedLabel(percent float32) string {
	if percent < 0 {
		percent = 0
	}
	return fmt.Sprintf("已使用 %d%%", int(percent+0.5))
}

// accBadge 是状态徽标（S-account-39）：全圆角胶囊，生效中强调底，到期危险底。
func accBadge(c *gui.Context, text string, danger bool) {
	t := c.Theme()
	b := gui.Badge(c, text).Shrink(0).
		Padding(accSpaceHalf, space2).Radius(accPillRadius).Label(text)
	if danger {
		b = b.Background(t.Danger.Alpha(accAlpha10)).TextColor(t.Danger)
	} else {
		b = b.Background(t.Accent.Alpha(accAlpha10)).TextColor(colorAccentInk.in(t))
	}
}

// accCardFrame 是卡框：surface 面、1px 主题描边、卡片圆角（U2 §8.3 的 radius）。
// 全屏的卡框都从这里出，描边处数才登记得住（L11）。
func accCardFrame(c *gui.Context, e gui.Element) gui.Element {
	t := c.Theme()
	return e.Radius(radius).Border(strokeHairline, t.Border).Background(t.Surface)
}

// accFieldFrame 是行内字段框（UID 芯片、邀请码行）：1px 主题描边 + 次要面。
func accFieldFrame(c *gui.Context, e gui.Element) gui.Element {
	t := c.Theme()
	return e.Border(strokeHairline, t.Border).Background(t.Surface.Alpha(accAlpha40))
}

// accResetLabel 是「2026年12月13日 重置」（S-account-65/84）。
func accResetLabel(at time.Time) string {
	return accDate(at) + " 重置"
}

// accDate 是日期口径「2026年12月13日」（D-07：本地化格式应由数据层给）。
func accDate(at time.Time) string {
	return fmt.Sprintf("%d年%d月%d日", at.Year(), int(at.Month()), at.Day())
}

// accAmount 是额度口径「10.8万 / 30万」「334 / 1,500 次」（D-07）。
func accAmount(used, limit float64, unit string) string {
	text := accNumber(used) + " / " + accNumber(limit)
	if unit != "" {
		text += " " + unit
	}
	return text
}

// accNumber 是数值口径：过万用「万」并留一位小数，其余加千分位。
func accNumber(v float64) string {
	if v >= 10000 {
		return strconv.FormatFloat(v/10000, 'f', 1, 64) + "万"
	}
	digits := strconv.FormatInt(int64(v+0.5), 10)
	if len(digits) <= 3 {
		return digits
	}
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// accountCopy 把一段内容写进剪贴板并给回执（S-account-14、104、108）。
func accountCopy(c *gui.Context, value string) {
	if value == "" {
		return
	}
	c.Services().WriteClipboard(value)
	c.Toast(accCopyCopied)
}
