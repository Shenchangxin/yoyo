// cut_test.go 锁住剪辑项目、会话绑定、省略和保留。依赖本机 ffmpeg 的用例在找不到二进制时跳过。

package video

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// TestReleaseCutBindReturnsHome 确认解除绑定后会话不再关联，项目仍能读到。
func TestReleaseCutBindReturnsHome(t *testing.T) {
	e := testEngine(t)
	p, err := e.CreateCutProject(CutProject{Title: "口播"})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.BindCutSession("sess-home", p.ID); err != nil {
		t.Fatal(err)
	}
	e.ReleaseCutBind("sess-home")
	if _, ok := e.CutSessionBind("sess-home"); ok {
		t.Fatal("session still bound")
	}
	if _, err := e.GetCutProject(p.ID); err != nil {
		t.Fatal(err)
	}
}

// TestCutProjectCRUD 确认项目能创建和读回，会话绑到该项目，列表里只有这一条。
func TestCutProjectCRUD(t *testing.T) {
	e := testEngine(t)
	p, err := e.CreateCutProject(CutProject{Title: "口播小样"})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == "" {
		t.Fatal("missing id")
	}
	got, err := e.GetCutProject(p.ID)
	if err != nil || got.Title != "口播小样" {
		t.Fatalf("get %+v %v", got, err)
	}
	if err := e.BindCutSession("sess-cut", p.ID); err != nil {
		t.Fatal(err)
	}
	id, ok := e.CutSessionBind("sess-cut")
	if !ok || id != p.ID {
		t.Fatalf("bind %s %v", id, ok)
	}
	got, err = e.GetCutProject(p.ID)
	if err != nil || got.SessionID != "sess-cut" {
		t.Fatalf("session %q %v", got.SessionID, err)
	}
	if e.LatestCutSession(p.ID) != "sess-cut" {
		t.Fatalf("latest %s", e.LatestCutSession(p.ID))
	}
	list := e.ListCutProjects()
	if len(list) != 1 {
		t.Fatalf("list %d", len(list))
	}
}

// TestCutKeepAndOmit 确认省略字幕会切开时间线，再省略一次会恢复，只保留一个片段时剪辑表只剩该片段。
func TestCutKeepAndOmit(t *testing.T) {
	e := testEngine(t)
	p, err := e.CreateCutProject(CutProject{Title: "口播"})
	if err != nil {
		t.Fatal(err)
	}
	p.Duration = 20
	p.Breakdown.Clips = []cut.Clip{
		{ID: "a", Start: 0, End: 8, Title: "a"},
		{ID: "b", Start: 8, End: 16, Title: "b"},
	}
	p.Captions.Cues = []cut.Cue{{Start: 1, End: 3, Text: "你好"}}
	p, err = e.saveCut(p)
	if err != nil {
		t.Fatal(err)
	}
	omitted, err := e.CutOmitCue(p.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !omitted.Captions.Cues[0].Omit {
		t.Fatal("expected omit")
	}
	if len(omitted.EditList.Cuts) < 2 || omitted.EditList.Cuts[0].End > 1.1 {
		t.Fatalf("punched %+v", omitted.EditList.Cuts)
	}
	back, err := e.CutOmitCue(p.ID, 0)
	if err != nil || back.Captions.Cues[0].Omit {
		t.Fatalf("restore %+v %v", back.Captions.Cues, err)
	}
	kept, err := e.CutKeep(p.ID, []string{"b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(kept.EditList.Cuts) != 1 || kept.EditList.Cuts[0].ID != "b" {
		t.Fatalf("keep %+v", kept.EditList.Cuts)
	}
}

// TestProviderMenuHidesUnkeyed 确认连接登记存在时菜单每一行都有 id，登记为空时跳过。
func TestProviderMenuHidesUnkeyed(t *testing.T) {
	e := testEngine(t)
	if e.Conn == nil {
		t.Skip("no connection registry")
	}
	menu := e.ProviderMenu("")
	for _, row := range menu {
		if row["id"] == nil {
			t.Fatal(row)
		}
	}
}

// TestOpenCutReleasesStudioBind 确认打开剪辑后工作室绑定解除、剪辑会话绑上，同一素材再次打开仍是原项目。
func TestOpenCutReleasesStudioBind(t *testing.T) {
	e := testEngine(t)
	hash, err := e.PutBytes([]byte("cut-source-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	studio, err := e.CreateStudioProject(StudioProject{Title: "通勤成片"})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.BindStudioSession("sess-open", studio.ID); err != nil {
		t.Fatal(err)
	}
	p, err := e.OpenCut(CutOpen{Title: "通勤成片", Hash: hash, SessionID: "sess-open"})
	if err != nil {
		t.Fatal(err)
	}
	if p.SourceHash != hash {
		t.Fatalf("source %s", p.SourceHash)
	}
	if _, ok := e.StudioSessionBind("sess-open"); ok {
		t.Fatal("studio bind should drop so conductor lane can attach")
	}
	id, ok := e.CutSessionBind("sess-open")
	if !ok || id != p.ID {
		t.Fatalf("cut bind %s %v", id, ok)
	}
	again, err := e.OpenCut(CutOpen{Hash: hash, SessionID: "sess-open"})
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != p.ID {
		t.Fatalf("expected reuse %s got %s", p.ID, again.ID)
	}
}

// TestOpenCutFromCanvasResource 确认用画布资源打开剪辑时，来源哈希等于该资源的内容哈希。
func TestOpenCutFromCanvasResource(t *testing.T) {
	e := testEngine(t)
	res, err := e.putCanvasBytes("video", "video/mp4", "clip.mp4", []byte("canvas-clip"), 0, 0, 0, "res-cut")
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.OpenCut(CutOpen{Title: "画布成片", ResourceID: "resource:res-cut"})
	if err != nil {
		t.Fatal(err)
	}
	if p.SourceHash != res.CASHash {
		t.Fatalf("hash %s want %s", p.SourceHash, res.CASHash)
	}
}

// TestVideoRouteRejectsGame 确认目的地为 game 时路由返回错误。
func TestVideoRouteRejectsGame(t *testing.T) {
	e := testEngine(t)
	if _, err := e.VideoRoute("sess-route", "game", "", "世界", "collect three relics"); err == nil {
		t.Fatal("expected game dest to fail")
	}
}

// TestVideoRouteCreatesCut 确认路由到 cut 时返回项目标识，并把会话绑到该项目。
func TestVideoRouteCreatesCut(t *testing.T) {
	e := testEngine(t)
	out, err := e.VideoRoute("sess-route", "cut", "", "胶片", "")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := out["id"].(string)
	if id == "" {
		t.Fatalf("%v", out)
	}
	got, ok := e.CutSessionBind("sess-route")
	if !ok || got != id {
		t.Fatalf("bind %s", got)
	}
}

// ffmpegBin 寻找 ffmpeg 和 ffprobe，找不到 ffmpeg 时跳过测试。
func ffmpegBin(t *testing.T) (string, string) {
	t.Helper()
	ff, probe := LookFFmpeg()
	if ff == "" {
		if p, err := exec.LookPath("ffmpeg"); err == nil {
			ff = p
		}
	}
	if probe == "" {
		if p, err := exec.LookPath("ffprobe"); err == nil {
			probe = p
		}
	}
	if ff == "" {
		t.Skip("ffmpeg missing")
	}
	return ff, probe
}

// makeTalkingClip 用 ffmpeg 合成一段样片，合成失败时跳过。
func makeTalkingClip(t *testing.T, dir, ff string) string {
	t.Helper()
	out := filepath.Join(dir, "talk.mp4")
	cmd := exec.Command(ff,
		"-y",
		"-f", "lavfi", "-i", "color=c=gray:s=320x240:d=12:r=15",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=4",
		"-f", "lavfi", "-i", "anullsrc=r=44100:cl=mono:d=4",
		"-f", "lavfi", "-i", "sine=frequency=330:duration=4",
		"-filter_complex", "[1:a][2:a][3:a]concat=n=3:v=0:a=1[a]",
		"-map", "0:v", "-map", "[a]", "-shortest",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac",
		out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("cannot make sample: %s", ffmpegErr(b, err))
	}
	return out
}

// TestSampleAIngestProposeKeepAudio 确认导入和分析后时长至少为 8，提案片段都有哈希且 energy 不小于 1，应用后有成片哈希。
func TestSampleAIngestProposeKeepAudio(t *testing.T) {
	ff, probe := ffmpegBin(t)
	e := testEngine(t)
	e.FFMPEG, e.FFProbe = ff, probe
	src := makeTalkingClip(t, t.TempDir(), ff)
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := e.PutBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.CutIngest("", hash, "口播小样A")
	if err != nil {
		t.Fatal(err)
	}
	if p.Duration < 8 {
		t.Fatalf("ingest duration %v", p.Duration)
	}
	p, err = e.CutAnalyze(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Duration < 8 {
		t.Fatalf("duration %v", p.Duration)
	}
	p, err = e.CutPropose(p.ID, 3, "8-15")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Breakdown.Clips) == 0 {
		t.Fatal("no clips")
	}
	for _, c := range p.Breakdown.Clips {
		if c.Hash == "" {
			t.Fatal("clip missing hash")
		}
		if c.Scores["energy"] < 1 {
			t.Fatalf("scores %+v", c.Scores)
		}
	}
	p, err = e.CutApply(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.ResultHash == "" {
		t.Fatal("no result")
	}
}

// TestSampleBCaptionsAndDuckMix 确认字幕至少有 2 条且有 SRT 哈希，混音后有成片哈希，闪避滤镜图不是空串。
func TestSampleBCaptionsAndDuckMix(t *testing.T) {
	ff, probe := ffmpegBin(t)
	e := testEngine(t)
	e.FFMPEG, e.FFProbe = ff, probe
	dir := t.TempDir()
	src := makeTalkingClip(t, dir, ff)
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := e.PutBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := e.CutIngest("", hash, "配音小样B")
	if err != nil {
		t.Fatal(err)
	}
	p, err = e.CutAnalyze(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	p, err = e.CutCaptions(p.ID, "This is a dubbed line. Another line follows.", "en")
	if err != nil {
		t.Fatal(err)
	}
	if p.Captions.SRTHash == "" || len(p.Captions.Cues) < 2 {
		t.Fatalf("captions %+v", p.Captions)
	}
	voice := filepath.Join(dir, "vo.wav")
	if b, err := exec.Command(ff, "-y", "-f", "lavfi", "-i", "sine=frequency=880:duration=3", voice).CombinedOutput(); err != nil {
		t.Skipf("vo: %s", ffmpegErr(b, err))
	}
	music := filepath.Join(dir, "bg.wav")
	if b, err := exec.Command(ff, "-y", "-f", "lavfi", "-i", "sine=frequency=120:duration=6", music).CombinedOutput(); err != nil {
		t.Skipf("bg: %s", ffmpegErr(b, err))
	}
	vh, err := e.PutBytes(mustRead(t, voice))
	if err != nil {
		t.Fatal(err)
	}
	mh, err := e.PutBytes(mustRead(t, music))
	if err != nil {
		t.Fatal(err)
	}
	p, err = e.CutSetStem(p.ID, "dub", vh)
	if err != nil {
		t.Fatal(err)
	}
	p, err = e.CutSetStem(p.ID, "music", mh)
	if err != nil {
		t.Fatal(err)
	}
	p, err = e.CutMix(p.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if p.ResultHash == "" {
		t.Fatal("mix produced no result")
	}
	filter, _ := cut.MixGraph(true, true, cut.MixOpts{Duck: true})
	if filter == "" {
		t.Fatal("duck graph empty")
	}
}

// mustRead 读出整个文件，失败则终止测试。
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
