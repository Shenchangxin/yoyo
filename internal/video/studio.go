package video

import (
	"fmt"
	"sort"
	"strings"
)

const (
	StudioDraft     = "draft"
	StudioAssembled = "assembled"
	StudioFailed    = "failed"

	GateSetup    = "setup"
	GateScript   = "script"
	GateAudio    = "audio"
	GateVisuals  = "visuals"
	GateCompose  = "compose"
	GateDelivery = "delivery"

	ScriptNeedsReview = "needs_review"
	ScriptApproved    = "approved"

	DeliveryDraft       = "draft"
	DeliveryPublishable = "publishable"
)

var studioGateOrder = []string{GateSetup, GateScript, GateAudio, GateVisuals, GateCompose, GateDelivery}

type StudioShot struct {
	ID        string  `json:"id,omitempty"`
	ImageHash string  `json:"image_hash,omitempty"`
	VideoHash string  `json:"video_hash,omitempty"`
	Caption   string  `json:"caption,omitempty"`
	Duration  float64 `json:"duration,omitempty"`
	Motion    string  `json:"motion,omitempty"`
	Prompt    string  `json:"prompt,omitempty"`
	Visual    string  `json:"visual,omitempty"`
	Stale     bool    `json:"stale,omitempty"`
	JobID     string  `json:"job_id,omitempty"`
	PrevImage string  `json:"prev_image_hash,omitempty"`
	PrevVideo string  `json:"prev_video_hash,omitempty"`
}

type StudioSegment struct {
	ID        string  `json:"id,omitempty"`
	Kind      string  `json:"kind,omitempty"`
	Narration string  `json:"narration,omitempty"`
	Visual    string  `json:"visual,omitempty"`
	Duration  float64 `json:"duration,omitempty"`
}

type StudioIdentity struct {
	CharacterID string `json:"character_id,omitempty"`
	StyleID     string `json:"style_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Preserve    string `json:"preserve,omitempty"`
	Prohibit    string `json:"prohibit,omitempty"`
	RefHash     string `json:"ref_hash,omitempty"`
}

type StudioSnapshot struct {
	N             int    `json:"n"`
	ResultHash    string `json:"result_hash,omitempty"`
	PosterHash    string `json:"poster_hash,omitempty"`
	NarrationHash string `json:"narration_hash,omitempty"`
	SubtitleHash  string `json:"subtitle_hash,omitempty"`
	CoverHash     string `json:"cover_hash,omitempty"`
	CopyHash      string `json:"copy_hash,omitempty"`
	Note          string `json:"note,omitempty"`
	CreatedAt     string `json:"created_at"`
}

type StudioDeliveryFile struct {
	Kind  string `json:"kind"`
	Hash  string `json:"hash,omitempty"`
	MIME  string `json:"mime,omitempty"`
	Bytes int64  `json:"bytes,omitempty"`
}

type StudioDelivery struct {
	Status       string               `json:"status,omitempty"`
	VideoHash    string               `json:"video_hash,omitempty"`
	SubtitleHash string               `json:"subtitle_hash,omitempty"`
	CoverHash    string               `json:"cover_hash,omitempty"`
	CopyHash     string               `json:"copy_hash,omitempty"`
	CopyText     string               `json:"copy_text,omitempty"`
	Files        []StudioDeliveryFile `json:"files,omitempty"`
	Note         string               `json:"note,omitempty"`
}

type StudioActivity struct {
	ID      string `json:"id"`
	At      string `json:"at"`
	Actor   string `json:"actor,omitempty"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type StudioIssue struct {
	ID      string `json:"id"`
	At      string `json:"at"`
	Level   string `json:"level,omitempty"`
	Gate    string `json:"gate,omitempty"`
	Message string `json:"message"`
}

type StudioStoryboard struct {
	Size        string       `json:"size,omitempty"`
	ImageMotion string       `json:"image_motion,omitempty"`
	Shots       []StudioShot `json:"shots,omitempty"`
	Narration   string       `json:"narration,omitempty"`
	BGM         string       `json:"bgm,omitempty"`
	Subtitle    string       `json:"subtitle,omitempty"`
	BGMVolume   float64      `json:"bgm_volume,omitempty"`
}

type StudioProject struct {
	ID             string           `json:"id"`
	Title          string           `json:"title"`
	TopicID        string           `json:"topic_id,omitempty"`
	Aspect         string           `json:"aspect,omitempty"`
	Style          string           `json:"style,omitempty"`
	PresetID       string           `json:"preset_id,omitempty"`
	TemplateID     string           `json:"template_id,omitempty"`
	Platform       string           `json:"platform,omitempty"`
	Tone           string           `json:"tone,omitempty"`
	Audience       string           `json:"audience,omitempty"`
	TargetDuration int              `json:"target_duration,omitempty"`
	Revision       int              `json:"revision,omitempty"`
	Gate           string           `json:"gate,omitempty"`
	Script         string           `json:"script,omitempty"`
	ScriptHash     string           `json:"script_hash,omitempty"`
	ScriptStatus   string           `json:"script_status,omitempty"`
	Hook           string           `json:"hook,omitempty"`
	CTA            string           `json:"cta,omitempty"`
	Segments       []StudioSegment  `json:"segments,omitempty"`
	Identity       StudioIdentity   `json:"identity"`
	SkipAudio      bool             `json:"skip_audio,omitempty"`
	Voice          string           `json:"voice,omitempty"`
	VoiceStyle     string           `json:"voice_style,omitempty"`
	VoiceSpeed     float64          `json:"voice_speed,omitempty"`
	Storyboard     StudioStoryboard `json:"storyboard"`
	NarrationHash  string           `json:"narration_hash,omitempty"`
	SubtitleHash   string           `json:"subtitle_hash,omitempty"`
	BGMHash        string           `json:"bgm_hash,omitempty"`
	ResultHash     string           `json:"result_hash,omitempty"`
	PosterHash     string           `json:"poster_hash,omitempty"`
	CoverHash      string           `json:"cover_hash,omitempty"`
	Covers         []string         `json:"covers,omitempty"`
	CopyHash       string           `json:"copy_hash,omitempty"`
	CopyText       string           `json:"copy_text,omitempty"`
	Versions       []StudioSnapshot `json:"versions,omitempty"`
	Delivery       StudioDelivery   `json:"delivery"`
	Activities     []StudioActivity `json:"activities,omitempty"`
	Issues         []StudioIssue    `json:"issues,omitempty"`
	AssetID        string           `json:"canvas_asset_id,omitempty"`
	Status         string           `json:"status"`
	Error          string           `json:"error,omitempty"`
	CreatedAt      string           `json:"created_at"`
	UpdatedAt      string           `json:"updated_at"`
}

func normalizeAspect(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "16:9", "1920x1080", "16x9":
		return "16:9"
	case "1:1", "1080x1080":
		return "1:1"
	default:
		return "9:16"
	}
}

func aspectSize(aspect string) (w, h int) {
	switch normalizeAspect(aspect) {
	case "16:9":
		return 1920, 1080
	case "1:1":
		return 1080, 1080
	default:
		return 1080, 1920
	}
}

func (e *Engine) ListStudioProjects() []StudioProject {
	var out []StudioProject
	for _, p := range loadCol[StudioProject](e, colStudioProjects) {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	if out == nil {
		return []StudioProject{}
	}
	return out
}

func (e *Engine) GetStudioProject(id string) (StudioProject, error) {
	p, err := getDoc[StudioProject](e, colStudioProjects, strings.TrimSpace(id))
	if err != nil {
		return StudioProject{}, fmt.Errorf("studio project not found")
	}
	return p, nil
}

func (e *Engine) CreateStudioProject(in StudioProject) (StudioProject, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		in.Title = "短片"
	}
	now := Now()
	in.ID = NewID()
	in.Aspect = normalizeAspect(in.Aspect)
	in.Status = StudioDraft
	in.ScriptStatus = ScriptNeedsReview
	in.Gate = GateScript
	in.Revision = 1
	if in.PresetID == "" {
		in.PresetID = presetIDForAspect(in.Aspect)
	}
	if preset, ok := LookupStudioPreset(in.PresetID); ok {
		in.Aspect = preset.Aspect
		if in.Platform == "" {
			in.Platform = preset.Platform
		}
		if in.TargetDuration == 0 {
			in.TargetDuration = preset.Duration
		}
	}
	if in.Tone == "" {
		in.Tone = "natural"
	}
	if in.Identity.CharacterID == "" {
		in.Identity = DefaultStudioIdentity()
	}
	in.CreatedAt = now
	in.UpdatedAt = now
	in.Storyboard.Shots = ensureShotIDs(in.Storyboard.Shots)
	if in.Storyboard.Size == "" {
		w, h := aspectSize(in.Aspect)
		in.Storyboard.Size = fmt.Sprintf("%dx%d", w, h)
	}
	if err := e.putDoc(colStudioProjects, in.ID, in); err != nil {
		return StudioProject{}, err
	}
	if in.TopicID != "" {
		if t, err := e.GetTopic(in.TopicID); err == nil {
			t.StudioProjectID = in.ID
			t.Status = TopicInProduction
			_, _ = e.SaveTopic(t)
		}
	}
	return in, nil
}

func (e *Engine) UpdateStudioProject(in StudioProject) (StudioProject, error) {
	cur, err := e.GetStudioProject(in.ID)
	if err != nil {
		return StudioProject{}, err
	}
	if strings.TrimSpace(in.Title) != "" {
		cur.Title = strings.TrimSpace(in.Title)
	}
	if strings.TrimSpace(in.Aspect) != "" {
		cur.Aspect = normalizeAspect(in.Aspect)
		w, h := aspectSize(cur.Aspect)
		cur.Storyboard.Size = fmt.Sprintf("%dx%d", w, h)
	}
	if strings.TrimSpace(in.Style) != "" {
		cur.Style = strings.TrimSpace(in.Style)
	}
	if strings.TrimSpace(in.PresetID) != "" {
		cur.PresetID = strings.TrimSpace(in.PresetID)
	}
	if strings.TrimSpace(in.TemplateID) != "" {
		cur.TemplateID = strings.TrimSpace(in.TemplateID)
	}
	if strings.TrimSpace(in.Platform) != "" {
		cur.Platform = strings.TrimSpace(in.Platform)
	}
	if strings.TrimSpace(in.Tone) != "" {
		cur.Tone = strings.TrimSpace(in.Tone)
	}
	if strings.TrimSpace(in.Audience) != "" {
		cur.Audience = strings.TrimSpace(in.Audience)
	}
	if in.TargetDuration > 0 {
		cur.TargetDuration = in.TargetDuration
	}
	if strings.TrimSpace(in.Gate) != "" {
		cur.Gate = strings.TrimSpace(in.Gate)
	}
	if strings.TrimSpace(in.Hook) != "" {
		cur.Hook = strings.TrimSpace(in.Hook)
	}
	if strings.TrimSpace(in.CTA) != "" {
		cur.CTA = strings.TrimSpace(in.CTA)
	}
	if in.Segments != nil {
		cur.Segments = in.Segments
	}
	if in.Identity.CharacterID != "" || in.Identity.StyleID != "" || in.Identity.Preserve != "" {
		cur.Identity = in.Identity
	}
	if strings.TrimSpace(in.Voice) != "" {
		cur.Voice = strings.TrimSpace(in.Voice)
	}
	if strings.TrimSpace(in.VoiceStyle) != "" {
		cur.VoiceStyle = strings.TrimSpace(in.VoiceStyle)
	}
	if in.VoiceSpeed > 0 {
		cur.VoiceSpeed = in.VoiceSpeed
	}
	if in.SkipAudio {
		cur.SkipAudio = true
	}
	if strings.TrimSpace(in.ResultHash) != "" {
		cur.ResultHash = strings.TrimSpace(in.ResultHash)
	}
	if in.Script != "" {
		if cur.Script != in.Script && cur.ScriptStatus == ScriptApproved {
			cur.ScriptStatus = ScriptNeedsReview
		}
		cur.Script = in.Script
		if e.CAS != nil {
			if h, err := e.PutBytes([]byte(in.Script)); err == nil {
				cur.ScriptHash = h
			}
		}
	}
	if in.TopicID != "" {
		cur.TopicID = in.TopicID
	}
	if in.Storyboard.Shots != nil {
		cur.Storyboard.Shots = e.mergeStudioShots(cur.Storyboard.Shots, ensureShotIDs(in.Storyboard.Shots))
	}
	if in.Storyboard.Size != "" {
		cur.Storyboard.Size = in.Storyboard.Size
	}
	if in.Storyboard.ImageMotion != "" {
		cur.Storyboard.ImageMotion = in.Storyboard.ImageMotion
	}
	if in.Storyboard.Narration != "" {
		cur.Storyboard.Narration = in.Storyboard.Narration
	}
	if in.Storyboard.Subtitle != "" {
		cur.Storyboard.Subtitle = in.Storyboard.Subtitle
	}
	if in.Storyboard.BGM != "" {
		cur.Storyboard.BGM = in.Storyboard.BGM
	}
	if in.Storyboard.BGMVolume > 0 {
		cur.Storyboard.BGMVolume = in.Storyboard.BGMVolume
	}
	if in.NarrationHash != "" {
		cur.NarrationHash = in.NarrationHash
		cur.Storyboard.Narration = in.NarrationHash
	}
	if in.SubtitleHash != "" {
		cur.SubtitleHash = in.SubtitleHash
		cur.Storyboard.Subtitle = in.SubtitleHash
	}
	if in.BGMHash != "" {
		cur.BGMHash = in.BGMHash
		cur.Storyboard.BGM = in.BGMHash
	}
	cur.Revision++
	cur.UpdatedAt = Now()
	if err := e.putDoc(colStudioProjects, cur.ID, cur); err != nil {
		return StudioProject{}, err
	}
	return cur, nil
}

func (e *Engine) PutStudioProject(p StudioProject) error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("id required")
	}
	return e.putDoc(colStudioProjects, p.ID, p)
}

func (e *Engine) DeleteStudioProject(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	m := e.loadStudioBinds()
	for sid, pid := range m {
		if pid == id {
			delete(m, sid)
		}
	}
	_ = e.saveStudioBinds(m)
	return e.delDoc(colStudioProjects, id)
}

func (e *Engine) BindStudioSession(sessionID, projectID string) error {
	sessionID = strings.TrimSpace(sessionID)
	projectID = strings.TrimSpace(projectID)
	if sessionID == "" || projectID == "" {
		return fmt.Errorf("session and project required")
	}
	if _, err := e.GetStudioProject(projectID); err != nil {
		return err
	}
	m := e.loadStudioBinds()
	next := make(map[string]string, len(m)+1)
	for sid, pid := range m {
		if sid == sessionID || pid == projectID {
			continue
		}
		next[sid] = pid
	}
	next[sessionID] = projectID
	return e.saveStudioBinds(next)
}

func (e *Engine) StudioSessionBind(sessionID string) (string, bool) {
	m := e.loadStudioBinds()
	id := strings.TrimSpace(m[strings.TrimSpace(sessionID)])
	return id, id != ""
}

func (e *Engine) PublishStudioAsset(p StudioProject) (StudioProject, error) {
	if p.ResultHash == "" {
		return p, fmt.Errorf("no assembled video")
	}
	raw, err := e.GetBytes(p.ResultHash)
	if err != nil {
		return p, err
	}
	w, h := aspectSize(p.Aspect)
	res, err := e.putCanvasResourceFromHash(p.ResultHash, "video", "video/mp4", int64(len(raw)), w, h, 0)
	if err != nil {
		return p, err
	}
	asset := e.upsertAsset(map[string]any{
		"folderId":   "studio-folder-shorts",
		"kind":       "video",
		"category":   "studio",
		"title":      p.Title,
		"resourceId": res.ID,
		"status":     "confirmed",
	})
	p.AssetID = strAnyMap(asset, "id")
	p.UpdatedAt = Now()
	_ = e.putDoc(colStudioProjects, p.ID, p)
	return p, nil
}

// StudioJobShouldSkip reports that a delayed shoot must not start an agent:
// the operator already has a bound chat or has moved the short past script.
func StudioJobShouldSkip(p StudioProject, boundSession string) bool {
	if strings.TrimSpace(boundSession) != "" {
		return true
	}
	if p.ScriptStatus == ScriptApproved {
		return true
	}
	if strings.TrimSpace(p.ResultHash) != "" {
		return true
	}
	g := p.FurthestGate()
	return g != GateSetup && g != GateScript
}

func (e *Engine) StudioFromTopic(topicID, aspect string) (StudioProject, error) {
	t, err := e.GetTopic(topicID)
	if err != nil {
		return StudioProject{}, err
	}
	if t.StudioProjectID != "" {
		if p, err := e.GetStudioProject(t.StudioProjectID); err == nil {
			if strings.TrimSpace(aspect) != "" && normalizeAspect(aspect) != p.Aspect {
				p.Aspect = normalizeAspect(aspect)
				return e.UpdateStudioProject(p)
			}
			return p, nil
		}
	}
	title := t.Title
	if t.Angle != "" {
		title = t.Angle
	}
	return e.CreateStudioProject(StudioProject{
		Title:   title,
		TopicID: t.ID,
		Aspect:  aspect,
		Script:  strings.TrimSpace(strings.TrimSpace(t.Title) + "\n" + t.Angle + "\n" + t.Note),
	})
}
