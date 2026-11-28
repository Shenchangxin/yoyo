// feishu_oauth.go 是飞书用户身份连接（日历、云文档、任务）的 OAuth：授权地址、各自申请的权限、换票地址，以及按供应商编出换票请求。
// 这几项都要用户身份的 user_access_token，大约两小时过期；授权时申请 offline_access 才会给 refresh_token，
// 之后 refresh.go 的 freshToken 在快过期时、或接口报 401 时用它换一张新的。飞书每次换票都给一张新的刷新令牌，旧的随即作废，
// 所以换完一定要把新的写回保险库（freshToken 已经这样做）。
// 和 Google、微软不同，飞书 v2 的换票接口只收 JSON，不收表单，所以首次换票（Exchange）和刷新（freshToken）都经 tokenRequest 编请求。
// 不拥有：日历读写（feishu_calendar.go）、云文档读写（feishu_docs.go）、任务读写（feishu_task.go）、通用的换票流程（refresh.go、live.go 的 Exchange）。

package connector

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// feishuAuthorizeURL 是飞书的用户授权页。
const feishuAuthorizeURL = "https://accounts.feishu.cn/open-apis/authen/v1/authorize"

// feishuCalendarScope 是飞书日历要申请的权限：日历读写，加 offline_access 拿刷新令牌。
const feishuCalendarScope = "calendar:calendar offline_access"

// feishuDocsScope 是飞书云文档要申请的权限：新版文档读写、云空间只读（列文件夹、搜索），加 offline_access。
// 搜索接口没开权限时，读取退回列文件夹再按名字筛（feishu_docs.go），所以这里不强求搜索权限。
const feishuDocsScope = "docx:document drive:drive:readonly offline_access"

// feishuTaskScope 是飞书任务要申请的权限：任务读写，加 offline_access。
const feishuTaskScope = "task:task:read task:task:write offline_access"

// isFeishuUser 在供应商是用用户身份授权的飞书连接（日历、云文档、任务）时为真。它们共用授权页、换票地址和 JSON 换票。
// provider 要已经转成小写、去掉空白。
func isFeishuUser(provider string) bool {
	return isFeishuCalendar(provider) || isFeishuDocs(provider) || isFeishuTask(provider)
}

// feishuScope 是这一项飞书连接授权时申请的权限。认不出的按日历算。
func feishuScope(provider string) string {
	switch {
	case isFeishuDocs(provider):
		return feishuDocsScope
	case isFeishuTask(provider):
		return feishuTaskScope
	default:
		return feishuCalendarScope
	}
}

// feishuTokenURL 是飞书 v2 的换票地址（授权码换票和刷新共用）。跟着 feishuOpenBase 走，测试可以一起换掉。
func feishuTokenURL() string {
	return feishuOpenBase + "/open-apis/authen/v2/oauth/token"
}

// feishuAuthURL 拼出飞书日历、云文档或任务的授权地址，权限按 provider 选（feishuScope）。
// clientID 是飞书应用的 App ID，redirect 要和应用后台「安全设置」里登记的重定向地址一致。
func feishuAuthURL(provider, clientID, redirect, state string) string {
	q := url.Values{
		"client_id":     {clientID},
		"redirect_uri":  {redirect},
		"response_type": {"code"},
		"scope":         {feishuScope(strings.ToLower(strings.TrimSpace(provider)))},
		"state":         {state},
	}
	return feishuAuthorizeURL + "?" + q.Encode()
}

// tokenRequest 按供应商把换票参数编成 POST 请求：飞书用户身份连接编成 JSON（空值不放进去），其余供应商照旧编成表单。
// provider 不区分大小写。form 里放 grant_type、code / refresh_token、client_id、client_secret、redirect_uri 这几项。
func tokenRequest(provider, endpoint string, form url.Values) (*http.Request, error) {
	if isFeishuUser(strings.ToLower(strings.TrimSpace(provider))) {
		body := map[string]string{}
		for k, v := range form {
			if len(v) > 0 && strings.TrimSpace(v[0]) != "" {
				body[k] = v[0]
			}
		}
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		return req, nil
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}
