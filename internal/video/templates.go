package video

import "github.com/Shenchangxin/yoyo/internal/connection"

// ProviderTemplates are non-secret starter rows. Keys go to vault, never here.
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

func ServiceTypes() []string {
	return []string{"image", "video", "tts", "storage", "search", "workflow"}
}
