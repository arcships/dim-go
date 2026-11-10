package session_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
)

// fakeConn 是一个最小的会话流对端。
type fakeConn struct {
	conn net.Conn
	mu   sync.Mutex
}

func (c *fakeConn) send(value any) {
	frame, err := protocol.EncodeFrame(value)
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, _ = c.conn.Write(frame)
}

func (c *fakeConn) reply(id protocol.ID, result any) {
	c.send(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func (c *fakeConn) notify(method string, params any) {
	c.send(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func (c *fakeConn) close() { _ = c.conn.Close() }

type requestHandler func(fc *fakeConn, id protocol.ID, method string, params json.RawMessage)

func (c *fakeConn) serve(handle requestHandler) {
	decoder := protocol.NewDecoder(0)
	buf := make([]byte, 4096)
	for {
		n, err := c.conn.Read(buf)
		if n > 0 {
			frames, _ := decoder.Push(buf[:n])
			for _, frame := range frames {
				var request struct {
					ID     json.RawMessage `json:"id"`
					Method string          `json:"method"`
					Params json.RawMessage `json:"params"`
				}
				if json.Unmarshal(frame, &request) != nil || request.Method == "" {
					continue
				}
				handle(c, protocol.ID(request.ID), request.Method, request.Params)
			}
		}
		if err != nil {
			return
		}
	}
}

// fakeOpener 为每次 OpenStream 建立一条 net.Pipe 连接并执行脚本。
type fakeOpener struct {
	mu      sync.Mutex
	opens   int
	scripts []func(fc *fakeConn)
	lastErr error
}

func (o *fakeOpener) OpenStream(_ context.Context) (*daemon.Connection, error) {
	o.mu.Lock()
	o.opens++
	index := o.opens
	var script func(fc *fakeConn)
	if index <= len(o.scripts) {
		script = o.scripts[index-1]
	}
	lastErr := o.lastErr
	o.mu.Unlock()
	if lastErr != nil && index > len(o.scripts) {
		return nil, lastErr
	}
	clientSide, serverSide := net.Pipe()
	fc := &fakeConn{conn: serverSide}
	if script == nil {
		script = func(fc *fakeConn) { fc.close() }
	}
	go script(fc)
	client := protocol.NewClient(clientSide, protocol.Options{RequestTimeout: 2 * time.Second})
	return &daemon.Connection{
		Client: client,
		Hello: protocol.HelloResult{
			InstanceID:      fmt.Sprintf("instance-%d", index),
			ServerVersion:   "test",
			ProtocolVersion: protocol.ProtocolVersion,
			DaemonProfile:   protocol.DaemonProfile{Kind: "mock"},
			Features:        []string{"session.history.latestTurn"},
		},
	}, nil
}

func (o *fakeOpener) opensCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.opens
}

// streamScript 生成一份常规脚本：attach 先发通知，再回执，然后读历史。
type streamScript struct {
	sessionID string
	head      int64
	attach    []int64 // attach 回执之前发出的 session.event 序号
	history   []protocol.RawEvent
	through   int64
	afterSeen *int64
	onAttach  func(fc *fakeConn)
	onHistory func(fc *fakeConn)
}

func (s streamScript) script() func(fc *fakeConn) {
	return func(fc *fakeConn) {
		fc.serve(func(fc *fakeConn, id protocol.ID, method string, params json.RawMessage) {
			switch method {
			case "session.attach":
				for _, sequence := range s.attach {
					fc.notify("session.event", eventNotice(s.sessionID, sequence))
				}
				if s.onAttach != nil {
					s.onAttach(fc)
				}
				fc.reply(id, map[string]any{
					"sessionId": s.sessionID,
					"sequence":  s.head,
					"running":   false,
				})
			case "session.history":
				var historyParams protocol.SessionHistoryParams
				_ = json.Unmarshal(params, &historyParams)
				if s.afterSeen != nil && historyParams.After != nil {
					*s.afterSeen = *historyParams.After
				}
				if s.onHistory != nil {
					s.onHistory(fc)
				}
				fc.reply(id, map[string]any{
					"events":    s.history,
					"through":   s.through,
					"nextAfter": s.through,
					"done":      true,
				})
			case "session.detach":
				fc.reply(id, nil)
			}
		})
	}
}

func event(id string, sequence int64) protocol.RawEvent {
	return protocol.RawEvent{
		ID:       id,
		Sequence: sequence,
		CreateAt: 1,
		Type:     "text",
		Name:     "delta",
	}
}

func eventNotice(sessionID string, sequence int64) map[string]any {
	return map[string]any{
		"sessionId": sessionID,
		"branchId":  "main",
		"event": map[string]any{
			"id":        fmt.Sprintf("event-%d", sequence),
			"sequence":  sequence,
			"create_at": 1,
			"type":      "text",
			"name":      "delta",
		},
	}
}

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestAcquireSubscribesBeforeHistoryAndKeepsEarlyNotification(t *testing.T) {
	sessionID := "s-1"
	live := make(chan int64, 8)
	opener := &fakeOpener{}
	opener.scripts = []func(fc *fakeConn){
		streamScript{
			sessionID: sessionID,
			head:      2,
			attach:    []int64{1, 3}, // 1 已被历史覆盖，3 是早于历史读取的实时事实
			history:   []protocol.RawEvent{event("e1", 1)},
			through:   2,
			onAttach: func(fc *fakeConn) {
				// 历史读完之后再发一条实时事实。
				go func() {
					time.Sleep(20 * time.Millisecond)
					fc.notify("session.event", eventNotice(sessionID, 4))
				}()
			},
		}.script(),
	}
	manager, err := session.NewManager(session.Config{
		Opener:            opener,
		ReconnectAttempts: 2,
		ReconnectDelay:    10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	defer manager.Close()

	snapshot, lease, err := manager.Acquire(context.Background(), sessionID, session.Listener{
		OnEvent: func(event session.Event) {
			var params protocol.SessionEventParams
			if json.Unmarshal(event.Params, &params) == nil {
				live <- params.Event.Sequence
			}
		},
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer lease.Release()

	if snapshot.Head != 2 {
		t.Fatalf("head: %d", snapshot.Head)
	}
	if len(snapshot.History.Events) != 1 || snapshot.History.Events[0].Sequence != 1 {
		t.Fatalf("initial history: %+v", snapshot.History.Events)
	}
	got := []int64{}
	for len(got) < 2 {
		select {
		case sequence := <-live:
			got = append(got, sequence)
		case <-time.After(2 * time.Second):
			t.Fatalf("missing events, got %v", got)
		}
	}
	if got[0] != 3 || got[1] != 4 {
		t.Fatalf("want [3 4], got %v", got)
	}
}

func TestHoldersKeepOneSubscriptionUntilAllReleased(t *testing.T) {
	sessionID := "s-2"
	var detaches int32
	var detachMu sync.Mutex
	script := func(fc *fakeConn) {
		fc.serve(func(fc *fakeConn, id protocol.ID, method string, params json.RawMessage) {
			switch method {
			case "session.attach":
				fc.reply(id, map[string]any{"sessionId": sessionID, "sequence": 0, "running": false})
			case "session.history":
				fc.reply(id, map[string]any{"events": []any{}, "through": 0, "nextAfter": 0, "done": true})
			case "session.detach":
				detachMu.Lock()
				detaches++
				detachMu.Unlock()
				fc.reply(id, nil)
			}
		})
	}
	opener := &fakeOpener{scripts: []func(fc *fakeConn){script, script}}
	manager, err := session.NewManager(session.Config{Opener: opener})
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	defer manager.Close()

	_, first, err := manager.Acquire(context.Background(), sessionID, session.Listener{})
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	_, second, err := manager.Acquire(context.Background(), sessionID, session.Listener{})
	if err != nil {
		t.Fatalf("second acquire: %v", err)
	}
	if opener.opensCount() != 1 {
		t.Fatalf("two holders must share one stream, streams=%d", opener.opensCount())
	}
	first.Release()
	time.Sleep(50 * time.Millisecond)
	detachMu.Lock()
	afterOne := detaches
	detachMu.Unlock()
	if afterOne != 0 {
		t.Fatalf("must not detach while a holder remains: %d", afterOne)
	}
	first.Release() // 重复释放无效
	second.Release()
	waitFor(t, "detach after last holder", func() bool {
		detachMu.Lock()
		defer detachMu.Unlock()
		return detaches == 1
	})
	// 全部释放后再打开是新流。
	if _, lease, err := manager.Acquire(context.Background(), sessionID, session.Listener{}); err != nil {
		t.Fatalf("reacquire: %v", err)
	} else {
		lease.Release()
	}
	waitFor(t, "new stream after release", func() bool { return opener.opensCount() == 2 })
}

func TestReconnectResumesFromSequence(t *testing.T) {
	sessionID := "s-3"
	var firstConnMu sync.Mutex
	var firstConn *fakeConn
	live := make(chan int64, 16)
	resyncs := make(chan session.Resync, 4)
	replays := make(chan session.HistoryReplay, 4)

	var afterSeen int64 = -1
	firstScript := streamScript{
		sessionID: sessionID,
		head:      2,
		history:   []protocol.RawEvent{event("e1", 1), event("e2", 2)},
		through:   2,
		onAttach: func(fc *fakeConn) {
			firstConnMu.Lock()
			firstConn = fc
			firstConnMu.Unlock()
			go func() {
				time.Sleep(20 * time.Millisecond)
				fc.notify("session.event", eventNotice(sessionID, 3))
				fc.notify("session.event", eventNotice(sessionID, 4))
			}()
		},
	}.script()
	secondScript := streamScript{
		sessionID: sessionID,
		head:      5,
		history:   []protocol.RawEvent{event("e5", 5)},
		through:   5,
		afterSeen: &afterSeen,
		onAttach: func(fc *fakeConn) {
			go func() {
				time.Sleep(20 * time.Millisecond)
				fc.notify("session.event", eventNotice(sessionID, 6))
			}()
		},
	}.script()

	opener := &fakeOpener{scripts: []func(fc *fakeConn){firstScript, secondScript}}
	manager, err := session.NewManager(session.Config{
		Opener:            opener,
		ReconnectAttempts: 3,
		ReconnectDelay:    10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	defer manager.Close()

	_, lease, err := manager.Acquire(context.Background(), sessionID, session.Listener{
		OnEvent: func(event session.Event) {
			var params protocol.SessionEventParams
			if json.Unmarshal(event.Params, &params) == nil {
				live <- params.Event.Sequence
			}
		},
		OnResync:  func(resync session.Resync) { resyncs <- resync },
		OnHistory: func(replay session.HistoryReplay) { replays <- replay },
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer lease.Release()

	// 先等实时事实 3、4 送达，再断线，保证重读起点确定。
	got := []int64{}
	for len(got) < 2 {
		select {
		case sequence := <-live:
			got = append(got, sequence)
		case <-time.After(2 * time.Second):
			t.Fatalf("missing live events, got %v", got)
		}
	}
	firstConnMu.Lock()
	conn := firstConn
	firstConnMu.Unlock()
	if conn == nil {
		t.Fatal("first stream was not captured")
	}
	conn.close()

	select {
	case resync := <-resyncs:
		if resync.FromSequence != 4 {
			t.Fatalf("resync from %d, want 4", resync.FromSequence)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no resync")
	}
	select {
	case replay := <-replays:
		if len(replay.History.Events) != 1 || replay.History.Events[0].Sequence != 5 {
			t.Fatalf("replay: %+v", replay.History.Events)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no history replay")
	}
	waitFor(t, "history after cursor", func() bool { return afterSeen == 4 })

	select {
	case sequence := <-live:
		if sequence != 6 {
			t.Fatalf("live after reconnect: %d", sequence)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no live event after reconnect")
	}
	// 已交付的事件不会重复：再等一小会儿确认没有 3/4/5 重放。
	time.Sleep(100 * time.Millisecond)
	for {
		select {
		case sequence := <-live:
			t.Fatalf("unexpected duplicate event %d", sequence)
		default:
			return
		}
	}
}

func TestVersionIncompatibleStopsReconnecting(t *testing.T) {
	sessionID := "s-4"
	first := streamScript{
		sessionID: sessionID,
		head:      1,
		history:   []protocol.RawEvent{event("e1", 1)},
		through:   1,
		onAttach: func(fc *fakeConn) {
			go func() {
				time.Sleep(20 * time.Millisecond)
				fc.close()
			}()
		},
	}.script()
	opener := &fakeOpener{scripts: []func(fc *fakeConn){first}}
	opener.lastErr = failure.New(failure.KindVersionIncompatible, "test", nil)
	manager, err := session.NewManager(session.Config{
		Opener:            opener,
		ReconnectAttempts: 5,
		ReconnectDelay:    10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	defer manager.Close()

	fatal := make(chan error, 1)
	_, lease, err := manager.Acquire(context.Background(), sessionID, session.Listener{
		OnFatal: func(err error) { fatal <- err },
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer lease.Release()

	select {
	case err := <-fatal:
		if !failure.Is(err, failure.KindVersionIncompatible) {
			t.Fatalf("fatal error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("version incompatibility did not stop the stream")
	}
	time.Sleep(50 * time.Millisecond)
	if opens := opener.opensCount(); opens != 2 {
		t.Fatalf("must stop after the incompatible attempt, opens=%d", opens)
	}
}

func TestSessionCallsRouteWithSessionID(t *testing.T) {
	sessionID := "s-7"
	var mu sync.Mutex
	seenSessionIDs := []string{}
	record := func(params json.RawMessage) {
		var envelope struct {
			SessionID string `json:"sessionId"`
		}
		_ = json.Unmarshal(params, &envelope)
		mu.Lock()
		seenSessionIDs = append(seenSessionIDs, envelope.SessionID)
		mu.Unlock()
	}
	script := func(fc *fakeConn) {
		fc.serve(func(fc *fakeConn, id protocol.ID, method string, params json.RawMessage) {
			record(params)
			switch method {
			case "session.attach":
				fc.reply(id, map[string]any{"sessionId": sessionID, "sequence": 0, "running": false})
			case "session.history":
				fc.reply(id, map[string]any{"events": []any{}, "through": 0, "nextAfter": 0, "done": true})
			case "session.methods":
				fc.reply(id, map[string]any{"methods": []any{
					map[string]any{"name": "attachment.put", "mutating": true},
					map[string]any{"name": "session.input.pending", "mutating": false},
				}})
			case "session.input_lookup":
				fc.reply(id, map[string]any{"status": "unknown"})
			case "session.input.pending":
				fc.reply(id, map[string]any{"branchId": "main", "through": 0, "inputs": []any{}, "next": nil})
			case "attachment.put":
				fc.reply(id, map[string]any{"attachmentId": "a-1", "path": "/tmp/a.txt", "bytes": 2})
			default:
				fc.reply(id, map[string]any{})
			}
		})
	}
	opener := &fakeOpener{scripts: []func(fc *fakeConn){script}}
	manager, err := session.NewManager(session.Config{Opener: opener})
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	defer manager.Close()

	_, lease, err := manager.Acquire(context.Background(), sessionID, session.Listener{})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer lease.Release()
	sub := lease.Subscription()
	ctx := context.Background()

	methods, err := sub.Methods(ctx)
	if err != nil || !methods.Has(protocol.MethodAttachmentPut) {
		t.Fatalf("methods: %+v, %v", methods, err)
	}
	lookup, err := sub.InputLookup(ctx, protocol.InputLookupParams{SubmissionID: "11111111-1111-4111-8111-111111111111"})
	if err != nil || lookup.Status != protocol.InputLookupUnknown {
		t.Fatalf("lookup: %+v, %v", lookup, err)
	}
	inputs, err := sub.PendingInputs(ctx, protocol.PendingQuery{BranchID: "main"})
	if err != nil || len(inputs.Inputs) != 0 {
		t.Fatalf("pending inputs: %+v, %v", inputs, err)
	}
	put, err := sub.PutAttachment(ctx, protocol.AttachmentPutParams{Name: "a.txt", Text: "hi"})
	if err != nil || put.AttachmentID != "a-1" {
		t.Fatalf("attachment: %+v, %v", put, err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seenSessionIDs) < 5 {
		t.Fatalf("calls were not routed: %v", seenSessionIDs)
	}
	for _, seen := range seenSessionIDs {
		if seen != sessionID {
			t.Fatalf("sessionId was not filled by the subscription: %q", seen)
		}
	}
}

func TestRecognizedNotificationsAreTyped(t *testing.T) {
	sessionID := "s-6"
	script := streamScript{
		sessionID: sessionID,
		head:      0,
		onAttach: func(fc *fakeConn) {
			fc.notify("approval.request", map[string]any{
				"sessionId": sessionID,
				"branchId":  "main",
				"requestId": "req-1",
				"source":    map[string]any{"kind": "tool", "call_id": "call-1"},
				"call":      map[string]any{"name": "exec", "input": map[string]any{"command": "ls"}},
				"reason":    "needs approval",
			})
			fc.notify("approval.resolved", map[string]any{
				"sessionId": sessionID, "branchId": "main", "requestId": "req-1", "decision": "allow",
			})
			fc.notify("session.notice", map[string]any{
				"sessionId": sessionID,
				"source":    "SessionStore",
				"kind":      "title",
				"data":      map[string]any{"title": "Hello", "titleSource": "auto", "titleRevision": 3},
			})
			fc.notify("approval.cancelled", map[string]any{
				"sessionId": sessionID, "branchId": "main", "reason": "missing-request-id",
			})
		},
	}.script()
	opener := &fakeOpener{scripts: []func(fc *fakeConn){script}}
	manager, err := session.NewManager(session.Config{Opener: opener})
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	defer manager.Close()

	events := make(chan session.Event, 8)
	_, lease, err := manager.Acquire(context.Background(), sessionID, session.Listener{
		OnEvent: func(event session.Event) { events <- event },
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer lease.Release()

	var collected []session.Event
	for len(collected) < 4 {
		select {
		case event := <-events:
			collected = append(collected, event)
		case <-time.After(2 * time.Second):
			t.Fatalf("missing notifications, got %d", len(collected))
		}
	}
	find := func(method string) session.Event {
		for _, event := range collected {
			if event.Method == method {
				return event
			}
		}
		t.Fatalf("missing %s", method)
		return session.Event{}
	}
	request := find("approval.request")
	if request.ApprovalRequest == nil || request.ApprovalRequest.Call.Name != "exec" {
		t.Fatalf("approval.request was not typed: %+v", request)
	}
	resolved := find("approval.resolved")
	if resolved.ApprovalResolved == nil || resolved.ApprovalResolved.Decision != "allow" {
		t.Fatalf("approval.resolved was not typed: %+v", resolved)
	}
	notice := find("session.notice")
	if notice.Title == nil || notice.Title.Title == nil || *notice.Title.Title != "Hello" || notice.Title.TitleRevision != 3 {
		t.Fatalf("title notice was not typed: %+v", notice)
	}
	cancelled := find("approval.cancelled")
	if cancelled.DecodeErr == nil || cancelled.ApprovalCancelled != nil {
		t.Fatalf("a malformed approval.cancelled must keep raw params and report DecodeErr: %+v", cancelled)
	}
	if len(cancelled.Params) == 0 {
		t.Fatal("raw params must be preserved")
	}
}

func TestManagementNotificationsAreTyped(t *testing.T) {
	sessionID := "s-8"
	script := streamScript{
		sessionID: sessionID,
		head:      0,
		onAttach: func(fc *fakeConn) {
			fc.notify("session.event", map[string]any{
				"sessionId": sessionID,
				"branchId":  "main",
				"event": map[string]any{
					"id": "e-1", "sequence": 1, "create_at": 1,
					"type": "permission", "name": "changed",
					"data": map[string]any{"policy": map[string]any{
						"preset": "read-only", "allowedTools": []any{}, "disallowedTools": []any{},
					}},
				},
			})
			fc.notify("session.notice", map[string]any{
				"sessionId": sessionID, "source": "Scheduler", "kind": "wakeup",
				"data": map[string]any{"branchId": "main", "wakeupId": "w-1", "state": "scheduled"},
			})
			fc.notify("management.progress", map[string]any{
				"id": 9, "method": "plugin.install", "data": map[string]any{"phase": "fetch"},
			})
			fc.notify("modality.realtime.event", map[string]any{
				"connectionId": "c-1", "event": map[string]any{"kind": "text"},
			})
			fc.notify("modality.realtime.ended", map[string]any{
				"connectionId": "c-1", "reason": "closed",
			})
		},
	}.script()
	opener := &fakeOpener{scripts: []func(fc *fakeConn){script}}
	manager, err := session.NewManager(session.Config{Opener: opener})
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	defer manager.Close()

	events := make(chan session.Event, 8)
	_, lease, err := manager.Acquire(context.Background(), sessionID, session.Listener{
		OnEvent: func(event session.Event) { events <- event },
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer lease.Release()

	var collected []session.Event
	for len(collected) < 5 {
		select {
		case event := <-events:
			collected = append(collected, event)
		case <-time.After(2 * time.Second):
			t.Fatalf("missing notifications, got %d", len(collected))
		}
	}
	find := func(method string) session.Event {
		for _, event := range collected {
			if event.Method == method {
				return event
			}
		}
		t.Fatalf("missing %s", method)
		return session.Event{}
	}
	if permission := find("session.event"); permission.PermissionChanged == nil ||
		permission.PermissionChanged.Policy.Preset != protocol.PermissionPresetReadOnly {
		t.Fatalf("permission.changed event was not typed: %+v", permission)
	}
	if notice := find("session.notice"); notice.Wakeup == nil || notice.Wakeup.WakeupID != "w-1" {
		t.Fatalf("wakeup notice was not typed: %+v", notice)
	}
	progress := find("management.progress")
	if progress.Progress == nil || progress.Progress.Method != "plugin.install" {
		t.Fatalf("management progress was not typed: %+v", progress)
	}
	if realtime := find("modality.realtime.event"); realtime.ModalityEvent == nil ||
		realtime.ModalityEvent.ConnectionID != "c-1" {
		t.Fatalf("realtime event was not typed: %+v", realtime)
	}
	if ended := find("modality.realtime.ended"); ended.ModalityEnded == nil || ended.ModalityEnded.Reason != "closed" {
		t.Fatalf("realtime ended was not typed: %+v", ended)
	}
}

func TestUnknownNotificationIsPreserved(t *testing.T) {
	sessionID := "s-5"
	script := streamScript{
		sessionID: sessionID,
		head:      0,
		onAttach: func(fc *fakeConn) {
			go func() {
				time.Sleep(10 * time.Millisecond)
				fc.notify("vendor.unknown", map[string]any{"value": 1})
			}()
		},
	}.script()
	opener := &fakeOpener{scripts: []func(fc *fakeConn){script}}
	manager, err := session.NewManager(session.Config{Opener: opener})
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	defer manager.Close()

	methods := make(chan string, 1)
	_, lease, err := manager.Acquire(context.Background(), sessionID, session.Listener{
		OnEvent: func(event session.Event) { methods <- event.Method },
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer lease.Release()
	select {
	case method := <-methods:
		if method != "vendor.unknown" {
			t.Fatalf("method: %q", method)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("unknown notification was not delivered")
	}
}
