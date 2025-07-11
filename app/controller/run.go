package controller

import (
	"EasyTools/app/connect/redis"
	"EasyTools/app/model"
	"context"
	"embed"
	"fmt"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// WailsRun 初始化
func WailsRun(assets embed.FS, port int, icon []byte) {
	// 创建控制器实例
	system := NewSystem()
	site := NewSite()
	server := NewServer()
	tool := NewTool()
	login := NewUser()
	infoSearch := NewInfoSearch()
	infoDeal := NewInfoDeal()
	redisDb := redis.NewRedis()
	checkVersion := CheckVersion()
	Unwxapp := NewUnWxapp()
	jwtcrack := NewJwtCrackController()
	note := NewNote()

	// 启动 Wails 服务
	err := wails.Run(&options.App{
		Title:  "EasyTools：一款实用的渗透测试工具箱 ",
		Width:  1180,
		Height: 720,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Frameless:        false,
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup: func(ctx context.Context) {

			// 设置 context 对象
			system.setCtx(ctx)
			server.setCtx(ctx)
			site.setCtx(ctx)
			tool.setCtx(ctx)
			login.setCtx(ctx)
			infoSearch.setCtx(ctx)
			infoDeal.setCtx(ctx)
			checkVersion.setCtx(ctx)
			Unwxapp.setCtx(ctx)
			jwtcrack.setCtx(ctx)
			note.setCtx(ctx)

			redisDb.SetCtx(ctx)

			server.start(port)
			if runtime.GOOS == "windows" {
				// 优先初始化数据库表结构
				server.schema(&model.User{}, &model.Sites{}, &model.Tools{}, &model.Password_data{}, &model.Google_query{}, &model.Antivirus_list{})
				// 异步执行文件释放（防止阻塞主流程）
				go server.initFile().initMianSha()
			} else {
				server.schema(&model.User{}, &model.Sites{}, &model.Tools{}, &model.Password_data{}, &model.Google_query{}, &model.Antivirus_list{})
				go server.initFile()
			}

		},

		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableFramelessWindowDecorations: true,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
			About: &mac.AboutInfo{
				Title:   "EasyTools",
				Message: "渗透测试工具箱",
				Icon:    icon,
			},
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Linux: &linux.Options{
			ProgramName:         "EasyTools",
			Icon:                icon,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyOnDemand,
			WindowIsTranslucent: false,
		},
		Bind: []interface{}{
			system,
			server,
			site,
			tool,
			login,
			infoSearch,
			infoDeal,
			redisDb,
			checkVersion,
			Unwxapp,
			jwtcrack,
			note,
		},
	})

	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}
