//go:build !windows
// +build !windows

package cli

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"EasyTools/app/controller/localguard"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// 仅放行本地回环 / Wails 来源（与 CLI 服务共用端口校验），阻断恶意网页 WebSocket
	CheckOrigin: func(r *http.Request) bool {
		port := 0
		cliServerPortMutex.Lock()
		port = cliServerPort
		cliServerPortMutex.Unlock()
		return localguard.Check(r, port)
	},
}

// WebTerminalSession 表示一个 Web 终端会话
type WebTerminalSession struct {
	ID           string
	Command      string
	Args         []string
	WorkingDir   string
	PTY          *os.File
	CommandProc  *exec.Cmd
	Context      context.Context
	Cancel       context.CancelFunc
	Mutex        sync.Mutex
	IsActive     bool
	LastActivity time.Time
}

var (
	webTerminalSessions = make(map[string]*WebTerminalSession)
	sessionsMutex       sync.RWMutex
)

// WebTerminalRequest Web 终端请求
type WebTerminalRequest struct {
	ExecutionID string `json:"executionId"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
}

// WebTerminalMessage Web 终端消息
type WebTerminalMessage struct {
	Type   string `json:"type"` // input, resize, ping, pong
	Data   string `json:"data,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// handleWebTerminal 处理 Web 终端 WebSocket 连接
func handleWebTerminal(w http.ResponseWriter, r *http.Request) {
	// 升级到 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// 从查询参数获取执行ID
	executionID := r.URL.Query().Get("executionId")
	if executionID == "" {
		log.Printf("Missing executionId parameter")
		conn.WriteJSON(WebTerminalMessage{
			Type: "error",
			Data: "Missing executionId parameter",
		})
		return
	}

	// 获取会话
	sessionsMutex.RLock()
	session, exists := webTerminalSessions[executionID]
	sessionsMutex.RUnlock()

	if !exists {
		log.Printf("Session not found: %s", executionID)
		conn.WriteJSON(WebTerminalMessage{
			Type: "error",
			Data: "Session not found",
		})
		return
	}

	// 设置会话活动状态
	session.Mutex.Lock()
	session.IsActive = true
	session.LastActivity = time.Now()
	session.Mutex.Unlock()

	log.Printf("Web terminal connected: %s", executionID)

	// 启动读取 PTY 输出的 goroutine
	go func() {
		buffer := make([]byte, 1024)
		for {
			session.Mutex.Lock()
			if !session.IsActive || session.PTY == nil {
				session.Mutex.Unlock()
				break
			}
			pty := session.PTY
			session.Mutex.Unlock()

			n, err := pty.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Printf("PTY read error: %v", err)
				}
				break
			}

			if n > 0 {
				// 使用BinaryMessage发送原始字节数据，保持终端输出的完整性
				err = conn.WriteMessage(websocket.BinaryMessage, buffer[:n])
				if err != nil {
					log.Printf("WebSocket write error: %v", err)
					break
				}
			}
		}
	}()

	// 处理 WebSocket 消息
	for {
		var msg WebTerminalMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		session.Mutex.Lock()
		session.LastActivity = time.Now()
		session.Mutex.Unlock()

		switch msg.Type {
		case "input":
			// 写入输入到 PTY
			// msg.Data 是字符串，需要转换为字节
			session.Mutex.Lock()
			if session.PTY != nil {
				// 直接写入字节数据，支持所有快捷键
				data := []byte(msg.Data)
				_, _ = session.PTY.Write(data)
				// if err != nil {
				// 	log.Printf("PTY write error: %v", err)
				// } else {
				// 	log.Printf("PTY wrote %d bytes", len(data))
				// }
			} else {
				log.Printf("PTY is nil, cannot write input")
			}
			session.Mutex.Unlock()

		case "resize":
			// 调整 PTY 大小
			if msg.Width > 0 && msg.Height > 0 {
				session.Mutex.Lock()
				if session.PTY != nil {
					err := pty.Setsize(session.PTY, &pty.Winsize{
						Rows: uint16(msg.Height),
						Cols: uint16(msg.Width),
					})
					if err != nil {
						log.Printf("PTY resize error: %v", err)
					}
				}
				session.Mutex.Unlock()
			}

		case "ping":
			// 响应 ping
			conn.WriteJSON(WebTerminalMessage{
				Type: "pong",
			})
		}
	}

	// 连接关闭
	session.Mutex.Lock()
	session.IsActive = false
	session.Mutex.Unlock()

	log.Printf("Web terminal disconnected: %s", executionID)
}

// createWebTerminalSession 创建 Web 终端会话
func createWebTerminalSession(executionID, command string, args []string, workingDir string) (*WebTerminalSession, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 根据操作系统选择 shell
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Windows 使用 cmd
		fullCommand := command
		if len(args) > 0 {
			fullCommand += " " + strings.Join(args, " ")
		}
		if workingDir != "" {
			fullCommand = fmt.Sprintf("cd /D %s && %s", workingDir, fullCommand)
		}
		cmd = exec.CommandContext(ctx, "cmd", "/K", fullCommand)
	case "darwin", "linux":
		// macOS/Linux 使用 shell
		shellPath := os.Getenv("SHELL")
		if shellPath == "" {
			if runtime.GOOS == "darwin" {
				shellPath = "/bin/zsh"
			} else {
				shellPath = "/bin/bash"
			}
		}

		// 构建命令
		fullCommand := command
		if len(args) > 0 {
			fullCommand += " " + strings.Join(args, " ")
		}

		cmd = exec.CommandContext(ctx, shellPath, "-c", fullCommand)
		if workingDir != "" {
			cmd.Dir = workingDir
		}
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// 设置环境变量
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")
	cmd.Env = append(cmd.Env, "COLORTERM=truecolor")

	// 创建 PTY
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	// 设置默认大小
	pty.Setsize(ptyFile, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})

	session := &WebTerminalSession{
		ID:           executionID,
		Command:      command,
		Args:         args,
		WorkingDir:   workingDir,
		PTY:          ptyFile,
		CommandProc:  cmd,
		Context:      ctx,
		Cancel:       cancel,
		IsActive:     false,
		LastActivity: time.Now(),
	}

	// 保存会话
	sessionsMutex.Lock()
	webTerminalSessions[executionID] = session
	sessionCount := len(webTerminalSessions)
	sessionsMutex.Unlock()

	log.Printf("Web terminal session saved: executionID=%s, total sessions=%d", executionID, sessionCount)

	// 等待命令结束
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("Command exited with error: %v", err)
		}

		// 清理资源
		session.Mutex.Lock()
		if session.PTY != nil {
			session.PTY.Close()
			session.PTY = nil
		}
		session.Mutex.Unlock()

		// 延迟删除会话（给客户端时间接收最后的输出）
		time.Sleep(5 * time.Second)
		sessionsMutex.Lock()
		delete(webTerminalSessions, executionID)
		sessionsMutex.Unlock()

		log.Printf("Web terminal session cleaned up: %s", executionID)
	}()

	log.Printf("Web terminal session created: %s", executionID)
	return session, nil
}

// stopWebTerminalSession 停止 Web 终端会话
func stopWebTerminalSession(executionID string) error {
	sessionsMutex.RLock()
	session, exists := webTerminalSessions[executionID]
	sessionsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("session not found: %s", executionID)
	}

	session.Mutex.Lock()
	defer session.Mutex.Unlock()

	if session.Cancel != nil {
		session.Cancel()
	}

	if session.CommandProc != nil && session.CommandProc.Process != nil {
		session.CommandProc.Process.Kill()
	}

	if session.PTY != nil {
		session.PTY.Close()
		session.PTY = nil
	}

	return nil
}
