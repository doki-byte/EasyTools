<template>
  <div class="app">
    <CustomTitleBar/>
    <!-- 主要内容容器 -->
    <div class="main-container" :class="{ 'with-menu': routeName !== 'login' }">
      <!-- 菜单栏 -->
      <div class="appMenu" v-if="routeName !== 'login'">
        <Menu :routeName="routeName" />
      </div>
      <!-- 主内容区域 -->
      <div class="appMain">
        <!-- 添加缓存功能 -->
        <router-view v-slot="{ Component }">
          <keep-alive :include="cachedPages">
            <component :is="Component" :key="routeName" />
          </keep-alive>
        </router-view>
      </div>
    </div>

    <!-- 毛玻璃效果退出确认弹框 -->
    <a-modal
        v-model:visible="exitModalVisible"
        :footer="null"
        :closable="false"
        :maskClosable="false"
        width="320px"
        class="glass-exit-modal"
        :body-style="{ padding: '0' }"
    >
      <div class="glass-dialog">
        <div class="glass-header">
          <div class="app-icon">
            <!-- 这里可以放您的应用图标 -->
            <div><img style="width: 50px; height: 50px; " src="/assets/system/appicon.png" alt=""/></div>
          </div>
          <h3 class="app-name">EasyTools😘</h3>
        </div>

        <div class="glass-content">
          <p class="question">确定要退出吗？🙃( ´･ω･)ﾉ(._.`)摸摸头</p>
        </div>

        <div class="glass-actions">
          <a-button
              class="glass-cancel"
              @click="cancelExit"
          >
            再想想
          </a-button>
          <a-button
              class="glass-confirm"
              @click="confirmExit"
          >
            退出
          </a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script>
import Menu from './components/Menu.vue'
import CustomTitleBar from "./components/CustomTitleBar.vue";
import { globalHotkeyManager } from '@/utils/globalHotkey';
import { ToggleShowHide, SetHotkey } from "../wailsjs/go/hotkey/HotKey";
import { EventsOn, EventsOff } from "../wailsjs/runtime";
import { GetOs, ExitApp } from "../wailsjs/go/system/System";

export default {
  name: 'App',
  components: {
    CustomTitleBar,
    Menu
  },
  data() {
    return {
      // 缓存页面列表 - 这些名称必须与页面组件的name属性匹配
      cachedPages: [
        'ToolsView',
        'SiteView',
        'AssistiveView',
        'InfoDealView',
        'ConnectView',
        'CyberChefView',
        'RandomInfoView',
        'BypassAvView',
        'NoteViews',
        'ProxyView',
        'RestMateView'
      ],
      hotkeyReady: false,
      hotkeyReadyTimeout: null,
      OS: '',
      exitModalVisible: false // 控制退出确认弹框显示
    }
  },
  async created() {
    try {
      this.OS = await GetOs();
      console.log('OS:', this.OS);
    } catch (error) {
      console.error('获取操作系统类型失败', error);
      this.OS = '';
    }
  },
  async mounted() {
    // 等待 OS 信息获取完成
    if (!this.OS) {
      try {
        this.OS = await GetOs();
      } catch (error) {
        console.error('获取操作系统类型失败', error);
        this.OS = '';
      }
    }
    if (this.OS === "windows") {
      // 初始化全局快捷键
      globalHotkeyManager.init();
      // 添加显示/隐藏快捷键监听
      globalHotkeyManager.addListener(this.handleShowHideHotkey);

      // 设置热键到后端
      await this.setHotkeyToBackend();
    }

    // 设置窗口关闭监听
    this.setupWindowCloseListener();
  },
  beforeUnmount() {
    if (this.OS === "windows") {
      // 移除监听器
      globalHotkeyManager.removeListener(this.handleShowHideHotkey);
      globalHotkeyManager.destroy();

      // 清除超时
      if (this.hotkeyReadyTimeout) {
        clearTimeout(this.hotkeyReadyTimeout);
      }

      // 移除事件监听
      try {
        EventsOff('hotkey-ready', this.handleHotkeyReady);
      } catch (error) {
        console.warn('EventsOff not available');
      }
    }

    // 移除窗口关闭事件监听
    this.removeWindowCloseListener();
  },
  methods: {
    async handleShowHideHotkey() {
      try {
        await ToggleShowHide();
      } catch(err) {
        console.error('ToggleShowHide returned error:', err);
      }
    },

    async setHotkeyToBackend() {
      const hotkey = globalHotkeyManager.hotkeyConfig.showHide;
      try {
        // 监听hotkey-ready事件
        EventsOn('hotkey-ready', this.handleHotkeyReady);

        // 设置超时，如果5秒内没有收到事件，则直接设置
        this.hotkeyReadyTimeout = setTimeout(() => {
          if (!this.hotkeyReady) {
            console.debug('hotkey-ready event not received within 5s, setting hotkey directly');
            SetHotkey(hotkey).catch(console.error);
          }
        }, 5000);
      } catch (error) {
        console.warn('EventsOn not available in App.vue, trying SetHotkey without event');
        SetHotkey(hotkey).catch(console.error);
      }
    },

    handleHotkeyReady() {
      this.hotkeyReady = true;
      clearTimeout(this.hotkeyReadyTimeout);
      const hotkey = globalHotkeyManager.hotkeyConfig.showHide;
      SetHotkey(hotkey).catch(console.error);
    },

    // 设置窗口关闭监听
    setupWindowCloseListener() {
      // 监听键盘 ESC 键
      window.addEventListener('keydown', this.handleKeydown);

      // 监听 Wails 的退出事件
      try {
        EventsOn('app-exit', this.handleAppExit);
      } catch (error) {
        console.warn('Wails EventsOn not available for exit events');
      }
    },

    // 移除窗口关闭监听
    removeWindowCloseListener() {
      window.removeEventListener('keydown', this.handleKeydown);

      try {
        EventsOff('app-exit', this.handleAppExit);
      } catch (error) {
        console.warn('Wails EventsOff not available for exit events');
      }
    },

    // 处理键盘事件
    handleKeydown(event) {
      if (event.key === 'Escape') {
        event.preventDefault();
        this.exitModalVisible = true;
      }
    },

    // 处理 Wails 退出事件
    handleAppExit() {
      this.exitModalVisible = true;
    },

    // 取消退出
    cancelExit() {
      this.exitModalVisible = false;
    },

    // 确认退出
    confirmExit() {
      this.exitModalVisible = false;
      // 调用 Wails 退出函数
      try {
        ExitApp();
      } catch (error) {
        console.error('Exit function not available:', error);
      }
    }
  },
  computed: {
    routeName() {
      return this.$route.name
    }
  }
}
</script>

<style scoped lang="scss">
/* 重置样式 */
html, body {
  margin: 0;
  padding: 0;
  height: 100%;
  overflow: hidden;
}

#app {
  height: 95vh;
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

.app {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  background: #f5f7fa;
  position: relative;
}

/* 主内容容器 */
.main-container {
  display: flex;
  flex: 1;
  height: calc(100vh - 30px); /* 减去标题栏高度 */
  overflow: hidden;
}

/* 有菜单时的布局 */
.main-container.with-menu {
  .appMenu {
    width: 130px;
    flex-shrink: 0;
    background: #ffffff;
    border-right: 1px solid #e8e8e8;
    box-shadow: 2px 0 8px rgba(0, 0, 0, 0.05);
    z-index: 100;
    overflow-y: auto;
  }

  .appMain {
    flex: 1;
    background: #ffffff;
    overflow: auto;
    position: relative;
    height: 96vh;
  }
}

/* 无菜单时的布局（登录页面） */
.main-container:not(.with-menu) {
  .appMain {
    flex: 1;
    background: #ffffff;
    overflow: auto;
    position: relative;
  }
}

/* 毛玻璃效果弹框样式 */
.glass-dialog {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
}

.glass-header {
  padding: 24px 24px 16px;
  text-align: center;
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
}

.app-icon {
  width: 48px;
  height: 48px;
  margin: 0 auto 12px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.app-name {
  margin: 0;
  color: #333;
  font-size: 16px;
  font-weight: 600;
}

.glass-content {
  padding: 20px 24px;
  text-align: center;
}

.question {
  margin: 0 0 8px 0;
  color: #333;
  font-size: 15px;
  font-weight: 500;
}

.sub-text {
  margin: 0;
  color: #666;
  font-size: 13px;
}

.glass-actions {
  display: flex;
  gap: 12px;
  padding: 16px 24px 24px;
}

.glass-cancel {
  flex: 1;
  border-radius: 8px;
  border: 1px solid #d9d9d9;
  background: transparent;
  height: 36px;
  transition: all 0.3s ease;
}

.glass-confirm {
  flex: 1;
  border-radius: 8px;
  background: linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%);
  border: none;
  color: white;
  height: 36px;
  transition: all 0.3s ease;
}

.glass-cancel:hover {
  border-color: #4096ff;
  color: #4096ff;
  background: rgba(64, 150, 255, 0.1);
  transform: translateY(-1px);
}

.glass-confirm:hover {
  background: linear-gradient(135deg, #ff8787 0%, #f76707 100%);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(255, 107, 107, 0.3);
}

/* 滚动条美化 */
.appMenu::-webkit-scrollbar,
.appMain::-webkit-scrollbar {
  width: 6px;
}

.appMenu::-webkit-scrollbar-track,
.appMain::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.appMenu::-webkit-scrollbar-thumb,
.appMain::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.appMenu::-webkit-scrollbar-thumb:hover,
.appMain::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .main-container.with-menu {
    .appMenu {
      width: 180px;
    }
  }
}
</style>

<style>
/* 全局样式，美化 Modal */
.glass-exit-modal .ant-modal-content {
  background: transparent;
  box-shadow: none;
  border-radius: 16px;
}

.glass-exit-modal .ant-modal-body {
  padding: 0 !important;
}

.glass-exit-modal .ant-modal-wrap {
  backdrop-filter: blur(4px);
}

.glass-exit-modal .ant-modal-mask {
  background-color: rgba(0, 0, 0, 0.3);
}

/* 确保标题栏在最顶层 */
.custom-titlebar {
  z-index: 1000;
}
</style>