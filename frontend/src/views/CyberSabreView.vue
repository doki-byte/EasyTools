<template>
  <el-container class="app-container"  direction="vertical">
    <!-- 标签栏 -->
    <el-tabs v-model="activeTab" class="et-module-tabs" :key="tabsKey" :style="tabsStyle">
      <el-tab-pane
          v-for="tab in visibleTabs"
          :key="tab.name"
          :label="tab.title"
          :name="tab.name"
      />
    </el-tabs>

    <el-main class="cyber-sabre-main">
      <keep-alive>
        <component :is="currentComponent" />
      </keep-alive>
    </el-main>
  </el-container>
</template>

<script>
import { defineAsyncComponent } from 'vue'
import {loadMenuOrder, moduleTabsConfig} from "@/utils/menuConfig";
import { tabNavigationBridge } from '@/utils/tabNavigationBridge'

/* 子模块全部异步：避免进入渗透测试时一次性打包/解析 8 个巨型 SFC */
const Dirsearch = defineAsyncComponent(() => import("@/views/cyberSabre/DirsearchView.vue"));
const FingerprintScanner = defineAsyncComponent(() => import("@/views/cyberSabre/FingerprintView.vue"));
const HostscanView = defineAsyncComponent(() => import("@/views/cyberSabre/HostscanView.vue"));
const WebscanView = defineAsyncComponent(() => import("@/views/cyberSabre/WebscanView.vue"));
const PocManagementView = defineAsyncComponent(() => import("@/views/cyberSabre/PocManagementView.vue"));
const BrowserScanView = defineAsyncComponent(() => import("@/views/cyberSabre/BrowserScanView.vue"));
const PasswordCrackView = defineAsyncComponent(() => import("@/views/cyberSabre/PasswordCrackView.vue"));
const DruidView = defineAsyncComponent(() => import("@/views/cyberSabre/DruidView.vue"));

export default {
  name: "CyberSabreView",
  mixins: [tabNavigationBridge('cyberSabre')],
  components: {
    Dirsearch,
    FingerprintScanner,
    HostscanView,
    WebscanView,
    PocManagementView,
    BrowserScanView,
    PasswordCrackView,
    DruidView
  },
  data() {
    return {
      activeTab: "", // 初始为空，等配置加载后设置
      moduleTabs: [], // 模块标签页配置
      tabsKey: Date.now(),

      // 添加配置加载状态
      configLoaded: false,
      
      // 字体颜色配置
      fontColor: '#ffffff',
      fontWeightRatio: 100,
      shadowRatio: 100,
      STORAGE_KEY: 'easytools:theme:background'
    }
  },
  computed: {
    // 计算可见的标签页 - 模仿 menu.vue 的实现
    visibleTabs() {
      if (!this.moduleTabs || this.moduleTabs.length === 0) {
        return [];
      }

      // 先过滤可见的，再排序
      const visible = this.moduleTabs.filter(tab => tab.visible !== false);
      return visible.sort((a, b) => (a.order || 0) - (b.order || 0));
    },

    // 当前标签对应的组件，配合 keep-alive 做缓存，避免切换标签时销毁状态
    currentComponent() {
      const map = {
        dirsearch: 'Dirsearch',
        fingerprint: 'FingerprintScanner',
        hostscan: 'HostscanView',
        webscan: 'WebscanView',
        pocManagement: 'PocManagementView',
        'browser-scan': 'BrowserScanView',
        'password-crack': 'PasswordCrackView',
        druid: 'DruidView'
      }
      const name = this.activeTab
      return map[name] || null
    },

    tabsStyle() {
      const fontWeight = 400 + (this.fontWeightRatio / 100) * 500
      const shadowOpacity = (this.shadowRatio / 100) * 0.1
      return {
        '--tabs-font-color': this.fontColor,
        '--tabs-font-weight': fontWeight,
        '--tabs-shadow': `0px 2px 2px rgba(0, 0, 0, ${shadowOpacity})`
      }
    }
  },

  watch: {
    // 监听 visibleTabs 变化，确保 activeTab 始终有效
    visibleTabs: {
      handler(newTabs) {
        if (newTabs.length > 0 && this.configLoaded) {
          this.ensureActiveTabValid();
        }
      },
      immediate: false
    }
  },

  async mounted() {
    // 先加载标签页配置，再加载数据
    await this.loadTabsConfig();
    this.configLoaded = true;
    
    // 加载字体颜色配置
    this.loadFontColor();

    // 监听菜单更新事件
    window.addEventListener('menu-order-updated', this.handleMenuOrderUpdated);
    window.addEventListener('cyberSabre-switch-tab', this.handleSwitchTab);
    window.addEventListener('theme-background-changed', this.handleThemeChange);
    // 监听联动填充事件，用于在目标模块未挂载时暂存数据
    window.addEventListener('cyberSabre-dirscan-fill', this.handleDirscanFillProxy);
    window.addEventListener('cyberSabre-fingerprint-fill', this.handleFingerprintFillProxy);
  },

  beforeUnmount() {
    window.removeEventListener('menu-order-updated', this.handleMenuOrderUpdated);
    window.removeEventListener('cyberSabre-switch-tab', this.handleSwitchTab);
    window.removeEventListener('theme-background-changed', this.handleThemeChange);
    window.removeEventListener('cyberSabre-dirscan-fill', this.handleDirscanFillProxy);
    window.removeEventListener('cyberSabre-fingerprint-fill', this.handleFingerprintFillProxy);
  },

  methods: {
    // 处理菜单顺序更新事件
    async handleMenuOrderUpdated(event) {
      // console.log('收到菜单配置更新事件:', event.detail);

      // 重新加载标签页配置
      await this.loadTabsConfig();

      // 强制重新渲染 el-tabs
      this.forceTabsRerender();
    },

    // 处理子组件请求切换标签页事件
    handleSwitchTab(event) {
      const name = event?.detail
      if (!name) return
      const exists = this.visibleTabs.some(tab => tab.name === name)
      if (exists) {
        this.activeTab = name
      }
    },

    // 强制重新渲染 el-tabs 组件
    forceTabsRerender() {
      // 方法1: 使用 key 强制重新渲染
      this.tabsKey = Date.now();

      // 方法2: 临时改变 activeTab 再改回来
      const currentActiveTab = this.activeTab;
      this.activeTab = '';

      this.$nextTick(() => {
        this.activeTab = currentActiveTab;
      });
    },

    // 加载标签页配置
    async loadTabsConfig() {
      try {
        const savedData = await loadMenuOrder();
        // console.log('加载的标签页数据:', savedData);

        const savedTabsOrder = savedData.tabs || {};
        const cyberSabreTabs = savedTabsOrder.cyberSabre || [];

        // 获取默认配置
        const defaultTabs = moduleTabsConfig.cyberSabre || [];

        // 创建标签页映射
        const tabMap = {};
        defaultTabs.forEach(tab => {
          tabMap[tab.name] = {
            ...tab,
            order: tab.defaultOrder,
            visible: tab.visible
          };
        });

        // 应用保存的配置
        cyberSabreTabs.forEach(savedTab => {
          if (tabMap[savedTab.name]) {
            tabMap[savedTab.name].order = savedTab.order;
            tabMap[savedTab.name].visible = savedTab.visible;
          }
        });

        // 转换为数组并排序
        this.moduleTabs = Object.values(tabMap)
            .sort((a, b) => (a.order || 0) - (b.order || 0));

        console.log('处理后的标签页:', this.moduleTabs);

        // 设置默认激活的标签页
        this.setDefaultActiveTab();

      } catch (error) {
        console.error('加载标签页配置失败:', error);
        this.setDefaultTabs();
      }
    },

    // 确保当前激活的标签页有效
    ensureActiveTabValid() {
      const visibleTabs = this.visibleTabs;
      if (visibleTabs.length === 0) {
        this.activeTab = '';
        return;
      }

      // 检查当前激活的标签页是否在可见标签页中
      const currentTabExists = visibleTabs.some(tab => tab.name === this.activeTab);
      if (!currentTabExists) {
        // 如果当前激活标签不存在，设置为第一个可见标签
        this.activeTab = visibleTabs[0].name;
      }
    },

    setDefaultActiveTab() {
      const visibleTabs = this.visibleTabs;
      if (visibleTabs.length > 0) {
        // 如果当前没有激活标签或者当前激活标签不可见，则设置为第一个可见标签
        if (!this.activeTab || !visibleTabs.some(tab => tab.name === this.activeTab)) {
          this.activeTab = visibleTabs[0].name;
        }
      } else {
        // 如果没有可见标签页，设置一个默认值
        this.activeTab = '';
      }
    },

    setDefaultTabs() {
      // 直接使用默认配置
      this.moduleTabs = moduleTabsConfig.cyberSabre.map(tab => ({
        ...tab,
        order: tab.defaultOrder
      }));
      this.setDefaultActiveTab();
    },

    jsonp(url, cbName, onSuccess) {
      window[cbName] = (res) => {
        onSuccess(res);
        delete window[cbName];
        document.head.removeChild(script);
      };
      const script = document.createElement('script');
      script.src = url;
      document.head.appendChild(script);
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
    },

    /**
     * 主机扫描联动目录扫描时的事件代理：
     * - 如果目录扫描模块尚未挂载（DirsearchView 还没被打开过），
     *   则将联动数据暂存到全局队列，等待模块挂载后再处理；
     * - 如果模块已经挂载，则直接交给模块自身的监听处理，避免重复。
     */
    handleDirscanFillProxy(event) {
      try {
        // DirsearchView 挂载时会在 window 上打一个就绪标记
        if (window.__dirsearchReady) {
          return;
        }
        const payload = event && event.detail;
        if (!payload) return;

        if (!Array.isArray(window.__cyberSabrePendingDirscan)) {
          window.__cyberSabrePendingDirscan = [];
        }
        window.__cyberSabrePendingDirscan.push(payload);
      } catch (e) {
        console.error('处理目录扫描联动事件失败:', e);
      }
    },

    /**
     * 主机扫描联动指纹识别时的事件代理：
     * - FingerprintView 未挂载时暂存数据到全局队列；
     * - 已挂载则由 FingerprintView 自己的监听处理。
     */
    handleFingerprintFillProxy(event) {
      try {
        if (window.__fingerprintReady) {
          return;
        }
        const payload = event && event.detail;
        if (!payload) return;

        if (!Array.isArray(window.__cyberSabrePendingFingerprint)) {
          window.__cyberSabrePendingFingerprint = [];
        }
        window.__cyberSabrePendingFingerprint.push(payload);
      } catch (e) {
        console.error('处理指纹识别联动事件失败:', e);
      }
    }
  }
};
</script>

<style scoped>
.app-container {
  min-height: 96vh;
  display: flex;
  flex-direction: column;
  background-color: transparent;
  padding: 0 12px;
  box-sizing: border-box;
}

:deep(.el-main.cyber-sabre-main) {
  --el-main-padding: 8px 0 0;
}
</style>

<style src="@/styles/cyberSabre-layout.css"></style>