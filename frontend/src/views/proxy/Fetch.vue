<template>
  <!-- 原有代码保持不变 -->
  <a-row :gutter="12">
    <a-col :span="18">
      <a-alert type='info'>如欲使用Hunter等进行搜索，请先配置好Key哦</a-alert>
    </a-col>
    <a-col :span="6">
      <a-button-group>
        <a-button type="outline" size="large" :disabled="loading" @click="getProxies">
          {{ loading ? '获取中...' : '获取' }}
        </a-button>
        <a-button type="outline" status="success" size="large" :disabled="loading || datasets.length === 0" @click="useFetchedDatasets">使用</a-button>
        <a-button type="outline" status="danger" size="large" :disabled="!loading" @click="cancelFetch">取消</a-button>
      </a-button-group>
    </a-col>
  </a-row>
  <br/>

  <!-- 日志显示区域 -->
  <div v-if="loading" class="log-container">
    <a-alert type="info" :closable="false">
      <template #icon>
        <a-spin :size="16" />
      </template>
      <div class="log-content">
        <div class="log-header">
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
        <div class="log-tips">
          <a-typography-text type="secondary">
            如长时间无响应，可点击"取消"按钮中断操作
          </a-typography-text>
        </div>
      </div>
    </a-alert>
  </div>

  <a-table
      :columns="columns"
      :loading="loading"
      :data="displayedData"
      :pagination="paginationConfig"
      :scroll="{ y: '615px' }"
      @page-change="handlePageChange"
      @page-size-change="handlePageSizeChange"
  />
</template>

<script setup lang="ts">
import {ref, onMounted, onUnmounted, nextTick, computed, reactive} from "vue";
import {FetchProxies, UseFetchedDatasets} from "../../../wailsjs/go/proxy/Proxy";
import {Notification} from '@arco-design/web-vue';
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
const timeoutRef = ref<NodeJS.Timeout | null>(null)

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

// 分页配置 - 使用响应式
const paginationConfig = computed(() => ({
  current: paginationState.current,
  pageSize: paginationState.pageSize,
  total: datasets.value.length,
  showPageSize: true,
  showJumper: true,
  showTotal: true,
  pageSizeOptions: [10, 20, 50, 100],
  pageSizeText: '每页条数',
  totalText: '总计',
  jumpText: '跳至'
}))

const columns = [
  {
    title: '序号',
    dataIndex: 'key',
    width: 80,
  },
  {
    title: '类型',
    dataIndex: 'kind',
    width: 120,
  },
  {
    title: '来源',
    dataIndex: 'source',
    width: 120,
  },
  {
    title: 'IP',
    dataIndex: 'address',
    width: 150,
    ellipsis: true,
    tooltip: true,
  },
]

// 分页事件处理
const handlePageChange = (page: number) => {
  paginationState.current = page
}

const handlePageSizeChange = (size: number) => {
  paginationState.pageSize = size
  paginationState.current = 1
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

// 监听后端进度事件
const setupProgressListener = () => {
  console.log('🔄 设置事件监听器...')

  EventsOn('fetch_start', (data: any) => {
    console.log('✅ 收到开始事件:', data)
    eventStatus.value = '收到开始事件'

    const message = typeof data === 'string' ? data : '开始获取代理数据...'
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

      // 处理代理数据 - 添加更详细的调试
      if (progress.proxies && Array.isArray(progress.proxies)) {
        console.log(`📦 处理代理数据: ${progress.proxies.length} 个代理`)

        const startIndex = datasets.value.length + 1
        const newProxies: ProxyInfo[] = progress.proxies.map((proxy: string, index: number) => ({
          key: (startIndex + index).toString(),
          source: progress.source || '未知来源',
          kind: 'socks5',
          address: proxy,
        }))

        console.log(`🆕 添加 ${newProxies.length} 个新代理`)

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
    const message = typeof data === 'string' ? data : '数据获取完成'

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
      Notification.success({
        title: '获取完成',
        content: message,
        duration: 2000,
      })
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

    Notification.error({
      title: '获取失败',
      content: errorMessage,
      duration: 3000,
    })
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

    Notification.info({
      title: '已取消',
      content: '数据获取操作已被取消',
      duration: 2000,
    })
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

function getProxies() {
  console.log('🚀 开始获取代理数据')

  // 先清理之前的资源
  cleanup()

  loading.value = true
  logs.value = []
  addLog('开始获取代理数据...', 'info')
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
        Notification.success({
          title: '部分数据获取完成',
          content: `由于部分API响应超时，已获取 ${datasets.value.length} 条代理数据`,
          duration: 3000,
        })
      } else {
        // 完全没有数据
        addLog('请求超时，未获取到任何数据', 'warning')
        Notification.warning({
          title: '请求超时',
          content: '获取代理数据时间过长，可能是网络问题或API服务不稳定，请稍后重试',
          duration: 3000,
          closable: true,
        })
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
      Notification.error({
        title: '错误',
        content: res.Message,
        duration: 3000,
        closable: true,
      })
      loading.value = false
      return
    }

    // 处理最终数据
    try {
      if (res.Data) {
        const finalData = JSON.parse(res.Data) as ProxyInfo[]
        console.log('🎯 最终数据长度:', finalData.length)

        if (finalData.length > 0) {
          datasets.value = finalData
          paginationState.total = finalData.length

          loading.value = false
          addLog(`已获取 ${finalData.length} 条代理数据`, 'success')

          Notification.success({
            title: '获取完成',
            content: `共获取 ${finalData.length} 条代理数据`,
            duration: 2000,
          })
        } else {
          loading.value = false
          addLog('未获取到任何代理数据', 'warning')
          Notification.warning({
            title: '无数据',
            content: '未获取到任何代理数据',
            duration: 2000,
          })
        }
      }
    } catch (error) {
      console.error('解析最终数据失败:', error)
      addLog('解析返回数据时发生错误', 'error')
      loading.value = false
      Notification.error({
        title: '数据处理失败',
        content: '解析返回数据时发生错误',
        duration: 2000,
      })
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
    Notification.error({
      title: '请求失败',
      content: error.message,
      duration: 3000,
      closable: true,
    })
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

function useFetchedDatasets() {
  if (datasets.value.length === 0) {
    Notification.warning({
      title: '无数据',
      content: '请先获取代理数据',
      duration: 2000,
    })
    return
  }

  // 立即触发跳转到运行模块
  emit('switchTab', '1')

  Notification.info({
    title: '任务开始',
    content: '正在跳转到运行页面...',
    duration: 1500,
    closable: true,
  });

  UseFetchedDatasets().then(res => {
    if (res.Code !== 200) {
      Notification.error({
        title: '错误',
        content: res.Message,
        duration: 2000,
        closable: true,
      });
      configStore.setStatus(3)
      return;
    }

    Notification.success({
      title: '任务完成',
      content: res.Message,
      duration: 2000,
    });
  }).catch(error => {
    Notification.error({
      title: '请求失败',
      content: error.message,
      duration: 2000,
    })
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

/* 添加表格行进入动画 */
:deep(.arco-table-tr) {
  transition: all 0.3s ease;
}

:deep(.arco-table-tr) {
  animation: fadeIn 0.5s ease-in;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.debug-info {
  margin-bottom: 16px;
}

.debug-info :deep(.arco-alert-content) {
  font-size: 12px;
}

/* 表格样式优化 */
:deep(.arco-table-pagination) {
  margin-top: 16px;
}

:deep(.arco-table-cell) {
  word-break: break-all;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .log-content {
    font-size: 12px;
  }

  :deep(.arco-table) {
    font-size: 12px;
  }
}

/* 滚动条样式 */
.log-messages::-webkit-scrollbar {
  width: 4px;
}

.log-messages::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 2px;
}

.log-messages::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 2px;
}

.log-messages::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
</style>