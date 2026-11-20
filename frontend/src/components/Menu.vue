<template>
  <div class="menu" :class="{ collapsed: isCollapsed }" :style="menuStyle">
    <div class="list">
      <el-tooltip
          v-for="(item, index) in visibleMenu"
          :key="item.name"
          :content="item.title"
          placement="right"
          :disabled="!isCollapsed"
          effect="dark"
      >
        <div
            class="item"
            @click="toPage(item.name)"
            :class="{ active: routeName === item.name }"
        >
          <el-icon>
            <component :is="item.icon" />
          </el-icon>
          <span class="menu-text">{{ item.title }}</span>
        </div>
      </el-tooltip>
    </div>

    <!-- 底部按钮区域：系统管理（如果被隐藏）+ 展开/折叠按钮 -->
    <div class="bottom-buttons">
      <!-- 系统管理按钮（仅在隐藏时显示在底部） -->
      <el-tooltip
          v-if="hiddenSystemManage"
          :content="hiddenSystemManage.title"
          placement="top"
          effect="dark"
      >
        <div
            class="system-manage-btn"
            @click="toPage(hiddenSystemManage.name)"
            :class="{ active: routeName === hiddenSystemManage.name }"
        >
          <el-icon>
            <component :is="hiddenSystemManage.icon" />
          </el-icon>
        </div>
      </el-tooltip>

      <!-- 展开/折叠按钮 -->
      <el-tooltip
          :content="isCollapsed ? '展开侧边栏' : '折叠侧边栏'"
          placement="top"
          effect="dark"
      >
        <div class="toggle-btn" @click="toggleCollapse">
          <el-icon>
            <component :is="isCollapsed ? 'DArrowRight' : 'DArrowLeft'" />
          </el-icon>
        </div>
      </el-tooltip>
    </div>

    <!-- 环境检测弹框 -->
    <el-dialog
        v-model="envCheckDialogVisible"
        width="580px"
        :close-on-click-modal="false"
        :close-on-press-escape="false"
        class="glass-env-check-dialog"
        :show-close="false"
    >
      <div class="glass-dialog-content">
        <div class="dialog-header env-check-header">
          <div class="header-decoration"></div>
          <div class="env-check-icon-wrapper">
            <div class="env-check-icon">
              <el-icon size="28"><Warning /></el-icon>
            </div>
            <div class="icon-pulse"></div>
          </div>
          <div class="header-text">
            <h3 class="dialog-title">环境安全检测</h3>
            <p class="dialog-subtitle">Environment Security Check</p>
          </div>
        </div>

        <div class="env-check-content">
          <!-- 虚拟机检测结果 -->
          <div class="check-item" v-if="!vmWarningAlreadyShown">
            <div class="check-item-header">
              <div class="check-type-icon-wrapper">
                <el-icon class="check-type-icon"><Monitor /></el-icon>
              </div>
              <span class="check-type-label">虚拟机检测</span>
            </div>
            <div v-if="vmCheckMessage" class="warning-message">
              <div class="message-icon-wrapper">
                <el-icon class="warning-icon"><WarningFilled /></el-icon>
              </div>
              <div class="message-text">
                <p class="message-title">检测提示</p>
                <p class="message-content">{{ vmCheckMessage }}</p>
              </div>
            </div>
            <div v-else class="safe-message">
              <div class="message-icon-wrapper success">
                <el-icon class="success-icon"><CircleCheckFilled /></el-icon>
              </div>
              <div class="message-text">
                <p class="message-title">检测通过</p>
                <p class="message-content">未检测到虚拟机环境</p>
              </div>
            </div>
          </div>

          <!-- 调试器检测结果 -->
          <div class="check-item">
            <div class="check-item-header">
              <div class="check-type-icon-wrapper">
                <el-icon class="check-type-icon"><Tools /></el-icon>
              </div>
              <span class="check-type-label">调试器检测</span>
            </div>
            <div v-if="debugCheckMessage" class="warning-message">
              <div class="message-icon-wrapper">
                <el-icon class="warning-icon"><WarningFilled /></el-icon>
              </div>
              <div class="message-text">
                <p class="message-title">检测提示</p>
                <p class="message-content">{{ debugCheckMessage }}</p>
              </div>
            </div>
            <div v-else class="safe-message">
              <div class="message-icon-wrapper success">
                <el-icon class="success-icon"><CircleCheckFilled /></el-icon>
              </div>
              <div class="message-text">
                <p class="message-title">检测通过</p>
                <p class="message-content">未检测到调试器环境</p>
              </div>
            </div>
          </div>

          <!-- 提示信息 -->
          <div class="env-check-tips" v-if="vmCheckMessage || debugCheckMessage">
            <div class="tip-item">
              <el-icon class="tip-icon"><InfoFilled /></el-icon>
              <span>建议在真实环境中使用以获得最佳体验</span>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="glass-dialog-footer env-check-footer">
          <el-button
              class="footer-btn confirm-btn"
              type="primary"
              @click="confirmEnvCheck"
          >
            <el-icon style="margin-right: 6px;"><Check /></el-icon>
            我知道了
          </el-button>
        </div>
      </template>
    </el-dialog>

  </div>
</template>

<script setup>
import {computed, markRaw, nextTick, onMounted, onUnmounted, reactive, ref, watch} from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { ElMessage } from 'element-plus';
import { defaultMenu, iconMap, loadMenuOrder, getFirstActiveMenuName } from '@/utils/menuConfig';
import { Warning, CircleCheckFilled, Monitor, Tools, InfoFilled, DArrowLeft, DArrowRight } from '@element-plus/icons-vue';
import { BrowserOpenURL } from "../../wailsjs/runtime";

// 当前路由名称
const routeName = ref('tool');
const router = useRouter();

// 监听路由变化，自动更新 routeName
watch(() => router.currentRoute.value, (newRoute) => {
  if (newRoute.name) {
    routeName.value = newRoute.name;
  }
}, { immediate: true });

// 响应式菜单项
const menuList = ref([]);

// 侧边栏展开/折叠状态
const SIDEBAR_COLLAPSED_KEY = 'easytools:sidebar:collapsed';
const isCollapsed = ref(false);

// 加载侧边栏状态
const loadSidebarState = () => {
  try {
    const saved = localStorage.getItem(SIDEBAR_COLLAPSED_KEY);
    if (saved !== null) {
      isCollapsed.value = JSON.parse(saved);
    }
  } catch (e) {
    console.error('加载侧边栏状态失败:', e);
  }
};

// 保存侧边栏状态
const saveSidebarState = () => {
  try {
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, JSON.stringify(isCollapsed.value));
    // 触发自定义事件，通知 App.vue 更新宽度
    window.dispatchEvent(new CustomEvent('sidebar-toggle', { 
      detail: { collapsed: isCollapsed.value } 
    }));
  } catch (e) {
    console.error('保存侧边栏状态失败:', e);
  }
};

// 切换展开/折叠
const toggleCollapse = () => {
  isCollapsed.value = !isCollapsed.value;
  saveSidebarState();
};

// 环境检测相关
const envCheckDialogVisible = ref(false);
const vmCheckMessage = ref('');
const debugCheckMessage = ref('');
const vmWarningAlreadyShown = ref(false); // 标记是否已在 Login 时显示过虚机警告

// 密码验证规则
const pwdRules = {
  oldPassword: [{ required: true, message: "请输入原密码", trigger: "blur" }],
  newPassword: [
    { required: true, message: "请输入新密码", trigger: "blur" },
    { min: 6, message: "密码长度至少6位", trigger: "blur" }
  ],
  confirmPassword: [
    { validator: (rule, value, callback) => {
        if (value !== pwdForm.newPassword) {
          callback(new Error("两次输入密码不一致"));
        } else {
          callback();
        }
      }, trigger: "blur" }
  ]
};

// 计算属性
const sortedMenuList = computed(() => {
  return menuList.value.slice().sort((a, b) => a.order - b.order);
});

const visibleMenu = computed(() => {
  const list = Array.isArray(sortedMenuList.value) ? sortedMenuList.value : [];
  // 过滤出可见的菜单，但排除系统管理（如果被隐藏，会显示在底部）
  return list.filter(i => {
    if (!i) return false;
    // 系统管理菜单如果被隐藏，不显示在主菜单中
    if (i.name === 'systemManage' && i.visible === false) {
      return false;
    }
    return i.visible === undefined ? true : !!i.visible;
  });
});

// 获取被隐藏的系统管理菜单项
const hiddenSystemManage = computed(() => {
  const systemManage = menuList.value.find(i => i && i.name === 'systemManage');
  if (systemManage && systemManage.visible === false) {
    return systemManage;
  }
  return null;
});


// 加载菜单配置
const loadMenu = async () => {
  const savedData = await loadMenuOrder();
  const savedOrder = savedData.main || []; // 取 main 数组，如果不存在则用空数组

  menuList.value = defaultMenu.map(item => {
    const savedItem = savedOrder.find(i => i.name === item.name);
    return {
      ...item,
      order: savedItem ? savedItem.order : item.defaultOrder,
      visible: savedItem ? (typeof savedItem.visible === 'boolean' ? savedItem.visible : item.visible) : item.visible,
      icon: markRaw(iconMap[item.icon])
    };
  });
};


// 事件处理
const handleMenuUpdated = (ev) => {
  loadMenu();
};

// 页面导航
function toPage(name) {
  if (routeName.value === name) {
    return;
  }
  routeName.value = name;
  router.push({ name }).catch(() => { });
}


// 环境检测提示
const checkEnvironment = () => {
  vmCheckMessage.value = '';
  vmWarningAlreadyShown.value = false;
  debugCheckMessage.value = '';
  envCheckDialogVisible.value = true;
}

const confirmEnvCheck = () => {
  envCheckDialogVisible.value = false;
  vmCheckMessage.value = '';
  debugCheckMessage.value = '';
}


// 字体颜色配置
const fontColor = ref('#26384e');
const fontWeightRatio = ref(100);
const shadowRatio = ref(100);
const STORAGE_KEY = 'easytools:theme:background';

// 加载字体颜色配置
const loadFontColor = () => {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      const config = JSON.parse(saved);
      if (config.fontColor) {
        fontColor.value = config.fontColor;
      }
      if (config.fontWeightRatio !== undefined) {
        fontWeightRatio.value = config.fontWeightRatio;
      }
      if (config.shadowRatio !== undefined) {
        shadowRatio.value = config.shadowRatio;
      }
    }
  } catch (e) {
    console.error('加载字体颜色配置失败:', e);
  }
};

// 计算菜单样式
const menuStyle = computed(() => {
  // 计算加粗程度：默认 400，100% 对应 400，0% 对应 300，200% 对应 600
  const fontWeight = 300 + (fontWeightRatio.value / 100) * 300
  // 计算阴影程度：默认 0 4px 12px rgba(0, 0, 0, 0.15)，根据比例调整透明度
  const shadowOpacity = (shadowRatio.value / 100) * 0.15
  return {
    '--menu-font-color': fontColor.value,
    '--menu-font-weight': fontWeight,
    '--menu-shadow': `0 4px 12px rgba(0, 0, 0, ${shadowOpacity})`,
    color: fontColor.value
  };
});

// 监听主题配置变化
const handleThemeChange = (event) => {
  if (event.detail && event.detail !== null) {
    if (event.detail.fontColor) {
      fontColor.value = event.detail.fontColor;
    }
    if (event.detail.fontWeightRatio !== undefined) {
      fontWeightRatio.value = event.detail.fontWeightRatio;
    }
    if (event.detail.shadowRatio !== undefined) {
      shadowRatio.value = event.detail.shadowRatio;
    }
  } else if (event.detail === null) {
    // 如果配置被清除，恢复默认值
    fontColor.value = '#26384e';
    fontWeightRatio.value = 100;
    shadowRatio.value = 100;
  } else {
    // 如果没有配置，加载本地存储的配置
    loadFontColor();
  }
};

// 生命周期
onMounted(async () => {
  if (!routeName.value) {
    // 获取第一个激活的菜单项
    const firstActiveMenu = await getFirstActiveMenuName();
    routeName.value = firstActiveMenu;
    router.push({ name: firstActiveMenu }).catch(() => { });
  }
  loadMenu();
  loadFontColor(); // 加载字体颜色配置
  loadSidebarState(); // 加载侧边栏状态

  window.addEventListener('menu-order-updated', handleMenuUpdated);
  window.addEventListener('theme-background-changed', handleThemeChange);
});

onUnmounted(() => {
  window.removeEventListener('menu-order-updated', handleMenuUpdated);
  window.removeEventListener('theme-background-changed', handleThemeChange);
});
</script>

<style scoped lang="scss">
.menu {
  height: 100%; /* 由 flex 容器拉伸，避免 vh 计算与标题栏(28px)差值导致的底部留缝/溢出 */
  overflow-y: auto; /* 菜单项多时自身滚动，不撑破容器 */
  padding: 0 10px;
  background: transparent;
  color: var(--menu-font-color, #26384e);
  display: flex;
  flex-direction: column;
  align-items: center;
  transition: all 0.3s ease;

  // 底部按钮容器
  .bottom-buttons {
    margin-top: auto;
    margin-bottom: 8px;
    display: flex;
    flex-direction: row;
    align-items: center;
    gap: 8px;
    align-self: flex-end;
    margin-right: 5px;
  }

  // 系统管理按钮（隐藏时显示在底部）
  .system-manage-btn {
    width: 28px;
    height: 28px;
    cursor: pointer;
    border-radius: 4px;
    display: flex;
    justify-content: center;
    align-items: center;
    background: rgba(0, 0, 0, 0.02);
    transition: all 0.3s ease;
    color: var(--menu-font-color, #26384e);
    opacity: 0.5;

    &:hover {
      background: rgba(0, 0, 0, 0.08);
      opacity: 0.8;
      transform: scale(1.1);
    }

    &.active {
      background: linear-gradient(135deg, #fc3d48 0%, #e02e38 100%);
      color: #ffffff;
      opacity: 1;
      box-shadow: 0 2px 8px rgba(252, 61, 72, 0.3);
    }

    .el-icon {
      font-size: 14px;
    }
  }

  // 切换按钮样式 - 放在底部角落，不显眼
  .toggle-btn {
    width: 28px;
    height: 28px;
    cursor: pointer;
    border-radius: 4px;
    display: flex;
    justify-content: center;
    align-items: center;
    background: rgba(0, 0, 0, 0.02);
    transition: all 0.3s ease;
    color: var(--menu-font-color, #26384e);
    opacity: 0.4;

    &:hover {
      background: rgba(0, 0, 0, 0.08);
      opacity: 0.7;
      transform: scale(1.1);
    }

    .el-icon {
      font-size: 12px;
    }
  }

  // 折叠状态
  &.collapsed {
    padding: 0 5px;

    .bottom-buttons {
      margin-right: 0;
    }

    .toggle-btn {
      width: 28px;
    }

    .item {
      width: 50px;
      min-width: 50px;

      .menu-text {
        opacity: 0;
        max-width: 0;
        margin-left: 0;
        overflow: hidden;
        transition: opacity 0.2s ease, max-width 0.3s ease, margin 0.3s ease;
      }

      el-icon {
        margin-right: 0;
      }
    }

  }

  .list {
    display: flex;
    flex-direction: column;
    align-items: center;
    flex-grow: 1;
  }

  .item {
    margin-top: 15px;
    cursor: pointer;
    width: 100px;
    height: 40px;
    line-height: 40px;
    text-align: center;
    font-size: 14px;
    border-radius: 10px;
    display: flex;
    justify-content: center;
    align-items: center;
    font-weight: var(--menu-font-weight, 400);
    background: linear-gradient(135deg, rgba(255, 255, 255, 0.15) 0%, rgba(255, 255, 255, 0.1) 100%);
    backdrop-filter: blur(20px) saturate(150%);
    -webkit-backdrop-filter: blur(20px) saturate(150%);
    box-shadow: 
      0 2px 8px rgba(0, 0, 0, 0.08),
      inset 0 1px 0 rgba(255, 255, 255, 0.3),
      0 0 0 1px rgba(255, 255, 255, 0.1);
    border: none;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);

    .menu-text {
      margin-left: 5px;
      opacity: 1;
      max-width: 200px;
      white-space: nowrap;
      overflow: hidden;
      transition: opacity 0.3s ease 0.05s, max-width 0.3s ease, margin 0.3s ease;
      display: inline-block;
    }

    el-icon {
      display: inline-block;
      margin-right: 10px;
      transition: margin 0.3s ease;
    }

    &:hover {
      background: linear-gradient(135deg, rgba(255, 255, 255, 0.25) 0%, rgba(255, 255, 255, 0.15) 100%);
      box-shadow: 
        0 4px 12px rgba(0, 0, 0, 0.12),
        inset 0 1px 0 rgba(255, 255, 255, 0.4),
        0 0 0 1px rgba(255, 255, 255, 0.15);
      transform: translateY(-1px);
      color: #4f5a6b;
    }

    &.active {
      background: linear-gradient(135deg, #fc3d48 0%, #e02e38 100%);
      color: #ffffff;
      box-shadow: 
        0 4px 16px rgba(252, 61, 72, 0.4),
        inset 0 1px 0 rgba(255, 255, 255, 0.2);
    }
  }

}

/* 毛玻璃对话框样式 */
.glass-dialog-content {
  background: transparent;
  backdrop-filter: none;
  border-radius: 0;
  border: none;
  box-shadow: none;
  overflow: hidden;
  transition: all 0.3s ease;
}

.dialog-header {
  display: flex;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  position: relative;
  overflow: hidden;
}

.update-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.password-header {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.env-check-header {
  background: linear-gradient(135deg, #ff9800 0%, #f57c00 100%);
  position: relative;
  overflow: hidden;
}

.env-check-header::before {
  content: '';
  position: absolute;
  top: -50%;
  right: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.1) 0%, transparent 70%);
  animation: rotate 20s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.header-decoration {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.5), transparent);
  animation: shimmer 2s ease-in-out infinite;
}

@keyframes shimmer {
  0%, 100% {
    opacity: 0.3;
  }
  50% {
    opacity: 1;
  }
}

.env-check-icon-wrapper {
  position: relative;
  margin-right: 16px;
  z-index: 2;
}

.env-check-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.25);
  backdrop-filter: blur(10px);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 2;
  transition: all 0.3s ease;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.env-check-icon:hover {
  transform: scale(1.05);
  background: rgba(255, 255, 255, 0.3);
}

.env-check-icon .el-icon {
  color: white;
  font-size: 28px;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.2));
}

.icon-pulse {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.3);
  animation: pulse 2s ease-in-out infinite;
  z-index: 1;
}

@keyframes pulse {
  0%, 100% {
    transform: translate(-50%, -50%) scale(1);
    opacity: 0.5;
  }
  50% {
    transform: translate(-50%, -50%) scale(1.2);
    opacity: 0;
  }
}

.dialog-subtitle {
  margin: 0;
  color: rgba(255, 255, 255, 0.85);
  font-size: 12px;
  font-weight: 400;
  letter-spacing: 1px;
  text-transform: uppercase;
}

.env-check-content {
  padding: 24px;
  background: linear-gradient(to bottom, #fafafa 0%, #ffffff 100%);
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.check-item {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.check-item-header {
  display: flex;
  align-items: center;
  gap: 12px;
  font-weight: 600;
  font-size: 15px;
  color: #333;
  margin-bottom: 4px;
}

.check-type-icon-wrapper {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: linear-gradient(135deg, #ff9800 0%, #f57c00 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 6px rgba(255, 152, 0, 0.3);
}

.check-type-icon {
  font-size: 18px;
  color: white;
}

.check-type-label {
  color: #333;
  font-weight: 600;
}

.warning-message {
  display: flex;
  gap: 16px;
  padding: 18px;
  background: linear-gradient(135deg, #fff8e1 0%, #ffe0b2 100%);
  border: 2px solid #ffb74d;
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(255, 152, 0, 0.15);
  transition: all 0.3s ease;
}

.warning-message:hover {
  box-shadow: 0 6px 16px rgba(255, 152, 0, 0.25);
  transform: translateY(-2px);
}

.message-icon-wrapper {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: linear-gradient(135deg, #ff9800 0%, #f57c00 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 8px rgba(255, 152, 0, 0.3);
}

.message-icon-wrapper.success {
  background: linear-gradient(135deg, #4caf50 0%, #388e3c 100%);
  box-shadow: 0 4px 8px rgba(76, 175, 80, 0.3);
}

.warning-message .warning-icon {
  font-size: 22px;
  color: white;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.2));
}

.message-text {
  flex: 1;
}

.message-title {
  margin: 0 0 6px 0;
  color: #e65100;
  font-size: 14px;
  font-weight: 600;
}

.message-content {
  margin: 0;
  color: #bf360c;
  font-size: 13px;
  line-height: 1.7;
}

.safe-message {
  display: flex;
  gap: 16px;
  padding: 18px;
  background: linear-gradient(135deg, #e8f5e9 0%, #c8e6c9 100%);
  border: 2px solid #81c784;
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(76, 175, 80, 0.15);
  transition: all 0.3s ease;
}

.safe-message:hover {
  box-shadow: 0 6px 16px rgba(76, 175, 80, 0.25);
  transform: translateY(-2px);
}

.safe-message .message-title {
  color: #2e7d32;
}

.safe-message .message-content {
  color: #1b5e20;
}

.safe-message .success-icon {
  font-size: 22px;
  color: white;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.2));
}

.env-check-tips {
  padding: 14px 18px;
  background: linear-gradient(135deg, #e3f2fd 0%, #bbdefb 100%);
  border-left: 4px solid #2196f3;
  border-radius: 10px;
  margin-top: 8px;
}

.tip-item {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #1565c0;
  font-size: 13px;
  line-height: 1.6;
}

.tip-icon {
  font-size: 16px;
  color: #2196f3;
  flex-shrink: 0;
}

.env-check-footer {
  background: transparent;
}

.confirm-btn {
  background: linear-gradient(135deg, #ff9800 0%, #f57c00 100%);
  border: none;
  color: white;
  box-shadow: 0 4px 12px rgba(255, 152, 0, 0.4);
  font-weight: 600;
  padding: 10px 24px;
  height: auto;
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.confirm-btn::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  width: 0;
  height: 0;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.3);
  transform: translate(-50%, -50%);
  transition: width 0.6s, height 0.6s;
}

.confirm-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, #fb8c00 0%, #ef6c00 100%);
  box-shadow: 0 6px 16px rgba(255, 152, 0, 0.5);
  transform: translateY(-2px);
}

.confirm-btn:hover:not(:disabled)::before {
  width: 300px;
  height: 300px;
}

.confirm-btn:active:not(:disabled) {
  transform: translateY(0);
  box-shadow: 0 2px 8px rgba(255, 152, 0, 0.4);
}

.confirm-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.update-icon-wrapper {
  position: relative;
  margin-right: 12px;
}

.update-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 2;
  transition: all 0.2s ease;
}

.update-icon .el-icon {
  color: white;
  font-size: 20px;
}

.password-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
  position: relative;
  z-index: 2;
  transition: all 0.2s ease;
  
  .el-icon {
    color: white;
    font-size: 20px;
  }
}

.header-text {
  flex: 1;
  z-index: 2;
  position: relative;
}

.dialog-title {
  margin: 0;
  color: white;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.3px;
}

.update-content {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px 20px;
  max-height: 50vh;
  overflow-y: auto;
  background: transparent;
  
  /* 美化滚动条 */
  &::-webkit-scrollbar {
    width: 6px;
  }
  
  &::-webkit-scrollbar-track {
    background: rgba(0, 0, 0, 0.03);
    border-radius: 3px;
  }
  
  &::-webkit-scrollbar-thumb {
    background: rgba(102, 126, 234, 0.4);
    border-radius: 3px;
    
    &:hover {
      background: rgba(102, 126, 234, 0.6);
    }
  }
}

.version-section,
.description-section,
.url-section {
  display: flex;
  align-items: flex-start;
}

.section-label {
  min-width: 75px;
  font-weight: 500;
  color: #666;
  margin-right: 10px;
  font-size: 13px;
  flex-shrink: 0;
}

.version-tag {
  background: linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%);
  color: white;
  border: none;
  font-weight: 600;
  padding: 4px 12px;
  font-size: 12px;
}

.description-section {
  flex-direction: column;

  .description-text {
    margin-left: 0;
    margin-top: 8px;
    width: 100%;
    max-height: 200px;
    overflow: hidden;
    border-radius: 0;
    background: transparent;
    border: none;
    padding: 0;
  }

  .description-text :deep(.el-scrollbar__wrap) {
    overflow-x: hidden;
    padding-right: 8px;
  }

  .description-text :deep(.el-scrollbar__bar) {
    opacity: 0.6;
  }

  .description-text :deep(.el-scrollbar__thumb) {
    background-color: rgba(102, 126, 234, 0.4);
    border-radius: 4px;
    
    &:hover {
      background-color: rgba(102, 126, 234, 0.6);
    }
  }

  .description-line {
    white-space: pre-wrap;
    word-wrap: break-word;
    margin: 4px 0;
    line-height: 1.5;
    color: #555;
    padding: 6px 0 6px 10px;
    border-left: 2px solid #667eea;
    font-size: 13px;
  }
}

.url-section {
  flex-direction: column;

  .url-text {
    margin-left: 0;
    margin-top: 8px;
    width: 100%;
    word-break: break-all;
  }

  .url-tag {
    word-break: break-all;
    cursor: pointer;
    transition: all 0.2s ease;
    border: 1px solid #667eea;
    color: #667eea;
    max-width: 100%;
    display: inline-block;
    padding: 4px 10px;
    border-radius: 4px;
    font-size: 12px;
    background: rgba(102, 126, 234, 0.05);

    &:hover {
      background: #667eea;
      color: white;
    }
  }

  .no-url {
    color: #999;
    font-style: italic;
  }
}

.glass-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  background: transparent;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 0;
}

.password-footer {
  background: transparent;
}

.footer-btn {
  border-radius: 6px;
  min-width: 80px;
  height: 32px;
  font-weight: 500;
  font-size: 13px;
  transition: all 0.2s ease;
  padding: 0 16px;
  
  &:active {
    transform: scale(0.98);
  }
}

.skip-btn {
  border: 1px solid #d9d9d9;
  color: #666;
  background: #fff;

  &:hover {
    border-color: #4096ff;
    color: #4096ff;
  }
}

.cancel-btn {
  border: 1px solid #d9d9d9;
  color: #666;
  background: #fff;

  &:hover:not(:disabled) {
    border-color: #ff4d4f;
    color: #ff4d4f;
  }
  
  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.download-btn {
  background: linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%);
  border: none;
  color: white;
  box-shadow: 0 2px 8px rgba(255, 107, 107, 0.3);

  &:hover:not(:disabled) {
    background: linear-gradient(135deg, #ff8787 0%, #f76707 100%);
    box-shadow: 0 3px 12px rgba(255, 107, 107, 0.4);
  }
  
  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}

.confirm-btn {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  border: none;
  color: white;
  box-shadow: 0 2px 8px rgba(79, 172, 254, 0.3);

  &:hover:not(:disabled) {
    background: linear-gradient(135deg, #5fb8ff 0%, #1af5ff 100%);
    box-shadow: 0 3px 12px rgba(79, 172, 254, 0.4);
  }
  
  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}

.password-form-wrapper {
  padding: 16px 20px;
  background: transparent;
}

.password-form {
  padding: 0;
}

.password-form :deep(.el-form-item) {
  margin-bottom: 16px;
  
  &:last-child {
    margin-bottom: 0;
  }
}

.password-form :deep(.el-form-item__label) {
  color: #666;
  font-weight: 500;
  font-size: 13px;
  padding-bottom: 6px;
}

.password-form :deep(.el-input__wrapper) {
  border-radius: 6px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
  background: #fff;
  
  &:hover {
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.12);
  }
  
  &.is-focus {
    box-shadow: 0 2px 8px rgba(79, 172, 254, 0.25);
  }
}

.password-form :deep(.el-input__inner) {
  padding-left: 10px;
  font-size: 13px;
}

.password-form :deep(.el-input__prefix) {
  left: 10px;
  color: #4facfe;
  font-size: 14px;
}

.password-form :deep(.el-input.is-disabled .el-input__wrapper) {
  background: rgba(245, 247, 250, 0.8);
  cursor: not-allowed;
}

.form-input {
  transition: all 0.3s ease;
}
</style>

<style>
/* 全局样式，美化 Element Plus Dialog */
.glass-update-dialog .el-dialog,
.glass-password-dialog .el-dialog,
.glass-env-check-dialog .el-dialog {
  background: rgba(255, 255, 255, 0.98) !important;
  backdrop-filter: blur(20px) !important;
  -webkit-backdrop-filter: blur(20px) !important;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15) !important;
  border-radius: 20px !important;
  overflow: hidden;
  border: none !important;
}

:deep(.el-dialog){
  padding: 0;
}

.glass-update-dialog .el-dialog__header,
.glass-password-dialog .el-dialog__header,
.glass-env-check-dialog .el-dialog__header {
  display: none;
}

.glass-update-dialog .el-dialog__body,
.glass-password-dialog .el-dialog__body,
.glass-env-check-dialog .el-dialog__body {
  padding: 0 !important;
  background: transparent !important;
}

.glass-update-dialog .glass-dialog-content,
.glass-password-dialog .glass-dialog-content,
.glass-env-check-dialog .glass-dialog-content {
  background: transparent !important;
}

.glass-update-dialog .el-dialog__footer,
.glass-password-dialog .el-dialog__footer,
.glass-env-check-dialog .el-dialog__footer {
  padding: 0 !important;
  background: transparent;
}

.glass-update-dialog .el-overlay,
.glass-password-dialog .el-overlay,
.glass-env-check-dialog .el-overlay {
  backdrop-filter: blur(8px) !important;
  -webkit-backdrop-filter: blur(8px) !important;
  background: rgba(0, 0, 0, 0.3) !important;
}

.glass-update-dialog .el-overlay-dialog,
.glass-password-dialog .el-overlay-dialog,
.glass-env-check-dialog .el-overlay-dialog {
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 新增进度条样式 */
.progress-section {
  display: flex;
  flex-direction: column;
  padding: 0;
  background: transparent;
  border-radius: 0;
  border: none;

  .progress-content {
    margin-left: 0;
    margin-top: 8px;

    .progress-text {
      margin-top: 6px;
      font-size: 12px;
      color: #667eea;
      text-align: center;
      font-weight: 500;
    }
  }
  
  :deep(.el-progress-bar__outer) {
    background: rgba(102, 126, 234, 0.1);
    border-radius: 10px;
  }
  
  :deep(.el-progress-bar__inner) {
    background: linear-gradient(90deg, #667eea, #764ba2);
    border-radius: 10px;
  }
}

/* 调整下载按钮样式 */
.download-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.cancel-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Element Plus Tooltip 样式优化 - 更小巧 */
:deep(.el-tooltip__popper) {
  font-size: 11px !important;
  padding: 3px 6px !important;
  max-width: 200px;
  line-height: 1.3;
  border-radius: 4px;
}

:deep(.el-tooltip__popper.is-dark) {
  background-color: rgba(0, 0, 0, 0.75) !important;
}

:deep(.el-tooltip__popper.is-light) {
  background-color: #fff !important;
  border: 1px solid #e4e7ed !important;
  color: #606266 !important;
}
</style>
