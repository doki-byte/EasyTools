<template>
  <div class="dirsearch-container">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar">
      <div class="toolbar-content">
        <div class="toolbar-left">
          <div class="module-title">
            <el-icon class="title-icon">
              <FolderOpened />
            </el-icon>
            <span class="title-text">目录扫描</span>
          </div>
        </div>
        <div class="toolbar-right">
          <el-button :icon="Plus" type="primary" plain size="default" @click="createNewTab">
            新建任务
          </el-button>
          <el-button :icon="Clock" type="info" plain size="default" @click="historyDialog = true">
            任务管理
          </el-button>
        </div>
      </div>
    </div>

    <!-- 标签页 -->
    <CustomTabs>
      <el-tabs v-model="activeTabId" type="border-card" class="panel-card glass-card main-tabs" closable
        @tab-remove="removeTab" @tab-click="handleTabClick" @contextmenu.prevent="handleTabsContextMenu">
        <el-tab-pane v-for="tab in scanTabs" :key="tab.id" :name="tab.id">
          <template #label>
            <span class="tab-label">
              <el-icon v-if="tab.scanning" class="tab-status-icon running">
                <VideoPlay />
              </el-icon>
              <el-icon v-else-if="tab.completed" class="tab-status-icon completed">
                <CircleCheck />
              </el-icon>
              <span class="tab-name">{{ tab.name }}</span>
              <el-badge v-if="tab.results.length > 0" :value="tab.results.length" :max="99" class="tab-badge" />
            </span>
          </template>

          <!-- 配置区域：左侧URL输入，右侧配置 -->
          <div class="config-section">
            <el-card shadow="never" class="config-card">
              <div class="config-layout">
                <!-- 左侧：URL输入框 -->
                <div class="url-input-section">
                  <div class="url-label">目标URL</div>
                  <IsolatedTextarea
                    :ref="(el) => setIsolatedRef('urls-' + tab.id, el)"
                    v-model="tab.urlsText"
                    :rows="7"
                    placeholder="每行一个URL，支持批量扫描&#10;例如：&#10;https://example1.com&#10;https://example2.com"
                    size="default"
                    class="url-textarea"
                  />
                  <div class="url-actions">
                    <el-button :type="tab.scanning ? 'danger' : 'primary'" :icon="tab.scanning ? VideoPause : VideoPlay"
                      @click="toggleScan(tab)" :loading="tab.loading && !tab.scanning" size="default"
                      style="width: 100%">
                      {{ tab.scanning ? '停止扫描' : '开始扫描' }}
                    </el-button>
                  </div>
                </div>

                <!-- 右侧：配置项 -->
                <div class="config-section-right">
                  <div class="config-group">
                    <div class="config-row">
                      <div class="config-item">
                        <label class="config-label">请求方法</label>
                        <el-select v-model="tab.config.method" size="default" style="width: 100%">
                          <el-option label="GET" value="GET" />
                          <el-option label="POST" value="POST" />
                          <el-option label="PUT" value="PUT" />
                          <el-option label="HEAD" value="HEAD" />
                          <el-option label="OPTIONS" value="OPTIONS" />
                        </el-select>
                      </div>

                      <div class="config-item">
                        <label class="config-label">线程数</label>
                        <el-input-number v-model="tab.config.threads" :min="1" :max="100" controls-position="right"
                          size="default" style="width: 100%" />
                      </div>

                      <div class="config-item">
                        <label class="config-label">超时(秒)</label>
                        <el-input-number v-model="tab.config.timeout" :min="1" :max="30" controls-position="right"
                          size="default" style="width: 100%" />
                      </div>

                      <div class="config-item">
                        <label class="config-label">递归深度</label>
                        <el-input-number v-model="tab.config.recursionDepth" :min="0" :max="5" controls-position="right"
                          size="default" style="width: 100%" />
                      </div>
                    </div>

                    <div class="config-row">
                      <div class="config-item full-width">
                        <label class="config-label">字典文件</label>
                        <el-select v-model="tab.selectedDicts" multiple collapse-tags collapse-tags-tooltip
                          placeholder="选择字典文件" size="default" style="width: 100%" @visible-change="loadDictList">
                          <el-option v-for="dict in dicts" :key="dict.name"
                            :label="`${dict.name} (${formatSize(dict.size)})`" :value="dict.name" />
                        </el-select>
                      </div>
                    </div>

                    <div class="config-row">
                      <div class="config-item full-width">
                        <label class="config-label">自定义字典</label>
                        <IsolatedTextarea
                          :ref="(el) => setIsolatedRef('customDict-' + tab.id, el)"
                          v-model="tab.config.customDict"
                          :rows="2"
                          placeholder="每行一个路径，支持自定义字典"
                          size="default"
                        />
                      </div>
                    </div>

                    <div class="config-row">
                      <div class="config-item">
                        <el-switch v-model="tab.config.followRedirects" active-text="跟随重定向" size="default" />
                      </div>
                      <div class="config-item">
                        <el-button :icon="Setting" @click="showAdvancedDialog = true; currentAdvancedTab = tab"
                          size="default" plain>
                          高级配置
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
                  <el-progress :percentage="Math.round(tab.stats?.progressPercent || tab.scanProgress || 0)"
                    :status="(tab.stats?.progressPercent || tab.scanProgress || 0) >= 100 ? 'success' : undefined"
                    :stroke-width="8" :show-text="true" :format="(percentage) => `${percentage}%`" />
                  <div class="progress-stats">
                    <el-tag size="small" type="info">
                      <el-icon>
                        <Clock />
                      </el-icon>
                      已用时: {{ formatDuration(tab.stats?.elapsedSeconds || 0) }}
                    </el-tag>
                    <el-tag v-if="tab.stats?.estimatedSeconds > 0" size="small" type="warning">
                      <el-icon>
                        <Timer />
                      </el-icon>
                      预计剩余: {{ formatDuration(tab.stats?.estimatedSeconds || 0) }}
                    </el-tag>
                    <el-tag size="small" type="success">
                      <el-icon>
                        <Connection />
                      </el-icon>
                      速度: {{ formatNumber(tab.stats?.requestsPerSecond || 0) }} 请求/秒
                    </el-tag>
                    <el-tag size="small">
                      已请求: {{ formatNumber(tab.stats?.totalRequests || 0) }} / {{ formatNumber(tab.stats?.plannedTotal
                        || 0) }}
                    </el-tag>
                    <el-tag size="small" type="success">
                      成功: {{ tab.stats?.success || 0 }}
                    </el-tag>
                    <el-tag size="small" type="danger">
                      失败: {{ tab.stats?.failed || 0 }}
                    </el-tag>
                    <el-tag size="small" type="warning">
                      发现: {{ tab.stats?.found || 0 }}
                    </el-tag>
                  </div>
                </el-card>
              </div>
            </transition>
          </div>

          <!-- 扫描结果表格 -->
          <div class="results-section" :data-dirsearch-tab="tab.id">
            <el-card shadow="never" class="results-card">
              <template #header>
                <div class="results-header">
                  <div class="results-title-group">
                    <span class="results-title">扫描结果</span>
                    <el-tag type="info" size="small">{{ getFilteredResults(tab).length }} 个</el-tag>
                  </div>
                  <div class="results-actions">
                    <el-button :icon="Download" :disabled="tab.results.length === 0" @click="exportToExcel(tab)"
                      size="default">
                      导出Excel
                    </el-button>
                    <el-button :icon="Delete" :disabled="tab.results.length === 0" @click="clearResults(tab)"
                      size="default" plain>
                      清空
                    </el-button>
                  </div>
                </div>
              </template>

              <div class="table-wrapper">
                <el-table :key="'dirsearch-results-' + tab.id" :data="getSortedResults(tab)" :height="tableHeight"
                  stripe border size="default" @row-click="handleRowClick"
                  @row-contextmenu="(row, column, event) => handleRowContextMenu(row, column, event, tab)"
                  class="results-table" :row-class-name="getRowClassName">
                  <el-table-column type="index" width="60" align="center" label="#" />
                  <el-table-column prop="status" width="138" align="center">
                    <template #header>
                      <TableColumnFilterHeader label="状态码" v-model="tab.columnFilters.status" :popover-width="240"
                        sortable :sort-asc-active="tab.sortField === 'status' && tab.sortOrder === 'asc'"
                        :sort-desc-active="tab.sortField === 'status' && tab.sortOrder === 'desc'"
                        @sort-asc="setTabSort(tab, 'status', 'asc')" @sort-desc="setTabSort(tab, 'status', 'desc')"
                        @filter-change="handleColumnFilterInput(tab)" />
                    </template>
                    <template #default="{ row }">
                      <el-tag :type="getStatusTagType(row.status)" size="default" effect="dark" class="status-tag">
                        {{ row.status }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="url" min-width="300">
                    <template #header>
                      <TableColumnFilterHeader label="URL" v-model="tab.columnFilters.url" :popover-width="280" sortable
                        :sort-asc-active="tab.sortField === 'url' && tab.sortOrder === 'asc'"
                        :sort-desc-active="tab.sortField === 'url' && tab.sortOrder === 'desc'"
                        @sort-asc="setTabSort(tab, 'url', 'asc')" @sort-desc="setTabSort(tab, 'url', 'desc')"
                        @filter-change="handleColumnFilterInput(tab)" />
                    </template>
                    <template #default="{ row }">
                      <div class="url-cell">
                        <span class="url-text" :title="row.url">{{ formatUrl(row.url) }}</span>
                        <div class="url-actions">
                          <el-button v-if="row.status === 200" type="text" size="small" :icon="Link"
                            @click.stop="openUrl(row.url)" title="打开链接" />
                          <el-button type="text" size="small" :icon="CopyDocument"
                            @click.stop="copyToClipboard(row.url)" title="复制链接" />
                        </div>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column prop="title" width="230">
                    <template #header>
                      <TableColumnFilterHeader label="标题" v-model="tab.columnFilters.title" :popover-width="260"
                        sortable :sort-asc-active="tab.sortField === 'title' && tab.sortOrder === 'asc'"
                        :sort-desc-active="tab.sortField === 'title' && tab.sortOrder === 'desc'"
                        @sort-asc="setTabSort(tab, 'title', 'asc')" @sort-desc="setTabSort(tab, 'title', 'desc')"
                        @filter-change="handleColumnFilterInput(tab)" />
                    </template>
                    <template #default="{ row }">
                      <el-tooltip :content="row.title || ''" placement="top"
                        :disabled="!row.title || row.title.length <= 25" effect="dark">
                        <div class="table-cell-text">
                          <span v-if="row.title">{{ row.title }}</span>
                          <span v-else class="empty-text">-</span>
                        </div>
                      </el-tooltip>
                    </template>
                  </el-table-column>
                  <el-table-column prop="size" width="148" align="right">
                    <template #header>
                      <TableColumnFilterHeader label="文件大小" v-model="tab.columnFilters.size" :popover-width="240"
                        placement="bottom-end" align-right sortable
                        :sort-asc-active="tab.sortField === 'size' && tab.sortOrder === 'asc'"
                        :sort-desc-active="tab.sortField === 'size' && tab.sortOrder === 'desc'"
                        @sort-asc="setTabSort(tab, 'size', 'asc')" @sort-desc="setTabSort(tab, 'size', 'desc')"
                        @filter-change="handleColumnFilterInput(tab)" />
                    </template>
                    <template #default="{ row }">
                      {{ formatSize(row.size) }}
                    </template>
                  </el-table-column>
                  <el-table-column prop="responsePacketSize" width="160" align="right">
                    <template #header>
                      <TableColumnFilterHeader label="响应包大小" v-model="tab.columnFilters.responsePacketSize"
                        :popover-width="240" placement="bottom-end" align-right sortable
                        :sort-asc-active="tab.sortField === 'responsePacketSize' && tab.sortOrder === 'asc'"
                        :sort-desc-active="tab.sortField === 'responsePacketSize' && tab.sortOrder === 'desc'"
                        @sort-asc="setTabSort(tab, 'responsePacketSize', 'asc')"
                        @sort-desc="setTabSort(tab, 'responsePacketSize', 'desc')"
                        @filter-change="handleColumnFilterInput(tab)" />
                    </template>
                    <template #default="{ row }">
                      {{ formatSize(row.responsePacketSize || row.size) }}
                    </template>
                  </el-table-column>
                  <el-table-column prop="time" width="142" align="right">
                    <template #header>
                      <TableColumnFilterHeader label="时间(ms)" v-model="tab.columnFilters.time" :popover-width="220"
                        placement="bottom-end" align-right sortable
                        :sort-asc-active="tab.sortField === 'time' && tab.sortOrder === 'asc'"
                        :sort-desc-active="tab.sortField === 'time' && tab.sortOrder === 'desc'"
                        @sort-asc="setTabSort(tab, 'time', 'asc')" @sort-desc="setTabSort(tab, 'time', 'desc')"
                        @filter-change="handleColumnFilterInput(tab)" />
                    </template>
                    <template #default="{ row }">
                      {{ row.time }}
                    </template>
                  </el-table-column>
                </el-table>
              </div>

              <div class="pagination-wrapper">
                <el-pagination small v-model:current-page="tab.currentPage" v-model:page-size="tab.pageSize"
                  :page-sizes="[10, 20, 50, 100, 200]" :total="getFilteredResults(tab).length"
                  layout="total, sizes, prev, pager, next, jumper" background
                  @size-change="(size) => handleSizeChange(size, tab)"
                  @current-change="(page) => handleCurrentChange(page, tab)" />
              </div>
            </el-card>
          </div>
        </el-tab-pane>
      </el-tabs>
    </CustomTabs>

    <!-- 高级配置对话框 -->
    <el-dialog v-model="showAdvancedDialog" title="高级配置" width="700px" destroy-on-close
      @closed="handleAdvancedDialogClosed">
      <div class="advanced-dialog-content" v-if="currentAdvancedTab">
        <div class="advanced-section">
          <h4 class="section-title">Fuzz配置</h4>
          <div class="advanced-row">
            <el-switch v-model="currentAdvancedTab.config.fuzz.enabled" active-text="启用Fuzz" size="default" />
          </div>
          <div v-if="currentAdvancedTab.config.fuzz.enabled" class="advanced-options">
            <div class="advanced-row">
              <div class="advanced-item">
                <label class="config-label">Fuzz位置</label>
                <el-select v-model="currentAdvancedTab.config.fuzz.position" size="default" style="width: 100%">
                  <el-option label="路径前缀" value="prefix" />
                  <el-option label="路径中间" value="middle" />
                  <el-option label="路径后缀" value="suffix" />
                  <el-option label="全路径" value="full" />
                </el-select>
              </div>
              <div class="advanced-item">
                <label class="config-label">Fuzz位数</label>
                <el-input-number v-model="currentAdvancedTab.config.fuzz.length" :min="1" :max="10"
                  controls-position="right" size="default" style="width: 100%" />
              </div>
            </div>
            <div class="advanced-row">
              <div class="advanced-item full-width">
                <label class="config-label">字符集</label>
                <el-select v-model="currentAdvancedTab.config.fuzz.charset" multiple placeholder="选择字符集" size="default"
                  style="width: 100%">
                  <el-option label="小写字母 (a-z)" value="lower" />
                  <el-option label="大写字母 (A-Z)" value="upper" />
                  <el-option label="数字 (0-9)" value="digits" />
                  <el-option label="特殊字符 (!@#$%)" value="special" />
                </el-select>
              </div>
            </div>
          </div>
        </div>

        <div class="advanced-section">
          <h4 class="section-title">过滤配置</h4>
          <div class="advanced-row">
            <div class="advanced-item full-width">
              <label class="config-label">状态码过滤</label>
              <el-select v-model="currentAdvancedTab.config.filter.statusCodes" multiple allow-create filterable
                :reserve-keyword="false" placeholder="选择或输入要过滤的状态码（支持手动输入）" size="default" style="width: 100%">
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
          <div class="advanced-row">
            <div class="advanced-item full-width">
              <label class="config-label">关键字过滤</label>
              <IsolatedTextarea
                v-if="currentAdvancedTab"
                :ref="(el) => setIsolatedRef('adv-kw-' + currentAdvancedTab.id, el)"
                v-model="currentAdvancedTab.config.filter.keywords"
                :rows="3"
                placeholder="每行一个关键字，匹配到的结果将被过滤"
                size="default"
              />
            </div>
          </div>
        </div>

        <div class="advanced-section">
          <h4 class="section-title">扩展名替换配置</h4>
          <div class="advanced-row">
            <div class="advanced-item full-width">
              <label class="config-label">%EXT%占位符替换</label>
              <IsolatedTextarea
                v-if="currentAdvancedTab"
                :ref="(el) => setIsolatedRef('adv-ext-' + currentAdvancedTab.id, el)"
                v-model="currentAdvancedTab.config.extensions"
                :rows="1"
                placeholder="php,jsp,asp,aspx,html,htm,js,json,xml,txt,bak,old,backup,swp,tmp"
                size="default"
              />
              <div class="config-hint" style="margin-top: 5px; color: #909399; font-size: 12px;">
                提示：字典中如 %EXT%.backup、%EXT%.bak 等路径会自动替换为 php.backup、jsp.backup 等
              </div>
            </div>
          </div>
        </div>

        <div class="advanced-section">
          <h4 class="section-title">请求配置</h4>
          <div class="advanced-row">
            <div class="advanced-item full-width">
              <label class="config-label">自定义请求头</label>
              <IsolatedTextarea
                v-if="currentAdvancedTab"
                :ref="(el) => setIsolatedRef('adv-hdr-' + currentAdvancedTab.id, el)"
                v-model="currentAdvancedTab.config.customHeaders"
                :rows="4"
                placeholder="每行一个请求头，格式: Key: Value"
                size="default"
              />
            </div>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 任务管理对话框 -->
    <el-dialog v-model="historyDialog" width="75%" destroy-on-close class="history-dialog"
      @opened="handleHistoryDialogOpened">
      <template #header>
        <div class="dialog-title" style="display: flex; justify-content: space-between; align-items: center;">
          <div>
            <el-icon :size="18" class="mr-5px">
              <Clock />
            </el-icon>任务管理
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
        <el-table :data="paginatedTasks" stripe border class="history-table" :height="400">
          <el-table-column type="index" width="60" label="#" align="center" />
          <el-table-column prop="name" label="任务名称" min-width="200">
            <template #default="{ row }">
              <el-tooltip :content="row.name || ''" placement="top" :disabled="!row.name || row.name.length <= 30"
                effect="dark">
                <div class="task-name-cell">
                  {{ row.name || '-' }}
                </div>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column prop="urlsText" label="目标" min-width="250">
            <template #default="{ row }">
              <el-tooltip v-if="row.urlsText && row.urlsText.split('\n').length > 2" :content="row.urlsText"
                placement="top" :show-after="300" effect="dark">
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
          <el-table-column label="进度" width="120" align="center">
            <template #default="{ row }">
              <el-progress v-if="row.status === 'running' || row.status === 'paused'" :percentage="row.progress || 0"
                :status="row.status === 'paused' ? 'warning' : undefined" :stroke-width="6" :show-text="true" />
              <span v-else-if="row.status === 'completed'">100%</span>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column label="关联任务" width="120" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.sourceTaskId" size="small" type="info">
                {{ getSourceTaskTypeName(row.sourceTaskType) }}
              </el-tag>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="创建时间" width="160" align="center">
            <template #default="{ row }">
              {{ formatTime(row.createdAt) }}
            </template>
          </el-table-column>
          <el-table-column prop="updatedAt" label="更新时间" width="160" align="center">
            <template #default="{ row }">
              {{ row.updatedAt ? formatTime(row.updatedAt) : '-' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" align="center">
            <template #default="{ row }">
              <el-button-group>
                <el-tooltip content="查看结果">
                  <el-button :icon="View" link @click="loadTask(row.taskId)" />
                </el-tooltip>
                <el-tooltip content="暂停任务" v-if="row.status === 'running'">
                  <el-button :icon="VideoPause" link @click="pauseTask(row.taskId)" />
                </el-tooltip>
                <el-tooltip content="恢复任务" v-if="row.status === 'paused'">
                  <el-button :icon="VideoPlay" link @click="resumeTask(row.taskId)" />
                </el-tooltip>
                <el-tooltip content="停止任务" v-if="row.status === 'running' || row.status === 'paused'">
                  <el-button :icon="VideoPause" link @click="stopTask(row.taskId)" />
                </el-tooltip>
                <el-tooltip content="删除">
                  <el-button :icon="Delete" link @click="deleteTask(row.taskId)" />
                </el-tooltip>
              </el-button-group>
            </template>
          </el-table-column>
        </el-table>
        <div class="history-pagination">
          <el-pagination small v-model:current-page="taskPagination.currentPage"
            v-model:page-size="taskPagination.pageSize" :total="tasks.length" :page-sizes="[10, 20, 50, 100, 200]"
            layout="total, sizes, prev, pager, next, jumper" background />
        </div>
      </div>
    </el-dialog>

    <!-- 右键菜单 -->
    <!-- 右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
    <Teleport to="body">
      <div v-if="contextMenuVisible" class="context-menu" :style="contextMenuStyle" @click.stop>
        <div class="context-menu-item" @click="handleViewPacket">
          <el-icon>
            <View />
          </el-icon>
          <span>查看数据包</span>
        </div>
        <div class="context-menu-item" @click="handleRunJsfinder">
          <el-icon>
            <View />
          </el-icon>
          <span>JSFinder 接口探测</span>
        </div>
        <div class="context-menu-item" @click="handleCopyRow">
          <el-icon>
            <CopyDocument />
          </el-icon>
          <span>复制当前行</span>
        </div>
        <div class="context-menu-item" @click="handleCopyAll">
          <el-icon>
            <CopyDocument />
          </el-icon>
          <span>复制所有内容</span>
        </div>
        <div class="context-menu-divider"></div>
        <div class="context-menu-item danger" @click="handleDeleteRow">
          <el-icon>
            <Delete />
          </el-icon>
          <span>删除当前行</span>
        </div>
      </div>
    </Teleport>

    <!-- 数据包查看对话框 -->
    <PacketViewer v-model="packetViewerVisible" :request-packet="selectedRow?.requestPacket || ''"
      :response-packet="selectedRow?.responsePacket || ''" :title="`数据包查看 - ${selectedRow ? selectedRow.url : ''}`"
      :show-send-actions="false" />

    <!-- 标签页右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
    <Teleport to="body">
      <ul class="tabs-context-menu" v-if="tabsContextMenuVisible" :style="tabsContextMenuStyle">
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
  FolderOpened,
  Timer,
  Connection
} from '@element-plus/icons-vue'
import { markRaw } from 'vue'
import { EventsOn, EventsOff, BrowserOpenURL } from '../../../wailsjs/runtime'
import { StartDirectoryScan, StopDirectoryScan, GetDictList } from '../../../wailsjs/go/dirsearch/DirsearchController'
import { SaveFile, GetAppPath, ReadFile } from '../../../wailsjs/go/controller/Note'
import { CreateTask as CreateBrowserScanTask } from '../../../wailsjs/go/browserScan/BrowserScan'
import * as XLSX from 'xlsx'
import PacketViewer from '../../components/PacketViewer.vue'
import CustomTabs from '../../components/CustomTabs.vue'
import TableColumnFilterHeader from '../../components/TableColumnFilterHeader.vue'
import IsolatedTextarea from '../../components/IsolatedTextarea.vue'

export default {
  name: 'DirectoryScanner',

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
    Plus,
    Clock,
    Refresh,
    CircleCheck,
    FolderOpened,
    CustomTabs,
    TableColumnFilterHeader,
    PacketViewer,
    IsolatedTextarea
  },

  data() {
    return {
      // 图标组件（使用 markRaw 避免响应式）
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
      FolderOpened: markRaw(FolderOpened),
      Timer: markRaw(Timer),
      Connection: markRaw(Connection),
      scanTabs: [],
      activeTabId: '',
      tabIdCounter: 0,
      historyDialog: false,
      historyList: [],
      historyLoading: false,
      historyPagination: {
        currentPage: 1,
        pageSize: 10
      },
      // 任务管理（独立于标签页）
      tasksLocal: {}, // { taskId: task }
      tasks: [], // 任务列表（用于显示）
      taskPagination: {
        currentPage: 1,
        pageSize: 10
      },
      taskIdCounter: 0,
      operationLock: new Set(), // 操作锁，防止重复操作
      showAdvancedDialog: false,
      currentAdvancedTab: null,
      dicts: [],
      tableHeight: 420,
      contextMenuVisible: false,
      contextMenuPos: { x: 0, y: 0 },
      selectedRow: null,
      selectedTab: null,
      packetViewerVisible: false,
      // 标签页右键菜单
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: '',
      // 联动目录扫描填充防抖控制（按 sourceTaskId 聚合）
      dirscanFillDebounceMap: {},
      dirscanFillPendingTargets: {},
      /** IsolatedTextarea 实例，用于开始扫描前 flush，避免未失焦时内容未回写 */
      isolatedTextareaRefs: {}
    }
  },

  computed: {
    currentTab() {
      return this.scanTabs.find(tab => tab.id === this.activeTabId) || null
    },
    paginatedHistory() {
      const start = (this.historyPagination.currentPage - 1) * this.historyPagination.pageSize
      const end = start + this.historyPagination.pageSize
      return this.historyList.slice(start, end)
    },
    paginatedTasks() {
      const start = (this.taskPagination.currentPage - 1) * this.taskPagination.pageSize
      const end = start + this.taskPagination.pageSize
      return this.tasks.slice(start, end)
    },
    contextMenuStyle() {
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
    activeTabId(newVal) {
      if (newVal) {
        this.$nextTick(() => {
          this.calcTableHeight()
        })
      }
    }
  },

  mounted() {
    this.loadDictList()
    this.loadTasks()
    this.calcTableHeight()
    window.addEventListener('resize', this.calcTableHeight)
    window.addEventListener('cyberSabre-dirscan-request', this.handleExternalDirscan)
    window.addEventListener('cyberSabre-dirscan-fill', this.handleExternalDirscanFill)
    this.bindWailsEvents()

    // 标记目录扫描模块已就绪，供父级事件代理判断
    window.__dirsearchReady = true
    // 标记组件已激活（用于 keep-alive 场景）
    window.__dirsearchActivated = true

    // 同步标签页的扫描状态（确保从其他页面跳转回来时状态正确）
    this.$nextTick(() => {
      this.syncTabsScanningState()
    })

    // 如果在本模块挂载之前有主机扫描联动过来的待处理任务，这里一次性处理
    if (Array.isArray(window.__cyberSabrePendingDirscan) && window.__cyberSabrePendingDirscan.length > 0) {
      try {
        window.__cyberSabrePendingDirscan.forEach(payload => {
          // 复用现有的联动处理逻辑
          this.handleExternalDirscanFill({ detail: payload })
        })
      } catch (e) {
        console.error('处理待处理目录扫描联动数据失败:', e)
      }
      // 清空队列，避免重复处理
      window.__cyberSabrePendingDirscan = []
    }

    if (this.scanTabs.length === 0) {
      this.createNewTab()
    }
    this.scanTabs.forEach(t => this.ensureTabColumnFilters(t))
  },

  // keep-alive 激活时
  activated() {
    // 标记组件已激活
    window.__dirsearchActivated = true
    // 同步标签页的扫描状态（确保从其他页面跳转回来时状态正确）
    this.$nextTick(() => {
      this.syncTabsScanningState()
    })
    // 处理待处理的联动任务（如果切换标签页时有新的联动任务）
    if (Array.isArray(window.__cyberSabrePendingDirscan) && window.__cyberSabrePendingDirscan.length > 0) {
      this.$nextTick(() => {
        try {
          window.__cyberSabrePendingDirscan.forEach(payload => {
            this.handleExternalDirscanFill({ detail: payload })
          })
          window.__cyberSabrePendingDirscan = []
        } catch (e) {
          console.error('处理待处理目录扫描联动数据失败:', e)
        }
      })
    }
  },

  // keep-alive 失活时
  deactivated() {
    // 标记组件未激活
    window.__dirsearchActivated = false
  },

  beforeUnmount() {
    window.removeEventListener('resize', this.calcTableHeight)
    window.removeEventListener('cyberSabre-dirscan-request', this.handleExternalDirscan)
    window.removeEventListener('cyberSabre-dirscan-fill', this.handleExternalDirscanFill)
    this.unbindWailsEvents()
    // 清理联动防抖定时器
    if (this.dirscanFillDebounceMap) {
      Object.keys(this.dirscanFillDebounceMap).forEach(key => {
        const timer = this.dirscanFillDebounceMap[key]
        if (timer) {
          clearTimeout(timer)
        }
      })
    }
    this.dirscanFillDebounceMap = {}
    this.dirscanFillPendingTargets = {}
    // 清理所有标签页的定时器
    this.scanTabs.forEach(tab => {
      if (tab.timeInterval) {
        clearInterval(tab.timeInterval)
        tab.timeInterval = null
      }
    })

    // 组件卸载时重置就绪标记
    if (window.__dirsearchReady) {
      window.__dirsearchReady = false
    }
    if (window.__dirsearchActivated) {
      window.__dirsearchActivated = false
    }
  },

  methods: {
    setIsolatedRef(key, el) {
      if (el) {
        this.isolatedTextareaRefs[key] = el
      } else {
        delete this.isolatedTextareaRefs[key]
      }
    },
    flushAdvancedTextareas(tabId) {
      ;[`adv-kw-${tabId}`, `adv-ext-${tabId}`, `adv-hdr-${tabId}`].forEach((k) => {
        this.isolatedTextareaRefs[k]?.flush?.()
      })
    },
    /** 将 IsolatedTextarea 中未失焦的内容写回 tab，避免开始扫描读到旧值 */
    flushPendingTextareas(tab) {
      if (!tab) return
      this.isolatedTextareaRefs[`urls-${tab.id}`]?.flush?.()
      this.isolatedTextareaRefs[`customDict-${tab.id}`]?.flush?.()
      if (this.currentAdvancedTab && this.currentAdvancedTab.id === tab.id) {
        this.flushAdvancedTextareas(tab.id)
      }
    },
    handleAdvancedDialogClosed() {
      if (this.currentAdvancedTab) {
        this.flushAdvancedTextareas(this.currentAdvancedTab.id)
      }
    },
    // 标签页管理
    createNewTab() {
      const tabId = `tab-${++this.tabIdCounter}`
      const tab = {
        id: tabId,
        name: `目录扫描-${new Date().toLocaleTimeString()}`,
        urlsText: '',
        config: {
          method: 'GET',
          threads: 30,
          timeout: 5,
          recursionDepth: 0,
          followRedirects: true,
          customHeaders: '',
          customDict: '',
          extensions: 'php,jsp,asp,aspx,html,htm,js,json,xml,txt,bak,old,backup,swp,tmp', // %EXT%替换扩展名列表，逗号分隔
          fuzz: {
            enabled: false,
            position: 'suffix',
            length: 3,
            charset: ['lower', 'digits'],
            customChars: '',
            generateCount: 1000
          },
          filter: {
            statusCodes: ['404', '400', '403', '500'],
            sizeRange: [0, 1000000],
            keywords: '',
            regex: ''
          }
        },
        selectedDicts: ['dicc.txt'],
        scanning: false,
        loading: false,
        completed: false,
        scanProgress: 0,
        scanStartTime: null,
        elapsedTime: '00:00',
        results: [],
        stats: {
          totalRequests: 0,
          success: 0,
          failed: 0,
          found: 0,
          requestsPerSecond: 0
        },
        baselineStats: {
          totalRequests: 0,
          success: 0,
          failed: 0,
          found: 0
        },
        searchKeyword: '',
        columnFilters: {
          status: '',
          url: '',
          title: '',
          size: '',
          responsePacketSize: '',
          time: ''
        },
        sortField: 'status',
        sortOrder: 'asc',
        currentPage: 1,
        pageSize: 10,
        timeInterval: null,
        createTime: Date.now(),
        isStopped: false // 标记是否已停止扫描
      }
      this.scanTabs.push(tab)
      this.activeTabId = tabId
      this.saveHistory()
      return tab
    },
    async removeTab(tabId) {
      const index = this.scanTabs.findIndex(tab => tab.id === tabId)
      if (index !== -1) {
        const tab = this.scanTabs[index]
        // 关闭标签页时，不停止扫描，允许后台继续运行
        // 清理定时器（仅前端的定时器，不影响后端扫描）
        if (tab.timeInterval) {
          clearInterval(tab.timeInterval)
          tab.timeInterval = null
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
          this.activeTabId = this.scanTabs[Math.min(index, this.scanTabs.length - 1)].id
        } else {
          // 避免进入无标签空白态，导致后续弹层/布局异常
          this.createNewTab()
        }
        this.saveHistory()
      }
    },
    handleTabClick(tab) {
      this.activeTabId = tab.paneName
      this.$nextTick(() => {
        this.calcTableHeight()
        setTimeout(() => this.calcTableHeight(), 50)
      })
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
        if (tab.timeInterval) {
          clearInterval(tab.timeInterval)
          tab.timeInterval = null
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
      this.saveHistory()
    },
    // 关闭所有标签页
    async handleCloseAllTabs() {
      this.hideTabsContextMenu()

      // 关闭所有标签页，不停止扫描，允许后台继续运行
      for (const tab of this.scanTabs) {
        // 不停止扫描，允许后台继续运行
        // 清理定时器（仅前端的定时器，不影响后端扫描）
        if (tab.timeInterval) {
          clearInterval(tab.timeInterval)
          tab.timeInterval = null
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
      this.saveHistory()
    },
    ensureTabColumnFilters(tab) {
      if (!tab || tab.columnFilters) return
      tab.columnFilters = {
        status: '',
        url: '',
        title: '',
        size: '',
        responsePacketSize: '',
        time: ''
      }
    },
    resetColumnFilters(tab) {
      this.ensureTabColumnFilters(tab)
      Object.assign(tab.columnFilters, {
        status: '',
        url: '',
        title: '',
        size: '',
        responsePacketSize: '',
        time: ''
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
    getFilteredResults(tab) {
      this.ensureTabColumnFilters(tab)
      let filtered = tab.results || []

      if (tab.searchKeyword) {
        const keyword = tab.searchKeyword.toLowerCase()
        filtered = filtered.filter(item => {
          return (
            item.url?.toLowerCase().includes(keyword) ||
            item.title?.toLowerCase().includes(keyword) ||
            String(item.status).includes(keyword)
          )
        })
      }

      if (tab.config.filter.statusCodes.length > 0) {
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

      const cf = tab.columnFilters
      const statusQ = (cf.status || '').trim().toLowerCase()
      if (statusQ) {
        filtered = filtered.filter(item =>
          String(item.status ?? '').toLowerCase().includes(statusQ)
        )
      }
      const urlQ = (cf.url || '').trim().toLowerCase()
      if (urlQ) {
        filtered = filtered.filter(item =>
          (item.url || '').toLowerCase().includes(urlQ)
        )
      }
      const titleQ = (cf.title || '').trim().toLowerCase()
      if (titleQ) {
        filtered = filtered.filter(item =>
          (item.title || '').toLowerCase().includes(titleQ)
        )
      }
      const sizeQ = (cf.size || '').trim().toLowerCase()
      if (sizeQ) {
        filtered = filtered.filter(item => {
          const raw = String(item.size ?? '')
          const fmt = this.formatSize(item.size).toLowerCase()
          return raw.includes(sizeQ) || fmt.includes(sizeQ)
        })
      }
      const pktQ = (cf.responsePacketSize || '').trim().toLowerCase()
      if (pktQ) {
        filtered = filtered.filter(item => {
          const v = item.responsePacketSize ?? item.size
          const raw = String(v ?? '')
          const fmt = this.formatSize(v).toLowerCase()
          return raw.includes(pktQ) || fmt.includes(pktQ)
        })
      }
      const timeQ = (cf.time || '').trim().toLowerCase()
      if (timeQ) {
        filtered = filtered.filter(item =>
          String(item.time ?? '').toLowerCase().includes(timeQ)
        )
      }

      return filtered
    },
    getSortedResults(tab) {
      const filtered = this.getFilteredResults(tab)
      const sorted = [...filtered]
      const prop = tab.sortField
      const order = tab.sortOrder

      if (prop && order) {
        const numericProps = ['status', 'size', 'time']
        sorted.sort((a, b) => {
          let aVal
          let bVal
          if (prop === 'responsePacketSize') {
            aVal = Number(a.responsePacketSize ?? a.size)
            bVal = Number(b.responsePacketSize ?? b.size)
          } else if (numericProps.includes(prop)) {
            aVal = Number(a[prop])
            bVal = Number(b[prop])
          } else {
            aVal = String(a[prop] ?? '').toLowerCase()
            bVal = String(b[prop] ?? '').toLowerCase()
          }
          if (typeof aVal === 'number') {
            if (Number.isNaN(aVal)) aVal = 0
            if (Number.isNaN(bVal)) bVal = 0
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
      const end = start + tab.pageSize
      return sorted.slice(start, end)
    },
    // 保存历史记录到文件
    async saveHistoryToFile(historyId, historyData) {
      try {
        const appPath = await GetAppPath()
        const pathSeparator = appPath.includes('\\') ? '\\' : '/'
        const resultsDir = `${appPath}${pathSeparator}logs${pathSeparator}dirsearch${pathSeparator}history`

        const timestamp = Date.now()
        const fileName = `dirsearch_history_${historyId}_${timestamp}.json`
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
    async saveHistory() {
      try {
        const history = this.scanTabs.map(tab => ({
          id: tab.id,
          name: tab.name,
          urlsText: tab.urlsText,
          config: JSON.parse(JSON.stringify(tab.config)),
          selectedDicts: tab.selectedDicts,
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
            selectedDicts: item.selectedDicts,
            results: undefined, // 不保存完整结果到localStorage
            resultsCount: item.results ? item.results.length : 0,
            stats: item.stats || {},
            createTime: item.createTime,
            resultsFilePath: item.resultsFilePath || null
          })
        }

        localStorage.setItem('dirsearchHistory', JSON.stringify(compressedHistory))
      } catch (error) {
        console.error('保存历史记录失败:', error)
      }
    },
    async loadHistory() {
      try {
        const historyStr = localStorage.getItem('dirsearchHistory')
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

          this.historyList = history.map(item => ({
            ...item,
            scanning: false,
            loading: false,
            completed: true,
            scanProgress: 0,
            scanStartTime: null,
            elapsedTime: '00:00',
            baselineStats: {
              totalRequests: 0,
              success: 0,
              failed: 0,
              found: 0
            },
            searchKeyword: '',
            columnFilters: {
              status: '',
              url: '',
              title: '',
              size: '',
              responsePacketSize: '',
              time: ''
            },
            sortField: 'status',
            sortOrder: 'asc',
            currentPage: 1,
            pageSize: 10,
            timeInterval: null
          }))
        }
      } catch (error) {
        console.error('加载历史记录失败:', error)
      }
    },
    handleHistoryDialogOpened() {
      this.loadTasks()
    },
    async refreshTasks() {
      this.historyLoading = true
      await this.loadTasks()

      // 同步任务状态到标签页
      this.syncTaskStatusToTabs()

      setTimeout(() => {
        this.historyLoading = false
        this.$message.success('刷新成功')
      }, 100)
    },
    // 同步标签页的扫描状态（专门用于解决跳转时状态不同步的问题）
    syncTabsScanningState() {
      // 先确保任务已加载
      if (!this.tasksLocal || Object.keys(this.tasksLocal).length === 0) {
        return
      }

      // 遍历所有标签页，检查对应的任务状态
      this.scanTabs.forEach(tab => {
        if (tab.taskId) {
          const task = this.tasksLocal[tab.taskId]
          if (task) {
            // 如果任务状态为 running，但标签页状态不是 scanning，则同步状态
            if (task.status === 'running' && !tab.scanning) {
              tab.scanning = true
              tab.loading = false
              // 如果扫描已开始，启动计时器
              if (!tab.timeInterval && tab.scanStartTime) {
                this.startTimeCounter(tab)
              }
            } else if (task.status !== 'running' && tab.scanning) {
              // 如果任务状态不是 running，但标签页状态是 scanning，也需要同步
              tab.scanning = false
              tab.loading = false
            }
            // 同步其他状态
            tab.completed = task.status === 'completed'
            tab.scanProgress = task.progress || 0
          }
        } else {
          // 如果标签页没有 taskId，但有 sourceTaskId，尝试通过 sourceTaskId 查找任务
          if (tab.sourceTaskId && tab.sourceTaskType) {
            const matchedTask = Object.values(this.tasksLocal).find(t =>
              t.sourceTaskId === tab.sourceTaskId &&
              t.sourceTaskType === tab.sourceTaskType &&
              t.status === 'running'
            )
            if (matchedTask && !tab.scanning) {
              // 找到匹配的运行中任务，更新标签页状态
              tab.scanning = true
              tab.loading = false
              tab.taskId = matchedTask.taskId
              tab.completed = matchedTask.status === 'completed'
              tab.scanProgress = matchedTask.progress || 0
            }
          }
        }
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
              ...task.stats
            }
          }

          // 如果任务有结果数据，同步结果（但不要覆盖标签页中可能更新的数据）
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

      // 同步完成后，再次调用 syncTabsScanningState 确保所有状态都正确
      this.syncTabsScanningState()
    },
    // 保存扫描结果到本地JSON文件
    async saveResultsToFile(taskId, results, stats) {
      try {
        const appPath = await GetAppPath()
        const pathSeparator = appPath.includes('\\') ? '\\' : '/'
        const resultsDir = `${appPath}${pathSeparator}logs${pathSeparator}dirsearch`

        const timestamp = Date.now()
        const fileName = `dirsearch_${taskId}_${timestamp}.json`
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
    },
    // 从文件加载扫描结果
    async loadResultsFromFile(filePath) {
      try {
        if (!filePath) return null
        const content = await ReadFile(filePath)
        return JSON.parse(content)
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
          // 如果任务有完整结果且没有保存到文件，则保存到文件
          if (task.results && task.results.length > 0 && !task.resultsFilePath) {
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

          localStorage.setItem('dirsearchTasks', JSON.stringify(limitedTasks))
        } else {
          localStorage.setItem('dirsearchTasks', tasksStr)
        }
      } catch (error) {
        if (error.name === 'QuotaExceededError') {
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

            localStorage.setItem('dirsearchTasks', JSON.stringify(limitedTasks))
            this.tasksLocal = limitedTasks
            this.tasks = Object.values(this.tasksLocal)
          } catch (e) {
            console.error('清理后保存任务仍然失败:', e)
            localStorage.removeItem('dirsearchTasks')
          }
        } else {
          console.error('保存任务失败:', error)
        }
      }
    },
    async loadTasks() {
      try {
        const tasksStr = localStorage.getItem('dirsearchTasks')
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

          // 加载任务后，同步标签页状态（确保状态正确）
          this.$nextTick(() => {
            this.syncTabsScanningState()
          })
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
        const tabId = `tab-${++this.tabIdCounter}`
        tab = {
          id: tabId,
          name: task.name,
          urlsText: task.urlsText || '',
          config: task.config || {
            method: 'GET',
            threads: 50,
            timeout: 10,
            recursionDepth: 0,
            followRedirects: true,
            customHeaders: '',
            customDict: '',
            extensions: '',
            fuzz: {
              enabled: false,
              position: 'suffix',
              length: 3,
              charset: ['lower', 'digits'],
              customChars: '',
              generateCount: 1000
            },
            filter: {
              statusCodes: ['404', '400', '403', '500'],
              sizeRange: [0, 1000000],
              keywords: '',
              regex: ''
            }
          },
          selectedDicts: task.config?.builtinDicts || ['dicc.txt'],
          scanning: task.status === 'running',
          loading: false,
          completed: task.status === 'completed',
          scanProgress: task.progress || 0,
          scanStartTime: null,
          elapsedTime: '00:00',
          results: Array.isArray(results) ? JSON.parse(JSON.stringify(results)) : [],
          stats: {
            totalRequests: stats.totalRequests || 0,
            success: stats.success || 0,
            failed: 0,
            found: 0,
            requestsPerSecond: 0
          },
          baselineStats: {
            totalRequests: 0,
            success: 0,
            failed: 0,
            found: 0
          },
          searchKeyword: '',
          columnFilters: {
            status: '',
            url: '',
            title: '',
            size: '',
            responsePacketSize: '',
            time: ''
          },
          sortField: 'status',
          sortOrder: 'asc',
          currentPage: 1,
          pageSize: 10,
          timeInterval: null,
          createTime: task.createdAt || Date.now(),
          taskId: taskId,
          isStopped: false // 标记是否已停止扫描
        }
        this.scanTabs.push(tab)
      } else {
        // 如果标签页已存在，更新标签页数据（确保数据同步）
        tab.name = task.name
        tab.urlsText = task.urlsText || ''
        tab.config = task.config ? JSON.parse(JSON.stringify(task.config)) : tab.config
        tab.scanning = task.status === 'running'
        tab.completed = task.status === 'completed'
        tab.scanProgress = task.progress || 0
        if (!tab.columnFilters) {
          tab.columnFilters = {
            status: '',
            url: '',
            title: '',
            size: '',
            responsePacketSize: '',
            time: ''
          }
        }
      }

      // 切换到该标签页
      this.activeTabId = tab.id
      this.historyDialog = false

      // 加载任务结果（如果有保存的结果）
      if (Array.isArray(results) && results.length > 0) {
        tab.results = JSON.parse(JSON.stringify(results))
        // 重新计算统计信息
        this.calculateStatsFromResults(tab)
        this.$message.success(`任务已加载到标签页，共 ${results.length} 条结果`)
      } else if (task.results && Array.isArray(task.results) && task.results.length > 0) {
        tab.results = JSON.parse(JSON.stringify(task.results))
        // 重新计算统计信息
        this.calculateStatsFromResults(tab)
        this.$message.success(`任务已加载到标签页，共 ${task.results.length} 条结果`)
      } else {
        this.$message.success('任务已加载到标签页')
      }
    },
    async pauseTask(taskId) {
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
        if (task.status !== 'running') {
          this.$message.warning('任务未运行，无法暂停')
          return
        }

        // 找到关联的标签页并暂停
        const tab = this.scanTabs.find(t => t.taskId === taskId)
        if (tab && tab.scanning) {
          await this.stopScan(tab)
        }

        task.status = 'paused'
        this.saveTasks()
        this.$message.success('任务已暂停')
      } catch (error) {
        console.error('暂停任务失败:', error)
        this.$message.error('暂停失败')
      } finally {
        setTimeout(() => {
          this.operationLock.delete(taskId)
        }, 1000)
      }
    },
    async resumeTask(taskId) {
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
        if (task.status !== 'paused') {
          this.$message.warning('任务未暂停，无法恢复')
          return
        }

        // 找到关联的标签页并恢复
        const tab = this.scanTabs.find(t => t.taskId === taskId)
        if (tab) {
          // 重新启动扫描
          await this.startScanInternal(tab, true, false, true)
        } else {
          // 如果没有标签页，创建新标签页并启动
          this.loadTask(taskId)
          const newTab = this.scanTabs.find(t => t.taskId === taskId)
          if (newTab) {
            await this.startScanInternal(newTab, true, false, true)
          }
        }

        task.status = 'running'
        this.saveTasks()
        this.$message.success('任务已恢复')
      } catch (error) {
        console.error('恢复任务失败:', error)
        this.$message.error('恢复失败')
      } finally {
        setTimeout(() => {
          this.operationLock.delete(taskId)
        }, 1000)
      }
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
      localStorage.removeItem('dirsearchTasks')
      this.scanTabs.forEach(tab => {
        tab.taskId = ''
        tab.scanning = false
      })
      this.$message.success('已清除全部任务')
    },
    loadHistoryTab(historyItem) {
      const tabId = historyItem.id
      const existingTab = this.scanTabs.find(tab => tab.id === tabId)
      if (existingTab) {
        this.activeTabId = tabId
        this.historyDialog = false
        return
      }
      const tab = {
        id: tabId,
        name: historyItem.name,
        urlsText: historyItem.urlsText || '',
        config: historyItem.config || {
          method: 'GET',
          threads: 50,
          timeout: 10,
          recursionDepth: 0,
          followRedirects: true,
          customHeaders: '',
          customDict: '',
          extensions: '',
          fuzz: {
            enabled: false,
            position: 'suffix',
            length: 3,
            charset: ['lower', 'digits'],
            customChars: '',
            generateCount: 1000
          },
          filter: {
            statusCodes: ['404', '400', '403', '500'],
            sizeRange: [0, 1000000],
            keywords: '',
            regex: ''
          }
        },
        selectedDicts: historyItem.selectedDicts || ['dicc.txt'],
        scanning: false,
        loading: false,
        completed: true,
        scanProgress: 0,
        scanStartTime: null,
        elapsedTime: '00:00',
        results: historyItem.results || [],
        stats: historyItem.stats || {
          totalRequests: 0,
          success: 0,
          failed: 0,
          found: 0,
          requestsPerSecond: 0
        },
        baselineStats: {
          totalRequests: 0,
          success: 0,
          failed: 0,
          found: 0
        },
        searchKeyword: '',
        columnFilters: {
          status: '',
          url: '',
          title: '',
          size: '',
          responsePacketSize: '',
          time: ''
        },
        sortField: 'status',
        sortOrder: 'asc',
        currentPage: 1,
        pageSize: 10,
        timeInterval: null,
        createTime: historyItem.createTime || Date.now(),
        isStopped: false // 标记是否已停止扫描
      }
      this.scanTabs.push(tab)
      this.activeTabId = tabId
      this.historyDialog = false
      this.saveHistory()
    },
    async deleteHistory(historyItem) {
      try {
        await this.$confirm('确定要删除这条历史记录吗？', '确认删除', { type: 'warning' })
      } catch {
        return
      }
      const index = this.historyList.findIndex(item => item.id === historyItem.id)
      if (index !== -1) {
        this.historyList.splice(index, 1)
        localStorage.setItem('dirsearchHistory', JSON.stringify(this.historyList))
        this.$message.success('删除成功')
      }
    },
    async clearAllHistory() {
      try {
        await this.$confirm('确定要清除所有历史记录吗？', '确认清除', { type: 'warning' })
      } catch {
        return
      }
      this.historyList = []
      localStorage.removeItem('dirsearchHistory')
      this.$message.success('清除成功')
    },
    formatTime(timestamp) {
      if (!timestamp) return '-'
      const date = new Date(timestamp)
      const year = date.getFullYear()
      const month = String(date.getMonth() + 1).padStart(2, '0')
      const day = String(date.getDate()).padStart(2, '0')
      const hours = String(date.getHours()).padStart(2, '0')
      const minutes = String(date.getMinutes()).padStart(2, '0')
      const seconds = String(date.getSeconds()).padStart(2, '0')
      return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
    },
    async loadDictList() {
      try {
        const dictList = await GetDictList()
        this.dicts = dictList || []
        // 如果当前标签页存在且没有选择字典，则默认选择 dicc.txt
        if (this.dicts.length > 0 && this.currentTab && (!this.currentTab.selectedDicts || this.currentTab.selectedDicts.length === 0)) {
          const dicc = this.dicts.find(d => d.name === 'dicc.txt')
          if (dicc) {
            this.currentTab.selectedDicts = ['dicc.txt']
          }
        }
      } catch (error) {
        console.error('加载字典列表失败:', error)
        this.$message.error({
          message: '加载字典列表失败',
          duration: 2000
        })
      }
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
          this.tableHeight = Math.max(height, 450)  // 最低不低于 220px
        }
      })
    },

    handleExternalDirscan(event) {
      const payload = event?.detail
      const targets = payload?.targets

      if (!Array.isArray(targets) || targets.length === 0) {
        this.$message.warning('目录扫描联动失败：缺少有效目标')
        return
      }

      const currentTab = this.currentTab || this.scanTabs[0]
      if (!currentTab) {
        this.createNewTab()
        return
      }

      // 检查当前标签是否正在扫描
      if (currentTab.scanning) {
        this.$message.warning('当前标签正在扫描中，请先停止扫描后再进行联动')
        return
      }

      this.applyExternalScanConfig(payload)
      // 自动联动时，直接保存记录并开始扫描，不触发覆盖提示
      this.$nextTick(() => {
        this.startScanInternal(currentTab, true) // 传入tab和true表示自动联动，跳过覆盖提示
      })
    },

    handleExternalDirscanFill(event) {
      const payload = event?.detail
      const targets = payload?.targets
      const sourceTaskId = payload?.sourceTaskId // 来源任务ID（用于关联）

      if (!Array.isArray(targets) || targets.length === 0) {
        return
      }

      // 如果提供了 sourceTaskId，使用防抖机制合并短时间内同一来源任务的联动请求
      if (sourceTaskId) {
        const debounceKey = `${sourceTaskId}-${payload?.sourceTaskType || 'hostscan'}`

        // 先检查是否已经存在相同 sourceTaskId 的标签页（即使防抖还在进行中）
        let existingTab = this.scanTabs.find(tab =>
          tab.sourceTaskId === sourceTaskId &&
          tab.sourceTaskType === (payload?.sourceTaskType || 'hostscan')
        )

        // 如果没有找到相同 sourceTaskId 的标签页，尝试复用默认标签页
        if (!existingTab && sourceTaskId) {
          // 优先复用默认标签页（没有关联来源任务的标签页）
          if (this.scanTabs.length === 1) {
            const onlyTab = this.scanTabs[0]
            if (!onlyTab.sourceTaskId) {
              onlyTab.urlsText = ''
              onlyTab.results = []
              onlyTab.scanning = false
              onlyTab.completed = false
              onlyTab.scanProgress = 0
              onlyTab.stats = {
                totalRequests: 0,
                success: 0,
                failed: 0,
                found: 0,
                requestsPerSecond: 0
              }
              this.resetColumnFilters(onlyTab)
              existingTab = onlyTab
              existingTab.sourceTaskId = sourceTaskId
              existingTab.sourceTaskType = payload?.sourceTaskType || 'hostscan'
            }
          }

          // 如果仍未匹配到标签，再尝试复用一个"空闲标签页"
          if (!existingTab) {
            existingTab = this.scanTabs.find(tab => {
              const noUrls = !tab.urlsText || tab.urlsText.trim().length === 0
              const noResults = !tab.results || tab.results.length === 0
              const notScanning = !tab.scanning
              const noSourceTaskId = !tab.sourceTaskId
              return noUrls && noResults && notScanning && noSourceTaskId
            }) || null

            if (existingTab) {
              existingTab.sourceTaskId = sourceTaskId
              existingTab.sourceTaskType = payload?.sourceTaskType || 'hostscan'
            }
          }
        }

        // 如果已存在标签页（包括复用的默认标签页），直接追加URL，不经过防抖
        if (existingTab) {
          const existingUrls = existingTab.urlsText.split('\n').filter(url => url.trim())
          const newUrls = targets.filter(url => !existingUrls.includes(url.trim()))
          if (newUrls.length > 0) {
            existingTab.urlsText = [...existingUrls, ...newUrls].join('\n')
          }
          this.activeTabId = existingTab.id
          this.applyExternalScanConfigToTab(payload, existingTab)
          // 如果标签页未在扫描中，自动开始扫描
          if (!existingTab.scanning) {
            this.$nextTick(() => {
              if (existingTab.selectedDicts.length === 0 && !existingTab.config.customDict.trim() && !existingTab.config.fuzz.enabled) {
                existingTab.selectedDicts = ['dicc.txt']
              }
              this.startScan(existingTab, true)
            })
          }
          return
        }

        // 收集 targets（合并所有同 sourceTaskId 的请求）
        if (!this.dirscanFillPendingTargets[debounceKey]) {
          this.dirscanFillPendingTargets[debounceKey] = {
            payload: { ...payload },
            targets: []
          }
        }

        const pending = this.dirscanFillPendingTargets[debounceKey]
        const existingTargets = pending.targets
        targets.forEach(url => {
          if (!existingTargets.includes(url)) {
            existingTargets.push(url)
          }
        })

        // 更新 payload 中的 targets（使用合并后的 targets）
        pending.payload.targets = [...pending.targets]

        // 清除之前的防抖定时器
        if (this.dirscanFillDebounceMap[debounceKey]) {
          clearTimeout(this.dirscanFillDebounceMap[debounceKey])
        }

        // 设置新的防抖定时器，延迟处理以避免重复创建标签页
        this.dirscanFillDebounceMap[debounceKey] = setTimeout(() => {
          const currentPending = this.dirscanFillPendingTargets[debounceKey]
          if (currentPending && currentPending.targets.length > 0) {
            // 在防抖定时器触发时，再次检查是否已经存在标签页（防止并发创建）
            let checkTab = this.scanTabs.find(tab =>
              tab.sourceTaskId === sourceTaskId &&
              tab.sourceTaskType === (currentPending.payload?.sourceTaskType || 'hostscan')
            )

            // 如果没有找到相同 sourceTaskId 的标签页，尝试复用默认标签页
            if (!checkTab && sourceTaskId) {
              // 优先复用默认标签页（没有关联来源任务的标签页）
              if (this.scanTabs.length === 1) {
                const onlyTab = this.scanTabs[0]
                if (!onlyTab.sourceTaskId) {
                  onlyTab.urlsText = ''
                  onlyTab.results = []
                  onlyTab.scanning = false
                  onlyTab.completed = false
                  onlyTab.scanProgress = 0
                  onlyTab.stats = {
                    totalRequests: 0,
                    success: 0,
                    failed: 0,
                    found: 0,
                    requestsPerSecond: 0
                  }
                  this.resetColumnFilters(onlyTab)
                  checkTab = onlyTab
                  checkTab.sourceTaskId = sourceTaskId
                  checkTab.sourceTaskType = currentPending.payload?.sourceTaskType || 'hostscan'
                }
              }

              // 如果仍未匹配到标签，再尝试复用一个"空闲标签页"
              if (!checkTab) {
                checkTab = this.scanTabs.find(tab => {
                  const noUrls = !tab.urlsText || tab.urlsText.trim().length === 0
                  const noResults = !tab.results || tab.results.length === 0
                  const notScanning = !tab.scanning
                  const noSourceTaskId = !tab.sourceTaskId
                  return noUrls && noResults && notScanning && noSourceTaskId
                }) || null

                if (checkTab) {
                  checkTab.sourceTaskId = sourceTaskId
                  checkTab.sourceTaskType = currentPending.payload?.sourceTaskType || 'hostscan'
                }
              }
            }

            if (checkTab) {
              // 如果已存在标签页，追加URL
              const existingUrls = checkTab.urlsText.split('\n').filter(url => url.trim())
              const newUrls = currentPending.targets.filter(url => !existingUrls.includes(url.trim()))
              if (newUrls.length > 0) {
                checkTab.urlsText = [...existingUrls, ...newUrls].join('\n')
              }
              this.activeTabId = checkTab.id
              this.applyExternalScanConfigToTab(currentPending.payload, checkTab)
              if (!checkTab.scanning) {
                this.$nextTick(() => {
                  if (checkTab.selectedDicts.length === 0 && !checkTab.config.customDict.trim() && !checkTab.config.fuzz.enabled) {
                    checkTab.selectedDicts = ['dicc.txt']
                  }
                  this.startScan(checkTab, true)
                })
              }
            } else {
              // 如果不存在标签页，才创建新标签页
              this._processDirscanFill(
                currentPending.payload,
                currentPending.targets,
                sourceTaskId
              )
            }
          }
          delete this.dirscanFillDebounceMap[debounceKey]
          delete this.dirscanFillPendingTargets[debounceKey]
        }, 200)

        return
      }

      // 如果组件已挂载且已激活，直接处理
      if (window.__dirsearchReady && window.__dirsearchActivated) {
        this._processDirscanFill(payload, targets, sourceTaskId)
        return
      }

      // 如果组件已挂载但未激活，将事件暂存到队列，等组件激活后再处理
      if (window.__dirsearchReady && !window.__dirsearchActivated) {
        if (!Array.isArray(window.__cyberSabrePendingDirscan)) {
          window.__cyberSabrePendingDirscan = []
        }
        window.__cyberSabrePendingDirscan.push(payload)
        return
      }

      // 如果组件未挂载，事件会被代理暂存，这里不需要处理
      // 组件挂载后会在 mounted() 中处理暂存的事件
    },

    _processDirscanFill(payload, targets, sourceTaskId) {
      // 检查是否有其他标签正在扫描
      const hasScanningTab = this.scanTabs.some(tab => tab.scanning && tab.id !== this.activeTabId)

      // 如果提供了 sourceTaskId，查找是否已存在相同来源任务的标签页
      // 这里需要再次检查，因为可能在调用此方法之前，已经有其他请求创建了标签
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

      // 如果没有找到相同 sourceTaskId 的标签页，尝试复用默认标签页（没有 sourceTaskId 的标签页）
      if (!targetTab && sourceTaskId) {
        // 优先复用默认标签页（没有关联来源任务的标签页）
        if (this.scanTabs.length === 1) {
          const onlyTab = this.scanTabs[0]
          // 没有关联来源任务视为"默认页"，可以被联动覆盖使用
          if (!onlyTab.sourceTaskId) {
            // 如果默认页里有旧结果/状态，这里主动清空，专门留给联动使用
            onlyTab.urlsText = ''
            onlyTab.results = []
            onlyTab.scanning = false
            onlyTab.completed = false
            onlyTab.scanProgress = 0
            onlyTab.stats = {
              totalRequests: 0,
              success: 0,
              failed: 0,
              found: 0,
              requestsPerSecond: 0
            }
            this.resetColumnFilters(onlyTab)
            targetTab = onlyTab
            // 设置来源任务信息
            targetTab.sourceTaskId = sourceTaskId
            targetTab.sourceTaskType = payload?.sourceTaskType || 'hostscan'
          }
        }

        // 如果仍未匹配到标签，再尝试复用一个"空闲标签页"（无URL、无结果、未在扫描）
        if (!targetTab) {
          targetTab = this.scanTabs.find(tab => {
            const noUrls = !tab.urlsText || tab.urlsText.trim().length === 0
            const noResults = !tab.results || tab.results.length === 0
            const notScanning = !tab.scanning
            const noSourceTaskId = !tab.sourceTaskId
            return noUrls && noResults && notScanning && noSourceTaskId
          }) || null

          if (targetTab && sourceTaskId) {
            targetTab.sourceTaskId = sourceTaskId
            targetTab.sourceTaskType = payload?.sourceTaskType || 'hostscan'
          }
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
        // 应用配置
        this.applyExternalScanConfigToTab(payload, targetTab)
        // 如果标签页未在扫描中，且没有其他标签正在扫描，自动开始扫描（联动时跳过确认，保留数据）
        if (!targetTab.scanning && !hasScanningTab) {
          // 立即设置扫描状态和加载状态，确保 UI 立即更新
          targetTab.scanning = true
          targetTab.loading = true
          // 设置扫描开始时间
          if (!targetTab.scanStartTime) {
            targetTab.scanStartTime = Date.now()
          }
          // 确保配置已应用后再启动扫描
          this.$nextTick(() => {
            // 再次检查配置是否已正确应用（特别是字典配置）
            if (targetTab.selectedDicts.length === 0 && !targetTab.config.customDict.trim() && !targetTab.config.fuzz.enabled) {
              // 如果字典未设置，使用默认字典
              targetTab.selectedDicts = ['dicc.txt']
            }
            this.startScan(targetTab, true) // 传入true表示联动扫描，跳过确认并保留数据
          })
        }
      } else {
        // 联动时自动创建新标签页
        const newTab = this.createNewTab()
        newTab.name = `目录扫描-联动-${new Date().toLocaleTimeString()}`

        // 记录来源任务ID（用于任务管理中的关联显示）
        if (sourceTaskId) {
          newTab.sourceTaskId = sourceTaskId
          newTab.sourceTaskType = payload?.sourceTaskType || 'hostscan' // 来源任务类型
        }

        // 切换到新标签页
        this.activeTabId = newTab.id

        // 先填充URL（确保新标签页有URL）
        if (targets.length > 0) {
          newTab.urlsText = targets.join('\n')
        }

        // 应用配置
        this.applyExternalScanConfigToTab(payload, newTab)

        // 自动联动时，只有在没有其他标签正在扫描时才自动开始扫描
        if (!hasScanningTab) {
          // 立即设置扫描状态和加载状态，确保 UI 立即更新
          newTab.scanning = true
          newTab.loading = true
          // 设置扫描开始时间
          newTab.scanStartTime = Date.now()
          // 确保配置已应用后再启动扫描
          this.$nextTick(() => {
            // 再次检查配置是否已正确应用（特别是字典配置）
            if (newTab.selectedDicts.length === 0 && !newTab.config.customDict.trim() && !newTab.config.fuzz.enabled) {
              // 如果字典未设置，使用默认字典
              newTab.selectedDicts = ['dicc.txt']
            }
            this.startScan(newTab, true) // 传入true表示联动扫描，跳过确认并保留数据
          })
        }
      }
    },

    applyExternalScanConfigToTab(payload, tab) {
      if (!tab) {
        tab = this.currentTab || this.scanTabs[0]
      }
      if (!tab) return

      const currentConfig = tab.config
      const currentFilter = currentConfig.filter

      // 支持两种格式：直接格式和嵌套config格式
      let configData = payload || {}
      if (payload?.config && typeof payload.config === 'object') {
        // 如果payload中有config字段，则合并targets和config
        configData = {
          targets: payload.targets || [],
          ...payload.config
        }
      }

      const {
        targets = [],
        method,
        threads,
        timeout,
        recursionDepth,
        followRedirects,
        customDict,
        customHeaders,
        builtinDicts,
        extensions,
        fuzz,
        filter = {}
      } = configData

      if (targets.length > 0) {
        // 如果是新创建的标签页（urlsText为空或只有空白字符），直接设置targets
        // 如果已有URL，则追加新的URL（去重）
        const currentUrls = (tab.urlsText || '').split('\n').filter(url => url.trim())
        const newUrls = targets.filter(url => !currentUrls.includes(url.trim()))
        if (newUrls.length > 0) {
          if (currentUrls.length > 0) {
            tab.urlsText = [...currentUrls, ...newUrls].join('\n')
          } else {
            tab.urlsText = targets.join('\n')
          }
        }
      }

      if (Array.isArray(builtinDicts) && builtinDicts.length > 0) {
        tab.selectedDicts = [...builtinDicts]
      } else if (tab.selectedDicts.length === 0) {
        tab.selectedDicts = ['dicc.txt']
      }

      // 合并状态码：如果外部传入状态码，则合并到现有配置中，而不是覆盖
      let statusCodesSource = currentFilter.statusCodes
      if (filter.statusCodes !== undefined && Array.isArray(filter.statusCodes) && filter.statusCodes.length > 0) {
        // 合并外部传入的状态码和现有状态码，去重
        const externalCodes = this.formatStatusCodesForForm(filter.statusCodes, [])
        const existingCodes = this.formatStatusCodesForForm(currentFilter.statusCodes, [])
        statusCodesSource = [...new Set([...existingCodes, ...externalCodes])]
      }

      const statusCodes = this.formatStatusCodesForForm(statusCodesSource, currentFilter.statusCodes)

      const keywordsText =
        filter.keywords !== undefined
          ? (Array.isArray(filter.keywords) ? filter.keywords.join('\n') : String(filter.keywords || ''))
          : currentFilter.keywords

      const sizeRange =
        Array.isArray(filter.sizeRange) && filter.sizeRange.length === 2
          ? [...filter.sizeRange]
          : [...currentFilter.sizeRange]

      const headersText =
        customHeaders !== undefined
          ? this.headersToText(customHeaders)
          : currentConfig.customHeaders

      // 处理扩展名配置
      const extensionsText =
        extensions !== undefined
          ? Array.isArray(extensions)
            ? extensions.join(',')
            : String(extensions || '')
          : currentConfig.extensions || ''

      tab.config = {
        ...currentConfig,
        method: method || currentConfig.method,
        threads: threads ?? currentConfig.threads,
        timeout: timeout ?? currentConfig.timeout,
        recursionDepth: recursionDepth ?? currentConfig.recursionDepth,
        followRedirects: followRedirects ?? currentConfig.followRedirects,
        customDict: customDict ?? currentConfig.customDict,
        customHeaders: headersText,
        extensions: extensionsText,
        fuzz: {
          ...currentConfig.fuzz,
          ...(fuzz || {})
        },
        filter: {
          ...currentFilter,
          statusCodes,
          sizeRange,
          keywords: keywordsText,
          regex: filter.regex ?? currentFilter.regex
        }
      }
    },
    applyExternalScanConfig(payload) {
      const tab = this.currentTab || this.scanTabs[0]
      if (!tab) return

      const currentConfig = tab.config
      const currentFilter = currentConfig.filter

      // 支持两种格式：直接格式和嵌套config格式
      let configData = payload || {}
      if (payload?.config && typeof payload.config === 'object') {
        // 如果payload中有config字段，则合并targets和config
        configData = {
          targets: payload.targets || [],
          ...payload.config
        }
      }

      const {
        targets = [],
        method,
        threads,
        timeout,
        recursionDepth,
        followRedirects,
        customDict,
        customHeaders,
        builtinDicts,
        extensions,
        fuzz,
        filter = {}
      } = configData

      if (targets.length > 0) {
        // 追加URL而不是覆盖，避免删除原有的URL
        const existingUrls = tab.urlsText
          .split('\n')
          .map(u => u.trim())
          .filter(u => u.length > 0)
        const newUrls = targets.filter(url => !existingUrls.includes(url))
        if (newUrls.length > 0) {
          if (existingUrls.length > 0) {
            tab.urlsText = existingUrls.join('\n') + '\n' + newUrls.join('\n')
          } else {
            tab.urlsText = newUrls.join('\n')
          }
        }
      }

      if (Array.isArray(builtinDicts) && builtinDicts.length > 0) {
        tab.selectedDicts = [...builtinDicts]
      } else if (tab.selectedDicts.length === 0) {
        tab.selectedDicts = ['dicc.txt']
      }

      // 合并状态码：如果外部传入状态码，则合并到现有配置中，而不是覆盖
      let statusCodesSource = currentFilter.statusCodes
      if (filter.statusCodes !== undefined && Array.isArray(filter.statusCodes) && filter.statusCodes.length > 0) {
        // 合并外部传入的状态码和现有状态码，去重
        const externalCodes = this.formatStatusCodesForForm(filter.statusCodes, [])
        const existingCodes = this.formatStatusCodesForForm(currentFilter.statusCodes, [])
        statusCodesSource = [...new Set([...existingCodes, ...externalCodes])]
      }

      const statusCodes = this.formatStatusCodesForForm(statusCodesSource, currentFilter.statusCodes)

      const keywordsText =
        filter.keywords !== undefined
          ? (Array.isArray(filter.keywords) ? filter.keywords.join('\n') : String(filter.keywords || ''))
          : currentFilter.keywords

      const sizeRange =
        Array.isArray(filter.sizeRange) && filter.sizeRange.length === 2
          ? [...filter.sizeRange]
          : [...currentFilter.sizeRange]

      const headersText =
        customHeaders !== undefined
          ? this.headersToText(customHeaders)
          : currentConfig.customHeaders

      // 处理扩展名配置
      const extensionsText =
        extensions !== undefined
          ? Array.isArray(extensions)
            ? extensions.join(',')
            : String(extensions || '')
          : currentConfig.extensions || ''

      tab.config = {
        ...currentConfig,
        method: method || currentConfig.method,
        threads: threads ?? currentConfig.threads,
        timeout: timeout ?? currentConfig.timeout,
        recursionDepth: recursionDepth ?? currentConfig.recursionDepth,
        followRedirects: followRedirects ?? currentConfig.followRedirects,
        customDict: customDict ?? currentConfig.customDict,
        customHeaders: headersText,
        extensions: extensionsText,
        fuzz: {
          ...currentConfig.fuzz,
          ...(fuzz || {})
        },
        filter: {
          ...currentFilter,
          statusCodes,
          sizeRange,
          keywords: keywordsText,
          regex: filter.regex ?? currentFilter.regex
        }
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

    bindWailsEvents() {
      EventsOn('dirsearchResult', this.handleScanResult)
      EventsOn('dirsearchProgress', this.handleScanProgress)
      EventsOn('dirsearchComplete', this.handleScanComplete)
      // EventsOn('dirsearchError', this.handleScanError)
    },

    unbindWailsEvents() {
      EventsOff('dirsearchResult')
      EventsOff('dirsearchProgress')
      EventsOff('dirsearchComplete')
      EventsOff('dirsearchError')
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

      // 检查任务状态：如果任务存在且状态为 running，说明扫描正在进行中
      let isTaskRunning = false
      if (taskId && this.tasksLocal[taskId]) {
        isTaskRunning = this.tasksLocal[taskId].status === 'running'
      }

      // 如果标签存在，检查是否已停止（但如果任务状态为 running，说明扫描正在进行，应该更新状态）
      if (tab && tab.isStopped) {
        return
      }

      // 如果标签存在但 scanning 为 false，但任务状态为 running，说明状态不同步，需要更新
      if (tab && !tab.scanning && isTaskRunning) {
        tab.scanning = true
        tab.loading = false
        // 如果还没有扫描开始时间，设置一个
        if (!tab.scanStartTime) {
          tab.scanStartTime = Date.now()
        }
        this.startTimeCounter(tab)
      }

      // 如果标签存在且已停止，或扫描未开始，则不处理结果（除非任务状态为 running）
      if (tab && !tab.scanning && !isTaskRunning) {
        return
      }

      // 如果标签存在，使用批量更新机制优化性能
      if (tab) {
        if (!tab.scanning && isTaskRunning) {
          tab.scanning = true
          tab.loading = false
          if (!tab.scanStartTime) {
            tab.scanStartTime = Date.now()
          }
          this.startTimeCounter(tab)
        }

        // 初始化批量更新缓冲区
        if (!tab.pendingResults) {
          tab.pendingResults = []
        }
        if (!tab.pendingStats) {
          tab.pendingStats = {
            found: 0,
            totalRequests: 0,
            success: 0,
            failed: 0
          }
        }

        // 将结果添加到缓冲区
        tab.pendingResults.push(result)
        tab.pendingStats.found++
        tab.pendingStats.totalRequests++

        if (result.status >= 200 && result.status < 400) {
          tab.pendingStats.success++
        } else {
          tab.pendingStats.failed++
        }

        // 使用requestAnimationFrame批量更新，减少DOM操作频率
        if (!tab.updateResultsTimer) {
          tab.updateResultsTimer = requestAnimationFrame(() => {
            this.flushPendingResults(tab)
            tab.updateResultsTimer = null
          })
        }
      }

      // 同步更新任务数据（无论标签是否存在）- 使用批量更新优化
      if (taskId && this.tasksLocal[taskId]) {
        // 初始化任务结果数组（如果不存在）
        if (!this.tasksLocal[taskId].results) {
          this.tasksLocal[taskId].results = []
        }

        // 初始化任务批量更新缓冲区
        if (!this.tasksLocal[taskId].pendingResults) {
          this.tasksLocal[taskId].pendingResults = []
        }

        // 将结果添加到任务缓冲区（避免深拷贝，直接引用）
        this.tasksLocal[taskId].pendingResults.push(result)

        // 使用防抖批量更新任务数据
        if (!this.tasksLocal[taskId].updateTaskTimer) {
          this.tasksLocal[taskId].updateTaskTimer = setTimeout(() => {
            this.flushPendingTaskResults(taskId, tab)
            this.tasksLocal[taskId].updateTaskTimer = null
          }, 500) // 500ms批量更新一次任务数据
        }
      }

      // 使用防抖保存历史记录，减少保存频率
      if (tab) {
        if (!tab.saveHistoryTimer) {
          tab.saveHistoryTimer = setTimeout(() => {
            this.saveHistory()
            tab.saveHistoryTimer = null
          }, 5000) // 5秒防抖，减少保存频率
        }
      }
    },

    // 批量刷新标签页的待处理结果
    flushPendingResults(tab) {
      if (!tab || !tab.pendingResults || tab.pendingResults.length === 0) {
        return
      }

      // 批量添加结果，减少Vue响应式更新次数
      const batchSize = Math.min(tab.pendingResults.length, 100) // 每次最多添加100条
      const batch = tab.pendingResults.splice(0, batchSize)
      tab.results.push(...batch)

      // 更新统计信息
      if (tab.pendingStats) {
        tab.stats.found += tab.pendingStats.found
        tab.stats.totalRequests += tab.pendingStats.totalRequests
        tab.stats.success += tab.pendingStats.success
        tab.stats.failed += tab.pendingStats.failed

        // 重置待处理统计
        tab.pendingStats = {
          found: 0,
          totalRequests: 0,
          success: 0,
          failed: 0
        }
      }

      // 更新速度统计
      if (tab.scanStartTime) {
        const elapsedSeconds = (Date.now() - tab.scanStartTime) / 1000
        if (elapsedSeconds > 0) {
          tab.stats.requestsPerSecond = Math.round(tab.stats.totalRequests / elapsedSeconds)
        }
      }

      // 如果还有待处理的结果，继续处理
      if (tab.pendingResults.length > 0) {
        tab.updateResultsTimer = requestAnimationFrame(() => {
          this.flushPendingResults(tab)
          tab.updateResultsTimer = null
        })
      }
    },

    // 批量刷新任务的待处理结果
    flushPendingTaskResults(taskId, tab) {
      if (!taskId || !this.tasksLocal[taskId] || !this.tasksLocal[taskId].pendingResults) {
        return
      }

      const task = this.tasksLocal[taskId]
      if (task.pendingResults.length === 0) {
        return
      }

      // 批量添加结果（深拷贝）
      const batch = task.pendingResults.splice(0, 100) // 每次最多处理100条
      const clonedBatch = batch.map(r => JSON.parse(JSON.stringify(r)))
      task.results.push(...clonedBatch)
      task.resultsCount = task.results.length

      // 更新统计信息
      if (tab) {
        task.completed = tab.stats.totalRequests
        task.total = tab.stats.totalRequests
        task.stats = JSON.parse(JSON.stringify(tab.stats))

        // 同步标签页的基本信息到任务
        if (tab.name) {
          task.name = tab.name
        }
        if (tab.urlsText) {
          task.urlsText = tab.urlsText
        }
        if (tab.config) {
          task.config = JSON.parse(JSON.stringify(tab.config))
        }
      } else {
        // 如果标签不存在，从任务数据中累加
        task.completed = (task.completed || 0) + batch.length
        task.total = (task.total || 0) + batch.length
      }

      task.updatedAt = Date.now()

      // 如果还有待处理的结果，继续处理
      if (task.pendingResults.length > 0) {
        task.updateTaskTimer = setTimeout(() => {
          this.flushPendingTaskResults(taskId, tab)
          task.updateTaskTimer = null
        }, 500)
      } else {
        // 所有结果处理完成，保存任务
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

      // 检查任务状态：如果任务存在且状态为 running，说明扫描正在进行中
      let isTaskRunning = false
      if (taskId && this.tasksLocal[taskId]) {
        isTaskRunning = this.tasksLocal[taskId].status === 'running'
      }

      // 如果标签存在，检查是否已停止（但如果任务状态为 running，说明扫描正在进行，应该更新状态）
      if (tab && tab.isStopped) {
        return
      }

      // 如果标签存在但 scanning 为 false，但任务状态为 running，说明状态不同步，需要更新
      if (tab && !tab.scanning && isTaskRunning) {
        tab.scanning = true
        tab.loading = false
        // 如果还没有扫描开始时间，设置一个
        if (!tab.scanStartTime) {
          tab.scanStartTime = Date.now()
        }
        this.startTimeCounter(tab)
      }

      // 如果标签存在且已停止，或扫描未开始，则不处理进度（除非任务状态为 running）
      if (tab && !tab.scanning && !isTaskRunning) {
        return
      }

      // 如果标签存在，更新标签数据
      if (tab) {
        if (!tab.scanning && isTaskRunning) {
          tab.scanning = true
          tab.loading = false
          if (!tab.scanStartTime) {
            tab.scanStartTime = Date.now()
          }
          this.startTimeCounter(tab)
        }
        tab.scanProgress = progress.percentage || 0
        if (progress.stats) {
          if (tab.baselineStats.found > 0) {
            tab.stats.requestsPerSecond = progress.stats.requestsPerSecond || tab.stats.requestsPerSecond
          } else {
            tab.stats = { ...tab.stats, ...progress.stats }
          }
        }
      }

      // 更新任务状态（无论标签是否存在）
      if (taskId && this.tasksLocal[taskId]) {
        if (progress?.percentage !== undefined) {
          this.tasksLocal[taskId].progress = progress.percentage
        }

        // 如果标签存在，保存完整结果
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
        }

        if (progress.stats) {
          this.tasksLocal[taskId].completed = progress.stats.totalRequests || (tab ? tab.stats.totalRequests : 0)
          this.tasksLocal[taskId].total = progress.stats.totalRequests || (tab ? tab.stats.totalRequests : 0)
        } else if (tab) {
          this.tasksLocal[taskId].completed = tab.stats.totalRequests
          this.tasksLocal[taskId].total = tab.stats.totalRequests
        }

        this.tasksLocal[taskId].updatedAt = Date.now()
        this.saveTasks()
      }
    },

    handleScanComplete(payload) {
      // 根据后端携带的 tabId 精确找到对应的标签页
      const tabId = payload?.tabId
      const tab = tabId ? this.scanTabs.find(t => t.id === tabId) : null

      // 如果标签页存在，更新标签页状态
      if (tab) {
        // 如果已停止扫描，忽略完成事件
        if (tab.isStopped) {
          return
        }

        // 清理批量更新定时器
        if (tab.updateResultsTimer) {
          cancelAnimationFrame(tab.updateResultsTimer)
          tab.updateResultsTimer = null
        }

        // 处理剩余待更新的结果
        if (tab.pendingResults && tab.pendingResults.length > 0) {
          this.flushPendingResults(tab)
        }

        // 清理历史保存定时器并立即保存
        if (tab.saveHistoryTimer) {
          clearTimeout(tab.saveHistoryTimer)
          tab.saveHistoryTimer = null
        }
        tab.scanning = false
        tab.loading = false
        tab.completed = true
        tab.scanProgress = 100
        tab.isStopped = false // 重置停止标记
        this.stopTimeCounter(tab)
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
        const task = this.tasksLocal[taskId]

        // 清理任务的批量更新定时器
        if (task.updateTaskTimer) {
          clearTimeout(task.updateTaskTimer)
          task.updateTaskTimer = null
        }

        // 处理剩余待更新的任务结果
        if (task.pendingResults && task.pendingResults.length > 0) {
          this.flushPendingTaskResults(taskId, tab)
        }

        task.status = 'completed'
        task.progress = 100

        // 如果标签页存在，使用标签页的数据；否则使用 payload 中的数据
        if (tab) {
          task.results = JSON.parse(JSON.stringify(tab.results))
          task.resultsCount = tab.results.length
          task.completed = tab.stats.totalRequests
          task.total = tab.stats.totalRequests
          task.stats = JSON.parse(JSON.stringify(tab.stats))

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
          this.tasksLocal[taskId].completed = payload.stats?.totalRequests || 0
          this.tasksLocal[taskId].total = payload.stats?.totalRequests || 0
          if (payload.stats) {
            this.tasksLocal[taskId].stats = JSON.parse(JSON.stringify(payload.stats))
          }
        }

        this.tasksLocal[taskId].updatedAt = Date.now()
        this.saveTasks()
      }

      if (tab) {
        this.saveHistory()
        this.$message.success({
          message: '扫描完成',
          duration: 2000
        })
      }
    },

    handleScanError(error) {
      this.$message.error({
        message: `扫描错误: ${error.message || error}`,
        duration: 2000
      })
      // 根据tabId找到对应的标签页并更新状态
      const tabId = error?.tabId || this.activeTabId
      const tab = this.scanTabs.find(t => t.id === tabId)
      if (tab) {
        tab.scanning = false
        tab.loading = false
      }
    },

    async toggleScan(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) {
        this.$message.warning('请先创建或选择一个任务')
        return
      }
      if (tab.scanning) {
        await this.stopScan(tab)
      } else {
        await this.startScan(tab)
      }
    },

    async startScan(tab, skipConfirm = false) {
      if (!tab) tab = this.currentTab
      if (!tab) {
        this.$message.warning('请先创建或选择一个任务')
        return
      }
      this.flushPendingTextareas(tab)
      // 解析URL列表
      const urls = tab.urlsText
        .split('\n')
        .map(url => url.trim())
        .filter(url => url && url.length > 0)

      if (urls.length === 0) {
        this.$message.warning('请输入至少一个目标URL')
        // 重置状态，因为验证失败
        tab.scanning = false
        tab.loading = false
        return
      }

      // 验证至少选择了字典或启用了Fuzz
      if (tab.selectedDicts.length === 0 && !tab.config.customDict.trim() && !tab.config.fuzz.enabled) {
        this.$message.warning('请至少选择一个字典文件、输入自定义字典或启用Fuzz')
        // 重置状态，因为验证失败
        tab.scanning = false
        tab.loading = false
        return
      }

      // 检查是否存在扫描记录（联动扫描时跳过确认，自动保留数据）
      let shouldClearResults = false
      let shouldKeepResults = false

      if (skipConfirm && tab.results.length > 0) {
        // 联动扫描时，自动保留数据
        shouldKeepResults = true
        shouldClearResults = false
      } else if (!skipConfirm && tab.results.length > 0) {
        // 手动扫描时才提示
        try {
          await this.$confirm(
            `当前已有 ${tab.results.length} 条扫描记录。您希望如何处理？`,
            '扫描记录提示',
            {
              confirmButtonText: '保存记录继续扫描',
              cancelButtonText: '删除记录重新扫描',
              type: 'warning',
              distinguishCancelAndClose: true,
              showCancelButton: true,
              showCloseButton: false,
              closeOnClickModal: false
            }
          )
          shouldKeepResults = true
        } catch (action) {
          if (action === 'cancel') {
            try {
              await this.$confirm(
                `将丢弃当前 ${tab.results.length} 条扫描记录并重新开始扫描。`,
                '删除记录重新扫描',
                { type: 'warning', confirmButtonText: '继续扫描' }
              )
              shouldClearResults = true
            } catch {
              tab.scanning = false
              tab.loading = false
              return
            }
          } else {
            tab.scanning = false
            tab.loading = false
            return
          }
        }
      }

      // 调用内部扫描函数
      await this.startScanInternal(tab, skipConfirm, shouldClearResults, shouldKeepResults)
    },

    async startScanInternal(tab, skipConfirm = false, shouldClearResults = false, shouldKeepResults = false) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      this.flushPendingTextareas(tab)
      // 解析URL列表
      const urls = tab.urlsText
        .split('\n')
        .map(url => url.trim())
        .filter(url => url && url.length > 0)

      if (urls.length === 0) {
        this.$message.warning('请输入至少一个目标URL')
        // 重置状态，因为验证失败
        tab.scanning = false
        tab.loading = false
        return
      }

      // 验证至少选择了字典或启用了Fuzz
      if (tab.selectedDicts.length === 0 && !tab.config.customDict.trim() && !tab.config.fuzz.enabled) {
        this.$message.warning('请至少选择一个字典文件、输入自定义字典或启用Fuzz')
        // 重置状态，因为验证失败
        tab.scanning = false
        tab.loading = false
        return
      }

      // 如果是自动联动（skipConfirm=true），默认保存记录并继续扫描
      if (skipConfirm && tab.results.length > 0) {
        shouldKeepResults = true
        shouldClearResults = false
      }

      try {
        // 重置停止标记
        tab.isStopped = false
        tab.scanning = true
        tab.loading = true
        tab.scanProgress = 0

        if (shouldClearResults) {
          tab.results = []
          this.resetStats(tab)
          this.resetBaselineStats(tab)
          tab.currentPage = 1
        } else if (shouldKeepResults) {
          this.calculateStatsFromResults(tab)
          tab.baselineStats = {
            totalRequests: tab.stats.totalRequests,
            success: tab.stats.success,
            failed: tab.stats.failed,
            found: tab.stats.found
          }
        } else {
          tab.results = []
          this.resetStats(tab)
          this.resetBaselineStats(tab)
        }

        tab.scanStartTime = Date.now()

        // 处理 keywords：将字符串转换为数组
        const keywordsArray = tab.config.filter.keywords
          ? tab.config.filter.keywords
            .split('\n')
            .map(k => k.trim())
            .filter(k => k.length > 0)
          : []

        const parsedStatusCodes = this.parseStatusCodes(tab.config.filter.statusCodes)

        // 解析扩展名列表
        const extensionsArray = tab.config.extensions
          ? tab.config.extensions
            .split(',')
            .map(ext => ext.trim())
            .filter(ext => ext.length > 0)
          : []

        const scanConfig = {
          targets: urls,
          method: tab.config.method,
          threads: tab.config.threads,
          timeout: tab.config.timeout,
          recursionDepth: tab.config.recursionDepth,
          followRedirects: tab.config.followRedirects,
          customHeaders: this.parseHeaders(tab.config.customHeaders),
          builtinDicts: tab.selectedDicts,
          customDict: tab.config.customDict,
          extensions: extensionsArray,
          fuzz: tab.config.fuzz,
          filter: {
            ...tab.config.filter,
            statusCodes: parsedStatusCodes,
            keywords: keywordsArray
          },
          tabId: tab.id
        }

        // 生成任务ID
        const taskId = `dirscan-${Date.now()}-${++this.taskIdCounter}`
        tab.taskId = taskId

        // 创建任务记录
        this.tasksLocal[taskId] = {
          taskId,
          name: tab.name || `目录扫描-${new Date().toLocaleTimeString()}`,
          urlsText: tab.urlsText,
          config: JSON.parse(JSON.stringify(scanConfig)),
          results: JSON.parse(JSON.stringify(tab.results)), // 保存完整的结果数组
          resultsCount: tab.results.length, // 保存结果数量用于显示
          status: 'running',
          progress: 0,
          completed: 0,
          total: 0,
          createdAt: Date.now(),
          updatedAt: Date.now(),
          // 关联任务信息
          sourceTaskId: tab.sourceTaskId || null,
          sourceTaskType: tab.sourceTaskType || null
        }
        this.tasks = Object.values(this.tasksLocal)
        this.saveTasks()

        // 调用后端API
        await StartDirectoryScan(JSON.stringify(scanConfig))

        tab.loading = false
        this.saveHistory()
        this.$message.success({
          message: '扫描已启动',
          duration: 2000
        })
      } catch (error) {
        const errorMsg = error?.message || error?.toString() || '未知错误'
        this.$message.error({
          message: `启动扫描失败: ${errorMsg}`,
          duration: 2000
        })
        tab.scanning = false
        tab.loading = false
      }
    },

    async stopScan(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return

      // 立即更新前端状态，标记为已停止
      tab.isStopped = true
      tab.scanning = false
      tab.loading = false
      // 使用 stopTimeCounter 方法统一清理定时器
      this.stopTimeCounter(tab)

      // 清理批量更新定时器
      if (tab.updateResultsTimer) {
        cancelAnimationFrame(tab.updateResultsTimer)
        tab.updateResultsTimer = null
      }

      // 处理剩余待更新的结果
      if (tab.pendingResults && tab.pendingResults.length > 0) {
        this.flushPendingResults(tab)
      }

      // 清理历史保存定时器
      if (tab.saveHistoryTimer) {
        clearTimeout(tab.saveHistoryTimer)
        tab.saveHistoryTimer = null
      }

      // 清理任务的批量更新定时器
      if (tab.taskId && this.tasksLocal[tab.taskId]) {
        const task = this.tasksLocal[tab.taskId]
        if (task.updateTaskTimer) {
          clearTimeout(task.updateTaskTimer)
          task.updateTaskTimer = null
        }
        // 处理剩余待更新的任务结果
        if (task.pendingResults && task.pendingResults.length > 0) {
          this.flushPendingTaskResults(tab.taskId, tab)
        }
      }

      try {
        await StopDirectoryScan()
        // 更新任务状态
        if (tab.taskId && this.tasksLocal[tab.taskId]) {
          this.tasksLocal[tab.taskId].status = 'stopped'
          this.tasksLocal[tab.taskId].results = JSON.parse(JSON.stringify(tab.results)) // 保存完整的结果数组
          this.tasksLocal[tab.taskId].resultsCount = tab.results.length
          this.tasksLocal[tab.taskId].stats = JSON.parse(JSON.stringify(tab.stats))
          this.tasksLocal[tab.taskId].progress = tab.scanProgress || 0

          // 同步标签页的基本信息到任务（确保任务管理和标签页显示一致）
          if (tab.name) {
            this.tasksLocal[tab.taskId].name = tab.name
          }
          if (tab.urlsText) {
            this.tasksLocal[tab.taskId].urlsText = tab.urlsText
          }
          if (tab.config) {
            this.tasksLocal[tab.taskId].config = JSON.parse(JSON.stringify(tab.config))
          }

          this.saveTasks()
        }
        this.saveHistory()
        this.$message.info({
          message: '扫描已停止',
          duration: 2000
        })
      } catch (error) {
        const errorMsg = error?.message || error?.toString() || '未知错误'
        this.$message.error({
          message: `停止扫描失败: ${errorMsg}`,
          duration: 2000
        })
        // 即使出错也要保持停止状态
      }
    },

    resetStats(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      tab.stats = {
        totalRequests: 0,
        success: 0,
        failed: 0,
        found: 0,
        requestsPerSecond: 0
      }
    },

    calculateStatsFromResults(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      let success = 0
      let failed = 0

      tab.results.forEach(result => {
        if (result.status >= 200 && result.status < 400) {
          success++
        } else {
          failed++
        }
      })

      tab.stats = {
        totalRequests: tab.results.length,
        success: success,
        failed: failed,
        found: tab.results.length,
        requestsPerSecond: 0
      }
    },

    resetBaselineStats(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      tab.baselineStats = {
        totalRequests: 0,
        success: 0,
        failed: 0,
        found: 0
      }
    },

    startTimeCounter(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      // 如果 scanStartTime 未设置，则设置为当前时间
      if (!tab.scanStartTime) {
        tab.scanStartTime = Date.now()
      }
      // 如果已有定时器，先清除
      if (tab.timeInterval) {
        clearInterval(tab.timeInterval)
        tab.timeInterval = null
      }
      this.updateElapsedTime(tab)
      tab.timeInterval = setInterval(() => {
        this.updateElapsedTime(tab)
        this.calcRequestsPerSecond(tab)
      }, 1000)
    },

    stopTimeCounter(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      if (tab.timeInterval) {
        clearInterval(tab.timeInterval)
        tab.timeInterval = null
      }
    },

    updateElapsedTime(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return

      // 如果 scanStartTime 未设置，设置为当前时间
      if (!tab.scanStartTime) {
        tab.scanStartTime = Date.now()
      }

      try {
        const elapsed = Date.now() - tab.scanStartTime
        // 确保 elapsed 不为负数
        if (elapsed < 0) {
          tab.scanStartTime = Date.now()
          tab.elapsedTime = '00:00'
          return
        }

        const minutes = Math.floor(elapsed / 60000)
        const seconds = Math.floor((elapsed % 60000) / 1000)

        tab.elapsedTime = `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
      } catch (error) {
        console.error('更新已用时间失败:', error)
        tab.elapsedTime = '00:00'
      }
    },

    calcRequestsPerSecond(tab) {
      if (!tab) tab = this.currentTab
      if (!tab || !tab.scanStartTime) return

      const elapsedSeconds = (Date.now() - tab.scanStartTime) / 1000
      if (elapsedSeconds > 0) {
        tab.stats.requestsPerSecond = Math.round(tab.stats.totalRequests / elapsedSeconds)
      }
    },

    parseStatusCodes(codes) {
      if (Array.isArray(codes)) {
        return codes
          .map(code => parseInt(code, 10))
          .filter(code => Number.isInteger(code))
      }

      if (typeof codes === 'string') {
        return codes
          .split(/[,\s]+/)
          .map(code => parseInt(code, 10))
          .filter(code => Number.isInteger(code))
      }

      if (typeof codes === 'number') {
        return Number.isInteger(codes) ? [codes] : []
      }

      return []
    },

    formatStatusCodesForForm(input, fallback = []) {
      if (Array.isArray(input)) {
        return input
          .map(code => {
            if (typeof code === 'number') return String(code)
            if (typeof code === 'string') return code.trim()
            return ''
          })
          .filter(code => code.length > 0)
      }

      if (typeof input === 'number') {
        return [String(input)]
      }

      if (typeof input === 'string') {
        return input
          .split(/[,\s]+/)
          .map(code => code.trim())
          .filter(code => code.length > 0)
      }

      return Array.isArray(fallback) ? [...fallback] : []
    },

    parseHeaders(headersText) {
      if (!headersText) return {}

      const headers = {}
      headersText.split('\n').forEach(line => {
        const match = line.match(/^([^:]+):\s*(.+)$/)
        if (match) {
          headers[match[1].trim()] = match[2].trim()
        }
      })

      return headers
    },

    exportToExcel(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      if (tab.results.length === 0) {
        this.$message.warning({
          message: '没有可导出的结果',
          duration: 2000
        })
        return
      }

      try {
        // 准备数据
        const filtered = this.getFilteredResults(tab)
        const data = filtered.map((item, index) => ({
          '序号': index + 1,
          '状态码': item.status,
          'URL': item.url,
          '标题': item.title || '-',
          '大小': this.formatSize(item.size),
          '响应时间(ms)': item.time,
          '发现时间': item.timestamp ? this.formatTime(item.timestamp) : '-'
        }))

        // 创建工作簿
        const wb = XLSX.utils.book_new()
        const ws = XLSX.utils.json_to_sheet(data)

        // 设置列宽
        ws['!cols'] = [
          { wch: 8 },  // 序号
          { wch: 10 }, // 状态码
          { wch: 60 }, // URL
          { wch: 30 }, // 标题
          { wch: 15 }, // 大小
          { wch: 15 }, // 响应时间
          { wch: 20 }  // 发现时间
        ]

        // 添加工作表
        XLSX.utils.book_append_sheet(wb, ws, '扫描结果')

        // 导出文件
        const fileName = `扫描结果_${new Date().getTime()}.xlsx`
        XLSX.writeFile(wb, fileName)

        this.$message.success({
          message: '导出成功',
          duration: 2000
        })
      } catch (error) {
        this.$message.error({
          message: `导出失败: ${error.message}`,
          duration: 2000
        })
      }
    },

    async clearResults(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      try {
        await this.$confirm('确定要清空所有扫描结果吗？', '确认', { type: 'warning' })
      } catch {
        return
      }
      tab.results = []
      this.resetStats(tab)
      this.resetBaselineStats(tab)
      this.resetColumnFilters(tab)
      tab.currentPage = 1
      this.saveHistory()
      this.$message.success({
        message: '已清空扫描结果',
        duration: 2000
      })
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
        return 'success-row'
      } else if (row.status >= 300 && row.status < 400) {
        return 'warning-row'
      }
      return ''
    },

    formatUrl(url) {
      try {
        const urlObj = new URL(url)
        return urlObj.pathname + urlObj.search
      } catch {
        return url
      }
    },

    formatSize(bytes) {
      if (bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    },

    formatTime(timestamp) {
      if (!timestamp) return ''
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

    formatHeaders(headers) {
      if (!headers) return ''

      if (typeof headers === 'object') {
        return Object.entries(headers)
          .map(([key, value]) => `${key}: ${value}`)
          .join('\n')
      }

      return String(headers)
    },

    openUrl(url) {
      BrowserOpenURL(url)
    },

    copyToClipboard(text) {
      navigator.clipboard.writeText(text).then(() => {
        this.$message.success('已复制到剪贴板')
      }).catch(() => {
        this.$message.error('复制失败')
      })
    },

    handleRowClick(row) {
      // 可以添加行点击的其他逻辑
    },

    handleSizeChange(size, tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      tab.pageSize = size
      tab.currentPage = 1
    },

    handleCurrentChange(page, tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      tab.currentPage = page
    },

    loadConfig() {
      const savedConfig = localStorage.getItem('dirScannerConfig')
      if (savedConfig) {
        try {
          Object.assign(this.config, JSON.parse(savedConfig))
        } catch (error) {
          console.error('加载配置失败:', error)
        }
      }
    },

    saveConfig() {
      try {
        localStorage.setItem('dirScannerConfig', JSON.stringify(this.config))
      } catch (error) {
        console.error('保存配置失败:', error)
      }
    },
    handleRowContextMenu(row, column, event, tab) {
      event.preventDefault()
      event.stopPropagation()
      this.selectedRow = row
      this.selectedTab = tab || this.currentTab

      let x = event.clientX || (event.originalEvent && event.originalEvent.clientX) || 0
      let y = event.clientY || (event.originalEvent && event.originalEvent.clientY) || 0

      // 先用点击位置显示菜单
      this.contextMenuPos = { x, y }
      this.contextMenuVisible = true

      // 再基于实际菜单高度做一次轻微修正，防止刚好溢出窗口底部
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
    handleViewPacket() {
      if (!this.selectedRow) return
      if (!this.selectedRow.requestPacket && !this.selectedRow.responsePacket) {
        this.$message.warning('该记录没有数据包信息')
        return
      }
      this.contextMenuVisible = false
      this.packetViewerVisible = true
    },
    async handleRunJsfinder() {
      if (!this.selectedRow || !this.selectedTab) return
      this.contextMenuVisible = false

      const url = this.selectedRow.url
      if (!url || !url.startsWith('http')) {
        this.$message.warning('仅支持对 HTTP/HTTPS URL 进行 JSFinder 接口探测')
        return
      }

      try {
        const config = {
          wait_time: 5,
          extract_rules: [],
          api_routes: ['/api', '/api/v1', '/api/v2', '/rest', '/graphql'],
          global_api_routes: ['/api', '/api/v1', '/api/v2'],
          allow_list: [],
          auto_test_api: true,
          test_methods: ['GET', 'POST'],
          worker_num: 10
        }

        await CreateBrowserScanTask(url, config)

        window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'browser-scan' }))
        this.$message.success(`已触发 JSFinder 接口探测: ${url}`)
      } catch (error) {
        this.$message.error(`JSFinder 接口探测失败: ${error?.message || error}`)
      }
    },
    handleCopyRow() {
      if (!this.selectedRow) return
      const rowText = `URL: ${this.selectedRow.url}\n状态码: ${this.selectedRow.status}\n标题: ${this.selectedRow.title || ''}\n大小: ${this.formatSize(this.selectedRow.size)}\n响应时间: ${this.selectedRow.time}ms`
      navigator.clipboard.writeText(rowText).then(() => {
        this.$message.success('已复制到剪贴板')
      }).catch(() => {
        this.$message.error('复制失败')
      })
      this.contextMenuVisible = false
    },
    handleCopyAll() {
      if (!this.selectedTab) return
      if (this.selectedTab.results.length === 0) {
        this.$message.warning('没有可复制的内容')
        return
      }
      const allText = this.selectedTab.results.map((r, idx) => {
        return `${idx + 1}. ${r.url} - ${r.status} - ${r.title || ''}`
      }).join('\n')
      navigator.clipboard.writeText(allText).then(() => {
        this.$message.success('已复制所有内容到剪贴板')
      }).catch(() => {
        this.$message.error('复制失败')
      })
      this.contextMenuVisible = false
    },
    async handleDeleteRow() {
      if (!this.selectedRow || !this.selectedTab) return
      try {
        await this.$confirm('确定要删除这条记录吗？', '确认删除', { type: 'warning' })
      } catch {
        this.contextMenuVisible = false
        return
      }
      const index = this.selectedTab.results.findIndex(r => r.url === this.selectedRow.url)
      if (index !== -1) {
        this.selectedTab.results.splice(index, 1)
        this.selectedTab.stats.found = this.selectedTab.results.length
        this.saveHistory()
        this.$message.success('已删除')
      }
      this.contextMenuVisible = false
    }
  }
}
</script>

<style lang="scss" scoped>
.dirsearch-container {
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

.url-textarea {
  :deep(.el-textarea__inner) {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 14px;
  }
}

.url-actions {
  display: flex;
  gap: 8px;
}

.config-section-right {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
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

.advanced-options {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
}

.advanced-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;

  .advanced-item {
    display: flex;
    flex-direction: column;
    gap: 6px;

    &.full-width {
      grid-column: 1 / -1;
    }
  }
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
  padding: 6px;
  border-bottom: 1px solid #e4e7ed;
  flex-wrap: wrap;
  gap: 12px;
}

.results-title-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.results-title {
  font-weight: 600;
  color: #303133;
  font-size: 14px;
}

.results-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.table-wrapper {
  flex: 1;
  min-height: 120px;
  overflow: hidden;
  padding: 12px;
}

.results-table {
  width: 100%;
  height: 100%;

  .result-col-header {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    width: 100%;
    min-width: 0;
  }

  .result-col-header--right {
    justify-content: flex-end;
  }

  .result-col-header__title {
    cursor: pointer;
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    border-bottom: 1px dashed transparent;

    &:hover {
      color: var(--el-color-primary);
      border-bottom-color: var(--el-color-primary-light-5);
    }

    &.has-filter {
      color: var(--el-color-primary);
      font-weight: 600;
    }
  }

  .result-col-header__sort-btns {
    display: inline-flex;
    flex-direction: column;
    gap: 0;
    line-height: 1;
    flex-shrink: 0;
  }

  .result-col-header__sort-ico {
    font-size: 12px;
    cursor: pointer;
    color: var(--el-text-color-secondary);
    transition: color 0.15s;

    &:hover {
      color: var(--el-color-primary);
    }

    &.active {
      color: var(--el-color-primary);
    }
  }

  :deep(.el-table__header-wrapper) {
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
  }

  :deep(.el-table__header th.el-table__cell) {
    vertical-align: middle;
  }

  :deep(.el-table__header .cell) {
    min-height: 34px;
    line-height: 1.35;
  }

  :deep(.success-row) {
    background-color: #f0f9ff;
  }

  :deep(.warning-row) {
    background-color: #fffbf0;
  }
}

.url-cell {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.url-text {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.url-actions {
  display: flex;
  gap: 4px;
  opacity: 0.7;
  transition: opacity 0.3s;

  &:hover {
    opacity: 1;
  }
}

.status-tag {
  min-width: 70px;
  font-weight: 600;
  font-size: 13px;
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

.response-dialog {
  :deep(.el-dialog__body) {
    padding: 0;
  }
}

.response-detail {
  height: 70vh;
  display: flex;
  flex-direction: column;
}

.response-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;

  :deep(.el-tabs__content) {
    flex: 1;
    display: flex;
  }

  :deep(.el-tab-pane) {
    flex: 1;
    display: flex;
    flex-direction: column;
  }
}

.response-content-wrapper,
.headers-wrapper,
.request-info-wrapper {
  flex: 1;
  overflow: auto;
  background: #f8f9fa;
  padding: 20px;
}

.response-content,
.response-headers {
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #303133;
  white-space: pre-wrap;
  word-break: break-all;
}

.request-info-wrapper {
  :deep(.el-descriptions) {
    margin-bottom: 24px;
  }
}

.request-headers {
  h4 {
    margin: 0 0 16px 0;
    color: #303133;
    font-size: 16px;
    font-weight: 600;
  }
}

// 自定义滚动条
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 4px;

  &:hover {
    background: #a8a8a8;
  }
}

// 响应式调整
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

  0%,
  100% {
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

// 任务管理对话框样式
:deep(.history-dialog) {
  --el-dialog-width: 75%;
  --el-dialog-margin-top: 8vh;

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

.task-stats {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;

  .stat-item {
    display: flex;
    align-items: center;
    gap: 4px;

    .stat-label {
      color: #909399;
      font-weight: 500;
    }

    .stat-value {
      color: #303133;
      font-weight: 600;
    }
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
</style>