package extensions

import (
	"context"
	"errors"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// Skill 是一个技能条目。来源用于区分内置、随包、用户安装与插件提供。
type Skill struct {
	ID          string
	Name        string
	Description string
	Version     string
	Source      Source
	Installed   bool
	Enabled     bool
}

// SkillReader 读取技能清单。只读，不安装、不启停。
type SkillReader interface {
	ReadSkills(ctx context.Context) ([]Skill, error)
}

// SkillIssue 是 skills.catalog 里的一条目录问题。
type SkillIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// SkillCatalog 是一次技能目录读取；Version 用于观察刷新后的版本变化。
type SkillCatalog struct {
	Version uint64
	Skills  []Skill
	Issues  []SkillIssue
}

// SkillCatalogReader 读取带版本与问题的技能目录。集成层实现它时，本包会保留
// 目录版本；只实现 SkillReader 时版本保持为 0。
type SkillCatalogReader interface {
	ReadSkillCatalog(ctx context.Context) (SkillCatalog, error)
}

// SkillInstallPayload 是 skill.install 意图的载荷。
type SkillInstallPayload struct {
	Source InstallSource `json:"source"`
}

// SkillEnabledPayload 是 skill.set_enabled 意图的载荷。
type SkillEnabledPayload struct {
	SkillID string `json:"skillId"`
	Enabled bool   `json:"enabled"`
}

// ListSkills 读取技能清单并更新镜像读数。读取方实现 SkillCatalogReader 时同时
// 更新目录版本与问题清单。
func (s *Service) ListSkills(ctx context.Context) (Snapshot[Skill], error) {
	if s.cfg.Skills == nil {
		return missingReader(s, "skills.catalog", &s.skills)
	}
	if reader, ok := s.cfg.Skills.(SkillCatalogReader); ok {
		return s.readSkillCatalog(ctx, reader)
	}
	return readInto(s, ctx, "skills.catalog", &s.skills, s.cfg.Skills.ReadSkills)
}

func (s *Service) readSkillCatalog(ctx context.Context, reader SkillCatalogReader) (Snapshot[Skill], error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.readMu.Lock()
	defer s.readMu.Unlock()
	if s.isClosed() {
		return Snapshot[Skill]{}, ErrClosed
	}
	catalog, err := reader.ReadSkillCatalog(ctx)
	now := s.cfg.Now()
	if err != nil {
		wrapped := s.wrapReadError("skills.catalog", err)
		s.mu.Lock()
		s.skills.ReadError = wrapped.Error()
		s.skills.CapabilityMissing = errors.Is(wrapped, ErrCapabilityMissing)
		snapshot := cloneSnapshot(s.skills)
		s.mu.Unlock()
		return snapshot, wrapped
	}
	s.mu.Lock()
	s.skills.Items = append([]Skill(nil), catalog.Skills...)
	s.skills.ReadAt = now
	s.skills.ReadError = ""
	s.skills.CapabilityMissing = false
	s.skillVersion = catalog.Version
	s.skillIssues = append([]SkillIssue(nil), catalog.Issues...)
	snapshot := cloneSnapshot(s.skills)
	s.mu.Unlock()
	return snapshot, nil
}

// SkillCatalogVersion 返回最近一次技能目录读取的版本，不发起请求。
func (s *Service) SkillCatalogVersion() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.skillVersion
}

// SkillIssues 返回最近一次技能目录读取的问题副本。
func (s *Service) SkillIssues() []SkillIssue {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]SkillIssue(nil), s.skillIssues...)
}

// SkillSnapshot 返回最近一次技能读数，不发起请求。
func (s *Service) SkillSnapshot() Snapshot[Skill] {
	return snapshotOf(s, &s.skills)
}

// RefreshSkills 强制刷新服务端技能目录。写意图只发一次；成功后调用 ListSkills
// 即可观察到条目与版本变化。
func (s *Service) RefreshSkills(ctx context.Context) (write.IntentView, error) {
	if err := s.requireFeature("skills.refresh"); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpSkillsRefresh, struct{}{}, true)
}

// ReloadSkillCatalog 按真实方法名 skills.reload_catalog 触发目录重载。alpha.21
// 的 hello 未声明该能力，入口直接返回能力缺失且不发请求。
func (s *Service) ReloadSkillCatalog(ctx context.Context) (write.IntentView, error) {
	if err := s.requireFeature("skills.reload_catalog"); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpSkillsReloadCatalog, struct{}{}, true)
}

// InstallSkill 从给定来源安装技能。写入经写登记，只发一次。
func (s *Service) InstallSkill(ctx context.Context, source InstallSource) (write.IntentView, error) {
	if err := source.validate(); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpSkillInstall, SkillInstallPayload{Source: source}, false)
}

// SetSkillEnabled 写入技能启用状态。状态由服务端保存，本端不留独立记录；重启后
// 重新读取即可得到同一结果。
func (s *Service) SetSkillEnabled(ctx context.Context, skillID string, enabled bool) (write.IntentView, error) {
	if err := requireText("技能标识", skillID); err != nil {
		return write.IntentView{}, err
	}
	return s.write(ctx, OpSkillSetEnabled, SkillEnabledPayload{SkillID: skillID, Enabled: enabled}, false)
}
