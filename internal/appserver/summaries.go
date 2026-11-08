package appserver

import (
	"context"
	"errors"
)

// Summaries reads session.summaries. The response is a bare array, not an
// object (§7.2). Older daemons reject includeClosed with -32602; that exact
// combination is retried once without the field, and every other error
// propagates unchanged.
func Summaries(ctx context.Context, client *Client, params SummariesParams) ([]SessionSummary, error) {
	raw, err := client.Call(ctx, "session.summaries", params)
	if err != nil {
		var rpcErr *RPCError
		if !params.IncludeClosed || !errors.As(err, &rpcErr) || rpcErr.Code != -32602 {
			return nil, err
		}
		raw, err = client.Call(ctx, "session.summaries", SummariesParams{})
		if err != nil {
			return nil, err
		}
	}
	return DecodeSummaries(raw)
}
