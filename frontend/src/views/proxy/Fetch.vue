<template>
  <el-row :gutter="12">
    <el-col :span="18">
      <el-alert type="info" :closable="false" show-icon>
        便携代理模块已直接复用空间测绘的搜索能力，API Key 请在配置页统一管理。
      </el-alert>
    </el-col>
    <el-col :span="6">
      <div class="fetch-actions">
        <el-button :disabled="loading" @click="getProxies">{{ loading ? '获取中...' : '获取' }}</el-button>
        <el-button type="success" plain :disabled="loading || datasets.length === 0" @click="useFetchedDatasets">使用</el-button>
        <el-button type="danger" plain :disabled="!loading" @click="cancelFetch">取消</el-button>
      </div>
    </el-col>
  </el-row>
  <br/>

  <div v-if="loading" class="log-container">
    <el-alert type="info" :closable="false">
      <template #default>
        <div class="log-content">
          <div class="log-header">
            <el-icon class="is-loading fetch-spin"><Loading /></el-icon>
            <span class="log-title">获取进度</span>
            <span class="log-count" v-if="datasets.length > 0">已获取 {{ datasets.length }} 条数据</span>
          </div>
          <div class="log-messages">
            <div
              v-for="(log, index) in displayedLogs"
              :key="index"
              class="log-item"
              :class="getLogClass(log)"
            >
              <span class="log-time">{{ log.time }}</span>
              <span class="log-text">{{ log.message }}</span>
            </div>
          </div>
          <p class="log-tips et-muted-tip">如长时间无响应，可点击「取消」按钮中断操作</p>
        </div>
      </template>
    </el-alert>
  </div>

  <div class="table-wrapper">
    <div class="proxy-table-wrap glass-card">
      <el-table
        :data="displayedData"
        v-loading="loading"
        stripe
        border
        class="proxy-table-inner"
        style="width: 100%"
        max-height="520"
      >
        <el-table-column prop="key" label="序号" width="150" />
        <el-table-column prop="kind" label="类型" width="220" />
        <el-table-column prop="source" label="来源" width="220" />
        <el-table-column prop="address" label="IP" min-width="200" show-overflow-tooltip />
      </el-table>
      <div class="proxy-table-pagination">
        <el-pagination
          v-model:current-page="paginationState.current"
          v-model:page-size="paginationState.pageSize"
          :total="datasets.length"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          background
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {ref, onMounted, onUnmounted, nextTick, computed, reactive} from "vue";
import {FetchProxies, UseFetchedDatasets} from "../../../wailsjs/go/proxy/Proxy";
import { ElMessage } from 'element-plus';
import { Loading } from '@element-plus/icons-vue';
import {useConfigStore} from "./store/types";
import { EventsOn, EventsOff } from '../../../wailsjs/runtime'

// 定义props和emit
interface Props {
  activeTab?: string;
}

interface Emits {
  (e: 'update:activeTab', tabKey: string): void;
  (e: 'switchTab', tabKey: string): void;
}

const props = withDefaults(defineProps<Props>(), {
  activeTab: '2'
})

const emit = defineEmits<Emits>()

// 原有的变量定义保持不变
interface ProxyInfo {
  key: string;
  source: string;
  kind: string;
  address: string;
}

interface LogEntry {
  time: string;
  message: string;
  type?: 'info' | 'success' | 'warning' | 'error';
}

const datasets = ref<ProxyInfo[]>([])
const configStore = useConfigStore()
const loading = ref(false)
const logs = ref<LogEntry[]>([])
const currentSource = ref('')
const eventStatus = ref('等待事件...')
const fallbackProgress = ref(false)
const cancelToken = ref<any>(null)
const timeoutRef = ref<ReturnType<typeof setTimeout> | null>(null)

// 分页状态管理
const paginationState = reactive({
  current: 1,
  pageSize: 10,
  total: 0
})

// 计算显示的表格数据
const displayedData = computed(() => {
  const start = (paginationState.current - 1) * paginationState.pageSize
  const end = start + paginationState.pageSize
  return datasets.value.slice(start, end)
})

// 计算显示的日志（只显示最近10条）
const displayedLogs = computed(() => {
  return logs.value.slice(-10)
})

const ensureAuthorized = async (): Promise<boolean> => {
  return true;
}

// 添加日志
const addLog = (message: string, type: 'info' | 'success' | 'warning' | 'error' = 'info') => {
  const now = new Date()
  const time = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`

  logs.value.push({
    time,
    message,
    type
  })

  // 自动滚动到底部
  nextTick(() => {
    const logContainer = document.querySelector('.log-messages')
    if (logContainer) {
      logContainer.scrollTop = logContainer.scrollHeight
    }
  })
}

// 获取日志样式类
const getLogClass = (log: LogEntry) => {
  return {
    'log-item-info': log.type === 'info',
    'log-item-success': log.type === 'success',
    'log-item-warning': log.type === 'warning',
    'log-item-error': log.type === 'error'
  }
}

// 获取当前xui模式（1或2表示开启）
const getCurrentXuiMode = () => {
  const xuiMode = configStore.getXui();
  return xuiMode === '1' || xuiMode === '2';
}

// 监听后端进度事件
const setupProgressListener = () => {
  console.log('🔄 设置事件监听器...')

  EventsOn('fetch_start', (data: any) => {
    console.log('✅ 收到开始事件:', data)
    eventStatus.value = '收到开始事件'

    const xuiMode = getCurrentXuiMode()
    const message = typeof data === 'string' ? data :
        (xuiMode ? '开始扫描X-UI面板...' : '开始获取代理数据...')
    addLog(message, 'info')

    currentSource.value = ''
    datasets.value = []
    // 重置分页状态
    paginationState.current = 1
    paginationState.total = 0
    loading.value = true
  })

  EventsOn('fetch_progress', (data: any) => {
    try {
      // 检查是否已取消
      if (cancelToken.value?.cancelled) {
        console.log('已取消，忽略进度更新')
        return
      }

      console.log('📊 收到进度事件:', data)
      eventStatus.value = '收到进度事件'

      let progress: any
      if (typeof data === 'string') {
        try {
          progress = JSON.parse(data)
        } catch {
          // 如果不是JSON，直接作为消息处理
          progress = { message: data }
        }
      } else {
        progress = data
      }

      // 直接显示进度消息
      if (progress.message) {
        addLog(progress.message, 'info')
      }

      // 处理代理数据
      if (progress.proxies && Array.isArray(progress.proxies)) {
        console.log(`📦 处理代理数据: ${progress.proxies.length} 个代理`)
        const startIndex = datasets.value.length + 1
        const xuiMode = getCurrentXuiMode()

        const newProxies: ProxyInfo[] = progress.proxies.map((proxy: string, index: number) => ({
          key: (startIndex + index).toString(),
          source: progress.source || '未知来源',
          kind: xuiMode ? 'xui' : 'socks5',
          address: proxy,
        }))

        console.log(`🆕 添加 ${newProxies.length} 个新代理，类型: ${xuiMode ? 'xui' : 'socks5'}`)

        // 使用响应式更新
        datasets.value = [...datasets.value, ...newProxies]

        // 更新分页总数
        paginationState.total = datasets.value.length

        // 强制更新显示
        nextTick(() => {
          console.log(`📊 数据集更新完成: ${datasets.value.length} 条记录`)
        })
      }
    } catch (error) {
      console.error('❌ 处理进度数据失败:', error)
      addLog('处理进度数据时发生错误: ' + error, 'error')
    }
  })

  EventsOn('fetch_complete', (data: any) => {
    // 检查是否已取消
    if (cancelToken.value?.cancelled) {
      console.log('已取消，忽略完成事件')
      return
    }

    console.log('🎉 收到完成事件:', data)
    eventStatus.value = '收到完成事件'

    const xuiMode = getCurrentXuiMode()
    const message = typeof data === 'string' ? data :
        (xuiMode ? 'X-UI面板扫描完成' : '数据获取完成')

    addLog(message, 'success')
    // 更新分页总数
    paginationState.total = datasets.value.length

    nextTick(() => {
      loading.value = false
      // 清除超时定时器
      if (timeoutRef.value) {
        clearTimeout(timeoutRef.value)
        timeoutRef.value = null
      }

      // 根据模式显示不同的成功消息
      if (xuiMode) {
        ElMessage.success('X-UI面板扫描完成，请查看Excel文件获取节点信息')
      } else {
        ElMessage.success(message)
      }
    })
  })

  EventsOn('fetch_error', (data: any) => {
    // 检查是否已取消
    if (cancelToken.value?.cancelled) {
      console.log('已取消，忽略错误事件')
      return
    }

    console.error('❌ 收到错误事件:', data)
    const errorMessage = typeof data === 'string' ? data : '数据获取过程中发生错误'

    addLog(errorMessage, 'error')
    loading.value = false

    // 清除超时定时器
    if (timeoutRef.value) {
      clearTimeout(timeoutRef.value)
      timeoutRef.value = null
    }

    ElMessage.error(errorMessage)
  })

  console.log('✅ 事件监听器设置完成')
}

// 取消获取操作
const cancelFetch = () => {
  if (loading.value) {
    console.log('用户取消获取操作')
    cancelToken.value = { cancelled: true }
    loading.value = false
    addLog('操作已取消', 'warning')

    // 清除超时定时器
    if (timeoutRef.value) {
      clearTimeout(timeoutRef.value)
      timeoutRef.value = null
    }

    ElMessage.info('数据获取操作已被取消')
  }
}

// 清理资源
const cleanup = () => {
  if (timeoutRef.value) {
    clearTimeout(timeoutRef.value)
    timeoutRef.value = null
  }
  cancelToken.value = { cancelled: true }
}

async function getProxies() {
  console.log('🚀 开始获取代理数据')

  // 启动前做一次授权检查（只有 X-UI 模式才需要校验）
  const xuiMode = getCurrentXuiMode();
  if (xuiMode) {
    const ok = await ensureAuthorized();
    if (!ok) {
      return;
    }
  }

  // 先清理之前的资源
  cleanup()

  loading.value = true
  logs.value = []

  // 获取当前xui模式用于显示不同的提示
  const currentXuiMode = getCurrentXuiMode()
  const startMessage = currentXuiMode ? '开始扫描X-UI面板...' : '开始获取代理数据...'

  addLog(startMessage, 'info')
  currentSource.value = ''
  datasets.value = []
  eventStatus.value = '等待事件...'
  fallbackProgress.value = false
  paginationState.current = 1
  paginationState.total = 0

  // 设置取消标记
  cancelToken.value = { cancelled: false }

  // 设置超时处理
  timeoutRef.value = setTimeout(() => {
    if (loading.value && !cancelToken.value?.cancelled) {
      console.log('⏰ 请求超时，强制结束加载状态')
      loading.value = false
      addLog('请求超时，正在处理已获取的数据...', 'warning')

      // 如果有部分数据，显示成功通知
      if (datasets.value.length > 0) {
        paginationState.total = datasets.value.length
        addLog(`已获取 ${datasets.value.length} 条代理数据`, 'success')

        if (currentXuiMode) {
          ElMessage.success(`由于部分面板响应超时，已扫描 ${datasets.value.length} 个X-UI面板`)
        } else {
          ElMessage.success(`由于部分API响应超时，已获取 ${datasets.value.length} 条代理数据`)
        }
      } else {
        // 完全没有数据
        addLog('请求超时，未获取到任何数据', 'warning')
        if (currentXuiMode) {
          ElMessage.warning('扫描X-UI面板时间过长，可能是网络问题或面板服务不稳定，请稍后重试')
        } else {
          ElMessage.warning('获取代理数据时间过长，可能是网络问题或API服务不稳定，请稍后重试')
        }
      }
    }
  }, 60000)

  console.log('📞 调用 FetchProxies...')
  FetchProxies().then(res => {
    // 检查是否已取消
    if (cancelToken.value?.cancelled) {
      console.log('请求已被取消，忽略响应')
      return
    }

    // 清除超时定时器
    if (timeoutRef.value) {
      clearTimeout(timeoutRef.value)
      timeoutRef.value = null
    }

    console.log('📨 FetchProxies响应:', res)

    if (res.Code !== 200) {
      addLog(`错误: ${res.Message}`, 'error')
      ElMessage.error(res.Message)
      loading.value = false
      return
    }

    // 处理最终数据
    try {
      if (res.Data) {
        let parsed: any
        try {
          parsed = JSON.parse(res.Data)
        } catch (e) {
          console.error('解析最终数据 JSON 失败:', e, '原始数据:', res.Data)
          addLog('解析返回数据 JSON 时发生错误', 'error')
          loading.value = false
          ElMessage.error('解析返回数据 JSON 时发生错误')
          return
        }

        // 后端可能返回 null / 对象等非数组结构，这里统一做兼容
        let finalData: ProxyInfo[] = []
        if (Array.isArray(parsed)) {
          finalData = parsed
        } else if (parsed && Array.isArray(parsed?.proxies)) {
          // 兼容形如 { proxies: [...] } 的结构
          finalData = parsed.proxies
        } else {
          console.warn('最终数据不是数组，按无数据处理。parsed =', parsed)
        }

        console.log('🎯 最终数据长度:', finalData.length)

        if (finalData.length > 0) {
          datasets.value = finalData
          paginationState.total = finalData.length

          loading.value = false

          const currentXuiMode = getCurrentXuiMode()
          const successMessage = currentXuiMode ?
              `已扫描 ${finalData.length} 个X-UI面板` :
              `已获取 ${finalData.length} 条代理数据`

          addLog(successMessage, 'success')

          if (currentXuiMode) {
            ElMessage.success(`共扫描 ${finalData.length} 个X-UI面板，节点信息已保存到Excel`)
          } else {
            ElMessage.success(`共获取 ${finalData.length} 条代理数据`)
          }
        } else {
          loading.value = false
          const currentXuiMode = getCurrentXuiMode()
          const noDataMessage = currentXuiMode ? '未扫描到任何可用的X-UI面板' : '未获取到任何代理数据'

          addLog(noDataMessage, 'warning')
          ElMessage.warning(noDataMessage)
        }
      }
    } catch (error) {
      console.error('解析最终数据失败:', error)
      addLog('解析返回数据时发生错误', 'error')
      loading.value = false
      ElMessage.error('解析返回数据时发生错误')
    }

    configStore.setStatus(1)
    console.log('✅ 代理数据获取流程完成')

  }).catch(error => {
    // 检查是否已取消
    if (cancelToken.value?.cancelled) {
      console.log('请求已被取消，忽略错误')
      return
    }

    // 清除超时定时器
    if (timeoutRef.value) {
      clearTimeout(timeoutRef.value)
      timeoutRef.value = null
    }

    console.error('❌ FetchProxies请求失败:', error)
    loading.value = false
    addLog('获取代理数据失败: ' + error.message, 'error')
    ElMessage.error(error.message)
  })
}

// 移除事件监听
const removeProgressListener = () => {
  EventsOff('fetch_start')
  EventsOff('fetch_progress')
  EventsOff('fetch_complete')
  EventsOff('fetch_error')
}

onMounted(() => {
  console.log('🔧 Fetch 组件已挂载')
  setupProgressListener()
})

onUnmounted(() => {
  console.log('🗑️ Fetch 组件已卸载')
  removeProgressListener()
  cleanup()
})

async function useFetchedDatasets() {
  if (datasets.value.length === 0) {
    ElMessage.error('请先获取代理数据信息')
    return
  }

  // 使用数据前也做一次授权确认（只有 X-UI 模式才需要校验）
  const currentXuiMode = getCurrentXuiMode();
  if (currentXuiMode) {
    const ok = await ensureAuthorized();
    if (!ok) {
      return;
    }
  }

  // 立即触发跳转到运行模块
  emit('switchTab', '1')

  // 根据xui模式显示不同的消息（如果上面已经获取过，这里可以复用）
  const startMessage = currentXuiMode ?
      'X-UI面板信息已保存到Excel，正在跳转到运行页面...' :
      '正在跳转到运行页面...'

  ElMessage.info(startMessage);

  UseFetchedDatasets().then(res => {
    if (res.Code !== 200) {
      ElMessage.error(res.Message);
      configStore.setStatus(3)
      return;
    }

    // 根据xui模式显示不同的成功消息
    if (currentXuiMode) {
      ElMessage.success('X-UI面板信息已保存到Excel文件，请手动查看使用');
    } else {
      ElMessage.success(res.Message);
    }
  }).catch(error => {
    ElMessage.error(error.message)
  })
}
</script>

<style scoped>
/* 原有的样式保持不变 */
.log-container {
  margin-bottom: 16px;
}

.log-content {
  width: 100%;
}

.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.log-title {
  font-weight: 500;
  font-size: 14px;
}

.log-count {
  font-size: 12px;
  color: #1890ff;
  background: #f0f8ff;
  padding: 2px 8px;
  border-radius: 4px;
}

.log-messages {
  max-height: 120px;
  overflow-y: auto;
  border: 1px solid #f0f0f0;
  border-radius: 4px;
  padding: 8px;
  background: #fafafa;
  margin-bottom: 8px;
}

.log-item {
  display: flex;
  align-items: flex-start;
  margin-bottom: 4px;
  font-size: 12px;
  line-height: 1.4;
}

.log-item:last-child {
  margin-bottom: 0;
}

.log-time {
  color: #666;
  min-width: 50px;
  margin-right: 8px;
  font-family: monospace;
}

.log-text {
  flex: 1;
  word-break: break-all;
}

/* 日志类型样式 */
.log-item-info .log-text {
  color: #1890ff;
}

.log-item-success .log-text {
  color: #52c41a;
}

.log-item-warning .log-text {
  color: #faad14;
}

.log-item-error .log-text {
  color: #ff4d4f;
}

.log-tips {
  margin-top: 8px;
  font-size: 12px;
}

.empty-state {
  margin: 40px 0;
  text-align: center;
}

.fetch-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
}

.fetch-spin {
  margin-right: 6px;
  vertical-align: middle;
}

.et-muted-tip {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.table-wrapper {
  position: relative;
  width: 100%;
}

.proxy-table-wrap.glass-card {
  background-color: #faf9f6;
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  padding: 16px;
}

.proxy-table-inner :deep(.el-table__body-wrapper) {
  max-height: calc(100vh - 520px);
}

.proxy-table-pagination {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}

@media (max-width: 768px) {
  .log-content {
    font-size: 12px;
  }
  .proxy-table-inner {
    font-size: 12px;
  }
}
</style>