package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// IdolCall is the idol desk. op selects the action; payload is a JSON object.
func (s *Service) IdolCall(op, payload string) (string, error) {
	raw, err := s.idolForward(op, payload)
	if err == nil && idolDeskEvent(op) {
		s.noteIdol()
	}
	return raw, err
}

func (s *Service) idolForward(op, payload string) (string, error) {
	if s.RPC != nil {
		v, err := s.call("idol.call", map[string]any{"op": op, "payload": payload})
		if err != nil {
			return "", err
		}
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	if s.App == nil {
		return "", errors.New("no app")
	}
	return s.App.IdolCall(op, payload)
}

// PlaceIdol pins this idol and shows her portrait on the desktop.
func (s *Service) PlaceIdol(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("pick an idol")
	}
	if _, err := s.IdolCall("pin", fmt.Sprintf(`{"id":%q}`, id)); err != nil {
		return err
	}
	s.companionDismissed.Store(false)
	if !s.GetConfig().CompanionEnabled {
		if err := s.SetConfig(map[string]any{"companion_enabled": true}); err != nil {
			return err
		}
	}
	return s.ShowCompanion()
}

func idolDeskEvent(op string) bool {
	switch strings.TrimSpace(op) {
	case "pin", "save_idol", "delete_idol":
		return true
	default:
		return false
	}
}

func (s *Service) noteIdol() {
	if s == nil || s.gui == nil {
		return
	}
	s.gui.Event.Emit("yoyo:idol", true)
}
