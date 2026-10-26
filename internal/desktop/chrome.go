package desktop

import (
	"encoding/json"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/icons"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// InstallChrome 装上托盘、菜单、全局热键和桌面通知，并把窗口焦点时清空通知历史。
// 它监听 yoyo:do-quit 并调用 forceQuit。通知函数先发 yoyo:pulse。kind 为 loading、toast 为假、没有通知服务，或 toastAllowed 拒绝时，不发系统通知。用户点通知后 Raise 窗口，并按会话发出 yoyo:focus。
// 同进程订阅 Hub，由 pumpDesktopEvents 发出 yoyo:item，状态变化发出 yoyo:state。RPC 模式下只转发 state.changed 为 yoyo:state、item.event 为 yoyo:item，approval.request 和 turn.completed 不转发。
// gui 写入 svc.gui，用来建系统托盘、听 yoyo:do-quit，并发出 yoyo:pulse、yoyo:focus、yoyo:state、yoyo:item。
// win 写入 svc.win。非 darwin 点击托盘和点通知时传给 showWindow，获得焦点时清空通知历史。
// svc 记下 gui、win 和托盘，并调用 GetConfig、RebuildMenus、toastAllowed、noteToastSent、clearToastHistoryOnFocus、sessionTitle、noteDesktopPulse。
func InstallChrome(gui *application.App, win application.Window, svc *Service, ns *notifications.NotificationService) {
	svc.gui = gui
	svc.win = win

	tray := gui.SystemTray.New()
	tray.SetLabel("Yoyo")
	tray.SetTooltip("Yoyo")
	if len(AppIcon) > 0 {
		tray.SetIcon(AppIcon)
		if runtime.GOOS == "darwin" {
			tray.SetDarkModeIcon(AppIcon)
		}
	} else if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else {
		tray.SetIcon(icons.DefaultWindowsIcon)
	}
	if runtime.GOOS == "darwin" {
		tray.OnClick(func() {
			svc.refreshChrome()
			tray.OpenMenu()
		})
	} else {
		tray.OnClick(func() {
			showWindow(win)
			svc.refreshChrome()
		})
	}
	tray.OnRightClick(func() {
		svc.refreshChrome()
		tray.OpenMenu()
	})
	svc.tray = tray
	svc.menuLocale = svc.GetConfig().Locale
	svc.RebuildMenus()
	startGlobalHotkeys(svc)

	gui.Event.On("yoyo:do-quit", func(e *application.CustomEvent) {
		forceQuit(gui, svc, win)
	})

	// group 是会话名。Windows 把它显示成通知标题和操作中心分组，裸会话 id 读起来像噪声。
	notify := func(id, title, body, session, group, kind string, toast bool) {
		gui.Event.Emit("yoyo:pulse", map[string]any{
			"kind":    kind,
			"title":   title,
			"body":    body,
			"session": session,
		})
		if kind == "loading" || !toast || ns == nil || !svc.toastAllowed(kind) {
			return
		}
		if session != "" && !strings.Contains(id, session) {
			id = id + "-" + session
		}
		if err := ns.SendNotification(notifications.NotificationOptions{
			ID:       id,
			Title:    title,
			Body:     body,
			ThreadID: group,
			Data:     map[string]any{"session": session},
		}); err == nil {
			svc.noteToastSent()
		}
	}
	// 窗口重新获得焦点时，用户已经回到应用里，这时清空操作中心。
	win.OnWindowEvent(events.Common.WindowFocus, func(*application.WindowEvent) { svc.clearToastHistoryOnFocus() })
	if ns != nil {
		_, _ = ns.RequestNotificationAuthorization()
		ns.OnNotificationResponse(func(result notifications.NotificationResult) {
			showWindow(win)
			session := sessionFromNote(result.Response.ID, result.Response.UserInfo)
			gui.Event.Emit("yoyo:focus", session)
		})
	}
	dispatch := func(typ, source, session string, payload map[string]any) {
		if answeredOrAsk(typ, payload) {
			return
		}
		kind := companionPulseKind(typ, source)
		if kind == "" {
			return
		}
		n := nativeCopy(svc.GetConfig().Locale)
		// 只有会带上会话名的种类才去查会话标题。loading 每次工具调用都会响，不查标题。
		name := ""
		group := "Yoyo"
		if kind == "done" || kind == "approval" || kind == "error" {
			name = svc.sessionTitle(session)
			if name != "" {
				group = name
			}
		}
		title := "Yoyo"
		body := n.TurnFinished
		id := "yoyo-turn"
		switch kind {
		case "loading":
			title = companionPulseTitle(kind, typ, n.Working, payloadStr(payload, "text"), firstPayloadStr(payload, "skill", "name"), payloadStr(payload, "arguments"))
			body = n.Working
			id = "yoyo-load"
		case "done":
			if name != "" {
				title = name
			}
			if s := oneLine(payloadStr(payload, "text"), 140); s != "" {
				body = s
			}
		case "approval":
			title = n.Approval
			body = lineJoin(name, approvalLine(payload))
			id = "yoyo-ask"
			if typ == "ask_user" {
				title = n.Ask
				body = lineJoin(name, oneLine(payloadStr(payload, "question"), 140))
			}
		case "error":
			title = n.ErrTitle
			body = lineJoin(name, notifyError(payload))
			id = "yoyo-err"
		case "eval":
			id = "yoyo-eval"
		case "evolve":
			id = "yoyo-evolve"
		case "update":
			title = firstPayloadStr(payload, "title", "version")
			if title == "" {
				title = n.UpdateReady
			}
			body = payloadStr(payload, "text")
			if body == "" {
				body = n.UpdateReady
			}
			id = "yoyo-update"
		}
		notify(id, title, body, session, group, kind, !quietOutcome(typ, payload))
		svc.noteDesktopPulse(kind, session)
	}
	if svc.App != nil {
		ch, _ := svc.App.Hub.Subscribe("*")
		go pumpDesktopEvents(gui, ch, dispatch)
		stateCh, unsubState := svc.App.SubscribeState()
		go func() {
			defer unsubState()
			for change := range stateCh {
				gui.Event.Emit("yoyo:state", map[string]any{
					"epoch":  change.Epoch,
					"topic":  change.Topic,
					"object": change.Object,
				})
			}
		}()
	} else if svc.RPC != nil {
		go func() {
			for msg := range svc.RPC.Notify {
				if msg.Method == "state.changed" {
					gui.Event.Emit("yoyo:state", msg.Params)
					continue
				}
				if msg.Method != "item.event" {
					continue
				}
				gui.Event.Emit("yoyo:item", msg.Params)
				var m map[string]any
				_ = json.Unmarshal(msg.Params, &m)
				typ, _ := m["type"].(string)
				source, _ := m["source"].(string)
				payload, _ := m["payload"].(map[string]any)
				session, _ := m["session_id"].(string)
				dispatch(typ, source, session, payload)
			}
		}()
	}
}

// pumpDesktopEvents 用容量 256 的通道把 Hub 事件转成 yoyo:item，载荷是 app.NoticeOf，并调用 dispatch。
// 通道满时，KeepHubEvent 为假的事件丢掉。为真的再试一次，仍满则也丢掉。保留的类型有 turn_end、turn_start、step_start、step_end、error、approval、user、inject、inbox，source 为 update 的 system，以及 assistant 流的 start 和 end。带 delta 的事件不保留。
// gui 用来发出 yoyo:item。
func pumpDesktopEvents(gui *application.App, ch <-chan trace.Event, dispatch func(typ, source, session string, payload map[string]any)) {
	emit := make(chan trace.Event, 256)
	go func() {
		for ev := range emit {
			gui.Event.Emit("yoyo:item", app.NoticeOf(ev))
			dispatch(string(ev.Type), ev.Source, ev.SessionID, ev.Payload)
		}
	}()
	for ev := range ch {
		select {
		case emit <- ev:
		default:
			if !app.KeepHubEvent(ev) {
				continue
			}
			select {
			case emit <- ev:
			default:
			}
		}
	}
}

// showWindow 调用 Raise，把窗口从最小化或屏幕外拉回来并聚焦。
// win 传给 Raise。
func showWindow(win application.Window) {
	Raise(win)
}

// sessionFromNote 优先读通知 UserInfo 里的 session。否则从 id 前缀 yoyo-turn-、yoyo-ask-、yoyo-err- 后面切出会话 id。都没有则返回空字符串。
// info 非 nil 时读取 session 字符串，非空则返回。
func sessionFromNote(id string, info map[string]any) string {
	if info != nil {
		if s, ok := info["session"].(string); ok && s != "" {
			return s
		}
	}
	for _, p := range []string{"yoyo-turn-", "yoyo-ask-", "yoyo-err-"} {
		if strings.HasPrefix(id, p) {
			return strings.TrimPrefix(id, p)
		}
	}
	return ""
}

// payloadStr 读取 payload 里的字符串字段。payload 为空或类型不是字符串时返回空字符串。
// key 用来从 p 里取字符串。
func payloadStr(p map[string]any, key string) string {
	if p == nil {
		return ""
	}
	s, _ := p[key].(string)
	return s
}

// firstPayloadStr 按 keys 的顺序返回第一个去掉空白后非空的字符串。
// p 和每个 key 传给 payloadStr。
func firstPayloadStr(p map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := strings.TrimSpace(payloadStr(p, key)); s != "" {
			return s
		}
	}
	return ""
}

// notifyError 优先返回 payload 的 title。没有 title 时把 error 收成一行，最多 160 个字。
func notifyError(p map[string]any) string {
	title := payloadStr(p, "title")
	if title != "" {
		return title
	}
	return oneLine(payloadStr(p, "error"), 160)
}

// sessionTitle 取会话标题，收成一行，最多 60 个字。id 为空、没有同进程 App、会话不存在，或标题是 session、video 时返回空字符串。RPC 模式下 App 为空，因此通知里没有会话名。
func (s *Service) sessionTitle(id string) string {
	if id == "" || s.App == nil {
		return ""
	}
	m, err := s.App.GetSession(id)
	if err != nil {
		return ""
	}
	t := oneLine(m.Title, 60)
	if t == "session" || t == "video" {
		return ""
	}
	return t
}
