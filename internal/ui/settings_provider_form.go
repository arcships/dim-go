package ui

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 模型供应里的供应商表单：新增与编辑共用同一套字段，对应参考
// 产品的「添加自定义提供商」面板（S-configuration-64 … 77）与编辑态
// （S-configuration-55 … 60）。字段按本仓库能写的事实收口：
//
//	名称       必填；不能与其他供应商重名（不区分大小写、去首尾空白）。
//	基础 URL   必填；形态必须是 http / https 的绝对地址。
//	API 密钥   新增必填、密码型输入（不回显明文，保存后也不回显）；编辑时留空
//	           表示不改（读不到已存密钥，界面不摆假的掩码值）。
//	模型清单   只收模型标识，可加行 / 删行；新增时随 provider.create 一起下发。
//
// 没有进表单的两项：参考的「协议」（5 个选项）与「User-Agent」——当前的写通道
// （Intents 的 CreateProvider / UpdateProvider）不带这两个字段，驱动固定 openai、
// 认证固定 bearer；摆出来也存不下去，所以不画（P8 不虚构）。
//
// 编辑态的模型清单只读：provider.update 的补丁里没有模型，模型的新建也没有对应
// 方法（只有启停与改字段）。可加删却存不下去就是假入口，所以编辑时只列已有模型，
// 新增时才是可编辑的行。模型的新增仍走参考里的「添加模型」对话框（S-78，未重建）。
//
// 表单错误就地显示：字段级错误在框下（S-47 / S-66 / S-68），保存失败的原因在底部
// formError 行（S-48）。错误文案带得出原因，不写裸 code。

const (
	// 表单几何：字段标签与输入框 gap-1（3.5）、字段之间 gap-4（14）、
	// 输入框高 32（design-baseline §3）、模型行高 32。
	configFormFieldGap  = space1
	configFormFieldGapV = space4
	configFormInputH    = screenControlH
	configFormPadV      = screenInputPadV
	configFormPadH      = screenSpace25
)

// configForm 是新增 / 编辑表单的草稿。它全是可丢的界面状态：不持有事实，
// 也不缓存服务端回执。
type configForm struct {
	// editing 为假是新增；为真时 id / key 指向被编辑的供应商。
	editing bool
	id      string
	key     string

	name    string
	baseURL string
	apiKey  string
	// models 是模型标识草稿（一行一个）。
	models []string

	nameErr    string
	baseURLErr string
	apiKeyErr  string
	modelsErr  string
	// formErr 是保存失败的原因（S-configuration-48）。
	formErr string
	// saving 为真时保存按钮禁用，等后台写入回执。
	saving bool
}

// configFormTitle 是面板标题（S-configuration-64）。
func (f *configForm) title() string {
	if f.editing {
		return "编辑供应商"
	}
	return "添加自定义提供商"
}

// configFormNew 是「添加」打开的空白草稿。
func configFormNew() *configForm { return &configForm{} }

// configFormEdit 是编辑一个供应商的草稿：名称、基础 URL 与模型清单按读数预填；
// API 密钥读不到（回执里只有「是否已配置」），留空表示不改。
func configFormEdit(provider configProvider, models []configModel) *configForm {
	form := &configForm{
		editing: true,
		id:      provider.id,
		key:     configKey(provider),
		name:    provider.name,
		baseURL: provider.baseURL,
	}
	for _, model := range models {
		id := model.id
		if id == "" {
			id = model.name
		}
		form.models = append(form.models, id)
	}
	return form
}

// validate 校验草稿，把错误写回字段（空表示该字段没问题）。
func (f *configForm) validate(providers []configProvider) bool {
	f.nameErr, f.baseURLErr, f.apiKeyErr, f.modelsErr = "", "", "", ""
	name := strings.TrimSpace(f.name)
	switch {
	case name == "":
		f.nameErr = "请填写名称"
	case configNameTaken(providers, name, f.key):
		f.nameErr = "名称已被其他供应商使用"
	}
	base := strings.TrimSpace(f.baseURL)
	switch {
	case base == "":
		f.baseURLErr = "请填写基础 URL"
	case !configValidBaseURL(base):
		f.baseURLErr = "基础 URL 需以 http:// 或 https:// 开头"
	}
	if !f.editing && strings.TrimSpace(f.apiKey) == "" {
		f.apiKeyErr = "请填写 API 密钥"
	}
	seen := make(map[string]bool, len(f.models))
	for index, model := range f.models {
		id := strings.TrimSpace(model)
		switch {
		case id == "":
			f.modelsErr = fmt.Sprintf("第 %d 个模型缺少标识", index+1)
		case seen[id]:
			f.modelsErr = "模型标识重复：" + id
		}
		if f.modelsErr != "" {
			break
		}
		seen[id] = true
	}
	return f.nameErr == "" && f.baseURLErr == "" && f.apiKeyErr == "" && f.modelsErr == ""
}

// drafts 把模型行收成下发用的条目（去掉空行，按输入顺序）。
func (f *configForm) drafts() []ProviderModelDraft {
	out := make([]ProviderModelDraft, 0, len(f.models))
	for _, model := range f.models {
		if id := strings.TrimSpace(model); id != "" {
			out = append(out, ProviderModelDraft{ID: id})
		}
	}
	return out
}

// configNameTaken 是重名校验：同名的另一家供应商（编辑时跳过自己）。
func configNameTaken(providers []configProvider, name, selfKey string) bool {
	for _, provider := range providers {
		if configKey(provider) == selfKey {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(provider.name), name) {
			return true
		}
	}
	return false
}

// configValidBaseURL 要求 http / https 的绝对地址（参考的 URL 形态判据）。
func configValidBaseURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}

// configFormClose 关掉表单回到详情态。
func configFormClose() { configScreen.form = nil }

// configFormSave 保存草稿：校验不过就不发请求；新增走 CreateProvider、编辑走
// UpdateProvider，失败原因回到表单底部，成功回到详情并选中该供应商。
func (a *App) configFormSave(data configView, form *configForm) {
	if form.saving {
		return
	}
	if !form.validate(data.providers) {
		return
	}
	name := strings.TrimSpace(form.name)
	baseURL := strings.TrimSpace(form.baseURL)
	apiKey := strings.TrimSpace(form.apiKey)
	if a.intents == nil {
		form.formErr = "未连接 App Server，无法保存"
		return
	}
	onFail := func(reason string) {
		if configScreen.form != form {
			return
		}
		form.saving = false
		form.formErr = reason
	}
	form.formErr = ""
	form.saving = true
	if form.editing {
		if form.id == "" {
			form.saving = false
			form.formErr = "这个供应商没有服务端标识，无法保存"
			return
		}
		id, key := form.id, form.key
		a.dispatchThen("保存供应商", func(ctx context.Context) error {
			return a.intents.UpdateProvider(ctx, id, name, baseURL, apiKey)
		}, func() {
			if configScreen.form == form {
				configFormClose()
			}
			configScreen.selected = key
		}, onFail)
		return
	}
	models := form.drafts()
	a.dispatchThen("新建供应商", func(ctx context.Context) error {
		return a.intents.CreateProvider(ctx, "", name, baseURL, apiKey, models)
	}, func() {
		if configScreen.form == form {
			configFormClose()
		}
		// 新建回执不带标识：下一帧的读数里按名字找到它再选中。
		configScreen.selectAfterCreate = name
	}, onFail)
}

// configFormPanel 是右栏的表单：标题 + 字段 + 模型清单 + 错误行 + 底部动作。
func (a *App) configFormPanel(c *gui.Context, data configView) {
	t := c.Theme()
	form := configScreen.form
	gui.Column(c).Width(configDetailContentWidth(c)).Grow(1).MinHeight(0).Children(func() {
		gui.Scroll(c).Grow(1).Label(form.title()).Children(func() {
			gui.Column(c).Shrink(0).Gap(configFormFieldGapV).Children(func() {
				typeH3.style(t, gui.Text(c, form.title())).TextColor(t.Text).
					Role(gui.RoleHeading).Level(3).SingleLine()
				// S-configuration-66 名称
				configFormField(c, &form.name, "名称", "我的自定义提供商", form.nameErr, false)
				// S-configuration-67 基础 URL
				configFormField(c, &form.baseURL, "基础 URL", "https://api.example.com/v1",
					form.baseURLErr, false)
				apiKeyPlaceholder := "sk-..."
				if form.editing {
					apiKeyPlaceholder = "留空则不改动已保存的密钥"
				}
				// S-configuration-68 API 密钥
				configFormField(c, &form.apiKey, "API 密钥", apiKeyPlaceholder,
					form.apiKeyErr, true)
				a.configFormModels(c, data, form)
				a.configFormFooter(c, data, form)
			})
		})
	})
}

// configFormField 是一个字段：标签 + 高 32 的输入框 + 框下错误行。
func configFormField(c *gui.Context, value *string, label, placeholder, errText string, password bool) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(configFormFieldGap).Children(func() {
		configRowStyle.style(t, gui.Text(c, label)).TextColor(t.Text).Label(label).SingleLine()
		input := gui.TextInput(c, value).Label(label).Placeholder(placeholder).
			Height(configFormInputH).Padding(configFormPadV, configFormPadH).
			Background(fieldFill(t))
		if password {
			// 密钥是密码型输入：屏幕上只有掩码，读屏也不回读内容。
			input = input.Password()
		}
		if errText != "" {
			// 错误文案就是屏幕上要读的那一行：不另给它 Label，否则 Label 会顶掉
			// 显示的文字（Text 的 Label 就是它的内容）。
			configRowStyle.style(t, gui.Text(c, errText)).TextColor(t.Danger).
				Role(gui.RoleAlert).MaxLines(2)
		}
	})
}

// configFormModels 是模型清单：新增时可加 / 删行，编辑时只读（见文件头）。
func (a *App) configFormModels(c *gui.Context, data configView, form *configForm) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(configFormFieldGap).Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(configGapRow).Children(func() {
			configRowStyle.style(t, gui.Text(c, "模型清单")).TextColor(t.Text).
				Label("模型清单").SingleLine()
			gui.Spacer(c)
			if form.editing {
				return
			}
			// S-configuration-71 添加模型
			if screenAction(c, iconPlus, "plus", "添加模型", rowRegular) {
				form.models = append(form.models, "")
			}
		})
		hint := "每行一个模型标识；新建后可在详情里启停。"
		if form.editing {
			hint = "模型在详情里管理：这里只列已有模型。"
		}
		configMicroStyle.style(t, gui.Text(c, hint)).TextColor(t.TextMuted).MaxLines(2)
		if len(form.models) == 0 {
			// S-configuration-75 还没有模型
			configMicroStyle.style(t, gui.Text(c, "还没有模型。可以手动添加，也可以留空。")).
				TextColor(t.TextMuted).MaxLines(2)
		}
		for index := range form.models {
			configFormModelRow(c, form, index, !form.editing)
		}
		if form.modelsErr != "" {
			configRowStyle.style(t, gui.Text(c, form.modelsErr)).TextColor(t.Danger).
				Role(gui.RoleAlert).MaxLines(2)
		}
	})
}

// configFormModelRow 是一行模型：标识输入 + 删除（新增时）。
func configFormModelRow(c *gui.Context, form *configForm, index int, editable bool) {
	t := c.Theme()
	label := fmt.Sprintf("模型 %d", index+1)
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(configGapRow).Children(func() {
		input := gui.TextInput(c, &form.models[index]).Label(label).
			Placeholder("模型标识").Height(configFormInputH).
			Padding(configFormPadV, configFormPadH).Grow(1).MinWidth(0).
			Background(fieldFill(t))
		if !editable {
			// 只读而不是禁用：值仍可选中复制、读屏也读得到；改模型没有写通道。
			input = input.ReadOnly(true)
		}
		if editable {
			if screenIconButton(c, iconXMark, "删除模型 "+fmt.Sprint(index+1),
				"删除这一行", t.TextMuted) {
				form.models = append(form.models[:index], form.models[index+1:]...)
			}
		}
	})
}

// configFormFooter 是底部动作（S-configuration-77）：取消 / 保存，失败原因在
// 动作上方（S-configuration-48 的 formError）。
func (a *App) configFormFooter(c *gui.Context, data configView, form *configForm) {
	t := c.Theme()
	// S-configuration-76 formError 行
	if form.formErr != "" {
		configRowStyle.style(t, gui.Text(c, form.formErr)).TextColor(t.Danger).
			Role(gui.RoleAlert).MaxLines(3)
	}
	gui.Row(c).Shrink(0).Justify(gui.End).Gap(configGapRow).Children(func() {
		// S-configuration-77 取消
		if screenButton(c, "取消", form.title()+"：取消", "", false, rowRegular) {
			configFormClose()
		}
		// S-configuration-77 保存
		save := gui.PrimaryButton(c, "保存").Label("保存").Height(rowRegular).
			Disabled(form.saving)
		if save.Clicked() {
			a.configFormSave(data, form)
		}
	})
}
