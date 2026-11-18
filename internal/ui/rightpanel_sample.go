package ui

import "strconv"

// 右栏的样例数据：文件树与改动列表。
//
// 样例不是事实：连着 Source 时视图仍读 Source；本产品还没有文件树 / 改动 /
// git 动作的读数（Source 见 source.go），所以两页先用样例把形态画全。文件名
// 只取仓库内相对路径的写法（docs 规则：不写本机目录），目录名是通用的项目名。

// rpNode 是文件树的一个节点：目录带子级，文件带改动（与 changes 里同一指针）。
type rpNode struct {
	name     string
	dir      bool
	children []*rpNode
	change   *rpChange
}

// rpChange 是一条文件改动（清单 D-28–D-72 的行数据）。
type rpChange struct {
	path    string
	added   int
	removed int
	staged  bool
	// status 是时间线改动块用的状态字母（M / A / D / ?）。
	status byte
	diff   []string
}

// rightSample 建一份样例：33 个根目录 + 31 个根文件（与运行时清单的 R 态同
// 数量），外加 1 个已暂存 + 2 个未暂存的改动。
func rightSample() ([]*rpNode, []*rpChange) {
	composer := &rpChange{
		path: "internal/ui/composer.go", added: 2, removed: 1, status: 'M',
		diff: []string{
			"diff --git a/internal/ui/composer.go b/internal/ui/composer.go",
			"--- a/internal/ui/composer.go",
			"+++ b/internal/ui/composer.go",
			"@@ -118,7 +118,9 @@ func (a *App) composer(c *gui.Context) {",
			" \tgui.Column(c).Grow(1).Children(func() {",
			"-\t\ta.queueRow(c)",
			"+\t\tif len(a.queue) > 0 {",
			"+\t\t\ta.queueRow(c)",
			"+\t\t}",
			" \t\ta.composerCard(c)",
			" \t})",
		},
	}
	timeline := &rpChange{
		path: "internal/ui/timeline.go", added: 1, removed: 0, status: 'M',
		diff: []string{
			"diff --git a/internal/ui/timeline.go b/internal/ui/timeline.go",
			"--- a/internal/ui/timeline.go",
			"+++ b/internal/ui/timeline.go",
			"@@ -792,1 +792,2 @@ func headerAvailableWidth(",
			"+\t\t// 状态条占用右栏宽度之后的可用宽。",
		},
	}
	statusbar := &rpChange{
		path: "internal/ui/statusbar.go", added: 1, removed: 0, staged: true, status: 'A',
		diff: []string{
			"diff --git a/internal/ui/statusbar.go b/internal/ui/statusbar.go",
			"new file mode 100644",
			"--- /dev/null",
			"+++ b/internal/ui/statusbar.go",
			"@@ -0,0 +1,1 @@",
			"+\tpackage ui",
		},
	}
	changes := []*rpChange{statusbar, composer, timeline}

	dir := func(name string, children ...*rpNode) *rpNode {
		return &rpNode{name: name, dir: true, children: children}
	}
	file := func(name string, change *rpChange) *rpNode {
		return &rpNode{name: name, change: change}
	}
	plain := func(names ...string) []*rpNode {
		nodes := make([]*rpNode, 0, len(names))
		for _, name := range names {
			nodes = append(nodes, file(name, nil))
		}
		return nodes
	}

	// samples 用来走「加载更多」：30 个子项，先给 20。
	paged := make([]*rpNode, 0, 30)
	for i := 1; i <= 30; i++ {
		paged = append(paged, file(rpSampleChildName(i), nil))
	}

	ui := dir("ui", file("composer.go", composer), file("timeline.go", timeline), file("statusbar.go", statusbar))
	tree := []*rpNode{
		dir("assets", plain("icon.png", "logo.svg")...),
		dir("benchmarks"),
		dir("build"),
		dir("cmd", dir("dimagent"), dir("dimagent-ui")),
		dir("config"),
		dir("docs", dir("domains"), dir("guides"), file("README.md", nil), file("index.md", nil)),
		dir("examples"),
		dir("experiments"),
		dir("fixtures"),
		dir("internal", dir("app"), ui, dir("wire"), file("README.md", nil)),
		dir("labs"),
		dir("lib"),
		dir("logs"),
		dir("media"),
		dir("notes"),
		dir("output"),
		dir("packages"),
		dir("plans"),
		dir("public"),
		dir("release"),
		dir("reports"),
		dir("samples", paged...),
		dir("scripts"),
		dir("server"),
		dir("specs"),
		dir("src"),
		dir("static"),
		dir("templates"),
		dir("tests"),
		dir("tmp"),
		dir("tools"),
		dir("vendor"),
		dir("web"),
	}
	root := plain(
		".DS_Store", ".gitignore", "AGENTS.md", "CHANGELOG.md", "LICENSE",
		"README.md", "REVIEW.md", "TODO.md", "api.json", "build.sh",
		"check.py", "compose.yml", "config.toml", "content.json", "deploy.sh",
		"fib57.py", "index.html", "main.go", "notes.txt", "package.json",
		"plan.md", "prime_test.py", "render-plan.md", "research_report.md", "setup.py",
		"test_render.py", "tsconfig.json", "tts-test.mp3", "ui-preview.html",
		"word-test.docx", "worldcup.md",
	)
	return append(tree, root...), changes
}

// rpSampleChildName 是分页目录里的子项名（f1、f2…的字符序）。
func rpSampleChildName(i int) string {
	return "f" + strconv.Itoa(i) + ".md"
}
