// 锁住回环和局域网地址算私网，公网地址不算。不枚举网卡。

package remote

import (
	"net"
	"testing"
)

// TestPrivatePeerAllowsLANAndLoopback 锁住 127.0.0.1、192.168.1.8、10.0.2.2 和 [::1] 算私网，8.8.8.8 不算。172.16.0.4 算私网，1.1.1.1 不算。
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
