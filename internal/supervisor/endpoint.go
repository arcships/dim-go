package supervisor

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Endpoint network names. The unix network is a filesystem socket; the pipe
// network is a Windows named pipe.
const (
	NetworkUnix = "unix"
	NetworkPipe = "pipe"
)

const (
	daemonSocketName = "daemon.sock"
	nativeHomeName   = "rust"
)

// ErrInsecureNativeHome marks a native home that is not private to the current
// user. The daemon link is fail-closed on this: the directory must be 0700 and
// owned by the current user (protocol-contract §18).
var ErrInsecureNativeHome = errors.New("supervisor: app server native home must be private (0700) and owned by the current user")

// Endpoint is one daemon transport endpoint: a Unix socket path or a Windows
// named-pipe path.
type Endpoint struct {
	Network string
	Address string
}

func (e Endpoint) Validate() error {
	switch e.Network {
	case NetworkUnix, NetworkPipe:
	default:
		return fmt.Errorf("supervisor: unknown endpoint network %q", e.Network)
	}
	if strings.TrimSpace(e.Address) == "" {
		return errors.New("supervisor: endpoint address must not be empty")
	}
	return nil
}

func (e Endpoint) String() string { return e.Network + ":" + e.Address }

// Link is the resolved daemon location for one product home. NativeHome is the
// canonical `<productHome>/rust` directory; ProductHome is the caller-supplied
// value and is what the daemon receives as `--product-home`.
type Link struct {
	ProductHome string
	NativeHome  string
	Endpoint    Endpoint
}

// ResolveLink resolves and validates the daemon endpoint for productHome.
//
// The native home `<productHome>/rust` is created with mode 0700 when missing.
// On Unix an existing directory must be exactly 0700 and owned by the current
// user. The endpoint is built from the canonicalized native home, so symlinked
// product homes address the same socket.
func ResolveLink(productHome string) (Link, error) {
	if strings.TrimSpace(productHome) == "" {
		return Link{}, errors.New("supervisor: productHome must not be empty")
	}
	abs, err := filepath.Abs(productHome)
	if err != nil {
		return Link{}, fmt.Errorf("supervisor: resolve productHome %q: %w", productHome, err)
	}
	nativeHome := filepath.Join(abs, nativeHomeName)
	if err := ensureNativeHome(nativeHome); err != nil {
		return Link{}, err
	}
	canonical, err := filepath.EvalSymlinks(nativeHome)
	if err != nil {
		return Link{}, fmt.Errorf("supervisor: canonicalize native home %s: %w", nativeHome, err)
	}
	canonical = filepath.Clean(canonical)
	endpoint, err := endpointFor(canonical)
	if err != nil {
		return Link{}, err
	}
	return Link{ProductHome: productHome, NativeHome: canonical, Endpoint: endpoint}, nil
}
