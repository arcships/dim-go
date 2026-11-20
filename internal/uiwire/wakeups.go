package uiwire

import (
	"context"
	"time"

	"github.com/arcships/dimagent-mygo/internal/extensions"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// 唤醒记录的读数（右栏任务面板「唤醒」子标签，K-35 / K-36）。
//
// 数据来源只有 `session.wakeup.list`（extensions.Service 的只读镜像）。界面每帧
// 读最近一次读数，读取本身按「会话 + 分支」只派一次：读回的通知由扩展域自己
// 订阅，变更到了再重绘。
const (
	// wakeupReadTimeout 是一次唤醒读取的上限：超时按失败写进读数，不挂住。
	wakeupReadTimeout = 8 * time.Second
)

// Wakeups 把最近一次唤醒读数映射成界面快照。
func (w *App) Wakeups() ui.WakeupsSnapshot {
	service := w.A.Extensions()
	if service == nil {
		return ui.WakeupsSnapshot{}
	}
	snapshot := service.WakeupSnapshot(extensions.WakeupScope{
		SessionID: w.currentSession(), BranchID: w.currentBranch(),
	})
	out := ui.WakeupsSnapshot{
		ReadError:         snapshot.ReadError,
		CapabilityMissing: snapshot.CapabilityMissing,
	}
	for _, item := range snapshot.Items {
		out.Items = append(out.Items, ui.WakeupSnapshot{
			ID:        item.ID,
			TriggerAt: item.TriggerAt,
			Prompt:    item.Prompt,
			State:     string(item.Status),
		})
	}
	return out
}

// requestWakeups 为当前会话 / 分支排一次唤醒读取；同一范围只问一次。
//
// 界面每帧读 Session()，所以这里只登记与派发，不等待：结果到了再触发重绘。
func (w *App) requestWakeups(v viewIdentity) {
	if v.sessionID == "" {
		return
	}
	key := v.sessionID + "\x00" + v.branchID
	w.wakeupsMu.Lock()
	asked := w.wakeupsAsked[key]
	w.wakeupsAsked[key] = true
	w.wakeupsMu.Unlock()
	if asked {
		return
	}
	go w.refreshWakeups(v.sessionID, v.branchID)
}

// refreshWakeups 读一次唤醒记录，存进扩展域的读数；失败原因由下一次读快照给出。
func (w *App) refreshWakeups(sessionID, branchID string) {
	service := w.A.Extensions()
	if service == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), wakeupReadTimeout)
	defer cancel()
	_, _ = service.ListWakeups(ctx, extensions.WakeupScope{
		SessionID: sessionID, BranchID: branchID,
	})
	w.changed()
}

// viewIdentity 是派发读取要的会话与分支。
type viewIdentity struct {
	sessionID string
	branchID  string
}
