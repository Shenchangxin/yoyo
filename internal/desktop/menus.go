package desktop

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const companionContextMenuID = "yoyo-companion"

func (s *Service) RebuildMenus() {
	// Do not use application.InvokeSync here: InstallChrome runs before gui.Run,
	// when App.impl is still nil and dispatchOnMainThread panics.
	s.chromeMu.Lock()
	defer s.chromeMu.Unlock()
	s.rebuildAppMenu()
	s.rebuildTrayAndCompanion()
}

func (s *Service) refreshChrome() {
	s.chromeMu.Lock()
	defer s.chromeMu.Unlock()
	s.rebuildTrayAndCompanion()
}

func (s *Service) showMainWindow() {
	showWindow(s.win)
	s.refreshChrome()
}

func (s *Service) strings() nativeStrings {
	locale := s.menuLocale
	if locale == "" {
		locale = s.GetConfig().Locale
	}
	return nativeCopy(locale)
}

func (s *Service) rebuildAppMenu() {
	gui := s.gui
	win := s.win
	if gui == nil {
		return
	}
	n := s.strings()

	menu := gui.NewMenu()
	if runtime.GOOS == "darwin" {
		menu.AddRole(application.AppMenu)
		// WKWebView has no clipboard shortcuts without this menu. WebView2 and
		// GTK already paste in inputs; Wails EditMenu also runs insertText/paste,
		// so Ctrl+V inserts the clipboard twice.
		menu.AddRole(application.EditMenu)
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
				s.refreshChrome()
				return
			}
			win.Close()
		}
	})
	file.Add(n.Quit).SetAccelerator("CmdOrCtrl+Q").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:quit", nil)
	})

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
	thread.Add(n.Stop).SetAccelerator("CmdOrCtrl+.").OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:command", "stop")
	})
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
}

func (s *Service) rebuildTrayAndCompanion() {
	s.applyTrayTooltip()
	s.rebuildTrayMenu()
	s.rebuildCompanionMenu()
}

func (s *Service) applyTrayTooltip() {
	if s.tray == nil {
		return
	}
	n := s.strings()
	st := s.chromeStatus()
	s.tray.SetTooltip(n.trayTooltip(st, trayWakeHint()))
	if runtime.GOOS == "darwin" {
		s.tray.SetLabel("Yoyo")
	}
}

func (s *Service) rebuildTrayMenu() {
	gui := s.gui
	win := s.win
	if gui == nil || s.tray == nil {
		return
	}
	n := s.strings()
	tmenu := gui.NewMenu()
	st := s.chromeStatus()
	tmenu.Add(n.chromeLine(st)).SetEnabled(false)

	tmenu.AddSeparator()
	if win != nil && win.IsVisible() {
		tmenu.Add(n.Hide).OnClick(func(ctx *application.Context) {
			win.Hide()
			s.refreshChrome()
		})
	} else {
		tmenu.Add(n.Show).OnClick(func(ctx *application.Context) {
			s.showMainWindow()
		})
	}
	if s.CompanionEnabled() {
		tmenu.AddCheckbox(n.Companion, s.companionShown()).OnClick(func(ctx *application.Context) {
			if ctx.ClickedMenuItem().Checked() {
				_ = s.ShowCompanion()
				return
			}
			s.HideCompanion()
		})
	}

	tmenu.AddSeparator()
	stop := tmenu.Add(n.Stop).OnClick(func(ctx *application.Context) {
		_ = s.Interrupt("")
	})
	if !s.agentWorking() {
		stop.SetEnabled(false)
	}
	tmenu.Add(n.NewChat).OnClick(func(ctx *application.Context) {
		s.showMainWindow()
		gui.Event.Emit("yoyo:command", "new")
	})
	if s.pendingApprovalCount() > 0 {
		tmenu.Add(n.ReviewApproval).OnClick(func(ctx *application.Context) {
			s.OpenCompanionChat()
		})
	}

	tmenu.AddSeparator()
	tmenu.Add(n.SettingsEllipsis).OnClick(func(ctx *application.Context) {
		s.showMainWindow()
		gui.Event.Emit("yoyo:command", "control")
	})
	tmenu.AddSeparator()
	tmenu.Add(n.Quit).OnClick(func(ctx *application.Context) {
		gui.Event.Emit("yoyo:quit", nil)
	})
	s.tray.SetMenu(tmenu)
	tmenu.Update()
}

func (s *Service) rebuildCompanionMenu() {
	gui := s.gui
	if gui == nil {
		return
	}
	if old, ok := gui.ContextMenu.Get(companionContextMenuID); ok && old != nil {
		old.Menu.Destroy()
	}
	gui.ContextMenu.Remove(companionContextMenuID)
	n := s.strings()
	menu := gui.ContextMenu.New()
	menu.Add(n.OpenChat).OnClick(func(ctx *application.Context) {
		s.OpenCompanionChat()
	})
	if s.companionCall.Load() {
		menu.Add(n.HangUp).OnClick(func(ctx *application.Context) {
			s.SetCompanionCall(false)
		})
		menu.AddSeparator()
	}
	stop := menu.Add(n.Stop).OnClick(func(ctx *application.Context) {
		_ = s.Interrupt("")
	})
	if !s.agentWorking() {
		stop.SetEnabled(false)
	}
	menu.AddSeparator()
	menu.Add(n.ResetPosition).OnClick(func(ctx *application.Context) {
		s.ResetCompanion()
	})
	menu.Add(n.HideCompanion).OnClick(func(ctx *application.Context) {
		s.HideCompanion()
	})
	gui.ContextMenu.Add(companionContextMenuID, menu)
	menu.Menu.Update()
}

func (s *Service) noteDesktopPulse(kind, session string) {
	if session != "" {
		s.lastPulseSession.Store(session)
	}
	switch kind {
	case "loading":
		was := s.agentBusy.Swap(true)
		if !was {
			s.refreshChrome()
			return
		}
		s.chromeMu.Lock()
		s.applyTrayTooltip()
		s.chromeMu.Unlock()
	case "approval":
		s.agentBusy.Store(false)
		s.refreshChrome()
	case "done", "error", "eval", "evolve", "update":
		s.agentBusy.Store(false)
		s.refreshChrome()
	}
}
