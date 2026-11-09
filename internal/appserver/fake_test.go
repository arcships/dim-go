package appserver

import (
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"
)

// fakeWaitTimeout bounds every wait in fake-server tests so a broken script
// fails with a message instead of hanging.
const fakeWaitTimeout = 2 * time.Second

type fakeRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func fakeBadParams() *fakeRPCError {
	return &fakeRPCError{Code: -32602, Message: "invalid params"}
}

func fakeUnknownMethod() *fakeRPCError {
	return &fakeRPCError{Code: -32601, Message: "unknown method", Data: json.RawMessage(`{"appError":"unknown_method"}`)}
}

// fakeCatalogPage is one catalog page keyed by the cursor that requests it.
type fakeCatalogPage struct {
	Sessions   []SessionSummary
	NextCursor *string
}

// catalogGate holds every catalog response until open, so tests can observe a
// traversal mid-flight without sleeping.
type catalogGate struct {
	reached     chan struct{}
	release     chan struct{}
	reachedOnce sync.Once
	releaseOnce sync.Once
}

func newCatalogGate() *catalogGate {
	return &catalogGate{reached: make(chan struct{}), release: make(chan struct{})}
}

func (g *catalogGate) wait() {
	g.reachedOnce.Do(func() { close(g.reached) })
	<-g.release
}

func (g *catalogGate) open() {
	g.releaseOnce.Do(func() { close(g.release) })
}

// fakeMutation scripts one mutation reply. Fault takes precedence over Result;
// both zero values mean an empty success object.
type fakeMutation struct {
	Result json.RawMessage
	Fault  *fakeRPCError
}

type fakeConfig struct {
	Hello HelloResult

	CatalogPages map[string]fakeCatalogPage
	CatalogGate  *catalogGate // optional: blocks catalog responses until opened

	Summaries          []SessionSummary
	SummariesOldDaemon bool

	CreateResult json.RawMessage
	CreateFault  *fakeRPCError

	Attach map[string]AttachResult

	// OnAttach runs inside the session.attach reply path, after the request is
	// recorded and before the reply is written. Tests use it to emit
	// notifications that must arrive before the attach reply.
	OnAttach func(*fakeServer)

	// DetachErr is returned by session.detach when set.
	DetachErr *fakeRPCError

	// HistoryRaw is served in order, one raw JSON result per session.history
	// request. Tests script exactly the pages they expect to be requested.
	HistoryRaw []json.RawMessage

	// InputLookupRaw is the session.input_lookup reply, and InputLookupFault an
	// optional error that replaces it.
	InputLookupRaw   json.RawMessage
	InputLookupFault *fakeRPCError

	// MethodsRaw scripts the session.methods read helper response.
	MethodsRaw json.RawMessage

	// KeysRaw/KeysFault script session.keys; BranchesRaw/BranchesFault script
	// session.branches. The params are validated before a reply is served.
	KeysRaw       json.RawMessage
	KeysFault     *fakeRPCError
	BranchesRaw   json.RawMessage
	BranchesFault *fakeRPCError

	// Mutations is served in order per mutation method. Tests that script a
	// request assert the queue was consumed exactly once.
	Mutations map[string][]fakeMutation

	// MutationGate blocks a mutation method before its reply so a caller can
	// time out while the request is provably in flight.
	MutationGate map[string]*catalogGate

	// CloseOnMethod drops the transport after recording the request, before
	// any reply, producing a connection_closed unknown result.
	CloseOnMethod map[string]bool
}

type fakeRequest struct {
	Method string
	Params json.RawMessage
}

// fakeServer is a deterministic daemon-side peer over net.Pipe. It serves only
// the methods this wave's tests need; everything else gets -32601.
type fakeServer struct {
	t    *testing.T
	conn net.Conn
	fr   *FrameReader
	cfg  fakeConfig

	writeMu sync.Mutex
	mu      sync.Mutex
	calls   []fakeRequest

	historyIndex int
	mutations    map[string][]fakeMutation
}

func newFakeServer(t *testing.T, cfg fakeConfig) (*fakeServer, *Client) {
	t.Helper()
	a, b := net.Pipe()
	server := &fakeServer{t: t, conn: b, fr: NewFrameReader(b), cfg: cfg, mutations: cloneFakeMutations(cfg.Mutations)}
	go server.serve()
	client := NewClient(a)
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

func (s *fakeServer) serve() {
	for {
		raw, err := s.fr.ReadFrame()
		if err != nil {
			return
		}
		if err := ValidateEnvelope(raw); err != nil {
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
		s.calls = append(s.calls, fakeRequest{Method: request.Method, Params: request.Params})
		closeNow := s.cfg.CloseOnMethod[request.Method]
		s.mu.Unlock()
		if closeNow {
			_ = s.conn.Close()
			return
		}
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
	case "hello":
		var p HelloParams
		if err := json.Unmarshal(params, &p); err != nil || p.Validate() != nil {
			return nil, fakeBadParams()
		}
		return s.cfg.Hello, nil
	case "session.catalog":
		return s.catalog(params)
	case "session.summaries":
		return s.summaries(params)
	case "session.create":
		var p CreateParams
		if err := json.Unmarshal(params, &p); err != nil || p.Validate() != nil {
			return nil, fakeBadParams()
		}
		if s.cfg.CreateFault != nil {
			return nil, s.cfg.CreateFault
		}
		if len(s.cfg.CreateResult) == 0 {
			return nil, fakeUnknownMethod()
		}
		return s.cfg.CreateResult, nil
	case "session.attach":
		var p AttachParams
		if err := json.Unmarshal(params, &p); err != nil || p.Validate() != nil {
			return nil, fakeBadParams()
		}
		result, ok := s.cfg.Attach[p.SessionID]
		if !ok {
			return nil, fakeBadParams()
		}
		if s.cfg.OnAttach != nil {
			s.cfg.OnAttach(s)
		}
		return result, nil
	case "session.detach":
		var p DetachParams
		if err := json.Unmarshal(params, &p); err != nil || p.Validate() != nil {
			return nil, fakeBadParams()
		}
		if s.cfg.DetachErr != nil {
			return nil, s.cfg.DetachErr
		}
		return struct{}{}, nil
	case "session.history":
		return s.history(params)
	case "session.input_lookup":
		var p InputLookupParams
		if err := json.Unmarshal(params, &p); err != nil || p.Validate() != nil {
			return nil, fakeBadParams()
		}
		if s.cfg.InputLookupFault != nil {
			return nil, s.cfg.InputLookupFault
		}
		if len(s.cfg.InputLookupRaw) == 0 {
			return nil, fakeBadParams()
		}
		return s.cfg.InputLookupRaw, nil
	case "session.methods":
		var p MethodsParams
		if err := json.Unmarshal(params, &p); err != nil || p.Validate() != nil || len(s.cfg.MethodsRaw) == 0 {
			return nil, fakeBadParams()
		}
		return s.cfg.MethodsRaw, nil
	case "session.keys":
		var p KeysParams
		if err := json.Unmarshal(params, &p); err != nil || p.Validate() != nil {
			return nil, fakeBadParams()
		}
		if s.cfg.KeysFault != nil {
			return nil, s.cfg.KeysFault
		}
		if len(s.cfg.KeysRaw) == 0 {
			return nil, fakeBadParams()
		}
		return s.cfg.KeysRaw, nil
	case "session.branches":
		var p BranchesParams
		if err := json.Unmarshal(params, &p); err != nil || p.Validate() != nil {
			return nil, fakeBadParams()
		}
		if s.cfg.BranchesFault != nil {
			return nil, s.cfg.BranchesFault
		}
		if len(s.cfg.BranchesRaw) == 0 {
			return nil, fakeBadParams()
		}
		return s.cfg.BranchesRaw, nil
	case "session.submit":
		return s.mutation(MutationSessionSubmit, params)
	case "session.stop":
		return s.mutation(MutationSessionStop, params)
	case "approval.respond":
		return s.mutation(MutationApprovalRespond, params)
	case "session.delete":
		return s.mutation(MutationSessionDelete, params)
	default:
		return nil, fakeUnknownMethod()
	}
}

func (s *fakeServer) catalog(raw json.RawMessage) (any, *fakeRPCError) {
	if gate := s.cfg.CatalogGate; gate != nil {
		gate.wait()
	}
	var p CatalogParams
	if err := json.Unmarshal(raw, &p); err != nil || p.Validate() != nil {
		return nil, fakeBadParams()
	}
	if (p.WithInternal != nil || p.WithAcpSessions != nil) && !s.cfg.Hello.HasFeature("session.catalog.origins") {
		// Without the origins capability these params are invalid; the client
		// must strip them and retry once (§7.1).
		return nil, fakeBadParams()
	}
	page, ok := s.cfg.CatalogPages[p.Cursor]
	if !ok {
		return nil, fakeBadParams()
	}
	sessions := page.Sessions
	if sessions == nil {
		sessions = []SessionSummary{}
	}
	return CatalogResult{Sessions: sessions, NextCursor: page.NextCursor}, nil
}

func (s *fakeServer) summaries(raw json.RawMessage) (any, *fakeRPCError) {
	var p SummariesParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fakeBadParams()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfg.SummariesOldDaemon && p.IncludeClosed {
		// An old daemon rejects includeClosed on every attempt; the client must
		// retry once without the field and never send it again.
		return nil, fakeBadParams()
	}
	summaries := s.cfg.Summaries
	if summaries == nil {
		summaries = []SessionSummary{}
	}
	return summaries, nil
}

func (s *fakeServer) history(raw json.RawMessage) (any, *fakeRPCError) {
	var p HistoryParams
	if err := json.Unmarshal(raw, &p); err != nil || p.Validate() != nil {
		return nil, fakeBadParams()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.historyIndex >= len(s.cfg.HistoryRaw) {
		return nil, fakeBadParams()
	}
	result := s.cfg.HistoryRaw[s.historyIndex]
	s.historyIndex++
	return result, nil
}

// mutation validates the request, optionally blocks on the method gate, then
// serves the next scripted reply. An empty queue is a test bug, reported as
// -32602 so it fails visibly instead of hanging.
func (s *fakeServer) mutation(method MutationMethod, raw json.RawMessage) (any, *fakeRPCError) {
	switch method {
	case MutationSessionSubmit:
		var p SubmitParams
		if err := json.Unmarshal(raw, &p); err != nil || p.Validate() != nil {
			return nil, fakeBadParams()
		}
	case MutationSessionStop:
		var p StopParams
		if err := json.Unmarshal(raw, &p); err != nil || p.Validate() != nil {
			return nil, fakeBadParams()
		}
	case MutationApprovalRespond:
		var p ApprovalRespondParams
		if err := json.Unmarshal(raw, &p); err != nil || p.Validate() != nil {
			return nil, fakeBadParams()
		}
	case MutationSessionDelete:
		var p DeleteParams
		if err := json.Unmarshal(raw, &p); err != nil || p.Validate() != nil {
			return nil, fakeBadParams()
		}
	default:
		return nil, fakeUnknownMethod()
	}
	if gate := s.cfg.MutationGate[string(method)]; gate != nil {
		gate.wait()
	}
	s.mu.Lock()
	queue := s.mutations[string(method)]
	if len(queue) == 0 {
		s.mu.Unlock()
		return nil, fakeBadParams()
	}
	next := queue[0]
	s.mutations[string(method)] = queue[1:]
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

// emit writes one scripted notification from the daemon side.
func (s *fakeServer) emit(method string, params any) {
	s.t.Helper()
	s.writeJSON(struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{"2.0", method, params})
}

func (s *fakeServer) writeJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, _ = s.conn.Write(append(data, '\n'))
}

func (s *fakeServer) recordedCalls(method string) []fakeRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []fakeRequest
	for _, call := range s.calls {
		if call.Method == method {
			out = append(out, call)
		}
	}
	return out
}

// recordedMethods is the request method order the server observed.
func (s *fakeServer) recordedMethods() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.calls))
	for _, call := range s.calls {
		out = append(out, call.Method)
	}
	return out
}

// closeConn drops the daemon-side transport, as an unexpected stream loss.
func (s *fakeServer) closeConn() { _ = s.conn.Close() }
