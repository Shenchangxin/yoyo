package update

import "testing"

func TestIsNewer(t *testing.T) {
	if !IsNewer("0.5.0", "0.4.0") {
		t.Fatal("0.5.0")
	}
	if IsNewer("0.4.0", "0.4.0") {
		t.Fatal("equal")
	}
	if IsNewer("0.3.9", "0.4.0") {
		t.Fatal("older")
	}
	if !IsNewer("v1.0.0", "0.9.9") {
		t.Fatal("v prefix")
	}
	if !IsNewer("1.0.0", "1.0.0-beta.1") {
		t.Fatal("release > pre")
	}
	if IsNewer("1.0.0-beta.1", "1.0.0") {
		t.Fatal("pre < release")
	}
}

func TestChannelsForVersion(t *testing.T) {
	for _, ver := range []string{"0.5.0", "0.5.0-beta.1", "0.0.0-dev.abc1234"} {
		got := ChannelsForVersion(ver)
		if len(got) != 1 || got[0] != DefaultChannel {
			t.Fatalf("%s -> %v", ver, got)
		}
	}
	if ChannelsForVersion("") != nil {
		t.Fatal("empty version publishes nothing")
	}
}
