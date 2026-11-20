//go:build !windows

package main

import "syscall"

// hiddenProcAttr 非 Windows 平台无窗口隐藏概念，返回 nil
func hiddenProcAttr() *syscall.SysProcAttr {
	return nil
}
