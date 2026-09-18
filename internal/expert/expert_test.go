package expert

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestUnsignedCannotAdmit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: demo\ndescription: d\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Admit(dir, t.TempDir(), nil)
	if err == nil {
		t.Fatal("unsigned must fail")
	}
}

func TestSignedAdmit(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	md := Distill("demo", "d", "body")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.sig"), []byte(Sign(priv, md)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	p, err := Admit(dir, dest, pub)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "demo" {
		t.Fatal(p.Name)
	}
	if _, err := os.Stat(filepath.Join(dest, "demo", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}
