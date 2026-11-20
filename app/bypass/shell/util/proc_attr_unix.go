//go:build !windows

package util

import "syscall"

func hideWindowAttr() *syscall.SysProcAttr {
	return nil
}
