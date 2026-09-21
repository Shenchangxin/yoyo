package video

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type memCAS struct {
	m map[string][]byte
}

func (m *memCAS) PutRaw(b []byte) (string, error) {
	if m.m == nil {
		m.m = map[string][]byte{}
	}
	h := NewID()
	m.m[h] = append([]byte(nil), b...)
	return h, nil
}
func (m *memCAS) GetRaw(hash string) ([]byte, error) {
	b, ok := m.m[hash]
	if !ok {
		return nil, os.ErrNotExist
	}
	return b, nil
}
func (m *memCAS) Path(hash string) string { return filepath.Join(os.TempDir(), hash) }

type memVault struct{ m map[string]string }

func (v *memVault) Lease(name string) (string, error) { return v.Get(name) }
func (v *memVault) Set(name, value string) {
	if v.m == nil {
		v.m = map[string]string{}
	}
	v.m[name] = value
}
func (v *memVault) Get(name string) (string, error) {
	if v.m == nil || v.m[name] == "" {
		return "", os.ErrNotExist
	}
	return v.m[name], nil
}

func testEngine(t *testing.T) *Engine {
	t.Helper()
	dir := t.TempDir()
	e, err := Open(dir, &memCAS{}, &memVault{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return e
}

func TestDramaCRUDAndDedupe(t *testing.T) {
	e := testEngine(t)
	d, err := e.CreateDrama(Drama{Title: "雨巷", Style: "3d", AspectRatio: "16:9"})
	if err != nil {
		t.Fatal(err)
	}
	ep, err := e.CreateEpisode(d.ID, "第一集", "林小雨在巷口等车。")
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.SaveCharacters(ep.ID, []Character{{Name: "林小雨（主角）", Appearance: "短发"}})
	if err != nil {
		t.Fatal(err)
	}
	out, err := e.SaveCharacters(ep.ID, []Character{{Name: "林小雨", Appearance: "黑大衣"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("dedupe got %d", len(out))
	}
	if out[0].Appearance != "黑大衣" {
		t.Fatalf("merge appearance %q", out[0].Appearance)
	}
	if err := e.SaveScript(ep.ID, "INT. 巷口 - NIGHT\n林小雨等车。"); err != nil {
		t.Fatal(err)
	}
	shots, err := e.SaveShots(ep.ID, []Shot{{Title: "等车", Description: "广角空镜后切近景。", Duration: 12, CharacterIDs: []string{"林小雨"}}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(shots) != 1 || len(shots[0].CharacterIDs) != 1 {
		t.Fatalf("shots %+v", shots)
	}
	b, err := e.Bundle(ep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Plan.Chars < 1 || len(b.Characters) != 1 {
		t.Fatalf("bundle %+v", b)
	}
}

func TestJobResumeQueued(t *testing.T) {
	e := testEngine(t)
	now := Now()
	j := Job{ID: NewID(), Type: "image", Status: "running", CreatedAt: now, UpdatedAt: now}
	if err := e.insertJob(j); err != nil {
		t.Fatal(err)
	}
	e.resumeJobs()
	got, err := e.GetJob(j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "queued" {
		t.Fatalf("status %s", got.Status)
	}
}

func TestJobResumeKeepsRemotePoll(t *testing.T) {
	e := testEngine(t)
	now := Now()
	j := Job{ID: NewID(), Type: "video", Status: "running", RemoteID: "task-1", CreatedAt: now, UpdatedAt: now}
	if err := e.insertJob(j); err != nil {
		t.Fatal(err)
	}
	e.resumeJobs()
	got, err := e.GetJob(j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "polling" {
		t.Fatalf("status %s", got.Status)
	}
}

func TestShotRefsSceneThenCastThenProp(t *testing.T) {
	e := testEngine(t)
	d, err := e.CreateDrama(Drama{Title: "槽序", Style: "3d", AspectRatio: "16:9"})
	if err != nil {
		t.Fatal(err)
	}
	ep, err := e.CreateEpisode(d.ID, "一", "原文")
	if err != nil {
		t.Fatal(err)
	}
	jpg := tinyJPEG()
	ch, err := e.PutBytes(jpg)
	if err != nil {
		t.Fatal(err)
	}
	sh, err := e.PutBytes(jpg)
	if err != nil {
		t.Fatal(err)
	}
	ph, err := e.PutBytes(jpg)
	if err != nil {
		t.Fatal(err)
	}
	chars, err := e.SaveCharacters(ep.ID, []Character{{Name: "林小雨", ImageHash: ch}})
	if err != nil {
		t.Fatal(err)
	}
	scenes, err := e.SaveScenes(ep.ID, []Scene{{Location: "巷口", ImageHash: sh}})
	if err != nil {
		t.Fatal(err)
	}
	props, err := e.SaveProps(ep.ID, []Prop{{Name: "油纸伞", ImageHash: ph}})
	if err != nil {
		t.Fatal(err)
	}
	refs, err := e.ShotRefs(Shot{SceneID: scenes[0].ID, CharacterIDs: []string{chars[0].ID}, PropIDs: []string{props[0].ID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 3 {
		t.Fatalf("refs %d", len(refs))
	}
	if refs[0].Name != "巷口" || refs[1].Name != "林小雨" || refs[2].Name != "油纸伞" {
		t.Fatalf("order %+v", refs)
	}
	prompt, urls := ResolvePromptRefs("@巷口 @林小雨 撑开 @油纸伞", refs, false)
	if len(urls) != 3 {
		t.Fatalf("urls %d", len(urls))
	}
	if !strings.Contains(prompt, "@图片1巷口") || !strings.Contains(prompt, "@图片2林小雨") || !strings.Contains(prompt, "@图片3油纸伞") {
		t.Fatalf("prompt %q", prompt)
	}
}

func tinyJPEG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 70})
	return buf.Bytes()
}

func TestProviderKeyStaysInVault(t *testing.T) {
	e := testEngine(t)
	v := e.Vault.(*memVault)
	p, err := e.UpsertProvider(Provider{ServiceType: "image", Provider: "openai", Name: "t", BaseURL: "https://api.openai.com", Model: "gpt-image-1"}, "sk-secret")
	if err != nil {
		t.Fatal(err)
	}
	if p.VaultKey == "" || !p.HasKey {
		t.Fatalf("provider %+v", p)
	}
	if v.m[p.VaultKey] != "sk-secret" {
		t.Fatal("key not in vault")
	}
	var leaked string
	_ = e.DB.QueryRow(`SELECT settings FROM providers WHERE id = ?`, p.ID).Scan(&leaked)
	if leaked == "sk-secret" {
		t.Fatal("key leaked into settings")
	}
}
