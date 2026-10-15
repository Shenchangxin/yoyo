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
	maxActions = 24
)

// Action is an element the page offers for interaction.
type Action struct {
	Selector string `json:"selector"`
	Tag      string `json:"tag"`
	Text     string `json:"text"`
	Disabled bool   `json:"disabled,omitempty"`
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
	Error  string  `json:"error"`
}

// pageHelpers holds the single definition of interactive, visible and text
// that the action list, click and type all share. Only function declarations
// go here: each evaluation wraps them in its own IIFE, so nothing leaks into
// the page's global scope or collides across calls.
const pageHelpers = `
function __yoyoVisible(el) {
  const r = el.getBoundingClientRect();
  if (r.width < 2 || r.height < 2) return false;
  const cs = getComputedStyle(el);
  return cs.visibility !== "hidden" && cs.display !== "none" && parseFloat(cs.opacity) > 0;
}
function __yoyoText(el) {
  const t = el.innerText || el.value || el.getAttribute("aria-label") || el.title || "";
  return String(t).replace(/\s+/g, " ").trim().slice(0, 90);
}
function __yoyoCandidates() {
  const out = Array.from(document.querySelectorAll('button,a[href],input:not([type=hidden]),select,textarea,summary,[role=button],[role=option],[role=link],[role=menuitem],[role=tab],[role=radio],[role=checkbox],[onclick],[data-opt],[data-action],[tabindex]:not([tabindex="-1"])'));
  const seen = new Set(out);
  if (document.body) {
    const all = document.body.querySelectorAll("*");
    for (let i = 0; i < all.length && i < 3000; i++) {
      const el = all[i];
      if (seen.has(el) || getComputedStyle(el).cursor !== "pointer") continue;
      const pe = el.parentElement;
      if (pe && getComputedStyle(pe).cursor === "pointer") continue;
      out.push(el);
      seen.add(el);
    }
  }
  return out;
}
function __yoyoSel(el) {
  const ident = /^[A-Za-z][\w-]*$/;
  const ok = (s) => { try { return document.querySelector(s) === el; } catch (e) { return false; } };
  if (el.id && ident.test(el.id) && ok("#" + el.id)) return "#" + el.id;
  const parts = [];
  for (let n = el; n && n.nodeType === 1 && n !== document.documentElement; n = n.parentElement) {
    if (n.id && ident.test(n.id)) { parts.unshift("#" + n.id); break; }
    let p = n.tagName.toLowerCase();
    if (n.parentElement) {
      const same = Array.from(n.parentElement.children).filter((c) => c.tagName === n.tagName);
      if (same.length > 1) p += ":nth-of-type(" + (same.indexOf(n) + 1) + ")";
    }
    parts.unshift(p);
  }
  const s = parts.join(" > ");
  return ok(s) ? s : "";
}
function __yoyoResolve(sel) {
  if (sel.indexOf("text=") === 0) {
    const want = sel.slice(5).trim();
    let best = null;
    let bestLen = Infinity;
    for (const c of __yoyoCandidates()) {
      const t = __yoyoText(c);
      if (!t || t.indexOf(want) < 0 || !__yoyoVisible(c)) continue;
      if (t.length < bestLen) { best = c; bestLen = t.length; }
    }
    return best;
  }
  return document.querySelector(sel);
}
function __yoyoDescribe(el) {
  if (!el) return "nothing";
  const id = el.id ? "#" + el.id : "";
  const cls = el.classList && el.classList.length ? "." + Array.from(el.classList).slice(0, 2).join(".") : "";
  const t = __yoyoText(el).slice(0, 30);
  return el.tagName.toLowerCase() + id + cls + (t ? ' "' + t + '"' : "");
}
`

// probeJS resolves sel, checks it, and either focuses it (focusOnly) or
// returns the centre point to press. It never clicks by itself.
func probeJS(sel string, focusOnly bool) string {
	quoted, _ := json.Marshal(sel)
	return fmt.Sprintf(`(() => {
%s
  const sel = %s;
  let el;
  try { el = __yoyoResolve(sel); } catch (e) { return JSON.stringify({ok:false, reason:"bad-selector", error:String(e)}); }
  if (!el) return JSON.stringify({ok:false, reason:"missing"});
  if (!__yoyoVisible(el)) return JSON.stringify({ok:false, reason:"hidden", text:__yoyoText(el)});
  if (el.disabled) return JSON.stringify({ok:false, reason:"disabled", text:__yoyoText(el)});
  el.scrollIntoView({block:"center", inline:"center"});
  if (%t) {
    el.focus();
    return JSON.stringify({ok:true, tag:el.tagName.toLowerCase(), text:__yoyoText(el)});
  }
  const r = el.getBoundingClientRect();
  const x = r.left + r.width / 2;
  const y = r.top + r.height / 2;
  const top = document.elementFromPoint(x, y);
  if (!top || (top !== el && !el.contains(top))) {
    return JSON.stringify({ok:false, reason:"covered", by:__yoyoDescribe(top)});
  }
  return JSON.stringify({ok:true, x:x, y:y, tag:el.tagName.toLowerCase(), text:__yoyoText(el)});
})()`, pageHelpers, quoted, focusOnly)
}

// actionsJS lists the visible, addressable interactive elements.
const actionsJS = `(() => {
` + pageHelpers + `
  const out = [];
  for (const el of __yoyoCandidates()) {
    if (!__yoyoVisible(el)) continue;
    const t = __yoyoText(el);
    let s = __yoyoSel(el);
    if (!s && t) s = "text=" + t.slice(0, 40);
    if (!s) continue;
    out.push({selector: s, tag: el.tagName.toLowerCase(), text: t, disabled: !!el.disabled});
    if (out.length >= 40) break;
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
	obs := fmt.Sprintf("clicked %s → <%s> %q at (%d,%d); %s", sel, p.Tag, truncRunes(p.Text, 40), int(p.X), int(p.Y), note)
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

// Fill is the historical name for Type.
func (h *Host) Fill(sel, text string) (string, error) { return h.Type(sel, text) }

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
	snap = h.snapshotLocked()
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
func (h *Host) waitFor(sel string, focusOnly bool) (probe, error) {
	deadline := time.Now().Add(actionWait)
	for {
		p, err := h.probeOn(sel, focusOnly)
		if err != nil {
			return probe{}, err
		}
		if p.OK {
			return p, nil
		}
		if p.Reason == "bad-selector" || !time.Now().Before(deadline) {
			return probe{}, h.explainMiss(sel, p)
		}
		time.Sleep(actionPoll)
	}
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
	if len(acts) > maxActions {
		acts = acts[:maxActions]
	}
	return acts, nil
}

func formatActions(acts []Action) string {
	var b strings.Builder
	for _, a := range acts {
		dis := ""
		if a.Disabled {
			dis = " (disabled)"
		}
		fmt.Fprintf(&b, "  %s  <%s> %q%s\n", a.Selector, a.Tag, a.Text, dis)
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
