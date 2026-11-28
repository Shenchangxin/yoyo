// feishu_calendar.go 用飞书开放平台的日历 v4 接口读写日程。
// 令牌是用户身份的 user_access_token（以用户身份授权、带日历读写权限的那种），不是机器人的 tenant_access_token：
// 机器人看不到个人的主日历。Endpoint 可以填一个 calendar_id；不填时用这个人的主日历（calendars/primary）。
//
// 读：GET calendars/{id}/events，按时间段（默认从现在起两周，查询里有两个 RFC3339 时间时用它们），去掉已取消的，按开始时间排好。
// 写：新建 POST events；修改 PATCH events/{event_id}，只带草稿里给了的字段（没给开始时间就不动时间）；删除 DELETE events/{event_id}。
// 飞书的日程没有版本号可以比对，所以修改和删除只要 event_id。
// 不拥有：参会人（飞书要另一个接口逐个加，留给 MCP 包）、重复日程（和 Google / Outlook 一样拒绝）。

package connector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// feishuOpenBase 是飞书开放平台的地址。
var feishuOpenBase = "https://open.feishu.cn"

func isFeishuCalendar(provider string) bool {
	return provider == "feishu-calendar"
}

// feishuAuthCode 在飞书返回的错误码表示令牌缺失、无效或过期时为真。
func feishuAuthCode(code int) bool {
	switch code {
	case 99991661, 99991663, 99991668, 99991677:
		return true
	}
	return false
}

// feishuLabel 是错误里写的哪一项飞书连接：英文名放进前缀（authExpired 靠前缀后面的「401」认出令牌失效），中文名写进给人看的那句。
type feishuLabel struct{ en, zh string }

// feishuCalendarWhat 是飞书日历在错误里的名字。
var feishuCalendarWhat = feishuLabel{"feishu calendar", "飞书日历"}

// feishuCall 调一次飞书日历的接口，见 feishuDo。
func feishuCall(client *http.Client, method, apiPath, token string, body any) ([]byte, error) {
	return feishuDo(client, method, apiPath, token, body, feishuCalendarWhat)
}

// feishuDo 调一次飞书接口，返回答复正文。body 非空时按 JSON 发。日历、云文档、任务共用（feishu_docs.go、feishu_task.go）。
// 飞书出错时 HTTP 状态可能是 200 也可能是 4xx，真正的结果看正文里的 code：不为 0 就是失败。
// 令牌问题单独说清要重新授权。what 是错误里写的连接名字。
func feishuDo(client *http.Client, method, apiPath, token string, body any, what feishuLabel) ([]byte, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("connector: no token")
	}
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, feishuOpenBase+apiPath, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var env struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	jerr := json.Unmarshal(raw, &env)
	// 错误里带上「401」：liveRead / liveSend 靠 authExpired 认出令牌失效，有刷新令牌时强制换一张再试一次。
	if resp.StatusCode == http.StatusUnauthorized || feishuAuthCode(env.Code) {
		return nil, fmt.Errorf("connector: %s 401 (code %d): %s的授权过期或无效，请在「连接」里重新授权", what.en, env.Code, what.zh)
	}
	if jerr != nil {
		return nil, fmt.Errorf("connector: %s: %d %s", what.en, resp.StatusCode, truncate(strings.TrimSpace(string(raw)), 300))
	}
	if env.Code != 0 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("connector: %s: %d %s", what.en, env.Code, env.Msg)
	}
	return raw, nil
}

// feishuCalendarID 是要读写的日历：Endpoint 填了就用它，没填就去要这个人的主日历。
func feishuCalendarID(client *http.Client, endpoint, token string) (string, error) {
	if id := strings.TrimSpace(endpoint); id != "" {
		return id, nil
	}
	raw, err := feishuCall(client, http.MethodPost, "/open-apis/calendar/v4/calendars/primary", token, map[string]any{})
	if err != nil {
		return "", err
	}
	var doc struct {
		Data struct {
			Calendars []struct {
				Calendar struct {
					ID string `json:"calendar_id"`
				} `json:"calendar"`
			} `json:"calendars"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &doc)
	for _, c := range doc.Data.Calendars {
		if id := strings.TrimSpace(c.Calendar.ID); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("connector: 飞书没有给出主日历，可以在连接里填一个 calendar_id")
}

// feishuTime 是飞书日程里的时间：定时日程给 timestamp（秒），全天日程给 date。
type feishuTime struct {
	Date      string `json:"date,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Timezone  string `json:"timezone,omitempty"`
}

// text 译成和其他日历一致的写法：定时日程是 UTC 的 RFC3339，全天日程是 2006-01-02。
func (t feishuTime) text() string {
	if n, err := strconv.ParseInt(strings.TrimSpace(t.Timestamp), 10, 64); err == nil && n > 0 {
		return time.Unix(n, 0).UTC().Format(time.RFC3339)
	}
	return strings.TrimSpace(t.Date)
}

// feishuEvents 读一段时间内的日程，按开始时间排好，最多 20 个。已取消的不给。
// 不像日历词、也不是时间段的查询按主题和地点做子串筛选（和 CalDAV 一致）。
func feishuEvents(client *http.Client, endpoint, token, query string) ([]map[string]string, error) {
	cid, err := feishuCalendarID(client, endpoint, token)
	if err != nil {
		return nil, err
	}
	start := time.Now().UTC()
	end := start.Add(caldavWindow)
	ranged := false
	if min, max, ok := ParseTimeRange(query); ok {
		start, end, ranged = min.UTC(), max.UTC(), true
	}
	q := url.Values{
		"start_time": {strconv.FormatInt(start.Unix(), 10)},
		"end_time":   {strconv.FormatInt(end.Unix(), 10)},
	}
	raw, err := feishuCall(client, http.MethodGet, "/open-apis/calendar/v4/calendars/"+url.PathEscape(cid)+"/events?"+q.Encode(), token, nil)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Data struct {
			Items []struct {
				ID       string     `json:"event_id"`
				Summary  string     `json:"summary"`
				Status   string     `json:"status"`
				Start    feishuTime `json:"start_time"`
				Location struct {
					Name string `json:"name"`
				} `json:"location"`
			} `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &doc)
	needle := ""
	if s := strings.TrimSpace(query); s != "" && !looksCalendar(s) && !ranged {
		needle = strings.ToLower(s)
	}
	out := []map[string]string{}
	for _, it := range doc.Data.Items {
		if strings.EqualFold(it.Status, "cancelled") || strings.TrimSpace(it.ID) == "" {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(it.Summary+" "+it.Location.Name), needle) {
			continue
		}
		out = append(out, map[string]string{
			"provider":    "feishu-calendar",
			"kind":        "calendar",
			"id":          it.ID,
			"event_id":    it.ID,
			"calendar_id": cid,
			"subject":     it.Summary,
			"start":       it.Start.text(),
			"location":    it.Location.Name,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i]["start"] < out[j]["start"] })
	if len(out) > caldavLatest {
		out = out[:caldavLatest]
	}
	if len(out) == 0 {
		out = append(out, map[string]string{"provider": "feishu-calendar", "kind": "calendar", "note": "no events", "query": query})
	}
	return out, nil
}

// feishuEventBody 把草稿写成飞书日程的请求体。patch 为真时只放草稿里给了的字段：
// 没给开始时间就不带 start_time / end_time，免得修改主题时把时间挪到「现在」。
// 结束不晚于开始时：定时日程按一小时算，全天日程按一天算。
func feishuEventBody(d Draft, patch bool) map[string]any {
	body := map[string]any{}
	if s := strings.TrimSpace(d.Subject); s != "" || !patch {
		body["summary"] = s
	}
	if s := strings.TrimSpace(d.Body); s != "" {
		body["description"] = s
	}
	if s := strings.TrimSpace(d.Location); s != "" {
		body["location"] = map[string]string{"name": s}
	}
	if patch && strings.TrimSpace(d.Start) == "" {
		return body
	}
	start, end := eventWindow(d)
	if d.AllDay {
		if !end.After(start) {
			end = start.AddDate(0, 0, 1)
		}
		body["start_time"] = feishuTime{Date: start.Format("2006-01-02")}
		body["end_time"] = feishuTime{Date: end.Format("2006-01-02")}
		return body
	}
	if !end.After(start) {
		end = start.Add(time.Hour)
	}
	// timestamp 是绝对时间；timezone 只影响飞书里怎么显示，草稿没写时区就不带。
	tz := strings.TrimSpace(d.TimeZone)
	body["start_time"] = feishuTime{Timestamp: strconv.FormatInt(start.Unix(), 10), Timezone: tz}
	body["end_time"] = feishuTime{Timestamp: strconv.FormatInt(end.Unix(), 10), Timezone: tz}
	return body
}

// feishuWriteEvent 按草稿新建、修改或删除一个飞书日程。规则和 liveWriteEvent 一致：拒绝重复日程；
// Op 含 delete 时删除；Op 含 update 或带了 EventID 时修改；否则新建。修改和删除要 EventID。
func feishuWriteEvent(client *http.Client, endpoint, token string, d Draft) error {
	if strings.TrimSpace(d.Recurrence) != "" {
		return fmt.Errorf("connector: recurring series must be edited in the calendar app (422)")
	}
	op := strings.ToLower(strings.TrimSpace(d.Op))
	eid := strings.TrimSpace(d.EventID)
	// 参会人先全部认出来，有认不出的就一个请求都不发（calendar_guests.go）。
	var guests []map[string]any
	if !strings.Contains(op, "delete") {
		g, err := feishuGuests(d)
		if err != nil {
			return err
		}
		guests = g
	}
	// 读日历时每条都带着 calendar_id，改和删时用它；没带时用账号上选的日历或主日历。
	cid := strings.TrimSpace(d.CalendarID)
	if cid == "" {
		c, err := feishuCalendarID(client, endpoint, token)
		if err != nil {
			return err
		}
		cid = c
	}
	base := "/open-apis/calendar/v4/calendars/" + url.PathEscape(cid) + "/events"
	switch {
	case strings.Contains(op, "delete"):
		if eid == "" {
			return fmt.Errorf("connector: calendar delete needs event_id")
		}
		_, err := feishuCall(client, http.MethodDelete, base+"/"+url.PathEscape(eid), token, nil)
		return err
	case strings.Contains(op, "update") || eid != "":
		if eid == "" {
			return fmt.Errorf("connector: calendar update needs event_id")
		}
		// 只加参会人时正文是空的，不发 PATCH。
		if body := feishuEventBody(d, true); len(body) > 0 || len(guests) == 0 {
			if _, err := feishuCall(client, http.MethodPatch, base+"/"+url.PathEscape(eid), token, body); err != nil {
				return err
			}
		}
		return feishuAddGuests(client, base, eid, token, guests)
	default:
		raw, err := feishuCall(client, http.MethodPost, base, token, feishuEventBody(d, false))
		if err != nil {
			return err
		}
		var doc struct {
			Data struct {
				Event struct {
					ID string `json:"event_id"`
				} `json:"event"`
			} `json:"data"`
		}
		_ = json.Unmarshal(raw, &doc)
		if len(guests) > 0 && strings.TrimSpace(doc.Data.Event.ID) == "" {
			return writtenError{fmt.Errorf("connector: 飞书日程已经建好，但答复里没有 event_id，参会人没加上。不要重新建，请在飞书里加参会人")}
		}
		return feishuAddGuests(client, base, strings.TrimSpace(doc.Data.Event.ID), token, guests)
	}
}

// feishuAddGuests 给写好的日程加参会人（POST events/{id}/attendees，用户按 open_id），并让飞书发通知。只加不减。
// 失败时日程本身已经写好了，返回 writtenError：liveSend 不会按授权过期重发，免得多出一个一样的日程。
func feishuAddGuests(client *http.Client, base, eid, token string, guests []map[string]any) error {
	if len(guests) == 0 {
		return nil
	}
	path := base + "/" + url.PathEscape(eid) + "/attendees?user_id_type=open_id"
	if _, err := feishuCall(client, http.MethodPost, path, token, map[string]any{"attendees": guests, "need_notification": true}); err != nil {
		return writtenError{fmt.Errorf("connector: 飞书日程已经写好（event_id %s），但参会人没加上：%s。不要重新建，改这个日程时再加参会人", eid, strings.TrimPrefix(err.Error(), "connector: "))}
	}
	return nil
}
