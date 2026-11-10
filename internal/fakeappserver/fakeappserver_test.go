package fakeappserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
	"github.com/arcships/dimagent-mygo/internal/appserver/transport"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
)

func loadHistory(t *testing.T) fakeappserver.HistoryFixture {
	t.Helper()
	fixture, err := fakeappserver.LoadHistoryFile("testdata/history-basic.json")
	if err != nil {
		t.Fatalf("加载历史样例: %v", err)
	}
	return fixture
}

func loadNotifications(t *testing.T) fakeappserver.NotificationFixture {
	t.Helper()
	fixture, err := fakeappserver.LoadNotificationsFile("testdata/notifications-basic.json")
	if err != nil {
		t.Fatalf("加载通知样例: %v", err)
	}
	return fixture
}

func startServer(t *testing.T, script fakeappserver.Script) *fakeappserver.Server {
	t.Helper()
	server, err := fakeappserver.Start(script)
	if err != nil {
		t.Fatalf("启动假服务端: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server
}

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("等待超时: %s", what)
}

func countMethod(methods []string, method string) int {
	count := 0
	for _, name := range methods {
		if name == method {
			count++
		}
	}
	return count
}

func testEvent(sequence int64) protocol.RawEvent {
	return protocol.RawEvent{
		ID:       fmt.Sprintf("live-%d", sequence),
		Sequence: sequence,
		CreateAt: 1762732809000,
		Type:     "text",
		Name:     "assistant",
		Data:     json.RawMessage(`{"text":"live"}`),
	}
}

// 路径一：正常握手与会话。用真实客户端（daemon + session）走完 hello、目录、
// 提交、停止、attach、初始历史、主动通知与 detach，并核对服务端记录的请求。
func TestNormalHandshakeAndSession(t *testing.T) {
	fixture := loadHistory(t)
	server := startServer(t, fakeappserver.Script{
		Features: fixture.Features,
		Sessions: []fakeappserver.Session{fixture.Session()},
		Replies: []fakeappserver.Reply{
			{Method: "session.catalog", Result: map[string]any{
				"sessions": []any{map[string]any{"sessionId": fixture.SessionID}},
			}},
			{Method: "session.submit", Result: map[string]any{
				"inputId":   "input-1",
				"sessionId": fixture.SessionID,
			}},
			{Method: "session.stop", Result: map[string]any{"stopped": true}},
		},
	})

	supervisor, err := daemon.New(daemon.Config{DataDir: server.DataDir()})
	if err != nil {
		t.Fatalf("创建连接监督者: %v", err)
	}
	defer supervisor.Close(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := supervisor.Connect(ctx)
	if err != nil {
		t.Fatalf("握手失败: %v", err)
	}
	if conn.Hello.ProtocolVersion != protocol.ProtocolVersion {
		t.Fatalf("hello 协议版本 %d，期望 %d", conn.Hello.ProtocolVersion, protocol.ProtocolVersion)
	}
	if !conn.HasFeature("session.history.latestTurn") {
		t.Fatalf("hello 未声明 session.history.latestTurn: %v", conn.Hello.Features)
	}

	hellos := server.RequestsFor("hello")
	if len(hellos) != 1 {
		t.Fatalf("hello 请求次数 %d，期望 1", len(hellos))
	}
	var helloParams protocol.HelloParams
	if err := json.Unmarshal(hellos[0].Params, &helloParams); err != nil {
		t.Fatalf("解析 hello 参数: %v", err)
	}
	if helloParams.ProtocolVersion != protocol.ProtocolVersion || helloParams.ClientKind != "other" {
		t.Fatalf("hello 参数不符合客户端约定: %+v", helloParams)
	}

	var catalog struct {
		Sessions []struct {
			SessionID string `json:"sessionId"`
		} `json:"sessions"`
	}
	if err := conn.Client.CallInto(ctx, "session.catalog", map[string]any{"limit": 10}, &catalog); err != nil {
		t.Fatalf("session.catalog: %v", err)
	}
	if len(catalog.Sessions) != 1 || catalog.Sessions[0].SessionID != fixture.SessionID {
		t.Fatalf("目录回执不符合脚本: %+v", catalog)
	}

	var submit struct {
		InputID string `json:"inputId"`
	}
	if err := conn.Client.CallInto(ctx, "session.submit", map[string]any{
		"sessionId": fixture.SessionID,
		"text":      "hello",
	}, &submit); err != nil {
		t.Fatalf("session.submit: %v", err)
	}
	if submit.InputID != "input-1" {
		t.Fatalf("提交回执 inputId=%q，期望 input-1", submit.InputID)
	}

	var stop struct {
		Stopped bool `json:"stopped"`
	}
	if err := conn.Client.CallInto(ctx, "session.stop", map[string]any{"sessionId": fixture.SessionID}, &stop); err != nil {
		t.Fatalf("session.stop: %v", err)
	}
	if !stop.Stopped {
		t.Fatal("停止回执未按脚本返回 stopped=true")
	}

	// 未在脚本里声明的方法按能力缺失处理，客户端应归为不可重试。
	err = conn.Client.CallInto(ctx, "session.scheduler.list", struct{}{}, nil)
	if !failure.Is(err, failure.KindNonRetryable) {
		t.Fatalf("未知方法分类错误: %v", err)
	}
	var rpcErr *protocol.RPCError
	if !errors.As(err, &rpcErr) || rpcErr.AppError() != "unknown_method" {
		t.Fatalf("未知方法 appError 不符合约定: %v", err)
	}

	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	defer manager.Close()

	live := make(chan string, 4)
	snapshot, lease, err := manager.Acquire(ctx, fixture.SessionID, session.Listener{
		OnEvent: func(event session.Event) { live <- event.Method },
	})
	if err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	defer lease.Release()

	if snapshot.Head != fixture.Head {
		t.Fatalf("头部序号 %d，期望 %d", snapshot.Head, fixture.Head)
	}
	gotSequences := make([]int64, 0, len(snapshot.History.Events))
	for _, event := range snapshot.History.Events {
		gotSequences = append(gotSequences, event.Sequence)
	}
	if want := fixture.LatestTurnSequences(); !slices.Equal(gotSequences, want) {
		t.Fatalf("初始历史序号 %v，期望最近一轮 %v", gotSequences, want)
	}

	historyRequests := server.RequestsFor("session.history")
	if len(historyRequests) != 1 {
		t.Fatalf("初始历史请求次数 %d，期望 1", len(historyRequests))
	}
	var historyParams protocol.SessionHistoryParams
	if err := json.Unmarshal(historyRequests[0].Params, &historyParams); err != nil {
		t.Fatalf("解析历史参数: %v", err)
	}
	if historyParams.LatestTurn == nil || !*historyParams.LatestTurn {
		t.Fatalf("初始历史未声明 latestTurn: %+v", historyParams)
	}
	if historyParams.Through == nil || *historyParams.Through != fixture.Head {
		t.Fatalf("初始历史 through=%v，期望头部 %d", historyParams.Through, fixture.Head)
	}

	// 主动推送通知：不依赖任何请求，直接到达会话流监听器。
	if err := server.Notify("session.state", map[string]any{
		"sessionId": fixture.SessionID,
		"state":     "ready",
	}); err != nil {
		t.Fatalf("推送通知: %v", err)
	}
	select {
	case method := <-live:
		if method != "session.state" {
			t.Fatalf("收到通知 %q，期望 session.state", method)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("主动推送的通知未到达")
	}

	lease.Release()
	waitFor(t, "最后一个持有者释放后 detach", func() bool {
		return len(server.RequestsFor("session.detach")) == 1
	})

	methods := server.Methods()
	for _, want := range []string{
		"hello", "session.catalog", "session.submit", "session.stop",
		"session.attach", "session.history", "session.detach",
	} {
		if countMethod(methods, want) == 0 {
			t.Fatalf("服务端没有记录到 %s，实际方法序列: %v", want, methods)
		}
	}
	// 控制连接与会话流各握手一次；同一会话的多个持有者不再新增连接。
	if count := countMethod(methods, "hello"); count != 2 {
		t.Fatalf("hello 请求次数 %d，期望控制连接与会话流各一次共 2 次", count)
	}

	// 收到的每一帧都在记录里，且收发两个方向都有。
	received := server.ReceivedFrames()
	if len(received) == 0 {
		t.Fatal("没有记录到收到的帧")
	}
	for _, frame := range received {
		if frame.Direction != fakeappserver.DirectionReceived || len(frame.Raw) == 0 {
			t.Fatalf("收到帧记录不完整: %+v", frame)
		}
	}
	if len(server.Frames()) <= len(received) {
		t.Fatal("没有记录到服务端发出的帧")
	}
}

// 路径二：断线后重连。会话流断开后按已交付序号重读，重读只拿游标之后的
// 事实，之后恢复实时投递，已交付的事实不重复。
func TestDisconnectAndReconnect(t *testing.T) {
	fixture := loadHistory(t)
	server := startServer(t, fakeappserver.Script{
		Features: fixture.Features,
		Sessions: []fakeappserver.Session{fixture.Session()},
	})

	supervisor, err := daemon.New(daemon.Config{DataDir: server.DataDir()})
	if err != nil {
		t.Fatalf("创建连接监督者: %v", err)
	}
	defer supervisor.Close(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := supervisor.Connect(ctx); err != nil {
		t.Fatalf("握手失败: %v", err)
	}

	manager, err := session.NewManager(session.Config{
		Opener:            supervisor,
		ReconnectAttempts: 5,
		ReconnectDelay:    20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	defer manager.Close()

	live := make(chan int64, 8)
	resyncs := make(chan session.Resync, 2)
	replays := make(chan session.HistoryReplay, 2)
	_, lease, err := manager.Acquire(ctx, fixture.SessionID, session.Listener{
		OnEvent: func(event session.Event) {
			var params protocol.SessionEventParams
			if json.Unmarshal(event.Params, &params) == nil && params.Event.Sequence > 0 {
				live <- params.Event.Sequence
			}
		},
		OnResync:  func(resync session.Resync) { resyncs <- resync },
		OnHistory: func(replay session.HistoryReplay) { replays <- replay },
	})
	if err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	defer lease.Release()

	// 实时事实 7 先交付，断线后的重读起点应是 7。
	if err := server.PushEvent(fixture.SessionID, testEvent(7)); err != nil {
		t.Fatalf("推送实时事实: %v", err)
	}
	select {
	case sequence := <-live:
		if sequence != 7 {
			t.Fatalf("实时事实序号 %d，期望 7", sequence)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("实时事实未到达")
	}

	// 断线期间服务端又落了一条事实 8，只能靠重读拿到。
	if err := server.AppendEvent(fixture.SessionID, testEvent(8)); err != nil {
		t.Fatalf("追加断线期间的事实: %v", err)
	}
	if dropped := server.DisconnectStreams(); dropped != 1 {
		t.Fatalf("断开的会话流数量 %d，期望 1", dropped)
	}

	select {
	case resync := <-resyncs:
		if resync.FromSequence != 7 {
			t.Fatalf("重同步起点 %d，期望 7", resync.FromSequence)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("没有收到重同步通知")
	}
	select {
	case replay := <-replays:
		if len(replay.History.Events) != 1 || replay.History.Events[0].Sequence != 8 {
			t.Fatalf("重读结果 %+v，期望只有序号 8", replay.History.Events)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("没有收到按序号重读的结果")
	}

	var reconnectAfter int64
	waitFor(t, "断线后按 after 游标重读", func() bool {
		for _, request := range server.RequestsFor("session.history") {
			var params protocol.SessionHistoryParams
			if json.Unmarshal(request.Params, &params) != nil || params.After == nil {
				continue
			}
			reconnectAfter = *params.After
			return true
		}
		return false
	})
	if reconnectAfter != 7 {
		t.Fatalf("重读游标 after=%d，期望 7", reconnectAfter)
	}
	if attaches := len(server.RequestsFor("session.attach")); attaches < 2 {
		t.Fatalf("重连后 attach 次数 %d，期望至少 2", attaches)
	}

	// 重连之后实时投递恢复，且已交付的 7 与重读的 8 不重复出现在实时通道。
	if err := server.PushEvent(fixture.SessionID, testEvent(9)); err != nil {
		t.Fatalf("重连后推送实时事实: %v", err)
	}
	select {
	case sequence := <-live:
		if sequence != 9 {
			t.Fatalf("重连后的实时事实 %d，期望 9", sequence)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("重连后实时事实未到达")
	}
	time.Sleep(100 * time.Millisecond)
	for {
		select {
		case sequence := <-live:
			t.Fatalf("出现重复或乱序的实时事实: %d", sequence)
		default:
			return
		}
	}
}

// 路径三：版本不符停止重试。第一次握手失败后进入终态，之后不再拨号。
func TestVersionMismatchStopsRetry(t *testing.T) {
	cases := []struct {
		name   string
		script fakeappserver.Script
	}{
		{
			name:   "hello 回执版本号不符",
			script: fakeappserver.Script{ProtocolVersion: protocol.ProtocolVersion - 1},
		},
		{
			name: "hello 以 appError 拒绝",
			script: fakeappserver.Script{HelloError: fakeappserver.AppError(
				-32000, "protocol_version", "unsupported protocol version",
			)},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := startServer(t, testCase.script)
			supervisor, err := daemon.New(daemon.Config{DataDir: server.DataDir()})
			if err != nil {
				t.Fatalf("创建连接监督者: %v", err)
			}
			defer supervisor.Close(nil)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if _, err := supervisor.Connect(ctx); !failure.Is(err, failure.KindVersionIncompatible) {
				t.Fatalf("首次握手错误分类不符: %v", err)
			}
			if _, err := supervisor.Connect(ctx); !failure.Is(err, failure.KindVersionIncompatible) {
				t.Fatalf("第二次连接没有停在版本不兼容: %v", err)
			}
			if accepted := server.Accepted(); accepted != 1 {
				t.Fatalf("版本不兼容后仍发生拨号: 接受连接 %d 次，期望 1", accepted)
			}
			if hellos := len(server.RequestsFor("hello")); hellos != 1 {
				t.Fatalf("hello 请求次数 %d，期望 1", hellos)
			}
		})
	}
}

// 乱序回执：后到的请求先回，先到的请求后回。客户端必须按标识关联，
// 不能把 ping 的回执当成历史回执。
func TestOutOfOrderRepliesAreCorrelated(t *testing.T) {
	fixture := loadHistory(t)
	server := startServer(t, fakeappserver.Script{
		Features: fixture.Features,
		Sessions: []fakeappserver.Session{fixture.Session()},
		Faults: []fakeappserver.Fault{{
			Method: "session.history",
			Delay:  300 * time.Millisecond,
		}},
	})

	raw, err := transport.Dial(context.Background(), server.Endpoint(), time.Second)
	if err != nil {
		t.Fatalf("连接假服务端: %v", err)
	}
	client := protocol.NewClient(raw, protocol.Options{RequestTimeout: 2 * time.Second})
	defer client.Close(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var history protocol.SessionHistoryResult
	historyDone := make(chan error, 1)
	go func() {
		historyDone <- client.CallInto(ctx, "session.history", protocol.SessionHistoryParams{
			SessionID: fixture.SessionID,
			BranchID:  "main",
		}, &history)
	}()
	waitFor(t, "服务端收到历史请求", func() bool {
		return len(server.RequestsFor("session.history")) == 1
	})

	var ping protocol.PingResult
	if err := client.CallInto(ctx, "daemon.ping", struct{}{}, &ping); err != nil {
		t.Fatalf("daemon.ping: %v", err)
	}
	if ping.ServerVersion == "" {
		t.Fatal("ping 回执缺少 serverVersion")
	}
	// ping 的回执先到：历史回执仍在延迟中。
	select {
	case err := <-historyDone:
		t.Fatalf("历史回执先于 ping 完成，未形成乱序: %v", err)
	default:
	}
	if err := <-historyDone; err != nil {
		t.Fatalf("延迟后的历史回执没有正确关联: %v", err)
	}
	if len(history.Events) != len(fixture.Events) {
		t.Fatalf("历史回执事件数 %d，期望 %d", len(history.Events), len(fixture.Events))
	}

	requests := server.Requests()
	if len(requests) != 2 {
		t.Fatalf("请求数 %d，期望 2", len(requests))
	}
	if requests[0].Method != "session.history" || requests[1].Method != "daemon.ping" {
		t.Fatalf("请求到达顺序不符合预期: %s, %s", requests[0].Method, requests[1].Method)
	}
}

// 超大帧：客户端按自己的上限丢弃该帧并报一次错，连接保持同步，后续帧照常交付。
func TestOversizedFrameIsDroppedAndConnectionContinues(t *testing.T) {
	fixture := loadHistory(t)
	server := startServer(t, fakeappserver.Script{
		Features: fixture.Features,
		Sessions: []fakeappserver.Session{fixture.Session()},
		Faults: []fakeappserver.Fault{{
			Method:             "hello",
			OversizeFrameBytes: 4096,
		}},
	})

	raw, err := transport.Dial(context.Background(), server.Endpoint(), time.Second)
	if err != nil {
		t.Fatalf("连接假服务端: %v", err)
	}
	var errorMu sync.Mutex
	var frameErrors []error
	client := protocol.NewClient(raw, protocol.Options{
		MaxFrameBytes:  1024,
		RequestTimeout: 2 * time.Second,
		OnError: func(err error) {
			errorMu.Lock()
			frameErrors = append(frameErrors, err)
			errorMu.Unlock()
		},
	})
	defer client.Close(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notifications := make(chan string, 4)
	client.OnNotification(func(message protocol.Message) { notifications <- message.Method })

	var hello protocol.HelloResult
	if err := client.CallInto(ctx, "hello", protocol.HelloParams{
		ProtocolVersion: protocol.ProtocolVersion,
		ClientKind:      "other",
	}, &hello); err != nil {
		t.Fatalf("超大帧之后的 hello 回执没有被处理: %v", err)
	}
	if hello.InstanceID == "" {
		t.Fatal("hello 回执缺少 instanceId")
	}

	// 超大帧之后连接仍可用：主动推送一条正常通知，客户端必须收到。
	if err := server.Notify("demo.ping", map[string]any{"ok": true}); err != nil {
		t.Fatalf("推送通知: %v", err)
	}
	select {
	case method := <-notifications:
		if method != "demo.ping" {
			t.Fatalf("收到通知 %q，期望 demo.ping", method)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("超大帧之后的正常通知没有到达")
	}

	var ping protocol.PingResult
	if err := client.CallInto(ctx, "daemon.ping", struct{}{}, &ping); err != nil {
		t.Fatalf("超大帧之后的 daemon.ping 失败: %v", err)
	}

	errorMu.Lock()
	defer errorMu.Unlock()
	if len(frameErrors) == 0 {
		t.Fatal("超大帧没有触发解码错误")
	}
	found := false
	for _, frameErr := range frameErrors {
		if !errors.Is(frameErr, protocol.ErrFrameTooLarge) {
			continue
		}
		var tooLarge *protocol.FrameTooLargeError
		if !errors.As(frameErr, &tooLarge) {
			t.Fatalf("超大帧错误类型不符: %T", frameErr)
		}
		if tooLarge.Bytes <= tooLarge.MaxBytes {
			t.Fatalf("超大帧错误字节数不合理: %+v", tooLarge)
		}
		found = true
	}
	if !found {
		t.Fatalf("没有观察到 ErrFrameTooLarge: %v", frameErrors)
	}
}

type observation struct {
	method   string
	sequence int64
}

func (o observation) label() string {
	if o.sequence > 0 {
		return fmt.Sprintf("session.event:%d", o.sequence)
	}
	return o.method
}

func notificationLabel(t *testing.T, notification fakeappserver.FixtureNotification) string {
	t.Helper()
	if notification.Method == "session.event" {
		var params protocol.SessionEventParams
		if err := json.Unmarshal(notification.Params, &params); err != nil {
			t.Fatalf("解析样例通知参数: %v", err)
		}
		return fmt.Sprintf("session.event:%d", params.Event.Sequence)
	}
	return notification.Method
}

// PushEventTimes 注入重复通知：同一事实连发两次，会话流只交付一次，头部只推进一次。
func TestPushEventTimesInjectsDuplicateNotification(t *testing.T) {
	fixture := loadHistory(t)
	server := startServer(t, fakeappserver.Script{
		Features: fixture.Features,
		Sessions: []fakeappserver.Session{fixture.Session()},
	})
	supervisor, err := daemon.New(daemon.Config{DataDir: server.DataDir()})
	if err != nil {
		t.Fatalf("创建连接监督者: %v", err)
	}
	defer supervisor.Close(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := supervisor.Connect(ctx); err != nil {
		t.Fatalf("握手失败: %v", err)
	}
	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	defer manager.Close()

	live := make(chan int64, 4)
	_, lease, err := manager.Acquire(ctx, fixture.SessionID, session.Listener{
		OnEvent: func(event session.Event) {
			var params protocol.SessionEventParams
			if json.Unmarshal(event.Params, &params) == nil && params.Event.Sequence > 0 {
				live <- params.Event.Sequence
			}
		},
	})
	if err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	defer lease.Release()

	if err := server.PushEventTimes(fixture.SessionID, testEvent(7), 2); err != nil {
		t.Fatalf("注入重复通知: %v", err)
	}
	select {
	case sequence := <-live:
		if sequence != 7 {
			t.Fatalf("实时事实序号 %d，期望 7", sequence)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("实时事实未到达")
	}
	select {
	case sequence := <-live:
		t.Fatalf("重复通知被交付: %d", sequence)
	case <-time.After(100 * time.Millisecond):
	}
	if head, err := server.Head(fixture.SessionID); err != nil || head != 7 {
		t.Fatalf("头部序号 %d（错误 %v），期望 7", head, err)
	}
}

// 固定通知样例：实时通知按序号前进，重复与乱序旧序号被丢弃，未知方法原样交付。
func TestNotificationFixtureDuplicateAndOutOfOrder(t *testing.T) {
	historyFixture := loadHistory(t)
	notificationFixture := loadNotifications(t)
	server := startServer(t, fakeappserver.Script{
		Features: historyFixture.Features,
		Sessions: []fakeappserver.Session{historyFixture.Session()},
	})

	supervisor, err := daemon.New(daemon.Config{DataDir: server.DataDir()})
	if err != nil {
		t.Fatalf("创建连接监督者: %v", err)
	}
	defer supervisor.Close(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := supervisor.Connect(ctx); err != nil {
		t.Fatalf("握手失败: %v", err)
	}

	manager, err := session.NewManager(session.Config{Opener: supervisor})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	defer manager.Close()

	observed := make(chan observation, 16)
	_, lease, err := manager.Acquire(ctx, historyFixture.SessionID, session.Listener{
		OnEvent: func(event session.Event) {
			if event.Method == "session.event" {
				var params protocol.SessionEventParams
				if json.Unmarshal(event.Params, &params) == nil && params.Event.Sequence > 0 {
					observed <- observation{method: event.Method, sequence: params.Event.Sequence}
					return
				}
			}
			observed <- observation{method: event.Method}
		},
	})
	if err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	defer lease.Release()

	expectations := make([]string, 0, len(notificationFixture.Notifications))
	for _, notification := range notificationFixture.Notifications {
		if err := server.NotifyRaw(notification.Method, notification.Params); err != nil {
			t.Fatalf("推送样例通知: %v", err)
		}
		switch notification.Expect {
		case fakeappserver.ExpectDelivered:
			want := notificationLabel(t, notification)
			expectations = append(expectations, want)
			select {
			case got := <-observed:
				if got.label() != want {
					t.Fatalf("通知 %q 交付为 %q，期望 %q", notification.Comment, got.label(), want)
				}
			case <-time.After(3 * time.Second):
				t.Fatalf("通知 %q 未交付", notification.Comment)
			}
		case fakeappserver.ExpectDropped:
			time.Sleep(20 * time.Millisecond)
		}
	}
	select {
	case extra := <-observed:
		t.Fatalf("出现样例之外的交付: %q", extra.label())
	case <-time.After(100 * time.Millisecond):
	}

	if !slices.Equal(expectations, []string{"session.event:7", "session.event:8", "vendor.unknown"}) {
		t.Fatalf("样例期望序列与设计不符: %v", expectations)
	}
}

// 故障注入的未知方法：即使脚本给了回执，命中故障的那次也按能力缺失返回，
// 之后同样的请求恢复脚本回执。
func TestInjectedUnknownMethodOverridesScriptedReply(t *testing.T) {
	server := startServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "session.submit", Result: map[string]any{"inputId": "input-1"}},
		},
		Faults: []fakeappserver.Fault{{
			Method:        "session.submit",
			Occurrence:    1,
			UnknownMethod: true,
		}},
	})

	raw, err := transport.Dial(context.Background(), server.Endpoint(), time.Second)
	if err != nil {
		t.Fatalf("连接假服务端: %v", err)
	}
	client := protocol.NewClient(raw, protocol.Options{RequestTimeout: 2 * time.Second})
	defer client.Close(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.CallInto(ctx, "session.submit", map[string]any{}, nil)
	if !failure.Is(err, failure.KindNonRetryable) {
		t.Fatalf("注入的未知方法分类错误: %v", err)
	}
	var rpcErr *protocol.RPCError
	if !errors.As(err, &rpcErr) || rpcErr.AppError() != "unknown_method" {
		t.Fatalf("注入的未知方法 appError 不符: %v", err)
	}

	var submit struct {
		InputID string `json:"inputId"`
	}
	if err := client.CallInto(ctx, "session.submit", map[string]any{}, &submit); err != nil {
		t.Fatalf("第二次提交应恢复脚本回执: %v", err)
	}
	if submit.InputID != "input-1" {
		t.Fatalf("第二次提交回执 inputId=%q，期望 input-1", submit.InputID)
	}
}
