package desktop

import (
	"errors"
	"net/url"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Shenchangxin/yoyo/internal/hostopen"
	"github.com/Shenchangxin/yoyo/internal/version"
)

func (s *Service) OpenPath(path string) error {
	if s.RPC != nil {
		_, err := s.call("host.open", map[string]any{"path": path, "kind": "path"})
		return err
	}
	if s.App != nil {
		return s.App.OpenPath(path)
	}
	return hostopen.Path(path)
}

func (s *Service) OpenInEditor(path string) error {
	if s.RPC != nil {
		_, err := s.call("host.open", map[string]any{"path": path, "kind": "editor"})
		return err
	}
	if s.App != nil {
		return s.App.OpenInEditor(path)
	}
	return hostopen.Editor(path)
}

func (s *Service) OpenWorkspaceTerminal(dir string) error {
	if s.RPC != nil {
		_, err := s.call("host.open", map[string]any{"path": dir, "kind": "terminal"})
		return err
	}
	if s.App != nil {
		return s.App.OpenWorkspaceTerminal(dir)
	}
	return hostopen.Terminal(dir)
}

func (s *Service) PopOutThread(id string) error {
	if id == "" {
		return errors.New("missing thread")
	}
	if s.gui == nil {
		return errors.New("no window host")
	}
	opts := application.WebviewWindowOptions{
		Title:            "Yoyo " + version.Version,
		Width:            720,
		Height:           880,
		MinWidth:         480,
		MinHeight:        560,
		BackgroundColour: application.NewRGB(14, 14, 14),
		URL:              "/?popout=" + url.QueryEscape(id) + "#popout=" + url.QueryEscape(id),
		Frameless:        runtime.GOOS != "darwin",
		InitialPosition:  application.WindowCentered,
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
			Icon: AppIcon,
		},
	}
	s.gui.Window.NewWithOptions(opts)
	return nil
}
