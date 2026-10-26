// 本文件把原始录制事件收成步骤：点击后紧跟的输入并成 fill，连续滚动只留最后一次，about:blank 上的空白点击并成导航。
// 输入值收成 {{名字}}。描述里的英文句子是步骤数据。不调用模型。

package browser

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// SemanticStep 是收好的一步。Kind 取 open、navigate、click、fill、select、press、scroll。
type SemanticStep struct {
	ID          string       `json:"id"`
	Kind        string       `json:"kind"`
	URL         string       `json:"url,omitempty"`
	Value       string       `json:"value,omitempty"`
	Key         string       `json:"key,omitempty"`
	Target      RecordTarget `json:"target,omitempty"`
	ExpectURL   string       `json:"expect_url,omitempty"`
	Description string       `json:"description,omitempty"`
}

// SkillInput 是草稿里要操作者提供的一项。Kind 取 secret、email、string。
type SkillInput struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
	Example  string `json:"example,omitempty"`
}

// StepsDocument 是一次录制收好的步骤、输入和技能名。
type StepsDocument struct {
	StartURL  string         `json:"start_url"`
	Steps     []SemanticStep `json:"steps"`
	Inputs    []SkillInput   `json:"inputs"`
	SkillName string         `json:"skill_name"`
}

// emailRE 用来把看起来像邮箱的值收成 email 输入。
var emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// collapseRecordEvents 把事件收成 StepsDocument。起始地址可回放时先加一步 open。点击后四条之内的同目标 input 或 change 并成 fill。连续 scroll 只留最后一条。
// events 按 Kind 处理 navigation、click、input、change、keydown 和 scroll，其他 Kind 忽略。
// startURL 非空且 skipReplayURL 为假时先加一步 open。导航地址与它不同时再加 navigate。最后写入文档的 StartURL，并传给 inferSkillName。
func collapseRecordEvents(events []RecordEvent, startURL string) StepsDocument {
	used := map[string]bool{}
	var inputs []SkillInput
	var steps []SemanticStep
	add := func(st SemanticStep) {
		st.ID = fmt.Sprintf("step_%03d", len(steps)+1)
		steps = append(steps, st)
	}
	if startURL != "" && !skipReplayURL(startURL) {
		add(SemanticStep{Kind: "open", URL: startURL, Description: "Open " + startURL})
	}
	for i := 0; i < len(events); i++ {
		ev := events[i]
		switch ev.Kind {
		case "navigation":
			if skipReplayURL(ev.URL) {
				continue
			}
			if len(steps) > 0 {
				prev := &steps[len(steps)-1]
				if prev.Kind == "click" || prev.Kind == "press" {
					prev.ExpectURL = pathHint(ev.URL)
					continue
				}
			}
			if ev.URL != startURL {
				add(SemanticStep{Kind: "navigate", URL: ev.URL, Description: "Navigate to " + ev.URL})
			}
		case "click":
			if isBlankBodyClick(ev) {
				if next := nextNavigation(events, i+1); next != "" && !skipReplayURL(next) {
					add(SemanticStep{Kind: "navigate", URL: next, Description: "Navigate to " + next})
					continue
				}
			}
			if isLowSignalClick(ev) {
				continue
			}
			if next, ok := nextInputOnSame(events, i+1, ev.Target); ok {
				val, in := varFor(ev.Target, next.Value, used)
				if in.Name != "" {
					inputs = append(inputs, in)
				}
				add(SemanticStep{
					Kind:        "fill",
					Target:      ev.Target,
					Value:       val,
					Description: "Fill " + targetName(ev.Target),
				})
				i = skipSameFill(events, i+1, ev.Target)
				continue
			}
			add(SemanticStep{Kind: "click", Target: ev.Target, Description: "Click " + targetName(ev.Target)})
		case "input", "change":
			if len(steps) > 0 && steps[len(steps)-1].Kind == "fill" && sameRecordTarget(steps[len(steps)-1].Target, ev.Target) {
				continue
			}
			val, in := varFor(ev.Target, ev.Value, used)
			if in.Name != "" {
				inputs = append(inputs, in)
			}
			kind := "fill"
			if isSelectTarget(ev.Target) {
				kind = "select"
			}
			add(SemanticStep{Kind: kind, Target: ev.Target, Value: val, Description: kind + " " + targetName(ev.Target)})
		case "keydown":
			if ev.Key == "" {
				continue
			}
			add(SemanticStep{Kind: "press", Key: ev.Key, Description: "Press " + ev.Key})
		case "scroll":
			// 下一条还是滚动时先跳过，只留这段滚动的最后一次
			if i+1 < len(events) && events[i+1].Kind == "scroll" {
				continue
			}
			add(SemanticStep{Kind: "scroll", Value: fmt.Sprintf("%d", int(ev.DeltaY)), Description: "Scroll"})
		}
	}
	return StepsDocument{
		StartURL:  startURL,
		Steps:     steps,
		Inputs:    inputs,
		SkillName: inferSkillName(startURL),
	}
}

// targetName 按无障碍名、标签、占位、正文、id、name 属性的顺序取第一个非空。都空时用标签名，再空则是 element。
func targetName(t RecordTarget) string {
	if s := firstNonEmpty(t.Name, t.Label, t.Placeholder, t.Text, t.ID, t.NameAttr); s != "" {
		return s
	}
	if t.Tag != "" {
		return t.Tag
	}
	return "element"
}

// sameRecordTarget 先比角色、名字、name 属性和 id 拼成的键，再看选择器候选有没有交集。
// b 与 a 比较角色、targetName、NameAttr 和 ID 拼成的键，再看 SelectorCandidates 有没有交集。
func sameRecordTarget(a, b RecordTarget) bool {
	key := func(t RecordTarget) string {
		return strings.ToLower(t.Role + "|" + targetName(t) + "|" + t.NameAttr + "|" + t.ID)
	}
	if key(a) != "|||" && key(a) == key(b) {
		return true
	}
	as, bs := map[string]struct{}{}, map[string]struct{}{}
	for _, s := range a.SelectorCandidates {
		as[s] = struct{}{}
	}
	for _, s := range b.SelectorCandidates {
		if _, ok := as[s]; ok {
			return true
		}
		bs[s] = struct{}{}
	}
	return false
}

// nextInputOnSame 在后面最多 4 条里找同目标的 input 或 change。先遇到 click 或 navigation 就返回没有。
// events 从 from 向后看，下标小于 from 加 4。
// from 是起始下标，循环不超过 from 加 4。
// target 传给 sameRecordTarget。
func nextInputOnSame(events []RecordEvent, from int, target RecordTarget) (RecordEvent, bool) {
	for i := from; i < len(events) && i < from+4; i++ {
		if events[i].Kind == "input" || events[i].Kind == "change" {
			if sameRecordTarget(events[i].Target, target) {
				return events[i], true
			}
		}
		if events[i].Kind == "click" || events[i].Kind == "navigation" {
			return RecordEvent{}, false
		}
	}
	return RecordEvent{}, false
}

// skipSameFill 跳过从 from 起连续的、同一目标的 input 和 change，返回最后一条的下标。
// events 从 from 向后看，Kind 是 input 或 change 且与 target 同一目标时继续。
// target 传给 sameRecordTarget。
func skipSameFill(events []RecordEvent, from int, target RecordTarget) int {
	i := from
	for i < len(events) && (events[i].Kind == "input" || events[i].Kind == "change") && sameRecordTarget(events[i].Target, target) {
		i++
	}
	return i - 1
}

// nextNavigation 在后面最多 5 条里找一条可回放的 navigation，返回它的地址。
// events 从 from 向后看，下标小于 from 加 5，找 Kind 为 navigation 且 skipReplayURL 为假的 URL。
// from 是起始下标。
func nextNavigation(events []RecordEvent, from int) string {
	for i := from; i < len(events) && i < from+5; i++ {
		if events[i].Kind == "navigation" && !skipReplayURL(events[i].URL) {
			return events[i].URL
		}
	}
	return ""
}

// isBlankBodyClick 在点击发生在 about:blank，且目标是 body、html 或没有标签时返回真。
// ev 的 Kind 不是 click 时返回假。URL 不是 about:blank 且不以 about:blank 开头时返回假。Target.Tag 转小写后是 body、html 或空字符串时返回真。
func isBlankBodyClick(ev RecordEvent) bool {
	if ev.Kind != "click" {
		return false
	}
	if ev.URL != "about:blank" && !strings.HasPrefix(ev.URL, "about:blank") {
		return false
	}
	tag := strings.ToLower(ev.Target.Tag)
	return tag == "body" || tag == "html" || tag == ""
}

// isLowSignalClick 丢掉正文很长的 div，以及没有选择器候选、正文里却带 :nth-of-type 的点击。
// ev 的 Target.Tag 是 div 且 Text 的长度大于等于 160 时返回真。SelectorCandidates 为空且 Text 含有 :nth-of-type 时返回真。
func isLowSignalClick(ev RecordEvent) bool {
	if ev.Target.Tag == "div" && len(ev.Target.Text) >= 160 {
		return true
	}
	if len(ev.Target.SelectorCandidates) == 0 && strings.Contains(ev.Target.Text, ":nth-of-type") {
		return true
	}
	return false
}

// isSelectTarget 在标签是 select 或角色是 combobox 时返回真。
func isSelectTarget(t RecordTarget) bool {
	return strings.EqualFold(t.Tag, "select") || strings.EqualFold(t.Role, "combobox")
}

// varFor 给一个输入值起 {{名字}}。已经是占位符的当成 secret。邮箱、搜索框各有固定名字。重名时加 _2、_3。
// target 的 Name、Label、Placeholder、NameAttr、ID、Text 拼成小写文本，用来选择名字。
// value 以 {{ 开头且以 }} 结尾时名字用 sensitive_value、种类用 secret。否则可能作为 email 或 query 的示例，并去掉空白后交给 emailRE。
// used 里已经有这个名字时从 _2 起加数字后缀，最后把用过的名字设为真。
func varFor(target RecordTarget, value string, used map[string]bool) (string, SkillInput) {
	hay := strings.ToLower(strings.Join([]string{
		target.Name, target.Label, target.Placeholder, target.NameAttr, target.ID, target.Text,
	}, " "))
	var base, kind, example string
	switch {
	case strings.HasPrefix(value, "{{") && strings.HasSuffix(value, "}}"):
		base, kind = "sensitive_value", "secret"
	case strings.Contains(hay, "email") || emailRE.MatchString(strings.TrimSpace(value)):
		base, kind, example = "email", "email", value
	case strings.Contains(hay, "search") || strings.Contains(hay, "query") || strings.Contains(hay, "搜"):
		base, kind, example = "query", "string", value
	default:
		base = slugInput(firstNonEmpty(target.NameAttr, target.ID, target.Placeholder, hay))
		kind, example = "string", value
	}
	name := base
	for i := 2; used[name]; i++ {
		name = fmt.Sprintf("%s_%d", base, i)
	}
	used[name] = true
	in := SkillInput{Name: name, Kind: kind, Required: true, Example: example}
	return "{{" + name + "}}", in
}

// slugInput 把字符串收成小写的名字，非字母数字变成下划线，最长 40。空的或纯数字时返回 value。
func slugInput(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if b.Len() > 0 && b.String()[b.Len()-1] != '_' {
			b.WriteByte('_')
		}
		if b.Len() >= 40 {
			break
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" || isDigits(out) {
		return "value"
	}
	return out
}

// isDigits 在非空且每个字符都是数字时返回真。
func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// pathHint 返回地址路径的最后一段，前面加斜杠。根路径或解析失败时返回空字符串。
// raw 交给 url.Parse。失败、路径为空或路径是 / 时返回空字符串。否则返回路径最后一段，前面加斜杠。
func pathHint(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Path == "" || u.Path == "/" {
		return ""
	}
	bits := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(bits) == 0 {
		return ""
	}
	return "/" + bits[len(bits)-1]
}

// inferSkillName 用起始地址的主机名做成 recorded-<主机>。没有主机时用 recorded-browser-workflow。www. 前缀和端口去掉。
// start 为空、url.Parse 失败或主机为空时返回 recorded-browser-workflow。否则用小写主机名去掉 www. 和端口，再经 slugInput 拼成 recorded- 加结果。
func inferSkillName(start string) string {
	if start == "" {
		return "recorded-browser-workflow"
	}
	u, err := url.Parse(start)
	if err != nil || u.Host == "" {
		return "recorded-browser-workflow"
	}
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")
	host = strings.Split(host, ":")[0]
	return "recorded-" + slugInput(host)
}
