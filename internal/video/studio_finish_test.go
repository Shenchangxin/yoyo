package video

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

func TestStillFilterKenBurnsCoversWithoutPadding(t *testing.T) {
	f := stillFilter(1080, 1920, 3, "ken-burns", "cover")
	if strings.Contains(f, "pad=") || !strings.Contains(f, "zoompan=") || !strings.Contains(f, "1.08") {
		t.Fatal(f)
	}
	if !strings.Contains(f, "scale=2160:3840") {
		t.Fatal(f)
	}
}

func TestStillFilterStaticAndFakeMotionsStayPut(t *testing.T) {
	sb := StudioStoryboard{}
	for _, motion := range []string{"static", "push-in", "pan-left", "zoom-out"} {
		norm := normalizeMotion(motion)
		fit := shotFit(sb, StudioShot{Motion: motion}, norm)
		f := stillFilter(1080, 1920, 3, motion, fit)
		if strings.Contains(f, "zoompan=") || !strings.Contains(f, "pad=") {
			t.Fatalf("%s fit %s → %s", motion, fit, f)
		}
	}
}

func TestStillCoverHoldsWithoutZoom(t *testing.T) {
	f := stillFilter(1080, 1920, 3, "static", "cover")
	if strings.Contains(f, "zoompan=") || strings.Contains(f, "pad=") || !strings.Contains(f, "crop=") {
		t.Fatal(f)
	}
}

func TestStillFilterContainSkipsPush(t *testing.T) {
	f := stillFilter(1080, 1920, 3, "ken-burns", "contain")
	if strings.Contains(f, "zoompan=") || !strings.Contains(f, "pad=") {
		t.Fatal(f)
	}
}

func TestClipShouldLoop(t *testing.T) {
	loop, warn := clipShouldLoop(2, 6)
	if !loop || !warn {
		t.Fatalf("loop %v warn %v", loop, warn)
	}
	loop, warn = clipShouldLoop(5, 6)
	if !loop || warn {
		t.Fatalf("short once: loop %v warn %v", loop, warn)
	}
	loop, warn = clipShouldLoop(8, 6)
	if loop || warn {
		t.Fatalf("long: loop %v warn %v", loop, warn)
	}
}

func TestSRTToASSWrapsCJK(t *testing.T) {
	line := strings.Repeat("字", 40)
	srt := "1\n00:00:00,000 --> 00:00:02,000\n" + line + "\n"
	ass := srtToASS(srt, 1080, 1920)
	if !strings.Contains(ass, "PlayResX: 1080") || !strings.Contains(ass, "PlayResY: 1920") {
		t.Fatal(ass)
	}
	if !strings.Contains(ass, `\N`) {
		t.Fatalf("expected wrap, got %s", ass)
	}
}

func TestAlignScriptToCuesKeepsScript(t *testing.T) {
	lines := []string{"第一句。", "第二句。"}
	cues := []cut.Cue{{Start: 0.2, End: 1.1, Text: "错"}, {Start: 1.2, End: 2.4, Text: "误"}}
	got := alignScriptToCues(lines, cues)
	if len(got) != 2 || got[0].Text != "第一句。" || got[0].Start != 0.2 || got[1].End != 2.4 {
		t.Fatalf("%+v", got)
	}
}

func TestSpeechPayloadCarriesSpeed(t *testing.T) {
	body := speechPayload("gpt-4o-mini-tts", "你好", "alloy", 1.25)
	if body["speed"] != 1.25 {
		t.Fatalf("%v", body)
	}
	if _, ok := speechPayload("m", "t", "alloy", 1)["speed"]; ok {
		t.Fatal("unity speed should be omitted")
	}
	if edgeRate(1.2) != "+20%" {
		t.Fatalf("%s", edgeRate(1.2))
	}
}

func TestMixGraphFadesMusicAtEnd(t *testing.T) {
	filter, _ := cut.MixGraph(true, true, cut.MixOpts{Duck: true, MusicVol: 0.22, SpeechVol: 1, FadeOut: 3, Timeline: 12})
	if !strings.Contains(filter, "afade=t=out:st=9.00:d=3.00") || !strings.Contains(filter, "sidechaincompress") {
		t.Fatal(filter)
	}
	plain, _ := cut.MixGraph(true, true, cut.MixOpts{Duck: true, MusicVol: 0.2, SpeechVol: 1})
	if strings.Contains(plain, "afade=") {
		t.Fatal(plain)
	}
}

func TestCleanDraftScript(t *testing.T) {
	got := cleanDraftScript("```\n口播：大家好。记住这一点。\n```")
	if got != "大家好。记住这一点。" {
		t.Fatalf("%q", got)
	}
}

func TestSpeakStudioSendsSpeedAndMarksEstimate(t *testing.T) {
	var saw map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/audio/speech") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&saw)
		_, _ = w.Write(bytes.Repeat([]byte("a"), 64))
	}))
	defer srv.Close()

	e := testEngine(t)
	reg, err := connection.Open(t.TempDir(), e.Vault)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Upsert(connection.Connection{
		ID: "speech", Name: "speech", Vendor: "openai", Endpoint: srv.URL + "/v1",
		Capabilities: []string{connection.CapSpeech}, Active: true,
		DefaultModel: map[string]string{connection.CapSpeech: "gpt-4o-mini-tts"},
	}, "k"); err != nil {
		t.Fatal(err)
	}
	e.Conn = reg
	e.HTTP = srv.Client()

	p, err := e.CreateStudioProject(StudioProject{Title: "语速", Script: "大家好。这是第二句。", VoiceSpeed: 1.25})
	if err != nil {
		t.Fatal(err)
	}
	p.ScriptStatus = ScriptApproved
	p.Storyboard.Shots = []StudioShot{{ID: "s1", Caption: "大家好。", Duration: 3, Motion: "ken-burns"}}
	if _, err := e.saveStudio(p); err != nil {
		t.Fatal(err)
	}
	got, err := e.SpeakStudio(p.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if saw["speed"] != 1.25 {
		t.Fatalf("speed payload %#v", saw["speed"])
	}
	if got.VoiceEngine != "cloud" || !got.SubtitleEstimated || got.NarrationHash == "" {
		t.Fatalf("%+v", got)
	}
	var warned bool
	for _, iss := range got.Issues {
		if iss.Level == "warn" && strings.Contains(iss.Message, "估算") {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("issues %+v", got.Issues)
	}
}

func TestStudioAutoStopsWhenPicturesMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/chat/completions"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []any{map[string]any{"message": map[string]string{"content": "大家好。先看这个变化。记住一件事。现在就试试。"}}},
			})
		case strings.HasSuffix(r.URL.Path, "/audio/speech"):
			_, _ = io.Copy(w, bytes.NewReader(bytes.Repeat([]byte("b"), 64)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	e := testEngine(t)
	reg, err := connection.Open(t.TempDir(), e.Vault)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Upsert(connection.Connection{
		ID: "both", Name: "both", Vendor: "openai", Endpoint: srv.URL + "/v1",
		Capabilities: []string{connection.CapChat, connection.CapSpeech}, Active: true,
		DefaultModel: map[string]string{connection.CapChat: "m", connection.CapSpeech: "tts"},
	}, "k"); err != nil {
		t.Fatal(err)
	}
	e.Conn = reg
	e.HTTP = srv.Client()

	p, err := e.CreateStudioProject(StudioProject{Title: "一键", Audience: "刚入门的人", TargetDuration: 30})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.StartStudioAuto(p.ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(8 * time.Second)
	var got StudioProject
	for time.Now().Before(deadline) {
		got, err = e.GetStudioProject(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.AutoPhase == "failed" || got.AutoPhase == "done" {
			break
		}
		time.Sleep(40 * time.Millisecond)
	}
	if got.AutoPhase != "failed" || !strings.Contains(got.AutoError, "画面") {
		t.Fatalf("phase %s err %s script %q", got.AutoPhase, got.AutoError, got.Script)
	}
	if got.ScriptStatus != ScriptApproved || got.NarrationHash == "" {
		t.Fatalf("script %s voice %s", got.ScriptStatus, got.NarrationHash)
	}
}

func TestAttachStudioBGMFromFolder(t *testing.T) {
	e := testEngine(t)
	p, err := e.CreateStudioProject(StudioProject{Title: "配乐"})
	if err != nil {
		t.Fatal(err)
	}
	dir := e.bgmDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bed.mp3"), bytes.Repeat([]byte{1}, 64), 0o600); err != nil {
		t.Fatal(err)
	}
	list := e.ListStudioBGM()
	if len(list.Items) != 1 || list.Items[0].Name != "bed.mp3" {
		t.Fatalf("%+v", list)
	}
	got, err := e.AttachStudioBGM(p.ID, "bed.mp3")
	if err != nil || got.BGMHash == "" {
		t.Fatalf("%v %+v", err, got.BGMHash)
	}
	cleared, err := e.ClearStudioBGM(p.ID)
	if err != nil || cleared.BGMHash != "" {
		t.Fatalf("%v %s", err, cleared.BGMHash)
	}
}
