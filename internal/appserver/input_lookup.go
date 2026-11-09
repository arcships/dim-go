package appserver

import (
	"context"
	"encoding/json"
)

const (
	// InputLookupAccepted means the daemon confirmed the submission was queued
	// and published; Event is that durable input event.
	InputLookupAccepted = "accepted"
	// InputLookupUnknown means the daemon has no record of the submission.
	// Callers must not resend it (§10, §14).
	InputLookupUnknown = "unknown"
)

// InputLookupParams identifies a submission for outcome read-back after an
// unknown session.submit result (§10).
//
// BranchID is optional in the request type and omitted from the wire when
// empty. Note that protocol v3 still rejects a missing branchId for every
// session method (dim-app-server connection.rs: "branchId is required in v2/v3
// and absent in v1"), so callers must send a branch id on v3. It is not
// defaulted here: guessing "main" would read the wrong timeline for a child
// branch.
//
// Upstream also parses submissionId as a UUID before dispatch; callers must use
// the UUID returned by session.submit. This layer only requires a non-empty
// string so fixtures and opaque replay paths keep working.
type InputLookupParams struct {
	SessionID    string `json:"sessionId"`
	BranchID     string `json:"branchId,omitempty"`
	SubmissionID string `json:"submissionId"`
}

func (p InputLookupParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	return requireNonEmpty("submissionId", p.SubmissionID)
}

// InputLookupResult is the upstream tagged union
// `{status:"accepted",event} | {status:"unknown"}`. Event is the durable input
// event for an accepted submission and is absent for an unknown one; a JSON
// null event decodes as absent.
type InputLookupResult struct {
	Status string    `json:"status"`
	Event  *RawEvent `json:"event,omitempty"`
}

func (r InputLookupResult) Validate() error {
	switch r.Status {
	case InputLookupAccepted:
		if r.Event == nil {
			return protocolErrorf("event", "missing for status accepted")
		}
		return r.Event.Validate()
	case InputLookupUnknown:
		if r.Event != nil {
			return protocolErrorf("event", "must be absent for status unknown")
		}
		return nil
	default:
		return protocolErrorf("status", "got %q, want accepted or unknown", r.Status)
	}
}

// Accepted reports whether the daemon confirmed the submission was accepted.
func (r InputLookupResult) Accepted() bool { return r.Status == InputLookupAccepted }

// DecodeInputLookupResult decodes and validates one session.input_lookup reply.
func DecodeInputLookupResult(raw json.RawMessage) (InputLookupResult, error) {
	var r InputLookupResult
	if err := decodeValidated("session.input_lookup result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// InputLookupOutcome performs the §10 read-back and decodes the tagged union.
// It never resends the submission and never derives a local fact. On any error
// the result is zero, so an unverified payload cannot be mistaken for an
// accepted one.
func InputLookupOutcome(ctx context.Context, client *Client, params InputLookupParams) (InputLookupResult, error) {
	raw, err := InputLookup(ctx, client, params)
	if err != nil {
		return InputLookupResult{}, err
	}
	result, err := DecodeInputLookupResult(raw)
	if err != nil {
		return InputLookupResult{}, err
	}
	return result, nil
}

// InputLookup performs the contract read-back and returns the reply verbatim.
// It is the compatibility path for callers that still pass the payload through
// without interpreting it (for example internal/session.LookupInput). New
// callers should use InputLookupOutcome, which enforces the tagged union.
func InputLookup(ctx context.Context, client *Client, params InputLookupParams) (json.RawMessage, error) {
	if client == nil {
		return nil, protocolErrorf("session.input_lookup", "nil client")
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	raw, err := client.Call(ctx, "session.input_lookup", params)
	if err != nil {
		return nil, err
	}
	return append(json.RawMessage(nil), raw...), nil
}
