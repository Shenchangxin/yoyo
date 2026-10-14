//go:build !windows

package desktop

// clearToastHistory does nothing off Windows. Only the Windows Action Center keeps
// toasts after they are shown, so the other platforms have no history to empty here.
func clearToastHistory(string) error { return nil }
