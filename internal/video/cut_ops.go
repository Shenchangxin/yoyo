// cut_ops.go 对剪辑工程做摄入、拆条、字幕、配音和路由，并按配方从源片渲染成片。
// 成片不在上一版成片上叠加：剪辑表、调色、原声增强、烧字幕和混音都记在工程上，每次从源片重做。ffmpeg 和网络调用都在工程锁外做，做完再在锁里写结果。

package video

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// CutIngest 把源哈希写入工程。标识为空时新建。能出封面就写入。场景分析留到后面，这里先读时长，避免桌面把 0 秒当成导入失败。换了源片时，旧的成片、片段和剪辑表一起清掉。封面和时长在锁外读。
// id 去掉空白后为空时用 title 和 hash 调用 CreateCutProject，否则按它读取工程。
// hash 去掉空白后为空则返回 source hash required，否则写入 SourceHash 和 Stems.Original，并传给 posterFromHash 和 probeCutDuration。
// title 非空时写入 Title；新建工程时也传给 CreateCutProject。
func (e *Engine) CutIngest(id, hash, title string) (CutProject, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return CutProject{}, fmt.Errorf("source hash required")
	}
	if strings.TrimSpace(id) == "" {
		p, err := e.CreateCutProject(CutProject{Title: title, SourceHash: hash})
		if err != nil {
			return CutProject{}, err
		}
		id = p.ID
	} else if _, err := e.GetCutProject(id); err != nil {
		return CutProject{}, err
	}
	poster, _ := e.posterFromHash(hash)
	dur := e.probeCutDuration(hash)
	return e.mutateCut(id, func(p *CutProject) error {
		if p.SourceHash != "" && p.SourceHash != hash {
			p.ResultHash = ""
			p.Stale = false
			p.UseCuts = false
			p.Breakdown = cut.Breakdown{}
			p.EditList = cut.EditList{}
		}
		p.SourceHash = hash
		p.Stems.Original = hash
		if title != "" {
			p.Title = title
		}
		if poster != "" {
			p.PosterHash = poster
		}
		if dur > 0 {
			p.Duration = dur
		}
		return nil
	})
}

// probeCutDuration 读源片时长。内容库有磁盘文件就直接探，不再抄一份。先用 ffprobe，没有就解析 ffmpeg -i 的 Duration 行；后者只读文件头，大文件也快。时长不在 0 到 24 小时之间时返回 0。
// hash 传给 mediaPath。
func (e *Engine) probeCutDuration(hash string) float64 {
	path, cleanup, err := e.mediaPath(hash, ".mp4")
	if err != nil {
		return 0
	}
	defer cleanup()
	if d := e.probeDurationFast(path); d > 0 {
		return d
	}
	d, err := e.cutBin().Duration(path)
	if err != nil || d <= 0 || d > 24*60*60 {
		return 0
	}
	return d
}

// probeDurationFast 用单独的 ffprobe 在 6 秒内读 format duration。没有 ffprobe 时返回 0，交给 ffmpeg -i 去解析。
// path 作为 ffprobe 的最后一个参数；读出的时长小于等于 0 或大于 24*60*60 时返回 0。
func (e *Engine) probeDurationFast(path string) float64 {
	bin := strings.TrimSpace(e.FFProbe)
	if bin == "" || strings.Contains(strings.ToLower(filepath.Base(bin)), "ffmpeg") {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	cmd := quiet(exec.CommandContext(ctx, bin, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path))
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(out))
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	d, err := strconv.ParseFloat(s, 64)
	if err != nil || d <= 0 || d > 24*60*60 {
		return 0
	}
	return d
}

// mediaSeconds 读一份媒体的秒数。读不出时返回 0。
// hash 传给 mediaPath。
func (e *Engine) mediaSeconds(hash string) float64 {
	path, cleanup, err := e.mediaPath(hash, ".bin")
	if err != nil {
		return 0
	}
	defer cleanup()
	if d := e.probeDurationFast(path); d > 0 {
		return d
	}
	d, _ := e.cutBin().Duration(path)
	return d
}

// mediaPath 返回能交给 ffmpeg 读的文件。内容库有磁盘文件就直接用，不再抄一份；只在内存里时写成临时文件，由 cleanup 删掉。
// hash 传给 casFile，没有磁盘文件时再传给 materialize。
// ext 是临时文件的后缀。
func (e *Engine) mediaPath(hash, ext string) (string, func(), error) {
	if path := e.casFile(hash); path != "" {
		return path, func() {}, nil
	}
	tmp, err := e.materialize(hash, ext)
	if err != nil {
		return "", func() {}, err
	}
	return tmp, func() { _ = os.Remove(tmp) }, nil
}

// runCtx 返回引擎的上下文，引擎关闭时跟着取消。没有时用 Background。
func (e *Engine) runCtx() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}

// CutAnalyze 确保 ffmpeg 后对源片做场景和静音分析，写回时长和拆解。已经切出的片段留着。分析在锁外做。
// id 传给 GetCutProject 和 mutateCut。
func (e *Engine) CutAnalyze(id string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	bd, err := e.analyzeCutSource(p)
	if err != nil {
		return p, err
	}
	return e.mutateCut(id, func(cur *CutProject) error {
		if cur.SourceHash != p.SourceHash {
			return fmt.Errorf("source changed while analyzing")
		}
		applyBreakdown(cur, bd)
		return nil
	})
}

// analyzeCutSource 读源片的时长、场景切点和静音区间。没有源片就拒绝。
// p 的 SourceHash 交给 mediaPath。
func (e *Engine) analyzeCutSource(p CutProject) (cut.Breakdown, error) {
	if p.SourceHash == "" {
		return cut.Breakdown{}, fmt.Errorf("ingest a source first")
	}
	if err := e.EnsureFFmpeg(e.runCtx()); err != nil {
		return cut.Breakdown{}, err
	}
	src, cleanup, err := e.mediaPath(p.SourceHash, ".mp4")
	if err != nil {
		return cut.Breakdown{}, err
	}
	defer cleanup()
	return e.cutBin().Analyze(src)
}

// applyBreakdown 写上时长、场景和静音。已经切出的片段留着。
func applyBreakdown(p *CutProject, bd cut.Breakdown) {
	p.Duration = bd.Duration
	p.Breakdown.Duration = bd.Duration
	p.Breakdown.Scenes = bd.Scenes
	p.Breakdown.Silences = bd.Silences
}

// CutPropose 在还没有拆解时先分析，再按窗口打分挑出片段，带音轨抽出来写入内容库，剪辑表换成这些片段。已有成片时把工程标成过期。分析和抽片在锁外做。
// id 传给 GetCutProject 和 mutateCut。
// count 传给 cut.Propose。
// window 传给 windowSeconds。
func (e *Engine) CutPropose(id string, count int, window string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	if p.SourceHash == "" {
		return p, fmt.Errorf("ingest a source first")
	}
	bd := p.Breakdown
	if bd.Duration <= 0 {
		if bd, err = e.analyzeCutSource(p); err != nil {
			return p, err
		}
	}
	clips := cut.Propose(bd.Duration, bd.Scenes, bd.Silences, count, windowSeconds(window))
	if err := e.EnsureFFmpeg(e.runCtx()); err != nil {
		return p, err
	}
	src, cleanup, err := e.mediaPath(p.SourceHash, ".mp4")
	if err != nil {
		return p, err
	}
	defer cleanup()
	tmp, err := os.MkdirTemp(e.Dir, "cut-prop-*")
	if err != nil {
		return p, err
	}
	defer os.RemoveAll(tmp)
	out := make([]cut.Clip, 0, len(clips))
	for i, c := range clips {
		path := filepath.Join(tmp, fmt.Sprintf("c%d.mp4", i))
		if err := e.cutBin().Extract(src, c.Start, c.Dur(), path); err != nil {
			return p, err
		}
		h, _, err := e.PutPath(path)
		if err != nil {
			return p, err
		}
		c.ID = NewID()
		c.Hash = h
		out = append(out, c)
	}
	return e.mutateCut(id, func(cur *CutProject) error {
		if cur.SourceHash != p.SourceHash {
			return fmt.Errorf("source changed while proposing")
		}
		if cur.Breakdown.Duration <= 0 {
			applyBreakdown(cur, bd)
		}
		cur.Breakdown.Clips = out
		cur.EditList.Cuts = out
		if cur.ResultHash != "" {
			cur.Stale = true
		}
		return nil
	})
}

// CutApply 按剪辑表合成成片，剪辑表空着时用拆出的片段。合成过之后成片一直按剪辑表来，调色、字幕和配音都在剪好的片子上重做。
// id 传给 mutateCut 和 renderCut。
func (e *Engine) CutApply(id string) (CutProject, error) {
	if _, err := e.mutateCut(id, func(p *CutProject) error {
		if len(p.EditList.Cuts) == 0 && len(p.Breakdown.Clips) == 0 {
			return fmt.Errorf("no cuts to apply")
		}
		if p.UseCuts {
			return errUnchanged
		}
		p.UseCuts = true
		return nil
	}); err != nil {
		return CutProject{}, err
	}
	return e.renderCut(id)
}

// CutExtractAudio 把成片或源片的音轨抽出来写进 Stems.Original，方便下载或另作他用。不改成片。抽取在锁外做。
// id 传给 GetCutProject 和 mutateCut。
func (e *Engine) CutExtractAudio(id string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	hash := firstNonEmpty(p.ResultHash, p.SourceHash)
	if hash == "" {
		return p, fmt.Errorf("no video")
	}
	if err := e.EnsureFFmpeg(e.runCtx()); err != nil {
		return p, err
	}
	src, cleanup, err := e.mediaPath(hash, ".mp4")
	if err != nil {
		return p, err
	}
	defer cleanup()
	tmp, err := os.MkdirTemp(e.Dir, "cut-audio-*")
	if err != nil {
		return p, err
	}
	defer os.RemoveAll(tmp)
	out := filepath.Join(tmp, "audio.m4a")
	if err := e.cutBin().ExtractAudio(src, out); err != nil {
		return p, err
	}
	h, _, err := e.PutPath(out)
	if err != nil {
		return p, err
	}
	return e.mutateCut(id, func(cur *CutProject) error {
		cur.Stems.Original = h
		return nil
	})
}

// CutEnhance 记下原声增强预设（voice、denoise、loudnorm；none 清掉，空着按 voice），再按配方重做成片。增强每次都从源片的原声算起，不会叠加。
// id 传给 mutateCut 和 renderCut。
// preset 去掉空白并转成小写后写进配方。
func (e *Engine) CutEnhance(id, preset string) (CutProject, error) {
	preset = strings.ToLower(strings.TrimSpace(preset))
	if preset == "" {
		preset = "voice"
	}
	if _, err := e.mutateCut(id, func(p *CutProject) error {
		if p.SourceHash == "" {
			return fmt.Errorf("no audio")
		}
		if preset == "none" {
			p.Enhance = ""
		} else {
			p.Enhance = preset
		}
		return nil
	}); err != nil {
		return CutProject{}, err
	}
	return e.renderCut(id)
}

// CutMix 把配音（或人声）和音乐混进成片，再按配方重做。默认原声压低垫在配音下面；replace 为真时配音替换原声。duck 为真时人声下面的原声和音乐会压低。成片长度跟着画面走，配音短也不会把片子剪短。
// id 传给 mutateCut 和 renderCut。
// duck 写成配方的 DuckOff 取反。
// replace 写成配方的 DubReplace。
func (e *Engine) CutMix(id string, duck, replace bool) (CutProject, error) {
	if _, err := e.mutateCut(id, func(p *CutProject) error {
		if firstNonEmpty(p.ResultHash, p.SourceHash) == "" {
			return fmt.Errorf("no video to mix")
		}
		if firstNonEmpty(p.Stems.Dub, p.Stems.Voice) == "" && p.Stems.Music == "" {
			return fmt.Errorf("no speech or music stem")
		}
		p.DuckOff = !duck
		p.DubReplace = replace
		return nil
	}); err != nil {
		return CutProject{}, err
	}
	return e.renderCut(id)
}

// CutTranscribe 用本机 whisper.cpp 听写源片。字幕时间按源片算，略过字幕时才能准确挖掉对应区间，烧字幕时再按剪辑表搬到成片上。没有时间轴但有文本时按时长铺开。SRT 和 VTT 分别入库，文本写入 Note。听写在锁外做。
// id 传给 GetCutProject 和 mutateCut。
func (e *Engine) CutTranscribe(id string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	if p.SourceHash == "" {
		return p, fmt.Errorf("no video")
	}
	if err := e.EnsureFFmpeg(e.runCtx()); err != nil {
		return p, err
	}
	src, cleanup, err := e.mediaPath(p.SourceHash, ".mp4")
	if err != nil {
		return p, err
	}
	defer cleanup()
	tmp, err := os.MkdirTemp(e.Dir, "cut-asr-*")
	if err != nil {
		return p, err
	}
	defer os.RemoveAll(tmp)
	cues, text, err := e.cutBin().Transcribe(src, tmp)
	if err != nil {
		return p, err
	}
	if len(cues) == 0 && text != "" {
		cues = cut.CuesFromText(text, p.Duration)
	}
	srtHash, vttHash := e.storeCaptions(cues)
	return e.mutateCut(id, func(cur *CutProject) error {
		if cur.SourceHash != p.SourceHash {
			return fmt.Errorf("source changed while transcribing")
		}
		cur.Captions.Cues = cues
		cur.Captions.Basis = ""
		cur.Captions.SRTHash = srtHash
		cur.Captions.VTTHash = vttHash
		if text != "" {
			cur.Note = text
		}
		if cur.Burn && cur.ResultHash != "" {
			cur.Stale = true
		}
		return nil
	})
}

// storeCaptions 把字幕编成 SRT 和 VTT 放进内容库。没有字幕时两个都是空串。
func (e *Engine) storeCaptions(cues []cut.Cue) (srtHash, vttHash string) {
	if len(cues) == 0 {
		return "", ""
	}
	if srt := cut.EncodeSRT(cues); srt != "" {
		if h, err := e.PutBytes([]byte(srt)); err == nil {
			srtHash = h
		}
	}
	if h, err := e.PutBytes([]byte(cut.EncodeVTT(cues))); err == nil {
		vttHash = h
	}
	return srtHash, vttHash
}

// CutCaptions 用传入文本按成片时长重铺字幕，时间按成片算；text 空着时只改语言。非空字幕会重写 SRT 和 VTT。烧着字幕的成片标成过期。
// id 传给 mutateCut。
// text 去掉空白后非空时传给 CuesFromText。
// lang 非空时写入 Captions.Language。
func (e *Engine) CutCaptions(id, text, lang string) (CutProject, error) {
	text = strings.TrimSpace(text)
	return e.mutateCut(id, func(p *CutProject) error {
		if text != "" {
			p.Captions.Cues = cut.CuesFromText(text, outputDuration(*p))
			p.Captions.Basis = cut.BasisResult
			if p.Burn && p.ResultHash != "" {
				p.Stale = true
			}
		}
		if lang != "" {
			p.Captions.Language = lang
		}
		p.Captions.SRTHash, p.Captions.VTTHash = e.storeCaptions(p.Captions.Cues)
		return nil
	})
}

// CutBurnin 打开烧字幕，再按配方重做成片。字幕用中文字体排成 ASS，源片字幕先按剪辑表搬到成片时间上。重复点不会烧两层；已经烧好且没过期时直接返回。
// id 传给 mutateCut 和 renderCut。
func (e *Engine) CutBurnin(id string) (CutProject, error) {
	fresh := false
	cur, err := e.mutateCut(id, func(p *CutProject) error {
		if firstNonEmpty(p.ResultHash, p.SourceHash) == "" || len(burnCues(*p)) == 0 {
			return fmt.Errorf("need video and captions — transcribe or set caption text first")
		}
		if p.Burn && p.ResultHash != "" && !p.Stale {
			fresh = true
			return errUnchanged
		}
		p.Burn = true
		return nil
	})
	if err != nil {
		return CutProject{}, err
	}
	if fresh {
		return cur, nil
	}
	return e.renderCut(id)
}

// CutDub 用文本、备注或字幕拼出配音稿，走设置里启用的语音账号，没有时用本机 edge-tts。配音写进 Dub 音轨；要混进成片再调 CutMix。
// 还没有字幕，或字幕本来就是照文案铺的，就按配音时长重铺一份，时间按成片算。已有成片时标成过期。配音在锁外做。
// id 传给 GetCutProject 和 mutateCut。
// text 去掉空白后为空时改用 Note，仍空则拼接字幕文本。
// voice 为空时写成 alloy，再传给 speakAudio。
func (e *Engine) CutDub(id, text, voice string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		text = strings.TrimSpace(p.Note)
	}
	if text == "" && len(p.Captions.Cues) > 0 {
		var b strings.Builder
		for _, c := range p.Captions.Cues {
			if c.Text != "" && !c.Omit {
				b.WriteString(c.Text)
				b.WriteByte(' ')
			}
		}
		text = strings.TrimSpace(b.String())
	}
	if text == "" {
		return p, fmt.Errorf("no dub text — transcribe or pass text")
	}
	spoken, err := e.speakAudio(text, voice, 0)
	if err != nil {
		return p, err
	}
	h, err := e.PutBytes(spoken.Audio)
	if err != nil {
		return p, err
	}
	dur := e.mediaSeconds(h)
	return e.mutateCut(id, func(cur *CutProject) error {
		cur.Stems.Dub = h
		if len(cur.Captions.Cues) == 0 || cur.Captions.Basis == cut.BasisResult {
			span := dur
			if span <= 0 {
				span = outputDuration(*cur)
			}
			cur.Captions.Cues = cut.CuesFromText(text, span)
			cur.Captions.Basis = cut.BasisResult
			cur.Captions.SRTHash, cur.Captions.VTTHash = e.storeCaptions(cur.Captions.Cues)
		}
		if cur.ResultHash != "" {
			cur.Stale = true
		}
		return nil
	})
}

// CutGrade 记下调色预设（warm、cool、contrast、fade；空着用默认的轻微提亮，none 清掉），再按配方从源片重做成片。重复调色不会叠加。
// id 传给 mutateCut 和 renderCut。
// preset 去掉空白并转成小写后写进配方。
func (e *Engine) CutGrade(id, preset string) (CutProject, error) {
	preset = strings.ToLower(strings.TrimSpace(preset))
	if preset == "" {
		preset = "default"
	}
	if _, err := e.mutateCut(id, func(p *CutProject) error {
		if p.SourceHash == "" {
			return fmt.Errorf("no video")
		}
		if preset == "none" {
			p.Grade = ""
		} else {
			p.Grade = preset
		}
		return nil
	}); err != nil {
		return CutProject{}, err
	}
	return e.renderCut(id)
}

// CutSetStem 按 music、sfx、voice、dub、original 写入对应音轨。其他种类返回 unknown stem。已有成片且换的是会混进去的音轨时标成过期。
// id 传给 mutateCut。
// kind 去掉空白并转成小写后选择 music、sfx、voice、dub 或 original，其他值返回 unknown stem。
// hash 去掉空白后写入对应音轨。
func (e *Engine) CutSetStem(id, kind, hash string) (CutProject, error) {
	hash = strings.TrimSpace(hash)
	kind = strings.ToLower(strings.TrimSpace(kind))
	return e.mutateCut(id, func(p *CutProject) error {
		switch kind {
		case "music":
			p.Stems.Music = hash
		case "sfx":
			p.Stems.SFX = hash
		case "voice":
			p.Stems.Voice = hash
		case "dub":
			p.Stems.Dub = hash
		case "original":
			p.Stems.Original = hash
			return nil
		default:
			return fmt.Errorf("unknown stem %s", kind)
		}
		if p.ResultHash != "" {
			p.Stale = true
		}
		return nil
	})
}

// posterFromHash 从内容寻址文件或临时物化文件抽出一帧封面并入库。
// hash 传给 casFile，没有文件时再传给 materialize。
func (e *Engine) posterFromHash(hash string) (string, error) {
	path := e.casFile(hash)
	temp := ""
	if path == "" {
		src, err := e.materialize(hash, ".mp4")
		if err != nil {
			return "", err
		}
		path, temp = src, src
	}
	if temp != "" {
		defer os.Remove(temp)
	}
	jpg, err := e.Poster(path)
	if err != nil {
		return "", err
	}
	return e.PutBytes(jpg)
}

// CutKeep 只留下指定片段，再按已省略的源片字幕挖掉对应区间。已有成片时标成过期。
// id 传给 mutateCut。
// clipIDs 去掉空白后非空的值收成保留集合，只留下 ID 在其中的拆解片段。
func (e *Engine) CutKeep(id string, clipIDs []string) (CutProject, error) {
	want := map[string]bool{}
	for _, clipID := range clipIDs {
		if clipID = strings.TrimSpace(clipID); clipID != "" {
			want[clipID] = true
		}
	}
	return e.mutateCut(id, func(p *CutProject) error {
		var kept []cut.Clip
		for _, c := range p.Breakdown.Clips {
			if want[c.ID] {
				kept = append(kept, c)
			}
		}
		if p.Captions.Basis != cut.BasisResult {
			for _, cue := range p.Captions.Cues {
				if cue.Omit {
					kept = cut.Punch(kept, cue.Start, cue.End)
				}
			}
		}
		if kept == nil {
			kept = []cut.Clip{}
		}
		p.EditList.Cuts = kept
		if p.ResultHash != "" {
			p.Stale = true
		}
		return nil
	})
}

// CutOmitCue 切换一条字幕的略过状态。源片字幕会改剪辑表：没有剪辑表时先铺整段或复制拆解片段，略过时挖掉区间，恢复时插回。照文案铺的字幕只影响烧进画面的字。已有成片时标成过期。
// id 传给 mutateCut。
// index 小于 0 或大于等于字幕条数时返回 cue not found，否则用来切换该条的 Omit。
func (e *Engine) CutOmitCue(id string, index int) (CutProject, error) {
	return e.mutateCut(id, func(p *CutProject) error {
		if index < 0 || index >= len(p.Captions.Cues) {
			return fmt.Errorf("cue not found")
		}
		cue := p.Captions.Cues[index]
		cue.Omit = !cue.Omit
		p.Captions.Cues[index] = cue
		if p.Captions.Basis != cut.BasisResult {
			if len(p.EditList.Cuts) == 0 {
				if p.Duration > 0 {
					p.EditList.Cuts = []cut.Clip{{Start: 0, End: p.Duration, Title: "source"}}
				} else if len(p.Breakdown.Clips) > 0 {
					p.EditList.Cuts = append([]cut.Clip{}, p.Breakdown.Clips...)
				}
			}
			if cue.Omit {
				p.EditList.Cuts = cut.Punch(p.EditList.Cuts, cue.Start, cue.End)
			} else {
				p.EditList.Cuts = cut.InsertSpan(p.EditList.Cuts, cue.Start, cue.End)
			}
		}
		if p.ResultHash != "" {
			p.Stale = true
		}
		return nil
	})
}

// cutsInUse 返回成片要用的剪辑表：合成过（UseCuts），或旧工程已经有成片和剪辑表时用剪辑表，剪辑表空着时用拆出的片段。返回 nil 表示整条源片。
func cutsInUse(p CutProject) []cut.Clip {
	if !p.UseCuts && !(p.ResultHash != "" && len(p.EditList.Cuts) > 0) {
		return nil
	}
	if len(p.EditList.Cuts) > 0 {
		return p.EditList.Cuts
	}
	return p.Breakdown.Clips
}

// outputDuration 是成片的秒数：用剪辑表时是各段之和，否则是源片时长。
func outputDuration(p CutProject) float64 {
	cuts := cutsInUse(p)
	if len(cuts) == 0 {
		return p.Duration
	}
	var d float64
	for _, c := range cuts {
		d += c.Dur()
	}
	return d
}

// burnCues 返回要烧进成片的字幕，时间按成片算。略过的不要；源片字幕按剪辑表搬到成片上。
func burnCues(p CutProject) []cut.Cue {
	var cues []cut.Cue
	for _, c := range p.Captions.Cues {
		if !c.Omit && strings.TrimSpace(c.Text) != "" {
			cues = append(cues, c)
		}
	}
	if p.Captions.Basis == cut.BasisResult {
		return cues
	}
	if cuts := cutsInUse(p); len(cuts) > 0 {
		return cut.MapCues(cues, cuts)
	}
	return cues
}

// cutRecipeKey 给成片配方算一个摘要。渲染期间配方又变了，写回时就把成片标成过期。
func cutRecipeKey(p CutProject) string {
	type span struct {
		Start, End float64
		Hash       string
	}
	var cuts []span
	for _, c := range cutsInUse(p) {
		cuts = append(cuts, span{c.Start, c.End, c.Hash})
	}
	var cues []cut.Cue
	if p.Burn {
		cues = burnCues(p)
	}
	raw, _ := json.Marshal(struct {
		Source, Grade, Enhance, Dub, Voice, Music string
		Burn, Replace, DuckOff                    bool
		Cuts                                      []span
		Cues                                      []cut.Cue
	}{p.SourceHash, p.Grade, p.Enhance, p.Stems.Dub, p.Stems.Voice, p.Stems.Music, p.Burn, p.DubReplace, p.DuckOff, cuts, cues})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// renderCut 按配方从源片重新做成片：剪辑表拼接，调色和烧字幕，原声增强，再混配音和音乐。每一步都从源片算起，重复调色或烧字幕不会叠加，重新合成也不会丢掉配音和调色。
// 配方什么都没有时成片就是源片，成片编号清空。长活在锁外做，做完在锁里写成片；做的过程中配方又变了，就把成片标成过期。
// id 传给 GetCutProject 和 mutateCut。
func (e *Engine) renderCut(id string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	if strings.TrimSpace(p.SourceHash) == "" {
		return p, fmt.Errorf("ingest a source first")
	}
	if err := e.EnsureFFmpeg(e.runCtx()); err != nil {
		return p, err
	}
	key := cutRecipeKey(p)
	bin := e.cutBin()
	tmp, err := os.MkdirTemp(e.Dir, "cut-render-*")
	if err != nil {
		return p, err
	}
	defer os.RemoveAll(tmp)
	src, cleanup, err := e.mediaPath(p.SourceHash, ".mp4")
	if err != nil {
		return p, err
	}
	defer cleanup()

	stage := src
	if cuts := cutsInUse(p); len(cuts) > 0 {
		parts := make([]string, 0, len(cuts))
		for i, c := range cuts {
			if c.Hash != "" {
				if path := e.casFile(c.Hash); path != "" {
					parts = append(parts, path)
					continue
				}
			}
			part := filepath.Join(tmp, fmt.Sprintf("s%d.mp4", i))
			if err := bin.Extract(src, c.Start, c.Dur(), part); err != nil {
				return p, err
			}
			parts = append(parts, part)
		}
		stage = filepath.Join(tmp, "cut.mp4")
		if err := bin.ConcatKeepAudio(parts, stage); err != nil {
			return p, err
		}
	}

	var vf []string
	if g := strings.TrimSpace(p.Grade); g != "" {
		vf = append(vf, cut.GradeFilter(g))
	}
	if p.Burn {
		if cues := burnCues(p); len(cues) > 0 {
			subPath, err := e.writeBurnSubs(tmp, stage, cues)
			if err != nil {
				return p, err
			}
			vf = append(vf, subtitleFilter(subPath))
		}
	}
	af := ""
	if en := strings.TrimSpace(p.Enhance); en != "" {
		af = cut.EnhanceFilter(en)
	}
	if len(vf) > 0 || af != "" {
		out := filepath.Join(tmp, "look.mp4")
		if err := bin.Look(stage, out, strings.Join(vf, ","), af); err != nil {
			return p, err
		}
		stage = out
	}

	if speech, music := firstNonEmpty(p.Stems.Dub, p.Stems.Voice), p.Stems.Music; speech != "" || music != "" {
		var sp, mu string
		if speech != "" {
			path, done, err := e.mediaPath(speech, ".m4a")
			if err != nil {
				return p, err
			}
			defer done()
			sp = path
		}
		if music != "" {
			path, done, err := e.mediaPath(music, ".mp3")
			if err != nil {
				return p, err
			}
			defer done()
			mu = path
		}
		out := filepath.Join(tmp, "mix.mp4")
		// 人声音轨（没有配音时）就是原声的替身，替换原声；配音默认垫着原声。
		replace := p.DubReplace || (p.Stems.Dub == "" && p.Stems.Voice != "")
		if err := bin.Mix(stage, sp, mu, out, cut.MixOpts{Duck: !p.DuckOff, ReplaceOrig: replace}); err != nil {
			return p, err
		}
		stage = out
	}

	hash := ""
	if stage != src {
		if hash, _, err = e.PutPath(stage); err != nil {
			return p, err
		}
	}
	return e.mutateCut(id, func(cur *CutProject) error {
		cur.ResultHash = hash
		cur.Stale = cutRecipeKey(*cur) != key
		return nil
	})
}

// writeBurnSubs 把要烧的字幕写成文件。能排成 ASS 就用 ASS，带上中文字体和按画面大小算的字号，否则写 SRT。
// tmp 是放字幕文件的目录。
// video 用来探画面宽高。
// cues 是按成片时间算好的字幕。
func (e *Engine) writeBurnSubs(tmp, video string, cues []cut.Cue) (string, error) {
	srt := cut.EncodeSRT(cues)
	w, h := e.probeFrame(video)
	if ass := srtToASS(srt, w, h); ass != "" {
		path := filepath.Join(tmp, "captions.ass")
		return path, os.WriteFile(path, []byte(ass), 0o600)
	}
	path := filepath.Join(tmp, "captions.srt")
	return path, os.WriteFile(path, []byte(srt), 0o600)
}

// ProviderMenu 列出该能力下已启用且有密钥的连接。默认连接会标上 default 和 default_for。
// cap 传给 Conn.List。
func (e *Engine) ProviderMenu(cap string) []map[string]any {
	if e.Conn == nil {
		return []map[string]any{}
	}
	list, err := e.Conn.List(cap)
	if err != nil {
		return []map[string]any{}
	}
	defs := e.Conn.Defaults()
	var out []map[string]any
	for _, c := range list {
		if !c.Active || !c.HasKey {
			continue
		}
		caps := connection.CapsOf(c)
		row := map[string]any{
			"id": c.ID, "name": c.Name, "vendor": c.Vendor,
			"protocol": c.Protocol, "capabilities": caps,
			"models": c.Models, "default": false,
		}
		for _, capName := range caps {
			if defs[capName] == c.ID {
				row["default"] = true
				row["default_for"] = capName
			}
		}
		out = append(out, row)
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out
}

// VideoRoute 按目的地绑定会话。短片没有标识时新建。短剧必须带集标识。画布只在标识非空时绑定。游戏世界返回不可用。其余目的地当成剪辑，没有标识时新建标题为「胶片」的工程。intent 不使用。成功时送出 video_route。
// sessionID 去掉空白后非空时按目的地调用 ReleaseWorkshopBinds，并绑定工作室、短剧、画布或剪辑会话。
// dest 去掉空白并转成小写后与 studio、shorts、短片、drama、短剧、canvas、画布、game、游戏、游戏世界、game-world 比较，其他值走剪辑。
// title 新建短片时为空则写成短片，新建剪辑时为空则写成胶片，并写入返回的 title。
func (e *Engine) VideoRoute(sessionID, dest, id, title, intent string) (map[string]any, error) {
	dest = strings.ToLower(strings.TrimSpace(dest))
	sessionID = strings.TrimSpace(sessionID)
	id = strings.TrimSpace(id)
	out := map[string]any{"dest": dest, "session_id": sessionID}
	switch dest {
	case "studio", "shorts", "短片":
		if id == "" {
			p, err := e.CreateStudioProject(StudioProject{Title: firstNonEmpty(title, "短片")})
			if err != nil {
				return nil, err
			}
			id = p.ID
			title = p.Title
			out["project"] = p
		}
		if sessionID != "" {
			e.ReleaseWorkshopBinds(sessionID)
			e.ReleaseCutBind(sessionID)
			if err := e.BindStudioSession(sessionID, id); err != nil {
				return nil, err
			}
		}
		out["pane"] = "studio"
		out["id"] = id
		out["title"] = title
	case "drama", "短剧":
		if id == "" {
			return nil, fmt.Errorf("drama dest needs an episode id")
		}
		ep, err := e.GetEpisode(id)
		if err != nil {
			return nil, err
		}
		if sessionID != "" {
			e.ReleaseWorkshopBinds(sessionID)
			e.ReleaseCutBind(sessionID)
			if err := e.BindSession(sessionID, id); err != nil {
				return nil, err
			}
		}
		out["pane"] = "drama"
		out["id"] = id
		out["drama_id"] = ep.DramaID
		out["title"] = firstNonEmpty(ep.Title, title)
	case "canvas", "画布":
		if sessionID != "" {
			e.ReleaseWorkshopBinds(sessionID)
			e.ReleaseCutBind(sessionID)
			if id != "" {
				if err := e.BindCanvasSession(sessionID, id); err != nil {
					return nil, err
				}
			}
		}
		out["pane"] = "canvas"
		out["id"] = id
		out["title"] = title
	case "game", "游戏", "游戏世界", "game-world":
		return nil, fmt.Errorf("game worlds are no longer available; pick 短片, 短剧, or 画布")
	default:
		if sessionID != "" {
			e.ReleaseWorkshopBinds(sessionID)
		}
		if id == "" {
			p, err := e.CreateCutProject(CutProject{Title: firstNonEmpty(title, "胶片")})
			if err != nil {
				return nil, err
			}
			id = p.ID
			out["project"] = p
		}
		if sessionID != "" {
			if err := e.BindCutSession(sessionID, id); err != nil {
				return nil, err
			}
		}
		out["pane"] = "create"
		out["id"] = id
		out["title"] = title
		out["dest"] = "cut"
	}
	_ = intent
	if e.OnHub != nil {
		e.OnHub("video_route", routeNotice(out))
	}
	return out, nil
}

// routeNotice 只留下界面切换要用的字段。桌面通知超过内联上限就不带载荷，整份工程塞进去会让界面收不到这次路由。
// out 里的 project 不抄过去。
func routeNotice(out map[string]any) map[string]any {
	notice := make(map[string]any, len(out))
	for _, k := range []string{"dest", "pane", "id", "title", "drama_id", "session_id"} {
		if v, ok := out[k]; ok {
			notice[k] = v
		}
	}
	return notice
}

// windowSeconds 把 15-30 收成 20 秒，60-90 收成 75 秒，8-15 收成 12 秒，其余收成 20 秒。
func windowSeconds(window string) float64 {
	switch strings.TrimSpace(window) {
	case "15-30":
		return 20
	case "60-90":
		return 75
	case "8-15":
		return 12
	default:
		return 20
	}
}
