package desktop

import (
	"encoding/json"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/icons"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
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
	tray.OnClick(func() { showWindow(win) })
	tray.OnRightClick(func() { tray.OpenMenu() })
	svc.tray = tray
	svc.menuLocale = svc.GetConfig().Locale
	svc.RebuildMenus()
	startGlobalHotkeys(svc)

	gui.Event.On("yoyo:do-quit", func(e *application.CustomEvent) {
		forceQuit(gui, svc, win)
	})

	notify := func(id, title, body, session, kind string) {
		gui.Event.Emit("yoyo:pulse", map[string]any{
			"kind":    kind,
			"title":   title,
			"body":    body,
			"session": session,
		})
		if kind == "loading" || ns == nil || !svc.NotifyAllowed() {
			return
		}
		if session != "" && !strings.Contains(id, session) {
			id = id + "-" + session
		}
		_ = ns.SendNotification(notifications.NotificationOptions{
			ID:       id,
			Title:    title,
			Body:     body,
			ThreadID: session,
			Data:     map[string]any{"session": session},
		})
	}
	if ns != nil {
		_, _ = ns.RequestNotificationAuthorization()
		ns.OnNotificationResponse(func(result notifications.NotificationResult) {
			showWindow(win)
			session := sessionFromNote(result.Response.ID, result.Response.UserInfo)
			gui.Event.Emit("yoyo:focus", session)
		})
	}
	dispatch := func(typ, source, session string, payload map[string]any) {
		kind := companionPulseKind(typ, source)
		if kind == "" {
			return
		}
		n := nativeCopy(svc.GetConfig().Locale)
		title := "Yoyo"
		body := n.TurnFinished
		id := "yoyo-turn"
		switch kind {
		case "loading":
			title = companionPulseTitle(kind, typ, n.Working, payloadStr(payload, "text"), payloadStr(payload, "name"))
			body = n.Working
			id = "yoyo-load"
		case "approval":
			title = n.Approval
			body = payloadStr(payload, "action")
			id = "yoyo-ask"
		case "error":
			title = n.ErrTitle
			body = notifyError(payload)
			id = "yoyo-err"
		case "eval":
			id = "yoyo-eval"
		case "evolve":
			id = "yoyo-evolve"
		}
		notify(id, title, body, session, kind)
	}
	if svc.App != nil {
		ch, _ := svc.App.Hub.Subscribe("*")
		go func() {
			for ev := range ch {
				gui.Event.Emit("yoyo:item", ev)
				dispatch(string(ev.Type), ev.Source, ev.SessionID, ev.Payload)
			}
		}()
	} else if svc.RPC != nil {
		go func() {
			for msg := range svc.RPC.Notify {
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

func (s *Service) RebuildMenus() {
	// Do not use application.InvokeSync here: InstallChrome runs before gui.Run,
	// when App.impl is still nil and dispatchOnMainThread panics.
	s.rebuildMenusNow()
}

func (s *Service) rebuildMenusNow() {
	gui := s.gui
	win := s.win
	if gui == nil {
		return
	}
	locale := s.menuLocale
	if locale == "" {
		locale = s.GetConfig().Locale
	}
	n := nativeCopy(locale)

	menu := gui.NewMenu()
	if runtime.GOOS == "darwin" {
		menu.AddRole(application.AppMenu)
	}
	file := menu.AddSubmenu(n.File)
	file.Add(n.NewSession).SetAccelerator("CmdOrCtrl+N").OnClick(func(ctx *application.Context) {
		m, err := s.CreateSession("")
		if err == nil {
			gui.Event.Emit("yoyo:sessions", m)
		}
	})
	file.Add(n.OpenWorkspace).SetAccelerator("CmdOrCtrl+O").OnClick(func(ctx *application.Context) {
		path, err := s.PickFolder()
		if err == nil && path != "" {
			gui.Event.Emit("yoyo:workspace", path)
		}
	})
	file.AddSeparator()
	file.Add(n.CloseWindow).SetAccelerator("CmdOrCtrl+W").OnClick(func(ctx *application.Context) {
		if win != nil {
			if s.CloseToTray() || s.CompanionEnabled() {
				win.Hide()
				return
			}
			win.Close()
		}
	})
	file.Add(n.Quit).SetAccelerator("CmdOrCtrl+Q").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:quit", nil)
	})

	menu.AddRole(application.EditMenu)

	view := menu.AddSubmenu(n.View)
	view.Add(n.ToggleSidebar).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "sidebar")
	})
	view.Add(n.ToggleReview).SetAccelerator("CmdOrCtrl+\\").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "review")
	})
	view.Add(n.ToggleDock).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "dock")
	})
	view.Add(n.OpenHarness).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "harness")
	})
	view.Add(n.OpenVideo).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "video")
	})
	view.AddSeparator()
	view.Add(n.Palette).SetAccelerator("CmdOrCtrl+K").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "palette")
	})
	view.Add(n.Settings).SetAccelerator("CmdOrCtrl+,").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "control")
	})

	thread := menu.AddSubmenu(n.Thread)
	thread.Add(n.Compact).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "compact")
	})
	thread.Add(n.Export).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "export")
	})
	thread.Add(n.Rename).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "rename")
	})
	thread.Add(n.Fork).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "fork")
	})
	thread.Add(n.Archive).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "archive")
	})

	harness := menu.AddSubmenu(n.Harness)
	harness.Add(n.OpenHarness).OnClick(func(ctx *application.Context) {
		showWindow(win)
		gui.Event.Emit("yoyo:command", "harness")
	})
	harness.Add(n.RunEval).OnClick(func(ctx *application.Context) {
		showWindow(win)
		gui.Event.Emit("yoyo:command", "run-eval")
	})
	harness.Add(n.RunCycle).OnClick(func(ctx *application.Context) {
		showWindow(win)
		gui.Event.Emit("yoyo:command", "run-evolve")
	})

	menu.AddRole(application.WindowMenu)

	help := menu.AddSubmenu(n.Help)
	help.Add(n.About).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:about", s.About())
		s.ShowAboutNative()
	})
	help.Add(n.Doctor).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:doctor", s.Doctor())
	})
	help.Add(n.Logs).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:logs", s.Logs(80))
	})
	gui.Menu.Set(menu)

	if s.tray == nil {
		return
	}
	tmenu := gui.NewMenu()
	tmenu.Add(n.Show).OnClick(func(ctx *application.Context) { showWindow(win) })
	tmenu.Add(n.Hide).OnClick(func(ctx *application.Context) {
		if win != nil {
			win.Hide()
		}
	})
	if s.CompanionEnabled() {
		label := n.ShowCompanion
		if s.companionShown() {
			label = n.HideCompanion
		}
		s.companionMenu = tmenu.Add(label).OnClick(func(ctx *application.Context) {
			s.ToggleCompanion()
		})
	} else {
		s.companionMenu = nil
	}
	tmenu.AddSeparator()
	tmenu.Add(n.NewChat).OnClick(func(ctx *application.Context) {
		showWindow(win)
		gui.Event.Emit("yoyo:command", "new")
	})
	tmenu.Add(n.Settings).OnClick(func(ctx *application.Context) {
		showWindow(win)
		gui.Event.Emit("yoyo:command", "control")
	})
	tmenu.AddSeparator()
	tmenu.Add(n.Quit).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:quit", nil)
	})
	s.tray.SetMenu(tmenu)
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

func notifyError(p map[string]any) string {
	title := payloadStr(p, "title")
	if title != "" {
		return title
	}
	err := payloadStr(p, "error")
	if len(err) > 160 {
		return err[:160] + "…"
	}
	return err
}
