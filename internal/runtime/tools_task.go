package runtime

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// TaskFunc runs an isolated child loop and returns a summary only —
// Claude Code's subagent pattern: the parent context never absorbs the
// child's tool transcript.
type TaskFunc func(prompt string, isolate bool) (string, error)

func (t *WorkspaceTools) task(prompt string, isolate bool) ToolResult {
	if strings.TrimSpace(prompt) == "" {
		return ToolResult{Err: fmt.Errorf("empty task prompt")}
	}
	if t != nil && t.Depth >= 1 {
		return ToolResult{Err: fmt.Errorf("nested subagent forbidden")}
	}
	if t == nil || t.Task == nil {
		return ToolResult{Err: fmt.Errorf("subagent not available")}
	}
	out, err := t.Task(prompt, isolate)
	if err != nil {
		return ToolResult{Content: out, Err: err}
	}
	capped, _ := capText(out, 2000)
	return ToolResult{Content: "SUBAGENT_SUMMARY:\n" + capped}
}

func spawnTask(ctx context.Context, parent RunRequest, prompt string, isolate bool) (string, error) {
	if parent.Tools != nil && parent.Tools.Depth >= 1 {
		return "", fmt.Errorf("nested subagent forbidden")
	}
	ws := parent.Workspace
	cleanup := func() {}
	if isolate && ws != "" {
		var err error
		ws, cleanup, err = IsolateWorkspace(parent.Workspace)
		if err != nil {
			return "", err
		}
		defer cleanup()
	}
	childLoop := parent.Loop
	childLoop.MaxTurns = 8
	childLoop.MaxToolMessages = 12
	var childTools *WorkspaceTools
	if parent.Tools != nil {
		cp := *parent.Tools
		cp.Workspace = ws
		cp.Depth = parent.Tools.Depth + 1
		cp.Task = nil
		if ws != "" {
			cp.Spill = NewSpill(filepath.Join(ws, ".yoyo", "context", "task"))
		}
		cp.PlanMode = parent.Tools.PlanMode
		childTools = &cp
	}
	out, err := Run(ctx, RunRequest{
		SessionID:        parent.SessionID + "-task",
		TaskID:           parent.TaskID,
		User:             prompt,
		Workspace:        ws,
		Harness:          parent.Harness,
		HarnessHash:      parent.HarnessHash,
		ModelFingerprint: parent.ModelFingerprint,
		Model:            parent.Model,
		Loop:             childLoop,
		Policy:           parent.Policy,
		Fragments:        parent.Fragments,
		Playbook:         parent.Playbook,
		Skills:           parent.Skills,
		Tools:            childTools,
		Client:           parent.Client,
		Trace:            parent.Trace,
		Events:           parent.Events,
		FileHooks:        parent.FileHooks,
		OnEvent:          parent.OnEvent,
		Meter:            parent.Meter,
	})
	payload := map[string]any{"prompt": prompt, "isolate": isolate, "summary": out}
	if err != nil {
		payload["error"] = err.Error()
	}
	emit(parent, trace.TypeSubagent, "runtime", payload)
	return out, err
}
