package projection

import (
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// 分页状态机名，诊断里用。
const machinePaging = "向前翻页"

// PageWindow 是一个分支的分页位置。
//
//   - Anchor 是最早已装载位置的序号（首屏是该轮起点，向前翻页是该页最早事实）：
//     视图按它判断"还差多少没读"，只由历史页向更早推进，实时事实永远不动它。
//   - Cursor 是下一次向前翻页的 before 游标：优先用服务端给的 nextBefore，否则用
//     锚点。
//   - HasOlder 表示服务端说还有更早的轮次。
type PageWindow struct {
	Anchor   int64
	Cursor   int64
	HasOlder bool
}

// ApplyFirstPage 折叠"最近一轮"的首屏历史页，并据此设置分页位置。
//
// 打开会话与断线重开都走这里：首屏只覆盖最近一轮，更早的轮次靠向前翻页补齐。
func (f *Folder) ApplyFirstPage(branchID string, page protocol.SessionHistoryResult) []Rejection {
	f.mu.Lock()
	defer f.mu.Unlock()
	branch := f.branchLocked(branchID)
	rejections := f.applyPageLocked(branch, page, 0)
	f.advanceWindowLocked(branch, page, 0)
	return rejections
}

// OlderPageRequest 给出下一次向前翻页要发的读取参数。没有更早的轮次时返回 false。
//
// Before 是排他上界：服务端返回序号小于它、且落在它之前最近一轮的那一页。
func (f *Folder) OlderPageRequest(branchID string, limit int) (protocol.SessionHistoryParams, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	branch := f.branchLocked(branchID)
	if !branch.hasOlder {
		return protocol.SessionHistoryParams{}, false
	}
	cursor := branch.cursor
	if cursor <= 0 {
		cursor = branch.anchor
	}
	if cursor <= 0 {
		return protocol.SessionHistoryParams{}, false
	}
	params := protocol.SessionHistoryParams{BranchID: branch.id, Before: &cursor}
	if limit > 0 {
		params.Limit = &limit
	}
	return params, true
}

// ApplyOlderPage 折叠一页向前翻页的结果。
//
// 只接受序号小于本次请求游标的事实：越界的事实被丢弃并带原因，不改变视图。
// 重复页不产生重复行，锚点也不会因为重复页往前挪。
func (f *Folder) ApplyOlderPage(branchID string, page protocol.SessionHistoryResult) []Rejection {
	f.mu.Lock()
	defer f.mu.Unlock()
	branch := f.branchLocked(branchID)

	bound := branch.cursor
	if bound <= 0 {
		bound = branch.anchor
	}
	known := len(branch.events)
	rejections := f.applyPageLocked(branch, page, bound)
	f.advanceWindowLocked(branch, page, bound)

	// 这一页没有带来任何新事实（重复页、空页）：再要一页也只会原地打转，停在这里并记诊断。
	if len(branch.events) == known && branch.hasOlder {
		branch.hasOlder = false
		f.recordDiagnosticLocked(Diagnostic{
			Machine: machinePaging,
			From:    "cursor",
			Event:   "older_page",
			Reason:  "这一页没有带来更早的事实，停止继续向前翻页",
		})
	}
	return rejections
}

// Window 返回分支的分页位置。
func (f *Folder) Window(branchID string) PageWindow {
	f.mu.Lock()
	defer f.mu.Unlock()
	branch := f.branchLocked(branchID)
	return PageWindow{Anchor: branch.anchor, Cursor: branch.cursor, HasOlder: branch.hasOlder}
}

// HasOlder 报告是否还有更早的轮次没装载。
func (f *Folder) HasOlder(branchID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.branchLocked(branchID).hasOlder
}

// ResetWindow 清空分页位置，让下一次首屏读取重新设锚点。
// 打开会话、断线重开（A 文档的"重设锚点重读"）都走这里。
func (f *Folder) ResetWindow(branchID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	branch := f.branchLocked(branchID)
	branch.anchor = 0
	branch.cursor = 0
	branch.hasOlder = false
}

// advanceWindowLocked 用一页历史推进分页位置。
//
// 锚点只向更早推进：重复页、重读首屏都不会把锚点往前挪，因此"已装载到哪一轮"
// 不会因为读取顺序而回退。游标优先后端给的 nextBefore，其次用锚点。
func (f *Folder) advanceWindowLocked(branch *branchState, page protocol.SessionHistoryResult, bound int64) {
	start := pageStart(page)
	progress := start > 0 || len(page.Events) > 0
	if bound > 0 && !progress {
		// 空页当作没有更早的内容，避免同一游标反复请求。
		branch.hasOlder = false
		return
	}
	if start > 0 && (branch.anchor == 0 || start < branch.anchor) {
		branch.anchor = start
	}
	if page.NextBefore != nil && *page.NextBefore > 0 {
		branch.cursor = *page.NextBefore
	} else {
		branch.cursor = branch.anchor
	}
	// 首屏（最近一轮）页带 turnStart：它的 done 只说明这一轮读完了，更早的轮次
	// 还在下面，只要这一轮的起点不在分支起点就还有更早的内容。向前翻页的页不带
	// turnStart：服务端用 done 说明是否已经翻到分支起点。
	if page.TurnStart != nil && *page.TurnStart > 0 {
		branch.hasOlder = *page.TurnStart > 1
	} else {
		branch.hasOlder = !page.Done && branch.cursor > 1
	}
}

// pageStart 取一页里最早的轮次起点：优先用服务端给的 turnStart，其次用页内最小
// 事件序号。两者都没有时返回 0。
func pageStart(page protocol.SessionHistoryResult) int64 {
	if page.TurnStart != nil && *page.TurnStart > 0 {
		return *page.TurnStart
	}
	var earliest int64
	for _, event := range page.Events {
		if earliest == 0 || event.Sequence < earliest {
			earliest = event.Sequence
		}
	}
	return earliest
}
