package video

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testCastBody() []byte {
	return []byte{0xff, 0xd8, 0xff, 0xd9}
}

func testCastAsset(file string, body []byte) characterCastAsset {
	sum := sha256.Sum256(body)
	return characterCastAsset{
		ID: "studio-cast-test-one", File: file, Title: "东亚幼年·测试", Note: "测试设定图。",
		Tags: []string{"角色", "测试"}, Category: "character", FolderID: "studio-folder-characters",
		Kind: "image", Width: 8, Height: 8, Bytes: len(body), MIME: "image/jpeg",
		SHA256: hex.EncodeToString(sum[:]),
	}
}

func testCastManifest(asset characterCastAsset) characterCastManifest {
	return characterCastManifest{
		ID: characterCastPackID, Version: 1, PublicPrefix: characterCastPublicPrefix, Source: "Yoyo 工作室",
		Folder: characterCastFolder{ID: "studio-folder-characters", Name: "角色资产", Position: 2},
		Assets: []characterCastAsset{asset},
	}
}

func writeTestCastZip(t *testing.T, man characterCastManifest, files map[string][]byte, extra map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "character-cast.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	raw, err := json.Marshal(man)
	if err != nil {
		t.Fatal(err)
	}
	mw, err := w.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mw.Write(raw); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range extra {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCharacterCastInstallImportAndServe(t *testing.T) {
	body := testCastBody()
	asset := testCastAsset("cast-test-one.jpg", body)
	zipPath := writeTestCastZip(t, testCastManifest(asset), map[string][]byte{asset.File: body}, nil)
	e := testEngine(t)
	blank := ""
	e.castSHA = &blank
	if err := e.installCharacterCastFile(zipPath); err != nil {
		t.Fatal(err)
	}
	if err := e.importCharacterCastNow(); err != nil {
		t.Fatal(err)
	}
	item, err := e.getAsset(asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if item["title"] != asset.Title || item["folderId"] != "studio-folder-characters" || item["category"] != "character" {
		t.Fatalf("imported asset: %+v", item)
	}
	if item["coverUrl"] != "/character-cast/cast-test-one.jpg" || item["note"] != asset.Note || item["source"] != "Yoyo 工作室" {
		t.Fatalf("cover/note: %+v", item)
	}
	data, _ := item["data"].(map[string]any)
	if data["dataUrl"] != "/character-cast/cast-test-one.jpg" || intAny(data["bytes"]) != len(body) {
		t.Fatalf("data: %+v", data)
	}
	req := httptest.NewRequest(http.MethodGet, "/character-cast/cast-test-one.jpg", nil)
	rec := httptest.NewRecorder()
	CharacterCastHandler(e.CharacterCastDir()).ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.String() != string(body) {
		t.Fatalf("serve %d %q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatal(rec.Header().Get("Content-Type"))
	}
	miss := httptest.NewRecorder()
	nextCalled := false
	CharacterCastMiddleware(e.CharacterCastDir(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(204)
	})).ServeHTTP(miss, httptest.NewRequest(http.MethodGet, "/character-cast/missing.jpg", nil))
	if !nextCalled || miss.Code != 204 {
		t.Fatalf("missing still should fall through, code %d", miss.Code)
	}
	if _, ok := characterCastName("/character-cast/../secret.jpg"); ok {
		t.Fatal("traversal name accepted")
	}
}

func TestCharacterCastZipSlipAndBadHash(t *testing.T) {
	e := testEngine(t)
	blank := ""
	e.castSHA = &blank
	slip := writeTestCastZip(t, characterCastManifest{}, nil, map[string]string{"../evil.jpg": "nope"})
	if err := e.installCharacterCastFile(slip); err == nil {
		t.Fatal("expected zip slip rejection")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(e.CharacterCastDir()), "evil.jpg")); err == nil {
		t.Fatal("zip slip wrote outside the pack dir")
	}
	body := testCastBody()
	asset := testCastAsset("cast-test-one.jpg", body)
	asset.SHA256 = strings.Repeat("ab", 32)
	bad := writeTestCastZip(t, testCastManifest(asset), map[string][]byte{asset.File: body}, nil)
	if err := e.installCharacterCastFile(bad); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("expected hash mismatch, got %v", err)
	}
}

func TestCharacterCastDownloadURLs(t *testing.T) {
	got := characterCastDownloadURLs("https://www.modelscope.ai/datasets/acme/yoyo/resolve/master/character-cast.zip")
	if len(got) != 2 {
		t.Fatalf("urls: %v", got)
	}
	if !strings.Contains(got[0], "?view=false") {
		t.Fatal(got[0])
	}
	if !strings.Contains(got[1], "/api/v1/datasets/acme/yoyo/repo?") || !strings.Contains(got[1], "FilePath=character-cast.zip") || !strings.Contains(got[1], "Revision=master") {
		t.Fatal(got[1])
	}
	plain := characterCastDownloadURLs("https://cdn.example.com/character-cast.zip")
	if len(plain) != 1 || plain[0] != "https://cdn.example.com/character-cast.zip" {
		t.Fatal(plain)
	}
	queried := characterCastDownloadURLs("https://www.modelscope.ai/datasets/acme/yoyo/resolve/master/character-cast.zip?view=false")
	if len(queried) != 2 || strings.Count(queried[0], "view=false") != 1 {
		t.Fatal(queried)
	}
	published := characterCastDownloadURLs(characterCastPackURL)
	if len(published) != 2 || !strings.Contains(published[0], "https://www.modelscope.ai/datasets/yoyo-agent/yoyo/resolve/master/sourcetime/character-cast.zip?view=false") {
		t.Fatal(published)
	}
	if !strings.Contains(published[1], "www.modelscope.ai/api/v1/datasets/yoyo-agent/yoyo/repo?") || !strings.Contains(published[1], "FilePath=sourcetime%2Fcharacter-cast.zip") || !strings.Contains(published[1], "Revision=master") {
		t.Fatal(published[1])
	}
}

func TestGoTestDoesNotFetchPublishedCast(t *testing.T) {
	t.Setenv(envCharacterCastURL, "")
	t.Setenv(envCharacterCastFile, "")
	t.Setenv(envCharacterCastDisable, "")
	kind, value := (*Engine)(nil).characterCastSource()
	if kind != "" || value != "" {
		t.Fatalf("test process would fetch %s %s", kind, value)
	}
}

func TestCharacterCastDownloadAndSchedule(t *testing.T) {
	body := testCastBody()
	asset := testCastAsset("cast-test-one.jpg", body)
	zipPath := writeTestCastZip(t, testCastManifest(asset), map[string][]byte{asset.File: body}, nil)
	payload, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := fileSHA256(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/character-cast.zip" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	e := testEngine(t)
	bad := strings.Repeat("cd", 32)
	e.castSHA = &bad
	e.castHTTP = srv.Client()
	if err := e.downloadAndInstallCharacterCast(context.Background(), srv.URL+"/character-cast.zip"); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("expected pinned mismatch, got %v", err)
	}

	e.castSHA = &sum
	e.castFile = zipPath
	e.StartCharacterCast()
	deadline := time.Now().Add(5 * time.Second)
	for {
		item, err := e.getAsset(asset.ID)
		if err == nil && item["title"] == asset.Title && item["coverUrl"] == "/character-cast/"+asset.File {
			meta, err := readCastMeta(e.CharacterCastDir())
			if err != nil {
				t.Fatal(err)
			}
			abs, _ := filepath.Abs(zipPath)
			if meta.Source != abs || !strings.EqualFold(meta.SHA256, sum) {
				t.Fatalf("meta %+v", meta)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("scheduled import did not finish")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestPublishedCharacterCastZip(t *testing.T) {
	zipPath := filepath.Join("..", "..", "dist", "character-cast.zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("dist/character-cast.zip is not in this checkout")
	}
	e := testEngine(t)
	if err := e.installCharacterCastFile(zipPath); err != nil {
		t.Fatal(err)
	}
	if err := e.importCharacterCastNow(); err != nil {
		t.Fatal(err)
	}
	page := e.listAssetsPage("", "", "studio-folder-characters", "active", "", 1, 200, false, "")
	n := 0
	var sample map[string]any
	for _, item := range page["assets"].([]map[string]any) {
		id := item["id"].(string)
		if !strings.HasPrefix(id, "studio-cast-") {
			continue
		}
		n++
		if id == "studio-cast-ea-child-storybook" {
			sample = item
		}
		if item["category"] != "character" || item["folderId"] != "studio-folder-characters" {
			t.Fatalf("asset %s: %+v", id, item)
		}
	}
	if n != 48 {
		t.Fatalf("imported %d character stills", n)
	}
	if sample["title"] != "东亚幼年·纸鸢绘本" || sample["coverUrl"] != "/character-cast/cast-ea-child-storybook.jpg" {
		t.Fatalf("sample: %+v", sample)
	}
	req := httptest.NewRequest(http.MethodGet, "/character-cast/cast-ea-child-storybook.jpg", nil)
	rec := httptest.NewRecorder()
	CharacterCastHandler(e.CharacterCastDir()).ServeHTTP(rec, req)
	if rec.Code != 200 || rec.Body.Len() < 1000 || rec.Body.Bytes()[0] != 0xff {
		t.Fatalf("serve real still %d bytes %d", rec.Code, rec.Body.Len())
	}
}

func TestCharacterCastSourcePrefersFileAndDisable(t *testing.T) {
	e := testEngine(t)
	e.castURL = "https://cdn.example.com/character-cast.zip"
	e.castFile = filepath.Join(t.TempDir(), "local.zip")
	kind, value := e.characterCastSource()
	abs, _ := filepath.Abs(e.castFile)
	if kind != "file" || value != abs {
		t.Fatalf("source %s %s", kind, value)
	}
	t.Setenv(envCharacterCastDisable, "1")
	if kind, value := e.characterCastSource(); kind != "" || value != "" {
		t.Fatalf("disable did not win: %s %s", kind, value)
	}
}
