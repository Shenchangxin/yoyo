package app

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Shenchangxin/yoyo/internal/isolation"
	"github.com/Shenchangxin/yoyo/internal/update"
)

func isolated() bool {
	return isolation.Status().Sandbox
}

func isolationKind() string {
	return isolation.Status().Kind
}

func (a *App) loadKeymapFile() {
	if len(a.Config.Keymap) > 0 {
		return
	}
	b, err := os.ReadFile(filepath.Join(a.Home.Root, "keymap.yaml"))
	if err != nil {
		return
	}
	var m map[string]string
	if yaml.Unmarshal(b, &m) == nil && len(m) > 0 {
		a.Config.Keymap = m
	}
}

func (a *App) writeKeymapFile() {
	km := a.Config.Keymap
	if km == nil {
		km = map[string]string{}
	}
	b, err := yaml.Marshal(km)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(a.Home.Root, "keymap.yaml"), b, 0o644)
}

// CheckUpdate reports staging state. If update_url and YOYO_UPDATE_PUBKEY are
// both set, it verifies the manifest and may download a newer payload into
// staging. Applying still requires a human.
func (a *App) CheckUpdate() map[string]any {
	p := a.StagingPath()
	st, err := os.Stat(p)
	size := int64(0)
	staged := err == nil
	if staged {
		size = st.Size()
	}
	out := map[string]any{
		"current":           a.Health(),
		"staging":           p,
		"staged":            staged,
		"size":              size,
		"url":               a.Config.UpdateURL,
		"channel":           a.Config.UpdateChannel,
		"pubkey_configured": len(update.PublicKeyFromEnv()) > 0,
	}
	if a.Config.UpdateURL == "" {
		return out
	}
	pub := update.PublicKeyFromEnv()
	if len(pub) == 0 {
		out["error"] = "YOYO_UPDATE_PUBKEY missing"
		return out
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	res, err := update.Check(ctx, a.Config.UpdateURL, nil, nil, pub)
	if err != nil {
		out["error"] = err.Error()
		return out
	}
	out["verified"] = res.Verified
	out["newer"] = res.Newer
	out["manifest"] = res.Manifest
	if !res.Verified || !res.Newer || res.Manifest.URL == "" {
		return out
	}
	bin, err := update.Fetch(ctx, res.Manifest.URL, 80<<20)
	if err != nil {
		out["error"] = err.Error()
		return out
	}
	path, err := update.Stage(filepath.Dir(p), bin, res.Manifest.SHA256)
	if err != nil {
		out["error"] = err.Error()
		return out
	}
	out["staged"] = true
	out["staging"] = path
	out["size"] = len(bin)
	out["downloaded"] = true
	return out
}
