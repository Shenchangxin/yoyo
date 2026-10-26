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

// WindowState 记下主窗口的位置、大小和是否最大化。FitRev 用来判断要不要按当前工作区重算尺寸。Refit 和 Recenter 不写入 window.json。
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

// windowStatePath 优先用 App.Home.Root 下的 window.json，其次 YOYO_HOME，再次用户主目录下的 .yoyo。主目录也取不到时返回空字符串。
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

// primaryWorkArea 返回主屏幕工作区的宽高。没有工作区尺寸时用屏幕尺寸。Wails 屏幕信息不可用时退回 winsize.PlatformWorkArea。
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

// screenWorkArea 返回窗口所在屏幕的工作区。取不到时退回 primaryWorkArea。
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

// virtualScreen 返回全部屏幕边界的并集，顺序是最小 x、最小 y、宽、高。没有可用屏幕时退回 winsize.PlatformVirtualScreen。
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

// markLost 在保存的窗口落在虚拟屏幕外时，把坐标清零、取消最大化，并标上 Recenter。
func markLost(saved *WindowState) {
	vx, vy, vw, vh := virtualScreen()
	if !winsize.LostOffscreen(saved.X, saved.Y, saved.W, saved.H, vx, vy, vw, vh) {
		return
	}
	saved.X, saved.Y = 0, 0
	saved.Max = false
	saved.Recenter = true
}

// FitWorkArea 按主工作区的 frac 比例算出首次启动的宽高。
func FitWorkArea(frac float64) (w, h int) {
	aw, ah := primaryWorkArea()
	return winsize.Fit(aw, ah, frac)
}

// RestoreGeometry 没有 window.json 时返回按当前工作区拟合的尺寸，并标上 Refit。读到的宽高小于最小值时抬到 MinW 和 MinH。最大化的记录只做 markLost。ShouldRefit 为假时原样返回并 markLost，为真时换成拟合尺寸并标上 Refit。
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

// Raise 把窗口拉回屏幕：取消最小化、显示、聚焦。位置落在 Windows 隐藏哨兵或虚拟桌面外时，先取消最大化再居中。win 为空则直接返回。
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

// ApplyLaunchGeometry 在 WindowRuntimeReady 和 WindowShow 上各登记一次调整，实际只跑一遍。SetSize 不能在 gui.Run 之前调用，否则 Wails 会在空的主线程上 panic。最大化且不需要重新居中时只 Raise。超出工作区或 Refit 时改成拟合尺寸并居中。Recenter 时居中。否则只在窗口丢失在屏幕外时居中。
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

// saveWindowState 把当前位置、大小、是否最大化和当前 FitRev 写成 window.json。窗口为空、最小化、不可见，或坐标是隐藏哨兵时不写。
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

// PersistWindow 在移动和缩放时保存几何，两次写入至少间隔 400 毫秒。正在退出时允许关闭并保存。CloseToTray 或桌面宠物开着时取消关闭、隐藏窗口并 refreshChrome。否则保存后让关闭继续。
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
		if svc.CloseToTray() || svc.CompanionEnabled() {
			e.Cancel()
			win.Hide()
			svc.refreshChrome()
			return
		}
		saveWindowState(svc, win)
	})
}
