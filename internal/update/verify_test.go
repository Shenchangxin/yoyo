// 检查清单签名：完整签名通过，截断或改过的正文被拒绝。

package update

import (
	"crypto/ed25519"
	"testing"
)

// TestVerifyRejectsBadSig 确认合法签名通过，截短的签名和改过的正文被拒绝。
func TestVerifyRejectsBadSig(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"version":"0.1.0","channel":"nightly","sha256":"abc"}`)
	sig := Sign(priv, payload)
	if err := Verify(pub, payload, sig); err != nil {
		t.Fatal(err)
	}
	if err := Verify(pub, payload, sig[:len(sig)-1]); err == nil {
		t.Fatal("truncated sig accepted")
	}
	if err := Verify(pub, append(payload, 'x'), sig); err == nil {
		t.Fatal("mutated payload accepted")
	}
}
