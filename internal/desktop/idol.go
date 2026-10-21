package desktop

import (
	"encoding/json"
	"errors"
)

// IdolCall is the idol desk. op selects the action; payload is a JSON object.
func (s *Service) IdolCall(op, payload string) (string, error) {
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
