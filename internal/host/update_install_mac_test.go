package host

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMacSignatureVerifierChecksTeamAndNotarization(t *testing.T) {
	base := t.TempDir()
	oldApp := filepath.Join(base, "old", "DimAgent.app")
	newApp := filepath.Join(base, "new", "DimAgent.app")

	newHooks := func(newTeam string, calls *[]string) InstallHooks {
		return InstallHooks{
			Run: func(_ context.Context, name string, args ...string) (string, error) {
				*calls = append(*calls, name+" "+strings.Join(args, " "))
				switch {
				case name == macCodesign && len(args) > 0 && args[0] == "--verify":
					return "verified", nil
				case name == macCodesign && len(args) > 0 && args[0] == "-d":
					team := "TEAM-A"
					if args[len(args)-1] == newApp {
						team = newTeam
					}
					return "Executable=" + args[len(args)-1] + "\nTeamIdentifier=" + team + "\n", nil
				default:
					return "", nil
				}
			},
		}
	}

	var calls []string
	installer := macInstaller{hooks: newHooks("TEAM-A", &calls)}
	if err := installer.verifyWithCommands(context.Background(), oldApp, newApp); err != nil {
		t.Fatalf("签名一致应通过: %v", err)
	}
	want := []string{
		macCodesign + " --verify --deep --strict --verbose=2 " + newApp,
		macCodesign + " -d --verbose=4 " + oldApp,
		macCodesign + " -d --verbose=4 " + newApp,
		macStapler + " validate " + newApp,
		macSpctl + " --assess --type execute --verbose=4 " + newApp,
	}
	if len(calls) != len(want) {
		t.Fatalf("命令调用 = %v", calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("第 %d 条命令 = %q，期望 %q", i+1, calls[i], want[i])
		}
	}

	calls = nil
	mismatch := macInstaller{hooks: newHooks("TEAM-B", &calls)}
	if err := mismatch.verifyWithCommands(context.Background(), oldApp, newApp); err == nil {
		t.Fatal("TeamIdentifier 不一致应失败")
	}
	for _, call := range calls {
		if strings.HasPrefix(call, macXcrun) || strings.HasPrefix(call, macSpctl) {
			t.Fatalf("TeamIdentifier 不一致后仍继续公证校验: %v", calls)
		}
	}
}

func TestMacInstallerFallsBackToUnzipWhenDittoMissing(t *testing.T) {
	content := []byte("2.1.0")
	dir := t.TempDir()
	bundle := filepath.Join(dir, "DimAgent.app")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatalf("建旧 .app: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "version.txt"), []byte("1.0.0"), 0o644); err != nil {
		t.Fatalf("写旧版本: %v", err)
	}

	var used []string
	hooks := InstallHooks{
		Run: func(_ context.Context, name string, args ...string) (string, error) {
			used = append(used, name)
			if name == macDitto {
				return "", &exec.Error{Name: macDitto, Err: exec.ErrNotFound}
			}
			if name == macUnzip {
				staging := args[len(args)-1]
				newApp := filepath.Join(staging, "DimAgent.app")
				if err := os.MkdirAll(newApp, 0o755); err != nil {
					return "", err
				}
				return "", os.WriteFile(filepath.Join(newApp, "version.txt"), []byte("2.1.0"), 0o644)
			}
			return "", nil
		},
		Spawn: func(context.Context, string, []string, []string) error { return nil },
	}

	updater, _ := newReadyUpdater(t, "2.1.0", content,
		cfgMac(bundle, hooks, func(context.Context, string, string) error { return nil }, nil))
	if err := updater.Switch(context.Background()); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	if len(used) < 2 || used[0] != macDitto || used[1] != macUnzip {
		t.Fatalf("解包命令 = %v，期望先 ditto 后 unzip", used)
	}
	if got := readText(t, filepath.Join(bundle, "version.txt")); got != "2.1.0" {
		t.Fatalf("安装后版本 = %q", got)
	}
}
