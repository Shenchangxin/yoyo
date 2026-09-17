package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	childLoop.AllowLLMCompact = false
	childID := parent.SessionID + "/tasks/" + shortID()
	var childTools *WorkspaceTools
	if parent.Tools != nil {
		cp := *parent.Tools
		cp.Workspace = ws
		cp.SessionID = childID
		cp.Depth = parent.Tools.Depth + 1
		cp.Task = nil
		cp.prefetch = nil
		if ws != "" {
			cp.Spill = NewSpill(filepath.Join(ws, ".yoyo", "context", "task", filepath.Base(childID)))
		}
		cp.PlanMode = parent.Tools.PlanMode
		childTools = &cp
	}
	user := prompt
	if strings.TrimSpace(parent.Loop.TaskInstruction) != "" {
		user = parent.Loop.TaskInstruction + "\n\n" + prompt
	}
	out, err := Run(ctx, RunRequest{
		SessionID:        childID,
		TaskID:           filepath.Base(childID),
		User:             user,
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
		StopHooks:        parent.StopHooks,
		OnEvent:          nil,
		Meter:            parent.Meter,
		Home:             parent.Home,
		ModelWindow:      parent.ModelWindow,
	})
	payload := map[string]any{"prompt": prompt, "isolate": isolate, "summary": out, "child": childID}
	if err != nil {
		payload["error"] = err.Error()
	}
	emit(parent, trace.TypeSubagent, "runtime", payload)
	return out, err
}

func shortID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
