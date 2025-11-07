package controller

import (
	"EasyTools/app/connect/redis"
	system2 "EasyTools/app/controller/system"
	"EasyTools/app/controller/unwxapp"
	hotkey2 "EasyTools/app/hotkey"
	"EasyTools/app/model"
	proxySetting "EasyTools/app/proxy"
	proxyFearch "EasyTools/app/proxy/client"
	"EasyTools/app/restmate"
	"context"
	"embed"
	"fmt"
	"github.com/wailsapp/wails/v2"
	runtime2 "runtime"
	"time"

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
	system := system2.NewSystem()
	site := NewSite()
	server := NewServer()
	tool := NewTool()
	login := NewUser()
	assistive := NewAssistive()
	infoDeal := NewInfoDeal()
	redisDb := redis.NewRedis()
	update := system2.NewUpdate()
	Unwxapp := unwxapp.NewUnWxapp()
	jwtcrack := NewJwtCrackController()
	note := NewNote()
	freeProxy := proxyFearch.NewProxy()
	hotkey := hotkey2.NewHotKey()
	systemTp := system2.NewSystemTp()
	newRestMate := restmate.NewRestMate()
	proxyManager := proxySetting.NewProxyManager()

	// 启动 Wails 服务
	err := wails.Run(&options.App{
		Title:  "EasyTools：一款实用的渗透测试工具箱 ",
		Width:  1220,
		Height: 850,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Frameless:        true,
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			runtime.EventsEmit(ctx, "app-exit")
			return true
		},
		OnStartup: func(ctx context.Context) {
			// 设置 context 对象
			system.SetCtx(ctx)
			server.SetCtx(ctx)
			site.SetCtx(ctx)
			tool.SetCtx(ctx)
			login.SetCtx(ctx)
			assistive.SetCtx(ctx)
			infoDeal.SetCtx(ctx)
			update.SetCtx(ctx)
			Unwxapp.SetCtx(ctx)
			jwtcrack.SetCtx(ctx)
			note.SetCtx(ctx)
			systemTp.Startup(ctx, systemTrayIcon)
			redisDb.SetCtx(ctx)
			freeProxy.SetCtx(ctx)

			hotkey.SetContext(ctx)
			newRestMate.Startup(ctx)
			proxyManager.SetCtx(ctx)
			server.start(port)

			// 设置全局代理管理器
			proxySetting.SetGlobalProxyManager(proxyManager)
			system2.SetGlobalContext(ctx)

			if runtime2.GOOS == "windows" {
				// 优先初始化数据库表结构
				server.Schema(&model.User{}, &model.Sites{}, &model.Tools{}, &model.Password_data{}, &model.Google_query{}, &model.Antivirus_list{}, &model.WechatConfig{}, &model.MiniAppInfo{}, &model.VersionTask{}, &model.ProxyConfig{})
			} else {
				server.Schema(&model.User{}, &model.Sites{}, &model.Tools{}, &model.Password_data{}, &model.Google_query{}, &model.Antivirus_list{}, &model.WechatConfig{}, &model.MiniAppInfo{}, &model.VersionTask{}, &model.ProxyConfig{})
			}
			// 异步释放文件
			go server.InitMianSha()

			// 恢复：启动监控协程 - 显示CPU和内存使用情况
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
						newTitle := fmt.Sprintf("EasyTools：一款实用的渗透测试工具箱  v1.9.4            CPU: %.2f%% | 自身: %.2f MB | 内存: %.2f%%",
							cpuUsage, memSelf, memTotal)

						// 更新窗口标题
						runtime.WindowSetTitle(ctx, newTitle)

						// 同时发送事件到前端，确保标题栏也能收到更新
						runtime.EventsEmit(ctx, "title-updated", newTitle)
					}
				}
			}()
			// 修复mac运行但是不自动显示窗口的bug
			runtime.WindowShow(ctx)
		},

		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableFramelessWindowDecorations: true,
			Theme:                             windows.Light,
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
			assistive,
			infoDeal,
			redisDb,
			update,
			Unwxapp,
			jwtcrack,
			note,
			freeProxy,
			hotkey,
			systemTp,
			newRestMate,
			proxyManager,
		},
	})

	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}
