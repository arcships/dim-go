// This file owns the binary's build version: the one fact a plain `go build`
// cannot know and a package script stamps in.
//
// version is a package-level string so the linker can set it:
// `go build -ldflags "-X main.version=1.2.3"` (scripts/package-macos.sh does
// this). A plain build leaves "dev", and the About page says so rather than
// inventing a number. The line `-version` prints names the app, the version and
// the App Server protocol the client speaks, so a packaged binary can answer
// "what am I" without opening a window or starting the daemon.

package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// version is the build version stamped at link time with -X main.version.
// A plain go build leaves "dev".
var version = "dev"

// appVersion is the version the window shows: the stamped one, or "dev" when a
// build supplied none (an empty -X value would otherwise draw a blank row).
func appVersion() string {
	if strings.TrimSpace(version) == "" {
		return "dev"
	}
	return version
}

// versionLine is the one line -version prints: the app name, the build version
// and the wire protocol version from the client package.
func versionLine() string {
	return fmt.Sprintf("DimAgent %s (protocol v%d)", appVersion(), appserver.WireProtocolVersion)
}

// printVersion handles -version and --version before the daemon or window
// starts. It reports whether an argument asked for the version, so the caller
// returns instead of opening a window; the printed line and an empty main return
// are the process's zero exit status.
func printVersion(args []string, out io.Writer) bool {
	for _, arg := range args {
		if arg == "-version" || arg == "--version" {
			fmt.Fprintln(out, versionLine())
			return true
		}
	}
	return false
}
