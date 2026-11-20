package system

import (
	"fmt"
	"net"
	"sync"
)

// PortManagerConfig 端口管理器
type PortManagerConfig struct {
	usedPorts map[int]bool
	mutex     sync.RWMutex
}

var PortManager = &PortManagerConfig{
	usedPorts: make(map[int]bool),
}

// GetAvailablePort 获取一个可用的端口（从指定端口开始递增）
func (pm *PortManagerConfig) GetAvailablePort(startPort int) (int, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for port := startPort; port < 65535; port++ {
		if pm.usedPorts[port] {
			continue
		}

		if isPortAvailable(port) {
			pm.usedPorts[port] = true
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found")
}

// GetAvailablePortInRange 在指定范围内获取可用端口
func (pm *PortManagerConfig) GetAvailablePortInRange(minPort, maxPort int) (int, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for port := minPort; port <= maxPort; port++ {
		if pm.usedPorts[port] {
			continue
		}

		if isPortAvailable(port) {
			pm.usedPorts[port] = true
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found in range %d-%d", minPort, maxPort)
}

// ReleasePort 释放端口
func (pm *PortManagerConfig) ReleasePort(port int) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	delete(pm.usedPorts, port)
}

// isPortAvailable 检查端口是否可用
func isPortAvailable(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// PortConfig 端口配置结构体
type PortConfig struct {
	MainPort      int `json:"mainPort"`
	SSHPort       int `json:"sshPort"`
	FTPPort       int `json:"ftpPort"`
	FuzzPort      int `json:"fuzzPort"`
	DBManagerPort int `json:"dbManagerPort"`
}
