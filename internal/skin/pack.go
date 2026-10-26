// 皮肤包、清单和编译结果的数据结构。资源 URL 以 /yoyo-skin/ 开头。

package skin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// TokenMap 是令牌名到字符串值的映射。
type TokenMap map[string]string

// Materials 是颗粒、玻璃模糊、壁纸适配和圆角缩放。
type Materials struct {
	Grain        float64 `json:"grain"`
	GlassBlur    string  `json:"glassBlur,omitempty"`
	WallpaperFit string  `json:"wallpaperFit,omitempty"`
	WallpaperDim float64 `json:"wallpaperDim"`
	RadiusScale  float64 `json:"radiusScale,omitempty"`
}

// Manifest 是皮肤清单。PreserveEvidence 为空指针时 EvidenceLocked 为真。
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

// Pack 是清单、两套令牌和原始文件字节。
type Pack struct {
	Manifest Manifest
	Dark     TokenMap
	Light    TokenMap
	Files    map[string][]byte
}

// Info 是给界面看的摘要，含预览色和资源 URL。
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

// FontFace 是编译后的一条字体，Family、URL 和 Role。
type FontFace struct {
	Family string `json:"family"`
	URL    string `json:"url"`
	Role   string `json:"role"`
}

// Resolved 是编译结果。Tokens 在用户包上是处理后的用户层，在内置包上是完整调色板。
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

// AssetPrefix 是皮肤资源 URL 的前缀，值为 /yoyo-skin/。
const AssetPrefix = "/yoyo-skin/"

// NewUserID 返回 user. 加上 6 个随机字节的十六进制。
func NewUserID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "user." + hex.EncodeToString(b)
}

// EvidenceLocked 在 PreserveEvidence 为 nil 或指向真时返回真。
func (m Manifest) EvidenceLocked() bool {
	if m.PreserveEvidence == nil {
		return true
	}
	return *m.PreserveEvidence
}

// Normalize 去掉空白，名称必填且不超过 64 字节，说明超过 280 字节则截断。未知能力返回错误。能力为空时写成 tokens。资源路径经 sanitizeRel。
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

// Normalize 把 Grain 和 WallpaperDim 夹到 0 和 1。WallpaperFit 只保留 cover、contain、tile，否则改成 cover。RadiusScale 小于等于 0 时写成 1，再夹到 0.5 和 2。非法的 GlassBlur 清空。
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

// Info 填上摘要。有壁纸或预览时拼出 AssetPrefix URL。预览色用 ink 和 neutral 作底，再盖上用户令牌。
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

// previewSwatch 只复制背景、侧栏、卡片、弱化色、强调色、边框和前景这几个键。
func previewSwatch(m TokenMap) TokenMap {
	out := TokenMap{}
	for _, k := range []string{"--background", "--sidebar", "--card", "--muted", "--accent", "--border", "--foreground"} {
		if v := m[k]; v != "" {
			out[k] = v
		}
	}
	return out
}

// HasCap 报告清单的能力列表里有没有这个名字。
func (p Pack) HasCap(cap string) bool {
	for _, c := range p.Manifest.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}
