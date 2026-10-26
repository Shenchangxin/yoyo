// 从已连接邮箱和没有里程碑的目标生成想法。日历账号在这里被跳过。

package personal

import (
	"regexp"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/connector"
)

var (
	// reDocument 匹配主题或正文里可能需要填写的词。
	reDocument = regexp.MustCompile(`(?i)form|permission|complete|fill|sign`)
	// reMeet 匹配主题或正文里可能在约时间的词。
	reMeet = regexp.MustCompile(`(?i)coffee|meet|available|schedule`)
)

// RefreshIdeas 从非日历账号的邮件里最多各提 5 条文档和约见想法，并为没有里程碑的进行中目标补一条计划想法。已发送或已成功处理的邮件会把 new 想法标成 dismissed。
func (e *Engine) RefreshIdeas() error {
	mail := e.mailViews()
	sent := map[string]bool{}
	completed := map[string]bool{}
	for _, m := range mail {
		if strings.HasPrefix(strings.ToLower(m.Label), "sent") {
			sent[m.ID] = true
		}
	}
	for _, t := range e.store.tasks() {
		if t.Status != StatusSucceeded {
			continue
		}
		if id := strMap(t.Input["message_id"]); id != "" {
			completed[string(t.Kind)+":"+id] = true
		}
	}
	obsolete := func(kind Kind, messageID string) bool {
		if messageID == "" {
			return false
		}
		return sent[messageID] || completed[string(kind)+":"+messageID]
	}
	for _, idea := range e.store.ideas() {
		if idea.Status != "new" {
			continue
		}
		if obsolete(idea.Kind, strMap(idea.Input["message_id"])) {
			e.store.casIdea(idea.ID, "new", func(cur *Idea) bool {
				cur.Status = "dismissed"
				return true
			})
		}
	}
	nDoc, nMeet := 0, 0
	for _, m := range mail {
		if obsolete(KindDocument, m.ID) {
			continue
		}
		if len(m.Attachments) == 0 || !reDocument.MatchString(m.Subject+" "+m.Body) {
			continue
		}
		nDoc++
		if nDoc > 5 {
			break
		}
		e.store.insertIdea(Idea{
			ID:     hashID("document", m.ID, m.Body),
			Title:  "I can help with " + m.Subject,
			Reason: m.Sender + " sent a document that may need your attention. I can prepare it and a reply for your review.",
			Evidence: []Evidence{{
				ID: m.ID, Kind: "mail", Title: m.Subject, Excerpt: clip(m.Body, 240),
			}},
			Prompt:    `Help complete the PDF from “` + m.Subject + `” and prepare a reply for review.`,
			Kind:      KindDocument,
			Input:     map[string]any{"message_id": m.ID},
			Status:    "new",
			CreatedAt: e.now(),
		})
	}
	for _, m := range mail {
		if obsolete(KindAgent, m.ID) {
			continue
		}
		if !reMeet.MatchString(m.Subject + " " + m.Body) {
			continue
		}
		nMeet++
		if nMeet > 5 {
			break
		}
		e.store.insertIdea(Idea{
			ID:     hashID("coordination", m.ID),
			Title:  "I can help coordinate " + m.Subject,
			Reason: m.Sender + " mentioned getting together. I can check your calendar and prepare a response for review.",
			Evidence: []Evidence{{
				ID: m.ID, Kind: "mail", Title: m.Subject, Excerpt: clip(m.Body, 240),
			}},
			Prompt:    `Review the email “` + m.Subject + `”, check my calendar, and propose a next step. Ask me about missing preferences before preparing a reply.`,
			Kind:      KindAgent,
			Input:     map[string]any{"message_id": m.ID},
			Status:    "new",
			CreatedAt: e.now(),
		})
	}
	for _, g := range e.store.goals() {
		if g.Status != "active" || len(g.Milestones) > 0 {
			continue
		}
		e.store.insertIdea(Idea{
			ID:     hashID("goal", g.ID, g.Description),
			Title:  "Let's make a plan for " + g.Title,
			Reason: "This goal has no milestones yet. A concrete plan will give it a next step.",
			Evidence: []Evidence{{
				ID: g.ID, Kind: "user", Title: g.Title, Excerpt: g.Description,
			}},
			Prompt:    "Create an actionable plan for " + g.Title + ". " + g.Description,
			Kind:      KindPlan,
			Input:     map[string]any{"goal_id": g.ID},
			Status:    "new",
			CreatedAt: e.now(),
		})
	}
	e.store.setIdeasAt(e.now())
	e.changed()
	return nil
}

// recoverIdeas 给状态为 accepted 但任务已经不在的想法重新 CreateTask。
func (e *Engine) recoverIdeas() {
	for _, idea := range e.store.ideas() {
		if idea.Status != "accepted" {
			continue
		}
		if idea.TaskID != "" {
			if _, ok := e.store.task(idea.TaskID); ok {
				continue
			}
		}
		t, err := e.CreateTask(idea.Title, idea.Prompt, idea.Kind, cloneMap(idea.Input), strMap(idea.Input["goal_id"]), "")
		if err != nil {
			continue
		}
		e.store.casIdea(idea.ID, "accepted", func(cur *Idea) bool {
			cur.TaskID = t.ID
			return true
		})
	}
}

// DecideIdea 在 action 为 dismiss 时把 new 标成 dismissed。接受 new 时，若同种类、同 message_id 已有未失败任务就复用，否则 CreateTask。
func (e *Engine) DecideIdea(id, action string) (Idea, error) {
	idea, ok := e.store.idea(id)
	if !ok {
		return Idea{}, errf("idea not found")
	}
	if idea.Status == "dismissed" || (idea.Status == "accepted" && action == "dismiss") {
		return idea, nil
	}
	if action == "dismiss" {
		out, ok := e.store.casIdea(id, "new", func(cur *Idea) bool {
			cur.Status = "dismissed"
			return true
		})
		if !ok {
			return idea, nil
		}
		e.changed()
		return out, nil
	}
	if idea.Status == "new" {
		if mid := strMap(idea.Input["message_id"]); mid != "" {
			for _, t := range e.store.tasks() {
				if t.Kind == idea.Kind && strMap(t.Input["message_id"]) == mid && t.Status != StatusFailed && t.Status != StatusCancelled {
					e.store.casIdea(id, "new", func(cur *Idea) bool {
						cur.Status = "accepted"
						cur.TaskID = t.ID
						return true
					})
					idea, _ = e.store.idea(id)
					e.changed()
					return idea, nil
				}
			}
		}
		t, err := e.CreateTask(idea.Title, idea.Prompt, idea.Kind, cloneMap(idea.Input), strMap(idea.Input["goal_id"]), "")
		if err != nil {
			return Idea{}, err
		}
		e.store.casIdea(id, "new", func(cur *Idea) bool {
			cur.Status = "accepted"
			cur.TaskID = t.ID
			return true
		})
		idea, _ = e.store.idea(id)
	}
	e.changed()
	return idea, nil
}

// mailViews 读取连接器里的邮件账号。日历账号跳过。没有主题却有 note 的占位项也跳过。
func (e *Engine) mailViews() []MailView {
	if e.Connectors == nil {
		return nil
	}
	var out []MailView
	for _, a := range e.Connectors.List() {
		if a.Kind != connector.KindMail && a.Kind != "" && a.Kind != connector.KindCalendar {
			continue
		}
		if a.Kind == connector.KindCalendar {
			continue
		}
		items, err := e.Connectors.Read(a.ID, "")
		if err != nil {
			continue
		}
		for _, it := range items {
			if it["note"] != "" && it["subject"] == "" {
				continue
			}
			mv := MailView{
				ID:       firstNonEmpty(it["id"], it["message_id"]),
				ThreadID: it["thread_id"],
				From:     it["from"],
				Sender:   firstNonEmpty(it["sender"], it["from"]),
				Subject:  it["subject"],
				Body:     it["body"],
				Label:    it["label"],
			}
			if at := it["attachments"]; at != "" {
				mv.Attachments = splitList(at)
			}
			if mv.ID == "" {
				mv.ID = hashID("mail", a.ID, mv.Subject, mv.Body)
			}
			out = append(out, mv)
		}
	}
	return out
}

// splitList 按逗号或分号切开并去掉空白项。
func splitList(s string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' }) {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
