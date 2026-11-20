<template>
  <div class="app">
    <CustomTitleBar/>
    <!-- 主要内容容器 -->
    <div class="main-container" :class="{ 'with-menu': routeName !== 'login' }">
      <!-- 菜单栏 -->
      <div class="appMenu" :class="{ collapsed: sidebarCollapsed }" v-if="routeName !== 'login'">
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
    <el-dialog
        v-model="exitModalVisible"
        width="320px"
        :show-close="false"
        :close-on-click-modal="false"
        :close-on-press-escape="true"
        class="glass-exit-dialog"
        align-center
        append-to-body
        modal-class="glass-exit-overlay"
    >
      <div class="glass-dialog">
        <div class="glass-header">
          <div class="app-icon">
            <div><img style="width: 50px; height: 50px; " src="/assets/system/appicon.png" alt=""/></div>
          </div>
          <h3 class="app-name">EasyTools😘</h3>
        </div>

        <div class="glass-content">
          <p class="question">确定要退出吗？🙃( ´･ω･)ﾉ(._.`)摸摸头</p>
        </div>

        <div class="glass-actions">
          <el-button class="glass-cancel" @click="cancelExit">再想想</el-button>
          <el-button type="primary" class="glass-confirm" @click="confirmExit">退出</el-button>
        </div>
      </div>
    </el-dialog>

    <GlobalSearchPalette v-model="globalSearchVisible" />

    <!-- 全局休息提示对话框 -->
    <el-dialog
        v-model="showRestDialog"
        title="休息提醒"
        width="500px"
        :close-on-click-modal="false"
        :close-on-press-escape="false"
    >
      <div class="rest-dialog-content">
        <div class="rest-icon">⏰</div>
        <h3>您已经工作很长时间了</h3>
        <p>玩个游戏放松一下吧！</p>
        <p class="rest-tip">适当的休息可以提高工作效率哦~</p>
      </div>
      <template #footer>
        <el-button @click="handleRestDialogClose">稍后再说</el-button>
        <el-button type="primary" @click="handleGoToGames">去玩游戏</el-button>
      </template>
    </el-dialog>

    <!-- 虚机警告弹框 -->
    <el-dialog
      v-model="vmWarningVisible"
      width="520px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      class="glass-vm-warning-dialog"
      :show-close="false"
    >
      <div class="glass-dialog-content">
        <div class="dialog-header vm-warning-header">
          <div class="header-decoration"></div>
          <div class="vm-warning-icon-wrapper">
            <div class="vm-warning-icon">
              <el-icon size="28"><Warning /></el-icon>
            </div>
            <div class="icon-pulse"></div>
          </div>
          <div class="header-text">
            <h3 class="dialog-title">虚拟机环境检测</h3>
            <p class="dialog-subtitle">Environment Detection</p>
          </div>
        </div>

        <div class="vm-warning-content">
          <div class="vm-warning-message">
            <div class="message-icon-wrapper">
              <el-icon class="warning-icon"><WarningFilled /></el-icon>
            </div>
            <div class="message-text">
              <p class="message-title">检测提示</p>
              <p class="message-content">{{ vmWarningMessage }}</p>
            </div>
          </div>
          
          <div class="vm-warning-tips">
            <div class="tip-item">
              <el-icon class="tip-icon"><InfoFilled /></el-icon>
              <span>建议在真实环境中使用以获得最佳体验</span>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="glass-dialog-footer vm-warning-footer">
          <el-button
              class="footer-btn confirm-btn"
              type="primary"
              @click="confirmVMWarning"
          >
            <el-icon style="margin-right: 6px;"><Check /></el-icon>
            我知道了
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import Menu from './components/Menu.vue'
import CustomTitleBar from "./components/CustomTitleBar.vue";
import GlobalSearchPalette from './components/GlobalSearchPalette.vue';
import { globalHotkeyManager } from '@/utils/globalHotkey';
import { ToggleShowHide, SetHotkey } from "../wailsjs/go/hotkey/HotKey";
import { EventsOn, EventsOff } from "../wailsjs/runtime";
import { GetOs, ExitApp, BackupConfigFiles } from "../wailsjs/go/system/System";
import { clearCloudServerTerminalStorageOnExit } from "@/utils/cloudServerTerminalStorage";
import {GetPortConfig} from "../wailsjs/go/controller/Server";
import { ElMessage } from 'element-plus';
import { Warning, WarningFilled, InfoFilled, Check } from '@element-plus/icons-vue';
import { applyThemeToApp, getThemeConfig } from '@/utils/themeBackground';

export default {
  name: 'App',
  components: {
    Check,
    Warning,
    InfoFilled,
    WarningFilled,
    CustomTitleBar,
    Menu,
    GlobalSearchPalette
  },
  provide() {
    return {
      openGlobalSearch: () => {
        this.globalSearchVisible = true
      }
    }
  },
  data() {
    return {
      // 缓存页面列表 - 这些名称必须与页面组件的name属性匹配
      cachedPages: [
        'ToolsView',
        'SiteView',
        "CyberSabreView",
        'AssistiveView',
        'InfoDealView',
        // 须与组件的 name 完全一致，否则离开路由后实例销毁，子标签会复位
        'CloudSecurityLayoutView',
        'InfoCollectionView',
        'ConnectView',
        'CtfToolsView',
        'RandomInfo',
        'BypassView',
        'NoteViews',
        'RestMateView',
        'CLIExecutorView'
      ],
      hotkeyReady: false,
      hotkeyReadyTimeout: null,
      OS: '',
      exitModalVisible: false, // 控制退出确认弹框显示
      portConfig: null,
      vmWarningVisible: false, // 控制虚机警告弹框显示
      vmWarningMessage: '', // 虚机警告消息
      // 全局休息提示相关
      showRestDialog: false,
      // 小概率触发相关
      smallProbabilityTriggered: false,
      smallProbabilityTimer: null,
      // 侧边栏折叠状态
      sidebarCollapsed: false,
      globalSearchVisible: false
    }
  },
  async created() {
      this.OS = '';
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
    await this.loadPortConfig();
    
    // 监听虚机警告事件
    try {
      EventsOn('vm-warning', this.handleVMWarning);
    } catch (error) {
      console.warn('EventsOn not available for vm-warning');
    }

    // 主题已在 main.js 中应用，这里不再重复应用
    // 只在主题配置变化时通过事件监听重新应用
    
    // 监听主题变化事件（只在主题配置变化时重新应用，不在路由切换时应用）
    window.addEventListener('theme-background-changed', this.handleThemeChanged);
    
    // 加载侧边栏状态
    this.loadSidebarState();
    
    // 监听侧边栏切换事件
    window.addEventListener('sidebar-toggle', this.handleSidebarToggle);
    
    // ========== 小概率弹出算法 ==========
    this.initSmallProbabilityCheck();
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
        EventsOff('vm-warning', this.handleVMWarning);
      } catch (error) {
        console.warn('EventsOff not available');
      }
    }

    // 移除窗口关闭事件监听
    this.removeWindowCloseListener();
    
    // 移除虚机警告事件监听
    try {
      EventsOff('vm-warning', this.handleVMWarning);
    } catch (error) {
      console.warn('EventsOff not available for vm-warning');
    }
    
    // 移除主题变化监听
    window.removeEventListener('theme-background-changed', this.handleThemeChanged);
    
    // 移除侧边栏切换监听
    window.removeEventListener('sidebar-toggle', this.handleSidebarToggle);
    
    // 清理小概率定时器
    this.clearSmallProbabilityTimer();
  },
  methods: {
    async loadPortConfig() {
      try {
        this.portConfig = await GetPortConfig();
        console.log('端口配置:', this.portConfig);
      } catch (error) {
        console.error('获取端口配置失败:', error);
        // 使用默认端口作为后备
        this.portConfig = {
          mainPort: 52867,
        };
      }
    },

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
    
    // 处理主题变化（只在主题配置变化时调用，不在路由切换时调用）
    handleThemeChanged(event) {
      // 主题配置变化时强制重新应用主题
      this.$nextTick(async () => {
        await applyThemeToApp(true); // 传入 true 强制重新应用
      });
    },

    // 加载侧边栏状态
    loadSidebarState() {
      try {
        const saved = localStorage.getItem('easytools:sidebar:collapsed');
        if (saved !== null) {
          this.sidebarCollapsed = JSON.parse(saved);
        }
      } catch (e) {
        console.error('加载侧边栏状态失败:', e);
      }
    },

    // 处理侧边栏切换事件
    handleSidebarToggle(event) {
      if (event.detail && event.detail.collapsed !== undefined) {
        this.sidebarCollapsed = event.detail.collapsed;
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

    // 判断是否需要备份（根据备份周期）
    shouldBackup(backupConfig) {
      if (!backupConfig.autoBackup) {
        return false;
      }

      const cycle = backupConfig.backupCycle || 'every_exit';
      
      // 每次退出都备份
      if (cycle === 'every_exit') {
        return true;
      }

      // 需要检查上次备份时间
      const lastBackupTime = backupConfig.lastBackupTime;
      if (!lastBackupTime) {
        // 没有备份记录，需要备份
        return true;
      }

      const lastBackup = new Date(lastBackupTime);
      const now = new Date();

      switch (cycle) {
        case 'daily': {
          // 每天一次：比较日期（年月日）
          const lastDate = new Date(lastBackup.getFullYear(), lastBackup.getMonth(), lastBackup.getDate());
          const nowDate = new Date(now.getFullYear(), now.getMonth(), now.getDate());
          return lastDate.getTime() < nowDate.getTime();
        }
        case 'weekly': {
          // 每周一次：比较是否超过7天
          const diffDays = Math.floor((now - lastBackup) / (1000 * 60 * 60 * 24));
          return diffDays >= 7;
        }
        case 'monthly': {
          // 每月一次：比较月份
          return lastBackup.getMonth() !== now.getMonth() || lastBackup.getFullYear() !== now.getFullYear();
        }
        default:
          return true;
      }
    },

    // 确认退出
    async confirmExit() {
      this.exitModalVisible = false;
      
      // 检查是否需要自动备份
      try {
        const backupConfigStr = localStorage.getItem('easytools:backup:config');
        if (backupConfigStr) {
          const backupConfig = JSON.parse(backupConfigStr);
          
          // 判断是否需要备份
          if (this.shouldBackup(backupConfig) && backupConfig.backupPath && backupConfig.selectedItems && backupConfig.selectedItems.length > 0) {
            const overwrite = backupConfig.backupMode === 'overwrite';

            // 覆盖模式下退出前进行确认，只有“创建副本”模式才直接备份
            if (overwrite) {
              try {
                await this.$confirm(
                  '当前自动备份为【覆盖模式】，将直接覆盖备份目录中的同名文件和文件夹，可能导致历史备份无法恢复。\n\n是否仍要执行本次自动备份？',
                  '自动备份确认',
                  {
                    confirmButtonText: '仍要覆盖备份',
                    cancelButtonText: '跳过本次备份',
                    type: 'warning'
                  }
                );
              } catch {
                // 用户选择跳过本次备份，继续退出流程
                overwrite && console.debug('用户在退出时取消了覆盖模式的自动备份，本次不执行备份');
                throw new Error('__SKIP_AUTO_BACKUP__');
              }
            }

            // 显示备份提示
            ElMessage.info('正在执行自动备份，请稍候...');
            
            try {
              await BackupConfigFiles(backupConfig.backupPath, backupConfig.selectedItems, overwrite);
              
              // 更新上次备份时间
              backupConfig.lastBackupTime = new Date().toISOString();
              localStorage.setItem('easytools:backup:config', JSON.stringify(backupConfig));
              
              ElMessage.success('备份完成，正在退出...');
              // 等待一下让用户看到消息
              await new Promise(resolve => setTimeout(resolve, 500));
            } catch (error) {
              console.error('自动备份失败:', error);
              ElMessage.error('备份失败: ' + (error.message || error));
              // 即使备份失败，也询问用户是否继续退出
              try {
                await this.$confirm('备份失败，是否仍要退出？', '提示', {
                  confirmButtonText: '退出',
                  cancelButtonText: '取消',
                  type: 'warning'
                });
              } catch {
                // 用户取消退出
                return;
              }
            }
          }
        }
      } catch (error) {
        if (error && error.message === '__SKIP_AUTO_BACKUP__') {
          // 用户主动跳过本次自动备份，不视为错误，直接继续退出
          console.debug('用户选择跳过本次自动备份，直接退出应用');
        } else {
          console.error('检查备份配置失败:', error);
        }
      }
      
      // 调用 Wails 退出函数前，清理只应在「真正退出应用」时清空的缓存
      try {
        clearCloudServerTerminalStorageOnExit();
        ExitApp();
      } catch (error) {
        console.error('Exit function not available:', error);
      }
    },
    
    // 处理虚机警告事件
    handleVMWarning(warning) {
      if (warning) {
        this.vmWarningMessage = warning;
        this.vmWarningVisible = true;
      }
    },
    // 确认虚机警告
    confirmVMWarning() {
      this.vmWarningVisible = false;
      this.vmWarningMessage = '';
      // 使用 sessionStorage 标记已经显示过虚机警告（仅本次会话有效）
      // 这样如果用户刷新页面或重新打开应用，Login 会再次显示
      sessionStorage.setItem('vm-warning-shown-from-login', 'true');
    },
    // ========== 全局休息提示相关方法 ==========
    handleRestDialogClose() {
      this.showRestDialog = false;
    },
    handleGoToGames() {
      this.showRestDialog = false;
      // 跳转到游戏页面
      if (this.$route.name !== 'games') {
        this.$router.push({ name: 'games' });
      }
    },
    
    // ========== 小概率弹出算法 ==========
    initSmallProbabilityCheck() {
      // 检查是否已经触发过（避免重复触发）
      const lastTriggerTime = localStorage.getItem('small_probability_last_trigger');
      const now = Date.now();
      
      // 如果12小时内已经触发过，不再触发（避免频繁弹出）
      if (lastTriggerTime) {
        const timeSinceLastTrigger = now - parseInt(lastTriggerTime, 10);
        const cooldownPeriod = 12 * 60 * 60 * 1000; // 24小时冷却期
        if (timeSinceLastTrigger < cooldownPeriod) {
          console.log('[小概率算法] 12小时内已触发过，跳过本次检查');
          return;
        }
      }
      
      // 生成0-1之间的随机数
      const randomValue = Math.random();
      
      // 小概率阈值：0.1
      const SMALL_PROBABILITY_THRESHOLD = 0.08;
      
      console.log(`[小概率算法] 随机数: ${randomValue}, 阈值: ${SMALL_PROBABILITY_THRESHOLD}`);
      
      if (randomValue < SMALL_PROBABILITY_THRESHOLD) {
        console.log('[小概率算法] 触发成功！将在10分钟后弹出提示');
        this.smallProbabilityTriggered = true;
        
        // 记录触发时间
        localStorage.setItem('small_probability_last_trigger', now.toString());
        
        // 等待3分钟（180秒）后弹出提示
        // 使用定时器，确保程序仍在运行
        this.smallProbabilityTimer = setTimeout(() => {
          // 检查是否还在应用中且没有显示对话框，且不是登录页面
          if (!this.showRestDialog && this.routeName !== 'login') {
            console.log('[小概率算法] 10分钟等待完成，弹出休息提示');
            this.showRestDialog = true;
          }
          this.smallProbabilityTriggered = false;
        }, 10 * 60 * 1000); // 10分钟 = 600000毫秒
      } else {
        console.log('[小概率算法] 未触发，继续正常运行');
      }
    },
    
    /**
     * 清理小概率定时器
     */
    clearSmallProbabilityTimer() {
      if (this.smallProbabilityTimer) {
        clearTimeout(this.smallProbabilityTimer);
        this.smallProbabilityTimer = null;
      }
    }
    // ========== 小概率弹出算法结束 ==========
  },
  computed: {
    routeName() {
      if (this.$route.path.startsWith('/cloudSecurity')) {
        return 'cloudSecurity';
      }
      return this.$route.name;
    }
  }
}
</script>

<style scoped lang="scss">
.app {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  max-width: 100vw;
  max-height: 100vh;
  background: #f5f7fa;
  position: relative;
  overflow: hidden;
}

/* 主内容容器 */
.main-container {
  display: flex;
  flex: 1;
  min-height: 0;
  max-height: 100%;
  overflow: hidden;
  position: relative;
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
    transition: width 0.3s ease;

    &.collapsed {
      width: 70px;
    }
  }

  .appMain {
    flex: 1;
    min-width: 0;
    min-height: 0;
    max-height: 100%;
    background: #ffffff;
    overflow: auto;
    position: relative;
    height: 100%;
  }
}

/* 无菜单时的布局（登录页面） */
.main-container:not(.with-menu) {
  .appMain {
    flex: 1;
    min-height: 0;
    max-height: 100%;
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

/* 虚机警告弹框样式 */
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

.vm-warning-header {
  background: linear-gradient(135deg, #ff9800 0%, #f57c00 100%);
  position: relative;
  overflow: hidden;
}

.vm-warning-header::before {
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

.vm-warning-icon-wrapper {
  position: relative;
  margin-right: 16px;
  z-index: 2;
}

.vm-warning-icon {
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

.vm-warning-icon:hover {
  transform: scale(1.05);
  background: rgba(255, 255, 255, 0.3);
}

.vm-warning-icon .el-icon {
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

.header-text {
  flex: 1;
  z-index: 2;
  position: relative;
}

.dialog-title {
  margin: 0 0 4px 0;
  color: white;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0.5px;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

.dialog-subtitle {
  margin: 0;
  color: rgba(255, 255, 255, 0.85);
  font-size: 12px;
  font-weight: 400;
  letter-spacing: 1px;
  text-transform: uppercase;
}

.vm-warning-content {
  padding: 24px;
  background: linear-gradient(to bottom, #fafafa 0%, #ffffff 100%);
}

.vm-warning-message {
  display: flex;
  gap: 16px;
  padding: 20px;
  background: linear-gradient(135deg, #fff8e1 0%, #ffe0b2 100%);
  border: 2px solid #ffb74d;
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(255, 152, 0, 0.15);
  transition: all 0.3s ease;
  margin-bottom: 16px;
}

.vm-warning-message:hover {
  box-shadow: 0 6px 16px rgba(255, 152, 0, 0.25);
  transform: translateY(-2px);
}

.message-icon-wrapper {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 10px;
  background: linear-gradient(135deg, #ff9800 0%, #f57c00 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 8px rgba(255, 152, 0, 0.3);
}

.vm-warning-message .warning-icon {
  font-size: 24px;
  color: white;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.2));
}

.message-text {
  flex: 1;
}

.message-title {
  margin: 0 0 8px 0;
  color: #e65100;
  font-size: 15px;
  font-weight: 600;
}

.message-content {
  margin: 0;
  color: #bf360c;
  font-size: 14px;
  line-height: 1.7;
}

.vm-warning-tips {
  padding: 12px 16px;
  background: linear-gradient(135deg, #e3f2fd 0%, #bbdefb 100%);
  border-left: 4px solid #2196f3;
  border-radius: 8px;
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

.glass-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  background: transparent;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 0;
}

.vm-warning-footer {
  background: transparent;
}

/* 全局休息提示对话框样式 */
.rest-dialog-content {
  text-align: center;
  padding: 20px;
}

.rest-icon {
  font-size: 64px;
  margin-bottom: 20px;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.1); }
}

.rest-dialog-content h3 {
  margin: 0 0 10px 0;
  font-size: 24px;
  color: #333;
}

.rest-dialog-content p {
  margin: 10px 0;
  font-size: 16px;
  color: #666;
}

.rest-tip {
  font-size: 14px;
  color: #999;
  font-style: italic;
}

.footer-btn {
  border-radius: 6px;
  min-width: 80px;
  height: 32px;
  font-weight: 500;
  font-size: 13px;
  transition: all 0.2s ease;
  padding: 0 16px;
}

.footer-btn:active {
  transform: scale(0.98);
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
</style>

<style>
/* 全局重置样式 - 防止滚动时出现空白 */
html, body {
  margin: 0;
  padding: 0;
  height: 100%;
  width: 100%;
  max-width: 100vw;
  max-height: 100vh;
  overflow: hidden;
  overflow-x: hidden;
  overflow-y: hidden;
  box-sizing: border-box;
}

*, *::before, *::after {
  box-sizing: border-box;
}

#app {
  height: 100vh;
  width: 100vw;
  max-width: 100vw;
  max-height: 100vh;
  overflow: hidden;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', '微软雅黑', Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  /* optimizeLegibility 会启用字距/连字重排，输入时每键可能触发昂贵布局；全局避免使用 */
  text-rendering: auto;
}

/* 表单控件强制走快速文本路径，避免从 #app 继承“美化排版”导致打字卡顿 */
input,
textarea,
.el-input__inner,
.el-input__wrapper,
.el-textarea__inner {
  text-rendering: auto !important;
}

/* 退出确认：Element 对话框透明壳 */
.glass-exit-dialog.el-dialog {
  background: transparent;
  box-shadow: none;
  border-radius: 16px;
}

.glass-exit-dialog .el-dialog__header {
  display: none;
}

.glass-exit-dialog .el-dialog__body {
  padding: 0 !important;
}

.glass-exit-overlay {
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  background-color: rgba(0, 0, 0, 0.3);
}

/* 确保标题栏在最顶层 */
.custom-titlebar {
  z-index: 1000;
}

/* 统一 ElMessage 样式 - 使用默认样式，仅统一位置和基本配置 */
.el-message {
  top: 20px !important;
}

/* 虚机警告弹框全局样式 */
.glass-vm-warning-dialog .el-dialog {
  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(20px);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
  border-radius: 20px;
  overflow: hidden;
  border: none;
}

.glass-vm-warning-dialog .el-dialog__header {
  display: none;
}

.glass-vm-warning-dialog .el-dialog__body {
  padding: 0 !important;
  background: transparent;
}

.glass-vm-warning-dialog .el-dialog__footer {
  padding: 0 !important;
  background: transparent;
}

.glass-vm-warning-dialog .el-overlay {
  backdrop-filter: blur(8px);
  background: rgba(0, 0, 0, 0.3);
}

.glass-vm-warning-dialog .el-overlay-dialog {
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>