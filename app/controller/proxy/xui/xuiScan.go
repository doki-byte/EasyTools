package xui

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

type XUIScanner struct {
	ctx          context.Context
	printLock    sync.Mutex
	excelLock    sync.Mutex
	switchHTTP   bool
	speedTest    bool
	httpUser     string
	httpPass     string
	excelFile    string
	client       *http.Client
	eventEmitter func(event string, data ...interface{})
	xuiMode      string // 0:关闭, 1:开启不添加HTTP代理, 2:开启添加HTTP代理
}

type ProxyInfo struct {
	URL        string
	Speed      string
	HTTPAddr   string
	HTTPUser   string
	HTTPPass   string
	HTTPPort   string
	HTTPSpeed  string
	VlessNodes string
	VmessNodes string
}

func NewXUIScanner(ctx context.Context, xuiMode string, eventEmitter func(event string, data ...interface{})) *XUIScanner {
	rand.Seed(time.Now().UnixNano())

	return &XUIScanner{
		ctx:          ctx,
		switchHTTP:   xuiMode == "2", // 只有模式2才开启HTTP代理
		speedTest:    true,
		httpUser:     "mKRZVpd7dy", // 默认用户名
		httpPass:     "1e0kimIcvT", // 默认密码
		excelFile:    "3x-ui可用代理集合.xlsx",
		eventEmitter: eventEmitter,
		xuiMode:      xuiMode,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 5 * time.Second,
		},
	}
}

// 设置HTTP认证信息（只设置用户名和密码，端口使用随机）
func (x *XUIScanner) SetHTTPAuth(user, pass string) {
	x.httpUser = user
	x.httpPass = pass
}

func (x *XUIScanner) ScanURLs(urls []string, threads int) {
	// 如果Xui模式为0，则不进行扫描
	if x.xuiMode == "0" {
		x.eventEmitter("log_update", "[INF] Xui模式已关闭，跳过扫描")
		return
	}

	x.eventEmitter("log_update", fmt.Sprintf("[INF] 开始x-ui面板扫描，模式：%s", x.xuiMode))
	x.initExcel()

	var wg sync.WaitGroup
	ch := make(chan string, threads*2)

	// 发射进度事件
	total := len(urls)
	processed := 0
	var progressMutex sync.Mutex

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range ch {
				x.processURL(u)

				// 更新进度
				progressMutex.Lock()
				processed++
				progress := float64(processed) / float64(total)
				x.eventEmitter("task_progress", fmt.Sprintf("%.2f", progress))
				progressMutex.Unlock()
			}
		}()
	}

	for _, u := range urls {
		ch <- u
	}

	close(ch)
	wg.Wait()

	x.eventEmitter("log_update", "[INF] x-ui面板扫描完成")
}

func (x *XUIScanner) initExcel() {
	f := excelize.NewFile()
	defer f.Close()

	headers := []string{"URL地址", "打开网址速度", "开启添加HTTP", "HTTP地址", "HTTP用户名",
		"HTTP密码", "HTTP端口", "连接baidu速度", "vless节点", "vmess节点"}

	styleID, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 12},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"20B2AA"}, Pattern: 1},
	})

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Sheet1", cell, h)
		f.SetCellStyle("Sheet1", cell, cell, styleID)
	}

	f.SetColWidth("Sheet1", "A", "A", 40)
	f.SetColWidth("Sheet1", "I", "J", 60)

	baseDir := x.getBaseDir()
	filePath := filepath.Join(baseDir, x.excelFile)
	if err := f.SaveAs(filePath); err != nil {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 创建Excel文件失败: %v", err))
	}
}

func (x *XUIScanner) processURL(rawURL string) {
	u := x.normalizeURL(rawURL)
	x.eventEmitter("log_update", fmt.Sprintf("[INF] 扫描: %s", u))

	start := time.Now()
	resp, err := x.client.PostForm(u+"/login", url.Values{
		"username": {"admin"},
		"password": {"admin"},
	})
	if err != nil {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 连接失败: %s - %v", u, err))
		return
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 解析响应失败: %s - %v", u, err))
		return
	}

	if result.Success {
		speed := fmt.Sprintf("%.2fs", time.Since(start).Seconds())
		x.eventEmitter("log_update", fmt.Sprintf("[SUC] 发现可用x-ui面板: %s (响应时间: %s)", u, speed))

		cookies := x.getCookies(resp.Cookies())
		proxyList := x.getProxyList(u, cookies)
		vless, vmess := x.getProxyNodes(u, proxyList)

		info := ProxyInfo{
			URL:        u,
			Speed:      speed,
			VlessNodes: strings.Join(vless, "\n"),
			VmessNodes: strings.Join(vmess, "\n"),
		}

		// 只有在模式2下才添加HTTP代理
		if x.xuiMode == "2" && x.switchHTTP {
			// 总是使用随机端口，避免端口冲突
			port := x.randomPort()

			if x.addHTTP(u, cookies, port) {
				info.HTTPAddr = x.getHost(u)
				info.HTTPUser = x.httpUser
				info.HTTPPass = x.httpPass
				info.HTTPPort = port

				if x.speedTest {
					x.eventEmitter("log_update", fmt.Sprintf("[INF] 测试HTTP代理速度: %s:%s", info.HTTPAddr, port))
					info.HTTPSpeed = x.testHTTPSpeed(info.HTTPAddr, port)
					x.eventEmitter("log_update", fmt.Sprintf("[INF] HTTP代理速度: %s", info.HTTPSpeed))
				}
			}
		}

		x.saveToExcel(info)
	} else {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 认证失败: %s", u))
	}
}

func (x *XUIScanner) getCookies(cookies []*http.Cookie) string {
	for _, c := range cookies {
		if c.Name == "3x-ui" {
			return c.Value
		}
	}
	return ""
}

func (x *XUIScanner) getProxyList(url, session string) map[string]interface{} {
	req, _ := http.NewRequest("GET", url+"/panel/api/inbounds/list", nil)
	req.AddCookie(&http.Cookie{Name: "3x-ui", Value: session})
	resp, err := x.client.Do(req)
	if err != nil {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 获取代理列表失败: %v", err))
		return nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func (x *XUIScanner) getProxyNodes(url string, data map[string]interface{}) ([]string, []string) {
	host := x.getHost(url)
	objs, ok := data["obj"].([]interface{})
	if !ok {
		return nil, nil
	}

	var vless, vmess []string
	for _, obj := range objs {
		item := obj.(map[string]interface{})
		protocol := item["protocol"].(string)

		switch protocol {
		case "vless":
			if node := x.buildVless(item, host); node != "" {
				vless = append(vless, node)
				x.eventEmitter("log_update", fmt.Sprintf("[INF] 生成vless节点: %s", node))
			}
		case "vmess":
			if node := x.buildVmess(item, host); node != "" {
				vmess = append(vmess, node)
				x.eventEmitter("log_update", fmt.Sprintf("[INF] 生成vmess节点: %s", node))
			}
		}
	}
	return vless, vmess
}

func (x *XUIScanner) buildVless(item map[string]interface{}, host string) string {
	settingsStr, ok := item["settings"].(string)
	if !ok {
		return ""
	}

	settings := x.parseSettings(settingsStr)
	stream := x.parseSettings(item["streamSettings"].(string))

	clients, ok := settings["clients"].([]interface{})
	if !ok || len(clients) == 0 {
		return ""
	}

	client := clients[0].(map[string]interface{})
	id, ok := client["id"].(string)
	if !ok || id == "" {
		return ""
	}

	network, _ := stream["network"].(string)
	security, _ := stream["security"].(string)
	port, _ := item["port"].(float64)
	remark, _ := item["remark"].(string)

	return fmt.Sprintf("vless://%s@%s:%v?type=%s&security=%s#%s",
		id,
		host,
		port,
		network,
		security,
		remark,
	)
}

func (x *XUIScanner) buildVmess(item map[string]interface{}, host string) string {
	settingsStr, ok := item["settings"].(string)
	if !ok {
		return ""
	}

	settings := x.parseSettings(settingsStr)
	stream := x.parseSettings(item["streamSettings"].(string))

	clients, ok := settings["clients"].([]interface{})
	if !ok || len(clients) == 0 {
		return ""
	}

	client := clients[0].(map[string]interface{})
	id, ok := client["id"].(string)
	if !ok || id == "" {
		return ""
	}

	port, _ := item["port"].(float64)
	remark, _ := item["remark"].(string)

	vmess := map[string]interface{}{
		"v":    "2",
		"ps":   remark,
		"add":  host,
		"port": port,
		"id":   id,
		"aid":  0,
		"net":  stream["network"],
		"type": "none",
		"tls":  stream["security"],
	}

	jsonData, _ := json.Marshal(vmess)
	return "vmess://" + base64.StdEncoding.EncodeToString(jsonData)
}

func (x *XUIScanner) parseSettings(s string) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal([]byte(s), &result)
	return result
}

func (x *XUIScanner) addHTTP(baseurl, session, port string) bool {
	sniffing := `{
  "enabled": false,
  "destOverride": [
    "http",
    "tls",
    "quic",
    "fakedns"
  ],
  "metadataOnly": false,
  "routeOnly": false
}`

	settings := fmt.Sprintf(`{
  "accounts": [
    {
      "user": "%s",
      "pass": "%s"
    }
  ],
  "allowTransparent": false
}`, x.httpUser, x.httpPass)

	data := url.Values{
		"up":                   {"0"},
		"down":                 {"0"},
		"total":                {"0"},
		"remark":               {"http"},
		"enable":               {"true"},
		"expiryTime":           {"0"},
		"trafficReset":         {"never"},
		"lastTrafficResetTime": {"0"},
		"listen":               {""},
		"port":                 {port},
		"protocol":             {"http"},
		"settings":             {settings},
		"streamSettings":       {""},
		"sniffing":             {sniffing},
	}

	req, _ := http.NewRequest("POST", baseurl+"/panel/api/inbounds/add", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "3x-ui", Value: session})

	resp, err := x.client.Do(req)
	if err != nil {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 添加HTTP代理失败: %v", err))
		return false
	}
	defer resp.Body.Close()

	var result struct{ Success bool }
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 解析添加HTTP响应失败: %v", err))
		return false
	}

	if result.Success {
		x.eventEmitter("log_update", fmt.Sprintf("[SUC] 成功添加HTTP代理，端口: %s", port))
	} else {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 添加HTTP代理失败，端口: %s", port))
	}

	return result.Success
}

func (x *XUIScanner) testHTTPSpeed(host, port string) string {
	// 使用配置的用户名和密码构建代理URL
	proxyURL := fmt.Sprintf("http://%s:%s@%s:%s", x.httpUser, x.httpPass, host, port)
	x.eventEmitter("log_update", fmt.Sprintf("[INF] 测试代理: %s", proxyURL))

	proxy, err := url.Parse(proxyURL)
	if err != nil {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 解析代理URL失败: %v", err))
		return "代理URL解析失败"
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(proxy),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get("https://www.baidu.com")
	if err != nil {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 代理测试失败: %v", err))
		return "连接失败"
	}
	defer resp.Body.Close()

	speed := time.Since(start).Seconds()
	x.eventEmitter("log_update", fmt.Sprintf("[INF] 代理测试成功，延迟: %.2fs", speed))

	return fmt.Sprintf("%.2fs", speed)
}

func (x *XUIScanner) saveToExcel(info ProxyInfo) {
	x.excelLock.Lock()
	defer x.excelLock.Unlock()

	baseDir := x.getBaseDir()
	filePath := filepath.Join(baseDir, x.excelFile)

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 打开Excel文件失败: %v", err))
		return
	}
	defer f.Close()

	rows, _ := f.GetRows("Sheet1")
	row := len(rows) + 1

	// 根据模式决定是否显示HTTP相关信息
	var httpEnabled string
	if x.xuiMode == "2" {
		httpEnabled = "是"
	} else {
		httpEnabled = "否"
	}

	values := []interface{}{
		info.URL, info.Speed, httpEnabled, info.HTTPAddr,
		info.HTTPUser, info.HTTPPass, info.HTTPPort,
		info.HTTPSpeed, info.VlessNodes, info.VmessNodes,
	}

	for i, v := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		f.SetCellValue("Sheet1", cell, v)
	}

	if err := f.Save(); err != nil {
		x.eventEmitter("log_update", fmt.Sprintf("[ERR] 保存Excel失败: %v", err))
	} else {
		x.eventEmitter("log_update", fmt.Sprintf("[SUC] 已保存到Excel: %s", info.URL))
	}
}

func (x *XUIScanner) normalizeURL(u string) string {
	if !strings.HasPrefix(u, "http") {
		u = "http://" + u
	}
	return strings.TrimRight(u, "/")
}

func (x *XUIScanner) randomPort() string {
	// 生成30000-65535范围内的随机端口
	return strconv.Itoa(rand.Intn(35535) + 30000)
}

func (x *XUIScanner) getHost(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	return parsed.Hostname()
}

func (x *XUIScanner) getBaseDir() string {
	return "."
}
