//go:build windows

package supervisor

import (
	"fmt"
	"os"
)

// ensureNativeHome creates the native home when missing. The Windows pipe is
// private by construction, so no mode or owner check applies.
func ensureNativeHome(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("supervisor: create native home %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("supervisor: stat native home %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("supervisor: native home %s is not a directory", path)
	}
	return nil
}

func endpointFor(nativeHome string) (Endpoint, error) {
	return Endpoint{Network: NetworkPipe, Address: windowsPipeName(nativeHome)}, nil
}
