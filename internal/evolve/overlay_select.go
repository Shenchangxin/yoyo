package evolve

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/runtime"
)

const (
	OverlayModeAgg = "agg"
	OverlayModeSeq = "seq"
)

// OverlaySelection is outside the R/P ledger. The pool is the k finals only.
// A composer that kept the base still occupies a slot, so choosing that
// unmodified final is how "pick base" happens — there is no extra abstain seat.
type OverlaySelection struct {
	Mode        string            `json:"mode,omitempty"`
	K           int               `json:"k,omitempty"`
	ChosenAnon  string            `json:"chosen_anon,omitempty"`
	ChosenLabel string            `json:"chosen_label,omitempty"`
	ChosenBase  bool              `json:"chosen_base,omitempty"`
	Why         string            `json:"why,omitempty"`
	Trivial     bool              `json:"trivial,omitempty"`
	Dropped     []string          `json:"dropped,omitempty"`
	IDMap       map[string]string `json:"-"`
}

type overlaySelectItem struct {
	final overlayFinal
	roll  OverlayRollout
}

func ClampOverlayMode(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case OverlayModeSeq, "sequential", "depth":
		return OverlayModeSeq
	default:
		return OverlayModeAgg
	}
}

func overlaySessLabel(k int, kind string, i int) string {
	if k <= 1 {
		return kind
	}
	return fmt.Sprintf("%s%d", kind, i)
}

func overlayNoteFall(res *OverlayResult, k int, kind string, i int, fell bool) {
	if !fell {
		return
	}
	if k == 1 {
		res.FellBack = append(res.FellBack, kind)
		return
	}
	res.FellBack = append(res.FellBack, fmt.Sprintf("%s%d", kind, i))
}

func anonID(index int) string {
	n := index + 1
	out := ""
	for n > 0 {
		n--
		out = string(rune('A'+n%26)) + out
		n /= 26
	}
	return out
}

func assignAnonIDs(labels []string, taskID string, k int, mode string) map[string]string {
	ordered := append([]string{}, labels...)
	sort.Strings(ordered)
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%s", taskID, k, mode)))
	seed := int64(binary.BigEndian.Uint64(sum[:8]))
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(ordered), func(i, j int) { ordered[i], ordered[j] = ordered[j], ordered[i] })
	out := map[string]string{}
	for i, lbl := range ordered {
		out[anonID(i)] = lbl
	}
	return out
}

func overlayPickFinal(ctx context.Context, opts OverlayOpts, parent MaterialSet, base OverlayRollout, finals []overlayFinal, rolls []OverlayRollout, held map[string]bool) OverlaySelection {
	mode := ClampOverlayMode(opts.Mode)
	k := ClampOverlayK(opts.K)
	sel := OverlaySelection{Mode: mode, K: k}
	var pool []overlaySelectItem
	for i, f := range finals {
		r := OverlayRollout{}
		if i < len(rolls) {
			r = rolls[i]
		}
		if !r.Delivered {
			sel.Dropped = append(sel.Dropped, f.label)
			continue
		}
		pool = append(pool, overlaySelectItem{final: f, roll: r})
	}
	if len(pool) == 0 {
		sel.ChosenBase = true
		sel.Why = "no candidate produced an artifact"
		return sel
	}
	labels := make([]string, len(pool))
	byLabel := map[string]overlaySelectItem{}
	for i, it := range pool {
		labels[i] = it.final.label
		byLabel[it.final.label] = it
	}
	taskID := opts.TaskID
	if taskID == "" {
		taskID = opts.SessionID
	}
	sel.IDMap = assignAnonIDs(labels, taskID, k, mode)
	if opts.Tree != nil {
		for anon, lbl := range sel.IDMap {
			it := byLabel[lbl]
			opts.Tree.writeSelection(anon, it.roll.Deliverable)
		}
	}

	if len(pool) == 1 {
		sel.Trivial = true
		sel.ChosenAnon = "A"
		sel.ChosenLabel = pool[0].final.label
		sel.Why = "trivial pool: single delivered final; no selector session"
	} else {
		anon, why, leaked := "", "", false
		if opts.Client != nil {
			anon, why, leaked = selectOverlayLLM(ctx, opts, sel.IDMap, byLabel, held)
		}
		if leaked || anon == "" || sel.IDMap[anon] == "" {
			anon, why = heuristicAnonPick(sel.IDMap, byLabel, base)
		}
		sel.ChosenAnon = anon
		sel.ChosenLabel = sel.IDMap[anon]
		sel.Why = why
	}

	it, ok := byLabel[sel.ChosenLabel]
	if !ok {
		sel.ChosenBase = true
		sel.ChosenLabel = ""
		return sel
	}
	if !it.final.adopted || it.final.succ.Hash == "" || it.final.succ.Hash == parent.Hash {
		sel.ChosenBase = true
	}
	sel.Why = stripHeld(sel.Why, held)
	return sel
}

func heuristicAnonPick(idMap map[string]string, byLabel map[string]overlaySelectItem, base OverlayRollout) (string, string) {
	anons := make([]string, 0, len(idMap))
	for a := range idMap {
		anons = append(anons, a)
	}
	sort.Strings(anons)
	baseBody := strings.TrimSpace(base.Deliverable)
	for _, a := range anons {
		it := byLabel[idMap[a]]
		body := strings.TrimSpace(it.roll.Deliverable)
		if body != "" && body != baseBody {
			return a, "anonymous deliverable diverged from the shared base"
		}
	}
	if len(anons) == 0 {
		return "", "empty pool"
	}
	return anons[0], "no deliverable contrast; first anonymous id (base is valid)"
}

func selectOverlayLLM(ctx context.Context, opts OverlayOpts, idMap map[string]string, byLabel map[string]overlaySelectItem, held map[string]bool) (anon, why string, leaked bool) {
	if opts.Client == nil {
		return "", "", false
	}
	job := "Pick ONE anonymized candidate in context/selection/. Harnesses are hidden. Picking an unmodified output is valid. Reply with <why> and a ```choice block."
	var text string
	var err error
	if opts.Tree != nil {
		text, err = overlayImproverChat(ctx, opts, overlaySelectSystem, opts.Tree.mission(job))
	} else {
		anons := make([]string, 0, len(idMap))
		for a := range idMap {
			anons = append(anons, a)
		}
		sort.Strings(anons)
		var body strings.Builder
		body.WriteString("Candidates (harnesses hidden):\n")
		for _, a := range anons {
			it := byLabel[idMap[a]]
			fmt.Fprintf(&body, "\n## %s\ndelivered=%v\n%s\n", a, it.roll.Delivered, capSelect(it.roll.Deliverable, 4000))
		}
		var msg runtime.Message
		msg, err = opts.Client.Chat(ctx, runtime.ChatRequest{
			Model: opts.Model,
			Messages: []runtime.Message{
				{Role: runtime.RoleSystem, Content: overlaySelectSystem},
				{Role: runtime.RoleUser, Content: stripHeld(body.String(), held)},
			},
		})
		text = msg.Content
	}
	if err != nil {
		return "", "", false
	}
	labels := make([]string, 0, len(idMap))
	for _, lbl := range idMap {
		labels = append(labels, lbl)
	}
	if selectionLeaked(text, opts.Parent.Hash, labels) {
		return "", "", true
	}
	anon, why, ok := parseOverlayChoice(text)
	if !ok || idMap[anon] == "" {
		return "", "", false
	}
	return anon, why, false
}

func parseOverlayChoice(content string) (anon, why string, ok bool) {
	if i := strings.Index(content, "<why>"); i >= 0 {
		rest := content[i+5:]
		if j := strings.Index(rest, "</why>"); j >= 0 {
			why = strings.TrimSpace(rest[:j])
		}
	}
	if i := strings.Index(content, "```choice"); i >= 0 {
		rest := content[i+len("```choice"):]
		rest = strings.TrimLeft(rest, "\n\r\t ")
		if j := strings.Index(rest, "```"); j >= 0 {
			anon = strings.TrimSpace(rest[:j])
		} else {
			anon = strings.TrimSpace(rest)
		}
		if n := strings.IndexAny(anon, "\n\r"); n >= 0 {
			anon = strings.TrimSpace(anon[:n])
		}
	}
	anon = strings.ToUpper(anon)
	return anon, why, anon != ""
}

func selectionLeaked(text, parentHash string, labels []string) bool {
	lower := strings.ToLower(text)
	for _, w := range []string{
		"adopt_inv", "adopt_dom", "playbook", "refs/active", "held-out",
		"pass_rate", "shouldpromote", "inv1", "dom1", "instruction slot",
	} {
		if strings.Contains(lower, w) {
			return true
		}
	}
	if parentHash != "" && strings.Contains(text, parentHash) {
		return true
	}
	for _, l := range labels {
		if strings.HasPrefix(l, "inv") || strings.HasPrefix(l, "dom") {
			if l != "" && strings.Contains(lower, strings.ToLower(l)) {
				return true
			}
		}
	}
	return false
}

func capSelect(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}
