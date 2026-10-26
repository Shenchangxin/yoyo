// 拉取并验签清单，把核对过的字节写到 yoyo.staging，不替换正在运行的文件。

package update

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxManifestBytes = 2 << 20

var (
	httpMu     sync.Mutex
	httpClient = defaultHTTPClient()
)

// defaultHTTPClient 返回超时 30 分钟的 HTTP 客户端。
func defaultHTTPClient() *http.Client {
	// 安装包可能到数百 MB，所以客户端超时是 30 分钟。清单请求的时限靠调用方的 context。
	return &http.Client{Timeout: 30 * time.Minute}
}

// SetHTTPClient 换掉包级 HTTP 客户端。传入 nil 时恢复 defaultHTTPClient。
// c 为 nil 时改成 defaultHTTPClient，否则存成包级客户端。
func SetHTTPClient(c *http.Client) {
	httpMu.Lock()
	defer httpMu.Unlock()
	if c == nil {
		httpClient = defaultHTTPClient()
		return
	}
	httpClient = c
}

// client 返回当前包级 HTTP 客户端。
func client() *http.Client {
	httpMu.Lock()
	defer httpMu.Unlock()
	return httpClient
}

// CheckResult 是一次清单检查的结果。Newer 表示清单版本比当前版本新。Verified 表示验签并解析成功。
type CheckResult struct {
	Current  Status   `json:"current"`
	Manifest Manifest `json:"manifest"`
	Asset    Asset    `json:"asset,omitempty"`
	Newer    bool     `json:"newer"`
	Verified bool     `json:"verified"`
}

// Check 验签并解析清单。payload 非空时直接用这份正文和 sig。否则按 feedAttemptURLs 依次 GET 清单和旁边的 .sig。签名被拒且正文仍是带版本的清单时立刻返回该错误。地址都失败且没有更早的错误时返回 "update: empty url"。
// ctx 在 payload 为空时交给 httpGet。
// manifestURL 在 payload 为空时交给 feedAttemptURLs。
// pub 交给 finishCheck。
func Check(ctx context.Context, manifestURL string, payload, sig []byte, pub ed25519.PublicKey) (CheckResult, error) {
	res := CheckResult{Current: Current("")}
	if len(payload) > 0 {
		return finishCheck(res, payload, sig, pub)
	}
	var last error
	for _, u := range feedAttemptURLs(manifestURL) {
		body, err := httpGet(ctx, u, true)
		if err != nil {
			last = err
			continue
		}
		gotSig := sig
		if len(gotSig) == 0 {
			gotSig, err = httpGet(ctx, SigURL(u), true)
			if err != nil {
				last = err
				continue
			}
		}
		out, err := finishCheck(res, body, gotSig, pub)
		if err == nil {
			return out, nil
		}
		last = err
		if signedManifestRejected(body, err) {
			return out, err
		}
	}
	if last == nil {
		last = fmt.Errorf("update: empty url")
	}
	return res, last
}

// finishCheck 先验签再解析。成功时 Verified 为真，Newer 来自 IsNewer，并按 DesktopOS 选择资源。
// res 在验签或解析失败时原样返回；成功时写入 Manifest、Verified 和 Newer。
// payload 和 pub、sig 一起交给 Verify，通过后再交给 ParseManifest。
// sig 和 pub、payload 一起交给 Verify。
// pub 和 payload、sig 一起交给 Verify。
func finishCheck(res CheckResult, payload, sig []byte, pub ed25519.PublicKey) (CheckResult, error) {
	if err := Verify(pub, payload, sig); err != nil {
		return res, err
	}
	m, err := ParseManifest(payload)
	if err != nil {
		return res, err
	}
	res.Manifest = m
	res.Verified = true
	res.Newer = IsNewer(m.Version, res.Current.Version)
	if asset, ok := m.SelectAsset(DesktopOS()); ok {
		res.Asset = asset
	}
	return res, nil
}

// feedAttemptURLs 先放清单地址本身。地址以 /channels/latest.json 结尾时，再追加同目录的 nightly.json 和 stable.json。空白地址返回 nil。
// manifestURL 去掉两端空白，空则返回 nil；以 /channels/latest.json 结尾时再追加同目录的 nightly.json 和 stable.json。
func feedAttemptURLs(manifestURL string) []string {
	manifestURL = strings.TrimSpace(manifestURL)
	if manifestURL == "" {
		return nil
	}
	out := []string{manifestURL}
	if !strings.HasSuffix(strings.TrimRight(manifestURL, "/"), "/channels/latest.json") {
		return out
	}
	base := strings.TrimSuffix(strings.TrimRight(manifestURL, "/"), "latest.json")
	for _, name := range []string{"nightly.json", "stable.json"} {
		u := base + name
		if u != manifestURL {
			out = append(out, u)
		}
	}
	return out
}

// signedManifestRejected 在错误文本含 "signature rejected"，且正文能解析出非空 version 时返回 true。
// payload 在错误文本含 signature rejected 时交给 ParseManifest。
// err 为 nil，或错误文本不含 signature rejected 时返回 false。
func signedManifestRejected(payload []byte, err error) bool {
	if err == nil || !strings.Contains(err.Error(), "signature rejected") {
		return false
	}
	m, perr := ParseManifest(payload)
	return perr == nil && strings.TrimSpace(m.Version) != ""
}

// Stage 把字节写到 destDir/yoyo.staging，权限 0o755。wantSHA 非空且与内容 SHA-256 不一致时返回 "update: sha256 mismatch"。本函数不替换正在运行的可执行文件。
// bin 用来算 SHA-256，并写到 destDir 下的暂存文件。
func Stage(destDir string, bin []byte, wantSHA string) (string, error) {
	sum := sha256.Sum256(bin)
	got := hex.EncodeToString(sum[:])
	if wantSHA != "" && !strings.EqualFold(wantSHA, got) {
		return "", fmt.Errorf("update: sha256 mismatch")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(destDir, StagingName())
	if err := os.WriteFile(path, bin, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

// httpGet 以最多 2 MiB 调用 Fetch。
// ctx 和 rawURL、nocache 一起交给 Fetch，字节上限是 maxManifestBytes。
// rawURL 交给 Fetch。
// nocache 交给 Fetch。
func httpGet(ctx context.Context, rawURL string, nocache bool) ([]byte, error) {
	return Fetch(ctx, rawURL, maxManifestBytes, nocache)
}

// Fetch 把 URL 读进内存，最多 max 字节。max 小于等于 0 时按 1 MiB。空地址返回 "update: empty url"。resolve 地址失败时再试数据集 API。本函数不写正在运行的可执行文件。
// ctx 交给 fetchOnce。
// rawURL 为空时返回错误，否则交给 candidateURLs。
// nocache 交给 fetchOnce。
func Fetch(ctx context.Context, rawURL string, max int64, nocache bool) ([]byte, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("update: empty url")
	}
	if max <= 0 {
		max = 1 << 20
	}
	var last error
	for _, u := range candidateURLs(rawURL) {
		b, err := fetchOnce(ctx, u, max, nocache)
		if err == nil {
			return b, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("update: fetch failed")
	}
	return nil, last
}

// fetchOnce 做一次 GET。状态码大于等于 300 返回 "update: http <码>"。超过 max 返回 "update: payload too large"。正文只有空白时返回 "update: empty body"。
// ctx 和 rawURL、nocache 一起交给 doGET，start 为 0。
// rawURL 交给 doGET。
// nocache 交给 doGET。
func fetchOnce(ctx context.Context, rawURL string, max int64, nocache bool) ([]byte, error) {
	resp, err := doGET(ctx, rawURL, nocache, 0)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("update: http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, fmt.Errorf("update: payload too large")
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, fmt.Errorf("update: empty body")
	}
	return b, nil
}

// doGET 发出 GET。User-Agent 用 UpdaterUA，Accept 为 */*，Referer 来自 Referer。nocache 为真时加上 Cache-Control 和 Pragma 的 no-cache。start 大于 0 时加 Range。
// ctx 交给 NewRequestWithContext。
// rawURL 作为 GET 地址。
func doGET(ctx context.Context, rawURL string, nocache bool, start int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UpdaterUA)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", Referer())
	if nocache {
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Pragma", "no-cache")
	}
	if start > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start))
	}
	return client().Do(req)
}
