package film

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestContractHasNoPendingRows(t *testing.T) {
	if rows := Pending(); len(rows) != 0 {
		t.Fatalf("pending %+v", rows)
	}
	impl, excl := 0, 0
	seen := map[string]bool{}
	for _, row := range Contract() {
		if seen[row.ID] {
			t.Fatalf("duplicate %s", row.ID)
		}
		seen[row.ID] = true
		switch row.State {
		case "implemented":
			impl++
		case "excluded":
			excl++
		default:
			t.Fatalf("state %s", row.State)
		}
	}
	if impl != 8 || excl != 7 {
		t.Fatalf("implemented %d excluded %d", impl, excl)
	}
}

func TestReferenceRoundTripKeepsWordsShotsAndSticker(t *testing.T) {
	p := FromReference(Input{
		ID: "f1", Title: "街拍", Structure: "interview", Duration: 6,
		Hash: "ref", Scenes: []float64{2, 4},
		Cues: []Cue{
			{Start: 0, End: 2, Text: "先看完", Speaker: "路人"},
			{Start: 2, End: 4, Text: "再换一句", Speaker: "路人"},
			{Start: 4, End: 6, Text: "最后出片", Speaker: "路人"},
		},
	})
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var back Project
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.Structure != "interview" || len(back.Words) < 6 || len(back.Shots) != 3 {
		t.Fatalf("project %+v", back)
	}
	if back.Shots[0].Visual != "reveal" || back.Shots[0].Region == nil || back.Shots[0].Region.Label != "画面" {
		t.Fatalf("shot %+v", back.Shots[0])
	}
	if len(back.Stickers) != 1 || !back.Shots[0].Beat {
		t.Fatalf("sticker or beat missing")
	}
	if back.Reference.Hash != "ref" {
		t.Fatal(back.Reference.Hash)
	}
}

func TestStampsAndLineRewriteKeepTheOtherAsset(t *testing.T) {
	words, err := WordsFromStamps([]byte(`{"words":[
		{"text":"剪","start":0,"end":1,"speaker":"旁白","line":"a"},
		{"text":"完","start":1,"end":2,"speaker":"旁白","line":"a"},
		{"text":"再","start":2,"end":3,"speaker":"旁白","line":"b"},
		{"text":"换","start":3,"end":4,"speaker":"旁白","line":"b"}
	]}`))
	if err != nil || len(words) != 4 || words[0].LineID != "a" {
		t.Fatalf("%v %+v", err, words)
	}
	p := ranking("")
	p = Build(p, BuildEnv{})
	var kept string
	for _, n := range p.Needs {
		if n.Key == "code:shot-0" && n.AssetHash != "" {
			kept = n.AssetHash
		}
	}
	if kept == "" {
		t.Fatal("code asset missing")
	}
	p = ReplaceLine(p, "b", "留下成片")
	for _, n := range p.Needs {
		if n.Key == "code:shot-0" && n.AssetHash != kept {
			t.Fatalf("unchanged shot hash %s -> %s", kept, n.AssetHash)
		}
		if n.Key == "code:shot-1" {
			t.Fatal("shot 1 is broll")
		}
		if n.Key == "image:shot-1" && n.AssetHash != "" {
			t.Fatal("rewritten line should drop the broll asset")
		}
	}
	joined := spoken(p)
	if !strings.Contains(joined, "留") || strings.Contains(joined, "负责") {
		t.Fatalf("spoken %s", joined)
	}
}

func TestSwapHostKeepsCodeAndRebuildsPicture(t *testing.T) {
	p := Build(ranking(""), BuildEnv{})
	var codeHash string
	for _, n := range p.Needs {
		if n.Key == "code:shot-0" {
			codeHash = n.AssetHash
		}
	}
	next := Swap(p, Brief{Host: "画布"})
	for _, n := range next.Needs {
		if n.Key == "code:shot-0" && n.AssetHash != codeHash {
			t.Fatalf("code changed %s %s", codeHash, n.AssetHash)
		}
		if n.Key == "image:shot-1" && (n.Status == "skipped" || !strings.Contains(n.Prompt, "画布")) {
			t.Fatalf("image need %+v", n)
		}
	}
	if next.Brief.Host != "画布" || next.Brief.Product != "创作" {
		t.Fatalf("brief %+v", next.Brief)
	}
}

func TestTemplatesCoverTheThreeStructures(t *testing.T) {
	for _, structure := range []string{"ranking", "podcast", "interview"} {
		p := FromTemplate(structure, "")
		if p.Structure != structure || len(p.Words) == 0 || len(p.Shots) == 0 || len(p.Stickers) == 0 {
			t.Fatalf("%s %+v", structure, p)
		}
		for _, shot := range p.Shots {
			if !shot.Beat || shot.Region == nil {
				t.Fatalf("shot %+v", shot)
			}
		}
	}
	pod := FromTemplate("podcast", "")
	if pod.Shots[0].Visual != "split" || pod.Words[0].Speaker == "" {
		t.Fatalf("podcast %+v", pod.Shots[0])
	}
	iv := FromTemplate("interview", "")
	found := false
	for _, n := range iv.Needs {
		if n.Kind == "matting" {
			found = true
		}
	}
	if !found || iv.Shots[0].Region.Label != "脸" {
		t.Fatal("interview reveal")
	}
}

func TestRouteRejectsTheForeignGateway(t *testing.T) {
	list := []Endpoint{
		{ID: "hub", Name: "HypiHub", Service: "image", BaseURL: "https://api.hypihub.example", Active: true, HasKey: true},
		{ID: "img", Name: "图像", Service: "image", Model: "gpt-image", Active: true, HasKey: true},
		{ID: "matte", Name: "抠像", Service: "image", Model: "matting", Active: true, HasKey: true},
	}
	got, err := Route("image", list)
	if err != nil || got.ID != "img" {
		t.Fatalf("%v %+v", err, got)
	}
	got, err = Route("matting", list)
	if err != nil || got.ID != "matte" {
		t.Fatalf("matte %v %+v", err, got)
	}
	if _, err := Route("video", list); err == nil {
		t.Fatal("expected missing video connection")
	}
}

func TestBuildPaintsWithoutAModelAccount(t *testing.T) {
	p := Build(FromTemplate("ranking", "本机工具排行"), BuildEnv{})
	var pngHash string
	for _, a := range p.Artifacts {
		if a.MediaType == "image/png" && a.Hash != "" {
			pngHash = a.Hash
		}
		if a.Name == "captions.srt" && !strings.Contains(a.MediaType, "subrip") {
			t.Fatal(a)
		}
	}
	if pngHash == "" {
		t.Fatalf("artifacts %+v", p.Artifacts)
	}
	raw, err := RenderCode(p, p.Shots[0])
	if err != nil || !bytes.HasPrefix(raw, []byte("\x89PNG")) {
		t.Fatalf("png %v %d", err, len(raw))
	}
	html := RenderHTML(p, p.Shots[0])
	if !strings.Contains(html, "board") || !strings.Contains(html, "本机工具排行") {
		t.Fatal(html)
	}
	waiting := 0
	for _, task := range p.Tasks {
		if task.Status == "waiting" || task.Status == "failed" {
			waiting++
		}
	}
	if waiting == 0 {
		t.Fatal("image and speech should wait without connections")
	}
	again := Build(p, BuildEnv{})
	for _, n := range again.Needs {
		if n.Kind == "code" && n.Status != "skipped" {
			t.Fatalf("code rerun %+v", n)
		}
	}
}

func TestBuildNamesTheVendorAndWaitsForMatting(t *testing.T) {
	p := FromTemplate("interview", "")
	p = Build(p, BuildEnv{
		Providers: []Endpoint{{
			ID: "c1", Vendor: "openai", Service: "image", Model: "matting",
			VaultKey: "vk", Active: true, HasKey: true,
		}},
		Dispatch: func(n Need) (string, error) {
			if n.Kind == "matting" {
				if n.Provider != "openai" || n.VaultKey != "vk" {
					return "", fmt.Errorf("routed %s %s", n.Provider, n.VaultKey)
				}
				return "", Wait("抠像连接已选中，这条供应商还没有抠像请求")
			}
			return "asset", nil
		},
	})
	for _, n := range p.Needs {
		if n.Kind == "matting" && (n.Status != "waiting" || n.Provider != "openai") {
			t.Fatalf("%+v", n)
		}
	}
}

func TestReferenceKeepsStampedWordsAndRegions(t *testing.T) {
	p := FromReference(Input{
		Title: "标注", Duration: 2,
		Words:   []Word{{ID: "w0", LineID: "a", Text: "词", Start: 0, End: 2}},
		Regions: []Box{{X: 0.2, Y: 0.2, W: 0.4, H: 0.4, Label: "脸"}},
	})
	if len(p.Words) != 1 || p.Words[0].Text != "词" || p.Shots[0].Region.Label != "脸" {
		t.Fatalf("%+v", p)
	}
}

func TestHookAndPriceLeaveTheUntouchedPicture(t *testing.T) {
	p := Build(ranking(""), BuildEnv{})
	for i := range p.Needs {
		if p.Needs[i].Key == "image:shot-1" {
			p.Needs[i].AssetHash = "body"
			p.Needs[i].Status = "done"
		}
	}
	var code string
	for _, n := range p.Needs {
		if n.Key == "code:shot-0" {
			code = n.AssetHash
		}
	}
	hooked := Swap(p, Brief{Hook: "换成新开头"})
	if !strings.Contains(spoken(hooked), "换成新开头") {
		t.Fatal(spoken(hooked))
	}
	for _, n := range hooked.Needs {
		if n.Key == "code:shot-0" && n.AssetHash == code {
			t.Fatal("opening picture should rerun")
		}
		if n.Key == "image:shot-1" && (n.AssetHash != "body" || n.Status != "skipped") {
			t.Fatalf("body %+v", n)
		}
	}
	priced := Swap(p, Brief{Price: "0", CTA: "打开创作"})
	for _, n := range priced.Needs {
		if n.Key == "code:shot-0" && n.AssetHash != code {
			t.Fatal("price changed the opening board")
		}
		if n.Key == "image:shot-1" && n.Status == "skipped" {
			t.Fatal("price should rebuild the last picture")
		}
	}
}

func TestForkKeepsTheParentAndLocalizeReflowsOneLine(t *testing.T) {
	p := Build(ranking(""), BuildEnv{})
	p.ID = "parent"
	child := Fork(p, Brief{Host: "画布"})
	if child.ParentID != "parent" || child.ID != "" {
		t.Fatalf("%+v", child)
	}
	if p.Brief.Host != "剪辑台" || child.Brief.Host != "画布" {
		t.Fatalf("parent %s child %s", p.Brief.Host, child.Brief.Host)
	}
	next := Localize(p, map[string]string{"b": "keep the cut"})
	if strings.Contains(spoken(next), "负责") || !strings.Contains(spoken(next), "keep the cut") {
		t.Fatal(spoken(next))
	}
	if !strings.Contains(spoken(next), "剪辑台") {
		t.Fatal(spoken(next))
	}
}

func TestFramesShowTheBoardTheRevealAndTheSticker(t *testing.T) {
	board := RenderHTML(ranking(""), ranking("").Shots[0])
	if !strings.Contains(board, "board") || !strings.Contains(board, "剪辑台") || !strings.Contains(board, "这条排第一") {
		t.Fatal(board)
	}
	if strings.Contains(board, "成片台") {
		t.Fatal("later rows should stay hidden on the first shot")
	}
	rank := ranking("")
	later := RenderHTMLMoment(rank, rank.Shots[1], rank.Words[3])
	if !strings.Contains(later, "成片台") || !strings.Contains(later, "<mark>创作</mark>") {
		t.Fatal(later)
	}
	iv := interview("")
	first := RenderHTML(iv, iv.Shots[0])
	if !strings.Contains(first, "先看完") || strings.Contains(first, "最后出片") {
		t.Fatal(first)
	}
	last := RenderHTML(iv, iv.Shots[2])
	if !strings.Contains(last, "最后出片") || !strings.Contains(last, "先看完") {
		t.Fatal(last)
	}
	pod := podcast("")
	split := RenderHTML(pod, pod.Shots[0])
	if !strings.Contains(split, "split") || !strings.Contains(split, "主播") || !strings.Contains(split, "嘉宾") {
		t.Fatal(split)
	}
	hand := RenderHTML(pod, pod.Shots[1])
	if !strings.Contains(hand, "handoff") || !strings.Contains(hand, "创作台") {
		t.Fatal(hand)
	}
	raw, err := RenderMoment(rank, rank.Shots[0], rank.Words[0])
	if err != nil || !bytes.HasPrefix(raw, []byte("\x89PNG")) {
		t.Fatal(err)
	}
}

func TestBeatLandsOnAWord(t *testing.T) {
	p := FromReference(Input{Duration: 4, Structure: "free"})
	if len(p.Shots) != 1 || p.Shots[0].Beat {
		t.Fatalf("%+v", p.Shots)
	}
	said := FromReference(Input{
		Duration: 4, Structure: "interview",
		Cues: []Cue{{Start: 0, End: 4, Text: "一二三四", Speaker: "路人"}},
	})
	if len(said.Shots) == 0 || !said.Shots[0].Beat {
		t.Fatalf("%+v", said.Shots)
	}
}

func TestRankingRowsRequestTheirOwnPictures(t *testing.T) {
	p := Plan(ranking(""))
	seen := map[string]bool{}
	for _, n := range p.Needs {
		if strings.HasPrefix(n.Key, "image:entry:") {
			seen[n.Prompt] = true
		}
	}
	if !seen["剪辑台"] || !seen["创作"] || !seen["成片台"] {
		t.Fatalf("%v", seen)
	}
	for i := range p.Needs {
		if p.Needs[i].Key == "image:entry:e0" {
			p.Needs[i].AssetHash = "row"
			p.Needs[i].Status = "done"
		}
	}
	next := Swap(p, Brief{Host: "画布"})
	for _, n := range next.Needs {
		if n.Key == "image:entry:e0" && (n.AssetHash != "row" || n.Status != "skipped") {
			t.Fatalf("%+v", n)
		}
	}
	plain, err := RenderMoment(ranking(""), ranking("").Shots[0], Word{})
	if err != nil {
		t.Fatal(err)
	}
	tile := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			tile.SetRGBA(x, y, color.RGBA{220, 20, 60, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, tile); err != nil {
		t.Fatal(err)
	}
	rank := ranking("")
	painted, err := RenderMomentImages(rank, rank.Shots[0], rank.Words[0], map[string][]byte{"e0": buf.Bytes()})
	if err != nil || bytes.Equal(plain, painted) {
		t.Fatalf("picture well did not change the frame %v", err)
	}
}

func TestMusicIsPlannedUntilABedIsStored(t *testing.T) {
	p := Plan(ranking(""))
	found := false
	for i := range p.Needs {
		if p.Needs[i].Kind == "music" {
			found = true
			p.Needs[i].AssetHash = "song"
			p.Needs[i].Status = "done"
		}
	}
	if !found {
		t.Fatal("music need")
	}
	kept := Swap(p, Brief{Host: "画布"})
	for _, n := range kept.Needs {
		if n.Kind == "music" && (n.AssetHash != "song" || n.Status != "skipped") {
			t.Fatalf("host %+v", n)
		}
	}
	changed := Swap(p, Brief{Product: "另一件"})
	for _, n := range changed.Needs {
		if n.Kind == "music" && n.Status == "skipped" {
			t.Fatal("product should rebuild the bed")
		}
	}
	p.Bed = "bed"
	again := Plan(p)
	for _, n := range again.Needs {
		if n.Kind == "music" {
			t.Fatal("stored bed")
		}
	}
}

func TestWhisperXSegmentsAndMatte(t *testing.T) {
	words, err := WordsFromStamps([]byte(`{"segments":[{"speaker":"旁白","words":[{"word":"剪","start":0,"end":0.4},{"word":"完","start":0.4,"end":0.8}]}]}`))
	if err != nil || len(words) != 2 || words[0].Text != "剪" || words[1].Speaker != "旁白" {
		t.Fatalf("%v %+v", err, words)
	}
	cmd := strings.Join(StampCommand("whisperx", "a.wav", "out"), " ")
	if !strings.Contains(cmd, "--output_format json") || !strings.Contains(cmd, "--output_dir out") {
		t.Fatal(cmd)
	}
	base, err := RenderMoment(ranking(""), ranking("").Shots[0], Word{})
	if err != nil {
		t.Fatal(err)
	}
	tile := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			tile.SetRGBA(x, y, color.RGBA{R: 20, G: 180, B: 40, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, tile); err != nil {
		t.Fatal(err)
	}
	rank := ranking("")
	box := rank.Shots[0].Region
	mixed, err := CompositeMatte(base, buf.Bytes(), box)
	if err != nil || bytes.Equal(base, mixed) {
		t.Fatalf("matte %v", err)
	}
	exact, err := Route("matting", []Endpoint{
		{ID: "loose", Name: "抠像", Service: "image", Model: "matting", Active: true, HasKey: true},
		{ID: "cut", Service: "matting", Vendor: "removebg", Active: true, HasKey: true},
	})
	if err != nil || exact.ID != "cut" {
		t.Fatalf("%v %+v", err, exact)
	}
}

func TestParseCrop(t *testing.T) {
	box, ok := ParseCrop("crop=320:240:10:20", 640, 480)
	if !ok || box.W != 0.5 || box.H != 0.5 || box.X != 10.0/640 || box.Y != 20.0/480 {
		t.Fatalf("%v %+v", ok, box)
	}
	if _, ok := ParseCrop("crop=0:1:0:0", 10, 10); ok {
		t.Fatal("empty crop")
	}
}

func TestRegionsAndDownloadCommand(t *testing.T) {
	boxes, err := ParseRegions([]byte(`{"regions":[{"x":0.1,"y":0.2,"w":0.3,"h":0.4,"label":"脸"}]}`))
	if err != nil || len(boxes) != 1 || boxes[0].Label != "脸" {
		t.Fatal(err)
	}
	p := ApplyRegions(FromTemplate("interview", ""), boxes)
	if p.Shots[0].Region.Label != "脸" || p.Shots[0].Region.W != 0.3 {
		t.Fatalf("%+v", p.Shots[0].Region)
	}
	cmd := DownloadCommand("yt-dlp", "https://example.com/a", "out.mp4")
	if strings.Join(cmd, " ") != "yt-dlp -o out.mp4 --no-playlist https://example.com/a" {
		t.Fatal(cmd)
	}
	if len(BrowserCommand("chrome", "a.html", "a.png")) < 3 {
		t.Fatal("browser")
	}
}
