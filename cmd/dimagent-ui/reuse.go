package main

// 复用已有 daemon：把对方的 socket 接进本端数据目录。
//
// 为什么不直接把数据目录指到对方的 home：偏好、日志与端点都挂在数据目录下，指过去
// 就会往对方的 home 写东西（AGENTS：复用时本端偏好与日志仍落自己的产品目录）。这里
// 只在 <数据目录>/rust/daemon.sock 放一条指向对方 socket 的软链，连接领域照旧按自己的
// 端点连过去，写盘全部留在本端目录；对方的 home 只被拨号一次，不被改写。
//
// 退出时这条软链由 cleanupEndpoint 移除（见 endpoint_cleanup.go）：它只是本次接入的
// 路径，不是本端拉起的实例留下的端点。

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// bridgeEndpoint 让本端端点指向已有 daemon 的 socket。
//
// 本端端点若已有残留文件（上次实例崩溃留下的 socket），先确认没人监听再删掉它：
// 发现链认定过它不可用，但删之前再探一次，避免把刚被别人占用的端点拔掉。
func bridgeEndpoint(ownSocket, daemonSocket string, log *startupLog) error {
	if runtime.GOOS == "windows" {
		// Windows 用命名管道，没有可以放软链的 socket 文件。
		return fmt.Errorf("Windows 用命名管道，不能把已有 daemon 接进本端端点")
	}
	if ownSocket == "" || daemonSocket == "" {
		return fmt.Errorf("缺少端点路径（本端 %q，已有 daemon %q）", ownSocket, daemonSocket)
	}
	if filepath.Clean(ownSocket) == filepath.Clean(daemonSocket) {
		// 已有 daemon 就在本端数据目录里：不用接。
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(ownSocket), 0o700); err != nil {
		return fmt.Errorf("准备本端端点目录: %w", err)
	}

	info, err := os.Lstat(ownSocket)
	switch {
	case err == nil && info.Mode()&os.ModeSymlink != 0:
		if target, readErr := os.Readlink(ownSocket); readErr == nil &&
			filepath.Clean(target) == filepath.Clean(daemonSocket) {
			return nil
		}
		// 指向别处：只删链接本身，目标 socket 是对方的，不动。
		if err := os.Remove(ownSocket); err != nil {
			return fmt.Errorf("替换本端复用端点桥: %w", err)
		}
	case err == nil:
		if alive, _ := endpointAlive(ownSocket); alive {
			return fmt.Errorf("本端端点仍有实例在监听：%s", ownSocket)
		}
		if err := os.Remove(ownSocket); err != nil {
			return fmt.Errorf("清理本端残留端点: %w", err)
		}
	case !os.IsNotExist(err):
		return fmt.Errorf("检查本端端点: %w", err)
	}

	if err := os.Symlink(daemonSocket, ownSocket); err != nil {
		return fmt.Errorf("把已有 daemon 接进本端端点: %w", err)
	}
	if log != nil {
		log.printf("复用接入：%s → %s", ownSocket, daemonSocket)
	}
	return nil
}

// removeBridge 删掉复用接入的软链；返回是否删了。
//
// 只删软链：目标 socket 属于对方的 daemon，本端不碰。真实端点文件（自己拉起的实例
// 留下的）不走这里，见 cleanupEndpoint。
func removeBridge(path string, log *startupLog) bool {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	if err := os.Remove(path); err != nil {
		if log != nil {
			log.printf("退出：移除复用接入 %s 失败：%v", path, err)
		}
		return false
	}
	if log != nil {
		log.printf("退出：已移除复用接入 %s", path)
	}
	return true
}
