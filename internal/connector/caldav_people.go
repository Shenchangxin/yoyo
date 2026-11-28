// caldav_people.go 管 CalDAV 日程里的组织者和参会人（RFC 5545 / RFC 6638）。
// 写：草稿的 To 和 Attendees 写成 ATTENDEE；知道账号本人的日历地址时写 ORGANIZER。iCloud、Fastmail、Nextcloud 这类
// 支持日程调度的服务器看到组织者是本人、又有参会人，会替人发邀请（和 Google、Outlook 写日程时一样）。
// 改：CalDAV 改日程是整份替换。草稿没写参会人时，把原日程的组织者和参会人原样写回，不然一改主题参会人就没了；
// 写了参会人时，原来就在的人留着原来那一行（他回过的「接受 / 拒绝」不丢），其余换成草稿里的。
// 不拥有：日程的读写请求（caldav.go）、查找账号本人的日历地址（caldav_discover.go）。

package connector

import (
	"net/http"
	"net/mail"
	"net/url"
	"path"
	"strings"
)

// icsPeople 是写进一个日程的组织者和参会人。
type icsPeople struct {
	// Organizer 是账号本人的日历地址（不带 mailto:），有参会人、原日程又没有组织者时写成 ORGANIZER。
	Organizer string
	// KeepOrganizer、Keep 是原日程里的 ORGANIZER 行和 ATTENDEE 行（折行已经接好）。
	KeepOrganizer string
	Keep          []string
}

// icsMailto 取出一行 ORGANIZER / ATTENDEE 里的邮箱（最后一个 mailto: 后面那段），参数里带引号的冒号不影响。没有时为空。
func icsMailto(line string) string {
	i := strings.LastIndex(strings.ToLower(line), "mailto:")
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(line[i+len("mailto:"):])
}

// icsEmails 是几行 ATTENDEE 里的邮箱，读日历时给人看。
func icsEmails(lines []string) []string {
	var out []string
	for _, l := range lines {
		if addr := icsMailto(l); addr != "" {
			out = append(out, addr)
		}
	}
	return out
}

// icsGuestLines 是 VEVENT 里的 ORGANIZER 和 ATTENDEE 几行，规则见文件头。没有参会人时不写 ORGANIZER。
// 原日程有组织者时用原来的（可能是别人发来的邀请，不能改成自己）；没有时用账号本人。组织者自己不再列成参会人。
func icsGuestLines(d Draft, who icsPeople) []string {
	guests := calendarGuests(d)
	if len(guests) == 0 {
		if len(who.Keep) == 0 {
			return nil
		}
		out := []string{}
		if who.KeepOrganizer != "" {
			out = append(out, who.KeepOrganizer)
		}
		return append(out, who.Keep...)
	}
	orgLine := who.KeepOrganizer
	if orgLine == "" {
		if org := strings.TrimSpace(who.Organizer); org != "" {
			orgLine = "ORGANIZER:mailto:" + org
		}
	}
	org := icsMailto(orgLine)
	kept := map[string]string{}
	for _, l := range who.Keep {
		if addr := strings.ToLower(icsMailto(l)); addr != "" {
			kept[addr] = l
		}
	}
	var att []string
	for _, addr := range guests {
		if org != "" && strings.EqualFold(addr, org) {
			continue
		}
		if l, ok := kept[strings.ToLower(addr)]; ok {
			att = append(att, l)
			continue
		}
		att = append(att, "ATTENDEE;RSVP=TRUE:mailto:"+addr)
	}
	if len(att) == 0 {
		return nil
	}
	if orgLine != "" {
		return append([]string{orgLine}, att...)
	}
	return att
}

// caldavOriginal 读出要整份替换的那个资源原来的 UID、组织者和参会人。UID 必须不变，不然服务器会当成另一个日程拒绝或重复；
// 组织者和参会人在草稿没写参会人时要原样写回。读不到时 UID 退回用资源文件名去掉 .ics，参会人为空。
func caldavOriginal(client *http.Client, target, urlUser, token string) (string, icsPeople) {
	fallback := ""
	if u, err := url.Parse(target); err == nil {
		fallback = strings.TrimSuffix(path.Base(u.Path), ".ics")
	}
	code, raw, err := caldavDo(client, http.MethodGet, target, urlUser, token, "", nil)
	if err != nil || code >= 300 {
		return fallback, icsPeople{}
	}
	for _, ev := range icsEvents(string(raw)) {
		if ev.UID != "" {
			return ev.UID, icsPeople{KeepOrganizer: ev.Organizer, Keep: ev.Attendees}
		}
	}
	return fallback, icsPeople{}
}

// caldavOrganizer 是写日程时的组织者：查找日历时服务器给的 calendar-user-address-set 里的 mailto 地址；
// 服务器没给时用像邮箱的登录名（iCloud 的 Apple ID、Fastmail 的账号都是邮箱）。都没有时为空，不写 ORGANIZER。
func caldavOrganizer(endpoint, token string) string {
	if self := caldavSelf(endpoint, token); self != "" {
		return self
	}
	_, urlUser, err := caldavCollection(endpoint)
	if err != nil {
		return ""
	}
	user, _ := caldavCreds(token, urlUser)
	if a, err := mail.ParseAddress(user); err == nil {
		return a.Address
	}
	return ""
}
