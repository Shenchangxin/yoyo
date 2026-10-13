package video

import (
	"fmt"
	"sort"
	"strings"
)

const (
	TopicInbox        = "inbox"
	TopicReady        = "ready"
	TopicScheduled    = "scheduled"
	TopicInProduction = "in_production"
	TopicDone         = "done"
)

// Topic is a content-selection card (not a personal-OS idea).
type Topic struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	SourcePlatform  string         `json:"source_platform,omitempty"`
	SourceTitle     string         `json:"source_title,omitempty"`
	Angle           string         `json:"angle,omitempty"`
	Format          string         `json:"format,omitempty"`
	Note            string         `json:"note,omitempty"`
	Scores          map[string]int `json:"scores,omitempty"`
	Status          string         `json:"status"`
	PersonaPageID   string         `json:"persona_page_id,omitempty"`
	StudioProjectID string         `json:"studio_project_id,omitempty"`
	SourceURL       string         `json:"source_url,omitempty"`
	SourceHash      string         `json:"source_hash,omitempty"`
	Options         []TopicOption  `json:"options,omitempty"`
	CreatedAt       string         `json:"created_at"`
	UpdatedAt       string         `json:"updated_at"`
}

type TopicOption struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Angle   string `json:"angle,omitempty"`
	Summary string `json:"summary,omitempty"`
}

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

func (e *Engine) GetTopic(id string) (Topic, error) {
	t, err := getDoc[Topic](e, colStudioTopics, strings.TrimSpace(id))
	if err != nil {
		return Topic{}, fmt.Errorf("topic not found")
	}
	return t, nil
}

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
		if in.Options == nil {
			in.Options = cur.Options
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

func (e *Engine) DeleteTopic(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	return e.delDoc(colStudioTopics, id)
}

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
