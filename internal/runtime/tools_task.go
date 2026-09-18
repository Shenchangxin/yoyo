package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Shenchangxin/yoyo/internal/trace"
)

// MaxTaskDepth is 2: parent (0) may spawn a child (1) which may spawn one
// more (2). Deeper nesting is forbidden.
const MaxTaskDepth = 2

// TaskFunc runs an isolated child loop and returns a summary only —
// Claude Code's subagent pattern: the parent context never absorbs the
// child's tool transcript.
type TaskFunc func(prompt string, isolate bool) (string, error)

func (t *WorkspaceTools) task(prompt string, isolate bool) ToolResult {
	if strings.TrimSpace(prompt) == "" {
		return ToolResult{Err: fmt.Errorf("empty task prompt")}
	}
	if t != nil && t.Depth >= MaxTaskDepth {
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
	if parent.Tools != nil && parent.Tools.Depth >= MaxTaskDepth {
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
		cp.MaxParallel = parent.Tools.MaxParallel
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

func (t *WorkspaceTools) taskFanout(prompts []string, isolate bool) ToolResult {
	if t != nil && t.Depth >= MaxTaskDepth {
		return ToolResult{Err: fmt.Errorf("nested subagent forbidden")}
	}
	if t == nil || t.Task == nil {
		return ToolResult{Err: fmt.Errorf("subagent not available")}
	}
	max := t.MaxParallel
	if max <= 0 {
		max = 4
	}
	if max > 8 {
		max = 8
	}
	if len(prompts) > max {
		prompts = prompts[:max]
	}
	type res struct {
		i   int
		out string
		err error
	}
	ch := make(chan res, len(prompts))
	sem := make(chan struct{}, max)
	var wg sync.WaitGroup
	for i, p := range prompts {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out, err := t.Task(p, isolate)
			ch <- res{i: i, out: out, err: err}
		}(i, p)
	}
	go func() { wg.Wait(); close(ch) }()
	got := make([]res, 0, len(prompts))
	for r := range ch {
		got = append(got, r)
	}
	sort.Slice(got, func(i, j int) bool { return got[i].i < got[j].i })
	var b strings.Builder
	for _, r := range got {
		fmt.Fprintf(&b, "## worker %d\n", r.i+1)
		if r.err != nil {
			fmt.Fprintf(&b, "ERROR: %s\n", r.err)
		}
		capped, _ := capText(r.out, 1200)
		b.WriteString(capped)
		b.WriteString("\n\n")
	}
	return ToolResult{Content: "SUBAGENT_FANOUT:\n" + b.String()}
}

func shortID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
