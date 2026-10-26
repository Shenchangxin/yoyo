package desktop

import (
	"encoding/base64"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/pages"
	"github.com/Shenchangxin/yoyo/internal/profile"
	"github.com/Shenchangxin/yoyo/internal/schedule"
)

// SetPaused 在 RPC 非空时调用 app.pause.set，否则 App.SetPaused。暂停时关掉宠物通话。无论成败都 refreshChrome。
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
	s.refreshChrome()
	return err
}

// IsPaused 在 RPC 非空时调用 health 并读取 paused。否则走 App.IsPaused。
func (s *Service) IsPaused() bool {
	if s.RPC != nil {
		v, _ := s.call("health", nil)
		m, _ := decode[map[string]any](v, nil)
		b, _ := m["paused"].(bool)
		return b
	}
	return s.App.IsPaused()
}

// PagesList 在 RPC 非空时调用 pages.list，否则走 App.PagesList。
func (s *Service) PagesList(q, spaceID string) []pages.Meta {
	if s.RPC != nil {
		v, _ := s.call("pages.list", map[string]any{"q": q, "space_id": spaceID})
		out, _ := decode[[]pages.Meta](v, nil)
		return out
	}
	return s.App.PagesList(q, spaceID)
}

// PagesGet 在 RPC 非空时调用 pages.get，否则走 App.PagesGet。
func (s *Service) PagesGet(spaceID, id string) (pages.Page, error) {
	if s.RPC != nil {
		v, err := s.call("pages.get", map[string]any{"id": id, "space_id": spaceID})
		return decode[pages.Page](v, err)
	}
	return s.App.PagesGet(spaceID, id)
}

// PagesSave 在 RPC 非空时调用 pages.save，并带上 expected_revision。否则走 App.PagesSave。
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

// PagesDelete 在 RPC 非空时调用 pages.delete，否则走 App.PagesDelete。
func (s *Service) PagesDelete(spaceID, id string) error {
	if s.RPC != nil {
		_, err := s.call("pages.delete", map[string]any{"id": id, "space_id": spaceID})
		return err
	}
	return s.App.PagesDelete(spaceID, id)
}

// PagesReviews 在 RPC 非空时调用 pages.reviews，否则走 App.PagesReviews。
func (s *Service) PagesReviews() []pages.Review {
	if s.RPC != nil {
		v, _ := s.call("pages.reviews", nil)
		out, _ := decode[[]pages.Review](v, nil)
		return out
	}
	return s.App.PagesReviews()
}

// PagesPropose 在 RPC 非空时调用 pages.propose，否则走 App.PagesPropose。
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

// PagesDecide 在 RPC 非空时调用 pages.review.decide。同进程把 PagesDecide 的页面和审阅包成 page 与 review。
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

// ProfilesList 在 RPC 非空时调用 profiles.list，否则走 App.ProfilesList。
func (s *Service) ProfilesList() []profile.Profile {
	if s.RPC != nil {
		v, _ := s.call("profiles.list", nil)
		out, _ := decode[[]profile.Profile](v, nil)
		return out
	}
	return s.App.ProfilesList()
}

// ProfilesSave 在 RPC 非空时调用 profiles.save，否则走 App.ProfilesSave。
func (s *Service) ProfilesSave(p profile.Profile) (profile.Profile, error) {
	if s.RPC != nil {
		v, err := s.call("profiles.save", p)
		return decode[profile.Profile](v, err)
	}
	return s.App.ProfilesSave(p)
}

// SetSessionProfile 在 RPC 非空时调用 thread.profile.set，否则走 App.SetSessionProfile。
func (s *Service) SetSessionProfile(id, profileID string) (any, error) {
	if s.RPC != nil {
		return s.call("thread.profile.set", map[string]any{"session": id, "profile_id": profileID})
	}
	return s.App.SetSessionProfile(id, profileID)
}

// SetSessionPage 在 RPC 非空时调用 thread.page.set，否则走 App.SetSessionPage。
func (s *Service) SetSessionPage(id, pageID string) (any, error) {
	if s.RPC != nil {
		return s.call("thread.page.set", map[string]any{"session": id, "page_id": pageID})
	}
	return s.App.SetSessionPage(id, pageID)
}

// ScheduleRetry 在 RPC 非空时调用 schedule.retry，否则走 App.ScheduleRetry。
func (s *Service) ScheduleRetry(id string) (schedule.Job, error) {
	if s.RPC != nil {
		v, err := s.call("schedule.retry", map[string]any{"id": id})
		return decode[schedule.Job](v, err)
	}
	return s.App.ScheduleRetry(id)
}

// ConnectorMention 在 RPC 非空时调用 connectors.mention，否则走 App.ConnectorMention。
func (s *Service) ConnectorMention(account, text string) error {
	if s.RPC != nil {
		_, err := s.call("connectors.mention", map[string]any{"account": account, "text": text})
		return err
	}
	return s.App.ConnectorMention(account, text)
}

// VoiceTranscribe 在 RPC 非空时调用 voice.transcribe，并返回 text。同进程先用 decodeSkinB64Local 解开音频，再 App.VoiceTranscribe。
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

// VoiceSpeak 在 RPC 非空时调用 voice.speak。同进程把音频编成标准 Base64，并带上 content_type。
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

// VoiceReceipt 在 RPC 非空时调用 voice.receipt，忽略错误。否则走 App.VoiceReceipt。
func (s *Service) VoiceReceipt(sessionID string, durationSec int, transcript string) {
	if s.RPC != nil {
		_, _ = s.call("voice.receipt", map[string]any{"session": sessionID, "duration_sec": durationSec, "transcript": transcript})
		return
	}
	s.App.VoiceReceipt(sessionID, durationSec, transcript)
}

// SetCompanionCall 为真时 beginCompanionCall，为假时 endCompanionCall，并允许改尺寸。
func (s *Service) SetCompanionCall(on bool) {
	if on {
		s.beginCompanionCall()
		return
	}
	s.endCompanionCall(true)
}

// decodeSkinB64Local 去掉空白。空字符串返回 nil 和 nil 错误。逗号前带 base64 时去掉 data URL 头，再用标准 Base64 解码。
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

// encodeB64 用标准 Base64 编码，不换行。
func encodeB64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
