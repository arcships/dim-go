package protocol_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

func TestDecoderAcceptsArbitraryChunks(t *testing.T) {
	decoder := protocol.NewDecoder(1024)
	frames, err := decoder.Push([]byte(`{"jsonrpc":"2.0","method":"a"}` + "\n" + `{"jsonrpc"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("want 1 frame, got %d", len(frames))
	}
	if decoder.Pending() != len(`{"jsonrpc"`) {
		t.Fatalf("want pending tail, got %d", decoder.Pending())
	}
	frames, err = decoder.Push([]byte(`:"2.0","method":"b"}` + "\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 1 || !strings.Contains(string(frames[0]), `"b"`) {
		t.Fatalf("want the second frame, got %q", frames)
	}
	if decoder.Pending() != 0 {
		t.Fatalf("want empty buffer, got %d", decoder.Pending())
	}
}

func TestDecoderSkipsBlankLinesAndCRLF(t *testing.T) {
	decoder := protocol.NewDecoder(1024)
	frames, err := decoder.Push([]byte("\r\n   \n" + `{"jsonrpc":"2.0","method":"a"}` + "\r\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("want 1 frame, got %d", len(frames))
	}
	if strings.HasSuffix(string(frames[0]), "\r") {
		t.Fatalf("frame still carries CR: %q", frames[0])
	}
}

func TestDecoderDropsOversizedCompleteFrameAndContinues(t *testing.T) {
	decoder := protocol.NewDecoder(48)
	big := `{"jsonrpc":"2.0","method":"` + strings.Repeat("x", 64) + `"}`
	frames, err := decoder.Push([]byte(`{"jsonrpc":"2.0","method":"ok"}` + "\n" + big + "\n" + `{"jsonrpc":"2.0","method":"ok2"}` + "\n"))
	if !errors.Is(err, protocol.ErrFrameTooLarge) {
		t.Fatalf("want frame-too-large, got %v", err)
	}
	var tooLarge *protocol.FrameTooLargeError
	if !errors.As(err, &tooLarge) || tooLarge.Bytes <= 48 || tooLarge.MaxBytes != 48 {
		t.Fatalf("unexpected frame-too-large detail: %+v", tooLarge)
	}
	if len(frames) != 2 {
		t.Fatalf("want the frames around the oversized one, got %d", len(frames))
	}
}

func TestDecoderDiscardsOversizedPendingLineUntilNewline(t *testing.T) {
	decoder := protocol.NewDecoder(48)
	frames, err := decoder.Push([]byte(strings.Repeat("y", 64)))
	if !errors.Is(err, protocol.ErrFrameTooLarge) || len(frames) != 0 {
		t.Fatalf("want frame-too-large with no frames, got %v, %d", err, len(frames))
	}
	if _, err := decoder.Push([]byte(` still the oversized line`)); err != nil {
		t.Fatalf("discard state must not report again: %v", err)
	}
	frames, err = decoder.Push([]byte("\n" + `{"jsonrpc":"2.0","method":"after"}` + "\n"))
	if err != nil {
		t.Fatalf("unexpected error after discard: %v", err)
	}
	if len(frames) != 1 || !strings.Contains(string(frames[0]), "after") {
		t.Fatalf("want the frame after the discarded line, got %q", frames)
	}
}

func TestDecoderDropsMalformedFrameAndContinues(t *testing.T) {
	decoder := protocol.NewDecoder(1024)
	frames, err := decoder.Push([]byte("not json\n" + `{"jsonrpc":"2.0","method":"ok"}` + "\n"))
	if !errors.Is(err, protocol.ErrMalformedFrame) {
		t.Fatalf("want malformed frame, got %v", err)
	}
	if len(frames) != 1 {
		t.Fatalf("want the good frame, got %d", len(frames))
	}
}

func TestEncodeFrameAddsNewline(t *testing.T) {
	frame, err := protocol.EncodeFrame(map[string]any{"jsonrpc": "2.0"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(frame) == 0 || frame[len(frame)-1] != '\n' {
		t.Fatalf("frame must end with newline: %q", frame)
	}
	var decoded map[string]any
	if err := json.Unmarshal(frame[:len(frame)-1], &decoded); err != nil {
		t.Fatalf("frame is not JSON: %v", err)
	}
}
