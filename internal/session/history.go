package session

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/arcships/dimagent-mygo/internal/appserver"
	"github.com/arcships/dimagent-mygo/internal/projection"
)

const (
	// compatibilityPageLimit and compatibilityPageCap come from
	// docs/protocol-contract.md §9: when session.history.latestTurn is absent,
	// scan backward in 512-event chunks, at most 64 pages.
	compatibilityPageLimit = 512
	compatibilityPageCap   = 64
	// latestTurnPageLimit and latestTurnPageCap bound the reverse latestTurn
	// traversal. A cap hit degrades the window instead of failing Open.
	latestTurnPageLimit = 4096
	latestTurnPageCap   = 64
)

// historyWindow is the selected first-screen history plus its observability.
// fallback means the latestTurn probe must be discarded and re-read through the
// compatibility scan.
type historyWindow struct {
	events   []projection.RawEvent
	status   HistoryStatus
	fallback bool
}

// readLatestTurn follows session.history.latestTurn: it starts at the attach
// head, pages backward with nextBefore until done, and requires a stable
// turnStart on every page. A turnStart drift is a soft signal to fall back to
// the compatibility scan; a missing turnStart/nextBefore is a protocol error.
func (s *Session) readLatestTurn(ctx context.Context, head int64) (historyWindow, error) {
	limit := latestTurnPageLimit
	through := head
	latestTurn := true
	params := appserver.HistoryParams{
		SessionID:  s.sessionID,
		BranchID:   appserver.MainBranchID,
		Limit:      &limit,
		Through:    &through,
		LatestTurn: &latestTurn,
	}
	var events []projection.RawEvent
	var turnStart *int64
	var lastBefore *int64
	pages := 0
	done := false
	for pages < latestTurnPageCap {
		if err := s.pendingFailure(); err != nil {
			return historyWindow{}, err
		}
		page, err := appserver.HistoryPage(ctx, s.client, params)
		if err != nil {
			if isHistoryTooLarge(err) {
				return degradedHistoryWindow(HistoryModeLatestTurn, head, pages, "history_too_large"), nil
			}
			return historyWindow{}, err
		}
		pages++
		if page.TurnStart == nil {
			return historyWindow{}, &appserver.ProtocolError{Field: "turnStart", Detail: "missing in latest-turn page"}
		}
		if turnStart == nil {
			value := *page.TurnStart
			turnStart = &value
		} else if *page.TurnStart != *turnStart {
			return historyWindow{fallback: true}, nil
		}
		if !page.Done {
			if page.NextBefore == nil {
				return historyWindow{}, &appserver.ProtocolError{Field: "nextBefore", Detail: "missing while done is false"}
			}
			next := *page.NextBefore
			if next < 0 || next > head {
				return historyWindow{}, &appserver.ProtocolError{
					Field:  "nextBefore",
					Detail: fmt.Sprintf("got %d, want between 0 and head=%d", next, head),
				}
			}
			if lastBefore != nil && next >= *lastBefore {
				return historyWindow{}, &appserver.ProtocolError{
					Field:  "nextBefore",
					Detail: fmt.Sprintf("got %d after %d; backward paging must decrease", next, *lastBefore),
				}
			}
			value := next
			lastBefore = &value
		}
		adapted, err := adaptEvents(page.Events)
		if err != nil {
			return historyWindow{}, err
		}
		events = append(events, adapted...)
		if page.Done {
			done = true
			break
		}
		before := *page.NextBefore
		params = appserver.HistoryParams{
			SessionID:  s.sessionID,
			BranchID:   appserver.MainBranchID,
			Limit:      &limit,
			Through:    &through,
			Before:     &before,
			LatestTurn: &latestTurn,
		}
	}
	ordered, err := orderedUniqueEvents(events)
	if err != nil {
		return historyWindow{}, err
	}
	status := HistoryStatus{
		Mode:      HistoryModeLatestTurn,
		Through:   head,
		TurnStart: cloneInt64(turnStart),
		HasOlder:  turnStart != nil && *turnStart > 1,
		Pages:     pages,
	}
	if !done && pages >= latestTurnPageCap {
		status.Degraded = true
		status.Reason = "page_cap"
		status.HasOlder = true
	}
	return historyWindow{events: ordered, status: status}, nil
}

// readCompatibility implements the no-latestTurn path: chunk backward from the
// attach head with {after, limit, through}, at most 64 chunks, stopping when the
// latest turn and its prompting input are both visible. A history_too_large
// reply or a page-cap hit degrades the window; it never fails Open.
func (s *Session) pendingFailure() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pendingErr
}

func (s *Session) readCompatibility(ctx context.Context, head int64, fallbackReason string) (historyWindow, error) {
	limit := compatibilityPageLimit
	through := head
	var events []projection.RawEvent
	pages := 0
	reachedStart := false
	stoppedAtTurn := false
	for pages < compatibilityPageCap {
		if err := s.pendingFailure(); err != nil {
			return historyWindow{}, err
		}
		after := through - int64(limit)
		if after < 0 {
			after = 0
		}
		windowThrough := through
		params := appserver.HistoryParams{
			SessionID: s.sessionID,
			BranchID:  appserver.MainBranchID,
			After:     &after,
			Limit:     &limit,
			Through:   &windowThrough,
		}
		page, err := appserver.HistoryPage(ctx, s.client, params)
		if err != nil {
			if isHistoryTooLarge(err) {
				ordered, orderedErr := orderedUniqueEvents(events)
				if orderedErr != nil {
					return historyWindow{}, orderedErr
				}
				selected, turnStart, _ := selectLatestTurn(ordered)
				status := HistoryStatus{
					Mode:      HistoryModeCompatibility,
					Through:   head,
					TurnStart: cloneInt64(turnStart),
					// A too-large atomic batch is a degraded first screen: the
					// contract requires hasOlder=true even when a partially read
					// page happened to start at sequence 1.
					HasOlder: true,
					Degraded: true,
					Reason:   "history_too_large",
					Pages:    pages,
					Fallback: fallbackReason != "",
				}
				return historyWindow{events: selected, status: status}, nil
			}
			return historyWindow{}, err
		}
		pages++
		adapted, err := adaptEvents(page.Events)
		if err != nil {
			return historyWindow{}, err
		}
		events = append(events, adapted...)
		if len(page.Events) == 0 {
			reachedStart = true
			break
		}
		earliest := earliestSequence(page.Events)
		if earliest <= 1 {
			reachedStart = true
			break
		}
		through = earliest - 1
		if through <= 0 {
			reachedStart = true
			break
		}
		if hasCompleteTurnWindow(events) {
			stoppedAtTurn = true
			break
		}
	}

	ordered, err := orderedUniqueEvents(events)
	if err != nil {
		return historyWindow{}, err
	}
	selected, turnStart, windowStart := selectLatestTurn(ordered)
	truncated := pages >= compatibilityPageCap && !reachedStart && !stoppedAtTurn
	status := HistoryStatus{
		Mode:      HistoryModeCompatibility,
		Through:   head,
		TurnStart: cloneInt64(turnStart),
		HasOlder:  truncated || windowStart > 1,
		Pages:     pages,
		Fallback:  fallbackReason != "",
	}
	if truncated {
		status.Degraded = true
		status.Reason = "page_cap"
		status.HasOlder = true
	}
	return historyWindow{events: selected, status: status}, nil
}

func degradedHistoryWindow(mode HistoryMode, head int64, pages int, reason string) historyWindow {
	return historyWindow{
		status: HistoryStatus{
			Mode:     mode,
			Through:  head,
			HasOlder: true,
			Degraded: true,
			Reason:   reason,
			Pages:    pages,
		},
	}
}

// isHistoryTooLarge recognizes the §9/§13 history_too_large reply. The window
// degrades instead of retrying the same atomic batch.
func isHistoryTooLarge(err error) bool {
	var rpcErr *appserver.RPCError
	if errors.As(err, &rpcErr) {
		return appserver.MapRPCError(rpcErr).Class == appserver.ErrorClassHistoryTooLarge
	}
	return appserver.IsAppError(err, appserver.AppErrorHistoryTooLarge)
}

func adaptEvents(events []appserver.RawEvent) ([]projection.RawEvent, error) {
	out := make([]projection.RawEvent, 0, len(events))
	for _, event := range events {
		adapted, err := AdaptEvent(event)
		if err != nil {
			return nil, err
		}
		out = append(out, adapted)
	}
	return out, nil
}

// orderedUniqueEvents sorts ascending by sequence and rejects one sequence
// mapped to two ids. The stream layer applies the same conflict rule to live
// events; history must not silently accept a different fact for a sequence.
func orderedUniqueEvents(events []projection.RawEvent) ([]projection.RawEvent, error) {
	bySequence := make(map[int64]projection.RawEvent, len(events))
	for _, event := range events {
		if existing, ok := bySequence[event.Sequence]; ok {
			if existing.ID != event.ID {
				return nil, &appserver.ProtocolError{
					Field:  "history.events",
					Detail: fmt.Sprintf("sequence %d arrived with id %q and %q", event.Sequence, existing.ID, event.ID),
				}
			}
			continue
		}
		bySequence[event.Sequence] = event
	}
	ordered := make([]projection.RawEvent, 0, len(bySequence))
	for _, event := range bySequence {
		ordered = append(ordered, event)
	}
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Sequence < ordered[j].Sequence })
	return ordered, nil
}

func earliestSequence(events []appserver.RawEvent) int64 {
	var earliest int64
	for _, event := range events {
		if earliest == 0 || event.Sequence < earliest {
			earliest = event.Sequence
		}
	}
	return earliest
}

// selectLatestTurn keeps the newest turn and its prompting input. When no turn
// start is visible it keeps every event read; when no prompting input is found
// it keeps the turn start onward. windowStart > 1 means older facts exist
// outside the window.
func selectLatestTurn(events []projection.RawEvent) ([]projection.RawEvent, *int64, int64) {
	if len(events) == 0 {
		return []projection.RawEvent{}, nil, 0
	}
	turnStart := latestTurnStart(events)
	if turnStart == nil {
		return events, nil, events[0].Sequence
	}
	inputSequence, hasInput := latestInputBefore(events, *turnStart)
	selected := make([]projection.RawEvent, 0, len(events))
	windowStart := *turnStart
	for _, event := range events {
		if event.Sequence >= *turnStart || (hasInput && event.Sequence == inputSequence) {
			selected = append(selected, event)
			if event.Sequence < windowStart {
				windowStart = event.Sequence
			}
		}
	}
	return selected, turnStart, windowStart
}

func hasCompleteTurnWindow(events []projection.RawEvent) bool {
	turnStart := latestTurnStart(events)
	if turnStart == nil {
		return false
	}
	if *turnStart <= 1 {
		return true
	}
	_, ok := latestInputBefore(events, *turnStart)
	return ok
}

func latestTurnStart(events []projection.RawEvent) *int64 {
	var latest *int64
	for i := range events {
		event := events[i]
		if !isTurnStart(event) {
			continue
		}
		sequence := event.Sequence
		if latest == nil || sequence > *latest {
			latest = &sequence
		}
	}
	return latest
}

func latestInputBefore(events []projection.RawEvent, before int64) (int64, bool) {
	var latest int64
	found := false
	for _, event := range events {
		if event.Sequence >= before || !isInputMessage(event) {
			continue
		}
		if !found || event.Sequence > latest {
			latest = event.Sequence
			found = true
		}
	}
	return latest, found
}

// isTurnStart recognizes the flattened Rust turn/start EventKind. Only the
// documented type/name pair is accepted; no tolerant aliases are invented.
func isTurnStart(event projection.RawEvent) bool {
	return event.Type == "turn" && event.Name == "start"
}

// isInputMessage recognizes the user input/message raw event used by the
// projection contract.
func isInputMessage(event projection.RawEvent) bool {
	return event.Type == "input" && event.Name == "message"
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
