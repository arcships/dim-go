// Command dimagent is the MyGo native UI entry point.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/integration"
	"github.com/arcships/dimagent-mygo/internal/supervisor"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

// stateFromDaemon is deliberately opt-in: the UI never guesses a product home
// and never reads local daemon persistence. The supervisor is the sole source
// of transport/session facts when DIMAGENT_PRODUCT_HOME is configured.
func stateFromDaemon(ctx context.Context, productHome string) (*dimui.State, func(context.Context) error, error) {
	if strings.TrimSpace(productHome) == "" {
		return &dimui.State{Connection: dimui.ConnUnknown}, nil, nil
	}
	sup, err := newSupervisor(productHome)
	if err != nil {
		return nil, nil, err
	}
	if _, err := sup.Start(ctx); err != nil {
		return nil, closeSupervisor(sup), err
	}
	rows, err := integration.Catalog(ctx, sup)
	if err != nil {
		return nil, closeSupervisor(sup), err
	}
	state := &dimui.State{Sessions: rows, Connection: dimui.ConnReady}
	if len(rows) == 0 {
		return state, closeSupervisor(sup), nil
	}
	state.Selected = rows[0].ID
	controller, err := integration.OpenSession(ctx, sup, state.Selected)
	if err != nil {
		return nil, closeSupervisor(sup), err
	}
	view := controller.State(summaryForRow(rows[0]))
	state.Messages = view.Messages
	state.Run = view.Run
	return state, func(closeCtx context.Context) error {
		if err := controller.Close(closeCtx); err != nil {
			_, _ = sup.Close(closeCtx)
			return err
		}
		_, err := sup.Close(closeCtx)
		return err
	}, nil
}

func summaryForRow(row dimui.Session) appserver.SessionSummary {
	return appserver.SessionSummary{SessionID: row.ID, Title: &row.Title}
}

func newSupervisor(productHome string) (*supervisor.Supervisor, error) {
	return supervisor.New(supervisor.Config{ProductHome: productHome})
}

func closeSupervisor(sup *supervisor.Supervisor) func(context.Context) error {
	return func(ctx context.Context) error {
		_, err := sup.Close(ctx)
		return err
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	state, closeResources, err := stateFromDaemon(ctx, os.Getenv("DIMAGENT_PRODUCT_HOME"))
	if err != nil {
		state = &dimui.State{Connection: dimui.ConnError, ConnectionError: fmt.Sprintf("%v", err)}
		log.Printf("dimagent: %v", err)
	}
	if closeResources != nil {
		defer func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = closeResources(closeCtx)
		}()
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "DimAgent",
			Width:   1080,
			Height:  720,
			Content: ui.View(state.View),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
