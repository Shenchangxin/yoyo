// Package update 检查签名清单、下载安装包，并用暂存文件替换正在运行的可执行文件。ApplyStaged 先把该文件改名为带 .old 后缀的路径，再拷贝暂存文件；拷贝失败时把 .old 改回原名。
package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	// ErrQuitNeeded 表示安装器已经拉起，当前进程应当退出，安装才能做完。
	ErrQuitNeeded = errors.New("update: quit to finish install")
	// ErrOpenedPackage 表示已经为用户打开了 DMG 或目录，剩下的步骤由用户完成。
	ErrOpenedPackage = errors.New("update: opened package for the user")
)

// Pending 是待安装的暂存包，由 MarkPending 写到目录下的 pending.json。
type Pending struct {
	Staging  string `json:"staging"`
	SHA256   string `json:"sha256,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Version  string `json:"version,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// pendingPath 返回 dir/pending.json。
func pendingPath(dir string) string {
	return filepath.Join(dir, "pending.json")
}

// MarkPending 创建目录并把 Pending 写成 pending.json，权限 0o644。
// dir 以 0o755 创建，并交给 pendingPath。
func MarkPending(dir string, p Pending) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pendingPath(dir), b, 0o644)
}

// ReadPending 读取 pending.json。
// dir 交给 pendingPath。
func ReadPending(dir string) (Pending, error) {
	var p Pending
	b, err := os.ReadFile(pendingPath(dir))
	if err != nil {
		return p, err
	}
	return p, json.Unmarshal(b, &p)
}

// ClearPending 删除 pending.json。文件本来就不存在时返回 nil。
// dir 交给 pendingPath。
func ClearPending(dir string) error {
	err := os.Remove(pendingPath(dir))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// StagingName 返回暂存文件名 yoyo.staging。
func StagingName() string { return "yoyo.staging" }

// ApplyStaged 用暂存文件替换 exePath 上正在运行的文件。exePath 为空时改用 os.Executable。两条路径先取绝对路径，相同则返回 "update: refusing to apply staging onto itself"。暂存文件不存在时返回带 "update: staging:" 的错误，大小为 0 时返回 "update: empty staging file"。接着删除已有的 exePath+".old"（删除错误被忽略），再把正在运行的文件改名为该 .old。改名失败时原文件还在，但原先的 .old 已经删掉，返回 "update: cannot move current binary aside: ..."。改名成功后把暂存文件拷到原路径，模式 0o755。拷贝失败时把 .old 改回原路径（改回的错误被忽略），并返回 "update: copy staging: ..."。runtime.GOOS 不是 windows 时，再对新文件 Chmod 0o755。
// stagingPath 取绝对路径，和 exePath 相同则返回错误；不存在或大小为 0 也返回错误，否则以 0o755 拷到 exePath。
func ApplyStaged(exePath, stagingPath string) error {
	if exePath == "" {
		var err error
		exePath, err = os.Executable()
		if err != nil {
			return err
		}
	}
	exePath, err := filepath.Abs(exePath)
	if err != nil {
		return err
	}
	stagingPath, err = filepath.Abs(stagingPath)
	if err != nil {
		return err
	}
	// 两条绝对路径相同就拒绝。不能把暂存文件应用到它自己身上。
	if exePath == stagingPath {
		return fmt.Errorf("update: refusing to apply staging onto itself")
	}
	st, err := os.Stat(stagingPath)
	if err != nil {
		return fmt.Errorf("update: staging: %w", err)
	}
	if st.Size() == 0 {
		return fmt.Errorf("update: empty staging file")
	}
	backup := exePath + ".old"
	_ = os.Remove(backup)
	// 先把正在运行的文件改名为 .old。失败时原文件还在，但已有的 .old 已经删掉。
	if err := os.Rename(exePath, backup); err != nil {
		return fmt.Errorf("update: cannot move current binary aside: %w", err)
	}
	if err := copyFile(stagingPath, exePath, 0o755); err != nil {
		// 拷贝失败就把 .old 改回原路径。改回的错误被忽略，调用方看到的是拷贝错误。
		_ = os.Rename(backup, exePath)
		return fmt.Errorf("update: copy staging: %w", err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(exePath, 0o755)
	}
	return nil
}

// ApplyDownloaded 按 pending.json 安装已下载的包。没有这份文件时，对 dir/yoyo.staging 调用 ApplyStaged。kind 为空则按文件名推断。nsis 和 exe 走 applyNSIS，成功后返回 ErrQuitNeeded。dmg 打开文件并返回 ErrOpenedPackage。appimage 走 applyAppImage：没有 APPIMAGE 环境变量时打开所在目录并返回 ErrOpenedPackage，替换成功后返回 ErrQuitNeeded。zip、tar.gz、tgz、cli 先抽出可执行文件再 ApplyStaged。其余 kind 直接对暂存路径 ApplyStaged。
// exePath 在没有 pending.json 时和暂存路径一起交给 ApplyStaged；nsis 和 exe 时交给 nsisDestDir；zip 一类抽出后再交给 ApplyStaged。
func ApplyDownloaded(exePath, dir string) error {
	p, err := ReadPending(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return ApplyStaged(exePath, filepath.Join(dir, StagingName()))
		}
		return err
	}
	path := p.Staging
	if path == "" {
		path = filepath.Join(dir, StagingName())
	}
	kind := strings.ToLower(strings.TrimSpace(p.Kind))
	if kind == "" {
		kind = kindFromName(path)
	}
	switch kind {
	case "nsis", "exe":
		if err := applyNSIS(path, nsisDestDir(exePath)); err != nil {
			return err
		}
		return ErrQuitNeeded
	case "dmg":
		if err := openPath(path); err != nil {
			return err
		}
		return ErrOpenedPackage
	case "appimage":
		if err := applyAppImage(path, exePath); err != nil {
			return err
		}
		return ErrQuitNeeded
	case "zip", "tar.gz", "tgz", "cli":
		bin, err := extractCLIBinary(path, dir)
		if err != nil {
			return err
		}
		return ApplyStaged(exePath, bin)
	default:
		return ApplyStaged(exePath, path)
	}
}

// kindFromName 按小写文件名推断种类：以 -setup.exe 或 installer.exe 结尾是 nsis，.dmg、.appimage、.zip、.tar.gz、.tgz 各归其类，其余是 bin。
// path 转成小写后看后缀。
func kindFromName(path string) string {
	n := strings.ToLower(path)
	switch {
	case strings.HasSuffix(n, "-setup.exe"), strings.HasSuffix(n, "installer.exe"):
		return "nsis"
	case strings.HasSuffix(n, ".dmg"):
		return "dmg"
	case strings.HasSuffix(n, ".appimage"):
		return "appimage"
	case strings.HasSuffix(n, ".zip"):
		return "zip"
	case strings.HasSuffix(n, ".tar.gz"), strings.HasSuffix(n, ".tgz"):
		return "tar.gz"
	default:
		return "bin"
	}
}

// installDir 返回可执行文件的目录。exePath 为空时用 os.Executable。都失败则返回空串。
func installDir(exePath string) string {
	if exePath == "" {
		exePath, _ = os.Executable()
	}
	if exePath != "" {
		if abs, err := filepath.Abs(exePath); err == nil {
			return filepath.Dir(abs)
		}
	}
	return ""
}

// nsisDestDir 是交给 NSIS 的 /D= 目录，已去掉末尾斜杠。skipNSISDestDir 为真时返回空串，安装器就用它自己编译进去的 InstallDir。
// exePath 交给 installDir，再去掉末尾斜杠。
func nsisDestDir(exePath string) string {
	dir := strings.TrimRight(installDir(exePath), `/\`)
	if skipNSISDestDir(dir) {
		return ""
	}
	return dir
}

// skipNSISDestDir 在目录为空、路径中含分隔符加 go-build，或目录名是 bin 且上一级有 go.mod 或 Taskfile.yml 时返回 true。
// dir 为空时返回 true；路径里含分隔符加 go-build，或目录名是 bin 且上一级有 go.mod 或 Taskfile.yml 时也返回 true。
func skipNSISDestDir(dir string) bool {
	if dir == "" {
		return true
	}
	cleaned := filepath.Clean(dir)
	lower := strings.ToLower(cleaned)
	if strings.Contains(lower, string(filepath.Separator)+"go-build") {
		return true
	}
	if !strings.EqualFold(filepath.Base(cleaned), "bin") {
		return false
	}
	parent := filepath.Dir(cleaned)
	for _, name := range []string{"go.mod", "Taskfile.yml"} {
		if _, err := os.Stat(filepath.Join(parent, name)); err == nil {
			return true
		}
	}
	return false
}

// extractCLIBinary 从 zip 或 tar.gz 抽出第一个可执行文件到 destDir/yoyo.staging。其他后缀原样返回归档路径。
// archive 转成小写后看后缀，是 zip 或 tar.gz、tgz 时抽出，否则原样返回。
func extractCLIBinary(archive, destDir string) (string, error) {
	lower := strings.ToLower(archive)
	out := filepath.Join(destDir, StagingName())
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return out, unzipFirstBinary(archive, out)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return out, untarFirstBinary(archive, out)
	default:
		return archive, nil
	}
}

// unzipFirstBinary 优先抽出名为 yoyo 或 yoyo.exe 的文件，没有则用遇到的第一个非目录项。压缩包里没有文件时返回 "update: zip has no files"。
// src 交给 zip.OpenReader。
// dest 和模式 0o755 一起交给 writeNewFile。
func unzipFirstBinary(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	var best *zip.File
	for i := range r.File {
		f := r.File[i]
		name := filepath.Base(strings.ReplaceAll(f.Name, "\\", "/"))
		if f.FileInfo().IsDir() {
			continue
		}
		if name == "yoyo" || name == "yoyo.exe" {
			best = f
			break
		}
		if best == nil {
			best = f
		}
	}
	if best == nil {
		return fmt.Errorf("update: zip has no files")
	}
	rc, err := best.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	return writeNewFile(dest, rc, 0o755)
}

// untarFirstBinary 从 gzip 包里的 tar 抽出普通文件。遇到名为 yoyo 或 yoyo.exe 的立刻停。没有则留下第一个普通文件。一个都没有时返回 "update: tar has no files"。
// src 打开后交给 gzip.NewReader。
// dest 和模式 0o755 一起交给 writeNewFile。
func untarFirstBinary(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var found bool
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Base(hdr.Name)
		if name == "yoyo" || name == "yoyo.exe" || !found {
			if err := writeNewFile(dest, tr, 0o755); err != nil {
				return err
			}
			found = true
			if name == "yoyo" || name == "yoyo.exe" {
				return nil
			}
		}
	}
	if !found {
		return fmt.Errorf("update: tar has no files")
	}
	return nil
}

// writeNewFile 创建或截断 dest 并把 r 全部写入。先保证父目录存在。
// mode 作为打开 dest 的权限。
func writeNewFile(dest string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, r)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

// copyFile 打开 src，再用 writeNewFile 写到 dst。
// mode 交给 writeNewFile。
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	return writeNewFile(dst, in, mode)
}
