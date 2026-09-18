//go:build !windows

package computeruse

func (h *Host) ensureVirtual() {
	h.mu.Lock()
	h.display = "virtual"
	h.mu.Unlock()
}

func (h *Host) closeVirtual() {}
