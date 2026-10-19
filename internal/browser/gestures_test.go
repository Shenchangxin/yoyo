package browser

import "testing"

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
