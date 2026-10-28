package desktop

import (
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// companionWindowName 是陪伴窗的名字。实时条目不投给它，脉搏仍走 yoyo:pulse。
const companionWindowName = "yoyo-companion"

// windowEvents 沿用默认 HTTP 绑定，自己把事件送到窗口。
// yoyo:item 跳过陪伴窗。弹出的会话窗仍接收，那里也要画同一份转录。
// 事件仍从 Wails 的邮箱出来，顺序和 yoyo:state、yoyo:pulse 一致。
type windowEvents struct {
	*application.HTTPTransport
}

// NewWindowEvents 做成应用的 Transport。logger 为空时用 Wails 的默认日志。
func NewWindowEvents(logger *slog.Logger) application.Transport {
	var opts []application.HTTPTransportOption
	if logger != nil {
		opts = append(opts, application.HTTPTransportWithLogger(logger))
	}
	return &windowEvents{HTTPTransport: application.NewHTTPTransport(opts...)}
}

// DispatchWailsEvent 把一条事件送到每个窗口。名字是 yoyo:item 时跳过陪伴窗。
// event 为 nil 或已取消则返回。没有应用时返回。
func (t *windowEvents) DispatchWailsEvent(event *application.CustomEvent) {
	if event == nil || event.IsCancelled() {
		return
	}
	gui := application.Get()
	if gui == nil {
		return
	}
	skipPet := event.Name == "yoyo:item"
	for _, w := range gui.Window.GetAll() {
		if w == nil {
			continue
		}
		if skipPet && w.Name() == companionWindowName {
			continue
		}
		w.DispatchWailsEvent(event)
	}
}
