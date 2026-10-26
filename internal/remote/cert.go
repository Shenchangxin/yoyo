// 本文件拥有自签证书和 TLS 监听。证书 DNS 名只有 yoyo.local，模板里没有 IP。
// 不接待浏览器。listen 调用 loadCert 和 listenTLS。指纹写进二维码，给手机钉证书。

package remote

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// loadCert 优先读取目录里的 cert.pem 和 key.pem。文件能装上但叶子解不开，就返回错误，不另签一张。
// 没有可用的一对时，用 P-256 新签一张：主体 yoyo-remote，DNS 名只有 yoyo.local，有效期从现在往前 1 小时到往后 10 年，用途是数字签名、密钥加密和服务器鉴别。
// 证书文件权限 0o644，私钥权限 0o600。返回的指纹是叶子证书 DER 的 SHA-256 十六进制。
// dir 和 cert.pem、key.pem 拼成要读写的路径。
func loadCert(dir string) (tls.Certificate, string, error) {
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if pair, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil && len(pair.Certificate) > 0 {
		leaf, err := x509.ParseCertificate(pair.Certificate[0])
		if err != nil {
			return tls.Certificate{}, "", err
		}
		pair.Leaf = leaf
		return pair, fingerprint(leaf.Raw), nil
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "yoyo-remote"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"yoyo.local"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return tls.Certificate{}, "", err
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return tls.Certificate{}, "", err
	}
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	pair.Leaf = leaf
	return pair, fingerprint(der), nil
}

// fingerprint 把证书 DER 做 SHA-256，再写成十六进制。二维码里的 fp 就是这个字符串，手机 TLS 用它钉证书。
// raw 传给 sha256.Sum256，结果写成十六进制。
func fingerprint(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// listenTLS 在所有网卡的这个端口上听 TLS，最低版本 TLS 1.2。这是手机 WebSocket 和媒体的门，不是浏览器那扇明文 HTTP。
// port 写进监听地址。
// cert 放进 tls.Config 的 Certificates。
func listenTLS(port int, cert tls.Certificate) (net.Listener, error) {
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	return tls.Listen("tcp", fmt.Sprintf(":%d", port), cfg)
}
