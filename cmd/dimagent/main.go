// Command dimagent is the MyGo native UI entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/integration"
	"github.com/arcships/dimagent-mygo/internal/session"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// sessionActions is the part of integration.SessionController the window uses:
// the live view it adapts and the mutations the shell's callbacks send. Keeping
// it small lets tests substitute a session without a daemon.
type sessionActions interface {
	LiveState(summary appserver.SessionSummary) *dimui.State
	// StateForBranch builds the view of one branch from the session's own
	// projection, so a branch the window opened shows that branch's history
	// instead of the default one.
	StateForBranch(summary appserver.SessionSummary, branchID string) *dimui.State
	// OpenBranch reads one branch's first-screen history. Branch switching has
	// no RPC of its own: the read is session.history for that branchId.
	OpenBranch(ctx context.Context, branchID string) (session.BranchOpenResult, error)
	// EndBranch asks the App Server to end one non-main branch. It removes no
	// local fact: the directory and projection change only when the App Server
	// reports the new state.
	EndBranch(ctx context.Context, branchID string) (appserver.BranchEndOutcome, error)
	// RefreshBranchDirectory re-reads session.branches into the read-only
	// directory the window shows.
	RefreshBranchDirectory(ctx context.Context) error
	// SessionBranchID is the branch the session submits and stops on by
	// default. A window that shows another branch names that branch explicitly;
	// this one stays the read-back branch for a submit that named none.
	SessionBranchID() string
	Submit(ctx context.Context, text string) (appserver.SubmitResult, error)
	// SubmitWithID also returns the submission id, which is the only key
	// session.input_lookup accepts when a submit's outcome is unknown.
	SubmitWithID(ctx context.Context, text string) (appserver.SubmitResult, string, error)
	// SubmitBranchWithID sends on the explicit branch the window shows and
	// keeps the idempotency key for a same-branch read-back.
	SubmitBranchWithID(ctx context.Context, branchID, text string) (appserver.SubmitResult, string, error)
	// LookupInput reads a submission back through session.input_lookup. It must
	// never resend the submission.
	LookupInput(ctx context.Context, branchID, submissionID string) (appserver.InputLookupResult, error)
	// LookupInputBranch is LookupInput on the branch the submission was sent
	// on, so a side-branch submission is not resolved against the default one.
	LookupInputBranch(ctx context.Context, branchID, submissionID string) (appserver.InputLookupResult, error)
	RespondApproval(ctx context.Context, branchID, requestID, decision string) (appserver.ApprovalRespondOutcome, error)
	Stop(ctx context.Context) error
	// StopBranch stops the branch the window shows instead of the session's
	// default branch.
	StopBranch(ctx context.Context, branchID string) error
	// Delete asks the App Server to delete the session. The row leaves the
	// catalog only after a nil error.
	Delete(ctx context.Context) error
	Reattach(ctx context.Context) error
	Close(ctx context.Context) error
}

// The window's controller must be the integration one, including the read-back
// helper an unknown submit is resolved with.
var _ sessionActions = (*integration.SessionController)(nil)

// sessionBridge owns the selected session's controller. It swaps the controller
// when the sidebar chooses another session and guards the pointer that the UI
// thread and background mutations share.
type sessionBridge struct {
	sup *supervisor.Supervisor

	// openSession opens a session through the supervisor. Tests replace it;
	// the app uses integration.OpenSession.
	openSession func(context.Context, *supervisor.Supervisor, string) (sessionActions, error)
	// createSession creates a session and opens its stream. Tests replace it;
	// the app uses integration.CreateSession.
	createSession func(context.Context, *supervisor.Supervisor, appserver.CreateParams) (sessionActions, appserver.CreateResult, error)
	// catalogSessions reads the authoritative catalog. Tests replace it; the
	// app uses integration.Catalog.
	catalogSessions func(context.Context, *supervisor.Supervisor) ([]dimui.Session, error)

	mu         sync.Mutex
	controller sessionActions
	summary    appserver.SessionSummary
}

var (
	errNoSession      = errors.New("dimagent: no session selected")
	errStaleSelection = errors.New("dimagent: selection changed before the action")
)

// current returns the controller and summary together, so an action cannot
// observe a half-swapped session selection.
func (b *sessionBridge) current() (sessionActions, appserver.SessionSummary) {
	if b == nil {
		return nil, appserver.SessionSummary{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.controller, b.summary
}

// controllerFor returns the current controller only when it belongs to the
// selected session. An empty id keeps the legacy fixture behavior of using the
// current controller; the window always passes the selected id.
func (b *sessionBridge) controllerFor(sessionID string) (sessionActions, error) {
	controller, summary := b.current()
	if controller == nil {
		return nil, errNoSession
	}
	if sessionID != "" && summary.SessionID != sessionID {
		return nil, errStaleSelection
	}
	return controller, nil
}

// LiveState returns a fresh view of the selected controller, or nil without one.
// It never reads a branch other than the controller's current branch.
func (b *sessionBridge) LiveState() *dimui.State {
	controller, summary := b.current()
	if controller == nil {
		return nil
	}
	return controller.LiveState(summary)
}

// LiveStateFor returns the controller's view for one branch of the selected
// session. A sessionID that no longer belongs to the attached controller yields
// nil, so a view read for an older selection is never published; an empty
// branchID asks for the session's default branch.
func (b *sessionBridge) LiveStateFor(sessionID, branchID string) *dimui.State {
	controller, summary := b.current()
	if controller == nil {
		return nil
	}
	if sessionID != "" && summary.SessionID != sessionID {
		return nil
	}
	return controller.StateForBranch(summary, branchID)
}

// OpenBranchFor opens one branch through the controller that still belongs to
// sessionID. The read is the App Server's; the result is not turned into local
// branch state here.
func (b *sessionBridge) OpenBranchFor(ctx context.Context, sessionID, branchID string) error {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return err
	}
	_, err = controller.OpenBranch(ctx, branchID)
	return err
}

// EndBranchFor ends one branch through the controller that still belongs to
// sessionID. The App Server's outcome is returned unchanged; no directory entry
// is dropped here.
func (b *sessionBridge) EndBranchFor(ctx context.Context, sessionID, branchID string) (appserver.BranchEndOutcome, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return "", err
	}
	return controller.EndBranch(ctx, branchID)
}

// RefreshBranchesFor re-reads the selected session's branch directory through
// the controller that still belongs to sessionID, so a directory read for an
// older selection cannot land on the new one.
func (b *sessionBridge) RefreshBranchesFor(ctx context.Context, sessionID string) error {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return err
	}
	return controller.RefreshBranchDirectory(ctx)
}

// SessionBranchID reports the branch the session submits and stops on when no
// branch is named. A branch the window opened is named explicitly instead; an
// empty displayed branch means the session's own branch, which is what this
// reports and what a read-back must name then.
func (b *sessionBridge) SessionBranchID() string {
	controller, _ := b.current()
	if controller == nil {
		return ""
	}
	return controller.SessionBranchID()
}

// SubmitFor sends text through the controller that still belongs to sessionID.
func (b *sessionBridge) SubmitFor(ctx context.Context, sessionID, text string) error {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return err
	}
	_, err = controller.Submit(ctx, text)
	return err
}

// SubmitWithIDFor sends text through the controller that still belongs to
// sessionID and keeps the submission id, so an unknown result can be read back
// instead of resent.
func (b *sessionBridge) SubmitWithIDFor(ctx context.Context, sessionID, text string) (appserver.SubmitResult, string, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return appserver.SubmitResult{}, "", err
	}
	return controller.SubmitWithID(ctx, text)
}

// SubmitBranchWithIDFor sends text on branchID through the controller that
// still belongs to sessionID. A submission captured for an older selection
// never reaches the newly attached session, which is what keeps a branch of the
// previous session from naming the new one's submit.
func (b *sessionBridge) SubmitBranchWithIDFor(ctx context.Context, sessionID, branchID, text string) (appserver.SubmitResult, string, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return appserver.SubmitResult{}, "", err
	}
	return controller.SubmitBranchWithID(ctx, branchID, text)
}

// LookupInputFor reads one submission back through the controller that still
// belongs to sessionID. It asks the App Server every time and never resends the
// submission.
func (b *sessionBridge) LookupInputFor(ctx context.Context, sessionID, branchID, submissionID string) (appserver.InputLookupResult, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return appserver.InputLookupResult{}, err
	}
	return controller.LookupInput(ctx, branchID, submissionID)
}

// LookupInputBranchFor reads one submission back on the branch it was sent on,
// through the controller that still belongs to sessionID.
func (b *sessionBridge) LookupInputBranchFor(ctx context.Context, sessionID, branchID, submissionID string) (appserver.InputLookupResult, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return appserver.InputLookupResult{}, err
	}
	return controller.LookupInputBranch(ctx, branchID, submissionID)
}

// Submit sends text through the current controller for fixture callers.
func (b *sessionBridge) Submit(ctx context.Context, text string) error {
	return b.SubmitFor(ctx, "", text)
}

// DeleteFor deletes the session through the controller that still belongs to
// sessionID. The bridge removes no row: the window does that after a nil error.
func (b *sessionBridge) DeleteFor(ctx context.Context, sessionID string) error {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return err
	}
	return controller.Delete(ctx)
}

// StopFor stops the controller that still belongs to sessionID.
func (b *sessionBridge) StopFor(ctx context.Context, sessionID string) error {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return err
	}
	return controller.Stop(ctx)
}

// StopBranchFor stops branchID through the controller that still belongs to
// sessionID, so a stop captured while a side branch was shown ends that branch
// and not another session's.
func (b *sessionBridge) StopBranchFor(ctx context.Context, sessionID, branchID string) error {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return err
	}
	return controller.StopBranch(ctx, branchID)
}

// Stop stops the current controller for fixture callers.
func (b *sessionBridge) Stop(ctx context.Context) error {
	return b.StopFor(ctx, "")
}

// submitOn sends text on the branch the window shows. A side branch is named
// explicitly so the App Server puts the input on the branch the user reads; the
// default branch has no branch-specific submit to name, so it keeps the session
// path and the session layer's own branch. Either way the branch that was named
// here is the branch a read-back must name.
func (b *sessionBridge) submitOn(ctx context.Context, sessionID, branchID, text string) (appserver.SubmitResult, string, error) {
	if branchID == "" || branchID == appserver.MainBranchID {
		return b.SubmitWithIDFor(ctx, sessionID, text)
	}
	return b.SubmitBranchWithIDFor(ctx, sessionID, branchID, text)
}

// stopOn stops the branch the window shows. The default branch keeps the
// session's branch-less stop; a side branch is named so the turn the user is
// reading is the one that ends.
func (b *sessionBridge) stopOn(ctx context.Context, sessionID, branchID string) error {
	if branchID == "" || branchID == appserver.MainBranchID {
		return b.StopFor(ctx, sessionID)
	}
	return b.StopBranchFor(ctx, sessionID, branchID)
}

// RespondApprovalFor sends one folded decision through the controller that
// still belongs to sessionID, so a click on an older conversation's approval
// cannot reach a newly attached session. It returns the App Server's own
// outcome; the caller reports it instead of assuming success.
func (b *sessionBridge) RespondApprovalFor(ctx context.Context, sessionID, branchID, requestID, decision string) (appserver.ApprovalRespondOutcome, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return "", err
	}
	return controller.RespondApproval(ctx, branchID, requestID, decision)
}

// reattachFor reconnects the controller that still belongs to sessionID.
func (b *sessionBridge) reattachFor(ctx context.Context, sessionID string) error {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return err
	}
	return controller.Reattach(ctx)
}

// open attaches row's session and makes it the current one, closing the
// previous controller once the new attachment succeeded.
func (b *sessionBridge) open(ctx context.Context, row dimui.Session) error {
	if b == nil {
		return errNoSession
	}
	open := b.openSession
	if open == nil {
		open = func(ctx context.Context, sup *supervisor.Supervisor, id string) (sessionActions, error) {
			return integration.OpenSession(ctx, sup, id)
		}
	}
	controller, err := open(ctx, b.sup, row.ID)
	if err != nil {
		return err
	}
	b.mu.Lock()
	previous := b.controller
	b.controller, b.summary = controller, summaryForRow(row)
	b.mu.Unlock()
	if previous != nil {
		_ = previous.Close(ctx)
	}
	return nil
}

// create asks the App Server for a new session and installs its controller, the
// same way open installs one. The row it returns carries the App Server's own
// session id; this layer invents no title and adds no session the daemon did not
// return.
func (b *sessionBridge) create(ctx context.Context, params appserver.CreateParams) (dimui.Session, error) {
	if b == nil {
		return dimui.Session{}, errNoSession
	}
	create := b.createSession
	if create == nil {
		create = func(ctx context.Context, sup *supervisor.Supervisor, params appserver.CreateParams) (sessionActions, appserver.CreateResult, error) {
			return integration.CreateSession(ctx, sup, params)
		}
	}
	controller, result, err := create(ctx, b.sup, params)
	if err != nil {
		if controller != nil {
			_ = controller.Close(ctx)
		}
		return dimui.Session{}, err
	}
	if controller == nil || result.SessionID == "" {
		if controller != nil {
			_ = controller.Close(ctx)
		}
		return dimui.Session{}, errors.New("dimagent: session.create returned no session")
	}
	// A session the catalog has not titled yet shows its id, which is what
	// integration.Catalog does for a summary without a title or preview.
	row := dimui.Session{ID: result.SessionID, Title: result.SessionID}
	b.mu.Lock()
	previous := b.controller
	b.controller, b.summary = controller, summaryForRow(row)
	b.mu.Unlock()
	if previous != nil {
		_ = previous.Close(ctx)
	}
	return row, nil
}

// catalog reads the authoritative session list through the supervisor.
func (b *sessionBridge) catalog(ctx context.Context) ([]dimui.Session, error) {
	if b == nil {
		return nil, errNoSession
	}
	read := b.catalogSessions
	if read == nil {
		read = func(ctx context.Context, sup *supervisor.Supervisor) ([]dimui.Session, error) {
			return integration.Catalog(ctx, sup)
		}
	}
	return read(ctx, b.sup)
}

// Close releases the selected session and the supervisor.
func (b *sessionBridge) Close(ctx context.Context) error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	controller := b.controller
	b.controller = nil
	b.mu.Unlock()
	if controller != nil {
		if err := controller.Close(ctx); err != nil {
			if b.sup != nil {
				_, _ = b.sup.Close(ctx)
			}
			return err
		}
	}
	if b.sup == nil {
		return nil
	}
	_, err := b.sup.Close(ctx)
	return err
}

// stateFromDaemon is deliberately opt-in: the UI never guesses a product home
// and never reads local daemon persistence. The supervisor is the sole source
// of transport/session facts when DIMAGENT_PRODUCT_HOME is configured.
func stateFromDaemon(ctx context.Context, productHome string) (*dimui.State, *sessionBridge, error) {
	if strings.TrimSpace(productHome) == "" {
		return &dimui.State{Connection: dimui.ConnUnknown}, nil, nil
	}
	sup, err := newSupervisor(productHome)
	if err != nil {
		return nil, nil, err
	}
	if _, err := sup.Start(ctx); err != nil {
		_, _ = sup.Close(context.Background())
		return nil, nil, err
	}
	bridge := &sessionBridge{sup: sup}
	rows, err := integration.Catalog(ctx, sup)
	if err != nil {
		_ = bridge.Close(context.Background())
		return nil, nil, err
	}
	state := &dimui.State{Sessions: rows, Connection: dimui.ConnReady}
	if len(rows) == 0 {
		return state, bridge, nil
	}
	if err := bridge.open(ctx, rows[0]); err != nil {
		_ = bridge.Close(context.Background())
		return nil, nil, err
	}
	// The controller's session drain folds stream events into memory; this
	// adapter only reads that in-memory projection, never a daemon file.
	applyView(state, bridge.LiveState())
	return state, bridge, nil
}

// applyView copies what the selected session's live view owns into state. It
// leaves the catalog, the draft and the callbacks alone: those belong to the
// window, not to one session's projection. A view for another session is stale
// (for example an old controller while a new attachment is still in flight) and
// must not overwrite the window's current selection.
func applyView(state, view *dimui.State) {
	if state == nil || view == nil {
		return
	}
	if state.Selected != "" && view.Selected != "" && state.Selected != view.Selected {
		return
	}
	state.Selected = view.Selected
	state.SelectedBranch = view.SelectedBranch
	state.Branches = append([]dimui.Branch(nil), view.Branches...)
	state.Connection = view.Connection
	state.ConnectionError = view.ConnectionError
	state.ConversationLoading = view.ConversationLoading
	state.SessionClosed = view.SessionClosed
	state.SessionClosedReason = view.SessionClosedReason
	state.Run = view.Run
	state.Messages = append([]dimui.Message(nil), view.Messages...)
	state.Keys = view.Keys
	state.ModelProgress = view.ModelProgress
	state.Notices = append([]dimui.Notice(nil), view.Notices...)
}

func summaryForRow(row dimui.Session) appserver.SessionSummary {
	return appserver.SessionSummary{SessionID: row.ID, Title: &row.Title}
}

func newSupervisor(productHome string) (*supervisor.Supervisor, error) {
	return supervisor.New(supervisor.Config{
		ProductHome: productHome,
		// The mock model is the daemon's own flag for acceptance runs: it echoes
		// turns without a provider. It is opt-in through the environment so a
		// normal launch never hides a missing model.
		MockModel: os.Getenv("DIMAGENT_MOCK_MODEL") == "1",
	})
}

// app owns the window's state and the callbacks the shell calls. State writes
// from background work go through Window.Update, which runs them on the main
// thread; the callbacks themselves run on the UI thread.
type app struct {
	state  *dimui.State
	bridge *sessionBridge
	win    *mygo.Window

	mu        sync.Mutex
	selectGen uint64
	// branchGen and branch are the window's branch display choice: branchGen
	// grows on every branch request and on every session selection, so a read
	// that belongs to an older choice is dropped, and branch is the branch the
	// window shows, moved only after the App Server opened it.
	branchGen     uint64
	branch        string
	refreshCancel context.CancelFunc
	// updateHook is test-only; production updates run through Window.Update.
	updateHook func(func(*dimui.State))
	// selectMu keeps session opens in order: a newer choice waits for the
	// attachment in flight, then replaces it. A creation installs a controller
	// the same way, so it takes the same lock.
	selectMu sync.Mutex
	// createParams builds session.create's request. Tests replace it; the app
	// uses defaultCreateParams.
	createParams func() (appserver.CreateParams, error)
	// appearance is the desktop-local appearance preference store: the values
	// loaded before the first frame and the snapshot the window writes when the
	// settings surface reports a choice. See settings.go.
	appearance appearance
}

func (a *app) bind() {
	a.state.OnSelect = a.selectSession
	a.state.OnDeleteSession = a.deleteSession
	a.state.OnSelectBranch = a.selectBranch
	a.state.OnEndBranch = a.endBranch
	a.state.OnRefreshBranches = a.refreshBranches
	a.state.OnCreateSession = a.createSession
	a.state.OnSubmit = a.submit
	a.state.OnStop = a.stop
	a.state.OnReconnect = a.reconnect
	a.state.OnApproval = a.respondApproval
}

// update applies fn to the window's state on the main thread. It does nothing
// before the window exists.
func (a *app) update(fn func(*dimui.State)) {
	if a.state == nil {
		return
	}
	if a.updateHook != nil {
		a.updateHook(fn)
		return
	}
	if a.win == nil {
		return
	}
	a.win.Update(func() { fn(a.state) })
}

func (a *app) session(id string) (dimui.Session, bool) {
	for _, row := range a.state.Sessions {
		if row.ID == id {
			return row, true
		}
	}
	return dimui.Session{}, false
}

// beginSelect records a new selection and returns its generation. It also
// forgets the branch the previous session displayed: a branch read still in
// flight belongs to that session and must not land on the new one.
func (a *app) beginSelect() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.selectGen++
	a.branchGen++
	a.branch = ""
	return a.selectGen
}

// currentSelection reports the latest selection generation.
func (a *app) currentSelection() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.selectGen
}

// selectionCurrent reports whether gen is still the latest selection.
func (a *app) selectionCurrent(gen uint64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.selectGen == gen
}

// displayBranch reports the branch the window shows and the generation a view
// of it belongs to.
func (a *app) displayBranch() (string, uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.branch, a.branchGen
}

// beginBranchSelect records a branch request and returns its generation. It does
// not move the displayed branch: only a branch the App Server opened may
// replace it.
func (a *app) beginBranchSelect() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.branchGen++
	return a.branchGen
}

// branchCurrent reports whether gen is still the newest branch request.
func (a *app) branchCurrent(gen uint64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.branchGen == gen
}

// commitBranch moves the displayed branch to the one the App Server opened and
// starts a new view generation, so a refresh that read the previous branch is
// dropped instead of painting over the new one. It refuses a request a newer
// choice already replaced.
func (a *app) commitBranch(branchID string, gen uint64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.branchGen != gen {
		return false
	}
	a.branch = branchID
	a.selectGen++
	return true
}

// resetBranch forgets the displayed branch for a selection that has none of its
// own, such as a newly created session: the session's default branch is the
// fact, and no branch read of the previous session may land on it.
func (a *app) resetBranch() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.branchGen++
	a.branch = ""
}

// view reads the selected controller's view for the branch the window displays.
// A branch the session layer has not loaded still comes from the controller,
// which falls back to the session's default branch; no branch is synthesized.
func (a *app) view() *dimui.State {
	branch, _ := a.displayBranch()
	return a.bridge.LiveStateFor("", branch)
}

// actionBranch reports the branch a submit or a stop names: the branch the
// window shows, or the session's own branch when it shows none. A displayed
// branch only moves after the App Server opened it, and a session selection
// clears it, so a branch the previous session showed can never name an action on
// the new one.
func (a *app) actionBranch() string {
	branch, _ := a.displayBranch()
	if branch != "" || a.bridge == nil {
		return branch
	}
	return a.bridge.SessionBranchID()
}

// startRefresh periodically rebuilds the selected view from the controller's
// public LiveState snapshot. It does not create local session/message facts.
// Calling it again replaces the previous loop.
func (a *app) startRefresh(interval time.Duration) {
	if interval <= 0 {
		interval = 250 * time.Millisecond
	}
	a.mu.Lock()
	if a.refreshCancel != nil {
		a.refreshCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.refreshCancel = cancel
	a.mu.Unlock()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if a.bridge == nil {
					continue
				}
				gen := a.currentSelection()
				branch, branchGen := a.displayBranch()
				view := a.bridge.LiveStateFor("", branch)
				if view == nil {
					continue
				}
				a.update(func(state *dimui.State) {
					// A selection or branch choice may complete while the view is
					// rebuilding; do not let that older snapshot overwrite the new
					// conversation.
					if !a.selectionCurrent(gen) || !a.branchCurrent(branchGen) {
						return
					}
					applyView(state, view)
				})
			}
		}
	}()
}

// stopRefresh terminates the background refresh loop.
func (a *app) stopRefresh() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.refreshCancel != nil {
		a.refreshCancel()
		a.refreshCancel = nil
	}
}

// notice publishes a bounded, inspectable window action outcome. It is
// deliberately a throwaway view fact: daemon/session remains the source of
// timeline truth. Action outcomes live apart from projection Notices so a live
// refresh cannot erase them.
func (a *app) notice(kind, id string) {
	a.update(func(state *dimui.State) {
		state.ActionNotices = appendActionNotice(state.ActionNotices, dimui.Notice{Kind: kind, ID: id})
		if strings.HasSuffix(kind, ".error") {
			state.ActionError = id
		} else {
			state.ActionError = ""
		}
	})
}

// appendActionNotice bounds the window-owned action outcome log.
func appendActionNotice(notices []dimui.Notice, notice dimui.Notice) []dimui.Notice {
	notices = append(notices, notice)
	if len(notices) > 8 {
		notices = append([]dimui.Notice(nil), notices[len(notices)-8:]...)
	}
	return notices
}

// selectSession opens the chosen session in the background and publishes its
// live view. The callback runs on the UI thread, so loading shows at once while
// the App Server call stays off the frame.
func (a *app) selectSession(id string) {
	if a.bridge == nil || a.state == nil {
		return
	}
	row, ok := a.session(id)
	if !ok {
		return
	}
	gen := a.beginSelect()
	a.state.ConversationLoading = true
	go func() {
		a.selectMu.Lock()
		defer a.selectMu.Unlock()
		if !a.selectionCurrent(gen) {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.bridge.open(ctx, row); err != nil {
			if !a.selectionCurrent(gen) {
				return
			}
			log.Printf("dimagent: open session %q: %v", id, err)
			// Restore the controller that is current now, not the one captured
			// before this open: an older open may have completed while this
			// attempt was in flight. If there is none, clear the selection but
			// keep the catalog's connection state.
			restore := a.view()
			a.update(func(state *dimui.State) {
				if !a.selectionCurrent(gen) {
					return
				}
				if restore != nil {
					state.Selected = restore.Selected
					applyView(state, restore)
				} else {
					connection, connectionError := state.Connection, state.ConnectionError
					applyView(state, &dimui.State{})
					state.Connection, state.ConnectionError = connection, connectionError
				}
				state.ConversationLoading = false
				state.ActionError = fmt.Sprintf("Open session %q: %v", id, err)
				state.ActionNotices = appendActionNotice(state.ActionNotices,
					dimui.Notice{Kind: "select.error", ID: state.ActionError})
			})
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		view := a.view()
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			applyView(state, view)
			state.Selected = row.ID
			state.Connection = dimui.ConnReady
			state.ConnectionError = ""
			state.ActionError = ""
			state.ActionStatus = ""
			state.ConversationLoading = false
		})
	}()
}

// deleteSession deletes one session through the App Server. The catalog row is
// removed only after the daemon confirmed; a failure keeps the row and reports
// through the banner. Deleting the displayed session closes its controller and
// selects the next remaining row, or clears the selection when none is left.
func (a *app) deleteSession(id string) {
	if a.bridge == nil || a.state == nil || id == "" {
		return
	}
	gen := a.currentSelection()
	go func() {
		a.selectMu.Lock()
		defer a.selectMu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := a.bridge.DeleteFor(ctx, id)
		if err != nil {
			// Deleting a session closes its stream, so the reply can be lost to
			// the very close it caused (§13 unknown result). The contract says to
			// read the catalog back, not to resend: a row the catalog no longer
			// lists is a confirmed delete.
			confirmed := false
			if errors.Is(err, appserver.ErrUnknownResult) {
				if rows, catalogErr := a.bridge.catalog(ctx); catalogErr == nil {
					confirmed = true
					for _, row := range rows {
						if row.ID == id {
							confirmed = false
							break
						}
					}
				}
			}
			if !confirmed {
				log.Printf("dimagent: delete session %q: %v", id, err)
				a.update(func(state *dimui.State) {
					if !a.selectionCurrent(gen) {
						return
					}
					state.ActionError = fmt.Sprintf("Delete session %q: %v", id, err)
					state.ActionNotices = appendActionNotice(state.ActionNotices,
						dimui.Notice{Kind: "delete.error", ID: state.ActionError})
				})
				return
			}
		}
		if !a.selectionCurrent(gen) {
			return
		}
		// Close before publishing the catalog change so observers that react to the
		// update cannot race the controller teardown.
		a.bridge.closeSessionFor(id)
		var next dimui.Session
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			remaining := make([]dimui.Session, 0, len(state.Sessions))
			for _, row := range state.Sessions {
				if row.ID == id {
					continue
				}
				remaining = append(remaining, row)
			}
			if state.Selected == id {
				state.Selected = ""
				state.SelectedBranch = ""
				state.Keys = dimui.KeysPanelView{}
				state.Messages = nil
				state.Branches = nil
			}
			state.Sessions = remaining
			if len(remaining) > 0 {
				next = remaining[0]
			}
		})
		if next.ID != "" {
			a.selectSession(next.ID)
		}
	}()
}

// closeSessionFor closes the controller of sessionID when it is the current
// one. It detaches that session's stream and leaves the supervisor running.
func (b *sessionBridge) closeSessionFor(sessionID string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	controller := b.controller
	if controller == nil || b.summary.SessionID != sessionID {
		b.mu.Unlock()
		return
	}
	b.controller = nil
	b.summary = appserver.SessionSummary{}
	b.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = controller.Close(ctx)
}

// selectBranch opens one branch of the selected session. Branch switching has
// no RPC of its own (docs/branch-attachment-plan.md §1.2): the controller reads
// that branch's first-screen history through session.history, and the window
// moves the displayed branch only after the App Server answered. A failed read
// keeps the current branch and reports through ActionError; a session selection
// or branch choice that changed while the read was in flight is dropped.
func (a *app) selectBranch(branchID string) {
	if a.bridge == nil || a.state == nil {
		return
	}
	if branchID == "" || branchID == a.state.SelectedBranch {
		return
	}
	sessionID := a.state.Selected
	gen := a.currentSelection()
	branchGen := a.beginBranchSelect()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.bridge.OpenBranchFor(ctx, sessionID, branchID); err != nil {
			log.Printf("dimagent: open branch %q: %v", branchID, err)
			a.update(func(state *dimui.State) {
				if !a.selectionCurrent(gen) || !a.branchCurrent(branchGen) {
					return
				}
				state.ActionError = fmt.Sprintf("Open branch %q: %v", branchID, err)
				state.ActionNotices = appendActionNotice(state.ActionNotices,
					dimui.Notice{Kind: "branch.error", ID: state.ActionError})
			})
			return
		}
		if !a.selectionCurrent(gen) || !a.branchCurrent(branchGen) {
			return
		}
		view := a.bridge.LiveStateFor(sessionID, branchID)
		if view == nil {
			a.update(func(state *dimui.State) {
				if !a.selectionCurrent(gen) || !a.branchCurrent(branchGen) {
					return
				}
				state.ActionError = fmt.Sprintf("Open branch %q: the session is no longer attached", branchID)
				state.ActionNotices = appendActionNotice(state.ActionNotices,
					dimui.Notice{Kind: "branch.error", ID: state.ActionError})
			})
			return
		}
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			// The view may fall back to the session's default branch when the
			// directory no longer proves this one; the displayed branch follows
			// what the controller returned, never the request.
			if !a.commitBranch(view.SelectedBranch, branchGen) {
				return
			}
			applyView(state, view)
			state.ActionError = ""
			state.ActionStatus = ""
		})
	}()
}

// endBranch asks the App Server to end one non-main branch, then re-reads the
// branch directory and the displayed view. The entry is not removed here or
// before the App Server answered: the directory keeps its rows until
// session.branches reports the end, and an already-ended reply is followed by
// the same read instead of a local delete. When the ended branch was the one
// displayed, the window falls back to main, so it never keeps showing a line the
// App Server closed. A failed end leaves the directory untouched and reports
// through ActionError.
func (a *app) endBranch(branchID string) {
	if a.bridge == nil || a.state == nil {
		return
	}
	if branchID == "" || branchID == appserver.MainBranchID {
		a.notice("branch.error", fmt.Sprintf("End branch %q: main cannot be ended", branchID))
		return
	}
	sessionID := a.state.Selected
	gen := a.currentSelection()
	_, branchGen := a.displayBranch()
	endedShown := a.state.SelectedBranch == branchID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := a.bridge.EndBranchFor(ctx, sessionID, branchID); err != nil {
			log.Printf("dimagent: end branch %q: %v", branchID, err)
			if a.selectionCurrent(gen) {
				a.notice("branch.error", fmt.Sprintf("End branch %q: %v", branchID, err))
			}
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		// The App Server accepted the end; re-read the directory so the row
		// leaves only when session.branches says it is gone.
		refreshErr := a.bridge.RefreshBranchesFor(ctx, sessionID)
		if refreshErr != nil {
			log.Printf("dimagent: refresh branches after ending %q: %v", branchID, refreshErr)
		}
		if !a.selectionCurrent(gen) {
			return
		}
		target, _ := a.displayBranch()
		if endedShown {
			target = appserver.MainBranchID
		}
		view := a.bridge.LiveStateFor(sessionID, target)
		if view == nil {
			a.update(func(state *dimui.State) {
				if !a.selectionCurrent(gen) {
					return
				}
				if endedShown && !a.commitBranch(appserver.MainBranchID, branchGen) {
					return
				}
				if endedShown {
					state.SelectedBranch = appserver.MainBranchID
				}
				state.ActionError = fmt.Sprintf("End branch %q: the window could not read the branch directory back", branchID)
				state.ActionNotices = appendActionNotice(state.ActionNotices,
					dimui.Notice{Kind: "branch.error", ID: state.ActionError})
			})
			return
		}
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			if endedShown {
				if !a.commitBranch(appserver.MainBranchID, branchGen) {
					return
				}
			} else if !a.branchCurrent(branchGen) {
				return
			}
			applyView(state, view)
			if endedShown {
				state.SelectedBranch = appserver.MainBranchID
			}
			state.ActionStatus = ""
		})
		if refreshErr != nil {
			a.notice("branch.error", fmt.Sprintf("Branch %q ended, but the branch directory refresh failed: %v", branchID, refreshErr))
			return
		}
		a.notice("branch.ended", branchID)
	}()
}

// refreshBranches re-reads session.branches and publishes the view the
// controller builds from the new directory. The window keeps the previous rows
// until that read returns; a failure reports through ActionError and deletes
// nothing.
func (a *app) refreshBranches() {
	if a.bridge == nil || a.state == nil {
		return
	}
	sessionID := a.state.Selected
	gen := a.currentSelection()
	_, branchGen := a.displayBranch()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.bridge.RefreshBranchesFor(ctx, sessionID); err != nil {
			log.Printf("dimagent: refresh branches: %v", err)
			if a.selectionCurrent(gen) {
				a.notice("branch.error", fmt.Sprintf("Refresh branches: %v", err))
			}
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		branch, _ := a.displayBranch()
		view := a.bridge.LiveStateFor(sessionID, branch)
		if view == nil {
			if a.selectionCurrent(gen) {
				a.notice("branch.error", "Refresh branches: the session is no longer attached")
			}
			return
		}
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) || !a.branchCurrent(branchGen) {
				return
			}
			applyView(state, view)
		})
	}()
}

// createSession asks the App Server for a new session, then republishes the
// catalog session.create returned and selects the new session. Nothing is added
// before the App Server answered: a rejected creation leaves the catalog and the
// selection as they were and only records the error.
func (a *app) createSession() {
	if a.state == nil {
		return
	}
	if a.bridge == nil {
		// Without a supervisor there is nothing to ask; say so instead of
		// leaving the click without an answer.
		a.notice("create.error", "Create session: no App Server is configured; set DIMAGENT_PRODUCT_HOME.")
		return
	}
	if a.state.CreatingSession {
		return
	}
	params, err := a.createParamsFor()
	if err != nil {
		log.Printf("dimagent: create session: %v", err)
		a.notice("create.error", fmt.Sprintf("Create session: %v", err))
		return
	}
	// The creation's result must not steal a selection the user made while it
	// was in flight; the same generation guard protects every other action.
	gen := a.currentSelection()
	a.state.CreatingSession = true
	go func() {
		a.selectMu.Lock()
		defer a.selectMu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		row, createErr := a.bridge.create(ctx, params)
		if createErr != nil {
			log.Printf("dimagent: create session: %v", createErr)
			a.update(func(state *dimui.State) {
				state.CreatingSession = false
				state.ActionStatus = ""
				if !a.selectionCurrent(gen) {
					return
				}
				state.ActionError = fmt.Sprintf("Create session: %v", createErr)
				state.ActionNotices = appendActionNotice(state.ActionNotices,
					dimui.Notice{Kind: "create.error", ID: state.ActionError})
			})
			return
		}
		rows, catalogErr := a.bridge.catalog(ctx)
		// A new session has no chosen branch: read its default view, and reset the
		// window's branch choice before folding it in.
		view := a.bridge.LiveState()
		a.update(func(state *dimui.State) {
			state.CreatingSession = false
			if !a.selectionCurrent(gen) {
				return
			}
			if catalogErr != nil {
				// The session exists; only the catalog read failed, so keep the
				// window's rows and add the created one instead of showing a session
				// the sidebar has no place for.
				state.Sessions = withCreatedRow(state.Sessions, row)
			} else {
				state.Sessions = withCreatedRow(rows, row)
			}
			// The new controller's view belongs to this session, so make it the
			// selection before folding the view in: applyView discards a view for
			// any other session, and the window still shows the previous one here.
			// A new session has no chosen branch, so drop the previous session's.
			state.Selected = row.ID
			a.resetBranch()
			applyView(state, view)
			state.ConversationLoading = false
			state.ActionStatus = ""
			if catalogErr != nil {
				state.ActionError = fmt.Sprintf("Session %s created, but the session catalog refresh failed: %v", row.ID, catalogErr)
				state.ActionNotices = appendActionNotice(state.ActionNotices,
					dimui.Notice{Kind: "catalog.error", ID: state.ActionError})
				return
			}
			state.ActionError = ""
			state.ActionNotices = appendActionNotice(state.ActionNotices,
				dimui.Notice{Kind: "create.accepted", ID: row.ID})
		})
	}()
}

// createParamsFor builds session.create's request, letting a test supply one.
func (a *app) createParamsFor() (appserver.CreateParams, error) {
	if a.createParams != nil {
		return a.createParams()
	}
	return defaultCreateParams()
}

// defaultCreateParams asks the App Server to create a session in the directory
// this process runs in, which is the only path the window knows without reading
// daemon state. No model or permission preset is sent: those are the App
// Server's to choose when the request omits them.
func defaultCreateParams() (appserver.CreateParams, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return appserver.CreateParams{}, fmt.Errorf("working directory: %w", err)
	}
	return appserver.CreateParams{Cwd: cwd}, nil
}

// withCreatedRow puts a session.create result into the window's catalog. The id
// is the App Server's own reply, so the row is a fact even while the catalog
// read has not caught up with it; the new session goes first, where the catalog
// puts its newest row.
func withCreatedRow(rows []dimui.Session, row dimui.Session) []dimui.Session {
	for _, existing := range rows {
		if existing.ID == row.ID {
			return rows
		}
	}
	return append([]dimui.Session{row}, rows...)
}

// submit sends the composer's text in the background and clears the draft only
// after the App Server accepted it. An ambiguous result is read back through
// session.input_lookup: it is never resent and never reported as accepted.
func (a *app) submit(text string) {
	if a.bridge == nil || a.state == nil {
		return
	}
	sessionID := a.state.Selected
	// A submission goes to the branch the window shows: the App Server opened
	// that branch, so it is the conversation the user is reading. With no branch
	// of its own the window shows the session's default branch, which is what
	// the branch-less submit names. The same branch names the read-back.
	branchID := a.actionBranch()
	gen := a.currentSelection()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, submissionID, err := a.bridge.submitOn(ctx, sessionID, branchID, text)
		if err != nil {
			log.Printf("dimagent: submit: %v", err)
			if errors.Is(err, appserver.ErrUnknownResult) {
				a.resolveUnknownSubmit(sessionID, branchID, submissionID, text, err, gen)
				return
			}
			if a.selectionCurrent(gen) {
				a.notice("submit.error", err.Error())
			}
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		view := a.view()
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			applyView(state, view)
			state.Draft = ""
			state.ActionStatus = ""
		})
		if a.selectionCurrent(gen) {
			a.notice("submit.accepted", "")
		}
	}()
}

// resolveUnknownSubmit reads an ambiguous session.submit back by its submission
// id. The App Server's answer alone decides what the window says: "accepted"
// means the daemon holds the input, anything else stays unknown, the draft is
// kept and the submission is not sent again.
func (a *app) resolveUnknownSubmit(sessionID, branchID, submissionID, text string, submitErr error, gen uint64) {
	var (
		result  appserver.InputLookupResult
		lookupE error
	)
	switch {
	case submissionID == "":
		lookupE = errors.New("the App Server returned no submission id to read back")
	case branchID == "":
		lookupE = errors.New("no branch is known for the submission")
	default:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// The read-back names the branch the submission was sent on, so an
		// unknown side-branch result is never resolved against the default one.
		result, lookupE = a.bridge.LookupInputBranchFor(ctx, sessionID, branchID, submissionID)
	}
	if lookupE != nil {
		detail := fmt.Sprintf("Submission result unknown (%v); read-back failed: %v", submitErr, lookupE)
		a.unknownOutcome(gen, detail)
		return
	}
	if !result.Accepted() {
		a.unknownOutcome(gen, "Submission result unknown: the App Server does not hold this submission.")
		return
	}
	view := a.view()
	a.update(func(state *dimui.State) {
		if !a.selectionCurrent(gen) {
			return
		}
		applyView(state, view)
		// The daemon holds this submission, so the draft it came from is sent.
		// Text typed after the send stays.
		if strings.TrimSpace(state.Draft) == text {
			state.Draft = ""
		}
		state.ActionError = ""
		state.ActionStatus = "Submission accepted"
		state.ActionNotices = appendActionNotice(state.ActionNotices,
			dimui.Notice{Kind: "submit.accepted", ID: submissionID})
	})
}

// unknownOutcome publishes an unresolved action: it keeps the alert and the
// notice kind so an unknown result can never read as success.
func (a *app) unknownOutcome(gen uint64, detail string) {
	a.update(func(state *dimui.State) {
		if !a.selectionCurrent(gen) {
			return
		}
		state.ActionNotices = appendActionNotice(state.ActionNotices,
			dimui.Notice{Kind: "submit.unknown", ID: detail})
		state.ActionError = detail
		state.ActionStatus = "Submission result unknown"
	})
}

// reconnect re-attaches the selected session after a transport loss. The
// controller rebuilds its projection from authoritative App Server history;
// this callback only coordinates the UI and never invents local facts.
func (a *app) reconnect() {
	if a.bridge == nil || a.state == nil {
		return
	}
	sessionID := a.state.Selected
	gen := a.currentSelection()
	a.update(func(state *dimui.State) {
		if !a.selectionCurrent(gen) {
			return
		}
		state.ConversationLoading = true
		state.ConnectionError = ""
	})
	go func() {
		// Reattach swaps the bridge controller, so serialize it with session
		// selection; otherwise a selection can close a freshly rebuilt stream.
		a.selectMu.Lock()
		defer a.selectMu.Unlock()
		if !a.selectionCurrent(gen) {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// Reconnect the control plane first: a daemon this window owns is
		// respawned here when its socket is gone, and every stale attachment is
		// closed. Only then can the session's own stream be rebuilt.
		if a.bridge.sup != nil {
			if _, err := a.bridge.sup.Reconnect(ctx); err != nil {
				if !a.selectionCurrent(gen) {
					return
				}
				log.Printf("dimagent: reconnect: %v", err)
				a.update(func(state *dimui.State) {
					if !a.selectionCurrent(gen) {
						return
					}
					state.ConversationLoading = false
					state.Connection = dimui.ConnError
					state.ConnectionError = fmt.Sprintf("%v", err)
				})
				return
			}
		}
		if err := a.bridge.reattachFor(ctx, sessionID); err != nil {
			if !a.selectionCurrent(gen) {
				return
			}
			log.Printf("dimagent: reconnect: %v", err)
			a.update(func(state *dimui.State) {
				if !a.selectionCurrent(gen) {
					return
				}
				// Another attempt may have healed the connection while this one
				// was in flight; do not turn a ready session back into an error.
				if state.Connection == dimui.ConnReady {
					state.ConversationLoading = false
					return
				}
				state.ConversationLoading = false
				state.Connection = dimui.ConnError
				state.ConnectionError = fmt.Sprintf("%v", err)
			})
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		view := a.view()
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			applyView(state, view)
			state.ActionError = ""
			state.ConversationLoading = false
		})
	}()
}

// stop asks the App Server to stop the running turn; the run state converges
// through the session stream rather than a local guess.
func (a *app) stop() {
	if a.bridge == nil || a.state == nil {
		return
	}
	sessionID := a.state.Selected
	// A stop names the branch the window shows, so stopping on a side branch
	// ends that branch's turn instead of the session's default one.
	branchID := a.actionBranch()
	gen := a.currentSelection()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.bridge.stopOn(ctx, sessionID, branchID); err != nil {
			log.Printf("dimagent: stop: %v", err)
			if a.selectionCurrent(gen) {
				a.notice("stop.error", err.Error())
			}
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		view := a.view()
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			applyView(state, view)
		})
		if a.selectionCurrent(gen) {
			a.notice("stop.accepted", "")
		}
	}()
}

// respondApproval sends one decision for one pending approval through the
// controller that still belongs to the selected session. The shell keeps the
// row: only the session projection's approval.resolved or approval.cancelled
// removes it. A failure is published as an error and the row stays available,
// so nothing here reports a success the App Server did not return.
func (a *app) respondApproval(branchID, requestID string, decision dimui.ApprovalDecision) {
	if a.bridge == nil || a.state == nil {
		return
	}
	wireDecision, ok := approvalDecision(decision)
	if !ok {
		a.notice("approval.error", fmt.Sprintf("unsupported approval decision %q", decision))
		return
	}
	if branchID == "" || requestID == "" {
		a.notice("approval.error", "approval request is missing its branch or request id")
		return
	}
	sessionID := a.state.Selected
	gen := a.currentSelection()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		outcome, err := a.bridge.RespondApprovalFor(ctx, sessionID, branchID, requestID, wireDecision)
		if err != nil {
			log.Printf("dimagent: approval.respond: %v", err)
			if a.selectionCurrent(gen) {
				a.notice("approval.error", err.Error())
			}
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		view := a.view()
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			applyView(state, view)
		})
		if outcome == appserver.ApprovalOutcomeCancelled {
			// The App Server no longer holds this approval; the decision was not
			// applied and must not read as one.
			a.notice("approval.cancelled", "The approval was already resolved on the App Server.")
			return
		}
		a.notice("approval.responded", "")
	}()
}

// approvalDecision maps the shell's decision onto approval.respond's wire value.
// An unknown decision is rejected before anything is sent.
func approvalDecision(decision dimui.ApprovalDecision) (string, bool) {
	switch decision {
	case dimui.ApprovalAllow:
		return appserver.DecisionAllow, true
	case dimui.ApprovalDeny:
		return appserver.DecisionDeny, true
	default:
		return "", false
	}
}

// envInt reads an optional integer environment value; an absent or malformed
// one is zero, which mygo reads as "center the window".
func envInt(name string) int {
	value, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		return 0
	}
	return value
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	state, bridge, err := stateFromDaemon(ctx, os.Getenv("DIMAGENT_PRODUCT_HOME"))
	cancel()
	if err != nil {
		state = &dimui.State{Connection: dimui.ConnError, ConnectionError: fmt.Sprintf("%v", err)}
		log.Printf("dimagent: %v", err)
	}
	application := &app{state: state, bridge: bridge}
	application.bind()
	// The appearance preferences are the desktop's own and load even without a
	// daemon: they decide how the shell draws, not what a session contains.
	application.loadAppearance()
	if bridge != nil {
		defer func() {
			application.stopRefresh()
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = bridge.Close(closeCtx)
		}()
	}
	mygo.App.WhenReady(func() {
		application.win = mygo.NewWindow(mygo.WindowOptions{
			Title: "DimAgent",
			// The reference window opens at 800×720 points; the sidebar's 256 DIP
			// is the same, so the proportion of chrome matches the desktop app.
			Width:  800,
			Height: 720,
			// A capture names its position so the window lands on the primary
			// display; a normal launch leaves both at zero and centers itself.
			X: envInt("DIMAGENT_WINDOW_X"),
			Y: envInt("DIMAGENT_WINDOW_Y"),
			// content draws the shell and then writes an appearance choice the
			// surface applied during that frame.
			Content: ui.View(application.content),
		})
		if application.bridge != nil {
			application.startRefresh(250 * time.Millisecond)
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
