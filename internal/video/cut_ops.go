// cut_ops.go 对剪辑工程做摄入、拆条、混音、字幕、配音和路由。

package video

import (
	"context"
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

// CutIngest 把源哈希写入工程。标识为空时新建。能出封面就写入。场景分析留到后面，这里先读时长，避免桌面把 0 秒当成导入失败。
// id 去掉空白后为空时用 title 和 hash 调用 CreateCutProject，否则传给 GetCutProject。
// hash 去掉空白后为空则返回 source hash required，否则写入 SourceHash 和 Stems.Original，并传给 posterFromHash 和 probeCutDuration。
// title 非空时写入 Title；新建工程时也传给 CreateCutProject。
func (e *Engine) CutIngest(id, hash, title string) (CutProject, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return CutProject{}, fmt.Errorf("source hash required")
	}
	var p CutProject
	var err error
	if strings.TrimSpace(id) == "" {
		p, err = e.CreateCutProject(CutProject{Title: title, SourceHash: hash})
	} else {
		p, err = e.GetCutProject(id)
	}
	if err != nil {
		return CutProject{}, err
	}
	p.SourceHash = hash
	p.Stems.Original = hash
	if title != "" {
		p.Title = title
	}
	if poster, perr := e.posterFromHash(hash); perr == nil {
		p.PosterHash = poster
	}
	if d := e.probeCutDuration(hash); d > 0 {
		p.Duration = d
	}
	return e.saveCut(p)
}

// probeCutDuration 先读内容寻址文件，没有文件时再物化一份。优先用 probeDurationFast。文件超过 64 MiB 就不再跑 ffmpeg -i。时长不在 0 到 24 小时之间时返回 0。
// hash 传给 casFile，没有文件时再传给 materialize；文件大于 64<<20，或时长小于等于 0、大于 24*60*60 时返回 0。
func (e *Engine) probeCutDuration(hash string) float64 {
	path := e.casFile(hash)
	if path == "" {
		src, err := e.materialize(hash, ".mp4")
		if err != nil {
			return 0
		}
		defer os.Remove(src)
		path = src
	}
	if d := e.probeDurationFast(path); d > 0 {
		return d
	}
	st, err := os.Stat(path)
	if err != nil || st.Size() > 64<<20 {
		return 0
	}
	d, err := e.cutBin().Duration(path)
	if err != nil || d <= 0 || d > 24*60*60 {
		return 0
	}
	return d
}

// probeDurationFast 用 ffprobe，没有就用 ffmpeg，6 秒超时读取 format duration。
// path 作为 ffprobe 或 ffmpeg 命令的最后一个参数；读出的时长小于等于 0 或大于 24*60*60 时返回 0。
func (e *Engine) probeDurationFast(path string) float64 {
	bin := strings.TrimSpace(e.FFProbe)
	if bin == "" {
		bin = strings.TrimSpace(e.FFMPEG)
	}
	if bin == "" {
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

// CutAnalyze 确保 ffmpeg 后对源片做场景分析，并写回时长和拆解。
// id 传给 GetCutProject。
func (e *Engine) CutAnalyze(id string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	if p.SourceHash == "" {
		return p, fmt.Errorf("ingest a source first")
	}
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return p, err
	}
	src, err := e.materialize(p.SourceHash, ".mp4")
	if err != nil {
		return p, err
	}
	defer os.Remove(src)
	bd, err := e.cutBin().Analyze(src)
	if err != nil {
		return p, err
	}
	p.Duration = bd.Duration
	p.Breakdown = bd
	return e.saveCut(p)
}

// CutPropose 在还没有时长时先分析，再按窗口切出片段写入内容寻址。已有成片时把工程标成过期。
// id 传给 GetCutProject，时长还没有时再传给 CutAnalyze。
// count 传给 cut.Propose。
// window 传给 windowSeconds。
func (e *Engine) CutPropose(id string, count int, window string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	if p.Breakdown.Duration <= 0 {
		p, err = e.CutAnalyze(id)
		if err != nil {
			return p, err
		}
	}
	span := windowSeconds(window)
	clips := cut.Propose(p.Duration, p.Breakdown.Scenes, p.Breakdown.Silences, count, span)
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return p, err
	}
	src, err := e.materialize(p.SourceHash, ".mp4")
	if err != nil {
		return p, err
	}
	defer os.Remove(src)
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
		raw, err := os.ReadFile(path)
		if err != nil {
			return p, err
		}
		h, err := e.PutBytes(raw)
		if err != nil {
			return p, err
		}
		c.ID = NewID()
		c.Hash = h
		out = append(out, c)
	}
	p.Breakdown.Clips = out
	p.EditList.Cuts = out
	if p.ResultHash != "" {
		p.Stale = true
	}
	return e.saveCut(p)
}

// CutApply 按剪辑表拼接，没有剪辑表时用拆解片段。有哈希就用字节，否则从源片截取。结果写入 ResultHash 并清掉过期标记。
// id 传给 GetCutProject。
func (e *Engine) CutApply(id string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	cuts := p.EditList.Cuts
	if len(cuts) == 0 {
		cuts = p.Breakdown.Clips
	}
	if len(cuts) == 0 {
		return p, fmt.Errorf("no cuts to apply")
	}
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return p, err
	}
	tmp, err := os.MkdirTemp(e.Dir, "cut-apply-*")
	if err != nil {
		return p, err
	}
	defer os.RemoveAll(tmp)
	var paths []string
	src := ""
	if p.SourceHash != "" {
		src, err = e.materialize(p.SourceHash, ".mp4")
		if err != nil {
			return p, err
		}
		defer os.Remove(src)
	}
	for i, c := range cuts {
		path := filepath.Join(tmp, fmt.Sprintf("s%d.mp4", i))
		if c.Hash != "" {
			raw, err := e.GetBytes(c.Hash)
			if err != nil {
				return p, err
			}
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				return p, err
			}
		} else if src != "" {
			if err := e.cutBin().Extract(src, c.Start, c.Dur(), path); err != nil {
				return p, err
			}
		} else {
			return p, fmt.Errorf("cut %d has no media", i)
		}
		paths = append(paths, path)
	}
	out := filepath.Join(tmp, "out.mp4")
	if err := e.cutBin().ConcatKeepAudio(paths, out); err != nil {
		return p, err
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return p, err
	}
	h, err := e.PutBytes(raw)
	if err != nil {
		return p, err
	}
	p.ResultHash = h
	p.Stale = false
	return e.saveCut(p)
}

// CutExtractAudio 从成片或源片抽出音轨，写入 Stems.Original。
// id 传给 GetCutProject。
func (e *Engine) CutExtractAudio(id string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	hash := firstNonEmpty(p.ResultHash, p.SourceHash)
	if hash == "" {
		return p, fmt.Errorf("no video")
	}
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return p, err
	}
	src, err := e.materialize(hash, ".mp4")
	if err != nil {
		return p, err
	}
	defer os.Remove(src)
	out := src + ".m4a"
	defer os.Remove(out)
	if err := e.cutBin().ExtractAudio(src, out); err != nil {
		return p, err
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return p, err
	}
	h, err := e.PutBytes(raw)
	if err != nil {
		return p, err
	}
	p.Stems.Original = h
	return e.saveCut(p)
}

// CutEnhance 按预设增强人声、原声、成片或源片，结果写入 Stems.Voice。
// id 传给 GetCutProject。
// preset 传给 EnhanceAudio。
func (e *Engine) CutEnhance(id, preset string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	hash := firstNonEmpty(p.Stems.Voice, p.Stems.Original, p.ResultHash, p.SourceHash)
	if hash == "" {
		return p, fmt.Errorf("no audio")
	}
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return p, err
	}
	src, err := e.materialize(hash, ".m4a")
	if err != nil {
		return p, err
	}
	defer os.Remove(src)
	out := src + ".enh.m4a"
	defer os.Remove(out)
	if err := e.cutBin().EnhanceAudio(src, out, preset); err != nil {
		return p, err
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return p, err
	}
	h, err := e.PutBytes(raw)
	if err != nil {
		return p, err
	}
	p.Stems.Voice = h
	return e.saveCut(p)
}

// CutMix 把配音或人声，以及音乐，混进成片或源片。有人声时替换原声。
// id 传给 GetCutProject。
// duck 传给 mixHashes。
func (e *Engine) CutMix(id string, duck bool) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	videoHash := firstNonEmpty(p.ResultHash, p.SourceHash)
	if videoHash == "" {
		return p, fmt.Errorf("no video to mix")
	}
	speech := firstNonEmpty(p.Stems.Dub, p.Stems.Voice)
	music := p.Stems.Music
	if speech == "" && music == "" {
		return p, fmt.Errorf("no speech or music stem")
	}
	h, err := e.mixHashes(videoHash, speech, music, duck, speech != "")
	if err != nil {
		return p, err
	}
	p.ResultHash = h
	return e.saveCut(p)
}

// mixHashes 物化视频、语音和音乐后调用 Mix，再把成片放进内容寻址。
// videoHash 传给 materialize，后缀是 .mp4。
// speechHash 非空时传给 materialize，后缀是 .mp3。
// musicHash 非空时传给 materialize，后缀是 .mp3。
// duck 写入 MixOpts 的 Duck。
// replaceOrig 写入 MixOpts 的 ReplaceOrig。
func (e *Engine) mixHashes(videoHash, speechHash, musicHash string, duck, replaceOrig bool) (string, error) {
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp(e.Dir, "cut-mix-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	video, err := e.materialize(videoHash, ".mp4")
	if err != nil {
		return "", err
	}
	defer os.Remove(video)
	var speech, music string
	if speechHash != "" {
		speech, err = e.materialize(speechHash, ".mp3")
		if err != nil {
			return "", err
		}
		defer os.Remove(speech)
	}
	if musicHash != "" {
		music, err = e.materialize(musicHash, ".mp3")
		if err != nil {
			return "", err
		}
		defer os.Remove(music)
	}
	out := filepath.Join(tmp, "mix.mp4")
	if err := e.cutBin().Mix(video, speech, music, out, cut.MixOpts{Duck: duck, ReplaceOrig: replaceOrig}); err != nil {
		return "", err
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return "", err
	}
	return e.PutBytes(raw)
}

// CutTranscribe 对成片或源片转写。没有时间轴但有文本时按时长铺开。SRT 和 VTT 分别入库，文本写入 Note。
// id 传给 GetCutProject。
func (e *Engine) CutTranscribe(id string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	hash := firstNonEmpty(p.ResultHash, p.SourceHash)
	if hash == "" {
		return p, fmt.Errorf("no video")
	}
	src, err := e.materialize(hash, ".mp4")
	if err != nil {
		return p, err
	}
	defer os.Remove(src)
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
	p.Captions.Cues = cues
	if srt := cut.EncodeSRT(cues); srt != "" {
		if h, err := e.PutBytes([]byte(srt)); err == nil {
			p.Captions.SRTHash = h
		}
	}
	if vtt := cut.EncodeVTT(cues); vtt != "" {
		if h, err := e.PutBytes([]byte(vtt)); err == nil {
			p.Captions.VTTHash = h
		}
	}
	if text != "" {
		p.Note = text
	}
	return e.saveCut(p)
}

// CutCaptions 用传入文本重铺时间轴，并按语言字段更新。非空字幕会重写 SRT 和 VTT。
// id 传给 GetCutProject。
// text 去掉空白后非空时传给 CuesFromText。
// lang 非空时写入 Captions.Language。
func (e *Engine) CutCaptions(id, text, lang string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	text = strings.TrimSpace(text)
	if text != "" {
		p.Captions.Cues = cut.CuesFromText(text, p.Duration)
	}
	if lang != "" {
		p.Captions.Language = lang
	}
	if srt := cut.EncodeSRT(p.Captions.Cues); srt != "" {
		if h, err := e.PutBytes([]byte(srt)); err == nil {
			p.Captions.SRTHash = h
		}
	}
	if vtt := cut.EncodeVTT(p.Captions.Cues); vtt != "" {
		if h, err := e.PutBytes([]byte(vtt)); err == nil {
			p.Captions.VTTHash = h
		}
	}
	return e.saveCut(p)
}

// CutBurnin 把已有 SRT 烧进成片或源片，结果写入 ResultHash。
// id 传给 GetCutProject。
func (e *Engine) CutBurnin(id string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	videoHash := firstNonEmpty(p.ResultHash, p.SourceHash)
	if videoHash == "" || p.Captions.SRTHash == "" {
		return p, fmt.Errorf("need video and SRT")
	}
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return p, err
	}
	tmp, err := os.MkdirTemp(e.Dir, "cut-burn-*")
	if err != nil {
		return p, err
	}
	defer os.RemoveAll(tmp)
	video, err := e.materialize(videoHash, ".mp4")
	if err != nil {
		return p, err
	}
	defer os.Remove(video)
	srt, err := e.materialize(p.Captions.SRTHash, ".srt")
	if err != nil {
		return p, err
	}
	defer os.Remove(srt)
	out := filepath.Join(tmp, "burn.mp4")
	if err := e.cutBin().BurnCaptions(video, srt, out); err != nil {
		return p, err
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return p, err
	}
	h, err := e.PutBytes(raw)
	if err != nil {
		return p, err
	}
	p.ResultHash = h
	return e.saveCut(p)
}

// CutDub 用文本、备注或字幕拼出配音稿，走当前语音适配器。模型空着时用 gpt-4o-mini-tts，音色空着时用 alloy。结果同时写入配音和人声。还没有字幕时按文本铺一条 SRT。
// id 传给 GetCutProject。
// text 去掉空白后为空时改用 Note，仍空则拼接字幕文本，再传给 ttsOpenAI。
// voice 为空时写成 alloy，再传给 ttsOpenAI。
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
			if c.Text != "" {
				b.WriteString(c.Text)
				b.WriteByte(' ')
			}
		}
		text = strings.TrimSpace(b.String())
	}
	if text == "" {
		return p, fmt.Errorf("no dub text — transcribe or pass text")
	}
	if e.Conn == nil {
		return p, fmt.Errorf("configure a speech adapter in Settings → Generation")
	}
	c, err := e.Conn.Active(connection.CapSpeech, "")
	if err != nil {
		return p, fmt.Errorf("configure a speech adapter in Settings → Generation")
	}
	key, err := e.Conn.Lease(c)
	if err != nil || strings.TrimSpace(key) == "" {
		return p, fmt.Errorf("speech adapter has no API key")
	}
	model := connection.ModelFor(c, connection.CapSpeech)
	if model == "" {
		model = "gpt-4o-mini-tts"
	}
	if voice == "" {
		voice = "alloy"
	}
	raw, _, err := e.ttsOpenAI(c.Endpoint, key, model, text, voice, 0)
	if err != nil {
		return p, err
	}
	h, err := e.PutBytes(raw)
	if err != nil {
		return p, err
	}
	p.Stems.Dub = h
	p.Stems.Voice = h
	if len(p.Captions.Cues) == 0 {
		p.Captions.Cues = cut.CuesFromText(text, p.Duration)
		if srt := cut.EncodeSRT(p.Captions.Cues); srt != "" {
			if sh, err := e.PutBytes([]byte(srt)); err == nil {
				p.Captions.SRTHash = sh
			}
		}
	}
	return e.saveCut(p)
}

// CutGrade 按预设给成片或源片调色，结果写入 ResultHash。
// id 传给 GetCutProject。
// preset 传给 Grade。
func (e *Engine) CutGrade(id, preset string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	hash := firstNonEmpty(p.ResultHash, p.SourceHash)
	if hash == "" {
		return p, fmt.Errorf("no video")
	}
	if err := e.EnsureFFmpeg(context.Background()); err != nil {
		return p, err
	}
	src, err := e.materialize(hash, ".mp4")
	if err != nil {
		return p, err
	}
	defer os.Remove(src)
	out := src + ".grade.mp4"
	defer os.Remove(out)
	if err := e.cutBin().Grade(src, out, preset); err != nil {
		return p, err
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return p, err
	}
	h, err := e.PutBytes(raw)
	if err != nil {
		return p, err
	}
	p.ResultHash = h
	return e.saveCut(p)
}

// CutSetStem 按 music、sfx、voice、dub、original 写入对应音轨。其他种类返回 unknown stem。
// id 传给 GetCutProject。
// kind 去掉空白并转成小写后选择 music、sfx、voice、dub 或 original，其他值返回 unknown stem。
// hash 去掉空白后写入对应音轨。
func (e *Engine) CutSetStem(id, kind, hash string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	hash = strings.TrimSpace(hash)
	switch strings.ToLower(strings.TrimSpace(kind)) {
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
	default:
		return p, fmt.Errorf("unknown stem %s", kind)
	}
	return e.saveCut(p)
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

// CutKeep 只留下指定片段，再按已省略的字幕挖掉对应区间。已有成片时标成过期。
// id 传给 GetCutProject。
// clipIDs 去掉空白后非空的值收成保留集合，只留下 ID 在其中的拆解片段。
func (e *Engine) CutKeep(id string, clipIDs []string) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	want := map[string]bool{}
	for _, clipID := range clipIDs {
		clipID = strings.TrimSpace(clipID)
		if clipID != "" {
			want[clipID] = true
		}
	}
	var kept []cut.Clip
	for _, c := range p.Breakdown.Clips {
		if want[c.ID] {
			kept = append(kept, c)
		}
	}
	for _, cue := range p.Captions.Cues {
		if cue.Omit {
			kept = cut.Punch(kept, cue.Start, cue.End)
		}
	}
	if kept == nil {
		kept = []cut.Clip{}
	}
	p.EditList.Cuts = kept
	if p.ResultHash != "" {
		p.Stale = true
	}
	return e.saveCut(p)
}

// CutOmitCue 切换一条字幕的省略状态。没有剪辑表时先铺整段或复制拆解片段，省略时挖掉区间，恢复时插回。已有成片时标成过期。
// id 传给 GetCutProject。
// index 小于 0 或大于等于字幕条数时返回 cue not found，否则用来切换该条的 Omit。
func (e *Engine) CutOmitCue(id string, index int) (CutProject, error) {
	p, err := e.GetCutProject(id)
	if err != nil {
		return CutProject{}, err
	}
	if index < 0 || index >= len(p.Captions.Cues) {
		return p, fmt.Errorf("cue not found")
	}
	cue := p.Captions.Cues[index]
	cue.Omit = !cue.Omit
	p.Captions.Cues[index] = cue
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
	if p.ResultHash != "" {
		p.Stale = true
	}
	return e.saveCut(p)
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
		e.OnHub("video_route", out)
	}
	return out, nil
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
