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
          <!-- 一级菜单 -->
          <div class="menu-section">
            <h3>主菜单管理(左侧支持拖拽)</h3>
            <div class="menu-list">
              <draggable
                  v-model="menuItems"
                  item-key="name"
                  @end="onDragEnd"
                  handle=".drag-handle"
                  tag="div"
              >
                <template #item="{ element }">
                  <div class="menu-item-container">
                    <div class="menu-item">
                      <el-icon class="drag-handle"><MoreFilled /></el-icon>
                      <el-icon class="menu-icon">
                        <component :is="element.icon" />
                      </el-icon>
                      <span class="menu-title">{{ element.title }}</span>
                      <el-tag type="info" size="small">{{ element.name }}</el-tag>

                      <!-- 可见性开关 -->
                      <div class="visibility-switch" @click.stop>
                        <el-switch
                            v-model="element.visible"
                            active-text="显示"
                            inactive-text="隐藏"
                            size="small"
                        />
                      </div>

                      <!-- 管理模块标签页按钮 -->
                      <el-button
                          v-if="hasModuleTabs(element.name)"
                          @click.stop="openModuleTabsDialog(element)"
                          class="manage-btn"
                          size="small"
                          type="primary"
                      >
                        <el-icon><Setting /></el-icon>
                        管理标签页
                      </el-button>
                    </div>
                  </div>
                </template>
              </draggable>
            </div>
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
        <!-- 快捷键管理代码保持不变 -->
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

    <!-- 模块标签页管理对话框 -->
    <el-dialog
        v-model="moduleTabsDialogVisible"
        :title="`${currentModule?.title} - 标签页管理`"
        width="600px"
        :close-on-click-modal="false"
    >
      <div class="module-tabs-dialog">
        <p class="dialog-description">拖拽调整标签页顺序，切换显示/隐藏状态</p>

        <div class="module-tabs-list">
          <draggable
              v-model="currentModuleTabs"
              item-key="name"
              @end="onModuleTabsDragEnd"
              handle=".drag-handle"
              tag="div"
          >
            <template #item="{ element: tabElement }">
              <div class="module-tab-item">
                <el-icon class="drag-handle"><MoreFilled /></el-icon>
                <span class="module-tab-title">{{ tabElement.title }}</span>
                <el-tag type="info" size="small">{{ tabElement.name }}</el-tag>

                <!-- 标签页可见性开关 -->
                <div class="visibility-switch" @click.stop>
                  <el-switch
                      v-model="tabElement.visible"
                      active-text="显示"
                      inactive-text="隐藏"
                      size="small"
                  />
                </div>
              </div>
            </template>
          </draggable>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="closeModuleTabsDialog">取消</el-button>
          <el-button type="primary" @click="saveModuleTabs">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </el-container>
</template>

<script>
import {markRaw} from 'vue';
import {ElMessage} from 'element-plus';
import draggable from 'vuedraggable';
import {MoreFilled, Setting} from '@element-plus/icons-vue';
import {defaultMenu, iconMap, loadMenuOrder, moduleTabsConfig, saveMenuOrder} from '@/utils/menuConfig';
import {SetHotkey, ToggleShowHide} from "../../wailsjs/go/hotkey/HotKey";
import {EventsOn} from "../../wailsjs/runtime";
import {globalHotkeyManager} from '@/utils/globalHotkey';
import {accelFromEvent, DEFAULT_HOTKEY, LOCALSTORAGE_KEY, normalizeAccelerator} from '@/utils/hotkeyUtils';
import {GetAutoStart, GetOs, SetAutoStart} from "../../wailsjs/go/controller/System";

export default {
  name: "SystemManageView",
  components: {
    draggable,
    MoreFilled,
    Setting
  },
  data() {
    return {
      menuItems: [],
      moduleTabsItems: {},
      activeTab: "menu-manager",
      hotkeyConfig: {
        showHide: globalHotkeyManager.hotkeyConfig.showHide
      },
      detecting: '',
      hotkeyMessage: '',
      hotkeyMessageType: 'info',
      _hotkeyListenerEnabled: false,
      _globalKeyHandler: null,
      isSaving: false,
      OS: '',
      autoStartEnabled: false,
      autoStartLoading: false,

      // 模块标签页对话框相关
      moduleTabsDialogVisible: false,
      currentModule: null,
      currentModuleTabs: []
    };
  },
  async created() {
    await this.loadMenuData();
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
    async loadMenuData() {
      try {
        const savedData = await loadMenuOrder();
        const savedOrder = savedData.main || [];
        const savedTabsOrder = savedData.tabs || {};

        // 加载一级菜单
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

        // 加载模块标签页
        this.moduleTabsItems = {};
        Object.keys(moduleTabsConfig).forEach(moduleName => {
          const moduleTabs = moduleTabsConfig[moduleName];
          const savedModuleTabsOrder = savedTabsOrder[moduleName] || [];

          this.moduleTabsItems[moduleName] = moduleTabs.map(item => {
            const savedItem = savedModuleTabsOrder.find(i => i.name === item.name);
            return {
              ...item,
              order: savedItem ? savedItem.order : item.defaultOrder,
              visible: savedItem
                  ? (typeof savedItem.visible === 'boolean' ? savedItem.visible : item.visible)
                  : item.visible
            };
          }).sort((a, b) => a.order - b.order);
        });
      } catch (error) {
        console.error('加载菜单数据失败:', error);
        // 使用默认数据
        this.menuItems = defaultMenu.map(item => ({
          ...item,
          icon: markRaw(iconMap[item.icon])
        })).sort((a, b) => a.defaultOrder - b.defaultOrder);

        this.moduleTabsItems = {};
        Object.keys(moduleTabsConfig).forEach(moduleName => {
          this.moduleTabsItems[moduleName] = moduleTabsConfig[moduleName].map(item => ({
            ...item
          })).sort((a, b) => a.defaultOrder - b.defaultOrder);
        });
      }
    },

    hasModuleTabs(menuName) {
      return this.moduleTabsItems[menuName] && this.moduleTabsItems[menuName].length > 0;
    },

    // 打开模块标签页管理对话框
    openModuleTabsDialog(module) {
      this.currentModule = module;

      // 直接使用原数据的引用，确保响应式
      this.currentModuleTabs = this.moduleTabsItems[module.name] || []; // 直接引用，不深拷贝

      // console.log('打开对话框时的标签页顺序:', this.currentModuleTabs.map(t => ({name: t.name, order: t.order})));

      this.moduleTabsDialogVisible = true;
    },

    // 保存模块标签页设置
    async saveModuleTabs() {
      if (this.currentModule) {
        try {
          // console.log('保存前的标签页顺序:', this.currentModuleTabs.map(t => ({name: t.name, order: t.order})));

          // 直接使用当前数据，不需要额外处理
          // this.moduleTabsItems[this.currentModule.name] 已经通过引用自动更新

          // 获取当前的主菜单和标签页配置
          const mainOrderToSave = this.menuItems.map(item => ({
            name: item.name,
            order: item.order,
            visible: typeof item.visible === 'boolean' ? item.visible : true
          }));

          const tabsOrderToSave = {};
          Object.keys(this.moduleTabsItems).forEach(moduleName => {
            tabsOrderToSave[moduleName] = this.moduleTabsItems[moduleName].map(item => ({
              name: item.name,
              order: item.order,
              visible: typeof item.visible === 'boolean' ? item.visible : true
            }));
          });

          // console.log('要保存的标签页数据:', tabsOrderToSave[this.currentModule.name]);

          // 保存到本地存储
          const success = await saveMenuOrder(mainOrderToSave, tabsOrderToSave);
          if (success) {
            // 派发更新事件，让其他组件重新加载配置
            window.dispatchEvent(new CustomEvent('menu-order-updated'));
            ElMessage.success(`${this.currentModule.title} 标签页设置已保存`);
            this.closeModuleTabsDialog();
          } else {
            ElMessage.error('保存失败，请重试');
          }
        } catch (error) {
          console.error('保存模块标签页失败:', error);
          ElMessage.error('保存失败: ' + error.message);
        }
      }
    },

    onDragEnd() {
      this.menuItems.forEach((item, index) => {
        item.order = index;
      });
    },

    // 简化拖拽处理，模仿一级菜单
    onModuleTabsDragEnd() {
      // 直接更新 order，就像一级菜单那样
      this.currentModuleTabs.forEach((item, index) => {
        item.order = index;
      });

      // console.log('拖拽后的顺序:', this.currentModuleTabs.map(t => ({name: t.name, order: t.order})));
    },

    // 关闭模块标签页管理对话框
    closeModuleTabsDialog() {
      this.moduleTabsDialogVisible = false;
      this.currentModule = null;
      this.currentModuleTabs = [];
    },


    async saveOrder() {
      try {
        const mainOrderToSave = this.menuItems.map(item => ({
          name: item.name,
          order: item.order,
          visible: typeof item.visible === 'boolean' ? item.visible : true
        }));

        const tabsOrderToSave = {};
        Object.keys(this.moduleTabsItems).forEach(moduleName => {
          tabsOrderToSave[moduleName] = this.moduleTabsItems[moduleName].map(item => ({
            name: item.name,
            order: item.order,
            visible: typeof item.visible === 'boolean' ? item.visible : true
          }));
        });

        const success = await saveMenuOrder(mainOrderToSave, tabsOrderToSave);
        if (success) {
          window.dispatchEvent(new CustomEvent('menu-order-updated', {
            detail: {
              time: Date.now(),
              type: 'both'
            }
          }));

          ElMessage.success('保存成功');
        } else {
          ElMessage.error('保存失败，请重试');
        }
      } catch (error) {
        console.error('保存菜单顺序失败:', error);
        ElMessage.error('保存失败: ' + error.message);
      }
    },

    resetOrder() {
      // 重置一级菜单
      this.menuItems = defaultMenu.map(item => ({
        ...item,
        order: item.defaultOrder,
        visible: item.visible,
        icon: markRaw(iconMap[item.icon])
      })).sort((a, b) => a.defaultOrder - b.defaultOrder);

      // 重置模块标签页
      Object.keys(moduleTabsConfig).forEach(moduleName => {
        this.moduleTabsItems[moduleName] = moduleTabsConfig[moduleName].map(item => ({
          ...item,
          order: item.defaultOrder,
          visible: item.visible
        })).sort((a, b) => a.defaultOrder - b.defaultOrder);
      });

      ElMessage.success('已重置为默认顺序');
      window.dispatchEvent(new CustomEvent('menu-order-updated', {
        detail: {
          time: Date.now(),
          reset: true,
          type: 'both'
        }
      }));
    },

    goBack() {
      this.$router.go(-1);
    },

    // 快捷键相关方法保持不变
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
  padding-left: 10px;
  border-radius: 10px 10px 10px 10px;
}

:deep(.el-main) {
  --el-main-padding: 0px;
}

.menu-manager {
  padding: 5px;
  max-width: 800px;
  margin: 0 auto;
}

.menu-section {
  margin-bottom: 20px;
}

.menu-section h3 {
  margin-bottom: 10px;
  color: #303133;
  font-size: 16px;
}

.menu-list {
  margin: 10px 0;
  border: 1px solid #e6e8eb;
  border-radius: 8px;
  overflow: hidden;
  user-select: none; /* 防止文字选择 */
}

/* 菜单项容器 */
.menu-item-container {
  background: #fff;
  border-bottom: 1px solid #e6e8eb;
}

.menu-item-container:last-child {
  border-bottom: none;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
}

/* 只在拖拽手柄上设置可拖拽光标 */
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

.manage-btn {
  margin-left: 12px;
}

/* 按钮区域 */
.action-buttons {
  display: flex;
  gap: 12px;
  justify-content: center;
  margin-top: 20px;
}

/* 模块标签页对话框样式 */
.module-tabs-dialog {
  padding: 10px 0;
}

.dialog-description {
  margin-bottom: 20px;
  color: #606266;
  font-size: 14px;
}

.module-tabs-list {
  border: 1px solid #e6e8eb;
  border-radius: 8px;
  overflow: hidden;
  user-select: none;
}

.module-tab-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #fff;
  border-bottom: 1px solid #f0f0f0;
}

.module-tab-item:last-child {
  border-bottom: none;
}

.module-tab-title {
  flex: 1;
  font-size: 14px;
  margin-right: 12px;
}

.module-tab-item .drag-handle {
  color: #c0c4cc;
  margin-right: 10px;
  cursor: move;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
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