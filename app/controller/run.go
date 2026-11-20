package controller

import (
	"EasyTools/app/controller/assistive"
	"EasyTools/app/controller/cloudSecurity"
	ftpservice "EasyTools/app/controller/connect/ftp/app/service"
	"EasyTools/app/controller/connect/nosql"
	sshservice "EasyTools/app/controller/connect/ssh/app/service"
	"EasyTools/app/controller/cyberSabre/browserScan"
	"EasyTools/app/controller/cyberSabre/crack"
	"EasyTools/app/controller/cyberSabre/dirsearch"
	"EasyTools/app/controller/cyberSabre/druid"
	"EasyTools/app/controller/cyberSabre/finger"
	"EasyTools/app/controller/cyberSabre/hostscan"
	"EasyTools/app/controller/cyberSabre/webscan"
	hotkey2 "EasyTools/app/controller/hotkey"
	"EasyTools/app/controller/infoCollection/domainCollection"
	"EasyTools/app/controller/infoDeal"
	"EasyTools/app/controller/infoDeal/fileDumper"
	"EasyTools/app/controller/infoDeal/jwt"
	"EasyTools/app/controller/pcap"
	proxySetting "EasyTools/app/controller/proxy"
	proxyFearch "EasyTools/app/controller/proxy/client"
	"EasyTools/app/controller/restmate"
	system2 "EasyTools/app/controller/system"
	"EasyTools/app/controller/unwxapp"
	"EasyTools/app/model"
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
	system := system2.NewSystem()
	site := NewSite()
	server := NewServer()
	tool := NewTool()
	update := system2.NewUpdate()
	softwareCenter := system2.NewSoftwareCenter()
	aiConfigManager := system2.NewAIConfigManager()
	hotkey := hotkey2.NewHotKey()
	systemTp := system2.NewSystemTp()
	proxyManager := proxySetting.NewProxyManager()

	nosqlDb := nosql.NewNoSQL()
	note := NewNote()
	freeProxy := proxyFearch.NewProxy()
	newRestMate := restmate.NewRestMate()

	pcapAnalyzer := pcap.NewPcapAnalyzer()

	sshDesktopBridge := &sshservice.SshDesktopBridge{}
	ftpDesktopBridge := &ftpservice.FtpDesktopBridge{}

	assistiveApp := assistive.NewAssistive()
	OssBucker := cloudSecurity.NewOssBucker()
	objectStorage := cloudSecurity.NewObjectStorage()
	cloudServer := cloudSecurity.NewCloudServer()
	QueryIp := infoDeal.NewQueryIp()
	browserScanController := browserScan.NewBrowserScan()
	Unwxapp := unwxapp.NewUnWxapp()
	jwtcrack := jwt.NewJwtCrackController()
	gitdumper := fileDumper.NewGitDumperApp()
	svndumper := fileDumper.NewSvnDumperApp()
	dsStoredumper := fileDumper.NewDSStoreDumperApp()
	indexDumper := fileDumper.NewIndexDumperApp()
	hgDumper := fileDumper.NewHgDumperApp()
	dirsearchController := dirsearch.NewDirsearchController()
	fingerprintController := finger.NewFingerprintController()
	hostscanController := hostscan.NewHostscanController()
	pocController := webscan.NewPocController()
	pocConfigController := webscan.NewPocConfigController()
	crackController := crack.NewCrackController()
	domainCollectionController := domainCollection.NewDomainCollectionController()
	druidController := druid.NewDruidController()

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
			server.start(port)
			// 在服务启动后更新端口配置（因为 start 方法会分配实际端口）
			system.SetPortConfig(server.PortConfig)

			site.SetCtx(ctx)
			tool.SetCtx(ctx)
			update.Startup(ctx)
			softwareCenter.Startup(ctx)
			aiConfigManager.Startup(ctx)
			note.SetCtx(ctx)
			hotkey.SetContext(ctx)
			systemTp.Startup(ctx, systemTrayIcon)
			nosqlDb.SetCtx(ctx)
			freeProxy.SetCtx(ctx)
			newRestMate.Startup(ctx)
			proxyManager.SetCtx(ctx)

			pcapAnalyzer.Startup(ctx)

			assistiveApp.SetCtx(ctx)
			OssBucker.SetCtx(ctx)
			objectStorage.SetCtx(ctx)
			cloudServer.SetCtx(ctx)
			QueryIp.SetCtx(ctx)
			browserScanController.Startup(ctx)
			Unwxapp.SetCtx(ctx)
			jwtcrack.SetCtx(ctx)
			gitdumper.Startup(ctx)
			svndumper.Startup(ctx)
			dsStoredumper.Startup(ctx)
			indexDumper.Startup(ctx)
			hgDumper.Startup(ctx)
			dirsearchController.Startup(ctx)
			fingerprintController.Startup(ctx)
			hostscanController.Startup(ctx)
			pocController.Startup(ctx)
			pocConfigController.Startup(ctx)
			crackController.Startup(ctx)
			domainCollectionController.Startup(ctx)
			druidController.Startup(ctx)
			// 设置端口配置到 DomainCollection（在 server.start 之后）
			if server.PortConfig != nil {
				domainCollectionController.DomainCollection.SetPortConfig(server.PortConfig)
			}

			// 设置全局代理管理器
			proxyManager.Initialize()
			proxySetting.SetGlobalProxyManager(proxyManager)
			system2.SetGlobalContext(ctx)

			if runtime2.GOOS == "windows" {
				// 优先初始化数据库表结构
				server.Schema(&model.ShellConfig{}, &model.Sites{}, &model.Tools{}, &model.Password_data{}, &model.Google_query{}, &model.Antivirus_list{}, &model.WechatConfig{}, &model.MiniAppInfo{}, &model.VersionTask{}, &model.ProxyConfig{}, &model.VulnerabilityInfo{}, &model.ProtocolConnection{}, &model.ObjectStorageCredential{}, &model.CloudServerCredential{})
			} else {
				server.Schema(&model.Sites{}, &model.Tools{}, &model.Password_data{}, &model.Google_query{}, &model.Antivirus_list{}, &model.WechatConfig{}, &model.MiniAppInfo{}, &model.VersionTask{}, &model.ProxyConfig{}, &model.VulnerabilityInfo{}, &model.ProtocolConnection{}, &model.ObjectStorageCredential{}, &model.CloudServerCredential{})
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
						newTitle := fmt.Sprintf("EasyTools：一款实用的渗透测试工具箱 v2.3.1            CPU: %.2f%% | 自身: %.2f MB | 内存: %.2f%%",
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
		DragAndDrop: DragAndDropOptions(),
		OnDomReady: func(ctx context.Context) {
			runtime.OnFileDrop(ctx, func(x, y int, paths []string) {
				if len(paths) > 0 {
					// 直接广播给前端，携带绝对路径
					runtime.EventsEmit(ctx, "global-file-drop", paths)
				}
			})
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
			update,
			note,
			nosqlDb,
			freeProxy,
			hotkey,
			systemTp,
			newRestMate,
			proxyManager,
			softwareCenter,
			aiConfigManager,

			Unwxapp,
			jwtcrack,
			pcapAnalyzer,

			sshDesktopBridge,
			ftpDesktopBridge,

			assistiveApp,
			OssBucker,
			objectStorage,
			cloudServer,
			QueryIp,
			browserScanController,
			gitdumper,
			svndumper,
			dsStoredumper,
			indexDumper,
			hgDumper,
			dirsearchController,
			fingerprintController,
			hostscanController,
			pocController,
			pocConfigController,
			crackController,
			domainCollectionController,
			druidController,
		},
	})

	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}

func DragAndDropOptions() *options.DragAndDrop {
	if runtime2.GOOS == "windows" {
		return &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		}
	} else {
		return &options.DragAndDrop{
			EnableFileDrop: true,
		}
	}
}
