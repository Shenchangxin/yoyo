// 锁住这些不变量：Provide 的服务随 Fiber 卸下而消失；放进 Effect 的监听在 Context.Dispose 后退订；
// Fiber 外面调用 Effect 会失败；Inject 缺服务时挂载失败；Waterfall 按登记顺序拼接载荷。
// 不拥有生产环境里的插件名单。

package kernel

import "testing"

// TestPluginProvideAndDispose 锁住 Fiber.Dispose 之后，服务不能再 Get 到，Fiber 列表里也没有这条。
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

// TestEffectOutsideFiber 锁住没有当前 Fiber 时 Effect 返回错误。
func TestEffectOutsideFiber(t *testing.T) {
	ctx := New()
	err := ctx.Effect(func() (func() error, error) {
		return func() error { return nil }, nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEventUnsubscribeOnDispose 锁住退订放进 Effect 后，Context.Dispose 再 Emit 不会增加命中次数。
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

// TestWaterfallTransforms 锁住两位监听者把载荷从 x 收成 x-a-b。
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

// TestInjectMissing 锁住 Inject 找不到 models 时 Plugin 返回错误。
func TestInjectMissing(t *testing.T) {
	ctx := New()
	_, err := ctx.Plugin("need", func(c *Context) error {
		return c.Inject("models")
	})
	if err == nil {
		t.Fatal("expected missing service")
	}
}
