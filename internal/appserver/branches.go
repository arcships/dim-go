package appserver

import (
	"context"
	"encoding/json"
	"fmt"
)

// session.branches is a public protocol-v3 read. Desktop's app-server-protocol
// wire.ts declares SessionBranchesParams/SessionBranchesResult and
// session-reader.ts opens it on the session stream when hello.features
// advertises session.branches. It is an optional read: a failure degrades the
// caller instead of failing Session open, and this layer never synthesizes a
// directory.

// BranchesParams is the public request shape: {sessionId}.
type BranchesParams struct {
	SessionID string `json:"sessionId"`
}

func (p BranchesParams) Validate() error {
	return requireNonEmpty("sessionId", p.SessionID)
}

// BranchSource is the public wire.ts branch provenance union: a tool call or a
// host operation. The union fields are snake_case on the wire.
type BranchSource struct {
	Kind        string `json:"kind"`
	TurnID      string `json:"turn_id,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
	CallID      string `json:"call_id,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
}

func (s BranchSource) Validate() error {
	switch s.Kind {
	case "tool":
		if err := requireNonEmpty("source.turn_id", s.TurnID); err != nil {
			return err
		}
		if err := requireNonEmpty("source.request_id", s.RequestID); err != nil {
			return err
		}
		return requireNonEmpty("source.call_id", s.CallID)
	case "host":
		return requireNonEmpty("source.operation_id", s.OperationID)
	default:
		return protocolErrorf("source.kind", "got %q, want tool or host", s.Kind)
	}
}

// BranchInfo is one session.branches directory entry from wire.ts. State is
// "open", "ending" or "ended"; Head is live while open/ending and fixed once
// ended; EndEvent is absent while open.
//
// Purpose is not part of the public wire.ts entry. It is retained only because
// internal/integration currently renders it as a branch label; do not treat it
// as contract.
type BranchInfo struct {
	BranchID       string        `json:"branchId"`
	ParentBranchID string        `json:"parentBranchId,omitempty"`
	BranchPoint    *int64        `json:"branchPoint,omitempty"`
	CreateEvent    *int64        `json:"createEvent,omitempty"`
	Source         *BranchSource `json:"source,omitempty"`
	State          string        `json:"state"`
	Head           int64         `json:"head"`
	EndEvent       *int64        `json:"endEvent,omitempty"`
	Purpose        *string       `json:"purpose,omitempty"`
}

func (b BranchInfo) Validate() error {
	if err := requireNonEmpty("branchId", b.BranchID); err != nil {
		return err
	}
	if !oneOf(b.State, "open", "ending", "ended") {
		return protocolErrorf("state", "got %q, want open, ending or ended", b.State)
	}
	if b.Head < 0 {
		return protocolErrorf("head", "got %d, want a non-negative integer", b.Head)
	}
	if b.BranchPoint != nil && *b.BranchPoint < 0 {
		return protocolErrorf("branchPoint", "got %d, want a non-negative integer", *b.BranchPoint)
	}
	if b.CreateEvent != nil && *b.CreateEvent < 0 {
		return protocolErrorf("createEvent", "got %d, want a non-negative integer", *b.CreateEvent)
	}
	if b.EndEvent != nil && *b.EndEvent < 0 {
		return protocolErrorf("endEvent", "got %d, want a non-negative integer", *b.EndEvent)
	}
	if b.Source != nil {
		return b.Source.Validate()
	}
	return nil
}

// BranchesResult is the public session.branches response.
type BranchesResult struct {
	Branches []BranchInfo `json:"branches"`
}

func (r BranchesResult) Validate() error {
	if r.Branches == nil {
		return protocolErrorf("branches", "missing array")
	}
	for i := range r.Branches {
		if err := r.Branches[i].Validate(); err != nil {
			return fmt.Errorf("branches[%d]: %w", i, err)
		}
	}
	return nil
}

// DecodeBranchesResult decodes and validates one session.branches response.
func DecodeBranchesResult(raw json.RawMessage) (BranchesResult, error) {
	var r BranchesResult
	if err := decodeValidated("session.branches result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// SessionBranches reads the branch directory for one session. Desktop gates the
// same request on hello.features "session.branches" (session-reader.ts), so a
// missing feature returns *CapabilityError instead of sending a request.
//
// It is an optional read: one request, no retry, and no local fallback. On any
// error the result is zero so a malformed directory cannot be presented as a
// complete one.
func SessionBranches(ctx context.Context, client *Client, params BranchesParams, features []string) (BranchesResult, error) {
	if !HasFeature(features, "session.branches") {
		return BranchesResult{}, &CapabilityError{Feature: "session.branches"}
	}
	if err := params.Validate(); err != nil {
		return BranchesResult{}, err
	}
	if client == nil {
		return BranchesResult{}, protocolErrorf("session.branches", "nil client")
	}
	raw, err := client.Call(ctx, "session.branches", params)
	if err != nil {
		return BranchesResult{}, err
	}
	result, err := DecodeBranchesResult(raw)
	if err != nil {
		return BranchesResult{}, err
	}
	return result, nil
}
