// 锁住行 diff 对小改动给出增删计数，超长文件仍能定位单行和两端改动，并在比较前去掉单独的 CR。

package textdiff

import (
	"strconv"
	"strings"
	"testing"
)

// TestUnifiedReplaceRegion 锁住一行替换的增删各为 1，补丁含 --- a/ 和对应的减号、加号行。
func TestUnifiedReplaceRegion(t *testing.T) {
	old := "func main() {\n\tfmt.Println(\"hi\")\n}\n"
	neu := "func main() {\n\tfmt.Println(\"yo\")\n}\n"
	added, removed, patch := Unified("main.go", old, neu, 40)
	if added != 1 || removed != 1 {
		t.Fatalf("added=%d removed=%d\n%s", added, removed, patch)
	}
	if !strings.Contains(patch, "--- a/main.go") || !strings.Contains(patch, `-	fmt.Println("hi")`) {
		t.Fatalf("patch %s", patch)
	}
	if !strings.Contains(patch, `+	fmt.Println("yo")`) {
		t.Fatalf("patch %s", patch)
	}
}

// TestUnifiedLargeFileOneLine 锁住 1236 行里只改一行时，增删计数仍各为 1。
func TestUnifiedLargeFileOneLine(t *testing.T) {
	const n = 1236
	old := make([]string, n)
	neu := make([]string, n)
	for i := 0; i < n; i++ {
		old[i] = strings.Repeat("x", 8) + " line " + strconv.Itoa(i)
		neu[i] = old[i]
	}
	neu[600] = "changed"
	added, removed, patch := Unified("big.go", strings.Join(old, "\n")+"\n", strings.Join(neu, "\n")+"\n", 40)
	if added != 1 || removed != 1 {
		t.Fatalf("one-line edit in %d-line file: added=%d removed=%d\n%s", n, added, removed, patch)
	}
}

// TestUnifiedStripsBareCR 锁住 CRLF 和 LF 在去掉 CR 后增删都是 0。
func TestUnifiedStripsBareCR(t *testing.T) {
	old := "a\r\nb\r\nc\n"
	neu := "a\nb\nc\n"
	added, removed, _ := Unified("cr.txt", old, neu, 40)
	if added != 0 || removed != 0 {
		t.Fatalf("CRLF vs LF after strip: added=%d removed=%d", added, removed)
	}
}

// TestUnifiedLargeFileTwoEnds 锁住首尾两处改动的增删各为 2。
func TestUnifiedLargeFileTwoEnds(t *testing.T) {
	const n = 1236
	old := make([]string, n)
	neu := make([]string, n)
	for i := 0; i < n; i++ {
		old[i] = "unique-" + strconv.Itoa(i)
		neu[i] = old[i]
	}
	neu[0] = "head"
	neu[n-1] = "tail"
	added, removed, _ := Unified("big.go", strings.Join(old, "\n")+"\n", strings.Join(neu, "\n")+"\n", 40)
	if added != 2 || removed != 2 {
		t.Fatalf("two distant edits: added=%d removed=%d", added, removed)
	}
}

// TestUnifiedDefaultsMaxHunk 锁住 maxHunk 不大于 0 时按 DefaultMaxHunk 截断，并追加截断标记。
func TestUnifiedDefaultsMaxHunk(t *testing.T) {
	var neu strings.Builder
	for i := 0; i < DefaultMaxHunk+50; i++ {
		neu.WriteString("line " + strconv.Itoa(i) + "\n")
	}
	added, _, patch := Unified("new.txt", "", neu.String(), 0)
	if added != DefaultMaxHunk+50 {
		t.Fatalf("added=%d", added)
	}
	if !strings.HasSuffix(patch, "@@ … @@") {
		t.Fatalf("expected truncation marker, got tail %q", patch[len(patch)-40:])
	}
}
