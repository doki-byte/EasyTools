package loader

import (
	"embed"
	"path"
)

//go:embed "module"
var moduleFolder embed.FS

// Modules = {"模板名":"模板内容"}
var Modules = make(map[string][]byte, 1) // 3为模板个数

// 通过embed将模块的loader装载进程序，不再依赖本地文件
func init() {
	n, _ := moduleFolder.ReadDir("module")
	for i := 0; i < len(n); i++ {
		nf, _ := n[i].Info()
		loaderFileContent, _ := moduleFolder.ReadFile(path.Join("module", nf.Name(), "main.go"))
		// loaderFileContent 为模板源码
		Modules[nf.Name()] = loaderFileContent
	}
}
