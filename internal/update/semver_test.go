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
	if got := ChannelsForVersion("0.5.0"); len(got) != 3 || got[0] != ChannelStable {
		t.Fatalf("%v", got)
	}
	if got := ChannelsForVersion("0.5.0-beta.1"); len(got) != 2 || got[0] != ChannelBeta {
		t.Fatalf("%v", got)
	}
	if got := ChannelsForVersion("0.0.0-dev.abc1234"); len(got) != 1 || got[0] != ChannelNightly {
		t.Fatalf("%v", got)
	}
}
