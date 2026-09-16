package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const EnvPubkey = "YOYO_UPDATE_PUBKEY"

// PublicKeyFromEnv reads a hex or standard-base64 ed25519 public key.
// Empty means check-only against already-staged bytes; never apply automatically.
func PublicKeyFromEnv() ed25519.PublicKey {
	raw := strings.TrimSpace(os.Getenv(EnvPubkey))
	if raw == "" {
		return nil
	}
	if b, err := hex.DecodeString(raw); err == nil && len(b) == ed25519.PublicKeySize {
		return ed25519.PublicKey(b)
	}
	if b, err := base64.StdEncoding.DecodeString(raw); err == nil && len(b) == ed25519.PublicKeySize {
		return ed25519.PublicKey(b)
	}
	return nil
}

// CheckResult is advisory. Applying a binary is a human/L3 action; this
// package never overwrites the running executable.
type CheckResult struct {
	Current  Status   `json:"current"`
	Manifest Manifest `json:"manifest"`
	Newer    bool     `json:"newer"`
	Verified bool     `json:"verified"`
}

func Check(ctx context.Context, manifestURL string, payload, sig []byte, pub ed25519.PublicKey) (CheckResult, error) {
	res := CheckResult{Current: Current("")}
	var err error
	if len(payload) == 0 {
		payload, err = httpGet(ctx, manifestURL)
		if err != nil {
			return res, err
		}
	}
	if len(sig) == 0 && manifestURL != "" {
		sig, err = httpGet(ctx, strings.TrimRight(manifestURL, "/")+".sig")
		if err != nil {
			return res, err
		}
	}
	if err := Verify(pub, payload, sig); err != nil {
		return res, err
	}
	m, err := ParseManifest(payload)
	if err != nil {
		return res, err
	}
	res.Manifest = m
	res.Verified = true
	res.Newer = m.Version != "" && m.Version != res.Current.Version
	return res, nil
}

// Stage writes a verified payload to destDir. It never replaces argv[0].
func Stage(destDir string, bin []byte, wantSHA string) (string, error) {
	sum := sha256.Sum256(bin)
	got := hex.EncodeToString(sum[:])
	if wantSHA != "" && !strings.EqualFold(wantSHA, got) {
		return "", fmt.Errorf("update: sha256 mismatch")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(destDir, "yoyo.staging")
	if err := os.WriteFile(path, bin, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	return Fetch(ctx, url, 1<<20)
}

// Fetch downloads url up to max bytes. It never writes argv[0].
func Fetch(ctx context.Context, url string, max int64) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("update: empty url")
	}
	if max <= 0 {
		max = 1 << 20
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("update: http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, fmt.Errorf("update: payload too large")
	}
	return b, nil
}
