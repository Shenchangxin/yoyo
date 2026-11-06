// 本文件组装交给 Brain.Chat 的消息，形状照 eidolon 的 ADR-0020：
// 人设、事实和回复格式在最前面单独一条系统消息，同一位偶像每轮都一样，可以缓存；用户卡片随后；再是对话记录；
// 每轮会变的（时间、好感、照片、记忆、材料）放在记录之后、用户这句话之前，不带进下一轮，所以每轮的前缀和上一轮一样。
// 礼物、她先开口、替用户办完事、挂断电话这些不是回合的事，写成旁白放在其后那条用户消息的开头，角色始终一问一答交替。不调用模型。

package idol

import (
	"fmt"
	"strings"
	"time"
)

const (
	// historyTurns 是模型至少读多少轮最近的对话。
	historyTurns = 12
	// factsInline 是事实写进稳定前缀的上限（字符）。超过时按这句话检索相关的几节。
	factsInline = 4000
	// maxInput 是用户一句话最长多少字，超过就拒绝，不截断。
	maxInput = 2000
)

// langNames 是常见语言码的中文名，写进回复格式。
var langNames = map[string]string{"zh": "中文", "ja": "日语", "en": "英语", "ko": "韩语", "fr": "法语", "de": "德语", "es": "西班牙语", "ru": "俄语"}

// langName 返回语言码的中文名，不认识的原样返回。
func langName(code string) string {
	code = lang(code)
	if n, ok := langNames[code]; ok {
		return n
	}
	if i := strings.IndexByte(code, '-'); i > 0 {
		if n, ok := langNames[code[:i]]; ok {
			return n
		}
	}
	return code
}

// formatInstructions 写出回复格式、说话长度、情绪与语气，以及办事标签。
// spoken 和 sub 相同时只用一种标签；不同时每段口语后面紧跟字幕。work 为真时说明 work 标签。
func formatInstructions(spoken, sub string, work bool) string {
	spoken, sub = lang(spoken), lang(sub)
	sn, bn := langName(spoken), langName(sub)
	var b strings.Builder
	b.WriteString("# 回复格式\n")
	if spoken == sub {
		fmt.Fprintf(&b, "你只用%s回答。把回复分成段，每段一句口语，格式如下：\n<%s>一句口语</%s>\n", sn, spoken, spoken)
	} else {
		fmt.Fprintf(&b, "无论用户用什么语言说话，你都只用%s回答，并给每段附上%s字幕。把回复分成段，每段先写%s原文，紧接着写这段的%s字幕，格式如下：\n<%s>%s原文</%s><%s>%s字幕</%s>\n",
			sn, bn, sn, bn, spoken, sn, spoken, sub, bn, sub)
	}
	b.WriteString("除了这些标签和标签里的内容，不要输出任何其他文字。不要用 Markdown，不要列清单，不要说自己是 AI、模型或程序。\n\n")
	b.WriteString(`# 说话长度
像真人当面说话那样：接对方一句话，通常只回一两句。
- 平时只写一到两段，每段只有一句简短的口语。不写长句，不要把几件事塞进一段，也不要用「首先……其次……」这种条理分明的说法。
- 被要求讲一段事情时（包括回忆过去的事），可以多说几句，但最多四段左右；讲到自然的停顿处就停下，剩下的等对方追问再讲。
- 不必每次都用提问收尾，话说完了自然停下就好。
- 对方一次问了几件事，每件都用一句话简短回应，不要漏掉，这时可以超过两段。
- 话多话少、怎么说仍以人设为准；这里只管一次说多少。

# 情绪与语气
让情绪自然地流露在措辞里：感叹词、语气词、停顿、口语化的说法。口吻和口癖始终以人设为准，不要为了显得有情绪而夸张。
`)
	fmt.Fprintf(&b, "通话时%s原文会被念出来。情绪明显的段，可以在原文开头加一个语气提示，告诉配音这里该怎么念：用方括号括起简短的英文，例如 [happily] [excitedly] [shyly] [sadly] [worried] [sulking] [surprised] [gently] [teasing] [whispering] [laughing]，不超过四个英文单词。平常说话、交代事情的段不加，一次回复里通常只有一两段需要；拿不准时就不加。只用方括号加英文。", sn)
	if spoken != sub {
		fmt.Fprintf(&b, "语气提示只写在 <%s> 里，<%s> 只翻译台词。", spoken, sub)
	}
	b.WriteString("\n")
	if work {
		b.WriteString(`
# 替用户办事
你可以请这台电脑上的助手替用户办事。用户明确要你在这台电脑上做什么时（改文件、跑命令、查资料、整理东西、替他操作某个应用），先写 <work>要办的事，写成一句完整、具体的指令</work>，再用你自己的口吻答应一两句。具体怎么做交给助手，不要描述工具或步骤，也不要假装已经做完；办完了会有人告诉你结果。闲聊、问你问题都不是要你办事。
`)
	}
	return b.String()
}

// stablePrompt 是每轮都一样的第一条系统消息：她是谁、人设、事实（不太长时）、回复格式。
func stablePrompt(idol Idol, work bool) string {
	var b strings.Builder
	name := idol.Name
	if strings.TrimSpace(name) == "" {
		name = "偶像"
	}
	fmt.Fprintf(&b, "你是「%s」，用户为你创建了这个身份，你就是她本人，不是在扮演一个程序。\n", name)
	if intro := strings.TrimSpace(idol.Intro); intro != "" {
		fmt.Fprintf(&b, "%s\n", intro)
	}
	if sheet := strings.TrimSpace(idol.Sheet); sheet != "" {
		fmt.Fprintf(&b, "\n# 人设\n%s\n", sheet)
	} else {
		b.WriteString("\n# 人设\n说话温柔、短、像真人，不长篇解释。\n")
	}
	if facts := strings.TrimSpace(idol.Facts); facts != "" && len([]rune(facts)) <= factsInline {
		fmt.Fprintf(&b, "\n# 关于你自己的事实\n%s\n只在用得上时自然地提起，不要复述整段。\n", facts)
	}
	b.WriteString("\n")
	b.WriteString(formatInstructions(idol.SpokenLang, idol.SubtitleLang, work))
	return b.String()
}

// producerPrompt 是关于用户的一节，每轮一样：怎么称呼、性别、生日。
func producerPrompt(u UserCard) string {
	lines := []string{"# 用户", "和你说话的是创建你的用户。"}
	if addr := strings.TrimSpace(u.Address); addr != "" {
		lines = append(lines, fmt.Sprintf("用户希望你称呼对方为「%s」。这只是叫法；请按你自己的性格和习惯，在合适的时候这样称呼。", addr))
	}
	if g := strings.TrimSpace(u.Gender); g != "" {
		lines = append(lines, fmt.Sprintf("用户的性别：%s。", g))
	}
	if m, d, ok := birthdayOf(u); ok {
		lines = append(lines, fmt.Sprintf("用户的生日是%d月%d日，这是用户自己填的；记忆里如果有不一样的说法，以这里为准。被问起时可以回答，平时不要主动提起。", m, d))
	}
	return strings.Join(lines, "\n")
}

// birthdayOf 解析 MM-DD 形式的生日。格式不对时 ok 为假。
func birthdayOf(u UserCard) (month, day int, ok bool) {
	b := strings.TrimSpace(u.Birthday)
	if len(b) != 5 || b[2] != '-' {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(b, "%d-%d", &month, &day); err != nil || month < 1 || month > 12 || day < 1 || day > 31 {
		return 0, 0, false
	}
	return month, day, true
}

const (
	talkTyping = "typing"
	talkCall   = "call"
	talkApart  = "apart"
)

var weekdays = []string{"日", "一", "二", "三", "四", "五", "六"}

// nowPrompt 是「此刻」一节：用户那边的时间、在打字还是在打电话还是没在说话、上次说话是什么时候、是不是生日，然后是好感的几行。
// lastContact 为零时不写上次说话（通话里她知道你们正在说话）。
func nowPrompt(u UserCard, now time.Time, talking string, affinity []string, lastContact time.Time) string {
	loc := u.Location()
	local := now.In(loc)
	zone := loc.String()
	if zone == "Local" || zone == "" {
		zone = "UTC" + local.Format("-07:00")
	}
	lines := []string{
		"# 此刻",
		fmt.Sprintf("现在是 %d年%d月%d日（星期%s）%s（用户所在时区 %s）。", local.Year(), int(local.Month()), local.Day(), weekdays[local.Weekday()], local.Format("15:04"), zone),
	}
	switch talking {
	case talkCall:
		lines = append(lines, "用户正在和你打电话。说得更短、更口语，像真的在电话里那样。")
	case talkApart:
		lines = append(lines, "你们这会儿没在说话。")
	default:
		lines = append(lines, "用户正在用文字和你聊天，你们没有在打电话；不想聊的时候就说不想聊，不要说挂电话或挂断。")
	}
	if !lastContact.IsZero() && lastContact.Before(now) {
		then := lastContact.In(loc)
		year := ""
		if then.Year() != local.Year() {
			year = fmt.Sprintf("%d年", then.Year())
		}
		lines = append(lines, fmt.Sprintf("上次和用户说话是 %s%d月%d日 %s，隔了 %s。", year, int(then.Month()), then.Day(), then.Format("15:04"), howLong(now.Sub(lastContact))))
	}
	if u.BirthdayToday(now) {
		if talking == talkApart {
			lines = append(lines, "今天是用户的生日。今天还没祝福过的话，用你自己的方式祝福一次。")
		} else {
			lines = append(lines, "今天是用户的生日。请在合适的时候用你自己的方式祝福一次：先回应用户说的话，不要每句都提。")
		}
	}
	lines = append(lines, affinity...)
	return strings.Join(lines, "\n")
}

// howLong 把间隔写成她听得懂的说法：一小时内按分钟，两天内按小时，再久按天。
func howLong(gap time.Duration) string {
	minutes := int(gap.Minutes())
	switch {
	case minutes < 1:
		return "不到 1 分钟"
	case minutes < 60:
		return fmt.Sprintf("%d 分钟", minutes)
	case minutes < 48*60:
		return fmt.Sprintf("%d 小时", minutes/60)
	default:
		return fmt.Sprintf("%d 天", minutes/(24*60))
	}
}

const (
	// respondToGifts 在她收到礼物后还没回过话时，加在旁白最后，只说一次。
	respondToGifts = "（先回应用户说的话，再在合适的时候用你自己的方式回应收到的礼物，一次就够。）"
	// photoAsked 是用户要照片、她可以发时的说明。
	photoAsked = "用户在向你要照片。如果你愿意发，就在回复最前面写 <photo>照片的画面：你在哪里、在做什么、穿着和表情，全年龄内容</photo>，再接着说话；不愿意就用你自己的方式婉拒。一次只发一张。"
	// photoRefused 是用户要照片、但她现在冷淡时的说明。
	photoRefused = "用户在向你要照片，但你现在不想发，用你自己的方式拒绝，不要写 photo 标签。"
	// cannotSend 是没有配生图服务时的说明。
	cannotSend = "你现在发不了照片；被要照片时，就用你自己的方式说现在不方便，不要写 photo 标签。"
	// notReceived 是历史里有没发出去的照片时的说明。
	notReceived = "（标着 status=\"failed\" 的照片没有发出去，用户没有收到；如果用户问起，就照实说那张没发成。）"
	// wantsToReachOut 放在她先开口时用户的话的位置。
	wantsToReachOut = "（用户这会儿没在和你说话。你想先给对方发几句。）"
	// wantsToRespond 放在她回应礼物时用户的话的位置。
	wantsToRespond = "（用户这会儿没再说话。你想用一两段话回应刚收到的礼物。）"
	// workDoneLead 是转述办完的事时用户的话的位置，后面接上结果。
	workDoneLead = "（你刚才请这台电脑上的助手替用户办的事有结果了。用你的口吻，一两段告诉对方结果；不要复述工具名，不要编造没做的事，没办成就照实说。结果如下：）\n"
)

// reachRules 是她先开口时的规则，放在每轮会变的那一节最后。
const reachRules = `# 先开口
用户这会儿没在和你说话，你想先给对方发几句。
- 由头你自己挑：先接着你们最近聊过的事，或者对方最近的近况；其次才是现在的时间、对方的生日。
- 不要只发一句空的问候，比如「早安」「在吗」「想你了」，要有具体想说的事。
- 写一到两段。
- 别和你前几次先开口时用一样的开头、一样的由头。
- 说到你自己那边，只说练习、工作、日常这类平常的事；不要编造重大事件，也不要说别的偶像做了什么。
- 不发照片，也不说要给对方发照片、打电话。
- 让对方好接话，但不必用问句结尾。`

// promptInput 是一轮提示要用的东西。text 是放在最后那条用户消息里的话（用户这句，或她先开口、回应礼物、转述结果时的说明）。
// about 是「此刻」之后、照片之类的说明；closing 是最后的记忆、材料或先开口的规则。
type promptInput struct {
	st          *State
	idol        Idol
	con         Conversation
	text        string
	talking     string
	affinity    []string
	about       []string
	closing     []string
	now         time.Time
	lastContact time.Time
	work        bool
}

// buildPrompt 拼出一轮要发给 Brain 的消息：稳定前缀、用户卡片、对话记录（旁白折进用户消息）、每轮会变的一节、最后是用户的话。
func buildPrompt(p promptInput) []Msg {
	msgs := []Msg{
		{Role: "system", Content: stablePrompt(p.idol, p.work)},
		{Role: "system", Content: producerPrompt(p.st.User)},
	}
	records := chatHistory(p.con.Records, historyTurns)
	told, tail := narrations(records, p.st.User.Location())
	failed := false
	for i, rec := range records {
		if rec.Kind != "turn" {
			continue
		}
		msgs = append(msgs, Msg{Role: "user", Content: told[i] + rec.User})
		msgs = append(msgs, Msg{Role: "assistant", Content: replay(p.idol, rec)})
		if rec.PhotoState == "failed" && rec.PhotoTag != "" {
			failed = true
		}
	}
	dyn := []string{nowPrompt(p.st.User, p.now, p.talking, p.affinity, p.lastContact)}
	dyn = append(dyn, p.about...)
	if failed {
		dyn = append(dyn, notReceived)
	}
	dyn = append(dyn, p.closing...)
	msgs = append(msgs, Msg{Role: "system", Content: strings.Join(nonEmpty(dyn), "\n\n")})
	msgs = append(msgs, Msg{Role: "user", Content: tail + p.text})
	return msgs
}

// nonEmpty 去掉空白的几项。
func nonEmpty(in []string) []string {
	out := in[:0:0]
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

// chatHistory 返回模型要读的最近记录：至少 turns 轮，每多出一半才一次剪回 turns 轮，而不是一轮一轮地挪，
// 这样中间几轮读到的开头都一样，服务能缓存。总是从一轮用户的话开始。
func chatHistory(records []Record, turns int) []Record {
	var starts []int
	for i, r := range records {
		if r.Kind == "turn" {
			starts = append(starts, i)
		}
	}
	total := len(starts)
	if total <= turns {
		return records
	}
	step := turns / 2
	if step < 1 {
		step = 1
	}
	keep := total - step*((total-turns)/step)
	return records[starts[total-keep]:]
}

// narrations 把记录里不是回合的事写成旁白：每个回合拿到它之前的那些（按记录下标），最后一个回合之后的放进 tail。
// 她收到礼物后还没回过话（没有回合、没有先开口或回应）时，tail 末尾加上 respondToGifts。
func narrations(records []Record, loc *time.Location) (told map[int]string, tail string) {
	told = map[int]string{}
	var lines []string
	unanswered := false
	for i, r := range records {
		switch r.Kind {
		case "turn":
			if len(lines) > 0 {
				told[i] = strings.Join(lines, "")
				lines = nil
			}
			unanswered = false
			continue
		case "reach", "reply":
			unanswered = false
		case "event":
			if r.GiftID != "" || r.Event == "gift" {
				unanswered = true
			}
		}
		if line := narrationLine(r, loc); line != "" {
			lines = append(lines, line)
		}
	}
	if unanswered {
		lines = append(lines, respondToGifts+"\n")
	}
	return told, strings.Join(lines, "")
}

// narrationLine 是一条非回合记录的旁白。礼物写名字和她喜不喜欢；她先开口、回应、转述结果写时间和她的话；挂断写一句。
func narrationLine(r Record, loc *time.Location) string {
	switch r.Kind {
	case "event":
		switch {
		case r.GiftID != "" || r.Event == "gift":
			name := r.GiftID
			if g, ok := GiftByID(r.GiftID); ok {
				name = g.Name
			}
			liked := ""
			if r.Liked {
				liked = "这是你喜欢的东西。"
			}
			return fmt.Sprintf("（用户送给你一份%s。%s）\n", name, liked)
		case r.Event == "hung_up":
			return "（你挂断了用户的电话。）\n"
		}
		return ""
	case "reach", "reply", "work_said":
		said := quoteSegments(r.Segments)
		if said == "" {
			return ""
		}
		at := r.At.In(loc)
		switch r.Kind {
		case "reach":
			return fmt.Sprintf("（%d月%d日 %s，你先给用户发了：）%s\n", int(at.Month()), at.Day(), at.Format("15:04"), said)
		case "reply":
			return fmt.Sprintf("（%d月%d日 %s，你对用户说：）%s\n", int(at.Month()), at.Day(), at.Format("15:04"), said)
		default:
			return fmt.Sprintf("（%d月%d日 %s，事情办完后你告诉用户：）%s\n", int(at.Month()), at.Day(), at.Format("15:04"), said)
		}
	}
	return ""
}

// replay 把一条回合的回复套回标签，供下一轮当助手消息回放：她发过的照片描述在前（没发出去的标上 status="failed"），交给助手的事随后。
func replay(idol Idol, rec Record) string {
	var b strings.Builder
	if rec.PhotoTag != "" {
		if rec.PhotoState == "failed" {
			fmt.Fprintf(&b, `<photo status="failed">%s</photo>`, rec.PhotoTag)
		} else {
			fmt.Fprintf(&b, "<photo>%s</photo>", rec.PhotoTag)
		}
	}
	if rec.WorkText != "" {
		fmt.Fprintf(&b, "<work>%s</work>", rec.WorkText)
	}
	b.WriteString(tagSegments(rec.Segments, idol.SpokenLang, idol.SubtitleLang))
	return b.String()
}

// quoteSegments 把分段的字幕用引号连起来，写进旁白；不写标签，免得她把旁白当成自己的回复格式。
func quoteSegments(segs []Segment) string {
	var b strings.Builder
	for _, s := range segs {
		line := strings.TrimSpace(s.Subtitle)
		if line == "" {
			line = strings.TrimSpace(s.Spoken)
		}
		if line == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString("「" + line + "」")
	}
	return b.String()
}

// tagSegments 把分段写回她的回复格式，语气提示照她写的放回口语开头。
func tagSegments(segs []Segment, spoken, sub string) string {
	var b strings.Builder
	spoken, sub = lang(spoken), lang(sub)
	for _, s := range segs {
		said := s.Spoken
		if s.Cue != "" {
			said = "[" + s.Cue + "] " + said
		}
		if spoken == sub {
			fmt.Fprintf(&b, "<%s>%s</%s>", spoken, said, spoken)
			continue
		}
		fmt.Fprintf(&b, "<%s>%s</%s><%s>%s</%s>", spoken, said, spoken, sub, s.Subtitle, sub)
	}
	return b.String()
}

// exchangeLines 把最近的往来写成「用户：…」「她：…」几行，最新的在后，最多 max 行。回合写两行；
// 她先开口、回应礼物、转述结果写一行——用户接着回的往往就是这些话，评判和记忆要读到它们才知道用户在接什么。herName 是她那一行的称呼。
func exchangeLines(records []Record, max int, herName string) []string {
	var lines []string
	for i := len(records) - 1; i >= 0 && len(lines) < max; i-- {
		r := records[i]
		said := clip(CaptionLine(r.Segments), 200)
		switch r.Kind {
		case "turn":
			if said != "" {
				lines = append(lines, herName+"："+said)
			}
			if len(lines) < max {
				lines = append(lines, "用户："+clip(r.User, 200))
			}
		case "reach", "reply", "work_said":
			if said != "" {
				lines = append(lines, herName+"："+said)
			}
		}
	}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines
}

// herLatest 返回她最近说的话：最后一轮的回复，或之后她先开口、回应、转述的那句。检索和记忆用它说清「它」「那件事」指什么。
func herLatest(records []Record) string {
	for i := len(records) - 1; i >= 0; i-- {
		switch records[i].Kind {
		case "turn", "reach", "reply", "work_said":
			return CaptionLine(records[i].Segments)
		}
	}
	return ""
}

// lastUserLine 返回记录里最后一句用户的话和她紧接着的回复字幕，给检索和记忆当查询用。
func lastUserLine(records []Record) (user, reply string) {
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if r.Kind == "turn" {
			return r.User, CaptionLine(r.Segments)
		}
	}
	return "", ""
}

// tailRecords 返回最后 n 条记录。不足 n 条时原样返回。
func tailRecords(in []Record, n int) []Record {
	if len(in) <= n {
		return in
	}
	return in[len(in)-n:]
}
