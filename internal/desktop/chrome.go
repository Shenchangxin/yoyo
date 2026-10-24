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

// InstallChrome adds native menus, a tray icon, and typed desktop notifications.
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

	// group is the conversation name: Windows shows it as the toast header and
	// the Action Center group, where a raw session id would read as noise.
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
	// Coming back to the window is when the user has seen the toasts, so Action Center is emptied then.
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
		// Only the kinds that show a conversation name pay for the session lookup;
		// loading pulses fire on every tool call.
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

// pumpDesktopEvents never lets a frozen WebView stall Hub consumption.
// Live events drop if the renderer is still painting; turn_end still tries.
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

func showWindow(win application.Window) {
	Raise(win)
}

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

func payloadStr(p map[string]any, key string) string {
	if p == nil {
		return ""
	}
	s, _ := p[key].(string)
	return s
}

func firstPayloadStr(p map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := strings.TrimSpace(payloadStr(p, key)); s != "" {
			return s
		}
	}
	return ""
}

func notifyError(p map[string]any) string {
	title := payloadStr(p, "title")
	if title != "" {
		return title
	}
	return oneLine(payloadStr(p, "error"), 160)
}

// sessionTitle is the name of a conversation for notification text. It is ""
// when the session is unknown (remote RPC mode) or still has a placeholder title.
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
