package mcpserve

import (
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/plugin/mcp"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// bearerTransport 给每个请求补上 Bearer token。现有的 mcp 客户端不带 Authorization 头，测试在传输层补。
type bearerTransport struct {
	token string
	base  http.RoundTripper
}

// RoundTrip 复制请求，加上 Authorization 头后交给 base。
func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(r)
}

// serveWorkerTools 建一个控制面，按 superagent 的五个 orchestrator_* 名字登记 echo 工具和真正的 schema。
func serveWorkerTools(t *testing.T) (*Server, string) {
	t.Helper()
	s := New()
	t.Cleanup(func() { _ = s.Close() })
	schemas := superagent.WorkerToolSchemas()
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	tools := make([]dsh.Tool, 0, len(names))
	for _, name := range names {
		tools = append(tools, echo(name))
	}
	token := NewToken()
	if err := s.Register(token, "t1", tools, schemas); err != nil {
		t.Fatal(err)
	}
	return s, token
}

// checkHost 检查 h 列出了五个工具和它们的 schema，并能调用 orchestrator_report 拿到结果。
func checkHost(t *testing.T, h *mcp.Host) {
	t.Helper()
	tools := h.Tools()
	if len(tools) != len(superagent.WorkerToolSchemas()) {
		t.Fatalf("tools = %+v", tools)
	}
	for _, tool := range tools {
		if tool.Server != "yoyo" || tool.InputSchema["type"] != "object" || tool.InputSchema["additionalProperties"] != false {
			t.Fatalf("tool = %+v", tool)
		}
		if tool.Name == "orchestrator_report" {
			state, _ := tool.InputSchema["properties"].(map[string]any)["state"].(map[string]any)
			if enum, _ := state["enum"].([]any); len(enum) != len(superagent.ReportStates) {
				t.Fatalf("report state schema = %v", state)
			}
		}
	}
	raw, err := h.Call("yoyo", "orchestrator_report", `{"state":"progress","summary":"half way"}`)
	if err != nil {
		t.Fatal(err)
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Content) != 1 || res.Content[0].Text != `orchestrator_report {"state":"progress","summary":"half way"}` || res.IsError {
		t.Fatalf("call = %s", raw)
	}
	if _, err := h.Call("yoyo", "not_a_tool", `{}`); err == nil {
		t.Fatal("unknown tool accepted")
	}
}

// TestPluginClientTalksToServerOverHTTP 锁住现有的 mcp 客户端经 HTTP 握手、列出工具和 schema、调用工具拿到结果。
func TestPluginClientTalksToServerOverHTTP(t *testing.T) {
	s, token := serveWorkerTools(t)
	old := http.DefaultTransport
	http.DefaultTransport = bearerTransport{token: token, base: old}
	t.Cleanup(func() { http.DefaultTransport = old })
	h := mcp.NewHost()
	h.Timeout = 20 * time.Second
	t.Cleanup(func() { _ = h.Close() })
	if err := h.StartHTTP("yoyo", s.URL()); err != nil {
		t.Fatal(err)
	}
	checkHost(t, h)
}

// TestPluginClientTalksThroughTheBridge 锁住现有的 mcp 客户端用 Content-Length 分帧经 stdio 桥接上控制面，结果和直接走 HTTP 相同。
func TestPluginClientTalksThroughTheBridge(t *testing.T) {
	s, token := serveWorkerTools(t)
	t.Setenv("YOYO_MCP_BRIDGE_HELPER", "1")
	t.Setenv("YOYO_MCP_URL", s.URL())
	t.Setenv("YOYO_MCP_TOKEN", token)
	h := mcp.NewHost()
	h.Timeout = 20 * time.Second
	t.Cleanup(func() { _ = h.Close() })
	if err := h.Start("yoyo", os.Args[0], helperArgs); err != nil {
		t.Fatal(err)
	}
	checkHost(t, h)
}
