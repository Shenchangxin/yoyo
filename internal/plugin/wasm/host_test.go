// 用只导出 add 的最小模块检查装入、调用、JSON ABI 和体积上限。

package wasm

import (
	"testing"
	"time"
)

// addWasm 是只导出 add(i32,i32)->i32 的最小 WASM 字节。
var addWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x07, 0x01, 0x60,
	0x02, 0x7f, 0x7f, 0x01, 0x7f, 0x03, 0x02, 0x01, 0x00, 0x07, 0x07, 0x01,
	0x03, 0x61, 0x64, 0x64, 0x00, 0x00, 0x0a, 0x09, 0x01, 0x07, 0x00, 0x20,
	0x00, 0x20, 0x01, 0x6a, 0x0b,
}

// TestLoadCallUnload 装入 add，确认 2 加 40 等于 42，卸载后 CallI32 失败。
func TestLoadCallUnload(t *testing.T) {
	h := NewHost()
	defer h.Close()
	m, err := h.Load("math", addWasm, "add", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.CallI32(2, 40)
	if err != nil {
		t.Fatal(err)
	}
	if got != 42 {
		t.Fatalf("got %d", got)
	}
	if err := h.Unload("math"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.CallI32(1, 1); err == nil {
		t.Fatal("expected disposed")
	}
}

// TestCallJSONRequiresAlloc 确认这份没有 alloc 的模块调用 CallJSON 会失败。
func TestCallJSONRequiresAlloc(t *testing.T) {
	h := NewHost()
	defer h.Close()
	m, err := h.Load("math", addWasm, "add", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.CallJSON([]byte(`{"a":1}`)); err == nil {
		t.Fatal("expected JSON ABI error")
	}
}

// TestRejectHuge 把 maxSize 设为 8 后，Load 必须因模块过大拒绝 addWasm。
func TestRejectHuge(t *testing.T) {
	h := NewHost()
	h.maxSize = 8
	if _, err := h.Load("x", addWasm, "add", 0); err == nil {
		t.Fatal("expected too large")
	}
}

// TestHostModuleInstantiatesWithClock 确认装入后 Scratch(0) 为 0。
func TestHostModuleInstantiatesWithClock(t *testing.T) {
	h := NewHost()
	defer h.Close()
	m, err := h.Load("math", addWasm, "add", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if m.Scratch(0) != 0 {
		t.Fatal("scratch should start zero")
	}
}
