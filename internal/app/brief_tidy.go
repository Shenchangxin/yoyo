// brief_tidy.go 让每日简报那一次模型顺手判断相近的记忆是同一件事、改了口，还是两件事，并把等人点头的建议写成简报里的卡。
// 拥有：附在简报材料后面的那一段（tidyAsk）、从模型答复里取出判断（takeTidy）、建议卡（tidyCards）、人点「对」「不对」（MemoryTidy）。
// 不拥有：挑哪几对、建议怎么存、点了「对」怎么并（memory/tidy.go）。模型的判断从不直接改记忆。

package app

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/memory"
)

const (
	// tidyAskMax 是一封简报最多顺手问几对。
	tidyAskMax = 4
	// tidyCardMax 是一封简报里最多放几张整理建议卡。
	tidyCardMax = 2
)

// tidyAsk 是附在简报材料后面的「记忆整理」一段，以及段里每个编号对应的那一对。不到时候（见 memory.TidyPairs）时两个都是空的。
func (a *App) tidyAsk() (string, map[string]memory.Pair) {
	if a == nil || a.Memory == nil {
		return "", nil
	}
	pairs := a.Memory.TidyPairs(tidyAskMax)
	if len(pairs) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("\n\n记忆整理。下面每一对是记忆里相近、但没有合并的两条，方括号里是这一对的编号，「早」是先记下的，「晚」是后来的。逐对判断：\n")
	b.WriteString("same：同一件事换了说法。text 写一句合并后的说法，只用这两句里有的内容，用原话的语言。\n")
	b.WriteString("change：后来改了口，只该留「晚」那一条。\n")
	b.WriteString("different：两件不同的事，或者看不出来。拿不准就写 different。\n")
	b.WriteString(`判断写进同一个 JSON 对象的 memory 数组：{"cards":[...],"memory":[{"pair":"t1","verdict":"same","text":""}]}。不要为记忆整理写卡，这些编号也不写进 refs。` + "\n")
	byID := map[string]memory.Pair{}
	for i, p := range pairs {
		id := "t" + strconv.Itoa(i+1)
		byID[id] = p
		fmt.Fprintf(&b, "[%s] 早（%s）%s\n     晚（%s）%s\n", id,
			p.Old.Touched().Local().Format("2006-01-02"), clipBrief(oneLineBrief(p.Old.Text), 240),
			p.New.Touched().Local().Format("2006-01-02"), clipBrief(oneLineBrief(p.New.Text), 240))
	}
	return b.String(), byID
}

// takeTidy 把模型对这几对的判断交给记忆库，存成等人点头的建议。模型没给判断时也记下问过了，攒够新记忆再问。
func (a *App) takeTidy(pairs map[string]memory.Pair, text string) {
	if a == nil || a.Memory == nil || len(pairs) == 0 {
		return
	}
	asked := make([]memory.Pair, 0, len(pairs))
	for _, p := range pairs {
		asked = append(asked, p)
	}
	a.Memory.Suggest(asked, parseTidy(text, pairs))
}

// parseTidy 从简报答复里取出 memory 数组。认不出的编号、重复的编号丢掉。
func parseTidy(text string, pairs map[string]memory.Pair) []memory.Verdict {
	type rawVerdict struct {
		Pair    string `json:"pair"`
		Verdict string `json:"verdict"`
		Text    string `json:"text"`
	}
	var found []rawVerdict
	for _, candidate := range jsonCandidates(text) {
		var body struct {
			Memory []rawVerdict `json:"memory"`
		}
		if json.Unmarshal([]byte(candidate), &body) == nil && body.Memory != nil {
			found = body.Memory
			break
		}
	}
	var out []memory.Verdict
	seen := map[string]bool{}
	for _, v := range found {
		id := strings.Trim(strings.TrimSpace(v.Pair), "[]")
		p, ok := pairs[id]
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, memory.Verdict{Pair: p, Verdict: v.Verdict, Text: v.Text})
	}
	return out
}

// tidyCards 把等人点头的整理建议写成简报里的卡，先提的在前，最多 tidyCardMax 张。
func (a *App) tidyCards(zh bool) []BriefCard {
	if a == nil || a.Memory == nil {
		return nil
	}
	var out []BriefCard
	for _, sug := range a.Memory.Suggestions() {
		if len(out) == tidyCardMax {
			break
		}
		out = append(out, tidyCard(sug, zh))
	}
	return out
}

// tidyCard 是一条建议的卡：左右两句原话（evidence，不能点），「对」「不对」两个按钮。Tidy 让界面知道按钮直接去处理这条建议，不放进输入框。
func tidyCard(sug memory.Suggestion, zh bool) BriefCard {
	card := BriefCard{
		ID:     sug.ID,
		Tidy:   sug.ID,
		Layout: "evidence",
		Kicker: pick(zh, "记忆整理", "Memory"),
		Items: []BriefBit{
			{Title: pick(zh, "以前", "Earlier"), Detail: clipBrief(sug.OldText, 240)},
			{Title: pick(zh, "后来", "Later"), Detail: clipBrief(sug.NewText, 240)},
		},
		Choices:   []BriefBit{},
		Primary:   pick(zh, "对", "Yes"),
		Secondary: pick(zh, "不对", "No"),
		Note:      pick(zh, "以前的说法放进历史，记忆页能撤销。点「不对」以后不再问这两条。", "The old wording stays in history and can be undone on the Memory page. No means this pair is not asked again."),
		Refs: []BriefRef{
			{ID: "old", Kind: "memory", Label: clipBrief(oneLineBrief(sug.OldText), 40), Target: sug.Old},
			{ID: "new", Kind: "memory", Label: clipBrief(oneLineBrief(sug.NewText), 40), Target: sug.New},
		},
	}
	if sug.Verdict == memory.VerdictSame {
		merged := strings.TrimSpace(sug.Text)
		if merged == "" {
			merged = sug.NewText
		}
		merged = clipBrief(oneLineBrief(merged), 120)
		card.Title = pick(zh, "这两条记忆像是同一件事", "These two memories look like the same thing")
		card.Lede = pick(zh, "点「对」并成一条：「"+merged+"」。", "Yes merges them into: “"+merged+"”.")
	} else {
		card.Title = pick(zh, "你像是改了主意", "Looks like you changed your mind")
		card.Lede = pick(zh, "点「对」只留后来的说法。", "Yes keeps only the later one.")
	}
	return card
}

// withoutTidy 去掉整理建议卡。IM 里点不了按钮，这几张只在桌面的简报里出现。
func withoutTidy(cards []BriefCard) []BriefCard {
	var out []BriefCard
	for _, c := range cards {
		if strings.TrimSpace(c.Tidy) == "" {
			out = append(out, c)
		}
	}
	return out
}

// TidyResult 是处理一条整理建议的结果。点「对」时 Item 是并好的那一条；建议已经处理过、过期或两条后来改过时 OK 为假，Error 写原因。
type TidyResult struct {
	OK    bool        `json:"ok"`
	Item  memory.Item `json:"item,omitzero"`
	Error string      `json:"error,omitempty"`
}

// MemoryTidy 处理一条整理建议：accept 为真时并成一条（旧说法进历史，可以撤销），为假时以后不再问这一对。
func (a *App) MemoryTidy(id string, accept bool) TidyResult {
	if a == nil || a.Memory == nil {
		return TidyResult{Error: "no memory"}
	}
	it, err := a.Memory.Settle(id, accept)
	if err != nil {
		return TidyResult{Error: err.Error()}
	}
	if accept {
		a.forgetPinSig()
		a.touchPoi()
	}
	return TidyResult{OK: true, Item: it}
}
