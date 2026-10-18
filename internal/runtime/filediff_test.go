package runtime

import (
	"strings"
	"testing"
)

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

func TestFileChangeOfCreatedFile(t *testing.T) {
	ch := fileChangeOf("web/app.ts", nil, []byte("export const n = 1\n"), true)
	if !ch.Created || ch.Added < 1 || ch.Removed != 0 {
		t.Fatalf("%+v", ch)
	}
	if !strings.Contains(ch.Patch, "+export const n = 1") {
		t.Fatalf("patch %s", ch.Patch)
	}
}

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

func TestUnifiedLineDiffStripsBareCR(t *testing.T) {
	old := "a\r\nb\r\nc\n"
	neu := "a\nb\nc\n"
	added, removed, _ := unifiedLineDiff("cr.txt", old, neu, 40)
	if added != 0 || removed != 0 {
		t.Fatalf("CRLF vs LF after strip: added=%d removed=%d", added, removed)
	}
}

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
