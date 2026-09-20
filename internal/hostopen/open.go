package hostopen

import (
	"fmt"
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
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
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
