package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// RuntimeManager 的运行时名称与来源。
const (
	RuntimeNode   = "node"
	RuntimePython = "python"
	RuntimeUV     = "uv"

	RuntimeSourceDownload = "download"
	RuntimeSourceSystem   = "system"
)

// RuntimeEntry 是 `runtime.list` 的一项。
type RuntimeEntry struct {
	Name string `json:"name"`
	// Source 是配置里的来源；Auto 表示每次启动是否自动获取。
	Source  string  `json:"source"`
	Auto    bool    `json:"auto"`
	Version *string `json:"version,omitempty"` // system 来源在检查前可能缺省
	// Status 是 off / acquiring / failed / starting / ready / down。
	Status string  `json:"status"`
	Reason *string `json:"reason,omitempty"`
}

// RuntimeListResult 是 `runtime.list` 回执（数组）。
type RuntimeListResult []RuntimeEntry

// ParseRuntimeListResult 解码并校验运行时清单；未知字段被忽略。
func ParseRuntimeListResult(raw json.RawMessage) (RuntimeListResult, error) {
	var result RuntimeListResult
	if err := decodeJSON(raw, &result, "runtime.list"); err != nil {
		return result, err
	}
	if result == nil {
		return result, errors.New("runtime.list must be an array")
	}
	for index, entry := range result {
		switch entry.Name {
		case RuntimeNode, RuntimePython, RuntimeUV:
		default:
			return result, fmt.Errorf("runtime.list[%d] has an unknown name %q", index, entry.Name)
		}
		switch entry.Source {
		case RuntimeSourceDownload, RuntimeSourceSystem:
		default:
			return result, fmt.Errorf("runtime.list[%d] has an invalid source %q", index, entry.Source)
		}
		switch entry.Status {
		case "off", "acquiring", "failed", "starting", "ready", "down":
		default:
			return result, fmt.Errorf("runtime.list[%d] has an invalid status %q", index, entry.Status)
		}
	}
	return result, nil
}

// RuntimeInstallParams 是 `runtime.install` 参数。
type RuntimeInstallParams struct {
	Name string `json:"name"`
}

// RuntimeConfigureParams 是 `runtime.configure` 参数。
type RuntimeConfigureParams struct {
	Name   string  `json:"name"`
	Source *string `json:"source,omitempty"`
	Auto   *bool   `json:"auto,omitempty"`
}

// RuntimeWatchItem 是 `runtime.watch` 通过 management.progress 发来的进度项。
type RuntimeWatchItem struct {
	Seq  int64           `json:"seq"`
	Kind string          `json:"kind"`
	Name *string         `json:"name,omitempty"`
	Raw  json.RawMessage `json:"-"`
}

// ParseRuntimeWatchItem 解码一条运行时进度；未知字段被忽略。
func ParseRuntimeWatchItem(raw json.RawMessage) (RuntimeWatchItem, error) {
	var item RuntimeWatchItem
	if err := decodeJSON(raw, &item, "runtime.watch item"); err != nil {
		return item, err
	}
	if item.Kind == "" {
		return item, errors.New("runtime.watch item is missing kind")
	}
	if item.Kind != "watching" && item.Name == nil {
		return item, errors.New("runtime.watch runtime item is missing name")
	}
	item.Raw = raw
	return item, nil
}
