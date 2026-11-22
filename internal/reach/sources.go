// 本文件拥有公开检索的来源表、默认话题，以及按用户忙闲伸缩的额度和间隔。
// 不拥有发请求、改写便条和记账。调用方是 app 的 reachFacts。来源都是进程内的公开接口，不调用外部命令。

package reach

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// KindFeed 是一条固定的 RSS 或 Atom。
	KindFeed = "feed"
	// KindHN 是 Hacker News 的按时间检索。
	KindHN = "hn"
	// KindArxiv 是 arXiv 的 cs.AI / cs.CL / cs.LG 新论文。
	KindArxiv = "arxiv"
	// KindGitHub 是 GitHub 近两周有更新的仓库检索。
	KindGitHub = "github"
	// KindDev 是 DEV 近一周的标签热文。
	KindDev = "devto"
	// KindLobsters 是 Lobsters 的 ai 标签。
	KindLobsters = "lobsters"
	// KindV2EX 是 V2EX 的 ai 节点。
	KindV2EX = "v2ex"
	// KindWeb 是免密钥的公开网页检索。Query 已经带好话题。
	KindWeb = "web"

	// PaceBusy 是用户这会儿在用 yoyo：回合还在跑，或二十分钟内说过话。
	PaceBusy = "busy"
	// PaceSteady 是二十分钟到三小时之间用过。
	PaceSteady = "steady"
	// PaceQuiet 是三小时内没在会话里说话，或还没有说过。应用开着、人在别处时用这一档。
	PaceQuiet = "quiet"

	// Courtesy 是刚说完话之后先不插资讯的间隔。短于忙档的开口间隔，只避开紧接着的下一句。
	Courtesy = 10 * time.Minute
)

// DefaultTopics 是每轮检索都带着的兴趣。用户自己的项目和事实句会再拼上去，不会换成别的话题。
var DefaultTopics = []string{"AI", "Agent", "人工智能", "大模型", "AI agent"}

// Source 是来源表里的一格。OnTopic 为真时，标题里不出现话题词也保留，因为这个来源本身就在话题上。
type Source struct {
	ID      string
	Name    string
	Kind    string
	URL     string
	OnTopic bool
}

// Probe 是一轮只读检索。Query 是要发出去的词，订阅源不用它拉正文，只在这一格没有结果时交给公开网页检索。
// Line 是要对上的画像短句，可以空。Topic 是这一轮轮到的默认话题。
type Probe struct {
	ID      string
	Name    string
	Kind    string
	URL     string
	Query   string
	Line    string
	Topic   string
	OnTopic bool
}

// Allowance 是某一忙闲档一天能说几条、两条之间至少隔多久、两次检索之间隔多久。
type Allowance struct {
	Daily int
	Speak time.Duration
	Probe time.Duration
}

// Catalog 返回固定来源。顺序中英交错，这样连续几条不会全是同一种语言。
// 地址都在这次改动前核对过能直接取到 RSS、Atom 或 JSON。登录墙和没有稳定订阅的站点不放进来。
func Catalog() []Source {
	return []Source{
		{ID: "hn", Name: "Hacker News", Kind: KindHN, OnTopic: true},
		{ID: "qbitai", Name: "量子位", Kind: KindFeed, URL: "https://www.qbitai.com/feed", OnTopic: true},
		{ID: "openai", Name: "OpenAI", Kind: KindFeed, URL: "https://openai.com/news/rss.xml", OnTopic: true},
		{ID: "oschina", Name: "开源中国", Kind: KindFeed, URL: "https://www.oschina.net/news/rss", OnTopic: false},
		{ID: "arxiv", Name: "arXiv", Kind: KindArxiv, OnTopic: true},
		{ID: "v2ex", Name: "V2EX", Kind: KindV2EX, OnTopic: true},
		{ID: "verge", Name: "The Verge", Kind: KindFeed, URL: "https://www.theverge.com/rss/ai-artificial-intelligence/index.xml", OnTopic: true},
		{ID: "github", Name: "GitHub", Kind: KindGitHub, OnTopic: true},
		{ID: "solidot", Name: "Solidot", Kind: KindFeed, URL: "https://www.solidot.org/index.rss", OnTopic: false},
		{ID: "deepmind", Name: "Google DeepMind", Kind: KindFeed, URL: "https://deepmind.google/blog/rss.xml", OnTopic: true},
		{ID: "devto", Name: "DEV", Kind: KindDev, OnTopic: true},
		{ID: "huggingface", Name: "Hugging Face", Kind: KindFeed, URL: "https://huggingface.co/blog/feed.xml", OnTopic: true},
		{ID: "lobsters", Name: "Lobsters", Kind: KindLobsters, OnTopic: true},
		{ID: "techcrunch", Name: "TechCrunch", Kind: KindFeed, URL: "https://techcrunch.com/category/artificial-intelligence/feed/", OnTopic: true},
		{ID: "google-ai", Name: "Google AI", Kind: KindFeed, URL: "https://blog.google/technology/ai/rss/", OnTopic: true},
		{ID: "marktechpost", Name: "MarkTechPost", Kind: KindFeed, URL: "https://www.marktechpost.com/feed/", OnTopic: true},
		{ID: "simon", Name: "Simon Willison", Kind: KindFeed, URL: "https://simonwillison.net/atom/entries/", OnTopic: true},
		{ID: "bilibili", Name: "哔哩哔哩", Kind: KindWeb, OnTopic: true},
		{ID: "web", Name: "公开网页", Kind: KindWeb, OnTopic: true},
	}
}

// AllowanceFor 按 pace 给出额度。认不出的档位按空闲，也就是一天更多、间隔更短。
// 忙：一天 6 条，两条至少隔 90 分钟，检索大约 70 分钟一次。
// 平稳：一天 8 条，隔 70 分钟，检索大约 40 分钟一次。
// 空闲：一天 12 条，隔 35 分钟，检索大约 22 分钟一次。
func AllowanceFor(pace string) Allowance {
	switch pace {
	case PaceBusy:
		return Allowance{Daily: 6, Speak: 90 * time.Minute, Probe: 70 * time.Minute}
	case PaceSteady:
		return Allowance{Daily: 8, Speak: 70 * time.Minute, Probe: 40 * time.Minute}
	default:
		return Allowance{Daily: 12, Speak: 35 * time.Minute, Probe: 22 * time.Minute}
	}
}

// ProbeAfter 是这一档的检索间隔，再加上当天里最多 5 分钟，避免每次都卡在同一个整分。
// pace 交给 AllowanceFor。now 为零时用当前时间取天数。
func ProbeAfter(pace string, now time.Time) time.Duration {
	if now.IsZero() {
		now = time.Now()
	}
	return AllowanceFor(pace).Probe + time.Duration(now.YearDay()%6)*time.Minute
}

// PaceOf 看用户最近一次说话离 now 有多远。running 为真时直接算忙，不管 userAt。
// userAt 为零，或早于 now 超过三小时，算空闲。二十分钟以内算忙，其间到三小时算平稳。
func PaceOf(userAt time.Time, running bool, now time.Time) string {
	if running {
		return PaceBusy
	}
	if userAt.IsZero() || now.IsZero() {
		return PaceQuiet
	}
	age := now.Sub(userAt)
	if age < 0 {
		return PaceBusy
	}
	switch {
	case age < 20*time.Minute:
		return PaceBusy
	case age < 3*time.Hour:
		return PaceSteady
	default:
		return PaceQuiet
	}
}

// Hold 在回合还在跑，或用户刚说完还没过 Courtesy 时为真。这段时间先不把资讯写进会话。
// userAt 为零且 running 为假时不按住。now 为零时不按刚说完的那一档。
func Hold(userAt time.Time, running bool, now time.Time) bool {
	if running {
		return true
	}
	if userAt.IsZero() || now.IsZero() {
		return false
	}
	age := now.Sub(userAt)
	return age >= 0 && age < Courtesy
}

// Next 到点且没有待说结果时，按 Cursor 轮下一个来源，并带上一个默认话题。
// lines 里有短句时，检索词是「话题 + 那一句」；订阅源的地址不变，这句只留给便条和没有结果时的补检索。
// log.Pending 还在，或还没到 log.Next，返回假。
func Next(log Log, lines []string, now time.Time) (Probe, bool) {
	if log.Pending != nil || !Due(log, now) {
		return Probe{}, false
	}
	cat := Catalog()
	if len(cat) == 0 || len(DefaultTopics) == 0 {
		return Probe{}, false
	}
	src := cat[slot(log.Cursor, len(cat))]
	topic := DefaultTopics[slot(log.Cursor, len(DefaultTopics))]
	line := ""
	if len(lines) > 0 {
		line = lines[slot(log.Cursor, len(lines))]
	}
	return Probe{
		ID: src.ID, Name: src.Name, Kind: src.Kind, URL: src.URL,
		Query: searchQuery(src, topic, line), Line: line, Topic: topic, OnTopic: src.OnTopic,
	}, true
}

// Prefer 把更贴近话题或画像的结果排到前面。onTopic 为假时，对不上的结果丢掉。
// topic 和 line 参与打分；两者都空时，只靠话题词。
func Prefer(items []Item, onTopic bool, topic, line string) []Item {
	type scored struct {
		item  Item
		score int
		index int
	}
	var kept []scored
	for i, item := range items {
		text := item.Title + "\n" + item.Snippet
		score := 0
		if line != "" && containsFold(text, line) {
			score = 2
		} else if topicHit(text) || containsFold(text, topic) {
			score = 1
		}
		if !onTopic && score == 0 {
			continue
		}
		kept = append(kept, scored{item: item, score: score, index: i})
	}
	for i := 1; i < len(kept); i++ {
		j := i
		for j > 0 && (kept[j].score > kept[j-1].score || (kept[j].score == kept[j-1].score && kept[j].index < kept[j-1].index)) {
			kept[j], kept[j-1] = kept[j-1], kept[j]
			j--
		}
	}
	out := make([]Item, len(kept))
	for i, one := range kept {
		out[i] = one.item
	}
	return out
}

func searchQuery(src Source, topic, line string) string {
	topic = strings.TrimSpace(topic)
	line = strings.TrimSpace(line)
	q := topic
	if line != "" {
		q = strings.TrimSpace(topic + " " + line)
	}
	if src.ID == "bilibili" {
		q = strings.TrimSpace("site:bilibili.com " + q)
	}
	if q == "" {
		q = "AI agent"
	}
	return q
}

func slot(n, size int) int {
	if size <= 0 {
		return 0
	}
	n %= size
	if n < 0 {
		n += size
	}
	return n
}

func containsFold(text, piece string) bool {
	piece = strings.TrimSpace(piece)
	if piece == "" || utf8.RuneCountInString(piece) < 2 {
		return false
	}
	lowerText := strings.ToLower(text)
	lowerPiece := strings.ToLower(piece)
	if latinPhrase(lowerPiece) {
		return hasLatinWord(lowerText, lowerPiece)
	}
	return strings.Contains(lowerText, lowerPiece)
}

func latinPhrase(text string) bool {
	for _, r := range text {
		if r > 127 {
			return false
		}
	}
	return true
}

func topicHit(text string) bool {
	lower := strings.ToLower(text)
	for _, word := range []string{"ai", "llm", "gpt", "agent"} {
		if hasLatinWord(lower, word) {
			return true
		}
	}
	for _, word := range []string{"人工智能", "大模型", "智能体", "机器学习", "深度学习"} {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

func hasLatinWord(lower, word string) bool {
	for i := 0; i+len(word) <= len(lower); {
		j := strings.Index(lower[i:], word)
		if j < 0 {
			return false
		}
		j += i
		before := j == 0 || !isLatin(lower[j-1])
		after := j+len(word) == len(lower) || !isLatin(lower[j+len(word)])
		if before && after {
			return true
		}
		i = j + len(word)
	}
	return false
}

func isLatin(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}
