// opencode.go 把控制面和自建模型写进工单目录的 opencode.json。
// OpenCode 对 session/new 交来的 MCP 服务器一律打开 Code Mode，报告工具就不会出现在模型的工具列表里。
// 配置里的 codemode: false 会把工具直接列出来。token 放进程环境变量，不写进文件。
// 自建端点同样要写成一个 provider：只把模型名交给 session/set_model 时，OpenCode 的目录里没有这个名字。

package acpworker

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// openCodeProviderID 是写进 OpenCode 的自建提供商编号。模型引用是 yoyo-llm/<模型名>。
const openCodeProviderID = "yoyo-llm"

// openCodeKeyEnv 是自建端点的钥匙所在的环境变量。值由工人进程带入，不写进配置文件。
const openCodeKeyEnv = "OPENAI_API_KEY"

// errOpenCodeConfig 是工单目录里已经有一份认不出的 OpenCode 配置，不能改。
var errOpenCodeConfig = errors.New("opencode config is not ours")

// OpenCodeProvider 是一个 OpenAI 兼容的自建端点。Model 是接口上的模型名，不能带斜杠。
type OpenCodeProvider struct {
	BaseURL string
	Model   string
}

// Ref 是 session/set_model 要用的模型引用。模型或地址为空时返回空串。
func (p OpenCodeProvider) Ref() string {
	model := strings.TrimSpace(p.Model)
	if model == "" || strings.Contains(model, "/") || strings.TrimSpace(p.BaseURL) == "" {
		return ""
	}
	return openCodeProviderID + "/" + model
}

// OpenCodeModelRef 在 model 还不是「提供商/模型」、而且有自建地址时，返回 OpenCode 认的模型引用。
func OpenCodeModelRef(model, baseURL string) (string, bool) {
	ref := (OpenCodeProvider{BaseURL: baseURL, Model: model}).Ref()
	return ref, ref != ""
}

// OpenCodeFile 是一次要写进工单目录的 OpenCode 配置。MCPURL 和 Provider 可以只给其中一个。
type OpenCodeFile struct {
	MCPURL   string
	Provider *OpenCodeProvider
}

// WriteOpenCodeFile 合并控制面和自建提供商。钥匙只引用环境变量。
func WriteOpenCodeFile(dir string, file OpenCodeFile) error {
	dir = filepath.Clean(dir)
	url := strings.TrimSpace(file.MCPURL)
	if dir == "" || dir == "." || (url == "" && (file.Provider == nil || file.Provider.Ref() == "")) {
		return errOpenCodeConfig
	}
	if _, err := os.Stat(filepath.Join(dir, "opencode.jsonc")); err == nil {
		return errOpenCodeConfig
	}
	path := filepath.Join(dir, "opencode.json")
	body := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil && len(bytes.TrimSpace(raw)) > 0 {
		if json.Unmarshal(raw, &body) != nil {
			return errOpenCodeConfig
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if url != "" {
		mcp, err := objectField(body, "mcp")
		if err != nil {
			return err
		}
		servers, err := objectField(mcp, "servers")
		if err != nil {
			return err
		}
		servers["yoyo"] = map[string]any{
			"type":     "remote",
			"url":      url,
			"oauth":    false,
			"codemode": false,
			"headers": map[string]any{
				"Authorization": "Bearer {env:YOYO_MCP_TOKEN}",
			},
		}
		mcp["servers"] = servers
		body["mcp"] = mcp
	}
	if file.Provider != nil {
		if err := writeOpenCodeProvider(body, *file.Provider); err != nil {
			return err
		}
	}
	if _, ok := body["$schema"]; !ok {
		body["$schema"] = "https://opencode.ai/config.json"
	}
	raw, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

// writeOpenCodeProvider 把自建端点写成 OpenCode 的 providers 条目，并把默认模型指过去。
func writeOpenCodeProvider(body map[string]any, p OpenCodeProvider) error {
	ref := p.Ref()
	if ref == "" {
		return errOpenCodeConfig
	}
	providers, err := objectField(body, "providers")
	if err != nil {
		return err
	}
	model := strings.TrimSpace(p.Model)
	providers[openCodeProviderID] = map[string]any{
		"name":    "Yoyo",
		"package": "@opencode/ai/providers/openai-compatible",
		"env":     []string{openCodeKeyEnv},
		"settings": map[string]any{
			"baseURL": strings.TrimSpace(p.BaseURL),
			"apiKey":  "{env:" + openCodeKeyEnv + "}",
		},
		"models": map[string]any{
			model: map[string]any{
				"name":    model,
				"modelID": model,
				"capabilities": map[string]any{
					"tools":  true,
					"input":  []string{"text"},
					"output": []string{"text"},
				},
			},
		},
	}
	body["providers"] = providers
	body["model"] = ref
	return nil
}

// objectField 取出 m 里 key 的对象。没有就新建。类型不对时返回 errOpenCodeConfig。
func objectField(m map[string]any, key string) (map[string]any, error) {
	switch v := m[key].(type) {
	case nil:
		out := map[string]any{}
		m[key] = out
		return out, nil
	case map[string]any:
		return v, nil
	default:
		return nil, errOpenCodeConfig
	}
}
