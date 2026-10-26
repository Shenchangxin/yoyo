// 用 ed25519 验更新清单的字节。

package update

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
)

// Manifest 是更新清单：版本、通道、可选说明，以及按平台给出的资源。
type Manifest struct {
	Schema    int              `json:"schema,omitempty"`
	Version   string           `json:"version"`
	Channel   string           `json:"channel"`
	PubDate   string           `json:"pub_date,omitempty"`
	Notes     string           `json:"notes,omitempty"`
	Mandatory bool             `json:"mandatory,omitempty"`
	SHA256    string           `json:"sha256,omitempty"`
	URL       string           `json:"url,omitempty"`
	Platforms map[string]Asset `json:"platforms,omitempty"`
}

// Asset 是某个平台的下载地址、文件名、SHA-256、大小和种类。
type Asset struct {
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Kind     string `json:"kind,omitempty"`
}

// ParseManifest 解析清单 JSON。缺少 version 时返回 "update: manifest missing version"。
func ParseManifest(b []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return m, err
	}
	if m.Version == "" {
		return m, fmt.Errorf("update: manifest missing version")
	}
	return m, nil
}

// Verify 用 ed25519 验清单字节。公钥长度不对返回 "update: bad public key"，正文为空返回 "update: empty body"，签名长度不对返回 "update: bad signature"，验签失败返回 "update: signature rejected"。
func Verify(pub ed25519.PublicKey, payload, sig []byte) error {
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("update: bad public key")
	}
	if len(payload) == 0 {
		return fmt.Errorf("update: empty body")
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("update: bad signature")
	}
	if !ed25519.Verify(pub, payload, sig) {
		return fmt.Errorf("update: signature rejected")
	}
	return nil
}

// Sign 用 ed25519 私钥对清单字节签名。
func Sign(priv ed25519.PrivateKey, payload []byte) []byte {
	return ed25519.Sign(priv, payload)
}
