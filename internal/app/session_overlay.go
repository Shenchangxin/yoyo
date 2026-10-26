// 会话上的 overlay。探针使用临时工作目录，采纳结果只钉住当前会话。

package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/evolve"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

const (
	// overlayProbeTimeout 是单次探针的默认时限。更短的 MaxWall 会把它压低。
	overlayProbeTimeout = 2 * time.Minute
	// overlayTrajCap 是探针轨迹文本的 rune 上限。
	overlayTrajCap = 50_000
	// overlayDeliverCap 是交付正文摘录的 rune 上限。
	overlayDeliverCap = 4_096
)

var (
	// overlayNamedFile 从任务原文里抽出带动词的文件名。
	overlayNamedFile = regexp.MustCompile(`(?i)(?:file named|write|create)\s+([A-Za-z0-9_./-]+\.[A-Za-z0-9]+)`)
	// overlayBareFile 从任务原文里抽出常见后缀的路径。
	overlayBareFile = regexp.MustCompile(`(?i)\b([A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)*\.(txt|md|json|csv|go|py|ts|html))\b`)
)

// OverlayRun 是会话 overlay 的参数。它不写 refs/active、canary 或 staging。采纳的后继只钉住当前会话。
type OverlayRun struct {
	SessionID string
	Task      string
	ReuseTurn bool
	K         int
	Mode      string
	MaxUSD    float64
	MaxWall   time.Duration
	// Client 在临时工作目录上跑探针。为空时用 ClientFor 取这个会话的客户端。
	Client runtime.Client
	// Improver 提出 INV/DOM 并合成。为空时走启发式提议，不向探针求解器要 JSON。
	Improver runtime.Client
}

// OverlaySession 对会话工具根做探针，跑给定宽度的 overlay。采纳后继时只把当前会话的 harness 钉住，不改 active、canary 和 staging。视频会话直接拒绝。
// ctx 为 nil 时改用 context.Background，再传给 acquireRun 和 Evolve.Overlay。
// run 用来读 SessionID、Task、ReuseTurn、K、Mode、MaxUSD、MaxWall、Client 和 Improver；SessionID 去掉空白后为空则返回错误，Client 为 nil 时改用 ClientFor，MaxUSD 小于等于 0 且配置上限大于 0 时改用配置上限。
func (a *App) OverlaySession(ctx context.Context, run OverlayRun) (evolve.OverlayResult, error) {
	var res evolve.OverlayResult
	if a == nil || a.Evolve == nil {
		return res, fmt.Errorf("overlay: evolve engine required")
	}
	if a.IsPaused() {
		return res, fmt.Errorf("paused")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID := strings.TrimSpace(run.SessionID)
	if sessionID == "" {
		return res, fmt.Errorf("overlay: session required")
	}
	meta, err := a.GetSession(sessionID)
	if err != nil {
		return res, fmt.Errorf("unknown session %s: %w", sessionID, err)
	}
	if session.NormalizeChannel(meta.Channel) == session.ChannelVideo {
		return res, fmt.Errorf("overlay: video sessions are not supported")
	}
	if meta.HarnessPolicy == "" {
		meta.HarnessPolicy = session.FollowActive
	}
	parentHash := session.ResolveHarness(session.NormalizePolicy(meta.HarnessPolicy), meta.Harness, a.ActiveHash())
	if parentHash == "" {
		return res, fmt.Errorf("overlay: session %s has no harness", sessionID)
	}
	parent, err := a.materialSet(parentHash)
	if err != nil {
		return res, err
	}

	task := runtime.NormalizeUserText(run.Task)
	var reuse *evolve.OverlayRollout
	if lastText, lastHash, lastTurn, ok := overlayLastTurn(a, sessionID); ok {
		if task == "" {
			task = lastText
		}
		if run.ReuseTurn && lastHash == parentHash && (run.Task == "" || lastText == runtime.NormalizeUserText(run.Task)) {
			got, body := overlayDelivered(meta.ToolRoot(), overlayWantedFiles(lastText))
			reuse = &evolve.OverlayRollout{
				Label:       "base",
				Delivered:   got,
				Deliverable: body,
				Trajectory:  overlayTrajectory(lastTurn),
				Cached:      true,
			}
		}
	}
	if task == "" {
		return res, fmt.Errorf("overlay: no task")
	}

	solver := run.Client
	improver := run.Improver
	if solver == nil {
		solver, err = a.ClientFor(meta)
		if err != nil {
			return res, err
		}
		if improver == nil {
			improver = solver
		}
	}
	maxUSD := run.MaxUSD
	if maxUSD <= 0 && a.Config.MaxBudgetUSD > 0 {
		maxUSD = a.Config.MaxBudgetUSD
	}
	model := a.Config.Model
	if meta.Model != "" {
		model = meta.Model
	}
	heldOut, heldInText := a.overlayHeld(parentHash)
	k := evolve.ClampOverlayK(run.K)
	var lib *evolve.Library
	var libText string
	domain := evolve.DomainSlug(meta.ProjectRoot())
	if a.Home != nil {
		lib = &evolve.Library{Root: a.Home.Library(), Domain: domain}
		libText = lib.Context()
	}

	ctx, release, err := a.acquireRun(sessionID, ctx)
	if err != nil {
		return res, err
	}
	defer release()

	toolRoot := meta.ToolRoot()
	roller := func(ctx context.Context, label string, mats evolve.MaterialSet) (evolve.OverlayRollout, error) {
		return a.overlayProbe(ctx, overlayProbeReq{
			label:    label,
			session:  sessionID,
			task:     task,
			toolRoot: toolRoot,
			model:    model,
			client:   solver,
			mats:     mats,
			maxWall:  run.MaxWall,
		})
	}

	res, err = a.Evolve.Overlay(ctx, evolve.OverlayOpts{
		Parent:          parent,
		Task:            task,
		Client:          improver,
		Model:           model,
		SessionID:       sessionID,
		TaskID:          sessionID,
		Domain:          domain,
		K:               k,
		Mode:            run.Mode,
		ReuseBase:       reuse,
		Roller:          roller,
		HeldOut:         heldOut,
		HeldInText:      heldInText,
		MaxUSD:          maxUSD,
		MaxWall:         run.MaxWall,
		Library:         lib,
		LibraryText:     libText,
		ParallelPropose: run.Client == nil && improver != nil,
	})
	if err != nil {
		return res, err
	}
	a.addSpendUsage(res.Spend)

	if res.Adopted && res.Hash != "" && res.Hash != parentHash {
		meta.Harness = res.Hash
		meta.HarnessPolicy = session.Pin
		meta.UpdatedAt = time.Now().UTC()
		if werr := a.writeSession(meta); werr != nil {
			return res, werr
		}
	}
	return res, nil
}

// overlayProbeReq 是一次探针要的标签、任务、工具根、模型和材料。
type overlayProbeReq struct {
	label    string
	session  string
	task     string
	toolRoot string
	model    string
	client   runtime.Client
	mats     evolve.MaterialSet
	maxWall  time.Duration
}

// overlayProbe 用 runtime.IsolateWorkspace 得到一份临时工作目录，在其中调用 runtime.Run，再按任务里点名的文件判断是否交付。
// ctx 用来做超时上下文，再传给 runtime.Run。
// req 用来读 label、toolRoot、maxWall、session、mats、model、task 和 client；toolRoot 去掉空白后为空则返回错误。
func (a *App) overlayProbe(ctx context.Context, req overlayProbeReq) (evolve.OverlayRollout, error) {
	out := evolve.OverlayRollout{Label: req.label}
	src := strings.TrimSpace(req.toolRoot)
	if src == "" {
		return out, fmt.Errorf("overlay: empty tool root")
	}
	work, cleanup, err := runtime.IsolateWorkspace(src)
	if err != nil {
		return out, err
	}
	defer cleanup()

	timeout := overlayProbeTimeout
	if req.maxWall > 0 && req.maxWall < timeout {
		timeout = req.maxWall
	}
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	caps := capability.NewBroker(capability.AutoPolicy{
		Allow: []capability.Level{capability.ReadWorkspace, capability.WriteWorkspace, capability.Shell},
	}, func(context.Context, capability.Request) (capability.Decision, error) {
		return capability.Always, nil
	})
	skillBodies, _, skillMeta := runtime.MapSkills(req.mats.Skills)
	advertised := append([]string(nil), req.mats.Snap.Tools...)
	if len(req.mats.Snap.EvalTools) > 0 {
		advertised = append([]string(nil), req.mats.Snap.EvalTools...)
	}
	probeID := fmt.Sprintf("overlay-%s-%s-%s", req.session, req.label, newID()[:8])
	tools := &runtime.WorkspaceTools{
		Workspace:  work,
		Home:       work,
		SessionID:  probeID,
		Caps:       caps,
		Skills:     skillBodies,
		SkillMeta:  skillMeta,
		Advertised: advertised,
	}
	loop := req.mats.Loop
	loop.AllowLLMCompact = false
	meter := &runtime.Meter{USDPerMTok: a.Config.USDPerMTok}
	_, runErr := runtime.Run(tctx, runtime.RunRequest{
		SessionID:        probeID,
		TaskID:           req.label,
		User:             req.task,
		Workspace:        work,
		Harness:          req.mats.Snap,
		HarnessHash:      req.mats.Hash,
		ModelFingerprint: Fingerprint(a.Config),
		Model:            req.model,
		Loop:             loop,
		Fragments:        req.mats.Fragments,
		Playbook:         req.mats.Playbook,
		Skills:           req.mats.Skills,
		Tools:            tools,
		Client:           req.client,
		Trace:            a.Traces,
		Meter:            meter,
	})
	_, _, usd := meter.Totals()
	out.USD = usd
	if runErr != nil {
		out.Error = runErr.Error()
	}
	if a.Traces != nil {
		if evs, rerr := a.Traces.Read(probeID); rerr == nil {
			out.Trajectory = overlayTrajectory(evs)
		}
	}
	out.Delivered, out.Deliverable = overlayDelivered(work, overlayWantedFiles(req.task))
	return out, nil
}

// overlayHeld 从套件取出 held-out 和 transfer 的题号，并把 held-in 的题目正文读出来。
// hash 传给 Materials。
func (a *App) overlayHeld(hash string) (heldOut, heldInText []string) {
	_, _, _, _, suite, _, err := a.Materials(hash)
	if err != nil {
		return nil, nil
	}
	heldOut = append(append([]string{}, suite.HeldOut...), suite.Transfer...)
	if a.Eval == nil {
		return heldOut, nil
	}
	for _, id := range suite.HeldIn {
		if s := a.Eval.ReadInstruction(suite, id); s != "" {
			heldInText = append(heldInText, s)
		}
	}
	return heldOut, heldInText
}

// overlayLastTurn 取最后一条非 steer 的用户消息，以及它到下一条用户消息之间的事件。
// sessionID 传给 Traces.Read。
func overlayLastTurn(a *App, sessionID string) (text, harness string, turn []trace.Event, ok bool) {
	if a == nil || a.Traces == nil {
		return "", "", nil, false
	}
	evs, err := a.Traces.Read(sessionID)
	if err != nil || len(evs) == 0 {
		return "", "", nil, false
	}
	last := -1
	for i, ev := range evs {
		if ev.Type == trace.TypeUser && !trace.IsSteerUser(ev) {
			last = i
		}
	}
	if last < 0 {
		return "", "", nil, false
	}
	text, _ = evs[last].Payload["text"].(string)
	text = runtime.NormalizeUserText(text)
	if text == "" {
		return "", "", nil, false
	}
	harness = evs[last].HarnessSnapshot
	end := len(evs)
	for i := last + 1; i < len(evs); i++ {
		if evs[i].Type == trace.TypeUser && !trace.IsSteerUser(evs[i]) {
			end = i
			break
		}
	}
	turn = evs[last:end]
	for _, ev := range turn {
		if ev.HarnessSnapshot != "" {
			harness = ev.HarnessSnapshot
		}
	}
	return text, harness, turn, true
}

// overlayWantedFiles 从任务原文抽出要交付的相对路径，丢掉空路径和带上级目录的路径。
// task 传给 overlayNamedFile 和 overlayBareFile 做匹配。
func overlayWantedFiles(task string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
		name = strings.TrimLeft(name, "./")
		if name == "" || strings.Contains(name, "..") || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	for _, m := range overlayNamedFile.FindAllStringSubmatch(task, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range overlayBareFile.FindAllStringSubmatch(task, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	return out
}

// overlayDelivered 在点名的文件都存在且非空时返回真，并带上第一个文件的摘录。
// root 去掉首尾空白后为空则返回 false，否则用来拼文件路径。
// wanted 长度为 0 时返回 false，否则逐项读文件。
func overlayDelivered(root string, wanted []string) (bool, string) {
	if strings.TrimSpace(root) == "" || len(wanted) == 0 {
		return false, ""
	}
	var first string
	for i, rel := range wanted {
		p := filepath.Join(root, filepath.FromSlash(rel))
		b, err := os.ReadFile(p)
		if err != nil || len(bytes.TrimSpace(b)) == 0 {
			return false, first
		}
		if i == 0 {
			first = capOverlayRunes(string(b), overlayDeliverCap)
		}
	}
	return true, first
}

// overlayTrajectory 把工具调用、结果、错误和助手正文收成一段轨迹文本。
// evs 按工具调用、结果、错误和助手正文收成一段文本。
func overlayTrajectory(evs []trace.Event) string {
	var b strings.Builder
	for _, ev := range evs {
		switch ev.Type {
		case trace.TypeToolCall:
			name, _ := ev.Payload["name"].(string)
			args, _ := ev.Payload["arguments"].(string)
			fmt.Fprintf(&b, "[ACTION] %s %s\n", name, capOverlayRunes(args, 500))
		case trace.TypeToolResult:
			name, _ := ev.Payload["name"].(string)
			content, _ := ev.Payload["content"].(string)
			kind := "OBSERVATION"
			if strings.Contains(strings.ToLower(content), "error") {
				kind = "ERROR"
			}
			fmt.Fprintf(&b, "[%s] %s %s\n", kind, name, capOverlayRunes(content, evolve.OverlayObsFloor))
		case trace.TypeError:
			text, _ := ev.Payload["text"].(string)
			if text == "" {
				text, _ = ev.Payload["error"].(string)
			}
			fmt.Fprintf(&b, "[ERROR] %s\n", capOverlayRunes(text, 2_000))
		case trace.TypeAssistant:
			text, _ := ev.Payload["text"].(string)
			if strings.TrimSpace(text) != "" {
				fmt.Fprintf(&b, "[ACTION] assistant %s\n", capOverlayRunes(text, 500))
			}
		}
	}
	return capOverlayRunes(b.String(), overlayTrajCap)
}

// capOverlayRunes 按 rune 截断字符串。上限小于等于 0 或未超出时原样返回。
func capOverlayRunes(s string, n int) string {
	if n <= 0 || utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// OverlayLibrary 只读该工作区领域的经验索引，不写引用。视频会话直接拒绝。
// sessionID 先 TrimSpace，空串则返回错误，否则传给 GetSession。
func (a *App) OverlayLibrary(sessionID string) (evolve.LibraryView, error) {
	var out evolve.LibraryView
	if a == nil || a.Home == nil {
		return out, fmt.Errorf("overlay: home required")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return out, fmt.Errorf("overlay: session required")
	}
	meta, err := a.GetSession(sessionID)
	if err != nil {
		return out, fmt.Errorf("unknown session %s: %w", sessionID, err)
	}
	if session.NormalizeChannel(meta.Channel) == session.ChannelVideo {
		return out, fmt.Errorf("overlay: video sessions are not supported")
	}
	heldOut, _ := a.overlayHeld(session.ResolveHarness(session.NormalizePolicy(meta.HarnessPolicy), meta.Harness, a.ActiveHash()))
	held := map[string]bool{}
	for _, id := range heldOut {
		if id != "" {
			held[id] = true
		}
	}
	lib := &evolve.Library{Root: a.Home.Library(), Domain: evolve.DomainSlug(meta.ProjectRoot())}
	return lib.View(held), nil
}

// OverlayFlagConflict 在会话 overlay 路径上拒绝再混入套件演进的旗标。没有旗标则返回 nil。
// names 逐项去掉首尾空白，空串丢掉；一个都不剩则返回 nil。
func OverlayFlagConflict(names ...string) error {
	var out []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return fmt.Errorf("overlay cannot mix with %s", strings.Join(out, ", "))
}
