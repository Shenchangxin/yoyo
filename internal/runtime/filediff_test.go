// 锁住行 diff 对小改动给出增删计数，超长文件仍能定位单行和两端改动，并在比较前去掉单独的 CR。

package runtime

import (
	"strings"
	"testing"
)

// TestUnifiedLineDiffReplaceRegion 锁住一行替换的增删各为 1，补丁含 --- a/ 和对应的减号、加号行。
func TestUnifiedLineDiffReplaceRegion(t *testing.T) {
	old := "func main() {\n\tfmt.Println(\"hi\")\n}\n"
	neu := "func main() {\n\tfmt.Println(\"yo\")\n}\n"
	added, removed, patch := unifiedLineDiff("main.go", old, neu, 40)
	if added != 1 || removed != 1 {
		t.Fatalf("added=%d removed=%d\n%s", added, removed, patch)
	}
	if !strings.Contains(patch, "--- a/main.go") || !strings.Contains(patch, `-"	fmt.Println("hi")`) {
		if !strings.Contains(patch, `-	fmt.Println("hi")`) {
			t.Fatalf("patch %s", patch)
		}
	}
	if !strings.Contains(patch, `+	fmt.Println("yo")`) {
		t.Fatalf("patch %s", patch)
	}
}

// TestFileChangeOfCreatedFile 锁住新建文件的 Created 为真、有新增、没有删除，补丁含加号行。
func TestFileChangeOfCreatedFile(t *testing.T) {
	ch := fileChangeOf("web/app.ts", nil, []byte("export const n = 1\n"), true)
	if !ch.Created || ch.Added < 1 || ch.Removed != 0 {
		t.Fatalf("%+v", ch)
	}
	if !strings.Contains(ch.Patch, "+export const n = 1") {
		t.Fatalf("patch %s", ch.Patch)
	}
}

// TestUnifiedLineDiffLargeFileOneLine 锁住 1236 行里只改一行时，增删计数仍各为 1。
func TestUnifiedLineDiffLargeFileOneLine(t *testing.T) {
	const n = 1236
	old := make([]string, n)
	neu := make([]string, n)
	for i := 0; i < n; i++ {
		old[i] = strings.Repeat("x", 8) + " line " + itoa(i)
		neu[i] = old[i]
	}
	neu[600] = "changed"
	added, removed, patch := unifiedLineDiff("big.go", strings.Join(old, "\n")+"\n", strings.Join(neu, "\n")+"\n", 40)
	if added != 1 || removed != 1 {
		t.Fatalf("one-line edit in %d-line file: added=%d removed=%d\n%s", n, added, removed, patch)
	}
}

// TestUnifiedLineDiffStripsBareCR 锁住 CRLF 和 LF 在去掉 CR 后增删都是 0。
func TestUnifiedLineDiffStripsBareCR(t *testing.T) {
	old := "a\r\nb\r\nc\n"
	neu := "a\nb\nc\n"
	added, removed, _ := unifiedLineDiff("cr.txt", old, neu, 40)
	if added != 0 || removed != 0 {
		t.Fatalf("CRLF vs LF after strip: added=%d removed=%d", added, removed)
	}
}

// TestUnifiedLineDiffLargeFileTwoEnds 锁住首尾两处改动的增删各为 2。
func TestUnifiedLineDiffLargeFileTwoEnds(t *testing.T) {
	const n = 1236
	old := make([]string, n)
	neu := make([]string, n)
	for i := 0; i < n; i++ {
		old[i] = "unique-" + itoa(i)
		neu[i] = old[i]
	}
	neu[0] = "head"
	neu[n-1] = "tail"
	added, removed, _ := unifiedLineDiff("big.go", strings.Join(old, "\n")+"\n", strings.Join(neu, "\n")+"\n", 40)
	if added != 2 || removed != 2 {
		t.Fatalf("two distant edits: added=%d removed=%d", added, removed)
	}
}
