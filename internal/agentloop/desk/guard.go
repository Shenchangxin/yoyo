// guard.go 是桌面回合的安全网。agentloop 代理每一步调用模型之前（agentloop.Options.PreStep），它看一眼这一轮新写进日志的事件：
// 发现原地打转，先在这一步的消息里追加一条提醒；提醒之后还在打转，就让这一步不进模型，回合以 blocked 结束，Host.Run 再发出停止卡片。
// 判据只看工具名、参数原文和错误文本，不认识具体工具，浏览器、shell、连接器一视同仁。
// 停止卡片只有 stuck 和 max_turns 两种，前端按这两种渲染。子代理也经 Host.Run，同样受保护。评测 Harbor 另外给一个硬步数上限，见 guard.maxSteps。

package desk

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode"

	"github.com/Shenchangxin/yoyo/internal/agentloop"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

const (
	// errorNudgeHits 是 3：同一条工具错误（工具名加错误首行）出现 3 次，提醒模型换做法。
	errorNudgeHits = 3
	// errorStopHits 是 6：同一条工具错误出现 6 次，停下这一轮，卡片种类 stuck。
	errorStopHits = 6
	// repeatNudgeHits 是 3：同一个调用（工具名和参数原文完全相同）出现 3 次，不论成败都提醒。反复 wait、反复打开同一页都落在这里。
	repeatNudgeHits = 3
	// repeatStopHits 是 8：同一个调用出现 8 次，停下这一轮，卡片种类 stuck。
	repeatStopHits = 8
	// sourceGuard 是提醒消息的来源。project.go 把它投影成一条 inject 轨迹，回放时看得见。
	sourceGuard = "guard"
)

const (
	errorNudgeEN  = "The same tool error keeps repeating (%s). Stop retrying that call. Use a different approach or finish with what you already have."
	errorNudgeZH  = "同一个工具错误在重复（%s）。不要再用同样的调用。换一条路，或用已经拿到的结果收束。"
	repeatNudgeEN = "You have made the same %s call %d times with identical arguments. The result will not change. Use what you already have, change the arguments or the approach, or finish."
	repeatNudgeZH = "你已经用完全相同的参数调用 %s %d 次，结果不会变。用已经拿到的结果，换参数或换做法，或者直接收束。"
)

// guard 是一轮桌面发送的停滞计数。Host.Run 每一轮新建一份，只看这一轮开始之后的事件。
// 轮数保护沿用 runtime.ChatSoftGuardTurns（192）和 runtime.ChatStallWindow（24）：步数到上限时，最近 24 步里有过进展就复位接着跑，否则停下，卡片种类 max_turns。
// 进展指一次成功的、不是只读的工具调用，并且这一轮里第一次出现同样的工具名和参数。只读查看和原样重做都不算。
type guard struct {
	mu      sync.Mutex
	session *agentloop.Session
	// next 是下一个要看的事件下标。
	next int
	// han 为真时提醒用中文，跟着这一轮操作者的原话走。
	han bool
	// steps 是这一轮已经跑过的步骤数，按 step/start 计。
	steps int
	// base 是轮数保护上一次复位时的步骤数。
	base int
	// progressAt 是最近一次有进展时的步骤数。
	progressAt int
	// askedTools 为真表示最近一条助手消息调用了工具，模型还在干活。模型已经说完时不做判断，免得把正常结束改成 blocked。
	askedTools bool
	calls      map[string]agentloop.ToolCall
	callHits   map[string]int
	errHits    map[string]int
	nudged     map[string]bool
	// maxSteps 大于 0 时是这一轮的硬步数上限，到了就停，不看进展，卡片种类 max_turns。评测 Harbor 用它保住可复现的轮数；桌面聊天传 0，只走上面的软保护。
	maxSteps int
	// stop 非空表示已经决定停下，card 是要发出的卡片。
	stop error
	card runtime.ErrorInfo
}

// newGuard 从 session 现有事件的末尾开始计数。user 是这一轮操作者的原话，决定提醒用中文还是英文。maxSteps 见 guard.maxSteps。
func newGuard(session *agentloop.Session, user string, maxSteps int) *guard {
	g := &guard{
		session:  session,
		maxSteps: maxSteps,
		han:      hasHan(user),
		calls:    map[string]agentloop.ToolCall{},
		callHits: map[string]int{},
		errHits:  map[string]int{},
		nudged:   map[string]bool{},
	}
	if session != nil {
		g.next = len(session.Events())
	}
	return g
}

// preStep 是 agentloop.Options.PreStep。返回 enter 为假时这一步不调用模型，回合以 blocked 结束。
// 有提醒时把它们接在 claimed 后面一起交给这一步。g 为 nil 时什么都不做。
func (g *guard) preStep(_ context.Context, claimed []agentloop.UserMessage) (bool, []agentloop.UserMessage, error) {
	if g == nil {
		return true, nil, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.scan()
	if g.stop != nil {
		return false, nil, nil
	}
	if !g.askedTools {
		return true, nil, nil
	}
	if sig, n := top(g.errHits); n >= errorStopHits {
		g.halt(runtime.StuckLoopErr("the same tool failure kept repeating: " + sig))
		return false, nil, nil
	}
	if key, n := top(g.callHits); n >= repeatStopHits {
		g.halt(runtime.StuckLoopErr("the same tool call kept repeating: " + callName(key)))
		return false, nil, nil
	}
	if g.maxSteps > 0 && g.steps >= g.maxSteps {
		g.stop = runtime.MaxTurnsErr()
		g.card = runtime.MaxTurnsInfo(g.maxSteps)
		return false, nil, nil
	}
	if g.steps-g.base >= runtime.ChatSoftGuardTurns {
		if g.steps-g.progressAt < runtime.ChatStallWindow {
			g.base = g.steps
		} else {
			g.stop = runtime.MaxTurnsErr()
			g.card = runtime.MaxTurnsInfo(runtime.ChatSoftGuardTurns)
			return false, nil, nil
		}
	}
	var notes []agentloop.UserMessage
	if sig, n := top(g.errHits); n >= errorNudgeHits && !g.nudged["e:"+sig] {
		g.nudged["e:"+sig] = true
		notes = append(notes, agentloop.UserMessage{Text: g.say(errorNudgeEN, errorNudgeZH, sig), Source: sourceGuard})
	}
	if key, n := top(g.callHits); n >= repeatNudgeHits && !g.nudged["c:"+key] {
		g.nudged["c:"+key] = true
		notes = append(notes, agentloop.UserMessage{Text: g.say(repeatNudgeEN, repeatNudgeZH, callName(key), n), Source: sourceGuard})
	}
	if len(notes) == 0 {
		return true, nil, nil
	}
	return true, append(append([]agentloop.UserMessage(nil), claimed...), notes...), nil
}

// halted 返回停止的错误和卡片。没有停下时 ok 为假。
func (g *guard) halted() (err error, card runtime.ErrorInfo, ok bool) {
	if g == nil {
		return nil, runtime.ErrorInfo{}, false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.stop, g.card, g.stop != nil
}

// halt 记下 err 并按它分类出卡片。调用时要已持有锁。
func (g *guard) halt(err error) {
	g.stop = err
	g.card = runtime.ClassifyError(err)
}

// scan 把 next 之后的新事件计进各项计数。检查点或修复让日志变短时，从新的末尾接着看。调用时要已持有锁。
func (g *guard) scan() {
	if g.session == nil {
		return
	}
	evs := g.session.Events()
	if g.next > len(evs) {
		g.next = len(evs)
	}
	for ; g.next < len(evs); g.next++ {
		ev := evs[g.next]
		switch ev.Type {
		case agentloop.EventStepStart:
			g.steps++
		case agentloop.EventAssistant:
			msg, _ := ev.Data.(agentloop.Assistant)
			g.askedTools = len(msg.ToolCalls) > 0
		case agentloop.EventToolCall:
			call, _ := ev.Data.(agentloop.ToolCall)
			g.calls[call.ID] = call
			g.callHits[callKey(call)]++
		case agentloop.EventToolResult:
			res, _ := ev.Data.(agentloop.ToolResultEvent)
			if res.IsError {
				g.errHits[errorKey(res)]++
				continue
			}
			call, ok := g.calls[res.ID]
			if ok && g.callHits[callKey(call)] == 1 && !runtime.HostAnn(call.Name).ReadOnly {
				g.progressAt = g.steps
			}
		}
	}
}

// say 按 han 选一份文案，再用 args 填进去。
func (g *guard) say(en, zh string, args ...any) string {
	if g.han {
		return fmt.Sprintf(zh, args...)
	}
	return fmt.Sprintf(en, args...)
}

// callKey 是工具名加参数原文，中间用 NUL 隔开。参数只要差一个字节就是另一个调用。
func callKey(call agentloop.ToolCall) string {
	return call.Name + "\x00" + call.Arguments
}

// callName 从 callKey 里取回工具名。
func callName(key string) string {
	name, _, _ := strings.Cut(key, "\x00")
	return name
}

// errorKey 是工具名加错误正文的第一行，第一行最多 160 个 rune。错误里常带的长路径和堆栈不进签名。
func errorKey(res agentloop.ToolResultEvent) string {
	line, _, _ := strings.Cut(strings.TrimSpace(res.Text), "\n")
	if r := []rune(line); len(r) > 160 {
		line = string(r[:160])
	}
	return res.Name + ": " + line
}

// top 返回计数最高的键和它的计数。计数相同时取字典序靠前的，结果稳定。hits 为空时返回空串和 0。
func top(hits map[string]int) (string, int) {
	best, n := "", 0
	for k, c := range hits {
		if c > n || (c == n && k < best) {
			best, n = k, c
		}
	}
	return best, n
}

// hasHan 在 text 含汉字时为真。
func hasHan(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
