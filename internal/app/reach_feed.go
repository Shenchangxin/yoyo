package app

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/reach"
	"github.com/Shenchangxin/yoyo/internal/webcontent"
)

var reachFileMu sync.Mutex

// reachFacts 把一条还没说过的公开检索交给开口门。检索本身在后台，不占这一轮刷新。
// 没到点或上一轮还在跑时什么都不加。来源按表轮换，每轮都带默认话题；搜不到就沉默，不编一条新闻。
// seen 里 reach: 开头的编号，以及日志里模型跳过的编号，都不再说。now 用来判断这一轮到不到点。
func (a *App) reachFacts(seen map[string]string, now time.Time) []initiative.Fact {
	if a == nil || a.Home == nil {
		return nil
	}
	log := a.readReachLog()
	seenIDs := map[string]bool{}
	for id := range seen {
		if strings.HasPrefix(id, "reach:") {
			seenIDs[id] = true
		}
	}
	for _, id := range log.Skip {
		seenIDs[id] = true
	}
	if log.Pending != nil && seenIDs[reach.ID(log.Pending.URL)] {
		a.updateReachLog(func(current *reach.Log) {
			if current.Pending != nil && seenIDs[reach.ID(current.Pending.URL)] {
				current.Pending = nil
			}
		})
		log.Pending = nil
	}
	if fact, ok := reach.Emit(log.Pending, seenIDs); ok {
		return []initiative.Fact{fact}
	}
	lines := a.reachLines()
	probe, ok := reach.Next(log, lines, now)
	if !ok || !a.reachMu.TryLock() {
		return nil
	}
	go a.runReach(probe, seenIDs, a.reachPace(now))
	return nil
}

func (a *App) reachLines() []string {
	if a == nil || a.Memory == nil {
		return nil
	}
	// 检索词会发到外部搜索。档案里是本人说过的话，可能有名字和私事，不拿去检索。
	var texts []string
	now := time.Now()
	for _, item := range a.Memory.All() {
		if item.Staging || item.Source == "machine" || memory.Expired(item, now) {
			continue
		}
		if item.Kind != memory.KindProject && item.Kind != memory.KindSemantic {
			continue
		}
		texts = append(texts, item.Text)
	}
	return reach.Lines(texts)
}

func (a *App) runReach(probe reach.Probe, seen map[string]bool, pace string) {
	defer a.reachMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 28*time.Second)
	defer cancel()
	now := time.Now()
	var pending *reach.Item
	var skip string
	if item := reach.Choose(searchProbe(ctx, probe), seen); item != nil {
		item.Line = probe.Line
		item.Topic = probe.Topic
		item.Source = probe.Name
		item.Note = a.reachNote(*item)
		if item.Note != "" {
			pending = item
		} else {
			skip = reach.ID(item.URL)
		}
	}
	wait := reach.ProbeAfter(pace, now)
	a.updateReachLog(func(log *reach.Log) {
		if pending != nil {
			log.Pending = pending
		}
		if skip != "" {
			log.Skip = reach.Remember(log.Skip, skip)
		}
		log.Cursor++
		log.Next = now.Add(wait).Format(time.RFC3339)
	})
}

// reachPace 用最近一次用户发言和当前有没有回合或任务组在跑，选出忙闲档。now 是这一轮的时刻。
func (a *App) reachPace(now time.Time) string {
	return reach.PaceOf(a.reachUserAt(), a.reachBusyNow(), now)
}

// reachBudget 返回这一刻的开口额度，以及要不要先按住。now 传给 PaceOf 和 Hold。
func (a *App) reachBudget(now time.Time) (reach.Allowance, bool) {
	at := a.reachUserAt()
	running := a.reachBusyNow()
	return reach.AllowanceFor(reach.PaceOf(at, running, now)), reach.Hold(at, running, now)
}

// reachBusyNow 在有会话回合还没结束，或有任务组还在做、还在等人时为真。
func (a *App) reachBusyNow() bool {
	if a == nil {
		return false
	}
	if len(a.RunningIDs()) > 0 {
		return true
	}
	return a.groupsRunning()
}

// reachUserAt 读日志里的用户发言时间。没有或解析不了时返回零值。
func (a *App) reachUserAt() time.Time {
	if a == nil {
		return time.Time{}
	}
	at, err := time.Parse(time.RFC3339, a.readReachLog().UserAt)
	if err != nil {
		return time.Time{}
	}
	return at
}

// touchReachUser 记下用户刚刚在会话里说了话。now 为零或没有家目录时什么都不写。
func (a *App) touchReachUser(now time.Time) {
	if a == nil || a.Home == nil || now.IsZero() {
		return
	}
	a.updateReachLog(func(log *reach.Log) {
		log.UserAt = now.UTC().Format(time.RFC3339)
	})
}

func (a *App) reachPath() string {
	if a == nil || a.Home == nil {
		return ""
	}
	return filepath.Join(a.Home.Memory(), "reach.json")
}

func (a *App) readReachLog() reach.Log {
	reachFileMu.Lock()
	defer reachFileMu.Unlock()
	return a.readReachLogLocked()
}

func (a *App) readReachLogLocked() reach.Log {
	path := a.reachPath()
	if path == "" {
		return reach.Log{}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return reach.Log{}
	}
	var log reach.Log
	_ = json.Unmarshal(b, &log)
	return log
}

func (a *App) updateReachLog(fn func(*reach.Log)) {
	if fn == nil {
		return
	}
	reachFileMu.Lock()
	defer reachFileMu.Unlock()
	log := a.readReachLogLocked()
	fn(&log)
	a.writeReachLogLocked(log)
}

func (a *App) writeReachLogLocked(log reach.Log) {
	path := a.reachPath()
	if path == "" {
		return
	}
	b, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o600)
}

const reachNotePrompt = `你在主会话里留一张便条，像同事顺手丢来的一条消息。不是简报，不是卡片，也不要分栏。

材料只有一条公开检索。来源名和「默认关注」只说明这条是从哪来的、为什么会搜到它，不是材料里的事实。只用标题、链接和摘要里出现过的话。材料里没有的数字、日期、价格、名额都不要写。

默认关注是 AI、Agent、人工智能这一类话题。对得上这个话题，或对得上那句画像，就留一张便条，不要因为「只是一条新闻」回静。

先用一句话说明它和画像或默认关注有没有关系。对得上，就说对在哪。对不上、但这条本身值得一看，就明说这是一条轻松的消息，跟手头的工作无关，想看再看。

然后用两三句讲检索实际说了什么。

最后写最小的下一步，并写明现在忽略也没有任何影响。不要催人立刻去做。

链接单独放在第二行。不要用标题、编号列表或 JSON。材料是空的、没有公开链接、或只是登录墙和广告时，只输出一个字：静。`

// reachNote 把一条检索写成便条。模型认为不值得开口时返回空串，调用失败时用材料本身拼一句，不编新事实。
func (a *App) reachNote(item reach.Item) string {
	packet := reachPacket(item)
	if packet == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	out, err := liveModel{app: a}.Complete(ctx, dsh.Request{
		Messages: []dsh.Message{
			{Role: "system", Text: reachNotePrompt},
			{Role: "user", Text: packet},
		},
		MaxTokens: 700,
	})
	note := strings.TrimSpace(out.Text)
	if err != nil || note == "" || note == "静" || note == "沉默" {
		if err != nil {
			return reachFallback(item)
		}
		return ""
	}
	if url := strings.TrimSpace(item.URL); url != "" && !strings.Contains(note, url) {
		note = url + "\n" + note
	}
	return clipBrief(note, 860)
}

func reachPacket(item reach.Item) string {
	if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.URL) == "" {
		return ""
	}
	var b strings.Builder
	if src := strings.TrimSpace(item.Source); src != "" {
		b.WriteString("来源 " + src + "\n")
	}
	if topic := strings.TrimSpace(item.Topic); topic != "" {
		b.WriteString("默认关注 " + topic + "\n")
	}
	if line := strings.TrimSpace(item.Line); line != "" {
		b.WriteString("要对上的画像 " + line + "\n")
	} else {
		b.WriteString("没有单独的画像短句。默认关注已经写在上面。\n")
	}
	b.WriteString("标题 " + strings.TrimSpace(item.Title) + "\n")
	b.WriteString("链接 " + strings.TrimSpace(item.URL) + "\n")
	if snip := strings.TrimSpace(item.Snippet); snip != "" {
		b.WriteString("摘要 " + snip + "\n")
	}
	return b.String()
}

func reachFallback(item reach.Item) string {
	var b strings.Builder
	if src := strings.TrimSpace(item.Source); src != "" {
		b.WriteString("来自" + src + "。\n")
	}
	if line := strings.TrimSpace(item.Line); line != "" {
		b.WriteString("跟你记过的「" + line + "」对得上。\n")
	} else if topic := strings.TrimSpace(item.Topic); topic != "" {
		b.WriteString("一条关于「" + topic + "」的消息。\n")
	} else {
		b.WriteString("一条轻松的消息，跟手头的工作无关。\n")
	}
	b.WriteString(strings.TrimSpace(item.URL) + "\n")
	b.WriteString(strings.TrimSpace(item.Title))
	if snip := strings.TrimSpace(item.Snippet); snip != "" {
		b.WriteString("\n" + snip)
	}
	b.WriteString("\n最小的下一步：有空再点开看一眼。现在忽略也没有任何影响。")
	return b.String()
}

// searchProbe 只在进程内检索。来源没有结果，或综合站点对不上话题时，用同一句检索词再走一次公开网页检索。
// 不查找、不执行 agent-reach。ctx 传给具体来源；订阅和 JSON 接口最多用其中 8 秒，剩下的留给补检索。
func searchProbe(ctx context.Context, probe reach.Probe) []reach.Item {
	items := reach.Prefer(nativeProbe(ctx, probe), probe.OnTopic, probe.Topic, probe.Line)
	if len(items) > 0 {
		return items
	}
	query := strings.TrimSpace(probe.Query)
	if query == "" {
		query = strings.TrimSpace(probe.Topic)
	}
	if query == "" {
		return nil
	}
	text, err := webcontent.SearchJina(ctx, http.DefaultClient, "", query)
	if err != nil {
		return nil
	}
	return reach.Prefer(reach.ParseHost(text), false, probe.Topic, probe.Line)
}

// nativeProbe 按 probe.Kind 走一条公开接口。ctx 在订阅和 JSON 上再限 8 秒；公开网页检索用原来的 ctx。
// 失败或空结果返回 nil，由 searchProbe 补一次。
func nativeProbe(ctx context.Context, probe reach.Probe) []reach.Item {
	if probe.Kind == reach.KindWeb {
		return exaItems(ctx, probe.Query)
	}
	short, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var (
		hits []webcontent.Hit
		err  error
	)
	switch probe.Kind {
	case reach.KindHN:
		hits, err = webcontent.HackerNews(short, http.DefaultClient, probe.Query, 5)
	case reach.KindArxiv:
		hits, err = webcontent.ArxivRecent(short, http.DefaultClient, probe.Query, 5)
	case reach.KindGitHub:
		hits, err = webcontent.GitHubUpdated(short, http.DefaultClient, probe.Query, 5)
	case reach.KindDev:
		hits, err = webcontent.DevToHot(short, http.DefaultClient, probe.Topic, 5)
	case reach.KindLobsters:
		hits, err = webcontent.LobstersAI(short, http.DefaultClient, 5)
	case reach.KindV2EX:
		hits, err = webcontent.V2EXNode(short, http.DefaultClient, "ai", 5)
	case reach.KindFeed:
		hits, err = webcontent.ReadFeed(short, http.DefaultClient, probe.URL, 5)
	default:
		return nil
	}
	if err != nil {
		return nil
	}
	return hitsToItems(hits)
}

func exaItems(ctx context.Context, query string) []reach.Item {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	text, err := webcontent.SearchExaMCP(ctx, http.DefaultClient, "", query)
	if err != nil || !webcontent.Useful(text) {
		return nil
	}
	if items := reach.ParseExa(text); len(items) > 0 {
		return items
	}
	return reach.ParseHost(text)
}

func hitsToItems(hits []webcontent.Hit) []reach.Item {
	var out []reach.Item
	for _, hit := range hits {
		out = append(out, reach.Item{Title: hit.Title, URL: hit.URL, Snippet: hit.Snippet})
	}
	return out
}
