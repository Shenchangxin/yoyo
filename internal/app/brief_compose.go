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
// Layout 为空时，界面按这张卡自己的内容选版式。
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
}

// TodayBrief 是模型按材料写成的简报。Woven 为假表示这一轮没有可保存的正文。Pending 表示上一次还在写。
type TodayBrief struct {
	Cards   []BriefCard `json:"cards"`
	Woven   bool        `json:"woven"`
	Pending bool        `json:"pending,omitempty"`
}

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

只输出一个 JSON 对象：{"cards":[{"id":"","kicker":"","title":"","lede":"","layout":"","items":[{"title":"","detail":"","tone":""}],"choices":[{"id":"","title":"","detail":""}],"stats":[{"label":"","value":""}],"leftLabel":"","rightLabel":"","mark":"","note":"","primary":"","secondary":"","task":"","alt":""}]}。用不到的字段留空或省掉。材料里有几件就写几件。`

// ComposeToday 按当前材料写这一档简报。slot 取 morning 或 evening。同一档、同一天、材料没变时返回上一次的结果。
func (a *App) ComposeToday(slot string) TodayBrief {
	if a == nil {
		return TodayBrief{}
	}
	slot = briefSlot(slot)
	packet := a.briefPacket()
	if !briefPacketWorth(packet) {
		return TodayBrief{}
	}
	sum := sha256.Sum256([]byte(slot + "\n" + time.Now().Format("2006-01-02") + "\n" + packet))
	sig := hex.EncodeToString(sum[:])
	a.briefMu.Lock()
	if a.briefSig == sig && a.briefCached.Woven {
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
	cards := parseTodayCards(out.Text)
	if len(cards) == 0 {
		return TodayBrief{}
	}
	got := TodayBrief{Cards: cards, Woven: true}
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
	name := "晨间简报"
	if slot == "evening" {
		name = "晚间简报"
	}
	return name + "\n本地时间 " + time.Now().Format("2006-01-02 15:04") + "\n\n" + packet
}

// briefPacket 把画像、经历、最近对话、日程、工单、桌面清点和本机感知收成一段材料。
func (a *App) briefPacket() string {
	if a == nil {
		return ""
	}
	var b strings.Builder
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
		for _, line := range p.Said {
			b.WriteString("本人说过 " + clipBrief(line, 240) + "\n")
		}
		for _, line := range p.Noted {
			b.WriteString("记下的档案 " + clipBrief(line, 240) + "\n")
		}
		for _, line := range p.Preferences {
			b.WriteString("偏好 " + clipBrief(line, 240) + "\n")
		}
		if len(p.Lines) > 0 {
			b.WriteString("工程\n")
			for _, line := range p.Lines {
				b.WriteString("- " + clipBrief(line.Name, 60))
				if line.State != "" {
					b.WriteString(" " + clipBrief(line.State, 40))
				}
				b.WriteByte('\n')
			}
		}
		for _, it := range a.Memory.Search("", memory.KindEpisodic, 16) {
			b.WriteString("经历 " + clipBrief(it.Text, 400) + "\n")
		}
	}
	if a.Home != nil && a.Traces != nil {
		meta, ok := a.savedMain()
		if !ok {
			meta.ID = ""
		}
		if meta.ID != "" {
			page, err := a.Traces.PageTurns(meta.ID, 0, 8)
			if err == nil {
				var lines []string
				for _, ev := range page.Events {
					if ev.Type != trace.TypeUser && ev.Type != trace.TypeAssistant {
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
		writeBriefDesk(&b, rt.View())
	}
	writeBriefCensus(&b, poi.ScanCensus(poi.DefaultRoots(), time.Now()))
	writeBriefSenses(&b, a)
	return b.String()
}

func writeBriefDesk(b *strings.Builder, view superagent.View) {
	var dates []string
	for _, row := range view.Queued {
		if row.Source != "calendar" || strings.TrimSpace(row.Text) == "" {
			continue
		}
		line := clipBrief(row.Text, 200)
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
	for _, ticket := range view.Tickets {
		if ticket.Stage == "ended" {
			continue
		}
		line := "工单 " + clipBrief(ticket.Title, 160) + " 阶段 " + ticket.Stage
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

func writeBriefCensus(b *strings.Builder, c poi.Census) {
	var lines []string
	for _, line := range c.Lines {
		name := scrubSense(line.Name)
		if name == "" {
			continue
		}
		row := "工程线 " + clipBrief(name, 80)
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
}

type rawBriefStat struct {
	Label string `json:"label"`
	Value any    `json:"value"`
}

// parseTodayCards 从模型正文里取出卡片。认不出或没有标题时为空。
func parseTodayCards(text string) []BriefCard {
	text = strings.TrimSpace(text)
	if i := strings.Index(text, "{"); i >= 0 {
		if j := strings.LastIndex(text, "}"); j > i {
			text = text[i : j+1]
		}
	}
	var body struct {
		Cards []rawBriefCard `json:"cards"`
	}
	if json.Unmarshal([]byte(text), &body) != nil {
		return nil
	}
	var out []BriefCard
	seen := map[string]bool{}
	for _, raw := range body.Cards {
		card := briefFromRaw(raw)
		if card.ID == "" || card.Title == "" || seen[card.ID] {
			continue
		}
		seen[card.ID] = true
		out = append(out, card)
	}
	return out
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
