package projection

import "sort"

// progressNotice 是 `session.model.progress` 通知的参数里视图需要的部分。
// 它没有序号：这是"还没变成事实"的增量，只作预览。
type progressNotice struct {
	SessionID      string `json:"sessionId"`
	BranchID       string `json:"branchId"`
	TurnID         string `json:"turnId,omitempty"`
	RequestID      string `json:"requestId,omitempty"`
	Text           string `json:"text"`
	TextOffset     int    `json:"textOffset"`
	Thinking       string `json:"thinking"`
	ThinkingOffset int    `json:"thinkingOffset"`
}

// previewEntry 是一条助手消息的末快照：只保留已落库内容之后的尾部。
type previewEntry struct {
	turnID    string
	requestID string
	text      string
	thinking  string
}

// applyProgress 整体替换一条末快照。偏移之前的内容已经落库，只留尾部，
// 因此同一条预览反复到达也不会叠加。
func (b *branchState) applyProgress(notice progressNotice) {
	if notice.TurnID == "" || notice.RequestID == "" {
		// 没有轮次与请求标识的预览无法与落库行对齐，按契约不进视图。
		return
	}
	key := notice.TurnID + "\x00" + notice.RequestID
	text := suffixFrom(notice.Text, notice.TextOffset)
	thinking := suffixFrom(notice.Thinking, notice.ThinkingOffset)
	if text == "" && thinking == "" {
		delete(b.preview, key)
		return
	}
	b.preview[key] = &previewEntry{
		turnID:    notice.TurnID,
		requestID: notice.RequestID,
		text:      text,
		thinking:  thinking,
	}
}

// suffixFrom 取偏移之后的尾部。偏移按字符计，越界按空处理。
func suffixFrom(text string, offset int) string {
	runes := []rune(text)
	if offset <= 0 {
		return string(runes)
	}
	if offset >= len(runes) {
		return ""
	}
	return string(runes[offset:])
}

// previewRows 把末快照摊成尾部行。一旦这条助手消息有了落库的文本或思考行，
// 预览整行消失：落库内容替换预览，不做字符级拼接，也不会重复。
func (s *foldState) previewRows(branch *branchState) []Row {
	if len(branch.preview) == 0 {
		return nil
	}
	keys := make([]string, 0, len(branch.preview))
	for key := range branch.preview {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var rows []Row
	for _, key := range keys {
		if _, landed := s.seenMessage[key]; landed {
			continue
		}
		entry := branch.preview[key]
		turnIndex := 0
		if turn, ok := s.turnByID[entry.turnID]; ok {
			turnIndex = turn.index
		}
		base := Row{
			Kind:      RowAssistantText,
			Pending:   true,
			BranchID:  branch.id,
			TurnIndex: turnIndex,
			TurnID:    entry.turnID,
			Status:    StatusRunning,
		}
		if entry.text != "" {
			row := base
			row.Text = entry.text
			rows = append(rows, row)
		}
		if entry.thinking != "" {
			row := base
			row.Kind = RowThinking
			row.Text = entry.thinking
			rows = append(rows, row)
		}
	}
	return rows
}
