package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
	if url == "" {
		return nil, fmt.Errorf("update: empty url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("update: http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}
