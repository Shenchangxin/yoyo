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
	menu := gui.NewMenu()
	if runtime.GOOS == "darwin" {
		menu.AddRole(application.AppMenu)
	}
	file := menu.AddSubmenu("File")
	file.Add("New Session").SetAccelerator("CmdOrCtrl+N").OnClick(func(ctx *application.Context) {
		m, err := svc.CreateSession("")
		if err == nil {
			gui.Event.Emit("yoyo:sessions", m)
		}
	})
	file.Add("Open Workspace…").SetAccelerator("CmdOrCtrl+O").OnClick(func(ctx *application.Context) {
		path, err := svc.PickFolder()
		if err == nil && path != "" {
			gui.Event.Emit("yoyo:workspace", path)
		}
	})
	file.AddSeparator()
	file.Add("Close Window").SetAccelerator("CmdOrCtrl+W").OnClick(func(ctx *application.Context) {
		if win != nil {
			if svc.CloseToTray() {
				win.Hide()
				return
			}
			win.Close()
		}
	})
	file.Add("Quit").SetAccelerator("CmdOrCtrl+Q").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:quit", nil)
	})

	menu.AddRole(application.EditMenu)

	view := menu.AddSubmenu("View")
	view.Add("Toggle Review").SetAccelerator("CmdOrCtrl+\\").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "review")
	})
	view.Add("Command Palette").SetAccelerator("CmdOrCtrl+K").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "palette")
	})
	view.Add("Control").SetAccelerator("CmdOrCtrl+,").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "control")
	})

	thread := menu.AddSubmenu("Thread")
	thread.Add("Compact context").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "compact")
	})
	thread.Add("Export markdown").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "export")
	})
	thread.Add("Rename").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "rename")
	})
	thread.Add("Fork").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "fork")
	})

	harness := menu.AddSubmenu("Harness")
	harness.Add("Run eval suite").OnClick(func(ctx *application.Context) {
		_, _ = svc.RunEval()
	})
	harness.Add("Run Terminal-Bench subset").OnClick(func(ctx *application.Context) {
		_, _ = svc.RunEvalTB()
	})
	harness.Add("Evolve").OnClick(func(ctx *application.Context) {
		_, _ = svc.Evolve()
	})
	harness.AddSeparator()
	harness.Add("Apply staged update").OnClick(func(ctx *application.Context) {
		_ = svc.ApplyUpdate()
	})

	help := menu.AddSubmenu("Help")
	help.Add("About Yoyo").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:about", svc.About())
		svc.ShowAboutNative()
	})
	help.Add("Doctor").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:doctor", svc.Doctor())
	})
	help.Add("Logs").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:logs", svc.Logs(80))
	})
	gui.Menu.Set(menu)

	tray := gui.SystemTray.New()
	tray.SetTooltip("Yoyo")
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else {
		tray.SetIcon(icons.DefaultWindowsIcon)
	}
	tmenu := gui.NewMenu()
	tmenu.Add("Show").OnClick(func(ctx *application.Context) {
		if win != nil {
			win.Show().Focus()
		}
	})
	tmenu.Add("Run eval").OnClick(func(ctx *application.Context) {
		_, _ = svc.RunEval()
	})
	tmenu.AddSeparator()
	tmenu.Add("Quit").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:quit", nil)
	})
	tray.SetMenu(tmenu)

	gui.Event.On("yoyo:do-quit", func(e *application.CustomEvent) {
		saveWindowState(svc, win)
		gui.Quit()
	})

	notify := func(id, title, body, session string) {
		if ns == nil {
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
			if win != nil {
				win.Show().Focus()
			}
			session := sessionFromNote(result.Response.ID, result.Response.UserInfo)
			gui.Event.Emit("yoyo:focus", session)
		})
	}
	if svc.App != nil {
		ch, _ := svc.App.Hub.Subscribe("*")
		go func() {
			for ev := range ch {
				gui.Event.Emit("yoyo:item", ev)
				switch string(ev.Type) {
				case "turn_end":
					notify("yoyo-turn", "Yoyo", "Turn finished", ev.SessionID)
				case "approval":
					notify("yoyo-ask", "Yoyo approval", payloadStr(ev.Payload, "action"), ev.SessionID)
				case "error":
					notify("yoyo-err", "Yoyo error", payloadStr(ev.Payload, "error"), ev.SessionID)
				}
			}
		}()
	} else if svc.RPC != nil {
		go func() {
			for n := range svc.RPC.Notify {
				if n.Method != "item.event" {
					continue
				}
				gui.Event.Emit("yoyo:item", n.Params)
				var m map[string]any
				_ = json.Unmarshal(n.Params, &m)
				typ, _ := m["type"].(string)
				payload, _ := m["payload"].(map[string]any)
				session, _ := m["session_id"].(string)
				switch typ {
				case "turn_end":
					notify("yoyo-turn", "Yoyo", "Turn finished", session)
				case "approval":
					notify("yoyo-ask", "Yoyo approval", payloadStr(payload, "action"), session)
				case "error":
					notify("yoyo-err", "Yoyo error", payloadStr(payload, "error"), session)
				}
			}
		}()
	}
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
