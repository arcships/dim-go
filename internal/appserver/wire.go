package appserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// Wire protocol v3 constants from docs/protocol-contract.md.
const (
	WireProtocolVersion = 3

	ClientKindCLI   = "cli"
	ClientKindWeb   = "web"
	ClientKindOther = "other"

	DaemonProfileProduct    = "product"
	DaemonProfileConfigured = "configured"
	DaemonProfileMock       = "mock"

	OriginClient   = "client"
	OriginACP      = "acp"
	OriginInternal = "internal"
	OriginUnknown  = "unknown"

	TitleSourceAuto   = "auto"
	TitleSourceManual = "manual"

	SubmitModeQueue  = "queue"
	SubmitModeSteer  = "steer"
	SubmitModeAppend = "append"

	DecisionAllow = "allow"
	DecisionDeny  = "deny"
)

// ErrProtocol marks a payload that violates docs/protocol-contract.md.
var ErrProtocol = errors.New("appserver: protocol error")

// ProtocolError reports a wire payload that does not match the contract. Use
// errors.Is(err, ErrProtocol) to detect it without matching a concrete field.
type ProtocolError struct {
	Field  string
	Detail string
}

func (e *ProtocolError) Error() string {
	if e.Field == "" {
		return "appserver: protocol error: " + e.Detail
	}
	return "appserver: protocol error: " + e.Field + ": " + e.Detail
}

func (e *ProtocolError) Unwrap() error { return ErrProtocol }

func protocolErrorf(field, format string, args ...any) error {
	return &ProtocolError{Field: field, Detail: fmt.Sprintf(format, args...)}
}

func requireNonEmpty(field, value string) error {
	if value == "" {
		return protocolErrorf(field, "must not be empty")
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func hasJSONValue(raw json.RawMessage) bool {
	raw = bytes.TrimSpace(raw)
	return len(raw) > 0 && !bytes.Equal(raw, []byte("null"))
}

// validator is implemented by every payload type that can be checked against
// the contract.
type validator interface {
	Validate() error
}

func decodeValidated(label string, raw json.RawMessage, v validator) error {
	if err := json.Unmarshal(raw, v); err != nil {
		return protocolErrorf(label, "invalid JSON: %v", err)
	}
	return v.Validate()
}

// HasFeature reports whether a hello.features snapshot contains name.
func HasFeature(features []string, name string) bool {
	for _, feature := range features {
		if feature == name {
			return true
		}
	}
	return false
}

// ----- hello (§6) -----

// HelloParams is the hello request sent on every connection.
type HelloParams struct {
	ProtocolVersion int    `json:"protocolVersion"`
	ClientKind      string `json:"clientKind"`
	ClientName      string `json:"clientName"`
}

func (p HelloParams) Validate() error {
	if p.ProtocolVersion != WireProtocolVersion {
		return protocolErrorf("protocolVersion", "got %d, want %d", p.ProtocolVersion, WireProtocolVersion)
	}
	if !oneOf(p.ClientKind, ClientKindCLI, ClientKindWeb, ClientKindOther) {
		return protocolErrorf("clientKind", "got %q, want cli, web or other", p.ClientKind)
	}
	return requireNonEmpty("clientName", p.ClientName)
}

// DaemonProfile is the discriminated union from hello.
type DaemonProfile struct {
	Kind              string `json:"kind"`
	ModelConfigSHA256 string `json:"modelConfigSha256,omitempty"`
}

func (p DaemonProfile) Validate() error {
	switch p.Kind {
	case DaemonProfileProduct, DaemonProfileMock:
		return nil
	case DaemonProfileConfigured:
		return requireNonEmpty("daemonProfile.modelConfigSha256", p.ModelConfigSHA256)
	default:
		return protocolErrorf("daemonProfile.kind", "got %q, want product, configured or mock", p.Kind)
	}
}

type HelloResult struct {
	InstanceID      string        `json:"instanceId"`
	ServerVersion   string        `json:"serverVersion"`
	ProtocolVersion int           `json:"protocolVersion"`
	DaemonProfile   DaemonProfile `json:"daemonProfile"`
	Features        []string      `json:"features,omitempty"`
}

func (r HelloResult) Validate() error {
	if r.ProtocolVersion != WireProtocolVersion {
		return protocolErrorf("protocolVersion", "got %d, want %d", r.ProtocolVersion, WireProtocolVersion)
	}
	if err := requireNonEmpty("instanceId", r.InstanceID); err != nil {
		return err
	}
	if err := r.DaemonProfile.Validate(); err != nil {
		return err
	}
	for i, feature := range r.Features {
		if feature == "" {
			return protocolErrorf(fmt.Sprintf("features[%d]", i), "must not be empty")
		}
	}
	return nil
}

func (r HelloResult) HasFeature(name string) bool { return HasFeature(r.Features, name) }

// DecodeHelloResult decodes and validates a hello response.
func DecodeHelloResult(raw json.RawMessage) (HelloResult, error) {
	var r HelloResult
	if err := decodeValidated("hello result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// ----- session.catalog (§7.1) -----

// CwdParam is session.catalog's cwd: a single workspace string or an array of
// workspace strings.
type CwdParam struct {
	Paths  []string
	IsList bool
}

// SingleCwd builds a scalar cwd.
func SingleCwd(path string) *CwdParam { return &CwdParam{Paths: []string{path}} }

// CwdList builds an array cwd.
func CwdList(paths ...string) *CwdParam {
	return &CwdParam{Paths: append([]string{}, paths...), IsList: true}
}

func (c CwdParam) MarshalJSON() ([]byte, error) {
	if c.IsList {
		return json.Marshal(c.Paths)
	}
	if len(c.Paths) == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(c.Paths[0])
}

func (c *CwdParam) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	*c = CwdParam{}
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil
	}
	if raw[0] == '[' {
		if err := json.Unmarshal(raw, &c.Paths); err != nil {
			return protocolErrorf("cwd", "must be a string or an array of strings: %v", err)
		}
		c.IsList = true
		return nil
	}
	var path string
	if err := json.Unmarshal(raw, &path); err != nil {
		return protocolErrorf("cwd", "must be a string or an array of strings: %v", err)
	}
	c.Paths = []string{path}
	return nil
}

func (c CwdParam) Validate() error {
	if len(c.Paths) == 0 {
		return protocolErrorf("cwd", "must contain at least one workspace")
	}
	for i, path := range c.Paths {
		if path == "" {
			return protocolErrorf(fmt.Sprintf("cwd[%d]", i), "must not be empty")
		}
	}
	return nil
}

type CatalogParams struct {
	Limit           *int      `json:"limit,omitempty"`
	Cursor          string    `json:"cursor,omitempty"`
	Query           string    `json:"query,omitempty"`
	Cwd             *CwdParam `json:"cwd,omitempty"`
	WithPreview     *bool     `json:"withPreview,omitempty"`
	WithInternal    *bool     `json:"withInternal,omitempty"`
	WithAcpSessions *bool     `json:"withAcpSessions,omitempty"`
}

func (p CatalogParams) Validate() error {
	if p.Limit != nil && *p.Limit <= 0 {
		return protocolErrorf("limit", "got %d, want a positive integer", *p.Limit)
	}
	if p.Cwd != nil {
		return p.Cwd.Validate()
	}
	return nil
}

// SessionModel is the model pair carried by summaries and attach replies.
type SessionModel struct {
	ProviderID string `json:"providerId"`
	ModelID    string `json:"modelId"`
}

// SessionSummary is the minimal catalog row from §7.1.
type SessionSummary struct {
	SessionID       string       `json:"sessionId"`
	Cwd             string       `json:"cwd"`
	Model           SessionModel `json:"model"`
	CreatedAt       int64        `json:"createdAt"`
	Running         bool         `json:"running"`
	Sequence        int64        `json:"sequence"`
	AttachedClients int          `json:"attachedClients"`
	UpdatedAt       int64        `json:"updatedAt"`
	AgentID         string       `json:"agentId,omitempty"`
	Origin          string       `json:"origin,omitempty"`
	ParentSessionID string       `json:"parentSessionId,omitempty"`
	Title           *string      `json:"title,omitempty"`
	TitleSource     *string      `json:"titleSource,omitempty"`
	TitleRevision   *int64       `json:"titleRevision,omitempty"`
	Preview         *string      `json:"preview,omitempty"`
	Closed          *bool        `json:"closed,omitempty"`
}

func (s SessionSummary) Validate() error {
	if err := requireNonEmpty("sessionId", s.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("cwd", s.Cwd); err != nil {
		return err
	}
	if err := requireNonEmpty("model.providerId", s.Model.ProviderID); err != nil {
		return err
	}
	if err := requireNonEmpty("model.modelId", s.Model.ModelID); err != nil {
		return err
	}
	if s.Sequence < 0 {
		return protocolErrorf("sequence", "got %d, want a non-negative integer", s.Sequence)
	}
	if s.AttachedClients < 0 {
		return protocolErrorf("attachedClients", "got %d, want a non-negative integer", s.AttachedClients)
	}
	if s.Origin != "" && !oneOf(s.Origin, OriginClient, OriginACP, OriginInternal, OriginUnknown) {
		return protocolErrorf("origin", "got %q, want client, acp, internal or unknown", s.Origin)
	}
	if s.TitleSource != nil && !oneOf(*s.TitleSource, TitleSourceAuto, TitleSourceManual) {
		return protocolErrorf("titleSource", "got %q, want auto or manual", *s.TitleSource)
	}
	return nil
}

type CatalogResult struct {
	Sessions   []SessionSummary `json:"sessions"`
	NextCursor *string          `json:"nextCursor,omitempty"`
}

func (r CatalogResult) Validate() error {
	if r.Sessions == nil {
		return protocolErrorf("sessions", "missing array")
	}
	for i := range r.Sessions {
		if err := r.Sessions[i].Validate(); err != nil {
			return fmt.Errorf("sessions[%d]: %w", i, err)
		}
	}
	return nil
}

// Next returns the next page cursor and whether another request should be
// made. A missing or empty cursor ends the traversal (§7.1).
func (r CatalogResult) Next() (string, bool) {
	if r.NextCursor == nil || *r.NextCursor == "" {
		return "", false
	}
	return *r.NextCursor, true
}

// DecodeCatalogResult decodes and validates one session.catalog page.
func DecodeCatalogResult(raw json.RawMessage) (CatalogResult, error) {
	var r CatalogResult
	if err := decodeValidated("session.catalog result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// ----- session.summaries (§7.2) -----

type SummariesParams struct {
	IncludeClosed bool `json:"includeClosed,omitempty"`
}

// DecodeSummaries decodes session.summaries' bare array response.
func DecodeSummaries(raw json.RawMessage) ([]SessionSummary, error) {
	var summaries []SessionSummary
	if err := json.Unmarshal(raw, &summaries); err != nil {
		return nil, protocolErrorf("session.summaries result", "invalid JSON array: %v", err)
	}
	if summaries == nil {
		return nil, protocolErrorf("session.summaries result", "missing array")
	}
	for i := range summaries {
		if err := summaries[i].Validate(); err != nil {
			return nil, fmt.Errorf("summaries[%d]: %w", i, err)
		}
	}
	return summaries, nil
}

// ----- session.attach (§8) -----

type AttachParams struct {
	SessionID string `json:"sessionId"`
}

func (p AttachParams) Validate() error { return requireNonEmpty("sessionId", p.SessionID) }

// DetachParams is the session.detach request (§8). The reply is ignored: the
// contract treats closing the connection as what releases the daemon-side
// attachment.
type DetachParams struct {
	SessionID string `json:"sessionId"`
}

func (p DetachParams) Validate() error { return requireNonEmpty("sessionId", p.SessionID) }

type ModelPreferences struct {
	ReasoningEffort string `json:"reasoningEffort,omitempty"`
	ServiceTier     string `json:"serviceTier,omitempty"`
}

type AttachResult struct {
	Sequence         *int64            `json:"sequence,omitempty"`
	Running          *bool             `json:"running,omitempty"`
	Model            *SessionModel     `json:"model,omitempty"`
	ModelPreferences *ModelPreferences `json:"modelPreferences,omitempty"`
}

func (r AttachResult) Validate() error {
	if r.Sequence != nil && *r.Sequence < 0 {
		return protocolErrorf("sequence", "got %d, want a non-negative integer", *r.Sequence)
	}
	return nil
}

// DecodeAttachResult decodes an attach response. Sequence may be absent on an
// older daemon; callers must fall back to session.history.
func DecodeAttachResult(raw json.RawMessage) (AttachResult, error) {
	var r AttachResult
	if err := decodeValidated("session.attach result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// ----- session.history (§9) -----

type HistoryParams struct {
	SessionID  string `json:"sessionId"`
	BranchID   string `json:"branchId,omitempty"`
	After      *int64 `json:"after,omitempty"`
	Limit      *int   `json:"limit,omitempty"`
	Through    *int64 `json:"through,omitempty"`
	Before     *int64 `json:"before,omitempty"`
	LatestTurn *bool  `json:"latestTurn,omitempty"`
}

func (p HistoryParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	if p.After != nil && *p.After < 0 {
		return protocolErrorf("after", "got %d, want a non-negative integer", *p.After)
	}
	if p.Through != nil && *p.Through < 0 {
		return protocolErrorf("through", "got %d, want a non-negative integer", *p.Through)
	}
	if p.Before != nil && *p.Before < 0 {
		return protocolErrorf("before", "got %d, want a non-negative integer", *p.Before)
	}
	if p.Limit != nil && *p.Limit <= 0 {
		return protocolErrorf("limit", "got %d, want a positive integer", *p.Limit)
	}
	return nil
}

// RawEvent is one durable timeline event. Data stays raw because the wire layer
// does not validate Rust EventKind payload shapes.
type RawEvent struct {
	ID       string          `json:"id"`
	Sequence int64           `json:"sequence"`
	CreateAt int64           `json:"create_at"`
	Tags     []string        `json:"tags,omitempty"`
	Type     string          `json:"type"`
	Name     string          `json:"name"`
	Data     json.RawMessage `json:"data,omitempty"`
}

func (e RawEvent) Validate() error {
	if err := requireNonEmpty("id", e.ID); err != nil {
		return err
	}
	if e.Sequence <= 0 {
		return protocolErrorf("sequence", "got %d, want a positive integer", e.Sequence)
	}
	if e.CreateAt <= 0 {
		return protocolErrorf("create_at", "got %d, want epoch milliseconds", e.CreateAt)
	}
	if err := requireNonEmpty("type", e.Type); err != nil {
		return err
	}
	return requireNonEmpty("name", e.Name)
}

// HistoryResult is one history page. Through and NextAfter are required by the
// contract, so they are pointers: a missing member must fail fast, not decode
// as zero.
type HistoryResult struct {
	Events     []RawEvent `json:"events"`
	Through    *int64     `json:"through"`
	NextAfter  *int64     `json:"nextAfter"`
	Done       bool       `json:"done"`
	NextBefore *int64     `json:"nextBefore,omitempty"`
	TurnStart  *int64     `json:"turnStart,omitempty"`
}

// Validate checks the §9 paging invariants against the request that produced
// this page:
//
//  1. after < sequence <= through for every event;
//  2. after <= nextAfter <= through;
//  3. nextAfter > after unless done;
//  4. a request-supplied through is echoed unchanged.
func (r HistoryResult) Validate(req HistoryParams) error {
	if r.Events == nil {
		return protocolErrorf("events", "missing array")
	}
	if r.Through == nil {
		return protocolErrorf("through", "missing")
	}
	if r.NextAfter == nil {
		return protocolErrorf("nextAfter", "missing")
	}
	if req.Through != nil && *req.Through != *r.Through {
		return protocolErrorf("through", "echoed %d, request sent %d", *r.Through, *req.Through)
	}
	var after int64
	if req.After != nil {
		after = *req.After
	}
	through := *r.Through
	if through < 0 {
		return protocolErrorf("through", "got %d, want a non-negative integer", through)
	}
	for i := range r.Events {
		event := r.Events[i]
		if err := event.Validate(); err != nil {
			return fmt.Errorf("events[%d]: %w", i, err)
		}
		if event.Sequence <= after {
			return protocolErrorf(fmt.Sprintf("events[%d].sequence", i), "got %d, want greater than after=%d", event.Sequence, after)
		}
		if event.Sequence > through {
			return protocolErrorf(fmt.Sprintf("events[%d].sequence", i), "got %d, want at most through=%d", event.Sequence, through)
		}
	}
	nextAfter := *r.NextAfter
	if nextAfter < after || nextAfter > through {
		return protocolErrorf("nextAfter", "got %d, want between after=%d and through=%d", nextAfter, after, through)
	}
	if !r.Done && nextAfter <= after {
		return protocolErrorf("nextAfter", "got %d, want greater than after=%d while done is false", nextAfter, after)
	}
	return nil
}

// DecodeHistoryResult decodes and validates one history page against its
// request.
func DecodeHistoryResult(raw json.RawMessage, req HistoryParams) (HistoryResult, error) {
	var r HistoryResult
	if err := json.Unmarshal(raw, &r); err != nil {
		return r, protocolErrorf("session.history result", "invalid JSON: %v", err)
	}
	if err := r.Validate(req); err != nil {
		return r, err
	}
	return r, nil
}

// ----- session.keys (public Desktop wire shape) -----

// session.keys is a public protocol-v3 read: Desktop's app-server-protocol
// wire.ts declares SessionKeyEntry/SessionKeysResult and session-reader.ts
// opens it on the session stream when hello.features advertises "session.keys".

// KeysParams is the request session-reader.ts issues: {sessionId, branchId,
// through}. Desktop always supplies all three, so nothing is defaulted here; a
// guessed branch or window would seed a different branch's keys.
type KeysParams struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	Through   *int64 `json:"through"`
}

func (p KeysParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("branchId", p.BranchID); err != nil {
		return err
	}
	if p.Through == nil {
		return protocolErrorf("through", "missing")
	}
	if *p.Through < 0 {
		return protocolErrorf("through", "got %d, want a non-negative integer", *p.Through)
	}
	return nil
}

// SessionKeyEntry is the latest live keyed block for one key on a branch. Text
// is the block body and may be empty; the wire layer does not interpret it.
type SessionKeyEntry struct {
	Key      string `json:"key"`
	Sequence int64  `json:"sequence"`
	InputID  string `json:"inputId"`
	Text     string `json:"text"`
}

// Validate enforces the wire.ts entry shape. Entry order and per-key uniqueness
// are daemon guarantees the client consumes as-is, so they are not client-side
// checks.
func (e SessionKeyEntry) Validate() error {
	if err := requireNonEmpty("key", e.Key); err != nil {
		return err
	}
	if e.Sequence <= 0 {
		return protocolErrorf("sequence", "got %d, want a positive integer", e.Sequence)
	}
	return requireNonEmpty("inputId", e.InputID)
}

// KeysResult is the session.keys response. Through is a pointer so a missing
// member fails fast instead of decoding as the zero sequence.
type KeysResult struct {
	Through *int64            `json:"through"`
	Entries []SessionKeyEntry `json:"entries"`
}

func (r KeysResult) Validate() error {
	if r.Through == nil {
		return protocolErrorf("through", "missing")
	}
	if *r.Through < 0 {
		return protocolErrorf("through", "got %d, want a non-negative integer", *r.Through)
	}
	if r.Entries == nil {
		return protocolErrorf("entries", "missing array")
	}
	for i := range r.Entries {
		if err := r.Entries[i].Validate(); err != nil {
			return fmt.Errorf("entries[%d]: %w", i, err)
		}
	}
	return nil
}

// DecodeKeysResult decodes and validates one session.keys response.
func DecodeKeysResult(raw json.RawMessage) (KeysResult, error) {
	var r KeysResult
	if err := decodeValidated("session.keys result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

// ----- session.submit / session.stop (§10) -----

type SubmitParams struct {
	SessionID    string          `json:"sessionId"`
	SubmissionID string          `json:"submissionId"`
	BranchID     string          `json:"branchId"`
	Text         string          `json:"text,omitempty"`
	User         json.RawMessage `json:"user,omitempty"`
	Mode         string          `json:"mode,omitempty"`
}

func (p SubmitParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("submissionId", p.SubmissionID); err != nil {
		return err
	}
	if err := requireNonEmpty("branchId", p.BranchID); err != nil {
		return err
	}
	if p.Text == "" && !hasJSONValue(p.User) {
		return protocolErrorf("text", "text or user must be set")
	}
	if p.Mode != "" && !oneOf(p.Mode, SubmitModeQueue, SubmitModeSteer, SubmitModeAppend) {
		return protocolErrorf("mode", "got %q, want queue, steer or append", p.Mode)
	}
	return nil
}

type SubmitResult struct {
	Event RawEvent `json:"event"`
}

func (r SubmitResult) Validate() error { return r.Event.Validate() }

// DecodeSubmitResult decodes the minimum session.submit response.
func DecodeSubmitResult(raw json.RawMessage) (SubmitResult, error) {
	var r SubmitResult
	if err := decodeValidated("session.submit result", raw, &r); err != nil {
		return r, err
	}
	return r, nil
}

type StopParams struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	Reason    string `json:"reason,omitempty"`
}

func (p StopParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	return requireNonEmpty("branchId", p.BranchID)
}

// ----- approval.respond (§12) -----

// ApprovalRespondParams collapses the UI options onto the wire decision before
// it reaches this type: allow_once/allow_all become allow, everything else
// becomes deny (§12).
type ApprovalRespondParams struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	RequestID string `json:"requestId"`
	Decision  string `json:"decision"`
}

func (p ApprovalRespondParams) Validate() error {
	if err := requireNonEmpty("sessionId", p.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("branchId", p.BranchID); err != nil {
		return err
	}
	if err := requireNonEmpty("requestId", p.RequestID); err != nil {
		return err
	}
	if !oneOf(p.Decision, DecisionAllow, DecisionDeny) {
		return protocolErrorf("decision", "got %q, want allow or deny", p.Decision)
	}
	return nil
}

// ----- session.delete (§5) -----

// DeleteParams is the minimal session.delete payload. docs/protocol-contract.md
// names session.delete on the control connection and its session_active refusal
// (§13), but defines neither additional request members nor a response body, so
// the Go client sends only sessionId and ignores the reply. Callers that need to
// know whether the row is gone must read session.catalog back.
type DeleteParams struct {
	SessionID string `json:"sessionId"`
}

func (p DeleteParams) Validate() error { return requireNonEmpty("sessionId", p.SessionID) }

// ----- notifications (§11) -----

type EventNotice struct {
	SessionID string   `json:"sessionId"`
	BranchID  string   `json:"branchId"`
	Event     RawEvent `json:"event"`
}

func (n EventNotice) Validate() error {
	if err := requireNonEmpty("sessionId", n.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("branchId", n.BranchID); err != nil {
		return err
	}
	return n.Event.Validate()
}

type StateNotice struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	Running   bool   `json:"running"`
	Stopping  bool   `json:"stopping"`
}

func (n StateNotice) Validate() error {
	if err := requireNonEmpty("sessionId", n.SessionID); err != nil {
		return err
	}
	return requireNonEmpty("branchId", n.BranchID)
}

type ApprovalSource struct {
	Kind   string `json:"kind"`
	CallID string `json:"callId,omitempty"`
}

type ApprovalCall struct {
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input,omitempty"`
}

type ApprovalRequest struct {
	SessionID string         `json:"sessionId"`
	BranchID  string         `json:"branchId"`
	RequestID string         `json:"requestId"`
	Source    ApprovalSource `json:"source"`
	Call      ApprovalCall   `json:"call"`
	Reason    string         `json:"reason,omitempty"`
}

func (n ApprovalRequest) Validate() error {
	if err := requireNonEmpty("sessionId", n.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("branchId", n.BranchID); err != nil {
		return err
	}
	if err := requireNonEmpty("requestId", n.RequestID); err != nil {
		return err
	}
	if err := requireNonEmpty("source.kind", n.Source.Kind); err != nil {
		return err
	}
	return requireNonEmpty("call.name", n.Call.Name)
}

type ApprovalResolved struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	RequestID string `json:"requestId"`
	Decision  string `json:"decision"`
}

func (n ApprovalResolved) Validate() error {
	if err := requireNonEmpty("sessionId", n.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("branchId", n.BranchID); err != nil {
		return err
	}
	if err := requireNonEmpty("requestId", n.RequestID); err != nil {
		return err
	}
	if !oneOf(n.Decision, DecisionAllow, DecisionDeny) {
		return protocolErrorf("decision", "got %q, want allow or deny", n.Decision)
	}
	return nil
}

type ApprovalCancelled struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	RequestID string `json:"requestId"`
	Reason    string `json:"reason,omitempty"`
}

func (n ApprovalCancelled) Validate() error {
	if err := requireNonEmpty("sessionId", n.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("branchId", n.BranchID); err != nil {
		return err
	}
	return requireNonEmpty("requestId", n.RequestID)
}

type SessionNotice struct {
	SessionID string          `json:"sessionId"`
	Source    string          `json:"source"`
	Kind      string          `json:"kind"`
	Data      json.RawMessage `json:"data,omitempty"`
}

func (n SessionNotice) Validate() error {
	if err := requireNonEmpty("sessionId", n.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("source", n.Source); err != nil {
		return err
	}
	return requireNonEmpty("kind", n.Kind)
}

type ClosedNotice struct {
	SessionID string `json:"sessionId"`
	Reason    string `json:"reason,omitempty"`
}

func (n ClosedNotice) Validate() error { return requireNonEmpty("sessionId", n.SessionID) }

type ModelProgress struct {
	SessionID      string            `json:"sessionId"`
	BranchID       string            `json:"branchId"`
	TurnID         string            `json:"turnId,omitempty"`
	RequestID      string            `json:"requestId,omitempty"`
	TextPartID     string            `json:"textPartId,omitempty"`
	ThinkingPartID string            `json:"thinkingPartId,omitempty"`
	Text           string            `json:"text,omitempty"`
	TextOffset     int64             `json:"textOffset"`
	Thinking       string            `json:"thinking,omitempty"`
	ThinkingOffset int64             `json:"thinkingOffset"`
	Tools          []json.RawMessage `json:"tools,omitempty"`
}

func (n ModelProgress) Validate() error {
	if err := requireNonEmpty("sessionId", n.SessionID); err != nil {
		return err
	}
	if err := requireNonEmpty("branchId", n.BranchID); err != nil {
		return err
	}
	if n.TextOffset < 0 {
		return protocolErrorf("textOffset", "got %d, want a non-negative integer", n.TextOffset)
	}
	if n.ThinkingOffset < 0 {
		return protocolErrorf("thinkingOffset", "got %d, want a non-negative integer", n.ThinkingOffset)
	}
	return nil
}

// DecodeNotification decodes a notification with a known §11 method into its
// typed payload. known is false for methods outside the contract; callers keep
// Notification.Params for those (unknown methods pass through verbatim).
func DecodeNotification(n Notification) (payload any, known bool, err error) {
	switch n.Method {
	case "session.event":
		var v EventNotice
		if err := decodeValidated("session.event params", n.Params, &v); err != nil {
			return nil, true, err
		}
		return v, true, nil
	case "session.state":
		var v StateNotice
		if err := decodeValidated("session.state params", n.Params, &v); err != nil {
			return nil, true, err
		}
		return v, true, nil
	case "approval.request":
		var v ApprovalRequest
		if err := decodeValidated("approval.request params", n.Params, &v); err != nil {
			return nil, true, err
		}
		return v, true, nil
	case "approval.resolved":
		var v ApprovalResolved
		if err := decodeValidated("approval.resolved params", n.Params, &v); err != nil {
			return nil, true, err
		}
		return v, true, nil
	case "approval.cancelled":
		var v ApprovalCancelled
		if err := decodeValidated("approval.cancelled params", n.Params, &v); err != nil {
			return nil, true, err
		}
		return v, true, nil
	case "session.notice":
		var v SessionNotice
		if err := decodeValidated("session.notice params", n.Params, &v); err != nil {
			return nil, true, err
		}
		return v, true, nil
	case "session.closed":
		var v ClosedNotice
		if err := decodeValidated("session.closed params", n.Params, &v); err != nil {
			return nil, true, err
		}
		return v, true, nil
	case "session.model.progress":
		var v ModelProgress
		if err := decodeValidated("session.model.progress params", n.Params, &v); err != nil {
			return nil, true, err
		}
		return v, true, nil
	default:
		return nil, false, nil
	}
}
