package proxy

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var listener net.Listener     // 全局监听器
var cancel context.CancelFunc // 用于取消监听的上下文

// 启动监听
func (p *Proxy) startListening() Response {
	// 检查缓存代理数
	if len(p.config.LiveProxyLists) == 0 {
		p.Error("缓存代理数为0，任务取消。")
		runtime.EventsEmit(p.ctx, "log_update", "[ERR] 缓存代理数为0。")
		runtime.EventsEmit(p.ctx, "log_update", "========================= 任务取消 ==========================")
		return p.errorResponse("缓存代理数为0，任务取消。")
	}

	// 检查监听器是否已存在
	if p.config.GetStatus() == 2 {
		// 取消监听
		cancel()
		listener.Close() // 关闭监听器
	}

	// 创建监听器
	var err error
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background()) // 创建带取消功能的上下文

	// 继续按原有逻辑使用个别代理
	socksAddress := strings.Replace(p.config.SocksAddress, "socks5://", "", -1)
	listener, err = net.Listen("tcp", socksAddress)
	if err != nil {
		p.Error("Error: %s\n", err.Error())
		runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[ERR] 监听失败 %s ", err.Error()))
		runtime.EventsEmit(p.ctx, "log_update", "========================= 任务取消 ==========================")
		return p.errorResponse(err.Error())
	}
	defer listener.Close()

	runtime.EventsEmit(p.ctx, "log_update", "======================== 开始监听 =========================")
	runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[INF] 开始监听 %s -- 挂上代理以使用", p.config.SocksAddress))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, p.config.CoroutineCount)

	// 监听连接
	for {
		select {
		case <-ctx.Done(): // 如果上下文被取消，退出监听
			runtime.EventsEmit(p.ctx, "log_update", "[INF] 监听已停止")
			wg.Wait() // 等待所有连接处理完成
			return p.successResponse("监听已成功停止", nil)
		default:
			conn, err := listener.Accept()
			if err != nil {
				runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[ERR] 接受连接失败 %s ", err.Error()))
				continue
			}

			semaphore <- struct{}{}
			wg.Add(1)
			go func(conn net.Conn) {
				defer wg.Done()
				defer func() { <-semaphore }()
				p.handleConnection(conn)
			}(conn)
		}
	}
}

// 停止监听
func (p *Proxy) StopListening() Response {
	if listener == nil {
		p.Error("监听服务未启动")
		runtime.EventsEmit(p.ctx, "log_update", "[ERR] 监听服务未启动")
		return p.errorResponse("监听服务未启动")
	}

	// 取消监听
	cancel()
	listener.Close() // 关闭监听器

	runtime.EventsEmit(p.ctx, "log_update", "[INF] 监听已停止")
	return p.successResponse("监听已成功停止", nil)
}

// 处理连接
func (p *Proxy) handleConnection(conn net.Conn) {
	defer conn.Close()

	if len(p.config.LiveProxyLists) == 0 {
		runtime.EventsEmit(p.ctx, "log_update", "[ERR] 没有可用代理")
		return
	}

	current := p.config.LiveProxyLists[rand.Intn(len(p.config.LiveProxyLists))]
	runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[INF] 当前使用代理 %s ", current))
	runtime.EventsEmit(p.ctx, "status_update", current)

	timeout, err := strconv.Atoi(p.config.Timeout)
	if err != nil {
		p.Debug("Invalid timeout value: %v", err)
	}
	socks, err := net.DialTimeout("tcp", current, time.Duration(timeout)*time.Second)
	if err != nil {
		p.Debug("DialTimeout error: %v", err)
	}

	if err != nil {
		runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[ERR] 连接代理失败 %s ", err.Error()))
		p.handleConnection(conn)
		return
	}
	defer socks.Close()

	var wg sync.WaitGroup
	ioCopy := func(dst io.Writer, src io.Reader) {
		defer wg.Done()
		_, err := io.Copy(dst, src)
		if err != nil {
			runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[ERR] 数据传输失败 %s ", err.Error()))
		}
	}

	wg.Add(2)
	go ioCopy(socks, conn)
	go ioCopy(conn, socks)
	wg.Wait()
}

// 停止任务，释放端口
func (p *Proxy) stopTask() Response {
	if p.config.GetStatus() == 2 {
		// 取消上下文，以停止监听
		cancel()         // 取消监听操作
		listener.Close() // 关闭监听器
		listener = nil   // 清空监听器
		runtime.EventsEmit(p.ctx, "log_update", "[INF] 停止监听服务")
		runtime.EventsEmit(p.ctx, "log_update", "======================== 任务停止 =========================")
	}
	p.config.SetStatus(0) // 更新任务状态为停止
	// 返回成功响应
	return p.successResponse("监听已成功停止", nil)
}
