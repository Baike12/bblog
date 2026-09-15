package render

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// mathTimeout 是单个公式的渲染上限，超过即中断 JS 执行并降级输出。
const mathTimeout = 200 * time.Millisecond

// MathRenderer 用 goja 执行 KaTeX 官方 JS 包，在服务端把 TeX 渲染成 HTML。
type MathRenderer struct {
	mu     sync.Mutex
	vm     *goja.Runtime
	render goja.Callable
}

// NewMathRenderer 加载 KaTeX 包并取出 renderToString。
func NewMathRenderer(bundle []byte) (*MathRenderer, error) {
	vm := goja.New()
	if _, err := vm.RunScript("katex.min.js", string(bundle)); err != nil {
		return nil, fmt.Errorf("加载 KaTeX 失败: %w", err)
	}
	val := vm.Get("katex")
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return nil, errors.New("KaTeX 包未暴露全局 katex 对象")
	}
	obj := val.ToObject(vm)
	fn, ok := goja.AssertFunction(obj.Get("renderToString"))
	if !ok {
		return nil, errors.New("katex.renderToString 不是可调用函数")
	}
	return &MathRenderer{vm: vm, render: fn}, nil
}

// Render 把 TeX 渲染为 KaTeX HTML。goja 运行时非并发安全，这里串行化调用。
func (m *MathRenderer) Render(tex string, display bool) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	timer := time.AfterFunc(mathTimeout, func() {
		m.vm.Interrupt("公式渲染超时")
	})
	defer timer.Stop()
	defer m.vm.ClearInterrupt()

	res, err := m.render(goja.Undefined(), m.vm.ToValue(tex), m.vm.ToValue(map[string]any{
		"displayMode":  display,
		"throwOnError": true,
	}))
	if err != nil {
		return "", err
	}
	return res.String(), nil
}
