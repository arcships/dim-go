package session

import (
	"context"
	"fmt"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// session.keys and session.branches are public Desktop calls (wire.ts and
// session-reader.ts) and both appserver helpers are public. Open gates each read
// on hello.features and treats it as optional enrichment: a failed or malformed
// read leaves the snapshot empty and records the reason, but never fails Open or
// synthesizes local branch/key facts.

// optionalKeysTimeout matches Desktop session-reader.ts:1041-1045.
const optionalKeysTimeout = 5 * time.Second

func (s *Session) readSessionKeys(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	readCtx, cancel := context.WithTimeout(ctx, optionalKeysTimeout)
	defer cancel()

	through := s.History().Through
	params := appserver.KeysParams{
		SessionID: s.sessionID,
		BranchID:  s.branchID,
		Through:   &through,
	}
	keys, err := appserver.SessionKeys(readCtx, s.client, params, s.features)
	if err != nil {
		s.failSessionKeys(err)
		return
	}
	s.mu.Lock()
	s.keySnapshot = keys
	s.keysLoaded = true
	s.keysStatus = ReadCapabilityStatus{Capability: "session.keys", Available: true}
	s.mu.Unlock()
}

func (s *Session) failSessionKeys(err error) {
	s.setKeysStatus(ReadCapabilityStatus{
		Capability: "session.keys",
		Available:  false,
		Reason:     err.Error(),
	})
	s.report(fmt.Errorf("session: session.keys unavailable: %w", err))
}

func (s *Session) setKeysStatus(status ReadCapabilityStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keysStatus = status
}

func (s *Session) readBranchDirectory(ctx context.Context) {
	if err := s.RefreshBranchDirectory(ctx); err != nil {
		s.report(fmt.Errorf("session: session.branches unavailable: %w", err))
	}
}

func (s *Session) setBranchesStatus(status ReadCapabilityStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.branchesStatus = status
}
