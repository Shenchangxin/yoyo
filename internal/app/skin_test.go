// 锁住新家目录没有激活皮肤、再次保存会留下壁纸，以及宿主工具名单里没有皮肤 RPC。

package app_test

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/skin"
	"github.com/Shenchangxin/yoyo/internal/tool"
)

// TestSkinDefaultsEmpty 锁住 Open 后 Config.Skin 为空，ListSkins 至少包含 6 个内置皮肤。
func TestSkinDefaultsEmpty(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.Config.Skin != "" {
		t.Fatalf("skin %q", a.Config.Skin)
	}
	list := a.ListSkins()
	if len(list) < 6 {
		t.Fatalf("expected builtin skins, got %d", len(list))
	}
}

// TestSkinSaveKeepsWallpaper 锁住只改令牌的再次保存仍 HasWallpaper，ActivateSkin 把 Config.Skin 写成该 id。
func TestSkinSaveKeepsWallpaper(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	dark, _ := skin.Palette("ink")
	light, _ := skin.Palette("paper")
	info, err := a.SaveSkin(app.SkinSaveIn{
		Name:             "Keep",
		PreserveEvidence: true,
		Dark:             dark,
		Light:            light,
		WallpaperB64:     base64.StdEncoding.EncodeToString(tinyPNGApp(t)),
		WallpaperName:    "wall.png",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !info.HasWallpaper {
		t.Fatal("expected wallpaper")
	}
	again, err := a.SaveSkin(app.SkinSaveIn{
		ID:               info.ID,
		Name:             "Keep",
		PreserveEvidence: true,
		Dark:             dark,
		Light:            light,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !again.HasWallpaper {
		t.Fatal("wallpaper dropped on token-only save")
	}
	if err := a.ActivateSkin(again.ID); err != nil {
		t.Fatal(err)
	}
	if a.Config.Skin != again.ID {
		t.Fatalf("activate %q", a.Config.Skin)
	}
}

// TestHostToolsHaveNoSkin 锁住宿主工具名不是 skins.list、skins.import 或 skins.save。
func TestHostToolsHaveNoSkin(t *testing.T) {
	for _, s := range tool.HostSpecs() {
		if s.Name == "skins.list" || s.Name == "skins.import" || s.Name == "skins.save" {
			t.Fatalf("skin RPC must not be an agent tool: %s", s.Name)
		}
	}
}

// tinyPNGApp 做一张 8×8 的 PNG，供壁纸保存测试使用。
func tinyPNGApp(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
