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

// TestRealDaemonSmoke is opt-in because it starts a real Rust daemon. It only
// checks the control plane; session execution and UI behavior need separate
// acceptance coverage.
func TestRealDaemonSmoke(t *testing.T) {
	binary := os.Getenv("DIMAGENT_REAL_APP_SERVER_BIN")
	if binary == "" {
		t.Skip("set DIMAGENT_REAL_APP_SERVER_BIN to run the real-daemon smoke")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("real app server binary: %v", err)
	}

	home, err := os.MkdirTemp("/tmp", "dimagent-smoke-")
	if err != nil {
		t.Fatalf("create short product home: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })

	sup, err := supervisor.New(supervisor.Config{
		ProductHome: home,
		ServerPath:  binary,
		MockModel:   true,
		ClientName:  "dimagent-mygo-real-smoke",
		ListenWait: supervisor.ListenWait{
			Timeout:  30 * time.Second,
			Interval: 50 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("new supervisor: %v", err)
	}
	t.Cleanup(func() {
		closeCtx, cancelClose := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelClose()
		if _, err := sup.Close(closeCtx); err != nil {
			t.Logf("cleanup close real daemon: %v", err)
		}
	})

	startCtx, cancelStart := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelStart()
	conn, err := sup.Start(startCtx)
	if err != nil {
		t.Fatalf("start real daemon: %v", err)
	}

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

	catalogCtx, cancelCatalog := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelCatalog()
	if _, err := Catalog(catalogCtx, sup); err != nil {
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
