package update

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
)

// Manifest is signed out-of-band. The agent loop has no tool that can
// rotate keys or rewrite this package (TCB: updater is frozen).
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

type Asset struct {
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Kind     string `json:"kind,omitempty"`
}

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
