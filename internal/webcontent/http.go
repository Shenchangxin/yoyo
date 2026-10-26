// Package webcontent 抓公开网页和搜索结果，并抽出 YouTube、Bilibili、V2EX、GitHub 仓库根的文本。
// get、postJSON、mcpPostSession 在 ctx 为 nil 时用 20 秒超时。响应体经 LimitReader 截到 maxBody，即 256<<10（256 KiB）。
// postJSON 和 mcpPostSession 把错误正文截到 400 字节。FormatHits 最多 maxHits 条（8），摘要截到 280 个 rune。
// YouTube 字幕截到 8000 个 rune。V2EX 每条正文截到 400 个 rune，最多 8 条。
// Exa REST 的 numResults 是 8，contents.text.maxCharacters 是 400。Jina Reader 正文短于 40 字节就当空。
package webcontent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// userAgent 是抓取请求的 User-Agent。
const userAgent = "Yoyo/0.3 (+https://github.com/Shenchangxin/yoyo)"

// defaultTimeout 是 ctx 为 nil 时的超时，20 秒。
const defaultTimeout = 20 * time.Second

// maxBody 是单次响应读入的字节上限，256<<10，即 256 KiB。get、postJSON 和 mcpPostSession 都用 LimitReader 截到这里。
const maxBody = 256 << 10

// httpClient 在 c 非空时用 c，否则用 http.DefaultClient。
func httpClient(c *http.Client) *http.Client {
	if c != nil {
		return c
	}
	return http.DefaultClient
}

// get 发 GET。ctx 为 nil 时用 20 秒超时。Accept 为空时用 text/html、application/json、text/plain 和 */*。
// 正文最多读 maxBody。返回状态码、最终 URL、Content-Type 和正文。本函数不因状态码大于等于 300 而返回错误。
// client 传给 httpClient。
// rawURL 作为 GET 的地址。构造请求失败时仍作为最终 URL 返回。
// accept 为空时改成 text/html, application/json;q=0.9, text/plain;q=0.8, */*;q=0.7，然后写入 Accept。
func get(ctx context.Context, client *http.Client, rawURL, accept string) (status int, finalURL, contentType, body string, err error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, rawURL, "", "", err
	}
	req.Header.Set("User-Agent", userAgent)
	if accept == "" {
		accept = "text/html, application/json;q=0.9, text/plain;q=0.8, */*;q=0.7"
	}
	req.Header.Set("Accept", accept)
	resp, err := httpClient(client).Do(req)
	if err != nil {
		return 0, rawURL, "", "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return resp.StatusCode, resp.Request.URL.String(), resp.Header.Get("Content-Type"), "", err
	}
	return resp.StatusCode, resp.Request.URL.String(), resp.Header.Get("Content-Type"), string(raw), nil
}

// postJSON 发 JSON POST。ctx 为 nil 时用 20 秒超时。authHeader 和 authVal 都非空才设置该头。
// 正文最多读 maxBody。状态码大于等于 300 时把正文截到 400 字节放进错误。成功时返回状态码和完整读到的正文。
// client 传给 httpClient。
// rawURL 作为 POST 的地址。
// payload 转成字符串后作为请求体。
func postJSON(ctx context.Context, client *http.Client, rawURL, authHeader, authVal string, payload []byte) (status int, body string, err error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(string(payload)))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if strings.TrimSpace(authHeader) != "" && strings.TrimSpace(authVal) != "" {
		req.Header.Set(authHeader, authVal)
	}
	resp, err := httpClient(client).Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return resp.StatusCode, "", err
	}
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 400 {
			msg = msg[:400]
		}
		return resp.StatusCode, msg, fmt.Errorf("http %d: %s", resp.StatusCode, msg)
	}
	return resp.StatusCode, string(raw), nil
}
