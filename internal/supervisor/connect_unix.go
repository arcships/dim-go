//go:build unix

package supervisor

import (
	"context"
	"fmt"
	"net"
)

func connect(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
	if endpoint.Network != NetworkUnix {
		return nil, fmt.Errorf("supervisor: endpoint network %q is not supported on this platform", endpoint.Network)
	}
	var dialer net.Dialer
	return dialer.DialContext(ctx, "unix", endpoint.Address)
}
