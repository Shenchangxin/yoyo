// publish.go 保存短片发布草稿，并给出各平台创作者后台的地址。

package video

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 发布记录使用这四种状态文本。
const (
	// PublishDraft 是草稿状态 draft。
	PublishDraft  = "draft"
	// PublishOpened 是已打开后台的状态 opened。
	PublishOpened = "opened"
	// PublishPosted 是已发布状态 posted。
	PublishPosted = "posted"
	// PublishFailed 是失败状态 failed。
	PublishFailed = "failed"
)

// PublishPlatform 是一个平台的标识、名称、创作地址和数据地址。
type PublishPlatform struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	CreatorURL   string `json:"creator_url"`
	AnalyticsURL string `json:"analytics_url"`
}

// PublishPlatforms 返回创作者后台的交接地址。Yoyo 不会替用户点击发布，也不内嵌 Playwright。通道是隔离的 Chrome，由操作者自己接手。
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

// LookupPublishPlatform 按小写标识查找平台。没有时返回假。
// id 去掉首尾空白并转成小写后，与 PublishPlatforms 各项的 ID 比较。
func LookupPublishPlatform(id string) (PublishPlatform, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, p := range PublishPlatforms() {
		if p.ID == id {
			return p, true
		}
	}
	return PublishPlatform{}, false
}

// Publish 是一条短片发布记录，含文案、媒体哈希、导出路径和阅读、点赞、评论数。
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

// normalizePublishStatus 把 open 收成 opened，把 done 和 published 收成 posted，把 error 收成 failed，其余收成 draft。
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

// ListPublishes 按工程和平台过滤发布记录，按更新时间从新到旧。没有记录时返回空切片。
// projectID 去掉首尾空白，非空时只保留 ProjectID 相等的记录。
// platform 去掉首尾空白并转成小写，非空时只保留 Platform 相等的记录。
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

// GetPublish 按标识读取发布记录。没有时返回 publish record not found。
// id 去掉首尾空白后传给 getDoc。
func (e *Engine) GetPublish(id string) (Publish, error) {
	p, err := getDoc[Publish](e, colStudioPublishes, strings.TrimSpace(id))
	if err != nil {
		return Publish{}, fmt.Errorf("publish record not found")
	}
	return p, nil
}

// SavePublish 只接受 xhs、douyin、kuaishou、channels、zhihu、bilibili。标题不能为空。新建时若挂着短片工程，空的媒体哈希和文案从工程补上。更新时保留原来的创建时间、导出路径、媒体哈希和已发布地址。
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

// ExportPublish 把成片写到视频目录 exports 下的 <id>.mp4，权限 0600。没有媒体哈希时返回 no assembled video on this draft — assemble first。
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

// MarkPublish 更新状态。非空的发布地址和备注会写上。阅读、点赞、评论只在大于 0 时覆盖。状态成为 posted 且挂着选题时，把选题标成完成。
// id 传给 GetPublish。
// status 传给 normalizePublishStatus。
// postedURL 去掉首尾空白后非空时写入 PostedURL。
// note 去掉首尾空白后非空时写入 Note。
// reads 大于 0 时写入 Reads。
// likes 大于 0 时写入 Likes。
// comments 大于 0 时写入 Comments。
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
