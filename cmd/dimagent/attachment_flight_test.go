package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

type gatedAttachmentSession struct {
	*fakeSession
	started chan string
	release chan error
}

func (f *gatedAttachmentSession) AttachmentPut(ctx context.Context, name, text string) (appserver.AttachmentPutResult, error) {
	f.started <- name + ":" + text
	select {
	case err := <-f.release:
		return appserver.AttachmentPutResult{Path: "/store/file.txt"}, err
	case <-ctx.Done():
		return appserver.AttachmentPutResult{}, ctx.Err()
	}
}
func TestAttachmentSingleFlightUsesWireIntent(t *testing.T) {
	for _, outcome := range []error{nil, errors.New("refused"), &appserver.UnknownResultError{Method: appserver.MutationAttachmentPut}} {
		name := "success"
		if outcome != nil {
			name = outcome.Error()
		}
		t.Run(name, func(t *testing.T) {
			path := writeAttachment(t, "file.txt", "one")
			samePayload := writeAttachment(t, "file.txt", "one")
			different := writeAttachment(t, "file.txt", "two")
			f := &gatedAttachmentSession{fakeSession: &fakeSession{}, started: make(chan string, 8), release: make(chan error, 8)}
			a := &app{state: &dimui.State{Selected: "s1", Draft: "existing", ActionError: "existing"}, bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}}
			a.updateHook = func(fn func(*dimui.State)) { fn(a.state) }
			done := make(chan struct{}, 8)
			send := func(path string) {
				gen := a.currentSelection()
				go func() { a.attachSelectedFile(gen, "s1", path); done <- struct{}{} }()
			}
			send(path)
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("upload not started")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			send(samePayload)
			select {
			case <-f.started:
				t.Fatal("duplicate payload reached controller")
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("duplicate not suppressed")
			}
			if a.state.ActionStatus != "Attachment upload already in progress; duplicate selection was not sent." {
				t.Fatal("duplicate upload suppression was not visible")
			}
			send(different)
			select {
			case payload := <-f.started:
				if payload != "file.txt:two" {
					t.Fatal("wrong payload")
				}
			case <-time.After(time.Second):
				t.Fatal("different content suppressed")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			f.release <- outcome
			f.release <- outcome
			for i := 0; i < 2; i++ {
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Fatal("upload not completed")
				}
			}
			if a.state.Draft != "existing" || a.state.ActionError != "existing" || a.state.ActionStatus != "Attachment upload already in progress; duplicate selection was not sent." {
				t.Fatal("stale upload published")
			}
			send(path)
			select {
			case <-f.started:
			case <-time.After(time.Second):
				t.Fatal("guard not released")
			}
			a.mu.Lock()
			a.selectGen++
			a.mu.Unlock()
			f.release <- outcome
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("retry did not complete")
			}
		})
	}
}
