package video

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

func studioGateIndex(g string) int {
	g = strings.TrimSpace(g)
	for i, id := range studioGateOrder {
		if id == g {
			return i
		}
	}
	return 0
}

func (p StudioProject) FurthestGate() string {
	if strings.TrimSpace(p.Script) == "" && p.ScriptStatus != ScriptApproved {
		return GateScript
	}
	if p.ScriptStatus != ScriptApproved {
		return GateScript
	}
	if p.NarrationHash == "" && !p.SkipAudio {
		return GateAudio
	}
	if !p.VisualsReady() {
		return GateVisuals
	}
	if p.ResultHash == "" {
		return GateCompose
	}
	return GateDelivery
}

func (p StudioProject) CanEnter(gate string) bool {
	return studioGateIndex(gate) <= studioGateIndex(p.FurthestGate())
}

func (p StudioProject) VisualsReady() bool {
	shots := p.Storyboard.Shots
	if len(shots) == 0 {
		return false
	}
	for _, s := range shots {
		if strings.TrimSpace(s.ImageHash) == "" && strings.TrimSpace(s.VideoHash) == "" {
			return false
		}
		if s.Stale {
			return false
		}
	}
	return true
}

func ensureShotIDs(shots []StudioShot) []StudioShot {
	for i := range shots {
		if strings.TrimSpace(shots[i].ID) == "" {
			shots[i].ID = NewID()
		}
		if strings.TrimSpace(shots[i].Motion) == "" {
			shots[i].Motion = "ken-burns"
		}
		if strings.TrimSpace(shots[i].Visual) == "" {
			shots[i].Visual = shots[i].Prompt
		}
		if shots[i].Duration <= 0 {
			shots[i].Duration = 3
		}
	}
	return shots
}

func (e *Engine) mergeStudioShots(cur, next []StudioShot) []StudioShot {
	byID := map[string]StudioShot{}
	for _, s := range cur {
		if s.ID != "" {
			byID[s.ID] = s
		}
	}
	out := make([]StudioShot, 0, len(next))
	for _, s := range next {
		if old, ok := byID[s.ID]; ok {
			visualChanged := strings.TrimSpace(s.Visual) != strings.TrimSpace(old.Visual) || strings.TrimSpace(s.Prompt) != strings.TrimSpace(old.Prompt)
			if visualChanged && (old.ImageHash != "" || old.VideoHash != "") {
				s.Stale = true
				if s.PrevImage == "" {
					s.PrevImage = old.ImageHash
				}
				if s.PrevVideo == "" {
					s.PrevVideo = old.VideoHash
				}
			}
			if s.ImageHash == "" {
				s.ImageHash = old.ImageHash
			}
			if s.VideoHash == "" {
				s.VideoHash = old.VideoHash
			}
			if s.JobID == "" {
				s.JobID = old.JobID
			}
		}
		out = append(out, s)
	}
	return out
}

func (e *Engine) noteStudio(p *StudioProject, kind, message string) {
	if p == nil {
		return
	}
	p.Activities = append(p.Activities, StudioActivity{
		ID: NewID(), At: Now(), Actor: "system", Kind: kind, Message: message,
	})
	if len(p.Activities) > 50 {
		p.Activities = p.Activities[len(p.Activities)-50:]
	}
}

func (e *Engine) flagStudio(p *StudioProject, gate, message string) {
	if p == nil || strings.TrimSpace(message) == "" {
		return
	}
	p.Issues = append(p.Issues, StudioIssue{
		ID: NewID(), At: Now(), Level: "error", Gate: gate, Message: message,
	})
	if len(p.Issues) > 20 {
		p.Issues = p.Issues[len(p.Issues)-20:]
	}
}

func (e *Engine) saveStudio(p StudioProject) (StudioProject, error) {
	p.Revision++
	p.UpdatedAt = Now()
	if err := e.putDoc(colStudioProjects, p.ID, p); err != nil {
		return StudioProject{}, err
	}
	return p, nil
}

func (e *Engine) ApplyStudioPreset(id, presetID string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	preset, ok := LookupStudioPreset(presetID)
	if !ok {
		return StudioProject{}, fmt.Errorf("unknown preset %q", presetID)
	}
	p.PresetID = preset.ID
	p.Aspect = preset.Aspect
	p.Platform = preset.Platform
	p.TargetDuration = preset.Duration
	w, h := aspectSize(p.Aspect)
	p.Storyboard.Size = fmt.Sprintf("%dx%d", w, h)
	e.noteStudio(&p, "preset", "locked "+preset.Name)
	return e.saveStudio(p)
}

func (e *Engine) ApplyStudioTemplate(id, templateID string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	tpl, ok := LookupStudioTemplate(templateID)
	if !ok {
		return StudioProject{}, fmt.Errorf("unknown template %q", templateID)
	}
	if _, err := e.ApplyStudioPreset(id, tpl.PresetID); err == nil {
		p, _ = e.GetStudioProject(id)
	}
	p.TemplateID = tpl.ID
	p.Style = tpl.Style
	p.Platform = tpl.Platform
	p.Tone = tpl.Tone
	p.Identity = identityFromCatalog(tpl.CharacterID, tpl.Style)
	if p.Title == "" || p.Title == "短片" {
		p.Title = tpl.Title
	}
	e.noteStudio(&p, "template", "started from "+tpl.Title)
	return e.saveStudio(p)
}

func identityFromCatalog(characterID, styleID string) StudioIdentity {
	ident := DefaultStudioIdentity()
	if c, ok := LookupStudioCharacter(characterID); ok {
		ident.CharacterID = c.ID
		ident.Name = c.Name
		ident.Preserve = c.Preserve
		ident.Prohibit = c.Prohibit
	}
	if s, ok := LookupStudioStyle(styleID); ok {
		ident.StyleID = s.ID
		ident.Preserve = strings.TrimSpace(ident.Preserve + "; " + s.Contract)
		ident.Prohibit = strings.TrimSpace(ident.Prohibit + "; " + s.Forbid)
	}
	return ident
}

func (e *Engine) SetStudioIdentity(id, characterID, styleID string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	p.Identity = identityFromCatalog(characterID, styleID)
	if styleID != "" {
		p.Style = styleID
	}
	e.noteStudio(&p, "identity", p.Identity.Name)
	return e.saveStudio(p)
}

func (e *Engine) ApproveStudioScript(id string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	p.Script = strings.TrimSpace(p.Script)
	if p.Script == "" {
		return p, fmt.Errorf("write a spoken script first")
	}
	if len(p.Segments) == 0 {
		p.Segments = parseSpokenSegments(p.Script, p.TargetDuration)
	}
	if p.Hook == "" && len(p.Segments) > 0 {
		p.Hook = p.Segments[0].Narration
	}
	if p.CTA == "" && len(p.Segments) > 0 {
		p.CTA = p.Segments[len(p.Segments)-1].Narration
	}
	if len(p.Storyboard.Shots) == 0 {
		p.Storyboard.Shots = shotsFromSegments(p.Segments)
	}
	p.ScriptStatus = ScriptApproved
	p.Gate = GateAudio
	e.noteStudio(&p, "approve", "script locked")
	return e.saveStudio(p)
}

func (e *Engine) SkipStudioAudio(id string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	if p.ScriptStatus != ScriptApproved {
		return p, fmt.Errorf("approve the script first")
	}
	p.SkipAudio = true
	p.Gate = GateVisuals
	e.noteStudio(&p, "skip-audio", "operator skipped voice")
	return e.saveStudio(p)
}

func parseSpokenSegments(script string, targetSec int) []StudioSegment {
	parts := splitSpeak(script)
	if len(parts) == 0 {
		return nil
	}
	if targetSec <= 0 {
		targetSec = 60
	}
	each := float64(targetSec) / float64(len(parts))
	if each < 2 {
		each = 2
	}
	out := make([]StudioSegment, 0, len(parts))
	for i, text := range parts {
		kind := "body"
		if i == 0 {
			kind = "hook"
		} else if i == len(parts)-1 {
			kind = "cta"
		}
		out = append(out, StudioSegment{
			ID: NewID(), Kind: kind, Narration: text,
			Visual: text, Duration: each,
		})
	}
	return out
}

func shotsFromSegments(segs []StudioSegment) []StudioShot {
	out := make([]StudioShot, 0, len(segs))
	for _, s := range segs {
		dur := s.Duration
		if dur <= 0 {
			dur = 3
		}
		out = append(out, StudioShot{
			ID: NewID(), Caption: s.Narration, Visual: s.Visual, Prompt: s.Visual,
			Duration: dur, Motion: "ken-burns",
		})
	}
	return ensureShotIDs(out)
}

func (e *Engine) ApplyStudioTiming(id string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	if p.NarrationHash == "" {
		return p, fmt.Errorf("speak first")
	}
	durs := durationsFromSRT(p, e)
	if sec, err := e.ProbeSeconds(p.NarrationHash); err == nil && sec > 0 {
		if len(durs) > 0 {
			var sum float64
			for _, d := range durs {
				sum += d
			}
			if sum > 0 {
				scale := sec / sum
				for i := range durs {
					durs[i] *= scale
				}
			}
		} else if len(p.Storyboard.Shots) > 0 {
			each := sec / float64(len(p.Storyboard.Shots))
			durs = make([]float64, len(p.Storyboard.Shots))
			for i := range durs {
				durs[i] = each
			}
		}
	}
	if len(durs) == 0 {
		return p, fmt.Errorf("could not read narration timing")
	}
	if len(p.Storyboard.Shots) == 0 {
		p.Storyboard.Shots = shotsFromSegments(p.Segments)
	}
	for i := range p.Storyboard.Shots {
		if i < len(durs) && durs[i] > 0 {
			p.Storyboard.Shots[i].Duration = durs[i]
		}
	}
	p.Gate = GateVisuals
	e.noteStudio(&p, "timing", "shot lengths follow voice")
	return e.saveStudio(p)
}

func durationsFromSRT(p StudioProject, e *Engine) []float64 {
	if p.SubtitleHash == "" || e == nil {
		return nil
	}
	raw, err := e.GetBytes(p.SubtitleHash)
	if err != nil {
		return nil
	}
	return parseSRTDurations(string(raw))
}

func parseSRTDurations(s string) []float64 {
	var out []float64
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for _, line := range lines {
		if !strings.Contains(line, "-->") {
			continue
		}
		parts := strings.Split(line, "-->")
		if len(parts) != 2 {
			continue
		}
		a := parseSRTStamp(strings.TrimSpace(parts[0]))
		b := parseSRTStamp(strings.TrimSpace(parts[1]))
		if b > a {
			out = append(out, b-a)
		}
	}
	return out
}

func parseSRTStamp(s string) float64 {
	s = strings.ReplaceAll(s, ",", ".")
	var h, m int
	var sec float64
	fmt.Sscanf(s, "%d:%d:%f", &h, &m, &sec)
	return float64(h*3600+m*60) + sec
}

func (e *Engine) ProbeSeconds(hash string) (float64, error) {
	if e.FFProbe == "" {
		return 0, fmt.Errorf("ffprobe missing")
	}
	path, err := e.materialize(hash, ".bin")
	if err != nil {
		return 0, err
	}
	defer osRemove(path)
	return probeFileSeconds(e.FFProbe, path)
}

type StudioPreflight struct {
	Gate     string   `json:"gate"`
	Ok       bool     `json:"ok"`
	Spends   []string `json:"spends,omitempty"`
	Missing  []string `json:"missing,omitempty"`
	Message  string   `json:"message,omitempty"`
	Furthest string   `json:"furthest"`
}

func (e *Engine) StudioPreflight(id, gate string) (StudioPreflight, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioPreflight{}, err
	}
	gate = strings.TrimSpace(gate)
	if gate == "" {
		gate = p.FurthestGate()
	}
	out := StudioPreflight{Gate: gate, Furthest: p.FurthestGate(), Ok: true}
	need := func(cap, label string) {
		out.Spends = append(out.Spends, label)
		if e.Conn == nil {
			out.Missing = append(out.Missing, label)
			out.Ok = false
			return
		}
		if _, err := e.ActiveProvider(cap, ""); err != nil {
			out.Missing = append(out.Missing, label)
			out.Ok = false
		}
	}
	switch gate {
	case GateAudio:
		need("speech", "speech")
	case GateVisuals:
		need("image", "image")
	case GateCompose:
		if e.FFMPEG == "" {
			out.Missing = append(out.Missing, "ffmpeg")
			out.Ok = false
		} else {
			out.Spends = append(out.Spends, "ffmpeg")
		}
	case GateDelivery:
		if p.ResultHash == "" {
			out.Missing = append(out.Missing, "assembled video")
			out.Ok = false
		}
	}
	if !out.Ok {
		out.Message = "configure " + strings.Join(out.Missing, ", ") + " in Settings → Generation"
	}
	return out, nil
}

func (e *Engine) PackageStudio(id string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	if p.ResultHash == "" {
		return p, fmt.Errorf("assemble first")
	}
	if p.CopyText == "" {
		p.CopyText = defaultPublishCopy(p)
	}
	if p.CopyHash == "" && e.CAS != nil {
		if h, err := e.PutBytes([]byte(p.CopyText)); err == nil {
			p.CopyHash = h
		}
	}
	if p.CoverHash == "" {
		p.CoverHash = p.PosterHash
	}
	files := []StudioDeliveryFile{
		{Kind: "video", Hash: p.ResultHash, MIME: "video/mp4"},
		{Kind: "subtitle", Hash: p.SubtitleHash, MIME: "application/x-subrip"},
		{Kind: "cover", Hash: p.CoverHash, MIME: "image/jpeg"},
		{Kind: "copy", Hash: p.CopyHash, MIME: "text/markdown"},
	}
	status := DeliveryDraft
	ready := 0
	for i, f := range files {
		if f.Hash == "" {
			continue
		}
		if raw, err := e.GetBytes(f.Hash); err == nil {
			files[i].Bytes = int64(len(raw))
			ready++
		}
	}
	note := ""
	if p.ResultHash != "" && p.CopyHash != "" {
		status = DeliveryPublishable
		if p.CoverHash == "" {
			note = "cover missing — generate or assemble a poster"
		}
	} else {
		note = "video is ready; cover or copy still missing"
	}
	p.Delivery = StudioDelivery{
		Status: status, VideoHash: p.ResultHash, SubtitleHash: p.SubtitleHash,
		CoverHash: p.CoverHash, CopyHash: p.CopyHash, CopyText: p.CopyText,
		Files: files, Note: note,
	}
	p.Gate = GateDelivery
	snap := StudioSnapshot{
		N: len(p.Versions) + 1, ResultHash: p.ResultHash, PosterHash: p.PosterHash,
		NarrationHash: p.NarrationHash, SubtitleHash: p.SubtitleHash,
		CoverHash: p.CoverHash, CopyHash: p.CopyHash, Note: "delivery", CreatedAt: Now(),
	}
	p.Versions = append(p.Versions, snap)
	e.noteStudio(&p, "package", status)
	return e.saveStudio(p)
}

func (e *Engine) RestoreStudioVersion(id string, n int) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	var snap StudioSnapshot
	for _, v := range p.Versions {
		if v.N == n {
			snap = v
			break
		}
	}
	if snap.N == 0 {
		return p, fmt.Errorf("version %d not found", n)
	}
	p.ResultHash = snap.ResultHash
	p.PosterHash = snap.PosterHash
	p.NarrationHash = snap.NarrationHash
	p.SubtitleHash = snap.SubtitleHash
	p.CoverHash = snap.CoverHash
	p.CopyHash = snap.CopyHash
	e.noteStudio(&p, "restore", fmt.Sprintf("V%d", n))
	return e.saveStudio(p)
}

func defaultPublishCopy(p StudioProject) string {
	platform := p.Platform
	if platform == "" {
		platform = "douyin"
	}
	title := strings.TrimSpace(p.Title)
	body := strings.TrimSpace(p.Script)
	if utf8.RuneCountInString(body) > 180 {
		runes := []rune(body)
		body = string(runes[:180]) + "…"
	}
	tags := "#短视频"
	switch platform {
	case "douyin":
		tags = "#抖音 #短视频"
	case "xhs":
		tags = "#小红书 #短视频"
	case "bilibili":
		tags = "#B站 #短片"
	case "channels":
		tags = "#视频号"
	}
	return "# " + title + "\n\n" + body + "\n\n" + tags + "\n"
}

func osRemove(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}
