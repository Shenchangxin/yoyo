// feishu_task.go 用飞书任务 v2 接口读写这个人的任务。令牌和飞书日历一样是用户身份的 user_access_token（feishu_oauth.go），
// 授权时申请 task:task:read、task:task:write 和 offline_access。
//
// 读：GET task/v2/tasks?type=my_tasks，默认只给没完成的；查询是「全部」「所有」「all」时连已完成的一起给，
//     其余查询按标题和描述做子串筛选。有截止时间的排前面，按截止时间先后排；最多 feishuTaskLatest 条。
// 写：草稿的 Subject 是标题，Body 是描述，End（没有时用 Start）是截止时间；ItemID（没有时用 EventID）是任务的 guid。
//     Op 含 delete 删除；含 complete 或 done 标成完成；含 reopen 重新打开；含 update 或带了 guid 时修改；否则新建。
//     截止时间写 RFC3339（带时区）、2006-01-02T15:04（按草稿时区，没写用本机时区）或 2006-01-02（全天）。
// 不拥有：清单（tasklist）、子任务、负责人和关注人（要 open_id）、重复任务（拒绝）。

package connector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// isFeishuTask 在供应商是飞书任务时为真。provider 要已经转成小写。
func isFeishuTask(provider string) bool {
	return provider == "feishu-task"
}

// feishuTaskWhat 是飞书任务在错误里的名字。
var feishuTaskWhat = feishuLabel{"feishu task", "飞书任务"}

// feishuTaskLatest 是一次最多给几条任务。
const feishuTaskLatest = 30

// feishuTaskDue 是飞书任务的截止时间：毫秒时间戳（字符串），全天任务另标 is_all_day。
type feishuTaskDue struct {
	Timestamp string `json:"timestamp,omitempty"`
	IsAllDay  bool   `json:"is_all_day,omitempty"`
}

// text 译成和日历一致的写法：定时是 UTC 的 RFC3339，全天是 2006-01-02。没有截止时间给空。
func (d *feishuTaskDue) text() string {
	if d == nil {
		return ""
	}
	n, err := strconv.ParseInt(strings.TrimSpace(d.Timestamp), 10, 64)
	if err != nil || n <= 0 {
		return ""
	}
	t := time.UnixMilli(n).UTC()
	if d.IsAllDay {
		return t.Format("2006-01-02")
	}
	return t.Format(time.RFC3339)
}

// taskDone 在 completed_at 不是空也不是 "0" 时为真（飞书没完成的任务给 "0"）。
func taskDone(completedAt string) bool {
	s := strings.TrimSpace(completedAt)
	return s != "" && s != "0"
}

// feishuTasks 读这个人的任务，见文件头。什么都没有时给一条 note。最多翻五页（每页五十条）。
func feishuTasks(client *http.Client, token, query string) ([]map[string]string, error) {
	q := strings.TrimSpace(query)
	all := false
	switch strings.ToLower(q) {
	case "all", "全部", "所有":
		all, q = true, ""
	}
	needle := strings.ToLower(q)
	var out []map[string]string
	pageToken := ""
	for page := 0; page < 5 && len(out) < feishuTaskLatest; page++ {
		v := url.Values{"page_size": {"50"}, "type": {"my_tasks"}, "user_id_type": {"open_id"}}
		if !all {
			v.Set("completed", "false")
		}
		if pageToken != "" {
			v.Set("page_token", pageToken)
		}
		raw, err := feishuDo(client, http.MethodGet, "/open-apis/task/v2/tasks?"+v.Encode(), token, nil, feishuTaskWhat)
		if err != nil {
			return nil, err
		}
		var doc struct {
			Data struct {
				Items []struct {
					GUID        string         `json:"guid"`
					Summary     string         `json:"summary"`
					Description string         `json:"description"`
					Due         *feishuTaskDue `json:"due"`
					CompletedAt string         `json:"completed_at"`
					URL         string         `json:"url"`
				} `json:"items"`
				HasMore   bool   `json:"has_more"`
				PageToken string `json:"page_token"`
			} `json:"data"`
		}
		_ = json.Unmarshal(raw, &doc)
		for _, it := range doc.Data.Items {
			if strings.TrimSpace(it.GUID) == "" {
				continue
			}
			if needle != "" && !strings.Contains(strings.ToLower(it.Summary+" "+it.Description), needle) {
				continue
			}
			out = append(out, map[string]string{
				"provider":    "feishu-task",
				"kind":        "task",
				"id":          it.GUID,
				"task_id":     it.GUID,
				"title":       strings.TrimSpace(it.Summary),
				"description": strings.TrimSpace(it.Description),
				"due":         it.Due.text(),
				"completed":   strconv.FormatBool(taskDone(it.CompletedAt)),
				"url":         it.URL,
			})
			if len(out) == feishuTaskLatest {
				break
			}
		}
		if !doc.Data.HasMore || doc.Data.PageToken == "" {
			break
		}
		pageToken = doc.Data.PageToken
	}
	sort.SliceStable(out, func(i, j int) bool {
		x, y := out[i]["due"], out[j]["due"]
		if (x == "") != (y == "") {
			return x != ""
		}
		return x < y
	})
	if len(out) == 0 {
		out = append(out, map[string]string{"provider": "feishu-task", "kind": "task", "note": "no tasks", "query": query})
	}
	return out, nil
}

// feishuDue 认出草稿里的截止时间（End，没有时用 Start），见文件头。没写时给 nil；写了认不出时报错，一个请求都不发。
func feishuDue(d Draft) (*feishuTaskDue, error) {
	raw := strings.TrimSpace(d.End)
	if raw == "" {
		raw = strings.TrimSpace(d.Start)
	}
	if raw == "" {
		return nil, nil
	}
	loc := time.Local
	if tz := strings.TrimSpace(d.TimeZone); tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	ms := func(t time.Time) string { return strconv.FormatInt(t.UnixMilli(), 10) }
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &feishuTaskDue{Timestamp: ms(t), IsAllDay: d.AllDay}, nil
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return &feishuTaskDue{Timestamp: ms(t), IsAllDay: d.AllDay}, nil
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", raw, loc); err == nil {
		return &feishuTaskDue{Timestamp: ms(t), IsAllDay: true}, nil
	}
	return nil, fmt.Errorf("connector: 截止时间 %q 认不出，请写成 2026-12-01 或 2026-12-01T18:00:00+08:00", raw)
}

// feishuWriteTask 按草稿新建、修改、完成、重新打开或删除一条飞书任务，见文件头。
func feishuWriteTask(client *http.Client, token string, d Draft) error {
	if strings.TrimSpace(d.Recurrence) != "" {
		return fmt.Errorf("connector: recurring tasks must be edited in the Feishu app (422)")
	}
	op := strings.ToLower(strings.TrimSpace(d.Op))
	guid := feishuItemID(d)
	one := "/open-apis/task/v2/tasks/" + url.PathEscape(guid)
	needGUID := func(what string) error {
		if guid == "" {
			return fmt.Errorf("connector: %s飞书任务要 item_id（读任务时每条都带着 id）", what)
		}
		return nil
	}
	patch := func(task map[string]any, fields []string) error {
		_, err := feishuDo(client, http.MethodPatch, one+"?user_id_type=open_id", token,
			map[string]any{"task": task, "update_fields": fields}, feishuTaskWhat)
		return err
	}
	switch {
	case strings.Contains(op, "delete"):
		if err := needGUID("删除"); err != nil {
			return err
		}
		_, err := feishuDo(client, http.MethodDelete, one, token, nil, feishuTaskWhat)
		return err
	case strings.Contains(op, "complete") || strings.Contains(op, "done"):
		if err := needGUID("完成"); err != nil {
			return err
		}
		return patch(map[string]any{"completed_at": strconv.FormatInt(time.Now().UnixMilli(), 10)}, []string{"completed_at"})
	case strings.Contains(op, "reopen"):
		if err := needGUID("重新打开"); err != nil {
			return err
		}
		return patch(map[string]any{"completed_at": "0"}, []string{"completed_at"})
	}
	due, err := feishuDue(d)
	if err != nil {
		return err
	}
	if strings.Contains(op, "update") || guid != "" {
		if err := needGUID("修改"); err != nil {
			return err
		}
		task := map[string]any{}
		var fields []string
		if s := strings.TrimSpace(d.Subject); s != "" {
			task["summary"], fields = s, append(fields, "summary")
		}
		if s := strings.TrimSpace(d.Body); s != "" {
			task["description"], fields = s, append(fields, "description")
		}
		if due != nil {
			task["due"], fields = due, append(fields, "due")
		}
		if len(fields) == 0 {
			return fmt.Errorf("connector: 草稿里没有要改的标题、描述或截止时间")
		}
		return patch(task, fields)
	}
	summary := strings.TrimSpace(d.Subject)
	if summary == "" {
		return fmt.Errorf("connector: 新建飞书任务要一个标题")
	}
	task := map[string]any{"summary": summary}
	if s := strings.TrimSpace(d.Body); s != "" {
		task["description"] = s
	}
	if due != nil {
		task["due"] = due
	}
	_, err = feishuDo(client, http.MethodPost, "/open-apis/task/v2/tasks?user_id_type=open_id", token, task, feishuTaskWhat)
	return err
}
