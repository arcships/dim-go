package host

import (
	"fmt"
	"strconv"
	"strings"
)

// semver 是解析后的版本号。只支持主版本.次版本.修订号加可选的预发布段与构建元数据。
type semver struct {
	major int
	minor int
	patch int
	pre   []string
}

func parseSemver(value string) (semver, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return semver{}, fmt.Errorf("host: 版本号为空")
	}
	// 构建元数据不参与比较，直接丢弃。
	if idx := strings.IndexByte(raw, '+'); idx >= 0 {
		if idx == len(raw)-1 {
			return semver{}, fmt.Errorf("host: 版本号 %q 的构建元数据为空", value)
		}
		raw = raw[:idx]
	}
	core := raw
	pre := ""
	if idx := strings.IndexByte(raw, '-'); idx >= 0 {
		core, pre = raw[:idx], raw[idx+1:]
		if pre == "" {
			return semver{}, fmt.Errorf("host: 版本号 %q 的预发布段为空", value)
		}
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return semver{}, fmt.Errorf("host: 版本号 %q 不是 x.y.z 形式", value)
	}
	var numbers [3]int
	for i, part := range parts {
		if part == "" {
			return semver{}, fmt.Errorf("host: 版本号 %q 有空的数字段", value)
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return semver{}, fmt.Errorf("host: 版本号 %q 的数字段不是十进制", value)
			}
		}
		number, err := strconv.Atoi(part)
		if err != nil {
			return semver{}, fmt.Errorf("host: 版本号 %q 的数字段不可解析: %w", value, err)
		}
		numbers[i] = number
	}

	var identifiers []string
	if pre != "" {
		identifiers = strings.Split(pre, ".")
		for _, identifier := range identifiers {
			if identifier == "" {
				return semver{}, fmt.Errorf("host: 版本号 %q 的预发布段为空", value)
			}
		}
	}
	return semver{major: numbers[0], minor: numbers[1], patch: numbers[2], pre: identifiers}, nil
}

// compareVersions 比较两个版本：a 小于 b 返回 -1，相等返回 0，大于返回 1。
func compareVersions(a, b string) (int, error) {
	left, err := parseSemver(a)
	if err != nil {
		return 0, err
	}
	right, err := parseSemver(b)
	if err != nil {
		return 0, err
	}
	leftNumbers := [...]int{left.major, left.minor, left.patch}
	rightNumbers := [...]int{right.major, right.minor, right.patch}
	for i := range leftNumbers {
		switch {
		case leftNumbers[i] < rightNumbers[i]:
			return -1, nil
		case leftNumbers[i] > rightNumbers[i]:
			return 1, nil
		}
	}

	// 正式版本高于同号的预发布版本。
	switch {
	case len(left.pre) == 0 && len(right.pre) == 0:
		return 0, nil
	case len(left.pre) == 0:
		return 1, nil
	case len(right.pre) == 0:
		return -1, nil
	}

	for i := 0; i < len(left.pre) && i < len(right.pre); i++ {
		if compared := comparePrereleaseIdentifier(left.pre[i], right.pre[i]); compared != 0 {
			return compared, nil
		}
	}
	switch {
	case len(left.pre) < len(right.pre):
		return -1, nil
	case len(left.pre) > len(right.pre):
		return 1, nil
	default:
		return 0, nil
	}
}

// comparePrereleaseIdentifier 按 semver 规则比较单个预发布标识：
// 纯数字按数值比较且低于字母标识，字母按字典序比较。
func comparePrereleaseIdentifier(a, b string) int {
	aNumber, aNumeric := parseNumericIdentifier(a)
	bNumber, bNumeric := parseNumericIdentifier(b)
	switch {
	case aNumeric && bNumeric:
		switch {
		case aNumber < bNumber:
			return -1
		case aNumber > bNumber:
			return 1
		default:
			return 0
		}
	case aNumeric:
		return -1
	case bNumeric:
		return 1
	default:
		return strings.Compare(a, b)
	}
}

func parseNumericIdentifier(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return number, true
}
