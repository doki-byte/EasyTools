<template>
  <div class="redis-container">
    <!-- 左侧：连接管理区域 -->
    <div class="redis-sidebar">
      <div class="sidebar-header">
        <ConnectionManage ref="connectionManageRef" title="新建连接" btn-type="primary" @emit-connection-list="flushConnectionList"/>
      </div>
      <div class="sidebar-content">
        <ConnectionList @emit-select-db="selectRedisDB" @emit-edit-connection="handleEditConnection" :flush="flushFlag"/>
      </div>
    </div>

    <!-- 中间和右侧：键列表和键值显示 -->
    <template v-if="tabs.length > 0">
      <el-tabs 
        v-model="activeTabId" 
        type="card" 
        closable
        @tab-remove="handleTabRemove"
        @tab-click="handleTabClick"
        @contextmenu.native.prevent="handleTabsContextMenu"
        class="redis-tabs"
      >
        <el-tab-pane
          v-for="tab in tabs"
          :key="tab.id"
          :label="tab.label"
          :name="tab.id"
        >
          <template #label>
            <span 
              class="tab-label"
            >
              <el-icon class="tab-icon"><Connection /></el-icon>
              {{ tab.label }}
            </span>
          </template>
          <div class="tab-content">
            <el-tabs v-model="tab.activeTab" class="redis-sub-tabs">
              <el-tab-pane label="键值" name="keys">
                <div class="redis-keys-value-container">
                  <div class="redis-keys-panel">
                    <Keys
                        :keyDB="tab.keyDB"
                        :keyConnIdentity="tab.keyConnIdentity"
                        @emit-select-key="(key) => selectKey(key, tab.id)"
                    />
                  </div>
                  <div class="redis-value-panel">
                    <KeyValue
                        :keyDB="tab.keyDB"
                        :keyConnIdentity="tab.keyConnIdentity"
                        :keyKey="tab.keyKey"
                    />
                  </div>
                </div>
              </el-tab-pane>
              <el-tab-pane label="命令执行" name="command">
                <Command 
                  :connIdentity="tab.keyConnIdentity" 
                  :dbType="tab.dbType"
                  mode="command" 
                />
              </el-tab-pane>
              <el-tab-pane label="详情" name="detail">
                <Command 
                  :connIdentity="tab.keyConnIdentity" 
                  :dbType="tab.dbType"
                  mode="detail" 
                />
              </el-tab-pane>
            </el-tabs>
          </div>
        </el-tab-pane>
      </el-tabs>
    </template>
    <div v-else class="redis-empty-state">
      <div class="empty-content">
        <el-icon class="empty-icon"><Connection /></el-icon>
        <p class="empty-text">请先选择一个数据库连接</p>
      </div>
    </div>
    <!-- 右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
    <Teleport to="body">
      <ul class="tabs-context-menu" v-if="tabsContextMenuVisible" :style="tabsContextMenuStyle">
        <li @click="handleCloseOtherTabs">关闭其他</li>
        <li @click="handleCloseAllTabs">全部关闭</li>
      </ul>
    </Teleport>
  </div>
</template>

<script>
import { defineAsyncComponent } from 'vue'
const Keys = defineAsyncComponent(() => import('@/views/nosql/Keys.vue'));
const ConnectionManage = defineAsyncComponent(() => import('@/views/nosql/ConnectionManage.vue'));
const KeyValue = defineAsyncComponent(() => import('@/views/nosql/KeyValue.vue'));
const ConnectionList = defineAsyncComponent(() => import('@/views/nosql/ConnectionList.vue'));
const Command = defineAsyncComponent(() => import('@/views/nosql/Command.vue'));
import { Connection } from "@element-plus/icons-vue";
import { ConnectionList as GetConnectionList } from "../../../wailsjs/go/nosql/NoSQL";

export default {
  name: "NoSQL",
  components: { ConnectionList, KeyValue, ConnectionManage, Keys, Connection, Command },
  data() {
    return {
      flushFlag: true,
      tabs: [],
      activeTabId: null,
      tabIdCounter: 0,
      connectionList: [], // 存储连接列表
      // 标签页右键菜单
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: '',
    }
  },
  computed: {
    // 只要有标签页，就认为有连接
    isConnected() {
      return this.tabs.length > 0;
    },
    tabsContextMenuStyle() {
      // 直接从事件获取坐标，使用 clientX/clientY（相对于视口，适用于 position: fixed）
      const { x, y } = this.tabsContextMenuPosition
      
      // 直接设置坐标，确保菜单显示在鼠标点击位置
      // 使用 position: fixed 和视口坐标，避免任何偏移
      return {
        top: `${y}px`,
        left: `${x}px`,
        position: 'fixed',
        zIndex: 3000
      }
    },
  },
  async mounted() {
    await this.loadConnectionList();
    // 监听填充连接表单事件
    window.addEventListener('fill-connect-form', this.handleFillConnectForm);
  },
  beforeUnmount() {
    window.removeEventListener('fill-connect-form', this.handleFillConnectForm);
  },
  methods: {
    // 正常业务开始
    flushConnectionList() {
      this.flushFlag = !this.flushFlag;
      // 重新加载连接列表
      this.loadConnectionList();
    },
    async loadConnectionList() {
      try {
        const res = await GetConnectionList();
        if (res.code === 200 && res.data) {
          this.connectionList = res.data;
        }
      } catch (error) {
        console.error('加载连接列表失败:', error);
        this.connectionList = [];
      }
    },
    getTab(tabId) {
      return this.tabs.find(t => t.id === tabId);
    },
    selectRedisDB(db, connIdentity) {
      // 检查是否已存在该连接的标签页
      const existingTab = this.tabs.find(tab => tab.keyConnIdentity === connIdentity && tab.keyDB === db);
      if (existingTab) {
        // 切换到已存在的标签页
        this.activeTabId = existingTab.id;
        return;
      }
      
      // 创建新标签页
      const tabId = `tab_${++this.tabIdCounter}`;
      const connection = this.getConnectionByIdentity(connIdentity);
      const hostAddr = connection ? connection.addr : '未知主机';
      
      // 判断数据库类型，格式化标签页名称
      const dbType = connection?.db_type || connection?.DbType || 'redis';
      const dbTypeLower = dbType.toLowerCase();
      let tabLabel;
      if (dbTypeLower === 'mongodb' || dbTypeLower === 'mongo') {
        // MongoDB: 数据库名是字符串，直接显示
        tabLabel = `${hostAddr} - ${db}`;
      } else {
        // Redis: 数据库编号，格式：主机地址 - DB0
        tabLabel = `${hostAddr} - DB${db}`;
      }
      
      const newTab = {
        id: tabId,
        keyDB: db, // 可能是数字（Redis）或字符串（MongoDB）
        keyConnIdentity: connIdentity,
        keyKey: null,
        label: tabLabel,
        activeTab: 'keys', // 默认显示键值标签页
        dbType: dbType // 存储数据库类型
      };
      
      this.tabs.push(newTab);
      this.activeTabId = tabId;
    },
    getConnectionByIdentity(identity) {
      // 从连接列表中查找连接信息
      // 支持 identity 和 Identity 两种字段名（兼容不同格式）
      return this.connectionList.find(conn => (conn.identity === identity || conn.Identity === identity)) || null;
    },
    getConnectionDbType(identity) {
      // 获取连接的数据库类型
      const connection = this.getConnectionByIdentity(identity);
      if (!connection) {
        return 'redis'; // 默认返回 redis
      }
      // 支持 db_type 和 DbType 两种字段名（兼容不同格式）
      return connection.db_type || connection.DbType || 'redis';
    },
    selectKey(key, tabId) {
      const tab = this.getTab(tabId);
      if (tab) {
        tab.keyKey = key;
      }
    },
    handleTabRemove(tabId) {
      const index = this.tabs.findIndex(t => t.id === tabId);
      if (index > -1) {
        this.tabs.splice(index, 1);
        if (this.tabs.length > 0) {
          this.activeTabId = this.tabs[Math.min(index, this.tabs.length - 1)].id;
        } else {
          this.activeTabId = null;
        }
      }
    },
    handleTabClick(tab) {
      // tab可能是字符串（name）或对象（包含paneName或name属性）
      if (typeof tab === 'string') {
        this.activeTabId = tab;
      } else if (tab && (tab.paneName || tab.name)) {
        this.activeTabId = tab.paneName || tab.name;
      }
    },
    // 标签页右键菜单处理
    handleTabsContextMenu(event, tabId = null) {
      event.preventDefault()
      event.stopPropagation()
      
      // 检查是否点击在标签页上
      const tabItem = event.target.closest('.el-tabs__item')
      if (!tabItem) {
        this.hideTabsContextMenu()
        return
      }
      
      // 如果传入了tabId，直接使用；否则尝试从DOM获取
      if (tabId) {
        this.tabsContextMenuTabId = tabId
      } else {
        // 尝试从DOM元素获取标签页ID
        const tabIdAttr = tabItem.id || tabItem.getAttribute('aria-controls') || tabItem.getAttribute('id')
        if (tabIdAttr) {
          // 提取tab ID，可能是 "tab-tab_1" 或 "pane-tab_1" 格式
          this.tabsContextMenuTabId = tabIdAttr.replace(/^tab-/, '').replace(/^pane-/, '')
        } else {
          // 如果无法从DOM获取，尝试从标签页文本匹配
          const labelText = tabItem.textContent?.trim()
          const matchedTab = this.tabs.find(t => t.label === labelText)
          this.tabsContextMenuTabId = matchedTab ? matchedTab.id : this.activeTabId
        }
      }

      // 直接从事件获取坐标，使用 clientX/clientY（相对于视口，适用于 position: fixed）
      // 确保获取的是相对于浏览器视口的坐标，避免滚动或定位导致的偏移
      const x = event.clientX !== undefined ? event.clientX : (event.pageX || 0)
      const y = event.clientY !== undefined ? event.clientY : (event.pageY || 0)
      
      // 直接设置坐标，确保菜单显示在鼠标点击位置
      this.tabsContextMenuPosition = { x, y }
      this.tabsContextMenuVisible = true
      
      // 点击其他地方时隐藏菜单
      const hideMenu = (e) => {
        if (!e.target.closest('.tabs-context-menu')) {
          this.hideTabsContextMenu()
          document.removeEventListener('click', hideMenu)
          document.removeEventListener('contextmenu', hideMenu)
        }
      }
      setTimeout(() => {
        document.addEventListener('click', hideMenu)
        document.addEventListener('contextmenu', hideMenu)
      }, 100)
    },
    hideTabsContextMenu() {
      this.tabsContextMenuVisible = false
    },
    // 关闭其他标签页（右键菜单）
    handleCloseOtherTabs() {
      this.hideTabsContextMenu()
      const targetId = this.tabsContextMenuTabId || this.activeTabId
      if (!targetId) return
      
      // 保留目标标签页，关闭其他标签页
      const targetTab = this.tabs.find(t => t.id === targetId)
      if (!targetTab) return
      
      // 关闭其他标签页
      this.tabs = this.tabs.filter(tab => tab.id === targetId)
      this.activeTabId = targetId
    },
    // 关闭所有标签页（右键菜单）
    handleCloseAllTabs() {
      this.hideTabsContextMenu()
      this.tabs = []
      this.activeTabId = null
    },
    // 处理填充连接表单事件
    handleFillConnectForm(event) {
      const { module, connectionInfo } = event.detail || {};
      // 处理NoSQL模块的请求（包括redis，因为redis会被映射到nosql标签页）
      if ((module !== 'nosql' && module !== 'redis') || !connectionInfo) {
        return;
      }
      
      // 通过ref访问ConnectionManage组件
      const connectionManage = this.$refs.connectionManageRef;
      if (!connectionManage) {
        console.error('ConnectionManage组件未找到');
        return;
      }
      
      // 根据模块类型设置db_type（如果未设置）
      const connectionInfoWithType = { ...connectionInfo };
      if (!connectionInfoWithType.db_type) {
        if (module === 'redis') {
          connectionInfoWithType.db_type = 'redis';
        } else if (module === 'mongodb') {
          connectionInfoWithType.db_type = 'mongodb';
        }
      }
      
      // 先填充表单数据（在打开对话框之前）
      if (connectionManage.fillForm) {
        connectionManage.fillForm(connectionInfoWithType);
      } else if (connectionManage.form) {
        // 如果fillForm方法不存在，直接操作form对象
        const form = connectionManage.form;
        if (connectionInfoWithType.host) form.addr = connectionInfoWithType.host;
        if (connectionInfoWithType.port) form.port = connectionInfoWithType.port;
        if (connectionInfoWithType.username !== undefined) form.username = connectionInfoWithType.username;
        if (connectionInfoWithType.password !== undefined) form.password = connectionInfoWithType.password;
        if (connectionInfoWithType.name) form.name = connectionInfoWithType.name;
        if (connectionInfoWithType.db_type) form.db_type = connectionInfoWithType.db_type;
      }
      
      // 打开对话框
      connectionManage.dialogVisible = true;
      
      // 等待对话框完全打开后，再次确认数据（因为watch可能会重置一些值）
      this.$nextTick(() => {
        setTimeout(() => {
          if (connectionManage.fillForm) {
            connectionManage.fillForm(connectionInfoWithType);
          } else if (connectionManage.form) {
            const form = connectionManage.form;
            if (connectionInfoWithType.host) form.addr = connectionInfoWithType.host;
            if (connectionInfoWithType.port) form.port = connectionInfoWithType.port;
            if (connectionInfoWithType.username !== undefined) form.username = connectionInfoWithType.username;
            if (connectionInfoWithType.password !== undefined) form.password = connectionInfoWithType.password;
            if (connectionInfoWithType.name) form.name = connectionInfoWithType.name;
            if (connectionInfoWithType.db_type) form.db_type = connectionInfoWithType.db_type;
          }
        }, 100);
      });
    },
    // 处理编辑连接事件
    handleEditConnection(item) {
      const connectionManage = this.$refs.connectionManageRef;
      if (!connectionManage) {
        console.error('ConnectionManage组件未找到');
        return;
      }
      
      // 填充表单数据（需要将item的字段映射到form字段）
      const connectionInfo = {
        identity: item.identity || item.Identity, // 支持两种字段名
        name: item.name,
        host: item.addr, // addr映射到host
        addr: item.addr, // 同时传递addr字段
        port: item.port,
        username: item.username,
        password: item.password,
        db_type: item.db_type,
        type: item.type,
        ssh_addr: item.ssh_addr,
        ssh_port: item.ssh_port,
        ssh_username: item.ssh_username,
        ssh_password: item.ssh_password
      };
      
      // 先填充表单数据
      if (connectionManage.fillForm) {
        connectionManage.fillForm(connectionInfo);
      }
      
      // 打开对话框
      connectionManage.dialogVisible = true;
    }
  }
}
</script>

<style scoped>
.redis-container {
  height: 100%;
  display: flex;
  gap: 8px;
  padding: 8px;
  box-sizing: border-box;
  overflow: hidden;
  background: transparent;
}

/* 左侧连接管理区域 */
.redis-sidebar {
  flex: 0 0 280px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid rgba(0, 0, 0, 0.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.sidebar-header {
  padding: 12px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
  background: rgba(255, 255, 255, 1);
  flex-shrink: 0;
}

.sidebar-content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.redis-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  padding: 12px;
}

.redis-tabs :deep(.el-tabs__content) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.redis-tabs :deep(.el-tab-pane) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 4px;
}

.tab-icon {
  font-size: 14px;
}

.tab-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.redis-sub-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.redis-sub-tabs :deep(.el-tabs__content) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.redis-sub-tabs :deep(.el-tab-pane) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.redis-keys-value-container {
  flex: 1;
  display: flex;
  gap: 8px;
  overflow: hidden;
  padding: 0;
}

.redis-keys-panel {
  flex: 0 0 280px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid rgba(0, 0, 0, 0.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* 键值显示面板 */
.redis-value-panel {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid rgba(0, 0, 0, 0.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* 空状态 */
.redis-empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.empty-content {
  text-align: center;
  color: rgba(0, 0, 0, 0.6);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  color: #409eff;
}

.empty-text {
  font-size: 14px;
  margin: 0;
  color: rgba(0, 0, 0, 0.6);
}

/* 标签页右键菜单样式 */
.tabs-context-menu {
  position: fixed !important;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  padding: 2px 0;
  margin: 0;
  list-style: none;
  z-index: 3000;
  min-width: 85px;
  /* 确保没有其他样式影响位置 */
  transform: none;
}

.tabs-context-menu li {
  padding: 6px 12px;
  cursor: pointer;
  font-size: 14px;
  color: #606266;
  transition: background-color 0.2s;
}

.tabs-context-menu li:hover {
  background-color: #f5f7fa;
  color: #409eff;
}

/* 确保 ConnectionList 组件填满其容器 */
:deep(.connection-main) {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}
</style>