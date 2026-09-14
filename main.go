package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/desktop"
	"github.com/Shenchangxin/yoyo/internal/version"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
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
	core, err := app.Open(os.Getenv("YOYO_HOME"), evals)
	if err != nil {
		log.Fatal(err)
	}

	svc := desktop.NewService(core)
	gui := application.New(application.Options{
		Name:        "Yoyo",
		Description: "Self-harnessing local agent workstation",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	gui.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Yoyo " + version.Version,
		Width:            1280,
		Height:           800,
		BackgroundColour: application.NewRGB(16, 17, 21),
		URL:              "/",
	})

	if err := gui.Run(); err != nil {
		log.Fatal(err)
	}
	_ = core.Close()
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
