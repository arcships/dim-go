package appserver

import (
	"context"
	"encoding/json"
)

// session.model.update is the public protocol-v3 write that switches the model
// of a Session's main branch. Desktop's app-server service gates it on
// hello.features "session.model.update" and sends
// {sessionId, branchId:"main", providerId, modelId}
// (electron-main/src/app-server/service.ts, setModel). The reply carries the
// pair the daemon adopted and an outcome:
//
//	{model: {providerId, modelId}, outcome: "adopted"|"unchanged"}
//
// The write has no idempotency key. A timeout or a lost transport is therefore
// an *UnknownResultError: the current pair is read back through
// session.summaries/session.attach and the request is never resent.

const (
	// ModelUpdateFeature is the hello.features capability that advertises
	// session.model.update. A connection without it keeps selection WIP.
	ModelUpdateFeature = "session.model.update"

	// ModelOutcomeAdopted means the daemon applied the pair.
	ModelOutcomeAdopted = "adopted"
	// ModelOutcomeUnchanged means the session already ran on that pair.
	ModelOutcomeUnchanged = "unchanged"
)

// ModelUpdateParams is the public session.model.update request. BranchID is
// optional on this side and sent as main when empty, matching Desktop's
// setModel; the daemon rejects any branch other than main with -32602.
type ModelUpdateParams struct {
	SessionID  string `json:"sessionId"`
	BranchID   string `json:"branchId,omitempty"`
	ProviderID string `json:"providerId"`
	ModelID    string `json:"modelId"`
}

func (p ModelUpdateParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("providerId", p.ProviderID); err != nil {
		return err
	}
	if err := requireNonEmpty("modelId", p.ModelID); err != nil {
		return err
	}
	if p.BranchID != "" && p.BranchID != MainBranchID {
		// The public contract is main-only; a side branch inherits the model at
		// its cut. Rejecting locally avoids a request the daemon must refuse.
		return protocolErrorf("branchId", "got %q, want %q", p.BranchID, MainBranchID)
	}
	return nil
}

// ModelUpdateResult is the public session.model.update reply.
type ModelUpdateResult struct {
	Model   SessionModel `json:"model"`
	Outcome string       `json:"outcome"`
}

func (r ModelUpdateResult) Validate() error {
	if err := requireNonEmpty("model.providerId", r.Model.ProviderID); err != nil {
		return err
	}
	if err := requireNonEmpty("model.modelId", r.Model.ModelID); err != nil {
		return err
	}
	if !oneOf(r.Outcome, ModelOutcomeAdopted, ModelOutcomeUnchanged) {
		return protocolErrorf("outcome", "got %q, want adopted or unchanged", r.Outcome)
	}
	return nil
}

// DecodeModelUpdateResult decodes and validates one session.model.update reply.
// Both fields are required: a reply without the adopted pair cannot be used as
// a read-back.
func DecodeModelUpdateResult(raw json.RawMessage) (ModelUpdateResult, error) {
	var result ModelUpdateResult
	if err := requireObjectFields("session.model.update result", raw, "model", "outcome"); err != nil {
		return result, err
	}
	if err := decodeValidated("session.model.update result", raw, &result); err != nil {
		return result, err
	}
	return result, nil
}

// ModelUpdate sends session.model.update.
//
// A connection that does not advertise ModelUpdateFeature returns
// *CapabilityError and sends nothing, so the caller keeps the picker WIP instead
// of pretending a selection. An empty BranchID is sent as main. A malformed
// reply is an *UnknownResultError because the write may have been applied; the
// pair is read back, never inferred.
func ModelUpdate(ctx context.Context, client *Client, params ModelUpdateParams, features []string) (ModelUpdateResult, error) {
	if !HasFeature(features, ModelUpdateFeature) {
		return ModelUpdateResult{}, &CapabilityError{Feature: ModelUpdateFeature}
	}
	if err := params.Validate(); err != nil {
		return ModelUpdateResult{}, err
	}
	if client == nil {
		return ModelUpdateResult{}, protocolErrorf("session.model.update", "nil client")
	}
	if params.BranchID == "" {
		params.BranchID = MainBranchID
	}
	raw, err := client.Call(ctx, string(MutationSessionModelUpdate), params)
	if err != nil {
		return ModelUpdateResult{}, mapMutationError(MutationSessionModelUpdate, err)
	}
	result, err := DecodeModelUpdateResult(raw)
	if err != nil {
		return ModelUpdateResult{}, &UnknownResultError{
			Method:   MutationSessionModelUpdate,
			ReadBack: mutationReadBack(MutationSessionModelUpdate),
			Err:      err,
		}
	}
	return result, nil
}
