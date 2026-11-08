package appserver

import "context"

// SessionKeys reads the optional session.keys snapshot (§9): the latest live
// keyed block per key on one branch, anchored at params.Through.
//
// It is a best-effort read. Callers open the session even when it fails; the
// error is for degradation and logging, not for failing open. It performs
// exactly one request: no retry, no local fallback, and no second session
// store. On any error the result is zero: a partially decoded or partial
// snapshot must never seed state.
//
// Without the session.keys connection feature the daemon does not serve the
// method, so the helper returns *CapabilityError instead of sending a request
// and surfacing -32601. Extract Features from that connection's hello reply;
// each connection has its own snapshot (§16).
func SessionKeys(ctx context.Context, client *Client, params KeysParams, features []string) (KeysResult, error) {
	if !HasFeature(features, "session.keys") {
		return KeysResult{}, &CapabilityError{Feature: "session.keys"}
	}
	if err := params.Validate(); err != nil {
		return KeysResult{}, err
	}
	raw, err := client.Call(ctx, "session.keys", params)
	if err != nil {
		return KeysResult{}, err
	}
	result, err := DecodeKeysResult(raw)
	if err != nil {
		// A malformed snapshot must not partially seed state, so the result is
		// zeroed instead of half-decoded.
		return KeysResult{}, err
	}
	return result, nil
}
