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

// ForkRequest 是一次会话分叉：从源会话的某个切点复制出一份新会话。
//
// 与分支不同：分支留在原会话内，分叉产生一个新会话，源会话不变。
type ForkRequest struct {
	// SessionID 是源会话。
	SessionID string
	// At 是分叉继承的最后一条主线序号（例如某条用户输入的序号）。
	// 与 TurnIndex 二选一；服务端要求该位置是静止点。
	At int64
	// TurnIndex 是主线轮次序号：给出时用投影的轮结束切点作为 At。
	// 与 At 二选一。
	TurnIndex int
	// Key 是可选的幂等键；空时按源会话与切点序号派生。
	Key string
}

// forkPayload 是分叉意图的参数。
type forkPayload struct {
	Op string `json:"op"`
	At int64  `json:"at"`
}

// forkResult 是分叉回执的结果形状。alpha 服务端只给 sessionId 与 sequence；
// 标题字段是可选的，服务端给出时才采用。
type forkResult struct {
	SessionID     string  `json:"sessionId"`
	Sequence      int64   `json:"sequence"`
	Title         *string `json:"title"`
	TitleSource   string  `json:"titleSource"`
	TitleRevision *int64  `json:"titleRevision"`
}

// ErrForkResultUnknown 表示写登记确认成功但没有可用的回执结果（例如未知后核对
// 命中）：服务端可能已经创建了新会话，但本包拿不到新会话标识，需要重读目录。
var ErrForkResultUnknown = errors.New("分叉结果缺少新会话标识，请重读目录")

// Fork 从源会话的切点分叉出一个新会话，成功后只把新会话并入目录，不重建整个
// 目录；源会话的条目不变。
//
// 切点只接受两种：显式的最后继承序号 At，或某一轮的结束处（TurnIndex，经投影的
// 切点查询）。两者都不给、或同时给，都在登记写意图之前拒绝。
func (c *Catalog) Fork(ctx context.Context, req ForkRequest) (Entry, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return Entry{}, fmt.Errorf("%w: 分叉缺少源会话", ErrInvalidRequest)
	}
	if req.At <= 0 && req.TurnIndex <= 0 {
		return Entry{}, fmt.Errorf("%w: 分叉缺少切点（给 At 或 TurnIndex）", ErrInvalidRequest)
	}
	if req.At > 0 && req.TurnIndex > 0 {
		return Entry{}, fmt.Errorf("%w: 分叉只能给一个切点", ErrInvalidRequest)
	}

	at := req.At
	if req.TurnIndex > 0 {
		if c.cfg.Branches == nil {
			return Entry{}, fmt.Errorf("%w: 分叉按轮次取切点需要切点来源", ErrBranchDirectoryUnavailable)
		}
		point := c.cfg.Branches.TurnCut(sessionID, mainBranchID, req.TurnIndex)
		switch point.Status {
		case CutReady:
			at = point.Sequence
			if at <= 0 {
				return Entry{}, fmt.Errorf(
					"%w: 分叉切点所在轮次没有可用的结束序号", ErrCutNotEnded,
				)
			}
		case CutNotEnded:
			return Entry{}, fmt.Errorf(
				"%w: 第 %d 轮还没结束，等这一轮结束后再分叉",
				ErrCutNotEnded, req.TurnIndex,
			)
		case CutNotLoaded:
			return Entry{}, fmt.Errorf(
				"%w: 第 %d 轮不在已加载范围，请先向前翻页到该轮",
				ErrCutNotLoaded, req.TurnIndex,
			)
		case CutNoBranch:
			return Entry{}, fmt.Errorf("%w: %s", ErrUnknownBranch, mainBranchID)
		default:
			return Entry{}, fmt.Errorf(
				"%w: 切点查询返回未知状态 %q", ErrBranchDirectoryMalformed, point.Status,
			)
		}
	}

	key := strings.TrimSpace(req.Key)
	if key == "" {
		key = forkKey(sessionID, at)
	}
	params, err := json.Marshal(forkPayload{Op: opFork, At: at})
	if err != nil {
		return Entry{}, fmt.Errorf("编码分叉意图: %w", err)
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
	settled, err := c.settle(ctx, OpFork, view)
	if err != nil {
		// 被拒或未结项：目录保持原样，原因已带回。
		return Entry{}, err
	}
	if len(settled.Result) == 0 {
		// 核对命中但没有回执：不知道新会话标识，不猜也不加行。
		return Entry{}, ErrForkResultUnknown
	}

	var result forkResult
	if err := json.Unmarshal(settled.Result, &result); err != nil {
		return Entry{}, fmt.Errorf("%w: 分叉回执无法解析", ErrMalformedResult)
	}
	if strings.TrimSpace(result.SessionID) == "" {
		return Entry{}, fmt.Errorf("%w: 分叉回执缺少会话标识", ErrMalformedResult)
	}
	forked := Entry{ID: result.SessionID}
	if result.Title != nil {
		forked.Title = *result.Title
		forked.TitleSource = TitleSource(result.TitleSource)
		if result.TitleRevision != nil {
			forked.TitleRevision = *result.TitleRevision
		}
	}
	// 回执通常只有标识与序号：按标识做一次单条目录读回，补齐标题与来源；
	// 读回失败不阻塞分叉，保留标识行并标记待补齐。
	if c.cfg.Lookup != nil {
		looked, ok, lookupErr := c.cfg.Lookup.ReadEntry(ctx, forked.ID)
		if lookupErr == nil && ok {
			if strings.TrimSpace(looked.ID) == "" {
				looked.ID = forked.ID
			}
			if looked.ID == forked.ID {
				looked.Incomplete = false
				return c.addEntry(looked), nil
			}
		}
	}
	forked.Incomplete = true
	return c.addEntry(forked), nil
}

// forkKey 按源会话与切点派生幂等键。
func forkKey(sessionID string, at int64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d", sessionID, at)))
	return "catalog.fork:" + hex.EncodeToString(sum[:16])
}
