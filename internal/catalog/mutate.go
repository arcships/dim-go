package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// 目录变更在写意图里的操作名。集成层的写发送方按这些操作名映射到服务端的
// 具体方法；本包不出现方法名与 wire 形状。三类变更都用写登记的
// KindSessionLifecycle 承载：新建时还没有会话标识，改名与删除必须带目标会话。
const (
	opCreate = "catalog.create"
	opRename = "catalog.rename"
	opDelete = "catalog.delete"
)

// 幂等键前缀。
const (
	keyPrefixCreate = "catalog.create"
	keyPrefixRename = "catalog.rename"
	keyPrefixDelete = "catalog.delete"
)

// Op 是一次目录变更的类型。
type Op string

const (
	// OpCreate 是新建会话。
	OpCreate Op = "create"
	// OpRename 是改名。
	OpRename Op = "rename"
	// OpDelete 是删除。
	OpDelete Op = "delete"
	// OpBranchCreate 是从某轮切点创建分支。
	OpBranchCreate Op = "branch_create"
	// OpBranchEnd 是结束分支。
	OpBranchEnd Op = "branch_end"
)

// TitleUpdate 是一条标题事实：自动标题通知或改名结果的提交值。按修订号合并，
// 修订号不大于当前值的更新被忽略。
type TitleUpdate struct {
	SessionID string
	Title     string
	Source    TitleSource
	Revision  int64
}

// CreateRequest 是一次新建。Cwd 必填。
type CreateRequest struct {
	Cwd string
	// Key 是可选的幂等键。空时为这次调用生成唯一键：两次新建是两次独立意图。
	// 调用方重试同一次新建时应显式给同一个键。
	Key string
}

// RenameRequest 是一次改名。
type RenameRequest struct {
	SessionID string
	// Title 是新标题；空字符串表示清除标题，由集成层按服务端语义下发。
	Title string
	// Key 可选；空时按会话与标题派生，用于去重完全相同的改名。
	Key string
}

// DeleteRequest 是一次删除。
type DeleteRequest struct {
	SessionID string
	// Key 可选；空时按会话标识派生，用于去重完全相同的删除。
	Key string
}

// createPayload 是新建意图的参数。集成层把它映射成服务端的创建请求。
type createPayload struct {
	Op  string `json:"op"`
	Cwd string `json:"cwd"`
}

// renamePayload 是改名意图的参数。会话标识由写登记的 SessionID 承载。
type renamePayload struct {
	Op    string `json:"op"`
	Title string `json:"title"`
}

// deletePayload 是删除意图的参数。
type deletePayload struct {
	Op string `json:"op"`
}

// createResult 是新建回执的结果形状。集成层把服务端回执原文放进
// write.Reply.Result 即可。
type createResult struct {
	SessionID string `json:"sessionId"`
}

// renameResult 是改名回执的结果形状。Title 为 null 表示标题被清除。
type renameResult struct {
	Title       *string `json:"title"`
	TitleSource string  `json:"titleSource"`
	Revision    int64   `json:"revision"`
}

// 错误值；调用方按 errors.Is 判断，不匹配错误文本。
var (
	// ErrInvalidRequest 表示请求字段不合法。
	ErrInvalidRequest = errors.New("目录请求不合法")
	// ErrUnknownSession 表示该会话不在本地目录集合里。
	ErrUnknownSession = errors.New("会话不在目录集合里")
	// ErrMalformedPage 表示服务端返回的目录页不符合约定。
	ErrMalformedPage = errors.New("目录页不符合约定")
	// ErrMalformedResult 表示变更回执没有给出可用的结果。
	ErrMalformedResult = errors.New("目录变更回执不符合约定")
	// ErrRejected 表示服务端明确拒绝了这次变更或结果需人工确认；本地行保持不变。
	ErrRejected = errors.New("目录变更被拒绝")
	// ErrTitleConflict 表示改名回执的标题修订号不高于本地已合并的修订号。
	ErrTitleConflict = errors.New("标题修订号冲突")
	// ErrArchiveUnavailable 表示未配置本地归档隐藏列表。
	ErrArchiveUnavailable = errors.New("未配置本地归档隐藏列表")
)

// RejectedError 是一次没有被确认成功的目录变更。Reason 来自写登记：可能是服务端
// 拒绝的原因，也可能是结果未知且需要人工确认。
type RejectedError struct {
	Op     Op
	Status write.Status
	Reason string
}

func (e *RejectedError) Error() string {
	reason := e.Reason
	if reason == "" {
		reason = "原因未给出"
	}
	return fmt.Sprintf("目录%s被拒绝（%s）：%s", opLabel(e.Op), e.Status, reason)
}

// Unwrap 让调用方用 errors.Is(err, ErrRejected) 分类。
func (e *RejectedError) Unwrap() error { return ErrRejected }

// PendingError 表示目录变更在等待期间没有结项（上下文结束或写登记关闭）。
// 服务端结果未知，本地集合保持不变。
type PendingError struct {
	Op     Op
	Status write.Status
	Reason string
}

func (e *PendingError) Error() string {
	reason := e.Reason
	if reason == "" {
		reason = "结果未定"
	}
	return fmt.Sprintf("目录%s结果未定（%s）：%s", opLabel(e.Op), e.Status, reason)
}

// TitleConflictError 表示改名回执的标题修订号不高于本地已合并的修订号：服务端
// 可能已经提交了更新的标题。本地标题保持不变，调用方应重读目录。
type TitleConflictError struct {
	SessionID     string
	LocalRevision int64
	ReplyRevision int64
}

func (e *TitleConflictError) Error() string {
	return fmt.Sprintf(
		"会话 %s 的标题修订号冲突：本地 %d，回执 %d；请重读目录",
		e.SessionID, e.LocalRevision, e.ReplyRevision,
	)
}

// Unwrap 让调用方用 errors.Is(err, ErrTitleConflict) 分类。
func (e *TitleConflictError) Unwrap() error { return ErrTitleConflict }

func opLabel(op Op) string {
	switch op {
	case OpCreate:
		return "新建"
	case OpRename:
		return "改名"
	case OpDelete:
		return "删除"
	case OpBranchCreate:
		return "创建分支"
	case OpBranchEnd:
		return "结束分支"
	}
	return string(op)
}

// Create 新建一个会话。成功后返回的条目只带服务端确认的会话标识；标题、工作目录
// 与时间在下一次目录读取时补齐，客户端不替服务端编造。
func (c *Catalog) Create(ctx context.Context, req CreateRequest) (Entry, error) {
	cwd := strings.TrimSpace(req.Cwd)
	if cwd == "" {
		return Entry{}, fmt.Errorf("%w: 新建缺少工作目录", ErrInvalidRequest)
	}
	key := strings.TrimSpace(req.Key)
	if key == "" {
		id, err := c.cfg.NewID()
		if err != nil {
			return Entry{}, fmt.Errorf("生成新建幂等键: %w", err)
		}
		key = keyPrefixCreate + ":" + id
	}
	params, err := json.Marshal(createPayload{Op: opCreate, Cwd: cwd})
	if err != nil {
		return Entry{}, fmt.Errorf("编码新建意图: %w", err)
	}

	view, err := c.cfg.Writes.Register(write.Intent{
		Key:    key,
		Kind:   write.KindSessionLifecycle,
		Params: params,
	})
	if err != nil {
		return Entry{}, err
	}
	settled, err := c.settle(ctx, OpCreate, view)
	if err != nil {
		return Entry{}, err
	}

	var result createResult
	if len(settled.Result) == 0 {
		return Entry{}, fmt.Errorf("%w: 新建回执没有结果", ErrMalformedResult)
	}
	if err := json.Unmarshal(settled.Result, &result); err != nil {
		return Entry{}, fmt.Errorf("%w: 新建回执无法解析", ErrMalformedResult)
	}
	if strings.TrimSpace(result.SessionID) == "" {
		return Entry{}, fmt.Errorf("%w: 新建回执缺少会话标识", ErrMalformedResult)
	}

	entry := Entry{ID: result.SessionID}
	c.mu.Lock()
	if idx, exists := c.index[entry.ID]; exists {
		// 目录里已有这一条时以目录事实为准，不用新建回执覆盖。
		entry = c.entries[idx]
	} else {
		c.index[entry.ID] = len(c.entries)
		c.entries = append(c.entries, entry)
	}
	c.mu.Unlock()
	return entry, nil
}

// Rename 通过写登记改名。成功后按修订号合并标题；回执修订号不高于本地时返回
// TitleConflictError 且不改动本地标题，调用方应重读目录。
func (c *Catalog) Rename(ctx context.Context, req RenameRequest) (Entry, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return Entry{}, fmt.Errorf("%w: 改名缺少会话标识", ErrInvalidRequest)
	}
	if _, ok := c.Entry(sessionID); !ok {
		return Entry{}, fmt.Errorf("%w: %s", ErrUnknownSession, sessionID)
	}

	key := strings.TrimSpace(req.Key)
	if key == "" {
		key = renameKey(sessionID, req.Title)
	}
	params, err := json.Marshal(renamePayload{Op: opRename, Title: req.Title})
	if err != nil {
		return Entry{}, fmt.Errorf("编码改名意图: %w", err)
	}

	view, err := c.cfg.Writes.Register(write.Intent{
		Key:       key,
		Kind:      write.KindSessionLifecycle,
		SessionID: sessionID,
		Params:    params,
	})
	if err != nil {
		return Entry{}, err
	}
	settled, err := c.settle(ctx, OpRename, view)
	if err != nil {
		return Entry{}, err
	}

	var result renameResult
	if len(settled.Result) == 0 {
		return Entry{}, fmt.Errorf("%w: 改名回执没有结果", ErrMalformedResult)
	}
	if err := json.Unmarshal(settled.Result, &result); err != nil {
		return Entry{}, fmt.Errorf("%w: 改名回执无法解析", ErrMalformedResult)
	}
	if result.Revision < 0 {
		return Entry{}, fmt.Errorf("%w: 改名回执修订号为负", ErrMalformedResult)
	}
	title := ""
	source := TitleSource("")
	if result.Title != nil {
		title = *result.Title
		source = TitleSource(result.TitleSource)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	idx, ok := c.index[sessionID]
	if !ok {
		return Entry{}, fmt.Errorf("%w: %s", ErrUnknownSession, sessionID)
	}
	current := c.entries[idx]
	switch {
	case result.Revision < current.TitleRevision:
		return current, &TitleConflictError{
			SessionID:     sessionID,
			LocalRevision: current.TitleRevision,
			ReplyRevision: result.Revision,
		}
	case result.Revision == current.TitleRevision:
		if current.Title == title && current.TitleSource == source {
			// 同一修订号、同一内容：视为重放，不产生变化。
			return current, nil
		}
		return current, &TitleConflictError{
			SessionID:     sessionID,
			LocalRevision: current.TitleRevision,
			ReplyRevision: result.Revision,
		}
	}
	current.Title = title
	current.TitleSource = source
	current.TitleRevision = result.Revision
	c.entries[idx] = current
	return current, nil
}

// Delete 删除一个会话。顺序固定为：先释放订阅，再登记删除意图；只有写登记确认
// 成功后才从本地集合移除该行，被拒或结果未定时保留并给出原因。
func (c *Catalog) Delete(ctx context.Context, req DeleteRequest) error {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return fmt.Errorf("%w: 删除缺少会话标识", ErrInvalidRequest)
	}
	if _, ok := c.Entry(sessionID); !ok {
		return fmt.Errorf("%w: %s", ErrUnknownSession, sessionID)
	}
	if c.cfg.Releaser != nil {
		if err := c.cfg.Releaser.ReleaseSession(sessionID); err != nil {
			return fmt.Errorf("释放会话订阅失败，未发出删除意图: %w", err)
		}
	}

	key := strings.TrimSpace(req.Key)
	if key == "" {
		key = deleteKey(sessionID)
	}
	params, err := json.Marshal(deletePayload{Op: opDelete})
	if err != nil {
		return fmt.Errorf("编码删除意图: %w", err)
	}
	view, err := c.cfg.Writes.Register(write.Intent{
		Key:       key,
		Kind:      write.KindSessionLifecycle,
		SessionID: sessionID,
		Params:    params,
	})
	if err != nil {
		return err
	}
	if _, err := c.settle(ctx, OpDelete, view); err != nil {
		// 被拒或结果未定：本地行保留，原因已带回。
		return err
	}

	c.removeEntry(sessionID)
	return nil
}

// settle 等待写意图进入终态，并把非成功的结果翻译成目录侧的错误。
func (c *Catalog) settle(
	ctx context.Context,
	op Op,
	view write.IntentView,
) (write.IntentView, error) {
	settled, waitErr := c.cfg.Writes.Wait(ctx, view.ID)
	switch settled.Status {
	case write.StatusSucceeded:
		return settled, nil
	case write.StatusFailed, write.StatusNotOccurred, write.StatusNeedsManualConfirmation:
		return settled, &RejectedError{Op: op, Status: settled.Status, Reason: settled.Reason}
	default:
		reason := settled.Reason
		if reason == "" && waitErr != nil {
			reason = waitErr.Error()
		}
		return settled, &PendingError{Op: op, Status: settled.Status, Reason: reason}
	}
}

// removeEntry 从本地集合移除一条条目；删除成功后的唯一入口。
func (c *Catalog) removeEntry(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	idx, ok := c.index[sessionID]
	if !ok {
		return
	}
	c.entries = append(c.entries[:idx], c.entries[idx+1:]...)
	delete(c.index, sessionID)
	for i := idx; i < len(c.entries); i++ {
		c.index[c.entries[i].ID] = i
	}
}

// renameKey 按会话与标题派生幂等键；不把标题原文写进键里。
func renameKey(sessionID, title string) string {
	sum := sha256.Sum256([]byte(sessionID + "\x00" + title))
	return keyPrefixRename + ":" + hex.EncodeToString(sum[:16])
}

// deleteKey 按会话标识派生幂等键。
func deleteKey(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return keyPrefixDelete + ":" + hex.EncodeToString(sum[:16])
}
