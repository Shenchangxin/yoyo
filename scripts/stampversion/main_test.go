package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFourPart(t *testing.T) {
	if got := fourPart("1.2.3"); got != "1.2.3.0" {
		t.Fatalf("fourPart: %s", got)
	}
}

func TestStampManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.manifest")
	src := `<assembly><assemblyIdentity type="win32" name="com.yoyo.agent" version="0.1.0.0" processorArchitecture="*"/></assembly>`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stampManifest(path, "0.2.0.0"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `version="0.2.0.0"`) {
		t.Fatalf("manifest: %s", got)
	}
	if strings.Contains(string(got), `version="0.1.0.0"`) {
		t.Fatal("old version still present")
	}
}

func TestStampPlistAndNFPM(t *testing.T) {
	dir := t.TempDir()
	plist := filepath.Join(dir, "Info.plist")
	body := `<?xml version="1.0"?>
<plist><dict>
	<key>CFBundleShortVersionString</key>
	<string>0.1.0</string>
	<key>CFBundleVersion</key>
	<string>0.1.0</string>
</dict></plist>
`
	if err := os.WriteFile(plist, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stampPlist(plist, "0.2.0"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(plist)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "<string>0.2.0</string>") {
		t.Fatalf("plist not stamped: %s", got)
	}

	nfpm := filepath.Join(dir, "nfpm.yaml")
	if err := os.WriteFile(nfpm, []byte("name: Yoyo\nversion: \"0.1.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stampNFPM(nfpm, "0.2.0-beta.1"); err != nil {
		t.Fatal(err)
	}
	n, err := os.ReadFile(nfpm)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(n), `version: "0.2.0-beta.1"`) {
		t.Fatalf("nfpm not stamped: %s", n)
	}
}

func TestStampNSIS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wails_tools.nsh")
	src := "!ifndef INFO_PRODUCTVERSION\n    !define INFO_PRODUCTVERSION \"0.1.0\"\n!endif\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stampNSIS(path, "0.2.0"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `!define INFO_PRODUCTVERSION "0.2.0"`) {
		t.Fatalf("nsis not stamped: %s", got)
	}
}
