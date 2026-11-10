package protocol_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// testServer 是一个最小的 JSON-RPC 对端，用来驱动客户端的分支。
type testServer struct {
	conn net.Conn

	mu     sync.Mutex
	frames int

	handle func(server *testServer, id protocol.ID, method string, params json.RawMessage)
}

func newTestClient(
	t *testing.T,
	handle func(server *testServer, id protocol.ID, method string, params json.RawMessage),
) *protocol.Client {
	t.Helper()
	clientSide, serverSide := net.Pipe()
	server := &testServer{conn: serverSide, handle: handle}
	go server.serve()
	client := protocol.NewClient(clientSide, protocol.Options{
		RequestTimeout: 300 * time.Millisecond,
	})
	t.Cleanup(func() {
		client.Close(errors.New("test cleanup"))
		_ = serverSide.Close()
	})
	return client
}

func (s *testServer) serve() {
	decoder := protocol.NewDecoder(0)
	buf := make([]byte, 4096)
	for {
		n, err := s.conn.Read(buf)
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
				s.mu.Lock()
				s.frames++
				s.mu.Unlock()
				s.handle(s, protocol.ID(request.ID), request.Method, request.Params)
			}
		}
		if err != nil {
			return
		}
	}
}

func (s *testServer) send(value any) {
	frame, err := protocol.EncodeFrame(value)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.conn.Write(frame); err != nil {
		return
	}
}

func (s *testServer) reply(id protocol.ID, result any) {
	s.send(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func (s *testServer) replyError(id protocol.ID, code int, message, appError string) {
	s.send(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": code, "message": message, "data": map[string]any{"appError": appError}},
	})
}

func (s *testServer) notify(method string, params any) {
	s.send(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func TestClientCorrelatesOutOfOrderReplies(t *testing.T) {
	ids := make(chan protocol.ID, 2)
	client := newTestClient(t, func(server *testServer, id protocol.ID, method string, _ json.RawMessage) {
		ids <- id
		if len(ids) == 2 {
			// 收集到两个请求后倒序回执：关联必须按 id，不按顺序。
			second := <-ids
			first := <-ids
			server.reply(second, map[string]any{"method": "second"})
			server.reply(first, map[string]any{"method": "first"})
		}
	})

	type outcome struct {
		method string
		err    error
	}
	results := make(chan outcome, 2)
	go func() {
		raw, err := client.Call(context.Background(), "a", nil)
		if err != nil {
			results <- outcome{err: err}
			return
		}
		var decoded struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(raw, &decoded)
		results <- outcome{method: decoded.Method}
	}()
	go func() {
		raw, err := client.Call(context.Background(), "b", nil)
		if err != nil {
			results <- outcome{err: err}
			return
		}
		var decoded struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(raw, &decoded)
		results <- outcome{method: decoded.Method}
	}()

	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatalf("call failed: %v", result.err)
			}
			got[result.method] = true
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for replies")
		}
	}
	if !got["first"] || !got["second"] {
		t.Fatalf("replies were not correlated: %+v", got)
	}
}

func TestClientTimesOutAndReportsRPCError(t *testing.T) {
	client := newTestClient(t, func(server *testServer, id protocol.ID, method string, _ json.RawMessage) {
		if method == "slow" {
			return
		}
		server.replyError(id, -32601, "no such method", "unknown_method")
	})

	_, err := client.Call(context.Background(), "slow", nil)
	if !errors.Is(err, protocol.ErrRequestTimeout) {
		t.Fatalf("want timeout, got %v", err)
	}
	var timeoutErr *protocol.TimeoutError
	if !errors.As(err, &timeoutErr) || timeoutErr.Method != "slow" {
		t.Fatalf("unexpected timeout error: %+v", err)
	}

	_, err = client.Call(context.Background(), "missing", nil)
	var rpcErr *protocol.RPCError
	if !errors.As(err, &rpcErr) || rpcErr.AppError() != "unknown_method" {
		t.Fatalf("want rpc error, got %v", err)
	}
}

func TestClientDispatchesUnknownNotificationsAndSurvivesBadFrames(t *testing.T) {
	client := newTestClient(t, func(server *testServer, id protocol.ID, method string, _ json.RawMessage) {
		if method == "notify" {
			// 先发坏帧，再发未知通知：坏帧只影响自己。
			server.mu.Lock()
			_, _ = server.conn.Write([]byte("not json\n"))
			server.mu.Unlock()
			server.notify("vendor.unknown", map[string]any{"x": 1})
			server.reply(id, map[string]any{"ok": true})
		}
	})

	events := make(chan string, 4)
	client.OnNotification(func(message protocol.Message) {
		events <- message.Method
	})

	if _, err := client.Call(context.Background(), "notify", nil); err != nil {
		t.Fatalf("call: %v", err)
	}
	select {
	case method := <-events:
		if method != "vendor.unknown" {
			t.Fatalf("unexpected notification %q", method)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("unknown notification was not delivered")
	}
}

func TestClientReportsDecodeErrorsThroughOnError(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer serverSide.Close()
	errorsSeen := make(chan error, 4)
	client := protocol.NewClient(clientSide, protocol.Options{
		RequestTimeout: 300 * time.Millisecond,
		OnError: func(err error) {
			errorsSeen <- err
		},
	})
	defer client.Close(errors.New("done"))

	go func() {
		_, _ = serverSide.Write([]byte("not json\n"))
	}()
	select {
	case err := <-errorsSeen:
		if !errors.Is(err, protocol.ErrMalformedFrame) {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("decode error was not reported")
	}
}

func TestClientCloseRejectsPendingCalls(t *testing.T) {
	client := newTestClient(t, func(server *testServer, id protocol.ID, method string, _ json.RawMessage) {
		// 永不回执，等客户端关闭。
	})
	clientDone := make(chan error, 1)
	go func() {
		_, err := client.Call(context.Background(), "hang", nil)
		clientDone <- err
	}()
	time.Sleep(50 * time.Millisecond)
	client.Close(errors.New("测试关闭"))
	select {
	case err := <-clientDone:
		if !errors.Is(err, protocol.ErrConnectionClosed) {
			t.Fatalf("want connection closed, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pending call was not rejected")
	}
	if _, err := client.Call(context.Background(), "after-close", nil); !errors.Is(err, protocol.ErrConnectionClosed) {
		t.Fatalf("call after close: %v", err)
	}
}
