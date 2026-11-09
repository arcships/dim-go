package appserver

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// attachment.put is a public protocol-v3 session method with an upstream design
// contract but no DTO in Desktop's app-server-protocol wire.ts. The public
// client calls it from service.ts and only reads result.path; the response shape
// below comes from the upstream Attachments design contract. Evidence is
// fake-server (acceptance.md L1): no real-daemon exchange has been observed yet.
//
// The method is a mutation with no idempotency key and no list/get counterpart,
// so a timeout, disconnect, or malformed reply is reported as unknown and is
// never resent automatically (§14).

// MaxAttachmentTextBytes is the upstream UTF-8 text limit for attachment.put.
const MaxAttachmentTextBytes = 10 * 1024 * 1024

// attachmentFrameHeadroom reserves room in the 16 MiB frame for the JSON-RPC
// envelope (method name, id, jsonrpc) so a request that only just fits the text
// limit cannot exceed MaxFrameSize once encoded.
const attachmentFrameHeadroom = 1024

// AttachmentPutParams is the attachment.put request. Name is optional; the
// daemon defaults it to paste.txt and validates its own filename rules.
type AttachmentPutParams struct {
	SessionID string `json:"sessionId"`
	Name      string `json:"name,omitempty"`
	Text      string `json:"text"`
}

func (p AttachmentPutParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	if p.Name != "" && strings.TrimSpace(p.Name) == "" {
		return protocolErrorf("name", "must not be blank")
	}
	if p.Text == "" {
		return protocolErrorf("text", "must not be empty")
	}
	if len(p.Text) > MaxAttachmentTextBytes {
		return protocolErrorf("text", "got %d bytes, want at most %d", len(p.Text), MaxAttachmentTextBytes)
	}
	if !utf8.ValidString(p.Text) {
		return protocolErrorf("text", "must be valid UTF-8")
	}
	return nil
}

// AttachmentPutResult is the attachment.put response. The public client reads
// path only; attachmentId and bytes come from the upstream contract and are
// validated when present. Bytes is a pointer so a missing member is not read as
// a stored file of zero bytes.
type AttachmentPutResult struct {
	AttachmentID string `json:"attachmentId"`
	Path         string `json:"path"`
	Bytes        *int64 `json:"bytes,omitempty"`
}

func (r AttachmentPutResult) Validate() error {
	if err := requireNonEmpty("attachmentId", r.AttachmentID); err != nil {
		return err
	}
	if err := requireNonEmpty("path", r.Path); err != nil {
		return err
	}
	if r.Bytes != nil && *r.Bytes < 0 {
		return protocolErrorf("bytes", "got %d, want a non-negative integer", *r.Bytes)
	}
	return nil
}

// DecodeAttachmentPutResult decodes and validates one attachment.put response.
func DecodeAttachmentPutResult(raw json.RawMessage) (AttachmentPutResult, error) {
	var r AttachmentPutResult
	if err := decodeValidated("attachment.put result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// AttachmentPut uploads one text attachment and returns its stored path. The
// request is checked against both the 10 MiB UTF-8 limit and the 16 MiB frame
// limit before anything is sent. Exactly one request is made.
//
// There is no attachment list/get method in the contract, so a timeout,
// disconnect, or malformed result is an *UnknownResultError with no read-back:
// the file may have been stored, and the caller must ask the user instead of
// retrying. attachment.put has no hello.features gate; hosts without the
// Attachments plugin answer unknown_method, surfaced as
// *MutationError.Unsupported (visible-disabled WIP).
func AttachmentPut(ctx context.Context, client *Client, params AttachmentPutParams) (AttachmentPutResult, error) {
	if err := params.Validate(); err != nil {
		return AttachmentPutResult{}, err
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return AttachmentPutResult{}, protocolErrorf("attachment.put", "cannot encode request: %v", err)
	}
	if len(encoded)+attachmentFrameHeadroom > MaxFrameSize {
		return AttachmentPutResult{}, protocolErrorf("text",
			"encoded request is %d bytes, leaving no room under the %d-byte frame limit", len(encoded), MaxFrameSize)
	}
	if client == nil {
		return AttachmentPutResult{}, protocolErrorf("attachment.put", "nil client")
	}
	raw, err := client.Call(ctx, string(MutationAttachmentPut), params)
	if err != nil {
		return AttachmentPutResult{}, mapMutationError(MutationAttachmentPut, err)
	}
	result, err := DecodeAttachmentPutResult(raw)
	if err != nil {
		return AttachmentPutResult{}, &UnknownResultError{
			Method: MutationAttachmentPut,
			Err:    err,
		}
	}
	return result, nil
}
