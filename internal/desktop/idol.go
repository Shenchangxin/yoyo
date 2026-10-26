package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// IdolCall 把 op 和 JSON payload 交给偶像桌面。pin、save_idol、delete_idol 成功后发出 yoyo:idol 且值为真。
func (s *Service) IdolCall(op, payload string) (string, error) {
	raw, err := s.idolForward(op, payload)
	if err == nil && idolDeskEvent(op) {
		s.noteIdol()
	}
	return raw, err
}

// idolForward 在 RPC 非空时调用 idol.call，并把结果重新编成 JSON 字符串。没有 App 时返回 no app。否则走 App.IdolCall。
// op 在 RPC 非空时放进 idol.call 的 op，否则传给 App.IdolCall。
// payload 在 RPC 非空时放进 idol.call 的 payload，否则传给 App.IdolCall。
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

// PlaceIdol 用 pin 固定这个偶像，清掉收起标记，必要时打开 companion_enabled，然后显示宠物窗口。id 为空白时返回 pick an idol。
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

// idolDeskEvent 在 op 是 pin、save_idol 或 delete_idol 时返回真。
func idolDeskEvent(op string) bool {
	switch strings.TrimSpace(op) {
	case "pin", "save_idol", "delete_idol":
		return true
	default:
		return false
	}
}

// noteIdol 在 gui 非空时发出 yoyo:idol，值为真。
func (s *Service) noteIdol() {
	if s == nil || s.gui == nil {
		return
	}
	s.gui.Event.Emit("yoyo:idol", true)
}
