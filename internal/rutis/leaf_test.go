package rutis

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFrameRoundTripAndLimits(t *testing.T) {
	var buffer bytes.Buffer
	if err := writeJSON(&buffer, map[string]any{"op": "hello", "value": "a\nb"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	frame, err := readFrame(bufio.NewReader(&buffer))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(frame, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded["value"] != "a\nb" {
		t.Fatalf("value: %#v", decoded["value"])
	}

	over := append(bytes.Repeat([]byte("x"), MaxFrameBytes+1), '\n')
	if _, err := readFrame(bufio.NewReader(bytes.NewReader(over))); err == nil {
		t.Fatal("want an error for an over-limit frame")
	}
	if err := writeJSON(io.Discard, strings.Repeat("x", MaxFrameBytes)); err == nil {
		t.Fatal("want an error for an over-limit message")
	}
}

func TestHandshakeAcceptsTheDaemonGreeting(t *testing.T) {
	leaf := &Leaf{peer: "ui-abc12345"}
	leafSide, daemonSide := net.Pipe()
	defer leafSide.Close()
	defer daemonSide.Close()

	daemonHello := Hello{
		Op:       "hello",
		Version:  ProtocolVersion,
		Endpoint: "dim-daemon",
		Capabilities: []string{
			"reentrant-sync", "plugins",
		},
	}
	go func() {
		// Read the leaf's greeting before answering.
		if _, err := readFrame(bufio.NewReader(daemonSide)); err != nil {
			return
		}
		_ = writeJSON(daemonSide, daemonHello)
	}()
	if err := leaf.handshake(leafSide, bufio.NewReader(leafSide), "dim-daemon"); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	if leaf.Greeting().Endpoint != "dim-daemon" || !leaf.Greeting().Has("plugins") {
		t.Fatalf("greeting: %+v", leaf.Greeting())
	}
}

func TestHandshakeRejectsWrongPeerAndVersion(t *testing.T) {
	cases := []struct {
		name  string
		hello Hello
		want  error
	}{
		{
			name:  "wrong peer",
			hello: Hello{Op: "hello", Version: ProtocolVersion, Endpoint: "other"},
			want:  ErrExpectedPeer,
		},
		{
			name:  "wrong version",
			hello: Hello{Op: "hello", Version: 3, Endpoint: "dim-daemon"},
			want:  ErrIncompatible,
		},
		{
			name:  "not hello",
			hello: Hello{Op: "invoke", Endpoint: "dim-daemon"},
			want:  ErrIncompatible,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			leaf := &Leaf{peer: "ui-abc12345"}
			leafSide, daemonSide := net.Pipe()
			defer leafSide.Close()
			defer daemonSide.Close()
			go func() {
				if _, err := readFrame(bufio.NewReader(daemonSide)); err != nil {
					return
				}
				_ = writeJSON(daemonSide, test.hello)
			}()
			err := leaf.handshake(leafSide, bufio.NewReader(leafSide), "dim-daemon")
			if !errors.Is(err, test.want) {
				t.Fatalf("want %v, got %v", test.want, err)
			}
		})
	}
}

func TestServeRefusesACallAndEndsWithTheChannel(t *testing.T) {
	denied, err := os.MkdirTemp("", "rutis-leaf-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(denied)
	leaf, err := Listen(denied)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	served := make(chan error, 1)
	go func() { served <- leaf.Serve(ctx, "dim-daemon") }()

	conn, err := net.Dial("unix", strings.TrimPrefix(leaf.Address(), "unix:"))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	reader := bufio.NewReader(conn)
	if _, err := readFrame(reader); err != nil {
		t.Fatalf("read leaf hello: %v", err)
	}
	if err := writeJSON(conn, Hello{Op: "hello", Version: ProtocolVersion, Endpoint: "dim-daemon"}); err != nil {
		t.Fatalf("write daemon hello: %v", err)
	}
	if err := writeJSON(conn, map[string]any{"op": "invoke", "id": "dim-daemon:1", "target": "", "method": "x"}); err != nil {
		t.Fatalf("write call: %v", err)
	}
	reply, err := readFrame(reader)
	if err != nil {
		t.Fatalf("read throw: %v", err)
	}
	var thrown struct {
		Op    string `json:"op"`
		Error struct {
			Name string `json:"name"`
		} `json:"error"`
	}
	if err := json.Unmarshal(reply, &thrown); err != nil || thrown.Op != "throw" || thrown.Error.Name != "BindingError" {
		t.Fatalf("reply: %s (%v)", reply, err)
	}
	_ = conn.Close()
	select {
	case err := <-served:
		if !errors.Is(err, ErrClosed) {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not end with the channel")
	}
}
