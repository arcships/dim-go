package extensions

import (
	"context"

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

// SkillInstallPayload 是 skill.install 意图的载荷。
type SkillInstallPayload struct {
	Source InstallSource `json:"source"`
}

// SkillEnabledPayload 是 skill.set_enabled 意图的载荷。
type SkillEnabledPayload struct {
	SkillID string `json:"skillId"`
	Enabled bool   `json:"enabled"`
}

// ListSkills 读取技能清单并更新镜像读数。
func (s *Service) ListSkills(ctx context.Context) (Snapshot[Skill], error) {
	if s.cfg.Skills == nil {
		return missingReader(s, "skills.catalog", &s.skills)
	}
	return readInto(s, ctx, "skills.catalog", &s.skills, s.cfg.Skills.ReadSkills)
}

// SkillSnapshot 返回最近一次技能读数，不发起请求。
func (s *Service) SkillSnapshot() Snapshot[Skill] {
	return snapshotOf(s, &s.skills)
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
