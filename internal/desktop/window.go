package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type WindowState struct {
	X   int  `json:"x"`
	Y   int  `json:"y"`
	W   int  `json:"w"`
	H   int  `json:"h"`
	Max bool `json:"max"`
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

func RestoreGeometry(svc *Service) WindowState {
	st := WindowState{W: 1440, H: 900}
	p := windowStatePath(svc)
	if p == "" {
		return st
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return st
	}
	_ = json.Unmarshal(b, &st)
	if st.W < 1080 {
		st.W = 1080
	}
	if st.H < 680 {
		st.H = 680
	}
	return st
}

func saveWindowState(svc *Service, win application.Window) {
	if win == nil {
		return
	}
	p := windowStatePath(svc)
	if p == "" {
		return
	}
	x, y := win.Position()
	st := WindowState{
		X:   x,
		Y:   y,
		W:   win.Width(),
		H:   win.Height(),
		Max: win.IsMaximised(),
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, b, 0o644)
}

func PersistWindow(svc *Service, win application.Window) {
	saveWindowState(svc, win)
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
		if svc.CloseToTray() {
			e.Cancel()
			win.Hide()
			return
		}
		saveWindowState(svc, win)
	})
}
