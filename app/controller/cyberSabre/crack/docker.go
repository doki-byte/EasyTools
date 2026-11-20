package crack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// DockerVersion Docker版本信息
type DockerVersion struct {
	Version       string `json:"Version"`
	ApiVersion    string `json:"ApiVersion"`
	MinAPIVersion string `json:"MinAPIVersion"`
	GitCommit     string `json:"GitCommit"`
	GoVersion     string `json:"GoVersion"`
	Os            string `json:"Os"`
	Arch          string `json:"Arch"`
	KernelVersion string `json:"KernelVersion"`
	BuildTime     string `json:"BuildTime"`
}

// DockerInfo Docker系统信息
type DockerInfo struct {
	ID                string `json:"ID"`
	Containers        int    `json:"Containers"`
	ContainersRunning int    `json:"ContainersRunning"`
	ContainersPaused  int    `json:"ContainersPaused"`
	ContainersStopped int    `json:"ContainersStopped"`
	Images            int    `json:"Images"`
	Driver            string `json:"Driver"`
	ServerVersion     string `json:"ServerVersion"`
	OperatingSystem   string `json:"OperatingSystem"`
	Architecture      string `json:"Architecture"`
	NCPU              int    `json:"NCPU"`
	MemTotal          int64  `json:"MemTotal"`
	Name              string `json:"Name"`
	KernelVersion     string `json:"KernelVersion"`
}

// DockerContainer Docker容器信息
type DockerContainer struct {
	Id     string   `json:"Id"`
	Names  []string `json:"Names"`
	Image  string   `json:"Image"`
	State  string   `json:"State"`
	Status string   `json:"Status"`
}

// DockerImage Docker镜像信息
type DockerImage struct {
	Id          string   `json:"Id"`
	RepoTags    []string `json:"RepoTags"`
	RepoDigests []string `json:"RepoDigests"`
	Created     int64    `json:"Created"`
	Size        int64    `json:"Size"`
	VirtualSize int64    `json:"VirtualSize"`
}

// createDockerHTTPClient 创建 Docker HTTP 客户端（支持SSH隧道）
func createDockerHTTPClient(sshConfig *SSHConfig) *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialWithSSHContext(ctx, sshConfig, network, addr, 10*time.Second)
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

// getDockerVersion 获取 Docker 版本信息
func getDockerVersion(client *http.Client, baseURL string) (*DockerVersion, error) {
	resp, err := client.Get(baseURL + "/version")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var version DockerVersion
	if err := json.Unmarshal(body, &version); err != nil {
		return nil, err
	}

	return &version, nil
}

// getDockerInfo 获取 Docker 系统信息
func getDockerInfo(client *http.Client, baseURL string) (*DockerInfo, error) {
	resp, err := client.Get(baseURL + "/info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info DockerInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// getDockerContainers 获取 Docker 容器列表
func getDockerContainers(client *http.Client, baseURL string) ([]DockerContainer, error) {
	resp, err := client.Get(baseURL + "/containers/json?all=true")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var containers []DockerContainer
	if err := json.Unmarshal(body, &containers); err != nil {
		return nil, err
	}

	return containers, nil
}

// getDockerImages 获取 Docker 镜像列表
func getDockerImages(client *http.Client, baseURL string) ([]DockerImage, error) {
	resp, err := client.Get(baseURL + "/images/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var images []DockerImage
	if err := json.Unmarshal(body, &images); err != nil {
		return nil, err
	}

	return images, nil
}

// ExecuteDockerCommand 执行 Docker 命令（支持SSH隧道）
func ExecuteDockerCommand(host, port string, sshConfig *SSHConfig, command string) (string, error) {
	addr := net.JoinHostPort(host, port)

	// 确定使用的协议
	scheme := "http"
	if port == "2376" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, addr)

	client := createDockerHTTPClient(sshConfig)

	// 解析命令
	command = strings.TrimSpace(command)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("命令为空")
	}

	// 支持的命令
	switch parts[0] {
	case "ps", "containers":
		// 列出容器
		containers, err := getDockerContainers(client, baseURL)
		if err != nil {
			return "", err
		}
		return formatContainersList(containers), nil

	case "images":
		// 列出镜像
		images, err := getDockerImages(client, baseURL)
		if err != nil {
			return "", err
		}
		return formatImagesList(images), nil

	case "info":
		// 系统信息
		info, err := getDockerInfo(client, baseURL)
		if err != nil {
			return "", err
		}
		return formatDockerInfo(info), nil

	case "version":
		// 版本信息
		version, err := getDockerVersion(client, baseURL)
		if err != nil {
			return "", err
		}
		return formatDockerVersion(version), nil

	default:
		return "", fmt.Errorf("不支持的命令: %s\n支持的命令: ps, containers, images, info, version", parts[0])
	}
}

// formatContainersList 格式化容器列表
func formatContainersList(containers []DockerContainer) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("容器列表 (共 %d 个):\n\n", len(containers)))

	for i, container := range containers {
		name := "未命名"
		if len(container.Names) > 0 {
			name = strings.TrimPrefix(container.Names[0], "/")
		}
		result.WriteString(fmt.Sprintf("[%d] %s\n", i+1, name))
		result.WriteString(fmt.Sprintf("    ID: %s\n", container.Id[:12]))
		result.WriteString(fmt.Sprintf("    镜像: %s\n", container.Image))
		result.WriteString(fmt.Sprintf("    状态: %s (%s)\n", container.State, container.Status))
		result.WriteString("\n")
	}

	return result.String()
}

// formatImagesList 格式化镜像列表
func formatImagesList(images []DockerImage) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("镜像列表 (共 %d 个):\n\n", len(images)))

	for i, image := range images {
		tags := "无标签"
		if len(image.RepoTags) > 0 && image.RepoTags[0] != "<none>:<none>" {
			tags = strings.Join(image.RepoTags, ", ")
		}
		result.WriteString(fmt.Sprintf("[%d] %s\n", i+1, tags))
		result.WriteString(fmt.Sprintf("    ID: %s\n", image.Id[:19]))
		result.WriteString(fmt.Sprintf("    大小: %.2f MB\n", float64(image.Size)/(1024*1024)))
		result.WriteString(fmt.Sprintf("    创建时间: %s\n", time.Unix(image.Created, 0).Format("2006-01-02 15:04:05")))
		result.WriteString("\n")
	}

	return result.String()
}

// formatDockerInfo 格式化 Docker 系统信息
func formatDockerInfo(info *DockerInfo) string {
	var result strings.Builder
	result.WriteString("Docker 系统信息:\n\n")
	result.WriteString(fmt.Sprintf("主机名: %s\n", info.Name))
	result.WriteString(fmt.Sprintf("操作系统: %s\n", info.OperatingSystem))
	result.WriteString(fmt.Sprintf("架构: %s\n", info.Architecture))
	result.WriteString(fmt.Sprintf("CPU 核心数: %d\n", info.NCPU))
	result.WriteString(fmt.Sprintf("内存: %.2f GB\n", float64(info.MemTotal)/(1024*1024*1024)))
	result.WriteString(fmt.Sprintf("存储驱动: %s\n", info.Driver))
	result.WriteString(fmt.Sprintf("容器总数: %d (运行: %d, 暂停: %d, 停止: %d)\n",
		info.Containers, info.ContainersRunning, info.ContainersPaused, info.ContainersStopped))
	result.WriteString(fmt.Sprintf("镜像数: %d\n", info.Images))
	return result.String()
}

// formatDockerVersion 格式化 Docker 版本信息
func formatDockerVersion(version *DockerVersion) string {
	var result strings.Builder
	result.WriteString("Docker 版本信息:\n\n")
	result.WriteString(fmt.Sprintf("版本: %s\n", version.Version))
	result.WriteString(fmt.Sprintf("API 版本: %s\n", version.ApiVersion))
	result.WriteString(fmt.Sprintf("Go 版本: %s\n", version.GoVersion))
	result.WriteString(fmt.Sprintf("系统: %s/%s\n", version.Os, version.Arch))
	if version.KernelVersion != "" {
		result.WriteString(fmt.Sprintf("内核版本: %s\n", version.KernelVersion))
	}
	return result.String()
}
