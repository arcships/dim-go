package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// 分支目录的状态。取值与投影一致：服务端可能给出 open / ending / ended。
type BranchStatus string

const (
	// BranchOpen 表示分支还在接收事实。
	BranchOpen BranchStatus = "open"
	// BranchEnding 表示结束请求已受理，等待服务端收敛为已结束。
	BranchEnding BranchStatus = "ending"
	// BranchEnded 表示分支已结束：头部固定，不再接收提交与新分支。
	BranchEnded BranchStatus = "ended"
)

// 分支目录里的默认分支标识。
const mainBranchID = "main"

// Branch 是分支目录的一项。CutSequence 是切点：父分支上某一轮结束处的序号。
type Branch struct {
	ID          string
	ParentID    string
	CutSequence int64
	Status      BranchStatus
	// Head 是该分支已见的最大序号。已结束分支的 Head 固定，不被后续更新抬高。
	Head int64
}

// BranchDirectory 是一个会话的分支目录。
type BranchDirectory struct {
	SessionID string
	Branches  []Branch
	// Complete 为 false 表示服务端没有提供目录，内容按投影的事实推导，可能缺项。
	Complete bool
	// Loaded 为 false 表示还没有读取过该会话的分支目录。
	Loaded bool
}

// BranchSource 提供会话投影（D2）已经推出的分支事实。目录领域只保存、合并与
// 校验，不自己折叠事实，也不另建一套目录推导。
type BranchSource interface {
	// ReadBranches 返回分支目录。complete 为 false 表示服务端未提供目录，
	// 内容按事实推导。
	ReadBranches(ctx context.Context, sessionID string) (branches []Branch, complete bool, err error)
	// TurnCut 返回某一轮结束处的父分支序号，并说明为什么不可用：
	// ready / not_ended / not_loaded / no_branch。本包按原因拒绝，不猜切点。
	TurnCut(sessionID, branchID string, turnIndex int) CutPoint
}

// 切点查询的状态与结果直接采用会话投影（D2）的公开类型，保证集成层一行转发。
type (
	// CutStatus 是切点查询的状态。
	CutStatus = projection.CutStatus
	// CutPoint 是一次切点查询的结果。
	CutPoint = projection.CutPoint
)

const (
	// CutReady 表示该轮已结束，切点可用。
	CutReady = projection.CutReady
	// CutNotEnded 表示该轮还没有结束事实，切点不可用。
	CutNotEnded = projection.CutNotEnded
	// CutNotLoaded 表示该轮不在已加载范围内，切点不可用。
	CutNotLoaded = projection.CutNotLoaded
	// CutNoBranch 表示分支不在折叠里（未知分支或不是本会话）。
	CutNoBranch = projection.CutNoBranch
)

// BranchCreateRequest 是一次从某轮切点创建分支的请求。
type BranchCreateRequest struct {
	SessionID string
	// ParentBranchID 是父分支；空按主线处理。
	ParentBranchID string
	// TurnIndex 是父分支上的轮次序号（从 1 开始）；切点取该轮结束处。
	TurnIndex int
	// Prompt 是分支的起始提问；服务端要求非空。
	Prompt string
	// Key 是可选的幂等键；空时按会话、父分支、切点轮次与提示词派生。
	Key string
}

// BranchEndRequest 是一次结束分支的请求。
type BranchEndRequest struct {
	SessionID string
	BranchID  string
	// Key 可选；空时按会话与分支派生。
	Key string
}

// 分支变更在写意图里的操作名。
const (
	opBranchCreate = "branch.create"
	opBranchEnd    = "branch.end"
)

// branchCreatePayload 是分支创建意图的参数。
type branchCreatePayload struct {
	Op             string `json:"op"`
	ParentBranchID string `json:"parentBranchId"`
	At             int64  `json:"at"`
	Prompt         string `json:"prompt"`
}

// branchEndPayload 是结束分支意图的参数。
type branchEndPayload struct {
	Op       string `json:"op"`
	BranchID string `json:"branchId"`
}

// branchCreateResult 是分支创建回执的结果形状。
type branchCreateResult struct {
	Branch *branchCreateInfo `json:"branch"`
}

// branchCreateInfo 是服务端给出的新分支。
type branchCreateInfo struct {
	BranchID       string `json:"branchId"`
	ParentBranchID string `json:"parentBranchId"`
	BranchPoint    *int64 `json:"branchPoint"`
	CreateEvent    *int64 `json:"createEvent"`
}

// 分支相关的错误值。
var (
	// ErrBranchDirectoryUnavailable 表示未配置分支目录来源。
	ErrBranchDirectoryUnavailable = errors.New("未配置分支目录来源")
	// ErrBranchDirectoryMalformed 表示分支目录不符合约定。
	ErrBranchDirectoryMalformed = errors.New("分支目录不符合约定")
	// ErrCutNotLoaded 表示切点轮次不在已加载范围内，应先向前翻页。
	ErrCutNotLoaded = errors.New("分支切点不在已加载范围内")
	// ErrCutNotEnded 表示切点轮次还没有结束，不能在这一轮开分支。
	ErrCutNotEnded = errors.New("分支切点所在轮次尚未结束")
	// ErrBranchEnded 表示分支已结束，不再接收提交或新分支。
	ErrBranchEnded = errors.New("分支已结束")
	// ErrUnknownBranch 表示分支不在目录里。
	ErrUnknownBranch = errors.New("分支不在目录里")
)

// branchDirectoryState 是一个会话的分支目录内部状态。
type branchDirectoryState struct {
	loaded   bool
	complete bool
	branches []Branch
	index    map[string]int
}

// RefreshBranches 通过会话投影读取一个会话的分支目录，并整体替换该会话的目录。
// 读取失败时保留原目录并把错误带回。
func (c *Catalog) RefreshBranches(ctx context.Context, sessionID string) (BranchDirectory, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return BranchDirectory{}, fmt.Errorf("%w: 读取分支目录缺少会话标识", ErrInvalidRequest)
	}
	if c.cfg.Branches == nil {
		return c.Branches(sessionID), ErrBranchDirectoryUnavailable
	}

	c.branchLoadMu.Lock()
	defer c.branchLoadMu.Unlock()

	branches, complete, err := c.cfg.Branches.ReadBranches(ctx, sessionID)
	if err != nil {
		// 读取失败不改本地目录。
		return c.Branches(sessionID), err
	}
	entries, index, err := sanitizeBranches(branches)
	if err != nil {
		return c.Branches(sessionID), err
	}

	c.mu.Lock()
	c.branches[sessionID] = &branchDirectoryState{
		loaded:   true,
		complete: complete,
		branches: entries,
		index:    index,
	}
	c.mu.Unlock()
	return c.Branches(sessionID), nil
}

// Branches 返回一个会话的分支目录副本；没有读取过时 Loaded 为 false。
func (c *Catalog) Branches(sessionID string) BranchDirectory {
	c.mu.Lock()
	defer c.mu.Unlock()
	state, ok := c.branches[sessionID]
	if !ok {
		return BranchDirectory{SessionID: sessionID}
	}
	return BranchDirectory{
		SessionID: sessionID,
		Branches:  append([]Branch(nil), state.branches...),
		Complete:  state.complete,
		Loaded:    state.loaded,
	}
}

// ApplyBranchUpdate 把一条分支事实并入目录：只改这一条，不重建整个目录。
// 状态只前进（open → ending → ended）；已结束分支的 Head 固定不变。
func (c *Catalog) ApplyBranchUpdate(sessionID string, update Branch) (Branch, error) {
	if strings.TrimSpace(sessionID) == "" {
		return Branch{}, fmt.Errorf("%w: 分支更新缺少会话标识", ErrInvalidRequest)
	}
	if strings.TrimSpace(update.ID) == "" {
		return Branch{}, fmt.Errorf("%w: 分支缺少标识", ErrInvalidRequest)
	}
	if !update.Status.valid() {
		return Branch{}, fmt.Errorf("%w: 未知分支状态 %q", ErrInvalidRequest, update.Status)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.branchStateLocked(sessionID)
	if !state.loaded {
		// 只有一条事实时目录不完整，不能冒充服务端完整目录。
		state.loaded = true
		state.complete = false
	}
	return c.mergeIntoStateLocked(state, update), nil
}

// CheckBranchWritable 供提交路径咨询：已结束分支不再接收新提交。
// 目录还没读取过时不拦截（客户端没有可判断的事实）。
func (c *Catalog) CheckBranchWritable(sessionID, branchID string) error {
	branchID = strings.TrimSpace(branchID)
	if branchID == "" || branchID == mainBranchID {
		return nil
	}
	directory := c.Branches(sessionID)
	if !directory.Loaded {
		return nil
	}
	for _, branch := range directory.Branches {
		if branch.ID != branchID {
			continue
		}
		if branch.Status == BranchEnded {
			return fmt.Errorf("%w: %s", ErrBranchEnded, branchID)
		}
		return nil
	}
	return fmt.Errorf("%w: %s", ErrUnknownBranch, branchID)
}

// CreateBranch 从父分支某一轮的结束处创建分支。切点不在已加载范围内、父分支已
// 结束或参数不合法时，在登记写意图之前拒绝。
func (c *Catalog) CreateBranch(ctx context.Context, req BranchCreateRequest) (Branch, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return Branch{}, fmt.Errorf("%w: 创建分支缺少会话标识", ErrInvalidRequest)
	}
	parentID := strings.TrimSpace(req.ParentBranchID)
	if parentID == "" {
		parentID = mainBranchID
	}
	if req.TurnIndex <= 0 {
		return Branch{}, fmt.Errorf("%w: 切点轮次必须为正", ErrInvalidRequest)
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return Branch{}, fmt.Errorf("%w: 创建分支缺少提示词", ErrInvalidRequest)
	}
	if c.cfg.Branches == nil {
		return Branch{}, ErrBranchDirectoryUnavailable
	}

	// 已结束分支不再开新分支；目录没读取过时不拦，交给服务端。
	directory := c.Branches(sessionID)
	if directory.Loaded {
		found := false
		for _, branch := range directory.Branches {
			if branch.ID != parentID {
				continue
			}
			found = true
			if branch.Status == BranchEnded {
				return Branch{}, fmt.Errorf("%w: 父分支 %s 不能再开分支", ErrBranchEnded, parentID)
			}
		}
		if !found && parentID != mainBranchID {
			return Branch{}, fmt.Errorf("%w: %s", ErrUnknownBranch, parentID)
		}
	}

	point := c.cfg.Branches.TurnCut(sessionID, parentID, req.TurnIndex)
	var cut int64
	switch point.Status {
	case CutReady:
		cut = point.Sequence
		if cut <= 0 {
			return Branch{}, fmt.Errorf(
				"%w: 父分支 %s 的第 %d 轮没有可用的结束序号",
				ErrCutNotEnded, parentID, req.TurnIndex,
			)
		}
	case CutNotEnded:
		return Branch{}, fmt.Errorf(
			"%w: 父分支 %s 的第 %d 轮还没结束，等这一轮结束后再开分支",
			ErrCutNotEnded, parentID, req.TurnIndex,
		)
	case CutNotLoaded:
		return Branch{}, fmt.Errorf(
			"%w: 父分支 %s 的第 %d 轮不在已加载范围，请先向前翻页到该轮",
			ErrCutNotLoaded, parentID, req.TurnIndex,
		)
	case CutNoBranch:
		return Branch{}, fmt.Errorf("%w: %s", ErrUnknownBranch, parentID)
	default:
		return Branch{}, fmt.Errorf(
			"%w: 切点查询返回未知状态 %q",
			ErrBranchDirectoryMalformed, point.Status,
		)
	}

	key := strings.TrimSpace(req.Key)
	if key == "" {
		key = branchCreateKey(sessionID, parentID, req.TurnIndex, prompt)
	}
	params, err := json.Marshal(branchCreatePayload{
		Op:             opBranchCreate,
		ParentBranchID: parentID,
		At:             cut,
		Prompt:         prompt,
	})
	if err != nil {
		return Branch{}, fmt.Errorf("编码分支创建意图: %w", err)
	}
	view, err := c.cfg.Writes.Register(write.Intent{
		Key:       key,
		Kind:      write.KindSessionLifecycle,
		SessionID: sessionID,
		Params:    params,
	})
	if err != nil {
		return Branch{}, err
	}
	settled, err := c.settle(ctx, OpBranchCreate, view)
	if err != nil {
		// 被拒或未结项：目录保持原样，原因已带回。
		return Branch{}, err
	}

	var result branchCreateResult
	if len(settled.Result) == 0 {
		return Branch{}, fmt.Errorf("%w: 分支创建回执没有结果", ErrMalformedResult)
	}
	if err := json.Unmarshal(settled.Result, &result); err != nil {
		return Branch{}, fmt.Errorf("%w: 分支创建回执无法解析", ErrMalformedResult)
	}
	if result.Branch == nil || strings.TrimSpace(result.Branch.BranchID) == "" {
		return Branch{}, fmt.Errorf("%w: 分支创建回执缺少分支标识", ErrMalformedResult)
	}
	created := Branch{
		ID:          result.Branch.BranchID,
		ParentID:    firstNonEmpty(result.Branch.ParentBranchID, parentID),
		CutSequence: cut,
		Status:      BranchOpen,
	}
	if result.Branch.BranchPoint != nil && *result.Branch.BranchPoint > 0 {
		created.CutSequence = *result.Branch.BranchPoint
	}
	// 只把新分支并入目录，不重建其他条目。
	return c.ApplyBranchUpdate(sessionID, created)
}

// EndBranch 结束一个非主线分支。成功后本地状态先收敛为 ending，等待目录读取或
// 事实更新给出 ended 与固定头；被拒时目录保持原样。
func (c *Catalog) EndBranch(ctx context.Context, req BranchEndRequest) (Branch, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return Branch{}, fmt.Errorf("%w: 结束分支缺少会话标识", ErrInvalidRequest)
	}
	branchID := strings.TrimSpace(req.BranchID)
	if branchID == "" {
		return Branch{}, fmt.Errorf("%w: 结束分支缺少分支标识", ErrInvalidRequest)
	}
	if branchID == mainBranchID {
		return Branch{}, fmt.Errorf("%w: 主线不能结束", ErrInvalidRequest)
	}

	// 已结束的分支不再发请求：状态已经收敛。
	directory := c.Branches(sessionID)
	if directory.Loaded {
		for _, branch := range directory.Branches {
			if branch.ID != branchID {
				continue
			}
			if branch.Status == BranchEnded {
				return branch, nil
			}
		}
	}

	key := strings.TrimSpace(req.Key)
	if key == "" {
		key = branchEndKey(sessionID, branchID)
	}
	params, err := json.Marshal(branchEndPayload{Op: opBranchEnd, BranchID: branchID})
	if err != nil {
		return Branch{}, fmt.Errorf("编码结束分支意图: %w", err)
	}
	view, err := c.cfg.Writes.Register(write.Intent{
		Key:       key,
		Kind:      write.KindSessionLifecycle,
		SessionID: sessionID,
		BranchID:  branchID,
		Params:    params,
	})
	if err != nil {
		return Branch{}, err
	}
	if _, err := c.settle(ctx, OpBranchEnd, view); err != nil {
		return Branch{}, err
	}
	return c.ApplyBranchUpdate(sessionID, Branch{ID: branchID, Status: BranchEnding})
}

// branchStateLocked 取一个会话的目录状态，没有时创建。调用方持有 c.mu。
func (c *Catalog) branchStateLocked(sessionID string) *branchDirectoryState {
	state, ok := c.branches[sessionID]
	if !ok {
		state = &branchDirectoryState{index: make(map[string]int)}
		c.branches[sessionID] = state
	}
	return state
}

// mergeBranch 把一条分支事实并入目录并返回合并后的条目。
func mergeBranch(branches []Branch, index map[string]int, update Branch) Branch {
	position, exists := index[update.ID]
	if !exists {
		index[update.ID] = len(branches)
		return update
	}
	existing := branches[position]
	merged := update
	if merged.ParentID == "" {
		merged.ParentID = existing.ParentID
	}
	if merged.CutSequence == 0 {
		merged.CutSequence = existing.CutSequence
	}
	// 状态只前进：open → ending → ended。
	if branchStatusRank(merged.Status) < branchStatusRank(existing.Status) {
		merged.Status = existing.Status
	}
	if merged.Head < existing.Head {
		merged.Head = existing.Head
	}
	// 已结束分支的头部固定。
	if existing.Status == BranchEnded {
		merged.Status = BranchEnded
		merged.Head = existing.Head
	}
	return merged
}

// mergeIntoStateLocked 合并一条分支事实并写回目录。
func (c *Catalog) mergeIntoStateLocked(state *branchDirectoryState, update Branch) Branch {
	merged := mergeBranch(state.branches, state.index, update)
	position := state.index[merged.ID]
	if position < len(state.branches) && state.branches[position].ID == merged.ID {
		state.branches[position] = merged
	} else {
		state.branches = append(state.branches, merged)
	}
	return merged
}

func sanitizeBranches(branches []Branch) ([]Branch, map[string]int, error) {
	entries := make([]Branch, 0, len(branches))
	index := make(map[string]int, len(branches))
	for _, branch := range branches {
		if strings.TrimSpace(branch.ID) == "" {
			return nil, nil, fmt.Errorf("%w: 分支缺少标识", ErrBranchDirectoryMalformed)
		}
		if !branch.Status.valid() {
			return nil, nil, fmt.Errorf("%w: 分支 %s 状态未知 %q", ErrBranchDirectoryMalformed, branch.ID, branch.Status)
		}
		if branch.Head < 0 || branch.CutSequence < 0 {
			return nil, nil, fmt.Errorf("%w: 分支 %s 的序号为负", ErrBranchDirectoryMalformed, branch.ID)
		}
		if _, duplicate := index[branch.ID]; duplicate {
			continue
		}
		index[branch.ID] = len(entries)
		entries = append(entries, branch)
	}
	return entries, index, nil
}

func (s BranchStatus) valid() bool {
	switch s {
	case BranchOpen, BranchEnding, BranchEnded:
		return true
	}
	return false
}

func branchStatusRank(status BranchStatus) int {
	switch status {
	case BranchOpen:
		return 1
	case BranchEnding:
		return 2
	case BranchEnded:
		return 3
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// branchCreateKey 按会话、父分支、切点轮次与提示词派生幂等键；不把提示词原文
// 写进键里。
func branchCreateKey(sessionID, parentID string, turnIndex int, prompt string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d\x00%s", sessionID, parentID, turnIndex, prompt)))
	return "catalog.branch_create:" + hex.EncodeToString(sum[:16])
}

// branchEndKey 按会话与分支派生幂等键。
func branchEndKey(sessionID, branchID string) string {
	sum := sha256.Sum256([]byte(sessionID + "\x00" + branchID))
	return "catalog.branch_end:" + hex.EncodeToString(sum[:16])
}
