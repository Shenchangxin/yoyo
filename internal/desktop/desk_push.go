package desktop

import (
	"fmt"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/poi"
)

// DeskPush 把简报或兴趣变化写进主会话。
func (s *Service) DeskPush(note app.DeskNote) (bool, error) {
	if s.RPC != nil {
		v, err := s.call("desk.push", map[string]any{
			"sessionId": note.SessionID,
			"kind":      note.Kind,
			"slot":      note.Slot,
			"day":       note.Day,
			"text":      note.Text,
			"chip":      note.Chip,
			"brief":     note.Brief,
		})
		if err != nil {
			return false, err
		}
		body, _ := v.(map[string]any)
		ok, _ := body["ok"].(bool)
		return ok, nil
	}
	if s.App == nil {
		return false, fmt.Errorf("app is not running")
	}
	return s.App.DeskPush(note), nil
}

// PoiRefresh 刷新兴趣库并返回这一版。
func (s *Service) PoiRefresh() (poi.Snapshot, error) {
	if s.RPC != nil {
		return decode[poi.Snapshot](s.call("poi.refresh", nil))
	}
	if s.App == nil {
		return poi.Snapshot{}, fmt.Errorf("app is not running")
	}
	return s.App.PoiRefresh(), nil
}

// PoiAck 记下这一版兴趣库已经推进过。
func (s *Service) PoiAck(sig string) (bool, error) {
	if s.RPC != nil {
		v, err := s.call("poi.ack", map[string]any{"sig": sig})
		if err != nil {
			return false, err
		}
		body, _ := v.(map[string]any)
		ok, _ := body["ok"].(bool)
		return ok, nil
	}
	if s.App == nil {
		return false, fmt.Errorf("app is not running")
	}
	return s.App.PoiAck(sig), nil
}
