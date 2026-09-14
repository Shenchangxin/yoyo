package kernel

import "sync"

// Fiber is one loaded plugin instance and the owner of its effects.
type Fiber struct {
	Name     string
	host     *Context
	mu       sync.Mutex
	effects  []func() error
	disposed bool
}

func (f *Fiber) addEffect(d func() error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.effects = append(f.effects, d)
}

// Dispose runs effect disposers in reverse registration order.
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

func (f *Fiber) Disposed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.disposed
}
