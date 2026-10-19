package browser

import (
	"regexp"
	"strings"
	"sync"
)

var refName = regexp.MustCompile(`^[a-z]{2,3}_\d+$`)

// refCache holds the last successful action walk. Refs themselves are
// resolved live in the page; the cache is the descriptor set used when a
// handle goes stale after a re-render and we retry by testid / name / id.
type refCache struct {
	mu    sync.Mutex
	list  []Action
	byRef map[string]Action
}

func (c *refCache) store(acts []Action) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.list = append([]Action(nil), acts...)
	c.byRef = make(map[string]Action, len(acts))
	for _, a := range acts {
		if a.Ref != "" {
			c.byRef[a.Ref] = a
		}
	}
}

func (c *refCache) lookup(ref string) (Action, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	a, ok := c.byRef[ref]
	return a, ok
}

func (c *refCache) all() []Action {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Action(nil), c.list...)
}

func (c *refCache) clear() {
	c.mu.Lock()
	c.list = nil
	c.byRef = nil
	c.mu.Unlock()
}

func (h *Host) invalidateRefs() {
	if h == nil {
		return
	}
	h.refs.clear()
}

// parseRef extracts btn_0 from ref=btn_0, ref:btn_0, [ref=btn_0], or a bare
// btn_0. Empty means the string is a CSS selector or text= handle.
func parseRef(sel string) string {
	s := strings.TrimSpace(sel)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "ref=") || strings.HasPrefix(low, "ref:") {
		return strings.TrimSpace(s[4:])
	}
	if refName.MatchString(s) {
		return s
	}
	return ""
}

func (h *Host) rememberedTarget(sel string) (Action, bool) {
	if h == nil {
		return Action{}, false
	}
	if ref := parseRef(sel); ref != "" {
		if a, ok := h.refs.lookup(ref); ok {
			return a, true
		}
		return Action{Ref: ref, Text: ref}, true
	}
	if strings.HasPrefix(sel, "text=") {
		name := strings.TrimSpace(sel[5:])
		return Action{Text: name, Name: name}, true
	}
	for _, a := range h.refs.all() {
		if a.Selector == sel {
			return a, true
		}
		if a.ID != "" && sel == "#"+a.ID {
			return a, true
		}
		if a.TestID != "" && (sel == `[data-testid="`+a.TestID+`"]` || sel == `[data-testid='`+a.TestID+`']`) {
			return a, true
		}
	}
	return Action{}, false
}

func (h *Host) matchFallback(sel string) (string, string) {
	want, ok := h.rememberedTarget(sel)
	if !ok {
		return "", ""
	}
	live, err := h.actionsList()
	if err != nil || len(live) == 0 {
		return "", ""
	}
	got, why, score := matchTarget(want, live)
	if score < matchThreshold || (got.Ref == "" && got.Selector == "") {
		return "", why
	}
	if got.Ref != "" {
		return "ref=" + got.Ref, why
	}
	return got.Selector, why
}

const matchThreshold = 45

// matchTarget scores live actions against a remembered descriptor.
// Weights follow the octop-browser matcher: selector/testid, id, name attr,
// accessible name, role. A score below 45 is a miss.
func matchTarget(want Action, have []Action) (Action, string, int) {
	wantSel := selectorSet(want)
	wantName := normMatch(firstNonEmpty(want.Name, want.Text, want.Placeholder))
	wantRole := normMatch(want.Role)
	wantID := normMatch(want.ID)
	wantNameAttr := normMatch(want.NameAttr)
	wantTest := normMatch(want.TestID)

	bestScore := 0
	bestWhy := ""
	var best Action
	for _, el := range have {
		score := 0
		var why []string
		elSel := selectorSet(el)
		if len(wantSel) > 0 && overlap(wantSel, elSel) {
			score += 80
			why = append(why, "selector")
		}
		if wantTest != "" && wantTest == normMatch(el.TestID) {
			score += 80
			why = append(why, "testid")
		}
		if wantID != "" && wantID == normMatch(el.ID) {
			score += 70
			why = append(why, "id")
		}
		if wantNameAttr != "" && wantNameAttr == normMatch(el.NameAttr) {
			score += 60
			why = append(why, "nameAttr")
		}
		if wantRole != "" && wantRole == normMatch(el.Role) {
			score += 25
			why = append(why, "role")
		}
		elName := normMatch(firstNonEmpty(el.Name, el.Text, el.Placeholder))
		if wantName != "" && elName == wantName {
			score += 50
			why = append(why, "name")
		} else if wantName != "" && elName != "" && strings.Contains(elName, wantName) {
			score += 20
			why = append(why, "partial-name")
		}
		if score > bestScore {
			bestScore = score
			bestWhy = strings.Join(why, "+")
			best = el
		}
	}
	if bestScore < matchThreshold {
		return Action{}, "not-found", bestScore
	}
	return best, bestWhy, bestScore
}

func selectorSet(a Action) map[string]struct{} {
	out := map[string]struct{}{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" {
			out[s] = struct{}{}
		}
	}
	add(a.Selector)
	if a.TestID != "" {
		add(`[data-testid="` + a.TestID + `"]`)
	}
	if a.ID != "" {
		add("#" + a.ID)
	}
	return out
}

func overlap(a, b map[string]struct{}) bool {
	for k := range a {
		if _, ok := b[k]; ok {
			return true
		}
	}
	return false
}

func normMatch(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
