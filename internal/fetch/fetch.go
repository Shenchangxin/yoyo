// Package fetch 下载并解开 agent 运行时和二进制包。
// 下载按顺序试每个地址，先写临时文件，sha256 对上再改名；解压先进 staging 目录，拒绝逃出目标目录的路径和符号链接。
// buddy 的托管安装调用它；它不知道装的是什么，也不改 PATH。
package fetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrHashMismatch 表示下载内容的 sha256 和期望的不一致。
var ErrHashMismatch = errors.New("fetch: sha256 mismatch")

// ErrTooLarge 表示下载内容超过了 MaxBytes。
var ErrTooLarge = errors.New("fetch: download larger than the limit")

const (
	// connectTimeout 是单个地址建连、TLS 握手和等响应头的上限。
	connectTimeout = 20 * time.Second
	// totalTimeout 是一次 Download 的总上限，含所有回退地址。
	totalTimeout = 25 * time.Minute
	// userAgent 是下载请求的 User-Agent。
	userAgent = "yoyo-fetch/1"
)

// Spec 是一次下载。
type Spec struct {
	// URLs 按顺序尝试，前一个失败才试下一个。
	URLs []string
	// SHA256 是期望的十六进制哈希。空时不校验，实际哈希写进 Result。
	SHA256 string
	// MaxBytes 大于 0 时，读到超过它就放弃这个地址。
	MaxBytes int64
	// Dest 是最终文件路径。上级目录不存在时创建。
	Dest string
	// Progress 在读到新数据时收到已读字节和总字节；总字节未知时为 0。可以为 nil。
	Progress func(done, total int64)
	// Client 为 nil 时用带代理和建连超时的默认客户端。测试用它换掉网络。
	Client *http.Client
}

// Result 是成功的那次下载。
type Result struct {
	// URL 是实际下载成功的地址。
	URL string
	// SHA256 是实际内容的十六进制哈希。
	SHA256 string
	// Size 是写下的字节数。
	Size int64
}

// defaultClient 走环境变量里的代理，建连、握手和等响应头各 20 秒。
func defaultClient() *http.Client {
	dialer := &net.Dialer{Timeout: connectTimeout, KeepAlive: 30 * time.Second}
	return &http.Client{Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   connectTimeout,
		ResponseHeaderTimeout: connectTimeout,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          4,
		IdleConnTimeout:       60 * time.Second,
	}}
}

// Download 按 spec 下载到 spec.Dest。每个地址先写 Dest.part，校验通过后改名，失败时删掉临时文件再试下一个。
// 全部失败时返回最后一个错误；ctx 取消时立刻返回。哈希不一致返回包着 ErrHashMismatch 的错误。
func Download(ctx context.Context, spec Spec) (Result, error) {
	if len(spec.URLs) == 0 {
		return Result{}, errors.New("fetch: no url")
	}
	if strings.TrimSpace(spec.Dest) == "" {
		return Result{}, errors.New("fetch: no destination")
	}
	if err := os.MkdirAll(filepath.Dir(spec.Dest), 0o755); err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, totalTimeout)
	defer cancel()
	client := spec.Client
	if client == nil {
		client = defaultClient()
	}
	var last error
	for _, raw := range spec.URLs {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		res, err := downloadOne(ctx, client, raw, spec)
		if err == nil {
			return res, nil
		}
		last = fmt.Errorf("%s: %w", hostOf(raw), err)
	}
	return Result{}, last
}

// downloadOne 从 raw 下载一次。成功时 Dest 已经是完整文件。
func downloadOne(ctx context.Context, client *http.Client, raw string, spec Spec) (Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("http %s", resp.Status)
	}
	total := resp.ContentLength
	if spec.MaxBytes > 0 && total > spec.MaxBytes {
		return Result{}, ErrTooLarge
	}
	tmp := spec.Dest + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return Result{}, err
	}
	keep := false
	defer func() {
		_ = f.Close()
		if !keep {
			_ = os.Remove(tmp)
		}
	}()
	sum := sha256.New()
	var body io.Reader = resp.Body
	if spec.MaxBytes > 0 {
		body = io.LimitReader(resp.Body, spec.MaxBytes+1)
	}
	body = &countReader{r: body, total: total, on: spec.Progress}
	n, err := io.Copy(io.MultiWriter(f, sum), body)
	if err != nil {
		return Result{}, err
	}
	if spec.MaxBytes > 0 && n > spec.MaxBytes {
		return Result{}, ErrTooLarge
	}
	if total > 0 && n != total {
		return Result{}, fmt.Errorf("short body: %d of %d bytes", n, total)
	}
	if err := f.Close(); err != nil {
		return Result{}, err
	}
	got := hex.EncodeToString(sum.Sum(nil))
	if want := strings.TrimSpace(spec.SHA256); want != "" && !strings.EqualFold(want, got) {
		return Result{}, ErrHashMismatch
	}
	_ = os.Remove(spec.Dest)
	if err := os.Rename(tmp, spec.Dest); err != nil {
		return Result{}, err
	}
	keep = true
	return Result{URL: raw, SHA256: got, Size: n}, nil
}

// countReader 累计读到的字节，每读一次就报一次进度。
type countReader struct {
	r     io.Reader
	n     int64
	total int64
	on    func(done, total int64)
}

// Read 读底层数据并报进度。
func (c *countReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.on != nil && n > 0 {
		c.on(c.n, c.total)
	}
	return n, err
}

// hostOf 返回 raw 的主机名，错误信息里只写它，不写完整地址。
func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "download"
	}
	return u.Host
}

// mirrorPrefix 是 GitHub 下载的国内镜像前缀。
const mirrorPrefix = "https://ghfast.top/"

// MirrorURLs 返回 raw 和它的镜像地址。只有 github.com 和 objects.githubusercontent.com 的 https 地址才追加镜像，其余原样返回一个。
func MirrorURLs(raw string) []string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return []string{raw}
	}
	switch strings.ToLower(u.Host) {
	case "github.com", "objects.githubusercontent.com":
		return []string{raw, mirrorPrefix + raw}
	default:
		return []string{raw}
	}
}
