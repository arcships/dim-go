// Package projection contains the deterministic, in-memory timeline projection.
package projection

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Clock is injected so projections never read the wall clock.
type Clock interface{ NowMillis() int64 }

// FixedClock is the deterministic Clock used by tests and fixtures.
type FixedClock int64

func (c FixedClock) NowMillis() int64 { return int64(c) }

// RawEvent is the minimal App Server wire event consumed by the projection.
// Sequence is the dedupe key; ID is provenance and message identity.
type RawEvent struct {
	ID       string         `json:"id"`
	Sequence int64          `json:"sequence"`
	CreateAt int64          `json:"create_at"`
	Type     string         `json:"type"`
	Name     string         `json:"name"`
	Data     map[string]any `json:"data"`
}

// Event is one timeline block owned by a Message.
type Event struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Status     string         `json:"status"`
	Content    string         `json:"content,omitempty"`
	ToolName   string         `json:"toolName,omitempty"`
	ToolParams map[string]any `json:"toolParams,omitempty"`
	StartTime  int64          `json:"startTime"`
	EndTime    *int64         `json:"endTime,omitempty"`
}

// Message is durable timeline view state. It is never persisted locally.
type Message struct {
	ID        string  `json:"id"`
	OrderKey  string  `json:"orderKey"`
	Role      string  `json:"role"`
	Content   string  `json:"content"`
	Status    string  `json:"status"`
	CreatedAt int64   `json:"createdAt"`
	UpdatedAt int64   `json:"updatedAt"`
	Events    []Event `json:"events"`
}

// Snapshot is the durable output of the timeline reducer.
type Snapshot struct {
	Messages               []Message `json:"messages"`
	TimelineUserMessageIds []string  `json:"timelineUserMessageIds"`
}

type pendingToolResult struct {
	sequence int64
	at       int64
	result   any
}

// Reducer is the single fact entry for one branch's durable timeline.
//
// Apply validates and dedupes raw events by sequence. Ascending sequences are
// folded eagerly; an out-of-order fact (the attach overlap where live notices
// are replayed before history pages) invalidates the fold so Snapshot rebuilds
// it from sequence order. History and live input therefore converge on the same
// durable snapshot regardless of arrival order, while Apply stays the only
// mutation entry.
type Reducer struct {
	branch      string
	clock       Clock
	events      map[int64]RawEvent
	state       *foldState
	maxSequence int64
}

// NewReducer builds an empty reducer for one branch. The clock is injected so
// durable timestamps can only come from raw event facts.
func NewReducer(branch string, clock Clock) *Reducer {
	if clock == nil {
		clock = FixedClock(0)
	}
	return &Reducer{
		branch: branch,
		clock:  clock,
		events: map[int64]RawEvent{},
		state:  newFoldState(branch),
	}
}

// IgnoredEvents returns raw events that carried no known timeline semantics.
// They are kept only for diagnostics and never become messages.
func (r *Reducer) IgnoredEvents() []RawEvent {
	return append([]RawEvent(nil), r.foldState().ignored...)
}

// OrderKey is the zero-padded sequence used to sort durable messages.
func OrderKey(sequence int64) string { return fmt.Sprintf("%020d", sequence) }

// ApplyHistory sorts one history page ascending and applies it through the
// same reducer used for live notices.
func (r *Reducer) ApplyHistory(events []RawEvent) error {
	sorted := append([]RawEvent(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Sequence < sorted[j].Sequence
	})
	for _, event := range sorted {
		if err := r.Apply(event); err != nil {
			return err
		}
	}
	return nil
}

// Apply records one raw event. Replaying a sequence is a no-op.
func (r *Reducer) Apply(e RawEvent) error {
	if e.ID == "" || e.Sequence <= 0 || e.Type == "" || e.Name == "" {
		return errors.New("projection: invalid raw event")
	}
	if _, ok := r.events[e.Sequence]; ok {
		return nil
	}
	r.events[e.Sequence] = e
	if r.state != nil && e.Sequence > r.maxSequence {
		r.state.apply(e)
	} else {
		// A lower sequence breaks the incremental order; rebuild from
		// sequence order on the next Snapshot.
		r.state = nil
	}
	if e.Sequence > r.maxSequence {
		r.maxSequence = e.Sequence
	}
	return nil
}

// foldState returns the current fold, rebuilding it from sequence order when
// an out-of-order event invalidated the incremental state.
func (r *Reducer) foldState() *foldState {
	if r.state != nil {
		return r.state
	}
	ordered := make([]RawEvent, 0, len(r.events))
	for _, event := range r.events {
		ordered = append(ordered, event)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Sequence < ordered[j].Sequence
	})
	state := newFoldState(r.branch)
	for _, event := range ordered {
		state.apply(event)
	}
	r.state = state
	return state
}

// Snapshot returns a deep copy sorted by first-created sequence and insertion
// order, so callers cannot mutate reducer state through the snapshot.
func (r *Reducer) Snapshot() Snapshot {
	return r.foldState().snapshot()
}

// foldState is the one fold shared by history and live input. Events reach it
// in ascending sequence order; order independence is guaranteed by the
// Reducer's sequence ordering, not by insertion-order heuristics here.
type foldState struct {
	branch       string
	messages     map[string]*Message
	order        map[string]int64
	positions    map[string]int
	nextPosition int
	assistant    *Message
	tools        map[string]*Message
	pending      map[string][]pendingToolResult
	resultSeqs   map[string][]int64
	consumed     map[string]int64
	ignored      []RawEvent
}

func newFoldState(branch string) *foldState {
	return &foldState{
		branch:     branch,
		messages:   map[string]*Message{},
		order:      map[string]int64{},
		positions:  map[string]int{},
		tools:      map[string]*Message{},
		pending:    map[string][]pendingToolResult{},
		resultSeqs: map[string][]int64{},
		consumed:   map[string]int64{},
	}
}

func (f *foldState) apply(e RawEvent) {
	if e.Data == nil {
		e.Data = map[string]any{}
	}
	switch e.Type + "/" + e.Name {
	case "input/message":
		f.applyInput(e)
	case "input/applied", "input/consumed":
		f.completeInput(e)
	case "input/deleted":
		f.deleteInput(e)
	case "step/model-start":
		f.ensureAssistant(e, e.Data)
	case "step/text":
		f.applyText(e)
	case "step/thinking":
		f.applyThinking(e)
	case "step/model-output-end":
		f.endAssistant(e)
	case "step/tool-use-start":
		f.applyToolStart(e)
	case "step/tool-use":
		f.applyToolUse(e)
	case "step/tool-call":
		f.applyToolCalls(e)
	case "step/tool-completed":
		f.applyToolCompleted(e)
	case "step/tool-result":
		f.applyToolResults(e)
	default:
		// Unknown event types are retained for diagnostics but never become
		// timeline messages.
		f.ignored = append(f.ignored, e)
	}
}

func (f *foldState) snapshot() Snapshot {
	ids := make([]string, 0, len(f.messages))
	for id := range f.messages {
		ids = append(ids, id)
	}
	sort.SliceStable(ids, func(i, j int) bool {
		left, right := ids[i], ids[j]
		if f.order[left] != f.order[right] {
			return f.order[left] < f.order[right]
		}
		return f.positions[left] < f.positions[right]
	})
	out := Snapshot{
		Messages:               make([]Message, 0, len(ids)),
		TimelineUserMessageIds: []string{},
	}
	for _, id := range ids {
		message := cloneMessage(f.messages[id])
		out.Messages = append(out.Messages, message)
		if message.Role == "user" {
			out.TimelineUserMessageIds = append(out.TimelineUserMessageIds, id)
		}
	}
	return out
}

func (f *foldState) applyInput(e RawEvent) {
	text := inputText(e.Data)
	// Environment/date state arrives as input/message raw blocks without a
	// user. It is model context, not a timeline message. An explicit null user
	// is treated the same as an absent user (upstream timeline-fold.ts:635).
	user, hasUser := e.Data["user"]
	if (!hasUser || user == nil) && text == "" {
		return
	}
	m := f.ensure(e.ID, "user", e.Sequence, e.CreateAt)
	if m.Role != "user" {
		return
	}
	m.Content = text
	m.Status = "sending"
	// A live consumed notice may arrive before history replays the input.
	if at, ok := f.consumed[e.ID]; ok {
		m.Status = "complete"
		m.UpdatedAt = maxInt64(m.UpdatedAt, at)
	}
}

func (f *foldState) completeInput(e RawEvent) {
	// Only the documented wire keys are accepted; tolerant aliases are
	// forbidden by the projection contract.
	id := firstString(e.Data, "input_id", "inputId")
	if id == "" {
		return
	}
	f.consumed[id] = maxInt64(f.consumed[id], e.CreateAt)
	if m := f.messages[id]; m != nil {
		m.Status = "complete"
		m.UpdatedAt = maxInt64(m.UpdatedAt, e.CreateAt)
	}
}

// deleteInput removes a user input that the server deleted. This mirrors
// upstream timeline-fold.ts:679-690, which drops the message from messages,
// the id index and the user-message list.
func (f *foldState) deleteInput(e RawEvent) {
	id := firstString(e.Data, "input_id", "inputId")
	if id == "" {
		return
	}
	delete(f.consumed, id)
	if m := f.messages[id]; m != nil && m.Role != "user" {
		return
	}
	delete(f.messages, id)
	delete(f.order, id)
	delete(f.positions, id)
}

func (f *foldState) ensureAssistant(e RawEvent, data map[string]any) *Message {
	turn, request := assistantIdentity(data)
	if turn == "" || request == "" {
		return nil
	}
	id := assistantMessageID(f.branch, turn, request)
	_, existed := f.messages[id]
	m := f.ensure(id, "assistant", e.Sequence, e.CreateAt)
	// Switching the active assistant does not mutate the previous message's
	// blocks. Upstream keys messages by request and closes blocks only from
	// events that target that message (timeline-fold.ts:718-736, 738-808), so
	// inter-assistant arrival order cannot change block boundaries.
	f.assistant = m
	if !existed {
		m.Status = "streaming"
	}
	m.UpdatedAt = maxInt64(m.UpdatedAt, e.CreateAt)
	return m
}

// resolveAssistant resolves the target assistant from the event's
// turn_id+request_id. Only an event with no complete identity falls back to
// the currently active assistant. A delta with no target is ignored.
func (f *foldState) resolveAssistant(e RawEvent) *Message {
	if turn, request := assistantIdentity(e.Data); turn != "" && request != "" {
		return f.ensureAssistant(e, e.Data)
	}
	return f.assistant
}

func assistantIdentity(data map[string]any) (string, string) {
	turn := firstString(data, "turn_id", "turnId")
	request := firstString(data, "request_id", "requestId")
	if turn == "" || request == "" {
		return "", ""
	}
	return turn, request
}

func assistantMessageID(branch, turn, request string) string {
	return "a:" + branch + ":" + turn + ":" + request
}

func (f *foldState) applyText(e RawEvent) {
	m := f.resolveAssistant(e)
	if m == nil {
		return
	}
	// Text closes a running thinking block, then reuses or opens a text block
	// on the target assistant.
	f.closeBlocks(m, e.CreateAt, "thinking")
	block := f.openBlock(m, e, "text_delta")
	delta := stringValue(e.Data["delta"])
	block.Content += delta
	m.Content += delta
	m.UpdatedAt = maxInt64(m.UpdatedAt, e.CreateAt)
}

func (f *foldState) applyThinking(e RawEvent) {
	m := f.resolveAssistant(e)
	if m == nil {
		return
	}
	// Thinking closes a running text block, then reuses or opens a thinking
	// block. Consecutive thinking deltas share one block.
	f.closeBlocks(m, e.CreateAt, "text_delta")
	block := f.openBlock(m, e, "thinking")
	block.Content += stringValue(e.Data["delta"])
	m.UpdatedAt = maxInt64(m.UpdatedAt, e.CreateAt)
}

func (f *foldState) endAssistant(e RawEvent) {
	m := f.resolveAssistant(e)
	if m == nil {
		return
	}
	status := eventStatus(e.Data["status"])
	m.Status = status
	// model-output-end closes every running block with the same terminal
	// status: complete, error or cancelled.
	for i := range m.Events {
		if m.Events[i].Status != "running" {
			continue
		}
		m.Events[i].Status = status
		end := e.CreateAt
		m.Events[i].EndTime = &end
	}
	m.UpdatedAt = maxInt64(m.UpdatedAt, e.CreateAt)
}

// openBlock returns the target message's running block of typ, creating one
// with the first event's ID when none is open.
func (f *foldState) openBlock(m *Message, e RawEvent, typ string) *Event {
	for i := len(m.Events) - 1; i >= 0; i-- {
		block := &m.Events[i]
		if block.Type == typ && block.Status == "running" {
			return block
		}
	}
	m.Events = append(m.Events, Event{
		ID:        e.ID,
		Type:      typ,
		Status:    "running",
		StartTime: e.CreateAt,
	})
	return &m.Events[len(m.Events)-1]
}

// closeBlocks closes running blocks of the given types on one message.
func (f *foldState) closeBlocks(m *Message, at int64, types ...string) {
	for i := range m.Events {
		block := &m.Events[i]
		if block.Status != "running" {
			continue
		}
		if len(types) > 0 && !containsString(types, block.Type) {
			continue
		}
		block.Status = "complete"
		end := at
		block.EndTime = &end
	}
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// closeAssistantBlocks closes text/thinking blocks on the assistant identified
// by the event. When the event carries no complete identity it falls back to
// the currently active assistant, preserving the previous local behavior.
func (f *foldState) closeAssistantBlocks(data map[string]any, at int64) {
	if turn, request := assistantIdentity(data); turn != "" && request != "" {
		if m := f.messages[assistantMessageID(f.branch, turn, request)]; m != nil {
			f.closeBlocks(m, at, "text_delta", "thinking")
		}
		return
	}
	if f.assistant != nil {
		f.closeBlocks(f.assistant, at, "text_delta", "thinking")
	}
}

func (f *foldState) applyToolStart(e RawEvent) {
	f.closeAssistantBlocks(e.Data, e.CreateAt)
	callID := firstString(e.Data, "call_id", "callId")
	if callID == "" {
		return
	}
	name := stringValue(e.Data["name"])
	if existing := f.tools[callID]; existing != nil {
		if name != "" {
			f.mergeToolCall(existing, map[string]any{"name": name})
		}
		return
	}
	if name == "" {
		name = "tool"
	}
	f.ensureTool(e, map[string]any{"id": callID, "name": name})
}

func (f *foldState) applyToolUse(e RawEvent) {
	f.closeAssistantBlocks(e.Data, e.CreateAt)
	call := mapValue(e.Data["call"])
	if call == nil {
		// Tolerate a flattened tool-use payload.
		call = map[string]any{}
		if id := firstString(e.Data, "call_id", "callId"); id != "" {
			call["id"] = id
		}
		if name := stringValue(e.Data["name"]); name != "" {
			call["name"] = name
		}
		if input, ok := e.Data["input"]; ok {
			call["input"] = input
		}
	}
	f.ensureTool(e, call)
}

func (f *foldState) applyToolCalls(e RawEvent) {
	f.closeAssistantBlocks(e.Data, e.CreateAt)
	calls, _ := e.Data["calls"].([]any)
	for _, value := range calls {
		f.ensureTool(e, mapValue(value))
	}
}

func (f *foldState) applyToolCompleted(e RawEvent) {
	callID := firstString(e.Data, "call_id", "callId")
	if callID == "" {
		return
	}
	f.completeTool(callID, e.Sequence, e.CreateAt, e.Data["result"])
}

func (f *foldState) applyToolResults(e RawEvent) {
	results, _ := e.Data["results"].([]any)
	for _, value := range results {
		result := mapValue(value)
		callID := firstString(result, "call_id", "callId")
		if callID == "" {
			continue
		}
		f.completeTool(callID, e.Sequence, e.CreateAt, result["result"])
	}
}

// ensureTool creates a tool message and call event, merging later
// authoritative facts into the same call. A result that arrived before its
// call is flushed here instead of creating a second tool.
func (f *foldState) ensureTool(e RawEvent, call map[string]any) *Message {
	callID := firstString(call, "id", "call_id", "callId")
	if callID == "" {
		return nil
	}
	if existing := f.tools[callID]; existing != nil {
		f.mergeToolCall(existing, call)
		return existing
	}
	name := stringValue(call["name"])
	if name == "" {
		name = "tool"
	}
	m := f.ensure(callID, "tool", e.Sequence, e.CreateAt)
	m.Status = "streaming"
	m.Events = append(m.Events, Event{
		ID:         "tool:" + callID + ":call",
		Type:       "tool_call",
		Status:     "running",
		ToolName:   name,
		ToolParams: cloneMap(mapValue(call["input"])),
		StartTime:  e.CreateAt,
	})
	f.tools[callID] = m
	for _, pending := range f.pending[callID] {
		f.appendToolResult(m, callID, pending.sequence, pending.at, pending.result)
	}
	delete(f.pending, callID)
	return m
}

func (f *foldState) mergeToolCall(m *Message, call map[string]any) {
	name := stringValue(call["name"])
	input := mapValue(call["input"])
	if name == "" && input == nil {
		return
	}
	for i := range m.Events {
		if m.Events[i].Type != "tool_call" {
			continue
		}
		if name != "" {
			m.Events[i].ToolName = name
		}
		if input != nil {
			m.Events[i].ToolParams = cloneMap(input)
		}
		return
	}
}

func (f *foldState) completeTool(callID string, sequence, at int64, result any) {
	m := f.tools[callID]
	if m == nil {
		f.pending[callID] = append(f.pending[callID], pendingToolResult{sequence: sequence, at: at, result: result})
		return
	}
	f.appendToolResult(m, callID, sequence, at, result)
}

// appendToolResult keeps result events in sequence order even when live
// notices arrive out of order, so history and live folds converge on the same
// durable event list. The result event ID is synthetic and fixed per call
// (`tool:<callId>:result`), matching the upstream TimelineFold; the result raw
// event's provenance id is deliberately not reused. Message content/status and
// the call end time always derive from the highest-sequence result, regardless
// of arrival order.
func (f *foldState) appendToolResult(m *Message, callID string, sequence, at int64, result any) {
	status := resultStatus(result)
	content := toolText(result)
	end := at
	event := Event{
		ID:        "tool:" + callID + ":result",
		Type:      "tool_result",
		Status:    status,
		Content:   content,
		StartTime: at,
		EndTime:   &end,
	}
	// Tool messages hold the call event followed by result events. Insert the
	// new result before any result with a larger sequence.
	seqs := f.resultSeqs[callID]
	index := len(seqs)
	for i, existing := range seqs {
		if sequence < existing {
			index = i
			break
		}
	}
	insertAt := 1 + index
	if insertAt > len(m.Events) {
		insertAt = len(m.Events)
	}
	m.Events = append(m.Events, Event{})
	copy(m.Events[insertAt+1:], m.Events[insertAt:])
	m.Events[insertAt] = event
	seqs = append(seqs, 0)
	copy(seqs[index+1:], seqs[index:])
	seqs[index] = sequence
	f.resultSeqs[callID] = seqs
	m.UpdatedAt = maxInt64(m.UpdatedAt, at)
	f.recomputeToolOutcome(m, callID)
}

// recomputeToolOutcome derives the tool message status/content and call end
// time from the highest-sequence result. Result events are kept after the call
// event in ascending sequence order, so the last event is the authoritative
// one.
func (f *foldState) recomputeToolOutcome(m *Message, callID string) {
	if len(f.resultSeqs[callID]) == 0 || len(m.Events) < 2 {
		return
	}
	last := m.Events[len(m.Events)-1]
	m.Status = last.Status
	m.Content = last.Content
	for i := range m.Events {
		if m.Events[i].Type != "tool_call" {
			continue
		}
		m.Events[i].Status = last.Status
		end := last.StartTime
		m.Events[i].EndTime = &end
		break
	}
}

func (f *foldState) ensure(id, role string, seq, at int64) *Message {
	if m := f.messages[id]; m != nil {
		return m
	}
	m := &Message{
		ID:        id,
		OrderKey:  OrderKey(seq),
		Role:      role,
		Status:    "streaming",
		CreatedAt: at,
		UpdatedAt: at,
	}
	f.messages[id] = m
	f.order[id] = seq
	f.positions[id] = f.nextPosition
	f.nextPosition++
	return m
}

func cloneMessage(m *Message) Message {
	out := *m
	out.Events = make([]Event, len(m.Events))
	for i, event := range m.Events {
		out.Events[i] = event
		if event.ToolParams != nil {
			out.Events[i].ToolParams = cloneMap(event.ToolParams)
		}
		if event.EndTime != nil {
			end := *event.EndTime
			out.Events[i].EndTime = &end
		}
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneValue(value)
	}
	return out
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = cloneValue(item)
		}
		return out
	default:
		return value
	}
}

func mapValue(value any) map[string]any {
	if value == nil {
		return nil
	}
	m, _ := value.(map[string]any)
	return m
}

// shellInputPattern matches the runtime's recorded shell-input marker. A `!`
// run is stored as one `input/message` raw block whose text is the marker; the
// recorded body is what stays visible after reload (timeline-fold.ts:289-302).
var shellInputPattern = regexp.MustCompile(`(?s)^<dim-shell-input\b[^>]*>(.*)</dim-shell-input>\s*$`)

// inputText mirrors upstream inputTextFromData (timeline-fold.ts:303-335): the
// user's own text blocks win, then ordinary text blocks, then recorded
// shell-input raw blocks joined with a newline. Any other raw block stays model
// context and never becomes timeline text. There is deliberately no fallback to
// an undocumented top-level `content` field: InputMessageData denies unknown
// fields, so such an event could only be fabricated locally.
func inputText(data map[string]any) string {
	if user := mapValue(data["user"]); user != nil {
		if text := textBlockText(user["content"]); text != "" {
			return text
		}
	}
	blocks, _ := data["blocks"].([]any)
	if text := textBlockText(blocks); text != "" {
		return text
	}
	bodies := make([]string, 0, len(blocks))
	for _, value := range blocks {
		block := mapValue(value)
		if stringValue(block["type"]) != "raw" {
			continue
		}
		if body := shellInputBody(stringValue(block["text"])); body != "" {
			bodies = append(bodies, body)
		}
	}
	return strings.Join(bodies, "\n")
}

// textBlockText joins the text of `type: "text"` content blocks. Other block
// types (images) contribute no timeline text.
func textBlockText(parts any) string {
	blocks, _ := parts.([]any)
	var text strings.Builder
	for _, part := range blocks {
		block := mapValue(part)
		if stringValue(block["type"]) == "text" {
			text.WriteString(stringValue(block["text"]))
		}
	}
	return text.String()
}

// shellInputBody decodes the escaped marker body. Entities decode one level
// only, in the upstream order, and the framing newlines are trimmed.
func shellInputBody(text string) string {
	match := shellInputPattern.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	body := match[1]
	body = strings.ReplaceAll(body, "&lt;", "<")
	body = strings.ReplaceAll(body, "&gt;", ">")
	body = strings.ReplaceAll(body, "&quot;", "\"")
	body = strings.ReplaceAll(body, "&#39;", "'")
	body = strings.ReplaceAll(body, "&amp;", "&")
	return strings.TrimSpace(body)
}

func eventStatus(status any) string {
	switch status {
	case "failed":
		return "error"
	case "cancelled":
		return "cancelled"
	default:
		return "complete"
	}
}

func resultStatus(result any) string {
	value := mapValue(result)
	if value == nil {
		return "complete"
	}
	if isError, ok := value["isError"].(bool); ok && isError {
		return "error"
	}
	return "complete"
}

func toolText(result any) string {
	value := mapValue(result)
	if value == nil {
		return ""
	}
	switch content := value["content"].(type) {
	case string:
		return content
	case []any:
		var text strings.Builder
		for _, part := range content {
			block := mapValue(part)
			if stringValue(block["type"]) == "text" {
				text.WriteString(stringValue(block["text"]))
			}
		}
		return text.String()
	}
	return ""
}

func stringValue(value any) string {
	s, _ := value.(string)
	return s
}

func firstString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(data[key]); value != "" {
			return value
		}
	}
	return ""
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// ParseSequence accepts sequence values from decoded JSON.
func ParseSequence(v any) (int64, error) {
	switch x := v.(type) {
	case int64:
		return x, nil
	case float64:
		return int64(x), nil
	case string:
		return strconv.ParseInt(x, 10, 64)
	}
	return 0, errors.New("invalid sequence")
}
