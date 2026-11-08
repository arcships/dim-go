package session

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// InputLookupParams is kept as an alias for callers of the session package.
type InputLookupParams = appserver.InputLookupParams

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
	params := appserver.InputLookupParams{
		SessionID:    s.sessionID,
		BranchID:     branchID,
		SubmissionID: submissionID,
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return appserver.InputLookup(ctx, s.client, params)
}
