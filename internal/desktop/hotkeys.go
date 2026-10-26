package desktop

// startGlobalHotkeys 在单独的 goroutine 里注册唤醒热键。
func startGlobalHotkeys(s *Service) {
	go registerWakeHotkey(s)
}
