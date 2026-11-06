package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Shenchangxin/yoyo/internal/dsh"
	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/osutil"
	"github.com/Shenchangxin/yoyo/internal/reach"
	"github.com/Shenchangxin/yoyo/internal/webcontent"
)

var reachFileMu sync.Mutex

// reachFacts 把一条还没说过的公开检索交给开口门。检索本身在后台，不占这一轮刷新。
// 没到点或上一轮还在跑时什么都不加。画像、新闻和热榜轮流查；搜不到就沉默，不编一条新闻。
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
	if log.Pending != nil && seenIDs[reach.ID(log.Pending.URL)] {
		log.Pending = nil
		a.writeReachLog(log)
	}
	if fact, ok := reach.Emit(log.Pending, seenIDs); ok {
		return []initiative.Fact{fact}
	}
	lines := a.reachLines()
	probe, ok := reach.Next(log, lines, now)
	if !ok || !a.reachMu.TryLock() {
		return nil
	}
	go a.runReach(probe, log, seenIDs)
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

func (a *App) runReach(probe reach.Probe, log reach.Log, seen map[string]bool) {
	defer a.reachMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 28*time.Second)
	defer cancel()
	now := time.Now()
	items := searchProbe(ctx, probe)
	if item := reach.Choose(items, seen); item != nil {
		item.Line = probe.Line
		item.Note = a.reachNote(*item)
		if item.Note != "" {
			log.Pending = item
		}
	}
	log.Cursor++
	log.Next = now.Add(reach.Gap(now)).Format(time.RFC3339)
	a.writeReachLog(log)
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

func (a *App) writeReachLog(log reach.Log) {
	reachFileMu.Lock()
	defer reachFileMu.Unlock()
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

材料只有一条公开检索，和它要对上的那句画像。只用材料里出现过的话。材料里没有的数字、日期、价格、名额都不要写。

先用一句话说明它和这句画像有没有关系。对得上，就说对在哪。对不上、但这条本身值得一看，就明说这是一条轻松的消息，跟手头的工作无关，想看再看。

然后用两三句讲检索实际说了什么。

最后写最小的下一步，并写明现在忽略也没有任何影响。不要催人立刻去做。

链接单独放在第二行。不要用标题、编号列表或 JSON。不值得打扰时，只输出一个字：静。`

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
	if line := strings.TrimSpace(item.Line); line != "" {
		b.WriteString("要对上的画像 " + line + "\n")
	} else {
		b.WriteString("这是外部热榜或新闻，不一定对得上手头的工作。\n")
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
	if line := strings.TrimSpace(item.Line); line != "" {
		b.WriteString("跟你记过的「" + line + "」对得上。\n")
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

// searchProbe 先走宿主里的公开渠道：网页和 X 用免密钥的 Exa，B 站热门和 V2EX 热帖用各自的公开接口。
// 这一步没有结果时，才试 PATH 上的 agent-reach。仍然没有，就用宿主的公开网页检索。
func searchProbe(ctx context.Context, probe reach.Probe) []reach.Item {
	if items := nativeProbe(ctx, probe); len(items) > 0 {
		return items
	}
	if bin, err := exec.LookPath("agent-reach"); err == nil && bin != "" {
		args := reach.Command(probe)
		cmd := exec.CommandContext(ctx, bin, args...)
		osutil.HideConsole(cmd)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		_ = cmd.Run()
		if items := reach.ParseEnvelope(stdout.String()); len(items) > 0 {
			return items
		}
	}
	query := strings.TrimSpace(probe.Fallback)
	if query == "" {
		query = strings.TrimSpace(probe.Query)
	}
	if query == "" {
		return nil
	}
	text, err := webcontent.SearchJina(ctx, http.DefaultClient, "", query)
	if err != nil {
		return nil
	}
	return reach.ParseHost(text)
}

// nativeProbe 在进程内完成 Agent-Reach 公开路径上的四个动作。登录、浏览器和字幕不在这里。
func nativeProbe(ctx context.Context, probe reach.Probe) []reach.Item {
	switch probe.Channel {
	case "bilibili":
		if probe.Action != "hot" {
			return nil
		}
		hits, err := webcontent.BilibiliPopular(ctx, http.DefaultClient, 5)
		if err != nil {
			return nil
		}
		return hitsToItems(hits)
	case "v2ex":
		if probe.Action != "hot" {
			return nil
		}
		hits, err := webcontent.V2EXHot(ctx, http.DefaultClient, 5)
		if err != nil {
			return nil
		}
		return hitsToItems(hits)
	case "web", "twitter":
		query := strings.TrimSpace(probe.Query)
		if probe.Channel == "twitter" {
			if query == "" {
				query = "热点"
			}
			query = "(" + query + ") (site:x.com/ OR site:twitter.com/)"
		}
		if query == "" {
			query = strings.TrimSpace(probe.Fallback)
		}
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
	default:
		return nil
	}
}

func hitsToItems(hits []webcontent.Hit) []reach.Item {
	var out []reach.Item
	for _, hit := range hits {
		out = append(out, reach.Item{Title: hit.Title, URL: hit.URL, Snippet: hit.Snippet})
	}
	return out
}
