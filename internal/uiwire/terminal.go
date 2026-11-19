package uiwire

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/arcships/dimagent-mygo/internal/terminal"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// 右栏终端的本机 pty 接线。
//
// 事实归属：pty 会话本身属于 internal/terminal，按会话登记在 app.App 的
// Registry 里（一个会话一个实例）；这里只做两件事——按会话起停终端，以及把
// pty 的输出流切成行给界面读。输出是「读到的字节」，不做行缓冲、不做命令
// 执行：命令由用户敲进 shell，不经过协议，也不产生服务端事实。
//
// 读法：后台一个读协程按会话读 pty，追加到该会话的行缓冲，读到字节就触发重绘；
// 界面每帧读快照（TerminalSnapshot.Lines），不阻塞帧。
const (
	// terminalLineLimit 是一个会话保留的输出行数上限：超出只丢最旧的行，界面
	// 滚到顶部时仍能看到最近的输出。
	terminalLineLimit = 5000
	// terminalLineBytes 是单行的字节上限：程序一直不换行时不无限攒着半行。
	terminalLineBytes = 64 << 10
	// terminalReadChunk 是一次读的字节数。
	terminalReadChunk = 4096
)

// terminalStream 是一个会话的 pty 输出行缓冲。
//
// 半行（还没读到换行）也留着：字节流可能在任意位置切开，界面不该因此多出
// 一行被截断的文本。
type terminalStream struct {
	mu      sync.Mutex
	lines   []string
	partial string
}

// append 追加一段读到的字节，按换行切成行；CRLF 行尾的 '\r' 去掉（其余控制
// 序列原样交给界面解释，见 internal/ui/terminal_ansi.go）。
func (s *terminalStream) append(chunk string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.partial += chunk
	for {
		at := strings.IndexByte(s.partial, '\n')
		if at < 0 {
			break
		}
		line := strings.TrimSuffix(s.partial[:at], "\r")
		s.partial = s.partial[at+1:]
		s.lines = append(s.lines, line)
	}
	if len(s.partial) > terminalLineBytes {
		s.lines = append(s.lines, s.partial)
		s.partial = ""
	}
	if len(s.lines) > terminalLineLimit {
		s.lines = append([]string(nil), s.lines[len(s.lines)-terminalLineLimit:]...)
	}
}

// snapshot 取一份输出行的拷贝（界面每帧读，不共享底层数组）。
func (s *terminalStream) snapshot() []string {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.lines)+1)
	out = append(out, s.lines...)
	// 半行也画出来：提示符通常没有换行就出现了，不画等于界面一直空着。
	if s.partial != "" {
		out = append(out, s.partial)
	}
	return out
}

// StartTerminal 在窗口正在看的会话里启动本机 shell。
func (w *App) StartTerminal(ctx context.Context) error {
	return w.startTerminalFor(ctx, w.currentSession())
}

// StopTerminal 停止窗口正在看的会话的终端。
func (w *App) StopTerminal(ctx context.Context) error {
	return w.stopTerminalFor(ctx, w.currentSession())
}

// WriteTerminal 把输入原样写回窗口正在看的会话的 pty。
func (w *App) WriteTerminal(ctx context.Context, data string) error {
	return w.writeTerminalFor(ctx, w.currentSession(), data)
}

// startTerminalFor 为指定会话启动终端，并挂上输出读协程。
//
// 工作目录只取目录里的会话事实：没有会话或会话没有工作目录时明确失败，不拿
// 工作区根顶替。该会话已有运行中的终端时接着读它的输出，不重复启动。
func (w *App) startTerminalFor(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("没有打开的会话")
	}
	registry := w.terminalsRegistry()
	if registry == nil {
		return errors.New("本机没有终端能力")
	}
	cwd := w.sessionWorkspace(sessionID)
	if strings.TrimSpace(cwd) == "" {
		return errors.New("这个会话没有工作目录")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	stream := w.terminalStreamEnsure(sessionID)
	if current, ok := registry.Get(sessionID); ok && current.Alive() {
		// 已有运行中的终端：接着读它的输出，不重复启动、也不清掉已读到的行。
		w.attachTerminalReader(sessionID, current, stream)
		return nil
	}
	stream = w.resetTerminalStream(sessionID)
	term, err := w.A.StartTerminal(sessionID, terminal.Options{})
	if err != nil {
		return fmt.Errorf("启动终端：%w", err)
	}
	w.attachTerminalReader(sessionID, term, stream)
	w.changed()
	return nil
}

// stopTerminalFor 停止指定会话的终端；没有终端实例时按已停止处理（关闭是可
// 重复的，界面上重复点不该报错）。
func (w *App) stopTerminalFor(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("没有打开的会话")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	registry := w.terminalsRegistry()
	if registry == nil {
		return nil
	}
	if err := registry.Stop(sessionID); err != nil && !errors.Is(err, terminal.ErrNoTerminal) {
		return fmt.Errorf("停止终端：%w", err)
	}
	w.changed()
	return nil
}

// writeTerminalFor 把输入原样写回指定会话的 pty。
//
// 没有运行中的终端时明确失败：写进一个已关闭的设备只会让用户以为敲进去了。
func (w *App) writeTerminalFor(ctx context.Context, sessionID, data string) error {
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("没有打开的会话")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	registry := w.terminalsRegistry()
	if registry == nil {
		return errors.New("本机没有终端能力")
	}
	term, ok := registry.Get(sessionID)
	if !ok || !term.Alive() {
		return errors.New("没有运行中的终端，先新建终端")
	}
	if _, err := term.Write([]byte(data)); err != nil {
		return fmt.Errorf("写入终端：%w", err)
	}
	return nil
}

// terminalsRegistry 取本机终端登记表；没有接上集成层时返回 nil。
func (w *App) terminalsRegistry() *terminal.Registry {
	if w.A == nil {
		return nil
	}
	return w.A.Terminals()
}

// resetTerminalStream 换一个新的输出缓冲（新终端从空开始），返回它。
func (w *App) resetTerminalStream(sessionID string) *terminalStream {
	stream := &terminalStream{}
	w.termMu.Lock()
	if w.termStreams == nil {
		w.termStreams = make(map[string]*terminalStream)
	}
	w.termStreams[sessionID] = stream
	w.termMu.Unlock()
	return stream
}

// terminalStreamEnsure 取会话的输出缓冲，没有就建一个空的（不清已读到的行）。
func (w *App) terminalStreamEnsure(sessionID string) *terminalStream {
	w.termMu.Lock()
	defer w.termMu.Unlock()
	if w.termStreams == nil {
		w.termStreams = make(map[string]*terminalStream)
	}
	stream, ok := w.termStreams[sessionID]
	if !ok || stream == nil {
		stream = &terminalStream{}
		w.termStreams[sessionID] = stream
	}
	return stream
}

// terminalStreamFor 取会话的输出缓冲；没有时返回 nil（快照里不给 Lines）。
func (w *App) terminalStreamFor(sessionID string) *terminalStream {
	w.termMu.Lock()
	defer w.termMu.Unlock()
	return w.termStreams[sessionID]
}

// attachTerminalReader 给一个终端实例挂读协程；同一实例只挂一个。
func (w *App) attachTerminalReader(sessionID string, term *terminal.Terminal, stream *terminalStream) {
	w.termMu.Lock()
	if w.termReaders == nil {
		w.termReaders = make(map[string]*terminal.Terminal)
	}
	if w.termReaders[sessionID] == term {
		w.termMu.Unlock()
		return
	}
	w.termReaders[sessionID] = term
	w.termMu.Unlock()
	go w.readTerminal(sessionID, term, stream)
}

// readTerminal 持续读 pty，把输出切行放进缓冲，读到就重绘；终端停止（EOF）即返回。
func (w *App) readTerminal(sessionID string, term *terminal.Terminal, stream *terminalStream) {
	defer func() {
		w.termMu.Lock()
		if w.termReaders[sessionID] == term {
			delete(w.termReaders, sessionID)
		}
		w.termMu.Unlock()
	}()
	buf := make([]byte, terminalReadChunk)
	for {
		n, err := term.Read(buf)
		if n > 0 {
			stream.append(string(buf[:n]))
			w.changed()
		}
		if err != nil {
			return
		}
	}
}

// Session 上的三个终端动作落到本窗口固定看的会话。
func (s *Session) StartTerminal(ctx context.Context) error {
	return s.App.startTerminalFor(ctx, s.ID())
}

func (s *Session) StopTerminal(ctx context.Context) error {
	return s.App.stopTerminalFor(ctx, s.ID())
}

func (s *Session) WriteTerminal(ctx context.Context, data string) error {
	return s.App.writeTerminalFor(ctx, s.ID(), data)
}

// terminalSnapshotFor 组装会话的终端快照：状态来自 Registry，输出行来自读到的
// 字节。终端停止后仍给最后读到的行（像真终端一样留着最后的输出）。
func (w *App) terminalSnapshotFor(sessionID string) ui.TerminalSnapshot {
	registry := w.terminalsRegistry()
	if registry == nil || sessionID == "" {
		return ui.TerminalSnapshot{}
	}
	term, ok := registry.Get(sessionID)
	if !ok {
		return ui.TerminalSnapshot{}
	}
	out := ui.TerminalSnapshot{Active: true, Running: term.Alive()}
	if !out.Running {
		if status, ok := term.ExitStatus(); ok {
			out.Exited = true
			out.ExitCode = status.Code
		}
	}
	out.Lines = w.terminalStreamFor(sessionID).snapshot()
	return out
}
