<template>
  <el-row :gutter="12">
    <el-col :span="18">
      <el-input
          readonly
          :model-value="configState.getFilePath()"
          placeholder="请选择包含代理URL的文件（可拖拽文件到此处）"
          class="glass-card"
      />
    </el-col>
    <el-col :span="6">
      <el-button
          :disabled="disabled || configState.getStatus() === 2"
          v-show="configState.getStatus() !== 2"
          @click="handleImportOrStart"
          class="import-btn"
          style="width: 100%; margin-bottom: 8px"
      >{{ hasDroppedFile ? '开始处理' : '导入' }}
      </el-button>
      <el-button
          v-show="configState.getStatus() === 2"
          style="width: 100%"
          @click="stopTask"
      >停止
      </el-button>
    </el-col>
  </el-row>

  <br>

  <el-row :gutter="12">
    <el-col :span="5">
      <el-card shadow="hover" class="card progress">
        <el-progress
            type="circle"
            :percentage="Math.min(100, Math.max(0, percent))"
            :status="progressStatus"
            :width="120"
        />
        <el-tag class="RunGetStatus" :type="tagType" style="margin-top: 10px">
          <span v-if="configState.getStatus() === 0">尚未测试</span>
          <span v-else-if="configState.getStatus() === 1">正在测试</span>
          <span v-else-if="configState.getStatus() === 2">测试完成</span>
          <span v-else>任务取消</span>
        </el-tag>
      </el-card>
    </el-col>

    <el-col :span="19">
      <el-card shadow="hover" class="card">
        <el-descriptions title="当前配置" :column="4" border>
          <el-descriptions-item
            v-for="(row, idx) in details"
            :key="idx"
            :label="row.label"
          >{{ row.value }}</el-descriptions-item>
        </el-descriptions>
      </el-card>
    </el-col>
  </el-row>

  <br>

  <div class="log-viewer" ref="logContainer">
    <p class="log-viewer-title">运行日志 <span style="font-size: 12px; color: #666;">(显示最近 {{ MAX_DISPLAY_LOGS }} 条)</span></p>
    <div v-for="(log, index) in displayedLogs" :key="index">
      <span :style="logStyle(log)">{{ log }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, nextTick, computed } from "vue";
import { ChooseFile, CheckDatasets } from "../../../wailsjs/go/proxy/Proxy";
import { ElMessage } from 'element-plus';
import {BrowserOpenURL, EventsEmit, EventsOff, EventsOn, OnFileDrop, OnFileDropOff} from "../../../wailsjs/runtime";
import {Config, useConfigStore} from "./store/types";

// 动态调用后端方法（因为 wailsjs 可能还没有生成）
const LoadFileFromPath = (filePath: string): Promise<Config> => {
  return (window as any).go?.proxy?.Proxy?.LoadFileFromPath?.(filePath) || Promise.reject('方法不存在');
};

const MAX_LOGS = 1000;  // 内存中保留的最大日志数量
const MAX_DISPLAY_LOGS = 400;  // 界面显示的最大日志数量

const logs = ref<string[]>([]);  // 日志数组
const logContainer = ref<HTMLElement | null>(null);  // 日志容器
const logQueue = ref<string[]>([]);  // 日志批量更新队列
let logBatchTimer: ReturnType<typeof setTimeout> | null = null;  // 批量更新定时器
let scrollTimer: ReturnType<typeof setTimeout> | null = null;  // 滚动防抖定时器

// 计算显示的日志（只显示最近的部分）
const displayedLogs = computed(() => {
  return logs.value.slice(-MAX_DISPLAY_LOGS);
});

const percent = ref(0.00);  // 进度条百分比
const disabled = ref(false);  // 禁用按钮

const started = ref(false);  // 任务是否已启动

const wailsDropPaths = ref<string[]>([]);  // Wails 拖拽路径
const isMounted = ref(false);  // 组件挂载状态
const hasDroppedFile = ref(false);  // 是否有拖拽的文件路径

const configState = useConfigStore();  // 获取配置状态
const data = ref<Config>();  // 配置信息
const details = ref([  // 配置详情
  {
    label: '监听绑定',
    value: configState.getSocksAddress(),
  },
  {
    label: '超时时间',
    value: `${configState.getTimeout()} s`,
  },
  {
    label: '协程数',
    value: `${configState.getCoroutineCount()}`,
  },
  {
    label: '有效代理',
    value: `${configState.getLiveProxies()} 条`,
  },
]);

const tagType = computed(() => {
  const s = configState.getStatus();
  if (s === 0 || s === 3) return 'danger';
  if (s === 2) return 'success';
  return 'primary';
});

const progressStatus = computed(() => {
  const s = configState.getStatus();
  if (s === 0 || s === 3) return 'exception';
  if (s === 2) return 'success';
  return undefined;
});

// 从日志中提取当前使用的代理IP
const extractCurrentProxyFromLog = (log: string): string | null => {
  // 匹配 "[INF] 当前使用代理 IP:PORT" 格式的日志
  const match = log.match(/\[INF\]\s*当前使用代理\s*([\d.]+:\d+)/);
  if (match && match[1]) {
    return match[1].split(':')[0]; // 返回IP部分
  }
  return null;
};

// 批量处理日志队列
const flushLogQueue = () => {
  if (logQueue.value.length === 0) {
    logBatchTimer = null;
    return;
  }

  // 保存新日志用于检查代理IP
  const newLogs = [...logQueue.value];
  
  // 批量添加日志
  logs.value.push(...newLogs);
  logQueue.value = [];

  // 限制日志数量，保留最近的一半
  if (logs.value.length > MAX_LOGS) {
    logs.value = logs.value.slice(-Math.floor(MAX_LOGS / 2));
  }

  // 只检查新添加的日志中是否有代理IP信息
  for (const log of newLogs) {
    const currentProxyIP = extractCurrentProxyFromLog(log);
    if (currentProxyIP) {
      configState.setCurrentIP(currentProxyIP);
      EventsEmit('current_ip', currentProxyIP);
      break;  // 只取最新的代理IP
    }
  }
  
  // 不再将日志发送到顶部全局日志中，这里本来就有日志记录

  // 防抖滚动到底部
  if (scrollTimer) {
    clearTimeout(scrollTimer);
  }
  scrollTimer = setTimeout(() => {
    nextTick(() => {
      if (logContainer.value) {
        logContainer.value.scrollTop = logContainer.value.scrollHeight;
      }
    });
    scrollTimer = null;
  }, 100);

  logBatchTimer = null;
};

// 处理日志输出（优化：批量更新，避免频繁DOM操作导致卡顿）
const handleLogEmits = (log: string) => {
  if (!log || typeof log !== 'string') return;

  // 添加到队列
  logQueue.value.push(log);

  // 如果定时器不存在，创建新的批量处理定时器
  if (!logBatchTimer) {
    logBatchTimer = setTimeout(flushLogQueue, 50);  // 50ms批量处理一次，平衡实时性和性能
  }
};

// 使用全局工具函数
const ensureAuthorized = async (): Promise<boolean> => {
  return true;
};

// 处理 Wails 拖拽文件
const handleFilePathInputDrop = async (paths: string[]) => {
  // 检查组件是否已挂载
  if (!isMounted.value) return;
  
  if (!paths || paths.length === 0) return;

  const filePath = paths[0];
  
  // 检查是否为绝对路径（Windows路径格式：C:\ 或 Unix路径格式：/）
  const isAbsolutePath = /^[a-zA-Z]:\\/.test(filePath) || filePath.startsWith('/');
  if (!isAbsolutePath) {
    return; // 只处理绝对路径
  }
  
  // 在 X-UI 模式下，导入代理文件前先做授权检查
  const xuiMode = configState.getXui && configState.getXui();
  if (xuiMode === "1" || xuiMode === "2") {
    const ok = await ensureAuthorized();
    if (!ok) {
      return;
    }
    // 再次检查组件是否已挂载（异步操作后）
    if (!isMounted.value) return;
  }

  // 设置文件路径
  configState.setFilePath(filePath);
  hasDroppedFile.value = true; // 标记已拖拽文件
  ElMessage.success(`已通过拖拽选择文件: ${filePath.split(/[\\/]/).pop()}`);
};

// 处理导入或开始处理
async function handleImportOrStart() {
  // 如果有拖拽的文件路径，直接开始处理
  if (hasDroppedFile.value && configState.getFilePath() && configState.getFilePath() !== '路径为空') {
    await startProcessing();
    return;
  }
  
  // 否则执行原有的导入逻辑
  await openFile();
}

// 开始处理文件
async function startProcessing() {
  // 在 X-UI 模式下，开始处理前先做授权检查
  const xuiMode = configState.getXui && configState.getXui();
  if (xuiMode === "1" || xuiMode === "2") {
    const ok = await ensureAuthorized();
    if (!ok) {
      return;
    }
  }

  disabled.value = true;
  
  const filePath = configState.getFilePath();
  if (!filePath || filePath === '路径为空') {
    ElMessage.error('请先选择文件');
    hasDroppedFile.value = false;
    disabled.value = false;
    return;
  }

  try {
    // 使用新的 LoadFileFromPath 方法来读取文件并开始处理
    const res = await LoadFileFromPath(filePath);
    if (res.Code !== 200) {
      // 检查是否有错误信息，如果有且不为空才显示
      const errorMsg = res.Error?.trim();
      if (errorMsg) {
        configState.setStatus(3);
        ElMessage.error(errorMsg);
      } else {
        // 如果没有错误信息，可能是任务已经成功启动（x-ui模式可能返回非200但实际成功）
        // 检查状态，如果状态是正在运行或已完成，不显示错误
        const status = configState.getStatus();
        if (status !== 1 && status !== 2) {
          configState.setStatus(3);
          ElMessage.error('读取文件失败');
        }
      }
      hasDroppedFile.value = false;
      disabled.value = false;
      return;
    }
    // 文件读取成功，LoadFileFromPath 会自动调用 CheckDatasets 开始处理
    hasDroppedFile.value = false;
  } catch (err: any) {
    // 如果 LoadFileFromPath 方法不存在，回退到 ChooseFile
    if (err === '方法不存在' || err.message?.includes('LoadFileFromPath')) {
      ElMessage.warning('请先重新编译项目以支持拖拽功能');
      hasDroppedFile.value = false;
      disabled.value = false;
      return;
    }
    // 检查错误信息是否为空
    const errorMsg = err.message?.trim() || err?.trim();
    if (errorMsg) {
      configState.setStatus(3);
      ElMessage.error(errorMsg);
    } else {
      configState.setStatus(3);
      ElMessage.error('读取文件失败');
    }
    hasDroppedFile.value = false;
    disabled.value = false;
  }
}

// 打开文件选择框
async function openFile() {
  // 在 X-UI 模式下，导入代理文件前先做授权检查
  const xuiMode = configState.getXui && configState.getXui();
  if (xuiMode === "1" || xuiMode === "2") {
    const ok = await ensureAuthorized();
    if (!ok) {
      return;
    }
  }

  disabled.value = true;
  ChooseFile().then((res) => {
    data.value = res as unknown as Config;
    if (data.value.Code !== 200) {
      configState.setStatus(3);
      // 检查错误信息是否为空
      const errorMsg = data.value.Error?.trim();
      if (errorMsg) {
        ElMessage.error(errorMsg);
      }
      return;
    }
    // 导入成功后重置拖拽标记
    hasDroppedFile.value = false;
  }).catch((err) => {
    configState.setStatus(3);
    // 检查错误信息是否为空
    const errorMsg = (err?.message || err)?.trim();
    if (errorMsg) {
      ElMessage.error(errorMsg);
    }
  }).finally(() => {
    disabled.value = false;
  });
}

// 日志样式
const logStyle = (log: string) => {
  if (log.includes('[ERR]')) {
    return { color: '#b35351' };
  } else if (log.includes('[INF]')) {
    return { color: '#29b445' };
  } else if (log.includes('[WAR]')) {
    return { color: '#b16f34' };
  }
  return {};
};

const stopTask = () => {
  console.log(configState.getStatus())
  if (configState.getStatus() !== 2) {
    ElMessage.error('当前没有正在运行的任务。');
    return;
  }

  configState.setStatus(0);
  disabled.value = true;

  // 停止任务时，先处理剩余的日志队列
  if (logQueue.value.length > 0) {
    flushLogQueue();
  }

  configState.stopTask().then(() => {
    configState.setStatus(0);
    disabled.value = false;
    ElMessage.success('任务已经成功停止。');
  }).catch((err: any) => {
    // 检查错误信息是否为空
    const errorMsg = (err?.message || err)?.trim();
    if (errorMsg) {
      ElMessage.error(errorMsg);
    }
    disabled.value = false;
  });
};

onMounted(() => {
  isMounted.value = true;
  configState.getProfile()
  
  // 设置全局文件拖拽监听（支持桌面拖拽）- 只使用 Wails 拖拽
  OnFileDrop((x: number, y: number, paths: string[]) => {
    // 检查组件是否已挂载
    if (!isMounted.value) return;
    if (paths && paths.length > 0) {
      handleFilePathInputDrop(paths);
    }
  }, false);

  // 全局事件广播（后端 runtime.EventsEmit）
  EventsOn('global-file-drop', (paths: string[]) => {
    // 检查组件是否已挂载
    if (!isMounted.value) return;
    if (paths && paths.length > 0) {
      handleFilePathInputDrop(paths);
    }
  });

  EventsOn('log_update', handleLogEmits);
  EventsOn('task_progress', (p: number | string) => {
    // 将字符串转换为数字（后端发送的是字符串格式 "1.00"）
    if (typeof p === 'string') {
      percent.value = parseFloat(p) || 0;
    } else {
      percent.value = p;
    }
  });
  EventsOn('start_task', (profile: Config) => {
    started.value = true;
    configState.setFilePath(profile.FilePath);
    configState.setStatus(1)
    details.value = [
      { label: '监听绑定', value: profile.SocksAddress },
      { label: '协程数',   value: `${profile.CoroutineCount}` },
      { label: '超时时间', value: `${profile.Timeout}s` },
    ]
    
    // 任务开始时清空日志和队列
    logs.value = [];
    logQueue.value = [];
    if (logBatchTimer) {
      clearTimeout(logBatchTimer);
      logBatchTimer = null;
    }
    if (scrollTimer) {
      clearTimeout(scrollTimer);
      scrollTimer = null;
    }
  });
  EventsOn('is_ready', (callback: string) => {
    started.value = false;
    configState.setStatus(2)

    const liveCount = parseInt(callback) || 0;
    configState.LiveProxies = liveCount;

    details.value = details.value.filter(item => item.label !== '有效代理');
    details.value.push({
      label: '有效代理',
      value: `${liveCount} 条`,
    });

    ElMessage.success(`共有 ${callback} 条有效数据`);
  });
});

onUnmounted(() => {
  // 先设置挂载状态为 false，防止回调继续执行
  isMounted.value = false;
  
  // 清理定时器
  if (logBatchTimer) {
    clearTimeout(logBatchTimer);
    logBatchTimer = null;
  }
  if (scrollTimer) {
    clearTimeout(scrollTimer);
    scrollTimer = null;
  }
  
  // 处理剩余的日志队列
  if (logQueue.value.length > 0) {
    flushLogQueue();
  }
  
  EventsOff('log_update');
  EventsOff('task_progress');
  EventsOff('start_task');
  EventsOff('is_ready');
  EventsOff('current_ip');
  EventsOff('global-file-drop');
  OnFileDropOff();
});
</script>

<style scoped>
/* 样式保持不变 */
.log-viewer {
  /* 高度随可用区域伸缩（嵌在模块页签/独立窗口均不过度超出），内容区内滚动 */
  height: clamp(240px, 46vh, 560px);
  min-height: 200px;
  border-radius: 8px;
  overflow-y: auto;
  background-color: rgba(255, 255, 255, 0.4);
  color: #e88500;
  scrollbar-width: thin;
  border: 1px solid rgba(197, 186, 186, 0.4);
}

.log-viewer-title{
  text-align: left;
  padding-left: 16px;
  color: #000000;
  font-weight: bold;
}

.RunGetStatus{
  border-radius: 5px;
}

.log-viewer::-webkit-scrollbar {
  width: 8px;
}

.log-viewer::-webkit-scrollbar-track {
  background: #1e1e1e;
}

.log-viewer::-webkit-scrollbar-thumb {
  background: #6b7280;
  border-radius: 4px;
}

.log-viewer::-webkit-scrollbar-thumb:hover {
  background: #4b5563;
}

.log-viewer > div {
  margin-left: 10px;
  padding: 2px;
}

.card {
  background-color: rgba(255, 255, 255, 0.4);
  border: 1px solid rgba(197, 186, 186, 0.4);
  border-radius: 8px;
}

.progress {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-left: 15px;
}

:deep(.import-btn) {
  color: rgb(226, 152, 13);
  border-color: var(--el-color-primary);
}
</style>