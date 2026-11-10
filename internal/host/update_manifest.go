package host

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	// defaultUpdateBaseURL 是现有客户端使用的更新源基址。
	defaultUpdateBaseURL = "https://dimcode.echooai.com"
	// defaultUpdateChannel 是现有客户端使用的更新通道路径。
	defaultUpdateChannel = "updates/stable"
	// maxManifestBytes 是版本清单的读取上限。
	maxManifestBytes = 1 << 20
)

// Release 是版本源给出的一次发布。形状对齐现有客户端的 electron-builder 通用清单。
type Release struct {
	// Version 是发布版本号，必须是 x.y.z 形式。
	Version string
	// ArtifactURL 是安装产物的绝对地址。
	ArtifactURL string
	// SHA512 是产物的 sha512 摘要，base64 或十六进制。
	SHA512 string
	// Size 是产物字节数；未知时为 0。
	Size int64
	// ReleaseDate 是版本源给出的发布时间原文，仅保留用于展示，不参与判定。
	ReleaseDate string
}

// UpdateManifestURL 按现有客户端的渠道约定推导版本清单地址。
// platform 取 runtime.GOOS（darwin/windows/linux），arch 取 runtime.GOARCH。
func UpdateManifestURL(baseURL, channel, platform, arch string) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultUpdateBaseURL
	}
	if strings.TrimSpace(channel) == "" {
		channel = defaultUpdateChannel
	}
	platformPath, manifestName, err := updatePlatformPath(platform, arch)
	if err != nil {
		return "", err
	}
	manifest := strings.TrimRight(strings.TrimSpace(baseURL), "/") +
		"/" + strings.Trim(strings.TrimSpace(channel), "/") +
		"/" + platformPath + "/" + manifestName
	parsed, err := url.Parse(manifest)
	if err != nil {
		return "", fmt.Errorf("host: 更新源地址不可解析: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("host: 更新源地址必须是 http 或 https: %s", manifest)
	}
	return parsed.String(), nil
}

func updatePlatformPath(platform, arch string) (string, string, error) {
	switch platform {
	case "darwin", "mac":
		if arch != "arm64" && arch != "x64" && arch != "amd64" {
			return "", "", fmt.Errorf("host: 不支持在 darwin/%s 上更新", arch)
		}
		if arch == "amd64" {
			arch = "x64"
		}
		return "mac/" + arch, "latest-mac.yml", nil
	case "windows", "win":
		if arch != "x64" && arch != "amd64" {
			return "", "", fmt.Errorf("host: 不支持在 windows/%s 上更新", arch)
		}
		return "win/x64", "latest.yml", nil
	case "linux":
		if arch != "x64" && arch != "amd64" {
			return "", "", fmt.Errorf("host: 不支持在 linux/%s 上更新", arch)
		}
		return "linux/x64", "latest-linux.yml", nil
	default:
		return "", "", fmt.Errorf("host: 不支持在 %s/%s 上更新", platform, arch)
	}
}

// ParseManifest 解析 electron-builder 形状的版本清单（latest*.yml 的最小 YAML 子集）：
// 顶层 version/path/sha512/size，以及 files 列表里的 url/sha512/size。
// 清单缺少版本号、产物地址或摘要一律视为失败，不进入下载。
func ParseManifest(manifestURL string, data []byte) (Release, error) {
	base, err := url.Parse(manifestURL)
	if err != nil || !base.IsAbs() {
		return Release{}, fmt.Errorf("host: 版本清单地址不可解析: %q", manifestURL)
	}
	fields, err := parseManifestFields(data)
	if err != nil {
		return Release{}, err
	}
	release := Release{
		Version:     fields.version,
		SHA512:      fields.sha512,
		ReleaseDate: fields.releaseDate,
	}
	if release.Version == "" {
		return Release{}, fmt.Errorf("host: 版本清单缺少 version")
	}
	if _, err := parseSemver(release.Version); err != nil {
		return Release{}, fmt.Errorf("host: 版本清单的 version 不合法: %w", err)
	}
	if fields.url == "" {
		return Release{}, fmt.Errorf("host: 版本清单缺少产物地址")
	}
	if release.SHA512 == "" {
		return Release{}, fmt.Errorf("host: 版本清单缺少 sha512 摘要")
	}
	if _, err := decodeDigest(release.SHA512); err != nil {
		return Release{}, fmt.Errorf("host: 版本清单的 sha512 摘要不合法: %w", err)
	}
	if fields.size != "" {
		size, err := strconv.ParseInt(fields.size, 10, 64)
		if err != nil || size < 0 {
			return Release{}, fmt.Errorf("host: 版本清单的 size 不合法: %q", fields.size)
		}
		release.Size = size
	}
	artifactURL, err := resolveArtifactURL(base, fields.url)
	if err != nil {
		return Release{}, err
	}
	release.ArtifactURL = artifactURL
	return release, nil
}

func resolveArtifactURL(base *url.URL, raw string) (string, error) {
	reference, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("host: 产物地址不可解析: %w", err)
	}
	resolved := base.ResolveReference(reference)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", fmt.Errorf("host: 产物地址必须是 http 或 https: %s", resolved)
	}
	return resolved.String(), nil
}

// decodeDigest 把 base64 或十六进制摘要解码成字节，并校验是 64 字节的 sha512。
// 十六进制串同时也是合法的 base64 字符，所以每种编码解出后都要看长度，不匹配就继续试下一种。
func decodeDigest(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	lengthErr := fmt.Errorf("摘要长度不是 64 字节")
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		decoded, err := encoding.DecodeString(trimmed)
		if err != nil {
			continue
		}
		if len(decoded) == 64 {
			return decoded, nil
		}
	}
	if decoded, err := hex.DecodeString(trimmed); err == nil {
		if len(decoded) == 64 {
			return decoded, nil
		}
	}
	return nil, lengthErr
}

// manifestFields 是清单解析的中间结果。
type manifestFields struct {
	version     string
	url         string
	sha512      string
	size        string
	releaseDate string
}

// manifestFile 是 files 列表里的一条产物。
type manifestFile struct {
	url    string
	sha512 string
	size   string
}

func parseManifestFields(data []byte) (manifestFields, error) {
	var (
		top        manifestFile
		topVersion string
		topDate    string
		files      []manifestFile
		section    string
		current    *manifestFile
	)

	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indented := len(line) != len(strings.TrimLeft(line, " \t"))
		if !indented {
			key, value, ok := splitManifestKey(trimmed)
			if !ok {
				section = ""
				current = nil
				continue
			}
			section = ""
			current = nil
			switch key {
			case "version":
				topVersion = value
			case "path":
				top.url = value
			case "sha512":
				top.sha512 = value
			case "size":
				top.size = value
			case "releaseDate":
				topDate = value
			case "files":
				section = "files"
			}
			continue
		}
		if section != "files" {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
			files = append(files, manifestFile{})
			current = &files[len(files)-1]
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if trimmed == "" {
				continue
			}
		}
		if current == nil {
			continue
		}
		key, value, ok := splitManifestKey(trimmed)
		if !ok {
			continue
		}
		switch key {
		case "url":
			current.url = value
		case "sha512":
			current.sha512 = value
		case "size":
			current.size = value
		}
	}

	fields := manifestFields{
		version:     topVersion,
		url:         top.url,
		sha512:      top.sha512,
		size:        top.size,
		releaseDate: topDate,
	}
	// 清单可能列出多个产物（dmg 与 zip 等），取第一条带地址的条目；
	// 条目缺少的摘要或大小回落到顶层字段。
	for _, file := range files {
		if file.url == "" {
			continue
		}
		fields.url = file.url
		fields.sha512 = file.sha512
		fields.size = file.size
		if fields.sha512 == "" {
			fields.sha512 = top.sha512
		}
		if fields.size == "" {
			fields.size = top.size
		}
		break
	}
	return fields, nil
}

func splitManifestKey(line string) (key, value string, ok bool) {
	idx := strings.IndexByte(line, ':')
	if idx <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	if len(value) >= 2 {
		if (value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"') {
			value = value[1 : len(value)-1]
		}
	}
	if key == "" {
		return "", "", false
	}
	return key, value, true
}
