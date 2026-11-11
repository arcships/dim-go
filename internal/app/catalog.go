package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/projection"
)

// CreateSession 新建会话，并显式带上权限档位。
//
// 默认策略下工具调用可能需要审批；自检与自动化路径应显式给档位（例如
// full-access），否则工具调用会停在等待审批。空档位表示按服务端默认，不猜。
func (a *App) CreateSession(ctx context.Context, cwd, permissionPreset string) (catalog.Entry, error) {
	domain := a.Catalog()
	if domain == nil {
		return catalog.Entry{}, ErrNotConnected
	}
	return domain.Create(ctx, catalog.CreateRequest{
		Cwd:              cwd,
		PermissionPreset: permissionPreset,
	})
}

// catalogReader 是目录领域的读取方：把 session.catalog 的一页映射成目录条目。
type catalogReader struct {
	app *App
}

// catalogWireSession 是目录回执里的一条会话。字段沿用 wire 的 camelCase。
type catalogWireSession struct {
	SessionID     string  `json:"sessionId"`
	Cwd           string  `json:"cwd"`
	CreatedAt     int64   `json:"createdAt"`
	UpdatedAt     int64   `json:"updatedAt"`
	Running       bool    `json:"running"`
	Title         *string `json:"title"`
	TitleSource   *string `json:"titleSource"`
	TitleRevision *int64  `json:"titleRevision"`
}

// ReadCatalog 实现 catalog.Reader。
func (r *catalogReader) ReadCatalog(
	ctx context.Context,
	query catalog.Query,
	cursor string,
) (catalog.Page, error) {
	client := r.app.control()
	if client == nil {
		return catalog.Page{}, failure.New(failure.KindConnectionMissing, "读取会话目录", ErrNotConnected)
	}
	params := map[string]any{"limit": query.Limit}
	if cursor != "" {
		params["cursor"] = cursor
	}
	if query.Search != "" {
		params["query"] = query.Search
	}
	if query.Workspace != "" {
		params["cwd"] = query.Workspace
	}
	raw, err := client.Call(ctx, "session.catalog", params)
	if err != nil {
		return catalog.Page{}, err
	}
	var result struct {
		Sessions   []catalogWireSession `json:"sessions"`
		NextCursor *string              `json:"nextCursor"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return catalog.Page{}, fmt.Errorf("目录回执无法解析: %w", err)
	}
	page := catalog.Page{Entries: make([]catalog.Entry, 0, len(result.Sessions))}
	for _, session := range result.Sessions {
		entry := catalog.Entry{
			ID:        session.SessionID,
			Workspace: session.Cwd,
			Running:   session.Running,
		}
		if session.CreatedAt > 0 {
			entry.CreatedAt = time.UnixMilli(session.CreatedAt).UTC()
		}
		if session.UpdatedAt > 0 {
			entry.UpdatedAt = time.UnixMilli(session.UpdatedAt).UTC()
		}
		if session.Title != nil {
			entry.Title = *session.Title
		}
		if session.TitleSource != nil {
			entry.TitleSource = catalog.TitleSource(*session.TitleSource)
		}
		if session.TitleRevision != nil {
			entry.TitleRevision = *session.TitleRevision
		}
		page.Entries = append(page.Entries, entry)
	}
	if result.NextCursor != nil {
		page.NextCursor = *result.NextCursor
	}
	return page, nil
}

// sessionReleaser 释放一个会话的全部订阅持有者；删除前必须先释放。
type sessionReleaser struct {
	app *App
}

// ReleaseSession 实现 catalog.SubscriptionReleaser。会话本来没打开时视为已释放。
func (r *sessionReleaser) ReleaseSession(sessionID string) error {
	return r.app.CloseSession(sessionID)
}

// turnHistorySource 用已打开会话的订阅做导出用的按轮历史来源。
type turnHistorySource struct {
	app *App
}

// ReadHistory 实现 catalog.HistorySource：按游标读历史，交给投影的按轮来源切分。
func (s *turnHistorySource) ReadHistory(
	ctx context.Context,
	request catalog.HistoryRequest,
) (catalog.HistoryPage, error) {
	runtime, ok := s.app.session(request.SessionID)
	if !ok {
		return catalog.HistoryPage{}, fmt.Errorf("%w: 导出前先打开该会话", ErrSessionNotOpen)
	}
	if runtime.lease == nil {
		return catalog.HistoryPage{}, fmt.Errorf("%w: 会话没有可用的历史订阅", ErrSessionNotOpen)
	}
	source, err := projection.NewTurnHistory(request.SessionID, runtime.lease.Subscription())
	if err != nil {
		return catalog.HistoryPage{}, fmt.Errorf("创建按轮历史来源: %w", err)
	}
	return source.ReadHistory(ctx, request)
}
