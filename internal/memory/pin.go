package memory

import (
	"sort"
	"strings"
	"time"
)

// Portrait 是已经留下的事实，按来源分开。每一组新的在前。不含推断出来的经历，也不含已经过去的安排。
type Portrait struct {
	Device      string
	Host        string
	OS          string
	Zone        string
	Git         string
	Hours       string
	Lines       []FactLine
	Said        []string
	Noted       []string
	Preferences []string
	Updated     time.Time
}

// FactLine 是一条工程线的名字和能核对的痕迹。
type FactLine struct {
	Name     string
	State    string
	Evidence string
	When     string
}

// pinSet 是画像按组排好的条目。当事人说的、偏好和模型记下的保留整条，好在写进提示词时带上日子。
type pinSet struct {
	device, host, os, zone, git, hours string
	lines                              []FactLine
	said, noted, prefs                 []Item
	updated                            time.Time
}

// pinSet 把条目分组。人点过「别再提」的不进画像。调用时要已持有锁。
func (s *Store) pinSet(now time.Time) pinSet {
	items := append([]Item(nil), s.items...)
	sort.SliceStable(items, func(i, j int) bool { return updatedOf(items[i]).After(updatedOf(items[j])) })
	var out pinSet
	// 后排的不进画像，还能搜到（backrow.go）。
	back := s.backSet(now)
	for _, it := range items {
		if it.Staging || it.Muted || back[it.ID] || strings.TrimSpace(it.Text) == "" || Expired(it, now) {
			continue
		}
		kept := true
		switch {
		case it.ID == "machine:device":
			out.device = it.Text
		case it.ID == "machine:host":
			out.host = it.Text
		case it.ID == "machine:os":
			out.os = it.Text
		case it.ID == "machine:zone":
			out.zone = it.Text
		case it.ID == "machine:git":
			out.git = it.Text
		case it.ID == "machine:hours" || (it.Source == "machine" && it.Project == "work"):
			out.hours = it.Text
		case it.Kind == KindProject:
			out.lines = append(out.lines, FactLine{Name: it.Text, State: it.State, Evidence: it.Evidence, When: it.When})
		case it.Kind == KindProfile && operatorSource(it.Source):
			out.said = append(out.said, it)
		case it.Kind == KindSemantic && operatorSource(it.Source):
			out.prefs = append(out.prefs, it)
		case it.Kind == KindProfile || it.Kind == KindSemantic:
			out.noted = append(out.noted, it)
		default:
			kept = false
		}
		if kept && updatedOf(it).After(out.updated) {
			out.updated = updatedOf(it)
		}
	}
	return out
}

// Portrait 把留下的条目分成设备、工作时间、工程线、当事人说过的、模型记下的和偏好。
// 偏好只收当事人自己说的；模型和工人记下的档案和偏好都在 Noted 里。
func (s *Store) Portrait() Portrait {
	s.mu.Lock()
	s.syncLocked()
	set := s.pinSet(s.clock())
	s.mu.Unlock()
	return Portrait{
		Device: set.device, Host: set.host, OS: set.os, Zone: set.zone, Git: set.git, Hours: set.hours,
		Lines: set.lines, Said: texts(set.said), Noted: texts(set.noted), Preferences: texts(set.prefs),
		Updated: set.updated,
	}
}

// ProfilePin 把已经留下的事实收成模型能读的短文。先写当事人说过的，再写偏好、模型记下的，本机的名字和时间放最后。
// 每一组新的在前；预算不够时先舍掉最旧的和本机的。budget 是字节上限，小于等于 0 时不限。
func (s *Store) ProfilePin(budget int) string {
	text, _ := s.ProfileUse(budget)
	return text
}

// ProfileUse 和 ProfilePin 写出同一段短文，并返回真正写进去的当事人说过的、偏好和模型记下的那几条。
// 本机清点不在返回里：那是名字和时间，不是人能说「不对」的记忆。界面用它列出这一轮带进去的记忆。
func (s *Store) ProfileUse(budget int) (string, []Item) {
	s.mu.Lock()
	s.syncLocked()
	now := s.clock()
	set := s.pinSet(now)
	s.mu.Unlock()
	w := pinWriter{budget: budget}
	var used []Item
	for _, g := range []struct {
		title string
		items []Item
	}{
		{"Said by the operator:\n", set.said},
		{"Preferences:\n", set.prefs},
		{"Noted by the assistant, not confirmed:\n", set.noted},
	} {
		for _, i := range w.group(g.title, pinLines(g.items, now)) {
			used = append(used, g.items[i])
		}
	}
	var machine []string
	for _, field := range []struct{ name, value string }{
		{"user", set.device}, {"host", set.host}, {"os", set.os}, {"zone", set.zone}, {"git", set.git}, {"weekday edits", set.hours},
	} {
		if field.value != "" {
			machine = append(machine, field.name+": "+field.value)
		}
	}
	for _, line := range set.lines {
		row := line.Name
		if line.State != "" {
			row += ": " + line.State
		}
		if line.Evidence != "" {
			row += "; " + line.Evidence
		}
		if line.When != "" {
			row += "; " + line.When
		}
		machine = append(machine, row)
	}
	w.group("From this machine, names and times only:\n", machine)
	return w.String(), used
}

// OperatorPin 是给工人看的短文：当事人说过的和他自己的偏好，最后是系统、时区和 Git 名字。
// 模型记下、当事人没确认的不写：工人能改文件、能往外发东西，没确认过的话不该指挥它。
// 也不写机器名、工作时间、工程线和经历，那些只对主会话有用。budget 是字节上限，小于等于 0 时不限。
func (s *Store) OperatorPin(budget int) string {
	s.mu.Lock()
	s.syncLocked()
	now := s.clock()
	set := s.pinSet(now)
	s.mu.Unlock()
	w := pinWriter{budget: budget}
	w.group("Said by the operator:\n", pinLines(set.said, now))
	w.group("Preferences:\n", pinLines(set.prefs, now))
	var system []string
	if joined := strings.Join(nonEmpty(set.os, set.zone), ", "); joined != "" {
		system = append(system, "system: "+joined)
	}
	if set.git != "" {
		system = append(system, "git name: "+set.git)
	}
	w.group("", system)
	return w.String()
}

// staleAfter 之后，写进提示词的一条会带上它的日子，让模型知道这是多久以前的话。
const staleAfter = 90 * 24 * time.Hour

// pinLines 是写进提示词的每一行。超过 staleAfter 的带上「as of 日子」。日子是绝对的，不会每天变，前缀缓存不受影响。
func pinLines(items []Item, now time.Time) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		line := it.Text
		if at := updatedOf(it); !at.IsZero() && now.Sub(at) > staleAfter {
			line += " (as of " + at.Local().Format(dateLayout) + ")"
		}
		out = append(out, line)
	}
	return out
}

func texts(items []Item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Text)
	}
	return out
}

// pinWriter 按字节预算写画像。
type pinWriter struct {
	b      strings.Builder
	budget int
}

func (w *pinWriter) fits(n int) bool {
	return w.budget <= 0 || w.b.Len()+n <= w.budget
}

// group 写一组，返回写进去的那几行的下标。放不下的一行跳过，后面更短的仍可以写进来；一行都放不下时连标题也不写。
func (w *pinWriter) group(title string, lines []string) []int {
	opened := false
	var wrote []int
	for i, line := range lines {
		row := "- " + line + "\n"
		need := len(row)
		if !opened {
			need += len(title)
		}
		if !w.fits(need) {
			continue
		}
		if !opened {
			w.b.WriteString(title)
			opened = true
		}
		w.b.WriteString(row)
		wrote = append(wrote, i)
	}
	return wrote
}

func (w *pinWriter) String() string { return w.b.String() }
