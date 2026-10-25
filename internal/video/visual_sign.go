package video

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

func signVisual(req *http.Request, body []byte, ak, sk string, now time.Time) {
	if req == nil || req.URL == nil {
		return
	}
	now = now.UTC()
	stamp := now.Format("20060102T150405Z")
	short := now.Format("20060102")
	sum := sha256.Sum256(body)
	payload := hex.EncodeToString(sum[:])
	host := req.URL.Host
	uri := req.URL.EscapedPath()
	if uri == "" {
		uri = "/"
	}
	query := req.URL.RawQuery
	canonicalHeaders := "content-type:application/json\n" +
		"host:" + host + "\n" +
		"x-content-sha256:" + payload + "\n" +
		"x-date:" + stamp + "\n"
	signed := "content-type;host;x-content-sha256;x-date"
	canonical := "POST\n" + uri + "\n" + query + "\n" + canonicalHeaders + "\n" + signed + "\n" + payload
	scope := short + "/cn-north-1/cv/request"
	canonSum := sha256.Sum256([]byte(canonical))
	stringToSign := "HMAC-SHA256\n" + stamp + "\n" + scope + "\n" + hex.EncodeToString(canonSum[:])
	signature := hex.EncodeToString(hmacSHA256(visualSigningKey(sk, short), []byte(stringToSign)))
	req.Host = host
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", host)
	req.Header.Set("X-Content-Sha256", payload)
	req.Header.Set("X-Date", stamp)
	req.Header.Set("Authorization", "HMAC-SHA256 Credential="+ak+"/"+scope+", SignedHeaders="+signed+", Signature="+signature)
}

func visualSigningKey(secret, shortDate string) []byte {
	kDate := hmacSHA256([]byte(secret), []byte(shortDate))
	kRegion := hmacSHA256(kDate, []byte("cn-north-1"))
	kService := hmacSHA256(kRegion, []byte("cv"))
	return hmacSHA256(kService, []byte("request"))
}

func hmacSHA256(key, msg []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(msg)
	return h.Sum(nil)
}
