// tools.go 把 runtime.WorkspaceTools 登记成 dsh 工具。
// shell 和 run_skill_script 超过前台预算后转入后台。命令仍走原来的 Call，不另起一套执行器。

package dshdesk

import (
	"context"
	"sync"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// toolBox 保存这一轮的工具集，以及还没结束的后台命令数。
type toolBox struct {
	mu    sync.Mutex
	tools *runtime.WorkspaceTools
	jobs  int
	note  chan struct{}
}

// set 换成 tools。tools 为 nil 时后续调用得到空工具错误。
func (b *toolBox) set(tools *runtime.WorkspaceTools) {
	b.mu.Lock()
	b.tools = tools
	b.mu.Unlock()
}

// get 返回当前工具集。
func (b *toolBox) get() *runtime.WorkspaceTools {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tools
}

// beginJob 记一个还在跑的后台命令。
func (b *toolBox) beginJob() {
	b.mu.Lock()
	b.jobs++
	b.mu.Unlock()
}

// endJob 结束一个后台命令，并通知等待中的 Run。jobs 已经是 0 时不再减。
func (b *toolBox) endJob() {
	b.mu.Lock()
	if b.jobs > 0 {
		b.jobs--
	}
	b.mu.Unlock()
	select {
	case b.note <- struct{}{}:
	default:
	}
}

// jobsLeft 返回还没结束的后台命令数。
func (b *toolBox) jobsLeft() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.jobs
}

// bind 把当前工具集登记进 reg。同名覆盖。shell 和 run_skill_script 走后台预算，其余直接 Call。
func (b *toolBox) bind(reg *dsh.Registry, agent *dsh.Agent) {
	if reg == nil {
		return
	}
	tools := b.get()
	for _, spec := range runtime.AllToolJSON(tools) {
		name, _ := spec.Function["name"].(string)
		if name == "" {
			continue
		}
		desc, _ := spec.Function["description"].(string)
		params := spec.Function["parameters"]
		toolName := name
		reg.Add(dsh.Tool{
			Name:        toolName,
			Description: desc,
			Parameters:  params,
			Mode: func(dsh.ToolCall) string {
				return b.mode(toolName)
			},
			Run: func(ctx context.Context, call dsh.ToolCall) (dsh.Result, error) {
				if toolName == "shell" || toolName == "run_skill_script" {
					return b.runShell(ctx, agent, toolName, call)
				}
				return b.runCall(toolName, call), nil
			},
		})
	}
}

// mode 返回 name 的执行方式。shell、run_skill_script，以及独占或开放世界的工具返回 ModeExclusive。只读或声明可并发的返回 ModeParallel。
func (b *toolBox) mode(name string) string {
	if exclusiveName(b.get(), name) {
		return dsh.ModeExclusive
	}
	ann := runtime.HostAnn(name)
	if ann.ReadOnly || ann.ConcurrencySafe {
		return dsh.ModeParallel
	}
	return dsh.ModeExclusive
}

// exclusiveName 对 shell 和 run_skill_script 返回真。tools 的 Extra 里 Exclusive 或 OpenWorld 为真也返回真。其余看 HostAnn。
func exclusiveName(tools *runtime.WorkspaceTools, name string) bool {
	if name == "shell" || name == "run_skill_script" {
		return true
	}
	if tools != nil && tools.Extra != nil {
		if extra, ok := tools.Extra[name]; ok && (extra.Exclusive || extra.OpenWorld) {
			return true
		}
	}
	ann := runtime.HostAnn(name)
	return ann.Exclusive || ann.OpenWorld
}

// runCall 同步执行 name。call 的参数原样交给 WorkspaceTools.Call。错误写进结果，不中断这一批。
func (b *toolBox) runCall(name string, call dsh.ToolCall) dsh.Result {
	tools := b.get()
	if tools == nil {
		return dsh.Result{Text: "no tools", IsError: true}
	}
	return resultOf(tools.Call(name, call.Arguments))
}

// runShell 在 dsh.DefaultForeground 内等待命令。ctx 取消时前台等待结束。超时后命令继续在这一轮的上下文里跑，完成时 Steer 一条来源为 job 的消息。
func (b *toolBox) runShell(ctx context.Context, agent *dsh.Agent, name string, call dsh.ToolCall) (dsh.Result, error) {
	b.beginJob()
	view := dsh.RunForeground(ctx, call.ID, dsh.DefaultForeground, func(context.Context) (string, error) {
		tools := b.get()
		if tools == nil {
			return "", errString("no tools")
		}
		res := tools.Call(name, call.Arguments)
		if res.Err != nil && res.Content == "" {
			return "", res.Err
		}
		return res.Content, res.Err
	}, func(output string, err error) {
		if ctx.Err() != nil {
			b.endJob()
			return
		}
		text := output
		if err != nil && text == "" {
			text = err.Error()
		}
		if agent != nil {
			agent.Steer(dsh.UserMessage{Text: "job " + call.ID + " finished\n" + text, Source: "job"})
		}
		b.endJob()
	})
	if !view.Background {
		b.endJob()
		if view.Err != nil && view.Output == "" {
			return dsh.Result{Text: view.Err.Error(), IsError: true}, nil
		}
		if view.Err != nil {
			return dsh.Result{Text: view.Output, IsError: true}, nil
		}
		return dsh.Result{Text: view.Output}, nil
	}
	return dsh.Result{Text: "command moved to background as " + call.ID}, nil
}

// resultOf 把 res 收成 dsh 结果。Err 非空时标成错误；正文为空时用错误文本。
func resultOf(res runtime.ToolResult) dsh.Result {
	text := res.Content
	if res.Err != nil {
		if text == "" {
			text = res.Err.Error()
		}
		return dsh.Result{Text: text, IsError: true}
	}
	return dsh.Result{Text: text}
}
