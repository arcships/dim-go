// Command dimagent is the MyGo native UI entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/integration"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// sessionActions is the part of integration.SessionController the window uses:
// the live view it adapts and the three mutations the shell's callbacks send.
// Keeping it small lets tests substitute a session without a daemon.
type sessionActions interface {
	LiveState(summary appserver.SessionSummary) *dimui.State
	Submit(ctx context.Context, text string) (appserver.SubmitResult, error)
	Stop(ctx context.Context) error
	Reattach(ctx context.Context) error
	Close(ctx context.Context) error
}

// sessionBridge owns the selected session's controller. It swaps the controller
// when the sidebar chooses another session and guards the pointer that the UI
// thread and background mutations share.
type sessionBridge struct {
	sup *supervisor.Supervisor

	// openSession opens a session through the supervisor. Tests replace it;
	// the app uses integration.OpenSession.
	openSession func(context.Context, *supervisor.Supervisor, string) (sessionActions, error)

	mu         sync.Mutex
	controller sessionActions
	summary    appserver.SessionSummary
}

var errNoSession = errors.New("dimagent: no session selected")

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

// LiveState returns a fresh view of the selected session, or nil without one.
func (b *sessionBridge) LiveState() *dimui.State {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	controller, summary := b.controller, b.summary
	b.mu.Unlock()
	if controller == nil {
		return nil
	}
	return controller.LiveState(summary)
}

// Submit sends text through the selected session.
func (b *sessionBridge) Submit(ctx context.Context, text string) error {
	if b == nil {
		return errNoSession
	}
	b.mu.Lock()
	controller := b.controller
	b.mu.Unlock()
	if controller == nil {
		return errNoSession
	}
	_, err := controller.Submit(ctx, text)
	return err
}

// Stop asks the selected session to stop its running turn.
func (b *sessionBridge) reattach(ctx context.Context) error {
	if b == nil {
		return errNoSession
	}
	b.mu.Lock()
	controller := b.controller
	b.mu.Unlock()
	if controller == nil {
		return errNoSession
	}
	return controller.Reattach(ctx)
}

func (b *sessionBridge) Stop(ctx context.Context) error {
	if b == nil {
		return errNoSession
	}
	b.mu.Lock()
	controller := b.controller
	b.mu.Unlock()
	if controller == nil {
		return errNoSession
	}
	return controller.Stop(ctx)
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
// window, not to one session's projection.
func applyView(state, view *dimui.State) {
	if state == nil || view == nil {
		return
	}
	state.Selected = view.Selected
	state.SelectedBranch = view.SelectedBranch
	state.Branches = append([]dimui.Branch(nil), view.Branches...)
	state.Connection = view.Connection
	state.ConnectionError = view.ConnectionError
	state.ConversationLoading = view.ConversationLoading
	state.Run = view.Run
	state.Messages = append([]dimui.Message(nil), view.Messages...)
	state.ModelProgress = view.ModelProgress
	state.Notices = append([]dimui.Notice(nil), view.Notices...)
}

func summaryForRow(row dimui.Session) appserver.SessionSummary {
	return appserver.SessionSummary{SessionID: row.ID, Title: &row.Title}
}

func newSupervisor(productHome string) (*supervisor.Supervisor, error) {
	return supervisor.New(supervisor.Config{ProductHome: productHome})
}

// app owns the window's state and the callbacks the shell calls. State writes
// from background work go through Window.Update, which runs them on the main
// thread; the callbacks themselves run on the UI thread.
type app struct {
	state  *dimui.State
	bridge *sessionBridge
	win    *mygo.Window

	mu            sync.Mutex
	selectGen     uint64
	refreshCancel context.CancelFunc
	// updateHook is test-only; production updates run through Window.Update.
	updateHook func(func(*dimui.State))
	// selectMu keeps session opens in order: a newer choice waits for the
	// attachment in flight, then replaces it.
	selectMu sync.Mutex
}

func (a *app) bind() {
	a.state.OnSelect = a.selectSession
	a.state.OnBranchSelect = a.selectBranch
	a.state.OnSubmit = a.submit
	a.state.OnStop = a.stop
	a.state.OnReconnect = a.reconnect
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

// beginSelect records a new selection and returns its generation.
func (a *app) beginSelect() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.selectGen++
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
				view := a.bridge.LiveState()
				if view == nil {
					continue
				}
				a.update(func(state *dimui.State) {
					// A selection may complete while LiveState is rebuilding; do
					// not let that older snapshot overwrite the new conversation.
					if !a.selectionCurrent(gen) {
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
			a.update(func(state *dimui.State) {
				state.ConversationLoading = false
				state.Connection = dimui.ConnError
				state.ConnectionError = fmt.Sprintf("%v", err)
			})
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		view := a.bridge.LiveState()
		a.update(func(state *dimui.State) {
			applyView(state, view)
			state.Selected = row.ID
			state.Connection = dimui.ConnReady
			state.ConnectionError = ""
			state.ConversationLoading = false
		})
	}()
}

// selectBranch refreshes the selected branch through the controller's existing
// session view/reducer. No branch mutation or local facts are created.
func (a *app) selectBranch(branchID string) {
	if a == nil || a.bridge == nil || a.state == nil || branchID == "" {
		return
	}
	a.bridge.mu.Lock()
	controller, summary := a.bridge.controller, a.bridge.summary
	a.bridge.mu.Unlock()
	branchViewer, ok := controller.(interface {
		StateForBranch(appserver.SessionSummary, string) *dimui.State
	})
	if !ok {
		return
	}
	view := branchViewer.StateForBranch(summary, branchID)
	a.update(func(state *dimui.State) {
		applyView(state, view)
		state.SelectedBranch = branchID
	})
}

// submit sends the composer's text in the background and clears the draft only
// after the App Server accepted it.
func (a *app) submit(text string) {
	if a.bridge == nil {
		return
	}
	gen := a.currentSelection()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.bridge.Submit(ctx, text); err != nil {
			log.Printf("dimagent: submit: %v", err)
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		view := a.bridge.LiveState()
		a.update(func(state *dimui.State) {
			applyView(state, view)
			state.Draft = ""
		})
	}()
}

// stop asks the App Server to stop the running turn; the run state converges
// through the session stream rather than a local guess.
// reconnect re-attaches the selected session after a transport loss. The
// controller rebuilds its projection from authoritative App Server history;
// this callback only coordinates the UI and never invents local facts.
func (a *app) reconnect() {
	if a.bridge == nil || a.state == nil {
		return
	}
	gen := a.currentSelection()
	a.update(func(state *dimui.State) {
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
		if err := a.bridge.reattach(ctx); err != nil {
			if !a.selectionCurrent(gen) {
				return
			}
			log.Printf("dimagent: reconnect: %v", err)
			a.update(func(state *dimui.State) {
				state.ConversationLoading = false
				state.Connection = dimui.ConnError
				state.ConnectionError = fmt.Sprintf("%v", err)
			})
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		view := a.bridge.LiveState()
		a.update(func(state *dimui.State) {
			applyView(state, view)
			state.ConversationLoading = false
		})
	}()
}

func (a *app) stop() {
	if a.bridge == nil {
		return
	}
	gen := a.currentSelection()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.bridge.Stop(ctx); err != nil {
			log.Printf("dimagent: stop: %v", err)
			return
		}
		if !a.selectionCurrent(gen) {
			return
		}
		view := a.bridge.LiveState()
		a.update(func(state *dimui.State) { applyView(state, view) })
	}()
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
			Title:   "DimAgent",
			Width:   1080,
			Height:  720,
			Content: ui.View(application.state.View),
		})
		if application.bridge != nil {
			application.startRefresh(250 * time.Millisecond)
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
