package skin

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"unicode/utf8"
)

const (
	MaxZipBytes       = 48 << 20
	MaxUncompressed   = 64 << 20
	MaxFiles          = 24
	MaxEntry          = 40 << 20
	MaxManifestBytes  = 32 << 10
	MaxTokenFileBytes = 64 << 10
)

var allowedFiles = map[string]struct{}{
	"manifest.json":           {},
	"tokens/dark.json":        {},
	"tokens/light.json":       {},
	"preview.png":             {},
	"preview.jpg":             {},
	"preview.jpeg":            {},
	"preview.webp":            {},
	"assets/wallpaper.png":    {},
	"assets/wallpaper.jpg":    {},
	"assets/wallpaper.jpeg":   {},
	"assets/wallpaper.webp":   {},
	"assets/fonts/sans.woff2": {},
	"assets/fonts/mono.woff2": {},
}

func Admit(data []byte) (Pack, error) {
	if len(data) == 0 || len(data) > MaxZipBytes {
		return Pack{}, fmt.Errorf("skin package must be between 1 and %d bytes", MaxZipBytes)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Pack{}, fmt.Errorf("not a zip/.yoyoskin archive")
	}
	if len(zr.File) == 0 || len(zr.File) > MaxFiles {
		return Pack{}, fmt.Errorf("skin package must contain between 1 and %d files", MaxFiles)
	}
	files := map[string][]byte{}
	var uncompressed int64
	for _, f := range zr.File {
		name, err := sanitizeZipPath(f.Name)
		if err != nil {
			return Pack{}, err
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if _, ok := allowedFiles[name]; !ok {
			return Pack{}, fmt.Errorf("unexpected file %q", name)
		}
		if f.UncompressedSize64 > MaxEntry {
			return Pack{}, fmt.Errorf("file %q is too large", name)
		}
		uncompressed += int64(f.UncompressedSize64)
		if uncompressed > MaxUncompressed {
			return Pack{}, fmt.Errorf("uncompressed package is too large")
		}
		rc, err := f.Open()
		if err != nil {
			return Pack{}, err
		}
		body, err := io.ReadAll(io.LimitReader(rc, MaxEntry+1))
		_ = rc.Close()
		if err != nil {
			return Pack{}, err
		}
		if len(body) > MaxEntry {
			return Pack{}, fmt.Errorf("file %q is too large", name)
		}
		files[name] = body
	}
	rawMan, ok := files["manifest.json"]
	if !ok {
		return Pack{}, fmt.Errorf("missing manifest.json")
	}
	if len(rawMan) > MaxManifestBytes || !utf8.Valid(rawMan) {
		return Pack{}, fmt.Errorf("invalid manifest.json")
	}
	var man Manifest
	if err := json.Unmarshal(rawMan, &man); err != nil {
		return Pack{}, fmt.Errorf("manifest.json: %w", err)
	}
	if err := man.Normalize(); err != nil {
		return Pack{}, err
	}
	if man.ID == "" || !ValidUserID(man.ID) {
		man.ID = NewUserID()
	}
	dark, err := parseTokens(files["tokens/dark.json"])
	if err != nil {
		return Pack{}, fmt.Errorf("tokens/dark.json: %w", err)
	}
	light, err := parseTokens(files["tokens/light.json"])
	if err != nil {
		return Pack{}, fmt.Errorf("tokens/light.json: %w", err)
	}
	if err := ValidateMap(dark); err != nil {
		return Pack{}, fmt.Errorf("dark tokens: %w", err)
	}
	if err := ValidateMap(light); err != nil {
		return Pack{}, fmt.Errorf("light tokens: %w", err)
	}
	pack := Pack{Manifest: man, Dark: dark, Light: light, Files: files}
	if err := pack.Prepare(); err != nil {
		return Pack{}, err
	}
	if err := pack.checkContrast(); err != nil {
		return Pack{}, err
	}
	return pack, nil
}

func parseTokens(raw []byte) (TokenMap, error) {
	if len(raw) == 0 {
		return TokenMap{}, nil
	}
	if len(raw) > MaxTokenFileBytes || !utf8.Valid(raw) {
		return nil, fmt.Errorf("invalid token file")
	}
	var m TokenMap
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = TokenMap{}
	}
	return m, nil
}

func (p *Pack) Prepare() error {
	return p.bindAssets()
}

func (p *Pack) bindAssets() error {
	man := &p.Manifest
	if man.Preview != "" {
		if _, ok := p.Files[man.Preview]; !ok {
			return fmt.Errorf("preview file missing")
		}
		fitted, out, err := fitImage(p.Files[man.Preview], man.Preview, MaxPreviewEdge)
		if err != nil {
			return err
		}
		p.replaceFile(man.Preview, out, fitted)
		man.Preview = out
	} else {
		for _, n := range []string{"preview.png", "preview.webp", "preview.jpg", "preview.jpeg"} {
			if _, ok := p.Files[n]; ok {
				fitted, out, err := fitImage(p.Files[n], n, MaxPreviewEdge)
				if err != nil {
					return err
				}
				p.replaceFile(n, out, fitted)
				man.Preview = out
				break
			}
		}
	}
	if p.HasCap(CapWallpaper) || man.Wallpaper != "" {
		name := man.Wallpaper
		if name == "" {
			for _, n := range []string{"assets/wallpaper.webp", "assets/wallpaper.png", "assets/wallpaper.jpg", "assets/wallpaper.jpeg"} {
				if _, ok := p.Files[n]; ok {
					name = n
					break
				}
			}
		} else if !strings.HasPrefix(name, "assets/") {
			name = "assets/" + name
		}
		if name == "" {
			if p.HasCap(CapWallpaper) {
				return fmt.Errorf("wallpaper capability set but no wallpaper file")
			}
		} else {
			body, ok := p.Files[name]
			if !ok {
				return fmt.Errorf("wallpaper file missing")
			}
			fitted, out, err := fitImage(body, name, MaxWallpaperEdge)
			if err != nil {
				return err
			}
			p.replaceFile(name, out, fitted)
			man.Wallpaper = strings.TrimPrefix(out, "assets/")
			if !containsCap(man.Capabilities, CapWallpaper) {
				man.Capabilities = append(man.Capabilities, CapWallpaper)
			}
		}
	}
	if p.HasCap(CapFonts) || man.FontSans != "" || man.FontMono != "" {
		if man.FontSans != "" {
			key := man.FontSans
			if !strings.HasPrefix(key, "assets/") {
				key = "assets/fonts/" + man.FontSans
			}
			body, ok := p.Files[key]
			if !ok {
				return fmt.Errorf("sans font missing")
			}
			if err := assertWOFF2(body); err != nil {
				return err
			}
			man.FontSans = strings.TrimPrefix(key, "assets/fonts/")
		} else if _, ok := p.Files["assets/fonts/sans.woff2"]; ok {
			if err := assertWOFF2(p.Files["assets/fonts/sans.woff2"]); err != nil {
				return err
			}
			man.FontSans = "sans.woff2"
		}
		if man.FontMono != "" {
			key := man.FontMono
			if !strings.HasPrefix(key, "assets/") {
				key = "assets/fonts/" + man.FontMono
			}
			body, ok := p.Files[key]
			if !ok {
				return fmt.Errorf("mono font missing")
			}
			if err := assertWOFF2(body); err != nil {
				return err
			}
			man.FontMono = strings.TrimPrefix(key, "assets/fonts/")
		} else if _, ok := p.Files["assets/fonts/mono.woff2"]; ok {
			if err := assertWOFF2(p.Files["assets/fonts/mono.woff2"]); err != nil {
				return err
			}
			man.FontMono = "mono.woff2"
		}
		if man.FontSans != "" || man.FontMono != "" {
			if !containsCap(man.Capabilities, CapFonts) {
				man.Capabilities = append(man.Capabilities, CapFonts)
			}
		} else if p.HasCap(CapFonts) {
			return fmt.Errorf("fonts capability set but no woff2 files")
		}
	}
	if !containsCap(man.Capabilities, CapTokens) {
		man.Capabilities = append([]string{CapTokens}, man.Capabilities...)
	}
	return nil
}

func (p Pack) checkContrast() error {
	for _, mode := range []struct {
		name string
		m    TokenMap
	}{{"dark", p.Dark}, {"light", p.Light}} {
		base, _ := Palette(map[bool]string{true: "ink", false: "neutral"}[mode.name == "dark"])
		merged := mergeMap(base, mode.m)
		rep := ContrastOf(merged)
		if !rep.Pass {
			return fmt.Errorf("%s: %s", mode.name, rep.Reason)
		}
	}
	return nil
}

func (p *Pack) replaceFile(oldName, newName string, body []byte) {
	if p.Files == nil {
		p.Files = map[string][]byte{}
	}
	if oldName != "" && oldName != newName {
		delete(p.Files, oldName)
	}
	p.Files[newName] = body
}

func assertWOFF2(body []byte) error {
	if len(body) < 8 {
		return fmt.Errorf("font too small")
	}
	if string(body[:4]) != "wOF2" {
		return fmt.Errorf("font must be woff2")
	}
	return nil
}

func sanitizeZipPath(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Clean(name)
	name = strings.TrimPrefix(name, "/")
	if name == "." || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, ":") {
		return "", fmt.Errorf("illegal path %q", name)
	}
	return name, nil
}

func sanitizeRel(name string) string {
	name = strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	name = path.Clean(name)
	name = strings.TrimPrefix(name, "/")
	if name == "." || name == ".." || strings.HasPrefix(name, "../") {
		return ""
	}
	if !reSafeIdent.MatchString(path.Base(name)) {
		return ""
	}
	return name
}

func containsCap(caps []string, cap string) bool {
	for _, c := range caps {
		if c == cap {
			return true
		}
	}
	return false
}
