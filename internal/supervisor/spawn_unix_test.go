//go:build unix

package supervisor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	process, err := DefaultSpawn(ctx, cfg)
	if err != nil {
		t.Fatalf("DefaultSpawn: %v", err)
	}
	if process.Pid() <= 0 {
		t.Fatalf("pid = %d, want a real child", process.Pid())
	}
	select {
	case waitErr := <-process.Done():
		if waitErr != nil {
			t.Fatalf("child exit: %v", waitErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("spawned child did not exit")
	}
	info, err := os.Stat(filepath.Join(home, daemonLogName))
	if err != nil {
		t.Fatalf("stat daemon log: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("daemon log is empty; child stderr was not redirected")
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("daemon log mode = %04o, want 0600", perm)
	}
}
