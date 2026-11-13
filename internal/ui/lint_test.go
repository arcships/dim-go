package ui

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
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
	"L16": "清单条目指回 U1 信息项编号",
	"L17": "让实现与元素表的形式一致：补图标或改清单（改形式=形式级）",
	"L18": "改窄窗退化（堆叠 / 滚动），或收窄定宽列；不要靠裁切",
	"L19": "在 U2-elements.json 里补 component 与 symbol，或在 internal/ui 里补上该组件",
	"L20": "给 a.md 设 markdownTheme（不是零值 options），并让样例包含全部块级结构",
}

// TestUILint 是检查器本体：工作区必须零违例。
func TestUILint(t *testing.T) {
	findings := lintWorkspace(t)
	if len(findings) == 0 {
		t.Logf("检查器全绿：%d 个检查项（见 docs/domains/U2-checks.md）", 20)
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

	// 第七类：元素表声明 icon+text，实现却只画了文字（评审删掉分支标记图标的那种）。
	textOnly := func(c *gui.Context) {
		gui.Row(c).Padding(space1, space2).Label("切换分支：feature/queue-rework").Children(func() {
			gui.Text(c, "feature/queue-rework")
		})
	}
	tt := gui.NewTester(textOnly, 600, 200)
	spec := elementEntry{
		ID: "seed", Element: "分支标记", Form: "icon+text", Checkable: true,
		Probe: "切换分支：feature/queue-rework", Text: "feature/queue-rework",
	}
	got := checkElementForm(t, tt, spec)
	if len(got) == 0 || got[0].check != "L17" {
		t.Fatalf("元素形式种子没有被 L17 报出：%v", got)
	}
	t.Logf("种子「元素缺图标」→ %s", got[0].String())

	// 第十类：清单说"用开关"，实现换成图标（或清单写 icon 而实现是开关）。
	switched := &actionDoc{Items: []actionEntry{{
		ID: "seed", Action: "停用插件", Form: "icon", Confirm: false,
	}}}
	if got := checkIconNotSwitch(t, switched); len(got) == 0 || got[0].check != "L5" {
		t.Fatalf("开关形式种子没有被 L5 报出：%v", got)
	} else {
		t.Logf("种子「开关形式与实现不一致」→ %s", got[0].String())
	}

	// 第九类：Markdown 主题被清空（零值 options）——块级样式就不归我们管了。
	if got := checkMarkdownTheme(theme(false, gui.Preferences{}), nil); len(got) == 0 || got[0].check != "L20" {
		t.Fatalf("Markdown 主题种子没有被 L20 报出：%v", got)
	} else {
		t.Logf("种子「Markdown 主题清空」→ %s", got[0].String())
	}

	// 第八类：定宽列把内容顶出窗口（评审在 720 档看到的那种）。
	wideRow := func(c *gui.Context) {
		gui.Row(c).Width(800).Children(func() {
			gui.Spacer(c)
			gui.Text(c, "匹配方式")
		})
	}
	wt := gui.NewTester(wideRow, 720, 200)
	if got := checkNoOverflow(wt, "seed", 720, 200); len(got) == 0 || got[0].check != "L18" {
		t.Fatalf("越界种子没有被 L18 报出：%v", got)
	} else {
		t.Logf("种子「内容越界」→ %s", got[0].String())
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
	var elements elementDoc
	loadJSON(t, "U2-elements.json", &elements)
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
	out = append(out, lintElementForms(t, &elements)...)
	out = append(out, lintNoOverflow(t)...)
	out = append(out, lintViewMapping(t, &elements)...)
	out = append(out, lintMarkdownBlocks(t)...)
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
			continue // 纯图标的动作由下面的反向检查管（不得有可见文字）
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
	// switch 形式：清单说"用开关"，实现就必须是框架 Switch（有开关语义、有开/关
	// 两态、命中区达标），不能换成图标按钮。
	out = append(out, checkSwitchForms(t, actions)...)

	// 反向：清单里 form=icon 的动作，渲染出来时不得是开关。
	out = append(out, checkIconNotSwitch(t, actions)...)

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

// extTabOf 是"停用某类"与扩展页签的对应（本批四类各一个页签）。
func extTabOf(action string) (int, bool) {
	switch action {
	case "停用技能":
		return 0, true
	case "停用插件":
		return 1, true
	case "停用 MCP":
		return 2, true
	case "停用钩子":
		return 3, true
	}
	return 0, false
}

// checkSwitchForms 判"清单 form=switch 的动作真的渲染成框架 Switch"。
func checkSwitchForms(t *testing.T, actions *actionDoc) []lintFinding {
	t.Helper()
	var out []lintFinding
	for _, it := range actions.Items {
		if it.Form != "switch" {
			continue
		}
		tab, ok := extTabOf(it.Action)
		if !ok {
			out = append(out, lintFinding{"L5", "UI-23、D5（" + it.ID + "）", "render",
				it.Action + " 标了 switch，但检查器不知道它落在哪个页签"})
			continue
		}
		app := NewApp(Options{State: StateDefault})
		app.screen, app.settingsSection, app.extTab = ScreenSettings, 3, tab
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		// 该类第一行的开关：名字是「启用：<名称>」。
		items := app.extItems(extKind(tab))
		if len(items) == 0 {
			out = append(out, lintFinding{"L5", "UI-23、D5（" + it.ID + "）", "render",
				it.Action + " 的页签没有样例行"})
			continue
		}
		want := "启用：" + items[0].name
		info, ok := tt.Access(want)
		if !ok {
			out = append(out, lintFinding{"L5", "UI-23、D5（" + it.ID + "）", "render",
				it.Action + " 找不到开关 " + want})
			continue
		}
		if info.Role != gui.RoleSwitch {
			out = append(out, lintFinding{"L5", "UI-23、D5（" + it.ID + "）", "render",
				fmt.Sprintf("%s 不是开关（role=%v）", it.Action, info.Role)})
			continue
		}
		// 命中区：框架 Switch 是 36×20，低于通用 24 的宽度下限但不低于高度下限。
		if info.Bounds.W < 36 || info.Bounds.H < 20 {
			out = append(out, lintFinding{"L5", "UI-23、D5（" + it.ID + "）", "render",
				fmt.Sprintf("%s 的开关命中区 %.0f×%.0f 小于 36×20", it.Action, info.Bounds.W, info.Bounds.H)})
		}
		if info.Checked != items[0].on {
			out = append(out, lintFinding{"L5", "UI-23、D5（" + it.ID + "）", "render",
				it.Action + " 的开关状态与数据不一致"})
		}
	}
	return out
}

// checkIconNotSwitch 判"清单 form 不是 switch 的动作没被渲染成开关"。
// 扩展类的启停动作按行渲染成「启用：<名称>」的开关；若清单说它是 icon / text，
// 却在这里查到开关语义，就是清单与实现不一致。
func checkIconNotSwitch(t *testing.T, actions *actionDoc) []lintFinding {
	t.Helper()
	var out []lintFinding
	for _, it := range actions.Items {
		if it.Form == "switch" {
			continue
		}
		if tab, ok := extTabOf(it.Action); ok {
			app := NewApp(Options{State: StateDefault})
			app.screen, app.settingsSection, app.extTab = ScreenSettings, 3, tab
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			items := app.extItems(extKind(tab))
			if len(items) == 0 {
				continue
			}
			if info, ok := tt.Access("启用：" + items[0].name); ok && info.Role == gui.RoleSwitch {
				out = append(out, lintFinding{"L5", "UI-23、D5（" + it.ID + "）", "render",
					it.Action + " 声明为 " + it.Form + "，却渲染成了开关"})
			}
			continue
		}
		app := NewApp(Options{State: StateRunning})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		if info, ok := tt.Access(it.Action); ok && info.Role == gui.RoleSwitch {
			out = append(out, lintFinding{"L5", "UI-23、D5（" + it.ID + "）", "render",
				it.Action + " 声明为 " + it.Form + "，却渲染成了开关"})
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

// L13：图标入口的命中区不小于 hit-min。有些入口只在编辑态 / 右栏出现，
// 因此按"在哪个准备态里出现"分组量。
func lintHitAreas(t *testing.T) []lintFinding {
	t.Helper()
	cases := []struct {
		label   string
		prepare func(*App)
	}{
		{"新建会话", nil}, {"搜索", nil}, {"收起左栏", nil}, {"打开右栏", nil},
		{"附件", nil}, {"发送", nil}, {"读取更早的记录", nil}, {"修改目标", nil},
		{"保存目标", func(a *App) { a.goalEditing = true }},
		{"取消编辑", func(a *App) { a.goalEditing = true }},
		{"取消任务", func(a *App) { a.rightOpen, a.rightTab = true, 2 }},
	}
	var out []lintFinding
	for _, c := range cases {
		app := NewApp(Options{State: StateDefault})
		if c.prepare != nil {
			c.prepare(app)
		}
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		info, ok := tt.Access(c.label)
		if !ok {
			out = append(out, lintFinding{"L13", "UI-22 §6.5", "render", "找不到入口 " + c.label})
			continue
		}
		if info.Bounds.W < hitMin || info.Bounds.H < hitMin {
			out = append(out, lintFinding{"L13", "UI-22 §6.5", "render",
				fmt.Sprintf("%s 的命中区 %.0f×%.0f 小于 %d", c.label, info.Bounds.W, info.Bounds.H, hitMin)})
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

// L20：Markdown 块样式存在性（U0 §3.1 的 M 系列）。
// 两段：① 助手正文的 Markdown 主题必须是我们声明的那份（不是零值 options）；
// ② 渲染一条含全部块级结构的回复后，各类块都要画出预期标记。
func lintMarkdownBlocks(t *testing.T) []lintFinding {
	t.Helper()
	var out []lintFinding

	// ① 配置：主题存在，且关键项与我们的 token 一致。
	th := theme(false, gui.Preferences{})
	md := markdownTheme(th)
	out = append(out, checkMarkdownTheme(th, md)...)

	// ② 渲染：逐块渲染最小文档，数像素判样式（块内部的文字不成可访问节点，
	// 靠 Find 找文字不可靠）。
	out = append(out, checkMarkdownRender(t, md, th)...)
	return out
}

// checkMarkdownTheme 判"主题是不是我们声明的那份"（种子自检直接调它）。
func checkMarkdownTheme(t *gui.Theme, md *markdown.Theme) []lintFinding {
	if md == nil {
		return []lintFinding{{"L20", "U0 §3.1、C4 §5", "render",
			"Markdown 主题为空：用了零值 options（默认样式）"}}
	}
	var out []lintFinding
	want := func(name string, got, expect gui.Color) {
		if got != expect {
			out = append(out, lintFinding{"L20", "U0 §3.1、C4 §5", "render",
				fmt.Sprintf("Markdown 的 %s 没取我们的 token", name)})
		}
	}
	want("链接色", md.Link, colorAccentInk.in(t))
	want("代码块底色", md.CodeBlockBackground, colorSurface.in(t))
	want("行内代码底色", md.CodeBackground, colorSurface.in(t))
	want("表头底色", md.TableHeader, colorSurface.in(t))
	want("引用竖线", md.QuoteBar, t.Border)
	if md.QuoteBarWidth < 2 {
		out = append(out, lintFinding{"L20", "U0 §3.1", "render", "引用竖线宽度 < 2"})
	}
	if md.CodeBlockRadius != radiusCode {
		out = append(out, lintFinding{"L20", "U0 §3.1", "render",
			fmt.Sprintf("代码块圆角 %v，想要 %v", md.CodeBlockRadius, radiusCode)})
	}
	// 字阶必须递减（h1 > h2 > … > h6）。
	for i := 1; i < len(md.Headings); i++ {
		if md.Headings[i].Size >= md.Headings[i-1].Size {
			out = append(out, lintFinding{"L20", "U0 §3.1", "render",
				fmt.Sprintf("标题字阶没有递减：h%d ≥ h%d", i+1, i)})
			break
		}
	}
	return out
}

// checkMarkdownRender 判"各类块画出了预期标记"（种子自检直接调它）。
//
// 按块各渲染一份最小文档，再数像素：Markdown 视图内部的文字不单独成可访问
// 节点，靠 Find 找文字不可靠，数颜色才是稳定的判据。
func checkMarkdownRender(t *testing.T, md *markdown.Theme, th *gui.Theme) []lintFinding {
	_ = t
	doc := func(src string) *gui.Tester {
		return gui.NewTester(func(c *gui.Context) {
			o := markdown.Options{Theme: md}
			markdown.View(c, src, &o)
		}, 600, 320)
	}
	var out []lintFinding
	fail := func(msg string) {
		out = append(out, lintFinding{"L20", "U0 §3.1", "render", msg})
	}

	// 标题：h2 的墨迹高度明显大于同文正文。
	h2 := inkHeight(doc("## 标题").Image())
	plain := inkHeight(doc("标题").Image())
	if h2 < plain*1.2 {
		fail(fmt.Sprintf("标题没有字号跳变：h2 墨迹 %.1f，正文 %.1f", h2, plain))
	}

	// 行内代码：chip 底色。
	if n := countNear(doc("用 `queue` 参数").Image(), md.CodeBackground, 12); n < 20 {
		fail(fmt.Sprintf("行内代码没有底色（命中 %d 像素）", n))
	}

	// 代码块：整块的面。
	if n := countNear(doc("```go\nx := 1\n```").Image(), md.CodeBlockBackground, 12); n < 200 {
		fail(fmt.Sprintf("代码块没有自己的面（命中 %d 像素）", n))
	}

	// 引用：左侧竖线（主题边界色）。
	if n := countNear(doc("> 引用的这一行").Image(), th.Border, 12); n < 10 {
		fail(fmt.Sprintf("引用块左侧没有竖线（命中 %d 像素）", n))
	}

	// 表格：表头底色。
	if n := countNear(doc("| a | b |\n|---|---|\n| 1 | 2 |").Image(), md.TableHeader, 12); n < 100 {
		fail(fmt.Sprintf("表格表头没有底色（命中 %d 像素）", n))
	}

	// 链接：强调色。
	if n := countNear(doc("[链接文字](https://example.com)").Image(), md.Link, 40); n < 5 {
		fail(fmt.Sprintf("链接没有用强调色（命中 %d 像素）", n))
	}

	// 三处呼吸（M2/M6/M7）：块间、列表项间、表格行距。
	out = append(out, checkMarkdownRhythm(md)...)
	return out
}

// checkMarkdownRhythm 量三处呼吸（U0 §3.1 的 M2/M6/M7）：块间、列表项间、
// 表格行距。用墨迹带（同一行文字）的起点差来量，避免依赖具体文字的坐标。
func checkMarkdownRhythm(md *markdown.Theme) []lintFinding {
	doc := func(src string) *image.RGBA {
		return gui.NewTester(func(c *gui.Context) {
			o := markdown.Options{Theme: md}
			markdown.View(c, src, &o)
		}, 600, 320).Image()
	}
	var out []lintFinding
	fail := func(msg string) { out = append(out, lintFinding{"L20", "U0 §3.1（M2/M6/M7）", "render", msg}) }

	// 块间：标题与下一段之间的墨迹间隙 ≥ 0.8em（13px 正文 → 约 10px）。
	bands := inkBands(doc("## 标题\n\n正文一段"), 0, 600)
	if len(bands) < 2 {
		fail("量不到标题与正文两行")
	} else if gap := bands[1][0] - bands[0][1] - 1; gap < 10 {
		fail(fmt.Sprintf("块间呼吸不足：标题到正文的墨迹间隙 %dpx（想要 ≥ 10）", gap))
	} else if g := float32(gap) / 13; g > 2.4 { // 13px 正文：上限 2.4em，防止调过头
		fail(fmt.Sprintf("块间呼吸过大：%.2fem（上限 2.4em）", g))
	}

	// 列表项间：两项的起点差 ≥ 行盒 + 0.2em（13px → 约 22px）。
	lb := inkBands(doc("- 第一项\n- 第二项"), 0, 600)
	if len(lb) < 2 {
		fail("量不到两个列表项")
	} else if pitch := lb[1][0] - lb[0][0]; pitch < 21 {
		fail(fmt.Sprintf("列表项间距不足：两项起点差 %dpx（想要 ≥ 21）", pitch))
	}

	// 表格行距：两条横向边线的间距 ≥ 2.2em（13px → 约 29px）。
	// 表格的边线与表头底色是连续的，横向墨迹带分不出行，所以量边线。
	lines := hLines(doc("| a | b |\n|---|---|\n| 1 | 2 |"), md.TableBorder, 600)
	if len(lines) < 2 {
		fail("量不到表格的横向边线")
	} else if pitch := lines[1] - lines[0]; pitch < 29 {
		fail(fmt.Sprintf("表格行高不足：边线间距 %dpx（想要 ≥ 29）", pitch))
	}
	return out
}

// hLines 找接近 want 的横向整行（表格边线），返回它们的 y。
func hLines(img *image.RGBA, want gui.Color, width int) []int {
	var out []int
	for y := 0; y < img.Bounds().Dy(); y++ {
		hits := 0
		for x := 0; x < width && x < img.Bounds().Dx(); x++ {
			c := img.RGBAAt(x, y)
			if abs(int(c.R)-int(want.R)) <= 12 && abs(int(c.G)-int(want.G)) <= 12 &&
				abs(int(c.B)-int(want.B)) <= 12 {
				hits++
			}
		}
		if hits >= 40 { // 两列表格的边线也远宽于 40px；整行像素里这条线最密
			// 相邻的像素行算同一条线。
			if len(out) == 0 || y-out[len(out)-1] > 1 {
				out = append(out, y)
			}
		}
	}
	return out
}

// inkBands 把图切成"有墨迹的行带"，返回每段的 [起, 止]。
func inkBands(img *image.RGBA, x0, x1 int) [][2]int {
	bg := img.RGBAAt(img.Bounds().Dx()-2, img.Bounds().Dy()-2)
	var out [][2]int
	start := -1
	for y := 0; y < img.Bounds().Dy(); y++ {
		dark := false
		for x := x0; x < x1 && x < img.Bounds().Dx(); x++ {
			c := img.RGBAAt(x, y)
			if abs(int(c.R)-int(bg.R))+abs(int(c.G)-int(bg.G))+abs(int(c.B)-int(bg.B)) > 60 {
				dark = true
				break
			}
		}
		if dark && start < 0 {
			start = y
		}
		if !dark && start >= 0 {
			out = append(out, [2]int{start, y - 1})
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, [2]int{start, img.Bounds().Dy() - 1})
	}
	return out
}

// inkHeight 是墨迹（与背景不同）的最大纵向跨度，用来比字号。
func inkHeight(img *image.RGBA) float32 {
	bg := img.RGBAAt(img.Bounds().Dx()-2, img.Bounds().Dy()-2)
	minY, maxY := img.Bounds().Dy(), -1
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x += 2 {
			c := img.RGBAAt(x, y)
			d := abs(int(c.R)-int(bg.R)) + abs(int(c.G)-int(bg.G)) + abs(int(c.B)-int(bg.B))
			if d > 30 {
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
				break
			}
		}
	}
	if maxY < 0 {
		return 0
	}
	return float32(maxY - minY + 1)
}

// countNear 数图里接近 want 的像素个数（块底色 / 边线 / 链接色）。
func countNear(img *image.RGBA, want gui.Color, tol int) int {
	n := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			c := img.RGBAAt(x, y)
			if abs(int(c.R)-int(want.R)) <= tol && abs(int(c.G)-int(want.G)) <= tol &&
				abs(int(c.B)-int(want.B)) <= tol {
				n++
			}
		}
	}
	return n
}

// differs 判某点与页面底色不同。
func differs(img *image.RGBA, x, y int, page color.RGBA) bool {
	if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
		return false
	}
	c := img.RGBAAt(x, y)
	d := abs(int(c.R)-int(page.R)) + abs(int(c.G)-int(page.G)) + abs(int(c.B)-int(page.B))
	return d > 12
}

// containsColor 判矩形里有没有接近 want 的像素（链接的强调色）。
func containsColor(img *image.RGBA, r gui.Rect, want gui.Color, tol int) bool {
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
				continue
			}
			c := img.RGBAAt(x, y)
			if abs(int(c.R)-int(want.R)) <= tol && abs(int(c.G)-int(want.G)) <= tol &&
				abs(int(c.B)-int(want.B)) <= tol {
				return true
			}
		}
	}
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// L19：view 映射覆盖率——元素表每一条都要指到一个组件（U4 §1）。
// 判法：`component` 与 `symbol` 非空，且 `symbol` 是 internal/ui 里真实存在的标识符
// （框架符号 `gui.X` / `markdown.X` 取最后一段，按同名方法名在源码里找）。
func lintViewMapping(t *testing.T, elements *elementDoc) []lintFinding {
	t.Helper()
	src, err := os.ReadFile(".")
	if err != nil {
		_ = src
	}
	var all []byte
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		all = append(all, b...)
	}
	var out []lintFinding
	for _, it := range elements.Items {
		if it.Component == "" || it.Symbol == "" {
			out = append(out, lintFinding{"L19", "U4 §1、§9.14 第 1 层", "docs",
				fmt.Sprintf("%s（%s）没有指到组件：component 或 symbol 为空", it.ID, it.Element)})
			continue
		}
		name := it.Symbol
		if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).Match(all) {
			out = append(out, lintFinding{"L19", "U4 §1、§9.14 第 1 层", "docs",
				fmt.Sprintf("%s（%s）指的组件 %q 在 internal/ui 里找不到", it.ID, it.Element, it.Symbol)})
		}
	}
	return out
}

// L18：元素的盒子不得越出窗口（含设置屏与主界面，1440 / 900 / 720 三档）。
// 用 Tester 的 Bounds 判：文字与带名字的元素（下拉、按钮、行）都算；滚出视口
// 的零面积行跳过。
func lintNoOverflow(t *testing.T) []lintFinding {
	t.Helper()
	var out []lintFinding
	screens := []struct {
		name string
		app  func() *App
	}{
		{"主界面·默认", func() *App { return NewApp(Options{State: StateDefault}) }},
		{"主界面·运行中", func() *App { return NewApp(Options{State: StateRunning}) }},
		{"主界面·等待同意", func() *App { return NewApp(Options{State: StateWaiting}) }},
		{"设置·外观", func() *App {
			a := NewApp(Options{State: StateDefault})
			a.screen, a.settingsSection = ScreenSettings, 0
			return a
		}},
		{"设置·供应商", func() *App {
			a := NewApp(Options{State: StateDefault})
			a.screen, a.settingsSection = ScreenSettings, 1
			return a
		}},
		{"设置·命令规则", func() *App {
			a := NewApp(Options{State: StateDefault})
			a.screen, a.settingsSection = ScreenSettings, 2
			return a
		}},
	}
	for _, sc := range screens {
		for _, width := range []int{1440, 900, 720} {
			tt := gui.NewTester(sc.app().View, width, WindowHeight)
			out = append(out, checkNoOverflow(tt, sc.name, width, WindowHeight)...)
		}
	}
	return out
}

// checkNoOverflow 是 L18 的纯判定（种子自检直接调它）。
func checkNoOverflow(tt *gui.Tester, screen string, width, height int) []lintFinding {
	var out []lintFinding
	for _, text := range tt.Texts() {
		// 优先用可访问节点的盒子：它是元素的完整盒子，不会被裁到窗口内；
		// Find 给的是可见文字的范围，被裁时看不出"元素其实伸出了窗口"。
		r, ok := tt.Find(text)
		if info, okA := tt.Access(text); okA {
			r, ok = info.Bounds, true
		}
		if !ok || r.W <= 0 || r.H <= 0 {
			continue // 没有可见面积：不在这一档的可见范围内
		}
		// 只查横向越界与"跑到窗口上边之外"：纵向超出下沿是滚动内容，属正常
		// （长列表本来就在视口外）。原来的四边全查会把滚动内容误报成越界。
		if r.X < -0.5 || r.Y < -0.5 || r.X+r.W > float32(width)+0.5 {
			out = append(out, lintFinding{"L18", "§9.14 第 3 层、UI-22", "render",
				fmt.Sprintf("%s @%d：%q 越出窗口（%+v）", screen, width, text, r)})
		}
	}
	return out
}

// L17：元素表声明的形式必须和渲染一致——`icon+text` 要有图标墨迹与可见文字，
// `icon` 不得有可见文字，`text` 必须有可见文字，`style` 要有墨迹。找不到元素
// 就报"清单与实现不一致"，不静默跳过（这是评审能静默绕过的那条通道）。
func lintElementForms(t *testing.T, elements *elementDoc) []lintFinding {
	t.Helper()
	var out []lintFinding
	for _, it := range elements.Items {
		tt := testerForWhen(t, it.When)
		if tt == nil {
			out = append(out, lintFinding{"L17", "UI-23、D5（" + it.ID + "）", "render",
				"未知的 when：" + it.When})
			continue
		}
		out = append(out, checkElementForm(t, tt, it)...)
	}
	return out
}

// testerForWhen 按清单里的 when 造出该元素出现的界面。
func testerForWhen(t *testing.T, when string) *gui.Tester {
	t.Helper()
	app := NewApp(Options{State: StateDefault})
	switch when {
	case "always":
	case "branch":
		app.branch, app.branches = branchSamples()
	case "goal-edit":
		app.goalEditing = true
	case "tasks":
		app.rightOpen, app.rightTab = true, 2
	case "waiting":
		app = NewApp(Options{State: StateWaiting})
	case "failed":
		app = NewApp(Options{State: StateFailed})
	case "empty":
		app = NewApp(Options{State: StateEmpty})
	case "running":
		app = NewApp(Options{State: StateRunning})
	case "settings":
		app.screen, app.settingsSection = ScreenSettings, 0
	case "settings-rules":
		app.screen, app.settingsSection = ScreenSettings, 2
	case "term":
		app.rightOpen, app.rightTab = true, 4
	case "term-running":
		app.rightOpen, app.rightTab = true, 4
		app.termState = termRunning
		app.termLines = termSamples(termRunning)
	case "term-exited":
		app.rightOpen, app.rightTab = true, 4
		app.termState = termExited
		app.termLines = termSamples(termExited)
	case "settings-ext":
		app.screen, app.settingsSection, app.extTab, app.extState = ScreenSettings, 3, 1, extLoaded
	case "settings-ext-empty":
		app.screen, app.settingsSection, app.extState = ScreenSettings, 3, extEmpty
	case "settings-ext-failed":
		app.screen, app.settingsSection, app.extState = ScreenSettings, 3, extFailed
	default:
		return nil
	}
	return gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// checkElementForm 是一条元素的判定（种子自检直接调它）。
func checkElementForm(t *testing.T, tt *gui.Tester, it elementEntry) []lintFinding {
	t.Helper()
	if !it.Checkable {
		return nil
	}
	info, ok := tt.Access(it.Probe)
	if !ok {
		return []lintFinding{{"L17", "UI-23、D5（" + it.ID + "）", "render",
			"清单与实现不一致：找不到「" + it.Element + "」（probe=" + it.Probe + "）"}}
	}
	fail := func(msg string) []lintFinding {
		return []lintFinding{{"L17", "UI-23、D5（" + it.ID + "）", "render", it.Element + "：" + msg}}
	}
	ink := hasInk(tt.Image(), info.Bounds)

	switch it.Form {
	case "icon+text":
		textRect, ok := visibleTextRect(tt, info.Bounds, it.Text)
		if !ok {
			return fail("声明为图标 + 文字，但找不到可见文字「" + it.Text + "」")
		}
		_ = textRect
		if !iconMarkIn(tt, info.Bounds, it.Mark) {
			return fail("声明为图标 + 文字，但文字旁边没有图标（图标：<mark> 找不到）")
		}
	case "icon":
		if !ink {
			return fail("声明为纯图标，但盒子里没有墨迹")
		}
		if info.Bounds.W > iconOnlyButtonWidth+space2 {
			return fail("声明为纯图标，但盒子比纯图标宽，说明画了字")
		}
	case "text":
		if !hasVisibleText(t, tt, info.Bounds, it.Text) {
			return fail("声明为纯文字，但找不到可见文字「" + it.Text + "」")
		}
	case "style":
		if !ink {
			return fail("声明为样式，但盒子里没有形状墨迹")
		}
	default:
		return fail("清单里的 form 不认识：" + it.Form)
	}
	return nil
}

// visibleTextRect 取 text 作为可见文本在 box 里的位置（不是只存在于可访问名）。
func visibleTextRect(tt *gui.Tester, box gui.Rect, text string) (gui.Rect, bool) {
	if text == "" {
		return gui.Rect{}, false
	}
	r, ok := tt.Find(text)
	if !ok {
		return gui.Rect{}, false
	}
	// 可见文字要在元素盒内（允许 1px 抗锯齿误差）。
	if r.X < box.X-1 || r.Y < box.Y-1 || r.X > box.X+box.W || r.Y > box.Y+box.H {
		return gui.Rect{}, false
	}
	return r, true
}

// hasVisibleText 是 visibleTextRect 的布尔版。
func hasVisibleText(t *testing.T, tt *gui.Tester, box gui.Rect, text string) bool {
	t.Helper()
	_, ok := visibleTextRect(tt, box, text)
	return ok
}

// iconMarkIn 判断元素盒里有没有该图标：图标由 iconGlyph 画，带一个「图标：<mark>」
// 标记（对辅助技术隐藏）。用结构判而不是像素判——文字本身的墨迹会掩盖"图标没了"。
func iconMarkIn(tt *gui.Tester, box gui.Rect, mark string) bool {
	if mark == "" {
		return false
	}
	r, ok := tt.Find("图标：" + mark)
	if !ok {
		return false
	}
	return r.X >= box.X-1 && r.Y >= box.Y-1 && r.X <= box.X+box.W && r.Y <= box.Y+box.H
}

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

// elementEntry 是 U2 的元素表清单（U2-elements.json）：元素 → 形式 → 理由。
type elementEntry struct {
	ID        string `json:"id"`
	Element   string `json:"element"`
	Item      string `json:"item"`
	Control   string `json:"control"`
	Form      string `json:"form"`
	Reason    string `json:"reason"`
	Icon      string `json:"icon"`
	Doc       string `json:"doc"`
	When      string `json:"when"`
	Probe     string `json:"probe"`
	Text      string `json:"text"`
	Mark      string `json:"mark"`
	Component string `json:"component"`
	Symbol    string `json:"symbol"`
	Checkable bool   `json:"checkable"`
	WhyNot    string `json:"why_not_checkable"`
}

type elementDoc struct {
	Items []elementEntry `json:"items"`
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
