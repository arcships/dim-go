package account

import (
	"context"
	"strings"
	"testing"
)

// TestLoginTransitionTableMatchesDocument 确认迁移表覆盖文档里的每一行，并且
// 等待回调有四种结束方式。
func TestLoginTransitionTableMatchesDocument(t *testing.T) {
	if err := validateLoginTable(loginTransitions); err != nil {
		t.Fatalf("默认登录尝试迁移表不合法：%v", err)
	}
	for _, event := range []loginEvent{
		loginEventCompleted,
		loginEventFailed,
		loginEventCanceled,
		loginEventTimedOut,
	} {
		if _, ok := loginTransitions[LoginWaiting][event]; !ok {
			t.Fatalf("等待回调缺少事件 %q", event)
		}
	}
	for _, phase := range []LoginPhase{LoginCompleted, LoginFailed, LoginCanceled, LoginTimedOut} {
		edges := loginTransitions[phase]
		if len(edges) != 1 {
			t.Fatalf("结束状态 %q 只应能重新发起：%+v", phase, edges)
		}
		if _, ok := edges[loginEventBegin]; !ok {
			t.Fatalf("结束状态 %q 缺少重新发起", phase)
		}
	}
}

// TestBrokenLoginTablesAreRejected 确认坏表在启动时被拒绝并点名原因。
func TestBrokenLoginTablesAreRejected(t *testing.T) {
	clone := func() map[LoginPhase]map[loginEvent]LoginPhase {
		copied := make(map[LoginPhase]map[loginEvent]LoginPhase, len(loginTransitions))
		for phase, edges := range loginTransitions {
			copied[phase] = make(map[loginEvent]LoginPhase, len(edges))
			for event, next := range edges {
				copied[phase][event] = next
			}
		}
		return copied
	}

	cases := []struct {
		name   string
		mutate func(map[LoginPhase]map[loginEvent]LoginPhase)
		want   string
	}{
		{
			name:   "缺少空闲入口",
			mutate: func(table map[LoginPhase]map[loginEvent]LoginPhase) { delete(table, LoginIdle) },
			want:   "缺少「空闲」入口",
		},
		{
			name: "空闲出现额外事件",
			mutate: func(table map[LoginPhase]map[loginEvent]LoginPhase) {
				table[LoginIdle][loginEventFailed] = LoginFailed
			},
			want: "只能作为入口",
		},
		{
			name:   "等待回调缺少超时",
			mutate: func(table map[LoginPhase]map[loginEvent]LoginPhase) { delete(table[LoginWaiting], loginEventTimedOut) },
			want:   "缺少事件",
		},
		{
			name: "结束状态出现结束事件",
			mutate: func(table map[LoginPhase]map[loginEvent]LoginPhase) {
				table[LoginFailed][loginEventFailed] = LoginFailed
			},
			want: "只能重新发起",
		},
		{
			name: "未知事件",
			mutate: func(table map[LoginPhase]map[loginEvent]LoginPhase) {
				table[LoginWaiting][loginEvent("unknown")] = LoginFailed
			},
			want: "未知事件",
		},
		{
			name: "未知目标状态",
			mutate: func(table map[LoginPhase]map[loginEvent]LoginPhase) {
				table[LoginWaiting][loginEventCompleted] = LoginPhase("nowhere")
			},
			want: "未知状态",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			table := clone()
			testCase.mutate(table)
			err := validateLoginTable(table)
			if err == nil {
				t.Fatal("坏表没有被拒绝")
			}
			if !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("错误没有点名原因：%v", err)
			}
		})
	}
}

// TestNewRejectsMissingStatusReader 确认没有账号状态读取来源时不能创建实例。
func TestNewRejectsMissingStatusReader(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("缺少状态读取来源时应拒绝创建")
	}
}

// TestAdvanceLoginDropsIllegalMoves 确认非法迁移被丢弃：状态不变、原因不变。
func TestAdvanceLoginDropsIllegalMoves(t *testing.T) {
	account := newTestAccount(t, Config{})
	account.advanceLogin(loginEventCompleted, "不该发生")
	state := account.LoginState()
	if state.Phase != LoginIdle {
		t.Fatalf("空闲状态下不应接受完成事件：%+v", state)
	}
	if state.Reason != "" {
		t.Fatalf("丢弃的事件不应留下原因：%q", state.Reason)
	}

	// 正常发起后再走取消，确认合法迁移仍然生效。
	ctx, cancel := context.WithCancel(context.Background())
	attempt := &loginAttempt{
		provider: Provider{ID: "provider-browser", LoginMethods: []LoginMethod{LoginMethodBrowser}},
		method:   LoginMethodBrowser,
		ctx:      ctx,
		cancel:   cancel,
	}
	if !account.beginLogin(attempt) {
		t.Fatal("空闲状态应能发起登录")
	}
	account.finishAttempt(attempt, LoginCanceled, "测试取消")
	state = account.LoginState()
	if state.Phase != LoginCanceled || state.Reason != "测试取消" {
		t.Fatalf("取消后的登录尝试状态不符：%+v", state)
	}
	if state.ProviderID != "provider-browser" || state.Method != LoginMethodBrowser {
		t.Fatalf("登录尝试没有记录 provider 与方式：%+v", state)
	}
}
