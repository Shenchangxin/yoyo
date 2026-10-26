// 选题卡片的状态和存取。选题是一条内容，不是个人操作系统里的想法。
// 不开拍，也不起草口播。

package video

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// TopicInbox 是刚收进来、还没排期的选题。认不出的状态也会收成这个。
	TopicInbox = "inbox"
	// TopicReady 是可以拿去开拍的选题。
	TopicReady = "ready"
	// TopicScheduled 是已排期的选题。queued 也会收成这个。
	TopicScheduled = "scheduled"
	// TopicInProduction 是已经挂上短片工程的选题。doing 和 in-progress 也会收成这个。
	TopicInProduction = "in_production"
	// TopicDone 是这条选题的短片已经做完。published 和 complete 也会收成这个。
	TopicDone = "done"
)

// Topic 是一张选题卡片，不是个人操作系统里的想法。
type Topic struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	SourcePlatform string `json:"source_platform,omitempty"`
	SourceTitle    string `json:"source_title,omitempty"`
	Angle          string `json:"angle,omitempty"`
	Format         string `json:"format,omitempty"`
	Note           string `json:"note,omitempty"`
	// Scores 是调用方自己起名的分数。本文件不解释每个键。
	Scores        map[string]int `json:"scores,omitempty"`
	Status        string         `json:"status"`
	PersonaPageID string         `json:"persona_page_id,omitempty"`
	// StudioProjectID 是这条选题已经开拍时指向的短片工程。
	StudioProjectID string `json:"studio_project_id,omitempty"`
	SourceURL       string `json:"source_url,omitempty"`
	SourceHash      string `json:"source_hash,omitempty"`
	// Options 是展开后待选的几个角度。选中一个才会写回标题和角度。
	Options   []TopicOption `json:"options,omitempty"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// TopicOption 是选题展开后的一个角度。
type TopicOption struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Angle   string `json:"angle,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// normalizeTopicStatus 把状态收成收件箱、就绪、已排期、制作中或完成。空的和认不出的都是收件箱。
func normalizeTopicStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case TopicReady:
		return TopicReady
	case TopicScheduled, "queued":
		return TopicScheduled
	case TopicInProduction, "doing", "in-progress":
		return TopicInProduction
	case TopicDone, "published", "complete":
		return TopicDone
	default:
		return TopicInbox
	}
}

// ListTopics 按更新时间从新到旧列出选题。status 空着就全部返回。没有时返回空切片。
func (e *Engine) ListTopics(status string) []Topic {
	want := strings.TrimSpace(strings.ToLower(status))
	if want != "" {
		want = normalizeTopicStatus(want)
	}
	var out []Topic
	for _, t := range loadCol[Topic](e, colStudioTopics) {
		if want != "" && t.Status != want {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	if out == nil {
		return []Topic{}
	}
	return out
}

// GetTopic 按编号读取一张选题。没有时返回 topic not found。
func (e *Engine) GetTopic(id string) (Topic, error) {
	t, err := getDoc[Topic](e, colStudioTopics, strings.TrimSpace(id))
	if err != nil {
		return Topic{}, fmt.Errorf("topic not found")
	}
	return t, nil
}

// SaveTopic 新建或更新选题。标题空着就拒绝。
// 更新时，传入的空字段沿用库里的值，空状态也沿用。编号对不上就按新卡片写，并补上创建时间。
func (e *Engine) SaveTopic(in Topic) (Topic, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return Topic{}, fmt.Errorf("title required")
	}
	now := Now()
	if strings.TrimSpace(in.ID) == "" {
		in.ID = NewID()
		in.CreatedAt = now
	} else if cur, err := getDoc[Topic](e, colStudioTopics, in.ID); err == nil {
		if in.CreatedAt == "" {
			in.CreatedAt = cur.CreatedAt
		}
		if in.StudioProjectID == "" {
			in.StudioProjectID = cur.StudioProjectID
		}
		if in.PersonaPageID == "" {
			in.PersonaPageID = cur.PersonaPageID
		}
		if in.SourceURL == "" {
			in.SourceURL = cur.SourceURL
		}
		if in.SourceHash == "" {
			in.SourceHash = cur.SourceHash
		}
		if in.SourcePlatform == "" {
			in.SourcePlatform = cur.SourcePlatform
		}
		if in.SourceTitle == "" {
			in.SourceTitle = cur.SourceTitle
		}
		if in.Angle == "" {
			in.Angle = cur.Angle
		}
		if in.Format == "" {
			in.Format = cur.Format
		}
		if in.Note == "" {
			in.Note = cur.Note
		}
		if in.Scores == nil {
			in.Scores = cur.Scores
		}
		if in.Options == nil {
			in.Options = cur.Options
		}
		if strings.TrimSpace(in.Status) == "" {
			in.Status = cur.Status
		}
	} else {
		in.CreatedAt = now
	}
	in.Status = normalizeTopicStatus(in.Status)
	in.UpdatedAt = now
	if err := e.putDoc(colStudioTopics, in.ID, in); err != nil {
		return Topic{}, err
	}
	return in, nil
}

// DeleteTopic 按编号删掉选题。编号空着就拒绝。不改已经挂上的短片工程。
func (e *Engine) DeleteTopic(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	return e.delDoc(colStudioTopics, id)
}

// TopicFromTrend 用热榜标题收一张收件箱选题。格式记成 short。热度非空时写进备注。
func (e *Engine) TopicFromTrend(platform, title, hot string) (Topic, error) {
	title = strings.TrimSpace(title)
	note := ""
	if strings.TrimSpace(hot) != "" {
		note = "热度 " + strings.TrimSpace(hot)
	}
	return e.SaveTopic(Topic{
		Title:          title,
		SourcePlatform: strings.TrimSpace(platform),
		SourceTitle:    title,
		Format:         "short",
		Note:           note,
		Status:         TopicInbox,
	})
}
