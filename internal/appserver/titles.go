package appserver

import (
	"context"
	"encoding/json"
)

// session.rename and the SessionStore title notice are the public title surface
// of protocol v3. session.rename is a session-level method advertised by
// session.methods, not a hello.features capability (§16); the request/reply
// shape is the public SessionTitle A.1 contract:
//
//	session.rename {sessionId, title: string|null}
//	  -> {title: string|null, titleSource: "auto"|"manual"|null, revision}
//
// The owned title rules (trim, length, single line) belong to the method's
// provider, so this layer validates only shape and never rewrites the text.

const (
	// NoticeSourceSessionStore is the session.notice source that owns title
	// metadata (public SessionTitle A.11).
	NoticeSourceSessionStore = "SessionStore"
	// NoticeKindTitle is the session.notice kind carrying
	// {title, titleSource, titleRevision}.
	NoticeKindTitle = "title"
)

// RenameParams is the public session.rename request. Title is a pointer because
// null clears the title; the field is always serialized so a caller omission
// can never be read as a clear.
type RenameParams struct {
	SessionID string  `json:"sessionId"`
	Title     *string `json:"title"`
}

func (p RenameParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	if p.Title != nil && *p.Title == "" {
		// 1..maxLength after trimming is the provider's rule; an empty string is
		// never valid, so it is rejected here instead of spending a round trip.
		return protocolErrorf("title", "must not be empty")
	}
	return nil
}

// RenameResult is the public session.rename reply. Title and TitleSource are
// both JSON null when the rename cleared the title; Revision is the committed
// title revision.
type RenameResult struct {
	Title       *string `json:"title"`
	TitleSource *string `json:"titleSource"`
	Revision    int64   `json:"revision"`
}

func (r RenameResult) Validate() error {
	if r.Revision < 0 {
		return protocolErrorf("revision", "got %d, want a non-negative integer", r.Revision)
	}
	if r.TitleSource != nil && !oneOf(*r.TitleSource, TitleSourceAuto, TitleSourceManual) {
		return protocolErrorf("titleSource", "got %q, want auto or manual", *r.TitleSource)
	}
	if (r.Title == nil) != (r.TitleSource == nil) {
		return protocolErrorf("titleSource", "must be null exactly when title is null")
	}
	return nil
}

// requireObjectFields reports a reply that does not carry every field the
// contract lists. Nullable fields still have to be present: a clear sends
// explicit nulls, so an absent field is a malformed reply, not a clear.
func requireObjectFields(label string, raw json.RawMessage, fields ...string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return protocolErrorf(label, "invalid JSON object: %v", err)
	}
	if object == nil {
		return protocolErrorf(label, "missing object")
	}
	for _, field := range fields {
		if _, ok := object[field]; !ok {
			return protocolErrorf(label+"."+field, "missing")
		}
	}
	return nil
}

// DecodeRenameResult decodes and validates one session.rename reply.
func DecodeRenameResult(raw json.RawMessage) (RenameResult, error) {
	var r RenameResult
	if err := requireObjectFields("session.rename result", raw, "title", "titleSource", "revision"); err != nil {
		return r, err
	}
	if err := decodeValidated("session.rename result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// Rename sends session.rename through the session stream. There is no
// hello.features gate: session.rename is reachable only when the session's
// method table advertises it, and a daemon that does not returns unknown_method
// (mapped to a capability error, never retried).
//
// Rename is a write without an idempotency key. A timeout or a lost transport
// is an *UnknownResultError: read the title back through session.summaries and
// never resend the rename blindly (public SessionTitle Appendix C).
func Rename(ctx context.Context, client *Client, params RenameParams) (RenameResult, error) {
	if client == nil {
		return RenameResult{}, protocolErrorf("session.rename", "nil client")
	}
	if err := params.Validate(); err != nil {
		return RenameResult{}, err
	}
	raw, err := client.Call(ctx, string(MutationSessionRename), params)
	if err != nil {
		return RenameResult{}, mapMutationError(MutationSessionRename, err)
	}
	result, err := DecodeRenameResult(raw)
	if err != nil {
		// The rename was admitted but its reply does not satisfy the contract.
		// The committed title can only be read back, never inferred.
		return RenameResult{}, &UnknownResultError{
			Method:   MutationSessionRename,
			ReadBack: mutationReadBack(MutationSessionRename),
			Err:      err,
		}
	}
	return result, nil
}

// TitleNotice is the data of a session.notice carrying title metadata:
// {title: string|null, titleSource: "auto"|"manual"|null, titleRevision}.
type TitleNotice struct {
	Title         *string `json:"title"`
	TitleSource   *string `json:"titleSource"`
	TitleRevision int64   `json:"titleRevision"`
}

func (n TitleNotice) Validate() error {
	if n.TitleRevision < 0 {
		return protocolErrorf("titleRevision", "got %d, want a non-negative integer", n.TitleRevision)
	}
	if n.TitleSource != nil && !oneOf(*n.TitleSource, TitleSourceAuto, TitleSourceManual) {
		return protocolErrorf("titleSource", "got %q, want auto or manual", *n.TitleSource)
	}
	if (n.Title == nil) != (n.TitleSource == nil) {
		return protocolErrorf("titleSource", "must be null exactly when title is null")
	}
	return nil
}

// DecodeTitleNotice decodes the data of a SessionStore title notice. The three
// fields are required; a clear sends explicit nulls.
func DecodeTitleNotice(raw json.RawMessage) (TitleNotice, error) {
	var n TitleNotice
	if err := requireObjectFields("session.notice title data", raw, "title", "titleSource", "titleRevision"); err != nil {
		return n, err
	}
	if err := decodeValidated("session.notice title data", raw, &n); err != nil {
		return n, err
	}
	return n, nil
}

// IsTitleNotice reports whether a session.notice carries title metadata owned
// by SessionStore.
func IsTitleNotice(notice SessionNotice) bool {
	return notice.Source == NoticeSourceSessionStore && notice.Kind == NoticeKindTitle
}
