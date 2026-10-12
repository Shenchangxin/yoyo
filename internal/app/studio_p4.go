package app

import (
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/pages"
	"github.com/Shenchangxin/yoyo/internal/video"
)

func (a *App) OpenStudioPublish(id string, analytics bool) (map[string]any, error) {
	eng, err := a.requireVideo()
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	p, err := eng.GetPublish(id)
	if err != nil {
		return nil, err
	}
	if !analytics {
		p, err = eng.ExportPublish(id)
		if err != nil {
			return nil, err
		}
	}
	target := p.CreatorURL
	if analytics {
		target = p.AnalyticsURL
	}
	if strings.TrimSpace(target) == "" {
		return nil, fmt.Errorf("no URL for platform %s", p.Platform)
	}
	opened := false
	if a.Browser != nil {
		_, _ = a.Browser.OpenURLLane(target, "isolated")
		if err := a.BrowserTakeoverAt(""); err == nil {
			opened = true
		}
	}
	if !opened {
		if err := a.OpenURL(target); err != nil {
			return nil, fmt.Errorf("open creator page: %w", err)
		}
	}
	if p.Status == "" || p.Status == video.PublishDraft {
		if marked, err := eng.MarkPublish(p.ID, video.PublishOpened, "", "", 0, 0, 0); err == nil {
			p = marked
		}
	}
	a.journalVideo("studio.publish.open", map[string]any{
		"id": p.ID, "platform": p.Platform, "analytics": analytics, "url": target, "export_path": p.ExportPath,
	})
	return a.enrichPublish(p), nil
}

func (a *App) CreateStudioPersona(name, topicID string) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "创作者人设"
	}
	pg, err := a.PagesSave(pages.Page{Title: name, Content: video.PersonaTemplate(name)}, 0)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"page": pg, "id": pg.ID, "title": pg.Title, "persona_page_id": pg.ID}
	topicID = strings.TrimSpace(topicID)
	if topicID != "" {
		if eng, err := a.requireVideo(); err == nil {
			if t, err := eng.GetTopic(topicID); err == nil {
				t.PersonaPageID = pg.ID
				if saved, err := eng.SaveTopic(t); err == nil {
					out["topic"] = saved
				}
			}
		}
	}
	a.journalVideo("studio.persona.create", map[string]any{"page_id": pg.ID, "topic_id": topicID})
	return out, nil
}

func (a *App) enrichPublish(p video.Publish) map[string]any {
	m := objJSON(p)
	m["media_url"] = a.mediaURL(p.MediaHash)
	return m
}

func strAnyFrom(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}
