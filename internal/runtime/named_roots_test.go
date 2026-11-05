package runtime

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNamedRootsKeepsNamedDirectories(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("drive paths")
	}
	text := "路径：C:\\Users\\scx\\Desktop\\game\\gugu-mart\n文件：`C:\\Users\\scx\\Desktop\\game\\gugu-mart\\concepts\\CONCEPT.md`"
	roots := NamedRoots(text, `C:\Users\scx\.yoyo`)
	wantDir := `C:\Users\scx\Desktop\game\gugu-mart`
	wantFileDir := `C:\Users\scx\Desktop\game\gugu-mart\concepts`
	if !hasRoot(roots, wantDir) || !hasRoot(roots, wantFileDir) {
		t.Fatalf("%v", roots)
	}
}

func TestNamedRootsDropsShallowHomeAndYoyo(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("drive paths")
	}
	text := `C:\ C:\Users C:\Users\scx\.yoyo\tickets\tkt-4 C:\Users\scx\Desktop\game\gugu-mart`
	roots := NamedRoots(text, `C:\Users\scx\.yoyo`)
	if len(roots) != 1 || !hasRoot(roots, `C:\Users\scx\Desktop\game\gugu-mart`) {
		t.Fatalf("%v", roots)
	}
}

func TestWriteFileNamedRoot(t *testing.T) {
	ws := t.TempDir()
	root := filepath.Join(t.TempDir(), "proj")
	tools := &WorkspaceTools{Workspace: ws, Roots: []string{root}}
	res := tools.writeFile(filepath.Join(root, "design", "A.md"), "hello")
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	body, err := os.ReadFile(filepath.Join(root, "design", "A.md"))
	if err != nil || string(body) != "hello" {
		t.Fatalf("%q %v", body, err)
	}
	outside := filepath.Join(filepath.Dir(root), "nope.txt")
	if res = tools.writeFile(outside, "x"); res.Err == nil {
		t.Fatal("wrote outside the named root")
	}
}

func hasRoot(roots []string, want string) bool {
	for _, root := range roots {
		if strings.EqualFold(root, want) {
			return true
		}
	}
	return false
}
