// calendar_pick.go 让 CalDAV 和飞书日历的账号选用哪一个日历：列出这个账号能放日程的日历，记下人选的那个。
// 没选时照旧自动：CalDAV 用查找到的第一个能放日程的日历，飞书用账号上填的 calendar_id 或主日历。
// 只能选列出来的那几个。CalDAV 选的日历地址读写时还会再核对一次，必须和账号的服务器是同一家（不把密码发到别处）。
// 不拥有：日程的读写（caldav.go、feishu_calendar.go）、查找日历主目录（caldav_discover.go）。

package connector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
)

// CalendarChoice 是一个可以选的日历。ID 是 CalDAV 的日历集合地址或飞书的 calendar_id。
type CalendarChoice struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Primary bool   `json:"primary,omitempty"`
	// Current 是现在读写用的那个：人选的，或者没选时自动用的。
	Current bool `json:"current,omitempty"`
}

// calendarPickable 在这家日历可以在界面里选日历时为真。Google、Outlook 读写主日历，草稿里可以另写 calendar_id。
func calendarPickable(provider string) bool {
	p := strings.ToLower(strings.TrimSpace(provider))
	return isCalDAV(p) || isFeishuCalendar(p)
}

// calendarEndpoint 是读写日程时交给 CalDAV / 飞书那一层的地址：选过日历就用选的那个，否则用账号上填的。
// CalDAV 选的日历不在账号服务器那一家时不用它（connectors.json 被改过也不会把密码发到别处）；
// 账号地址里写了用户名（令牌只填密码的写法）时带到选的地址上。
func calendarEndpoint(a Account) string {
	cal := strings.TrimSpace(a.Calendar)
	if cal == "" {
		return a.Endpoint
	}
	if !isCalDAV(strings.ToLower(a.Provider)) {
		return cal
	}
	base, _, err := caldavCollection(a.Endpoint)
	c, _, cerr := caldavCollection(cal)
	if err != nil || cerr != nil || !sameSite(base.Hostname(), c.Hostname()) {
		return a.Endpoint
	}
	if u, err := url.Parse(strings.TrimSpace(a.Endpoint)); err == nil && u.User != nil {
		c.User = url.User(u.User.Username())
	}
	return c.String()
}

// Calendars 列出这个账号能放日程的日历，标出现在用的那个。只有 CalDAV 和飞书日历能选。
// 令牌快过期时先换一张；被拒时换一张再试一次（只是读，重试不会多写东西）。
func (b *Broker) Calendars(id string) ([]CalendarChoice, error) {
	b.mu.Lock()
	a, ok := b.find(id)
	b.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: 没有账号 %s", id)
	}
	if !calendarPickable(a.Provider) {
		return nil, fmt.Errorf("connector: %s 不用选日历", a.Provider)
	}
	tok, err := b.freshToken(a, false)
	if err != nil {
		return nil, err
	}
	list, err := b.calendarsWith(a, tok)
	if err != nil && authExpired(err) && tok.RefreshToken != "" {
		again, rerr := b.freshToken(a, true)
		if rerr != nil {
			return nil, rerr
		}
		list, err = b.calendarsWith(a, again)
	}
	return list, err
}

// calendarsWith 用已经拿到的令牌列一次日历，并标出现在用的那个。
func (b *Broker) calendarsWith(a Account, tok Token) ([]CalendarChoice, error) {
	if isCalDAV(strings.ToLower(a.Provider)) {
		list, err := caldavCalendars(b.client(), a.Endpoint, tok.AccessToken)
		if err != nil {
			return nil, err
		}
		cur := strings.TrimSpace(a.Calendar)
		if cur == "" {
			if coll, _, err := caldavResolve(b.client(), a.Endpoint, tok.AccessToken); err == nil {
				cur = coll.String()
			}
		}
		markCurrent(list, cur)
		return list, nil
	}
	list, err := feishuCalendars(b.client(), tok.AccessToken)
	if err != nil {
		return nil, err
	}
	cur := strings.TrimSpace(a.Calendar)
	if cur == "" {
		cur = strings.TrimSpace(a.Endpoint)
	}
	if cur == "" {
		for _, c := range list {
			if c.Primary {
				cur = c.ID
				break
			}
		}
	}
	markCurrent(list, cur)
	return list, nil
}

// markCurrent 把 ID 是 cur 的那个标成现在用的（CalDAV 地址末尾的 / 不算差别）。
func markCurrent(list []CalendarChoice, cur string) {
	key := strings.TrimSuffix(strings.TrimSpace(cur), "/")
	for i := range list {
		list[i].Current = key != "" && strings.TrimSuffix(list[i].ID, "/") == key
	}
}

// ChooseCalendar 记下这个账号以后读写哪一个日历，写回 connectors.json。cal 为空时回到自动选。
// cal 必须是 Calendars 列出来的那几个之一；名字一并记下，界面上直接显示。
func (b *Broker) ChooseCalendar(id, cal string) (Account, error) {
	cal = strings.TrimSpace(cal)
	name := ""
	if cal != "" {
		list, err := b.Calendars(id)
		if err != nil {
			return Account{}, err
		}
		found := false
		for _, c := range list {
			if strings.TrimSuffix(c.ID, "/") == strings.TrimSuffix(cal, "/") {
				cal, name, found = c.ID, c.Name, true
				break
			}
		}
		if !found {
			return Account{}, fmt.Errorf("connector: 这个账号下没有日历 %s，先重新列一遍再选", cal)
		}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.acct {
		if b.acct[i].ID != id {
			continue
		}
		if !calendarPickable(b.acct[i].Provider) {
			return b.acct[i], fmt.Errorf("connector: %s 不用选日历", b.acct[i].Provider)
		}
		b.acct[i].Calendar, b.acct[i].CalendarName = cal, name
		return b.acct[i], b.flush()
	}
	return Account{}, fmt.Errorf("connector: 没有账号 %s", id)
}

// davCalendar 是日历主目录下的一个能放日程的日历。
type davCalendar struct {
	url  *url.URL
	name string
}

// caldavCollections 在日历主目录下（Depth: 1）按服务器给的顺序列出能放日程的日历。主目录自己、待办日历、不是同一家主机的都跳过。
// 名字取 displayname，没有时用地址最后一段。第二个返回值是最后问的那个地址（跟过跳转之后），报错时用。
func caldavCollections(client *http.Client, home *url.URL, urlUser, token string) ([]davCalendar, *url.URL, error) {
	final, code, ms, err := davPropfind(client, home, urlUser, token, "1")
	if err != nil {
		return nil, home, err
	}
	if code == http.StatusUnauthorized || code == http.StatusForbidden {
		return nil, final, caldavStatus(code, nil, "discover")
	}
	if code != http.StatusMultiStatus {
		return nil, final, nil
	}
	var out []davCalendar
	seen := map[string]bool{}
	for _, r := range ms.Responses {
		u, err := davRef(final, r.Href)
		if err != nil || strings.TrimSuffix(u.Path, "/") == strings.TrimSuffix(final.Path, "/") {
			continue
		}
		if !strings.HasSuffix(u.Path, "/") {
			u.Path += "/"
			u.RawPath = ""
		}
		for _, ps := range r.Propstat {
			if !davOK(ps.Status) || !ps.Prop.isCalendar() || !ps.Prop.holdsEvents() || seen[u.String()] {
				continue
			}
			seen[u.String()] = true
			name := strings.TrimSpace(ps.Prop.DisplayName)
			if name == "" {
				name = path.Base(strings.TrimSuffix(u.Path, "/"))
			}
			out = append(out, davCalendar{url: u, name: name})
		}
	}
	return out, final, nil
}

// caldavCalendars 列出账号所在日历主目录下能放日程的日历。
// 账号上填的就是一个日历集合、服务器又不告诉主目录时，把它的上一级当主目录；上一级什么都没列出来时至少给出它自己。
func caldavCalendars(client *http.Client, endpoint, token string) ([]CalendarChoice, error) {
	base, urlUser, err := caldavCollection(endpoint)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("connector: no token")
	}
	f, err := caldavDiscover(client, base, urlUser, token)
	if err != nil {
		return nil, err
	}
	home := f.home
	if home == nil && f.coll != nil {
		if dir := path.Dir(strings.TrimSuffix(f.coll.Path, "/")); dir != "." && dir != "/" {
			up := *f.coll
			up.Path, up.RawPath = dir+"/", ""
			home = &up
		}
	}
	var out []CalendarChoice
	if home != nil {
		cals, _, err := caldavCollections(client, home, urlUser, token)
		if err != nil {
			return nil, err
		}
		for _, c := range cals {
			out = append(out, CalendarChoice{ID: c.url.String(), Name: c.name})
		}
	}
	if len(out) == 0 && f.coll != nil {
		out = append(out, CalendarChoice{ID: f.coll.String(), Name: path.Base(strings.TrimSuffix(f.coll.Path, "/"))})
	}
	return out, nil
}

// feishuCalendars 列出这个人能写的飞书日历：自己的和有编辑权限的共享日历，主日历排第一。会议室、已删除和只读的日历不给。
// 最多翻五页（每页两百个）。
func feishuCalendars(client *http.Client, token string) ([]CalendarChoice, error) {
	var out []CalendarChoice
	pageToken := ""
	for page := 0; page < 5; page++ {
		q := url.Values{"page_size": {"200"}}
		if pageToken != "" {
			q.Set("page_token", pageToken)
		}
		raw, err := feishuCall(client, http.MethodGet, "/open-apis/calendar/v4/calendars?"+q.Encode(), token, nil)
		if err != nil {
			return nil, err
		}
		var doc struct {
			Data struct {
				HasMore   bool   `json:"has_more"`
				PageToken string `json:"page_token"`
				List      []struct {
					ID        string `json:"calendar_id"`
					Summary   string `json:"summary"`
					Type      string `json:"type"`
					Role      string `json:"role"`
					IsDeleted bool   `json:"is_deleted"`
				} `json:"calendar_list"`
			} `json:"data"`
		}
		_ = json.Unmarshal(raw, &doc)
		for _, c := range doc.Data.List {
			role := strings.ToLower(strings.TrimSpace(c.Role))
			if c.IsDeleted || strings.TrimSpace(c.ID) == "" || strings.EqualFold(c.Type, "resource") || (role != "owner" && role != "writer") {
				continue
			}
			primary := strings.EqualFold(c.Type, "primary")
			name := strings.TrimSpace(c.Summary)
			if name == "" && primary {
				name = "主日历"
			}
			if name == "" {
				name = c.ID
			}
			out = append(out, CalendarChoice{ID: c.ID, Name: name, Primary: primary})
		}
		if !doc.Data.HasMore || doc.Data.PageToken == "" {
			break
		}
		pageToken = doc.Data.PageToken
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Primary && !out[j].Primary })
	return out, nil
}
