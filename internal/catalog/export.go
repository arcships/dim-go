package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/arcships/dimagent-mygo/internal/projection"
)

// 导出文件格式。版本号随格式变化递增；读取方先看格式名与版本号。
const (
	// ExportFormat 是导出文件的格式名。
	ExportFormat = "dimagent.session-export"
	// ExportFormatVersion 是当前导出格式版本。
	ExportFormatVersion = 1
)

// 导出默认值。
const (
	defaultExportPageLimit = 100
	defaultExportMaxPages  = 10000
)

// 导出相关的错误值。
var (
	// ErrExportUnavailable 表示未配置历史读取来源。
	ErrExportUnavailable = errors.New("未配置历史读取来源")
	// ErrRepeatedCursor 表示历史读取的游标重复，继续读会原地打转。
	ErrRepeatedCursor = errors.New("导出游标重复")
	// ErrMalformedTurn 表示历史页里的轮次不符合约定。
	ErrMalformedTurn = errors.New("导出轮次不符合约定")
	// ErrConflictingTurn 表示同一轮次序号出现两份不同内容。
	ErrConflictingTurn = errors.New("同一轮次出现两份不同内容")
	// ErrExportLimit 表示导出的页数超过上限。
	ErrExportLimit = errors.New("导出页数超过上限")
)

// 按轮历史的形状直接采用会话投影（D2）公开的类型。这样 *projection.TurnHistory
// 直接满足 HistorySource，不需要在目录领域再写一层转换，也不会出现第二套折叠。
type (
	// HistoryRequest 是一次按轮历史读取：会话、分支、向前游标与条数上限。
	HistoryRequest = projection.HistoryRequest
	// Turn 是一轮的只读快照；Data 由会话投影给出，本包不解释。
	Turn = projection.HistoryTurn
	// HistoryPage 是一页按轮历史。
	HistoryPage = projection.HistoryPage
)

// HistorySource 按游标读取按轮历史页。实现方是会话投影的按轮来源；本包只按页
// 推进、去重与统计，不折叠内容。
type HistorySource interface {
	ReadHistory(ctx context.Context, req HistoryRequest) (HistoryPage, error)
}

// ExportRequest 是一次导出。
type ExportRequest struct {
	SessionID string
	// BranchID 为空表示主线，由历史读取方决定默认值。
	BranchID string
	// Path 是目标文件路径；目录必须已存在。
	Path string
	// Title 是写进文件头的标题；空时取本地目录里的标题。
	Title string
	// PageLimit 是每页轮次数；0 用默认值。
	PageLimit int
	// MaxPages 是页数上限，防止服务端游标不前进；0 用默认值。
	MaxPages int
}

// ExportResult 是一次成功导出的结果。
type ExportResult struct {
	Path  string
	Turns int
	Pages int
	Bytes int64
}

// exportDocument 是导出文件的落盘结构。
type exportDocument struct {
	Format        string       `json:"format"`
	FormatVersion int          `json:"formatVersion"`
	ExportedAt    time.Time    `json:"exportedAt"`
	SessionID     string       `json:"sessionId"`
	BranchID      string       `json:"branchId,omitempty"`
	Title         string       `json:"title,omitempty"`
	TurnCount     int          `json:"turnCount"`
	Turns         []exportTurn `json:"turns"`
}

// exportTurn 是文件里的一轮。
type exportTurn struct {
	Index int64           `json:"index"`
	Data  json.RawMessage `json:"data"`
}

// Export 按游标读完全部历史后再生成文件。任何一页失败、游标重复、轮次内容冲突
// 都不会产出文件；目标路径上的旧文件保持不变。
func (c *Catalog) Export(ctx context.Context, req ExportRequest) (ExportResult, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return ExportResult{}, fmt.Errorf("%w: 导出缺少会话标识", ErrInvalidRequest)
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return ExportResult{}, fmt.Errorf("%w: 导出缺少文件路径", ErrInvalidRequest)
	}
	if c.cfg.History == nil {
		return ExportResult{}, ErrExportUnavailable
	}
	limit := req.PageLimit
	if limit <= 0 {
		limit = defaultExportPageLimit
	}
	maxPages := req.MaxPages
	if maxPages <= 0 {
		maxPages = defaultExportMaxPages
	}
	title := req.Title
	if title == "" {
		if entry, ok := c.Entry(sessionID); ok {
			title = entry.Title
		}
	}

	turns, pages, err := c.collectTurns(ctx, HistoryRequest{
		SessionID: sessionID,
		BranchID:  req.BranchID,
		Limit:     limit,
	}, maxPages)
	if err != nil {
		return ExportResult{}, err
	}

	document := exportDocument{
		Format:        ExportFormat,
		FormatVersion: ExportFormatVersion,
		ExportedAt:    c.cfg.Now().UTC(),
		SessionID:     sessionID,
		BranchID:      req.BranchID,
		Title:         title,
		TurnCount:     len(turns),
		Turns:         turns,
	}
	raw, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return ExportResult{}, fmt.Errorf("编码导出文件: %w", err)
	}
	raw = append(raw, '\n')
	// 全部读完之后才写文件：中途失败不会留下半成品。
	if err := writeFileAtomic(path, raw); err != nil {
		return ExportResult{}, fmt.Errorf("写入导出文件: %w", err)
	}
	return ExportResult{
		Path:  path,
		Turns: len(turns),
		Pages: pages,
		Bytes: int64(len(raw)),
	}, nil
}

// collectTurns 按游标读完所有页，并按轮次序号去重、排序。
func (c *Catalog) collectTurns(
	ctx context.Context,
	base HistoryRequest,
	maxPages int,
) ([]exportTurn, int, error) {
	seen := make(map[int64]json.RawMessage)
	requested := map[int64]bool{0: true}
	current := base
	pages := 0

	for {
		if pages >= maxPages {
			return nil, pages, ErrExportLimit
		}
		page, err := c.cfg.History.ReadHistory(ctx, current)
		if err != nil {
			return nil, pages, err
		}
		pages++

		for _, turn := range page.Turns {
			if turn.Index <= 0 {
				return nil, pages, fmt.Errorf("%w: 轮次序号 %d", ErrMalformedTurn, turn.Index)
			}
			if !json.Valid(turn.Data) {
				return nil, pages, fmt.Errorf("%w: 轮次 %d 的内容不是合法 JSON", ErrMalformedTurn, turn.Index)
			}
			data := append(json.RawMessage(nil), turn.Data...)
			if existing, ok := seen[turn.Index]; ok {
				if !bytes.Equal(existing, data) {
					return nil, pages, fmt.Errorf("%w: 轮次 %d", ErrConflictingTurn, turn.Index)
				}
				continue
			}
			seen[turn.Index] = data
		}

		if page.Done || page.NextBefore == 0 {
			break
		}
		if requested[page.NextBefore] {
			return nil, pages, fmt.Errorf("%w: %d", ErrRepeatedCursor, page.NextBefore)
		}
		requested[page.NextBefore] = true
		current.Before = page.NextBefore
	}

	indexes := make([]int64, 0, len(seen))
	for index := range seen {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool { return indexes[i] < indexes[j] })

	turns := make([]exportTurn, 0, len(indexes))
	for _, index := range indexes {
		turns = append(turns, exportTurn{Index: index, Data: seen[index]})
	}
	return turns, pages, nil
}
