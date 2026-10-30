// 锁住按描述建片、缺下载时留住地址、分叉不动原片，以及交接用画出来的画面。

package video

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/video/film"
)

// TestDescribeBuildAndSwapKeepsTheBoard 锁住排行榜能在本机构建，换宿主后代码画面的产物编号不变。有 ffmpeg 时还会有成片。
func TestDescribeBuildAndSwapKeepsTheBoard(t *testing.T) {
	e := testEngine(t)
	p, err := e.DescribeFilm("ranking", "本机工具排行", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Structure != "ranking" || len(p.Needs) == 0 {
		t.Fatalf("%+v", p)
	}
	built, err := e.BuildFilm(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	var code string
	for _, n := range built.Needs {
		if n.Kind == "code" && n.AssetHash != "" {
			code = n.AssetHash
		}
	}
	if code == "" {
		t.Fatalf("needs %+v", built.Needs)
	}
	if e.FFmpegOK() {
		found := false
		for _, a := range built.Artifacts {
			if a.Name == "film.mp4" && a.Hash != "" {
				found = true
			}
		}
		if !found {
			t.Fatalf("artifacts %+v", built.Artifacts)
		}
	}
	if len(built.Tasks) == 0 || len(built.Artifacts) == 0 {
		t.Fatalf("tasks %d artifacts %d", len(built.Tasks), len(built.Artifacts))
	}
	swapped, err := e.SwapFilm(p.ID, film.Brief{Host: "画布"})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range swapped.Needs {
		if n.Kind == "code" && n.AssetHash != code {
			t.Fatalf("code asset changed %s -> %s", code, n.AssetHash)
		}
	}
	if !strings.Contains(swapped.Brief.Host, "画布") {
		t.Fatal(swapped.Brief.Host)
	}
}

// TestShapeFilmKeepsTheRanking 锁住改名次以后结构还是排行榜，题目和第一名写进工程。
func TestShapeFilmKeepsTheRanking(t *testing.T) {
	e := testEngine(t)
	p, err := e.DescribeFilm("ranking", "", "")
	if err != nil {
		t.Fatal(err)
	}
	shaped, err := e.ShapeFilm(p.ID, film.Cast{Subject: "三样", Rows: []string{"手机", "耳机", "杯子"}})
	if err != nil {
		t.Fatal(err)
	}
	if shaped.ID != p.ID || shaped.Structure != "ranking" || shaped.Title != "三样" || shaped.Entries[0].Label != "手机" {
		t.Fatalf("%+v", shaped)
	}
}

// TestOpenFilmKeepsTheURLWhenDownloadIsMissing 锁住没有 yt-dlp 时参考地址仍留在工程里，并写进备注。
func TestOpenFilmKeepsTheURLWhenDownloadIsMissing(t *testing.T) {
	e := testEngine(t)
	e.YTDLP = ""
	p, err := e.OpenFilm(film.Input{URL: "https://example.com/a", Title: "参考"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Reference.URL != "https://example.com/a" || !strings.Contains(p.Note, "yt-dlp") {
		t.Fatalf("%+v", p)
	}
	got, err := e.GetFilm(p.ID)
	if err != nil || got.Title != "参考" || len(got.Needs) == 0 {
		t.Fatalf("%v %+v", err, got)
	}
}

// TestForkLeavesTheOriginalAndLanguageWaitsForChat 锁住分叉不改原片。没有对话连接时换语言只记备注，原文不改。
func TestForkLeavesTheOriginalAndLanguageWaitsForChat(t *testing.T) {
	e := testEngine(t)
	p, err := e.DescribeFilm("ranking", "", "")
	if err != nil {
		t.Fatal(err)
	}
	child, err := e.ForkFilm(p.ID, film.Brief{Host: "画布"})
	if err != nil {
		t.Fatal(err)
	}
	parent, err := e.GetFilm(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if parent.Brief.Host == "画布" || child.ParentID != p.ID || child.Brief.Host != "画布" {
		t.Fatalf("parent %+v child %+v", parent.Brief, child.Brief)
	}
	noted, err := e.SwapFilm(p.ID, film.Brief{Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if noted.Brief.Language != "en" || !strings.Contains(noted.Note, "对话") {
		t.Fatalf("%+v", noted)
	}
	if !strings.Contains(film.LineTexts(noted)[0].Text, "剪辑台") {
		t.Fatal(film.LineTexts(noted))
	}
}

// TestFilmHandoffUsesThePaintedPicture 锁住构建之后交接给剪辑台的是已有画面，而且带有标题。
func TestFilmHandoffUsesThePaintedPicture(t *testing.T) {
	e := testEngine(t)
	p, err := e.DescribeFilm("podcast", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.BuildFilm(p.ID); err != nil {
		t.Fatal(err)
	}
	cut, err := e.HandoffFilm(p.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if cut.SourceHash == "" || cut.Title == "" {
		t.Fatalf("%+v", cut)
	}
}
