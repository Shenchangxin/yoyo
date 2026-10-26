// 皮肤的列出、导入、激活和保存。内置调色板不写入用户皮肤库。保存时壁纸、字体和预览的 base64 为空则沿用已有文件。

package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/skin"
)

// SkinSaveIn 是保存用户皮肤的入参。壁纸、无衬线和等宽字体、预览图用 base64。PreserveEvidence 会写进清单。
type SkinSaveIn struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Author           string         `json:"author"`
	Description      string         `json:"description"`
	PreserveEvidence bool           `json:"preserveEvidence"`
	Dark             skin.TokenMap  `json:"dark"`
	Light            skin.TokenMap  `json:"light"`
	Materials        skin.Materials `json:"materials"`
	WallpaperB64     string         `json:"wallpaper_b64"`
	WallpaperName    string         `json:"wallpaper_name"`
	FontSansB64      string         `json:"font_sans_b64"`
	FontMonoB64      string         `json:"font_mono_b64"`
	PreviewB64       string         `json:"preview_b64"`
}

// skinStore 返回皮肤库。尚未创建且 Home 存在时用 Home.Skins 新建。App 为空返回 nil。
func (a *App) skinStore() *skin.Store {
	if a == nil {
		return nil
	}
	if a.Skins == nil && a.Home != nil {
		a.Skins = skin.NewStore(a.Home.Skins())
	}
	return a.Skins
}

// ListSkins 先列内置皮肤，再追加用户库里的皮肤。库不存在时只返回内置。
func (a *App) ListSkins() []skin.Info {
	out := make([]skin.Info, 0, 8)
	for _, id := range skin.BuiltinIDs() {
		out = append(out, skin.BuiltinInfo(id))
	}
	st := a.skinStore()
	if st == nil {
		return out
	}
	out = append(out, st.List()...)
	return out
}

// GetSkin 对内置调色板返回成对的 dark 和 light。用户皮肤从库里 Load。库不存在返回 no skin store。
func (a *App) GetSkin(id string) (map[string]any, error) {
	if pal, ok := skin.BuiltinPaletteID(id); ok {
		darkID, lightID := skin.PairForBuiltin(pal)
		dark, _ := skin.Palette(darkID)
		light, _ := skin.Palette(lightID)
		info := skin.BuiltinInfo(pal)
		return map[string]any{
			"info":  info,
			"dark":  dark,
			"light": light,
		}, nil
	}
	st := a.skinStore()
	if st == nil {
		return nil, fmt.Errorf("no skin store")
	}
	p, err := st.Load(id)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"info":  p.Info(),
		"dark":  p.Dark,
		"light": p.Light,
	}, nil
}

// ImportSkin 用 skin.Admit 接纳字节再 Save，返回保存后的 Info。
func (a *App) ImportSkin(raw []byte) (skin.Info, error) {
	p, err := skin.Admit(raw)
	if err != nil {
		return skin.Info{}, err
	}
	st := a.skinStore()
	if st == nil {
		return skin.Info{}, fmt.Errorf("no skin store")
	}
	saved, err := st.Save(p)
	if err != nil {
		return skin.Info{}, err
	}
	return saved.Info(), nil
}

// ImportSkinPath 读文件后调用 ImportSkin。
func (a *App) ImportSkinPath(path string) (skin.Info, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return skin.Info{}, err
	}
	return a.ImportSkin(b)
}

// ExportSkin 导出字节，文件名优先用清单名称加 .yoyoskin，否则用 id。
func (a *App) ExportSkin(id string) (string, []byte, error) {
	st := a.skinStore()
	if st == nil {
		return "", nil, fmt.Errorf("no skin store")
	}
	b, err := st.Export(id)
	if err != nil {
		return "", nil, err
	}
	name := id + ".yoyoskin"
	if p, err := st.Load(id); err == nil && p.Manifest.Name != "" {
		name = sanitizeFileName(p.Manifest.Name) + ".yoyoskin"
	}
	return name, b, nil
}

// DeleteSkin 删除用户皮肤。Config.Skin 等于该 id 时先清空并 SaveConfig。
func (a *App) DeleteSkin(id string) error {
	st := a.skinStore()
	if st == nil {
		return fmt.Errorf("no skin store")
	}
	if a.Config.Skin == id {
		a.Config.Skin = ""
		_ = a.SaveConfig()
	}
	return st.Delete(id)
}

// ActivateSkin 在 id 为空时清空 Config.Skin。内置 ink、dim、slate 写入 PaletteDark，neutral、paper、mist 写入 PaletteLight，并清空 Skin。用户皮肤把 Config.Skin 设为该 id。
func (a *App) ActivateSkin(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		a.Config.Skin = ""
		return a.SaveConfig()
	}
	if pal, ok := skin.BuiltinPaletteID(id); ok {
		a.Config.Skin = ""
		switch pal {
		case "ink", "dim", "slate":
			a.Config.PaletteDark = pal
		case "neutral", "paper", "mist":
			a.Config.PaletteLight = pal
		}
		return a.SaveConfig()
	}
	st := a.skinStore()
	if st == nil {
		return fmt.Errorf("no skin store")
	}
	if _, err := st.Load(id); err != nil {
		return err
	}
	a.Config.Skin = id
	return a.SaveConfig()
}

// ResolveSkin 只认 light 和 dark，其他 mode 当成 dark。id 为空时用配置里的调色板。用户皮肤 Load 失败时回落到 fallbackPalette，并返回该错误。
func (a *App) ResolveSkin(id, mode string) (skin.Resolved, error) {
	if mode != "light" {
		mode = "dark"
	}
	id = strings.TrimSpace(id)
	if id == "" {
		pal := a.Config.PaletteDark
		if mode == "light" {
			pal = a.Config.PaletteLight
		}
		return skin.CompileBuiltin(pal, mode), nil
	}
	if pal, ok := skin.BuiltinPaletteID(id); ok {
		return skin.CompileBuiltin(pal, mode), nil
	}
	st := a.skinStore()
	if st == nil {
		return skin.CompileBuiltin(fallbackPalette(mode, a), mode), nil
	}
	p, err := st.Load(id)
	if err != nil {
		return skin.CompileBuiltin(fallbackPalette(mode, a), mode), err
	}
	return skin.Compile(p, mode)
}

// SaveSkin 写入令牌，并在 base64 非空时附上壁纸、字体或预览。对应字段为空且已有皮肤时沿用旧文件。名称空则用 Custom。用户 id 不合法时清空，再由库分配。
func (a *App) SaveSkin(in SkinSaveIn) (skin.Info, error) {
	st := a.skinStore()
	if st == nil {
		return skin.Info{}, fmt.Errorf("no skin store")
	}
	ev := in.PreserveEvidence
	man := skin.Manifest{
		ID:               strings.TrimSpace(in.ID),
		Name:             strings.TrimSpace(in.Name),
		Author:           strings.TrimSpace(in.Author),
		Description:      strings.TrimSpace(in.Description),
		Capabilities:     []string{skin.CapTokens},
		PreserveEvidence: &ev,
		Materials:        in.Materials,
	}
	if man.Name == "" {
		man.Name = "Custom"
	}
	files := map[string][]byte{}
	var existing skin.Pack
	if man.ID != "" && skin.ValidUserID(man.ID) {
		if p, err := st.Load(man.ID); err == nil {
			existing = p
		}
	}
	if b, err := decodeB64(in.WallpaperB64); err == nil && len(b) > 0 {
		name := "wallpaper.png"
		if n := strings.ToLower(in.WallpaperName); strings.HasSuffix(n, ".webp") {
			name = "wallpaper.webp"
		} else if strings.HasSuffix(n, ".jpg") || strings.HasSuffix(n, ".jpeg") {
			name = "wallpaper.jpg"
		}
		files["assets/"+name] = b
		man.Wallpaper = name
		man.Capabilities = append(man.Capabilities, skin.CapWallpaper)
	} else if existing.Manifest.Wallpaper != "" {
		man.Wallpaper = existing.Manifest.Wallpaper
		key := "assets/" + existing.Manifest.Wallpaper
		if body := existing.Files[key]; len(body) > 0 {
			files[key] = body
		}
		if !containsStr(man.Capabilities, skin.CapWallpaper) {
			man.Capabilities = append(man.Capabilities, skin.CapWallpaper)
		}
	}
	if b, err := decodeB64(in.FontSansB64); err == nil && len(b) > 0 {
		files["assets/fonts/sans.woff2"] = b
		man.FontSans = "sans.woff2"
		man.Capabilities = append(man.Capabilities, skin.CapFonts)
	} else if existing.Manifest.FontSans != "" {
		man.FontSans = existing.Manifest.FontSans
		key := "assets/fonts/" + existing.Manifest.FontSans
		if body := existing.Files[key]; len(body) > 0 {
			files[key] = body
		}
		if !containsStr(man.Capabilities, skin.CapFonts) {
			man.Capabilities = append(man.Capabilities, skin.CapFonts)
		}
	}
	if b, err := decodeB64(in.FontMonoB64); err == nil && len(b) > 0 {
		files["assets/fonts/mono.woff2"] = b
		man.FontMono = "mono.woff2"
		if !containsStr(man.Capabilities, skin.CapFonts) {
			man.Capabilities = append(man.Capabilities, skin.CapFonts)
		}
	} else if existing.Manifest.FontMono != "" {
		man.FontMono = existing.Manifest.FontMono
		key := "assets/fonts/" + existing.Manifest.FontMono
		if body := existing.Files[key]; len(body) > 0 {
			files[key] = body
		}
		if !containsStr(man.Capabilities, skin.CapFonts) {
			man.Capabilities = append(man.Capabilities, skin.CapFonts)
		}
	}
	if b, err := decodeB64(in.PreviewB64); err == nil && len(b) > 0 {
		files["preview.png"] = b
		man.Preview = "preview.png"
	} else if existing.Manifest.Preview != "" {
		man.Preview = existing.Manifest.Preview
		if body := existing.Files[existing.Manifest.Preview]; len(body) > 0 {
			files[existing.Manifest.Preview] = body
		}
	}
	if man.ID != "" && !skin.ValidUserID(man.ID) {
		man.ID = ""
	}
	p := skin.Pack{Manifest: man, Dark: in.Dark, Light: in.Light, Files: files}
	if err := p.Manifest.Normalize(); err != nil {
		return skin.Info{}, err
	}
	if p.Manifest.ID == "" {
		p.Manifest.ID = skin.NewUserID()
	}
	if err := p.Prepare(); err != nil {
		return skin.Info{}, err
	}
	saved, err := st.Save(p)
	if err != nil {
		return skin.Info{}, err
	}
	return saved.Info(), nil
}

// fallbackPalette 在 light 模式优先用 PaletteLight，否则用 PaletteDark。都空时 light 返回 neutral，dark 返回 ink。
func fallbackPalette(mode string, a *App) string {
	if a != nil && mode == "light" && a.Config.PaletteLight != "" {
		return a.Config.PaletteLight
	}
	if a != nil && a.Config.PaletteDark != "" {
		return a.Config.PaletteDark
	}
	if mode == "light" {
		return "neutral"
	}
	return "ink"
}

// sanitizeFileName 把空格换成连字符，只保留字母、数字、连字符和下划线。结果为空时返回 skin。
func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "skin"
	}
	var b strings.Builder
	for _, r := range name {
		if r == ' ' {
			b.WriteByte('-')
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "skin"
	}
	return out
}

// containsStr 判断切片里是否有相等的字符串。
func containsStr(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
