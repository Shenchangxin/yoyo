package update

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const nsisApplyArg = "--yoyo-nsis-apply"

type nsisApplySpec struct {
	Installer string `json:"installer"`
	DestDir   string `json:"dest,omitempty"`
	WaitPID   int    `json:"wait_pid"`
}

func applyNSIS(installer, destDir string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("update: nsis apply is windows-only")
	}
	installer, err := filepath.Abs(installer)
	if err != nil {
		return err
	}
	destDir = strings.TrimRight(destDir, `/\`)
	if strings.ContainsAny(installer, `"`) || strings.ContainsAny(destDir, `"`) {
		return fmt.Errorf("update: installer path contains quotes")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	specPath := installer + ".apply.json"
	body, err := json.Marshal(nsisApplySpec{Installer: installer, DestDir: destDir, WaitPID: os.Getpid()})
	if err != nil {
		return err
	}
	if err := os.WriteFile(specPath, body, 0o600); err != nil {
		return err
	}
	// runas starts a process outside the GUI job via consent.exe, so the
	// helper survives quit. Hidden cmd.exe + CreateProcess cannot elevate
	// the machine-scope NSIS installer and fails with no UI.
	return startElevatedApply(exe, specPath)
}

func nsisCmdLine(installer, destDir string) string {
	s := `"` + installer + `" /S`
	if destDir != "" {
		s += " /D=" + destDir
	}
	return s
}

// HandleNSISHelper runs the elevated waiter/installer when this process was
// started with --yoyo-nsis-apply. Call it before creating any GUI.
func HandleNSISHelper() bool {
	if len(os.Args) < 3 || os.Args[1] != nsisApplyArg {
		return false
	}
	if err := runNSISHelper(os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return true
}

func runNSISHelper(specPath string) error {
	appendHelperLog(specPath, "helper start spec=%s pid=%d", specPath, os.Getpid())
	b, err := os.ReadFile(specPath)
	if err != nil {
		return err
	}
	var spec nsisApplySpec
	if err := json.Unmarshal(b, &spec); err != nil {
		return err
	}
	if spec.Installer == "" {
		return fmt.Errorf("update: apply spec missing installer")
	}
	appendHelperLog(specPath, "wait pid=%d installer=%s dest=%s", spec.WaitPID, spec.Installer, spec.DestDir)
	waitForPID(spec.WaitPID, 90*time.Second)
	appendHelperLog(specPath, "parent gone, running installer")
	err = runNSISInstaller(spec.Installer, spec.DestDir)
	if err != nil {
		appendHelperLog(specPath, "installer error: %v", err)
		return err
	}
	appendHelperLog(specPath, "installer finished")
	return nil
}

func appendHelperLog(specPath, format string, args ...any) {
	f, err := os.OpenFile(specPath+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
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
