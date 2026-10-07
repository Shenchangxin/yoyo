package runtime

import "testing"

func TestParseUnixFindGlob(t *testing.T) {
	pat, ok := parseUnixFindGlob(`find . -name "*.pdf"`)
	if !ok || pat != "*.pdf" {
		t.Fatalf("got %q %v", pat, ok)
	}
	pat, ok = parseUnixFindGlob(`find src -type f -name '*.go'`)
	if !ok || pat != "src/**/*.go" {
		t.Fatalf("got %q %v", pat, ok)
	}
	if _, ok := parseUnixFindGlob(`find "needle" notes.txt`); ok {
		t.Fatal("windows FIND text search must not rewrite")
	}
	if !unixFindMisuse(`find . -type f`) {
		t.Fatal("expected unix find misuse")
	}
	if unixFindMisuse(`find "needle" notes.txt`) {
		t.Fatal("text search is valid FIND.exe")
	}
}
