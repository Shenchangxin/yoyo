// calendar_guests.go 认出日程草稿里的参会人。CalDAV、Google、Outlook 只认邮箱；飞书还认用户（ou_）、群（oc_）和会议室（omm_）。
// 也放「已经写出去一部分」的错误：日程建好了、参会人没加上时，不能按授权过期再发一遍，不然会多出一个一样的日程。
// 不拥有：各家怎么把参会人写进请求（caldav_people.go、feishu_calendar.go、live.go）。

package connector

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

// calendarGuests 是草稿里参会人的邮箱：To 加 Attendees，去重，保持先后。不带 @ 的（比如 To 里写的时间）不算。
func calendarGuests(d Draft) []string {
	return mailRecipients(d.To, strings.Join(d.Attendees, ","))
}

// splitGuests 把一栏参会人按逗号、分号（含全角）、顿号和换行拆开，去掉空白。「名字 <地址>」整段留着，由调用方解析。
func splitGuests(field string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(field, func(r rune) bool { return strings.ContainsRune(",;，；、\n", r) }) {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// feishuGuest 把一个参会人原文译成飞书「加参会人」接口里的一项，第二个返回值是去重用的键。认不出时 ok 为假。
//   - 邮箱（可以写成「名字 <地址>」）：外部参会人 third_party；
//   - ou_ 开头：飞书用户的 open_id；oc_ 开头：群；omm_ 开头：会议室。
func feishuGuest(raw string) (map[string]any, string, bool) {
	s := strings.TrimSpace(raw)
	switch {
	case strings.Contains(s, "@"):
		addr := s
		if a, err := mail.ParseAddress(s); err == nil {
			addr = a.Address
		}
		if !strings.Contains(addr, "@") || strings.ContainsAny(addr, " <>") {
			return nil, "", false
		}
		return map[string]any{"type": "third_party", "third_party_email": addr}, "mail:" + strings.ToLower(addr), true
	case strings.HasPrefix(s, "ou_"):
		return map[string]any{"type": "user", "user_id": s}, "user:" + s, true
	case strings.HasPrefix(s, "oc_"):
		return map[string]any{"type": "chat", "chat_id": s}, "chat:" + s, true
	case strings.HasPrefix(s, "omm_"):
		return map[string]any{"type": "resource", "room_id": s}, "room:" + s, true
	}
	return nil, "", false
}

// feishuGuests 认出草稿里的参会人，去重，保持先后。Attendees 里每一项都要认得出：认不出就报错，一个请求都不发，
// 免得日程建好了参会人却只加上一半。To 那一栏只取认得出的（老写法里 To 可能放的是时间）。
func feishuGuests(d Draft) ([]map[string]any, error) {
	seen := map[string]bool{}
	var out []map[string]any
	add := func(raw string, strict bool) error {
		g, key, ok := feishuGuest(raw)
		if !ok {
			if strict {
				return fmt.Errorf("connector: 认不出参会人 %q。飞书日历的参会人要写邮箱、用户的 open_id（ou_ 开头）、群（oc_ 开头）或会议室（omm_ 开头）", raw)
			}
			return nil
		}
		if !seen[key] {
			seen[key] = true
			out = append(out, g)
		}
		return nil
	}
	for _, p := range splitGuests(d.To) {
		_ = add(p, false)
	}
	for _, field := range d.Attendees {
		for _, p := range splitGuests(field) {
			if err := add(p, true); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// writtenError 是「已经写出去一部分」的失败：日程建好或改好了，后面一步（加参会人）没成功。
// liveSend 见到它不按授权过期重发，免得再建一个一样的日程。
type writtenError struct{ error }

func (e writtenError) Unwrap() error { return e.error }

// alreadyWritten 在 err 是（或包着）writtenError 时为真。
func alreadyWritten(err error) bool {
	var w writtenError
	return errors.As(err, &w)
}
