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
	//使用 10086 作为ginServer的端口
	controller.WailsRun(assets, 10086, icon)
}
