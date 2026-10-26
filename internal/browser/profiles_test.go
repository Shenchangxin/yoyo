// 本文件锁住配置名校验，以及没启动 Chrome 时也能换配置名。由 go test 调用。

package browser

import "testing"

// TestSanitizeProfileName 锁住空名是 default，attached 和空格被拒绝，github 可以通过。
func TestSanitizeProfileName(t *testing.T) {
	got, err := SanitizeProfileName("")
	if err != nil || got != "default" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := SanitizeProfileName("attached"); err == nil {
		t.Fatal("attached is reserved")
	}
	if _, err := SanitizeProfileName("has space"); err == nil {
		t.Fatal("spaces rejected")
	}
	got, err = SanitizeProfileName("github")
	if err != nil || got != "github" {
		t.Fatalf("%q %v", got, err)
	}
}

// TestUseProfileChangesNameWithoutChrome 锁住 UseProfile 把名字改成 work，列表第一项仍是 default。
func TestUseProfileChangesNameWithoutChrome(t *testing.T) {
	h, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	if err := h.UseProfile("work"); err != nil {
		t.Fatal(err)
	}
	if h.ProfileName() != "work" {
		t.Fatalf("name %s", h.ProfileName())
	}
	listed := h.ListProfiles()
	if listed[0] != "default" {
		t.Fatalf("%v", listed)
	}
}
