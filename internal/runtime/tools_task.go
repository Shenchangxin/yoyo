// task 工具的子代理。父循环只拿摘要，不吸收子轨迹。桌面预算在 childLoopFor：先用父循环上已经套过的 TaskMaxTurns 和 TaskMaxToolMessages，SoftHorizon 时再按 ChatMaxTurns、ChatMaxToolMessages 的一半抬下限。

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

	"github.com/Shenchangxin/yoyo/internal/artifact"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// MaxTaskDepth 是 2。父级深度 0 可以生出深度 1，深度 1 还可以再生一次到 2。更深禁止。
const MaxTaskDepth = 2

// TaskFunc 跑一轮子循环并只返回摘要。父上下文不吸收子工具轨迹。profile 是 explore、implement 或 qa。
type TaskFunc func(prompt string, isolate bool, profile string) (string, error)

// task 在提示为空、深度已达 MaxTaskDepth 或 Task 为空时返回错误。成功后把摘要截到 2000 个 rune，前缀 SUBAGENT_SUMMARY。
func (t *WorkspaceTools) task(prompt string, isolate bool, profile string) ToolResult {
	if strings.TrimSpace(prompt) == "" {
		return ToolResult{Err: fmt.Errorf("empty task prompt")}
	}
	if t != nil && t.Depth >= MaxTaskDepth {
		return ToolResult{Err: fmt.Errorf("nested subagent forbidden")}
	}
	if t == nil || t.Task == nil {
		return ToolResult{Err: fmt.Errorf("subagent not available")}
	}
	out, err := t.Task(prompt, isolate, profile)
	if err != nil {
		return ToolResult{Content: out, Err: err}
	}
	capped, _ := capText(out, 2000)
	return ToolResult{Content: "SUBAGENT_SUMMARY:\n" + capped}
}

// spawnTask 在父深度已达 MaxTaskDepth 时返回错误。先发出 phase 为 start 的子代理事件，再按 isolate 调用 IsolateWorkspace。子工具深度加一，Task 置空，ReadOnly 由 profile 决定，并 StripPackBootstrap。子循环的计划模式改成 auto。返回前发出 phase 为 complete 的事件。
func spawnTask(ctx context.Context, parent RunRequest, prompt string, isolate bool, profile string) (string, error) {
	if parent.Tools != nil && parent.Tools.Depth >= MaxTaskDepth {
		return "", fmt.Errorf("nested subagent forbidden")
	}
	ws := parent.Workspace
	childLoop := childLoopFor(parent, profile)
	childID := ChildSessionID(parent.SessionID)
	if parent.OnChild != nil {
		parent.OnChild(childID, prompt, false)
		defer parent.OnChild(childID, prompt, true)
	}
	// 先发出子代理卡片，再做隔离和 Run。复制失败时父轨迹里仍能看到这张卡片，而不是只剩一条工具结果。
	emit(parent, trace.TypeSubagent, "runtime", map[string]any{
		"prompt": prompt, "isolate": isolate, "child": childID, "phase": "start", "profile": profile,
	})
	cleanup := func() {}
	if isolate && ws != "" {
		var err error
		ws, cleanup, err = IsolateWorkspace(parent.Workspace)
		if err != nil {
			return "", err
		}
		defer cleanup()
	}
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
		cp.PlanMode = false
		cp.ReadOnly = readOnlyTaskProfile(profile)
		cp.MaxParallel = parent.Tools.MaxParallel
		cp.ChatOverlay = false
		cp.VerifyHint = false
		if cp.ReadOnly {
			cp.Advertised = readOnlyAdvertised(cp.Advertised)
		}
		StripPackBootstrap(&cp)
		cp.SkillHints = nil
		cp.Autoloaded = nil
		cp.SkillAutoInvoke = "off"
		childTools = &cp
	}
	user := prompt
	if strings.TrimSpace(parent.Loop.TaskInstruction) != "" {
		user = parent.Loop.TaskInstruction + "\n\n" + prompt
	}
	childPolicy := parent.Policy
	if strings.EqualFold(strings.TrimSpace(childPolicy.Mode), "plan") {
		childPolicy.Mode = "auto"
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
		Policy:           childPolicy,
		Fragments:        OverlayConduct(parent.Fragments, ConductOpts{Plan: childLoop.PlanMode, Methodology: false}),
		Playbook:         parent.Playbook,
		Skills:           parent.Skills,
		Tools:            childTools,
		Client:           parent.Client,
		Trace:            parent.Trace,
		Events:           parent.Events,
		FileHooks:        parent.FileHooks,
		StopHooks:        parent.StopHooks,
		PreCompactHooks:  parent.PreCompactHooks,
		OnEvent:          parent.OnEvent,
		Meter:            parent.Meter,
		Observe:          parent.Observe,
		Home:             parent.Home,
		ModelWindow:      parent.ModelWindow,
		TurnID:           diaglog.NewTurnID(),
		TraceID:          parent.TraceID,
		ShowThinking:     parent.ShowThinking,
	})
	payload := map[string]any{"prompt": prompt, "isolate": isolate, "summary": out, "child": childID, "phase": "complete", "profile": profile}
	if err != nil {
		payload["error"] = err.Error()
	}
	emit(parent, trace.TypeSubagent, "runtime", payload)
	return out, err
}

// taskFanout 并行跑多条提示。深度已达 MaxTaskDepth 或 Task 为空时返回错误。并行度取 MaxParallel，小于等于 0 时用 4，大于 8 时用 8。超出的提示被丢掉。每条摘要截到 1200 个 rune。
func (t *WorkspaceTools) taskFanout(prompts []string, isolate bool, profile string) ToolResult {
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
			out, err := t.Task(p, isolate, profile)
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

// readOnlyTaskProfile 在 profile 为 explore 或 qa 时返回真，其余为假。
func readOnlyTaskProfile(profile string) bool {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "explore", "qa":
		return true
	default:
		return false
	}
}

// readOnlyAdvertised 只保留 readOnlySubagentCall 允许的名字。过滤后为空时填入 read_file、list_dir、glob、grep、web_fetch、web_search、reach_status、recall_context、view_image。
func readOnlyAdvertised(have []string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(n string) {
		if n == "" || seen[n] || !readOnlySubagentCall(n) {
			return
		}
		seen[n] = true
		out = append(out, n)
	}
	for _, n := range have {
		add(n)
	}
	if len(out) == 0 {
		for _, n := range []string{"read_file", "list_dir", "glob", "grep", "web_fetch", "web_search", "reach_status", "recall_context", "view_image"} {
			add(n)
		}
	}
	return out
}

// exploreInstruction 要求只读工具、返回发现，并禁止写计划、调用 update_plan 和把文件正文抄回父级。
const exploreInstruction = "You are an explore subagent. Execute the prompt with read-only tools and return structured findings in your final letter. Do not write a plan, do not call update_plan, and do not copy file bodies back to the parent."

// qaInstruction 要求核对产物是否存在且符合规格，并禁止实现新功能和写计划。
const qaInstruction = "You are an independent evaluator. Check whether the required artifacts exist and match the spec. Return a verdict. Do not implement new features and do not write a plan."

// childLoopFor 给子循环定 MaxTurns 和 MaxToolMessages，并关掉 AllowLLMCompact。未设置时先用 24 轮、60 条工具消息。父循环 TaskMaxTurns、TaskMaxToolMessages 大于 0 时改用该字段。SoftHorizon 为真时，轮数低于 ChatMaxTurns/2 才抬到该值，工具消息低于 ChatMaxToolMessages/2 才抬到该值。ChatMaxTurns 是 64，一半是 32，32 只是下限，不是固定轮数。ChatMaxToolMessages 是 160，一半是 80。桌面会话进入循环前会先走 ApplyChatHorizon：TaskMaxTurns 低于 ChatTaskMaxTurns 时抬到 40，TaskMaxToolMessages 低于 ChatTaskMaxTools 时抬到 60。软地平线再套上面的下限，因此已经是 40 的轮数不会被改成 32，已经是 60 的工具消息会因为低于 80 而被抬到 80。explore 和 qa 会在 TaskInstruction 前接上对应说明。PlanMode 设为假。
func childLoopFor(parent RunRequest, profile string) artifact.LoopPreset {
	loop := parent.Loop
	loop.AllowLLMCompact = false
	turns, tools := 24, 60
	if loop.TaskMaxTurns > 0 {
		turns = loop.TaskMaxTurns
	}
	if loop.TaskMaxToolMessages > 0 {
		tools = loop.TaskMaxToolMessages
	}
	if parent.SoftHorizon {
		if turns < ChatMaxTurns/2 {
			turns = ChatMaxTurns / 2
		}
		if tools < ChatMaxToolMessages/2 {
			tools = ChatMaxToolMessages / 2
		}
	}
	loop.PlanMode = false
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case "explore":
		if strings.TrimSpace(loop.TaskInstruction) == "" {
			loop.TaskInstruction = exploreInstruction
		} else {
			loop.TaskInstruction = exploreInstruction + "\n\n" + loop.TaskInstruction
		}
	case "qa":
		if strings.TrimSpace(loop.TaskInstruction) == "" {
			loop.TaskInstruction = qaInstruction
		} else {
			loop.TaskInstruction = qaInstruction + "\n\n" + loop.TaskInstruction
		}
	}
	loop.MaxTurns = turns
	loop.MaxToolMessages = tools
	return loop
}

// taskSep 是 --task--，taskSepLegacy 是 /tasks/。子会话 ID 用前者拼接。
const (
	// taskSep 是 --task--，插在父会话 ID 和短随机串之间。
	taskSep = "--task--"
	// taskSepLegacy 是 /tasks/，只用于识别旧的子会话 ID。
	taskSepLegacy = "/tasks/"
)

// ChildSessionID 在父 ID 为空时用 session，再接 taskSep 和 12 位十六进制。结果不含斜杠。
func ChildSessionID(parent string) string {
	parent = strings.TrimSpace(parent)
	if parent == "" {
		parent = "session"
	}
	return parent + taskSep + shortID()
}

// IsChildSessionID 在 ID 含 taskSep 或 taskSepLegacy 时返回真。
func IsChildSessionID(id string) bool {
	return strings.Contains(id, taskSep) || strings.Contains(id, taskSepLegacy)
}

// ParentOfSession 返回 taskSep 或 taskSepLegacy 之前的父 ID。都没有则返回空串。
func ParentOfSession(id string) string {
	if i := strings.Index(id, taskSep); i >= 0 {
		return id[:i]
	}
	if i := strings.Index(id, taskSepLegacy); i >= 0 {
		return id[:i]
	}
	return ""
}

// shortID 读 6 字节随机数，编码成 12 位十六进制。
func shortID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
