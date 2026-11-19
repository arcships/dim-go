package ui

// 本文件是目录选择的界面侧：macOS 弹系统选择器，其他平台与失败时退回同一个
// 小框手输路径。能力本体在 internal/dirpick，这里只做三件事：
//   - 在后台协程里跑选择命令（View 每帧重建，不能等用户点完对话框）；
//   - 取消不改任何状态，失败留一句可读原因；
//   - 选中的目录写进对应位置并落盘本地偏好（Preferences）：左栏多一组工作区，
//     Hooks 的项目来源卡换成这个目录。
//
// 两处入口：左栏「添加工作区」（sidebar.go 的 addWorkspace）与设置 › Hooks 的
// 项目来源卡（settings_hooks_screen.go 的 hkSourceCard）。非 macOS 没有系统
// 选择器：同一个入口改成手输路径，平台差异只在这里出现。

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/dirpick"
)

// dirPickTarget 是目录选择的落点。
type dirPickTarget int

const (
	// dirPickWorkspace 是左栏「添加工作区」。
	dirPickWorkspace dirPickTarget = iota
	// dirPickHooksProject 是设置 › Hooks 的项目目录。
	dirPickHooksProject
	// dirPickPluginFolder 是添加插件弹窗的本地文件夹。
	dirPickPluginFolder
)

// dirPickState 是一处入口的界面状态（可丢，不是事实）。
type dirPickState struct {
	// open 是手输 / 失败原因的小框开着；manual 为真表示框里是输入位。
	open   bool
	manual bool
	draft  string
	// failure 是一句可读原因；空表示没有失败。
	failure string
	// busy 为真表示系统选择器开着、后台命令还没回来。
	busy bool
}

// dirPickOutcome 是后台选择交回渲染线程的一次结果。
type dirPickOutcome struct {
	target dirPickTarget
	path   string
	err    error
}

// dirPickTimeout 是一次系统目录选择的等待上限：超时按失败留原因。
const dirPickTimeout = 5 * time.Minute

// 本文件的界面语。
const (
	dirPickCopyWorkspaceTitle = "添加工作区"
	dirPickCopyWorkspaceField = "工作区目录"
	dirPickCopyHooksTitle     = "选择项目目录"
	dirPickCopyHooksField     = "项目目录"
	dirPickCopyManualHint     = "系统目录选择器只在 macOS 上可用，这里直接输入目录路径。"
	dirPickCopyPlaceholder    = "/path/to/dir"
	dirPickCopyCancel         = "取消"
	dirPickCopyCancelLabel    = "取消选择目录"
	dirPickCopyConfirm        = "添加"
	dirPickCopyConfirmLabel   = "使用这个目录"
	dirPickCopyRetry          = "重新选择"
	dirPickCopyRetryLabel     = "重新选择目录"
	dirPickCopyEmptyDraft     = "没有填写目录路径"
)

// pickState 取一处入口的界面状态。
func (a *App) pickState(target dirPickTarget) *dirPickState {
	switch target {
	case dirPickHooksProject:
		return &a.hooksPick
	case dirPickPluginFolder:
		return &a.pluginPick
	}
	return &a.addWorkspacePick
}

// dirPickPlatform 是这次选择的平台；测试注入，空表示真实平台。
func (a *App) dirPickPlatform() string {
	if a.dirPickGOOS != "" {
		return a.dirPickGOOS
	}
	return runtime.GOOS
}

// dirPickRunnerFor 是这次选择用的执行器；测试注入，nil 表示真实命令。
func (a *App) dirPickRunnerFor() dirpick.Runner {
	if a.dirPickRunner != nil {
		return a.dirPickRunner
	}
	return dirpick.Run
}

// startDirPick 打开一处入口的目录选择：macOS 弹系统选择器，其他平台直接开手输框。
// 选择命令在后台协程里跑，结果经 pickResults 交回渲染线程（见 applyDirPickResults）。
func (a *App) startDirPick(target dirPickTarget) {
	st := a.pickState(target)
	if st.busy {
		return
	}
	st.failure = ""
	goos := a.dirPickPlatform()
	if goos != "darwin" {
		st.open, st.manual = true, true
		return
	}
	st.busy = true
	run := a.dirPickRunnerFor()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), dirPickTimeout)
		defer cancel()
		path, err := dirpick.Choose(ctx, run, goos)
		a.pickMu.Lock()
		a.pickResults = append(a.pickResults, dirPickOutcome{target: target, path: path, err: err})
		a.pickMu.Unlock()
		if a.changed != nil {
			a.changed()
		}
	}()
}

// applyDirPickResults 在渲染线程上把后台选择的结果落到界面状态。
func (a *App) applyDirPickResults() {
	a.pickMu.Lock()
	results := a.pickResults
	a.pickResults = nil
	a.pickMu.Unlock()
	for _, result := range results {
		a.applyDirPick(result)
	}
}

// applyDirPick 落一次结果：取消不改状态，不支持与失败留原因，成功写进对应位置。
func (a *App) applyDirPick(result dirPickOutcome) {
	st := a.pickState(result.target)
	st.busy = false
	switch {
	case errors.Is(result.err, dirpick.ErrCancelled):
		// 取消不是失败：什么都不改，也不报错。
		return
	case errors.Is(result.err, dirpick.ErrUnsupported):
		st.open, st.manual = true, true
		return
	case result.err != nil:
		st.open, st.manual, st.failure = true, false, result.err.Error()
		return
	}
	a.acceptDir(result.target, result.path, st)
}

// acceptDir 校验并落下一个目录路径：不可读时留在小框里写原因，成功写进对应位置
// 并落盘本地偏好。
func (a *App) acceptDir(target dirPickTarget, raw string, st *dirPickState) {
	path, err := checkDirPath(raw)
	if err != nil {
		st.open, st.failure = true, err.Error()
		return
	}
	st.open, st.manual, st.failure = false, false, ""
	switch target {
	case dirPickHooksProject:
		a.hooksProjectPath = path
	case dirPickPluginFolder:
		a.acceptPluginFolder(path)
		return
	default:
		a.addWorkspacePath(path)
	}
	a.persistPreferences()
}

// checkDirPath 判选中的路径是不是一个可读目录，返回去空白后的路径。
// 只有可读目录才登记成工作区（R-LB-169：不新增无法展开的空分组）。
func checkDirPath(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", errors.New(dirPickCopyEmptyDraft)
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("读不到用户主目录：%v", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	info, err := dirStat(path)
	if err != nil {
		switch {
		case errors.Is(err, os.ErrPermission):
			return "", fmt.Errorf("没有权限读取目录：%s", path)
		case errors.Is(err, os.ErrNotExist):
			return "", fmt.Errorf("目录不存在：%s", path)
		}
		return "", fmt.Errorf("目录不可读：%s", path)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("这不是目录：%s", path)
	}
	return path, nil
}

// dirStat 是路径检查的可注入入口：测试用它模拟权限不足，不碰真实文件系统。
var dirStat = os.Stat

// dirPickDialog 画目录选择的小框：手输时给输入位与「添加」，系统选择器失败时
// 给原因与「重新选择」。
func (a *App) dirPickDialog(c *gui.Context, target dirPickTarget) {
	st := a.pickState(target)
	if !st.open {
		return
	}
	t := c.Theme()
	title, field := dirPickCopyWorkspaceTitle, dirPickCopyWorkspaceField
	switch target {
	case dirPickHooksProject:
		title, field = dirPickCopyHooksTitle, dirPickCopyHooksField
	case dirPickPluginFolder:
		title, field = pluginManageAdd, pluginSourceLabel
	}
	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space2).Role(gui.RoleAlertDialog).Label(title)
		panel.Children(func() {
			typeH3.style(t, gui.Text(c, title)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(3).SingleLine()
			if st.manual {
				screenHint(c, dirPickCopyManualHint)
				input := gui.TextInputBase(c, &st.draft).Label(field).
					Placeholder(dirPickCopyPlaceholder)
				input.BorderWidth(strokeHairline).BorderColor(t.Border)
			}
			if st.failure != "" {
				typeSecondary.style(t, gui.Text(c, st.failure)).TextColor(t.Danger).
					Role(gui.RoleAlert).Label(st.failure)
			}
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				if screenButton(c, dirPickCopyCancel, dirPickCopyCancelLabel, "", false, rowRegular) {
					st.open, st.failure = false, ""
					open = false
				}
				if st.manual {
					if screenButton(c, dirPickCopyConfirm, dirPickCopyConfirmLabel, "", true, rowRegular) {
						a.acceptDir(target, st.draft, st)
					}
					return
				}
				if st.failure != "" {
					if screenButton(c, dirPickCopyRetry, dirPickCopyRetryLabel, "", false, rowRegular) {
						st.open, st.failure = false, ""
						open = false
						a.startDirPick(target)
					}
				}
			})
		})
		panel.Width(accDialogWidth(c))
	}))

	if !open || m.Dismissed() {
		st.open, st.failure = false, ""
	}
}
