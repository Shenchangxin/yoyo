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
	LibraryNoEntry  = "no_entry"
	LibraryAdded    = "added"
	libraryIndexCap = 8_000
	libraryUnknown  = "unknown"
)

var (
	librarySlugCut = regexp.MustCompile(`[^\p{L}\p{N}._-]+`)
	libraryBanned  = []string{
		"pass_rate", "shouldpromote", "held-out", "held_out",
		"grader", "rubric", "benchmark", "harbor-index",
	}
)

// Experience is one transferable library row. All three of cue, lesson and
// evidence must be non-empty or the write is NO_ENTRY — same contract as
// Harness². Reflection sits outside the overlay ledger.
type Experience struct {
	Cue        string        `json:"cue,omitempty"`
	Lesson     string        `json:"lesson,omitempty"`
	Evidence   string        `json:"evidence,omitempty"`
	Confidence string        `json:"confidence,omitempty"`
	Edits      string        `json:"edits,omitempty"`
	Verdicts   []EditVerdict `json:"verdicts,omitempty"`
}

// EditVerdict is one component row from the reflector. A lesson may rest
// only on helped / hurt / no observed effect — never on confounded alone.
type EditVerdict struct {
	Component  string `json:"component,omitempty"`
	Candidates string `json:"candidates,omitempty"`
	Change     string `json:"change,omitempty"`
	Observed   string `json:"observed,omitempty"`
	Verdict    string `json:"verdict,omitempty"`
}

// HasExperience is the NO_ENTRY gate: missing any of the three fields must
// not create an on-disk entry.
func HasExperience(e Experience) bool {
	return strings.TrimSpace(e.Cue) != "" &&
		strings.TrimSpace(e.Lesson) != "" &&
		strings.TrimSpace(e.Evidence) != ""
}

// Library is one domain's on-disk experience store. Retrieval is by
// construction: a domain directory never holds another domain's entries.
// Adopted matter is a CAS hash, not a copied tree — CAS already is the tree.
type Library struct {
	Root   string
	Domain string
}

// DomainSlug maps a workspace path to one directory name. Separators never
// survive, so a domain cannot escape its folder.
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

func taskSlug(taskID string) string {
	s := DomainSlug(taskID)
	if s == libraryUnknown && strings.TrimSpace(taskID) != "" {
		return librarySlugCut.ReplaceAllString(strings.TrimSpace(taskID), "_")
	}
	return s
}

func (l *Library) Dir() string {
	if l == nil {
		return ""
	}
	return filepath.Join(l.Root, DomainSlug(l.Domain))
}

func (l *Library) IndexPath() string {
	return filepath.Join(l.Dir(), "INDEX.md")
}

func (l *Library) entryDir(taskID string) string {
	return filepath.Join(l.Dir(), "entries", taskSlug(taskID))
}

// LibraryEntry is one on-disk experience row for the operator browser.
type LibraryEntry struct {
	ID       string `json:"id"`
	Cue      string `json:"cue,omitempty"`
	Lesson   string `json:"lesson,omitempty"`
	Evidence string `json:"evidence,omitempty"`
	Hash     string `json:"hash,omitempty"`
}

// LibraryView is the read-only INDEX for one domain.
type LibraryView struct {
	Domain  string         `json:"domain"`
	Index   string         `json:"index,omitempty"`
	Entries []LibraryEntry `json:"entries,omitempty"`
}

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

// Context is the improver-facing INDEX.md, or empty when this domain has
// never stored a complete experience. Held-out ids must already be absent.
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

// Add stores one adopted snapshot hash plus a complete experience. Incomplete
// rows, banned grader tokens, or held-out identities are NO_ENTRY (false, nil).
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

func experienceField(path, name string) string {
	for _, line := range strings.Split(readSmall(path), "\n") {
		pref := "- **" + name + "**:"
		if strings.HasPrefix(line, pref) {
			return strings.TrimSpace(strings.TrimPrefix(line, pref))
		}
	}
	return ""
}

func readSmall(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

// SanitizeExperience strips held-out identities. Emptying a required field
// becomes NO_ENTRY at the caller.
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

func experienceBanned(exp Experience) bool {
	blob := strings.ToLower(exp.Cue + "\n" + exp.Lesson + "\n" + exp.Evidence + "\n" + exp.Edits + "\n" + renderEditTable(exp.Verdicts))
	for _, w := range libraryBanned {
		if strings.Contains(blob, w) {
			return true
		}
	}
	return false
}

// WriteL1 expands an adopted CAS snapshot into L1 files under the entry's
// harness/ directory. Playbook and prompt fragments are never written.
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
