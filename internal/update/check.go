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
	"sync"
	"time"
)

const maxManifestBytes = 2 << 20

var (
	httpMu     sync.Mutex
	httpClient = defaultHTTPClient()
)

func defaultHTTPClient() *http.Client {
	// Installers can be hundreds of MB; JSON checks rely on the request context.
	return &http.Client{Timeout: 30 * time.Minute}
}

func SetHTTPClient(c *http.Client) {
	httpMu.Lock()
	defer httpMu.Unlock()
	if c == nil {
		httpClient = defaultHTTPClient()
		return
	}
	httpClient = c
}

func client() *http.Client {
	httpMu.Lock()
	defer httpMu.Unlock()
	return httpClient
}

// CheckResult is advisory. Applying a package is a human/L3 action; this
// package never overwrites the running executable on its own.
type CheckResult struct {
	Current  Status   `json:"current"`
	Manifest Manifest `json:"manifest"`
	Asset    Asset    `json:"asset,omitempty"`
	Newer    bool     `json:"newer"`
	Verified bool     `json:"verified"`
}

func Check(ctx context.Context, manifestURL string, payload, sig []byte, pub ed25519.PublicKey) (CheckResult, error) {
	res := CheckResult{Current: Current("")}
	var err error
	if len(payload) == 0 {
		payload, err = httpGet(ctx, manifestURL, true)
		if err != nil {
			return res, err
		}
	}
	if len(sig) == 0 && manifestURL != "" {
		sig, err = httpGet(ctx, SigURL(manifestURL), true)
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
	res.Newer = IsNewer(m.Version, res.Current.Version)
	if asset, ok := m.SelectAsset(DesktopOS()); ok {
		res.Asset = asset
	}
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
	path := filepath.Join(destDir, StagingName())
	if err := os.WriteFile(path, bin, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func httpGet(ctx context.Context, rawURL string, nocache bool) ([]byte, error) {
	return Fetch(ctx, rawURL, maxManifestBytes, nocache)
}

// Fetch downloads url up to max bytes into memory. It never writes argv[0].
// ModelScope resolve URLs fall back to the dataset API on failure.
func Fetch(ctx context.Context, rawURL string, max int64, nocache bool) ([]byte, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("update: empty url")
	}
	if max <= 0 {
		max = 1 << 20
	}
	var last error
	for _, u := range candidateURLs(rawURL) {
		b, err := fetchOnce(ctx, u, max, nocache)
		if err == nil {
			return b, nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("update: fetch failed")
	}
	return nil, last
}

func fetchOnce(ctx context.Context, rawURL string, max int64, nocache bool) ([]byte, error) {
	resp, err := doGET(ctx, rawURL, nocache, 0)
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

func doGET(ctx context.Context, rawURL string, nocache bool, start int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UpdaterUA)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", Referer())
	if nocache {
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Pragma", "no-cache")
	}
	if start > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", start))
	}
	return client().Do(req)
}
