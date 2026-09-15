package wasm

import (
	"testing"
	"time"
)

// add(i32,i32)->i32
var addWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x07, 0x01, 0x60,
	0x02, 0x7f, 0x7f, 0x01, 0x7f, 0x03, 0x02, 0x01, 0x00, 0x07, 0x07, 0x01,
	0x03, 0x61, 0x64, 0x64, 0x00, 0x00, 0x0a, 0x09, 0x01, 0x07, 0x00, 0x20,
	0x00, 0x20, 0x01, 0x6a, 0x0b,
}

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

func TestRejectHuge(t *testing.T) {
	h := NewHost()
	h.maxSize = 8
	if _, err := h.Load("x", addWasm, "add", 0); err == nil {
		t.Fatal("expected too large")
	}
}

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
