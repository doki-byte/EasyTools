import { LOCALSTORAGE_KEY, DEFAULT_HOTKEY, normalizeAccelerator, accelFromEvent } from './hotkeyUtils';
import { ToggleShowHide } from '../../wailsjs/go/hotkey/HotKey';

class GlobalHotkeyManager {
    constructor() {
        this.hotkeyConfig = { showHide: DEFAULT_HOTKEY };
        this._listeners = new Set();
        this._enabled = true;
        this._globalKeyHandler = this._globalKeyHandler.bind(this);
    }

    init() {
        this.loadHotkeyFromStorage();
        window.addEventListener('keydown', this._globalKeyHandler, true);
        console.debug('Global hotkey listener enabled');
    }

    destroy() {
        window.removeEventListener('keydown', this._globalKeyHandler, true);
        this._listeners.clear();
    }

    loadHotkeyFromStorage() {
        try {
            const v = localStorage.getItem(LOCALSTORAGE_KEY);
            this.hotkeyConfig.showHide = v ? normalizeAccelerator(v) : DEFAULT_HOTKEY;
        } catch (err) {
            console.error('读取 localStorage 失败:', err);
            this.hotkeyConfig.showHide = DEFAULT_HOTKEY;
        }
    }

    saveHotkeyConfig(accelerator) {
        try {
            const norm = normalizeAccelerator(accelerator);
            localStorage.setItem(LOCALSTORAGE_KEY, norm);
            this.hotkeyConfig.showHide = norm;
            return true;
        } catch (error) {
            console.error('保存 localStorage 失败:', error);
            return false;
        }
    }

    addListener(callback) { this._listeners.add(callback); }
    removeListener(callback) { this._listeners.delete(callback); }
    setEnabled(enabled) { this._enabled = enabled; }

    _globalKeyHandler(e) {
        if (!this._enabled) return;

        // 输入框中忽略
        const active = document.activeElement;
        if (active && (active.tagName === 'INPUT' || active.tagName === 'TEXTAREA' || active.isContentEditable)) return;

        const pressed = accelFromEvent(e);
        if (!pressed) return;

        const stored = normalizeAccelerator(this.hotkeyConfig.showHide || '');
        if (stored && pressed.toLowerCase() === stored.toLowerCase()) {
            e.preventDefault();
            e.stopPropagation();

            try {
                ToggleShowHide();
            } catch (err) {
                console.error('ToggleShowHide error:', err);
            }

            // 通知所有监听器（可选）
            this._listeners.forEach(cb => {
                try { cb(); } catch (err) { console.error(err); }
            });
        }
    }
}

export const globalHotkeyManager = new GlobalHotkeyManager();
