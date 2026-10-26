// 本文件锁住未签名的包不能 Admit，签过名的包会把 SKILL.md 写到目标目录。
// 不替换 wasm 验签。由 go test 调用。

package expert

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

// TestUnsignedCannotAdmit 要求公钥为 nil 时 Admit 返回错误。
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

// TestSignedAdmit 用 Distill 的正文签名后 Admit，要求名字是 demo，并且目标目录里有 SKILL.md。
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
