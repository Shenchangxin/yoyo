// 锁住文件往返：Set 写入的 default 能被 Get 读回，同一目录再次 New 后仍然能读到。
// vault.json 权限为 0o600 或 0o666 时通过；其他权限只在路径分隔符是 / 时失败。
// 锁住测试进程不碰操作系统钥匙串：New 在测试里只写文件；只写文件的仓库写、读、删、重开都不调钥匙串，
// 也读不到、删不掉钥匙串里同名的那一条。启用钥匙串的那条路用记数的假钥匙串锁住，不经过真的。
// 本测试不锁环境变量回落。

package vault

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// fakeKeychain 是记数的假钥匙串。items 是里面现有的条目，calls 记下每一次调用。
type fakeKeychain struct {
	items map[string]string
	calls []string
}

// swapKeychain 把钥匙串的三个操作换成假的，测试结束时换回去。seed 是假钥匙串里原有的条目。
func swapKeychain(t *testing.T, seed map[string]string) *fakeKeychain {
	t.Helper()
	k := &fakeKeychain{items: map[string]string{}}
	for name, value := range seed {
		k.items[name] = value
	}
	set, get, del := kcSet, kcGet, kcDelete
	kcSet = func(name, value string) error {
		k.calls = append(k.calls, "set "+name)
		k.items[name] = value
		return nil
	}
	kcGet = func(name string) (string, error) {
		k.calls = append(k.calls, "get "+name)
		v, ok := k.items[name]
		if !ok {
			return "", errors.New("not found")
		}
		return v, nil
	}
	kcDelete = func(name string) error {
		k.calls = append(k.calls, "delete "+name)
		delete(k.items, name)
		return nil
	}
	t.Cleanup(func() { kcSet, kcGet, kcDelete = set, get, del })
	return k
}

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

// TestNewInTestsIsFileOnly 锁住测试进程里 New 不启用钥匙串。钥匙串不分 home，
// 测试开在临时目录里的仓库一旦启用它，改的就是这台电脑上真实的密钥。
func TestNewInTestsIsFileOnly(t *testing.T) {
	k := swapKeychain(t, map[string]string{"default": "the person's key"})
	dir := t.TempDir()
	s := New(dir)
	if on, _ := s.Status()["keychain"].(bool); on {
		t.Fatal("New enabled the keychain inside a test binary")
	}
	s.Set("default", "sk-test")
	s.Set("conn.c1.key", "sk-conn")
	if got, err := s.Get("default"); err != nil || got != "sk-test" {
		t.Fatalf("get = %q %v", got, err)
	}
	if got, err := s.GetStored("absent"); err == nil {
		t.Fatalf("absent key = %q", got)
	}
	s.Set("conn.c1.key", "")
	// 重开时文件里已有条目：启用钥匙串的仓库会在这里把它们迁进去。
	if got, err := New(dir).Get("default"); err != nil || got != "sk-test" {
		t.Fatalf("reopen = %q %v", got, err)
	}
	if len(k.calls) != 0 {
		t.Fatalf("keychain calls = %v", k.calls)
	}
	if got := k.items["default"]; got != "the person's key" {
		t.Fatalf("keychain default = %q", got)
	}
}

// TestFileOnlyStoreLeavesTheKeychainAlone 锁住只写文件的仓库：钥匙串里同名的那一条它读不到，删自己的密钥时也不删它。
func TestFileOnlyStoreLeavesTheKeychainAlone(t *testing.T) {
	k := swapKeychain(t, map[string]string{"default": "the person's key"})
	s := NewFileOnly(t.TempDir())
	if got, err := s.GetStored("default"); err == nil {
		t.Fatalf("file-only store read the keychain: %q", got)
	}
	s.Set("default", "mine")
	s.Set("default", "")
	if got, err := s.GetStored("default"); err == nil {
		t.Fatalf("deleted key still reads %q", got)
	}
	if len(k.calls) != 0 {
		t.Fatalf("keychain calls = %v", k.calls)
	}
	if got := k.items["default"]; got != "the person's key" {
		t.Fatalf("keychain default = %q", got)
	}
}

// TestKeychainStoreWritesReadsAndDeletes 锁住启用钥匙串的那条路：写入和删除都落到钥匙串，
// 内存和文件里没有的名字从钥匙串读，重开时把文件里的条目迁进去。
func TestKeychainStoreWritesReadsAndDeletes(t *testing.T) {
	k := swapKeychain(t, map[string]string{"elsewhere": "from keychain"})
	dir := t.TempDir()
	s := open(dir, true)
	s.Set("default", "sk-test")
	if got := k.items["default"]; got != "sk-test" {
		t.Fatalf("keychain default = %q", got)
	}
	if got, err := s.GetStored("elsewhere"); err != nil || got != "from keychain" {
		t.Fatalf("keychain read = %q %v", got, err)
	}
	s.Set("default", "")
	if _, ok := k.items["default"]; ok {
		t.Fatal("delete left the keychain entry")
	}

	s.Set("kept", "sk-kept")
	delete(k.items, "kept")
	open(dir, true)
	if got := k.items["kept"]; got != "sk-kept" {
		t.Fatalf("reopen migrated %q", got)
	}
}
