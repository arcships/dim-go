package host

import "testing"

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		left, right string
		want        int
		wantErr     bool
	}{
		{"1.0.0", "1.0.0", 0, false},
		{"1.0.0", "1.0.1", -1, false},
		{"1.0.1", "1.0.0", 1, false},
		{"2.0.0", "1.9.9", 1, false},
		{"1.0.0-alpha", "1.0.0", -1, false},
		{"1.0.0", "1.0.0-alpha", 1, false},
		{"1.0.0-alpha", "1.0.0-alpha.1", -1, false},
		{"1.0.0-alpha.2", "1.0.0-alpha.10", -1, false},
		{"1.0.0-alpha", "1.0.0-beta", -1, false},
		{"1.0.0+build.5", "1.0.0+build.9", 0, false},
		{"0.1.0-dev", "0.1.0", -1, false},
		{"1.2", "1.2.0", 0, true},
		{"", "1.0.0", 0, true},
		{"1.0.0.0", "1.0.0", 0, true},
		{"v1.0.0", "1.0.0", 0, true},
		{"1.0.0-", "1.0.0", 0, true},
	}
	for _, testCase := range cases {
		got, err := compareVersions(testCase.left, testCase.right)
		if testCase.wantErr {
			if err == nil {
				t.Fatalf("compareVersions(%q, %q) 应报错", testCase.left, testCase.right)
			}
			continue
		}
		if err != nil {
			t.Fatalf("compareVersions(%q, %q): %v", testCase.left, testCase.right, err)
		}
		if got != testCase.want {
			t.Fatalf("compareVersions(%q, %q) = %d，期望 %d", testCase.left, testCase.right, got, testCase.want)
		}
	}
}
