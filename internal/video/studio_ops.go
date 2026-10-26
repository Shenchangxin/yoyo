// 短片合成：把每一镜编码、接起来、混上口播和配乐、烧上字幕，再放进素材库。
// 另有改画幅、烧字幕和相册幻灯。没有 ffmpeg 就不编码。

package video

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// parseSize 从「宽x高」读像素。读不出就按画幅给尺寸。宽和高都会收成偶数。
func parseSize(size, aspect string) (int, int) {
	size = strings.ToLower(strings.TrimSpace(size))
	if i := strings.Index(size, "x"); i > 0 {
		w, _ := strconv.Atoi(size[:i])
		h, _ := strconv.Atoi(size[i+1:])
		if w > 0 && h > 0 {
			return even(w), even(h)
		}
	}
	return aspectSize(aspect)
}

// even 把奇数加一，偶数原样返回。编码要求宽高是偶数。
func even(n int) int {
	if n%2 != 0 {
		return n + 1
	}
	return n
}

// materialize 把内容库里的一份对象写成临时文件。扩展名空着就用 .bin。
func (e *Engine) materialize(hash, ext string) (string, error) {
	if ext == "" {
		ext = ".bin"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	if src := e.casFile(hash); src != "" {
		in, err := os.Open(src)
		if err != nil {
			return "", err
		}
		defer in.Close()
		return writeTemp(e.Dir, ext, in)
	}
	// 磁盘上没有这份对象时，字节在内存库里。测试会走到这里。
	raw, err := e.GetBytes(hash)
	if err != nil {
		return "", err
	}
	return writeTemp(e.Dir, ext, bytes.NewReader(raw))
}

// writeTemp 把读到的字节写进临时文件。拷贝或关闭失败时删掉这个文件并返回错误。
func writeTemp(dir, ext string, r io.Reader) (string, error) {
	f, err := os.CreateTemp(dir, "studio-*"+ext)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// AssembleStudio 把分镜合成一条 mp4。没有镜头，或某一镜已过期但仍带着图或视频，就拒绝。没有 ffmpeg 也不编码。
// 视频会循环补到镜头时长，短到要循环不止一次就记一条警告。字幕烧录失败时仍保留没烧字幕的成片。成功后抽出海报，追加版本，并放进素材库。放进素材库失败时，成片编号已经写上。
func (e *Engine) AssembleStudio(id string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	sb := p.Storyboard
	if p.NarrationHash != "" {
		sb.Narration = p.NarrationHash
	}
	if p.SubtitleHash != "" {
		sb.Subtitle = p.SubtitleHash
	}
	if p.BGMHash != "" {
		sb.BGM = p.BGMHash
	}
	if len(sb.Shots) == 0 {
		return StudioProject{}, fmt.Errorf("storyboard has no shots")
	}
	for _, shot := range sb.Shots {
		if shot.Stale && (strings.TrimSpace(shot.ImageHash) != "" || strings.TrimSpace(shot.VideoHash) != "") {
			return StudioProject{}, fmt.Errorf("shot %s is stale; regenerate before assemble", firstNonEmpty(shot.ID, shot.Caption))
		}
	}
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return StudioProject{}, err
	}
	w, h := parseSize(sb.Size, p.Aspect)
	tmp, err := os.MkdirTemp(e.Dir, "assemble-*")
	if err != nil {
		return StudioProject{}, err
	}
	defer os.RemoveAll(tmp)

	var clips []string
	for i, shot := range sb.Shots {
		dur := shot.Duration
		if dur <= 0 {
			dur = 3
		}
		clip := filepath.Join(tmp, fmt.Sprintf("shot-%02d.mp4", i))
		motion := normalizeMotion(shot.Motion)
		if shot.Motion == "" {
			motion = normalizeMotion(sb.ImageMotion)
		}
		fit := shotFit(sb, shot, motion)
		if shot.VideoHash != "" {
			src, err := e.materialize(shot.VideoHash, ".mp4")
			if err != nil {
				p.Status, p.Error = StudioFailed, err.Error()
				_ = e.putDoc(colStudioProjects, p.ID, p)
				return p, err
			}
			defer os.Remove(src)
			if e.FFProbe != "" {
				if srcDur, perr := probeFileSeconds(e.FFProbe, src); perr == nil {
					if _, warn := clipShouldLoop(srcDur, dur); warn {
						label := firstNonEmpty(shot.Caption, shot.ID)
						e.warnStudio(&p, GateCompose, fmt.Sprintf("镜头「%s」只有 %.1f 秒，已循环补到 %.1f 秒。", label, srcDur, dur))
					}
				}
			}
			vf := frameFilter(w, h, fit)
			if err := e.encodeShot(clip, shotCacheKey("v", shot.VideoHash, dur, motion, fit, w, h), func() error {
				cmd := quiet(exec.Command(e.FFMPEG, "-y", "-stream_loop", "-1", "-i", src, "-t", fmt.Sprintf("%.2f", dur), "-vf", vf,
					"-c:v", "libx264", "-pix_fmt", "yuv420p", "-an", clip))
				out, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("%s", ffmpegErr(out, err))
				}
				return nil
			}); err != nil {
				p.Status, p.Error = StudioFailed, err.Error()
				_ = e.putDoc(colStudioProjects, p.ID, p)
				return p, err
			}
		} else if shot.ImageHash != "" {
			src, err := e.materialize(shot.ImageHash, ".png")
			if err != nil {
				p.Status, p.Error = StudioFailed, err.Error()
				_ = e.putDoc(colStudioProjects, p.ID, p)
				return p, err
			}
			defer os.Remove(src)
			vf := stillFilter(w, h, dur, motion, fit)
			if err := e.encodeShot(clip, shotCacheKey("i", shot.ImageHash, dur, motion, fit, w, h), func() error {
				args := []string{"-y", "-loop", "1", "-i", src, "-t", fmt.Sprintf("%.2f", dur), "-vf", vf,
					"-c:v", "libx264", "-pix_fmt", "yuv420p", "-an", clip}
				cmd := quiet(exec.Command(e.FFMPEG, args...))
				out, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Errorf("%s", ffmpegErr(out, err))
				}
				return nil
			}); err != nil {
				p.Status, p.Error = StudioFailed, err.Error()
				_ = e.putDoc(colStudioProjects, p.ID, p)
				return p, err
			}
		} else {
			continue
		}
		clips = append(clips, clip)
	}
	if len(clips) == 0 {
		return StudioProject{}, fmt.Errorf("no usable shots (need image_hash or video_hash)")
	}
	merged, err := e.Concat(clips)
	if err != nil {
		p.Status, p.Error = StudioFailed, err.Error()
		_ = e.putDoc(colStudioProjects, p.ID, p)
		return p, err
	}
	base := filepath.Join(tmp, "base.mp4")
	if err := os.WriteFile(base, merged, 0o600); err != nil {
		return StudioProject{}, err
	}
	final := base
	if sb.Narration != "" || sb.BGM != "" {
		mixed, err := e.mixAudio(tmp, base, sb.Narration, sb.BGM, sb.BGMVolume)
		if err != nil {
			p.Status, p.Error = StudioFailed, err.Error()
			_ = e.putDoc(colStudioProjects, p.ID, p)
			return p, err
		}
		final = mixed
	}
	if sb.Subtitle != "" {
		burned, err := e.burnSubtitles(tmp, final, sb.Subtitle, w, h)
		if err == nil {
			final = burned
		}
	}
	raw, err := os.ReadFile(final)
	if err != nil {
		return StudioProject{}, err
	}
	hash, err := e.PutBytes(raw)
	if err != nil {
		return StudioProject{}, err
	}
	p.ResultHash = hash
	p.Status = StudioAssembled
	p.Error = ""
	p.Gate = GateCompose
	if poster, err := e.Poster(final); err == nil {
		if ph, err := e.PutBytes(poster); err == nil {
			p.PosterHash = ph
			if p.CoverHash == "" {
				p.CoverHash = ph
			}
		}
	}
	e.noteStudio(&p, "assemble", "cut ready")
	p.Versions = append(p.Versions, StudioSnapshot{
		N: len(p.Versions) + 1, ResultHash: p.ResultHash, PosterHash: p.PosterHash,
		NarrationHash: p.NarrationHash, SubtitleHash: p.SubtitleHash, CoverHash: p.CoverHash,
		Note: "assemble", CreatedAt: Now(),
	})
	saved, err := e.saveStudio(p)
	if err != nil {
		return p, err
	}
	published, err := e.PublishStudioAsset(saved)
	if err != nil {
		_ = e.putDoc(colStudioProjects, saved.ID, saved)
		return saved, err
	}
	return published, nil
}

// shotCacheKey 用种类、内容编号、时长、运镜、适配和尺寸算出镜头缓存的键。
func shotCacheKey(kind, hash string, dur float64, motion, fit string, w, h int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%.3f|%s|%s|%d|%d", kind, hash, dur, motion, fit, w, h)))
	return hex.EncodeToString(sum[:])
}

// encodeShot 先抄缓存里的成片。没有缓存或拷贝失败就重新编码，成功后再写入缓存。
func (e *Engine) encodeShot(dest, key string, encode func() error) error {
	if cached := e.shotCacheFile(key); cached != "" {
		in, err := os.Open(cached)
		if err == nil {
			defer in.Close()
			out, err := os.Create(dest)
			if err == nil {
				_, copyErr := io.Copy(out, in)
				closeErr := out.Close()
				if copyErr == nil && closeErr == nil {
					return nil
				}
			}
		}
	}
	if err := encode(); err != nil {
		return err
	}
	e.storeShotCache(key, dest)
	return nil
}

// shotCacheFile 返回缓存文件路径。目录空着、键空着，或文件小于 32 字节，就返回空串。
func (e *Engine) shotCacheFile(key string) string {
	if strings.TrimSpace(e.Dir) == "" || key == "" {
		return ""
	}
	p := filepath.Join(e.Dir, "shot-cache", key+".mp4")
	st, err := os.Stat(p)
	if err != nil || st.Size() < 32 {
		return ""
	}
	return p
}

// storeShotCache 把刚编好的镜头抄进 shot-cache。先写临时文件，再改名。任何一步失败都留下原片，不把半截文件当成缓存。
func (e *Engine) storeShotCache(key, src string) {
	if strings.TrimSpace(e.Dir) == "" || key == "" {
		return
	}
	dir := filepath.Join(e.Dir, "shot-cache")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	tmp := filepath.Join(dir, key+".part")
	out, err := os.Create(tmp)
	if err != nil {
		return
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(tmp)
		return
	}
	_ = os.Rename(tmp, filepath.Join(dir, key+".mp4"))
}

// mixAudio 把口播和配乐混进成片。两者都空就原样返回视频。配乐音量小于等于 0 时用 0.22。口播和配乐同时在时压低配乐。成片长过 3.2 秒时配乐在结尾淡出 3 秒。
func (e *Engine) mixAudio(tmp, video, narrationHash, bgmHash string, bgmVol float64) (string, error) {
	if narrationHash == "" && bgmHash == "" {
		return video, nil
	}
	var narr, bgm string
	var err error
	if narrationHash != "" {
		narr, err = e.materialize(narrationHash, ".mp3")
		if err != nil {
			return "", err
		}
		defer os.Remove(narr)
	}
	if bgmHash != "" {
		bgm, err = e.materialize(bgmHash, ".mp3")
		if err != nil {
			return "", err
		}
		defer os.Remove(bgm)
	}
	if bgmVol <= 0 {
		bgmVol = 0.22
	}
	out := filepath.Join(tmp, "mixed.mp4")
	opts := cut.MixOpts{
		Duck:        narr != "" && bgm != "",
		ReplaceOrig: narr != "",
		MusicVol:    bgmVol,
		SpeechVol:   1,
	}
	if bgm != "" && e.FFProbe != "" {
		if sec, err := probeFileSeconds(e.FFProbe, video); err == nil && sec > 3.2 {
			opts.FadeOut = 3
			opts.Timeline = sec
		}
	}
	if err := e.cutBin().Mix(video, narr, bgm, out, opts); err != nil {
		return "", err
	}
	return out, nil
}

// burnSubtitles 把字幕烧进成片。能转成 ASS 就烧 ASS，否则烧原来的 SRT。宽高不知道时先探测画面。
func (e *Engine) burnSubtitles(tmp, video, subHash string, w, h int) (string, error) {
	src, err := e.materialize(subHash, ".srt")
	if err != nil {
		return "", err
	}
	defer os.Remove(src)
	burnPath := src
	if w <= 0 || h <= 0 {
		w, h = e.probeFrame(video)
	}
	if raw, err := os.ReadFile(src); err == nil {
		if ass := srtToASS(string(raw), w, h); ass != "" {
			assPath := filepath.Join(tmp, "captions.ass")
			if err := os.WriteFile(assPath, []byte(ass), 0o600); err == nil {
				burnPath = assPath
			}
		}
	}
	out := filepath.Join(tmp, "sub.mp4")
	vf := subtitleFilter(burnPath)
	cmd := quiet(exec.Command(e.FFMPEG, "-y", "-i", video, "-vf", vf, "-c:a", "copy", out))
	if b, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%s", ffmpegErr(b, err))
	}
	return out, nil
}

// subtitleFilter 拼 ffmpeg 的 subtitles 滤镜。冒号和单引号会转义，并指定 UTF-8。
func subtitleFilter(path string) string {
	slash := filepath.ToSlash(path)
	slash = strings.ReplaceAll(slash, ":", `\:`)
	slash = strings.ReplaceAll(slash, "'", `\'`)
	return "subtitles='" + slash + "':charenc=UTF-8"
}

// probeFrame 用 ffprobe 读视频宽高。读不到或小于 16 就返回 1080×1920。
func (e *Engine) probeFrame(path string) (int, int) {
	if e.FFProbe == "" {
		return 1080, 1920
	}
	cmd := quiet(exec.Command(e.FFProbe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0:s=x", path))
	b, err := cmd.CombinedOutput()
	if err != nil {
		return 1080, 1920
	}
	parts := strings.Split(strings.TrimSpace(string(b)), "x")
	if len(parts) != 2 {
		return 1080, 1920
	}
	w, _ := strconv.Atoi(parts[0])
	h, _ := strconv.Atoi(parts[1])
	if w < 16 || h < 16 {
		return 1080, 1920
	}
	return even(w), even(h)
}

// Reframe 把一条视频垫成指定画幅，不裁切主体。先尝试保留音轨，失败再去掉音轨。结果放进内容库。
func (e *Engine) Reframe(hash, aspect string) (MediaRef, error) {
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return MediaRef{}, err
	}
	src, err := e.materialize(hash, ".mp4")
	if err != nil {
		return MediaRef{}, err
	}
	defer os.Remove(src)
	w, h := aspectSize(aspect)
	tmp, err := os.CreateTemp(e.Dir, "reframe-*.mp4")
	if err != nil {
		return MediaRef{}, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	vf := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1", w, h, w, h)
	cmd := quiet(exec.Command(e.FFMPEG, "-y", "-i", src, "-vf", vf, "-c:a", "copy", tmp.Name()))
	if b, err := cmd.CombinedOutput(); err != nil {
		cmd = quiet(exec.Command(e.FFMPEG, "-y", "-i", src, "-vf", vf, "-an", tmp.Name()))
		if b2, err2 := cmd.CombinedOutput(); err2 != nil {
			return MediaRef{}, fmt.Errorf("%s", ffmpegErr(append(b, b2...), err2))
		}
	}
	raw, err := os.ReadFile(tmp.Name())
	if err != nil {
		return MediaRef{}, err
	}
	out, err := e.PutBytes(raw)
	if err != nil {
		return MediaRef{}, err
	}
	return MediaRef{Kind: "video", CASHash: out, MIME: "video/mp4", Width: w, Height: h, Source: "studio"}, nil
}

// BurnInCaptions 把一份字幕烧进一条视频，结果放进内容库。没有 ffmpeg 就拒绝。
func (e *Engine) BurnInCaptions(hash, subHash string) (MediaRef, error) {
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return MediaRef{}, err
	}
	tmp, err := os.MkdirTemp(e.Dir, "burn-*")
	if err != nil {
		return MediaRef{}, err
	}
	defer os.RemoveAll(tmp)
	src, err := e.materialize(hash, ".mp4")
	if err != nil {
		return MediaRef{}, err
	}
	defer os.Remove(src)
	out, err := e.burnSubtitles(tmp, src, subHash, 0, 0)
	if err != nil {
		return MediaRef{}, err
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return MediaRef{}, err
	}
	cas, err := e.PutBytes(raw)
	if err != nil {
		return MediaRef{}, err
	}
	return MediaRef{Kind: "video", CASHash: cas, MIME: "video/mp4", Source: "studio"}, nil
}

// Slideshow 用几张图做一条肯伯恩斯相册短片。会先建一条标题为「相册短片」的工程再合成。时长小于等于 0 时每张 3 秒。
func (e *Engine) Slideshow(hashes []string, aspect string, dur float64) (MediaRef, error) {
	p, err := e.CreateStudioProject(StudioProject{Title: "相册短片", Aspect: aspect})
	if err != nil {
		return MediaRef{}, err
	}
	if dur <= 0 {
		dur = 3
	}
	shots := make([]StudioShot, 0, len(hashes))
	for _, h := range hashes {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		shots = append(shots, StudioShot{ImageHash: h, Duration: dur, Motion: "ken-burns"})
	}
	p.Storyboard.Shots = shots
	p.Storyboard.ImageMotion = "ken-burns"
	p, err = e.UpdateStudioProject(p)
	if err != nil {
		return MediaRef{}, err
	}
	p, err = e.AssembleStudio(p.ID)
	if err != nil {
		return MediaRef{}, err
	}
	return MediaRef{Kind: "video", CASHash: p.ResultHash, MIME: "video/mp4", PosterHash: p.PosterHash, Source: "studio", OriginID: p.ID}, nil
}

// ffmpegErr 取 ffmpeg 输出的最后几行当错误文本。输出是空的就用错误本身。
func ffmpegErr(out []byte, err error) string {
	tail := strings.TrimSpace(string(out))
	if n := strings.LastIndex(tail, "\n"); n >= 0 && len(tail) > 400 {
		lines := strings.Split(tail, "\n")
		if len(lines) > 8 {
			lines = lines[len(lines)-8:]
		}
		tail = strings.Join(lines, "\n")
	}
	if tail == "" {
		return err.Error()
	}
	return tail
}
