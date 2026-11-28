// feishu_docs.go 用飞书开放平台的云空间和新版文档（docx）接口找、读、写这个人的云文档。
// 令牌和飞书日历一样是用户身份的 user_access_token（feishu_oauth.go），授权时申请 docx:document、drive:drive:readonly 和 offline_access。
// 账号上的 Endpoint 可以填一个文件夹 token：列文件和新建文档都在这个文件夹里；不填是「我的空间」根目录。
//
// 读：查询是「doc:<document_id>」或飞书文档链接（…/docx/<id>）时读这一篇的标题和纯文本正文；
//     查询为空时按最近编辑列出文件夹里的文件；其余查询先走云文档搜索，搜索没开权限或出错时退回列文件夹再按名字筛。
// 写：草稿的 Subject 是标题，Body 是正文（按行写成段落；# / ## / ### 开头写成标题，- / * 开头写成无序列表，1. 开头写成有序列表）。
//     Op 含 append 或 update、或者带了 ItemID（没有时用 EventID）时把正文追加到那篇文档末尾；否则新建一篇。
//     新建成功但正文没写全时返回 writtenError：liveSend 不会重发，免得多出一篇一样的文档。
// 不拥有：删除文档（yoyo 不删云文档，要删请在飞书里删）、旧版文档和表格的正文、改写已有段落。

package connector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// isFeishuDocs 在供应商是飞书云文档时为真。provider 要已经转成小写。
func isFeishuDocs(provider string) bool {
	return provider == "feishu-docs"
}

// feishuDocsWhat 是飞书云文档在错误里的名字。
var feishuDocsWhat = feishuLabel{"feishu docs", "飞书云文档"}

// feishuDocsLatest 是一次最多给几篇文档。
const feishuDocsLatest = 20

// feishuBlockBatch 是一次最多追加几个块（飞书一次最多收 50 个）。
const feishuBlockBatch = 50

// orderedLine 认出有序列表的行首：「1. 」「2) 」「3、」。「2026.11 报告」这种数字后面不跟空白的不算。
var orderedLine = regexp.MustCompile(`^\d{1,3}(?:[.)）]\s+|、\s*)`)

// feishuItemID 是草稿要改的那一项：ItemID，没有时用 EventID（老写法）。
func feishuItemID(d Draft) string {
	if id := strings.TrimSpace(d.ItemID); id != "" {
		return id
	}
	return strings.TrimSpace(d.EventID)
}

// feishuSeconds 把飞书给的秒数时间戳译成 UTC 的 RFC3339。认不出时给空。
func feishuSeconds(s string) string {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n <= 0 {
		return ""
	}
	return time.Unix(n, 0).UTC().Format(time.RFC3339)
}

// feishuDocToken 认出查询里点名的一篇新版文档：「doc:<id>」，或者路径里有 /docx/<id> 的飞书链接。认不出时给空。
func feishuDocToken(query string) string {
	q := strings.TrimSpace(query)
	if rest, ok := strings.CutPrefix(q, "doc:"); ok {
		return strings.TrimSpace(rest)
	}
	if !strings.HasPrefix(q, "https://") {
		return ""
	}
	u, err := url.Parse(q)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "docx" && parts[i+1] != "" {
			return parts[i+1]
		}
	}
	return ""
}

// feishuDocs 按查询读云文档，见文件头。什么都没有时给一条 note。
func feishuDocs(client *http.Client, folder, token, query string) ([]map[string]string, error) {
	if id := feishuDocToken(query); id != "" {
		return feishuDocText(client, token, id)
	}
	q := strings.TrimSpace(query)
	var rows []map[string]string
	var err error
	if q != "" {
		rows, err = feishuDocSearch(client, token, q)
		// 令牌失效要交回 liveRead 换票重试；别的错误（多半是没开搜索权限）退回列文件夹。
		if err != nil && authExpired(err) {
			return nil, err
		}
	}
	if q == "" || err != nil {
		rows, err = feishuDocList(client, folder, token, q)
		if err != nil {
			return nil, err
		}
	}
	if len(rows) == 0 {
		rows = append(rows, map[string]string{"provider": "feishu-docs", "kind": "drive", "note": "no documents", "query": query})
	}
	return rows, nil
}

// feishuDocText 读一篇新版文档的标题和纯文本正文。
func feishuDocText(client *http.Client, token, id string) ([]map[string]string, error) {
	base := "/open-apis/docx/v1/documents/" + url.PathEscape(id)
	raw, err := feishuDo(client, http.MethodGet, base, token, nil, feishuDocsWhat)
	if err != nil {
		return nil, err
	}
	var meta struct {
		Data struct {
			Document struct {
				Title string `json:"title"`
			} `json:"document"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &meta)
	raw, err = feishuDo(client, http.MethodGet, base+"/raw_content", token, nil, feishuDocsWhat)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &doc)
	return []map[string]string{{
		"provider": "feishu-docs",
		"kind":     "drive",
		"id":       id,
		"doc_id":   id,
		"type":     "docx",
		"title":    strings.TrimSpace(meta.Data.Document.Title),
		"body":     strings.TrimSpace(doc.Data.Content),
	}}, nil
}

// feishuDocSearch 用云文档搜索找标题或正文里有 q 的文档，最多 feishuDocsLatest 篇。
// 搜索结果里没有链接；新版文档可以接着用「doc:<id>」读正文。
func feishuDocSearch(client *http.Client, token, q string) ([]map[string]string, error) {
	raw, err := feishuDo(client, http.MethodPost, "/open-apis/suite/docs-api/search/object", token,
		map[string]any{"search_key": q, "count": feishuDocsLatest, "offset": 0}, feishuDocsWhat)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Data struct {
			Entities []struct {
				Token string `json:"docs_token"`
				Type  string `json:"docs_type"`
				Title string `json:"title"`
			} `json:"docs_entities"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &doc)
	var out []map[string]string
	for _, e := range doc.Data.Entities {
		if strings.TrimSpace(e.Token) == "" {
			continue
		}
		out = append(out, map[string]string{
			"provider": "feishu-docs",
			"kind":     "drive",
			"id":       e.Token,
			"doc_id":   e.Token,
			"type":     e.Type,
			"title":    strings.TrimSpace(e.Title),
		})
		if len(out) == feishuDocsLatest {
			break
		}
	}
	return out, nil
}

// feishuDocList 按最近编辑列出文件夹里的文件（folder 为空是「我的空间」根目录），needle 非空时只留名字里有它的。
// 筛选时最多翻三页（每页五十个）。
func feishuDocList(client *http.Client, folder, token, needle string) ([]map[string]string, error) {
	needle = strings.ToLower(strings.TrimSpace(needle))
	var out []map[string]string
	pageToken := ""
	for page := 0; page < 3 && len(out) < feishuDocsLatest; page++ {
		v := url.Values{"page_size": {"50"}, "order_by": {"EditedTime"}, "direction": {"DESC"}}
		if f := strings.TrimSpace(folder); f != "" {
			v.Set("folder_token", f)
		}
		if pageToken != "" {
			v.Set("page_token", pageToken)
		}
		raw, err := feishuDo(client, http.MethodGet, "/open-apis/drive/v1/files?"+v.Encode(), token, nil, feishuDocsWhat)
		if err != nil {
			return nil, err
		}
		var doc struct {
			Data struct {
				Files []struct {
					Token        string `json:"token"`
					Name         string `json:"name"`
					Type         string `json:"type"`
					URL          string `json:"url"`
					ModifiedTime string `json:"modified_time"`
				} `json:"files"`
				HasMore       bool   `json:"has_more"`
				NextPageToken string `json:"next_page_token"`
			} `json:"data"`
		}
		_ = json.Unmarshal(raw, &doc)
		for _, f := range doc.Data.Files {
			if strings.TrimSpace(f.Token) == "" {
				continue
			}
			if needle != "" && !strings.Contains(strings.ToLower(f.Name), needle) {
				continue
			}
			out = append(out, map[string]string{
				"provider": "feishu-docs",
				"kind":     "drive",
				"id":       f.Token,
				"doc_id":   f.Token,
				"type":     f.Type,
				"title":    strings.TrimSpace(f.Name),
				"url":      f.URL,
				"modified": feishuSeconds(f.ModifiedTime),
			})
			if len(out) == feishuDocsLatest {
				break
			}
		}
		if !doc.Data.HasMore || doc.Data.NextPageToken == "" {
			break
		}
		pageToken = doc.Data.NextPageToken
	}
	return out, nil
}

// feishuBlocks 把正文按行写成飞书文档的块：空行跳过；# / ## / ### 是一到三级标题，- / * 是无序列表，1. 是有序列表，其余是段落。
func feishuBlocks(body string) []map[string]any {
	var out []map[string]any
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		typ, key := 2, "text"
		switch {
		case strings.HasPrefix(s, "### "):
			typ, key, s = 5, "heading3", s[4:]
		case strings.HasPrefix(s, "## "):
			typ, key, s = 4, "heading2", s[3:]
		case strings.HasPrefix(s, "# "):
			typ, key, s = 3, "heading1", s[2:]
		case strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "* "):
			typ, key, s = 12, "bullet", s[2:]
		case orderedLine.MatchString(s):
			typ, key, s = 13, "ordered", orderedLine.ReplaceAllString(s, "")
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, map[string]any{
			"block_type": typ,
			key: map[string]any{
				"elements": []map[string]any{{"text_run": map[string]string{"content": s}}},
			},
		})
	}
	return out
}

// feishuWriteDoc 按草稿新建一篇文档或往已有文档末尾追加，见文件头。folder 是账号上填的文件夹 token。
func feishuWriteDoc(client *http.Client, folder, token string, d Draft) error {
	op := strings.ToLower(strings.TrimSpace(d.Op))
	if strings.Contains(op, "delete") {
		return fmt.Errorf("connector: yoyo 不删除飞书文档，要删请在飞书里删")
	}
	blocks := feishuBlocks(d.Body)
	id := feishuItemID(d)
	if strings.Contains(op, "append") || strings.Contains(op, "update") || id != "" {
		if id == "" {
			return fmt.Errorf("connector: 追加到飞书文档要 item_id（读文档时每条都带着 id）")
		}
		if len(blocks) == 0 {
			return fmt.Errorf("connector: 没有要追加到飞书文档的内容")
		}
		return feishuAppendBlocks(client, token, id, blocks, false)
	}
	title := strings.TrimSpace(d.Subject)
	if title == "" {
		return fmt.Errorf("connector: 新建飞书文档要一个标题")
	}
	body := map[string]any{"title": title}
	if f := strings.TrimSpace(folder); f != "" {
		body["folder_token"] = f
	}
	raw, err := feishuDo(client, http.MethodPost, "/open-apis/docx/v1/documents", token, body, feishuDocsWhat)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return nil
	}
	var doc struct {
		Data struct {
			Document struct {
				ID string `json:"document_id"`
			} `json:"document"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &doc)
	newID := strings.TrimSpace(doc.Data.Document.ID)
	if newID == "" {
		return writtenError{fmt.Errorf("connector: 飞书文档「%s」已经建好，但答复里没有 document_id，正文没写进去。不要重新建，请在飞书里补上正文", title)}
	}
	return feishuAppendBlocks(client, token, newID, blocks, true)
}

// feishuAppendBlocks 把块按批追加到文档末尾。created 为真表示文档是刚建的。
// 已经写进去一部分（文档是刚建的，或者前面的批次成功了）之后再失败时返回 writtenError，liveSend 不会整份重发。
func feishuAppendBlocks(client *http.Client, token, doc string, blocks []map[string]any, created bool) error {
	path := "/open-apis/docx/v1/documents/" + url.PathEscape(doc) + "/blocks/" + url.PathEscape(doc) + "/children?document_revision_id=-1"
	for i := 0; i < len(blocks); i += feishuBlockBatch {
		j := i + feishuBlockBatch
		if j > len(blocks) {
			j = len(blocks)
		}
		if _, err := feishuDo(client, http.MethodPost, path, token, map[string]any{"children": blocks[i:j]}, feishuDocsWhat); err != nil {
			if created || i > 0 {
				return writtenError{fmt.Errorf("connector: 飞书文档 %s 已经写进去一部分（%d/%d 段），后面的没写上：%s。不要重新建，在飞书里补上或者只追加剩下的", doc, i, len(blocks), strings.TrimPrefix(err.Error(), "connector: "))}
			}
			return err
		}
	}
	return nil
}
