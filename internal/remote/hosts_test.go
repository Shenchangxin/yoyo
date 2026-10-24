package remote

import (
	"net"
	"testing"
)

func TestPrivatePeerAllowsLANAndLoopback(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:1", "192.168.1.8:9", "10.0.2.2:443", "[::1]:1"} {
		if !privatePeer(addr) {
			t.Fatalf("expected private %s", addr)
		}
	}
	if privatePeer("8.8.8.8:443") {
		t.Fatal("public address must not reach the workstation")
	}
	if !isPrivate(net.ParseIP("172.16.0.4")) || isPrivate(net.ParseIP("1.1.1.1")) {
		t.Fatal("range")
	}
}
