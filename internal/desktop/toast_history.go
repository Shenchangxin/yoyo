package desktop

import "strings"

// 操作中心会留着通知直到被删掉，而 Wails 只能按 tag 和 group 删一条，它发出的通知不带这两个字段。所以 Yoyo 不撤回单条通知。
// 窗口回到前台时清空自己的通知历史：这时人已经在应用里，铃铛仍留着还没处理的记录。

// toastAppID 是 Wails 注册通知用的 AppUserModelID，等于 main.go 里的应用名 Yoyo。清空操作中心时必须用同一个字符串。
const toastAppID = "Yoyo"

// clearToastHistoryScript 生成清空某个应用操作中心历史的 PowerShell。单引号会加倍，避免提前结束字符串。
// appID 里的单引号加倍后放进 History.Clear 的参数。
func clearToastHistoryScript(appID string) string {
	q := strings.ReplaceAll(appID, "'", "''")
	return "[void][Windows.UI.Notifications.ToastNotificationManager,Windows.UI.Notifications,ContentType=WindowsRuntime];" +
		"[Windows.UI.Notifications.ToastNotificationManager]::History.Clear('" + q + "')"
}

// noteToastSent 记下操作中心里可能有我们发出的通知。
func (s *Service) noteToastSent() { s.toastsSinceClear.Store(true) }

// takeToastClear 报告上次清空之后是否发过通知，并把标记复位。
func (s *Service) takeToastClear() bool { return s.toastsSinceClear.Swap(false) }

// clearToastHistoryOnFocus 在窗口获得焦点时清空操作中心。上次清空后没发过通知则直接返回，避免每次聚焦都启动 PowerShell。
func (s *Service) clearToastHistoryOnFocus() {
	if !s.takeToastClear() {
		return
	}
	go func() { _ = clearToastHistory(toastAppID) }()
}
