package wasm

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
)

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

func TestVerifyRejectsEmptyKey(t *testing.T) {
	if err := Verify(nil, []byte("m"), make([]byte, ed25519.SignatureSize)); err == nil {
		t.Fatal("empty key")
	}
}
