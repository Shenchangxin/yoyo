// Package hostopen 用系统程序打开路径、网页、编辑器和终端，或在文件管理器里定位文件。
// URL 只接受 http、https 和 file。javascript、data 以及其他协议会被拒绝。
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

// Path 打开一个本地路径。空白返回 os.ErrInvalid。路径先 Clean。
// Stat 成功且是目录时走 openDir，否则走 launch。Stat 失败时仍把路径交给 launch。
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

// URL 用系统浏览器打开 http 或 https。file 交给 Path。
// javascript、data 以及其他协议被拒绝。
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

// Editor 用编辑器打开 path。空白返回 os.ErrInvalid。
// 依次尝试 VISUAL、EDITOR、code、cursor、notepad。LookPath 成功就用那个路径，Start 成功即返回。
// 全部失败时改走 Path。
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

// Terminal 在 dir 打开系统终端。空白返回 os.ErrInvalid，然后 Clean。
// Windows 先找 wt，参数是 -d 和目录；找不到就执行 cmd /c start cmd.exe /k cd /d 接上目录。
// Darwin 执行 open -a Terminal。其他系统依次找 x-terminal-emulator、gnome-terminal、konsole、kitty、alacritty、xterm。
// gnome-terminal 用 --working-directory= 拼目录。kitty 和 alacritty 用 --working-directory 和目录两个参数。其余把 Dir 设成该目录。
// 一个都找不到时返回 no system terminal found。
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

// Reveal 在文件管理器里显示 path。空白返回 os.ErrInvalid。Stat 失败时返回该错误。
// 目录走 openDir。Windows 对文件执行 explorer /select,。Darwin 执行 open -R。其他系统对父目录执行 xdg-open。
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

// openDir 打开目录。Windows 用 explorer，Darwin 用 open，其他系统用 xdg-open。
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

// launch 打开文件或 http(s) 地址。Windows 用 rundll32 url.dll,FileProtocolHandler，Darwin 用 open，其他系统用 xdg-open。
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

// resolveURL 解析原始字符串。空白返回 os.ErrInvalid。解析失败时包上 open url。
// http 和 https 要求 Host 非空，file 为假。file 交给 fileURLPath，file 为真。没有协议时返回 invalid url。
// 其他协议返回 refusing to open，并带上协议名。
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

// fileURLPath 从 file URL 取出路径。Path 为空时用 Opaque。两者都空时返回 invalid file url。
// PathUnescape 失败就用原文。Windows 上 Host 非空且不是 localhost 时拼成 \\Host 加路径；否则去掉开头的斜杠再 Clean。
// 其他系统保证以 / 开头再 Clean。
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
