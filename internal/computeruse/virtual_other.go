//go:build !windows

// 非 Windows 上只把 display 字符串设为 virtual，不创建显示设备。

package computeruse

// ensureVirtual 把 display 设为 virtual。
func (h *Host) ensureVirtual() {
	h.mu.Lock()
	h.display = "virtual"
	h.mu.Unlock()
}

// closeVirtual 不做任何事。
func (h *Host) closeVirtual() {}
