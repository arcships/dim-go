package main

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	// productHomeEnv names the product home explicitly. A set-but-blank value
	// starts the window without a daemon, which is how the capture scripts
	// take a disconnected screenshot.
	productHomeEnv = "DIMAGENT_PRODUCT_HOME"
	// codeHomeEnv is the DimCode home used by the daemon and the desktop app.
	codeHomeEnv = "DIMCODE_HOME"
	// noAutoConnectEnv disables discovery: the window starts without a daemon
	// even when a local product home exists.
	noAutoConnectEnv = "DIMAGENT_NO_AUTO_CONNECT"
	// defaultCodeHomeDir is the product home under the user's home directory.
	defaultCodeHomeDir = ".dimcode"
)

// productHome is the path handed to the supervisor and whether discovery found
// it instead of the environment naming it. auto is what makes the entry point
// log the choice; an explicit DIMAGENT_PRODUCT_HOME stays silent.
type productHome struct {
	path string
	auto bool
}

// resolveProductHome picks the product home for this launch. Resolution order:
//
//  1. DIMAGENT_PRODUCT_HOME when it is set, even when blank (blank means "no
//     daemon", the existing opt-out the capture scripts use);
//  2. nothing when DIMAGENT_NO_AUTO_CONNECT is set;
//  3. DIMCODE_HOME when it is set and not blank;
//  4. <home>/.dimcode when it is a directory that looks like a product home;
//  5. nothing, which keeps the window's not-connected state.
//
// It only names a path: it checks which entries the default home has and never
// opens a session, SQLite, JSONL or daemon.log file. The supervisor alone
// talks to the daemon.
func resolveProductHome() productHome {
	if value, ok := os.LookupEnv(productHomeEnv); ok {
		// An explicit value wins even when blank: DIMAGENT_PRODUCT_HOME= is an
		// explicit "do not connect", not "discover something for me".
		return productHome{path: strings.TrimSpace(value)}
	}
	if autoConnectDisabled() {
		return productHome{}
	}
	if value := strings.TrimSpace(os.Getenv(codeHomeEnv)); value != "" {
		return productHome{path: value, auto: true}
	}
	if path := defaultProductHome(); path != "" {
		return productHome{path: path, auto: true}
	}
	return productHome{}
}

// autoConnectDisabled reports whether the environment opts out of discovery.
// An absent, blank, or false-looking value keeps discovery on; anything else
// turns it off so `DIMAGENT_NO_AUTO_CONNECT=1` does what it says.
func autoConnectDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(noAutoConnectEnv))) {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// defaultProductHome returns <home>/.dimcode when it exists, is a directory,
// and carries a product-home marker: the rust/ directory that holds the daemon
// socket, or a config/auth file. Marker detection is stat-only; the files are
// never read.
func defaultProductHome() string {
	userHome, err := os.UserHomeDir()
	if err != nil || userHome == "" {
		return ""
	}
	candidate := filepath.Join(userHome, defaultCodeHomeDir)
	info, err := os.Stat(candidate)
	if err != nil || !info.IsDir() {
		return ""
	}
	if rustDir, err := os.Stat(filepath.Join(candidate, "rust")); err == nil && rustDir.IsDir() {
		return candidate
	}
	for _, marker := range []string{"config.json", "auth.json"} {
		if _, err := os.Stat(filepath.Join(candidate, marker)); err == nil {
			return candidate
		}
	}
	return ""
}
