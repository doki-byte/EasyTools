<template>
  <el-container class="container">
    <!-- 标签栏 -->
    <el-tabs v-model="activeTab" class="tabs">
      <el-tab-pane label="菜单管理" name="menu-manager" />
      <div v-if="OS === 'windows'">
        <el-tab-pane label="全局管理" name="hotkey-manager" />
      </div>
    </el-tabs>

    <el-main>
      <!-- 菜单管理 -->
      <div v-if="activeTab === 'menu-manager'">
        <div class="menu-manager">
          <p>拖拽调整菜单项顺序，点击保存应用更改（可切换显示/隐藏）</p>
          <div class="menu-list">
            <draggable
                v-model="menuItems"
                item-key="name"
                @end="onDragEnd"
                handle=".drag-handle"
            >
              <template #item="{ element, index }">
                <div class="menu-item">
                  <el-icon class="drag-handle"><MoreFilled /></el-icon>
                  <el-icon class="menu-icon">
                    <component :is="element.icon" />
                  </el-icon>
                  <span class="menu-title">{{ element.title }}</span>
                  <el-tag type="info" size="small">{{ element.name }}</el-tag>

                  <!-- 可见性开关 -->
                  <div class="visibility-switch">
                    <el-switch
                        v-model="element.visible"
                        active-text="显示"
                        inactive-text="隐藏"
                        size="small"
                    />
                  </div>
                </div>
              </template>
            </draggable>
          </div>

          <div class="action-buttons">
            <el-button type="primary" @click="saveOrder">保存顺序</el-button>
            <el-button @click="resetOrder">重置为默认顺序</el-button>
            <el-button @click="goBack">返回</el-button>
          </div>
        </div>
      </div>

      <!-- 快捷键管理 -->
      <div v-if="activeTab === 'hotkey-manager' && OS === 'windows'" class="hotkey-manager">
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

        <!-- 开机自启设置 -->
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

        <div v-if="hotkeyMessage" :class="['message', hotkeyMessageType]">{{ hotkeyMessage }}</div>
      </div>
    </el-main>
  </el-container>
</template>

<script>
import { markRaw } from 'vue';
import { ElMessage } from 'element-plus';
import draggable from 'vuedraggable';
import { MoreFilled } from '@element-plus/icons-vue';
import { defaultMenu, iconMap, loadMenuOrder, saveMenuOrder } from '@/utils/menuConfig';
import { ToggleShowHide, SetHotkey } from "../../wailsjs/go/hotkey/HotKey";
import { EventsOn } from "../../wailsjs/runtime";
import { globalHotkeyManager } from '@/utils/globalHotkey';
import { LOCALSTORAGE_KEY, DEFAULT_HOTKEY, normalizeAccelerator, accelFromEvent } from '@/utils/hotkeyUtils';
import {GetAutoStart, GetOs, SetAutoStart} from "../../wailsjs/go/controller/System";

export default {
  name: "SystemManageView",
  components: {
    draggable,
    MoreFilled
  },
  data() {
    return {
      menuItems: [],
      activeTab: "menu-manager",
      hotkeyConfig: {
        showHide: globalHotkeyManager.hotkeyConfig.showHide
      },
      detecting: '',
      hotkeyMessage: '',
      hotkeyMessageType: 'info',
      _hotkeyListenerEnabled: false,
      _globalKeyHandler: null,
      isSaving: false, // 添加保存状态标志
      OS: '',
      autoStartEnabled: false, // 新增：开机自启状态
      autoStartLoading: false // 新增：加载状态
    };
  },
  async created() {
    await this.loadMenuData();
    try {
      this.OS = await GetOs();
      if (this.OS === "windows") {
        this.loadHotkeyFromStorage();
        await this.applyHotkeyConfig();
        await this.loadAutoStartStatus(); // 加载开机自启状态
      }
    } catch (error) {
      console.error('获取操作系统类型失败', error);
      this.OS = '';
    }
  },
  mounted() {
    if (this.OS === "windows") {
      // 启用全局监听的函数（延迟到 hotkey-ready）
      const enableGlobalListener = () => {
        if (this._hotkeyListenerEnabled) return;

        this._globalKeyHandler = (e) => {
          if (this.detecting) return;

          // 如果当前元素在输入场景则跳过
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

      // helper: 向后端下发当前热键（如果绑定存在）
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

      // 1) 等待后端发 hotkey-ready 事件（后端在 OnStartup 里 emit）
      try {
        EventsOn('hotkey-ready', async () => {
          console.debug('received hotkey-ready from backend, enabling hotkey listener');
          enableGlobalListener();

          // 后端就绪后把当前 hotkey 配置下发一次，确保后端注册系统级热键
          await sendHotkeyToBackend();
        });
      } catch (e) {
        console.warn('EventsOn not available or failed, enabling listener immediately as fallback', e);
        enableGlobalListener();

        // fallback: 直接尝试下发热键（后端若尚未就绪会报错，已在 sendHotkeyToBackend 内处理）
        sendHotkeyToBackend();
      }

      // 2) 保险机制：若事件错过，1s 后自动启用（避免一直等待）
      setTimeout(() => {
        if (!this._hotkeyListenerEnabled) {
          console.debug('fallback: enabling hotkey listener after timeout');
          enableGlobalListener();
        }

        // 再次尝试确保后端注册（防止前面两步都没下发成功）
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
    async loadMenuData() {
      const savedOrder = await loadMenuOrder();
      let fullMenu = defaultMenu.map(item => {
        const savedItem = savedOrder.find(i => i.name === item.name);
        return {
          ...item,
          order: savedItem ? savedItem.order : item.defaultOrder,
          visible: savedItem
              ? (typeof savedItem.visible === 'boolean' ? savedItem.visible : item.visible)
              : item.visible,
          icon: markRaw(iconMap[item.icon])
        };
      });

      this.menuItems = fullMenu.sort((a, b) => a.order - b.order);
    },

    onDragEnd() {
      this.menuItems.forEach((item, index) => {
        item.order = index;
      });
    },

    async saveOrder() {
      const orderToSave = this.menuItems.map(item => ({
        name: item.name,
        order: item.order,
        visible: typeof item.visible === 'boolean' ? item.visible : true
      }));

      const success = await saveMenuOrder(orderToSave);
      if (success) {
        ElMessage.success('菜单顺序与可见性已保存');
        window.dispatchEvent(new CustomEvent('menu-order-updated', { detail: { time: Date.now() } }));
      } else {
        ElMessage.error('保存失败，请重试');
      }
    },

    resetOrder() {
      this.menuItems = this.menuItems
          .map(item => {
            const def = defaultMenu.find(i => i.name === item.name);
            return {
              ...item,
              order: def.defaultOrder,
              visible: def.visible
            };
          })
          .sort((a, b) => a.order - b.order);

      window.dispatchEvent(new CustomEvent('menu-order-updated', { detail: { time: Date.now(), reset: true } }));
    },

    goBack() {
      this.$router.go(-1);
    },

    loadHotkeyFromStorage() {
      this.hotkeyConfig.showHide = globalHotkeyManager.hotkeyConfig.showHide;
    },

    // 新增：应用热键配置到后端
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

    // 保存按钮点击 - 添加防抖和状态检查
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

      // 通知后端注册全局热键（系统级）
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

    // 恢复默认设置 - 添加防抖和状态检查
    resetHotkeyToDefault() {
      const norm = DEFAULT_HOTKEY;
      this.hotkeyConfig.showHide = norm;

      globalHotkeyManager.hotkeyConfig.showHide = norm;
      localStorage.setItem(LOCALSTORAGE_KEY, norm);

      // 通知后端
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

      // 修改后立即生效
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

        // 立即生效
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


    // 新增：加载开机自启状态
    async loadAutoStartStatus() {
      try {
        this.autoStartEnabled = await GetAutoStart();
        console.debug('开机自启状态加载成功:', this.autoStartEnabled);
      } catch (error) {
        console.error('获取开机自启状态失败:', error);
        this.autoStartEnabled = false;
      }
    },

    // 新增：处理开机自启状态改变
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
          // 恢复原状态
          this.autoStartEnabled = !enabled;
        }
      } catch (error) {
        console.error('设置开机自启失败:', error);
        this.showHotkeyMessage('设置开机自启失败: ' + error.message, 'error');
        // 恢复原状态
        this.autoStartEnabled = !enabled;
      } finally {
        this.autoStartLoading = false;
      }
    },
  }
};
</script>




<style scoped>
/* 页面容器 */
.container {
  height: 100vh;
  display: flex;
  margin-left: 10px;
  flex-direction: column;
  background-color: #f8f9fb;
}

/* 顶部 Tabs 样式 */
.tabs {
  background-color: #ffffff;
  box-shadow: 0px 2px 2px rgba(0, 0, 0, 0.1);
  margin-bottom: 5px;
  /* border-bottom: 2px solid #ebeef5; */
  padding-left: 10px;
  /* 增加左边距 */
  border-radius: 10px 10px 10px 10px;
}

:deep(.el-main) {
  --el-main-padding: 0px;
}

.menu-manager {
  padding: 5px;
  max-width: 700px;
  margin: 0 auto;
}

.menu-list {
  margin: 20px 0;
  border: 1px solid #e6e8eb;
  border-radius: 8px;
  overflow: hidden;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #fff;
  border-bottom: 1px solid #e6e8eb;
  cursor: move;
}

.menu-item:last-child {
  border-bottom: none;
}

.drag-handle {
  margin-right: 12px;
  color: #909399;
  cursor: move;
}

.menu-icon {
  margin-right: 12px;
  color: #606266;
}

.menu-title {
  flex: 1;
  font-size: 14px;
}

.visibility-switch {
  margin-left: 12px;
  display: flex;
  align-items: center;
}

/* 按钮区域 */
.action-buttons {
  display: flex;
  gap: 12px;
  justify-content: center;
  margin-top: 12px;
}

/* 快捷键管理样式 */

:deep(h2) {
  display: block;
  font-size: 1.5em;
  margin-block-start: 0.83em;
  margin-block-end: 0.83em;
  margin-inline-start: 0px;
  margin-inline-end: 0px;
  font-weight: bold;
  unicode-bidi: isolate;
}

.hotkey-manager {
  padding: 20px;
  max-width: 500px;
  margin: 0 auto;
}

/* 新增开机自启样式 */
.auto-start-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.auto-start-desc {
  font-size: 12px;
  color: #909399;
}

/* 调整热单项的间距 */
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

/* 响应式调整 */
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