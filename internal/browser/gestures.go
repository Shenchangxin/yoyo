// 本文件拥有滚动、悬停、下拉选择、命名按键和条件等待。
// 指针和按键都走 Input.dispatch* 的真实事件。不调用 runtime.Run。

package browser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Scroll 按方向滚动 amount 像素。方向是 up、down、left、right，空则 down。amount 小于等于 0 时用 400，大于 4000 时收成 4000。给了选择器就先等到它可点，再用它的坐标；否则用视口中心。
// direction 作为 scroll 的 detail 传给 record。去掉空白并转小写后为空则当成 down。不是 up、down、left、right 时返回错误，错误里带上原来的 direction。
// sel 去掉空白后非空时传给 waitFor，用返回的坐标。否则用视口中心。
func (h *Host) Scroll(direction string, amount int, sel string) (string, error) {
	h.record("scroll", direction)
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return "", fmt.Errorf("browser: scroll: %w", err)
	}
	dir := strings.ToLower(strings.TrimSpace(direction))
	if dir == "" {
		dir = "down"
	}
	if amount <= 0 {
		amount = 400
	}
	if amount > 4000 {
		amount = 4000
	}
	x, y := 400.0, 300.0
	if strings.TrimSpace(sel) != "" {
		p, err := h.waitFor(sel, false)
		if err != nil {
			return "", err
		}
		x, y = p.X, p.Y
	} else if cx, cy, err := h.viewportCenter(); err == nil {
		x, y = cx, cy
	}
	dx, dy := 0, 0
	switch dir {
	case "up":
		dy = -amount
	case "down":
		dy = amount
	case "left":
		dx = -amount
	case "right":
		dx = amount
	default:
		return "", fmt.Errorf("browser: scroll direction %q must be up, down, left, or right", direction)
	}
	if _, err := h.callCDP("Input.dispatchMouseEvent", map[string]any{
		"type": "mouseWheel", "x": x, "y": y, "deltaX": dx, "deltaY": dy,
	}); err != nil {
		return "", err
	}
	time.Sleep(120 * time.Millisecond)
	h.syncView()
	h.refreshActions()
	return fmt.Sprintf("scrolled %s by %dpx%s%s", dir, amount, h.pageErrorsNote(), h.actionsNote()), nil
}

// viewportCenter 读 window.innerWidth 和 innerHeight 的一半。读失败或边长小于 2 时返回 400, 300。
func (h *Host) viewportCenter() (float64, float64, error) {
	raw, err := h.evalJS(`JSON.stringify({w: window.innerWidth||800, h: window.innerHeight||600})`)
	if err != nil {
		return 0, 0, err
	}
	var box struct{ W, H float64 }
	if err := json.Unmarshal([]byte(cleanJS(raw)), &box); err != nil || box.W < 2 || box.H < 2 {
		return 400, 300, nil
	}
	return box.W / 2, box.H / 2, nil
}

// Hover 把指针移到选择器上，不按下。只在 mouseenter 时打开的菜单需要这个。
// sel 作为 hover 的 detail 传给 record，并传给 waitFor。返回文字里 Ref 非空时改用 ref= 加 Ref，否则用 sel。
func (h *Host) Hover(sel string) (string, error) {
	h.record("hover", sel)
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return "", fmt.Errorf("browser: hover %s: %w", sel, err)
	}
	p, err := h.waitFor(sel, false)
	if err != nil {
		return "", err
	}
	if _, err := h.callCDP("Input.dispatchMouseEvent", map[string]any{
		"type": "mouseMoved", "x": p.X, "y": p.Y,
	}); err != nil {
		return "", err
	}
	time.Sleep(80 * time.Millisecond)
	h.syncView()
	h.refreshActions()
	handle := sel
	if p.Ref != "" {
		handle = "ref=" + p.Ref
	}
	return fmt.Sprintf("hovered %s → <%s> %q at (%d,%d)%s%s", handle, p.Tag, truncRunes(p.Text, 40), int(p.X), int(p.Y), h.pageErrorsNote(), h.actionsNote()), nil
}

// Select 等到控件可聚焦，再把 value 写上并派发 input 和 change。选择器或值为空时返回错误。
// sel 作为 select 的 detail 传给 record。去掉空白后为空时返回 browser: select needs selector/ref and value。否则传给 waitFor，并经 json.Marshal 写进页面脚本。
func (h *Host) Select(sel, value string) (string, error) {
	h.record("select", sel)
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return "", fmt.Errorf("browser: select %s: %w", sel, err)
	}
	if strings.TrimSpace(sel) == "" || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("browser: select needs selector/ref and value")
	}
	if _, err := h.waitFor(sel, true); err != nil {
		return "", err
	}
	quotedSel, _ := json.Marshal(sel)
	quotedVal, _ := json.Marshal(value)
	raw, err := h.evalJS(fmt.Sprintf(`(() => {
%s
  const rec = __yoyoResolveRec(%s);
  if (!rec || !rec.el) return JSON.stringify({ok:false, reason:"missing"});
  const el = rec.el;
  el.value = %s;
  el.dispatchEvent(new Event("input", {bubbles:true}));
  el.dispatchEvent(new Event("change", {bubbles:true}));
  return JSON.stringify({ok:true, value: el.value, tag: el.tagName.toLowerCase(), ref: rec.ref});
})()`, pageHelpers, quotedSel, quotedVal))
	if err != nil {
		return "", err
	}
	var out struct {
		OK     bool   `json:"ok"`
		Reason string `json:"reason"`
		Value  string `json:"value"`
		Tag    string `json:"tag"`
		Ref    string `json:"ref"`
	}
	if err := json.Unmarshal([]byte(cleanJS(raw)), &out); err != nil || !out.OK {
		why := out.Reason
		if why == "" {
			why = "select failed"
		}
		return "", fmt.Errorf("browser: %s", why)
	}
	h.syncView()
	h.refreshActions()
	handle := sel
	if out.Ref != "" {
		handle = "ref=" + out.Ref
	}
	return fmt.Sprintf("selected %q on %s <%s>%s%s", out.Value, handle, out.Tag, h.pageErrorsNote(), h.actionsNote()), nil
}

// Press 发送一个命名按键：Enter、Tab、Escape、Backspace、空格或方向键。单个字符走 keyFor。
func (h *Host) Press(key string) (string, error) {
	h.record("press", key)
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return "", fmt.Errorf("browser: press: %w", err)
	}
	spec, err := namedKey(key)
	if err != nil {
		return "", err
	}
	down := map[string]any{"type": "keyDown", "key": spec.key, "code": spec.code, "windowsVirtualKeyCode": spec.vk}
	if spec.text != "" {
		down["text"] = spec.text
		down["unmodifiedText"] = spec.text
	}
	up := map[string]any{"type": "keyUp", "key": spec.key, "code": spec.code, "windowsVirtualKeyCode": spec.vk}
	if _, err := h.callCDP("Input.dispatchKeyEvent", down); err != nil {
		return "", err
	}
	if _, err := h.callCDP("Input.dispatchKeyEvent", up); err != nil {
		return "", err
	}
	time.Sleep(80 * time.Millisecond)
	h.syncView()
	h.refreshActions()
	return fmt.Sprintf("pressed %s%s%s", spec.key, h.pageErrorsNote(), h.actionsNote()), nil
}

// namedKey 把名字或单个字符收成 keySpec。空字符串返回错误。不认识的名字返回 unsupported key。
// raw 去掉空白后为空时返回 browser: press needs a key。只有一个字符时传给 keyFor。否则转小写后匹配 enter、return、tab、escape、esc、backspace、space、spacebar、arrowup、up、arrowdown、down、arrowleft、left、arrowright、right、home、end，其余返回 unsupported key，错误里带上原来的 raw。
func namedKey(raw string) (keySpec, error) {
	k := strings.TrimSpace(raw)
	if k == "" {
		return keySpec{}, fmt.Errorf("browser: press needs a key")
	}
	if len([]rune(k)) == 1 {
		return keyFor([]rune(k)[0]), nil
	}
	switch strings.ToLower(k) {
	case "enter", "return":
		return keySpec{"Enter", "Enter", "\r", 13}, nil
	case "tab":
		return keySpec{"Tab", "Tab", "", 9}, nil
	case "escape", "esc":
		return keySpec{"Escape", "Escape", "", 27}, nil
	case "backspace":
		return keySpec{"Backspace", "Backspace", "", 8}, nil
	case "space", "spacebar":
		return keySpec{" ", "Space", " ", 32}, nil
	case "arrowup", "up":
		return keySpec{"ArrowUp", "ArrowUp", "", 38}, nil
	case "arrowdown", "down":
		return keySpec{"ArrowDown", "ArrowDown", "", 40}, nil
	case "arrowleft", "left":
		return keySpec{"ArrowLeft", "ArrowLeft", "", 37}, nil
	case "arrowright", "right":
		return keySpec{"ArrowRight", "ArrowRight", "", 39}, nil
	case "home":
		return keySpec{"Home", "Home", "", 36}, nil
	case "end":
		return keySpec{"End", "End", "", 35}, nil
	default:
		return keySpec{}, fmt.Errorf("browser: unsupported key %q (Enter, Tab, Escape, Backspace, Space, arrows)", raw)
	}
}

// WaitCondition 轮询到选择器可操作、正文包含 contains，或地址包含 urlSub。三个都空时只睡眠 d。d 小于等于 0 时用 200 毫秒。超时返回最后一次没满足的原因。
// sel 去掉空白。非空时传给 probeOn，并写进 wait 的 detail。三个条件都空时不探测，只睡眠。
func (h *Host) WaitCondition(sel, contains, urlSub string, d time.Duration) (string, error) {
	if d <= 0 {
		d = 200 * time.Millisecond
	}
	sel = strings.TrimSpace(sel)
	contains = strings.TrimSpace(contains)
	urlSub = strings.TrimSpace(urlSub)
	if sel == "" && contains == "" && urlSub == "" {
		time.Sleep(d)
		return fmt.Sprintf("waited %d ms", d.Milliseconds()), nil
	}
	h.record("wait", sel+" "+contains+" "+urlSub)
	if err := h.ensureChrome(h.previewStartURL()); err != nil {
		return "", fmt.Errorf("browser: wait: %w", err)
	}
	deadline := time.Now().Add(d)
	var last string
	for {
		ok := true
		var parts []string
		if urlSub != "" {
			href, _ := h.evalOnConn("location.href")
			href = cleanJS(href)
			if !strings.Contains(href, urlSub) {
				ok = false
				last = fmt.Sprintf("url %q does not contain %q", href, urlSub)
			} else {
				parts = append(parts, "url")
			}
		}
		if sel != "" {
			p, err := h.probeOn(sel, false)
			if err != nil || !p.OK {
				ok = false
				if err != nil {
					last = err.Error()
				} else {
					last = p.Reason
				}
			} else {
				parts = append(parts, "selector")
			}
		}
		if contains != "" {
			text, _ := h.evalOnConn(`document.body ? (document.body.innerText || "") : ""`)
			if !strings.Contains(cleanJS(text), contains) {
				ok = false
				last = fmt.Sprintf("page text does not contain %q", truncRunes(contains, 80))
			} else {
				parts = append(parts, "text")
			}
		}
		if ok {
			h.syncView()
			h.refreshActions()
			return fmt.Sprintf("wait satisfied (%s)%s%s", strings.Join(parts, "+"), h.pageErrorsNote(), h.actionsNote()), nil
		}
		if !time.Now().Before(deadline) {
			if last == "" {
				last = "timeout"
			}
			return "", fmt.Errorf("browser: wait: %s", last)
		}
		time.Sleep(actionPoll)
	}
}
