package supervisor

import (
	"regexp"
	"testing"
)

func TestWindowsPipeName(t *testing.T) {
	// Expected values are sha256("dimcode:"+home)[:32] computed independently.
	cases := []struct {
		name string
		home string
		want string
	}{
		{
			name: "windows user home",
			home: `C:\Users\alice\AppData\Roaming\dimcode\rust`,
			want: `\\.\pipe\dim-rust-da58c9c0d11f24597e546ed16b290042`,
		},
		{
			name: "verbatim windows path",
			home: `\\?\C:\Users\alice\AppData\Roaming\dimcode\rust`,
			want: `\\.\pipe\dim-rust-ef2b8eb694a8a49e550d4da8e188574f`,
		},
		{
			name: "macOS application support",
			home: `/Users/alice/Library/Application Support/dimcode/rust`,
			want: `\\.\pipe\dim-rust-681b0cfee915433d62a06837034c41bb`,
		},
		{
			name: "temp path",
			home: `/var/folders/aa/bb/T/dim/rust`,
			want: `\\.\pipe\dim-rust-728c40372580cbafa0ce209b92dfd87d`,
		},
		{
			name: "empty path",
			home: ``,
			want: `\\.\pipe\dim-rust-125fe758cd40f24c4ae888a9ee5fe3f5`,
		},
	}
	shape := regexp.MustCompile(`^\\\\\.\\pipe\\dim-rust-[0-9a-f]{32}$`)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := windowsPipeName(tc.home)
			if got != tc.want {
				t.Fatalf("windowsPipeName(%q) = %q, want %q", tc.home, got, tc.want)
			}
			if !shape.MatchString(got) {
				t.Fatalf("windowsPipeName(%q) = %q, want the §18 pipe shape", tc.home, got)
			}
		})
	}
}

func TestWindowsPipeNameIsStable(t *testing.T) {
	home := `C:\Users\alice\AppData\Roaming\dimcode\rust`
	first := windowsPipeName(home)
	for i := 0; i < 10; i++ {
		if got := windowsPipeName(home); got != first {
			t.Fatalf("iteration %d: %q != %q", i, got, first)
		}
	}
	if len(first) != len(windowsPipePrefix)+32 {
		t.Fatalf("pipe name %q has unexpected length %d", first, len(first))
	}
	if first == windowsPipeName(home+"x") {
		t.Fatal("different homes must not share a pipe name")
	}
}
