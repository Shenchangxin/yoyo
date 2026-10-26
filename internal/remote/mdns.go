// 本文件在局域网宣告 _yoyo._tcp。TXT 带 fp 和 v=1，端口是手机的 TLS 端口。
// 不宣告浏览器的 HTTP 端口。没有非回环地址时不宣告。listen 成功后调用 announce。

package remote

import (
	"net"
	"os"

	"github.com/hashicorp/mdns"
)

// announce 用主机名宣告 TLS 端口。主机名为空时用 yoyo。先停掉上一次的宣告。
// lanHosts 里的回环地址跳过。一个非回环地址都没有，或端口是 0，或建服务失败，就保持安静。
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
