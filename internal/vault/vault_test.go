// 锁住文件往返：Set 写入的 default 能被 Get 读回，同一目录再次 New 后仍然能读到。
// vault.json 权限为 0o600 或 0o666 时通过；其他权限只在路径分隔符是 / 时失败。
// 本测试不锁环境变量回落，也不锁钥匙串里的值。

package vault

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFileRoundTrip 确认密钥经文件重开后仍能读回，并检查 vault.json 的权限。
func TestFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	s.Set("default", "sk-test")
	got, err := s.Get("default")
	if err != nil || got != "sk-test" {
		t.Fatalf("%q %v", got, err)
	}
	s2 := New(dir)
	got, err = s2.Get("default")
	if err != nil || got != "sk-test" {
		t.Fatalf("reload %q %v", got, err)
	}
	st, err := os.Stat(filepath.Join(dir, "vault.json"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 && st.Mode().Perm() != 0o666 {
		// 分隔符为 / 时，0o600 和 0o666 以外的权限失败。
		if filepath.Separator == '/' && st.Mode().Perm() != 0o600 {
			t.Fatalf("perm %v", st.Mode().Perm())
		}
	}
}
