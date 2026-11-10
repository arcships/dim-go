package projection_test

// 这个文件只做一件事：证明本包给出的按轮历史形状与目录领域（D4）导出的
// HistorySource 完全对得上，而且不需要本包反向导入 internal/catalog。
//
// 目录领域把三个类型改成指向本包的别名即可，HistorySource 直接成立：
//
//	type HistoryRequest = projection.HistoryRequest
//	type HistoryPage    = projection.HistoryPage
//	type Turn           = projection.HistoryTurn
//
// 下面的 mirror* 类型逐字抄自 internal/catalog/export.go 的字段定义（不导入该包）。
// 两边字段能互相赋值、访问方式一致，就说明别名替换后编译通过。

import (
	"context"
	"encoding/json"

	"github.com/arcships/dimagent-mygo/internal/projection"
)

// mirrorHistoryRequest 抄自 catalog.HistoryRequest。
type mirrorHistoryRequest struct {
	SessionID string
	BranchID  string
	Before    int64
	Limit     int
}

// mirrorTurn 抄自 catalog.Turn。
type mirrorTurn struct {
	Index int64
	Data  json.RawMessage
}

// mirrorHistoryPage 抄自 catalog.HistoryPage。
type mirrorHistoryPage struct {
	Turns      []mirrorTurn
	NextBefore int64
	Done       bool
}

// 字段级双向赋值：任何一处名字或类型漂移都编译不过。
func shapeMatches(
	request projection.HistoryRequest,
	turn projection.HistoryTurn,
	page projection.HistoryPage,
) {
	mirrorRequest := mirrorHistoryRequest{
		SessionID: request.SessionID,
		BranchID:  request.BranchID,
		Before:    request.Before,
		Limit:     request.Limit,
	}
	_ = projection.HistoryRequest{
		SessionID: mirrorRequest.SessionID,
		BranchID:  mirrorRequest.BranchID,
		Before:    mirrorRequest.Before,
		Limit:     mirrorRequest.Limit,
	}

	mirror := mirrorTurn{Index: turn.Index, Data: turn.Data}
	_ = projection.HistoryTurn{Index: mirror.Index, Data: mirror.Data}

	mirrorPage := mirrorHistoryPage{NextBefore: page.NextBefore, Done: page.Done}
	_ = projection.HistoryPage{NextBefore: mirrorPage.NextBefore, Done: mirrorPage.Done}
}

// 目录领域把类型换成别名之后，HistorySource 由 *TurnHistory 直接满足。
type aliasedRequest = projection.HistoryRequest
type aliasedPage = projection.HistoryPage

type aliasedHistorySource interface {
	ReadHistory(ctx context.Context, req aliasedRequest) (aliasedPage, error)
}

var _ aliasedHistorySource = (*projection.TurnHistory)(nil)

// 目录领域分支来源接口里的切点方法，抄自 internal/catalog/branch.go：
//
//	TurnCut(sessionID, branchID string, turnIndex int) (cut int64, ok bool)
//
// 本包的同名方法与之同形，集成层一行转发即可。
type turnCutSource interface {
	TurnCut(sessionID, branchID string, turnIndex int) (int64, bool)
}

var _ turnCutSource = (*projection.Folder)(nil)
