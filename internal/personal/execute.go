package personal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/inbox"
	"github.com/Shenchangxin/yoyo/internal/office"
)

type lostLeaseError struct{}

func (lostLeaseError) Error() string { return "personal: lost lease" }

func isLostLease(err error) bool {
	_, ok := err.(lostLeaseError)
	return ok
}

func shaText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func (e *Engine) execute(ctx context.Context, t Task, leaseID string) (Task, error) {
	guard := func() error {
		cur, ok := e.store.task(t.ID)
		if !ok || cur.LeaseID != leaseID || cur.Status != StatusRunning {
			return lostLeaseError{}
		}
		if ctx.Err() != nil {
			return lostLeaseError{}
		}
		return nil
	}
	checkpoint := func(patch func(*Task)) error {
		if err := guard(); err != nil {
			return err
		}
		out, ok := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
			patch(cur)
			cur.UpdatedAt = e.now()
			return true
		})
		if !ok {
			return lostLeaseError{}
		}
		t = out
		return nil
	}

	if t.ActionID != "" {
		action, ok := e.store.proposal(t.ActionID)
		if !ok {
			return t, errf("the linked review could not be found")
		}
		switch action.Status {
		case ProposalSucceeded:
			if t.Kind == KindDocument {
				return e.finish(t, leaseID, action.Result)
			}
			_ = checkpoint(func(cur *Task) {
				cur.State["approvalResult"] = action.Result
				cur.ActionID = ""
			})
		case ProposalAwaiting, ProposalExecuting, ProposalOutcomeUnknown:
			q := ""
			if action.Status == ProposalOutcomeUnknown {
				q = "Send outcome is unknown. Check the provider before creating another action."
			}
			out, _ := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
				cur.Status = StatusWaitingApproval
				cur.Question = q
				cur.LeaseID = ""
				cur.LeaseUntil = nil
				cur.UpdatedAt = e.now()
				return true
			})
			return out, nil
		default:
			return t, errf("reviewed action %s: %s", action.Status, action.Error)
		}
	}

	switch t.Kind {
	case KindDocument:
		return e.runDocument(ctx, t, leaseID, guard, checkpoint)
	case KindMonitor:
		return e.runMonitor(ctx, t, leaseID, guard, checkpoint)
	case KindFinance:
		return e.runFinance(t, leaseID, checkpoint)
	case KindPlan:
		return e.runPlan(t, leaseID, checkpoint)
	default:
		out, _ := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
			cur.Status = StatusWaitingInput
			cur.Question = "Continue this work in chat, then call complete_work."
			cur.LeaseID = ""
			cur.LeaseUntil = nil
			cur.UpdatedAt = e.now()
			return true
		})
		return out, nil
	}
}

func (e *Engine) finish(t Task, leaseID, result string) (Task, error) {
	out, ok := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
		cur.Status = StatusSucceeded
		cur.Result = result
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = e.now()
		for i := range cur.Plan {
			cur.Plan[i].Status = "succeeded"
		}
		return true
	})
	if !ok {
		return t, lostLeaseError{}
	}
	e.notify(inbox.KindWork, out.Title, result, out.ID, "task-done:"+out.ID)
	return out, nil
}

func (e *Engine) runDocument(ctx context.Context, t Task, leaseID string, guard func() error, checkpoint func(func(*Task)) error) (Task, error) {
	_ = ctx
	sourcePath := strMap(t.Input["path"])
	messageID := strMap(t.Input["message_id"])
	account := strMap(t.Input["account"])
	if sourcePath == "" {
		sourcePath = strMap(t.State["source_path"])
	}
	if sourcePath == "" && messageID != "" && e.Connectors != nil && account != "" {
		items, err := e.Connectors.Read(account, messageID)
		if err != nil {
			return t, err
		}
		if len(items) > 0 {
			if p := items[0]["path"]; p != "" {
				sourcePath = p
			}
			t.State["mail_from"] = firstNonEmpty(items[0]["from"], items[0]["sender"])
			t.State["mail_subject"] = items[0]["subject"]
			t.State["mail_thread"] = items[0]["thread_id"]
			t.State["mail_id"] = firstNonEmpty(items[0]["id"], messageID)
			if at := items[0]["attachments"]; at != "" {
				sourcePath = firstNonEmpty(sourcePath, strings.Split(at, ",")[0])
			}
		}
	}
	if sourcePath == "" {
		out, ok := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
			cur.Status = StatusWaitingInput
			cur.Question = "Need a workspace path to the form (or a mail attachment). Reply with path=..."
			cur.LeaseID = ""
			cur.LeaseUntil = nil
			cur.UpdatedAt = e.now()
			return true
		})
		if !ok {
			return t, lostLeaseError{}
		}
		return out, nil
	}
	abs := sourcePath
	if e.Workspace != "" && !filepath.IsAbs(abs) {
		abs = filepath.Join(e.Workspace, abs)
	}
	fields := map[string]string{}
	switch v := t.Input["fields"].(type) {
	case map[string]string:
		fields = v
	case map[string]any:
		for k, x := range v {
			fields[k] = fmt.Sprint(x)
		}
	}
	if len(fields) == 0 {
		names := fieldNames(abs)
		if len(names) == 0 {
			names = []string{"name", "date", "notes"}
		}
		out, ok := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
			cur.Status = StatusWaitingInput
			cur.Question = "Enter the form values you want to use. Supported fields: " + strings.Join(names, ", ") + ". The original file will stay intact. Reply as JSON."
			cur.State["source_path"] = sourcePath
			cur.State["missing_fields"] = names
			cur.LeaseID = ""
			cur.LeaseUntil = nil
			cur.UpdatedAt = e.now()
			return true
		})
		if !ok {
			return t, lostLeaseError{}
		}
		return out, nil
	}
	filled := strMap(t.State["filled_path"])
	if filled == "" {
		if err := guard(); err != nil {
			return t, err
		}
		dir := e.filesDir(t.ID)
		parent := abs
		filledAbs := filepath.Join(dir, "filled.pdf")
		body := formatFields(fields)
		if err := office.Create(filledAbs, office.CreateReq{Kind: office.KindPDF, Title: "Filled copy", Body: body}); err != nil {
			return t, err
		}
		meta, _ := json.MarshalIndent(fields, "", "  ")
		_ = os.WriteFile(filepath.Join(dir, "values.json"), meta, 0o644)
		filled = filledAbs
		art := Artifact{
			ID: e.newID("art"), TaskID: t.ID, Kind: "document", Title: "Filled copy",
			Summary: "Original intact at " + parent, Path: filled, Parent: parent,
			CreatedAt: e.now(),
		}
		if err := checkpoint(func(cur *Task) {
			cur.State["source_path"] = sourcePath
			cur.State["filled_path"] = filled
			cur.State["parent_path"] = parent
			cur.ArtifactIDs = append(cur.ArtifactIDs, art.ID)
			if len(cur.Plan) > 1 {
				cur.Plan[0].Status = "succeeded"
				cur.Plan[1].Status = "succeeded"
			}
		}); err != nil {
			return t, err
		}
		e.store.putArtifact(art)
	}
	to := strMap(t.State["mail_from"])
	if to == "" {
		to = strMap(t.Input["to"])
	}
	subj := strMap(t.State["mail_subject"])
	if subj == "" {
		subj = "Completed form"
	}
	if !strings.HasPrefix(strings.ToLower(subj), "re:") {
		subj = "Re: " + subj
	}
	body := strMap(t.Input["reply"])
	if body == "" {
		body = "Hello,\n\nPlease find the completed form attached.\n\nThank you."
	}
	acct := account
	if acct == "" && e.Connectors != nil {
		for _, a := range e.Connectors.List() {
			if string(a.Kind) == "mail" {
				acct = a.ID
				break
			}
		}
	}
	prop, err := e.Propose(ProposalEmailSend, map[string]any{
		"to": to, "subject": subj, "body": body,
		"attachments": []string{filled},
		"thread_id":   strMap(t.State["mail_thread"]),
		"reply_to":    strMap(t.State["mail_id"]),
	}, acct, t.ID, "document-reply", "document-reply:"+t.ID)
	if err != nil {
		return t, err
	}
	out, ok := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
		cur.Status = StatusWaitingApproval
		cur.ActionID = prop.ID
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = e.now()
		if len(cur.Plan) > 3 {
			cur.Plan[2].Status = "succeeded"
			cur.Plan[3].Status = "waiting"
		}
		return true
	})
	if !ok {
		return t, lostLeaseError{}
	}
	return out, nil
}

func fieldNames(path string) []string {
	text, err := office.Query(path)
	if err != nil {
		return nil
	}
	var names []string
	seen := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(v) == "" {
			k = strings.TrimSpace(k)
			if k != "" && !seen[k] {
				seen[k] = true
				names = append(names, k)
			}
		}
	}
	return names
}

func formatFields(fields map[string]string) string {
	var b strings.Builder
	b.WriteString("Filled copy. Original file was not modified.\n\n")
	for k, v := range fields {
		fmt.Fprintf(&b, "%s: %s\n", k, v)
	}
	return b.String()
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

var pricePat = regexp.MustCompile(`(?:\$|USD)\s*(\d+(?:,\d{3})*(?:\.\d{1,2})?)`)

func matchesPrice(text string, threshold float64) bool {
	for _, m := range pricePat.FindAllStringSubmatch(text, -1) {
		n, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64)
		if err == nil && n < threshold {
			return true
		}
	}
	return false
}

func (e *Engine) runMonitor(ctx context.Context, t Task, leaseID string, guard func() error, checkpoint func(func(*Task)) error) (Task, error) {
	_ = checkpoint
	id := strMap(t.Input["monitorId"])
	m, ok := e.store.monitor(id)
	if !ok {
		return t, errf("monitor not found")
	}
	if m.Status != "active" {
		st := StatusCancelled
		if m.Status == "paused" {
			st = StatusPaused
		}
		out, _ := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
			cur.Status = st
			cur.LeaseID = ""
			cur.LeaseUntil = nil
			cur.UpdatedAt = e.now()
			return true
		})
		return out, nil
	}
	text, title, err := e.observe(ctx, m)
	if err != nil {
		return e.monitorFail(t, m, leaseID, err)
	}
	textFlat := strings.Join(strings.Fields(text), " ")
	currentHash := shaText(textFlat)
	previousHash := strMap(t.State["lastHash"])
	if previousHash == "" {
		previousHash = m.LastHash
	}
	var matched bool
	switch m.Condition {
	case "contains":
		matched = strings.Contains(strings.ToLower(textFlat), strings.ToLower(m.Value))
	case "price_below":
		n, _ := strconv.ParseFloat(m.Value, 64)
		matched = matchesPrice(textFlat, n)
	default:
		matched = previousHash != "" && previousHash != currentHash
	}
	previouslyMatched := false
	if v, ok := t.State["matched"].(bool); ok {
		previouslyMatched = v
	}
	change := m.Condition == "change"
	lines := []string{}
	timelessHash := ""
	if change {
		lines = pageLines(text, 0)
		timelessHash = shaText(withoutRelativeTimes(textFlat))
	}
	var baseline *MonitorPage
	if matched && change {
		if p, ok := e.store.page(m.ID); ok && p.Hash == previousHash {
			baseline = &p
		}
	}
	quiet := baseline != nil && baseline.TimelessHash != "" && baseline.TimelessHash == timelessHash
	seen := false
	switch v := t.State["baseline"].(type) {
	case bool:
		seen = v
	}
	shouldNotify := matched && !quiet && seen && (m.Condition == "change" || !previouslyMatched)
	var changes string
	var count string
	if baseline != nil && shouldNotify {
		diff := diffPage(baseline.Lines, lines)
		changes = describePageDiff(diff)
		count = countPageDiff(diff)
	}
	alertSeq := 0
	if v, ok := t.State["alertSequence"].(float64); ok {
		alertSeq = int(v)
	} else if v, ok := t.State["alertSequence"].(int); ok {
		alertSeq = v
	}
	if shouldNotify {
		alertSeq++
	}
	next := e.now().Add(time.Duration(m.IntervalMinutes) * time.Minute)
	if err := guard(); err != nil {
		return t, err
	}
	_, ok = e.store.casMonitor(m.ID, "active", m.Checks, func(cur *Monitor) bool {
		cur.Checks = m.Checks + 1
		cur.LastCheckedAt = e.now()
		cur.LastHash = currentHash
		cur.LastValue = clip(textFlat, 1000)
		cur.NextCheckAt = next
		cur.Error = ""
		return true
	})
	if !ok {
		return t, lostLeaseError{}
	}
	if change {
		e.store.putPage(MonitorPage{ID: m.ID, Hash: currentHash, TimelessHash: timelessHash, Lines: lines})
	}
	result := "Watching. I'll check again on schedule."
	var notice map[string]any
	if shouldNotify {
		body := "Condition met at " + m.URL + ": " + clip(textFlat, 240)
		if changes != "" {
			body = "Changed at " + m.URL + "\n" + changes
		}
		result = "Change found. A notification is ready."
		if count != "" {
			result = "Change found: " + count + ". A notification is ready."
		}
		key := fmt.Sprintf("monitor:%s:%d:%s", m.ID, alertSeq, currentHash)
		e.notify(inbox.KindWatch, m.Title, body, t.ID, key)
		notice = map[string]any{"title": m.Title, "body": body, "key": key}
	}
	out, ok := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
		cur.Status = StatusScheduled
		cur.NextRunAt = &next
		cur.Result = result
		cur.Error = ""
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = e.now()
		cur.State["lastHash"] = currentHash
		cur.State["matched"] = matched
		cur.State["alertSequence"] = alertSeq
		cur.State["failures"] = 0
		cur.State["baseline"] = true
		cur.State["notice"] = notice
		cur.Evidence = []Evidence{{ID: m.ID, Kind: "web", Title: title, URL: m.URL, Excerpt: clip(textFlat, 600)}}
		for i := range cur.Plan {
			cur.Plan[i].Status = "succeeded"
		}
		return true
	})
	if !ok {
		return t, lostLeaseError{}
	}
	return out, nil
}

func (e *Engine) observe(ctx context.Context, m Monitor) (text, title string, err error) {
	if strings.HasPrefix(m.URL, "sample://") {
		key := strings.TrimPrefix(m.URL, "sample://")
		text = e.store.sampleGet(key)
		if text == "" {
			text = "No tables available. Check again later."
		}
		return text, "Sample " + key, nil
	}
	fn := e.Fetch
	if fn == nil {
		fn = SafeGET
	}
	body, err := fn(ctx, m.URL)
	if err != nil {
		return "", "", err
	}
	return stripTags(body), m.Title, nil
}

func stripTags(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		if r == '<' {
			in = true
			continue
		}
		if r == '>' {
			in = false
			b.WriteByte(' ')
			continue
		}
		if !in {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (e *Engine) monitorFail(t Task, m Monitor, leaseID string, err error) (Task, error) {
	failures := 0
	if v, ok := t.State["failures"].(float64); ok {
		failures = int(v)
	} else if v, ok := t.State["failures"].(int); ok {
		failures = v
	}
	failures++
	streak := 0
	if v, ok := t.State["failureStreak"].(float64); ok {
		streak = int(v)
	} else if v, ok := t.State["failureStreak"].(int); ok {
		streak = v
	}
	if failures == 1 {
		streak++
	}
	delay := time.Duration(1<<failures) * time.Minute
	if delay > time.Hour {
		delay = time.Hour
	}
	next := e.now().Add(delay)
	e.store.casMonitor(m.ID, "active", -1, func(cur *Monitor) bool {
		cur.Error = err.Error()
		cur.NextCheckAt = next
		if failures >= 5 {
			cur.Status = "paused"
		}
		return true
	})
	st := StatusScheduled
	if failures >= 5 {
		st = StatusPaused
	}
	key := fmt.Sprintf("watch-error:%s:%d:%s", t.ID, streak, map[bool]string{true: "paused", false: "retry"}[failures >= 5])
	e.notify(inbox.KindWatch, "Watch needs attention", err.Error(), t.ID, key)
	out, ok := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
		cur.Status = st
		cur.Error = err.Error()
		cur.NextRunAt = &next
		cur.LeaseID = ""
		cur.LeaseUntil = nil
		cur.UpdatedAt = e.now()
		cur.State["failures"] = failures
		cur.State["failureStreak"] = streak
		return true
	})
	if !ok {
		return t, lostLeaseError{}
	}
	return out, nil
}

func (e *Engine) runFinance(t Task, leaseID string, checkpoint func(func(*Task)) error) (Task, error) {
	csv := strMap(t.Input["csv"])
	if csv == "" {
		p := strMap(t.Input["path"])
		if p != "" && e.Workspace != "" && !filepath.IsAbs(p) {
			p = filepath.Join(e.Workspace, p)
		}
		if p != "" {
			b, err := os.ReadFile(p)
			if err != nil {
				return t, err
			}
			csv = string(b)
		}
	}
	if csv == "" {
		out, ok := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
			cur.Status = StatusWaitingInput
			cur.Question = "Paste the transaction CSV (date,description,amount,category) or a workspace path."
			cur.LeaseID = ""
			cur.LeaseUntil = nil
			cur.UpdatedAt = e.now()
			return true
		})
		if !ok {
			return t, lostLeaseError{}
		}
		return out, nil
	}
	rep, err := analyzeSpending(csv)
	if err != nil {
		return t, err
	}
	raw, _ := json.MarshalIndent(rep, "", "  ")
	dir := e.filesDir(t.ID)
	path := filepath.Join(dir, "spending.json")
	_ = os.WriteFile(path, raw, 0o644)
	art := Artifact{
		ID: e.newID("art"), TaskID: t.ID, Kind: "finance", Title: "Spending tracker",
		Summary: fmt.Sprintf("%d transactions · %.2f spent. Imported data only.", rep.Count, rep.Spending),
		Path:    path, Data: map[string]any{"income": rep.Income, "spending": rep.Spending, "saved": rep.Saved, "count": rep.Count},
		CreatedAt: e.now(),
	}
	e.store.putArtifact(art)
	_ = checkpoint
	return e.finish(t, leaseID, art.Summary)
}

func (e *Engine) runPlan(t Task, leaseID string, checkpoint func(func(*Task)) error) (Task, error) {
	_ = checkpoint
	var titles []string
	switch v := t.Input["milestones"].(type) {
	case []string:
		titles = v
	case []any:
		for _, x := range v {
			if s := strings.TrimSpace(fmt.Sprint(x)); s != "" {
				titles = append(titles, s)
			}
		}
	}
	goalID := t.GoalID
	if goalID == "" {
		goalID = strMap(t.Input["goal_id"])
	}
	g, gok := e.store.goal(goalID)
	if len(titles) == 0 && gok {
		for _, m := range g.Milestones {
			titles = append(titles, m.Title)
		}
	}
	if len(titles) == 0 {
		out, ok := e.store.casTask(t.ID, string(StatusRunning), leaseID, func(cur *Task) bool {
			cur.Status = StatusWaitingInput
			cur.Question = "Reply with milestone titles, one per line."
			cur.LeaseID = ""
			cur.LeaseUntil = nil
			cur.UpdatedAt = e.now()
			return true
		})
		if !ok {
			return t, lostLeaseError{}
		}
		return out, nil
	}
	if gok {
		g.Milestones = nil
		for i, title := range titles {
			g.Milestones = append(g.Milestones, Milestone{ID: fmt.Sprintf("m%d", i+1), Title: clip(title, 200)})
		}
		e.store.putGoal(g)
	}
	art := Artifact{
		ID: e.newID("art"), TaskID: t.ID, Kind: "plan", Title: "Plan",
		Summary:   fmt.Sprintf("%d milestones", len(titles)),
		Data:      map[string]any{"milestones": titles, "goal_id": goalID},
		CreatedAt: e.now(),
	}
	e.store.putArtifact(art)
	return e.finish(t, leaseID, art.Summary)
}
