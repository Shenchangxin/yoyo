package im

import (
	"strings"
	"time"
)

type rateWindow struct {
	start time.Time
	n     int
}

// Admit 决定这条消息要不要进入对话。命中时返回去掉唤醒前缀之后的正文。
func (rt *Runtime) Admit(in Inbound) (Inbound, bool) {
	rt.mu.Lock()
	s := rt.settings
	rt.mu.Unlock()
	if in.Text == "" && !in.Mentioned {
		return in, false
	}
	if s.IgnoreSelf && in.SelfID != "" && in.SenderID == in.SelfID {
		return in, false
	}
	if len(s.AllowUsers) > 0 && !contains(s.AllowUsers, in.SenderID) {
		return in, false
	}
	if in.Kind == "group" && len(s.AllowGroups) > 0 && !contains(s.AllowGroups, in.ChatID) {
		return in, false
	}
	text := strings.TrimSpace(in.Text)
	woke := false
	for _, p := range s.WakePrefixes {
		if p != "" && strings.HasPrefix(text, p) {
			text = strings.TrimSpace(strings.TrimPrefix(text, p))
			woke = true
			break
		}
	}
	direct := in.Kind != "group"
	if direct {
		if !s.FriendNeedsWake || woke || len(s.WakePrefixes) == 0 {
			woke = true
		}
	} else {
		mentioned := in.Mentioned
		if in.AtAll && s.IgnoreAtAll {
			mentioned = false
		}
		if mentioned {
			woke = true
		}
	}
	if !woke {
		return in, false
	}
	if strings.TrimSpace(text) == "" {
		return in, false
	}
	if !rt.allowRate(s, in) {
		return in, false
	}
	in.Text = text
	in.Admin = contains(s.Admins, in.SenderID)
	return in, true
}

func (rt *Runtime) allowRate(s Settings, in Inbound) bool {
	if s.RatePerMinute <= 0 {
		return true
	}
	key := in.AccountID + "\n" + in.SenderID
	now := time.Now()
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.rates == nil {
		rt.rates = map[string]rateWindow{}
	}
	w := rt.rates[key]
	if w.start.IsZero() || now.Sub(w.start) >= time.Minute {
		w = rateWindow{start: now, n: 0}
	}
	if w.n >= s.RatePerMinute {
		rt.rates[key] = w
		return false
	}
	w.n++
	rt.rates[key] = w
	return true
}

func contains(list []string, v string) bool {
	v = strings.TrimSpace(v)
	for _, item := range list {
		if strings.TrimSpace(item) == v && v != "" {
			return true
		}
	}
	return false
}
