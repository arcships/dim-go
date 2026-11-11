package host

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

func testDigest(data []byte) string {
	sum := sha512.Sum512(data)
	return base64.StdEncoding.EncodeToString(sum[:])
}

func TestUpdateManifestURLMatchesExistingChannel(t *testing.T) {
	cases := []struct {
		baseURL  string
		channel  string
		platform string
		arch     string
		want     string
	}{
		{
			"", "", "darwin", "arm64",
			"https://dimcode.echooai.com/updates/stable/mac/arm64/latest-mac.yml",
		},
		{
			"https://dimcode.echooai.com/", "updates/stable", "darwin", "amd64",
			"https://dimcode.echooai.com/updates/stable/mac/x64/latest-mac.yml",
		},
		{
			"https://example.test", "updates/beta", "windows", "amd64",
			"https://example.test/updates/beta/win/x64/latest.yml",
		},
		{
			"https://example.test", "updates/stable", "linux", "amd64",
			"https://example.test/updates/stable/linux/x64/latest-linux.yml",
		},
	}
	for _, testCase := range cases {
		got, err := UpdateManifestURL(testCase.baseURL, testCase.channel, testCase.platform, testCase.arch)
		if err != nil {
			t.Fatalf("UpdateManifestURL(%s/%s): %v", testCase.platform, testCase.arch, err)
		}
		if got != testCase.want {
			t.Fatalf("UpdateManifestURL(%s/%s) = %q，期望 %q", testCase.platform, testCase.arch, got, testCase.want)
		}
	}
	for _, unsupported := range [][2]string{{"plan9", "amd64"}, {"darwin", "386"}, {"windows", "arm64"}, {"linux", "arm64"}} {
		if _, err := UpdateManifestURL("https://example.test", "", unsupported[0], unsupported[1]); err == nil {
			t.Fatalf("%s/%s 应返回不支持", unsupported[0], unsupported[1])
		}
	}
}

func TestParseManifestElectronBuilderShape(t *testing.T) {
	content := []byte("2.1.0")
	manifestURL := "https://example.test/updates/stable/mac/arm64/latest-mac.yml"
	body := strings.Join([]string{
		"version: 2.1.0",
		"files:",
		"  - url: DimAgent-2.1.0-arm64.dmg",
		"    sha512: " + testDigest(content),
		"    size: " + "5",
		"    blockMapSize: 99",
		"  - url: DimAgent-2.1.0-arm64.zip",
		"    sha512: " + testDigest([]byte("zip")),
		"    size: 3",
		"path: DimAgent-2.1.0-arm64.dmg",
		"sha512: " + testDigest(content),
		"releaseDate: '2026-11-10T00:00:00.000Z'",
	}, "\n")

	release, err := ParseManifest(manifestURL, []byte(body))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if release.Version != "2.1.0" {
		t.Fatalf("版本 = %q", release.Version)
	}
	wantURL := "https://example.test/updates/stable/mac/arm64/DimAgent-2.1.0-arm64.dmg"
	if release.ArtifactURL != wantURL {
		t.Fatalf("产物地址 = %q，期望 %q", release.ArtifactURL, wantURL)
	}
	if release.SHA512 != testDigest(content) {
		t.Fatalf("摘要 = %q", release.SHA512)
	}
	if release.Size != 5 {
		t.Fatalf("大小 = %d", release.Size)
	}
	if release.ReleaseDate != "2026-11-10T00:00:00.000Z" {
		t.Fatalf("发布时间 = %q", release.ReleaseDate)
	}
}

func TestParseManifestFallsBackToTopLevelFields(t *testing.T) {
	content := []byte("artifact")
	body := strings.Join([]string{
		"version: '1.2.3'",
		"path: 'pkg/App-1.2.3.bin'",
		"sha512: " + testDigest(content),
		"size: 8",
	}, "\n")
	release, err := ParseManifest("https://example.test/a/b/latest-mac.yml", []byte(body))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if release.ArtifactURL != "https://example.test/a/b/pkg/App-1.2.3.bin" {
		t.Fatalf("产物地址 = %q", release.ArtifactURL)
	}
	if release.Version != "1.2.3" || release.Size != 8 {
		t.Fatalf("清单 = %+v", release)
	}
}

func TestParseManifestRejectsIncompleteManifests(t *testing.T) {
	digest := testDigest([]byte("x"))
	cases := []struct {
		name string
		body string
	}{
		{"缺 version", "path: app.bin\nsha512: " + digest},
		{"version 不合法", "version: v1\npath: app.bin\nsha512: " + digest},
		{"缺产物地址", "version: 1.0.0\nsha512: " + digest},
		{"缺摘要", "version: 1.0.0\npath: app.bin"},
		{"摘要不合法", "version: 1.0.0\npath: app.bin\nsha512: not-a-digest"},
		{"大小不合法", "version: 1.0.0\npath: app.bin\nsha512: " + digest + "\nsize: 12x"},
	}
	for _, testCase := range cases {
		if _, err := ParseManifest("https://example.test/latest.yml", []byte(testCase.body)); err == nil {
			t.Fatalf("%s 应被拒绝", testCase.name)
		}
	}
	if _, err := ParseManifest("ftp://example.test/latest.yml", []byte("version: 1.0.0")); err == nil {
		t.Fatal("非 http(s) 清单地址应被拒绝")
	}
}

func TestParseManifestAcceptsHexDigest(t *testing.T) {
	content := []byte("hex-digest")
	sum := sha512.Sum512(content)
	body := "version: 1.0.0\npath: app.bin\nsha512: " + hex.EncodeToString(sum[:])
	release, err := ParseManifest("https://example.test/latest.yml", []byte(body))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	decoded, err := decodeDigest(release.SHA512)
	if err != nil {
		t.Fatalf("decodeDigest: %v", err)
	}
	if string(decoded) != string(sum[:]) {
		t.Fatal("十六进制摘要解码结果不一致")
	}
}

func TestParseManifestEscapesSpacesInArtifactURL(t *testing.T) {
	body := "version: 9.9.9\nfiles:\n  - url: DimAgent Setup 9.9.9.exe\n    sha512: " +
		testDigest([]byte("x")) + "\n    size: 1\n"
	release, err := ParseManifest("https://example.test/updates/stable/win/x64/latest.yml", []byte(body))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	want := "https://example.test/updates/stable/win/x64/DimAgent%20Setup%209.9.9.exe"
	if release.ArtifactURL != want {
		t.Fatalf("产物地址 = %q，期望 %q", release.ArtifactURL, want)
	}
}
