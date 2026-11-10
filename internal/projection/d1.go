package projection

import (
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
)

// ApplyListenerEvent 折叠 D1 会话流送来的一条实时通知。
//
// D1 的 session.Listener.OnEvent 参数是 {Method, Params}，与本包 ApplyNotice
// 的入参一一对应；本方法只是把这条交界写清楚，消费方不需要自己拆信封。
func (f *Folder) ApplyListenerEvent(event session.Event) []Rejection {
	return f.ApplyNotice(event.Method, event.Params)
}

// ApplyListenerHistory 折叠 D1 断线重读的结果（session.Listener.OnHistory）。
//
// D1 的会话流固定主线，因此重读结果折叠进 main；重读覆盖到的序号会把尾部行
// 换成正式标识，不会叠加出第二条。
func (f *Folder) ApplyListenerHistory(replay session.HistoryReplay) []Rejection {
	return f.ApplyHistory(mainBranch, replay.History)
}

// ApplySnapshot 折叠 D1 打开会话时给的初始快照。快照里的历史是初始窗口，
// 向前翻页仍由调用方按 NextBefore 发起并交给 ApplyHistory。
func (f *Folder) ApplySnapshot(snapshot *session.Snapshot) []Rejection {
	if snapshot == nil {
		return nil
	}
	return f.ApplyFirstPage(mainBranch, snapshot.History)
}

// ApplySnapshotBranch 折叠指定分支的初始快照。
func (f *Folder) ApplySnapshotBranch(branchID string, snapshot *session.Snapshot) []Rejection {
	if snapshot == nil {
		return nil
	}
	return f.ApplyFirstPage(branchID, snapshot.History)
}

// Listen 把本包接上 D1 的会话流，返回填好的监听器。
//
// 用法与 D1 的 Manager 一致：先 Acquire 拿 snapshot 与 lease，再把它交给本包。
// 非法事实的处置由调用方决定：这里只把它们收集起来，不向用户报错。
func Listen(folder *Folder, onReject func(Rejection)) session.Listener {
	listener := session.Listener{
		OnEvent: func(event session.Event) {
			rejections := folder.ApplyListenerEvent(event)
			reportRejections(rejections, onReject)
		},
		OnHistory: func(replay session.HistoryReplay) {
			rejections := folder.ApplyListenerHistory(replay)
			reportRejections(rejections, onReject)
		},
		// 断线先清空未落库尾部，再按序号重读；重读覆盖到的事实会重新折叠进来。
		OnResync: func(session.Resync) {
			folder.OnDisconnect(mainBranch)
		},
	}
	return listener
}

func reportRejections(rejections []Rejection, onReject func(Rejection)) {
	if onReject == nil {
		return
	}
	for _, rejection := range rejections {
		onReject(rejection)
	}
}
