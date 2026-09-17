package wasm

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
)

// Verify checks an ed25519 signature over raw WASM bytes. Unsigned modules
// must never be checked out to refs/active.
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

func Sign(priv ed25519.PrivateKey, module []byte) []byte {
	return ed25519.Sign(priv, module)
}

func ParseKey(raw string) (ed25519.PublicKey, error) {
	b, err := hex.DecodeString(raw)
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("wasm: bad public key")
	}
	return ed25519.PublicKey(b), nil
}

func ParseSig(raw string) ([]byte, error) {
	b, err := hex.DecodeString(raw)
	if err != nil || len(b) != ed25519.SignatureSize {
		return nil, fmt.Errorf("wasm: bad signature hex")
	}
	return b, nil
}
