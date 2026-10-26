// source.go 保存短剧原文，并按窗口、章节和空行给出可切的接缝。

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

// 这些数值是原文切片和成片时长估算用的窗口。
const (
	// sliceWindow 是默认读取窗口，1800 个字符。
	sliceWindow     = 1800
	// sliceOverlap 是相邻切片重叠的字符数，200。
	sliceOverlap    = 200
	// maxReadWindow 是一次读取的上限，2400 个字符。
	maxReadWindow   = 2400
	// seriesRuneMin 是直接当成连续剧的字符数，2000。
	seriesRuneMin   = 2000
	// chapterMinRunes 是靠章节标题判定连续剧时的最少字符数，400。
	chapterMinRunes = 400
	// episodeWindow 是分集切分窗口，4000 个字符。
	episodeWindow   = 4000
	// maxEpisodeRunes 是单集上限，8000 个字符。
	maxEpisodeRunes = 8000
	// minEpisodeRunes 是单集下限，600 个字符。
	minEpisodeRunes = 600
	// maxFilmSeconds 是单集成片估算的上限，90 秒。
	maxFilmSeconds  = 90
	// minFilmSeconds 是单集成片估算的下限，45 秒。
	minFilmSeconds  = 45
	// peekRunes 是切片预览的字符数，180。
	peekRunes       = 180
)

// chapterRe 匹配「第…章节回部卷」或 Chapter 加数字，忽略大小写。
var chapterRe = regexp.MustCompile(`(?mi)(?:^|\n)[ \t]*((第[0-9一二三四五六七八九十百千零两]+[章节回部卷])|(Chapter\s+\d+))`)

// LooksLikeSeries 在去掉空白后不少于 2000 字，或至少两个章节标题且不少于 400 字时返回真。
func LooksLikeSeries(text string) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(text))
	if n >= seriesRuneMin {
		return true
	}
	return len(chapterStarts(text)) >= 2 && n >= chapterMinRunes
}

// IngestSource 写入原文。剧不存在、文本为空，或方案已经是 committed 时失败。同一剧已有原文时沿用原标识。
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

// IngestSourceFile 读取 txt、md 原文，或用 office.Query 抽出 docx、pdf、pptx、xlsx。其他扩展名在前 800 字节里空字节超过二十分之一时拒绝。标题取文件名去掉扩展名。
func (e *Engine) IngestSourceFile(dramaID, path string) (Source, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Source{}, err
	}
	text := string(raw)
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".markdown":
		// txt 和 md 本身就是文本。
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

// GetSource 按剧标识找出原文。没有时返回 source not found。
func (e *Engine) GetSource(dramaID string) (Source, error) {
	for _, s := range loadCol[Source](e, colSources) {
		if s.DramaID == dramaID {
			return s, nil
		}
	}
	return Source{}, fmt.Errorf("source not found")
}

// SourceText 先读 sources 目录里的文本，没有再按哈希从内容寻址取出。
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

// ReadSource 按偏移、切片序号或查询读一段原文。切片序号从 1 计。长度空着时用 1800，超过 2400 时截断。不是整篇时 truncated 为真。
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

// putSourceText 把原文写到 sources/<id>.txt。有内容寻址时再放入并返回哈希，没有时返回空哈希。
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

// buildSlices 按 1800 字开窗、200 字重叠切开，并带上章节标题和 180 字预览。
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

// headingAt 在起点之后 80 字的第一行匹配章节标题时返回该行。
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

// CandidateSeams 收集 0、文末、章节标题、空行，以及每隔 1600 字的位置。正则吃到的行首换行会再记下一字。
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
		// 正则吃进的行首换行要跳过。
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

// chapterStarts 返回去重并排序后的章节起点。行首换行不计入起点。
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

// snapToSeam 返回离目标最近的接缝。接缝为空时返回原位置。
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

// runeSlice 按字符下标截取，越界时夹到文本两端。
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

// filmTarget 按每分钟 500 字再除以 6 估算成片秒数。不足 400 字时返回 30。400 字及以上夹在 45 到 90 秒。空文本返回 45。
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

// minInt 返回两个整数里较小的一个。
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt 返回两个整数里较大的一个。
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// absInt 返回整数的绝对值。
func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// looksBinary 查看前 800 字节，空字节超过二十分之一时当成二进制。
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

// queryOfficeText 调用 office.Query 抽出办公文档里的文本。
func queryOfficeText(path string) (string, error) {
	return office.Query(path)
}
