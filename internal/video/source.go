package video

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/office"
)

const (
	sliceWindow     = 1800
	sliceOverlap    = 200
	maxReadWindow   = 2400
	seriesRuneMin   = 2000
	chapterMinRunes = 400
	episodeWindow   = 4000
	maxEpisodeRunes = 8000
	minEpisodeRunes = 600
	maxFilmSeconds  = 90
	minFilmSeconds  = 45
	peekRunes       = 180
)

var chapterRe = regexp.MustCompile(`(?mi)(?:^|\n)[ \t]*((第[0-9一二三四五六七八九十百千零两]+[章节回部卷])|(Chapter\s+\d+))`)

func LooksLikeSeries(text string) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(text))
	if n >= seriesRuneMin {
		return true
	}
	return len(chapterStarts(text)) >= 2 && n >= chapterMinRunes
}

func (e *Engine) IngestSource(dramaID, text, title string) (Source, error) {
	if _, err := e.GetDrama(dramaID); err != nil {
		return Source{}, err
	}
	text = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(text), "\r\n", "\n"), "\r", "\n")
	if text == "" {
		return Source{}, fmt.Errorf("source text is empty")
	}
	if plan, err := e.GetPlan(dramaID); err == nil && plan.Status == "committed" {
		return Source{}, fmt.Errorf("this series already has a committed map — start a new series")
	}
	now := Now()
	src := Source{ID: NewID(), DramaID: dramaID, Title: strings.TrimSpace(title), CreatedAt: now, UpdatedAt: now}
	if old, err := e.GetSource(dramaID); err == nil {
		src.ID = old.ID
		src.CreatedAt = old.CreatedAt
	}
	hash, err := e.putSourceText(src.ID, text)
	if err != nil {
		return Source{}, err
	}
	src.Hash = hash
	src.RuneCount = utf8.RuneCountInString(text)
	src.Slices = buildSlices(text)
	if err := e.putDoc(colSources, src.ID, src); err != nil {
		return Source{}, err
	}
	return src, nil
}

func (e *Engine) IngestSourceFile(dramaID, path string) (Source, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Source{}, err
	}
	text := string(raw)
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".markdown":
		// already text
	case ".docx", ".pdf", ".pptx", ".xlsx":
		extracted, err := queryOfficeText(path)
		if err != nil {
			return Source{}, err
		}
		text = extracted
	default:
		if looksBinary(raw) {
			return Source{}, fmt.Errorf("unsupported source file %s", ext)
		}
	}
	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return e.IngestSource(dramaID, text, title)
}

func (e *Engine) GetSource(dramaID string) (Source, error) {
	for _, s := range loadCol[Source](e, colSources) {
		if s.DramaID == dramaID {
			return s, nil
		}
	}
	return Source{}, fmt.Errorf("source not found")
}

func (e *Engine) SourceText(src Source) (string, error) {
	p := filepath.Join(e.Dir, "sources", src.ID+".txt")
	if b, err := os.ReadFile(p); err == nil && len(b) > 0 {
		return string(b), nil
	}
	if e.CAS != nil && src.Hash != "" {
		b, err := e.CAS.GetRaw(src.Hash)
		if err == nil {
			return string(b), nil
		}
	}
	return "", fmt.Errorf("source text missing")
}

func (e *Engine) ReadSource(dramaID string, offset, length, sliceIndex int, query string) (map[string]any, error) {
	src, err := e.GetSource(dramaID)
	if err != nil {
		return nil, err
	}
	text, err := e.SourceText(src)
	if err != nil {
		return nil, err
	}
	runes := []rune(text)
	n := len(runes)
	if sliceIndex > 0 && sliceIndex <= len(src.Slices) {
		sl := src.Slices[sliceIndex-1]
		offset = sl.Start
		if length <= 0 {
			length = sl.End - sl.Start
		}
	}
	if q := strings.TrimSpace(query); q != "" {
		idx := strings.Index(text, q)
		if idx < 0 {
			idx = strings.Index(strings.ToLower(text), strings.ToLower(q))
		}
		if idx >= 0 {
			offset = utf8.RuneCountInString(text[:idx])
			if length <= 0 {
				length = sliceWindow
			}
		}
	}
	if offset < 0 {
		offset = 0
	}
	if length <= 0 {
		length = sliceWindow
	}
	if length > maxReadWindow {
		length = maxReadWindow
	}
	end := offset + length
	if end > n {
		end = n
	}
	if offset > n {
		offset = n
	}
	return map[string]any{
		"drama_id":   dramaID,
		"source_id":  src.ID,
		"offset":     offset,
		"end":        end,
		"rune_count": n,
		"text":       string(runes[offset:end]),
		"truncated":  end < n || offset > 0,
		"slices":     src.Slices,
	}, nil
}

func (e *Engine) putSourceText(id, text string) (string, error) {
	dir := filepath.Join(e.Dir, "sources")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, id+".txt"), []byte(text), 0o644); err != nil {
		return "", err
	}
	if e.CAS == nil {
		return "", nil
	}
	return e.CAS.PutRaw([]byte(text))
}

func buildSlices(text string) []SourceSlice {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return nil
	}
	var out []SourceSlice
	start := 0
	i := 0
	for start < n {
		end := start + sliceWindow
		if end > n {
			end = n
		}
		heading := headingAt(runes, start)
		out = append(out, SourceSlice{
			Index: i + 1, Start: start, End: end, Heading: heading,
			Peek: string(runes[start:minInt(n, start+peekRunes)]),
		})
		i++
		if end >= n {
			break
		}
		start = end - sliceOverlap
		if start <= out[len(out)-1].Start {
			start = end
		}
	}
	return out
}

func headingAt(runes []rune, start int) string {
	if start < 0 || start >= len(runes) {
		return ""
	}
	line := string(runes[start:minInt(len(runes), start+80)])
	if i := strings.IndexAny(line, "\n"); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	if chapterRe.MatchString("\n" + line) {
		return line
	}
	return ""
}

func CandidateSeams(text string) []int {
	runes := []rune(text)
	n := len(runes)
	seen := map[int]bool{0: true, n: true}
	add := func(i int) {
		if i < 0 {
			i = 0
		}
		if i > n {
			i = n
		}
		seen[i] = true
	}
	for _, m := range chapterRe.FindAllStringIndex(text, -1) {
		add(utf8.RuneCountInString(text[:m[0]]))
		// skip the leading newline captured by the regex
		if m[0] < len(text) && (text[m[0]] == '\n' || text[m[0]] == '\r') {
			add(utf8.RuneCountInString(text[:m[0]]) + 1)
		}
	}
	for i := 0; i < n-1; i++ {
		if runes[i] == '\n' && runes[i+1] == '\n' {
			add(i + 2)
		}
	}
	step := sliceWindow - sliceOverlap
	if step < 1 {
		step = sliceWindow
	}
	for i := sliceWindow; i < n; i += step {
		add(i)
	}
	out := make([]int, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func chapterStarts(text string) []int {
	var out []int
	seen := map[int]bool{}
	for _, m := range chapterRe.FindAllStringIndex(text, -1) {
		off := utf8.RuneCountInString(text[:m[0]])
		if m[0] < len(text) && (text[m[0]] == '\n' || text[m[0]] == '\r') {
			off++
		}
		if seen[off] {
			continue
		}
		seen[off] = true
		out = append(out, off)
	}
	sort.Ints(out)
	return out
}

func snapToSeam(seams []int, pos int) int {
	if len(seams) == 0 {
		return pos
	}
	best := seams[0]
	bestD := absInt(pos - best)
	for _, s := range seams {
		if d := absInt(pos - s); d < bestD {
			best, bestD = s, d
		}
	}
	return best
}

func runeSlice(text string, start, end int) string {
	runes := []rune(text)
	n := len(runes)
	if start < 0 {
		start = 0
	}
	if end > n {
		end = n
	}
	if start >= end {
		return ""
	}
	return string(runes[start:end])
}

func filmTarget(sourceRunes int) int {
	if sourceRunes < 1 {
		return minFilmSeconds
	}
	sec := sourceRunes * 60 / (CharsPerMinute * 6)
	if sourceRunes < 400 {
		sec = 30
	}
	if sec < minFilmSeconds && sourceRunes >= 400 {
		sec = minFilmSeconds
	}
	if sec > maxFilmSeconds {
		sec = maxFilmSeconds
	}
	return sec
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func looksBinary(b []byte) bool {
	n := len(b)
	if n > 800 {
		n = 800
	}
	zeros := 0
	for i := 0; i < n; i++ {
		if b[i] == 0 {
			zeros++
		}
	}
	return zeros > n/20
}

func queryOfficeText(path string) (string, error) {
	return office.Query(path)
}
