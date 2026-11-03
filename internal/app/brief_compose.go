// brief_compose.go 在人拉开「今天」时，用画像、记忆、最近对话和日程写一封简报。
// 不拥有推送。材料没变时不再叫模型。没有对话、经历、日程、工单或工程时，不编一封。

package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// BriefBit 是简报卡上的一行。
type BriefBit struct {
	ID     string `json:"id,omitempty"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

// BriefCard 是「今天」里的一张卡。Task 是点按钮后放进输入框的请求，不直接发出。
type BriefCard struct {
	ID        string     `json:"id"`
	Kicker    string     `json:"kicker,omitempty"`
	Title     string     `json:"title"`
	Lede      string     `json:"lede,omitempty"`
	Layout    string     `json:"layout,omitempty"`
	Items     []BriefBit `json:"items"`
	Choices   []BriefBit `json:"choices"`
	Primary   string     `json:"primary,omitempty"`
	Secondary string     `json:"secondary,omitempty"`
	Task      string     `json:"task,omitempty"`
	Alt       string     `json:"alt,omitempty"`
}

// TodayBrief 是模型按材料写成的简报。Woven 为假时界面继续用本地模板。Pending 表示上一次还在写。
type TodayBrief struct {
	Cards   []BriefCard `json:"cards"`
	Woven   bool        `json:"woven"`
	Pending bool        `json:"pending,omitempty"`
}

const briefPrompt = "你在写「今天」这一栏的简报。只使用材料里已经出现的事实。不要编新闻、健康、电量、窗口标题，也不要编材料里没有的会议、岗位或文件。这个人最近说过的事、留下的经历、到期的日程、进行中的工单和工程线，各写成能扫一眼的卡。有下一步就给一个按钮，task 是点下去放进输入框的完整请求，不要直接开工。没有可用的事就不要编。输出 JSON，形如 {\"cards\":[{\"id\":\"\",\"kicker\":\"\",\"title\":\"\",\"lede\":\"\",\"layout\":\"evidence\",\"items\":[{\"title\":\"\",\"detail\":\"\"}],\"choices\":[],\"primary\":\"\",\"secondary\":\"\",\"task\":\"\",\"alt\":\"\"}]}。layout 只能是 evidence、rows、checks、compare、chips 或 plain。最多四张卡。"

// ComposeToday 按当前材料写简报。材料没变时返回上一次的结果。
func (a *App) ComposeToday() TodayBrief {
	if a == nil {
		return TodayBrief{}
	}
	packet := a.briefPacket()
	if !briefPacketWorth(packet) {
		return TodayBrief{}
	}
	sum := sha256.Sum256([]byte(packet))
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
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	out, err := liveModel{app: a}.Complete(ctx, dsh.Request{
		Messages: []dsh.Message{
			{Role: "system", Text: briefPrompt},
			{Role: "user", Text: packet},
		},
		MaxTokens: 1200,
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

// briefPacket 把画像、经历、最近对话、日程、工单和工程收成一段材料。
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
			b.WriteString("本人说过 " + clipBrief(line, 160) + "\n")
		}
		for _, line := range p.Noted {
			b.WriteString("记下的档案 " + clipBrief(line, 160) + "\n")
		}
		for _, line := range p.Preferences {
			b.WriteString("偏好 " + clipBrief(line, 160) + "\n")
		}
		if len(p.Lines) > 0 {
			b.WriteString("工程\n")
			for _, line := range p.Lines {
				b.WriteString("- " + clipBrief(line.Name, 40))
				if line.State != "" {
					b.WriteString(" " + clipBrief(line.State, 20))
				}
				b.WriteByte('\n')
			}
		}
		for _, it := range a.Memory.Search("", memory.KindEpisodic, 8) {
			b.WriteString("经历 " + clipBrief(it.Text, 240) + "\n")
		}
	}
	if a.Home != nil && a.Traces != nil {
		meta, ok := a.savedMain()
		if !ok {
			meta.ID = ""
		}
		if meta.ID != "" {
			page, err := a.Traces.PageTurns(meta.ID, 0, 4)
			if err == nil {
				var lines []string
				for _, ev := range page.Events {
					if ev.Type != trace.TypeUser && ev.Type != trace.TypeAssistant {
						continue
					}
					text := clipBrief(payloadText(ev), 400)
					if text == "" {
						continue
					}
					who := "助手"
					if ev.Type == trace.TypeUser {
						who = "本人"
					}
					lines = append(lines, who+" "+text)
				}
				if len(lines) > 12 {
					lines = lines[len(lines)-12:]
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
		view := rt.View()
		var dates []string
		for _, row := range view.Queued {
			if row.Source != "calendar" || strings.TrimSpace(row.Text) == "" {
				continue
			}
			line := clipBrief(row.Text, 120)
			if row.Due != "" {
				line += " " + row.Due
			}
			dates = append(dates, line)
			if len(dates) == 6 {
				break
			}
		}
		if len(dates) > 0 {
			b.WriteString("日历\n")
			for _, line := range dates {
				b.WriteString("- " + line + "\n")
			}
		}
		n := 0
		for _, ticket := range view.Tickets {
			if ticket.Stage == "ended" {
				continue
			}
			b.WriteString("工单 " + clipBrief(ticket.Title, 80) + " " + ticket.Stage + "\n")
			n++
			if n == 6 {
				break
			}
		}
	}
	return b.String()
}

// briefPacketWorth 在材料里有能写成简报的事时为真。只有机器名字时不算。
func briefPacketWorth(packet string) bool {
	for _, mark := range []string{"本人说过", "记下的档案", "偏好", "工程", "经历", "最近说过", "日历", "工单"} {
		if strings.Contains(packet, mark) {
			return true
		}
	}
	return false
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
		Cards []BriefCard `json:"cards"`
	}
	if json.Unmarshal([]byte(text), &body) != nil {
		return nil
	}
	var out []BriefCard
	seen := map[string]bool{}
	for _, card := range body.Cards {
		card.ID = strings.TrimSpace(card.ID)
		card.Title = strings.TrimSpace(card.Title)
		if card.ID == "" || card.Title == "" || seen[card.ID] {
			continue
		}
		switch card.Layout {
		case "evidence", "rows", "checks", "compare", "chips", "plain":
		default:
			card.Layout = "evidence"
		}
		if card.Items == nil {
			card.Items = []BriefBit{}
		}
		if card.Choices == nil {
			card.Choices = []BriefBit{}
		}
		seen[card.ID] = true
		out = append(out, card)
		if len(out) == 4 {
			break
		}
	}
	return out
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
