package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// Branch switching has no RPC of its own (docs/branch-attachment-plan.md §1.2):
// the App Server owns branch facts, and a branch becomes viewable by reading
// session.history for its branchId. This file only folds that read into the
// existing per-branch projection. Nothing here writes local branch state or
// treats a missing capability as an empty branch.
//
// Per-branch session.keys is a separate optional read (a keyed display block,
// not durable history) and is deliberately not loaded here.

var (
	// ErrUnknownBranch marks a branch that cannot be proven to exist on this
	// session. The view is never synthesized for it.
	ErrUnknownBranch = errors.New("session: unknown branch")
	// ErrMainBranchEnd marks an attempt to end the main branch; branch_end is
	// defined for non-main lines only.
	ErrMainBranchEnd = errors.New("session: the main branch cannot be ended")
)

// UnknownBranchError names the branch whose existence could not be proven and
// why (missing directory entry, or session.branches unavailable and no event
// observed for it).
type UnknownBranchError struct {
	SessionID string
	BranchID  string
	Reason    string
}

func (e *UnknownBranchError) Error() string {
	return fmt.Sprintf("session: %s: unknown branch %q: %s", e.SessionID, e.BranchID, e.Reason)
}

// Is lets errors.Is(err, ErrUnknownBranch) match any unknown-branch report.
func (e *UnknownBranchError) Is(target error) bool { return target == ErrUnknownBranch }

// BranchOpenResult reports one OpenBranch call. AlreadyLoaded means the branch
// had finished its first-screen read and no request was sent.
type BranchOpenResult struct {
	BranchID      string
	AlreadyLoaded bool
	History       HistoryStatus
	Head          int64
}

// BranchLoaded reports whether OpenBranch finished the first-screen history
// read for branchID. Live events alone do not make a branch loaded.
func (s *Session) BranchLoaded(branchID string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	branch := s.branches[branchID]
	return branch != nil && branch.loaded
}

// OpenBranch loads one branch's first-screen history window and folds it into
// that branch's own projection, using the same latestTurn/compatibility strategy
// as Open. It is idempotent: a branch that already finished its first-screen
// read returns AlreadyLoaded without another request.
//
// Existence is only accepted from the App Server: a session.branches entry
// (whose Head bounds the read, and is fixed once the branch ended) or events
// already observed for that branch on this stream. An unknown branch is an
// *UnknownBranchError; no empty projection is created for it.
func (s *Session) OpenBranch(ctx context.Context, branchID string) (BranchOpenResult, error) {
	if s == nil {
		return BranchOpenResult{}, fmt.Errorf("session: nil session")
	}
	if branchID == "" {
		return BranchOpenResult{}, fmt.Errorf("session: empty branch id")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if !appserver.HasFeature(s.features, "session.history") {
		return BranchOpenResult{}, &appserver.CapabilityError{Feature: "session.history"}
	}

	s.mu.Lock()
	if branch := s.branches[branchID]; branch != nil && branch.loaded {
		result := BranchOpenResult{
			BranchID:      branchID,
			AlreadyLoaded: true,
			History:       cloneHistoryStatus(branch.history),
			Head:          s.headLocked(branchID),
		}
		s.mu.Unlock()
		return result, nil
	}
	head, reason := s.branchHeadLocked(branchID)
	s.mu.Unlock()
	if reason != "" {
		return BranchOpenResult{}, &UnknownBranchError{SessionID: s.sessionID, BranchID: branchID, Reason: reason}
	}

	var window historyWindow
	var err error
	if appserver.HasFeature(s.features, "session.history.latestTurn") {
		window, err = s.readLatestTurn(ctx, branchID, head)
		if err != nil {
			return BranchOpenResult{}, s.openFailure(err)
		}
		if window.fallback {
			window, err = s.readCompatibility(ctx, branchID, head, "latest_turn_turn_start_drift")
			if err != nil {
				return BranchOpenResult{}, s.openFailure(err)
			}
		}
	} else {
		window, err = s.readCompatibility(ctx, branchID, head, "")
		if err != nil {
			return BranchOpenResult{}, s.openFailure(err)
		}
	}

	s.mu.Lock()
	for _, event := range window.events {
		if err := s.applyEventLocked(branchID, event); err != nil {
			s.mu.Unlock()
			return BranchOpenResult{}, s.openFailure(err)
		}
	}
	branch := s.branchLocked(branchID)
	// The daemon head bounds the window; live events may already have advanced
	// it past that point. Never lower it.
	if head > branch.head {
		branch.head = head
	}
	branch.history = window.status
	branch.loaded = true
	result := BranchOpenResult{
		BranchID: branchID,
		History:  cloneHistoryStatus(branch.history),
		Head:     branch.head,
	}
	s.mu.Unlock()
	return result, nil
}

// branchHeadLocked returns the through bound for a branch's first-screen read,
// or a non-empty reason when the App Server has not proven the branch exists.
// It must be called with s.mu held.
func (s *Session) branchHeadLocked(branchID string) (int64, string) {
	observed := s.branches[branchID]
	if branchID == appserver.MainBranchID {
		return s.stream.Head(), ""
	}
	if s.branchesLoaded {
		for _, entry := range s.branchDirectory.Branches {
			if entry.BranchID != branchID {
				continue
			}
			head := entry.Head
			if observed != nil && observed.head > head {
				head = observed.head
			}
			return head, ""
		}
	}
	// A branch with events or state on this stream exists even when the
	// directory snapshot is missing or stale; those facts come from the daemon.
	if observed != nil && (observed.head > 0 || observed.state != nil) {
		return observed.head, ""
	}
	if !s.branchesLoaded {
		return 0, "session.branches is unavailable and no event for this branch was observed"
	}
	return 0, "branch is not in the session.branches directory"
}

// RefreshBranchDirectory re-reads session.branches and replaces the read-only
// snapshot. A missing capability or a failed read leaves the previous snapshot
// untouched and records the reason in BranchDirectoryStatus; it never
// synthesizes a directory.
func (s *Session) RefreshBranchDirectory(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("session: nil session")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	branches, err := appserver.SessionBranches(ctx, s.client, appserver.BranchesParams{SessionID: s.sessionID}, s.features)
	if err != nil {
		s.setBranchesStatus(ReadCapabilityStatus{
			Capability: "session.branches",
			Available:  false,
			Reason:     err.Error(),
		})
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.branchDirectory = branches
	s.branchesLoaded = true
	s.branchesStatus = ReadCapabilityStatus{Capability: "session.branches", Available: true}
	return nil
}

// EndBranch asks the App Server to end one non-main branch. It never deletes
// local branch facts: the directory entry, projection and head stay until the
// daemon reports the new state through session.branches/session.state.
//
// branch_ended maps to appserver.BranchEndOutcomeAlreadyEnded with a nil error
// (nothing changed, do not retry); unknown_method stays an
// *appserver.MutationError matching appserver.ErrCapabilityUnavailable so the
// entry can be shown as visible-disabled WIP.
func (s *Session) EndBranch(ctx context.Context, branchID string) (appserver.BranchEndOutcome, error) {
	if s == nil {
		return "", fmt.Errorf("session: nil session")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if branchID == "" {
		return "", fmt.Errorf("session: empty branch id")
	}
	if branchID == appserver.MainBranchID {
		return "", ErrMainBranchEnd
	}
	return appserver.BranchEnd(ctx, s.client, appserver.BranchEndParams{
		SessionID: s.sessionID,
		BranchID:  branchID,
	})
}

// BranchCreate asks the App Server to create a host branch (side chat) from
// parentBranchID at the given parent-branch sequence. At is the cut point the
// window resolved from a user input's sequence; the App Server remains the
// final validator of the boundary.
//
// session.branch_create has no documented hello.features gate of its own. The
// window cannot present the created line without session.branches, so this
// layer gates on that directory capability and reports a missing one as
// *appserver.CapabilityError instead of sending a request whose result could not
// be shown. The call is a write: it is performed exactly once, and an unknown
// result or subscription_changed reply is reported to the caller, never resent.
func (s *Session) BranchCreate(ctx context.Context, parentBranchID string, at int64, prompt string) (appserver.BranchCreateResult, error) {
	if s == nil {
		return appserver.BranchCreateResult{}, fmt.Errorf("session: nil session")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if !appserver.HasFeature(s.features, "session.branches") {
		return appserver.BranchCreateResult{}, &appserver.CapabilityError{Feature: "session.branches"}
	}
	if parentBranchID == "" {
		return appserver.BranchCreateResult{}, fmt.Errorf("session: empty parent branch id")
	}
	if at < 0 {
		return appserver.BranchCreateResult{}, fmt.Errorf("session: branch cut point %d is negative", at)
	}
	if strings.TrimSpace(prompt) == "" {
		return appserver.BranchCreateResult{}, fmt.Errorf("session: empty branch prompt")
	}
	return appserver.BranchCreate(ctx, s.client, appserver.BranchCreateParams{
		SessionID:      s.sessionID,
		ParentBranchID: parentBranchID,
		At:             &at,
		Prompt:         prompt,
	})
}
