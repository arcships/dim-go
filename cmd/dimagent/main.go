// Command dimagent is the MyGo native UI entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"

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
	// BranchCreate asks the App Server to create a side branch from a parent
	// sequence. The returned branch is the App Server's; the window opens it
	// before showing it and never invents a local branch.
	BranchCreate(ctx context.Context, parentBranchID string, at int64, prompt string) (appserver.BranchCreateResult, error)
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
	// Rename sets or clears the selected session's manual title through
	// session.rename. A nil title clears it; the title the daemon committed is
	// authoritative.
	Rename(ctx context.Context, title *string) (appserver.RenameResult, error)
	// AttachmentPut stores one UTF-8 text attachment through attachment.put and
	// returns the App Server's stored path. It is sent once; there is no
	// read-back method for an unknown result.
	AttachmentPut(ctx context.Context, name, text string) (appserver.AttachmentPutResult, error)
	// Title returns the newest title metadata the session observed, ordered
	// against the catalog summary by revision.
	Title(summary appserver.SessionSummary) session.TitleState
	// Models reads the App Server model catalog and stored default for the
	// attached session. It sends no mutation and keeps no local catalog.
	Models(ctx context.Context) (session.ModelCatalog, error)
	// Model is the provider/model pair the attach reply reported, if any.
	Model() (session.ModelSelection, bool)
	// UpdateModel switches the session's main-branch model through
	// session.model.update. The caller never resends an unknown result.
	UpdateModel(ctx context.Context, providerID, modelID string) (appserver.ModelUpdateResult, error)
	// RefreshModel reads the current model pair back from session.summaries.
	RefreshModel(ctx context.Context) (session.ModelSelection, bool, error)
	// CanUpdateModel reports whether the App Server advertises
	// session.model.update, so the window can keep the picker WIP.
	CanUpdateModel() bool
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

// BranchFromFor creates a side branch through the controller that still belongs
// to sessionID. The reply is the App Server's branch; nothing is cached or
// invented here.
func (b *sessionBridge) BranchFromFor(ctx context.Context, sessionID, parentBranchID string, at int64, prompt string) (appserver.BranchCreateResult, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return appserver.BranchCreateResult{}, err
	}
	return controller.BranchCreate(ctx, parentBranchID, at, prompt)
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

// RenameFor sets or clears the title through the controller that still belongs
// to sessionID. The bridge changes no local title: the window refreshes the
// catalog and lets the daemon's reply order the title.
func (b *sessionBridge) RenameFor(ctx context.Context, sessionID string, title *string) (appserver.RenameResult, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return appserver.RenameResult{}, err
	}
	return controller.Rename(ctx, title)
}

// AttachmentPutFor stores a text attachment through the controller that still
// belongs to sessionID. The bridge adds no local file or path: the returned path
// is the App Server's, and an unknown result is passed through unchanged.
func (b *sessionBridge) AttachmentPutFor(ctx context.Context, sessionID, name, text string) (appserver.AttachmentPutResult, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return appserver.AttachmentPutResult{}, err
	}
	return controller.AttachmentPut(ctx, name, text)
}

// ModelsFor reads the model catalog through the controller that still belongs
// to sessionID. The read is non-mutating and returns the daemon's current
// answer; a missing capability is an error the window shows in place of a model.
func (b *sessionBridge) ModelsFor(ctx context.Context, sessionID string) (session.ModelCatalog, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return session.ModelCatalog{}, err
	}
	return controller.Models(ctx)
}

// ModelFor reports the model pair the attached session reported.
func (b *sessionBridge) ModelFor(sessionID string) (session.ModelSelection, bool) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return session.ModelSelection{}, false
	}
	return controller.Model()
}

// UpdateModelFor writes the model switch through the controller that still
// belongs to sessionID. It never retries: an unknown result is returned so the
// window can read the pair back instead.
func (b *sessionBridge) UpdateModelFor(ctx context.Context, sessionID, providerID, modelID string) (appserver.ModelUpdateResult, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return appserver.ModelUpdateResult{}, err
	}
	return controller.UpdateModel(ctx, providerID, modelID)
}

// RefreshModelFor reads the current model pair back through the controller that
// still belongs to sessionID.
func (b *sessionBridge) RefreshModelFor(ctx context.Context, sessionID string) (session.ModelSelection, bool, error) {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return session.ModelSelection{}, false, err
	}
	return controller.RefreshModel(ctx)
}

// CanUpdateModelFor reports whether the controller that belongs to sessionID
// advertises session.model.update. It sends no request.
func (b *sessionBridge) CanUpdateModelFor(sessionID string) bool {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return false
	}
	return controller.CanUpdateModel()
}

// TitleFor merges the session's live title metadata against the catalog
// summary. The window uses it after a rename so a committed title shows without
// waiting for the next catalog read.
func (b *sessionBridge) TitleFor(sessionID string, summary appserver.SessionSummary) session.TitleState {
	controller, err := b.controllerFor(sessionID)
	if err != nil {
		return session.TitleState{}
	}
	return controller.Title(summary)
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
	// An untitled session has no title fact yet: the shell draws its localized
	// placeholder until the catalog reports a title or preview.
	row := dimui.Session{ID: result.SessionID}
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
	state.ModelCatalog, state.ModelStatus = readSessionModels(ctx, bridge, rows[0].ID)
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
	return supervisor.New(supervisorConfig(productHome))
}

// supervisorConfig builds the daemon config from the environment. A model
// config path is handed to the spawned daemon as an argument only; this
// process never reads the file, so a real-model acceptance run stays the
// daemon's own concern.
func supervisorConfig(productHome string) supervisor.Config {
	return supervisor.Config{
		ProductHome: productHome,
		// The mock model is the daemon's own flag for acceptance runs: it echoes
		// turns without a provider. It is opt-in through the environment so a
		// normal launch never hides a missing model.
		MockModel: os.Getenv("DIMAGENT_MOCK_MODEL") == "1",
		// The model config is the daemon's own flag for real-model acceptance
		// runs. It is opt-in through the environment; an empty value keeps the
		// daemon's default resolution.
		ModelConfigPath: strings.TrimSpace(os.Getenv("DIMAGENT_MODEL_CONFIG")),
	}
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
	// lookupRetries, lookupInterval and lookupBudget override the read-back
	// retry policy; zero values use the submitLookup* defaults. Tests shrink the
	// interval so a retry sequence runs in milliseconds.
	lookupRetries  int
	lookupInterval time.Duration
	lookupBudget   time.Duration
	// selectionCtx is cancelled when the selection moves or its session closes,
	// so a read-back tied to the old selection stops waiting and never publishes
	// into the new one.
	selectionCtx    context.Context
	selectionCancel context.CancelFunc
	// selectMu keeps session opens in order: a newer choice waits for the
	// attachment in flight, then replaces it. A creation installs a controller
	// the same way, so it takes the same lock.
	selectMu sync.Mutex
	// submitMu serializes the window's submit entry point. A second trigger while
	// the first request (including unknown-result read-back) is in flight is
	// ignored, preventing duplicate runs from repeated UI events.
	submitMu       sync.Mutex
	submitInFlight bool
	// Pending end requests are UI concurrency guards, not branch facts. Scope
	// them to the target identity across selection changes; generation only
	// gates publication. Other sessions and branches remain independent.
	endBranchMu       sync.Mutex
	endBranchInFlight map[branchEndFlight]struct{}
	// A pending approval decision excludes conflicting decisions for the same
	// daemon request, independent of selection generation and decision value.
	approvalMu       sync.Mutex
	approvalInFlight map[approvalFlight]struct{}
	// Deduplicate only identical concurrent branch-create intent, not different
	// prompts or cut points. These are transient request guards, not branches.
	branchCreateMu       sync.Mutex
	branchCreateInFlight map[branchCreateFlight]struct{}
	// createParams builds session.create's request. Tests replace it; the app
	// uses defaultCreateParams.
	createParams func() (appserver.CreateParams, error)
	// attachmentPicker opens the native text-file picker. Tests replace it; the
	// app uses the MyGo dialog attached to the window.
	attachmentPicker func() (string, error)
	// appearance is the desktop-local appearance preference store: the values
	// loaded before the first frame and the snapshot the window writes when the
	// settings surface reports a choice. See settings.go.
	appearance appearance
	// desktop is the desktop-local state file: appearance preferences, window
	// geometry and drafts. It is not session truth; see desktop.go.
	desktop *desktopStore
	// window is the window geometry bookkeeping; see window.go.
	window windowBookkeeping
}

func (a *app) bind() {
	a.state.OnSelect = a.selectSession
	a.state.OnDeleteSession = a.deleteSession
	a.state.OnPinSession = a.togglePin
	a.state.OnArchiveSession = a.toggleArchive
	a.state.OnRestoreSession = a.restoreArchived
	a.state.OnRename = a.renameSession
	a.state.OnSelectModel = a.selectModel
	a.state.OnSelectBranch = a.selectBranch
	a.state.OnBranchFrom = a.branchFrom
	a.state.OnEndBranch = a.endBranch
	a.state.OnRefreshBranches = a.refreshBranches
	a.state.OnCreateSession = a.createSession
	a.state.OnSubmit = a.submit
	a.state.OnAttachFile = a.attachFile
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
// flight belongs to that session and must not land on the new one, and it stops
// any read-back waiting on the replaced selection.
func (a *app) beginSelect() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.selectGen++
	a.branchGen++
	a.branch = ""
	a.resetSelectionContext()
	return a.selectGen
}

// resetSelectionContext cancels work tied to the previous selection and gives
// the current one a fresh context. The caller holds a.mu.
func (a *app) resetSelectionContext() {
	if a.selectionCancel != nil {
		a.selectionCancel()
	}
	a.selectionCtx, a.selectionCancel = context.WithCancel(context.Background())
}

// selectionContext returns the context cancelled when the current selection
// moves or closes. A read-back waits on it so a session switch stops the retry
// instead of publishing the old submission's outcome on the new session.
func (a *app) selectionContext() context.Context {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.selectionCtx == nil {
		a.selectionCtx, a.selectionCancel = context.WithCancel(context.Background())
	}
	return a.selectionCtx
}

// closeSelection stops background work tied to the current selection without
// moving it. A closed session has nothing left to resolve, so its pending
// read-back must stop waiting for a late App Server answer.
func (a *app) closeSelection() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.resetSelectionContext()
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
		// The model catalog is a remote, non-mutating read. It is refreshed on a
		// slower cadence than the in-memory projection so the window follows a
		// provider change without asking provider.management on every tick.
		const modelRefresh = 2 * time.Second
		lastModels := time.Time{}
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
				var (
					modelCatalog dimui.ModelCatalog
					modelStatus  dimui.ModelStatus
					readModels   bool
				)
				if view.Selected != "" && time.Since(lastModels) >= modelRefresh {
					lastModels = time.Now()
					modelCatalog, modelStatus = a.readModels(ctx, view.Selected)
					readModels = true
				}
				a.update(func(state *dimui.State) {
					// A selection or branch choice may complete while the view is
					// rebuilding; do not let that older snapshot overwrite the new
					// conversation.
					if !a.selectionCurrent(gen) || !a.branchCurrent(branchGen) {
						return
					}
					applyView(state, view)
					if readModels {
						state.ModelCatalog = modelCatalog
						state.ModelStatus = modelStatus
					}
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
	// The draft belongs to the session the window is leaving. Save it before the
	// selection moves, so switching back restores it instead of showing the new
	// session's text.
	outgoing := a.state.Selected
	if outgoing != id {
		a.rememberDraft(outgoing)
	}
	gen := a.beginSelect()
	a.state.ConversationLoading = true
	// The model facts belong to the session being left. Drop them while the new
	// attachment is in flight so the chip never shows one session's model under
	// another; the read below fills them once the App Server answered.
	a.state.ModelCatalog = dimui.ModelCatalog{}
	a.state.ModelStatus = dimui.ModelStatus{}
	a.state.ModelListOpen = false
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
			var (
				modelCatalog dimui.ModelCatalog
				modelStatus  dimui.ModelStatus
			)
			if restore != nil && restore.Selected != "" {
				modelCatalog, modelStatus = a.readModels(ctx, restore.Selected)
			}
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
				state.ModelCatalog = modelCatalog
				state.ModelStatus = modelStatus
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
		modelCatalog, modelStatus := a.readModels(ctx, row.ID)
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			// The new selection is published before the view is folded in, so
			// applyView cannot discard the view of the session just opened when the
			// caller (the search palette, for example) did not write the sidebar's
			// choice first. The terminal state of the session being left is not
			// carried over: the new controller's view reports whether it is closed.
			state.Selected = row.ID
			state.SessionClosed = false
			state.SessionClosedReason = ""
			state.ClosedSessionTitle = ""
			state.ClosedSessionProject = ""
			applyView(state, view)
			state.Selected = row.ID
			state.ModelCatalog = modelCatalog
			state.ModelStatus = modelStatus
			if row.ID != outgoing {
				// Restore the draft saved for the session the window now shows.
				state.Draft = a.draftFor(row.ID)
			}
			state.Connection = dimui.ConnReady
			state.ConnectionError = ""
			state.ActionError = ""
			state.ActionStatus = ""
			state.ConversationLoading = false
		})
	}()
}

// closedReasonDeleted is the reason the main area shows for a session whose
// session.delete the App Server confirmed. The row leaves the catalog, so the
// window keeps the confirmed outcome as terminal display state instead of
// inventing a session fact.
const closedReasonDeleted = "Deleted"

// deleteSession deletes one session through the App Server. The catalog row is
// removed only after the daemon confirmed; a failure keeps the row and reports
// through the banner. Deleting the displayed session closes its controller but
// keeps the selection and its rows on screen as the terminal closed state until
// the user chooses another session or creates one; the sidebar row is gone
// because the App Server confirmed the delete.
func (a *app) deleteSession(id string) {
	if a.bridge == nil || a.state == nil || id == "" {
		return
	}
	gen := a.currentSelection()
	selected := a.state.Selected
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
					state.ActionStatus = ""
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
		if selected == id {
			// The displayed session is closed, so a pending read-back has nothing
			// left to resolve and must not publish into the terminal state.
			a.closeSelection()
		}
		// The session is gone, so its desktop draft is too. This touches no
		// session fact: the App Server's confirmed delete is the fact.
		a.clearDraft(id)
		// The deleted id no longer names a row, so the desktop's pinned and
		// archived display lists must not keep it.
		a.forgetHiddenSession(id)
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			remaining := make([]dimui.Session, 0, len(state.Sessions))
			var deleted dimui.Session
			for _, row := range state.Sessions {
				if row.ID == id {
					deleted = row
					continue
				}
				remaining = append(remaining, row)
			}
			state.Sessions = remaining
			if state.Selected != id {
				return
			}
			// The confirmed delete leaves the main area in the terminal closed
			// state instead of switching to another row: the selection stays, so
			// the conversation and the closed banner remain until the user chooses
			// another session or creates one. The deleted session owns no branch,
			// draft, keys, model or run any more; its rows stay as history, and its
			// catalog labels stay for the toolbar to name it.
			state.ClosedSessionTitle = deleted.Title
			state.ClosedSessionProject = deleted.Group
			state.SelectedBranch = ""
			state.Draft = ""
			state.Keys = dimui.KeysPanelView{}
			state.Branches = nil
			state.ModelCatalog = dimui.ModelCatalog{}
			state.ModelStatus = dimui.ModelStatus{}
			state.ModelListOpen = false
			state.Run = dimui.RunIdle
			state.SessionClosed = true
			state.SessionClosedReason = closedReasonDeleted
			state.ActionError = ""
			state.ActionStatus = ""
		})
	}()
}

// renameSession sets or clears the selected session's manual title through
// session.rename, then re-reads the catalog so the row shows the committed
// title. The request is sent once and never retried: an unsupported method
// (unknown_method) stays a visible WIP failure, and an unknown result is
// reported rather than guessed at. The title is never changed locally.
func (a *app) renameSession(title string) {
	if a.bridge == nil || a.state == nil {
		return
	}
	sessionID := a.state.Selected
	if sessionID == "" {
		a.notice("rename.error", "Rename session: no session is selected.")
		return
	}
	trimmed := strings.TrimSpace(title)
	var request *string
	if trimmed != "" {
		request = &trimmed
	}
	gen := a.currentSelection()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		result, err := a.bridge.RenameFor(ctx, sessionID, request)
		if err != nil {
			log.Printf("dimagent: rename session %q: %v", sessionID, err)
			if !a.selectionCurrent(gen) {
				return
			}
			message := fmt.Sprintf("Rename session: %v", err)
			if errors.Is(err, appserver.ErrCapabilityUnavailable) {
				// unknown_method: session.rename is absent upstream, so the
				// entry stays a WIP gap instead of looking like a bad title.
				message = fmt.Sprintf("Rename session: session.rename is unavailable on this App Server (unknown_method); the title was not changed. This entry stays WIP until the App Server offers the method. %v", err)
			}
			a.update(func(state *dimui.State) {
				if !a.selectionCurrent(gen) {
					return
				}
				state.ActionError = message
				state.ActionStatus = ""
				state.ActionNotices = appendActionNotice(state.ActionNotices,
					dimui.Notice{Kind: "rename.error", ID: message})
			})
			return
		}
		rows, catalogErr := a.bridge.catalog(ctx)
		// The daemon's reply orders the title against the catalog summary. Read
		// the merged metadata for the selected row so the committed revision
		// shows even when the list read raced the rename.
		titleKnown := false
		titleText := ""
		if catalogErr == nil {
			for _, row := range rows {
				if row.ID != sessionID {
					continue
				}
				state := a.bridge.TitleFor(sessionID, summaryForRow(row))
				if state.Known {
					titleKnown = true
					if state.Text != nil {
						titleText = *state.Text
					}
				}
				break
			}
		}
		outcome := "cleared"
		if result.Title != nil && *result.Title != "" {
			outcome = *result.Title
		}
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			if catalogErr != nil {
				state.ActionError = fmt.Sprintf("Session renamed, but the session catalog refresh failed: %v", catalogErr)
				state.ActionNotices = appendActionNotice(state.ActionNotices,
					dimui.Notice{Kind: "catalog.error", ID: state.ActionError})
				return
			}
			state.Sessions = rows
			if titleKnown {
				for i := range state.Sessions {
					if state.Sessions[i].ID == sessionID {
						state.Sessions[i].Title = titleText
						break
					}
				}
			}
			state.ActionError = ""
			state.ActionStatus = ""
			state.ActionNotices = appendActionNotice(state.ActionNotices,
				dimui.Notice{Kind: "rename.accepted", ID: outcome})
		})
	}()
}

// attachmentTextExtensions are the text file extensions the picker offers and
// the upload accepts, matching the Electron client's text attachment set. The
// picker filters on them, and a file chosen through another path is checked
// again before it is read.
var attachmentTextExtensions = []string{"txt", "md", "json", "yaml", "yml", "csv", "log"}

// attachmentPickerTimeout bounds one attachment.put request. The method has no
// idempotency key and no read-back, so the request is never resent after the
// context expires.
const attachmentPickerTimeout = 30 * time.Second

// attachFile opens the native text-file picker and stores the chosen file
// through attachment.put. The dialog blocks until it is dismissed, so it runs on
// its own goroutine; a cancelled picker changes nothing. The draft link is added
// only after the App Server returned a path, so a dismissed dialog or a failed
// upload leaves the draft untouched.
func (a *app) attachFile() {
	if a.bridge == nil || a.state == nil {
		return
	}
	sessionID := a.state.Selected
	if sessionID == "" {
		a.notice("attach.error", "Attach file: no session is selected.")
		return
	}
	gen := a.currentSelection()
	go func() {
		path, err := a.pickAttachment()
		if err != nil {
			log.Printf("dimagent: attachment picker: %v", err)
			if a.selectionCurrent(gen) {
				a.reportAttachmentFailure(gen, fmt.Sprintf("Attach file: the file picker failed: %v", err))
			}
			return
		}
		a.attachSelectedFile(gen, sessionID, path)
	}()
}

// pickAttachment returns the chosen file path, or "" when the picker was
// cancelled. Tests replace the dialog; the app attaches it to the window so it
// opens as a sheet where the platform supports one.
func (a *app) pickAttachment() (string, error) {
	if a.attachmentPicker != nil {
		return a.attachmentPicker()
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: a.win,
		Title:  "Attach text file",
		Filters: []mygo.FileFilter{{
			Name:       "Text files",
			Extensions: attachmentTextExtensions,
		}},
	})
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", nil
	}
	return paths[0], nil
}

// attachSelectedFile reads one chosen text file and sends it through
// attachment.put once. An empty path is a cancelled picker and sends nothing. A
// file that cannot be read, is too large, is not UTF-8, or is not one of the
// text extensions is reported and never uploaded.
func (a *app) attachSelectedFile(gen uint64, sessionID, path string) {
	if path == "" {
		return
	}
	name, text, err := readTextAttachment(path)
	if err != nil {
		log.Printf("dimagent: attach %q: %v", path, err)
		a.reportAttachmentFailure(gen, fmt.Sprintf("Attach file: %v", err))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), attachmentPickerTimeout)
	defer cancel()
	result, err := a.bridge.AttachmentPutFor(ctx, sessionID, name, text)
	if err != nil {
		log.Printf("dimagent: attach %q for %q: %v", path, sessionID, err)
		a.reportAttachmentFailure(gen, attachmentErrorMessage(err))
		return
	}
	a.update(func(state *dimui.State) {
		if !a.selectionCurrent(gen) {
			return
		}
		state.Draft = appendAttachmentLink(state.Draft, name, result.Path)
		state.ActionError = ""
		state.ActionStatus = "Attachment added"
	})
}

// reportAttachmentFailure publishes the upload or read failure as the window's
// action error and clears any older success line, so an attachment that was not
// stored never looks added.
func (a *app) reportAttachmentFailure(gen uint64, message string) {
	a.update(func(state *dimui.State) {
		if !a.selectionCurrent(gen) {
			return
		}
		state.ActionError = message
		state.ActionStatus = ""
		state.ActionNotices = appendActionNotice(state.ActionNotices,
			dimui.Notice{Kind: "attach.error", ID: message})
	})
}

// attachmentErrorMessage explains the three attachment.put outcomes the window
// can see. unknown_method means the host has no Attachments plugin, so the entry
// stays a visible WIP gap; an unknown result has no read-back method and is
// never resent; every other failure is reported as the App Server sent it.
func attachmentErrorMessage(err error) string {
	switch {
	case errors.Is(err, appserver.ErrCapabilityUnavailable):
		return fmt.Sprintf("Attach file: attachment.put is unavailable on this App Server (unknown_method); the file was not attached. This entry stays WIP until the App Server offers the method. %v", err)
	case errors.Is(err, appserver.ErrUnknownResult):
		return fmt.Sprintf("Attach file: the App Server did not confirm the upload (%v); attachment.put has no read-back method, so the request was not resent.", err)
	default:
		return fmt.Sprintf("Attach file: %v", err)
	}
}

// readTextAttachment reads one supported text file and returns its base name
// and UTF-8 text. The extension check runs before the read so a binary file the
// picker let through is rejected without loading it. The size and UTF-8 rules
// are attachment.put's own limits.
func readTextAttachment(path string) (string, string, error) {
	name := filepath.Base(path)
	if !isAttachmentTextFile(name) {
		return "", "", fmt.Errorf("%s is not a supported text file (%s); image attachments are WIP",
			name, strings.Join(attachmentTextExtensions, ", "))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read %s: %w", name, err)
	}
	if len(data) == 0 {
		return "", "", fmt.Errorf("%s is empty; attachment.put requires non-empty text", name)
	}
	if len(data) > appserver.MaxAttachmentTextBytes {
		return "", "", fmt.Errorf("%s is %d bytes, above the %d-byte text attachment limit",
			name, len(data), appserver.MaxAttachmentTextBytes)
	}
	if !utf8.Valid(data) {
		return "", "", fmt.Errorf("%s is not valid UTF-8 text; only UTF-8 text files can be attached", name)
	}
	return name, string(data), nil
}

// isAttachmentTextFile reports whether name carries one of the text extensions
// the attachment entry offers.
func isAttachmentTextFile(name string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	for _, allowed := range attachmentTextExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

// appendAttachmentLink adds the stored path to the draft as a Markdown link, as
// the Electron client does: a blank line separates it from existing text, a
// draft that already ends in a newline gets the link directly, and the label and
// destination escape the characters Markdown would otherwise read.
func appendAttachmentLink(draft, name, path string) string {
	label := strings.NewReplacer(`\`, `\\`, `[`, `\[`, `]`, `\]`).Replace(name)
	target := strings.NewReplacer(`\`, `\\`, `)`, `\)`).Replace(path)
	link := "[" + label + "](" + target + ")"
	if draft == "" || strings.HasSuffix(draft, "\n") {
		return draft + link
	}
	return draft + "\n\n" + link
}

// selectModel switches the selected session's model through session.model.update.
// The write is sent once and never retried. Success is published only from the
// App Server: RefreshModel reads the next session.summaries pair, and the reply's
// adopted pair is the fallback when that read fails. An unknown result is read
// back, never resent; a definitive failure keeps the previous chip and reports
// ActionError.
func (a *app) selectModel(providerID, modelID string) {
	if a.bridge == nil || a.state == nil {
		return
	}
	sessionID := a.state.Selected
	if sessionID == "" {
		a.notice("model.error", "Select model: no session is selected.")
		return
	}
	if providerID == "" || modelID == "" {
		a.notice("model.error", "Select model: the App Server did not name both a provider and a model.")
		return
	}
	gen := a.currentSelection()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		result, err := a.bridge.UpdateModelFor(ctx, sessionID, providerID, modelID)
		if err != nil {
			log.Printf("dimagent: select model %s/%s for %q: %v", providerID, modelID, sessionID, err)
			if !a.selectionCurrent(gen) {
				return
			}
			if errors.Is(err, appserver.ErrUnknownResult) {
				// The daemon may have applied the write. Resolve the doubt by
				// reading the current pair back; never resend the update.
				selection, ok, readErr := a.bridge.RefreshModelFor(ctx, sessionID)
				catalog, status := a.readModels(ctx, sessionID)
				a.update(func(state *dimui.State) {
					if !a.selectionCurrent(gen) {
						return
					}
					message := fmt.Sprintf("Select model: the App Server did not confirm the change (%v); the current model was read back from the App Server and the request was not resent.", err)
					if readErr != nil || !ok {
						message = fmt.Sprintf("Select model: the App Server did not confirm the change (%v) and reading the current model back failed (%v); the request was not resent.", err, readBackReason(readErr, ok))
					} else {
						state.ModelCatalog = catalog
						state.ModelStatus = status
						message = fmt.Sprintf("Select model: the App Server did not confirm the change (%v); the current model was read back as %s/%s and the request was not resent.", err, selection.ProviderID, selection.ModelID)
					}
					state.ActionError = message
					state.ActionStatus = ""
					state.ActionNotices = appendActionNotice(state.ActionNotices,
						dimui.Notice{Kind: "model.unknown", ID: message})
				})
				return
			}
			message := fmt.Sprintf("Select model: %v", err)
			if errors.Is(err, appserver.ErrCapabilityUnavailable) {
				// The Hello features or the session method table lacks
				// session.model.update: keep the entry WIP instead of pretending
				// the window can switch the model.
				message = fmt.Sprintf("Select model: session.model.update is unavailable on this App Server (unknown_method); the model was not changed. This entry stays WIP until the App Server offers the method. %v", err)
			}
			a.update(func(state *dimui.State) {
				if !a.selectionCurrent(gen) {
					return
				}
				state.ActionError = message
				state.ActionStatus = ""
				state.ActionNotices = appendActionNotice(state.ActionNotices,
					dimui.Notice{Kind: "model.error", ID: message})
			})
			return
		}
		// Success. The chip follows the App Server's current pair; the reply is
		// the fallback only when the read-back itself failed, and the requested
		// pair is never used as a local selection.
		selection, ok, readErr := a.bridge.RefreshModelFor(ctx, sessionID)
		catalog, status := a.readModels(ctx, sessionID)
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			if readErr != nil || !ok {
				outcome := result.Model.ProviderID + "/" + result.Model.ModelID
				catalog = markCurrentModel(catalog, result.Model.ProviderID, result.Model.ModelID)
				state.ModelCatalog = catalog
				state.ModelStatus = status
				state.ActionError = fmt.Sprintf("Model changed, but reading the current model back failed: %v", readBackReason(readErr, ok))
				state.ActionStatus = ""
				state.ActionNotices = appendActionNotice(state.ActionNotices,
					dimui.Notice{Kind: "model.selected", ID: outcome})
				return
			}
			outcome := selection.ProviderID + "/" + selection.ModelID
			state.ModelCatalog = catalog
			state.ModelStatus = status
			state.ActionError = ""
			state.ActionStatus = ""
			state.ActionNotices = appendActionNotice(state.ActionNotices,
				dimui.Notice{Kind: "model.selected", ID: outcome})
		})
	}()
}

// readBackReason explains why the model read-back did not produce a pair.
func readBackReason(err error, ok bool) string {
	if err != nil {
		return err.Error()
	}
	if !ok {
		return "session.summaries did not list this session"
	}
	return "no current pair was reported"
}

// markCurrentModel makes a catalog copy name the given pair as current. It is
// used only after the App Server's update reply adopted that pair but the
// session.summaries read-back failed; the reply is still a daemon fact.
func markCurrentModel(catalog dimui.ModelCatalog, providerID, modelID string) dimui.ModelCatalog {
	catalog.Current = providerID + "/" + modelID
	catalog.CurrentProviderID = providerID
	catalog.CurrentModelID = modelID
	catalog.HasCurrent = providerID != "" && modelID != ""
	for i := range catalog.Choices {
		catalog.Choices[i].Current = catalog.Choices[i].ProviderID == providerID && catalog.Choices[i].ModelID == modelID
	}
	return catalog
}

// readModels reads the selected session's model pair and catalog for display.
// The pair comes from the attach reply or a later session.summaries read-back;
// the catalog and its capability status come from
// provider.management/provider.selection. A failed catalog read leaves an empty
// list and a status that names the reason, never a made-up model.
func (a *app) readModels(ctx context.Context, sessionID string) (dimui.ModelCatalog, dimui.ModelStatus) {
	return readSessionModels(ctx, a.bridge, sessionID)
}

// readSessionModels is readModels for a bridge the caller holds directly; the
// first frame reads it before the app exists.
func readSessionModels(ctx context.Context, bridge *sessionBridge, sessionID string) (dimui.ModelCatalog, dimui.ModelStatus) {
	if bridge == nil {
		return dimui.ModelCatalog{}, dimui.ModelStatus{Reason: reasonNoModelClient}
	}
	selectable := bridge.CanUpdateModelFor(sessionID)
	current, hasCurrent := bridge.ModelFor(sessionID)
	catalog, err := bridge.ModelsFor(ctx, sessionID)
	if err != nil {
		status := dimui.ModelStatus{Selectable: selectable, Reason: err.Error()}
		out := dimui.ModelCatalog{}
		if hasCurrent && current.ProviderID != "" && current.ModelID != "" {
			out.Current = current.ProviderID + "/" + current.ModelID
			out.CurrentProviderID = current.ProviderID
			out.CurrentModelID = current.ModelID
			out.HasCurrent = true
		}
		return out, status
	}
	status := dimui.ModelStatus{Available: true, Selectable: selectable}
	if !selectable {
		// The catalog is readable but session.model.update is not advertised:
		// the list stays disabled and names the missing method as the WIP gap.
		status.Reason = reasonModelUpdateMissing
	}
	return modelCatalogView(catalog, current, hasCurrent), status
}

// reasonModelUpdateMissing explains the model list while the App Server does not
// advertise session.model.update. It is the window's own text; the wire method
// name is stated literally.
const reasonModelUpdateMissing = "Unavailable: this App Server does not advertise session.model.update, so choosing a model stays WIP."

// reasonNoModelClient explains the model chip while no session is attached to
// read the catalog from. It is the window's own text, not a wire error.
const reasonNoModelClient = "Unavailable: no session is attached to read the model catalog."

// modelCatalogView maps the session layer's model snapshot onto the shell's
// display copy. The current pair is the session's latest App Server pair; the
// catalog's own pair is shown only when the session reported none.
func modelCatalogView(catalog session.ModelCatalog, current session.ModelSelection, hasCurrent bool) dimui.ModelCatalog {
	if catalog.HasCurrent {
		current, hasCurrent = catalog.Current, true
	}
	out := dimui.ModelCatalog{HasCurrent: hasCurrent}
	if hasCurrent && current.ProviderID != "" && current.ModelID != "" {
		out.Current = current.ProviderID + "/" + current.ModelID
		out.CurrentProviderID = current.ProviderID
		out.CurrentModelID = current.ModelID
	} else {
		out.HasCurrent = false
	}
	for _, model := range catalog.Models {
		enabled := model.Enabled
		reason := model.UnavailableReason
		if len(model.MissingLimits) > 0 {
			// The picker must not present a model whose declared limits are
			// incomplete as selectable, even when the provider enables it.
			enabled = false
			if reason == "" {
				reason = "missing limits: " + strings.Join(model.MissingLimits, ", ")
			}
		}
		choice := dimui.ModelChoice{
			ProviderID:      model.ProviderID,
			ProviderName:    model.ProviderName,
			ModelID:         model.ModelID,
			DisplayName:     model.DisplayName,
			ContextWindow:   model.ContextWindow,
			MaxOutputTokens: model.MaxOutputTokens,
			ToolCalling:     model.ToolCalling,
			Vision:          model.Vision,
			Reasoning:       model.Reasoning,
			Source:          model.Source,
			Enabled:         enabled,
			Reason:          reason,
			Current: out.HasCurrent &&
				model.ProviderID == current.ProviderID && model.ModelID == current.ModelID,
		}
		out.Choices = append(out.Choices, choice)
	}
	return out
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

type branchCreateFlight struct {
	sessionID      string
	parentBranchID string
	at             int64
	prompt         string
}

// branchFrom creates a side branch from the cut point a user row resolved,
// then re-reads the branch directory and opens the branch the App Server
// returned. Nothing is created locally: the row's sequence only names the cut
// point, the returned branch id is the new fact, and the window switches to it
// only after OpenBranch read that line's history. A failure leaves the displayed
// branch and rows untouched and reports through ActionError; a created branch
// that cannot be opened is reported as created-but-not-shown instead of being
// presented as the selected line.
func (a *app) branchFrom(sessionID, branchID string, at int64, prompt string) {
	if a.bridge == nil || a.state == nil {
		return
	}
	if sessionID == "" {
		sessionID = a.state.Selected
	}
	if branchID == "" {
		branchID = a.actionBranch()
	}
	if branchID == "" {
		a.notice("branch.error", "Branch from this message: the window has no parent branch")
		return
	}
	if at < 0 {
		a.notice("branch.error", fmt.Sprintf("Branch from this message: invalid cut point %d", at))
		return
	}
	if strings.TrimSpace(prompt) == "" {
		a.notice("branch.error", "Branch from this message: the prompt is empty")
		return
	}
	flight := branchCreateFlight{sessionID: sessionID, parentBranchID: branchID, at: at, prompt: prompt}
	a.branchCreateMu.Lock()
	if _, pending := a.branchCreateInFlight[flight]; pending {
		a.branchCreateMu.Unlock()
		return
	}
	if a.branchCreateInFlight == nil {
		a.branchCreateInFlight = make(map[branchCreateFlight]struct{})
	}
	a.branchCreateInFlight[flight] = struct{}{}
	a.branchCreateMu.Unlock()
	gen := a.currentSelection()
	branchGen := a.beginBranchSelect()
	go func() {
		defer func() {
			a.branchCreateMu.Lock()
			delete(a.branchCreateInFlight, flight)
			a.branchCreateMu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		created, err := a.bridge.BranchFromFor(ctx, sessionID, branchID, at, prompt)
		if err != nil {
			log.Printf("dimagent: branch from message at %d: %v", at, err)
			if a.selectionCurrent(gen) {
				message := fmt.Sprintf("Branch from this message: %v", err)
				if errors.Is(err, appserver.ErrUnknownResult) {
					message = fmt.Sprintf("Branch creation result unknown: a branch may already have been created; trying again may create a duplicate. Check the branch directory before deciding whether to try again; the directory cannot conclusively identify this request. The request was not resent. (%v)", err)
				}
				a.notice("branch.error", message)
			}
			return
		}
		if created.Branch == nil || created.Branch.BranchID == "" {
			// appserver.BranchCreate already rejects a reply without an id; this
			// guard keeps a caller from switching to a branch the reply did not
			// name, even if that validation is bypassed.
			if a.selectionCurrent(gen) {
				a.notice("branch.error", "Branch from this message: the App Server returned no branch id")
			}
			return
		}
		newBranchID := created.Branch.BranchID
		if !a.selectionCurrent(gen) {
			return
		}
		// The directory is re-read before the new line is opened, so the sidebar
		// row and the selected branch come from the same daemon state. A failed
		// read does not undo the create: the branch still opens from its own
		// history, and the failure is reported after the switch.
		refreshErr := a.bridge.RefreshBranchesFor(ctx, sessionID)
		if refreshErr != nil {
			log.Printf("dimagent: refresh branches after creating %q: %v", newBranchID, refreshErr)
		}
		if !a.selectionCurrent(gen) {
			return
		}
		if err := a.bridge.OpenBranchFor(ctx, sessionID, newBranchID); err != nil {
			log.Printf("dimagent: open created branch %q: %v", newBranchID, err)
			if a.selectionCurrent(gen) {
				a.notice("branch.error", fmt.Sprintf("Branch %q was created, but it could not be opened: %v", newBranchID, err))
			}
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		view := a.bridge.LiveStateFor(sessionID, newBranchID)
		if view == nil {
			if a.selectionCurrent(gen) {
				a.notice("branch.error", fmt.Sprintf("Branch %q was created, but the session is no longer attached", newBranchID))
			}
			return
		}
		a.update(func(state *dimui.State) {
			if !a.selectionCurrent(gen) {
				return
			}
			// The displayed branch follows what the controller returned, never the
			// request: the App Server's reply is the new line's identity.
			if !a.commitBranch(view.SelectedBranch, branchGen) {
				return
			}
			applyView(state, view)
			state.ActionError = ""
			state.ActionStatus = ""
		})
		if refreshErr != nil {
			a.notice("branch.error", fmt.Sprintf("Branch %q was created, but the branch directory refresh failed: %v", newBranchID, refreshErr))
			return
		}
		a.notice("branch.created", newBranchID)
	}()
}

type branchEndFlight struct {
	sessionID string
	branchID  string
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
	flight := branchEndFlight{sessionID: sessionID, branchID: branchID}
	a.endBranchMu.Lock()
	if _, pending := a.endBranchInFlight[flight]; pending {
		a.endBranchMu.Unlock()
		return
	}
	if a.endBranchInFlight == nil {
		a.endBranchInFlight = make(map[branchEndFlight]struct{})
	}
	a.endBranchInFlight[flight] = struct{}{}
	a.endBranchMu.Unlock()
	go func() {
		defer func() {
			a.endBranchMu.Lock()
			delete(a.endBranchInFlight, flight)
			a.endBranchMu.Unlock()
		}()
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
	// A creation moves the window to a new session, so the composer text
	// belongs to the session being left; save it before the selection moves.
	a.rememberDraft(a.state.Selected)
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
		modelCatalog, modelStatus := a.readModels(ctx, row.ID)
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
			// A new session has no chosen branch, so drop the previous session's,
			// and it cannot carry the terminal state of the session being left.
			state.Selected = row.ID
			state.Draft = a.draftFor(row.ID)
			a.resetBranch()
			state.SessionClosed = false
			state.SessionClosedReason = ""
			state.ClosedSessionTitle = ""
			state.ClosedSessionProject = ""
			applyView(state, view)
			state.ModelCatalog = modelCatalog
			state.ModelStatus = modelStatus
			state.ModelListOpen = false
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

// A submission whose result is unknown is read back, never sent again (§10).
// The App Server may finish it after the request timed out, so the read-only
// lookup is retried within a small budget before the window calls the result
// unknown. Every retry names the same submission id and branch.
const (
	// submitLookupRetries is the number of read-only retries after the first
	// lookup did not confirm the submission. With submitLookupInterval that
	// stays inside submitLookupBudget.
	submitLookupRetries = 3
	// submitLookupInterval separates two read-backs.
	submitLookupInterval = 2 * time.Second
	// submitLookupBudget bounds the whole read-back, retries included.
	submitLookupBudget = 8 * time.Second
)

// submit sends the composer's text in the background and clears the draft only
// after the App Server accepted it. An ambiguous result is read back through
// session.input_lookup: it is never resent, and only the App Server's accepted
// answer reports it accepted.
func (a *app) submit(text string) {
	if a.bridge == nil || a.state == nil {
		return
	}
	a.submitMu.Lock()
	if a.submitInFlight {
		a.submitMu.Unlock()
		return
	}
	a.submitInFlight = true
	a.submitMu.Unlock()
	sessionID := a.state.Selected
	// A submission goes to the branch the window shows: the App Server opened
	// that branch, so it is the conversation the user is reading. With no branch
	// of its own the window shows the session's default branch, which is what
	// the branch-less submit names. The same branch names the read-back.
	branchID := a.actionBranch()
	gen := a.currentSelection()
	go func() {
		defer func() {
			a.submitMu.Lock()
			a.submitInFlight = false
			a.submitMu.Unlock()
		}()
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
		// The App Server holds this text, so its saved draft is done. Only the
		// draft that matches what was sent is cleared: text typed after the send
		// stays.
		a.clearDraftIfMatches(sessionID, text)
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
// id. The App Server may finish the submission after the request timed out, so
// the read-only lookup is retried within a small budget: only an "accepted"
// answer resolves it, anything else stays unknown, the draft is kept and the
// submission is never sent again. A selection change or a closed session stops
// the retries before they can publish into another session.
func (a *app) resolveUnknownSubmit(sessionID, branchID, submissionID, text string, submitErr error, gen uint64) {
	switch {
	case submissionID == "":
		detail := fmt.Sprintf("Submission result unknown (%v); read-back failed: %v",
			submitErr, errors.New("the App Server returned no submission id to read back"))
		a.unknownOutcome(gen, sessionID, detail)
		return
	case branchID == "":
		detail := fmt.Sprintf("Submission result unknown (%v); read-back failed: %v",
			submitErr, errors.New("no branch is known for the submission"))
		a.unknownOutcome(gen, sessionID, detail)
		return
	}
	// Capture the selection's context before the first read so a switch while
	// this function is between attempts cancels the wait below.
	selCtx := a.selectionContext()
	retries, interval, budget := a.lookupPolicy()
	deadline := time.Now().Add(budget)
	var (
		result  appserver.InputLookupResult
		lookupE error
	)
	for attempt := 0; ; attempt++ {
		if selCtx.Err() != nil || !a.selectionCurrent(gen) {
			return
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		ctx, cancel := context.WithTimeout(selCtx, remaining)
		// The read-back names the branch the submission was sent on, so an
		// unknown side-branch result is never resolved against the default one.
		result, lookupE = a.bridge.LookupInputBranchFor(ctx, sessionID, branchID, submissionID)
		cancel()
		if selCtx.Err() != nil || !a.selectionCurrent(gen) {
			return
		}
		if lookupE == nil && result.Accepted() {
			a.acceptResolvedSubmit(sessionID, submissionID, text, gen)
			return
		}
		if attempt >= retries {
			break
		}
		if !a.waitLookupRetry(selCtx, interval, deadline) {
			return
		}
	}
	if lookupE != nil {
		a.unknownOutcome(gen, sessionID, fmt.Sprintf("Submission result unknown (%v); read-back failed: %v", submitErr, lookupE))
		return
	}
	a.unknownOutcome(gen, sessionID, "Submission result unknown: the App Server does not hold this submission.")
}

// lookupPolicy returns the read-back retry policy: how many retries follow the
// first lookup, the pause between attempts, and the budget for all of them.
func (a *app) lookupPolicy() (retries int, interval, budget time.Duration) {
	retries, interval, budget = a.lookupRetries, a.lookupInterval, a.lookupBudget
	if retries <= 0 {
		retries = submitLookupRetries
	}
	if interval <= 0 {
		interval = submitLookupInterval
	}
	if budget <= 0 {
		budget = submitLookupBudget
	}
	return retries, interval, budget
}

// waitLookupRetry pauses between two read-backs without outliving the budget,
// and reports whether another attempt may run. A selection change or a closed
// session wakes the wait and ends the retries.
func (a *app) waitLookupRetry(selCtx context.Context, interval time.Duration, deadline time.Time) bool {
	wait := interval
	if remaining := time.Until(deadline); remaining < wait {
		wait = remaining
	}
	if wait <= 0 {
		return false
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-selCtx.Done():
		return false
	}
}

// acceptResolvedSubmit publishes a submission the read-back proved the App
// Server holds, exactly as a direct acceptance: the saved draft is done, any
// unknown outcome it superseded is cleared, and the window says accepted.
func (a *app) acceptResolvedSubmit(sessionID, submissionID, text string, gen uint64) {
	a.clearDraftIfMatches(sessionID, text)
	view := a.view()
	a.update(func(state *dimui.State) {
		// The read-back belongs to one session: a state that no longer shows it
		// must not receive its outcome, even when the generation did not move
		// (session.create selects a new session without a new open).
		if !a.selectionCurrent(gen) || state.Selected != sessionID {
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
		state.ActionNotices = withoutActionNotice(state.ActionNotices, "submit.unknown")
		state.ActionNotices = appendActionNotice(state.ActionNotices,
			dimui.Notice{Kind: "submit.accepted", ID: submissionID})
	})
}

// withoutActionNotice drops every notice of one kind, so an outcome a later
// read-back proves wrong cannot stay visible beside the corrected one.
func withoutActionNotice(notices []dimui.Notice, kind string) []dimui.Notice {
	kept := make([]dimui.Notice, 0, len(notices))
	for _, notice := range notices {
		if notice.Kind != kind {
			kept = append(kept, notice)
		}
	}
	return kept
}

// unknownOutcome publishes an unresolved action: it keeps the alert and the
// notice kind so an unknown result can never read as success. Like the accepted
// path, it only writes into the session the submission belonged to.
func (a *app) unknownOutcome(gen uint64, sessionID, detail string) {
	a.update(func(state *dimui.State) {
		if !a.selectionCurrent(gen) || state.Selected != sessionID {
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

type approvalFlight struct {
	sessionID string
	branchID  string
	requestID string
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
	flight := approvalFlight{sessionID: sessionID, branchID: branchID, requestID: requestID}
	a.approvalMu.Lock()
	if _, pending := a.approvalInFlight[flight]; pending {
		a.approvalMu.Unlock()
		return
	}
	if a.approvalInFlight == nil {
		a.approvalInFlight = make(map[approvalFlight]struct{})
	}
	a.approvalInFlight[flight] = struct{}{}
	a.approvalMu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		defer func() {
			a.approvalMu.Lock()
			delete(a.approvalInFlight, flight)
			a.approvalMu.Unlock()
		}()
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
	// -version answers before the daemon, the window and the bundle's URL
	// registration: it prints one line and the process exits 0.
	if printVersion(os.Args[1:], os.Stdout) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	state, bridge, err := stateFromDaemon(ctx, os.Getenv("DIMAGENT_PRODUCT_HOME"))
	cancel()
	if err != nil {
		state = &dimui.State{Connection: dimui.ConnError, ConnectionError: fmt.Sprintf("%v", err)}
		log.Printf("dimagent: %v", err)
	}
	application := &app{state: state, bridge: bridge, desktop: newDesktopStore()}
	application.bind()
	// The desktop's own state loads even without a daemon: it decides how the
	// shell draws, where the window opens and what draft each session keeps, not
	// what a session contains.
	application.loadDesktop()
	application.loadAppearance()
	// The build version is a fact of this binary, not of the saved
	// preferences, so it is set after the load that replaces the view. The
	// About page draws it; an unstamped build reads "dev".
	state.Settings.AppVersion = appVersion()
	if bridge != nil {
		defer func() {
			application.stopRefresh()
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = bridge.Close(closeCtx)
		}()
	}
	// Conversation links are the app's own scheme. Registration is best effort:
	// on macOS the scheme must also be declared for the bundle (urlSchemes in
	// mygo.config.ts), so a plain development binary cannot claim it. The failure
	// is recorded and the window still starts; a copied link is still the link
	// the handler below parses once the system delivers it.
	if err := mygo.App.RegisterURLScheme(dimui.ConversationLinkScheme); err != nil {
		log.Printf("dimagent: register the %q URL scheme: %v", dimui.ConversationLinkScheme, err)
	}
	mygo.App.OnOpenURL(application.openURL)
	mygo.App.WhenReady(func() {
		application.win = mygo.NewWindow(application.windowOptions())
		application.bindWindow()
		if application.bridge != nil {
			application.startRefresh(250 * time.Millisecond)
		}
	})
	runErr := mygo.App.Run()
	// A quit that did not close the window still keeps the user's draft and the
	// window's last geometry. Both saves are idempotent, so the close handler
	// having run already costs nothing.
	application.stopWindowTimer()
	application.saveWindowNow()
	application.saveCurrentDraft()
	if runErr != nil {
		log.Fatal(runErr)
	}
}
