package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 模型供应」详情面板的重建部分：认证方式、协议、名称 / 基础 URL /
// API 密钥的读数行、「测试连接 · 保存」、模型数据空态、断开 / 删除确认，以及
// 「添加模型」对话框。
//
// 元素对应 docs/domains/requirements/inventory-settings-rt.md §7 的
// S-configuration-44 … 63、70 … 75、81；每个函数头写它覆盖的元素，映射脚本按
// 那行注释定位渲染点。
//
// 只画有通道的字段：名称、基础 URL、API 密钥（走 provider.update）；协议、认证
// 方式、AWS 凭证、Region、User-Agent、组织 / 团队只在读数里出现，没有写入字段，
// 因此要么按读数只读展示（协议 / 认证方式），要么整行不画（AWS / Region /
// User-Agent / 组织），原因见 unavailable.go。
//
// 断开与删除没有协议方法：按钮保留、确认框照开，确认键置灰并写原因（DEC-04 的
// 同一条口径：留入口、写原因，不摆假动作）。「添加模型」也是本地草稿：能填、能
// 校验、能全选与过滤，保存没有通道，保存键置灰写原因。

// ---- 详情：读数行（S-configuration-44 / 46 / 50 / 54 / 55 / 56 / 57 / 58 / 61） ----

// configProviderBody 是读数里一个非托管供应商的详情：认证与协议读数 + 三个字段
// 行 + 测试连接 + 模型块 + 断开 / 删除。
func (a *App) configProviderBody(c *gui.Context, data configView) {
	gui.Column(c).Grow(1).MinHeight(0).Gap(configDetailGap).Children(func() {
		gui.Column(c).Shrink(0).Gap(configGapRow).Children(func() {
			// S-configuration-44 认证方式（读数，只读）
			configAuthRow(c, data.selected)
			// S-configuration-55 协议（读数，只读）
			configProtocolRow(c, data.selected)
			// S-configuration-56 名称 / 57 基础 URL / 46 与 61 API 密钥
			configCredentialRows(c, data)
			// S-configuration-50 测试连接 · 保存
			a.configTestConnectionRow(c, data)
		})
		a.configModelsPanel(c, data)
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Justify(gui.SpaceBetween).
			Gap(configGapRow).Children(func() {
			// S-configuration-62 / 63 断开与删除（确认框里置灰）
			configDangerRow(c, data.selected)
			// S-configuration-27 / 28 刷新与查看套餐
			a.configDetailFooter(c, data)
		})
	})
}

// configAuthState 把读数里的认证方式换成界面语：认不出来的保留原文。
func configAuthState(provider configProvider) string {
	switch strings.ToLower(strings.TrimSpace(provider.authType)) {
	case "bearer", "api_key", "apikey", "api-key":
		return "API Key"
	case "aws", "sigv4":
		return "Access Key"
	case "":
		return "—"
	}
	return provider.authType
}

// configAuthRow 是认证方式一行（S-configuration-44）：只显示当前那一个标签。
//
// 切换认证方式要写协议字段，provider.update 不带它，所以这里只读：整行置灰，
// 原因在 tooltip（noteProviderAuthWrite）。
func configAuthRow(c *gui.Context, provider configProvider) {
	t := c.Theme()
	authState := configAuthState(provider)
	gui.Column(c).Shrink(0).Gap(configGapIcon).Children(func() {
		configMicroStyle.style(t, gui.Text(c, "Authentication")).TextColor(t.TextMuted).
			Label("Authentication").SingleLine()
		chip := gui.Row(c).Shrink(0).AlignItems(gui.Center).
			Padding(configBadgePadV, configBadgePadH).Radius(configBadgeRadius).
			Background(t.Accent.Alpha(configBadgeFill)).Label("认证方式：" + authState).
			Tooltip(noteProviderAuthWrite).Disabled(true).Opacity(noteDisabledAlpha)
		chip.Children(func() {
			configBadgeStyle.style(t, gui.Text(c, authState)).TextColor(colorAccentInk.in(t)).
				SingleLine()
		})
	})
}

// configProtocolRow 是协议一行（S-configuration-55）：只读，显示映射后的名字；
// 协议名认不出来时显示服务端给的原始标识（S-configuration-55 的「异常」行）。
func configProtocolRow(c *gui.Context, provider configProvider) {
	t := c.Theme()
	rawReason := ""
	state := configProtocolName(provider.driverKind)
	if state == "" {
		// 协议名未映射：原始标识照写，tooltip 说明为什么不认识。
		state = strings.TrimSpace(provider.driverKind)
		rawReason = "协议名未映射，显示原始标识"
	}
	if state == "" {
		state = "—"
	}
	row := gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(configGapRow).
		Padding(configModelPadV, 0).Label("协议").Disabled(true).
		Opacity(noteDisabledAlpha).Tooltip(noteProviderProtocolWrite)
	if rawReason != "" {
		row = row.Tooltip(rawReason)
	}
	row.Children(func() {
		configMicroStyle.style(t, gui.Text(c, "协议")).TextColor(t.TextMuted).
			Width(configFieldLabelW).SingleLine()
		configRowStyle.style(t, gui.Text(c, state)).TextColor(t.Text).
			Grow(1).MinWidth(0).SingleLine()
	})
}

// configProtocolName 是协议标识到界面语的映射（参考的 ADAPTER_LABELS）；认不出
// 来返回空，让调用方显示原文。
func configProtocolName(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "openai":
		return "OpenAI"
	case "openai-responses", "openai_responses":
		return "OpenAI Responses"
	case "anthropic":
		return "Anthropic"
	case "gemini":
		return "Gemini"
	case "new-api", "new_api":
		return "New API"
	}
	return ""
}

// configCredentialRows 是三个可写字段的读数行（S-configuration-56 / 57 / 46 / 61）：
// 名称、基础 URL、API 密钥。每行显示读数值，行尾是进编辑态的入口。
func configCredentialRows(c *gui.Context, data configView) {
	provider := data.selected
	configValueRow(c, "名称", dashIfEmpty(provider.name), "编辑名称", func() {
		configScreen.form = configFormEdit(provider, data.models)
	})
	configValueRow(c, "基础 URL", dashIfEmpty(provider.baseURL), "编辑基础 URL", func() {
		configScreen.form = configFormEdit(provider, data.models)
	})
	// S-configuration-46 / S-configuration-61 API 密钥：只显示「已设置」的掩码，
	// 从不回显原文；点「修改」进编辑态，空着表示不改。
	credentialState := "未设置"
	if provider.credentialConfigured {
		credentialState = "••••••••"
	}
	configValueRow(c, "API 密钥", credentialState, "修改 API 密钥", func() {
		configScreen.form = configFormEdit(provider, data.models)
	})
	if provider.credentialConfigured {
		// 读屏读「已设置」，不读掩码字符（S-configuration-61 的无障碍行）。
		configMicroStyle.style(c.Theme(), gui.Text(c, "API 密钥已设置")).
			TextColor(c.Theme().TextMuted).Label("API 密钥已设置")
	}
}

// configValueRow 是一行字段读数：标签 + 值 + 行尾动作。值单行截断，悬停看全。
func configValueRow(c *gui.Context, label, value, actionLabel string, action func()) {
	t := c.Theme()
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(configGapRow).
		Padding(configModelPadV, 0).Children(func() {
		configMicroStyle.style(t, gui.Text(c, label)).TextColor(t.TextMuted).
			Width(configFieldLabelW).Label(label).SingleLine()
		configRowStyle.style(t, gui.Text(c, value)).TextColor(t.Text).
			Grow(1).MinWidth(0).SingleLine().Tooltip(value)
		if action != nil && screenAction(c, iconPencil, "edit", actionLabel, rowCompact) {
			action()
		}
	})
}

// configTestConnectionRow 是「测试连接 · 保存」（S-configuration-50）。
//
// 测试连接走 provider.probe（只读，不改配置）：进行中按钮禁用（S-configuration-50
// 的「测试禁用，保存可用」），失败原因写在行内、已填字段不动；成功给一句结果。
func (a *App) configTestConnectionRow(c *gui.Context, data configView) {
	t := c.Theme()
	provider := data.selected
	gui.Column(c).Shrink(0).Gap(configGapIcon).Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(configGapRow).Children(func() {
			testLabel := "测试连接"
			if configScreen.probing {
				testLabel = "正在测试…"
			}
			disabled := configScreen.probing || provider.id == ""
			tip := noteProviderProbeBusy
			if provider.id == "" {
				tip = noteProviderTestUnavailable
			}
			if configButton(c, testLabel, "测试连接", disabled, tip) {
				a.configProbeProvider(provider, probeTest)
			}
			// 保存 = 进编辑态：表单里才有校验与写入口（S-configuration-58 / 60）。
			if screenAction(c, iconPencil, "edit", "保存", rowCompact) {
				configScreen.form = configFormEdit(provider, data.models)
			}
		})
		if configScreen.probeErr != "" {
			// S-configuration-50 的「异常」：提示失败原因，不清空已填字段。
			configMicroStyle.style(t, gui.Text(c, "连接失败："+configScreen.probeErr)).
				TextColor(t.Danger).Role(gui.RoleAlert).MaxLines(2)
		} else if configScreen.probeNote != "" {
			configMicroStyle.style(t, gui.Text(c, configScreen.probeNote)).
				TextColor(t.TextMuted).Role(gui.RoleStatus).MaxLines(1)
		}
	})
}

// configDangerRow 是详情底部的断开与删除（S-configuration-62 / 63）：两个入口都
// 打开确认框；确认键置灰写原因（协议没有断开与删除方法）。
func configDangerRow(c *gui.Context, provider configProvider) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Gap(configGapRow).Children(func() {
		if screenAction(c, screenIconBan, "ban", "断开此提供商", rowCompact) {
			configScreen.disconnectFor = configKey(provider)
		}
		if screenAction(c, screenIconTrash, "trash", "删除此提供商", rowCompact) {
			configScreen.deleteFor = configKey(provider)
		}
		if configScreen.disconnectFor == "" && configScreen.deleteFor == "" {
			configMicroStyle.style(t, gui.Text(c, "断开与删除都要连接服务端才能生效")).
				TextColor(t.TextMuted).MaxLines(1)
		}
	})
}

// ---- 断开 / 删除确认（S-configuration-62 / 63） ----

// configProviderDialogs 画断开与删除的确认框：取消与 Esc 关闭且不改状态，确认键
// 置灰写原因（没有协议方法）。
func configProviderDialogs(c *gui.Context) {
	configDangerDialog(c, "disconnect")
	configDangerDialog(c, "delete")
}

// configDangerDialog 是两种确认框的共同形态：标题、说明、取消 / 确认。
func configDangerDialog(c *gui.Context, kind string) {
	t := c.Theme()
	target := &configScreen.disconnectFor
	title, confirm, description := "断开此提供商", "断开连接",
		"断开后要重新填写凭据才能使用这个提供商。"
	note := noteProviderDisconnectWrite
	if kind == "delete" {
		target = &configScreen.deleteFor
		title, confirm, description = "删除此提供商", "删除",
			"这个提供商及其模型设置会从列表里移除。"
		note = noteProviderDeleteWrite
	}
	if *target == "" {
		return
	}
	open := true
	dialog := gui.Modal(c, &open, func() {
		screenDialogPanel(c, title, func() {
			typeBody.style(t, gui.Text(c, description)).TextColor(t.TextMuted).
				Selectable().MaxLines(3)
			typeMicro.style(t, gui.Text(c, note)).TextColor(t.TextMuted).MaxLines(2)
			gui.Row(c).Justify(gui.End).Gap(screenSpace2).Children(func() {
				if screenButton(c, "取消", "取消"+title, "", false, rowRegular) {
					open = false
				}
				blocked := gui.PrimaryButton(c, confirm).Label(confirm).
					Height(rowRegular).Disabled(true).Opacity(noteDisabledAlpha).
					Tooltip(note)
				_ = blocked
			})
		})
	})
	if !open || dialog.Dismissed() {
		*target = ""
	}
}

// ---- 模型块：空态与「添加模型」（S-configuration-54 / 70 / 71 / 72 / 75） ----

// configModelsEmptyRow 是没有任何模型数据时的空态（S-configuration-54）：一句
// 点名「刷新」的说明 + 一个刷新按钮；拉取失败时这句话换成错误行。
func (a *App) configModelsEmptyRow(c *gui.Context, data configView) {
	t := c.Theme()
	showRefreshHint := true
	gui.Column(c).Shrink(0).Gap(configGapIcon).Padding(configModelPadV, 0).
		Children(func() {
			if configScreen.probeErr != "" {
				// S-configuration-54 的「异常」：拉取失败换成错误行。
				configMicroStyle.style(t, gui.Text(c, "拉取模型失败："+configScreen.probeErr)).
					TextColor(t.Danger).Role(gui.RoleAlert).MaxLines(2)
				return
			}
			if showRefreshHint {
				configRowStyle.style(t, gui.Text(c, "还没有模型数据，点击「刷新」获取")).
					TextColor(t.TextMuted).Selectable().Label("还没有模型数据，点击「刷新」获取")
			}
			if screenAction(c, screenIconRefresh, "refresh", "刷新", rowCompact) {
				a.configProbeProvider(data.selected, probeModels)
			}
		})
}

// configModelDraft 是「添加模型」对话框的本地草稿：模型标识行、搜索词与全选。
// 它不持有事实，也不发写请求；保存键没有通道，置灰写原因。
type configModelDraft struct {
	rows    []string
	query   string
	on      []bool
	formErr string
}

// configModelDraftOpen 打开「添加模型」对话框的空白草稿。
func configModelDraftOpen() *configModelDraft {
	return &configModelDraft{rows: []string{""}, on: []bool{true}}
}

// configModelDraftPanel 是「添加模型」对话框（S-configuration-70 / 71 / 72 / 75）：
// 计数与全选、添加一行与刷新、搜索、草稿行、空态、错误行与底部的保存。
func (a *App) configModelDraftPanel(c *gui.Context, data configView) {
	d := configScreen.draft
	if d == nil {
		return
	}
	t := c.Theme()
	open := true
	dialog := gui.Modal(c, &open, func() {
		screenDialogPanel(c, "添加模型", func() {
			// S-configuration-70 模型 · {visible}/{total}
			visible := d.visibleIndexes()
			label := fmt.Sprintf("模型 · %d / %d", d.enabledCount(), len(d.rows))
			gui.Row(c).AlignItems(gui.Center).Gap(configGapRow).Children(func() {
				state := gui.CheckOff
				switch {
				case d.enabledCount() == len(d.rows) && len(d.rows) > 0:
					state = gui.CheckOn
				case d.enabledCount() > 0:
					state = gui.CheckMixed
				}
				box := gui.CheckboxStateBase(c, &state).Size(configCheckbox, configCheckbox).
					Shrink(0).Radius(configBadgeRadius).Label("模型")
				if box.Changed() {
					next := state == gui.CheckOn
					for i := range d.on {
						d.on[i] = next
					}
				}
				_ = visible
				configMicroStyle.style(t, gui.Text(c, label)).TextColor(t.TextMuted).
					Label(label).SingleLine()
			})
			// S-configuration-71 添加模型 · 刷新
			gui.Row(c).AlignItems(gui.Center).Gap(configGapRow).Children(func() {
				if screenAction(c, iconPlus, "plus", "添加模型", rowCompact) {
					d.rows = append(d.rows, "")
					d.on = append(d.on, true)
				}
				// 协议不支持拉清单（或没有供应商）时刷新禁用并写原因。
				refreshDisabled := data.selected.id == "" || !data.selected.remoteModelList
				if configButton(c, "刷新", "刷新", refreshDisabled,
					noteProviderModelListUnavailable) {
					a.configProbeProvider(data.selected, probeModels)
				}
			})
			// S-configuration-72 搜索模型名称或 ID
			configSearchField(c, &d.query, "搜索模型名称或 ID", "搜索模型名称或 ID")
			if len(d.rows) == 0 {
				// S-configuration-75 还没有模型
				configMicroStyle.style(t,
					gui.Text(c, "还没有模型。可以手动添加，也可以在提供商支持模型列表时点击刷新。")).
					TextColor(t.TextMuted).MaxLines(2)
				return
			}
			if len(d.visibleIndexes()) == 0 {
				configMicroStyle.style(t, gui.Text(c, "没有匹配的模型")).
					TextColor(t.TextMuted).MaxLines(2)
				return
			}
			for _, i := range d.visibleIndexes() {
				index := i
				gui.Row(c).AlignItems(gui.Center).Gap(configGapRow).Children(func() {
					on := d.on[index]
					if gui.Checkbox(c, &on, fmt.Sprintf("模型 %d", index+1)).
						Label(fmt.Sprintf("模型 %d", index+1)).Changed() {
						d.on[index] = on
					}
					input := gui.TextInput(c, &d.rows[index]).
						Label(fmt.Sprintf("草稿模型 %d", index+1)).Placeholder("模型标识").
						Height(configFormInputH).Padding(configFormPadV, configFormPadH).
						Grow(1).MinWidth(0).Background(fieldFill(t))
					_ = input
					if screenIconButton(c, iconXMark, fmt.Sprintf("删除草稿模型 %d", index+1),
						"删除这一行", t.TextMuted) {
						d.removeAt(index)
					}
				})
			}
			if d.formErr != "" {
				// 保存试过一次：校验不过写字段原因，校验通过写「没有通道」。
				configRowStyle.style(t, gui.Text(c, d.formErr)).TextColor(t.Danger).
					Role(gui.RoleAlert).MaxLines(2)
			}
			gui.Row(c).Justify(gui.End).Gap(screenSpace2).Children(func() {
				if screenButton(c, "取消", "取消添加模型", "", false, rowRegular) {
					open = false
				}
				// 保存先做本地校验：填了合法标识也发不出写请求（模型的新建只有
				// provider.create 带清单，改已有供应商的模型没有方法），原因就地写。
				if gui.PrimaryButton(c, "保存").Label("保存").Height(rowRegular).Clicked() {
					if errText := d.validate(); errText != "" {
						d.formErr = errText
					} else {
						d.formErr = noteModelCreateWrite
					}
				}
			})
		})
	})
	if !open || dialog.Dismissed() {
		configScreen.draft = nil
	}
}

// visibleIndexes 是过滤后的草稿行下标（名称子串、不区分大小写）。
func (d *configModelDraft) visibleIndexes() []int {
	out := make([]int, 0, len(d.rows))
	for i, row := range d.rows {
		if configMatches(d.query, row) {
			out = append(out, i)
		}
	}
	return out
}

// enabledCount 是勾上的草稿行数。
func (d *configModelDraft) enabledCount() int {
	n := 0
	for _, on := range d.on {
		if on {
			n++
		}
	}
	return n
}

// removeAt 删掉一行草稿（勾选位一起删）。
func (d *configModelDraft) removeAt(index int) {
	if index < 0 || index >= len(d.rows) {
		return
	}
	d.rows = append(d.rows[:index], d.rows[index+1:]...)
	d.on = append(d.on[:index], d.on[index+1:]...)
}

// validate 校验草稿：标识必填、不能重复；返回第一条错误（空表示没问题）。
func (d *configModelDraft) validate() string {
	seen := make(map[string]bool, len(d.rows))
	for i, row := range d.rows {
		id := strings.TrimSpace(row)
		switch {
		case id == "":
			return fmt.Sprintf("第 %d 个模型缺少标识", i+1)
		case seen[id]:
			return "模型标识重复：" + id
		}
		seen[id] = true
	}
	return ""
}

// sortedDraftIDs 是草稿里的标识（去空、排序），给将来的写通道用。
func (d *configModelDraft) sortedDraftIDs() []string {
	out := make([]string, 0, len(d.rows))
	for _, row := range d.rows {
		if id := strings.TrimSpace(row); id != "" {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// ---- 加载骨架（S-configuration-81） ----

// configLoadingSkeleton 是读数还没到时的骨架：底色取次级面，行列宽高与真实项
// 一致，读屏只播一次「正在加载供应商」。
func configLoadingSkeleton(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Grow(1).Gap(configGapRow).Label("加载骨架").
		Role(gui.RoleStatus).Children(func() {
		configMicroStyle.style(t, gui.Text(c, "正在加载供应商")).TextColor(t.TextMuted).
			Label("正在加载供应商").SingleLine()
		for i := 0; i < 4; i++ {
			gui.Row(c).Shrink(0).Height(configRowH).AlignItems(gui.Center).
				Gap(configGapRow).Children(func() {
				box := gui.Box(c).Size(configIcon, configIcon).Shrink(0).
					Radius(configIcon / 2).Background(t.SurfaceHover).
					AccessHidden(true)
				_ = box
				gui.Box(c).Grow(1).Height(configIconTiny).Shrink(0).
					Radius(configBadgeRadius).Background(t.SurfaceHover).
					AccessHidden(true)
			})
		}
	})
}

// ---- 探测（测试连接 / 刷新） ----

// probeKind 是一次探测的用途：界面对结果的写法不同（行内结果 / 错误行）。
type probeKind int

const (
	probeTest probeKind = iota
	probeModels
)

// configProbeProvider 重新探测一个 provider（provider.probe，只读，不写配置）。
// 结果由下一帧的 ProviderRecords / Models 读数带回来；进行中两个入口都禁用。
func (a *App) configProbeProvider(provider configProvider, kind probeKind) {
	if a.intents == nil || provider.id == "" || configScreen.probing {
		return
	}
	id := provider.id
	configScreen.probing = true
	configScreen.probeErr = ""
	configScreen.probeNote = ""
	onFail := func(reason string) {
		configScreen.probing = false
		configScreen.probeErr = reason
	}
	a.dispatchThen("探测供应商", func(ctx context.Context) error {
		_, err := a.intents.ProbeProvider(ctx, id)
		return err
	}, func() {
		configScreen.probing = false
		configScreen.probeErr = ""
		if kind == probeTest {
			configScreen.probeNote = "连接正常"
		} else {
			configScreen.probeNote = "已刷新模型清单"
		}
	}, onFail)
}

// configFieldLabelW 是字段标签列宽：四个字（「基础 URL」占两档）排得下，值与
// 动作从同一列起。
const configFieldLabelW = 97

// configButton 是一个描边文字按钮：disabled 为真时整钮置灰、原因进 tooltip。
func configButton(c *gui.Context, text, label string, disabled bool, tip string) bool {
	t := c.Theme()
	b := gui.ButtonBase(c).Shrink(0).Height(rowCompact).Padding(controlPadV, controlPadH).
		Radius(screenRadiusCtl).Background(t.Background).Label(label).Tooltip(tip).
		Justify(gui.Center)
	b.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	if disabled {
		b = b.Disabled(true).Opacity(noteDisabledAlpha)
	} else {
		switch {
		case b.Pressed():
			b = b.Background(t.SurfacePressed)
		case b.Hovered():
			b = b.Background(t.SurfaceHover)
		}
	}
	ink := t.Text
	if disabled {
		ink = t.TextMuted
	}
	b.Children(func() {
		configRowStyle.style(t, gui.Text(c, text)).TextColor(ink).SingleLine()
	})
	return !disabled && b.Clicked()
}
