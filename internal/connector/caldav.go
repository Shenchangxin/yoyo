// caldav.go 用 CalDAV（RFC 4791）读写一个日历集合：iCloud、Fastmail、Nextcloud、群晖、Radicale 等都走这一套。
// 账号上的 Endpoint 是日历集合的完整 https 地址（以 / 结尾，没写会补上）；令牌是「用户名:密码」，
// iCloud、Fastmail 这类开了两步验证的要填应用专用密码。用户名也可以写在地址里（https://user@host/...），令牌只填密码。
//
// 读：REPORT calendar-query，按时间段（默认从现在起两周，查询里有两个 RFC3339 时间时用它们）取事件，
// 每个日程资源取第一个 VEVENT 的 UID、SUMMARY、DTSTART、LOCATION，加上资源地址（作为 id / event_id）和 ETag（作为 target_version）。
// 写：新建是 PUT <集合>/<随机 UID>.ics 带 If-None-Match: *（不会盖掉已有的）；
// 修改是整份替换，PUT 原地址带 If-Match: <ETag>，日程在别处被改过时服务器回 412，不会盖掉别人的改动；
// 删除是 DELETE 带 If-Match。资源地址只认这个日历所在的服务器，不会把账号密码发到别的主机。
// 不拥有：重复日程（和 Google / Outlook 一样拒绝，留给日历应用）、邀请回执。

package connector

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	// 嵌入时区库（约 450KB）：用户的 Windows 机器上一般没有 Go 的 zoneinfo，不嵌的话 time.LoadLocation("Asia/Shanghai")
	// 会失败，带 TZID 的日程就会被当成本机时区解释，换台机器时间就错。
	_ "time/tzdata"
	"unicode/utf8"
)

// caldavWindow 是查询里没写时间段时往后读多久。
const caldavWindow = 14 * 24 * time.Hour

// caldavLatest 是一次最多给出几个日程，和 Google / Outlook 读日历一致。
const caldavLatest = 20

func isCalDAV(provider string) bool {
	return provider == "caldav"
}

// davMultistatus 是 REPORT 的 207 答复。encoding/xml 的标签不带命名空间时按本地名匹配，DAV: 和 caldav 两个命名空间都能认。
type davMultistatus struct {
	Responses []struct {
		Href     string `xml:"href"`
		Propstat []struct {
			Prop struct {
				ETag string `xml:"getetag"`
				Data string `xml:"calendar-data"`
			} `xml:"prop"`
			Status string `xml:"status"`
		} `xml:"propstat"`
	} `xml:"response"`
}

// caldavCollection 把 Endpoint 收成日历集合地址：必须是 https，路径以 / 结尾，去掉查询串和片段。
// 地址里带的用户名取出来单独返回，并从地址上去掉，免得它出现在日志和日程 id 里。
func caldavCollection(endpoint string) (*url.URL, string, error) {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || !strings.EqualFold(u.Scheme, "https") || u.Host == "" {
		return nil, "", fmt.Errorf("connector: CalDAV 要填日历集合的完整 https 地址，例如 https://caldav.fastmail.com/dav/calendars/user/you@fastmail.com/Default/")
	}
	user := ""
	if u.User != nil {
		user = u.User.Username()
		u.User = nil
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
		u.RawPath = ""
	}
	u.RawQuery, u.Fragment = "", ""
	return u, user, nil
}

// caldavCreds 从令牌里拆出用户名和密码。令牌是「用户名:密码」时按第一个冒号切开（用户名里不会有冒号，密码里可以有）；
// 没有冒号时整个令牌是密码，用户名用地址里写的那个。
func caldavCreds(token, urlUser string) (string, string) {
	if user, pass, ok := strings.Cut(token, ":"); ok {
		if strings.TrimSpace(user) == "" {
			user = urlUser
		}
		return strings.TrimSpace(user), pass
	}
	return urlUser, token
}

// caldavDo 发一次带 Basic 认证的请求，返回状态码和答复正文（最多 2MB）。网络错误原样返回。
func caldavDo(client *http.Client, method, target, urlUser, token, body string, hdr map[string]string) (int, []byte, error) {
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, target, rd)
	if err != nil {
		return 0, nil, err
	}
	user, pass := caldavCreds(token, urlUser)
	req.SetBasicAuth(user, pass)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return resp.StatusCode, raw, nil
}

// caldavStatus 把失败的状态码译成给人看的错误。2xx（包括 207）返回 nil。
func caldavStatus(code int, raw []byte, what string) error {
	switch {
	case code == 401 || code == 403:
		return fmt.Errorf("connector: CalDAV 登录失败（%d）。令牌要填「用户名:密码」，iCloud、Fastmail 这类开了两步验证的要用应用专用密码", code)
	case code == 412:
		return fmt.Errorf("connector: 这个日程在别处被改过了，先重新读一遍再改 (412)")
	case code >= 300:
		return fmt.Errorf("connector: caldav %s: %d %s", what, code, truncate(strings.TrimSpace(string(raw)), 300))
	}
	return nil
}

// caldavList 读日历集合里一段时间内的日程，按开始时间排好，最多 caldavLatest 个。
// query 里有两个 RFC3339 时间时用作时间段；不像日历词、也不是时间段的查询按主题和地点做子串筛选。
func caldavList(client *http.Client, endpoint, token, query string) ([]map[string]string, error) {
	// 填的可以是服务器、主体、日历主目录或集合地址：caldavResolve 找到日历集合（没令牌时不发请求），见 caldav_discover.go。
	coll, urlUser, err := caldavResolve(client, endpoint, token)
	if err != nil {
		return nil, err
	}
	start := time.Now().UTC()
	end := start.Add(caldavWindow)
	ranged := false
	if min, max, ok := ParseTimeRange(query); ok {
		start, end, ranged = min.UTC(), max.UTC(), true
	}
	report := `<?xml version="1.0" encoding="utf-8"?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop><d:getetag/><c:calendar-data/></d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VEVENT">
        <c:time-range start="` + icsUTC(start) + `" end="` + icsUTC(end) + `"/>
      </c:comp-filter>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`
	code, raw, err := caldavDo(client, "REPORT", coll.String(), urlUser, token, report, map[string]string{
		"Depth":        "1",
		"Content-Type": `application/xml; charset="utf-8"`,
	})
	if err != nil {
		return nil, fmt.Errorf("connector: %s", err.Error())
	}
	if err := caldavStatus(code, raw, "list"); err != nil {
		return nil, err
	}
	var ms davMultistatus
	if err := xml.Unmarshal(raw, &ms); err != nil {
		return nil, fmt.Errorf("connector: caldav list: 答复不是 multistatus（%s）", err.Error())
	}
	// 文字筛选和 gcalList 的 q 参数同一条规则：查询不像日历词、也不是时间段时才算关键词。
	needle := ""
	if q := strings.TrimSpace(query); q != "" && !looksCalendar(q) && !ranged {
		needle = strings.ToLower(q)
	}
	out := []map[string]string{}
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			// 只收 200 的那一组属性；404 的 propstat 是服务器说「这个属性没有」。
			if ps.Prop.Data == "" || (ps.Status != "" && !strings.Contains(ps.Status, " 200")) {
				continue
			}
			evs := icsEvents(ps.Prop.Data)
			if len(evs) == 0 {
				continue
			}
			// 一个资源里有多个 VEVENT 时是重复日程的例外项，第一个是主日程。
			ev := evs[0]
			href := coll.String()
			if ref, err := url.Parse(strings.TrimSpace(r.Href)); err == nil && strings.TrimSpace(r.Href) != "" {
				href = coll.ResolveReference(ref).String()
			}
			if needle != "" && !strings.Contains(strings.ToLower(ev.Summary+" "+ev.Location), needle) {
				continue
			}
			out = append(out, map[string]string{
				"provider": "caldav",
				"kind":     "calendar",
				"id":       href,
				"event_id": href,
				"uid":      ev.UID,
				"subject":  ev.Summary,
				"start":    ev.Start,
				"location": ev.Location,
				"etag":     strings.TrimSpace(ps.Prop.ETag),
				// 参会人的邮箱，逗号隔开；改日程时草稿不写参会人就保留原来的。
				"attendees": strings.Join(icsEmails(ev.Attendees), ","),
			})
			break
		}
	}
	// 开始时间都写成 UTC 的 RFC3339 或全天的 2006-01-02，按字符串排就是按时间排（同一天全天日程排在前面）。
	sort.SliceStable(out, func(i, j int) bool { return out[i]["start"] < out[j]["start"] })
	if len(out) > caldavLatest {
		out = out[:caldavLatest]
	}
	if len(out) == 0 {
		out = append(out, map[string]string{"provider": "caldav", "kind": "calendar", "note": "no events", "query": query})
	}
	return out, nil
}

// caldavWrite 按草稿新建、修改或删除一个日程。规则和 liveWriteEvent 一致：拒绝重复日程；Op 含 delete 时删除；
// Op 含 update 或带了 EventID 时修改；否则新建。修改和删除都要 EventID（读日历时给出的资源地址）和 TargetVersion（ETag）。
func caldavWrite(client *http.Client, endpoint, token string, d Draft) error {
	if strings.TrimSpace(d.Recurrence) != "" {
		return fmt.Errorf("connector: recurring series must be edited in the calendar app (422)")
	}
	// 和读日历一样先找到日历集合（caldav_discover.go）：账号上只填服务器地址时，新建的日程也要落进那个日历，
	// 而不是写到服务器根目录。找过的集合记一小时，写日程一般不会多发请求。没令牌时不发请求。
	coll, urlUser, err := caldavResolve(client, endpoint, token)
	if err != nil {
		return err
	}
	op := strings.ToLower(strings.TrimSpace(d.Op))
	switch {
	case strings.Contains(op, "delete"):
		if strings.TrimSpace(d.EventID) == "" || strings.TrimSpace(d.TargetVersion) == "" {
			return fmt.Errorf("connector: calendar delete needs event_id and target_version")
		}
		target, err := caldavTarget(coll, d.EventID)
		if err != nil {
			return err
		}
		code, raw, err := caldavDo(client, http.MethodDelete, target, urlUser, token, "", map[string]string{"If-Match": d.TargetVersion})
		if err != nil {
			return fmt.Errorf("connector: %s", err.Error())
		}
		// 已经不在了就算删掉了。
		if code == http.StatusNotFound {
			return nil
		}
		return caldavStatus(code, raw, "delete")
	case strings.Contains(op, "update") || strings.TrimSpace(d.EventID) != "":
		if strings.TrimSpace(d.EventID) == "" || strings.TrimSpace(d.TargetVersion) == "" {
			return fmt.Errorf("connector: calendar update needs event_id and target_version")
		}
		// CalDAV 的修改是整份替换：没给的字段会被清掉，所以至少要有主题和开始时间，免得把日程改成空的。
		if strings.TrimSpace(d.Subject) == "" || strings.TrimSpace(d.Start) == "" {
			return fmt.Errorf("connector: CalDAV 改日程是整份替换，要给出主题和开始时间")
		}
		target, err := caldavTarget(coll, d.EventID)
		if err != nil {
			return err
		}
		// 整份替换：UID 不变；草稿没写参会人时原日程的组织者和参会人原样写回（caldav_people.go）。
		uid, who := caldavOriginal(client, target, urlUser, token)
		who.Organizer = caldavOrganizer(endpoint, token)
		start, end := eventWindow(d)
		code, raw, err := caldavDo(client, http.MethodPut, target, urlUser, token, eventICSFor(uid, d, who, start, end, time.Now()), map[string]string{
			"If-Match":     d.TargetVersion,
			"Content-Type": "text/calendar; charset=utf-8",
		})
		if err != nil {
			return fmt.Errorf("connector: %s", err.Error())
		}
		return caldavStatus(code, raw, "update")
	default:
		uid := randomHex(16) + "-yoyo"
		target := coll.ResolveReference(&url.URL{Path: uid + ".ics"}).String()
		start, end := eventWindow(d)
		// 有参会人时写上组织者（账号本人），支持日程调度的服务器才会替人发邀请。
		who := icsPeople{Organizer: caldavOrganizer(endpoint, token)}
		code, raw, err := caldavDo(client, http.MethodPut, target, urlUser, token, eventICSFor(uid, d, who, start, end, time.Now()), map[string]string{
			"If-None-Match": "*",
			"Content-Type":  "text/calendar; charset=utf-8",
		})
		if err != nil {
			return fmt.Errorf("connector: %s", err.Error())
		}
		return caldavStatus(code, raw, "create")
	}
}

// caldavTarget 把 EventID 解析成要改的资源地址。
// 读日历时给出的是完整地址或以 / 开头的路径，按日历集合解析，并且只认同一台服务器（不把账号密码发到别处）；
// 其他写法当作 UID，拼成 <集合>/<UID>.ics。服务器给资源起的名字不一定是 UID，所以最好用读日历时给出的地址。
func caldavTarget(coll *url.URL, eventID string) (string, error) {
	id := strings.TrimSpace(eventID)
	if strings.HasPrefix(id, "/") || strings.Contains(id, "://") {
		ref, err := url.Parse(id)
		if err != nil {
			return "", fmt.Errorf("connector: 日程编号不是有效的地址")
		}
		t := coll.ResolveReference(ref)
		if !strings.EqualFold(t.Scheme, coll.Scheme) || !strings.EqualFold(t.Host, coll.Host) {
			return "", fmt.Errorf("connector: 日程地址在 %s 上，不是这个日历的服务器", t.Host)
		}
		t.User = nil
		return t.String(), nil
	}
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "..") {
		return "", fmt.Errorf("connector: 日程编号 %q 不能用", id)
	}
	return coll.ResolveReference(&url.URL{Path: id + ".ics"}).String(), nil
}

// eventICS 把草稿写成一份只有一个 VEVENT 的 iCalendar。时间一律写 UTC（带 Z），全天日程写 VALUE=DATE。
// 结束不晚于开始时：定时日程按一小时算，全天日程按一天算。参会人写成 ATTENDEE（服务器可能据此发邀请，和 Google 写日程时一样）。
// 每行按 RFC 5545 折到 75 个字节以内，文字里的反斜杠、分号、逗号和换行转义。
func eventICS(uid string, d Draft, start, end, now time.Time) string {
	return eventICSFor(uid, d, icsPeople{}, start, end, now)
}

// eventICSFor 是 eventICS 带上组织者和原日程的参会人，ORGANIZER / ATTENDEE 几行的规则见 caldav_people.go。
func eventICSFor(uid string, d Draft, who icsPeople, start, end, now time.Time) string {
	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//yoyo//yoyo//ZH",
		"CALSCALE:GREGORIAN",
		"BEGIN:VEVENT",
		"UID:" + uid,
		"DTSTAMP:" + icsUTC(now),
	}
	if d.AllDay {
		if !end.After(start) {
			end = start.AddDate(0, 0, 1)
		}
		lines = append(lines, "DTSTART;VALUE=DATE:"+start.Format("20060102"), "DTEND;VALUE=DATE:"+end.Format("20060102"))
	} else {
		if !end.After(start) {
			end = start.Add(time.Hour)
		}
		lines = append(lines, "DTSTART:"+icsUTC(start), "DTEND:"+icsUTC(end))
	}
	if s := strings.TrimSpace(d.Subject); s != "" {
		lines = append(lines, "SUMMARY:"+icsEscape(s))
	}
	if s := strings.TrimSpace(d.Location); s != "" {
		lines = append(lines, "LOCATION:"+icsEscape(s))
	}
	if s := strings.TrimSpace(d.Body); s != "" {
		lines = append(lines, "DESCRIPTION:"+icsEscape(s))
	}
	// ORGANIZER 和 ATTENDEE：To 和 Attendees 都算参会人；整份替换时没写参会人就写回原来的（caldav_people.go）。
	lines = append(lines, icsGuestLines(d, who)...)
	lines = append(lines, "END:VEVENT", "END:VCALENDAR")
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(icsFold(l))
		b.WriteString("\r\n")
	}
	return b.String()
}

// icsUTC 是 iCalendar 的 UTC 时间写法：20261130T010000Z。
func icsUTC(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// icsEscape 按 RFC 5545 转义文字值。Replacer 一遍扫过去，不会把转义出来的反斜杠再转一次。
func icsEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`).Replace(s)
}

// icsUnescape 是 icsEscape 的反过程。参数顺序保证「\\n」先认成反斜杠加 n，而不是换行。
func icsUnescape(s string) string {
	return strings.NewReplacer(`\\`, `\`, `\;`, ";", `\,`, ",", `\n`, "\n", `\N`, "\n").Replace(s)
}

// icsFold 把一行折到每段不超过 75 个字节，续行以一个空格开头。只在字符边界上折，不会把一个汉字劈成两半。
func icsFold(line string) string {
	if len(line) <= 75 {
		return line
	}
	var b strings.Builder
	n := 0
	for _, r := range line {
		size := utf8.RuneLen(r)
		if size < 0 {
			size = 3
		}
		if n+size > 75 {
			b.WriteString("\r\n ")
			n = 1
		}
		b.WriteRune(r)
		n += size
	}
	return b.String()
}

// icsEvent 是从一份 iCalendar 里取出的一个日程。Start 是 UTC 的 RFC3339，全天日程是 2006-01-02。
type icsEvent struct {
	UID, Summary, Start, Location string
	// Organizer、Attendees 是 ORGANIZER 行和每一行 ATTENDEE 的原样（折行已经接好），整份替换时原样写回。
	Organizer string
	Attendees []string
}

// icsEvents 取出一份 iCalendar 里所有 VEVENT 的几项。先把折行接回去；VEVENT 里嵌着的 VALARM 等子组件不算，
// 免得提醒的 SUMMARY / DESCRIPTION 盖掉日程本身的。
func icsEvents(data string) []icsEvent {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	var lines []string
	for _, l := range strings.Split(data, "\n") {
		if (strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t")) && len(lines) > 0 {
			lines[len(lines)-1] += l[1:]
			continue
		}
		lines = append(lines, l)
	}
	var out []icsEvent
	var cur *icsEvent
	depth := 0
	for _, l := range lines {
		name, params, value := icsProp(l)
		switch {
		case name == "BEGIN" && strings.EqualFold(value, "VEVENT"):
			cur, depth = &icsEvent{}, 0
		case name == "END" && strings.EqualFold(value, "VEVENT"):
			if cur != nil {
				out = append(out, *cur)
			}
			cur = nil
		case cur == nil:
		case name == "BEGIN":
			depth++
		case name == "END":
			depth--
		case depth > 0:
		case name == "UID":
			cur.UID = value
		case name == "SUMMARY":
			cur.Summary = icsUnescape(value)
		case name == "LOCATION":
			cur.Location = icsUnescape(value)
		case name == "DTSTART":
			cur.Start = icsTime(params, value)
		case name == "ORGANIZER":
			cur.Organizer = l
		case name == "ATTENDEE":
			cur.Attendees = append(cur.Attendees, l)
		}
	}
	return out
}

// icsProp 把一行拆成属性名（大写）、参数（键大写，值去掉引号）和值。不是「名字:值」的行返回空名字。
func icsProp(line string) (string, map[string]string, string) {
	head, value, ok := strings.Cut(line, ":")
	if !ok {
		return "", nil, ""
	}
	parts := strings.Split(head, ";")
	params := map[string]string{}
	for _, p := range parts[1:] {
		k, v, _ := strings.Cut(p, "=")
		params[strings.ToUpper(strings.TrimSpace(k))] = strings.Trim(strings.TrimSpace(v), `"`)
	}
	return strings.ToUpper(strings.TrimSpace(parts[0])), params, strings.TrimSpace(value)
}

// icsTime 把 DTSTART 译成 UTC 的 RFC3339；全天日程（VALUE=DATE 或 8 位日期）译成 2006-01-02。
// 带 TZID 的本地时间按那个时区解释，时区库里没有时按本机时区。认不出的写法原样返回。
func icsTime(params map[string]string, v string) string {
	if strings.EqualFold(params["VALUE"], "DATE") || len(v) == 8 {
		if t, err := time.Parse("20060102", v); err == nil {
			return t.Format("2006-01-02")
		}
		return v
	}
	if strings.HasSuffix(v, "Z") {
		if t, err := time.Parse("20060102T150405Z", v); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
		return v
	}
	loc := time.Local
	if tz := params["TZID"]; tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	if t, err := time.ParseInLocation("20060102T150405", v, loc); err == nil {
		return t.UTC().Format(time.RFC3339)
	}
	return v
}

// randomHex 返回 n 个随机字节的十六进制。随机数取不到时退回用纳秒时间，只要不重复就行。
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
