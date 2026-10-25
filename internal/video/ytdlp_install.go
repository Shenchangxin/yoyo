package video

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const ytdlpMaxBytes = 80 << 20

var ytdlpAssetURL = defaultYTDLPURL

func defaultYTDLPURL() string {
	switch runtime.GOOS {
	case "windows":
		return "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	case "darwin":
		return "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_macos"
	default:
		return "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"
	}
}

// EnsureYTDLP returns the local yt-dlp binary, downloading the current GitHub release when it is missing.
func (e *Engine) EnsureYTDLP(ctx context.Context) (string, error) {
	if e == nil {
		return "", fmt.Errorf("video engine is missing")
	}
	e.lookFilmTools()
	if strings.TrimSpace(e.YTDLP) != "" {
		return e.YTDLP, nil
	}
	return e.downloadYTDLP(ctx)
}

func (e *Engine) downloadYTDLP(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	dir := filepath.Join(e.Dir, "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := "yt-dlp"
	if runtime.GOOS == "windows" {
		name = "yt-dlp.exe"
	}
	dest := filepath.Join(dir, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ytdlpAssetURL(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Yoyo")
	var client HTTPDoer = e.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("yt-dlp download %d", res.StatusCode)
	}
	tmp, err := os.CreateTemp(dir, "yt-dlp-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	n, copyErr := io.Copy(tmp, io.LimitReader(res.Body, ytdlpMaxBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil || n == 0 || n > ytdlpMaxBytes {
		os.Remove(tmpName)
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return "", fmt.Errorf("yt-dlp download was empty or too large")
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	if err := os.Rename(tmpName, dest); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	e.YTDLP = dest
	return dest, nil
}
