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
	overlayProbeTimeout = 2 * time.Minute
	overlayTrajCap      = 50_000
	overlayDeliverCap   = 4_096
)

var (
	overlayNamedFile = regexp.MustCompile(`(?i)(?:file named|write|create)\s+([A-Za-z0-9_./-]+\.[A-Za-z0-9]+)`)
	overlayBareFile  = regexp.MustCompile(`(?i)\b([A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)*\.(txt|md|json|csv|go|py|ts|html))\b`)
)

// OverlayRun is the operator-facing session PPC overlay. It never writes
// refs/active, canary, or staging. An adopted successor pins this session only.
type OverlayRun struct {
	SessionID string
	Task      string
	ReuseTurn bool
	K         int
	Mode      string
	MaxUSD    float64
	MaxWall   time.Duration
	// Client runs isolated probes (solver). Nil uses ClientFor(session).
	Client runtime.Client
	// Improver proposes INV/DOM and composes. Nil uses the heuristic proposer
	// so the probe solver is never asked for JSON.
	Improver runtime.Client
}

// OverlaySession copies the session tool root for each probe, rolls the
// width-k PPC overlay (R=4k, P=3k), and pins the current session if a
// successor is adopted.
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

func capOverlayRunes(s string, n int) string {
	if n <= 0 || utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// OverlayLibrary is the read-only per-domain experience INDEX for this
// session's workspace. It never writes refs.
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

// OverlayFlagConflict rejects suite-evolve flags on the session overlay path.
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
