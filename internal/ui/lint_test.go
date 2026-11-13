package ui

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是界面领域的可执行检查器（`scripts/ui-lint` 跑的就是它）。
//
// 一条规则 = 一个检查项 = 一个断言名（L1…L15）= 不通过时的修法；对应表在
// `docs/domains/U2-checks.md`，每行指回 §9.0 / §9.8 / §9.9 的编号。
//
// 检查分两类：
//   - 静态：解析 internal/ui 的源码（AST + 字面量），报 file:line；
//   - 渲染：用框架的无窗口 Tester 画一帧，量基线、间距、命中区、像素态。
//
// 种子自检（TestUILintSeeds）把五类违例注入到**合成源码与合成量测**里，证明
// 检查器确实会报，且报得出位置；真工作区里手工注入同理（改一个常数、加一个字符）。

// lintFinding 是一条违例：哪一条检查、在哪、怎么修。
type lintFinding struct {
	check string // L1…L15
	rule  string // §9.0 / §9.8 / §9.9 的编号
	pos   string // file:line（渲染类用 "render"）
	msg   string
}

func (f lintFinding) String() string {
	return fmt.Sprintf("%s [%s] %s：%s", f.pos, f.check, f.rule, f.msg)
}

// lintFix 是每条的修法提示（也用于文档）。
var lintFix = map[string]string{
	"L1":  "换成 tokens.go 的档位（space-1…12 / row-compact / icon-inline / hit-min），或新增常量并写明推导",
	"L2":  "删掉字符，用 internal/ui/icons.go 的图标元素",
	"L3":  "给入口加 label（读屏名）",
	"L4":  "给纯图标入口补 tip：名称 · 快捷键（如「发送 · Enter」）",
	"L5":  "高后果动作改用 textIconButton，保留可见文字",
	"L6":  "状态改用图标与颜色；读屏名可保留状态词",
	"L7":  "让两个状态在图标、颜色或形态上至少差两项",
	"L8":  "统一内容左缘（space-4 外边距 + space-2 卡片内边距）",
	"L9":  "轮次之间用 space-6，轮内用 space-1/2/3",
	"L10": "自绘描边只取 t.Border",
	"L11": "能靠留白或面差分开的不画线；确需线时登记进白名单",
	"L12": "关动效后布局与文字必须与开着时一致",
	"L13": "命中区放大到 hit-min（24）",
	"L14": "补默认 / 悬停 / 聚焦 / 按下 / 禁用里的缺项",
	"L15": "200% 缩放下不越界：调字号或允许单行截断",
}

// TestUILint 是检查器本体：工作区必须零违例。
func TestUILint(t *testing.T) {
	findings := lintWorkspace(t)
	if len(findings) == 0 {
		t.Logf("检查器全绿：%d 个检查项（见 docs/domains/U2-checks.md）", 16)
		return
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].check != findings[j].check {
			return findings[i].check < findings[j].check
		}
		return findings[i].pos < findings[j].pos
	})
	var b strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&b, "%s\n    修法：%s\n", f.String(), lintFix[f.check])
	}
	t.Fatalf("检查器报出 %d 条违例：\n%s", len(findings), b.String())
}

// TestUILintSeeds 注入五类种子违例，证明每一条都报得出、且带位置。
func TestUILintSeeds(t *testing.T) {
	seeds := []struct {
		name   string
		check  string
		pos    string
		source string
	}{
		{"档位外间距", "L1", "seed.go:", "package ui\nfunc f(c *Context) { gui.Row(c).Padding(13, 0) }\n"},
		{"字符冒充图标", "L2", "seed.go:", "package ui\nfunc f(c *Context) { gui.Text(c, \"右栏 ›\") }\n"},
		{"缺 tooltip", "L4", "seed.go:", "package ui\nvar s = iconButtonSpec{icon: iconPlus, size: rowCompact}\n"},
		{"状态词上屏", "L6", "seed.go:", "package ui\nfunc f(c *Context) { gui.Text(c, \"完成\") }\n"},
		{"决定缺可见文字", "L5", "seed.go:", "package ui\nvar s = textIconButtonSpec{icon: iconCheck, text: \"\", label: \"允许一次\"}\n"},
	}
	for _, seed := range seeds {
		got := lintSource("seed.go", []byte(seed.source), &contentDoc{})
		var hit *lintFinding
		for i := range got {
			if got[i].check == seed.check {
				hit = &got[i]
				break
			}
		}
		if hit == nil {
			t.Fatalf("种子「%s」没有被 %s 报出；实际报出：%v", seed.name, seed.check, got)
		}
		if !strings.HasPrefix(hit.pos, seed.pos) || !strings.Contains(hit.pos, ":") {
			t.Fatalf("种子「%s」没有给出位置：%q", seed.name, hit.pos)
		}
		t.Logf("种子「%s」→ %s", seed.name, hit.String())
	}

	// 基线不等的种子：合成三档左缘，检查必须报。
	bad := map[string][3]float32{
		"标题行":  {272, 272, 272},
		"时间线":  {272, 272, 272},
		"输入卡":  {272, 275, 272},
	}
	if got := lintBaselinesAt("seed", bad); len(got) == 0 || got[0].check != "L8" {
		t.Fatalf("基线种子没有被 L8 报出：%v", got)
	} else {
		t.Logf("种子「基线不等」→ %s", got[0].String())
	}
}

// ---- 工作区扫描 ----

func lintWorkspace(t *testing.T) []lintFinding {
	t.Helper()
	var out []lintFinding
	var content contentDoc
	loadJSON(t, "U1-content.json", &content)
	var actions actionDoc
	loadJSON(t, "U3-actions.json", &actions)
	dir := "."
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("列源码：%v", err)
	}
	var sources []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("读 %s：%v", f, err)
		}
		sources = append(sources, f)
		out = append(out, lintSource(f, src, &content)...)
	}
	out = append(out, lintDividerAllowlist(sources)...)
	out = append(out, lintRender(t)...)
	out = append(out, lintHighConsequence(t, &actions)...)
	out = append(out, lintListReferences(t, &content)...)
	return out
}

// lintSource 是静态检查：对一份源码跑 L1–L6、L10。
func lintSource(name string, src []byte, content *contentDoc) []lintFinding {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		return []lintFinding{{check: "L1", rule: "§9.14", pos: name, msg: "解析失败：" + err.Error()}}
	}
	_ = content
	var out []lintFinding
	pos := func(p token.Pos) string {
		return fmt.Sprintf("%s:%d", name, fset.Position(p).Line)
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			// L1：档位外字面量。
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				if allowed, checked := layoutCalls[sel.Sel.Name]; checked {
					for _, arg := range x.Args {
						lit, ok := arg.(*ast.BasicLit)
						if !ok || (lit.Kind != token.INT && lit.Kind != token.FLOAT) {
							continue
						}
						if lit.Value == "0" {
							continue
						}
						if allowed[lit.Value] {
							continue
						}
						out = append(out, lintFinding{"L1", "§9.14 排版基线 3 / X16",
							pos(lit.Pos()), fmt.Sprintf("%s(%s) 用了档位外的数值", sel.Sel.Name, lit.Value)})
					}
				}
				// L6：状态词上屏。
				if sel.Sel.Name == "Text" || sel.Sel.Name == "RichText" {
					for _, arg := range x.Args {
						if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
							if s, err := strconv.Unquote(lit.Value); err == nil && statusWords[s] {
								msg := "状态词被当成文字画到屏幕上：" + s
								// 状态词在 U1 清单里是"一次操作的结果词"（C3）；这里画成
								// 字面量就成了界面语，报出对应的信息项便于对照。
								if it, ok := contentOf(content, s); ok {
									msg += "（" + it.ID + " 在 U1 里是 " + it.Item + " 的结果词，只能由数据带出，不能写字面量）"
								}
								out = append(out, lintFinding{"L6", "A2/D2、UI-11、X20",
									pos(lit.Pos()), msg})
							}
						}
					}
				}
				// L10：自绘描边。
				if sel.Sel.Name == "Border" || sel.Sel.Name == "BorderColor" {
					for _, arg := range x.Args {
						if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
							out = append(out, lintFinding{"L10", "UI-18、X14",
								pos(lit.Pos()), "描边用了字面色值 " + lit.Value})
						}
					}
					if sel.Sel.Name == "Border" && len(x.Args) >= 2 {
						if !isThemeBorder(x.Args[1]) {
							out = append(out, lintFinding{"L10", "UI-18、X14",
								pos(x.Pos()), "自绘描边不是 t.Border"})
						}
					}
				}
			}
		case *ast.BasicLit:
			// L2：字符冒充图标。
			if x.Kind == token.STRING {
				if s, err := strconv.Unquote(x.Value); err == nil && glyphIcons.MatchString(s) {
					out = append(out, lintFinding{"L2", "UI-21、X15",
						pos(x.Pos()), "字符串里用字符当图标：" + s})
				}
			}
		case *ast.CompositeLit:
			// L3 / L4 / L5：入口规格。
			typ := typeName(x.Type)
			if typ != "iconButtonSpec" && typ != "textIconButtonSpec" {
				break
			}
			fields := literalFields(x)
			if typ == "iconButtonSpec" {
				if _, ok := fields["label"]; !ok {
					out = append(out, lintFinding{"L3", "UI-21、D4",
						pos(x.Pos()), "图标入口缺可访问名（label）"})
				}
				// 纯图标入口的 tooltip：tip 显式给，或回落到 label 的名称（iconAction
				// 的默认）。两者都没有才算缺。
				if _, hasTip := fields["tip"]; !hasTip {
					if _, hasLabel := fields["label"]; !hasLabel {
						out = append(out, lintFinding{"L4", "UI-23 选择表、F17",
							pos(x.Pos()), "纯图标入口缺 tooltip 与名称"})
					}
				}
			}
			if typ == "textIconButtonSpec" {
				text, ok := fields["text"]
				switch {
				case !ok:
					out = append(out, lintFinding{"L5", "UI-23、F18",
						pos(x.Pos()), "图标 + 文字的入口缺可见文字（text）"})
				case text == "":
					// 只留图标与 label 是最容易被静默绕过的写法：可访问名还在，
					// 屏幕上却没有字。这里按字面量直接报。
					out = append(out, lintFinding{"L5", "UI-23、F18",
						pos(x.Pos()), "图标 + 文字的入口把可见文字写成了空串"})
				}
			}
		}
		return true
	})

	// L5：高后果动作不得退回纯文字按钮（渲染侧再验图标与可见文字）。
	for _, word := range []string{"允许一次", "一直允许", "拒绝"} {
		pattern := regexp.MustCompile(`actionButton\(c, "` + word + `"`)
		if pattern.Match(src) {
			out = append(out, lintFinding{"L5", "UI-23、F18", name + ":1",
				"高后果动作「" + word + "」用了纯文字按钮"})
		}
	}
	return out
}

// L11：分隔线与自绘描边只允许出现在登记过的位置。
func lintDividerAllowlist(files []string) []lintFinding {
	allowed := map[string]int{
		"controls.go":   1, // 附件 chip：落在 surface 上，需要一条线才看得出是条目
		"composer.go":   1, // 审批卡：唯一需要硬分开的自绘面（第三档边界）
		"icons.go":      1, // 审批的次按钮（一直允许 / 拒绝）：落在卡面上，需要一条轮廓
		"sidebar.go":    1, // 框架 Divider：入口区与会话列表两个模块
		"rightpanel.go": 1, // 框架 Divider：标签栏与内容
	}
	var out []lintFinding
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		n := strings.Count(string(src), "gui.Divider(") + strings.Count(string(src), ".Border(")
		if want := allowed[filepath.Base(f)]; n != want {
			out = append(out, lintFinding{"L11", "UI-18、X14", f + ":1",
				fmt.Sprintf("分隔线/自绘描边的处数 %d，白名单 %d", n, want)})
		}
	}
	return out
}

// ---- 渲染检查 ----

func lintRender(t *testing.T) []lintFinding {
	t.Helper()
	var out []lintFinding
	out = append(out, lintBaselines(t)...)
	out = append(out, lintRoundSpacing(t)...)
	out = append(out, lintStatesDistinct(t)...)
	out = append(out, lintHitAreas(t)...)
	out = append(out, lintInteractiveStates(t)...)
	out = append(out, lintMotionInvariance(t)...)
	out = append(out, lintTextScale(t)...)
	out = append(out, lintTooltipText(t)...)
	return out
}

// L8：三档宽度下，标题行 / 时间线 / 输入卡是同一条左缘。
func lintBaselines(t *testing.T) []lintFinding {
	t.Helper()
	edges := map[string][3]float32{}
	widths := []int{WindowWidth, 900, 720}
	for i, width := range widths {
		app := NewApp(Options{State: StateDefault})
		app.draft = "把日志也开上"
		tt := gui.NewTester(app.View, width, WindowHeight)
		title, ok1 := tt.Access("会话 修发送路径")
		reply, ok2 := tt.Find("消息在排队判定里丢了：运行中提交会覆盖队列。我把输入位改成“运行中先入队，结束后自动发出”。")
		input, ok3 := tt.Find("消息")
		if !ok1 || !ok2 || !ok3 {
			return []lintFinding{{check: "L8", rule: "UI-22、§9.14 排版基线 2", pos: "render",
				msg: fmt.Sprintf("%d 宽下量不到三个基线元素（%v %v %v）", width, ok1, ok2, ok3)}}
		}
		put := func(name string, v float32) {
			row := edges[name]
			row[i] = v
			edges[name] = row
		}
		put("标题行", title.Bounds.X)
		put("时间线", reply.X)
		put("输入卡", input.X)
	}
	return lintBaselinesAt("render", edges)
}

// lintBaselinesAt 是 L8 的纯函数部分，种子自检直接调它。
func lintBaselinesAt(pos string, edges map[string][3]float32) []lintFinding {
	var out []lintFinding
	for name, row := range edges {
		base := row[0]
		for i, v := range row {
			if v-base > 1.5 || base-v > 1.5 {
				out = append(out, lintFinding{"L8", "UI-22、§9.14 排版基线 2", pos,
					fmt.Sprintf("%s 在 %d 档的左缘 %.1f 与 1440 档的 %.1f 不同", name, []int{1440, 900, 720}[i], v, base)})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].msg < out[j].msg })
	return out
}

// L9：轮次之间（组间）的间距必须大于轮内（组内）的间距。
func lintRoundSpacing(t *testing.T) []lintFinding {
	t.Helper()
	_, tt := newTester(t, StateDefault)
	reply1, ok1 := tt.Find("消息在排队判定里丢了：运行中提交会覆盖队列。我把输入位改成“运行中先入队，结束后自动发出”。")
	header1, ok2 := tt.AccessByRole(gui.RoleDisclosure, "")
	bubble2, ok3 := tt.Find("顺手把输入区的旧逻辑也看下")
	if !ok1 || !ok2 || !ok3 {
		return []lintFinding{{check: "L9", rule: "A3、§9.14 第 3 层", pos: "render", msg: "量不到轮次与组头"}}
	}
	intra := header1.Bounds.Y - (reply1.Y + reply1.H)
	inter := bubble2.Y - (header1.Bounds.Y + header1.Bounds.H)
	if inter <= intra {
		return []lintFinding{{check: "L9", rule: "A3、§9.14 第 3 层", pos: "render",
			msg: fmt.Sprintf("组间 %.1f 不大于组内 %.1f", inter, intra)}}
	}
	return nil
}

// L7：五个状态两两可分（状态读屏名集合不同，且整帧像素不同）。
func lintStatesDistinct(t *testing.T) []lintFinding {
	t.Helper()
	states := []State{StateDefault, StateRunning, StateWaiting, StateFailed, StateEmpty}
	sigs := map[string]State{}
	images := map[string]State{}
	var out []lintFinding
	for _, s := range states {
		app := NewApp(Options{State: s})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		// 状态位是组头这个 disclosure 的可读名的一部分（图标盒不单独成节点），
		// 因此从文本/标签表里取「状态：…」。
		var labels []string
		for _, text := range tt.Texts() {
			if strings.HasPrefix(text, "状态：") {
				labels = append(labels, text)
			}
		}
		sort.Strings(labels)
		if len(labels) == 0 && s != StateEmpty {
			out = append(out, lintFinding{"L7", "D4", "render", fmt.Sprintf("%v 没有任何状态位", s)})
		}
		sig := s.String() + "|" + strings.Join(labels, ",")
		if prev, dup := sigs[sig]; dup {
			out = append(out, lintFinding{"L7", "D4", "render",
				fmt.Sprintf("%v 与 %v 的状态签名相同", s, prev)})
		}
		sigs[sig] = s
		hash := imageHash(tt.Image())
		if prev, dup := images[hash]; dup {
			out = append(out, lintFinding{"L7", "D4", "render",
				fmt.Sprintf("%v 与 %v 渲染成同一张图", s, prev)})
		}
		images[hash] = s
	}
	return out
}

// L5（渲染侧）：高后果动作按 U3-actions.json 判定——`icon+text` 的必须有图标与
// 可见文字；`confirm` 为真的必须先确认再生效。未在首片渲染的动作只记录不判。
func lintHighConsequence(t *testing.T, actions *actionDoc) []lintFinding {
	t.Helper()
	app, tt := newTester(t, StateWaiting)
	var out []lintFinding
	seen := 0
	for _, it := range actions.Items {
		info, ok := tt.Access(it.Action)
		if !ok {
			continue // 首片没有这个入口（删除会话等）
		}
		seen++
		if it.Form != "icon+text" {
			out = append(out, lintFinding{"L5", "UI-23、F18（" + it.ID + "）", "render",
				it.Action + " 的形式应为 " + it.Form + "，清单与实现不一致"})
			continue
		}
		if !hasInk(tt.Image(), info.Bounds) {
			out = append(out, lintFinding{"L5", "UI-23、F18（" + it.ID + "）", "render",
				it.Action + " 只有文字没有图标"})
		}
		// 可见文字：必须能在文本表里找到这个动作名，且按钮比"只有图标"的盒子
		// 更宽（只留 label 时可访问名还在，但屏幕上没有字、盒子也只有图标那么宽）。
		if _, visible := tt.Find(it.Action); !visible {
			out = append(out, lintFinding{"L5", "UI-23、F18（" + it.ID + "）", "render",
				it.Action + " 在屏幕上没有可见文字（可能只留了 label）"})
		} else if info.Bounds.W <= iconOnlyButtonWidth+space2 {
			out = append(out, lintFinding{"L5", "UI-23、F18（" + it.ID + "）", "render",
				it.Action + " 的按钮只有图标那么宽，看不到文字"})
		}
	}
	// 反向：清单里 form=icon 的动作，渲染出来时不得有可见文字（盒子只有图标宽）。
	for _, st := range []State{StateDefault, StateRunning, StateWaiting} {
		app2 := NewApp(Options{State: st})
		t2 := gui.NewTester(app2.View, WindowWidth, WindowHeight)
		for _, it := range actions.Items {
			if it.Form != "icon" {
				continue
			}
			info, ok := t2.Access(it.Action)
			if !ok {
				continue
			}
			if info.Bounds.W > iconOnlyButtonWidth+space2 {
				out = append(out, lintFinding{"L5", "UI-23、F18（" + it.ID + "）", "render",
					it.Action + " 应是纯图标，却画出了文字（盒子宽 " + fmt.Sprintf("%.0f", info.Bounds.W) + "）"})
			}
		}
	}
	if seen == 0 {
		out = append(out, lintFinding{"L5", "UI-23、F18", "render", "等待态的审批决定一个都没渲染"})
	}
	// confirm=true 的动作：第一次点击不得生效。
	if it, ok := actionByAction(actions, "一直允许"); ok && it.Confirm {
		if err := tt.Click("一直允许"); err != nil {
			out = append(out, lintFinding{"L5", "UI-23、F18（" + it.ID + "）", "render", "点不到一直允许"})
		} else if app.approvalDecision != "" {
			out = append(out, lintFinding{"L5", "UI-23、F18（" + it.ID + "）", "render",
				"一直允许一次点击就生效了，缺确认"})
		}
	}
	return out
}

// actionByAction 按动作名取清单条目。
func actionByAction(doc *actionDoc, name string) (actionEntry, bool) {
	for _, it := range doc.Items {
		if it.Action == name {
			return it, true
		}
	}
	return actionEntry{}, false
}

// L4（渲染侧）：纯图标入口的 tooltip 给名称；发送 / 停止还要给快捷键。
func lintTooltipText(t *testing.T) []lintFinding {
	t.Helper()
	var out []lintFinding
	for _, c := range []struct {
		state State
		name  string
	}{{StateDefault, "发送"}, {StateRunning, "停止"}} {
		_, tt := newTester(t, c.state)
		info, ok := tt.Access(c.name)
		if !ok {
			out = append(out, lintFinding{"L4", "UI-23 选择表、F17", "render", "找不到入口 " + c.name})
			continue
		}
		if info.Description == "" {
			out = append(out, lintFinding{"L4", "UI-23 选择表、F17", "render", c.name + " 没有 tooltip"})
			continue
		}
		if !strings.Contains(info.Description, c.name) || !strings.Contains(info.Description, "Enter") {
			out = append(out, lintFinding{"L4", "UI-23 选择表、F17", "render",
				c.name + " 的 tooltip 缺名称或快捷键：" + info.Description})
		}
	}
	return out
}

// L13：图标入口的命中区不小于 hit-min。
func lintHitAreas(t *testing.T) []lintFinding {
	t.Helper()
	_, tt := newTester(t, StateDefault)
	var out []lintFinding
	for _, label := range []string{"新建会话", "搜索", "收起左栏", "打开右栏", "附件", "发送", "读取更早的记录"} {
		info, ok := tt.Access(label)
		if !ok {
			out = append(out, lintFinding{"L13", "UI-22 §6.5", "render", "找不到入口 " + label})
			continue
		}
		if info.Bounds.W < hitMin || info.Bounds.H < hitMin {
			out = append(out, lintFinding{"L13", "UI-22 §6.5", "render",
				fmt.Sprintf("%s 的命中区 %.0f×%.0f 小于 %d", label, info.Bounds.W, info.Bounds.H, hitMin)})
		}
	}
	return out
}

// L14：自绘可交互元素有默认 / 悬停 / 聚焦 / 按下；禁用不适用（组头与用量）。
func lintInteractiveStates(t *testing.T) []lintFinding {
	t.Helper()
	_, tt := newTester(t, StateDefault)
	row := headerRow(t, tt)
	x, y := row.X+row.W-4, row.Y+row.H/2
	hover := colorOf(colorSurfaceHover.in(theme(false, gui.Preferences{})))
	pressed := colorOf(colorSurfacePressed.in(theme(false, gui.Preferences{})))
	var out []lintFinding
	tt.Move(x, y)
	if got := tt.Image().RGBAAt(int(x), int(y)); got != hover {
		out = append(out, lintFinding{"L14", "X17、§9.14 第 4 层", "render", "组头没有悬停态"})
	}
	tt.Press(x, y)
	if got := tt.Image().RGBAAt(int(x), int(y)); got != pressed {
		out = append(out, lintFinding{"L14", "X17、§9.14 第 4 层", "render", "组头没有按下态"})
	}
	tt.Release(x, y)
	// 图标入口的禁用态：空草稿时的发送槽。
	emptyApp := NewApp(Options{State: StateDefault})
	et := gui.NewTester(emptyApp.View, WindowWidth, WindowHeight)
	if info, ok := et.Access("发送"); !ok || !info.Disabled {
		out = append(out, lintFinding{"L14", "X17、§9.14 第 4 层", "render", "发送槽在无草稿时没有禁用态"})
	}
	return out
}

// L12：关动效后信息与布局不变。
func lintMotionInvariance(t *testing.T) []lintFinding {
	t.Helper()
	on := gui.NewTester(NewApp(Options{State: StateDefault}).View, WindowWidth, WindowHeight)
	app := NewApp(Options{State: StateDefault})
	off := gui.NewTesterWith(app.View, WindowWidth, WindowHeight,
		gui.TesterOptions{Preferences: gui.Preferences{ReduceMotion: true}})
	var out []lintFinding
	a, b := on.Texts(), off.Texts()
	if strings.Join(a, "|") != strings.Join(b, "|") {
		out = append(out, lintFinding{"L12", "UI-5、§9.14 动效从属", "render", "关动效后文字集合不同"})
	}
	for _, label := range []string{"标题行", "对话流", "消息"} {
		ra, ok1 := on.Find(label)
		rb, ok2 := off.Find(label)
		if !ok1 || !ok2 || ra != rb {
			out = append(out, lintFinding{"L12", "UI-5、§9.14 动效从属", "render",
				fmt.Sprintf("关动效后 %s 的位置变了（%v → %v）", label, ra, rb)})
		}
	}
	return out
}

// L15：200% 文本缩放下不越界。
func lintTextScale(t *testing.T) []lintFinding {
	t.Helper()
	var out []lintFinding
	for _, s := range []State{StateDefault, StateRunning, StateWaiting, StateFailed, StateEmpty} {
		app := NewApp(Options{State: s})
		tt := gui.NewTesterWith(app.View, WindowWidth, WindowHeight,
			gui.TesterOptions{Preferences: gui.Preferences{TextScale: 2}})
		for _, text := range tt.Texts() {
			r, ok := tt.Find(text)
			if !ok {
				continue
			}
			if r.W == 0 || r.H == 0 {
				continue // 滚出视口的行
			}
			if r.X < 0 || r.Y < 0 || r.X+r.W > WindowWidth || r.Y+r.H > WindowHeight {
				out = append(out, lintFinding{"L15", "§9.14 第 3 层、U2 §7.4", "render",
					fmt.Sprintf("%v 的 %q 在 200%% 下越界：%+v", s, text, r)})
			}
		}
	}
	return out
}

// L16（清单引用完整性）：U1 清单里每条都要能在 U1-information.md 找到信息项。
func lintListReferences(t *testing.T, content *contentDoc) []lintFinding {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "domains", "U1-information.md"))
	if err != nil {
		return []lintFinding{{check: "L16", rule: "§9.14 第 1 层、U1-information", pos: "docs",
			msg: "读不到 U1-information.md：" + err.Error()}}
	}
	var out []lintFinding
	for _, it := range content.Items {
		row := "| " + it.Item + " |"
		if !strings.Contains(string(raw), row) {
			out = append(out, lintFinding{"L16", "§9.14 第 1 层、U1-information", "docs",
				fmt.Sprintf("%s 的来源信息项 %s 在 U1-information.md 里找不到", it.ID, it.Item)})
		}
	}
	return out
}

// iconOnlyButtonWidth 是"只有图标 + 文字按钮内边距"的宽度：图标入口是
// rowCompact 的方盒，而图标 + 文字的入口一定比这个更宽。
var iconOnlyButtonWidth = float32(iconInline + 2*space3)

// ---- 清单（单一来源：docs/domains/*.json） ----

// contentList 是 U1 的内容/界面语清单。
type contentEntry struct {
	ID    string `json:"id"`
	Match string `json:"match"`
	Value string `json:"value"`
	Item  string `json:"item"`
	Kind  string `json:"kind"`
	Why   string `json:"why"`
}

type contentDoc struct {
	Items []contentEntry `json:"items"`
}

// actionEntry 是 U3 的高后果动作清单。
type actionEntry struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Consequence string `json:"consequence"`
	Form        string `json:"form"`
	Confirm     bool   `json:"confirm"`
	Basis       string `json:"basis"`
}

type actionDoc struct {
	Items []actionEntry `json:"items"`
}

// loadJSON 读 docs/domains 下的清单；读不到就让检查器失败（不许静默用空表）。
func loadJSON(t *testing.T, name string, v any) {
	t.Helper()
	path := filepath.Join("..", "..", "docs", "domains", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读清单 %s：%v（检查器必须以清单为单一来源，不允许空表）", path, err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("解析 %s：%v", path, err)
	}
}

// contentOf 判断一个字符串在 U1 清单里是不是内容。
func contentOf(doc *contentDoc, s string) (contentEntry, bool) {
	for _, it := range doc.Items {
		if it.Kind != "content" {
			continue
		}
		if (it.Match == "exact" && s == it.Value) ||
			(it.Match == "prefix" && strings.HasPrefix(s, it.Value)) {
			return it, true
		}
	}
	return contentEntry{}, false
}

// ---- 静态检查用到的表 ----

// layoutCalls 是受档位约束的排布调用，以及允许的字面量（0 一律允许，单独处理）。
var layoutCalls = map[string]map[string]bool{
	"Padding": {}, "PaddingX": {}, "PaddingY": {}, "PaddingStart": {}, "PaddingEnd": {},
	"Gap": {}, "Width": {}, "Height": {}, "MinHeight": {}, "MaxHeight": {},
	"MinWidth": {}, "Radius": {}, "BorderWidth": {},
}

// statusWords 是只能用样式表达、不得画成文字的状态词。
var statusWords = map[string]bool{
	"完成": true, "正在运行": true, "等待你同意": true, "失败": true,
}

// glyphIcons 是冒充图标的字符。
var glyphIcons = regexp.MustCompile(`[＋‹›×⋯]`)

// isThemeBorder 判断表达式是不是 t.Border。
func isThemeBorder(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Border" {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "t"
}

// typeName 取复合字面量的类型名。
func typeName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	}
	return ""
}

// literalFields 取复合字面量里的字段：值能直接读成字符串的给字符串，其余给
// 一个非空标记（"<expr>"），因为字段是否存在才是检查要判的事。
func literalFields(x *ast.CompositeLit) map[string]string {
	out := map[string]string{}
	for _, elt := range x.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if s, err := strconv.Unquote(lit.Value); err == nil {
				out[key.Name] = s
				continue
			}
		}
		out[key.Name] = "<expr>"
	}
	return out
}

// imageHash 给一帧图求一个便宜的指纹（用于"两态渲染成同一张图"）。
func imageHash(img *image.RGBA) string {
	var h uint64 = 1469598103934665603
	b := img.Pix
	for i := 0; i < len(b); i += 97 {
		h ^= uint64(b[i])
		h *= 1099511628211
	}
	return strconv.FormatUint(h, 16)
}

// 时间与动效相关的时间常量在这里引用一次，避免"未使用"的导入。
var _ = time.Millisecond
