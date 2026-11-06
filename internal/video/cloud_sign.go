// cloud_sign.go 给腾讯云接口签 TC3-HMAC-SHA256，并拆开密钥框里用冒号连起来的多段凭证。
// 火山引擎视觉服务的签名在 visual_sign.go。

package video

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// sha256Hex 返回 b 的 SHA-256 十六进制摘要。
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// signTC3 按腾讯云 TC3-HMAC-SHA256 给 POST JSON 请求加鉴权头和公共参数头。
// req 的地址要先设好；Content-Type 会写成 application/json; charset=utf-8。
// body 是请求正文。secretID、secretKey 是腾讯云 API 密钥。service 是域名前缀，比如 tts。
// action、version 是接口名和版本。region 空着就不写 X-TC-Region。now 是签名时间，和服务器相差超过 5 分钟会被拒。
func signTC3(req *http.Request, body []byte, secretID, secretKey, service, action, version, region string, now time.Time) {
	const ct = "application/json; charset=utf-8"
	ts := strconv.FormatInt(now.Unix(), 10)
	date := now.UTC().Format("2006-01-02")
	req.Header.Set("Content-Type", ct)
	canonical := "POST\n/\n\ncontent-type:" + ct + "\nhost:" + req.URL.Host + "\n\ncontent-type;host\n" + sha256Hex(body)
	scope := date + "/" + service + "/tc3_request"
	toSign := "TC3-HMAC-SHA256\n" + ts + "\n" + scope + "\n" + sha256Hex([]byte(canonical))
	k := hmacSHA256([]byte("TC3"+secretKey), []byte(date))
	k = hmacSHA256(k, []byte(service))
	k = hmacSHA256(k, []byte("tc3_request"))
	sig := hex.EncodeToString(hmacSHA256(k, []byte(toSign)))
	req.Header.Set("Authorization", "TC3-HMAC-SHA256 Credential="+secretID+"/"+scope+", SignedHeaders=content-type;host, Signature="+sig)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Timestamp", ts)
	req.Header.Set("X-TC-Version", version)
	if region != "" {
		req.Header.Set("X-TC-Region", region)
	}
}

// splitCred 把密钥框里用英文冒号连起来的几段拆开，比如 AK:SK、AppID:Token、APPID:APIKey:APISecret。
// 段数不够或有空段时返回 nil。最后一段可以再含冒号。
func splitCred(key string, n int) []string {
	parts := strings.SplitN(strings.TrimSpace(key), ":", n)
	if len(parts) != n {
		return nil
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil
		}
	}
	return parts
}

// settingStr 从连接的附加设置里按顺序取第一个非空的字符串。
func settingStr(s map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := s[k]; ok {
			if str := strings.TrimSpace(anyString(v)); str != "" {
				return str
			}
		}
	}
	return ""
}

// anyString 把设置里的值收成字符串。数字按十进制写。
func anyString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return ""
	}
}
