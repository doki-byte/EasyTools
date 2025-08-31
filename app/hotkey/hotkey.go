//go:build !windows
// +build !windows

package hotkey

import (
	"context"
	"errors"
)

// HotKey 空实现，用于非Windows平台
type HotKey struct {
	ctx context.Context
}

func NewHotKey() *HotKey {
	return &HotKey{}
}

func (h *HotKey) SetContext(ctx context.Context) { h.ctx = ctx }

func (h *HotKey) ToggleShowHide() error {
	return errors.New("hotkey functionality is only available on Windows")
}

func (h *HotKey) IsVisible() bool {
	return true
}

func (h *HotKey) SetHotkey(accel string) error {
	return errors.New("hotkey functionality is only available on Windows")
}

func (h *HotKey) UnregisterHotkey() error {
	return errors.New("hotkey functionality is only available on Windows")
}
