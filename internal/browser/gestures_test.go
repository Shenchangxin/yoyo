// 本文件锁住命名按键：enter 变成 Enter，未知的 F13 失败，单个字符走 keyFor。
// 由 go test 调用。

package browser

import "testing"

// TestNamedKey 锁住 enter 的 key 是 Enter，F13 返回错误，单个字符 a 的 key 是 a。
func TestNamedKey(t *testing.T) {
	k, err := namedKey("enter")
	if err != nil || k.key != "Enter" {
		t.Fatalf("%+v %v", k, err)
	}
	if _, err := namedKey("F13"); err == nil {
		t.Fatal("unknown key must fail")
	}
	k, err = namedKey("a")
	if err != nil || k.key != "a" {
		t.Fatalf("rune %+v %v", k, err)
	}
}
