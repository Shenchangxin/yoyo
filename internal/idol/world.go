// 本文件拥有成就解锁、统计、按用户这句话从事实和材料里挑相关的几段，以及她什么时候先开口（ADR-0026）。不调用模型。

package idol

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"
)

const (
	// nightStartHour 和 nightEndHour 是用户时区里她不先开口的时段：22 点到次日 8 点。
	nightStartHour = 22
	nightEndHour   = 8
	// morningSpread 是落在夜里的开口推到早上后，最多再往后多久。
	morningSpread = 2 * time.Hour
	// reachWindow 是上次说话之后多久以内她还会先开口。
	reachWindow = 7 * 24 * time.Hour
	// reachPerDay 是一天里所有偶像加起来最多先开口几次。
	reachPerDay = 3
	// loreMinScore 是一段材料算作相关的最低分。
	loreMinScore = 1.5
	// loreSections 是一轮最多带几段材料。
	loreSections = 2
)

// reachIntervals 是上次说话之后多久她先开口，按档位：越亲近来得越早。不在表里的档位不先开口。
var reachIntervals = map[string][2]time.Duration{
	TierIntimate: {3 * time.Hour, 6 * time.Hour},
	TierClose:    {8 * time.Hour, 16 * time.Hour},
	TierTrust:    {20 * time.Hour, 36 * time.Hour},
}

// unlock 按已有对话、礼物和好感补成就。每个新成就加一次奖励抽卡；intimate 再加一张重置卡。返回这次新加上的 id。
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
		tier := a.recovered(now).Tier
		if tier == TierClose || tier == TierIntimate {
			grant("close:"+a.IdolID, false)
		}
		if tier == TierIntimate {
			grant("intimate:"+a.IdolID, true)
		}
	}
	return add
}

// Stats 是给界面看的计数：回合、通话、礼物、仍成立的记忆、剩余抽奖、重置卡、每位偶像的档位、成就和当天任务。
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

// Stats 汇总当前状态。通话只计接通的。记忆跳过已被取代的。档位按 now 补上回暖。
func (st State) Stats(now time.Time) Stats {
	out := Stats{
		DrawsLeft:    st.Draws + st.BonusDraws,
		ResetCards:   st.ResetCards,
		Tiers:        map[string]string{},
		Achievements: append([]string{}, st.Achievements...),
		Tasks:        st.Tasks,
	}
	for _, a := range st.Affinity {
		out.Tiers[a.IdolID] = a.recovered(now).Tier
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

// retrieve 从她的事实（太长、没写进稳定前缀时）和她自己的材料里挑和 query 最相关的几段，写成提示的一节。没有相关的返回空串。
func retrieve(idol Idol, lore []Lore, query string) string {
	if strings.TrimSpace(query) == "" {
		return ""
	}
	type doc struct{ title, body string }
	var docs []doc
	if facts := strings.TrimSpace(idol.Facts); len([]rune(facts)) > factsInline {
		for _, sec := range splitSections(facts) {
			if strings.TrimSpace(sec.body) != "" {
				docs = append(docs, doc{sec.title, sec.body})
			}
		}
	}
	for _, d := range lore {
		if d.IdolID == idol.ID && strings.TrimSpace(d.Body) != "" {
			docs = append(docs, doc{strings.TrimSpace(d.Title), d.Body})
		}
	}
	if len(docs) == 0 {
		return ""
	}
	texts := make([]string, len(docs))
	for i, d := range docs {
		texts[i] = d.title + "\n" + d.body
	}
	var b strings.Builder
	n := 0
	for _, h := range rank(query, texts) {
		if h.score < loreMinScore || n >= loreSections {
			break
		}
		if n == 0 {
			b.WriteString("# 和这句话有关的材料\n")
		}
		d := docs[h.index]
		if d.title != "" {
			fmt.Fprintf(&b, "## %s\n", d.title)
		}
		b.WriteString(clip(d.body, 700) + "\n")
		n++
	}
	if n == 0 {
		return ""
	}
	b.WriteString("只在用得上时提起，不要复述整段材料。")
	return b.String()
}

// section 是 Markdown 里一个标题和它下面的正文。
type section struct{ title, body string }

// splitSections 按以 # 开头的行切开 Markdown。
func splitSections(md string) []section {
	var out []section
	var cur section
	flush := func() {
		if strings.TrimSpace(cur.body) != "" || cur.title != "" {
			out = append(out, cur)
		}
	}
	for _, ln := range strings.Split(md, "\n") {
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

// reachDue 判断现在该不该让这位偶像先开口。总开关关着、冷却中、冷淡或疏远、这段对话已结束或还没说过话、
// 上次说话之后她已经先开口过（等用户回话）、超过 reachWindow、夜里、还没到她的时间、今天已经开口满 reachPerDay 次，都不开口。
// 她的时间是上次说话加上她这一档的间隔里的一个点；这个点按上次说话的时间定下，同一次说话不会变，下次说话再换一个。
func reachDue(st *State, idol Idol, now time.Time) bool {
	if !st.Services.ReachOut {
		return false
	}
	aff := st.affinityOf(idol.ID, now)
	span, ok := reachIntervals[aff.Tier]
	if aff.InCooldown || !ok {
		return false
	}
	con, ok := currentOf(st, idol.ID)
	if !ok || con.Ended || countTurns(con) == 0 {
		return false
	}
	contact := lastContactOf(st, idol.ID)
	if contact.IsZero() || now.Sub(contact) > reachWindow {
		return false
	}
	for _, r := range con.Records {
		if r.Kind == "reach" && r.At.After(contact) {
			return false
		}
	}
	if st.User.Night(now) {
		return false
	}
	due := contact.Add(span[0] + time.Duration(placeOf(idol.ID, contact, "span")*float64(span[1]-span[0])))
	due = pastNight(st.User, due, placeOf(idol.ID, contact, "morning"))
	if now.Before(due) {
		return false
	}
	today := st.day(now)
	n := 0
	for _, r := range st.Reaches {
		if st.day(r.At) == today {
			n++
		}
	}
	return n < reachPerDay
}

// placeOf 为一次说话定下 0 到 1 之间的一个点，同一位偶像、同一次说话、同一用途总是同一个点。
func placeOf(idolID string, contact time.Time, salt string) float64 {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s|%d|%s", idolID, contact.UnixNano(), salt)
	return float64(h.Sum64()%1000) / 1000
}

// pastNight 把落在用户夜里的时间推到那之后的早上 8 点，再往后 place 份的 morningSpread。
func pastNight(u UserCard, due time.Time, place float64) time.Time {
	loc := u.Location()
	local := due.In(loc)
	if local.Hour() >= nightEndHour && local.Hour() < nightStartHour {
		return due
	}
	day := local
	if local.Hour() >= nightStartHour {
		day = local.AddDate(0, 0, 1)
	}
	morning := time.Date(day.Year(), day.Month(), day.Day(), nightEndHour, 0, 0, 0, loc)
	return morning.Add(time.Duration(place * float64(morningSpread)))
}
