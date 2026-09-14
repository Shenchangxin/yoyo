package kernel

import (
	"fmt"
	"sync"
)

// Context is the spatiotemporal composition root. Plugins register revertible
// effects; disposing a fiber undoes them in reverse order.
type Context struct {
	mu       sync.Mutex
	services map[string]any
	fibers   []*Fiber
	events   *EventBus
	current  *Fiber
	name     string
}

// New returns a root kernel context.
func New() *Context {
	return &Context{
		services: make(map[string]any),
		events:   NewEventBus(),
		name:     "root",
	}
}

func (c *Context) Events() *EventBus { return c.events }

func (c *Context) Name() string { return c.name }

// PluginFunc mounts a capability onto a context.
type PluginFunc func(ctx *Context) error

// Plugin mounts apply as a named fiber. On failure the fiber is disposed.
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
		_ = f.Dispose()
		c.removeFiber(f)
		return nil, err
	}
	return f, nil
}

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

// Effect registers a revertible setup against the current fiber.
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

// Provide registers a named service that is removed when the owning fiber unloads.
func (c *Context) Provide(name string, svc any) error {
	return c.Effect(func() (func() error, error) {
		c.mu.Lock()
		if _, exists := c.services[name]; exists {
			c.mu.Unlock()
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

// Inject requires services to exist at mount time (spatial composability, v1).
func (c *Context) Inject(names ...string) error {
	for _, name := range names {
		if _, ok := c.Get(name); !ok {
			return fmt.Errorf("kernel: missing required service %q", name)
		}
	}
	return nil
}

func (c *Context) Get(name string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.services[name]
	return s, ok
}

func (c *Context) MustGet(name string) any {
	s, ok := c.Get(name)
	if !ok {
		panic("kernel: missing service " + name)
	}
	return s
}

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

// Dispose unloads every fiber in reverse mount order.
func (c *Context) Dispose() error {
	c.mu.Lock()
	fibers := append([]*Fiber(nil), c.fibers...)
	c.mu.Unlock()
	var first error
	for i := len(fibers) - 1; i >= 0; i-- {
		if err := fibers[i].Dispose(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
