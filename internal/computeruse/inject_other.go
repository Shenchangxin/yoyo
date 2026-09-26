//go:build !windows

package computeruse

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func (h *Host) inject(op, app, detail string) (string, error) {
	op = strings.ToLower(strings.TrimSpace(op))
	if op == "" || op == "record" {
		return "", nil
	}
	display := strings.TrimSpace(os.Getenv("YOYO_CU_DISPLAY"))
	if display == "" {
		return "", fmt.Errorf("computer_use: set YOYO_CU_DISPLAY to a virtual display; refusing the operator screen")
	}
	switch op {
	case "open", "launch":
		cmd := exec.Command(app, strings.Fields(detail)...)
		cmd.Env = append(os.Environ(), "DISPLAY="+display)
		return "", cmd.Start()
	case "click", "type", "key":
		if _, err := exec.LookPath("xdotool"); err != nil {
			return "", fmt.Errorf("computer_use: xdotool required on %s", runtime.GOOS)
		}
		args := []string{}
		switch op {
		case "click":
			args = []string{"click", "1"}
		case "type":
			args = []string{"type", "--", detail}
		case "key":
			args = []string{"key", detail}
		}
		cmd := exec.Command("xdotool", args...)
		cmd.Env = append(os.Environ(), "DISPLAY="+display)
		return "", cmd.Run()
	case "screenshot", "shot":
		path := filepath.Join(h.dir, "last.png")
		return path, h.Capture(path)
	default:
		return "", fmt.Errorf("computer_use: unknown op %q", op)
	}
}

func (h *Host) Capture(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	display := strings.TrimSpace(os.Getenv("YOYO_CU_DISPLAY"))
	if display == "" {
		return fmt.Errorf("computer_use: set YOYO_CU_DISPLAY to a virtual display; refusing the operator screen")
	}
	if _, err := exec.LookPath("import"); err == nil {
		cmd := exec.Command("import", "-window", "root", path)
		cmd.Env = append(os.Environ(), "DISPLAY="+display)
		return cmd.Run()
	}
	return fmt.Errorf("computer_use: no capture tool on virtual display %s", display)
}
