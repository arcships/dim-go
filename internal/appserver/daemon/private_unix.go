//go:build !windows

package daemon

import (
	"fmt"
	"os"
	"syscall"
)

// checkPrivateDir 要求 App Server home 是 0700 且属于当前用户。
func checkPrivateDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("App Server home 不是目录: %s", path)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("App Server home 必须是 0700: %s", path)
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && int(stat.Uid) != os.Getuid() {
		return fmt.Errorf("App Server home 必须属于当前用户: %s", path)
	}
	return nil
}
