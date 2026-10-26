// 本文件按大小滚动日志，并把旧文件压成 path.N.gz。
// 默认大小是 DefaultMaxSize，默认备份个数是 DefaultFiles，默认保留天数是 DefaultAgeDays。
// 当前正在写的文件不参与按天数删除。序号大于 MaxFiles，或修改时间早于截止日期的备份会被删掉。

package diaglog

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RotatorOpts 是滚动参数。小于等于 0 的字段在 OpenRotator 里改成包级默认。
type RotatorOpts struct {
	MaxSize    int64
	MaxFiles   int
	MaxAgeDays int
}

// Rotator 把字节追加到一个文件，超过 MaxSize 时换成 gzip 备份。
type Rotator struct {
	mu     sync.Mutex
	path   string
	file   *os.File
	size   int64
	opt    RotatorOpts
	closed bool
}

// OpenRotator 打开 path 并追加。MaxSize、MaxFiles、MaxAgeDays 小于等于 0 时分别用 DefaultMaxSize、DefaultFiles、DefaultAgeDays。
// 父目录权限 0o755。文件以 O_CREATE、O_APPEND、O_WRONLY 打开，权限 0o600。
func OpenRotator(path string, opt RotatorOpts) (*Rotator, error) {
	if opt.MaxSize <= 0 {
		opt.MaxSize = DefaultMaxSize
	}
	if opt.MaxFiles <= 0 {
		opt.MaxFiles = DefaultFiles
	}
	if opt.MaxAgeDays <= 0 {
		opt.MaxAgeDays = DefaultAgeDays
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	r := &Rotator{path: path, opt: opt}
	if err := r.openLocked(); err != nil {
		return nil, err
	}
	return r, nil
}

// Path 返回当前日志路径。r 为 nil 时返回空字符串。
func (r *Rotator) Path() string {
	if r == nil {
		return ""
	}
	return r.path
}

// Write 追加 p。r 为 nil 或已关闭时返回 io.ErrClosedPipe。
// 仅当当前大小大于 0，并且当前大小加上 len(p) 大于 MaxSize 时才滚动。空文件上的一次大写入不会先滚动。
func (r *Rotator) Write(p []byte) (int, error) {
	if r == nil {
		return 0, io.ErrClosedPipe
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return 0, io.ErrClosedPipe
	}
	if r.file == nil {
		if err := r.openLocked(); err != nil {
			return 0, err
		}
	}
	if r.size > 0 && r.size+int64(len(p)) > r.opt.MaxSize {
		if err := r.rotateLocked(); err != nil {
			return 0, err
		}
	}
	n, err := r.file.Write(p)
	r.size += int64(n)
	return n, err
}

// Close 关掉文件并拒绝后续写入。文件已经是 nil 时返回 nil。
func (r *Rotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}

// Cleanup 按备份个数和修改时间删除旧备份。持锁调用 cleanupLocked。
func (r *Rotator) Cleanup() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cleanupLocked()
}

// openLocked 以追加方式打开当前文件，权限 0o600，并把 size 设成现有长度。Stat 失败时关掉刚打开的句柄。
func (r *Rotator) openLocked() error {
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	r.file = f
	r.size = st.Size()
	return nil
}

// rotateLocked 把现有备份从 MaxFiles-1 号往下改名为下一号，再把当前文件压成 .1.gz。
// 压缩失败时把当前文件改名为去掉 .gz 的 .1。成功则删掉未压缩的当前文件。然后 Cleanup，size 归零并重新打开。
func (r *Rotator) rotateLocked() error {
	if r.file != nil {
		_ = r.file.Close()
		r.file = nil
	}
	for i := r.opt.MaxFiles - 1; i >= 1; i-- {
		src := r.backup(i)
		dst := r.backup(i + 1)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		_ = os.Remove(dst)
		_ = os.Rename(src, dst)
	}
	first := r.backup(1)
	_ = os.Remove(first)
	if err := gzipFile(r.path, first); err != nil {
		_ = os.Rename(r.path, strings.TrimSuffix(first, ".gz"))
	} else {
		_ = os.Remove(r.path)
	}
	_ = r.cleanupLocked()
	r.size = 0
	return r.openLocked()
}

// backup 返回 path 加序号再加 .gz，例如 path.1.gz。
func (r *Rotator) backup(n int) string {
	return r.path + "." + strconv.Itoa(n) + ".gz"
}

// cleanupLocked 删除同目录里以前缀匹配、且不是当前文件名的条目。
// 序号大于 MaxFiles，或修改时间早于现在减去 MaxAgeDays 个 24 小时的，都会删除。当前文件名本身跳过。
func (r *Rotator) cleanupLocked() error {
	cutoff := time.Now().Add(-time.Duration(r.opt.MaxAgeDays) * 24 * time.Hour)
	dir := filepath.Dir(r.path)
	base := filepath.Base(r.path)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range ents {
		name := e.Name()
		if !strings.HasPrefix(name, base) {
			continue
		}
		if name == base {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		n := rotateIndex(name, base)
		if n > r.opt.MaxFiles || info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
}

// rotateIndex 从文件名取出备份序号。去掉 base 和一个点，再去掉末尾的 .gz。不是整数时返回 0。
func rotateIndex(name, base string) int {
	rest := strings.TrimPrefix(name, base+".")
	rest = strings.TrimSuffix(rest, ".gz")
	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0
	}
	return n
}

// gzipFile 把 src 压到 dst。dst 以 O_CREATE、O_TRUNC、O_WRONLY 打开，权限 0o600。复制、gzip 关闭和文件关闭的错误按这个顺序返回。
func gzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(out)
	_, copyErr := io.Copy(gz, in)
	closeErr := gz.Close()
	outErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return outErr
}
