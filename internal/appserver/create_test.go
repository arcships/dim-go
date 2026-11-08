package appserver

import (
	"context"
	"errors"
	"testing"
)

func TestCreateValidatesParamsBeforeCall(t *testing.T) {
	server, client := newFakeServer(t, fakeConfig{Hello: fakeHello()})
	_, err := Create(context.Background(), client, CreateParams{})
	if err == nil {
		t.Fatal("Create accepted empty params")
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.calls) != 0 {
		t.Fatalf("calls = %d, want 0", len(server.calls))
	}
}

func TestCreateDecodesResult(t *testing.T) {
	server, client := newFakeServer(t, fakeConfig{
		Hello:        fakeHello(),
		CreateResult: []byte(`{"sessionId":"new-1","model":{"providerId":"p","modelId":"m"},"modelPreferences":{"serviceTier":"default"}}`),
	})
	result, err := Create(context.Background(), client, CreateParams{Cwd: "/tmp", Model: &CreateModel{Provider: "p", ModelID: "m"}, PermissionPolicy: &PermissionPolicy{Preset: "default"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != "new-1" || result.Model == nil || result.Model.ModelID != "m" {
		t.Fatalf("result = %+v", result)
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if len(server.calls) != 1 || server.calls[0].Method != "session.create" {
		t.Fatalf("calls = %+v", server.calls)
	}
}

func TestCreateUnknownMethodPropagates(t *testing.T) {
	_, client := newFakeServer(t, fakeConfig{Hello: fakeHello()})
	_, err := Create(context.Background(), client, CreateParams{Cwd: "/tmp"})
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("error = %T %v, want RPCError", err, err)
	}
	if rpcErr.Code != -32601 || rpcAppError(rpcErr.Data) != AppErrorUnknownMethod {
		t.Fatalf("rpc error = %+v", rpcErr)
	}
}

func TestCreateRejectsMalformedResult(t *testing.T) {
	_, client := newFakeServer(t, fakeConfig{Hello: fakeHello(), CreateResult: []byte(`{"model":{}}`)})
	_, err := Create(context.Background(), client, CreateParams{Cwd: "/tmp"})
	if err == nil {
		t.Fatal("malformed create result accepted")
	}
}
