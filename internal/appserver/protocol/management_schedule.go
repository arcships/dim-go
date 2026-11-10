package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ScheduleJob 是 `schedule.list` 里的一条任务（wire 为 snake_case）。
// 复杂子文档（schedule/model/policy/created_by）按 owner 原样保留。
type ScheduleJob struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Schedule      json.RawMessage `json:"schedule"`
	Prompt        string          `json:"prompt"`
	Cwd           string          `json:"cwd"`
	Model         json.RawMessage `json:"model,omitempty"`
	AgentID       string          `json:"agent_id,omitempty"`
	Mode          string          `json:"mode"`
	SessionID     string          `json:"session_id,omitempty"`
	Policy        string          `json:"policy,omitempty"`
	PolicyCap     json.RawMessage `json:"policy_cap"`
	Enabled       bool            `json:"enabled"`
	EnabledAt     *string         `json:"enabled_at,omitempty"`
	MaxRunMinutes int64           `json:"max_run_minutes"`
	Revision      int64           `json:"revision"`
	CreatedBy     json.RawMessage `json:"created_by"`
	// alpha.21 在 list 里额外给出下一次窗口与最近一次运行。
	NextWindow json.RawMessage `json:"next_window,omitempty"`
	LastRun    json.RawMessage `json:"last_run,omitempty"`
}

// ScheduleListResult 是 `schedule.list` 回执。
type ScheduleListResult struct {
	Jobs []ScheduleJob `json:"jobs"`
}

// ParseScheduleListResult 解码并校验计划任务清单；未知字段被忽略。
func ParseScheduleListResult(raw json.RawMessage) (ScheduleListResult, error) {
	var result ScheduleListResult
	if err := decodeJSON(raw, &result, "schedule.list"); err != nil {
		return result, err
	}
	if result.Jobs == nil {
		return result, errors.New("schedule.list is missing jobs")
	}
	for index, job := range result.Jobs {
		if err := job.validate("schedule.list"); err != nil {
			return result, fmt.Errorf("jobs[%d]: %w", index, err)
		}
	}
	return result, nil
}

// ScheduleSaveParams 是 `schedule.save` 参数；job 是部分任务文档。
type ScheduleSaveParams struct {
	Job      json.RawMessage `json:"job"`
	Revision *int64          `json:"revision,omitempty"`
}

// ScheduleTransition 是保存时对任务策略的说明；字段按 owner 原样保留。
type ScheduleTransition struct {
	From json.RawMessage `json:"from"`
	To   json.RawMessage `json:"to"`
}

// ScheduleSaveResult 是 `schedule.save` 回执。
type ScheduleSaveResult struct {
	Transition *ScheduleTransition `json:"transition,omitempty"`
	Job        ScheduleJob         `json:"job"`
}

// ParseScheduleSaveResult 解码并校验一次计划任务保存；未知字段被忽略。
func ParseScheduleSaveResult(raw json.RawMessage) (ScheduleSaveResult, error) {
	var result ScheduleSaveResult
	if err := decodeJSON(raw, &result, "schedule.save"); err != nil {
		return result, err
	}
	if err := result.Job.validate("schedule.save"); err != nil {
		return result, err
	}
	return result, nil
}

// ScheduleDeleteParams 是 `schedule.delete` 参数。
type ScheduleDeleteParams struct {
	JobID string `json:"job_id"`
}

// ScheduleDeleteResult 是 `schedule.delete` 回执。
type ScheduleDeleteResult struct {
	JobID string `json:"job_id"`
}

// ParseScheduleDeleteResult 解码并校验一次计划任务删除；未知字段被忽略。
func ParseScheduleDeleteResult(raw json.RawMessage) (ScheduleDeleteResult, error) {
	var result ScheduleDeleteResult
	if err := decodeJSON(raw, &result, "schedule.delete"); err != nil {
		return result, err
	}
	if result.JobID == "" {
		return result, errors.New("schedule.delete is missing job_id")
	}
	return result, nil
}

// ScheduleSetEnabledParams 是 `schedule.set_enabled` 参数。
type ScheduleSetEnabledParams struct {
	JobID   string `json:"job_id"`
	Enabled bool   `json:"enabled"`
}

// ScheduleSetEnabledResult 是 `schedule.set_enabled` 回执。
type ScheduleSetEnabledResult struct {
	Job ScheduleJob `json:"job"`
}

// ParseScheduleSetEnabledResult 解码并校验一次启停；未知字段被忽略。
func ParseScheduleSetEnabledResult(raw json.RawMessage) (ScheduleSetEnabledResult, error) {
	var result ScheduleSetEnabledResult
	if err := decodeJSON(raw, &result, "schedule.set_enabled"); err != nil {
		return result, err
	}
	if err := result.Job.validate("schedule.set_enabled"); err != nil {
		return result, err
	}
	return result, nil
}

// ScheduleHistoryParams 是 `schedule.history` 参数。
type ScheduleHistoryParams struct {
	JobID string `json:"job_id"`
	Limit *int   `json:"limit,omitempty"`
}

// ScheduleRun 是 `schedule.history` 里的一次运行（snake_case）。
type ScheduleRun struct {
	RunID         string          `json:"run_id"`
	JobID         string          `json:"job_id"`
	Window        string          `json:"window"`
	State         string          `json:"state"`
	DeadlineAt    *string         `json:"deadline_at,omitempty"`
	SessionID     *string         `json:"session_id,omitempty"`
	SessionOrigin *string         `json:"session_origin,omitempty"`
	Policy        json.RawMessage `json:"policy,omitempty"`
	InputEventID  *string         `json:"input_event_id,omitempty"`
	SubmissionID  string          `json:"submission_id"`
	ReportTo      json.RawMessage `json:"report_to,omitempty"`
	CreatedAt     string          `json:"created_at"`
	EndedAt       *string         `json:"ended_at,omitempty"`
}

// ScheduleHistoryResult 是 `schedule.history` 回执。
type ScheduleHistoryResult struct {
	Runs []ScheduleRun `json:"runs"`
}

// ParseScheduleHistoryResult 解码并校验运行历史；未知字段被忽略。
func ParseScheduleHistoryResult(raw json.RawMessage) (ScheduleHistoryResult, error) {
	var result ScheduleHistoryResult
	if err := decodeJSON(raw, &result, "schedule.history"); err != nil {
		return result, err
	}
	if result.Runs == nil {
		return result, errors.New("schedule.history is missing runs")
	}
	for index, run := range result.Runs {
		if run.RunID == "" || run.JobID == "" || run.State == "" {
			return result, fmt.Errorf("schedule.history runs[%d] is missing identity", index)
		}
	}
	return result, nil
}

// ScheduleRunNowParams 是 `schedule.run_now` 参数。
type ScheduleRunNowParams struct {
	JobID string `json:"job_id"`
}

// ScheduleWatchParams 是 `schedule.watch` 参数（无字段）。
type ScheduleWatchParams struct{}

// ScheduleWatchItem 是 `schedule.watch` 通过 management.progress 发来的进度项。
type ScheduleWatchItem struct {
	Seq  int64           `json:"seq"`
	Kind string          `json:"kind"`
	Raw  json.RawMessage `json:"-"`
}

// ParseScheduleWatchItem 解码一条计划任务进度；未知字段被忽略。
func ParseScheduleWatchItem(raw json.RawMessage) (ScheduleWatchItem, error) {
	var item ScheduleWatchItem
	if err := decodeJSON(raw, &item, "schedule.watch item"); err != nil {
		return item, err
	}
	if item.Kind == "" {
		return item, errors.New("schedule.watch item is missing kind")
	}
	item.Raw = raw
	return item, nil
}

func (j ScheduleJob) validate(label string) error {
	if j.ID == "" || j.Name == "" || j.Cwd == "" || j.Mode == "" {
		return fmt.Errorf("%s job is missing identity", label)
	}
	if len(j.Schedule) == 0 || len(j.PolicyCap) == 0 {
		return fmt.Errorf("%s job is missing schedule or policy_cap", label)
	}
	return nil
}
