package desktop

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// WatchSignals makes Ctrl+C / SIGTERM actually leave the process.
// Wails' default handler only calls Quit(), and Close-to-tray cancels that.
func WatchSignals(gui *application.App, svc *Service, win application.Window) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		signal.Stop(ch)
		forceQuit(gui, svc, win)
	}()
}

func forceQuit(gui *application.App, svc *Service, win application.Window) {
	if svc != nil {
		svc.BeginQuit()
	}
	saveWindowState(svc, win)
	if svc != nil && svc.tray != nil {
		svc.tray.Destroy()
		svc.tray = nil
	}
	if gui != nil {
		gui.Quit()
	}
	time.AfterFunc(1200*time.Millisecond, func() {
		os.Exit(0)
	})
}
