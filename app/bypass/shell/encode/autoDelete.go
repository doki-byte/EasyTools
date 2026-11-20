package encode

var (
	AutoDeleteString = []string{``, `
	AutoDeleteFmt "fmt"
	AutoDeleteFilepath "path/filepath"
	"io"
	AutoDeleteOs "os"
	"os/exec"
	AutoDeteSyscall "syscall"
	AutoDeleteTime "time"
`, `
// 复制文件函数
func copyDeleteFile(src, dst string) error {
	source, err := AutoDeleteOs.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := AutoDeleteOs.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func copyReplaceDelete(exePath string) {
	// 创建一个临时副本文件名
	tmpFileName := AutoDeleteFmt.Sprintf("delete_me_%d.exe", AutoDeleteTime.Now().UnixNano())
	tmpFile := AutoDeleteFilepath.Join(AutoDeleteOs.TempDir(), tmpFileName)

	// 复制当前程序到临时文件
	if err := copyDeleteFile(exePath, tmpFile); err != nil {
		AutoDeleteFmt.Println("创建临时副本失败:", err)
		return
	}

	// 启动临时副本并传递原始程序路径和当前PID
	cmd := exec.Command(tmpFile, "--cleanup", exePath, AutoDeleteFmt.Sprint(AutoDeleteOs.Getpid()))
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &AutoDeteSyscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		AutoDeleteFmt.Println("启动清理程序失败:", err)
		return
	}

	AutoDeleteFmt.Println("清理程序已启动，原始程序即将退出")
	AutoDeleteOs.Exit(0)
}

// 清理函数 - 删除原始文件和临时副本
func cleanup() {
	if len(AutoDeleteOs.Args) < 4 {
		AutoDeleteFmt.Println("清理程序需要目标文件路径和PID")
		AutoDeleteOs.Exit(1)
	}

	target := AutoDeleteOs.Args[2] // 原始程序路径
	self := AutoDeleteOs.Args[0]   // 当前程序路径（临时副本）
	pid := AutoDeleteOs.Args[3]    // 原始进程PID

	// 1. 强制终止原始进程
	AutoDeleteFmt.Println("强制终止原始进程...")
	killCmd := exec.Command("taskkill", "/F", "/PID", pid)
	killCmd.Stdout = nil
	killCmd.Stderr = nil
	killCmd.SysProcAttr = &AutoDeteSyscall.SysProcAttr{HideWindow: true}
	if err := killCmd.Run(); err != nil {
		AutoDeleteFmt.Println("终止进程失败:", err)
	}

	// 2. 尝试删除目标文件
	AutoDeleteFmt.Println("删除目标文件...")
	if err := AutoDeleteOs.Remove(target); err != nil {
		AutoDeleteFmt.Println("删除目标文件失败:", err)
	} else {
		AutoDeleteFmt.Println("目标文件已删除")
	}

	// 3. 创建批处理文件来删除自身
	AutoDeleteFmt.Println("创建自我删除脚本...")
	batContent := AutoDeleteFmt.Sprintf(""+
		"@echo off\n"+
		"AutoDeleteTimeout /t 1 /nobreak >nul\n"+
		"del /f /q \"%s\" >nul 2>&1\n"+
		"del /f /q \"%%~f0\" >nul 2>&1\n",
		self)

	batFile := AutoDeleteFilepath.Join(AutoDeleteOs.TempDir(), AutoDeleteFmt.Sprintf("delete_self_%d.bat", AutoDeleteTime.Now().UnixNano()))
	if err := AutoDeleteOs.WriteFile(batFile, []byte(batContent), 0755); err != nil {
		AutoDeleteFmt.Println("创建批处理失败:", err)
		return
	}

	// 4. 执行批处理文件并立即退出
	cmd := exec.Command("cmd.exe", "/C", batFile)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &AutoDeteSyscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		AutoDeleteFmt.Println("启动批处理失败:", err)
	} else {
		AutoDeleteFmt.Println("已安排删除自身")
	}
	AutoDeleteOs.Exit(0)
}

func CheckAutoDelete(deadline string) {
	// 检查是否作为清理程序启动
	if len(AutoDeleteOs.Args) > 1 && AutoDeleteOs.Args[1] == "--cleanup" {
		cleanup()
		return
	}

	// 获取当前可执行文件路径
	exePath, err := AutoDeleteOs.Executable()
	if err != nil {
		AutoDeleteFmt.Println("获取可执行路径失败:", err)
		return
	}

	// 解析截止时间
	expiryDate, err := AutoDeleteTime.ParseInLocation("2006-01-02 15:04:05", deadline, AutoDeleteTime.Local)
	if err != nil {
		AutoDeleteFmt.Println("解析日期失败:", err)
		return
	}

	// 立即检查是否过期
	if AutoDeleteTime.Now().After(expiryDate) {
		AutoDeleteFmt.Println("程序已过期，将自动删除")
		copyReplaceDelete(exePath)
		return
	}

	// 启动后台协程持续检查截止时间
	go func() {
		for {
			// 计算距离过期的时间
			AutoDeleteTimeLeft := AutoDeleteTime.Until(expiryDate)
			if AutoDeleteTimeLeft <= 0 {
				AutoDeleteFmt.Println("程序运行中检测到过期，将自动删除")
				copyReplaceDelete(exePath)
				return
			}

			checkInterval := AutoDeleteTime.Hour
			if AutoDeleteTimeLeft < AutoDeleteTime.Hour {
				checkInterval = AutoDeleteTime.Minute
			}
			if AutoDeleteTimeLeft < AutoDeleteTime.Minute {
				checkInterval = AutoDeleteTime.Second
			}

			// 等待下次检查
			AutoDeleteTime.Sleep(checkInterval)
		}
	}()
}
`, `
	CheckAutoDelete("{{.Deadline}}")`,
	}
)
