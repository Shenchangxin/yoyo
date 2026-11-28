package connector

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// feishuHit 是假飞书收到的一个请求。
type feishuHit struct {
	method string
	path   string
	query  url.Values
	body   map[string]any
}

// feishuStub 换上一个假的飞书并连好一个账号（令牌 u1）：handle 按请求给答复正文，所有请求按先后记下。
// 令牌不是 u1 时一律回令牌过期。
func feishuStub(t *testing.T, provider string, kind Kind, endpoint string, handle func(feishuHit) string) (*Broker, Account, *[]feishuHit) {
	t.Helper()
	prev := feishuOpenBase
	t.Cleanup(func() { feishuOpenBase = prev })
	feishuOpenBase = "https://feishu.test"
	hits := &[]feishuHit{}
	b, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b.SetTokens(memTokens{})
	b.SetHTTP(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		h := feishuHit{method: r.Method, path: r.URL.Path, query: r.URL.Query()}
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &h.body)
		}
		*hits = append(*hits, h)
		if r.Header.Get("Authorization") != "Bearer u1" {
			return jsonResp(200, `{"code":99991677,"msg":"token expired"}`), nil
		}
		return jsonResp(200, handle(h)), nil
	})})
	acct := b.Connect(Account{Provider: provider, Kind: kind, Label: provider, Endpoint: endpoint})
	if err := b.SaveToken(acct.VaultKey, Token{AccessToken: "u1"}); err != nil {
		t.Fatal(err)
	}
	return b, acct, hits
}

// feishuSend 存一份草稿再发出去。
func feishuSend(t *testing.T, b *Broker, d Draft) error {
	t.Helper()
	d = b.PutDraft(d)
	_, err := b.Send(d.ID)
	return err
}

// TestFeishuDocsAndTaskOAuth 确认飞书云文档和任务各自申请自己的权限（都带 offline_access），换票和刷新都走飞书 v2 的 JSON 换票，
// 目录里标成能直接用的 OAuth 连接，种类分别是 drive 和 task。
func TestFeishuDocsAndTaskOAuth(t *testing.T) {
	b, _ := Open(t.TempDir())
	for provider, scope := range map[string]string{"feishu-docs": "docx:document", "feishu-task": "task:task:write", "feishu-calendar": "calendar:calendar"} {
		raw, state, err := b.AuthURL(provider, "cli_a1", "http://127.0.0.1:3080/oauth")
		if err != nil {
			t.Fatalf("%s: %v", provider, err)
		}
		u, _ := url.Parse(raw)
		q := u.Query()
		if !strings.HasPrefix(raw, feishuAuthorizeURL+"?") || q.Get("state") != state || !strings.Contains(q.Get("scope"), scope) || !strings.Contains(q.Get("scope"), "offline_access") {
			t.Errorf("%s auth url = %s", provider, raw)
		}
		if tokenEndpoint(provider) != feishuTokenURL() {
			t.Errorf("%s token endpoint = %q", provider, tokenEndpoint(provider))
		}
		req, err := tokenRequest(provider, feishuTokenURL(), url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"r1"}})
		if err != nil || !strings.HasPrefix(req.Header.Get("Content-Type"), "application/json") {
			t.Errorf("%s token request is not JSON: %v", provider, err)
		}
	}
	kinds := map[string]Kind{"feishu-docs": KindDrive, "feishu-task": KindTask}
	for _, e := range Catalog() {
		k, ok := kinds[e.Provider]
		if !ok {
			continue
		}
		if e.Kind != k || e.Access != "oauth" || e.Ready != "live" {
			t.Errorf("%s catalog = %+v", e.Provider, e)
		}
		delete(kinds, e.Provider)
	}
	if len(kinds) != 0 {
		t.Fatalf("missing from catalog: %v", kinds)
	}
}

// TestFeishuTaskReadAndWrite 确认飞书任务：默认只读没完成的、有截止时间的排前面、查询按标题和描述筛、「全部」连已完成的一起读；
// 新建带全天截止时间，完成、修改、删除都打到那条任务上；截止时间认不出或缺标题时一个请求都不发。
func TestFeishuTaskReadAndWrite(t *testing.T) {
	b, acct, hits := feishuStub(t, "feishu-task", KindTask, "", func(h feishuHit) string {
		switch {
		case h.method == http.MethodGet && h.path == "/open-apis/task/v2/tasks":
			return `{"code":0,"data":{"has_more":false,"items":[` +
				`{"guid":"g1","summary":"买菜","description":"牛奶和鸡蛋","completed_at":"0"},` +
				`{"guid":"g2","summary":"交周报","due":{"timestamp":"1796137200000","is_all_day":false},"completed_at":"0","url":"https://applink.feishu.cn/t/g2"}]}}`
		case h.method == http.MethodPost && h.path == "/open-apis/task/v2/tasks":
			return `{"code":0,"data":{"task":{"guid":"g9"}}}`
		}
		return `{"code":0,"data":{}}`
	})
	last := func() feishuHit { return (*hits)[len(*hits)-1] }

	rows, err := b.Read(acct.ID, "")
	if err != nil || len(rows) != 2 {
		t.Fatalf("read = %+v %v", rows, err)
	}
	if rows[0]["id"] != "g2" || rows[0]["due"] != time.UnixMilli(1796137200000).UTC().Format(time.RFC3339) || rows[0]["url"] == "" {
		t.Fatalf("task with a due date first: %+v", rows[0])
	}
	if rows[1]["title"] != "买菜" || rows[1]["completed"] != "false" || rows[1]["due"] != "" {
		t.Fatalf("second task = %+v", rows[1])
	}
	if q := (*hits)[0].query; q.Get("type") != "my_tasks" || q.Get("completed") != "false" {
		t.Fatalf("list query = %v", q)
	}
	if rows, _ = b.Read(acct.ID, "鸡蛋"); len(rows) != 1 || rows[0]["id"] != "g1" {
		t.Fatalf("filtered read = %+v", rows)
	}
	if _, _ = b.Read(acct.ID, "全部"); last().query.Get("completed") != "" {
		t.Fatalf("「全部」 should include finished tasks: %v", last().query)
	}

	if err := feishuSend(t, b, Draft{Account: acct.ID, Subject: "订机票", Body: "春节回家", End: "2026-12-20"}); err != nil {
		t.Fatal(err)
	}
	due, _ := last().body["due"].(map[string]any)
	if last().method != http.MethodPost || last().path != "/open-apis/task/v2/tasks" || last().body["summary"] != "订机票" ||
		last().body["description"] != "春节回家" || due["is_all_day"] != true || due["timestamp"] == "" {
		t.Fatalf("create = %+v", last())
	}

	if err := feishuSend(t, b, Draft{Account: acct.ID, ItemID: "g2", Op: "complete"}); err != nil {
		t.Fatal(err)
	}
	task, _ := last().body["task"].(map[string]any)
	if last().method != http.MethodPatch || last().path != "/open-apis/task/v2/tasks/g2" || task["completed_at"] == nil || task["completed_at"] == "0" ||
		!strings.Contains(fmt.Sprint(last().body["update_fields"]), "completed_at") {
		t.Fatalf("complete = %+v", last())
	}

	if err := feishuSend(t, b, Draft{Account: acct.ID, ItemID: "g2", Subject: "交周报（改）", End: "2026-12-01T18:00:00+08:00"}); err != nil {
		t.Fatal(err)
	}
	task, _ = last().body["task"].(map[string]any)
	due, _ = task["due"].(map[string]any)
	want := fmt.Sprint(time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC).UnixMilli())
	fields := fmt.Sprint(last().body["update_fields"])
	if last().method != http.MethodPatch || task["summary"] != "交周报（改）" || due["timestamp"] != want ||
		!strings.Contains(fields, "summary") || !strings.Contains(fields, "due") || strings.Contains(fields, "description") {
		t.Fatalf("update = %+v", last())
	}

	if err := feishuSend(t, b, Draft{Account: acct.ID, EventID: "g1", Op: "delete"}); err != nil {
		t.Fatal(err)
	}
	if last().method != http.MethodDelete || last().path != "/open-apis/task/v2/tasks/g1" {
		t.Fatalf("delete = %+v", last())
	}

	n := len(*hits)
	if err := feishuSend(t, b, Draft{Account: acct.ID, Subject: "x", End: "下周"}); err == nil {
		t.Fatal("an unreadable due date must be refused")
	}
	if err := feishuSend(t, b, Draft{Account: acct.ID}); err == nil {
		t.Fatal("a task without a title must be refused")
	}
	if len(*hits) != n {
		t.Fatalf("refused drafts must not call Feishu: %+v", (*hits)[n:])
	}
}

// TestFeishuDocsReadAndWrite 确认飞书云文档：列文件夹、搜索没开权限时退回按名字筛、按 doc:<id> 或链接读正文；
// 新建文档写进账号的文件夹，正文按行写成标题、列表和段落；建好但正文没写上时不重发；追加打到那篇文档；不删文档。
func TestFeishuDocsReadAndWrite(t *testing.T) {
	failBlocks := false
	b, acct, hits := feishuStub(t, "feishu-docs", KindDrive, "fld1", func(h feishuHit) string {
		switch {
		case h.method == http.MethodGet && h.path == "/open-apis/drive/v1/files":
			return `{"code":0,"data":{"has_more":false,"files":[` +
				`{"token":"d1","name":"周报模板","type":"docx","url":"https://x.feishu.cn/docx/d1","modified_time":"1796000000"},` +
				`{"token":"s1","name":"预算","type":"sheet","url":"https://x.feishu.cn/sheets/s1"}]}}`
		case h.path == "/open-apis/suite/docs-api/search/object":
			return `{"code":99991672,"msg":"app has no permission"}`
		case h.method == http.MethodGet && h.path == "/open-apis/docx/v1/documents/d1":
			return `{"code":0,"data":{"document":{"document_id":"d1","title":"周报模板"}}}`
		case h.method == http.MethodGet && h.path == "/open-apis/docx/v1/documents/d1/raw_content":
			return `{"code":0,"data":{"content":"本周完成了接入。\n"}}`
		case h.method == http.MethodPost && h.path == "/open-apis/docx/v1/documents":
			return `{"code":0,"data":{"document":{"document_id":"n1","title":"周会纪要"}}}`
		case h.method == http.MethodPost && strings.HasSuffix(h.path, "/children"):
			if failBlocks {
				return `{"code":1770001,"msg":"invalid param"}`
			}
			return `{"code":0,"data":{}}`
		}
		return `{"code":404,"msg":"unexpected"}`
	})
	last := func() feishuHit { return (*hits)[len(*hits)-1] }

	rows, err := b.Read(acct.ID, "")
	if err != nil || len(rows) != 2 || rows[0]["id"] != "d1" || rows[0]["title"] != "周报模板" || rows[0]["modified"] == "" || rows[0]["url"] == "" {
		t.Fatalf("list = %+v %v", rows, err)
	}
	if q := (*hits)[0].query; q.Get("folder_token") != "fld1" || q.Get("order_by") != "EditedTime" {
		t.Fatalf("list query = %v", q)
	}
	if rows, err = b.Read(acct.ID, "预算"); err != nil || len(rows) != 1 || rows[0]["id"] != "s1" {
		t.Fatalf("search falls back to the folder listing: %+v %v", rows, err)
	}
	for _, q := range []string{"doc:d1", "https://x.feishu.cn/docx/d1?from=share"} {
		rows, err = b.Read(acct.ID, q)
		if err != nil || len(rows) != 1 || rows[0]["title"] != "周报模板" || !strings.HasPrefix(rows[0]["body"], "本周完成") {
			t.Fatalf("read %s = %+v %v", q, rows, err)
		}
	}

	n := len(*hits)
	if err := feishuSend(t, b, Draft{Account: acct.ID, Subject: "周会纪要", Body: "# 结论\n- 先做 A\n\n1. 第一步\n正文一段"}); err != nil {
		t.Fatal(err)
	}
	if len(*hits) != n+2 {
		t.Fatalf("create should be two calls: %+v", (*hits)[n:])
	}
	created, appended := (*hits)[n], (*hits)[n+1]
	if created.body["title"] != "周会纪要" || created.body["folder_token"] != "fld1" {
		t.Fatalf("create body = %+v", created.body)
	}
	if appended.path != "/open-apis/docx/v1/documents/n1/blocks/n1/children" {
		t.Fatalf("blocks go to the new document: %s", appended.path)
	}
	kids, _ := appended.body["children"].([]any)
	if len(kids) != 4 {
		t.Fatalf("children = %+v", appended.body)
	}
	for i, want := range []float64{3, 12, 13, 2} {
		kid, _ := kids[i].(map[string]any)
		if kid["block_type"] != want {
			t.Fatalf("block %d = %+v, want type %v", i, kid, want)
		}
	}
	if s := fmt.Sprint(kids[0]); !strings.Contains(s, "结论") || strings.Contains(s, "#") {
		t.Fatalf("heading text = %s", s)
	}

	failBlocks = true
	if err := feishuSend(t, b, Draft{Account: acct.ID, Subject: "周会纪要", Body: "正文"}); err == nil || !alreadyWritten(err) {
		t.Fatalf("a created doc whose body failed is already written: %v", err)
	}
	failBlocks = false

	if err := feishuSend(t, b, Draft{Account: acct.ID, ItemID: "d1", Op: "append", Body: "补充一句"}); err != nil {
		t.Fatal(err)
	}
	if last().path != "/open-apis/docx/v1/documents/d1/blocks/d1/children" {
		t.Fatalf("append = %+v", last())
	}

	n = len(*hits)
	if err := feishuSend(t, b, Draft{Account: acct.ID, ItemID: "d1", Op: "delete"}); err == nil {
		t.Fatal("yoyo does not delete Feishu docs")
	}
	if err := feishuSend(t, b, Draft{Account: acct.ID, Body: "没有标题"}); err == nil {
		t.Fatal("a new doc needs a title")
	}
	if len(*hits) != n {
		t.Fatalf("refused drafts must not call Feishu: %+v", (*hits)[n:])
	}
}
