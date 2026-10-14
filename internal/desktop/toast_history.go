package desktop

import "strings"

// Action Center keeps a toast until it is removed, and Wails can only remove one by its
// tag and group, which its toasts do not carry. So Yoyo does not retract single toasts.
// It empties its own history when the window comes back to the foreground: by then the
// user is in the app, and the bell still holds whatever is open.

// toastAppID is the AppUserModelID Wails registers for toasts. It is the application
// Name in main.go, so the history clear has to use the same string.
const toastAppID = "Yoyo"

// clearToastHistoryScript is the PowerShell that empties Action Center for one app.
// The ID is quoted so an apostrophe in it cannot end the string early.
func clearToastHistoryScript(appID string) string {
	q := strings.ReplaceAll(appID, "'", "''")
	return "[void][Windows.UI.Notifications.ToastNotificationManager,Windows.UI.Notifications,ContentType=WindowsRuntime];" +
		"[Windows.UI.Notifications.ToastNotificationManager]::History.Clear('" + q + "')"
}

// noteToastSent records that Action Center may now hold a toast of ours.
func (s *Service) noteToastSent() { s.toastsSinceClear.Store(true) }

// takeToastClear reports whether a toast went out since the last clear, and resets the mark.
func (s *Service) takeToastClear() bool { return s.toastsSinceClear.Swap(false) }

// clearToastHistoryOnFocus empties Action Center when the window gains focus. It only
// starts the clear when a toast went out since the last one, so frequent focus changes
// do not each start PowerShell.
func (s *Service) clearToastHistoryOnFocus() {
	if !s.takeToastClear() {
		return
	}
	go func() { _ = clearToastHistory(toastAppID) }()
}
