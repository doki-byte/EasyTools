package main

import (
	"EasyTools/app/controller"
	"embed"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	//使用 52867 作为ginServer的端口
	controller.WailsRun(assets, 52867, icon)
}
