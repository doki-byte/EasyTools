package util

import "syscall"

// HideWindowAttr 返回隐藏窗口的进程属性（非 Windows 平台返回 nil），
// 供跨平台编译的模块复用。
func HideWindowAttr() *syscall.SysProcAttr {
	return hideWindowAttr()
}
