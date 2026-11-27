// task 工具的子代理。父循环只拿摘要，不吸收子轨迹。子循环预算在 childLoopFor：用父循环上的 TaskMaxTurns 和 TaskMaxToolMessages。

package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
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

// shortID 读 6 字节随机数，编码成 12 位十六进制。
func shortID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
