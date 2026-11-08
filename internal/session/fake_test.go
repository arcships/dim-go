package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

const testWait = 3 * time.Second

type fakeRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func invalidParams() *fakeRPCError { return &fakeRPCError{Code: -32602, Message: "invalid params"} }

func unknownMethod() *fakeRPCError {
	return &fakeRPCError{Code: -32601, Message: "unknown method", Data: json.RawMessage(`{"appError":"unknown_method"}`)}
}

func submissionUnknown() *fakeRPCError {
	return &fakeRPCError{Code: -32000, Message: "submission unknown", Data: json.RawMessage(`{"appError":"submission_unknown"}`)}
}

func historyTooLarge() *fakeRPCError {
	return &fakeRPCError{Code: -32000, Message: "history too large", Data: json.RawMessage(`{"appError":"history_too_large"}`)}
}

type fakeGate struct {
	ch chan struct{}
	mu sync.Mutex
}

func newFakeGate() *fakeGate { return &fakeGate{ch: make(chan struct{})} }

func (g *fakeGate) wait() { <-g.ch }

func (g *fakeGate) open() {
	g.mu.Lock()
	defer g.mu.Unlock()
	select {
	case <-g.ch:
	default:
		close(g.ch)
	}
}

type fakeMutation struct {
	Result json.RawMessage
	Fault  *fakeRPCError
}

type fakeHistoryFunc func(appserver.HistoryParams) (any, *fakeRPCError)

type fakeConfig struct {
	Attach       appserver.AttachResult
	AttachErr    *fakeRPCError
	OnAttach     func(*fakeServer)
	History      fakeHistoryFunc
	DetachErr    *fakeRPCError
	Mutations    map[string][]fakeMutation
	MutationGate map[string]*fakeGate
}

type fakeCall struct {
	Method string
	Params json.RawMessage
}

type fakeServer struct {
	t    *testing.T
	conn net.Conn
	fr   *appserver.FrameReader
	cfg  fakeConfig

	writeMu sync.Mutex
	mu      sync.Mutex
	calls   []fakeCall

	mutations map[string][]fakeMutation
	gates     map[string]*fakeGate
}

func newFakeServer(t *testing.T, cfg fakeConfig) (*fakeServer, *appserver.Client) {
	t.Helper()
	a, b := net.Pipe()
	server := &fakeServer{
		t:         t,
		conn:      b,
		fr:        appserver.NewFrameReader(b),
		cfg:       cfg,
		mutations: cloneFakeMutations(cfg.Mutations),
		gates:     cfg.MutationGate,
	}
	go server.serve()
	client := appserver.NewClient(a)
	t.Cleanup(func() {
		client.Close()
		_ = b.Close()
	})
	return server, client
}

func cloneFakeMutations(in map[string][]fakeMutation) map[string][]fakeMutation {
	out := make(map[string][]fakeMutation, len(in))
	for method, queue := range in {
		out[method] = append([]fakeMutation(nil), queue...)
	}
	return out
}

func (s *fakeServer) callsFor(method string) []fakeCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []fakeCall
	for _, call := range s.calls {
		if call.Method == method {
			out = append(out, call)
		}
	}
	return out
}

func (s *fakeServer) serve() {
	for {
		raw, err := s.fr.ReadFrame()
		if err != nil {
			return
		}
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(raw, &request); err != nil {
			return
		}
		s.mu.Lock()
		s.calls = append(s.calls, fakeCall{Method: request.Method, Params: request.Params})
		s.mu.Unlock()
		result, fault := s.dispatch(request.Method, request.Params)
		if fault != nil {
			s.writeError(request.ID, fault)
			continue
		}
		s.writeResult(request.ID, result)
	}
}

func (s *fakeServer) dispatch(method string, params json.RawMessage) (any, *fakeRPCError) {
	switch method {
	case "session.attach":
		var p appserver.AttachParams
		if err := json.Unmarshal(params, &p); err != nil || p.Validate() != nil {
			return nil, invalidParams()
		}
		if s.cfg.AttachErr != nil {
			return nil, s.cfg.AttachErr
		}
		if s.cfg.OnAttach != nil {
			s.cfg.OnAttach(s)
		}
		return s.cfg.Attach, nil
	case "session.history":
		var p appserver.HistoryParams
		if err := json.Unmarshal(params, &p); err != nil || p.Validate() != nil {
			return nil, invalidParams()
		}
		if s.cfg.History == nil {
			return nil, invalidParams()
		}
		return s.cfg.History(p)
	case "session.detach":
		var p appserver.DetachParams
		if err := json.Unmarshal(params, &p); err != nil || p.Validate() != nil {
			return nil, invalidParams()
		}
		if s.cfg.DetachErr != nil {
			return nil, s.cfg.DetachErr
		}
		return struct{}{}, nil
	case "session.submit", "session.stop", "approval.respond", "session.delete":
		return s.mutation(method, params)
	default:
		return nil, unknownMethod()
	}
}

func (s *fakeServer) mutation(method string, params json.RawMessage) (any, *fakeRPCError) {
	var request any
	switch method {
	case "session.submit":
		var p appserver.SubmitParams
		request = &p
	case "session.stop":
		var p appserver.StopParams
		request = &p
	case "approval.respond":
		var p appserver.ApprovalRespondParams
		request = &p
	case "session.delete":
		var p appserver.DeleteParams
		request = &p
	}
	if err := json.Unmarshal(params, request); err != nil {
		return nil, invalidParams()
	}
	switch value := request.(type) {
	case *appserver.SubmitParams:
		if value.Validate() != nil {
			return nil, invalidParams()
		}
	case *appserver.StopParams:
		if value.Validate() != nil {
			return nil, invalidParams()
		}
	case *appserver.ApprovalRespondParams:
		if value.Validate() != nil {
			return nil, invalidParams()
		}
	case *appserver.DeleteParams:
		if value.Validate() != nil {
			return nil, invalidParams()
		}
	}
	if gate := s.gates[method]; gate != nil {
		gate.wait()
	}
	s.mu.Lock()
	queue := s.mutations[method]
	if len(queue) == 0 {
		s.mu.Unlock()
		return nil, invalidParams()
	}
	next := queue[0]
	s.mutations[method] = queue[1:]
	s.mu.Unlock()
	if next.Fault != nil {
		return nil, next.Fault
	}
	if len(next.Result) == 0 {
		return json.RawMessage(`{}`), nil
	}
	return next.Result, nil
}

func (s *fakeServer) writeResult(id json.RawMessage, result any) {
	s.writeJSON(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  any             `json:"result"`
	}{"2.0", id, result})
}

func (s *fakeServer) writeError(id json.RawMessage, fault *fakeRPCError) {
	s.writeJSON(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Error   *fakeRPCError   `json:"error"`
	}{"2.0", id, fault})
}

func (s *fakeServer) emit(method string, params any) {
	s.t.Helper()
	s.writeJSON(struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{"2.0", method, params})
}

func (s *fakeServer) writeJSON(value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, _ = s.conn.Write(append(data, '\n'))
}

func (s *fakeServer) recorded(method string) []fakeCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []fakeCall
	for _, call := range s.calls {
		if call.Method == method {
			out = append(out, call)
		}
	}
	return out
}

func (s *fakeServer) closeConn() { _ = s.conn.Close() }

func rawEventWithID(id string, sequence int64, typ, name string, data map[string]any) appserver.RawEvent {
	payload, _ := json.Marshal(data)
	return appserver.RawEvent{
		ID:       id,
		Sequence: sequence,
		CreateAt: sequence * 1000,
		Type:     typ,
		Name:     name,
		Data:     payload,
	}
}

func rawEvent(sequence int64, typ, name string, data map[string]any) appserver.RawEvent {
	return rawEventWithID(fmt.Sprintf("e-%d", sequence), sequence, typ, name, data)
}

func userEvent(sequence int64, text string) appserver.RawEvent {
	return rawEvent(sequence, "input", "message", map[string]any{
		"user": map[string]any{
			"content": []any{map[string]any{"type": "text", "text": text}},
		},
	})
}

func userEventWithID(id string, sequence int64, text string) appserver.RawEvent {
	return rawEventWithID(id, sequence, "input", "message", map[string]any{
		"user": map[string]any{
			"content": []any{map[string]any{"type": "text", "text": text}},
		},
	})
}

func assistantEvent(sequence int64, turnID, requestID, delta string) appserver.RawEvent {
	return assistantEventWithID(fmt.Sprintf("e-%d", sequence), sequence, turnID, requestID, delta)
}

func assistantEventWithID(id string, sequence int64, turnID, requestID, delta string) appserver.RawEvent {
	return rawEventWithID(id, sequence, "step", "text", map[string]any{
		"turn_id":    turnID,
		"request_id": requestID,
		"delta":      delta,
	})
}

func turnStartEvent(sequence int64, turnID string) appserver.RawEvent {
	return rawEvent(sequence, "turn", "start", map[string]any{"turn_id": turnID})
}

func ignoredEvent(sequence int64) appserver.RawEvent {
	return rawEvent(sequence, "fixture", "ignored", nil)
}

func historyPage(t *testing.T, events []appserver.RawEvent, through, nextAfter int64, done bool, turnStart, nextBefore *int64) json.RawMessage {
	t.Helper()
	if events == nil {
		events = []appserver.RawEvent{}
	}
	payload := map[string]any{
		"events":    events,
		"through":   through,
		"nextAfter": nextAfter,
		"done":      done,
	}
	if turnStart != nil {
		payload["turnStart"] = *turnStart
	}
	if nextBefore != nil {
		payload["nextBefore"] = *nextBefore
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal history page: %v", err)
	}
	return data
}

func int64Ptr(value int64) *int64 { return &value }

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func openTestSession(t *testing.T, cfg fakeConfig, features []string, opts OpenOptions) (*fakeServer, *appserver.Client, *Session) {
	t.Helper()
	server, client := newFakeServer(t, cfg)
	s, err := OpenWithOptions(testContext(t), client, features, "s1", opts)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close(context.Background()) })
	return server, client, s
}

func waitFor(t *testing.T, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(testWait)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", message)
}

func waitSessionDone(t *testing.T, s *Session) {
	t.Helper()
	select {
	case <-s.Done():
	case <-time.After(testWait):
		t.Fatal("session did not reach a terminal state")
	}
}

func waitClientDone(t *testing.T, client *appserver.Client) {
	t.Helper()
	select {
	case <-client.Done():
	case <-time.After(testWait):
		t.Fatal("client was not closed")
	}
}

func collectErrors(t *testing.T, errs <-chan error) []error {
	t.Helper()
	var out []error
	timeout := time.After(testWait)
	for {
		select {
		case err, ok := <-errs:
			if !ok {
				return out
			}
			out = append(out, err)
		case <-timeout:
			t.Fatal("errors channel did not close")
			return out
		}
	}
}

func emptyHistory(t *testing.T, head int64) fakeHistoryFunc {
	return func(params appserver.HistoryParams) (any, *fakeRPCError) {
		if params.Through == nil || *params.Through != head {
			t.Fatalf("history through = %#v, want %d", params.Through, head)
		}
		return historyPage(t, nil, head, head, true, nil, nil), nil
	}
}
