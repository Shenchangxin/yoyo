package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MaxPackageBytes = 512 << 20

type ProgressFunc func(done, total int64)

func Download(ctx context.Context, asset Asset, destDir string, progress ProgressFunc) (string, error) {
	if asset.URL == "" {
		return "", fmt.Errorf("update: empty asset url")
	}
	name := strings.TrimSpace(asset.Filename)
	if name == "" {
		name = filepath.Base(asset.URL)
	}
	name = filepath.Base(name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "yoyo-update.bin"
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(destDir, name)
	var last error
	for _, u := range candidateURLs(asset.URL) {
		err := downloadURL(ctx, u, dest, asset.SHA256, asset.Size, progress)
		if err == nil {
			return dest, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("update: download failed")
	}
	return "", last
}

func candidateURLs(raw string) []string {
	out := []string{raw}
	if fallback := APIFileURLFromResolve(raw); fallback != "" && fallback != raw {
		out = append(out, fallback)
	}
	return out
}

func APIFileURLFromResolve(raw string) string {
	const marker = "/resolve/"
	i := strings.Index(raw, marker)
	if i < 0 {
		return ""
	}
	// .../datasets/{repo}/resolve/{rev}/{file}
	head := raw[:i]
	rest := raw[i+len(marker):]
	rev, file, ok := strings.Cut(rest, "/")
	if !ok || file == "" {
		return ""
	}
	repo := ""
	if j := strings.LastIndex(head, "/datasets/"); j >= 0 {
		repo = strings.Trim(head[j+len("/datasets/"):], "/")
		head = head[:j]
	}
	if repo == "" {
		return ""
	}
	q := url.Values{}
	q.Set("Revision", rev)
	q.Set("FilePath", file)
	return strings.TrimRight(head, "/") + "/api/v1/datasets/" + repo + "/repo?" + q.Encode()
}

func downloadURL(ctx context.Context, rawURL, dest, wantSHA string, expect int64, progress ProgressFunc) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 25*time.Minute)
		defer cancel()
	}
	tmp := dest + ".tmp"
	var start int64
	if st, err := os.Stat(tmp); err == nil {
		start = st.Size()
		if start > MaxPackageBytes {
			_ = os.Remove(tmp)
			start = 0
		}
	}
	resp, err := doGET(ctx, rawURL, false, start)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if start > 0 && resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		return finishDownload(tmp, dest, wantSHA)
	}
	resume := start > 0 && resp.StatusCode == http.StatusPartialContent
	if !resume {
		if resp.StatusCode >= 300 {
			return fmt.Errorf("update: download http %d", resp.StatusCode)
		}
		start = 0
	}
	total := resp.ContentLength
	if resume && total > 0 {
		total += start
	}
	if total <= 0 {
		total = expect
	}
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	ok := false
	closed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	h := sha256.New()
	if resume {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		if _, err := io.Copy(h, io.LimitReader(f, start)); err != nil {
			return err
		}
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			return err
		}
	} else {
		if err := f.Truncate(0); err != nil {
			return err
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}
	remain := MaxPackageBytes + 1 - start
	if remain < 1 {
		return fmt.Errorf("update: package larger than %d bytes", MaxPackageBytes)
	}
	pr := &progressReader{r: resp.Body, n: start, total: total, on: progress}
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(pr, remain))
	if err != nil {
		return err
	}
	if start+n > MaxPackageBytes {
		return fmt.Errorf("update: package larger than %d bytes", MaxPackageBytes)
	}
	if err := f.Close(); err != nil {
		return err
	}
	closed = true
	if wantSHA != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if !strings.EqualFold(got, wantSHA) {
			return fmt.Errorf("update: sha256 mismatch")
		}
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(dest)
		if err2 := os.Rename(tmp, dest); err2 != nil {
			return err2
		}
	}
	ok = true
	return nil
}

func finishDownload(tmp, dest, wantSHA string) error {
	f, err := os.Open(tmp)
	if err != nil {
		return err
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, MaxPackageBytes+1))
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if n > MaxPackageBytes {
		_ = os.Remove(tmp)
		return fmt.Errorf("update: package larger than %d bytes", MaxPackageBytes)
	}
	if wantSHA != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if !strings.EqualFold(got, wantSHA) {
			_ = os.Remove(tmp)
			return fmt.Errorf("update: sha256 mismatch")
		}
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(dest)
		return os.Rename(tmp, dest)
	}
	return nil
}

type progressReader struct {
	r     io.Reader
	n     int64
	total int64
	on    ProgressFunc
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.n += int64(n)
	if p.on != nil {
		p.on(p.n, p.total)
	}
	return n, err
}
