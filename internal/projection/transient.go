package projection

import "errors"

// ModelProgressTool is one in-flight tool call in a model progress snapshot.
type ModelProgressTool struct {
	CallID    string `json:"callId"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ModelProgress is the transient `session.model.progress` replacement
// snapshot. It is view-only and never enters the durable timeline.
type ModelProgress struct {
	SessionID      string              `json:"sessionId"`
	BranchID       string              `json:"branchId"`
	TurnID         string              `json:"turnId,omitempty"`
	RequestID      string              `json:"requestId,omitempty"`
	TextPartID     string              `json:"textPartId,omitempty"`
	ThinkingPartID string              `json:"thinkingPartId,omitempty"`
	Text           string              `json:"text"`
	TextOffset     int                 `json:"textOffset"`
	Thinking       string              `json:"thinking"`
	ThinkingOffset int                 `json:"thinkingOffset"`
	Tools          []ModelProgressTool `json:"tools"`
}

// TransientNotice records one non-durable notification (approval, question,
// model progress, permission request, ...). ReceivedAt is local observation
// time from the injected clock.
type TransientNotice struct {
	Kind       string         `json:"kind"`
	ID         string         `json:"id"`
	Sequence   int64          `json:"sequence,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
	ReceivedAt int64          `json:"receivedAt"`
}

// TransientSnapshot is the independent output of transient projections.
type TransientSnapshot struct {
	ModelProgress *ModelProgress    `json:"modelProgress,omitempty"`
	Notices       []TransientNotice `json:"notices"`
}

// TransientReducer folds ephemeral notices for one session branch. Its output
// never changes durable message IDs, order keys or block boundaries.
type TransientReducer struct {
	sessionID string
	branchID  string
	clock     Clock
	seenSeq   bool
	sequence  int64
	progress  *ModelProgress
	notices   []TransientNotice
	noticeAt  map[string]int
}

func NewTransientReducer(sessionID, branchID string, clock Clock) *TransientReducer {
	if clock == nil {
		clock = FixedClock(0)
	}
	return &TransientReducer{
		sessionID: sessionID,
		branchID:  branchID,
		clock:     clock,
		notices:   []TransientNotice{},
		noticeAt:  map[string]int{},
	}
}

// ApplyNotice records or replaces one transient notice. Replaying the same id
// replaces it in place, so transient replays cannot grow the list. A notice
// whose sequence is older than the recorded one is stale and ignored.
func (t *TransientReducer) ApplyNotice(kind, id string, sequence int64, payload map[string]any) error {
	if kind == "" || id == "" {
		return errors.New("projection: transient notice requires kind and id")
	}
	if index, ok := t.noticeAt[id]; ok {
		existing := &t.notices[index]
		if sequence > 0 && existing.Sequence > 0 && sequence < existing.Sequence {
			return nil
		}
		existing.Kind = kind
		existing.Sequence = sequence
		existing.Payload = cloneMap(payload)
		existing.ReceivedAt = t.clock.NowMillis()
		return nil
	}
	t.noticeAt[id] = len(t.notices)
	t.notices = append(t.notices, TransientNotice{
		Kind:       kind,
		ID:         id,
		Sequence:   sequence,
		Payload:    cloneMap(payload),
		ReceivedAt: t.clock.NowMillis(),
	})
	return nil
}

// ApplyModelProgress replaces the current preview. A lower sequence is stale
// and ignored; an empty preview clears it. A non-positive sequence means the
// notice carries no ordering fact and is applied as-is.
func (t *TransientReducer) ApplyModelProgress(sequence int64, progress ModelProgress) error {
	if progress.SessionID != "" && progress.SessionID != t.sessionID {
		return errors.New("projection: model progress session mismatch")
	}
	if progress.BranchID != "" && progress.BranchID != t.branchID {
		return errors.New("projection: model progress branch mismatch")
	}
	if sequence > 0 {
		if t.seenSeq && sequence < t.sequence {
			return nil
		}
		t.seenSeq = true
		t.sequence = sequence
	}
	if progress.Text == "" && progress.Thinking == "" && len(progress.Tools) == 0 {
		t.progress = nil
		return nil
	}
	next := progress
	next.SessionID = t.sessionID
	next.BranchID = t.branchID
	next.Tools = append([]ModelProgressTool{}, progress.Tools...)
	t.progress = &next
	return nil
}

// Snapshot returns an independent copy of the transient state.
func (t *TransientReducer) Snapshot() TransientSnapshot {
	out := TransientSnapshot{Notices: make([]TransientNotice, len(t.notices))}
	for i, notice := range t.notices {
		out.Notices[i] = notice
		out.Notices[i].Payload = cloneMap(notice.Payload)
	}
	if t.progress != nil {
		progress := *t.progress
		progress.Tools = append([]ModelProgressTool{}, t.progress.Tools...)
		out.ModelProgress = &progress
	}
	return out
}
