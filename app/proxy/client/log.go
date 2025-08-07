package proxy

import (
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (p *Proxy) Debug(msg ...interface{}) {
	runtime.LogDebug(p.ctx, fmt.Sprint(msg...))
}

func (p *Proxy) Info(msg ...interface{}) {
	runtime.LogInfo(p.ctx, fmt.Sprint(msg...))
}

func (p *Proxy) Warn(msg ...interface{}) {
	runtime.LogWarning(p.ctx, fmt.Sprint(msg...))
}

func (p *Proxy) Error(msg ...interface{}) {
	runtime.LogError(p.ctx, fmt.Sprint(msg...))
}

func (p *Proxy) Fatal(msg ...interface{}) {
	runtime.LogFatal(p.ctx, fmt.Sprint(msg...))
}
