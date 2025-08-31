package controller

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

	// 只有一个退出菜单，点击直接退出（不再确认）
	a.exit = systray.AddMenuItem("退出", "退出程序")
	if a.exit != nil {
		a.exit.Click(func() {
			// 如果有窗口上下文，先隐藏窗口
			if a.ctx != nil {
				wruntime.WindowHide(a.ctx)
			}

			// 通知切换协程退出（非阻塞）
			select {
			case <-a.done:
				// already closed
			default:
				close(a.done)
			}

			// 清理托盘并直接退出进程
			systray.Quit()
			os.Exit(0)
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
		"最好的防御，\n是攻击者的视角。",
		"我们不是在寻找漏洞，\n我们是在寻找信任的边界。",
		"代码的沉默，\n比任何告警都更响亮。",
		"在二进制的世界里，\n好奇心是唯一的通行证。",
		"我们拆解，\n是为了更好地构建。",
		"没有无法攻破的系统，\n只有尚未被发现的路径。",
		"一行代码，\n可以颠覆一个帝国。",
		"耐心不是等待，\n而是在寂静中持续地行动。",
		"工具只是延伸，\n真正的武器是你的大脑。",
		"异常之处，\n即是入口。",
		"成功，\n往往藏在被忽略的细节里。",
		"每一次失败，\n都是通往“未授权”的路标。",
		"真正的专家，\n是那些知道自己无知的人。",
		"在漏洞的海洋里，\n唯一不变的就是变化本身。",
		"舒适区是技能生锈的温床。",
		"今天发现的漏洞，\n是明天更坚固的基石。",
		"我们不是黑客，\n我们是数字世界的“白帽骑士”。",
		"你的每一次报告，\n都在让世界变得更安全一点。",
		"比发现漏洞更重要的，\n是守护信任。",
		"站在黑暗中，\n是为了点亮一盏灯。",
		"愿我们归来仍是少年",
		"只会幻想而不行动的人，\n永远也体会不到收获果实时的喜悦。",
		"不去期望，\n失去了不会伤心，\n得到了便是惊喜。",
		"生活不是林黛玉，\n不会因为忧伤而风情万种。",
		"让生活的句号圈住的人，\n是无法前时半步的。",
		"人生，\n就要闯出条路来！",
		"金钱损失了还能挽回，\n一旦失去信誉就很难挽回。",
		"同在一个环境中生活，\n强者与弱者的分界就在于谁能改变它。",
		"没有热忱，\n世间便无进步。",
		"谁不是一边受伤，\n一边学会坚强。",
		"每天都冒出很多念头，\n那些不死的才叫做梦想。",
		"有目标的人生才有方向有规划的人生，\n更精彩!",
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

	// 周期性更新 tooltip，模拟“每次悬停都可能不同”的效果
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

func (a *SystemTp) Startup(ctx context.Context, systemTrayIcon []byte) {
	a.ctx = ctx

	// 监听应用关闭事件
	wruntime.EventsOn(ctx, "quit", func(optionalData ...interface{}) {
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
