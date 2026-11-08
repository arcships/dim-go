package appserver

import (
	"context"
	"errors"
)

// ErrCapabilityUnavailable marks a read helper that cannot run because the
// connection's hello.features snapshot lacks a required capability. It is
// deliberately not an empty result: callers must disable the entry point and
// explain why.
var ErrCapabilityUnavailable = errors.New("appserver: capability unavailable")

// CapabilityError names the missing feature.
type CapabilityError struct {
	Feature string
}

func (e *CapabilityError) Error() string {
	return "appserver: capability unavailable: " + e.Feature
}

func (e *CapabilityError) Unwrap() error { return ErrCapabilityUnavailable }

// maxCatalogPages bounds one traversal so a misbehaving daemon cannot loop
// forever. Cursor cycles are rejected earlier.
const maxCatalogPages = 10000

// Catalog fetches every session.catalog page sequentially and merges them.
//
// It omits withInternal/withAcpSessions when session.catalog.origins is absent
// from features, and if the daemon still answers -32602 while those params are
// set it strips them and retries exactly once. A list cwd needs
// session.catalog.workspaces; without it Catalog returns *CapabilityError
// instead of degrading the request locally. A missing session.catalog feature
// returns *CapabilityError, never an empty list.
//
// When an error is returned the result may hold the pages fetched so far; it is
// not a complete snapshot and callers must not present it as one.
func Catalog(ctx context.Context, client *Client, params CatalogParams, features []string) (CatalogResult, error) {
	if !HasFeature(features, "session.catalog") {
		return CatalogResult{}, &CapabilityError{Feature: "session.catalog"}
	}
	if err := params.Validate(); err != nil {
		return CatalogResult{}, err
	}
	if params.Cwd != nil && params.Cwd.IsList && !HasFeature(features, "session.catalog.workspaces") {
		return CatalogResult{}, &CapabilityError{Feature: "session.catalog.workspaces"}
	}
	if !HasFeature(features, "session.catalog.origins") {
		params.WithInternal = nil
		params.WithAcpSessions = nil
	}
	merged := CatalogResult{Sessions: []SessionSummary{}}
	originsRetried := false
	seen := map[string]bool{params.Cursor: true}
	for page := 0; page < maxCatalogPages; page++ {
		raw, err := client.Call(ctx, "session.catalog", params)
		if err != nil {
			var rpcErr *RPCError
			originsSet := params.WithInternal != nil || params.WithAcpSessions != nil
			if !originsRetried && originsSet && errors.As(err, &rpcErr) && rpcErr.Code == -32602 {
				params.WithInternal = nil
				params.WithAcpSessions = nil
				originsRetried = true
				continue
			}
			return merged, err
		}
		result, err := DecodeCatalogResult(raw)
		if err != nil {
			return merged, err
		}
		merged.Sessions = append(merged.Sessions, result.Sessions...)
		cursor, ok := result.Next()
		if !ok {
			return merged, nil
		}
		if seen[cursor] {
			return merged, protocolErrorf("nextCursor", "repeated cursor %q", cursor)
		}
		seen[cursor] = true
		params.Cursor = cursor
	}
	return merged, protocolErrorf("session.catalog", "pagination exceeded %d pages", maxCatalogPages)
}
