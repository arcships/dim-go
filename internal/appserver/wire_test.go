package appserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), fakeWaitTimeout)
	t.Cleanup(cancel)
	return ctx
}

func intPtr(v int) *int          { return &v }
func int64Ptr(v int64) *int64    { return &v }
func boolPtr(v bool) *bool       { return &v }
func stringPtr(v string) *string { return &v }

func rawJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return data
}

const fakeCreatedAt = int64(1700000000000)

func fakeHello(features ...string) HelloResult {
	return HelloResult{
		InstanceID:      "fake-instance",
		ServerVersion:   "fake-1.0",
		ProtocolVersion: WireProtocolVersion,
		DaemonProfile:   DaemonProfile{Kind: DaemonProfileProduct},
		Features:        features,
	}
}

func fakeSummary(id string, sequence int64) SessionSummary {
	return SessionSummary{
		SessionID:       id,
		Cwd:             "/workspace",
		Model:           SessionModel{ProviderID: "provider", ModelID: "model"},
		CreatedAt:       fakeCreatedAt,
		Running:         false,
		Sequence:        sequence,
		AttachedClients: 1,
		UpdatedAt:       fakeCreatedAt,
	}
}

func fakeEvent(sequence int64) map[string]any {
	return map[string]any{
		"id":        fmt.Sprintf("event-%d", sequence),
		"sequence":  sequence,
		"create_at": fakeCreatedAt,
		"type":      "message",
		"name":      "user_input",
		"data":      map[string]any{"turn_id": "turn-1"},
	}
}

func eventWithout(key string) map[string]any {
	event := fakeEvent(5)
	delete(event, key)
	return event
}

// ----- hello -----

func TestHelloValidationMatrix(t *testing.T) {
	params := HelloParams{ProtocolVersion: WireProtocolVersion, ClientKind: ClientKindOther, ClientName: "dimagent-mygo"}
	cases := []struct {
		name    string
		hello   HelloResult
		wantErr bool
	}{
		{"product", fakeHello("session.catalog"), false},
		{"mock", HelloResult{InstanceID: "i", ServerVersion: "s", ProtocolVersion: WireProtocolVersion, DaemonProfile: DaemonProfile{Kind: DaemonProfileMock}}, false},
		{"configured", HelloResult{InstanceID: "i", ServerVersion: "s", ProtocolVersion: WireProtocolVersion, DaemonProfile: DaemonProfile{Kind: DaemonProfileConfigured, ModelConfigSHA256: strings.Repeat("a", 64)}}, false},
		{"wrong protocol version", HelloResult{InstanceID: "i", ProtocolVersion: 2, DaemonProfile: DaemonProfile{Kind: DaemonProfileProduct}}, true},
		{"empty instance id", HelloResult{ProtocolVersion: WireProtocolVersion, DaemonProfile: DaemonProfile{Kind: DaemonProfileProduct}}, true},
		{"unknown profile kind", HelloResult{InstanceID: "i", ProtocolVersion: WireProtocolVersion, DaemonProfile: DaemonProfile{Kind: "other"}}, true},
		{"configured without sha", HelloResult{InstanceID: "i", ProtocolVersion: WireProtocolVersion, DaemonProfile: DaemonProfile{Kind: DaemonProfileConfigured}}, true},
		{"empty feature", HelloResult{InstanceID: "i", ProtocolVersion: WireProtocolVersion, DaemonProfile: DaemonProfile{Kind: DaemonProfileProduct}, Features: []string{""}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testContext(t)
			_, client := newFakeServer(t, fakeConfig{Hello: tc.hello})
			raw, err := client.Call(ctx, "hello", params)
			if err != nil {
				t.Fatalf("hello call: %v", err)
			}
			_, err = DecodeHelloResult(raw)
			if tc.wantErr {
				if !errors.Is(err, ErrProtocol) {
					t.Fatalf("DecodeHelloResult error = %v, want ErrProtocol", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeHelloResult: %v", err)
			}
		})
	}
}

func TestHelloParamsValidation(t *testing.T) {
	valid := HelloParams{ProtocolVersion: WireProtocolVersion, ClientKind: ClientKindOther, ClientName: "dimagent-mygo"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}
	invalid := []HelloParams{
		{ProtocolVersion: 2, ClientKind: ClientKindOther, ClientName: "dimagent-mygo"},
		{ProtocolVersion: WireProtocolVersion, ClientKind: "browser", ClientName: "dimagent-mygo"},
		{ProtocolVersion: WireProtocolVersion, ClientKind: ClientKindOther},
	}
	for _, params := range invalid {
		if err := params.Validate(); !errors.Is(err, ErrProtocol) {
			t.Errorf("Validate(%+v) = %v, want ErrProtocol", params, err)
		}
	}
}

// ----- session.attach + history head fallback -----

func attachHead(t *testing.T, ctx context.Context, client *Client, sessionID string) (int64, AttachResult) {
	t.Helper()
	raw, err := client.Call(ctx, "session.attach", AttachParams{SessionID: sessionID})
	if err != nil {
		t.Fatalf("session.attach: %v", err)
	}
	attach, err := DecodeAttachResult(raw)
	if err != nil {
		t.Fatalf("decode session.attach: %v", err)
	}
	if attach.Sequence != nil {
		return *attach.Sequence, attach
	}
	after := int64(0)
	limit := 1
	req := HistoryParams{SessionID: sessionID, BranchID: "main", After: &after, Limit: &limit}
	raw, err = client.Call(ctx, "session.history", req)
	if err != nil {
		t.Fatalf("session.history head fallback: %v", err)
	}
	history, err := DecodeHistoryResult(raw, req)
	if err != nil {
		t.Fatalf("decode head fallback: %v", err)
	}
	return *history.Through, attach
}

func TestAttachHead(t *testing.T) {
	t.Run("sequence present", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:  fakeHello("session.history"),
			Attach: map[string]AttachResult{"s1": {Sequence: int64Ptr(41), Running: boolPtr(true)}},
		})
		head, attach := attachHead(t, ctx, client, "s1")
		if head != 41 {
			t.Fatalf("head = %d, want 41", head)
		}
		if attach.Running == nil || !*attach.Running {
			t.Fatalf("running = %v", attach.Running)
		}
		if calls := server.recordedCalls("session.history"); len(calls) != 0 {
			t.Fatalf("history was called despite attach.sequence: %d calls", len(calls))
		}
	})

	t.Run("history head fallback", func(t *testing.T) {
		ctx := testContext(t)
		server, client := newFakeServer(t, fakeConfig{
			Hello:  fakeHello("session.history"),
			Attach: map[string]AttachResult{"s1": {}},
			HistoryRaw: []json.RawMessage{rawJSON(t, map[string]any{
				"events":    []any{fakeEvent(1)},
				"through":   int64(7),
				"nextAfter": int64(1),
				"done":      false,
			})},
		})
		head, attach := attachHead(t, ctx, client, "s1")
		if attach.Sequence != nil {
			t.Fatalf("sequence = %d, want absent", *attach.Sequence)
		}
		if head != 7 {
			t.Fatalf("head = %d, want fallback through 7", head)
		}
		calls := server.recordedCalls("session.history")
		if len(calls) != 1 {
			t.Fatalf("history calls = %d, want 1", len(calls))
		}
		var params HistoryParams
		if err := json.Unmarshal(calls[0].Params, &params); err != nil {
			t.Fatalf("history params: %v", err)
		}
		if params.SessionID != "s1" || params.BranchID != "main" {
			t.Fatalf("history params = %#v", params)
		}
		if params.After == nil || *params.After != 0 {
			t.Fatalf("after = %v, want explicit 0", params.After)
		}
		if params.Limit == nil || *params.Limit != 1 {
			t.Fatalf("limit = %v, want 1", params.Limit)
		}
		if params.Through != nil {
			t.Fatalf("through = %d, want absent", *params.Through)
		}
	})
}

// ----- history paging invariants -----

func TestHistoryPagingInvariantsFailFast(t *testing.T) {
	after := int64(4)
	through := int64(10)
	req := HistoryParams{SessionID: "s1", After: &after, Through: &through}

	page := func(events []any, through, nextAfter any, done bool) map[string]any {
		return map[string]any{"events": events, "through": through, "nextAfter": nextAfter, "done": done}
	}
	cases := []struct {
		name    string
		build   func(t *testing.T) json.RawMessage
		wantErr bool
	}{
		{"valid page", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{fakeEvent(5)}, through, int64(5), false))
		}, false},
		{"event at cursor", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{fakeEvent(4)}, through, int64(5), false))
		}, true},
		{"event past through", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{fakeEvent(11)}, through, int64(5), false))
		}, true},
		{"nextAfter before after", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{fakeEvent(5)}, through, int64(3), false))
		}, true},
		{"nextAfter past through", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{fakeEvent(5)}, through, int64(11), false))
		}, true},
		{"not done at cursor", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{fakeEvent(5)}, through, int64(4), false))
		}, true},
		{"through echo mismatch", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{fakeEvent(5)}, int64(9), int64(5), false))
		}, true},
		{"missing through", func(t *testing.T) json.RawMessage {
			result := page([]any{fakeEvent(5)}, nil, int64(5), false)
			delete(result, "through")
			return rawJSON(t, result)
		}, true},
		{"missing nextAfter", func(t *testing.T) json.RawMessage {
			result := page([]any{fakeEvent(5)}, through, nil, false)
			delete(result, "nextAfter")
			return rawJSON(t, result)
		}, true},
		{"missing events", func(t *testing.T) json.RawMessage {
			result := page(nil, through, int64(5), false)
			delete(result, "events")
			return rawJSON(t, result)
		}, true},
		{"event missing id", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{eventWithout("id")}, through, int64(5), false))
		}, true},
		{"event zero sequence", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{fakeEvent(0)}, through, int64(5), false))
		}, true},
		{"event missing create_at", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{eventWithout("create_at")}, through, int64(5), false))
		}, true},
		{"event missing type", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{eventWithout("type")}, through, int64(5), false))
		}, true},
		{"event missing name", func(t *testing.T) json.RawMessage {
			return rawJSON(t, page([]any{eventWithout("name")}, through, int64(5), false))
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testContext(t)
			_, client := newFakeServer(t, fakeConfig{
				Hello:      fakeHello("session.history"),
				HistoryRaw: []json.RawMessage{tc.build(t)},
			})
			raw, err := client.Call(ctx, "session.history", req)
			if err != nil {
				t.Fatalf("session.history: %v", err)
			}
			_, err = DecodeHistoryResult(raw, req)
			if tc.wantErr {
				if !errors.Is(err, ErrProtocol) {
					t.Fatalf("DecodeHistoryResult error = %v, want ErrProtocol", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeHistoryResult: %v", err)
			}
		})
	}
}

// ----- notifications -----

func TestUnknownNotificationPassesThrough(t *testing.T) {
	server, client := newFakeServer(t, fakeConfig{Hello: fakeHello()})
	params := json.RawMessage(`{"nested":{"list":[1,2,{"x":null}],"text":"\u2713"},"extra":true}`)
	server.emit("future.notice", params)
	select {
	case notification := <-client.Notifications():
		if notification.Method != "future.notice" {
			t.Fatalf("method = %q", notification.Method)
		}
		if string(notification.Params) != string(params) {
			t.Fatalf("params = %s, want %s", notification.Params, params)
		}
		payload, known, err := DecodeNotification(notification)
		if known || payload != nil || err != nil {
			t.Fatalf("unknown notification decoded as known=%v payload=%v err=%v", known, payload, err)
		}
	case <-time.After(fakeWaitTimeout):
		t.Fatal("notification did not arrive")
	}
}

func TestDecodeNotificationKnownMethods(t *testing.T) {
	event := fakeEvent(12)
	cases := []struct {
		name    string
		method  string
		params  map[string]any
		wantErr bool
		check   func(t *testing.T, payload any)
	}{
		{
			name:   "session.event",
			method: "session.event",
			params: map[string]any{"sessionId": "s1", "branchId": "main", "event": event},
			check: func(t *testing.T, payload any) {
				notice, ok := payload.(EventNotice)
				if !ok {
					t.Fatalf("payload type = %T", payload)
				}
				if notice.Event.Sequence != 12 || notice.Event.CreateAt != fakeCreatedAt {
					t.Fatalf("event = %#v", notice.Event)
				}
				if string(notice.Event.Data) != `{"turn_id":"turn-1"}` {
					t.Fatalf("event data = %s", notice.Event.Data)
				}
			},
		},
		{
			name:   "session.state",
			method: "session.state",
			params: map[string]any{"sessionId": "s1", "branchId": "main", "running": true, "stopping": false},
			check: func(t *testing.T, payload any) {
				notice, ok := payload.(StateNotice)
				if !ok {
					t.Fatalf("payload type = %T", payload)
				}
				if notice.SessionID != "s1" || !notice.Running || notice.Stopping {
					t.Fatalf("state = %#v", notice)
				}
			},
		},
		{
			name:   "approval.request",
			method: "approval.request",
			params: map[string]any{
				"sessionId": "s1", "branchId": "main", "requestId": "r1",
				"source": map[string]any{"kind": "tool", "callId": "c1"},
				"call":   map[string]any{"name": "shell", "input": map[string]any{"cmd": "ls"}},
				"reason": "needs approval",
			},
			check: func(t *testing.T, payload any) {
				notice, ok := payload.(ApprovalRequest)
				if !ok {
					t.Fatalf("payload type = %T", payload)
				}
				if notice.Source.Kind != "tool" || notice.Call.Name != "shell" {
					t.Fatalf("approval = %#v", notice)
				}
				if string(notice.Call.Input) != `{"cmd":"ls"}` {
					t.Fatalf("call input = %s", notice.Call.Input)
				}
			},
		},
		{
			name:   "approval.resolved",
			method: "approval.resolved",
			params: map[string]any{"sessionId": "s1", "branchId": "main", "requestId": "r1", "decision": DecisionAllow},
			check: func(t *testing.T, payload any) {
				notice, ok := payload.(ApprovalResolved)
				if !ok || notice.Decision != DecisionAllow {
					t.Fatalf("payload = %#v (%T)", payload, payload)
				}
			},
		},
		{
			name:   "approval.cancelled",
			method: "approval.cancelled",
			params: map[string]any{"sessionId": "s1", "branchId": "main", "requestId": "r1", "reason": "gone"},
			check: func(t *testing.T, payload any) {
				notice, ok := payload.(ApprovalCancelled)
				if !ok || notice.RequestID != "r1" {
					t.Fatalf("payload = %#v (%T)", payload, payload)
				}
			},
		},
		{
			name:   "session.notice",
			method: "session.notice",
			params: map[string]any{"sessionId": "s1", "source": "fixture", "kind": "title", "data": map[string]any{"title": "hello"}},
			check: func(t *testing.T, payload any) {
				notice, ok := payload.(SessionNotice)
				if !ok {
					t.Fatalf("payload type = %T", payload)
				}
				if notice.Source != "fixture" || notice.Kind != "title" {
					t.Fatalf("notice = %#v", notice)
				}
				if string(notice.Data) != `{"title":"hello"}` {
					t.Fatalf("data = %s", notice.Data)
				}
			},
		},
		{
			name:   "session.closed",
			method: "session.closed",
			params: map[string]any{"sessionId": "s1", "reason": "done"},
			check: func(t *testing.T, payload any) {
				notice, ok := payload.(ClosedNotice)
				if !ok || notice.SessionID != "s1" || notice.Reason != "done" {
					t.Fatalf("payload = %#v (%T)", payload, payload)
				}
			},
		},
		{
			name:   "session.model.progress",
			method: "session.model.progress",
			params: map[string]any{
				"sessionId": "s1", "branchId": "main", "turnId": "t1",
				"text": "Hello", "textOffset": 0, "thinkingOffset": 0,
				"tools": []any{map[string]any{"name": "shell"}},
			},
			check: func(t *testing.T, payload any) {
				progress, ok := payload.(ModelProgress)
				if !ok {
					t.Fatalf("payload type = %T", payload)
				}
				if progress.Text != "Hello" || len(progress.Tools) != 1 {
					t.Fatalf("progress = %#v", progress)
				}
				if string(progress.Tools[0]) != `{"name":"shell"}` {
					t.Fatalf("tool = %s", progress.Tools[0])
				}
			},
		},
		{
			name:    "invalid decision",
			method:  "approval.resolved",
			params:  map[string]any{"sessionId": "s1", "branchId": "main", "requestId": "r1", "decision": "maybe"},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, known, err := DecodeNotification(Notification{Method: tc.method, Params: rawJSON(t, tc.params)})
			if !known {
				t.Fatal("known = false for a §11 method")
			}
			if tc.wantErr {
				if !errors.Is(err, ErrProtocol) {
					t.Fatalf("error = %v, want ErrProtocol", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeNotification: %v", err)
			}
			if tc.check != nil {
				tc.check(t, payload)
			}
		})
	}
}

// ----- real fixture -----

func TestNotificationFixtureParsesAsEnvelopes(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "wire", "notifications.ndjson"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	count := 0
	for i, line := range bytes.Split(data, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		count++
		if err := ValidateEnvelope(line); err != nil {
			t.Fatalf("line %d is not a valid envelope: %v", i+1, err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(line, &fields); err != nil {
			t.Fatalf("line %d: %v", i+1, err)
		}
		methodRaw, ok := fields["method"]
		if !ok {
			t.Fatalf("line %d is not a notification", i+1)
		}
		var method string
		if err := json.Unmarshal(methodRaw, &method); err != nil || method == "" {
			t.Fatalf("line %d method = %q, %v", i+1, method, err)
		}
		if _, hasID := fields["id"]; hasID {
			t.Fatalf("line %d notification carries an id", i+1)
		}
	}
	if count == 0 {
		t.Fatal("fixture contains no notifications")
	}
}

// ----- request params -----

func TestSubmitStopParamsValidation(t *testing.T) {
	validSubmit := []SubmitParams{
		{SessionID: "s1", SubmissionID: "sub-1", BranchID: "main", Text: "hello"},
		{SessionID: "s1", SubmissionID: "sub-1", BranchID: "main", User: json.RawMessage(`{"id":"u1"}`)},
		{SessionID: "s1", SubmissionID: "sub-1", BranchID: "main", Text: "hello", Mode: SubmitModeSteer},
	}
	for _, params := range validSubmit {
		if err := params.Validate(); err != nil {
			t.Errorf("valid SubmitParams %+v rejected: %v", params, err)
		}
	}
	invalidSubmit := []SubmitParams{
		{SubmissionID: "sub-1", BranchID: "main", Text: "hello"},
		{SessionID: "s1", BranchID: "main", Text: "hello"},
		{SessionID: "s1", SubmissionID: "sub-1", Text: "hello"},
		{SessionID: "s1", SubmissionID: "sub-1", BranchID: "main"},
		{SessionID: "s1", SubmissionID: "sub-1", BranchID: "main", Text: "hello", Mode: "append-now"},
	}
	for _, params := range invalidSubmit {
		if err := params.Validate(); !errors.Is(err, ErrProtocol) {
			t.Errorf("invalid SubmitParams %+v = %v, want ErrProtocol", params, err)
		}
	}

	if err := (StopParams{SessionID: "s1", BranchID: "main"}).Validate(); err != nil {
		t.Errorf("valid StopParams rejected: %v", err)
	}
	for _, params := range []StopParams{
		{BranchID: "main"},
		{SessionID: "s1"},
	} {
		if err := params.Validate(); !errors.Is(err, ErrProtocol) {
			t.Errorf("invalid StopParams %+v = %v, want ErrProtocol", params, err)
		}
	}
}

func TestCwdParamUnion(t *testing.T) {
	cases := []struct {
		name string
		cwd  *CwdParam
		want string
	}{
		{"single", SingleCwd("/workspace"), `"/workspace"`},
		{"list", CwdList("/a", "/b"), `["/a","/b"]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.cwd)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(raw) != tc.want {
				t.Fatalf("marshal = %s, want %s", raw, tc.want)
			}
			var decoded CwdParam
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if err := decoded.Validate(); err != nil {
				t.Fatalf("decode of %s rejected: %v", raw, err)
			}
			if decoded.IsList != tc.cwd.IsList || strings.Join(decoded.Paths, ",") != strings.Join(tc.cwd.Paths, ",") {
				t.Fatalf("decoded = %#v, want %#v", decoded, tc.cwd)
			}
		})
	}
	if err := (CwdParam{}).Validate(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("empty cwd error = %v, want ErrProtocol", err)
	}
}
