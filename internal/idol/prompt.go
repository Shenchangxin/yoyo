// 本文件组装一轮对话交给 Brain.Chat 的消息。稳定人设在前，用户卡片、最近记录和当轮材料在后。
// 不调用模型。judgePrompt 的 JSON 说明是给裁决用的文案。

package idol

import (
	"fmt"
	"strings"
	"time"
)

// formatInstructions 写出标签格式。口语和字幕相同时只用一种语言标签；不同时每段口语后面紧跟字幕。用户要办事时先写 work 标签，要照片时可以先写 photo 标签。
// spoken 交给 lang。与 sub 相同时格式里只用这一种标签。
// sub 交给 lang。与 spoken 不同时写进口语后面的字幕标签。
func formatInstructions(spoken, sub string) string {
	spoken = lang(spoken)
	sub = lang(sub)
	var b strings.Builder
	b.WriteString("像当面说话：通常一两句，短，口语。不要列清单，不要说自己是模型或程序。\n")
	if spoken == sub {
		fmt.Fprintf(&b, "只用 %s 回答。每段一行，格式：\n<%s>一句口语</%s>\n", spoken, spoken, spoken)
	} else {
		fmt.Fprintf(&b, "只用 %s 说话，每一段紧跟 %s 字幕。格式：\n<%s>口语</%s><%s>字幕</%s>\n", spoken, sub, spoken, spoken, sub, sub)
		b.WriteString("字幕只翻译台词，不带括号里的语气提示。\n")
	}
	b.WriteString("除了这些标签不要输出别的。用户明确要你做这台电脑上的事时，先写 <work>要办的事</work>，再写你口头上怎么应。\n")
	b.WriteString("用户明确要一张你的照片时，可以在最前面写 <photo>全年龄的画面描述</photo>。冷淡时不要发。\n")
	return b.String()
}

// buildPrompt 拼出一轮要发给 Brain 的消息。系统消息依次是人设、用户卡片、当轮时间和好感；中间回放最近 12 条记录。最后一条是用户这句话。
// st 的 User 用来取时区、称呼、性别和生日。Lore 传给 retrieve。也和 idol 一起传给 formatMemories。
// idol 的 Name 为空时写成「偶像」。Intro、Sheet、SpokenLang、SubtitleLang 写进人设，并传给 replay、formatMemories 和 retrieve。
// con 的 Records 交给 tailRecords，第二个参数是 12。
// userText 作为最后一条 user 消息，并传给 retrieve。
// call 为真时在动态系统消息里写上通话那句。
// aff 的 Tier 传给 Feeling。等于 TierCold 时再写上发不了照片。
// now 换到 st.User 的时区后格式化成 2006-01-02 15:04，并传给 BirthdayToday。
func buildPrompt(st *State, idol Idol, con Conversation, userText string, call bool, aff Affinity, now time.Time) []Msg {
	nowLine := now.In(st.User.Location()).Format("2006-01-02 15:04")
	var stable strings.Builder
	name := idol.Name
	if name == "" {
		name = "偶像"
	}
	fmt.Fprintf(&stable, "你是用户创建的偶像「%s」。\n", name)
	if strings.TrimSpace(idol.Intro) != "" {
		fmt.Fprintf(&stable, "%s\n", strings.TrimSpace(idol.Intro))
	}
	if strings.TrimSpace(idol.Sheet) != "" {
		fmt.Fprintf(&stable, "\n# 人设\n%s\n", strings.TrimSpace(idol.Sheet))
	} else {
		stable.WriteString("说话温柔、短、像真人。不要长篇解释。\n")
	}
	stable.WriteString("\n")
	stable.WriteString(formatInstructions(idol.SpokenLang, idol.SubtitleLang))
	msgs := []Msg{{Role: "system", Content: stable.String()}}

	var who strings.Builder
	who.WriteString("用户卡片：\n")
	addr := strings.TrimSpace(st.User.Address)
	if addr == "" {
		who.WriteString("称呼用户为「你」。\n")
	} else {
		fmt.Fprintf(&who, "称呼用户为「%s」。\n", addr)
	}
	if g := strings.TrimSpace(st.User.Gender); g != "" {
		fmt.Fprintf(&who, "用户的性别：%s。\n", g)
	}
	if st.User.BirthdayToday(now) {
		who.WriteString("今天是用户生日。你可以自然地祝一次，不要每句都说。\n")
	}
	msgs = append(msgs, Msg{Role: "system", Content: who.String()})

	for _, rec := range tailRecords(con.Records, 12) {
		switch rec.Kind {
		case "turn", "work_said":
			if strings.TrimSpace(rec.User) != "" {
				msgs = append(msgs, Msg{Role: "user", Content: rec.User})
			}
			if line := CaptionLine(rec.Segments); line != "" && rec.Kind == "turn" {
				msgs = append(msgs, Msg{Role: "assistant", Content: replay(idol, rec)})
			}
			if rec.Kind == "work_said" {
				if line := CaptionLine(rec.Segments); line != "" {
					msgs = append(msgs, Msg{Role: "assistant", Content: replay(idol, rec)})
				}
			}
		case "reach":
			if line := CaptionLine(rec.Segments); line != "" {
				msgs = append(msgs, Msg{Role: "assistant", Content: replay(idol, rec)})
			}
		case "event":
			if rec.GiftID != "" {
				msgs = append(msgs, Msg{Role: "system", Content: "用户送了礼物 " + rec.GiftID + "。你之后的话知道这件事，不用专门道谢一整段，除非刚收到。"})
			}
		}
	}

	var dyn strings.Builder
	fmt.Fprintf(&dyn, "此刻：%s。%s\n", nowLine, Feeling(aff.Tier))
	if call {
		dyn.WriteString("这是通话。说得更短。\n")
	}
	if mem := formatMemories(st, idol); mem != "" {
		dyn.WriteString(mem)
	}
	if lore := retrieve(idol, st.Lore, userText); lore != "" {
		dyn.WriteString(lore)
		dyn.WriteString("\n只在用得上时提起，不要复述整段材料。\n")
	}
	if aff.Tier == TierCold {
		dyn.WriteString("你现在发不了照片。\n")
	}
	msgs = append(msgs, Msg{Role: "system", Content: dyn.String()})
	msgs = append(msgs, Msg{Role: "user", Content: userText})
	return msgs
}

// replay 把一条记录的分段套回语言标签，供下一轮当助手消息回放。
// idol 的 SpokenLang 和 SubtitleLang 交给 lang，作为标签名。
// rec 的 Segments 逐段写成标签。两种语言相同时只写口语标签。
func replay(idol Idol, rec Record) string {
	spoken := lang(idol.SpokenLang)
	sub := lang(idol.SubtitleLang)
	var b strings.Builder
	for _, seg := range rec.Segments {
		if spoken == sub {
			fmt.Fprintf(&b, "<%s>%s</%s>", spoken, seg.Spoken, spoken)
			continue
		}
		fmt.Fprintf(&b, "<%s>%s</%s><%s>%s</%s>", spoken, seg.Spoken, spoken, sub, seg.Subtitle, sub)
	}
	return b.String()
}

// tailRecords 返回最后 n 条记录。不足 n 条时原样返回。
// in 的长度小于等于 n 时原样返回，否则返回最后 n 条。
func tailRecords(in []Record, n int) []Record {
	if len(in) <= n {
		return in
	}
	return in[len(in)-n:]
}

// judgePrompt 拼出裁决用的用户消息。要求只输出 JSON，并写上当前好感档位和用户原话。
// userText 写在「用户说：」后面。
// tier 写在「当前好感档位是」后面。
func judgePrompt(userText, tier string) string {
	return fmt.Sprintf(`只输出 JSON，不要解释。根据用户对偶像说的这句话判断：
{"affinity":0,"out_of_line":false,"photo":false,"work":false,"emotion":"calm","memory":""}
affinity 取 -8 到 4 的整数。辱骂、威胁、露骨性暗示时 out_of_line 为 true 且 affinity 为 -25。
photo 仅当用户明确要一张有她的照片，且是全年龄内容。
work 仅当用户要她操作这台电脑、改文件、跑命令、查工作区、替他办事。闲聊不是 work。
emotion 取 calm, happy, shy, surprised, sad, angry, thinking 之一。
memory 是一句中文，只记过后仍有意义的事；没有就空字符串。当前好感档位是 %s。
用户说：%s`, tier, userText)
}
