// Package osutil 读写系统剪贴板，并把主屏幕截到文件。
// Windows 上的剪贴板和截图会调用 HideConsole。HideConsole 与 NoConsole 按平台分文件实现。
package osutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ClipboardRead 读取系统剪贴板文本。
// Windows 执行 powershell -NoProfile -Command Get-Clipboard，并先调用 HideConsole，返回值去掉末尾的 \r 和 \n。
// Darwin 执行 pbpaste，不去掉末尾空白。其他系统在 PATH 上有 wl-paste 时用它，否则用 xclip -selection clipboard -o。
func ClipboardRead() (string, error) {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("powershell", "-NoProfile", "-Command", "Get-Clipboard")
		HideConsole(cmd)
		out, err := cmd.Output()
		return strings.TrimRight(string(out), "\r\n"), err
	case "darwin":
		out, err := exec.Command("pbpaste").Output()
		return string(out), err
	default:
		if _, err := exec.LookPath("wl-paste"); err == nil {
			out, err := exec.Command("wl-paste").Output()
			return string(out), err
		}
		out, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output()
		return string(out), err
	}
}

// ClipboardWrite 把 text 写入系统剪贴板。
// Windows 执行 powershell -NoProfile -Command Set-Clipboard -Value $input，标准输入是 text，并先调用 HideConsole。
// Darwin 执行 pbcopy。其他系统在 PATH 上有 wl-copy 时用它且不另加参数，否则用 xclip -selection clipboard。文本都从标准输入送入。
func ClipboardWrite(text string) error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("powershell", "-NoProfile", "-Command", "Set-Clipboard -Value $input")
		cmd.Stdin = strings.NewReader(text)
		HideConsole(cmd)
		return cmd.Run()
	case "darwin":
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	default:
		bin := "xclip"
		args := []string{"-selection", "clipboard"}
		if _, err := exec.LookPath("wl-copy"); err == nil {
			bin, args = "wl-copy", nil
		}
		cmd := exec.Command(bin, args...)
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
}

// Screenshot 把主屏幕截到 path。先创建父目录，权限 0o755。
// Windows 用 powershell 抓主屏幕并保存，路径里的单引号写成两个单引号，并先调用 HideConsole。
// Darwin 执行 screencapture -x。其他系统在 PATH 上有 gnome-screenshot 时用它的 -f，否则执行 import -window root。
func Screenshot(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $b=[System.Windows.Forms.Screen]::PrimaryScreen.Bounds; $bmp=New-Object System.Drawing.Bitmap $b.Width,$b.Height; $g=[System.Drawing.Graphics]::FromImage($bmp); $g.CopyFromScreen($b.Location,[System.Drawing.Point]::Empty,$b.Size); $bmp.Save('%s');`, strings.ReplaceAll(path, `'`, `''`))
		cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
		HideConsole(cmd)
		return cmd.Run()
	case "darwin":
		return exec.Command("screencapture", "-x", path).Run()
	default:
		if _, err := exec.LookPath("gnome-screenshot"); err == nil {
			return exec.Command("gnome-screenshot", "-f", path).Run()
		}
		return exec.Command("import", "-window", "root", path).Run()
	}
}
