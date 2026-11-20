package service

import (
	"EasyTools/app/controller/connect/ftp/app/model"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"golang.org/x/crypto/ssh"
)

// FtpDesktopBridge 提供给 Wails 前端调用的桥接结构（桌面端专用）
// 用于上传本地文件/目录到 FTP 服务器
type FtpDesktopBridge struct{}

// UploadLocalPaths 根据本地绝对路径列表，将文件/目录上传到 FTP 服务器的目标目录。
// configID: FTP 配置 ID
// targetDir: 远程目标目录，例如 "/tmp" 或 "/"
// paths: 本地绝对路径列表（文件或目录）
func (b *FtpDesktopBridge) UploadLocalPaths(configID string, targetDir string, paths []string) (string, error) {
	if configID == "" {
		return "", errors.New("config_id 不能为空")
	}
	if len(paths) == 0 {
		return "", errors.New("本地路径列表为空")
	}

	// 获取 FTP 配置
	configIDInt, err := strconv.Atoi(configID)
	if err != nil {
		return "", fmt.Errorf("无效的配置ID: %v", err)
	}

	// 从数据库获取配置
	var configModel model.FtpConfig
	config, err := configModel.FindByID(uint(configIDInt))
	if err != nil {
		return "", fmt.Errorf("获取FTP配置失败: %v", err)
	}

	// 解析端口
	port, err := strconv.Atoi(config.Port)
	if err != nil {
		return "", fmt.Errorf("无效的端口: %v", err)
	}

	// 解析 SSH 端口（如果使用 SSH 隧道）
	var sshPort int
	if config.ConnectionType == "ssh" && config.SSHPort != "" {
		sshPort, err = strconv.Atoi(config.SSHPort)
		if err != nil {
			sshPort = 22 // 默认 SSH 端口
		}
	}

	// 密码已经在 AfterFind hook 中解码，直接使用
	password := config.Password
	sshPassword := ""
	if config.ConnectionType == "ssh" {
		sshPassword = config.SSHPassword
	}

	if targetDir == "" {
		targetDir = "/"
	}
	// 规范化远程目录
	if !strings.HasPrefix(targetDir, "/") {
		targetDir = "/" + targetDir
	}
	targetDir = path.Clean(targetDir)
	if !strings.HasSuffix(targetDir, "/") && targetDir != "/" {
		targetDir += "/"
	}

	var uploaded, failed int

	// 为每个路径上传文件
	for _, p := range paths {
		if p == "" {
			failed++
			continue
		}

		info, statErr := os.Stat(p)
		if statErr != nil {
			slog.Error("本地文件状态读取失败", "path", p, "err", statErr)
			failed++
			continue
		}

		if info.IsDir() {
			// 目录：递归上传，保持目录结构
			if err := b.uploadDirRecursive(
				config.Host, port, config.Username, password,
				config.SSHAddr, sshPort, config.SSHUsername, sshPassword,
				p, targetDir, &uploaded, &failed,
			); err != nil {
				slog.Error("上传目录失败", "local_dir", p, "err", err)
			}
		} else {
			// 单文件：直接上传到目标目录
			if err := b.uploadSingleFile(
				config.Host, port, config.Username, password,
				config.SSHAddr, sshPort, config.SSHUsername, sshPassword,
				p, targetDir, "",
			); err != nil {
				slog.Error("上传文件失败", "local_file", p, "err", err)
				failed++
			} else {
				uploaded++
			}
		}
	}

	msg := fmt.Sprintf("成功: %d 个，失败: %d 个", uploaded, failed)
	if failed > 0 {
		return msg, errors.New(msg)
	}
	return msg, nil
}

// uploadDirRecursive 递归上传本地目录下的所有文件，保持相对目录结构
func (b *FtpDesktopBridge) uploadDirRecursive(
	host string, port int, username, password string,
	sshAddr string, sshPort int, sshUsername, sshPassword string,
	localRoot, remoteRoot string, uploaded, failed *int,
) error {
	localRoot = filepath.Clean(localRoot)

	walkErr := filepath.Walk(localRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Error("遍历本地目录出错", "path", p, "err", err)
			*failed++
			return nil
		}
		if info.IsDir() {
			// 目录本身在上传文件时顺带创建，不单独处理
			return nil
		}

		rel, relErr := filepath.Rel(localRoot, p)
		if relErr != nil {
			slog.Error("计算相对路径失败", "root", localRoot, "path", p, "err", relErr)
			*failed++
			return nil
		}

		subDir := filepath.Dir(rel)
		if subDir == "." {
			subDir = ""
		}

		if err := b.uploadSingleFile(
			host, port, username, password,
			sshAddr, sshPort, sshUsername, sshPassword,
			p, remoteRoot, subDir,
		); err != nil {
			slog.Error("上传目录中的文件失败", "local_file", p, "err", err)
			*failed++
		} else {
			*uploaded++
		}
		return nil
	})

	return walkErr
}

// uploadSingleFile 上传单个本地文件到远程，subDir 为相对 remoteRoot 的子目录，可为空
func (b *FtpDesktopBridge) uploadSingleFile(
	host string, port int, username, password string,
	sshAddr string, sshPort int, sshUsername, sshPassword string,
	localPath, remoteRoot, subDir string,
) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	fileName := filepath.Base(localPath)

	// 构建远程路径
	remoteDir := remoteRoot
	if subDir != "" && subDir != "." {
		// 将本地子目录转换为远程路径（统一使用 / ）
		remoteDir = path.Join(remoteRoot, filepath.ToSlash(subDir))
		if !strings.HasSuffix(remoteDir, "/") {
			remoteDir += "/"
		}
	}

	remotePath := path.Join(remoteDir, fileName)
	// 确保路径格式正确
	remotePath = strings.ReplaceAll(remotePath, "\\", "/")
	remotePath = strings.ReplaceAll(remotePath, "//", "/")

	// 连接 FTP
	conn, err := b.connectFTP(host, port, username, password, sshAddr, sshPort, sshUsername, sshPassword)
	if err != nil {
		return fmt.Errorf("连接FTP失败: %v", err)
	}
	defer func() {
		_ = conn.Quit()
	}()

	// 确保远程目录存在（递归创建）
	if remoteDir != "/" {
		dirParts := strings.Split(strings.Trim(remoteDir, "/"), "/")
		currentDir := "/"
		for _, part := range dirParts {
			if part == "" {
				continue
			}
			// 构建当前目录路径（不使用 path.Join，因为需要保持 / 分隔符）
			if currentDir == "/" {
				currentDir = "/" + part
			} else {
				currentDir = currentDir + "/" + part
			}
			// 尝试创建目录（如果已存在会返回错误，忽略即可）
			_ = conn.MakeDir(currentDir)
		}
	}

	// 上传文件
	err = conn.Stor(remotePath, f)
	if err != nil {
		return fmt.Errorf("上传文件失败: %v", err)
	}

	return nil
}

// connectFTP 连接 FTP 服务器（支持 SSH 隧道）
func (b *FtpDesktopBridge) connectFTP(host string, port int, username, password string, sshAddr string, sshPort int, sshUsername, sshPassword string) (*ftp.ServerConn, error) {
	address := fmt.Sprintf("%s:%d", host, port)

	// 如果配置了SSH隧道，通过SSH隧道连接
	if sshAddr != "" && sshPort > 0 && sshUsername != "" {
		sshClient, err := b.getSSHClient(sshUsername, sshPassword, sshAddr, sshPort)
		if err != nil {
			return nil, fmt.Errorf("SSH连接失败: %v", err)
		}

		// 通过SSH隧道连接到FTP服务器
		// 由于jlaffaye/ftp库不支持自定义dialer，我们需要使用端口转发
		// 创建一个本地监听端口进行转发
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			sshClient.Close()
			return nil, fmt.Errorf("创建本地监听端口失败: %v", err)
		}

		localAddr := listener.Addr().String()

		// 启动goroutine进行端口转发
		go func() {
			defer listener.Close()
			for {
				localConn, err := listener.Accept()
				if err != nil {
					return
				}

				// 通过SSH隧道连接到远程FTP服务器
				remoteConn, err := sshClient.Dial("tcp", address)
				if err != nil {
					localConn.Close()
					continue
				}

				// 进行双向数据转发
				go func() {
					defer localConn.Close()
					defer remoteConn.Close()
					io.Copy(localConn, remoteConn)
				}()
				go func() {
					defer localConn.Close()
					defer remoteConn.Close()
					io.Copy(remoteConn, localConn)
				}()
			}
		}()

		// 连接到本地端口（通过SSH隧道转发）
		conn, err := ftp.Dial(localAddr, ftp.DialWithTimeout(5*time.Second))
		if err != nil {
			listener.Close()
			sshClient.Close()
			return nil, fmt.Errorf("通过SSH隧道连接FTP失败: %v", err)
		}

		if err := conn.Login(username, password); err != nil {
			conn.Quit()
			listener.Close()
			sshClient.Close()
			return nil, err
		}

		return conn, nil
	}

	// 直接连接
	conn, err := ftp.Dial(address, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return nil, err
	}
	if err := conn.Login(username, password); err != nil {
		return nil, err
	}
	return conn, nil
}

// getSSHClient 创建SSH客户端连接
func (b *FtpDesktopBridge) getSSHClient(username, password, addr string, port int) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	sshAddr := fmt.Sprintf("%s:%d", addr, port)
	client, err := ssh.Dial("tcp", sshAddr, config)
	if err != nil {
		return nil, err
	}
	return client, nil
}
