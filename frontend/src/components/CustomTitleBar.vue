<template>
  <div class="custom-titlebar" :class="{ 'mac-style': isMac }" :style="titlebarStyle" style="--wails-draggable: drag;">
    <!-- macOS 样式：窗口控制按钮在左侧 -->
    <div v-if="isMac" class="window-controls mac-controls">
      <button class="titlebar-btn control-btn mac-close-btn" @click="close" title="关闭"
        style="--wails-draggable: no-drag;">
        <el-icon>
          <Close />
        </el-icon>
      </button>
      <button class="titlebar-btn control-btn mac-minimize-btn" @click="minimize" title="最小化"
        style="--wails-draggable: no-drag;">
        <el-icon>
          <Minus />
        </el-icon>
      </button>
      <button class="titlebar-btn control-btn mac-maximize-btn" @click="toggleMaximize"
        :title="isMaximized ? '还原' : '最大化'" style="--wails-draggable: no-drag;">
        <el-icon v-if="!isMaximized">
          <FullScreen />
        </el-icon>
        <el-icon v-else>
          <CopyDocument />
        </el-icon>
      </button>
    </div>

    <!-- 应用图标和名称区域（可拖动） -->
    <div class="titlebar-left" :class="{ 'mac-left': isMac }" @dblclick="toggleMaximize">
      <div class="app-brand">
        <div class="app-icon">
          <div class="icon-placeholder"><img style="width: 20px; height: 20px; " src="/assets/system/appicon.png"
              alt="" /></div>
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
      <el-tooltip :content="globalSearchTooltipText" placement="bottom" effect="light">
        <button type="button" class="titlebar-btn proxy-indicator" @click="onGlobalSearchClick" style="--wails-draggable: no-drag;">
          <el-icon size="16"><Search /></el-icon>
        </button>
      </el-tooltip>

      <!-- 动态浏览器打开按钮 -->
      <el-tooltip v-if="showBrowserButton" :content="browserTooltip" placement="bottom" effect="light">
        <button class="titlebar-btn proxy-indicator" @click="openInBrowser" style="--wails-draggable: no-drag;">
          <el-icon size="16">
            <Promotion />
          </el-icon>
        </button>
      </el-tooltip>

      <!-- 代理状态指示器 -->
      <el-tooltip :content="proxyTooltip" placement="bottom" effect="light">
        <button class="titlebar-btn proxy-indicator" :class="{
          'system-proxy': proxyStatus.systemEnabled,
          'program-proxy': proxyStatus.globalEnabled && !proxyStatus.systemEnabled,
          'no-proxy': !proxyStatus.globalEnabled && !proxyStatus.systemEnabled,
          'testing': proxyLoading,
          'error': proxyError
        }" @click="openProxySettings" style="--wails-draggable: no-drag;">
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
        <button class="titlebar-btn proxy-indicator" @click="openGithub" style="--wails-draggable: no-drag;">
          <el-icon size="16">
            <Link />
          </el-icon>
        </button>
      </el-tooltip>

      <!-- Github Issues -->
      <el-tooltip content="提交BUG" placement="bottom" effect="light">
        <button class="titlebar-btn proxy-indicator" @click="openGithubIssues" style="--wails-draggable: no-drag;">
          <el-icon size="16">
            <WarnTriangleFilled />
          </el-icon>
        </button>
      </el-tooltip>

      <!-- 系统日志按钮 -->
      <el-tooltip content="系统日志" placement="bottom" effect="light">
        <button class="titlebar-btn proxy-indicator" @click.stop="logDrawerVisible = true"
          style="--wails-draggable: no-drag;">
          <el-icon size="16">
            <Document />
          </el-icon>
          <span class="log-badge" v-if="unreadLogCount > 0">
            {{ unreadLogCount > 99 ? '99+' : unreadLogCount }}
          </span>
        </button>
      </el-tooltip>

      <!-- 版本更新检测按钮 -->
      <el-tooltip :content="hasUpdate ? `发现新版本 ${latestVersion}` : '检查更新'" placement="bottom" effect="light">
        <button class="titlebar-btn proxy-indicator" @click="handleCheckUpdate" style="--wails-draggable: no-drag;">
          <el-icon size="16">
            <Odometer />
          </el-icon>
          <el-badge :is-dot="hasUpdate" style="margin-left: 4px;"></el-badge>
        </button>
      </el-tooltip>

      <!-- Windows/Linux 样式：窗口控制按钮在右侧 -->
      <div v-if="!isMac" class="window-controls">
        <button class="titlebar-btn control-btn" @click="minimize" title="最小化" style="--wails-draggable: no-drag;">
          <el-icon>
            <Minus />
          </el-icon>
        </button>
        <button class="titlebar-btn control-btn" @click="toggleMaximize" :title="isMaximized ? '还原' : '最大化'"
          style="--wails-draggable: no-drag;">
          <el-icon v-if="!isMaximized">
            <FullScreen />
          </el-icon>
          <el-icon v-else>
            <CopyDocument />
          </el-icon>
        </button>
        <button class="titlebar-btn control-btn close-btn" @click="close" title="关闭"
          style="--wails-draggable: no-drag;">
          <el-icon>
            <Close />
          </el-icon>
        </button>
      </div>
    </div>

    <!-- 更新对话框 -->
    <el-dialog
        v-model="updateDialogVisible"
        title="检查更新"
        width="520px"
        :close-on-click-modal="false"
        :show-close="false"
        class="glass-update-dialog"
        :before-close="handleDialogClose"
    >
      <div class="glass-dialog-content">
        <div class="dialog-header update-header">
          <div class="update-icon-wrapper">
            <div class="update-icon">
              <el-icon size="20"><Odometer /></el-icon>
            </div>
          </div>
          <div class="header-text">
            <h3 class="dialog-title">发现新版本</h3>
          </div>
        </div>

        <div class="update-content">
          <div class="version-section">
            <span class="section-label">最新版本:</span>
            <el-tag class="version-tag" effect="dark">{{ latestVersion }}</el-tag>
          </div>

          <div class="description-section">
            <span class="section-label">更新内容:</span>
            <div class="description-text">
              <el-scrollbar max-height="200px">
                <p
                    v-for="(line, index) in formattedDescription"
                    :key="index"
                    class="description-line"
                >
                  {{ line }}
                </p>
              </el-scrollbar>
            </div>
          </div>

          <!-- 下载进度 -->
          <div v-if="isDownloading" class="progress-section">
            <span class="section-label">下载进度:</span>
            <div class="progress-content">
              <el-progress
                  :percentage="downloadProgress"
                  :stroke-width="8"
                  :show-text="true"
                  status="success"
              />
              <div class="progress-text">{{ downloadStatus }}</div>
            </div>
          </div>

          <div class="url-section">
            <span class="section-label">下载地址:</span>
            <div class="url-text">
              <el-tag
                  v-if="releaseUrl"
                  class="url-tag"
                  effect="plain"
                  @click="handleUrlClick"
              >
                {{ releaseUrl }}
              </el-tag>
              <span v-else class="no-url">暂无下载地址</span>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="glass-dialog-footer">
          <el-button
              class="footer-btn skip-btn"
              size="small"
              @click="handleSkipToday"
              :disabled="isDownloading"
          >
            今日不再提示
          </el-button>
          <el-button
              class="footer-btn cancel-btn"
              size="small"
              @click="handleCancelUpdate"
              :disabled="!isDownloading || isCancelling"
          >
            {{ isCancelling ? '取消中...' : '取消下载' }}
          </el-button>
          <el-button
              class="footer-btn download-btn"
              @click="handleDownloadUpdate"
              :disabled="isDownloading"
              :loading="isDownloading"
              type="success"
          >
            <el-icon style="margin-right: 4px;"><Download /></el-icon>
            {{ isDownloading ? `下载中...` : '自动下载更新' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 系统日志侧边栏 -->
    <el-drawer v-model="logDrawerVisible" title="系统日志" size="420px" direction="rtl" append-to-body :with-header="true">
      <div class="log-drawer">
        <div class="log-toolbar">
          <div class="log-toolbar-left">
            <el-select v-model="logLevelFilter" size="small" style="width: 120px">
              <el-option label="全部级别" value="all" />
              <el-option label="信息" value="info" />
              <el-option label="调试" value="debug" />
              <el-option label="警告" value="warning" />
              <el-option label="错误" value="error" />
            </el-select>
          </div>
          <div class="log-toolbar-right">
            <el-button type="text" size="small" @click="toggleLogAutoScroll">
              {{ logAutoScroll ? '暂停滚动' : '自动滚动' }}
            </el-button>
            <el-button type="text" size="small" @click="copyLogs" :disabled="filteredLogs.length === 0">
              复制
            </el-button>
            <el-button type="text" size="small" @click="clearLogs" :disabled="logs.length === 0">
              清空
            </el-button>
          </div>
        </div>

        <div class="log-list" ref="logListEl">
          <div v-for="(log, index) in filteredLogs" :key="index" class="log-item"
            :class="'log-level-' + (log.level || 'info')">
            <span class="log-time">{{ formatLogTime(log.timestamp) }}</span>
            <span class="log-source" v-if="log.source">[{{ log.source }}]</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
          <div v-if="filteredLogs.length === 0" class="log-empty">
            暂无日志，执行任务时的运行信息会在这里展示。
          </div>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { inject, ref, reactive, onMounted, computed, onUnmounted, nextTick, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Connection, Minus, CopyDocument, Close, FullScreen, Promotion, Document, Download, Odometer, Search, Link, WarnTriangleFilled } from '@element-plus/icons-vue'
import { GetProxyStatus } from "../../wailsjs/go/proxy/ProxyManager"
import { discoverLocalPortConfig } from '@/utils/localServicePorts'
import { EventsOn, EventsOff, BrowserOpenURL } from '../../wailsjs/runtime'
import { WindowMinimise, WindowToggleMaximise, EventsEmit } from "../../wailsjs/runtime"
import { ElNotification, ElMessage, ElMessageBox } from 'element-plus'
import { GetOs } from '../../wailsjs/go/system/System'
import { GetLatestRelease, DownloadAndUpdate, RestartApplication, CancelDownload } from '../../wailsjs/go/system/Update'
import { getStoredGlobalSearchHotkey } from '@/utils/globalSearchHotkey'

const openGlobalSearch = inject('openGlobalSearch', null)

function onGlobalSearchClick() {
  if (typeof openGlobalSearch === 'function') {
    openGlobalSearch()
  }
}

const router = useRouter()
const route = useRoute()
const proxyLoading = ref(false)
const proxyError = ref(false)
const isMaximized = ref(false)
const isMac = ref(false)

// 字体颜色配置
const fontColor = ref('#ffffff')
const fontWeightRatio = ref(100)
const shadowRatio = ref(100)
const STORAGE_KEY = 'easytools:theme:background'

const globalSearchAccelLabel = ref(getStoredGlobalSearchHotkey())
const globalSearchTooltipText = computed(() => `全局搜索 ${globalSearchAccelLabel.value}`)

// 加载字体颜色配置
const loadFontColor = () => {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) {
      const config = JSON.parse(saved)
      if (config.fontColor) {
        fontColor.value = config.fontColor
      }
      if (config.fontWeightRatio !== undefined) {
        fontWeightRatio.value = config.fontWeightRatio
      }
      if (config.shadowRatio !== undefined) {
        shadowRatio.value = config.shadowRatio
      }
    }
  } catch (e) {
    console.error('加载字体颜色配置失败:', e)
  }
}

// 计算标题栏样式
const titlebarStyle = computed(() => {
  // 计算加粗程度：默认 600，100% 对应 600，0% 对应 400，200% 对应 800
  const fontWeight = 400 + (fontWeightRatio.value / 100) * 400
  // 计算阴影程度：默认 0 1px 3px rgba(0, 0, 0, 0.12)，根据比例调整透明度
  const shadowOpacity1 = (shadowRatio.value / 100) * 0.12
  const shadowOpacity2 = (shadowRatio.value / 100) * 0.24
  return {
    '--titlebar-font-color': fontColor.value,
    '--titlebar-font-weight': fontWeight,
    '--titlebar-shadow': `0 1px 3px rgba(0, 0, 0, ${shadowOpacity1}), 0 1px 2px rgba(0, 0, 0, ${shadowOpacity2})`,
    color: fontColor.value
  }
})

// 新增：端口配置
const portConfig = ref(null)
const portConfigLoading = ref(false)

// 使用 ref 来跟踪已经显示过提示的路由
const shownTips = ref(new Set())

// 日志侧边栏状态
const logDrawerVisible = ref(false)
const logs = ref([])
const unreadLogCount = ref(0)
const logLevelFilter = ref('all')
const logAutoScroll = ref(true)
const logListEl = ref(null)
const MAX_LOGS = 1000

// 版本更新相关状态
const updateDialogVisible = ref(false)
const hasUpdate = ref(false)
const latestVersion = ref('')
const releaseUrl = ref('')
const releaseDescription = ref('')
const loading = ref(false)

// 下载状态
const isDownloading = ref(false)
const isCancelling = ref(false)
const downloadProgress = ref(0)
const downloadStatus = ref('')

// 代理状态
const proxyStatus = reactive({
  globalEnabled: false,
  systemEnabled: false,
  enabledModules: 0,
  disabledModules: 0,
  totalModules: 0
})

// 计算过滤后的日志
const filteredLogs = computed(() => {
  if (logLevelFilter.value === 'all') return logs.value
  return logs.value.filter(log => (log.level || 'info') === logLevelFilter.value)
})

// 日志批量更新队列
const logQueue = ref([])
let logBatchTimer = null

// 推入日志并截断长度（优化：批量更新，避免频繁DOM操作导致卡顿）
const pushLog = (entry) => {
  if (!entry || !entry.message) return

  // 如果日志抽屉关闭，只更新未读计数，不立即添加到日志数组（减少不必要的处理）
  if (!logDrawerVisible.value) {
    unreadLogCount.value++
    // 仍然添加到队列，但延迟处理
    logQueue.value.push({
      message: String(entry.message),
      level: entry.level || 'info',
      timestamp: entry.timestamp || new Date().toISOString(),
      source: entry.source || '系统'
    })
    
    // 批量处理队列（每100ms处理一次，避免频繁更新）
    if (!logBatchTimer) {
      logBatchTimer = setTimeout(() => {
        if (logQueue.value.length > 0) {
          logs.value.push(...logQueue.value)
          logQueue.value = []
          
          // 限制日志数量
          if (logs.value.length > MAX_LOGS) {
            logs.value = logs.value.slice(-Math.floor(MAX_LOGS / 2))
          }
        }
        logBatchTimer = null
      }, 100)
    }
    return
  }

  // 日志抽屉打开时，立即添加但使用批量更新优化
  logQueue.value.push({
    message: String(entry.message),
    level: entry.level || 'info',
    timestamp: entry.timestamp || new Date().toISOString(),
    source: entry.source || '系统'
  })

  // 批量处理（每50ms处理一次，平衡实时性和性能）
  if (!logBatchTimer) {
    logBatchTimer = setTimeout(() => {
      if (logQueue.value.length > 0) {
        logs.value.push(...logQueue.value)
        logQueue.value = []
        
        // 限制日志数量
        if (logs.value.length > MAX_LOGS) {
          logs.value = logs.value.slice(-Math.floor(MAX_LOGS / 2))
        }
      }
      logBatchTimer = null
    }, 50)
  }
}

// 监听日志变化自动滚动（优化：使用防抖，减少频繁滚动）
let scrollTimer = null
watch(
  () => filteredLogs.value.length,
  async () => {
    if (!logAutoScroll.value || !logDrawerVisible.value) return
    
    // 清除之前的定时器
    if (scrollTimer) {
      clearTimeout(scrollTimer)
    }
    
    // 防抖：延迟100ms执行滚动，避免频繁DOM操作
    scrollTimer = setTimeout(async () => {
      await nextTick()
      const el = logListEl.value
      if (el) {
        el.scrollTop = el.scrollHeight
      }
      scrollTimer = null
    }, 100)
  }
)

// 打开/关闭抽屉时重置未读计数，并处理积压的日志
watch(logDrawerVisible, (visible) => {
  if (visible) {
    unreadLogCount.value = 0
    // 打开抽屉时，立即处理队列中积压的日志
    if (logQueue.value.length > 0) {
      logs.value.push(...logQueue.value)
      logQueue.value = []
      
      // 限制日志数量
      if (logs.value.length > MAX_LOGS) {
        logs.value = logs.value.slice(-Math.floor(MAX_LOGS / 2))
      }
    }
  }
})

// 日志工具方法
const formatLogTime = (ts) => {
  if (!ts) return ''
  try {
    const d = new Date(ts)
    return d.toLocaleTimeString()
  } catch {
    return String(ts)
  }
}

const clearLogs = () => {
  logs.value = []
  unreadLogCount.value = 0
}

const toggleLogAutoScroll = () => {
  logAutoScroll.value = !logAutoScroll.value
}

const copyLogs = () => {
  if (!logs.value.length) return
  const text = filteredLogs.value
    .map(l => `[${formatLogTime(l.timestamp)}]${l.source ? ' [' + l.source + ']' : ''} ${l.message}`)
    .join('\n')
  navigator.clipboard?.writeText(text)
}

// 默认标题
const defaultTitle = ref('EasyTools：一款实用的渗透测试工具箱 v2.3.0')

// 监听窗口状态变化
try {
  EventsOn('window-state-changed', (state) => {
    isMaximized.value = state === 'maximised'
  })
} catch (error) {
  console.error('注册窗口状态监听失败:', error)
}

// 日志事件监听（全局）
try {
  EventsOn('log_event', (eventData) => {
    // pcapView 等模块使用的通用日志事件
    pushLog({
      message: eventData?.message ?? '',
      level: eventData?.level || 'info',
      timestamp: eventData?.timestamp,
      source: eventData?.source || '分析模块'
    })
  })

  EventsOn('log_update', (payload) => {
    // 代理模块使用的日志事件，payload 可能是字符串或对象
    if (typeof payload === 'string') {
      pushLog({ message: payload, level: 'info', source: '代理模块' })
    } else if (payload && typeof payload === 'object') {
      pushLog({
        message: payload.message || JSON.stringify(payload),
        level: payload.level || 'info',
        timestamp: payload.timestamp,
        source: payload.source || '代理模块'
      })
    }
  })
} catch (error) {
  console.error('注册日志事件监听失败:', error)
}

// 计算是否显示浏览器按钮
// 目前仅在需要在浏览器中打开的模块（如 ctfTools）中显示
const showBrowserButton = computed(() => {
  return route.name === 'ctfTools'
})

// 新增：计算浏览器按钮提示
const browserTooltip = computed(() => {
  if (!portConfig.value) {
    return '加载端口配置中...'
  }

  let tooltip = '在浏览器中打开'
  if (route.name === 'ctfTools') {
    tooltip += ` (主服务端口: ${portConfig.value.mainPort})`
  }
  return tooltip
})

// 在浏览器中打开
const openInBrowser = () => {
  if (!portConfig.value) {
    ElNotification({
      title: '错误',
      message: '端口配置未加载，请稍后重试',
      type: 'error',
      duration: 2000
    })
    return
  }

  let url = ''
  if (route.name === 'ctfTools') {
    url = `http://127.0.0.1:${portConfig.value.mainPort}/CyberChef/index.html`
  }

  if (url) {
    try {
      BrowserOpenURL(url)
    } catch (error) {
      console.error('打开浏览器失败:', error)
      window.open(url, '_blank')
    }
  } else {
    window.open(url, '_blank')
  }
}

// 窗口标题计算
const currentTitle = computed(() => {
  const docTitle = document.title
  return docTitle && docTitle !== 'EasyTools' ? docTitle : defaultTitle.value
})

// 新增：加载端口配置（Wails 绑定 → 本地缓存 → 本机主程序探测，兼容网页预览）
const loadPortConfig = async () => {
  portConfigLoading.value = true
  try {
    const config = await discoverLocalPortConfig()
    if (config) {
      portConfig.value = config
      console.log('端口配置加载成功:', config)
    } else {
      // 完全无后端时保留历史固定端口兜底（仅影响浏览器打开等弱功能）
      portConfig.value = {
        mainPort: 52867,
        ftpPort: 52869,
        fuzzPort: 52870
      }
      console.warn('未检测到本地 EasyTools 服务，使用默认端口兜底')
    }
  } finally {
    portConfigLoading.value = false
  }
}

// 打开Github
const openGithub = () => {
  try {
    BrowserOpenURL("https://github.com/doki-byte/EasyTools")
  } catch (error) {
    console.error('打开浏览器失败:', error)
  }
}

// 打开GithubIssues
const openGithubIssues = () => {
  try {
    BrowserOpenURL("https://github.com/doki-byte/EasyTools/issues")
  } catch (error) {
    console.error('打开浏览器失败:', error)
  }
}

// 计算代理状态类
const proxyStatusClass = computed(() => {
  if (proxyLoading.value) return 'loading'
  if (proxyError.value) return 'error'
  if (proxyStatus.systemEnabled) return 'system-enabled'
  if (proxyStatus.globalEnabled) return 'program-enabled'
  return 'no-proxy'
})

// 计算代理提示信息
const proxyTooltip = computed(() => {
  if (proxyLoading.value) {
    return '加载代理状态...'
  }

  if (proxyError.value) {
    return '代理状态加载失败，点击重试'
  }

  let tip = ''
  if (proxyStatus.systemEnabled) {
    tip = `系统代理: ✅ 已启用`
  } else if (proxyStatus.globalEnabled) {
    tip = `程序代理: ✅ 已启用`
  } else {
    tip = `代理状态: ❌ 未启用`
  }

  if (proxyStatus.enabledModules > 0) {
    tip += `\n${proxyStatus.enabledModules} 个模块强制使用代理`
  }
  if (proxyStatus.disabledModules > 0) {
    tip += `\n${proxyStatus.disabledModules} 个模块强制直连`
  }

  return tip
})

// 格式化更新说明
const formattedDescription = computed(() => {
  if (!releaseDescription.value) return []
  return releaseDescription.value
      .replace(/\n(?!\n?$)/g, '\n')
      .split('\n')
      .filter(line => line.trim() !== '')
})

// 版本更新逻辑
const checkUpdate = async () => {
  loading.value = true
  try {
    const result = await GetLatestRelease()
    if (result.hasUpdate) {
      hasUpdate.value = true
      latestVersion.value = result.latestRelease.tag_name
      releaseUrl.value = result.latestRelease.html_url
      releaseDescription.value = result.latestRelease.body || '暂无更新说明'
      return true
    } else {
      ElMessage.success("当前已经是最新版啦(*^▽^*)")
      return false
    }
  } catch (error) {
    console.error('版本检查失败:', error)
    ElMessage.error("无法连接到更新服务器")
    return false
  } finally {
    loading.value = false
  }
}

// 自动检查更新
const autoCheckUpdate = async () => {
  const skipTodayKey = 'EasyTools-SkipUpdateToday'
  const skipDate = localStorage.getItem(skipTodayKey)
  const today = new Date().toDateString()

  try {
    const result = await GetLatestRelease()
    if (result.hasUpdate) {
      hasUpdate.value = true
      latestVersion.value = result.latestRelease.tag_name
      releaseUrl.value = result.latestRelease.html_url
      releaseDescription.value = result.latestRelease.body || '暂无更新说明'

      if (skipDate === today) {
        updateDialogVisible.value = false
        return
      }
      updateDialogVisible.value = true
    }
  } catch (error) {
    console.error('自动版本检查失败:', error)
  }
}

// 处理取消下载
const handleCancelUpdate = async () => {
  if (isDownloading.value) {
    ElMessageBox.confirm('确定要取消下载吗？', '取消下载', {
      confirmButtonText: '确定',
      cancelButtonText: '继续下载',
      type: 'warning',
    }).then(async () => {
      isCancelling.value = true
      downloadStatus.value = '正在取消下载...'

      try {
        const result = await CancelDownload()
        if (result.error) {
          ElMessage.error(result.msg)
        } else {
          ElMessage.info("下载更新已取消")
        }
      } catch (error) {
        console.error('取消下载失败:', error)
        ElMessage.error("取消下载时出现错误")
      } finally {
        resetDownloadState()
      }
    }).catch(() => {
      // 用户选择继续下载，不做任何操作
    })
  } else {
    // 如果不在下载中，直接关闭对话框
    updateDialogVisible.value = false
  }
}

// 重置下载状态
const resetDownloadState = () => {
  isDownloading.value = false
  isCancelling.value = false
  downloadProgress.value = 0
  downloadStatus.value = ''
}

// 处理下载更新
const handleDownloadUpdate = async () => {
  isDownloading.value = true
  downloadProgress.value = 0
  downloadStatus.value = '开始下载...'

  // 强制更新一次，确保状态显示
  await nextTick()

  try {
    const result = await DownloadAndUpdate()

    if (result.error) {
      downloadStatus.value = '下载失败'
      ElMessage.error('下载更新失败: ' + result.msg)
    } else {
      downloadStatus.value = '下载完成'
      ElMessage.success('更新成功: ' + result.msg)

      // 根据系统提示重启
      if (result.msg.includes('重启') || result.msg.includes('重启应用')) {
        ElMessageBox.confirm('更新成功，是否立即重启应用？', '提示', {
          confirmButtonText: '重启',
          cancelButtonText: '稍后',
          type: 'success'
        }).then(async () => {
          await RestartApplication()
        }).catch(() => {
          updateDialogVisible.value = false
          resetDownloadState()
        })
      } else {
        updateDialogVisible.value = false
        resetDownloadState()
      }
    }
  } catch (error) {
    console.error('下载更新失败:', error)
    downloadStatus.value = '下载出错'
    ElMessage.error('下载更新失败: ' + error.message)
    resetDownloadState()
  }
}

const handleCheckUpdate = async () => {
  const hasNewVersion = await checkUpdate()
  if (hasNewVersion) {
    updateDialogVisible.value = true
  }
}

const handleSkipToday = () => {
  const today = new Date().toDateString()
  localStorage.setItem('EasyTools-SkipUpdateToday', today)
  updateDialogVisible.value = false
  ElMessage.info("今日将不再提示更新")
}

const handleDialogClose = (done) => {
  if (isDownloading.value) {
    handleCancelUpdate()
  } else {
    updateDialogVisible.value = false
    done()
  }
}

const handleUrlClick = () => {
  if (releaseUrl.value) {
    BrowserOpenURL(releaseUrl.value)
  }
}

// 下载进度监听器
const handleDownloadProgress = (progress) => {
  // 如果正在取消，不再更新进度
  if (isCancelling.value) {
    return
  }

  downloadProgress.value = Math.round(progress)
  // 同时更新状态文本
  downloadStatus.value = `下载中... ${downloadProgress.value}%`
  console.log('下载进度:', progress, '四舍五入后:', downloadProgress.value)

  // 如果正在下载且对话框未打开，自动打开对话框显示进度
  if (progress > 0 && progress < 100 && !updateDialogVisible.value) {
    updateDialogVisible.value = true
  }
}

// 下载取消事件监听器
const handleDownloadCancelled = () => {
  console.log('收到下载取消事件')
  resetDownloadState()
  ElMessage.info("下载任务已被成功取消")
}


// 打开代理设置
const openProxySettings = async () => {
  // 先跳转
  router.push('/systemManage?tab=proxy-config')
  
  // 跳转后检测代理状态并提示
  await nextTick()
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
        systemEnabled: !!status.systemEnabled,
        enabledModules: Number(status.enabledModules) || 0,
        disabledModules: Number(status.disabledModules) || 0,
        totalModules: Number(status.totalModules) || 0
      })
    } else {
      Object.assign(proxyStatus, {
        globalEnabled: false,
        systemEnabled: false,
        enabledModules: 0,
        disabledModules: 0,
        totalModules: 0
      })
    }
  } catch (error) {
    proxyError.value = true
      Object.assign(proxyStatus, {
        globalEnabled: false,
        systemEnabled: false,
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
        systemEnabled: !!status.systemEnabled,
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

  // 监听代理配置保存事件（包括开关切换）
  // 同时监听 Wails 事件和浏览器 DOM 事件
  EventsOn('proxy-config-saved', () => {
    loadProxyStatus()
  })
  
  // 监听浏览器 DOM 事件（前端触发的事件）
  window.addEventListener('proxy-config-saved', () => {
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

  // 新增：监听端口配置更新事件（如果需要）
  EventsOn('port-config-updated', (config) => {
    if (config) {
      portConfig.value = config
      console.log('端口配置已更新:', config)
    }
  })
}

// 窗口控制方法
const minimize = () => {

  try {
    WindowMinimise()
  } catch (error) {
    console.error('最小化窗口失败:', error)
  }
}
const toggleMaximize = () => {
  try {
    WindowToggleMaximise()
  } catch (error) {
    console.error('切换最大化状态失败:', error)
  }
}

const close = () => {
  try {
    // 统一通过 app-exit 事件走 App.vue 的退出确认与清理逻辑
    EventsEmit('app-exit')
  } catch (error) {
    console.error('关闭应用失败:', error)
  }
}

// 监听主题配置变化
const handleThemeChange = (event) => {
  if (event.detail && event.detail !== null) {
    if (event.detail.fontColor) {
      fontColor.value = event.detail.fontColor
    }
    if (event.detail.fontWeightRatio !== undefined) {
      fontWeightRatio.value = event.detail.fontWeightRatio
    }
    if (event.detail.shadowRatio !== undefined) {
      shadowRatio.value = event.detail.shadowRatio
    }
  } else if (event.detail === null) {
    // 如果配置被清除，恢复默认值
    fontColor.value = '#ffffff'
    fontWeightRatio.value = 100
    shadowRatio.value = 100
  } else {
    // 如果没有配置，加载本地存储的配置
    loadFontColor()
  }
  globalSearchAccelLabel.value = getStoredGlobalSearchHotkey()
}

// 检测平台类型
const detectPlatform = async () => {
  try {
    const env = await GetOs()
    isMac.value = env === 'darwin'
    console.log('检测到平台:', env, 'isMac:', isMac.value)
  } catch (error) {
    console.error('检测平台失败:', error)
    // 默认使用非 macOS 样式
    isMac.value = false
  }
}

// 初始化
onMounted(async () => {
  await nextTick()
  await detectPlatform() // 检测平台
  await loadProxyStatus()
  await loadPortConfig() // 新增：加载端口配置
  setupProxyEvents()
  loadFontColor() // 加载字体颜色配置
  globalSearchAccelLabel.value = getStoredGlobalSearchHotkey()

  // 自动检查更新
  autoCheckUpdate()
  
  // 注册下载进度监听器
  EventsOn('downloadProgress', handleDownloadProgress)
  // 注册下载取消监听器
  EventsOn('downloadCancelled', handleDownloadCancelled)
  
  // 监听主题配置变化事件
  window.addEventListener('theme-background-changed', handleThemeChange)
})

onUnmounted(() => {
  EventsOff('proxy-status-changed')
  EventsOff('global-proxy-updated')
  EventsOff('module-proxy-status-changed')
  EventsOff('proxy-config-saved')
  EventsOff('proxy-error')
  
  // 移除浏览器 DOM 事件监听
  window.removeEventListener('proxy-config-saved', loadProxyStatus)
  EventsOff('window-state-changed')
  EventsOff('log_event')
  EventsOff('log_update')
  EventsOff('title-updated')
  EventsOff('port-config-updated') // 新增：取消监听端口配置更新
  EventsOff('downloadProgress')
  EventsOff('downloadCancelled')
  window.removeEventListener('theme-background-changed', handleThemeChange)
})
</script>

<style scoped>
/* 立体感顶部栏样式 */
.custom-titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 28px;
  background: linear-gradient(135deg, #7f91e3 0%, rgba(78, 159, 237, 0.58) 100%);
  color: var(--titlebar-font-color, #ffffff);
  padding: 0 0 0 12px;
  user-select: none;
  position: relative;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;

  /* 增加立体感的样式 */
  box-shadow: var(--titlebar-shadow, 0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.24)),
    inset 0 1px 0 rgba(255, 255, 255, 0.2),
    inset 0 -1px 0 rgba(0, 0, 0, 0.1);

  border-bottom: 1px solid rgba(255, 255, 255, 0.15);
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

/* macOS 样式时的调整 */
.custom-titlebar.mac-style {
  padding-left: 0;
}

/* 添加顶部栏底部阴影，增加深度感 */
.custom-titlebar::after {
  content: '';
  position: absolute;
  bottom: -4px;
  left: 0;
  right: 0;
  height: 4px;
  background: linear-gradient(to bottom, rgba(0, 0, 0, 0.1), transparent);
  pointer-events: none;
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
  padding: 4px 10px;
  border-radius: 6px;
  transition: all 0.2s;
  position: relative;
  overflow: hidden;
}

.app-brand::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.5), transparent);
}

.app-brand:hover {
  background: rgba(255, 255, 255, 0.15);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.2),
    0 1px 3px rgba(0, 0, 0, 0.1);
}

.app-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.icon-placeholder {
  font-size: 0px;
  filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.2));
}

.app-title {
  font-weight: var(--titlebar-font-weight, 600);
  font-size: 14px;
  letter-spacing: 0.5px;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

.window-title {
  font-size: 13px;
  opacity: 0.9;
  font-weight: calc(var(--titlebar-font-weight, 500) * 0.83);
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  padding: 4px 12px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.05);
  box-shadow:
    inset 0 1px 1px rgba(255, 255, 255, 0.1),
    inset 0 -1px 1px rgba(0, 0, 0, 0.05);
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
  color: var(--titlebar-font-color, #ffffff);
  position: relative;

  /* 按钮立体效果 */
  box-shadow:
    0 1px 1px rgba(255, 255, 255, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
  border: 1px solid transparent;
}

.titlebar-btn:hover {
  background: rgba(255, 255, 255, 0.15);
  box-shadow:
    0 1px 3px rgba(0, 0, 0, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.1);
  transform: translateY(-1px);
}

.control-btn {
  width: 32px;
  height: 28px; /* 与 28px 标题栏对齐，避免上下溢出 */
  margin-left: 2px;
}

.control-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.close-btn:hover {
  background: #ff5f57 !important;
  box-shadow:
    0 1px 3px rgba(0, 0, 0, 0.3),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}

.window-controls {
  display: flex;
  align-items: center;
  margin-left: 8px;
  padding-right: 4px;
}

/* macOS 样式窗口控制按钮 */
.mac-controls {
  margin-left: 0;
  padding-left: 12px;
  padding-right: 0;
  gap: 8px;
}

.mac-controls .control-btn {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  padding: 0;
  margin: 0;
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: none;
  transition: all 0.2s ease;
}

.mac-controls .control-btn :deep(.el-icon) {
  display: none;
}

.mac-close-btn {
  background: #ff5f57;
}

.mac-close-btn:hover {
  background: #ff3b30;
  transform: scale(1.1);
}

.mac-minimize-btn {
  background: #ffbd2e;
}

.mac-minimize-btn:hover {
  background: #ff9500;
  transform: scale(1.1);
}

.mac-maximize-btn {
  background: #28c840;
}

.mac-maximize-btn:hover {
  background: #1dbd2e;
  transform: scale(1.1);
}

/* macOS 样式时左侧区域调整 */
.titlebar-left.mac-left {
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

  /* 立体效果 */
  box-shadow:
    0 1px 3px rgba(0, 0, 0, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
}

/* 无代理状态 - 红色 */
.proxy-indicator.no-proxy {
  background: rgba(244, 67, 54, 0.15);
  border-color: rgba(244, 67, 54, 0.3);
  box-shadow:
    0 1px 3px rgba(244, 67, 54, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
}

/* 程序代理状态 - 绿色 */
.proxy-indicator.program-proxy {
  background: rgba(76, 175, 80, 0.2);
  border-color: rgba(76, 175, 80, 0.3);
  box-shadow:
    0 1px 3px rgba(76, 175, 80, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}

/* 系统代理状态 - 醒目的蓝紫色 */
.proxy-indicator.system-proxy {
  background: rgba(156, 39, 176, 0.25);
  border-color: rgba(156, 39, 176, 0.4);
  box-shadow:
    0 1px 4px rgba(156, 39, 176, 0.3),
    0 0 8px rgba(156, 39, 176, 0.15),
    inset 0 1px 0 rgba(255, 255, 255, 0.25);
  animation: systemProxyPulse 2s ease-in-out infinite;
}

@keyframes systemProxyPulse {
  0%, 100% {
    box-shadow:
      0 1px 4px rgba(156, 39, 176, 0.3),
      0 0 8px rgba(156, 39, 176, 0.15),
      inset 0 1px 0 rgba(255, 255, 255, 0.25);
  }
  50% {
    box-shadow:
      0 1px 6px rgba(156, 39, 176, 0.4),
      0 0 12px rgba(156, 39, 176, 0.25),
      inset 0 1px 0 rgba(255, 255, 255, 0.3);
  }
}

.proxy-indicator.testing {
  animation: pulse 1.5s infinite;
  background: rgba(255, 193, 7, 0.2);
  box-shadow:
    0 1px 3px rgba(255, 193, 7, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}

.proxy-indicator.error {
  background: rgba(244, 67, 54, 0.2);
  border-color: rgba(244, 67, 54, 0.3);
  box-shadow:
    0 1px 3px rgba(244, 67, 54, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}

@keyframes pulse {
  0% {
    opacity: 1;
    box-shadow:
      0 1px 3px rgba(255, 193, 7, 0.2),
      inset 0 1px 0 rgba(255, 255, 255, 0.2);
  }

  50% {
    opacity: 0.7;
    box-shadow:
      0 1px 6px rgba(255, 193, 7, 0.4),
      inset 0 1px 0 rgba(255, 255, 255, 0.3);
  }

  100% {
    opacity: 1;
    box-shadow:
      0 1px 3px rgba(255, 193, 7, 0.2),
      inset 0 1px 0 rgba(255, 255, 255, 0.2);
  }
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

  /* 立体徽章效果 */
  box-shadow:
    0 1px 3px rgba(0, 0, 0, 0.3),
    0 0 0 1px rgba(255, 255, 255, 0.2);
}

/* 日志按钮徽标 */
.log-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 16px;
  padding: 0 4px;
  background: #f97316;
  color: white;
  border-radius: 999px;
  font-size: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
  z-index: 2;
  font-weight: bold;
  box-shadow:
    0 1px 3px rgba(0, 0, 0, 0.3),
    0 0 0 1px rgba(255, 255, 255, 0.2);
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

  /* 立体点效果 */
  box-shadow:
    0 0 2px rgba(0, 0, 0, 0.5),
    inset 0 1px 1px rgba(255, 255, 255, 0.3);
}

/* 无代理状态点 - 红色 */
.status-dot.no-proxy {
  background: #f44336;
  box-shadow:
    0 0 3px rgba(244, 67, 54, 0.6),
    inset 0 1px 1px rgba(255, 255, 255, 0.3);
}

/* 程序代理状态点 - 绿色 */
.status-dot.program-enabled {
  background: #4caf50;
  box-shadow:
    0 0 4px #4caf50,
    inset 0 1px 1px rgba(255, 255, 255, 0.5);
}

/* 系统代理状态点 - 醒目的蓝紫色，带脉冲效果 */
.status-dot.system-enabled {
  background: #9c27b0;
  box-shadow:
    0 0 6px rgba(156, 39, 176, 0.8),
    0 0 12px rgba(156, 39, 176, 0.4),
    inset 0 1px 1px rgba(255, 255, 255, 0.6);
  animation: systemProxyDotPulse 2s ease-in-out infinite;
}

@keyframes systemProxyDotPulse {
  0%, 100% {
    box-shadow:
      0 0 6px rgba(156, 39, 176, 0.8),
      0 0 12px rgba(156, 39, 176, 0.4),
      inset 0 1px 1px rgba(255, 255, 255, 0.6);
  }
  50% {
    box-shadow:
      0 0 8px rgba(156, 39, 176, 1),
      0 0 16px rgba(156, 39, 176, 0.6),
      inset 0 1px 1px rgba(255, 255, 255, 0.7);
  }
}

.status-dot.loading {
  background: #ffc107;
  animation: pulse 1s infinite;
  box-shadow:
    0 0 4px #ffc107,
    inset 0 1px 1px rgba(255, 255, 255, 0.5);
}

.status-dot.error {
  background: #f44336;
  box-shadow:
    0 0 4px #f44336,
    inset 0 1px 1px rgba(255, 255, 255, 0.5);
}

/* 日志抽屉样式 */
.log-drawer {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.log-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.log-toolbar-right>.el-button+.el-button {
  margin-left: 4px;
}

.log-list {
  flex: 1;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
  background: #0f172a;
  color: #e5e7eb;
  padding: 8px 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 12px;
  overflow-y: auto;
}

.log-item {
  display: flex;
  align-items: baseline;
  gap: 6px;
  padding: 2px 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-time {
  color: #9ca3af;
  flex-shrink: 0;
}

.log-source {
  color: #a5b4fc;
  flex-shrink: 0;
}

.log-message {
  flex: 1;
}

.log-empty {
  text-align: center;
  padding: 24px 8px;
  color: #9ca3af;
}

.log-level-info .log-message {
  color: #e5e7eb;
}

.log-level-debug .log-message {
  color: #60a5fa;
}

.log-level-warning .log-message {
  color: #fbbf24;
}

.log-level-error .log-message {
  color: #f97373;
}

:deep(.el-drawer) {
  padding: 8px;
}

:deep(.el-drawer__body) {
  padding: 8px;
  overflow: hidden; /* 由内部 .log-list 负责滚动，避免双滚动条 */
}

/* 版本更新对话框样式 */
.glass-update-dialog .el-dialog {
  background: rgba(255, 255, 255, 0.98) !important;
  backdrop-filter: blur(20px) !important;
  -webkit-backdrop-filter: blur(20px) !important;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15) !important;
  border-radius: 20px !important;
  overflow: hidden;
  border: none !important;
}

.glass-update-dialog .el-dialog__header {
  display: none;
}

.glass-update-dialog .el-dialog__body {
  padding: 0 !important;
  background: transparent !important;
}

.glass-update-dialog .el-dialog__footer {
  padding: 0 !important;
  background: transparent;
}

.glass-update-dialog .el-overlay {
  backdrop-filter: blur(8px) !important;
  -webkit-backdrop-filter: blur(8px) !important;
  background: rgba(0, 0, 0, 0.3) !important;
}

.glass-dialog-content {
  background: transparent;
}

.dialog-header {
  display: flex;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
  position: relative;
  overflow: hidden;
}

.update-header {
  background: linear-gradient(135deg, #90d4d3 0%, #cec2d9 100%);
}

.update-icon-wrapper {
  position: relative;
  margin-right: 12px;
}

.update-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 2;
  transition: all 0.2s ease;
}

.update-icon .el-icon {
  color: white;
  font-size: 20px;
}

.header-text {
  flex: 1;
  z-index: 2;
  position: relative;
}

.dialog-title {
  margin: 0;
  color: white;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.3px;
}

.update-content {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px 20px;
  max-height: 50vh;
  overflow-y: auto;
  background: transparent;
}

.update-content::-webkit-scrollbar {
  width: 6px;
}

.update-content::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.03);
  border-radius: 3px;
}

.update-content::-webkit-scrollbar-thumb {
  background: rgba(102, 126, 234, 0.4);
  border-radius: 3px;
}

.update-content::-webkit-scrollbar-thumb:hover {
  background: rgba(102, 126, 234, 0.6);
}

.version-section,
.description-section,
.url-section {
  display: flex;
  align-items: flex-start;
}

.section-label {
  min-width: 75px;
  font-weight: 500;
  color: #666;
  margin-right: 10px;
  font-size: 13px;
  flex-shrink: 0;
}

.version-tag {
  background: linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%);
  color: white;
  border: none;
  font-weight: 600;
  padding: 4px 12px;
  font-size: 12px;
}

.description-section {
  flex-direction: column;
}

.description-text {
  margin-left: 0;
  margin-top: 8px;
  width: 100%;
  max-height: 200px;
  overflow: hidden;
  border-radius: 0;
  background: transparent;
  border: none;
  padding: 0;
}

.description-text :deep(.el-scrollbar__wrap) {
  overflow-x: hidden;
  padding-right: 8px;
}

.description-text :deep(.el-scrollbar__bar) {
  opacity: 0.6;
}

.description-text :deep(.el-scrollbar__thumb) {
  background-color: rgba(102, 126, 234, 0.4);
  border-radius: 4px;
}

.description-text :deep(.el-scrollbar__thumb:hover) {
  background-color: rgba(102, 126, 234, 0.6);
}

.description-line {
  white-space: pre-wrap;
  word-wrap: break-word;
  margin: 4px 0;
  line-height: 1.5;
  color: #555;
  padding: 6px 0 6px 10px;
  border-left: 2px solid #667eea;
  font-size: 13px;
}

.url-section {
  flex-direction: column;
}

.url-text {
  margin-left: 0;
  margin-top: 8px;
  width: 100%;
  word-break: break-all;
}

.url-tag {
  word-break: break-all;
  cursor: pointer;
  transition: all 0.2s ease;


  max-width: 100%;
  display: inline-block;
  padding: 4px 10px;

  font-size: 12px;
  background: rgba(102, 126, 234, 0.05);
}

.url-tag:hover {
  background: #88e9d1;
  color: white;
}

.no-url {
  color: #999;
  font-style: italic;
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

.skip-btn {
  border: 1px solid #d9d9d9;
  color: #666;
  background: #fff;
}

.skip-btn:hover {
  border-color: #4096ff;
  color: #4096ff;
}

.cancel-btn {
  border: 1px solid #d9d9d9;
  color: #666;
  background: #fff;
}

.cancel-btn:hover:not(:disabled) {
  border-color: #ff4d4f;
  color: #ff4d4f;
}

.cancel-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.download-btn {
  border: none;
  color: white;
  box-shadow: 0 2px 8px rgba(255, 107, 107, 0.3);
}

.download-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, #8aec90 0%, #bbeb5a 100%);
  box-shadow: 0 3px 12px rgba(255, 107, 107, 0.4);
}

.download-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.progress-section {
  display: flex;
  flex-direction: column;
  padding: 0;
  background: transparent;
  border-radius: 0;
  border: none;
}

.progress-content {
  margin-left: 0;
  margin-top: 8px;
}

.progress-text {
  margin-top: 6px;
  font-size: 12px;
  color: #667eea;
  text-align: center;
  font-weight: 500;
}

.progress-section :deep(.el-progress-bar__outer) {
  background: rgba(102, 126, 234, 0.1);
  border-radius: 10px;
}

.progress-section :deep(.el-progress-bar__inner) {
  background: linear-gradient(90deg, #667eea, #764ba2);
  border-radius: 10px;
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