package hostopen

import (
	"runtime"
	"strings"
	"testing"
)

func TestResolveURLHTTP(t *testing.T) {
	got, file, err := resolveURL("https://example.com/x?q=1")
	if err != nil {
		t.Fatal(err)
	}
	if file {
		t.Fatal("https should not be treated as a file")
	}
	if got != "https://example.com/x?q=1" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveURLRejects(t *testing.T) {
	for _, raw := range []string{
		"",
		"example.com",
		"javascript:alert(1)",
		"data:text/html,hi",
		"ftp://files.example",
		"http://",
	} {
		if _, _, err := resolveURL(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

func TestResolveURLFile(t *testing.T) {
	got, file, err := resolveURL("file:///C:/tmp/page.html")
	if err != nil {
		t.Fatal(err)
	}
	if !file {
		t.Fatal("file:// should resolve as a path")
	}
	if runtime.GOOS == "windows" {
		if !strings.EqualFold(got, `C:\tmp\page.html`) && !strings.EqualFold(got, `C:/tmp/page.html`) {
			t.Fatalf("windows path %q", got)
		}
		return
	}
	if !strings.Contains(got, "page.html") {
		t.Fatalf("unix path %q", got)
	}
}

func TestURLRejectsUnsafeSchemes(t *testing.T) {
	for _, raw := range []string{"javascript:alert(1)", "data:text/html,x", ""} {
		if err := URL(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}
