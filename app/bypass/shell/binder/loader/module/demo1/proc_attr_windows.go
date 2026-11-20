//go:build windows

package main

import "syscall"

// hiddenProcAttr 返回隐藏子进程窗口的进程属性（Windows）
func hiddenProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
