package browser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Interaction model for the isolated browser.
//
// Click and type act the way a person does: the target is resolved in the
// live document, must be visible and enabled, and must be the topmost element
// at its centre (a real pointer press reaches nothing else). Key presses are
// real key events, so keyboard-driven games see them. Every failure names the
// actions that are actually on the page, so the agent corrects its selector
// from ground truth instead of guessing.

const (
	// actionWait is the auto-wait window the browser_click / browser_type
	// specs promise. The old implementation waited 3 s but never got a page
	// to answer, so it always timed out.
	actionWait = 4 * time.Second
	actionPoll = 120 * time.Millisecond
	// maxActions bounds the action list shown to the model.
	maxActions = 48
	// maxActionCollect is how many interactive nodes the page walk keeps
	// before formatting. Structured snapshots can use the extra ones.
	maxActionCollect = 60
)

// Action is an element the page offers for interaction. Ref is a short
// handle (btn_0) assigned by the live walk; it is valid until navigation.
type Action struct {
	Ref         string   `json:"ref,omitempty"`
	Selector    string   `json:"selector"`
	Tag         string   `json:"tag"`
	Text        string   `json:"text"`
	Disabled    bool     `json:"disabled,omitempty"`
	Role        string   `json:"role,omitempty"`
	TestID      string   `json:"testId,omitempty"`
	Name        string   `json:"name,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	ID          string   `json:"id,omitempty"`
	NameAttr    string   `json:"nameAttr,omitempty"`
	Href        string   `json:"href,omitempty"`
	Type        string   `json:"type,omitempty"`
	FramePath   []string `json:"framePath,omitempty"`
	ShadowPath  []string `json:"shadowPath,omitempty"`
}

// probe is the JSON result of one resolve-and-check evaluation.
type probe struct {
	OK     bool    `json:"ok"`
	Reason string  `json:"reason"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Tag    string  `json:"tag"`
	Text   string  `json:"text"`
	By     string  `json:"by"`
	Ref    string  `json:"ref"`
	Error  string  `json:"error"`
}

// probeJS resolves sel, checks it, and either focuses it (focusOnly) or
// returns the centre point to press. It never clicks by itself.
func probeJS(sel string, focusOnly bool) string {
	quoted, _ := json.Marshal(sel)
	return fmt.Sprintf(`(() => {
%s
  const sel = %s;
  let rec;
  try { rec = __yoyoResolveRec(sel); } catch (e) { return JSON.stringify({ok:false, reason:"bad-selector", error:String(e)}); }
  if (!rec || !rec.el) return JSON.stringify({ok:false, reason:"missing"});
  const el = rec.el;
  if (!__yoyoVisible(el)) return JSON.stringify({ok:false, reason:"hidden", text:__yoyoText(el), ref:rec.ref});
  if (el.disabled) return JSON.stringify({ok:false, reason:"disabled", text:__yoyoText(el), ref:rec.ref});
  el.scrollIntoView({block:"center", inline:"center"});
  if (%t) {
    el.focus();
    return JSON.stringify({ok:true, tag:el.tagName.toLowerCase(), text:__yoyoText(el), ref:rec.ref});
  }
  const hit = __yoyoHitOK(rec);
  if (!hit.ok) {
    return JSON.stringify({ok:false, reason:"covered", by:hit.by, ref:rec.ref, text:__yoyoText(el)});
  }
  return JSON.stringify({ok:true, x:hit.x, y:hit.y, tag:el.tagName.toLowerCase(), text:__yoyoText(el), ref:rec.ref});
})()`, pageHelpers, quoted, focusOnly)
}

// actionsJS lists the visible, addressable interactive elements with refs.
const actionsJS = `(() => {
` + pageHelpers + `
  const out = [];
  for (const c of __yoyoCandidates()) {
    const a = __yoyoActionOf(c);
    if (!a.selector && !a.ref) continue;
    out.push(a);
    if (out.length >= 60) break;
  }
  return JSON.stringify(out);
})()`

// Click resolves sel, waits for it to become actionable, presses it with real
// pointer events, and reports whether the visible page text changed.
func (h *Host) Click(sel string) (string, error) {
	h.record("click", sel)
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return "", fmt.Errorf("browser: click %s: %w", sel, err)
	}
	before := h.pageText()
	p, err := h.clickCDP(sel)
	if err != nil {
		h.setLastAction("")
		return "", err
	}
	time.Sleep(150 * time.Millisecond)
	h.syncView()
	h.refreshActions()
	note := "page text changed"
	if h.pageText() == before {
		note = "no visible text change"
	}
	handle := sel
	if p.Ref != "" {
		handle = "ref=" + p.Ref
	}
	obs := fmt.Sprintf("clicked %s → <%s> %q at (%d,%d); %s", handle, p.Tag, truncRunes(p.Text, 40), int(p.X), int(p.Y), note)
	h.setLastAction(obs)
	return obs + h.pageErrorsNote() + h.actionsNote(), nil
}

// Type focuses sel (when given) and sends each character as a real key
// press, so keydown handlers on document or window receive it.
func (h *Host) Type(sel, text string) (string, error) {
	h.record("type", sel)
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return "", fmt.Errorf("browser: type into %s: %w", sel, err)
	}
	before := h.pageText()
	if err := h.typeCDP(sel, text); err != nil {
		return "", err
	}
	time.Sleep(150 * time.Millisecond)
	h.syncView()
	h.refreshActions()
	note := "page text changed"
	if h.pageText() == before {
		note = "no visible text change"
	}
	target := sel
	if strings.TrimSpace(target) == "" {
		target = "the focused element"
	}
	return fmt.Sprintf("typed %d key(s) into %s; %s%s%s", utf8.RuneCountInString(text), target, note, h.pageErrorsNote(), h.actionsNote()), nil
}

// Fill replaces the current field value. Type appends key events; Fill
// selects all, deletes, then types, so a second fill does not concatenate.
func (h *Host) Fill(sel, text string) (string, error) {
	h.record("fill", sel)
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return "", fmt.Errorf("browser: fill %s: %w", sel, err)
	}
	before := h.pageText()
	if strings.TrimSpace(sel) == "" {
		return "", fmt.Errorf("browser: fill needs a selector or ref")
	}
	if _, err := h.waitFor(sel, true); err != nil {
		return "", err
	}
	if err := h.selectAllClear(); err != nil {
		return "", err
	}
	if err := h.typeCDP("", text); err != nil {
		return "", err
	}
	time.Sleep(150 * time.Millisecond)
	h.syncView()
	h.refreshActions()
	note := "page text changed"
	if h.pageText() == before {
		note = "no visible text change"
	}
	return fmt.Sprintf("filled %d rune(s) into %s; %s%s%s", utf8.RuneCountInString(text), sel, note, h.pageErrorsNote(), h.actionsNote()), nil
}

// WaitVisible waits until sel is actionable. It does not click. browser_act
// uses this for assert steps so a sequence can stop before a covered target.
func (h *Host) WaitVisible(sel string) (string, error) {
	h.record("wait", sel)
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return "", fmt.Errorf("browser: wait %s: %w", sel, err)
	}
	p, err := h.waitFor(sel, false)
	if err != nil {
		return "", err
	}
	h.syncView()
	h.refreshActions()
	return fmt.Sprintf("visible %s → <%s> %q at (%d,%d)%s%s", sel, p.Tag, truncRunes(p.Text, 40), int(p.X), int(p.Y), h.pageErrorsNote(), h.actionsNote()), nil
}

// Refresh re-reads the live page (URL, title, text, actions) without
// navigating. It is what browser_snapshot reports. live is false when Chrome
// is not running, in which case the snapshot is the last one taken.
func (h *Host) Refresh() (snap Snapshot, live bool) {
	live = h.chromeAlive()
	if live {
		h.syncView()
		h.refreshActions()
	}
	snap = h.snapshotWithElements()
	if live {
		snap.Errors = h.takePageErrors()
	}
	return snap, live
}

func (h *Host) clickCDP(sel string) (probe, error) {
	p, err := h.waitFor(sel, false)
	if err != nil {
		return probe{}, err
	}
	for _, typ := range []string{"mouseMoved", "mousePressed", "mouseReleased"} {
		ev := map[string]any{"type": typ, "x": p.X, "y": p.Y}
		if typ != "mouseMoved" {
			ev["button"] = "left"
			ev["clickCount"] = 1
		}
		if _, err := h.callCDP("Input.dispatchMouseEvent", ev); err != nil {
			return probe{}, err
		}
	}
	return p, nil
}

func (h *Host) typeCDP(sel, text string) error {
	if strings.TrimSpace(sel) != "" {
		if _, err := h.waitFor(sel, true); err != nil {
			return err
		}
	}
	for _, r := range text {
		if err := h.keyTap(r); err != nil {
			return err
		}
	}
	return nil
}

// waitFor polls the probe until the target is actionable or the window ends.
// A miss on a stale ref retries once against the last snapshot's descriptors
// (testid / id / accessible name) so a re-render does not force the model
// to guess a new CSS path.
func (h *Host) waitFor(sel string, focusOnly bool) (probe, error) {
	deadline := time.Now().Add(actionWait)
	var last probe
	for {
		p, err := h.probeOn(sel, focusOnly)
		if err != nil {
			return probe{}, err
		}
		last = p
		if p.OK {
			return p, nil
		}
		if p.Reason == "bad-selector" || !time.Now().Before(deadline) {
			break
		}
		time.Sleep(actionPoll)
	}
	if alt, why := h.matchFallback(sel); alt != "" && alt != sel {
		p, err := h.probeOn(alt, focusOnly)
		if err == nil && p.OK {
			h.record("match", why+": "+sel+" → "+alt)
			return p, nil
		}
	}
	return probe{}, h.explainMiss(sel, last)
}

func (h *Host) probeOn(sel string, focusOnly bool) (probe, error) {
	out, err := h.evalJS(probeJS(sel, focusOnly))
	if err != nil {
		return probe{}, err
	}
	var p probe
	if err := json.Unmarshal([]byte(cleanJS(out)), &p); err != nil {
		return probe{}, fmt.Errorf("browser: probe for %s returned no result", sel)
	}
	return p, nil
}

// coveredWhy words a failed hit test. An empty hit means elementFromPoint
// returned null at the element's centre: the centre is outside the viewport or
// clipped by an ancestor. Nothing covers it, so "covered by nothing" misled
// the agent into hunting for an overlay that does not exist.
func coveredWhy(sel, by string) string {
	if by == "" || by == "nothing" {
		return fmt.Sprintf("%s has no hit-test element at its centre (outside the viewport or clipped by an ancestor), so a real click would not reach it", sel)
	}
	return fmt.Sprintf("%s is covered by %s, so a real click would not reach it", sel, by)
}

// explainMiss turns a probe failure into a message that names the actual
// actions on the page, so the next selector is chosen from ground truth.
func (h *Host) explainMiss(sel string, p probe) error {
	var why string
	switch p.Reason {
	case "missing":
		why = fmt.Sprintf("selector %s matched no element", sel)
	case "hidden":
		why = fmt.Sprintf("%s is not visible", sel)
	case "disabled":
		why = fmt.Sprintf("%s is disabled", sel)
	case "covered":
		why = coveredWhy(sel, p.By)
	case "bad-selector":
		why = fmt.Sprintf("%s is not a valid CSS selector (%s); use CSS or text=<visible text>", sel, p.Error)
	default:
		why = fmt.Sprintf("%s: %s", sel, p.Reason)
	}
	if p.Reason == "bad-selector" {
		return fmt.Errorf("browser: %s", why)
	}
	if acts, err := h.actionsList(); err == nil && len(acts) > 0 {
		why += "\nvisible actions now (use one of these selectors):\n" + formatActions(acts)
	} else {
		why += "\nno visible interactive elements on this page"
	}
	return fmt.Errorf("browser: %s", why)
}

// keyTap sends one character as a keyDown/keyUp pair with text, which fires
// keydown, keypress and input the way the keyboard does.
func (h *Host) selectAllClear() error {
	// Ctrl+A and Meta+A: Chrome ignores the modifier that is not native.
	for _, modifiers := range []int{2, 4} {
		down := map[string]any{"type": "keyDown", "key": "a", "code": "KeyA", "windowsVirtualKeyCode": 65, "modifiers": modifiers}
		up := map[string]any{"type": "keyUp", "key": "a", "code": "KeyA", "windowsVirtualKeyCode": 65, "modifiers": modifiers}
		if _, err := h.callCDP("Input.dispatchKeyEvent", down); err != nil {
			return err
		}
		if _, err := h.callCDP("Input.dispatchKeyEvent", up); err != nil {
			return err
		}
	}
	if _, err := h.callCDP("Input.dispatchKeyEvent", map[string]any{"type": "keyDown", "key": "Backspace", "code": "Backspace", "windowsVirtualKeyCode": 8}); err != nil {
		return err
	}
	_, err := h.callCDP("Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": "Backspace", "code": "Backspace", "windowsVirtualKeyCode": 8})
	return err
}

func (h *Host) keyTap(r rune) error {
	k := keyFor(r)
	down := map[string]any{"type": "keyDown", "key": k.key, "code": k.code, "windowsVirtualKeyCode": k.vk}
	if k.text != "" {
		down["text"] = k.text
		down["unmodifiedText"] = k.text
	}
	up := map[string]any{"type": "keyUp", "key": k.key, "code": k.code, "windowsVirtualKeyCode": k.vk}
	if _, err := h.callCDP("Input.dispatchKeyEvent", down); err != nil {
		return err
	}
	_, err := h.callCDP("Input.dispatchKeyEvent", up)
	return err
}

type keySpec struct {
	key, code, text string
	vk              int
}

func keyFor(r rune) keySpec {
	switch {
	case r == '\n' || r == '\r':
		return keySpec{"Enter", "Enter", "\r", 13}
	case r == '\t':
		return keySpec{"Tab", "Tab", "", 9}
	case r == ' ':
		return keySpec{" ", "Space", " ", 32}
	case r >= '0' && r <= '9':
		return keySpec{string(r), "Digit" + string(r), string(r), int(r)}
	case r >= 'a' && r <= 'z':
		up := r - 32
		return keySpec{string(r), "Key" + string(up), string(r), int(up)}
	case r >= 'A' && r <= 'Z':
		return keySpec{string(r), "Key" + string(r), string(r), int(r)}
	}
	return keySpec{string(r), "", string(r), 0}
}

func (h *Host) actionsList() ([]Action, error) {
	out, err := h.evalJS(actionsJS)
	if err != nil {
		return nil, err
	}
	var acts []Action
	if err := json.Unmarshal([]byte(cleanJS(out)), &acts); err != nil {
		return nil, err
	}
	if len(acts) > maxActionCollect {
		acts = acts[:maxActionCollect]
	}
	return acts, nil
}

func formatActions(acts []Action) string {
	if len(acts) > maxActions {
		acts = acts[:maxActions]
	}
	var b strings.Builder
	for _, a := range acts {
		dis := ""
		if a.Disabled {
			dis = " (disabled)"
		}
		ref := a.Ref
		if ref != "" {
			ref = "[ref=" + ref + "]  "
		}
		fmt.Fprintf(&b, "  %s%s  <%s> %q%s\n", ref, a.Selector, a.Tag, a.Text, dis)
	}
	return strings.TrimRight(b.String(), "\n")
}

// refreshActions caches the formatted action list on the host. It never
// relaunches Chrome; with no live page it leaves the previous list in place.
func (h *Host) refreshActions() {
	if !h.chromeAlive() {
		return
	}
	acts, err := h.actionsList()
	if err != nil {
		return
	}
	h.refs.store(acts)
	h.mu.Lock()
	h.actions = formatActions(acts)
	h.mu.Unlock()
}

func (h *Host) actionsNote() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.actions == "" {
		return ""
	}
	return "\nvisible actions now:\n" + h.actions
}

func (h *Host) pageText() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.snap
}

func (h *Host) setLastAction(s string) {
	h.mu.Lock()
	h.lastClick = s
	h.mu.Unlock()
}

func truncRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}
