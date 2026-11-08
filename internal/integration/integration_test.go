package integration

import (
	"context"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

func TestOpenSessionRejectsInvalidInputsBeforeDial(t *testing.T) {
	if _, err := OpenSession(nil, nil, "s1"); err == nil {
		t.Fatal("nil context should fail")
	}
	if _, err := OpenSession(context.Background(), nil, "s1"); err == nil {
		t.Fatal("nil supervisor should fail")
	}
	sup, err := supervisor.New(supervisor.Config{ProductHome: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenSession(context.Background(), sup, ""); err == nil {
		t.Fatal("empty session id should fail")
	}
}

func TestLiveStateWithoutSessionIsDisconnected(t *testing.T) {
	controller := &SessionController{}
	state := controller.LiveState(appserver.SessionSummary{SessionID: "s1"})
	if state == nil {
		t.Fatal("LiveState returned nil")
	}
	if state.Connection != ui.ConnUnknown {
		t.Fatalf("connection = %v, want unknown", state.Connection)
	}
	if len(state.Messages) != 0 {
		t.Fatalf("messages = %d, want empty", len(state.Messages))
	}
}

func TestCatalogRequiresStartedSupervisor(t *testing.T) {
	if _, err := Catalog(nil, nil); err == nil {
		t.Fatal("nil context should fail")
	}
	sup, err := supervisor.New(supervisor.Config{ProductHome: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Catalog(context.Background(), sup); err != supervisor.ErrNotStarted {
		t.Fatalf("Catalog error = %v, want %v", err, supervisor.ErrNotStarted)
	}
}
