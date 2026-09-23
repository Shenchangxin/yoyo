package video

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
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
	if !strings.Contains(p.VaultKey, p.ID) {
		t.Fatalf("vault key should be per-row, got %s", p.VaultKey)
	}
}

func TestUpdateDramaPersistsAspect(t *testing.T) {
	e := testEngine(t)
	d, err := e.CreateDrama(Drama{Title: "画幅", AspectRatio: "9:16"})
	if err != nil {
		t.Fatal(err)
	}
	if d.AspectRatio != "9:16" {
		t.Fatalf("create %s", d.AspectRatio)
	}
	got, err := e.UpdateDrama(Drama{ID: d.ID, AspectRatio: "1:1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.AspectRatio != "1:1" {
		t.Fatalf("update %s", got.AspectRatio)
	}
}

func TestSkipRewriteCopiesSource(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "跳过"})
	ep, _ := e.CreateEpisode(d.ID, "一", "原文就是剧本。")
	if err := e.SkipRewrite(ep.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := e.GetEpisode(ep.ID)
	if got.ScriptContent != "原文就是剧本。" {
		t.Fatalf("script %q", got.ScriptContent)
	}
}

func TestNarratorHasNoStill(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "旁白"})
	ep, _ := e.CreateEpisode(d.ID, "一", "旁白：夜色。")
	out, err := e.SaveCharacters(ep.ID, []Character{{Name: "旁白", Role: "narrator"}, {Name: "林小雨"}})
	if err != nil {
		t.Fatal(err)
	}
	var narrator, face string
	for _, c := range out {
		if IsNarrator(c.Name, c.Role) {
			narrator = c.ID
		} else {
			face = c.ID
		}
	}
	if narrator == "" || face == "" {
		t.Fatalf("chars %+v", out)
	}
	_, _ = e.UpsertProvider(Provider{ServiceType: "image", Provider: "openai", Name: "t", BaseURL: "https://example", Model: "gpt-image-2", IsActive: true}, "k")
	if _, err := e.GenerateAsset("character", narrator, ep.ID); err == nil {
		t.Fatal("narrator should refuse still")
	}
	jobs, err := e.GenerateMissingAssets(ep.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range jobs {
		if j.CharacterID == narrator {
			t.Fatal("missing stills queued narrator")
		}
	}
}

func TestPatchShotUsesProviderDuration(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "时长"})
	ep, _ := e.CreateEpisode(d.ID, "一", "原文")
	p, err := e.UpsertProvider(Provider{ServiceType: "video", Provider: "aliyun", Name: "wan", BaseURL: "https://example", Model: "wan3.0-video", IsActive: true}, "k")
	if err != nil {
		t.Fatal(err)
	}
	ep.VideoProviderID = p.ID
	_, _ = e.UpdateEpisode(ep)
	shots, err := e.SaveShots(ep.ID, []Shot{{Title: "过渡", ShotType: "transition", Duration: 8}}, true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.PatchShot(shots[0].ID, map[string]any{"duration": 8})
	if err != nil {
		t.Fatal(err)
	}
	if got.Duration != 8 {
		t.Fatalf("wan 8s became %d", got.Duration)
	}
}

func TestImportHuobaoColumnNamesAndLinks(t *testing.T) {
	e := testEngine(t)
	srcPath := filepath.Join(t.TempDir(), "huobao.sqlite")
	src, err := sql.Open("sqlite", srcPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = src.Exec(`
CREATE TABLE dramas (id INTEGER PRIMARY KEY, title TEXT, description TEXT, genre TEXT, style TEXT, aspect_ratio TEXT, deleted_at TEXT);
CREATE TABLE episodes (id INTEGER PRIMARY KEY, drama_id INTEGER, episode_number INTEGER, title TEXT, content TEXT, script_content TEXT, resolution TEXT, video_url TEXT, deleted_at TEXT);
CREATE TABLE characters (id INTEGER PRIMARY KEY, drama_id INTEGER, name TEXT, role TEXT, appearance TEXT, styling TEXT, final_prompt TEXT, image_url TEXT, deleted_at TEXT);
CREATE TABLE scenes (id INTEGER PRIMARY KEY, drama_id INTEGER, location TEXT, time TEXT, prompt TEXT, lighting TEXT, final_prompt TEXT, image_url TEXT, deleted_at TEXT);
CREATE TABLE props (id INTEGER PRIMARY KEY, drama_id INTEGER, name TEXT, type TEXT, description TEXT, final_prompt TEXT, image_url TEXT, deleted_at TEXT);
CREATE TABLE episode_characters (episode_id INTEGER, character_id INTEGER);
CREATE TABLE episode_scenes (episode_id INTEGER, scene_id INTEGER);
CREATE TABLE episode_props (episode_id INTEGER, prop_id INTEGER);
CREATE TABLE storyboards (id INTEGER PRIMARY KEY, episode_id INTEGER, scene_id INTEGER, storyboard_number INTEGER, title TEXT, shot_type TEXT, description TEXT, video_prompt TEXT, duration INTEGER, video_url TEXT, deleted_at TEXT);
CREATE TABLE storyboard_characters (storyboard_id INTEGER, character_id INTEGER);
CREATE TABLE storyboard_props (storyboard_id INTEGER, prop_id INTEGER);
INSERT INTO dramas(id,title,description,genre,style,aspect_ratio,deleted_at) VALUES(1,'雨巷','','','3d','9:16','');
INSERT INTO episodes(id,drama_id,episode_number,title,content,script_content,resolution,video_url,deleted_at) VALUES(1,1,1,'第一集','原文','剧本','720p','','');
INSERT INTO characters(id,drama_id,name,role,appearance,styling,final_prompt,image_url,deleted_at) VALUES(1,1,'林小雨','主角','短发','大衣','','','');
INSERT INTO characters(id,drama_id,name,role,appearance,styling,final_prompt,image_url,deleted_at) VALUES(2,1,'旁白','narrator','','','','','');
INSERT INTO scenes(id,drama_id,location,time,prompt,lighting,final_prompt,image_url,deleted_at) VALUES(1,1,'巷口','夜','湿','路灯','','','');
INSERT INTO episode_characters(episode_id,character_id) VALUES(1,1);
INSERT INTO episode_scenes(episode_id,scene_id) VALUES(1,1);
INSERT INTO storyboards(id,episode_id,scene_id,storyboard_number,title,shot_type,description,video_prompt,duration,video_url,deleted_at)
 VALUES(1,1,1,1,'等车','narrative','【镜头1】等车','@林小雨 回头',8,'','');
INSERT INTO storyboard_characters(storyboard_id,character_id) VALUES(1,1);
`)
	if err != nil {
		t.Fatal(err)
	}
	src.Close()
	out, err := e.ImportHuobao(srcPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if out["dramas"] != 1 || out["episodes"] != 1 {
		t.Fatalf("%+v", out)
	}
	list, _ := e.ListDramas()
	if list[0].AspectRatio != "9:16" {
		t.Fatalf("aspect %s", list[0].AspectRatio)
	}
	eps, _ := e.ListEpisodes(list[0].ID)
	b, err := e.Bundle(eps[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	linkedChars := 0
	for _, c := range b.Characters {
		if c.Linked {
			linkedChars++
		}
	}
	if linkedChars != 1 {
		t.Fatalf("linked chars %d (narrator should stay unlinked)", linkedChars)
	}
	if len(b.Shots) != 1 || b.Shots[0].Duration < 8 {
		t.Fatalf("shots %+v", b.Shots)
	}
	if b.Shots[0].SceneID == "" || len(b.Shots[0].CharacterIDs) != 1 {
		t.Fatalf("bindings %+v", b.Shots[0])
	}
}

func pipelineStatus(raw, stage string) string {
	m := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &m)
	st, _ := m[stage].(map[string]any)
	if st == nil {
		return ""
	}
	return fmt.Sprint(st["status"])
}

func TestWriteBackMarksAssetsDone(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "完成"})
	ep, _ := e.CreateEpisode(d.ID, "一", "文")
	chars, err := e.SaveCharacters(ep.ID, []Character{{Name: "林小雨"}})
	if err != nil || len(chars) == 0 {
		t.Fatalf("chars %v %+v", err, chars)
	}
	h, err := e.PutBytes([]byte("still"))
	if err != nil {
		t.Fatal(err)
	}
	j := Job{ID: NewID(), Type: "image", Status: "succeeded", ResultHash: h, EpisodeID: ep.ID, CharacterID: chars[0].ID}
	if err := e.writeBack(j); err != nil {
		t.Fatal(err)
	}
	got, _ := e.GetEpisode(ep.ID)
	if pipelineStatus(got.Pipeline, "assets") != "done" {
		t.Fatalf("pipeline %s", got.Pipeline)
	}
}

func TestBundleHydratesProviders(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "适配器"})
	ep, _ := e.CreateEpisode(d.ID, "一", "文")
	if ep.ImageProviderID != "" {
		t.Fatalf("created with provider %s", ep.ImageProviderID)
	}
	p, err := e.UpsertProvider(Provider{ServiceType: "image", Provider: "openai", Name: "t", BaseURL: "https://example", Model: "gpt-image-2", IsActive: true, IsDefault: true}, "k")
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.Bundle(ep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Episode.ImageProviderID != p.ID {
		t.Fatalf("hydrate %q want %s", b.Episode.ImageProviderID, p.ID)
	}
	if b.Episode.ImageModel != "gpt-image-2" {
		t.Fatalf("model %q", b.Episode.ImageModel)
	}
}

func TestGenerateMissingShotsContinues(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "批量镜头"})
	ep, _ := e.CreateEpisode(d.ID, "一", "文")
	_, err := e.UpsertProvider(Provider{ServiceType: "video", Provider: "aliyun", Name: "wan", BaseURL: "https://example", Model: "wan3.0-video", IsActive: true}, "k")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.SaveShots(ep.ID, []Shot{
		{Title: "空", ShotType: "narrative"},
		{Title: "有词", ShotType: "narrative", VideoPrompt: "@林小雨 回头", Duration: 8},
	}, true); err != nil {
		t.Fatal(err)
	}
	jobs, err := e.GenerateMissingShots(ep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("jobs %d", len(jobs))
	}
}

func TestPatchEpisodeKeepsContent(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "补丁"})
	ep, _ := e.CreateEpisode(d.ID, "一", "原文不该丢")
	p, err := e.UpsertProvider(Provider{ServiceType: "image", Provider: "openai", Name: "OpenAI", BaseURL: "https://example", Model: "gpt-image-2", Models: `["gpt-image-1","gpt-image-2"]`, IsActive: true}, "k")
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.PatchEpisode(ep.ID, map[string]any{"image_provider_id": p.ID, "image_model": "gpt-image-1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "原文不该丢" {
		t.Fatalf("content %q", got.Content)
	}
	if got.ImageProviderID != p.ID || got.ImageModel != "gpt-image-1" {
		t.Fatalf("pick %+v", got)
	}
}

func TestEpisodePicksProviderModel(t *testing.T) {
	e := testEngine(t)
	d, _ := e.CreateDrama(Drama{Title: "多模型"})
	ep, _ := e.CreateEpisode(d.ID, "一", "文")
	def, err := e.UpsertProvider(Provider{ServiceType: "image", Provider: "openai", Name: "OpenAI", BaseURL: "https://a.example", Model: "gpt-image-2", Models: `["gpt-image-1","gpt-image-2"]`, IsActive: true, IsDefault: true}, "k1")
	if err != nil {
		t.Fatal(err)
	}
	alt, err := e.UpsertProvider(Provider{ServiceType: "image", Provider: "volcengine", Name: "Seedream", BaseURL: "https://b.example", Model: "doubao-seedream-5-0-260128", Models: `["doubao-seedream-5-0-260128","doubao-seedream-4-0-250828"]`, IsActive: true}, "k2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.PatchEpisode(ep.ID, map[string]any{"image_provider_id": alt.ID, "image_model": "doubao-seedream-4-0-250828"}); err != nil {
		t.Fatal(err)
	}
	chars, err := e.SaveCharacters(ep.ID, []Character{{Name: "林小雨"}})
	if err != nil || len(chars) == 0 {
		t.Fatalf("chars %v", err)
	}
	j, err := e.GenerateAsset("character", chars[0].ID, ep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if j.Provider != "volcengine" || j.Model != "doubao-seedream-4-0-250828" {
		t.Fatalf("job provider=%s model=%s", j.Provider, j.Model)
	}
	if j.VaultKey != alt.VaultKey {
		t.Fatalf("vault %s want %s (default was %s)", j.VaultKey, alt.VaultKey, def.VaultKey)
	}
}

func TestSecondAdapterKeepsOwnName(t *testing.T) {
	e := testEngine(t)
	a, err := e.UpsertProvider(Provider{ServiceType: "image", Provider: "openai", Name: "OpenAI Image", BaseURL: "https://a", Model: "gpt-image-2", IsActive: true}, "k1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.UpsertProvider(Provider{ServiceType: "image", Provider: "openai", Name: "OpenAI Image", BaseURL: "https://b", Model: "gpt-image-1", IsActive: true}, "k2")
	if err != nil {
		t.Fatal(err)
	}
	if a.Name == b.Name {
		t.Fatalf("names collided %q", a.Name)
	}
	if a.VaultKey == b.VaultKey {
		t.Fatalf("shared vault %s", a.VaultKey)
	}
}
