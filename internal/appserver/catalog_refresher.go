package appserver

import (
	"context"
	"encoding/json"
	"sync"
)

// CatalogRefresher coalesces concurrent session.catalog refreshes with the same
// params into one traversal (contract §7.1 single-flight). It is safe for
// concurrent use. The stateless Catalog helper keeps its own semantics.
type CatalogRefresher struct {
	client   *Client
	features []string

	mu      sync.Mutex
	flights map[string]*catalogFlight

	// joined, when non-nil, runs after a caller joins an in-flight traversal.
	// Tests use it to observe coalescing deterministically.
	joined func()
}

type catalogFlight struct {
	done   chan struct{}
	result CatalogResult
	err    error
}

// NewCatalogRefresher fixes the client and the connection-level hello.features
// snapshot used by every traversal.
func NewCatalogRefresher(client *Client, features []string) *CatalogRefresher {
	return &CatalogRefresher{
		client:   client,
		features: append([]string(nil), features...),
		flights:  make(map[string]*catalogFlight),
	}
}

// Refresh returns the merged catalog for params. Non-forced calls with the same
// params join an in-flight traversal and receive the same result or error.
// A forced call starts a new traversal even when one is already in flight;
// existing waiters keep waiting on their original traversal.
//
// The shared traversal is not tied to one caller's context, so a caller timing
// out does not cancel the refresh for everyone else; that caller's wait instead
// returns request_timeout/request_aborted.
func (r *CatalogRefresher) Refresh(ctx context.Context, params CatalogParams, force bool) (CatalogResult, error) {
	if err := ctx.Err(); err != nil {
		return CatalogResult{}, localContextError(ctx)
	}
	key, err := catalogKey(params)
	if err != nil {
		return CatalogResult{}, err
	}
	r.mu.Lock()
	if !force {
		if flight, ok := r.flights[key]; ok {
			joined := r.joined
			r.mu.Unlock()
			if joined != nil {
				joined()
			}
			return waitCatalog(ctx, flight)
		}
	}
	flight := &catalogFlight{done: make(chan struct{})}
	r.flights[key] = flight
	r.mu.Unlock()
	go r.run(flight, params, key)
	return waitCatalog(ctx, flight)
}

func (r *CatalogRefresher) run(flight *catalogFlight, params CatalogParams, key string) {
	defer close(flight.done)
	flight.result, flight.err = Catalog(context.Background(), r.client, params, r.features)
	r.mu.Lock()
	// A forced refresh may have replaced this flight; only remove our own.
	if r.flights[key] == flight {
		delete(r.flights, key)
	}
	r.mu.Unlock()
}

func waitCatalog(ctx context.Context, flight *catalogFlight) (CatalogResult, error) {
	select {
	case <-flight.done:
		return flight.result, flight.err
	case <-ctx.Done():
		return CatalogResult{}, localContextError(ctx)
	}
}

func catalogKey(params CatalogParams) (string, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
