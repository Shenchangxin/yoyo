package browser

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

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

type SkillInput struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
	Example  string `json:"example,omitempty"`
}

type StepsDocument struct {
	StartURL  string         `json:"start_url"`
	Steps     []SemanticStep `json:"steps"`
	Inputs    []SkillInput   `json:"inputs"`
	SkillName string         `json:"skill_name"`
}

var emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

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
			// keep the last scroll before the next non-scroll
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

func targetName(t RecordTarget) string {
	if s := firstNonEmpty(t.Name, t.Label, t.Placeholder, t.Text, t.ID, t.NameAttr); s != "" {
		return s
	}
	if t.Tag != "" {
		return t.Tag
	}
	return "element"
}

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

func skipSameFill(events []RecordEvent, from int, target RecordTarget) int {
	i := from
	for i < len(events) && (events[i].Kind == "input" || events[i].Kind == "change") && sameRecordTarget(events[i].Target, target) {
		i++
	}
	return i - 1
}

func nextNavigation(events []RecordEvent, from int) string {
	for i := from; i < len(events) && i < from+5; i++ {
		if events[i].Kind == "navigation" && !skipReplayURL(events[i].URL) {
			return events[i].URL
		}
	}
	return ""
}

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

func isLowSignalClick(ev RecordEvent) bool {
	if ev.Target.Tag == "div" && len(ev.Target.Text) >= 160 {
		return true
	}
	if len(ev.Target.SelectorCandidates) == 0 && strings.Contains(ev.Target.Text, ":nth-of-type") {
		return true
	}
	return false
}

func isSelectTarget(t RecordTarget) bool {
	return strings.EqualFold(t.Tag, "select") || strings.EqualFold(t.Role, "combobox")
}

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

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

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
