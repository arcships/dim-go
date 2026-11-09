//go:build unix

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/session"
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

// waitForHead polls the session's public in-memory view until the branch head
// reaches at least want. It never reads daemon persistence.
func waitForHead(t *testing.T, s *session.Session, want int64, timeout time.Duration) int64 {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var head int64
	for {
		head = s.View(appserver.MainBranchID).Head
		if head >= want {
			return head
		}
		if time.Now().After(deadline) {
			t.Fatalf("branch head did not reach %d within %s (last head %d)", want, timeout, head)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// waitSettled polls the public session.state notice until it reports a settled
// (not running, not stopping) branch, or fails when a running state never
// converges. It exercises the contract that stop converges through
// session.state. A daemon that never emits a state notice is reported, not
// treated as success.
func waitSettled(t *testing.T, s *session.Session, timeout time.Duration) appserver.StateNotice {
	t.Helper()
	deadline := time.Now().Add(timeout)
	last, seen := s.BranchState(appserver.MainBranchID)
	t.Logf("post-stop state: %+v (seen=%v)", last, seen)
	for {
		state, ok := s.BranchState(appserver.MainBranchID)
		if ok && !state.Running && !state.Stopping {
			return state
		}
		if time.Now().After(deadline) {
			if !ok {
				t.Fatalf("daemon emitted no session.state for %s within %s; stop convergence cannot be observed", appserver.MainBranchID, timeout)
			}
			t.Fatalf("session.state did not settle within %s: %+v", timeout, state)
		}
		time.Sleep(25 * time.Millisecond)
	}
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
// submit -> stop -> cleanup path. It is separately opt-in because a daemon may
// be configured with no usable model/provider. The test never inspects daemon
// persistence; every fact comes from protocol replies or the session's public
// in-memory projection.
func TestRealDaemonSessionFlow(t *testing.T) {
	if os.Getenv("DIMAGENT_REAL_SESSION_FLOW") == "" {
		t.Skip("set DIMAGENT_REAL_SESSION_FLOW=1 to run real session acceptance")
	}
	binary := realDaemonBinary(t)
	sup, _ := startRealDaemon(t, binary)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// create: session.create over the control connection, then session.attach
	// and the first session.history window on a dedicated stream.
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
	t.Cleanup(func() { _ = controller.Close(context.Background()) })

	// attach: the dedicated stream was opened against the created session.
	if got := controller.Session.SessionID(); got != created.SessionID {
		t.Fatalf("attached session id = %q, want created id %q", got, created.SessionID)
	}
	if got := controller.Session.AttachHead(); got < 1 {
		t.Fatalf("attach head = %d, want the branch/start head", got)
	}

	// history: a newly created mock session is an intentional empty window. The
	// daemon records branch/start at sequence 1, so attach head is 1 and
	// latestTurn returns that single non-turn event with done=true and no
	// turnStart. The session must accept it as an empty window rather than a
	// protocol error, without a compatibility retry.
	if !controller.Session.Ready() {
		t.Fatal("new session is not ready after attach/history")
	}
	status := controller.Session.History()
	if status.TurnStart != nil || status.HasOlder || status.Degraded {
		t.Fatalf("new session history = %#v, want empty window", status)
	}
	if status.Through < 1 {
		t.Fatalf("new session history through = %d, want the branch/start head", status.Through)
	}
	if view := controller.Session.View(appserver.MainBranchID); view.Head < status.Through {
		t.Fatalf("new session view head = %d, want at least history through %d", view.Head, status.Through)
	}
	if snapshot := controller.Session.Snapshot(); len(snapshot.Messages) != 0 {
		t.Fatalf("new session snapshot = %#v, want empty", snapshot.Messages)
	}
	initialHead := controller.Session.View(appserver.MainBranchID).Head

	// submit: the daemon accepts the input and returns the durable event. The
	// session stream must fold that event into the same public projection.
	text := os.Getenv("DIMAGENT_REAL_SUBMIT_TEXT")
	if text == "" {
		text = "hello from real-daemon acceptance"
	}
	result, submissionID, err := controller.SubmitWithID(ctx, text)
	if err != nil {
		t.Fatalf("session.submit: %v", err)
	}
	if result.Event.ID == "" {
		t.Fatal("session.submit returned empty event id")
	}
	if submissionID == "" {
		t.Fatal("session.submit returned empty submission id")
	}
	if result.Event.Sequence <= initialHead {
		t.Fatalf("submit event sequence = %d, want above initial head %d", result.Event.Sequence, initialHead)
	}
	head := waitForHead(t, controller.Session, result.Event.Sequence, 30*time.Second)
	t.Logf("submit folded: event=%s sequence=%d head=%d", result.Event.ID, result.Event.Sequence, head)

	// branch_create/OpenBranch/EndBranch: a real side line is created through the
	// public method, its own first-screen history is read, and it is ended. A
	// daemon without the branch feature skips this block instead of faking it.
	if conn := controller.Conn; conn != nil && conn.HasFeature("session.branches") {
		// The cut point is the parent branch's sequence the line hangs off; the
		// accepted input's own sequence is the one point the test can prove.
		at := result.Event.Sequence
		createdBranch, err := appserver.BranchCreate(ctx, conn.Client(), appserver.BranchCreateParams{
			SessionID:      created.SessionID,
			ParentBranchID: appserver.MainBranchID,
			At:             &at,
			Prompt:         "side: explain the branch",
		})
		if err != nil {
			t.Fatalf("session.branch_create: %v", err)
		}
		if createdBranch.Branch == nil || createdBranch.Branch.BranchID == "" {
			t.Fatal("session.branch_create returned no branch id")
		}
		opened, err := controller.OpenBranch(ctx, createdBranch.Branch.BranchID)
		if err != nil {
			t.Fatalf("OpenBranch: %v", err)
		}
		if opened.AlreadyLoaded {
			t.Fatal("OpenBranch reported an already loaded branch for a new line")
		}
		if opened.Head < 0 {
			t.Fatalf("OpenBranch head = %d, want a non-negative sequence", opened.Head)
		}
		if err := controller.RefreshBranchDirectory(ctx); err != nil {
			t.Fatalf("RefreshBranchDirectory: %v", err)
		}
		outcome, err := controller.EndBranch(ctx, createdBranch.Branch.BranchID)
		if err != nil {
			t.Fatalf("EndBranch: %v", err)
		}
		if outcome == "" {
			t.Fatal("EndBranch returned an empty outcome")
		}
		t.Logf("branch %s opened at head %d and ended (%s)", createdBranch.Branch.BranchID, opened.Head, outcome)
	} else {
		t.Log("daemon does not advertise session.branches; branch round trip skipped")
	}

	// stop: session.stop is accepted and the stopped state converges through
	// session.state. The stop reply body is intentionally ignored; only the
	// notification is authoritative.
	if err := controller.Stop(ctx); err != nil {
		t.Fatalf("session.stop: %v", err)
	}
	settled := waitSettled(t, controller.Session, 30*time.Second)
	if settled.BranchID != appserver.MainBranchID {
		t.Fatalf("settled state branch = %q, want %q", settled.BranchID, appserver.MainBranchID)
	}
	if settled.SessionID != created.SessionID {
		t.Fatalf("settled state session = %q, want %q", settled.SessionID, created.SessionID)
	}

	// cleanup: detach the session stream, then close the owned daemon and assert
	// it actually shut down. The supervisor cleanup is a no-op after this.
	detachCtx, cancelDetach := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelDetach()
	if err := controller.Close(detachCtx); err != nil {
		t.Fatalf("session detach: %v", err)
	}
	closeCtx, cancelClose := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelClose()
	report, err := sup.Close(closeCtx)
	if err != nil {
		t.Fatalf("close real daemon: %v", err)
	}
	if !report.Owned || !report.Shutdown {
		t.Fatalf("close report = %+v, want owned and shutdown", report)
	}
}
