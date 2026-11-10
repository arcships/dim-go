package daemon

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/transport"
)

func TestResolveEndpointCreatesPrivateHome(t *testing.T) {
	dataDir := t.TempDir()
	ep, err := resolveEndpoint(dataDir)
	if err != nil {
		t.Fatalf("resolve endpoint: %v", err)
	}
	nativeHome := filepath.Join(dataDir, "rust")
	info, err := os.Stat(nativeHome)
	if err != nil {
		t.Fatalf("native home: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("native home must be private, mode=%v", info.Mode().Perm())
	}
	if runtime.GOOS == "windows" {
		if ep.network.Network != transport.NetworkPipe {
			t.Fatalf("windows network: %q", ep.network.Network)
		}
		if !strings.HasPrefix(ep.network.Address, `\\.\pipe\dim-rust-`) {
			t.Fatalf("windows pipe name: %q", ep.network.Address)
		}
		return
	}
	if ep.network.Network != transport.NetworkUnix {
		t.Fatalf("unix network: %q", ep.network.Network)
	}
	if filepath.Dir(ep.network.Address) != ep.nativeHome {
		t.Fatalf("socket must live in the native home: %q / %q", ep.network.Address, ep.nativeHome)
	}
	if filepath.Base(ep.network.Address) != "daemon.sock" {
		t.Fatalf("socket name: %q", ep.network.Address)
	}
}

func TestResolveEndpointRejectsNonPrivateHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission check")
	}
	dataDir := t.TempDir()
	nativeHome := filepath.Join(dataDir, "rust")
	if err := os.MkdirAll(nativeHome, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := resolveEndpoint(dataDir); err == nil {
		t.Fatal("want an error for a non-private native home")
	}
}

func TestBuildDaemonArgs(t *testing.T) {
	unixEndpoint := endpoint{nativeHome: "/data/rust", network: transport.Endpoint{
		Network: transport.NetworkUnix,
		Address: "/data/rust/daemon.sock",
	}}

	args := buildDaemonArgs(Config{DataDir: "/data", MockModel: true}, unixEndpoint)
	joined := strings.Join(args, " ")
	for _, want := range []string{"--unix /data/rust/daemon.sock", "--no-ws", "--home /data/rust", "--mock-model"} {
		if !strings.Contains(joined, want) {
			t.Errorf("mock args missing %q: %s", want, joined)
		}
	}
	if strings.Contains(joined, "--product-home") {
		t.Errorf("mock model must not use --product-home: %s", joined)
	}

	args = buildDaemonArgs(Config{DataDir: "/data", ModelConfig: "/tmp/model.json"}, unixEndpoint)
	joined = strings.Join(args, " ")
	if !strings.Contains(joined, "--model-config /tmp/model.json") {
		t.Errorf("model config missing: %s", joined)
	}

	args = buildDaemonArgs(Config{DataDir: "/data"}, unixEndpoint)
	joined = strings.Join(args, " ")
	if !strings.Contains(joined, "--product-home /data") {
		t.Errorf("product home missing: %s", joined)
	}

	pipeEndpoint := endpoint{nativeHome: `C:\data\rust`, network: transport.Endpoint{
		Network: transport.NetworkPipe,
		Address: `\\.\pipe\dim-rust-test`,
	}}
	args = buildDaemonArgs(Config{DataDir: `C:\data`, MockModel: true}, pipeEndpoint)
	if !strings.Contains(strings.Join(args, " "), "--pipe") {
		t.Errorf("pipe args: %v", args)
	}
}

func TestWindowsPipeNameIsStable(t *testing.T) {
	first := windowsPipeName("/data/rust")
	second := windowsPipeName("/data/rust")
	if first != second {
		t.Fatalf("pipe name is not stable: %q / %q", first, second)
	}
	if windowsPipeName("/data/other") == first {
		t.Fatal("different homes must give different pipe names")
	}
	if !strings.HasPrefix(first, `\\.\pipe\dim-rust-`) || len(first) != len(`\\.\pipe\dim-rust-`)+32 {
		t.Fatalf("unexpected pipe name: %q", first)
	}
}
