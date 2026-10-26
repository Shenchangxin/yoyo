// 本文件按上次遍历记下的描述，在页面重绘后把过期的 ref 对回现在的控件。
// ref 本身在页面里重新走一遍才解析。分数低于 matchThreshold（45）算没对上。不调用模型。

package browser

import (
	"regexp"
	"strings"
	"sync"
)

// refName 匹配两到三个小写字母、下划线、数字，例如 btn_0。
var refName = regexp.MustCompile(`^[a-z]{2,3}_\d+$`)

// refCache 保存上次成功的控件遍历。页面重绘后句柄过期时，用这里的 testid、名字和 id 再试一次。
type refCache struct {
	mu    sync.Mutex
	list  []Action
	byRef map[string]Action
}

// store 换成这一批控件，并按 Ref 建索引。
// acts 复制进 list。Ref 非空的条目以 Ref 为键写入 byRef。
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

// lookup 按 ref 取出上次记下的控件。
func (c *refCache) lookup(ref string) (Action, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	a, ok := c.byRef[ref]
	return a, ok
}

// all 返回控件列表的副本。
func (c *refCache) all() []Action {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Action(nil), c.list...)
}

// clear 丢掉列表和索引。
func (c *refCache) clear() {
	c.mu.Lock()
	c.list = nil
	c.byRef = nil
	c.mu.Unlock()
}

// invalidateRefs 清掉宿主上的 ref 缓存。导航和换配置时调用。
func (h *Host) invalidateRefs() {
	if h == nil {
		return
	}
	h.refs.clear()
}

// parseRef 从 ref=btn_0、ref:btn_0、[ref=btn_0] 或单独的 btn_0 里取出句柄。空字符串表示这是 CSS 选择器或 text= 句柄。
// sel 去掉空白后为空则返回空字符串。成对的方括号去掉后，以 ref= 或 ref: 开头时返回第 4 个字符之后并再去掉空白。整段匹配 refName 时返回它，否则返回空字符串。
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

// rememberedTarget 把选择器对回缓存里的描述。ref 对上就返回那条；对不上也返回一个只带 Ref 和 Text 的 Action。text= 取出可见文字。CSS、#id 和 data-testid 再在全部缓存里找。
// sel 传给 parseRef。得到的 ref 非空时用来 lookup，没有缓存则返回只带该 Ref 和 Text 的 Action。以 text= 开头时取第 5 个字符之后的文字。否则在缓存里按 Selector、# 加 ID，以及 data-testid 比较。
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

// matchFallback 用记住的描述对当前页面再匹配一次。分数低于 matchThreshold，或结果既没有 Ref 也没有 Selector 时返回空。
// sel 传给 rememberedTarget。
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

// matchThreshold 是 matchTarget 的及格分。低于 45 算没对上。
const matchThreshold = 45

// matchTarget 给每个实况控件打分。选择器或 testid 重合加 80，id 加 70，name 属性加 60，无障碍名完全相同加 50、部分包含加 20，role 加 25。低于 45 时返回 not-found。
// want 的 Selector、TestID、ID、NameAttr、Role、Name、Text、Placeholder 用来打分。
// have 里每个控件与 want 比较，取得分最高且不低于 matchThreshold 的一个。
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

// selectorSet 收集选择器、data-testid 属性和 #id。
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

// overlap 看两个集合有没有相同的键。
// b 用来查找 a 里的键是否也存在。
func overlap(a, b map[string]struct{}) bool {
	for k := range a {
		if _, ok := b[k]; ok {
			return true
		}
	}
	return false
}

// normMatch 去掉空白并转成小写，供打分比较。
// s 去掉空白后转成小写。
func normMatch(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// firstNonEmpty 返回第一个非空白字符串。都空则返回空字符串。
// ss 从前往后找，返回第一个去掉空白后非空的字符串。都没有时返回空字符串。
func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}
