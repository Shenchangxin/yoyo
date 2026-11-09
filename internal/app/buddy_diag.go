// buddy_diag.go 拥有 Buddy 和外部 agent 在诊断日志里的样子：组件 buddy 记状态变化和人做的操作，组件 acp 记连接诊断和工人的失败，
// 调试分类 acp 打开时记下每一帧协议，打码后截到 2 KB；以及 Doctor 里的 Buddy 摘要。
// 不拥有打码规则（buddy.Redact）和日志落盘（internal/diaglog）。密钥值只用来打码，不写进任何一行。

package app

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/acp"
	"github.com/Shenchangxin/yoyo/internal/buddy"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
)

const (
	// acpDebugCat 是记协议帧的调试分类，用 YOYO_DEBUG=acp 或配置里的 log.debug 打开。
	acpDebugCat = "acp"
	// acpFrameMax 是一帧写进日志的最多字节数。
	acpFrameMax = 2 << 10
)

// buddyStates 记着每项最后写进日志的状态，安装进度这类刷新不重复写。
type buddyStates struct {
	mu   sync.Mutex
	last map[string]buddy.State
}

// changed 记下 id 的新状态，和上次一样时 ok 为假。第一次见到时 prev 为空串。
func (s *buddyStates) changed(id string, state buddy.State) (prev buddy.State, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last == nil {
		s.last = map[string]buddy.State{}
	}
	prev, seen := s.last[id]
	if seen && prev == state {
		return prev, false
	}
	s.last[id] = state
	return prev, true
}

// noteBuddy 在一项的状态变了时写一行 component=buddy 的日志，出错是警告。没装的项第一次出现时不写，免得一启动就列满整个目录。
func (a *App) noteBuddy(v buddy.View) {
	prev, ok := a.buddyLog.changed(v.ID, v.State)
	if !ok || (prev == "" && v.State == buddy.StateNotInstalled) {
		return
	}
	args := []any{"component", "buddy", "agent", v.ID, "state", string(v.State)}
	if prev != "" {
		args = append(args, "from", string(prev))
	}
	if d := strings.TrimSpace(v.Detail); d != "" {
		args = append(args, "detail", buddy.Redact(d))
	}
	if v.State == buddy.StateError {
		diaglog.Warn("buddy state", args...)
		return
	}
	diaglog.Info("buddy state", args...)
}

// logBuddyAction 写一行 component=buddy 的操作日志：哪一项（不针对某一项时 id 为空，不写 agent）、做了什么、成没成。
// 失败是警告，错误文本打过码。
func logBuddyAction(action, id string, err error, args ...any) {
	head := []any{"component", "buddy", "action", action}
	if id != "" {
		head = append(head, "agent", id)
	}
	args = append(head, args...)
	if err != nil {
		diaglog.Warn("buddy action failed", append(args, "err", buddy.Redact(err.Error()))...)
		return
	}
	diaglog.Info("buddy action", args...)
}

// patchFields 是一次配置改了哪些字段。密钥只列名字。
func patchFields(p buddy.ConfigPatch) []string {
	var out []string
	add := func(set bool, name string) {
		if set {
			out = append(out, name)
		}
	}
	add(p.Enabled != nil, "enabled")
	add(p.Mode != nil, "mode")
	add(p.Model != nil, "model")
	add(p.Approval != nil, "approval")
	add(p.MaxParallel != nil, "maxParallel")
	add(p.Args != nil, "args")
	add(p.Env != nil, "env")
	add(p.Reuse != nil, "reuse")
	add(p.Strengths != nil, "strengths")
	names := make([]string, 0, len(p.Secrets))
	for name := range p.Secrets {
		names = append(names, "secret:"+name)
	}
	slices.Sort(names)
	return append(out, names...)
}

// withACPDiag 给一次连接补上诊断：Log 为空时连接的坏行记成 component=acp 的警告；Frame 为空时装上帧日志，
// 每帧先看调试分类 acp 开没开，开了才打码、截断、写出。打码用到的密钥值从启动环境里像密钥的变量取。
func withACPDiag(opts acp.Options) acp.Options {
	name := opts.Name
	secrets := envSecrets(opts.Env)
	if opts.Log == nil {
		opts.Log = func(s string) {
			diaglog.Warn("acp connection", "component", "acp", "agent", name, "detail", buddy.Redact(s, secrets...))
		}
	}
	if opts.Frame == nil {
		opts.Frame = func(dir string, line []byte) {
			if !diaglog.DebugOn(acpDebugCat) {
				return
			}
			diaglog.Debug("acp frame", "component", "acp", "cat", acpDebugCat, "agent", name, "dir", dir, "frame", frameText(line, secrets))
		}
	}
	return opts
}

// envSecrets 取 env 里名字像密钥的变量的值。Buddy 的启动环境只放行白名单变量，像密钥的都是为这个 agent 注入的。
func envSecrets(env []string) []string {
	var out []string
	for _, kv := range env {
		i := strings.IndexByte(kv, '=')
		if i > 0 && buddy.SecretLike(kv[:i]) && kv[i+1:] != "" {
			out = append(out, kv[i+1:])
		}
	}
	return out
}

// frameText 是一帧写进日志的样子：先整帧打码再截断，免得密钥被截成半截躲过打码；截断时不切开一个字符，并注明原来有多少字节。
func frameText(line []byte, secrets []string) string {
	s := buddy.Redact(string(line), secrets...)
	if len(s) <= acpFrameMax {
		return s
	}
	cut := acpFrameMax
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + fmt.Sprintf("…(%d bytes)", len(line))
}

// buddyDoctor 是 Doctor 里的 Buddy 一节：管理器的摘要、控制面在不在听、登记着几张工单，以及记协议帧的调试分类开没开。
// 不开始检测，也不启动控制面。
func (a *App) buddyDoctor() map[string]any {
	m := a.Buddies()
	if m == nil {
		return map[string]any{"enabled": false}
	}
	out := map[string]any{"enabled": true, "summary": m.Summary(), "frames": diaglog.DebugOn(acpDebugCat)}
	if a.control != nil {
		out["control"] = map[string]any{"listening": a.control.URL() != "", "tickets": a.control.Registered()}
	}
	return out
}
