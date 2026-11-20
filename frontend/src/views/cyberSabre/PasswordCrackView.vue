<template>
  <div class="password-crack-container">
    <div class="main-layout">
      <!-- 左侧边栏：服务选择 -->
      <div class="sidebar">
        <div class="sidebar-header">
          <div class="sidebar-title">Serv</div>
          <div class="sidebar-title">Port</div>
        </div>
        <div class="service-list">
          <div 
            v-for="service in services" 
            :key="service.name"
            class="service-item"
          >
            <el-checkbox 
              v-model="service.checked" 
              @change="handleServiceChange"
            />
            <span class="service-name">{{ service.name }}</span>
            <el-input 
              v-model.number="service.port" 
              size="small" 
              class="port-input"
              @change="handlePortChange(service)"
            />
          </div>
        </div>
      </div>

      <!-- 右侧主内容区 -->
      <div class="main-content">
        <!-- 顶部控制面板 -->
        <div class="control-panel">
          <!-- 配置区域（合并在一起） -->
          <div class="config-section">
            <div class="config-row">
              <div class="target-input-area">
                <el-input
                  v-model="targetText"
                  type="textarea"
                  :rows="7"
                  placeholder="每行一个目标IP或域名，例如：&#10;172.18.2.211&#10;192.168.1.1&#10;example.com"
                  class="target-textarea"
                />
                <div class="target-actions">
                  <el-button size="small" :icon="Refresh" @click="refreshTargets">刷新</el-button>
                  <el-button size="small" type="primary" @click="openDictConfig">字典管理</el-button>
                  <el-button size="small" @click="openSSHConfig">SSH隧道</el-button>
                </div>
              </div>
              <div class="config-items">
                <div class="config-group">
                  <div class="param-item">
                    <label>Oracle监听</label>
                    <el-input v-model="crackParams.oracleListener" size="small" placeholder="orcl" style="width: 120px" />
                  </div>
                  <div class="param-item">
                    <label>指令执行</label>
                    <el-input v-model="crackParams.command" size="small" placeholder="whoami" style="width: 120px" />
                  </div>
                  <div class="param-item">
                    <label>线程</label>
                    <el-input-number 
                      v-model="globalSettings.threads" 
                      :min="1" 
                      :max="1000"
                      size="small"
                      controls-position="right"
                      style="width: 100px"
                    />
                  </div>
                  <div class="param-item">
                    <label>超时</label>
                    <el-input-number 
                      v-model="globalSettings.timeout" 
                      :min="1000" 
                      :max="30000"
                      :step="1000"
                      size="small"
                      controls-position="right"
                      style="width: 100px"
                    />
                  </div>
                  <div class="param-item">
                    <el-checkbox v-model="globalSettings.onlyOneAccount">
                      仅破解一个账户
                    </el-checkbox>
                  </div>
                  <div class="param-item">
                    <el-switch 
                      v-model="globalSettings.fingerprint"
                      active-text="指纹识别"
                    />
                  </div>
                </div>
                <div class="action-buttons">
                  <div class="alive-toggle">
                    <el-switch 
                      v-model="globalSettings.checkAlive"
                      active-text="存活检测"
                    />
                  </div>
                  <el-button 
                    type="success" 
                    size="small"
                    :icon="VideoPlay" 
                    @click="startCrack"
                    :loading="cracking"
                    :disabled="cracking"
                  >
                    Crack
                  </el-button>
                  <el-button 
                    type="danger" 
                    size="small"
                    :icon="VideoPause" 
                    @click="stopCrack"
                    :disabled="!cracking"
                  >
                    Stop
                  </el-button>
                  <el-button 
                    type="primary" 
                    size="small"
                    :icon="Download" 
                    @click="handleExport"
                    :disabled="results.length === 0"
                  >
                    导出
                  </el-button>
                </div>
                <div class="progress-inline">
                  <div class="progress-card alive">
                    <div class="progress-head">
                      <span class="progress-title">存活</span>
                      <span class="progress-value">{{ alivePercent }}%</span>
                    </div>
                    <el-progress :percentage="alivePercent" :show-text="false" :stroke-width="8" />
                  </div>
                  <div class="progress-card crack">
                    <div class="progress-head">
                      <span class="progress-title">爆破</span>
                      <span class="progress-value">{{ crackPercent }}%</span>
                    </div>
                    <el-progress :percentage="crackPercent" :show-text="false" :stroke-width="8" />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 结果表格 -->
        <div class="results-section">
          <div class="results-toolbar">
            <el-button 
              type="danger" 
              size="small"
              :disabled="results.length === 0"
              @click="handleClearResults"
            >
              清空
            </el-button>
          </div>
          <el-table
            :data="pagedResults"
            v-loading="loading"
            stripe
            border
            height="100%"
            style="width: 100%"
            empty-text="No Data"
            @row-contextmenu="handleRowContextMenu"
          >
            <el-table-column prop="id" min-width="100">
              <template #header>
                <TableColumnFilterHeader
                  v-model="columnFilters.id"
                  label="ID"
                  sortable
                  :sort-asc-active="crackSort.prop === 'id' && crackSort.order === 'ascending'"
                  :sort-desc-active="crackSort.prop === 'id' && crackSort.order === 'descending'"
                  @sort-asc="() => handleCrackColSort('id', 'ascending')"
                  @sort-desc="() => handleCrackColSort('id', 'descending')"
                  @filter-change="handleCrackColumnFilter"
                />
              </template>
            </el-table-column>
            <el-table-column prop="host" min-width="160">
              <template #header>
                <TableColumnFilterHeader
                  v-model="columnFilters.host"
                  label="IP"
                  sortable
                  :sort-asc-active="crackSort.prop === 'host' && crackSort.order === 'ascending'"
                  :sort-desc-active="crackSort.prop === 'host' && crackSort.order === 'descending'"
                  @sort-asc="() => handleCrackColSort('host', 'ascending')"
                  @sort-desc="() => handleCrackColSort('host', 'descending')"
                  @filter-change="handleCrackColumnFilter"
                />
              </template>
            </el-table-column>
            <el-table-column prop="port" min-width="100">
              <template #header>
                <TableColumnFilterHeader
                  v-model="columnFilters.port"
                  label="Port"
                  sortable
                  :sort-asc-active="crackSort.prop === 'port' && crackSort.order === 'ascending'"
                  :sort-desc-active="crackSort.prop === 'port' && crackSort.order === 'descending'"
                  @sort-asc="() => handleCrackColSort('port', 'ascending')"
                  @sort-desc="() => handleCrackColSort('port', 'descending')"
                  @filter-change="handleCrackColumnFilter"
                />
              </template>
            </el-table-column>
            <el-table-column prop="protocol" min-width="120">
              <template #header>
                <TableColumnFilterHeader
                  v-model="columnFilters.protocol"
                  label="Serv"
                  sortable
                  :sort-asc-active="crackSort.prop === 'protocol' && crackSort.order === 'ascending'"
                  :sort-desc-active="crackSort.prop === 'protocol' && crackSort.order === 'descending'"
                  @sort-asc="() => handleCrackColSort('protocol', 'ascending')"
                  @sort-desc="() => handleCrackColSort('protocol', 'descending')"
                  @filter-change="handleCrackColumnFilter"
                />
              </template>
            </el-table-column>
            <el-table-column prop="username" min-width="128">
              <template #header>
                <TableColumnFilterHeader v-model="columnFilters.username" label="User" @filter-change="handleCrackColumnFilter" />
              </template>
            </el-table-column>
            <el-table-column prop="password" min-width="160">
              <template #header>
                <TableColumnFilterHeader v-model="columnFilters.password" label="Pass" @filter-change="handleCrackColumnFilter" />
              </template>
              <template #default="{ row }">
                <span>{{ row.password || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="message" min-width="220" show-overflow-tooltip>
              <template #header>
                <TableColumnFilterHeader v-model="columnFilters.message" label="Banner" @filter-change="handleCrackColumnFilter" />
              </template>
            </el-table-column>
            <el-table-column min-width="132">
              <template #header>
                <TableColumnFilterHeader v-model="columnFilters.command_result" label="CmdExec" @filter-change="handleCrackColumnFilter" />
              </template>
              <template #default="{ row }">
                <span>{{ row.command_result || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" min-width="196">
              <template #header>
                <TableColumnFilterHeader
                  v-model="columnFilters.created_at"
                  label="Time"
                  sortable
                  :sort-asc-active="crackSort.prop === 'created_at' && crackSort.order === 'ascending'"
                  :sort-desc-active="crackSort.prop === 'created_at' && crackSort.order === 'descending'"
                  @sort-asc="() => handleCrackColSort('created_at', 'ascending')"
                  @sort-desc="() => handleCrackColSort('created_at', 'descending')"
                  @filter-change="handleCrackColumnFilter"
                />
              </template>
              <template #default="{ row }">
                {{ formatTime(row.created_at) }}
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 底部状态栏 -->
      <div class="status-bar">
        <div class="status-item">
          <span class="status-label">服务存活:</span>
          <span class="status-value">{{ statistics.aliveCount || 0 }}</span>
        </div>
        <div class="status-item">
          <span class="status-label">破解成功:</span>
          <span class="status-value">{{ statistics.successCount || 0 }}</span>
        </div>
        <div class="status-item">
          <span class="status-label">检测队列:</span>
          <span class="status-value">{{ statistics.queueCurrent || 0 }}/{{ statistics.queueTotal || 0 }}</span>
        </div>
        <div class="status-item">
          <span class="status-label">用时:</span>
          <span class="status-value">{{ elapsedTime }}s</span>
        </div>
        <div class="status-spacer"></div>
        <div class="status-pagination">
          <el-pagination
            small
            background
            layout="total, sizes, prev, pager, next, jumper"
            :total="totalResults"
            :page-size="pageSize"
            :current-page="currentPage"
            :page-sizes="[10, 20, 50, 100, 200]"
            @current-change="handlePageChange"
            @size-change="handleSizeChange"
          />
        </div>
      </div>
      </div>
    </div>

    <!-- SSH隧道配置对话框 -->
    <el-dialog v-model="sshConfigVisible" title="SSH隧道配置" width="600px">
      <el-form :model="sshConfig" label-width="100px" size="default">
        <el-form-item label="启用隧道">
          <el-switch v-model="sshConfig.enabled" />
        </el-form-item>
        <el-form-item label="SSH主机">
          <el-select 
            v-model="sshConfig.address" 
            placeholder="从简练助手选择或手动输入"
            filterable
            allow-create
            style="width: 100%"
          >
            <el-option
              v-for="host in sshHostList"
              :key="host.id"
              :label="`${host.name} (${host.address}:${host.port})`"
              :value="host.address"
              @click="selectSSHHost(host)"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="SSH端口">
          <el-input-number 
            v-model="sshConfig.port" 
            :min="1" 
            :max="65535"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="SSH用户">
          <el-input v-model="sshConfig.username" placeholder="SSH用户名" />
        </el-form-item>
        <el-form-item label="SSH密码">
          <el-input 
            v-model="sshConfig.password" 
            type="password" 
            placeholder="SSH密码"
            show-password
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadSSHHostsFromConnect">从简练助手获取</el-button>
          <el-button @click="saveSSHConfig">保存配置</el-button>
          <el-button @click="loadSSHConfig">加载配置</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>

    <!-- 字典配置对话框 -->
    <el-dialog v-model="dictConfigVisible" title="字典管理" width="800px">
      <div class="dict-config-content">
        <el-tabs v-model="dictActiveTab" type="border-card">
          <el-tab-pane label="用户名字典" name="username">
            <div class="dict-section">
              <div class="dict-header">
                <el-select
                  v-model="selectedProtocol"
                  placeholder="选择协议"
                  style="width: 200px; margin-bottom: 16px;"
                  @change="loadUsernameDict"
                >
                  <el-option
                    v-for="protocol in availableProtocols"
                    :key="protocol"
                    :label="protocol"
                    :value="protocol"
                  />
                </el-select>
                <div style="display: flex; gap: 8px;">
                  <el-button
                    type="warning"
                    @click="resetUsernameDict"
                    :loading="saving"
                    :disabled="!selectedProtocol"
                  >
                    恢复默认
                  </el-button>
                  <el-button
                    type="primary"
                    @click="saveUsernameDict"
                    :loading="saving"
                    :disabled="!selectedProtocol"
                  >
                    保存
                  </el-button>
                </div>
              </div>
              <el-input
                v-model="usernameDict"
                type="textarea"
                :rows="15"
                placeholder="每行一个用户名"
                style="width: 100%;"
              />
              <div class="dict-info">
                <el-text type="info" size="small">
                  当前协议: {{ selectedProtocol || '未选择' }} | 
                  条目数: {{ usernameDict ? usernameDict.split('\n').filter(line => line.trim()).length : 0 }}
                </el-text>
              </div>
            </div>
          </el-tab-pane>
          <el-tab-pane label="密码字典" name="password">
            <div class="dict-section">
              <div class="dict-header">
                <div style="display: flex; gap: 8px;">
                  <el-button
                    type="warning"
                    @click="resetPasswordDict"
                    :loading="saving"
                  >
                    恢复默认
                  </el-button>
                  <el-button
                    type="primary"
                    @click="savePasswordDict"
                    :loading="saving"
                  >
                    保存
                  </el-button>
                </div>
              </div>
              <el-input
                v-model="passwordDict"
                type="textarea"
                :rows="15"
                placeholder="每行一个密码"
                style="width: 100%;"
              />
              <div class="dict-info">
                <el-text type="info" size="small">
                  条目数: {{ passwordDict ? passwordDict.split('\n').filter(line => line.trim()).length : 0 }}
                </el-text>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-dialog>

    <!-- 右键菜单 -->
    <Teleport to="body">
      <div
        v-if="rowContextMenuVisible"
        class="context-menu"
        :style="rowContextMenuStyle"
        @click.stop
      >
        <div v-if="canSendToConnect(selectedRow)" class="context-menu-item" @click="handleSendToConnect">
          <el-icon><Promotion /></el-icon>
          <span>发送到简练助手 - {{ getConnectModuleName(selectedRow) }}</span>
        </div>
        <div v-if="canExecuteCommand(selectedRow)" class="context-menu-item" @click="handleExecuteCommand">
          <el-icon><DocumentCopy /></el-icon>
          <span>执行命令</span>
        </div>
        <div v-if="canSendToConnect(selectedRow) || canExecuteCommand(selectedRow)" class="context-menu-divider"></div>
        <div class="context-menu-item" @click="handleCopyRow">
          <el-icon><DocumentCopy /></el-icon>
          <span>复制当前行</span>
        </div>
      </div>
    </Teleport>

    <!-- 命令执行对话框 -->
    <el-dialog
      v-model="commandDialogVisible"
      :title="`执行命令 - ${commandDialogState.protocolDisplay}://${commandDialogState.host}:${commandDialogState.port}`"
      width="930px"
      top="10vh"
      destroy-on-close
      class="command-exec-dialog"
    >
      <div class="command-dialog-container">
        <!-- 顶部信息栏 -->
        <div class="command-dialog-header">
          <div class="target-info-compact">
            <el-tag type="primary" size="small">{{ commandDialogState.protocolDisplay }}</el-tag>
            <span class="target-address">{{ commandDialogState.host }}:{{ commandDialogState.port }}</span>
            <el-divider direction="vertical" />
            <span class="target-auth">{{ commandDialogState.username || '无用户' }} / {{ commandDialogState.password || '无密码' }}</span>
          </div>
          <div class="header-actions">
            <el-button
              :icon="DocumentCopy"
              size="small"
              text
              @click="copyCommandResult"
              :disabled="!commandResult"
            >
              复制结果
            </el-button>
          </div>
        </div>

        <!-- 主要内容区域 -->
        <div class="command-dialog-main">
          <!-- 左侧：命令输入区 -->
          <div class="command-input-section-new">
            <div class="section-header">
              <span class="section-title">命令输入</span>
            </div>
            
            <!-- 常用命令快速选择 -->
            <div class="quick-commands" v-if="currentCommonCommands.length > 0">
              <div class="quick-commands-label">快速选择</div>
              <div class="quick-commands-list">
                <el-tag
                  v-for="item in currentCommonCommands"
                  :key="item.command"
                  :type="selectedCommonCommand === item.command ? 'primary' : 'info'"
                  size="small"
                  class="quick-command-tag"
                  @click="handleQuickCommandClick(item.command)"
                >
                  {{ item.command }}
                </el-tag>
              </div>
            </div>

            <!-- 命令输入框 -->
            <div class="command-input-wrapper">
              <el-input
                v-model="commandInput"
                type="textarea"
                :rows="4"
                :placeholder="getCommandPlaceholder(commandDialogState.protocol)"
                class="command-textarea-new"
                :class="{ 'command-invalid': commandInput && !validateCommand(commandDialogState.protocol, commandInput) }"
              />
              <div 
                v-if="commandInput && !validateCommand(commandDialogState.protocol, commandInput) && hasStrictCommandLimit(commandDialogState.protocol)" 
                class="command-warning-new"
              >
                <el-icon class="warning-icon"><Warning /></el-icon>
                <span class="warning-text">此命令不在支持列表中，可能执行失败</span>
              </div>
              <div 
                v-if="getProtocolCommandHint(commandDialogState.protocol)" 
                class="command-hint"
              >
                <el-icon><InfoFilled /></el-icon>
                <span>{{ getProtocolCommandHint(commandDialogState.protocol) }}</span>
              </div>
            </div>
          </div>

          <!-- 右侧：结果展示区 -->
          <div class="command-result-section-new">
            <div class="section-header">
              <span class="section-title">执行结果</span>
              <el-button
                :icon="DocumentCopy"
                size="small"
                text
                @click="copyCommandResult"
                :disabled="!commandResult"
              >
                复制
              </el-button>
            </div>
            <div class="result-content">
              <VueMonacoEditor
                v-if="commandResult"
                :value="commandResult"
                language="plaintext"
                theme="http-dark-theme"
                :height="resultEditorHeight"
                class="code-editor-new"
                :options="resultMonacoOptions"
              />
              <div v-else class="empty-result-new">
                <el-empty description="命令执行结果将在此显示" :image-size="100" />
              </div>
            </div>
          </div>
        </div>

        <!-- 底部操作栏 -->
        <div class="command-dialog-footer">
          <el-button
            size="default"
            @click="commandDialogVisible = false"
          >
            关闭
          </el-button>
          <el-button
            type="primary"
            size="default"
            :loading="commandExecuting"
            :disabled="!commandInput || !commandInput.trim()"
            @click="doExecuteCommand"
          >
            <el-icon v-if="!commandExecuting"><VideoPlay /></el-icon>
            执行命令
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, onUnmounted, onBeforeUnmount, onActivated, onDeactivated, markRaw } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Download, VideoPlay, VideoPause, Promotion, DocumentCopy, Warning, InfoFilled } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'
import 'monaco-editor/esm/vs/basic-languages/monaco.contribution'
import { GetPortConfig } from '../../../wailsjs/go/controller/Server'
import { getToken } from '@/utils/token'
import {
  NewCrackScanenr,
  LoadUsernameDictFromFile,
  LoadPasswordDictFromFile,
  SaveUsernameDictToFile,
  SavePasswordDictToFile,
  GetAllPasswordCrackResults,
  ExportPasswordCrackResults,
  ExecuteCommandOnConnection,
  CheckPortAlive,
  CheckPortsAlive,
  GetTaskProgress,
  AddPasswordCrackResult,
  ListAvailableUsernameDicts,
  StopCrackTask,
  ResetUsernameDictToDefault,
  ResetPasswordDictToDefault,
  BatchDeletePasswordCrackResults,
  ClearAllPasswordCrackResults
} from '../../../wailsjs/go/crack/CrackController'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime'
import TableColumnFilterHeader from '@/components/TableColumnFilterHeader.vue'

export default {
  name: 'PasswordCrackView',
  components: {
    Refresh,
    Download,
    VideoPlay,
    VideoPause,
    Promotion,
    DocumentCopy: markRaw(DocumentCopy),
    Warning: markRaw(Warning),
    InfoFilled: markRaw(InfoFilled),
    VueMonacoEditor,
    TableColumnFilterHeader
  },
  setup() {
    const loading = ref(false)
    const results = ref([])
    const columnFilters = reactive({
      id: '',
      host: '',
      port: '',
      protocol: '',
      username: '',
      password: '',
      message: '',
      command_result: '',
      created_at: ''
    })
    const crackSort = reactive({ prop: '', order: '' })
    const formatTime = (timeStr) => {
      if (!timeStr) return '-'
      try {
        const date = new Date(timeStr)
        return date.toLocaleString('zh-CN')
      } catch {
        return timeStr
      }
    }
    const filteredCrackResults = computed(() => {
      const rows = results.value
      const entries = Object.entries(columnFilters).filter(([, v]) => (v || '').trim())
      if (!entries.length) return rows
      return rows.filter((row) =>
        entries.every(([key, raw]) => {
          const q = String(raw).trim().toLowerCase()
          if (key === 'created_at') {
            const t1 = String(row.created_at || '').toLowerCase()
            const t2 = formatTime(row.created_at).toLowerCase()
            return t1.includes(q) || t2.includes(q)
          }
          if (key === 'command_result') {
            return String(row.command_result || '').toLowerCase().includes(q)
          }
          const val = row[key]
          return String(val != null ? val : '').toLowerCase().includes(q)
        })
      )
    })
    const currentPage = ref(1)
    const pageSize = ref(20)
    const handleCrackColumnFilter = () => {
      currentPage.value = 1
    }
    const pagedResults = computed(() => {
      const list = filteredCrackResults.value
      const start = (currentPage.value - 1) * pageSize.value
      const end = start + pageSize.value
      return list.slice(start, end)
    })
    const totalResults = computed(() => filteredCrackResults.value.length)
    const handlePageChange = (page) => {
      currentPage.value = page
    }
    const handleSizeChange = (size) => {
      pageSize.value = size
      currentPage.value = 1 // 改变每页条数时重置到第一页
    }
    const cracking = ref(false)
    const crackTaskId = ref(null)
    const startTime = ref(null)
    const elapsedTime = ref(0)
    const elapsedTimer = ref(null)
    const expectedTaskKeys = ref(new Set())
    const expectedTaskList = ref([])
    const isFromHostscan = ref(false) // 标记是否来自hostscan
    
    const targetText = ref()
    
    const dictConfigVisible = ref(false)
    const dictActiveTab = ref('username')
    const usernameDict = ref('')
    const passwordDict = ref('')
    const currentProtocol = ref('')
    const selectedProtocol = ref('')
    const availableProtocols = ref([])
    const saving = ref(false)
    
    // SSH隧道配置
    const sshConfigVisible = ref(false)
    const sshConfig = reactive({
      enabled: false,
      address: '',
      port: 22,
      username: '',
      password: ''
    })
    const sshHostList = ref([])
    const sshApiClient = ref(null)
    
    // 初始化SSH API客户端
    const initSSHApiClient = async () => {
      if (sshApiClient.value) return sshApiClient.value
      
      try {
        const config = await GetPortConfig()
        const port = config?.sshPort || 52868
        const host = '127.0.0.1'
        const baseURL = `http://${host}:${port}`
        
        const client = axios.create({
          baseURL: baseURL,
          timeout: 20000
        })
        
        // 添加请求拦截器，添加token认证
        client.interceptors.request.use(
          (req) => {
            req.headers.Time = String(Date.now())
            const token = getToken()
            if (token) {
              req.headers.Authorization = token
            }
            return req
          },
          (err) => Promise.reject(err)
        )
        
        sshApiClient.value = client
        return client
      } catch (error) {
        console.error('初始化SSH API客户端失败:', error)
        return null
      }
    }
    
    const alivePercent = ref(0)
    const crackPercent = ref(0)
    
    const statistics = reactive({
      aliveCount: 0,
      successCount: 0,
      queueCurrent: 0,
      queueTotal: 0
    })
    
    // 右键菜单相关
    const rowContextMenuVisible = ref(false)
    const rowContextMenuStyle = ref({})
    const selectedRow = ref(null)
    const router = useRouter()

    // 命令执行对话框状态
    const commandDialogVisible = ref(false)
    const commandExecuting = ref(false)
    const commandInput = ref('')
    const commandResult = ref('')
    const selectedCommonCommand = ref('')
    const commandDialogState = reactive({
      id: 0,
      protocol: '',
      protocolDisplay: '',
      host: '',
      port: '',
      username: '',
      password: ''
    })
    
    // Monaco Editor 配置
    const resultMonacoOptions = {
      minimap: { enabled: false },
      wordWrap: 'on',
      fontSize: 13,
      scrollbar: { verticalScrollbarSize: 8 },
      automaticLayout: true,
      scrollBeyondLastLine: false,
      readOnly: true,
      lineNumbers: 'on',
      theme: 'http-dark-theme'
    }
    
    const resultEditorHeight = computed(() => {
      return 'calc(75vh - 280px)'
    })
    
    const services = ref([
      { name: 'SSH', port: 22, checked: false },
      { name: 'RDP', port: 3389, checked: false },
      { name: 'SMB', port: 445, checked: false },
      { name: 'MYSQL', port: 3306, checked: false },
      { name: 'SQLServer', port: 1433, checked: false },
      { name: 'Oracle', port: 1521, checked: false },
      { name: 'MongoDB', port: 27017, checked: false },
      { name: 'Redis', port: 6379, checked: false },
      { name: 'PostgreSQL', port: 5432, checked: false },
      { name: 'DMSQL', port: 5236, checked: false },
      { name: 'MemCached', port: 11211, checked: false },
      { name: 'Elasticsearch', port: 9200, checked: false },
      { name: 'FTP', port: 21, checked: false },
      { name: 'Telnet', port: 23, checked: false },
      { name: 'WinRM', port: 5985, checked: false },
      { name: 'VNC', port: 5900, checked: false },
      { name: 'SVN', port: 3690, checked: false },
      { name: 'Tomcat', port: 8080, checked: false },
      { name: 'WebLogic', port: 7001, checked: false },
      { name: 'Jboss', port: 8081, checked: false },
      { name: 'Zookeeper', port: 2181, checked: false },
      { name: 'ActiveMQ', port: 61613, checked: false },
      { name: 'MQTT', port: 1883, checked: false },
      { name: 'RabbitMQ', port: 5672, checked: false }
    ])
    
    const crackParams = reactive({
      oracleListener: 'orcl',
      command: 'whoami'
    })
    
    const globalSettings = reactive({
      threads: 1,
      timeout: 3000,
      onlyOneAccount: true,
      fingerprint: false,
      checkAlive: false
    })
    
    const protocolMap = {
      'SSH': 'ssh',
      'RDP': 'rdp',
      'SMB': 'smb',
      'MYSQL': 'mysql',
      'SQLServer': 'mssql',
      'Oracle': 'oracle',
      'MongoDB': 'mongodb',
      'Redis': 'redis',
      'PostgreSQL': 'postgresql',
      'DMSQL': 'dm',
      'MemCached': 'memcached',
      'Elasticsearch': 'elasticsearch',
      'FTP': 'ftp',
      'Telnet': 'telnet',
      'WinRM': 'winrm',
      'VNC': 'vnc',
      'SVN': 'svn',
      'Tomcat': 'tomcat',
      'WebLogic': 'weblogic',
      'Jboss': 'jboss',
      'Zookeeper': 'zookeeper',
      'ActiveMQ': 'activemq',
      'MQTT': 'mqtt',
      'RabbitMQ': 'rabbitmq'
    }

    // 确保选中了协议，优先使用已勾选的服务
    const ensureCurrentProtocol = () => {
      if (currentProtocol.value) return true
      const checkedServices = services.value.filter(s => s.checked)
      if (checkedServices.length === 0) {
        ElMessage.warning('请先选择至少一个服务')
        return false
      }
      currentProtocol.value = protocolMap[checkedServices[0].name] || 'ssh'
      return true
    }

    // 各协议常用命令（参考 MPET-main，重点覆盖未在简练助手中实现的 API / 服务）
    const COMMON_COMMANDS = {
      ssh: [
        { label: 'whoami', command: 'whoami' },
        { label: 'pwd', command: 'pwd' },
        { label: 'ls -la', command: 'ls -la' },
        { label: 'uname -a', command: 'uname -a' }
      ],
      redis: [
        { label: 'INFO', command: 'INFO' },
        { label: 'KEYS *', command: 'KEYS *' },
        { label: 'DBSIZE', command: 'DBSIZE' }
      ],
      mysql: [
        { label: 'SHOW DATABASES', command: 'SHOW DATABASES' },
        { label: 'SELECT VERSION()', command: 'SELECT VERSION()' }
      ],
      postgresql: [
        { label: '\\l', command: '\\l' },
        { label: 'SELECT version()', command: 'SELECT version()' }
      ],
      mongodb: [
        { label: 'show dbs', command: 'show dbs' },
        { label: 'db.stats()', command: 'db.stats()' }
      ],
      memcached: [
        { label: 'stats', command: 'stats' },
        { label: 'stats items', command: 'stats items' },
        { label: 'stats slabs', command: 'stats slabs' },
        { label: 'version', command: 'version' }
      ],
      zookeeper: [
        { label: 'stat - 服务器统计信息', command: 'stat' },
        { label: 'srvr - 服务器信息', command: 'srvr' },
        { label: 'mntr - 监控信息', command: 'mntr' },
        { label: 'conf - 配置信息', command: 'conf' },
        { label: 'envi - 环境变量', command: 'envi' },
        { label: 'ruok - 服务器是否运行', command: 'ruok' },
        { label: 'cons - 连接信息', command: 'cons' },
        { label: 'crst - 重置连接统计', command: 'crst' },
        { label: 'dump - 会话和临时节点', command: 'dump' }
      ],
      elasticsearch: [
        { label: 'health - 集群健康', command: 'health' },
        { label: 'stats - 集群统计', command: 'stats' },
        { label: 'nodes - 节点信息', command: 'nodes' },
        { label: 'indices - 索引列表', command: 'indices' },
        { label: 'shards - 分片信息', command: 'shards' },
        { label: 'allocation - 分配信息', command: 'allocation' },
        { label: 'count - 文档计数', command: 'count' },
        { label: 'settings - 集群设置', command: 'settings' },
        { label: 'version - 版本信息', command: 'version' },
        { label: 'tasks - 任务列表', command: 'tasks' },
        { label: 'pending - 待处理任务', command: 'pending' }
      ],
      kafka: [
        { label: 'cluster-info', command: 'cluster-info' },
        { label: 'topics', command: 'topics' },
        { label: 'brokers', command: 'brokers' }
      ],
      etcd: [
        { label: 'status - 状态信息', command: 'status' },
        { label: 'version - 版本信息', command: 'version' },
        { label: 'list - 列出键值', command: 'list' },
        { label: 'get <key> - 获取值', command: 'get /key' },
        { label: 'put <key> <value> - 设置值', command: 'put /key value' },
        { label: 'del <key> - 删除键', command: 'del /key' }
      ],
      docker: [
        { label: 'ps - 容器列表', command: 'ps' },
        { label: 'containers - 容器列表', command: 'containers' },
        { label: 'images - 镜像列表', command: 'images' },
        { label: 'info - 系统信息', command: 'info' },
        { label: 'version - 版本信息', command: 'version' }
      ],
      kubernetes: [
        { label: 'namespaces - 命名空间', command: 'namespaces' },
        { label: 'pods - Pod列表', command: 'pods' },
        { label: 'secrets - Secret列表', command: 'secrets' },
        { label: 'version - 版本信息', command: 'version' }
      ],
      rabbitmq: [
        { label: 'overview - 概览信息', command: 'overview' },
        { label: 'connections - 连接列表', command: 'connections' },
        { label: 'channels - 通道列表', command: 'channels' },
        { label: 'exchanges - 交换机列表', command: 'exchanges' },
        { label: 'queues - 队列列表', command: 'queues' },
        { label: 'nodes - 节点信息', command: 'nodes' }
      ],
      mqtt: [
        { label: 'connack - 连接测试', command: 'connack' },
        { label: 'ping - 保活检测', command: 'ping' },
        { label: 'publish - 发布消息', command: 'publish test/hello hello' },
        { label: 'subscribe - 订阅消息', command: 'subscribe test/hello 5' }
      ],
      wmi: [
        { label: 'nic - 网卡信息', command: 'nic' }
      ],
      rdp: [
        { label: 'screenshot', command: 'screenshot' }
      ],
      vnc: [
        { label: 'screenshot', command: 'screenshot' }
      ]
    }
    
    // 协议支持的命令列表（用于验证和提示）
    const PROTOCOL_VALID_COMMANDS = {
      zookeeper: ['envi', 'conf', 'cons', 'crst', 'dump', 'ruok', 'stat', 'srvr', 'mntr'],
      docker: ['ps', 'containers', 'images', 'info', 'version'],
      kubernetes: ['namespaces', 'ns', 'pods', 'secrets', 'version'],
      k8s: ['namespaces', 'ns', 'pods', 'secrets', 'version'],
      elasticsearch: ['health', 'stats', 'nodes', 'indices', 'shards', 'allocation', 'count', 'settings', 'version', 'tasks', 'pending'],
      rabbitmq: ['overview', 'connections', 'channels', 'exchanges', 'queues', 'nodes'],
      mqtt: ['connack', 'ping', 'publish', 'subscribe'],
      etcd: ['status', 'version', 'list', 'ls', 'get', 'put', 'del', 'delete'],
      wmi: ['nic']
    }
    
    // 获取协议支持的命令提示
    const getProtocolCommandHint = (protocol) => {
      const proto = (protocol || '').toLowerCase()
      const validCommands = PROTOCOL_VALID_COMMANDS[proto]
      if (validCommands && validCommands.length > 0) {
        // 对于 etcd，显示更详细的提示
        if (proto === 'etcd') {
          return `支持的命令: status, version, list [prefix], get <key>, put <key> <value>, del <key>`
        }
        // 对于 kubernetes，显示别名
        if (proto === 'kubernetes' || proto === 'k8s') {
          return `支持的命令: namespaces (或 ns), pods, secrets, version`
        }
        if (proto === 'mqtt') {
          return `支持的命令: connack, ping, publish <topic> [message], subscribe <topic> [seconds]`
        }
        return `支持的命令: ${validCommands.join(', ')}`
      }
      return null
    }
    
    // 验证命令是否有效
    const validateCommand = (protocol, command) => {
      const proto = (protocol || '').toLowerCase()
      const validCommands = PROTOCOL_VALID_COMMANDS[proto]
      if (validCommands && validCommands.length > 0) {
        const cmd = command.trim().toLowerCase()
        // 提取第一个单词（命令名）
        const firstWord = cmd.split(/\s+/)[0]
        return validCommands.includes(firstWord)
      }
      return true // 没有限制的协议返回 true
    }
    
    // 判断协议是否有严格的命令限制（需要显示警告）
    // 对于支持参数的命令（如 etcd 的 get/put），只要命令名正确就不显示警告
    const hasStrictCommandLimit = (protocol) => {
      const proto = (protocol || '').toLowerCase()
      // 这些协议有严格的命令限制，需要显示警告
      const strictProtocols = ['zookeeper', 'wmi']
      return strictProtocols.includes(proto)
    }

    const currentCommonCommands = computed(() => {
      const proto = (commandDialogState.protocol || '').toLowerCase()
      return COMMON_COMMANDS[proto] || []
    })
    
    const refreshTargets = async () => {
      // 刷新时，清除startTime，以便显示所有结果
      startTime.value = null
      await loadResults()
      ElMessage.success('已刷新结果列表')
    }
    
    const openDictConfig = async () => {
      dictConfigVisible.value = true
      // 打开时加载协议列表和字典
      await loadProtocols()
      if (dictActiveTab.value === 'password') {
        await loadPasswordDict()
      }
    }
    
    // 加载可用的协议列表
    const loadProtocols = async () => {
      try {
        const protocols = await ListAvailableUsernameDicts()
        availableProtocols.value = protocols || []
        if (protocols && protocols.length > 0 && !selectedProtocol.value) {
          selectedProtocol.value = protocols[0]
          await loadUsernameDict()
        }
      } catch (error) {
        console.error('加载协议列表失败:', error)
        ElMessage.error('加载协议列表失败')
      }
    }
    
    // 加载用户名字典
    const loadUsernameDict = async () => {
      if (!selectedProtocol.value) {
        usernameDict.value = ''
        return
      }
      try {
        const usernames = await LoadUsernameDictFromFile(selectedProtocol.value)
        usernameDict.value = usernames ? usernames.join('\n') : ''
      } catch (error) {
        console.error('加载用户名字典失败:', error)
        ElMessage.error('加载用户名字典失败')
        usernameDict.value = ''
      }
    }
    
    // 加载密码字典
    const loadPasswordDict = async () => {
      try {
        const passwords = await LoadPasswordDictFromFile()
        passwordDict.value = passwords ? passwords.join('\n') : ''
      } catch (error) {
        console.error('加载密码字典失败:', error)
        ElMessage.error('加载密码字典失败')
        passwordDict.value = ''
      }
    }
    
    // SSH隧道配置相关函数
    const openSSHConfig = async () => {
      sshConfigVisible.value = true
      // 打开时自动加载SSH主机列表
      await loadSSHHostsFromConnect()
    }
    
    // 从简练助手获取SSH主机列表（直接从后端config读取）
    const loadSSHHostsFromConnect = async () => {
      try {
        const client = await initSSHApiClient()
        if (!client) {
          ElMessage.warning('SSH API客户端初始化失败，请手动输入')
          return
        }
        
        const response = await client.get('/api/conn_conf')
        if (response.data.code === 0) {
          const hosts = response.data.data || []
          sshHostList.value = hosts.map(host => ({
            id: host.id || 0,
            name: host.name || '',
            address: host.address || '',
            port: host.port || 22,
            user: host.user || '',
            pwd: host.pwd || '' // 注意：密码可能为空
          }))
          
          if (sshHostList.value.length > 0) {
            ElMessage.success(`已加载 ${sshHostList.value.length} 个SSH主机`)
          } else {
            ElMessage.info('简练助手中暂无SSH主机配置，请手动输入')
          }
        } else {
          ElMessage.warning(response.data.msg || '获取SSH主机列表失败')
        }
      } catch (error) {
        console.error('获取SSH主机列表失败:', error)
        ElMessage.warning('无法获取SSH主机列表，请手动输入')
      }
    }
    
    // 选择SSH主机
    const selectSSHHost = (host) => {
      sshConfig.address = host.address || ''
      sshConfig.port = host.port || 22
      sshConfig.username = host.user || ''
      // 注意：密码通常不会保存在主机配置中，需要手动输入
      if (host.pwd) {
        sshConfig.password = host.pwd
      }
    }
    
    // 保存SSH配置（如果是手动输入的，保存到后端config）
    const saveSSHConfig = async () => {
      try {
        // 如果手动输入的SSH配置不在主机列表中，则添加到后端config
        if (sshConfig.address && sshConfig.username) {
          const existingHost = sshHostList.value.find(
            h => h.address === sshConfig.address && 
                 h.port === sshConfig.port && 
                 h.user === sshConfig.username
          )
          
          if (!existingHost) {
            // 手动输入的新主机，保存到后端config
            const client = await initSSHApiClient()
            if (client) {
              const newHost = {
                name: `密码爆破-${sshConfig.address}:${sshConfig.port}`,
                address: sshConfig.address,
                port: sshConfig.port,
                user: sshConfig.username,
                pwd: sshConfig.password || '',
                auth_type: 'pwd',
                net_type: 'tcp4'
              }
              
              const response = await client.post('/api/conn_conf', newHost)
              if (response.data.code === 0) {
                // 重新加载主机列表
                await loadSSHHostsFromConnect()
                ElMessage.success('SSH配置已保存到简练助手')
              } else {
                ElMessage.warning(response.data.msg || '保存到后端失败')
              }
            }
          }
        }
        
        ElMessage.success('SSH配置已保存')
      } catch (error) {
        console.error('保存SSH配置失败:', error)
        ElMessage.error('保存SSH配置失败: ' + (error.message || '未知错误'))
      }
    }
    
    // 加载SSH配置（保留为空函数用于兼容性，实际不再需要）
    const loadSSHConfig = () => {
      // 配置现在直接从后端config读取，不需要额外的加载逻辑
    }
    
    // 保存用户名字典
    const saveUsernameDict = async () => {
      if (!selectedProtocol.value) {
        ElMessage.warning('请先选择协议')
        return
      }
      saving.value = true
      try {
        const usernames = usernameDict.value
          .split('\n')
          .map(line => line.trim())
          .filter(line => line.length > 0)
        await SaveUsernameDictToFile(selectedProtocol.value, usernames)
        ElMessage.success('用户名字典保存成功')
      } catch (error) {
        console.error('保存用户名字典失败:', error)
        ElMessage.error('保存用户名字典失败: ' + (error.message || error))
      } finally {
        saving.value = false
      }
    }
    
    // 保存密码字典
    const savePasswordDict = async () => {
      saving.value = true
      try {
        const passwords = passwordDict.value
          .split('\n')
          .map(line => line.trim())
          .filter(line => line.length > 0)
        await SavePasswordDictToFile(passwords)
        ElMessage.success('密码字典保存成功')
      } catch (error) {
        console.error('保存密码字典失败:', error)
        ElMessage.error('保存密码字典失败: ' + (error.message || error))
      } finally {
        saving.value = false
      }
    }
    
    // 恢复用户名字典为默认值
    const resetUsernameDict = async () => {
      if (!selectedProtocol.value) {
        ElMessage.warning('请先选择协议')
        return
      }
      
      try {
        await ElMessageBox.confirm(
          `将恢复协议「${selectedProtocol.value}」的用户名字典为默认值，当前内容将被覆盖且不可恢复。`,
          '恢复默认字典',
          { type: 'warning' }
        )
        
        saving.value = true
        await ResetUsernameDictToDefault(selectedProtocol.value)
        // 重新加载字典
        await loadUsernameDict()
        ElMessage.success('用户名字典已恢复为默认值')
      } catch (error) {
        if (error !== 'cancel') {
          console.error('恢复用户名字典失败:', error)
          ElMessage.error('恢复用户名字典失败: ' + (error.message || error))
        }
      } finally {
        saving.value = false
      }
    }
    
    // 恢复密码字典为默认值
    const resetPasswordDict = async () => {
      try {
        await ElMessageBox.confirm(
          '将恢复密码字典为默认值，当前内容将被覆盖且不可恢复。',
          '恢复默认字典',
          { type: 'warning' }
        )
        
        saving.value = true
        await ResetPasswordDictToDefault()
        // 重新加载字典
        await loadPasswordDict()
        ElMessage.success('密码字典已恢复为默认值')
      } catch (error) {
        if (error !== 'cancel') {
          console.error('恢复密码字典失败:', error)
          ElMessage.error('恢复密码字典失败: ' + (error.message || error))
        }
      } finally {
        saving.value = false
      }
    }
    
    const handleServiceChange = () => {
      const checked = services.value.filter(s => s.checked)
      if (checked.length > 0) {
        currentProtocol.value = protocolMap[checked[0].name] || 'ssh'
      } else {
        currentProtocol.value = ''
      }
    }
    
    const handlePortChange = (service) => {
      // 端口变化时的处理
    }
    
    const startCrack = async () => {
      const checkedServices = services.value.filter(s => s.checked)
      if (checkedServices.length === 0) {
        ElMessage.warning('请至少选择一个服务')
        return
      }
      
      const targets = targetText.value.trim().split('\n')
        .filter(line => line.trim().length > 0)
      if (targets.length === 0) {
        ElMessage.warning('请至少输入一个目标')
        return
      }
      
      // 如果不是来自hostscan，且有历史结果，提示用户选择
      let shouldClearResults = false
      if (!isFromHostscan.value && results.value.length > 0) {
        try {
          await ElMessageBox.confirm(
            `当前已有 ${results.value.length} 条历史结果。若点击「清除」将清空上述结果，仅显示本次任务结果；点击「保留」则不清空并追加本次结果。`,
            '清除历史结果',
            {
              confirmButtonText: '清除',
              cancelButtonText: '保留',
              distinguishCancelAndClose: true,
              type: 'warning'
            }
          )
          shouldClearResults = true
        } catch (error) {
          if (error === 'close') {
            return
          }
          shouldClearResults = false
        }
      } else if (isFromHostscan.value) {
        // 来自hostscan，默认追加模式
        shouldClearResults = false
        isFromHostscan.value = false
      } else {
        // 没有历史结果，清除模式
        shouldClearResults = true
      }
      
      try {
        // 存活检测：可选功能，如果启用则提前过滤端口
        let aliveSet = new Set()
        let useAliveFilter = false
        
        // 辅助函数：解析目标地址，如果已包含端口则提取host和port，否则使用service.port
        const parseTarget = (target, servicePort) => {
          const portMatch = target.match(/^(.+):(\d+)$/)
          if (portMatch) {
            // target已经包含端口，使用target中的端口
            return {
              host: portMatch[1],
              port: parseInt(portMatch[2])
            }
          } else {
            // target不包含端口，使用service的默认端口
            return {
              host: target,
              port: servicePort
            }
          }
        }
        
        if (globalSettings.checkAlive) {
          const timeoutMs = globalSettings.timeout || 3000
          const timeoutSeconds = Math.max(1, Math.min(30, Math.ceil(timeoutMs / 1000)))
          const targetPortPairs = []
          checkedServices.forEach(service => {
            targets.forEach(target => {
              const parsed = parseTarget(target, service.port)
              targetPortPairs.push(`${parsed.host}:${parsed.port}`)
            })
          })
          const uniquePairs = Array.from(new Set(targetPortPairs))
          statistics.queueTotal = uniquePairs.length
          statistics.queueCurrent = 0
          
          let aliveList = []
          let detectionFailed = false
          try {
            aliveList = await CheckPortsAlive(uniquePairs, [], timeoutSeconds)
          } catch (err) {
            console.warn('端口存活检测失败，将直接尝试爆破：', err)
            ElMessage.warning('端口存活检测失败，将尝试爆破所有目标')
            detectionFailed = true
          }
          
          aliveSet = new Set((aliveList || []).map(item => (item || '').toLowerCase()))
          statistics.aliveCount = aliveList?.length || 0
          alivePercent.value = uniquePairs.length > 0 ? Math.round((statistics.aliveCount / uniquePairs.length) * 100) : 0
          
          // 如果检测成功但未发现存活端口，直接中止任务
          if (!detectionFailed && aliveSet.size === 0) {
            ElMessage.error('未检测到存活端口，任务已中止。请检查目标地址和端口是否正确')
            return
          }
          
          useAliveFilter = !detectionFailed && aliveSet.size > 0
        } else {
          // 未启用存活检测，统计所有目标，默认全部存活
          const targetPortPairs = []
          checkedServices.forEach(service => {
            targets.forEach(target => {
              const parsed = parseTarget(target, service.port)
              targetPortPairs.push(`${parsed.host}:${parsed.port}`)
            })
          })
          const uniquePairs = Array.from(new Set(targetPortPairs))
          statistics.queueTotal = uniquePairs.length
          statistics.queueCurrent = 0
          // 未启用存活检测时，默认所有目标都存活
          statistics.aliveCount = uniquePairs.length
          alivePercent.value = 100
          // 将所有目标添加到存活集合
          uniquePairs.forEach(pair => {
            aliveSet.add(pair.toLowerCase())
          })
          useAliveFilter = false // 不使用过滤，因为默认全部存活
        }

        // 加载密码字典：优先使用用户编辑的字典，如果为空则从文件加载
        let passwords = []
        if (passwordDict.value && passwordDict.value.trim()) {
          // 使用用户编辑的密码字典
          passwords = passwordDict.value.split('\n')
            .map(p => p.trim())
            .filter(p => p.length > 0)
        } else {
          // 如果用户没有编辑字典，从文件加载
          const filePasswords = await LoadPasswordDictFromFile()
          if (Array.isArray(filePasswords)) {
            passwords = filePasswords
          }
        }
        if (!passwords || passwords.length === 0) {
          ElMessage.warning('密码字典为空，请先配置字典')
          return
        }
        // 去重
        passwords = Array.from(new Set(passwords))
        
        // 如果选择清除，先清空结果（真正删除数据库记录并重置ID）
        if (shouldClearResults) {
          try {
            // 使用新的清空方法，会删除所有记录并重置ID
            await ClearAllPasswordCrackResults()
            // console.log('[密码爆破] startCrack: 已清除所有历史记录并重置ID')
            
            // 清空前端显示
            results.value = []
            statistics.successCount = 0
            statistics.aliveCount = 0
            statistics.queueCurrent = 0
            statistics.queueTotal = 0
          } catch (error) {
            // console.error('[密码爆破] startCrack: 清除历史记录失败:', error)
            ElMessage.warning('清除历史记录失败，将继续执行任务')
          }
        }
        
        cracking.value = true
        crackTaskId.value = `crack-${Date.now()}`
        // 记录任务开始时间，用于过滤只显示本次任务的结果
        // 无论是否清除，都要设置startTime，以便正确过滤新结果
        // 减去1秒，确保包含任务开始时的结果
        startTime.value = Date.now() - 1000
        // console.log('[密码爆破] startCrack: 任务开始时间设置为:', new Date(startTime.value).toLocaleString())
        elapsedTime.value = 0
        expectedTaskKeys.value = new Set()
        expectedTaskList.value = []
        
        elapsedTimer.value = setInterval(() => {
          if (startTime.value) {
            elapsedTime.value = Math.floor((Date.now() - startTime.value) / 1000)
          }
        }, 1000)
        
        crackPercent.value = 0
        
        // 为每个选中的服务和目标启动破解任务
        // 优化：根据协议选择对应字典，多IP多协议遍历循环
        let totalTasks = 0
        
        // 按协议分组，为每个协议加载对应的用户名字典
        const protocolUsernameCache = {}
        
        for (const service of checkedServices) {
          const protocol = protocolMap[service.name] || service.name.toLowerCase()
          
          // 为每个协议加载对应的用户名字典（只加载一次）
          if (!protocolUsernameCache[protocol]) {
            let usernames = []
            if (usernameDict.value && usernameDict.value.trim()) {
              // 使用用户编辑的用户名字典
              usernames = usernameDict.value.split('\n')
                .map(u => u.trim())
                .filter(u => u.length > 0)
            } else {
              // 如果用户没有编辑字典，从文件加载该协议的用户名字典
              try {
                const builtinUsers = await LoadUsernameDictFromFile(protocol)
                if (Array.isArray(builtinUsers)) {
                  usernames.push(...builtinUsers)
                }
              } catch (e) {
                console.warn(`加载协议 ${protocol} 用户名字典失败:`, e)
              }
            }
            
            // 如果仍然为空，使用默认用户名
            if (usernames.length === 0) {
              usernames = ['admin', 'root', 'administrator', 'test', 'user']
            }
            // 去重
            protocolUsernameCache[protocol] = Array.from(new Set(usernames))
          }
          
          const usernames = protocolUsernameCache[protocol]
          
          // 遍历所有目标
          for (const target of targets) {
            // 解析目标地址，如果已包含端口则使用该端口，否则使用service的默认端口
            const parsed = parseTarget(target, service.port)
            const finalTarget = parsed.host
            const finalPort = parsed.port
            
            const targetKey = `${finalTarget}:${finalPort}`.toLowerCase()
            if (useAliveFilter && !aliveSet.has(targetKey)) {
              continue
            }
            totalTasks++
            const targetUrl = `${protocol}://${finalTarget}:${finalPort}`
            const expectedKey = `${protocol.toLowerCase()}://${finalTarget}:${finalPort}`
            expectedTaskList.value.push(expectedKey)
            expectedTaskKeys.value.add(expectedKey)
            
            try {
              // SSH隧道配置
              let sshConfigJSON = ''
              if (sshConfig.enabled && sshConfig.address && sshConfig.username && sshConfig.password) {
                sshConfigJSON = JSON.stringify({
                  enabled: true,
                  address: sshConfig.address,
                  port: String(sshConfig.port || 22),
                  username: sshConfig.username,
                  password: sshConfig.password
                })
              }
              
              // 爆破配置
              const crackConfigJSON = JSON.stringify({
                threads: globalSettings.threads || 5,
                timeout: globalSettings.timeout || 3000,
                only_one_account: globalSettings.onlyOneAccount !== false
              })
              
              await NewCrackScanenr(crackTaskId.value, targetUrl, usernames, passwords, sshConfigJSON, crackConfigJSON)
            } catch (error) {
              console.error(`启动破解任务失败: ${targetUrl}`, error)
            }
          }
        }
        
        statistics.queueTotal = expectedTaskKeys.value.size
        statistics.queueCurrent = 0
        ElMessage.success(`已启动 ${totalTasks} 个破解任务`)
        
        // 开始轮询结果
        pollResults()
      } catch (error) {
        ElMessage.error('启动破解失败: ' + error)
        cracking.value = false
        if (elapsedTimer.value) {
          clearInterval(elapsedTimer.value)
          elapsedTimer.value = null
        }
      }
    }
    
    const stopCrack = async () => {
      if (!crackTaskId.value) {
        ElMessage.warning('没有正在运行的任务')
        return
      }
      
      try {
        await StopCrackTask(crackTaskId.value)
        cracking.value = false
        // 停止任务后，保持startTime不变，继续只显示本次任务的结果
        // 如果用户想查看所有结果，可以点击刷新按钮
        if (elapsedTimer.value) {
          clearInterval(elapsedTimer.value)
          elapsedTimer.value = null
        }
        // 停止时不清零进度，保持当前进度显示
        ElMessage.success('已停止破解任务')
        // 不重新加载所有结果，保持当前过滤状态
      } catch (error) {
        console.error('停止任务失败:', error)
        ElMessage.error('停止任务失败: ' + (error.message || error))
      }
    }
    
    const pollResults = async () => {
      if (!cracking.value) return
      
      try {
        // 加载所有结果
        const allResults = await GetAllPasswordCrackResults()
        // console.log('[密码爆破] pollResults: 从数据库加载了', allResults.length, '条结果')
        
        // 如果正在执行新任务，只显示任务开始后的新结果
        if (startTime.value && startTime.value > 0) {
          const taskStartTime = startTime.value
          // console.log('[密码爆破] pollResults: 任务开始时间:', new Date(taskStartTime).toLocaleString())
          
          // 只显示任务开始时间之后的结果（本次任务的新结果）
          const newResults = allResults.filter(result => {
            if (!result.created_at) {
              // console.log('[密码爆破] pollResults: 结果缺少created_at:', result)
              return false
            }
            const resultTime = new Date(result.created_at).getTime()
            if (isNaN(resultTime)) {
              // console.warn('[密码爆破] pollResults: 无法解析时间:', result.created_at, result)
              // 如果无法解析时间，默认显示（可能是新结果）
              return true
            }
            const isNew = resultTime >= taskStartTime
            if (!isNew) {
              console.log('[密码爆破] pollResults: 结果被过滤 (结果时间早于任务开始时间):', {
                created_at: result.created_at,
                resultTime: new Date(resultTime).toLocaleString(),
                resultTimeMs: resultTime,
                taskStartTime: new Date(taskStartTime).toLocaleString(),
                taskStartTimeMs: taskStartTime,
                diff: taskStartTime - resultTime,
                result: { id: result.id, protocol: result.protocol, host: result.host, port: result.port }
              })
            }
            return isNew
          })
          
          // console.log('[密码爆破] pollResults: 过滤后的新结果数量:', newResults.length, '总结果数:', allResults.length)
          
          // 检查是否是追加模式（保留历史结果）
          // 如果当前results中有时间早于startTime的记录，说明是追加模式
          const hasOldResults = results.value.some(r => {
            if (!r.created_at) return false
            const resultTime = new Date(r.created_at).getTime()
            return resultTime < taskStartTime
          })
          
          if (hasOldResults) {
            // 追加模式：合并历史结果和新结果，去重
            const existingIds = new Set(results.value.map(r => r.id))
            const additionalResults = newResults.filter(r => !existingIds.has(r.id))
            results.value = [...results.value, ...additionalResults]
            // console.log('[密码爆破] pollResults: 追加模式，新增', additionalResults.length, '条结果')
          } else {
            // 清除模式：只显示新结果
            results.value = newResults
            // console.log('[密码爆破] pollResults: 清除模式，显示', newResults.length, '条结果')
          }
        } else {
          // 没有设置开始时间，显示所有结果（不应该发生，但作为兜底）
          results.value = allResults
          // console.log('[密码爆破] pollResults: 无startTime，显示所有', allResults.length, '条结果')
        }
        
        // 重置分页，保持在第一页展示最新
        currentPage.value = 1
        // console.log('[密码爆破] pollResults: 最终结果数量:', results.value.length)
        
        // 从后端获取任务进度
        if (crackTaskId.value) {
          try {
            const progress = await GetTaskProgress(crackTaskId.value)
            if (progress) {
              statistics.queueTotal = progress.totalTargets || 0
              statistics.queueCurrent = progress.currentTarget || 0
              
              // 如果任务已完成，更新状态
              if (progress.status === 'completed') {
                cracking.value = false
                crackPercent.value = 100
                // 任务完成后，保持startTime不变，继续只显示本次任务的结果
                ElMessage.success('密码爆破任务已完成')
                return
              }
              
              // 计算进度百分比
              if (statistics.queueTotal > 0) {
                const percent = Math.round((statistics.queueCurrent / statistics.queueTotal) * 100)
                if (cracking.value) {
                  // 进行中时：如果进度达到100%，说明任务完成
                  if (percent >= 100) {
                    cracking.value = false
                    crackPercent.value = 100
                    // 任务完成后，保持startTime不变，继续只显示本次任务的结果
                    ElMessage.success('密码爆破任务已完成')
                    if (elapsedTimer.value) {
                      clearInterval(elapsedTimer.value)
                      elapsedTimer.value = null
                    }
                    return
                  }
                  crackPercent.value = Math.max(1, Math.min(99, percent)) // 进行中时保持 1-99 的进度反馈
                } else {
                  crackPercent.value = Math.min(100, percent)
                }
              } else {
                crackPercent.value = 0
              }
            }
          } catch (error) {
            // 如果后端还没有任务进度信息，使用原来的计算方式
            console.warn('获取任务进度失败，使用备用计算方式:', error)
            const expectedKeys = expectedTaskKeys.value || new Set()
            const completedKeys = new Set()

            results.value.forEach(row => {
              const proto = (row.protocol || '').toLowerCase()
              const host = (row.host || row.target || '').toLowerCase()
              const port = row.port || row.Port
              if (!proto || !host || !port) return
              const key = `${proto}://${host}:${port}`
              if (expectedKeys.has(key)) {
                completedKeys.add(key)
              }
            })

            // 如果没有设置queueTotal，使用预期任务数量
            if (statistics.queueTotal === 0 && expectedKeys.size > 0) {
              statistics.queueTotal = expectedKeys.size
            }

            statistics.queueCurrent = completedKeys.size > 0
              ? completedKeys.size
              : Math.min(results.value.length, statistics.queueTotal || results.value.length)

            if (statistics.queueTotal > 0) {
              const percent = Math.round((statistics.queueCurrent / statistics.queueTotal) * 100)
              if (cracking.value) {
                // 进行中时：如果进度达到100%，说明任务完成
                if (percent >= 100) {
                  cracking.value = false
                  crackPercent.value = 100
                  // 任务完成后，保持startTime不变，继续只显示本次任务的结果
                  ElMessage.success('密码爆破任务已完成')
                  if (elapsedTimer.value) {
                    clearInterval(elapsedTimer.value)
                    elapsedTimer.value = null
                  }
                  return
                }
                crackPercent.value = Math.max(1, Math.min(99, percent))
              } else {
                crackPercent.value = Math.min(100, percent)
              }
            } else {
              crackPercent.value = 0
            }
          }
        }

        statistics.successCount = results.value.filter(r => r.status === 'success').length
        // 未启用存活检测时，保持存活进度为100%
        if (!globalSettings.checkAlive) {
          // 如果queueTotal已设置，确保aliveCount与之保持一致
          if (statistics.queueTotal > 0 && statistics.aliveCount !== statistics.queueTotal) {
            statistics.aliveCount = statistics.queueTotal
            alivePercent.value = 100
          }
        }

        // 自动完成：所有协议/目标任务都有结果
        if (statistics.queueTotal > 0 && statistics.queueCurrent >= statistics.queueTotal) {
          cracking.value = false
          crackPercent.value = 100
          // 任务完成后，保持startTime不变，继续只显示本次任务的结果
          ElMessage.success('密码爆破任务已完成')
          if (elapsedTimer.value) {
            clearInterval(elapsedTimer.value)
            elapsedTimer.value = null
          }
          return
        }
        
        // 如果没有任务进度信息，尝试通过结果数量判断是否完成
        if (statistics.queueTotal === 0 && expectedTaskKeys.value.size > 0) {
          // 检查是否有足够的结果覆盖所有预期任务
          const expectedKeys = expectedTaskKeys.value || new Set()
          const completedKeys = new Set()
          results.value.forEach(row => {
            const proto = (row.protocol || '').toLowerCase()
            const host = (row.host || row.target || '').toLowerCase()
            const port = row.port || row.Port
            if (!proto || !host || !port) return
            const key = `${proto}://${host}:${port}`
            if (expectedKeys.has(key)) {
              completedKeys.add(key)
            }
          })
          
          // 如果所有预期任务都有结果，认为任务完成
          if (completedKeys.size >= expectedKeys.size) {
            cracking.value = false
            crackPercent.value = 100
            statistics.queueCurrent = completedKeys.size
            statistics.queueTotal = expectedKeys.size
            // 任务完成后，保持startTime不变，继续只显示本次任务的结果
            ElMessage.success('密码爆破任务已完成')
            if (elapsedTimer.value) {
              clearInterval(elapsedTimer.value)
              elapsedTimer.value = null
            }
            return
          }
        }
        
        setTimeout(pollResults, 2000)
      } catch (error) {
        console.error('轮询结果失败:', error)
        setTimeout(pollResults, 5000)
      }
    }
    
    const loadResults = async () => {
      loading.value = true
      try {
        const allResults = await GetAllPasswordCrackResults()
        // console.log('[密码爆破] loadResults: 从数据库加载了', allResults.length, '条结果')
        
        // 如果有startTime，只显示本次任务的结果（无论任务是否完成）
        if (startTime.value && startTime.value > 0) {
          const taskStartTime = startTime.value
          // console.log('[密码爆破] loadResults: 任务开始时间:', new Date(taskStartTime).toLocaleString())
          
          // 检查是否是追加模式（保留历史结果）
          const hasOldResults = results.value.some(r => {
            if (!r.created_at) return false
            const resultTime = new Date(r.created_at).getTime()
            return resultTime < taskStartTime
          })
          
          const newResults = allResults.filter(result => {
            if (!result.created_at) {
              // console.log('[密码爆破] loadResults: 结果缺少created_at:', result)
              return false
            }
            const resultTime = new Date(result.created_at).getTime()
            if (isNaN(resultTime)) {
              // console.warn('[密码爆破] loadResults: 无法解析时间:', result.created_at, result)
              // 如果无法解析时间，默认显示（可能是新结果）
              return true
            }
            const isNew = resultTime >= taskStartTime
            if (!isNew) {
              console.log('[密码爆破] loadResults: 结果被过滤 (结果时间早于任务开始时间):', {
                created_at: result.created_at,
                resultTime: new Date(resultTime).toLocaleString(),
                resultTimeMs: resultTime,
                taskStartTime: new Date(taskStartTime).toLocaleString(),
                taskStartTimeMs: taskStartTime,
                diff: taskStartTime - resultTime,
                result: { id: result.id, protocol: result.protocol, host: result.host, port: result.port }
              })
            } else {
              // console.log('[密码爆破] loadResults: 结果通过过滤:', {
              //   created_at: result.created_at,
              //   resultTime: new Date(resultTime).toLocaleString(),
              //   result: { id: result.id, protocol: result.protocol, host: result.host, port: result.port }
              // })
            }
            return isNew
          })
          
          // console.log('[密码爆破] loadResults: 过滤后的新结果数量:', newResults.length, '总结果数:', allResults.length)
          
          if (hasOldResults) {
            // 追加模式：合并历史结果和新结果，去重
            const existingIds = new Set(results.value.map(r => r.id))
            const additionalResults = newResults.filter(r => !existingIds.has(r.id))
            results.value = [...results.value, ...additionalResults]
            // console.log('[密码爆破] loadResults: 追加模式，新增', additionalResults.length, '条结果')
          } else {
            // 清除模式：只显示新结果
            results.value = newResults
            // console.log('[密码爆破] loadResults: 清除模式，显示', newResults.length, '条结果')
          }
        } else {
          // 没有设置开始时间，显示所有结果
          results.value = allResults
        }
        // 重置分页，保持在第一页展示最新
        currentPage.value = 1
      } catch (error) {
        ElMessage.error('加载结果失败: ' + error)
      } finally {
        loading.value = false
      }
    }
    
    const handleExport = async () => {
      try {
        const filepath = await ExportPasswordCrackResults('all')
        ElMessage.success(`导出成功: ${filepath}`)
      } catch (error) {
        ElMessage.error('导出失败: ' + error)
      }
    }
    
    const handleClearResults = async () => {
      // 先加载所有结果，检查是否有数据
      let allResults = []
      try {
        allResults = await GetAllPasswordCrackResults()
      } catch (error) {
        console.error('加载结果失败:', error)
      }
      
      if (allResults.length === 0 && results.value.length === 0) {
        ElMessage.info('结果列表已为空')
        return
      }
      
      try {
        await ElMessageBox.confirm(
          '确定清空所有结果？此操作不可恢复，ID 将从 1 重新开始。',
          '清空结果',
          { type: 'warning' }
        )
        try {
          await ClearAllPasswordCrackResults()
          results.value = []
          Object.keys(columnFilters).forEach((k) => {
            columnFilters[k] = ''
          })
          crackSort.prop = ''
          crackSort.order = ''
          statistics.successCount = 0
          statistics.aliveCount = 0
          statistics.queueCurrent = 0
          statistics.queueTotal = 0
          startTime.value = null
          currentPage.value = 1
          ElMessage.success(`已清空所有结果，ID已重置`)
        } catch (error) {
          console.error('清空结果失败:', error)
          ElMessage.error('清空结果失败: ' + (error.message || error))
        }
      } catch {
        /* 用户取消 */
      }
    }
    
    const handleSortChange = ({ prop, order }) => {
      if (order && prop) {
        results.value.sort((a, b) => {
          const aVal = a[prop]
          const bVal = b[prop]
          if (order === 'ascending') {
            return aVal > bVal ? 1 : -1
          }
          return aVal < bVal ? 1 : -1
        })
        crackSort.prop = prop
        crackSort.order = order
      } else {
        crackSort.prop = ''
        crackSort.order = ''
      }
    }

    const handleCrackColSort = (prop, order) => {
      handleSortChange({ prop, order })
    }
    
    // 右键菜单相关函数
    const handleRowContextMenu = (row, column, event) => {
      event.preventDefault()
      event.stopPropagation()
      selectedRow.value = row
      
      const x = event.clientX || 0
      const y = event.clientY || 0
      
      rowContextMenuStyle.value = {
        position: 'fixed',
        left: `${x}px`,
        top: `${y}px`,
        zIndex: 9999
      }
      
      rowContextMenuVisible.value = true
      
      nextTick(() => {
        const menuEl = document.querySelector('.context-menu')
        if (!menuEl) return
        
        const menuHeight = menuEl.offsetHeight || 0
        const windowHeight = window.innerHeight
        if (y + menuHeight > windowHeight - 8) {
          let adjustedY = windowHeight - menuHeight - 8
          if (adjustedY < 8) adjustedY = 8
          rowContextMenuStyle.value = {
            ...rowContextMenuStyle.value,
            top: `${adjustedY}px`
          }
        }
      })
      
      // 点击其他地方关闭菜单
      const closeMenu = (e) => {
        if (!e.target.closest('.context-menu')) {
          rowContextMenuVisible.value = false
          document.removeEventListener('click', closeMenu)
          document.removeEventListener('contextmenu', closeMenu)
        }
      }
      setTimeout(() => {
        document.addEventListener('click', closeMenu)
        document.addEventListener('contextmenu', closeMenu)
      }, 100)
    }
    
    // 判断是否可以发送到简练助手
    const canSendToConnect = (row) => {
      if (!row || !row.protocol) return false
      const protocol = (row.protocol || '').toLowerCase()
      // 简练助手支持的协议：SSH、Redis、FTP、MySQL、PostgreSQL、MongoDB、MSSQL、Oracle等
      const supportedProtocols = [
        'ssh', 'redis', 'ftp',
        'mysql', 'postgresql', 'mongodb', 'mssql', 'oracle'
      ]
      // 需要有密码字段（可以是空字符串或"未授权访问"，发送时会自动转换为空字符串）
      const hasPassword = row.password && row.password.trim() !== ''
      return supportedProtocols.includes(protocol) && hasPassword
    }
    
    // 获取连接模块名称
    const getConnectModuleName = (row) => {
      if (!row || !row.protocol) return ''
      const protocol = (row.protocol || '').toLowerCase()
      const nameMap = {
        'ssh': 'SSH',
        'redis': 'Redis',
        'ftp': 'FTP',
        'mysql': 'MySQL',
        'postgresql': 'PostgreSQL',
        'mongodb': 'MongoDB',
        'mssql': 'SQLServer',
        'oracle': 'Oracle'
      }
      return nameMap[protocol] || ''
    }
    
    // 判断是否可以执行命令（重点针对未在简练助手里实现的协议，包括 Docker / K8s / 各类 API 服务）
    const canExecuteCommand = (row) => {
      if (!row || !row.protocol) return false
      const protocol = (row.protocol || '').toLowerCase()
      // 简练助手已经实现的协议：交给"发送到简练助手"，这里不再弹命令对话框
      const connectProtocols = ['ssh', 'redis', 'ftp', 'mysql', 'postgresql', 'mongodb', 'mssql', 'oracle']
      // 这里列出希望在本页支持"执行命令"的协议（包含 API / 服务类，如 docker、k8s 等）
      const allSupported = [
        'smb',
        'telnet',
        'winrm',
        'rdp',
        'vnc',
        'memcached',
        'zookeeper',
        'elasticsearch',
        'kafka',
        'etcd',
        'docker',
        'kubernetes',
        'activemq',
        'mqtt',
        'rsync',
        'rmi',
        'jdwp',
        'adb',
        'socks5'
      ]
      // 对于未授权访问的情况（password为"未授权访问"），也应该允许执行命令
      const hasPassword = row.password && row.password.trim() !== ''
      return allSupported.includes(protocol) && !connectProtocols.includes(protocol) && hasPassword
    }
    
    // 发送到简练助手
    const handleSendToConnect = () => {
      rowContextMenuVisible.value = false
      
      if (!selectedRow.value || !canSendToConnect(selectedRow.value)) {
        ElMessage.warning('无法发送到简练助手')
        return
      }
      
      const row = selectedRow.value
      const protocol = (row.protocol || '').toLowerCase()
      
      // 提取连接信息
      // 对于未授权访问的情况，password应该是空字符串，而不是"未授权访问"
      let password = row.password || ''
      if (password === '未授权访问') {
        password = ''
      }
      
      const connectionInfo = {
        host: row.host,
        port: row.port || '',
        username: row.username || '',
        password: password,
        name: `${getConnectModuleName(row)} - ${row.host}:${row.port}`
      }
      
      // 跳转到简练助手页面
      router.push('/Connect').then(() => {
        setTimeout(() => {
          // 触发全局事件，通知简练助手模块打开对话框并填充数据
          window.dispatchEvent(new CustomEvent('open-connect-dialog', {
            detail: {
              module: protocol,
              connectionInfo: connectionInfo
            }
          }))
          ElMessage.success(`已发送到${getConnectModuleName(row)}模块`)
        }, 300)
      }).catch(err => {
        console.error('跳转失败:', err)
        ElMessage.error('跳转到简练助手失败')
      })
    }
    
    // 执行命令（弹出对话框）
    const handleExecuteCommand = () => {
      rowContextMenuVisible.value = false
      
      if (!selectedRow.value || !canExecuteCommand(selectedRow.value)) {
        ElMessage.warning('当前协议不支持执行命令')
        return
      }
      
      const row = selectedRow.value

      commandDialogState.id = row.id || 0
      commandDialogState.protocol = (row.protocol || '').toLowerCase()
      commandDialogState.protocolDisplay = row.protocol || commandDialogState.protocol.toUpperCase()
      commandDialogState.host = row.host || ''
      commandDialogState.port = String(row.port || '')
      commandDialogState.username = row.username || ''
      commandDialogState.password = row.password || ''

      const proto = commandDialogState.protocol
      const commonList = COMMON_COMMANDS[proto] || []
      if (commonList.length > 0) {
        selectedCommonCommand.value = commonList[0].command
        commandInput.value = commonList[0].command
      } else {
        selectedCommonCommand.value = ''
        commandInput.value = crackParams.command || 'whoami'
      }

      commandResult.value = row.command_result || ''
      commandExecuting.value = false

      commandDialogVisible.value = true
    }

    const handleCommonCommandChange = (cmd) => {
      if (cmd) {
        commandInput.value = cmd
      }
    }
    
    // 快速命令点击
    const handleQuickCommandClick = (cmd) => {
      commandInput.value = cmd
      selectedCommonCommand.value = cmd
    }

    // 获取命令输入框的占位符
    const getCommandPlaceholder = (protocol) => {
      const proto = (protocol || '').toLowerCase()
      const placeholders = {
        zookeeper: '输入四字命令，例如：stat, srvr, mntr, conf, envi, ruok',
        docker: '输入命令，例如：ps, images, info, version',
        kubernetes: '输入命令，例如：namespaces, pods, secrets, version',
        k8s: '输入命令，例如：namespaces, pods, secrets, version',
        elasticsearch: '输入命令，例如：health, stats, nodes, indices',
        rabbitmq: '输入命令，例如：overview, connections, queues, nodes',
        mqtt: '输入命令，例如：connack, ping, publish test/hello hello, subscribe test/hello 5',
        etcd: '输入命令，例如：status, version, list, get /key, put /key value',
        wmi: '输入命令，例如：nic'
      }
      return placeholders[proto] || '输入要执行的命令'
    }
    
      const doExecuteCommand = async () => {
      const id = commandDialogState.id
      const cmd = (commandInput.value || '').trim()
      if (!id || !cmd) {
        ElMessage.warning('缺少必要信息，无法执行命令')
        return
      }
      
      // 仅对严格限制的协议进行验证并阻止执行
      // 其他协议允许自定义命令，由后端验证
      if (hasStrictCommandLimit(commandDialogState.protocol) && !validateCommand(commandDialogState.protocol, cmd)) {
        const hint = getProtocolCommandHint(commandDialogState.protocol)
        ElMessage.warning(hint || '命令格式不正确，请使用支持的命令')
        return
      }

      // 准备SSH隧道配置（如果启用）
      let sshConfigJSON = ''
      if (sshConfig.enabled && sshConfig.address && sshConfig.username && sshConfig.password) {
        sshConfigJSON = JSON.stringify({
          enabled: true,
          address: sshConfig.address,
          port: String(sshConfig.port || 22),
          username: sshConfig.username,
          password: sshConfig.password
        })
      }

      try {
        commandExecuting.value = true
        const result = await ExecuteCommandOnConnection(id, cmd, sshConfigJSON)
        commandResult.value = result || '(命令执行成功，无输出)'

        const idx = results.value.findIndex(r => r.id === id)
        if (idx !== -1) {
          results.value[idx].command_result = commandResult.value
        }

        ElMessage.success('命令执行成功')
      } catch (error) {
        const msg = typeof error === 'string' ? error : (error?.message || String(error))
        ElMessage.error('命令执行失败: ' + msg)
        // 如果是命令不支持的错误，显示更友好的提示
        if (msg.includes('不支持的命令')) {
          const hint = getProtocolCommandHint(commandDialogState.protocol)
          if (hint) {
            ElMessage.info(hint)
          }
        }
      } finally {
        commandExecuting.value = false
      }
    }
    
    // 复制当前行
    const handleCopyRow = () => {
      rowContextMenuVisible.value = false
      
      if (!selectedRow.value) return
      
      const row = selectedRow.value
      const text = `协议: ${row.protocol}\n主机: ${row.host}\n端口: ${row.port}\n用户名: ${row.username || ''}\n密码: ${row.password || ''}\nBanner: ${row.message || ''}`
      
      // 使用 Clipboard API
      if (navigator.clipboard) {
        navigator.clipboard.writeText(text).then(() => {
          ElMessage.success('已复制到剪贴板')
        }).catch(() => {
          ElMessage.error('复制失败')
        })
      } else {
        // 降级方案
        const textarea = document.createElement('textarea')
        textarea.value = text
        textarea.style.position = 'fixed'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.select()
        try {
          document.execCommand('copy')
          ElMessage.success('已复制到剪贴板')
        } catch (e) {
          ElMessage.error('复制失败')
        }
        document.body.removeChild(textarea)
      }
    }
    
    // 复制命令执行结果
    const copyCommandResult = () => {
      if (!commandResult.value) {
        ElMessage.warning('无执行结果可复制')
        return
      }
      
      if (navigator.clipboard) {
        navigator.clipboard.writeText(commandResult.value).then(() => {
          ElMessage.success('执行结果已复制到剪贴板')
        }).catch(() => {
          ElMessage.error('复制失败')
        })
      } else {
        const textarea = document.createElement('textarea')
        textarea.value = commandResult.value
        textarea.style.position = 'fixed'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.select()
        try {
          document.execCommand('copy')
          ElMessage.success('执行结果已复制到剪贴板')
        } catch (e) {
          ElMessage.error('复制失败')
        }
        document.body.removeChild(textarea)
      }
    }
    
    // 存活检测函数（通过后端API检测端口是否开放）
    const checkPortAlive = async (host, port, timeout = 3000) => {
      try {
        const timeoutSeconds = Math.max(1, Math.min(30, Math.ceil(timeout / 1000)))
        return await CheckPortAlive(host, Number(port), timeoutSeconds)
      } catch (error) {
        console.warn('端口存活检测失败，默认返回存活以避免阻塞: ', error)
        return true
      }
    }
    
    onMounted(() => {
      loadResults()
      
      // 监听密码爆破结果事件
      const handleCrackResult = async (result) => {
        try {
          // console.log('[密码爆破] 收到crackResult事件:', result)
          
          if (!result || !result.protocol || !result.host) {
            // console.warn('[密码爆破] 收到无效的结果事件:', result)
            return
          }
          
          const protocolConnection = {
            protocol: result.protocol || '',
            host: result.host || '',
            port: result.port || '',
            username: result.username || '',
            password: result.password || '',
            status: result.status || 'success',
            message: result.message || '密码爆破成功',
            credential: result.credential || (result.username && result.password ? `${result.username}/${result.password}` : ''),
            vuln_url: result.vuln_url || (result.protocol && result.host && result.port ? `${result.protocol.toLowerCase()}://${result.host}:${result.port}` : ''),
            command_result: result.command_result || ''
          }
          
          // console.log('[密码爆破] 准备保存结果:', protocolConnection)
          
          const connectionJSON = JSON.stringify(protocolConnection)
          await AddPasswordCrackResult(connectionJSON)
          
          // console.log('[密码爆破] 结果已保存到数据库，开始刷新列表')
          
          // 重新加载结果列表
          await loadResults()
          
          // console.log('[密码爆破] 已保存结果并刷新列表:', protocolConnection)
        } catch (error) {
          console.error('[密码爆破] 保存结果失败:', error)
        }
      }
      
      EventsOn('crackResult', handleCrackResult)
      // console.log('[密码爆破] 已注册crackResult事件监听器')
      
      // 监听来自主机扫描模块的发送事件
      const handleReceiveCrackTargets = (event) => {
        const data = event.detail || {}
        if (data && data.targets && Array.isArray(data.targets)) {
          // 标记为来自hostscan
          isFromHostscan.value = true
          
          // 如果正在执行任务，不自动填充，避免干扰当前任务
          if (cracking.value) {
            ElMessage.info('当前有任务正在执行，已接收主机扫描目标但未自动配置。任务完成后可手动配置。')
            return
          }
          
          // 解析目标并自动勾选协议和端口
          const targetLines = []
          const protocolMap = {}
          
          // 兼容显式传入的主机列表
          if (Array.isArray(data.hosts)) {
            data.hosts.forEach(h => {
              if (typeof h === 'string' && h.trim() && !targetLines.includes(h.trim())) {
                targetLines.push(h.trim())
              }
            })
          }
          
          // 兼容显式传入的协议-端口映射
          if (data.protocols && typeof data.protocols === 'object') {
            Object.keys(data.protocols).forEach(key => {
              const upperKey = key.toUpperCase()
              const ports = Array.isArray(data.protocols[key]) ? data.protocols[key] : [data.protocols[key]]
              protocolMap[upperKey] = ports
            })
          }
          
          data.targets.forEach(target => {
            if (typeof target === 'string') {
              // 格式: protocol://host:port
              const match = target.match(/^(\w+):\/\/([^:]+):(\d+)$/)
              if (match) {
                const [, proto, host, port] = match
                const protocolKey = proto.toUpperCase()
                const portNum = parseInt(port)
                
                // 记录协议和端口
                if (!protocolMap[protocolKey]) {
                  protocolMap[protocolKey] = []
                }
                if (!protocolMap[protocolKey].includes(portNum)) {
                  protocolMap[protocolKey].push(portNum)
                }
                
                // 添加到目标列表
                if (!targetLines.includes(host)) {
                  targetLines.push(host)
                }
              }
            }
          })
          
          // 更新目标文本
          if (targetLines.length > 0) {
            targetText.value = targetLines.join('\n')
          }
          
          // 自动勾选匹配的协议和端口
          services.value.forEach(service => {
            const protocolKey = service.name.toUpperCase()
            if (protocolMap[protocolKey] && protocolMap[protocolKey].length > 0) {
              service.checked = true
              // 如果有多个端口，使用第一个
              service.port = protocolMap[protocolKey][0] || service.port
            }
          })
          
          ElMessage.success('已接收主机扫描目标并自动配置')
        }
      }
      
      // 监听来自主机扫描模块的发送事件
      window.addEventListener('send-to-crack', handleReceiveCrackTargets)
      
      // 监听来自简练助手的SSH主机列表更新事件
      const handleSSHHostListUpdate = (event) => {
        const hosts = event.detail?.hosts || []
        if (Array.isArray(hosts) && hosts.length > 0) {
          sshHostList.value = hosts.map(host => ({
            id: host.id || 0,
            name: host.name || '',
            address: host.address || '',
            port: host.port || 22,
            user: host.user || '',
            pwd: host.pwd || ''
          }))
        }
      }
      window.addEventListener('ssh-host-list-updated', handleSSHHostListUpdate)
      
      // 监听刷新密码爆破结果事件（来自主机扫描模块）
      const handleRefreshCrackResults = () => {
        // console.log('[密码爆破] 收到刷新结果事件，重新加载数据')
        loadResults()
      }
      window.addEventListener('refresh-crack-results', handleRefreshCrackResults)
      
      return () => {
        EventsOff('crackResult')
        EventsOff('crackTaskCompleted')
        window.removeEventListener('send-to-crack', handleReceiveCrackTargets)
        window.removeEventListener('ssh-host-list-updated', handleSSHHostListUpdate)
        window.removeEventListener('refresh-crack-results', handleRefreshCrackResults)
      }
    })
    
    onBeforeUnmount(() => {
      if (elapsedTimer.value) {
        clearInterval(elapsedTimer.value)
      }
    })
    
    onUnmounted(() => {
      if (elapsedTimer.value) {
        clearInterval(elapsedTimer.value)
      }
    })
    
    // keep-alive 激活时，刷新数据并确保事件监听器已注册
    onActivated(() => {
      // console.log('[密码爆破] 组件已激活，刷新数据')
      loadResults()
    })
    
    // keep-alive 失活时
    onDeactivated(() => {
      console.log('[密码爆破] 组件已失活')
    })
    
    return {
      // 数据与分页
      loading,
      results,
      columnFilters,
      crackSort,
      handleCrackColSort,
      handleCrackColumnFilter,
      pagedResults,
      totalResults,
      currentPage,
      pageSize,
      cracking,
      targetText,
      services,
      crackParams,
      globalSettings,
      alivePercent,
      crackPercent,
      statistics,
      elapsedTime,
      dictConfigVisible,
      dictActiveTab,
      usernameDict,
      passwordDict,
      selectedProtocol,
      availableProtocols,
      saving,
      loadProtocols,
      loadUsernameDict,
      loadPasswordDict,
      resetUsernameDict,
      resetPasswordDict,
      rowContextMenuVisible,
      rowContextMenuStyle,
      selectedRow,
      // 图标
      Refresh,
      Download,
      VideoPlay,
      VideoPause,
      Promotion,
      DocumentCopy,
      refreshTargets,
      openDictConfig,
      openSSHConfig,
      sshConfigVisible,
      sshConfig,
      sshHostList,
      loadSSHHostsFromConnect,
      selectSSHHost,
      saveSSHConfig,
      loadSSHConfig,
      saveUsernameDict,
      savePasswordDict,
      handleServiceChange,
      handlePortChange,
      startCrack,
      stopCrack,
      handleExport,
      handleClearResults,
      handlePageChange,
      handleSizeChange,
      formatTime,
      handleRowContextMenu,
      canSendToConnect,
      getConnectModuleName,
      canExecuteCommand,
      handleSendToConnect,
      handleExecuteCommand,
      handleCopyRow,
      commandDialogVisible,
      commandDialogState,
      currentCommonCommands,
      selectedCommonCommand,
      commandInput,
      commandResult,
      commandExecuting,
      handleCommonCommandChange,
      doExecuteCommand,
      copyCommandResult,
      resultMonacoOptions,
      resultEditorHeight,
      DocumentCopy: markRaw(DocumentCopy),
      Warning: markRaw(Warning),
      InfoFilled: markRaw(InfoFilled),
      getProtocolCommandHint,
      validateCommand,
      hasStrictCommandLimit,
      getCommandPlaceholder,
      handleQuickCommandClick
    }
  }
}
</script>

<style scoped>
.password-crack-container {
  height: 92vh;
  display: flex;
  flex-direction: column;
  padding: 10px;
  background-color: #f5f7fa;
  overflow: hidden;
}

.main-layout {
  display: flex;
  height: 100%;
  gap: 10px;
  overflow: hidden;
}

/* 左侧边栏 */
.sidebar {
  width: 200px;
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.sidebar-header {
  display: grid;
  grid-template-columns: 1fr 1fr;
  padding: 8px;
  border-bottom: 1px solid #e4e7ed;
  background-color: #f5f7fa;
  flex-shrink: 0;
}

.sidebar-title {
  font-weight: bold;
  font-size: 13px;
  text-align: center;
  padding: 4px;
}

.service-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px;
  height: 0;
}

.service-item {
  display: grid;
  grid-template-columns: 30px 1fr 60px;
  align-items: center;
  gap: 6px;
  padding: 6px 4px;
  border-bottom: 1px solid #f0f0f0;
}

.service-item:hover {
  background-color: #f5f7fa;
}

.service-name {
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.port-input {
  width: 100%;
}

.port-input :deep(.el-input__inner) {
  text-align: center;
  padding: 0 4px;
  font-size: 12px;
}

/* 主内容区 */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  height: 100%;
  overflow: hidden;
}

/* 控制面板 */
.control-panel {
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 12px;
  flex-shrink: 0;
}

/* 配置区域（合并） */
.config-section {
  width: 100%;
}

.config-row {
  display: grid;
  grid-template-columns: minmax(360px, 1.1fr) minmax(420px, 1fr);
  gap: 12px;
  align-items: start;
}

.target-input-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.target-textarea :deep(.el-textarea__inner) {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.5;
}

.target-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.target-actions .el-button {
  font-size: 12px;
  padding: 6px 12px;
}

.config-items {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.config-group {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px 12px;
  align-items: center;
}

.param-item {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.param-item label {
  font-size: 13px;
  white-space: nowrap;
  color: #606266;
}

.action-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}

.action-buttons .el-button {
  font-size: 13px;
  padding: 6px 15px;
}

.alive-toggle {
  display: flex;
  align-items: center;
  padding-right: 8px;
}

.progress-rows {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.progress-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.progress-row label {
  font-size: 13px;
  min-width: 80px;
}

.progress-row :deep(.el-progress) {
  flex: 1;
}

/* 存活/爆破：一行双卡片进度 */
.progress-inline {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.progress-card {
  border: 1px solid #eef0f3;
  background: linear-gradient(180deg, #ffffff 0%, #fbfcfe 100%);
  border-radius: 10px;
  padding: 10px 12px;
}

.progress-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 8px;
}

.progress-title {
  font-size: 13px;
  color: #606266;
  font-weight: 600;
  letter-spacing: 0.2px;
}

.progress-value {
  font-variant-numeric: tabular-nums;
  font-size: 14px;
  font-weight: 800;
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid transparent;
}

.progress-card.alive .progress-value {
  color: #067d68;
  background: rgba(16, 185, 129, 0.10);
  border-color: rgba(16, 185, 129, 0.22);
}

.progress-card.crack .progress-value {
  color: #0b5ed7;
  background: rgba(64, 158, 255, 0.12);
  border-color: rgba(64, 158, 255, 0.28);
}

/* 进度条颜色（Element Plus 内部结构，用 deep 覆盖） */
.progress-card.alive :deep(.el-progress-bar__inner) {
  background-color: #10b981;
}

.progress-card.crack :deep(.el-progress-bar__inner) {
  background-color: #409eff;
}

@media (max-width: 620px) {
  .progress-inline {
    grid-template-columns: 1fr;
  }
}

/* 响应式：窄屏时上下堆叠，参数区自动变为 2/1 列 */
@media (max-width: 1200px) {
  .config-row {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .config-group {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 620px) {
  .config-group {
    grid-template-columns: 1fr;
  }
}

/* 结果表格区域 */
.results-section {
  flex: 1;
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 12px;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.results-toolbar {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  padding-bottom: 8px;
  border-bottom: 1px solid #e4e7ed;
  margin-bottom: 8px;
}

.results-section :deep(.el-table) {
  font-size: 12px;
}

.results-section :deep(.el-table th) {
  background-color: #f5f7fa;
  padding: 8px 0;
  font-weight: 600;
}

.results-section :deep(.el-table td) {
  padding: 6px 0;
}

.results-section :deep(.el-table__empty-block) {
  padding: 60px 0;
}

.status-bar {
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 8px 10px;
  display: flex;
  align-items: center;
  gap: 16px;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #606266;
}

.status-label {
  color: #909399;
}

.status-value {
  font-weight: 600;
}

.status-spacer {
  flex: 1;
}

.status-pagination {
  display: flex;
  align-items: center;
}

/* 底部状态栏 */
/* 字典配置对话框 */
.dict-config-content {
  padding: 8px 0;
}

.dict-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dict-actions {
  display: flex;
  gap: 8px;
}

.dict-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.dict-info {
  margin-top: 12px;
  text-align: right;
}

.dict-section :deep(.el-textarea__inner) {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.6;
}

/* 右键菜单样式 */
.context-menu {
  position: fixed;
  background-color: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  padding: 4px 0;
  z-index: 9999;
  min-width: 160px;
  font-size: 14px;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  cursor: pointer;
  color: #606266;
}

.context-menu-item:hover {
  background-color: #f5f7fa;
  color: #409eff;
}

.context-menu-item.disabled {
  color: #c0c4cc;
  cursor: not-allowed;
}

.context-menu-item.disabled:hover {
  background-color: transparent;
  color: #c0c4cc;
}

.context-menu-divider {
  height: 1px;
  background-color: #e4e7ed;
  margin: 4px 0;
}

/* 命令执行对话框样式 */
.command-exec-dialog {
  :deep(.el-dialog__body) {
    padding: 0;
  }
  
  :deep(.el-dialog__header) {
    padding: 16px 20px;
    border-bottom: 1px solid #e4e7ed;
    background: #fafafa;
  }
}

.command-dialog-container {
  display: flex;
  flex-direction: column;
  height: 75vh;
  background: #fff;
}

/* 顶部信息栏 */
.command-dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: linear-gradient(135deg, #f5f7fa 0%, #ffffff 100%);
  border-bottom: 1px solid #e4e7ed;
}

.target-info-compact {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 13px;
}

.target-address {
  font-family: 'Courier New', monospace;
  font-weight: 600;
  color: #303133;
}

.target-auth {
  color: #606266;
  font-size: 12px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 主要内容区域 */
.command-dialog-main {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1px;
  overflow: hidden;
  background: #e4e7ed;
  min-height: 0;
}

/* 命令输入区域 */
.command-input-section-new {
  display: flex;
  flex-direction: column;
  background: #fff;
  overflow: hidden;
  padding: 16px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 2px solid #f0f2f5;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

/* 快速命令选择 */
.quick-commands {
  margin-bottom: 16px;
}

.quick-commands-label {
  font-size: 12px;
  color: #909399;
  margin-bottom: 8px;
  font-weight: 500;
}

.quick-commands-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.quick-command-tag {
  cursor: pointer;
  transition: all 0.2s;
  user-select: none;
}

.quick-command-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

/* 命令输入框 */
.command-input-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 0;
}

.command-textarea-new {
  flex: 1;
}

.command-textarea-new :deep(.el-textarea__inner) {
  font-family: 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  resize: none;
  border: 1px solid #dcdfe6;
  transition: all 0.2s;
  border-radius: 6px;
}

.command-textarea-new :deep(.el-textarea__inner):focus {
  border-color: #409eff;
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.1);
}

.command-textarea-new.command-invalid :deep(.el-textarea__inner) {
  border-color: #f56c6c;
}

.command-textarea-new.command-invalid :deep(.el-textarea__inner):focus {
  border-color: #f56c6c;
  box-shadow: 0 0 0 2px rgba(245, 108, 108, 0.1);
}

.command-warning-new {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: #fef0f0;
  border: 1px solid #fde2e2;
  border-radius: 4px;
  font-size: 12px;
  color: #f56c6c;
}

.warning-icon {
  font-size: 14px;
}

.warning-text {
  flex: 1;
}

.command-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: #ecf5ff;
  border: 1px solid #b3d8ff;
  border-radius: 4px;
  font-size: 12px;
  color: #409eff;
}

.command-hint .el-icon {
  font-size: 14px;
}

/* 结果展示区域 */
.command-result-section-new {
  display: flex;
  flex-direction: column;
  background: #fff;
  overflow: hidden;
}

.result-content {
  flex: 1;
  overflow: hidden;
  background: #1e1e1e;
  min-height: 0;
  position: relative;
}

.empty-result-new {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  background: #1e1e1e;
  min-height: 300px;
}

.empty-result-new :deep(.el-empty__description) {
  color: #909399;
}

.code-editor-new {
  border-radius: 0;
  overflow: hidden;
  border: none;
  height: 100% !important;
}

.code-editor-new :deep(.monaco-editor),
.code-editor-new :deep(.monaco-editor-background) {
  background: #1e1e1e;
}

.code-editor-new :deep(.monaco-editor) {
  font-family: "JetBrains Mono", "Fira Code", monospace;
}

/* 底部操作栏 */
.command-dialog-footer {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  background: #fafafa;
  border-top: 1px solid #e4e7ed;
}

/* 自定义滚动条 */
.command-input-section-new::-webkit-scrollbar {
  width: 6px;
}

.command-input-section-new::-webkit-scrollbar-track {
  background: #f5f7fa;
}

.command-input-section-new::-webkit-scrollbar-thumb {
  background: #c0c4cc;
  border-radius: 3px;
}

.command-input-section-new::-webkit-scrollbar-thumb:hover {
  background: #a8abb2;
}

.result-content::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.result-content::-webkit-scrollbar-track {
  background: #1e1e1e;
}

.result-content::-webkit-scrollbar-thumb {
  background: #424242;
  border-radius: 4px;
}

.result-content::-webkit-scrollbar-thumb:hover {
  background: #4e4e4e;
}

/* 响应式调整 */
@media (max-width: 1200px) {
  .command-dialog-main {
    grid-template-columns: 1fr;
    grid-template-rows: auto 1fr;
  }
  
  .command-input-section-new {
    max-height: 300px;
    border-bottom: 1px solid #e4e7ed;
  }
  
  .command-result-section-new {
    flex: 1;
  }
}
</style>