// perm.go 拥有外部 agent 的权限请求：把调用的 kind、路径、命令和原始参数交给注入的 Approve，再按结果选选项。
// 允许时先选 allow_once，没有才选 allow_always；拒绝时选 reject_once，并在日志里把这次调用记成被拒绝；这一轮被取消时答 cancelled。
// 等得久时在工单上写「等批准：…」。
// 不拥有能力级别的对应和 Broker（app 的 Approve），也不切换 agent 的模式。

package acpworker

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// attentionDelay 是审批等多久之后才在工单上写「等批准」。马上就有结果的不写，免得一闪而过。
var attentionDelay = time.Second

// Approval 是一次权限请求交给 app 判断的内容。
type Approval struct {
	// Kind 是 ACP 的种类：read、edit、delete、move、search、execute、think、fetch、switch_mode、other。
	Kind string
	// Name 是归一后的工具名，例如 shell、edit_file、orchestrator_report。
	Name string
	// Title 是 agent 给人看的说明。
	Title string
	// Paths 是调用碰到的绝对路径。
	Paths []string
	// Command 是执行类调用的命令。
	Command string
	// Input 是调用的原始参数。
	Input json.RawMessage
}

// Label 是给人看的一句：命令、路径或标题。
func (a Approval) Label() string {
	switch {
	case a.Command != "":
		return clipRunes(oneLine(a.Command), 160)
	case len(a.Paths) > 0:
		return clipRunes(a.Name+" "+strings.Join(a.Paths, "、"), 160)
	case a.Title != "":
		return clipRunes(oneLine(a.Title), 160)
	default:
		return a.Name
	}
}

// approvalOf 把合并后的调用换成 Approval。
func approvalOf(c toolCall, dir string) Approval {
	a := Approval{Kind: c.kind, Name: c.name, Title: c.title, Paths: callPaths(&c, dir), Input: c.rawInput}
	if c.kind == "execute" {
		a.Command = commandOf(&c)
	}
	return a
}

// onPermission 处理 session/request_permission。Approve 为 nil、出错或拒绝时选拒绝；对方没给允许选项时也拒绝。
func (w *Worker) onPermission(ctx context.Context, req acp.PermissionRequest) (acp.PermissionOutcome, error) {
	ap := approvalOf(w.jr.merge(req.ToolCall), w.spec.Dir)
	stop := w.waitNote(ap.Label())
	ok := false
	if w.cfg.Approve != nil {
		allowed, err := w.cfg.Approve(ctx, ap)
		ok = err == nil && allowed
	}
	stop()
	if ctx.Err() != nil {
		return acp.Cancelled(), nil
	}
	if ok {
		if out := acp.Allow(req.Options); out.Outcome != "" {
			return out, nil
		}
	}
	w.jr.deny(req.ToolCall.ToolCallID)
	return acp.Reject(req.Options), nil
}

// waitNote 在 attentionDelay 之后写「等批准：label」。返回的函数停掉计时；已经写过时清掉。
func (w *Worker) waitNote(label string) func() {
	if w.spec.OnAttention == nil {
		return func() {}
	}
	var mu sync.Mutex
	shown, stopped := false, false
	timer := time.AfterFunc(attentionDelay, func() {
		mu.Lock()
		defer mu.Unlock()
		if stopped {
			return
		}
		shown = true
		w.spec.OnAttention("等批准：" + label)
	})
	return func() {
		timer.Stop()
		mu.Lock()
		defer mu.Unlock()
		stopped = true
		if shown {
			w.spec.OnAttention("")
		}
	}
}
