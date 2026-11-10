package protocol_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

func TestParseMessageNotificationKeepsUnknownMethod(t *testing.T) {
	message, err := protocol.ParseMessage([]byte(`{"jsonrpc":"2.0","method":"vendor.unknown","params":{"x":1}}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if message.Kind != protocol.KindNotification || message.Method != "vendor.unknown" {
		t.Fatalf("unexpected message: %+v", message)
	}
	if string(message.Params) != `{"x":1}` {
		t.Fatalf("params not preserved: %s", message.Params)
	}
}

func TestParseMessageReplies(t *testing.T) {
	success, err := protocol.ParseMessage([]byte(`{"jsonrpc":"2.0","id":7,"result":{"ok":true}}`))
	if err != nil {
		t.Fatalf("parse success: %v", err)
	}
	if success.Kind != protocol.KindResult || string(success.ID) != "7" {
		t.Fatalf("unexpected success: %+v", success)
	}
	if string(success.Result) != `{"ok":true}` {
		t.Fatalf("result not preserved: %s", success.Result)
	}

	failureReply, err := protocol.ParseMessage([]byte(
		`{"jsonrpc":"2.0","id":"abc","error":{"code":-32601,"message":"no","data":{"appError":"unknown_method"}}}`,
	))
	if err != nil {
		t.Fatalf("parse failure: %v", err)
	}
	if failureReply.Kind != protocol.KindError || failureReply.Error.AppError() != "unknown_method" {
		t.Fatalf("unexpected failure: %+v", failureReply)
	}
}

func TestParseMessageRejectsMalformedEnvelopes(t *testing.T) {
	cases := map[string]string{
		"missing version":          `{"id":1,"result":null}`,
		"wrong version":            `{"jsonrpc":"1.0","id":1,"result":null}`,
		"notification with id":     `{"jsonrpc":"2.0","id":1,"method":"x"}`,
		"reply without id":         `{"jsonrpc":"2.0","result":null}`,
		"reply with both":          `{"jsonrpc":"2.0","id":1,"result":null,"error":{"code":1,"message":"m"}}`,
		"reply with neither":       `{"jsonrpc":"2.0","id":1}`,
		"invalid id":               `{"jsonrpc":"2.0","id":true,"result":null}`,
		"invalid error object":     `{"jsonrpc":"2.0","id":1,"error":{"message":"m"}}`,
		"invalid error appError":   `{"jsonrpc":"2.0","id":1,"error":{"code":1,"message":"m","data":{"appError":2}}}`,
		"successful reply null id": `{"jsonrpc":"2.0","id":null,"result":null}`,
	}
	for name, frame := range cases {
		if _, err := protocol.ParseMessage([]byte(frame)); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestIDValidationAndMarshalling(t *testing.T) {
	for _, valid := range []protocol.ID{"1", "0", `"abc"`, "9007199254740991"} {
		if !valid.Valid() {
			t.Errorf("id %q should be valid", valid)
		}
	}
	for _, invalid := range []protocol.ID{"", "1.5", "abc", `"unterminated`, "-1x"} {
		if invalid.Valid() {
			t.Errorf("id %q should be invalid", invalid)
		}
	}
	frame, err := protocol.EncodeFrame(protocol.Request{JSONRPC: "2.0", ID: "7", Method: "hello"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.Contains(string(frame), `"id":7`) {
		t.Fatalf("numeric id must stay numeric: %s", frame)
	}
}

func TestRPCErrorUnwrapClassificationInputs(t *testing.T) {
	rpcErr := &protocol.RPCError{Code: -32601, Message: "no", Data: []byte(`{"appError":"unknown_method"}`)}
	if rpcErr.AppError() != "unknown_method" {
		t.Fatalf("appError: %q", rpcErr.AppError())
	}
	var target *protocol.RPCError
	if !errors.As(error(rpcErr), &target) {
		t.Fatal("RPCError must be extractable with errors.As")
	}
}
