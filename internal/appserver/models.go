package appserver

import (
	"context"
	"encoding/json"
	"fmt"
)

// Provider/model reads.
//
// The public App Server contract exposes the model catalog as process-owned
// Provider management (docs/design/appserver/provider-management.md §3.1):
//
//   - provider.management {providerId?} returns every known Provider with the
//     model descriptors it would serve, its enablement, the reason a model is
//     unavailable and its per-model preferences;
//   - provider.selection {cwd?} returns the stored default selection.
//
// Both are advertised in hello.features for local v3 Desktop/CLI callers, and
// dim-electron's app-server service reads them the same way
// (electron-main/src/app-server/service.ts). They are non-mutating ("Drain"
// interruption) reads.
//
// These types deliberately decode the fields a model picker consumes and ignore
// the rest of the management snapshot (drivers, connection, account state,
// model metadata). Unknown fields are tolerated; a field this decoder requires
// must be present and well formed, otherwise the reply is a protocol error and
// no partial catalog is returned.

// ProviderManagementParams selects the management scope. An empty ProviderID
// reads every known Provider; a non-empty ID reads exactly that entry.
type ProviderManagementParams struct {
	ProviderID string `json:"providerId,omitempty"`
}

// Validate accepts both the all-providers and the single-provider scope.
func (p ProviderManagementParams) Validate() error { return nil }

// CatalogCapabilities is the model descriptor's declared capability set. A
// missing limit is undeclared, not zero.
type CatalogCapabilities struct {
	Modalities       []string `json:"modalities"`
	ToolCalling      bool     `json:"toolCalling"`
	Streaming        bool     `json:"streaming"`
	StructuredOutput bool     `json:"structuredOutput"`
	Vision           bool     `json:"vision"`
	Audio            bool     `json:"audio"`
	Reasoning        bool     `json:"reasoning"`
	ContextWindow    *int64   `json:"contextWindow,omitempty"`
	MaxOutputTokens  *int64   `json:"maxOutputTokens,omitempty"`
}

func (c CatalogCapabilities) Validate() error {
	if c.ContextWindow != nil && *c.ContextWindow <= 0 {
		return protocolErrorf("capabilities.contextWindow", "got %d, want a positive integer", *c.ContextWindow)
	}
	if c.MaxOutputTokens != nil && *c.MaxOutputTokens <= 0 {
		return protocolErrorf("capabilities.maxOutputTokens", "got %d, want a positive integer", *c.MaxOutputTokens)
	}
	return nil
}

// ModelDescriptor is the model identity and declared capabilities from one
// Provider record snapshot.
type ModelDescriptor struct {
	ModelID      string              `json:"modelId"`
	DisplayName  string              `json:"displayName"`
	Status       string              `json:"status"`
	Capabilities CatalogCapabilities `json:"capabilities"`
}

func (m ModelDescriptor) Validate() error {
	if err := requireNonEmpty("model.modelId", m.ModelID); err != nil {
		return err
	}
	if err := requireNonEmpty("model.displayName", m.DisplayName); err != nil {
		return err
	}
	if !oneOf(m.Status, "active", "beta", "deprecated", "unavailable") {
		return protocolErrorf("model.status", "got %q, want active, beta, deprecated or unavailable", m.Status)
	}
	return m.Capabilities.Validate()
}

// ManagementModel is one model as the Provider serves it.
type ManagementModel struct {
	Model             ModelDescriptor `json:"model"`
	Source            string          `json:"source"`
	Enabled           bool            `json:"enabled"`
	Endpoint          *string         `json:"endpoint,omitempty"`
	MissingLimits     []string        `json:"missingLimits,omitempty"`
	UnavailableReason *string         `json:"unavailableReason,omitempty"`
	Actions           []string        `json:"actions"`
}

func (m ManagementModel) Validate() error {
	if err := m.Model.Validate(); err != nil {
		return err
	}
	if !oneOf(m.Source, "default", "override", "user") {
		return protocolErrorf("source", "got %q, want default, override or user", m.Source)
	}
	for i, limit := range m.MissingLimits {
		if !oneOf(limit, "contextWindow", "maxOutputTokens") {
			return protocolErrorf(fmt.Sprintf("missingLimits[%d]", i), "got %q, want contextWindow or maxOutputTokens", limit)
		}
	}
	return nil
}

// ManagementProvider is the Provider-record subset a model picker needs.
// Providers whose management entry has no local catalog record report Models
// as nil; that is "not listed", not an empty catalog.
type ManagementProvider struct {
	ProviderID          string                      `json:"providerId"`
	DisplayName         string                      `json:"displayName"`
	Source              string                      `json:"source"`
	CredentialAuthority string                      `json:"credentialAuthority"`
	Version             *int64                      `json:"version,omitempty"`
	Enabled             *bool                       `json:"enabled,omitempty"`
	Models              []ManagementModel           `json:"models,omitempty"`
	Preferences         map[string]ModelPreferences `json:"preferences"`
	ModelsListed        *string                     `json:"modelsListed,omitempty"`
}

func (p ManagementProvider) Validate() error {
	if err := requireNonEmpty("providerId", p.ProviderID); err != nil {
		return err
	}
	if err := requireNonEmpty("displayName", p.DisplayName); err != nil {
		return err
	}
	if !oneOf(p.Source, "builtin", "custom", "managed") {
		return protocolErrorf("source", "got %q, want builtin, custom or managed", p.Source)
	}
	if !oneOf(p.CredentialAuthority, "static", "openai_codex", "grok", "account") {
		return protocolErrorf("credentialAuthority", "got %q, want static, openai_codex, grok or account", p.CredentialAuthority)
	}
	if p.Version != nil && *p.Version < 0 {
		return protocolErrorf("version", "got %d, want a non-negative integer", *p.Version)
	}
	for i := range p.Models {
		if err := p.Models[i].Validate(); err != nil {
			return fmt.Errorf("models[%d]: %w", i, err)
		}
	}
	return nil
}

// ProviderRecordError reports one unhealthy Provider record without hiding the
// healthy entries.
type ProviderRecordError struct {
	ProviderID string `json:"providerId"`
	Message    string `json:"message"`
}

func (e ProviderRecordError) Validate() error {
	if err := requireNonEmpty("providerId", e.ProviderID); err != nil {
		return err
	}
	return requireNonEmpty("message", e.Message)
}

// ProviderManagementResult is the provider.management read snapshot.
type ProviderManagementResult struct {
	Providers []ManagementProvider  `json:"providers"`
	Errors    []ProviderRecordError `json:"errors"`
}

func (r ProviderManagementResult) Validate() error {
	if r.Providers == nil {
		return protocolErrorf("providers", "missing array")
	}
	seen := make(map[string]struct{}, len(r.Providers))
	for i := range r.Providers {
		if err := r.Providers[i].Validate(); err != nil {
			return fmt.Errorf("providers[%d]: %w", i, err)
		}
		if _, duplicate := seen[r.Providers[i].ProviderID]; duplicate {
			return protocolErrorf(fmt.Sprintf("providers[%d].providerId", i), "duplicate %q", r.Providers[i].ProviderID)
		}
		seen[r.Providers[i].ProviderID] = struct{}{}
	}
	for i := range r.Errors {
		if err := r.Errors[i].Validate(); err != nil {
			return fmt.Errorf("errors[%d]: %w", i, err)
		}
	}
	return nil
}

// DecodeProviderManagementResult decodes and validates one provider.management
// reply.
func DecodeProviderManagementResult(raw json.RawMessage) (ProviderManagementResult, error) {
	var r ProviderManagementResult
	if err := decodeValidated("provider.management result", raw, &r); err != nil {
		return ProviderManagementResult{}, err
	}
	return r, nil
}

// ProviderManagement reads the model catalog through provider.management.
//
// It is a single non-mutating read. A connection that does not advertise
// provider.management returns *CapabilityError instead of an empty catalog, so
// callers disable the picker and explain why rather than drawing "no models".
func ProviderManagement(ctx context.Context, client *Client, params ProviderManagementParams, features []string) (ProviderManagementResult, error) {
	if !HasFeature(features, "provider.management") {
		return ProviderManagementResult{}, &CapabilityError{Feature: "provider.management"}
	}
	if err := params.Validate(); err != nil {
		return ProviderManagementResult{}, err
	}
	if client == nil {
		return ProviderManagementResult{}, protocolErrorf("provider.management", "nil client")
	}
	raw, err := client.Call(ctx, "provider.management", params)
	if err != nil {
		return ProviderManagementResult{}, err
	}
	return DecodeProviderManagementResult(raw)
}

// ProviderSelectionParams is the default-selection scope: absent cwd reads the
// global selection, a cwd reads the stored workspace selection.
type ProviderSelectionParams struct {
	Cwd string `json:"cwd,omitempty"`
}

// Validate accepts both scopes.
func (p ProviderSelectionParams) Validate() error { return nil }

// ProviderSelection is one stored default model pair.
type ProviderSelection struct {
	ProviderID string `json:"providerId"`
	ModelID    string `json:"modelId"`
	Version    *int64 `json:"version"`
}

func (s ProviderSelection) Validate() error {
	if err := requireNonEmpty("selection.providerId", s.ProviderID); err != nil {
		return err
	}
	if err := requireNonEmpty("selection.modelId", s.ModelID); err != nil {
		return err
	}
	if s.Version == nil {
		return protocolErrorf("selection.version", "missing integer")
	}
	if *s.Version < 0 {
		return protocolErrorf("selection.version", "got %d, want a non-negative integer", *s.Version)
	}
	return nil
}

// ProviderSelectionResult is the provider.selection reply. A null or absent
// selection means the daemon has no stored default; it is never replaced by a
// local guess.
type ProviderSelectionResult struct {
	Selection *ProviderSelection `json:"selection"`
}

func (r ProviderSelectionResult) Validate() error {
	if r.Selection == nil {
		return nil
	}
	return r.Selection.Validate()
}

// DecodeProviderSelectionResult decodes and validates one provider.selection
// reply.
func DecodeProviderSelectionResult(raw json.RawMessage) (ProviderSelectionResult, error) {
	var r ProviderSelectionResult
	if err := decodeValidated("provider.selection result", raw, &r); err != nil {
		return ProviderSelectionResult{}, err
	}
	return r, nil
}

// ProviderSelection reads the stored default selection through
// provider.selection. A connection that does not advertise the method returns
// *CapabilityError; a null selection is a valid result.
func ReadProviderSelection(ctx context.Context, client *Client, params ProviderSelectionParams, features []string) (ProviderSelectionResult, error) {
	if !HasFeature(features, "provider.selection") {
		return ProviderSelectionResult{}, &CapabilityError{Feature: "provider.selection"}
	}
	if err := params.Validate(); err != nil {
		return ProviderSelectionResult{}, err
	}
	if client == nil {
		return ProviderSelectionResult{}, protocolErrorf("provider.selection", "nil client")
	}
	raw, err := client.Call(ctx, "provider.selection", params)
	if err != nil {
		return ProviderSelectionResult{}, err
	}
	return DecodeProviderSelectionResult(raw)
}
