package desktop

import (
	"errors"
	"runtime"
	"strconv"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/winsize"
)

func companionOrigin() (x, y int) {
	aw, ah := primaryWorkArea()
	x = aw - companionW - 28
	y = ah - companionH - 56
	return clampCompanion(x, y)
}

func companionWorkBoxes() []hitBox {
	gui := application.Get()
	if gui == nil {
		return nil
	}
	var out []hitBox
	for _, scr := range gui.Screen.GetAll() {
		if scr == nil {
			continue
		}
		work := scr.WorkArea
		if work.Width <= 0 || work.Height <= 0 {
			work = scr.Bounds
		}
		if work.Width <= 0 || work.Height <= 0 {
			continue
		}
		out = append(out, hitBox{work.X, work.Y, work.Width, work.Height})
	}
	return out
}

func clampCompanion(x, y int) (int, int) {
	vx, vy, vw, vh := virtualScreen()
	return clampCompanionTo(x, y, companionW, companionH, companionPad, companionWorkBoxes(), vx, vy, vw, vh)
}

func companionCursorLocal(win application.Window) (lx, ly, ww, wh float64, ok bool) {
	if lx, ly, ww, wh, ok = companionCursorLocalNative(win); ok {
		return lx, ly, ww, wh, true
	}
	cx, cy, ok := desktopCursorCSS()
	if !ok || win == nil {
		return 0, 0, 0, 0, false
	}
	wx, wy := win.Position()
	wi, hi := win.Size()
	return cx - float64(wx), cy - float64(wy), float64(wi), float64(hi), true
}

func (s *Service) SetCompanionCaption(on bool) {
	s.companionCaption.Store(on)
}

func (s *Service) PlaceCompanion(x, y int) {
	win := s.companion
	if win == nil {
		return
	}
	if winsize.HiddenSentinel(x, y) {
		return
	}
	nx, ny := clampCompanion(x, y)
	wx, wy := win.Position()
	if nx == wx && ny == wy {
		return
	}
	if !s.companionClamping.CompareAndSwap(false, true) {
		return
	}
	defer s.companionClamping.Store(false)
	win.SetPosition(nx, ny)
}

func (s *Service) pinCompanion(win application.Window) {
	if win == nil || !s.companionClamping.CompareAndSwap(false, true) {
		return
	}
	defer s.companionClamping.Store(false)
	x, y := win.Position()
	if winsize.HiddenSentinel(x, y) {
		return
	}
	nx, ny := clampCompanion(x, y)
	if nx != x || ny != y {
		win.SetPosition(nx, ny)
	}
}

func (s *Service) companionShown() bool {
	return s.companion != nil && !s.companionDismissed.Load()
}

func (s *Service) syncCompanionMenu() {
	item := s.companionMenu
	if item == nil {
		return
	}
	n := nativeCopy(s.menuLocale)
	if s.companionShown() {
		item.SetLabel(n.HideCompanion)
		return
	}
	item.SetLabel(n.ShowCompanion)
}

func (s *Service) ToggleCompanion() {
	if s.companionShown() {
		s.HideCompanion()
		return
	}
	_ = s.ShowCompanion()
}

func (s *Service) syncCompanion(cfg app.Config) {
	if !cfg.CompanionEnabled {
		s.companionDismissed.Store(true)
		s.CloseCompanion()
		s.emitCompanion("hidden")
		return
	}
	if s.companionDismissed.Load() {
		s.CloseCompanion()
		return
	}
	_ = s.OpenCompanion()
}

func (s *Service) PrepareCompanion() {
	s.syncCompanion(s.GetConfig())
}

func (s *Service) CloseCompanion() {
	win := s.companion
	s.companion = nil
	if win != nil {
		s.companionClosing.Store(true)
		win.Close()
		s.companionClosing.Store(false)
	}
	s.syncCompanionMenu()
}

func (s *Service) DismissCompanion() {
	s.companionDismissed.Store(true)
	s.CloseCompanion()
	s.emitCompanion("hidden")
}

func (s *Service) HideCompanion() {
	s.DismissCompanion()
}

func (s *Service) ShowCompanion() error {
	s.companionDismissed.Store(false)
	if !s.GetConfig().CompanionEnabled {
		return nil
	}
	return s.OpenCompanion()
}

func (s *Service) emitCompanion(action string) {
	if s.gui == nil {
		return
	}
	s.gui.Event.Emit("yoyo:companion", action)
	s.syncCompanionMenu()
}

func (s *Service) OpenCompanion() error {
	if s.gui == nil {
		return errors.New("no window host")
	}
	s.companionDismissed.Store(false)
	s.startCursorWatch()
	if s.companion != nil {
		s.companion.Show()
		s.companion.SetAlwaysOnTop(true)
		hideCompanionTaskbar(s.companion)
		applyCompanionShape(s.companion, false)
		setCompanionPassthrough(s.companion, true)
		s.pinCompanion(s.companion)
		s.emitCompanion("shown")
		return nil
	}
	x, y := companionOrigin()
	opts := application.WebviewWindowOptions{
		Name:                  "yoyo-companion",
		Title:                 "Yoyo",
		Width:                 companionW,
		Height:                companionH,
		MinWidth:              companionW,
		MinHeight:             companionH,
		MaxWidth:              companionW,
		MaxHeight:             companionH,
		DisableResize:         true,
		Frameless:             true,
		AlwaysOnTop:           true,
		BackgroundType:        application.BackgroundTypeTransparent,
		BackgroundColour:      application.NewRGBA(0, 0, 0, 0),
		URL:                   "/?surface=companion",
		InitialPosition:       application.WindowXY,
		X:                     x,
		Y:                     y,
		MinimiseButtonState:   application.ButtonHidden,
		MaximiseButtonState:   application.ButtonHidden,
		CloseButtonState:      application.ButtonHidden,
		FullscreenButtonState: application.ButtonHidden,
		Mac: application.MacWindow{
			TitleBar:    application.MacTitleBarHiddenInset,
			WindowClass: application.MacWindowClassPanel,
			PanelPreferences: application.MacPanelPreferences{
				FloatingPanel: true,
				UtilityWindow: true,
			},
		},
		Windows: application.WindowsWindow{
			Theme:                             application.SystemDefault,
			DisableMenu:                       true,
			DisableFramelessWindowDecorations: true,
			NonClientRegionSupport:            true,
			HiddenOnTaskbar:                   true,
			WindowDidMoveDebounceMS:           16,
		},
		Linux: application.LinuxWindow{
			Icon: AppIcon,
		},
	}
	win := s.gui.Window.NewWithOptions(opts)
	s.companion = win
	s.companionPlaced.Store(false)
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if s.companion == win {
			s.companion = nil
		}
		if s.Quitting() || s.companionClosing.Load() {
			return
		}
		s.companionDismissed.Store(true)
		s.emitCompanion("hidden")
	})
	win.OnWindowEvent(events.Common.WindowDidMove, func(*application.WindowEvent) {
		s.pinCompanion(win)
	})
	if runtime.GOOS == "windows" {
		win.OnWindowEvent(events.Windows.WindowEndMove, func(*application.WindowEvent) {
			s.pinCompanion(win)
		})
	}
	win.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		if s.companionPlaced.CompareAndSwap(false, true) {
			px, py := companionOrigin()
			win.SetSize(companionW, companionH)
			win.SetPosition(px, py)
			win.SetAlwaysOnTop(true)
		}
		if !s.GetConfig().CompanionEnabled || s.companionDismissed.Load() {
			s.CloseCompanion()
			return
		}
		win.Show()
		win.SetAlwaysOnTop(true)
		hideCompanionTaskbar(win)
		applyCompanionShape(win, false)
		setCompanionPassthrough(win, true)
		s.pinCompanion(win)
		s.emitCompanion("shown")
	})
	return nil
}

func (s *Service) startCursorWatch() {
	if !s.cursorWatch.CompareAndSwap(false, true) {
		return
	}
	go func() {
		tick := time.NewTicker(8 * time.Millisecond)
		defer tick.Stop()
		var lastX, lastY float64
		var grabX, grabY float64
		var armed, dragging, downPrev bool
		passthrough := true
		var idlePins uint8
		for range tick.C {
			if s.quitting.Load() {
				return
			}
			win := s.companion
			if win == nil || !win.IsVisible() {
				armed, dragging, downPrev = false, false, false
				continue
			}
			caption := s.companionCaption.Load()
			lx, ly, ww, wh, locOK := companionCursorLocal(win)
			hit := locOK && CompanionHit(lx, ly, ww, wh, caption)
			down := leftMouseDown()
			if locOK {
				want := !(hit || dragging || (armed && down))
				if want != passthrough {
					passthrough = want
					setCompanionPassthrough(win, want)
				}
			}
			cx, cy, curOK := desktopCursorCSS()
			if down && !downPrev && hit && curOK {
				wx, wy := win.Position()
				armed = true
				grabX = cx - float64(wx)
				grabY = cy - float64(wy)
			}
			if !down {
				armed, dragging = false, false
			} else if armed && curOK {
				wx, wy := win.Position()
				nx := cx - grabX
				ny := cy - grabY
				if !dragging && (absF(nx-float64(wx)) >= 4 || absF(ny-float64(wy)) >= 4) {
					dragging = true
				}
				if dragging {
					ix, iy := clampCompanion(int(nx+0.5), int(ny+0.5))
					if ix != wx || iy != wy {
						win.SetPosition(ix, iy)
					}
				}
			}
			downPrev = down
			if !dragging {
				idlePins++
				if armed || idlePins >= 4 {
					idlePins = 0
					s.pinCompanion(win)
				}
			} else {
				idlePins = 0
			}
			if !curOK {
				continue
			}
			if absF(cx-lastX) < 0.25 && absF(cy-lastY) < 0.25 {
				continue
			}
			lastX, lastY = cx, cy
			wx, wy := win.Position()
			win.ExecJS("window.__yoyoCursor&&window.__yoyoCursor(" +
				strconv.FormatFloat(cx, 'f', 1, 64) + "," +
				strconv.FormatFloat(cy, 'f', 1, 64) + "," +
				strconv.Itoa(wx) + "," +
				strconv.Itoa(wy) + ")")
		}
	}()
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
