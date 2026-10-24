package remote

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestMethodCatalogMatchesClient(t *testing.T) {
	got := renderCatalog()
	path := clientCatalogPath(t)
	if os.Getenv("YOYO_WRITE_METHODS") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	have, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(have) != got {
		t.Fatalf("handheld/lib/methods.g.dart drifted from the Go method table; regenerate with YOYO_WRITE_METHODS=1")
	}
}

func clientCatalogPath(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd + "/../../handheld/lib/methods.g.dart"
}

type catalogMethod struct {
	name    string
	group   string
	confirm bool
	denied  bool
	fields  []string
}

func renderCatalog() string {
	methods := collectMethods()
	var b strings.Builder
	b.WriteString("// Generated from the Go method table. Regenerate with YOYO_WRITE_METHODS=1.\n\n")
	b.WriteString("class RemoteMethod {\n")
	b.WriteString("  const RemoteMethod(this.name, this.group, this.confirm, this.denied, this.fields);\n")
	b.WriteString("  final String name;\n")
	b.WriteString("  final String group;\n")
	b.WriteString("  final bool confirm;\n")
	b.WriteString("  final bool denied;\n")
	b.WriteString("  final List<String> fields;\n")
	b.WriteString("}\n\n")
	b.WriteString("const remoteMethods = <RemoteMethod>[\n")
	for _, m := range methods {
		b.WriteString("  RemoteMethod('")
		b.WriteString(m.name)
		b.WriteString("', '")
		b.WriteString(m.group)
		b.WriteString("', ")
		b.WriteString(boolWord(m.confirm))
		b.WriteString(", ")
		b.WriteString(boolWord(m.denied))
		b.WriteString(", [")
		for i, f := range m.fields {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString("'")
			b.WriteString(f)
			b.WriteString("'")
		}
		b.WriteString("]),\n")
	}
	b.WriteString("];\n")
	return b.String()
}

func boolWord(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func collectMethods() []catalogMethod {
	blobs := []string{"../api/rpc.go", "../app/video.go", "node.go"}
	byName := map[string]catalogMethod{}
	lineRe := regexp.MustCompile(`(?m)^\tcase\s+(.+)$`)
	quoteRe := regexp.MustCompile(`"([^"]+)"`)
	for _, path := range blobs {
		text, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		src := string(text)
		indexes := lineRe.FindAllStringSubmatchIndex(src, -1)
		for i, idx := range indexes {
			names := quoteRe.FindAllStringSubmatch(src[idx[2]:idx[3]], -1)
			end := len(src)
			if i+1 < len(indexes) {
				end = indexes[i+1][0]
			}
			if j := strings.Index(src[idx[1]:end], "\n\tdefault:"); j >= 0 {
				end = idx[1] + j
			}
			body := src[idx[1]:end]
			fields := fieldsIn(body)
			for _, name := range names {
				method := name[1]
				if !methodName(method) {
					continue
				}
				if prev, ok := byName[method]; ok {
					fields = union(prev.fields, fields)
				}
				byName[method] = catalogMethod{
					name:    method,
					group:   methodGroup(method),
					confirm: NeedsConfirm(method),
					denied:  PhoneDenied(method),
					fields:  fields,
				}
			}
		}
	}
	out := make([]catalogMethod, 0, len(byName))
	for _, m := range byName {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func methodName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

func methodGroup(name string) string {
	switch {
	case strings.HasPrefix(name, "thread.") || strings.HasPrefix(name, "turn.") || strings.HasPrefix(name, "trajectory.") || strings.HasPrefix(name, "trace.") || strings.HasPrefix(name, "live.") || strings.HasPrefix(name, "session.") || strings.HasPrefix(name, "approvals.") || strings.HasPrefix(name, "approval.") || strings.HasPrefix(name, "spill."):
		return "会话"
	case strings.HasPrefix(name, "video.") || strings.HasPrefix(name, "drama.") || strings.HasPrefix(name, "media.") || strings.HasPrefix(name, "canvas."):
		return "视频"
	case strings.HasPrefix(name, "harness.") || strings.HasPrefix(name, "evolve.") || strings.HasPrefix(name, "packs.") || strings.HasPrefix(name, "skills.") || strings.HasPrefix(name, "eval."):
		return "实验室"
	case strings.HasPrefix(name, "provider.") || strings.HasPrefix(name, "key.") || strings.HasPrefix(name, "config.") || strings.HasPrefix(name, "skins.") || strings.HasPrefix(name, "update.") || name == "app.pause.set" || strings.HasPrefix(name, "connectors.") || strings.HasPrefix(name, "connections."):
		return "设置"
	case strings.HasPrefix(name, "workspace.") || strings.HasPrefix(name, "pages.") || strings.HasPrefix(name, "archive.") || strings.HasPrefix(name, "inbox.") || strings.HasPrefix(name, "memory.") || strings.HasPrefix(name, "schedule.") || strings.HasPrefix(name, "projects.") || strings.HasPrefix(name, "context."):
		return "工作区"
	case strings.HasPrefix(name, "idol.") || strings.HasPrefix(name, "voice.") || strings.HasPrefix(name, "personal.") || strings.HasPrefix(name, "phone."):
		return "陪伴"
	case strings.HasPrefix(name, "remote."):
		return "配对"
	default:
		return "其他"
	}
}

func fieldsIn(body string) []string {
	tag := regexp.MustCompile(`json:"([^"]+)"`)
	strArg := regexp.MustCompile(`\bstr\("([^"]+)"\)`)
	seen := map[string]bool{}
	var out []string
	add := func(raw string) {
		name := strings.Split(raw, ",")[0]
		if name == "" || name == "-" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	for _, m := range tag.FindAllStringSubmatch(body, -1) {
		add(m[1])
	}
	for _, m := range strArg.FindAllStringSubmatch(body, -1) {
		add(m[1])
	}
	sort.Strings(out)
	return out
}

func union(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range [][]string{a, b} {
		for _, field := range list {
			if seen[field] {
				continue
			}
			seen[field] = true
			out = append(out, field)
		}
	}
	sort.Strings(out)
	return out
}
