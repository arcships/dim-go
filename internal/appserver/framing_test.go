package appserver

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestFrameReaderHandlesChunksCRLFAndBlankLines(t *testing.T) {
	input := "\n\r\n{\"jsonrpc\":\"2.0\",\"method\":\"ping\"}\r\n{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":null}\n"
	r := NewFrameReader(bytes.NewBufferString(input))
	var got []string
	for {
		frame, err := r.ReadFrame()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, string(frame))
	}
	if len(got) != 2 || got[0] != `{"jsonrpc":"2.0","method":"ping"}` || got[1] != `{"jsonrpc":"2.0","id":1,"result":null}` {
		t.Fatalf("frames = %#v", got)
	}
}

func TestFrameReaderRejectsOversizedFrame(t *testing.T) {
	_, err := NewFrameReader(strings.NewReader(strings.Repeat("x", MaxFrameSize+1) + "\n")).ReadFrame()
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("error = %v, want ErrFrameTooLarge", err)
	}
}

func TestFrameReaderRejectsUnterminatedFrame(t *testing.T) {
	_, err := NewFrameReader(strings.NewReader(`{"jsonrpc":"2.0","method":"x"}`)).ReadFrame()
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("error = %v, want unexpected EOF", err)
	}
}

func TestValidateEnvelope(t *testing.T) {
	valid := []string{
		`{"jsonrpc":"2.0","method":"unknown","params":{"x":1}}`,
		`{"jsonrpc":"2.0","id":1,"method":"session.catalog"}`,
		`{"jsonrpc":"2.0","id":"7","result":null}`,
		`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"x","data":{"appError":"connection_closed"}}}`,
		`{"jsonrpc":"2.0","id":null,"error":{"code":-32000,"message":"x"}}`,
	}
	for _, raw := range valid {
		if err := ValidateEnvelope([]byte(raw)); err != nil {
			t.Errorf("valid envelope rejected: %v", err)
		}
	}
	invalid := []string{
		`{"jsonrpc":"1.0","method":"x"}`,
		`{"jsonrpc":"2.0","id":1,"method":"x","result":null}`,
		`{"jsonrpc":"2.0","id":1,"result":null,"error":{"code":1,"message":"x"}}`,
		`{"jsonrpc":"2.0","id":1}`,
		`{"jsonrpc":"2.0","method":"x","id":null}`,
		`{"jsonrpc":"2.0","id":1,"error":{"code":1,"message":"x","data":{"appError":2}}}`,
	}
	for _, raw := range invalid {
		if err := ValidateEnvelope([]byte(raw)); err == nil {
			t.Errorf("invalid envelope accepted: %s", raw)
		}
	}
}
