package app

import "context"

// HistoryKinds 统计本会话历史里的事实类型与条数（`type/name`），不含任何内容。
//
// 只读、按页读取，上限与变更读取一致；用于自检确认“命令结果写回了历史”，
// 不把正文带进证据。
func (a *App) HistoryKinds(ctx context.Context, sessionID string) (map[string]int, error) {
	runtime, err := a.openRuntime(sessionID, "session.history")
	if err != nil {
		return nil, err
	}
	events, err := (&changeSource{app: a}).collectFacts(ctx, runtime)
	if err != nil {
		return nil, err
	}
	kinds := make(map[string]int, len(events))
	for _, event := range events {
		kinds[event.Type+"/"+event.Name]++
	}
	return kinds, nil
}
