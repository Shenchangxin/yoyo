// 监视用的公开 HTTP GET。拒绝 localhost、云元数据，以及回环、私网、链路本地和未指定地址。
// 用的是 http.DefaultClient，没有自定义重定向检查。

package personal

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// fetchUA 是 SafeGET 发送的 User-Agent。
const fetchUA = "Yoyo/0.3 (+https://github.com/Shenchangxin/yoyo)"

// SafeGET 只接受带主机名的 http 或 https。blockedHost 为真时返回错误。超时 15 秒，最多读 256 KiB。状态码大于等于 400 时返回错误。
// ctx 为 nil 时改用 Background，再套上 15 秒超时，传给 http.NewRequestWithContext。
// rawURL 交给 url.Parse。协议不是 http 或 https，或 Host 为空时返回错误。
func SafeGET(ctx context.Context, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errf("need public http(s) URL")
	}
	if blockedHost(u.Hostname()) {
		return "", errf("host is not allowed")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", fetchUA)
	req.Header.Set("Accept", "text/html, text/plain;q=0.9, */*;q=0.8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("personal: HTTP %d", resp.StatusCode)
	}
	return string(raw), nil
}

// blockedHost 拒绝 localhost、以 .localhost 结尾的名字和 metadata.google.internal。IP 直接判断；域名解析失败，或任一地址被 blockedIP 命中，也拒绝。
func blockedHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") || h == "metadata.google.internal" {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return blockedIP(ip)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return true
	}
	for _, ip := range ips {
		if blockedIP(ip) {
			return true
		}
	}
	return false
}

// blockedIP 在地址是回环、私网、链路本地单播、链路本地多播或未指定地址时返回真。
// ip 在 IsLoopback、IsPrivate、IsLinkLocalUnicast、IsLinkLocalMulticast 或 IsUnspecified 任一为真时返回 true。
func blockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}
