<template>
  <div class="app">
    <!-- 原有布局 -->
    <div class="appMenu" v-if="routeName !== 'login'">
      <Menu :routeName="routeName" />
    </div>
    <div class="appMain">
      <!-- 添加缓存功能 -->
      <router-view v-slot="{ Component }">
        <keep-alive :include="cachedPages">
          <component :is="Component" :key="routeName" />
        </keep-alive>
      </router-view>
    </div>
  </div>
</template>

<script>
import Menu from './components/Menu.vue'
import { globalHotkeyManager } from '@/utils/globalHotkey';
import { ToggleShowHide, SetHotkey } from "../wailsjs/go/hotkey/HotKey";
import { EventsOn, EventsOff } from "../wailsjs/runtime";
import {GetOs} from "../wailsjs/go/controller/System";
import {getToken} from "@/utils/token";

export default {
  name: 'App',
  components: {
    Menu
  },
  data() {
    return {
      // 缓存页面列表 - 这些名称必须与页面组件的name属性匹配
      cachedPages: [
        'ToolsView',
        'SiteView',
        'InfoSearchView',
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
      OS:''
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
  },
  beforeUnmount() {
    this.OS = GetOs();
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
/* 原有样式保持不动 */
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

html,
body {
  margin: 0;
  padding: 0;
}

body {
  width: 100vw;
  height: 100vh;
  background: #ffffff;
}

.app {
  display: flex;
  align-items: center;

  .appMenu {
    flex-shrink: 0;
  }

  .appMain {
    height: 100vh;
    overflow: auto;
    flex: 1;
  }
}
</style>