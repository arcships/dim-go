package appserver

import "context"

// SessionKeys reads the optional session.keys snapshot: the latest live keyed
// block per key on one branch, anchored at params.Through.
//
// Desktop (session-reader.ts:1041-1045) gates the same request on hello.features
// "session.keys" and caps it at 5s; it degrades on failure instead of failing
// Session open. This helper returns the error and leaves degradation to the
// caller, performs exactly one request, and returns a zero result on any error
// so a partially decoded snapshot cannot seed state.
func SessionKeys(ctx context.Context, client *Client, params KeysParams, features []string) (KeysResult, error) {
	if !HasFeature(features, "session.keys") {
		return KeysResult{}, &CapabilityError{Feature: "session.keys"}
	}
	if err := params.Validate(); err != nil {
		return KeysResult{}, err
	}
	if client == nil {
		return KeysResult{}, protocolErrorf("session.keys", "nil client")
	}
	raw, err := client.Call(ctx, "session.keys", params)
	if err != nil {
		return KeysResult{}, err
	}
	result, err := DecodeKeysResult(raw)
	if err != nil {
		return KeysResult{}, err
	}
	return result, nil
}
