// 本文件拥有一条已挂载插件 Fiber，以及它按登记顺序保存的撤销函数。
// 不拥有服务表，也不拥有事件总线；那两份在 Context 上。
// Context.Plugin 创建它。调用方和 Context.Dispose 调用 Dispose。

package kernel

import "sync"

// Fiber 是一次 Plugin 挂载留下的实例。
type Fiber struct {
	Name string
	// 为 nil 时 Dispose 不回写宿主列表。Plugin 创建的实例总会带上宿主。
	host *Context
	mu   sync.Mutex
	// 按登记顺序追加。Dispose 从最后一条往前调用。
	effects  []func() error
	disposed bool
}

// addEffect 把撤销函数接到登记顺序的末尾。调用方是 Context.Effect。
func (f *Fiber) addEffect(d func() error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.effects = append(f.effects, d)
}

// Dispose 按注册的相反顺序撤销。已经卸过则直接返回 nil，撤销函数不会再跑。
// 某个撤销函数返回错误时，剩下的仍会调用，最后只返回第一个错误。
// 卸下标志在调用撤销函数之前就置上，所以失败后再调用不会重试。
// 不论撤销是否报错，只要 host 非空，都会把自己从 Context 的列表里移除。
func (f *Fiber) Dispose() error {
	f.mu.Lock()
	if f.disposed {
		f.mu.Unlock()
		return nil
	}
	f.disposed = true
	effects := append([]func() error(nil), f.effects...)
	f.mu.Unlock()
	var first error
	// 与注册顺序相反。
	for i := len(effects) - 1; i >= 0; i-- {
		if err := effects[i](); err != nil && first == nil {
			first = err
		}
	}
	if f.host != nil {
		f.host.removeFiber(f)
	}
	return first
}

// Disposed 报告是否已经卸下。
func (f *Fiber) Disposed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.disposed
}
