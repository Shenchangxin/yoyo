package remote

import (
	"net"
	"os"

	"github.com/hashicorp/mdns"
)

func (n *Node) announce() {
	if n == nil {
		return
	}
	n.mu.Lock()
	if n.mdnsStop != nil {
		n.mdnsStop()
		n.mdnsStop = nil
	}
	port := n.port
	fp := n.fp
	n.mu.Unlock()
	if port == 0 {
		return
	}
	host, _ := os.Hostname()
	if host == "" {
		host = "yoyo"
	}
	var ips []net.IP
	for _, s := range lanHosts() {
		ip := net.ParseIP(s)
		if ip == nil || ip.IsLoopback() {
			continue
		}
		ips = append(ips, ip)
	}
	if len(ips) == 0 {
		return
	}
	svc, err := mdns.NewMDNSService(host, "_yoyo._tcp", "", "", port, ips, []string{"fp=" + fp, "v=1"})
	if err != nil {
		return
	}
	server, err := mdns.NewServer(&mdns.Config{Zone: svc})
	if err != nil {
		return
	}
	n.mu.Lock()
	n.mdnsStop = func() { _ = server.Shutdown() }
	n.mu.Unlock()
}
