//go:build darwin

package hostscan

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// checkAliveICMP 使用ICMP检测主机存活
func (s *HostScanner) checkAliveICMP(host string, timeout time.Duration) bool {
	// 使用系统ping命令实现ICMP检测
	ctx, cancel := context.WithTimeout(context.Background(), timeout+time.Second)
	defer cancel()

	// macOS: ping -c 1 -W timeout_ms host
	// -c 1: 只发送1个包
	// -W timeout_ms: 超时时间（毫秒，macOS 的 -W 参数单位是毫秒）
	timeoutMs := int(timeout.Milliseconds())
	if timeoutMs < 1000 {
		timeoutMs = 1000
	}
	if timeoutMs > 60000 {
		timeoutMs = 60000 // 最大60秒
	}
	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", strconv.Itoa(timeoutMs), host)

	// 捕获输出以检查ping是否成功
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// 命令执行失败，主机不存活
		return false
	}

	// 检查输出中是否包含成功标识
	output := strings.ToLower(stdout.String())
	// macOS ping成功时通常包含 "1 received" 或 "1 packets transmitted, 1 received"
	return strings.Contains(output, "1 received") ||
		strings.Contains(output, "1 packets transmitted, 1 received") ||
		strings.Contains(output, "time=") ||
		strings.Contains(output, "bytes from")
}
