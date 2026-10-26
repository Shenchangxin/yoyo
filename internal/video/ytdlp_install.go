// ytdlp_install.go 在视频目录的 bin 里准备 yt-dlp 可执行文件。

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

// ytdlpMaxBytes 是单次下载接受的最大字节数，80 MiB。读到的字节再多就拒绝。
const ytdlpMaxBytes = 80 << 20

// ytdlpAssetURL 是这次要下载的地址。测试会换掉它。
var ytdlpAssetURL = defaultYTDLPURL

// defaultYTDLPURL 按操作系统返回 GitHub latest 发布里的文件：Windows 是 yt-dlp.exe，darwin 是 yt-dlp_macos，其余是 yt-dlp。
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

// EnsureYTDLP 返回本机 yt-dlp 路径。先看已发现的路径，没有就下载当前发布。引擎为空时返回错误。
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

// downloadYTDLP 用 GET 把文件下到视频目录的 bin。上下文为空时用 Background，超时 5 分钟。请求带 User-Agent Yoyo；有 Engine.HTTP 就用它，否则用超时 5 分钟的 http.Client。状态码大于等于 300、空正文或超过 ytdlpMaxBytes 都失败并删掉临时文件。成功后把临时文件设为 0755，改名为 yt-dlp（Windows 上是 yt-dlp.exe），再记到 Engine.YTDLP。
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
