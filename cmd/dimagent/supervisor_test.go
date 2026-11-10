package main

import (
	"path/filepath"
	"testing"
)

// TestSupervisorConfigPassesModelConfigPath checks that DIMAGENT_MODEL_CONFIG
// reaches the supervisor as the daemon's --model-config path. The path is only
// forwarded: the entry point never reads the file.
func TestSupervisorConfigPassesModelConfigPath(t *testing.T) {
	t.Setenv("DIMAGENT_MODEL_CONFIG", "/etc/dim/model.json")
	t.Setenv("DIMAGENT_MOCK_MODEL", "")

	cfg := supervisorConfig("/tmp/dim-product-home")
	if cfg.ModelConfigPath != "/etc/dim/model.json" {
		t.Fatalf("ModelConfigPath = %q, want /etc/dim/model.json", cfg.ModelConfigPath)
	}
	if cfg.ProductHome != "/tmp/dim-product-home" {
		t.Fatalf("ProductHome = %q, want /tmp/dim-product-home", cfg.ProductHome)
	}
	if cfg.MockModel {
		t.Fatal("MockModel must stay opt-in")
	}
}

// TestSupervisorConfigWithoutModelConfigPath checks that an unset or blank
// value leaves the daemon's default resolution alone.
func TestSupervisorConfigWithoutModelConfigPath(t *testing.T) {
	for _, value := range []string{"", "   "} {
		t.Setenv("DIMAGENT_MODEL_CONFIG", value)
		t.Setenv("DIMAGENT_MOCK_MODEL", "")

		cfg := supervisorConfig("/tmp/dim-product-home")
		if cfg.ModelConfigPath != "" {
			t.Fatalf("ModelConfigPath = %q for env %q, want empty", cfg.ModelConfigPath, value)
		}
	}
}

// TestSupervisorConfigKeepsMockAndHome checks that the model config adds a
// daemon flag without changing what the existing environment already sets.
func TestSupervisorConfigKeepsMockAndHome(t *testing.T) {
	t.Setenv("DIMAGENT_MODEL_CONFIG", filepath.Join(t.TempDir(), "model.json"))
	t.Setenv("DIMAGENT_MOCK_MODEL", "1")

	cfg := supervisorConfig("/tmp/dim-product-home")
	if cfg.ProductHome != "/tmp/dim-product-home" {
		t.Fatalf("ProductHome = %q, want the configured home", cfg.ProductHome)
	}
	if !cfg.MockModel {
		t.Fatal("MockModel must still follow DIMAGENT_MOCK_MODEL")
	}
	if cfg.ModelConfigPath == "" {
		t.Fatal("ModelConfigPath must be kept alongside the mock model")
	}
}

// TestNewSupervisorStillRequiresProductHome checks that the model config does
// not relax the supervisor's product-home requirement.
func TestNewSupervisorStillRequiresProductHome(t *testing.T) {
	t.Setenv("DIMAGENT_MODEL_CONFIG", "/etc/dim/model.json")
	t.Setenv("DIMAGENT_MOCK_MODEL", "1")

	if _, err := newSupervisor(""); err == nil {
		t.Fatal("newSupervisor must still reject an empty product home")
	}
	sup, err := newSupervisor("/tmp/dim-product-home")
	if err != nil {
		t.Fatalf("newSupervisor: %v", err)
	}
	if sup == nil {
		t.Fatal("newSupervisor returned a nil supervisor")
	}
}
