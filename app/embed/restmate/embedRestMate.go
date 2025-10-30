package restmate

import (
	"embed"
)

// 使用最全面的嵌入模式

//go:embed restmate_db.json
//go:embed restmate_env.json
//go:embed restmate_jar.json
var Assets embed.FS
