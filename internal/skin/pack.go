package skin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

type TokenMap map[string]string

type Materials struct {
	Grain        float64 `json:"grain"`
	GlassBlur    string  `json:"glassBlur,omitempty"`
	WallpaperFit string  `json:"wallpaperFit,omitempty"`
	WallpaperDim float64 `json:"wallpaperDim"`
	RadiusScale  float64 `json:"radiusScale,omitempty"`
}

type Manifest struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Author           string    `json:"author,omitempty"`
	Description      string    `json:"description,omitempty"`
	Version          string    `json:"version,omitempty"`
	MinApp           string    `json:"minApp,omitempty"`
	Capabilities     []string  `json:"capabilities"`
	PreserveEvidence *bool     `json:"preserveEvidence,omitempty"`
	Materials        Materials `json:"materials"`
	Wallpaper        string    `json:"wallpaper,omitempty"`
	FontSans         string    `json:"fontSans,omitempty"`
	FontMono         string    `json:"fontMono,omitempty"`
	Preview          string    `json:"preview,omitempty"`
}

type Pack struct {
	Manifest Manifest
	Dark     TokenMap
	Light    TokenMap
	Files    map[string][]byte
}

type Info struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Author           string    `json:"author,omitempty"`
	Description      string    `json:"description,omitempty"`
	Builtin          bool      `json:"builtin"`
	PreserveEvidence bool      `json:"preserveEvidence"`
	Capabilities     []string  `json:"capabilities,omitempty"`
	HasWallpaper     bool      `json:"hasWallpaper"`
	HasFonts         bool      `json:"hasFonts"`
	Dark             string    `json:"dark,omitempty"`
	Light            string    `json:"light,omitempty"`
	PreviewDark      TokenMap  `json:"previewDark,omitempty"`
	PreviewLight     TokenMap  `json:"previewLight,omitempty"`
	Materials        Materials `json:"materials"`
	WallpaperURL     string    `json:"wallpaperUrl,omitempty"`
	PreviewURL       string    `json:"previewUrl,omitempty"`
}

type FontFace struct {
	Family string `json:"family"`
	URL    string `json:"url"`
	Role   string `json:"role"`
}

type Resolved struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Builtin          bool           `json:"builtin"`
	Mode             string         `json:"mode"`
	PreserveEvidence bool           `json:"preserveEvidence"`
	Tokens           TokenMap       `json:"tokens"`
	WindowRGB        []int          `json:"windowRgb"`
	WallpaperURL     string         `json:"wallpaperUrl,omitempty"`
	WallpaperFile    string         `json:"wallpaperFile,omitempty"`
	Fonts            []FontFace     `json:"fonts,omitempty"`
	Materials        Materials      `json:"materials"`
	Contrast         ContrastReport `json:"contrast"`
}

const AssetPrefix = "/yoyo-skin/"

func NewUserID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "user." + hex.EncodeToString(b)
}

func (m Manifest) EvidenceLocked() bool {
	if m.PreserveEvidence == nil {
		return true
	}
	return *m.PreserveEvidence
}

func (m *Manifest) Normalize() error {
	if m == nil {
		return fmt.Errorf("missing manifest")
	}
	m.ID = strings.TrimSpace(m.ID)
	m.Name = strings.TrimSpace(m.Name)
	m.Author = strings.TrimSpace(m.Author)
	m.Description = strings.TrimSpace(m.Description)
	m.Version = strings.TrimSpace(m.Version)
	if m.Name == "" {
		return fmt.Errorf("skin name is required")
	}
	if len(m.Name) > 64 {
		return fmt.Errorf("skin name too long")
	}
	if len(m.Description) > 280 {
		m.Description = m.Description[:280]
	}
	caps := make([]string, 0, len(m.Capabilities))
	seen := map[string]bool{}
	for _, c := range m.Capabilities {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		if !KnownCap(c) {
			return fmt.Errorf("unknown capability %q", c)
		}
		seen[c] = true
		caps = append(caps, c)
	}
	if len(caps) == 0 {
		caps = []string{CapTokens}
	}
	m.Capabilities = caps
	m.Materials = m.Materials.Normalize()
	m.Wallpaper = sanitizeRel(m.Wallpaper)
	m.FontSans = sanitizeRel(m.FontSans)
	m.FontMono = sanitizeRel(m.FontMono)
	m.Preview = sanitizeRel(m.Preview)
	return nil
}

func (m Materials) Normalize() Materials {
	if m.Grain < 0 {
		m.Grain = 0
	}
	if m.Grain > 1 {
		m.Grain = 1
	}
	if m.WallpaperDim < 0 {
		m.WallpaperDim = 0
	}
	if m.WallpaperDim > 1 {
		m.WallpaperDim = 1
	}
	switch m.WallpaperFit {
	case "cover", "contain", "tile":
	default:
		m.WallpaperFit = "cover"
	}
	if m.RadiusScale <= 0 {
		m.RadiusScale = 1
	}
	if m.RadiusScale < 0.5 {
		m.RadiusScale = 0.5
	}
	if m.RadiusScale > 2 {
		m.RadiusScale = 2
	}
	if m.GlassBlur != "" {
		if err := ValidateValue(KindDimension, m.GlassBlur); err != nil {
			m.GlassBlur = ""
		}
	}
	return m
}

func (p Pack) Info() Info {
	ev := p.Manifest.EvidenceLocked()
	id := p.Manifest.ID
	info := Info{
		ID:               id,
		Name:             p.Manifest.Name,
		Author:           p.Manifest.Author,
		Description:      p.Manifest.Description,
		PreserveEvidence: ev,
		Capabilities:     p.Manifest.Capabilities,
		HasWallpaper:     p.Manifest.Wallpaper != "",
		HasFonts:         p.Manifest.FontSans != "" || p.Manifest.FontMono != "",
		Materials:        p.Manifest.Materials,
	}
	if info.HasWallpaper {
		info.WallpaperURL = AssetPrefix + id + "/assets/" + p.Manifest.Wallpaper
	}
	if p.Manifest.Preview != "" {
		info.PreviewURL = AssetPrefix + id + "/" + p.Manifest.Preview
	}
	darkBase, _ := Palette("ink")
	lightBase, _ := Palette("neutral")
	info.PreviewDark = previewSwatch(mergeMap(darkBase, p.Dark))
	info.PreviewLight = previewSwatch(mergeMap(lightBase, p.Light))
	return info
}

func previewSwatch(m TokenMap) TokenMap {
	out := TokenMap{}
	for _, k := range []string{"--background", "--sidebar", "--card", "--muted", "--accent", "--border", "--foreground"} {
		if v := m[k]; v != "" {
			out[k] = v
		}
	}
	return out
}

func (p Pack) HasCap(cap string) bool {
	for _, c := range p.Manifest.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}
