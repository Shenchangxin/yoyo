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

	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/memory"
	"github.com/Shenchangxin/yoyo/internal/osutil"
	"github.com/Shenchangxin/yoyo/internal/reach"
	"github.com/Shenchangxin/yoyo/internal/webcontent"
)

var reachFileMu sync.Mutex

// reachFacts 把一条还没说过的公开检索交给开口门。检索本身在后台，不占这一轮刷新。
// 没有画像、没到点、或命令还在跑时，什么都不加。搜不到就沉默，不编一条新闻。
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
	query, ok := reach.Plan(log, lines, now)
	if !ok || !a.reachMu.TryLock() {
		return nil
	}
	go a.runReach(query, log, seenIDs)
	return nil
}

func (a *App) reachLines() []string {
	if a == nil || a.Memory == nil {
		return nil
	}
	var texts []string
	for _, item := range a.Memory.All() {
		if item.Staging || item.Source == "machine" {
			continue
		}
		if item.Kind != memory.KindProject && item.Kind != memory.KindProfile && item.Kind != memory.KindSemantic {
			continue
		}
		texts = append(texts, item.Text)
	}
	return reach.Lines(texts)
}

func (a *App) runReach(query string, log reach.Log, seen map[string]bool) {
	defer a.reachMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
	defer cancel()
	now := time.Now()
	items, _ := searchReach(ctx, query)
	if item := reach.Choose(items, seen); item != nil {
		item.Line = query
		log.Pending = item
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

// searchReach 优先用 PATH 上的 agent-reach 做只读网页检索。没有这条命令时，用宿主的公开检索。
func searchReach(ctx context.Context, query string) ([]reach.Item, string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ""
	}
	if bin, err := exec.LookPath("agent-reach"); err == nil && bin != "" {
		cmd := exec.CommandContext(ctx, bin, "web", "search", query, "--limit", "3", "--timeout", "15")
		osutil.HideConsole(cmd)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		_ = cmd.Run()
		if items := reach.ParseEnvelope(stdout.String()); len(items) > 0 {
			return items, "agent-reach"
		}
	}
	text, err := webcontent.SearchJina(ctx, http.DefaultClient, "", query)
	if err != nil {
		return nil, ""
	}
	return reach.ParseHost(text), "host"
}
