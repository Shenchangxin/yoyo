// tools.go 按模型顺序执行一批工具调用。
// 独占调用自成屏障；可并行调用走有界池。取消时未开始的调用写成 ABORTED_BEFORE_DISPATCH。

package dsh

import (
	"context"
	"sort"
	"sync"
)

const (
	// ModeParallel 表示调用可以和其他可并行调用重叠。
	ModeParallel = "parallel"
	// ModeExclusive 表示调用单独占住这一批，前后都不重叠。
	ModeExclusive = "exclusive"
	// DefaultMaxParallel 是一步里同时执行的可并行调用上限，与 dsh-agent-loop 的默认值相同。
	DefaultMaxParallel = 10
)

// Result 是一次工具执行的模型可见结果。Concludes 为真时本回合在这批之后结束。
type Result struct {
	Text      string
	IsError   bool
	Code      string
	Concludes bool
	Contexts  []UserMessage
}

// Tool 是注册表里的一个工具。Mode 返回 ModeParallel 或 ModeExclusive。空 Mode 视为独占。
// Description 进入请求里的工具清单。Parameters 是 JSON Schema，原样发给模型。
type Tool struct {
	Name        string
	Description string
	Parameters  any
	Mode        func(call ToolCall) string
	Run         func(ctx context.Context, call ToolCall) (Result, error)
}

// ToolSchema 是发给模型的工具名、说明和参数模式。Parameters 为 nil 时只发名字和说明。
type ToolSchema struct {
	Name        string
	Description string
	Parameters  any
}

// Registry 按名字保存工具。缺名字时执行阶段返回错误结果，调度阶段视为独占。
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry 返回空注册表。
func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

// Reset 清空注册表。下一轮工具清单以重新登记的为准。
func (r *Registry) Reset() {
	r.mu.Lock()
	r.tools = map[string]Tool{}
	r.mu.Unlock()
}

// Add 登记 tool。同名后来的覆盖先前的。tool.Name 是查找键。
func (r *Registry) Add(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.tools == nil {
		r.tools = map[string]Tool{}
	}
	r.tools[tool.Name] = tool
}

// Schemas 按名字排序返回工具清单。排序让同一次注册的请求字节稳定。
func (r *Registry) Schemas() []ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]ToolSchema, 0, len(names))
	for _, name := range names {
		t := r.tools[name]
		out = append(out, ToolSchema{Name: t.Name, Description: t.Description, Parameters: t.Parameters})
	}
	return out
}

// get 按 name 取工具。没有这个名字时 ok 为假。
func (r *Registry) get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// mode 返回 call 的执行方式。没注册或没写 Mode 时视为独占。
func (r *Registry) mode(call ToolCall) string {
	t, ok := r.get(call.Name)
	if !ok || t.Mode == nil {
		return ModeExclusive
	}
	if t.Mode(call) == ModeParallel {
		return ModeParallel
	}
	return ModeExclusive
}

// ExecuteBatch 执行 calls。session 非空时按开始和结果写入日志。turn 与 step 记在这些事件上。
// maxParallel 小于 1 时改用 DefaultMaxParallel。ctx 取消后不再开始新调用。
// 返回本批是否有工具声明结束回合，以及工具附带的下一步上下文。
func ExecuteBatch(ctx context.Context, session *Session, turn, step int, calls []ToolCall, reg *Registry, maxParallel int) (bool, []UserMessage, error) {
	if maxParallel < 1 {
		maxParallel = DefaultMaxParallel
	}
	if reg == nil {
		reg = NewRegistry()
	}
	var contexts []UserMessage
	concluded := false
	next := 0
	for next < len(calls) {
		mode := reg.mode(calls[next])
		group := calls[next:]
		if mode != ModeParallel {
			group = calls[next : next+1]
		}
		consumed, aborted, groupConcluded, extra, err := runGroup(ctx, session, turn, step, reg, group, mode, maxParallel)
		contexts = append(contexts, extra...)
		concluded = concluded || groupConcluded
		next += consumed
		if err != nil {
			return concluded, contexts, err
		}
		if aborted {
			for _, call := range calls[next:] {
				writeAbort(session, turn, step, call)
			}
			return concluded, contexts, nil
		}
	}
	return concluded, contexts, nil
}

// runGroup 执行 group 里的一段。mode 为并行时取到第一个独占调用之前；否则只跑第一条。
// ctx 取消后不再开始新调用。session 非空时把开始和结果记到 turn 与 step。reg 提供实现。maxParallel 是同时运行的上限。
func runGroup(ctx context.Context, session *Session, turn, step int, reg *Registry, group []ToolCall, mode string, maxParallel int) (consumed int, aborted bool, concluded bool, contexts []UserMessage, err error) {
	limit := 1
	if mode == ModeParallel {
		limit = len(group)
		for i := 1; i < len(group); i++ {
			if reg.mode(group[i]) != ModeParallel {
				limit = i
				break
			}
		}
	}
	started := make([]bool, limit)
	results := make([]Result, limit)
	errs := make([]error, limit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, maxParallel)
	for i := 0; i < limit; i++ {
		if ctx.Err() != nil {
			aborted = true
			break
		}
		select {
		case <-ctx.Done():
			aborted = true
		case sem <- struct{}{}:
			if aborted || ctx.Err() != nil {
				<-sem
				aborted = true
				break
			}
			idx := i
			if session != nil {
				session.Append(EventToolCall, turn, step, group[idx])
			}
			started[idx] = true
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				res, runErr := dispatch(ctx, reg, group[idx])
				mu.Lock()
				results[idx] = res
				errs[idx] = runErr
				mu.Unlock()
			}()
		}
		if aborted {
			break
		}
	}
	wg.Wait()
	for i := 0; i < limit; i++ {
		if !started[i] {
			writeAbort(session, turn, step, group[i])
			continue
		}
		res := results[i]
		if errs[i] != nil && res.Text == "" {
			res.Text = errs[i].Error()
			res.IsError = true
		}
		writeResult(session, turn, step, group[i], res)
		if res.Concludes {
			concluded = true
		}
		contexts = append(contexts, res.Contexts...)
	}
	if aborted {
		return limit, true, concluded, contexts, nil
	}
	return limit, false, concluded, contexts, nil
}

// dispatch 执行 reg 里 call 对应的工具。ctx 已经取消时不调用，并返回未分派结果。
func dispatch(ctx context.Context, reg *Registry, call ToolCall) (Result, error) {
	tool, ok := reg.get(call.Name)
	if !ok || tool.Run == nil {
		return Result{Text: "tool not found: " + call.Name, IsError: true, Code: "TOOL_NOT_FOUND"}, nil
	}
	if ctx.Err() != nil {
		return Result{Text: textAborted, IsError: true, Code: ToolAbortedBeforeDispatch}, nil
	}
	return tool.Run(ctx, call)
}

// writeResult 把 call 的 res 写成工具结果。session 为 nil 时不写。turn 与 step 记在事件上。
func writeResult(session *Session, turn, step int, call ToolCall, res Result) {
	if session == nil {
		return
	}
	session.Append(EventToolResult, turn, step, ToolResultEvent{
		ID: call.ID, Name: call.Name, Text: res.Text, IsError: res.IsError, Code: res.Code,
	})
}

// writeAbort 为还没开始的 call 补上调用记录和未分派结果。session 为 nil 时不写。turn 与 step 记在事件上。
func writeAbort(session *Session, turn, step int, call ToolCall) {
	if session == nil {
		return
	}
	session.Append(EventToolCall, turn, step, call)
	writeResult(session, turn, step, call, Result{Text: textAborted, IsError: true, Code: ToolAbortedBeforeDispatch})
}
