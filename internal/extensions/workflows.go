package extensions

import (
	"context"
	"time"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// WorkflowStatus 是一次工作流运行的状态。
type WorkflowStatus string

const (
	WorkflowQueued    WorkflowStatus = "queued"
	WorkflowRunning   WorkflowStatus = "running"
	WorkflowSucceeded WorkflowStatus = "succeeded"
	WorkflowFailed    WorkflowStatus = "failed"
	WorkflowStopped   WorkflowStatus = "stopped"
	WorkflowUnknown   WorkflowStatus = "unknown"
)

// WorkflowStep 是定义里的一个步骤摘要。
type WorkflowStep struct {
	ID   string
	Name string
	Kind string
}

// WorkflowDefinition 是一份工作流定义。内容由服务端给出，本端只展示与启动。
type WorkflowDefinition struct {
	ID          string
	Name        string
	Description string
	Version     string
	Source      Source
	Steps       []WorkflowStep
}

// WorkflowRun 是一次工作流运行的读数。
type WorkflowRun struct {
	ID         string
	WorkflowID string
	Status     WorkflowStatus
	StartedAt  time.Time
	EndedAt    time.Time
	Error      string
}

// WorkflowReader 读取工作流定义与运行。只读。
type WorkflowReader interface {
	ReadWorkflows(ctx context.Context) ([]WorkflowDefinition, error)
	ReadWorkflow(ctx context.Context, workflowID string) (WorkflowDefinition, error)
	ReadWorkflowRuns(ctx context.Context, workflowID string) ([]WorkflowRun, error)
}

// WorkflowStartPayload 是 workflow.start 意图的载荷。
type WorkflowStartPayload struct {
	WorkflowID string            `json:"workflowId"`
	Inputs     map[string]string `json:"inputs,omitempty"`
}

// WorkflowStopPayload 是 workflow.stop 意图的载荷。
type WorkflowStopPayload struct {
	RunID string `json:"runId"`
}

// ListWorkflows 读取工作流定义清单并更新镜像读数。
func (s *Service) ListWorkflows(ctx context.Context) (Snapshot[WorkflowDefinition], error) {
	if s.cfg.Workflows == nil {
		return missingReader(s, "workflow.list", &s.workflows)
	}
	return readInto(s, ctx, "workflow.list", &s.workflows, s.cfg.Workflows.ReadWorkflows)
}

// WorkflowSnapshot 返回最近一次工作流定义读数，不发起请求。
func (s *Service) WorkflowSnapshot() Snapshot[WorkflowDefinition] {
	return snapshotOf(s, &s.workflows)
}

// ReadWorkflow 读取一份工作流定义。它不进入清单镜像，只作为浏览结果。
func (s *Service) ReadWorkflow(ctx context.Context, workflowID string) (WorkflowDefinition, error) {
	if err := requireText("工作流标识", workflowID); err != nil {
		return WorkflowDefinition{}, err
	}
	if s.cfg.Workflows == nil {
		return WorkflowDefinition{}, s.wrapReadError("workflow.get", ErrCapabilityMissing)
	}
	s.readMu.Lock()
	defer s.readMu.Unlock()
	if s.isClosed() {
		return WorkflowDefinition{}, ErrClosed
	}
	definition, err := s.cfg.Workflows.ReadWorkflow(ctx, workflowID)
	if err != nil {
		return WorkflowDefinition{}, s.wrapReadError("workflow.get", err)
	}
	definition.Steps = append([]WorkflowStep(nil), definition.Steps...)
	return definition, nil
}

// ListWorkflowRuns 读取某个工作流的运行并更新镜像读数。之后的变更通知只为
// 已经读过运行的工作流重读。
func (s *Service) ListWorkflowRuns(ctx context.Context, workflowID string) (Snapshot[WorkflowRun], error) {
	if err := requireText("工作流标识", workflowID); err != nil {
		return Snapshot[WorkflowRun]{}, err
	}
	s.watchWorkflow(workflowID)
	if s.cfg.Workflows == nil {
		return missingReader(s, "workflow.runs", s.workflowRunTarget(workflowID))
	}
	return s.readWorkflowRuns(ctx, workflowID)
}

func (s *Service) readWorkflowRuns(ctx context.Context, workflowID string) (Snapshot[WorkflowRun], error) {
	if s.cfg.Workflows == nil {
		return missingReader(s, "workflow.runs", s.workflowRunTarget(workflowID))
	}
	s.mu.Lock()
	target, ok := s.workflowRuns[workflowID]
	if !ok {
		target = &Snapshot[WorkflowRun]{}
		s.workflowRuns[workflowID] = target
	}
	s.mu.Unlock()
	return readInto(s, ctx, "workflow.runs", target, func(readCtx context.Context) ([]WorkflowRun, error) {
		return s.cfg.Workflows.ReadWorkflowRuns(readCtx, workflowID)
	})
}

// WorkflowRunSnapshot 返回某个工作流最近一次运行读数，不发起请求。
func (s *Service) WorkflowRunSnapshot(workflowID string) Snapshot[WorkflowRun] {
	s.mu.Lock()
	target, ok := s.workflowRuns[workflowID]
	s.mu.Unlock()
	if !ok {
		return Snapshot[WorkflowRun]{}
	}
	return snapshotOf(s, target)
}

func (s *Service) workflowRunTarget(workflowID string) *Snapshot[WorkflowRun] {
	s.mu.Lock()
	defer s.mu.Unlock()
	target, ok := s.workflowRuns[workflowID]
	if !ok {
		target = &Snapshot[WorkflowRun]{}
		s.workflowRuns[workflowID] = target
	}
	return target
}

func (s *Service) watchWorkflow(workflowID string) {
	s.mu.Lock()
	s.workflowWatch[workflowID] = struct{}{}
	s.mu.Unlock()
}

// StartWorkflow 启动一次工作流运行。每次显式调用都登记一条新的写意图；上一次
// 尚未结项时只等待同一条意图，不重复启动。
func (s *Service) StartWorkflow(ctx context.Context, workflowID string, inputs map[string]string) (write.IntentView, error) {
	if err := requireText("工作流标识", workflowID); err != nil {
		return write.IntentView{}, err
	}
	// alpha.21 没有注册 workflow.*；没有读取方即能力缺失，写入口也不发请求。
	if s.cfg.Workflows == nil {
		return write.IntentView{}, s.wrapReadError("workflow.start", ErrCapabilityMissing)
	}
	copied := make(map[string]string, len(inputs))
	for key, value := range inputs {
		copied[key] = value
	}
	s.watchWorkflow(workflowID)
	return s.write(ctx, OpWorkflowStart, WorkflowStartPayload{WorkflowID: workflowID, Inputs: copied}, true)
}

// StopWorkflow 停止一次工作流运行。重复停止同一次运行只保留一条意图。
func (s *Service) StopWorkflow(ctx context.Context, runID string) (write.IntentView, error) {
	if err := requireText("工作流运行标识", runID); err != nil {
		return write.IntentView{}, err
	}
	// 与 StartWorkflow 同理：服务端没有该能力时零请求。
	if s.cfg.Workflows == nil {
		return write.IntentView{}, s.wrapReadError("workflow.stop", ErrCapabilityMissing)
	}
	return s.write(ctx, OpWorkflowStop, WorkflowStopPayload{RunID: runID}, false)
}
