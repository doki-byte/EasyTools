//go:build !windows

package skyshadow

import "syscall"

func hideWindowAttr() *syscall.SysProcAttr {
	return nil
}
