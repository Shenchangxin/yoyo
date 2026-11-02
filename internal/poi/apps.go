package poi

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// AppRoot 是一个有价值的本机应用，或最近使用过的文件名目录。Kind 决定怎么读：code、obsidian、recent、presence。
// presence 只记下应用在，不打开里面的文件。
type AppRoot struct {
	Name string
	Dir  string
	Kind string
}

var folderURI = regexp.MustCompile(`"(?:folderUri|folder)"\s*:\s*"([^"\\]+)"`)

// DefaultAppRoots 找出本机已经在用的编辑器、笔记和办公应用。浏览器和飞书只记「在用」，不读历史、Cookie 和令牌。
func DefaultAppRoots() []AppRoot {
	roaming := os.Getenv("APPDATA")
	local := os.Getenv("LOCALAPPDATA")
	var roots []AppRoot
	add := func(name, base, rel, kind string) {
		if base == "" {
			return
		}
		dir := filepath.Join(base, rel)
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			return
		}
		roots = append(roots, AppRoot{Name: name, Dir: dir, Kind: kind})
	}
	add("Cursor", roaming, "Cursor", "code")
	add("Code", roaming, "Code", "code")
	add("Windsurf", roaming, "Windsurf", "code")
	add("Trae", roaming, "Trae", "code")
	add("Obsidian", roaming, "obsidian", "obsidian")
	add("飞书", roaming, "LarkShell", "presence")
	add("Slack", roaming, "Slack", "presence")
	add("Notion", roaming, "Notion", "presence")
	add("Figma", roaming, "Figma", "presence")
	add("Discord", roaming, "discord", "presence")
	add("Typora", roaming, "Typora", "presence")
	add("GitHub Desktop", roaming, "GitHub Desktop", "presence")
	add("Docker", roaming, "Docker", "presence")
	add("Linear", roaming, "Linear", "presence")
	add("Chrome", local, filepath.Join("Google", "Chrome"), "presence")
	add("Edge", local, filepath.Join("Microsoft", "Edge"), "presence")
	if roaming != "" {
		add("最近的文件", roaming, filepath.Join("Microsoft", "Windows", "Recent"), "recent")
	}
	return roots
}

// ScanAppRoots 从这些目录收名字。code 和 obsidian 读工作区或库的名字。recent 只读快捷方式的文件名。正文、Cookie 和令牌不读。
func ScanAppRoots(roots []AppRoot, now time.Time) []Observation {
	var out []Observation
	seen := map[string]bool{}
	push := func(name, kind, evidence string) {
		name = clipName(name)
		if name == "" || secretName(name) {
			return
		}
		key := kind + ":" + strings.ToLower(name)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, Observation{Name: name, Kind: kind, Evidence: evidence})
	}
	for _, root := range roots {
		if strings.TrimSpace(root.Dir) == "" {
			continue
		}
		switch root.Kind {
		case "code":
			push(root.Name, "app", "app")
			for _, name := range codeNames(root.Dir, now) {
				push(name, "trace", "trace")
			}
		case "obsidian":
			push(root.Name, "app", "app")
			for _, name := range obsidianNames(root.Dir) {
				push(name, "trace", "trace")
			}
		case "recent":
			for _, name := range recentNames(root.Dir, now) {
				push(name, "trace", "trace")
			}
		default:
			push(root.Name, "app", "app")
		}
		if len(out) >= 48 {
			break
		}
	}
	if len(out) > 48 {
		out = out[:48]
	}
	return out
}

func codeNames(root string, now time.Time) []string {
	var names []string
	names = append(names, uriNames(readHead(filepath.Join(root, "User", "globalStorage", "storage.json")))...)
	ws := filepath.Join(root, "User", "workspaceStorage")
	entries, err := os.ReadDir(ws)
	if err != nil {
		return uniqueNames(names)
	}
	cutoff := now.Add(-45 * 24 * time.Hour)
	type hit struct {
		name string
		at   time.Time
	}
	var hits []hit
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(ws, entry.Name(), "workspace.json")
		info, err := os.Stat(path)
		if err != nil || info.ModTime().Before(cutoff) || info.Size() > 64*1024 {
			continue
		}
		for _, name := range uriNames(readHead(path)) {
			hits = append(hits, hit{name: name, at: info.ModTime()})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].at.After(hits[j].at) })
	for _, hit := range hits {
		names = append(names, hit.name)
		if len(names) >= 12 {
			break
		}
	}
	return uniqueNames(names)
}

func obsidianNames(root string) []string {
	b := readHead(filepath.Join(root, "obsidian.json"))
	if len(b) == 0 {
		return nil
	}
	var body struct {
		Vaults map[string]struct {
			Path string `json:"path"`
		} `json:"vaults"`
	}
	if json.Unmarshal(b, &body) != nil {
		return nil
	}
	var names []string
	for _, vault := range body.Vaults {
		names = append(names, lastSegment(vault.Path))
	}
	sort.Strings(names)
	return uniqueNames(names)
}

func recentNames(dir string, now time.Time) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	cutoff := now.Add(-45 * 24 * time.Hour)
	type hit struct {
		name string
		at   time.Time
	}
	var hits []hit
	for _, entry := range entries {
		if entry.IsDir() || skipEntry(entry.Name()) {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".lnk") {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().Before(cutoff) {
			continue
		}
		stem := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		hits = append(hits, hit{name: stem, at: info.ModTime()})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].at.After(hits[j].at) })
	var names []string
	for _, hit := range hits {
		names = append(names, hit.name)
		if len(names) >= 12 {
			break
		}
	}
	return names
}

func uriNames(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var names []string
	for _, match := range folderURI.FindAllSubmatch(raw, 24) {
		names = append(names, lastSegment(string(match[1])))
	}
	return names
}

func lastSegment(raw string) string {
	if decoded, err := url.PathUnescape(raw); err == nil {
		raw = decoded
	}
	raw = strings.TrimPrefix(raw, "file://")
	raw = strings.ReplaceAll(raw, "\\", "/")
	raw = strings.TrimRight(raw, "/")
	if i := strings.LastIndex(raw, "/"); i >= 0 {
		raw = raw[i+1:]
	}
	return raw
}

func uniqueNames(names []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, name := range names {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, name)
	}
	return out
}

func secretName(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "secret"),
		strings.Contains(lower, "token"),
		strings.Contains(lower, "password"),
		strings.Contains(lower, "cookie"),
		strings.Contains(lower, "credential"):
		return true
	default:
		return false
	}
}

func readHead(path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		return nil
	}
	const limit = 8 << 20
	n := st.Size()
	if n > limit {
		n = limit
	}
	b, err := io.ReadAll(io.LimitReader(f, n))
	if err != nil {
		return nil
	}
	return b
}
