package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/Shenchangxin/yoyo/internal/api"
	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/desktop"
	"github.com/Shenchangxin/yoyo/internal/version"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	evals := resolveEvals()
	home := os.Getenv("YOYO_HOME")
	if os.Getenv("YOYO_WORKER") == "1" {
		core, err := app.Open(home, evals)
		if err != nil {
			log.Fatal(err)
		}
		defer core.Close()
		if err := api.ServeRPC(context.Background(), core, os.Stdin, os.Stdout); err != nil {
			log.Fatal(err)
		}
		return
	}

	svc, core, err := openDesktop(home, evals)
	if err != nil {
		log.Fatal(err)
	}
	if core != nil {
		defer core.Close()
	}
	defer svc.Close()

	ns := notifications.New()
	gui := application.New(application.Options{
		Name:        "Yoyo",
		Description: "Self-harnessing local agent workstation",
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
	})

	win := gui.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:              "Yoyo " + version.Version,
		Width:              1440,
		Height:             900,
		MinWidth:           1080,
		MinHeight:          680,
		BackgroundColour:   application.NewRGB(14, 14, 14),
		URL:                "/",
		Frameless:          runtime.GOOS != "darwin",
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
	})
	desktop.InstallChrome(gui, win, svc, ns)

	if err := gui.Run(); err != nil {
		log.Fatal(err)
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
