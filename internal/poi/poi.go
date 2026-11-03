// Package poi 维护一份本地兴趣库。名字来自已生效的记忆，以及桌面、文档、下载目录的表层文件名。
// 不读取文件正文。权重按天衰减，名字集合变化后由调用方决定要不要推进会话。
package poi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Point 是兴趣库里的一个点。Links 记下同一段记忆里一起出现的其他点。
type Point struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Kind     string    `json:"kind"`
	Weight   int       `json:"weight"`
	Evidence string    `json:"evidence,omitempty"`
	Links    []string  `json:"links,omitempty"`
	Updated  time.Time `json:"updated"`
}

// Observation 是这一轮看见的一个名字。Evidence 只标来源，不带路径。
type Observation struct {
	Name     string
	Kind     string
	Evidence string
}

// Snapshot 是刷新后的库，以及相对上次已宣布的名字集合多了什么、少了什么。
type Snapshot struct {
	Points  []Point  `json:"points"`
	Added   []string `json:"added"`
	Dropped []string `json:"dropped"`
	Sig     string   `json:"sig"`
	Pending bool     `json:"pending"`
	Census  Census   `json:"census,omitempty"`
}

// Store 把兴趣库写在 dir/poi.json。
type Store struct {
	path      string
	points    []Point
	announced []string
	decayedOn string
}

type diskBody struct {
	Points    []Point  `json:"points"`
	Announced []string `json:"announced"`
	DecayedOn string   `json:"decayedOn"`
}

// Open 打开 dir 里的兴趣库。目录不存在时创建。文件还不存在时从空库开始。
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "poi.json")}
	_ = s.load()
	return s, nil
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var body diskBody
	if err := json.Unmarshal(b, &body); err != nil {
		return err
	}
	s.points = body.Points
	s.announced = body.Announced
	s.decayedOn = body.DecayedOn
	return nil
}

func (s *Store) flush() error {
	body := diskBody{Points: s.points, Announced: s.announced, DecayedOn: s.decayedOn}
	b, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Refresh 用这一轮看见的名字更新库。同一天里没再看见的点不减权重，跨天才减。blobs 用来把同一段文字里的点连起来。
func (s *Store) Refresh(obs []Observation, blobs []string, now time.Time) Snapshot {
	day := now.Format("2006-01-02")
	decay := s.decayedOn != day
	seen := map[string]Observation{}
	var order []string
	for _, item := range obs {
		name := clipName(item.Name)
		kind := strings.TrimSpace(item.Kind)
		if name == "" || kind == "" {
			continue
		}
		id := kind + ":" + strings.ToLower(name)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = Observation{Name: name, Kind: kind, Evidence: item.Evidence}
		order = append(order, id)
	}
	byID := map[string]Point{}
	for _, pt := range s.points {
		byID[pt.ID] = pt
	}
	var next []Point
	for _, id := range order {
		item := seen[id]
		pt, ok := byID[id]
		if !ok {
			pt = Point{ID: id, Name: item.Name, Kind: item.Kind, Weight: 1}
		} else if decay && pt.Weight < 12 {
			pt.Weight++
		}
		pt.Name = item.Name
		pt.Kind = item.Kind
		pt.Evidence = item.Evidence
		pt.Updated = now.UTC()
		next = append(next, pt)
		delete(byID, id)
	}
	for _, pt := range s.points {
		if _, ok := seen[pt.ID]; ok {
			continue
		}
		if decay {
			pt.Weight--
		}
		if pt.Weight <= 0 {
			continue
		}
		pt.Updated = now.UTC()
		next = append(next, pt)
	}
	linkPoints(next, blobs)
	sort.Slice(next, func(i, j int) bool {
		if next[i].Weight != next[j].Weight {
			return next[i].Weight > next[j].Weight
		}
		return next[i].Name < next[j].Name
	})
	s.points = next
	if decay {
		s.decayedOn = day
	}
	_ = s.flush()
	return s.snapshot()
}

// Ack 把当前名字集合记成已经推进过。sig 对不上当前库时不改。
func (s *Store) Ack(sig string) bool {
	snap := s.snapshot()
	if sig == "" || sig != snap.Sig {
		return false
	}
	s.announced = namesOf(s.points)
	_ = s.flush()
	return true
}

func (s *Store) snapshot() Snapshot {
	names := namesOf(s.points)
	old := map[string]bool{}
	for _, name := range s.announced {
		old[name] = true
	}
	cur := map[string]bool{}
	for _, name := range names {
		cur[name] = true
	}
	var added, dropped []string
	for _, name := range names {
		if !old[name] {
			added = append(added, name)
		}
	}
	for _, name := range s.announced {
		if !cur[name] {
			dropped = append(dropped, name)
		}
	}
	sort.Strings(added)
	sort.Strings(dropped)
	sig := strings.Join(names, "\n")
	pending := len(added) > 0 || len(dropped) > 0
	if len(s.announced) == 0 && len(names) > 0 {
		pending = true
	}
	out := append([]Point(nil), s.points...)
	return Snapshot{Points: out, Added: added, Dropped: dropped, Sig: sig, Pending: pending}
}

func namesOf(points []Point) []string {
	names := make([]string, 0, len(points))
	seen := map[string]bool{}
	for _, pt := range points {
		if pt.Name == "" || seen[pt.Name] {
			continue
		}
		seen[pt.Name] = true
		names = append(names, pt.Name)
	}
	sort.Strings(names)
	return names
}

func linkPoints(points []Point, blobs []string) {
	for i := range points {
		points[i].Links = nil
	}
	if len(blobs) == 0 || len(points) < 2 {
		return
	}
	for _, blob := range blobs {
		text := strings.ToLower(blob)
		if strings.TrimSpace(text) == "" {
			continue
		}
		var hit []string
		for _, pt := range points {
			if utf8.RuneCountInString(pt.Name) < 2 {
				continue
			}
			if strings.Contains(text, strings.ToLower(pt.Name)) {
				hit = append(hit, pt.ID)
			}
		}
		if len(hit) < 2 {
			continue
		}
		if len(hit) > 6 {
			hit = hit[:6]
		}
		for i := range points {
			for _, id := range hit {
				if points[i].ID == id {
					points[i].Links = mergeIDs(points[i].Links, hit, points[i].ID)
				}
			}
		}
	}
}

func mergeIDs(have, add []string, self string) []string {
	seen := map[string]bool{}
	for _, id := range have {
		seen[id] = true
	}
	for _, id := range add {
		if id == self || seen[id] {
			continue
		}
		seen[id] = true
		have = append(have, id)
		if len(have) >= 4 {
			break
		}
	}
	return have
}

func clipName(text string) string {
	text = strings.TrimSpace(strings.Split(text, "\n")[0])
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) <= 24 {
		return text
	}
	runes := []rune(text)
	return string(runes[:24])
}

// DefaultRoots 是用户主目录下桌面、文档和下载的表层。OneDrive 里的同名目录也算。不存在的路径省掉。
func DefaultRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	var roots []string
	for _, base := range []string{home, filepath.Join(home, "OneDrive")} {
		for _, name := range []string{"Desktop", "Documents", "Downloads"} {
			p := filepath.Join(base, name)
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				roots = append(roots, p)
			}
		}
	}
	return roots
}

// ScanRoots 只读这些目录的第一层名字。文件只留近 45 天改过的，正文不读。
func ScanRoots(roots []string, now time.Time) []Observation {
	cutoff := now.Add(-45 * 24 * time.Hour)
	var dirs []Observation
	type fileHit struct {
		obs Observation
		at  time.Time
	}
	var files []fileHit
	seen := map[string]bool{}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			raw := entry.Name()
			if skipEntry(raw) {
				continue
			}
			name := raw
			if !entry.IsDir() {
				name = strings.TrimSuffix(raw, filepath.Ext(raw))
			}
			name = clipName(name)
			if name == "" || seen[strings.ToLower(name)] {
				continue
			}
			if entry.IsDir() {
				seen[strings.ToLower(name)] = true
				dirs = append(dirs, Observation{Name: name, Kind: "desk", Evidence: "desk"})
				continue
			}
			info, err := entry.Info()
			if err != nil || info.ModTime().Before(cutoff) {
				continue
			}
			seen[strings.ToLower(name)] = true
			files = append(files, fileHit{obs: Observation{Name: name, Kind: "desk", Evidence: "desk"}, at: info.ModTime()})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].at.After(files[j].at) })
	out := dirs
	for _, file := range files {
		out = append(out, file.obs)
		if len(out) >= 40 {
			break
		}
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

func skipEntry(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" || strings.HasPrefix(lower, ".") {
		return true
	}
	switch lower {
	case "desktop.ini", "thumbs.db", "node_modules", "$recycle.bin":
		return true
	}
	return false
}
