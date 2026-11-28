package app

import (
	"strings"

	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// finishText 在工单报告后面接上交给人的最后一步：价格、截止和要打开的页面。主会话和 IM 主通道读到的是同一段。
func finishText(text string, f *superagent.Finish) string {
	if f == nil {
		return text
	}
	var lines []string
	if f.Price != "" {
		lines = append(lines, "价格："+f.Price)
	}
	if f.Deadline != "" {
		lines = append(lines, "截止："+f.Deadline)
	}
	if f.URL != "" {
		lines = append(lines, "打开："+f.URL)
	}
	if len(lines) == 0 {
		return text
	}
	return strings.TrimSpace(text + "\n" + strings.Join(lines, "\n"))
}

// finishPayload 是交接卡在会话载荷里的样子。
func finishPayload(f *superagent.Finish) map[string]any {
	out := map[string]any{"what": f.What}
	for k, v := range map[string]string{"url": f.URL, "price": f.Price, "deadline": f.Deadline} {
		if v != "" {
			out[k] = v
		}
	}
	return out
}
