package appserver

import (
	"context"
	"encoding/json"
	"fmt"
)

// CreateModel is the model selector accepted by session.create. Unlike
// SessionModel (a response/catalog shape), the request uses provider.
type CreateModel struct {
	Provider string `json:"provider"`
	ModelID  string `json:"modelId"`
}

func (m CreateModel) Validate() error {
	if err := requireNonEmpty("model.provider", m.Provider); err != nil {
		return err
	}
	return requireNonEmpty("model.modelId", m.ModelID)
}

// PermissionPolicy selects the daemon permission preset for a new session.
type PermissionPolicy struct {
	Preset string `json:"preset"`
}

func (p PermissionPolicy) Validate() error {
	return requireNonEmpty("permissionPolicy.preset", p.Preset)
}

// CreateParams is the control-plane session.create request.
type CreateParams struct {
	Cwd              string            `json:"cwd"`
	Model            *CreateModel      `json:"model,omitempty"`
	ModelPreferences *ModelPreferences `json:"modelPreferences,omitempty"`
	PermissionPolicy *PermissionPolicy `json:"permissionPolicy,omitempty"`
}

func (p CreateParams) Validate() error {
	if err := requireNonEmpty("cwd", p.Cwd); err != nil {
		return err
	}
	if p.Model != nil {
		if err := p.Model.Validate(); err != nil {
			return err
		}
	}
	if p.PermissionPolicy != nil {
		if err := p.PermissionPolicy.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// CreateResult is the response from session.create. sessionId is mandatory;
// optional model fields are retained when supplied by the daemon.
type CreateResult struct {
	SessionID        string            `json:"sessionId"`
	Model            *SessionModel     `json:"model,omitempty"`
	ModelPreferences *ModelPreferences `json:"modelPreferences,omitempty"`
}

func (r CreateResult) Validate() error {
	if err := requireNonEmpty("sessionId", r.SessionID); err != nil {
		return err
	}
	if r.Model != nil {
		if err := requireNonEmpty("model.providerId", r.Model.ProviderID); err != nil {
			return err
		}
		if err := requireNonEmpty("model.modelId", r.Model.ModelID); err != nil {
			return err
		}
	}
	return nil
}
func DecodeCreateResult(raw json.RawMessage) (CreateResult, error) {
	var r CreateResult
	if err := decodeValidated("session.create result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// Create creates a session through the public App Server control connection.
// There is intentionally no feature gate: unsupported daemons return their
// protocol unknown_method error unchanged.
func Create(ctx context.Context, client *Client, params CreateParams) (CreateResult, error) {
	if client == nil {
		return CreateResult{}, fmt.Errorf("appserver: nil client")
	}
	if err := params.Validate(); err != nil {
		return CreateResult{}, err
	}
	raw, err := client.Call(ctx, "session.create", params)
	if err != nil {
		return CreateResult{}, err
	}
	result, err := DecodeCreateResult(raw)
	if err != nil {
		return CreateResult{}, err
	}
	return result, nil
}
