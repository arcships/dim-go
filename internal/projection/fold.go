package projection

import (
	"encoding/json"
	"errors"
	"sort"
	"sync"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// mainBranch 是主线分支标识，与服务端一致。
const mainBranch = "main"

// Config 是折叠器配置。
type Config struct {
	// SessionID 是服务端给出的会话标识。
	SessionID string
}

// Folder 是一份会话的折叠器：每个分支一份事实集合、一份折叠结果、一份未落库尾部。
type Folder struct {
	mu        sync.Mutex
	sessionID string

	branches map[string]*branchState
	order    []string

	selected    string
	selection   Selection
	loadPhase   LoadPhase
	diagnostics []Diagnostic
}

// branchState 是一个分支的折叠状态。
type branchState struct {
	id     string
	status BranchStatus
	parent string

	branchPoint    int64
	branchPointSeq int64
	endSequence    int64
	loadedThrough  int64
	maxSequence    int64
	anchor         int64
	cursor         int64
	hasOlder       bool
	events         map[int64]protocol.RawEvent
	fold           *foldState
	seeds          seedSet
	preview        map[string]*previewEntry
}

func newBranchState(id string) *branchState {
	return &branchState{
		id:      id,
		status:  BranchOpen,
		events:  make(map[int64]protocol.RawEvent),
		preview: make(map[string]*previewEntry),
	}
}

// New 创建折叠器。迁移表在这里校验：不合法即启动失败。
func New(cfg Config) (*Folder, error) {
	if cfg.SessionID == "" {
		return nil, errors.New("会话投影缺少会话标识")
	}
	if err := validateTransitionTables(); err != nil {
		return nil, err
	}
	folder := &Folder{
		sessionID: cfg.SessionID,
		branches:  make(map[string]*branchState),
		selected:  mainBranch,
		selection: SelectionMain,
		loadPhase: PhaseClosed,
	}
	folder.branches[mainBranch] = newBranchState(mainBranch)
	folder.order = []string{mainBranch}
	return folder, nil
}

// SessionID 返回会话标识。
func (f *Folder) SessionID() string { return f.sessionID }

// ApplyHistory 折叠一页历史。历史页先按序号升序，再逐条交给同一个折叠入口。
//
// Through 是这次读取覆盖到的头部序号：它推进"已落库"的位置，并由视图把尾部行
// 换成正式标识。翻页锚点（向前翻页）由调用方负责，本方法只折叠收到的事实。
func (f *Folder) ApplyHistory(branchID string, page protocol.SessionHistoryResult) []Rejection {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.applyPageLocked(f.branchLocked(branchID), page, 0)
}

// applyPageLocked 折叠一页历史。bound 大于 0 时是这一页的排他上界：序号不小于它
// 的事实不属于这一页（越界），丢弃并带原因，不进视图。
func (f *Folder) applyPageLocked(branch *branchState, page protocol.SessionHistoryResult, bound int64) []Rejection {
	events := append([]protocol.RawEvent(nil), page.Events...)
	sort.SliceStable(events, func(i, j int) bool { return events[i].Sequence < events[j].Sequence })

	var rejections []Rejection
	for _, event := range events {
		if bound > 0 && event.Sequence >= bound {
			rejections = append(rejections, Rejection{
				BranchID: branch.id,
				Sequence: event.Sequence,
				Type:     event.Type,
				Name:     event.Name,
				Reason:   ReasonOutOfWindow,
				Detail:   "事实不在这次读取的窗口内",
			})
			continue
		}
		rejections = append(rejections, f.applyFactLocked(branch, event)...)
	}
	if page.Through > branch.loadedThrough {
		branch.loadedThrough = page.Through
	}
	if page.Through > branch.maxSequence {
		branch.maxSequence = page.Through
	}
	return rejections
}

// ApplyNotice 折叠一条实时通知。只认 `session.event` 与 `session.model.progress`；
// 其余通知归交互裁决与管理面，本包不看也不改。
func (f *Folder) ApplyNotice(method string, params json.RawMessage) []Rejection {
	switch method {
	case "session.event":
		var notice protocol.SessionEventParams
		if err := json.Unmarshal(params, &notice); err != nil {
			return []Rejection{{Reason: ReasonMalformedNotice, Detail: "session.event 参数无法解码"}}
		}
		if notice.SessionID != "" && notice.SessionID != f.sessionID {
			return []Rejection{{
				BranchID: notice.BranchID,
				Sequence: notice.Event.Sequence,
				Type:     notice.Event.Type,
				Name:     notice.Event.Name,
				Reason:   ReasonSessionMismatch,
				Detail:   "通知属于别的会话",
			}}
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.applyFactLocked(f.branchLocked(notice.BranchID), notice.Event)

	case "session.model.progress":
		var notice progressNotice
		if err := json.Unmarshal(params, &notice); err != nil {
			return []Rejection{{Reason: ReasonMalformedNotice, Detail: "session.model.progress 参数无法解码"}}
		}
		if notice.SessionID != "" && notice.SessionID != f.sessionID {
			return []Rejection{{Reason: ReasonSessionMismatch, Detail: "通知属于别的会话"}}
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.branchLocked(notice.BranchID).applyProgress(notice)
		return nil

	default:
		return nil
	}
}

// applyHistoryWindow 折叠一页历史并声明它的排他上界，供按轮读取使用。
// 它复用同一个折叠入口，只是不改分页位置：按轮读取自己管游标。
func (f *Folder) applyHistoryWindow(branchID string, page protocol.SessionHistoryResult, bound int64) []Rejection {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.applyPageLocked(f.branchLocked(branchID), page, bound)
}

// branchLocked 取分支；未知分支按事实建桶（分支创建事实可能晚于它自己的实时事实）。
func (f *Folder) branchLocked(branchID string) *branchState {
	if branchID == "" {
		branchID = mainBranch
	}
	if branch, ok := f.branches[branchID]; ok {
		return branch
	}
	branch := newBranchState(branchID)
	f.branches[branchID] = branch
	f.order = append(f.order, branchID)
	return branch
}

// applyFactLocked 是历史与实时共同的落点：校验、去重、分支目录、折叠。
func (f *Folder) applyFactLocked(branch *branchState, event protocol.RawEvent) []Rejection {
	rejection := func(reason, detail string) []Rejection {
		return []Rejection{{
			BranchID: branch.id,
			Sequence: event.Sequence,
			Type:     event.Type,
			Name:     event.Name,
			Reason:   reason,
			Detail:   detail,
		}}
	}

	switch {
	case event.ID == "":
		return rejection(ReasonInvalidEvent, "缺少事件标识")
	case event.Sequence <= 0:
		return rejection(ReasonInvalidEvent, "缺少序号")
	case event.Type == "" || event.Name == "":
		return rejection(ReasonInvalidEvent, "缺少类型或名字")
	}
	if branch.status == BranchEnded && branch.endSequence > 0 && event.Sequence > branch.endSequence {
		if branch.id == f.selected {
			f.recordSelectionLocked(EndedBranchFactEvent)
		}
		return rejection(ReasonBranchEnded, "分支已结束，不再接收新事实")
	}
	if _, duplicate := branch.events[event.Sequence]; duplicate {
		// attach 重放与重读都会带来同一序号：同一事实只折叠一次。
		return nil
	}
	branch.events[event.Sequence] = event
	if event.Sequence > branch.maxSequence {
		branch.maxSequence = event.Sequence
	}
	f.observeDirectoryLocked(branch, event)
	f.foldLocked(branch, event)
	return nil
}

// foldLocked 把事实交给折叠。乱序（序号低于已折叠到的位置）会让已折叠结果失效，
// 下次读视图时按序号整体重建，因此乱序与重复都不会产生重复行。
func (f *Folder) foldLocked(branch *branchState, event protocol.RawEvent) {
	if branch.fold == nil {
		// 重建会包含刚存进来的这条事实。
		return
	}
	if event.Sequence < branch.fold.maxSequence {
		branch.fold = nil
		return
	}
	branch.fold.apply(event)
}

// observeDirectoryLocked 记录分支目录事实。目录只记录事实带来的分支身份与状态，
// 不由客户端推导出更深的结构。
func (f *Folder) observeDirectoryLocked(branch *branchState, event protocol.RawEvent) {
	data := decodeObject(event.Data)
	switch event.Type + "/" + event.Name {
	case "branch/create":
		target := firstString(data, "branch", "branchId")
		if target == "" || target == branch.id {
			return
		}
		child := f.branchLocked(target)
		if child.parent == "" {
			child.parent = branch.id
		}
		if at := object(data["at"])["sequence"]; at != nil && child.branchPoint == 0 {
			child.branchPoint, child.branchPointSeq = toInt64(at), event.Sequence
		}
	case "branch/start":
		if branch.id == mainBranch {
			return
		}
		if parent := firstString(data, "parent", "parentBranchId"); parent != "" && branch.parent == "" {
			branch.parent = parent
		}
		if point := firstInt64(data, "createEvent", "create_event"); point != 0 && branch.branchPoint == 0 {
			branch.branchPoint, branch.branchPointSeq = point, event.Sequence
		}
	case "branch/end", "branch/ended":
		if branch.id == mainBranch || branch.status == BranchEnded {
			return
		}
		branch.status = BranchEnded
		branch.endSequence = event.Sequence
	case "branch/resume":
		// 第一版不重开已结束分支：恢复后的事实按新分支处理，不在本包内推导。
	}
}

// SeedPendingInputs 用服务端在固定截止点读到的排队输入替换本地清单。
// 清单只覆盖到 through：已有事实里序号不超过 through 的结算不要再改它。
func (f *Folder) SeedPendingInputs(branchID string, through int64, items []PendingInput) {
	f.seed(branchID, func(set *seedSet) {
		set.inputs = seeded[PendingInput]{through: through, items: append([]PendingInput(nil), items...)}
	})
}

// SeedPendingQuestions 用服务端在固定截止点读到的待答提问替换本地清单。
func (f *Folder) SeedPendingQuestions(branchID string, through int64, items []PendingQuestion) {
	f.seed(branchID, func(set *seedSet) {
		set.questions = seeded[PendingQuestion]{through: through, items: append([]PendingQuestion(nil), items...)}
	})
}

// SeedPendingTasks 用服务端在固定截止点读到的未结束后台任务替换本地清单。
func (f *Folder) SeedPendingTasks(branchID string, through int64, items []PendingTask) {
	f.seed(branchID, func(set *seedSet) {
		set.tasks = seeded[PendingTask]{through: through, items: append([]PendingTask(nil), items...)}
	})
}

// SeedPendingCollaborations 用服务端在固定截止点读到的协作请求替换本地清单。
func (f *Folder) SeedPendingCollaborations(branchID string, through int64, items []PendingCollaboration) {
	f.seed(branchID, func(set *seedSet) {
		set.collabs = seeded[PendingCollaboration]{through: through, items: append([]PendingCollaboration(nil), items...)}
	})
}

func (f *Folder) seed(branchID string, apply func(*seedSet)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	branch := f.branchLocked(branchID)
	apply(&branch.seeds)
	// 清单是折叠的输入之一：换清单即重建，保证装载与事件结算只有一处实现。
	branch.fold = nil
}

// OnDisconnect 清空未落库尾部：只保留已被历史覆盖的事实，清空实时预览。
// 之后重读历史会把它们重新折叠进来，因此不会叠加出第二份。
func (f *Folder) OnDisconnect(branchID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	branch := f.branchLocked(branchID)
	f.dropTailLocked(branch)
}

func (f *Folder) dropTailLocked(branch *branchState) {
	for sequence := range branch.events {
		if sequence > branch.loadedThrough {
			delete(branch.events, sequence)
		}
	}
	branch.fold = nil
	branch.preview = make(map[string]*previewEntry)
	branch.maxSequence = branch.loadedThrough
	for sequence := range branch.events {
		if sequence > branch.maxSequence {
			branch.maxSequence = sequence
		}
	}
}

// Head 返回分支头部序号：历史覆盖到的位置与实时事实里较大的那个。
func (f *Folder) Head(branchID string) int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	branch, ok := f.branches[branchID]
	if !ok {
		return 0
	}
	return branch.maxSequence
}

// Selected 返回当前选中的分支。
func (f *Folder) Selected() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.selected
}

// Selection 返回分支选择状态。
func (f *Folder) Selection() Selection {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.selection
}

// SelectBranch 切换查看的分支。未知分支被拒绝；已结束的分支可以只读查看
// （它不再接收新事实，见 applyFactLocked）。从分支 A 直接切到分支 B 走
// "切回主线"+"选某个分支"两步，不新增表外迁移。
func (f *Folder) SelectBranch(branchID string) error {
	if branchID == "" {
		branchID = mainBranch
	}
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.branches[branchID]; !ok {
		f.recordDiagnosticLocked(Diagnostic{
			Machine: machineSelection,
			From:    string(f.selection),
			Event:   string(SelectBranchEvent),
			Reason:  ReasonUnknownBranch,
		})
		return errors.New("未知分支: " + branchID)
	}
	if branchID == f.selected {
		return nil
	}
	if f.selection == SelectionBranch && branchID != mainBranch {
		f.recordSelectionLocked(BackToMainEvent)
		f.selected = mainBranch
	}
	if branchID == mainBranch {
		f.recordSelectionLocked(BackToMainEvent)
		f.selected = mainBranch
		return nil
	}
	f.recordSelectionLocked(SelectBranchEvent)
	f.selected = branchID
	return nil
}

// recordSelectionLocked 驱动分支选择状态机；非法迁移只记诊断，状态不变。
func (f *Folder) recordSelectionLocked(event SelectionEvent) {
	to, ok := lookupTransition(selectionTable, string(f.selection), string(event))
	if !ok {
		f.recordDiagnosticLocked(Diagnostic{
			Machine: machineSelection,
			From:    string(f.selection),
			Event:   string(event),
			Reason:  "非法迁移",
		})
		return
	}
	f.selection = Selection(to)
}

// AdvanceLoad 驱动"打开与装载"状态机。非法迁移被丢弃并记一条诊断，返回的
// Transition.OK 为假；调用方决定怎么报。
func (f *Folder) AdvanceLoad(event LoadEvent) Transition {
	f.mu.Lock()
	defer f.mu.Unlock()

	result := Transition{Machine: machineLoad, From: string(f.loadPhase), Event: string(event)}
	to, ok := lookupTransition(loadTable, string(f.loadPhase), string(event))
	if !ok {
		result.Reason = "非法迁移"
		f.recordDiagnosticLocked(Diagnostic{
			Machine: machineLoad,
			From:    string(f.loadPhase),
			Event:   string(event),
			Reason:  "非法迁移",
		})
		return result
	}
	result.OK = true
	result.To = to
	if event == LoadConnectionLost {
		// 断线即清空未落库尾部；重读后不再叠加。分页位置回到"未装载"，
		// 重开时按 A 文档重设锚点重读。
		for _, branch := range f.branches {
			f.dropTailLocked(branch)
			branch.anchor = 0
			branch.cursor = 0
			branch.hasOlder = false
		}
	}
	if event == LoadSessionClosed {
		for _, branch := range f.branches {
			f.dropTailLocked(branch)
		}
	}
	f.loadPhase = LoadPhase(to)
	return result
}

// LoadPhase 返回装载阶段。
func (f *Folder) LoadPhase() LoadPhase {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.loadPhase
}

// Diagnostics 返回一路累积的诊断。
func (f *Folder) Diagnostics() []Diagnostic {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Diagnostic(nil), f.diagnostics...)
}

func (f *Folder) recordDiagnosticLocked(diagnostic Diagnostic) {
	f.diagnostics = append(f.diagnostics, diagnostic)
}

// Branches 返回分支目录，按首次出现的顺序。
func (f *Folder) Branches() []BranchInfo {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]BranchInfo, 0, len(f.order))
	for _, id := range f.order {
		branch, ok := f.branches[id]
		if !ok {
			continue
		}
		out = append(out, BranchInfo{
			BranchID:    branch.id,
			ParentID:    branch.parent,
			BranchPoint: branch.branchPoint,
			Status:      branch.status,
			Head:        branch.maxSequence,
		})
	}
	return out
}

// View 返回当前选中分支的只读视图。
func (f *Folder) View() View {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.viewLocked(f.branchLocked(f.selected))
}

// BranchView 返回指定分支的只读视图。
func (f *Folder) BranchView(branchID string) (View, bool) {
	if branchID == "" {
		branchID = mainBranch
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	branch, ok := f.branches[branchID]
	if !ok {
		return View{}, false
	}
	return f.viewLocked(branch), true
}

func (f *Folder) viewLocked(branch *branchState) View {
	state := branch.foldState()
	view := View{
		SessionID:    f.sessionID,
		BranchID:     branch.id,
		BranchStatus: branch.status,
		Head:         branch.maxSequence,
		LoadPhase:    f.loadPhase,
		Selection:    f.selection,
		Goal:         state.goal.snapshot(),
		Pending:      state.inv.snapshot(),
		Anchor:       branch.anchor,
		HasOlder:     branch.hasOlder,
	}
	for _, row := range state.rows {
		if row.deleted {
			continue
		}
		rowView := row.view(branch.loadedThrough)
		if rowView.Pending {
			view.Unlanded++
		}
		view.Rows = append(view.Rows, rowView)
	}
	for _, row := range state.previewRows(branch) {
		view.Unlanded++
		view.Rows = append(view.Rows, row)
	}
	view.Turns = state.turnViews()
	return view
}
