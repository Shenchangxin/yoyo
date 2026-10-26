// 读取编译进二进制的更新公钥，或被环境变量覆盖。

package update

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// EnvPubkey 是公钥环境变量名 YOYO_UPDATE_PUBKEY。
const EnvPubkey = "YOYO_UPDATE_PUBKEY"

// EnvPrivkey 是私钥环境变量名 YOYO_UPDATE_PRIVKEY。
const EnvPrivkey = "YOYO_UPDATE_PRIVKEY"

// EmbeddedPubkeyHex 是编译进二进制的 Ed25519 公钥，32 字节十六进制。PublicKey 在 YOYO_UPDATE_PUBKEY 能解出 32 字节时用环境变量，否则用这个值。
var EmbeddedPubkeyHex = "34eac26cf404348ba84c1926805745dae32d2e97a98c729569d9ef783cf0a01b"

// PublicKeyFromEnv 解析 YOYO_UPDATE_PUBKEY。空或解析失败时返回 nil。
func PublicKeyFromEnv() ed25519.PublicKey {
	return parsePublicKey(os.Getenv(EnvPubkey))
}

// PublicKey 在环境变量能解出 32 字节公钥时用它，否则用 EmbeddedPubkeyHex。
func PublicKey() ed25519.PublicKey {
	if pub := PublicKeyFromEnv(); len(pub) == ed25519.PublicKeySize {
		return pub
	}
	return parsePublicKey(EmbeddedPubkeyHex)
}

// parsePublicKey 接受十六进制、标准 base64 或 RawStd base64，长度必须是 32 字节。失败返回 nil。
// raw 去掉两端空白，空则返回 nil；否则按十六进制、标准 base64、RawStd base64 试，长度必须是 ed25519.PublicKeySize。
func parsePublicKey(raw string) ed25519.PublicKey {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if b, err := hex.DecodeString(raw); err == nil && len(b) == ed25519.PublicKeySize {
		return ed25519.PublicKey(b)
	}
	if b, err := base64.StdEncoding.DecodeString(raw); err == nil && len(b) == ed25519.PublicKeySize {
		return ed25519.PublicKey(b)
	}
	if b, err := base64.RawStdEncoding.DecodeString(raw); err == nil && len(b) == ed25519.PublicKeySize {
		return ed25519.PublicKey(b)
	}
	return nil
}

// ParsePrivateKey 先按十六进制再按标准 base64 解析。32 字节当作种子，64 字节当作私钥。空串返回 "update: empty private key"。其他长度返回 "update: private key length"。
// raw 去掉两端空白，空则返回错误；先按十六进制再按标准 base64 解。
func ParsePrivateKey(raw string) (ed25519.PrivateKey, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("update: empty private key")
	}
	var b []byte
	var err error
	if b, err = hex.DecodeString(raw); err != nil {
		b, err = base64.StdEncoding.DecodeString(raw)
	}
	if err != nil {
		return nil, fmt.Errorf("update: private key: %w", err)
	}
	switch len(b) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(b), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(b), nil
	default:
		return nil, fmt.Errorf("update: private key length %d", len(b))
	}
}

// PrivateKeyFromEnv 解析 YOYO_UPDATE_PRIVKEY。
func PrivateKeyFromEnv() (ed25519.PrivateKey, error) {
	return ParsePrivateKey(os.Getenv(EnvPrivkey))
}
