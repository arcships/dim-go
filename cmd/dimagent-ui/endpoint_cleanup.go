package main

// 退出清理：正常退出后删掉自家拉起的端点文件（<数据目录>/rust/daemon.sock）。
//
// 为什么在入口做：端点文件是自家拉起的实例在数据目录里留下的唯一痕迹，服务端被
// 杀或旧版本没自己收走时会一直留着；它会让下一次启动的发现链多一条「端点残留」，
// 也让外部验收看到不该存在的文件。
//
// 只删「文件在、但没人在听」的端点：外部实例还活着时拨号能连上，那是它的
// socket，不动它。删不掉只记日志，退出路径不因为这一步失败而卡住。
//
// 复用已有 daemon 时本端端点是一条软链（见 reuse.go）：它只是本次接入的路径，
// 退出就移除；软链指向的对方 socket 不动。

import (
	"os"
	"runtime"
)

// cleanupEndpoint 删掉没有实例在监听的残留端点；nil 安全，Windows 不适用。
func cleanupEndpoint(path string, log *startupLog) {
	if path == "" || runtime.GOOS == "windows" {
		// Windows 用命名管道，没有可删的 socket 文件。
		return
	}
	if removeBridge(path, log) {
		return
	}
	if _, err := os.Stat(path); err != nil {
		// 没有残留：服务端自己收走了，或者这次压根没起过实例。
		return
	}
	alive, reason := endpointAlive(path)
	if alive {
		log.printf("退出：端点 %s 仍有实例在监听，保留", path)
		return
	}
	if reason == "" {
		reason = "连不上"
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.printf("退出：删除残留端点 %s 失败：%v", path, err)
		return
	}
	log.printf("退出：已删除残留端点 %s（%s）", path, reason)
}
