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
    <el-main>
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

const SSH = defineAsyncComponent(() => import('@/views/connect/ssh.vue'));
const FTP = defineAsyncComponent(() => import('@/views/connect/ftp.vue'));
const NoSQL = defineAsyncComponent(() => import('@/views/connect/nosql.vue'));
const DBManager = defineAsyncComponent(() => import('@/views/connect/dbmanager.vue'));

export default {
  name: "ConnectView",
  mixins: [tabNavigationBridge('connect')],
  components: { NoSQL, FTP, SSH, DBManager },
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
        ssh: 'SSH',
        ftp: 'FTP',
        nosql: 'NoSQL',
        dbmanager: 'DBManager'
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
    // 监听 visibleTabs 变化，确保 activeTab 始终有效
    visibleTabs: {
      handler(newTabs) {
        if (newTabs.length > 0 && this.configLoaded) {
          this.ensureActiveTabValid();
        }
      },
      immediate: false
    },
  },

  async mounted() {
    // 加载标签页配置
    await this.loadTabsConfig();
    this.configLoaded = true;
    
    // 加载字体颜色配置
    this.loadFontColor();

    // 监听菜单更新事件
    window.addEventListener('menu-order-updated', this.handleMenuOrderUpdated);
    window.addEventListener('theme-background-changed', this.handleThemeChange);
    
    // 监听打开连接对话框事件（从漏洞扫描发送）
    window.addEventListener('open-connect-dialog', this.handleOpenConnectDialog);

    // 移除原有的弹窗提示，因为提示现在在顶部按钮上显示
  },

  beforeUnmount() {
    // 在组件卸载时移除事件监听器
    window.removeEventListener('menu-order-updated', this.handleMenuOrderUpdated);
    window.removeEventListener('theme-background-changed', this.handleThemeChange);
    window.removeEventListener('open-connect-dialog', this.handleOpenConnectDialog);
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
        const connectTabs = savedTabsOrder.connect || [];

        // 获取默认配置
        const defaultTabs = moduleTabsConfig.connect || [];

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
        connectTabs.forEach(savedTab => {
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
      this.moduleTabs = moduleTabsConfig.connect.map(tab => ({
        ...tab,
        order: tab.defaultOrder
      }));
      this.setDefaultActiveTab();
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
    
    // 处理打开连接对话框事件
    handleOpenConnectDialog(event) {
      const { module, connectionInfo } = event.detail || {};
      if (!module || !connectionInfo) {
        console.error('打开连接对话框事件缺少必要参数');
        return;
      }
      
      // 映射模块名称到标签页名称
      const moduleMap = {
        ssh: 'ssh',
        nosql: 'nosql',
        redis: 'nosql', // 向后兼容
        ftp: 'ftp',
        mysql: 'dbmanager',
        postgresql: 'dbmanager',
        mongodb: 'dbmanager',
        mssql: 'dbmanager',
        oracle: 'dbmanager'
      };
      
      const targetTab = moduleMap[module];
      if (!targetTab) {
        console.error('无法识别的模块类型:', module);
        return;
      }
      
      // 切换到对应的标签页
      this.activeTab = targetTab;
      
      // 等待标签页切换后，通过事件通知子组件打开对话框并填充数据
      this.$nextTick(() => {
        // 延迟一小段时间确保组件已经渲染完成
        setTimeout(() => {
          // 触发子组件级别的事件
          window.dispatchEvent(new CustomEvent('fill-connect-form', {
            detail: {
              module: module,
              connectionInfo: connectionInfo
            }
          }));
        }, 200);
      });
    }
  },
};
</script>

<style scoped>
.el-main {
  padding: 0;
  overflow: hidden;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* 页面容器 - 移除硬编码背景色，让主题背景透过来 */
.app-container {
  height: 100%;
  max-height: 100%;
  display: flex;
  flex-direction: column;
  background-color: transparent;
  padding: 0 10px;
  box-sizing: border-box;
  overflow: hidden;
}



</style>