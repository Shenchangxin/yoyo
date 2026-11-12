// 把一份文档的文字切成节，再把每个单元切成检索块。
// 节是阅读和定位的单位：大约 2800 个符文，不超过 4200，在标题和段落处断开。
// 块是检索的单位：大约 900 个符文，不超过 1400，是单元文字的原样片段，并带着它所在的标题路径。
// 这里只做纯文本运算，不读文件也不碰数据库。

package learn

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	// sectionTarget 是一节的目标长度，按符文算。攒到这个长度就在下一个段落处断开。
	sectionTarget = 2800
	// sectionCap 是一节的硬上限。
	sectionCap = 4200
	// sectionHeadingMin 是遇到标题就另起一节所需的最少长度。太短的节不值得单独翻一页。
	sectionHeadingMin = 900

	// chunkTarget 是一个检索块的目标长度，按符文算。
	chunkTarget = 900
	// chunkCap 是一个检索块的硬上限。
	chunkCap = 1400
	// chunkHeadingMin 是遇到标题就另起一块所需的最少长度。
	chunkHeadingMin = 300
	// chunkOverlap 是在句子中间硬切时两块之间重叠的符文数。
	chunkOverlap = 80

	// maxOutline 是一份文档最多记多少条提纲。
	maxOutline = 5000
	// maxHeadingRunes 是标题和标题路径的最大长度。
	maxHeadingRunes = 200
)

// block 是切节之前的一小块文字：一个段落、一个标题或一段围栏代码。
// Level 大于 0 时这一块是标题，Title 是去掉标记后的标题文字。Fence 不为空时这一块是围栏代码，值是围栏标记。
type block struct {
	Text  string
	Level int
	Title string
	Fence string
}

// packItem 是参加装箱的一项：它在文字里的起止位置，以及它是不是标题。
type packItem struct {
	start, end int
	heading    bool
}

var (
	mdLinkRe  = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	mdBlankRe = regexp.MustCompile(`\n[ \t]*\n`)
)

// atxHeading 判断 line 是不是 ATX 标题（一到六个 # 后跟空白）。是的话返回级别和去掉标记的标题文字。
func atxHeading(line string) (int, string, bool) {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 {
		return 0, "", false
	}
	n := 0
	for n < len(t) && t[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return 0, "", false
	}
	if n < len(t) && t[n] != ' ' && t[n] != '\t' {
		return 0, "", false
	}
	return n, headingTitle(t[n:]), true
}

// headingTitle 把标题行的正文 s 收成纯文字：去掉结尾的 #、链接和图片的地址、强调和代码标记，最长 200 个符文。
func headingTitle(s string) string {
	s = strings.TrimSpace(s)
	if cut := strings.TrimRight(s, "#"); cut != s && (cut == "" || strings.HasSuffix(cut, " ") || strings.HasSuffix(cut, "\t")) {
		s = strings.TrimSpace(cut)
	}
	s = mdLinkRe.ReplaceAllString(s, "$1")
	s = strings.NewReplacer("**", "", "__", "", "`", "", "*", "").Replace(s)
	return clipRunes(strings.Join(strings.Fields(s), " "), maxHeadingRunes)
}

// fenceMarker 判断 line 是不是代码围栏的开头或结尾。是的话返回围栏标记（至少三个 ` 或 ~）和标记后面的文字。
func fenceMarker(line string) (string, string, bool) {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 || len(t) < 3 {
		return "", "", false
	}
	c := t[0]
	if c != '`' && c != '~' {
		return "", "", false
	}
	n := 0
	for n < len(t) && t[n] == c {
		n++
	}
	if n < 3 {
		return "", "", false
	}
	return t[:n], t[n:], true
}

// markdownBlocks 把 markdown 文本 text 切成块。空行分开段落，围栏代码整段算一块，围栏外的 ATX 标题各自成块。
// 块里的文字保持 markdown 原样，阅读器负责渲染。
func markdownBlocks(text string) []block {
	var out []block
	var cur []string
	fence := ""
	flush := func(mark string) {
		body := strings.Trim(strings.Join(cur, "\n"), "\n")
		cur = cur[:0]
		if strings.TrimSpace(body) != "" {
			out = append(out, block{Text: body, Fence: mark})
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if fence != "" {
			cur = append(cur, line)
			if mark, rest, ok := fenceMarker(line); ok && mark[0] == fence[0] && len(mark) >= len(fence) && strings.TrimSpace(rest) == "" {
				flush(fence)
				fence = ""
			}
			continue
		}
		if mark, _, ok := fenceMarker(line); ok {
			flush("")
			fence = mark
			cur = append(cur, line)
			continue
		}
		if strings.TrimSpace(line) == "" {
			flush("")
			continue
		}
		if level, title, ok := atxHeading(line); ok {
			flush("")
			b := block{Text: strings.TrimRight(line, " \t")}
			if title != "" {
				b.Level, b.Title = level, title
			}
			out = append(out, b)
			continue
		}
		cur = append(cur, line)
	}
	flush(fence)
	return out
}

// textBlocks 把纯文本 text 按空行切成段落块。换页符当作段落分隔。纯文本没有标题。
func textBlocks(text string) []block {
	text = strings.ReplaceAll(text, "\f", "\n\n")
	var out []block
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			out = append(out, block{Text: strings.Join(cur, "\n")})
			cur = cur[:0]
		}
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		cur = append(cur, line)
	}
	flush()
	return out
}

// bestCut 在 r 里为从 start 开始的一段找断点，返回断点位置和它是不是硬切。
// 断点落在 start 之后 target 的一半到 hardCap 之间。优先空行，其次换行，再次句末标点，同级里取离 target 最近的。
// 都没有时在 target 处硬切。调用方保证 r 在 start 之后长于 hardCap。
func bestCut(r []rune, start, target, hardCap int) (int, bool) {
	lo, hi := start+target/2, start+hardCap
	if hi > len(r)-1 {
		hi = len(r) - 1
	}
	want := start + target
	best, bestRank, bestDist := -1, 0, 0
	for p := lo; p <= hi; p++ {
		rank := 0
		prev := r[p-1]
		switch {
		case prev == '\n' && p >= 2 && r[p-2] == '\n' && r[p] != '\n':
			rank = 3
		case prev == '\n' && r[p] != '\n':
			rank = 2
		case strings.ContainsRune("。！？；", prev):
			rank = 1
		case strings.ContainsRune(".!?;", prev) && (r[p] == ' ' || r[p] == '\n'):
			rank = 1
		}
		if rank == 0 {
			continue
		}
		dist := p - want
		if dist < 0 {
			dist = -dist
		}
		if rank > bestRank || (rank == bestRank && dist < bestDist) {
			best, bestRank, bestDist = p, rank, dist
		}
	}
	if best < 0 {
		return want, true
	}
	return best, false
}

// splitSpans 把 r 切成若干段，每段不超过 hardCap，尽量接近 target。返回每段的起止位置。
// overlap 大于 0 时，硬切处的下一段往回多带 overlap 个符文。r 不超过 hardCap 时原样返回一段。
func splitSpans(r []rune, target, hardCap, overlap int) [][2]int {
	var out [][2]int
	start := 0
	for len(r)-start > hardCap {
		cut, hard := bestCut(r, start, target, hardCap)
		out = append(out, [2]int{start, cut})
		start = cut
		if hard && overlap > 0 && overlap < target/2 {
			start = cut - overlap
		}
	}
	if start < len(r) {
		out = append(out, [2]int{start, len(r)})
	}
	return out
}

// splitOversize 把 blocks 里超过 hardCap 的块切小，其余原样留下。target 是切出来每段的目标长度。
// 围栏代码切开后每一段都重新带上围栏，这样每一节单独渲染时仍是代码。超长的标题当作正文。
func splitOversize(blocks []block, target, hardCap int) []block {
	out := make([]block, 0, len(blocks))
	for _, b := range blocks {
		r := []rune(b.Text)
		if len(r) <= hardCap {
			out = append(out, b)
			continue
		}
		if b.Fence != "" {
			out = append(out, splitFence(b, target, hardCap)...)
			continue
		}
		for _, sp := range splitSpans(r, target, hardCap, 0) {
			piece := strings.Trim(string(r[sp[0]:sp[1]]), "\n")
			if strings.TrimSpace(piece) != "" {
				out = append(out, block{Text: piece})
			}
		}
	}
	return out
}

// splitFence 把一段超长的围栏代码 b 切成几段，每段都带开栏行和收栏标记。target 和 hardCap 是每段连同围栏在内的长度限制。
func splitFence(b block, target, hardCap int) []block {
	lines := strings.Split(b.Text, "\n")
	open := lines[0]
	body := lines[1:]
	if n := len(body); n > 0 {
		if mark, rest, ok := fenceMarker(body[n-1]); ok && mark[0] == b.Fence[0] && len(mark) >= len(b.Fence) && strings.TrimSpace(rest) == "" {
			body = body[:n-1]
		}
	}
	overhead := utf8.RuneCountInString(open) + utf8.RuneCountInString(b.Fence) + 2
	if overhead > hardCap/4 {
		open = b.Fence
		overhead = 2*utf8.RuneCountInString(b.Fence) + 2
	}
	r := []rune(strings.Join(body, "\n"))
	var out []block
	for _, sp := range splitSpans(r, target-overhead, hardCap-overhead, 0) {
		piece := strings.Trim(string(r[sp[0]:sp[1]]), "\n")
		if strings.TrimSpace(piece) == "" {
			continue
		}
		out = append(out, block{Text: open + "\n" + piece + "\n" + b.Fence, Fence: b.Fence})
	}
	return out
}

// packGroups 把按顺序排好的 items 装成若干组，返回每组在 items 里的起止下标。
// 一组攒到 target 就收尾；再加一项会超过 hardCap 时先收尾；遇到标题而当前组已有 headingMin 那么长时另起一组。
// 收尾时落在组末的标题挪到下一组，免得标题和它的正文分开。单项超过 hardCap 时那一组也会超过，调用方要先把它切小。
func packGroups(items []packItem, target, hardCap, headingMin int) [][2]int {
	var groups [][2]int
	first := 0
	size := func(end int) int {
		if end <= first {
			return 0
		}
		return items[end-1].end - items[first].start
	}
	flush := func(end int) {
		keep := end
		for keep > first && items[keep-1].heading {
			keep--
		}
		if keep == first {
			keep = end
		}
		groups = append(groups, [2]int{first, keep})
		first = keep
	}
	for i, it := range items {
		if i > first && it.heading && !items[i-1].heading && size(i) >= headingMin {
			flush(i)
		}
		for i > first && it.end-items[first].start > hardCap {
			flush(i)
		}
		if !it.heading && size(i+1) >= target {
			flush(i + 1)
		}
	}
	if first < len(items) {
		groups = append(groups, [2]int{first, len(items)})
	}
	return groups
}

// sectionize 把 blocks 装成节，返回以节为单元的文档。同一节里的块用一个空行连接。
// 提纲记下每个标题落在第几节、节内第几个符文。blocks 为空时返回没有单元的文档。
func sectionize(blocks []block) parsedDoc {
	doc := parsedDoc{Unit: UnitSection}
	blocks = splitOversize(blocks, sectionTarget, sectionCap)
	items := make([]packItem, len(blocks))
	pos := 0
	for i, b := range blocks {
		n := utf8.RuneCountInString(b.Text)
		items[i] = packItem{start: pos, end: pos + n, heading: b.Level > 0}
		pos += n + 2
	}
	for _, g := range packGroups(items, sectionTarget, sectionCap, sectionHeadingMin) {
		locator := len(doc.Units) + 1
		base := items[g[0]].start
		var sb strings.Builder
		for i := g[0]; i < g[1]; i++ {
			if i > g[0] {
				sb.WriteString("\n\n")
			}
			b := blocks[i]
			if b.Level > 0 && b.Title != "" && len(doc.Outline) < maxOutline {
				doc.Outline = append(doc.Outline, OutlineEntry{Locator: locator, Title: b.Title, Level: b.Level, Offset: items[i].start - base})
			}
			sb.WriteString(b.Text)
		}
		doc.Units = append(doc.Units, PageText{Page: locator, Text: sb.String(), Status: unitOK})
		doc.Chars += items[g[1]-1].end - base
	}
	return doc
}

// paragraphSpans 返回 r 里每个段落的起止位置。只含空白的行分开段落，段落本身不含首尾的空行。
func paragraphSpans(r []rune) [][2]int {
	var out [][2]int
	start, end := -1, 0
	lineStart, blank := 0, true
	closeLine := func(lineEnd int) {
		if blank {
			if start >= 0 {
				out = append(out, [2]int{start, end})
				start = -1
			}
			return
		}
		if start < 0 {
			start = lineStart
		}
		end = lineEnd
	}
	for i, c := range r {
		if c == '\n' {
			closeLine(i)
			lineStart, blank = i+1, true
			continue
		}
		if c != ' ' && c != '\t' {
			blank = false
		}
	}
	closeLine(len(r))
	if start >= 0 {
		out = append(out, [2]int{start, end})
	}
	return out
}

// chunkSpans 把一个单元的文字 r 切成检索块，返回每块的起止符文位置。每块都是原文的连续片段。
// headings 是这个单元里各标题的起始位置，标题开头的段落尽量和它后面的正文留在同一块；为空时不考虑标题。
func chunkSpans(r []rune, headings []int) [][2]int {
	isHeading := map[int]bool{}
	for _, h := range headings {
		isHeading[h] = true
	}
	var items []packItem
	for _, p := range paragraphSpans(r) {
		if p[1]-p[0] <= chunkCap {
			items = append(items, packItem{start: p[0], end: p[1], heading: isHeading[p[0]]})
			continue
		}
		for _, sp := range splitSpans(r[p[0]:p[1]], chunkTarget, chunkCap, chunkOverlap) {
			items = append(items, packItem{start: p[0] + sp[0], end: p[0] + sp[1]})
		}
	}
	var out [][2]int
	for _, g := range packGroups(items, chunkTarget, chunkCap, chunkHeadingMin) {
		out = append(out, [2]int{items[g[0]].start, items[g[1]-1].end})
	}
	return out
}

// headingTrail 记着当前位置上方的各级标题。
type headingTrail struct {
	levels []int
	titles []string
}

// push 记下一个新标题。level 是它的级别，同级和更深的旧标题先被它顶掉；title 是标题文字。
func (t *headingTrail) push(level int, title string) {
	for len(t.levels) > 0 && t.levels[len(t.levels)-1] >= level {
		t.levels = t.levels[:len(t.levels)-1]
		t.titles = t.titles[:len(t.titles)-1]
	}
	t.levels = append(t.levels, level)
	t.titles = append(t.titles, title)
}

// path 返回从最上级到当前标题的路径，用 " > " 连接，最长 200 个符文。没有标题时是空串。
func (t *headingTrail) path() string {
	return clipRunes(strings.Join(t.titles, " > "), maxHeadingRunes)
}

// docChunk 是一份文档里的一个检索块。Locator 是它所在的单元，Start 是它在单元文字里的符文位置，Heading 是它所在的标题路径。
type docChunk struct {
	Locator int
	Start   int
	Text    string
	Heading string
}

// chunkDoc 把 doc 的每个有文字的单元切成检索块，并给每块标上标题路径。标题路径跨单元延续。
func chunkDoc(doc parsedDoc) []docChunk {
	outline := append([]OutlineEntry(nil), doc.Outline...)
	sort.SliceStable(outline, func(i, j int) bool {
		if outline[i].Locator != outline[j].Locator {
			return outline[i].Locator < outline[j].Locator
		}
		return outline[i].Offset < outline[j].Offset
	})
	var out []docChunk
	var trail headingTrail
	next := 0
	for _, u := range doc.Units {
		if u.Status != unitOK || strings.TrimSpace(u.Text) == "" {
			continue
		}
		var headings []int
		if doc.Unit == UnitSection {
			for _, e := range outline {
				if e.Locator == u.Page {
					headings = append(headings, e.Offset)
				}
			}
		}
		r := []rune(u.Text)
		for _, sp := range chunkSpans(r, headings) {
			for next < len(outline) && (outline[next].Locator < u.Page || (outline[next].Locator == u.Page && outline[next].Offset <= sp[0])) {
				trail.push(outline[next].Level, outline[next].Title)
				next++
			}
			out = append(out, docChunk{Locator: u.Page, Start: sp[0], Text: string(r[sp[0]:sp[1]]), Heading: trail.path()})
		}
	}
	return out
}

// collapseBlank 把 text 里连续的空行压成一个空行，并去掉首尾空白。
func collapseBlank(text string) string {
	for {
		next := mdBlankRe.ReplaceAllString(text, "\n\n")
		if next == text {
			break
		}
		text = next
	}
	return strings.TrimSpace(text)
}
