package desktop

import (
	"encoding/base64"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/pages"
	"github.com/Shenchangxin/yoyo/internal/profile"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

func (s *Service) SetPaused(paused bool) error {
	var err error
	if s.RPC != nil {
		_, err = s.call("app.pause.set", map[string]any{"paused": paused})
	} else {
		err = s.App.SetPaused(paused)
	}
	if paused {
		s.SetCompanionCall(false)
	}
	s.RebuildMenus()
	return err
}

func (s *Service) IsPaused() bool {
	if s.RPC != nil {
		v, _ := s.call("health", nil)
		m, _ := decode[map[string]any](v, nil)
		b, _ := m["paused"].(bool)
		return b
	}
	return s.App.IsPaused()
}

func (s *Service) PagesList(q, spaceID string) []pages.Meta {
	if s.RPC != nil {
		v, _ := s.call("pages.list", map[string]any{"q": q, "space_id": spaceID})
		out, _ := decode[[]pages.Meta](v, nil)
		return out
	}
	return s.App.PagesList(q, spaceID)
}

func (s *Service) PagesGet(spaceID, id string) (pages.Page, error) {
	if s.RPC != nil {
		v, err := s.call("pages.get", map[string]any{"id": id, "space_id": spaceID})
		return decode[pages.Page](v, err)
	}
	return s.App.PagesGet(spaceID, id)
}

func (s *Service) PagesSave(p pages.Page, expected int) (pages.Page, error) {
	if s.RPC != nil {
		body := map[string]any{
			"id": p.ID, "space_id": p.SpaceID, "parent_id": p.ParentID,
			"title": p.Title, "content": p.Content, "expected_revision": expected,
		}
		v, err := s.call("pages.save", body)
		return decode[pages.Page](v, err)
	}
	return s.App.PagesSave(p, expected)
}

func (s *Service) PagesReviews() []pages.Review {
	if s.RPC != nil {
		v, _ := s.call("pages.reviews", nil)
		out, _ := decode[[]pages.Review](v, nil)
		return out
	}
	return s.App.PagesReviews()
}

func (s *Service) PagesPropose(sessionID, title, content, pageID, spaceID, parentID string, expected int) (pages.Review, error) {
	if s.RPC != nil {
		v, err := s.call("pages.propose", map[string]any{
			"session": sessionID, "title": title, "content": content, "id": pageID,
			"space_id": spaceID, "parent_id": parentID, "expected_revision": expected,
		})
		return decode[pages.Review](v, err)
	}
	return s.App.PagesPropose(sessionID, title, content, pageID, spaceID, parentID, expected)
}

func (s *Service) PagesDecide(id, hash string, approve bool) (map[string]any, error) {
	if s.RPC != nil {
		v, err := s.call("pages.review.decide", map[string]any{"id": id, "hash": hash, "approve": approve})
		return decode[map[string]any](v, err)
	}
	pg, rev, err := s.App.PagesDecide(id, hash, approve)
	if err != nil {
		return nil, err
	}
	return map[string]any{"page": pg, "review": rev}, nil
}

func (s *Service) ProfilesList() []profile.Profile {
	if s.RPC != nil {
		v, _ := s.call("profiles.list", nil)
		out, _ := decode[[]profile.Profile](v, nil)
		return out
	}
	return s.App.ProfilesList()
}

func (s *Service) ProfilesSave(p profile.Profile) (profile.Profile, error) {
	if s.RPC != nil {
		v, err := s.call("profiles.save", p)
		return decode[profile.Profile](v, err)
	}
	return s.App.ProfilesSave(p)
}

func (s *Service) SetSessionProfile(id, profileID string) (any, error) {
	if s.RPC != nil {
		return s.call("thread.profile.set", map[string]any{"session": id, "profile_id": profileID})
	}
	return s.App.SetSessionProfile(id, profileID)
}

func (s *Service) SetSessionPage(id, pageID string) (any, error) {
	if s.RPC != nil {
		return s.call("thread.page.set", map[string]any{"session": id, "page_id": pageID})
	}
	return s.App.SetSessionPage(id, pageID)
}

func (s *Service) ScheduleRetry(id string) (schedule.Job, error) {
	if s.RPC != nil {
		v, err := s.call("schedule.retry", map[string]any{"id": id})
		return decode[schedule.Job](v, err)
	}
	return s.App.ScheduleRetry(id)
}

func (s *Service) ConnectorMention(account, text string) error {
	if s.RPC != nil {
		_, err := s.call("connectors.mention", map[string]any{"account": account, "text": text})
		return err
	}
	return s.App.ConnectorMention(account, text)
}

func (s *Service) VoiceTranscribe(filename, audioB64 string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("voice.transcribe", map[string]any{"filename": filename, "audio": audioB64})
		m, e := decode[map[string]any](v, err)
		if e != nil {
			return "", e
		}
		text, _ := m["text"].(string)
		return text, nil
	}
	raw, err := decodeSkinB64Local(audioB64)
	if err != nil {
		return "", err
	}
	return s.App.VoiceTranscribe(filename, raw)
}

func (s *Service) VoiceSpeak(text string) (map[string]any, error) {
	if s.RPC != nil {
		v, err := s.call("voice.speak", map[string]any{"text": text})
		return decode[map[string]any](v, err)
	}
	b, ctype, err := s.App.VoiceSpeak(text)
	if err != nil {
		return nil, err
	}
	return map[string]any{"audio": encodeB64(b), "content_type": ctype}, nil
}

func (s *Service) VoiceReceipt(sessionID string, durationSec int, transcript string) {
	if s.RPC != nil {
		_, _ = s.call("voice.receipt", map[string]any{"session": sessionID, "duration_sec": durationSec, "transcript": transcript})
		return
	}
	s.App.VoiceReceipt(sessionID, durationSec, transcript)
}

func (s *Service) SetCompanionCall(on bool) {
	s.companionCall.Store(on)
	companionInteractive.Store(on)
	if s.gui != nil {
		s.gui.Event.Emit("yoyo:call", on)
	}
	win := s.companion
	if win == nil {
		return
	}
	if on {
		win.SetSize(companionCallW, companionCallH)
		setCompanionPassthrough(win, false)
		applyCompanionShape(win, true)
		return
	}
	win.SetSize(companionW, companionH)
	applyCompanionShape(win, s.companionCaption.Load())
}

func decodeSkinB64Local(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if i := strings.Index(s, ","); i >= 0 && strings.Contains(s[:i], "base64") {
		s = s[i+1:]
	}
	return base64.StdEncoding.DecodeString(s)
}

func encodeB64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
