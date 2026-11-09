package session

import (
	"context"
	"fmt"
	"sort"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// ModelSelection is one provider/model pair reported by the App Server.
type ModelSelection struct {
	ProviderID string
	ModelID    string
}

// ModelOption is one selectable model as the Provider serves it. Enabled and
// UnavailableReason are distinct facts: an enabled model can still be
// unavailable because its declared limits are missing, and the picker must not
// present it as selectable. ProviderEnabled is nil when the account-owned
// record has no Provider enable flag.
type ModelOption struct {
	ProviderID        string
	ProviderName      string
	ProviderEnabled   *bool
	ModelID           string
	DisplayName       string
	Source            string
	Enabled           bool
	UnavailableReason string
	MissingLimits     []string
	ContextWindow     *int64
	MaxOutputTokens   *int64
	ToolCalling       bool
	Vision            bool
	Reasoning         bool
}

// ModelCatalog is a read-only snapshot of the App Server model catalog plus the
// daemon's stored default selection. Current is the model pair the attached
// session reported; HasCurrent is false when the attach reply carried no
// complete pair. DefaultStatus explains a degraded provider.selection read; the
// catalog itself is still the daemon's answer, never an empty placeholder.
type ModelCatalog struct {
	Models        []ModelOption
	Current       ModelSelection
	HasCurrent    bool
	Default       *ModelSelection
	DefaultStatus ReadCapabilityStatus
}

// Model returns the model pair the attach reply reported for this session. ok is
// false when the daemon did not report a complete pair; no local selection is
// invented.
func (s *Session) Model() (ModelSelection, bool) {
	if s == nil {
		return ModelSelection{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.model, s.hasModel
}

// Models reads the model catalog through provider.management and the stored
// default through provider.selection. The read is non-mutating and always asks
// the App Server; this package keeps no second catalog.
//
// A connection that does not advertise provider.management returns
// *appserver.CapabilityError so the caller disables the picker and explains the
// gap. A failed or malformed provider.selection read degrades to a nil Default
// with DefaultStatus naming the reason, because the catalog is still usable; a
// failed or malformed provider.management read returns an error and no catalog.
func (s *Session) Models(ctx context.Context) (ModelCatalog, error) {
	if s == nil {
		return ModelCatalog{}, fmt.Errorf("session: nil session")
	}
	if s.client == nil {
		return ModelCatalog{}, fmt.Errorf("session: nil client")
	}
	if !appserver.HasFeature(s.features, "provider.management") {
		return ModelCatalog{}, s.failModels(&appserver.CapabilityError{Feature: "provider.management"})
	}
	result, err := appserver.ProviderManagement(ctx, s.client, appserver.ProviderManagementParams{}, s.features)
	if err != nil {
		return ModelCatalog{}, s.failModels(err)
	}
	s.mu.Lock()
	current, hasCurrent := s.model, s.hasModel
	s.modelsStatus = ReadCapabilityStatus{Capability: "provider.management", Available: true}
	s.mu.Unlock()
	catalog := ModelCatalog{
		Models:     modelOptions(result),
		Current:    current,
		HasCurrent: hasCurrent,
	}
	if !appserver.HasFeature(s.features, "provider.selection") {
		catalog.DefaultStatus = ReadCapabilityStatus{
			Capability: "provider.selection",
			Available:  false,
			Reason:     "hello.features does not advertise provider.selection",
		}
		return catalog, nil
	}
	selection, err := appserver.ReadProviderSelection(ctx, s.client, appserver.ProviderSelectionParams{}, s.features)
	if err != nil {
		catalog.DefaultStatus = ReadCapabilityStatus{
			Capability: "provider.selection",
			Available:  false,
			Reason:     err.Error(),
		}
		return catalog, nil
	}
	catalog.DefaultStatus = ReadCapabilityStatus{Capability: "provider.selection", Available: true}
	if selection.Selection != nil {
		catalog.Default = &ModelSelection{
			ProviderID: selection.Selection.ProviderID,
			ModelID:    selection.Selection.ModelID,
		}
	}
	return catalog, nil
}

// failModels records the read failure for callers that report capability state.
func (s *Session) failModels(err error) error {
	reason := err.Error()
	s.mu.Lock()
	s.modelsStatus = ReadCapabilityStatus{Capability: "provider.management", Available: false, Reason: reason}
	s.mu.Unlock()
	return err
}

// ModelsStatus reports whether the last provider.management read succeeded and,
// if not, why.
func (s *Session) ModelsStatus() ReadCapabilityStatus {
	if s == nil {
		return ReadCapabilityStatus{Capability: "provider.management"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.modelsStatus
	if status.Capability == "" {
		status.Capability = "provider.management"
	}
	return status
}

// modelOptions flattens the Provider entries into one deterministic picker
// list. The order is derived from the daemon's provider and model names so
// repeated reads render the same list; no local preference reorders or hides an
// entry.
func modelOptions(result appserver.ProviderManagementResult) []ModelOption {
	options := make([]ModelOption, 0)
	for _, provider := range result.Providers {
		for _, model := range provider.Models {
			option := ModelOption{
				ProviderID:      provider.ProviderID,
				ProviderName:    provider.DisplayName,
				ProviderEnabled: cloneBool(provider.Enabled),
				ModelID:         model.Model.ModelID,
				DisplayName:     model.Model.DisplayName,
				Source:          model.Source,
				Enabled:         model.Enabled,
				MissingLimits:   append([]string(nil), model.MissingLimits...),
				ContextWindow:   cloneInt64(model.Model.Capabilities.ContextWindow),
				MaxOutputTokens: cloneInt64(model.Model.Capabilities.MaxOutputTokens),
				ToolCalling:     model.Model.Capabilities.ToolCalling,
				Vision:          model.Model.Capabilities.Vision,
				Reasoning:       model.Model.Capabilities.Reasoning,
			}
			if model.UnavailableReason != nil {
				option.UnavailableReason = *model.UnavailableReason
			}
			options = append(options, option)
		}
	}
	sort.SliceStable(options, func(i, j int) bool {
		left, right := options[i], options[j]
		if left.ProviderName != right.ProviderName {
			return left.ProviderName < right.ProviderName
		}
		if left.ProviderID != right.ProviderID {
			return left.ProviderID < right.ProviderID
		}
		if left.DisplayName != right.DisplayName {
			return left.DisplayName < right.DisplayName
		}
		return left.ModelID < right.ModelID
	})
	return options
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
