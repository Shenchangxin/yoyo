package desktop

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// companionContextMenuID 是 yoyo-companion，前端用这个 id 认宠物右键菜单。
const companionContextMenuID = "yoyo-companion"

// RebuildMenus 在锁内重建应用菜单、托盘和宠物菜单。
func (s *Service) RebuildMenus() {
	// 这里不能用 application.InvokeSync。InstallChrome 在 gui.Run 之前运行，那时 App.impl 还是 nil，dispatchOnMainThread 会 panic。
	s.chromeMu.Lock()
	defer s.chromeMu.Unlock()
	s.rebuildAppMenu()
	s.rebuildTrayAndCompanion()
}

// refreshChrome 在锁内重建托盘提示、托盘菜单和宠物菜单，不重建应用菜单。
func (s *Service) refreshChrome() {
	s.chromeMu.Lock()
	defer s.chromeMu.Unlock()
	s.rebuildTrayAndCompanion()
}

// showMainWindow 显示主窗口并 refreshChrome。
func (s *Service) showMainWindow() {
	showWindow(s.win)
	s.refreshChrome()
}

// strings 按 menuLocale 取文案。menuLocale 为空时用配置里的 locale。
func (s *Service) strings() nativeStrings {
	locale := s.menuLocale
	if locale == "" {
		locale = s.GetConfig().Locale
	}
	return nativeCopy(locale)
}

// rebuildAppMenu 重建应用菜单。darwin 上加应用菜单和编辑菜单。文件菜单发出 yoyo:sessions、yoyo:workspace 和 yoyo:quit。查看菜单的 yoyo:command 是 sidebar、review、dock、harness、video、palette、control。会话菜单是 stop、compact、export、rename、fork、archive。Harness 菜单是 harness、run-eval、run-evolve。帮助菜单发出 yoyo:about、yoyo:doctor 和 yoyo:logs。关窗口在 CloseToTray 或宠物开着时改为隐藏。
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
		// 没有这份菜单时 WKWebView 没有剪贴板快捷键。WebView2 和 GTK 在输入框里已经能粘贴；Wails 的 EditMenu 还会再执行 insertText 或 paste，Ctrl+V 会把剪贴板插入两遍。
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

// rebuildTrayAndCompanion 依次更新托盘提示、托盘菜单和宠物右键菜单。
func (s *Service) rebuildTrayAndCompanion() {
	s.applyTrayTooltip()
	s.rebuildTrayMenu()
	s.rebuildCompanionMenu()
}

// applyTrayTooltip 按当前托盘状态和是否显示唤醒提示设置工具提示。darwin 上标签固定为 Yoyo。
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

// rebuildTrayMenu 按窗口是否可见放显示或隐藏，按配置放宠物开关。停止项直接 Interrupt 空会话，没有正在跑的回合时禁用。新对话发出 yoyo:command 且值为 new。有待审批时放审阅项并打开宠物对话。设置发出 yoyo:command 且值为 control。退出发出 yoyo:quit。
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

// rebuildCompanionMenu 重建 id 为 yoyo-companion 的右键菜单：打开对话、通话中才有的挂断、停止、重置位置、隐藏宠物。停止在没有正在跑的回合时禁用。
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

// noteDesktopPulse 记下非空的会话 id。loading 把 agentBusy 设成真，第一次时重建菜单，已经在忙则只改托盘提示。approval、done、error、eval、evolve、update 把 agentBusy 设成假并重建菜单。
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
