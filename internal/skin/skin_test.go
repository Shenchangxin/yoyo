package skin

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestBuiltinPalettesContrast(t *testing.T) {
	for _, id := range BuiltinIDs() {
		m, ok := Palette(id)
		if !ok {
			t.Fatalf("missing %s", id)
		}
		if err := ValidateMap(m); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		rep := ContrastOf(m)
		if !rep.Pass {
			t.Fatalf("%s contrast: %+v", id, rep)
		}
	}
}

func TestRejectsCSSInjection(t *testing.T) {
	err := ValidateValue(KindColor, "red; } body { background: url(https://evil)")
	if err == nil {
		t.Fatal("expected reject")
	}
	err = ValidateName("--not-a-token")
	if err == nil {
		t.Fatal("expected reject unknown token")
	}
}

func TestAdmitRoundTrip(t *testing.T) {
	img := tinyPNG(t)
	dark, _ := Palette("ink")
	light, _ := Palette("paper")
	man := Manifest{
		ID:           "user.roundtrip",
		Name:         "Roundtrip",
		Capabilities: []string{CapTokens, CapWallpaper},
		Wallpaper:    "wallpaper.png",
	}
	ev := true
	man.PreserveEvidence = &ev
	pack := Pack{Manifest: man, Dark: dark, Light: light, Files: map[string][]byte{
		"assets/wallpaper.png": img,
	}}
	raw, err := pack.Zip()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Admit(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Manifest.Name != "Roundtrip" {
		t.Fatalf("name %q", got.Manifest.Name)
	}
	dir := t.TempDir()
	st := NewStore(dir)
	saved, err := st.Save(got)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Compile(saved, "dark")
	if err != nil {
		t.Fatal(err)
	}
	if res.Tokens["--background"] != "#1c1c1a" {
		t.Fatalf("bg %s", res.Tokens["--background"])
	}
	if res.WallpaperURL == "" {
		t.Fatal("expected wallpaper url")
	}
	ctype, body, err := st.File(saved.Manifest.ID, "assets/wallpaper.png")
	if err != nil {
		t.Fatal(err)
	}
	if ctype != "image/png" || len(body) == 0 {
		t.Fatalf("serve %s %d", ctype, len(body))
	}
}

func TestAdmitRejectsTraversal(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("../evil.json")
	_, _ = w.Write([]byte(`{}`))
	_ = zw.Close()
	if _, err := Admit(buf.Bytes()); err == nil {
		t.Fatal("expected reject")
	}
}

func TestNormalizeSkinID(t *testing.T) {
	if NormalizeSkinID("builtin.ink") != "" {
		t.Fatal("builtin must clear")
	}
	if NormalizeSkinID("user.abc") != "user.abc" {
		t.Fatal("user id")
	}
	if NormalizeSkinID("../x") != "" {
		t.Fatal("path")
	}
}

func TestStoreJail(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)
	if err := os.MkdirAll(filepath.Join(dir, "user.x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.File("user.x", "../x"); err == nil {
		t.Fatal("escape")
	}
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCompileWallpaperOnlyKeepsBuiltinContrast(t *testing.T) {
	p := Pack{
		Manifest: Manifest{ID: "user.wall", Name: "W", Capabilities: []string{CapTokens, CapWallpaper}, Wallpaper: "wallpaper.png"},
		Dark:     TokenMap{},
		Light:    TokenMap{},
		Files:    map[string][]byte{"assets/wallpaper.png": tinyPNG(t)},
	}
	res, err := Compile(p, "dark")
	if err != nil {
		t.Fatal(err)
	}
	if res.Tokens["--background"] != "" {
		t.Fatalf("wallpaper-only must not overlay palette tokens: %v", res.Tokens)
	}
	if res.WallpaperURL == "" {
		t.Fatal("expected wallpaper url")
	}
}

func TestCompileRadiusScale(t *testing.T) {
	dark, _ := Palette("ink")
	light, _ := Palette("neutral")
	p := Pack{Manifest: Manifest{ID: "user.radius", Name: "R", Capabilities: []string{CapTokens}, Materials: Materials{RadiusScale: 2}}, Dark: dark, Light: light}
	res, err := Compile(p, "dark")
	if err != nil {
		t.Fatal(err)
	}
	if res.Tokens["--radius"] == dark["--radius"] {
		t.Fatalf("radius should scale: %s", res.Tokens["--radius"])
	}
}

func TestAdmitRejectsUnknownCap(t *testing.T) {
	dark, _ := Palette("ink")
	light, _ := Palette("neutral")
	p := Pack{Manifest: Manifest{ID: "user.badcap", Name: "X", Capabilities: []string{CapTokens, "javascript"}}, Dark: dark, Light: light}
	raw, err := p.Zip()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Admit(raw); err == nil {
		t.Fatal("expected unknown capability reject")
	}
}

func TestAdmitRejectsBadFont(t *testing.T) {
	dark, _ := Palette("ink")
	light, _ := Palette("neutral")
	p := Pack{
		Manifest: Manifest{ID: "user.badfont", Name: "F", Capabilities: []string{CapTokens, CapFonts}, FontSans: "sans.woff2"},
		Dark:     dark,
		Light:    light,
		Files:    map[string][]byte{"assets/fonts/sans.woff2": []byte("not a font")},
	}
	raw, err := p.Zip()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Admit(raw); err == nil {
		t.Fatal("expected font reject")
	}
}

func TestFileRejectsUnknownRel(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)
	id := "user.jail"
	if err := os.MkdirAll(filepath.Join(dir, id, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id, "assets", "secret.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.File(id, "assets/secret.txt"); err == nil {
		t.Fatal("unknown rel")
	}
}

func TestJSONTokens(t *testing.T) {
	raw, _ := json.Marshal(ink)
	var m TokenMap
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMap(m); err != nil {
		t.Fatal(err)
	}
}

func TestFitImageKeepsSmallPNG(t *testing.T) {
	raw := tinyPNG(t)
	got, name, err := fitImage(raw, "assets/wallpaper.png", MaxWallpaperEdge)
	if err != nil {
		t.Fatal(err)
	}
	if name != "assets/wallpaper.png" {
		t.Fatalf("name %s", name)
	}
	if !bytes.Equal(got, raw) {
		t.Fatal("small png should be kept")
	}
}

func TestPrepareDownsamplesWallpaper(t *testing.T) {
	dark, _ := Palette("ink")
	light, _ := Palette("paper")
	img := image.NewRGBA(image.Rect(0, 0, 5000, 120))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	p := Pack{
		Manifest: Manifest{
			ID:           "user.wide",
			Name:         "Wide",
			Capabilities: []string{CapTokens, CapWallpaper},
			Wallpaper:    "wallpaper.jpg",
		},
		Dark:  dark,
		Light: light,
		Files: map[string][]byte{"assets/wallpaper.jpg": buf.Bytes()},
	}
	if err := p.Prepare(); err != nil {
		t.Fatal(err)
	}
	body := p.Files["assets/"+p.Manifest.Wallpaper]
	cfg, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width > MaxWallpaperEdge || cfg.Height > MaxWallpaperEdge {
		t.Fatalf("still too big %dx%d", cfg.Width, cfg.Height)
	}
	if cfg.Width != MaxWallpaperEdge {
		t.Fatalf("long edge %d want %d", cfg.Width, MaxWallpaperEdge)
	}
}
