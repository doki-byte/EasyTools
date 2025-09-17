package controller

import (
	"EasyTools/app/connect/redis"
	hotkey2 "EasyTools/app/hotkey"
	"EasyTools/app/model"
	"EasyTools/app/proxy/client"
	"EasyTools/app/restmate"
	"context"
	"embed"
	"fmt"
	runtime2 "runtime"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// WailsRun 初始化
func WailsRun(assets embed.FS, port int, appIcon, systemTrayIcon []byte) {
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
	freeProxy := proxy.NewProxy()
	hotkey := hotkey2.NewHotKey()
	systemTp := NewSystemTp()
	restmate := restmate.NewRestMate()

	// 启动 Wails 服务
	err := wails.Run(&options.App{
		Title:  "EasyTools：一款实用的渗透测试工具箱 ",
		Width:  1200,
		Height: 800,
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
			systemTp.Startup(ctx, systemTrayIcon)
			redisDb.SetCtx(ctx)
			freeProxy.SetCtx(ctx)

			hotkey.SetContext(ctx)
			restmate.Startup(ctx)

			server.start(port)
			if runtime2.GOOS == "windows" {
				// 优先初始化数据库表结构
				server.schema(&model.User{}, &model.Sites{}, &model.Tools{}, &model.Password_data{}, &model.Google_query{}, &model.Antivirus_list{})
				// 异步执行文件释放（防止阻塞主流程）
				go server.initFile().initMianSha()
			} else {
				server.schema(&model.User{}, &model.Sites{}, &model.Tools{}, &model.Password_data{}, &model.Google_query{}, &model.Antivirus_list{})
				go server.initFile()
			}
			// 启动监控协程
			go func() {
				ticker := time.NewTicker(1 * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						// 获取系统监控数据
						cpuUsage := system.GetCPUUsage()      // 实现获取CPU使用率的方法
						memSelf := system.GetMemUsageSelf()   // 实现获取自身内存使用的方法
						memTotal := system.GetMemUsageTotal() // 实现获取总内存使用的方法

						// 格式化标题
						newTitle := fmt.Sprintf("EasyTools：一款实用的渗透测试工具箱  v1.8.7            CPU: %.2f%% | 自身: %.2f MB | 内存: %.2f%%",
							cpuUsage, memSelf, memTotal)

						// 更新窗口标题
						runtime.WindowSetTitle(ctx, newTitle)
					}
				}
			}()

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
				Icon:    appIcon,
			},
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Linux: &linux.Options{
			ProgramName:         "EasyTools",
			Icon:                appIcon,
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
			freeProxy,
			hotkey,
			systemTp,
			restmate,
		},
	})

	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}
