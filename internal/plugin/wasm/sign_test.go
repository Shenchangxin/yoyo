// 检查 ed25519 签名往返、篡改拒绝和空公钥拒绝。

package wasm

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
)

// TestSignVerifyRoundTrip 确认签名能验过，多一个字节的模块被拒绝，非法十六进制公钥被拒绝，合法公钥和签名能解析。
func TestSignVerifyRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	mod := []byte{0x00, 0x61, 0x73, 0x6d}
	sig := Sign(priv, mod)
	if err := Verify(pub, mod, sig); err != nil {
		t.Fatal(err)
	}
	if err := Verify(pub, append(mod, 0x00), sig); err == nil {
		t.Fatal("tampered module accepted")
	}
	if _, err := ParseKey("zz"); err == nil {
		t.Fatal("bad key")
	}
	if _, err := ParseKey(hex.EncodeToString(pub)); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseSig(hex.EncodeToString(sig)); err != nil {
		t.Fatal(err)
	}
}

// TestVerifyRejectsEmptyKey 确认公钥为 nil 时 Verify 返回错误。
func TestVerifyRejectsEmptyKey(t *testing.T) {
	if err := Verify(nil, []byte("m"), make([]byte, ed25519.SignatureSize)); err == nil {
		t.Fatal("empty key")
	}
}
