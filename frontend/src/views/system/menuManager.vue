<template>
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
                  <el-tooltip
                      v-if="element.name === 'systemManage'"
                      content="隐藏后，系统管理将显示在缩放按钮旁边"
                      placement="top"
                  >
                    <el-switch
                        v-model="element.visible"
                        active-text="显示"
                        inactive-text="隐藏"
                        size="small"
                    />
                  </el-tooltip>
                  <el-switch
                      v-else
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

    <div class="action-buttons">
      <el-button type="primary" @click="saveOrder">保存顺序</el-button>
      <el-button @click="resetOrder">重置为默认顺序</el-button>
    </div>
  </div>
</template>

<script>
import draggable from "vuedraggable";
import {defaultMenu, iconMap, loadMenuOrder, moduleTabsConfig, saveMenuOrder} from "@/utils/menuConfig";
import {markRaw} from "vue";
import {ElMessage} from "element-plus";
import {MoreFilled, Setting} from "@element-plus/icons-vue";

export default {
  name: 'MenuManager',
  components: {Setting, MoreFilled, draggable},
  data() {
    return {
      menuItems: [],
      moduleTabsItems: {},

      _hotkeyListenerEnabled: false,
      _globalKeyHandler: null,
      isSaving: false,
      OS: '',


      // 模块标签页对话框相关
      moduleTabsDialogVisible: false,
      currentModule: null,
      currentModuleTabs: [],
    }
  },
  async created() {
    await this.loadMenuData();
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
      this.currentModuleTabs = this.moduleTabsItems[module.name] || [];
      this.moduleTabsDialogVisible = true;
    },

    // 保存模块标签页设置
    async saveModuleTabs() {
      if (this.currentModule) {
        try {
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

    onModuleTabsDragEnd() {
      this.currentModuleTabs.forEach((item, index) => {
        item.order = index;
      });
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

  }
}
</script>

<style scoped>
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
  user-select: none;
}

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

.action-buttons {
  display: flex;
  gap: 12px;
  justify-content: center;
  margin-top: 20px;
}

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
</style>