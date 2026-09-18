package desktop

func startGlobalHotkeys(s *Service) {
	go registerWakeHotkey(s)
}
