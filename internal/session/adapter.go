package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/projection"
)

// AdaptEvent converts an app-server event to the projection event contract.
func AdaptEvent(e appserver.RawEvent) (projection.RawEvent, error) {
	if e.ID == "" {
		return projection.RawEvent{}, fmt.Errorf("session: event id empty: %w", appserver.ErrProtocol)
	}
	data := map[string]any{}
	raw := bytes.TrimSpace(e.Data)
	if len(raw) > 0 && !bytes.Equal(raw, []byte("null")) {
		if err := json.Unmarshal(raw, &data); err != nil {
			return projection.RawEvent{}, fmt.Errorf("session: event data: %w: %v", appserver.ErrProtocol, err)
		}
		if data == nil {
			data = map[string]any{}
		}
	}
	return projection.RawEvent{ID: e.ID, Sequence: e.Sequence, CreateAt: e.CreateAt, Type: e.Type, Name: e.Name, Data: data}, nil
}
