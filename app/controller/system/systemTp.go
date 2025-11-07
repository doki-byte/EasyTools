package system

import (
	"context"
	"github.com/energye/systray"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"math/rand"
	"os"
	"time"
)

type SystemTp struct {
	Base
	exit *systray.MenuItem
	ctx  context.Context

	// 用于通知切换协程退出
	done chan struct{}
}

func NewSystemTp() *SystemTp {
	return &SystemTp{
		done: make(chan struct{}),
	}
}

func (a *SystemTp) systemTray(systemTrayIcon []byte) {
	// 设置托盘图标
	if len(systemTrayIcon) > 0 {
		systray.SetIcon(systemTrayIcon)
	}

	// 添加退出菜单，点击后发送事件到前端显示确认对话框
	a.exit = systray.AddMenuItem("退出", "退出程序")
	if a.exit != nil {
		a.exit.Click(func() {
			// 发送事件到前端，显示毛玻璃效果的退出确认对话框
			if a.ctx != nil {
				wruntime.EventsEmit(a.ctx, "app-exit")
			} else {
				// 如果没有上下文，直接退出
				a.quitApplication()
			}
		})
	}

	// 左键单击：直接显示主窗口
	systray.SetOnClick(func(menu systray.IMenu) {
		if a.ctx != nil {
			wruntime.WindowShow(a.ctx)
		}
	})

	// 右键单击：显示菜单（只包含"退出"）
	systray.SetOnRClick(func(menu systray.IMenu) {
		menu.ShowMenu()
	})

	// 语句集合（保持与你原来句子相同或略作格式化）
	quotes := []string{
		"耐心不是等待，\n而是在寂静中持续地行动。",
		"成功，\n往往藏在被忽略的细节里。",
		"愿我们归来仍是少年",
		"生活不是林黛玉，\n不会因为忧伤而风情万种。",
		"让生活的句号圈住的人，\n是无法前时半步的。",
		"人生，\n就要闯出条路来！",
		"天再高又怎样，\n踮起脚尖就更接近阳光。",
		"业精于勤，荒于嬉；\n行成于思，毁于随。",
		"我们能够失望，\n但不能盲目。",
		"没有伞的孩子必须努力奔跑！",
		"岂能尽人如意，\n但求无愧于心！",
		"只会幻想而不行动的人，\n永远也体会不到收获果实时的喜悦。",
		"希望您喜欢EasyTools \n(✪ω✪)",
	}

	// 初始设置一个提示，避免刚启动时为空
	rand.Seed(time.Now().UnixNano())
	initial := quotes[rand.Intn(len(quotes))]
	systray.SetTooltip("EasyTools ヾ(๑╹◡╹)ﾉ \n" + initial)

	// 周期性更新 tooltip，模拟"每次悬停都可能不同"的效果
	ticker := time.NewTicker(5 * time.Second) // 你可以改成 1s / 5s 等
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				q := quotes[rand.Intn(len(quotes))]
				systray.SetTooltip("EasyTools ヾ(๑╹◡╹)ﾉ \n" + q)
			case <-a.done:
				return
			}
		}
	}()
}

func (a *SystemTp) quitApplication() {
	// 通知切换协程退出（非阻塞）
	select {
	case <-a.done:
		// already closed
	default:
		close(a.done)
	}

	// 跨平台退出处理：清理托盘并退出
	systray.Quit()
	os.Exit(0)
}

// ConfirmExit 前端调用的确认退出方法
func (a *SystemTp) ConfirmExit() {
	a.quitApplication()
}

// CancelExit 前端调用的取消退出方法（可选，用于清理状态）
func (a *SystemTp) CancelExit() {
	// 这里可以添加取消退出时的清理逻辑
	// 目前不需要特殊处理
}

func (a *SystemTp) Startup(ctx context.Context, systemTrayIcon []byte) {
	a.ctx = ctx

	// 监听应用关闭事件
	wruntime.EventsOn(ctx, "quit", func(optionalData ...interface{}) {
		a.quitApplication()
	})

	// 监听前端确认退出事件
	wruntime.EventsOn(ctx, "confirm-exit", func(optionalData ...interface{}) {
		a.quitApplication()
	})

	go func() {
		systray.Run(func() {
			a.systemTray(systemTrayIcon)
		}, func() {
			// 回调里确保完全退出
			os.Exit(0)
		})
	}()
}
