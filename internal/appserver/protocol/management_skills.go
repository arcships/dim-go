package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// PluginSource 是 `plugin.install` 的来源：本地目录或 Git。
type PluginSource struct {
	Path string `json:"path,omitempty"`
	Git  string `json:"git,omitempty"`
	Ref  string `json:"ref,omitempty"`
}

// PluginInstallParams 是 `plugin.install` 参数。
type PluginInstallParams struct {
	Source PluginSource `json:"source"`
	ID     string       `json:"id,omitempty"`
	Enable *bool        `json:"enable,omitempty"`
}

// PluginNameParams 是 `plugin.enable` / `disable` / `reload` 参数。
type PluginNameParams struct {
	Name string  `json:"name"`
	Use  *string `json:"use,omitempty"`
}

// PluginIDParams 是 `plugin.remove` 参数。
type PluginIDParams struct {
	ID  string  `json:"id"`
	Ref *string `json:"ref,omitempty"`
}

// PluginCandidate 是插件名下一个可用来源。
type PluginCandidate struct {
	ID     string          `json:"id"`
	Kinds  []string        `json:"kinds"`
	Source json.RawMessage `json:"source,omitempty"`
}

// PluginEntry 是 `plugin.list` 的一项。
type PluginEntry struct {
	Name       string            `json:"name"`
	Use        *string           `json:"use"`
	Off        bool              `json:"off"`
	Locked     bool              `json:"locked"`
	Candidates []PluginCandidate `json:"candidates"`
	Status     string            `json:"status,omitempty"`
	Error      string            `json:"error,omitempty"`
	Refused    []json.RawMessage `json:"refused,omitempty"`
}

// PluginListResult 是 `plugin.list` 回执（数组）。
type PluginListResult []PluginEntry

// ParsePluginListResult 解码并校验插件清单；未知字段被忽略。
func ParsePluginListResult(raw json.RawMessage) (PluginListResult, error) {
	var result PluginListResult
	if err := decodeJSON(raw, &result, "plugin.list"); err != nil {
		return result, err
	}
	if result == nil {
		return result, errors.New("plugin.list must be an array")
	}
	for index, entry := range result {
		if entry.Name == "" {
			return result, fmt.Errorf("plugin.list[%d] is missing name", index)
		}
		if entry.Candidates == nil {
			return result, fmt.Errorf("plugin.list[%d] is missing candidates", index)
		}
	}
	return result, nil
}

// PluginOperationResult 是 enable/disable/reload/update/remove 的回执。
// disable/remove 返回空对象；enable/reload 可能带 status。
type PluginOperationResult struct {
	Status string          `json:"status,omitempty"`
	Error  string          `json:"error,omitempty"`
	Raw    json.RawMessage `json:"-"`
}

// ParsePluginOperationResult 解码一次插件变更回执；空对象是合法结果。
func ParsePluginOperationResult(raw json.RawMessage) (PluginOperationResult, error) {
	var result PluginOperationResult
	if err := decodeJSON(raw, &result, "plugin.operation"); err != nil {
		return result, err
	}
	result.Raw = raw
	return result, nil
}

// SkillIssue 是一条 skill 目录问题。
type SkillIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// SkillEntry 是 `skills.catalog` 的一项。
type SkillEntry struct {
	ID           string   `json:"id"`
	Source       string   `json:"source"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	FilePath     string   `json:"filePath"`
	AllowedTools []string `json:"allowedTools"`
	Enabled      bool     `json:"enabled"`
	ShadowedBy   *string  `json:"shadowedBy"`
}

// SkillsCatalogResult 是 `skills.catalog` 回执。
type SkillsCatalogResult struct {
	Version uint64       `json:"version"`
	Entries []SkillEntry `json:"entries"`
	Issues  []SkillIssue `json:"issues"`
}

// ParseSkillsCatalogResult 解码并校验技能目录；未知字段被忽略。
func ParseSkillsCatalogResult(raw json.RawMessage) (SkillsCatalogResult, error) {
	var result SkillsCatalogResult
	if err := decodeJSON(raw, &result, "skills.catalog"); err != nil {
		return result, err
	}
	if result.Entries == nil || result.Issues == nil {
		return result, errors.New("skills.catalog is missing entries or issues")
	}
	for index, entry := range result.Entries {
		if entry.ID == "" || entry.Name == "" || entry.FilePath == "" {
			return result, fmt.Errorf("skills.catalog entries[%d] is missing identity", index)
		}
	}
	return result, nil
}

// SkillsRefreshResult 是 `skills.refresh` 回执。
type SkillsRefreshResult struct {
	Version uint64       `json:"version"`
	Issues  []SkillIssue `json:"issues"`
}

// ParseSkillsRefreshResult 解码并校验技能刷新；未知字段被忽略。
func ParseSkillsRefreshResult(raw json.RawMessage) (SkillsRefreshResult, error) {
	var result SkillsRefreshResult
	if err := decodeJSON(raw, &result, "skills.refresh"); err != nil {
		return result, err
	}
	if result.Issues == nil {
		return result, errors.New("skills.refresh is missing issues")
	}
	return result, nil
}

// SkillInstallationSource 是一次安装的来源（路径或 Git）。
type SkillInstallationSource struct {
	Path string `json:"path,omitempty"`
	Git  string `json:"git,omitempty"`
	Ref  string `json:"ref,omitempty"`
}

// SkillInstallationSkill 是安装记录里的一个 skill。
type SkillInstallationSkill struct {
	Path string `json:"path"`
	ID   string `json:"id"`
}

// SkillInstallation 是一条安装记录。
type SkillInstallation struct {
	ID       string                   `json:"id"`
	Source   SkillInstallationSource  `json:"source"`
	Revision string                   `json:"revision"`
	Skills   []SkillInstallationSkill `json:"skills"`
}

// SkillsInstallationsResult 是 `skills.installations` 回执。
type SkillsInstallationsResult struct {
	Installations []SkillInstallation `json:"installations"`
}

// ParseSkillsInstallationsResult 解码并校验安装记录；未知字段被忽略。
func ParseSkillsInstallationsResult(raw json.RawMessage) (SkillsInstallationsResult, error) {
	var result SkillsInstallationsResult
	if err := decodeJSON(raw, &result, "skills.installations"); err != nil {
		return result, err
	}
	if result.Installations == nil {
		return result, errors.New("skills.installations is missing installations")
	}
	for index, installation := range result.Installations {
		if installation.ID == "" || installation.Revision == "" {
			return result, fmt.Errorf("skills.installations[%d] is missing identity", index)
		}
	}
	return result, nil
}

// SkillsSetEnabledParams 是 `skills.set_enabled` 参数。
type SkillsSetEnabledParams struct {
	Cwd     string `json:"cwd,omitempty"`
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

// SkillsPreviewParams 是 `skills.preview` 参数。
type SkillsPreviewParams struct {
	Source PluginSource `json:"source"`
	ID     string       `json:"id,omitempty"`
	Cwd    string       `json:"cwd,omitempty"`
}

// SelectedSkill 是安装时勾选的一项。
type SelectedSkill struct {
	Path     string `json:"path"`
	Revision string `json:"revision"`
}

// SkillsInstallParams 是 `skills.install` 参数。
type SkillsInstallParams struct {
	Source    PluginSource    `json:"source"`
	ID        string          `json:"id,omitempty"`
	Skills    []SelectedSkill `json:"skills"`
	Overwrite bool            `json:"overwrite,omitempty"`
}

// SkillsInstallationParams 是 `skills.remove` / `skills.update` 参数。
type SkillsInstallationParams struct {
	ID string `json:"id"`
}

// HookSource 是 `hooks.catalog` 里的一个来源。
type HookSource struct {
	Source  string  `json:"source"`
	Path    *string `json:"path,omitempty"`
	Enabled bool    `json:"enabled"`
	State   string  `json:"state"`
	Message *string `json:"message,omitempty"`
}

// HooksCatalogResult 是 `hooks.catalog` 回执。用户文档按原样保留给编辑者。
type HooksCatalogResult struct {
	UserRevision     string            `json:"userRevision"`
	UserDocument     json.RawMessage   `json:"userDocument"`
	ProjectsRevision string            `json:"projectsRevision"`
	DisableAllHooks  bool              `json:"disableAllHooks"`
	Sources          []HookSource      `json:"sources"`
	Hooks            []json.RawMessage `json:"hooks"`
}

// ParseHooksCatalogResult 解码并校验钩子目录；未知字段被忽略。
func ParseHooksCatalogResult(raw json.RawMessage) (HooksCatalogResult, error) {
	var result HooksCatalogResult
	if err := decodeJSON(raw, &result, "hooks.catalog"); err != nil {
		return result, err
	}
	if result.UserRevision == "" || result.ProjectsRevision == "" {
		return result, errors.New("hooks.catalog is missing a revision")
	}
	if len(result.UserDocument) == 0 {
		return result, errors.New("hooks.catalog is missing userDocument")
	}
	if result.Sources == nil || result.Hooks == nil {
		return result, errors.New("hooks.catalog is missing sources or hooks")
	}
	return result, nil
}

// HooksSaveParams 是 `hooks.save` 参数。
type HooksSaveParams struct {
	ExpectedRevision string          `json:"expectedRevision"`
	Document         json.RawMessage `json:"document"`
}

// HooksSaveResult 是 `hooks.save` 回执。
type HooksSaveResult struct {
	Revision string `json:"revision"`
}

// ParseHooksSaveResult 解码并校验一次钩子写入；未知字段被忽略。
func ParseHooksSaveResult(raw json.RawMessage) (HooksSaveResult, error) {
	var result HooksSaveResult
	if err := decodeJSON(raw, &result, "hooks.save"); err != nil {
		return result, err
	}
	if result.Revision == "" {
		return result, errors.New("hooks.save is missing revision")
	}
	return result, nil
}

// HooksSetProjectEnabledParams 是 `hooks.set_enabled` 参数。
type HooksSetProjectEnabledParams struct {
	Cwd              string `json:"cwd"`
	Enabled          bool   `json:"enabled"`
	ExpectedRevision string `json:"expectedRevision"`
}

// HooksSetProjectEnabledResult 是 `hooks.set_enabled` 回执。
type HooksSetProjectEnabledResult struct {
	Revision string `json:"revision"`
}
