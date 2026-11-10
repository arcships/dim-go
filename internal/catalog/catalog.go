// Package catalog 是会话目录（D4）：目录读取与过滤分页、新建、改名与自动标题、
// 删除、本地归档隐藏列表与导出。
//
// 边界：
//   - 目录读取只按服务端返回的页与游标推进，本地集合是这些页按到达顺序的合并；
//     不在本地实现过滤、排序或索引，过滤参数原样交给读取方。
//   - 新建、改名、删除都登记成写意图，由写登记（D3）发送一次并收敛；本包不自行
//     发请求、不自行重试。
//   - 删除只有写登记确认成功后才从本地集合移除该行；被拒或结果未定时保留并给出
//     原因。
//   - 归档隐藏列表是本地状态：原子写、带版本、损坏重建为空，只在用户显式操作时变，
//     从不写回服务端。
//   - 导出按游标读完全部历史后再生成文件；中途失败不产出文件，文件带格式版本号。
//
// 不包含：界面、连接实现、协议 wire 定义、会话投影的折叠内部、审批语义。
package catalog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// 默认值与上限。
const (
	// DefaultPageSize 是未指定页大小时的单页条数。
	DefaultPageSize = 50
	// maxCatalogPages 是一次目录加载最多推进的页数，防止游标不前进时无限循环。
	maxCatalogPages = 10000
)

// TitleSource 是标题来源。名称与服务端一致，但本包不解释其业务含义。
type TitleSource string

const (
	// TitleSourceAuto 表示由服务端自动生成的标题。
	TitleSourceAuto TitleSource = "auto"
	// TitleSourceManual 表示用户改过的标题。
	TitleSourceManual TitleSource = "manual"
)

// Entry 是目录里的一条会话。字段只承载目录可见的事实，不复制会话内容。
//
// 新建成功后、下一次目录读取之前，本地只会有 ID：标题、工作目录与时间都等
// 服务端页补齐，客户端不替服务端编造。
type Entry struct {
	// ID 是服务端给出的会话标识，目录里的身份。
	ID string
	// Title 是当前标题；空表示服务端还没有标题。
	Title string
	// TitleSource 是标题来源；空表示未知。
	TitleSource TitleSource
	// TitleRevision 是标题修订号。标题通知与改名结果都按它合并，较小的不覆盖较大的。
	TitleRevision int64
	// Workspace 是会话的工作目录。
	Workspace string
	// CreatedAt 与 UpdatedAt 是服务端给出的时间。
	CreatedAt time.Time
	UpdatedAt time.Time
	// Running 是服务端给出的运行状态。
	Running bool
}

// Query 是一次目录读取的业务口径。所有过滤参数原样交给读取方，本包不在本地
// 重建过滤语义；游标不在这里，它由服务端返回并由本包保存。
type Query struct {
	// Search 是关键字过滤；空表示不过滤。
	Search string
	// Workspace 是工作目录过滤；空表示不过滤。
	Workspace string
	// Limit 是单页条数；0 表示使用 DefaultPageSize。
	Limit int
}

// Page 是服务端返回的一页目录。NextCursor 为空表示没有更多。
type Page struct {
	Entries    []Entry
	NextCursor string
}

// Reader 读取目录的一页。cursor 为空表示第一页；实现方只按服务端事实返回，
// 不在本地过滤或重排。它只读，不写。
type Reader interface {
	ReadCatalog(ctx context.Context, query Query, cursor string) (Page, error)
}

// SubscriptionReleaser 释放一个会话的全部订阅持有者。删除必须先释放订阅，
// 再由写登记发出删除意图；释放失败时不发删除意图。
type SubscriptionReleaser interface {
	ReleaseSession(sessionID string) error
}

// LoadResult 是一次分页推进的结果。
type LoadResult struct {
	// Added 是本次新加入本地集合的条目数。
	Added int
	// Duplicates 是因会话标识已存在而跳过的条目数（含页内重复）。
	Duplicates int
	// HasMore 表示是否还有下一页。
	HasMore bool
	// Stopped 表示因为游标重复或页数到上限而停止继续推进。
	Stopped bool
}

// Snapshot 是目录的只读视图。已归档隐藏的条目不出现，只给出数量。
type Snapshot struct {
	Entries []Entry
	Hidden  int
	HasMore bool
	Query   Query
}

// Config 是目录配置。Reader 与 Writes 必填；Archive、Releaser、History 按能力使用。
type Config struct {
	// Reader 读取服务端目录页。
	Reader Reader
	// Writes 是写登记；新建、改名、删除都经它登记。
	Writes *write.Manager
	// Archive 是本地归档隐藏列表；nil 表示不提供归档能力。
	Archive *Archive
	// Releaser 释放会话订阅；nil 表示本进程没有持有该会话的订阅。
	Releaser SubscriptionReleaser
	// History 按游标读取历史；导出使用，nil 表示不提供导出能力。
	History HistorySource
	// Branches 提供分支目录与轮次切点；nil 表示不提供分支能力。
	Branches BranchSource
	// Now 与 NewID 可在测试中替换；nil 时使用系统实现。
	Now   func() time.Time
	NewID func() (string, error)
}

func (c Config) withDefaults() Config {
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.NewID == nil {
		c.NewID = randomID
	}
	return c
}

// Catalog 持有目录页合并后的本地集合。所有导出方法可并发调用。
type Catalog struct {
	cfg Config

	// loadMu 串行化整次分页推进，保证本地集合按页顺序合并。
	loadMu sync.Mutex
	// branchLoadMu 串行化分支目录的整次读取，避免两次读取交错覆盖。
	branchLoadMu sync.Mutex

	mu          sync.Mutex
	branches    map[string]*branchDirectoryState
	entries     []Entry
	index       map[string]int
	query       Query
	cursor      string
	hasMore     bool
	pages       int
	seenCursors map[string]bool
}

// New 创建目录。Reader 与 Writes 必填；归档隐藏列表与导出来源可选。
func New(cfg Config) (*Catalog, error) {
	if cfg.Reader == nil {
		return nil, errors.New("目录缺少读取接口")
	}
	if cfg.Writes == nil {
		return nil, errors.New("目录缺少写登记")
	}
	cfg = cfg.withDefaults()
	return &Catalog{
		cfg:         cfg,
		index:       make(map[string]int),
		seenCursors: make(map[string]bool),
		branches:    make(map[string]*branchDirectoryState),
	}, nil
}

// Refresh 按给定过滤条件从第一页重取目录，并用这一页重建本地集合。
//
// 过滤条件变化、或页面损坏后恢复时使用。更早的页由 LoadMore 重新推进；
// 重建不会保留服务端已经不再返回的旧行。
func (c *Catalog) Refresh(ctx context.Context, query Query) (LoadResult, error) {
	query = c.normalizeQuery(query)

	c.loadMu.Lock()
	defer c.loadMu.Unlock()

	page, err := c.cfg.Reader.ReadCatalog(ctx, query, "")
	if err != nil {
		return LoadResult{}, err
	}
	incoming, duplicates, err := sanitizePage(page)
	if err != nil {
		return LoadResult{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 页面是存在与否的唯一依据；标题按修订号合并，避免本地已合并的更新标题
	// 被更旧的页面回退。
	entries := make([]Entry, 0, len(incoming))
	index := make(map[string]int, len(incoming))
	for _, entry := range incoming {
		if previous, ok := c.entryLocked(entry.ID); ok {
			entry = mergeTitle(previous, entry)
		}
		index[entry.ID] = len(entries)
		entries = append(entries, entry)
	}
	c.entries = entries
	c.index = index
	c.query = query
	c.cursor = page.NextCursor
	c.hasMore = page.NextCursor != ""
	c.pages = 1
	c.seenCursors = map[string]bool{"": true}

	return LoadResult{
		Added:      len(entries),
		Duplicates: duplicates,
		HasMore:    c.hasMore,
	}, nil
}

// LoadMore 用服务端给的下一个游标推进一页，并把页内新条目追加到本地集合。
// 已存在的会话标识不会重复加入；游标重复或页数到上限时停止推进。
func (c *Catalog) LoadMore(ctx context.Context) (LoadResult, error) {
	c.loadMu.Lock()
	defer c.loadMu.Unlock()

	c.mu.Lock()
	if !c.hasMore || c.cursor == "" {
		result := LoadResult{HasMore: false}
		c.mu.Unlock()
		return result, nil
	}
	cursor := c.cursor
	query := c.query
	if c.seenCursors[cursor] {
		c.hasMore = false
		c.cursor = ""
		c.mu.Unlock()
		return LoadResult{HasMore: false, Stopped: true}, nil
	}
	if c.pages >= maxCatalogPages {
		c.hasMore = false
		c.cursor = ""
		c.mu.Unlock()
		return LoadResult{HasMore: false, Stopped: true}, nil
	}
	c.mu.Unlock()

	page, err := c.cfg.Reader.ReadCatalog(ctx, query, cursor)
	if err != nil {
		// 读取失败不推进游标，重试仍是同一页。
		return LoadResult{}, err
	}
	incoming, duplicates, err := sanitizePage(page)
	if err != nil {
		return LoadResult{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.seenCursors[cursor] = true
	c.pages++

	added := 0
	for _, entry := range incoming {
		if _, exists := c.index[entry.ID]; exists {
			duplicates++
			continue
		}
		c.index[entry.ID] = len(c.entries)
		c.entries = append(c.entries, entry)
		added++
	}

	stopped := false
	switch {
	case page.NextCursor == "":
		c.hasMore = false
		c.cursor = ""
	case c.seenCursors[page.NextCursor]:
		// 服务端把游标指回已经请求过的位置：视为重复页，停止推进。
		c.hasMore = false
		c.cursor = ""
		stopped = true
	default:
		c.cursor = page.NextCursor
		c.hasMore = true
	}

	return LoadResult{
		Added:      added,
		Duplicates: duplicates,
		HasMore:    c.hasMore,
		Stopped:    stopped,
	}, nil
}

// Snapshot 返回目录的只读视图；已归档隐藏的条目只计数、不出现在列表里。
func (c *Catalog) Snapshot() Snapshot {
	c.mu.Lock()
	entries := append([]Entry(nil), c.entries...)
	query := c.query
	hasMore := c.hasMore
	c.mu.Unlock()

	visible := make([]Entry, 0, len(entries))
	hidden := 0
	for _, entry := range entries {
		if c.hidden(entry.ID) {
			hidden++
			continue
		}
		visible = append(visible, entry)
	}
	return Snapshot{Entries: visible, Hidden: hidden, HasMore: hasMore, Query: query}
}

// HiddenEntries 返回被本地归档隐藏的条目，顺序与服务端页一致，供「已归档」列表使用。
func (c *Catalog) HiddenEntries() []Entry {
	c.mu.Lock()
	entries := append([]Entry(nil), c.entries...)
	c.mu.Unlock()

	hidden := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if c.hidden(entry.ID) {
			hidden = append(hidden, entry)
		}
	}
	return hidden
}

// Entry 返回一条条目的只读副本。
func (c *Catalog) Entry(sessionID string) (Entry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.entryLocked(sessionID)
}

// ApplyTitle 合并一条标题事实（自动标题通知或改名结果），按修订号取大。
// 修订号不大于当前值的更新被忽略；返回是否真的改动了本地条目。
//
// 目录集合只由服务端页推进：标题事实不会凭空新增一条条目。
func (c *Catalog) ApplyTitle(update TitleUpdate) bool {
	if update.SessionID == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	idx, ok := c.index[update.SessionID]
	if !ok {
		return false
	}
	entry := c.entries[idx]
	if update.Revision <= entry.TitleRevision {
		return false
	}
	entry.Title = update.Title
	entry.TitleSource = update.Source
	entry.TitleRevision = update.Revision
	c.entries[idx] = entry
	return true
}

// Hide 把一个会话加入本地归档隐藏列表。它只改本地状态，不影响服务端事实。
func (c *Catalog) Hide(sessionID string) error {
	if c.cfg.Archive == nil {
		return ErrArchiveUnavailable
	}
	return c.cfg.Archive.Hide(sessionID)
}

// Unhide 把一个会话移出本地归档隐藏列表。
func (c *Catalog) Unhide(sessionID string) error {
	if c.cfg.Archive == nil {
		return ErrArchiveUnavailable
	}
	return c.cfg.Archive.Unhide(sessionID)
}

// Hidden 报告一个会话是否被本地归档隐藏。
func (c *Catalog) Hidden(sessionID string) bool {
	return c.hidden(sessionID)
}

func (c *Catalog) hidden(sessionID string) bool {
	if c.cfg.Archive == nil || sessionID == "" {
		return false
	}
	return c.cfg.Archive.Hidden(sessionID)
}

func (c *Catalog) entryLocked(sessionID string) (Entry, bool) {
	idx, ok := c.index[sessionID]
	if !ok {
		return Entry{}, false
	}
	return c.entries[idx], true
}

func (c *Catalog) normalizeQuery(query Query) Query {
	if query.Limit <= 0 {
		query.Limit = DefaultPageSize
	}
	return query
}

// sanitizePage 校验一页条目并去掉页内重复的会话标识。
func sanitizePage(page Page) ([]Entry, int, error) {
	entries := make([]Entry, 0, len(page.Entries))
	seen := make(map[string]bool, len(page.Entries))
	duplicates := 0
	for _, entry := range page.Entries {
		if entry.ID == "" {
			return nil, 0, fmt.Errorf("%w: 条目缺少会话标识", ErrMalformedPage)
		}
		if seen[entry.ID] {
			duplicates++
			continue
		}
		seen[entry.ID] = true
		entries = append(entries, entry)
	}
	return entries, duplicates, nil
}

// mergeTitle 在重取页面时保留修订号更高的本地标题。
func mergeTitle(previous, incoming Entry) Entry {
	if previous.TitleRevision <= incoming.TitleRevision {
		return incoming
	}
	incoming.Title = previous.Title
	incoming.TitleSource = previous.TitleSource
	incoming.TitleRevision = previous.TitleRevision
	return incoming
}

func randomID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
