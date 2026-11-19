package uiwire

import (
	"context"
	"fmt"
	"sync"

	"github.com/arcships/dimagent-mygo/internal/ui"
)

// Session 是共享连接上一个窗口的会话视图：它只固定「这个窗口在看哪个会话」，
// 连接、目录、设置与缓存仍只有一份（见 App）。
//
// 为什么按窗口固定会话：多窗口里每个窗口要能各自选中一个会话，而 App Server
// 只该有一条控制连接与一份订阅。会话标识为空表示跟随 App 的当前会话（主窗口
// 启动阶段用）；窗口一旦打开过某个会话，就固定在自己选的那个上，互不影响。
//
// 读与写都按这个标识取：快照、审批、提问、终端、改动、提交、分支、目标、命令
// 执行都是「某个会话的操作」。连接级动作（重连、偏好、供应商、账号、目录改名
// 与归档等本来就带会话标识的方法）原样从 App 上取，不在这里另做一份。
type Session struct {
	*App
	mu  sync.RWMutex
	pin string
}

// 窗口视图就是界面要的三个接口；少实现一个方法在编译期就报错。
var (
	_ ui.Source     = (*Session)(nil)
	_ ui.Intents    = (*Session)(nil)
	_ ui.PanelFacts = (*Session)(nil)
)

// Pin 返回一个绑定到会话的窗口视图。sessionID 为空表示跟随 App 的当前会话。
func (w *App) Pin(sessionID string) *Session {
	return &Session{App: w, pin: sessionID}
}

// ID 是本窗口当前看的会话；没有固定时跟随 App 的当前会话。
func (s *Session) ID() string {
	s.mu.RLock()
	id := s.pin
	s.mu.RUnlock()
	if id != "" {
		return id
	}
	return s.App.currentSession()
}

// SetPin 把本窗口固定到某个会话（空串表示重新跟随 App 的当前会话）。
func (s *Session) SetPin(id string) {
	s.mu.Lock()
	s.pin = id
	s.mu.Unlock()
}

// ---- 读：会话相关的事实按本窗口的会话取 ----

func (s *Session) Session() (ui.SessionSnapshot, bool) { return s.App.sessionSnapshot(s.ID()) }

func (s *Session) Approval() (ui.ApprovalSnapshot, bool) { return s.App.approvalFor(s.ID()) }

func (s *Session) Question() (ui.QuestionSnapshot, bool) { return s.App.questionFor(s.ID()) }

func (s *Session) Settings() ui.SettingsSnapshot { return s.App.settingsFor(s.ID()) }

func (s *Session) Terminal() ui.TerminalSnapshot { return s.App.terminalFor(s.ID()) }

func (s *Session) Changes(request ui.ChangesRequest) ui.ChangeSet {
	return s.App.changesFor(request, s.ID())
}

// ---- 写：会话写入落到本窗口的会话 ----

func (s *Session) OpenSession(ctx context.Context, name string) error {
	if _, err := s.App.A.OpenSession(ctx, name); err != nil {
		return fmt.Errorf("打开会话 %s：%w", name, err)
	}
	s.SetPin(name)
	s.App.changed()
	return nil
}

func (s *Session) Submit(ctx context.Context, text string) error {
	return s.SubmitWithContent(ctx, text, nil)
}

func (s *Session) SubmitWithContent(
	ctx context.Context,
	text string,
	blocks []ui.ContentBlockSnapshot,
) error {
	return s.App.submitWithContentFor(ctx, s.ID(), text, blocks)
}

func (s *Session) Queue(ctx context.Context, text string) error { return s.Submit(ctx, text) }

func (s *Session) Stop(ctx context.Context) error { return s.App.stopFor(ctx, s.ID()) }

func (s *Session) RespondApproval(ctx context.Context, decision string) error {
	return s.App.respondApprovalFor(ctx, s.ID(), decision)
}

func (s *Session) RespondQuestion(ctx context.Context, answer string) error {
	return s.App.respondQuestionFor(ctx, s.ID(), answer)
}

func (s *Session) SkipQuestion(ctx context.Context) error {
	return s.App.skipQuestionFor(ctx, s.ID())
}

func (s *Session) RetrySubmit(ctx context.Context) error { return s.Submit(ctx, "") }

func (s *Session) CreateSession(ctx context.Context, cwd string) error {
	return s.App.createSessionFor(ctx, s.ID(), cwd)
}

func (s *Session) DeleteSession(ctx context.Context, sessionID string) error {
	next, err := s.App.deleteSessionFor(ctx, s.ID(), sessionID)
	if err != nil {
		return err
	}
	s.SetPin(next)
	return nil
}

func (s *Session) SetPermissionTier(ctx context.Context, tier string) error {
	return s.App.setPermissionTierFor(ctx, s.ID(), tier)
}

// CompactContext 压缩本窗口会话的上下文（斜杠命令 /compact）。
func (s *Session) CompactContext(ctx context.Context) error {
	return s.App.compactContextFor(ctx, s.ID())
}

func (s *Session) SelectBranch(ctx context.Context, branchID string) error {
	return s.App.selectBranchFor(ctx, s.ID(), branchID)
}

func (s *Session) CreateBranch(ctx context.Context, parentID string, turnIndex int, prompt string) error {
	return s.App.createBranchFor(ctx, s.ID(), parentID, turnIndex, prompt)
}

func (s *Session) EndBranch(ctx context.Context, branchID string) error {
	return s.App.endBranchFor(ctx, s.ID(), branchID)
}

func (s *Session) WriteAttachment(
	ctx context.Context,
	name, text string,
) (ui.AttachmentRefSnapshot, error) {
	return s.App.writeAttachmentFor(ctx, s.ID(), name, text)
}

func (s *Session) ReadGoal(ctx context.Context) (string, error) {
	return s.App.readGoalFor(ctx, s.ID())
}

func (s *Session) SetGoal(ctx context.Context, objective string) error {
	return s.App.setGoalFor(ctx, s.ID(), objective)
}

func (s *Session) EditGoal(ctx context.Context, objective string) error {
	return s.App.editGoalFor(ctx, s.ID(), objective)
}

func (s *Session) PauseGoal(ctx context.Context) error { return s.App.pauseGoalFor(ctx, s.ID()) }

func (s *Session) ResumeGoal(ctx context.Context) error { return s.App.resumeGoalFor(ctx, s.ID()) }

func (s *Session) ClearGoal(ctx context.Context) error { return s.App.clearGoalFor(ctx, s.ID()) }

func (s *Session) StopExecTask(ctx context.Context, taskID string) error {
	return s.App.stopExecTaskFor(ctx, s.ID(), taskID)
}

func (s *Session) RunShell(ctx context.Context, command string) (string, error) {
	return s.App.runShellFor(ctx, s.ID(), command)
}

func (s *Session) StopShell(ctx context.Context, runID string) error {
	return s.App.stopShellFor(ctx, s.ID(), runID)
}

func (s *Session) ImportMCP(ctx context.Context, sourceID string) (int, error) {
	return s.App.importMCPFor(ctx, s.ID(), sourceID)
}
