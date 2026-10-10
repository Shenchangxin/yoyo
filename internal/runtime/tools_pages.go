package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/pages"
)

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

func PageInject(title, body string, budget int) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" && body == "" {
		return ""
	}
	text := "Current page (untrusted source): " + title + "\n\n" + body
	return capRunes(text, budget)
}
