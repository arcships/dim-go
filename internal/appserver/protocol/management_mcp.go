package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
)

// MCP 作用域。
const (
	MCPScopeUser    = "user"
	MCPScopeProject = "project"
	MCPScopePlugin  = "plugin"
)

// MCPAuthStatus 是 `mcp.catalog` 一项的鉴权状态。
type MCPAuthStatus struct {
	Status        string   `json:"status"`
	MissingScopes []string `json:"missingScopes,omitempty"`
}

// MCPCatalogEntry 是 `mcp.catalog` 里的服务器条目。
type MCPCatalogEntry struct {
	ID        string         `json:"id"`
	Source    string         `json:"source"`
	Enabled   bool           `json:"enabled"`
	Transport string         `json:"transport"`
	Auth      *MCPAuthStatus `json:"auth,omitempty"`
}

// MCPInvalidCatalogEntry 是无法解码的条目，与有效条目分开列出。
type MCPInvalidCatalogEntry struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Error  string `json:"error"`
}

// MCPCatalogResult 是 `mcp.catalog` 回执；应用状态按 owner 文档原样保留。
type MCPCatalogResult struct {
	UserRevision       string            `json:"userRevision"`
	ProjectRevision    *string           `json:"projectRevision"`
	UserApplication    json.RawMessage   `json:"userApplication"`
	PluginApplication  json.RawMessage   `json:"pluginApplication"`
	ProjectApplication json.RawMessage   `json:"projectApplication"`
	Servers            []MCPCatalogEntry `json:"servers"`
	Invalid            []MCPInvalidCatalogEntry
}

// ParseMCPCatalogResult 解码并校验 MCP 目录；未知字段被忽略。
func ParseMCPCatalogResult(raw json.RawMessage) (MCPCatalogResult, error) {
	var result MCPCatalogResult
	if err := decodeJSON(raw, &result, "mcp.catalog"); err != nil {
		return result, err
	}
	if result.UserRevision == "" {
		return result, errors.New("mcp.catalog is missing userRevision")
	}
	if result.Servers == nil || result.Invalid == nil {
		return result, errors.New("mcp.catalog is missing servers or invalid")
	}
	for index, server := range result.Servers {
		switch server.Source {
		case MCPScopeUser, MCPScopeProject, MCPScopePlugin:
		default:
			return result, fmt.Errorf("mcp.catalog servers[%d] has an invalid source %q", index, server.Source)
		}
		if server.ID == "" || server.Transport == "" {
			return result, fmt.Errorf("mcp.catalog servers[%d] is missing identity", index)
		}
	}
	return result, nil
}

// MCPImportSource 是一个可导入的客户端配置来源。
type MCPImportSource struct {
	ID      string `json:"id"`
	Client  string `json:"client"`
	Scope   string `json:"scope"`
	Path    string `json:"path"`
	State   string `json:"state"`
	Entries int    `json:"entries"`
}

// MCPImportSourcesResult 是 `mcp.import.sources` 回执。
type MCPImportSourcesResult struct {
	Sources []MCPImportSource `json:"sources"`
}

// ParseMCPImportSourcesResult 解码并校验导入来源；未知字段被忽略。
func ParseMCPImportSourcesResult(raw json.RawMessage) (MCPImportSourcesResult, error) {
	var result MCPImportSourcesResult
	if err := decodeJSON(raw, &result, "mcp.import.sources"); err != nil {
		return result, err
	}
	if result.Sources == nil {
		return result, errors.New("mcp.import.sources is missing sources")
	}
	for index, source := range result.Sources {
		if source.ID == "" || source.Client == "" {
			return result, fmt.Errorf("mcp.import.sources[%d] is missing identity", index)
		}
	}
	return result, nil
}

// MCPConfigurationResult 是 `mcp.configuration` 回执；配置文档按原样保留。
type MCPConfigurationResult struct {
	Revision           *string         `json:"revision"`
	Enabled            bool            `json:"enabled"`
	SecretsInUserScope bool            `json:"secretsInUserScope"`
	Configuration      json.RawMessage `json:"configuration"`
}

// ParseMCPConfigurationResult 解码并校验一条 MCP 配置；未知字段被忽略。
func ParseMCPConfigurationResult(raw json.RawMessage) (MCPConfigurationResult, error) {
	var result MCPConfigurationResult
	if err := decodeJSON(raw, &result, "mcp.configuration"); err != nil {
		return result, err
	}
	if len(result.Configuration) == 0 {
		return result, errors.New("mcp.configuration is missing configuration")
	}
	return result, nil
}

// MCPTargetParams 是只按 id+source 定位一个 MCP 条目的参数。
type MCPTargetParams struct {
	Cwd    string `json:"cwd,omitempty"`
	ID     string `json:"id"`
	Source string `json:"source"`
}

// MCPSetEnabledParams 是 `mcp.set_enabled` 参数。
type MCPSetEnabledParams struct {
	Cwd             string `json:"cwd,omitempty"`
	ID              string `json:"id"`
	Source          string `json:"source"`
	ExpectedVersion string `json:"expectedVersion"`
	Enabled         bool   `json:"enabled"`
}

// MCPRemoveParams 是 `mcp.remove` 参数。
type MCPRemoveParams struct {
	Cwd             string `json:"cwd,omitempty"`
	ID              string `json:"id"`
	Source          string `json:"source"`
	ExpectedVersion string `json:"expectedVersion"`
}

// MCPMoveParams 是 `mcp.move` 参数。
type MCPMoveParams struct {
	Cwd                 string `json:"cwd,omitempty"`
	ID                  string `json:"id"`
	From                string `json:"from"`
	To                  string `json:"to"`
	ExpectedFromVersion string `json:"expectedFromVersion"`
	ExpectedToVersion   string `json:"expectedToVersion"`
}

// MCPSaveParams 是 `mcp.save` 参数；配置文档按原样发送。
type MCPSaveParams struct {
	Cwd             string          `json:"cwd,omitempty"`
	ID              string          `json:"id"`
	Source          string          `json:"source"`
	ExpectedVersion string          `json:"expectedVersion"`
	Configuration   json.RawMessage `json:"configuration"`
}

// MCPImportPreviewParams 是 `mcp.import.preview` 参数。
type MCPImportPreviewParams struct {
	Cwd    string `json:"cwd,omitempty"`
	Source string `json:"source"`
}

// MCPImportApplyParams 是 `mcp.import.apply` 参数。
type MCPImportApplyParams struct {
	Cwd             string   `json:"cwd,omitempty"`
	Source          string   `json:"source"`
	IDs             []string `json:"ids"`
	ExpectedVersion string   `json:"expectedVersion,omitempty"`
}
