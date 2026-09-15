package wasm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type Module struct {
	ID      string
	Export  string
	Timeout time.Duration
	rt      wazero.Runtime
	mod     api.Module
	logs    *strings.Builder
	scratch *[16]uint32
}

type Host struct {
	mu      sync.Mutex
	loaded  map[string]*Module
	maxSize int
}

func NewHost() *Host {
	return &Host{loaded: map[string]*Module{}, maxSize: 8 << 20}
}

func (h *Host) MaxSize() int { return h.maxSize }

func (h *Host) Load(id string, bin []byte, export string, timeout time.Duration) (*Module, error) {
	if len(bin) > h.maxSize {
		return nil, fmt.Errorf("wasm: module too large (%d)", len(bin))
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	if export == "" {
		export = "add"
	}
	rt := wazero.NewRuntime(context.Background())
	logs := &strings.Builder{}
	scratch := &[16]uint32{}
	nowFn := time.Now
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// Host funcs only — no WASI (no guest FS/net). Clock and scratch are
	// integers; abort traps; log is a capped sink.
	_, err := rt.NewHostModuleBuilder("yoyo").
		NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, off, n uint32) {
			if n > 2048 {
				n = 2048
			}
			buf, ok := m.Memory().Read(off, n)
			if !ok || logs.Len() > 8192 {
				return
			}
			logs.Write(buf)
		}).
		Export("log").
		NewFunctionBuilder().
		WithFunc(func() uint32 {
			return uint32(nowFn().Unix())
		}).
		Export("now").
		NewFunctionBuilder().
		WithFunc(func(code uint32) {
			panic(fmt.Sprintf("wasm abort %d", code))
		}).
		Export("abort").
		NewFunctionBuilder().
		WithFunc(func(i, v uint32) {
			if i < uint32(len(scratch)) {
				scratch[i] = v
			}
		}).
		Export("scratch_set").
		NewFunctionBuilder().
		WithFunc(func(i uint32) uint32 {
			if i < uint32(len(scratch)) {
				return scratch[i]
			}
			return 0
		}).
		Export("scratch_get").
		Instantiate(ctx)
	if err != nil {
		_ = rt.Close(context.Background())
		return nil, fmt.Errorf("wasm: host: %w", err)
	}
	mod, err := rt.Instantiate(ctx, bin)
	if err != nil {
		_ = rt.Close(context.Background())
		return nil, fmt.Errorf("wasm: instantiate: %w", err)
	}
	m := &Module{ID: id, Export: export, Timeout: timeout, rt: rt, mod: mod, logs: logs, scratch: scratch}
	h.mu.Lock()
	if old, ok := h.loaded[id]; ok {
		_ = old.Close()
	}
	h.loaded[id] = m
	h.mu.Unlock()
	return m, nil
}

func (h *Host) Get(id string) *Module {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.loaded[id]
}

func (h *Host) Unload(id string) error {
	h.mu.Lock()
	m, ok := h.loaded[id]
	delete(h.loaded, id)
	h.mu.Unlock()
	if !ok {
		return nil
	}
	return m.Close()
}

func (h *Host) List() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ids := make([]string, 0, len(h.loaded))
	for id := range h.loaded {
		ids = append(ids, id)
	}
	return ids
}

func (h *Host) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var first error
	for id, m := range h.loaded {
		if err := m.Close(); err != nil && first == nil {
			first = err
		}
		delete(h.loaded, id)
	}
	return first
}

func (m *Module) Close() error {
	if m == nil || m.rt == nil {
		return nil
	}
	err := m.rt.Close(context.Background())
	m.rt = nil
	m.mod = nil
	return err
}

func (m *Module) Logs() string {
	if m == nil || m.logs == nil {
		return ""
	}
	return m.logs.String()
}

func (m *Module) Scratch(i int) uint32 {
	if m == nil || m.scratch == nil || i < 0 || i >= len(m.scratch) {
		return 0
	}
	return m.scratch[i]
}

func (m *Module) CallI32(args ...uint64) (uint64, error) {
	if m == nil || m.mod == nil {
		return 0, fmt.Errorf("wasm: module disposed")
	}
	fn := m.mod.ExportedFunction(m.Export)
	if fn == nil {
		return 0, fmt.Errorf("wasm: missing export %q", m.Export)
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.Timeout)
	defer cancel()
	out, err := fn.Call(ctx, args...)
	if err != nil {
		return 0, err
	}
	if len(out) == 0 {
		return 0, nil
	}
	return out[0], nil
}
