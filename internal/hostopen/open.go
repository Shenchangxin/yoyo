package hostopen

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func Path(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return os.ErrInvalid
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return openDir(path)
	}
	return launch(path)
}

// URL opens http(s) in the system browser. file:// is handed to Path.
// javascript:, data:, and other schemes are refused.
func URL(raw string) error {
	target, file, err := resolveURL(raw)
	if err != nil {
		return err
	}
	if file {
		return Path(target)
	}
	return launch(target)
}

func Editor(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return os.ErrInvalid
	}
	candidates := []string{os.Getenv("VISUAL"), os.Getenv("EDITOR"), "code", "cursor", "notepad"}
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		bin := c
		if p, err := exec.LookPath(c); err == nil {
			bin = p
		}
		if err := exec.Command(bin, path).Start(); err == nil {
			return nil
		}
	}
	return Path(path)
}

func Terminal(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return os.ErrInvalid
	}
	dir = filepath.Clean(dir)
	switch runtime.GOOS {
	case "windows":
		if p, err := exec.LookPath("wt"); err == nil {
			return exec.Command(p, "-d", dir).Start()
		}
		return exec.Command("cmd", "/c", "start", "cmd.exe", "/k", "cd /d "+dir).Start()
	case "darwin":
		return exec.Command("open", "-a", "Terminal", dir).Start()
	default:
		for _, name := range []string{"x-terminal-emulator", "gnome-terminal", "konsole", "kitty", "alacritty", "xterm"} {
			p, err := exec.LookPath(name)
			if err != nil {
				continue
			}
			switch name {
			case "gnome-terminal":
				return exec.Command(p, "--working-directory="+dir).Start()
			case "kitty", "alacritty":
				return exec.Command(p, "--working-directory", dir).Start()
			default:
				cmd := exec.Command(p)
				cmd.Dir = dir
				return cmd.Start()
			}
		}
		return fmt.Errorf("no system terminal found")
	}
}

func Reveal(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return os.ErrInvalid
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return openDir(path)
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", "/select,", path).Start()
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	default:
		return exec.Command("xdg-open", filepath.Dir(path)).Start()
	}
}

func openDir(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func launch(target string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
	case "darwin":
		return exec.Command("open", target).Start()
	default:
		return exec.Command("xdg-open", target).Start()
	}
}

func resolveURL(raw string) (target string, file bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, os.ErrInvalid
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false, fmt.Errorf("open url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host == "" {
			return "", false, fmt.Errorf("invalid url")
		}
		return u.String(), false, nil
	case "file":
		p, err := fileURLPath(u)
		if err != nil {
			return "", false, err
		}
		return p, true, nil
	case "":
		return "", false, fmt.Errorf("invalid url")
	default:
		return "", false, fmt.Errorf("refusing to open %s URL", u.Scheme)
	}
}

func fileURLPath(u *url.URL) (string, error) {
	raw := u.Path
	if raw == "" {
		raw = u.Opaque
	}
	if raw == "" {
		return "", fmt.Errorf("invalid file url")
	}
	p, err := url.PathUnescape(raw)
	if err != nil {
		p = raw
	}
	if runtime.GOOS == "windows" {
		if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
			return `\\` + u.Host + filepath.FromSlash(strings.TrimPrefix(p, "/")), nil
		}
		p = strings.TrimPrefix(p, "/")
		return filepath.Clean(p), nil
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return filepath.Clean(p), nil
}
