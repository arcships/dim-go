package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// InputLookupParams is the documented session.input_lookup request from
// docs/protocol-contract.md §10. The contract defines the request but no
// response schema, so LookupInput returns the raw JSON result instead of
// inventing a local shape.
type InputLookupParams struct {
	SessionID    string `json:"sessionId"`
	BranchID     string `json:"branchId"`
	SubmissionID string `json:"submissionId"`
}

// Validate checks the parameters the App Server contract requires.
func (p InputLookupParams) Validate() error {
	if strings.TrimSpace(p.SessionID) == "" {
		return &appserver.ProtocolError{Field: "sessionId", Detail: "must not be empty"}
	}
	if strings.TrimSpace(p.BranchID) == "" {
		return &appserver.ProtocolError{Field: "branchId", Detail: "must not be empty"}
	}
	if strings.TrimSpace(p.SubmissionID) == "" {
		return &appserver.ProtocolError{Field: "submissionId", Detail: "must not be empty"}
	}
	return nil
}

// LookupInput reads a submission back by its idempotency key after an unknown
// submit result. It never resends the mutation and never derives a local fact;
// the App Server result is returned verbatim because its schema is not frozen
// in the protocol contract.
func (s *Session) LookupInput(ctx context.Context, branchID, submissionID string) (json.RawMessage, error) {
	if s == nil {
		return nil, fmt.Errorf("session: nil session")
	}
	if s.client == nil {
		return nil, fmt.Errorf("session: nil client")
	}
	if branchID == "" {
		branchID = s.branchID
	}
	params := InputLookupParams{
		SessionID:    s.sessionID,
		BranchID:     branchID,
		SubmissionID: submissionID,
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	raw, err := s.client.Call(ctx, "session.input_lookup", params)
	if err != nil {
		return nil, err
	}
	return append(json.RawMessage(nil), raw...), nil
}
