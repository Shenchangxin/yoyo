package idol

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPresetSeed(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	view := st.View()
	if view.PresetRev != presetRevision {
		t.Fatalf("rev %d", view.PresetRev)
	}
	if view.User.Address != "制作人" {
		t.Fatalf("address %q", view.User.Address)
	}
	if len(view.Idols) != 4 {
		t.Fatalf("idols %d", len(view.Idols))
	}
	if len(view.Circles) != 1 || view.Circles[0].ID != "sc-illumination" {
		t.Fatalf("circles %+v", view.Circles)
	}
	for _, it := range view.Idols {
		if it.ID == "sc-fuyuko" {
			t.Fatal("retired idol was seeded")
		}
	}
	var yoyo, mano Idol
	for _, it := range view.Idols {
		switch it.ID {
		case petIdolID:
			yoyo = it
		case "sc-mano":
			mano = it
		}
	}
	if yoyo.Kind != "pet" || !yoyo.Pinned || yoyo.Name != "Yoyo" {
		t.Fatalf("yoyo %+v", yoyo)
	}
	if mano.Name != "樱木真乃" || mano.Kind != "catalog" || mano.CircleID == "" || mano.OutfitID != "stand/1030010010" {
		t.Fatalf("mano %+v", mano)
	}
	if !strings.Contains(mano.Sheet, "真乃") || !strings.Contains(mano.Facts, "基本信息") {
		t.Fatal("sheet or facts missing")
	}
	if len(mano.LikedGifts) == 0 || mano.LikedGifts[0] != "flower" {
		t.Fatalf("gifts %#v", mano.LikedGifts)
	}
	blob, err := st.ReadBlob(mano.Portrait)
	if err != nil || len(blob) < 32 || blob[0] != 0xff || blob[1] != 0xd8 {
		t.Fatalf("portrait %v %d", err, len(blob))
	}
	again, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.View().Idols) != len(view.Idols) {
		t.Fatalf("duplicated %d %d", len(again.View().Idols), len(view.Idols))
	}
	if err := st.Update(func(s *State) error {
		var next []Idol
		for _, it := range s.Idols {
			if it.ID != petIdolID {
				next = append(next, it)
			}
		}
		s.Idols = next
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	back, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range back.View().Idols {
		if it.ID == petIdolID {
			t.Fatal("deleted pet returned")
		}
	}
}

func TestPresetDropsRetiredIdols(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`{"preset_rev":2,"circles":[{"id":"sc-illumination","name":"illumination STARS"},{"id":"sc-straylight","name":"Straylight"},{"id":"mine-circle","name":"宿舍"}],"idols":[{"id":"yoyo","name":"Yoyo","kind":"pet","color":"#c47b8a","spoken_lang":"zh","subtitle_lang":"zh"},{"id":"sc-mano","name":"樱木真乃","kind":"catalog","color":"#ffbad6","spoken_lang":"ja","subtitle_lang":"zh"},{"id":"sc-fuyuko","name":"黛冬优子","kind":"catalog","pinned":true,"color":"#334455","spoken_lang":"ja","subtitle_lang":"zh"},{"id":"mine","name":"小灯","color":"#112233","spoken_lang":"zh","subtitle_lang":"zh"}],"conversations":[{"id":"c1","idol_id":"sc-fuyuko"}],"memories":[{"id":"m1","text":"旧","source_idol":"sc-fuyuko"}]}`)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	view := st.View()
	if view.PresetRev != presetRevision {
		t.Fatalf("rev %d", view.PresetRev)
	}
	for _, it := range view.Idols {
		if it.ID == "sc-fuyuko" {
			t.Fatal("retired idol stayed")
		}
	}
	var yoyo, mine, mano bool
	for _, it := range view.Idols {
		switch it.ID {
		case petIdolID:
			yoyo = it.Pinned
		case "mine":
			mine = true
		case "sc-mano":
			mano = true
		}
	}
	if !yoyo || !mine || !mano {
		t.Fatalf("yoyo pinned=%v mine=%v mano=%v idols=%d", yoyo, mine, mano, len(view.Idols))
	}
	for _, c := range view.Circles {
		if c.ID == "sc-straylight" {
			t.Fatal("retired circle stayed")
		}
	}
	if len(view.Conversations) != 0 || len(view.Memories) != 0 {
		t.Fatalf("leftovers conv %d mem %d", len(view.Conversations), len(view.Memories))
	}
}

func TestPresetKeepsExistingPin(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`{"idols":[{"id":"mine","name":"小灯","pinned":true,"color":"#112233","spoken_lang":"zh","subtitle_lang":"zh"}],"user":{"address":"老板"}}`)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	view := st.View()
	if view.User.Address != "老板" {
		t.Fatalf("address %q", view.User.Address)
	}
	var mine, yoyo Idol
	var saw bool
	for _, it := range view.Idols {
		if it.ID == "mine" {
			mine = it
		}
		if it.ID == petIdolID {
			yoyo = it
			saw = true
		}
	}
	if !mine.Pinned || !saw || yoyo.Pinned {
		t.Fatalf("mine pinned=%v yoyo pinned=%v saw=%v", mine.Pinned, yoyo.Pinned, saw)
	}
}
