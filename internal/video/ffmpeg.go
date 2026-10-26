// ffmpeg.go 查找 ffmpeg，拼接成片，并抽出封面和压缩静帧。

package video

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// LookFFmpeg 在默认目录里找 ffmpeg 和 ffprobe。
func LookFFmpeg() (ffmpeg, ffprobe string) {
	return LookFFmpegIn(nil)
}

// LookFFmpegIn 先读 FFMPEG_BIN 和 FFPROBE_BIN。没有再在 extra、可执行文件旁、用户目录 .yoyo/video/bin、LOCALAPPDATA/yoyo/video/bin，以及 C:\ffmpeg\bin、C:\Program Files\ffmpeg\bin、/usr/local/bin、/opt/homebrew/bin 里找。Windows 上文件名带 .exe。extra 优先于 PATH。
func LookFFmpegIn(extra []string) (ffmpeg, ffprobe string) {
	if v := os.Getenv("FFMPEG_BIN"); v != "" {
		ffmpeg = v
	}
	if v := os.Getenv("FFPROBE_BIN"); v != "" {
		ffprobe = v
	}
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	cands := []string{dir, filepath.Join(dir, "bin"), filepath.Join(dir, "resources")}
	if home, err := os.UserHomeDir(); err == nil {
		cands = append(cands, filepath.Join(home, ".yoyo", "video", "bin"))
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		cands = append(cands, filepath.Join(local, "yoyo", "video", "bin"))
	}
	cands = append(cands,
		`C:\ffmpeg\bin`,
		`C:\Program Files\ffmpeg\bin`,
		`/usr/local/bin`,
		`/opt/homebrew/bin`,
	)
	name, probe := "ffmpeg", "ffprobe"
	if runtime.GOOS == "windows" {
		name += ".exe"
		probe += ".exe"
	}
	if ffmpeg == "" {
		ffmpeg = lookFile(name, extra)
	}
	if ffprobe == "" {
		ffprobe = lookFile(probe, extra)
	}
	if ffmpeg == "" {
		ffmpeg = look(name, cands)
	}
	if ffprobe == "" {
		ffprobe = look(probe, cands)
	}
	return ffmpeg, ffprobe
}

// look 先查 PATH，没有再在给定目录里找同名文件。
// name 传给 LookPath，没有再传给 lookFile。
// dirs 传给 lookFile。
func look(name string, dirs []string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return lookFile(name, dirs)
}

// lookFile 返回目录里第一个存在的普通文件。空目录跳过。
// name 接到每个目录后面再 Stat。
// dirs 里去掉空白后为空的项跳过。
func lookFile(name string, dirs []string) string {
	for _, d := range dirs {
		if strings.TrimSpace(d) == "" {
			continue
		}
		p := filepath.Join(d, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// FFmpegOK 在 FFMPEG 路径非空时为真。
func (e *Engine) FFmpegOK() bool {
	return e.FFMPEG != ""
}

// Concat 用后台上下文把多段拼成一个 mp4，并返回字节。
// paths 传给 ConcatContext。
func (e *Engine) Concat(paths []string) ([]byte, error) {
	return e.ConcatContext(context.Background(), paths)
}

// ConcatContext 先确保 ffmpeg 可用。任一片段不在磁盘上就返回 clip N is missing on disk。用 concat 列表拼接，视频 libx264、medium、crf 23，音频 aac、48 kHz、192k。带音频失败时去掉音频再拼。上下文已取消时返回该错误。
// ctx 为 nil 时换成 Background，并传给 EnsureFFmpeg 和 CommandContext。
// paths 逐个 Stat，不存在时返回 clip N is missing on disk，并写入 concat 列表。
func (e *Engine) ConcatContext(ctx context.Context, paths []string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := e.EnsureFFmpeg(ctx); err != nil {
		return nil, err
	}
	for i, p := range paths {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("clip %d is missing on disk", i+1)
		}
	}
	tmp, err := os.MkdirTemp(e.Dir, "merge-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	list := filepath.Join(tmp, "list.txt")
	var b strings.Builder
	for _, p := range paths {
		slash := filepath.ToSlash(p)
		fmt.Fprintf(&b, "file '%s'\n", strings.ReplaceAll(slash, `'`, `'\''`))
	}
	if err := os.WriteFile(list, []byte(b.String()), 0o600); err != nil {
		return nil, err
	}
	out := filepath.Join(tmp, "out.mp4")
	cmd := quiet(exec.CommandContext(ctx, e.FFMPEG, "-y", "-f", "concat", "-safe", "0", "-i", list,
		"-fflags", "+genpts", "-c:v", "libx264", "-preset", "medium", "-crf", "23",
		"-c:a", "aac", "-ar", "48000", "-b:a", "192k", "-movflags", "+faststart", out))
	cmd.Dir = tmp
	if _, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		cmd = quiet(exec.CommandContext(ctx, e.FFMPEG, "-y", "-f", "concat", "-safe", "0", "-i", list,
			"-fflags", "+genpts", "-c:v", "libx264", "-preset", "medium", "-crf", "23",
			"-an", "-movflags", "+faststart", out))
		cmd.Dir = tmp
		if outb, err := cmd.CombinedOutput(); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("%s", ffmpegErr(outb, err))
		}
	}
	return os.ReadFile(out)
}

// Poster 从视频抽出一帧 JPEG。ffmpeg 路径为空时返回 ffmpeg missing。-ss 放在 -i 前面，跳到关键帧，而不是从头解码。只取一小帧、不要音频，超时 8 秒，避免大文件把导入挂住。先缩到长边不超过 640，失败再取原尺寸。空文件视为失败。
// videoPath 作为 ffmpeg 的 -i 参数。
func (e *Engine) Poster(videoPath string) ([]byte, error) {
	if strings.TrimSpace(e.FFMPEG) == "" {
		return nil, fmt.Errorf("ffmpeg missing")
	}
	tmp, err := os.CreateTemp(e.Dir, "poster-*.jpg")
	if err != nil {
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	attempts := [][]string{
		{"-y", "-ss", "0.3", "-i", videoPath, "-frames:v", "1", "-an", "-threads", "1", "-vf", "scale='min(640,iw)':-2", tmp.Name()},
		{"-y", "-ss", "0.3", "-i", videoPath, "-frames:v", "1", "-an", "-threads", "1", tmp.Name()},
	}
	var last error
	for _, args := range attempts {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		cmd := quiet(exec.CommandContext(ctx, e.FFMPEG, args...))
		if err := cmd.Run(); err != nil {
			last = err
			continue
		}
		b, err := os.ReadFile(tmp.Name())
		if err != nil || len(b) == 0 {
			if err == nil {
				err = fmt.Errorf("empty poster")
			}
			last = err
			continue
		}
		return b, nil
	}
	if last == nil {
		last = fmt.Errorf("poster failed")
	}
	return nil, last
}

// CompressJPEG 解码图片。maxEdge 大于 0 且任一边更长时，把长边缩到 maxEdge，用 CatmullRom。quality 小于等于 0 时用 68。结果编成 JPEG。
// src 传给 image.Decode。
func CompressJPEG(src []byte, maxEdge int, quality int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("empty image")
	}
	if maxEdge > 0 && (w > maxEdge || h > maxEdge) {
		if w > h {
			h = h * maxEdge / w
			w = maxEdge
		} else {
			w = w * maxEdge / h
			h = maxEdge
		}
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
		img = dst
	}
	if quality <= 0 {
		quality = 68
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DataURLJPEG 返回 image/jpeg 的 base64 数据 URL。
// jpegBytes 编成 base64 后接在 data:image/jpeg;base64, 后面。
func DataURLJPEG(jpegBytes []byte) string {
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(jpegBytes)
}
