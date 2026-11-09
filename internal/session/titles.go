package session

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// TitleState is the session title metadata the client knows: text (nil means a
// cleared title), source, and the committed revision. Known is false until the
// daemon supplies the triple through a rename reply or a SessionStore title
// notice; a catalog summary seeds it on the first list read.
type TitleState struct {
	Text     *string
	Source   *string
	Revision int64
	Known    bool
}

// Title returns the newest title metadata this session observed. The value is a
// copy; callers may keep it after the session updates.
func (s *Session) Title() TitleState {
	if s == nil {
		return TitleState{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneTitleState(s.title)
}

// Rename sets or clears the manual title (title nil clears it). session.rename
// is a session-level method advertised by session.methods, not a hello.features
// capability, so this helper invents no local gate: a daemon that lacks the
// method answers unknown_method, which the appserver layer maps to a capability
// error (visible-disabled WIP, never retried). The captured method table is for
// display and is deliberately not used to second-guess the daemon, because a
// withdrawn namespace can register again after a reload.
//
// Rename never retries. A timeout or a lost transport is an unknown result the
// caller resolves by reading session.summaries; a new title revision is kept
// only when it is newer than what this session already observed.
func (s *Session) Rename(ctx context.Context, title *string) (appserver.RenameResult, error) {
	if s == nil {
		return appserver.RenameResult{}, fmt.Errorf("session: nil session")
	}
	result, err := appserver.Rename(ctx, s.client, appserver.RenameParams{SessionID: s.sessionID, Title: title})
	if err != nil {
		return appserver.RenameResult{}, err
	}
	s.mu.Lock()
	s.applyTitleLocked(TitleState{
		Text:     result.Title,
		Source:   result.TitleSource,
		Revision: result.Revision,
		Known:    true,
	})
	s.mu.Unlock()
	return result, nil
}

// applyTitleNotice keeps the title metadata of a SessionStore title notice.
// Notices are not replayed, so a malformed one is reported instead of being
// silently dropped; it never clears or invents title state.
func (s *Session) applyTitleNotice(raw json.RawMessage) error {
	notice, err := appserver.DecodeTitleNotice(raw)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyTitleLocked(TitleState{
		Text:     notice.Title,
		Source:   notice.TitleSource,
		Revision: notice.TitleRevision,
		Known:    true,
	})
	return nil
}

// applyTitleLocked stores next only when it is newer than the observed state.
// Title revisions are per-Session and monotonic; a lower or equal revision is a
// stale list read or an out-of-order notice and must not overwrite the title.
func (s *Session) applyTitleLocked(next TitleState) {
	if !next.Known {
		return
	}
	if s.title.Known && next.Revision <= s.title.Revision {
		return
	}
	s.title = cloneTitleState(next)
}

func cloneTitleState(in TitleState) TitleState {
	return TitleState{
		Text:     cloneString(in.Text),
		Source:   cloneString(in.Source),
		Revision: in.Revision,
		Known:    in.Known,
	}
}
