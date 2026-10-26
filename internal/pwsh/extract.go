// 本文件把 zip 解到目标目录，跳过符号链接和含 .. 的路径。字节上限在 pin.go。

package pwsh

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractZip 把 zipPath 解到 destDir，并返回其中的 pwsh 可执行文件路径。
// 跳过空名、目录项、符号链接、safeRel 拒绝的名字，以及落在 destDir 外面的路径。
// 累计 UncompressedSize64 超过 maxUncompressedBytes，或单条超过 maxZipBytes，就失败。找不到 pwsh.exe 时错误文案是 pwsh: archive has no pwsh.exe。
func extractZip(zipPath, destDir string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("pwsh: unzip: %w", err)
	}
	defer r.Close()
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	root := filepath.Clean(destDir)
	var total uint64
	for _, f := range r.File {
		name := zipEntryName(f.Name)
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			continue
		}
		rel, ok := safeRel(name)
		if !ok {
			continue
		}
		total += f.UncompressedSize64
		if total > maxUncompressedBytes {
			return "", fmt.Errorf("pwsh: uncompressed archive larger than %d bytes", maxUncompressedBytes)
		}
		if f.UncompressedSize64 > maxZipBytes {
			return "", fmt.Errorf("pwsh: zip entry too large")
		}
		dest := filepath.Join(root, rel)
		if !within(root, dest) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return "", err
		}
		if err := writeZipFile(f, dest); err != nil {
			return "", err
		}
	}
	bin := lookDir(root)
	if bin == "" {
		return "", fmt.Errorf("pwsh: archive has no pwsh.exe")
	}
	return bin, nil
}

// zipEntryName 去掉两端空白，并把反斜杠换成斜杠。
func zipEntryName(name string) string {
	return strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
}

// safeRel 把 zip 内路径收成相对路径。空、.、或任何一段是 .. 时返回 false。
func safeRel(name string) (string, bool) {
	name = zipEntryName(name)
	if name == "" || name == "." {
		return "", false
	}
	parts := strings.Split(name, "/")
	var keep []string
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		if p == ".." {
			return "", false
		}
		keep = append(keep, p)
	}
	if len(keep) == 0 {
		return "", false
	}
	return filepath.Join(keep...), true
}

// within 报告 dest 是否落在 root 里。Rel 失败、结果以 .. 开头或是绝对路径时返回 false。dest 就是 root 时返回 true。
func within(root, dest string) bool {
	rel, err := filepath.Rel(root, dest)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != "" && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}

// writeZipFile 把条目写到 dest.tmp，权限 0o755，再改名为 dest。复制用 LimitReader 截到 maxZipBytes+1。复制或关闭失败时删掉临时文件。
func writeZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	tmp := dest + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, io.LimitReader(rc, maxZipBytes+1))
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	_ = os.Remove(dest)
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
