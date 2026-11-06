// brief_compose.go 用画像、记忆、最近对话、日程、工单、桌面清点和本机感知写一封简报。
// 不拥有推送，也不准备一份模板。材料没变时不再叫模型。材料里没有能写的事时，不编一封。

package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/poi"
	"github.com/Shenchangxin/yoyo/internal/superagent"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// BriefBit 是简报卡上的一行。Tone 只在 rows 上着色，取 moving、quiet、stalled。
type BriefBit struct {
	ID     string `json:"id,omitempty"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
	Tone   string `json:"tone,omitempty"`
}

// BriefStat 是 stats 版式上的一个量。Label 和 Value 都来自材料。
type BriefStat struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// BriefCard 是「今天」里的一张卡。Task 是点按钮后放进输入框的请求，不直接发出。
// Layout 为空时，界面按这张卡自己的内容选版式。Refs 是这张卡依据的材料，只留材料里真的出现过的。
type BriefCard struct {
	ID         string      `json:"id"`
	Kicker     string      `json:"kicker,omitempty"`
	Title      string      `json:"title"`
	Lede       string      `json:"lede,omitempty"`
	Layout     string      `json:"layout,omitempty"`
	Items      []BriefBit  `json:"items"`
	Choices    []BriefBit  `json:"choices"`
	Stats      []BriefStat `json:"stats,omitempty"`
	Mark       string      `json:"mark,omitempty"`
	LeftLabel  string      `json:"leftLabel,omitempty"`
	RightLabel string      `json:"rightLabel,omitempty"`
	Note       string      `json:"note,omitempty"`
	Primary    string      `json:"primary,omitempty"`
	Secondary  string      `json:"secondary,omitempty"`
	Task       string      `json:"task,omitempty"`
	Alt        string      `json:"alt,omitempty"`
	Refs       []BriefRef  `json:"refs,omitempty"`
}

// BriefRef 是一张卡依据的一条材料。Kind 取 ticket、date、line、memory。Target 是界面打开它时用的编号：工单编号或记忆编号。
type BriefRef struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Label  string `json:"label"`
	Target string `json:"target,omitempty"`
}

// TodayBrief 是模型按材料写成的简报。Woven 为真时 Cards 是可保存的正文。
// Quiet 为真表示材料里确实没有值得写的事，界面写一封安静的信，不推进主会话。两者都为假表示这一次没写成，可以稍后再试。Pending 表示上一次还在写。
type TodayBrief struct {
	Cards   []BriefCard `json:"cards"`
	Woven   bool        `json:"woven"`
	Quiet   bool        `json:"quiet,omitempty"`
	Pending bool        `json:"pending,omitempty"`
}

// briefMaxCards 是一封简报最多留几张卡。扫一眼就能读完。
const briefMaxCards = 6

const briefPrompt = `你在写一封简报。材料是这台电脑上已经核对过的事实。只用材料里出现过的内容。材料里没有的会议、文件、数字、电量、健康、窗口标题、岗位都不要编。材料里没有值得写的事，就输出 {"cards":[]}。用材料里的语言写。

怎么排由你决定。不要套固定栏目，也不要为了凑版式编内容。每张卡选一种版式：

rows：几条并列的事。items 有 title、detail，tone 可以是 moving、quiet、stalled。
checks：可逐项核对。items 有 title、detail。
evidence：一条条证据。items 有 title、detail。
compare：左右对照。choices 恰好两项，每项有 id、title、detail。leftLabel、rightLabel 是两侧小标题。
letter：一边是已经有的，一边是还没有的。items 是左边，choices 是右边。leftLabel、rightLabel 是两侧标题。
stats：数得出来的量。stats 是若干 {label, value}，label 和 value 都来自材料。
chips：一组短名字。items 只写 title。
plain：一两句就够。可以没有 items。

每张卡都要有 id、kicker、title、lede。kicker 是一行小分类。title 是这张卡的判断，不要写成栏目名。lede 用一句话说明为什么现在值得看。

有下一步、而且人一点就能接着做时，写 primary（按钮上的短字）和 task（点下去放进输入框的完整请求，不要直接开工）。有第二个走法时再写 secondary 和 alt。没有下一步就不要写按钮。

mark 是卡底部的一行小记，可省略。note 是一句不带按钮的说明，可省略。

材料里方括号中的是编号。每张卡写 refs，列出这张卡依据的编号。编号只写进 refs，不写进标题和正文。没有编号可依据的判断，用 plain 一两句写清，不要编。

三到五张卡，要人先处理的在前。没有的部分整段不写，不要为了凑数编卡。材料里实在没有值得写的事，就输出 {"cards":[]}。

只输出一个 JSON 对象：{"cards":[{"id":"","kicker":"","title":"","lede":"","layout":"","items":[{"title":"","detail":"","tone":""}],"choices":[{"id":"","title":"","detail":""}],"stats":[{"label":"","value":""}],"leftLabel":"","rightLabel":"","mark":"","note":"","primary":"","secondary":"","task":"","alt":"","refs":[""]}]}。用不到的字段留空或省掉。`

const briefMorningAsk = "晨间简报。先写要人先定的：过期和今天到期的、在等回答的、卡住的。再写今天最值得推进的一到三件，每件写清下一步。能等的放最后，或者不写。"

const briefEveningAsk = "晚间简报。先写今天做成了什么：今天结束的工单和成形的文件。再写还开着、要挪到明天的，说清卡在哪。最后用一张卡只问一个问题：明天先碰哪一件。早上已经说过的不要再说一遍。"

// ComposeToday 按当前材料写这一档简报。slot 取 morning 或 evening。同一档、同一天、材料没变时返回上一次的结果，安静的结果也一样。
func (a *App) ComposeToday(slot string) TodayBrief {
	if a == nil {
		return TodayBrief{}
	}
	slot = briefSlot(slot)
	now := time.Now()
	material := a.briefMaterial(slot, now)
	packet := material.b.String()
	if !briefPacketWorth(packet) {
		return TodayBrief{Quiet: true}
	}
	sum := sha256.Sum256([]byte(slot + "\n" + now.Format("2006-01-02") + "\n" + packet))
	sig := hex.EncodeToString(sum[:])
	a.briefMu.Lock()
	if a.briefSig == sig && (a.briefCached.Woven || a.briefCached.Quiet) {
		cached := a.briefCached
		a.briefMu.Unlock()
		return cached
	}
	if a.briefBusy {
		a.briefMu.Unlock()
		return TodayBrief{Pending: true}
	}
	a.briefBusy = true
	a.briefMu.Unlock()
	defer func() {
		a.briefMu.Lock()
		a.briefBusy = false
		a.briefMu.Unlock()
	}()
	tokens := maxOutputTokensFor(a.Config.Model)
	if tokens <= 0 {
		tokens = 8192
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	out, err := liveModel{app: a}.Complete(ctx, dsh.Request{
		Messages: []dsh.Message{
			{Role: "system", Text: briefPrompt},
			{Role: "user", Text: briefAsk(slot, packet)},
		},
		MaxTokens: tokens,
	})
	if err != nil {
		return TodayBrief{}
	}
	cards, ok := parseTodayBrief(out.Text)
	if !ok {
		return TodayBrief{}
	}
	got := TodayBrief{Quiet: true}
	if cards = groundRefs(cards, material.refs); len(cards) > 0 {
		got = TodayBrief{Cards: cards, Woven: true}
	}
	a.briefMu.Lock()
	a.briefSig = sig
	a.briefCached = got
	a.briefMu.Unlock()
	return got
}

func briefSlot(slot string) string {
	if strings.TrimSpace(slot) == "evening" {
		return "evening"
	}
	return "morning"
}

func briefAsk(slot, packet string) string {
	ask := briefMorningAsk
	if slot == "evening" {
		ask = briefEveningAsk
	}
	return ask + "\n本地时间 " + time.Now().Format("2006-01-02 15:04") + "\n\n" + packet
}

// briefMaterial 是写一封简报用的材料，以及材料里出现过的编号。模型只能引用这些编号。
type briefMaterial struct {
	b    strings.Builder
	refs map[string]BriefRef
	seq  map[string]int
}

// mark 给一条材料一个编号，写成方括号。工单直接用工单编号，其余按种类顺序编。
func (m *briefMaterial) mark(prefix, kind, label, target string) string {
	if m.refs == nil {
		m.refs = map[string]BriefRef{}
		m.seq = map[string]int{}
	}
	id := target
	if kind != "ticket" {
		m.seq[prefix]++
		id = prefix + strconv.Itoa(m.seq[prefix])
	}
	ref := BriefRef{ID: id, Kind: kind, Label: clipBrief(oneLineBrief(label), 40)}
	if kind == "ticket" || kind == "memory" {
		ref.Target = target
	}
	m.refs[id] = ref
	return "[" + id + "]"
}

// briefMaterial 把画像、经历、最近对话、日程、工单、桌面清点和本机感知收成一段材料。
// 主会话里系统推进的简报、短讯和工人回报不算「最近说过」，免得简报复述自己。晚间还带上早上那封说过什么。
func (a *App) briefMaterial(slot string, now time.Time) *briefMaterial {
	m := &briefMaterial{}
	if a == nil {
		return m
	}
	b := &m.b
	if a.Memory != nil {
		p := a.Memory.Portrait()
		if p.Git != "" {
			b.WriteString("称呼 " + clipBrief(p.Git, 40) + "\n")
		}
		if p.Host != "" || p.OS != "" {
			b.WriteString("这台机器 " + clipBrief(strings.TrimSpace(p.Host+" "+p.OS), 80) + "\n")
		}
		if p.Hours != "" {
			b.WriteString("工作时间 " + clipBrief(p.Hours, 40) + "\n")
		}
		for _, it := range a.Memory.All() {
			if it.Staging || it.Source == "machine" || strings.TrimSpace(it.Text) == "" || memory.Expired(it, now) {
				continue
			}
			label := ""
			switch {
			case it.Kind == memory.KindSemantic:
				label = "偏好"
			case it.Kind == memory.KindProfile && memoryOperatorSource(it.Source):
				label = "本人说过"
			case it.Kind == memory.KindProfile:
				label = "记下的档案"
			case it.Kind == memory.KindProject:
				label = "记下的项目"
			default:
				continue
			}
			b.WriteString(label + " " + m.mark("m", "memory", it.Text, it.ID) + " " + clipBrief(it.Text, 240) + "\n")
		}
		for _, it := range a.Memory.Search("", memory.KindEpisodic, 16) {
			b.WriteString("经历 " + m.mark("m", "memory", it.Text, it.ID) + " " + clipBrief(it.Text, 400) + "\n")
		}
	}
	if a.Home != nil && a.Traces != nil {
		if meta, ok := a.savedMain(); ok && meta.ID != "" {
			if page, err := a.Traces.PageTurns(meta.ID, 0, 8); err == nil {
				var lines []string
				for _, ev := range page.Events {
					if ev.Type != trace.TypeUser && ev.Type != trace.TypeAssistant {
						continue
					}
					if pushedBySystem(ev) {
						continue
					}
					text := clipBrief(payloadText(ev), 800)
					if text == "" {
						continue
					}
					who := "助手"
					if ev.Type == trace.TypeUser {
						who = "本人"
					}
					lines = append(lines, who+" "+text)
				}
				if len(lines) > 24 {
					lines = lines[len(lines)-24:]
				}
				if len(lines) > 0 {
					b.WriteString("最近说过\n")
					for _, line := range lines {
						b.WriteString(line)
						b.WriteByte('\n')
					}
				}
			}
		}
	}
	if rt := a.orchestrator(); rt != nil {
		writeBriefDesk(m, rt.View(), now)
	}
	writeBriefCensus(m, poi.ScanCensus(poi.DefaultRoots(), now))
	writeBriefSenses(b, a)
	if slot == "evening" {
		writeMorningSaid(b, a.BriefList(), now.Format("2006-01-02"))
	}
	return m
}

// pushedBySystem 在这条是开口短讯、简报、兴趣变化、工人回报或通知时为真。它们不是本人和助手的对话。
func pushedBySystem(ev trace.Event) bool {
	switch ev.Source {
	case "desk", "worker", "notice":
		return true
	}
	kind, _ := ev.Payload["kind"].(string)
	switch kind {
	case "brief", "speak", "poi", "handoff", "status":
		return true
	}
	return false
}

// writeMorningSaid 把当天早上那封简报的标题列出来，晚间那封就不再说一遍。
func writeMorningSaid(b *strings.Builder, records []BriefRecord, day string) {
	for _, rec := range records {
		if rec.Day != day || rec.Slot != "morning" {
			continue
		}
		var body struct {
			Cards []struct {
				Title string `json:"title"`
			} `json:"cards"`
		}
		if json.Unmarshal([]byte(rec.Brief), &body) != nil || len(body.Cards) == 0 {
			return
		}
		b.WriteString("早上的简报说过\n")
		for _, card := range body.Cards {
			if title := strings.TrimSpace(card.Title); title != "" {
				b.WriteString("- " + clipBrief(title, 120) + "\n")
			}
		}
		return
	}
}

func writeBriefDesk(m *briefMaterial, view superagent.View, now time.Time) {
	b := &m.b
	var dates []string
	for _, row := range view.Queued {
		if row.Source != "calendar" || strings.TrimSpace(row.Text) == "" {
			continue
		}
		line := m.mark("d", "date", row.Text, row.ID) + " " + clipBrief(row.Text, 200)
		if row.Due != "" {
			line += " " + row.Due
		}
		dates = append(dates, line)
	}
	if len(dates) > 0 {
		b.WriteString("日历\n")
		for _, line := range dates {
			b.WriteString("- " + line + "\n")
		}
	}
	since := now.Add(-24 * time.Hour)
	var ended []string
	for _, ticket := range view.Tickets {
		if ticket.Stage != "ended" {
			continue
		}
		at, err := time.Parse(time.RFC3339, ticket.UpdatedAt)
		if err != nil || at.Before(since) {
			continue
		}
		line := m.mark("", "ticket", ticket.Title, ticket.ID) + " " + clipBrief(ticket.Title, 160) + " " + ticketState(ticket) + " " + at.In(now.Location()).Format("01-02 15:04")
		if summary := lastTicketSummary(ticket); summary != "" {
			line += " 报告 " + clipBrief(summary, 300)
		}
		ended = append(ended, line)
	}
	if len(ended) > 0 {
		b.WriteString("最近一天结束的工单\n")
		for _, line := range ended {
			b.WriteString("- " + line + "\n")
		}
	}
	for _, ticket := range view.Tickets {
		if ticket.Stage == "ended" {
			continue
		}
		line := "工单 " + m.mark("", "ticket", ticket.Title, ticket.ID) + " " + clipBrief(ticket.Title, 160) + " 阶段 " + ticket.Stage
		if ticket.Action != "" {
			line += " 等 " + clipBrief(ticket.Action, 40)
		}
		if ticket.Attention != "" {
			line += " 注意 " + clipBrief(ticket.Attention, 120)
		}
		b.WriteString(line + "\n")
		if len(ticket.Assignments) == 0 {
			continue
		}
		last := ticket.Assignments[len(ticket.Assignments)-1]
		summary := last.Summary
		if summary == "" {
			summary = last.Report
		}
		if summary != "" {
			b.WriteString("  报告 " + clipBrief(summary, 500) + "\n")
		}
		for _, row := range last.Lines {
			if row.Question != "" {
				b.WriteString("  提问 " + clipBrief(row.Question, 240) + "\n")
			}
		}
	}
	if view.Load.Running > 0 || view.Load.Queued > 0 {
		b.WriteString(fmt.Sprintf("执行 正在做 %d 排队 %d\n", view.Load.Running, view.Load.Queued))
	}
	for _, problem := range view.Problems {
		text := scrubSense(strings.TrimSpace(problem.Title + " " + problem.Detail))
		if text == "" {
			continue
		}
		b.WriteString("要处理 " + clipBrief(problem.Kind+" "+text, 240) + "\n")
	}
	n := 0
	for i := len(view.Decisions) - 1; i >= 0; i-- {
		row := view.Decisions[i]
		if row.Dismissed || strings.TrimSpace(row.Reason) == "" {
			continue
		}
		b.WriteString("决定 " + row.Kind + " " + clipBrief(row.Reason, 200) + "\n")
		n++
		if n == 12 {
			break
		}
	}
}

func writeBriefCensus(m *briefMaterial, c poi.Census) {
	b := &m.b
	var lines []string
	for _, line := range c.Lines {
		name := scrubSense(line.Name)
		if name == "" {
			continue
		}
		row := "工程线 " + m.mark("l", "line", name, "") + " " + clipBrief(name, 80)
		if line.State != "" {
			row += " " + clipBrief(line.State, 40)
		}
		if evidence := scrubSense(line.Evidence); evidence != "" {
			row += " " + clipBrief(evidence, 80)
		}
		if line.When != "" {
			row += " " + clipBrief(line.When, 40)
		}
		lines = append(lines, row)
	}
	for _, item := range c.Finished {
		if name := scrubSense(item.Name); name != "" {
			lines = append(lines, "已成形 "+clipBrief(name, 80))
		}
	}
	for _, item := range c.Empty {
		name := scrubSense(item.Name)
		if name == "" {
			continue
		}
		row := "空文件 " + clipBrief(name, 80)
		if item.Line != "" {
			row += " 属于 " + clipBrief(item.Line, 40)
		}
		lines = append(lines, row)
	}
	for _, item := range c.Place {
		name := scrubSense(item.Name)
		if name == "" {
			continue
		}
		row := "对得上 " + clipBrief(name, 80)
		if item.Line != "" {
			row += " → " + clipBrief(item.Line, 40)
		}
		lines = append(lines, row)
	}
	for _, item := range c.Archive {
		if name := scrubSense(item.Name); name != "" {
			lines = append(lines, "安装包 "+clipBrief(name, 80))
		}
	}
	if c.Projects > 0 || c.Downloads > 0 {
		lines = append(lines, fmt.Sprintf("数量 工程 %d 近 45 天下载 %d", c.Projects, c.Downloads))
	}
	if len(lines) == 0 {
		return
	}
	b.WriteString("桌面清点\n")
	for _, line := range lines {
		b.WriteString("- " + line + "\n")
	}
}

func writeBriefSenses(b *strings.Builder, a *App) {
	facts, _ := initiative.LocalFacts(poi.DefaultRoots())
	var lines []string
	for _, fact := range facts {
		switch fact.Code {
		case "idle":
			if fact.Detail != "" {
				lines = append(lines, "离开 "+fact.Detail)
			}
		case "power":
			line := "电量 " + fact.Name
			if fact.Detail == "charging" {
				line += " 充电中"
			}
			lines = append(lines, line)
		case "focus":
			if name := scrubSense(fact.Name); name != "" {
				lines = append(lines, "前台程序 "+clipBrief(name, 60))
			}
		case "net":
			if fact.State == "offline" {
				lines = append(lines, "网络不在线")
			}
		case "git":
			name := scrubSense(fact.Name)
			if name == "" {
				continue
			}
			line := "仓库 " + clipBrief(name, 60)
			if fact.Detail != "" {
				line += " 分支 " + clipBrief(fact.Detail, 40)
			}
			if fact.State == "dirty" {
				line += " 有未提交"
			}
			lines = append(lines, line)
		}
	}
	if a != nil {
		for _, fact := range a.connectorFacts() {
			if fact.State != "linked" {
				continue
			}
			switch fact.Code {
			case "mail":
				lines = append(lines, "邮件已连接")
			case "calendar":
				lines = append(lines, "日历账号已连接")
			}
		}
	}
	if len(lines) == 0 {
		return
	}
	b.WriteString("感知\n")
	for _, line := range lines {
		b.WriteString("- " + line + "\n")
	}
}

// briefPacketWorth 在材料里有能写成简报的事时为真。只有机器名字时不算。
func briefPacketWorth(packet string) bool {
	for _, mark := range []string{
		"本人说过", "记下的档案", "偏好", "工程", "经历", "最近说过", "日历", "工单",
		"要处理", "决定 ", "执行 ", "工程线 ", "已成形 ", "空文件 ", "对得上 ", "安装包 ", "数量 ",
		"离开 ", "前台程序 ", "仓库 ", "网络不在线", "邮件已连接", "日历账号已连接",
	} {
		if strings.Contains(packet, mark) {
			return true
		}
	}
	return false
}

type rawBriefCard struct {
	ID         string         `json:"id"`
	Kicker     string         `json:"kicker"`
	Title      string         `json:"title"`
	Lede       string         `json:"lede"`
	Layout     string         `json:"layout"`
	Items      []BriefBit     `json:"items"`
	Choices    []BriefBit     `json:"choices"`
	Stats      []rawBriefStat `json:"stats"`
	Mark       string         `json:"mark"`
	LeftLabel  string         `json:"leftLabel"`
	RightLabel string         `json:"rightLabel"`
	Note       string         `json:"note"`
	Primary    string         `json:"primary"`
	Secondary  string         `json:"secondary"`
	Task       string         `json:"task"`
	Alt        string         `json:"alt"`
	Refs       []string       `json:"refs"`
}

type rawBriefStat struct {
	Label string `json:"label"`
	Value any    `json:"value"`
}

// parseTodayCards 从模型正文里取出卡片。认不出或没有标题时为空。
func parseTodayCards(text string) []BriefCard {
	cards, _ := parseTodayBrief(text)
	return cards
}

// parseTodayBrief 从模型正文里取出卡片，最多 briefMaxCards 张。ok 为真表示正文里确实是一个带 cards 的对象，空数组也算。
// 先认 json 代码块，再认第一个配平的花括号对象，最后才退回首尾花括号。
func parseTodayBrief(text string) ([]BriefCard, bool) {
	var body struct {
		Cards *[]rawBriefCard `json:"cards"`
	}
	parsed := false
	for _, candidate := range jsonCandidates(text) {
		body.Cards = nil
		if json.Unmarshal([]byte(candidate), &body) == nil && body.Cards != nil {
			parsed = true
			break
		}
	}
	if !parsed {
		return nil, false
	}
	var out []BriefCard
	seen := map[string]bool{}
	for _, raw := range *body.Cards {
		card := briefFromRaw(raw)
		if card.ID == "" || card.Title == "" || seen[card.ID] {
			continue
		}
		seen[card.ID] = true
		out = append(out, card)
		if len(out) == briefMaxCards {
			break
		}
	}
	return out, true
}

// jsonCandidates 列出正文里可能是那个 JSON 对象的几段，先试最可靠的。
func jsonCandidates(text string) []string {
	text = strings.TrimSpace(text)
	var out []string
	if i := strings.Index(text, "```"); i >= 0 {
		rest := text[i+3:]
		rest = strings.TrimPrefix(strings.TrimPrefix(rest, "json"), "JSON")
		if j := strings.Index(rest, "```"); j >= 0 {
			out = append(out, strings.TrimSpace(rest[:j]))
		}
	}
	if obj := balancedObject(text); obj != "" {
		out = append(out, obj)
	}
	if i := strings.Index(text, "{"); i >= 0 {
		if j := strings.LastIndex(text, "}"); j > i {
			out = append(out, text[i:j+1])
		}
	}
	return out
}

// balancedObject 取出第一个带 "cards" 的配平花括号对象。字符串里的花括号不算。
func balancedObject(text string) string {
	for start := strings.Index(text, "{"); start >= 0; {
		depth, inString, escaped := 0, false, false
		for i := start; i < len(text); i++ {
			c := text[i]
			switch {
			case escaped:
				escaped = false
			case inString && c == '\\':
				escaped = true
			case c == '"':
				inString = !inString
			case inString:
			case c == '{':
				depth++
			case c == '}':
				depth--
				if depth == 0 {
					if obj := text[start : i+1]; strings.Contains(obj, `"cards"`) {
						return obj
					}
					i = len(text)
				}
			}
		}
		next := strings.Index(text[start+1:], "{")
		if next < 0 {
			break
		}
		start += next + 1
	}
	return ""
}

// groundRefs 把卡上的编号换成材料里那一条，认不出的编号丢掉。同一张卡里重复的只留一次。
func groundRefs(cards []BriefCard, known map[string]BriefRef) []BriefCard {
	for i := range cards {
		var kept []BriefRef
		seen := map[string]bool{}
		for _, ref := range cards[i].Refs {
			id := strings.Trim(strings.TrimSpace(ref.ID), "[]")
			got, ok := known[id]
			if !ok || seen[id] {
				continue
			}
			seen[id] = true
			kept = append(kept, got)
		}
		cards[i].Refs = kept
	}
	return cards
}

func memoryOperatorSource(source string) bool {
	switch strings.TrimSpace(source) {
	case "", "user", "User confirmed in chat":
		return true
	default:
		return false
	}
}

func oneLineBrief(text string) string {
	text = strings.TrimSpace(text)
	if i := strings.IndexAny(text, "\r\n"); i >= 0 {
		text = text[:i]
	}
	return strings.TrimSpace(text)
}

func briefFromRaw(raw rawBriefCard) BriefCard {
	card := BriefCard{
		ID:         strings.TrimSpace(raw.ID),
		Kicker:     strings.TrimSpace(raw.Kicker),
		Title:      strings.TrimSpace(raw.Title),
		Lede:       strings.TrimSpace(raw.Lede),
		Layout:     briefLayout(raw.Layout),
		Items:      compactBits(raw.Items),
		Choices:    compactBits(raw.Choices),
		Stats:      compactStats(raw.Stats),
		Mark:       strings.TrimSpace(raw.Mark),
		LeftLabel:  strings.TrimSpace(raw.LeftLabel),
		RightLabel: strings.TrimSpace(raw.RightLabel),
		Note:       strings.TrimSpace(raw.Note),
		Primary:    strings.TrimSpace(raw.Primary),
		Secondary:  strings.TrimSpace(raw.Secondary),
		Task:       strings.TrimSpace(raw.Task),
		Alt:        strings.TrimSpace(raw.Alt),
	}
	for _, id := range raw.Refs {
		if id = strings.TrimSpace(id); id != "" {
			card.Refs = append(card.Refs, BriefRef{ID: id})
		}
	}
	return card
}

func briefLayout(layout string) string {
	switch strings.ToLower(strings.TrimSpace(layout)) {
	case "evidence", "rows", "checks", "compare", "chips", "plain", "letter", "stats":
		return strings.ToLower(strings.TrimSpace(layout))
	default:
		return ""
	}
}

func compactBits(in []BriefBit) []BriefBit {
	out := make([]BriefBit, 0, len(in))
	for _, bit := range in {
		bit.ID = strings.TrimSpace(bit.ID)
		bit.Title = strings.TrimSpace(bit.Title)
		bit.Detail = strings.TrimSpace(bit.Detail)
		switch strings.ToLower(strings.TrimSpace(bit.Tone)) {
		case "moving", "quiet", "stalled":
			bit.Tone = strings.ToLower(strings.TrimSpace(bit.Tone))
		default:
			bit.Tone = ""
		}
		if bit.Title == "" && bit.Detail == "" {
			continue
		}
		if bit.Title == "" {
			bit.Title = bit.Detail
			bit.Detail = ""
		}
		out = append(out, bit)
	}
	if out == nil {
		return []BriefBit{}
	}
	return out
}

func compactStats(in []rawBriefStat) []BriefStat {
	var out []BriefStat
	for _, stat := range in {
		label := strings.TrimSpace(stat.Label)
		value := scalarText(stat.Value)
		if label == "" || value == "" {
			continue
		}
		out = append(out, BriefStat{Label: label, Value: value})
	}
	return out
}

func scalarText(v any) string {
	switch n := v.(type) {
	case string:
		return strings.TrimSpace(n)
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return ""
		}
		if n == math.Trunc(n) {
			return strconv.FormatInt(int64(n), 10)
		}
		return strconv.FormatFloat(n, 'f', -1, 64)
	case bool:
		if n {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func payloadText(ev trace.Event) string {
	if ev.Payload == nil {
		return ""
	}
	text, _ := ev.Payload["text"].(string)
	return strings.TrimSpace(text)
}

func clipBrief(text string, n int) string {
	text = strings.TrimSpace(text)
	r := []rune(text)
	if n < 1 || len(r) <= n {
		return text
	}
	return string(r[:n])
}
