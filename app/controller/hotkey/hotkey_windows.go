package hotkey

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.design/x/hotkey"
)

// HotKey 负责在后端注册系统级全局热键并调用 toggleShowHide
type HotKey struct {
	ctx           context.Context
	mu            sync.Mutex
	isVisible     bool
	hk            *hotkey.Hotkey
	currentAccel  string
	stopListen    chan struct{}
	skipFirst     bool // 新增：用于跳过第一次触发的标志
	isRegistering bool // 新增：标记是否正在注册
}

func NewHotKey() *HotKey {
	return &HotKey{
		isVisible:     true,
		stopListen:    make(chan struct{}),
		skipFirst:     false, // 初始化为false
		isRegistering: false, // 初始化为false
	}
}

func (h *HotKey) SetContext(ctx context.Context) { h.ctx = ctx }

// ToggleShowHide 仍然保留给前端调用（wails RPC）
func (h *HotKey) ToggleShowHide() error {
	if h == nil {
		return errors.New("HotKey is nil")
	}
	if h.ctx == nil {
		return errors.New("wails context not set")
	}
	return h.toggleShowHide()
}

func (h *HotKey) toggleShowHide() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.isVisible {
		wruntime.WindowHide(h.ctx)
		h.isVisible = false
	} else {
		// show & bring to front
		wruntime.WindowShow(h.ctx)
		wruntime.WindowUnminimise(h.ctx)
		// set always on top briefly to ensure focus (可根据需求调整)
		wruntime.WindowSetAlwaysOnTop(h.ctx, true)
		// 取消置顶在前端或后端可选，这里保持置顶短时间后由前端控制也可
		h.isVisible = true
	}
	return nil
}

// IsVisible 返回当前缓存的可见性（注意：不是绝对窗口状态）
func (h *HotKey) IsVisible() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.isVisible
}

// 如果已有热键，会先注销旧热键再注册新热键。
func (h *HotKey) SetHotkey(accel string) error {
	h.mu.Lock()

	// 如果正在注册中，直接返回
	if h.isRegistering {
		h.mu.Unlock()
		return nil
	}

	// 如果热键没有变化，直接返回
	if h.currentAccel == accel {
		h.mu.Unlock()
		return nil
	}

	h.isRegistering = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		h.isRegistering = false
		h.mu.Unlock()
	}()

	h.mu.Lock()
	// 取消之前的监听（如果有）
	if h.hk != nil {
		_ = h.hk.Unregister()
		h.hk = nil
		// 触发 stop 用于结束 goroutine
		select {
		case <-h.stopListen:
			// already closed, recreate
			h.stopListen = make(chan struct{})
		default:
			close(h.stopListen)
			h.stopListen = make(chan struct{})
		}
		h.currentAccel = ""
	}

	accel = strings.TrimSpace(accel)
	if accel == "" {
		// empty means unregister only
		h.currentAccel = ""
		h.mu.Unlock()
		return nil
	}

	// 解析 accelerator -> modifiers 和 key
	mods, key, err := parseAccelerator(accel)
	if err != nil {
		h.mu.Unlock()
		return err
	}

	// 创建 hotkey
	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		h.mu.Unlock()
		return err
	}

	h.hk = hk
	h.currentAccel = accel

	// 设置跳过第一次触发的标志
	h.skipFirst = true

	// 启动监听 goroutine（非阻塞）
	go h.listenHotkey(hk, h.stopListen)

	h.mu.Unlock()

	// 5秒后自动清除跳过标志（防止永远跳过）
	go func() {
		time.Sleep(5 * time.Second)
		h.mu.Lock()
		h.skipFirst = false
		h.mu.Unlock()
	}()

	//log.Printf("hotkey: registered %s\n", accel)
	return nil
}

// UnregisterHotkey: 手动注销
func (h *HotKey) UnregisterHotkey() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hk != nil {
		_ = h.hk.Unregister()
		h.hk = nil
		select {
		case <-h.stopListen:
			// already closed
		default:
			close(h.stopListen)
		}
		h.currentAccel = ""
	}
	return nil
}

func (h *HotKey) listenHotkey(hk *hotkey.Hotkey, stop chan struct{}) {
	kd := hk.Keydown()
	ku := hk.Keyup()
	for {
		select {
		case <-kd:
			// 检查是否需要跳过第一次触发
			h.mu.Lock()
			shouldSkip := h.skipFirst
			if shouldSkip {
				h.skipFirst = false // 清除标志，只跳过这一次
			}
			h.mu.Unlock()

			if shouldSkip {
				//log.Println("hotkey: skipping first trigger after registration")
				continue
			}

			// 收到按下事件：直接切换窗口
			// 为避免在系统线程/主线程上阻塞，使用 goroutine 调用（toggle 会使用 wruntime）
			go func() {
				if err := h.toggleShowHide(); err != nil {
					//log.Println("hotkey toggle error:", err)
				}
			}()
		case <-ku:
			// key up (通常不需要处理)
		case <-stop:
			return
		}
	}
}

// 支持常见修饰键：Ctrl / Alt / Shift / Cmd(Cmd 或 Meta)
/*
   注意：这里做了常用键的匹配（A-Z, 0-9, F1-F12, Enter, Space, Escape, Arrow keys 等）。
   若你在实际环境中需要更多键，请按需补充映射表或直接使用 keycode（hotkey.Key(0x15)）。
*/
func parseAccelerator(acc string) ([]hotkey.Modifier, hotkey.Key, error) {
	parts := strings.Split(acc, "+")
	var mods []hotkey.Modifier
	var keyPart string

	for _, p := range parts {
		pp := strings.TrimSpace(strings.ToLower(p))
		switch pp {
		case "ctrl", "control":
			mods = append(mods, hotkey.ModCtrl)
		case "shift":
			mods = append(mods, hotkey.ModShift)
		case "alt", "option":
			// 根据平台使用正确的 Alt 修饰符
			mods = append(mods, getAltModifier())
		case "cmd", "meta", "command":
			// 根据平台使用正确的 Cmd/Windows 修饰符
			mods = append(mods, getCmdModifier())
		default:
			// 不是修饰键，认为是主体键
			keyPart = pp
		}
	}

	if keyPart == "" {
		return nil, 0, errors.New("no key found in accelerator")
	}

	// 字母
	if len(keyPart) == 1 && keyPart[0] >= 'a' && keyPart[0] <= 'z' {
		switch strings.ToUpper(keyPart) {
		case "A":
			return mods, hotkey.KeyA, nil
		case "B":
			return mods, hotkey.KeyB, nil
		case "C":
			return mods, hotkey.KeyC, nil
		case "D":
			return mods, hotkey.KeyD, nil
		case "E":
			return mods, hotkey.KeyE, nil
		case "F":
			return mods, hotkey.KeyF, nil
		case "G":
			return mods, hotkey.KeyG, nil
		case "H":
			return mods, hotkey.KeyH, nil
		case "I":
			return mods, hotkey.KeyI, nil
		case "J":
			return mods, hotkey.KeyJ, nil
		case "K":
			return mods, hotkey.KeyK, nil
		case "L":
			return mods, hotkey.KeyL, nil
		case "M":
			return mods, hotkey.KeyM, nil
		case "N":
			return mods, hotkey.KeyN, nil
		case "O":
			return mods, hotkey.KeyO, nil
		case "P":
			return mods, hotkey.KeyP, nil
		case "Q":
			return mods, hotkey.KeyQ, nil
		case "R":
			return mods, hotkey.KeyR, nil
		case "S":
			return mods, hotkey.KeyS, nil
		case "T":
			return mods, hotkey.KeyT, nil
		case "U":
			return mods, hotkey.KeyU, nil
		case "V":
			return mods, hotkey.KeyV, nil
		case "W":
			return mods, hotkey.KeyW, nil
		case "X":
			return mods, hotkey.KeyX, nil
		case "Y":
			return mods, hotkey.KeyY, nil
		case "Z":
			return mods, hotkey.KeyZ, nil
		}
	}

	// 数字
	if len(keyPart) == 1 && keyPart[0] >= '0' && keyPart[0] <= '9' {
		switch keyPart {
		case "0":
			return mods, hotkey.Key0, nil
		case "1":
			return mods, hotkey.Key1, nil
		case "2":
			return mods, hotkey.Key2, nil
		case "3":
			return mods, hotkey.Key3, nil
		case "4":
			return mods, hotkey.Key4, nil
		case "5":
			return mods, hotkey.Key5, nil
		case "6":
			return mods, hotkey.Key6, nil
		case "7":
			return mods, hotkey.Key7, nil
		case "8":
			return mods, hotkey.Key8, nil
		case "9":
			return mods, hotkey.Key9, nil
		}
	}

	// 功能键 / 特殊键
	switch strings.ToLower(keyPart) {
	case "space", "spacebar":
		return mods, hotkey.KeySpace, nil
	case "esc", "escape":
		return mods, hotkey.KeyEscape, nil
	case "up", "arrowup":
		return mods, hotkey.KeyUp, nil
	case "down", "arrowdown":
		return mods, hotkey.KeyDown, nil
	case "left", "arrowleft":
		return mods, hotkey.KeyLeft, nil
	case "right", "arrowright":
		return mods, hotkey.KeyRight, nil
	}

	// F1-F12
	if strings.HasPrefix(strings.ToUpper(keyPart), "F") {
		switch strings.ToUpper(keyPart) {
		case "F1":
			return mods, hotkey.KeyF1, nil
		case "F2":
			return mods, hotkey.KeyF2, nil
		case "F3":
			return mods, hotkey.KeyF3, nil
		case "F4":
			return mods, hotkey.KeyF4, nil
		case "F5":
			return mods, hotkey.KeyF5, nil
		case "F6":
			return mods, hotkey.KeyF6, nil
		case "F7":
			return mods, hotkey.KeyF7, nil
		case "F8":
			return mods, hotkey.KeyF8, nil
		case "F9":
			return mods, hotkey.KeyF9, nil
		case "F10":
			return mods, hotkey.KeyF10, nil
		case "F11":
			return mods, hotkey.KeyF11, nil
		case "F12":
			return mods, hotkey.KeyF12, nil
		}
	}

	// 若未匹配到，返回错误（可扩展）
	return nil, 0, errors.New("unsupported key: " + keyPart)
}

// getAltModifier 返回 Windows 平台的 Alt 修饰符
func getAltModifier() hotkey.Modifier {
	return hotkey.ModAlt
}

// getCmdModifier 返回 Windows 平台的 Windows 键修饰符
func getCmdModifier() hotkey.Modifier {
	return hotkey.ModWin
}
