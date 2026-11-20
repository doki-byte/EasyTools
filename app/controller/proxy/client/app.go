package proxy

import (
	"EasyTools/app/controller/proxy/config"
	request2 "EasyTools/app/controller/proxy/request"
	"bufio"
	"context"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"strings"
)

type Proxy struct {
	ctx      context.Context
	stopChan chan struct{}

	rpm    *request2.ProxyManager
	config *config.Config
}

func NewProxy() *Proxy {
	return &Proxy{
		config: config.GetConfig(),
		rpm: request2.NewProxyManager([]request2.ProxyFetcher{
			//&request.Free89,
			//&request.FreeHappy,
			//&request.FreeQiYun,
			&request2.HunterConfig{},
			&request2.QuakeConfig{},
			&request2.FofaConfig{},
		}),
	}
}

// SetCtx 设置上下文对象
func (p *Proxy) SetCtx(ctx context.Context) {
	p.ctx = ctx
}

func (p *Proxy) GetProfile() config.Config {
	return p.config.GetProfile()
}

func (p *Proxy) FetchProxies() Response {
	proxies, err := p.rpm.FetchAll()
	if err != nil {
		return p.errorResponse(err)
	}

	table, err := p.rpm.RenderTable()
	if err != nil {
		return p.errorResponse(err)
	}

	p.config.SetAllProxies(proxies)
	return Response{
		Code:    200,
		Message: "抓取成功",
		Data:    string(table),
	}
}

func (p *Proxy) ChooseFile() config.Config {
	p.config.Code = 200

	// 尝试打开文件对话框，忽略 WebView2 的 Focus 错误（这是 Wails 框架的已知问题）
	filePath, err := runtime.OpenFileDialog(p.ctx, runtime.OpenDialogOptions{
		Title:           "请选择配置文件",
		ShowHiddenFiles: true,
		Filters: []runtime.FileFilter{
			{DisplayName: "配置文件", Pattern: "*.txt"},
		},
	})
	
	// 如果出错但文件路径不为空，仍然继续处理（WebView2 Focus 错误不影响功能）
	if err != nil && filePath == "" {
		p.config.Code = 400
		p.config.Error = "打开文件对话框失败: " + err.Error()
		return p.config.GetProfile()
	}
	
	p.config.FilePath = filePath
	if p.config.FilePath == "" {
		p.config.Code = 400
		p.config.Error = "未选择配置文件"
		return p.config.GetProfile()
	}

	f, errOpen := os.Open(p.config.FilePath)
	if errOpen != nil {
		p.config.Code = 400
		p.config.Error = errOpen.Error()
		return p.config.GetProfile()
	}
	defer f.Close()

	stat, _ := f.Stat()
	if stat.Size() == 0 {
		runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[WARN] 配置文件 %s 是空的", p.config.FilePath))
		p.config.Code = 400
		p.config.Error = "配置文件为空"
		return p.config.GetProfile()
	}

	// 按行读取代理数据
	var lists []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 忽略空行和注释行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lists = append(lists, line)
	}

	if err := scanner.Err(); err != nil {
		p.config.Code = 400
		p.config.Error = err.Error()
		return p.config.GetProfile()
	}

	p.config.SetAllProxies(lists)
	runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[INF] 配置文件 %s 读取成功，共 %d 条数据", p.config.FilePath, len(lists)))

	rsp := p.CheckDatasets()
	if rsp.Code != 200 {
		p.Error("测试有误： %s\n", rsp.Message)
		p.config.Code = 400
		p.config.Error = rsp.Message
		return p.config.GetProfile()
	}

	return p.config.GetProfile()
}

// LoadFileFromPath 从指定路径读取文件并开始处理（不打开文件对话框）
func (p *Proxy) LoadFileFromPath(filePath string) config.Config {
	p.config.Code = 200
	p.config.FilePath = filePath

	if p.config.FilePath == "" {
		p.config.Code = 400
		p.config.Error = "文件路径为空"
		return p.config.GetProfile()
	}

	f, errOpen := os.Open(p.config.FilePath)
	if errOpen != nil {
		p.config.Code = 400
		p.config.Error = errOpen.Error()
		return p.config.GetProfile()
	}
	defer f.Close()

	stat, _ := f.Stat()
	if stat.Size() == 0 {
		runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[WARN] 配置文件 %s 是空的", p.config.FilePath))
		p.config.Code = 400
		p.config.Error = "配置文件为空"
		return p.config.GetProfile()
	}

	// 按行读取代理数据
	var lists []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 忽略空行和注释行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lists = append(lists, line)
	}

	if err := scanner.Err(); err != nil {
		p.config.Code = 400
		p.config.Error = err.Error()
		return p.config.GetProfile()
	}

	p.config.SetAllProxies(lists)
	runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[INF] 配置文件 %s 读取成功，共 %d 条数据", p.config.FilePath, len(lists)))

	rsp := p.CheckDatasets()
	if rsp.Code != 200 {
		p.Error("测试有误： %s\n", rsp.Message)
		p.config.Code = 400
		p.config.Error = rsp.Message
		return p.config.GetProfile()
	}

	return p.config.GetProfile()
}

func (p *Proxy) UseFetchedDatasets() Response {
	p.StopListening()
	p.stopTask()
	runtime.EventsEmit(p.ctx, "log_update", "[INF] 使用抓取的代理")
	return p.CheckDatasets()
}

func (p *Proxy) SaveConfig(data config.Config) string {
	p.config = &data
	p.StopListening()
	p.stopTask()
	err := p.config.SaveConfig()
	if err != nil {
		return "保存失败: " + err.Error()
	}
	return "保存成功"
}
