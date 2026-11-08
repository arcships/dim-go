package appserver

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

// TestClientErrorReportsAreObservable floods Errors past its buffer, then proves
// that a later valid response still correlates and that no report is lost
// silently: delivered + DroppedErrors equals the number of framed violations.
func TestClientErrorReportsAreObservable(t *testing.T) {
	a, b := net.Pipe()
	t.Cleanup(func() { _ = b.Close() })
	client := NewClient(a)
	t.Cleanup(client.Close)

	out := callAsync(context.Background(), client, "daemon.ping", nil)
	if err := b.SetReadDeadline(time.Now().Add(testPeerTimeout)); err != nil {
		t.Fatalf("read deadline: %v", err)
	}
	raw, err := NewFrameReader(b).ReadFrame()
	if err != nil {
		t.Fatalf("read request: %v", err)
	}
	if err := b.SetReadDeadline(time.Time{}); err != nil {
		t.Fatalf("clear read deadline: %v", err)
	}
	var request struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("decode request %s: %v", raw, err)
	}

	malformed := []string{
		`not json`,
		`{"jsonrpc":"2.0","id":1,"result":null,"error":{"code":-32602,"message":"both"}}`,
		`{"jsonrpc":"2.0","id":9,"method":"future.notice"}`,
		`{"jsonrpc":"2.0","id":null,"error":{"code":-32000,"message":"orphan"}}`,
	}
	const repeats = 5 // 20 reports > the 8-slot Errors buffer
	if err := b.SetWriteDeadline(time.Now().Add(testPeerTimeout)); err != nil {
		t.Fatalf("write deadline: %v", err)
	}
	for i := 0; i < repeats; i++ {
		for _, frame := range malformed {
			if _, err := b.Write([]byte(frame + "\n")); err != nil {
				t.Fatalf("write malformed frame: %v", err)
			}
		}
	}
	if _, err := b.Write([]byte(`{"jsonrpc":"2.0","id":` + string(request.ID) + `,"result":"pong"}` + "\n")); err != nil {
		t.Fatalf("write response: %v", err)
	}
	if err := b.SetWriteDeadline(time.Time{}); err != nil {
		t.Fatalf("clear write deadline: %v", err)
	}

	outcome := awaitOutcome(t, out)
	if outcome.err != nil {
		t.Fatalf("call after error flood failed: %v", outcome.err)
	}
	if string(outcome.raw) != `"pong"` {
		t.Fatalf("result = %s, want pong", outcome.raw)
	}

	drained := 0
drain:
	for {
		select {
		case err := <-client.Errors():
			if err == nil {
				t.Fatal("nil error report")
			}
			drained++
		default:
			break drain
		}
	}
	total := uint64(repeats * len(malformed))
	if got := uint64(drained) + client.DroppedErrors(); got != total {
		t.Fatalf("observable reports = %d delivered + %d dropped, want %d", drained, client.DroppedErrors(), total)
	}
	if client.DroppedErrors() == 0 {
		t.Fatal("expected observable loss once the error buffer filled")
	}
}
