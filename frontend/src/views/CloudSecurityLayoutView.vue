<template>
  <el-container class="app-container" direction="vertical">
    <el-tabs v-model="activeTab" class="et-module-tabs" :key="tabsKey" :style="tabsStyle">
      <el-tab-pane
        v-for="tab in visibleTabs"
        :key="tab.name"
        :label="tab.title"
        :name="tab.name"
      />
    </el-tabs>

    <el-main class="csl-main">
      <div class="cloud-security-main-fill">
        <keep-alive include="OssBucketPanel,ObjectStoragePanel,CloudServerPanel">
          <component :is="currentComponent" />
        </keep-alive>
      </div>
    </el-main>
  </el-container>
</template>

<script>
import { defineAsyncComponent } from 'vue'
import { loadMenuOrder, moduleTabsConfig } from '@/utils/menuConfig';
import { tabNavigationBridge } from '@/utils/tabNavigationBridge'

const OssBucketPanel = defineAsyncComponent(() => import('@/views/cloudSecurity/OssBucketPanel.vue'));
const ObjectStoragePanel = defineAsyncComponent(() => import('@/views/cloudSecurity/ObjectStoragePanel.vue'));
const CloudServerPanel = defineAsyncComponent(() => import('@/views/cloudSecurity/CloudServerPanel.vue'));

export default {
  name: 'CloudSecurityLayoutView',
  mixins: [tabNavigationBridge('cloudSecurity')],
  components: { OssBucketPanel, ObjectStoragePanel, CloudServerPanel },
  data() {
    return {
      activeTab: '',
      moduleTabs: [],
      tabsKey: Date.now(),
      configLoaded: false,
      fontColor: '#ffffff',
      fontWeightRatio: 100,
      shadowRatio: 100,
      STORAGE_KEY: 'easytools:theme:background'
    };
  },
  computed: {
    visibleTabs() {
      if (!this.moduleTabs || this.moduleTabs.length === 0) {
        return [];
      }
      const visible = this.moduleTabs.filter((tab) => tab.visible !== false);
      return visible.sort((a, b) => (a.order || 0) - (b.order || 0));
    },
    currentComponent() {
      const map = {
        ossBucketPanel: 'OssBucketPanel',
        objectStoragePanel: 'ObjectStoragePanel',
        cloudServerPanel: 'CloudServerPanel'
      };
      return map[this.activeTab] || null;
    },
    tabsStyle() {
      const fontWeight = 400 + (this.fontWeightRatio / 100) * 500;
      const shadowOpacity = (this.shadowRatio / 100) * 0.1;
      return {
        '--tabs-font-color': this.fontColor,
        '--tabs-font-weight': fontWeight,
        '--tabs-shadow': `0px 2px 2px rgba(0, 0, 0, ${shadowOpacity})`
      };
    }
  },
  watch: {
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
    await this.loadTabsConfig();
    this.configLoaded = true;
    this.loadFontColor();
    window.addEventListener('menu-order-updated', this.handleMenuOrderUpdated);
    window.addEventListener('theme-background-changed', this.handleThemeChange);
  },
  beforeUnmount() {
    window.removeEventListener('menu-order-updated', this.handleMenuOrderUpdated);
    window.removeEventListener('theme-background-changed', this.handleThemeChange);
  },
  methods: {
    async handleMenuOrderUpdated() {
      await this.loadTabsConfig();
      this.tabsKey = Date.now();
      const current = this.activeTab;
      this.activeTab = '';
      this.$nextTick(() => {
        this.activeTab = current;
      });
    },
    async loadTabsConfig() {
      try {
        const savedData = await loadMenuOrder();
        const savedTabsOrder = savedData.tabs || {};
        const savedModule = savedTabsOrder.cloudSecurity || [];
        const defaultTabs = moduleTabsConfig.cloudSecurity || [];
        const tabMap = {};
        defaultTabs.forEach((tab) => {
          tabMap[tab.name] = {
            ...tab,
            order: tab.defaultOrder,
            visible: tab.visible
          };
        });
        savedModule.forEach((savedTab) => {
          if (tabMap[savedTab.name]) {
            tabMap[savedTab.name].order = savedTab.order;
            tabMap[savedTab.name].visible = savedTab.visible;
          }
        });
        this.moduleTabs = Object.values(tabMap).sort((a, b) => (a.order || 0) - (b.order || 0));
        this.setDefaultActiveTab();
      } catch (e) {
        console.error('加载云上安全标签页配置失败:', e);
        this.setDefaultTabs();
      }
    },
    ensureActiveTabValid() {
      const visibleTabs = this.visibleTabs;
      if (visibleTabs.length === 0) {
        this.activeTab = '';
        return;
      }
      const ok = visibleTabs.some((tab) => tab.name === this.activeTab);
      if (!ok) {
        this.activeTab = visibleTabs[0].name;
      }
    },
    setDefaultActiveTab() {
      const visibleTabs = this.visibleTabs;
      if (visibleTabs.length > 0) {
        if (!this.activeTab || !visibleTabs.some((tab) => tab.name === this.activeTab)) {
          this.activeTab = visibleTabs[0].name;
        }
      } else {
        this.activeTab = '';
      }
    },
    setDefaultTabs() {
      this.moduleTabs = (moduleTabsConfig.cloudSecurity || []).map((tab) => ({
        ...tab,
        order: tab.defaultOrder
      }));
      this.setDefaultActiveTab();
    },
    loadFontColor() {
      try {
        const saved = localStorage.getItem(this.STORAGE_KEY);
        if (saved) {
          const config = JSON.parse(saved);
          if (config.fontColor) this.fontColor = config.fontColor;
          if (config.fontWeightRatio !== undefined) this.fontWeightRatio = config.fontWeightRatio;
          if (config.shadowRatio !== undefined) this.shadowRatio = config.shadowRatio;
        }
      } catch (e) {
        console.error('加载主题标签样式失败:', e);
      }
    },
    handleThemeChange(event) {
      if (event.detail && event.detail !== null) {
        if (event.detail.fontColor) this.fontColor = event.detail.fontColor;
        if (event.detail.fontWeightRatio !== undefined) this.fontWeightRatio = event.detail.fontWeightRatio;
        if (event.detail.shadowRatio !== undefined) this.shadowRatio = event.detail.shadowRatio;
      } else if (event.detail === null) {
        this.fontColor = '#ffffff';
        this.fontWeightRatio = 100;
        this.shadowRatio = 100;
      } else {
        this.loadFontColor();
      }
    }
  }
};
</script>

<style scoped>
.csl-main {
  flex: 1;
  min-height: 0;
  padding: 0 4px 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.app-container {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background-color: transparent;
  padding: 0 10px;
  box-sizing: border-box;
}

.cloud-security-main-fill {
  flex: 1;
  min-height: 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.cloud-security-main-fill :deep(.oss-content) {
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow-y: auto;
}

:deep(.el-main) {
  --el-main-padding: 0px;
}
</style>
