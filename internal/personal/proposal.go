package personal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connector"
	"github.com/Shenchangxin/yoyo/internal/inbox"
)

const proposalTTL = 30 * time.Minute

func proposalHash(kind ProposalKind, data map[string]any, account, targetVersion string) string {
	payload, _ := json.Marshal(struct {
		Kind          ProposalKind   `json:"kind"`
		Data          map[string]any `json:"data"`
		Account       string         `json:"account"`
		TargetVersion string         `json:"target_version"`
	}{kind, data, account, targetVersion})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (e *Engine) Propose(kind ProposalKind, data map[string]any, account, taskID, source, idempotency string) (Proposal, error) {
	if data == nil {
		data = map[string]any{}
	}
	kind = ProposalKind(strings.TrimSpace(string(kind)))
	switch kind {
	case ProposalEmailSend, ProposalCalendarCreate, ProposalCalendarUpdate, ProposalCalendarDelete:
	default:
		return Proposal{}, errf("proposal kind must be email.send or calendar.create|update|delete")
	}
	if rec := strMap(data["recurrence"]); rec != "" {
		return Proposal{}, errf("recurring series must be edited in the calendar app (422)")
	}
	target := strMap(data["target_version"])
	if kind == ProposalCalendarUpdate || kind == ProposalCalendarDelete {
		if strMap(data["event_id"]) == "" || target == "" {
			return Proposal{}, errf("calendar update/delete needs event_id and target_version")
		}
	}
	if account != "" && e.Connectors != nil {
		found := false
		for _, a := range e.Connectors.List() {
			if a.ID == account {
				found = true
				break
			}
		}
		if !found {
			return Proposal{}, errf("connect an account before preparing an action")
		}
	}
	id := idempotency
	if id == "" {
		id = e.newID("act")
	} else {
		id = hashID("act", id)
		if existing, ok := e.store.proposal(id); ok {
			return existing, nil
		}
	}
	now := e.now()
	p := Proposal{
		ID:            id,
		TaskID:        taskID,
		Account:       account,
		Title:         proposalTitle(kind, data),
		Kind:          kind,
		Data:          data,
		Status:        ProposalAwaiting,
		TargetVersion: target,
		Hash:          proposalHash(kind, data, account, target),
		CreatedAt:     now,
		ExpiresAt:     now.Add(proposalTTL),
		Source:        source,
		Activity:      []Activity{{At: now, Status: string(ProposalAwaiting), Detail: "Ready for your review"}},
	}
	e.store.putProposal(p)
	e.notify(inbox.KindProposal, p.Title, "Hash "+p.Hash[:12]+" · expires in 30m", taskID, "proposal:"+p.ID)
	return p, nil
}

func proposalTitle(kind ProposalKind, data map[string]any) string {
	switch kind {
	case ProposalEmailSend:
		return fmt.Sprintf("Send “%s”", strMap(data["subject"]))
	case ProposalCalendarDelete:
		title := strMap(data["title"])
		if title == "" {
			title = strMap(data["subject"])
		}
		return "Delete " + title
	case ProposalCalendarUpdate:
		title := strMap(data["title"])
		if title == "" {
			title = strMap(data["subject"])
		}
		return "Update " + title
	default:
		title := strMap(data["title"])
		if title == "" {
			title = strMap(data["subject"])
		}
		return "Create " + title
	}
}

func (e *Engine) RecordChatSend(d connector.Draft) Proposal {
	data := map[string]any{
		"to": d.To, "subject": d.Subject, "body": d.Body, "draft_id": d.ID,
		"cc": d.Cc, "bcc": d.Bcc, "thread_id": d.ThreadID, "reply_to": d.ReplyTo,
		"attachments": d.Attachments, "location": d.Location, "end": d.End,
		"time_zone": d.TimeZone, "all_day": d.AllDay, "start": d.Start,
		"calendar_id": d.CalendarID, "attendees": d.Attendees,
		"event_id": d.EventID, "target_version": d.TargetVersion, "recurrence": d.Recurrence,
	}
	kind := ProposalEmailSend
	if looksCalendarDraft(d) {
		kind = ProposalCalendarCreate
		if d.EventID != "" && strings.EqualFold(d.Op, "delete") {
			kind = ProposalCalendarDelete
		} else if d.EventID != "" {
			kind = ProposalCalendarUpdate
		}
	}
	p, err := e.Propose(kind, data, d.Account, "", "chat", "chat:"+d.ID)
	if err != nil {
		return Proposal{}
	}
	return p
}

func looksCalendarDraft(d connector.Draft) bool {
	if d.Start != "" || d.End != "" || d.Location != "" || d.CalendarID != "" || d.EventID != "" {
		return true
	}
	return strings.Contains(d.To, "T") && !strings.Contains(d.To, "@")
}

func (e *Engine) FinishChatSend(id string, sendErr error) Proposal {
	p, ok := e.store.proposal(id)
	if !ok {
		return Proposal{}
	}
	if sendErr != nil {
		if unknownOutcome(sendErr) {
			p.Status = ProposalOutcomeUnknown
		} else {
			p.Status = ProposalFailed
		}
		p.Error = sendErr.Error()
		p.Activity = append(p.Activity, Activity{At: e.now(), Status: string(p.Status), Detail: p.Error})
	} else {
		p.Status = ProposalSucceeded
		p.Result = "sent"
		p.Activity = append(p.Activity, Activity{At: e.now(), Status: string(p.Status), Detail: "sent"})
	}
	e.store.putProposal(p)
	e.changed()
	return p
}

func (e *Engine) Decide(id, hash string, approve bool) (Proposal, error) {
	p, ok := e.store.proposal(id)
	if !ok {
		return Proposal{}, errf("action not found")
	}
	if hash != "" && p.Hash != hash {
		return Proposal{}, errf("this proposal changed; open its latest review before deciding")
	}
	if p.Status != ProposalAwaiting {
		return p, nil
	}
	if !e.now().Before(p.ExpiresAt) {
		e.store.casProposal(id, string(ProposalAwaiting), p.Hash, func(cur *Proposal) bool {
			cur.Status = ProposalExpired
			cur.Activity = append(cur.Activity, Activity{At: e.now(), Status: string(ProposalExpired), Detail: "expired"})
			return true
		})
		return Proposal{}, errf("this review expired; create a fresh proposal")
	}
	if !approve {
		out, ok := e.store.casProposal(id, string(ProposalAwaiting), p.Hash, func(cur *Proposal) bool {
			cur.Status = ProposalDenied
			cur.Activity = append(cur.Activity, Activity{At: e.now(), Status: string(ProposalDenied), Detail: "Declined; no changes made"})
			return true
		})
		if !ok {
			return p, nil
		}
		if p.TaskID != "" {
			e.store.casTask(p.TaskID, "", "", func(cur *Task) bool {
				cur.Status = StatusCancelled
				cur.Error = "operator denied the send"
				cur.UpdatedAt = e.now()
				return true
			})
		}
		e.changed()
		return out, nil
	}
	if p.Account != "" && e.Connectors != nil {
		found := false
		for _, a := range e.Connectors.List() {
			if a.ID == p.Account {
				found = true
				break
			}
		}
		if !found {
			return Proposal{}, errf("account or connection changed; prepare a new action")
		}
	}
	if p.TaskID != "" {
		t, ok := e.store.task(p.TaskID)
		if !ok || (t.Status != StatusWaitingApproval && t.Status != StatusRunning) {
			return Proposal{}, errf("resume the task before approving this action")
		}
	}
	claimed, ok := e.store.casProposal(id, string(ProposalAwaiting), p.Hash, func(cur *Proposal) bool {
		cur.Status = ProposalExecuting
		cur.Activity = append(cur.Activity, Activity{At: e.now(), Status: string(ProposalExecuting), Detail: "Approved; execution started"})
		return true
	})
	if !ok {
		cur, _ := e.store.proposal(id)
		return cur, nil
	}
	result, err := e.executeProposal(claimed)
	if err != nil {
		st := ProposalFailed
		if unknownOutcome(err) {
			st = ProposalOutcomeUnknown
		}
		out, _ := e.store.casProposal(id, string(ProposalExecuting), "", func(cur *Proposal) bool {
			cur.Status = st
			cur.Error = err.Error()
			cur.Activity = append(cur.Activity, Activity{At: e.now(), Status: string(st), Detail: err.Error()})
			return true
		})
		return out, err
	}
	out, _ := e.store.casProposal(id, string(ProposalExecuting), "", func(cur *Proposal) bool {
		cur.Status = ProposalSucceeded
		cur.Result = result
		cur.Activity = append(cur.Activity, Activity{At: e.now(), Status: string(ProposalSucceeded), Detail: result})
		return true
	})
	if claimed.TaskID != "" {
		e.store.casTask(claimed.TaskID, "", "", func(cur *Task) bool {
			cur.Status = StatusSucceeded
			cur.Result = result
			cur.UpdatedAt = e.now()
			cur.LeaseID = ""
			cur.LeaseUntil = nil
			return true
		})
		if t, ok := e.store.task(claimed.TaskID); ok {
			e.notify(inbox.KindWork, t.Title, result, t.ID, "task-done:"+t.ID)
		}
	}
	return out, nil
}

func (e *Engine) executeProposal(p Proposal) (string, error) {
	if rec := strMap(p.Data["recurrence"]); rec != "" {
		return "", errf("recurring series must be edited in the calendar app (422)")
	}
	if e.Connectors == nil {
		return "", errf("no connectors")
	}
	d := draftFromData(p.Account, p.Data)
	d.Op = string(p.Kind)
	d.TargetVersion = p.TargetVersion
	d = e.Connectors.PutDraft(d)
	sent, err := e.Connectors.Send(d.ID)
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(sent)
	return "sent " + string(b), nil
}

func draftFromData(account string, data map[string]any) connector.Draft {
	d := connector.Draft{
		Account:       account,
		To:            strMap(data["to"]),
		Subject:       strMap(data["subject"]),
		Body:          strMap(data["body"]),
		Cc:            strMap(data["cc"]),
		Bcc:           strMap(data["bcc"]),
		ThreadID:      strMap(data["thread_id"]),
		ReplyTo:       strMap(data["reply_to"]),
		Location:      strMap(data["location"]),
		TimeZone:      strMap(data["time_zone"]),
		Start:         strMap(data["start"]),
		End:           strMap(data["end"]),
		CalendarID:    strMap(data["calendar_id"]),
		EventID:       strMap(data["event_id"]),
		TargetVersion: strMap(data["target_version"]),
		Recurrence:    strMap(data["recurrence"]),
		Op:            strMap(data["op"]),
	}
	if v, ok := data["all_day"].(bool); ok {
		d.AllDay = v
	}
	if v, ok := data["attachments"].([]string); ok {
		d.Attachments = v
	} else if v, ok := data["attachments"].([]any); ok {
		for _, x := range v {
			if s := strings.TrimSpace(fmt.Sprint(x)); s != "" {
				d.Attachments = append(d.Attachments, s)
			}
		}
	}
	if v, ok := data["attendees"].([]string); ok {
		d.Attendees = v
	} else if v, ok := data["attendees"].([]any); ok {
		for _, x := range v {
			if s := strings.TrimSpace(fmt.Sprint(x)); s != "" {
				d.Attendees = append(d.Attendees, s)
			}
		}
	}
	if title := strMap(data["title"]); title != "" && d.Subject == "" {
		d.Subject = title
	}
	return d
}

func (e *Engine) expireProposals() {
	now := e.now()
	for _, p := range e.store.snapshot().Proposals {
		if p.Status != ProposalAwaiting || now.Before(p.ExpiresAt) {
			continue
		}
		e.store.casProposal(p.ID, string(ProposalAwaiting), p.Hash, func(cur *Proposal) bool {
			cur.Status = ProposalExpired
			cur.Activity = append(cur.Activity, Activity{At: now, Status: string(ProposalExpired), Detail: "expired"})
			return true
		})
	}
}

func unknownOutcome(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "outcome_unknown") || strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline") {
		return true
	}
	if strings.Contains(msg, "429") {
		return true
	}
	for code := 500; code <= 599; code++ {
		s := strconv.Itoa(code)
		if strings.Contains(msg, "http "+s) || strings.Contains(msg, " "+s) {
			return true
		}
	}
	return false
}
