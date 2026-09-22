package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/Shenchangxin/yoyo/internal/api"
	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/desktop"
	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/version"
)

//go:embed all:frontend/dist
var assets embed.FS

func fail(err error) {
	if err == nil {
		return
	}
	diaglog.Error("fatal", "component", "boot", "err", err)
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	evals := resolveEvals()
	home := os.Getenv("YOYO_HOME")
	if os.Getenv("YOYO_WORKER") == "1" {
		core, err := app.Open(home, evals)
		if err != nil {
			fail(err)
		}
		defer core.Close()
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
			Handler: application.AssetFileServerFS(assets),
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

	if err := gui.Run(); err != nil {
		fail(err)
	}
}

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

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
