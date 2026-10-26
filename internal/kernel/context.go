// Package kernel 是插件的组合根。应用启动、个人能力和演化候选用 Plugin 把能力挂上来。
// Provide 见到同名服务已经注册就返回错误，不覆盖。
// Fiber.Dispose 按注册的相反顺序撤销效果。Context.Dispose 按挂载的相反顺序卸下各条 Fiber。
package kernel

import (
	"fmt"
	"sync"
)

// Context 是进程内的组合根，按名字保存服务，并记住当前正在挂载的 Fiber。
type Context struct {
	mu       sync.Mutex
	services map[string]any
	fibers   []*Fiber
	events   *EventBus
	// 零值 nil 表示不在 Plugin 的回调里，这时 Effect 会拒绝登记。
	current *Fiber
	name    string
}

// New 返回名字为 root 的内核。服务表是空的，事件总线已经可用，还没有 Fiber。
func New() *Context {
	return &Context{
		services: make(map[string]any),
		events:   NewEventBus(),
		name:     "root",
	}
}

// Events 返回事件总线。
func (c *Context) Events() *EventBus { return c.events }

// Name 返回 New 写入的名字。
func (c *Context) Name() string { return c.name }

// PluginFunc 在一次挂载里向 Context 登记服务和效果。它返回错误时，这次 Fiber 会被撤销。
type PluginFunc func(ctx *Context) error

// Plugin 以 name 挂上一条 Fiber，把它设为当前插件后调用 apply。
// 应用启动、个人能力和演化候选都会走到这里。
// apply 为 nil 时返回错误，不创建 Fiber。回调进行时这条 Fiber 已经在列表里。
// apply 返回错误时，已登记的效果会被撤销，Fiber 从列表移除，调用方得到 nil 和 apply 的错误。
// Dispose 自己的错误被丢掉。成功时 Fiber 留在列表里。这里不检查重名，同名可以挂上两条。
func (c *Context) Plugin(name string, apply PluginFunc) (*Fiber, error) {
	if apply == nil {
		return nil, fmt.Errorf("kernel: nil plugin %q", name)
	}
	f := &Fiber{Name: name, host: c}
	c.mu.Lock()
	prev := c.current
	c.current = f
	c.fibers = append(c.fibers, f)
	c.mu.Unlock()

	err := apply(c)

	c.mu.Lock()
	c.current = prev
	c.mu.Unlock()

	if err != nil {
		// 当前 Fiber 已恢复成外层。Dispose 的错误不向外传，避免盖住 apply 的失败。
		_ = f.Dispose()
		c.removeFiber(f)
		return nil, err
	}
	return f, nil
}

// removeFiber 按指针从列表去掉 target。列表里没有它时保持原样。
func (c *Context) removeFiber(target *Fiber) {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.fibers[:0]
	for _, f := range c.fibers {
		if f != target {
			out = append(out, f)
		}
	}
	c.fibers = out
}

// Effect 把 setup 的撤销函数记到当前 Fiber 上。只应在 Plugin 的回调里调用。
// 当前 Fiber 为 nil 时返回错误，setup 不会执行。
// setup 失败时不登记撤销函数，也不替 setup 回滚它已经做的事。
// setup 返回的 dispose 为 nil 时没有可撤销动作，调用仍然成功。
// 嵌套 Plugin 时，效果记在最内层那条 Fiber 上。
func (c *Context) Effect(setup func() (dispose func() error, err error)) error {
	c.mu.Lock()
	f := c.current
	c.mu.Unlock()
	if f == nil {
		return fmt.Errorf("kernel: effect registered outside a fiber")
	}
	dispose, err := setup()
	if err != nil {
		return err
	}
	if dispose != nil {
		f.addEffect(dispose)
	}
	return nil
}

// Provide 以 name 登记服务 svc，并在拥有它的 Fiber 卸下时按名字从表里删除。
// 见到同名服务已经注册就返回错误，不覆盖，svc 也不会写入。
// 必须在 Fiber 内调用。失败时服务表与调用前一致。
// 可以把 nil 登记进去；这个名字随后仍算存在，再次 Provide 还会被拒绝。
func (c *Context) Provide(name string, svc any) error {
	return c.Effect(func() (func() error, error) {
		c.mu.Lock()
		if _, exists := c.services[name]; exists {
			c.mu.Unlock()
			// 同名已经登记就拒绝，不覆盖。nil 也算这个名字已存在。
			return nil, fmt.Errorf("kernel: service %q already registered", name)
		}
		c.services[name] = svc
		c.mu.Unlock()
		return func() error {
			c.mu.Lock()
			delete(c.services, name)
			c.mu.Unlock()
			return nil
		}, nil
	})
}

// Inject 在调用这一刻检查 names 里的服务是否都已登记。缺任何一个就返回错误，后面的名字不再看。
// 名字列表为空时直接成功。它不登记效果，服务以后被卸下时也不会再通知。
func (c *Context) Inject(names ...string) error {
	for _, name := range names {
		if _, ok := c.Get(name); !ok {
			return fmt.Errorf("kernel: missing required service %q", name)
		}
	}
	return nil
}

// Get 按名字取出服务。没有这个名字时 ok 为 false。用 nil 登记过的名字返回 nil, true。
// name 作为 services 的键。
func (c *Context) Get(name string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.services[name]
	return s, ok
}

// MustGet 按名字取出服务。名字不存在时 panic。登记过的 nil 会原样返回，不 panic。
// name 交给 Get；不存在时 panic，文本里带上这个名字。
func (c *Context) MustGet(name string) any {
	s, ok := c.Get(name)
	if !ok {
		panic("kernel: missing service " + name)
	}
	return s
}

// Fibers 按挂载顺序列出尚未卸下的 Fiber 名字。
func (c *Context) Fibers() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	names := make([]string, 0, len(c.fibers))
	for _, f := range c.fibers {
		if !f.Disposed() {
			names = append(names, f.Name)
		}
	}
	return names
}

// Fiber 返回同名且尚未卸下的第一条。没有则返回 nil。
// name 与尚未卸下的 Fiber 的 Name 比较，返回第一条。
func (c *Context) Fiber(name string) *Fiber {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, f := range c.fibers {
		if f.Name == name && !f.Disposed() {
			return f
		}
	}
	return nil
}

// Dispose 先复制当时的 Fiber 列表，再按挂载的相反顺序卸下。复制之后新挂上的不在这次里面。
// 服务表靠各 Fiber 上的撤销函数清理，这里不按名字逐个删除。
// 某条卸下失败时，其余的仍会卸，最后只返回第一个错误。
func (c *Context) Dispose() error {
	c.mu.Lock()
	fibers := append([]*Fiber(nil), c.fibers...)
	c.mu.Unlock()
	var first error
	// 后挂上的先卸。
	for i := len(fibers) - 1; i >= 0; i-- {
		if err := fibers[i].Dispose(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
