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
