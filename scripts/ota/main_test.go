package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/update"
)

func TestBuildSignedChannels(t *testing.T) {
	dist := t.TempDir()
	ver := "0.9.0"
	files := []string{
		"Yoyo-" + ver + "-windows-x64-setup.exe",
		"Yoyo-" + ver + "-darwin-universal.dmg",
		"Yoyo-" + ver + "-linux-amd64.AppImage",
		"yoyo-cli-" + ver + "-linux-amd64.tar.gz",
	}
	var sums string
	for _, name := range files {
		b := []byte("payload-" + name)
		if err := os.WriteFile(filepath.Join(dist, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(b)
		sums += hex.EncodeToString(h[:]) + "  " + name + "\n"
	}
	if err := os.WriteFile(filepath.Join(dist, "SHA256SUMS.txt"), []byte(sums), 0o644); err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	t.Setenv(update.EnvPrivkey, hex.EncodeToString(priv))
	t.Setenv(update.EnvRepo, "demo/yoyo-releases")
	if err := run([]string{"-dist", dist, "-version", ver, "-out", out, "-notes", "test"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(out, "channels", "stable.json"))
	if err != nil {
		t.Fatal(err)
	}
	sig, err := os.ReadFile(filepath.Join(out, "channels", "stable.json.sig"))
	if err != nil {
		t.Fatal(err)
	}
	if err := update.Verify(pub, body, sig); err != nil {
		t.Fatal(err)
	}
	m, err := update.ParseManifest(body)
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != ver || m.Platforms["windows-amd64"].Kind != "nsis" {
		t.Fatalf("%+v", m)
	}
	if m.Platforms["linux-amd64-cli"].Kind != "tar.gz" {
		t.Fatalf("cli %v", m.Platforms)
	}
	if _, err := os.Stat(filepath.Join(out, "releases", "v"+ver, files[0])); err != nil {
		t.Fatal(err)
	}
}
