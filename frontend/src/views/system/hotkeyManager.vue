<template>
  <div class="glass-card" style="padding: 20px; margin-bottom: 20px;">
    <h2>快捷键设置</h2>

    <div class="hotkey-item">
    <label>显示/隐藏界面：</label>
    <div class="input-group">
      <input
          type="text"
          v-model="hotkeyConfig.showHide"
          @keydown.prevent="onKeyDown($event, 'showHide')"
          placeholder="点击此处并按下快捷键"
          class="hotkey-input"
          ref="hotkeyInput"
      />
      <el-button @click="startDetection('showHide')" class="detect-btn">
        {{ detecting === 'showHide' ? '检测中...' : '按键识别' }}
      </el-button>
    </div>
  </div>

  <div class="button-group">
    <el-button type="primary" @click="saveHotkeyConfig" class="save-btn">保存设置</el-button>
    <el-button @click="resetHotkeyToDefault" class="reset-btn">关闭</el-button>
  </div>

  </div>

  <!-- 开机自启设置 -->
  <div class="glass-card" style="padding: 20px;">
    <h2>开机自启</h2>
    <div class="hotkey-item">
    <div class="auto-start-group">
      <el-checkbox
          v-model="autoStartEnabled"
          @change="handleAutoStartChange"
          :disabled="autoStartLoading"
      >
        {{ autoStartLoading ? '处理中...' : '开机自动启动' }}
      </el-checkbox>
      <span class="auto-start-desc">勾选后程序将在开机时自动启动</span>
    </div>
  </div>
  </div>

  <div v-if="hotkeyMessage" :class="['message', hotkeyMessageType]">{{ hotkeyMessage }}</div>
</template>

<script>
import {GetAutoStart, GetOs, SetAutoStart} from "../../../wailsjs/go/system/System";
import {accelFromEvent, DEFAULT_HOTKEY, LOCALSTORAGE_KEY, normalizeAccelerator} from "@/utils/hotkeyUtils";
import {SetHotkey, ToggleShowHide} from "../../../wailsjs/go/hotkey/HotKey";
import {EventsOn} from "../../../wailsjs/runtime";
import {globalHotkeyManager} from "@/utils/globalHotkey";

export default {
  name: 'HotkeyManager',
  data() {
    return {
      hotkeyConfig: {
        showHide: globalHotkeyManager.hotkeyConfig.showHide
      },
      detecting: '',
      hotkeyMessage: '',
      hotkeyMessageType: 'info',
      autoStartEnabled: false,
      autoStartLoading: false,
    }
  },
  async created() {
    try {
      this.OS = await GetOs();
      if (this.OS === "windows") {
        this.loadHotkeyFromStorage();
        await this.applyHotkeyConfig();
        await this.loadAutoStartStatus();
      }
    } catch (error) {
      console.error('获取操作系统类型失败', error);
      this.OS = '';
    }

    // 从路由查询参数中获取tab，如果存在则设置activeTab
    if (this.$route.query.tab) {
      this.activeTab = this.$route.query.tab;
    }

  },
  mounted() {
    if (this.OS === "windows") {
      const enableGlobalListener = () => {
        if (this._hotkeyListenerEnabled) return;

        this._globalKeyHandler = (e) => {
          if (this.detecting) return;

          const active = document.activeElement;
          if (active && (active.tagName === 'INPUT' || active.tagName === 'TEXTAREA' || active.isContentEditable)) {
            return;
          }

          const pressed = accelFromEvent(e);
          if (!pressed) return;

          const stored = normalizeAccelerator(this.hotkeyConfig.showHide || '');
          if (stored && pressed.toLowerCase() === stored.toLowerCase()) {
            if (typeof ToggleShowHide === 'function') {
              ToggleShowHide()
                  .then(() => console.debug('ToggleShowHide called successfully'))
                  .catch(err => {
                    console.error('ToggleShowHide returned error:', err);
                    this.showHotkeyMessage('切换失败: ' + (err?.message || err), 'error');
                  });
            } else {
              console.warn('ToggleShowHide 未找到，请确认 wailsjs 绑定是否生成并正确导入');
            }
          }
        };

        window.addEventListener('keydown', this._globalKeyHandler, true);
        this._hotkeyListenerEnabled = true;
        console.debug('hotkey global listener enabled');
      };

      const sendHotkeyToBackend = async () => {
        try {
          const current = this.hotkeyConfig.showHide || DEFAULT_HOTKEY;
          if (typeof SetHotkey === 'function') {
            await SetHotkey(current);
            console.debug('SetHotkey init:', current);
          } else {
            console.warn('SetHotkey binding 未找到');
          }
        } catch (err) {
          console.warn('SetHotkey init failed:', err);
        }
      };

      try {
        EventsOn('hotkey-ready', async () => {
          console.debug('received hotkey-ready from backend, enabling hotkey listener');
          enableGlobalListener();
          await sendHotkeyToBackend();
        });
      } catch (e) {
        console.warn('EventsOn not available or failed, enabling listener immediately as fallback', e);
        enableGlobalListener();
        sendHotkeyToBackend();
      }

      setTimeout(() => {
        if (!this._hotkeyListenerEnabled) {
          console.debug('fallback: enabling hotkey listener after timeout');
          enableGlobalListener();
        }
        sendHotkeyToBackend();
      }, 1000);
    }
  },
  beforeUnmount() {
    if (this.OS === "windows") {
      if (this._globalKeyHandler) {
        window.removeEventListener('keydown', this._globalKeyHandler, true);
        this._globalKeyHandler = null;
      }
      this._hotkeyListenerEnabled = false;
    }
  },
  methods: {
    // 快捷键相关方法
    loadHotkeyFromStorage() {
      this.hotkeyConfig.showHide = globalHotkeyManager.hotkeyConfig.showHide;
    },

    async applyHotkeyConfig() {
      const norm = normalizeAccelerator(this.hotkeyConfig.showHide || '');
      if (!norm) return;

      try {
        if (typeof SetHotkey === 'function') {
          await SetHotkey(norm);
          console.debug('Hotkey applied on startup:', norm);
        }
      } catch (err) {
        console.error('Failed to apply hotkey on startup:', err);
      }
    },

    async saveHotkeyConfig() {
      if (this.isSaving) return;
      this.isSaving = true;

      const norm = normalizeAccelerator(this.hotkeyConfig.showHide || '');
      if (!norm) {
        this.showHotkeyMessage('快捷键不能为空', 'error');
        this.isSaving = false;
        return false;
      }

      localStorage.setItem(LOCALSTORAGE_KEY, norm);
      globalHotkeyManager.hotkeyConfig.showHide = norm;

      try {
        if (typeof SetHotkey === 'function') {
          await SetHotkey(norm);
          console.debug('Backend SetHotkey called:', norm);
        } else {
          console.warn('SetHotkey binding 未找到');
        }
      } catch (err) {
        console.error('SetHotkey error:', err);
        this.showHotkeyMessage('后端注册全局热键失败: ' + (err?.message || err), 'error');
        this.isSaving = false;
        return false;
      }

      this.showHotkeyMessage('快捷键设置已生效', 'info');
      this.isSaving = false;
      return true;
    },

    resetHotkeyToDefault() {
      const norm = DEFAULT_HOTKEY;
      this.hotkeyConfig.showHide = norm;

      globalHotkeyManager.hotkeyConfig.showHide = norm;
      localStorage.setItem(LOCALSTORAGE_KEY, norm);

      try {
        if (typeof SetHotkey === 'function') {
          SetHotkey(norm).catch(err => console.warn('SetHotkey failed:', err));
        }
      } catch (e) { console.warn(e); }

      this.showHotkeyMessage('已关闭快捷键功能', 'info');
    },

    onKeyDown(event, field) {
      event.preventDefault();
      let key = event.key;
      if (key === ' ') key = 'Space';
      if (key === 'Escape') {
        this.hotkeyConfig[field] = '';
        return;
      }

      const modifiers = [];
      if (event.ctrlKey) modifiers.push('Ctrl');
      if (event.altKey) modifiers.push('Alt');
      if (event.shiftKey) modifiers.push('Shift');
      if (event.metaKey) modifiers.push('Cmd');

      if (['Control', 'Alt', 'Shift', 'Meta'].includes(key)) return;

      const accelerator = modifiers.length > 0 ? modifiers.join('+') + '+' + key : key;
      this.hotkeyConfig[field] = normalizeAccelerator(accelerator);

      if (field === 'showHide') {
        globalHotkeyManager.hotkeyConfig.showHide = this.hotkeyConfig.showHide;
        localStorage.setItem(LOCALSTORAGE_KEY, this.hotkeyConfig.showHide);
        console.debug('Hotkey updated via input:', this.hotkeyConfig.showHide);
      }
    },

    startDetection(field) {
      if (this.detecting) return;
      this.detecting = field;
      this.showHotkeyMessage('请按下您想要的快捷键组合（10秒超时）', 'info');

      const timeoutMs = 10000;
      const keyHandler = (event) => {
        event.preventDefault();
        if (event.key === 'Escape') {
          cleanup();
          this.showHotkeyMessage('按键检测已取消', 'info');
          return;
        }
        let key = event.key;
        if (key === ' ') key = 'Space';
        if (['Control', 'Alt', 'Shift', 'Meta'].includes(key)) return;

        const modifiers = [];
        if (event.ctrlKey) modifiers.push('Ctrl');
        if (event.altKey) modifiers.push('Alt');
        if (event.shiftKey) modifiers.push('Shift');
        if (event.metaKey) modifiers.push('Cmd');

        const accelerator = modifiers.length > 0 ? modifiers.join('+') + '+' + key : key;
        const norm = normalizeAccelerator(accelerator);

        cleanup();
        this.hotkeyConfig[field] = norm;

        globalHotkeyManager.hotkeyConfig.showHide = norm;
        localStorage.setItem(LOCALSTORAGE_KEY, norm);

        this.showHotkeyMessage('按键已识别: ' + norm, 'info');
        console.debug('Hotkey detected:', norm);
      };

      const timeoutId = setTimeout(() => {
        cleanup();
        this.showHotkeyMessage('按键识别超时', 'error');
      }, timeoutMs);

      const cleanup = () => {
        window.removeEventListener('keydown', keyHandler, true);
        clearTimeout(timeoutId);
        this.detecting = '';
      };

      window.addEventListener('keydown', keyHandler, true);
    },

    showHotkeyMessage(text, type) {
      this.hotkeyMessage = text;
      this.hotkeyMessageType = type;
      setTimeout(() => {
        this.hotkeyMessage = '';
      }, 3000);
    },

    async loadAutoStartStatus() {
      try {
        this.autoStartEnabled = await GetAutoStart();
        console.debug('开机自启状态加载成功:', this.autoStartEnabled);
      } catch (error) {
        console.error('获取开机自启状态失败:', error);
        this.autoStartEnabled = false;
      }
    },

    async handleAutoStartChange(enabled) {
      this.autoStartLoading = true;
      try {
        const success = await SetAutoStart(enabled);
        if (success) {
          this.showHotkeyMessage(
              enabled ? '已开启开机自启' : '已关闭开机自启',
              'info'
          );
          this.autoStartEnabled = enabled;
        } else {
          this.showHotkeyMessage(
              enabled ? '开启开机自启失败' : '关闭开机自启失败',
              'error'
          );
          this.autoStartEnabled = !enabled;
        }
      } catch (error) {
        console.error('设置开机自启失败:', error);
        this.showHotkeyMessage('设置开机自启失败: ' + error.message, 'error');
        this.autoStartEnabled = !enabled;
      } finally {
        this.autoStartLoading = false;
      }
    },
  }
}
</script>

<style scoped>
.hotkey-manager {
  padding: 20px;
  max-width: 500px;
  margin: 0 auto;
}

.auto-start-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.auto-start-desc {
  font-size: 12px;
  color: #909399;
}

.hotkey-item {
  margin-bottom: 24px;
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #ebeef5;
}

.hotkey-item label {
  display: block;
  margin-bottom: 12px;
  font-weight: 600;
  color: #303133;
}

@media (max-width: 768px) {
  .auto-start-group {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}

.input-group {
  display: flex;
  gap: 10px;
}

.hotkey-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  font-family: monospace;
}

.hotkey-input:focus {
  outline: none;
  border-color: #409eff;
}

.detect-btn {
  padding: 8px 12px;
}

.button-group {
  display: flex;
  gap: 10px;
  margin-top: 20px;
  justify-content: center;
}

.message {
  margin-top: 15px;
  padding: 10px;
  border-radius: 4px;
  text-align: center;
}

.message.info {
  background-color: #f0f9eb;
  color: #67c23a;
  border: 1px solid #e1f3d8;
}

.message.error {
  background-color: #fef0f0;
  color: #f56c6c;
  border: 1px solid #fde2e2;
}
</style>