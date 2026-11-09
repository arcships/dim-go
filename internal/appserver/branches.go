package appserver

import (
	"context"
	"encoding/json"
	"errors"
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
type BranchInfo struct {
	BranchID       string        `json:"branchId"`
	ParentBranchID string        `json:"parentBranchId,omitempty"`
	BranchPoint    *int64        `json:"branchPoint,omitempty"`
	CreateEvent    *int64        `json:"createEvent,omitempty"`
	Source         *BranchSource `json:"source,omitempty"`
	State          string        `json:"state"`
	Head           int64         `json:"head"`
	EndEvent       *int64        `json:"endEvent,omitempty"`
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

// ----- session.branch_create / session.branch_end -----
//
// Neither method has a DTO in Desktop's app-server-protocol wire.ts. The request
// shapes come from the public client call sites (service.ts) and the upstream
// design contract; the branch_create response shape has public-client evidence
// only. Evidence is fake-server (acceptance.md L1): no real-daemon exchange has
// been observed yet. Do not add compatibility aliases if upstream returns a
// different shape; that is a wire fix.

// BranchCreateParams creates a host branch (side chat) from an existing branch.
// At is the parent branch sequence the line is cut from; a pointer keeps a
// missing field distinct from the valid cut point 0.
type BranchCreateParams struct {
	SessionID      string `json:"sessionId"`
	ParentBranchID string `json:"parentBranchId"`
	At             *int64 `json:"at"`
	Prompt         string `json:"prompt"`
}

func (p BranchCreateParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("parentBranchId", p.ParentBranchID); err != nil {
		return err
	}
	if p.At == nil {
		return protocolErrorf("at", "missing")
	}
	if *p.At < 0 {
		return protocolErrorf("at", "got %d, want a non-negative integer", *p.At)
	}
	return requireNonEmpty("prompt", p.Prompt)
}

// BranchCreateInfo is the branch object the public client decodes from
// session.branch_create. BranchID is required: the public client treats a reply
// without it as an error instead of guessing which line was created.
type BranchCreateInfo struct {
	BranchID       string `json:"branchId"`
	ParentBranchID string `json:"parentBranchId,omitempty"`
	BranchPoint    *int64 `json:"branchPoint,omitempty"`
	CreateEvent    *int64 `json:"createEvent,omitempty"`
}

func (b BranchCreateInfo) Validate() error {
	if err := requireNonEmpty("branchId", b.BranchID); err != nil {
		return err
	}
	if b.BranchPoint != nil && *b.BranchPoint < 0 {
		return protocolErrorf("branchPoint", "got %d, want a non-negative integer", *b.BranchPoint)
	}
	if b.CreateEvent != nil && *b.CreateEvent < 0 {
		return protocolErrorf("createEvent", "got %d, want a non-negative integer", *b.CreateEvent)
	}
	return nil
}

// BranchCreateResult is the public-client response shape
// {branch?:{branchId?, parentBranchId?, branchPoint?, createEvent?}}. The
// pointer keeps "branch omitted" distinguishable from an empty object; both are
// protocol errors.
type BranchCreateResult struct {
	Branch *BranchCreateInfo `json:"branch"`
}

func (r BranchCreateResult) Validate() error {
	if r.Branch == nil {
		return protocolErrorf("branch", "missing")
	}
	return r.Branch.Validate()
}

// DecodeBranchCreateResult decodes and validates one session.branch_create
// response.
func DecodeBranchCreateResult(raw json.RawMessage) (BranchCreateResult, error) {
	var r BranchCreateResult
	if err := decodeValidated("session.branch_create result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// BranchCreate sends session.branch_create. It performs exactly one request and
// never resends: a timeout or disconnect is an *UnknownResultError whose
// read-back is session.branches (the directory can show the new line, but cannot
// prove that a timed-out create did not happen). A reply that omits branchId is
// also an unknown result wrapping the *ProtocolError.
//
// branch_create has no documented hello.features gate; callers detect an older
// host through *MutationError.Unsupported (appError unknown_method) and keep the
// entry visible-disabled.
func BranchCreate(ctx context.Context, client *Client, params BranchCreateParams) (BranchCreateResult, error) {
	if err := params.Validate(); err != nil {
		return BranchCreateResult{}, err
	}
	if client == nil {
		return BranchCreateResult{}, protocolErrorf("session.branch_create", "nil client")
	}
	raw, err := client.Call(ctx, string(MutationSessionBranchCreate), params)
	if err != nil {
		return BranchCreateResult{}, mapMutationError(MutationSessionBranchCreate, err)
	}
	result, err := DecodeBranchCreateResult(raw)
	if err != nil {
		return BranchCreateResult{}, &UnknownResultError{
			Method:   MutationSessionBranchCreate,
			ReadBack: mutationReadBack(MutationSessionBranchCreate),
			Err:      err,
		}
	}
	return result, nil
}

// BranchEndParams ends one non-main branch.
type BranchEndParams struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
}

func (p BranchEndParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	return requireNonEmpty("branchId", p.BranchID)
}

// BranchEndOutcome is the typed result of session.branch_end.
type BranchEndOutcome string

const (
	// BranchEndOutcomeEnded means the daemon accepted the end request. The
	// authoritative state still converges through session.branches/session.state.
	BranchEndOutcomeEnded BranchEndOutcome = "ended"
	// BranchEndOutcomeAlreadyEnded is the definitive branch_ended reply: the line
	// was already closed, nothing changed, and the request must not be retried.
	BranchEndOutcomeAlreadyEnded BranchEndOutcome = "already_ended"
)

// BranchEnd sends session.branch_end and maps the two contract-defined outcomes
// to typed values instead of an error:
//
//   - success -> BranchEndOutcomeEnded
//   - appError branch_ended -> BranchEndOutcomeAlreadyEnded, nil (no retry)
//
// appError unknown_method stays a *MutationError matching
// ErrCapabilityUnavailable (visible-disabled WIP). Timeout/disconnect is an
// *UnknownResultError with session.branches as the read-back. -32602 keeps the
// §13 precedence: invalid params, never "already ended".
func BranchEnd(ctx context.Context, client *Client, params BranchEndParams) (BranchEndOutcome, error) {
	if err := params.Validate(); err != nil {
		return "", err
	}
	if client == nil {
		return "", protocolErrorf("session.branch_end", "nil client")
	}
	if _, err := client.Call(ctx, string(MutationSessionBranchEnd), params); err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code != -32602 && rpcAppError(rpcErr.Data) == AppErrorBranchEnded {
			return BranchEndOutcomeAlreadyEnded, nil
		}
		return "", mapMutationError(MutationSessionBranchEnd, err)
	}
	return BranchEndOutcomeEnded, nil
}
