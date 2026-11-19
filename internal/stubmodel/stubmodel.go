// Package stubmodel 是真实服务端证据用例的模型替身：一个 OpenAI 兼容的
// 流式端点，按请求里的工具清单决定下一步。
//
// 为什么需要它：App Server 的 `--mock-model` 档不带产品级装配（没有权限档位读写、
// 没有工作区工具），真正需要审批与工具执行的回路只能跑在 `--model-config` 档上；
// 这一档要求一个真实模型端点。替身让「服务端 + 会话 + 工具 + 审批」都是真的，
// 只有语言模型是假的，也不依赖任何外部凭据或本机 home。
//
// 它只用于测试与证据，不进产品路径。
package stubmodel

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

// ToolCommand 是替身发起的那一次 exec 调用要跑的命令。
const ToolCommand = "echo e2e-ok"

// Answer 是替身在工具结果之后回的文本。
const Answer = "工具已经执行，输出是 e2e-ok。"

// toolArguments 是 exec 的入参：命令与标题都是必填。
const toolArguments = `{"command":"` + ToolCommand + `","title":"e2e echo"}`

// Model 是模型替身。Requests 统计收到的补全请求数，ToolCalls 统计发起工具
// 调用的次数；两者只用于证据里的断言与日志。
type Model struct {
	Requests  atomic.Int64
	ToolCalls atomic.Int64
}

// New 返回一个替身；把它挂到 httptest.Server 上即可。
func New() *Model { return &Model{} }

// ServeHTTP 回答两种请求：`GET .../models` 给模型清单，`POST .../chat/completions`
// 给流式补全。补全请求带工具清单且还没有工具结果时发起一次 exec 调用；
// 其余情况（已有工具结果、标题/摘要这类不带工具的请求）回一段纯文本。
func (m *Model) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/models") {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"e2e-stub","object":"model"}]}`)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	m.Requests.Add(1)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, _ := w.(http.Flusher)

	payload := string(body)
	hasTools := strings.Contains(payload, `"tools"`)
	hasToolResult := strings.Contains(payload, `"role":"tool"`)
	if hasTools && !hasToolResult {
		m.ToolCalls.Add(1)
		m.chunk(w, flusher, map[string]any{
			"choices": []any{map[string]any{
				"index": 0,
				"delta": map[string]any{
					"role": "assistant",
					"tool_calls": []any{map[string]any{
						"index":    0,
						"id":       "call_e2e_exec",
						"type":     "function",
						"function": map[string]any{"name": "exec", "arguments": toolArguments},
					}},
				},
				"finish_reason": nil,
			}},
		})
		m.chunk(w, flusher, finish("tool_calls"))
	} else {
		m.chunk(w, flusher, map[string]any{
			"choices": []any{map[string]any{
				"index":         0,
				"delta":         map[string]any{"role": "assistant", "content": Answer},
				"finish_reason": nil,
			}},
		})
		m.chunk(w, flusher, finish("stop"))
	}
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}

func (m *Model) chunk(w io.Writer, flusher http.Flusher, payload map[string]any) {
	payload["id"] = "chatcmpl-e2e"
	payload["object"] = "chat.completion.chunk"
	payload["created"] = 1
	payload["model"] = "e2e-stub"
	raw, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", raw)
	if flusher != nil {
		flusher.Flush()
	}
}

func finish(reason string) map[string]any {
	return map[string]any{
		"choices": []any{map[string]any{
			"index":         0,
			"delta":         map[string]any{},
			"finish_reason": reason,
		}},
	}
}

// WriteConfig 写 App Server 的开发档模型配置（`--model-config`）：一个
// OpenAI 兼容 provider 指向替身地址。文件里没有凭据。
func WriteConfig(path, baseURL string) error {
	config := map[string]any{
		"defaultModel": map[string]any{"providerId": "local", "modelId": "e2e-stub"},
		"providers": []any{map[string]any{
			"id":      "local",
			"kind":    "openai-compatible",
			"baseUrl": baseURL,
			"models": []any{map[string]any{
				"id":              "e2e-stub",
				"contextWindow":   128000,
				"maxOutputTokens": 4096,
				"config":          map[string]any{"maxTokens": 4096, "temperature": 0},
			}},
		}},
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}
