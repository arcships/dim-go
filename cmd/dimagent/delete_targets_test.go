package main

import (
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

func TestAppDeleteDifferentTargetsRemainQueued(t *testing.T) {
	f := &gatedDeleteSession{fakeSession: &fakeSession{}, started: make(chan struct{}, 8), release: make(chan error, 8)}
	b := &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}
	a := &app{state: &dimui.State{Selected: "s1"}, bridge: b}
	a.updateHook = func(fn func(*dimui.State)) { fn(a.state) }
	a.deleteSession("s1")
	select {
	case <-f.started:
	case <-time.After(time.Second):
		t.Fatal("first target not sent")
	}
	a.deleteSession("s2")
	a.deleteMu.Lock()
	pending := len(a.deleteInFlight)
	a.deleteMu.Unlock()
	if pending != 2 {
		t.Fatalf("distinct target suppressed: pending=%d", pending)
	}
	// Simulate selecting s2 while the first delete is pending. Lifecycle work
	// stays serialized, but s2 must eventually reach its own attached controller.
	a.mu.Lock()
	a.selectGen++
	a.mu.Unlock()
	b.mu.Lock()
	b.summary.SessionID = "s2"
	b.mu.Unlock()
	f.release <- nil
	select {
	case <-f.started:
	case <-time.After(time.Second):
		t.Fatal("second target not sent after lifecycle lock released")
	}
	f.release <- nil
	waitDeleteFlights(t, a)
}
