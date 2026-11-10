package session

import (
	"context"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// AttachmentPut stores one text attachment for the selected session and returns
// the path the App Server reports. The request is sent once: attachment.put has
// no idempotency key and no list/get counterpart, so a lost or malformed reply
// is reported as unknown and never resent. A host without the Attachments plugin
// answers unknown_method, which the helper surfaces as a capability error so the
// caller can keep the entry a visible WIP gap.
func (s *Session) AttachmentPut(ctx context.Context, name, text string) (appserver.AttachmentPutResult, error) {
	if s == nil {
		return appserver.AttachmentPutResult{}, fmt.Errorf("session: nil session")
	}
	if s.client == nil {
		return appserver.AttachmentPutResult{}, fmt.Errorf("session: nil client")
	}
	return appserver.AttachmentPut(ctx, s.client, appserver.AttachmentPutParams{
		SessionID: s.sessionID,
		Name:      name,
		Text:      text,
	})
}
