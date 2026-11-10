package projection

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// 按轮历史（导出用）的默认值与上限。
const (
	// defaultTurnPageLimit 是每次按轮读取返回的轮次数。
	defaultTurnPageLimit = 100
	// maxTurnPageLimit 是调用方能要求的轮次上限。
	maxTurnPageLimit = 1000
	// turnHistoryServerPageLimit 是发给服务端的单页事实上限。
	turnHistoryServerPageLimit = 500
	// turnHistoryMaxReads 是装载一个分支全部历史最多向服务端要几页。游标不前进、
	// 页反复到达时靠它收敛；超出即报错，不产出半份历史。
	turnHistoryMaxReads = 10000
)

// 按轮历史的错误值。
var (
	// ErrTurnHistoryNoReader 表示没有配置历史读取来源。
	ErrTurnHistoryNoReader = errors.New("按轮历史：缺少历史读取来源")
	// ErrTurnHistoryNoSession 表示没有会话标识。
	ErrTurnHistoryNoSession = errors.New("按轮历史：缺少会话标识")
	// ErrTurnHistorySessionMismatch 表示请求的会话与来源不一致。
	ErrTurnHistorySessionMismatch = errors.New("按轮历史：请求的会话与来源不一致")
	// ErrTurnHistoryBranchMismatch 表示请求的分支与来源不一致。
	ErrTurnHistoryBranchMismatch = errors.New("按轮历史：请求的分支与来源不一致")
	// ErrTurnHistoryTooManyPages 表示历史页数超过上限，来源拒绝给出不完整的结果。
	ErrTurnHistoryTooManyPages = errors.New("按轮历史：历史页数超过上限")
)

// HistoryRequest 是一次按轮历史读取。
//
// Before 是向前翻页游标（排他上界），0 表示从最近一轮开始；Limit 是这一页最多
// 返回几轮，0 用默认值。
type HistoryRequest struct {
	SessionID string
	BranchID  string
	Before    int64
	Limit     int
}

// HistoryTurn 是一轮的只读快照。Data 是本轮的行数据（合法 JSON，格式见
// turnDocument）；目录领域只按轮收集与排序，不解释内容。
type HistoryTurn struct {
	// Index 是轮次在会话内的序号，从 1 开始，按轮次起点排序。
	Index int64
	// Data 是该轮的行数据。
	Data json.RawMessage
}

// HistoryPage 是一页按轮历史。
type HistoryPage struct {
	// Turns 按轮次序号升序。
	Turns []HistoryTurn
	// NextBefore 是下一页游标；0 表示没有更早的轮次。
	NextBefore int64
	// Done 表示没有更早的轮次了。
	Done bool
}

// HistoryReader 读一页原始历史。D1 的 session.Subscription.History 直接满足它。
type HistoryReader interface {
	History(ctx context.Context, params protocol.SessionHistoryParams) (protocol.SessionHistoryResult, error)
}

// turnDocument 是一轮在导出里的落盘形状。行沿用视图的行结构与 JSON 字段名。
type turnDocument struct {
	Index       int64      `json:"index"`
	TurnID      string     `json:"turnId,omitempty"`
	Status      TurnStatus `json:"status"`
	Outcome     string     `json:"outcome,omitempty"`
	StartedAt   int64      `json:"startedAt"`
	EndedAt     int64      `json:"endedAt,omitempty"`
	EndSequence int64      `json:"endSequence,omitempty"`
	Rows        []Row      `json:"rows"`
}

// TurnHistory 是按轮历史来源：按游标读原始历史页，交给会话投影的折叠，再按轮
// 切分输出。它不新建第二套折叠，也不缓存到磁盘。
//
// 一次导出本来就要读全部轮次，所以第一次读取时把该分支的历史读全：轮次序号
// 因此是会话内稳定的 1..N（按轮次起点排序），不会因为翻页顺序而改变。之后的
// 每次读取只是按游标切页。
//
// 一个实例只服务一个会话的一个分支。
type TurnHistory struct {
	mu        sync.Mutex
	sessionID string
	branchID  string
	reader    HistoryReader
	folder    *Folder
	loaded    bool
	rejected  []Rejection
}

// NewTurnHistory 创建按轮历史来源。会话标识用来校验请求，分支由第一次读取决定。
func NewTurnHistory(sessionID string, reader HistoryReader) (*TurnHistory, error) {
	if sessionID == "" {
		return nil, ErrTurnHistoryNoSession
	}
	if reader == nil {
		return nil, ErrTurnHistoryNoReader
	}
	folder, err := New(Config{SessionID: sessionID})
	if err != nil {
		return nil, err
	}
	return &TurnHistory{sessionID: sessionID, reader: reader, folder: folder}, nil
}

// ReadHistory 返回一页按轮历史：读原始历史页 → 现有折叠入口 → 按轮切分。
func (h *TurnHistory) ReadHistory(ctx context.Context, req HistoryRequest) (HistoryPage, error) {
	if req.SessionID != "" && req.SessionID != h.sessionID {
		return HistoryPage{}, ErrTurnHistorySessionMismatch
	}
	branch := req.BranchID
	if branch == "" {
		branch = mainBranch
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultTurnPageLimit
	}
	if limit > maxTurnPageLimit {
		limit = maxTurnPageLimit
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.branchID == "" {
		h.branchID = branch
	} else if branch != h.branchID {
		return HistoryPage{}, ErrTurnHistoryBranchMismatch
	}
	if err := h.loadLocked(ctx); err != nil {
		return HistoryPage{}, err
	}
	return h.pageLocked(req.Before, limit), nil
}

// Rejections 返回按轮读取过程中被丢弃的事实。原因与折叠入口一致。
func (h *TurnHistory) Rejections() []Rejection {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]Rejection(nil), h.rejected...)
}

// loadLocked 把该分支的全部历史读进折叠。只做一次；之后的读取都从折叠里切页。
func (h *TurnHistory) loadLocked(ctx context.Context) error {
	if h.loaded {
		return nil
	}
	cursor := int64(0)
	for reads := 0; ; reads++ {
		if reads >= turnHistoryMaxReads {
			return ErrTurnHistoryTooManyPages
		}
		page, err := h.readPage(ctx, cursor)
		if err != nil {
			return err
		}
		if rejected := h.folder.applyHistoryWindow(h.branchID, page, cursor); len(rejected) > 0 {
			h.rejected = append(h.rejected, rejected...)
		}
		next, ok := historyNext(cursor, page, reads == 0)
		if !ok {
			break
		}
		cursor = next
	}
	h.loaded = true
	return nil
}

// readPage 向服务端要一页原始历史。cursor 为 0 时读最近一轮，否则读它之前的窗口。
func (h *TurnHistory) readPage(ctx context.Context, cursor int64) (protocol.SessionHistoryResult, error) {
	limit := turnHistoryServerPageLimit
	params := protocol.SessionHistoryParams{BranchID: h.branchID, Limit: &limit}
	if cursor > 0 {
		params.Before = &cursor
	} else {
		latest := true
		params.LatestTurn = &latest
	}
	return h.reader.History(ctx, params)
}

// pageLocked 把折叠结果按轮切成一页。
func (h *TurnHistory) pageLocked(before int64, limit int) HistoryPage {
	view, ok := h.folder.BranchView(h.branchID)
	if !ok {
		return HistoryPage{Turns: []HistoryTurn{}, Done: true}
	}
	ordered := orderedTurns(view.Turns)
	exportIndex := make(map[int]int64, len(ordered))
	for position, turn := range ordered {
		exportIndex[turn.Index] = int64(position + 1)
	}
	buckets := bucketRowsByTurn(view.Rows, ordered, exportIndex)

	picked := selectTurns(ordered, before, limit)
	page := HistoryPage{Turns: make([]HistoryTurn, 0, len(picked))}
	for _, turn := range picked {
		document := turnDocument{
			Index:       exportIndex[turn.Index],
			TurnID:      turn.ID,
			Status:      turn.Status,
			Outcome:     turn.Outcome,
			StartedAt:   turn.StartedAt,
			EndedAt:     turn.EndedAt,
			EndSequence: turn.EndSequence,
			Rows:        buckets[exportIndex[turn.Index]],
		}
		if document.Rows == nil {
			document.Rows = []Row{}
		}
		raw, err := json.Marshal(document)
		if err != nil {
			// 行里只有字符串、数字与原始 JSON，编码不会失败；失败也不产出半页。
			return HistoryPage{Turns: []HistoryTurn{}, Done: true}
		}
		page.Turns = append(page.Turns, HistoryTurn{Index: document.Index, Data: raw})
	}
	if len(picked) == 0 {
		page.Done = true
		return page
	}
	oldest := picked[0]
	page.NextBefore = oldest.Sequence
	page.Done = !hasTurnBelow(ordered, oldest.Sequence)
	return page
}

// orderedTurns 按轮次起点把轮次排成会话顺序。折叠里的顺序是"装载顺序"，向前
// 翻页时恰好是倒序，所以这里按起点重新排序，序号才是稳定的 1..N。
func orderedTurns(turns []Turn) []Turn {
	ordered := append([]Turn(nil), turns...)
	sort.SliceStable(ordered, func(left, right int) bool {
		if ordered[left].Sequence != ordered[right].Sequence {
			return ordered[left].Sequence < ordered[right].Sequence
		}
		return ordered[left].Index < ordered[right].Index
	})
	return ordered
}

// selectTurns 取序号小于 before 的轮次（before 为 0 时取全部），只保留最新 limit 轮。
func selectTurns(turns []Turn, before int64, limit int) []Turn {
	picked := make([]Turn, 0, len(turns))
	for _, turn := range turns {
		if before > 0 && turn.Sequence >= before {
			continue
		}
		picked = append(picked, turn)
	}
	if limit > 0 && len(picked) > limit {
		picked = picked[len(picked)-limit:]
	}
	return picked
}

// hasTurnBelow 报告还有没有比 sequence 更早的轮次。
func hasTurnBelow(turns []Turn, sequence int64) bool {
	for _, turn := range turns {
		if turn.Sequence < sequence {
			return true
		}
	}
	return false
}

// bucketRowsByTurn 把行按轮次归桶，键是导出序号。未归属任何轮次的行（例如还没
// 开轮的排队输入）跟它前面最近的一轮走；出现在第一轮之前的行归第一轮。
func bucketRowsByTurn(rows []Row, ordered []Turn, exportIndex map[int]int64) map[int64][]Row {
	buckets := make(map[int64][]Row, len(ordered))
	current := int64(0)
	var leading []Row
	for _, row := range rows {
		if row.Pending {
			// 导出只写服务端已确认的历史；没落库的行不进文件。
			continue
		}
		if row.TurnIndex > 0 {
			mapped, known := exportIndex[row.TurnIndex]
			if !known {
				// 行指向的轮次不在视图里：这一行没有可归属的轮，不导出。
				continue
			}
			current = mapped
			row.TurnIndex = int(current)
			if len(leading) > 0 {
				buckets[current] = append(append([]Row(nil), leading...), buckets[current]...)
				leading = nil
			}
			buckets[current] = append(buckets[current], row)
			continue
		}
		if current > 0 {
			buckets[current] = append(buckets[current], row)
			continue
		}
		leading = append(leading, row)
	}
	if len(leading) > 0 && len(ordered) > 0 {
		first := exportIndex[ordered[0].Index]
		for index := range leading {
			if leading[index].TurnIndex > 0 {
				leading[index].TurnIndex = int(first)
			}
		}
		buckets[first] = append(buckets[first], leading...)
	}
	return buckets
}

// historyNext 从一页历史里取下一个游标。
//
// 首屏窗口页的 done 只表示"这一轮读完了"，更早的轮次还在下面，所以首屏不看
// done，只看游标是否已经到分支起点；之后的向前翻页才按 done 收口。
func historyNext(cursor int64, page protocol.SessionHistoryResult, firstPage bool) (int64, bool) {
	if page.NextBefore == nil {
		return 0, false
	}
	next := *page.NextBefore
	if next <= 1 {
		return 0, false
	}
	if cursor > 0 && next >= cursor {
		return 0, false
	}
	if !firstPage && page.Done {
		return 0, false
	}
	return next, true
}
