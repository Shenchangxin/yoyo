package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

var lookAgentReach = func() string {
	p, err := exec.LookPath("agent-reach")
	if err != nil {
		return ""
	}
	return p
}

var runAgentReachDoctor = func(ctx context.Context, bin string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, "doctor", "--json")
	hideConsole(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if out != "" {
			return out, fmt.Errorf("%s", msg)
		}
		return "", fmt.Errorf("%s", msg)
	}
	return out, nil
}

func (t *WorkspaceTools) reachStatus() ToolResult {
	var b strings.Builder
	b.WriteString("Internet backends (host tools first; never curl)\n")
	b.WriteString("\nsearch:\n")
	b.WriteString("  web_search uses the configured Search connection when it returns hits.\n")
	b.WriteString("  builtin fallbacks: Jina Search, then DuckDuckGo HTML (structured).\n")
	if t != nil && t.SearchAPI != nil {
		b.WriteString("  SearchAPI: hooked\n")
	} else {
		b.WriteString("  SearchAPI: not hooked (eval/no personal session)\n")
	}
	b.WriteString("\nread:\n")
	b.WriteString("  web_fetch extracts text; YouTube/Bilibili/V2EX/GitHub repo URLs are structured.\n")
	b.WriteString("  thin or blocked HTML retries via Jina Reader. Loopback stays local.\n")
	b.WriteString("  login walls: isolated browser_takeover + named profile. Not the daily Chrome.\n")
	bin := lookAgentReach()
	if bin == "" {
		b.WriteString("\nagent-reach CLI: not on PATH\n")
		b.WriteString("  optional sidecar. Do not install unless the operator asked.\n")
		b.WriteString("  generic search/read must still use web_search / web_fetch.\n")
		return ToolResult{Content: b.String()}
	}
	b.WriteString("\nagent-reach CLI: " + bin + "\n")
	ctx := context.Background()
	if t != nil && t.Ctx != nil {
		ctx = t.Ctx
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	out, err := runAgentReachDoctor(ctx, bin)
	if err != nil {
		b.WriteString("  doctor: " + err.Error() + "\n")
		if strings.TrimSpace(out) == "" {
			return ToolResult{Content: b.String()}
		}
	}
	capped, _ := capText(formatDoctorJSON(out), 8_000)
	b.WriteString("  doctor:\n")
	b.WriteString(capped)
	b.WriteString("\n  Use listed CLIs via shell only for platform-specific reads (gh, yt-dlp, bili, twitter).\n")
	b.WriteString("  Never curl, never OpenCLI onto the daily Chrome, never CDP 9222 unless lane=attached.\n")
	return ToolResult{Content: b.String()}
}

func formatDoctorJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var top map[string]any
	if json.Unmarshal([]byte(raw), &top) != nil {
		return raw
	}
	keys := make([]string, 0, len(top))
	for k := range top {
		keys = append(keys, k)
	}
	// stable-ish: json map iteration is random; sort via strings
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		m, _ := top[k].(map[string]any)
		if m == nil {
			continue
		}
		status, _ := m["status"].(string)
		active, _ := m["active_backend"].(string)
		msg, _ := m["message"].(string)
		fmt.Fprintf(&b, "  - %s: status=%s", k, status)
		if active != "" {
			fmt.Fprintf(&b, " backend=%s", active)
		}
		if msg != "" {
			msg = strings.Join(strings.Fields(msg), " ")
			if len([]rune(msg)) > 160 {
				msg = string([]rune(msg)[:160]) + "…"
			}
			fmt.Fprintf(&b, " — %s", msg)
		}
		b.WriteByte('\n')
	}
	if b.Len() == 0 {
		return raw
	}
	return b.String()
}
