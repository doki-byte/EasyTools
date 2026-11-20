<template>
  <div class="redis-command-container">
    <!-- 命令执行模式 -->
    <div v-if="mode === 'command'" class="command-panel">
      <div class="command-input-section">
        <div class="command-input-row">
          <el-select
              v-model="command"
              :placeholder="commandPlaceholder"
              filterable
              allow-create
              default-first-option
              @keyup.enter="executeCommand"
              size="default"
              style="flex: 1; margin-right: 8px;"
          >
            <el-option
                v-for="cmd in commonCommands"
                :key="cmd.id || cmd.command"
                :label="`${cmd.name} (${cmd.command})`"
                :value="cmd.command"
            />
          </el-select>
          <el-button @click="showCommandConfigDialog = true" :icon="Setting" title="配置命令">配置</el-button>
          <el-button @click="executeCommand" type="primary" :icon="Right" :loading="executing">执行</el-button>
        </div>
      </div>


      <div class="result-section" v-if="commandResult">
        <div class="result-header-row">
          <span class="result-title">执行结果</span>
          <el-alert
              :title="`执行${commandResult.success ? '成功' : '失败'} - 耗时: ${commandResult.duration}ms`"
              :type="commandResult.success ? 'success' : 'error'"
              :closable="false"
              :description="commandResult.error || ''"
              class="result-alert"
              :show-icon="false"
          />
        </div>
        <div class="result-content">
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

    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-panel">
      <div class="detail-toolbar">
        <el-button type="primary" @click="loadDbInfo" :loading="infoLoading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
      <div class="detail-content">
        <el-table
            :data="info"
            border
            stripe
            style="width: 100%"
            height="100%"
            size="small"
            v-loading="infoLoading"
        >
          <el-table-column type="index" width="60"/>
          <el-table-column prop="key" label="Key" min-width="200"/>
          <el-table-column prop="value" label="Value" show-overflow-tooltip />
        </el-table>
      </div>
    </div>

    <!-- 命令配置对话框 -->
    <el-dialog
        v-model="showCommandConfigDialog"
        title="命令配置管理"
        width="900px"
        @open="async () => { await initApiClient(); await loadCommands(); }"
    >
      <div class="config-toolbar">
        <el-button type="primary" @click="handleAddCommand" :icon="Plus">新增</el-button>
        <el-button @click="handleResetCommands" type="warning" :icon="Refresh">恢复初始值</el-button>
        <el-button @click="loadCommands" :icon="Refresh">刷新</el-button>
        <div style="flex: 1;"></div>
        <el-input
            v-model="commandSearch"
            placeholder="搜索命令"
            clearable
            style="width: 200px;"
            @input="loadCommands"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
      <el-table
          :data="commandList"
          border
          stripe
          style="width: 100%; margin-top: 12px;"
          v-loading="commandLoading"
      >
        <el-table-column type="index" width="60" label="#" />
        <el-table-column prop="name" label="名称" min-width="150" />
        <el-table-column prop="command" label="命令" min-width="300" show-overflow-tooltip />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleEditCommand(row)">编辑</el-button>
            <el-button link type="danger" size="small" @click="handleDeleteCommand(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
          v-model:current-page="commandPagination.page"
          v-model:page-size="commandPagination.pageSize"
          :total="commandPagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          style="margin-top: 12px; justify-content: flex-end;"
          @size-change="loadCommands"
          @current-change="loadCommands"
      />
    </el-dialog>

    <!-- 命令编辑对话框 -->
    <el-dialog
        v-model="showCommandEditDialog"
        :title="editingCommand.id ? '编辑命令' : '新增命令'"
        width="600px"
    >
      <el-form :model="editingCommand" label-width="80px">
        <el-form-item label="名称" required>
          <el-input v-model="editingCommand.name" placeholder="请输入命令名称" />
        </el-form-item>
        <el-form-item label="命令" required>
          <el-input
              v-model="editingCommand.command"
              type="textarea"
              :rows="3"
              placeholder="请输入命令"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCommandEditDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSaveCommand">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue';
import { DbInfo, ExecuteCommand, GetCommandHistory } from '../../../wailsjs/go/nosql/NoSQL';
import { GetPortConfig } from '../../../wailsjs/go/controller/Server';
import { ElMessage, ElMessageBox } from 'element-plus';
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';
import { CopyDocument, Right, Refresh, Setting, Plus, Search } from "@element-plus/icons-vue";
import axios from 'axios';

// 定义 props
const props = defineProps({
  connIdentity: String,
  dbType: {
    type: String,
    default: 'redis' // 'redis' 或 'mongodb'
  },
  mode: {
    type: String,
    default: 'command', // 'command' 或 'detail'
    validator: (value) => ['command', 'detail'].includes(value)
  }
});

// 响应式数据
const info = ref([]);
const infoLoading = ref(false);
const command = ref('');
const commandResult = ref(null);
const commandHistory = ref([]);
const executing = ref(false);

// API 客户端
const apiClient = ref(null);
const portConfig = ref(null);

// 命令列表
const commonCommands = ref([]);
const commandList = ref([]);
const commandLoading = ref(false);
const commandSearch = ref('');
const commandPagination = ref({
  page: 1,
  pageSize: 10,
  total: 0
});

// 配置对话框
const showCommandConfigDialog = ref(false);
const showCommandEditDialog = ref(false);
const editingCommand = ref({
  id: '',
  name: '',
  command: ''
});

// 初始化 API 客户端
const initApiClient = async () => {
  if (apiClient.value) {
    return; // 已经初始化过了
  }
  try {
    portConfig.value = await GetPortConfig();
    if (!portConfig.value) {
      console.error('无法获取端口配置');
      return;
    }
    const port = portConfig.value?.mainPort || 52867;
    const baseURL = `http://127.0.0.1:${port}`;
    apiClient.value = axios.create({
      baseURL: baseURL,
      timeout: 20000,
    });
    console.log('API 客户端初始化成功，端口:', port);
  } catch (error) {
    console.error('初始化API客户端失败:', error);
    apiClient.value = null;
  }
};

// 检查服务器是否可用
const checkServerAvailable = async (retries = 3, delay = 500) => {
  if (!apiClient.value) return false;
  
  for (let i = 0; i < retries; i++) {
    try {
      const response = await apiClient.value.get('/api/port-config', { timeout: 2000 });
      if (response.status === 200) {
        return true;
      }
    } catch (error) {
      if (i < retries - 1) {
        await new Promise(resolve => setTimeout(resolve, delay));
      }
    }
  }
  return false;
};

// 加载命令列表
const loadCommands = async () => {
  // 确保 API 客户端已初始化
  if (!apiClient.value) {
    await initApiClient();
  }
  if (!apiClient.value) {
    ElMessage.error('API 客户端初始化失败，请检查服务是否启动');
    return;
  }
  
  // 检查服务器是否可用
  const serverAvailable = await checkServerAvailable();
  if (!serverAvailable) {
    ElMessage.warning('服务器尚未启动，请稍候再试');
    return;
  }
  
  commandLoading.value = true;
  try {
    const dbType = (props.dbType || 'redis').toLowerCase();
    const dbTypeMap = {
      'mongodb': 'mongodb',
      'mongo': 'mongodb',
      'redis': 'redis'
    };
    const actualDbType = dbTypeMap[dbType] || 'redis';
    
    const response = await apiClient.value.get('/api/config/nosql-commands', {
      params: {
        dbType: actualDbType,
        page: commandPagination.value.page,
        pageSize: commandPagination.value.pageSize,
        search: commandSearch.value || undefined
      }
    });
    
    if (response.data.code === 0) {
      const data = response.data.data;
      commandList.value = data.list || [];
      commandPagination.value.total = data.total || 0;
      // 更新常用命令列表（用于下拉框）
      commonCommands.value = data.list || [];
    } else {
      ElMessage.warning(response.data.msg || '加载命令列表失败');
    }
  } catch (error) {
    console.error('加载命令列表失败:', error);
    const errorMsg = error.response?.data?.msg || error.message || '加载命令列表失败';
    
    // 如果是连接错误，提示用户等待服务器启动
    if (error.code === 'ERR_NETWORK' || error.code === 'ECONNREFUSED') {
      ElMessage.warning('服务器连接失败，请确保后端服务已启动');
      // 重置 API 客户端，下次重试
      apiClient.value = null;
    } else {
      ElMessage.error(errorMsg);
    }
  } finally {
    commandLoading.value = false;
  }
};

// 新增命令
const handleAddCommand = () => {
  editingCommand.value = {
    id: '',
    name: '',
    command: ''
  };
  showCommandEditDialog.value = true;
};

// 编辑命令
const handleEditCommand = (row) => {
  editingCommand.value = {
    id: row.id,
    name: row.name,
    command: row.command
  };
  showCommandEditDialog.value = true;
};

// 保存命令
const handleSaveCommand = async () => {
  if (!editingCommand.value.name || !editingCommand.value.command) {
    ElMessage.warning('请填写完整信息');
    return;
  }
  
  if (!apiClient.value) {
    await initApiClient();
  }
  if (!apiClient.value) return;
  
  try {
    const dbType = (props.dbType || 'redis').toLowerCase();
    const dbTypeMap = {
      'mongodb': 'mongodb',
      'mongo': 'mongodb',
      'redis': 'redis'
    };
    const actualDbType = dbTypeMap[dbType] || 'redis';
    
    if (editingCommand.value.id) {
      // 更新
      await apiClient.value.put(`/api/config/nosql-commands/${editingCommand.value.id}?dbType=${actualDbType}`, {
        name: editingCommand.value.name,
        command: editingCommand.value.command
      });
      ElMessage.success('更新成功');
    } else {
      // 新增
      await apiClient.value.post(`/api/config/nosql-commands?dbType=${actualDbType}`, {
        name: editingCommand.value.name,
        command: editingCommand.value.command
      });
      ElMessage.success('新增成功');
    }
    
    showCommandEditDialog.value = false;
    await loadCommands();
  } catch (error) {
    console.error('保存命令失败:', error);
    ElMessage.error('保存命令失败: ' + (error.response?.data?.msg || error.message));
  }
};

// 删除命令
const handleDeleteCommand = async (row) => {
  if (!apiClient.value) {
    await initApiClient();
  }
  if (!apiClient.value) return;

  try {
    await confirmDangerousAction({
      title: '删除命令',
      message: `将删除自定义命令「${row.name || row.command || row.id}」。`,
      expectedInput: DANGER_DELETE_TOKEN
    });
  } catch {
    return;
  }

  try {
    const dbType = (props.dbType || 'redis').toLowerCase();
    const dbTypeMap = {
      'mongodb': 'mongodb',
      'mongo': 'mongodb',
      'redis': 'redis'
    };
    const actualDbType = dbTypeMap[dbType] || 'redis';
    
    await apiClient.value.delete(`/api/config/nosql-commands/${row.id}?dbType=${actualDbType}`);
    ElMessage.success('删除成功');
    await loadCommands();
  } catch (error) {
    console.error('删除命令失败:', error);
    ElMessage.error('删除命令失败: ' + (error.response?.data?.msg || error.message));
  }
};

// 恢复初始值
const handleResetCommands = async () => {
  if (!apiClient.value) {
    await initApiClient();
  }
  if (!apiClient.value) return;
  
  try {
    await confirmDangerousAction({
      title: '恢复初始命令',
      message: '确定要恢复初始值吗？这将覆盖所有自定义命令。',
      expectedInput: DANGER_DELETE_TOKEN
    });

    await apiClient.value.post('/api/config/nosql-commands/reset');
    ElMessage.success('恢复初始值成功');
    await loadCommands();
  } catch (error) {
    if (error !== 'cancel') {
      console.error('恢复初始值失败:', error);
      ElMessage.error('恢复初始值失败: ' + (error.response?.data?.msg || error.message));
    }
  }
};

// 计算属性
const showCommonCommands = computed(() => !command.value.trim());

// 命令输入提示文本
const commandPlaceholder = computed(() => {
  const dbTypeLower = (props.dbType || 'redis').toLowerCase();
  if (dbTypeLower === 'mongodb' || dbTypeLower === 'mongo') {
    return '输入 MongoDB 命令，例如：test.users.find({}) 或 admin.runCommand({"serverStatus": 1})';
  }
  return '输入 Redis 命令，例如：GET key 或 HSET hash field value';
});

// 方法定义
const loadCommandHistory = () => {
  if (!props.connIdentity) return;
  GetCommandHistory(props.connIdentity).then(res => {
    if (res.code === 200 && res.data) {
      commandHistory.value = res.data;
    }
  });
};

const loadDbInfo = () => {
  if (!props.connIdentity) {
    ElMessage.warning('请先选择连接');
    return;
  }
  infoLoading.value = true;
  DbInfo(props.connIdentity).then(res => {
    infoLoading.value = false;
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return;
    }
    info.value = res.data || [];
  }).catch(err => {
    infoLoading.value = false;
    ElMessage.error(`加载详情失败: ${err.message}`);
  });
};

const executeCommand = () => {
  if (!command.value.trim()) {
    ElMessage.warning('请输入命令')
    return;
  }

  if (!props.connIdentity) {
    ElMessage.warning('请先选择连接');
    return;
  }

  executing.value = true;
  ExecuteCommand(props.connIdentity, command.value).then(res => {
    executing.value = false;
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
    executing.value = false;
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

// 监听连接变化，加载历史记录
watch(() => props.connIdentity, (newVal) => {
  if (newVal) {
    loadCommandHistory();
  }
});

// 监听模式变化，如果切换到详情模式且没有数据，自动加载
watch(() => props.mode, (newMode) => {
  if (newMode === 'detail' && props.connIdentity && info.value.length === 0) {
    loadDbInfo();
  }
});

// 生命周期
onMounted(async () => {
  // 初始化 API 客户端
  await initApiClient();
  // 预加载命令列表到下拉框（用于下拉选择）
  if (apiClient.value) {
    try {
      await loadCommands();
    } catch (error) {
      console.error('预加载命令列表失败:', error);
    }
  }
  if (props.connIdentity) {
    loadCommandHistory();
  }
});

// 监听数据库类型变化，重新加载命令
watch(() => props.dbType, () => {
  loadCommands();
});
</script>

<style scoped>
.redis-command-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.command-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 12px;
}

.command-input-section {
  margin-bottom: 16px;
}

.command-input-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.config-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.command-prompt {
  color: #409EFF;
  font-weight: 600;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.common-commands {
  margin-bottom: 16px;
}

.common-commands-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.command-tag {
  cursor: pointer;
  margin-right: 8px;
  margin-bottom: 8px;
}

.result-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  margin-top: 16px;
}

.result-header-row {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  gap: 12px;
}

.result-title {
  font-size: 14px;
  font-weight: 600;
  color: #333;
  white-space: nowrap;
}

.result-alert {
  flex: 1;
  margin: 0;
  min-width: 0;
}

.result-alert :deep(.el-alert__content) {
  padding: 0;
}

.result-alert :deep(.el-alert__title) {
  font-size: 13px;
  line-height: 1.5;
  margin: 0;
  word-break: break-word;
}

.result-alert :deep(.el-alert__description) {
  font-size: 12px;
  margin-top: 4px;
  word-break: break-word;
}

.result-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 400px;
}

.result-data {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  margin-top: 12px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
}

.result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: rgba(0, 0, 0, 0.02);
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
  font-size: 12px;
  flex-shrink: 0;
}

.result-type {
  font-weight: 600;
  color: #409EFF;
  font-size: 13px;
}

.result-count {
  margin-left: auto;
  margin-right: 10px;
  color: #409EFF;
  font-weight: 600;
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
  flex: 1;
  padding: 0;
  overflow: auto;
}

.result-body::-webkit-scrollbar {
  width: 6px;
}

.result-body::-webkit-scrollbar-track {
  background: transparent;
}

.result-body::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 3px;
}

.result-body::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.3);
}

/* Redis INFO 结果美化 */
.redis-info-result {
  padding: 12px;
}

.info-section {
  margin-bottom: 12px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  overflow: hidden;
  background: rgba(0, 0, 0, 0.02);
}

.section-title {
  margin: 0;
  padding: 8px 12px;
  background: rgba(64, 158, 255, 0.15);
  color: #409EFF;
  font-size: 14px;
  font-weight: 600;
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
}

.section-content {
  padding: 0;
}

.info-item {
  display: flex;
  align-items: flex-start;
  padding: 8px 12px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
  transition: background-color 0.2s;
}

.info-item:hover {
  background: rgba(0, 0, 0, 0.03);
}

.info-item:last-child {
  border-bottom: none;
}

.info-key {
  flex: 0 0 200px;
  font-weight: 600;
  color: #409EFF;
  font-size: 13px;
  word-break: break-word;
}

.info-value {
  flex: 1;
  color: #E6A23C;
  font-size: 13px;
  font-weight: 500;
  word-break: break-word;
  padding-left: 8px;
}

/* 普通文本结果 */
.result-body pre {
  margin: 0;
  padding: 12px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
  background: transparent;
  font-weight: 500;
}

.result-body pre.string {
  color: #67c23a;
  font-weight: 600;
}

.result-body pre.array {
  color: #409EFF;
  font-weight: 600;
}

.result-body pre.object {
  color: #909399;
  font-weight: 600;
}

.result-body pre.number {
  color: #E6A23C;
  font-weight: 600;
}

.result-body pre.null {
  color: #999;
  font-style: italic;
  font-weight: 500;
}

.empty-result {
  padding: 40px 0;
  text-align: center;
  color: #8c8c8c;
}

.command-history {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.history-list {
  max-height: 120px;
  overflow-y: auto;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.02);
}

.history-list::-webkit-scrollbar {
  width: 6px;
}

.history-list::-webkit-scrollbar-track {
  background: transparent;
}

.history-list::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 3px;
}

.history-list::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.3);
}

.history-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
  cursor: pointer;
  transition: background-color 0.2s;
}

.history-item:hover {
  background-color: rgba(0, 0, 0, 0.03);
}

.history-item:last-child {
  border-bottom: none;
}

.command-text {
  flex: 1;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  color: #409EFF;
  font-weight: 500;
}

.result-status {
  margin-left: 10px;
  font-weight: bold;
}

.result-status.success {
  color: #67c23a;
}

.result-status.error {
  color: #f56c6c;
}

/* 详情面板 */
.detail-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 12px;
}

.detail-toolbar {
  margin-bottom: 12px;
  flex-shrink: 0;
}

.detail-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
