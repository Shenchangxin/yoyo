// caldav_discover.go 让 CalDAV 账号上填服务器地址（https://caldav.icloud.com/）、主体地址或日历主目录地址也能用，
// 不必自己去找日历集合的完整地址。按 RFC 4791 / 6638 / 6764 的顺序找：
//  1. PROPFIND 填的地址（Depth: 0）：它本身就是日历（resourcetype 里有 calendar）就直接用；
//     给了 calendar-home-set 就去日历主目录；给了 current-user-principal 就先去主体地址再读 calendar-home-set；
//  2. 都没有时按 RFC 6764 试 /.well-known/caldav（服务器一般跳到真正的 DAV 根）；
//  3. 在日历主目录里 PROPFIND（Depth: 1），挑第一个能放日程（supported-calendar-component-set 含 VEVENT，或没声明）的日历；
//     主目录也找不到时，把填的地址当成主目录再试一次。
//
// 跳转和服务器给出的地址只跟同一家的主机（主机名最后两段相同，如 caldav.icloud.com → p52-caldav.icloud.com），而且必须是 https：
// 不把账号密码发到别家。Go 的客户端遇到 301 / 302 会把 PROPFIND 改成 GET，所以这里自己处理跳转，最多三跳。
// 找到的集合在内存里记一小时（按地址和用户名），一次读日历不会每次都多走三四个请求；重启或过一小时后重新找。
// 不拥有：日程的读写（caldav.go）。

package connector

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// caldavFoundKeep 是找到的日历集合记多久。
const caldavFoundKeep = time.Hour

// caldavHit 是记下的一次查找结果。
type caldavHit struct {
	coll string
	// self 是查找时见到的账号本人日历地址（calendar-user-address-set 里的 mailto），写日程时用作组织者。
	self string
	at   time.Time
}

// caldavFound 是「填的地址\n用户名 → 日历集合地址」。只在内存里。
var caldavFound sync.Map

// davPropfindBody 一次问齐查找要的几项：资源类型、当前用户主体、日历主目录、支持的组件。
const davPropfindBody = `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop><d:resourcetype/><d:displayname/><d:current-user-principal/><c:calendar-home-set/><c:calendar-user-address-set/><c:supported-calendar-component-set/></d:prop>
</d:propfind>`

// davProp 是 PROPFIND 答复里一个资源的属性。encoding/xml 的标签不带命名空间时按本地名匹配，DAV: 和 caldav 两个命名空间都认。
type davProp struct {
	// ResourceType 收下 resourcetype 下的每个子元素，只看名字（collection、calendar、schedule-inbox……）。
	ResourceType struct {
		Kinds []struct {
			XMLName xml.Name
		} `xml:",any"`
	} `xml:"resourcetype"`
	Principal struct {
		Href string `xml:"href"`
	} `xml:"current-user-principal"`
	Home struct {
		Href string `xml:"href"`
	} `xml:"calendar-home-set"`
	Comps struct {
		Comp []struct {
			Name string `xml:"name,attr"`
		} `xml:"comp"`
	} `xml:"supported-calendar-component-set"`
	// DisplayName 是日历的名字，在界面里选日历时显示。
	DisplayName string `xml:"displayname"`
	// Addr 是主体上的 calendar-user-address-set：这个人的日历地址，mailto: 那个用作写日程时的 ORGANIZER。
	Addr struct {
		Hrefs []string `xml:"href"`
	} `xml:"calendar-user-address-set"`
}

// mailto 是 calendar-user-address-set 里第一个 mailto: 地址（不带 mailto:）。没有时为空。
func (p davProp) mailto() string {
	for _, h := range p.Addr.Hrefs {
		h = strings.TrimSpace(h)
		if len(h) > len("mailto:") && strings.EqualFold(h[:len("mailto:")], "mailto:") {
			return strings.TrimSpace(h[len("mailto:"):])
		}
	}
	return ""
}

// davProps 是 PROPFIND 的 207 答复。
type davProps struct {
	Responses []struct {
		Href     string `xml:"href"`
		Propstat []struct {
			Prop   davProp `xml:"prop"`
			Status string  `xml:"status"`
		} `xml:"propstat"`
	} `xml:"response"`
}

// isCalendar 在资源类型里有 calendar 时为真（schedule-inbox / outbox 不算）。
func (p davProp) isCalendar() bool {
	for _, k := range p.ResourceType.Kinds {
		if k.XMLName.Local == "calendar" {
			return true
		}
	}
	return false
}

// holdsEvents 在日历能放日程时为真：声明了支持的组件就要含 VEVENT（只放待办的 VTODO 日历不算）；没声明的按能放算。
func (p davProp) holdsEvents() bool {
	if len(p.Comps.Comp) == 0 {
		return true
	}
	for _, c := range p.Comps.Comp {
		if strings.EqualFold(c.Name, "VEVENT") {
			return true
		}
	}
	return false
}

// davOK 在 propstat 的状态是 200（或没写）时为真；404 的那一组是服务器说「这个属性没有」。
func davOK(status string) bool {
	return status == "" || strings.Contains(status, " 200")
}

// sameSite 在两个主机名是同一家时为真：完全相同，或者最后两段相同（caldav.icloud.com 和 p52-caldav.icloud.com）。
func sameSite(a, b string) bool {
	a, b = strings.ToLower(strings.TrimSuffix(a, ".")), strings.ToLower(strings.TrimSuffix(b, "."))
	if a == b {
		return true
	}
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	if len(pa) < 2 || len(pb) < 2 {
		return false
	}
	return pa[len(pa)-2] == pb[len(pb)-2] && pa[len(pa)-1] == pb[len(pb)-1]
}

// davRef 把服务器给的地址（绝对地址或路径）按 base 解析。只认 https、同一家主机；去掉地址里的用户名、查询串和片段。
func davRef(base *url.URL, href string) (*url.URL, error) {
	ref, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return nil, fmt.Errorf("connector: 日历服务器给的地址 %q 不能用", href)
	}
	t := base.ResolveReference(ref)
	if !strings.EqualFold(t.Scheme, "https") {
		return nil, fmt.Errorf("connector: 日历服务器把地址指到了非 https 的 %s，没有跟过去", t.Host)
	}
	if !sameSite(base.Hostname(), t.Hostname()) {
		return nil, fmt.Errorf("connector: 日历服务器把地址指到了 %s，和 %s 不是同一家，没有跟过去", t.Host, base.Host)
	}
	t.User = nil
	t.RawQuery, t.Fragment = "", ""
	return t, nil
}

// davPropfind 发一次 PROPFIND，自己跟跳转（最多三跳，每一跳都过 davRef）。返回最后问的那个地址、状态码和解析好的 207 答复。
// 不是 207 时答复为空、不报错，由调用方按状态码决定；网络错误和 207 解析失败才报错。
func davPropfind(client *http.Client, target *url.URL, urlUser, token, depth string) (*url.URL, int, davProps, error) {
	if client == nil {
		client = http.DefaultClient
	}
	// 复制一份客户端，只改跳转规则：别的设置（传输、超时）照旧。
	nc := *client
	nc.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	user, pass := caldavCreds(token, urlUser)
	cur := target
	for hop := 0; ; hop++ {
		req, err := http.NewRequest("PROPFIND", cur.String(), strings.NewReader(davPropfindBody))
		if err != nil {
			return cur, 0, davProps{}, err
		}
		req.SetBasicAuth(user, pass)
		req.Header.Set("Depth", depth)
		req.Header.Set("Content-Type", `application/xml; charset="utf-8"`)
		resp, err := nc.Do(req)
		if err != nil {
			return cur, 0, davProps{}, fmt.Errorf("connector: %s", err.Error())
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
			loc := resp.Header.Get("Location")
			if loc == "" || hop >= 3 {
				return cur, resp.StatusCode, davProps{}, nil
			}
			next, err := davRef(cur, loc)
			if err != nil {
				return cur, 0, davProps{}, err
			}
			cur = next
			continue
		}
		var ms davProps
		if resp.StatusCode == http.StatusMultiStatus {
			if err := xml.Unmarshal(raw, &ms); err != nil {
				return cur, resp.StatusCode, davProps{}, fmt.Errorf("connector: caldav discover: %s 的答复不是 multistatus", cur.Host)
			}
		}
		return cur, resp.StatusCode, ms, nil
	}
}

// caldavResolve 把账号上填的地址变成日历集合地址：先看一小时内找过的，没有就 caldavDiscover 现找。
// 返回的集合地址不带用户名；第二个返回值是地址里写的用户名（令牌里没写用户名时用它）。没有令牌时不发请求。
// 找的时候顺手记下账号本人的日历地址（caldavSelf 读），写日程时用作组织者。
func caldavResolve(client *http.Client, endpoint, token string) (*url.URL, string, error) {
	base, urlUser, err := caldavCollection(endpoint)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(token) == "" {
		return nil, "", fmt.Errorf("connector: no token")
	}
	key := caldavKey(base, urlUser, token)
	if v, ok := caldavFound.Load(key); ok {
		if hit := v.(caldavHit); time.Since(hit.at) < caldavFoundKeep {
			if u, err := url.Parse(hit.coll); err == nil {
				return u, urlUser, nil
			}
		}
	}
	f, err := caldavDiscover(client, base, urlUser, token)
	if err != nil {
		return nil, "", err
	}
	caldavFound.Store(key, caldavHit{coll: f.coll.String(), self: f.self, at: time.Now()})
	return f.coll, urlUser, nil
}

// caldavKey 是 caldavFound 里的键：填的地址和登录名。
func caldavKey(base *url.URL, urlUser, token string) string {
	user, _ := caldavCreds(token, urlUser)
	return base.String() + "\n" + user
}

// caldavSelf 是一小时内查找日历时记下的账号本人日历地址（mailto: 后面那段）。没找过、或者服务器没给时为空。不发请求。
func caldavSelf(endpoint, token string) string {
	base, urlUser, err := caldavCollection(endpoint)
	if err != nil {
		return ""
	}
	if v, ok := caldavFound.Load(caldavKey(base, urlUser, token)); ok {
		if hit := v.(caldavHit); time.Since(hit.at) < caldavFoundKeep {
			return hit.self
		}
	}
	return ""
}

// davFound 是一次查找的结果：日历主目录（直接填了日历集合时为空）、挑中的日历集合，和账号本人的日历地址。
type davFound struct {
	home, coll *url.URL
	self       string
}

// caldavDiscover 按文件头的顺序找日历集合。登录被拒时说清要用应用专用密码；什么都没找到时说清可以直接填集合地址。
func caldavDiscover(client *http.Client, base *url.URL, urlUser, token string) (davFound, error) {
	var f davFound
	home, coll, self, err := caldavHome(client, base, urlUser, token, 0)
	f.self = self
	if err != nil {
		return f, err
	}
	if coll != nil {
		f.coll = coll
		return f, nil
	}
	if home == nil {
		// 填的是服务器根地址、又不回主体：按 RFC 6764 试 /.well-known/caldav，服务器一般跳到真正的 DAV 根。
		wk := base.ResolveReference(&url.URL{Path: "/.well-known/caldav"})
		var wself string
		home, coll, wself, err = caldavHome(client, wk, urlUser, token, 0)
		if f.self == "" {
			f.self = wself
		}
		if err != nil {
			return f, err
		}
		if coll != nil {
			f.coll = coll
			return f, nil
		}
	}
	if home == nil {
		// 最后把填的地址当成日历主目录：有的服务器在主目录上不回 calendar-home-set。
		home = base
	}
	f.home = home
	f.coll, err = caldavPick(client, home, urlUser, token)
	return f, err
}

// caldavHome 问 target 自己（Depth: 0）：它就是日历时返回 coll；给了日历主目录时返回 home；给了主体地址时去主体那里再问一次（最多两跳）。
// 都没有、或者地址不存在（404 / 405 等）时两个都返回 nil，由调用方换别的路找。401 / 403 报登录失败。
// self 是一路上见到的 calendar-user-address-set 里的 mailto 地址。填的就是日历集合时，它给了主体地址就多问主体一次拿这个地址。
func caldavHome(client *http.Client, target *url.URL, urlUser, token string, hops int) (home, coll *url.URL, self string, err error) {
	if hops > 2 {
		return nil, nil, "", nil
	}
	final, code, ms, err := davPropfind(client, target, urlUser, token, "0")
	if err != nil {
		return nil, nil, "", err
	}
	if code == http.StatusUnauthorized || code == http.StatusForbidden {
		return nil, nil, "", caldavStatus(code, nil, "discover")
	}
	if code != http.StatusMultiStatus {
		return nil, nil, "", nil
	}
	principal := ""
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if !davOK(ps.Status) {
				continue
			}
			p := ps.Prop
			if m := p.mailto(); m != "" && self == "" {
				self = m
			}
			if pr := strings.TrimSpace(p.Principal.Href); pr != "" && principal == "" {
				principal = pr
			}
			if p.isCalendar() {
				c := *final
				c.User = nil
				if self == "" && principal != "" && hops == 0 {
					self = caldavPrincipalMail(client, final, principal, urlUser, token)
				}
				return nil, &c, self, nil
			}
			if h := strings.TrimSpace(p.Home.Href); h != "" {
				u, err := davRef(final, h)
				return u, nil, self, err
			}
		}
	}
	if principal == "" {
		return nil, nil, self, nil
	}
	pu, err := davRef(final, principal)
	if err != nil {
		return nil, nil, self, err
	}
	// 主体地址就是刚问过的这个时不再问，免得兜圈子。
	if strings.TrimSuffix(pu.String(), "/") == strings.TrimSuffix(final.String(), "/") {
		return nil, nil, self, nil
	}
	h, c, deeper, err := caldavHome(client, pu, urlUser, token, hops+1)
	if self == "" {
		self = deeper
	}
	return h, c, self, err
}

// caldavPrincipalMail 问一次主体地址（Depth: 0），取 calendar-user-address-set 里的 mailto 地址。出错或没给时为空，不影响找日历。
func caldavPrincipalMail(client *http.Client, base *url.URL, href, urlUser, token string) string {
	pu, err := davRef(base, href)
	if err != nil {
		return ""
	}
	_, code, ms, err := davPropfind(client, pu, urlUser, token, "0")
	if err != nil || code != http.StatusMultiStatus {
		return ""
	}
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if davOK(ps.Status) {
				if m := ps.Prop.mailto(); m != "" {
					return m
				}
			}
		}
	}
	return ""
}

// caldavPick 在日历主目录下挑第一个能放日程的日历（caldavCollections 列出的第一个，按服务器给的顺序）。
func caldavPick(client *http.Client, home *url.URL, urlUser, token string) (*url.URL, error) {
	cals, final, err := caldavCollections(client, home, urlUser, token)
	if err != nil {
		return nil, err
	}
	if len(cals) > 0 {
		return cals[0].url, nil
	}
	return nil, fmt.Errorf("connector: 在 %s 下没找到能放日程的日历。可以直接填日历集合的完整地址", final.String())
}
