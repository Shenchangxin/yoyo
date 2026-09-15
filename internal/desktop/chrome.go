package desktop

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/icons"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// InstallChrome adds native menus, a tray icon, and desktop notifications.
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
	file.AddSeparator()
	file.Add("Quit").SetAccelerator("CmdOrCtrl+Q").OnClick(func(ctx *application.Context) {
		gui.Quit()
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
		gui.Quit()
	})
	tray.SetMenu(tmenu)

	notify := func(title, body string) {
		if ns == nil {
			return
		}
		_ = ns.SendNotification(notifications.NotificationOptions{
			ID:    "yoyo-" + title,
			Title: title,
			Body:  body,
		})
	}
	if ns != nil {
		_, _ = ns.RequestNotificationAuthorization()
	}
	if svc.App != nil {
		ch, _ := svc.App.Hub.Subscribe("*")
		go func() {
			for ev := range ch {
				gui.Event.Emit("yoyo:item", ev)
				switch string(ev.Type) {
				case "turn_end":
					notify("Yoyo", "Turn finished")
				case "approval":
					notify("Yoyo approval", payloadStr(ev.Payload, "action"))
				case "error":
					notify("Yoyo error", payloadStr(ev.Payload, "error"))
				}
			}
		}()
	} else if svc.RPC != nil {
		go func() {
			for n := range svc.RPC.Notify {
				if n.Method == "item.event" {
					gui.Event.Emit("yoyo:item", n.Params)
					notify("Yoyo", "agent event")
				}
			}
		}()
	}
}

func payloadStr(p map[string]any, key string) string {
	if p == nil {
		return ""
	}
	s, _ := p[key].(string)
	return s
}
