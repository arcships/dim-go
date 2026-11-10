//go:build windows

package daemon

import (
	"fmt"
	"os"
)

// checkPrivateDir 在 Windows 上只确认目录存在；命名管道的访问控制由管道
// 自身的 ACL 负责，后续项与管道传输一起实现。
func checkPrivateDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("App Server home 不是目录: %s", path)
	}
	return nil
}
