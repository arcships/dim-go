package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「从本地文件夹 / Git 仓库添加插件」弹窗（G-H1）与本地 manifest 的
// 预览。
//
// 流程：选目录 / 填路径 → 校验 dim-plugin.json → 预览条目（名称、版本、技能）
// → 添加插件 → 结果。服务端在 plugin.install 时还会再校验一次 manifest；这里
// 的校验只为了避免把明显不成立的东西发出去，并把要装的东西先摆出来。
//
// manifest 的口径取服务端 plugin_manager/record.rs：name 必填且不含控制字符；
// version 是可选的 x.y.z；dependencies 只接受 npm（且带 package.json 与
// package-lock.json）；code.entry / skills / mcp / hooks 必须是插件目录内的
// 相对路径。本端不读服务端的任何文件，只读用户刚选中的这个目录。

const (
	pluginInstallModeLocal = 0
	pluginInstallModeGit   = 1

	pluginDialogTitle = "从 Git 添加插件"
	pluginDialogDesc  = "从 GitHub 仓库或 Git URL 添加。"
	pluginModeLocal   = "本地文件夹"
	pluginModeGit     = "Git 仓库"
	pluginSourceLabel = "来源"
	pluginSourcePlace = "/path/to/plugin"
	pluginPickFolder  = "选择文件夹"
	pluginGitLabel    = "Git 仓库"
	pluginGitPlace    = "owner/repo 或 https://github.com/owner/repo"
	pluginRefLabel    = "Git ref"
	pluginRefPlace    = "main"
	pluginIDLabel     = "插件 id（可选）"
	pluginIDPlace     = "根据 manifest 自动生成"
	pluginEnableLabel = "安装后启用"
	pluginPreview     = "预览"
	pluginEmptySkills = "此插件不包含技能。"
	pluginAdd         = "添加插件"
	pluginAdding      = "正在安装…"
	pluginCancel      = "取消"
	pluginValidate    = "校验并预览"
	pluginPathEmpty   = "请选择插件文件夹。"
	pluginGitEmpty    = "请输入 Git 仓库。"
	pluginNoManifest  = "目录里没有 dim-plugin.json。"
	pluginManifestBad = "dim-plugin.json 无效："
	pluginSkillsEntry = "技能"
)

// pluginManifestPreview 是 manifest 校验通过后要上屏的内容。
type pluginManifestPreview struct {
	Path     string
	Name     string
	Version  string
	Skills   []string
	Code     bool
	MCP      bool
	Hooks    bool
	manifest pluginManifestFile
}

// pluginManifestFile 是 manifest 的界面所需字段。未知字段忽略：服务端会在安装
// 时按它自己的边界校验，界面只做与预览相关的检查。
type pluginManifestFile struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Description  string `json:"description"`
	Dependencies string `json:"dependencies"`
	Code         *struct {
		Entry string `json:"entry"`
	} `json:"code"`
	Skills string `json:"skills"`
	MCP    string `json:"mcp"`
	Hooks  string `json:"hooks"`
}

// pluginInstallState 是安装弹窗的可丢视图状态。
type pluginInstallState struct {
	open bool
	mode int
	path string
	git  string
	ref  string
	id   string
	// enable 是「安装后启用」，默认开。
	enable bool
	// preview / previewErr 是最近一次本地 manifest 校验的结果。
	preview    *pluginManifestPreview
	previewErr string
	installing bool
	err        string
}

// openPluginInstall 打开安装弹窗：默认本地文件夹、安装后启用。
func (a *App) openPluginInstall() {
	a.pluginInstall = pluginInstallState{open: true, enable: true}
}

// acceptPluginFolder 是目录选择落点：写进路径并立刻校验 manifest。
func (a *App) acceptPluginFolder(path string) {
	a.pluginInstall.path = path
	a.validatePluginFolder()
}

// validatePluginFolder 校验本地目录的 dim-plugin.json 并生成预览。
func (a *App) validatePluginFolder() {
	a.pluginInstall.preview = nil
	preview, err := readPluginManifest(a.pluginInstall.path)
	if err != nil {
		a.pluginInstall.previewErr = err.Error()
		return
	}
	a.pluginInstall.previewErr = ""
	a.pluginInstall.preview = preview
}

// pluginInstallProblem 返回「添加插件」被禁用的原因：空串表示可点。
func (a *App) pluginInstallProblem() string {
	st := &a.pluginInstall
	switch {
	case st.installing:
		return pluginAdding
	case a.wired() && a.intents == nil:
		return noteNoWriteChannel
	case st.mode == pluginInstallModeGit:
		if strings.TrimSpace(st.git) == "" {
			return pluginGitEmpty
		}
		return ""
	default:
		if strings.TrimSpace(st.path) == "" {
			return pluginPathEmpty
		}
		if st.previewErr != "" {
			return st.previewErr
		}
		if st.preview == nil {
			return "先校验本地 manifest"
		}
		return ""
	}
}

// confirmPluginInstall 发起一次 plugin.install。
func (a *App) confirmPluginInstall() {
	st := &a.pluginInstall
	if a.pluginInstallProblem() != "" {
		return
	}
	kind, ref := "local", strings.TrimSpace(st.path)
	if st.mode == pluginInstallModeGit {
		kind, ref = "git", strings.TrimSpace(st.git)
	}
	label := ref
	if st.mode == pluginInstallModeGit && strings.TrimSpace(st.ref) != "" {
		label = ref + "@" + strings.TrimSpace(st.ref)
	}
	if a.intents == nil {
		// 样例：没有写通道，明确拒绝而不是假装装上。
		st.err = noteNoWriteChannel
		return
	}
	id := strings.TrimSpace(st.id)
	gitRef := ""
	if st.mode == pluginInstallModeGit {
		gitRef = strings.TrimSpace(st.ref)
	}
	enable := st.enable
	st.installing = true
	st.err = ""
	a.dispatchQuiet("添加插件", func(ctx context.Context) error {
		return a.intents.InstallPlugin(ctx, kind, ref, gitRef, id, enable)
	}, func() {
		a.pluginInstall = pluginInstallState{}
		a.pluginManage.notice = fmt.Sprintf(pluginInstalledFmt, label)
	}, func(msg string) {
		a.pluginInstall.installing = false
		a.pluginInstall.err = msg
	})
}

// pluginInstallView 画安装弹窗（G-H1）。
func (a *App) pluginInstallView(c *gui.Context) {
	st := &a.pluginInstall
	if !st.open {
		return
	}
	t := c.Theme()
	open := true
	problem := a.pluginInstallProblem()
	m := modalShell(gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space3).Role(gui.RoleDialog).Label(pluginDialogTitle)
		panel.Children(func() {
			// G-H1-01 「从 Git 添加插件」
			typeDialogTitle.style(t, gui.Text(c, pluginDialogTitle)).TextColor(t.Text).
				SingleLine().Role(gui.RoleHeading).Level(2)
			// G-H1-02 「从 GitHub 仓库或 Git URL 添加。」
			typeBody.style(t, gui.Text(c, pluginDialogDesc)).TextColor(t.TextMuted)
			// G-H1-03 本地文件夹 / Git 仓库
			seg := gui.Segmented(c, &st.mode, pluginModeLocal, pluginModeGit).
				Label("安装来源")
			if st.installing {
				seg = seg.Disabled(true).Opacity(0.5)
			}
			if st.mode == pluginInstallModeLocal {
				gui.Column(c).Gap(space2).Children(func() {
					// G-H1-04 「来源」
					typeBodyStrong.style(t, gui.Text(c, pluginSourceLabel)).
						TextColor(t.Text).Label(pluginSourceLabel)
					gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
						gui.TextInput(c, &st.path).Label(pluginSourceLabel).
							Placeholder(pluginSourcePlace).Grow(1).Height(plgCtlH).
							Padding(screenInputPadV, screenSpace25).
							Disabled(st.installing)
						if accIconButton(c, accIconSpec{
							// G-H1-05 「选择文件夹」
							icon: iconFolderOpen, label: pluginPickFolder,
							tip: pluginPickFolder, size: plgIconSize, hit: plgCtlH,
							disabled: st.installing,
						}) {
							a.startDirPick(dirPickPluginFolder)
						}
					})
					gui.Row(c).Gap(space2).Children(func() {
						if pluginDialogButton(c, t, pluginValidate, false, false, st.installing, "") {
							a.validatePluginFolder()
						}
					})
					a.pluginManifestPreview(c, t)
				})
			} else {
				gui.Column(c).Gap(space2).Children(func() {
					// G-H1-06 「Git 仓库」+ 占位
					typeBodyStrong.style(t, gui.Text(c, pluginGitLabel)).
						TextColor(t.Text).Label(pluginGitLabel)
					gui.TextInput(c, &st.git).Label(pluginGitLabel).
						Placeholder(pluginGitPlace).Height(plgCtlH).
						Padding(screenInputPadV, screenSpace25).
						Disabled(st.installing)
					// G-H1-07 「Git ref」
					typeBodyStrong.style(t, gui.Text(c, pluginRefLabel)).
						TextColor(t.Text).Label(pluginRefLabel)
					gui.TextInput(c, &st.ref).Label(pluginRefLabel).
						Placeholder(pluginRefPlace).Height(plgCtlH).
						Padding(screenInputPadV, screenSpace25).
						Disabled(st.installing)
					typeSecondary.style(t, gui.Text(c, "Git 仓库的清单在安装时会先校验。")).
						TextColor(t.TextMuted)
				})
			}
			gui.Column(c).Gap(space2).Children(func() {
				// G-H1-08 「插件 id（可选）」
				typeBodyStrong.style(t, gui.Text(c, pluginIDLabel)).
					TextColor(t.Text).Label(pluginIDLabel)
				gui.TextInput(c, &st.id).Label(pluginIDLabel).
					Placeholder(pluginIDPlace).Height(plgCtlH).
					Padding(screenInputPadV, screenSpace25).
					Disabled(st.installing)
			})
			gui.Row(c).AlignItems(gui.Center).Gap(space2).Children(func() {
				// G-H1-09 「安装后启用」
				typeBody.style(t, gui.Text(c, pluginEnableLabel)).TextColor(t.Text).
					Label(pluginEnableLabel)
				gui.Spacer(c)
				gui.Switch(c, &st.enable).Size(32, 18).
					Radius(plgSwitchH / 2).Label(pluginEnableLabel).Disabled(st.installing)
			})
			// G-H1-10 请选择插件文件夹 / 请输入 Git 仓库 / 服务端错误
			if st.err != "" {
				typeSecondary.style(t, gui.Text(c, st.err)).TextColor(t.Danger).
					Role(gui.RoleAlert).Label(st.err)
			} else if st.mode == pluginInstallModeLocal && st.previewErr != "" &&
				strings.TrimSpace(st.path) != "" {
				typeSecondary.style(t, gui.Text(c, st.previewErr)).TextColor(t.Danger).
					Role(gui.RoleAlert).Label(st.previewErr)
			}
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				// G-H1-11 取消
				if pluginDialogButton(c, t, pluginCancel, false, false, st.installing, "") &&
					!st.installing {
					st.open = false
				}
				// G-H1-12 添加插件
				label := pluginAdd
				if st.installing {
					label = pluginAdding
				}
				if pluginDialogButton(c, t, label, true, st.installing, problem != "", problem) {
					a.confirmPluginInstall()
				}
			})
		})
		panel.Width(pluginInstallDialogWidth(c))
	}))

	if !open || m.Dismissed() {
		if st.installing {
			st.open = true
			return
		}
		// 取消不安装：整份状态清掉。
		a.pluginInstall = pluginInstallState{}
	}
}

// pluginManifestPreview 画本地 manifest 的预览（名称 / 版本 / 技能条目）。
func (a *App) pluginManifestPreview(c *gui.Context, t *gui.Theme) {
	st := &a.pluginInstall
	if st.preview == nil {
		return
	}
	p := st.preview
	lines := []string{pluginPreview + "：" + p.Name}
	if p.Version != "" {
		lines[0] += "　v" + p.Version
	}
	gui.Column(c).Shrink(0).Padding(space2, space2).Radius(radius).
		Border(strokeHairline, t.Border).Background(t.Surface).Children(func() {
		for _, line := range lines {
			typeSecondaryStrong.style(t, gui.Text(c, line)).TextColor(t.Text).
				Label(line)
		}
		if len(p.Skills) == 0 {
			typeSecondary.style(t, gui.Text(c, pluginEmptySkills)).
				TextColor(t.TextMuted).Label(pluginEmptySkills)
			return
		}
		typeSecondary.style(t, gui.Text(c, fmt.Sprintf("%s（%d）", pluginSkillsEntry, len(p.Skills)))).
			TextColor(t.TextMuted)
		for _, skill := range p.Skills {
			typeSecondary.style(t, gui.Text(c, "· "+skill)).TextColor(t.TextMuted).
				Label(skill)
		}
	})
}

// pluginDialogButton 是弹窗里的按钮：高 32；主按钮实心主色，次按钮描边。
func pluginDialogButton(c *gui.Context, t *gui.Theme, text string, primary, busy, disabled bool, tip string) bool {
	return pluginDialogButtonLabeled(c, t, text, text, primary, busy, disabled, tip)
}

// pluginDialogButtonLabeled 与 pluginDialogButton 相同，可另给读屏名（同一段
// 可见文字在不同弹窗里指不同动作时用）。
func pluginDialogButtonLabeled(c *gui.Context, t *gui.Theme, text, label string, primary, busy, disabled bool, tip string) bool {
	b := gui.ButtonBase(c).Shrink(0).Height(plgCtlH).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Label(label).Background(t.Background)
	if primary {
		b = b.Background(t.Accent)
	}
	if tip != "" {
		b = b.Tooltip(tip)
	}
	if disabled {
		b = b.Disabled(true)
	}
	switch {
	case disabled:
		b = b.Opacity(0.5)
	case !primary && b.Pressed():
		b = b.Background(t.SurfacePressed)
	case !primary && b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	textColor := t.Text
	if primary {
		textColor = t.AccentText
	}
	b.Children(func() {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
			if busy {
				gui.Spinner(c).Size(plgIconSize, plgIconSize)
			}
			plgBtnTextSty.style(t, gui.Text(c, text)).TextColor(textColor).SingleLine()
		})
	})
	return b.Clicked()
}

// pluginInstallDialogWidth 是安装弹窗的内容宽：参考 sm:max-w-[30rem] = 420。
func pluginInstallDialogWidth(c *gui.Context) float32 {
	return dialogWidth(c, 385)
}

// readPluginManifest 读并校验一个插件目录的 dim-plugin.json，生成预览。
func readPluginManifest(rawDir string) (*pluginManifestPreview, error) {
	dir := strings.TrimSpace(rawDir)
	if dir == "" {
		return nil, errors.New(pluginPathEmpty)
	}
	info, err := dirStat(dir)
	if err != nil {
		return nil, fmt.Errorf("读不到插件目录：%s", dir)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("这不是目录：%s", dir)
	}
	manifestPath := filepath.Join(dir, "dim-plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, errors.New(pluginNoManifest)
	}
	var file pluginManifestFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("%s%s", pluginManifestBad, err)
	}
	if strings.TrimSpace(file.Name) == "" || strings.IndexFunc(file.Name, unicode.IsControl) >= 0 {
		return nil, fmt.Errorf("%sname 必须是非空名称", pluginManifestBad)
	}
	if file.Version != "" && !pluginVersionOK(file.Version) {
		return nil, fmt.Errorf("%sversion 必须是三段数字（如 1.2.0）", pluginManifestBad)
	}
	if file.Dependencies != "" && file.Dependencies != "npm" {
		return nil, fmt.Errorf("%sdependencies 只接受 npm", pluginManifestBad)
	}
	if file.Dependencies == "npm" {
		for _, name := range []string{"package.json", "package-lock.json"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				return nil, fmt.Errorf("%sdependencies 需要同目录的 %s", pluginManifestBad, name)
			}
		}
	}
	preview := &pluginManifestPreview{Path: dir, Name: file.Name, Version: file.Version, manifest: file}
	if file.Code != nil {
		entry, err := pluginRelativePath(dir, file.Code.Entry, "code.entry")
		if err != nil {
			return nil, fmt.Errorf("%s%s", pluginManifestBad, err)
		}
		if info, err := os.Stat(entry); err != nil || info.IsDir() {
			return nil, fmt.Errorf("%scode.entry 不是一个文件", pluginManifestBad)
		}
		preview.Code = true
	}
	if file.Skills != "" {
		skillsDir, err := pluginRelativePath(dir, file.Skills, "skills")
		if err != nil {
			return nil, fmt.Errorf("%s%s", pluginManifestBad, err)
		}
		preview.Skills = pluginSkillNames(skillsDir)
	}
	if file.MCP != "" {
		if _, err := pluginRelativePath(dir, file.MCP, "mcp"); err != nil {
			return nil, fmt.Errorf("%s%s", pluginManifestBad, err)
		}
		preview.MCP = true
	}
	if file.Hooks != "" {
		if _, err := pluginRelativePath(dir, file.Hooks, "hooks"); err != nil {
			return nil, fmt.Errorf("%s%s", pluginManifestBad, err)
		}
		preview.Hooks = true
	}
	return preview, nil
}

// pluginRelativePath 把 manifest 里的字段解析成插件目录内的路径：不接受空、
// 绝对路径与跳出目录的写法（与服务端的边界一致）。
func pluginRelativePath(dir, relative, field string) (string, error) {
	relative = strings.TrimSpace(relative)
	if relative == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("%s 必须是目录内的相对路径", field)
	}
	resolved := filepath.Clean(filepath.Join(dir, relative))
	base := filepath.Clean(dir)
	if resolved != base && !strings.HasPrefix(resolved, base+string(filepath.Separator)) {
		return "", fmt.Errorf("%s 不能跳出插件目录", field)
	}
	return resolved, nil
}

// pluginVersionOK 判三段数字的版本号。
func pluginVersionOK(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// pluginSkillNames 列出技能目录里的条目：`*.md` 取文件名，含 SKILL.md 的子目录
// 取目录名。读不到目录时返回空（预览写「此插件不包含技能」）。
func pluginSkillNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			if _, err := os.Stat(filepath.Join(dir, name, "SKILL.md")); err == nil {
				out = append(out, name)
			}
			continue
		}
		if strings.EqualFold(filepath.Ext(name), ".md") {
			out = append(out, strings.TrimSuffix(name, filepath.Ext(name)))
		}
	}
	sort.Strings(out)
	return out
}
