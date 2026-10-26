//go:build windows

// Windows 上的键鼠、启动和抓屏。有虚拟桌面句柄时先 SwitchDesktop，截图在没有句柄时拒绝。

package computeruse

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

var (
	// procSwitchDesktop 是 user32.SwitchDesktop。
	procSwitchDesktop = user32.NewProc("SwitchDesktop")
	// procSetCursorPos 是 user32.SetCursorPos。
	procSetCursorPos = user32.NewProc("SetCursorPos")
	// procMouseEvent 是 user32.mouse_event。
	procMouseEvent = user32.NewProc("mouse_event")
	// procKeybdEvent 是 user32.keybd_event。
	procKeybdEvent = user32.NewProc("keybd_event")
	// procOpenInputDesktop 是 user32.OpenInputDesktop。
	procOpenInputDesktop = user32.NewProc("OpenInputDesktop")
)

// inject 对 open 和 launch 调用 launchOnVirtual。click、type、key 直接调 user32，不检查句柄是否为 0。screenshot 在句柄为 0 时返回错误，不抓主屏。句柄非 0 时先切到该桌面，结束再切回。
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

// Capture 在桌面句柄为 0 时返回错误，不抓屏。句柄非 0 时切到该桌面再 captureDesktop，失败则报告不能切换并拒绝主屏。
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

// launchOnVirtual 启动可执行文件。名字不含斜杠时尝试查找同名 .exe。没有把桌面名字传给进程。
func launchOnVirtual(app, detail string) error {
	bin := app
	if !strings.ContainsAny(bin, `/\`) {
		if p, err := exec.LookPath(app + ".exe"); err == nil {
			bin = p
		}
	}
	cmd := exec.Command(bin, strings.Fields(detail)...)
	osutil.HideConsole(cmd)
	return cmd.Start()
}

// parsePoint 从明细里解析 x、y。不足两个数时返回 40, 40。
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

// typeUnicode 把每个码元交给 keybd_event。换行发送回车的按下和抬起。
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

// keyTap 发送 esc、tab、enter 或 space。其他名字发送回车。
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

// captureDesktop 用 PowerShell 按 PrimaryScreen 的范围 CopyFromScreen，写到 path。调用方负责是否已经切换桌面。
func captureDesktop(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	escaped := strings.ReplaceAll(path, `'`, `''`)
	ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $b=[System.Windows.Forms.Screen]::PrimaryScreen.Bounds; $bmp=New-Object System.Drawing.Bitmap $b.Width,$b.Height; $g=[System.Drawing.Graphics]::FromImage($bmp); $g.CopyFromScreen($b.Location,[System.Drawing.Point]::Empty,$b.Size); $bmp.Save('%s');`, escaped)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	osutil.HideConsole(cmd)
	return cmd.Run()
}
