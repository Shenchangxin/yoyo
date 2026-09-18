package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/Shenchangxin/yoyo/internal/winsize"
)

type WindowState struct {
	X        int  `json:"x"`
	Y        int  `json:"y"`
	W        int  `json:"w"`
	H        int  `json:"h"`
	Max      bool `json:"max"`
	FitRev   int  `json:"fit_rev,omitempty"`
	Refit    bool `json:"-"`
	Recenter bool `json:"-"`
}

func windowStatePath(svc *Service) string {
	if svc != nil && svc.App != nil && svc.App.Home != nil {
		return filepath.Join(svc.App.Home.Root, "window.json")
	}
	if h := os.Getenv("YOYO_HOME"); h != "" {
		return filepath.Join(h, "window.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".yoyo", "window.json")
}

func primaryWorkArea() (int, int) {
	if gui := application.Get(); gui != nil {
		scr := gui.Screen.GetPrimary()
		if scr == nil {
			if all := gui.Screen.GetAll(); len(all) > 0 {
				scr = all[0]
			}
		}
		if scr != nil {
			w, h := scr.WorkArea.Width, scr.WorkArea.Height
			if w <= 0 || h <= 0 {
				w, h = scr.Size.Width, scr.Size.Height
			}
			if w > 0 && h > 0 {
				return w, h
			}
		}
	}
	return winsize.PlatformWorkArea()
}

func screenWorkArea(win application.Window) (int, int) {
	if win != nil {
		if scr, err := win.GetScreen(); err == nil && scr != nil {
			w, h := scr.WorkArea.Width, scr.WorkArea.Height
			if w <= 0 || h <= 0 {
				w, h = scr.Size.Width, scr.Size.Height
			}
			if w > 0 && h > 0 {
				return w, h
			}
		}
	}
	return primaryWorkArea()
}

func virtualScreen() (int, int, int, int) {
	if gui := application.Get(); gui != nil {
		all := gui.Screen.GetAll()
		var minX, minY, maxX, maxY int
		n := 0
		for _, scr := range all {
			if scr == nil {
				continue
			}
			b := scr.Bounds
			if b.Width <= 0 || b.Height <= 0 {
				b.X, b.Y = scr.X, scr.Y
				b.Width, b.Height = scr.Size.Width, scr.Size.Height
			}
			if b.Width <= 0 || b.Height <= 0 {
				continue
			}
			if n == 0 {
				minX, minY = b.X, b.Y
				maxX, maxY = b.X+b.Width, b.Y+b.Height
			} else {
				if b.X < minX {
					minX = b.X
				}
				if b.Y < minY {
					minY = b.Y
				}
				if r := b.X + b.Width; r > maxX {
					maxX = r
				}
				if bottom := b.Y + b.Height; bottom > maxY {
					maxY = bottom
				}
			}
			n++
		}
		if n > 0 {
			return minX, minY, maxX - minX, maxY - minY
		}
	}
	return winsize.PlatformVirtualScreen()
}

func markLost(saved *WindowState) {
	vx, vy, vw, vh := virtualScreen()
	if !winsize.LostOffscreen(saved.X, saved.Y, saved.W, saved.H, vx, vy, vw, vh) {
		return
	}
	saved.X, saved.Y = 0, 0
	saved.Max = false
	saved.Recenter = true
}

// FitWorkArea returns a first-launch size as a fraction of the primary work area.
func FitWorkArea(frac float64) (w, h int) {
	aw, ah := primaryWorkArea()
	return winsize.Fit(aw, ah, frac)
}

func RestoreGeometry(svc *Service) WindowState {
	aw, ah := primaryWorkArea()
	fittedW, fittedH := winsize.Fit(aw, ah, winsize.DefaultFrac)
	fresh := WindowState{W: fittedW, H: fittedH, FitRev: winsize.CurrentFitRev, Refit: true}
	p := windowStatePath(svc)
	if p == "" {
		return fresh
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return fresh
	}
	saved := WindowState{}
	if json.Unmarshal(b, &saved) != nil {
		return fresh
	}
	if saved.W < winsize.MinW {
		saved.W = winsize.MinW
	}
	if saved.H < winsize.MinH {
		saved.H = winsize.MinH
	}
	if saved.Max {
		saved.FitRev = winsize.CurrentFitRev
		markLost(&saved)
		return saved
	}
	if !winsize.ShouldRefit(saved.W, saved.H, saved.FitRev, fittedW, fittedH, aw, ah) {
		markLost(&saved)
		return saved
	}
	saved.W, saved.H = fittedW, fittedH
	saved.FitRev = winsize.CurrentFitRev
	saved.Refit = true
	return saved
}

// Raise brings the window on-screen: unminimise, show, focus, and center if it
// was restored at the Windows hide sentinel or off the virtual desktop.
func Raise(win application.Window) {
	if win == nil {
		return
	}
	if win.IsMinimised() {
		win.UnMinimise()
	}
	win.Show()
	x, y := win.Position()
	w, h := win.Width(), win.Height()
	vx, vy, vw, vh := virtualScreen()
	if winsize.LostOffscreen(x, y, w, h, vx, vy, vw, vh) {
		if win.IsMaximised() {
			win.UnMaximise()
		}
		win.Center()
	}
	win.Focus()
}

// ApplyLaunchGeometry registers a one-shot resize after the window runtime is
// up. SetSize must not run before gui.Run — Wails panics on a nil main thread.
func ApplyLaunchGeometry(win application.Window, saved WindowState) {
	if win == nil {
		return
	}
	var once sync.Once
	apply := func() {
		once.Do(func() {
			if saved.Max && !saved.Recenter {
				Raise(win)
				return
			}
			aw, ah := screenWorkArea(win)
			fw, fh := winsize.Fit(aw, ah, winsize.DefaultFrac)
			cw, ch := win.Width(), win.Height()
			if winsize.Overflows(cw, ch, aw, ah) || saved.Refit {
				if cw != fw || ch != fh {
					win.SetSize(fw, fh)
				}
				win.Center()
			} else if saved.Recenter {
				win.Center()
			} else {
				x, y := win.Position()
				vx, vy, vw, vh := virtualScreen()
				if winsize.LostOffscreen(x, y, cw, ch, vx, vy, vw, vh) {
					win.Center()
				}
			}
			Raise(win)
		})
	}
	win.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) { apply() })
	win.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) { apply() })
}

func saveWindowState(svc *Service, win application.Window) {
	if win == nil {
		return
	}
	if win.IsMinimised() || !win.IsVisible() {
		return
	}
	p := windowStatePath(svc)
	if p == "" {
		return
	}
	x, y := win.Position()
	if winsize.HiddenSentinel(x, y) {
		return
	}
	st := WindowState{
		X:      x,
		Y:      y,
		W:      win.Width(),
		H:      win.Height(),
		Max:    win.IsMaximised(),
		FitRev: winsize.CurrentFitRev,
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, b, 0o644)
}

func PersistWindow(svc *Service, win application.Window) {
	var last time.Time
	win.OnWindowEvent(events.Common.WindowDidMove, func(e *application.WindowEvent) {
		if time.Since(last) < 400*time.Millisecond {
			return
		}
		last = time.Now()
		saveWindowState(svc, win)
	})
	win.OnWindowEvent(events.Common.WindowDidResize, func(e *application.WindowEvent) {
		if time.Since(last) < 400*time.Millisecond {
			return
		}
		last = time.Now()
		saveWindowState(svc, win)
	})
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if svc.Quitting() {
			saveWindowState(svc, win)
			return
		}
		if svc.CloseToTray() {
			e.Cancel()
			win.Hide()
			return
		}
		saveWindowState(svc, win)
	})
}
