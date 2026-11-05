package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Shenchangxin/yoyo/internal/hostopen"
	"github.com/Shenchangxin/yoyo/internal/version"
)

// OpenPath 在 RPC 非空时调用 host.open，kind 为 path。否则有 App 时走 App.OpenPath，再否则 hostopen.Path。
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

// RevealPath 在文件管理器里定位 path。RPC 走 host.open，kind 为 reveal。
func (s *Service) RevealPath(path string) error {
	if s.RPC != nil {
		_, err := s.call("host.open", map[string]any{"path": path, "kind": "reveal"})
		return err
	}
	if s.App != nil {
		return s.App.RevealPath(path)
	}
	return hostopen.Reveal(path)
}

// OpenURL 在 RPC 非空时调用 host.open，kind 为 url。否则有 App 时走 App.OpenURL，再否则 hostopen.URL。
// raw 在 RPC 非空时作为 host.open 的 path。有 App 时传给 App.OpenURL。否则传给 hostopen.URL。
func (s *Service) OpenURL(raw string) error {
	if s.RPC != nil {
		_, err := s.call("host.open", map[string]any{"path": raw, "kind": "url"})
		return err
	}
	if s.App != nil {
		return s.App.OpenURL(raw)
	}
	return hostopen.URL(raw)
}

// OpenInEditor 在 RPC 非空时调用 host.open，kind 为 editor。否则有 App 时走 App.OpenInEditor，再否则 hostopen.Editor。
// path 在 RPC 非空时作为 host.open 的 path。有 App 时传给 App.OpenInEditor。否则传给 hostopen.Editor。
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

// OpenWorkspaceTerminal 在 RPC 非空时调用 host.open，kind 为 terminal。否则有 App 时走 App.OpenWorkspaceTerminal，再否则 hostopen.Terminal。
// dir 在 RPC 非空时作为 host.open 的 path。有 App 时传给 App.OpenWorkspaceTerminal。否则传给 hostopen.Terminal。
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

// OpenFilePreview 为这个文件再开一个独立窗口。主窗口不被挡住，可以继续操作。
// path 为空返回 missing path。没有 gui 返回 no window host。
func (s *Service) OpenFilePreview(workspace, path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("missing path")
	}
	if s.gui == nil {
		return errors.New("no window host")
	}
	q := url.Values{}
	q.Set("filepreview", "1")
	q.Set("workspace", strings.TrimSpace(workspace))
	q.Set("path", path)
	raw := q.Encode()
	title := path
	if i := strings.LastIndexAny(path, `\/`); i >= 0 && i < len(path)-1 {
		title = path[i+1:]
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(workspace) + "\n" + path))
	name := "file-preview-" + hex.EncodeToString(sum[:8])
	if existing, ok := s.gui.Window.GetByName(name); ok {
		existing.Show()
		existing.Focus()
		return nil
	}
	opts := application.WebviewWindowOptions{
		Name:             name,
		Title:            title,
		Width:            1100,
		Height:           820,
		MinWidth:         640,
		MinHeight:        480,
		BackgroundColour: application.NewRGB(246, 246, 244),
		URL:              "/?" + raw + "#filepreview=1&" + raw,
		Frameless:        false,
		InitialPosition:  application.WindowCentered,
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarDefault,
		},
		Windows: application.WindowsWindow{
			Theme:       application.SystemDefault,
			DisableMenu: true,
		},
		Linux: application.LinuxWindow{
			Icon: AppIcon,
		},
	}
	s.gui.Window.NewWithOptions(opts)
	return nil
}

// PopOutThread 为这条会话再开一个窗口，地址是 /?popout= 加上转义后的 id，哈希同样带 popout。darwin 保留系统标题栏，其他系统无边框。id 为空返回 missing thread，没有 gui 返回 no window host。
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
		BackgroundColour: application.NewRGB(28, 29, 31),
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
