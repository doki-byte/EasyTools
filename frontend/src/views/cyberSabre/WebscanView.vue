<template>
  <div class="webscan-page">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar">
      <div class="toolbar-content">
        <div class="toolbar-left">
          <div class="module-title">
            <el-icon class="title-icon"><Warning /></el-icon>
            <span class="title-text">漏洞扫描</span>
          </div>
          <div class="poc-info">
            <span class="info-label">可用 POC:</span>
            <span class="info-value">{{ dashboard.pocLength }}</span>
          </div>
          <el-button link @click="hotReloadPoc" :loading="reloading" size="small">
            <template #icon>
              <el-icon :size="16"><Refresh /></el-icon>
            </template>
            热加载 POC
          </el-button>
        </div>
        <div class="toolbar-right">
          <el-button 
            :icon="Plus" 
            type="primary" 
            plain
            size="default"
            @click="openNewWebscanDialog" 
            v-if="!form.runnningStatus"
          >
            新建任务
          </el-button>
          <el-button 
            :icon="Clock" 
            type="info" 
            plain
            size="default"
            @click="historyDialog = true"
          >
            任务管理
          </el-button>
          <el-button 
            :icon="Setting" 
            type="info" 
            plain
            size="default"
            @click="showConfigDialog = true"
          >
            扫描配置
          </el-button>
        </div>
      </div>
    </div>
    
    <!-- 整页标签 -->
    <CustomTabs>
      <el-tabs 
        v-model="activeTabId" 
        type="border-card" 
        class="panel-card glass-card main-tabs"
        closable
        @tab-remove="removeTab"
        @tab-click="handleTabClick"
        @contextmenu.prevent="handleTabsContextMenu"
      >
        <el-tab-pane
          v-for="tab in scanTabs"
          :key="tab.id"
          :name="tab.id"
        >
          <template #label>
            <span class="tab-label">
              <el-icon v-if="tab.runningStatus" class="tab-status-icon running">
                <VideoPlay />
              </el-icon>
              <el-icon v-else-if="tab.taskId && getTaskStatus(tab.taskId) === 'completed'" class="tab-status-icon completed">
                <CircleCheck />
              </el-icon>
              <el-icon v-else-if="tab.taskId && getTaskStatus(tab.taskId) === 'stopped'" class="tab-status-icon stopped">
                <VideoPause />
              </el-icon>
              <el-icon v-else-if="tab.taskId && getTaskStatus(tab.taskId) === 'paused'" class="tab-status-icon paused">
                <VideoPause />
              </el-icon>
              <span class="tab-name">{{ tab.name }}</span>
              <el-badge 
                v-if="tab.dashboard.vulnCount > 0" 
                :value="tab.dashboard.vulnCount" 
                :max="99" 
                class="tab-badge"
              />
            </span>
          </template>

          <!-- 每个标签页自己的 Overview 区域 -->
          <el-card class="mb-12 overview-card glass-card">
            <div class="overview-header">
              <div class="overview-title-section">
                <div class="title-with-progress">
                  <span class="title">漏洞扫描中心</span>
                  <div class="progress-inline">
                    <el-icon class="icon icon-red">
                      <Warning />
                    </el-icon>
                    <div class="progress-content">
                      <div class="progress-bar-wrapper">
                        <el-progress 
                          :percentage="getTabProgressPercentage(tab)" 
                          :show-text="false" 
                          :stroke-width="8" 
                          class="inline-progress"
                        />
                        <span class="progress-percentage">{{ getTabProgressPercentage(tab) }}%</span>
                      </div>
                      <div class="progress-meta">
                        <span>目标 {{ tab.progressStats.total }}</span>
                        <span>完成 {{ tab.progressStats.completed }}</span>
                        <span>漏洞 {{ tab.dashboard.vulnCount }}</span>
                      </div>
                    </div>
                  </div>
                </div>
                <span v-if="tab.id === activeTabId" class="tab-indicator">当前任务: {{ tab.name }}</span>
              </div>
              <div class="overview-content-section">
                <div class="risk-level-display">
                  <el-tooltip v-for="(value, level) in tab.dashboard.riskLevel" :key="level" :content="level">
                    <div class="risk-badge" :class="getBadgeClass(level)">
                      {{ value }}
                    </div>
                  </el-tooltip>
                </div>
                <div class="overview-actions">
                  <el-button
                    v-if="tab.runningStatus && getTaskStatus(tab.taskId) !== 'paused'"
                    type="warning"
                    size="small"
                    :icon="VideoPause"
                    @click="pauseScanForTab(tab)"
                  >
                    暂停任务
                  </el-button>
                  <el-button
                    v-if="getTaskStatus(tab.taskId) === 'paused'"
                    type="success"
                    size="small"
                    :icon="VideoPlay"
                    @click="resumeScanForTab(tab)"
                  >
                    恢复任务
                  </el-button>
                  <el-button
                    v-if="tab.runningStatus || getTaskStatus(tab.taskId) === 'paused'"
                    type="danger"
                    size="small"
                    :icon="VideoPause"
                    @click="stopScanForTab(tab)"
                  >
                    停止任务
                  </el-button>
                </div>
              </div>
            </div>
          </el-card>

          <!-- 当前标签页的结果列表 -->
          <div class="filter-toolbar mb-10px" style="flex-shrink: 0;">
            <div class="flex-between">
              <el-space wrap>
                <el-select
                  v-model="currentTab.filterConfig.severity"
                  placeholder="严重程度"
                  clearable
                  style="width: 150px"
                  @change="() => currentTab.pagination.currentPage = 1"
                >
                  <el-option label="全部" value="" />
                  <el-option label="CRITICAL" value="CRITICAL" />
                  <el-option label="HIGH" value="HIGH" />
                  <el-option label="MEDIUM" value="MEDIUM" />
                  <el-option label="LOW" value="LOW" />
                  <el-option label="INFO" value="INFO" />
                </el-select>
                <el-input
                  v-model="currentTab.filterConfig.keyword"
                  placeholder="搜索漏洞名称/URL/Template"
                  clearable
                  style="width: 280px; max-width: 100%;"
                >
                  <template #prefix>
                    <el-icon><Search /></el-icon>
                  </template>
                </el-input>
                <el-select
                  v-model="currentTab.filterConfig.protocol"
                  placeholder="协议"
                  clearable
                  style="width: 140px"
                  @change="() => currentTab.pagination.currentPage = 1"
                >
                  <el-option label="全部" value="" />
                  <el-option label="HTTP" value="HTTP" />
                  <el-option label="HTTPS" value="HTTPS" />
                  <el-option label="其他" value="OTHER" />
                </el-select>
                <el-button :icon="Refresh" @click="resetFilters">重置搜索</el-button>
              </el-space>
              <el-space>
                <el-button :icon="Download" :disabled="!currentTab || currentTab.vp.table.result.length===0" @click="exportToExcel">导出Excel</el-button>
                <el-button :icon="Delete" plain :disabled="!currentTab || currentTab.vp.table.result.length===0" @click="clearResults">清空</el-button>
              </el-space>
            </div>
          </div>
          <div class="table-wrapper" style="flex: 1; min-height: 0; overflow: hidden;">
            <el-table
              class="vuln-table glass-card"
              :data="currentFilteredPageContent"
              stripe
              :height="tableHeight"
              :highlight-current-row="true"
              :header-cell-style="{ 'text-align': 'center' }"
              :row-class-name="({ row }) => getRowSeverityClass(row)"
              @sort-change="(data) => currentTab.vp.ctrl.sortChange(data, true)"
              @row-contextmenu="(row, column, event) => handleRowContextMenu(row, column, event)"
            >
            <el-table-column prop="vuln_name" min-width="220">
              <template #header>
                <TableColumnFilterHeader
                  label="名称"
                  v-model="currentTab.columnFilters.vuln_name"
                  :popover-width="280"
                  @filter-change="onWebscanColumnFilterChange"
                />
              </template>
              <template #default="scope">
                <el-tooltip
                    :content="scope.row.vuln_name || '未知漏洞'"
                    placement="top"
                    :disabled="!scope.row.vuln_name || scope.row.vuln_name.length <= 25"
                    effect="dark"
                >
                  <div class="vuln-name">
                    <div class="vuln-name-label">{{ scope.row.vuln_name || '未知漏洞' }}</div>
                    <div class="vuln-template" v-if="scope.row.template_id">#{{ scope.row.template_id }}</div>
                  </div>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column
              prop="severity"
              width="168"
            >
              <template #header>
                <TableColumnFilterHeader
                  label="Severity"
                  v-model="currentTab.columnFilters.severity"
                  :popover-width="220"
                  @filter-change="onWebscanColumnFilterChange"
                />
              </template>
              <template #default="scope">
                <span class="severity-pill" :class="'severity-pill--' + (scope.row.severity || 'unknown').toLowerCase()">
                  {{ (scope.row.severity || 'UNKNOWN').toUpperCase() }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="vuln_url" min-width="270">
              <template #header>
                <TableColumnFilterHeader
                  label="URL"
                  v-model="currentTab.columnFilters.vuln_url"
                  :popover-width="280"
                  @filter-change="onWebscanColumnFilterChange"
                />
              </template>
              <template #default="scope">
                <el-tooltip
                    :content="scope.row.vuln_url || ''"
                    placement="top"
                    :disabled="!scope.row.vuln_url || scope.row.vuln_url.length <= 30"
                    effect="dark"
                >
                <span class="url-text-row">
                <el-link
                  type="primary"
                  :underline="false"
                  class="url-text"
                  :title="scope.row.vuln_url"
                  @click="BrowserOpenURL(scope.row.vuln_url)"
                >{{ scope.row.vuln_url }}</el-link>
                <el-button :icon="Link" type="primary" link @click="BrowserOpenURL(scope.row.vuln_url)" />
                </span>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column prop="protocol" width="100">
              <template #header>
                <TableColumnFilterHeader
                  label="协议"
                  v-model="currentTab.columnFilters.protocol"
                  :popover-width="200"
                  @filter-change="onWebscanColumnFilterChange"
                />
              </template>
              <template #default="scope">
                <span class="protocol-chip">{{ (scope.row.protocol || '未知').toUpperCase() }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="status_code" width="108" align="center">
              <template #header>
                <TableColumnFilterHeader
                  label="状态码"
                  v-model="currentTab.columnFilters.status_code"
                  :popover-width="200"
                  @filter-change="onWebscanColumnFilterChange"
                />
              </template>
              <template #default="scope">
                <span v-if="scope.row.status_code" class="status-code-badge" :class="getStatusCodeClass(scope.row.status_code)">
                  {{ scope.row.status_code }}
                </span>
                <span v-else class="empty-text">-</span>
              </template>
            </el-table-column>
            <el-table-column prop="response_time" width="92" align="center">
              <template #header>
                <TableColumnFilterHeader
                  label="响应时间"
                  v-model="currentTab.columnFilters.response_time"
                  :popover-width="200"
                  @filter-change="onWebscanColumnFilterChange"
                />
              </template>
              <template #default="scope">
                <span v-if="scope.row.response_time">{{ scope.row.response_time }}s</span>
                <span v-else class="empty-text">-</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="140" align="center">
              <template #default="scope">
                <el-tooltip content="详情">
                  <el-button :icon="Reading" type="primary" link @click="showDetail(scope.row)" />
                </el-tooltip>
                <el-tooltip content="删除">
                  <el-button :icon="Delete" link @click="deleteVuln(scope.row)" />
                </el-tooltip>
              </template>
            </el-table-column>
            <template #empty>
              <el-empty />
            </template>
          </el-table>
          </div>
          <div class="pagination-wrapper">
            <el-pagination small
              v-model:current-page="currentTab.pagination.currentPage"
              v-model:page-size="currentTab.pagination.pageSize"
              :page-sizes="[10, 20, 50, 100, 200]"
              :total="currentFilteredResults.length"
              layout="total, sizes, prev, pager, next, jumper"
              background
              @size-change="handleFilteredSizeChange"
              @current-change="handleFilteredCurrentChange"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </CustomTabs>

    <!-- 新建网站扫描 Dialog -->
    <el-dialog 
      v-model="form.newWebscanDialog" 
      class="new-task-dialog"
      width="70%"
      @opened="handleNewTaskDialogOpened" 
      @closed="handleNewTaskDialogClosed"
    >
      <template #header>
        <div class="dialog-title">
          <el-icon :size="18" class="mr-6px"><Plus /></el-icon>新建网站扫描
        </div>
      </template>
      <el-form label-width="120px" v-if="form.newWebscanDialog">
        <el-form-item label="任务名称:">
          <el-input v-model="form.taskName" />
        </el-form-item>
        <el-form-item label="目标地址:">
          <CustomTextarea v-model="form.input" :rows="6" style="width: 100%;" placeholder="支持 txt / csv，每行一个 URL，将自动去重并合并到输入框" />
        </el-form-item>
        <el-form-item label="追加POC:">
          <el-input v-model="form.appendPoc">
            <template #suffix>
              <el-tooltip content="保存配置">
                <el-button :icon="DocumentAdd" link @click="saveAppendPoc" />
              </el-tooltip>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="模式:">
          <el-select v-model="config.webscanOption">
            <el-option label="指纹漏洞扫描" :value="3" />
            <el-option label="指定漏洞扫描" :value="5" />
            <el-option label="全量漏洞扫描" :value="4" />
          </el-select>
        </el-form-item>
        <el-form-item label="请求头:">
          <el-input v-model="config.customHeaders" type="textarea" :rows="3" placeholder="每行一个请求头" />
        </el-form-item>
        <el-form-item label="指定指纹:" v-show="config.webscanOption === 3">
          <el-select 
            v-model="config.customTags" 
            filterable 
            :filter-method="filterFingerOptions"
            multiple 
            clearable 
            :options="fingerOptions"
            :reserve-keyword="false"
            :loading="fingerOptionsLoading"
            placeholder="请选择指纹"
            @visible-change="handleFingerSelectVisible"
          />
        </el-form-item>
        <el-form-item label="指定漏洞:" v-show="config.webscanOption === 5" class="custom-template-item">
          <el-select 
            v-model="config.customTemplate" 
            filterable 
            :filter-method="filterTemplateOptions"
            multiple 
            clearable 
            :options="templateOptions"
            :reserve-keyword="false"
            :loading="templateOptionsLoading"
            placeholder="请选择漏洞模板"
            @visible-change="handleTemplateSelectVisible"
            class="custom-template-select"
            popper-class="custom-template-popper"
          />
        </el-form-item>
        <el-form-item label="高级配置:">
          <el-checkbox label="无指纹跳过" v-model="config.skipNucleiWithoutTags" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
          <el-button link type="primary" @click="openConfigDialogFromTask">
            <el-icon><Setting /></el-icon>
            扫描配置
          </el-button>
          <div>
            <el-button @click="form.newWebscanDialog=false">取消</el-button>
            <el-button type="primary" @click="startScan">开始任务</el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <!-- 任务管理 -->
    <el-dialog v-model="historyDialog" width="75%" destroy-on-close class="history-dialog" @opened="handleHistoryDialogOpened">
      <template #header>
        <div class="dialog-title" style="display: flex; justify-content: space-between; align-items: center;">
          <div>
            <el-icon :size="18" class="mr-5px"><Clock /></el-icon>任务管理
          </div>
          <div>
            <el-button :icon="Refresh" text @click="refreshTasks" :loading="tasksLoading">刷新</el-button>
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
          <el-table-column prop="name" label="任务名称" width="120">
            <template #default="{ row }">
              <el-tooltip
                  :content="row.name || ''"
                  placement="top"
                  :disabled="!row.name"
                  effect="dark"
              >
                <div class="task-name-cell">
                  {{ row.name || '-' }}
                </div>
              </el-tooltip>
            </template>
          </el-table-column>
        <el-table-column prop="targets" label="目标" width="150">
          <template #default="{ row }">
            <el-tooltip
                :content="row.targets || ''"
                placement="top"
                :disabled="!row.targets || row.targets.length <= 25"
                effect="dark"
            >
              <div class="task-targets-cell">
                {{ row.targets || '-' }}
              </div>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="vuln" label="漏洞" width="80" />
        <el-table-column label="状态" width="100" align="center">
          <template #default="scope">
            <el-tag :type="getStatusTagType(scope.row.status)">
              {{ getStatusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="130">
          <template #default="scope">
            <div v-if="scope.row.status === 'running' || scope.row.status === 'paused'">
              <el-progress 
                :percentage="getTaskProgressPercentage(scope.row)" 
                :stroke-width="8"
                :show-text="true"
                :format="() => `${scope.row.completed || 0}/${scope.row.total || 0}`"
              />
            </div>
            <div v-else-if="scope.row.status === 'completed'">
              <el-progress 
                :percentage="100" 
                :stroke-width="8"
                :show-text="true"
                :format="() => `${scope.row.completed || 0}/${scope.row.total || 0}`"
                status="success"
              />
            </div>
            <span v-else class="empty-text">
              {{ scope.row.status === 'stopped' ? '已停止' : '-' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" align="center">
          <template #default="scope">
            <el-button-group>
              <el-tooltip content="查看结果">
                <el-button :icon="View" link @click="loadTask(scope.row.taskId)" />
              </el-tooltip>
              <el-tooltip content="恢复进度" v-if="scope.row.status === 'running'">
                <el-button :icon="Refresh" link @click="restoreTaskProgress(scope.row.taskId)" />
              </el-tooltip>
              <el-tooltip content="重新执行" v-if="scope.row.status !== 'running' && scope.row.status !== 'paused'">
                <el-button :icon="VideoPlay" link @click="reExecuteTask(scope.row)" />
              </el-tooltip>
              <el-tooltip content="恢复任务" v-if="scope.row.status === 'paused'">
                <el-button :icon="VideoPlay" link @click="resumeTask(scope.row.taskId)" />
              </el-tooltip>
              <el-tooltip content="暂停任务" v-if="scope.row.status === 'running'">
                <el-button :icon="VideoPause" link @click="pauseTask(scope.row.taskId)" />
              </el-tooltip>
              <el-tooltip content="停止任务" v-if="scope.row.status === 'running' || scope.row.status === 'paused'">
                <el-button :icon="VideoPause" link @click="stopTask(scope.row.taskId)" />
              </el-tooltip>
              <el-tooltip content="删除">
                <el-button :icon="Delete" link @click="deleteTask(scope.row.taskId)" />
              </el-tooltip>
            </el-button-group>
          </template>
        </el-table-column>
      </el-table>
        <div class="pagination-wrapper">
          <el-pagination small
            v-model:current-page="taskPagination.currentPage"
            v-model:page-size="taskPagination.pageSize"
            :page-sizes="[10, 20, 50, 100, 200]"
            :total="tasks.length"
            layout="total, sizes, prev, pager, next, jumper"
            background
            @size-change="handleTaskPageSizeChange"
            @current-change="handleTaskPageChange"
          />
        </div>
      </div>
    </el-dialog>

    <!-- 详情对话框 -->
    <el-dialog v-model="detailDialog" title="漏洞详情" width="90%" top="5vh" destroy-on-close>
      <div v-if="selectedRow" class="detail-wrapper">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="漏洞名称">{{ selectedRow.vuln_name }}</el-descriptions-item>
          <el-descriptions-item label="严重程度">
            <el-tag :type="getTagTypeBySeverity(selectedRow.severity)">{{ selectedRow.severity }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="POC ID">{{ selectedRow.template_id }}</el-descriptions-item>
          <el-descriptions-item label="协议">{{ selectedRow.protocol }}</el-descriptions-item>
          <el-descriptions-item label="目标URL" :span="2">
            <el-link type="primary" @click="BrowserOpenURL(selectedRow.vuln_url)">{{ selectedRow.vuln_url }}</el-link>
          </el-descriptions-item>
          <el-descriptions-item v-if="selectedRow.extract" label="用户名/密码" :span="2">
            <el-tag type="success" size="large">{{ selectedRow.extract }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="描述" :span="2">{{ selectedRow.description || '无' }}</el-descriptions-item>
          <el-descriptions-item label="参考" :span="2">{{ selectedRow.reference || '无' }}</el-descriptions-item>
        </el-descriptions>
        <el-divider>数据包
          <el-button 
            v-if="selectedRow.request" 
            type="primary" 
            size="small" 
            style="margin-left: 10px;"
            :icon="Promotion"
            @click="sendToPacketSender"
          >
            发送到数据包发包
          </el-button>
        </el-divider>
        <div class="packet-inline-panels">
          <div class="packet-inline-panel">
            <div class="packet-inline-header">请求</div>
            <VueMonacoEditor
              v-if="monacoReady"
              :value="selectedRow.request || ''"
              :language="detectPacketLanguage(selectedRow.request || '')"
              theme="http-dark-theme"
              height="240px"
              :options="{
                readOnly: true,
                lineNumbers: 'on',
                minimap: { enabled: false },
                automaticLayout: true,
                wordWrap: 'on',
              }"
            />
            <pre v-else class="packet-fallback">{{ selectedRow.request || '' }}</pre>
          </div>
          <div class="packet-inline-panel">
            <div class="packet-inline-header">响应</div>
            <VueMonacoEditor
              v-if="monacoReady"
              :value="selectedRow.response || ''"
              :language="detectPacketLanguage(selectedRow.response || '')"
              theme="http-dark-theme"
              height="300px"
              :options="{
                readOnly: true,
                lineNumbers: 'on',
                minimap: { enabled: false },
                automaticLayout: true,
                wordWrap: 'on',
              }"
            />
            <pre v-else class="packet-fallback">{{ selectedRow.response || '' }}</pre>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 数据包高亮查看 -->
    <PacketViewer
      v-model="packetViewerVisible"
      :request-packet="selectedRow?.request || ''"
      :response-packet="selectedRow?.response || ''"
      title="数据包查看"
      :editable="false"
      :show-send-actions="false"
      :enable-save="false"
    />

    <!-- POC 配置 -->
    <el-dialog v-model="showConfigDialog" title="扫描配置管理" class="config-dialog" width="65%" destroy-on-close @opened="handleConfigDialogOpened">
      <PocConfigPanel ref="pocConfigPanelRef" @close="showConfigDialog = false" />
    </el-dialog>

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

    <!-- 表格行右键菜单 - 发送到简练助手 -->
    <Teleport to="body">
      <div
        v-if="rowContextMenuVisible"
        class="context-menu"
        :style="rowContextMenuStyle"
        @click.stop
      >
        <div 
          v-if="connectModuleType" 
          class="context-menu-item" 
          @click="handleSendToConnectModule"
        >
          <el-icon><Promotion /></el-icon>
          <span>发送到简练助手 - {{ connectModuleName }}</span>
        </div>
        <div v-else class="context-menu-item disabled">
          <span>无法识别的漏洞类型</span>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script>
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount, onActivated, onDeactivated, nextTick } from 'vue'
import dayjs from 'dayjs'
import { useRouter } from 'vue-router'
import { VueMonacoEditor, useMonaco } from '@guolao/vue-monaco-editor'
import {
  VideoPause,
  VideoPlay,
  Plus,
  Link,
  Warning,
  Refresh,
  Filter,
  Reading,
  Delete,
  Download,
  Clock,
  View,
  Setting,
  DocumentAdd,
  Upload,
  Search,
  Promotion,
  CircleCheck
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { EventsOn, EventsOff, BrowserOpenURL } from '../../../wailsjs/runtime'
import {
  StartPocScan,
  StopPocScan,
  PausePocScan,
  ResumePocScan,
  GetFingerPocMap,
  ListPocs,
  ListScanTasks,
  DeleteScanTask,
  GetScanResults,
  SaveAppendPoc,
  GetAppendPoc,
  ClearScanTasks,
  EnsureCrackTaskRecord
} from '../../../wailsjs/go/webscan/PocController'
import { AddPasswordCrackResult } from '../../../wailsjs/go/crack/CrackController'
import { GetPocConfig, ReloadPocTags } from '../../../wailsjs/go/webscan/PocConfigController'
import { GetGlobalProxy } from '../../../wailsjs/go/proxy/ProxyManager'
import { SaveFile, GetAppPath, ReadFile } from '../../../wailsjs/go/controller/Note'
import PocConfigPanel from './PocConfigPanel.vue'
import PacketViewer from '../../components/PacketViewer.vue'
import { registerHTTPLanguage, detectPacketLanguage } from '../../utils/http-language'
import CustomTabs from '../../components/CustomTabs.vue'
import CustomTextarea from '../../components/CustomTextarea.vue'
import TableColumnFilterHeader from '../../components/TableColumnFilterHeader.vue'
import usePagination from '../../utils/usePagination'
import { getTagTypeBySeverity, getBadgeClass } from '../../utils/styleUtils'
import * as XLSX from 'xlsx'

export default {
  name: 'WebscanView',
  computed: {
    DocumentAdd() {
      return DocumentAdd
    },
    Upload() {
      return Upload
    },
    Setting() {
      return Setting
    },
    Clock() {
      return Clock
    },
    Download() {
      return Download
    },
    Reading() {
      return Reading
    },
    VideoPause() {
      return VideoPause
    },
    VideoPlay() {
      return VideoPlay
    },
    Plus() {
      return Plus
    },
    View() {
      return View
    },
    Delete() {
      return Delete
    },
    CircleCheck() {
      return CircleCheck
    }
  },
  components: {
    PocConfigPanel,
    CustomTabs,
    CustomTextarea,
    TableColumnFilterHeader,
    PacketViewer,
    VueMonacoEditor,
    VideoPause,
    VideoPlay,
    Plus,
    Link,
    Warning,
    Refresh,
    Filter,
    Reading,
    Delete,
    Download,
    Clock,
    View,
    Setting,
  DocumentAdd,
  Upload
  },
  setup() {
    const router = useRouter()
    
    // 全局仪表盘（POC数量等）
    const dashboard = reactive({
      riskLevel: {
        CRITICAL: 0,
        HIGH: 0,
        MEDIUM: 0,
        LOW: 0,
        INFO: 0
      },
      nucleiPercentage: 0,
      pocLength: 0,
      vulnCount: 0
    })
    
    // 标签页管理
    const scanTabs = ref([])
    const activeTabId = ref('')
    let tabIdCounter = 0
    
    // 创建新标签页（仅用于网站漏洞扫描）
    const createTab = (name, taskId = '') => {
      const tabId = `tab-${++tabIdCounter}`
      const newTab = {
        id: tabId,
        name: name || `扫描任务 ${tabIdCounter}`,
        taskId: taskId,
        vp: usePagination(20),
        filterConfig: reactive({
          severity: '',
          keyword: '',
          protocol: ''
        }),
        pagination: reactive({
          currentPage: 1,
          pageSize: 10
        }),
        columnFilters: reactive({
          vuln_name: '',
          severity: '',
          vuln_url: '',
          protocol: '',
          status_code: '',
          response_time: ''
        }),
        dashboard: reactive({
          riskLevel: {
            CRITICAL: 0,
            HIGH: 0,
            MEDIUM: 0,
            LOW: 0,
            INFO: 0
          },
          nucleiPercentage: 0,
          vulnCount: 0
        }),
        progressStats: reactive({
          total: 0,
          completed: 0,
          vuln: 0
        }),
        lastProgress: ref(0),
        runningStatus: false
      }
      scanTabs.value.push(newTab)
      activeTabId.value = tabId
      return newTab
    }
    
    // 获取当前活动标签页
    const currentTab = computed(() => {
      return scanTabs.value.find(tab => tab.id === activeTabId.value) || null
    })
    
    // 当前标签页的仪表盘数据
    const currentDashboard = computed(() => {
      return currentTab.value?.dashboard || dashboard
    })
    
    // 当前标签页的进度统计
    const currentProgressStats = computed(() => {
      return currentTab.value?.progressStats || { total: 0, completed: 0, vuln: 0 }
    })
    
    // 实时计算进度百分比
    const computedProgressPercentage = computed(() => {
      const stats = currentProgressStats.value
      if (stats.total > 0) {
        return Math.round((stats.completed / stats.total) * 100)
      }
      // 如果没有总数，使用仪表盘中的百分比
      return currentDashboard.value.nucleiPercentage || 0
    })
    
    // 当前标签页的过滤结果
    const currentFilteredResults = computed(() => {
      if (!currentTab.value) return []
      const tab = currentTab.value
      let results = [...tab.vp.table.result]
      if (tab.filterConfig.severity) {
        results = results.filter(item => (item.severity || '').toUpperCase() === tab.filterConfig.severity.toUpperCase())
      }
      if (tab.filterConfig.keyword) {
        const k = tab.filterConfig.keyword.toLowerCase()
        results = results.filter(item =>
          (item.vuln_name || '').toLowerCase().includes(k) ||
          (item.vuln_url || '').toLowerCase().includes(k) ||
          (item.template_id || '').toLowerCase().includes(k)
        )
      }
      if (tab.filterConfig.protocol) {
        if (tab.filterConfig.protocol === 'OTHER') {
          results = results.filter(item => {
            const proto = (item.protocol || '').toUpperCase()
            return proto !== 'HTTP' && proto !== 'HTTPS'
          })
        } else {
          results = results.filter(item => (item.protocol || '').toUpperCase() === tab.filterConfig.protocol.toUpperCase())
        }
      }
      const cf = tab.columnFilters || {}
      const nq = (cf.vuln_name || '').trim().toLowerCase()
      if (nq) {
        results = results.filter(item =>
          (item.vuln_name || '').toLowerCase().includes(nq) ||
          String(item.template_id || '').toLowerCase().includes(nq)
        )
      }
      const sq = (cf.severity || '').trim().toLowerCase()
      if (sq) {
        results = results.filter(item => (item.severity || '').toLowerCase().includes(sq))
      }
      const uq = (cf.vuln_url || '').trim().toLowerCase()
      if (uq) {
        results = results.filter(item => (item.vuln_url || '').toLowerCase().includes(uq))
      }
      const pq = (cf.protocol || '').trim().toLowerCase()
      if (pq) {
        results = results.filter(item => (item.protocol || '').toLowerCase().includes(pq))
      }
      const scq = (cf.status_code || '').trim().toLowerCase()
      if (scq) {
        results = results.filter(item => String(item.status_code ?? '').toLowerCase().includes(scq))
      }
      const rtq = (cf.response_time || '').trim().toLowerCase()
      if (rtq) {
        results = results.filter(item => String(item.response_time ?? '').toLowerCase().includes(rtq))
      }
      return results
    })
    
    // 当前标签页的分页内容
    const currentFilteredPageContent = computed(() => {
      if (!currentTab.value) return []
      const start = (currentTab.value.pagination.currentPage - 1) * currentTab.value.pagination.pageSize
      return currentFilteredResults.value.slice(start, start + currentTab.value.pagination.pageSize)
    })
    
    // 获取活动标签页名称
    const getActiveTabName = () => {
      return currentTab.value?.name || '无活动标签'
    }
    
    // 获取任务状态
    const getTaskStatus = (taskId) => {
      return tasksLocal.value[taskId]?.status || ''
    }
    
    // 移除标签页
    const removeTab = async (tabId) => {
      const index = scanTabs.value.findIndex(tab => tab.id === tabId)
      if (index === -1) return
      
      const tab = scanTabs.value[index]
      
      // 重要：关闭标签页时，任务数据必须保留在任务管理中，不能删除
      if (tab.taskId) {
        const task = tasksLocal.value[tab.taskId]
        if (task) {
          // 保存标签页中的结果数据到任务中
          // WebscanView 的结果存储在 tab.vp.table.result 中
          if (tab.vp && tab.vp.table && tab.vp.table.result) {
            // 保存结果数量
            task.resultsCount = tab.vp.table.result.length || 0
            
            // 保存进度统计信息
            if (tab.progressStats) {
              task.total = tab.progressStats.total || 0
              task.completed = tab.progressStats.completed || 0
            }
            
            // 保存仪表板数据
            if (tab.dashboard) {
              task.dashboard = JSON.parse(JSON.stringify(tab.dashboard))
            }
          }
          
          // 如果正在扫描，保持任务状态为 running，不改为 stopped
          if (tab.runningStatus && task.status !== 'running') {
            task.status = 'running'
          } else if (!tab.runningStatus && task.status === 'running') {
            // 如果标签页已停止扫描，但任务状态还是running，保持running状态
            // 因为可能在其他地方还在运行
          }
          
          // 保存任务数据到localStorage
          await saveTasksToStorage()
        }
        // 不断开关联，让任务继续在后台运行，任务数据保留在任务管理中
      }
      
      // 如果关闭的是当前活动标签页，切换到其他标签页
      if (activeTabId.value === tabId) {
        if (scanTabs.value.length > 1) {
          // 优先切换到下一个标签页，如果没有则切换到上一个
          const nextIndex = index < scanTabs.value.length - 1 ? index + 1 : index - 1
          activeTabId.value = scanTabs.value[nextIndex].id
        }
      }
      
      scanTabs.value.splice(index, 1)
      
      // 如果没有标签页了，清空活动标签页ID
      if (scanTabs.value.length === 0) {
        // 避免进入无标签空白态，导致后续弹层/布局异常
        createTab('新建扫描')
      }
    }
    
    // 标签页点击事件
    const handleTabClick = (tab) => {
      activeTabId.value = tab.paneName
    }
    
    // 标签页右键菜单
    const tabsContextMenuVisible = ref(false)
    const tabsContextMenuStyle = reactive({ left: '0px', top: '0px' })
    const tabsContextMenuTabId = ref('')
    
    // 表格行右键菜单
    const rowContextMenuVisible = ref(false)
    const rowContextMenuStyle = reactive({ left: '0px', top: '0px' })
    const selectedVulnRow = ref(null)
    const connectModuleType = ref(null) // 'ssh', 'redis', 'ftp', 'mysql', 'postgresql', 'mongodb', 'mssql', 'oracle'
    const connectModuleName = ref('')
    
    const handleTabsContextMenu = (event) => {
      event.preventDefault()
      event.stopPropagation()
      
      // 检查是否点击在标签页上
      const tabItem = event.target.closest('.el-tabs__item')
      if (!tabItem) {
        hideTabsContextMenu()
        return
      }
      
      const tabIdAttr = tabItem.id || tabItem.getAttribute('aria-controls')
      tabsContextMenuTabId.value = tabIdAttr
        ? tabIdAttr.replace(/^tab-/, '').replace(/^pane-/, '')
        : activeTabId.value

      // 直接从事件获取坐标，优先使用 clientX/clientY（相对于视口，适用于 position: fixed）
      const x = event.clientX || 0
      const y = event.clientY || 0
      
      // 直接设置坐标，确保菜单显示在鼠标点击位置
      tabsContextMenuStyle.left = `${x}px`
      tabsContextMenuStyle.top = `${y}px`
      tabsContextMenuVisible.value = true
      
      // 点击其他地方时隐藏菜单
      const hideMenu = (e) => {
        if (!e.target.closest('.tabs-context-menu')) {
          hideTabsContextMenu()
          document.removeEventListener('click', hideMenu)
          document.removeEventListener('contextmenu', hideMenu)
        }
      }
      setTimeout(() => {
        document.addEventListener('click', hideMenu)
        document.addEventListener('contextmenu', hideMenu)
      }, 100)
    }
    
    const hideTabsContextMenu = () => {
      tabsContextMenuVisible.value = false
    }
    
    // 表格行右键菜单处理
    const handleRowContextMenu = (row, column, event) => {
      event.preventDefault()
      event.stopPropagation()
      
      selectedVulnRow.value = row
      
      // 识别漏洞类型
      const moduleInfo = identifyVulnerabilityType(row)
      connectModuleType.value = moduleInfo.type
      connectModuleName.value = moduleInfo.name
      
      // 先按鼠标位置显示菜单
      const x = event.clientX || 0
      const y = event.clientY || 0
      
      rowContextMenuStyle.left = `${x}px`
      rowContextMenuStyle.top = `${y}px`
      rowContextMenuVisible.value = true
      
      // 再基于实际菜单高度轻微修正，防止刚好溢出窗口底部
      nextTick(() => {
        const menuEl = document.querySelector('.context-menu')
        if (!menuEl) return
        
        const menuHeight = menuEl.offsetHeight || 0
        const windowHeight = window.innerHeight
        if (y + menuHeight > windowHeight - 8) {
          let adjustedY = windowHeight - menuHeight - 8
          if (adjustedY < 8) adjustedY = 8
          rowContextMenuStyle.top = `${adjustedY}px`
        }
      })
      
      // 点击其他地方时隐藏菜单
      const hideMenu = (e) => {
        if (!e.target.closest('.context-menu')) {
          rowContextMenuVisible.value = false
          document.removeEventListener('click', hideMenu)
          document.removeEventListener('contextmenu', hideMenu)
        }
      }
      setTimeout(() => {
        document.addEventListener('click', hideMenu)
        document.addEventListener('contextmenu', hideMenu)
      }, 100)
    }
    
    // 识别漏洞类型
    const identifyVulnerabilityType = (row) => {
      const vulnName = (row.vuln_name || '').toLowerCase()
      const protocol = (row.protocol || '').toLowerCase()
      const vulnUrl = (row.vuln_url || '').toLowerCase()
      
      // 识别SSH相关漏洞
      if (protocol === 'ssh' || vulnName.includes('ssh') || vulnUrl.includes('ssh://')) {
        return { type: 'ssh', name: 'SSH' }
      }
      
      // 识别Redis相关漏洞
      if (protocol === 'redis' || vulnName.includes('redis') || vulnUrl.includes('redis://')) {
        return { type: 'redis', name: 'Redis' }
      }
      
      // 识别FTP相关漏洞
      if (protocol === 'ftp' || vulnName.includes('ftp') || vulnUrl.includes('ftp://')) {
        return { type: 'ftp', name: 'FTP' }
      }
      
      // 识别MySQL相关漏洞
      if (protocol === 'mysql' || vulnName.includes('mysql') || vulnUrl.includes('mysql://')) {
        return { type: 'mysql', name: 'MySQL' }
      }
      
      // 识别PostgreSQL相关漏洞
      if (protocol === 'postgresql' || vulnName.includes('postgresql') || vulnName.includes('postgres') || vulnUrl.includes('postgresql://') || vulnUrl.includes('postgres://')) {
        return { type: 'postgresql', name: 'PostgreSQL' }
      }
      
      // 识别MongoDB相关漏洞
      if (protocol === 'mongodb' || vulnName.includes('mongodb') || vulnUrl.includes('mongodb://')) {
        return { type: 'mongodb', name: 'MongoDB' }
      }
      
      // 识别SQL Server相关漏洞
      if (protocol === 'mssql' || vulnName.includes('mssql') || vulnName.includes('sql server') || vulnUrl.includes('mssql://')) {
        return { type: 'mssql', name: 'SQL Server' }
      }
      
      // 识别Oracle相关漏洞
      if (protocol === 'oracle' || vulnName.includes('oracle') || vulnUrl.includes('oracle://')) {
        return { type: 'oracle', name: 'Oracle' }
      }
      
      return { type: null, name: '' }
    }
    
    // 从漏洞结果中提取连接信息
    const extractConnectionInfo = (row) => {
      const info = {
        host: '',
        port: '',
        username: '',
        password: '',
        name: ''
      }
      
      // 识别漏洞类型（从row中直接识别，不依赖外部变量）
      const vulnName = (row.vuln_name || '').toLowerCase()
      const protocol = (row.protocol || '').toLowerCase()
      const vulnUrl = (row.vuln_url || '').toLowerCase()
      const isRedisType = protocol === 'redis' || vulnName.includes('redis') || vulnUrl.includes('redis://')
      
      // 从vuln_url中提取host和port
      if (vulnUrl) {
        try {
          // 处理各种URL格式: http://host:port, ssh://host:port, redis://host:port, mysql://host:port等
          // 匹配协议://主机:端口 或 协议://用户名:密码@主机:端口
          let urlMatch = vulnUrl.match(/^(?:[^:]+:\/\/)?(?:([^:@]+):([^@]+)@)?([^:/]+)(?::(\d+))?/i)
          if (urlMatch) {
            // 如果有用户名密码在URL中
            if (urlMatch[1] && urlMatch[2]) {
              info.username = urlMatch[1]
              info.password = urlMatch[2]
            }
            // 主机地址
            info.host = urlMatch[3] || ''
            // 端口号
            if (urlMatch[4]) {
              info.port = urlMatch[4]
            }
          }
          // 如果上面的正则没匹配到，尝试简单匹配
          if (!info.host) {
            urlMatch = vulnUrl.match(/(?:^|\/\/)([^:/@]+)(?::(\d+))?/)
            if (urlMatch) {
              info.host = urlMatch[1]
              if (urlMatch[2]) {
                info.port = urlMatch[2]
              }
            }
          }
        } catch (e) {
          console.error('解析URL失败:', e)
        }
      }
      
      // 从extract字段提取用户名和密码（如果存在）
      const extract = row.extract || ''
      if (extract) {
        try {
          // 尝试解析JSON格式的extract
          if (extract.startsWith('{')) {
            const extractData = JSON.parse(extract)
            info.username = extractData.username || extractData.user || extractData.login || info.username
            info.password = extractData.password || extractData.pass || extractData.pwd || info.password
          } else {
            // 尝试多种格式：
            // 1. username/password 格式 (如: root/123456)
            // 2. username:password 格式
            // 3. user=xxx,pass=xxx 格式
            // 4. username: xxx, password: xxx 格式
            // 5. 对于Redis：纯密码格式（不包含/或:）
            
            // 格式1: username/password (如: root/123456)
            const slashMatch = extract.match(/^([^/]+)\/(.+)$/)
            if (slashMatch && !info.username && !info.password) {
              info.username = slashMatch[1].trim()
              info.password = slashMatch[2].trim()
            } else {
              // 格式2: username:password
              const colonMatch = extract.match(/^([^:]+):([^:]+)$/)
              if (colonMatch && !info.username && !info.password) {
                info.username = colonMatch[1].trim()
                info.password = colonMatch[2].trim()
              } else {
                // 格式3: user=xxx,pass=xxx 或 username=xxx,password=xxx
                const userPassMatch = extract.match(/(?:username|user|login)[=:]\s*([^,\s]+)/i)
                const passMatch = extract.match(/(?:password|pass|pwd)[=:]\s*([^,\s]+)/i)
                if (userPassMatch && !info.username) info.username = userPassMatch[1].trim()
                if (passMatch && !info.password) info.password = passMatch[1].trim()
                
                // 格式5: 对于Redis等只需要密码的服务，如果extract不包含/或:，且密码为空，直接作为密码
                // 这是Redis的特殊处理：extract字段通常只包含密码，不包含用户名
                if (isRedisType && !info.password && !extract.includes('/') && !extract.includes(':')) {
                  info.password = extract.trim()
                  console.log('[extractConnectionInfo] Redis密码提取:', extract.trim())
                }
              }
            }
            
            // 如果仍未提取到密码，且是Redis类型，尝试直接将extract作为密码
            // Redis的extract字段通常直接就是密码（如: "123456"）
            if (isRedisType && !info.password && extract.trim()) {
              info.password = extract.trim()
              console.log('[extractConnectionInfo] Redis密码提取（备用）:', extract.trim())
            }
          }
        } catch (e) {
          console.error('解析extract失败:', e)
        }
      }
      
      // 从description字段提取用户名和密码（如果还未提取到）
      // 对于Redis等只需要密码的服务，只检查密码即可
      const needsPassword = isRedisType ? !info.password : (!info.username || !info.password)
      
      if (needsPassword && row.description) {
        try {
          const desc = row.description || ''
          // 尝试从描述中提取 (如: "用户名/密码: root/123456" 或 "密码: 123456")
          if (isRedisType && !info.password) {
            // Redis专用：尝试提取纯密码格式
            // 格式: "密码: 123456" 或 "password: 123456" 等
            const redisPassMatch = desc.match(/(?:密码|password|pass|pwd)[:：]\s*([^\s,，]+)/i)
            if (redisPassMatch) {
              info.password = redisPassMatch[1].trim()
              console.log('[extractConnectionInfo] 从description提取Redis密码:', redisPassMatch[1].trim())
            }
          }
          // 通用格式：用户名/密码
          if (!info.password || (!isRedisType && !info.username)) {
            const descMatch = desc.match(/(?:用户名[\/:]密码|username[\/:]password|user[\/:]pass)[:：]\s*([^\/\s]+)[\/:]([^\s]+)/i)
            if (descMatch) {
              if (!info.username && !isRedisType) info.username = descMatch[1].trim()
              if (!info.password) {
                info.password = descMatch[2].trim()
                console.log('[extractConnectionInfo] 从description提取密码:', descMatch[2].trim())
              }
            }
          }
        } catch (e) {
          console.error('从description提取失败:', e)
        }
      }
      
      // 设置默认端口
      if (!info.port) {
        const defaults = {
          ssh: '22',
          redis: '6379',
          ftp: '21',
          mysql: '3306',
          postgresql: '5432',
          mongodb: '27017',
          mssql: '1433',
          oracle: '1521'
        }
        // 使用识别的类型或外部变量（如果存在）
        const moduleType = isRedisType ? 'redis' : 
                          (protocol === 'ssh' || vulnName.includes('ssh') || vulnUrl.includes('ssh://')) ? 'ssh' :
                          (protocol === 'ftp' || vulnName.includes('ftp') || vulnUrl.includes('ftp://')) ? 'ftp' :
                          connectModuleType.value || ''
        info.port = defaults[moduleType] || ''
      }
      
      // 生成连接名称（使用识别的类型或外部变量）
      const moduleName = isRedisType ? 'Redis' : 
                        (protocol === 'ssh' || vulnName.includes('ssh') || vulnUrl.includes('ssh://')) ? 'SSH' :
                        (protocol === 'ftp' || vulnName.includes('ftp') || vulnUrl.includes('ftp://')) ? 'FTP' :
                        connectModuleName.value || '未知'
      info.name = `${moduleName} - ${info.host}:${info.port}`
      
      console.log('[extractConnectionInfo] 提取的连接信息:', { ...info, password: info.password ? '***' : '(空)' })
      
      return info
    }
    
    // 发送到简练助手对应模块
    const handleSendToConnectModule = () => {
      rowContextMenuVisible.value = false
      
      if (!selectedVulnRow.value || !connectModuleType.value) {
        ElMessage.warning('无法识别漏洞类型')
        return
      }
      
      // 提取连接信息
      const connectionInfo = extractConnectionInfo(selectedVulnRow.value)
      
      if (!connectionInfo.host) {
        ElMessage.warning('无法从漏洞结果中提取连接信息')
        return
      }
      
      // 跳转到简练助手页面
      router.push('/Connect').then(() => {
        // 等待页面加载后，通过事件通知子组件打开对话框
        setTimeout(() => {
          // 触发全局事件，通知简练助手模块打开对话框并填充数据
          window.dispatchEvent(new CustomEvent('open-connect-dialog', {
            detail: {
              module: connectModuleType.value,
              connectionInfo: connectionInfo
            }
          }))
          ElMessage.success(`已发送到${connectModuleName.value}模块`)
        }, 300)
      }).catch(err => {
        console.error('跳转失败:', err)
        ElMessage.error('跳转到简练助手失败')
      })
    }
    
    // 关闭其他标签页
    const handleCloseOtherTabs = async () => {
      hideTabsContextMenu()
      
      const targetId = tabsContextMenuTabId.value || activeTabId.value
      const targetTab = scanTabs.value.find(tab => tab.id === targetId)
      
      if (!targetTab) return
      
      // 关闭其他标签页，保存任务数据
      const tabsToClose = scanTabs.value.filter(tab => tab.id !== targetId)
      for (const tab of tabsToClose) {
        // 重要：关闭标签页时，任务数据必须保留在任务管理中，不能删除
        if (tab.taskId) {
          const task = tasksLocal.value[tab.taskId]
          if (task) {
            // 保存标签页中的结果数据到任务中
            if (tab.vp && tab.vp.table && tab.vp.table.result) {
              // 保存结果数量
              task.resultsCount = tab.vp.table.result.length || 0
              
              // 保存进度统计信息
              if (tab.progressStats) {
                task.total = tab.progressStats.total || 0
                task.completed = tab.progressStats.completed || 0
              }
              
              // 保存仪表板数据
              if (tab.dashboard) {
                task.dashboard = JSON.parse(JSON.stringify(tab.dashboard))
              }
            }
            
            // 如果正在扫描，保持任务状态为 running，不改为 stopped
            if (tab.runningStatus && task.status !== 'running') {
              task.status = 'running'
            }
            
            // 保存任务数据到localStorage
            await saveTasksToStorage()
          }
          // 不断开关联，让任务继续在后台运行，任务数据保留在任务管理中
        }
      }
      
      // 关闭其他标签页
      scanTabs.value = scanTabs.value.filter(tab => tab.id === targetId)
      activeTabId.value = targetId
    }
    
    // 关闭所有标签页
    const handleCloseAllTabs = async () => {
      hideTabsContextMenu()
      
      // 关闭所有标签页，保存任务数据
      for (const tab of scanTabs.value) {
        // 重要：关闭标签页时，任务数据必须保留在任务管理中，不能删除
        if (tab.taskId) {
          const task = tasksLocal.value[tab.taskId]
          if (task) {
            // 保存标签页中的结果数据到任务中
            if (tab.vp && tab.vp.table && tab.vp.table.result) {
              // 保存结果数量
              task.resultsCount = tab.vp.table.result.length || 0
              
              // 保存进度统计信息
              if (tab.progressStats) {
                task.total = tab.progressStats.total || 0
                task.completed = tab.progressStats.completed || 0
              }
              
              // 保存仪表板数据
              if (tab.dashboard) {
                task.dashboard = JSON.parse(JSON.stringify(tab.dashboard))
              }
            }
            
            // 如果正在扫描，保持任务状态为 running，不改为 stopped
            if (tab.runningStatus && task.status !== 'running') {
              task.status = 'running'
            }
            
            // 保存任务数据到localStorage
            await saveTasksToStorage()
          }
          // 不断开关联，让任务继续在后台运行，任务数据保留在任务管理中
        }
      }
      
      // 保留一个空白任务标签，避免页面进入 empty 态
      scanTabs.value = []
      createTab('新建扫描')
    }
    
    const tasks = ref([])
    
    // 任务管理分页
    const taskPagination = reactive({
      currentPage: 1,
      pageSize: 10
    })
    
    // 分页后的任务列表
    const paginatedTasks = computed(() => {
      const start = (taskPagination.currentPage - 1) * taskPagination.pageSize
      return tasks.value.slice(start, start + taskPagination.pageSize)
    })

    // 表格高度自适应计算
    const tableHeight = computed(() => {
      // 计算可用高度：视口高度 - 顶部工具栏 - overview卡片 - 筛选工具栏 - 分页器等
      // 使用更灵活的计算方式，确保在不同屏幕尺寸下都能正常显示
      // 大约减去：工具栏60px + overview 150px + 筛选50px + 分页40px + padding 50px = 350px
      // 使用动态计算，让表格可以自适应
      if (typeof window !== 'undefined') {
        const viewportHeight = window.innerHeight || 800
        const calculatedHeight = Math.max(300, viewportHeight - 350)
        return calculatedHeight
      }
      return 400 // 默认高度
    })

    // 端口预设列表（参考 HostscanView 配置，保证两处一致）
    const TOP100_PORTS = [
      80,23,443,21,22,25,3389,110,445,139,143,53,135,3306,8080,1723,111,5900,8888,81,10000,514,5060,2000,8443,8000,32768,554,1433,49152,2001,8008,49154,5666,5000,5631,49153,8081,2049,88,79,5800,2121,1110,49155,6000,513,990,5357,543,544,5101,389,8009,444,9999,5009,7070,5190,3000,5432,3986,1900,6646,5051,49157,873,1755,2717,4899,9100,82,83,85,888,1521,3443,4430,4433,4443,5443,5985,6379,7001,8001,8002,8003,8010,8082,8086,8088,8089,8090,9000,9043,9200,9443,10443,11211,27017,8848,7001
    ]

    const TOP1000_PORTS = [
      7,9,11,13,17,19,21,22,23,26,37,38,43,49,51,53,67,69,70,79,80,81,82,83,84,85,86,88,89,102,104,111,113,119,121,123,135,137,138,139,143,161,175,179,199,211,264,311,389,443,444,445,465,500,502,503,512,515,520,548,554,564,587,623,631,636,646,666,771,777,789,800,801,808,853,873,880,888,902,992,993,995,999,1000,1022,1023,1024,1025,1026,1027,1080,1099,1177,1194,1200,1201,1234,1241,1248,1260,1290,1311,1344,1400,1433,1443,1471,1494,1515,1521,1554,1588,1701,1720,1723,1741,1777,1863,1883,1900,1911,1935,1962,1967,1991,2000,2001,2002,2020,2022,2030,2031,2049,2052,2053,2077,2080,2082,2083,2086,2087,2095,2096,2121,2123,2152,2181,2222,2223,2252,2323,2332,2375,2376,2379,2401,2404,2424,2427,2443,2455,2480,2501,2601,2628,3000,3001,3002,3006,3128,3260,3283,3288,3299,3306,3307,3310,3333,3388,3389,3390,3443,3460,3541,3542,3689,3690,3749,3780,4000,4022,4040,4063,4064,4111,4343,4369,4430,4433,4443,4444,4500,4505,4567,4664,4712,4730,4782,4786,4840,4848,4880,4911,4949,5000,5001,5002,5004,5005,5006,5007,5009,5050,5060,5084,5222,5258,5269,5351,5353,5357,5400,5432,5443,5500,5554,5555,5556,5560,5577,5601,5631,5672,5678,5683,5800,5801,5900,5901,5902,5903,5938,5984,5985,5986,6000,6001,6002,6003,6005,6006,6050,6060,6068,6110,6363,6379,6443,6488,6560,6565,6581,6588,6590,6600,6664,6665,6666,6667,6668,6669,6699,6881,6998,7000,7001,7002,7003,7005,7006,7010,7014,7025,7070,7071,7077,7080,7170,7288,7306,7307,7312,7401,7443,7474,7493,7537,7547,7548,7634,7657,7777,7779,7788,7911,8000,8001,8002,8003,8004,8006,8008,8009,8010,8015,8020,8025,8030,8040,8060,8069,8080,8081,8082,8083,8084,8085,8086,8087,8088,8089,8090,8091,8092,8093,8094,8095,8096,8097,8098,8099,8111,8112,8118,8123,8125,8126,8139,8159,8161,8180,8181,8182,8200,8222,8291,8333,8334,8377,8378,8388,8443,8500,8545,8546,8554,8649,8686,8800,8834,8880,8883,8887,8888,8889,8899,8983,8999,9000,9001,9002,9003,9009,9010,9030,9042,9050,9051,9080,9083,9090,9091,9100,9151,9191,9200,9295,9333,9418,9443,9444,9527,9530,9595,9600,9653,9700,9711,9869,9944,9981,9997,9999,10000,10001,10003,10162,10243,10333,10443,10554,11001,11211,11300,11310,12300,12345,13579,14000,14147,14265,16010,16030,16992,16993,17000,18000,18001,18080,18081,18245,18246,19999,20000,20547,20880,22105,22222,23023,23424,25000,25105,25565,27015,27017,28017,28080,29876,30001,32400,33338,33890,37215,37777,41795,45554,49151,49152,49153,49154,49155,50000,50050,50070,50100,51106,52869,55442,55553,55555,60001,60010,60030,60443,61613,61616,62078,64738,8848
    ]

    const truncateBanner = (text, maxLength = 50) => {
      if (!text) return ''
      const str = text.toString()
      if (str.length <= maxLength) return str
      return `${str.slice(0, maxLength)}...`
    }

    const formatLogLevel = (level) => {
      const val = (level || 'INFO').toString().toUpperCase()
      return val
    }

    const getLogTagType = (level) => {
      const lv = (level || '').toLowerCase()
      if (lv.includes('error') || lv.includes('fail')) return 'danger'
      if (lv.includes('warn')) return 'warning'
      if (lv.includes('success') || lv === 'ok') return 'success'
      return 'info'
    }

    const form = reactive({
      input: '',
      taskName: '',
      appendPoc: '',
      runnningStatus: false,
      taskId: '',
      newWebscanDialog: false
    })

    const config = reactive({
      customTemplate: [],
      customTags: [],
      webscanOption: 3,
      skipNucleiWithoutTags: false,
      customHeaders: ''
    })
    
    // 新建任务下拉框控制
    const newTaskToolbarPopoverRef = ref(null)
    const newTaskEmptyPopoverRef = ref(null)
    // 仅保留变量以兼容原有逻辑（已不再使用弹出菜单）
    const newTaskToolbarPopoverVisible = ref(false)
    const newTaskEmptyPopoverVisible = ref(false)
    
    const fingerOptions = ref([])
    const templateOptions = ref([])
    const fingerAllTags = ref([])
    const templateAllOptions = ref([])
    const tasksLocal = ref({})
    
    // 保存扫描结果到本地JSON文件
    const saveResultsToFile = async (taskId, results, stats) => {
      try {
        const appPath = await GetAppPath()
        const pathSeparator = appPath.includes('\\') ? '\\' : '/'
        const resultsDir = `${appPath}${pathSeparator}logs${pathSeparator}webscan`
        
        const timestamp = Date.now()
        const fileName = `webscan_${taskId}_${timestamp}.json`
        const filePath = `${resultsDir}${pathSeparator}${fileName}`
        
        const dataToSave = {
          taskId,
          timestamp: new Date().toISOString(),
          results: results || [],
          stats: stats || {}
        }
        
        await SaveFile(filePath, JSON.stringify(dataToSave, null, 2))
        return filePath
      } catch (error) {
        console.error('保存结果到文件失败:', error)
        return null
      }
    }
    
    // 从文件加载扫描结果
    const loadResultsFromFile = async (filePath) => {
      try {
        if (!filePath) return null
        const content = await ReadFile(filePath)
        return JSON.parse(content)
      } catch (error) {
        console.error('从文件加载结果失败:', error)
        return null
      }
    }
    
    // 统一的任务保存函数：将完整结果保存到文件，localStorage只保存元数据
    const saveTasksToStorage = async () => {
      try {
        // 压缩任务数据：完整结果保存到文件，localStorage只保存元数据
        const compressedTasks = {}
        for (const [taskId, task] of Object.entries(tasksLocal.value)) {
          // 如果任务有完整结果且没有保存到文件，则保存到文件
          // 注意：WebscanView的任务结果存储在标签页中，这里只保存任务元数据
          compressedTasks[taskId] = {
            ...task,
            // 不保存完整结果数组到localStorage（结果在标签页中）
            results: undefined,
            resultsCount: task.results ? task.results.length : 0,
            // 保存文件路径，用于后续加载
            resultsFilePath: task.resultsFilePath || null
          }
        }
        
        const tasksStr = JSON.stringify(compressedTasks)
        
        // 检查数据大小
        if (tasksStr.length > 2 * 1024 * 1024) { // 如果超过 2MB
          // 清理旧任务，只保留最近100个任务
          const taskArray = Object.entries(compressedTasks)
            .sort((a, b) => (b[1].createdAt || 0) - (a[1].createdAt || 0))
            .slice(0, 100)
          
          const limitedTasks = {}
          taskArray.forEach(([taskId, task]) => {
            limitedTasks[taskId] = task
          })
          
          localStorage.setItem('webscanTasks', JSON.stringify(limitedTasks))
        } else {
          localStorage.setItem('webscanTasks', tasksStr)
        }
      } catch (error) {
        if (error.name === 'QuotaExceededError') {
          console.warn('存储空间不足，清理旧任务')
          try {
            const taskArray = Object.entries(tasksLocal.value)
              .sort((a, b) => (b[1].createdAt || 0) - (a[1].createdAt || 0))
              .slice(0, 50)
            
            const limitedTasks = {}
            taskArray.forEach(([taskId, task]) => {
              limitedTasks[taskId] = {
                ...task,
                results: undefined,
                resultsCount: task.results ? task.results.length : 0,
                resultsFilePath: task.resultsFilePath || null
              }
            })
            
            localStorage.setItem('webscanTasks', JSON.stringify(limitedTasks))
            tasksLocal.value = limitedTasks
            tasks.value = Object.values(tasksLocal.value)
          } catch (e) {
            console.error('清理后保存任务仍然失败:', e)
            localStorage.removeItem('webscanTasks')
          }
        } else {
          console.error('保存任务失败:', error)
        }
      }
    }
    const fingerOptionsLoading = ref(false)
    const templateOptionsLoading = ref(false)
    const fingerOptionsLoaded = ref(false)
    const templateOptionsLoaded = ref(false)
    const pocMapCache = ref(null)
    const tasksCache = ref(null)
    const tasksCacheTime = ref(0)
    const TASKS_CACHE_DURATION = 30000 // 30秒缓存
    const tasksLoading = ref(false)
    
    // 防抖：避免快速点击导致的状态竞争
    // 使用统一的任务锁，确保同一任务的任何操作（暂停/恢复/停止）都互斥
    const operationLock = ref(new Set())

    const MAX_PRELOAD_OPTIONS = 30

    const detailDialog = ref(false)
    const selectedRow = ref(null)
    const historyDialog = ref(false)
    const showConfigDialog = ref(false)
    const reloading = ref(false)
    const pocConfigPanelRef = ref(null)
    const packetViewerVisible = ref(false)
    const monacoReady = ref(false)

    const hotReloadPoc = async () => {
      try {
        reloading.value = true
        // 先让后端从磁盘重建指纹/POC 映射（ListPocs 只统计文件，不会更新 WorkFlowDB）
        await ReloadPocTags()
        pocMapCache.value = null

        // 使用 ListPocs 获取所有 POC 数量（与 POC 管理界面保持一致）
        const pocList = await ListPocs()
        dashboard.pocLength = pocList ? pocList.length : 0

        const pocMap = await fetchPocMap(true)
        pocMapCache.value = pocMap || {}
        buildOptionsFromMap(pocMapCache.value)
        // 热加载后始终同步下拉数据，避免「已标记 loaded 但选项仍为空」或仅重启后才出现的问题
        fingerOptions.value = limitWithSelected(fingerAllTags.value, config.customTags)
        templateOptions.value = limitWithSelected(templateAllOptions.value, config.customTemplate)
        fingerOptionsLoaded.value = true
        templateOptionsLoaded.value = true

        ElMessage.success('POC 已热加载')
      } catch (e) {
        console.error(e)
        ElMessage.error('热加载失败')
      } finally {
        reloading.value = false
      }
    }


    const resetRiskLevel = () => {
      Object.keys(dashboard.riskLevel).forEach(k => {
        dashboard.riskLevel[k] = 0
      })
    }

    const resetDashboard = () => {
      resetRiskLevel()
      dashboard.vulnCount = 0
      dashboard.nucleiPercentage = 0
    }

    const resetProgressStats = () => {
      progressStats.total = 0
      progressStats.completed = 0
      progressStats.vuln = 0
    }

    const rebuildDashboardFromResults = (tab, list) => {
      if (!tab) return
      const tabList = list || tab.vp.table.result
      
      // 重置当前标签页的风险等级
      Object.keys(tab.dashboard.riskLevel).forEach(k => {
        tab.dashboard.riskLevel[k] = 0
      })
      
      tab.dashboard.vulnCount = tabList.length
      tabList.forEach(item => {
        const key = (item.severity || '').toUpperCase()
        if (tab.dashboard.riskLevel[key] !== undefined) {
          tab.dashboard.riskLevel[key]++
        }
      })
      tab.pagination.currentPage = 1
    }

    // 从响应包中提取状态码
    const extractStatusCode = (response) => {
      if (!response || typeof response !== 'string') return ''
      // HTTP 响应格式: HTTP/1.1 200 OK 或 HTTP/2 200
      const match = response.match(/HTTP\/[\d.]+ (\d{3})/)
      if (match && match[1]) {
        return match[1]
      }
      return ''
    }

    const normalizeResult = (item) => ({
      id: item.id || `${Date.now()}-${Math.random()}`,
      template_id: item.template_id || item.TemplateID,
      vuln_name: item.vuln_name || item.VulnName,
      vuln_url: item.vuln_url || item.VulnURL,
      severity: (item.severity || item.Severity || 'INFO').toUpperCase(),
      protocol: item.protocol || item.Protocol || '',
      status_code: item.status_code || item.statusCode || item.StatusCode || extractStatusCode(item.response || item.Response || ''),
      response_time: item.response_time || item.ResponseTime || '',
      description: item.description || item.Description || '',
      reference: item.reference || item.Reference || '',
      request: item.request || item.Request || '',
      response: item.response || item.Response || '',
      extract: item.extract || item.Extract || '' // 账号密码信息（格式：username/password 或 password）
    })

    const syncTasksValue = (records) => {
      const map = {}
      ;(records || []).forEach(r => {
        map[r.task_id] = {
          taskId: r.task_id,
          name: r.name || r.task_id,
          targets: (r.targets || []).join('\n'),
          vuln: r.vuln || 0,
          config: r.config || {},
          createdAt: r.created_at,
          status: r.status || 'completed',
          progress: r.progress || 0,
          completed: r.completed || 0,
          total: r.total || 0
        }
      })
      tasksLocal.value = map
      tasks.value = Object.values(map)
      
      // 如果当前页没有数据了，重置到第一页
      const totalPages = Math.ceil(tasks.value.length / taskPagination.pageSize)
      if (taskPagination.currentPage > totalPages && totalPages > 0) {
        taskPagination.currentPage = 1
      }
    }

    const loadTasksFromBackend = async (force = false) => {
      // 使用缓存机制，避免频繁请求
      const now = Date.now()
      if (!force && tasksCache.value && (now - tasksCacheTime.value) < TASKS_CACHE_DURATION) {
        syncTasksValue(tasksCache.value)
        return
      }
      
      tasksLoading.value = true
      try {
        const records = await ListScanTasks()
        const recordsList = records || []
        tasksCache.value = recordsList
        tasksCacheTime.value = now
        syncTasksValue(recordsList)
      } catch (e) {
        console.error(e)
        ElMessage.error('加载任务记录失败')
      } finally {
        tasksLoading.value = false
      }
    }

    const handleHistoryDialogOpened = async () => {
      // 对话框打开时，先使用缓存快速显示，然后异步刷新（如果需要）
      const now = Date.now()
      if (tasksCache.value && (now - tasksCacheTime.value) < TASKS_CACHE_DURATION) {
        syncTasksValue(tasksCache.value)
        // 后台异步刷新，不阻塞UI
        loadTasksFromBackend(false).catch(e => console.error('后台刷新任务失败:', e))
      } else {
        // 缓存过期或不存在，立即加载
        await loadTasksFromBackend(false)
      }
    }

    const refreshTasks = async () => {
      await loadTasksFromBackend(true)
      
      // 同步任务状态到标签页
      syncTaskStatusToTabs()
      
      ElMessage.success('任务列表已刷新')
    }
    
    // 同步任务状态到标签页
    const syncTaskStatusToTabs = () => {
      // 遍历所有任务，如果标签页存在，同步状态
      for (const [taskId, task] of Object.entries(tasksLocal.value)) {
        const tab = scanTabs.value.find(t => t.taskId === taskId)
        if (tab) {
          // 同步任务基本信息（确保任务管理和标签页显示一致）
          if (task.name && task.name !== tab.name) {
            tab.name = task.name
          }
          
          // 同步任务状态
          tab.runningStatus = task.status === 'running'
          
          // 同步进度信息
          if (task.status === 'completed') {
            tab.dashboard.nucleiPercentage = 100
            if (task.total === 0 && task.completed > 0) {
              tab.progressStats.total = task.completed
              tab.progressStats.completed = task.completed
            } else {
              tab.progressStats.total = task.total ?? 0
              tab.progressStats.completed = task.completed ?? 0
            }
          } else {
            tab.dashboard.nucleiPercentage = task.progress ?? 0
            tab.progressStats.total = task.total ?? 0
            tab.progressStats.completed = task.completed ?? 0
          }
          
          // 同步仪表板数据
          if (task.dashboard) {
            Object.assign(tab.dashboard, task.dashboard)
          }
        }
      }
    }

    const loadAppendPoc = async () => {
      try {
        const saved = await GetAppendPoc()
        form.appendPoc = saved || ''
      } catch (e) {
        console.error(e)
      }
    }


    const openNewTaskDialog = () => {
      // 默认打开网站扫描对话框
      form.newWebscanDialog = true
    }

    const openNewWebscanDialog = () => {
      // 关闭下拉菜单
      newTaskToolbarPopoverVisible.value = false
      newTaskEmptyPopoverVisible.value = false
      form.inputType = 0
      form.newWebscanDialog = true
    }


    const handleNewTaskDialogOpened = async () => {
      // 对话框打开时异步加载选项，不阻塞UI
      // 先显示空列表，然后后台加载数据
      preloadSelectOptions().catch(e => {
        console.error('预加载选项失败:', e)
      })
    }

    const handleNewTaskDialogClosed = () => {
      // 关闭对话框时，不保留任何输入/选择状态，下次打开是“干净”的
      form.input = ''
      form.taskName = ''
      form.appendPoc = ''
      config.customHeaders = ''
      config.customTags = []
      config.customTemplate = []
      config.skipNucleiWithoutTags = false
      config.webscanOption = 3

      // 同时重置下拉选项加载状态，这样下次打开时可以重新拉取最新数据
      fingerOptionsLoaded.value = false
      templateOptionsLoaded.value = false
      fingerOptions.value = []
      templateOptions.value = []
      pocMapCache.value = null
      fingerAllTags.value = []
      templateAllOptions.value = []
    }

    const handleOpenNewTaskEvent = (event) => {
      const detail = event?.detail || {}
      // 如果从其他模块（如主机扫描、POC 管理）传入了目标，则预填到输入框
      if (detail.targets && Array.isArray(detail.targets) && detail.targets.length) {
        // 去重后填充
        const uniqueTargets = Array.from(new Set(detail.targets.filter(Boolean)))
        form.input = uniqueTargets.join('\n')
      }
      // 如果从 POC 管理传入了模板，则预填到“指定漏洞”中，并自动切换到“指定漏洞扫描”模式
      if (detail.templates && Array.isArray(detail.templates) && detail.templates.length) {
        config.customTemplate = [...detail.templates]
        config.webscanOption = 5
      }
      if (detail.taskName) {
        form.taskName = detail.taskName
      }
      openNewTaskDialog()
    }

    const handleRefreshTasksEvent = () => {
      loadTasksFromBackend(true)
    }

    const handleOpenTaskEvent = async (event) => {
      const detail = event?.detail || {}
      const taskId = detail.taskId || detail.id || detail
      if (!taskId) return
      // 先刷新任务列表，确保获取最新数据
      await loadTasksFromBackend(true)
      // 然后加载任务
      await loadTask(taskId)
      // 如果任务正在运行，确保实时刷新已启动
      const task = tasksLocal.value[taskId]
      if (task && task.status === 'running') {
        // 确保定期刷新已启动
        startTaskRefresh()
      }
    }

    const withSelected = (list, selected) => {
      const map = new Map(list.map(o => [o.value, o]))
      ;(selected || []).forEach(val => {
        if (!map.has(val)) {
          map.set(val, { label: val, value: val })
        }
      })
      return Array.from(map.values())
    }

    const limitWithSelected = (list, selected, limit = MAX_PRELOAD_OPTIONS) => {
      const merged = withSelected(list, selected)
      if (limit && limit > 0) {
        return merged.slice(0, limit)
      }
      return merged
    }

    const fetchPocMap = async (force = false) => {
      // 使用缓存，避免重复请求
      if (!force && pocMapCache.value) return pocMapCache.value
      try {
        const pocMap = await GetFingerPocMap()
        pocMapCache.value = pocMap || {}
        return pocMapCache.value
      } catch (e) {
        console.error('获取指纹POC映射失败:', e)
        return pocMapCache.value || {}
      }
    }

    const buildOptionsFromMap = (pocMap) => {
      const options = []
      const fingerSet = new Set()
      Object.entries(pocMap || {}).forEach(([name, tags]) => {
        options.push({ label: name, value: name })
        ;(tags || []).forEach(tag => fingerSet.add(tag))
      })
      fingerAllTags.value = Array.from(fingerSet).map(t => ({ label: t, value: t }))
      templateAllOptions.value = options
    }

    const preloadSelectOptions = async (force = false) => {
      // 如果强制刷新或者数据未加载，则执行加载
      if (!force && fingerOptionsLoaded.value && templateOptionsLoaded.value) return
      
      // 如果正在加载，避免重复请求
      if (fingerOptionsLoading.value || templateOptionsLoading.value) return
      
      try {
        fingerOptionsLoading.value = true
        templateOptionsLoading.value = true
        
        let pocMap = await fetchPocMap(force)
        
        // 如果数据为空，尝试重新加载标签后再获取（异步进行，不阻塞）
        if (!pocMap || Object.keys(pocMap).length === 0) {
          // 异步重新加载标签，不阻塞当前流程
          ReloadPocTags().then(async () => {
            // 等待一小段时间让后端完成加载
            await new Promise(resolve => setTimeout(resolve, 500))
            const refreshedMap = await fetchPocMap(true)
            if (refreshedMap && Object.keys(refreshedMap).length > 0) {
              buildOptionsFromMap(refreshedMap)
              fingerOptions.value = limitWithSelected(fingerAllTags.value, config.customTags)
              templateOptions.value = limitWithSelected(templateAllOptions.value, config.customTemplate)
              fingerOptionsLoaded.value = true
              templateOptionsLoaded.value = true
            }
          }).catch(reloadError => {
            console.warn('重新加载标签失败:', reloadError)
          })
          // 即使数据为空，也先设置加载状态为完成，避免重复请求
          fingerOptionsLoaded.value = true
          templateOptionsLoaded.value = true
        } else {
          buildOptionsFromMap(pocMap)
          fingerOptions.value = limitWithSelected(fingerAllTags.value, config.customTags)
          templateOptions.value = limitWithSelected(templateAllOptions.value, config.customTemplate)
          fingerOptionsLoaded.value = true
          templateOptionsLoaded.value = true
        }
      } catch (e) {
        console.error('预加载指纹/漏洞选项失败:', e)
        // 不显示错误提示，避免干扰用户
      } finally {
        fingerOptionsLoading.value = false
        templateOptionsLoading.value = false
      }
    }

    const filterFingerOptions = async (query) => {
      const keyword = (query || '').toLowerCase()
      const forceFetch = !!keyword // 搜索时强制从后端取全量
      if (!fingerOptionsLoaded.value || forceFetch) {
        fingerOptionsLoading.value = true
        try {
          const pocMap = await fetchPocMap(forceFetch)
          buildOptionsFromMap(pocMap)
          fingerOptionsLoaded.value = true
        } catch (e) {
          console.error(e)
          ElMessage.error('加载指纹选项失败')
        } finally {
          fingerOptionsLoading.value = false
        }
      }
      const list = keyword
        ? fingerAllTags.value.filter(o => o.label.toLowerCase().includes(keyword))
        : fingerAllTags.value
      // 预加载时仅显示少量，但搜索时全量展示匹配项
      fingerOptions.value = keyword
        ? withSelected(list, config.customTags)
        : limitWithSelected(list, config.customTags)
    }

    const filterTemplateOptions = async (query) => {
      const keyword = (query || '').toLowerCase()
      const forceFetch = !!keyword // 搜索时强制从后端取全量
      if (!templateOptionsLoaded.value || forceFetch) {
        templateOptionsLoading.value = true
        try {
          const pocMap = await fetchPocMap(forceFetch)
          buildOptionsFromMap(pocMap)
          templateOptionsLoaded.value = true
        } catch (e) {
          console.error(e)
          ElMessage.error('加载模板选项失败')
        } finally {
          templateOptionsLoading.value = false
        }
      }
      const list = keyword
        ? templateAllOptions.value.filter(o => o.label.toLowerCase().includes(keyword))
        : templateAllOptions.value
      // 预加载时仅显示少量，但搜索时全量展示匹配项
      templateOptions.value = keyword
        ? withSelected(list, config.customTemplate)
        : limitWithSelected(list, config.customTemplate)
    }

    // 加载指纹选项（懒加载）
    const loadFingerOptions = async () => {
      if (fingerOptionsLoaded.value) return
      if (fingerOptionsLoading.value) return // 防止重复请求
      
      fingerOptionsLoading.value = true
      try {
        const pocMap = await fetchPocMap()
        buildOptionsFromMap(pocMap)
        fingerOptions.value = limitWithSelected(fingerAllTags.value, config.customTags)
        fingerOptionsLoaded.value = true
      } catch (e) {
        console.error(e)
        ElMessage.error('加载指纹选项失败')
      } finally {
        fingerOptionsLoading.value = false
      }
    }

    // 加载模板选项（懒加载）
    const loadTemplateOptions = async () => {
      if (templateOptionsLoaded.value) return
      if (templateOptionsLoading.value) return // 防止重复请求
      
      templateOptionsLoading.value = true
      try {
        const pocMap = await fetchPocMap()
        buildOptionsFromMap(pocMap)
        templateOptions.value = limitWithSelected(templateAllOptions.value, config.customTemplate)
        templateOptionsLoaded.value = true
      } catch (e) {
        console.error(e)
        ElMessage.error('加载模板选项失败')
      } finally {
        templateOptionsLoading.value = false
      }
    }

    // 指纹下拉框可见性变化
    const handleFingerSelectVisible = (visible) => {
      if (visible && !fingerOptionsLoaded.value) {
        loadFingerOptions()
      }
    }

    // 模板下拉框可见性变化
    const handleTemplateSelectVisible = (visible) => {
      if (visible && !templateOptionsLoaded.value) {
        loadTemplateOptions()
      }
    }

    const resetResultsState = () => {
      vp.table.result = []
      vp.ctrl.watchResultChange(vp.table)
      resetDashboard()
    }

    const startScan = async () => {
      if (!form.input.trim()) {
        ElMessage.warning('目标不能为空')
        return
      }
      const targets = form.input.split('\n').map(t => t.trim()).filter(Boolean)
      if (targets.length === 0) {
        ElMessage.warning('请输入有效目标')
        return
      }
      
      // 获取全局代理配置
      let proxyString = ''
      try {
        const globalProxy = await GetGlobalProxy()
        if (globalProxy && globalProxy.enabled && globalProxy.config) {
          const proxyConfig = globalProxy.config
          // 格式化代理URL
          if (proxyConfig.type && proxyConfig.host && proxyConfig.port) {
            const scheme = proxyConfig.type.toLowerCase()
            const host = proxyConfig.host
            const port = proxyConfig.port
            
            // 如果有认证信息
            if (proxyConfig.auth && proxyConfig.auth.username) {
              const username = encodeURIComponent(proxyConfig.auth.username)
              const password = encodeURIComponent(proxyConfig.auth.password || '')
              proxyString = `${scheme}://${username}:${password}@${host}:${port}`
            } else {
              proxyString = `${scheme}://${host}:${port}`
            }
          }
        }
      } catch (e) {
        console.error('获取全局代理配置失败:', e)
      }
      
      // 获取扫描配置（包括模板目录，后端会基于 Tags 自动展开到具体模板）
      let scanConfig = { threads: 5, timeout: 10, rate_limit: 150, retries: 1 }
      let templateFolders = []
      try {
        const configData = await GetPocConfig()
        if (configData) {
          scanConfig = {
            threads: configData.threads || 5,
            timeout: configData.timeout || 10,
            rate_limit: configData.rate_limit || 150,
            retries: configData.retries || 1
          }
          // 后端需要模板目录才能根据指纹展开模板列表
          templateFolders = (configData.template_folders || []).filter(Boolean)
        }
      } catch (e) {
        console.error('获取扫描配置失败:', e)
      }

      // 根据模式决定如何组织 nuclei 参数：
      // - 指纹漏洞扫描（webscanOption === 3）：只使用 tags 参数（等价于 nuclei -tags <指纹> -u <目标> ...）
      // - 指定漏洞扫描（webscanOption === 5）：只使用 template_file，不使用 tags
      // - 全量漏洞扫描（webscanOption === 4）：不下发 tags/template_file，后端根据模板目录做全量扫描
      const isFingerScan = config.webscanOption === 3
      const isTemplateScan = config.webscanOption === 5

      // "指定漏洞"中选择的模板文件名（仅指定漏洞扫描模式）
      const selectedTemplates = isTemplateScan && Array.isArray(config.customTemplate)
        ? [...config.customTemplate]
        : []
      
      // 清理和规范化模板路径
      // template_file 应该是相对路径（不含扩展名），如 "cves/2023/CVE-2023-1234" 或 "WVP-GB28181-info"
      // 如果用户输入了绝对路径，提取文件名（不含扩展名），让后端在配置的模板文件夹中查找
      const normalizeTemplatePath = (path) => {
        if (!path || typeof path !== 'string') return null
        const trimmed = path.trim()
        if (trimmed.length === 0) return null
        
        let templateName = trimmed
        
        // 如果包含文件扩展名，移除扩展名
        if (templateName.endsWith('.yaml') || templateName.endsWith('.yml')) {
          templateName = templateName.replace(/\.(yaml|yml)$/i, '')
        }
        
        // 判断是否为绝对路径
        const isWindowsAbsolute = /^[A-Za-z]:[\\/]/.test(templateName)
        const isUnixAbsolute = templateName.startsWith('/') && !templateName.startsWith('./') && !templateName.startsWith('../')
        
        if (isWindowsAbsolute || isUnixAbsolute) {
          // 从绝对路径中提取文件名（不含扩展名）
          // 支持 Windows 和 Unix 路径分隔符
          const parts = templateName.split(/[\\/]/)
          const fileName = parts[parts.length - 1]
          if (fileName) {
            // 返回文件名（相对路径），让后端在配置的模板文件夹中查找
            templateName = fileName
          } else {
            // 如果提取失败，返回 null
            return null
          }
        } else {
          // 相对路径，直接使用（后端会在模板文件夹中查找）
          // 确保路径使用正斜杠（后端会处理）
          templateName = templateName.replace(/\\/g, '/')
        }
        
        return templateName
      }
      
      // 规范化所有模板路径，并去重
      const normalizedTemplates = selectedTemplates
        .map(normalizeTemplatePath)
        .filter(name => name !== null && name.length > 0)
      const uniqueTemplates = [...new Set(normalizedTemplates)]
      
      if (selectedTemplates.length > 0 && uniqueTemplates.length === 0) {
        ElMessage.warning('指定的漏洞模板路径无效，无法提取模板名称')
        return
      }
      
      // 注意：无论输入的是绝对路径还是相对路径，都会提取模板名称
      // 后端会在配置的模板文件夹中自动查找对应的文件
      
      const hasFingerTags = isFingerScan && Array.isArray(config.customTags) && config.customTags.length > 0
      const hasCustomTemplates = uniqueTemplates.length > 0

      // 按需求组合 tags / template_file：
      // - 指纹扫描（模式3）：tags = 指纹；template_file 为空
      // - 指定漏洞扫描（模式5）：template_file = 指定漏洞；tags 为空
      // - 全量扫描（模式4）：两者都为空，由后端根据模板目录处理
      let tagsParam = []
      let templateFileParam = []
      if (isFingerScan) {
        // 指纹扫描模式，只使用指纹标签
        if (hasFingerTags) {
          tagsParam = [...config.customTags]
        }
      } else if (isTemplateScan) {
        // 指定漏洞扫描模式，只需要模板文件，不需要指纹
        if (hasCustomTemplates) {
          templateFileParam = uniqueTemplates
        }
      }
      
      // 每次新建任务都创建新标签页
      const activeTab = createTab(form.taskName || '新建扫描')
      
      // 更新当前标签页状态
      activeTab.runningStatus = true
      activeTab.taskId = '' // 先清空，等待任务创建后赋值
      
      const payload = {
        targets,
        task_name: form.taskName,
        append_poc: form.appendPoc,
        tags: tagsParam,
        custom_tags: [],
        template_file: templateFileParam,
        template_folders: templateFolders,
        custom_headers: config.customHeaders,
        proxy: proxyString,
        skip_nuclei_without_tags: config.skipNucleiWithoutTags,
        thread_safe: true,
        threads: scanConfig.threads,
        timeout: scanConfig.timeout,
        rate_limit: scanConfig.rate_limit,
        retries: scanConfig.retries
      }
      try {
        const taskId = await StartPocScan(JSON.stringify(payload))
        activeTab.taskId = taskId
        activeTab.name = form.taskName || `Task-${taskId.slice(0, 6)}`
        
        tasksLocal.value[taskId] = {
          taskId,
          name: activeTab.name,
          targets: targets.join('\n'),
          vuln: 0,
          config: payload
        }
        tasks.value = Object.values(tasksLocal.value)
        await loadTasksFromBackend(true)
        form.newWebscanDialog = false
      } catch (e) {
        console.error(e)
        ElMessage.error('启动失败')
        activeTab.runningStatus = false
      }
    }

    const stopScan = async () => {
      if (!currentTab.value || !currentTab.value.taskId) {
        ElMessage.warning('当前没有运行中的任务')
        return
      }
      try {
        await StopPocScan(currentTab.value.taskId)
        currentTab.value.runningStatus = false
        await loadTasksFromBackend(true)
      } catch (e) {
        console.error(e)
        ElMessage.error('停止失败')
      }
    }

    const pauseScan = async () => {
      if (!currentTab.value || !currentTab.value.taskId) {
        ElMessage.warning('当前没有运行中的任务')
        return
      }
      try {
        await PausePocScan(currentTab.value.taskId)
        currentTab.value.runningStatus = false
        await loadTasksFromBackend(true)
        ElMessage.success('任务已暂停')
      } catch (e) {
        console.error(e)
        ElMessage.error('暂停失败')
      }
    }

    const resumeScan = async () => {
      if (!currentTab.value || !currentTab.value.taskId) {
        ElMessage.warning('当前没有可恢复的任务')
        return
      }
      await resumeScanForTab(currentTab.value)
    }

    // 为指定标签页停止扫描
    const stopScanForTab = async (tab) => {
      if (!tab || !tab.taskId) {
        ElMessage.warning('当前没有运行中的任务')
        return
      }
      const taskId = tab.taskId
      
      // 防抖：如果该任务正在任何操作中，忽略本次请求
      if (operationLock.value.has(taskId)) {
        console.log(`任务 ${taskId} 正在操作中，忽略停止请求`)
        return
      }
      
      operationLock.value.add(taskId)
      try {
        await StopPocScan(taskId)
        tab.runningStatus = false
        await loadTasksFromBackend(true)
      } catch (e) {
        console.error(e)
        ElMessage.error('停止失败')
      } finally {
        // 延迟解锁
        setTimeout(() => {
          operationLock.value.delete(taskId)
        }, 1000)
      }
    }

    // 为指定标签页暂停扫描
    const pauseScanForTab = async (tab) => {
      if (!tab || !tab.taskId) {
        ElMessage.warning('当前没有运行中的任务')
        return
      }
      const taskId = tab.taskId
      
      // 防抖：如果该任务正在任何操作中，忽略本次请求
      // 使用统一的任务锁，而不是分别的暂停/恢复锁
      if (operationLock.value.has(taskId)) {
        console.log(`任务 ${taskId} 正在操作中，忽略暂停请求`)
        return
      }
      
      operationLock.value.add(taskId)
      try {
        // 先检查任务状态，确保任务正在运行
        await loadTasksFromBackend(true)
        const task = tasksLocal.value[taskId]
        if (!task) {
          ElMessage.warning('任务不存在')
          operationLock.value.delete(taskId)
          return
        }
        // 如果任务已完成或已停止，不能暂停
        if (task.status === 'completed' || task.status === 'stopped') {
          ElMessage.warning('任务已完成或已停止，无法暂停')
          if (tab.taskId === taskId) {
            tab.runningStatus = false
            if (task.status === 'completed') {
              tab.dashboard.nucleiPercentage = 100
            }
          }
          operationLock.value.delete(taskId)
          return
        }
        if (task.status !== 'running') {
          ElMessage.warning(`任务状态为 ${getStatusText(task.status)}，无法暂停`)
          operationLock.value.delete(taskId)
          return
        }
        
        // 暂停指定任务
        await PausePocScan(taskId)
        
        // 等待一小段时间，确保后端状态已完全更新
        await new Promise(resolve => setTimeout(resolve, 300))
        
        // 强制刷新任务状态，确保获取最新状态
        await loadTasksFromBackend(true)
        
        // 再次检查任务状态，确保只更新对应标签页
        const updatedTask = tasksLocal.value[taskId]
        if (updatedTask && tab.taskId === taskId) {
          if (updatedTask.status === 'paused') {
            tab.runningStatus = false
            ElMessage.success('任务已暂停')
          } else if (updatedTask.status === 'completed' || updatedTask.status === 'stopped') {
            tab.runningStatus = false
            if (updatedTask.status === 'completed') {
              tab.dashboard.nucleiPercentage = 100
            }
            ElMessage.warning('任务已完成或已停止')
          } else if (updatedTask.status === 'running') {
            // 如果状态仍然是 running，说明暂停可能失败，但不显示错误，等待事件更新
            console.warn(`任务 ${taskId} 暂停后状态仍为 running，等待事件更新`)
          }
        }
      } catch (e) {
        console.error('暂停任务失败:', e)
        const errorMsg = e?.message || e?.toString() || '未知错误'
        ElMessage.error(`暂停失败: ${errorMsg}`)
        // 错误时也刷新状态，确保前端状态与后端一致
        try {
          await loadTasksFromBackend(true)
          const task = tasksLocal.value[taskId]
          if (task && tab.taskId === taskId) {
            if (task.status === 'paused') {
              tab.runningStatus = false
            } else if (task.status === 'running') {
              tab.runningStatus = true
            } else {
              tab.runningStatus = false
            }
          }
        } catch (refreshError) {
          console.error('刷新任务状态失败:', refreshError)
        }
      } finally {
        // 延迟解锁，确保事件处理完成后再解锁（增加延迟时间）
        setTimeout(() => {
          operationLock.value.delete(taskId)
        }, 1500)
      }
    }

    // 为指定标签页恢复扫描
    const resumeScanForTab = async (tab) => {
      if (!tab || !tab.taskId) {
        ElMessage.warning('当前没有可恢复的任务')
        return
      }
      const taskId = tab.taskId
      
      // 防抖：如果该任务正在任何操作中，忽略本次请求
      // 使用统一的任务锁，而不是分别的暂停/恢复锁
      if (operationLock.value.has(taskId)) {
        console.log(`任务 ${taskId} 正在操作中，忽略恢复请求`)
        return
      }
      
      operationLock.value.add(taskId)
      try {
        // 先刷新任务状态，确保获取最新状态
        await loadTasksFromBackend(true)
        const task = tasksLocal.value[taskId]
        if (!task) {
          ElMessage.warning('任务不存在')
          operationLock.value.delete(taskId)
          return
        }
        // 如果任务已完成或已停止，不能恢复
        if (task.status === 'completed' || task.status === 'stopped') {
          ElMessage.warning('任务已完成或已停止，无法恢复')
          // 更新标签页状态
          if (tab.taskId === taskId) {
            tab.runningStatus = false
            if (task.status === 'completed') {
              tab.dashboard.nucleiPercentage = 100
            }
          }
          operationLock.value.delete(taskId)
          return
        }
        if (task.status !== 'paused') {
          ElMessage.warning(`任务状态为 ${getStatusText(task.status)}，无法恢复`)
          operationLock.value.delete(taskId)
          return
        }
        
        // 恢复指定任务
        await ResumePocScan(taskId)
        
        // 等待一小段时间，确保后端状态已完全更新
        await new Promise(resolve => setTimeout(resolve, 300))
        
        // 强制刷新任务状态，确保获取最新状态
        await loadTasksFromBackend(true)
        
        // 再次检查任务状态，确保只更新对应标签页
        const updatedTask = tasksLocal.value[taskId]
        if (updatedTask && tab.taskId === taskId) {
          // 如果任务已完成或已停止，更新为完成状态
          if (updatedTask.status === 'completed' || updatedTask.status === 'stopped') {
            tab.runningStatus = false
            if (updatedTask.status === 'completed') {
              tab.dashboard.nucleiPercentage = 100
            }
            ElMessage.warning('任务已完成或已停止')
          } else if (updatedTask.status === 'running') {
            tab.runningStatus = true
            // 立即同步进度信息，确保进度条能正常更新
            if (updatedTask.total > 0) {
              tab.progressStats.total = updatedTask.total
              tab.progressStats.completed = updatedTask.completed || 0
            }
            if (updatedTask.progress !== undefined && updatedTask.progress !== null) {
              tab.dashboard.nucleiPercentage = updatedTask.progress
              if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
                tab.lastProgress.value = updatedTask.progress
              } else {
                tab.lastProgress = ref(updatedTask.progress)
              }
            }
            ElMessage.success('任务已恢复')
          } else if (updatedTask.status === 'paused') {
            // 如果状态仍然是 paused，说明恢复可能失败，但不显示错误，等待事件更新
            console.warn(`任务 ${taskId} 恢复后状态仍为 paused，等待事件更新`)
          }
        }
        
        // 确保定期刷新已启动
        startTaskRefresh()
      } catch (e) {
        console.error('恢复任务失败:', e)
        const errorMsg = e?.message || e?.toString() || '未知错误'
        // 如果任务已经完成，更新状态
        if (errorMsg.includes('已完成') || errorMsg.includes('已完成扫描')) {
          await loadTasksFromBackend(true)
          const task = tasksLocal.value[taskId]
          if (task && (task.status === 'completed' || task.status === 'stopped')) {
            if (tab.taskId === taskId) {
              tab.runningStatus = false
              if (task.status === 'completed') {
                tab.dashboard.nucleiPercentage = 100
              }
            }
            ElMessage.warning('任务已完成，无法恢复')
            operationLock.value.delete(taskId)
            return
          }
        }
        // 如果任务已经恢复或不存在，提供更友好的提示
        if (errorMsg.includes('不存在') || errorMsg.includes('未运行')) {
          // 尝试更新任务状态
          await loadTasksFromBackend(true)
          const task = tasksLocal.value[taskId]
          if (task && task.status === 'running') {
            ElMessage.warning('任务已经处于运行状态')
            if (tab.taskId === taskId) {
              tab.runningStatus = true
              // 同步进度信息
              if (task.total > 0) {
                tab.progressStats.total = task.total
                tab.progressStats.completed = task.completed || 0
              }
              if (task.progress !== undefined && task.progress !== null) {
                tab.dashboard.nucleiPercentage = task.progress
                if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
                  tab.lastProgress.value = task.progress
                } else {
                  tab.lastProgress = ref(task.progress)
                }
              }
            }
            operationLock.value.delete(taskId)
            return
          }
        }
        ElMessage.error(`恢复失败: ${errorMsg}`)
        // 错误时也刷新状态，确保前端状态与后端一致
        try {
          await loadTasksFromBackend(true)
          const task = tasksLocal.value[taskId]
          if (task && tab.taskId === taskId) {
            if (task.status === 'running') {
              tab.runningStatus = true
            } else if (task.status === 'paused') {
              tab.runningStatus = false
            } else {
              tab.runningStatus = false
            }
          }
        } catch (refreshError) {
          console.error('刷新任务状态失败:', refreshError)
        }
      } finally {
        // 延迟解锁，确保事件处理完成后再解锁（增加延迟时间）
        // 但不要在这里释放，因为有些路径已经在 return 前释放了
        // 只在正常流程完成后释放
        if (operationLock.value.has(taskId)) {
          setTimeout(() => {
            operationLock.value.delete(taskId)
          }, 1500)
        }
      }
    }

    // 获取标签页的进度百分比
    const getTabProgressPercentage = (tab) => {
      if (!tab) return 0
      
      // 如果任务已完成，强制返回 100%
      if (tab.taskId) {
        const task = tasksLocal.value[tab.taskId]
        if (task && task.status === 'completed') {
          return 100
        }
      }
      
      const stats = tab.progressStats || { total: 0, completed: 0 }
      // 优先使用 progressStats 计算进度
      if (stats.total > 0) {
        return Math.round((stats.completed / stats.total) * 100)
      }
      
      // 如果 total 为 0，尝试使用 lastProgress 保持上一次的进度值（避免跳到 0%）
      if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
        const lastValue = tab.lastProgress.value
        if (lastValue > 0) {
          return lastValue
        }
      }
      
      // 最后回退到 nucleiPercentage
      return tab.dashboard?.nucleiPercentage || 0
    }
    
    // 获取任务进度百分比（统一计算方式，用于任务管理）
    const getTaskProgressPercentage = (task) => {
      if (!task) return 0
      
      // 如果任务已完成，强制返回 100%
      if (task.status === 'completed') {
        return 100
      }
      
      // 优先使用 total 和 completed 计算进度（与标签页保持一致）
      if (task.total > 0) {
        return Math.round((task.completed / task.total) * 100)
      }
      
      // 如果没有 total，使用 progress 字段
      if (task.progress !== undefined && task.progress !== null) {
        return Math.max(0, Math.min(100, task.progress))
      }
      
      return 0
    }

    const stopTask = async (taskId) => {
      if (!taskId) {
        ElMessage.warning('任务ID不存在')
        return
      }
      
      // 防抖：如果该任务正在任何操作中，忽略本次请求
      if (operationLock.value.has(taskId)) {
        console.log(`任务 ${taskId} 正在操作中，忽略停止请求`)
        return
      }
      
      // 先检查任务状态，确保任务可以停止
      await loadTasksFromBackend(true)
      const task = tasksLocal.value[taskId]
      if (!task) {
        ElMessage.warning('任务不存在')
        return
      }
      
      // 如果任务已完成或已停止，不需要再次停止
      if (task.status === 'completed' || task.status === 'stopped') {
        ElMessage.warning('任务已完成或已停止，无需再次停止')
        return
      }
      
      operationLock.value.add(taskId)
      try {
        await StopPocScan(taskId)
        ElMessage.success('任务已停止')
        
        // 等待一小段时间，确保后端状态已更新
        await new Promise(resolve => setTimeout(resolve, 300))
        
        // 刷新任务列表
        await loadTasksFromBackend(true)
        
        // 更新标签页状态
        const tab = scanTabs.value.find(t => t.taskId === taskId)
        if (tab && tab.taskId === taskId) {
          tab.runningStatus = false
        }
      } catch (e) {
        console.error('停止任务失败:', e)
        const errorMsg = e?.message || e?.toString() || '未知错误'
        
        // 如果错误信息表明任务已经停止或不存在，尝试刷新状态
        if (errorMsg.includes('不存在') || errorMsg.includes('已停止') || errorMsg.includes('已完成')) {
          await loadTasksFromBackend(true)
          const updatedTask = tasksLocal.value[taskId]
          if (updatedTask && (updatedTask.status === 'stopped' || updatedTask.status === 'completed')) {
            ElMessage.warning('任务已经停止')
            const tab = scanTabs.value.find(t => t.taskId === taskId)
            if (tab && tab.taskId === taskId) {
              tab.runningStatus = false
            }
          } else {
            ElMessage.error(`停止失败: ${errorMsg}`)
          }
        } else {
          ElMessage.error(`停止失败: ${errorMsg}`)
        }
      } finally {
        // 延迟解锁，确保操作完成
        setTimeout(() => {
          operationLock.value.delete(taskId)
        }, 1000)
      }
    }

    const pauseTask = async (taskId) => {
      if (!taskId) {
        ElMessage.warning('任务ID不存在')
        return
      }
      
      // 防抖：如果该任务正在任何操作中，忽略本次请求
      if (operationLock.value.has(taskId)) {
        console.log(`任务 ${taskId} 正在操作中，忽略暂停请求`)
        return
      }
      
      operationLock.value.add(taskId)
      try {
        // 先检查任务状态，确保任务正在运行
        await loadTasksFromBackend(true)
        const task = tasksLocal.value[taskId]
        if (!task) {
          ElMessage.warning('任务不存在')
          operationLock.value.delete(taskId)
          return
        }
        // 如果任务已完成或已停止，不能暂停
        if (task.status === 'completed' || task.status === 'stopped') {
          ElMessage.warning('任务已完成或已停止，无法暂停')
          // 更新标签页状态
          const tab = scanTabs.value.find(t => t.taskId === taskId)
          if (tab && tab.taskId === taskId) {
            tab.runningStatus = false
            if (task.status === 'completed') {
              tab.dashboard.nucleiPercentage = 100
            }
          }
          operationLock.value.delete(taskId)
          return
        }
        if (task.status !== 'running') {
          ElMessage.warning(`任务状态为 ${getStatusText(task.status)}，无法暂停`)
          operationLock.value.delete(taskId)
          return
        }
        
        // 暂停指定任务
        await PausePocScan(taskId)
        
        // 等待一小段时间，确保后端状态已完全更新
        await new Promise(resolve => setTimeout(resolve, 300))
        
        // 强制刷新任务状态，确保获取最新状态
        await loadTasksFromBackend(true)
        
        // 更新标签页状态（只更新匹配的标签页）
        const tab = scanTabs.value.find(t => t.taskId === taskId)
        if (tab && tab.taskId === taskId) {
          const updatedTask = tasksLocal.value[taskId]
          if (updatedTask) {
            if (updatedTask.status === 'paused') {
              tab.runningStatus = false
              ElMessage.success('任务已暂停')
            } else if (updatedTask.status === 'completed' || updatedTask.status === 'stopped') {
              tab.runningStatus = false
              if (updatedTask.status === 'completed') {
                tab.dashboard.nucleiPercentage = 100
              }
              ElMessage.warning('任务已完成或已停止')
            }
          }
        }
      } catch (e) {
        console.error('暂停任务失败:', e)
        const errorMsg = e?.message || e?.toString() || '未知错误'
        // 如果任务已经暂停或不存在，不显示错误
        if (errorMsg.includes('不存在') || errorMsg.includes('未运行')) {
          // 尝试更新任务状态
          await loadTasksFromBackend(true)
          const task = tasksLocal.value[taskId]
          if (task && task.status === 'paused') {
            ElMessage.warning('任务已经处于暂停状态')
            const tab = scanTabs.value.find(t => t.taskId === taskId)
            if (tab && tab.taskId === taskId) {
              tab.runningStatus = false
            }
            operationLock.value.delete(taskId)
            return
          }
        }
        ElMessage.error(`暂停失败: ${errorMsg}`)
        // 错误时也刷新状态，确保前端状态与后端一致
        try {
          await loadTasksFromBackend(true)
          const task = tasksLocal.value[taskId]
          if (task) {
            const tab = scanTabs.value.find(t => t.taskId === taskId)
            if (tab && tab.taskId === taskId) {
              if (task.status === 'paused') {
                tab.runningStatus = false
              } else if (task.status === 'running') {
                tab.runningStatus = true
              } else {
                tab.runningStatus = false
              }
            }
          }
        } catch (refreshError) {
          console.error('刷新任务状态失败:', refreshError)
        }
      } finally {
        // 延迟解锁，确保事件处理完成后再解锁
        setTimeout(() => {
          operationLock.value.delete(taskId)
        }, 1500)
      }
    }

    const resumeTask = async (taskId) => {
      if (!taskId) {
        ElMessage.warning('任务ID不存在')
        return
      }
      
      // 防抖：如果该任务正在任何操作中，忽略本次请求
      if (operationLock.value.has(taskId)) {
        console.log(`任务 ${taskId} 正在操作中，忽略恢复请求`)
        return
      }
      
      operationLock.value.add(taskId)
      try {
        // 先刷新任务状态，确保获取最新状态
        await loadTasksFromBackend(true)
        const task = tasksLocal.value[taskId]
        if (!task) {
          ElMessage.warning('任务不存在')
          operationLock.value.delete(taskId)
          return
        }
        // 如果任务已完成或已停止，不能恢复
        if (task.status === 'completed' || task.status === 'stopped') {
          ElMessage.warning('任务已完成或已停止，无法恢复')
          // 更新标签页状态
          const tab = scanTabs.value.find(t => t.taskId === taskId)
          if (tab && tab.taskId === taskId) {
            tab.runningStatus = false
            if (task.status === 'completed') {
              tab.dashboard.nucleiPercentage = 100
            }
          }
          operationLock.value.delete(taskId)
          return
        }
        if (task.status !== 'paused') {
          ElMessage.warning(`任务状态为 ${getStatusText(task.status)}，无法恢复`)
          operationLock.value.delete(taskId)
          return
        }
        
        // 恢复指定任务
        await ResumePocScan(taskId)
        
        // 等待一小段时间，确保后端状态已完全更新
        await new Promise(resolve => setTimeout(resolve, 300))
        
        // 强制刷新任务状态，确保获取最新状态
        await loadTasksFromBackend(true)
        
        // 更新标签页状态（只更新匹配的标签页）
        const tab = scanTabs.value.find(t => t.taskId === taskId)
        if (tab && tab.taskId === taskId) {
          const updatedTask = tasksLocal.value[taskId]
          if (updatedTask) {
            // 如果任务已完成或已停止，更新为完成状态
            if (updatedTask.status === 'completed' || updatedTask.status === 'stopped') {
              tab.runningStatus = false
              if (updatedTask.status === 'completed') {
                tab.dashboard.nucleiPercentage = 100
              }
              ElMessage.warning('任务已完成或已停止')
            } else if (updatedTask.status === 'running') {
              tab.runningStatus = true
              // 立即同步进度信息，确保进度条能正常更新
              if (updatedTask.total > 0) {
                tab.progressStats.total = updatedTask.total
                tab.progressStats.completed = updatedTask.completed || 0
              }
              if (updatedTask.progress !== undefined && updatedTask.progress !== null) {
                tab.dashboard.nucleiPercentage = updatedTask.progress
                if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
                  tab.lastProgress.value = updatedTask.progress
                } else {
                  tab.lastProgress = ref(updatedTask.progress)
                }
              }
              ElMessage.success('任务已恢复')
            } else if (updatedTask.status === 'paused') {
              // 如果状态仍然是 paused，说明恢复可能失败，但不显示错误，等待事件更新
              console.warn(`任务 ${taskId} 恢复后状态仍为 paused，等待事件更新`)
            }
          }
        }
        
        // 确保定期刷新已启动
        startTaskRefresh()
      } catch (e) {
        console.error('恢复任务失败:', e)
        const errorMsg = e?.message || e?.toString() || '未知错误'
        // 如果任务已经完成，更新状态
        if (errorMsg.includes('已完成') || errorMsg.includes('已完成扫描')) {
          await loadTasksFromBackend(true)
          const task = tasksLocal.value[taskId]
          if (task && (task.status === 'completed' || task.status === 'stopped')) {
            const tab = scanTabs.value.find(t => t.taskId === taskId)
            if (tab && tab.taskId === taskId) {
              tab.runningStatus = false
              if (task.status === 'completed') {
                tab.dashboard.nucleiPercentage = 100
              }
            }
            ElMessage.warning('任务已完成，无法恢复')
            operationLock.value.delete(taskId)
            return
          }
        }
        // 如果任务已经恢复或不存在，提供更友好的提示
        if (errorMsg.includes('不存在') || errorMsg.includes('未运行')) {
          // 尝试更新任务状态
          await loadTasksFromBackend(true)
          const task = tasksLocal.value[taskId]
          if (task && task.status === 'running') {
            ElMessage.warning('任务已经处于运行状态')
            const tab = scanTabs.value.find(t => t.taskId === taskId)
            if (tab && tab.taskId === taskId) {
              tab.runningStatus = true
              // 同步进度信息
              if (task.total > 0) {
                tab.progressStats.total = task.total
                tab.progressStats.completed = task.completed || 0
              }
              if (task.progress !== undefined && task.progress !== null) {
                tab.dashboard.nucleiPercentage = task.progress
                if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
                  tab.lastProgress.value = task.progress
                } else {
                  tab.lastProgress = ref(task.progress)
                }
              }
            }
            operationLock.value.delete(taskId)
            return
          }
        }
        ElMessage.error(`恢复失败: ${errorMsg}`)
        // 错误时也刷新状态，确保前端状态与后端一致
        try {
          await loadTasksFromBackend(true)
          const task = tasksLocal.value[taskId]
          if (task) {
            const tab = scanTabs.value.find(t => t.taskId === taskId)
            if (tab && tab.taskId === taskId) {
              if (task.status === 'running') {
                tab.runningStatus = true
              } else if (task.status === 'paused') {
                tab.runningStatus = false
              } else {
                tab.runningStatus = false
              }
            }
          }
        } catch (refreshError) {
          console.error('刷新任务状态失败:', refreshError)
        }
      } finally {
        // 延迟解锁，确保事件处理完成后再解锁
        // 但不要在这里释放，因为有些路径已经在 return 前释放了
        // 只在正常流程完成后释放
        if (operationLock.value.has(taskId)) {
          setTimeout(() => {
            operationLock.value.delete(taskId)
          }, 1500)
        }
      }
    }

    const deleteVuln = (row) => {
      if (!currentTab.value) return
      currentTab.value.vp.table.result = currentTab.value.vp.table.result.filter(item => item.id !== row.id)
      currentTab.value.vp.ctrl.watchResultChange(currentTab.value.vp.table)
      rebuildDashboardFromResults(currentTab.value)
    }

    const clearResults = async () => {
      if (!currentTab.value) return
      try {
        await ElMessageBox.confirm('确定清空当前标签页的扫描结果？', '清空结果', { type: 'warning' })
        currentTab.value.vp.table.result = []
        currentTab.value.vp.ctrl.watchResultChange(currentTab.value.vp.table)
        rebuildDashboardFromResults(currentTab.value)
      } catch {
        /* cancel */
      }
    }

    const showDetail = (row) => {
      selectedRow.value = row
      detailDialog.value = true
    }

    const loadTask = async (taskId) => {
      try {
        // 先确保任务列表是最新的
        if (!tasksLocal.value[taskId]) {
          await loadTasksFromBackend(true)
        }
        
        const task = tasksLocal.value[taskId]
        if (!task) {
          ElMessage.error('任务不存在')
          return
        }
        
        // 查找是否已有该任务的标签页
        let tab = scanTabs.value.find(t => t.taskId === taskId)
        if (!tab) {
          // 创建新标签页
          tab = createTab(task.name || `Task-${taskId.slice(0, 6)}`, taskId)
        }
        
        // 切换到该标签页
        activeTabId.value = tab.id
        
        historyDialog.value = false
        
        // 加载任务结果
        const results = await GetScanResults(taskId)
        const normalized = (results || []).map(r => normalizeResult(r))
        tab.vp.table.result = normalized
        tab.vp.ctrl.watchResultChange(tab.vp.table)
        rebuildDashboardFromResults(tab, normalized)
        
        // 恢复进度显示
        // 如果任务已完成，强制设置进度为 100%
        if (task.status === 'completed') {
          tab.dashboard.nucleiPercentage = 100
          // 如果 total 为 0，但任务已完成，设置 total = completed 以确保进度显示正确
          if (task.total === 0 && task.completed > 0) {
            tab.progressStats.total = task.completed
            tab.progressStats.completed = task.completed
          } else {
            tab.progressStats.total = task.total ?? 0
            tab.progressStats.completed = task.completed ?? 0
          }
        } else {
          tab.dashboard.nucleiPercentage = task.progress ?? 0
          tab.progressStats.total = task.total ?? 0
          tab.progressStats.completed = task.completed ?? 0
        }
        tab.progressStats.vuln = normalized.length
        // 确保 lastProgress 是 ref 对象
        // 检查是否为 ref 对象：必须是对象且包含 value 属性，且不是数组
        if (tab.lastProgress && 
            typeof tab.lastProgress === 'object' && 
            !Array.isArray(tab.lastProgress) &&
            'value' in tab.lastProgress &&
            typeof tab.lastProgress.value !== 'undefined') {
          tab.lastProgress.value = task.progress ?? 0
        } else {
          // 如果不是 ref 对象，创建新的 ref
          tab.lastProgress = ref(task.progress ?? 0)
        }
        
        // 如果任务正在运行或已暂停，设置运行状态并确保实时刷新已启动
        if (task.status === 'running') {
          tab.runningStatus = true
          startTaskRefresh()
        } else if (task.status === 'paused') {
          tab.runningStatus = false
        } else {
          tab.runningStatus = false
        }
        
        // 同步任务名称和配置（如果标签页已存在）
        if (tab.taskId === taskId) {
          tab.name = task.name || tab.name
        }
        
        ElMessage.success(`已加载 ${normalized.length} 条结果`)
      } catch (e) {
        console.error('加载任务失败:', e)
        const errorMsg = e?.message || e?.toString() || '未知错误'
        ElMessage.error(`加载任务失败: ${errorMsg}`)
      }
    }

    const reExecuteTask = async (task) => {
      if (!task || !task.config) {
        ElMessage.error('任务配置不存在')
        return
      }
      
      // 先预填任务配置到表单，然后打开新建任务对话框让用户修改
      const targets = task.targets.split('\n').map(t => t.trim()).filter(Boolean)
      if (targets.length === 0) {
        ElMessage.warning('任务目标为空')
        return
      }
      
      // 预填表单数据
      form.input = task.targets || ''
      form.taskName = task.name ? `重新执行-${task.name}` : `重新执行-${task.taskId.slice(0, 6)}`
      form.appendPoc = task.config.append_poc || ''
      config.customHeaders = task.config.custom_headers || ''
      config.customTags = task.config.tags || []
      // template_file 是文件列表，template_folders 是目录列表，不能混用
      // 只使用 template_file，template_folders 是配置项，不应该在重新执行时使用
      // 规范化模板路径（从绝对路径中提取文件名，让后端在配置的模板文件夹中查找）
      const templateFiles = Array.isArray(task.config.template_file) ? task.config.template_file : []
      const normalizeTemplatePath = (path) => {
        if (!path || typeof path !== 'string') return null
        const trimmed = path.trim()
        if (trimmed.length === 0) return null
        
        let templateName = trimmed
        
        // 如果包含文件扩展名，移除扩展名
        if (templateName.endsWith('.yaml') || templateName.endsWith('.yml')) {
          templateName = templateName.replace(/\.(yaml|yml)$/i, '')
        }
        
        // 判断是否为绝对路径
        const isWindowsAbsolute = /^[A-Za-z]:[\\/]/.test(templateName)
        const isUnixAbsolute = templateName.startsWith('/') && !templateName.startsWith('./') && !templateName.startsWith('../')
        
        if (isWindowsAbsolute || isUnixAbsolute) {
          // 从绝对路径中提取文件名（不含扩展名）
          // 返回文件名（相对路径），让后端在配置的模板文件夹中查找
          const parts = templateName.split(/[\\/]/)
          const fileName = parts[parts.length - 1]
          if (fileName) {
            templateName = fileName
          } else {
            return null
          }
        } else {
          // 相对路径，直接使用（后端会在模板文件夹中查找）
          // 确保路径使用正斜杠（后端会处理）
          templateName = templateName.replace(/\\/g, '/')
        }
        
        return templateName
      }
      const normalizedTemplates = templateFiles
        .map(normalizeTemplatePath)
        .filter(name => name !== null && name.length > 0)
      config.customTemplate = [...new Set(normalizedTemplates)]
      config.skipNucleiWithoutTags = task.config.skip_nuclei_without_tags || false
      
      // 根据配置判断扫描模式
      if (task.config.template_file && task.config.template_file.length > 0) {
        // 如果指定了模板文件，判断是否有tags
        if (task.config.tags && task.config.tags.length > 0) {
          // 指纹扫描模式下，不应该有指定漏洞，清空 customTemplate
          config.webscanOption = 3 // 指纹漏洞扫描
          config.customTemplate = []
        } else {
          config.webscanOption = 5 // 指定漏洞扫描
        }
      } else if (task.config.tags && task.config.tags.length > 0) {
        config.webscanOption = 3 // 指纹漏洞扫描
        config.customTemplate = [] // 指纹扫描模式下，清空指定漏洞
      } else {
        config.webscanOption = 4 // 全量漏洞扫描
        config.customTemplate = [] // 全量扫描模式下，清空指定漏洞
      }
      
      // 关闭任务管理对话框，打开新建任务对话框
      historyDialog.value = false
      form.newWebscanDialog = true
      
      // 预加载选项（如果需要）
      if (!fingerOptionsLoaded.value || !templateOptionsLoaded.value) {
        await preloadSelectOptions()
      }
    }

    const resetFilters = () => {
      if (!currentTab.value) return
      currentTab.value.filterConfig.severity = ''
      currentTab.value.filterConfig.keyword = ''
      currentTab.value.filterConfig.protocol = ''
      if (currentTab.value.columnFilters) {
        currentTab.value.columnFilters.vuln_name = ''
        currentTab.value.columnFilters.severity = ''
        currentTab.value.columnFilters.vuln_url = ''
        currentTab.value.columnFilters.protocol = ''
        currentTab.value.columnFilters.status_code = ''
        currentTab.value.columnFilters.response_time = ''
      }
      currentTab.value.pagination.currentPage = 1
    }

    const onWebscanColumnFilterChange = () => {
      if (currentTab.value) {
        currentTab.value.pagination.currentPage = 1
      }
    }

    const handleFilteredSizeChange = (size) => {
      if (!currentTab.value) return
      currentTab.value.pagination.pageSize = size
      currentTab.value.pagination.currentPage = 1
    }

    const handleFilteredCurrentChange = (page) => {
      if (!currentTab.value) return
      currentTab.value.pagination.currentPage = page
    }
    
    // 任务管理分页处理
    const handleTaskPageSizeChange = (size) => {
      taskPagination.pageSize = size
      taskPagination.currentPage = 1
    }
    
    const handleTaskPageChange = (page) => {
      taskPagination.currentPage = page
    }

    const clearAllTasks = async () => {
      try {
        await ElMessageBox.confirm(
          '确定清除所有历史任务和结果？此操作不可恢复。',
          '清除全部任务',
          { type: 'warning' }
        )
      } catch (e) {
        if (e === 'cancel' || e === 'close') return
        throw e
      }
      try {
        await ClearScanTasks()
        tasksLocal.value = {}
        tasks.value = []
        tasksCache.value = null
        tasksCacheTime.value = 0
        scanTabs.value.forEach(tab => {
          tab.vp.table.result = []
          tab.vp.ctrl.watchResultChange(tab.vp.table)
          rebuildDashboardFromResults(tab)
          tab.taskId = ''
          tab.runningStatus = false
        })
        form.taskId = ''
        ElMessage.success('已清除全部任务')
      } catch (e) {
        console.error(e)
        ElMessage.error('清除失败')
      }
    }

    const deleteTask = async (taskId) => {
      try {
        await ElMessageBox.confirm('确定删除该任务及其后端记录？', '删除任务', { type: 'warning' })
      } catch (e) {
        if (e === 'cancel' || e === 'close') return
        throw e
      }
      try {
        await DeleteScanTask(taskId)
        delete tasksLocal.value[taskId]
        tasks.value = Object.values(tasksLocal.value)
        tasksCache.value = null
        tasksCacheTime.value = 0
        const tab = scanTabs.value.find(t => t.taskId === taskId)
        if (tab) {
          tab.vp.table.result = []
          tab.vp.ctrl.watchResultChange(tab.vp.table)
          rebuildDashboardFromResults(tab)
          tab.taskId = ''
          tab.runningStatus = false
        }
        ElMessage.success('删除成功')
      } catch (e) {
        console.error(e)
        ElMessage.error('删除失败')
      }
    }

    const exportToExcel = () => {
      if (!currentTab.value || !currentTab.value.vp.table.result.length) {
        ElMessage.warning('没有数据可导出')
        return
      }
      const data = currentTab.value.vp.table.result.map((item, idx) => ({
        序号: idx + 1,
        漏洞名称: item.vuln_name || '',
        严重程度: item.severity || '',
        URL: item.vuln_url || '',
        POC: item.template_id || '',
        协议: item.protocol || '',
        状态码: item.status_code || '',
        响应时间: item.response_time || '',
        用户名密码: item.extract || '' // 添加密码爆破的账号密码字段
      }))
      const ws = XLSX.utils.json_to_sheet(data)
      const wb = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(wb, ws, '漏洞结果')
      const fileName = `${currentTab.value.name || '漏洞扫描结果'}_${new Date().toISOString().slice(0,10)}.xlsx`
      XLSX.writeFile(wb, fileName)
      ElMessage.success('导出成功')
    }

    const saveAppendPoc = async () => {
      try {
        await SaveAppendPoc(form.appendPoc || '')
        ElMessage.success('已保存追加 POC 目录')
      } catch (e) {
        console.error(e)
        ElMessage.error('保存失败')
      }
    }

    const handleConfigDialogOpened = async () => {
      // 对话框打开后，确保配置面板加载数据
      // 由于使用了 destroy-on-close，组件会重新创建，mounted 钩子会自动调用 loadConfig
      // 这里作为备用，确保配置被加载
      await nextTick()
      if (pocConfigPanelRef.value && pocConfigPanelRef.value.loadConfig) {
        try {
          await pocConfigPanelRef.value.loadConfig()
        } catch (e) {
          console.error('加载配置失败:', e)
        }
      }
    }

    // 从任务对话框中打开扫描配置对话框（不关闭任务对话框）
    const openConfigDialogFromTask = () => {
      showConfigDialog.value = true
    }

    const handleResult = async (result) => {
      console.log('[漏洞扫描] 收到结果事件:', result)
      console.log('[漏洞扫描] 结果事件详情 - vuln_name:', result.vuln_name || result.vulnName || result.VulnName, 
                  'protocol:', result.protocol || result.Protocol, 
                  'extract:', result.extract || result.Extract,
                  'vuln_url:', result.vuln_url || result.vulnUrl || result.VulnURL)
      const resultTaskId = result.task_id || result.taskId || result.TaskId
      if (!resultTaskId) {
        console.warn('[漏洞扫描] 结果事件缺少 taskId:', result)
        // 即使没有taskId，如果是密码爆破结果，也尝试创建标签页显示
        if (result.extract || result.Extract) {
          console.log('[漏洞扫描] 检测到密码爆破结果，但缺少taskId，使用默认taskId')
          const defaultTaskId = `crack-${Date.now()}`
          result.task_id = defaultTaskId
          result.taskId = defaultTaskId
          result.TaskId = defaultTaskId
          return await handleResult(result) // 递归调用，使用新的taskId
        }
        return
      }
      console.log('[漏洞扫描] 处理结果，taskId:', resultTaskId, '协议:', result.protocol || result.Protocol)
      
      // 检查是否是密码爆破任务
      const isCrackTask = resultTaskId.startsWith('hostscan-crack-') || resultTaskId.startsWith('crack-')
      
      // 如果是密码爆破任务，先确保后端任务记录存在，获取正确的taskId和任务名称
      let actualTaskId = resultTaskId
      if (isCrackTask) {
        try {
          // 先确保后端任务记录存在，后端会返回正确的taskId（带时间戳的）
          await EnsureCrackTaskRecord(resultTaskId, '')
          // 从后端加载任务列表，获取正确的任务名称和taskId
          await loadTasksFromBackend(true)
          
          // 查找最近创建的密码爆破任务（10分钟内）
          const now = Date.now()
          const timeWindow = 10 * 60 * 1000 // 10分钟
          let recentCrackTask = null
          
          for (const [taskId, task] of Object.entries(tasksLocal.value)) {
            if ((task.name && task.name.includes('密码爆破结果')) || 
                taskId.startsWith('hostscan-crack-') || 
                taskId.startsWith('crack-')) {
              // 解析创建时间（可能是时间戳或日期字符串）
              let taskTime = 0
              if (typeof task.createdAt === 'number') {
                taskTime = task.createdAt
              } else if (typeof task.createdAt === 'string') {
                const parsed = Date.parse(task.createdAt)
                taskTime = isNaN(parsed) ? 0 : parsed
              } else {
                taskTime = 0
              }
              // 检查任务是否在时间窗口内
              if (taskTime > 0 && now - taskTime <= timeWindow) {
                recentCrackTask = { taskId, task }
                break
              }
            }
          }
          
          if (recentCrackTask) {
            actualTaskId = recentCrackTask.taskId
          } else {
            // 如果没有找到最近的任务，查找最新的密码爆破任务
            let latestCrackTask = null
            let latestTime = 0
            for (const [taskId, task] of Object.entries(tasksLocal.value)) {
              if ((task.name && task.name.includes('密码爆破结果')) || 
                  taskId.startsWith('hostscan-crack-') || 
                  taskId.startsWith('crack-')) {
                const taskTime = task.createdAt || Date.parse(task.createdAt) || 0
                if (taskTime > latestTime) {
                  latestTime = taskTime
                  latestCrackTask = { taskId, task }
                }
              }
            }
            if (latestCrackTask) {
              actualTaskId = latestCrackTask.taskId
            }
          }
        } catch (e) {
          console.warn('[漏洞扫描] 获取密码爆破任务信息失败:', e)
        }
      }
      
      // 找到对应的标签页
      let tab = scanTabs.value.find(t => t.taskId === actualTaskId)
      
      // 如果找不到对应的标签页（可能是密码爆破等外部任务），创建或使用默认标签页
      if (!tab) {
        // 检查是否是密码爆破任务
        if (isCrackTask) {
          // 使用后端返回的任务名称，如果没有则使用默认名称
          const taskName = tasksLocal.value[actualTaskId]?.name || `密码爆破结果-${new Date().toISOString().slice(0, 19).replace(/[-:T]/g, '').slice(0, 15)}`
          tab = createTab(taskName, actualTaskId)
          // 切换到新标签页（内部标签页）
          activeTabId.value = tab.id
          
          // 确保任务列表中有该任务（从后端加载的数据）
          if (!tasksLocal.value[actualTaskId]) {
            tasksLocal.value[actualTaskId] = {
              taskId: actualTaskId,
              name: taskName,
              status: 'completed', // 密码爆破任务直接标记为已完成，不显示进度条
              vuln: 0,
              createdAt: Date.now(),
              updatedAt: Date.now()
            }
            tasks.value = Object.values(tasksLocal.value)
            // 保存到本地存储
            await saveTasksToStorage()
          } else {
            // 如果任务已存在，使用后端返回的任务信息
            tab.name = tasksLocal.value[actualTaskId].name
            // 同步任务状态到标签页
            syncTaskStatusToTabs()
          }
          // 设置标签页状态为已完成，不显示进度条
          tab.runningStatus = false
          tab.progressStats.completed = 1
          tab.progressStats.total = 1
        } else {
          // 其他未知任务，创建默认标签页
          tab = createTab(`扫描结果-${resultTaskId.slice(0, 8)}`, resultTaskId)
          activeTabId.value = tab.id
          
          // 初始化任务到任务列表（如果不存在）
          if (!tasksLocal.value[resultTaskId]) {
            tasksLocal.value[resultTaskId] = {
              taskId: resultTaskId,
              name: `扫描任务-${resultTaskId.slice(0, 8)}`,
              status: 'running',
              vuln: 0,
              createdAt: Date.now(),
              updatedAt: Date.now()
            }
            tasks.value = Object.values(tasksLocal.value)
            await saveTasksToStorage()
          }
        }
      }
      
      // 先创建 vulnResult 对象
      const vulnResult = {
        id: Date.now() + Math.random(),
        template_id: result.template_id || result.templateId || result.TemplateID,
        vuln_name: result.vuln_name || result.vulnName || result.VulnName,
        vuln_url: result.vuln_url || result.vulnUrl || result.VulnURL,
        severity: (result.severity || result.Severity || 'INFO').toUpperCase(),
        protocol: result.protocol || result.Protocol || '',
        status_code: result.status_code || result.statusCode || result.StatusCode || extractStatusCode(result.response || result.Response || ''),
        response_time: result.response_time || result.responseTime || result.ResponseTime || '',
        description: result.description || result.Description || '',
        reference: result.reference || result.Reference || '',
        request: result.request || result.Request || '',
        response: result.response || result.Response || '',
        extract: result.extract || result.Extract || '' // 保存密码爆破的用户名/密码信息
      }
      
      if (!tab) {
        console.error('[漏洞扫描] 无法找到或创建标签页，taskId:', resultTaskId)
        return
      }
      
      // 如果是密码爆破结果，先保存到后端，获取正确的任务ID
      if (isCrackTask) {
        try {
          // 从原始 result 对象中获取数据，确保字段完整
          // 特别注意：extract 字段包含账号密码信息，必须正确提取
          const templateId = result.template_id || result.templateId || result.TemplateID || vulnResult.template_id || ''
          const vulnName = result.vuln_name || result.vulnName || result.VulnName || vulnResult.vuln_name || (result.protocol || result.Protocol || vulnResult.protocol ? `${(result.protocol || result.Protocol || vulnResult.protocol).toUpperCase()}弱口令` : '未知漏洞')
          const vulnUrl = result.vuln_url || result.vulnUrl || result.VulnURL || vulnResult.vuln_url || ''
          const severity = result.severity || result.Severity || vulnResult.severity || 'HIGH'
          const protocol = result.protocol || result.Protocol || vulnResult.protocol || ''
          // 优先从原始 result 对象获取 extract，确保账号密码信息不丢失
          const extract = result.extract || result.Extract || vulnResult.extract || ''
          const request = result.request || result.Request || vulnResult.request || ''
          const response = result.response || result.Response || vulnResult.response || ''
          const description = result.description || result.Description || vulnResult.description || (protocol ? `${protocol.toUpperCase()}弱口令漏洞` : '未知漏洞')
          const reference = result.reference || result.Reference || vulnResult.reference || ''
          const responseTime = result.response_time || result.responseTime || result.ResponseTime || vulnResult.response_time || ''
          
          // 调试日志：检查 extract 字段是否正确提取
          console.log('[漏洞扫描] 提取字段详情:', {
            '原始result.extract': result.extract,
            '原始result.Extract': result.Extract,
            'vulnResult.extract': vulnResult.extract,
            '最终extract': extract,
            'protocol': protocol,
            'vuln_name': vulnName
          })
          
          // 构造结果对象，使用Go结构体的JSON tag格式（蛇形命名）
          // 注意：task_id 不需要在这里设置，后端会根据时间窗口自动分配到正确的任务
          const resultData = {
            template_id: templateId,
            vuln_name: vulnName,
            vuln_url: vulnUrl,
            severity: severity.toUpperCase(),
            protocol: protocol,
            extract: extract, // 账号密码信息（格式：username/password 或 password）
            request: request,
            response: response,
            description: description,
            reference: reference,
            response_time: responseTime
          }
          
          // 验证 extract 字段是否存在
          if (!extract) {
            console.warn('[漏洞扫描] 警告：extract 字段为空，账号密码信息可能丢失！', {
              result: result,
              vulnResult: vulnResult
            })
          }
          
          const resultJSON = JSON.stringify(resultData)
          
          console.log('[漏洞扫描] 保存密码爆破结果到JSON:', resultData)
          console.log('[漏洞扫描] 保存的JSON字符串:', resultJSON)
          
          // 同时保存到密码爆破模块
          try {
            // 解析 URL 获取主机和端口
            const urlMatch = vulnUrl.match(/^(\w+):\/\/([^:]+)(?::(\d+))?/)
            let host = ''
            let port = ''
            if (urlMatch) {
              host = urlMatch[2]
              port = urlMatch[3] || ''
            }
            
            // 解析 extract 获取用户名和密码
            let username = ''
            let password = ''
            if (extract) {
              const parts = extract.split('/')
              if (parts.length >= 2) {
                username = parts[0]
                password = parts.slice(1).join('/')
              } else {
                password = extract
              }
            }
            
            const protocolConnection = {
              protocol: protocol,
              host: host,
              port: port,
              username: username,
              password: password,
              status: 'success',
              message: vulnName || '密码爆破成功',
              credential: extract,
              vuln_url: vulnUrl,
              description: description || `${protocol}弱口令漏洞`
            }
            
            const connectionJSON = JSON.stringify(protocolConnection)
            await AddPasswordCrackResult(connectionJSON)
            console.log('[漏洞扫描] 已保存到密码爆破模块:', protocolConnection)
          } catch (error) {
            console.error('[漏洞扫描] 保存到密码爆破模块失败:', error)
            // 不阻断主流程，只记录错误
          }
          
          // 传入原始taskId和结果JSON，后端会根据时间窗口自动分配到正确的任务（追加到最近的任务或创建新任务）
          await EnsureCrackTaskRecord(resultTaskId, resultJSON)
          // 重新加载任务列表，确保获取最新的任务记录（包括结果数据）
          await loadTasksFromBackend(true)
          
          // 重新查找正确的任务ID（后端可能创建了新任务或追加到现有任务）
          const now = Date.now()
          const timeWindow = 10 * 60 * 1000 // 10分钟
          let correctTaskId = actualTaskId
          let recentCrackTask = null
          
          for (const [taskId, task] of Object.entries(tasksLocal.value)) {
            if ((task.name && task.name.includes('密码爆破结果')) || 
                taskId.startsWith('hostscan-crack-') || 
                taskId.startsWith('crack-')) {
              // 解析创建时间（可能是时间戳或日期字符串）
              let taskTime = 0
              if (typeof task.createdAt === 'number') {
                taskTime = task.createdAt
              } else if (typeof task.createdAt === 'string') {
                const parsed = Date.parse(task.createdAt)
                taskTime = isNaN(parsed) ? 0 : parsed
              } else {
                taskTime = 0
              }
              if (taskTime > 0 && now - taskTime <= timeWindow) {
                recentCrackTask = { taskId, task }
                break
              }
            }
          }
          
          if (recentCrackTask) {
            correctTaskId = recentCrackTask.taskId
          }
          
          // 找到或创建正确的标签页
          let correctTab = scanTabs.value.find(t => t.taskId === correctTaskId)
          if (!correctTab) {
            // 创建新标签页
            const taskName = tasksLocal.value[correctTaskId]?.name || `密码爆破结果-${new Date().toISOString().slice(0, 19).replace(/[-:T]/g, '').slice(0, 15)}`
            correctTab = createTab(taskName, correctTaskId)
            activeTabId.value = correctTab.id
            correctTab.runningStatus = false
            correctTab.progressStats.completed = 1
            correctTab.progressStats.total = 1
          }
          
          // 如果当前标签页不正确，移除结果并添加到正确标签页
          if (tab && tab.taskId !== correctTaskId) {
            // 从当前标签页移除结果（如果已添加）
            const index = tab.vp.table.result.findIndex(r => r.id === vulnResult.id)
            if (index >= 0) {
              tab.vp.table.result.splice(index, 1)
            }
            tab = correctTab
          } else {
            tab = correctTab
          }
          
          // 添加结果到正确的标签页
          console.log('[漏洞扫描] 添加结果到标签页:', tab.id, vulnResult)
          tab.vp.table.result.unshift(vulnResult)
          tab.vp.ctrl.watchResultChange(tab.vp.table)
          rebuildDashboardFromResults(tab)
          
          // 同步任务状态到标签页（包括任务名称）
          syncTaskStatusToTabs()
          // 更新本地任务列表中的漏洞数量（从后端加载的数据）
          if (tasksLocal.value[correctTaskId]) {
            // 使用从后端加载的漏洞数量，确保数据准确
            tasksLocal.value[correctTaskId].vuln = tab.vp.table.result.length
            tasksLocal.value[correctTaskId].updatedAt = Date.now()
            // 确保使用后端返回的任务名称
            if (tasksLocal.value[correctTaskId].name && tab) {
              tab.name = tasksLocal.value[correctTaskId].name
            }
            tasks.value = Object.values(tasksLocal.value)
            // 保存到本地存储
            await saveTasksToStorage()
          }
        } catch (e) {
          console.error('[漏洞扫描] 确保密码爆破任务记录失败:', e)
          // 即使失败，也添加到当前标签页并更新本地任务列表
          console.log('[漏洞扫描] 添加结果到标签页（失败回退）:', tab.id, vulnResult)
          tab.vp.table.result.unshift(vulnResult)
          tab.vp.ctrl.watchResultChange(tab.vp.table)
          rebuildDashboardFromResults(tab)
          if (tasksLocal.value[actualTaskId]) {
            tasksLocal.value[actualTaskId].vuln = tab.vp.table.result.length
            tasksLocal.value[actualTaskId].updatedAt = Date.now()
            tasks.value = Object.values(tasksLocal.value)
            await saveTasksToStorage()
          }
        }
      } else {
        // 非密码爆破结果，直接添加到标签页
        console.log('[漏洞扫描] 添加结果到标签页:', tab.id, vulnResult)
        tab.vp.table.result.unshift(vulnResult)
        tab.vp.ctrl.watchResultChange(tab.vp.table)
        rebuildDashboardFromResults(tab)
        
        // 更新任务列表中的漏洞计数和更新时间（使用实际的taskId）
        const updateTaskId = actualTaskId || resultTaskId
        // 非密码爆破结果，直接更新本地任务列表
        if (tasksLocal.value[updateTaskId]) {
          tasksLocal.value[updateTaskId].vuln = tab.vp.table.result.length
          tasksLocal.value[updateTaskId].updatedAt = Date.now()
          tasks.value = Object.values(tasksLocal.value)
          // 保存到本地存储
          await saveTasksToStorage()
        }
      }
    }

    const normalizeProgressPayload = (payload) => {
      if (!payload) return null
      if (Array.isArray(payload)) {
        return payload[0] || null
      }
      if (typeof payload === 'string') {
        try {
          return JSON.parse(payload)
        } catch (e) {
          return null
        }
      }
      if (payload && typeof payload === 'object' && payload.detail) {
        return payload.detail
      }
      return payload
    }

    const handleProgress = (payload) => {
      const p = normalizeProgressPayload(payload)
      if (!p) return

      const taskId = p.task_id || p.taskId
      if (!taskId) return

      const stats = p.stats || p.Stats || {}
      
      // 更新任务列表中的进度
      if (tasksLocal.value[taskId]) {
        const task = tasksLocal.value[taskId]
        let needUpdate = false
        
        if (stats) {
          const newTotal = Number(stats.totalTargets || stats.TotalTargets || task.total || 0)
          const newCompleted = Number(stats.completed || stats.Completed || task.completed || 0)
          // 确保 total 和 completed 只增不减（除非是初始化）
          let finalTotal = newTotal
          let finalCompleted = newCompleted
          if (task.total > 0 && newTotal > 0 && newTotal < task.total) {
            // 如果新的 total 小于已有的 total，保持原有值（避免回退）
            finalTotal = task.total
          }
          if (task.completed > 0 && newCompleted < task.completed) {
            // 如果新的 completed 小于已有的 completed，保持原有值（避免回退）
            finalCompleted = task.completed
          }
          if (task.total !== finalTotal || task.completed !== finalCompleted) {
            task.total = finalTotal
            task.completed = finalCompleted
            needUpdate = true
          }
        }
        const percentage = p.percentage ?? p.percent ?? p.progress ?? p.Percentage ?? p.Progress
        if (percentage !== undefined) {
          let num = Number(percentage)
          if (!Number.isNaN(num)) {
            if (num > 0 && num <= 1) {
              num = num * 100
            }
            const newProgress = Math.max(0, Math.min(100, Math.round(num)))
            if (task.progress !== newProgress) {
              task.progress = newProgress
              needUpdate = true
            }
          }
        }
        if (task.status !== 'running') {
          task.status = 'running'
          needUpdate = true
        }
        
        if (needUpdate) {
          nextTick(() => {
            tasks.value = Object.values(tasksLocal.value)
          })
        }
      }

      // 找到对应的标签页并更新进度
      const tab = scanTabs.value.find(t => t.taskId === taskId)
      if (tab) {
        if (stats) {
          const newTotal = Number(stats.totalTargets || stats.TotalTargets || tab.progressStats.total || 0)
          const newCompleted = Number(stats.completed || stats.Completed || tab.progressStats.completed || 0)
          // 只有当 newTotal > 0 时才更新 total，避免将已有的 total 覆盖为 0
          // 如果 newTotal 为 0 但 tab.progressStats.total > 0，保持原有值
          let finalTotal = newTotal > 0 ? newTotal : (tab.progressStats.total > 0 ? tab.progressStats.total : 0)
          // 确保 total 和 completed 只增不减（除非是初始化）
          if (tab.progressStats.total > 0 && finalTotal > 0 && finalTotal < tab.progressStats.total) {
            finalTotal = tab.progressStats.total
          }
          let finalCompleted = newCompleted
          if (tab.progressStats.completed > 0 && newCompleted < tab.progressStats.completed) {
            // 如果新的 completed 小于已有的 completed，保持原有值（避免回退）
            finalCompleted = tab.progressStats.completed
          }
          if (tab.progressStats.total !== finalTotal || tab.progressStats.completed !== finalCompleted) {
            tab.progressStats.total = finalTotal
            tab.progressStats.completed = finalCompleted
            tab.progressStats.vuln = tab.dashboard.vulnCount
          }
        }

        let computedPercent
        if (tab.progressStats.total > 0) {
          computedPercent = Math.round((tab.progressStats.completed / tab.progressStats.total) * 100)
        } else {
          const percentage = p.percentage ?? p.percent ?? p.progress ?? p.Percentage ?? p.Progress
          if (percentage !== undefined) {
            let num = Number(percentage)
            if (!Number.isNaN(num)) {
              if (num > 0 && num <= 1) {
                num = num * 100
              }
              computedPercent = Math.round(num)
            }
          }
        }

        if (computedPercent !== undefined) {
          const currentPercent = Math.max(0, Math.min(100, computedPercent))
          // 更新 nucleiPercentage（包括 0，因为这是初始状态）
          tab.dashboard.nucleiPercentage = currentPercent
          
          // 更新 lastProgress，但只有当进度 > 0 时才更新，避免将有效的进度值覆盖为 0
          if (currentPercent > 0) {
            if (tab.lastProgress && 
                typeof tab.lastProgress === 'object' && 
                !Array.isArray(tab.lastProgress) &&
                'value' in tab.lastProgress) {
              tab.lastProgress.value = currentPercent
            } else {
              tab.lastProgress = ref(currentPercent)
            }
          }
        }
      }
    }

    const severityClassMap = {
      CRITICAL: 'row-critical',
      HIGH: 'row-high',
      MEDIUM: 'row-medium',
      LOW: 'row-low',
      INFO: 'row-info'
    }

    const getRowSeverityClass = (row) => {
      const key = (row?.severity || '').toUpperCase()
      return severityClassMap[key] || ''
    }

    // 根据状态码返回对应的样式类
    const getStatusCodeClass = (statusCode) => {
      if (!statusCode) return ''
      const code = parseInt(statusCode)
      if (code >= 200 && code < 300) {
        return 'status-code-success' // 2xx 成功
      } else if (code >= 300 && code < 400) {
        return 'status-code-redirect' // 3xx 重定向
      } else if (code >= 400 && code < 500) {
        return 'status-code-client-error' // 4xx 客户端错误
      } else if (code >= 500) {
        return 'status-code-server-error' // 5xx 服务器错误
      }
      return ''
    }

    const sendToPacketSender = () => {
      if (!selectedRow.value || !selectedRow.value.request) {
        ElMessage.warning('没有可用的请求包数据')
        return
      }
      
      // 将数据包信息存储到 sessionStorage
      const packetData = {
        requestName: selectedRow.value.vuln_name || '漏洞请求',
        requestPacket: selectedRow.value.request || '',
        responsePacket: selectedRow.value.response || '',
        fromWebscan: true,
        timestamp: Date.now()
      }
      
      try {
        sessionStorage.setItem('easytools:restmate:packet', JSON.stringify(packetData))
        ElMessage.success('数据包已准备，正在跳转...')
        detailDialog.value = false
        // 跳转到 Restmate 页面
        router.push({ name: 'restmate', query: { openPacket: 'true' } })
      } catch (e) {
        console.error('保存数据包失败:', e)
        ElMessage.error('数据传递失败')
      }
    }

    const getStatusTagType = (status) => {
      const statusMap = {
        'running': 'primary',
        'completed': 'success',
        'stopped': 'danger',
        'paused': 'warning'
      }
      return statusMap[status] || 'info'
    }

    const getStatusText = (status) => {
      const statusMap = {
        'running': '运行中',
        'completed': '已完成',
        'stopped': '已停止',
        'paused': '已暂停'
      }
      return statusMap[status] || '未知'
    }

    const restoreTaskProgress = async (taskId) => {
      if (!taskId) {
        ElMessage.warning('任务ID不存在')
        return
      }
      
      // 直接使用 loadTask 函数，它已经适配了多标签页架构
      await loadTask(taskId)
    }

    // 定期刷新任务列表（用于更新正在运行的任务进度）
    let taskRefreshInterval = null
    const startTaskRefresh = () => {
      if (taskRefreshInterval) return
      taskRefreshInterval = setInterval(() => {
        // 只刷新任务列表，不阻塞UI
        loadTasksFromBackend(false).then(() => {
          // 刷新后同步标签页状态
          scanTabs.value.forEach(tab => {
            if (tab.taskId) {
              const task = tasksLocal.value[tab.taskId]
              if (task) {
                // 如果任务已完成或已停止，更新标签页状态
                if (task.status === 'completed' || task.status === 'stopped') {
                  tab.runningStatus = false
                  if (task.status === 'completed') {
                    tab.dashboard.nucleiPercentage = 100
                    // 如果 total 为 0 但任务已完成，确保进度统计正确
                    if (tab.progressStats.total === 0 && task.completed > 0) {
                      tab.progressStats.total = task.completed
                      tab.progressStats.completed = task.completed
                    } else if (task.total > 0) {
                      tab.progressStats.total = task.total
                      tab.progressStats.completed = task.completed
                    }
                    // 更新 lastProgress
                    if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
                      tab.lastProgress.value = 100
                    } else {
                      tab.lastProgress = ref(100)
                    }
                  }
                } else if (task.status === 'running') {
                  tab.runningStatus = true
                  // 更新进度统计，但只在有有效数据时更新，避免将 total 覆盖为 0
                  // 并且确保 completed 只增不减
                  if (task.total > 0) {
                    // 确保 total 只增不减
                    if (tab.progressStats.total > 0 && task.total < tab.progressStats.total) {
                      // 保持原有的 total，不更新
                    } else {
                      tab.progressStats.total = task.total
                    }
                    // 确保 completed 只增不减
                    if (task.completed >= (tab.progressStats.completed || 0)) {
                      tab.progressStats.completed = task.completed
                    }
                  }
                  
                  // 更新 nucleiPercentage 和 lastProgress
                  // 优先使用 progressStats 计算进度，如果没有则使用 task.progress
                  if (tab.progressStats.total > 0) {
                    // 使用 progressStats 计算进度
                    const computedProgress = Math.round((tab.progressStats.completed / tab.progressStats.total) * 100)
                    tab.dashboard.nucleiPercentage = computedProgress
                    if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
                      tab.lastProgress.value = computedProgress
                    } else {
                      tab.lastProgress = ref(computedProgress)
                    }
                  } else if (task.progress !== undefined && task.progress !== null && task.progress >= 0) {
                    // 如果没有 progressStats，使用 task.progress
                    const newProgress = task.progress
                    // 只有在新的进度值 > 0 时才更新，避免将有效的进度值覆盖为 0
                    if (newProgress > 0) {
                      tab.dashboard.nucleiPercentage = newProgress
                      // 更新 lastProgress
                      if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
                        tab.lastProgress.value = newProgress
                      } else {
                        tab.lastProgress = ref(newProgress)
                      }
                    } else if (newProgress === 0 && tab.dashboard.nucleiPercentage === 0) {
                      // 只有当当前也是 0 时才更新为 0（初始状态）
                      tab.dashboard.nucleiPercentage = 0
                    }
                    // 如果 newProgress 为 0 但已有进度 > 0，不更新（保持上次进度）
                  }
                } else if (task.status === 'paused') {
                  tab.runningStatus = false
                }
              }
            }
          })
        }).catch(e => console.error('定期刷新任务列表失败:', e))
      }, 3000) // 每3秒刷新一次
    }
    const stopTaskRefresh = () => {
      if (taskRefreshInterval) {
        clearInterval(taskRefreshInterval)
        taskRefreshInterval = null
      }
    }

    // 监听扫描模式变化，切换到指纹扫描模式时清空指定漏洞
    watch(() => config.webscanOption, (newVal) => {
      if (newVal === 3) {
        // 切换到指纹扫描模式时，清空指定漏洞
        config.customTemplate = []
      }
    })

    onMounted(async () => {
      // 标记漏洞扫描模块已就绪，供父级事件代理判断
      window.__webscanReady = true
      // 标记组件已激活（用于 keep-alive 场景）
      window.__webscanActivated = true
      
      // 1. 先热加载 POC，保证后端已经完成基本的模板扫描/统计
      await hotReloadPoc()

      // 2. 主动预加载指纹/漏洞下拉选项，避免用户首次直接打开"新建网站扫描"时看到空数据
      //    这样就不需要先进入「POC 管理」再回来才能看到指纹 / 漏洞列表
      await preloadSelectOptions()

      // 3. 加载历史任务与追加 POC 配置
      await loadTasksFromBackend()
      await loadAppendPoc()

      if (scanTabs.value.length === 0) {
        createTab('新建扫描')
      }
      
      // 启动定期刷新任务列表
      startTaskRefresh()
      
      // 注册 HTTP 语言高亮（用于详情内联展示）
      const { monacoRef } = useMonaco()
      const tryRegister = () => {
        if (monacoRef.value) {
          registerHTTPLanguage(monacoRef.value)
          monacoReady.value = true
        } else {
          setTimeout(tryRegister, 100)
        }
      }
      tryRegister()
      EventsOn('nucleiResult', handleResult)
      EventsOn('pocScanResult', handleResult) // 监听pocScanResult事件以实时刷新漏洞信息
      EventsOn('pocScanProgress', handleProgress)
      // 监听日志事件，显示漏洞扫描任务的日志信息
      EventsOn('log_event', (logData) => {
        if (!logData || !logData.message) return
        
        const level = logData.level || 'info'
        const message = logData.message || ''
        const source = logData.source || ''
        const time = logData.time || dayjs().format('HH:mm:ss')
      
        // 检查是否是漏洞扫描相关的日志
        if (message.includes('漏洞扫描') || message.includes('POC') || message.includes('nuclei')) {
          // 在控制台输出日志，方便调试
          console.log(`[漏洞扫描][${level}] ${message}`)
          
          // 如果是任务启动日志，显示成功消息
          if (message.includes('漏洞扫描任务启动')) {
            ElMessage.success({
              message: message,
              duration: 3000,
              showClose: true
            })
          }
        }

        // 端口扫描日志此前用于“主机扫描”联动，已迁移到端口扫描模块，这里不再处理
      })
      EventsOn('pocScanComplete', async (data) => {
        const taskId = (data && data.task_id) || (data && data.taskId)
        if (!taskId) return
        
        // 找到对应的标签页
        const tab = scanTabs.value.find(t => t.taskId === taskId)
        if (tab) {
          tab.runningStatus = false
          tab.dashboard.nucleiPercentage = 100
        }
        
        // 强制刷新任务列表，确保获取最新状态
        loadTasksFromBackend(true).then(() => {
          // 刷新后再次更新状态，确保状态正确
          if (tasksLocal.value[taskId]) {
            const task = tasksLocal.value[taskId]
            // 如果后端返回的状态是 completed 或 stopped，都视为完成
            if (task.status === 'completed' || task.status === 'stopped') {
              task.status = 'completed'
              task.progress = 100
              tasks.value = Object.values(tasksLocal.value)
              
              // 确保标签页状态也更新
              if (tab) {
                tab.runningStatus = false
                tab.dashboard.nucleiPercentage = 100
                // 确保进度统计正确（处理 total 为 0 的情况）
                if (task.total > 0) {
                  tab.progressStats.total = task.total
                  tab.progressStats.completed = task.completed
                } else if (task.completed > 0) {
                  tab.progressStats.total = task.completed
                  tab.progressStats.completed = task.completed
                }
              }
            }
          }
        }).catch(e => console.error('刷新任务列表失败:', e))
        
        // 释放操作锁
        operationLock.value.delete(taskId)
      })
      EventsOn('pocScanPaused', (data) => {
        const taskId = (data && data.task_id) || (data && data.taskId)
        if (!taskId) return
        
        // 如果任务正在操作中（有操作锁），延迟处理事件，避免与操作函数冲突
        if (operationLock.value.has(taskId)) {
          setTimeout(() => {
            // 延迟后刷新任务状态，确保获取最新状态
            loadTasksFromBackend(true).then(() => {
              const tab = scanTabs.value.find(t => t.taskId === taskId)
              if (tab && tab.taskId === taskId) {
                const task = tasksLocal.value[taskId]
                if (task) {
                  if (task.status === 'paused') {
                    tab.runningStatus = false
                  }
                }
              }
            }).catch(e => console.error('延迟刷新任务状态失败:', e))
          }, 500)
          return
        }
        
        // 只更新匹配的标签页
        const tab = scanTabs.value.find(t => t.taskId === taskId)
        if (tab && tab.taskId === taskId) {
          tab.runningStatus = false
        }
        
        // 只更新匹配的任务
        if (tasksLocal.value[taskId] && tasksLocal.value[taskId].taskId === taskId) {
          tasksLocal.value[taskId].status = 'paused'
          tasks.value = Object.values(tasksLocal.value)
        }
      })
      EventsOn('pocScanResumed', async (data) => {
        const taskId = (data && data.task_id) || (data && data.taskId)
        if (!taskId) return
        
        // 如果任务正在操作中（有操作锁），延迟处理事件，避免与操作函数冲突
        if (operationLock.value.has(taskId)) {
          setTimeout(async () => {
            // 延迟后刷新任务状态，确保获取最新状态
            try {
              await loadTasksFromBackend(true)
              const tab = scanTabs.value.find(t => t.taskId === taskId)
              if (tab && tab.taskId === taskId) {
                const task = tasksLocal.value[taskId]
                if (task) {
                  if (task.status === 'running') {
                    tab.runningStatus = true
                    // 同步最新的进度信息（使用任务记录的进度信息，确保显示正确的已扫描进度）
                    if (task.total > 0) {
                      tab.progressStats.total = task.total
                      tab.progressStats.completed = task.completed || 0
                    }
                    // 如果有进度百分比，也同步
                    if (task.progress !== undefined && task.progress !== null) {
                      tab.dashboard.nucleiPercentage = task.progress
                      if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
                        tab.lastProgress.value = task.progress
                      } else {
                        tab.lastProgress = ref(task.progress)
                      }
                    } else if (task.total > 0 && task.completed >= 0) {
                      // 如果没有进度百分比，使用 total 和 completed 计算
                      const calculatedProgress = Math.round((task.completed / task.total) * 100)
                      tab.dashboard.nucleiPercentage = calculatedProgress
                      if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
                        tab.lastProgress.value = calculatedProgress
                      } else {
                        tab.lastProgress = ref(calculatedProgress)
                      }
                    }
                  }
                }
              }
            } catch (e) {
              console.error('延迟刷新任务状态失败:', e)
            }
          }, 500)
          return
        }
        
        // 刷新任务状态，获取最新的进度信息
        await loadTasksFromBackend(true)
        
        // 只更新匹配的标签页
        const tab = scanTabs.value.find(t => t.taskId === taskId)
        if (tab && tab.taskId === taskId) {
          tab.runningStatus = true
          // 同步最新的进度信息（使用任务记录的进度信息，确保显示正确的已扫描进度）
          const task = tasksLocal.value[taskId]
          if (task) {
            if (task.total > 0) {
              tab.progressStats.total = task.total
              tab.progressStats.completed = task.completed || 0
            }
            // 如果有进度百分比，也同步
            if (task.progress !== undefined && task.progress !== null) {
              tab.dashboard.nucleiPercentage = task.progress
              if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
                tab.lastProgress.value = task.progress
              } else {
                tab.lastProgress = ref(task.progress)
              }
            } else if (task.total > 0 && task.completed >= 0) {
              // 如果没有进度百分比，使用 total 和 completed 计算
              const calculatedProgress = Math.round((task.completed / task.total) * 100)
              tab.dashboard.nucleiPercentage = calculatedProgress
              if (tab.lastProgress && typeof tab.lastProgress === 'object' && 'value' in tab.lastProgress) {
                tab.lastProgress.value = calculatedProgress
              } else {
                tab.lastProgress = ref(calculatedProgress)
              }
            }
          }
        }
        
        // 只更新匹配的任务
        if (tasksLocal.value[taskId] && tasksLocal.value[taskId].taskId === taskId) {
          tasksLocal.value[taskId].status = 'running'
          tasks.value = Object.values(tasksLocal.value)
        }
      })
      EventsOn('pocScanError', (err) => {
        const taskId = (err && err.task_id) || (err && err.taskId)
        if (!taskId) return
        
        // 只更新匹配的标签页
        const tab = scanTabs.value.find(t => t.taskId === taskId)
        if (tab && tab.taskId === taskId) {
          tab.runningStatus = false
        }
        
        // 释放操作锁
        operationLock.value.delete(taskId)
      })
      EventsOn('pocScanStopped', (data) => {
        const taskId = (data && data.task_id) || (data && data.taskId)
        if (!taskId) return
        
        // 强制刷新任务列表，确保获取最新状态
        loadTasksFromBackend(true).then(() => {
          // 更新标签页状态
          const tab = scanTabs.value.find(t => t.taskId === taskId)
          if (tab && tab.taskId === taskId) {
            tab.runningStatus = false
          }
          
          // 更新任务状态
          if (tasksLocal.value[taskId]) {
            tasksLocal.value[taskId].status = 'stopped'
            tasks.value = Object.values(tasksLocal.value)
          }
        }).catch(e => console.error('刷新任务列表失败:', e))
        
        // 释放操作锁
        operationLock.value.delete(taskId)
      })

      // 监听来自 POC 管理页面的"自动打开新任务"事件
      window.addEventListener('webscan-open-new-task', handleOpenNewTaskEvent)
      // 监听来自 POC 管理页面的"刷新任务列表"事件
      window.addEventListener('webscan-refresh-tasks', handleRefreshTasksEvent)
      // 监听来自 POC 管理页面的"打开指定任务"事件
      window.addEventListener('webscan-open-task', handleOpenTaskEvent)
    })

    // keep-alive 激活时
    onActivated(() => {
      // 标记组件已激活
      window.__webscanActivated = true
    })

    // keep-alive 失活时
    onDeactivated(() => {
      // 标记组件未激活
      window.__webscanActivated = false
    })

    onBeforeUnmount(() => {
      stopTaskRefresh()
      // 组件卸载时重置就绪标记
      if (window.__webscanReady) {
        window.__webscanReady = false
      }
      if (window.__webscanActivated) {
        window.__webscanActivated = false
      }
      EventsOff('nucleiResult')
      EventsOff('pocScanResult')
      EventsOff('pocScanProgress')
      EventsOff('pocScanComplete')
      EventsOff('pocScanError')
      EventsOff('pocScanPaused')
      EventsOff('pocScanResumed')
      EventsOff('pocScanStopped')
      EventsOff('log_event') // 移除日志事件监听
      window.removeEventListener('webscan-open-new-task', handleOpenNewTaskEvent)
      window.removeEventListener('webscan-refresh-tasks', handleRefreshTasksEvent)
      window.removeEventListener('webscan-open-task', handleOpenTaskEvent)
    })

      return {
      dashboard,
      currentDashboard,
      currentProgressStats,
      computedProgressPercentage,
      form,
      config,
      scanTabs,
      activeTabId,
      currentTab,
      currentFilteredResults,
      currentFilteredPageContent,
      getActiveTabName,
      removeTab,
      handleTabClick,
      tasks,
      detailDialog,
      selectedRow,
      historyDialog,
      showConfigDialog,
      reloading,
      fingerOptions,
      templateOptions,
      packetViewerVisible,
      monacoReady,
      detectPacketLanguage,
      pocConfigPanelRef,
      handleConfigDialogOpened,
      openConfigDialogFromTask,
      newTaskToolbarPopoverVisible,
      newTaskEmptyPopoverVisible,
      handleNewTaskDialogOpened,
      handleNewTaskDialogClosed,
      handleHistoryDialogOpened,
      refreshTasks,
      tasksLoading,
      fingerOptionsLoading,
      templateOptionsLoading,
      handleFingerSelectVisible,
      handleTemplateSelectVisible,
      filterFingerOptions,
      filterTemplateOptions,
      truncateBanner,
      formatLogLevel,
      getLogTagType,
      // icons
      Plus,
      Link,
      Warning,
      Refresh,
      Filter,
      Reading,
      Delete,
      Download,
      Clock,
      View,
      Setting,
      DocumentAdd,
      Upload,
      VideoPlay,
      startScan,
      stopScan,
      pauseScan,
      resumeScan,
      pauseTask,
      resumeTask,
      openNewTaskDialog,
      openNewWebscanDialog,
      hotReloadPoc,
      showDetail,
      deleteVuln,
      clearResults,
      exportToExcel,
      deleteTask,
      loadTask,
      reExecuteTask,
      getTagTypeBySeverity,
      getBadgeClass,
      saveAppendPoc,
      stopTask,
      getStatusTagType,
      getStatusText,
      restoreTaskProgress,
      resetFilters,
      onWebscanColumnFilterChange,
      handleFilteredSizeChange,
      handleFilteredCurrentChange,
      clearAllTasks,
      paginatedTasks,
      taskPagination,
      handleTaskPageSizeChange,
      handleTaskPageChange,
      getTaskProgressPercentage,
      Search,
      getRowSeverityClass,
      getStatusCodeClass,
      sendToPacketSender,
      Promotion,
      CircleCheck,
      handleTabsContextMenu,
      handleCloseOtherTabs,
      handleCloseAllTabs,
      tabsContextMenuVisible,
      tabsContextMenuStyle,
      getTaskStatus,
      getTabProgressPercentage,
      pauseScanForTab,
      resumeScanForTab,
      stopScanForTab,
      tableHeight,
      handleRowContextMenu,
      rowContextMenuVisible,
      rowContextMenuStyle,
      connectModuleType,
      connectModuleName,
      handleSendToConnectModule,
      BrowserOpenURL
    }
  }
}
</script>

<style scoped>
.webscan-page {
  padding: 2px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: transparent;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.top-toolbar {
  flex-shrink: 0;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid #ebeef5;
}

.toolbar-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  width: 100%;
  box-sizing: border-box;
  flex-wrap: wrap;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
  
  .module-title {
    display: flex;
    align-items: center;
    gap: 10px;
    
    .title-icon {
      font-size: 20px;
      color: #e6a23c;
    }
    
    .title-text {
      font-size: 16px;
      font-weight: 600;
      color: #303133;
      letter-spacing: 0.5px;
    }
  }
}

.poc-info {
  display: flex;
  align-items: center;
  gap: 8px;
  background: transparent;
}

.info-label {
  color: #606266;
  font-size: 14px;
}

.info-value {
  color: #da1e1e;
  font-weight: 600;
  font-size: 16px;
}

.toolbar-right {
  display: flex;
  align-items: center;
  background: transparent;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
  max-width: 200px;
  background: transparent;
}

.tab-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 150px;
  display: inline-block;
}

.overview-card {
  border-radius: 3px;
  margin-bottom: 5px;
}

.overview-card :deep(.el-card__body) {
  padding: 20px;
}

.overview-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
  width: 100%;
  box-sizing: border-box;
  flex-wrap: wrap;
}

.overview-title-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
  background: transparent;
}

.title-with-progress {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  background: transparent;
}

.title {
  font-size: 20px;
  font-weight: 700;
  color: #1f2937;
  white-space: nowrap;
}

.progress-inline {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 200px;
  max-width: 400px;
  background: transparent;
  box-sizing: border-box;
}

.progress-inline .icon {
  font-size: 16px;
  flex-shrink: 0;
}

.progress-inline .progress-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
  min-width: 0;
  background: transparent;
}

.progress-bar-wrapper {
  display: flex;
  align-items: center;
  gap: 10px;
  background: transparent;
}

.inline-progress {
  flex: 1;
  min-width: 0;
}

.inline-progress :deep(.el-progress-bar__outer) {
  border-radius: 10px;
  background-color: #e9ecef;
}

.inline-progress :deep(.el-progress-bar__inner) {
  border-radius: 10px;
  background: linear-gradient(90deg, #409eff 0%, #66b1ff 100%);
}

.progress-percentage {
  font-size: 14px;
  color: #409eff;
  font-weight: 600;
  white-space: nowrap;
  flex-shrink: 0;
}

.icon-red { color: #e74c3c; }

.url-text-row {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
}

.url-text-row .url-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.progress-meta {
  display: flex;
  gap: 24px;
  font-size: 13px;
  color: #606266;
  margin-top: 4px;
}

.progress-meta span {
  font-weight: 500;
}

.progress-meta span:first-child {
  color: #409eff;
}

.progress-meta span:nth-child(2) {
  color: #67c23a;
}

.progress-meta span:last-child {
  color: #e74c3c;
}

.overview-content-section {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-shrink: 0;
  background: transparent;
}

.risk-level-display {
  display: flex;
  gap: 8px;
  align-items: center;
  background: transparent;
}

.risk-badge {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 600;
  font-size: 14px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: transform 0.2s;
}

.risk-badge:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.risk-badge-critical { background: linear-gradient(135deg, #c0392b, #e74c3c); }
.risk-badge-red { background: linear-gradient(135deg, #e74c3c, #ff6b6b); }
.risk-badge-yellow { background: linear-gradient(135deg, #f39c12, #fdd835); }
.risk-badge-green { background: linear-gradient(135deg, #27ae60, #2ecc71); }
.risk-badge-gray { background: linear-gradient(135deg, #95a5a6, #bdc3c7); }

.overview-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  background: transparent;
}

.filter-toolbar{
  padding: 10px;
  width: 100%;
  box-sizing: border-box;
}

.table-wrapper {
  width: 100%;
  overflow: hidden;
}

.vuln-table {
  width: 100%;
  box-sizing: border-box;
}

.glass-card {
  border: 1px solid rgba(255, 255, 255, 0.55);
  box-shadow: 0 10px 28px rgba(69, 107, 183, 0.18);
  backdrop-filter: blur(3px);
}

.panel-card {
  flex: 1;
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-card :deep(.el-tabs) {
  flex: 1;
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-card :deep(.el-tabs__header) {
  background: transparent;
  border-bottom: none;
  flex-shrink: 0;
}

.panel-card :deep(.el-tabs__content) {
  padding: 0px;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-card :deep(.el-tab-pane) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.drawer-title {
  font-weight: 700;
}

.detail-wrapper {
  max-height: 70vh;
  overflow: auto;
}

.packet-inline-panels {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.packet-inline-panel {
  background: #1e1e1e;
  border: 1px solid #2f3136;
  border-radius: 6px;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.packet-inline-header {
  color: #d4d4d4;
  font-weight: 600;
  font-size: 13px;
}

.packet-fallback {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 8px;
  border-radius: 4px;
  min-height: 200px;
  white-space: pre-wrap;
  font-family: "JetBrains Mono", "Fira Code", monospace;
  border: 1px solid #2f3136;
}

.table-actions {
  display: flex;
  gap: 8px;
  background: transparent;
}

.empty-text {
  color: #c0c4cc;
}

.vuln-table :deep(.el-table__body-wrapper) {
  font-size: 13px;
}

.vuln-name {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  background: transparent;
}

.vuln-name-label {
  font-weight: 600;
  color: #1f2937;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  width: 100%;
}

.vuln-template {
  font-size: 12px;
  color: #4b5563;
  background: #edf2ff;
  border-radius: 6px;
  padding: 2px 8px;
}

.severity-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 75px;
  padding: 4px 12px;
  border-radius: 999px;
  font-weight: 600;
  color: #fff;
  font-size: 12px;
  text-transform: uppercase;
}

.severity-pill--critical { background: linear-gradient(120deg, #f82929, #ff6b6b); box-shadow: 0 6px 18px rgba(248, 41, 41, 0.3); }
.severity-pill--high { background: linear-gradient(120deg, #ff6a00, #fddb92); color: #fff; }
.severity-pill--medium { background: linear-gradient(120deg, #fcb045, #fd1d1d); }
.severity-pill--low { background: linear-gradient(120deg, #56ab2f, #a8e063); }
.severity-pill--info { background: linear-gradient(120deg, #36d1dc, #5b86e5); }
.severity-pill--unknown { background: #a0aec0; }

.protocol-chip {
  display: inline-flex;
  padding: 2px 12px;
  border-radius: 999px;
  background: #f4f6fb;
  color: #4b5563;
  font-weight: 600;
  font-size: 12px;
}

/* 指定漏洞选择框高度限制，超出滚动 */
.custom-template-item :deep(.el-select__selection) {
  max-height: 260px;
  overflow-y: auto;
}

/* 指定漏洞下拉列表滚动高度限制（使用 :deep 包裹 popper，避免 scoped 失效） */
:deep(.custom-template-popper .el-select-dropdown__wrap) {
  max-height: 260px;
  overflow-y: auto;
}

:deep(.row-critical td) { background: rgba(248, 41, 41, 0.05); }
:deep(.row-high td) { background: rgba(255, 106, 0, 0.05); }
:deep(.row-medium td) { background: rgba(252, 176, 69, 0.06); }
:deep(.row-low td) { background: rgba(86, 171, 47, 0.05); }
:deep(.row-info td) { background: rgba(91, 134, 229, 0.05); }
:deep(.el-table tr.row-critical:hover > td),
:deep(.el-table tr.row-high:hover > td),
:deep(.el-table tr.row-medium:hover > td),
:deep(.el-table tr.row-low:hover > td),
:deep(.el-table tr.row-info:hover > td) {
  background: rgba(59, 130, 246, 0.08);
}

.status-code-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 50px;
  padding: 4px 10px;
  border-radius: 6px;
  font-weight: 600;
  font-size: 12px;
  color: #fff;
}

.status-code-success {
  background: linear-gradient(120deg, #67c23a, #85ce61);
}

.status-code-redirect {
  background: linear-gradient(120deg, #e6a23c, #f0a020);
}

.status-code-client-error {
  background: linear-gradient(120deg, #f56c6c, #ff7875);
}

.status-code-server-error {
  background: linear-gradient(120deg, #f82929, #ff4d4f);
}

/* 表格行右键菜单样式 */
.context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  z-index: 3000;
  min-width: 200px;
  padding: 4px 0;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  cursor: pointer;
  font-size: 14px;
  color: #606266;
  transition: background-color 0.2s;
}

.context-menu-item:hover {
  background-color: #f5f7fa;
  color: #409eff;
}

.context-menu-item.disabled {
  color: #909399;
  cursor: not-allowed;
}

.context-menu-item.disabled:hover {
  background-color: transparent;
  color: #909399;
}

.tab-indicator {
  margin-left: 12px;
  font-size: 13px;
  color: #606266;
  font-weight: normal;
}

.panel-card :deep(.el-tabs__header) {
  margin-bottom: 0;
}

.panel-card :deep(.el-tabs__item) {
  padding: 0 20px;
  height: 40px;
  line-height: 40px;
  color: var(--tabs-font-color, #fff);
  font-weight: var(--tabs-font-weight, bold);
}

.panel-card :deep(.el-tabs__item.is-active) {
  color: var(--tabs-font-color, #fff);
  font-weight: var(--tabs-font-weight, bold);
}

.panel-card :deep(.el-tabs__extra) {
  padding: 0 10px;
}

.main-tabs :deep(.el-tabs__item) {
  padding: 0 20px;
  height: 40px;
  line-height: 40px;
  color: var(--tabs-font-color, #fff);
  font-weight: var(--tabs-font-weight, bold);
}

.main-tabs :deep(.el-tabs__item.is-active) {
  color: var(--tabs-font-color, #0389fe);
  font-weight: var(--tabs-font-weight, bold);
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
  background: transparent;
}

.tab-status-icon {
  font-size: 14px;
}

.tab-status-icon.running {
  color: #409eff;
  animation: pulse 1.5s ease-in-out infinite;
}

.tab-status-icon.completed {
  color: #67c23a;
}

.tab-status-icon.stopped {
  color: #e6a23c;
}

.tab-status-icon.paused {
  color: #f39c12;
}

.tab-badge {
  margin-left: 4px;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

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

/* 新建任务对话框和扫描配置对话框自适应宽度和高度 */
:deep(.new-task-dialog),
:deep(.config-dialog) {
  --el-dialog-width: 65%;
  --el-dialog-margin-top: 2vh;
  max-width: 78vw;
  max-height: 90%;
}

:deep(.history-dialog) {
  --el-dialog-width: 75%;
  --el-dialog-margin-top: 5vh;
  max-width: 90vw;
}

:deep(.new-task-dialog .el-dialog),
:deep(.config-dialog .el-dialog) {
  max-width: 90vw;
  width: 65% !important;
  margin: 5vh auto 0;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

:deep(.history-dialog .el-overlay-dialog) {
  display: flex;
  align-items: flex-start;
  justify-content: center;
}

:deep(.history-dialog .el-dialog) {
  max-width: 90vw;
  width: 75% !important;
  margin: 5vh auto 0;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  left: auto !important;
  transform: none !important;
}

:deep(.new-task-dialog .el-dialog__header),
:deep(.config-dialog .el-dialog__header),
:deep(.history-dialog .el-dialog__header) {
  flex-shrink: 0;
  padding: 20px 20px 10px;
}

:deep(.new-task-dialog .el-dialog__body),
:deep(.config-dialog .el-dialog__body),
:deep(.history-dialog .el-dialog__body) {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 4px;
  box-sizing: border-box;
}

:deep(.new-task-dialog .el-dialog__footer),
:deep(.config-dialog .el-dialog__footer),
:deep(.history-dialog .el-dialog__footer) {
  flex-shrink: 0;
  padding: 10px 20px 20px;
}

/* 任务管理对话框内容区域 */
.history-dialog-content {
  display: flex;
  flex-direction: column;
  min-height: 0;
  width: 100%;
  box-sizing: border-box;
}

.history-table {
  flex: 1;
  min-height: 400px;
  width: 100%;
}

.task-name-cell,
.task-targets-cell,
.table-cell-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  width: 100%;
}

/* 新建任务对话框表单区域 */
:deep(.new-task-dialog .el-form) {
  width: 100%;
  box-sizing: border-box;
  /* 移除 max-height，让表单自然填充可用空间 */
}

:deep(.new-task-dialog .el-form-item) {
  width: 100%;
  box-sizing: border-box;
}

:deep(.new-task-dialog .el-input),
:deep(.new-task-dialog .el-textarea),
:deep(.new-task-dialog .el-select) {
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
}

/* 扫描配置对话框内容 */
:deep(.config-dialog .el-dialog__body) {
  display: flex;
  flex-direction: column;
  min-height: 0;
  width: 100%;
  box-sizing: border-box;
}

/* 与 PasswordCrackView 统一：漏洞结果表密度与表头（保留 glass-card 仅作用于容器） */
.webscan-page .table-wrapper :deep(.el-table) {
  font-size: 12px;
}
.webscan-page .table-wrapper :deep(.el-table th.el-table__cell) {
  background-color: #f5f7fa;
  padding: 8px 0;
  font-weight: 600;
}
.webscan-page .table-wrapper :deep(.el-table td.el-table__cell) {
  padding: 6px 0;
}
.webscan-page .table-wrapper :deep(.el-table__empty-block) {
  padding: 60px 0;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 8px 10px;
  border-top: 1px solid #e4e7ed;
}
</style>

