package cyberchef

import (
	"embed"
)

// 使用最全面的嵌入模式
//
//go:embed all:assets
//go:embed all:images
//go:embed all:modules
//go:embed index.html
var Assets embed.FS
