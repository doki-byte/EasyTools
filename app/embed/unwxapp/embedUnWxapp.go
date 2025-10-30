package unwxapp

import (
	"embed"
)

// 使用最全面的嵌入模式
//
//go:embed all:node_modules
//go:embed index.js parser-928e23b1.js traverse-252284fd.js
var Assets embed.FS
