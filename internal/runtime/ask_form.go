package runtime

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	askNoOperator = "no operator answer; continue with a reasonable default"
	askIdleNote   = "note: operator idle; used recommended"
	askAnswerNote = "note: operator answered"
)

const askContinueNudge = "You ended the turn to wait for a nod. Do not. Call ask_user now with questions, each having id, prompt, options (id and label), and recommended (the option id you would pick). One card sits above the composer; if the operator does not touch it, this turn continues with recommended. After the tool result, keep going. Do not write another letter that waits for a new message."

const askContinueNudgeZH = "你用结束本轮来等用户点头。不要这样。现在调用 ask_user：questions 里每题都要有 id、prompt、options（id 和 label）和 recommended（你认为最好的选项 id）。同一张卡片停在输入框上方，用户不操作就用 recommended 在本轮继续。工具返回后接着做，不要再写一封等新消息的信。"

// AskOption is one choice on the composer question card.
type AskOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// AskQuestion is one page of that card. Recommended is the option used when
// the operator never touches the card.
type AskQuestion struct {
	ID          string      `json:"id"`
	Prompt      string      `json:"prompt"`
	Options     []AskOption `json:"options,omitempty"`
	Recommended string      `json:"recommended,omitempty"`
}

// AskForm is the payload ask_user blocks on. The desktop shows every question
// in one card and pages them with previous/next.
type AskForm struct {
	Title     string        `json:"title,omitempty"`
	Questions []AskQuestion `json:"questions"`
}

func parseAskForm(args map[string]any) AskForm {
	form := AskForm{Title: strings.TrimSpace(str(args["title"]))}
	form.Questions = questionsOf(args["questions"])
	if len(form.Questions) == 0 {
		prompt := strings.TrimSpace(str(args["question"]))
		if prompt != "" {
			form.Questions = []AskQuestion{{ID: "q1", Prompt: prompt}}
		}
	}
	normalizeAskForm(&form)
	return form
}

func questionsOf(raw any) []AskQuestion {
	if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
		var parsed any
		if json.Unmarshal([]byte(s), &parsed) == nil {
			raw = parsed
		}
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]AskQuestion, 0, len(list))
	for i, item := range list {
		switch v := item.(type) {
		case map[string]any:
			q := AskQuestion{
				ID:          strings.TrimSpace(str(v["id"])),
				Prompt:      strings.TrimSpace(str(v["prompt"])),
				Recommended: strings.TrimSpace(str(v["recommended"])),
			}
			if q.Prompt == "" {
				q.Prompt = strings.TrimSpace(str(v["question"]))
			}
			q.Options = optionsOf(v["options"])
			if q.ID == "" {
				q.ID = fmt.Sprintf("q%d", i+1)
			}
			if q.Prompt != "" {
				out = append(out, q)
			}
		case string:
			prompt := strings.TrimSpace(v)
			if prompt != "" {
				out = append(out, AskQuestion{ID: fmt.Sprintf("q%d", i+1), Prompt: prompt})
			}
		}
	}
	return out
}

func optionsOf(raw any) []AskOption {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]AskOption, 0, len(list))
	for i, item := range list {
		switch v := item.(type) {
		case map[string]any:
			id := strings.TrimSpace(str(v["id"]))
			label := strings.TrimSpace(str(v["label"]))
			if label == "" {
				label = id
			}
			if id == "" {
				id = fmt.Sprintf("%d", i+1)
			}
			if label == "" {
				continue
			}
			out = append(out, AskOption{ID: id, Label: label})
		case string:
			label := strings.TrimSpace(v)
			if label == "" {
				continue
			}
			out = append(out, AskOption{ID: fmt.Sprintf("%d", i+1), Label: label})
		}
	}
	return out
}

func normalizeAskForm(form *AskForm) {
	if form == nil {
		return
	}
	if len(form.Questions) > 6 {
		form.Questions = form.Questions[:6]
	}
	for i := range form.Questions {
		q := &form.Questions[i]
		if q.ID == "" {
			q.ID = fmt.Sprintf("q%d", i+1)
		}
		if len(q.Options) > 8 {
			q.Options = q.Options[:8]
		}
		if len(q.Options) == 0 {
			q.Recommended = ""
			continue
		}
		ok := false
		for _, o := range q.Options {
			if o.ID == q.Recommended {
				ok = true
				break
			}
		}
		if !ok {
			q.Recommended = q.Options[0].ID
		}
	}
}

func (t *WorkspaceTools) askForm(form AskForm) ToolResult {
	normalizeAskForm(&form)
	if len(form.Questions) == 0 {
		return ToolResult{Err: fmt.Errorf("ask_user: empty question")}
	}
	payload, err := json.Marshal(form)
	if err != nil {
		return ToolResult{Err: err}
	}
	if t == nil || t.AskUser == nil {
		return ToolResult{Content: AskIdleAnswer(string(payload))}
	}
	ans, err := t.AskUser(string(payload))
	return ToolResult{Content: ans, Err: err}
}

// AskPrompt is the short text stored on the ask event. The card reads the
// JSON command, not this line.
func AskPrompt(payload string) string {
	form, ok := decodeAskPayload(payload)
	if !ok {
		return strings.TrimSpace(payload)
	}
	if title := strings.TrimSpace(form.Title); title != "" {
		return title
	}
	if len(form.Questions) > 0 {
		return form.Questions[0].Prompt
	}
	return strings.TrimSpace(payload)
}

// AskIdleAnswer is the tool result when nobody is there to touch the card,
// or the desktop idle timer fires with the recommended options.
func AskIdleAnswer(payload string) string {
	form, ok := decodeAskPayload(payload)
	if !ok || !askCanIdle(form) {
		return askNoOperator
	}
	return formatAsk(form, true)
}

func askCanIdle(form AskForm) bool {
	if len(form.Questions) == 0 {
		return false
	}
	for _, q := range form.Questions {
		if len(q.Options) == 0 {
			return false
		}
	}
	return true
}

func formatAsk(form AskForm, idle bool) string {
	var b strings.Builder
	for _, q := range form.Questions {
		id := q.Recommended
		label := ""
		for _, o := range q.Options {
			if o.ID == id {
				label = o.Label
				break
			}
		}
		if label == "" && len(q.Options) > 0 {
			id = q.Options[0].ID
			label = q.Options[0].Label
		}
		if len(q.Options) == 0 {
			fmt.Fprintf(&b, "%s: continue with your best judgment\n", q.ID)
			continue
		}
		if label != "" {
			fmt.Fprintf(&b, "%s: %s — %s\n", q.ID, id, label)
		} else {
			fmt.Fprintf(&b, "%s: %s\n", q.ID, id)
		}
	}
	if idle {
		b.WriteString(askIdleNote)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

func decodeAskPayload(payload string) (AskForm, bool) {
	payload = strings.TrimSpace(payload)
	if !strings.HasPrefix(payload, "{") {
		return AskForm{}, false
	}
	var form AskForm
	if json.Unmarshal([]byte(payload), &form) != nil || len(form.Questions) == 0 {
		return AskForm{}, false
	}
	normalizeAskForm(&form)
	return form, true
}

// chosenOption finds the option id the card wrote into an ask_user result.
func chosenOption(answer string, ids []string) string {
	for _, line := range strings.Split(answer, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "note:") {
			continue
		}
		_, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		rest = strings.TrimSpace(rest)
		id, _, _ := strings.Cut(rest, " — ")
		id = strings.TrimSpace(id)
		for _, want := range ids {
			if id == want {
				return want
			}
		}
	}
	return ""
}

func waitsForNod(text string) bool {
	s := strings.ToLower(text)
	if strings.TrimSpace(s) == "" {
		return false
	}
	for _, needle := range nodNeedles {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

var nodNeedles = []string{
	"请确认",
	"点头",
	"等你确认",
	"等你回复",
	"等你决定",
	"等你点头",
	"是否接受",
	"please confirm",
	"wait for your",
	"waiting for your",
	"once you approve",
	"before i write",
	"before i start",
	"before i implement",
	"shall i proceed",
	"do you want me to proceed",
	"let me know which",
	"your go-ahead",
	"your approval",
}
