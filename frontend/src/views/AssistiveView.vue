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

    <!-- 内容区域 -->
    <el-main
      :class="[
        'content',
        { 'shell-syntax-active': activeTab === 'shell-syntax', 'assistive-proxy-main': activeTab === 'portable-proxy' },
      ]"
    >
      <keep-alive>
        <component :is="currentComponent" />
      </keep-alive>
    </el-main>
  </el-container>
</template>

<script>
import { defineAsyncComponent } from 'vue'
import { loadMenuOrder, moduleTabsConfig } from '@/utils/menuConfig';
import { tabNavigationBridge } from '@/utils/tabNavigationBridge'

const MapCheck = defineAsyncComponent(() => import('@/views/assistive/mapCheck.vue'));
const Password = defineAsyncComponent(() => import('@/views/assistive/password.vue'));
const Process = defineAsyncComponent(() => import('@/views/assistive/process.vue'));
const Google = defineAsyncComponent(() => import('@/views/assistive/google.vue'));
const Shell = defineAsyncComponent(() => import('@/views/assistive/shell.vue'));
const Fscan = defineAsyncComponent(() => import('@/views/assistive/fscan.vue'));
const IpBanDeal = defineAsyncComponent(() => import('@/views/assistive/ipBanDeal.vue'));
const RandomInfo = defineAsyncComponent(() => import('@/views/assistive/RandomInfo.vue'));
const PortableProxyEmbed = defineAsyncComponent(() => import('@/views/assistive/PortableProxyEmbed.vue'));

export default {
  name: "AssistiveView",
  mixins: [tabNavigationBridge('assistive')],
  components: {
    PortableProxyEmbed,
    RandomInfo,
    RandomInfoView: RandomInfo,
    IpBanDeal,
    Fscan,
    MapCheck,
    Shell,
    Google,
    Process,
    Password,
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
    };
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

    currentComponent() {
      const map = {
        'password-query': 'Password',
        'process-query': 'Process',
        'google-syntax': 'Google',
        'shell-syntax': 'Shell',
        'MapCheck-query': 'MapCheck',
        'fscan-deal': 'Fscan',
        'ip-ban-deal': 'IpBanDeal',
        'randomInfo': 'RandomInfo',
        'portable-proxy': 'PortableProxyEmbed',
      }
      return map[this.activeTab] || null
    },
    
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

  watch: {
    '$route'(to, from) {
      if (
        to.name === 'assistive' &&
        (!from || to.fullPath !== from.fullPath)
      ) {
        this.$nextTick(() => this.applyAssistiveDeepLink(to));
      }
    },
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

    this.$nextTick(() => this.applyAssistiveDeepLink(this.$route));

    // 监听菜单更新事件
    window.addEventListener('menu-order-updated', this.handleMenuOrderUpdated);
    window.addEventListener('theme-background-changed', this.handleThemeChange);
  },

  beforeUnmount() {
    window.removeEventListener('menu-order-updated', this.handleMenuOrderUpdated);
    window.removeEventListener('theme-background-changed', this.handleThemeChange);
  },

  methods: {
    /** /assistive?tab=portable-proxy（或兼容旧 proxy）直达子标签 */
    applyAssistiveDeepLink(route = this.$route) {
      try {
        const raw = route.query && route.query.tab;
        if (raw == null || raw === '') return;
        let target = String(raw);
        if (target === 'proxy') target = 'portable-proxy';
        const ok = this.visibleTabs.some((t) => t.name === target);
        if (!ok) return;
        this.activeTab = target;
        const q = { ...route.query };
        delete q.tab;
        const clean = {};
        Object.keys(q).forEach((k) => {
          if (q[k] !== undefined) clean[k] = q[k];
        });
        this.$nextTick(() => {
          this.$router.replace({ path: route.path, query: clean }).catch(() => {});
        });
      } catch (_) {
        /* ignore */
      }
    },

    // 处理菜单顺序更新事件
    async handleMenuOrderUpdated(event) {
      // console.log('收到菜单配置更新事件:', event.detail);

      // 重新加载标签页配置
      await this.loadTabsConfig();

      // 强制重新渲染 el-tabs
      this.forceTabsRerender();
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
        const assistiveTabs = savedTabsOrder.assistive || [];

        // 获取默认配置
        const defaultTabs = moduleTabsConfig.assistive || [];

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
        assistiveTabs.forEach(savedTab => {
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
      this.moduleTabs = moduleTabsConfig.assistive.map(tab => ({
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
    }
  }
};
</script>

<style scoped>
/* 页面容器 - 移除硬编码背景色，让主题背景透过来 */
.app-container {
  height: 95vh;
  display: flex;
  padding: 5px;
  flex-direction: column;
  background-color: transparent;
}

.content {
  flex-grow: 1;
  display: flex;
  flex-direction: column;
  padding: 20px;
  background-color: #ffffff;
  box-shadow: 0px 0px 10px rgba(0, 0, 0, 0.05);
  border-radius: 8px;
  box-sizing: border-box;
}

.content.assistive-proxy-main {
  padding: 10px 12px !important;
  overflow: hidden !important;
  background-color: rgba(250, 249, 246, 0.95);
}

/* 主内容区域 */
.el-main {
  padding: 0 8px;
  flex: 1;
  min-height: 0;
  overflow: auto;
}

/* 标签页内容通用样式 */
.tab-content {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
  padding: 5px 8px;
}

/* 通用卡片样式 */
.tab-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 200px;
}

.tab-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.tab-card-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  min-height: 0;
}

/* 通用滚动条样式 */
.tab-card-content::-webkit-scrollbar {
  width: 6px;
}

.tab-card-content::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.tab-card-content::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.tab-card-content::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

/* 文本样式 */
:deep(h4){
  display: block;
  margin-block-start: 1.33em;
  margin-block-end: 1.33em;
  margin-inline-start: 0px;
  margin-inline-end: 0px;
  font-weight: bold;
  unicode-bidi: isolate;
}

:deep(h5){
  display: block;
  font-size: 0.83em;
  margin-block-start: 1.67em;
  margin-block-end: 1.67em;
  margin-inline-start: 0;
  margin-inline-end: 0;
  font-weight: bold;
  unicode-bidi: isolate;
}

:deep(p){
  display: block;
  margin-block-start: 1em;
  margin-block-end: 1em;
  margin-inline-start: 0;
  margin-inline-end: 0;
  unicode-bidi: isolate;
}
</style>