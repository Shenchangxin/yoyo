package video

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	PublishDraft  = "draft"
	PublishOpened = "opened"
	PublishPosted = "posted"
	PublishFailed = "failed"
)

type PublishPlatform struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	CreatorURL   string `json:"creator_url"`
	AnalyticsURL string `json:"analytics_url"`
}

// PublishPlatforms are creator-backend handoff URLs. Yoyo never clicks Publish
// and never embeds Playwright; isolated Chrome + operator takeover is the lane.
func PublishPlatforms() []PublishPlatform {
	return []PublishPlatform{
		{ID: "xhs", Label: "小红书", CreatorURL: "https://creator.xiaohongshu.com/publish/publish", AnalyticsURL: "https://creator.xiaohongshu.com/statistics/data-analysis"},
		{ID: "douyin", Label: "抖音", CreatorURL: "https://creator.douyin.com/creator-micro/content/upload", AnalyticsURL: "https://creator.douyin.com/creator-micro/data/following/stats"},
		{ID: "kuaishou", Label: "快手", CreatorURL: "https://cp.kuaishou.com/article/publish/video", AnalyticsURL: "https://cp.kuaishou.com/statistics/analyze"},
		{ID: "channels", Label: "视频号", CreatorURL: "https://channels.weixin.qq.com/platform/post/create", AnalyticsURL: "https://channels.weixin.qq.com/platform/post/list"},
		{ID: "zhihu", Label: "知乎", CreatorURL: "https://www.zhihu.com/videomaker", AnalyticsURL: "https://www.zhihu.com/creator"},
		{ID: "bilibili", Label: "B站", CreatorURL: "https://member.bilibili.com/platform/upload/video/frame", AnalyticsURL: "https://member.bilibili.com/platform/home"},
	}
}

func LookupPublishPlatform(id string) (PublishPlatform, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, p := range PublishPlatforms() {
		if p.ID == id {
			return p, true
		}
	}
	return PublishPlatform{}, false
}

type Publish struct {
	ID           string `json:"id"`
	ProjectID    string `json:"project_id,omitempty"`
	TopicID      string `json:"topic_id,omitempty"`
	Platform     string `json:"platform"`
	Title        string `json:"title"`
	Caption      string `json:"caption,omitempty"`
	Tags         string `json:"tags,omitempty"`
	MediaHash    string `json:"media_hash,omitempty"`
	ExportPath   string `json:"export_path,omitempty"`
	CreatorURL   string `json:"creator_url,omitempty"`
	AnalyticsURL string `json:"analytics_url,omitempty"`
	PostedURL    string `json:"posted_url,omitempty"`
	Status       string `json:"status"`
	Note         string `json:"note,omitempty"`
	Reads        int    `json:"reads,omitempty"`
	Likes        int    `json:"likes,omitempty"`
	Comments     int    `json:"comments,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func normalizePublishStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case PublishOpened, "open":
		return PublishOpened
	case PublishPosted, "done", "published":
		return PublishPosted
	case PublishFailed, "error":
		return PublishFailed
	default:
		return PublishDraft
	}
}

func (e *Engine) ListPublishes(projectID, platform string) []Publish {
	projectID = strings.TrimSpace(projectID)
	platform = strings.ToLower(strings.TrimSpace(platform))
	var out []Publish
	for _, p := range loadCol[Publish](e, colStudioPublishes) {
		if projectID != "" && p.ProjectID != projectID {
			continue
		}
		if platform != "" && p.Platform != platform {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	if out == nil {
		return []Publish{}
	}
	return out
}

func (e *Engine) GetPublish(id string) (Publish, error) {
	p, err := getDoc[Publish](e, colStudioPublishes, strings.TrimSpace(id))
	if err != nil {
		return Publish{}, fmt.Errorf("publish record not found")
	}
	return p, nil
}

func (e *Engine) SavePublish(in Publish) (Publish, error) {
	plat, ok := LookupPublishPlatform(in.Platform)
	if !ok {
		return Publish{}, fmt.Errorf("unknown platform %q — use xhs, douyin, kuaishou, channels, zhihu, or bilibili", in.Platform)
	}
	in.Platform = plat.ID
	in.CreatorURL = plat.CreatorURL
	in.AnalyticsURL = plat.AnalyticsURL
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return Publish{}, fmt.Errorf("title required")
	}
	now := Now()
	if strings.TrimSpace(in.ID) == "" {
		in.ID = NewID()
		in.CreatedAt = now
		if in.ProjectID != "" {
			if p, err := e.GetStudioProject(in.ProjectID); err == nil {
				if in.MediaHash == "" {
					in.MediaHash = p.ResultHash
				}
				if in.Caption == "" {
					in.Caption = strings.TrimSpace(p.Script)
				}
			}
		}
	} else if cur, err := getDoc[Publish](e, colStudioPublishes, in.ID); err == nil {
		if in.CreatedAt == "" {
			in.CreatedAt = cur.CreatedAt
		}
		if in.ExportPath == "" {
			in.ExportPath = cur.ExportPath
		}
		if in.MediaHash == "" {
			in.MediaHash = cur.MediaHash
		}
		if in.PostedURL == "" {
			in.PostedURL = cur.PostedURL
		}
	} else {
		in.CreatedAt = now
	}
	in.Status = normalizePublishStatus(in.Status)
	in.UpdatedAt = now
	if err := e.putDoc(colStudioPublishes, in.ID, in); err != nil {
		return Publish{}, err
	}
	return in, nil
}

func (e *Engine) ExportPublish(id string) (Publish, error) {
	p, err := e.GetPublish(id)
	if err != nil {
		return Publish{}, err
	}
	if p.MediaHash == "" {
		return Publish{}, fmt.Errorf("no assembled video on this draft — assemble first")
	}
	raw, err := e.GetBytes(p.MediaHash)
	if err != nil {
		return Publish{}, err
	}
	dir := filepath.Join(e.Dir, "exports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Publish{}, err
	}
	path := filepath.Join(dir, p.ID+".mp4")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return Publish{}, err
	}
	p.ExportPath = path
	p.UpdatedAt = Now()
	if err := e.putDoc(colStudioPublishes, p.ID, p); err != nil {
		return Publish{}, err
	}
	return p, nil
}

func (e *Engine) MarkPublish(id, status, postedURL, note string, reads, likes, comments int) (Publish, error) {
	p, err := e.GetPublish(id)
	if err != nil {
		return Publish{}, err
	}
	p.Status = normalizePublishStatus(status)
	if strings.TrimSpace(postedURL) != "" {
		p.PostedURL = strings.TrimSpace(postedURL)
	}
	if strings.TrimSpace(note) != "" {
		p.Note = strings.TrimSpace(note)
	}
	if reads > 0 {
		p.Reads = reads
	}
	if likes > 0 {
		p.Likes = likes
	}
	if comments > 0 {
		p.Comments = comments
	}
	p.UpdatedAt = Now()
	if p.Status == PublishPosted && p.TopicID != "" {
		if t, err := e.GetTopic(p.TopicID); err == nil {
			t.Status = TopicDone
			_, _ = e.SaveTopic(t)
		}
	}
	if err := e.putDoc(colStudioPublishes, p.ID, p); err != nil {
		return Publish{}, err
	}
	return p, nil
}
