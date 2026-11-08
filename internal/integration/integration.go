package integration

import (
	"context"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/session"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	"github.com/arcships/dimagent-mygo/internal/ui"
	"github.com/arcships/dimagent-mygo/internal/viewmodel"
)

// SessionController owns the transport and in-memory session view for one
// selected session. It is the narrow supervisor -> session -> viewmodel bridge;
// no daemon files or local persistence are consulted.
type SessionController struct {
	Supervisor *supervisor.Supervisor
	Conn       *supervisor.Connection
	Session    *session.Session
}

// OpenSession opens a dedicated stream through sup and loads the first history
// window. The returned controller owns the stream connection until Close.
func OpenSession(ctx context.Context, sup *supervisor.Supervisor, sessionID string) (*SessionController, error) {
	if ctx == nil {
		return nil, fmt.Errorf("integration: nil context")
	}
	if sup == nil {
		return nil, fmt.Errorf("integration: nil supervisor")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("integration: empty session id")
	}
	conn, err := sup.OpenStream(ctx)
	if err != nil {
		return nil, err
	}
	s, err := session.OpenWithOptions(ctx, conn.Client(), conn.Features(), sessionID, session.OpenOptions{CloseTransport: conn.Close})
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &SessionController{Supervisor: sup, Conn: conn, Session: s}, nil
}

// Close releases the session attachment and its supervisor-tracked stream.
func (c *SessionController) Close(ctx context.Context) error {
	if c == nil || c.Session == nil {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("integration: nil context")
	}
	return c.Session.Detach(ctx)
}

// State builds throwaway UI state from catalog and the current session view.
// Facts remain owned by App Server/projection; this function only adapts them.
// A controller without a session returns an empty, disconnected state.
func (c *SessionController) State(summary appserver.SessionSummary) *ui.State {
	if c == nil || c.Session == nil {
		return &ui.State{Connection: ui.ConnUnknown}
	}
	state := &ui.State{
		Sessions:   []ui.Session{{ID: summary.SessionID, Title: sessionTitle(summary)}},
		Selected:   summary.SessionID,
		Connection: ui.ConnReady,
		Messages:   viewmodel.Build(c.Session.Snapshot()),
		Run:        ui.RunIdle,
	}
	if view := c.Session.View(c.Session.BranchID()); view.State != nil {
		if view.State.Running {
			state.Run = ui.RunRunning
		}
		if view.State.Stopping {
			state.Run = ui.RunStopping
		}
	} else {
		state.Run = viewmodel.RunStateFrom(c.Session.Snapshot())
	}
	return state
}

func sessionTitle(s appserver.SessionSummary) string {
	if s.Title != nil && *s.Title != "" {
		return *s.Title
	}
	if s.Preview != nil && *s.Preview != "" {
		return *s.Preview
	}
	return s.SessionID
}

// LiveState returns a fresh view of the selected session. Session's drain
// goroutine continuously folds live stream events into its in-memory projection;
// this method intentionally rebuilds throwaway UI rows on each call rather than
// introducing a second event/fact source. Callers can invoke it after a render
// tick or event notification to refresh the UI.
func (c *SessionController) LiveState(summary appserver.SessionSummary) *ui.State {
	return c.State(summary)
}

// Catalog reads the authoritative session list over the supervisor control
// connection and adapts it to sidebar rows.
func Catalog(ctx context.Context, sup *supervisor.Supervisor) ([]ui.Session, error) {
	if ctx == nil {
		return nil, fmt.Errorf("integration: nil context")
	}
	if sup == nil {
		return nil, fmt.Errorf("integration: nil supervisor")
	}
	conn, ok := sup.Control()
	if !ok {
		return nil, supervisor.ErrNotStarted
	}
	limit := 200
	result, err := appserver.Catalog(ctx, conn.Client(), appserver.CatalogParams{Limit: &limit}, conn.Features())
	if err != nil {
		return nil, err
	}
	rows := make([]ui.Session, 0, len(result.Sessions))
	for _, summary := range result.Sessions {
		rows = append(rows, ui.Session{ID: summary.SessionID, Title: sessionTitle(summary)})
	}
	return rows, nil
}
