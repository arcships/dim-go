package appserver

import "context"

// HistoryPage reads one session.history page and enforces the §9 paging
// invariants against the request that produced it. Window and latestTurn
// selection are intentionally left to a higher layer.
func HistoryPage(ctx context.Context, client *Client, params HistoryParams) (HistoryResult, error) {
	if err := params.Validate(); err != nil {
		return HistoryResult{}, err
	}
	raw, err := client.Call(ctx, "session.history", params)
	if err != nil {
		return HistoryResult{}, err
	}
	return DecodeHistoryResult(raw, params)
}
