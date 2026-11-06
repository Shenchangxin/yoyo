package app

import (
	"strconv"
	"strings"
	"time"
)

// DeskConfig 是主动式的节奏，由设置里的「主动与通知」写入。
// BriefMorning、BriefEvening 是 HH:MM；off 表示关掉那一档；空表示用默认的 08:30 和 21:30。
// Level 取 less、normal 或 more，决定前端按住哪几类开口，这里原样存。
// QuietStart、QuietEnd 是 HH:MM，可以跨午夜；任一为空或两者相等表示没有安静时段。
// QuietWeekend 为真时，周六周日只推审批和出错。
type DeskConfig struct {
	BriefMorning string `yaml:"brief_morning,omitempty" json:"brief_morning,omitempty"`
	BriefEvening string `yaml:"brief_evening,omitempty" json:"brief_evening,omitempty"`
	Level        string `yaml:"level,omitempty" json:"level,omitempty"`
	QuietStart   string `yaml:"quiet_start,omitempty" json:"quiet_start,omitempty"`
	QuietEnd     string `yaml:"quiet_end,omitempty" json:"quiet_end,omitempty"`
	QuietWeekend bool   `yaml:"quiet_weekend,omitempty" json:"quiet_weekend,omitempty"`
}

// Quiet 在 t 落进安静时段时为真。开始那一分钟算在里面，结束那一分钟不算。
func (d DeskConfig) Quiet(t time.Time) bool {
	start, okStart := clockMinutes(d.QuietStart)
	end, okEnd := clockMinutes(d.QuietEnd)
	if !okStart || !okEnd || start == end {
		return false
	}
	now := t.Hour()*60 + t.Minute()
	if start < end {
		return now >= start && now < end
	}
	return now >= start || now < end
}

// Weekend 在 QuietWeekend 打开且 t 是周六或周日时为真。
func (d DeskConfig) Weekend(t time.Time) bool {
	if !d.QuietWeekend {
		return false
	}
	day := t.Weekday()
	return day == time.Saturday || day == time.Sunday
}

// HoldToast 说这一类系统通知现在该不该压住。安静时段全压；周末只放审批和出错。
// 压住的只是系统通知，记录、桌面伴侣的气泡和今天栏里的「需要你」照常。
func (d DeskConfig) HoldToast(kind string, t time.Time) bool {
	if d.Quiet(t) {
		return true
	}
	return d.Weekend(t) && kind != "approval" && kind != "error"
}

// clockMinutes 把 HH:MM 读成当天的分钟数。读不出时第二个返回值为假。
func clockMinutes(s string) (int, bool) {
	hh, mm, ok := strings.Cut(strings.TrimSpace(s), ":")
	if !ok {
		return 0, false
	}
	h, errH := strconv.Atoi(hh)
	m, errM := strconv.Atoi(mm)
	if errH != nil || errM != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}
