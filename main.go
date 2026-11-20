package main

import (
	"EasyTools/app/controller"
	"EasyTools/app/controller/cli"
	"EasyTools/app/controller/system"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/appicon.ico
var sysTrayIcon []byte

func main() {
	// CLI 模式：启动 MCP stdio server（供 Cursor 等客户端通过 stdio 连接）
	if len(os.Args) >= 2 && os.Args[1] == "mcp" {
		// MCP 模式下将日志输出到文件，避免干扰 JSON-RPC 协议的 stdout
		// 同时避免被外部进程拉起控制台时覆盖 stdout
		baseDir := system.GetAppBaseDir()
		logsDir := filepath.Join(baseDir, "logs")
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			// 如果无法创建日志目录，回退到 stderr
			log.SetOutput(os.Stderr)
			log.Printf("Failed to create logs directory: %v, using stderr", err)
			fmt.Fprintf(os.Stderr, "[MCP] Log directory creation failed, using stderr\n")
		} else {
			logFile := filepath.Join(logsDir, fmt.Sprintf("mcp_%s.log", time.Now().Format("2006-01-02")))
			file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				// 如果无法打开日志文件，回退到 stderr
				log.SetOutput(os.Stderr)
				log.Printf("Failed to open log file: %v, using stderr", err)
				fmt.Fprintf(os.Stderr, "[MCP] Log file open failed: %v, using stderr\n", err)
			} else {
				// 不要 defer close，因为 StartMCPServerStdio 是阻塞的
				// 使用自定义 logger 确保立即刷新
				logger := log.New(file, "[MCP] ", log.LstdFlags)
				log.SetOutput(file)
				log.SetPrefix("[MCP] ")
				// 立即写入启动日志并刷新
				logger.Printf("MCP server starting... (PID: %d, Log: %s)", os.Getpid(), logFile)
				file.Sync()
				// 同时在 stderr 输出日志文件路径，方便调试
				fmt.Fprintf(os.Stderr, "[MCP] Logging to: %s\n", logFile)
			}
		}
		if log.Prefix() == "" {
			log.SetPrefix("[MCP] ")
		}
		
		log.Printf("MCP server initialized, waiting for requests...")
		// 确保日志立即刷新
		flushLog()
		
		// 目前仅支持 stdio，第二个参数可选 --stdio
		if err := cli.StartMCPServerStdio(); err != nil {
			log.Printf("MCP server error: %v", err)
			flushLog()
			os.Exit(1)
		}
		flushLog()
		return
	}

	// 使用 0 表示自动选择端口
	controller.WailsRun(assets, 0, appIcon, sysTrayIcon)
}

// flushLog 刷新日志输出
func flushLog() {
	if w := log.Writer(); w != nil {
		if f, ok := w.(*os.File); ok && f != os.Stderr && f != os.Stdout {
			f.Sync()
		}
	}
}
