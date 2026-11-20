//go:build darwin || linux

package system

import (
	"context"
)

type SystemTp struct {
	Base
}

func NewSystemTp() *SystemTp {
	return &SystemTp{}
}

func (a *SystemTp) systemTray(systemTrayIcon []byte) {}

func (a *SystemTp) Startup(ctx context.Context, systemTrayIcon []byte) {}
