package crack

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// K8sVersion Kubernetes版本信息
type K8sVersion struct {
	Major        string `json:"major"`
	Minor        string `json:"minor"`
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

// K8sPod Pod信息
type K8sPod struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Status struct {
		Phase string `json:"phase"`
	} `json:"status"`
	Spec struct {
		Containers []struct {
			Name  string `json:"name"`
			Image string `json:"image"`
		} `json:"containers"`
	} `json:"spec"`
}

// K8sPodList Pod列表
type K8sPodList struct {
	Items []K8sPod `json:"items"`
}

// K8sNamespace 命名空间信息
type K8sNamespace struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
}

// K8sNamespaceList 命名空间列表
type K8sNamespaceList struct {
	Items []K8sNamespace `json:"items"`
}

// K8sSecret Secret信息
type K8sSecret struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Type string            `json:"type"`
	Data map[string]string `json:"data"`
}

// K8sSecretList Secret列表
type K8sSecretList struct {
	Items []K8sSecret `json:"items"`
}

// createK8sHTTPClient 创建 Kubernetes HTTP 客户端（支持SSH隧道）
func createK8sHTTPClient(sshConfig *SSHConfig) *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialWithSSHContext(ctx, sshConfig, network, addr, 10*time.Second)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

// getK8sVersion 获取 Kubernetes 版本信息
func getK8sVersion(client *http.Client, baseURL string) (*K8sVersion, error) {
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

	var version K8sVersion
	if err := json.Unmarshal(body, &version); err != nil {
		return nil, err
	}

	return &version, nil
}

// getK8sNamespaces 获取命名空间列表
func getK8sNamespaces(client *http.Client, baseURL string) ([]string, error) {
	resp, err := client.Get(baseURL + "/api/v1/namespaces")
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

	var nsList K8sNamespaceList
	if err := json.Unmarshal(body, &nsList); err != nil {
		return nil, err
	}

	namespaces := make([]string, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, ns.Metadata.Name)
	}

	return namespaces, nil
}

// getK8sPods 获取 Pod 列表
func getK8sPods(client *http.Client, baseURL string) ([]K8sPod, error) {
	resp, err := client.Get(baseURL + "/api/v1/pods")
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

	var podList K8sPodList
	if err := json.Unmarshal(body, &podList); err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// getK8sSecrets 获取 Secret 列表
func getK8sSecrets(client *http.Client, baseURL string, limit int) ([]K8sSecret, error) {
	url := fmt.Sprintf("%s/api/v1/secrets?limit=%d", baseURL, limit)
	resp, err := client.Get(url)
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

	var secretList K8sSecretList
	if err := json.Unmarshal(body, &secretList); err != nil {
		return nil, err
	}

	return secretList.Items, nil
}

// ExecuteKubernetesCommand 执行 Kubernetes 命令（支持SSH隧道）
func ExecuteKubernetesCommand(host, port string, sshConfig *SSHConfig, command string) (string, error) {
	addr := net.JoinHostPort(host, port)

	// 确定使用的协议
	scheme := "https"
	if port == "8080" || port == "10255" {
		scheme = "http"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, addr)

	client := createK8sHTTPClient(sshConfig)

	// 解析命令
	command = strings.TrimSpace(command)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("命令为空")
	}

	// 支持的命令
	switch parts[0] {
	case "namespaces", "ns":
		// 列出命名空间
		namespaces, err := getK8sNamespaces(client, baseURL)
		if err != nil {
			return "", err
		}
		return formatK8sNamespaces(namespaces), nil

	case "pods":
		// 列出 Pod
		pods, err := getK8sPods(client, baseURL)
		if err != nil {
			return "", err
		}
		return formatK8sPods(pods), nil

	case "secrets":
		// 列出 Secret
		limit := 10
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &limit)
		}
		secrets, err := getK8sSecrets(client, baseURL, limit)
		if err != nil {
			return "", err
		}
		return formatK8sSecrets(secrets), nil

	case "version":
		// 版本信息
		version, err := getK8sVersion(client, baseURL)
		if err != nil {
			return "", err
		}
		return formatK8sVersion(version), nil

	default:
		return "", fmt.Errorf("不支持的命令: %s\n支持的命令: namespaces, pods, secrets, version", parts[0])
	}
}

// formatK8sNamespaces 格式化命名空间列表
func formatK8sNamespaces(namespaces []string) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("命名空间列表 (共 %d 个):\n\n", len(namespaces)))

	for i, ns := range namespaces {
		result.WriteString(fmt.Sprintf("[%d] %s\n", i+1, ns))
	}

	return result.String()
}

// formatK8sPods 格式化 Pod 列表
func formatK8sPods(pods []K8sPod) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Pod 列表 (共 %d 个):\n\n", len(pods)))

	for i, pod := range pods {
		result.WriteString(fmt.Sprintf("[%d] %s/%s\n", i+1, pod.Metadata.Namespace, pod.Metadata.Name))
		result.WriteString(fmt.Sprintf("    状态: %s\n", pod.Status.Phase))
		if len(pod.Spec.Containers) > 0 {
			result.WriteString("    容器:\n")
			for _, container := range pod.Spec.Containers {
				result.WriteString(fmt.Sprintf("      - %s: %s\n", container.Name, container.Image))
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

// formatK8sSecrets 格式化 Secret 列表
func formatK8sSecrets(secrets []K8sSecret) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Secret 列表 (共 %d 个):\n\n", len(secrets)))

	for i, secret := range secrets {
		result.WriteString(fmt.Sprintf("[%d] %s/%s\n", i+1, secret.Metadata.Namespace, secret.Metadata.Name))
		result.WriteString(fmt.Sprintf("    类型: %s\n", secret.Type))
		result.WriteString(fmt.Sprintf("    数据项: %d 个\n", len(secret.Data)))
		if len(secret.Data) > 0 {
			result.WriteString("    键:\n")
			for key := range secret.Data {
				result.WriteString(fmt.Sprintf("      - %s\n", key))
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

// formatK8sVersion 格式化版本信息
func formatK8sVersion(version *K8sVersion) string {
	var result strings.Builder
	result.WriteString("Kubernetes 版本信息:\n\n")
	result.WriteString(fmt.Sprintf("版本: %s\n", version.GitVersion))
	result.WriteString(fmt.Sprintf("Go 版本: %s\n", version.GoVersion))
	result.WriteString(fmt.Sprintf("平台: %s\n", version.Platform))
	result.WriteString(fmt.Sprintf("构建日期: %s\n", version.BuildDate))
	result.WriteString(fmt.Sprintf("Git Commit: %s\n", version.GitCommit))
	return result.String()
}
