package update

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
)

// Manifest is signed out-of-band. The agent loop has no tool that can
// rotate keys or rewrite this package (TCB: updater is frozen).
type Manifest struct {
	Version string `json:"version"`
	Channel string `json:"channel"`
	SHA256  string `json:"sha256"`
}

func ParseManifest(b []byte) (Manifest, error) {
	var m Manifest
	return m, json.Unmarshal(b, &m)
}

func Verify(pub ed25519.PublicKey, payload, sig []byte) error {
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("update: bad public key")
	}
	if !ed25519.Verify(pub, payload, sig) {
		return fmt.Errorf("update: signature rejected")
	}
	return nil
}

func Sign(priv ed25519.PrivateKey, payload []byte) []byte {
	return ed25519.Sign(priv, payload)
}
