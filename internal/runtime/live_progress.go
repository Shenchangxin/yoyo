package runtime

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

type toolProgressGate struct {
	id string
	n  int
	t  time.Time
}

func (g *toolProgressGate) allow(id string, n int, now time.Time) bool {
	if id == "" {
		return false
	}
	if id == g.id && n-g.n < 2048 && now.Sub(g.t) < 250*time.Millisecond {
		return false
	}
	g.id = id
	g.n = n
	g.t = now
	return true
}

func emitToolProgress(req RunRequest, roundID string, tc ToolCall) {
	if strings.TrimSpace(tc.ID) == "" || strings.TrimSpace(tc.Name) == "" {
		return
	}
	path := salvageJSONStringField(tc.Arguments, "path")
	args := map[string]any{}
	if path != "" {
		args["path"] = path
	}
	raw, _ := json.Marshal(args)
	payload := map[string]any{
		"name":      tc.Name,
		"id":        tc.ID,
		"round":     roundID,
		"progress":  true,
		"bytes":     len(tc.Arguments),
		"arguments": string(raw),
	}
	if path != "" {
		payload["path"] = path
	}
	emit(req, trace.TypeToolCall, "model", payload)
}

func payloadProgress(p map[string]any) bool {
	if p == nil {
		return false
	}
	v, _ := p["progress"].(bool)
	return v
}
