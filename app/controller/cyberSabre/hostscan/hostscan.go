package hostscan

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"EasyTools/app/controller/proxy"

	netproxy "golang.org/x/net/proxy"
)

// AliveCheckConfig 存活检测配置
type AliveCheckConfig struct {
	Type    string `json:"type"`    // icmp, tcp_port, http, custom
	Ports   []int  `json:"ports"`   // 端口列表（用于tcp_port类型）
	URL     string `json:"url"`     // URL（用于http类型）
	Timeout int    `json:"timeout"` // 超时时间（秒）
	Enabled bool   `json:"enabled"` // 是否启用
}

// HostscanConfig 主机扫描配置
type HostscanConfig struct {
	Targets               []string          `json:"targets"`              // IP 或域名，支持多行
	Ports                 string            `json:"ports"`                // 端口表达式，例如 "80,443,8080-8090"
	PortPreset            string            `json:"portPreset"`           // 端口预设：top100, top1000, noPrinter, custom
	CheckAlive            bool              `json:"checkAlive"`           // 是否检测IP存活
	AliveCheck            *AliveCheckConfig `json:"aliveCheck,omitempty"` // 存活检测配置
	RandomizePorts        bool              `json:"randomizePorts"`       // 是否乱序扫描端口
	Threads               int               `json:"threads"`
	Timeout               int               `json:"timeout"`               // 秒
	ExcludePorts          []int             `json:"excludePorts"`          // 排除的端口列表
	EnablePortFingerprint bool              `json:"enablePortFingerprint"` // 是否启用端口指纹识别
	// EmitClosedPorts 为 true 时将关闭/过滤端口也通过 results 推送到前端（默认 false，减轻 Wails/前端 IPC 压力）
	EmitClosedPorts          bool `json:"emitClosedPorts"`
	DisableFirewallDetection bool           `json:"disableFirewallDetection"` // 是否关闭防火墙全端口检测
	FirewallDetectSampleSize int            `json:"firewallDetectSampleSize"` // 防火墙检测采样端口数
	FirewallDetectThreshold float64         `json:"firewallDetectThreshold"`  // 防火墙检测触发阈值(0~1)
	TabId                 string            `json:"tabId,omitempty"`       // 前端标签页ID，用于事件路由
}

// HostscanResult 单个主机扫描结果
type HostscanResult struct {
	Target         string   `json:"target"`
	Port           int      `json:"port"`
	Open           bool     `json:"open"`
	Service        string   `json:"service,omitempty"`  // 推测/识别到的服务名或应用
	Protocol       string   `json:"protocol,omitempty"` // 协议（目前为 tcp/http/https 等）
	Banner         string   `json:"banner,omitempty"`   // 协议 banner 原始数据（截断）
	Title          string   `json:"title,omitempty"`    // HTTP 页面标题（如可获取）
	Location       string   `json:"location,omitempty"` // IP 归属地（如可获取）
	Error          string   `json:"error,omitempty"`
	Status         string   `json:"status,omitempty"`
	StatusCode     int      `json:"statusCode,omitempty"`     // HTTP状态码（HTTP服务）或0（非HTTP服务）
	RequestPacket  string   `json:"requestPacket,omitempty"`  // 请求包原始数据
	ResponsePacket string   `json:"responsePacket,omitempty"` // 响应包原始数据
	Technologies   []string `json:"technologies,omitempty"`   // 指纹识别技术栈
	CMS            string   `json:"cms,omitempty"`            // CMS信息
}

// HostScanStats 主机扫描统计
type HostScanStats struct {
	TotalPorts       int64   `json:"totalPorts"`
	Checked          int64   `json:"checked"`
	OpenCount        int64   `json:"openCount"`
	ClosedOrFiltered int64   `json:"closedOrFiltered"`
	StartTime        int64   `json:"startTime"`
	PortsPerSecond   float64 `json:"portsPerSecond"`
	CurrentTarget    string  `json:"currentTarget"`
	// 新增字段
	ProgressPercent  float64 `json:"progressPercent"`  // 进度百分比
	ElapsedSeconds   float64 `json:"elapsedSeconds"`   // 已用时间（秒）
	EstimatedSeconds float64 `json:"estimatedSeconds"` // 预计剩余时间（秒）
	TotalTargets     int64   `json:"totalTargets"`     // 总目标数
	CompletedTargets int64   `json:"completedTargets"` // 已完成目标数
	// ScanPhase 为 alive_probe 时，checked/totalPorts 表示存活探测进度，非端口扫描
	ScanPhase          string `json:"scanPhase,omitempty"`          // alive_probe | port_scan
	PortScanTotalPorts int64  `json:"portScanTotalPorts,omitempty"` // 端口扫描任务总数（存活阶段时用于前端提示）
}

// AliveCheckResult 存活检测结果
type AliveCheckResult struct {
	TotalIPs    int `json:"totalIPs"`    // 总IP数（不包括域名）
	AliveIPs    int `json:"aliveIPs"`    // 存活IP数
	DeadIPs     int `json:"deadIPs"`     // 不存活IP数
	DomainCount int `json:"domainCount"` // 域名数量（不参与存活检测）
}

type HostscanNotice struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Host    string `json:"host,omitempty"`
}

// HostScanner 主机扫描器
type HostScanner struct {
	config        *HostscanConfig
	ctx           context.Context
	cancel        context.CancelFunc
	results       chan HostscanResult
	errors        chan error
	done          chan bool
	stats         HostScanStats
	currentTarget atomic.Value
	locationCache sync.Map
	aliveResult   *AliveCheckResult // 存活检测结果
	aliveResultMu sync.Mutex        // 保护存活检测结果
	tabId         string            // 前端标签页ID，用于事件路由
	proxyDialer   netproxy.Dialer   // 代理dialer（支持SOCKS5、HTTP、HTTPS）
	noticeMu      sync.Mutex
	notices       []HostscanNotice
	fullPortScanRequested bool
	// 存活检测阶段结束信号（供控制器在正确时机推送 hostscanAliveCheck）
	alivePhaseDone chan struct{}
	alivePhaseOnce sync.Once
	// 存活探测进度（端口扫描未开始时 GetStats 用其填充 checked/totalPorts）
	aliveProbeTotal int64
	aliveProbeDone  int64
	// portPhaseStarted：0 表示尚未进入端口投递阶段（可能正在存活探测）
	portPhaseStarted  int32
	portPhaseStartMs  int64
}

// 单段 CIDR 地址数超过此值时不展开到切片，改为流式遍历（支持 /8 等大网段，避免 OOM）
const maxCIDRExpandInline = 65536

// 所有「已展开」的独立目标（小 CIDR + 单行 IP/域名）总数上限，防止误粘贴海量条目撑爆内存
const maxTotalExpandedHosts = 5_000_000

// 存活探测 worker 数上限
const maxAliveCheckWorkers = 256

// 端口扫描并发 goroutine 上限（实际仍受任务队列与系统资源约束）
const maxPortScanWorkers = 2048

// NewHostScannerFromJSON 从 JSON 创建扫描器
func NewHostScannerFromJSON(configJSON string) (*HostScanner, error) {
	var cfg HostscanConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, err
	}
	return NewHostScanner(&cfg)
}

// dialTCP 统一的TCP连接方法，支持SOCKS5、HTTP、HTTPS代理
func (s *HostScanner) dialTCP(network, address string, timeout time.Duration) (net.Conn, error) {
	if s.proxyDialer != nil {
		// 使用代理（SOCKS5、HTTP或HTTPS）
		// 使用context控制超时，确保端口扫描的超时时间生效
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		type dialResult struct {
			conn net.Conn
			err  error
		}
		resultChan := make(chan dialResult, 1)

		go func() {
			conn, err := s.proxyDialer.Dial(network, address)
			// 立即尝试发送结果
			select {
			case resultChan <- dialResult{conn: conn, err: err}:
				// 成功发送结果
			case <-ctx.Done():
				// 如果context已取消（超时），关闭连接
				if conn != nil {
					conn.Close()
				}
			}
		}()

		select {
		case <-ctx.Done():
			// 超时，返回超时错误
			// 注意：goroutine可能还在运行，但我们已经超时了，返回错误
			// 如果goroutine后续完成，连接会被关闭（在goroutine中处理）
			return nil, fmt.Errorf("连接超时")
		case result := <-resultChan:
			// 检查是否在超时后返回（虽然不太可能，但为了安全）
			select {
			case <-ctx.Done():
				// 如果context已取消，关闭连接并返回超时错误
				if result.conn != nil {
					result.conn.Close()
				}
				return nil, fmt.Errorf("连接超时")
			default:
				// 正常返回结果
				// 如果 Dial 返回错误，直接返回错误（端口关闭）
				// 如果 Dial 成功，返回连接（端口开放）
				return result.conn, result.err
			}
		}
	}
	// 直连
	return net.DialTimeout(network, address, timeout)
}

// NewHostScanner 创建主机扫描器
func NewHostScanner(cfg *HostscanConfig) (*HostScanner, error) {
	// 检查代理类型：端口扫描只支持 SOCKS5 代理或直连
	if proxy.GlobalProxyManager != nil {
		proxyInfo := proxy.GlobalProxyManager.GetGlobalProxy()
		if proxyInfo.Enabled && proxyInfo.Config != nil {
			proxyType := proxyInfo.Config.Type
			if proxyType == "http" || proxyType == "https" {
				return nil, fmt.Errorf("端口扫描功能仅支持 SOCKS5 代理或直连模式，当前配置的是 %s 代理。请切换到 SOCKS5 代理或关闭代理后重试", strings.ToUpper(proxyType))
			}
		}
	}

	if cfg.Threads <= 0 {
		cfg.Threads = 500
	}
	if cfg.Threads > maxPortScanWorkers {
		cfg.Threads = maxPortScanWorkers
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 3
	}

	resolved, err := resolveScanTargets(cfg.Targets)
	if err != nil {
		return nil, err
	}
	if totalScanningUnits(resolved) == 0 {
		return nil, fmt.Errorf("至少需要一个目标")
	}

	// 处理端口预设
	var ports []int
	if cfg.PortPreset != "" && cfg.PortPreset != "custom" {
		ports = getPresetPorts(cfg.PortPreset)
		// 如果同时提供了手动端口，则合并
		if cfg.Ports != "" {
			customPorts, err2 := parsePorts(cfg.Ports)
			if err2 == nil && len(customPorts) > 0 {
				portSet := make(map[int]struct{})
				for _, p := range ports {
					portSet[p] = struct{}{}
				}
				for _, p := range customPorts {
					portSet[p] = struct{}{}
				}
				ports = make([]int, 0, len(portSet))
				for p := range portSet {
					ports = append(ports, p)
				}
			}
		}
	} else {
		// 使用手动输入的端口（默认custom或空字符串）
		if cfg.Ports == "" {
			return nil, fmt.Errorf("端口列表为空，请选择端口预设或输入端口范围")
		}
		ports, err = parsePorts(cfg.Ports)
		if err != nil {
			return nil, err
		}
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("端口列表为空")
	}

	// 排除指定端口
	if len(cfg.ExcludePorts) > 0 {
		excludeMap := make(map[int]bool)
		for _, p := range cfg.ExcludePorts {
			excludeMap[p] = true
		}
		filteredPorts := make([]int, 0, len(ports))
		for _, p := range ports {
			if !excludeMap[p] {
				filteredPorts = append(filteredPorts, p)
			}
		}
		ports = filteredPorts
		if len(ports) == 0 {
			return nil, fmt.Errorf("排除端口后，端口列表为空")
		}
	}

	// 先记录“是否请求了全端口扫描”，避免后续重排端口顺序后无法识别
	fullPortScanRequested := false
	if len(ports) == 65535 {
		tmp := append([]int(nil), ports...)
		sort.Ints(tmp)
		if tmp[0] == 1 && tmp[len(tmp)-1] == 65535 {
			fullPortScanRequested = true
		}
	}

	// 规划扫描顺序
	sort.Ints(ports)
	if isFullPortRange(ports) {
		// 全端口扫描：先跑常见Top1000，再对剩余端口做二分扫描
		ports = buildFullScanOrder(ports)
	} else {
		// 其它情况直接使用二分扫描顺序，快速覆盖全范围
		ports = buildBinaryPortOrder(ports)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 获取代理dialer（支持SOCKS5、HTTP、HTTPS）
	var proxyDialer netproxy.Dialer
	if proxy.GlobalProxyManager != nil {
		dialer, err := proxy.GlobalProxyManager.GetProxyDialer()
		if err == nil && dialer != nil {
			proxyDialer = dialer
		}
	}

	scanner := &HostScanner{
		config:      cfg,
		ctx:         ctx,
		cancel:      cancel,
		results:     make(chan HostscanResult, 1024),
		errors:      make(chan error, 64),
		done:        make(chan bool, 1),
		alivePhaseDone: make(chan struct{}),
		proxyDialer: proxyDialer,
		fullPortScanRequested: fullPortScanRequested,
		stats: HostScanStats{
			TotalPorts: int64(totalScanningUnits(resolved)) * int64(len(ports)),
			StartTime:  time.Now().UnixMilli(),
		},
		tabId: cfg.TabId,
	}

	go scanner.scan(resolved, ports)

	return scanner, nil
}

// Start 返回结果与错误通道
func (s *HostScanner) Start() (<-chan HostscanResult, <-chan error) {
	return s.results, s.errors
}

// AlivePhaseDone 在存活检测阶段结束（或跳过存活检测）后关闭，供控制器同步推送 hostscanAliveCheck
func (s *HostScanner) AlivePhaseDone() <-chan struct{} {
	return s.alivePhaseDone
}

func (s *HostScanner) markAlivePhaseDone() {
	s.alivePhaseOnce.Do(func() {
		close(s.alivePhaseDone)
	})
}

// Stop 停止扫描
func (s *HostScanner) Stop() {
	// 取消context，停止所有goroutine
	s.cancel()
	// 发送完成信号（使用非阻塞方式，如果已发送过则跳过）
	select {
	case s.done <- true:
	default:
		// 如果done channel已满，说明已经发送过停止信号，忽略
	}
}

// GetStats 获取当前统计
func (s *HostScanner) GetStats() HostScanStats {
	now := time.Now()
	elapsed := float64(now.UnixMilli()-s.stats.StartTime) / 1000.0
	portChecked := atomic.LoadInt64(&s.stats.Checked)
	stats := s.stats

	stats.OpenCount = atomic.LoadInt64(&s.stats.OpenCount)
	stats.ClosedOrFiltered = atomic.LoadInt64(&s.stats.ClosedOrFiltered)
	stats.ElapsedSeconds = elapsed

	aliveTotal := atomic.LoadInt64(&s.aliveProbeTotal)
	aliveDone := atomic.LoadInt64(&s.aliveProbeDone)
	inAliveProbe := s.config.CheckAlive && atomic.LoadInt32(&s.portPhaseStarted) == 0 &&
		aliveTotal > 0 && aliveDone < aliveTotal

	if inAliveProbe {
		stats.ScanPhase = "alive_probe"
		stats.PortScanTotalPorts = s.stats.TotalPorts
		stats.Checked = aliveDone
		stats.TotalPorts = aliveTotal
		if aliveTotal > 0 {
			stats.ProgressPercent = float64(aliveDone) / float64(aliveTotal) * 100
			if stats.ProgressPercent > 100 {
				stats.ProgressPercent = 100
			}
		}
		if elapsed > 0 {
			stats.PortsPerSecond = float64(aliveDone) / elapsed
		}
		if aliveDone > 0 && aliveDone < aliveTotal {
			rate := float64(aliveDone) / elapsed
			if rate > 0 {
				stats.EstimatedSeconds = float64(aliveTotal-aliveDone) / rate
			}
		}
	} else {
		stats.ScanPhase = "port_scan"
		stats.Checked = portChecked
		stats.TotalPorts = s.stats.TotalPorts
		portStart := atomic.LoadInt64(&s.portPhaseStartMs)
		portElapsed := float64(now.UnixMilli()-portStart) / 1000.0
		if portElapsed < 0.001 {
			portElapsed = 0.001
		}
		if portChecked > 0 {
			stats.PortsPerSecond = float64(portChecked) / portElapsed
		} else if elapsed > 0 {
			stats.PortsPerSecond = 0
		}
		if stats.TotalPorts > 0 {
			stats.ProgressPercent = float64(portChecked) / float64(stats.TotalPorts) * 100
			if stats.ProgressPercent > 100 {
				stats.ProgressPercent = 100
			}
		}
		if stats.PortsPerSecond > 0 && stats.TotalPorts > portChecked {
			remaining := float64(stats.TotalPorts - portChecked)
			stats.EstimatedSeconds = remaining / stats.PortsPerSecond
		}
	}

	if v := s.currentTarget.Load(); v != nil {
		if target, ok := v.(string); ok {
			stats.CurrentTarget = target
		}
	}

	return stats
}

// dispatchHostPortJobs 将某一主机上的端口探测任务写入队列；返回 true 表示应中止整个扫描
func (s *HostScanner) dispatchHostPortJobs(host string, allPorts []int, fullRangeScan bool, jobs chan<- job) bool {
	targetPorts := allPorts
	if fullRangeScan && !s.config.DisableFirewallDetection {
		suspected, openCount, checkedCount := s.looksLikeFirewallAllOpen(host)
		if suspected {
			targetPorts = buildTargetTop100Ports(allPorts)
			s.addNotice(HostscanNotice{
				Type: "firewall_auto_downgrade",
				Host: host,
				Message: fmt.Sprintf(
					"目标 %s 疑似触发防火墙“全端口存活”策略（采样命中 %d/%d），已自动切换为 Top100 端口扫描。如需继续全端口扫描，请在高级配置中关闭防火墙检测。",
					host,
					openCount,
					checkedCount,
				),
			})
		}
	}
	for _, port := range targetPorts {
		select {
		case <-s.ctx.Done():
			return true
		default:
		}
		select {
		case jobs <- job{host: host, port: port}:
		case <-s.ctx.Done():
			return true
		}
	}
	return false
}

type job struct {
	host string
	port int
}

func (s *HostScanner) scan(work resolvedScanTargets, ports []int) {
	defer close(s.results)
	defer close(s.errors)

	if !s.config.CheckAlive {
		atomic.StoreInt64(&s.portPhaseStartMs, time.Now().UnixMilli())
		atomic.StoreInt32(&s.portPhaseStarted, 1)
	}

	if s.config.CheckAlive {
		aliveHosts, _, _ := s.filterAliveTargets(work)
		s.markAlivePhaseDone()
		if len(aliveHosts) == 0 {
			atomic.StoreInt64(&s.portPhaseStartMs, time.Now().UnixMilli())
			atomic.StoreInt32(&s.portPhaseStarted, 1)
			s.signalDone()
			return
		}
		work = resolvedScanTargets{Hosts: aliveHosts}
		atomic.StoreInt64(&s.portPhaseStartMs, time.Now().UnixMilli())
		atomic.StoreInt32(&s.portPhaseStarted, 1)
	} else {
		s.markAlivePhaseDone()
	}

	jobs := make(chan job, 1024)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			select {
			case <-s.ctx.Done():
				return
			default:
			}
			s.scanSingle(j.host, j.port)
		}
	}

	for i := 0; i < s.config.Threads; i++ {
		wg.Add(1)
		go worker()
	}

	fullRangeScan := s.fullPortScanRequested

	for _, host := range work.Hosts {
		select {
		case <-s.ctx.Done():
			close(jobs)
			wg.Wait()
			s.signalDone()
			return
		default:
		}
		if s.dispatchHostPortJobs(host, ports, fullRangeScan, jobs) {
			close(jobs)
			wg.Wait()
			s.signalDone()
			return
		}
	}

	for _, nn := range work.LargeNets {
		ip4 := nn.IP.To4()
		if ip4 == nil {
			continue
		}
		for cur := ip4.Mask(nn.Mask); nn.Contains(cur); incIP(cur) {
			select {
			case <-s.ctx.Done():
				close(jobs)
				wg.Wait()
				s.signalDone()
				return
			default:
			}
			ipCopy := make(net.IP, len(cur))
			copy(ipCopy, cur)
			if s.dispatchHostPortJobs(ipCopy.String(), ports, fullRangeScan, jobs) {
				close(jobs)
				wg.Wait()
				s.signalDone()
				return
			}
		}
	}

	close(jobs)
	wg.Wait()
	s.signalDone()
}

func buildTargetTop100Ports(basePorts []int) []int {
	baseSet := make(map[int]struct{}, len(basePorts))
	for _, p := range basePorts {
		baseSet[p] = struct{}{}
	}
	top100 := getTop100Ports()
	out := make([]int, 0, len(top100))
	for _, p := range top100 {
		if _, ok := baseSet[p]; ok {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return top100
	}
	return out
}

// looksLikeFirewallAllOpen 通过小样本探测判断是否疑似“全端口都存活”的防火墙策略。
func (s *HostScanner) looksLikeFirewallAllOpen(host string) (bool, int, int) {
	timeout := time.Duration(s.config.Timeout) * time.Second
	if timeout > 2*time.Second {
		timeout = 2 * time.Second
	}
	if timeout <= 0 {
		timeout = 1 * time.Second
	}

	samplePorts := []int{1, 7, 19, 37, 79, 111, 389, 515, 873, 1900, 2049, 5432, 6000, 65535}
	sampleSize := s.config.FirewallDetectSampleSize
	if sampleSize <= 0 || sampleSize > len(samplePorts) {
		sampleSize = len(samplePorts)
	}
	threshold := s.config.FirewallDetectThreshold
	if threshold <= 0 || threshold > 1 {
		threshold = 0.9
	}

	openCount := 0
	checkedCount := 0
	for _, p := range samplePorts[:sampleSize] {
		select {
		case <-s.ctx.Done():
			return false, openCount, checkedCount
		default:
		}
		checkedCount++
		addr := net.JoinHostPort(host, strconv.Itoa(p))
		conn, err := s.dialTCP("tcp", addr, timeout)
		if err == nil {
			openCount++
			_ = conn.Close()
		}
	}
	if checkedCount == 0 {
		return false, 0, 0
	}
	ratio := float64(openCount) / float64(checkedCount)
	return ratio >= threshold, openCount, checkedCount
}

func (s *HostScanner) signalDone() {
	select {
	case s.done <- true:
	default:
	}
}

func (s *HostScanner) addNotice(notice HostscanNotice) {
	if strings.TrimSpace(notice.Message) == "" {
		return
	}
	s.noticeMu.Lock()
	s.notices = append(s.notices, notice)
	s.noticeMu.Unlock()
}

func (s *HostScanner) PopNotices() []HostscanNotice {
	s.noticeMu.Lock()
	defer s.noticeMu.Unlock()
	if len(s.notices) == 0 {
		return nil
	}
	out := make([]HostscanNotice, len(s.notices))
	copy(out, s.notices)
	s.notices = nil
	return out
}

func (s *HostScanner) scanSingle(host string, port int) {
	// 首先检查context是否已取消
	select {
	case <-s.ctx.Done():
		return
	default:
	}

	address := net.JoinHostPort(host, strconv.Itoa(port))
	s.currentTarget.Store(address)

	timeout := time.Duration(s.config.Timeout) * time.Second

	// 使用统一的TCP连接方法（支持SOCKS5代理）
	// 默认只连一次，保证整体速度；
	// 只有在使用代理时并且出现“超时/临时错误”才进行一次轻量级重试，兼顾准确率与性能。
	var conn net.Conn
	var err error

	// 第一次尝试
	select {
	case <-s.ctx.Done():
		return
	default:
	}
	conn, err = s.dialTCP("tcp", address, timeout)

	// 仅在代理环境下，对超时/临时错误再重试一次
	if err != nil && s.proxyDialer != nil {
		if netErr, ok := err.(net.Error); ok && (netErr.Timeout() || netErr.Temporary()) {
			select {
			case <-s.ctx.Done():
				return
			default:
			}
			time.Sleep(50 * time.Millisecond)
			conn, err = s.dialTCP("tcp", address, timeout)
		}
	}

	atomic.AddInt64(&s.stats.Checked, 1)

	if err != nil {
		// 任何错误都视为端口关闭（包括超时、连接拒绝等）
		atomic.AddInt64(&s.stats.ClosedOrFiltered, 1)
		if s.config.EmitClosedPorts {
			select {
			case s.results <- HostscanResult{
				Target:   host,
				Port:     port,
				Open:     false,
				Protocol: "tcp",
				Error:    err.Error(),
				Status:   "closed",
			}:
			case <-s.ctx.Done():
			}
		}
		return
	}

	// 连接验证通过，认为端口开放
	atomic.AddInt64(&s.stats.OpenCount, 1)
	_ = conn.Close()

	// 优先尝试HTTP协议探测
	var isHTTP bool
	var httpResult HTTPBannerResult
	var technologies []string
	var cms string
	var title string
	var banner string
	var service string
	var protocol string
	var requestPacket string
	var responsePacket string
	var statusCode int

	// 先尝试HTTP探测（包括HTTPS）
	httpResult = probeHTTPBanner(host, port, timeout, "")
	if httpResult.ResponsePacket != "" {
		// HTTP探测成功，检查响应是否真的是HTTP响应
		responseLower := strings.ToLower(httpResult.ResponsePacket)
		// 检查HTTP响应特征
		isHTTPResponse := strings.HasPrefix(responseLower, "http/") ||
			strings.Contains(responseLower, "content-type:") ||
			strings.Contains(responseLower, "server:") ||
			strings.Contains(responseLower, "location:") ||
			strings.Contains(responseLower, "status:")

		// 检查是否是JSON格式的API响应（可能是HTTP服务返回的JSON）
		isJSONResponse := strings.HasPrefix(strings.TrimSpace(httpResult.ResponsePacket), "{") &&
			(strings.Contains(responseLower, "server_id") ||
				strings.Contains(responseLower, "\"server\"") ||
				strings.Contains(responseLower, "\"status\"") ||
				strings.Contains(responseLower, "\"message\""))

		if isHTTPResponse || isJSONResponse {
			// 确认是HTTP服务或API服务
			isHTTP = true
			requestPacket = httpResult.RequestPacket
			responsePacket = httpResult.ResponsePacket
			statusCode = httpResult.StatusCode
			service = httpResult.Service
			if service == "" {
				service = "http"
			}

			// 判断是HTTP还是HTTPS
			if port == 443 || strings.Contains(strings.ToLower(service), "https") {
				protocol = "https"
			} else {
				protocol = "http"
			}

			// 提取标题和banner
			title = extractTitleFromBanner(responsePacket)
			banner = buildBannerFromRaw(service, responsePacket)

			// 如果开启端口指纹识别，进行指纹识别
			if s.config.EnablePortFingerprint {
				fingerResult := performPortFingerprint(host, port, protocol, timeout, responsePacket)
				if fingerResult != nil {
					technologies = fingerResult.Technologies
					cms = fingerResult.CMS
					// 如果指纹识别获取到标题，使用指纹识别的标题
					if fingerResult.Title != "" {
						title = fingerResult.Title
					}
				}
			}

			// 如果是HTTP/HTTPS服务但没有获取到标题，使用"None"代替
			if title == "" {
				title = "None"
			}
		}
	}

	// 如果HTTP探测失败，使用原有的enhanceService逻辑
	if !isHTTP {
		// 重新连接进行服务识别
		conn2, err := s.dialTCP("tcp", address, timeout)
		if err != nil {
			// 连接失败，返回基本结果
			location := s.getLocationForHost(host)
			result := HostscanResult{
				Target:   host,
				Port:     port,
				Open:     true,
				Service:  "unknown",
				Protocol: "tcp",
				Banner:   "",
				Location: location,
				Status:   "open",
			}
			select {
			case s.results <- result:
			case <-s.ctx.Done():
			}
			return
		}

		service, raw := enhanceService(host, port, conn2, timeout)
		_ = conn2.Close()

		protocol = detectProtocol(port, service)
		title = extractTitleFromBanner(raw)
		banner = buildBannerFromRaw(service, raw)
		// 保存原始响应包，用于后续查看
		responsePacket = raw

		// 如果开启端口指纹识别，尝试获取更完整的banner信息
		// 注意：enhanceService已经尝试获取banner了，这里主要是为了确保获取完整信息
		if s.config.EnablePortFingerprint && (raw == "" || len(raw) < 100) {
			// 如果banner信息较少，尝试再次连接获取更完整的信息
			conn3, err := s.dialTCP("tcp", address, timeout)
			if err == nil {
				_ = conn3.SetReadDeadline(time.Now().Add(timeout))
				// 根据服务类型发送特定探测
				if port == 6379 {
					// Redis: 发送INFO命令获取详细信息
					_, _ = conn3.Write([]byte("INFO server\r\n"))
				} else if port == 3306 {
					// MySQL: 读取握手包
					buf := make([]byte, 512)
					n, _ := conn3.Read(buf)
					if n > 0 {
						raw = string(buf[:n])
						banner = buildBannerFromRaw(service, raw)
						responsePacket = raw
					}
				} else {
					// 通用：读取更多数据
					buf := make([]byte, 4096)
					n, _ := conn3.Read(buf)
					if n > 0 {
						raw = string(buf[:n])
						banner = buildBannerFromRaw(service, raw)
						responsePacket = raw
					}
				}
				conn3.Close()
			}
		}
	}

	location := s.getLocationForHost(host)

	result := HostscanResult{
		Target:         host,
		Port:           port,
		Open:           true,
		Service:        service,
		Protocol:       protocol,
		Banner:         banner,
		Title:          title,
		Location:       location,
		Status:         "open",
		StatusCode:     statusCode,
		RequestPacket:  requestPacket,
		ResponsePacket: responsePacket,
		Technologies:   technologies,
		CMS:            cms,
	}

	select {
	case s.results <- result:
	case <-s.ctx.Done():
		return
	}
}

// 通过简单 banner 进一步识别真实服务，并返回服务名和原始 banner
func enhanceService(host string, port int, conn net.Conn, timeout time.Duration) (string, string) {
	// 只有拿到协议证据后才确认服务，避免端口先验误判
	base := ""

	// 如果是典型 Web 端口，优先通过 HTTP 请求获取更丰富的信息（Server/Header/Title）
	// 注意：这里只返回service和raw，数据包会在scanSingle中单独处理
	if base == "http" || base == "https" {
		if result := probeHTTPBanner(host, port, timeout, base); result.Raw != "" {
			return result.Service, result.Raw
		}
	}

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	reader := bufio.NewReader(conn)

	var bannerBuf []byte

	// Redis: 尝试发送 PING\r\n，期待 +PONG（不限制端口，因为Redis可能运行在其他端口）
	// 先尝试读取一行，如果是Redis响应格式，直接返回
	peekLine, _ := reader.Peek(10)
	if len(peekLine) > 0 && (peekLine[0] == '+' || peekLine[0] == '-' || peekLine[0] == ':') {
		// 可能是Redis响应格式，尝试读取
		line, _ := reader.ReadString('\n')
		bannerBuf = []byte(line)
		lineUpper := strings.ToUpper(strings.TrimSpace(line))
		if strings.HasPrefix(lineUpper, "+PONG") ||
			strings.HasPrefix(lineUpper, "+OK") ||
			strings.HasPrefix(lineUpper, "-ERR") ||
			strings.HasPrefix(lineUpper, ":") {
			// 确认是Redis响应，尝试获取更多信息
			_, _ = conn.Write([]byte("INFO server\r\n"))
			info, _ := reader.ReadString('\n')
			if strings.Contains(strings.ToLower(info), "redis_version") {
				return "redis", extractRedisVersion(info)
			}
			return "redis", truncateBanner(string(bannerBuf))
		}
		// 如果不是Redis响应，将读取的内容放回（通过重新连接或使用其他方式）
		// 这里我们继续处理，因为可能已经读取了部分数据
	}

	// 如果端口是6379，主动发送PING命令
	if port == 6379 {
		_, _ = conn.Write([]byte("PING\r\n"))
		line, _ := reader.ReadString('\n')
		bannerBuf = []byte(line)
		if strings.HasPrefix(strings.ToUpper(line), "+PONG") {
			return "redis", truncateBanner(string(bannerBuf))
		}
	}

	// MySQL: 读取握手包，里面带有版本和 "mysql" 关键字
	if port == 3306 {
		buf := make([]byte, 512)
		n, _ := reader.Read(buf)
		if n > 0 {
			bannerBuf = buf[:n]
			if mysqlVer := extractMySQLVersionFromHandshake(bannerBuf); mysqlVer != "" {
				return "mysql", mysqlVer
			}
			banner := string(bannerBuf)
			bannerLower := strings.ToLower(banner)
			if strings.Contains(bannerLower, "mysql") {
				// 尝试提取MySQL版本信息
				version := extractMySQLVersion(banner)
				if version != "" {
					return "mysql", version
				}
				return "mysql", truncateBanner(banner)
			}
		}
	}

	// SMB: 端口445，尝试读取SMB协议头
	if port == 445 {
		buf := make([]byte, 512)
		n, _ := reader.Read(buf)
		if n > 0 {
			bannerBuf = buf[:n]
			// SMB协议特征：通常以 \x00\x00 开头，包含SMB标识
			// 或者直接包含SMB字符串
			smbInfo := extractSMBInfo(buf[:n])
			if smbInfo != "" && smbInfo != "SMB" {
				return "smb", smbInfo
			}
			// 尝试将二进制数据转换为可读字符串
			bannerStr := truncateBanner(string(buf[:n]))
			if bannerStr != "" {
				return "smb", "SMB"
			}
		}
		// 端口445默认认为是SMB服务
		return "smb", "SMB"
	}

	// SSH: 端口22，读取SSH banner并提取版本信息
	if port == 22 {
		line, _ := reader.ReadString('\n')
		if line != "" {
			bannerBuf = []byte(line)
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(line), "ssh-") {
				// 提取SSH版本信息，格式：SSH-2.0-OpenSSH_8.2
				sshVersion := extractSSHVersion(line)
				if sshVersion != "" {
					return "ssh", sshVersion
				}
				return "ssh", truncateBanner(line)
			}
		}
	}

	// FTP: 端口21，读取FTP欢迎消息（可能有多行）
	if port == 21 {
		var ftpBanner strings.Builder
		for i := 0; i < 5; i++ { // 最多读取5行
			line, err := reader.ReadString('\n')
			if err != nil || line == "" {
				break
			}
			ftpBanner.WriteString(line)
			line = strings.TrimSpace(line)
			// FTP欢迎消息通常以220开头，但可能有多行
			if strings.HasPrefix(line, "220") {
				// 继续读取直到遇到非220开头的行或超时
				for j := 0; j < 3; j++ {
					nextLine, err := reader.ReadString('\n')
					if err != nil {
						break
					}
					ftpBanner.WriteString(nextLine)
					if !strings.HasPrefix(strings.TrimSpace(nextLine), "220") {
						break
					}
				}
				bannerStr := truncateBanner(ftpBanner.String())
				return "ftp", bannerStr
			}
		}
		if ftpBanner.Len() > 0 {
			return "ftp", truncateBanner(ftpBanner.String())
		}
	}

	// Telnet: 端口23，读取telnet banner（可能有多行）
	if port == 23 {
		var telnetBanner strings.Builder
		// Telnet会立即返回banner，读取多行直到超时或遇到特定字符
		for i := 0; i < 10; i++ {
			line, err := reader.ReadString('\n')
			if err != nil || line == "" {
				break
			}
			telnetBanner.WriteString(line)
			// 如果遇到登录提示，停止读取
			if strings.Contains(strings.ToLower(line), "login:") ||
				strings.Contains(strings.ToLower(line), "username:") ||
				strings.Contains(strings.ToLower(line), "password:") {
				break
			}
		}
		if telnetBanner.Len() > 0 {
			return "telnet", truncateBanner(telnetBanner.String())
		}
	}

	// SMTP: 端口25，读取SMTP欢迎消息
	if port == 25 {
		line, _ := reader.ReadString('\n')
		if line != "" {
			bannerBuf = []byte(line)
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "220") {
				// 提取SMTP服务器信息
				smtpInfo := extractSMTPInfo(line)
				if smtpInfo != "" {
					return "smtp", smtpInfo
				}
				return "smtp", truncateBanner(line)
			}
		}
	}

	// POP3: 端口110，读取POP3欢迎消息
	if port == 110 {
		line, _ := reader.ReadString('\n')
		if line != "" {
			bannerBuf = []byte(line)
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "+OK") {
				return "pop3", truncateBanner(line)
			}
		}
	}

	// IMAP: 端口143，读取IMAP欢迎消息
	if port == 143 {
		line, _ := reader.ReadString('\n')
		if line != "" {
			bannerBuf = []byte(line)
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "* OK") || strings.HasPrefix(line, "* PREAUTH") {
				return "imap", truncateBanner(line)
			}
		}
	}

	// PostgreSQL: 端口5432，读取PostgreSQL握手包
	if port == 5432 {
		buf := make([]byte, 512)
		n, _ := reader.Read(buf)
		if n > 0 {
			bannerBuf = buf[:n]
			banner := string(bannerBuf)
			// PostgreSQL握手包通常以长度字节开始，然后包含版本信息
			// 尝试查找版本字符串
			pgVersion := extractPostgreSQLVersion(banner)
			if pgVersion != "" {
				return "postgresql", pgVersion
			}
			// 如果包含postgres关键字
			if strings.Contains(strings.ToLower(banner), "postgres") {
				return "postgresql", truncateBanner(banner)
			}
		}
	}

	// MSSQL: 端口1433，读取SQL Server握手包
	if port == 1433 {
		buf := make([]byte, 512)
		n, _ := reader.Read(buf)
		if n > 0 {
			bannerBuf = buf[:n]
			banner := string(bannerBuf)
			// SQL Server握手包可能包含版本信息
			mssqlInfo := extractMSSQLInfo(banner)
			if mssqlInfo != "" {
				return "mssql", mssqlInfo
			}
			if strings.Contains(strings.ToLower(banner), "sql server") ||
				strings.Contains(strings.ToLower(banner), "microsoft sql") {
				return "mssql", truncateBanner(banner)
			}
		}
	}

	// RDP: 端口3389，读取RDP协议头
	if port == 3389 {
		buf := make([]byte, 512)
		n, _ := reader.Read(buf)
		if n > 0 {
			bannerBuf = buf[:n]
			// RDP协议通常以0x03 0x00开头
			if len(buf) >= 2 && buf[0] == 0x03 && buf[1] == 0x00 {
				return "rdp", "RDP (Microsoft Remote Desktop)"
			}
		}
	}

	// VNC: 端口5900，读取VNC协议头
	if port == 5900 {
		buf := make([]byte, 12)
		n, _ := reader.Read(buf)
		if n >= 12 {
			bannerBuf = buf[:n]
			// VNC协议通常以"RFB"开头
			if strings.HasPrefix(string(buf), "RFB") {
				version := string(buf[:12])
				return "vnc", truncateBanner(version)
			}
		}
	}

	// MongoDB: 端口27017，读取MongoDB握手包
	if port == 27017 {
		buf := make([]byte, 512)
		n, _ := reader.Read(buf)
		if n > 0 {
			bannerBuf = buf[:n]
			banner := string(bannerBuf)
			if strings.Contains(strings.ToLower(banner), "mongodb") {
				mongoVersion := extractMongoDBVersion(banner)
				if mongoVersion != "" {
					return "mongodb", mongoVersion
				}
				return "mongodb", truncateBanner(banner)
			}
		}
	}

	// 通用：读取banner（支持多行读取，类似telnet）
	// 对于未知服务，尝试读取多行banner直到超时或遇到特定条件
	// 注意：如果前面的特定服务检测已经读取了banner，这里不会再读取
	if len(bannerBuf) == 0 {
		var bannerBuilder strings.Builder
		var totalRead int
		maxReadSize := 4096 // 最大读取4KB

		// 先尝试单次读取
		buf := make([]byte, 1024)
		n, _ := reader.Read(buf)
		if n > 0 {
			bannerBuf = buf[:n]
			bannerBuilder.Write(buf[:n])
			totalRead += n

			// 对于文本协议，尝试继续读取多行
			bannerStr := string(buf[:n])
			// 如果包含换行符，可能是文本协议，尝试读取更多行
			if strings.Contains(bannerStr, "\n") {
				for totalRead < maxReadSize {
					// 设置较短的超时时间，避免等待太久
					conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
					nextBuf := make([]byte, 512)
					nextN, err := reader.Read(nextBuf)
					if err != nil || nextN == 0 {
						break
					}
					bannerBuilder.Write(nextBuf[:nextN])
					totalRead += nextN
					// 如果遇到登录提示或特定结束标志，停止读取
					nextStr := string(nextBuf[:nextN])
					if strings.Contains(strings.ToLower(nextStr), "login:") ||
						strings.Contains(strings.ToLower(nextStr), "username:") ||
						strings.Contains(strings.ToLower(nextStr), "password:") ||
						strings.Contains(strings.ToLower(nextStr), "press any key") {
						break
					}
				}
				// 更新bannerBuf为完整内容
				if bannerBuilder.Len() > n {
					bannerBuf = []byte(bannerBuilder.String())
				}
			}
		}

		if bannerBuilder.Len() > 0 {
			bannerStr := bannerBuilder.String()
			bannerLower := strings.ToLower(bannerStr)

			// 检查是否是JSON格式的响应（可能是API服务，如4222端口的VRML服务或NATS服务器）
			// 检查是否以"INFO "开头（NATS服务器特征）
			if strings.HasPrefix(strings.TrimSpace(bannerStr), "INFO ") {
				// 提取JSON部分
				jsonPart := strings.TrimSpace(bannerStr[5:]) // 去掉"INFO "前缀
				if strings.HasPrefix(jsonPart, "{") {
					// 检查是否包含server_id、server_name等特征
					if strings.Contains(bannerLower, "server_id") ||
						strings.Contains(bannerLower, "server_name") ||
						strings.Contains(bannerLower, "\"server\"") {
						// 检查是否是NATS服务器（通常包含go版本和proto字段）
						if strings.Contains(bannerLower, "\"go\"") ||
							strings.Contains(bannerLower, "\"proto\"") ||
							strings.Contains(bannerLower, "\"version\"") {
							return "nats", truncateBanner(bannerStr)
						}
						// 检查是否是VRML服务
						if strings.Contains(bannerLower, "vrml") {
							return "vrml-multi-use", truncateBanner(bannerStr)
						}
						// 其他JSON API服务，但包含server_id，可能是VRML类型服务
						// 根据端口4222判断，这通常是VRML或NATS服务
						if strings.Contains(bannerLower, "server_id") {
							return "vrml-multi-use", truncateBanner(bannerStr)
						}
						// 其他JSON API服务
						return "api-json", truncateBanner(bannerStr)
					}
				}
			}
			// 检查是否直接是JSON格式（不以INFO开头）
			if strings.HasPrefix(strings.TrimSpace(bannerStr), "{") {
				// 尝试解析JSON，提取服务信息
				if strings.Contains(bannerLower, "server_id") || strings.Contains(bannerLower, "\"server\"") {
					// 检查是否是NATS服务器
					if strings.Contains(bannerLower, "\"go\"") ||
						strings.Contains(bannerLower, "\"proto\"") ||
						strings.Contains(bannerLower, "\"version\"") {
						return "nats", truncateBanner(bannerStr)
					}
					// 可能是VRML或其他API服务
					if strings.Contains(bannerLower, "vrml") {
						return "vrml-multi-use", truncateBanner(bannerStr)
					}
					// 通用JSON API服务
					return "api-json", truncateBanner(bannerStr)
				}
			}

			switch {
			case strings.Contains(bannerLower, "vrml"):
				return "vrml-multi-use", truncateBanner(bannerStr)
			case strings.Contains(bannerLower, "nginx"):
				return "nginx", truncateBanner(bannerStr)
			case strings.Contains(bannerLower, "apache"):
				return "apache", truncateBanner(bannerStr)
			case strings.Contains(bannerLower, "iis"):
				return "microsoft-iis", truncateBanner(bannerStr)
			case strings.Contains(bannerLower, "tomcat"):
				return "tomcat", truncateBanner(bannerStr)
			case strings.Contains(bannerLower, "redis"):
				// 尝试获取Redis版本信息
				_, _ = conn.Write([]byte("INFO server\r\n"))
				info, _ := reader.ReadString('\n')
				if strings.Contains(strings.ToLower(info), "redis_version") {
					return "redis", extractRedisVersion(info)
				}
				return "redis", truncateBanner(bannerStr)
			case strings.Contains(bannerLower, "mysql"):
				version := extractMySQLVersion(bannerStr)
				if version != "" {
					return "mysql", version
				}
				return "mysql", truncateBanner(bannerStr)
			case strings.Contains(bannerLower, "mongodb"):
				return "mongodb", truncateBanner(bannerStr)
			case strings.Contains(bannerLower, "http"):
				if port == 443 {
					return "https", truncateBanner(bannerStr)
				}
				return "http", truncateBanner(bannerStr)
			}
		}
	}

	// 如果已经读取了banner，返回它；否则返回基础服务名
	if len(bannerBuf) > 0 {
		if base != "" {
			return base, truncateBanner(string(bannerBuf))
		}
		return "unknown", truncateBanner(string(bannerBuf))
	}
	if base != "" {
		return base, ""
	}
	return "unknown", ""
}

func guessServiceByPort(port int) string {
	switch port {
	case 80:
		return "http"
	case 443:
		return "https"
	case 8080, 8081, 8000, 8888:
		return "http-alt"
	case 21:
		return "ftp"
	case 22:
		return "ssh"
	case 23:
		return "telnet"
	case 25:
		return "smtp"
	case 53:
		return "dns"
	case 110:
		return "pop3"
	case 143:
		return "imap"
	case 445:
		return "smb"
	case 3306:
		return "mysql"
	case 3389:
		return "rdp"
	case 5432:
		return "postgresql"
	case 5900:
		return "vnc"
	case 6379:
		return "redis"
	case 1433:
		return "mssql"
	case 27017:
		return "mongodb"
	default:
		return ""
	}
}

// resolvedScanTargets 扫描目标：不超过 maxCIDRExpandInline 的 CIDR 展开到 Hosts；更大网段保留在 LargeNets 中流式遍历
type resolvedScanTargets struct {
	Hosts     []string
	LargeNets []*net.IPNet
}

func ipv4CIDRHostCount(ipNet *net.IPNet) (int64, error) {
	if ipNet.IP.To4() == nil {
		return 0, fmt.Errorf("IPv6 CIDR 暂不支持展开为扫描目标，请使用 IPv4 或具体地址")
	}
	ones, bits := ipNet.Mask.Size()
	if bits != 32 {
		return 0, fmt.Errorf("无效的 IPv4 CIDR")
	}
	hostBits := 32 - ones
	if hostBits < 0 || hostBits > 32 {
		return 0, fmt.Errorf("无效的 IPv4 CIDR")
	}
	return int64(uint64(1) << uint(hostBits)), nil
}

func cloneIPNet(n *net.IPNet) *net.IPNet {
	if n == nil {
		return nil
	}
	c := *n
	c.IP = append(net.IP(nil), n.IP...)
	c.Mask = append(net.IPMask(nil), n.Mask...)
	return &c
}

func totalScanningUnits(r resolvedScanTargets) int64 {
	var n int64
	n += int64(len(r.Hosts))
	for _, nn := range r.LargeNets {
		c, err := ipv4CIDRHostCount(nn)
		if err != nil {
			continue
		}
		n += c
	}
	return n
}

func resolveScanTargets(raw []string) (resolvedScanTargets, error) {
	var r resolvedScanTargets
	seen := make(map[string]struct{})
	for _, t := range raw {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "http://") {
			t = strings.TrimPrefix(t, "http://")
		} else if strings.HasPrefix(t, "https://") {
			t = strings.TrimPrefix(t, "https://")
		}

		if _, ipNet, err := net.ParseCIDR(t); err == nil && ipNet != nil {
			cnt, err := ipv4CIDRHostCount(ipNet)
			if err != nil {
				return r, err
			}
			if cnt > int64(maxCIDRExpandInline) {
				r.LargeNets = append(r.LargeNets, cloneIPNet(ipNet))
				continue
			}
			if int64(len(seen))+cnt > int64(maxTotalExpandedHosts) {
				return r, fmt.Errorf("展开后的目标数量将超过 %d，请减少条目或拆分任务", maxTotalExpandedHosts)
			}
			for _, ip := range expandCIDR(ipNet) {
				if _, ok := seen[ip]; ok {
					continue
				}
				seen[ip] = struct{}{}
				r.Hosts = append(r.Hosts, ip)
			}
			continue
		}

		if idx := strings.Index(t, "/"); idx > 0 {
			t = t[:idx]
		}
		if _, ok := seen[t]; ok {
			continue
		}
		if len(seen) >= maxTotalExpandedHosts {
			return r, fmt.Errorf("目标数量超过上限 %d", maxTotalExpandedHosts)
		}
		seen[t] = struct{}{}
		r.Hosts = append(r.Hosts, t)
	}
	return r, nil
}

func parsePorts(expr string) ([]int, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, nil
	}
	parts := strings.Split(expr, ",")
	portSet := make(map[int]struct{})

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.Contains(p, "-") {
			bounds := strings.SplitN(p, "-", 2)
			if len(bounds) != 2 {
				return nil, fmt.Errorf("端口范围格式错误: %s", p)
			}
			start, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err1 != nil || err2 != nil || start <= 0 || end <= 0 || start > 65535 || end > 65535 || start > end {
				return nil, fmt.Errorf("端口范围非法: %s", p)
			}
			for i := start; i <= end; i++ {
				portSet[i] = struct{}{}
			}
		} else {
			val, err := strconv.Atoi(p)
			if err != nil || val <= 0 || val > 65535 {
				return nil, fmt.Errorf("端口非法: %s", p)
			}
			portSet[val] = struct{}{}
		}
	}

	var ports []int
	for port := range portSet {
		ports = append(ports, port)
	}
	return ports, nil
}

// 将 CIDR 段展开为 IP 列表（仅处理 IPv4）
func expandCIDR(ipNet *net.IPNet) []string {
	var ips []string
	ip := ipNet.IP.To4()
	if ip == nil {
		return ips
	}

	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); incIP(ip) {
		// 复制一份，否则后续 incIP 会修改底层数组
		ipCopy := make(net.IP, len(ip))
		copy(ipCopy, ip)
		ips = append(ips, ipCopy.String())
	}
	return ips
}

// 自增 IPv4 地址
func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

// 根据服务和端口简单推断协议
func detectProtocol(port int, service string) string {
	s := strings.ToLower(service)
	switch {
	case s == "https" || port == 443:
		return "https"
	case strings.Contains(s, "http") || port == 80 || port == 8080 || port == 8000 || port == 8888:
		return "http"
	default:
		return "tcp"
	}
}

// 判断是否为全端口扫描（1-65535）
func isFullPortRange(ports []int) bool {
	if len(ports) != 65535 {
		return false
	}
	if ports[0] != 1 || ports[len(ports)-1] != 65535 {
		return false
	}
	return true
}

// 基于二分顺序构建扫描列表，确保端口覆盖更均匀
func buildBinaryPortOrder(sortedPorts []int) []int {
	if len(sortedPorts) == 0 {
		return nil
	}

	order := make([]int, 0, len(sortedPorts))
	queue := [][]int{sortedPorts}

	for len(queue) > 0 {
		segment := queue[0]
		queue = queue[1:]
		if len(segment) == 0 {
			continue
		}
		mid := len(segment) / 2
		order = append(order, segment[mid])

		// BFS 方式推进，保证不同区间被穿插扫描
		left := segment[:mid]
		right := segment[mid+1:]
		if len(left) > 0 {
			queue = append(queue, left)
		}
		if len(right) > 0 {
			queue = append(queue, right)
		}
	}

	return order
}

// 全端口扫描时：Top1000 先行，其余端口走二分扫描顺序
func buildFullScanOrder(sortedAllPorts []int) []int {
	if len(sortedAllPorts) == 0 {
		return nil
	}

	allSet := make(map[int]struct{}, len(sortedAllPorts))
	for _, p := range sortedAllPorts {
		allSet[p] = struct{}{}
	}

	visited := make(map[int]struct{}, len(sortedAllPorts))
	order := make([]int, 0, len(sortedAllPorts))

	// 先追加 Top1000
	for _, p := range getTop1000Ports() {
		if _, ok := allSet[p]; !ok {
			continue
		}
		if _, seen := visited[p]; seen {
			continue
		}
		order = append(order, p)
		visited[p] = struct{}{}
	}

	// 处理剩余端口
	rest := make([]int, 0, len(sortedAllPorts)-len(order))
	for _, p := range sortedAllPorts {
		if _, seen := visited[p]; seen {
			continue
		}
		rest = append(rest, p)
	}

	order = append(order, buildBinaryPortOrder(rest)...)
	return order
}

// 对 banner 做截断，避免过长
func truncateBanner(banner string) string {
	const maxLen = 512
	banner = strings.ReplaceAll(banner, "\x00", "")
	banner = strings.TrimSpace(banner)
	if len(banner) <= maxLen {
		return banner
	}
	return banner[:maxLen] + "..."
}

// 从 banner / HTML 中提取 <title>
func extractTitleFromBanner(banner string) string {
	if banner == "" {
		return ""
	}
	re := regexp.MustCompile(`(?is)<title>(.*?)</title>`)
	matches := re.FindStringSubmatch(banner)
	if len(matches) < 2 {
		return ""
	}
	title := html.UnescapeString(strings.TrimSpace(matches[1]))
	return title
}

// 根据原始内容和服务名构造更友好的 banner 文本
func buildBannerFromRaw(service, raw string) string {
	if raw == "" {
		if service != "" {
			return service
		}
		return "unknown"
	}

	rawLower := strings.ToLower(raw)
	serviceLower := strings.ToLower(service)

	// 优先从 Server 头提取
	if idx := strings.Index(rawLower, "server:"); idx != -1 {
		line := raw[idx+len("server:"):]
		if p := strings.IndexAny(line, "\r\n"); p != -1 {
			line = line[:p]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	// SMB服务特殊处理
	if serviceLower == "smb" || strings.Contains(rawLower, "smb") {
		if raw != "SMB" && raw != "smb" {
			return truncateBanner(raw)
		}
		return "SMB"
	}

	// MySQL 版本信息（从握手包中提取）
	if serviceLower == "mysql" || strings.Contains(rawLower, "mysql") {
		version := extractMySQLVersion(raw)
		if version != "" {
			return version
		}
		// 如果没有提取到版本，返回包含mysql的原始内容
		if idx := strings.Index(rawLower, "mysql"); idx != -1 {
			line := raw[idx:]
			if p := strings.IndexAny(line, "\x00\r\n"); p != -1 {
				line = line[:p]
			}
			line = strings.TrimSpace(line)
			if line != "" {
				return line
			}
		}
		return "MySQL"
	}

	// Redis版本信息
	if serviceLower == "redis" || strings.Contains(rawLower, "redis") {
		if strings.Contains(rawLower, "redis_version") {
			return extractRedisVersion(raw)
		}
		if raw != "" && raw != "redis" {
			return truncateBanner(raw)
		}
		return "Redis"
	}

	// SSH banner
	if serviceLower == "ssh" || strings.HasPrefix(rawLower, "ssh-") {
		return truncateBanner(raw)
	}

	// FTP banner
	if serviceLower == "ftp" || strings.HasPrefix(raw, "220") {
		return truncateBanner(raw)
	}

	// 特判 Yii
	if strings.Contains(raw, "Yii PHP Framework") {
		return "Yii PHP Framework"
	}

	// 常见 nginx 形式：nginx/1.18.0
	if idx := strings.Index(rawLower, "nginx/"); idx != -1 {
		line := raw[idx:]
		if p := strings.IndexAny(line, "\r\n"); p != -1 {
			line = line[:p]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	// Apache版本
	if idx := strings.Index(rawLower, "apache/"); idx != -1 {
		line := raw[idx:]
		if p := strings.IndexAny(line, "\r\n"); p != -1 {
			line = line[:p]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	// 如果原始内容不是空的，返回原始内容（截断后）
	if raw != "" && raw != service {
		return truncateBanner(raw)
	}

	if service != "" {
		return service
	}
	return "unknown"
}

// extractMySQLVersion 从MySQL握手包中提取版本信息
func extractMySQLVersion(banner string) string {
	// MySQL握手包格式：协议版本(1字节) + 服务器版本(以null结尾的字符串)
	// 查找 "mysql" 关键字后的版本号
	lower := strings.ToLower(banner)
	idx := strings.Index(lower, "mysql")
	if idx == -1 {
		return ""
	}

	// 尝试找到版本号模式 (如 5.7.30, 8.0.21等)
	versionRe := regexp.MustCompile(`(?i)(\d+\.\d+\.\d+)`)
	matches := versionRe.FindStringSubmatch(banner)
	if len(matches) > 1 {
		return "MySQL " + matches[1]
	}

	// 如果没找到标准版本号，返回包含mysql的部分
	part := banner[idx:]
	if p := strings.IndexAny(part, "\x00\r\n"); p != -1 {
		part = part[:p]
	}
	part = strings.TrimSpace(part)
	if part != "" {
		return part
	}

	return ""
}

// extractMySQLVersionFromHandshake 从MySQL握手包提取版本，避免仅按端口误判。
func extractMySQLVersionFromHandshake(buf []byte) string {
	if len(buf) < 8 {
		return ""
	}
	// MySQL握手包第5字节常见为协议版本 0x0a
	if buf[4] != 0x0a {
		return ""
	}
	verStart := 5
	verEnd := -1
	for i := verStart; i < len(buf); i++ {
		if buf[i] == 0x00 {
			verEnd = i
			break
		}
	}
	if verEnd <= verStart {
		return ""
	}
	version := strings.TrimSpace(string(buf[verStart:verEnd]))
	if version == "" {
		return ""
	}
	if regexp.MustCompile(`^\d+\.\d+(\.\d+)?`).MatchString(version) {
		return "MySQL " + version
	}
	if strings.Contains(strings.ToLower(version), "mariadb") {
		return "MariaDB " + version
	}
	return ""
}

// extractRedisVersion 从Redis INFO命令响应中提取版本信息
func extractRedisVersion(info string) string {
	versionRe := regexp.MustCompile(`redis_version:([^\r\n]+)`)
	matches := versionRe.FindStringSubmatch(info)
	if len(matches) > 1 {
		return "Redis " + strings.TrimSpace(matches[1])
	}
	return ""
}

// extractSSHVersion 从SSH banner中提取版本信息
func extractSSHVersion(banner string) string {
	// SSH banner格式：SSH-2.0-OpenSSH_8.2p1 或 SSH-1.99-OpenSSH_3.9p1
	// 提取协议版本和服务版本
	parts := strings.Fields(banner)
	if len(parts) == 0 {
		return ""
	}

	// 第一部分通常是 SSH-2.0-xxx 或 SSH-1.99-xxx
	sshPart := parts[0]
	if !strings.HasPrefix(sshPart, "SSH-") {
		return ""
	}

	// 提取服务信息（如 OpenSSH_8.2p1）
	if len(parts) > 1 {
		serviceInfo := parts[1]
		// 移除常见的版本前缀
		serviceInfo = strings.TrimPrefix(serviceInfo, "OpenSSH_")
		serviceInfo = strings.TrimPrefix(serviceInfo, "Dropbear_")
		serviceInfo = strings.TrimPrefix(serviceInfo, "libssh_")

		// 提取协议版本
		protocolVersion := "2.0"
		if strings.Contains(sshPart, "1.99") {
			protocolVersion = "1.99"
		} else if strings.Contains(sshPart, "1.5") {
			protocolVersion = "1.5"
		}

		if serviceInfo != "" {
			return fmt.Sprintf("SSH-%s %s", protocolVersion, serviceInfo)
		}
		return fmt.Sprintf("SSH-%s", protocolVersion)
	}

	return sshPart
}

// extractSMBInfo 从SMB协议头中提取信息
func extractSMBInfo(buf []byte) string {
	if len(buf) < 4 {
		return "SMB"
	}

	// SMB协议标识：通常是 \x00\x00 后跟 SMB版本标识
	// SMB 1.0/CIFS: \x00\x00\x00...\xffSMB
	// SMB 2.x: \x00\x00\x00\x00...\xfeSMB
	// SMB 3.x: 类似SMB 2.x但版本号不同

	// 查找SMB标识
	for i := 0; i < len(buf)-3; i++ {
		if buf[i] == 'S' && buf[i+1] == 'M' && buf[i+2] == 'B' {
			// 检查是否是SMB 2.x/3.x (前面有0xFE)
			if i > 0 && buf[i-1] == 0xFE {
				// 尝试读取SMB版本（在SMB标识后的字节）
				if i+5 < len(buf) {
					major := buf[i+3]
					minor := buf[i+4]
					// SMB 2.0, 2.1, 3.0, 3.1.1等
					if major == 2 {
						if minor == 0 {
							return "SMB 2.0"
						} else if minor == 1 {
							return "SMB 2.1"
						}
					} else if major == 3 {
						if minor == 0 {
							return "SMB 3.0"
						} else if minor == 2 {
							return "SMB 3.1.1"
						}
					}
					return fmt.Sprintf("SMB %d.%d", major, minor)
				}
				return "SMB 2.x/3.x"
			}
			// SMB 1.0/CIFS
			return "SMB 1.0/CIFS"
		}
	}

	return "SMB"
}

// extractSMTPInfo 从SMTP欢迎消息中提取服务器信息
func extractSMTPInfo(banner string) string {
	// SMTP格式：220 hostname ESMTP server-version
	// 例如：220 mail.example.com ESMTP Postfix 3.4.2
	parts := strings.Fields(banner)
	if len(parts) < 2 {
		return ""
	}

	// 提取服务器名称和版本信息
	var result strings.Builder
	for i, part := range parts {
		if i == 0 {
			continue // 跳过状态码220
		}
		if i == 1 {
			result.WriteString(part) // 主机名
		} else {
			result.WriteString(" ")
			result.WriteString(part)
		}
		// 如果遇到版本号模式，保留更多信息
		if strings.Contains(part, ".") && len(part) < 20 {
			// 可能是版本号，继续读取
		}
	}

	return truncateBanner(result.String())
}

// extractPostgreSQLVersion 从PostgreSQL握手包中提取版本信息
func extractPostgreSQLVersion(banner string) string {
	// PostgreSQL握手包是二进制格式，但可能包含可读的版本字符串
	// 查找版本号模式
	versionRe := regexp.MustCompile(`(?i)(\d+\.\d+(?:\.\d+)?)`)
	matches := versionRe.FindStringSubmatch(banner)
	if len(matches) > 1 {
		return "PostgreSQL " + matches[1]
	}

	// 查找PostgreSQL关键字
	if strings.Contains(strings.ToLower(banner), "postgresql") {
		// 尝试提取版本
		parts := strings.Fields(banner)
		for _, part := range parts {
			if versionRe.MatchString(part) {
				return "PostgreSQL " + part
			}
		}
	}

	return ""
}

// extractMSSQLInfo 从SQL Server握手包中提取信息
func extractMSSQLInfo(banner string) string {
	// SQL Server握手包是二进制格式，但可能包含版本信息
	// 查找版本号模式
	versionRe := regexp.MustCompile(`(?i)(\d+\.\d+(?:\.\d+)?\.\d+)`)
	matches := versionRe.FindStringSubmatch(banner)
	if len(matches) > 1 {
		return "Microsoft SQL Server " + matches[1]
	}

	// 查找SQL Server关键字
	lower := strings.ToLower(banner)
	if strings.Contains(lower, "sql server") || strings.Contains(lower, "microsoft sql") {
		// 尝试提取版本
		parts := strings.Fields(banner)
		for _, part := range parts {
			if versionRe.MatchString(part) {
				return "Microsoft SQL Server " + part
			}
		}
		return "Microsoft SQL Server"
	}

	return ""
}

// extractMongoDBVersion 从MongoDB握手包中提取版本信息
func extractMongoDBVersion(banner string) string {
	// MongoDB握手包可能包含版本信息
	// 查找版本号模式
	versionRe := regexp.MustCompile(`(?i)(\d+\.\d+\.\d+)`)
	matches := versionRe.FindStringSubmatch(banner)
	if len(matches) > 1 {
		return "MongoDB " + matches[1]
	}

	// 查找MongoDB关键字
	if strings.Contains(strings.ToLower(banner), "mongodb") {
		parts := strings.Fields(banner)
		for _, part := range parts {
			if versionRe.MatchString(part) {
				return "MongoDB " + part
			}
		}
		return "MongoDB"
	}

	return ""
}

// HTTPBannerResult HTTP banner探测结果
type HTTPBannerResult struct {
	Service        string
	Raw            string
	RequestPacket  string
	ResponsePacket string
	StatusCode     int // HTTP状态码
}

// 对 Web 端口发起一次 HTTP 请求，返回服务类型和原始内容（头+部分 body）
// 优先尝试HTTP，如果是443端口则尝试HTTPS
// 优化：使用httpx管理器的客户端配置，提高连接复用
func probeHTTPBanner(host string, port int, timeout time.Duration, base string) HTTPBannerResult {
	result := HTTPBannerResult{
		Service: base,
		Raw:     "",
	}

	// 获取HTTP客户端（已优化连接池配置）
	client := proxy.GlobalProxyManager.GetHTTPClient()

	// 如果是443端口，优先尝试HTTPS
	if port == 443 || base == "https" {
		result = tryHTTPRequest(host, port, "https", timeout, client)
		if result.ResponsePacket != "" {
			return result
		}
	}

	// 尝试HTTP（对于所有端口，包括443如果HTTPS失败）
	result = tryHTTPRequest(host, port, "http", timeout, client)
	return result
}

// tryHTTPRequest 尝试发送HTTP/HTTPS请求
func tryHTTPRequest(host string, port int, scheme string, timeout time.Duration, client *http.Client) HTTPBannerResult {
	result := HTTPBannerResult{
		Service: scheme,
		Raw:     "",
	}

	hostPort := host
	if (scheme == "http" && port != 80) || (scheme == "https" && port != 443) {
		hostPort = net.JoinHostPort(host, strconv.Itoa(port))
	}

	url := fmt.Sprintf("%s://%s/", scheme, hostPort)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return result
	}

	// 设置User-Agent和Accept头，确保请求更标准
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")

	// 构建请求包字符串
	var reqBuilder strings.Builder
	reqBuilder.WriteString(fmt.Sprintf("%s %s HTTP/1.1\r\n", req.Method, req.URL.Path))
	reqBuilder.WriteString(fmt.Sprintf("Host: %s\r\n", hostPort))
	for k, v := range req.Header {
		for _, val := range v {
			reqBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
		}
	}
	reqBuilder.WriteString("\r\n")
	result.RequestPacket = reqBuilder.String()

	resp, err := client.Do(req)
	if err != nil {
		return result
	}
	defer resp.Body.Close()

	// 构建响应包字符串
	var respBuilder strings.Builder
	respBuilder.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", resp.StatusCode, resp.Status))
	for k, v := range resp.Header {
		for _, val := range v {
			respBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
		}
	}
	respBuilder.WriteString("\r\n")

	var sb strings.Builder
	// 记录 Server 和 X-Powered-By 头
	if server := resp.Header.Get("Server"); server != "" {
		sb.WriteString("Server: ")
		sb.WriteString(server)
		sb.WriteString("\r\n")
	}
	if xp := resp.Header.Get("X-Powered-By"); xp != "" {
		sb.WriteString("X-Powered-By: ")
		sb.WriteString(xp)
		sb.WriteString("\r\n")
	}

	// 读取部分 body 用于提取标题或关键字
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err == nil && len(body) > 0 {
		sb.WriteString("\r\n")
		sb.Write(body)
		respBuilder.Write(body)
	}

	result.Raw = truncateBanner(sb.String())
	result.ResponsePacket = respBuilder.String()
	result.StatusCode = resp.StatusCode

	// 根据 Server 头粗略识别服务
	serverHeader := strings.ToLower(resp.Header.Get("Server"))
	switch {
	case strings.Contains(serverHeader, "nginx"):
		result.Service = "nginx"
	case strings.Contains(serverHeader, "apache"):
		result.Service = "apache"
	case strings.Contains(serverHeader, "iis"):
		result.Service = "microsoft-iis"
	}

	return result
}

// PortFingerprintResult 端口指纹识别结果
type PortFingerprintResult struct {
	Technologies []string
	CMS          string
	Title        string
}

// performPortFingerprint 对单个端口进行指纹识别
func performPortFingerprint(host string, port int, protocol string, timeout time.Duration, responsePacket string) *PortFingerprintResult {
	result := &PortFingerprintResult{
		Technologies: []string{},
	}

	// 构建URL
	scheme := protocol
	if scheme == "" {
		if port == 443 {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	hostPort := host
	if (scheme == "http" && port != 80) || (scheme == "https" && port != 443) {
		hostPort = net.JoinHostPort(host, strconv.Itoa(port))
	}

	url := fmt.Sprintf("%s://%s/", scheme, hostPort)

	// 创建HTTP客户端
	client := proxy.GlobalProxyManager.GetHTTPClient()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil
	}

	bodyString := string(body)
	headerText := headerToString(resp.Header)

	// 提取标题
	result.Title = extractTitleFromBanner(bodyString)

	// 简单的指纹识别：从响应头和响应体中提取技术栈
	// Server头
	if server := resp.Header.Get("Server"); server != "" {
		serverLower := strings.ToLower(server)
		if strings.Contains(serverLower, "nginx") {
			result.Technologies = append(result.Technologies, "Nginx")
		} else if strings.Contains(serverLower, "apache") {
			result.Technologies = append(result.Technologies, "Apache")
		} else if strings.Contains(serverLower, "iis") {
			result.Technologies = append(result.Technologies, "IIS")
		} else if strings.Contains(serverLower, "tomcat") {
			result.Technologies = append(result.Technologies, "Tomcat")
		} else {
			result.Technologies = append(result.Technologies, server)
		}
	}

	// X-Powered-By头
	if poweredBy := resp.Header.Get("X-Powered-By"); poweredBy != "" {
		result.Technologies = append(result.Technologies, poweredBy)
	}

	// 从响应体中提取常见技术栈
	bodyLower := strings.ToLower(bodyString)
	headerLower := strings.ToLower(headerText)
	combined := bodyLower + " " + headerLower

	// 常见技术栈关键词
	techKeywords := map[string]string{
		"spring": "Spring", "springboot": "SpringBoot", "spring-boot": "SpringBoot",
		"vue": "Vue", "react": "React", "angular": "Angular",
		"jquery": "jQuery", "bootstrap": "Bootstrap",
		"wordpress": "WordPress", "drupal": "Drupal", "joomla": "Joomla",
		"php": "PHP", "laravel": "Laravel", "symfony": "Symfony",
		"python": "Python", "django": "Django", "flask": "Flask",
		"node": "Node.js", "express": "Express", "koa": "Koa",
		"java": "Java", "jsp": "JSP",
		".net": ".NET", "asp.net": "ASP.NET",
		"ruby": "Ruby", "rails": "Rails",
		"go": "Go", "golang": "Go",
		"docker": "Docker", "kubernetes": "Kubernetes",
		"jenkins": "Jenkins", "gitlab": "GitLab",
		"vmware": "VMware", "vsphere": "vSphere",
	}

	for keyword, techName := range techKeywords {
		if strings.Contains(combined, keyword) {
			// 避免重复
			found := false
			for _, t := range result.Technologies {
				if t == techName {
					found = true
					break
				}
			}
			if !found {
				result.Technologies = append(result.Technologies, techName)
			}
		}
	}

	// 尝试识别CMS
	if strings.Contains(combined, "wordpress") {
		result.CMS = "WordPress"
	} else if strings.Contains(combined, "drupal") {
		result.CMS = "Drupal"
	} else if strings.Contains(combined, "joomla") {
		result.CMS = "Joomla"
	} else if strings.Contains(combined, "discuz") {
		result.CMS = "Discuz"
	} else if strings.Contains(combined, "phpcms") {
		result.CMS = "PHPCMS"
	} else if strings.Contains(combined, "dedecms") {
		result.CMS = "DedeCMS"
	}

	return result
}

// headerToString 将HTTP头转换为字符串
func headerToString(header http.Header) string {
	var sb strings.Builder
	for k, v := range header {
		for _, val := range v {
			sb.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
		}
	}
	return sb.String()
}

// 获取 IP 归属地（带缓存，避免重复查询）
func (s *HostScanner) getLocationForHost(host string) string {
	// 目前仅对直接输入 IP 的目标做归属地查询
	if net.ParseIP(host) == nil {
		return ""
	}

	if v, ok := s.locationCache.Load(host); ok {
		if str, ok2 := v.(string); ok2 {
			return str
		}
	}

	location, err := queryIPLocationForScanner(host)
	if err != nil {
		location = ""
	}

	if location != "" {
		s.locationCache.Store(host, location)
	}
	return location
}

// 端口扫描内部使用的简化 IP 归属地查询
func queryIPLocationForScanner(ip string) (string, error) {
	client := proxy.GlobalProxyManager.GetHTTPClient()
	url := fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN", ip)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var body struct {
		Status     string `json:"status"`
		Message    string `json:"message"`
		Country    string `json:"country"`
		RegionName string `json:"regionName"`
		City       string `json:"city"`
		ISP        string `json:"isp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}

	if body.Status != "success" {
		if body.Message != "" {
			return "", fmt.Errorf("查询失败: %s", body.Message)
		}
		return "", fmt.Errorf("查询失败")
	}

	var parts []string
	if body.Country != "" {
		parts = append(parts, body.Country)
	}
	if body.RegionName != "" {
		parts = append(parts, body.RegionName)
	}
	if body.City != "" {
		parts = append(parts, body.City)
	}
	if body.ISP != "" {
		parts = append(parts, body.ISP)
	}

	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, ", "), nil
}

// getPresetPorts 获取预设端口列表
func getPresetPorts(preset string) []int {
	switch preset {
	case "top100":
		return getTop100Ports()
	case "top1000":
		return getTop1000Ports()
	case "noPrinter":
		return getNoPrinterPorts()
	default:
		return []int{}
	}
}

// getTop100Ports 返回最常见的100个端口
func getTop100Ports() []int {
	return []int{
		21, 22, 23, 25, 53, 80, 110, 111, 135, 139,
		143, 443, 445, 993, 995, 1723, 3306, 3389, 5900, 8080,
		8443, 8888, 9200, 27017, 6379, 7001, 8000, 8001, 8081, 9000,
		9090, 1433, 1521, 5432, 27017, 11211, 5000, 5001, 5984, 5985,
		5986, 5672, 15672, 8161, 8162, 8163, 8164, 8165, 8166, 8167,
		8168, 8169, 8170, 8171, 8172, 8173, 8174, 8175, 8176, 8177,
		8178, 8179, 8180, 8181, 8182, 8183, 8184, 8185, 8186, 8187,
		8188, 8189, 8190, 8191, 8192, 8193, 8194, 8195, 8196, 8197,
		8198, 8199, 8200, 8201, 8202, 8203, 8204, 8205, 8206, 8207,
		8208, 8209, 8210, 8211, 8212, 8213, 8214, 8215, 8216, 8217,
	}
}

// getTop1000Ports 返回最常见的1000个端口
func getTop1000Ports() []int {
	// 基于nmap-services和实际扫描经验的最常见端口
	commonPorts := []int{
		// 最常用端口
		21, 22, 23, 25, 53, 80, 110, 111, 135, 139,
		143, 443, 445, 993, 995, 1723, 3306, 3389, 5900, 8080,
		8443, 8888, 9200, 27017, 6379, 7001, 8000, 8001, 8081, 9000,
		9090, 1433, 1521, 5432, 11211, 5000, 5001, 5984, 5985, 5986,
		5672, 15672, 8161, 8162, 8163, 8164, 8165, 8166, 8167, 8168,
		// 扩展常见端口
		20, 69, 79, 88, 102, 110, 119, 123, 135, 139,
		143, 161, 162, 179, 389, 443, 445, 465, 514, 515,
		587, 636, 993, 995, 1080, 1433, 1521, 1723, 3306, 3389,
		5432, 5900, 6379, 7001, 8000, 8001, 8080, 8081, 8443, 8888,
		9000, 9090, 9200, 11211, 27017,
		// 1-1000范围内的其他端口
	}

	portSet := make(map[int]struct{})
	for _, p := range commonPorts {
		portSet[p] = struct{}{}
	}

	// 添加1-1000范围内的其他端口
	for i := 1; i <= 1000; i++ {
		if _, exists := portSet[i]; !exists {
			portSet[i] = struct{}{}
		}
	}

	ports := make([]int, 0, len(portSet))
	for p := range portSet {
		ports = append(ports, p)
	}
	return ports
}

// getNoPrinterPorts 返回过滤掉打印机端口的常见端口列表
func getNoPrinterPorts() []int {
	// 打印机常见端口
	printerPorts := map[int]bool{
		515:  true, // LPD
		631:  true, // IPP
		9100: true, // Raw printing
		9101: true,
		9102: true,
		9103: true,
		9104: true,
		9105: true,
		9106: true,
		9107: true,
		9108: true,
		9109: true,
	}

	allPorts := getTop100Ports()
	filtered := make([]int, 0, len(allPorts))
	for _, port := range allPorts {
		if !printerPorts[port] {
			filtered = append(filtered, port)
		}
	}
	return filtered
}

// filterAliveTargets 过滤存活的 IP 目标；大网段从 LargeNets 流式投递，不展开整表
func (s *HostScanner) filterAliveTargets(r resolvedScanTargets) ([]string, int, int) {
	aliveTargets := make([]string, 0)
	var totalIPs int
	var domainCount int

	ipList := make([]string, 0)
	for _, target := range r.Hosts {
		select {
		case <-s.ctx.Done():
			s.aliveResultMu.Lock()
			s.aliveResult = &AliveCheckResult{
				TotalIPs:    totalIPs,
				AliveIPs:    0,
				DeadIPs:     totalIPs,
				DomainCount: domainCount,
			}
			s.aliveResultMu.Unlock()
			return aliveTargets, 0, totalIPs
		default:
		}

		if net.ParseIP(target) == nil {
			aliveTargets = append(aliveTargets, target)
			domainCount++
			continue
		}

		ipList = append(ipList, target)
		totalIPs++
	}

	for _, nn := range r.LargeNets {
		c, err := ipv4CIDRHostCount(nn)
		if err != nil {
			continue
		}
		totalIPs += int(c)
	}

	if totalIPs == 0 {
		s.aliveResultMu.Lock()
		s.aliveResult = &AliveCheckResult{
			TotalIPs:    0,
			AliveIPs:    0,
			DeadIPs:     0,
			DomainCount: domainCount,
		}
		s.aliveResultMu.Unlock()
		return aliveTargets, 0, 0
	}

	atomic.StoreInt64(&s.aliveProbeDone, 0)
	atomic.StoreInt64(&s.aliveProbeTotal, int64(totalIPs))

	aliveChan := make(chan string, 1024)
	jobs := make(chan string, 1024)

	go func() {
		defer close(jobs)
		for _, t := range ipList {
			select {
			case <-s.ctx.Done():
				return
			case jobs <- t:
			}
		}
		for _, nn := range r.LargeNets {
			ip4 := nn.IP.To4()
			if ip4 == nil {
				continue
			}
			for cur := ip4.Mask(nn.Mask); nn.Contains(cur); incIP(cur) {
				select {
				case <-s.ctx.Done():
					return
				default:
				}
				ipCopy := make(net.IP, len(cur))
				copy(ipCopy, cur)
				select {
				case <-s.ctx.Done():
					return
				case jobs <- ipCopy.String():
				}
			}
		}
	}()

	workerCount := maxAliveCheckWorkers
	if totalIPs < workerCount {
		workerCount = totalIPs
	}
	if workerCount < 1 {
		workerCount = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range jobs {
				select {
				case <-s.ctx.Done():
					return
				default:
				}
				if s.isHostAlive(host) {
					select {
					case aliveChan <- host:
					case <-s.ctx.Done():
					}
				}
				atomic.AddInt64(&s.aliveProbeDone, 1)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(aliveChan)
	}()

	aliveIPCount := 0
	for {
		select {
		case <-s.ctx.Done():
			s.aliveResultMu.Lock()
			s.aliveResult = &AliveCheckResult{
				TotalIPs:    totalIPs,
				AliveIPs:    aliveIPCount,
				DeadIPs:     totalIPs - aliveIPCount,
				DomainCount: domainCount,
			}
			s.aliveResultMu.Unlock()
			return aliveTargets, aliveIPCount, totalIPs
		case host, ok := <-aliveChan:
			if !ok {
				s.aliveResultMu.Lock()
				s.aliveResult = &AliveCheckResult{
					TotalIPs:    totalIPs,
					AliveIPs:    aliveIPCount,
					DeadIPs:     totalIPs - aliveIPCount,
					DomainCount: domainCount,
				}
				s.aliveResultMu.Unlock()
				return aliveTargets, aliveIPCount, totalIPs
			}
			aliveTargets = append(aliveTargets, host)
			aliveIPCount++
		}
	}
}

// GetAliveCheckResult 获取存活检测结果
func (s *HostScanner) GetAliveCheckResult() *AliveCheckResult {
	s.aliveResultMu.Lock()
	defer s.aliveResultMu.Unlock()
	return s.aliveResult
}

// isHostAlive 检测主机是否存活（支持多种检测方式）
func (s *HostScanner) isHostAlive(host string) bool {
	// 如果配置了自定义存活检测，使用自定义配置
	if s.config.AliveCheck != nil && s.config.AliveCheck.Enabled {
		return s.checkAliveWithConfig(host, s.config.AliveCheck)
	}

	// 默认使用常见端口检测
	// 注意：使用更短的超时时间进行存活检测，避免等待太久
	aliveTimeout := time.Duration(s.config.Timeout) * time.Second
	if aliveTimeout > 2*time.Second {
		aliveTimeout = 2 * time.Second // 存活检测最多2秒
	}

	commonPorts := []int{80, 443, 22, 135, 445, 3389}

	// 并发检测多个端口，只要有一个端口能连接就认为存活
	type portResult struct {
		alive bool
	}
	resultChan := make(chan portResult, len(commonPorts))

	// 启动所有goroutine
	for _, port := range commonPorts {
		select {
		case <-s.ctx.Done():
			return false
		default:
		}

		go func(p int) {
			address := net.JoinHostPort(host, strconv.Itoa(p))
			conn, err := s.dialTCP("tcp", address, aliveTimeout)
			if err == nil {
				conn.Close()
				select {
				case resultChan <- portResult{alive: true}:
				case <-s.ctx.Done():
				}
			} else {
				select {
				case resultChan <- portResult{alive: false}:
				case <-s.ctx.Done():
				}
			}
		}(port)
	}

	// 等待所有检测完成，只要有一个成功就立即返回true
	for i := 0; i < len(commonPorts); i++ {
		select {
		case <-s.ctx.Done():
			return false
		case result := <-resultChan:
			if result.alive {
				// 如果有一个端口成功，立即返回，不等待其他结果
				return true
			}
		}
	}

	return false
}

// checkAliveWithConfig 根据配置检测主机存活
func (s *HostScanner) checkAliveWithConfig(host string, config *AliveCheckConfig) bool {
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(s.config.Timeout) * time.Second
	}

	switch config.Type {
	case "icmp":
		return s.checkAliveICMP(host, timeout)
	case "tcp_port":
		if len(config.Ports) == 0 {
			// 如果没有指定端口，使用默认端口
			config.Ports = []int{80, 443, 22}
		}
		return s.checkAliveTCP(host, config.Ports, timeout)
	case "http":
		url := config.URL
		if url == "" {
			url = fmt.Sprintf("http://%s/", host)
		}
		return s.checkAliveHTTP(url, timeout)
	default:
		// 默认使用TCP端口检测
		ports := config.Ports
		if len(ports) == 0 {
			ports = []int{80, 443, 22}
		}
		return s.checkAliveTCP(host, ports, timeout)
	}
}

// checkAliveTCP 使用TCP端口检测主机存活
func (s *HostScanner) checkAliveTCP(host string, ports []int, timeout time.Duration) bool {
	for _, port := range ports {
		select {
		case <-s.ctx.Done():
			return false
		default:
		}

		address := net.JoinHostPort(host, strconv.Itoa(port))
		conn, err := s.dialTCP("tcp", address, timeout)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

// checkAliveHTTP 使用HTTP请求检测主机存活
func (s *HostScanner) checkAliveHTTP(url string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := proxy.GlobalProxyManager.GetHTTPClient()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 只要返回了响应就认为存活
	return resp.StatusCode > 0
}
