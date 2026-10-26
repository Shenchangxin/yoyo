// 本文件拥有成就解锁、统计，以及按用户这句话从人设事实和材料里挑一段。
// reachDue 决定偶像能不能主动开口。不调用模型。

package idol

import (
	"strings"
	"time"
	"unicode"
)

// unlock 按已有对话、礼物和好感补成就。每个新成就加一次奖励抽卡；intimate 再加一张重置卡。返回这次新加上的 id。
// now 没有被读取，只赋给空白标识符。
func (st *State) unlock(now time.Time) []string {
	have := map[string]bool{}
	for _, id := range st.Achievements {
		have[id] = true
	}
	var add []string
	grant := func(id string, reset bool) {
		if have[id] {
			return
		}
		have[id] = true
		st.Achievements = append(st.Achievements, id)
		add = append(add, id)
		st.BonusDraws++
		if reset {
			st.ResetCards++
		}
	}
	turns := 0
	idols := map[string]bool{}
	days := map[string]bool{}
	gifts := 0
	for _, c := range st.Conversations {
		for _, r := range c.Records {
			if r.Kind == "turn" {
				turns++
				idols[c.IdolID] = true
				days[r.At.In(st.User.Location()).Format("2006-01-02")] = true
			}
			if r.GiftID != "" {
				gifts++
			}
		}
	}
	if turns >= 1 {
		grant("first_hello", false)
	}
	if turns >= 10 {
		grant("ten_turns", false)
	}
	if gifts >= 1 {
		grant("first_gift", false)
	}
	if len(idols) >= 3 {
		grant("three_idols", false)
	}
	if len(days) >= 3 {
		grant("three_days", false)
	}
	for _, a := range st.Affinity {
		if TierOf(a.Score) == TierClose || TierOf(a.Score) == TierIntimate {
			grant("close:"+a.IdolID, false)
		}
		if TierOf(a.Score) == TierIntimate {
			grant("intimate:"+a.IdolID, true)
		}
	}
	_ = now
	return add
}

// Stats 是给界面看的计数：回合、通话、礼物、未盖过的记忆、剩余抽卡、重置卡、每位偶像的档位、成就和当天任务。
type Stats struct {
	Turns        int               `json:"turns"`
	CallSec      int               `json:"call_sec"`
	Calls        int               `json:"calls"`
	Gifts        int               `json:"gifts"`
	Memories     int               `json:"memories"`
	DrawsLeft    int               `json:"draws_left"`
	ResetCards   int               `json:"reset_cards"`
	Tiers        map[string]string `json:"tiers"`
	Achievements []string          `json:"achievements"`
	Tasks        DayMark           `json:"tasks"`
}

// Stats 汇总当前状态。通话只计 Connected 为真的记录。记忆跳过已被盖过的。
func (st State) Stats() Stats {
	out := Stats{
		DrawsLeft:    st.Draws + st.BonusDraws,
		ResetCards:   st.ResetCards,
		Tiers:        map[string]string{},
		Achievements: append([]string{}, st.Achievements...),
		Tasks:        st.Tasks,
	}
	for _, a := range st.Affinity {
		out.Tiers[a.IdolID] = TierOf(a.Score)
	}
	for _, m := range st.Memories {
		if m.SupersededBy == "" {
			out.Memories++
		}
	}
	for _, c := range st.Calls {
		if c.Connected {
			out.Calls++
			out.CallSec += c.Seconds
		}
	}
	for _, c := range st.Conversations {
		for _, r := range c.Records {
			if r.Kind == "turn" {
				out.Turns++
			}
			if r.GiftID != "" {
				out.Gifts++
			}
		}
	}
	return out
}

// retrieve 从偶像的事实分段和她自己的材料里挑与 query 重叠词最多的一段。没有重叠就返回空。正文长于 600 个字符时截断。
// idol 的 Facts 交给 splitSections，ID 用来挑选材料。
// lore 里 IdolID 等于 idol.ID 的 Title 和 Body 参与挑选。
func retrieve(idol Idol, lore []Lore, query string) string {
	type hit struct {
		title, body string
		score       int
	}
	var hits []hit
	add := func(title, body string) {
		body = strings.TrimSpace(body)
		if body == "" {
			return
		}
		hits = append(hits, hit{title: title, body: body, score: overlap(query, title+" "+body)})
	}
	for _, sec := range splitSections(idol.Facts) {
		add(sec.title, sec.body)
	}
	for _, doc := range lore {
		if doc.IdolID == idol.ID {
			add(doc.Title, doc.Body)
		}
	}
	if len(hits) == 0 {
		return ""
	}
	best := hits[0]
	for _, h := range hits[1:] {
		if h.score > best.score {
			best = h
		}
	}
	if best.score == 0 && len([]rune(query)) > 0 {
		// 对不上词的短句不拉材料
		if len([]rune(query)) < 8 {
			return ""
		}
	}
	if best.score == 0 {
		return ""
	}
	body := best.body
	if len([]rune(body)) > 600 {
		body = string([]rune(body)[:600])
	}
	if best.title == "" {
		return "相关材料：\n" + body
	}
	return "相关材料（" + best.title + "）：\n" + body
}

// section 是 Markdown 里一个标题和它下面的正文。
type section struct{ title, body string }

// splitSections 按以 # 开头的行切开 Markdown。
// md 按换行切开，以 # 开头的行成为新段标题。
func splitSections(md string) []section {
	lines := strings.Split(md, "\n")
	var out []section
	var cur section
	flush := func() {
		if strings.TrimSpace(cur.body) != "" || cur.title != "" {
			out = append(out, cur)
		}
	}
	for _, ln := range lines {
		if strings.HasPrefix(ln, "#") {
			flush()
			cur = section{title: strings.TrimSpace(strings.TrimLeft(ln, "#"))}
			continue
		}
		if cur.body != "" {
			cur.body += "\n"
		}
		cur.body += ln
	}
	flush()
	return out
}

// overlap 数 query 和 doc 共有多少个至少两个字符的词。
func overlap(query, doc string) int {
	q := tokens(query)
	d := tokens(doc)
	if len(q) == 0 || len(d) == 0 {
		return 0
	}
	set := map[string]bool{}
	for _, t := range d {
		set[t] = true
	}
	n := 0
	for _, t := range q {
		if set[t] {
			n++
		}
	}
	return n
}

// tokens 把字母和数字收成词，丢掉短于两个字符的，并转成小写。
func tokens(s string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		w := strings.ToLower(cur.String())
		if len([]rune(w)) >= 2 {
			out = append(out, w)
		}
		cur.Reset()
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

// reachDue 判断现在该不该让这位偶像先开口。总开关 ReachOut 关掉、冷却中、cold、distant、夜间、当天已有 3 次、没有进行中的对话、上一次还没得到回复，都不开口。
// trust 间隔 36 小时，close 是 18 小时，intimate 是 8 小时。距上次超过 14 天也不再开口。
// idol 的 ID 传给 affinityOf 和 currentOf，并与 Reaches 的 IdolID 比较。
// now 传给 Cooling、Night 和 day，并与 lastAt 相减。
func reachDue(st *State, idol Idol, now time.Time) bool {
	if !st.Services.ReachOut {
		return false
	}
	aff := st.affinityOf(idol.ID)
	if aff.Cooling(now) || aff.Tier == TierCold || aff.Tier == TierDistant {
		return false
	}
	if st.User.Night(now) {
		return false
	}
	var last ReachMark
	var have bool
	today := 0
	day := st.day(now)
	for _, r := range st.Reaches {
		if r.At.In(st.User.Location()).Format("2006-01-02") == day {
			today++
		}
		if r.IdolID == idol.ID {
			last = r
			have = true
		}
	}
	if today >= 3 {
		return false
	}
	con, ok := currentOf(st, idol.ID)
	if !ok {
		return false
	}
	turns := 0
	var lastAt time.Time
	for _, r := range con.Records {
		if r.Kind == "turn" {
			turns++
			lastAt = r.At
		}
		if r.Kind == "reach" && r.At.After(lastAt) {
			lastAt = r.At
		}
	}
	if turns == 0 || con.Ended {
		return false
	}
	if have && !last.Answered {
		return false
	}
	wait := 36 * time.Hour
	switch aff.Tier {
	case TierClose:
		wait = 18 * time.Hour
	case TierIntimate:
		wait = 8 * time.Hour
	}
	if lastAt.IsZero() {
		return false
	}
	if now.Sub(lastAt) < wait {
		return false
	}
	if now.Sub(lastAt) > 14*24*time.Hour {
		return false
	}
	return true
}
