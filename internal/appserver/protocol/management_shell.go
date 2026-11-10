package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// 本机 shell 运行方式。
const (
	ShellModeRecord  = "record"
	ShellModePrivate = "private"
)

// ShellRunParams 是 `shell.run` 参数。
type ShellRunParams struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	RunID     string `json:"runId"`
	Command   string `json:"command"`
	Mode      string `json:"mode"`
}

// ShellRunRecorded 是 record 模式下结果写入时间线的回执：成功带 sequence，
// 失败带 error。
type ShellRunRecorded struct {
	Sequence int64           `json:"sequence,omitempty"`
	Error    json.RawMessage `json:"error,omitempty"`
}

// ShellRunResult 是 `shell.run` 回执。
type ShellRunResult struct {
	RunID      string            `json:"runId"`
	Status     string            `json:"status"`
	Output     string            `json:"output"`
	Truncated  bool              `json:"truncated"`
	DurationMS int64             `json:"durationMs"`
	ExitCode   *int              `json:"exitCode,omitempty"`
	Signal     *string           `json:"signal,omitempty"`
	Reason     *string           `json:"reason,omitempty"`
	Recorded   *ShellRunRecorded `json:"recorded,omitempty"`
}

// ParseShellRunResult 解码并校验一次 shell 运行；未知字段被忽略。
func ParseShellRunResult(raw json.RawMessage) (ShellRunResult, error) {
	var result ShellRunResult
	if err := decodeJSON(raw, &result, "shell.run"); err != nil {
		return result, err
	}
	if result.RunID == "" {
		return result, errors.New("shell.run is missing runId")
	}
	switch result.Status {
	case "completed", "failed", "stopped":
	default:
		return result, fmt.Errorf("shell.run has an invalid status %q", result.Status)
	}
	return result, nil
}

// ShellRunInfo 是 `shell.list` 里的一次运行。
type ShellRunInfo struct {
	RunID     string `json:"runId"`
	BranchID  string `json:"branchId"`
	Mode      string `json:"mode"`
	Command   string `json:"command"`
	StartedAt int64  `json:"startedAt"`
}

// ShellListParams 是 `shell.list` 参数。
type ShellListParams struct {
	SessionID string `json:"sessionId"`
}

// ShellListResult 是 `shell.list` 回执。
type ShellListResult struct {
	Runs []ShellRunInfo `json:"runs"`
}

// ParseShellListResult 解码并校验运行清单；未知字段被忽略。
func ParseShellListResult(raw json.RawMessage) (ShellListResult, error) {
	var result ShellListResult
	if err := decodeJSON(raw, &result, "shell.list"); err != nil {
		return result, err
	}
	if result.Runs == nil {
		return result, errors.New("shell.list is missing runs")
	}
	for index, run := range result.Runs {
		if run.RunID == "" || run.Command == "" {
			return result, fmt.Errorf("shell.list runs[%d] is missing identity", index)
		}
	}
	return result, nil
}

// ShellStopParams 是 `shell.stop` 参数。
type ShellStopParams struct {
	SessionID string `json:"sessionId"`
	RunID     string `json:"runId"`
}

// ShellStopResult 是 `shell.stop` 回执。
type ShellStopResult struct {
	Stopped bool `json:"stopped"`
}

// SessionWakeupCreateParams 是 `session.wakeup.create` 参数；`at` 为 RFC 3339。
type SessionWakeupCreateParams struct {
	SessionID string `json:"sessionId"`
	BranchID  string `json:"branchId"`
	At        string `json:"at"`
	Prompt    string `json:"prompt"`
}

// SessionWakeupCreateResult 是 `session.wakeup.create` 回执。
type SessionWakeupCreateResult struct {
	WakeupID string `json:"wakeupId"`
}

// ParseSessionWakeupCreateResult 解码并校验一次唤醒创建；未知字段被忽略。
func ParseSessionWakeupCreateResult(raw json.RawMessage) (SessionWakeupCreateResult, error) {
	var result SessionWakeupCreateResult
	if err := decodeJSON(raw, &result, "session.wakeup.create"); err != nil {
		return result, err
	}
	if result.WakeupID == "" {
		return result, errors.New("session.wakeup.create is missing wakeupId")
	}
	return result, nil
}

// SessionWakeupListParams 是 `session.wakeup.list` 参数；省略 branchId 时为主线。
type SessionWakeupListParams struct {
	SessionID string  `json:"sessionId"`
	BranchID  *string `json:"branchId,omitempty"`
}

// SessionWakeup 是 `session.wakeup.list` 里的一条记录（camelCase）。
type SessionWakeup struct {
	WakeupID string `json:"wakeupId"`
	At       string `json:"at"`
	Prompt   string `json:"prompt"`
	State    string `json:"state"`
}

// SessionWakeupListResult 是 `session.wakeup.list` 回执。
type SessionWakeupListResult struct {
	Wakeups []SessionWakeup `json:"wakeups"`
}

// ParseSessionWakeupListResult 解码并校验唤醒清单；未知字段被忽略。
func ParseSessionWakeupListResult(raw json.RawMessage) (SessionWakeupListResult, error) {
	var result SessionWakeupListResult
	if err := decodeJSON(raw, &result, "session.wakeup.list"); err != nil {
		return result, err
	}
	if result.Wakeups == nil {
		return result, errors.New("session.wakeup.list is missing wakeups")
	}
	for index, wakeup := range result.Wakeups {
		switch wakeup.State {
		case "scheduled", "fired", "cancelled", "expired":
		default:
			return result, fmt.Errorf("session.wakeup.list wakeups[%d] has an invalid state %q", index, wakeup.State)
		}
		if wakeup.WakeupID == "" || wakeup.At == "" {
			return result, fmt.Errorf("session.wakeup.list wakeups[%d] is missing identity", index)
		}
	}
	return result, nil
}

// SessionWakeupCancelParams 是 `session.wakeup.cancel` 参数。
type SessionWakeupCancelParams struct {
	SessionID string `json:"sessionId"`
	WakeupID  string `json:"wakeupId"`
}

// SessionWakeupCancelResult 是 `session.wakeup.cancel` 回执：
// 本次取消成功时为空对象；已经结算的记录返回当前状态。
type SessionWakeupCancelResult struct {
	State string `json:"state,omitempty"`
}

// WakeupNoticeData 是 `session.notice` 的 Scheduler/wakeup 数据。
type WakeupNoticeData struct {
	BranchID string `json:"branchId"`
	WakeupID string `json:"wakeupId"`
	State    string `json:"state"`
}

// IsWakeup 判断这条通知是否是唤醒记录变化。
func (n SessionNotice) IsWakeup() bool {
	return n.Source == "Scheduler" && n.Kind == "wakeup"
}

// ParseWakeup 解码唤醒通知数据；不是唤醒通知或字段不合法时返回错误。
func (n SessionNotice) ParseWakeup() (WakeupNoticeData, error) {
	if !n.IsWakeup() {
		return WakeupNoticeData{}, fmt.Errorf("session.notice is %s/%s, not a wakeup change", n.Source, n.Kind)
	}
	var data WakeupNoticeData
	if err := decodeJSON(n.Data, &data, "session.notice wakeup"); err != nil {
		return data, err
	}
	if data.WakeupID == "" || data.BranchID == "" {
		return data, errors.New("wakeup notice is missing identity")
	}
	switch data.State {
	case "scheduled", "fired", "cancelled", "expired":
	default:
		return data, fmt.Errorf("wakeup notice has an invalid state %q", data.State)
	}
	return data, nil
}
