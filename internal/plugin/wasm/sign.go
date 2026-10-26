// 用 ed25519 校验 WASM 字节。本文件不读取 YOYO_WASM_PUBKEY，也不把模块写入 active。

package wasm

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
)

// Verify 用 ed25519 校验原始 WASM 字节。公钥长度不是 ed25519.PublicKeySize（32）时返回 "wasm: missing public key"。签名长度不是 ed25519.SignatureSize（64）时返回 "wasm: bad signature length"。验签失败返回 "wasm: signature rejected"。未签名模块不会在这里被拦住：Host.Load 不调用本函数，未签名字节可以装入宿主。本文件没有写入 active 的路径。
// module 和公钥、签名一起交给 ed25519.Verify。
func Verify(pub ed25519.PublicKey, module, sig []byte) error {
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("wasm: missing public key")
	}
	if len(sig) != ed25519.SignatureSize {
		return fmt.Errorf("wasm: bad signature length")
	}
	if !ed25519.Verify(pub, module, sig) {
		return fmt.Errorf("wasm: signature rejected")
	}
	return nil
}

// Sign 用 ed25519 私钥对模块字节签名。
// priv 和 module 一起交给 ed25519.Sign。
// module 作为被签名的字节。
func Sign(priv ed25519.PrivateKey, module []byte) []byte {
	return ed25519.Sign(priv, module)
}

// ParseKey 把十六进制解成 32 字节公钥。失败返回 "wasm: bad public key"。
// raw 用 hex.DecodeString 解开，长度不是 ed25519.PublicKeySize 时返回错误。
func ParseKey(raw string) (ed25519.PublicKey, error) {
	b, err := hex.DecodeString(raw)
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("wasm: bad public key")
	}
	return ed25519.PublicKey(b), nil
}

// ParseSig 把十六进制解成 64 字节签名。失败返回 "wasm: bad signature hex"。
// raw 用 hex.DecodeString 解开，长度不是 ed25519.SignatureSize 时返回错误。
func ParseSig(raw string) ([]byte, error) {
	b, err := hex.DecodeString(raw)
	if err != nil || len(b) != ed25519.SignatureSize {
		return nil, fmt.Errorf("wasm: bad signature hex")
	}
	return b, nil
}
