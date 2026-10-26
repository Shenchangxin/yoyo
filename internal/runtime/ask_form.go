// ask_user 的问题卡：解析参数、空闲时采用 recommended，并识别等用户点头的结束语。
// 不画界面。桌面用 AskUser 回调挡住工具。没有回调时走 AskIdleAnswer。

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

// AskOption 是作曲问题卡上的一个选项。
type AskOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// AskQuestion 是这张卡的一页。Recommended 是操作者没碰卡片时采用的选项。
type AskQuestion struct {
	ID          string      `json:"id"`
	Prompt      string      `json:"prompt"`
	Options     []AskOption `json:"options,omitempty"`
	Recommended string      `json:"recommended,omitempty"`
}

// AskForm 是 ask_user 阻塞时的载荷。桌面把每道题放在同一张卡里，用上一页和下一页翻页。
type AskForm struct {
	Title     string        `json:"title,omitempty"`
	Questions []AskQuestion `json:"questions"`
}

// parseAskForm 从 questions 或单个 question 解析表单，再规范化。questions 可以是 JSON 字符串。
// args 提供 title、questions，以及 questions 为空时的 question。
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

// questionsOf 解析问题列表。缺 id 时用 q 加序号。prompt 为空的对象丢掉。字符串项当成只有题干的问题。
// raw 是列表本身。字符串且去掉空白后非空时先按 JSON 解开，再要求是数组。
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

// optionsOf 解析选项。label 为空时用 id。id 为空时用从 1 起的序号。字符串项的 id 也是序号。
// raw 不是数组时返回 nil，否则逐项读对象或字符串。
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

// normalizeAskForm 最多留 6 道题，每题最多 8 个选项。recommended 不在选项里时改成第一项。没有选项则清空 recommended。
// form 为 nil 时直接返回，否则就地截断并改写 Questions。
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

// askForm 把表单交给 AskUser。没有回调时返回空闲答案。没有题目时返回错误。
// form 先规范化，题目为空则报错，否则编成 JSON 交给 AskUser 或 AskIdleAnswer。
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

// AskPrompt 是记在询问事件上的短文本。卡片读的是 JSON 命令，不是这一行。优先用标题，否则用第一题的题干。
// payload 交给 decodeAskPayload，解不开或没有可用标题和题干时返回去掉空白后的原文。
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

// AskIdleAnswer 是没人碰卡片，或桌面空闲计时器用推荐选项触发时的工具结果。
// 任一道题没有选项时返回 askNoOperator，不编造选择。
// payload 交给 decodeAskPayload，解不开或 askCanIdle 为假时返回 askNoOperator。
func AskIdleAnswer(payload string) string {
	form, ok := decodeAskPayload(payload)
	if !ok || !askCanIdle(form) {
		return askNoOperator
	}
	return formatAsk(form, true)
}

// askCanIdle 在每道题都有选项时才允许用推荐项继续。
// form 的 Questions 为空，或任一道题没有选项时返回 false。
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

// formatAsk 把每题的推荐项写成 “id: 选项 — 标签”。idle 为真时附上 askIdleNote。
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

// decodeAskPayload 解析以 { 开头的 JSON 表单。没有题目或 JSON 无效时返回 false。
// payload 去掉空白后必须以前缀 { 开头，再解成表单。
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

// chosenOption 从 ask_user 结果里找出卡片写下的选项 id。跳过 note: 行。只接受 ids 里有的 id。
// answer 按行切开，跳过空行和 note: 前缀，从冒号后取选项 id。
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

// waitsForNod 判断结束语是不是在等用户点头或确认，而不是已经做完。
// text 转成小写，去掉空白后为空则返回 false，否则看是否包含 nodNeedles。
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
