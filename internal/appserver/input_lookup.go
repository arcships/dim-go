package appserver

import (
	"context"
	"encoding/json"
)

// InputLookupParams identifies a submission for outcome read-back after an
// unknown session.submit result. The result schema is intentionally opaque.
type InputLookupParams struct {
	SessionID    string `json:"sessionId"`
	BranchID     string `json:"branchId"`
	SubmissionID string `json:"submissionId"`
}

func (p InputLookupParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("branchId", p.BranchID); err != nil {
		return err
	}
	return requireNonEmpty("submissionId", p.SubmissionID)
}

// InputLookup performs the contract read-back without interpreting the
// upstream response shape.
func InputLookup(ctx context.Context, client *Client, params InputLookupParams) (json.RawMessage, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	raw, err := client.Call(ctx, "session.input_lookup", params)
	if err != nil {
		return nil, err
	}
	return append(json.RawMessage(nil), raw...), nil
}
