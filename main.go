// main 是窗口进程，用 Wails 打开主窗口，单实例 ID 是 com.yoyo.workstation。
// YOYO_WORKER=1 时不创建窗口，把 stdin 和 stdout 交给 JSON-RPC，stdout 是协议。
// YOYO_ISOLATE=1 时再启动同一可执行文件并带 YOYO_WORKER=1。健康检查超过 8 秒或失败则退回同进程。这不是安全沙箱，工作者仍握有 vault 和工具。
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/Shenchangxin/yoyo/internal/api"
	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/desktop"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/home"
	"github.com/Shenchangxin/yoyo/internal/idol"
	"github.com/Shenchangxin/yoyo/internal/remote"
	"github.com/Shenchangxin/yoyo/internal/skin"
	"github.com/Shenchangxin/yoyo/internal/update"
	"github.com/Shenchangxin/yoyo/internal/version"
	"github.com/Shenchangxin/yoyo/internal/video"
)

//go:embed all:frontend/dist
var assets embed.FS

// fail 在 err 非空时把诊断写到 stderr，并以状态 1 退出。err 为空则直接返回。
func fail(err error) {
	if err == nil {
		return
	}
	diaglog.Error("fatal", "component", "boot", "err", err)
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// attachRemoteUI 先找磁盘上的 frontend/dist：当前目录，以及可执行文件旁边的 frontend/dist。index.html 是文件就交给 node.UseStatic。两处都没有时，用编译进二进制的 frontend/dist。
func attachRemoteUI(node *remote.Node) {
	if node == nil {
		return
	}
	// 手机端在磁盘上已经有前端文件时直接读它们，所以前端重新构建后，进程一重启就能到微信。打包的程序没有这份目录时，退回编译进二进制的 frontend/dist。
	for _, dir := range []string{"frontend/dist", filepath.Join(filepath.Dir(executablePath()), "frontend", "dist")} {
		if st, err := os.Stat(filepath.Join(dir, "index.html")); err == nil && !st.IsDir() {
			node.UseStatic(http.FileServer(http.Dir(dir)))
			return
		}
	}
	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		return
	}
	node.UseStatic(http.FileServer(http.FS(sub)))
}

// executablePath 返回 os.Executable 的路径。取失败时返回 "."。
func executablePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return exe
}

// main 先让 update.HandleNSISHelper 处理安装器辅助进程，接管则直接返回。然后设置 YOYO_DESKTOP=1。
// YOYO_WORKER=1 时不创建窗口：打开 App，remote.Bind 成功就用 attachRemoteUI 挂上静态页，再把 stdin 和 stdout 交给 api.ServeRPC。stdout 是协议。
// 否则 openDesktop 打开桌面服务，创建名为 Yoyo 的 Wails 应用和主窗口。单实例 ID 是 com.yoyo.workstation，第二实例启动时 desktop.Raise 已有窗口。
func main() {
	if update.HandleNSISHelper() {
		return
	}
	_ = os.Setenv("YOYO_DESKTOP", "1")
	evals := resolveEvals()
	home := os.Getenv("YOYO_HOME")
	if os.Getenv("YOYO_WORKER") == "1" {
		core, err := app.Open(home, evals)
		if err != nil {
			fail(err)
		}
		defer core.Close()
		if node, err := remote.Bind(core); err == nil && node != nil {
			attachRemoteUI(node)
			defer node.Close()
		}
		if err := api.ServeRPC(context.Background(), core, os.Stdin, os.Stdout); err != nil {
			fail(err)
		}
		return
	}

	lg, _ := diaglog.Boot(diaglog.ProcessGUI)
	svc, core, err := openDesktop(home, evals)
	if err != nil {
		fail(err)
	}
	if core != nil {
		defer core.Close()
		if node, err := remote.Bind(core); err == nil && node != nil {
			attachRemoteUI(node)
			defer node.Close()
		}
	}
	defer svc.Close()

	var win application.Window
	ns := notifications.New()
	var wailsLog *slog.Logger
	if lg != nil {
		wailsLog = lg.WailsLogger()
	} else if core != nil && core.Log != nil {
		wailsLog = core.Log.WailsLogger()
	}
	gui := application.New(application.Options{
		Name:                        "Yoyo",
		Description:                 "Self-harnessing local agent workstation",
		DisableDefaultSignalHandler: true,
		Logger:                      wailsLog,
		Services: []application.Service{
			application.NewService(svc),
			application.NewService(ns),
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: skinAssetMiddleware(core, home),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.yoyo.workstation",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				desktop.Raise(win)
			},
		},
	})

	geom := desktop.RestoreGeometry(svc)
	cfg := svc.GetConfig()
	opts := application.WebviewWindowOptions{
		Title:              "Yoyo " + version.Version,
		Width:              geom.W,
		Height:             geom.H,
		MinWidth:           1080,
		MinHeight:          680,
		BackgroundColour:   application.NewRGB(28, 29, 31),
		URL:                "/",
		Frameless:          runtime.GOOS != "darwin",
		AlwaysOnTop:        cfg.AlwaysOnTop,
		InitialPosition:    application.WindowCentered,
		UseApplicationMenu: runtime.GOOS == "darwin",
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
		},
		Windows: application.WindowsWindow{
			Theme:                             application.SystemDefault,
			DisableMenu:                       true,
			DisableFramelessWindowDecorations: false,
			NonClientRegionSupport:            true,
		},
		Linux: application.LinuxWindow{
			Icon: desktop.AppIcon,
		},
	}
	if geom.Max && !geom.Recenter {
		opts.StartState = application.WindowStateMaximised
	} else if !geom.Refit && !geom.Recenter && (geom.X != 0 || geom.Y != 0) {
		opts.InitialPosition = application.WindowXY
		opts.X = geom.X
		opts.Y = geom.Y
	}
	win = gui.Window.NewWithOptions(opts)
	desktop.ApplyLaunchGeometry(win, geom)
	svc.Attach(gui, win)
	desktop.InstallChrome(gui, win, svc, ns)
	desktop.PersistWindow(svc, win)
	desktop.WatchSignals(gui, svc, win)
	svc.PrepareCompanion()

	if err := gui.Run(); err != nil {
		fail(err)
	}
}

// openDesktop 在 YOYO_ISOLATE=1 时调用 desktop.OpenIsolated，由它再启动同一可执行文件并带 YOYO_WORKER=1。否则在本进程 app.Open，并用 desktop.NewService 包住同一个 App。
func openDesktop(home, evals string) (*desktop.Service, *app.App, error) {
	if os.Getenv("YOYO_ISOLATE") == "1" {
		return desktop.OpenIsolated(home, evals)
	}
	core, err := app.Open(home, evals)
	if err != nil {
		return nil, nil, err
	}
	return desktop.NewService(core), core, nil
}

// skinAssetMiddleware 返回套在静态资源上的中间件：视频媒体、角色立绘、偶像立牌，最里面是皮肤。core 为空时按 homeRoot 打开 home，取皮肤库和 idols/stands。
func skinAssetMiddleware(core *app.App, homeRoot string) func(http.Handler) http.Handler {
	var store *skin.Store
	if core != nil && core.Skins != nil {
		store = core.Skins
	} else {
		d, err := home.Open(homeRoot)
		if err == nil {
			store = skin.NewStore(d.Skins())
		}
	}
	standDir := ""
	if core != nil && core.Home != nil {
		standDir = filepath.Join(core.Home.Idols(), "stands")
	} else if d, err := home.Open(homeRoot); err == nil {
		standDir = filepath.Join(d.Idols(), "stands")
	}
	return func(next http.Handler) http.Handler {
		var eng *video.Engine
		if core != nil {
			eng = core.Video
		}
		return video.MediaAssetMiddleware(eng, video.CharacterCastMiddleware(
			video.CharacterCastRoot(homeRoot),
			idol.StandMiddleware(standDir, skin.Middleware(store, next)),
		))
	}
}

// resolveEvals 优先返回 YOYO_EVALS。否则依次看当前目录和可执行文件目录下的 evals 子目录，都不是目录时返回相对路径 evals。
func resolveEvals() string {
	if v := os.Getenv("YOYO_EVALS"); v != "" {
		return v
	}
	evals := "evals"
	if wd, err := os.Getwd(); err == nil {
		if p := filepath.Join(wd, "evals"); dirExists(p) {
			evals = p
		}
	}
	if exe, err := os.Executable(); err == nil {
		if p := filepath.Join(filepath.Dir(exe), "evals"); dirExists(p) {
			evals = p
		}
	}
	return evals
}

// dirExists 在路径存在且是目录时返回真。
func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
