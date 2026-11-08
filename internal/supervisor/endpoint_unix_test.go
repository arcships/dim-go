//go:build unix

package supervisor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLinkUnix(t *testing.T) {
	home := t.TempDir()
	link, err := ResolveLink(home)
	if err != nil {
		t.Fatalf("ResolveLink: %v", err)
	}
	wantNative, err := filepath.EvalSymlinks(filepath.Join(home, nativeHomeName))
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	if link.NativeHome != wantNative {
		t.Fatalf("NativeHome = %q, want %q", link.NativeHome, wantNative)
	}
	wantEndpoint := Endpoint{Network: NetworkUnix, Address: filepath.Join(wantNative, daemonSocketName)}
	if link.Endpoint != wantEndpoint {
		t.Fatalf("Endpoint = %v, want %v", link.Endpoint, wantEndpoint)
	}
	if link.ProductHome != home {
		t.Fatalf("ProductHome = %q, want %q", link.ProductHome, home)
	}
	info, err := os.Stat(filepath.Join(home, nativeHomeName))
	if err != nil {
		t.Fatalf("stat native home: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Fatalf("native home mode = %04o, want 0700", perm)
	}
}

// TestEndpointForUnix pins the Unix endpoint selection shape: the socket is
// `<canonical native home>/daemon.sock`.
func TestEndpointForUnix(t *testing.T) {
	home := t.TempDir()
	endpoint, err := endpointFor(home)
	if err != nil {
		t.Fatalf("endpointFor: %v", err)
	}
	want := Endpoint{Network: NetworkUnix, Address: filepath.Join(home, daemonSocketName)}
	if endpoint != want {
		t.Fatalf("endpointFor(%q) = %v, want %v", home, endpoint, want)
	}
}

// TestResolveLinkRelativeProductHome pins that endpoint selection works from a
// relative product home and that the endpoint is built from the absolute
// canonical native home while ProductHome keeps the caller's value.
func TestResolveLinkRelativeProductHome(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	link, err := ResolveLink("product")
	if err != nil {
		t.Fatalf("ResolveLink: %v", err)
	}
	wantNative, err := filepath.EvalSymlinks(filepath.Join(root, "product", nativeHomeName))
	if err != nil {
		t.Fatalf("eval native home: %v", err)
	}
	if link.NativeHome != wantNative {
		t.Fatalf("NativeHome = %q, want %q", link.NativeHome, wantNative)
	}
	wantEndpoint := Endpoint{Network: NetworkUnix, Address: filepath.Join(wantNative, daemonSocketName)}
	if link.Endpoint != wantEndpoint {
		t.Fatalf("Endpoint = %v, want %v", link.Endpoint, wantEndpoint)
	}
	if link.ProductHome != "product" {
		t.Fatalf("ProductHome = %q, want the caller value %q", link.ProductHome, "product")
	}
}

func TestResolveLinkCreatesNativeHome(t *testing.T) {
	parent := t.TempDir()
	home := filepath.Join(parent, "nested", "product")
	link, err := ResolveLink(home)
	if err != nil {
		t.Fatalf("ResolveLink: %v", err)
	}
	info, err := os.Stat(link.NativeHome)
	if err != nil {
		t.Fatalf("stat native home: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("native home was not created as a directory")
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Fatalf("native home mode = %04o, want 0700", perm)
	}
}

func TestResolveLinkRejectsInsecureMode(t *testing.T) {
	for _, mode := range []os.FileMode{0o755, 0o750, 0o701, 0o770, 0o777, 0o600} {
		t.Run(mode.String(), func(t *testing.T) {
			home := t.TempDir()
			native := filepath.Join(home, nativeHomeName)
			if err := os.Mkdir(native, 0o700); err != nil {
				t.Fatalf("mkdir native home: %v", err)
			}
			if err := os.Chmod(native, mode); err != nil {
				t.Fatalf("chmod native home: %v", err)
			}
			if _, err := ResolveLink(home); !errors.Is(err, ErrInsecureNativeHome) {
				t.Fatalf("ResolveLink error = %v, want ErrInsecureNativeHome", err)
			}
		})
	}
}

func TestValidateNativeHomeRejectsOtherOwner(t *testing.T) {
	home := t.TempDir()
	native := filepath.Join(home, nativeHomeName)
	if err := os.Mkdir(native, 0o700); err != nil {
		t.Fatalf("mkdir native home: %v", err)
	}
	info, err := os.Stat(native)
	if err != nil {
		t.Fatalf("stat native home: %v", err)
	}
	if err := validateNativeHomeDir(native, info, os.Geteuid()); err != nil {
		t.Fatalf("current owner rejected: %v", err)
	}
	if err := validateNativeHomeDir(native, info, os.Geteuid()+1); !errors.Is(err, ErrInsecureNativeHome) {
		t.Fatalf("other owner error = %v, want ErrInsecureNativeHome", err)
	}
}

func TestValidateNativeHomeRejectsFile(t *testing.T) {
	home := t.TempDir()
	file := filepath.Join(home, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if err := validateNativeHomeDir(file, info, os.Geteuid()); !errors.Is(err, ErrInsecureNativeHome) {
		t.Fatalf("file error = %v, want ErrInsecureNativeHome", err)
	}
}

func TestResolveLinkCanonicalizesSymlink(t *testing.T) {
	realHome := t.TempDir()
	linkDir := t.TempDir()
	symlink := filepath.Join(linkDir, "product")
	if err := os.Symlink(realHome, symlink); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	link, err := ResolveLink(symlink)
	if err != nil {
		t.Fatalf("ResolveLink: %v", err)
	}
	wantNative, err := filepath.EvalSymlinks(filepath.Join(realHome, nativeHomeName))
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	if link.NativeHome != wantNative {
		t.Fatalf("NativeHome = %q, want canonical %q", link.NativeHome, wantNative)
	}
	if link.Endpoint.Address != filepath.Join(wantNative, daemonSocketName) {
		t.Fatalf("socket path = %q, want under %q", link.Endpoint.Address, wantNative)
	}
	if link.ProductHome != symlink {
		t.Fatalf("ProductHome = %q, want the caller value %q", link.ProductHome, symlink)
	}
}

func TestResolveLinkRejectsBadProductHome(t *testing.T) {
	for _, home := range []string{"", "   "} {
		if _, err := ResolveLink(home); err == nil {
			t.Fatalf("ResolveLink(%q) succeeded, want an error", home)
		}
	}
	parent := t.TempDir()
	file := filepath.Join(parent, "product")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := ResolveLink(file); err == nil {
		t.Fatal("ResolveLink with a regular-file product home succeeded")
	}
}
