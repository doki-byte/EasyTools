<template>
  <el-container class="app-container" direction="vertical">
    <!-- 标签栏 -->
    <el-tabs v-model="activeTab" class="et-module-tabs" @tab-change="onTabChange" :style="tabsStyle">
      <el-tab-pane label="菜单管理" name="menu-manager" />
      <div v-if="OS === 'windows'">
        <el-tab-pane label="全局管理" name="hotkey-manager" />
      </div>
      <el-tab-pane label="代理配置" name="proxy-config" />
      <el-tab-pane label="AI配置" name="ai-config" />
      <el-tab-pane label="软件中心" name="software-center" />
      <el-tab-pane label="主题背景" name="theme-background" />
      <el-tab-pane label="配置备份" name="backup-manager" />
    </el-tabs>

    <el-main>
      <!-- 菜单管理 -->
      <div v-if="activeTab === 'menu-manager'">
        <MenuManager/>
      </div>

      <!-- 快捷键管理 -->
      <div v-if="activeTab === 'hotkey-manager' && OS === 'windows'" class="hotkey-manager">
        <HotkeyManager/>
      </div>

      <!-- 代理配置内容 -->
      <div v-if="activeTab === 'proxy-config'" class="proxy-config-content">
        <ProxyManager/>
      </div>

      <!-- AI配置 -->
      <div v-if="activeTab === 'ai-config'" class="ai-config-content">
        <AIConfigManager/>
      </div>

      <!-- 软件中心 -->
      <div v-if="activeTab === 'software-center'">
        <SoftwareCenter/>
      </div>

      <!-- 主题背景 -->
      <div v-if="activeTab === 'theme-background'">
        <ThemeBackground/>
      </div>

      <!-- 配置备份 -->
      <div v-if="activeTab === 'backup-manager'">
        <BackupManager/>
      </div>
    </el-main>
  </el-container>
</template>

<script>
import { defineAsyncComponent } from 'vue'
import {GetOs} from "../../wailsjs/go/system/System";
import {House} from "@element-plus/icons-vue";

const MenuManager = defineAsyncComponent(() => import('@/views/system/menuManager.vue'));
const HotkeyManager = defineAsyncComponent(() => import('@/views/system/hotkeyManager.vue'));
const ProxyManager = defineAsyncComponent(() => import('@/views/system/proxyManager.vue'));
const AIConfigManager = defineAsyncComponent(() => import('@/views/system/aiConfigManager.vue'));
const SoftwareCenter = defineAsyncComponent(() => import('@/views/system/softwareCenter.vue'));
const ThemeBackground = defineAsyncComponent(() => import('@/views/system/themeBackground.vue'));
const BackupManager = defineAsyncComponent(() => import('@/views/system/backupManager.vue'));

export default {
  name: "SystemManageView",
  components: { ProxyManager, HotkeyManager, MenuManager, AIConfigManager, SoftwareCenter, ThemeBackground, BackupManager, House },
  data() {
    return {
      activeTab: "menu-manager",
      OS: '',
      
      // 字体颜色配置
      fontColor: '#ffffff',
      fontWeightRatio: 100,
      shadowRatio: 100,
      STORAGE_KEY: 'easytools:theme:background'
    };
  },
  computed: {
    // 计算标签页样式
    tabsStyle() {
      // 计算加粗程度：默认 bold (700)，100% 对应 700，0% 对应 400，200% 对应 900
      const fontWeight = 400 + (this.fontWeightRatio / 100) * 500
      // 计算阴影程度：默认 0px 2px 2px rgba(0, 0, 0, 0.1)，根据比例调整透明度
      const shadowOpacity = (this.shadowRatio / 100) * 0.1
      return {
        '--tabs-font-color': this.fontColor,
        '--tabs-font-weight': fontWeight,
        '--tabs-shadow': `0px 2px 2px rgba(0, 0, 0, ${shadowOpacity})`
      }
    }
  },
  async created() {
    try {
      this.OS = await GetOs();
    } catch (error) {
      console.error('获取操作系统类型失败', error);
      this.OS = '';
    }

    // 初始化时检查URL参数
    this.updateTabFromQuery();
  },
  mounted() {
    // 加载字体颜色配置
    this.loadFontColor();
    
    // 监听路由变化
    this.$watch(
        () => this.$route.query,
        (toQuery, fromQuery) => {
          this.updateTabFromQuery();
        }
    );
    
    // 监听主题配置变化
    window.addEventListener('theme-background-changed', this.handleThemeChange);
  },
  beforeUnmount() {
    window.removeEventListener('theme-background-changed', this.handleThemeChange);
  },
  methods: {
    // 从URL查询参数更新标签页
    updateTabFromQuery() {
      const tab = this.$route.query.tab;
      if (tab && ['menu-manager', 'hotkey-manager', 'proxy-config', 'ai-config', 'software-center', 'theme-background', 'backup-manager'].includes(tab)) {
        this.activeTab = tab;
      }
    },

    // 标签页切换时更新URL
    onTabChange(tabName) {
      // 更新URL但不添加新的历史记录
      this.$router.replace({
        query: { tab: tabName }
      });
    },

    // 加载字体颜色配置
    loadFontColor() {
      try {
        const saved = localStorage.getItem(this.STORAGE_KEY);
        if (saved) {
          const config = JSON.parse(saved);
          if (config.fontColor) {
            this.fontColor = config.fontColor;
          }
          if (config.fontWeightRatio !== undefined) {
            this.fontWeightRatio = config.fontWeightRatio;
          }
          if (config.shadowRatio !== undefined) {
            this.shadowRatio = config.shadowRatio;
          }
        }
      } catch (e) {
        console.error('加载字体颜色配置失败:', e);
      }
    },

    // 监听主题配置变化
    handleThemeChange(event) {
      if (event.detail && event.detail !== null) {
        if (event.detail.fontColor) {
          this.fontColor = event.detail.fontColor;
        }
        if (event.detail.fontWeightRatio !== undefined) {
          this.fontWeightRatio = event.detail.fontWeightRatio;
        }
        if (event.detail.shadowRatio !== undefined) {
          this.shadowRatio = event.detail.shadowRatio;
        }
      } else if (event.detail === null) {
        // 如果配置被清除，恢复默认值
        this.fontColor = '#ffffff';
        this.fontWeightRatio = 100;
        this.shadowRatio = 100;
      } else {
        // 如果没有配置，加载本地存储的配置
        this.loadFontColor();
      }
    }
  }
};
</script>

<style scoped>
/* 移除硬编码背景色，让主题背景透过来 */
.app-container {
  min-height: 96vh;
  display: flex;
  flex-direction: column;
  background-color: transparent;
  padding: 0 10px;
  box-sizing: border-box;
}



:deep(.el-main) {
  --el-main-padding: 0px;
}
:deep(h2) {
  display: block;
  font-size: 1.5em;
  margin-block-start: 0.83em;
  margin-block-end: 0.83em;
  margin-inline-start: 0;
  margin-inline-end: 0;
  font-weight: bold;
  unicode-bidi: isolate;
}
</style>