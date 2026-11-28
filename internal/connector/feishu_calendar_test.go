package connector

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// feishuCall 记下的一次请求。
type feishuSeen struct {
	method, path, query, body, auth string
}

// fakeFeishu 冒充飞书日历接口：primary 给出主日历 cal_1；列日程给三条（一条已取消、次序和时间先后相反）；其余写操作回 code 0。
func fakeFeishu(calls *[]feishuSeen) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		*calls = append(*calls, feishuSeen{r.Method, r.URL.Path, r.URL.RawQuery, readBody(r), r.Header.Get("Authorization")})
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/open-apis/calendar/v4/calendars/primary":
			return jsonResp(200, `{"code":0,"data":{"calendars":[{"calendar":{"calendar_id":"cal_1"}}]}}`), nil
		case r.Method == http.MethodGet:
			return jsonResp(200, `{"code":0,"data":{"items":[`+
				`{"event_id":"e2","summary":"评审","status":"confirmed","start_time":{"timestamp":"1795999200"},"location":{"name":"三楼"}},`+
				`{"event_id":"e0","summary":"取消了","status":"cancelled","start_time":{"timestamp":"1795900000"}},`+
				`{"event_id":"e1","summary":"团建","status":"confirmed","start_time":{"date":"2026-11-29"}}]}}`), nil
		default:
			return jsonResp(200, `{"code":0,"data":{}}`), nil
		}
	})}
}

// TestFeishuCalendarReads 确认没填 calendar_id 时先要主日历；列日程带查询里的时间段（秒），去掉已取消的，按开始时间排好，
// 定时日程的时间换成 UTC 的 RFC3339。
func TestFeishuCalendarReads(t *testing.T) {
	var calls []feishuSeen
	rows, err := feishuEvents(fakeFeishu(&calls), "", "u-tok", "2026-11-28T00:00:00Z 2026-12-05T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].path != "/open-apis/calendar/v4/calendars/primary" || calls[1].path != "/open-apis/calendar/v4/calendars/cal_1/events" {
		t.Fatalf("calls = %+v", calls)
	}
	from := strconv.FormatInt(time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC).Unix(), 10)
	to := strconv.FormatInt(time.Date(2026, 12, 5, 0, 0, 0, 0, time.UTC).Unix(), 10)
	if !strings.Contains(calls[1].query, "start_time="+from) || !strings.Contains(calls[1].query, "end_time="+to) || calls[1].auth != "Bearer u-tok" {
		t.Fatalf("list request = %+v", calls[1])
	}
	if len(rows) != 2 || rows[0]["id"] != "e1" || rows[0]["start"] != "2026-11-29" || rows[1]["id"] != "e2" {
		t.Fatalf("rows = %+v", rows)
	}
	if want := time.Unix(1795999200, 0).UTC().Format(time.RFC3339); rows[1]["start"] != want || rows[1]["location"] != "三楼" || rows[1]["calendar_id"] != "cal_1" {
		t.Fatalf("timed row = %+v, want start %s", rows[1], want)
	}
}

// TestFeishuCalendarWrites 确认填了 calendar_id 时不再要主日历；只改主题的修改不带时间（不会把日程挪到现在）；
// 新建带秒级时间戳和时区；删除走 DELETE；令牌失效和业务错误都报出来。
func TestFeishuCalendarWrites(t *testing.T) {
	var calls []feishuSeen
	client := fakeFeishu(&calls)

	if err := feishuWriteEvent(client, "cal_x", "u-tok", Draft{EventID: "e2", Subject: "评审（改）"}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPatch || calls[0].path != "/open-apis/calendar/v4/calendars/cal_x/events/e2" ||
		!strings.Contains(calls[0].body, "评审（改）") || strings.Contains(calls[0].body, "start_time") {
		t.Fatalf("patch = %+v", calls)
	}

	calls = nil
	if err := feishuWriteEvent(client, "cal_x", "u-tok", Draft{Subject: "新会", Start: "2026-11-30T02:00:00Z", TimeZone: "Asia/Shanghai"}); err != nil {
		t.Fatal(err)
	}
	ts := strconv.FormatInt(time.Date(2026, 11, 30, 2, 0, 0, 0, time.UTC).Unix(), 10)
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != "/open-apis/calendar/v4/calendars/cal_x/events" ||
		!strings.Contains(calls[0].body, `"timestamp":"`+ts+`"`) || !strings.Contains(calls[0].body, "Asia/Shanghai") {
		t.Fatalf("create = %+v", calls)
	}

	calls = nil
	if err := feishuWriteEvent(client, "cal_x", "u-tok", Draft{Op: "calendar.delete", EventID: "e2"}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != "/open-apis/calendar/v4/calendars/cal_x/events/e2" {
		t.Fatalf("delete = %+v", calls)
	}
	if err := feishuWriteEvent(client, "cal_x", "u-tok", Draft{Op: "calendar.delete"}); err == nil {
		t.Fatal("delete without event_id must fail")
	}

	for body, want := range map[string]string{
		`{"code":99991663,"msg":"invalid access token"}`: "重新授权",
		`{"code":190002,"msg":"no permission"}`:          "190002",
	} {
		bad := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return jsonResp(200, body), nil
		})}
		if _, err := feishuEvents(bad, "cal_x", "u-tok", ""); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: err = %v, want %q", body, err, want)
		}
	}
}
