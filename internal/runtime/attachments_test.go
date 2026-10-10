package runtime

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImagePartsUsesMultimodalNotBinaryDump(t *testing.T) {
	raw := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 1, 2, 3, 4}
	b64 := base64.StdEncoding.EncodeToString(raw)
	atts := []Attachment{{Name: "shot.png", MIME: "image/png", DataB64: b64}}
	text := ExpandAttachments("", atts, 2400)
	if strings.Contains(text, "binary ") {
		t.Fatalf("image should not dump as binary: %s", text)
	}
	if !strings.Contains(text, "multimodal") {
		t.Fatalf("want multimodal note, got %s", text)
	}
	parts := ImageParts("", atts)
	if len(parts) != 1 || parts[0].Type != "image_url" || !strings.HasPrefix(parts[0].ImageURL, "data:image/png") {
		t.Fatalf("%+v", parts)
	}
}

func TestMaterializeOutsideMarkdown(t *testing.T) {
	ws := t.TempDir()
	outside := filepath.Join(t.TempDir(), "我的小说.md")
	body := "# 第一章\n很长的小说正文\n"
	if err := os.WriteFile(outside, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	atts, err := MaterializeAttachments(ws, []Attachment{{Path: outside, Name: "我的小说.md", MIME: "text/markdown"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(atts) != 1 {
		t.Fatalf("atts=%d", len(atts))
	}
	if atts[0].DataB64 != "" {
		t.Fatalf("expected bytes cleared, got %d", len(atts[0].DataB64))
	}
	if !strings.HasPrefix(atts[0].Path, ".yoyo/uploads/") {
		t.Fatalf("path %q", atts[0].Path)
	}
	raw, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(atts[0].Path)))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != body {
		t.Fatalf("copied %q", raw)
	}
	msg := StampFileMentions("@skill:novel-to-game", atts)
	if !strings.Contains(msg, "@skill:novel-to-game") || !strings.Contains(msg, "@file:.yoyo/uploads/") {
		t.Fatalf("stamp %q", msg)
	}
	inject := ExpandAttachments(ws, atts, 2400)
	if strings.Contains(inject, "escapes") {
		t.Fatalf("jail after import: %s", inject)
	}
	if !strings.Contains(inject, "第一章") {
		t.Fatalf("preview missing: %s", inject)
	}
}

func TestMaterializeKeepsWorkspaceRelative(t *testing.T) {
	ws := t.TempDir()
	src := filepath.Join(ws, "notes.md")
	if err := os.WriteFile(src, []byte("already inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	atts, err := MaterializeAttachments(ws, []Attachment{{Path: src, Name: "notes.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if atts[0].Path != "notes.md" {
		t.Fatalf("path %q", atts[0].Path)
	}
	if _, err := os.Stat(filepath.Join(ws, ".yoyo", "uploads")); !os.IsNotExist(err) {
		t.Fatalf("should not copy in-workspace file: %v", err)
	}
}

func TestMaterializeDataB64Text(t *testing.T) {
	ws := t.TempDir()
	body := "drag-dropped novel"
	b64 := base64.StdEncoding.EncodeToString([]byte(body))
	atts, err := MaterializeAttachments(ws, []Attachment{{Name: "drop.md", MIME: "text/markdown", DataB64: b64}})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(atts[0].Path)))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != body {
		t.Fatalf("got %q", raw)
	}
}

func TestMaterializeUniqueNames(t *testing.T) {
	ws := t.TempDir()
	outside := t.TempDir()
	a := filepath.Join(outside, "same.md")
	b := filepath.Join(outside, "other.md")
	if err := os.WriteFile(a, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := MaterializeAttachments(ws, []Attachment{{Path: a, Name: "same.md"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := MaterializeAttachments(ws, []Attachment{{Path: b, Name: "same.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if first[0].Path == second[0].Path {
		t.Fatalf("collision %q", first[0].Path)
	}
	if !strings.Contains(second[0].Path, "same-2.md") {
		t.Fatalf("second %q", second[0].Path)
	}
}

func TestStampFileMentionsSkipsImagesAndDupes(t *testing.T) {
	msg := StampFileMentions("@file:.yoyo/uploads/a.md please", []Attachment{
		{Path: ".yoyo/uploads/a.md", Name: "a.md"},
		{Path: "shot.png", Name: "shot.png", MIME: "image/png"},
	})
	if strings.Count(msg, "@file:") != 1 {
		t.Fatalf("duped: %q", msg)
	}
	if strings.Contains(msg, "shot.png") {
		t.Fatalf("stamped image: %q", msg)
	}
}
