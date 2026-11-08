//go:build windows

package supervisor

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveLinkWindowsUsesPipe(t *testing.T) {
	home := t.TempDir()
	link, err := ResolveLink(home)
	if err != nil {
		t.Fatalf("ResolveLink: %v", err)
	}
	if link.Endpoint.Network != NetworkPipe {
		t.Fatalf("Endpoint.Network = %q, want %q", link.Endpoint.Network, NetworkPipe)
	}
	if want := windowsPipeName(link.NativeHome); link.Endpoint.Address != want {
		t.Fatalf("Endpoint.Address = %q, want %q", link.Endpoint.Address, want)
	}
	if !strings.HasPrefix(link.Endpoint.Address, windowsPipePrefix) {
		t.Fatalf("pipe %q lacks prefix %q", link.Endpoint.Address, windowsPipePrefix)
	}
	if filepath.Base(link.NativeHome) != nativeHomeName {
		t.Fatalf("NativeHome = %q, want a %q directory", link.NativeHome, nativeHomeName)
	}
}
