package account

import (
	"context"
	"fmt"
	"time"
)

// LoginPhase 是一次浏览器登录尝试在客户端本地的状态。它描述的是「这次发起」走到
// 哪一步，不是账号事实；账号是否登录只由服务端读取结果决定。
type LoginPhase string

const (
	// LoginIdle 表示没有进行中的登录尝试。
	LoginIdle LoginPhase = "idle"
	// LoginWaiting 表示已打开浏览器，正在等待环回回调。
	LoginWaiting LoginPhase = "waiting"
	// LoginCompleted 表示回调已交给授权来源完成，授权本身成功。
	LoginCompleted LoginPhase = "completed"
	// LoginFailed 表示发起、回调校验或完成交换失败。
	LoginFailed LoginPhase = "failed"
	// LoginCanceled 表示用户或调用方取消了这次登录。
	LoginCanceled LoginPhase = "canceled"
	// LoginTimedOut 表示等待回调超时。
	LoginTimedOut LoginPhase = "timed_out"
)

// loginEvent 驱动登录尝试状态机的事件。它不出包。
type loginEvent string

const (
	loginEventBegin     loginEvent = "begin"
	loginEventCompleted loginEvent = "completed"
	loginEventFailed    loginEvent = "failed"
	loginEventCanceled  loginEvent = "canceled"
	loginEventTimedOut  loginEvent = "timed_out"
)

// loginTransitions 是登录尝试的迁移表：
//
//	空闲 + 发起 → 等待回调
//	等待回调 + 完成 → 完成
//	等待回调 + 失败 → 失败
//	等待回调 + 取消 → 已取消
//	等待回调 + 超时 → 已超时
//	完成/失败/已取消/已超时 + 发起 → 等待回调（允许重新登录）
var loginTransitions = map[LoginPhase]map[loginEvent]LoginPhase{
	LoginIdle: {
		loginEventBegin: LoginWaiting,
	},
	LoginWaiting: {
		loginEventCompleted: LoginCompleted,
		loginEventFailed:    LoginFailed,
		loginEventCanceled:  LoginCanceled,
		loginEventTimedOut:  LoginTimedOut,
	},
	LoginCompleted: {
		loginEventBegin: LoginWaiting,
	},
	LoginFailed: {
		loginEventBegin: LoginWaiting,
	},
	LoginCanceled: {
		loginEventBegin: LoginWaiting,
	},
	LoginTimedOut: {
		loginEventBegin: LoginWaiting,
	},
}

// allLoginPhases 是迁移表允许出现的状态全集。
var allLoginPhases = []LoginPhase{
	LoginIdle,
	LoginWaiting,
	LoginCompleted,
	LoginFailed,
	LoginCanceled,
	LoginTimedOut,
}

// allLoginEvents 是迁移表允许出现的事件全集。
var allLoginEvents = []loginEvent{
	loginEventBegin,
	loginEventCompleted,
	loginEventFailed,
	loginEventCanceled,
	loginEventTimedOut,
}

// validateLoginTable 校验登录尝试迁移表：来源与目标都在状态集合内、事件都在事件
// 集合内；空闲只能作为入口；等待回调必须有四种结束方式；结束态只能重新发起。
// 启动时不合法即拒绝创建，不带病运行。
func validateLoginTable(table map[LoginPhase]map[loginEvent]LoginPhase) error {
	phases := make(map[LoginPhase]bool, len(allLoginPhases))
	for _, phase := range allLoginPhases {
		phases[phase] = true
	}
	events := make(map[loginEvent]bool, len(allLoginEvents))
	for _, event := range allLoginEvents {
		events[event] = true
	}

	for _, from := range allLoginPhases {
		edges, listed := table[from]
		if !listed {
			continue
		}
		for event, to := range edges {
			if !events[event] {
				return fmt.Errorf("登录尝试状态迁移表不合法：%q 上的未知事件 %q", from, event)
			}
			if !phases[to] {
				return fmt.Errorf("登录尝试状态迁移表不合法：%q 经 %q 到未知状态 %q", from, event, to)
			}
		}
	}

	idle, ok := table[LoginIdle]
	if !ok {
		return fmt.Errorf("登录尝试状态迁移表不合法：缺少「空闲」入口")
	}
	if _, ok := idle[loginEventBegin]; !ok {
		return fmt.Errorf("登录尝试状态迁移表不合法：「空闲」缺少发起迁移")
	}
	for event := range idle {
		if event != loginEventBegin {
			return fmt.Errorf("登录尝试状态迁移表不合法：「空闲」只能作为入口，出现了事件 %q", event)
		}
	}

	waiting, ok := table[LoginWaiting]
	if !ok {
		return fmt.Errorf("登录尝试状态迁移表不合法：缺少「等待回调」状态")
	}
	for _, event := range []loginEvent{
		loginEventCompleted,
		loginEventFailed,
		loginEventCanceled,
		loginEventTimedOut,
	} {
		if _, ok := waiting[event]; !ok {
			return fmt.Errorf("登录尝试状态迁移表不合法：「等待回调」缺少事件 %q 的出边", event)
		}
	}

	for _, terminal := range []LoginPhase{LoginCompleted, LoginFailed, LoginCanceled, LoginTimedOut} {
		edges, ok := table[terminal]
		if !ok {
			return fmt.Errorf("登录尝试状态迁移表不合法：缺少结束状态 %q", terminal)
		}
		for event := range edges {
			if event != loginEventBegin {
				return fmt.Errorf("登录尝试状态迁移表不合法：结束状态 %q 只能重新发起，出现事件 %q", terminal, event)
			}
		}
	}
	return nil
}

// LoginState 是一次登录尝试的只读状态。Reason 只在失败、取消或超时时非空。
type LoginState struct {
	Phase     LoginPhase
	StartedAt time.Time
	Reason    string
}

// LoginState 返回当前登录尝试状态。
func (a *Account) LoginState() LoginState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return LoginState{
		Phase:     a.loginPhase,
		StartedAt: a.loginFrom,
		Reason:    a.loginError,
	}
}

// beginLogin 进入等待回调。已有登录在进行时不接受第二次发起。
func (a *Account) beginLogin(cancel context.CancelFunc) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.loginCancel != nil {
		return false
	}
	next, ok := loginTransitions[a.loginPhase][loginEventBegin]
	if !ok {
		return false
	}
	a.loginPhase = next
	a.loginFrom = a.cfg.Now()
	a.loginError = ""
	a.loginCancel = cancel
	return true
}

// advanceLogin 查表迁移登录尝试状态。非法迁移丢弃事件、状态不变、不向用户报错。
func (a *Account) advanceLogin(event loginEvent, reason string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	next, ok := loginTransitions[a.loginPhase][event]
	if !ok {
		return
	}
	a.loginPhase = next
	if event == loginEventBegin {
		a.loginError = ""
		return
	}
	a.loginError = reason
}

// finishLogin 结束一次登录尝试，释放取消句柄。
func (a *Account) finishLogin() {
	a.mu.Lock()
	a.loginCancel = nil
	a.mu.Unlock()
}
