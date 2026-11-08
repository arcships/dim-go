package integration

import (
	"context"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/supervisor"
)

func TestOpenSessionRejectsInvalidInputsBeforeDial(t *testing.T) {
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

func TestCatalogRequiresStartedSupervisor(t *testing.T) {
	sup, err := supervisor.New(supervisor.Config{ProductHome: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Catalog(context.Background(), sup); err != supervisor.ErrNotStarted {
		t.Fatalf("Catalog error = %v, want %v", err, supervisor.ErrNotStarted)
	}
}
