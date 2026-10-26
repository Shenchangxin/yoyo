// Package wasm 用 wazero 装入并调用访客模块。不提供 WASI。NewHost 把模块字节上限设为 8<<20（8 MiB），比的是 len(bin)，不是访客线性内存。Load 在超时小于等于 0 时使用 2 秒。ed25519 验签在 sign.go：本包不读 YOYO_WASM_PUBKEY，也不写入 active。Host.Load 不验签，未签名模块可以装入。
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

// Module 是一份已实例化的模块。Timeout 是后续调用的时限。Export 是 CallI32 的导出名；CallJSON 优先找 call_json。
type Module struct {
	ID      string
	Export  string
	Timeout time.Duration
	rt      wazero.Runtime
	mod     api.Module
	logs    *strings.Builder
	scratch *[16]uint32
}

// Host 按 id 保存已装入的模块。maxSize 是 Load 接受的模块字节上限。
type Host struct {
	mu      sync.Mutex
	loaded  map[string]*Module
	maxSize int
}

// NewHost 返回空表，并把 maxSize 设为 8<<20（8 MiB）。
func NewHost() *Host {
	return &Host{loaded: map[string]*Module{}, maxSize: 8 << 20}
}

// MaxSize 返回 maxSize。
func (h *Host) MaxSize() int { return h.maxSize }

// Load 装入一份 WASM 模块。len(bin) 大于 maxSize 时返回 "wasm: module too large (%d)"；NewHost 把该上限设为 8<<20（8 MiB）。timeout 小于等于 0 时改为 2 秒。export 为空时用 "add"。只注册宿主模块 "yoyo"，不提供 WASI，访客没有文件系统和网络。log 单次读取超过 2048 字节会截到 2048；日志累计长度已经大于 8192 字节时，这次不再写入。now 返回 Unix 秒。abort 以 panic 抛出 "wasm abort %d"。scratch_set 与 scratch_get 使用长度为 16 的 uint32 数组，下标越界则忽略写入或返回 0。装入时的 context 超时就是这份 timeout。之后的 CallI32 和 CallJSON 各自再开一份同样的 Timeout。本函数不验签，未签名字节也会装入。同一 id 已存在时先关闭旧模块。
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

// Get 按 id 返回模块。没有则返回 nil。
func (h *Host) Get(id string) *Module {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.loaded[id]
}

// Unload 卸下并关闭该模块。id 不存在时返回 nil。
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

// List 返回已装入的 id，顺序不定。
func (h *Host) List() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ids := make([]string, 0, len(h.loaded))
	for id := range h.loaded {
		ids = append(ids, id)
	}
	return ids
}

// Close 关闭全部模块并清空表，返回遇到的第一个错误。
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

// Close 关闭这份模块的 runtime。模块或 runtime 为 nil 时返回 nil。
func (m *Module) Close() error {
	if m == nil || m.rt == nil {
		return nil
	}
	err := m.rt.Close(context.Background())
	m.rt = nil
	m.mod = nil
	return err
}

// Logs 返回 log 宿主函数累积的文本。模块或缓冲为 nil 时返回空串。
func (m *Module) Logs() string {
	if m == nil || m.logs == nil {
		return ""
	}
	return m.logs.String()
}

// Scratch 返回槽 i 的 uint32。模块为 nil，或下标不在 0 到 15，则返回 0。
func (m *Module) Scratch(i int) uint32 {
	if m == nil || m.scratch == nil || i < 0 || i >= len(m.scratch) {
		return 0
	}
	return m.scratch[i]
}

// CallI32 调用 Export，参数和返回值按 uint64 传递。模块已关闭时返回 "wasm: module disposed"。没有该导出时返回 "wasm: missing export"。调用时限是 Timeout。没有返回值时返回 0。
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

// CallJSON 按 L2 访客 ABI 调用：JSON 字节进、JSON 字节出，没有 WASI 文件系统。访客必须导出 alloc(i32)->i32，并导出 call_json(i32,i32)->i64（高 32 位是指针，低 32 位是长度）；没有 call_json 时改用 Export，签名相同。缺少 alloc 返回 "wasm: JSON ABI requires alloc"。两者都没有时返回 "wasm: missing call_json export"。输出长度为 0 或没有打包返回值时返回 {}。调用时限是 Timeout。
func (m *Module) CallJSON(input []byte) ([]byte, error) {
	if m == nil || m.mod == nil {
		return nil, fmt.Errorf("wasm: module disposed")
	}
	mem := m.mod.Memory()
	if mem == nil {
		return nil, fmt.Errorf("wasm: no memory")
	}
	alloc := m.mod.ExportedFunction("alloc")
	if alloc == nil {
		return nil, fmt.Errorf("wasm: JSON ABI requires alloc")
	}
	fn := m.mod.ExportedFunction("call_json")
	if fn == nil {
		fn = m.mod.ExportedFunction(m.Export)
	}
	if fn == nil {
		return nil, fmt.Errorf("wasm: missing call_json export")
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.Timeout)
	defer cancel()
	n := uint64(len(input))
	ptrOut, err := alloc.Call(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("wasm: alloc: %w", err)
	}
	if len(ptrOut) == 0 {
		return nil, fmt.Errorf("wasm: alloc returned nothing")
	}
	ptr := uint32(ptrOut[0])
	if n > 0 && !mem.Write(ptr, input) {
		return nil, fmt.Errorf("wasm: write input failed")
	}
	packed, err := fn.Call(ctx, uint64(ptr), n)
	if err != nil {
		return nil, err
	}
	if len(packed) == 0 {
		return []byte("{}"), nil
	}
	v := packed[0]
	outPtr := uint32(v >> 32)
	outLen := uint32(v)
	if outLen == 0 {
		return []byte("{}"), nil
	}
	buf, ok := mem.Read(outPtr, outLen)
	if !ok {
		return nil, fmt.Errorf("wasm: read output failed")
	}
	out := make([]byte, len(buf))
	copy(out, buf)
	return out, nil
}
