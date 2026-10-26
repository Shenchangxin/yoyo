// 本文件拥有隔离浏览器的点击、输入、填充和等待。指针和按键都是 CDP 的真实输入事件。
// 失败时列出页面上实际有的控件。不调用 runtime.Run。actionsJS 里的脚本是数据。

package browser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// actionWait 是点击和输入自动等待的时间，4 秒。
	actionWait = 4 * time.Second
	// actionPoll 是等待循环的间隔，120 毫秒。
	actionPoll = 120 * time.Millisecond
	// maxActions 是展示给模型的控件列表上限，48。
	maxActions = 48
	// maxActionCollect 是页面遍历保留的控件数，60。结构化快照可以用到这些多出来的。
	maxActionCollect = 60
)

// Action 是页面上可交互的一个元素。Ref 是这次遍历分配的短句柄，例如 btn_0，导航之前有效。
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

// probe 是一次解析并检查的 JSON 结果。OK 为假时 Reason 说明原因。
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

// probeJS 在页面里解析 sel 并检查它。focusOnly 为真时只聚焦，否则返回要点按的中心点。它自己不点击。
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

// actionsJS 列出可见、能定位的交互元素并带上 ref。脚本正文是数据。
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

// Click 解析选择器，等到它可点，用真实指针事件按下，并报告可见正文有没有变化。
// sel 作为 click 的 detail 传给 record，并传给 clickCDP。返回文字里 Ref 非空时改用 ref= 加 Ref，否则用 sel。
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

// Type 在给了选择器时先聚焦，再把每个字符当成真实按键发出，这样文档或 window 上的 keydown 能收到。
// sel 作为 type 的 detail 传给 record，并和 text 一起传给 typeCDP。去掉空白后为空时，返回文字里写成 the focused element。
// text 传给 typeCDP。返回文字里的按键数是 text 的 rune 个数。
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

// Fill 替换字段里的现有内容。先全选并删除，再输入，所以第二次填充不会接在后面。
// sel 作为 fill 的 detail 传给 record。去掉空白后为空时返回 browser: fill needs a selector or ref。否则传给 waitFor。返回文字里仍用 sel。
// text 传给 typeCDP，选择器传的是空字符串。返回文字里的个数是 text 的 rune 个数。
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

// WaitVisible 等到选择器可操作，但不点击。序列可以在目标被挡住之前停下来。
// sel 作为 wait 的 detail 传给 record，并传给 waitFor。返回文字里带上 sel。
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

// Refresh 重新读取当前页的地址、标题、正文和控件，不导航。Chrome 没在跑时 live 为假，快照是上一份。
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

// clickCDP 在等到目标可点后，依次发送 mouseMoved、mousePressed、mouseReleased。
// sel 传给 waitFor。得到坐标后发送 mouseMoved、mousePressed 和 mouseReleased。
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

// typeCDP 在选择器非空时先聚焦，再对每个字符调用 keyTap。
// sel 去掉空白后非空时传给 waitFor，focusOnly 为真。
// text 的每个 rune 传给 keyTap。
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

// waitFor 轮询到目标可操作或时间用尽。过期的 ref 会按上次快照的 testid、id 或无障碍名再试一次，避免重绘后去猜新的 CSS。
// sel 传给 probeOn。未成功时传给 matchFallback，得到的选择器不同再传给 probeOn，并把 sel 写进 match 的 detail。仍失败则传给 explainMiss。
// focusOnly 传给 probeOn。回退匹配时也传给 probeOn。
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

// probeOn 执行 probeJS 并把 JSON 解成 probe。没有结果时返回错误。
// sel 传给 probeJS。JSON 解不开时错误文案带上 sel。
// focusOnly 传给 probeJS。
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

// coveredWhy 说明命中测试为什么失败。by 为空或 nothing 时，中心点没有命中元素，是在视口外或被祖先裁切，并不是被一层东西盖住。
// sel 放进返回字符串的开头。
func coveredWhy(sel, by string) string {
	if by == "" || by == "nothing" {
		return fmt.Sprintf("%s has no hit-test element at its centre (outside the viewport or clipped by an ancestor), so a real click would not reach it", sel)
	}
	return fmt.Sprintf("%s is covered by %s, so a real click would not reach it", sel, by)
}

// explainMiss 把一次失败写成错误，并附上页面上实际有的控件，供下一次选择器用。
// sel 按 p.Reason 写进错误说明。Reason 是 covered 时传给 coveredWhy。
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

// selectAllClear 依次发送 Ctrl+A 和 Meta+A，再按退格，用来清掉当前字段。
func (h *Host) selectAllClear() error {
	// Ctrl+A 和 Meta+A 都发一遍。Chrome 会忽略不是本机的那个修饰键。
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

// keyTap 把一个字符发成带 text 的 keyDown 和 keyUp，从而触发 keydown、keypress 和 input。
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

// keySpec 是一次按键的 key、code、text 和 Windows 虚拟键码。
type keySpec struct {
	key, code, text string
	vk              int
}

// keyFor 把一个字符收成 keySpec。换行是 Enter，制表符是 Tab，数字和字母带上 Digit 或 Key 代码。
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

// actionsList 执行 actionsJS，并把结果截到 maxActionCollect。
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

// formatActions 把控件收成多行文字，每行带 ref、选择器、标签和正文。超过 maxActions 的丢掉。禁用的标上 disabled。
// acts 长度大于 maxActions 时只留前面 maxActions 条。每条写出 Ref、Selector、Tag 和 Text，Disabled 为真时附上 (disabled)。
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

// refreshActions 把格式化后的控件列表记在宿主上，并写入 ref 缓存。Chrome 没在跑时保留上一份，不重新启动。
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

// actionsNote 在有控件列表时返回带换行的「visible actions now」文本，没有则返回空字符串。
func (h *Host) actionsNote() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.actions == "" {
		return ""
	}
	return "\nvisible actions now:\n" + h.actions
}

// pageText 返回宿主记下的页面正文。
func (h *Host) pageText() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.snap
}

// setLastAction 记下最近一次点击的观察文字。
func (h *Host) setLastAction(s string) {
	h.mu.Lock()
	h.lastClick = s
	h.mu.Unlock()
}

// truncRunes 按字符截断。超过 n 个字符时末尾加省略号。
func truncRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}
