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

type RotatorOpts struct {
	MaxSize    int64
	MaxFiles   int
	MaxAgeDays int
}

type Rotator struct {
	mu      sync.Mutex
	path    string
	file    *os.File
	size    int64
	opt     RotatorOpts
	closed  bool
}

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

func (r *Rotator) Path() string {
	if r == nil {
		return ""
	}
	return r.path
}

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

func (r *Rotator) Cleanup() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cleanupLocked()
}

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

func (r *Rotator) backup(n int) string {
	return r.path + "." + strconv.Itoa(n) + ".gz"
}

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

func rotateIndex(name, base string) int {
	rest := strings.TrimPrefix(name, base+".")
	rest = strings.TrimSuffix(rest, ".gz")
	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0
	}
	return n
}

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
