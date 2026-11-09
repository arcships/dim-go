// This file checks the build-version path end to end: the -version line, the
// "dev" fallback of a plain build and the -X main.version stamp the packaging
// script uses.

package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver"
)

// TestVersionLine pins the one line -version prints: the app name, the build
// version and the wire protocol the client speaks.
func TestVersionLine(t *testing.T) {
	saved := version
	t.Cleanup(func() { version = saved })

	for _, tc := range []struct{ stamped, want string }{
		{stamped: "1.2.3", want: "DimAgent 1.2.3 (protocol v3)"},
		{stamped: "dev", want: "DimAgent dev (protocol v3)"},
		{stamped: "", want: "DimAgent dev (protocol v3)"},
		{stamped: "  ", want: "DimAgent dev (protocol v3)"},
	} {
		version = tc.stamped
		if got := versionLine(); got != tc.want {
			t.Errorf("version %q draws %q, want %q", tc.stamped, got, tc.want)
		}
	}
	if appserver.WireProtocolVersion != 3 {
		t.Errorf("the wire protocol moved to %d; the pinned line needs updating", appserver.WireProtocolVersion)
	}
}

// TestPrintVersionHandlesBothFlags checks the argument handling the main guard
// relies on: -version and --version print and report handled, while other
// arguments (including none) report false so the window still starts.
func TestPrintVersionHandlesBothFlags(t *testing.T) {
	want := versionLine()
	for _, args := range [][]string{{"-version"}, {"--version"}, {"--verbose", "-version"}} {
		var out bytes.Buffer
		if ok := printVersion(args, &out); !ok {
			t.Errorf("printVersion(%q) did not handle the version flag", args)
		}
		if got := out.String(); got != want+"\n" {
			t.Errorf("printVersion(%q) wrote %q, want %q", args, got, want+"\n")
		}
	}
	for _, args := range [][]string{nil, {}, {"--verbose"}, {"-v"}, {"version"}} {
		var out bytes.Buffer
		if ok := printVersion(args, &out); ok {
			t.Errorf("printVersion(%q) handled a non-version argument", args)
		}
		if out.Len() != 0 {
			t.Errorf("printVersion(%q) wrote %q, want nothing", args, out.String())
		}
	}
}

// TestVersionFlagProcess checks the shipped binary: `dimagent -version` and
// `dimagent --version` print the line and exit 0 without opening a window, and
// a binary built with the packaging script's -ldflags reports the stamped
// version. It builds cmd/dimagent, so it is skipped in -short runs and when the
// go tool is unavailable.
func TestVersionFlagProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the command binary")
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go tool is not on PATH: %v", err)
	}
	build := func(name string, ldflags string) string {
		t.Helper()
		bin := filepath.Join(t.TempDir(), name)
		args := []string{"build", "-o", bin}
		if ldflags != "" {
			args = append(args, "-ldflags", ldflags)
		}
		args = append(args, ".")
		cmd := exec.Command(goTool, args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build %s: %v\n%s", name, err, out)
		}
		return bin
	}
	runVersion := func(bin string, flag string, want string) {
		t.Helper()
		out, err := exec.Command(bin, flag).Output()
		if err != nil {
			t.Fatalf("%s %s exited with %v, output %q", filepath.Base(bin), flag, err, out)
		}
		if got := string(out); got != want+"\n" {
			t.Errorf("%s %s wrote %q, want %q", filepath.Base(bin), flag, got, want+"\n")
		}
	}

	plain := build("dimagent-plain", "")
	plainWant := fmt.Sprintf("DimAgent dev (protocol v%d)", appserver.WireProtocolVersion)
	for _, flag := range []string{"-version", "--version"} {
		runVersion(plain, flag, plainWant)
	}

	stamped := build("dimagent-stamped", "-X main.version=1.2.3")
	stampedWant := fmt.Sprintf("DimAgent 1.2.3 (protocol v%d)", appserver.WireProtocolVersion)
	runVersion(stamped, "-version", stampedWant)
}
