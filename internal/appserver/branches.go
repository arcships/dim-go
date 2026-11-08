package appserver

import (
	"context"
	"encoding/json"
	"fmt"
)

// BranchesParams selects the branch directory for one session.
type BranchesParams struct {
	SessionID string `json:"sessionId"`
}

func (p BranchesParams) Validate() error {
	return requireNonEmpty("sessionId", p.SessionID)
}

// BranchInfo is one session.branches directory entry. Source and end retain
// their protocol JSON shape because branch provenance is not interpreted by
// the read helper.
type BranchInfo struct {
	BranchID       string          `json:"branchId"`
	ParentBranchID string          `json:"parentBranchId,omitempty"`
	BranchPoint    *int64          `json:"branchPoint,omitempty"`
	CreateEvent    *int64          `json:"createEvent,omitempty"`
	Source         json.RawMessage `json:"source,omitempty"`
	State          string          `json:"state"`
	Purpose        *string         `json:"purpose,omitempty"`
	CutMessages    *int64          `json:"cutMessages,omitempty"`
	End            json.RawMessage `json:"end,omitempty"`
	Head           int64           `json:"head"`
	EndEvent       *int64          `json:"endEvent,omitempty"`
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
	if b.CutMessages != nil && *b.CutMessages < 0 {
		return protocolErrorf("cutMessages", "got %d, want a non-negative integer", *b.CutMessages)
	}
	if b.EndEvent != nil && *b.EndEvent < 0 {
		return protocolErrorf("endEvent", "got %d, want a non-negative integer", *b.EndEvent)
	}
	return nil
}

// BranchesResult is the session.branches response.
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

func DecodeBranchesResult(raw json.RawMessage) (BranchesResult, error) {
	var r BranchesResult
	if err := decodeValidated("session.branches result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// SessionBranches reads the optional branch directory. It performs one
// request and never substitutes a local branch list when the capability is
// absent or the result is malformed.
func SessionBranches(ctx context.Context, client *Client, params BranchesParams, features []string) (BranchesResult, error) {
	if !HasFeature(features, "session.branches") {
		return BranchesResult{}, &CapabilityError{Feature: "session.branches"}
	}
	if err := params.Validate(); err != nil {
		return BranchesResult{}, err
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
