//go:build windows

package computeruse

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

var (
	procSwitchDesktop     = user32.NewProc("SwitchDesktop")
	procSetCursorPos      = user32.NewProc("SetCursorPos")
	procMouseEvent        = user32.NewProc("mouse_event")
	procKeybdEvent        = user32.NewProc("keybd_event")
	procOpenInputDesktop  = user32.NewProc("OpenInputDesktop")
)

func (h *Host) inject(op, app, detail string) (string, error) {
	op = strings.ToLower(strings.TrimSpace(op))
	if op == "" || op == "record" {
		return "", nil
	}
	if h.hwnd != 0 {
		input, _, _ := procOpenInputDesktop.Call(0, 0, 0x01FF)
		_, _, _ = procSwitchDesktop.Call(h.hwnd)
		if input != 0 {
			defer func() {
				_, _, _ = procSwitchDesktop.Call(input)
				_, _, _ = procCloseDesktop.Call(input)
			}()
		}
	}
	switch op {
	case "open", "launch":
		return "", launchOnVirtual(app, detail)
	case "click":
		x, y := parsePoint(detail)
		_, _, _ = procSetCursorPos.Call(uintptr(x), uintptr(y))
		_, _, _ = procMouseEvent.Call(0x0002, 0, 0, 0, 0)
		_, _, _ = procMouseEvent.Call(0x0004, 0, 0, 0, 0)
		return "", nil
	case "type":
		return "", typeUnicode(detail)
	case "key":
		return "", keyTap(detail)
	case "screenshot", "shot":
		if h.hwnd == 0 {
			return "", fmt.Errorf("computer_use: no virtual desktop; refusing primary screen")
		}
		path := filepath.Join(h.dir, "last.png")
		if err := captureDesktop(path); err != nil {
			return "", err
		}
		return path, nil
	default:
		return "", fmt.Errorf("computer_use: unknown op %q", op)
	}
}

func (h *Host) Capture(path string) error {
	h.ensureVirtual()
	h.mu.Lock()
	desk := h.hwnd
	h.mu.Unlock()
	if desk == 0 {
		return fmt.Errorf("computer_use: no virtual desktop; refusing primary screen")
	}
	input, _, _ := procOpenInputDesktop.Call(0, 0, 0x01FF)
	r, _, err := procSwitchDesktop.Call(desk)
	if r == 0 {
		if input != 0 {
			_, _, _ = procCloseDesktop.Call(input)
		}
		return fmt.Errorf("computer_use: cannot switch to virtual desktop (%v); refusing primary screen", err)
	}
	defer func() {
		if input != 0 {
			_, _, _ = procSwitchDesktop.Call(input)
			_, _, _ = procCloseDesktop.Call(input)
		}
	}()
	return captureDesktop(path)
}

func launchOnVirtual(app, detail string) error {
	bin := app
	if !strings.ContainsAny(bin, `/\`) {
		if p, err := exec.LookPath(app + ".exe"); err == nil {
			bin = p
		}
	}
	cmd := exec.Command(bin, strings.Fields(detail)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

func parsePoint(detail string) (int, int) {
	fields := strings.FieldsFunc(detail, func(r rune) bool {
		return r == ',' || r == ' ' || r == 'x'
	})
	if len(fields) >= 2 {
		x, _ := strconv.Atoi(fields[0])
		y, _ := strconv.Atoi(fields[1])
		return x, y
	}
	return 40, 40
}

func typeUnicode(text string) error {
	for _, r := range text {
		if r == '\n' {
			_, _, _ = procKeybdEvent.Call(0x0D, 0, 0, 0)
			_, _, _ = procKeybdEvent.Call(0x0D, 0, 2, 0)
			continue
		}
		vk := uintptr(r)
		_, _, _ = procKeybdEvent.Call(vk, 0, 0, 0)
		_, _, _ = procKeybdEvent.Call(vk, 0, 2, 0)
	}
	return nil
}

func keyTap(name string) error {
	vk := uintptr(0x0D)
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "esc", "escape":
		vk = 0x1B
	case "tab":
		vk = 0x09
	case "enter", "return":
		vk = 0x0D
	case "space":
		vk = 0x20
	}
	_, _, _ = procKeybdEvent.Call(vk, 0, 0, 0)
	_, _, _ = procKeybdEvent.Call(vk, 0, 2, 0)
	return nil
}

func captureDesktop(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	escaped := strings.ReplaceAll(path, `'`, `''`)
	ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $b=[System.Windows.Forms.Screen]::PrimaryScreen.Bounds; $bmp=New-Object System.Drawing.Bitmap $b.Width,$b.Height; $g=[System.Drawing.Graphics]::FromImage($bmp); $g.CopyFromScreen($b.Location,[System.Drawing.Point]::Empty,$b.Size); $bmp.Save('%s');`, escaped)
	return exec.Command("powershell", "-NoProfile", "-Command", ps).Run()
}
