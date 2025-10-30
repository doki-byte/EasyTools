package dict

import (
	"embed"
)

// 使用最全面的嵌入模式
//
//go:embed *
var Assets embed.FS
