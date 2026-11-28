package connector

import (
	"net/http"
	"strings"
	"testing"
)

// TestCalDAVWritesOrganizerAndGuests 确认新建日程时：组织者取自主体的 calendar-user-address-set（填的是日历集合时多问主体一次）；
// To 和 Attendees 都算参会人，按邮箱去重；组织者自己不列成参会人。
func TestCalDAVWritesOrganizerAndGuests(t *testing.T) {
	put := ""
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == "PROPFIND" && r.URL.Path == "/dav/cal/":
			return jsonResp(207, davMS(davResp(r.URL.Path, `<d:resourcetype><d:collection/><c:calendar/></d:resourcetype><d:current-user-principal><d:href>/p/ada/</d:href></d:current-user-principal>`))), nil
		case r.Method == "PROPFIND" && r.URL.Path == "/p/ada/":
			return jsonResp(207, davMS(davResp(r.URL.Path, `<c:calendar-user-address-set><d:href>/p/ada/</d:href><d:href>mailto:ada@example.com</d:href></c:calendar-user-address-set>`))), nil
		case r.Method == http.MethodPut:
			put = readBody(r)
			return jsonResp(201, ""), nil
		}
		return jsonResp(404, ""), nil
	})}
	d := Draft{Subject: "评审", Start: "2026-11-30T02:00:00Z", To: "bob@example.com", Attendees: []string{"Bob <bob@example.com>", "carol@example.com", "ada@example.com"}}
	if err := caldavWrite(client, "https://org.example.com/dav/cal/", "ada:pw", d); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(put, "ORGANIZER:mailto:ada@example.com\r\n") || strings.Count(put, "mailto:bob@example.com") != 1 ||
		!strings.Contains(put, "ATTENDEE;RSVP=TRUE:mailto:carol@example.com\r\n") || strings.Contains(put, "ATTENDEE;RSVP=TRUE:mailto:ada@example.com") {
		t.Fatalf("ics:\n%s", put)
	}

	// 没有参会人时不写 ORGANIZER。
	put = ""
	if err := caldavWrite(client, "https://org.example.com/dav/cal/", "ada:pw", Draft{Subject: "自己", Start: "2026-11-30T02:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(put, "ORGANIZER") || strings.Contains(put, "ATTENDEE") {
		t.Fatalf("solo ics:\n%s", put)
	}
}

// TestCalDAVUpdateKeepsGuests 确认整份替换时：草稿没写参会人就把原来的组织者和参会人原样写回；
// 写了参会人时原来就在的人留着原来那一行（回过的「接受」不丢），新的人加上，不在草稿里的去掉；UID 不变。
func TestCalDAVUpdateKeepsGuests(t *testing.T) {
	var puts []string
	original := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:orig-uid\r\nORGANIZER;CN=Ada:mailto:ada@example.com\r\n" +
		"ATTENDEE;PARTSTAT=ACCEPTED:mailto:bob@example.com\r\nSUMMARY:旧\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.Method {
		case "PROPFIND":
			return jsonResp(207, davMS(davResp(r.URL.Path, `<d:resourcetype><d:collection/><c:calendar/></d:resourcetype>`))), nil
		case http.MethodGet:
			return jsonResp(200, original), nil
		case http.MethodPut:
			puts = append(puts, readBody(r))
			return jsonResp(204, ""), nil
		}
		return jsonResp(404, ""), nil
	})}
	ep := "https://keep.example.com/dav/cal/"
	upd := Draft{Op: "calendar.update", EventID: ep + "b.ics", TargetVersion: `"e"`, Subject: "改", Start: "2026-11-30T03:00:00Z"}
	if err := caldavWrite(client, ep, "ada:pw", upd); err != nil {
		t.Fatal(err)
	}
	upd.Attendees = []string{"bob@example.com", "dan@example.com"}
	if err := caldavWrite(client, ep, "ada:pw", upd); err != nil {
		t.Fatal(err)
	}
	upd.Attendees = []string{"dan@example.com"}
	if err := caldavWrite(client, ep, "ada:pw", upd); err != nil {
		t.Fatal(err)
	}
	if len(puts) != 3 {
		t.Fatalf("puts = %d", len(puts))
	}
	org, bob := "ORGANIZER;CN=Ada:mailto:ada@example.com\r\n", "ATTENDEE;PARTSTAT=ACCEPTED:mailto:bob@example.com\r\n"
	for i, p := range puts {
		if !strings.Contains(p, "UID:orig-uid\r\n") || !strings.Contains(p, org) {
			t.Fatalf("put %d lost uid or organizer:\n%s", i, p)
		}
	}
	if !strings.Contains(puts[0], bob) || strings.Contains(puts[0], "dan@") {
		t.Fatalf("kept guests:\n%s", puts[0])
	}
	if !strings.Contains(puts[1], bob) || !strings.Contains(puts[1], "ATTENDEE;RSVP=TRUE:mailto:dan@example.com\r\n") {
		t.Fatalf("added guest:\n%s", puts[1])
	}
	if strings.Contains(puts[2], "bob@") || !strings.Contains(puts[2], "mailto:dan@example.com") {
		t.Fatalf("replaced guests:\n%s", puts[2])
	}
}

// fakeFeishuGuests 冒充飞书：新建日程回 e9；加参会人的答复是 *attendees（空时回 code 0）。
func fakeFeishuGuests(calls *[]feishuSeen, attendees *string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		*calls = append(*calls, feishuSeen{r.Method, r.URL.Path, r.URL.RawQuery, readBody(r), r.Header.Get("Authorization")})
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/attendees"):
			if *attendees != "" {
				return jsonResp(200, *attendees), nil
			}
			return jsonResp(200, `{"code":0,"data":{}}`), nil
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/events"):
			return jsonResp(200, `{"code":0,"data":{"event":{"event_id":"e9"}}}`), nil
		}
		return jsonResp(200, `{"code":0,"data":{}}`), nil
	})}
}

// TestFeishuAddsGuests 确认飞书新建日程后按 open_id 加参会人（邮箱、用户、群、会议室各认一种，去重，要发通知）；
// 认不出的参会人一个请求都不发；只加参会人的修改不发 PATCH；草稿里的 calendar_id 优先；加参会人失败时说清日程已经建好。
func TestFeishuAddsGuests(t *testing.T) {
	var calls []feishuSeen
	attendees := ""
	client := fakeFeishuGuests(&calls, &attendees)

	d := Draft{Subject: "周会", Start: "2026-11-30T02:00:00Z", To: "bob@example.com", Attendees: []string{"Bob <bob@example.com>，ou_x", "oc_y", "omm_z"}}
	if err := feishuWriteEvent(client, "cal_x", "u-tok", d); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[1].path != "/open-apis/calendar/v4/calendars/cal_x/events/e9/attendees" || calls[1].query != "user_id_type=open_id" {
		t.Fatalf("calls = %+v", calls)
	}
	body := calls[1].body
	if strings.Count(body, "bob@example.com") != 1 || !strings.Contains(body, `"third_party_email":"bob@example.com"`) ||
		!strings.Contains(body, `"user_id":"ou_x"`) || !strings.Contains(body, `"chat_id":"oc_y"`) || !strings.Contains(body, `"room_id":"omm_z"`) ||
		!strings.Contains(body, `"need_notification":true`) {
		t.Fatalf("attendees body = %s", body)
	}

	calls = nil
	if err := feishuWriteEvent(client, "cal_x", "u-tok", Draft{Subject: "周会", Attendees: []string{"鲍勃"}}); err == nil || !strings.Contains(err.Error(), "鲍勃") || len(calls) != 0 {
		t.Fatalf("unknown guest: %v %+v", err, calls)
	}

	calls = nil
	if err := feishuWriteEvent(client, "cal_x", "u-tok", Draft{EventID: "e2", CalendarID: "cal_shared", Attendees: []string{"ou_x"}}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != "/open-apis/calendar/v4/calendars/cal_shared/events/e2/attendees" {
		t.Fatalf("guest-only update = %+v", calls)
	}

	calls, attendees = nil, `{"code":190002,"msg":"no permission"}`
	err := feishuWriteEvent(client, "cal_x", "u-tok", d)
	if err == nil || !alreadyWritten(err) || !strings.Contains(err.Error(), "e9") || !strings.Contains(err.Error(), "不要重新建") {
		t.Fatalf("attendee failure = %v", err)
	}
}

// TestSendDoesNotRetryWrittenEvent 确认日程建好、加参会人时令牌被拒，liveSend 不按授权过期换票重发（不会多出一个一样的日程）。
func TestSendDoesNotRetryWrittenEvent(t *testing.T) {
	b, _ := Open(t.TempDir())
	toks := memTokens{"connector.f1": `{"access_token":"u-tok","refresh_token":"r-tok","client_id":"cid"}`}
	b.SetTokens(toks)
	var calls []feishuSeen
	attendees := `{"code":99991663,"msg":"invalid access token"}`
	b.http = fakeFeishuGuests(&calls, &attendees)
	a := b.Connect(Account{ID: "f1", Kind: KindCalendar, Provider: "feishu-calendar", Endpoint: "cal_x", VaultKey: "connector.f1"})
	err := b.liveSend(a, Draft{Subject: "周会", Start: "2026-11-30T02:00:00Z", Attendees: []string{"ou_x"}})
	if err == nil || !alreadyWritten(err) {
		t.Fatalf("err = %v", err)
	}
	creates := 0
	for _, c := range calls {
		if c.method == http.MethodPost && strings.HasSuffix(c.path, "/events") {
			creates++
		}
		if strings.Contains(c.path, "token") {
			t.Fatalf("refreshed after a partial write: %+v", c)
		}
	}
	if creates != 1 {
		t.Fatalf("creates = %d, calls = %+v", creates, calls)
	}
}

// pickServer 冒充一个 CalDAV 服务器：根地址给主体，主体给日历主目录 /cal/ada/；主目录下有「工作」「待办」（只放 VTODO）和「家里」；
// 每个日历自己被问时说自己是日历。REPORT 记下问的地址。
func pickServer(report *string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		p := r.URL.Path
		switch {
		case r.Method == "REPORT":
			*report = r.URL.String()
			return jsonResp(207, davMS()), nil
		case r.Method != "PROPFIND":
			return jsonResp(404, ""), nil
		case p == "/":
			return jsonResp(207, davMS(davResp(p, `<d:resourcetype><d:collection/></d:resourcetype><d:current-user-principal><d:href>/p/</d:href></d:current-user-principal>`))), nil
		case p == "/p/":
			return jsonResp(207, davMS(davResp(p, `<c:calendar-home-set><d:href>/cal/ada/</d:href></c:calendar-home-set><c:calendar-user-address-set><d:href>mailto:ada@example.com</d:href></c:calendar-user-address-set>`))), nil
		case p == "/cal/ada/" && r.Header.Get("Depth") == "1":
			return jsonResp(207, davMS(
				davResp("/cal/ada/", `<d:resourcetype><d:collection/></d:resourcetype>`),
				davResp("/cal/ada/work/", `<d:resourcetype><d:collection/><c:calendar/></d:resourcetype><d:displayname>工作</d:displayname><c:supported-calendar-component-set><c:comp name="VEVENT"/></c:supported-calendar-component-set>`),
				davResp("/cal/ada/todo/", `<d:resourcetype><d:collection/><c:calendar/></d:resourcetype><d:displayname>待办</d:displayname><c:supported-calendar-component-set><c:comp name="VTODO"/></c:supported-calendar-component-set>`),
				davResp("/cal/ada/home/", `<d:resourcetype><d:collection/><c:calendar/></d:resourcetype><d:displayname>家里</d:displayname>`),
			)), nil
		case strings.HasPrefix(p, "/cal/ada/") && p != "/cal/ada/":
			return jsonResp(207, davMS(davResp(p, `<d:resourcetype><d:collection/><c:calendar/></d:resourcetype>`))), nil
		}
		return jsonResp(404, ""), nil
	})}
}

// TestCalDAVChooseCalendar 确认能列出主目录下能放日程的日历（待办日历不算），没选时标出自动用的那个；
// 选了之后读日历走选的那个，重新打开还记得；不在列表里的不让选；清空回到自动选；选的地址不在同一家主机时不用它。
func TestCalDAVChooseCalendar(t *testing.T) {
	dir := t.TempDir()
	b, _ := Open(dir)
	b.SetTokens(memTokens{"connector.c1": "ada:pw"})
	report := ""
	b.http = pickServer(&report)
	b.Connect(Account{ID: "c1", Kind: KindCalendar, Provider: "caldav", Endpoint: "https://pick.example.com/", VaultKey: "connector.c1"})

	list, err := b.Calendars("c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "工作" || !list[0].Current || list[1].Name != "家里" || list[1].Current ||
		list[1].ID != "https://pick.example.com/cal/ada/home/" {
		t.Fatalf("list = %+v", list)
	}

	acct, err := b.ChooseCalendar("c1", "https://pick.example.com/cal/ada/home")
	if err != nil || acct.Calendar != "https://pick.example.com/cal/ada/home/" || acct.CalendarName != "家里" {
		t.Fatalf("choose = %+v %v", acct, err)
	}
	if _, err := b.Read("c1", ""); err != nil {
		t.Fatal(err)
	}
	if report != "https://pick.example.com/cal/ada/home/" {
		t.Fatalf("read went to %q", report)
	}
	again, _ := Open(dir)
	if got := again.List(); len(got) != 1 || got[0].Calendar != "https://pick.example.com/cal/ada/home/" {
		t.Fatalf("reopened = %+v", got)
	}

	if _, err := b.ChooseCalendar("c1", "https://pick.example.com/cal/ada/todo/"); err == nil {
		t.Fatal("a task-only calendar must not be choosable")
	}
	if acct, err := b.ChooseCalendar("c1", ""); err != nil || acct.Calendar != "" || acct.CalendarName != "" {
		t.Fatalf("clear = %+v %v", acct, err)
	}
	foreign := Account{Provider: "caldav", Endpoint: "https://pick.example.com/", Calendar: "https://evil.test/cal/"}
	if got := calendarEndpoint(foreign); got != foreign.Endpoint {
		t.Fatalf("foreign calendar used: %q", got)
	}
}

// TestFeishuCalendarsList 确认飞书只列出能写的日历（只读、会议室、已删除的不给），主日历排第一、没名字时叫「主日历」，按 page_token 翻页。
func TestFeishuCalendarsList(t *testing.T) {
	pages := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		pages++
		if r.URL.Query().Get("page_token") == "" {
			return jsonResp(200, `{"code":0,"data":{"has_more":true,"page_token":"p2","calendar_list":[`+
				`{"calendar_id":"c_shared","summary":"团队","type":"shared","role":"writer"},`+
				`{"calendar_id":"c_ro","summary":"公司假期","type":"shared","role":"reader"},`+
				`{"calendar_id":"c_room","summary":"三楼","type":"resource","role":"owner"}]}}`), nil
		}
		return jsonResp(200, `{"code":0,"data":{"has_more":false,"calendar_list":[`+
			`{"calendar_id":"c_gone","summary":"旧","type":"shared","role":"owner","is_deleted":true},`+
			`{"calendar_id":"c_me","summary":"","type":"primary","role":"owner"}]}}`), nil
	})}
	list, err := feishuCalendars(client, "u-tok")
	if err != nil {
		t.Fatal(err)
	}
	if pages != 2 || len(list) != 2 || list[0].ID != "c_me" || !list[0].Primary || list[0].Name != "主日历" || list[1].ID != "c_shared" {
		t.Fatalf("pages = %d, list = %+v", pages, list)
	}
}
