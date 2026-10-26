// 按领域目录存放可迁移经验。cue、lesson、evidence 三者缺一，或正文含禁用词、held 编号时不落盘。
// 检索只读本领域目录。采纳的材料记内容寻址哈希，L1 文件由 WriteL1 展开。Overlay 在采纳后调用，不移动 refs。

package evolve

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

const (
	// LibraryNoEntry 表示没有写入经验，值为 no_entry。
	LibraryNoEntry = "no_entry"
	// LibraryAdded 表示已经写入一条经验，值为 added。
	LibraryAdded = "added"
	// libraryIndexCap 是 Context 返回的 INDEX 字节上限。
	libraryIndexCap = 8_000
	// libraryUnknown 是空路径或无法成名时的领域目录名。
	libraryUnknown = "unknown"
)

var (
	// librarySlugCut 把目录名里字母和数字以外的片段换成下划线。点、下划线和连字符保留。
	librarySlugCut = regexp.MustCompile(`[^\p{L}\p{N}._-]+`)
	// libraryBanned 是经验正文里出现就不能落盘的子串，比较时用小写。
	libraryBanned = []string{
		"pass_rate", "shouldpromote", "held-out", "held_out",
		"grader", "rubric", "benchmark", "harbor-index",
	}
)

// Experience 是一条可迁移经验。Cue、Lesson、Evidence 都非空才算完整，否则 Add 不建条目。它不计入 Overlay 的 R/P 账本。
type Experience struct {
	Cue        string        `json:"cue,omitempty"`
	Lesson     string        `json:"lesson,omitempty"`
	Evidence   string        `json:"evidence,omitempty"`
	Confidence string        `json:"confidence,omitempty"`
	Edits      string        `json:"edits,omitempty"`
	Verdicts   []EditVerdict `json:"verdicts,omitempty"`
}

// EditVerdict 是一个组件上的判定行。experienceTransferable 只承认 helped、hurt 和 no observed effect，单靠 confounded 不算可迁移。
type EditVerdict struct {
	Component  string `json:"component,omitempty"`
	Candidates string `json:"candidates,omitempty"`
	Change     string `json:"change,omitempty"`
	Observed   string `json:"observed,omitempty"`
	Verdict    string `json:"verdict,omitempty"`
}

// HasExperience 在 Cue、Lesson、Evidence 去空白后都非空时为真。
func HasExperience(e Experience) bool {
	return strings.TrimSpace(e.Cue) != "" &&
		strings.TrimSpace(e.Lesson) != "" &&
		strings.TrimSpace(e.Evidence) != ""
}

// Library 是一个领域的磁盘经验库。目录名来自 DomainSlug，一个领域目录不放另一个领域的条目。采纳结果存的是内容寻址哈希。
type Library struct {
	Root   string
	Domain string
}

// DomainSlug 取路径最后一段，去掉分隔符和首尾的点、下划线、连字符，再把大写改成小写。空结果用 libraryUnknown。
func DomainSlug(workspace string) string {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return libraryUnknown
	}
	base := filepath.Base(filepath.Clean(workspace))
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return libraryUnknown
	}
	s := librarySlugCut.ReplaceAllString(base, "_")
	s = strings.Trim(s, "._-")
	if s == "" {
		return libraryUnknown
	}
	var b strings.Builder
	for _, r := range s {
		if unicode.IsUpper(r) {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if out == "" {
		return libraryUnknown
	}
	return out
}

// taskSlug 用 DomainSlug 给题号起目录名。结果是 libraryUnknown 且题号非空时，改用把非字母数字换成下划线的原文。
func taskSlug(taskID string) string {
	s := DomainSlug(taskID)
	if s == libraryUnknown && strings.TrimSpace(taskID) != "" {
		return librarySlugCut.ReplaceAllString(strings.TrimSpace(taskID), "_")
	}
	return s
}

// Dir 返回根目录下本领域的文件夹。库为 nil 时返回空串。
func (l *Library) Dir() string {
	if l == nil {
		return ""
	}
	return filepath.Join(l.Root, DomainSlug(l.Domain))
}

// IndexPath 返回本领域的 INDEX.md 路径。
func (l *Library) IndexPath() string {
	return filepath.Join(l.Dir(), "INDEX.md")
}

// entryDir 返回该题号在 entries 下的目录。
func (l *Library) entryDir(taskID string) string {
	return filepath.Join(l.Dir(), "entries", taskSlug(taskID))
}

// LibraryEntry 是给操作者看的一行经验：编号、三字段和采纳哈希。
type LibraryEntry struct {
	ID       string `json:"id"`
	Cue      string `json:"cue,omitempty"`
	Lesson   string `json:"lesson,omitempty"`
	Evidence string `json:"evidence,omitempty"`
	Hash     string `json:"hash,omitempty"`
}

// LibraryView 是一个领域的只读索引和条目列表。
type LibraryView struct {
	Domain  string         `json:"domain"`
	Index   string         `json:"index,omitempty"`
	Entries []LibraryEntry `json:"entries,omitempty"`
}

// View 列出 entries 下的子目录。正文仍含 held 编号的行跳过。库为 nil 时领域名为 libraryUnknown。
func (l *Library) View(held map[string]bool) LibraryView {
	out := LibraryView{Domain: libraryUnknown}
	if l == nil {
		return out
	}
	out.Domain = DomainSlug(l.Domain)
	out.Index = stripHeld(l.Context(), held)
	dir := filepath.Join(l.Dir(), "entries")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		ed := filepath.Join(dir, e.Name())
		row := LibraryEntry{
			ID:       e.Name(),
			Cue:      stripHeld(experienceField(filepath.Join(ed, "EXPERIENCE.md"), "cue"), held),
			Lesson:   stripHeld(experienceField(filepath.Join(ed, "EXPERIENCE.md"), "lesson"), held),
			Evidence: stripHeld(experienceField(filepath.Join(ed, "EXPERIENCE.md"), "evidence"), held),
			Hash:     strings.TrimSpace(readSmall(filepath.Join(ed, "hash"))),
		}
		if experienceHasHeld(Experience{Cue: row.Cue, Lesson: row.Lesson, Evidence: row.Evidence}, held) {
			continue
		}
		out.Entries = append(out.Entries, row)
	}
	return out
}

// Context 读 INDEX.md。库或根为空、或文件读失败时返回空串。长于 libraryIndexCap 字节时截断。
func (l *Library) Context() string {
	if l == nil || l.Root == "" {
		return ""
	}
	b, err := os.ReadFile(l.IndexPath())
	if err != nil {
		return ""
	}
	s := string(b)
	if len(s) > libraryIndexCap {
		s = s[:libraryIndexCap]
	}
	return s
}

// Add 写入一条完整经验及其采纳哈希，并重建索引。根为空或题号为空时返回错误。经验含 held 编号、清洗后不完整，或命中 libraryBanned 时返回 false 和 nil，不建文件。
func (l *Library) Add(taskID, adoptedHash, why string, exp Experience, held map[string]bool) (bool, error) {
	if l == nil || strings.TrimSpace(l.Root) == "" {
		return false, fmt.Errorf("library: root required")
	}
	l.Domain = DomainSlug(l.Domain)
	if experienceHasHeld(exp, held) {
		return false, nil
	}
	exp = SanitizeExperience(exp, held)
	if !HasExperience(exp) || experienceBanned(exp) {
		return false, nil
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false, fmt.Errorf("library: task id required")
	}
	d := l.entryDir(taskID)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return false, err
	}
	body := renderExperienceMD(taskID, adoptedHash, why, exp)
	if err := os.WriteFile(filepath.Join(d, "EXPERIENCE.md"), []byte(body), 0o644); err != nil {
		return false, err
	}
	if err := os.WriteFile(filepath.Join(d, "hash"), []byte(strings.TrimSpace(adoptedHash)+"\n"), 0o644); err != nil {
		return false, err
	}
	if err := l.rebuildIndex(); err != nil {
		return false, err
	}
	return true, nil
}

// rebuildIndex 按 entries 子目录重写 INDEX.md。没有条目时写上空库说明。
func (l *Library) rebuildIndex() error {
	dir := l.Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entriesDir := filepath.Join(dir, "entries")
	ents, _ := os.ReadDir(entriesDir)
	var b strings.Builder
	fmt.Fprintf(&b, "# Library — %s\n\n", l.Domain)
	b.WriteString("Each entry is an earlier task in THIS domain whose harness was chosen, stored ")
	b.WriteString("with its CAS hash, L1 materials under harness/, and why it was picked. Browse this index, then open the ")
	b.WriteString("entries whose cue matches the shape of the matter in front of you.\n\n")
	n := 0
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		n++
		ed := filepath.Join(entriesDir, e.Name())
		cue := experienceField(filepath.Join(ed, "EXPERIENCE.md"), "cue")
		hash := strings.TrimSpace(readSmall(filepath.Join(ed, "hash")))
		fmt.Fprintf(&b, "## `entries/%s/`\n", e.Name())
		if hash != "" {
			fmt.Fprintf(&b, "- harness: `%s` — `entries/%s/harness/` (L1 only)\n", hash, e.Name())
		}
		fmt.Fprintf(&b, "- why:     `entries/%s/EXPERIENCE.md`\n", e.Name())
		if cue != "" {
			fmt.Fprintf(&b, "- cue:     %s\n", cue)
		}
		b.WriteByte('\n')
	}
	if n == 0 {
		b.WriteString("_(empty — this is the first task in this domain)_\n")
	}
	return os.WriteFile(l.IndexPath(), []byte(b.String()), 0o644)
}

// renderExperienceMD 把题号、哈希、选择理由、编辑表和可迁移字段写成 EXPERIENCE.md 正文。
func renderExperienceMD(taskID, hash, why string, exp Experience) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", taskID)
	if hash != "" {
		fmt.Fprintf(&b, "- adopted: `%s`\n\n", hash)
	}
	if strings.TrimSpace(why) != "" {
		b.WriteString("## Why this was chosen\n")
		b.WriteString(strings.TrimSpace(why))
		b.WriteString("\n\n")
	}
	if strings.TrimSpace(exp.Edits) != "" {
		b.WriteString("## What each edit did\n")
		b.WriteString(strings.TrimSpace(exp.Edits))
		b.WriteString("\n\n")
	}
	b.WriteString("## Transferable lesson\n")
	for _, kv := range [][2]string{
		{"cue", exp.Cue},
		{"lesson", exp.Lesson},
		{"evidence", exp.Evidence},
		{"confidence", exp.Confidence},
	} {
		if strings.TrimSpace(kv[1]) == "" {
			continue
		}
		fmt.Fprintf(&b, "- **%s**: %s\n", kv[0], strings.TrimSpace(kv[1]))
	}
	b.WriteByte('\n')
	return b.String()
}

// experienceField 从 EXPERIENCE.md 里取 `- **name**:` 这一行的余下文本。
func experienceField(path, name string) string {
	for _, line := range strings.Split(readSmall(path), "\n") {
		pref := "- **" + name + "**:"
		if strings.HasPrefix(line, pref) {
			return strings.TrimSpace(strings.TrimPrefix(line, pref))
		}
	}
	return ""
}

// readSmall 读整个文件。读失败时返回空串。
func readSmall(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

// SanitizeExperience 用 stripHeld 去掉各字段里的 held 编号并去掉首尾空白。必填字段被清空后，由调用方把它当成不写入。
func SanitizeExperience(exp Experience, held map[string]bool) Experience {
	exp.Cue = strings.TrimSpace(stripHeld(exp.Cue, held))
	exp.Lesson = strings.TrimSpace(stripHeld(exp.Lesson, held))
	exp.Evidence = strings.TrimSpace(stripHeld(exp.Evidence, held))
	exp.Confidence = strings.TrimSpace(stripHeld(exp.Confidence, held))
	exp.Edits = strings.TrimSpace(stripHeld(exp.Edits, held))
	for i := range exp.Verdicts {
		exp.Verdicts[i].Component = strings.TrimSpace(stripHeld(exp.Verdicts[i].Component, held))
		exp.Verdicts[i].Candidates = strings.TrimSpace(stripHeld(exp.Verdicts[i].Candidates, held))
		exp.Verdicts[i].Change = strings.TrimSpace(stripHeld(exp.Verdicts[i].Change, held))
		exp.Verdicts[i].Observed = strings.TrimSpace(stripHeld(exp.Verdicts[i].Observed, held))
		exp.Verdicts[i].Verdict = strings.TrimSpace(stripHeld(exp.Verdicts[i].Verdict, held))
	}
	return exp
}

// experienceHasHeld 在 cue、lesson、evidence、edits 或判定表里出现任一非空 held 编号时为真。held 为空时为假。
func experienceHasHeld(exp Experience, held map[string]bool) bool {
	if len(held) == 0 {
		return false
	}
	blob := exp.Cue + "\n" + exp.Lesson + "\n" + exp.Evidence + "\n" + exp.Edits + "\n" + renderEditTable(exp.Verdicts)
	for id := range held {
		if id != "" && strings.Contains(blob, id) {
			return true
		}
	}
	return false
}

// experienceBanned 在小写拼接正文里命中 libraryBanned 任一子串时为真。
func experienceBanned(exp Experience) bool {
	blob := strings.ToLower(exp.Cue + "\n" + exp.Lesson + "\n" + exp.Evidence + "\n" + exp.Edits + "\n" + renderEditTable(exp.Verdicts))
	for _, w := range libraryBanned {
		if strings.Contains(blob, w) {
			return true
		}
	}
	return false
}

// WriteL1 把采纳快照展开成条目 harness/ 下的 L1 文件。RenderL1 不含剧本和提示片段。含评分禁用词的正文跳过。库、库对象或哈希为空时返回 nil。
func (l *Library) WriteL1(cas *artifact.Store, taskID, hash string, held map[string]bool) error {
	if l == nil || cas == nil || strings.TrimSpace(hash) == "" {
		return nil
	}
	snap, err := cas.GetSnapshot(hash)
	if err != nil {
		return err
	}
	mats := MaterialsFromSnapshot(cas, snap, hash, MaterialSet{})
	dir := filepath.Join(l.entryDir(taskID), "harness")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for rel, body := range RenderL1(mats) {
		body = stripHeld(body, held)
		if overlayHasGrader(body) {
			continue
		}
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return l.rebuildIndex()
}
