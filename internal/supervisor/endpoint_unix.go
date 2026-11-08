//go:build unix

package supervisor

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// ensureNativeHome creates the native home when missing and enforces §18's
// privacy rules on an existing one.
func ensureNativeHome(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("supervisor: create native home %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("supervisor: stat native home %s: %w", path, err)
	}
	return validateNativeHomeDir(path, info, os.Geteuid())
}

// validateNativeHomeDir enforces that path is a directory with exactly mode
// 0700 and owned by euid. A wrong owner or mode is a local configuration error,
// not "daemon absent": callers must surface it instead of spawning.
func validateNativeHomeDir(path string, info os.FileInfo, euid int) error {
	if info == nil || !info.IsDir() {
		return fmt.Errorf("%w: %s is not a directory", ErrInsecureNativeHome, path)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		return fmt.Errorf("%w: %s has mode %04o, want 0700", ErrInsecureNativeHome, path, perm)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("%w: %s has no owner information", ErrInsecureNativeHome, path)
	}
	if int(stat.Uid) != euid {
		return fmt.Errorf("%w: %s is owned by uid %d, want %d", ErrInsecureNativeHome, path, stat.Uid, euid)
	}
	return nil
}

func endpointFor(nativeHome string) (Endpoint, error) {
	return Endpoint{Network: NetworkUnix, Address: filepath.Join(nativeHome, daemonSocketName)}, nil
}
