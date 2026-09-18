package osutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func ClipboardRead() (string, error) {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("powershell", "-NoProfile", "-Command", "Get-Clipboard").Output()
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

func ClipboardWrite(text string) error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("powershell", "-NoProfile", "-Command", "Set-Clipboard -Value $input")
		cmd.Stdin = strings.NewReader(text)
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

func Screenshot(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $b=[System.Windows.Forms.Screen]::PrimaryScreen.Bounds; $bmp=New-Object System.Drawing.Bitmap $b.Width,$b.Height; $g=[System.Drawing.Graphics]::FromImage($bmp); $g.CopyFromScreen($b.Location,[System.Drawing.Point]::Empty,$b.Size); $bmp.Save('%s');`, strings.ReplaceAll(path, `'`, `''`))
		return exec.Command("powershell", "-NoProfile", "-Command", ps).Run()
	case "darwin":
		return exec.Command("screencapture", "-x", path).Run()
	default:
		if _, err := exec.LookPath("gnome-screenshot"); err == nil {
			return exec.Command("gnome-screenshot", "-f", path).Run()
		}
		return exec.Command("import", "-window", "root", path).Run()
	}
}
