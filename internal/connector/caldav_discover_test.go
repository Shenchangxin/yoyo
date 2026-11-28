package connector

import (
	"net/http"
	"strings"
	"testing"
)

// caldavSelfCalendar 是「问的这个地址本身就是日历」的 PROPFIND 答复。
const caldavSelfCalendar = `<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav"><d:response><d:href>./</d:href><d:propstat><d:prop>` +
	`<d:resourcetype><d:collection/><c:calendar/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`

// davMS 把几个 response 包成 207 答复。
func davMS(responses ...string) string {
	return `<?xml version="1.0" encoding="utf-8"?><d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">` +
		strings.Join(responses, "") + `</d:multistatus>`
}

// davResp 是一个资源的 response，props 是 <d:prop> 里面的内容，状态是 200。
func davResp(href, props string) string {
	return `<d:response><d:href>` + href + `</d:href><d:propstat><d:prop>` + props + `</d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`
}

// TestCalDAVDiscoversFromServerRoot 模拟 iCloud：填的是服务器根地址，主体在根上给出，日历主目录在另一台同一家的主机上；
// 主目录里先列出一个只放待办的日历，要跳过它，挑能放日程的那个；之后的读日历直接用记下的集合，不再查找。
func TestCalDAVDiscoversFromServerRoot(t *testing.T) {
	var propfinds []string
	report := ""
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if user, pass, ok := r.BasicAuth(); !ok || user != "ada" || pass != "pw" {
			t.Errorf("basic auth = %q %q %v", user, pass, ok)
		}
		switch r.Method {
		case "PROPFIND":
			propfinds = append(propfinds, r.URL.String()+" depth="+r.Header.Get("Depth"))
			switch r.URL.String() {
			case "https://caldav.example.com/":
				return jsonResp(207, davMS(davResp("/", `<d:resourcetype><d:collection/></d:resourcetype><d:current-user-principal><d:href>/123/principal/</d:href></d:current-user-principal>`))), nil
			case "https://caldav.example.com/123/principal/":
				return jsonResp(207, davMS(davResp("/123/principal/", `<c:calendar-home-set><d:href>https://p52-caldav.example.com:443/123/calendars/</d:href></c:calendar-home-set>`))), nil
			case "https://p52-caldav.example.com:443/123/calendars/":
				return jsonResp(207, davMS(
					davResp("/123/calendars/", `<d:resourcetype><d:collection/></d:resourcetype>`),
					davResp("/123/calendars/tasks/", `<d:resourcetype><d:collection/><c:calendar/></d:resourcetype><c:supported-calendar-component-set><c:comp name="VTODO"/></c:supported-calendar-component-set>`),
					davResp("/123/calendars/home/", `<d:resourcetype><d:collection/><c:calendar/></d:resourcetype><c:supported-calendar-component-set><c:comp name="VEVENT"/></c:supported-calendar-component-set>`),
				)), nil
			}
			return jsonResp(404, ""), nil
		case "REPORT":
			report = r.URL.String()
			return jsonResp(207, caldavReport), nil
		}
		return jsonResp(500, ""), nil
	})}
	rows, err := caldavList(client, "https://caldav.example.com", "ada:pw", "")
	if err != nil {
		t.Fatal(err)
	}
	if report != "https://p52-caldav.example.com:443/123/calendars/home/" || len(rows) != 2 {
		t.Fatalf("report at %q, rows = %+v", report, rows)
	}
	if len(propfinds) != 3 || !strings.HasSuffix(propfinds[2], "depth=1") {
		t.Fatalf("propfinds = %q", propfinds)
	}
	if _, err := caldavList(client, "https://caldav.example.com/", "ada:pw", ""); err != nil || len(propfinds) != 3 {
		t.Fatalf("second read must reuse the found collection: %v %q", err, propfinds)
	}
}

// TestCalDAVWriteUsesDiscoveredCollection 确认只填服务器地址时，新建日程 PUT 到查找到的日历集合里，不写到服务器根目录。
func TestCalDAVWriteUsesDiscoveredCollection(t *testing.T) {
	put := ""
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodPut:
			put = r.URL.String()
			return jsonResp(201, ""), nil
		case r.Method != "PROPFIND":
			return jsonResp(500, ""), nil
		case r.URL.String() == "https://write.example.com/":
			return jsonResp(207, davMS(davResp("/", `<c:calendar-home-set><d:href>/cal/ada/</d:href></c:calendar-home-set>`))), nil
		case r.URL.String() == "https://write.example.com/cal/ada/":
			return jsonResp(207, davMS(
				davResp("/cal/ada/", `<d:resourcetype><d:collection/></d:resourcetype>`),
				davResp("/cal/ada/work/", `<d:resourcetype><d:collection/><c:calendar/></d:resourcetype>`),
			)), nil
		}
		return jsonResp(404, ""), nil
	})}
	if err := caldavWrite(client, "https://write.example.com/", "ada:pw", Draft{Subject: "评审", Start: "2026-11-30T02:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(put, "https://write.example.com/cal/ada/work/") || !strings.HasSuffix(put, ".ics") {
		t.Fatalf("put at %q", put)
	}
}

// TestCalDAVDiscoversViaWellKnown 模拟 Nextcloud：根地址什么都不给，/.well-known/caldav 跳到 DAV 根，
// DAV 根直接给出日历主目录；主目录里的日历没声明支持的组件，按能放日程算。
func TestCalDAVDiscoversViaWellKnown(t *testing.T) {
	report := ""
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == "REPORT" {
			report = r.URL.String()
			return jsonResp(207, caldavReport), nil
		}
		switch r.URL.String() {
		case "https://dav.example.org/":
			return jsonResp(207, davMS(davResp("/", `<d:resourcetype><d:collection/></d:resourcetype>`))), nil
		case "https://dav.example.org/.well-known/caldav":
			resp := jsonResp(301, "")
			resp.Header.Set("Location", "/remote.php/dav/")
			return resp, nil
		case "https://dav.example.org/remote.php/dav/":
			return jsonResp(207, davMS(davResp("/remote.php/dav/", `<c:calendar-home-set><d:href>/remote.php/dav/calendars/ada/</d:href></c:calendar-home-set>`))), nil
		case "https://dav.example.org/remote.php/dav/calendars/ada/":
			if r.Header.Get("Depth") != "1" {
				t.Errorf("home listing depth = %q", r.Header.Get("Depth"))
			}
			return jsonResp(207, davMS(
				davResp("/remote.php/dav/calendars/ada/", `<d:resourcetype><d:collection/></d:resourcetype>`),
				davResp("/remote.php/dav/calendars/ada/personal", `<d:resourcetype><d:collection/><c:calendar/></d:resourcetype>`),
			)), nil
		}
		return jsonResp(404, ""), nil
	})}
	if _, err := caldavList(client, "https://dav.example.org", "ada:pw", ""); err != nil {
		t.Fatal(err)
	}
	if report != "https://dav.example.org/remote.php/dav/calendars/ada/personal/" {
		t.Fatalf("report at %q", report)
	}
}

// TestCalDAVDiscoveryStaysOnTheSameSite 确认服务器把日历主目录指到别家主机时不跟过去（不把密码发出去）；
// 什么都找不到时说清可以直接填集合地址。
func TestCalDAVDiscoveryStaysOnTheSameSite(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Hostname() == "evil.test" {
			t.Errorf("credentials sent to %s", r.URL)
		}
		if r.URL.Hostname() == "dav2.example.org" {
			return jsonResp(207, davMS(davResp("/", `<c:calendar-home-set><d:href>https://evil.test/cal/</d:href></c:calendar-home-set>`))), nil
		}
		// empty.example.net：只是个普通目录，没有主体、没有主目录、下面也没有日历。
		if r.URL.Path == "/.well-known/caldav" {
			return jsonResp(404, ""), nil
		}
		return jsonResp(207, davMS(davResp(r.URL.Path, `<d:resourcetype><d:collection/></d:resourcetype>`))), nil
	})}
	if _, err := caldavList(client, "https://dav2.example.org/", "ada:pw", ""); err == nil || !strings.Contains(err.Error(), "evil.test") {
		t.Fatalf("foreign home err = %v", err)
	}
	if _, err := caldavList(client, "https://empty.example.net/", "ada:pw", ""); err == nil || !strings.Contains(err.Error(), "没找到") {
		t.Fatalf("nothing found err = %v", err)
	}
	if !sameSite("caldav.icloud.com", "p52-caldav.icloud.com") || sameSite("caldav.icloud.com", "icloud.com.evil.test") {
		t.Fatal("sameSite")
	}
}
