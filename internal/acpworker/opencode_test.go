package acpworker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteOpenCodeMCPListsToolsDirectly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "opencode.json"), []byte("{\n  \"plugin\": [\"keep\"]\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const url = "http://127.0.0.1:9/mcp"
	const token = "super-secret-token"
	if err := WriteOpenCodeFile(dir, OpenCodeFile{MCPURL: url}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	if json.Valid(raw) == false {
		t.Fatalf("invalid json: %s", raw)
	}
	text := string(raw)
	if strings.Contains(text, token) {
		t.Fatalf("token leaked into config: %s", text)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if plugins, _ := body["plugin"].([]any); len(plugins) != 1 || plugins[0] != "keep" {
		t.Fatalf("existing keys lost: %s", text)
	}
	mcp := body["mcp"].(map[string]any)
	servers := mcp["servers"].(map[string]any)
	yoyo := servers["yoyo"].(map[string]any)
	if yoyo["type"] != "remote" || yoyo["url"] != url || yoyo["codemode"] != false || yoyo["oauth"] != false {
		t.Fatalf("server = %#v", yoyo)
	}
	headers := yoyo["headers"].(map[string]any)
	if headers["Authorization"] != "Bearer {env:YOYO_MCP_TOKEN}" {
		t.Fatalf("headers = %#v", headers)
	}
	if err := os.WriteFile(filepath.Join(dir, "opencode.jsonc"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteOpenCodeFile(dir, OpenCodeFile{MCPURL: url}); err == nil {
		t.Fatal("an existing jsonc config should be left alone")
	}
}

func TestWriteOpenCodeFileRegistersProvider(t *testing.T) {
	dir := t.TempDir()
	const url = "http://127.0.0.1:9/mcp"
	const base = "https://server.example.com/v1"
	const model = "AIPC-deepseek-v4.1-flash"
	const key = "sk-do-not-write"
	if err := WriteOpenCodeFile(dir, OpenCodeFile{
		MCPURL:   url,
		Provider: &OpenCodeProvider{BaseURL: base, Model: model},
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), key) {
		t.Fatalf("api key leaked into config: %s", raw)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body["model"] != "yoyo-llm/"+model {
		t.Fatalf("model = %#v", body["model"])
	}
	providers := body["providers"].(map[string]any)
	item := providers["yoyo-llm"].(map[string]any)
	if item["package"] != "@opencode/ai/providers/openai-compatible" {
		t.Fatalf("package = %#v", item["package"])
	}
	settings := item["settings"].(map[string]any)
	if settings["baseURL"] != base || settings["apiKey"] != "{env:OPENAI_API_KEY}" {
		t.Fatalf("settings = %#v", settings)
	}
	models := item["models"].(map[string]any)
	entry := models[model].(map[string]any)
	if entry["modelID"] != model {
		t.Fatalf("model entry = %#v", entry)
	}
	caps := entry["capabilities"].(map[string]any)
	if caps["tools"] != true {
		t.Fatalf("capabilities = %#v", caps)
	}
	mcp := body["mcp"].(map[string]any)
	servers := mcp["servers"].(map[string]any)
	if _, ok := servers["yoyo"]; !ok {
		t.Fatal("mcp server missing")
	}
	if _, ok := OpenCodeModelRef("opencode/tidepool-alpha-free", base); ok {
		t.Fatal("a catalog model should stay as the agent named it")
	}
	ref, ok := OpenCodeModelRef(model, base)
	if !ok || ref != "yoyo-llm/"+model {
		t.Fatalf("ref = %q ok=%v", ref, ok)
	}
}
