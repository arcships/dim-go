package projection

import (
	"sort"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// rowState 是折叠出来的行。key 是包内标识：服务端标识或临时键，不出包。
type rowState struct {
	key       string
	serviceID string
	kind      RowKind
	branchID  string
	selfSeq   int64
	lastSeq   int64
	createdAt int64
	updatedAt int64
	status    RowStatus
	text      string
	turnIndex int
	turnID    string
	tool      *ToolCall
	goal      *GoalChange
	deleted   bool
}

// view 把内部行转成只读行。有未落库增量的行按未落库处理：不带标识，
// 落库后原地换服务端标识，行的位置不变。
func (r *rowState) view(loadedThrough int64) Row {
	pending := r.lastSeq > loadedThrough
	row := Row{
		Kind:      r.kind,
		Pending:   pending,
		BranchID:  r.branchID,
		Sequence:  r.selfSeq,
		TurnIndex: r.turnIndex,
		TurnID:    r.turnID,
		CreatedAt: r.createdAt,
		UpdatedAt: r.updatedAt,
		Status:    r.status,
		Text:      r.text,
	}
	if !pending {
		row.ID = r.serviceID
	}
	if r.tool != nil {
		tool := *r.tool
		tool.Input = cloneRaw(tool.Input)
		row.Tool = &tool
	}
	if r.goal != nil {
		change := *r.goal
		row.Goal = &change
	}
	return row
}

// turnState 是折叠出来的轮次。
type turnState struct {
	id          string
	index       int
	sequence    int64
	startedAt   int64
	endedAt     int64
	endSequence int64
	status      TurnStatus
	outcome     string
}

// foldState 是一份分支折叠结果的唯一实现。历史与实时都走 apply。
type foldState struct {
	branchID    string
	maxSequence int64

	rows  []*rowState
	index map[string]*rowState

	turns    []*turnState
	turnByID map[string]*turnState
	current  *turnState

	openText     map[string]*rowState
	openThinking map[string]*rowState
	seenMessage  map[string]struct{}
	tools        map[string]*rowState
	orphanResult map[string][]toolResult

	goal       *goalFold
	inv        *inventory
	usage      usageState
	context    contextState
	compaction compactionState
}

// toolResult 是结果先于调用到达时暂存的一份结果事实。
type toolResult struct {
	sequence int64
	at       int64
	text     string
	isError  bool
}

func newFoldState(branchID string) *foldState {
	return &foldState{
		branchID:     branchID,
		index:        make(map[string]*rowState),
		turnByID:     make(map[string]*turnState),
		openText:     make(map[string]*rowState),
		openThinking: make(map[string]*rowState),
		seenMessage:  make(map[string]struct{}),
		tools:        make(map[string]*rowState),
		orphanResult: make(map[string][]toolResult),
		goal:         newGoalFold(),
		inv:          newInventory(),
	}
}

// foldState 返回当前折叠结果。折叠为空时按"清单 + 按序号事件"重建：
// 乱序、清单替换、断线清理都走同一条重建路径。
func (b *branchState) foldState() *foldState {
	if b.fold != nil {
		return b.fold
	}
	state := newFoldState(b.id)
	b.seeds.applyTo(state)
	sequences := make([]int64, 0, len(b.events))
	for sequence := range b.events {
		sequences = append(sequences, sequence)
	}
	sort.Slice(sequences, func(i, j int) bool { return sequences[i] < sequences[j] })
	for _, sequence := range sequences {
		state.apply(b.events[sequence])
	}
	b.fold = state
	return state
}

// ensureRow 取或建一行。key 相同即同一行：重复事实不会再开一行。
func (s *foldState) ensureRow(key, serviceID string, kind RowKind, event protocol.RawEvent) *rowState {
	if row, ok := s.index[key]; ok {
		return row
	}
	row := &rowState{
		key:       key,
		serviceID: serviceID,
		kind:      kind,
		branchID:  s.branchID,
		selfSeq:   event.Sequence,
		lastSeq:   event.Sequence,
		createdAt: event.CreateAt,
		updatedAt: event.CreateAt,
		status:    StatusComplete,
	}
	s.index[key] = row
	s.rows = append(s.rows, row)
	return row
}

// touch 用一条事实更新行的修改位置与时间。
func (r *rowState) touch(event protocol.RawEvent) {
	r.lastSeq = event.Sequence
	if event.CreateAt > r.updatedAt {
		r.updatedAt = event.CreateAt
	}
}

// ensureTurn 取或建一轮。轮序号按首次出现的顺序，从 1 开始。
func (s *foldState) ensureTurn(id string, event protocol.RawEvent) *turnState {
	if turn, ok := s.turnByID[id]; ok {
		return turn
	}
	turn := &turnState{
		id:        id,
		index:     len(s.turns) + 1,
		sequence:  event.Sequence,
		startedAt: event.CreateAt,
		status:    TurnRunning,
	}
	s.turns = append(s.turns, turn)
	s.turnByID[id] = turn
	return turn
}

// turnFor 解析事实所属轮次：优先用事实自带的轮次标识，其次用当前进行中的一轮。
func (s *foldState) turnFor(data map[string]any, event protocol.RawEvent) (int, string) {
	if id := firstString(data, "turn_id", "turnId"); id != "" {
		turn := s.ensureTurn(id, event)
		return turn.index, turn.id
	}
	if s.current != nil {
		return s.current.index, s.current.id
	}
	return 0, ""
}

// turnViews 输出轮索引。
func (s *foldState) turnViews() []Turn {
	out := make([]Turn, 0, len(s.turns))
	for _, turn := range s.turns {
		out = append(out, Turn{
			Index:       turn.index,
			ID:          turn.id,
			Status:      turn.status,
			Outcome:     turn.outcome,
			Sequence:    turn.sequence,
			StartedAt:   turn.startedAt,
			EndedAt:     turn.endedAt,
			EndSequence: turn.endSequence,
		})
	}
	return out
}
