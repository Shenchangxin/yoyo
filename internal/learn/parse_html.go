// 把 HTML 读成可以阅读的文字：保留标题、段落、列表、代码和表格行，丢掉脚本、样式和导航。
// 导入 .html 文件时得到纯文字块，网页导入时得到 markdown。这里是一个容错的小词法器，不构造 DOM，也不执行任何脚本。
// 抓取网页不在这里，见 library_url.go。

package learn

import (
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/htmlindex"
)

// htmlBlock 是 HTML 里的一个块级片段。Kind 是 heading、para、item、pre、row 或 rule。
// Plain 是纯文字，MD 是带行内标记的 markdown。Main 表示它在 article 或 main 里面。
type htmlBlock struct {
	Kind    string
	Level   int
	Plain   string
	MD      string
	Main    bool
	Quote   bool
	Ordered bool
	Depth   int
}

// htmlSkipAlways 是内容永远不算正文的标签。
var htmlSkipAlways = map[string]bool{
	"script": true, "style": true, "noscript": true, "template": true, "svg": true, "iframe": true,
	"canvas": true, "object": true, "select": true, "button": true, "head": true, "math": true,
}

// htmlSkipChrome 是页面外壳的标签。严格模式下跳过，正文太少时放宽重读。
var htmlSkipChrome = map[string]bool{"nav": true, "footer": true, "aside": true, "form": true}

// htmlBlockTags 是会打断段落的标签。
var htmlBlockTags = map[string]bool{
	"p": true, "div": true, "section": true, "article": true, "main": true, "header": true, "ul": true, "ol": true,
	"table": true, "tbody": true, "thead": true, "tfoot": true, "dl": true, "dt": true, "dd": true, "figure": true,
	"figcaption": true, "details": true, "summary": true, "body": true, "html": true, "address": true, "caption": true,
	"nav": true, "footer": true, "aside": true, "form": true, "center": true, "fieldset": true,
}

var (
	htmlCharsetRe = regexp.MustCompile(`(?i)charset\s*=\s*["']?\s*([a-z0-9_\-]+)`)
	htmlSpaceRe   = regexp.MustCompile(`[ \t\r\n\f\x{00A0}]+`)
)

// decodeHTML 把网页字节 b 解成 UTF-8 文本。contentType 是响应头里的 Content-Type，可以为空。
// 字符集先看 contentType，再看文档开头的 meta 声明，都没有时按 decodeText 的规则猜。
func decodeHTML(b []byte, contentType string) string {
	label := ""
	if m := htmlCharsetRe.FindStringSubmatch(contentType); m != nil {
		label = m[1]
	} else {
		head := b
		if len(head) > 4096 {
			head = head[:4096]
		}
		if m := htmlCharsetRe.FindSubmatch(head); m != nil {
			label = string(m[1])
		}
	}
	label = strings.ToLower(label)
	if label != "" && label != "utf-8" && label != "utf8" {
		if enc, err := htmlindex.Get(label); err == nil {
			if out, err := enc.NewDecoder().Bytes(b); err == nil {
				return normalizeText(strings.ToValidUTF8(string(out), "\uFFFD"))
			}
		}
	}
	return decodeText(b)
}

// htmlScanner 边读标签边攒块。plain 和 md 是当前块的两份行内文字。
type htmlScanner struct {
	loose   bool
	base    *url.URL
	blocks  []htmlBlock
	title   string
	plain   []byte
	md      []byte
	marks   []htmlMark
	kind    string
	level   int
	pre     bool
	main    int
	quote   int
	lists   []bool
	skipTag string
	skipN   int
}

// htmlMark 记着一个还没闭合的行内标签在 md 里从哪里开始。
type htmlMark struct {
	tag   string
	start int
	href  string
}

// htmlBlocks 把 HTML 文本 src 读成块，返回页面标题和块。标题取 title 标签，没有时取第一个 h1。
// 先按严格模式读（跳过导航、页脚、侧栏和表单），正文不足 200 个符文时放宽再读一次，取文字多的那次。
// 有 article 或 main 且里面的文字够多时只留它们里面的块。
func htmlBlocks(src string) (string, []htmlBlock) {
	return htmlBlocksBase(src, nil)
}

// htmlBlocksBase 和 htmlBlocks 一样，base 不为 nil 时相对链接按它补全成绝对地址。
func htmlBlocksBase(src string, base *url.URL) (string, []htmlBlock) {
	strict := &htmlScanner{base: base}
	strict.scan(src)
	title, blocks := strict.title, strict.blocks
	if htmlRunes(blocks) < 200 {
		loose := &htmlScanner{base: base, loose: true}
		loose.scan(src)
		if htmlRunes(loose.blocks) > htmlRunes(blocks) {
			blocks = loose.blocks
		}
		if title == "" {
			title = loose.title
		}
	}
	if title == "" {
		for _, b := range blocks {
			if b.Kind == "heading" && b.Level == 1 {
				title = b.Plain
				break
			}
		}
	}
	return clipRunes(strings.TrimSpace(title), maxHeadingRunes), pickMain(blocks)
}

// htmlRunes 返回 blocks 里纯文字的符文总数。
func htmlRunes(blocks []htmlBlock) int {
	n := 0
	for _, b := range blocks {
		n += utf8.RuneCountInString(b.Plain)
	}
	return n
}

// pickMain 在 article 或 main 里的文字不少于 400 个符文且占全文三成以上时只返回它们里面的块，否则原样返回 blocks。
func pickMain(blocks []htmlBlock) []htmlBlock {
	total, main := 0, 0
	for _, b := range blocks {
		n := utf8.RuneCountInString(b.Plain)
		total += n
		if b.Main {
			main += n
		}
	}
	if main < 400 || main*10 < total*3 {
		return blocks
	}
	out := make([]htmlBlock, 0, len(blocks))
	for _, b := range blocks {
		if b.Main {
			out = append(out, b)
		}
	}
	return out
}

// scan 读完整个文档 src，把块攒进 h.blocks。标签不配对、属性没有引号、文档半截断掉都不会出错。
func (h *htmlScanner) scan(src string) {
	h.kind = "para"
	i := 0
	for i < len(src) {
		lt := strings.IndexByte(src[i:], '<')
		if lt < 0 {
			h.text(src[i:])
			break
		}
		if lt > 0 {
			h.text(src[i : i+lt])
		}
		i += lt
		rest := src[i:]
		switch {
		case strings.HasPrefix(rest, "<!--"):
			end := strings.Index(rest[4:], "-->")
			if end < 0 {
				return
			}
			i += 4 + end + 3
			continue
		case strings.HasPrefix(rest, "<!") || strings.HasPrefix(rest, "<?"):
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				return
			}
			i += end + 1
			continue
		}
		name, attrs, closing, selfClose, size := parseTag(rest)
		if size == 0 {
			h.text("<")
			i++
			continue
		}
		i += size
		if !closing && (name == "script" || name == "style" || name == "title" || name == "textarea") && !selfClose {
			end := indexFold(src[i:], "</"+name)
			body := src[i:]
			if end >= 0 {
				body = src[i : i+end]
			}
			if name == "title" && h.title == "" && h.skipN == 0 || name == "title" && h.title == "" && h.skipTag == "head" {
				h.title = strings.TrimSpace(htmlSpaceRe.ReplaceAllString(html.UnescapeString(body), " "))
			}
			if end < 0 {
				return
			}
			i += end
			if gt := strings.IndexByte(src[i:], '>'); gt >= 0 {
				i += gt + 1
			} else {
				return
			}
			continue
		}
		if closing {
			h.closeTag(name)
		} else {
			h.openTag(name, attrs, selfClose)
		}
	}
	h.flush()
}

// indexFold 返回 needle 在 s 里第一次出现的位置，不分大小写。needle 要是小写的 ASCII。找不到返回 -1。
func indexFold(s, needle string) int {
	n := len(needle)
	for i := 0; i+n <= len(s); i++ {
		if s[i] == '<' && strings.EqualFold(s[i:i+n], needle) {
			return i
		}
	}
	return -1
}

// parseTag 解析 s 开头的一个标签。返回小写的标签名、属性、是不是结束标签、是不是自闭合，以及标签占的字节数。
// s 的开头不是合法标签时字节数为 0。标签到文档末尾都没有 > 时把剩下的全部算进去。
func parseTag(s string) (name string, attrs map[string]string, closing, selfClose bool, size int) {
	i := 1
	if i < len(s) && s[i] == '/' {
		closing = true
		i++
	}
	start := i
	for i < len(s) && (isASCIILetter(s[i]) || (i > start && (s[i] >= '0' && s[i] <= '9' || s[i] == '-' || s[i] == ':'))) {
		i++
	}
	if i == start {
		return "", nil, false, false, 0
	}
	name = strings.ToLower(s[start:i])
	for i < len(s) && s[i] != '>' {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == '/') {
			if s[i] == '/' {
				selfClose = true
			}
			i++
		}
		if i >= len(s) || s[i] == '>' {
			break
		}
		selfClose = false
		ks := i
		for i < len(s) && s[i] != '=' && s[i] != '>' && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' && s[i] != '/' {
			i++
		}
		key := strings.ToLower(s[ks:i])
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
			i++
		}
		val := ""
		if i < len(s) && s[i] == '=' {
			i++
			for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
				i++
			}
			if i < len(s) && (s[i] == '"' || s[i] == '\'') {
				q := s[i]
				i++
				vs := i
				for i < len(s) && s[i] != q {
					i++
				}
				val = s[vs:i]
				if i < len(s) {
					i++
				}
			} else {
				vs := i
				for i < len(s) && s[i] != '>' && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' {
					i++
				}
				val = s[vs:i]
			}
		}
		if key != "" && (key == "href" || key == "role") {
			if attrs == nil {
				attrs = map[string]string{}
			}
			attrs[key] = html.UnescapeString(val)
		}
		if key == "" && i < len(s) && s[i] != '>' {
			i++
		}
	}
	if i < len(s) {
		i++
	}
	return name, attrs, closing, selfClose, i
}

// isASCIILetter 在 c 是英文字母时为真。
func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// skipped 在当前位置处于被跳过的标签里面时为真。
func (h *htmlScanner) skipped() bool {
	return h.skipN > 0
}

// text 把一段标签之间的文字 raw 加进当前块。pre 里面保留空白，其余地方把连续空白压成一个空格。
func (h *htmlScanner) text(raw string) {
	if h.skipped() || raw == "" {
		return
	}
	s := html.UnescapeString(raw)
	if h.pre {
		s = normalizeText(s)
		h.plain = append(h.plain, s...)
		h.md = append(h.md, s...)
		return
	}
	s = htmlSpaceRe.ReplaceAllString(s, " ")
	if s == " " && (len(h.plain) == 0 || h.plain[len(h.plain)-1] == ' ' || h.plain[len(h.plain)-1] == '\n') {
		return
	}
	if len(h.plain) == 0 || h.plain[len(h.plain)-1] == ' ' || h.plain[len(h.plain)-1] == '\n' {
		s = strings.TrimLeft(s, " ")
	}
	h.plain = append(h.plain, s...)
	h.md = append(h.md, s...)
}

// flush 把攒着的行内文字收成一个块，种类取当前上下文。没有文字时什么都不做。没闭合的行内标记一并丢掉。
func (h *htmlScanner) flush() {
	plain, md := string(h.plain), string(h.md)
	h.plain, h.md, h.marks = h.plain[:0], h.md[:0], h.marks[:0]
	if h.kind != "pre" {
		plain, md = strings.TrimSpace(plain), strings.TrimSpace(md)
	} else {
		plain, md = strings.Trim(plain, "\n"), strings.Trim(md, "\n")
	}
	if strings.TrimSpace(plain) == "" {
		return
	}
	b := htmlBlock{Kind: h.kind, Level: h.level, Plain: plain, MD: md, Main: h.main > 0, Quote: h.quote > 0}
	if h.kind == "item" {
		b.Depth = len(h.lists)
		b.Ordered = len(h.lists) > 0 && h.lists[len(h.lists)-1]
	}
	h.blocks = append(h.blocks, b)
}

// openTag 处理一个开始标签。name 是小写标签名，attrs 里只有 href 和 role，selfClose 表示它写成了自闭合。
func (h *htmlScanner) openTag(name string, attrs map[string]string, selfClose bool) {
	if h.skipped() {
		if name == h.skipTag && !selfClose {
			h.skipN++
		}
		return
	}
	if htmlSkipAlways[name] || (!h.loose && htmlSkipChrome[name]) {
		h.flush()
		if !selfClose {
			h.skipTag, h.skipN = name, 1
		}
		return
	}
	switch name {
	case "article", "main":
		h.flush()
		h.main++
	case "h1", "h2", "h3", "h4", "h5", "h6":
		h.flush()
		h.kind, h.level = "heading", int(name[1]-'0')
	case "ul", "ol":
		h.flush()
		h.lists = append(h.lists, name == "ol")
	case "li":
		h.flush()
		h.kind = "item"
	case "pre":
		h.flush()
		h.kind, h.pre = "pre", true
	case "blockquote":
		h.flush()
		h.quote++
	case "tr":
		h.flush()
		h.kind = "row"
	case "td", "th":
		if len(strings.TrimSpace(string(h.plain))) > 0 {
			h.plain = append(h.plain, " | "...)
			h.md = append(h.md, " | "...)
		}
	case "br":
		if h.pre {
			h.plain = append(h.plain, '\n')
			h.md = append(h.md, '\n')
		} else if len(h.plain) > 0 {
			h.plain = append(h.plain, '\n')
			h.md = append(h.md, '\n')
		}
	case "hr":
		h.flush()
		h.blocks = append(h.blocks, htmlBlock{Kind: "rule", Main: h.main > 0})
	case "a":
		h.marks = append(h.marks, htmlMark{tag: "a", start: len(h.md), href: h.resolve(attrs["href"])})
	case "strong", "b", "em", "i", "code":
		if !h.pre {
			h.marks = append(h.marks, htmlMark{tag: name, start: len(h.md)})
		}
	default:
		if htmlBlockTags[name] {
			h.flush()
		}
	}
	if attrs["role"] == "main" && name != "main" && name != "article" && !selfClose {
		// role=main 的容器没有专门的结束标签可认，这里只把它之后的内容都当作正文。
		h.main++
	}
}

// closeTag 处理一个结束标签，name 是小写标签名。多出来的结束标签不会把计数减成负数。
func (h *htmlScanner) closeTag(name string) {
	if h.skipped() {
		if name == h.skipTag {
			h.skipN--
			if h.skipN == 0 {
				h.skipTag = ""
			}
		}
		return
	}
	switch name {
	case "article", "main":
		h.flush()
		if h.main > 0 {
			h.main--
		}
	case "h1", "h2", "h3", "h4", "h5", "h6":
		h.flush()
		h.kind, h.level = "para", 0
	case "ul", "ol":
		h.flush()
		if len(h.lists) > 0 {
			h.lists = h.lists[:len(h.lists)-1]
		}
		h.kind = "para"
	case "li":
		h.flush()
		h.kind = "para"
	case "pre":
		h.flush()
		h.kind, h.pre = "para", false
	case "blockquote":
		h.flush()
		if h.quote > 0 {
			h.quote--
		}
	case "tr":
		h.flush()
		h.kind = "para"
	case "a", "strong", "b", "em", "i", "code":
		h.closeMark(name)
	default:
		if htmlBlockTags[name] {
			h.flush()
		}
	}
}

// closeMark 闭合最近一个名为 tag 的行内标记，把它包住的文字改写成 markdown。里面没有文字时不留任何标记。
func (h *htmlScanner) closeMark(tag string) {
	at := -1
	for i := len(h.marks) - 1; i >= 0; i-- {
		if h.marks[i].tag == tag {
			at = i
			break
		}
	}
	if at < 0 {
		return
	}
	mark := h.marks[at]
	h.marks = h.marks[:at]
	if mark.start > len(h.md) {
		return
	}
	inner := string(h.md[mark.start:])
	core := strings.TrimSpace(inner)
	if core == "" {
		return
	}
	lead := inner[:len(inner)-len(strings.TrimLeft(inner, " \n"))]
	trail := inner[len(strings.TrimRight(inner, " \n")):]
	var wrapped string
	switch tag {
	case "a":
		if mark.href == "" {
			return
		}
		wrapped = "[" + strings.ReplaceAll(core, "\n", " ") + "](" + mark.href + ")"
	case "strong", "b":
		wrapped = "**" + core + "**"
	case "em", "i":
		wrapped = "*" + core + "*"
	default:
		if strings.Contains(core, "`") {
			return
		}
		wrapped = "`" + core + "`"
	}
	h.md = append(h.md[:mark.start], lead+wrapped+trail...)
}

// resolve 把链接地址 href 收成可以写进 markdown 的绝对地址。只留 http 和 https；相对地址在有 base 时补全，否则丢掉。
func (h *htmlScanner) resolve(href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") {
		return ""
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if h.base != nil {
		u = h.base.ResolveReference(u)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	out := u.String()
	if strings.ContainsAny(out, " ()<>\n") {
		return ""
	}
	return out
}

// plainBlocks 把 HTML 块收成切节用的纯文字块。标题保留级别，连续的列表项和表格行各自并成一块，分隔线丢掉。
func plainBlocks(blocks []htmlBlock) []block {
	var out []block
	lastKind := ""
	for _, b := range blocks {
		text := b.Plain
		switch b.Kind {
		case "rule":
			lastKind = ""
			continue
		case "heading":
			title := clipRunes(strings.Join(strings.Fields(text), " "), maxHeadingRunes)
			out = append(out, block{Text: title, Level: b.Level, Title: title})
		case "item", "row":
			if b.Kind == "item" {
				text = strings.Repeat("  ", max(b.Depth-1, 0)) + "- " + strings.ReplaceAll(text, "\n", " ")
			}
			if lastKind == b.Kind && len(out) > 0 {
				out[len(out)-1].Text += "\n" + text
			} else {
				out = append(out, block{Text: text})
			}
		default:
			out = append(out, block{Text: text})
		}
		lastKind = b.Kind
	}
	return out
}

// htmlMarkdown 把 HTML 块写成 markdown 文本。标题写成 ATX 标题，列表、引用、代码和表格用对应的 markdown 写法。
func htmlMarkdown(blocks []htmlBlock) string {
	var parts []string
	lastKind := ""
	items := 0
	join := func(text string, kind string) {
		if lastKind == kind && len(parts) > 0 && (kind == "item" || kind == "row") {
			parts[len(parts)-1] += "\n" + text
			return
		}
		parts = append(parts, text)
	}
	for i, b := range blocks {
		text := b.MD
		switch b.Kind {
		case "rule":
			parts = append(parts, "---")
		case "heading":
			parts = append(parts, strings.Repeat("#", b.Level)+" "+strings.Join(strings.Fields(text), " "))
		case "item":
			if lastKind != "item" {
				items = 0
			}
			items++
			bullet := "- "
			if b.Ordered {
				bullet = strconv.Itoa(items) + ". "
			}
			text = strings.Repeat("  ", max(b.Depth-1, 0)) + bullet + strings.ReplaceAll(text, "\n", " ")
			if b.Quote {
				text = "> " + text
			}
			join(text, "item")
		case "row":
			row := "| " + strings.ReplaceAll(text, "\n", " ") + " |"
			first := lastKind != "row"
			join(row, "row")
			if first && i+1 < len(blocks) && blocks[i+1].Kind == "row" {
				cols := strings.Count(text, " | ") + 1
				parts[len(parts)-1] += "\n|" + strings.Repeat(" --- |", cols)
			}
		case "pre":
			fence := "```"
			if strings.Contains(text, "```") {
				fence = "~~~~"
			}
			parts = append(parts, fence+"\n"+text+"\n"+fence)
		default:
			text = escapeLeading(text)
			if b.Quote {
				text = "> " + strings.ReplaceAll(text, "\n", "\n> ")
			}
			parts = append(parts, text)
		}
		lastKind = b.Kind
	}
	return collapseBlank(strings.Join(parts, "\n\n"))
}

// escapeLeading 在段落 text 的开头恰好像标题或围栏时加一个反斜杠，免得正文被当成 markdown 结构。
func escapeLeading(text string) string {
	if _, _, ok := atxHeading(text[:lineEnd(text)]); ok {
		return `\` + text
	}
	if _, _, ok := fenceMarker(text[:lineEnd(text)]); ok {
		return `\` + text
	}
	return text
}

// lineEnd 返回 text 第一行的长度。
func lineEnd(text string) int {
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return i
	}
	return len(text)
}

// HTMLToMarkdown 把 HTML 文本 src 转成 markdown，返回页面标题和正文。base 是页面地址，用来补全相对链接，可以为空。
func HTMLToMarkdown(src, base string) (string, string) {
	var bu *url.URL
	if base != "" {
		if u, err := url.Parse(base); err == nil {
			bu = u
		}
	}
	title, blocks := htmlBlocksBase(src, bu)
	return title, htmlMarkdown(blocks)
}

// PageFromHTML 把抓下来的网页字节 body 收成标题和 markdown 正文。contentType 是响应头里的 Content-Type，用来认字符集，可以为空。
// pageURL 是网页地址，用来补全相对链接，可以为空。
func PageFromHTML(body []byte, contentType, pageURL string) (title, markdown string) {
	return HTMLToMarkdown(decodeHTML(body, contentType), pageURL)
}
