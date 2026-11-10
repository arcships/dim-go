package projection

import (
	"context"
	"errors"
)

// 后台任务输出：按任务标识分页读取。只记游标与单页字节上限，不把整份输出留在
// 内存里，也不落盘——调用方拿到一页就交出去。

// 任务输出分页的字节上限，与服务端一致。
const (
	// defaultExecOutputPageBytes 是每页默认的字节上限。
	defaultExecOutputPageBytes = 64 << 10
	// maxExecOutputPageBytes 是服务端允许的最大单页字节数。
	maxExecOutputPageBytes = 4 << 20
)

// 任务输出的错误值。
var (
	// ErrExecOutputNoReader 表示没有配置输出读取来源。
	ErrExecOutputNoReader = errors.New("任务输出：缺少读取来源")
	// ErrExecOutputNoTask 表示没有任务标识。
	ErrExecOutputNoTask = errors.New("任务输出：缺少任务标识")
)

// ExecOutputRequest 是一次任务输出分页读取。Cursor 是按字节的游标，Limit 是这一页
// 的字节上限。
type ExecOutputRequest struct {
	SessionID string
	BranchID  string
	TaskID    string
	Cursor    int64
	Limit     int
}

// ExecOutputPage 是一页任务输出。
//
//   - Text 是这一页的文本；服务端会丢掉控制字符，所以它可能比 Bytes 短。
//   - Bytes 是这一页消费的输出字节数，游标按它前进，不按文本长度。
//   - NextCursor 只在服务端明确给出时才有意义；给 0 表示它没有给。
//   - Truncated 表示输出文件达到了服务端的字节上限。
type ExecOutputPage struct {
	Text       string
	Bytes      int64
	NextCursor int64
	Truncated  bool
}

// ExecOutputReader 读一页任务输出。集成层用 `session.exec.output.page` 实现。
type ExecOutputReader interface {
	ReadExecOutput(ctx context.Context, req ExecOutputRequest) (ExecOutputPage, error)
}

// ExecOutputChunk 是 Next 交出去的一页。
type ExecOutputChunk struct {
	// Text 是这一页的文本，交给调用方显示或写出；本包不保存它。
	Text string
	// Bytes 是这一页消费的字节数。
	Bytes int64
	// Cursor 是读完这一页之后的游标。
	Cursor int64
	// EOF 表示任务已结束且没有更多字节了。任务还在跑时空页只表示"暂时没有新输出"。
	EOF bool
	// Truncated 表示输出文件达到服务端上限，之后的内容不会再有。
	Truncated bool
}

// ExecOutput 按任务读取输出。它只持有游标，不缓存整份输出。
type ExecOutput struct {
	folder    *Folder
	sessionID string
	branchID  string
	taskID    string
	limit     int
	reader    ExecOutputReader
	cursor    int64
	eof       bool
}

// NewExecOutput 创建任务输出读取器。folder 用来判断任务是否还在运行：结束后读到
// 空页才算 EOF。
func NewExecOutput(folder *Folder, branchID, taskID string, reader ExecOutputReader) (*ExecOutput, error) {
	if reader == nil {
		return nil, ErrExecOutputNoReader
	}
	if taskID == "" {
		return nil, ErrExecOutputNoTask
	}
	if branchID == "" {
		branchID = mainBranch
	}
	output := &ExecOutput{
		folder:   folder,
		branchID: branchID,
		taskID:   taskID,
		limit:    defaultExecOutputPageBytes,
		reader:   reader,
	}
	if folder != nil {
		output.sessionID = folder.sessionID
	}
	return output, nil
}

// SetLimit 设置单页字节上限；超过服务端上限按上限算，小于 1 用默认值。
func (o *ExecOutput) SetLimit(limit int) {
	switch {
	case limit <= 0:
		o.limit = defaultExecOutputPageBytes
	case limit > maxExecOutputPageBytes:
		o.limit = maxExecOutputPageBytes
	default:
		o.limit = limit
	}
}

// Cursor 返回当前游标。
func (o *ExecOutput) Cursor() int64 { return o.cursor }

// EOF 报告是否已经读完（任务结束且没有更多字节）。
func (o *ExecOutput) EOF() bool { return o.eof }

// Next 读下一页。任务还在跑时，空页只是"暂时没有新输出"，不算 EOF；任务结束后读到
// 空页即 EOF。
func (o *ExecOutput) Next(ctx context.Context) (ExecOutputChunk, error) {
	if o.eof {
		return ExecOutputChunk{Cursor: o.cursor, EOF: true}, nil
	}
	page, err := o.reader.ReadExecOutput(ctx, ExecOutputRequest{
		SessionID: o.sessionID,
		BranchID:  o.branchID,
		TaskID:    o.taskID,
		Cursor:    o.cursor,
		Limit:     o.limit,
	})
	if err != nil {
		return ExecOutputChunk{}, err
	}
	next := o.cursor + page.Bytes
	if page.NextCursor > next {
		next = page.NextCursor
	}
	chunk := ExecOutputChunk{
		Text:      page.Text,
		Bytes:     page.Bytes,
		Cursor:    next,
		Truncated: page.Truncated,
	}
	o.cursor = next
	if page.Bytes == 0 && page.Text == "" && !o.taskRunning() {
		o.eof = true
		chunk.EOF = true
	}
	return chunk, nil
}

// taskRunning 报告任务是否还在清单里。没有折叠器时按"已结束"处理：调用方要自己
// 决定读到什么程度。
func (o *ExecOutput) taskRunning() bool {
	if o.folder == nil {
		return false
	}
	return o.folder.TaskRunning(o.branchID, o.taskID)
}
