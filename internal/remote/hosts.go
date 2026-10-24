package remote

import (
	"net"
	"strings"
)

func isPrivate(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate() {
		return true
	}
	return false
}

func privatePeer(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return isPrivate(net.ParseIP(host))
}

func skipIface(name string) bool {
	n := strings.ToLower(name)
	for _, part := range []string{"vethernet", "hyper-v", "virtualbox", "docker", "vmware", "bluetooth", "loopback", "wsl", "npcap", "tailscale"} {
		if strings.Contains(n, part) {
			return true
		}
	}
	return false
}

// lanHosts prefers real LAN addresses over virtual adapters and loopback.
func lanHosts() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{"127.0.0.1"}
	}
	var lan, local []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		virtual := skipIface(iface.Name)
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || !isPrivate(ip) {
				continue
			}
			s := ip.String()
			if ip.IsLoopback() || virtual {
				local = append(local, s)
				continue
			}
			lan = append(lan, s)
		}
	}
	if len(lan) > 0 {
		return lan
	}
	if len(local) > 0 {
		return local
	}
	return []string{"127.0.0.1"}
}
