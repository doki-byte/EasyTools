<template>
  <div class="browser-scan-page">
    <!-- 顶部工具栏（与目录扫描等一致） -->
    <div class="top-toolbar">
      <div class="toolbar-content">
        <div class="toolbar-left">
          <div class="module-title">
            <el-icon class="title-icon"><Connection /></el-icon>
            <span class="title-text">Druid 利用</span>
          </div>
        </div>
        <div class="toolbar-right">
          <el-button type="primary" plain size="default" @click="addTask">
            <el-icon><Plus /></el-icon>
            新建任务
          </el-button>
          <el-button type="info" plain size="default" @click="historyDialog = true">
            <el-icon><Clock /></el-icon>
            任务管理
          </el-button>
        </div>
      </div>
    </div>

    <CustomTabs>
      <el-tabs
        v-model="activeTaskId"
        type="border-card"
        class="panel-card glass-card main-tabs"
        closable
        @tab-remove="removeTask"
        @tab-click="switchTask"
        @contextmenu.prevent="handleTabsContextMenu"
      >
        <el-tab-pane v-for="task in tasks" :key="task.id" :name="task.id">
          <template #label>
            <span class="tab-label">
              <el-icon v-if="task.status === 'running'" class="tab-status-icon running">
                <VideoPlay />
              </el-icon>
              <el-icon v-else-if="task.status === 'completed'" class="tab-status-icon completed">
                <CircleCheck />
              </el-icon>
              <el-icon v-else-if="task.status === 'error'" class="tab-status-icon error">
                <CircleClose />
              </el-icon>
              <span class="tab-name">{{ task.name }}</span>
            </span>
          </template>

          <div class="task-content">
            <!-- 配置区域 -->
            <div class="config-section">
              <el-card shadow="never" class="glass-card config-card">
                <div class="config-layout">
                  <!-- 左侧：URL输入 -->
                  <div class="url-input-section">
                    <div class="url-label">目标URL</div>
                    <el-input
                      v-model="task.url"
                      placeholder="例如：http://192.168.1.1:8080"
                      size="default"
                      class="url-input"
                      @keyup.enter="startScan(task.id)"
                    />
                    <div class="url-actions">
                      <el-button
                        :type="task.status === 'running' ? 'danger' : 'primary'"
                        size="default"
                        style="width: 100%"
                        :loading="task.status === 'starting' || task.status === 'stopping'"
                        :disabled="task.status === 'stopping' || task.status === 'starting'"
                        @click="task.status === 'running' ? stopScan(task.id) : startScan(task.id)"
                      >
                        {{ task.status === 'running' ? '停止扫描' : (task.status === 'stopping' ? '正在停止中' : (task.status === 'starting' ? '正在启动...' : '开始扫描')) }}
                      </el-button>
                    </div>
                  </div>

                  <!-- 右侧：配置项 -->
                  <div class="config-section-right">
                    <div class="config-group">
                      <div class="config-row single-line-config">
                        <div class="config-item">
                          <label class="config-label">用户名（可选）</label>
                          <el-input
                            v-model="task.config.username"
                            placeholder="如果 Druid 需要登录"
                            size="default"
                            clearable
                            style="width: 100%"
                          />
                        </div>
                        <div class="config-item">
                          <label class="config-label">密码（可选）</label>
                          <el-input
                            v-model="task.config.password"
                            type="password"
                            placeholder="如果 Druid 需要登录"
                            size="default"
                            show-password
                            clearable
                            style="width: 100%"
                          />
                        </div>
                        <div class="config-item">
                          <label class="config-label">超时时间（秒）</label>
                          <el-input-number
                            v-model="task.config.timeout"
                            :min="1"
                            :max="60"
                            controls-position="right"
                            size="default"
                            style="width: 100%"
                          />
                        </div>
                        <div class="config-item">
                          <label class="config-label">任务状态</label>
                          <el-tag
                            :type="task.status === 'running' ? 'warning' : (task.status === 'completed' ? 'success' : (task.status === 'error' ? 'danger' : 'info'))"
                            effect="dark"
                            style="width: fit-content"
                          >
                            {{ getStatusText(task.status) }}
                          </el-tag>
                        </div>
                        <div class="config-item">
                          <label class="config-label">操作</label>
                          <el-button type="success" size="small" effect="dark" @click="exportToExcel(task.id)" :disabled="!task.result || !task.result.success" style="width: 100%">
                            导出结果
                          </el-button>
                        </div>
                      </div>

                      <div class="config-row" style="display: flex; align-items: center; gap: 10px; margin-top: 10px;">
                        <div class="config-item" style="flex: 1;">
                          <label class="config-label">自定义请求头（可选）</label>
                          <IsolatedTextarea
                            :ref="(el) => setIsolatedRef('druid-hdr-' + task.id, el)"
                            v-model="task.config.customHeadersText"
                            :rows="1"
                            placeholder="每行一个请求头，格式: Key: Value"
                            size="default"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </el-card>
            </div>

            <!-- 结果展示区域 -->
            <div class="results-section">
              <el-card shadow="never" class="glass-card results-card">
                <el-tabs :model-value="getActiveResultTab(task.id)" @update:model-value="(val) => setActiveResultTab(task.id, val)" type="border-card" class="result-tabs">
                  <el-tab-pane label="Sessions" name="sessions">
                    <div class="result-toolbar" style="padding: 10px;">
                      <el-button type="primary" size="small" @click="batchSendToRestmate(task, 'sessions')" :disabled="!getSelectedSessions(task.id).length">
                        批量发送到发包 ({{ getSelectedSessions(task.id).length }})
                      </el-button>
                      <el-button type="success" size="small" @click="sendAllSessionsToRestmate(task)" :disabled="!task.result?.sessions || task.result.sessions.length === 0">
                        全部发送到发包 ({{ task.result?.sessions?.length || 0 }})
                      </el-button>
                      <el-button size="small" @click="clearSelectedSessions(task.id)">清空选择</el-button>
                    </div>
                    <el-table 
                      :ref="el => setSessionsTableRef(task.id, el)"
                      :data="getPaginatedData(task.id, 'sessions', formatArrayData(task.result?.sessions))" 
                      border 
                      stripe 
                      size="small" 
                      max-height="400" 
                      class="glass-card"
                      row-key="value"
                      @selection-change="(selection) => handleSessionsSelectionChange(task.id, selection)"
                    >
                      <el-table-column type="selection" width="55" :reserve-selection="true" />
                      <el-table-column prop="value" show-overflow-tooltip min-width="200">
                        <template #header>
                          <TableColumnFilterHeader
                            v-model="task.colFilters.sessions"
                            label="Session"
                            @filter-change="handleDruidColFilter(task.id, 'sessions')"
                          />
                        </template>
                        <template #default="{ row }">
                          <el-tooltip
                              :content="row.value || ''"
                              placement="top"
                              :disabled="!row.value || row.value.length <= 30"
                              effect="dark"
                          >
                            <div class="table-cell-text">
                              {{ row.value || '-' }}
                            </div>
                          </el-tooltip>
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="200" fixed="right">
                        <template #default="scope">
                          <el-button-group class="action-button-group">
                            <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                            <el-button size="small" type="primary" @click="sendToRestmate(task, 'session', scope.row.value)">发送到发包</el-button>
                          </el-button-group>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'sessions').page"
                        v-model:page-size="getPaginationConfig(task.id, 'sessions').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'sessions', formatArrayData(task.result?.sessions))"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'sessions', $event)"
                        @current-change="handlePageChange(task.id, 'sessions', $event)"
                      />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="SQLs" name="sqls">
                    <el-table :data="getPaginatedData(task.id, 'sqls', formatArrayData(task.result?.sqls))" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip min-width="200">
                        <template #header>
                          <TableColumnFilterHeader
                            v-model="task.colFilters.sqls"
                            label="SQL"
                            @filter-change="handleDruidColFilter(task.id, 'sqls')"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="100" fixed="right">
                        <template #default="scope">
                          <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'sqls').page"
                        v-model:page-size="getPaginationConfig(task.id, 'sqls').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'sqls', formatArrayData(task.result?.sqls))"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'sqls', $event)"
                        @current-change="handlePageChange(task.id, 'sqls', $event)"
                      />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="URIs" name="uris">
                    <div class="result-toolbar" style="padding: 10px;">
                      <el-button type="primary" size="small" @click="batchSendToRestmate(task, 'uris')" :disabled="!getSelectedUris(task.id).length">
                        批量发送到发包 ({{ getSelectedUris(task.id).length }})
                      </el-button>
                      <el-button type="success" size="small" @click="sendAllUrisToRestmate(task)" :disabled="!task.result?.uris || task.result.uris.length === 0">
                        全部发送到发包 ({{ task.result?.uris?.length || 0 }})
                      </el-button>
                      <el-button size="small" @click="clearSelectedUris(task.id)">清空选择</el-button>
                    </div>
                    <el-table 
                      :ref="el => setUrisTableRef(task.id, el)"
                      :data="getPaginatedData(task.id, 'uris', formatArrayData(task.result?.uris))" 
                      border 
                      stripe 
                      size="small" 
                      max-height="400" 
                      class="glass-card"
                      row-key="value"
                      @selection-change="(selection) => handleUrisSelectionChange(task.id, selection)"
                    >
                      <el-table-column type="selection" width="55" :reserve-selection="true" />
                      <el-table-column prop="value" show-overflow-tooltip min-width="200">
                        <template #header>
                          <TableColumnFilterHeader
                            v-model="task.colFilters.uris"
                            label="URI"
                            @filter-change="handleDruidColFilter(task.id, 'uris')"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="200" fixed="right">
                        <template #default="scope">
                          <el-button-group>
                            <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                            <el-button size="small" type="primary" @click="sendToRestmate(task, 'uri', scope.row.value)">发送到发包</el-button>
                          </el-button-group>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'uris').page"
                        v-model:page-size="getPaginationConfig(task.id, 'uris').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'uris', formatArrayData(task.result?.uris))"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'uris', $event)"
                        @current-change="handlePageChange(task.id, 'uris', $event)"
                      />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="JDBC信息" name="jdbc">
                    <div class="info-display">
                      <div v-if="task.result?.jdbcInfo?.userName" class="info-item">
                        <span class="info-label">用户名:</span>
                        <span class="info-value">{{ task.result.jdbcInfo.userName }}</span>
                        <el-button size="small" @click="copyToClipboard(task.result.jdbcInfo.userName)" style="margin-left: 10px;">复制</el-button>
                      </div>
                      <div v-if="task.result?.jdbcInfo?.jdbcUrl" class="info-item">
                        <span class="info-label">JDBC URL:</span>
                        <span class="info-value">{{ task.result.jdbcInfo.jdbcUrl }}</span>
                        <el-button size="small" @click="copyToClipboard(task.result.jdbcInfo.jdbcUrl)" style="margin-left: 10px;">复制</el-button>
                      </div>
                      <div v-if="task.result?.jdbcInfo?.javaClassPath" class="info-item">
                        <span class="info-label">Java Class Path:</span>
                        <span class="info-value">{{ task.result.jdbcInfo.javaClassPath }}</span>
                        <el-button size="small" @click="copyToClipboard(task.result.jdbcInfo.javaClassPath)" style="margin-left: 10px;">复制</el-button>
                      </div>
                      <div v-if="task.result?.jdbcInfo?.javaVMName" class="info-item">
                        <span class="info-label">Java VM Name:</span>
                        <span class="info-value">{{ task.result.jdbcInfo.javaVMName }}</span>
                        <el-button size="small" @click="copyToClipboard(task.result.jdbcInfo.javaVMName)" style="margin-left: 10px;">复制</el-button>
                      </div>
                      <div v-if="task.result?.jdbcInfo?.javaVersion" class="info-item">
                        <span class="info-label">Java Version:</span>
                        <span class="info-value">{{ task.result.jdbcInfo.javaVersion }}</span>
                        <el-button size="small" @click="copyToClipboard(task.result.jdbcInfo.javaVersion)" style="margin-left: 10px;">复制</el-button>
                      </div>
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="基础信息" name="basic">
                    <div class="info-display">
                      <div v-if="task.result?.basicInfo?.javaClassPath" class="info-item">
                        <span class="info-label">Java Class Path:</span>
                        <span class="info-value">{{ task.result.basicInfo.javaClassPath }}</span>
                        <el-button size="small" @click="copyToClipboard(task.result.basicInfo.javaClassPath)" style="margin-left: 10px;">复制</el-button>
                      </div>
                      <div v-if="task.result?.basicInfo?.javaVMName" class="info-item">
                        <span class="info-label">Java VM Name:</span>
                        <span class="info-value">{{ task.result.basicInfo.javaVMName }}</span>
                        <el-button size="small" @click="copyToClipboard(task.result.basicInfo.javaVMName)" style="margin-left: 10px;">复制</el-button>
                      </div>
                      <div v-if="task.result?.basicInfo?.javaVersion" class="info-item">
                        <span class="info-label">Java Version:</span>
                        <span class="info-value">{{ task.result.basicInfo.javaVersion }}</span>
                        <el-button size="small" @click="copyToClipboard(task.result.basicInfo.javaVersion)" style="margin-left: 10px;">复制</el-button>
                      </div>
                    </div>
                  </el-tab-pane>
                </el-tabs>
              </el-card>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </CustomTabs>

    <!-- 标签页右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
    <Teleport to="body">
      <ul 
        class="tabs-context-menu" 
        v-if="tabsContextMenuVisible" 
        :style="tabsContextMenuStyle"
      >
        <li @click="handleCloseOtherTabs">关闭其他</li>
        <li @click="handleCloseAllTabs">全部关闭</li>
      </ul>
    </Teleport>
    
    <!-- 任务管理对话框 -->
    <el-dialog
      v-model="historyDialog"
      width="75%"
      destroy-on-close
      class="history-dialog"
      @opened="handleHistoryDialogOpened"
      @closed="handleHistoryDialogClosed"
    >
      <template #header>
        <div class="dialog-title" style="display: flex; justify-content: space-between; align-items: center;">
          <div>
            <el-icon :size="18" class="mr-5px"><Clock /></el-icon>任务管理
          </div>
          <div>
            <el-button text @click="refreshTasks(true)" :loading="tasksLoading">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
            <el-button type="danger" text @click="clearAllTasks">
              清除全部
            </el-button>
          </div>
        </div>
      </template>
      <div class="history-dialog-content">
        <el-table
          :data="paginatedTasks"
          stripe
          border
          v-loading="tasksLoading"
          class="history-table"
          :height="400"
        >
          <el-table-column type="index" width="60" align="center" label="#" />
          <el-table-column prop="url" label="目标URL" min-width="250">
            <template #default="{ row }">
              <el-tooltip
                  :content="row.url || ''"
                  placement="top"
                  :disabled="!row.url || row.url.length <= 30"
                  effect="dark"
              >
                <div class="task-name-cell">
                  {{ row.url || '-' }}
                </div>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100" align="center">
            <template #default="scope">
              <el-tag :type="getStatusTagType(scope.row.status)">
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="createTime" label="创建时间" width="180" align="center">
            <template #default="{ row }">
              {{ formatTime(row.createTime) }}
            </template>
          </el-table-column>
          <el-table-column prop="updateTime" label="更新时间" width="180" align="center">
            <template #default="{ row }">
              {{ formatTime(row.updateTime) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" align="center" fixed="right">
            <template #default="scope">
              <el-button-group>
                <el-tooltip content="加载任务数据" placement="top">
                  <el-button link @click="loadTaskData(scope.row.id)">
                    <el-icon><View /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <el-button link @click="deleteTaskFromHistory(scope.row.id)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </el-tooltip>
              </el-button-group>
            </template>
          </el-table-column>
        </el-table>
        <div class="history-pagination">
          <el-pagination small
            v-model:current-page="taskPagination.currentPage"
            v-model:page-size="taskPagination.pageSize"
            :page-sizes="[10, 20, 50, 100, 200]"
            :total="allTasks.length"
            layout="total, sizes, prev, pager, next, jumper"
            background
            @size-change="handleTaskPageSizeChange"
            @current-change="handleTaskPageChange"
          />
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { ElMessage, ElMessageBox } from 'element-plus';
import { VideoPlay, VideoPause, CircleCheck, CircleClose, Clock, Refresh, Delete, View, Plus, Connection, Link, CopyDocument, Download } from '@element-plus/icons-vue';
import { ScanDruid } from '../../../wailsjs/go/druid/DruidController';
import { SaveFile, GetAppPath, ReadFile } from '../../../wailsjs/go/controller/Note';
import { AddCollection, UpsertRequest } from '../../../wailsjs/go/restmate/RestMate';
import { nanoid } from 'nanoid';
import CustomTabs from '@/components/CustomTabs.vue';
import TableColumnFilterHeader from '@/components/TableColumnFilterHeader.vue';
import IsolatedTextarea from '@/components/IsolatedTextarea.vue';

export default {
  name: 'DruidView',
  components: {
    CustomTabs,
    TableColumnFilterHeader,
    IsolatedTextarea,
    VideoPlay,
    VideoPause,
    CircleCheck,
    CircleClose,
    Clock,
    Refresh,
    Delete,
    View,
    Plus,
    Connection,
    Link,
    CopyDocument,
    Download,
  },
  data() {
    return {
      activeTaskId: '',
      activeResultTab: {}, // 每个任务的结果标签页状态 { taskId: 'sessions' }
      tasks: [],
      taskIdCounter: 0,
      // 分页配置：每个任务每个标签页的分页状态 { taskId: { tabName: { page: 1, pageSize: 20 } } }
      paginationConfig: {},
      defaultPageSize: 10, // 默认每页显示数量
      // 任务管理相关
      historyDialog: false,
      allTasks: [], // 所有任务（用于任务管理）
      tasksLoading: false,
      tasksLocal: {}, // 本地任务存储 { taskId: task }
      isolatedTextareaRefs: {},
      // 任务管理分页
      taskPagination: {
        currentPage: 1,
        pageSize: 10
      },
      // 标签页右键菜单相关
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: '',
      // 选中的 URIs { taskId: [uri1, uri2, ...] }
      selectedUris: {},
      // URIs 表格引用 { taskId: tableRef }
      urisTableRefs: {},
      // 选中的 Sessions { taskId: [session1, session2, ...] }
      selectedSessions: {},
      // Sessions 表格引用 { taskId: tableRef }
      sessionsTableRefs: {},
      // 保存防抖定时器
      saveTimer: null
    };
  },
  computed: {
    // 获取分页后的任务列表（用于任务管理）
    paginatedTasks() {
      const start = (this.taskPagination.currentPage - 1) * this.taskPagination.pageSize;
      const end = start + this.taskPagination.pageSize;
      return this.allTasks.slice(start, end);
    },
    // 标签页右键菜单样式
    tabsContextMenuStyle() {
      return {
        left: `${this.tabsContextMenuPosition.x}px`,
        top: `${this.tabsContextMenuPosition.y}px`
      };
    }
  },
  async mounted() {
    await this.loadTasks();
    if (this.tasks.length === 0) {
      this.addTask();
    }
  },
  methods: {
    setIsolatedRef(key, el) {
      if (el) {
        this.isolatedTextareaRefs[key] = el;
      } else {
        delete this.isolatedTextareaRefs[key];
      }
    },
    flushDruidHeadersForTask(task) {
      if (!task) return;
      this.isolatedTextareaRefs[`druid-hdr-${task.id}`]?.flush?.();
    },
    // 格式化数组数据为表格格式
    formatArrayData(arr) {
      if (!Array.isArray(arr)) return [];
      return arr.map(item => ({ value: item }));
    },

    // 保存扫描结果到本地JSON文件
    async saveResultsToFile(taskId, result) {
      try {
        const appPath = await GetAppPath();
        const pathSeparator = appPath.includes('\\') ? '\\' : '/';
        const resultsDir = `${appPath}${pathSeparator}logs${pathSeparator}druid`;
        
        const timestamp = Date.now();
        const fileName = `druid_${taskId}_${timestamp}.json`;
        const filePath = `${resultsDir}${pathSeparator}${fileName}`;
        
        const dataToSave = {
          taskId,
          timestamp: new Date().toISOString(),
          result: result || null
        };
        
        await SaveFile(filePath, JSON.stringify(dataToSave, null, 2));
        return filePath;
      } catch (error) {
        console.error('保存结果到文件失败:', error);
        return null;
      }
    },
    
    // 从文件加载扫描结果
    async loadResultsFromFile(filePath) {
      try {
        if (!filePath) return null;
        const content = await ReadFile(filePath);
        return JSON.parse(content);
      } catch (error) {
        console.error('从文件加载结果失败:', error);
        return null;
      }
    },

    // 加载任务
    async loadTasks() {
      try {
        const tasksStr = localStorage.getItem('druidTasks');
        if (tasksStr) {
          const tasksData = JSON.parse(tasksStr);
          this.tasksLocal = tasksData;
          
          // 加载任务列表，但不立即加载完整结果（按需加载）
          this.tasks = await Promise.all(Object.values(tasksData).map(async task => {
            let result = null;
            
            // 如果任务有结果文件路径，从文件加载完整结果
            if (task.resultsFilePath) {
              const fileData = await this.loadResultsFromFile(task.resultsFilePath);
              if (fileData && fileData.result) {
                result = fileData.result;
              }
            } else if (task.result) {
              // 兼容旧数据：如果直接有 result 字段，使用它
              result = task.result;
            }
            
            return {
              ...task,
              status: task.status || 'pending',
              result: result,
              config: task.config || {
                username: '',
                password: '',
                customHeadersText: '',
                timeout: 10,
              },
              colFilters: task.colFilters || {}
            };
          }));
          
          // 设置活动任务
          if (this.tasks.length > 0 && !this.activeTaskId) {
            this.activeTaskId = this.tasks[0].id;
          }
        } else {
          this.tasks = [];
          this.tasksLocal = {};
        }
      } catch (error) {
        console.error('加载任务失败:', error);
      }
    },

    // 保存任务
    async saveTasks() {
      try {
        // 先同步 tasks 的最新数据到 tasksLocal（确保用户输入的内容被保存）
        this.tasks.forEach(task => {
          if (!this.tasksLocal[task.id]) {
            this.tasksLocal[task.id] = {};
          }
          // 同步任务数据，但保留 resultsFilePath（如果存在）
          const existingTask = this.tasksLocal[task.id];
          this.tasksLocal[task.id] = {
            ...task,
            // 保留原有的 resultsFilePath，避免重复保存文件
            resultsFilePath: existingTask?.resultsFilePath || null,
            // 更新更新时间
            updateTime: Date.now()
          };
        });

        // 更新任务时间戳
        Object.values(this.tasksLocal).forEach(task => {
          if (this.tasks.find(t => t.id === task.id)) {
            task.updateTime = Date.now();
          }
        });

        // 压缩任务数据：完整结果保存到文件，localStorage只保存元数据
        const compressedTasks = {};
        for (const [taskId, task] of Object.entries(this.tasksLocal)) {
          // 如果任务有完整结果且没有保存到文件，则保存到文件
          if (task.result && !task.resultsFilePath) {
            const filePath = await this.saveResultsToFile(taskId, task.result);
            if (filePath) {
              task.resultsFilePath = filePath;
            }
          }
          
          compressedTasks[taskId] = {
            ...task,
            // 不保存完整结果到localStorage，只保存文件路径
            result: undefined,
            resultsCount: task.result ? 1 : 0,
            // 保存文件路径，用于后续加载
            resultsFilePath: task.resultsFilePath || null
          };
        }
        
        const tasksStr = JSON.stringify(compressedTasks);
        
        // 检查数据大小，如果超过 2MB，只保留最近100个任务
        if (tasksStr.length > 2 * 1024 * 1024) {
          const taskArray = Object.entries(compressedTasks)
            .sort((a, b) => (b[1].createTime || 0) - (a[1].createTime || 0))
            .slice(0, 100);
          
          const limitedTasks = {};
          taskArray.forEach(([taskId, task]) => {
            limitedTasks[taskId] = task;
          });
          
          localStorage.setItem('druidTasks', JSON.stringify(limitedTasks));
          this.tasksLocal = limitedTasks;
        } else {
          localStorage.setItem('druidTasks', tasksStr);
        }
      } catch (error) {
        if (error.name === 'QuotaExceededError') {
          console.warn('存储空间不足，清理旧任务');
          try {
            const taskArray = Object.entries(this.tasksLocal)
              .sort((a, b) => (b[1].createTime || 0) - (a[1].createTime || 0))
              .slice(0, 50);
            
            const limitedTasks = {};
            taskArray.forEach(([taskId, task]) => {
              limitedTasks[taskId] = {
                ...task,
                result: undefined,
                resultsCount: task.result ? 1 : 0,
                resultsFilePath: task.resultsFilePath || null
              };
            });
            
            localStorage.setItem('druidTasks', JSON.stringify(limitedTasks));
            this.tasksLocal = limitedTasks;
          } catch (e) {
            console.error('清理旧任务失败:', e);
          }
        } else {
          console.error('保存任务失败:', error);
        }
      }
    },

    // 添加任务
    addTask() {
      const taskId = `druid_${Date.now()}_${++this.taskIdCounter}`;
      const newTask = {
        id: taskId,
        name: `Druid利用-${new Date().toLocaleTimeString()}`,
        url: '',
        status: 'pending',
        config: {
          username: '',
          password: '',
          customHeadersText: '',
          timeout: 10,
        },
        result: null,
        createTime: Date.now(),
        updateTime: Date.now(),
        colFilters: {}
      };
      
      this.tasks.push(newTask);
      this.tasksLocal[taskId] = { ...newTask };
      this.activeTaskId = taskId;
      // 初始化新任务的结果标签页状态
      this.activeResultTab[taskId] = 'sessions';
      this.saveTasks();
    },

    // 开始扫描
    async startScan(taskId) {
      const task = this.tasks.find(t => t.id === taskId);
      if (!task) return;
      this.flushDruidHeadersForTask(task);

      if (!task.url) {
        ElMessage.warning('请输入目标 URL');
        return;
      }

      if (task.status === 'running' || task.status === 'starting' || task.status === 'stopping') {
        return;
      }

      try {
        task.status = 'starting';
        task.updateTime = Date.now();

        const configJSON = JSON.stringify({
          url: task.url,
          username: task.config.username,
          password: task.config.password,
          headers: this.parseHeaders(task.config.customHeadersText),
          timeout: task.config.timeout,
        });

        const result = await ScanDruid(configJSON);
        task.result = result;
        task.status = result.success ? 'completed' : 'error';
        task.updateTime = Date.now();

        // 更新本地存储
        this.tasksLocal[taskId] = { ...task };
        
        // 扫描完成后立即保存（包括保存结果到文件）
        await this.saveTasks();
        
        // 如果扫描成功，确保结果已保存到文件
        if (result.success && task.result) {
          const existingTask = this.tasksLocal[taskId];
          if (!existingTask?.resultsFilePath) {
            // 如果文件保存失败，再次尝试保存
            const filePath = await this.saveResultsToFile(taskId, task.result);
            if (filePath) {
              this.tasksLocal[taskId].resultsFilePath = filePath;
              // 再次保存到 localStorage
              await this.saveTasks();
            }
          }
        }

        if (result.success) {
          ElMessage.success('扫描完成');
        } else {
          ElMessage.error(result.message || '扫描失败');
        }
      } catch (error) {
        console.error('扫描失败:', error);
        task.status = 'error';
        task.result = {
          success: false,
          message: error.message || '扫描失败',
        };
        task.updateTime = Date.now();
        this.tasksLocal[taskId] = { ...task };
        this.saveTasks();
        ElMessage.error('扫描失败: ' + (error.message || error));
      }
    },

    // 停止扫描
    stopScan(taskId) {
      const task = this.tasks.find(t => t.id === taskId);
      if (!task) return;

      if (task.status === 'running') {
        task.status = 'stopped';
        task.updateTime = Date.now();
        this.tasksLocal[taskId] = { ...task };
        this.saveTasks();
        ElMessage.info('已停止扫描');
      }
    },

    // 解析请求头
    parseHeaders(customHeadersText) {
      const headers = {};
      if (!customHeadersText) {
        return headers;
      }

      const lines = customHeadersText.split('\n');
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;

        const colonIndex = trimmed.indexOf(':');
        if (colonIndex > 0) {
          const key = trimmed.substring(0, colonIndex).trim();
          const value = trimmed.substring(colonIndex + 1).trim();
          if (key && value) {
            headers[key] = value;
          }
        }
      }
      return headers;
    },

    // 删除任务
    removeTask(taskId) {
      const task = this.tasks.find(t => t.id === taskId);
      if (!task) return;
      
      this.tasks = this.tasks.filter(t => t.id !== taskId);
      delete this.tasksLocal[taskId];
      delete this.activeResultTab[taskId];
      delete this.paginationConfig[taskId];
      delete this.selectedUris[taskId];
      delete this.urisTableRefs[taskId];
      delete this.selectedSessions[taskId];
      delete this.sessionsTableRefs[taskId];
      this.saveTasks();
      
      if (this.activeTaskId === taskId) {
        this.activeTaskId = this.tasks.length > 0 ? this.tasks[0].id : '';
      }

      // 避免进入无标签空白态，保持一个可操作的空白任务标签
      if (this.tasks.length === 0) {
        this.addTask();
      }
    },

    // 切换任务
    switchTask(tab) {
      this.activeTaskId = tab.name;
    },

    // 标签页右键菜单处理
    handleTabsContextMenu(event) {
      event.preventDefault()
      event.stopPropagation()
      
      // 检查是否点击在标签页上
      const tabItem = event.target.closest('.el-tabs__item')
      if (!tabItem) {
        this.hideTabsContextMenu()
        return
      }
      
      const tabIdAttr = tabItem.id || tabItem.getAttribute('aria-controls')
      this.tabsContextMenuTabId = tabIdAttr
          ? tabIdAttr.replace(/^tab-/, '').replace(/^pane-/, '')
          : this.activeTaskId

      // 直接从事件获取坐标，优先使用 clientX/clientY（相对于视口，适用于 position: fixed）
      const x = event.clientX || 0
      const y = event.clientY || 0
      
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
    // 关闭其他标签页
    async handleCloseOtherTabs() {
      this.hideTabsContextMenu()

      const targetId = this.tabsContextMenuTabId || this.activeTaskId
      const targetTask = this.tasks.find(task => task.id === targetId)
      
      if (!targetTask) {
        return
      }

      // 关闭其他标签页，不停止扫描，允许后台继续运行
      const tasksToClose = this.tasks.filter(task => task.id !== targetId)
      for (const task of tasksToClose) {
        // 清理该任务的结果标签页状态和分页配置
        delete this.activeResultTab[task.id]
        delete this.paginationConfig[task.id]
      }

      // 更新标签列表，只保留目标标签
      this.tasks = [targetTask]
      this.activeTaskId = targetId
    },
    // 关闭所有标签页
    async handleCloseAllTabs() {
      this.hideTabsContextMenu()
      
      // 关闭所有标签页，不停止扫描，允许后台继续运行
      for (const task of this.tasks) {
        // 清理该任务的结果标签页状态和分页配置
        delete this.activeResultTab[task.id]
        delete this.paginationConfig[task.id]
      }

      // 清空标签列表
      this.tasks = []
      // 保留一个空白任务标签，避免页面进入 empty 态
      this.addTask()
    },

    // 复制到剪贴板
    copyToClipboard(text) {
      if (!text) {
        ElMessage.warning('没有可复制的内容');
        return;
      }

      navigator.clipboard.writeText(text).then(() => {
        ElMessage.success('已复制到剪贴板');
      }).catch(() => {
        ElMessage.error('复制失败');
      });
    },

    // 发送到 Restmate
    sendToRestmate(task, type, value) {
      const storageKey = 'easytools:restmate:packet';
      const baseURL = task.url.replace(/\/$/, '');
      
      if (type === 'session' && value) {
        // 单个发包：随机选择 URL
        const requestPacket = this.buildSessionRequestPacket(baseURL, value, task.config, task.result?.uris);
        
        const packetData = {
          fromWebscan: true,
          requestName: `Druid Session - ${value.substring(0, 20)}...`,
          requestPacket: requestPacket,
          responsePacket: '',
          timestamp: Date.now(),
        };
        
        sessionStorage.setItem(storageKey, JSON.stringify(packetData));
        this.$router.push({ path: '/restmate', query: { openPacket: 'true' } });
        ElMessage.success('已发送到 Restmate 发包功能');
      } else if (type === 'uri' && value) {
        let fullURL = value;
        if (!value.startsWith('http://') && !value.startsWith('https://')) {
          fullURL = baseURL + (value.startsWith('/') ? value : '/' + value);
        }
        
        const urlObj = new URL(fullURL);
        const headers = this.parseHeaders(task.config.customHeadersText);
        
        let requestPacket = `GET ${urlObj.pathname + urlObj.search} HTTP/1.1\r\n`;
        requestPacket += `Host: ${urlObj.host}\r\n`;
        requestPacket += `User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36\r\n`;
        
        for (const [key, val] of Object.entries(headers)) {
          requestPacket += `${key}: ${val}\r\n`;
        }
        
        requestPacket += `\r\n`;
        
        const packetData = {
          fromWebscan: true,
          requestName: `Druid URI - ${value}`,
          requestPacket: requestPacket,
          responsePacket: '',
          timestamp: Date.now(),
        };
        
        sessionStorage.setItem(storageKey, JSON.stringify(packetData));
        this.$router.push({ path: '/restmate', query: { openPacket: 'true' } });
        ElMessage.success('已发送到 Restmate 发包功能');
      }
    },

    // 处理 URIs 选择变化
    handleUrisSelectionChange(taskId, selection) {
      if (!this.selectedUris[taskId]) {
        this.selectedUris[taskId] = [];
      }
      this.selectedUris[taskId] = selection.map(row => row.value);
    },

    // 获取选中的 URIs
    getSelectedUris(taskId) {
      return this.selectedUris[taskId] || [];
    },

    // 设置 URIs 表格引用
    setUrisTableRef(taskId, el) {
      if (el) {
        this.urisTableRefs[taskId] = el;
      }
    },

    // 清空选中的 URIs
    clearSelectedUris(taskId) {
      this.selectedUris[taskId] = [];
      // 清空表格选择
      this.$nextTick(() => {
        const table = this.urisTableRefs[taskId];
        if (table && table.clearSelection) {
          table.clearSelection();
        }
      });
    },

    // 处理 Sessions 选择变化
    handleSessionsSelectionChange(taskId, selection) {
      if (!this.selectedSessions[taskId]) {
        this.selectedSessions[taskId] = [];
      }
      this.selectedSessions[taskId] = selection.map(row => row.value);
    },

    // 获取选中的 Sessions
    getSelectedSessions(taskId) {
      return this.selectedSessions[taskId] || [];
    },

    // 设置 Sessions 表格引用
    setSessionsTableRef(taskId, el) {
      if (el) {
        this.sessionsTableRefs[taskId] = el;
      }
    },

    // 清空选中的 Sessions
    clearSelectedSessions(taskId) {
      this.selectedSessions[taskId] = [];
      // 清空表格选择
      this.$nextTick(() => {
        const table = this.sessionsTableRefs[taskId];
        if (table && table.clearSelection) {
          table.clearSelection();
        }
      });
    },

    // 批量发送到 Restmate
    async batchSendToRestmate(task, type) {
      if (type === 'uris') {
        const selectedUris = this.getSelectedUris(task.id);
        if (!selectedUris || selectedUris.length === 0) {
          ElMessage.warning('请先选择要发送的 URI');
          return;
        }

        await this.sendUrisToRestmate(task, selectedUris);
      } else if (type === 'sessions') {
        const selectedSessions = this.getSelectedSessions(task.id);
        if (!selectedSessions || selectedSessions.length === 0) {
          ElMessage.warning('请先选择要发送的 Session');
          return;
        }

        await this.sendSessionsToRestmate(task, selectedSessions);
      }
    },

    // 全部发送到 Restmate
    async sendAllUrisToRestmate(task) {
      if (!task.result?.uris || task.result.uris.length === 0) {
        ElMessage.warning('没有可发送的 URI');
        return;
      }

      await this.sendUrisToRestmate(task, task.result.uris);
    },

    // 全部发送 Sessions 到 Restmate
    async sendAllSessionsToRestmate(task) {
      if (!task.result?.sessions || task.result.sessions.length === 0) {
        ElMessage.warning('没有可发送的 Session');
        return;
      }

      await this.sendSessionsToRestmate(task, task.result.sessions);
    },

    // 发送 URIs 到 Restmate（通用方法）
    async sendUrisToRestmate(task, uris) {
      if (!uris || uris.length === 0) {
        ElMessage.warning('没有可发送的 URI');
        return;
      }

      try {
        // 创建分组
        const collectionId = nanoid();
        const collectionName = `Druid URIs - ${task.url} - ${new Date().toLocaleString()}`;
        
        const createRes = await AddCollection(collectionId, collectionName);
        if (!createRes.success) {
          ElMessage.error('创建分组失败: ' + (createRes.msg || '未知错误'));
          return;
        }

        // 批量创建请求
        const baseURL = task.url.replace(/\/$/, '');
        const headersObj = this.parseHeaders(task.config.customHeadersText);
        // 将 headers 对象转换为 KeyValue 数组格式
        const headersArray = Object.entries(headersObj).map(([key, value]) => ({
          id: nanoid(),
          key: key,
          value: value,
          active: true
        }));
        let successCount = 0;
        let failCount = 0;

        for (const uri of uris) {
          try {
            let fullURL = uri;
            if (!uri.startsWith('http://') && !uri.startsWith('https://')) {
              fullURL = baseURL + (uri.startsWith('/') ? uri : '/' + uri);
            }

            const urlObj = new URL(fullURL);
            
            let requestPacket = `GET ${urlObj.pathname + urlObj.search} HTTP/1.1\r\n`;
            requestPacket += `Host: ${urlObj.host}\r\n`;
            requestPacket += `User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36\r\n`;
            
            for (const [key, val] of Object.entries(headersObj)) {
              requestPacket += `${key}: ${val}\r\n`;
            }
            
            requestPacket += `\r\n`;

            const requestId = nanoid();
            const request = {
              id: requestId,
              coll_id: collectionId,
              name: `Druid URI - ${uri}`,
              method: 'GET',
              url: fullURL,
              requestPacket: requestPacket,
              responsePacket: '',
              headers: headersArray,
              params: [],
              body: {
                bodyType: 'json',
                bodyRaw: '',
                formData: []
              },
              description: `从 Druid 扫描导入: ${task.url}`
            };

            const upsertRes = await UpsertRequest(request);
            if (upsertRes.success) {
              successCount++;
            } else {
              failCount++;
              console.error(`添加请求失败: ${uri}`, upsertRes.msg);
            }
          } catch (error) {
            failCount++;
            console.error(`处理 URI 失败: ${uri}`, error);
          }
        }

        // 清空选择
        this.clearSelectedUris(task.id);

        // 跳转到 Restmate
        this.$router.push({ path: '/restmate' });
        
        if (failCount === 0) {
          ElMessage.success(`成功导入 ${successCount} 个 URI 到分组「${collectionName}」`);
        } else {
          ElMessage.warning(`导入完成：成功 ${successCount} 个，失败 ${failCount} 个`);
        }
      } catch (error) {
        console.error('批量发送失败:', error);
        ElMessage.error('批量发送失败: ' + (error.message || error));
      }
    },

    // 从 URIs 中随机选择一个 URI 并构建完整 URL
    getRandomUriUrl(baseURL, uris) {
      let targetPath = '/druid/index.html';
      let fullURL = baseURL + targetPath;
      
      if (uris && uris.length > 0) {
        const randomIndex = Math.floor(Math.random() * uris.length);
        const randomUri = uris[randomIndex];
        
        // 如果 URI 是完整 URL，直接使用
        if (randomUri.startsWith('http://') || randomUri.startsWith('https://')) {
          fullURL = randomUri;
          try {
            const urlObj = new URL(randomUri);
            targetPath = urlObj.pathname + urlObj.search;
          } catch (e) {
            targetPath = randomUri.startsWith('/') ? randomUri : '/' + randomUri;
          }
        } else {
          // 相对路径，拼接 baseURL
          targetPath = randomUri.startsWith('/') ? randomUri : '/' + randomUri;
          fullURL = baseURL + targetPath;
        }
      }
      
      return { fullURL, targetPath };
    },

    // 构建 Session 请求包（单个发包使用，随机选择URL）
    buildSessionRequestPacket(baseURL, session, config, uris) {
      const headers = this.parseHeaders(config.customHeadersText);
      // 单个发包：从 URIs 中随机选择一个 URL
      const { fullURL, targetPath } = this.getRandomUriUrl(baseURL, uris);
      const urlObj = new URL(fullURL);
      
      let packet = `GET ${targetPath} HTTP/1.1\r\n`;
      packet += `Host: ${urlObj.host}\r\n`;
      packet += `Cookie: ${session}\r\n`;
      packet += `User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36\r\n`;
      
      for (const [key, value] of Object.entries(headers)) {
        packet += `${key}: ${value}\r\n`;
      }
      
      packet += `\r\n`;
      
      return packet;
    },

    // 发送 Sessions 到 Restmate（通用方法）
    async sendSessionsToRestmate(task, sessions) {
      if (!sessions || sessions.length === 0) {
        ElMessage.warning('没有可发送的 Session');
        return;
      }

      try {
        // 创建分组
        const collectionId = nanoid();
        const collectionName = `Druid Sessions - ${task.url} - ${new Date().toLocaleString()}`;
        
        const createRes = await AddCollection(collectionId, collectionName);
        if (!createRes.success) {
          ElMessage.error('创建分组失败: ' + (createRes.msg || '未知错误'));
          return;
        }

        // 批量创建请求
        const baseURL = task.url.replace(/\/$/, '');
        const headersObj = this.parseHeaders(task.config.customHeadersText);
        
        // 从 baseURL 中提取 host（域名或IP+端口）
        let hostValue = '';
        try {
          const urlObj = new URL(baseURL);
          hostValue = urlObj.host; // 包含端口号，如 example.com:8080
        } catch (e) {
          // 如果解析失败，尝试从 baseURL 中提取
          const match = baseURL.match(/https?:\/\/([^\/]+)/);
          if (match) {
            hostValue = match[1];
          } else {
            hostValue = baseURL.replace(/https?:\/\//, '').split('/')[0];
          }
        }
        
        let successCount = 0;
        let failCount = 0;

        for (const session of sessions) {
          try {
            // 为每个 session 创建独立的 headers 数组，包含 Host 和 Cookie
            const sessionHeadersArray = [
              {
                id: nanoid(),
                key: 'Host',
                value: hostValue,
                active: true
              },
              {
                id: nanoid(),
                key: 'Cookie',
                value: session,
                active: true
              },
              {
                id: nanoid(),
                key: 'User-Agent',
                value: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
                active: true
              },
              // 添加自定义 headers
              ...Object.entries(headersObj).map(([key, value]) => ({
                id: nanoid(),
                key: key,
                value: value,
                active: true
              }))
            ];

            // 批量发包：请求路径使用 {{baseurl}} 占位符，Host 使用实际域名/IP
            // 请求行：GET {{baseurl}} HTTP/1.1（支持相对路径或完整 URL）
            let requestPacket = `GET {{baseurl}} HTTP/1.1\r\n`;
            requestPacket += `Host: ${hostValue}\r\n`;
            requestPacket += `Cookie: ${session}\r\n`;
            requestPacket += `User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36\r\n`;
            
            for (const [key, val] of Object.entries(headersObj)) {
              requestPacket += `${key}: ${val}\r\n`;
            }
            
            requestPacket += `\r\n`;

            // URL 字段：使用 http://host/{{baseurl}} 格式，确保 buildRequestPacket 能正确解析
            // 如果用户设置相对路径（如 /jld-ops/sys/tenant/queryList），会组合为 http://host/jld-ops/sys/tenant/queryList
            // buildRequestPacket 会从 URL 中提取路径部分（/jld-ops/sys/tenant/queryList），构建正确的请求包
            // 如果用户设置完整 URL，会导致重复，但 buildRequestPacket 会正确提取路径部分
            const requestId = nanoid();
            const request = {
              id: requestId,
              coll_id: collectionId,
              name: `Druid Session - ${session.substring(0, 30)}...`,
              method: 'GET',
              url: `http://${hostValue}/{{baseurl}}`,
              requestPacket: requestPacket,
              responsePacket: '',
              headers: sessionHeadersArray,
              params: [],
              body: {
                bodyType: 'json',
                bodyRaw: '',
                formData: []
              },
              description: `从 Druid 扫描导入: ${task.url}`
            };

            const upsertRes = await UpsertRequest(request);
            if (upsertRes.success) {
              successCount++;
            } else {
              failCount++;
              console.error(`添加请求失败: ${session}`, upsertRes.msg);
            }
          } catch (error) {
            failCount++;
            console.error(`处理 Session 失败: ${session}`, error);
          }
        }

        // 清空选择
        this.clearSelectedSessions(task.id);

        // 跳转到 Restmate
        this.$router.push({ path: '/restmate' });
        
        if (failCount === 0) {
          ElMessage.success(`成功导入 ${successCount} 个 Session 到分组「${collectionName}」`);
        } else {
          ElMessage.warning(`导入完成：成功 ${successCount} 个，失败 ${failCount} 个`);
        }
      } catch (error) {
        console.error('批量发送失败:', error);
        ElMessage.error('批量发送失败: ' + (error.message || error));
      }
    },

    // 导出数据
    exportToExcel(taskId) {
      const task = this.tasks.find(t => t.id === taskId);
      if (!task || !task.result || !task.result.success) {
        ElMessage.warning('没有可导出的数据');
        return;
      }

      const data = {
        url: task.url,
        scanTime: new Date(task.updateTime).toISOString(),
        sessions: task.result.sessions || [],
        sqls: task.result.sqls || [],
        uris: task.result.uris || [],
        jdbcInfo: task.result.jdbcInfo || {},
        basicInfo: task.result.basicInfo || {},
      };

      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `druid_scan_${taskId}_${Date.now()}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      ElMessage.success('导出成功');
    },

    // 分页相关方法
    getPaginationConfig(taskId, tabName) {
      if (!this.paginationConfig[taskId]) {
        this.paginationConfig[taskId] = {};
      }
      if (!this.paginationConfig[taskId][tabName]) {
        this.paginationConfig[taskId][tabName] = {
          page: 1,
          pageSize: this.defaultPageSize
        };
      }
      return this.paginationConfig[taskId][tabName];
    },

    ensureTaskColFilters(task) {
      if (!task) return
      if (!task.colFilters) task.colFilters = {}
    },
    applyValueColFilter(taskId, tabName, dataList) {
      if (!dataList || !Array.isArray(dataList)) return []
      const task = this.tasks.find(t => t.id === taskId)
      if (!task) return [...dataList]
      this.ensureTaskColFilters(task)
      const q = (task.colFilters[tabName] || '').trim().toLowerCase()
      if (!q) return [...dataList]
      return dataList.filter(row => String(row?.value ?? '').toLowerCase().includes(q))
    },
    getColFilteredTotal(taskId, tabName, dataList) {
      return this.applyValueColFilter(taskId, tabName, dataList || []).length
    },
    handleDruidColFilter(taskId, tabName) {
      const c = this.getPaginationConfig(taskId, tabName)
      c.page = 1
    },
    getPaginatedData(taskId, tabName, dataList) {
      if (!dataList || !Array.isArray(dataList)) return []
      const list = this.applyValueColFilter(taskId, tabName, dataList)
      const config = this.getPaginationConfig(taskId, tabName)
      const start = (config.page - 1) * config.pageSize
      const end = start + config.pageSize
      return list.slice(start, end)
    },

    handlePageSizeChange(taskId, tabName, pageSize) {
      const config = this.getPaginationConfig(taskId, tabName);
      config.pageSize = pageSize;
      config.page = 1;
    },

    handlePageChange(taskId, tabName, page) {
      const config = this.getPaginationConfig(taskId, tabName);
      config.page = page;
    },

    getActiveResultTab(taskId) {
      return this.activeResultTab[taskId] || 'sessions';
    },

    setActiveResultTab(taskId, tabName) {
      this.activeResultTab[taskId] = tabName;
    },

    // 任务管理相关方法
    async handleHistoryDialogOpened() {
      await this.refreshTasks(false);
    },

    handleHistoryDialogClosed() {
      this.tasksLoading = false;
    },

    async refreshTasks(force = false) {
      if (this.tasksLoading && !force) {
        return;
      }
      
      this.tasksLoading = true;
      try {
        // 从 localStorage 加载所有任务
        const tasksStr = localStorage.getItem('druidTasks');
        if (tasksStr) {
          const tasksData = JSON.parse(tasksStr);
          this.allTasks = Object.values(tasksData)
            .map(task => ({
              id: task.id,
              url: task.url || '',
              status: task.status || 'pending',
              createTime: task.createTime || 0,
              updateTime: task.updateTime || 0,
            }))
            .sort((a, b) => (b.updateTime || 0) - (a.updateTime || 0));
        } else {
          this.allTasks = [];
        }
        
        this.taskPagination.currentPage = 1;
      } catch (error) {
        console.error('刷新任务列表失败:', error);
        ElMessage.error('刷新任务列表失败: ' + error);
        this.allTasks = [];
      } finally {
        this.tasksLoading = false;
      }
    },

    async loadTaskData(taskId) {
      try {
        let task = this.tasksLocal[taskId];
        if (!task) {
          // 尝试从 localStorage 加载
          const tasksStr = localStorage.getItem('druidTasks');
          if (tasksStr) {
            const tasksData = JSON.parse(tasksStr);
            task = tasksData[taskId];
            if (task) {
              this.tasksLocal[taskId] = task;
            }
          }
        }
        
        if (!task) {
          ElMessage.warning('任务不存在');
          return;
        }

        // 如果任务有结果文件路径，从文件加载完整结果
        let result = null;
        if (task.resultsFilePath) {
          const fileData = await this.loadResultsFromFile(task.resultsFilePath);
          if (fileData && fileData.result) {
            result = fileData.result;
          }
        } else if (task.result) {
          // 兼容旧数据：如果直接有 result 字段，使用它
          result = task.result;
        }

        // 构建完整的任务对象
        const fullTask = {
          ...task,
          status: task.status || 'pending',
          result: result,
          config: task.config || {
            username: '',
            password: '',
            customHeadersText: '',
            timeout: 10,
          },
          colFilters: task.colFilters || {}
        };

        // 检查是否已存在该任务
        const existingTask = this.tasks.find(t => t.id === taskId);
        if (existingTask) {
          // 更新现有任务
          Object.assign(existingTask, fullTask);
          this.activeTaskId = taskId;
          ElMessage.success('任务数据已加载');
        } else {
          // 创建新任务
          this.tasks.push(fullTask);
          this.tasksLocal[taskId] = fullTask;
          this.activeTaskId = taskId;
          if (!this.activeResultTab[taskId]) {
            this.activeResultTab[taskId] = 'sessions';
          }
          ElMessage.success('任务数据已加载');
        }

        this.historyDialog = false;
      } catch (error) {
        console.error('加载任务数据失败:', error);
        ElMessage.error('加载任务数据失败: ' + error);
      }
    },

    async deleteTaskFromHistory(taskId) {
      try {
        await ElMessageBox.confirm('确定要删除该任务吗？此操作不可恢复。', '确认删除', { type: 'warning' });
        this.tasks = this.tasks.filter(t => t.id !== taskId);
        delete this.tasksLocal[taskId];
        delete this.activeResultTab[taskId];
        delete this.paginationConfig[taskId];
        delete this.selectedUris[taskId];
        delete this.urisTableRefs[taskId];
        delete this.selectedSessions[taskId];
        delete this.sessionsTableRefs[taskId];
        
        // 从任务管理列表中移除
        this.allTasks = this.allTasks.filter(t => t.id !== taskId);
        
        // 保存到 localStorage
        await this.saveTasks();
        
        // 如果删除的是当前活动任务，切换到其他任务
        if (this.activeTaskId === taskId) {
          this.activeTaskId = this.tasks.length > 0 ? this.tasks[0].id : '';
        }
        
        ElMessage.success('任务已删除');
      } catch (error) {
        if (error !== 'cancel') {
          console.error('删除任务失败:', error);
          ElMessage.error('删除任务失败: ' + error);
        }
      }
    },

    async clearAllTasks() {
      try {
        await ElMessageBox.confirm('确定要清除所有任务吗？此操作不可恢复。', '确认清除', { type: 'warning' });
        this.tasks = [];
        this.tasksLocal = {};
        this.allTasks = [];
        this.activeTaskId = '';
        this.activeResultTab = {};
        this.paginationConfig = {};
        
        // 清空 localStorage
        localStorage.removeItem('druidTasks');
        
        ElMessage.success('所有任务已清除');
      } catch (error) {
        if (error !== 'cancel') {
          console.error('清除任务失败:', error);
        }
      }
    },

    handleTaskPageSizeChange(pageSize) {
      this.taskPagination.pageSize = pageSize;
      this.taskPagination.currentPage = 1;
    },

    handleTaskPageChange(page) {
      this.taskPagination.currentPage = page;
    },

    getStatusTagType(status) {
      const statusMap = {
        'running': 'warning',
        'completed': 'success',
        'stopped': 'info',
        'error': 'danger',
        'pending': 'info',
        'starting': 'warning',
        'stopping': 'warning'
      };
      return statusMap[status] || 'info';
    },

    getStatusText(status) {
      const statusMap = {
        'running': '运行中',
        'completed': '已完成',
        'stopped': '已停止',
        'error': '错误',
        'pending': '待开始',
        'starting': '启动中',
        'stopping': '停止中'
      };
      return statusMap[status] || status || '未知';
    },

    formatTime(time) {
      if (!time) return '-';
      try {
        const date = new Date(time);
        return date.toLocaleString('zh-CN', {
          year: 'numeric',
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit'
        });
      } catch (e) {
        return time;
      }
    },
  },
  watch: {
    tasks: {
      handler() {
        // 当任务变化时自动保存（使用防抖，避免频繁保存）
        if (this.saveTimer) {
          clearTimeout(this.saveTimer);
        }
        this.saveTimer = setTimeout(() => {
          this.saveTasks();
        }, 500); // 500ms 防抖延迟
      },
      deep: true
    }
  },
  beforeUnmount() {
    // 组件销毁前清除定时器
    if (this.saveTimer) {
      clearTimeout(this.saveTimer);
    }
  }
};
</script>

<style lang="scss" scoped>
.browser-scan-page {
  height: 100%;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-sizing: border-box;
}

// 顶部工具栏，与目录扫描等保持一致
.top-toolbar {
  flex-shrink: 0;
  margin-bottom: 16px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid #ebeef5;
  
  .toolbar-content {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .toolbar-left {
    display: flex;
    align-items: center;
    
    .module-title {
      display: flex;
      align-items: center;
      gap: 10px;
      
      .title-icon {
        font-size: 20px;
        color: #409eff;
      }
      
      .title-text {
        font-size: 16px;
        font-weight: 600;
        color: #303133;
        letter-spacing: 0.5px;
      }
    }
  }
  
  .toolbar-right {
    display: flex;
    align-items: center;
    gap: 10px;
  }
}

.glass-card {
  border: 1px solid rgba(255, 255, 255, 0.55);
  box-shadow: 0 10px 28px rgba(69, 107, 183, 0.18);
  backdrop-filter: blur(3px);
}

// 标签页样式
.panel-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  
  :deep(.el-tabs) {
    flex: 1;
    min-height: 0;
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  
  :deep(.el-tabs__content) {
    flex: 1;
    min-height: 0;
    display: flex;
    padding: 0px;
    flex-direction: column;
  }
  
  :deep(.el-tab-pane) {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  :deep(.el-tab) {
    min-height: 400px;
  }
}

.main-tabs {
  min-height: 0;
}

.task-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.config-section {
  flex-shrink: 0;
  background-color: #ffffff;
}

.results-section {
  flex: 1;
  min-height: 500px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* 与 PasswordCrackView 统一：结果表密度与表头 */
.results-section :deep(.el-table) {
  font-size: 12px;
}
.results-section :deep(.el-table th.el-table__cell) {
  background-color: #f5f7fa;
  padding: 8px 0;
  font-weight: 600;
}
.results-section :deep(.el-table td.el-table__cell) {
  padding: 6px 0;
}
.results-section :deep(.el-table__empty-block) {
  padding: 60px 0;
}

.config-card {
  border-radius: 10px;
  background: transparent;
}

.config-card :deep(.el-card__body) {
  padding: 12px;
}

.config-layout {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.url-input-section {
  width: 280px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.url-label {
  font-weight: 600;
  color: #303133;
  font-size: 13px;
}

.url-actions {
  display: flex;
  gap: 8px;
}

.config-section-right {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.config-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.config-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px;
}

.single-line-config {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 10px;
  align-items: start;
}

.config-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.config-item.full-width {
  grid-column: 1 / -1;
}

.config-label {
  font-size: 12px;
  color: #606266;
  font-weight: 500;
}

.results-card {
  height: 100%;
  min-height: 460px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.results-card :deep(.el-card__body) {
  height: 100%;
  min-height: 0;
  overflow: hidden;
  box-sizing: border-box;
  padding: 5px;
}

.result-tabs {
  height: 100%;
  min-height: 460px;
  display: flex;
  flex-direction: column;
}

.result-tabs :deep(.el-tabs__content) {
  flex: 1;
  min-height: 460px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.result-tabs :deep(.el-tab-pane) {
  flex: 1;
  min-height: 460px;
  overflow: auto;
}

.info-display {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 15px;
  background-color: rgba(0, 0, 0, 0.02);
  border-radius: 4px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px;
  background: #f5f7fa;
  border-radius: 4px;
}

.info-label {
  font-weight: 500;
  min-width: 150px;
  color: #606266;
}

.info-value {
  flex: 1;
  word-break: break-all;
  color: #303133;
}

.empty-info {
  padding: 40px 20px;
  text-align: center;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
  
  .tab-status-icon {
    font-size: 14px;
    
    &.running {
      color: #409eff;
      animation: pulse 1.5s ease-in-out infinite;
    }
    
    &.completed {
      color: #67c23a;
    }
    
    &.error {
      color: #f56c6c;
    }
  }
  
  .tab-name {
    max-width: 150px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 8px 10px;
  margin: 0;
  border-top: 1px solid #e4e7ed;
  box-sizing: border-box;
}

// 任务管理对话框样式
:deep(.history-dialog .el-overlay-dialog) {
  display: flex !important;
  align-items: flex-start !important;
  justify-content: center !important;
  width: 100% !important;
}

:deep(.history-dialog .el-dialog) {
  max-width: 90vw !important;
  width: 75% !important;
  margin: 5vh auto 0 !important;
  max-height: 85vh !important;
  display: flex !important;
  flex-direction: column !important;
  overflow: hidden !important;
  position: relative !important;
  left: auto !important;
  right: auto !important;
  transform: none !important;
}

:deep(.history-dialog .el-dialog__header) {
  flex-shrink: 0;
  padding: 20px 20px 10px;
}

:deep(.history-dialog .el-dialog__body) {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 4px;
  box-sizing: border-box;
}

:deep(.history-dialog .el-dialog__footer) {
  flex-shrink: 0;
  padding: 10px 20px 20px;
}

.history-dialog-content {
  display: flex;
  flex-direction: column;
  min-height: 0;
  width: 100%;
  box-sizing: border-box;
}

.history-table {
  flex: 1;
  overflow: auto;
  
  :deep(.el-button--link) {
    padding: 5px 8px;
    min-height: auto;
    font-size: 16px;
  }
  
  :deep(.el-button-group) {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
  
  :deep(.action-button-group) {
    gap: 12px;
  }
}

.history-pagination {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 8px 10px;
  border-top: 1px solid #e4e7ed;
}

.mr-5px {
  margin-right: 5px;
}

.task-name-cell {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

// 标签页右键菜单样式
.tabs-context-menu {
  position: fixed;
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.55);
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.12);
  border-radius: 6px;
  padding: 4px 0;
  z-index: 3000;
  min-width: 120px;
  list-style: none;
  margin: 0;
}

.tabs-context-menu li {
  padding: 8px 16px;
  cursor: pointer;
  font-size: 13px;
  color: #303133;
  transition: background-color 0.2s;
}

.tabs-context-menu li:hover {
  background-color: #f5f7fa;
  color: #409eff;
}

// 表格单元格文本样式
.table-cell-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  width: 100%;
}

// 表格空状态最小高度样式
.results-card :deep(.el-table) {
  min-height: 360px;
}

.results-card :deep(.el-table__empty-block) {
  min-height: 360px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.results-card :deep(.el-table__body-wrapper) {
  min-height: 360px;
}
</style>
