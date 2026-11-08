package supervisor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	// DefaultListenTimeout and DefaultListenInterval are the §18 wait-for-listen
	// bounds: 10 seconds with a 50 ms retry.
	DefaultListenTimeout  = 10 * time.Second
	DefaultListenInterval = 50 * time.Millisecond

	daemonLogName       = "daemon.log"
	defaultServerBinary = "dim-app-server"
)

// ErrListenTimeout marks a spawned daemon that never opened its endpoint
// within the wait budget.
var ErrListenTimeout = errors.New("supervisor: timed out waiting for the app server to listen")

// Process is a spawned daemon child. Supervisor never signals it directly: an
// owned daemon is stopped through daemon.shutdown after the connection check.
type Process interface {
	Pid() int
	Done() <-chan error
}

// SpawnConfig is one daemon launch request.
type SpawnConfig struct {
	// ServerPath is the dim-app-server binary. When empty the default binary is
	// looked up on PATH.
	ServerPath string
	Endpoint   Endpoint
	// NativeHome is the canonical `<productHome>/rust` directory. It is passed
	// as --home and is where daemon.log is written.
	NativeHome      string
	ProductHome     string
	MockModel       bool
	DefaultsPath    string
	ModelConfigPath string
}

func (c SpawnConfig) Validate() error {
	_, err := BuildDaemonArgs(c)
	return err
}

// BuildDaemonArgs builds the protocol-contract §18 launch arguments:
//
//	--unix <endpoint> | --pipe <endpoint>
//	--no-ws --home <nativeHome>
//	[--mock-model] [--defaults <path>] [--model-config <path>]
//	[--product-home <productHome>]
//
// The order and the product-home fallback match the Desktop supervisor: a
// model config replaces product-home, and a mock model suppresses it.
func BuildDaemonArgs(cfg SpawnConfig) ([]string, error) {
	if err := cfg.Endpoint.Validate(); err != nil {
		return nil, err
	}
	if cfg.NativeHome == "" {
		return nil, errors.New("supervisor: spawn needs a native home")
	}
	transport := "--unix"
	if cfg.Endpoint.Network == NetworkPipe {
		transport = "--pipe"
	}
	args := make([]string, 0, 12)
	args = append(args, transport, cfg.Endpoint.Address, "--no-ws", "--home", cfg.NativeHome)
	if cfg.MockModel {
		args = append(args, "--mock-model")
	}
	if cfg.DefaultsPath != "" {
		args = append(args, "--defaults", cfg.DefaultsPath)
	}
	if cfg.ModelConfigPath != "" {
		args = append(args, "--model-config", cfg.ModelConfigPath)
	} else if !cfg.MockModel {
		if cfg.ProductHome == "" {
			return nil, errors.New("supervisor: spawn needs a product home when no model config or mock model is set")
		}
		args = append(args, "--product-home", cfg.ProductHome)
	}
	return args, nil
}

// SpawnFunc starts a daemon. It is injectable so tests never launch a real
// binary.
type SpawnFunc func(ctx context.Context, cfg SpawnConfig) (Process, error)

// DefaultSpawn starts the daemon detached with stdout and stderr appended to
// `<nativeHome>/daemon.log`. The log is opened write-only and is never read by
// this package.
func DefaultSpawn(ctx context.Context, cfg SpawnConfig) (Process, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	binary := cfg.ServerPath
	if binary == "" {
		found, err := exec.LookPath(defaultServerBinary)
		if err != nil {
			return nil, fmt.Errorf("supervisor: locate %s: %w", defaultServerBinary, err)
		}
		binary = found
	}
	info, err := os.Stat(binary)
	if err != nil {
		return nil, fmt.Errorf("supervisor: app server binary %s: %w", binary, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("supervisor: app server binary %s is a directory", binary)
	}
	args, err := BuildDaemonArgs(cfg)
	if err != nil {
		return nil, err
	}
	logFile, err := openDaemonLog(cfg.NativeHome)
	if err != nil {
		return nil, err
	}
	// exec.Command is used instead of CommandContext: a detached daemon
	// outlives the caller's context, and Supervisor owns its lifecycle.
	cmd := exec.Command(binary, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = detachedSysProcAttr()
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("supervisor: start app server: %w", err)
	}
	// The child keeps its own handle; the parent must not leak the log fd.
	_ = logFile.Close()
	proc := &execProcess{cmd: cmd, done: make(chan error, 1)}
	go func() {
		proc.done <- cmd.Wait()
		close(proc.done)
	}()
	return proc, nil
}

func openDaemonLog(nativeHome string) (*os.File, error) {
	path := filepath.Join(nativeHome, daemonLogName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("supervisor: open daemon log %s: %w", path, err)
	}
	return file, nil
}

type execProcess struct {
	cmd  *exec.Cmd
	done chan error
}

func (p *execProcess) Pid() int           { return p.cmd.Process.Pid }
func (p *execProcess) Done() <-chan error { return p.done }

// ListenWait bounds waitForListen.
type ListenWait struct {
	Timeout  time.Duration
	Interval time.Duration
}

var DefaultListenWait = ListenWait{Timeout: DefaultListenTimeout, Interval: DefaultListenInterval}

func (w ListenWait) normalized() ListenWait {
	if w.Timeout <= 0 {
		w.Timeout = DefaultListenTimeout
	}
	if w.Interval <= 0 {
		w.Interval = DefaultListenInterval
	}
	return w
}

// waitForListen retries the dial until the endpoint accepts a connection. Only
// a missing endpoint is retried; any other failure is surfaced immediately.
// The successful probe connection is closed before returning.
func waitForListen(ctx context.Context, endpoint Endpoint, dial DialFunc, wait ListenWait) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if dial == nil {
		dial = DefaultDial
	}
	if err := endpoint.Validate(); err != nil {
		return err
	}
	wait = wait.normalized()
	deadline := time.Now().Add(wait.Timeout)
	var lastErr error
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("%w: %s after %s: %v", ErrListenTimeout, endpoint, wait.Timeout, lastErr)
		}
		attemptLimit := ConnectTimeout
		if remaining < attemptLimit {
			attemptLimit = remaining
		}
		attemptCtx, cancel := connectAttemptContext(ctx, attemptLimit)
		conn, err := attemptDial(attemptCtx, endpoint, dial)
		cancel()
		if err == nil {
			if conn != nil {
				_ = conn.Close()
			}
			return nil
		}
		if !IsMissingEndpoint(err) {
			return fmt.Errorf("supervisor: waiting for %s: %w", endpoint, err)
		}
		lastErr = err
		pause := wait.Interval
		if remaining := time.Until(deadline); remaining < pause {
			pause = remaining
		}
		if pause <= 0 {
			return fmt.Errorf("%w: %s after %s: %v", ErrListenTimeout, endpoint, wait.Timeout, lastErr)
		}
		timer := time.NewTimer(pause)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
