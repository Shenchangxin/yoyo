// 按安装包种类收尾：Windows NSIS、打开 DMG 或目录、替换 AppImage。

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

// applyNSIS 只在 Windows 上把安装参数写成 installer 旁边的 .apply.json，再以 runas 拉起当前可执行文件并带上 --yoyo-nsis-apply。路径里有双引号时返回 "update: installer path contains quotes"。非 Windows 返回 "update: nsis apply is windows-only"。
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
	// runas 经 consent.exe 在 GUI 作业外起进程，辅助进程在主程序退出后还在。隐藏的 cmd.exe 加 CreateProcess 提不起机器范围的 NSIS 安装器，而且没有界面。
	return startElevatedApply(exe, specPath)
}

// nsisCmdLine 拼出静默安装命令：安装器路径加引号，再加 /S。destDir 非空时再加 /D=，目录本身不加引号。
func nsisCmdLine(installer, destDir string) string {
	s := `"` + installer + `" /S`
	if destDir != "" {
		s += " /D=" + destDir
	}
	return s
}

// HandleNSISHelper 在参数是 --yoyo-nsis-apply 时跑提权后的等待和安装。参数不匹配返回 false。安装失败时把错误打到标准错误并 os.Exit(1)。成功时返回 true。
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

// runNSISHelper 读 .apply.json，最多等原进程 90 秒，再跑安装器。缺少 installer 字段时返回 "update: apply spec missing installer"。
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

// appendHelperLog 把一行时间戳日志追加到 specPath 加 .log。文件打不开就放弃。
func appendHelperLog(specPath, format string, args ...any) {
	f, err := os.OpenFile(specPath+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

// openPath 在 darwin 上执行 open，在 windows 上执行 cmd.exe /C start，其余执行 xdg-open。
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

// applyAppImage 在 APPIMAGE 为空时打开下载文件所在目录并返回 ErrOpenedPackage。否则把下载件拷到目标加 .new，把正在运行的目标改名为 .old，再把 .new 改成目标名。改名失败时把 .old 改回去。成功后启动新的目标文件。
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
