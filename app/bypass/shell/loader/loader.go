package loader

import (
	"embed"
	"path"
)

//go:embed "module"
var moduleFolder embed.FS

var Modules = make(map[string][]byte, 3) // 3为模板个数

// 通过embed将模块的loader装载进程序，不再依赖本地文件
func init() {
	n, _ := moduleFolder.ReadDir("module")
	for i := 0; i < len(n); i++ {
		nf, _ := n[i].Info()
		templateName := "main.tpl"
		loaderFileContent, err := moduleFolder.ReadFile(path.Join("module", nf.Name(), templateName))
		if err != nil {
			// 兼容旧版 main.go 命名
			loaderFileContent, err = moduleFolder.ReadFile(path.Join("module", nf.Name(), "main.go"))
			if err != nil {
				continue
			}
		}
		Modules[nf.Name()] = loaderFileContent
	}
}
