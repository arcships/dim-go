//go:build unix

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
)

func realDaemonBinary(t *testing.T) string {
	t.Helper()
	binary := os.Getenv("DIMAGENT_REAL_APP_SERVER_BIN")
	if binary == "" {
		t.Skip("set DIMAGENT_REAL_APP_SERVER_BIN to run the real-daemon acceptance tests")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("real app server binary: %v", err)
	}
	return binary
}

func startRealDaemon(t *testing.T, binary string) (*supervisor.Supervisor, *supervisor.Connection) {
	t.Helper()
	home, err := os.MkdirTemp("/tmp", "dimagent-smoke-")
	if err != nil {
		t.Fatalf("create short product home: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	sup, err := supervisor.New(supervisor.Config{
		ProductHome: home, ServerPath: binary, MockModel: true,
		ClientName: "dimagent-mygo-real-smoke",
		ListenWait: supervisor.ListenWait{Timeout: 30 * time.Second, Interval: 50 * time.Millisecond},
	})
	if err != nil {
		t.Fatalf("new supervisor: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := sup.Close(ctx); err != nil {
			t.Logf("cleanup close real daemon: %v", err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	conn, err := sup.Start(ctx)
	if err != nil {
		t.Fatalf("start real daemon: %v", err)
	}
	return sup, conn
}

// TestRealDaemonSmoke is opt-in because it starts a real Rust daemon. It checks
// only the public control plane; no daemon files or private state are read.
func TestRealDaemonSmoke(t *testing.T) {
	binary := realDaemonBinary(t)
	sup, conn := startRealDaemon(t, binary)
	hello := conn.Hello()
	if hello.ProtocolVersion != appserver.WireProtocolVersion {
		t.Fatalf("protocol version = %d, want %d", hello.ProtocolVersion, appserver.WireProtocolVersion)
	}
	if hello.InstanceID == "" {
		t.Fatal("hello instanceId is empty")
	}
	if hello.DaemonProfile.Kind != appserver.DaemonProfileMock {
		t.Fatalf("daemon profile = %q, want %q", hello.DaemonProfile.Kind, appserver.DaemonProfileMock)
	}
	if !conn.HasFeature("session.catalog") {
		t.Fatal("daemon did not advertise session.catalog")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := Catalog(ctx, sup); err != nil {
		t.Fatalf("catalog: %v", err)
	}
	closeCtx, cancelClose := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelClose()
	report, err := sup.Close(closeCtx)
	if err != nil {
		t.Fatalf("close real daemon: %v", err)
	}
	if !report.Owned || !report.Shutdown {
		t.Fatalf("close report = %+v, want owned and shutdown", report)
	}
}

// TestRealDaemonSessionFlow validates the public create -> attach -> history ->
// submit path. It is separately opt-in because a daemon may be configured with
// no usable model/provider. The test never inspects daemon persistence.
func TestRealDaemonSessionFlow(t *testing.T) {
	if os.Getenv("DIMAGENT_REAL_SESSION_FLOW") == "" {
		t.Skip("set DIMAGENT_REAL_SESSION_FLOW=1 to run real session acceptance")
	}
	binary := realDaemonBinary(t)
	sup, _ := startRealDaemon(t, binary)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	controller, created, err := CreateSession(ctx, sup, appserver.CreateParams{Cwd: os.TempDir()})
	if err != nil {
		t.Fatalf("session.create/attach/history: %v", err)
	}
	if created.SessionID == "" {
		t.Fatal("session.create returned empty session id")
	}
	if controller.Session == nil {
		t.Fatal("session flow returned nil session")
	}
	if got := controller.Session.SessionID(); got != created.SessionID {
		t.Fatalf("attached session id = %q, want created id %q", got, created.SessionID)
	}
	t.Cleanup(func() { _ = controller.Close(context.Background()) })
	text := os.Getenv("DIMAGENT_REAL_SUBMIT_TEXT")
	if text == "" {
		text = "hello from real-daemon acceptance"
	}
	result, err := controller.Submit(ctx, text)
	if err != nil {
		t.Fatalf("session.submit: %v", err)
	}
	if result.Event.ID == "" {
		t.Fatal("session.submit returned empty event id")
	}
}
