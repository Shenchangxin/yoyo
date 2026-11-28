package connector

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	// 嵌入时区库，让带 TZID 的日程在没有 Go 安装目录的机器上也按 Asia/Shanghai 解释，测试结果不随机器变。
	_ "time/tzdata"
)

// caldavReport 是 REPORT 的答复：两个日程，次序和时间先后相反。
// b 的主题折了行、带转义的逗号，里面嵌着一个 VALARM（它的 SUMMARY 不该盖掉日程的），开始时间带 TZID；a 是全天日程。
const caldavReport = `<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
<d:response><d:href>/dav/cal/b.ics</d:href><d:propstat><d:prop><d:getetag>"e2"</d:getetag><cal:calendar-data>BEGIN:VCALENDAR
BEGIN:VEVENT
UID:b-uid
SUMMARY:周会\, 第二
 次
DTSTART;TZID=Asia/Shanghai:20261130T100000
LOCATION:三楼
BEGIN:VALARM
SUMMARY:提醒
END:VALARM
END:VEVENT
END:VCALENDAR
</cal:calendar-data></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>
<d:response><d:href>/dav/cal/a.ics</d:href><d:propstat><d:prop><d:getetag>"e1"</d:getetag><cal:calendar-data>BEGIN:VCALENDAR
BEGIN:VEVENT
UID:a-uid
SUMMARY:全天
DTSTART;VALUE=DATE:20261129
END:VEVENT
END:VCALENDAR
</cal:calendar-data></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>
</d:multistatus>`

// readBody 读出请求正文；GET 这类没有正文的返回空串。
func readBody(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	raw, _ := io.ReadAll(r.Body)
	return string(raw)
}

// TestCalDAVListReadsEvents 确认 REPORT 带 Depth: 1、Basic 认证和查询里的时间段；答复里的日程按开始时间排好，
// 折行接回、转义解开、VALARM 不混进来、TZID 换成 UTC、资源地址和 ETag 带上；关键词查询按主题和地点筛。
func TestCalDAVListReadsEvents(t *testing.T) {
	var seen *http.Request
	var body string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		// 查找日历集合时问的 PROPFIND：回答「这个地址本身就是日历」，不记进 seen。
		if r.Method == "PROPFIND" {
			return jsonResp(207, caldavSelfCalendar), nil
		}
		seen, body = r, readBody(r)
		return jsonResp(207, caldavReport), nil
	})}
	rows, err := caldavList(client, "https://caldav.example.com/dav/cal", "ada:app-pass", "2026-11-28T00:00:00Z 2026-12-05T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if seen.Method != "REPORT" || seen.URL.String() != "https://caldav.example.com/dav/cal/" || seen.Header.Get("Depth") != "1" {
		t.Fatalf("request = %s %s depth=%q", seen.Method, seen.URL, seen.Header.Get("Depth"))
	}
	if user, pass, ok := seen.BasicAuth(); !ok || user != "ada" || pass != "app-pass" {
		t.Fatalf("basic auth = %q %q %v", user, pass, ok)
	}
	if !strings.Contains(body, `start="20261128T000000Z"`) || !strings.Contains(body, `end="20261205T000000Z"`) {
		t.Fatalf("time range missing:\n%s", body)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	if r := rows[0]; r["subject"] != "全天" || r["start"] != "2026-11-29" || r["etag"] != `"e1"` || r["id"] != "https://caldav.example.com/dav/cal/a.ics" {
		t.Fatalf("all-day first: %+v", r)
	}
	if r := rows[1]; r["subject"] != "周会, 第二次" || r["start"] != "2026-11-30T02:00:00Z" || r["location"] != "三楼" ||
		r["uid"] != "b-uid" || r["event_id"] != "https://caldav.example.com/dav/cal/b.ics" || r["etag"] != `"e2"` {
		t.Fatalf("timed event: %+v", r)
	}
	filtered, err := caldavList(client, "https://caldav.example.com/dav/cal/", "ada:app-pass", "三楼")
	if err != nil || len(filtered) != 1 || filtered[0]["uid"] != "b-uid" {
		t.Fatalf("keyword filter = %+v %v", filtered, err)
	}
}

// TestCalDAVListErrors 确认不是 https 的地址不去连，登录被拒时说清要用应用专用密码。
func TestCalDAVListErrors(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResp(401, ""), nil
	})}
	if _, err := caldavList(client, "http://caldav.example.com/dav/cal/", "a:b", ""); err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("plain http err = %v", err)
	}
	if _, err := caldavList(client, "https://caldav.example.com/dav/cal/", "a:b", ""); err == nil || !strings.Contains(err.Error(), "应用专用密码") {
		t.Fatalf("401 err = %v", err)
	}
}

// TestCalDAVWriteUsesETags 确认新建带 If-None-Match: *；修改先读原来的 UID、再带 If-Match 整份替换，412 时说清别处改过；
// 缺主题的修改、别的主机上的日程地址都不发请求；删除带 If-Match，已经不在也算删掉；重复日程拒绝。
func TestCalDAVWriteUsesETags(t *testing.T) {
	type call struct{ method, url, ifMatch, ifNone, body string }
	var calls []call
	status := 201
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		// 查找日历集合的 PROPFIND：回答「这个地址本身就是日历」，不记进 calls，下面的断言只看读写请求。
		if r.Method == "PROPFIND" {
			return jsonResp(207, caldavSelfCalendar), nil
		}
		calls = append(calls, call{r.Method, r.URL.String(), r.Header.Get("If-Match"), r.Header.Get("If-None-Match"), readBody(r)})
		if r.Method == http.MethodGet {
			return jsonResp(200, "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:server-uid\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"), nil
		}
		return jsonResp(status, ""), nil
	})}
	ep := "https://caldav.example.com/dav/cal/"

	if err := caldavWrite(client, ep, "ada:pw", Draft{Subject: "评审", Start: "2026-11-30T02:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("create calls = %+v", calls)
	}
	if c := calls[0]; c.method != http.MethodPut || !strings.HasPrefix(c.url, ep) || !strings.HasSuffix(c.url, "-yoyo.ics") || c.ifNone != "*" ||
		!strings.Contains(c.body, "SUMMARY:评审\r\n") || !strings.Contains(c.body, "DTSTART:20261130T020000Z\r\n") || !strings.Contains(c.body, "DTEND:") {
		t.Fatalf("create = %+v", c)
	}

	calls, status = nil, 204
	upd := Draft{Op: "calendar.update", EventID: ep + "b.ics", TargetVersion: `"e2"`, Subject: "评审（改）", Start: "2026-11-30T03:00:00Z"}
	if err := caldavWrite(client, ep, "ada:pw", upd); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].method != http.MethodGet || calls[1].method != http.MethodPut ||
		calls[1].url != ep+"b.ics" || calls[1].ifMatch != `"e2"` || !strings.Contains(calls[1].body, "UID:server-uid\r\n") {
		t.Fatalf("update calls = %+v", calls)
	}

	calls, status = nil, 412
	if err := caldavWrite(client, ep, "ada:pw", upd); err == nil || !strings.Contains(err.Error(), "412") {
		t.Fatalf("conflict err = %v", err)
	}

	calls = nil
	if err := caldavWrite(client, ep, "ada:pw", Draft{EventID: ep + "b.ics", TargetVersion: `"e2"`, Start: "2026-11-30T03:00:00Z"}); err == nil || len(calls) != 0 {
		t.Fatalf("update without subject: %v %+v", err, calls)
	}
	if err := caldavWrite(client, ep, "ada:pw", Draft{Op: "calendar.delete", EventID: "https://evil.example.com/x.ics", TargetVersion: "e"}); err == nil || len(calls) != 0 {
		t.Fatalf("foreign host: %v %+v", err, calls)
	}
	if err := caldavWrite(client, ep, "ada:pw", Draft{Subject: "例会", Recurrence: "RRULE:FREQ=WEEKLY"}); err == nil || len(calls) != 0 {
		t.Fatalf("recurrence: %v %+v", err, calls)
	}

	calls, status = nil, 404
	if err := caldavWrite(client, ep, "ada:pw", Draft{Op: "calendar.delete", EventID: "/dav/cal/b.ics", TargetVersion: `"e2"`}); err != nil {
		t.Fatalf("delete of a gone event = %v", err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].url != ep+"b.ics" || calls[0].ifMatch != `"e2"` {
		t.Fatalf("delete calls = %+v", calls)
	}
}

// TestEventICSFoldsAndRoundTrips 确认长主题折到每行不超过 75 字节（不劈开汉字），分号、逗号、反斜杠转义后读回来和原文一样；
// 全天日程没给结束时按一天算。
func TestEventICSFoldsAndRoundTrips(t *testing.T) {
	long := strings.Repeat("季度评审;预算,排期\\", 8)
	start := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)
	ics := eventICS("u1", Draft{Subject: long, Location: "三楼", AllDay: true}, start, time.Time{}, start)
	for _, line := range strings.Split(strings.TrimSuffix(ics, "\r\n"), "\r\n") {
		if len(line) > 75 {
			t.Fatalf("line longer than 75 bytes (%d): %q", len(line), line)
		}
	}
	if !strings.Contains(ics, "DTSTART;VALUE=DATE:20261130\r\nDTEND;VALUE=DATE:20261201\r\n") {
		t.Fatalf("all-day window:\n%s", ics)
	}
	evs := icsEvents(ics)
	if len(evs) != 1 || evs[0].UID != "u1" || evs[0].Summary != long || evs[0].Location != "三楼" || evs[0].Start != "2026-11-30" {
		t.Fatalf("round trip = %+v", evs)
	}
}
