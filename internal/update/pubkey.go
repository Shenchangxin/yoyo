package update

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

const EnvPubkey = "YOYO_UPDATE_PUBKEY"
const EnvPrivkey = "YOYO_UPDATE_PRIVKEY"

// EmbeddedPubkeyHex is the compiled-in Ed25519 public key (32-byte hex).
// Override at test/runtime with YOYO_UPDATE_PUBKEY. Rotate by replacing this
// value and the GitHub secret YOYO_UPDATE_PRIVKEY together.
var EmbeddedPubkeyHex = "92d0f1163610b8a807d9e4641c9f8cdde8dd53dc992756c438b3368c26206e66"

// PublicKeyFromEnv reads a hex or standard-base64 ed25519 public key.
// Empty means the compiled-in key should be used via PublicKey.
func PublicKeyFromEnv() ed25519.PublicKey {
	return parsePublicKey(os.Getenv(EnvPubkey))
}

// PublicKey returns YOYO_UPDATE_PUBKEY if set, otherwise the compiled-in key.
func PublicKey() ed25519.PublicKey {
	if pub := PublicKeyFromEnv(); len(pub) == ed25519.PublicKeySize {
		return pub
	}
	return parsePublicKey(EmbeddedPubkeyHex)
}

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

// ParsePrivateKey reads a hex or standard-base64 ed25519 seed (32) or key (64).
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

func PrivateKeyFromEnv() (ed25519.PrivateKey, error) {
	return ParsePrivateKey(os.Getenv(EnvPrivkey))
}
