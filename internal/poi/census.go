package poi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Census 是第一封简报用的桌面清点。只看目录名、清单里的项目名、文件大小和修改时间。
// 不读正文，不读环境变量文件、令牌和浏览记录。
type Census struct {
	User      string       `json:"user,omitempty"`
	Host      string       `json:"host,omitempty"`
	OS        string       `json:"os,omitempty"`
	Zone      string       `json:"zone,omitempty"`
	Git       string       `json:"git,omitempty"`
	Hours     string       `json:"hours,omitempty"`
	Weekday   bool         `json:"weekday,omitempty"`
	Lines     []CensusLine `json:"lines,omitempty"`
	Finished  []CensusItem `json:"finished,omitempty"`
	Empty     []CensusItem `json:"empty,omitempty"`
	Place     []CensusItem `json:"place,omitempty"`
	Archive   []CensusItem `json:"archive,omitempty"`
	Projects  int          `json:"projects,omitempty"`
	Downloads int          `json:"downloads,omitempty"`
}

// CensusLine 是一条能对上清单文件的工程线。State 只表示最近有没有改、有没有 0 字节文件。
type CensusLine struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	Evidence string `json:"evidence,omitempty"`
	When     string `json:"when,omitempty"`
}

// CensusItem 是桌面或下载里的一个文件名。Kind 是 empty、installer、deliverable、match。
type CensusItem struct {
	Name string `json:"name"`
	Line string `json:"line,omitempty"`
	Kind string `json:"kind,omitempty"`
}

type projectHit struct {
	CensusLine
	at    time.Time
	times []time.Time
	empty string
}

var moduleLine = regexp.MustCompile(`(?m)^module\s+(\S+)`)
var quotedName = regexp.MustCompile(`(?m)^name\s*=\s*"([^"]+)"`)

// ScanCensus 清点这些目录的第一层。工程只认 go.mod、package.json、Cargo.toml、pyproject.toml 里的名字。
func ScanCensus(roots []string, now time.Time) Census {
	var projects []projectHit
	var loose []looseFile
	var times []time.Time
	downloads := 0
	for _, root := range roots {
		kind := rootKind(root)
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			raw := entry.Name()
			if skipEntry(raw) || secretName(raw) {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			path := filepath.Join(root, raw)
			if entry.IsDir() {
				if hit, ok := readProject(path, raw, info.ModTime(), now); ok {
					projects = append(projects, hit)
					times = append(times, weekdayTimes(hit.times)...)
				}
				continue
			}
			if kind == "downloads" && info.ModTime().After(now.Add(-45*24*time.Hour)) {
				downloads++
			}
			if info.ModTime().After(now.Add(-21 * 24 * time.Hour)) {
				times = append(times, weekdayTimes([]time.Time{info.ModTime()})...)
			}
			if kind == "desktop" || kind == "downloads" {
				loose = append(loose, looseFile{name: raw, size: info.Size(), at: info.ModTime()})
			}
		}
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].at.After(projects[j].at) })
	out := Census{
		User:      localUser(),
		Host:      machineHost(),
		OS:        machineOS(),
		Zone:      machineZone(),
		Git:       gitUserName(),
		Hours:     hoursLabel(times),
		Weekday:   now.Weekday() == time.Saturday || now.Weekday() == time.Sunday,
		Projects:  len(projects),
		Downloads: downloads,
	}
	lines := projects
	if len(lines) > 3 {
		lines = lines[:3]
	}
	for _, hit := range lines {
		out.Lines = append(out.Lines, hit.CensusLine)
	}
	for _, hit := range projects {
		if hit.empty == "" || len(out.Empty) >= 4 {
			continue
		}
		out.Empty = append(out.Empty, CensusItem{Name: clipName(hit.empty), Line: hit.Name, Kind: "empty"})
	}
	for _, file := range loose {
		stem := strings.TrimSuffix(file.name, filepath.Ext(file.name))
		if secretName(stem) {
			continue
		}
		switch {
		case file.size == 0 && len(out.Empty) < 4:
			out.Empty = append(out.Empty, CensusItem{Name: clipName(stem), Kind: "empty"})
		case installerExt(file.name) && len(out.Archive) < 6:
			out.Archive = append(out.Archive, CensusItem{Name: clipName(file.name), Kind: "installer"})
		case deliverable(file, now) && len(out.Finished) < 4:
			out.Finished = append(out.Finished, CensusItem{Name: clipName(stem), Kind: "deliverable"})
		default:
			if line := matchingLine(stem, out.Lines); line != "" && len(out.Place) < 6 {
				out.Place = append(out.Place, CensusItem{Name: clipName(stem), Line: line, Kind: "match"})
			}
		}
	}
	return out
}

type looseFile struct {
	name string
	size int64
	at   time.Time
}

func readProject(dir, dirName string, dirTime, now time.Time) (projectHit, bool) {
	manifest := ""
	for _, file := range []string{"go.mod", "package.json", "Cargo.toml", "pyproject.toml"} {
		st, err := os.Stat(filepath.Join(dir, file))
		if err != nil || st.IsDir() || st.Size() > 1<<20 {
			continue
		}
		manifest = file
		break
	}
	if manifest == "" {
		return projectHit{}, false
	}
	name := manifestName(filepath.Join(dir, manifest), manifest)
	if name == "" || genericName(name) {
		name = dirName
	}
	name = clipName(name)
	if name == "" || secretName(name) {
		return projectHit{}, false
	}
	newest := dirTime
	evidence := manifest
	var times []time.Time
	empty := ""
	children, _ := os.ReadDir(dir)
	for _, child := range children {
		if child.IsDir() || skipEntry(child.Name()) || secretName(child.Name()) {
			continue
		}
		info, err := child.Info()
		if err != nil {
			continue
		}
		if info.Size() == 0 && empty == "" {
			empty = child.Name()
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
			evidence = child.Name()
		}
		if info.ModTime().After(now.Add(-21 * 24 * time.Hour)) {
			times = append(times, info.ModTime())
		}
		if len(times) >= 30 {
			break
		}
	}
	if newest.Before(now.Add(-90 * 24 * time.Hour)) {
		return projectHit{}, false
	}
	state := "quiet"
	if empty != "" {
		state = "stalled"
		evidence = empty
	} else if newest.After(now.Add(-14 * 24 * time.Hour)) {
		state = "moving"
	}
	return projectHit{
		CensusLine: CensusLine{Name: name, State: state, Evidence: clipName(evidence), When: newest.Format("2006-01-02")},
		at:         newest,
		times:      times,
		empty:      empty,
	}, true
}

func manifestName(path, file string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, 8192)
	n, _ := f.Read(buf)
	body := string(buf[:n])
	switch file {
	case "go.mod":
		if m := moduleLine.FindStringSubmatch(body); len(m) == 2 {
			return lastSegment(m[1])
		}
	case "package.json":
		var doc struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(buf[:n], &doc) == nil {
			return lastSegment(doc.Name)
		}
	default:
		if m := quotedName.FindStringSubmatch(body); len(m) == 2 {
			return lastSegment(m[1])
		}
	}
	return ""
}

func genericName(name string) bool {
	switch strings.ToLower(name) {
	case "src", "app", "main", "project", "repo", "code", "web", "frontend", "backend":
		return true
	default:
		return false
	}
}

func rootKind(path string) string {
	switch strings.ToLower(filepath.Base(path)) {
	case "desktop", "documents", "downloads":
		return strings.ToLower(filepath.Base(path))
	default:
		return "other"
	}
}

func installerExt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".exe", ".msi", ".dmg", ".pkg", ".zip", ".rar", ".7z", ".iso":
		return true
	default:
		return false
	}
}

func deliverable(file looseFile, now time.Time) bool {
	if file.size < 8<<10 || !file.at.Before(now.Add(-48*time.Hour)) {
		return false
	}
	switch strings.ToLower(filepath.Ext(file.name)) {
	case ".html", ".htm", ".mp4", ".pdf", ".docx", ".pptx", ".mov":
		return true
	default:
		return false
	}
}

func matchingLine(stem string, lines []CensusLine) string {
	file := strings.ToLower(stem)
	for _, line := range lines {
		name := strings.ToLower(line.Name)
		if utf8.RuneCountInString(name) < 2 {
			continue
		}
		if strings.Contains(file, name) {
			return line.Name
		}
	}
	return ""
}

func weekdayTimes(in []time.Time) []time.Time {
	var out []time.Time
	for _, t := range in {
		if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
			continue
		}
		out = append(out, t)
	}
	return out
}

func hoursLabel(times []time.Time) string {
	if len(times) < 6 {
		return ""
	}
	mins := make([]int, len(times))
	for i, t := range times {
		mins[i] = t.Hour()*60 + t.Minute()
	}
	sort.Ints(mins)
	lo := mins[len(mins)/10]
	hi := mins[len(mins)-1-len(mins)/10]
	if hi-lo < 60 {
		return ""
	}
	return formatClock(lo) + "–" + formatClock(hi)
}

func formatClock(min int) string {
	return strings.TrimSpace(strings.ReplaceAll(
		time.Date(0, 1, 1, min/60, min%60, 0, 0, time.UTC).Format("15:04"),
		" ",
		"",
	))
}

func machineHost() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	name = clipName(strings.TrimSpace(name))
	if secretName(name) {
		return ""
	}
	return name
}

func machineOS() string {
	if name := productName(); name != "" && !secretName(name) {
		return clipName(name)
	}
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	default:
		return runtime.GOOS
	}
}

func machineZone() string {
	name := time.Now().Location().String()
	if name == "" || name == "Local" || secretName(name) {
		return ""
	}
	return clipName(name)
}

func gitUserName() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	f, err := os.Open(filepath.Join(home, ".gitconfig"))
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, 64<<10)
	n, _ := f.Read(buf)
	section := ""
	for _, line := range strings.Split(string(buf[:n]), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.Trim(line, "[]"))
			if i := strings.IndexAny(section, " \t"); i >= 0 {
				section = section[:i]
			}
			continue
		}
		if section != "user" {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(strings.ToLower(key)) != "name" {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if val == "" || secretName(val) || utf8.RuneCountInString(val) > 40 {
			return ""
		}
		return val
	}
	return ""
}

func localUser() string {
	name := os.Getenv("USERNAME")
	if name == "" {
		name = os.Getenv("USER")
	}
	name = clipName(name)
	if name == "" || secretName(name) || strings.EqualFold(name, "administrator") {
		return ""
	}
	return name
}
