package video

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

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
	list := e.ListCutProjects()
	if len(list) != 1 {
		t.Fatalf("list %d", len(list))
	}
}

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

func TestVideoRouteCreatesCut(t *testing.T) {
	e := testEngine(t)
	out, err := e.VideoRoute("sess-route", "cut", "", "胶片")
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

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
