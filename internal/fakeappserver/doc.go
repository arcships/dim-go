// Package fakeappserver 提供可编程的假 App Server 与固定样例，供各领域在不接
// 真实服务的情况下验证连接、会话与故障行为。
//
// 三行起一个假服务端并交给真实客户端：
//
//	server, err := fakeappserver.Start(fakeappserver.Script{Sessions: []fakeappserver.Session{...}})
//	defer server.Close()
//	supervisor, err := daemon.New(daemon.Config{DataDir: server.DataDir()})
//
// 能力边界：
//   - 只监听本地 Unix socket，路径与真实客户端一致（<dataDir>/rust/daemon.sock）；
//   - 按脚本应答 hello、会话目录、历史、attach/detach、提交、停止等方法；
//   - 记录收到的每一帧与发出的每一帧，供断言直接读取；
//   - 支持主动推送通知，以及断线、乱序、重复、超大帧、未知方法、版本不符
//     六类故障注入；
//   - 固定样例在 testdata 下，每个样例用 comment/description 说明它验证什么。
//
// 协议帧只由 internal/appserver/protocol 编解码，本包不复制 wire 类型；
// 脚本里未预置 wire 形状的方法（目录、提交、停止等）由调用方给出回执值。
//
// 仅支持 Unix socket。Windows 命名管道与真实平台上的可执行证据仍由 D1 负责，
// 本包在 Windows 上返回明确错误，不假装可用。
package fakeappserver
