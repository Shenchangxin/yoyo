package video

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

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
	return e.saveCut(p)
}

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
	return e.saveCut(p)
}

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
	return e.saveCut(p)
}

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
	raw, err := e.ttsOpenAI(c.Endpoint, key, model, text, voice)
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

func (e *Engine) posterFromHash(hash string) (string, error) {
	path := e.FilePath(hash)
	if path == "" {
		src, err := e.materialize(hash, ".mp4")
		if err != nil {
			return "", err
		}
		defer os.Remove(src)
		path = src
	}
	jpg, err := e.Poster(path)
	if err != nil {
		return "", err
	}
	return e.PutBytes(jpg)
}

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

func (e *Engine) VideoRoute(sessionID, dest, id, title, intent string) (map[string]any, error) {
	dest = strings.ToLower(strings.TrimSpace(dest))
	sessionID = strings.TrimSpace(sessionID)
	id = strings.TrimSpace(id)
	out := map[string]any{"dest": dest}
	switch dest {
	case "studio", "shorts", "短片":
		if id == "" {
			p, err := e.CreateStudioProject(StudioProject{Title: firstNonEmpty(title, "短片")})
			if err != nil {
				return nil, err
			}
			id = p.ID
			out["project"] = p
		}
		if sessionID != "" {
			if err := e.BindStudioSession(sessionID, id); err != nil {
				return nil, err
			}
		}
		out["pane"] = "studio"
		out["id"] = id
	case "drama", "短剧":
		if id == "" {
			return nil, fmt.Errorf("drama dest needs an episode id")
		}
		if sessionID != "" {
			if err := e.BindSession(sessionID, id); err != nil {
				return nil, err
			}
		}
		out["pane"] = "drama"
		out["id"] = id
	case "canvas", "画布":
		if id != "" && sessionID != "" {
			if err := e.BindCanvasSession(sessionID, id); err != nil {
				return nil, err
			}
		}
		out["pane"] = "canvas"
		out["id"] = id
	case "game", "游戏", "游戏世界", "game-world":
		return nil, fmt.Errorf("game worlds are no longer available; pick 短片, 短剧, or 画布")
	default:
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
		out["dest"] = "cut"
	}
	if e.OnHub != nil {
		e.OnHub("video_route", out)
	}
	return out, nil
}

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
