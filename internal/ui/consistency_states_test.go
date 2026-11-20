package ui

import (
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 一致性与四态的可执行证据（docs/cleanliness.md §3）：
//   - 同一事实在不同屏用同一个词（骨架读屏名、权限条重试名、空态标题）；
//   - 读取中不与空态混用（连接建立中写「正在连接…」）。

// TestSidebarConnectingIsNotAnEmptyList：连接还在建立时左栏写「正在连接…」，
// 不写「暂无会话」——读到了、没有会话，与还没读到是两态。
func TestSidebarConnectingIsNotAnEmptyList(t *testing.T) {
	src := &fakeSource{conn: Connection{Phase: "connecting"}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	if showsText(tt, "正在连接…") == false {
		t.Fatalf("连接建立中没有写「正在连接…」：%v", tt.Texts())
	}
	if showsText(tt, "暂无会话") {
		t.Fatalf("连接建立中把读取中画成了空态「暂无会话」：%v", tt.Texts())
	}
}

// TestTimelineConnectingIsNotAnEmptyConversation：连接建立中时间线写「正在连接…」，
// 不写「当前会话还没有消息」——那时还没有会话可读。
func TestTimelineConnectingIsNotAnEmptyConversation(t *testing.T) {
	src := &fakeSource{conn: Connection{Phase: "connecting"}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	if !showsText(tt, "正在连接…") {
		t.Fatalf("连接建立中没有写「正在连接…」：%v", tt.Texts())
	}
	if showsText(tt, "当前会话还没有消息") {
		t.Fatalf("连接建立中把读取中画成了「当前会话还没有消息」：%v", tt.Texts())
	}
}

// TestPanelSkeletonNamesThePanel：文件页与改动页共用骨架，但读屏分别说
// 「正在加载文件树」与「正在加载改动」，不沿用同一个名字。
func TestPanelSkeletonNamesThePanel(t *testing.T) {
	cases := []struct {
		name    string
		tab     int
		src     func() *fakePanelSource
		want    string
		notWant string
	}{
		{
			name: "文件页",
			tab:  tabFiles,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.trees[""] = FileTreeSnapshot{Phase: PanelReadLoading}
				return src
			},
			want:    "正在加载文件树",
			notWant: "正在加载改动",
		},
		{
			name: "改动页",
			tab:  tabChanges,
			src: func() *fakePanelSource {
				src := newFakePanelSource("/work")
				src.changes = ChangeSet{Phase: PanelReadLoading}
				return src
			},
			want:    "正在加载改动",
			notWant: "正在加载文件树",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, tt := rightWiredTester(t, tc.tab, tc.src())
			tt.Frame()
			if !showsText(tt, tc.want) {
				t.Fatalf("缺少骨架读屏名 %q：%v", tc.want, tt.Texts())
			}
			if showsText(tt, tc.notWant) {
				t.Fatalf("骨架读屏名串到了另一页 %q：%v", tc.notWant, tt.Texts())
			}
		})
	}
}

// TestPermissionBarRetryNamesThePanel：没有写入权限的警告条两页共用，但「重试」
// 的读屏名分别是「重试读取文件树」与「重试读取改动」。
func TestPermissionBarRetryNamesThePanel(t *testing.T) {
	files := newFakePanelSource("/work")
	tree := readyTree(FileEntrySnapshot{Name: "main.go"})
	tree.Denied = true
	files.trees[""] = tree
	_, tt := rightWiredTester(t, tabFiles, files)
	tt.Frame()
	if !showsText(tt, "重试读取文件树") {
		t.Fatalf("文件页权限条没有「重试读取文件树」：%v", tt.Texts())
	}

	changes := newFakePanelSource("/work")
	changes.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main", Restricted: true}
	_, tt = rightWiredTester(t, tabChanges, changes)
	tt.Frame()
	if !showsText(tt, "重试读取改动") {
		t.Fatalf("改动页权限条没有「重试读取改动」：%v", tt.Texts())
	}
	if showsText(tt, "重试读取文件树") {
		t.Fatalf("改动页权限条沿用了文件页的读屏名：%v", tt.Texts())
	}
}

// TestNoPanelsSayNotOpenSessionThreeWays：七个面板的「未打开会话」空态标题
// 只有一个叫法。
func TestNoPanelsSayNotOpenSessionThreeWays(t *testing.T) {
	for _, tab := range []int{tabFiles, tabChanges, tabTasks, tabGoal, tabSideChat, tabSubagent, tabTerminal} {
		app := NewApp(Options{State: StateDefault, Source: wiredNoSession()})
		app.rightOpen, app.rightTab = true, tab
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		tt.Frame()
		texts := strings.Join(tt.Texts(), "\n")
		if strings.Contains(texts, "还没有打开会话") {
			t.Fatalf("标签 %d 的空态仍写「还没有打开会话」：%v", tab, tt.Texts())
		}
	}
}

// TestSidebarDisconnectedIsNotAnEmptyList：断线且目录为空时写「连接已断开」，
// 不写「暂无会话」——读不到不等于读到了没有。
func TestSidebarDisconnectedIsNotAnEmptyList(t *testing.T) {
	src := &fakeSource{conn: Connection{Phase: "disconnected", Reason: "连接被网络中断"}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	if !showsText(tt, "连接已断开") {
		t.Fatalf("断线没有在左栏写「连接已断开」：%v", tt.Texts())
	}
	if showsText(tt, "暂无会话") {
		t.Fatalf("断线把读不到画成了空态「暂无会话」：%v", tt.Texts())
	}
}

// TestInlineRenameSharesDialogRules：行内快捷改名与重命名弹窗同一套名字判断
// （控制字符留在编辑态并写原因；没改名字不发请求）——同一个名字不在弹窗被拒、
// 在行内被接受。
func TestInlineRenameSharesDialogRules(t *testing.T) {
	stub := &renameTestIntents{renamed: make(chan [2]string, 1)}
	app := NewApp(Options{State: StateDefault, Source: wiredSource(), Intents: stub})

	app.renaming, app.renameDraft = 0, "会话\n名"
	app.saveRename(0)
	if app.renaming != 0 {
		t.Fatalf("控制字符没有留在编辑态：renaming=%d", app.renaming)
	}
	if app.headerNote != renameCopyCtrlSession {
		t.Fatalf("控制字符没有写原因，headerNote=%q", app.headerNote)
	}
	select {
	case got := <-stub.renamed:
		t.Fatalf("控制字符仍发了 RenameSession：%v", got)
	default:
	}

	app.headerNote = ""
	app.renameDraft = app.sessionName(0, app.sessionIDAt(0))
	app.saveRename(0)
	if app.renaming != -1 {
		t.Fatalf("名字没改仍留在编辑态：renaming=%d", app.renaming)
	}
	select {
	case got := <-stub.renamed:
		t.Fatalf("名字没改仍发了 RenameSession：%v", got)
	default:
	}
}
