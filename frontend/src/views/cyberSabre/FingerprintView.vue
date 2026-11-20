<template>
  <div class="fingerprint-container">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar">
      <div class="toolbar-content">
        <div class="toolbar-left">
          <div class="module-title">
            <el-icon class="title-icon"><ChatLineRound /></el-icon>
            <span class="title-text">指纹识别</span>
          </div>
        </div>
        <div class="toolbar-right">
          <el-button 
            :icon="Plus" 
            type="primary" 
            plain
            size="default"
            @click="createNewTab" 
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
        </div>
      </div>
    </div>

    <!-- 标签页 -->
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
              <el-icon v-if="tab.scanning" class="tab-status-icon running">
                <VideoPlay />
              </el-icon>
              <el-icon v-else-if="tab.completed" class="tab-status-icon completed">
                <CircleCheck />
              </el-icon>
              <span class="tab-name">{{ tab.name }}</span>
              <el-badge
                v-if="tab.results.length > 0"
                :value="tab.results.length"
                :max="99"
                class="tab-badge"
              />
            </span>
          </template>

          <div class="config-section">
            <el-card shadow="never" class="config-card">
              <div class="config-layout">
                <div class="url-input-section">
                  <div class="url-label">目标URL</div>
                  <el-input
                    v-model="tab.urlsText"
                    type="textarea"
                    :rows="4"
                    placeholder="每行一个URL，支持批量识别"
                    size="default"
                    class="url-textarea"
                  />
                  <div class="url-actions">
                    <el-button
                      :type="tab.scanning ? 'danger' : 'primary'"
                      :icon="tab.scanning ? VideoPause : VideoPlay"
                      @click="toggleScan(tab)"
                      :loading="tab.loading && !tab.scanning"
                      size="default"
                      style="flex: 1"
                    >
                      {{ tab.scanning ? '停止识别' : '开始识别' }}
                    </el-button>
                  </div>
                  <div class="quick-stats">
                    <div class="quick-stat-item">
                      <span class="label">已识别</span>
                      <span class="value">{{ tab.stats.completed }}/{{ tab.stats.totalTargets }}</span>
                    </div>
                    <div class="quick-stat-item success">
                      <span class="label">成功</span>
                      <span class="value">{{ tab.stats.success }}</span>
                    </div>
                    <div class="quick-stat-item danger">
                      <span class="label">失败</span>
                      <span class="value">{{ tab.stats.failed }}</span>
                    </div>
                  </div>
                </div>

                <div class="config-section-right">
                  <div class="config-group">
                    <div class="config-row">
                      <div class="config-item">
                        <label class="config-label">超时(秒)</label>
                        <el-input-number
                          v-model="tab.config.timeout"
                          :min="3"
                          :max="30"
                          controls-position="right"
                          size="default"
                          style="width: 100%"
                        />
                      </div>
                      <div class="config-item">
                        <label class="config-label">线程数</label>
                        <el-input-number
                          v-model="tab.config.threads"
                          :min="1"
                          :max="50"
                          controls-position="right"
                          size="default"
                          style="width: 100%"
                        />
                      </div>
                      <div class="config-item">
                        <label class="config-label">字典线程</label>
                        <el-input-number
                          v-model="tab.config.dirThreads"
                          :min="1"
                          :max="30"
                          controls-position="right"
                          size="default"
                          style="width: 100%"
                        />
                      </div>
                      <div class="config-item">
                        <label class="config-label">字典上限</label>
                        <el-input-number
                          v-model="tab.config.maxDirChecks"
                          :min="50"
                          :max="2000"
                          controls-position="right"
                          size="default"
                          style="width: 100%"
                        />
                      </div>
                    </div>

                    <div class="config-row switch-row">
                      <div class="config-item">
                        <el-switch v-model="tab.config.useWappalyzer" active-text="Wappalyzer" :disabled="tab.config.checkAlive" />
                      </div>
                      <div class="config-item">
                        <el-switch v-model="tab.config.useTideRules" active-text="标题/响应规则" :disabled="tab.config.checkAlive" />
                      </div>
                      <div class="config-item">
                        <el-switch v-model="tab.config.usePoweredBy" active-text="Powered-By" :disabled="tab.config.checkAlive" />
                      </div>
                      <div class="config-item">
                        <el-switch v-model="tab.config.dirMode" active-text="目录指纹" :disabled="tab.config.checkAlive" />
                      </div>
                      <div class="config-item">
                        <el-tooltip content="不会进行指纹识别" placement="top">
                          <el-switch v-model="tab.config.checkAlive" active-text="URL存活判断" />
                        </el-tooltip>
                      </div>
                    </div>

                    <div class="config-row">
                      <div class="config-item full-width">
                        <label class="config-label">自定义请求头</label>
                        <el-input
                          v-model="tab.config.customHeaders"
                          type="textarea"
                          :rows="3"
                          placeholder="每行一个请求头，例如：User-Agent: xxx"
                          size="default"
                        />
                      </div>
                    </div>

                    <div class="config-row">
                      <div class="config-item">
                        <el-button
                          :icon="Setting"
                          @click="showAdvancedDialog = true; currentAdvancedTab = tab"
                          size="default"
                          plain
                        >
                          高级配置
                        </el-button>
                      </div>
                      <div class="config-item">
                        <el-button
                          :icon="Setting"
                          @click="showFingerprintManageDialog = true"
                          size="default"
                          plain
                        >
                          指纹管理
                        </el-button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
          </el-card>

          <!-- 扫描进度显示 - 固定在配置卡片下方，不撑开配置区域 -->
          <transition name="progress-fade">
            <div v-if="tab.scanning" class="progress-section">
              <el-card shadow="never" class="progress-card">
                <el-progress
                  :percentage="Math.round(tab.stats?.progressPercent || tab.scanProgress || 0)"
                  :status="(tab.stats?.progressPercent || tab.scanProgress || 0) >= 100 ? 'success' : undefined"
                  :stroke-width="8"
                  :show-text="true"
                  :format="(percentage) => `${percentage}%`"
                />
                <div class="progress-stats">
                  <el-tag size="small" type="info">
                    <el-icon><Clock /></el-icon>
                    已用时: {{ formatDuration(tab.stats?.elapsedSeconds || 0) }}
                  </el-tag>
                  <el-tag v-if="tab.stats?.estimatedSeconds > 0" size="small" type="warning">
                    <el-icon><Timer /></el-icon>
                    预计剩余: {{ formatDuration(tab.stats?.estimatedSeconds || 0) }}
                  </el-tag>
                  <el-tag size="small" type="success">
                    <el-icon><Connection /></el-icon>
                    速度: {{ formatNumber(tab.stats?.targetsPerSecond || 0) }} 目标/秒
                  </el-tag>
                  <el-tag size="small">
                    已完成: {{ formatNumber(tab.stats?.completed || 0) }} / {{ formatNumber(tab.stats?.totalTargets || 0) }}
                  </el-tag>
                  <el-tag size="small" type="success">
                    成功: {{ tab.stats?.success || 0 }}
                  </el-tag>
                  <el-tag size="small" type="danger">
                    失败: {{ tab.stats?.failed || 0 }}
                  </el-tag>
                </div>
              </el-card>
            </div>
          </transition>
        </div>

        <div class="results-section">
          <el-card shadow="never" class="results-card">
            <template #header>
              <div class="results-header">
                <div class="results-title-group">
                  <span class="results-title">识别结果</span>
                  <el-tag type="info" size="small">{{ getFilteredResults(tab).length }} 条</el-tag>
                </div>
                <div class="results-actions">
                  <el-button :icon="Download" :disabled="tab.results.length === 0" @click="exportToExcel(tab)">
                    导出Excel
                  </el-button>
                  <el-button :icon="Delete" plain :disabled="tab.results.length === 0" @click="clearResults(tab)">
                    清空
                  </el-button>
                </div>
              </div>
            </template>

            <div class="table-wrapper">
              <el-table
                :data="getPagedResults(tab)"
                :height="tableHeight"
                stripe
                border
                size="default"
                @row-contextmenu="(row, column, event) => handleRowContextMenu(row, column, event, tab)"
                :row-class-name="getRowClassName"
                class="fingerprint-table"
              >
            <el-table-column type="index" width="60" label="#" />

            <el-table-column prop="status" width="100" align="center">
              <template #header>
                <TableColumnFilterHeader
                  label="状态"
                  v-model="tab.columnFilters.status"
                  :popover-width="220"
                  sortable
                  :sort-asc-active="tab.sortField === 'status' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'status' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'status', 'asc')"
                  @sort-desc="setTabSort(tab, 'status', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">
                <el-tag
                  :type="getStatusTagType(row.status)"
                  size="small"
                  effect="dark"
                  class="status-tag"
                >
                  {{ row.status !== undefined && row.status !== null ? row.status : '-' }}
                </el-tag>
              </template>
            </el-table-column>

            <el-table-column prop="url" min-width="180">
              <template #header>
                <TableColumnFilterHeader
                  label="URL"
                  v-model="tab.columnFilters.url"
                  :popover-width="280"
                  sortable
                  :sort-asc-active="tab.sortField === 'url' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'url' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'url', 'asc')"
                  @sort-desc="setTabSort(tab, 'url', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">
                <div class="url-cell">
                  <span class="url-text" :title="row.url">{{ row.url }}</span>
                </div>
              </template>
            </el-table-column>

            <el-table-column prop="title" min-width="230">
              <template #header>
                <TableColumnFilterHeader
                  label="标题"
                  v-model="tab.columnFilters.title"
                  :popover-width="260"
                  sortable
                  :sort-asc-active="tab.sortField === 'title' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'title' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'title', 'asc')"
                  @sort-desc="setTabSort(tab, 'title', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">
                <el-tooltip
                    :content="row.title || ''"
                    placement="top"
                    :disabled="!row.title || row.title.length <= 30"
                    effect="dark"
                >
                  <div class="table-cell-text">
                    {{ row.title || '-' }}
                  </div>
                </el-tooltip>
              </template>
            </el-table-column>

            <el-table-column prop="length" width="110" align="right">
              <template #header>
                <TableColumnFilterHeader
                  label="长度"
                  v-model="tab.columnFilters.length"
                  :popover-width="220"
                  placement="bottom-end"
                  align-right
                  sortable
                  :sort-asc-active="tab.sortField === 'length' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'length' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'length', 'asc')"
                  @sort-desc="setTabSort(tab, 'length', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">
                <span>{{ formatLength(row.length) }}</span>
              </template>
            </el-table-column>

            <el-table-column min-width="200">
              <template #header>
                <TableColumnFilterHeader
                  label="技术栈"
                  v-model="tab.columnFilters.technologies"
                  :popover-width="260"
                  sortable
                  :sort-asc-active="tab.sortField === 'technologies' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'technologies' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'technologies', 'asc')"
                  @sort-desc="setTabSort(tab, 'technologies', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">
                <div class="tag-list">
                  <!-- 显示所有技术栈信息 -->
                  <el-tag
                    v-for="tech in getAllTechnologies(row)"
                    :key="tech"
                    size="small"
                    class="tech-tag"
                    effect="plain"
                    :type="getTechTagType(tech, row)"
                  >
                    {{ tech }}
                  </el-tag>
                  <span v-if="getAllTechnologies(row).length === 0" class="empty-text">
                    -
                  </span>
                </div>
              </template>
            </el-table-column>

            <el-table-column prop="responseTime" width="130" align="right">
              <template #header>
                <TableColumnFilterHeader
                  label="耗时(ms)"
                  v-model="tab.columnFilters.responseTime"
                  :popover-width="220"
                  placement="bottom-end"
                  align-right
                  sortable
                  :sort-asc-active="tab.sortField === 'responseTime' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'responseTime' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'responseTime', 'asc')"
                  @sort-desc="setTabSort(tab, 'responseTime', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">{{ row.responseTime }}</template>
            </el-table-column>
          </el-table>
        </div>

              <div class="pagination-wrapper">
                <el-pagination small
                  v-model:current-page="tab.currentPage"
                  v-model:page-size="tab.pageSize"
                  :total="getFilteredResults(tab).length"
                  :page-sizes="[10, 20, 50, 100, 200]"
                  layout="total, sizes, prev, pager, next, jumper"
                  background
                  @size-change="(size) => handleSizeChange(size, tab)"
                  @current-change="(page) => handleCurrentChange(page, tab)"
                />
              </div>
            </el-card>
          </div>
        </el-tab-pane>
      </el-tabs>
    </CustomTabs>

    <el-dialog
      v-model="detailDialogVisible"
      title="识别详情"
      width="70%"
      top="8vh"
      destroy-on-close
    >
      <div v-if="selectedResult" class="detail-wrapper">
        <el-descriptions :column="2" border size="default">
          <el-descriptions-item label="URL">{{ selectedResult.url }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ selectedResult.status }}</el-descriptions-item>
          <el-descriptions-item label="标题">{{ selectedResult.title || '-' }}</el-descriptions-item>
          <el-descriptions-item label="耗时">{{ selectedResult.responseTime }} ms</el-descriptions-item>
          <el-descriptions-item label="长度">
            {{ formatLength(selectedResult.length) }}
          </el-descriptions-item>
          <el-descriptions-item label="技术栈" :span="2">
            <div class="tag-list">
              <el-tag
                v-for="tech in getAllTechnologies(selectedResult)"
                :key="tech"
                size="small"
                class="tech-tag"
                effect="plain"
                :type="getTechTagType(tech, selectedResult)"
              >
                {{ tech }}
              </el-tag>
              <span v-if="getAllTechnologies(selectedResult).length === 0" class="empty-text">
                -
              </span>
            </div>
          </el-descriptions-item>
        </el-descriptions>

        <div class="detail-block">
          <h4>响应头</h4>
          <pre>{{ formatHeaders(selectedResult.headers) }}</pre>
        </div>

        <div v-if="selectedResult.directoryHits?.length" class="detail-block">
          <h4>目录指纹命中</h4>
          <el-timeline>
            <el-timeline-item
              v-for="hit in selectedResult.directoryHits"
              :key="hit.path"
              type="success"
            >
              {{ hit.rule }} => {{ hit.path }} ({{ hit.pattern }})
            </el-timeline-item>
          </el-timeline>
        </div>

        <div v-if="selectedResult.evidence?.length" class="detail-block">
          <h4>匹配证据</h4>
          <el-timeline>
            <el-timeline-item
              v-for="(item, idx) in selectedResult.evidence"
              :key="idx"
              :type="item.type === 'wappalyzer' ? 'primary' : 'info'"
            >
              [{{ item.type }}] {{ item.name }} - {{ item.detail }}
              <span v-if="item.extra"> ({{ item.extra }})</span>
            </el-timeline-item>
          </el-timeline>
        </div>
      </div>
    </el-dialog>

    <!-- 右键菜单 -->
    <!-- 右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
    <Teleport to="body">
      <div
        v-if="contextMenuVisible"
        class="context-menu"
        :style="contextMenuStyle"
        @click.stop
      >
        <div class="context-menu-item" @click="handleViewPacket">
          <el-icon><View /></el-icon>
          <span>查看数据包</span>
        </div>
        <div class="context-menu-item" @click="handleViewDetail">
          <el-icon><View /></el-icon>
          <span>查看详情</span>
        </div>
        <div class="context-menu-item" @click="handleCopyRow">
          <el-icon><CopyDocument /></el-icon>
          <span>复制当前行</span>
        </div>
        <div class="context-menu-item" @click="handleCopyUrl">
          <el-icon><Link /></el-icon>
          <span>复制URL</span>
        </div>
        <div class="context-menu-item" @click="handleOpenUrl">
          <el-icon><Link /></el-icon>
          <span>打开链接</span>
        </div>
        <div class="context-menu-divider"></div>
        <div class="context-menu-item danger" @click="handleDeleteRow">
          <el-icon><Delete /></el-icon>
          <span>删除当前行</span>
        </div>
      </div>
    </Teleport>

    <!-- 数据包查看对话框 -->
    <PacketViewer
      v-model="packetViewerVisible"
      :request-packet="selectedRow?.requestPacket || ''"
      :response-packet="selectedRow?.responsePacket || ''"
      :title="`数据包查看 - ${selectedRow ? selectedRow.url : ''}`"
      :show-send-actions="false"
    />

    <!-- 高级配置对话框 -->
    <el-dialog
      v-model="showAdvancedDialog"
      title="高级配置"
      width="600px"
      destroy-on-close
      class="advanced-config-dialog"
    >
      <div class="advanced-dialog-content" v-if="currentAdvancedTab">
        <div class="advanced-section">
          <div class="advanced-row">
            <div class="advanced-item full-width">
              <label class="config-label">状态码过滤</label>
              <el-select
                v-model="currentAdvancedTab.config.filter.statusCodes"
                multiple
                allow-create
                filterable
                :reserve-keyword="false"
                placeholder="选择或输入要过滤的状态码（支持手动输入）"
                size="default"
                style="width: 100%"
              >
                <el-option label="0" value="0" />
                <el-option label="200" value="200" />
                <el-option label="301" value="301" />
                <el-option label="302" value="302" />
                <el-option label="400" value="400" />
                <el-option label="403" value="403" />
                <el-option label="404" value="404" />
                <el-option label="500" value="500" />
              </el-select>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showAdvancedDialog = false">取消</el-button>
        <el-button type="primary" @click="saveAdvancedConfig">确定</el-button>
      </template>
    </el-dialog>

    <!-- 历史记录对话框 -->
    <el-dialog
      v-model="historyDialog"
      width="75%"
      destroy-on-close
      class="history-dialog"
      @opened="handleHistoryDialogOpened"
    >
      <template #header>
        <div class="dialog-title" style="display: flex; justify-content: space-between; align-items: center;">
          <div>
            <el-icon :size="18" class="mr-5px"><Clock /></el-icon>任务管理
          </div>
          <div>
            <el-button :icon="Refresh" text @click="refreshTasks" :loading="historyLoading">刷新</el-button>
            <el-button type="danger" text @click="clearAllTasks">
              清除全部
            </el-button>
          </div>
        </div>
      </template>
      <div class="history-dialog-content">
        <div class="history-table">
          <el-table
            :data="paginatedTasks"
            stripe
            border
            size="default"
            v-loading="historyLoading"
            max-height="500"
          >
            <el-table-column prop="name" label="任务名称" min-width="150">
              <template #default="{ row }">
                <el-tooltip
                    :content="row.name || ''"
                    placement="top"
                    :disabled="!row.name || row.name.length <= 30"
                    effect="dark"
                >
                  <div class="task-name-cell">
                    {{ row.name || '-' }}
                  </div>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column prop="urlsText" label="目标" min-width="250">
              <template #default="{ row }">
                <el-tooltip 
                  v-if="row.urlsText && row.urlsText.split('\n').length > 2" 
                  :content="row.urlsText" 
                  placement="top"
                  :show-after="300"
                  effect="dark"
                >
                  <div class="targets-text-truncated">
                    {{ row.urlsText.split('\n').slice(0, 2).join(', ') }}...
                  </div>
                </el-tooltip>
                <span v-else>{{ row.urlsText || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="getStatusTagType(row.status)">
                  {{ getStatusText(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="结果数量" width="100" align="center">
              <template #default="{ row }">
                {{ Array.isArray(row.results) ? row.results.length : (row.resultsCount || row.results || 0) }}
              </template>
            </el-table-column>
            <el-table-column label="关联任务" width="150" align="center">
              <template #default="{ row }">
                <el-tag v-if="row.sourceTaskId" size="small" type="info">
                  {{ getSourceTaskTypeName(row.sourceTaskType) }}
                </el-tag>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column prop="createdAt" label="创建时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.createdAt) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="200" align="center">
              <template #default="{ row }">
                <el-button-group>
                  <el-tooltip content="查看结果">
                    <el-button :icon="View" link @click="loadTask(row.taskId)" />
                  </el-tooltip>
                  <el-tooltip content="停止任务" v-if="row.status === 'running'">
                    <el-button :icon="VideoPause" link @click="stopTask(row.taskId)" />
                  </el-tooltip>
                  <el-tooltip content="删除">
                    <el-button :icon="Delete" link @click="deleteTask(row.taskId)" />
                  </el-tooltip>
                </el-button-group>
              </template>
            </el-table-column>
          </el-table>
        </div>
        <div class="history-pagination">
          <el-pagination small
            v-model:current-page="taskPagination.currentPage"
            v-model:page-size="taskPagination.pageSize"
            :total="tasks.length"
            :page-sizes="[10, 20, 50, 100, 200]"
            layout="total, sizes, prev, pager, next, jumper"
            background
          />
        </div>
      </div>
    </el-dialog>

    <!-- 指纹管理对话框 -->
    <el-dialog
      v-model="showFingerprintManageDialog"
      width="80%"
      class="fingerprint-manage-dialog"
    >
      <FingerprintManagePanel @refresh="handleFingerprintRefresh" />
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
  </div>
</template>

<script>
import {
  Search,
  VideoPlay,
  VideoPause,
  Download,
  Delete,
  Link,
  CopyDocument,
  View,
  Setting,
  Plus,
  Clock,
  Refresh,
  CircleCheck,
  ChatLineRound,
  Timer,
  Connection
} from '@element-plus/icons-vue'
import { markRaw } from 'vue'
import { EventsOn, EventsOff, BrowserOpenURL } from '../../../wailsjs/runtime'
import { StartFingerprintScan, StopFingerprintScan } from '../../../wailsjs/go/finger/FingerprintController'
import { SaveFile, GetAppPath, ReadFile } from '../../../wailsjs/go/controller/Note'
import * as XLSX from 'xlsx'
import PacketViewer from '../../components/PacketViewer.vue'
import FingerprintManagePanel from './FingerprintManagePanel.vue'
import CustomTabs from '../../components/CustomTabs.vue'
import TableColumnFilterHeader from '../../components/TableColumnFilterHeader.vue'

export default {
  name: 'FingerprintScanner',
  components: {
    Search,
    VideoPlay,
    VideoPause,
    Download,
    Delete,
    Link,
    CopyDocument,
    View,
    Setting,
    Plus: markRaw(Plus),
    Clock: markRaw(Clock),
    Refresh: markRaw(Refresh),
    CircleCheck: markRaw(CircleCheck),
    ChatLineRound: markRaw(ChatLineRound),
    Timer: markRaw(Timer),
    Connection: markRaw(Connection),
    CustomTabs
  },
  data() {
    return {
      Search: markRaw(Search),
      VideoPlay: markRaw(VideoPlay),
      VideoPause: markRaw(VideoPause),
      Download: markRaw(Download),
      Delete: markRaw(Delete),
      Link: markRaw(Link),
      CopyDocument: markRaw(CopyDocument),
      View: markRaw(View),
      Setting: markRaw(Setting),
      Plus: markRaw(Plus),
      Clock: markRaw(Clock),
      Refresh: markRaw(Refresh),
      CircleCheck: markRaw(CircleCheck),
      Timer: markRaw(Timer),
      Connection: markRaw(Connection),
      ChatLineRound: markRaw(ChatLineRound),
      scanTabs: [],
      activeTabId: null,
      historyDialog: false,
      historyList: [],
      historyPagination: {
        page: 1,
        size: 20
      },
      historyLoading: false,
      // 任务管理（独立于标签页）
      tasksLocal: {},
      tasks: [],
      taskPagination: {
        currentPage: 1,
        pageSize: 10
      },
      taskIdCounter: 0,
      operationLock: new Set(),
      showAdvancedDialog: false,
      currentAdvancedTab: null,
      selectedTab: null,
      tableHeight: 420,
      selectedResult: null,
      detailDialogVisible: false,
      contextMenuVisible: false,
      contextMenuPos: { x: 0, y: 0 },
      selectedRow: null,
      packetViewerVisible: false,
      showFingerprintManageDialog: false,
      // 标签页右键菜单
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: '',
      // 联动请求防抖：防止短时间内同一 sourceTaskId 的联动请求创建多个标签页
      fingerprintFillDebounceMap: {},
      // 收集同一 sourceTaskId 的 targets
      fingerprintFillPendingTargets: {}
    }
  },
  computed: {
    currentTab() {
      return this.scanTabs.find(tab => tab.id === this.activeTabId) || this.scanTabs[0]
    },
    paginatedHistory() {
      const start = (this.historyPagination.page - 1) * this.historyPagination.size
      return this.historyList.slice(start, start + this.historyPagination.size)
    },
    paginatedTasks() {
      const start = (this.taskPagination.currentPage - 1) * this.taskPagination.pageSize
      const end = start + this.taskPagination.pageSize
      return this.tasks.slice(start, end)
    },
    contextMenuStyle() {
      // 直接使用鼠标坐标，不做任何调整，确保菜单显示在鼠标点击位置
      const { x, y } = this.contextMenuPos
      
      return {
        top: `${y}px`,
        left: `${x}px`,
        position: 'fixed'
      }
    },
    tabsContextMenuStyle() {
      // 直接从事件获取坐标，优先使用 clientX/clientY（相对于视口，适用于 position: fixed）
      const { x, y } = this.tabsContextMenuPosition
      
      // 直接设置坐标，确保菜单显示在鼠标点击位置
      return {
        top: `${y}px`,
        left: `${x}px`,
        position: 'fixed'
      }
    }
  },
  watch: {
    'currentTab.scanning'(val) {
      if (val) {
        this.startTimer(this.currentTab)
      } else {
        this.stopTimer(this.currentTab)
      }
    }
  },
  mounted() {
    this.bindEvents()
    this.calcTableHeight()
    this.loadHistory()
    this.loadTasks()
    window.addEventListener('resize', this.calcTableHeight)
    window.addEventListener('cyberSabre-fingerprint-fill', this.handleExternalFingerprintFill)

    // 标记指纹识别模块已就绪，供父级事件代理判断
    window.__fingerprintReady = true
    // 标记组件已激活（用于 keep-alive 场景）
    window.__fingerprintActivated = true

    // 如果在本模块挂载之前有主机扫描联动过来的待处理任务，这里一次性处理
    if (Array.isArray(window.__cyberSabrePendingFingerprint) && window.__cyberSabrePendingFingerprint.length > 0) {
      try {
        window.__cyberSabrePendingFingerprint.forEach(payload => {
          this.handleExternalFingerprintFill({ detail: payload })
        })
      } catch (e) {
        console.error('处理待处理指纹识别联动数据失败:', e)
      }
      window.__cyberSabrePendingFingerprint = []
    }

    if (this.scanTabs.length === 0) {
      this.createNewTab()
    }
    this.scanTabs.forEach(t => this.ensureTabColumnFilters(t))
  },

  // keep-alive 激活时
  activated() {
    // 标记组件已激活
    window.__fingerprintActivated = true
    // 处理待处理的联动任务（如果切换标签页时有新的联动任务）
    if (Array.isArray(window.__cyberSabrePendingFingerprint) && window.__cyberSabrePendingFingerprint.length > 0) {
      this.$nextTick(() => {
        try {
          window.__cyberSabrePendingFingerprint.forEach(payload => {
            this.handleExternalFingerprintFill({ detail: payload })
          })
          window.__cyberSabrePendingFingerprint = []
        } catch (e) {
          console.error('处理待处理指纹识别联动数据失败:', e)
        }
      })
    }
  },

  // keep-alive 失活时
  deactivated() {
    // 标记组件未激活
    window.__fingerprintActivated = false
  },

  beforeUnmount() {
    this.unbindEvents()
    window.removeEventListener('resize', this.calcTableHeight)
    window.removeEventListener('cyberSabre-fingerprint-fill', this.handleExternalFingerprintFill)
    this.scanTabs.forEach(tab => {
      if (tab.timer) {
        clearInterval(tab.timer)
      }
    })
    if (window.__fingerprintReady) {
      window.__fingerprintReady = false
    }
    if (window.__fingerprintActivated) {
      window.__fingerprintActivated = false
    }
  },
  methods: {
    // 标签页管理
    createNewTab() {
      const tabId = `fingerprint-${Date.now()}`
      const tab = {
        id: tabId,
        name: `指纹识别-${this.scanTabs.length + 1}`,
        urlsText: '',
        config: {
          timeout: 5,
          threads: 30,
          dirThreads: 20,
          maxDirChecks: 400,
          dirMode: false,
          useWappalyzer: true,
          useTideRules: true,
          useCmsDict: true,
          usePoweredBy: true,
          checkAlive: false,
          followRedirect: true,
          customHeaders: '',
          filter: {
            statusCodes: ['0']
          }
        },
        scanning: false,
        loading: false,
        completed: false,
        scanProgress: 0,
        results: [],
        stats: {
          totalTargets: 0,
          completed: 0,
          success: 0,
          failed: 0,
          targetsPerSecond: 0
        },
        searchKeyword: '',
        columnFilters: {
          status: '',
          url: '',
          title: '',
          length: '',
          technologies: '',
          responseTime: ''
        },
        sortField: 'status',
        sortOrder: 'desc',
        currentPage: 1,
        pageSize: 10,
        elapsedTime: '00:00',
        scanStartTime: null,
        timer: null,
        createTime: Date.now()
      }
      this.scanTabs.push(tab)
      this.activeTabId = tabId
      return tab
    },
    async removeTab(tabId) {
      const index = this.scanTabs.findIndex(tab => tab.id === tabId)
      if (index !== -1) {
        const tab = this.scanTabs[index]
        // 关闭标签页时，不停止扫描，允许后台继续运行
        // 清理定时器（仅前端的定时器，不影响后端扫描）
        if (tab.timer) {
          clearInterval(tab.timer)
          tab.timer = null
        }
        // 保存当前任务数据，但保持任务状态（如果正在扫描则保持为 running）
        // 重要：关闭标签页时，任务数据必须保留在任务管理中，不能删除
        if (tab.taskId) {
          const task = this.tasksLocal[tab.taskId]
          if (task) {
            // 保存完整的结果数组到任务中
            task.results = JSON.parse(JSON.stringify(tab.results || []))
            task.resultsCount = tab.results ? tab.results.length : 0
            task.stats = JSON.parse(JSON.stringify(tab.stats || {}))
            task.progress = tab.scanProgress || 0
            
            // 如果正在扫描，保持任务状态为 running，不改为 stopped
            if (tab.scanning && task.status !== 'running') {
              task.status = 'running'
            } else if (!tab.scanning && task.status === 'running') {
              // 如果标签页已停止扫描，但任务状态还是running，保持running状态
              // 因为可能在其他地方还在运行
            }
            
            // 保存任务数据到localStorage
            await this.saveTasks()
          }
          // 不断开关联，让任务继续在后台运行，任务数据保留在任务管理中
        }
        this.scanTabs.splice(index, 1)
        if (this.scanTabs.length > 0) {
          this.activeTabId = this.scanTabs[0].id
        } else {
          // 避免进入无标签空白态，导致后续弹层/布局异常
          this.createNewTab()
        }
      }
    },
    handleTabClick(tab) {
      this.activeTabId = tab.paneName
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
          : this.activeTabId

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

      const targetId = this.tabsContextMenuTabId || this.activeTabId
      const targetTab = this.scanTabs.find(tab => tab.id === targetId)
      
      if (!targetTab) {
        return
      }

      // 关闭其他标签页，不停止扫描，允许后台继续运行
      const tabsToClose = this.scanTabs.filter(tab => tab.id !== targetId)
      for (const tab of tabsToClose) {
        // 不停止扫描，允许后台继续运行
        // 清理定时器（仅前端的定时器，不影响后端扫描）
        if (tab.timer) {
          clearInterval(tab.timer)
          tab.timer = null
        }
        // 保存当前任务数据，但保持任务状态（如果正在扫描则保持为 running）
        // 重要：关闭标签页时，任务数据必须保留在任务管理中，不能删除
        if (tab.taskId) {
          const task = this.tasksLocal[tab.taskId]
          if (task) {
            // 保存完整的结果数组到任务中
            task.results = JSON.parse(JSON.stringify(tab.results || []))
            task.resultsCount = tab.results ? tab.results.length : 0
            task.stats = JSON.parse(JSON.stringify(tab.stats || {}))
            task.progress = tab.scanProgress || 0
            
            // 如果正在扫描，保持任务状态为 running，不改为 stopped
            if (tab.scanning && task.status !== 'running') {
              task.status = 'running'
            }
            
            // 保存任务数据到localStorage
            await this.saveTasks()
          }
          // 不断开关联，让任务继续在后台运行，任务数据保留在任务管理中
        }
      }

      // 更新标签列表，只保留目标标签
      this.scanTabs = [targetTab]
      this.activeTabId = targetId
    },
    // 关闭所有标签页
    async handleCloseAllTabs() {
      this.hideTabsContextMenu()
      
      // 关闭所有标签页，不停止扫描，允许后台继续运行
      for (const tab of this.scanTabs) {
        // 不停止扫描，允许后台继续运行
        // 清理定时器（仅前端的定时器，不影响后端扫描）
        if (tab.timer) {
          clearInterval(tab.timer)
          tab.timer = null
        }
        // 保存当前任务数据，但保持任务状态（如果正在扫描则保持为 running）
        // 重要：关闭标签页时，任务数据必须保留在任务管理中，不能删除
        if (tab.taskId) {
          const task = this.tasksLocal[tab.taskId]
          if (task) {
            // 保存完整的结果数组到任务中
            task.results = JSON.parse(JSON.stringify(tab.results || []))
            task.resultsCount = tab.results ? tab.results.length : 0
            task.stats = JSON.parse(JSON.stringify(tab.stats || {}))
            task.progress = tab.scanProgress || 0
            
            // 如果正在扫描，保持任务状态为 running，不改为 stopped
            if (tab.scanning && task.status !== 'running') {
              task.status = 'running'
            }
            
            // 保存任务数据到localStorage
            await this.saveTasks()
          }
          // 不断开关联，让任务继续在后台运行，任务数据保留在任务管理中
        }
      }

      // 清空所有标签页
      this.scanTabs = []
      // 保留一个空白任务标签，避免页面进入 empty 态
      this.createNewTab()
    },
    // 保存历史记录到文件
    async saveHistoryToFile(historyId, historyData) {
      try {
        const appPath = await GetAppPath()
        const pathSeparator = appPath.includes('\\') ? '\\' : '/'
        const resultsDir = `${appPath}${pathSeparator}logs${pathSeparator}fingerprint${pathSeparator}history`
        
        const timestamp = Date.now()
        const fileName = `fingerprint_history_${historyId}_${timestamp}.json`
        const filePath = `${resultsDir}${pathSeparator}${fileName}`
        
        await SaveFile(filePath, JSON.stringify(historyData, null, 2))
        return filePath
      } catch (error) {
        console.error('保存历史记录到文件失败:', error)
        return null
      }
    },
    // 从文件加载历史记录
    async loadHistoryFromFile(filePath) {
      try {
        if (!filePath) return null
        const content = await ReadFile(filePath)
        return JSON.parse(content)
      } catch (error) {
        console.error('从文件加载历史记录失败:', error)
        return null
      }
    },
    // 历史记录管理
    async loadHistory() {
      try {
        const historyStr = localStorage.getItem('fingerprint-history')
        if (historyStr) {
          const history = JSON.parse(historyStr)
          
          // 从文件加载完整结果
          for (const item of history) {
            if (item.resultsFilePath && (!item.results || item.results.length === 0)) {
              const fileData = await this.loadHistoryFromFile(item.resultsFilePath)
              if (fileData) {
                item.results = fileData.results || []
                item.stats = fileData.stats || item.stats || {}
              }
            }
          }
          
          this.historyList = history
        }
      } catch (e) {
        console.error('加载历史记录失败:', e)
        this.historyList = []
      }
    },
    async saveHistory() {
      try {
        const history = this.scanTabs
          .filter(tab => tab.completed && tab.results.length > 0)
          .map(tab => ({
            id: tab.id,
            name: tab.name,
            urlsText: tab.urlsText,
            config: tab.config,
            results: tab.results,
            stats: tab.stats,
            createTime: tab.createTime
          }))
        
        // 将完整结果保存到文件，localStorage只保存元数据
        const compressedHistory = []
        for (const item of history) {
          if (item.results && item.results.length > 0 && !item.resultsFilePath) {
            const filePath = await this.saveHistoryToFile(item.id, {
              results: item.results,
              stats: item.stats
            })
            if (filePath) {
              item.resultsFilePath = filePath
            }
          }
          
          compressedHistory.push({
            id: item.id,
            name: item.name,
            urlsText: item.urlsText,
            config: item.config,
            results: undefined, // 不保存完整结果到localStorage
            resultsCount: item.results ? item.results.length : 0,
            stats: item.stats || {},
            createTime: item.createTime,
            resultsFilePath: item.resultsFilePath || null
          })
        }
        
        localStorage.setItem('fingerprint-history', JSON.stringify(compressedHistory))
        this.historyList = history // 保持完整数据在内存中
      } catch (e) {
        console.error('保存历史记录失败:', e)
      }
    },
    async refreshTasks() {
      this.historyLoading = true
      await this.loadTasks()
      
      // 同步任务状态到标签页
      this.syncTaskStatusToTabs()
      
      this.$nextTick(() => {
        this.historyLoading = false
      })
    },
    // 同步任务状态到标签页
    syncTaskStatusToTabs() {
      // 遍历所有任务，如果标签页存在，同步状态
      for (const [taskId, task] of Object.entries(this.tasksLocal)) {
        const tab = this.scanTabs.find(t => t.taskId === taskId)
        if (tab) {
          // 同步任务基本信息（确保任务管理和标签页显示一致）
          if (task.name && task.name !== tab.name) {
            tab.name = task.name
          }
          if (task.urlsText && task.urlsText !== tab.urlsText) {
            tab.urlsText = task.urlsText
          }
        if (!tab.columnFilters) {
          tab.columnFilters = {
            status: '',
            url: '',
            title: '',
            length: '',
            technologies: '',
            responseTime: ''
          }
        }
        if (tab.sortField === undefined) {
          tab.sortField = 'status'
          tab.sortOrder = 'desc'
        }
        if (task.config) {
          tab.config = JSON.parse(JSON.stringify(task.config))
        }
          
          // 同步任务状态
          tab.scanning = task.status === 'running'
          tab.completed = task.status === 'completed'
          tab.scanProgress = task.progress || 0
          
          // 同步统计信息
          if (task.stats) {
            tab.stats = {
              ...tab.stats,
              ...task.stats,
              totalTargets: task.total || task.stats.totalTargets || tab.stats.totalTargets || 0,
              completed: task.completed || task.stats.completed || tab.stats.completed || 0
            }
          }
          
          // 如果任务有结果数据，同步结果（但不要覆盖标签页中可能更新的数据）
          // 只有当标签页没有结果或结果数量不匹配时才更新
          if (task.results && Array.isArray(task.results) && task.results.length > 0) {
            if (!tab.results || tab.results.length === 0 || tab.results.length < task.results.length) {
              // 从文件加载完整结果
              if (task.resultsFilePath) {
                this.loadResultsFromFile(task.resultsFilePath).then(fileData => {
                  if (fileData && fileData.results) {
                    const tab = this.scanTabs.find(t => t.taskId === taskId)
                    if (tab) {
                      tab.results = JSON.parse(JSON.stringify(fileData.results))
                      if (fileData.stats) {
                        tab.stats = { ...tab.stats, ...fileData.stats }
                      }
                    }
                  }
                }).catch(err => {
                  console.warn('同步任务结果失败:', err)
                })
              } else if (task.results) {
                tab.results = JSON.parse(JSON.stringify(task.results))
              }
            }
          }
        }
      }
    },
    // 保存扫描结果到本地JSON文件
    async saveResultsToFile(taskId, results, stats) {
      try {
        const appPath = await GetAppPath()
        // 使用路径分隔符兼容不同系统
        const pathSeparator = appPath.includes('\\') ? '\\' : '/'
        const resultsDir = `${appPath}${pathSeparator}logs${pathSeparator}fingerprint`
        
        // 生成文件名（使用任务ID和时间戳）
        const timestamp = Date.now()
        const fileName = `fingerprint_${taskId}_${timestamp}.json`
        const filePath = `${resultsDir}${pathSeparator}${fileName}`
        
        const dataToSave = {
          taskId,
          timestamp: new Date().toISOString(),
          results: results || [],
          stats: stats || {}
        }
        
        // SaveFile 会自动创建目录（如果不存在）
        await SaveFile(filePath, JSON.stringify(dataToSave, null, 2))
        return filePath
      } catch (error) {
        console.error('保存结果到文件失败:', error)
        return null
      }
    },
    // 从文件加载扫描结果
    async loadResultsFromFile(filePath) {
      try {
        if (!filePath) return null
        const content = await ReadFile(filePath)
        const data = JSON.parse(content)
        return data
      } catch (error) {
        console.error('从文件加载结果失败:', error)
        return null
      }
    },
    // 任务管理方法
    async saveTasks() {
      try {
        // 压缩任务数据：完整结果保存到文件，localStorage只保存元数据
        const compressedTasks = {}
        for (const [taskId, task] of Object.entries(this.tasksLocal)) {
          // 如果任务有完整结果，保存到文件（无论是否已有文件路径，都要更新）
          if (task.results && Array.isArray(task.results) && task.results.length > 0) {
            // 如果已有文件路径，也重新保存以更新内容
            const filePath = await this.saveResultsToFile(
              taskId,
              task.results,
              task.stats
            )
            if (filePath) {
              task.resultsFilePath = filePath
            }
          }
          
          compressedTasks[taskId] = {
            ...task,
            // 不保存完整结果数组到localStorage，只保存文件路径
            results: undefined,
            resultsCount: task.results ? task.results.length : 0,
            stats: task.stats || {},
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
          
          localStorage.setItem('fingerprintTasks', JSON.stringify(limitedTasks))
        } else {
          localStorage.setItem('fingerprintTasks', tasksStr)
        }
      } catch (error) {
        if (error.name === 'QuotaExceededError') {
          // 存储空间不足，清理旧数据
          console.warn('存储空间不足，清理旧任务')
          try {
            const taskArray = Object.entries(this.tasksLocal)
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
            
            localStorage.setItem('fingerprintTasks', JSON.stringify(limitedTasks))
            this.tasksLocal = limitedTasks
            this.tasks = Object.values(this.tasksLocal)
          } catch (e) {
            console.error('清理后保存任务仍然失败:', e)
            localStorage.removeItem('fingerprintTasks')
          }
        } else {
          console.error('保存任务失败:', error)
        }
      }
    },
    async loadTasks() {
      try {
        const tasksStr = localStorage.getItem('fingerprintTasks')
        if (tasksStr) {
          const tasksLocal = JSON.parse(tasksStr)
          
          // 从文件加载完整结果
          for (const [taskId, task] of Object.entries(tasksLocal)) {
            if (task.resultsFilePath && !task.results) {
              const fileData = await this.loadResultsFromFile(task.resultsFilePath)
              if (fileData) {
                task.results = fileData.results || []
                task.stats = fileData.stats || task.stats || {}
              }
            }
          }
          
          this.tasksLocal = tasksLocal
          this.tasks = Object.values(this.tasksLocal)
        }
      } catch (error) {
        console.error('加载任务失败:', error)
        this.tasksLocal = {}
        this.tasks = []
      }
    },
    getStatusTagType(status) {
      const statusMap = {
        running: 'success',
        paused: 'warning',
        stopped: 'info',
        completed: 'success'
      }
      return statusMap[status] || 'info'
    },
    getStatusText(status) {
      const statusMap = {
        running: '运行中',
        paused: '已暂停',
        stopped: '已停止',
        completed: '已完成'
      }
      return statusMap[status] || '未知'
    },
    getSourceTaskTypeName(type) {
      const typeMap = {
        hostscan: '主机扫描',
        dirsearch: '目录扫描',
        fingerprint: '指纹识别',
        webscan: '漏洞扫描'
      }
      return typeMap[type] || type || '未知'
    },
    async loadTask(taskId) {
      const task = this.tasksLocal[taskId]
      if (!task) {
        this.$message.error('任务不存在')
        return
      }
      
      // 如果任务结果在文件中，从文件加载完整数据
      let results = task.results || []
      let stats = task.stats || {}
      if (task.resultsFilePath) {
        // 优先从文件加载，确保数据是最新的
        const fileData = await this.loadResultsFromFile(task.resultsFilePath)
        if (fileData) {
          results = fileData.results || []
          stats = fileData.stats || task.stats || {}
          // 更新任务数据到内存
          task.results = results
          task.stats = stats
        } else if (task.results && Array.isArray(task.results) && task.results.length > 0) {
          // 如果文件加载失败，使用内存中的数据
          results = JSON.parse(JSON.stringify(task.results))
          stats = JSON.parse(JSON.stringify(task.stats || {}))
        }
      } else if (task.results && Array.isArray(task.results) && task.results.length > 0) {
        // 如果没有文件路径，直接使用内存中的数据
        results = JSON.parse(JSON.stringify(task.results))
        stats = JSON.parse(JSON.stringify(task.stats || {}))
      }
      
      // 查找是否已有该任务的标签页
      let tab = this.scanTabs.find(t => t.taskId === taskId)
      if (!tab) {
        // 创建新标签页
        const tabId = `tab-${Date.now()}`
        tab = {
          id: tabId,
          name: task.name,
          urlsText: task.urlsText || '',
          config: task.config ? JSON.parse(JSON.stringify(task.config)) : {
            timeout: 10,
            threads: 50,
            dirThreads: 10,
            maxDirChecks: 5,
            dirMode: false,
            useWappalyzer: true,
            useTideRules: true,
            useCmsDict: true,
            usePoweredBy: true,
            checkAlive: false,
            followRedirect: true,
            customHeaders: '',
            filter: {
              statusCodes: ['0']
            }
          },
          scanning: task.status === 'running',
          loading: false,
          completed: task.status === 'completed',
          results: Array.isArray(results) ? JSON.parse(JSON.stringify(results)) : [], // 加载完整结果数组
          scanProgress: task.progress || 0,
          scanStartTime: null,
          stats: {
            totalTargets: task.total || stats.totalTargets || (Array.isArray(results) ? results.length : 0),
            completed: task.completed || stats.completed || (Array.isArray(results) ? results.length : 0),
            success: stats.success || 0,
            failed: stats.failed || 0,
            targetsPerSecond: 0
          },
          searchKeyword: '',
          columnFilters: {
            status: '',
            url: '',
            title: '',
            length: '',
            technologies: '',
            responseTime: ''
          },
          sortField: 'status',
          sortOrder: 'desc',
          currentPage: 1,
          pageSize: 10,
          createTime: task.createdAt || Date.now(),
          taskId: taskId,
          sourceTaskId: task.sourceTaskId || null,
          sourceTaskType: task.sourceTaskType || null,
          timer: null,
          elapsedTime: '00:00'
        }
        this.scanTabs.push(tab)
      } else {
        // 如果标签页已存在，更新标签页数据（确保数据同步）
        tab.name = task.name
        tab.urlsText = task.urlsText || ''
        tab.config = task.config ? JSON.parse(JSON.stringify(task.config)) : tab.config
        tab.scanning = task.status === 'running'
        tab.completed = task.status === 'completed'
        tab.results = Array.isArray(results) ? JSON.parse(JSON.stringify(results)) : []
        tab.scanProgress = task.progress || 0
        tab.stats = {
          totalTargets: task.total || stats.totalTargets || tab.stats.totalTargets || 0,
          completed: task.completed || stats.completed || tab.stats.completed || 0,
          success: stats.success || tab.stats.success || 0,
          failed: stats.failed || tab.stats.failed || 0,
          targetsPerSecond: tab.stats.targetsPerSecond || 0
        }
        if (!tab.columnFilters) {
          tab.columnFilters = {
            status: '',
            url: '',
            title: '',
            length: '',
            technologies: '',
            responseTime: ''
          }
        }
        if (tab.sortField === undefined) {
          tab.sortField = 'status'
          tab.sortOrder = 'desc'
        }
      }
      
      // 切换到该标签页
      this.activeTabId = tab.id
      this.historyDialog = false
      this.$message.success('任务已加载到标签页')
    },
    async stopTask(taskId) {
      if (this.operationLock.has(taskId)) {
        return
      }
      this.operationLock.add(taskId)
      try {
        const task = this.tasksLocal[taskId]
        if (!task) {
          this.$message.warning('任务不存在')
          return
        }
        
        // 找到关联的标签页并停止
        const tab = this.scanTabs.find(t => t.taskId === taskId)
        if (tab && tab.scanning) {
          await this.stopScan(tab)
        }
        
        task.status = 'stopped'
        this.saveTasks()
        this.$message.success('任务已停止')
      } catch (error) {
        console.error('停止任务失败:', error)
        this.$message.error('停止失败')
      } finally {
        setTimeout(() => {
          this.operationLock.delete(taskId)
        }, 1000)
      }
    },
    async deleteTask(taskId) {
      try {
        await this.$confirm('确定删除该任务？', '删除任务', { type: 'warning' })
      } catch {
        return
      }
      delete this.tasksLocal[taskId]
      this.tasks = Object.values(this.tasksLocal)
      this.saveTasks()
      const tab = this.scanTabs.find(t => t.taskId === taskId)
      if (tab) {
        tab.taskId = ''
        tab.scanning = false
      }
      this.$message.success('删除成功')
    },
    async clearAllTasks() {
      try {
        await this.$confirm('确定清除所有任务？此操作不可恢复。', '清除全部任务', { type: 'warning' })
      } catch {
        return
      }
      this.tasksLocal = {}
      this.tasks = []
      localStorage.removeItem('fingerprintTasks')
      this.scanTabs.forEach(tab => {
        tab.taskId = ''
        tab.scanning = false
      })
      this.$message.success('已清除全部任务')
    },
    handleHistoryDialogOpened() {
      this.refreshTasks()
    },
    refreshTasks() {
      this.loadHistory()
    },
    async loadHistoryTab(historyItem) {
      // 如果历史记录结果在文件中，从文件加载
      let results = historyItem.results || []
      let stats = historyItem.stats || {}
      if (historyItem.resultsFilePath && (!historyItem.results || historyItem.results.length === 0)) {
        const fileData = await this.loadHistoryFromFile(historyItem.resultsFilePath)
        if (fileData) {
          results = fileData.results || []
          stats = fileData.stats || historyItem.stats || {}
        }
      }
      
      const tab = {
        ...historyItem,
        results: results,
        stats: stats,
        scanning: false,
        loading: false,
        completed: true,
        scanProgress: 100,
        searchKeyword: '',
        columnFilters: {
          status: '',
          url: '',
          title: '',
          length: '',
          technologies: '',
          responseTime: ''
        },
        sortField: 'status',
        sortOrder: 'desc',
        currentPage: 1,
        pageSize: 10,
        elapsedTime: '00:00',
        scanStartTime: null,
        timer: null
      }
      this.scanTabs.push(tab)
      this.activeTabId = tab.id
      this.historyDialog = false
    },
    async deleteHistory(id) {
      try {
        await this.$confirm('确定要删除这条历史记录吗？', '确认删除', { type: 'warning' })
      } catch {
        return
      }
      this.historyList = this.historyList.filter(item => item.id !== id)
      localStorage.setItem('fingerprint-history', JSON.stringify(this.historyList))
      this.$message.success('已删除')
    },
    async clearAllHistory() {
      try {
        await this.$confirm('确定要清空所有历史记录吗？', '确认清空', { type: 'warning' })
      } catch {
        return
      }
      this.historyList = []
      localStorage.removeItem('fingerprint-history')
      this.$message.success('已清空')
    },
    formatTime(timestamp) {
      if (!timestamp) return '-'
      const date = new Date(timestamp)
      return date.toLocaleString('zh-CN')
    },
    // 格式化时间（秒数转可读格式）
    formatDuration(seconds) {
      if (!seconds || seconds < 0) return '0秒'
      if (seconds < 60) return `${Math.round(seconds)}秒`
      if (seconds < 3600) {
        const mins = Math.floor(seconds / 60)
        const secs = Math.round(seconds % 60)
        return secs > 0 ? `${mins}分${secs}秒` : `${mins}分`
      }
      const hours = Math.floor(seconds / 3600)
      const mins = Math.floor((seconds % 3600) / 60)
      const secs = Math.round(seconds % 60)
      if (mins === 0 && secs === 0) return `${hours}小时`
      if (secs === 0) return `${hours}小时${mins}分`
      return `${hours}小时${mins}分${secs}秒`
    },
    // 格式化数字（添加千分位）
    formatNumber(num) {
      if (num === null || num === undefined) return '0'
      return num.toLocaleString('zh-CN')
    },
    ensureTabColumnFilters(tab) {
      if (!tab || tab.columnFilters) return
      tab.columnFilters = {
        status: '',
        url: '',
        title: '',
        length: '',
        technologies: '',
        responseTime: ''
      }
    },
    resetColumnFilters(tab) {
      this.ensureTabColumnFilters(tab)
      Object.assign(tab.columnFilters, {
        status: '',
        url: '',
        title: '',
        length: '',
        technologies: '',
        responseTime: ''
      })
    },
    setTabSort(tab, prop, order) {
      if (!tab) return
      this.ensureTabColumnFilters(tab)
      tab.sortField = prop
      tab.sortOrder = order
      tab.currentPage = 1
    },
    handleColumnFilterInput(tab) {
      if (!tab) return
      tab.currentPage = 1
    },
    getFingerprintTechSearchText(row) {
      if (!row) return ''
      return (this.getAllTechnologies(row) || []).join(' ').toLowerCase()
    },
    // 辅助方法
    getFilteredResults(tab) {
      if (!tab) return []
      this.ensureTabColumnFilters(tab)
      let filtered = tab.results || []

      if (tab.config?.filter?.statusCodes && tab.config.filter.statusCodes.length > 0) {
        filtered = filtered.filter(item => {
          let status = item.status
          if (status === undefined || status === null) {
            status = 0
          }
          const statusStr = String(status)
          const statusNum = Number(status)
          const shouldFilter = tab.config.filter.statusCodes.some(code => {
            const codeNum = Number(code)
            return code === statusStr || codeNum === statusNum
          })
          return !shouldFilter
        })
      }

      if (tab.searchKeyword) {
        const keyword = tab.searchKeyword.toLowerCase()
        filtered = filtered.filter(item => {
          return (
            item.url?.toLowerCase().includes(keyword) ||
            (item.title || '').toLowerCase().includes(keyword) ||
            (item.technologies || []).some(tech => tech.toLowerCase().includes(keyword))
          )
        })
      }

      const cf = tab.columnFilters
      const st = (cf.status || '').trim().toLowerCase()
      if (st) {
        filtered = filtered.filter(item => String(item.status ?? '').toLowerCase().includes(st))
      }
      const uq = (cf.url || '').trim().toLowerCase()
      if (uq) {
        filtered = filtered.filter(item => (item.url || '').toLowerCase().includes(uq))
      }
      const tq = (cf.title || '').trim().toLowerCase()
      if (tq) {
        filtered = filtered.filter(item => (item.title || '').toLowerCase().includes(tq))
      }
      const lq = (cf.length || '').trim().toLowerCase()
      if (lq) {
        filtered = filtered.filter(item => {
          const raw = String(item.length ?? '')
          const fmt = String(this.formatLength(item.length) || '').toLowerCase()
          return raw.includes(lq) || fmt.includes(lq)
        })
      }
      const techQ = (cf.technologies || '').trim().toLowerCase()
      if (techQ) {
        filtered = filtered.filter(item => this.getFingerprintTechSearchText(item).includes(techQ))
      }
      const rt = (cf.responseTime || '').trim().toLowerCase()
      if (rt) {
        filtered = filtered.filter(item => String(item.responseTime ?? '').toLowerCase().includes(rt))
      }
      return filtered
    },
    getPagedResults(tab) {
      if (!tab) return []
      const filtered = this.getFilteredResults(tab)
      const sorted = [...filtered].map(item => ({
        ...item,
        techCount: item.technologies ? item.technologies.length : 0
      }))
      const prop = tab.sortField
      const order = tab.sortOrder
      if (prop && order) {
        sorted.sort((a, b) => {
          let aVal
          let bVal
          if (prop === 'status' || prop === 'length' || prop === 'responseTime') {
            aVal = Number(a[prop]) || 0
            bVal = Number(b[prop]) || 0
          } else if (prop === 'technologies') {
            aVal = this.getFingerprintTechSearchText(a)
            bVal = this.getFingerprintTechSearchText(b)
          } else {
            aVal = String(a[prop] ?? '').toLowerCase()
            bVal = String(b[prop] ?? '').toLowerCase()
          }
          let cmp = 0
          if (typeof aVal === 'string') {
            cmp = aVal.localeCompare(bVal)
          } else {
            cmp = aVal < bVal ? -1 : aVal > bVal ? 1 : 0
          }
          return order === 'asc' ? cmp : -cmp
        })
      }
      const start = (tab.currentPage - 1) * tab.pageSize
      return sorted.slice(start, start + tab.pageSize)
    },
    handleSizeChange(size, tab) {
      tab.pageSize = size
      tab.currentPage = 1
    },
    handleCurrentChange(page, tab) {
      tab.currentPage = page
    },
    saveAdvancedConfig() {
      this.showAdvancedDialog = false
      this.$message.success('配置已保存')
    },
    bindEvents() {
      EventsOn('fingerprintResult', this.handleScanResult)
      EventsOn('fingerprintProgress', this.handleScanProgress)
      // EventsOn('fingerprintError', this.handleScanError)
      EventsOn('fingerprintComplete', this.handleScanComplete)
    },
    unbindEvents() {
      EventsOff('fingerprintResult')
      EventsOff('fingerprintProgress')
      // EventsOff('fingerprintError')
      EventsOff('fingerprintComplete')
    },
    calcTableHeight() {
      this.$nextTick(() => {
        const tabId = String(this.activeTabId)
        if (!tabId) return

        const section = this.$el.querySelector(
          `.results-section[data-dirsearch-tab="${tabId}"]`
        )
        if (!section) return

        const resultsCard = section.querySelector('.results-card')
        const header = resultsCard?.querySelector('.results-header')
        const pagination = section.querySelector('.pagination-wrapper')

        if (resultsCard && header && pagination) {
          const height =
            resultsCard.clientHeight -
            header.clientHeight -
            pagination.clientHeight -
            35   // 留出少量安全余量，可根据实际微调
          this.tableHeight = Math.max(height, 450)  // 最低不低于 450px
        }
      })
    },
    async toggleScan(tab) {
      const currentTab = tab || this.currentTab
      if (!currentTab) return
      if (currentTab.scanning) {
        await this.stopScan(currentTab)
      } else {
        await this.startScan(currentTab)
      }
    },
    async startScan(tab, skipConfirm = false) {
      const currentTab = tab || this.currentTab
      if (!currentTab) return
      
      const urls = currentTab.urlsText
        .split('\n')
        .map(url => url.trim())
        .filter(url => url.length > 0)

      if (urls.length === 0) {
        this.$message.warning('请输入至少一个URL')
        return
      }

      // 如果是联动扫描（skipConfirm=true），保留现有结果
      const shouldKeepResults = skipConfirm && currentTab.results.length > 0

      try {
        currentTab.scanning = true
        currentTab.loading = true
        currentTab.completed = false
        currentTab.scanProgress = 0
        currentTab.scanStartTime = Date.now()
        
        // 如果是联动扫描且已有结果，保留现有统计信息，只更新总数
        if (shouldKeepResults) {
          // 保留现有结果和统计信息，只更新总数（累加新URL数量）
          const existingUrls = new Set(currentTab.results.map(r => r.url))
          const newUrlsCount = urls.filter(url => !existingUrls.has(url)).length
          currentTab.stats.totalTargets = (currentTab.stats.totalTargets || 0) + newUrlsCount
        } else {
          // 首次扫描或手动扫描，重置统计信息
          currentTab.stats = {
            totalTargets: urls.length,
            completed: 0,
            success: 0,
            failed: 0,
            targetsPerSecond: 0
          }
        }

        const scanConfig = {
          targets: urls,
          timeout: currentTab.config.timeout,
          threads: currentTab.config.threads,
          dirThreads: currentTab.config.dirThreads,
          maxDirChecks: currentTab.config.maxDirChecks,
          dirMode: currentTab.config.dirMode,
          useWappalyzer: currentTab.config.useWappalyzer,
          useTideRules: currentTab.config.useTideRules,
          useCmsDict: currentTab.config.useCmsDict,
          usePoweredBy: currentTab.config.usePoweredBy,
          checkAlive: currentTab.config.checkAlive,
          followRedirect: currentTab.config.followRedirect,
          headers: this.parseHeaders(currentTab.config.customHeaders),
          tabId: currentTab.id // 传递tabId用于事件处理
        }

        // 生成任务ID
        const taskId = `fingerprint-${Date.now()}-${++this.taskIdCounter}`
        currentTab.taskId = taskId
        
        // 创建任务记录（保存完整结果数组）
        this.tasksLocal[taskId] = {
          taskId,
          name: currentTab.name || `指纹识别-${new Date().toLocaleTimeString()}`,
          urlsText: currentTab.urlsText,
          config: JSON.parse(JSON.stringify(scanConfig)),
          results: JSON.parse(JSON.stringify(currentTab.results)), // 保存完整结果数组
          resultsCount: currentTab.results.length, // 保留数量字段用于显示
          status: 'running',
          progress: 0,
          completed: 0,
          total: urls.length,
          createdAt: Date.now(),
          // 关联任务信息
          sourceTaskId: currentTab.sourceTaskId || null,
          sourceTaskType: currentTab.sourceTaskType || null
        }
        this.tasks = Object.values(this.tasksLocal)
        this.saveTasks()

        await StartFingerprintScan(JSON.stringify(scanConfig))
        currentTab.loading = false
        this.$message.success('指纹识别已启动')
      } catch (error) {
        const msg = error?.message || error?.toString() || '未知错误'
        this.$message.error(`启动失败: ${msg}`)
        currentTab.scanning = false
        currentTab.loading = false
      }
    },
    async stopScan(tab) {
      const currentTab = tab || this.currentTab
      if (!currentTab) return
      
      // 清理批量更新定时器
      if (currentTab.updateResultsTimer) {
        cancelAnimationFrame(currentTab.updateResultsTimer)
        currentTab.updateResultsTimer = null
      }
      
      // 处理剩余待更新的结果
      if (currentTab.pendingResults && currentTab.pendingResults.length > 0) {
        this.flushPendingResults(currentTab)
      }
      
      try {
        await StopFingerprintScan()
        // 更新任务状态（保存完整结果）
        if (currentTab.taskId && this.tasksLocal[currentTab.taskId]) {
          this.tasksLocal[currentTab.taskId].status = 'stopped'
          this.tasksLocal[currentTab.taskId].results = JSON.parse(JSON.stringify(currentTab.results))
          this.tasksLocal[currentTab.taskId].resultsCount = currentTab.results.length
          this.tasksLocal[currentTab.taskId].stats = JSON.parse(JSON.stringify(currentTab.stats))
          this.tasksLocal[currentTab.taskId].progress = currentTab.scanProgress || 0
          
          // 同步标签页的基本信息到任务（确保任务管理和标签页显示一致）
          if (currentTab.name) {
            this.tasksLocal[currentTab.taskId].name = currentTab.name
          }
          if (currentTab.urlsText) {
            this.tasksLocal[currentTab.taskId].urlsText = currentTab.urlsText
          }
          if (currentTab.config) {
            this.tasksLocal[currentTab.taskId].config = JSON.parse(JSON.stringify(currentTab.config))
          }
          
          this.saveTasks()
        }
        this.$message.info('识别已停止')
      } catch (error) {
        this.$message.error(`停止失败: ${error?.message || error}`)
      } finally {
        currentTab.scanning = false
        currentTab.loading = false
      }
    },
    handleScanResult(result) {
      // 根据后端携带的 tabId 精确找到对应的标签页
      // 不再回退到 activeTabId，避免在有任务运行时新建标签页导致结果串到新标签的问题
      const tabId = result.tabId
      const tab = tabId ? this.scanTabs.find(t => t.id === tabId) : null
      
      // 通过 tabId 找到对应的任务（即使标签已关闭）
      let taskId = null
      if (tab && tab.taskId) {
        taskId = tab.taskId
      } else if (tabId) {
        // 如果标签不存在，通过 tabId 查找任务
        const task = Object.values(this.tasksLocal).find(t => t.config?.tabId === tabId)
        if (task) {
          taskId = task.taskId || task.id
        }
      }
      
      // 如果标签存在，使用批量更新机制优化性能
      if (tab) {
        // 初始化批量更新缓冲区
        if (!tab.pendingResults) {
          tab.pendingResults = []
        }
        if (!tab.pendingStats) {
          tab.pendingStats = {
            completed: 0,
            success: 0,
            failed: 0
          }
        }
        
        // 将结果添加到缓冲区
        tab.pendingResults.push(result)
        tab.pendingStats.completed += 1
        if (result.error) {
          tab.pendingStats.failed += 1
        } else {
          tab.pendingStats.success += 1
        }
        
        // 使用requestAnimationFrame批量更新，减少DOM操作频率
        if (!tab.updateResultsTimer) {
          tab.updateResultsTimer = requestAnimationFrame(() => {
            this.flushPendingResults(tab)
            tab.updateResultsTimer = null
          })
        }
      }
      
      // 同步更新任务数据（无论标签是否存在）
      if (taskId && this.tasksLocal[taskId]) {
        // 初始化任务结果数组（如果不存在）
        if (!this.tasksLocal[taskId].results) {
          this.tasksLocal[taskId].results = []
        }
        
        // 添加结果到数组开头（与标签一致）
        this.tasksLocal[taskId].results.unshift(JSON.parse(JSON.stringify(result)))
        this.tasksLocal[taskId].resultsCount = this.tasksLocal[taskId].results.length
        
        // 更新任务统计信息
        if (tab) {
          this.tasksLocal[taskId].stats = JSON.parse(JSON.stringify(tab.stats))
          // 同步标签页的基本信息到任务（确保任务管理和标签页显示一致）
          if (tab.name) {
            this.tasksLocal[taskId].name = tab.name
          }
          if (tab.urlsText) {
            this.tasksLocal[taskId].urlsText = tab.urlsText
          }
          if (tab.config) {
            this.tasksLocal[taskId].config = JSON.parse(JSON.stringify(tab.config))
          }
        }
        
        // 如果标签页存在，确保标签页数据与任务数据同步
        if (tab && tab.taskId === taskId) {
          // 标签页数据已经更新，这里只需要确保任务数据是最新的
        }
        
        this.saveTasks()
      }
    },
    handleScanProgress(progress) {
      // 根据后端携带的 tabId 精确找到对应的标签页
      const tabId = progress?.tabId
      const tab = tabId ? this.scanTabs.find(t => t.id === tabId) : null
      
      // 通过 tabId 找到对应的任务（即使标签已关闭）
      let taskId = null
      if (tab && tab.taskId) {
        taskId = tab.taskId
      } else if (tabId) {
        // 如果标签不存在，通过 tabId 查找任务
        const task = Object.values(this.tasksLocal).find(t => t.config?.tabId === tabId)
        if (task) {
          taskId = task.taskId || task.id
        }
      }
      
      // 如果标签存在，更新标签数据
      if (tab) {
        if (progress?.percentage !== undefined) {
          tab.scanProgress = progress.percentage
        }
        if (progress?.stats) {
          tab.stats = {
            ...tab.stats,
            ...progress.stats
          }
        }
      }
      
      // 同步更新任务数据（无论标签是否存在）
      if (taskId && this.tasksLocal[taskId]) {
        if (progress?.percentage !== undefined) {
          this.tasksLocal[taskId].progress = progress.percentage
        }
        if (progress?.stats) {
          this.tasksLocal[taskId].completed = progress.stats.completed || (tab ? tab.stats.completed : 0)
          this.tasksLocal[taskId].total = progress.stats.totalTargets || (tab ? tab.stats.totalTargets : 0)
          this.tasksLocal[taskId].stats = JSON.parse(JSON.stringify(progress.stats))
        } else if (tab) {
          this.tasksLocal[taskId].completed = tab.stats.completed
          this.tasksLocal[taskId].total = tab.stats.totalTargets
          this.tasksLocal[taskId].stats = JSON.parse(JSON.stringify(tab.stats))
        }
        
        // 如果标签页存在，同步标签页的基本信息到任务（确保任务管理和标签页显示一致）
        if (tab) {
          if (tab.name) {
            this.tasksLocal[taskId].name = tab.name
          }
          if (tab.urlsText) {
            this.tasksLocal[taskId].urlsText = tab.urlsText
          }
          if (tab.config) {
            this.tasksLocal[taskId].config = JSON.parse(JSON.stringify(tab.config))
          }
        }
        
        // 如果标签页存在，同步更新标签页的进度
        if (tab && tab.taskId === taskId) {
          // 标签页数据已经更新，这里只需要确保任务数据是最新的
        }
        
        this.saveTasks()
      }
    },

    // 批量刷新标签页的待处理结果（指纹识别）
    flushPendingResults(tab) {
      if (!tab || !tab.pendingResults || tab.pendingResults.length === 0) {
        return
      }
      
      // 批量添加结果到数组开头，减少Vue响应式更新次数
      const batchSize = Math.min(tab.pendingResults.length, 50) // 每次最多添加50条
      const batch = tab.pendingResults.splice(0, batchSize)
      tab.results.unshift(...batch)
      
      // 更新统计信息
      if (tab.pendingStats) {
        tab.stats.completed += tab.pendingStats.completed
        tab.stats.success += tab.pendingStats.success
        tab.stats.failed += tab.pendingStats.failed
        
        // 重置待处理统计
        tab.pendingStats = {
          completed: 0,
          success: 0,
          failed: 0
        }
      }
      
      // 更新速度统计
      this.updateSpeed(tab)
      
      // 如果还有待处理的结果，继续处理
      if (tab.pendingResults.length > 0) {
        tab.updateResultsTimer = requestAnimationFrame(() => {
          this.flushPendingResults(tab)
          tab.updateResultsTimer = null
        })
      }
    },

    handleScanError(error) {
      if (error?.message) {
        this.$message.warning(error.message)
      }
    },
    handleScanComplete(payload) {
      // 根据tabId找到对应的标签页
      const tabId = payload?.tabId || this.activeTabId
      const tab = this.scanTabs.find(t => t.id === tabId)
      
      // 如果标签页存在，更新标签页状态
      if (tab) {
        // 清理批量更新定时器
        if (tab.updateResultsTimer) {
          cancelAnimationFrame(tab.updateResultsTimer)
          tab.updateResultsTimer = null
        }
        
        // 处理剩余待更新的结果
        if (tab.pendingResults && tab.pendingResults.length > 0) {
          this.flushPendingResults(tab)
        }
        
        tab.scanning = false
        tab.loading = false
        tab.completed = true
        tab.scanProgress = 100
      }
      
      // 无论标签页是否存在，都要更新任务状态（标签可能已关闭，但任务仍在运行）
      // 通过 taskId 或 payload 中的信息找到任务
      let taskId = null
      if (tab && tab.taskId) {
        taskId = tab.taskId
      } else if (payload?.taskId) {
        taskId = payload.taskId
      } else {
        // 如果都没有，尝试通过 tabId 查找任务（任务中可能保存了 tabId）
        const task = Object.values(this.tasksLocal).find(t => t.tabId === tabId)
        if (task) {
          taskId = task.id || task.taskId
        }
      }
      
      if (taskId && this.tasksLocal[taskId]) {
        this.tasksLocal[taskId].status = 'completed'
        this.tasksLocal[taskId].progress = 100
        
        // 如果标签页存在，使用标签页的数据；否则使用 payload 中的数据
        if (tab) {
          this.tasksLocal[taskId].results = JSON.parse(JSON.stringify(tab.results))
          this.tasksLocal[taskId].resultsCount = tab.results.length
          this.tasksLocal[taskId].stats = JSON.parse(JSON.stringify(tab.stats))
          
          // 同步标签页的基本信息到任务（确保任务管理和标签页显示一致）
          if (tab.name) {
            this.tasksLocal[taskId].name = tab.name
          }
          if (tab.urlsText) {
            this.tasksLocal[taskId].urlsText = tab.urlsText
          }
          if (tab.config) {
            this.tasksLocal[taskId].config = JSON.parse(JSON.stringify(tab.config))
          }
        } else if (payload?.results) {
          // 如果标签页不存在，尝试使用 payload 中的数据
          this.tasksLocal[taskId].results = JSON.parse(JSON.stringify(payload.results))
          this.tasksLocal[taskId].resultsCount = payload.results?.length || 0
          if (payload.stats) {
            this.tasksLocal[taskId].stats = JSON.parse(JSON.stringify(payload.stats))
          }
        }
        
        // 如果标签页存在，确保标签页状态与任务状态同步
        if (tab && tab.taskId === taskId) {
          tab.scanning = false
          tab.loading = false
          tab.completed = true
          tab.scanProgress = 100
        }
        
        this.saveTasks()
      }
      
      if (tab) {
        this.saveHistory()
        this.$message.success('识别完成')
      }
    },
    parseHeaders(text) {
      const headers = {}
      if (!text) return headers
      text.split('\n').forEach(line => {
        const match = line.match(/^([^:]+):\s*(.+)$/)
        if (match) {
          headers[match[1].trim()] = match[2].trim()
        }
      })
      return headers
    },
    handleExternalFingerprintFill(event) {
      const payload = event?.detail
      const targets = payload?.targets
      const sourceTaskId = payload?.sourceTaskId // 来源任务ID（用于关联）

      if (!Array.isArray(targets) || targets.length === 0) {
        return
      }

      // 如果提供了 sourceTaskId，使用防抖机制合并短时间内同一 sourceTaskId 的联动请求
      if (sourceTaskId) {
        const debounceKey = `${sourceTaskId}-${payload?.sourceTaskType || 'hostscan'}`
        
        // 收集 targets（合并所有同 sourceTaskId 的请求）
        if (!this.fingerprintFillPendingTargets[debounceKey]) {
          this.fingerprintFillPendingTargets[debounceKey] = {
            payload: { ...payload }, // 深拷贝 payload，避免被后续请求覆盖
            targets: []
          }
        }
        // 合并 targets（去重）
        const existingTargets = this.fingerprintFillPendingTargets[debounceKey].targets
        targets.forEach(target => {
          if (!existingTargets.includes(target)) {
            existingTargets.push(target)
          }
        })
        // 更新 payload 中的 targets（使用合并后的 targets）
        this.fingerprintFillPendingTargets[debounceKey].payload.targets = [...this.fingerprintFillPendingTargets[debounceKey].targets]
        
        // 清除之前的防抖定时器
        if (this.fingerprintFillDebounceMap[debounceKey]) {
          clearTimeout(this.fingerprintFillDebounceMap[debounceKey])
        }
        
        // 设置新的防抖定时器，延迟处理以避免重复创建标签页
        this.fingerprintFillDebounceMap[debounceKey] = setTimeout(() => {
          const pending = this.fingerprintFillPendingTargets[debounceKey]
          if (pending && pending.targets.length > 0) {
            // 使用合并后的 payload 和 targets
            this._processFingerprintFill(pending.payload, pending.targets, sourceTaskId)
          }
          delete this.fingerprintFillDebounceMap[debounceKey]
          delete this.fingerprintFillPendingTargets[debounceKey]
        }, 200) // 200ms 防抖，合并短时间内同一 sourceTaskId 的请求
        
        return
      }
      
      // 如果没有 sourceTaskId，直接处理（通常是手动触发）
      this._processFingerprintFill(payload, targets, sourceTaskId)
    },
    
    _processFingerprintFill(payload, targets, sourceTaskId) {
      // 如果组件已挂载且已激活，直接处理
      if (window.__fingerprintReady && window.__fingerprintActivated) {
        this.__doProcessFingerprintFill(payload, targets, sourceTaskId)
        return
      }

      // 如果组件已挂载但未激活，将事件暂存到队列，等组件激活后再处理
      if (window.__fingerprintReady && !window.__fingerprintActivated) {
        if (!Array.isArray(window.__cyberSabrePendingFingerprint)) {
          window.__cyberSabrePendingFingerprint = []
        }
        window.__cyberSabrePendingFingerprint.push(payload)
        return
      }

      // 如果组件未挂载，事件会被代理暂存，这里不需要处理
      // 组件挂载后会在 mounted() 中处理暂存的事件
    },

    __doProcessFingerprintFill(payload, targets, sourceTaskId) {
      // 如果提供了 sourceTaskId，查找是否已存在相同来源任务的标签页
      let targetTab = null
      if (sourceTaskId) {
        targetTab = this.scanTabs.find(tab => 
          tab.sourceTaskId === sourceTaskId && 
          tab.sourceTaskType === (payload?.sourceTaskType || 'hostscan')
        )
        // 如果找到了标签页，确保sourceTaskId和sourceTaskType已设置（防止后续查找失败）
        if (targetTab && !targetTab.sourceTaskId) {
          targetTab.sourceTaskId = sourceTaskId
          targetTab.sourceTaskType = payload?.sourceTaskType || 'hostscan'
        }
      }

      if (targetTab) {
        // 如果已存在相同来源任务的标签页，追加URL而不是创建新标签页
        const existingUrls = targetTab.urlsText.split('\n').filter(url => url.trim())
        const newUrls = targets.filter(url => !existingUrls.includes(url.trim()))
        if (newUrls.length > 0) {
          targetTab.urlsText = [...existingUrls, ...newUrls].join('\n')
        }
        // 切换到现有标签页
        this.activeTabId = targetTab.id
        // 应用配置（如果提供了）
        if (payload?.config) {
          const cfg = payload.config
          if (cfg.timeout !== undefined) targetTab.config.timeout = cfg.timeout
          if (cfg.threads !== undefined) targetTab.config.threads = cfg.threads
          if (cfg.dirThreads !== undefined) targetTab.config.dirThreads = cfg.dirThreads
          if (cfg.maxDirChecks !== undefined) targetTab.config.maxDirChecks = cfg.maxDirChecks
          if (cfg.dirMode !== undefined) targetTab.config.dirMode = cfg.dirMode
          if (cfg.useWappalyzer !== undefined) targetTab.config.useWappalyzer = cfg.useWappalyzer
          if (cfg.useTideRules !== undefined) targetTab.config.useTideRules = cfg.useTideRules
          if (cfg.useCmsDict !== undefined) targetTab.config.useCmsDict = cfg.useCmsDict
          if (cfg.usePoweredBy !== undefined) targetTab.config.usePoweredBy = cfg.usePoweredBy
          if (cfg.followRedirect !== undefined) targetTab.config.followRedirect = cfg.followRedirect
          if (cfg.headers !== undefined) {
            targetTab.config.customHeaders = this.headersToText(cfg.headers)
          }
        }
        // 如果标签页未在扫描中，自动开始扫描（联动时跳过确认，保留数据）
        // 注意：如果标签页正在扫描中，新的 URL 会被追加，但需要等待当前扫描完成后才能扫描新的 URL
        // 如果需要立即扫描新的 URL，可以手动停止并重新启动扫描
        if (!targetTab.scanning) {
          this.$nextTick(() => {
            this.startScan(targetTab, true) // 传入true表示联动扫描，跳过确认并保留数据
          })
        }
      } else {
        // 联动时自动创建新标签页
        const newTab = this.createNewTab()
        newTab.name = `指纹识别-联动-${new Date().toLocaleTimeString()}`
        newTab.urlsText = targets.join('\n')
        
        // 记录来源任务ID（用于任务管理中的关联显示）
        if (sourceTaskId) {
          newTab.sourceTaskId = sourceTaskId
          newTab.sourceTaskType = payload?.sourceTaskType || 'hostscan' // 来源任务类型：hostscan, dirsearch, webscan等
        }

        // 应用配置（如果提供了）
        if (payload?.config) {
          const cfg = payload.config
          if (cfg.timeout !== undefined) newTab.config.timeout = cfg.timeout
          if (cfg.threads !== undefined) newTab.config.threads = cfg.threads
          if (cfg.dirThreads !== undefined) newTab.config.dirThreads = cfg.dirThreads
          if (cfg.maxDirChecks !== undefined) newTab.config.maxDirChecks = cfg.maxDirChecks
          if (cfg.dirMode !== undefined) newTab.config.dirMode = cfg.dirMode
          if (cfg.useWappalyzer !== undefined) newTab.config.useWappalyzer = cfg.useWappalyzer
          if (cfg.useTideRules !== undefined) newTab.config.useTideRules = cfg.useTideRules
          if (cfg.useCmsDict !== undefined) newTab.config.useCmsDict = cfg.useCmsDict
          if (cfg.usePoweredBy !== undefined) newTab.config.usePoweredBy = cfg.usePoweredBy
          if (cfg.followRedirect !== undefined) newTab.config.followRedirect = cfg.followRedirect
          if (cfg.headers !== undefined) {
            newTab.config.customHeaders = this.headersToText(cfg.headers)
          }
        }

        // 自动联动时，直接启动扫描，不触发覆盖提示（保留数据）
        this.$nextTick(() => {
          this.startScan(newTab, true) // 传入true表示联动扫描，跳过确认并保留数据
        })
      }
    },
    async startScanInternal(skipConfirm = false, shouldClearResults = false, shouldKeepResults = false) {
      const urls = this.urlsText
        .split('\n')
        .map(url => url.trim())
        .filter(url => url.length > 0)

      if (urls.length === 0) {
        this.$message.warning('请输入至少一个URL')
        return
      }

      // 如果是自动联动（skipConfirm=true），默认保存记录并继续扫描
      if (skipConfirm && this.results.length > 0) {
        shouldKeepResults = true
        shouldClearResults = false
      }

      try {
        this.scanning = true
        this.loading = true
        this.scanProgress = 0
        this.scanStartTime = Date.now()
        
        if (shouldClearResults) {
          this.results = []
          this.stats = {
            totalTargets: 0,
            completed: 0,
            success: 0,
            failed: 0,
            targetsPerSecond: 0
          }
        } else if (shouldKeepResults) {
          // 保存记录继续扫描：保留现有结果
          // 统计信息会在扫描过程中更新
        } else {
          // 首次扫描：清空结果和统计
          this.results = []
          this.stats = {
            totalTargets: urls.length,
            completed: 0,
            success: 0,
            failed: 0,
            targetsPerSecond: 0
          }
        }

        const scanConfig = {
          targets: urls,
          timeout: this.config.timeout,
          threads: this.config.threads,
          dirThreads: this.config.dirThreads,
          maxDirChecks: this.config.maxDirChecks,
          dirMode: this.config.dirMode,
          useWappalyzer: this.config.useWappalyzer,
          useTideRules: this.config.useTideRules,
          useCmsDict: this.config.useCmsDict,
          usePoweredBy: this.config.usePoweredBy,
          followRedirect: this.config.followRedirect,
          headers: this.parseHeaders(this.config.customHeaders)
        }

        await StartFingerprintScan(JSON.stringify(scanConfig))
        this.loading = false
        this.$message.success('指纹识别已启动')
      } catch (error) {
        const msg = error?.message || error?.toString() || '未知错误'
        this.$message.error(`启动失败: ${msg}`)
        this.scanning = false
        this.loading = false
      }
    },
    headersToText(headers) {
      if (!headers) return ''
      if (typeof headers === 'string') return headers
      if (typeof headers === 'object') {
        return Object.entries(headers)
          .map(([key, value]) => `${key}: ${value}`)
          .join('\n')
      }
      return ''
    },
    async clearResults(tab) {
      const currentTab = tab || this.currentTab
      if (!currentTab) return
      try {
        await this.$confirm('确定要清空当前识别结果吗？', '清空结果', { type: 'warning' })
      } catch {
        return
      }
      currentTab.results = []
      this.resetColumnFilters(currentTab)
      currentTab.currentPage = 1
      currentTab.stats.completed = 0
      currentTab.stats.success = 0
      currentTab.stats.failed = 0
      this.saveHistory()
      this.$message.success('已清空')
    },
    showDetail(row) {
      this.selectedResult = row
      this.detailDialogVisible = true
    },
    getStatusTagType(status) {
      if (status >= 200 && status < 300) return 'success'
      if (status >= 300 && status < 400) return 'warning'
      if (status >= 400 && status < 500) return 'danger'
      if (status >= 500) return 'danger'
      return 'info'
    },
    getRowClassName({ row }) {
      if (row.status >= 200 && row.status < 300) {
        return 'row-success'
      }
      if (row.status >= 300 && row.status < 400) {
        return 'row-warning'
      }
      if (row.status >= 400) {
        return 'row-error'
      }
      return ''
    },
    formatHeaders(headers) {
      if (!headers) return ''
      return Object.entries(headers)
        .map(([key, value]) => `${key}: ${value}`)
        .join('\n')
    },
    // 获取所有技术栈信息（包括technologies、cms、poweredBy、tideMatches等）
    getAllTechnologies(row) {
      if (!row) return []
      const allTechs = []
      
      // 添加technologies
      if (row.technologies && row.technologies.length > 0) {
        allTechs.push(...row.technologies)
      }
      
      // 添加CMS
      if (row.cms) {
        allTechs.push(`CMS: ${row.cms}`)
      }
      
      // 添加Powered By
      if (row.poweredBy && row.poweredBy.length > 0) {
        row.poweredBy.forEach(pb => {
          allTechs.push(`Powered By: ${pb}`)
        })
      }
      
      // 添加Tide匹配
      if (row.tideMatches && row.tideMatches.length > 0) {
        row.tideMatches.forEach(tm => {
          allTechs.push(`Tide: ${tm}`)
        })
      }
      
      // 智能去重：处理大小写不同和带前缀的重复
      return this.deduplicateTechnologies(allTechs)
    },
    // 去重技术栈：处理大小写不同和带前缀的重复
    deduplicateTechnologies(techs) {
      if (!techs || techs.length === 0) return []
      
      // 提取技术栈实际名称（去掉前缀，转为小写用于比较）
      const extractTechName = (tech) => {
        let name = tech.trim()
        if (name.includes(':')) {
          const parts = name.split(':')
          if (parts.length > 1) {
            name = parts.slice(1).join(':').trim()
          }
        }
        return name.toLowerCase()
      }
      
      // 用于存储最终结果：key为小写技术名称，value为最终要显示的原始字符串
      const techMap = new Map()
      // 记录哪些技术栈已经有不带前缀的版本
      const noPrefixSet = new Set()
      
      // 第一遍遍历：先处理所有不带前缀的技术栈，建立基础映射
      for (const tech of techs) {
        if (!tech || tech.trim() === '') continue
        const normalizedTech = tech.trim()
        if (!normalizedTech.includes(':')) {
          const techName = extractTechName(normalizedTech)
          if (!techMap.has(techName)) {
            techMap.set(techName, normalizedTech)
            noPrefixSet.add(techName)
          }
        }
      }
      
      // 第二遍遍历：处理带前缀的技术栈，仅当没有不带前缀的版本时才添加
      for (const tech of techs) {
        if (!tech || tech.trim() === '') continue
        const normalizedTech = tech.trim()
        if (normalizedTech.includes(':')) {
          const techName = extractTechName(normalizedTech)
          // 只有当这个技术栈没有不带前缀的版本时，才添加带前缀的版本
          // 并且只保留第一个遇到的带前缀版本
          if (!noPrefixSet.has(techName) && !techMap.has(techName)) {
            techMap.set(techName, normalizedTech)
          }
        }
      }
      
      // 第三遍：按原始顺序构建结果数组，只保留每个技术栈的一个版本
      const result = []
      const added = new Set() // 记录已经添加的技术栈（基于小写名称）
      
      for (const tech of techs) {
        if (!tech || tech.trim() === '') continue
        const normalizedTech = tech.trim()
        const techName = extractTechName(normalizedTech)
        
        // 如果这个技术栈还没添加过，且它在techMap中（说明是最终要保留的版本）
        if (!added.has(techName) && techMap.has(techName)) {
          // 获取最终要显示的版本并添加
          const finalVersion = techMap.get(techName)
          result.push(finalVersion)
          added.add(techName)
        }
      }
      
      return result
    },
    // 根据技术栈类型返回不同的tag类型
    getTechTagType(tech, row) {
      if (tech.startsWith('CMS:')) return 'warning'
      if (tech.startsWith('Powered By:')) return 'info'
      if (tech.startsWith('Tide:')) return 'success'
      return 'plain'
    },
    // 格式化长度显示
    formatLength(length) {
      if (!length && length !== 0) return '-'
      if (length < 1024) {
        return `${length} B`
      } else if (length < 1024 * 1024) {
        return `${(length / 1024).toFixed(2)} KB`
      } else {
        return `${(length / (1024 * 1024)).toFixed(2)} MB`
      }
    },
    exportToExcel(tab) {
      const currentTab = tab || this.currentTab
      if (!currentTab) return
      if (currentTab.results.length === 0) {
        this.$message.warning('没有可导出的数据')
        return
      }
      const data = currentTab.results.map((item, index) => ({
        序号: index + 1,
        URL: item.url,
        状态码: item.status,
        标题: item.title || '-',
        长度: item.length ? this.formatLength(item.length) : '-',
        技术栈: this.getAllTechnologies(item).join(', '),
        响应时间: item.responseTime
      }))
      const wb = XLSX.utils.book_new()
      const ws = XLSX.utils.json_to_sheet(data)
      ws['!cols'] = [
        { wch: 6 },
        { wch: 50 },
        { wch: 10 },
        { wch: 30 },
        { wch: 12 },
        { wch: 60 },
        { wch: 12 }
      ]
      XLSX.utils.book_append_sheet(wb, ws, '指纹识别')
      XLSX.writeFile(wb, `指纹识别_${Date.now()}.xlsx`)
      this.$message.success('导出成功')
    },
    startTimer(tab) {
      const currentTab = tab || this.currentTab
      if (!currentTab) return
      this.updateElapsedTime(currentTab)
      currentTab.timer = setInterval(() => {
        this.updateElapsedTime(currentTab)
        this.updateSpeed(currentTab)
      }, 1000)
    },
    stopTimer(tab) {
      const currentTab = tab || this.currentTab
      if (!currentTab) return
      if (currentTab.timer) {
        clearInterval(currentTab.timer)
        currentTab.timer = null
      }
    },
    updateElapsedTime(tab) {
      const currentTab = tab || this.currentTab
      if (!currentTab || !currentTab.scanStartTime) return
      const elapsed = Date.now() - currentTab.scanStartTime
      const minutes = Math.floor(elapsed / 60000)
      const seconds = Math.floor((elapsed % 60000) / 1000)
      currentTab.elapsedTime = `${minutes.toString().padStart(2, '0')}:${seconds
        .toString()
        .padStart(2, '0')}`
    },
    updateSpeed(tab) {
      const currentTab = tab || this.currentTab
      if (!currentTab || !currentTab.scanStartTime) return
      const elapsedSeconds = (Date.now() - currentTab.scanStartTime) / 1000
      if (elapsedSeconds > 0) {
        currentTab.stats.targetsPerSecond = currentTab.stats.completed / elapsedSeconds
      }
    },
    handleRowContextMenu(row, column, event, tab) {
      event.preventDefault()
      event.stopPropagation()
      this.selectedRow = row
      this.selectedTab = tab || this.currentTab
      
      let x = event.clientX || (event.originalEvent && event.originalEvent.clientX) || 0
      let y = event.clientY || (event.originalEvent && event.originalEvent.clientY) || 0
      
      this.contextMenuPos = { x, y }
      this.contextMenuVisible = true
      
      this.$nextTick(() => {
        const menuEl = document.querySelector('.context-menu')
        if (!menuEl) return
        
        const menuHeight = menuEl.offsetHeight || 0
        const windowHeight = window.innerHeight
        
        if (y + menuHeight > windowHeight - 8) {
          let adjustedY = windowHeight - menuHeight - 8
          if (adjustedY < 8) adjustedY = 8
          this.contextMenuPos = { x, y: adjustedY }
        }
      })
      
      const closeMenu = (e) => {
        if (!e.target.closest('.context-menu')) {
          this.contextMenuVisible = false
          document.removeEventListener('click', closeMenu)
          document.removeEventListener('contextmenu', closeMenu)
        }
      }
      setTimeout(() => {
        document.addEventListener('click', closeMenu)
        document.addEventListener('contextmenu', closeMenu)
      }, 100)
    },
    handleViewDetail() {
      if (!this.selectedRow) return
      this.contextMenuVisible = false
      this.showDetail(this.selectedRow)
    },
    handleViewPacket() {
      if (!this.selectedRow) return
      if (!this.selectedRow.requestPacket && !this.selectedRow.responsePacket) {
        this.$message.warning('该记录没有数据包信息')
        return
      }
      this.contextMenuVisible = false
      this.packetViewerVisible = true
    },
    handleCopyRow() {
      if (!this.selectedRow) return
      const rowText = `URL: ${this.selectedRow.url}\n状态码: ${this.selectedRow.status}\n标题: ${this.selectedRow.title || ''}\n长度: ${this.formatLength(this.selectedRow.length)}\n技术栈: ${this.getAllTechnologies(this.selectedRow).join(', ')}\n响应时间: ${this.selectedRow.responseTime}ms`
      navigator.clipboard.writeText(rowText).then(() => {
        this.$message.success('已复制到剪贴板')
      }).catch(() => {
        this.$message.error('复制失败')
      })
      this.contextMenuVisible = false
    },
    handleCopyUrl() {
      if (!this.selectedRow) return
      navigator.clipboard.writeText(this.selectedRow.url).then(() => {
        this.$message.success('已复制URL到剪贴板')
      }).catch(() => {
        this.$message.error('复制失败')
      })
      this.contextMenuVisible = false
    },
    handleOpenUrl() {
      if (!this.selectedRow || !this.selectedRow.url) return
      BrowserOpenURL(this.selectedRow.url)
      this.contextMenuVisible = false
    },
    async handleDeleteRow() {
      if (!this.selectedRow) return
      try {
        await this.$confirm('确定要删除这条记录吗？', '确认删除', { type: 'warning' })
      } catch {
        this.contextMenuVisible = false
        return
      }
      const index = this.results.findIndex(r => r.url === this.selectedRow.url && r.status === this.selectedRow.status)
      if (index !== -1) {
        this.results.splice(index, 1)
        this.$message.success('已删除')
        if (this.selectedRow.status >= 200 && this.selectedRow.status < 300) {
          this.stats.success = Math.max(0, this.stats.success - 1)
        } else {
          this.stats.failed = Math.max(0, this.stats.failed - 1)
        }
        this.stats.completed = Math.max(0, this.stats.completed - 1)
      }
      this.contextMenuVisible = false
    },
    handleFingerprintRefresh() {
      // 指纹更新后的回调，可以在这里触发重新扫描或其他操作
      this.$message.success('指纹规则已更新')
    }
  },
  components: {
    PacketViewer,
    FingerprintManagePanel,
    TableColumnFilterHeader
  }
}
</script>

<style lang="scss" scoped>
.fingerprint-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: transparent;
  padding: 6px;
  overflow: hidden;
  box-sizing: border-box;
}

// 顶部工具栏
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

.config-section {
  flex-shrink: 0;
}

.config-card {
  :deep(.el-card__body) {
    padding: 12px;
  }
}

.config-layout {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.url-input-section {
  width: 260px;
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

.quick-stats {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 4px;
  padding: 4px 6px;
  background: #f0f2f5;
  border-radius: 4px;
  font-size: 12px;
  color: #606266;
  gap: 6px;
}

.quick-stat-item {
  display: flex;
  align-items: center;
  gap: 4px;

  .label {
    color: #909399;
  }

  .value {
    font-weight: 600;
    color: #303133;
  }

  &.success .value {
    color: #67c23a;
  }

  &.danger .value {
    color: #f56c6c;
  }
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
  gap: 8px;
}

.config-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 8px;

  .config-item {
    display: flex;
    flex-direction: column;
    gap: 4px;

    &.full-width {
      grid-column: 1 / -1;
    }
  }
}

.switch-row {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 12px;
  
  .config-item {
    flex: 1;
    min-width: 0;
    white-space: nowrap;
    
    :deep(.el-switch) {
      width: 100%;
      
      .el-switch__label {
        font-size: 12px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }
      
      &.is-disabled {
        opacity: 0.5;
        
        .el-switch__core {
          background-color: #c0c4cc;
          border-color: #c0c4cc;
        }
        
        .el-switch__label {
          color: #c0c4cc;
        }
      }
    }
  }
}

.config-label {
  font-size: 12px;
  color: #606266;
  font-weight: 500;
}

.progress-section {
  margin-top: 12px;
  width: 100%;
  position: relative;
  z-index: 1;
}

.progress-card {
  width: 100%;
  border: 1px solid #e4e7ed;
  background: #fafafa;
  
  :deep(.el-card__body) {
    padding: 14px 20px;
  }
}

.progress-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
  font-size: 12px;
  color: #606266;
}

/* 进度条过渡动画 */
.progress-fade-enter-active,
.progress-fade-leave-active {
  transition: all 0.3s ease;
}

.progress-fade-enter-from {
  opacity: 0;
  transform: translateY(-10px);
}

.progress-fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

.results-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
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

.results-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  :deep(.el-card__body) {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 0;
    overflow: hidden;
  }
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid #e4e7ed;
  flex-wrap: wrap;
  gap: 10px;
}

.results-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.table-wrapper {
  flex: 1;
  padding: 12px;
  overflow: hidden;
}

.fingerprint-table {
  width: 100%;
  height: 100%;

  :deep(.el-table__header-wrapper) {
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
  }

  :deep(.row-success) {
    background-color: #f0f9eb;
  }

  :deep(.row-warning) {
    background-color: #fdf6ec;
  }

  :deep(.row-error) {
    background-color: #fef0f0;
  }
}

.status-tag {
  min-width: 20px;
  text-align: center;
}

.url-cell {
  display: flex;
  align-items: center;
}

.url-text {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.tech-tag,
.pb-tag {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-text {
  color: #c0c4cc;
  font-style: italic;
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

.detail-wrapper {
  max-height: 70vh;
  overflow: auto;

  pre {
    background: #f5f7fa;
    padding: 12px;
    border-radius: 4px;
    font-size: 12px;
    line-height: 1.6;
  }
}

.detail-block {
  margin-top: 18px;

  h4 {
    margin-bottom: 8px;
  }
}

@media (max-width: 1200px) {
  .config-layout {
    flex-direction: column;
  }
  .url-input-section {
    width: 100%;
  }
}

.context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  z-index: 3000;
  min-width: 160px;
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

  &:hover {
    background-color: #f5f7fa;
  }

  &.danger {
    color: #f56c6c;

    &:hover {
      background-color: #fef0f0;
    }
  }

  .el-icon {
    font-size: 16px;
  }
}

.context-menu-divider {
  height: 1px;
  background: #e4e7ed;
  margin: 4px 0;
}

.advanced-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.advanced-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 6px;
}

.section-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  padding-bottom: 8px;
  border-bottom: 1px solid #e4e7ed;
}

.advanced-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
  margin-top: 8px;

  .advanced-item {
    display: flex;
    flex-direction: column;
    gap: 6px;

    &.full-width {
      grid-column: 1 / -1;
    }
  }
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
    flex-direction: column;
    padding: 0px;
  }
  
  :deep(.el-tab-pane) {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
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
    
    &.stopped {
      color: #909399;
    }
  }
  
  .tab-name {
    max-width: 150px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  
  .tab-badge {
    margin-left: 4px;
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

// 高级配置对话框
.advanced-config-dialog {
  .advanced-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
    background: #f8f9fa;
    border-radius: 6px;
    margin-bottom: 16px;
    
    &:last-child {
      margin-bottom: 0;
    }
  }
  
  .section-title {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: #303133;
    padding-bottom: 8px;
    border-bottom: 1px solid #e4e7ed;
  }
  
  .advanced-row {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 12px;
    margin-top: 8px;
    
    .advanced-item {
      display: flex;
      flex-direction: column;
      gap: 6px;
      
      &.full-width {
        grid-column: 1 / -1;
      }
    }
  }
}

// 历史记录对话框
:deep(.history-dialog) {
  .el-overlay-dialog {
    display: flex;
    align-items: flex-start;
    justify-content: center;
  }

  .el-dialog {
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

  .el-dialog__header {
    flex-shrink: 0;
    padding: 20px 20px 10px;
  }

  .el-dialog__body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    overflow-x: hidden;
    padding: 4px;
    box-sizing: border-box;
  }

  .el-dialog__footer {
    flex-shrink: 0;
    padding: 10px 20px 20px;
  }
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

.targets-text-truncated {
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-name-cell,
.table-cell-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  width: 100%;
}

.mb-10px {
  margin-bottom: 10px;
}

/* 指纹管理对话框样式 - 移除外层滚动条 */
:deep(.fingerprint-manage-dialog .el-dialog) {
  height: 85vh;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

:deep(.fingerprint-manage-dialog .el-dialog__header) {
  flex-shrink: 0;
  padding: 20px 20px 10px;
  border-bottom: 1px solid #e4e7ed;
}

:deep(.fingerprint-manage-dialog .el-dialog__body) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 0;
  display: flex;
  flex-direction: column;
}

:deep(.fingerprint-manage-dialog .fingerprint-manage-panel) {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  box-sizing: border-box;
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
</style>

