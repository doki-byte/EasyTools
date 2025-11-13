<template>
  <div class="custom-titlebar" style="--wails-draggable: drag;">
    <!-- 应用图标和名称区域（可拖动） -->
    <div class="titlebar-left" @dblclick="toggleMaximize">
      <div class="app-brand">
        <div class="app-icon">
          <div class="icon-placeholder"><img style="width: 20px; height: 20px; " src="/assets/system/appicon.png" alt=""/></div>
        </div>
        <span class="app-title">EasyTools</span>
      </div>
    </div>

    <!-- 窗口标题区域（可拖动） -->
    <div class="titlebar-center">
      <span class="window-title">{{ currentTitle }}</span>
    </div>

    <!-- 功能按钮区域（不可拖动） -->
    <div class="titlebar-right">
      <!-- 动态浏览器打开按钮 -->
      <el-tooltip v-if="showBrowserButton" content="在浏览器中打开" placement="bottom" effect="light">
        <button
            class="titlebar-btn proxy-indicator"
            @click="openInBrowser"
            style="--wails-draggable: no-drag;"
        >
          <el-icon size="16">
            <Promotion />
          </el-icon>
        </button>
      </el-tooltip>

      <!-- 代理状态指示器 -->
      <el-tooltip :content="proxyTooltip" placement="bottom" effect="light">
        <button
            class="titlebar-btn proxy-indicator"
            :class="{
            'active': proxyStatus.globalEnabled,
            'testing': proxyLoading,
            'error': proxyError
          }"
            @click="openProxySettings"
            style="--wails-draggable: no-drag;"
        >
          <el-icon size="16">
            <Connection />
          </el-icon>
          <span class="proxy-badge" v-if="proxyStatus.enabledModules > 0">
            {{ proxyStatus.enabledModules }}
          </span>
          <!-- 状态指示点 -->
          <span class="status-dot" :class="proxyStatusClass"></span>
        </button>
      </el-tooltip>

      <!-- Github -->
      <el-tooltip content="Github" placement="bottom" effect="light">
        <button
            class="titlebar-btn proxy-indicator"
            @click="openGithub"
            style="--wails-draggable: no-drag;"
        >
          <el-icon size="16">
            <IconGithub />
          </el-icon>
        </button>
      </el-tooltip>

      <!-- Github Issues -->
      <el-tooltip content="提交BUG" placement="bottom" effect="light">
        <button
            class="titlebar-btn proxy-indicator"
            @click="openGithubIssues"
            style="--wails-draggable: no-drag;"
        >
          <el-icon size="16">
            <IconBug />
          </el-icon>
        </button>
      </el-tooltip>

      <!-- 窗口控制按钮 -->
      <div class="window-controls">
        <button
            class="titlebar-btn control-btn"
            @click="minimize"
            title="最小化"
            style="--wails-draggable: no-drag;"
        >
          <el-icon><Minus /></el-icon>
        </button>
        <button
            class="titlebar-btn control-btn"
            @click="toggleMaximize"
            :title="isMaximized ? '还原' : '最大化'"
            style="--wails-draggable: no-drag;"
        >
          <el-icon v-if="!isMaximized"><FullScreen /></el-icon>
          <el-icon v-else><CopyDocument /></el-icon>
        </button>
        <button
            class="titlebar-btn control-btn close-btn"
            @click="close"
            title="关闭"
            style="--wails-draggable: no-drag;"
        >
          <el-icon><Close /></el-icon>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, onUnmounted, nextTick, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Connection, Minus, CopyDocument, Close, FullScreen, Promotion } from '@element-plus/icons-vue'
import { GetProxyStatus } from "../../wailsjs/go/proxy/ProxyManager"
import {EventsOn, EventsOff, BrowserOpenURL} from '../../wailsjs/runtime'
import { WindowMinimise, WindowToggleMaximise, Quit } from "../../wailsjs/runtime"
import {ElNotification} from 'element-plus'

const router = useRouter()
const route = useRoute()
const proxyLoading = ref(false)
const proxyError = ref(false)
const isMaximized = ref(false)

// 使用 ref 来跟踪已经显示过提示的路由
const shownTips = ref(new Set())

// 代理状态
const proxyStatus = reactive({
  globalEnabled: false,
  enabledModules: 0,
  disabledModules: 0,
  totalModules: 0
})

// 默认标题
const defaultTitle = ref('EasyTools：一款实用的渗透测试工具箱 v1.9.5')

// 监听窗口状态变化
EventsOn('window-state-changed', (state) => {
  isMaximized.value = state === 'maximised'
})

// 计算是否显示浏览器按钮
const showBrowserButton = computed(() => {
  return route.name === 'connect' || route.name === 'cyberchef'
})

// 显示提示消息的函数
const showBrowserTip = (routeName) => {
  // 检查本次会话中是否已经显示过提示
  if (!shownTips.value.has(routeName)) {
    // 延迟显示Message提示
    setTimeout(() => {
      let message = '点击右上角按钮可以在浏览器中打开此页面'

      ElNotification({
        title: '提示',
        message: message,
        type: 'info',
        position: 'top-right',  // 默认就是右上角
        duration: 3000          // 可选：自动关闭时间，单位 ms
      });

      // 标记为已显示（仅本次会话有效）
      shownTips.value.add(routeName)
    }, 1000)
  }
}

// 监听路由变化
watch(() => route.name, (newRouteName) => {
  if (newRouteName === 'connect' || newRouteName === 'cyberchef') {
    showBrowserTip(newRouteName)
  }
}, { immediate: true })

// 在浏览器中打开
const openInBrowser = () => {
  let url = ''
  if (route.name === 'connect') {
    url = 'http://127.0.0.1:52868/'
  } else if (route.name === 'cyberchef') {
    url = 'http://127.0.0.1:52867/CyberChef/index.html'
  }

  if (url) {
    BrowserOpenURL(url)
  }
}

// 窗口标题计算
const currentTitle = computed(() => {
  const docTitle = document.title
  return docTitle && docTitle !== 'EasyTools' ? docTitle : defaultTitle.value
})

// 打开Github
const openGithub = () => {
  BrowserOpenURL("https://github.com/doki-byte/EasyTools")
}

// 打开GithubIssues
const openGithubIssues = () => {
  BrowserOpenURL("https://github.com/doki-byte/EasyTools/issues")
}

// 计算代理状态类
const proxyStatusClass = computed(() => {
  if (proxyLoading.value) return 'loading'
  if (proxyError.value) return 'error'
  if (proxyStatus.globalEnabled) return 'enabled'
  return 'disabled'
})

// 计算代理提示信息
const proxyTooltip = computed(() => {
  if (proxyLoading.value) {
    return '加载代理状态...'
  }

  if (proxyError.value) {
    return '代理状态加载失败，点击重试'
  }

  let tip = `全局代理: ${proxyStatus.globalEnabled ? '✅ 已启用' : '❌ 已禁用'}`

  if (proxyStatus.enabledModules > 0) {
    tip += `\n${proxyStatus.enabledModules} 个模块强制使用代理`
  }
  if (proxyStatus.disabledModules > 0) {
    tip += `\n${proxyStatus.disabledModules} 个模块强制直连`
  }

  return tip
})

// 打开代理设置
const openProxySettings = () => {
  router.push('/systemManage?tab=proxy-config')
}

// 加载代理状态
const loadProxyStatus = async () => {
  proxyLoading.value = true
  proxyError.value = false
  try {
    const status = await GetProxyStatus()

    if (status && typeof status === 'object') {
      Object.assign(proxyStatus, {
        globalEnabled: !!status.globalEnabled,
        enabledModules: Number(status.enabledModules) || 0,
        disabledModules: Number(status.disabledModules) || 0,
        totalModules: Number(status.totalModules) || 0
      })
    } else {
      Object.assign(proxyStatus, {
        globalEnabled: false,
        enabledModules: 0,
        disabledModules: 0,
        totalModules: 0
      })
    }
  } catch (error) {
    proxyError.value = true
    Object.assign(proxyStatus, {
      globalEnabled: false,
      enabledModules: 0,
      disabledModules: 0,
      totalModules: 0
    })
  } finally {
    proxyLoading.value = false
  }
}

// 事件监听设置
const setupProxyEvents = () => {
  EventsOn('proxy-status-changed', (status) => {
    if (status && typeof status === 'object') {
      Object.assign(proxyStatus, {
        globalEnabled: !!status.globalEnabled,
        enabledModules: Number(status.enabledModules) || 0,
        disabledModules: Number(status.disabledModules) || 0,
        totalModules: Number(status.totalModules) || 0
      })
      proxyError.value = false
    }
  })

  EventsOn('global-proxy-updated', () => {
    loadProxyStatus()
  })

  EventsOn('module-proxy-status-changed', () => {
    loadProxyStatus()
  })

  EventsOn('proxy-error', (error) => {
    proxyError.value = true
  })

  EventsOn('title-updated', (newTitle) => {
    if (newTitle && typeof newTitle === 'string') {
      defaultTitle.value = newTitle
    }
  })
}

// 窗口控制方法
const minimize = () => {
  WindowMinimise()
}

const toggleMaximize = () => {
  WindowToggleMaximise()
}

const close = () => {
  Quit()
}

// 初始化
onMounted(async () => {
  await nextTick()
  await loadProxyStatus()
  setupProxyEvents()
})

onUnmounted(() => {
  EventsOff('proxy-status-changed')
  EventsOff('global-proxy-updated')
  EventsOff('module-proxy-status-changed')
  EventsOff('proxy-error')
  EventsOff('window-state-changed')
  EventsOff('title-updated')
})
</script>

<style scoped>
/* 保持原有的样式不变 */
.custom-titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 30px;
  background: linear-gradient(135deg, #7f91e3 0%, rgba(78, 159, 237, 0.58) 100%);
  color: white;
  padding: 0 0 0 12px;
  user-select: none;
  position: relative;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.custom-titlebar::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,0.3), transparent);
}

.titlebar-left,
.titlebar-center,
.titlebar-right {
  display: flex;
  align-items: center;
  height: 100%;
}

.titlebar-left {
  flex: 1;
}

.titlebar-center {
  flex: 2;
  justify-content: center;
}

.titlebar-right {
  flex: 1;
  justify-content: flex-end;
}

.app-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-radius: 6px;
  transition: background-color 0.2s;
}

.app-brand:hover {
  background: rgba(255, 255, 255, 0.1);
}

.app-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.icon-placeholder {
  font-size: 16px;
  filter: drop-shadow(0 1px 2px rgba(0,0,0,0.2));
}

.app-title {
  font-weight: 600;
  font-size: 14px;
  letter-spacing: 0.5px;
  text-shadow: 0 1px 2px rgba(0,0,0,0.2);
}

.window-title {
  font-size: 13px;
  opacity: 0.9;
  font-weight: 500;
  text-shadow: 0 1px 2px rgba(0,0,0,0.2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.titlebar-btn {
  border: none;
  background: transparent;
  padding: 6px 10px;
  cursor: pointer;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  color: white;
  position: relative;
}

.titlebar-btn:hover {
  background: rgba(255, 255, 255, 0.15);
}

.control-btn {
  width: 32px;
  height: 32px;
  margin-left: 2px;
}

.control-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.close-btn:hover {
  background: #ff5f57 !important;
}

.window-controls {
  display: flex;
  align-items: center;
  margin-left: 8px;
}

/* 代理指示器样式 */
.proxy-indicator {
  position: relative;
  margin-right: 4px;
  padding: 6px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.proxy-indicator.active {
  background: rgba(76, 175, 80, 0.2);
  border-color: rgba(76, 175, 80, 0.3);
}

.proxy-indicator.testing {
  animation: pulse 1.5s infinite;
  background: rgba(255, 193, 7, 0.2);
}

.proxy-indicator.error {
  background: rgba(244, 67, 54, 0.2);
  border-color: rgba(244, 67, 54, 0.3);
}

@keyframes pulse {
  0% { opacity: 1; }
  50% { opacity: 0.7; }
  100% { opacity: 1; }
}

.proxy-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  background: #ff4757;
  color: white;
  border-radius: 50%;
  width: 16px;
  height: 16px;
  font-size: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
  z-index: 2;
  font-weight: bold;
  box-shadow: 0 1px 3px rgba(0,0,0,0.3);
}

/* 状态指示点 */
.status-dot {
  position: absolute;
  bottom: 4px;
  right: 4px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  border: 1px solid rgba(255, 255, 255, 0.8);
  z-index: 1;
}

.status-dot.enabled {
  background: #4caf50;
  box-shadow: 0 0 4px #4caf50;
}

.status-dot.disabled {
  background: #9e9e9e;
}

.status-dot.loading {
  background: #ffc107;
  animation: pulse 1s infinite;
  box-shadow: 0 0 4px #ffc107;
}

.status-dot.error {
  background: #f44336;
  box-shadow: 0 0 4px #f44336;
}

/* 响应式设计 */
@media (max-width: 600px) {
  .window-title {
    display: none;
  }

  .titlebar-center {
    flex: 0;
  }
}
</style>