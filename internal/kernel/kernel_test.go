package kernel

import "testing"

func TestPluginProvideAndDispose(t *testing.T) {
	ctx := New()
	fiber, err := ctx.Plugin("tools", func(c *Context) error {
		return c.Provide("tools", map[string]int{"n": 1})
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ctx.Get("tools"); !ok {
		t.Fatal("expected tools service")
	}
	if err := fiber.Dispose(); err != nil {
		t.Fatal(err)
	}
	if _, ok := ctx.Get("tools"); ok {
		t.Fatal("service leaked after dispose")
	}
	if ctx.Fiber("tools") != nil {
		t.Fatal("fiber still listed after dispose")
	}
}

func TestEffectOutsideFiber(t *testing.T) {
	ctx := New()
	err := ctx.Effect(func() (func() error, error) {
		return func() error { return nil }, nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEventUnsubscribeOnDispose(t *testing.T) {
	ctx := New()
	var hits int
	_, err := ctx.Plugin("obs", func(c *Context) error {
		return c.Effect(func() (func() error, error) {
			off := c.Events().On("tick", func(payload any) (any, error) {
				hits++
				return nil, nil
			})
			return func() error { off(); return nil }, nil
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx.Events().Emit("tick", nil)
	if hits != 1 {
		t.Fatalf("hits=%d", hits)
	}
	if err := ctx.Dispose(); err != nil {
		t.Fatal(err)
	}
	ctx.Events().Emit("tick", nil)
	if hits != 1 {
		t.Fatalf("listener leaked, hits=%d", hits)
	}
}

func TestWaterfallTransforms(t *testing.T) {
	b := NewEventBus()
	b.On("agent/request", func(payload any) (any, error) {
		return payload.(string) + "-a", nil
	})
	b.On("agent/request", func(payload any) (any, error) {
		return payload.(string) + "-b", nil
	})
	out, err := b.Waterfall("agent/request", "x")
	if err != nil {
		t.Fatal(err)
	}
	if out != "x-a-b" {
		t.Fatalf("got %v", out)
	}
}

func TestInjectMissing(t *testing.T) {
	ctx := New()
	_, err := ctx.Plugin("need", func(c *Context) error {
		return c.Inject("models")
	})
	if err == nil {
		t.Fatal("expected missing service")
	}
}
