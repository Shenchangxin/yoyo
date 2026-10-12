package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func applyNSIS(installer, destDir string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("update: nsis apply is windows-only")
	}
	installer, err := filepath.Abs(installer)
	if err != nil {
		return err
	}
	if destDir == "" {
		destDir = installDir("")
	}
	// /D= must be the last NSIS argument and is not quoted.
	inner := `"` + installer + `" /S`
	if destDir != "" {
		inner += ` /D=` + destDir
	}
	script := fmt.Sprintf("ping -n 3 127.0.0.1 >nul & %s", inner)
	cmd := exec.Command("cmd.exe", "/C", script)
	hideWindow(cmd)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func openPath(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd.exe", "/C", "start", "", path)
		hideWindow(cmd)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

func applyAppImage(downloaded, _ string) error {
	target := os.Getenv("APPIMAGE")
	if target == "" {
		dir := filepath.Dir(downloaded)
		if err := openPath(dir); err != nil {
			return err
		}
		return ErrOpenedPackage
	}
	target, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	downloaded, err = filepath.Abs(downloaded)
	if err != nil {
		return err
	}
	tmp := target + ".new"
	if err := copyFile(downloaded, tmp, 0o755); err != nil {
		return err
	}
	_ = os.Chmod(tmp, 0o755)
	backup := target + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Rename(backup, target)
		return err
	}
	_ = os.Chmod(target, 0o755)
	cmd := exec.Command(target)
	cmd.Dir = filepath.Dir(target)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}
