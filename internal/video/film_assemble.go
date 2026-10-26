// 把按词画出的帧、人声、垫乐和字幕合成一条成片。
// 没有 ffmpeg，或编码失败时，不写 film.mp4，已经画出的帧留在临时目录里随目录一起清掉。

package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/film"
)

// assembleFilm 按词把画面收成一条 mp4，并写成产物 film.mp4。没有 ffmpeg、没有可用的帧，或编码失败，就原样返回工程。垫乐有的话混在人声下面。
func (e *Engine) assembleFilm(p film.Project) film.Project {
	if strings.TrimSpace(e.FFMPEG) == "" {
		return p
	}
	dir, err := os.MkdirTemp(e.Dir, "film-mux-*")
	if err != nil {
		return p
	}
	defer os.RemoveAll(dir)
	var lines []string
	var last string
	n := 0
	for _, shot := range p.Shots {
		for _, word := range film.ShotWords(p, shot) {
			png, err := e.filmFrame(p, shot, word)
			if err != nil || len(png) == 0 {
				continue
			}
			dst := filepath.Join(dir, fmt.Sprintf("f-%d.png", n))
			if err := os.WriteFile(dst, png, 0o644); err != nil {
				return p
			}
			dur := word.End - word.Start
			if dur < 0.25 {
				dur = 0.25
			}
			if dur > 8 {
				dur = 8
			}
			lines = append(lines, "file '"+escapeConcat(dst)+"'", fmt.Sprintf("duration %.3f", dur))
			last = dst
			n++
		}
	}
	if n == 0 {
		return p
	}
	lines = append(lines, "file '"+escapeConcat(last)+"'")
	list := filepath.Join(dir, "list.txt")
	if err := os.WriteFile(list, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return p
	}
	speech := artifactPath(e, dir, p, "speech", ".mp3")
	srt := artifactPath(e, dir, p, "captions.srt", ".srt")
	bed := ""
	if strings.TrimSpace(p.Bed) != "" {
		if path, err := e.materialize(p.Bed, ".mp3"); err == nil {
			dst := filepath.Join(dir, "bed.mp3")
			if err := os.Rename(path, dst); err != nil {
				raw, rerr := os.ReadFile(path)
				os.Remove(path)
				if rerr == nil {
					_ = os.WriteFile(dst, raw, 0o644)
					bed = dst
				}
			} else {
				bed = dst
			}
		}
	}
	out := filepath.Join(dir, "film.mp4")
	if err := e.muxFilm(list, speech, bed, srt, out); err != nil {
		return p
	}
	raw, err := os.ReadFile(out)
	if err != nil || len(raw) == 0 {
		return p
	}
	hash, err := e.PutBytes(raw)
	if err != nil || hash == "" {
		return p
	}
	p.Artifacts = append(p.Artifacts, film.Artifact{Name: "film.mp4", Hash: hash, MediaType: "video/mp4"})
	return p
}

// filmFrame 画出这一词的一帧。排行榜会把已有的行图填进格子。揭示镜头有抠像时叠到区域上。broll 有生成图时，排行榜叠在下半部分，其余把词和贴纸画到图上。没有生成图就用代码画面。
// p 的 Structure 为 ranking 时收集 image:entry 图并传给 RenderMomentImages，broll 时再决定 CompositeInsert 或 Annotate。
// shot 传给 RenderMomentImages；Visual 为 reveal 时按区域叠抠像，为 broll 时查找 image: 加 ID 的图。
// word 传给 RenderMomentImages，非排行榜的 broll 再传给 Annotate。
func (e *Engine) filmFrame(p film.Project, shot film.Shot, word film.Word) ([]byte, error) {
	pictures := map[string][]byte{}
	if p.Structure == "ranking" {
		for _, entry := range film.VisibleEntries(p, shot) {
			for _, a := range p.Artifacts {
				if a.Name == "image:entry:"+entry.ID && a.Hash != "" {
					if raw, err := e.GetBytes(a.Hash); err == nil && len(raw) > 0 {
						pictures[entry.ID] = raw
					}
				}
			}
		}
	}
	base, err := film.RenderMomentImages(p, shot, word, pictures)
	if err != nil {
		return nil, err
	}
	if shot.Visual == "reveal" {
		for _, a := range p.Artifacts {
			if a.Name == "matting" && a.Hash != "" {
				if raw, err := e.GetBytes(a.Hash); err == nil && len(raw) > 0 {
					if mixed, err := film.CompositeMatte(base, raw, shot.Region); err == nil {
						base = mixed
					}
				}
			}
		}
	}
	if shot.Visual != "broll" {
		return base, nil
	}
	hash := ""
	for _, a := range p.Artifacts {
		if a.Name == "image:"+shot.ID && strings.HasPrefix(a.MediaType, "image/") && a.Hash != "" {
			hash = a.Hash
			break
		}
	}
	if hash == "" {
		return base, nil
	}
	raw, err := e.GetBytes(hash)
	if err != nil || len(raw) == 0 {
		return base, nil
	}
	if p.Structure == "ranking" {
		if mixed, err := film.CompositeInsert(base, raw); err == nil {
			return mixed, nil
		}
		return base, nil
	}
	if over, err := film.Annotate(raw, p, shot, word); err == nil {
		return over, nil
	}
	return base, nil
}

// artifactPath 把指定产物从内容库拷到工作目录。找不到或拷贝失败就返回空串。
// e 调用 materialize。
// dir 与文件名拼成拷贝目标。
// p 的 Artifacts 按名字和哈希查找。
// name 用来匹配产物名，冒号换成连字符后作为文件名。
// ext 传给 materialize，文件名没有扩展名时再接上。
func artifactPath(e *Engine, dir string, p film.Project, name, ext string) string {
	for _, a := range p.Artifacts {
		if a.Name != name || a.Hash == "" {
			continue
		}
		path, err := e.materialize(a.Hash, ext)
		if err != nil {
			return ""
		}
		base := strings.ReplaceAll(name, ":", "-")
		if filepath.Ext(base) == "" {
			base += ext
		}
		dst := filepath.Join(dir, base)
		if err := os.Rename(path, dst); err != nil {
			raw, rerr := os.ReadFile(path)
			os.Remove(path)
			if rerr != nil || os.WriteFile(dst, raw, 0o644) != nil {
				return ""
			}
		}
		return dst
	}
	return ""
}

// muxFilm 先带字幕封装。字幕导致失败时再去掉字幕试一次。
// list 传给 runMux。
// speech 传给 runMux。
// bed 传给 runMux。
// srt 非空且带字幕封装失败时，第二次用空串传给 runMux。
// out 传给 runMux。
func (e *Engine) muxFilm(list, speech, bed, srt, out string) error {
	err := runMux(e, list, speech, bed, srt, out)
	if err != nil && srt != "" {
		return runMux(e, list, speech, bed, "", out)
	}
	return err
}

// runMux 用 ffmpeg 把静帧序列、人声和垫乐收成 H.264。人声和垫乐同时在时，垫乐音量是 0.18。都没有音轨就不要音频。
// list 作为 concat 输入。
// speech 非空时作为下一路输入。
// bed 非空时作为下一路输入，与 speech 同时非空时音量写成 0.18。
// srt 非空时交给 ffmpegSubPath。
// out 作为输出文件。
func runMux(e *Engine, list, speech, bed, srt, out string) error {
	args := []string{"-y", "-f", "concat", "-safe", "0", "-i", list}
	if speech != "" {
		args = append(args, "-i", speech)
	}
	if bed != "" {
		args = append(args, "-i", bed)
	}
	vf := "scale=trunc(iw/2)*2:trunc(ih/2)*2"
	if srt != "" {
		vf += ",subtitles=" + ffmpegSubPath(srt)
	}
	if speech != "" && bed != "" {
		fc := "[0:v]" + vf + "[v];[1:a]volume=1[a1];[2:a]volume=0.18[a2];[a1][a2]amix=inputs=2:duration=first:dropout_transition=0[a]"
		args = append(args, "-filter_complex", fc, "-map", "[v]", "-map", "[a]", "-c:a", "aac", "-shortest")
	} else {
		args = append(args, "-vf", vf)
		if speech != "" || bed != "" {
			args = append(args, "-c:a", "aac", "-shortest")
		} else {
			args = append(args, "-an")
		}
	}
	args = append(args, "-r", "30", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-movflags", "+faststart", out)
	cmd := quiet(exec.Command(e.FFMPEG, args...))
	if _, err := cmd.CombinedOutput(); err != nil {
		return err
	}
	return nil
}

// escapeConcat 给 concat 列表转义单引号，并把路径换成斜杠。
// path 换成斜杠，并把单引号转义。
func escapeConcat(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), "'", `'\''`)
}

// ffmpegSubPath 给 subtitles 滤镜转义冒号和单引号。
// path 换成斜杠，冒号和单引号转义后再包上单引号。
func ffmpegSubPath(path string) string {
	s := filepath.ToSlash(path)
	s = strings.ReplaceAll(s, ":", `\:`)
	s = strings.ReplaceAll(s, "'", `\'`)
	return "'" + s + "'"
}
