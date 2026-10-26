// 本文件判断来源是不是私网，并挑出写进二维码和 mDNS 的 IPv4。
// 不监听端口。privatePeer 给 WebSocket、媒体和浏览器门用。lanHosts 给 BeginPair 和 announce 用。

package remote

import (
	"net"
	"strings"
)

// isPrivate 判断地址是不是回环、链路本地单播，或 net.IP.IsPrivate。nil 返回假。
func isPrivate(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate() {
		return true
	}
	return false
}

// privatePeer 从来源地址取出主机并交给 isPrivate。SplitHostPort 失败时把整串当主机。
// 这只区分地址类别。浏览器门即使放行，走的仍是明文 HTTP。
func privatePeer(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return isPrivate(net.ParseIP(host))
}

// skipIface 按名字判断虚拟网卡或回环。转成小写后，只要包含 vethernet、hyper-v、virtualbox、docker、vmware、bluetooth、loopback、wsl、npcap、tailscale 之一就跳过。
func skipIface(name string) bool {
	n := strings.ToLower(name)
	for _, part := range []string{"vethernet", "hyper-v", "virtualbox", "docker", "vmware", "bluetooth", "loopback", "wsl", "npcap", "tailscale"} {
		if strings.Contains(n, part) {
			return true
		}
	}
	return false
}

// lanHosts 优先返回正在工作的真实局域网 IPv4。回环地址，以及 skipIface 认定的网卡，只有在没有真实局域网地址时才用。
// 都没有，或网卡列不出来，就返回 127.0.0.1。IPv6 不收。
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
