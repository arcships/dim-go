package supervisor

import (
	"encoding/json"
	"fmt"
)

// PingResult is daemon.ping's reply (protocol-contract §6). Connections is
// required because Close's ownership rule depends on it; a missing value is a
// protocol error rather than a zero that would read as "no other clients".
type PingResult struct {
	Sessions      *int   `json:"sessions,omitempty"`
	Connections   *int   `json:"connections"`
	ServerVersion string `json:"serverVersion,omitempty"`
}

// DecodePingResult decodes and validates a daemon.ping reply.
func DecodePingResult(raw json.RawMessage) (PingResult, error) {
	var result PingResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return result, fmt.Errorf("supervisor: daemon.ping result: invalid JSON: %w", err)
	}
	if result.Connections == nil {
		return result, fmt.Errorf("supervisor: daemon.ping result: missing connections")
	}
	if *result.Connections < 1 {
		return result, fmt.Errorf("supervisor: daemon.ping result: connections is %d, want at least 1", *result.Connections)
	}
	if result.Sessions != nil && *result.Sessions < 0 {
		return result, fmt.Errorf("supervisor: daemon.ping result: sessions is %d, want a non-negative integer", *result.Sessions)
	}
	return result, nil
}

// ShutdownParams is daemon.shutdown's request. Supervisor always sends
// force:false (protocol-contract §18).
type ShutdownParams struct {
	Force bool `json:"force"`
}
