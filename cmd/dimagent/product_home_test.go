package main

import (
	"os"
	"path/filepath"
	"testing"
)

// unsetEnv removes an environment variable for one test and restores the
// previous value (including "was absent") afterwards.
func unsetEnv(t *testing.T, name string) {
	t.Helper()
	previous, ok := os.LookupEnv(name)
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("unset %s: %v", name, err)
	}
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(name, previous)
			return
		}
		_ = os.Unsetenv(name)
	})
}

// productHomeFixture points HOME at a temporary directory so the default
// discovery looks there, and clears the variables the resolver reads.
func productHomeFixture(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	unsetEnv(t, productHomeEnv)
	t.Setenv(codeHomeEnv, "")
	t.Setenv(noAutoConnectEnv, "")
	return home
}

// writeProductHome creates <home>/.dimcode with the given entries.
func writeProductHome(t *testing.T, home string, entries ...string) string {
	t.Helper()
	dir := filepath.Join(home, defaultCodeHomeDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir product home: %v", err)
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry)
		if entry == "rust" {
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", entry, err)
			}
			continue
		}
		if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
			t.Fatalf("write %s: %v", entry, err)
		}
	}
	return dir
}

// TestResolveProductHomePrefersExplicit checks that DIMAGENT_PRODUCT_HOME wins
// over both discovery rules and is not reported as auto-discovered.
func TestResolveProductHomePrefersExplicit(t *testing.T) {
	home := productHomeFixture(t)
	writeProductHome(t, home, "rust")
	t.Setenv(productHomeEnv, "/explicit/home")
	t.Setenv(codeHomeEnv, "/code/home")

	got := resolveProductHome()
	if got.path != "/explicit/home" {
		t.Fatalf("path = %q, want the explicit home", got.path)
	}
	if got.auto {
		t.Fatal("an explicit product home must not be reported as discovered")
	}
}

// TestResolveProductHomeUsesCodeHome checks that DIMCODE_HOME is the second
// choice and is logged as discovered.
func TestResolveProductHomeUsesCodeHome(t *testing.T) {
	productHomeFixture(t)
	t.Setenv(codeHomeEnv, "/code/home")

	got := resolveProductHome()
	if got.path != "/code/home" {
		t.Fatalf("path = %q, want DIMCODE_HOME", got.path)
	}
	if !got.auto {
		t.Fatal("a DIMCODE_HOME home must be reported as discovered")
	}
}

// TestResolveProductHomeDiscoversDefaultHome checks each product-home marker:
// the rust/ socket directory or a config/auth file.
func TestResolveProductHomeDiscoversDefaultHome(t *testing.T) {
	for _, entries := range [][]string{
		{"rust"},
		{"config.json"},
		{"auth.json"},
		{"rust", "config.json", "auth.json"},
	} {
		home := productHomeFixture(t)
		writeProductHome(t, home, entries...)

		got := resolveProductHome()
		want := filepath.Join(home, defaultCodeHomeDir)
		if got.path != want {
			t.Fatalf("entries %v: path = %q, want %q", entries, got.path, want)
		}
		if !got.auto {
			t.Fatalf("entries %v: the default home must be reported as discovered", entries)
		}
	}
}

// TestResolveProductHomeWithoutUsableHome checks that a missing, file-shaped,
// or markerless ~/.dimcode keeps the window's not-connected state.
func TestResolveProductHomeWithoutUsableHome(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		productHomeFixture(t)
		if got := resolveProductHome(); got.path != "" || got.auto {
			t.Fatalf("product home = %#v, want none", got)
		}
	})
	t.Run("no home directory", func(t *testing.T) {
		productHomeFixture(t)
		unsetEnv(t, "HOME")
		if got := resolveProductHome(); got.path != "" {
			t.Fatalf("product home = %q, want none without a home directory", got.path)
		}
	})
	t.Run("not a directory", func(t *testing.T) {
		home := productHomeFixture(t)
		if err := os.WriteFile(filepath.Join(home, defaultCodeHomeDir), []byte("x"), 0o600); err != nil {
			t.Fatalf("write file home: %v", err)
		}
		if got := resolveProductHome(); got.path != "" {
			t.Fatalf("product home = %q, want none for a file", got.path)
		}
	})
	t.Run("no markers", func(t *testing.T) {
		home := productHomeFixture(t)
		writeProductHome(t, home, "sessions.db")
		if got := resolveProductHome(); got.path != "" {
			t.Fatalf("product home = %q, want none without a product marker", got.path)
		}
	})
}

// TestResolveProductHomeOptOut checks both ways to keep the window offline: the
// explicit opt-out variable, which also ignores DIMCODE_HOME, and an explicitly
// blank DIMAGENT_PRODUCT_HOME, which the capture scripts already use.
func TestResolveProductHomeOptOut(t *testing.T) {
	home := productHomeFixture(t)
	writeProductHome(t, home, "rust")
	t.Setenv(codeHomeEnv, "/code/home")
	t.Setenv(noAutoConnectEnv, "1")

	if got := resolveProductHome(); got.path != "" || got.auto {
		t.Fatalf("product home = %#v, want none when auto-connect is off", got)
	}

	t.Setenv(productHomeEnv, "")
	if got := resolveProductHome(); got.path != "" {
		t.Fatalf("product home = %q, want none for a blank explicit value", got.path)
	}
}

// TestAutoConnectDisabledValues pins which DIMAGENT_NO_AUTO_CONNECT values opt
// out, so an accidental "0" does not silently disable discovery.
func TestAutoConnectDisabledValues(t *testing.T) {
	for value, want := range map[string]bool{
		"":       false,
		"0":      false,
		"false":  false,
		"off":    false,
		"no":     false,
		"1":      true,
		"true":   true,
		"TRUE":   true,
		"  yes ": true,
	} {
		t.Setenv(noAutoConnectEnv, value)
		if got := autoConnectDisabled(); got != want {
			t.Fatalf("autoConnectDisabled(%q) = %v, want %v", value, got, want)
		}
	}
}

// TestResolveProductHomeDoesNotReadHomeContents checks that discovery only
// looks at directory entries. The marker entries are directories here, so any
// attempt to open config.json or auth.json as a file fails regardless of the
// user running the test; resolution must not depend on their contents.
func TestResolveProductHomeDoesNotReadHomeContents(t *testing.T) {
	home := productHomeFixture(t)
	dir := writeProductHome(t, home, "rust")
	for _, entry := range []string{"config.json", "auth.json"} {
		path := filepath.Join(dir, entry)
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", entry, err)
		}
		if err := os.WriteFile(filepath.Join(path, "secret"), []byte("not-read"), 0o600); err != nil {
			t.Fatalf("write %s/secret: %v", entry, err)
		}
		if _, err := os.ReadFile(path); err == nil {
			t.Fatalf("test setup: %s must not be readable as a file", entry)
		}
	}

	got := resolveProductHome()
	if got.path != dir {
		t.Fatalf("path = %q, want %q", got.path, dir)
	}
}
