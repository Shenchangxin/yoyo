// templates.go 从连接模板展开不含密钥的供应商起步行。

package video

import "github.com/Shenchangxin/yoyo/internal/connection"

// ProviderTemplates 按连接模板的能力展开供应商行。这些行不含密钥，密钥进保险库，不写在这里。聊天、OTEL、MCP 和 protocol 能力被跳过，标识留空。
func ProviderTemplates() []Provider {
	var out []Provider
	for _, c := range connection.Templates() {
		caps := c.Capabilities
		if len(caps) == 0 {
			caps = []string{connection.CapChat}
		}
		for _, cap := range caps {
			if cap == connection.CapChat || cap == connection.CapOTEL || cap == connection.CapMCP || cap == connection.CapProtocol {
				continue
			}
			p := providerFromConn(c, cap, nil)
			p.ID = ""
			out = append(out, p)
		}
	}
	if out == nil {
		out = []Provider{}
	}
	return out
}

// ServiceTypes 返回 image、video、tts、music、matting、stock、asr、storage、search 和 workflow。
func ServiceTypes() []string {
	return []string{"image", "video", "tts", "music", "matting", "stock", "asr", "storage", "search", "workflow"}
}
