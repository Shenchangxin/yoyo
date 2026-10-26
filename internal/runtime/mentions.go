// 把用户消息里的 @file、@folder、@skill、@harness、@session 和裸 @路径收成有界注入块。字节预算默认 2400。主文件不占这份预算。不负责会话片段排版，那在 tools_thread.go 的 FormatSessionMentions。

package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/capability"
)

// Mention 是用户点名的工作记忆，不是 harness 钉。展开成有界注入块，而不是把仓库倒进提示。
type Mention struct {
	Kind string `json:"kind"` // file、folder、harness、skill、session
	Ref  string `json:"ref"`
}

// reFile、reFolder、reSkill、reSession、reHarness、reBareQuoted、reBareAt 是解析 @ 提及的正则。带类型的每类最多 8 处，folder 最多 4 处，裸 @ 也最多 8 处。
var (
	// reFile 匹配 @file: 后的非空白。
	reFile = regexp.MustCompile(`@file:(\S+)`)
	// reFolder 匹配 @folder: 后的非空白。
	reFolder = regexp.MustCompile(`@folder:(\S+)`)
	// reSkill 匹配 @skill: 后的非空白。
	reSkill = regexp.MustCompile(`@skill:(\S+)`)
	// reSession 匹配 @session: 后的非空白。
	reSession = regexp.MustCompile(`@session:(\S+)`)
	// reHarness 匹配独立的 @harness。
	reHarness = regexp.MustCompile(`@harness\b`)
	// reBareQuoted 匹配 @"..." 形式的裸提及。
	reBareQuoted = regexp.MustCompile(`(?:^|\s)@"([^"]+)"`)
	// reBareAt 匹配不带引号的裸 @标记。
	reBareAt = regexp.MustCompile(`(?:^|\s)@([^\s"]+)`)
)

// ParseMentions 只解析带类型的 @file、@folder、@skill、@session 和 @harness。同类同引用只留一次。不解析裸 @路径。
func ParseMentions(text string) []Mention {
	var out []Mention
	seen := map[string]bool{}
	add := func(m Mention) {
		key := m.Kind + ":" + m.Ref
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, m)
	}
	for _, m := range reFile.FindAllStringSubmatch(text, 8) {
		add(Mention{Kind: "file", Ref: m[1]})
	}
	for _, m := range reFolder.FindAllStringSubmatch(text, 4) {
		add(Mention{Kind: "folder", Ref: m[1]})
	}
	for _, m := range reSkill.FindAllStringSubmatch(text, 8) {
		add(Mention{Kind: "skill", Ref: strings.Trim(m[1], `'"`)})
	}
	for _, m := range reSession.FindAllStringSubmatch(text, 8) {
		add(Mention{Kind: "session", Ref: strings.Trim(m[1], `'"`)})
	}
	if reHarness.MatchString(text) {
		add(Mention{Kind: "harness", Ref: "active"})
	}
	return out
}

// parseBareWorkspaceMentions 把还没被带类型语法占住的裸 @ 解析成工作区文件或目录。workspace 或 text 为空则返回 nil。跳过 harness 和 file、folder、skill、session 这四种已带冒号的前缀。
func parseBareWorkspaceMentions(workspace, text string, seen map[string]bool) []Mention {
	if workspace == "" || text == "" {
		return nil
	}
	var extra []Mention
	for _, raw := range bareAtTokens(text) {
		if raw == "" || raw == "harness" {
			continue
		}
		if i := strings.IndexByte(raw, ':'); i >= 0 {
			switch raw[:i] {
			case "file", "folder", "skill", "session":
				continue
			}
		}
		rel, kind, ok := resolveWorkspaceRef(workspace, raw)
		if !ok {
			continue
		}
		key := kind + ":" + rel
		if seen[key] {
			continue
		}
		seen[key] = true
		extra = append(extra, Mention{Kind: kind, Ref: rel})
	}
	return extra
}

// bareAtTokens 收集引号和裸 @ 标记，去掉首尾斜杠，并去重。每类正则最多取 8 处。
func bareAtTokens(text string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(raw string) {
		raw = strings.TrimSpace(strings.Trim(raw, `"'`))
		raw = strings.TrimSuffix(filepath.ToSlash(raw), "/")
		if raw == "" || seen[raw] {
			return
		}
		seen[raw] = true
		out = append(out, raw)
	}
	for _, m := range reBareQuoted.FindAllStringSubmatch(text, 8) {
		add(m[1])
	}
	for _, m := range reBareAt.FindAllStringSubmatch(text, 8) {
		add(m[1])
	}
	return out
}

// resolveWorkspaceRef 把 @ 路径收成工作区相对路径。jailPath 之后 Stat 成功则按文件或目录返回。带斜杠但 Stat 失败则不再猜。没有斜杠的名字只有在忽略规则下的树里恰好一个同名文件或目录时才展开，两个匹配则失败。
func resolveWorkspaceRef(root, raw string) (rel, kind string, ok bool) {
	raw = strings.TrimSpace(filepath.ToSlash(raw))
	raw = strings.TrimSuffix(raw, "/")
	if raw == "" {
		return "", "", false
	}
	if p, err := jailPath(root, raw); err == nil {
		if st, err := os.Stat(p); err == nil {
			kind = "file"
			if st.IsDir() {
				kind = "folder"
			}
			return raw, kind, true
		}
	}
	if strings.Contains(raw, "/") {
		return "", "", false
	}
	hit, ok := uniqueBaseHit(root, raw)
	if !ok {
		return "", "", false
	}
	return hit.Path, hit.Kind, true
}

// uniqueBaseHit 在忽略规则下的树里按基名找唯一命中。目录命中的 Kind 收成 folder。零个或多于一个都返回假。
func uniqueBaseHit(root, base string) (FileHit, bool) {
	want := strings.ToLower(base)
	root = filepath.Clean(root)
	ign := loadIgnore(root)
	n := 0
	var hit FileHit
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if path == root {
			return nil
		}
		if info.IsDir() && ign.skipDir(info.Name()) {
			return filepath.SkipDir
		}
		if strings.ToLower(info.Name()) != want {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		if !info.IsDir() && ign.skipFile(relSlash, info.Name()) {
			return nil
		}
		n++
		kind := "file"
		if info.IsDir() {
			kind = "dir"
		}
		hit = FileHit{Path: relSlash, Kind: kind}
		if n > 1 {
			return filepath.SkipAll
		}
		return nil
	})
	if n != 1 {
		return FileHit{}, false
	}
	kind := "file"
	if hit.Kind == "dir" {
		kind = "folder"
	}
	return FileHit{Path: hit.Path, Kind: kind}, true
}

// ExpandMentions 把 ExpandMentionsPacks 的技能表和技能目录都传成 nil。
func ExpandMentions(workspace, text, harnessNote string, budget int) (inject string, refs []Mention) {
	return ExpandMentionsSkills(workspace, text, harnessNote, nil, budget)
}

// ExpandMentionsSkills 把技能目录传成 nil，其余交给 ExpandMentionsPacks。
func ExpandMentionsSkills(workspace, text, harnessNote string, skills map[string]string, budget int) (inject string, refs []Mention) {
	return ExpandMentionsPacks(workspace, text, harnessNote, skills, nil, budget)
}

// mentionFileRunes 是 800，mentionPrimaryRunes 是 12000，mentionPrimaryBytes 是 8*1024。
const (
	// mentionFileRunes 是 800。普通文件注入按这个 rune 数截断正文。
	mentionFileRunes = 800
	// mentionPrimaryRunes 是 12000。主文件只注入提纲，提纲按这个 rune 数截断。
	mentionPrimaryRunes = 12_000
	// mentionPrimaryBytes 是 8*1024。正文超过这个字节数就视为主文件。
	mentionPrimaryBytes = 8 * 1024
)

// ExpandMentionsPacks 先解析带类型提及，再补裸 @路径。budget 小于等于 0 时用 2400，按字节累计 chunk 的长度。主文件（fileLooksPrimary 为真）不计入这份预算。其余片段在已写入内容后再超出预算时写下 mentions truncated 并停止。remain 小于 32 的片段被 mentionChunk 丢掉。
func ExpandMentionsPacks(workspace, text, harnessNote string, skills map[string]string, skillDirs map[string]string, budget int) (inject string, refs []Mention) {
	refs = ParseMentions(text)
	seen := map[string]bool{}
	for _, m := range refs {
		seen[m.Kind+":"+m.Ref] = true
	}
	refs = append(refs, parseBareWorkspaceMentions(workspace, text, seen)...)
	if budget <= 0 {
		budget = 2400
	}
	var b strings.Builder
	used := 0
	for _, m := range refs {
		if m.Kind == "file" && fileLooksPrimary(workspace, m.Ref) {
			chunk := mentionPrimaryFile(workspace, m.Ref)
			if chunk != "" {
				b.WriteString(chunk)
			}
			continue
		}
		remain := budget - used
		chunk := mentionChunk(workspace, harnessNote, skills, skillDirs, m, remain)
		if chunk == "" {
			continue
		}
		if used+len(chunk) > budget && used > 0 {
			b.WriteString("…[mentions truncated]\n")
			break
		}
		b.WriteString(chunk)
		used += len(chunk)
	}
	return strings.TrimSpace(b.String()), refs
}

// ExpandNamedSkills 按名字注入技能正文。budget 小于等于 0 时用 2400 字节。已写入后再超出则写下 mentions truncated 并停止。
func ExpandNamedSkills(skills, skillDirs map[string]string, names []string, budget int) string {
	if budget <= 0 {
		budget = 2400
	}
	var b strings.Builder
	used := 0
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		remain := budget - used
		chunk := mentionChunk("", "", skills, skillDirs, Mention{Kind: "skill", Ref: name}, remain)
		if chunk == "" {
			continue
		}
		if used+len(chunk) > budget && used > 0 {
			b.WriteString("…[mentions truncated]\n")
			break
		}
		b.WriteString(chunk)
		used += len(chunk)
	}
	return strings.TrimSpace(b.String())
}

// MentionSkillNames 按出现顺序返回去重后的 skill 引用。
func MentionSkillNames(refs []Mention) []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range refs {
		if r.Kind != "skill" || r.Ref == "" || seen[r.Ref] {
			continue
		}
		seen[r.Ref] = true
		out = append(out, r.Ref)
	}
	return out
}

// lookupSkillMention 先按原名取技能，再按大小写不敏感的名字，最后按目录基名。skills 为空或 ref 为空则失败。
func lookupSkillMention(skills, skillDirs map[string]string, ref string) (body, dir string, ok bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" || skills == nil {
		return "", "", false
	}
	if body, ok = skills[ref]; ok {
		if skillDirs != nil {
			dir = skillDirs[ref]
		}
		return body, dir, true
	}
	for k, v := range skills {
		if strings.EqualFold(k, ref) {
			if skillDirs != nil {
				dir = skillDirs[k]
			}
			return v, dir, true
		}
	}
	if skillDirs == nil {
		return "", "", false
	}
	for k, d := range skillDirs {
		if strings.EqualFold(k, ref) || (d != "" && strings.EqualFold(filepath.Base(d), ref)) {
			if body, ok = skills[k]; ok {
				return body, d, true
			}
		}
	}
	return "", "", false
}

// mentionChunk 在 remain 小于 32 时返回空串。skill 没有目录时写 no skill catalog，未知技能写 unknown skill，正文经 DecorateSkillBody 再按 remain 截断。harness 用 harnessNote，空则写 refs/active。session 返回空串。folder 最多列 40 项。普通文件按 mentionFileRunes 截断；字节看起来像主文件时改走 mentionPrimaryFile。
func mentionChunk(workspace, harnessNote string, skills, skillDirs map[string]string, m Mention, remain int) string {
	if remain < 32 {
		return ""
	}
	switch m.Kind {
	case "skill":
		body, dir, ok := lookupSkillMention(skills, skillDirs, m.Ref)
		if skills == nil {
			return fmt.Sprintf("## @skill:%s\nERROR: no skill catalog\n", m.Ref)
		}
		if !ok {
			return fmt.Sprintf("## @skill:%s\nERROR: unknown skill\n", m.Ref)
		}
		body = DecorateSkillBody(m.Ref, body, dir)
		capped, trunc := capText(body, remain)
		out := "## @skill:" + m.Ref + "\n" + capped
		if trunc {
			out += "\n…[truncated]"
		}
		return out + "\n"
	case "harness":
		note := harnessNote
		if note == "" {
			note = "refs/active"
		}
		return "## @harness\n" + note + "\n"
	case "session":
		return ""
	case "folder":
		p, err := jailPath(workspace, m.Ref)
		if err != nil {
			return fmt.Sprintf("## @folder:%s\nERROR: %s\n", m.Ref, err)
		}
		ents, err := os.ReadDir(p)
		if err != nil {
			return fmt.Sprintf("## @folder:%s\nERROR: %s\n", m.Ref, err)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "## @folder:%s\n", m.Ref)
		n := 0
		for _, e := range ents {
			if n >= 40 {
				b.WriteString("…\n")
				break
			}
			if e.IsDir() {
				b.WriteString("d ")
			} else {
				b.WriteString("f ")
			}
			b.WriteString(e.Name())
			b.WriteByte('\n')
			n++
		}
		return b.String()
	default:
		p, err := jailPath(workspace, m.Ref)
		if err != nil {
			return fmt.Sprintf("## @file:%s\nERROR: %s\n", m.Ref, err)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return fmt.Sprintf("## @file:%s\nERROR: %s\n", m.Ref, err)
		}
		if looksPrimaryBytes(raw) {
			return mentionPrimaryFile(workspace, m.Ref)
		}
		body := string(raw)
		capped, trunc := capText(body, mentionFileRunes)
		var b strings.Builder
		fmt.Fprintf(&b, "## @file:%s\n", m.Ref)
		b.WriteString(capped)
		if trunc {
			b.WriteString("\n…[truncated; use read_file for more]")
		}
		b.WriteByte('\n')
		return b.String()
	}
}

// fileLooksPrimary 在 jailPath 和读文件都成功、且 looksPrimaryBytes 为真时返回真。
func fileLooksPrimary(workspace, rel string) bool {
	p, err := jailPath(workspace, rel)
	if err != nil {
		return false
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	return looksPrimaryBytes(raw)
}

// looksPrimaryBytes 在长度超过 mentionPrimaryBytes（8*1024）或 rune 数超过 4000 时返回真。
func looksPrimaryBytes(raw []byte) bool {
	if len(raw) > mentionPrimaryBytes {
		return true
	}
	return utf8.RuneCount(raw) > 4000
}

// mentionPrimaryFile 写出字节数和 rune 数，并要求用 read_file 读原文。非文本只给一句说明。文本则附上 fileOutline，上限是 mentionPrimaryRunes。
func mentionPrimaryFile(workspace, rel string) string {
	p, err := jailPath(workspace, rel)
	if err != nil {
		return fmt.Sprintf("## @file:%s\nERROR: %s\n", rel, err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return fmt.Sprintf("## @file:%s\nERROR: %s\n", rel, err)
	}
	runes := utf8.RuneCount(raw)
	var b strings.Builder
	fmt.Fprintf(&b, "## @file:%s\n%d bytes, %d runes. This is the task source of truth — call read_file on this path (or task profile=explore for chapter batches). Do not invent chapters from this outline.\n", rel, len(raw), runes)
	if !looksTextBytes(raw) {
		b.WriteString("(binary or non-text; use read_file / view_image)\n")
		return b.String()
	}
	outline := fileOutline(string(raw), mentionPrimaryRunes)
	b.WriteString(outline)
	if !strings.HasSuffix(outline, "\n") {
		b.WriteByte('\n')
	}
	return b.String()
}

// looksTextBytes 只检查前 1024 字节。出现 0 则返回假。
func looksTextBytes(raw []byte) bool {
	n := len(raw)
	if n > 1024 {
		n = 1024
	}
	for _, c := range raw[:n] {
		if c == 0 {
			return false
		}
	}
	return true
}

// reOutlineLine 匹配 Markdown 标题，或「第…回/章/节/卷/篇」这种行。
var reOutlineLine = regexp.MustCompile(`(?m)^(#{1,6}\s+\S+|第.{0,12}[回章节卷篇].+)$`)

// fileOutline 最多收 80 条标题，再附上前 80 行正文，最后按 maxRunes 截断。
func fileOutline(body string, maxRunes int) string {
	var lines []string
	seen := map[string]bool{}
	for _, m := range reOutlineLine.FindAllString(body, 80) {
		s := strings.TrimSpace(m)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		lines = append(lines, s)
	}
	var b strings.Builder
	if len(lines) > 0 {
		b.WriteString("Outline:\n")
		for _, s := range lines {
			b.WriteString("- ")
			b.WriteString(s)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	b.WriteString("Head:\n")
	n := 0
	for _, line := range strings.Split(body, "\n") {
		if n >= 80 {
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
		n++
	}
	out := b.String()
	capped, trunc := capText(out, maxRunes)
	if trunc {
		capped += "\n…[outline truncated; use read_file for the rest]"
	}
	return capped
}

// jailPath 去掉引号后做 CanonicalizeToolPath。相对路径接到 workspace。Clean 之后用 WithinWorkspace 拒绝逃出工作区。本函数不调用 filepath.EvalSymlinks。
func jailPath(workspace, rel string) (string, error) {
	rel = strings.Trim(rel, `"'`)
	rel = capability.CanonicalizeToolPath(rel)
	workspace = capability.CanonicalizeToolPath(workspace)
	p := rel
	if !filepath.IsAbs(p) {
		p = filepath.Join(workspace, rel)
	}
	p = filepath.Clean(p)
	if !capability.WithinWorkspace(workspace, p) {
		return "", fmt.Errorf("path escapes workspace")
	}
	return p, nil
}
