package crack

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ExecuteWMICommand 精简版 WMI 命令执行（仅 Windows，参考 MPET-main 的 wmic 方式）
//
// 注意：这不是网络隧道协议，不支持 SSH 隧道；它依赖本机执行 wmic 并远程查询。
// command 当前只支持：
// - nic: 读取网卡信息（wmic nic get）
func ExecuteWMICommand(host, username, password string, command string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", fmt.Errorf("当前系统不支持 WMI（仅支持 Windows 环境执行 wmic）")
	}

	cmd := strings.TrimSpace(strings.ToLower(command))
	if cmd == "" {
		return "", fmt.Errorf("命令为空")
	}

	if cmd != "nic" {
		return "", fmt.Errorf("不支持的命令: %s\n支持的命令: nic", command)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	args := []string{}
	if strings.TrimSpace(host) != "" {
		args = append(args, "/node:"+strings.TrimSpace(host))
	}
	if strings.TrimSpace(username) != "" {
		args = append(args, "/user:"+strings.TrimSpace(username))
		if strings.TrimSpace(password) != "" {
			args = append(args, "/password:"+strings.TrimSpace(password))
		}
	}
	args = append(args, "nic", "get")

	c := exec.CommandContext(ctx, "wmic", args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	if err := c.Run(); err != nil {
		errMsg := fmt.Sprintf("wmic 执行失败: %v", err)
		if stderr.Len() > 0 {
			errMsg = fmt.Sprintf("%s（%s）", errMsg, strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		out = "命令执行成功，但未返回任何内容"
	}
	return out, nil
}

