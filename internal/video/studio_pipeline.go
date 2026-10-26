// 短片的闸门、口播锁定、镜头时长和交付包。
// 能走到哪一闸，看口播、配音、画面和成片是否齐。合成本身不在这里。

package video

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// studioGateIndex 返回闸门在顺序里的位置。认不出的名字返回 0，也就是 GateSetup。
func studioGateIndex(g string) int {
	g = strings.TrimSpace(g)
	for i, id := range studioGateOrder {
		if id == g {
			return i
		}
	}
	return 0
}

// FurthestGate 按现有材料给出最远能进的闸门。没锁定口播停在口播，没配音且没跳过停在配音，画面不齐停在画面，没成片停在合成，否则是交付。
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

// CanEnter 判断这个闸门没有超过 FurthestGate。
func (p StudioProject) CanEnter(gate string) bool {
	return studioGateIndex(gate) <= studioGateIndex(p.FurthestGate())
}

// VisualsReady 在每一镜都有图或视频、且没有过期时返回真。一镜都没有就返回假。
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

// ensureShotIDs 给缺编号的镜头补编号。运镜空着写成 ken-burns，画面说明空着就用提示，时长小于等于 0 就写成 3 秒。
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

// mergeStudioShots 按编号合并镜头。新镜头没带图、视频或任务编号时留下原来的。画面说明或提示变了，且原来已有图或视频，就标成过期并记下上一版。
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

// noteStudio 追加一条系统记录。超过 50 条就只留最新的 50 条。工程是空指针就什么都不做。
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

// flagStudio 记一条错误，级别是 error。
func (e *Engine) flagStudio(p *StudioProject, gate, message string) {
	e.addStudioIssue(p, "error", gate, message)
}

// warnStudio 记一条警告。同一闸、同一句话的旧警告会先去掉，避免重复。空消息不记。
func (e *Engine) warnStudio(p *StudioProject, gate, message string) {
	if p == nil || strings.TrimSpace(message) == "" {
		return
	}
	kept := p.Issues[:0]
	for _, iss := range p.Issues {
		if iss.Gate == gate && iss.Message == message {
			continue
		}
		kept = append(kept, iss)
	}
	p.Issues = kept
	e.addStudioIssue(p, "warn", gate, message)
}

// addStudioIssue 追加一条问题。超过 20 条就只留最新的 20 条。空消息不记。
func (e *Engine) addStudioIssue(p *StudioProject, level, gate, message string) {
	if p == nil || strings.TrimSpace(message) == "" {
		return
	}
	p.Issues = append(p.Issues, StudioIssue{
		ID: NewID(), At: Now(), Level: level, Gate: gate, Message: message,
	})
	if len(p.Issues) > 20 {
		p.Issues = p.Issues[len(p.Issues)-20:]
	}
}

// saveStudio 把修订号加一并写回工程。编号必须已经有。
func (e *Engine) saveStudio(p StudioProject) (StudioProject, error) {
	p.Revision++
	p.UpdatedAt = Now()
	if err := e.putDoc(colStudioProjects, p.ID, p); err != nil {
		return StudioProject{}, err
	}
	return p, nil
}

// ApplyStudioPreset 把预设的画幅、平台和时长写上工程，并改分镜尺寸。预设不存在就拒绝。
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

// ApplyStudioTemplate 先套模板上的预设，再写风格、平台、语气和身份。标题还是空的或「短片」时改成模板标题。模板不存在就拒绝。
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

// identityFromCatalog 用目录里的角色和风格拼身份。风格的约定和禁止会接在角色的后面。两边都没有就用默认口播主持。
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

// SetStudioIdentity 按目录重写身份。风格编号非空时也写到工程的 Style。
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

// ApproveStudioScript 锁住口播。正文是空的就拒绝。还没有分段就按句子拆开，没有镜头就按分段生成。钩子和收束空着时用第一段和最后一段。闸门改到配音。
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

// SkipStudioAudio 在口播已锁定后跳过配音，闸门改到画面。还没锁定就拒绝。
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

// parseSpokenSegments 按句子拆口播。第一段是钩子，最后一段是收束，中间是正文。目标秒数空着按 60 秒，再平分到每段，每段至少 2 秒。
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

// shotsFromSegments 按分段生成镜头。字幕和画面说明都用这段口播，运镜是 ken-burns。时长小于等于 0 就写成 3 秒。
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

// ApplyStudioTiming 让镜头时长跟着口播走。还没有配音就拒绝。
// 有字幕时先读每条字幕的时长，再用探测到的音频总长按比例拉开。没有字幕但探测到时长，就平分到现有镜头。闸门改到画面。
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

// durationsFromSRT 从工程的字幕文件读出每条的时长。没有字幕或读不到就返回 nil。
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

// parseSRTDurations 读出 SRT 里每一条的时长。没有箭头的行会跳过。结束不大于开始的也跳过。
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

// parseSRTStamp 把 SRT 时间戳换成秒。逗号会先换成小数点。
func parseSRTStamp(s string) float64 {
	s = strings.ReplaceAll(s, ",", ".")
	var h, m int
	var sec float64
	fmt.Sscanf(s, "%d:%d:%f", &h, &m, &sec)
	return float64(h*3600+m*60) + sec
}

// ProbeSeconds 用 ffprobe 读一份内容库媒体的时长。没有 ffprobe 或取不出文件就返回错误。
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

// StudioPreflight 是进某一闸之前，还缺什么、会用到什么。
type StudioPreflight struct {
	Gate string `json:"gate"`
	Ok   bool   `json:"ok"`
	// Spends 是这一闸可能会用到的能力。
	Spends []string `json:"spends,omitempty"`
	// Missing 是这一闸还没配上的能力。
	Missing []string `json:"missing,omitempty"`
	Message string   `json:"message,omitempty"`
	// Furthest 是这则工程当前最远能到的闸门。
	Furthest string `json:"furthest"`
}

// StudioPreflight 检查进这一闸还缺什么。闸门空着就用 FurthestGate。
// 配音看语音连接，画面看图像连接，合成看 ffmpeg，交付看成片在不在。缺了就把 Ok 设成假。
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

// PackageStudio 打交付包。还没有成片就拒绝。文案空着就按平台写一版，封面空着就用海报。
// 成片和文案都在时标成可发布。封面仍缺会写在备注里。成功后闸门改到交付，并追加一个版本。
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

// RestoreStudioVersion 把成片、海报、口播、字幕、封面和文案编号恢复成第 n 版。不改口播正文和分镜。找不到这一版就拒绝。
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

// defaultPublishCopy 用标题、口播和平台标签拼一段发布文案。口播超过 180 个字就截断。平台空着按抖音。
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

// osRemove 删掉临时文件。路径空着或删除失败都忽略。
func osRemove(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}
