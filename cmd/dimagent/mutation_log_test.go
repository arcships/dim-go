package main

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	dimui "github.com/arcships/dimagent-mygo/internal/ui"
)

func TestMutationFailureLogsExcludeSensitiveDetails(t *testing.T) {
	for _, operation := range []string{"model", "approval", "stop", "delete"} {
		t.Run(operation, func(t *testing.T) {
			const secret = "sensitive-response-token-and-user-content"
			var output bytes.Buffer
			old := log.Writer()
			log.SetOutput(&output)
			defer log.SetOutput(old)
			f := &fakeSession{updateModelErr: errors.New(secret), approvalErr: errors.New(secret), stopErr: errors.New(secret), deleteErr: errors.New(secret)}
			a := &app{state: &dimui.State{Selected: "s1"}, bridge: &sessionBridge{controller: f, summary: appserver.SessionSummary{SessionID: "s1"}}}
			done := make(chan struct{}, 1)
			a.updateHook = func(fn func(*dimui.State)) { fn(a.state); done <- struct{}{} }
			switch operation {
			case "model":
				a.selectModel("private-provider", "private-model")
			case "approval":
				a.respondApproval("main", "req-1", dimui.ApprovalAllow)
			case "stop":
				a.stop()
			case "delete":
				a.deleteSession("s1")
			}
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("failure not published")
			}
			if !strings.Contains(a.state.ActionError, secret) {
				t.Fatal("user-visible error lost")
			}
			logged := output.String()
			for _, s := range []string{secret, "private-provider", "private-model"} {
				if strings.Contains(logged, s) {
					t.Fatal("sensitive error or input logged")
				}
			}
			if !strings.Contains(logged, "s1") {
				t.Fatal("log lost safe session context")
			}
		})
	}
}
