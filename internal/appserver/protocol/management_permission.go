package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// 权限档位（与 wire 一致，kebab-case）。
const (
	PermissionPresetReadOnly       = "read-only"
	PermissionPresetWorkspaceWrite = "workspace-write"
	PermissionPresetFullAccess     = "full-access"
)

// PermissionPolicy 是会话分支的权限策略。
type PermissionPolicy struct {
	Preset          string   `json:"preset"`
	AllowedTools    []string `json:"allowedTools"`
	DisallowedTools []string `json:"disallowedTools"`
}

// Validate 校验档位取值；未知档位按不可重试处理。
func (p PermissionPolicy) Validate(label string) error {
	switch p.Preset {
	case PermissionPresetReadOnly, PermissionPresetWorkspaceWrite, PermissionPresetFullAccess:
		return nil
	default:
		return fmt.Errorf("%s has an invalid preset %q", label, p.Preset)
	}
}

// PermissionGetParams 是 `session.permission.get` 参数。
type PermissionGetParams struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
}

// PermissionResult 是 `session.permission.get` / `update` 的回执。
type PermissionResult struct {
	Version int64            `json:"version"`
	Policy  PermissionPolicy `json:"policy"`
}

// ParsePermissionResult 解码并校验一次权限读取；未知字段被忽略。
func ParsePermissionResult(raw json.RawMessage) (PermissionResult, error) {
	var result PermissionResult
	if err := decodeJSON(raw, &result, "session.permission"); err != nil {
		return result, err
	}
	if err := result.Policy.Validate("session.permission policy"); err != nil {
		return result, err
	}
	return result, nil
}

// PermissionUpdateParams 是 `session.permission.update` 参数：必须带上一版本号。
type PermissionUpdateParams struct {
	SessionID       string           `json:"sessionId"`
	BranchID        string           `json:"branchId"`
	ExpectedVersion int64            `json:"expectedVersion"`
	Policy          PermissionPolicy `json:"policy"`
}

// PermissionChanged 是时间线事件 `permission/changed` 的数据。
// 服务端没有 `permission.changed` 通知；切换档位的唯一事实来源是这条事件。
type PermissionChanged struct {
	Policy PermissionPolicy `json:"policy"`
}

// ParsePermissionChanged 解码 `permission/changed` 事件的 data；未知字段被忽略。
func ParsePermissionChanged(raw json.RawMessage) (PermissionChanged, error) {
	var changed PermissionChanged
	if err := decodeJSON(raw, &changed, "permission/changed"); err != nil {
		return changed, err
	}
	if err := changed.Policy.Validate("permission/changed policy"); err != nil {
		return changed, err
	}
	return changed, nil
}

// SessionRenameParams 是 `session.rename` 参数：`title` 为 null 表示清空标题。
type SessionRenameParams struct {
	SessionID string  `json:"sessionId"`
	Title     *string `json:"title"`
}

// SessionRenameResult 是 `session.rename` 回执。
type SessionRenameResult struct {
	Title       *string `json:"title"`
	TitleSource *string `json:"titleSource"`
	Revision    int64   `json:"revision"`
}

// ParseSessionRenameResult 解码并校验一次改名；未知字段被忽略。
func ParseSessionRenameResult(raw json.RawMessage) (SessionRenameResult, error) {
	var result SessionRenameResult
	if err := decodeJSON(raw, &result, "session.rename"); err != nil {
		return result, err
	}
	if result.Revision < 1 {
		return result, errors.New("session.rename is missing revision")
	}
	if result.Title != nil && result.TitleSource == nil {
		return result, errors.New("session.rename is missing titleSource")
	}
	return result, nil
}

// StorageReportResult 是 `storage.report` 回执（字节数）。
type StorageReportResult struct {
	DatabaseBytes    int64 `json:"databaseBytes"`
	WALBytes         int64 `json:"walBytes"`
	ReclaimableBytes int64 `json:"reclaimableBytes"`
}

// ParseStorageReportResult 解码存储读数；未知字段被忽略。
func ParseStorageReportResult(raw json.RawMessage) (StorageReportResult, error) {
	var result StorageReportResult
	if err := decodeJSON(raw, &result, "storage.report"); err != nil {
		return result, err
	}
	if result.DatabaseBytes < 0 || result.WALBytes < 0 || result.ReclaimableBytes < 0 {
		return result, errors.New("storage.report has a negative byte count")
	}
	return result, nil
}

// ManagementProgressNotice 是任意管理方法在回执前发来的进度通知，
// 只发给发起调用的连接，按 `id` 与请求关联。
type ManagementProgressNotice struct {
	ID     ID              `json:"id"`
	Method string          `json:"method"`
	Data   json.RawMessage `json:"data"`
}

// ParseManagementProgressNotice 解码一条管理进度；未知字段被忽略。
func ParseManagementProgressNotice(raw json.RawMessage) (ManagementProgressNotice, error) {
	var notice ManagementProgressNotice
	if err := decodeJSON(raw, &notice, "management.progress"); err != nil {
		return notice, err
	}
	if notice.ID == "" || !notice.ID.Valid() {
		return notice, errors.New("management.progress is missing a valid id")
	}
	if notice.Method == "" {
		return notice, errors.New("management.progress is missing method")
	}
	return notice, nil
}
