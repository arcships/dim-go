package appserver

import (
	"context"
	"encoding/json"
	"fmt"
)

// MethodsParams selects the session-scoped method table.
type MethodsParams struct {
	SessionID string `json:"sessionId"`
}

func (p MethodsParams) Validate() error {
	return requireNonEmpty("sessionId", p.SessionID)
}

// MethodDeclaration is one method advertised by session.methods.
type MethodDeclaration struct {
	Name     string `json:"name"`
	Mutating bool   `json:"mutating"`
}

func (m MethodDeclaration) Validate() error {
	return requireNonEmpty("name", m.Name)
}

// MethodsResult is the session.methods response.
type MethodsResult struct {
	Methods []MethodDeclaration `json:"methods"`
}

func (r MethodsResult) Validate() error {
	if r.Methods == nil {
		return protocolErrorf("methods", "missing array")
	}
	for i := range r.Methods {
		if err := r.Methods[i].Validate(); err != nil {
			return fmt.Errorf("methods[%d]: %w", i, err)
		}
	}
	return nil
}

func DecodeMethodsResult(raw json.RawMessage) (MethodsResult, error) {
	var r MethodsResult
	if err := decodeValidated("session.methods result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// SessionMethods reads the session-scoped capability table. §16 defines
// session.methods as a built-in session-level method: it is not one of the
// connection-level hello.features, so there is no supportable feature gate and
// the features argument is accepted only for call-site compatibility.
//
// Malformed results are never returned as a partial table.
func SessionMethods(ctx context.Context, client *Client, params MethodsParams, _ []string) (MethodsResult, error) {
	if err := params.Validate(); err != nil {
		return MethodsResult{}, err
	}
	raw, err := client.Call(ctx, "session.methods", params)
	if err != nil {
		return MethodsResult{}, err
	}
	result, err := DecodeMethodsResult(raw)
	if err != nil {
		return MethodsResult{}, err
	}
	return result, nil
}
