// extract.go 拥有解压：zip、tar.gz、tar.bz2 和裸二进制，先解到 dest.staging，完成后改名为 dest。
// 不拥有下载和校验（fetch.go）。buddy 的二进制安装和私有 Node/uv 运行时安装调用它。

package fetch

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// 压缩包种类。
const (
	KindZip    = "zip"
	KindTarGz  = "tar.gz"
	KindTarBz2 = "tar.bz2"
	KindRaw    = "raw"
)

// ErrUnsafePath 表示包里有绝对路径、.. 或逃出目标目录的链接。
var ErrUnsafePath = errors.New("fetch: archive entry escapes the destination")

// maxExtractBytes 是一次解压写出的总字节上限，挡住压缩炸弹。
const maxExtractBytes = 4 << 30

// KindOf 按文件名猜压缩包种类。认不出的当成裸二进制。
func KindOf(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return KindZip
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return KindTarGz
	case strings.HasSuffix(lower, ".tar.bz2"), strings.HasSuffix(lower, ".tbz2"), strings.HasSuffix(lower, ".tbz"):
		return KindTarBz2
	default:
		return KindRaw
	}
}

// Extract 把 archive 按 kind 解到 dest。strip 是每条路径要去掉的前几级目录，和 tar --strip-components 一样。
// 先写 dest.staging，全部成功后删掉旧的 dest 再改名；中途失败时删掉 staging，旧的 dest 不动。
// 裸二进制按 archive 的文件名放进 dest，并加上可执行位。
func Extract(archive, dest, kind string, strip int) error {
	if kind == "" {
		kind = KindOf(archive)
	}
	staging := dest + ".staging"
	_ = os.RemoveAll(staging)
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	var err error
	switch kind {
	case KindZip:
		err = extractZip(archive, staging, strip)
	case KindTarGz:
		err = extractTar(archive, staging, strip, func(r io.Reader) (io.Reader, error) { return gzip.NewReader(r) })
	case KindTarBz2:
		err = extractTar(archive, staging, strip, func(r io.Reader) (io.Reader, error) { return bzip2.NewReader(r), nil })
	case KindRaw:
		err = copyRaw(archive, filepath.Join(staging, filepath.Base(archive)))
	default:
		err = fmt.Errorf("fetch: unknown archive kind %q", kind)
	}
	if err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if err := os.RemoveAll(dest); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if err := os.Rename(staging, dest); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	return nil
}

// entryPath 把包里的 name 去掉前 strip 级，变成 root 下的路径。返回空串表示这一条被 strip 吃掉了。
// 绝对路径、盘符、.. 一律拒绝。
func entryPath(root, name string, strip int) (string, error) {
	clean := strings.ReplaceAll(name, `\`, "/")
	if strings.HasPrefix(clean, "/") || filepath.VolumeName(clean) != "" || strings.Contains(clean, ":") {
		return "", ErrUnsafePath
	}
	var parts []string
	for _, part := range strings.Split(clean, "/") {
		switch part {
		case "", ".":
			continue
		case "..":
			return "", ErrUnsafePath
		}
		parts = append(parts, part)
	}
	if len(parts) <= strip {
		return "", nil
	}
	parts = parts[strip:]
	full := filepath.Join(append([]string{root}, parts...)...)
	if !within(root, full) {
		return "", ErrUnsafePath
	}
	return full, nil
}

// within 在 target 等于 root 或在 root 之下时为真。
func within(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

// linkTarget 检查 at 处指向 target 的符号链接解析后仍在 root 里。绝对目标一律拒绝。
func linkTarget(root, at, target string) error {
	t := strings.ReplaceAll(target, `\`, "/")
	if t == "" || strings.HasPrefix(t, "/") || filepath.VolumeName(t) != "" || strings.Contains(t, ":") {
		return ErrUnsafePath
	}
	resolved := filepath.Join(filepath.Dir(at), filepath.FromSlash(path.Clean(t)))
	if !within(root, resolved) {
		return ErrUnsafePath
	}
	return nil
}

// budget 记录这次解压还能写多少字节。
type budget struct{ left int64 }

// copyN 从 r 拷到 w，超过剩余额度时报错。
func (b *budget) copyN(w io.Writer, r io.Reader) error {
	n, err := io.Copy(w, io.LimitReader(r, b.left+1))
	b.left -= n
	if err != nil {
		return err
	}
	if b.left < 0 {
		return fmt.Errorf("fetch: archive expands past %d bytes", int64(maxExtractBytes))
	}
	return nil
}

// writeFile 把 r 写到 full，mode 带可执行位时保留。
func writeFile(full string, r io.Reader, mode os.FileMode, b *budget) error {
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	perm := os.FileMode(0o644)
	if mode&0o111 != 0 {
		perm = 0o755
	}
	f, err := os.OpenFile(full, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if err := b.copyN(f, r); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Chmod(full, perm)
}

// extractZip 把 zip 解到 root。符号链接只在目标仍在 root 里时创建。
func extractZip(archive, root string, strip int) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	b := &budget{left: maxExtractBytes}
	for _, f := range zr.File {
		full, err := entryPath(root, f.Name, strip)
		if err != nil {
			return err
		}
		if full == "" {
			continue
		}
		mode := f.Mode()
		switch {
		case mode.IsDir():
			if err := os.MkdirAll(full, 0o755); err != nil {
				return err
			}
		case mode&os.ModeSymlink != 0:
			rc, err := f.Open()
			if err != nil {
				return err
			}
			target, err := io.ReadAll(io.LimitReader(rc, 4096))
			_ = rc.Close()
			if err != nil {
				return err
			}
			if err := makeLink(root, full, string(target)); err != nil {
				return err
			}
		default:
			rc, err := f.Open()
			if err != nil {
				return err
			}
			err = writeFile(full, rc, mode, b)
			_ = rc.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// extractTar 把 tar 解到 root。wrap 套上 gzip 或 bzip2。硬链接和符号链接的目标都必须在 root 里。
func extractTar(archive, root string, strip int, wrap func(io.Reader) (io.Reader, error)) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	r, err := wrap(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(r)
	b := &budget{left: maxExtractBytes}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		full, err := entryPath(root, hdr.Name, strip)
		if err != nil {
			return err
		}
		if full == "" {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(full, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := writeFile(full, tr, os.FileMode(hdr.Mode), b); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := makeLink(root, full, hdr.Linkname); err != nil {
				return err
			}
		case tar.TypeLink:
			src, err := entryPath(root, hdr.Linkname, strip)
			if err != nil || src == "" {
				return ErrUnsafePath
			}
			if err := copyFile(src, full); err != nil {
				return err
			}
		default:
			// 设备、FIFO 等不需要，跳过。
		}
	}
}

// makeLink 在 full 处建指向 target 的符号链接。目标逃出 root 时拒绝；系统不允许建链接（Windows 没有权限）时改成复制。
func makeLink(root, full, target string) error {
	if err := linkTarget(root, full, target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	_ = os.Remove(full)
	if err := os.Symlink(filepath.FromSlash(target), full); err == nil {
		return nil
	}
	src := filepath.Join(filepath.Dir(full), filepath.FromSlash(target))
	if info, err := os.Stat(src); err == nil && !info.IsDir() {
		return copyFile(src, full)
	}
	return nil
}

// copyFile 把 src 复制到 dst，保留可执行位。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	b := &budget{left: maxExtractBytes}
	return writeFile(dst, in, info.Mode(), b)
}

// copyRaw 把裸二进制复制到 dst 并加上可执行位。
func copyRaw(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	b := &budget{left: maxExtractBytes}
	return writeFile(dst, in, 0o755, b)
}
