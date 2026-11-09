//go:build unix

package supervisor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// waitChildExit waits for a spawned child to be reaped. The healthy path is
// event-driven (cmd.Wait publishes the exit), so there is no fixed observation
// window that can expire while the OS is still scheduling the child. The bound
// is only a safety net and is derived from the test deadline instead of a
// hardcoded sleep-sized timeout.
func waitChildExit(t *testing.T, process Process) {
	t.Helper()
	timeout := 30 * time.Second
	if deadline, ok := t.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < timeout {
			timeout = remaining
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case waitErr := <-process.Done():
		if waitErr != nil {
			t.Fatalf("child exit: %v", waitErr)
		}
	case <-timer.C:
		t.Fatalf("spawned child did not exit within %s", timeout)
	}
}

func TestDefaultSpawnRunsDetachedAndLogs(t *testing.T) {
	script := filepath.Join(t.TempDir(), "fake-daemon.sh")
	content := "#!/bin/sh\necho spawned >&2\nexit 0\n"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatalf("write script: %v", err)
	}
	home := t.TempDir()
	cfg := SpawnConfig{
		ServerPath:  script,
		Endpoint:    Endpoint{Network: NetworkUnix, Address: filepath.Join(t.TempDir(), "daemon.sock")},
		NativeHome:  home,
		ProductHome: t.TempDir(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	process, err := DefaultSpawn(ctx, cfg)
	if err != nil {
		t.Fatalf("DefaultSpawn: %v", err)
	}
	if process.Pid() <= 0 {
		t.Fatalf("pid = %d, want a real child", process.Pid())
	}
	waitChildExit(t, process)
	// The child wrote stderr before exiting, so the bytes are settled once the
	// process is reaped; asserting content (not just size) keeps a truncated or
	// misdirected log from passing.
	logPath := filepath.Join(home, daemonLogName)
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("stat daemon log: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("daemon log mode = %04o, want 0600", perm)
	}
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read daemon log: %v", err)
	}
	if !strings.Contains(string(logged), "spawned") {
		t.Fatalf("daemon log = %q, want child stderr", logged)
	}
}
