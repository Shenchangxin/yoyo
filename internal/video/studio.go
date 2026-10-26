// 短片工程的状态、闸门和文档。创建、修改、删除和会话绑定在这里。
// 合成、配音和一键成片不在这个文件里。

package video

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// StudioDraft 是短片还没合成成片时的状态。
	StudioDraft = "draft"
	// StudioAssembled 是成片已经写进工程时的状态。
	StudioAssembled = "assembled"
	// StudioFailed 是合成失败时写上的状态。
	StudioFailed = "failed"

	// GateSetup 是闸门顺序的起点。认不出的闸门名也算在这一档。
	GateSetup = "setup"
	// GateScript 是还在写口播、尚未锁定的闸门。
	GateScript = "script"
	// GateAudio 是口播已锁定、还没配音或还没声明跳过配音的闸门。
	GateAudio = "audio"
	// GateVisuals 是配音已完成或已跳过、画面还没齐的闸门。
	GateVisuals = "visuals"
	// GateCompose 是画面已齐、成片还没写上的闸门。
	GateCompose = "compose"
	// GateDelivery 是成片已经有了、可以打包交付的闸门。
	GateDelivery = "delivery"

	// ScriptNeedsReview 表示口播还不能用来配音或生图。
	ScriptNeedsReview = "needs_review"
	// ScriptApproved 表示口播已锁定，配音和生图可以接着做。
	ScriptApproved = "approved"

	// DeliveryDraft 是交付包还缺成片或文案时的状态。
	DeliveryDraft = "draft"
	// DeliveryPublishable 是成片和文案都在时的交付状态。并不代表已经发出去。
	DeliveryPublishable = "publishable"
)

var studioGateOrder = []string{GateSetup, GateScript, GateAudio, GateVisuals, GateCompose, GateDelivery}

// StudioShot 是分镜里的一镜。
type StudioShot struct {
	ID        string  `json:"id,omitempty"`
	ImageHash string  `json:"image_hash,omitempty"`
	VideoHash string  `json:"video_hash,omitempty"`
	Caption   string  `json:"caption,omitempty"`
	Duration  float64 `json:"duration,omitempty"`
	Motion    string  `json:"motion,omitempty"`
	// Fit 是这一镜的 cover 或 contain。空着时用分镜上的 Fit，分镜也空着就按运镜决定铺满还是留边。
	Fit    string `json:"fit,omitempty"`
	Prompt string `json:"prompt,omitempty"`
	Visual string `json:"visual,omitempty"`
	// Stale 表示画面说明变了，已有的图或视频还不能拿去合成。
	Stale bool `json:"stale,omitempty"`
	// JobID 是这一镜最近一次生图或生视频的任务。排队时写上，结果回来后仍留着。
	JobID string `json:"job_id,omitempty"`
	// PrevImage 是换画面之前留下的上一张图。
	PrevImage string `json:"prev_image_hash,omitempty"`
	// PrevVideo 是换画面之前留下的上一段视频。
	PrevVideo string `json:"prev_video_hash,omitempty"`
}

// StudioSegment 是口播拆开后的一段，用来生成对应的镜头。
type StudioSegment struct {
	ID        string  `json:"id,omitempty"`
	Kind      string  `json:"kind,omitempty"`
	Narration string  `json:"narration,omitempty"`
	Visual    string  `json:"visual,omitempty"`
	Duration  float64 `json:"duration,omitempty"`
}

// StudioIdentity 是写进镜头提示的角色和风格约定。
type StudioIdentity struct {
	CharacterID string `json:"character_id,omitempty"`
	StyleID     string `json:"style_id,omitempty"`
	Name        string `json:"name,omitempty"`
	// Preserve 是生成时要保住的外形和风格，会写进提示。
	Preserve string `json:"preserve,omitempty"`
	// Prohibit 是生成时不要出现的东西，会写进提示。
	Prohibit string `json:"prohibit,omitempty"`
	// RefHash 是身份上的参考图。镜头自己还没有图时，生图会用它。
	RefHash string `json:"ref_hash,omitempty"`
}

// StudioSnapshot 是一次合成或打包时留下的版本。
type StudioSnapshot struct {
	// N 是从 1 起的版本号。
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

// StudioDeliveryFile 是交付包里的一个文件。
type StudioDeliveryFile struct {
	Kind  string `json:"kind"`
	Hash  string `json:"hash,omitempty"`
	MIME  string `json:"mime,omitempty"`
	Bytes int64  `json:"bytes,omitempty"`
}

// StudioDelivery 是准备交给操作者的成片、字幕、封面和文案。
type StudioDelivery struct {
	// Status 是 DeliveryDraft 或 DeliveryPublishable。
	Status       string               `json:"status,omitempty"`
	VideoHash    string               `json:"video_hash,omitempty"`
	SubtitleHash string               `json:"subtitle_hash,omitempty"`
	CoverHash    string               `json:"cover_hash,omitempty"`
	CopyHash     string               `json:"copy_hash,omitempty"`
	CopyText     string               `json:"copy_text,omitempty"`
	Files        []StudioDeliveryFile `json:"files,omitempty"`
	Note         string               `json:"note,omitempty"`
}

// StudioActivity 是工程上的一条操作记录，最多留 50 条。
type StudioActivity struct {
	ID      string `json:"id"`
	At      string `json:"at"`
	Actor   string `json:"actor,omitempty"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// StudioIssue 是某一闸上的错误或警告，最多留 20 条。
type StudioIssue struct {
	ID      string `json:"id"`
	At      string `json:"at"`
	Level   string `json:"level,omitempty"`
	Gate    string `json:"gate,omitempty"`
	Message string `json:"message"`
}

// StudioStoryboard 是整条短片的分镜、口播、字幕和配乐。
type StudioStoryboard struct {
	Size        string       `json:"size,omitempty"`
	ImageMotion string       `json:"image_motion,omitempty"`
	Shots       []StudioShot `json:"shots,omitempty"`
	Narration   string       `json:"narration,omitempty"`
	BGM         string       `json:"bgm,omitempty"`
	Subtitle    string       `json:"subtitle,omitempty"`
	BGMVolume   float64      `json:"bgm_volume,omitempty"`
	// Fit 是镜头没单独写时用的 cover 或 contain。
	Fit string `json:"fit,omitempty"`
}

// StudioProject 是一条短片工程。闸门和一键成片的进度都记在这里。
type StudioProject struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	TopicID        string `json:"topic_id,omitempty"`
	Aspect         string `json:"aspect,omitempty"`
	Style          string `json:"style,omitempty"`
	PresetID       string `json:"preset_id,omitempty"`
	TemplateID     string `json:"template_id,omitempty"`
	Platform       string `json:"platform,omitempty"`
	Tone           string `json:"tone,omitempty"`
	Audience       string `json:"audience,omitempty"`
	TargetDuration int    `json:"target_duration,omitempty"`
	// Revision 每次经 UpdateStudioProject 或 saveStudio 保存都会加一。
	Revision int `json:"revision,omitempty"`
	// Gate 是记下的闸门。能走到哪一步看 FurthestGate。
	Gate       string `json:"gate,omitempty"`
	Script     string `json:"script,omitempty"`
	ScriptHash string `json:"script_hash,omitempty"`
	// ScriptStatus 是 ScriptNeedsReview 或 ScriptApproved。改口播会把已锁定的退回待审。
	ScriptStatus string          `json:"script_status,omitempty"`
	Hook         string          `json:"hook,omitempty"`
	CTA          string          `json:"cta,omitempty"`
	Segments     []StudioSegment `json:"segments,omitempty"`
	Identity     StudioIdentity  `json:"identity"`
	// SkipAudio 为真表示口播锁定后跳过配音，直接去画面。
	SkipAudio  bool    `json:"skip_audio,omitempty"`
	Voice      string  `json:"voice,omitempty"`
	VoiceStyle string  `json:"voice_style,omitempty"`
	VoiceSpeed float64 `json:"voice_speed,omitempty"`
	// VoiceEngine 是 cloud 或 edge，记下这次配音走了哪条。
	VoiceEngine   string           `json:"voice_engine,omitempty"`
	Storyboard    StudioStoryboard `json:"storyboard"`
	NarrationHash string           `json:"narration_hash,omitempty"`
	SubtitleHash  string           `json:"subtitle_hash,omitempty"`
	BGMHash       string           `json:"bgm_hash,omitempty"`
	ResultHash    string           `json:"result_hash,omitempty"`
	PosterHash    string           `json:"poster_hash,omitempty"`
	CoverHash     string           `json:"cover_hash,omitempty"`
	// Covers 是封面候选的内容库编号，选中的一张在 CoverHash。
	Covers     []string         `json:"covers,omitempty"`
	CopyHash   string           `json:"copy_hash,omitempty"`
	CopyText   string           `json:"copy_text,omitempty"`
	Versions   []StudioSnapshot `json:"versions,omitempty"`
	Delivery   StudioDelivery   `json:"delivery"`
	Activities []StudioActivity `json:"activities,omitempty"`
	Issues     []StudioIssue    `json:"issues,omitempty"`
	// AssetID 是成片写进画布素材库之后的编号。
	AssetID string `json:"canvas_asset_id,omitempty"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	// AutoPhase 是一键成片的阶段：queued、script、voice、visuals、compose、done 或 failed。
	AutoPhase string `json:"auto_phase,omitempty"`
	// AutoError 是一键成片失败时的原因。成功时清空。
	AutoError string `json:"auto_error,omitempty"`
	// SubtitleEstimated 为真表示字幕按字数估算，不是听写对齐的结果。
	SubtitleEstimated bool `json:"subtitle_estimated,omitempty"`
	// Brief 是选题备注。起草口播时可以参考，不要求照念。
	Brief     string `json:"brief,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// normalizeAspect 把画幅收成 16:9、1:1 或 9:16。认不出的都当成 9:16。
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

// aspectSize 给出画幅对应的像素。16:9 是 1920×1080，1:1 是 1080×1080，其余是 1080×1920。
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

// ListStudioProjects 按更新时间从新到旧列出短片工程。没有时返回空切片。
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

// GetStudioProject 按编号读取一条短片。没有时返回 studio project not found。
func (e *Engine) GetStudioProject(id string) (StudioProject, error) {
	p, err := getDoc[StudioProject](e, colStudioProjects, strings.TrimSpace(id))
	if err != nil {
		return StudioProject{}, fmt.Errorf("studio project not found")
	}
	return p, nil
}

// CreateStudioProject 新建一条草稿。标题空着就写成「短片」，口播待审，闸门停在口播，修订号从 1 开始。
// 没指定预设时按画幅挑一个。没指定身份时用默认口播主持。带了选题就把选题标成制作中并记上工程编号。
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

// UpdateStudioProject 只覆盖传入的非空字段。镜头切片会整表换上，空切片会清空，nil 不动。
// 已锁定的口播一旦改字就退回待审。画面说明变了且这一镜已有图或视频时，合并时会标成过期。
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
	if f := strings.TrimSpace(in.Storyboard.Fit); f == "cover" || f == "contain" {
		cur.Storyboard.Fit = f
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

// PutStudioProject 按现有编号整份写回。编号空着就拒绝。
func (e *Engine) PutStudioProject(p StudioProject) error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("id required")
	}
	return e.putDoc(colStudioProjects, p.ID, p)
}

// DeleteStudioProject 删掉工程，并拆掉所有指着它的会话绑定。编号空着就拒绝。
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

// BindStudioSession 把会话和短片绑在一起。同一会话或同一条短片上原有的绑定会先拆掉，所以两边都只留一对。
// 会话或工程编号空着，或工程不存在，就拒绝。
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

// StudioSessionBind 返回这个会话绑着的短片编号。没绑定时第二个结果是假。
func (e *Engine) StudioSessionBind(sessionID string) (string, bool) {
	m := e.loadStudioBinds()
	id := strings.TrimSpace(m[strings.TrimSpace(sessionID)])
	return id, id != ""
}

// PublishStudioAsset 把成片放进画布素材库的短片文件夹。还没有成片就拒绝。写入失败时工程里的成片编号仍在。
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

// StudioJobShouldSkip 判断延后的开拍不要再拉起助手。
// 已经绑了会话、口播已锁定、已有成片，或闸门已经走过口播，都返回真。空白草稿返回假。
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

// StudioFromTopic 按选题打开短片。选题已经挂着工程就复用，画幅不同时只改画幅。否则用选题标题或角度新建，并把标题、角度和备注写进口播。
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
