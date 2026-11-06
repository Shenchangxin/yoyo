// 本文件拥有记忆：用户告诉偶像的、值得跨对话记住的事。
// 一轮记下之后，后台模型读这一轮的对话，用中文写下用户透露的事；写之前对照和这轮最相关的已有记忆，新的一条可以取代旧的（ADR-0014）。
// 被取代的不再被找到，但仍列在记忆页，直到用户删掉；删掉取代它的那条，它就换由更新的那条取代，或者重新成立。
// 回合开始时按这句话找出相关的几条，再补几条最近的，放进提示。同一圈子的偶像共享记忆，别人那里听来的要用转述口吻（ADR-0002）。

package idol

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// shownMemories 是写记忆时给后台模型看的已有记忆条数，编了号，免得重复写或漏掉要取代的。
	shownMemories = 20
	// exchangeHistory 是后台模型读最新一句之前的几行往来（她先开口的那句也算一行）。
	exchangeHistory = 4
	// maxNewMemories 是一轮最多写几条。
	maxNewMemories = 5
	// maxMemoryLength 是一条记忆最多留多少字。
	maxMemoryLength = 200
	// foundMemories 是一轮提示最多带几条相关的记忆。
	foundMemories = 4
	// latestMemories 是相关的之外再补几条最近的。
	latestMemories = 3
	// reachMemoryWindow 是她先开口时能当作近况提起的记忆最久多少天前写下。
	reachMemoryWindow = 14 * 24 * time.Hour
	// memoryMinScore 是一条记忆算作相关的最低分。
	memoryMinScore = 1.5
)

// newMemory 是后台模型写下的一条：正文和它取代的已有记忆编号（从 1 起）。
type newMemory struct {
	Text       string `json:"text"`
	Supersedes []int  `json:"supersedes"`
}

const memoryInstructions = `你负责替偶像「%s」记住关于用户的事。用户就是正在和她对话的人。
读下面的对话，从用户最新的一句话里找出值得长期记住的、关于用户本人的信息。

只记过后仍有意义的事。
要记的：用户自己透露的个人情况、喜好、经历、计划、身边的人和宠物、希望她记住的事；会持续几天、朋友过后会问起的近况，例如生病、考试、面试、搬家、出差。
不要记的：当天就会过去的状态（例如累、困、饿、无聊、心情不好）和一时的情绪，同一句话里还说了值得记的事时只记那件事、不记这种状态；寒暄；她说的话；她自己的设定；用户向她提的问题；用户请她在电脑上办的具体任务。

每条记忆是一句能单独看懂的中文陈述，以「用户」开头，例如「用户养了一只叫小白的猫」。
提到「明天」「下周六」这样的相对时间时，换算成具体日期写下来。

「已经记住的事」按编号列出了和这轮对话最相关的记忆，写之前先对照：
- 和已经记住的一样的，不要再写。
- 更新或修正了已经记住的事（例如搬了家、换了工作），写一条单独看也完整的新记忆，写清从什么变成了什么，例如「用户从大阪搬到了东京」，并在 supersedes 里写上被它取代的记忆的编号，可以写多个。
- 补充了已经记住的事的细节（例如说出了猫的名字），写一条把细节合进去的完整记忆，取代原来那条。
- 一件事只变了一部分的，新记忆要把仍然成立的部分也写进去，例如原来记着养了一只猫和一只狗、现在狗去世了，就写「用户的狗去世了，现在还养着那只猫」，取代原来那条。
- 两件事可以同时成立的（例如喜欢喝咖啡，最近也开始喝茶），新的照常写，不取代旧的。

只输出 JSON，格式为 {"memories": [{"text": "……", "supersedes": [编号]}]}，不取代任何记忆时 supersedes 写 []；没有值得记住的就输出 {"memories": []}。`

// memoryPrompt 拼出写记忆的消息：今天的日期、编号的已有记忆、最近两轮对话和这一轮。
func memoryPrompt(idol Idol, user UserCard, history []Record, text string, reply []Segment, shown []Memory, now time.Time) []Msg {
	name := strings.TrimSpace(idol.Name)
	if name == "" {
		name = "偶像"
	}
	local := now.In(user.Location())
	var b strings.Builder
	fmt.Fprintf(&b, "今天是 %d年%d月%d日（星期%s，用户所在时区）。\n\n# 已经记住的事\n", local.Year(), int(local.Month()), local.Day(), weekdays[local.Weekday()])
	if len(shown) == 0 {
		b.WriteString("（还没有）\n")
	}
	for i, m := range shown {
		fmt.Fprintf(&b, "%d. %s\n", i+1, m.Text)
	}
	fmt.Fprintf(&b, "\n# 用户和%s的对话\n", name)
	for _, ln := range exchangeLines(history, exchangeHistory, name) {
		b.WriteString(ln + "\n")
	}
	fmt.Fprintf(&b, "用户（最新的一句话）：%s\n%s：%s\n", text, name, CaptionLine(reply))
	return []Msg{
		{Role: "system", Content: fmt.Sprintf(memoryInstructions, name)},
		{Role: "user", Content: b.String()},
	}
}

// parseMemories 读后台模型写的 JSON。读不出时 ok 为假；空的、过长的条目在 addMemories 里处理。
func parseMemories(raw string) ([]newMemory, bool) {
	var p struct {
		Memories []newMemory `json:"memories"`
	}
	if err := json.Unmarshal([]byte(extractJSON(raw)), &p); err != nil {
		return nil, false
	}
	return p.Memories, true
}

// addMemories 记下这一轮写出的记忆，并让每条取代它点名的、给模型看过且仍成立的旧记忆。返回新加的。
// 用户已有一模一样的一条时不再加：仍成立就跳过；已被取代就让它重新成立，再去取代它点名的。
func (st *State) addMemories(idol Idol, written []newMemory, shown []Memory, now time.Time) []Memory {
	var added []Memory
	for n, w := range written {
		if n >= maxNewMemories {
			break
		}
		text := strings.TrimSpace(w.Text)
		if text == "" || strings.EqualFold(text, "NONE") {
			continue
		}
		if r := []rune(text); len(r) > maxMemoryLength {
			text = string(r[:maxMemoryLength])
		}
		id := ""
		for i := range st.Memories {
			if st.Memories[i].Text != text {
				continue
			}
			id = st.Memories[i].ID
			if st.Memories[i].SupersededBy == "" {
				id = "-"
			} else {
				st.Memories[i].SupersededBy = ""
			}
			break
		}
		if id == "-" {
			continue
		}
		if id == "" {
			m := Memory{ID: newID(), Text: text, SourceIdol: idol.ID, CircleID: idol.CircleID, At: now}
			st.Memories = append(st.Memories, m)
			added = append(added, m)
			id = m.ID
		}
		for _, k := range w.Supersedes {
			if k < 1 || k > len(shown) || shown[k-1].ID == id {
				continue
			}
			for i := range st.Memories {
				if st.Memories[i].ID == shown[k-1].ID && st.Memories[i].SupersededBy == "" {
					st.Memories[i].SupersededBy = id
				}
			}
		}
	}
	return added
}

// holding 返回这位偶像看得见、仍成立的记忆：她自己的，以及同一圈子里别人的。按写下的先后排。ownOnly 为真时只要她自己的。
func (st *State) holding(idol Idol, ownOnly bool) []Memory {
	var out []Memory
	for _, m := range st.Memories {
		if m.SupersededBy != "" {
			continue
		}
		own := m.SourceIdol == idol.ID
		shared := !ownOnly && idol.CircleID != "" && m.CircleID == idol.CircleID
		if own || shared {
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out
}

// shownFor 是写记忆时给后台模型看的已有记忆：和 query 最相关的在前，不够 shownMemories 条时用最近的补上。
func (st *State) shownFor(idol Idol, query string) []Memory {
	all := st.holding(idol, false)
	if len(all) <= shownMemories {
		return all
	}
	texts := make([]string, len(all))
	for i, m := range all {
		texts[i] = m.Text
	}
	picked := map[int]bool{}
	var out []Memory
	for _, h := range rank(query, texts) {
		if len(out) >= shownMemories/2 {
			break
		}
		picked[h.index] = true
		out = append(out, all[h.index])
	}
	for i := len(all) - 1; i >= 0 && len(out) < shownMemories; i-- {
		if !picked[i] {
			out = append(out, all[i])
		}
	}
	return out
}

// recallMemories 是一轮提示里带的记忆：和 query 相关的最多 foundMemories 条（最相关的在前），再补最多 latestMemories 条最近的。
// 记忆不多时全部带上。
func (st *State) recallMemories(idol Idol, query string) []Memory {
	all := st.holding(idol, false)
	if len(all) <= foundMemories+latestMemories {
		return all
	}
	texts := make([]string, len(all))
	for i, m := range all {
		texts[i] = m.Text
	}
	picked := map[int]bool{}
	var out []Memory
	for _, h := range rank(query, texts) {
		if h.score < memoryMinScore || len(out) >= foundMemories {
			break
		}
		picked[h.index] = true
		out = append(out, all[h.index])
	}
	extra := 0
	for i := len(all) - 1; i >= 0 && extra < latestMemories; i-- {
		if !picked[i] {
			out = append(out, all[i])
			extra++
		}
	}
	return out
}

// reachMemories 是她先开口时带的记忆：只要用户亲口告诉她的，和上一轮相关的在前，再补 reachMemoryWindow 内最近写下的几条。
func (st *State) reachMemories(idol Idol, query string, now time.Time) []Memory {
	own := st.holding(idol, true)
	texts := make([]string, len(own))
	for i, m := range own {
		texts[i] = m.Text
	}
	picked := map[int]bool{}
	var out []Memory
	if strings.TrimSpace(query) != "" {
		for _, h := range rank(query, texts) {
			if h.score < memoryMinScore || len(out) >= foundMemories {
				break
			}
			picked[h.index] = true
			out = append(out, own[h.index])
		}
	}
	extra := 0
	for i := len(own) - 1; i >= 0 && extra < 5; i-- {
		if picked[i] || now.Sub(own[i].At) > reachMemoryWindow {
			continue
		}
		out = append(out, own[i])
		extra++
	}
	return out
}

// formatMemories 把记忆写成提示的一节。用户亲口对她说的照直写；圈子里别人听来的标明出处，只能转述。
func formatMemories(st *State, idol Idol, mems []Memory) string {
	if len(mems) == 0 {
		return ""
	}
	names := map[string]string{}
	for _, it := range st.Idols {
		names[it.ID] = it.Name
	}
	loc := st.User.Location()
	var b strings.Builder
	b.WriteString("# 你记得的关于用户的事\n用得上时自然地接上，不要一次全说出来；别人告诉你的，只能用转述的口吻说。\n")
	for _, m := range mems {
		at := m.At.In(loc)
		when := fmt.Sprintf("%d月%d日", int(at.Month()), at.Day())
		if m.SourceIdol == idol.ID || m.SourceIdol == "" {
			fmt.Fprintf(&b, "- （%s，用户亲口对你说的）%s\n", when, m.Text)
			continue
		}
		name := names[m.SourceIdol]
		if name == "" {
			name = "另一位"
		}
		fmt.Fprintf(&b, "- （%s，听%s说的，不是用户亲口告诉你的）%s\n", when, name, m.Text)
	}
	return strings.TrimRight(b.String(), "\n")
}

// deleteMemory 删掉一条记忆。被它取代的那些改由取代它的那条取代；它没被取代时，那些重新成立。找不到时 ok 为假。
func (st *State) deleteMemory(id string) bool {
	idx := -1
	for i := range st.Memories {
		if st.Memories[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	next := st.Memories[idx].SupersededBy
	for i := range st.Memories {
		if st.Memories[i].SupersededBy == id {
			st.Memories[i].SupersededBy = next
		}
	}
	st.Memories = append(st.Memories[:idx], st.Memories[idx+1:]...)
	return true
}
