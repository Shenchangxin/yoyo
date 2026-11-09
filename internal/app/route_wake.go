// route_wake.go 在任务组有了结果或要人处理时唤醒路由：在发起任务组的会话上开一轮没有用户行的路由回合，
// 任务组摘要就是这一轮的输入；做完了的任务组把路由的最后一段话记成结论。会话正忙、有排队的回合或应用暂停时让编排稍后再试。
// 不决定什么时候唤醒、唤醒几次（superagent 的 watchGroups），也不拼摘要（superagent 的 GroupDigest）。

package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/superagent"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// wakeCtxKey 是 ctx 里唤醒的键。
type wakeCtxKey struct{}

// turnWake 是唤醒路由的这一轮：写给路由的任务组摘要，以及任务组编号和状态。
type turnWake struct {
	Text  string
	Group string
	State string
}

// withWake 把 w 放进 ctx。sendLocked 看到它时不收这一轮的桌面消息，也不写用户行。
func withWake(ctx context.Context, w turnWake) context.Context {
	return context.WithValue(ctx, wakeCtxKey{}, w)
}

// wakeOf 取出 ctx 里的唤醒。没有，或摘要是空的时 ok 为假。
func wakeOf(ctx context.Context) (turnWake, bool) {
	w, ok := ctx.Value(wakeCtxKey{}).(turnWake)
	return w, ok && strings.TrimSpace(w.Text) != ""
}

// wakeRouter 是编排的唤醒回调。session 为空或那条会话已经没有了时回到主会话；主会话也没有时丢掉这次唤醒。
// 应用暂停、会话在跑、运行槽还没放或还有排队的回合时返回 ErrLater，两秒后再试：人排着的话先跑，唤醒排在后面。
func (a *App) wakeRouter(session, group, state, digest string) error {
	if a == nil {
		return nil
	}
	sid := a.wakeSession(session)
	if sid == "" {
		return nil
	}
	if a.IsPaused() || a.runSlotHeld(sid) || len(a.QueueList(sid)) > 0 {
		return superagent.ErrLater
	}
	return a.launchWake(sid, group, state, digest, nil)
}

// wakeSession 是唤醒要去的会话：session 还在就用它，否则用主会话。都没有时为空串。
func (a *App) wakeSession(session string) string {
	if session = strings.TrimSpace(session); session != "" {
		if _, err := a.GetSession(session); err == nil {
			return session
		}
	}
	if meta, ok := a.savedMain(); ok {
		return meta.ID
	}
	return ""
}

// pumpWake 让 sessionID 上等着的任务组唤醒马上再试。编排还没打开时什么都不做，不为这个去打开它。
func (a *App) pumpWake(sessionID string) {
	if a == nil {
		return
	}
	a.deskMu.Lock()
	rt := a.orch
	a.deskMu.Unlock()
	if rt != nil {
		rt.PumpWake(sessionID)
	}
}

// runSlotHeld 在 sessionID 的运行槽还占着时为真，包括这一轮已经跑完、槽还没放的那一小段。
func (a *App) runSlotHeld(sessionID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.runs[sessionID] != nil
}

// launchWake 占住 sessionID 的运行槽，异步跑一轮唤醒路由。client 为空时按会话设置取模型。槽被占着时返回 ErrLater。
// 这一轮跑完后，state 是 done 的任务组把路由的最后一段话记成结论；先放掉槽，再取下一条排队的回合。
func (a *App) launchWake(sessionID, group, state, digest string, client runtime.Client) error {
	ctx, cancel := chatTurnContext()
	ctx, release, err := a.acquireRun(sessionID, ctx)
	if err != nil {
		cancel()
		if err == errBusy {
			return superagent.ErrLater
		}
		return err
	}
	if a.Threads != nil {
		a.Threads.MarkTurnInFlight(sessionID, true, false)
	}
	ctx = withWake(ctx, turnWake{Text: digest, Group: group, State: state})
	go func() {
		defer a.kickQueue(sessionID)
		defer cancel()
		defer release()
		defer func() {
			if rec := recover(); rec != nil {
				path := diaglog.WriteCrash(fmt.Sprintf("wake panic: %v", rec), rec)
				payload := map[string]any{"error": fmt.Sprintf("wake panic: %v", rec)}
				if path != "" {
					payload["crash_path"] = path
				}
				if a.Hub != nil {
					a.Hub.Publish(trace.Event{Type: trace.TypeError, Source: "runtime", SessionID: sessionID, Payload: payload})
				}
			}
		}()
		out, err := a.sendLocked(ctx, sessionID, "", client, nil, false, nil, true, false, "")
		// sendLocked 在取不到模型这类早退里不会清掉进行中的标记，重启后会把这一轮当成没跑完再续一次。
		if a.Threads != nil {
			a.Threads.MarkTurnInFlight(sessionID, false, false)
		}
		if err != nil || state != superagent.GroupDone || strings.TrimSpace(out) == "" {
			return
		}
		if rt := a.orchestrator(); rt != nil {
			_ = rt.SetGroupVerdict(group, out)
		}
	}()
	return nil
}
