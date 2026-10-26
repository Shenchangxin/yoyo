package desktop

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// WatchSignals 监听 Ctrl+C 和 SIGTERM。收到后停止监听并 forceQuit。Wails 默认只调用 Quit，关到托盘会把那次退出取消掉，所以这里自己收尾。
// gui 传给 forceQuit。
// svc 传给 forceQuit。
// win 传给 forceQuit。
func WatchSignals(gui *application.App, svc *Service, win application.Window) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		signal.Stop(ch)
		forceQuit(gui, svc, win)
	}()
}

// forceQuit 先 BeginQuit，保存窗口几何，销毁托盘，再 gui.Quit。1200 毫秒后若进程还在，os.Exit(0)。
// svc 非 nil 时调用 BeginQuit。和 win 一起传给 saveWindowState。tray 非 nil 时销毁托盘。
// win 和 svc 一起传给 saveWindowState。
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
