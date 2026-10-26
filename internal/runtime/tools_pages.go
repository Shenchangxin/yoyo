// 页面库的读取、列表和提交审阅。review_page 走 WriteWorkspace，ForceAsk 为假。注入当前页正文的截断在 PageInject。

package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/pages"
)

// readPage 在 Pages 为空时返回错误。Get 成功后把页面 JSON 标成 untrusted page。
// id 作为第二参交给 Pages.Get。
// spaceID 作为第一参交给 Pages.Get。
func (t *WorkspaceTools) readPage(id, spaceID string) ToolResult {
	if t == nil || t.Pages == nil {
		return ToolResult{Err: fmt.Errorf("pages store unavailable")}
	}
	pg, err := t.Pages.Get(spaceID, id)
	if err != nil {
		return ToolResult{Err: err}
	}
	b, _ := json.MarshalIndent(pg, "", "  ")
	return ToolResult{Content: "untrusted page:\n" + string(b)}
}

// listPages 在 Pages 为空时返回 []。query 非空则 Search，否则 List。
// spaceID 在 query 去掉空白后非空时交给 Search，否则交给 List。
func (t *WorkspaceTools) listPages(query, spaceID string) ToolResult {
	if t == nil || t.Pages == nil {
		return ToolResult{Content: "[]"}
	}
	var list []pages.Meta
	if strings.TrimSpace(query) != "" {
		list = t.Pages.Search(query, spaceID)
	} else {
		list = t.Pages.List(spaceID)
	}
	b, _ := json.MarshalIndent(list, "", "  ")
	return ToolResult{Content: string(b)}
}

// reviewPage 要求 title 和 content 都非空白。能力请求的级别是 WriteWorkspace，动作是 review_page，ForceAsk 为假。Propose 成功后若有 Inbox 就推一条提案，并返回 awaiting page review。
// args 读取 title、content、space_id、id、parent_id 和 expected_revision；title 或 content 去掉空白后为空则返回错误。
func (t *WorkspaceTools) reviewPage(args map[string]any) ToolResult {
	if t == nil || t.Pages == nil {
		return ToolResult{Err: fmt.Errorf("pages store unavailable")}
	}
	title := str(args["title"])
	content := str(args["content"])
	if strings.TrimSpace(title) == "" || strings.TrimSpace(content) == "" {
		return ToolResult{Err: fmt.Errorf("review_page needs title and content")}
	}
	req := capability.Request{
		Level:     capability.WriteWorkspace,
		Action:    "review_page",
		Command:   title,
		SessionID: t.SessionID,
		Workspace: t.Workspace,
		ForceAsk:  false,
	}
	if t.Caps != nil {
		if t.Ctx != nil {
			if err := t.Caps.CheckCtx(t.Ctx, req); err != nil {
				return ToolResult{Err: err}
			}
		} else if err := t.Caps.Check(req); err != nil {
			return ToolResult{Err: err}
		}
	}
	rev, err := t.Pages.Propose(pages.Review{
		ThreadID:         t.SessionID,
		SessionID:        t.SessionID,
		ToolCallID:       t.liveCall,
		SpaceID:          str(args["space_id"]),
		PageID:           str(args["id"]),
		ParentID:         str(args["parent_id"]),
		Title:            title,
		Content:          content,
		ExpectedRevision: intArg(args["expected_revision"]),
	})
	if err != nil {
		return ToolResult{Err: err}
	}
	if t.Inbox != nil {
		t.Inbox.Push(inbox.Item{Kind: inbox.KindProposal, Title: "Page: " + rev.Title, Body: rev.Hash, SessionID: t.SessionID, Key: rev.ID})
	}
	payload, _ := json.Marshal(map[string]any{
		"review_id": rev.ID, "hash": rev.Hash, "title": rev.Title,
		"status": rev.Status, "page_id": rev.PageID, "kind": "page.save",
	})
	return ToolResult{Content: "awaiting page review " + string(payload)}
}

// PageInject 把标题和正文标成不可信来源。两者都空则返回空串。再用 capRunes 按 budget 截断。
// title 去掉空白；和 body 都空时返回空串，否则写进 Current page 那一行。
// body 去掉空白；和 title 都空时返回空串，否则接在标题后面。
func PageInject(title, body string, budget int) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" && body == "" {
		return ""
	}
	text := "Current page (untrusted source): " + title + "\n\n" + body
	return capRunes(text, budget)
}
