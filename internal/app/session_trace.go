// 给检查器看的轨迹投影，以及 spill 正文读取。

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// spillAPIBytes 是 SpillBlob 返回正文的字节上限，超出则截断并标 Truncated。
const spillAPIBytes = 80_000

// SessionTrace 是给检查器看的轨迹投影、统计和产物预览。
type SessionTrace struct {
	SessionID string               `json:"session_id"`
	Title     string               `json:"title,omitempty"`
	Workspace string               `json:"workspace,omitempty"`
	Harness   string               `json:"harness,omitempty"`
	Model     string               `json:"model,omitempty"`
	CreatedAt time.Time            `json:"created_at,omitempty"`
	StartedAt time.Time            `json:"started_at,omitempty"`
	EndedAt   time.Time            `json:"ended_at,omitempty"`
	Stats     runtime.TraceStats   `json:"stats"`
	Events    []runtime.TraceEvent `json:"events"`
	Artifacts []TraceArtifact      `json:"artifacts"`
}

// TraceArtifact 是检查器列表里的一条产物，正文只带预览。
type TraceArtifact struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Label   string `json:"label"`
	Bytes   int    `json:"bytes,omitempty"`
	Preview string `json:"preview,omitempty"`
}

// SpillBlob 是按 id 读出的 spill 正文，超限时标 Truncated。
type SpillBlob struct {
	ID        string `json:"id"`
	Bytes     int    `json:"bytes"`
	Text      string `json:"text"`
	Truncated bool   `json:"truncated,omitempty"`
}

// SessionTrace 投影该会话的轨迹，并附上笔记、spill 和待审批等产物。空 id 返回错误。
func (a *App) SessionTrace(id string) (SessionTrace, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return SessionTrace{}, fmt.Errorf("empty session id")
	}
	meta, _ := a.GetSession(id)
	evs, err := a.Traces.Read(id)
	if err != nil {
		return SessionTrace{}, err
	}
	view := runtime.TrimTraceView(runtime.ProjectTrace(evs), 0)
	if view.Events == nil {
		view.Events = []runtime.TraceEvent{}
	}
	if view.Stats.Tokens == 0 {
		if sh := a.ContextUsage(id); sh.Tokens > 0 {
			view.Stats.Tokens = sh.Tokens
		}
	}
	spill := runtime.BindSpill(a.Home.Root, meta.Workspace, id)
	arts := a.collectTraceArtifacts(id, meta, spill)
	if arts == nil {
		arts = []TraceArtifact{}
	}
	var spillBytes int
	if spill != nil {
		for _, sid := range spill.ListIDs() {
			if sid == "notes" {
				continue
			}
			spillBytes += spill.Size(sid)
		}
	}
	view.Stats.SpillBytes = spillBytes
	model := meta.Model
	if model == "" {
		model = a.Config.Model
	}
	return SessionTrace{
		SessionID: id,
		Title:     meta.Title,
		Workspace: meta.Workspace,
		Harness:   meta.Harness,
		Model:     model,
		CreatedAt: meta.CreatedAt,
		StartedAt: view.StartedAt,
		EndedAt:   view.EndedAt,
		Stats:     view.Stats,
		Events:    view.Events,
		Artifacts: arts,
	}, nil
}

// SpillBlob 按 id 读出 spill 正文。超过字节上限则截断。
// sessionID 先 TrimSpace，空串则返回错误，否则传给 GetSession 和 BindSpill。
// blobID 先 TrimSpace，空串则返回错误，否则传给 spill.Get，并写入返回值的 ID。
func (a *App) SpillBlob(sessionID, blobID string) (SpillBlob, error) {
	sessionID = strings.TrimSpace(sessionID)
	blobID = strings.TrimSpace(blobID)
	if sessionID == "" || blobID == "" {
		return SpillBlob{}, fmt.Errorf("empty spill id")
	}
	meta, _ := a.GetSession(sessionID)
	spill := runtime.BindSpill(a.Home.Root, meta.Workspace, sessionID)
	if spill == nil {
		return SpillBlob{}, os.ErrNotExist
	}
	text, err := spill.Get(blobID)
	if err != nil {
		return SpillBlob{}, err
	}
	n := len(text)
	trunc := false
	if n > spillAPIBytes {
		text = text[:spillAPIBytes]
		trunc = true
	}
	return SpillBlob{ID: blobID, Bytes: n, Text: text, Truncated: trunc}, nil
}

// collectTraceArtifacts 收集轨迹文件、meta、harness、笔记、spill、索引、溢出文件和待审批。spill 条数有上限，只留末尾。
// id 用来拼轨迹和 meta 文件名，并传给 WorkspaceContextDir，也用来匹配待审批的 SessionID。
func (a *App) collectTraceArtifacts(id string, meta SessionMeta, spill *runtime.Spill) []TraceArtifact {
	var out []TraceArtifact
	if art, ok := fileArtifact("trajectory", id+".jsonl", filepath.Join(a.Home.Sessions(), id+".jsonl")); ok {
		out = append(out, art)
	}
	if art, ok := fileArtifact("meta", id+".meta.json", filepath.Join(a.Home.Sessions(), id+".meta.json")); ok {
		out = append(out, art)
	}
	if meta.Harness != "" {
		out = append(out, TraceArtifact{Kind: "harness", ID: meta.Harness, Label: "harness " + shortHash(meta.Harness)})
	}
	if spill != nil {
		if notes := runtime.ReadNotes(spill); notes != "" {
			out = append(out, TraceArtifact{
				Kind:    "notes",
				ID:      "notes",
				Label:   "session notes",
				Bytes:   len(notes),
				Preview: capPreview(notes),
			})
		}
		ids := spill.ListIDs()
		sort.Strings(ids)
		var spills []TraceArtifact
		for _, sid := range ids {
			if sid == "notes" {
				continue
			}
			spills = append(spills, TraceArtifact{
				Kind:  "spill",
				ID:    sid,
				Label: sid,
				Bytes: spill.Size(sid),
			})
		}
		// maxSpillArts 是投影里最多带上的 spill 条数，只留列表末尾。
		const maxSpillArts = 24
		if len(spills) > maxSpillArts {
			spills = spills[len(spills)-maxSpillArts:]
		}
		out = append(out, spills...)
	}
	if dir := runtime.WorkspaceContextDir(meta.Workspace, id); dir != "" {
		if art, ok := fileArtifact("index", "INDEX.md", filepath.Join(dir, "INDEX.md")); ok {
			out = append(out, art)
		}
	}
	if meta.Workspace != "" {
		over := filepath.Join(meta.Workspace, ".yoyo", "overflow")
		ents, err := os.ReadDir(over)
		if err == nil {
			for _, e := range ents {
				if e.IsDir() {
					continue
				}
				if art, ok := fileArtifact("overflow", e.Name(), filepath.Join(over, e.Name())); ok {
					out = append(out, art)
				}
			}
		}
	}
	if a.Gate != nil {
		for _, o := range a.Gate.Pending() {
			if o.Request.SessionID != id {
				continue
			}
			label := strings.TrimSpace(o.Request.Action + " " + o.Request.Command)
			out = append(out, TraceArtifact{
				Kind:    "approval",
				ID:      o.ID,
				Label:   firstLabel(label, "pending approval"),
				Preview: strings.TrimSpace(o.Request.Path),
			})
		}
	}
	return out
}

// fileArtifact 在路径是普通文件时做成一条产物。较小的文本文件带上预览。
// kind 写入产物的 Kind。
// label 写入 ID 和 Label，后缀是 .md、.json 或 .txt 且文件不超过 8000 字节时用来决定是否读预览。
// path 传给 os.Stat，符合上面条件时再 ReadFile。
func fileArtifact(kind, label, path string) (TraceArtifact, bool) {
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return TraceArtifact{}, false
	}
	art := TraceArtifact{Kind: kind, ID: label, Label: label, Bytes: int(fi.Size())}
	if fi.Size() > 0 && fi.Size() <= 8_000 && (strings.HasSuffix(label, ".md") || strings.HasSuffix(label, ".json") || strings.HasSuffix(label, ".txt")) {
		if b, err := os.ReadFile(path); err == nil {
			art.Preview = capPreview(string(b))
		}
	}
	return art, true
}

// capPreview 把预览截到固定的 rune 数。
// s 的 rune 数超过 4000 时截断并加上省略号。
func capPreview(s string) string {
	// n 是预览最多保留的 rune 数。
	const n = 4_000
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}

// shortHash 返回哈希的前 12 个字符。更短则原样返回。
func shortHash(h string) string {
	if len(h) <= 12 {
		return h
	}
	return h[:12]
}

// firstLabel 返回第一个非空字符串。都空则返回空串。
// ss 按顺序取第一条去掉首尾空白后非空的字符串。
func firstLabel(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
