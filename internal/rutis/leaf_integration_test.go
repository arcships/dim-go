package rutis_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/rutis"
)

// TestRealDaemonUIFLeaf runs the whole UI-leaf handshake against a real App
// Server. It is skipped unless DIMCODE_APP_SERVER_BIN names a binary; the
// binary's extra arguments come from DIMCODE_APP_SERVER_ARGS, or default to
// `app-server` for the `dimcode-server` executable.
func TestRealDaemonUIFLeaf(t *testing.T) {
	server := os.Getenv("DIMCODE_APP_SERVER_BIN")
	if server == "" {
		t.Skip("DIMCODE_APP_SERVER_BIN is not set")
	}
	args := strings.Fields(os.Getenv("DIMCODE_APP_SERVER_ARGS"))
	if len(args) == 0 && strings.Contains(filepath.Base(server), "dimcode-server") {
		args = []string{"app-server"}
	}
	dataDir, err := os.MkdirTemp("/tmp", "d1-uileaf-")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	supervisor, err := daemon.New(daemon.Config{
		DataDir:    dataDir,
		ServerPath: server,
		ServerArgs: args,
		MockModel:  true,
	})
	if err != nil {
		t.Fatalf("supervisor: %v", err)
	}
	connection, err := supervisor.Connect(ctx)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	work := filepath.Join(dataDir, "work")
	if err := os.MkdirAll(work, 0o700); err != nil {
		t.Fatalf("work dir: %v", err)
	}
	var created protocol.SessionCreateResult
	if err := connection.Client.CallInto(ctx, "session.create", map[string]any{"cwd": work}, &created); err != nil {
		t.Fatalf("create session: %v", err)
	}
	stream, err := supervisor.OpenStream(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	defer stream.Close(nil)
	if err := stream.Client.CallInto(ctx, "session.attach", protocol.SessionAttachParams{SessionID: created.SessionID}, nil); err != nil {
		t.Fatalf("attach: %v", err)
	}

	leaf, err := rutis.Listen(filepath.Join(dataDir, "rust"))
	if err != nil {
		t.Fatalf("leaf listen: %v", err)
	}
	defer leaf.Close()
	served := make(chan error, 1)
	go func() { served <- leaf.Serve(ctx, "dim-daemon") }()

	var opened protocol.UIOpened
	if err := stream.Client.CallInto(ctx, protocol.MethodUIOpen, protocol.UIOpenParams{
		SessionID: created.SessionID,
		Kind:      protocol.UiKindTUI,
		Peer:      leaf.Peer(),
		Address:   leaf.Address(),
	}, &opened); err != nil {
		t.Fatalf("ui.open: %v", err)
	}
	if opened.UIID != leaf.Peer() || opened.Token == "" {
		t.Fatalf("ui.open reply: %+v", opened)
	}
	if greeting := leaf.Greeting(); greeting.Endpoint != "dim-daemon" {
		t.Fatalf("leaf greeting: %+v", greeting)
	}
	sent, received := leaf.HandshakeSizes()
	t.Logf("leaf peer=%s greeting=%+v hello_bytes=%d far_hello_bytes=%d token_bytes=%d generation=%d",
		leaf.Peer(), leaf.Greeting(), sent, received, len(opened.Token), opened.Generation)

	var attached protocol.UIAttached
	if err := stream.Client.CallInto(ctx, protocol.MethodUIAttach, protocol.UIAttachParams{
		SessionID: created.SessionID,
		Kind:      protocol.UiKindTUI,
		UIID:      &opened.UIID,
		Token:     &opened.Token,
	}, &attached); err != nil {
		t.Fatalf("ui.attach: %v", err)
	}
	if attached.UIID != opened.UIID || attached.Generation != opened.Generation {
		t.Fatalf("ui.attach reply: %+v", attached)
	}

	var status protocol.UIStatusResult
	if err := connection.Client.CallInto(ctx, protocol.MethodUIStatus, struct{}{}, &status); err != nil {
		t.Fatalf("ui.status: %v", err)
	}
	found := false
	for _, instance := range status.Instances {
		if instance.UIID == opened.UIID && instance.State == protocol.UiStateOnline {
			found = true
		}
	}
	if !found {
		t.Fatalf("ui.status did not list the leaf: %+v", status)
	}
	t.Logf("ui.status instances=%d state=%s generation=%d", len(status.Instances), protocol.UiStateOnline, attached.Generation)

	var ignored struct{}
	if err := connection.Client.CallInto(ctx, protocol.MethodUIClose, protocol.UICloseParams{UIID: opened.UIID}, &ignored); err != nil {
		t.Fatalf("ui.close: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := connection.Client.CallInto(ctx, protocol.MethodUIStatus, struct{}{}, &status); err != nil {
			t.Fatalf("ui.status after close: %v", err)
		}
		listed := false
		for _, instance := range status.Instances {
			if instance.UIID == opened.UIID {
				listed = true
			}
		}
		if !listed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("ui.status still lists the leaf: %+v", status)
		}
		time.Sleep(50 * time.Millisecond)
	}
	select {
	case err := <-served:
		if !errors.Is(err, rutis.ErrClosed) {
			t.Fatalf("leaf serve: %v", err)
		}
		t.Logf("leaf session ended after ui.close: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("leaf session did not end after ui.close")
	}
	result, err := supervisor.Shutdown(ctx)
	if err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if !result.Stopped {
		t.Fatalf("daemon was not reclaimed: %+v", result)
	}
}
