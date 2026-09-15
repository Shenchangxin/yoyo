package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckAndStage(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"version":"0.2.0","channel":"nightly","sha256":"abc"}`)
	sig := Sign(priv, payload)
	res, err := Check(context.Background(), "", payload, sig, pub)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Verified || !res.Newer || res.Manifest.Version != "0.2.0" {
		t.Fatalf("%+v", res)
	}
	bin := []byte("fake-binary")
	sum := sha256.Sum256(bin)
	path, err := Stage(t.TempDir(), bin, hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("empty stage path")
	}
	if _, err := Stage(t.TempDir(), bin, "deadbeef"); err == nil {
		t.Fatal("bad sha accepted")
	}
}

func TestApplyStagedRenamesRunningPath(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "yoyo.bin")
	if err := os.WriteFile(exe, []byte("old-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(dir, StagingName())
	if err := os.WriteFile(stage, []byte("new-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ApplyStaged(exe, stage); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-bytes" {
		t.Fatalf("%q", got)
	}
	old, err := os.ReadFile(exe + ".old")
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != "old-bytes" {
		t.Fatalf("backup %q", old)
	}
}
