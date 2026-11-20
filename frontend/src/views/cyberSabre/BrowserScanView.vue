<template>
  <div class="browser-scan-page">
    <!-- 顶部工具栏（与目录扫描等一致） -->
    <div class="top-toolbar">
      <div class="toolbar-content">
        <div class="toolbar-left">
          <div class="module-title">
            <el-icon class="title-icon"><Compass /></el-icon>
            <span class="title-text">JSFinder</span>
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
                      placeholder="例如：https://example.com"
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
                          <label class="config-label">HTTP方法</label>
                          <el-select v-model="task.config.testMethods" multiple placeholder="选择HTTP方法" size="default" style="width: 100%">
                            <el-option label="GET" value="GET" />
                            <el-option label="POST" value="POST" />
                            <el-option label="PUT" value="PUT" />
                            <el-option label="PATCH" value="PATCH" />
                            <el-option label="HEAD" value="HEAD" />
                            <el-option label="OPTIONS" value="OPTIONS" />
                          </el-select>
                        </div>
                        <div class="config-item">
                          <label class="config-label">等待时间(秒)</label>
                          <el-input-number
                            v-model="task.config.waitTime"
                            :min="0"
                            :max="60"
                            controls-position="right"
                            size="default"
                            style="width: 100%"
                          />
                        </div>
                        <div class="config-item">
                          <label class="config-label">并发线程数</label>
                          <el-input-number
                            v-model="task.config.workerNum"
                            :min="1"
                            :max="50"
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
                            {{ task.status || 'pending' }}
                          </el-tag>
                        </div>
                      </div>

                      <div class="config-row" style="display: flex; align-items: center; gap: 10px;">
                        <el-checkbox v-model="task.config.autoTestApi" size="default" style="flex: 1; margin: 0;" @change="(value) => handleAutoTestApiCheckboxChange(task.id, value)">
                          <span>自动对接口进行发包测试（过滤敏感接口配置中的关键词）</span>
                        </el-checkbox>
                        <el-button type="text" size="small" @click="showApiRoutesConfig = true">
                          配置常见API接口
                        </el-button>
                      </div>

                      <div class="config-row" style="display: flex; align-items: center; gap: 10px; margin-top: 10px;">
                        <el-button type="success" size="small" @click="exportToExcel(task.id)">
                          导出结果到XLSX
                        </el-button>
                        <el-button type="info" size="small" @click="showSensitiveConfig = true">
                          配置敏感接口过滤
                        </el-button>
                        <el-button type="warning" size="small" @click="showExtractRulesConfig = true">
                          管理提取规则
                        </el-button>
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
                  <el-tab-pane label="端点与网络" name="endpoints">
                    <el-tabs
                      class="jsfinder-nested-tabs"
                      type="card"
                      :model-value="getResultSubTabEndpoints(task.id)"
                      @update:model-value="(v) => setResultSubTabEndpoints(task.id, v)"
                    >
                  <el-tab-pane label="URLs" name="urls">
                    <el-table :data="getPaginatedData(task.id, 'urls', task.results.urls)" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="URL"
                            v-model="task.colFilters['urls']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'urls')"
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
                      <el-table-column label="操作" width="100">
                        <template #default="scope">
                          <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'urls').page"
                        v-model:page-size="getPaginationConfig(task.id, 'urls').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'urls', task.results.urls)"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'urls', $event)"
                        @current-change="handlePageChange(task.id, 'urls', $event)"
                      />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="API端点" name="api">
                    <el-table :data="getPaginatedData(task.id, 'api', task.results.apiEndpoints)" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="API端点"
                            v-model="task.colFilters['api']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'api')"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="100">
                        <template #default="scope">
                          <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'api').page"
                        v-model:page-size="getPaginationConfig(task.id, 'api').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'api', task.results.apiEndpoints)"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'api', $event)"
                        @current-change="handlePageChange(task.id, 'api', $event)"
                      />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="域名" name="domains">
                    <el-table :data="getPaginatedData(task.id, 'domains', task.results.domains)" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="域名"
                            v-model="task.colFilters['domains']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'domains')"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="100">
                        <template #default="scope">
                          <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'domains').page"
                        v-model:page-size="getPaginationConfig(task.id, 'domains').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'domains', task.results.domains)"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'domains', $event)"
                        @current-change="handlePageChange(task.id, 'domains', $event)"
                      />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="IP地址" name="ips">
                    <el-table :data="getPaginatedData(task.id, 'ips', task.results.ips)" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="IP地址"
                            v-model="task.colFilters['ips']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'ips')"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="100">
                        <template #default="scope">
                          <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'ips').page"
                        v-model:page-size="getPaginationConfig(task.id, 'ips').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'ips', task.results.ips)"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'ips', $event)"
                        @current-change="handlePageChange(task.id, 'ips', $event)"
                      />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="IP端口" name="ipports">
                    <el-table :data="getPaginatedData(task.id, 'ipports', task.results.ipPorts)" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="IP端口"
                            v-model="task.colFilters['ipports']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'ipports')"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="100">
                        <template #default="scope">
                          <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'ipports').page"
                        v-model:page-size="getPaginationConfig(task.id, 'ipports').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'ipports', task.results.ipPorts)"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'ipports', $event)"
                        @current-change="handlePageChange(task.id, 'ipports', $event)"
                      />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="路径" name="paths">
                    <el-table :data="getPaginatedData(task.id, 'paths', task.results.paths)" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="路径"
                            v-model="task.colFilters['paths']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'paths')"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="100">
                        <template #default="scope">
                          <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'paths').page"
                        v-model:page-size="getPaginationConfig(task.id, 'paths').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'paths', task.results.paths)"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'paths', $event)"
                        @current-change="handlePageChange(task.id, 'paths', $event)"
                      />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="不完整路径" name="incompletepaths">
                    <el-table :data="getPaginatedData(task.id, 'incompletepaths', task.results.incompletePaths)" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="不完整路径"
                            v-model="task.colFilters['incompletepaths']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'incompletepaths')"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="100">
                        <template #default="scope">
                          <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'incompletepaths').page"
                        v-model:page-size="getPaginationConfig(task.id, 'incompletepaths').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'incompletepaths', task.results.incompletePaths)"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'incompletepaths', $event)"
                        @current-change="handlePageChange(task.id, 'incompletepaths', $event)"
                      />
                    </div>
                  </el-tab-pane>
                    </el-tabs>
                  </el-tab-pane>

                  <el-tab-pane label="前端与资源" name="frontend">
                    <el-tabs
                      class="jsfinder-nested-tabs"
                      type="card"
                      :model-value="getResultSubTabFrontend(task.id)"
                      @update:model-value="(v) => setResultSubTabFrontend(task.id, v)"
                    >
                  <el-tab-pane label="Vue / Router" name="vue">
                    <div class="vue-scan-block">
                      <el-alert type="info" show-icon :closable="false" class="vue-scan-alert">
                        参考 VueCrack：Headless 页面内注入 detector，检测 Vue、列出路由并处理路由守卫（与扩展同源脚本）。仅反映扫描时刻页面状态。
                      </el-alert>
                      <template v-if="getVueAnalysisForTask(task)">
                        <el-descriptions :column="2" border size="small" class="vue-scan-desc">
                          <el-descriptions-item label="Vue">{{ getVueAnalysisForTask(task).vueDetected ? '是' : '否' }}</el-descriptions-item>
                          <el-descriptions-item label="Vue Router">{{ getVueAnalysisForTask(task).routerDetected ? '是' : '否' }}</el-descriptions-item>
                          <el-descriptions-item label="版本">{{ getVueAnalysisForTask(task).vueVersion || '-' }}</el-descriptions-item>
                          <el-descriptions-item label="Router base">{{ getVueAnalysisForTask(task).routerBase || '-' }}</el-descriptions-item>
                          <el-descriptions-item label="当前 path" :span="2">{{ getVueAnalysisForTask(task).currentPath || '-' }}</el-descriptions-item>
                          <el-descriptions-item v-if="getVueAnalysisForTask(task).pageAnalysis?.detectedBasePath" label="链接推断 base" :span="2">
                            {{ getVueAnalysisForTask(task).pageAnalysis.detectedBasePath }}
                          </el-descriptions-item>
                          <el-descriptions-item v-if="getVueAnalysisForTask(task).error" label="错误" :span="2">
                            <span class="vue-scan-err">{{ getVueAnalysisForTask(task).error }}</span>
                          </el-descriptions-item>
                        </el-descriptions>
                        <div class="vue-scan-actions">
                          <el-button type="primary" size="small" :disabled="!getVueAnalysisForTask(task).allRoutes?.length" @click="copyVuePaths(task)">复制全部路径</el-button>
                          <el-button type="primary" plain size="small" :disabled="!buildVueFullUrlRows(task).length" @click="copyVueFullUrls(task)">复制完整 URL</el-button>
                        </div>
                        <div v-if="getVueAnalysisForTask(task).modifiedRoutes?.length" class="vue-subtitle">已修改 auth 的 meta（{{ getVueAnalysisForTask(task).modifiedRoutes.length }}）</div>
                        <el-table
                          v-if="getVueAnalysisForTask(task).modifiedRoutes?.length"
                          :data="getVueAnalysisForTask(task).modifiedRoutes"
                          border
                          stripe
                          size="small"
                          max-height="220"
                          class="glass-card"
                        >
                          <el-table-column prop="name" label="名称" width="140" show-overflow-tooltip />
                          <el-table-column prop="path" label="路径" show-overflow-tooltip />
                        </el-table>
                        <div class="vue-subtitle">全部路由（{{ (getVueAnalysisForTask(task).allRoutes || []).length }}）</div>
                        <el-table
                          :data="getVueAnalysisForTask(task).allRoutes || []"
                          border
                          stripe
                          size="small"
                          max-height="360"
                          class="glass-card"
                        >
                          <el-table-column prop="name" label="名称" width="140" show-overflow-tooltip />
                          <el-table-column prop="path" label="路径" min-width="160" show-overflow-tooltip />
                          <el-table-column label="meta" min-width="120" show-overflow-tooltip>
                            <template #default="{ row }">{{ formatVueMetaShort(row.meta) }}</template>
                          </el-table-column>
                          <el-table-column label="操作" width="160" fixed="right">
                            <template #default="{ row }">
                              <el-button size="small" @click="copyToClipboard(row.path || '')">复制路径</el-button>
                              <el-button size="small" type="primary" link :disabled="!vueRowFullUrl(task, row)" @click="BrowserOpenURL(vueRowFullUrl(task, row))">打开</el-button>
                            </template>
                          </el-table-column>
                        </el-table>
                      </template>
                      <el-empty v-else description="暂无 Vue 分析数据（任务完成后从后端加载，字段 vue_analysis）" />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="JS文件" name="jsfiles">
                    <el-table :data="getPaginatedData(task.id, 'jsfiles', task.results.jsFiles)" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="JS文件"
                            v-model="task.colFilters['jsfiles']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'jsfiles')"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="100">
                        <template #default="scope">
                          <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'jsfiles').page"
                        v-model:page-size="getPaginationConfig(task.id, 'jsfiles').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'jsfiles', task.results.jsFiles)"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'jsfiles', $event)"
                        @current-change="handlePageChange(task.id, 'jsfiles', $event)"
                      />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane label="静态文件" name="staticfiles">
                    <el-table :data="getPaginatedData(task.id, 'staticfiles', task.results.staticFiles)" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="静态文件"
                            v-model="task.colFilters['staticfiles']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'staticfiles')"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="100">
                        <template #default="scope">
                          <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'staticfiles').page"
                        v-model:page-size="getPaginationConfig(task.id, 'staticfiles').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getColFilteredTotal(task.id, 'staticfiles', task.results.staticFiles)"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'staticfiles', $event)"
                        @current-change="handlePageChange(task.id, 'staticfiles', $event)"
                      />
                    </div>
                  </el-tab-pane>
                    </el-tabs>
                  </el-tab-pane>

                  <!-- <el-tab-pane label="算法" name="algorithms">
                    <el-table :data="getPaginatedData(task.id, 'algorithms', task.results.algorithms)" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="算法"
                            v-model="task.colFilters['algorithms']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'algorithms')"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="100">
                        <template #default="scope">
                          <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <el-pagination small
                      v-model:current-page="getPaginationConfig(task.id, 'algorithms').page"
                      v-model:page-size="getPaginationConfig(task.id, 'algorithms').pageSize"
                      :page-sizes="[10, 20, 50, 100]"
                      :total="getColFilteredTotal(task.id, 'algorithms', task.results.algorithms)"
                      layout="total, sizes, prev, pager, next, jumper"
                      size="small"
                      style="margin-top: 10px; justify-content: flex-end;"
                      @size-change="handlePageSizeChange(task.id, 'algorithms', $event)"
                      @current-change="handlePageChange(task.id, 'algorithms', $event)"
                    />
                  </el-tab-pane> -->

                  <el-tab-pane label="敏感信息" name="secrets">
                    <el-tabs
                      type="border-card"
                      class="nested-tabs"
                      :model-value="getResultSubTabSecrets(task.id)"
                      @update:model-value="(v) => setResultSubTabSecrets(task.id, v)"
                    >
                      <el-tab-pane label="JWT" name="jwt">
                        <el-table :data="getPaginatedData(task.id, 'jwt', task.results.jwt)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="JWT Token"
                            v-model="task.colFilters['jwt']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'jwt')"
                          />
                        </template>
                      </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'jwt').page"
                            v-model:page-size="getPaginationConfig(task.id, 'jwt').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'jwt', task.results.jwt)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'jwt', $event)"
                            @current-change="handlePageChange(task.id, 'jwt', $event)"
                          />
                        </div>
                      </el-tab-pane>
                      <el-tab-pane label="手机号" name="mobile">
                        <el-table :data="getPaginatedData(task.id, 'mobile', task.results.mobiles)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value">
                        <template #header>
                          <TableColumnFilterHeader
                            label="手机号"
                            v-model="task.colFilters['mobile']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'mobile')"
                          />
                        </template>
                      </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'mobile').page"
                            v-model:page-size="getPaginationConfig(task.id, 'mobile').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'mobile', task.results.mobiles)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'mobile', $event)"
                            @current-change="handlePageChange(task.id, 'mobile', $event)"
                          />
                        </div>
                      </el-tab-pane>
                      <el-tab-pane label="邮箱" name="email">
                        <el-table :data="getPaginatedData(task.id, 'email', task.results.emails)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value">
                        <template #header>
                          <TableColumnFilterHeader
                            label="邮箱"
                            v-model="task.colFilters['email']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'email')"
                          />
                        </template>
                      </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'email').page"
                            v-model:page-size="getPaginationConfig(task.id, 'email').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'email', task.results.emails)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'email', $event)"
                            @current-change="handlePageChange(task.id, 'email', $event)"
                          />
                        </div>
                      </el-tab-pane>
                      <el-tab-pane label="SecurityKey" name="securitykey">
                        <el-table :data="getPaginatedData(task.id, 'securitykey', task.results.securityKeys)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="SecurityKey"
                            v-model="task.colFilters['securitykey']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'securitykey')"
                          />
                        </template>
                      </el-table-column>
                          <el-table-column label="操作" width="100">
                            <template #default="scope">
                              <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                            </template>
                          </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'securitykey').page"
                            v-model:page-size="getPaginationConfig(task.id, 'securitykey').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'securitykey', task.results.securityKeys)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'securitykey', $event)"
                            @current-change="handlePageChange(task.id, 'securitykey', $event)"
                          />
                        </div>
                      </el-tab-pane>
                      <el-tab-pane label="Password" name="password">
                        <el-table :data="getPaginatedData(task.id, 'password', task.results.passwords)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="Password"
                            v-model="task.colFilters['password']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'password')"
                          />
                        </template>
                      </el-table-column>
                          <el-table-column label="操作" width="100">
                            <template #default="scope">
                              <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                            </template>
                          </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'password').page"
                            v-model:page-size="getPaginationConfig(task.id, 'password').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'password', task.results.passwords)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'password', $event)"
                            @current-change="handlePageChange(task.id, 'password', $event)"
                          />
                        </div>
                      </el-tab-pane>
                      <el-tab-pane label="Username" name="username">
                        <el-table :data="getPaginatedData(task.id, 'username', task.results.usernames)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="Username"
                            v-model="task.colFilters['username']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'username')"
                          />
                        </template>
                      </el-table-column>
                          <el-table-column label="操作" width="100">
                            <template #default="scope">
                              <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                            </template>
                          </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'username').page"
                            v-model:page-size="getPaginationConfig(task.id, 'username').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'username', task.results.usernames)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'username', $event)"
                            @current-change="handlePageChange(task.id, 'username', $event)"
                          />
                        </div>
                      </el-tab-pane>
                      <el-tab-pane label="AKSK" name="aksk">
                        <el-table :data="getPaginatedData(task.id, 'aksk', task.results.aksk)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="AKSK"
                            v-model="task.colFilters['aksk']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'aksk')"
                          />
                        </template>
                      </el-table-column>
                          <el-table-column label="操作" width="100">
                            <template #default="scope">
                              <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                            </template>
                          </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'aksk').page"
                            v-model:page-size="getPaginationConfig(task.id, 'aksk').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'aksk', task.results.aksk)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'aksk', $event)"
                            @current-change="handlePageChange(task.id, 'aksk', $event)"
                          />
                        </div>
                      </el-tab-pane>
                      <el-tab-pane label="AppID" name="appid">
                        <el-table :data="getPaginatedData(task.id, 'appid', task.results.appIds)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="AppID"
                            v-model="task.colFilters['appid']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'appid')"
                          />
                        </template>
                      </el-table-column>
                          <el-table-column label="操作" width="100">
                            <template #default="scope">
                              <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                            </template>
                          </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'appid').page"
                            v-model:page-size="getPaginationConfig(task.id, 'appid').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'appid', task.results.appIds)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'appid', $event)"
                            @current-change="handlePageChange(task.id, 'appid', $event)"
                          />
                        </div>
                      </el-tab-pane>
                      <el-tab-pane label="AccessKey" name="accesskey">
                        <el-table :data="getPaginatedData(task.id, 'accesskey', task.results.accessKeys)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="AccessKey"
                            v-model="task.colFilters['accesskey']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'accesskey')"
                          />
                        </template>
                      </el-table-column>
                          <el-table-column label="操作" width="100">
                            <template #default="scope">
                              <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                            </template>
                          </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'accesskey').page"
                            v-model:page-size="getPaginationConfig(task.id, 'accesskey').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'accesskey', task.results.accessKeys)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'accesskey', $event)"
                            @current-change="handlePageChange(task.id, 'accesskey', $event)"
                          />
                        </div>
                      </el-tab-pane>
                      <el-tab-pane label="API Key" name="apikey">
                        <el-table :data="getPaginatedData(task.id, 'apikey', task.results.apiKeys)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="API Key"
                            v-model="task.colFilters['apikey']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'apikey')"
                          />
                        </template>
                      </el-table-column>
                          <el-table-column label="操作" width="100">
                            <template #default="scope">
                              <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                            </template>
                          </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'apikey').page"
                            v-model:page-size="getPaginationConfig(task.id, 'apikey').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'apikey', task.results.apiKeys)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'apikey', $event)"
                            @current-change="handlePageChange(task.id, 'apikey', $event)"
                          />
                        </div>
                      </el-tab-pane>
                      <el-tab-pane label="Secret Key" name="secretkey">
                        <el-table :data="getPaginatedData(task.id, 'secretkey', task.results.secretKeys)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="Secret Key"
                            v-model="task.colFilters['secretkey']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'secretkey')"
                          />
                        </template>
                      </el-table-column>
                          <el-table-column label="操作" width="100">
                            <template #default="scope">
                              <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                            </template>
                          </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'secretkey').page"
                            v-model:page-size="getPaginationConfig(task.id, 'secretkey').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'secretkey', task.results.secretKeys)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'secretkey', $event)"
                            @current-change="handlePageChange(task.id, 'secretkey', $event)"
                          />
                        </div>
                      </el-tab-pane>
                      <el-tab-pane label="身份证" name="idcard">
                        <el-table :data="getPaginatedData(task.id, 'idcard', task.results.idCards)" border stripe size="small" max-height="300" class="glass-card">
                      <el-table-column prop="value" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="身份证"
                            v-model="task.colFilters['idcard']"
                            :popover-width="260"
                            @filter-change="handleBrowserColFilter(task.id, 'idcard')"
                          />
                        </template>
                      </el-table-column>
                          <el-table-column label="操作" width="100">
                            <template #default="scope">
                              <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
                            </template>
                          </el-table-column>
                        </el-table>
                        <div class="pagination-wrapper">
                          <el-pagination small
                            v-model:current-page="getPaginationConfig(task.id, 'idcard').page"
                            v-model:page-size="getPaginationConfig(task.id, 'idcard').pageSize"
                            :page-sizes="[10, 20, 50, 100, 200]"
                            :total="getColFilteredTotal(task.id, 'idcard', task.results.idCards)"
                            layout="total, sizes, prev, pager, next, jumper"
                            background
                            @size-change="handlePageSizeChange(task.id, 'idcard', $event)"
                            @current-change="handlePageChange(task.id, 'idcard', $event)"
                          />
                        </div>
                      </el-tab-pane>
                    </el-tabs>
                  </el-tab-pane>

                  <el-tab-pane label="数据包日志" name="packets">
                    <div class="result-toolbar">
                      <el-button type="primary" size="small" @click="clearPacketLogs(task.id)">
                        清空日志
                      </el-button>
                      <el-button type="info" size="small" @click="showStatusFilter = !showStatusFilter">
                        {{ showStatusFilter ? '隐藏过滤' : '显示过滤' }}
                      </el-button>
                    </div>
                    <div v-if="showStatusFilter" class="status-filter-bar">
                      <el-checkbox-group v-model="filteredStatusCodes" size="small">
                        <el-checkbox label="2xx">2xx (成功)</el-checkbox>
                        <el-checkbox label="3xx">3xx (重定向)</el-checkbox>
                        <el-checkbox label="4xx">4xx (客户端错误)</el-checkbox>
                        <el-checkbox label="5xx">5xx (服务器错误)</el-checkbox>
                        <el-checkbox label="other">其他</el-checkbox>
                      </el-checkbox-group>
                    </div>
                    <el-table :data="getPaginatedPacketRows(task)" border stripe size="small" max-height="400" class="glass-card">
                      <el-table-column prop="url" show-overflow-tooltip width="300">
                        <template #header>
                          <TableColumnFilterHeader
                            label="URL"
                            v-model="task.packetColFilters.url"
                            :popover-width="260"
                            @filter-change="handlePacketColFilter(task.id)"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column prop="method" width="88">
                        <template #header>
                          <TableColumnFilterHeader
                            label="方法"
                            v-model="task.packetColFilters.method"
                            :popover-width="200"
                            @filter-change="handlePacketColFilter(task.id)"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column prop="statusCode" width="120" sortable :sort-method="sortByStatusCode">
                        <template #header>
                          <TableColumnFilterHeader
                            label="状态码"
                            v-model="task.packetColFilters.statusCode"
                            :popover-width="200"
                            @filter-change="handlePacketColFilter(task.id)"
                          />
                        </template>
                        <template #default="scope">
                          <el-tag
                            :type="getStatusCodeType(scope.row.statusCode || scope.row.status_code)"
                            size="small"
                          >
                            {{ scope.row.statusCode || scope.row.status_code || '-' }}
                          </el-tag>
                        </template>
                      </el-table-column>
                      <el-table-column prop="duration" width="108">
                        <template #header>
                          <TableColumnFilterHeader
                            label="耗时"
                            v-model="task.packetColFilters.duration"
                            :popover-width="200"
                            @filter-change="handlePacketColFilter(task.id)"
                          />
                        </template>
                      </el-table-column>
                      <el-table-column prop="extractedInfo" show-overflow-tooltip>
                        <template #header>
                          <TableColumnFilterHeader
                            label="提取信息"
                            v-model="task.packetColFilters.extracted"
                            :popover-width="220"
                            @filter-change="handlePacketColFilter(task.id)"
                          />
                        </template>
                        <template #default="scope">
                          <el-tag v-if="scope.row.extractedInfo && scope.row.extractedInfo.length > 0" size="small" type="info">
                            {{ scope.row.extractedInfo.length }} 条
                          </el-tag>
                          <span v-else>-</span>
                        </template>
                      </el-table-column>
                      <el-table-column label="操作" width="150" fixed="right">
                        <template #default="scope">
                          <el-button 
                            size="small" 
                            :disabled="!scope.row.extractedInfo || scope.row.extractedInfo.length === 0"
                            @click="viewExtractedInfo(task.id, scope.row)"
                          >
                            查看敏感信息
                          </el-button>
                        </template>
                      </el-table-column>
                    </el-table>
                    <div class="pagination-wrapper">
                      <el-pagination small
                        v-model:current-page="getPaginationConfig(task.id, 'packets').page"
                        v-model:page-size="getPaginationConfig(task.id, 'packets').pageSize"
                        :page-sizes="[10, 20, 50, 100, 200]"
                        :total="getPacketFilteredTotal(task)"
                        layout="total, sizes, prev, pager, next, jumper"
                        background
                        @size-change="handlePageSizeChange(task.id, 'packets', $event)"
                        @current-change="handlePageChange(task.id, 'packets', $event)"
                      />
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
    
    <!-- 敏感接口过滤配置对话框 -->
    <el-dialog
      v-model="showSensitiveConfig"
      title="敏感接口过滤配置"
      width="600px"
      destroy-on-close
    >
      <div class="sensitive-config">
        <p style="margin-bottom: 15px; color: #909399;">
          包含以下关键词的接口将被自动过滤（每行一个关键词）
        </p>
        <el-input
          v-model="sensitiveKeywordsText"
          type="textarea"
          :rows="8"
          placeholder="例如: delete, insert, update, remove, drop, destroy"
        />
        <div style="margin-top: 15px; text-align: right;">
          <el-button @click="showSensitiveConfig = false">取消</el-button>
          <el-button type="primary" @click="saveSensitiveKeywords">保存</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- 常见API接口配置对话框 -->
    <el-dialog
      v-model="showApiRoutesConfig"
      title="常见API接口配置"
      width="600px"
      destroy-on-close
    >
      <div class="api-routes-config">
        <p style="margin-bottom: 15px; color: #909399;">
          配置常见的API接口前缀，扫描时会自动拼接这些前缀与提取的路由（每行一个）
        </p>
        <el-input
          v-model="commonApiRoutesText"
          type="textarea"
          :rows="10"
          placeholder="/api&#10;/api/v1&#10;/api/v2&#10;/rest&#10;/graphql"
        />
        <div style="margin-top: 15px; text-align: right;">
          <el-button @click="showApiRoutesConfig = false">取消</el-button>
          <el-button type="primary" @click="saveCommonApiRoutes">保存</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- 响应包信息提取结果对话框 -->
    <el-dialog
      v-model="showExtractedInfo"
      title="响应包信息提取结果"
      width="800px"
      destroy-on-close
    >
      <el-table :data="paginatedExtractedInfoList" border stripe size="small" max-height="400">
        <el-table-column prop="rule" label="匹配规则" width="200" show-overflow-tooltip />
        <el-table-column prop="value" label="提取内容" show-overflow-tooltip />
        <el-table-column label="操作" width="100">
          <template #default="scope">
            <el-button size="small" @click="copyToClipboard(scope.row.value)">复制</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrapper">
        <el-pagination small
          v-model:current-page="extractedInfoCurrentPage"
          v-model:page-size="extractedInfoPageSize"
          :page-sizes="[10, 20, 50, 100, 200]"
          :total="extractedInfoList?.length || 0"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="handleExtractedInfoPageSizeChange"
          @current-change="handleExtractedInfoPageChange"
        />
      </div>
    </el-dialog>

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
            <el-button :icon="Refresh" text @click="refreshTasks(true)" :loading="tasksLoading">刷新</el-button>
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
          <el-table-column prop="start_time" label="开始时间" width="180" align="center">
            <template #default="{ row }">
              {{ formatTime(row.start_time) }}
            </template>
          </el-table-column>
          <el-table-column prop="end_time" label="结束时间" width="180" align="center">
            <template #default="{ row }">
              {{ formatTime(row.end_time) }}
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

    <!-- 提取规则管理对话框 -->
    <el-dialog
      v-model="showExtractRulesConfig"
      title="提取规则管理"
      width="900px"
      destroy-on-close
    >
      <div class="extract-rules-config">
        <div class="rules-toolbar">
          <el-button type="primary" size="small" @click="handleAddExtractRule">
            添加规则
          </el-button>
          <el-button type="success" size="small" @click="saveExtractRules">
            保存所有规则
          </el-button>
          <el-button type="info" size="small" @click="resetExtractRules">
            重置为默认规则
          </el-button>
          <span style="color: #909399; margin-left: 10px;">正则测试地址: https://regex101.com/</span>
        </div>
        <el-table :data="paginatedExtractRulesList" border stripe size="small" max-height="400" class="rules-table">
          <el-table-column prop="name" label="规则名称" width="150" show-overflow-tooltip />
          <el-table-column prop="pattern" label="正则表达式" show-overflow-tooltip>
            <template #default="scope">
              <el-input
                v-if="scope.row.editing"
                v-model="scope.row.editPattern"
                size="small"
                @keyup.enter="saveExtractRuleEdit(scope.row)"
                @blur="saveExtractRuleEdit(scope.row)"
              />
              <span v-else>{{ scope.row.pattern }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="scope">
              <el-button
                v-if="!scope.row.editing"
                size="small"
                @click="startExtractRuleEdit(scope.row)"
              >
                编辑
              </el-button>
              <el-button
                v-else
                size="small"
                type="success"
                @click="saveExtractRuleEdit(scope.row)"
              >
                保存
              </el-button>
              <el-button
                size="small"
                type="danger"
                @click="handleDeleteExtractRule(scope.row.name)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="pagination-wrapper">
          <el-pagination small
            v-model:current-page="extractRulesCurrentPage"
            v-model:page-size="extractRulesPageSize"
            :page-sizes="[10, 20, 50, 100, 200]"
            :total="extractRulesList.length"
            layout="total, sizes, prev, pager, next, jumper"
            background
            @size-change="handleExtractRulesPageSizeChange"
            @current-change="handleExtractRulesPageChange"
          />
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { CreateTask, GetTask, GetAllTasks, StopTask, DeleteTask, ExportResults, GetExtractRules, SaveExtractRules, UpdateExtractRule, DeleteExtractRule } from '../../../wailsjs/go/browserScan/BrowserScan';
// 注意：GetTasksBasicInfo 需要重新生成 wails 绑定后才会出现
// 临时使用 GetAllTasks 但只提取基本信息
// GetTaskLogs 通过动态调用，因为可能还未生成绑定
import { GetWechatRules } from '../../../wailsjs/go/unwxapp/UnWxapp';
import { ElMessage, ElMessageBox } from 'element-plus';
import CustomTabs from '@/components/CustomTabs.vue';
import { Plus, Compass, VideoPlay, CircleCheck, CircleClose, Clock, Refresh, Delete, View } from '@element-plus/icons-vue';
import TableColumnFilterHeader from '../../components/TableColumnFilterHeader.vue';
import { BrowserOpenURL } from '../../../wailsjs/runtime';

export default {
  name: 'BrowserScanView',
  components: { CustomTabs, Plus, Compass, VideoPlay, CircleCheck, CircleClose, Clock, Refresh, Delete, View, TableColumnFilterHeader },
  data() {
    return {
      activeTaskId: '',
      // 顶层结果标签：endpoints | frontend | secrets | packets
      activeResultTab: {},
      // 二级：端点与网络（urls / api / domains …）
      resultSubTabEndpoints: {},
      // 二级：前端与资源（vue / jsfiles / staticfiles）
      resultSubTabFrontend: {},
      // 二级：敏感信息内分类
      resultSubTabSecrets: {},
      tasks: [],
      taskUpdateTimer: null,
      sensitiveKeywords: ['delete', 'insert', 'update', 'remove', 'drop', 'destroy'],
      showStatusFilter: false,
      filteredStatusCodes: ['2xx', '3xx', '4xx', '5xx', 'other'],
      showSensitiveConfig: false,
      showApiRoutesConfig: false,
      commonApiRoutes: ['/api', '/api/v1', '/api/v2', '/rest', '/graphql'],
      taskIdCounter: 0, // 用于生成唯一任务ID
      showExtractedInfo: false,
      extractedInfoList: [],
      showExtractRulesConfig: false,
      extractRules: {}, // 提取规则对象 {name: pattern}
      extractRulesList: [], // 提取规则列表，用于表格显示
      extractRulesPageSize: 10, // 每页显示数量
      extractRulesCurrentPage: 1, // 当前页码
      _eventsBound: false, // 标记事件是否已绑定
      handleLogEventWrapper: null,
      handlePacketEventWrapper: null,
      _loadTasksInProgress: false, // 标记loadTasks是否正在执行
      _packetEventQueue: [], // 数据包事件队列（用于防抖）
      _packetEventTimer: null, // 数据包事件防抖定时器
      // 分页配置：每个任务每个标签页的分页状态 { taskId: { tabName: { page: 1, pageSize: 20 } } }
      paginationConfig: {},
      defaultPageSize: 10, // 默认每页显示数量
      // 提取信息对话框分页
      extractedInfoPageSize: 10,
      extractedInfoCurrentPage: 1,
      // 任务管理相关
      historyDialog: false,
      allTasks: [], // 所有任务（用于任务管理）
      tasksLoading: false,
      tasksCache: null, // 任务列表缓存
      tasksCacheTime: 0, // 缓存时间戳
      TASKS_CACHE_DURATION: 30000, // 30秒缓存
      taskPagination: {
        currentPage: 1,
        pageSize: 10
      },
      // 标签页右键菜单相关
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: ''
    };
  },
  computed: {
    sensitiveKeywordsText: {
      get() {
        return this.sensitiveKeywords.join('\n');
      },
      set(value) {
        this.sensitiveKeywords = value.split('\n')
          .map(k => k.trim())
          .filter(k => k.length > 0);
      }
    },
    commonApiRoutesText: {
      get() {
        return this.commonApiRoutes.join('\n');
      },
      set(value) {
        this.commonApiRoutes = value.split('\n')
          .map(k => k.trim())
          .filter(k => k.length > 0);
      }
    },
    // 分页后的提取规则列表
    paginatedExtractRulesList() {
      const start = (this.extractRulesCurrentPage - 1) * this.extractRulesPageSize;
      const end = start + this.extractRulesPageSize;
      return this.extractRulesList.slice(start, end);
    },
    // 获取分页后的提取信息列表
    paginatedExtractedInfoList() {
      const start = (this.extractedInfoCurrentPage - 1) * this.extractedInfoPageSize;
      const end = start + this.extractedInfoPageSize;
      return this.extractedInfoList.slice(start, end);
    },
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
    this.startTaskUpdateTimer();
    this.loadSensitiveKeywords();
    this.loadCommonApiRoutes();
    
    // 监听日志事件 - 通过runtime.EventsOn监听
    // 注意：这里的事件监听器是全局的，不需要在每个组件实例中重复绑定
    // 但为了安全，我们只在未绑定时才绑定
    if (window.go?.main?.App?.On && !this._eventsBound) {
      this.handleLogEventWrapper = (data) => {
        this.handleLogEvent(data);
      };
      this.handlePacketEventWrapper = (data) => {
        this.handlePacketEvent(data);
      };
      
      window.go.main.App.On('browser_scan_log', this.handleLogEventWrapper);
      window.go.main.App.On('browser_scan_packet', this.handlePacketEventWrapper);
      
      this._eventsBound = true;
    }
  },
  beforeUnmount() {
    if (this.taskUpdateTimer) {
      clearInterval(this.taskUpdateTimer);
    }
    // 清除数据包事件防抖定时器
    if (this._packetEventTimer) {
      clearTimeout(this._packetEventTimer);
      this._packetEventTimer = null;
    }
    // 处理队列中剩余的事件
    if (this._packetEventQueue.length > 0) {
      this._processPacketEventQueue();
    }
    // 移除事件监听器（如果有对应的Off方法）
    if (window.go?.main?.App?.Off && this._eventsBound) {
      if (this.handleLogEventWrapper) {
        window.go.main.App.Off('browser_scan_log', this.handleLogEventWrapper);
      }
      if (this.handlePacketEventWrapper) {
        window.go.main.App.Off('browser_scan_packet', this.handlePacketEventWrapper);
      }
      this._eventsBound = false;
    }
  },
  methods: {
    async loadTasks() {
      // 防抖：如果已经在加载中，跳过本次调用
      // 注意：这里不能使用 tasksLoading，因为它是任务管理的加载状态
      // 使用一个单独的标志来防止并发调用 loadTasks
      if (this._loadTasksInProgress) {
        return;
      }
      
      this._loadTasksInProgress = true;
      try {
        // 保存当前的前端任务（用于匹配和更新）
        const currentTasksMap = new Map();
        this.tasks.forEach(task => {
          currentTasksMap.set(task.id, task);
        });
        
        // 保存前端pending任务（未启动的任务）
        const pendingTasks = Array.from(currentTasksMap.values()).filter(t => t.status === 'pending' && !t.backendTaskId);
        
        // 优化：从后端获取所有任务，但只处理已打开的任务
        // 这样可以避免每次刷新都加载大量任务数据导致卡顿
        // 只获取基本信息，不加载完整数据
        const backendTasks = await GetAllTasks();
        
        // 合并任务：使用前端任务ID作为key，保留前端任务结构
        const mergedTasks = [];
        
        // 记录已添加到mergedTasks的任务ID，避免重复
        const addedTaskIds = new Set();
        
        // 1. 处理后端任务：只更新已打开的前端任务，避免创建所有任务的完整数据
        // 只处理当前打开的任务或运行中的任务，不加载所有历史任务
        const openedTaskIds = new Set(this.tasks.map(t => t.id).concat(this.tasks.map(t => t.backendTaskId).filter(Boolean)));
        const runningTaskIds = new Set(this.tasks.filter(t => t.status === 'running').map(t => t.backendTaskId || t.id));
        
        (backendTasks || []).forEach(backendTask => {
          // 只处理已打开的任务或正在运行的任务
          const shouldProcess = openedTaskIds.has(backendTask.id) || runningTaskIds.has(backendTask.id);
          if (!shouldProcess && backendTask.status !== 'running') {
            // 跳过未打开且未运行的任务，避免加载大量数据
            return;
          }
          // 查找是否有前端任务对应这个后端任务（通过backendTaskId匹配）
          const frontendTask = Array.from(currentTasksMap.values()).find(
            t => t.backendTaskId === backendTask.id
          );
          
          if (frontendTask) {
            this.ensureTaskColFilters(frontendTask)
            // 找到对应的前端任务，只更新状态等基本信息，保留已有的数据
            // 这样可以避免每次刷新都重新加载大量数据
            const statusChanged = frontendTask.status !== backendTask.status;
            const shouldUpdateData = statusChanged || !frontendTask.results || !frontendTask.packetLogs || frontendTask.packetLogs.length === 0;
            
            frontendTask.name = backendTask.url || frontendTask.name || `任务-${backendTask.id}`;
            frontendTask.url = backendTask.url || frontendTask.url;
            frontendTask.status = backendTask.status || frontendTask.status;
            frontendTask.backendTaskId = backendTask.id;
            
            // 只在状态变化或数据缺失时才更新完整数据
            if (shouldUpdateData) {
              frontendTask.config = {
                waitTime: backendTask.config?.waitTime || frontendTask.config?.waitTime || 5,
                apiRoutes: backendTask.config?.apiRoutes || frontendTask.config?.apiRoutes || [],
                globalApiRoutes: backendTask.config?.globalApiRoutes || frontendTask.config?.globalApiRoutes || [],
                allowList: backendTask.config?.allowList || frontendTask.config?.allowList || [],
                autoTestApi: backendTask.config?.autoTestApi || frontendTask.config?.autoTestApi || false,
                testMethods: backendTask.config?.testMethods || frontendTask.config?.testMethods || ['GET'],
                workerNum: backendTask.config?.workerNum || backendTask.config?.worker_num || frontendTask.config?.workerNum || 10
              };
              frontendTask.results = this.formatResults(backendTask.results || {});
              // 优化：合并packetLogs时限制数量以节省内存
              const backendLogs = backendTask.packet_logs || backendTask.packetLogs || [];
              // 进一步优化：只保留最新的100条（而不是500条）以节省内存
              const latestBackendLogs = backendLogs.slice(-100);
              const mergedLogs = this.mergePacketLogs(frontendTask.packetLogs || [], latestBackendLogs);
              // 限制数据包日志数量为100条（保留最新的）以节省内存
              frontendTask.packetLogs = mergedLogs.slice(-100).map(log => ({
                url: log.url || '',
                method: log.method || '',
                statusCode: log.status_code || log.statusCode || 0,
                duration: log.duration || '',
                // 优化：截断过长的请求/响应包，进一步减少内存占用（从1000字符减少到500字符）
                requestPacket: (log.request_packet || log.requestPacket || '').substring(0, 500),
                responsePacket: (log.response_packet || log.responsePacket || '').substring(0, 500),
                extractedInfo: log.extracted_info || log.extractedInfo || []
              }));
            }
            // Vue 分析在后端扫描末尾写入；running 时若未整表刷新 results，也要合并 vue_analysis
            const vueSrc = backendTask.results?.vue_analysis ?? backendTask.results?.vueAnalysis;
            if (vueSrc && frontendTask.results) {
              frontendTask.results.vueAnalysis = this.normalizeVueAnalysis(vueSrc);
            }

            mergedTasks.push(frontendTask);
            addedTaskIds.add(frontendTask.id);
            // 确保已有任务的结果标签页状态已初始化
            if (!this.activeResultTab[frontendTask.id]) {
              this.activeResultTab[frontendTask.id] = 'endpoints';
            }
            this.ensureResultSubTabs(frontendTask.id);
          } else {
            // 没有对应的前端任务，创建新任务（使用后端ID作为前端ID）
            const newTask = {
              id: backendTask.id,
              name: backendTask.url || `任务-${backendTask.id}`,
              url: backendTask.url,
              status: backendTask.status,
              backendTaskId: backendTask.id,
              config: {
                waitTime: backendTask.config?.waitTime || 5,
                apiRoutes: backendTask.config?.apiRoutes || [],
                globalApiRoutes: backendTask.config?.globalApiRoutes || [],
                allowList: backendTask.config?.allowList || [],
                autoTestApi: backendTask.config?.autoTestApi || false,
                testMethods: backendTask.config?.testMethods || ['GET'],
                workerNum: backendTask.config?.workerNum || backendTask.config?.worker_num || 10
              },
              results: this.formatResults(backendTask.results || {}),
              // 优化：限制数据包日志数量为100条（保留最新的）以节省内存（从500减少到100）
              // 后端JSON字段名是packet_logs，需要兼容处理，统一字段名格式
              packetLogs: (backendTask.packet_logs || backendTask.packetLogs || []).slice(-100).map(log => ({
                url: log.url || '',
                method: log.method || '',
                statusCode: log.status_code || log.statusCode || 0,
                duration: log.duration || '',
                // 优化：截断过长的请求/响应包，进一步减少内存占用（从1000字符减少到500字符）
                requestPacket: (log.request_packet || log.requestPacket || '').substring(0, 500),
                responsePacket: (log.response_packet || log.responsePacket || '').substring(0, 500),
                extractedInfo: log.extracted_info || log.extractedInfo || []
              })),
              colFilters: {},
              packetColFilters: {
                url: '',
                method: '',
                statusCode: '',
                duration: '',
                extracted: ''
              }
            };
            mergedTasks.push(newTask);
            addedTaskIds.add(backendTask.id);
            // 初始化从后端加载的任务的结果标签页状态
            if (!this.activeResultTab[backendTask.id]) {
              this.activeResultTab[backendTask.id] = 'endpoints';
            }
            this.ensureResultSubTabs(backendTask.id);
          }
        });
        
        // 2. 添加前端pending任务（这些任务还未启动，不在后端），但要避免重复
        pendingTasks.forEach(task => {
          // 只添加尚未添加到mergedTasks的pending任务
          if (!addedTaskIds.has(task.id)) {
            mergedTasks.push(task);
            addedTaskIds.add(task.id);
            // 确保pending任务的结果标签页状态已初始化
            if (!this.activeResultTab[task.id]) {
              this.activeResultTab[task.id] = 'endpoints';
            }
            this.ensureResultSubTabs(task.id);
          }
        });
        
        this.tasks = mergedTasks;
        
        // 如果当前活动任务仍然存在，保持活动状态；否则选择第一个
        if (this.activeTaskId && this.tasks.find(t => t.id === this.activeTaskId)) {
          // 活动任务仍然存在，保持不变
        } else if (this.tasks.length > 0) {
          this.activeTaskId = this.tasks[0].id;
        } else {
          this.activeTaskId = '';
        }
      } catch (error) {
        console.error('加载任务失败:', error);
        // 出错时保留前端pending任务
        this.tasks = this.tasks.filter(t => t.status === 'pending' && !t.backendTaskId);
      } finally {
        this._loadTasksInProgress = false;
      }
    },
    
    normalizeVueAnalysis(v) {
      if (!v) return null;
      return {
        vueDetected: v.vue_detected ?? v.vueDetected ?? false,
        routerDetected: v.router_detected ?? v.routerDetected ?? false,
        vueVersion: v.vue_version ?? v.vueVersion ?? '',
        routerBase: v.router_base ?? v.routerBase ?? '',
        currentPath: v.current_path ?? v.currentPath ?? '',
        error: v.error ?? '',
        modifiedRoutes: v.modified_routes || v.modifiedRoutes || [],
        allRoutes: v.all_routes || v.allRoutes || [],
        pageAnalysis: v.page_analysis || v.pageAnalysis || null
      };
    },

    formatResults(results) {
      const formatArray = (arr) => {
        if (!Array.isArray(arr)) return [];
        return arr.map(item => ({ value: item }));
      };
      
      return {
        urls: formatArray(results.urls),
        apiEndpoints: formatArray(results.apiEndpoints),
        domains: formatArray(results.domains),
        ips: formatArray(results.ips),
        ipPorts: formatArray(results.ipPorts || results.ip_ports || []),
        paths: formatArray(results.paths || []),
        incompletePaths: formatArray(results.incompletePaths || results.incomplete_paths || []),
        jsFiles: formatArray(results.jsFiles || results.js_files || []),
        staticFiles: formatArray(results.staticFiles || results.static_files || []),
        algorithms: formatArray(results.algorithms || []),
        jwt: formatArray(results.jwt),
        mobiles: formatArray(results.mobiles),
        emails: formatArray(results.emails),
        secrets: formatArray(results.secrets || []),
        // 敏感信息分类字段
        securityKeys: formatArray(results.security_keys || []),
        passwords: formatArray(results.passwords || []),
        usernames: formatArray(results.usernames || []),
        aksk: formatArray(results.aksk || []),
        appIds: formatArray(results.app_ids || []),
        accessKeys: formatArray(results.access_keys || []),
        apiKeys: formatArray(results.api_keys || []),
        secretKeys: formatArray(results.secret_keys || []),
        idCards: formatArray(results.id_cards || []),
        vueAnalysis: this.normalizeVueAnalysis(results.vue_analysis || results.vueAnalysis)
      };
    },

    // 合并数据包日志：保留前端已有的，添加后端新的（去重）
    mergePacketLogs(frontendLogs, backendLogs) {
      const merged = [...(frontendLogs || [])];
      const seen = new Set();
      
      // 记录已有数据包的key
      frontendLogs.forEach(log => {
        const statusCode = log.status_code || log.statusCode || 0;
        const key = `${log.url || ''}_${log.method || ''}_${statusCode}`;
        seen.add(key);
      });
      
      // 添加后端新的数据包，统一字段名格式
      (backendLogs || []).forEach(log => {
        const statusCode = log.status_code || log.statusCode || 0;
        const key = `${log.url || ''}_${log.method || ''}_${statusCode}`;
        if (!seen.has(key)) {
          // 统一字段名格式
          merged.push({
            url: log.url || '',
            method: log.method || '',
            statusCode: statusCode,
            duration: log.duration || '',
            requestPacket: log.request_packet || log.requestPacket || '',
            responsePacket: log.response_packet || log.responsePacket || '',
            extractedInfo: log.extracted_info || log.extractedInfo || []
          });
          seen.add(key);
        }
      });
      
      return merged;
    },
    
    addTask() {
      const taskId = `task_${Date.now()}_${++this.taskIdCounter}`;
      const newTask = {
        id: taskId,
        name: `JSFinder-${new Date().toLocaleTimeString()}`,
        url: '',
        status: 'pending', // pending状态表示还未启动
        backendTaskId: null, // 启动后会有后端任务ID
        config: {
          waitTime: 5,
          apiRoutes: [],
          globalApiRoutes: [],
          allowList: [],
          autoTestApi: false,
          testMethods: ['GET'],
          workerNum: 10 // 默认10个线程
        },
            results: {
              urls: [],
              apiEndpoints: [],
              domains: [],
              ips: [],
              jsFiles: [],
              jwt: [],
              mobiles: [],
              emails: [],
              vueAnalysis: null
            },
            packetLogs: [],
            colFilters: {},
            packetColFilters: {
              url: '',
              method: '',
              statusCode: '',
              duration: '',
              extracted: ''
            }
      };
      
      this.tasks.push(newTask);
      this.activeTaskId = taskId;
      // 初始化新任务的结果标签页状态
      this.activeResultTab[taskId] = 'endpoints';
      this.ensureResultSubTabs(taskId);
    },
    
    async startScan(taskId) {
      const task = this.tasks.find(t => t.id === taskId);
      if (!task) return;
      
      if (!task.url) {
        ElMessage.warning('请输入目标URL');
        return;
      }
      
      // 如果任务正在运行或正在停止，不执行启动操作
      if (task.status === 'running' || task.status === 'stopping' || task.status === 'starting') {
        return;
      }
      
      // 保存原始状态，用于错误恢复
      const originalStatus = task.status;
      
      try {
        // 立即设置状态为 starting，显示加载状态和进度提示
        task.status = 'starting';
        
        // 如果任务之前有 backendTaskId（已停止的任务），清除它以创建新任务
        // 保留旧的 backendTaskId 用于历史记录，但创建新的任务
        const oldBackendTaskId = task.backendTaskId;
        
        // 使用配置的常见API路由
        const apiRoutes = this.commonApiRoutes || [];
        
        // 确保配置字段名与后端匹配（驼峰转蛇形）
        const config = {
          wait_time: task.config.waitTime || 5,
          extract_rules: task.config.extractRules || [],
          api_routes: apiRoutes,
          global_api_routes: task.config.globalApiRoutes || [],
          allow_list: task.config.allowList || [],
          auto_test_api: task.config.autoTestApi || false, // 确保正确传递
          test_methods: task.config.testMethods || ['GET'],
          worker_num: task.config.workerNum || 10 // 默认10个线程
        };
        
        // 调用后端API创建新任务
        const backendTaskId = await CreateTask(task.url, config);
        
        // 更新任务状态，关联新的后端任务ID
        task.backendTaskId = backendTaskId;
        task.status = 'running';
        
        // 如果任务之前有数据，可以选择清空或保留
        // 为了允许重新扫描，这里不清空数据，让用户可以看到历史数据
        
        ElMessage.success('扫描任务已启动');
        
        // 立即加载任务状态和进度信息，而不是等待定时器
        // 这样可以立即显示扫描进度
        this.loadTasks().catch(err => {
          console.error('启动后加载任务状态失败:', err);
        });
      } catch (error) {
        ElMessage.error('启动扫描失败: ' + error);
        // 启动失败时，恢复到原始状态
        if (!task.backendTaskId) {
          task.status = originalStatus || 'pending'; // 恢复到原始状态或 pending
        } else {
          // 如果之前有 backendTaskId（已停止的任务），恢复到 stopped 状态
          task.status = 'stopped';
        }
      }
    },
    
    async stopScan(taskId) {
      const task = this.tasks.find(t => t.id === taskId);
      if (!task) {
        ElMessage.warning('任务不存在');
        return;
      }
      
      if (!task.backendTaskId) {
        ElMessage.warning('任务尚未启动，无法停止');
        return;
      }

      if (task.status !== 'running') {
        ElMessage.warning('任务当前不在运行状态');
        return;
      }
      
      try {
        // 先更新状态为 stopping，避免重复点击
        task.status = 'stopping';
        
        // 保存 backendTaskId，用于停止后端任务
        const backendTaskId = task.backendTaskId;
        
        // 不等待StopTask完成，立即更新状态为stopped
        // 这样用户界面能立即响应，不会一直显示"正在停止中"
        task.status = 'stopped';
        ElMessage.success('扫描任务正在停止...');
        
        // 异步调用后端StopTask，不阻塞UI
        StopTask(backendTaskId).catch(error => {
          console.error('停止扫描失败:', error);
          // 如果后端停止失败，尝试刷新任务状态
          this.loadTasks();
        });
        
        // 立即刷新任务列表以获取最新状态
        // 注意：不要清除 backendTaskId，以便保留任务历史数据
        // 但在 startScan 中会创建新的任务，旧的 backendTaskId 会被替换
        await this.loadTasks();
      } catch (error) {
        console.error('停止扫描失败:', error);
        // 恢复状态为 running（如果停止失败）
        if (task.status === 'stopping') {
          task.status = 'running';
        }
        ElMessage.error('停止扫描失败: ' + error);
      }
    },
    
    async removeTask(taskId) {
      if (taskId === 'add') return;
      
      const task = this.tasks.find(t => t.id === taskId);
      if (!task) return;
      
      // 删除标签页时，只从前端任务列表中移除，不删除后端任务
      // 这样任务管理中的数据会保留，用户可以通过任务管理重新加载任务
      this.tasks = this.tasks.filter(t => t.id !== taskId);
      
      // 清理该任务的结果标签页状态和分页配置
      delete this.activeResultTab[taskId];
      delete this.paginationConfig[taskId];
      delete this.resultSubTabEndpoints[taskId];
      delete this.resultSubTabFrontend[taskId];
      delete this.resultSubTabSecrets[taskId];

      // 更新活动任务
      if (this.activeTaskId === taskId) {
        this.activeTaskId = this.tasks.length > 0 ? this.tasks[0].id : '';
      }

      // 避免进入无标签空白态，保持一个可操作的空白任务标签
      if (this.tasks.length === 0) {
        this.addTask();
      }
    },
    
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
        delete this.resultSubTabEndpoints[task.id]
        delete this.resultSubTabFrontend[task.id]
        delete this.resultSubTabSecrets[task.id]
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
        delete this.resultSubTabEndpoints[task.id]
        delete this.resultSubTabFrontend[task.id]
        delete this.resultSubTabSecrets[task.id]
      }

      // 清空标签列表
      this.tasks = []
      // 保留一个空白任务标签，避免页面进入 empty 态
      this.addTask()
    },
    
    copyToClipboard(text) {
      navigator.clipboard.writeText(text).then(() => {
        ElMessage.success('已复制到剪贴板');
      });
    },

    BrowserOpenURL,

    getVueAnalysisForTask(task) {
      if (!task?.results) return null;
      return task.results.vueAnalysis || task.results.vue_analysis || null;
    },

    formatVueMetaShort(meta) {
      if (!meta || typeof meta !== 'object') return '-';
      try {
        const s = JSON.stringify(meta);
        return s.length > 120 ? `${s.slice(0, 120)}…` : s;
      } catch {
        return '-';
      }
    },

    /**
     * 与 VueCrack popup.js displayUrlList 中 URL 拼接逻辑对齐：
     * - 含 # 或 #/：按 hash 规则拼（baseUrl 取 # 前含 # 一段）
     * - 否则 history：standardUrl = origin + '/' + cleanPath；若 detectedBasePath 且 path 不以该前缀开头，再算 basePathUrl = origin + detectedBasePath + '/' + cleanPath
     * - detectedBasePath 优先 routerBase → pageAnalysis.detectedBasePath → pathname 多段时取首段
     * JSFinder 补充：扫描 URL 无 # 但 pathname 非根时，增加 pathAppBase + '/#/' + cleanPath（Rod 扫描常为无 hash 的入口地址）
     */
    buildVueFullUrlRows(task) {
      const va = this.getVueAnalysisForTask(task);
      const routes = (va?.allRoutes || []).filter(
        (r) => r && typeof r === 'object' && r.path && typeof r.path === 'string'
      );
      if (!routes.length || !task.url) return [];

      let urlObj;
      try {
        urlObj = new URL(task.url.trim());
      } catch {
        return [];
      }

      const currentUrl = task.url.trim();
      const domainBase = urlObj.origin;
      const currentPath = urlObj.pathname;

      let baseUrl = '';
      let isHistoryMode = false;
      if (currentUrl.includes('#/') || currentUrl.includes('#')) {
        const hashIndex = currentUrl.indexOf('#');
        baseUrl = currentUrl.substring(0, hashIndex + 1);
      } else {
        isHistoryMode = true;
        baseUrl = domainBase;
      }

      let detectedBasePath = '';
      if (va?.routerBase) detectedBasePath = String(va.routerBase);
      else if (va?.pageAnalysis?.detectedBasePath) detectedBasePath = String(va.pageAnalysis.detectedBasePath);
      else {
        const pathSegments = currentPath.split('/').filter(Boolean);
        if (pathSegments.length > 1) detectedBasePath = `/${pathSegments[0]}`;
      }
      if (detectedBasePath && !detectedBasePath.startsWith('/')) {
        detectedBasePath = `/${detectedBasePath}`;
      }
      detectedBasePath = detectedBasePath.replace(/\/$/, '');

      const pathTail = (currentPath || '').replace(/\/+$/, '');
      const pathAppBase = !pathTail ? domainBase : `${domainBase}${pathTail}`;

      const cleanUrlFn = (s) => s.replace(/([^:]\/)\/+/g, '$1').replace(/\/$/, '');

      const rows = [];
      for (const route of routes) {
        const path = route.path;
        const cleanPath = path.startsWith('/') ? path.substring(1) : path;
        let standardUrl;
        let basePathUrl = null;

        if (isHistoryMode) {
          standardUrl = `${baseUrl}/${cleanPath}`;
          if (detectedBasePath && !path.startsWith(detectedBasePath)) {
            basePathUrl = `${baseUrl}${detectedBasePath}/${cleanPath}`;
          }
        } else if (baseUrl.endsWith('#')) {
          standardUrl = `${baseUrl}/${cleanPath}`;
        } else if (baseUrl.endsWith('#/')) {
          standardUrl = `${baseUrl}${cleanPath}`;
        } else {
          standardUrl = `${baseUrl}#/${cleanPath}`;
        }

        standardUrl = cleanUrlFn(standardUrl);
        if (basePathUrl) basePathUrl = cleanUrlFn(basePathUrl);

        let hashSupplement = null;
        if (isHistoryMode && pathAppBase !== domainBase) {
          hashSupplement = cleanUrlFn(`${pathAppBase}/#/${cleanPath}`);
        }

        const candidates = [];
        if (hashSupplement && hashSupplement !== standardUrl) {
          candidates.push(hashSupplement);
        }
        if (basePathUrl && !candidates.includes(basePathUrl) && basePathUrl !== standardUrl) {
          candidates.push(basePathUrl);
        }
        if (!candidates.includes(standardUrl)) {
          candidates.push(standardUrl);
        }

        candidates.forEach((url, i) => {
          rows.push({ path, url, altBase: i > 0 });
        });
      }
      return rows;
    },

    vueRowFullUrl(task, row) {
      const rows = this.buildVueFullUrlRows(task).filter((x) => x.path === row.path);
      const primary = rows.find((x) => !x.altBase);
      return (primary || rows[0])?.url || '';
    },

    copyVuePaths(task) {
      const va = this.getVueAnalysisForTask(task);
      const text = (va?.allRoutes || []).map((r) => r.path).filter(Boolean).join('\n');
      if (!text) {
        ElMessage.warning('没有可复制的路径');
        return;
      }
      this.copyToClipboard(text);
    },

    copyVueFullUrls(task) {
      const urls = this.buildVueFullUrlRows(task).map((x) => x.url).filter(Boolean);
      const text = [...new Set(urls)].join('\n');
      if (!text) {
        ElMessage.warning('没有可复制的完整 URL');
        return;
      }
      this.copyToClipboard(text);
    },

    startTaskUpdateTimer() {
      // 优化：增加更新间隔，减少频繁加载（从8秒改为10秒）
      // 这样可以减少内存占用和CPU使用
      // 进一步优化：只更新运行中的任务，避免频繁加载已完成任务的大量数据
      this.taskUpdateTimer = setInterval(() => {
        // 检查是否有运行中的任务，如果没有则不更新
        const hasRunningTask = this.tasks.some(t => t.status === 'running');
        if (hasRunningTask) {
          // 只更新运行中的任务，不加载已完成任务的完整数据
          // 使用防抖，避免在加载过程中重复调用
          if (!this.tasksLoading) {
            this.loadTasks();
          }
        }
      }, 10000); // 增加到10秒，进一步减少更新频率
    },
    
    handleLogEvent(data) {
      // 日志相关功能已移除，此方法保留以兼容事件系统
    },
    
    // 处理自动测试API复选框变化
    async handleAutoTestApiCheckboxChange(taskId, value) {
      if (value) {
      }
    },
    
    // 保存敏感关键词配置
    saveSensitiveKeywords() {
      localStorage.setItem('browserScan:sensitiveKeywords', JSON.stringify(this.sensitiveKeywords));
      ElMessage.success('敏感关键词配置已保存');
      this.showSensitiveConfig = false;
    },
    
    // 加载敏感关键词配置
    loadSensitiveKeywords() {
      const saved = localStorage.getItem('browserScan:sensitiveKeywords');
      if (saved) {
        try {
          this.sensitiveKeywords = JSON.parse(saved);
        } catch (e) {
          console.error('加载敏感关键词配置失败:', e);
        }
      }
    },

    // 保存常见API路由配置
    saveCommonApiRoutes() {
      localStorage.setItem('browserScan:commonApiRoutes', JSON.stringify(this.commonApiRoutes));
      ElMessage.success('常见API接口配置已保存');
      this.showApiRoutesConfig = false;
    },

    // 加载常见API路由配置
    loadCommonApiRoutes() {
      const saved = localStorage.getItem('browserScan:commonApiRoutes');
      if (saved) {
        try {
          this.commonApiRoutes = JSON.parse(saved);
        } catch (e) {
          console.error('加载常见API路由配置失败:', e);
        }
      }
    },

    // 处理数据包事件（优化内存占用和性能，使用防抖机制）
    handlePacketEvent(data) {
      // 将事件添加到队列中，而不是立即处理
      this._packetEventQueue.push(data);
      
      // 如果定时器不存在，创建一个新的防抖定时器
      if (!this._packetEventTimer) {
        // 使用防抖，每200ms批量处理一次事件，减少UI更新频率
        this._packetEventTimer = setTimeout(() => {
          this._processPacketEventQueue();
          this._packetEventTimer = null;
        }, 200);
      }
    },
    
    // 批量处理数据包事件队列（防抖处理）
    _processPacketEventQueue() {
      if (this._packetEventQueue.length === 0) {
        return;
      }
      
      // 使用nextTick批量更新，减少Vue响应式更新的开销
      this.$nextTick(() => {
        const queue = [...this._packetEventQueue];
        this._packetEventQueue = [];
        
        // 按任务ID分组处理事件
        const taskEvents = {};
        queue.forEach(data => {
          const task = this.tasks.find(t => {
            return t.id === data.task_id || t.backendTaskId === data.task_id;
          });
          
          if (!task) {
            return;
          }
          
          // 如果任务不在前台（不是当前活动的任务），跳过更新
          if (this.activeTaskId !== task.id && task.status !== 'running') {
            return;
          }
          
          if (!taskEvents[task.id]) {
            taskEvents[task.id] = { task, events: [] };
          }
          taskEvents[task.id].events.push(data);
        });
        
        // 批量处理每个任务的事件
        Object.values(taskEvents).forEach(({ task, events }) => {
          // 确保packetLogs数组存在
          if (!task.packetLogs) {
            task.packetLogs = [];
          }
          
          // 批量添加数据包（去重）
          const seen = new Set();
          const recentLogs = task.packetLogs.slice(-50);
          recentLogs.forEach(log => {
            const key = `${log.url}_${log.method}_${log.statusCode}`;
            seen.add(key);
          });
          
          const newPackets = [];
          events.forEach(data => {
            if (!data.packet) return;
            
            const packetUrl = data.packet?.url || '';
            const packetMethod = data.packet?.method || '';
            const packetStatusCode = data.packet?.status_code || data.packet?.statusCode || 0;
            const key = `${packetUrl}_${packetMethod}_${packetStatusCode}`;
            
            if (!seen.has(key)) {
              seen.add(key);
              newPackets.push({
                url: packetUrl,
                method: packetMethod,
                statusCode: packetStatusCode,
                duration: data.packet.duration || '',
                requestPacket: (data.packet.request_packet || data.packet.requestPacket || '').substring(0, 1000),
                responsePacket: (data.packet.response_packet || data.packet.responsePacket || '').substring(0, 1000),
                extractedInfo: data.packet.extracted_info || data.packet.extractedInfo || []
              });
            }
          });
          
          // 批量添加新数据包
          if (newPackets.length > 0) {
            // 限制总数为500条
            const totalLength = task.packetLogs.length + newPackets.length;
            if (totalLength > 500) {
              const removeCount = totalLength - 500;
              task.packetLogs.splice(0, removeCount);
            }
            
            // 批量添加新数据包（一次性push多个元素，比多次push效率更高）
            task.packetLogs.push(...newPackets);
          }
        });
      });
    },

    // 查看提取的敏感信息
    viewExtractedInfo(taskId, packet) {
      const extractedInfo = packet.extracted_info || packet.extractedInfo || [];
      if (extractedInfo && extractedInfo.length > 0) {
        this.extractedInfoList = extractedInfo;
        this.extractedInfoCurrentPage = 1; // 重置分页到第一页
        this.showExtractedInfo = true;
      } else {
        ElMessage.warning('该数据包没有提取到敏感信息');
      }
    },

    // 清空数据包日志
    clearPacketLogs(taskId) {
      const task = this.tasks.find(t => t.id === taskId);
      if (task) {
        task.packetLogs = [];
        ElMessage.success('数据包日志已清空');
      }
    },
    
    // 获取状态码类型
    getStatusCodeType(statusCode) {
      if (!statusCode) return 'info';
      if (statusCode >= 200 && statusCode < 300) return 'success';
      if (statusCode >= 300 && statusCode < 400) return 'warning';
      if (statusCode >= 400 && statusCode < 500) return 'danger';
      if (statusCode >= 500) return 'danger';
      return 'info';
    },
    
    // 状态码排序
    sortByStatusCode(a, b) {
      const statusA = a.statusCode || a.status_code || 0;
      const statusB = b.statusCode || b.status_code || 0;
      return statusA - statusB;
    },
    
    // 过滤数据包日志
    filteredPacketLogs(task) {
      if (!task || !task.packetLogs) return [];
      
      // 如果没有选择任何过滤条件，返回所有
      if (!this.filteredStatusCodes || this.filteredStatusCodes.length === 0) {
        return task.packetLogs;
      }
      
      return task.packetLogs.filter(packet => {
        const statusCode = packet.statusCode || packet.status_code || 0;
        
        // 判断状态码属于哪个范围
        let statusRange = 'other';
        if (statusCode >= 200 && statusCode < 300) {
          statusRange = '2xx';
        } else if (statusCode >= 300 && statusCode < 400) {
          statusRange = '3xx';
        } else if (statusCode >= 400 && statusCode < 500) {
          statusRange = '4xx';
        } else if (statusCode >= 500) {
          statusRange = '5xx';
        }
        
        return this.filteredStatusCodes.includes(statusRange);
      });
    },

    // 导出到Excel
    async exportToExcel(taskId) {
      try {
        const task = this.tasks.find(t => t.id === taskId);
        if (!task) {
          ElMessage.warning('任务不存在');
          return;
        }

        const backendTaskId = task.backendTaskId || task.id;
        const filePath = await ExportResults(backendTaskId);
        if (filePath) {
          ElMessage.success(`导出成功：${filePath}`);
        } else {
          ElMessage.error('导出失败');
        }
      } catch (error) {
        ElMessage.error('导出失败: ' + error);
      }
    },

    // 加载提取规则
    async loadExtractRules() {
      try {
        const rules = await GetExtractRules();
        this.extractRules = rules || {};
        this.updateExtractRulesList();
      } catch (error) {
        console.error('加载提取规则失败:', error);
        ElMessage.error('加载提取规则失败');
      }
    },

    // 更新提取规则列表（用于表格显示）
    updateExtractRulesList() {
      this.extractRulesList = Object.keys(this.extractRules || {}).map(name => ({
        name,
        pattern: this.extractRules[name],
        editing: false,
        editPattern: this.extractRules[name]
      }));
    },

    // 保存所有提取规则
    async saveExtractRules() {
      try {
        // 从列表中更新规则对象
        const rules = {};
        this.extractRulesList.forEach(item => {
          if (item.name && item.pattern) {
            rules[item.name] = item.pattern;
          }
        });
        this.extractRules = rules;
        await SaveExtractRules(rules);
        ElMessage.success('提取规则已保存');
        this.showExtractRulesConfig = false;
      } catch (error) {
        console.error('保存提取规则失败:', error);
        ElMessage.error('保存提取规则失败: ' + error);
      }
    },

    // 添加提取规则
    handleAddExtractRule() {
      ElMessageBox.prompt('请输入规则名称和正则表达式（用冒号分隔）', '添加提取规则', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputPlaceholder: '例如：custom_rule:正则表达式'
      }).then(({ value }) => {
        const parts = value.split(':');
        if (parts.length < 2) {
          ElMessage.warning('格式错误，请输入：规则名称:正则表达式');
          return;
        }
        const name = parts[0].trim();
        const pattern = parts.slice(1).join(':').trim();
        if (!name || !pattern) {
          ElMessage.warning('规则名称和正则表达式不能为空');
          return;
        }
        if (this.extractRules[name]) {
          ElMessage.warning('规则名称已存在');
          return;
        }
        this.extractRules[name] = pattern;
        this.updateExtractRulesList();
        ElMessage.success('规则已添加');
      }).catch(() => {});
    },

    // 开始编辑规则
    startExtractRuleEdit(row) {
      row.editing = true;
      row.editPattern = row.pattern;
    },

    // 保存规则编辑
    async saveExtractRuleEdit(row) {
      if (!row.editPattern || !row.editPattern.trim()) {
        ElMessage.warning('正则表达式不能为空');
        row.editPattern = row.pattern;
        row.editing = false;
        return;
      }
      row.pattern = row.editPattern.trim();
      row.editing = false;
      // 立即保存到后端
      try {
        await UpdateExtractRule(row.name, row.pattern);
        this.extractRules[row.name] = row.pattern;
        ElMessage.success('规则已更新');
      } catch (error) {
        ElMessage.error('更新规则失败: ' + error);
        // 恢复原值
        row.pattern = this.extractRules[row.name];
        row.editPattern = row.pattern;
      }
    },

    // 删除提取规则
    async handleDeleteExtractRule(ruleName) {
      try {
        await ElMessageBox.confirm(`确定要删除规则「${ruleName}」吗？`, '确认删除', { type: 'warning' });
        await DeleteExtractRule(ruleName);
        delete this.extractRules[ruleName];
        this.updateExtractRulesList();
        ElMessage.success('规则已删除');
      } catch (error) {
        if (error !== 'cancel') {
          console.error('删除规则失败:', error);
          ElMessage.error('删除规则失败: ' + error);
        }
      }
    },

    // 重置为默认规则
    async resetExtractRules() {
      try {
        await ElMessageBox.confirm('确定要重置为默认规则吗？当前规则将被覆盖。', '确认重置', { type: 'warning' });
        await this.loadExtractRules();
        this.extractRulesCurrentPage = 1; // 重置到第一页
        ElMessage.success('已重置为默认规则');
      } catch (error) {
        if (error !== 'cancel') {
          console.error('重置规则失败:', error);
        }
      }
    },

    // 分页事件处理
    handleExtractRulesPageSizeChange(val) {
      this.extractRulesPageSize = val;
      this.extractRulesCurrentPage = 1; // 改变每页数量时重置到第一页
    },

    handleExtractRulesPageChange(val) {
      this.extractRulesCurrentPage = val;
    },

    // 获取任务标签页的分页配置
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
      if (!task.packetColFilters) {
        task.packetColFilters = { url: '', method: '', statusCode: '', duration: '', extracted: '' }
      }
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
    handleBrowserColFilter(taskId, tabName) {
      const c = this.getPaginationConfig(taskId, tabName)
      c.page = 1
    },
    filterPacketRowsForCols(task, rows) {
      if (!task || !Array.isArray(rows)) return []
      this.ensureTaskColFilters(task)
      const f = task.packetColFilters
      const u = (f.url || '').trim().toLowerCase()
      const m = (f.method || '').trim().toLowerCase()
      const s = (f.statusCode || '').trim().toLowerCase()
      const d = (f.duration || '').trim().toLowerCase()
      const e = (f.extracted || '').trim().toLowerCase()
      return rows.filter(row => {
        if (u && !(String(row.url || '').toLowerCase().includes(u))) return false
        if (m && !(String(row.method || '').toLowerCase().includes(m))) return false
        const sc = row.statusCode ?? row.status_code
        if (s && !(String(sc ?? '').toLowerCase().includes(s))) return false
        if (d && !(String(row.duration ?? '').toLowerCase().includes(d))) return false
        if (e) {
          const cnt = Array.isArray(row.extractedInfo) ? row.extractedInfo.length : 0
          if (!String(cnt).includes(e) && !(`提取${cnt}`.toLowerCase().includes(e))) {
            const infoStr = JSON.stringify(row.extractedInfo || []).toLowerCase()
            if (!infoStr.includes(e)) return false
          }
        }
        return true
      })
    },
    getPacketFilteredTotal(task) {
      const base = this.filteredPacketLogs(task) || []
      return this.filterPacketRowsForCols(task, base).length
    },
    getPaginatedPacketRows(task) {
      const base = this.filteredPacketLogs(task) || []
      const filtered = this.filterPacketRowsForCols(task, base)
      const taskId = task.id
      const tabName = 'packets'
      const config = this.getPaginationConfig(taskId, tabName)
      const start = (config.page - 1) * config.pageSize
      const end = start + config.pageSize
      return filtered.slice(start, end)
    },
    handlePacketColFilter(taskId) {
      const c = this.getPaginationConfig(taskId, 'packets')
      c.page = 1
    },
    // 获取分页后的数据列表
    getPaginatedData(taskId, tabName, dataList) {
      if (!dataList || !Array.isArray(dataList)) return []
      const list = this.applyValueColFilter(taskId, tabName, dataList)
      const config = this.getPaginationConfig(taskId, tabName)
      const start = (config.page - 1) * config.pageSize
      const end = start + config.pageSize
      return list.slice(start, end)
    },

    // 分页大小改变处理
    handlePageSizeChange(taskId, tabName, pageSize) {
      const config = this.getPaginationConfig(taskId, tabName);
      config.pageSize = pageSize;
      config.page = 1; // 改变每页数量时重置到第一页
    },

    // 分页页码改变处理
    handlePageChange(taskId, tabName, page) {
      const config = this.getPaginationConfig(taskId, tabName);
      config.page = page;
    },

    // 顶层结果标签：endpoints | frontend | secrets | packets（兼容旧版扁平 name）
    getActiveResultTab(taskId) {
      let t = this.activeResultTab[taskId];
      if (!t) return 'endpoints';
      if (['endpoints', 'frontend', 'secrets', 'packets'].includes(t)) return t;
      const ep = new Set(['urls', 'api', 'domains', 'ips', 'ipports', 'paths', 'incompletepaths']);
      if (ep.has(t)) {
        this.resultSubTabEndpoints[taskId] = this.resultSubTabEndpoints[taskId] || t;
        this.activeResultTab[taskId] = 'endpoints';
        return 'endpoints';
      }
      const fe = new Set(['vue', 'jsfiles', 'staticfiles']);
      if (fe.has(t)) {
        this.resultSubTabFrontend[taskId] = this.resultSubTabFrontend[taskId] || t;
        this.activeResultTab[taskId] = 'frontend';
        return 'frontend';
      }
      const sec = new Set(['jwt', 'mobile', 'email', 'securitykey', 'password', 'username', 'aksk', 'appid', 'accesskey', 'apikey', 'secretkey', 'idcard']);
      if (sec.has(t)) {
        this.resultSubTabSecrets[taskId] = this.resultSubTabSecrets[taskId] || t;
        this.activeResultTab[taskId] = 'secrets';
        return 'secrets';
      }
      return 'endpoints';
    },

    setActiveResultTab(taskId, tabName) {
      this.activeResultTab[taskId] = tabName;
    },

    ensureResultSubTabs(taskId) {
      if (!this.resultSubTabEndpoints[taskId]) this.resultSubTabEndpoints[taskId] = 'urls';
      if (!this.resultSubTabFrontend[taskId]) this.resultSubTabFrontend[taskId] = 'vue';
      if (!this.resultSubTabSecrets[taskId]) this.resultSubTabSecrets[taskId] = 'jwt';
    },

    getResultSubTabEndpoints(taskId) {
      return this.resultSubTabEndpoints[taskId] || 'urls';
    },

    setResultSubTabEndpoints(taskId, name) {
      this.resultSubTabEndpoints[taskId] = name;
      this.activeResultTab[taskId] = 'endpoints';
    },

    getResultSubTabFrontend(taskId) {
      return this.resultSubTabFrontend[taskId] || 'vue';
    },

    setResultSubTabFrontend(taskId, name) {
      this.resultSubTabFrontend[taskId] = name;
      this.activeResultTab[taskId] = 'frontend';
    },

    getResultSubTabSecrets(taskId) {
      return this.resultSubTabSecrets[taskId] || 'jwt';
    },

    setResultSubTabSecrets(taskId, name) {
      this.resultSubTabSecrets[taskId] = name;
      this.activeResultTab[taskId] = 'secrets';
    },

    // 提取信息对话框分页处理
    handleExtractedInfoPageSizeChange(val) {
      this.extractedInfoPageSize = val;
      this.extractedInfoCurrentPage = 1;
    },

    handleExtractedInfoPageChange(val) {
      this.extractedInfoCurrentPage = val;
    },


    // 任务管理相关方法
    async handleHistoryDialogOpened() {
      // 对话框打开时，先使用缓存快速显示，然后异步刷新（如果需要）
      const now = Date.now();
      if (this.tasksCache && (now - this.tasksCacheTime) < this.TASKS_CACHE_DURATION) {
        // 使用缓存快速显示
        this.allTasks = this.tasksCache;
        // 后台异步刷新，不阻塞UI
        this.refreshTasks(false).catch(e => console.error('后台刷新任务失败:', e));
      } else {
        // 缓存过期或不存在，立即加载
        await this.refreshTasks(false);
      }
    },

    handleHistoryDialogClosed() {
      // 对话框关闭时不清空任务列表（保留缓存以便下次快速显示）
      // 只重置加载状态
      this.tasksLoading = false;
    },

    async refreshTasks(force = false) {
      // 使用缓存机制，避免频繁请求
      const now = Date.now();
      if (!force && this.tasksCache && (now - this.tasksCacheTime) < this.TASKS_CACHE_DURATION) {
        this.allTasks = this.tasksCache;
        return;
      }
      
      // 如果已经在加载中，避免重复加载
      if (this.tasksLoading) {
        return;
      }
      
      this.tasksLoading = true;
      try {
        // 优化：尝试使用 GetTasksBasicInfo API（如果可用）
        // 如果 API 不存在，则使用 GetAllTasks 但只提取基本信息
        let tasks = [];
        
        try {
          // 优先使用优化后的 API（如果已生成绑定）
          if (window.go?.browserScan?.BrowserScan?.GetTasksBasicInfo) {
            tasks = await window.go.browserScan.BrowserScan.GetTasksBasicInfo();
          } else {
            // 降级方案：使用 GetAllTasks，但只提取基本信息字段
            const allTasks = await GetAllTasks();
            tasks = (allTasks || []).map(task => ({
              id: task.id || '',
              url: task.url || '',
              status: task.status || 'unknown',
              start_time: task.start_time || task.startTime || null,
              end_time: task.end_time || task.endTime || null
            }));
          }
        } catch (apiError) {
          // 如果 GetTasksBasicInfo 不存在，使用 GetAllTasks
          const allTasks = await GetAllTasks();
          tasks = (allTasks || []).map(task => ({
            id: task.id || '',
            url: task.url || '',
            status: task.status || 'unknown',
            start_time: task.start_time || task.startTime || null,
            end_time: task.end_time || task.endTime || null
          }));
        }
        
        // 只提取任务的基本信息字段，不加载完整数据（results、packetLogs等）
        // 这样可以大幅减少内存占用和加载时间
        const sortedTasks = tasks.map(task => ({
          id: task.id || '',
          url: task.url || '',
          status: task.status || 'unknown',
          start_time: task.start_time || null,
          end_time: task.end_time || null
        })).sort((a, b) => {
          // 按开始时间倒序排序
          const timeA = a.start_time ? new Date(a.start_time).getTime() : 0;
          const timeB = b.start_time ? new Date(b.start_time).getTime() : 0;
          return timeB - timeA;
        });
        
        // 更新缓存
        this.tasksCache = sortedTasks;
        this.tasksCacheTime = now;
        this.allTasks = sortedTasks;
        
        // 重置分页到第一页
        this.taskPagination.currentPage = 1;
      } catch (error) {
        console.error('加载任务列表失败:', error);
        ElMessage.error('加载任务列表失败: ' + error);
        this.allTasks = [];
      } finally {
        this.tasksLoading = false;
      }
    },

    async loadTaskData(taskId) {
      try {
        // 先从后端加载完整任务数据
        const task = await GetTask(taskId);
        if (!task) {
          ElMessage.warning('任务不存在');
          // 从任务列表中移除不存在的任务
          this.allTasks = this.allTasks.filter(t => t.id !== taskId);
          // 更新缓存，移除不存在的任务
          if (this.tasksCache) {
            this.tasksCache = this.tasksCache.filter(t => t.id !== taskId);
          }
          return;
        }

        // 检查是否已存在该任务
        const existingTask = this.tasks.find(t => t.id === taskId || t.backendTaskId === taskId);
        if (existingTask) {
          // 如果已存在，更新数据
          Object.assign(existingTask, {
            id: task.id,
            name: task.url || `任务-${task.id}`,
            url: task.url,
            status: task.status,
            backendTaskId: task.id,
            config: {
              waitTime: task.config?.waitTime || task.config?.wait_time || 5,
              apiRoutes: task.config?.apiRoutes || task.config?.api_routes || [],
              globalApiRoutes: task.config?.globalApiRoutes || task.config?.global_api_routes || [],
              allowList: task.config?.allowList || task.config?.allow_list || [],
              autoTestApi: task.config?.autoTestApi !== undefined ? task.config.autoTestApi : (task.config?.auto_test_api || false),
              testMethods: task.config?.testMethods || task.config?.test_methods || ['GET'],
              workerNum: task.config?.workerNum || task.config?.worker_num || 10
            },
            results: this.formatResults(task.results || {}),
            // 优化：限制数据包日志数量为500条（保留最新的）以节省内存
            packetLogs: (task.packet_logs || task.packetLogs || []).slice(-500).map(log => ({
              url: log.url || '',
              method: log.method || '',
              statusCode: log.status_code || log.statusCode || 0,
              duration: log.duration || '',
              // 优化：截断过长的请求/响应包
              requestPacket: (log.request_packet || log.requestPacket || '').substring(0, 1000),
              responsePacket: (log.response_packet || log.responsePacket || '').substring(0, 1000),
              extractedInfo: log.extracted_info || log.extractedInfo || []
            }))
          });
          // 重置分页配置到第一页
          this.resetTaskPagination(existingTask.id);
          this.ensureResultSubTabs(existingTask.id);
          // 切换到该任务
          this.activeTaskId = existingTask.id;
          ElMessage.success('任务数据已加载');
        } else {
          // 如果不存在，创建新任务
          const newTask = {
            id: task.id,
            name: task.url || `任务-${task.id}`,
            url: task.url,
            status: task.status,
            backendTaskId: task.id,
            config: {
              waitTime: task.config?.waitTime || task.config?.wait_time || 5,
              apiRoutes: task.config?.apiRoutes || task.config?.api_routes || [],
              globalApiRoutes: task.config?.globalApiRoutes || task.config?.global_api_routes || [],
              allowList: task.config?.allowList || task.config?.allow_list || [],
              autoTestApi: task.config?.autoTestApi !== undefined ? task.config.autoTestApi : (task.config?.auto_test_api || false),
              testMethods: task.config?.testMethods || task.config?.test_methods || ['GET'],
              workerNum: task.config?.workerNum || task.config?.worker_num || 10
            },
            results: this.formatResults(task.results || {}),
            // 优化：限制数据包日志数量为500条（保留最新的）以节省内存
            packetLogs: (task.packet_logs || task.packetLogs || []).slice(-500).map(log => ({
              url: log.url || '',
              method: log.method || '',
              statusCode: log.status_code || log.statusCode || 0,
              duration: log.duration || '',
              // 优化：截断过长的请求/响应包
              requestPacket: (log.request_packet || log.requestPacket || '').substring(0, 1000),
              responsePacket: (log.response_packet || log.responsePacket || '').substring(0, 1000),
              extractedInfo: log.extracted_info || log.extractedInfo || []
            }))
          };
          this.tasks.push(newTask);
          // 重置分页配置到第一页
          this.resetTaskPagination(newTask.id);
          this.activeResultTab[newTask.id] = 'endpoints';
          this.ensureResultSubTabs(newTask.id);
          this.activeTaskId = newTask.id;
          ElMessage.success('任务数据已加载');
        }

        // 关闭任务管理对话框
        this.historyDialog = false;
      } catch (error) {
        console.error('加载任务数据失败:', error);
        ElMessage.error('加载任务数据失败: ' + error);
      }
    },

    async deleteTaskFromHistory(taskId) {
      try {
        await ElMessageBox.confirm('确定要删除该任务吗？此操作不可恢复。', '确认删除', { type: 'warning' });
        await DeleteTask(taskId);
        
        // 从前端任务列表中移除
        this.tasks = this.tasks.filter(t => {
          if (t.id === taskId || t.backendTaskId === taskId) {
            delete this.activeResultTab[t.id];
            delete this.paginationConfig[t.id];
            delete this.resultSubTabEndpoints[t.id];
            delete this.resultSubTabFrontend[t.id];
            delete this.resultSubTabSecrets[t.id];
            return false;
          }
          return true;
        });

        // 从任务管理列表中移除（立即更新UI）
        this.allTasks = this.allTasks.filter(t => t.id !== taskId);
        // 同时更新缓存
        if (this.tasksCache) {
          this.tasksCache = this.tasksCache.filter(t => t.id !== taskId);
        }
        
        // 强制刷新任务列表（确保数据一致性）
        await this.refreshTasks(true);
        
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
        const taskIds = this.allTasks.map(t => t.id);
        for (const taskId of taskIds) {
          try {
            await DeleteTask(taskId);
          } catch (e) {
            console.error(`删除任务 ${taskId} 失败:`, e);
          }
        }

        // 清空前端任务列表（只保留pending任务）
        this.tasks = this.tasks.filter(t => t.status === 'pending' && !t.backendTaskId);
        
        // 清空任务管理列表和缓存（参考漏洞扫描的实现）
        this.allTasks = [];
        this.tasksCache = null;
        this.tasksCacheTime = 0;
        
        // 刷新任务列表
        await this.refreshTasks(true);
        
        ElMessage.success('所有任务已清除');
      } catch (error) {
        if (error !== 'cancel') {
          console.error('清除任务失败:', error);
          ElMessage.error('清除任务失败: ' + error);
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
        'pending': 'info'
      };
      return statusMap[status] || 'info';
    },

    getStatusText(status) {
      const statusMap = {
        'running': '运行中',
        'completed': '已完成',
        'stopped': '已停止',
        'error': '错误',
        'pending': '待开始'
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

    // 重置任务的所有分页配置到第一页
    resetTaskPagination(taskId) {
      // 重置结果标签页的分页
      const resultTabs = ['urls', 'api', 'domains', 'ips', 'ipports', 'paths', 'incompletepaths', 'vue', 'jsfiles', 'staticfiles', 'algorithms', 'jwt', 'mobile', 'email', 'securitykey', 'password', 'username', 'aksk', 'appid', 'accesskey', 'apikey', 'secretkey', 'idcard', 'packets'];
      resultTabs.forEach(tabName => {
        const config = this.getPaginationConfig(taskId, tabName);
        config.page = 1;
      });
    }
  },
  watch: {
    showExtractRulesConfig(newVal) {
      if (newVal) {
        this.loadExtractRules();
        this.extractRulesCurrentPage = 1; // 打开对话框时重置到第一页
      }
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
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  align-items: start;
}

.status-filter-bar {
  margin-bottom: 10px;
  padding: 10px;
  background: #f5f7fa;
  border-radius: 4px;
  border: 1px solid #ebeef5;
  
  :deep(.el-checkbox-group) {
    display: flex;
    gap: 15px;
    flex-wrap: wrap;
  }
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

.jsfinder-nested-tabs {
  margin-top: 2px;
}

.jsfinder-nested-tabs :deep(.el-tabs__header) {
  margin-bottom: 8px;
}

.vue-scan-block {
  padding: 8px 4px;
}

.vue-scan-alert {
  margin-bottom: 12px;
}

.vue-scan-desc {
  margin-bottom: 12px;
}

.vue-scan-actions {
  margin: 10px 0 14px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.vue-subtitle {
  font-size: 13px;
  font-weight: 600;
  margin: 10px 0 6px;
  color: #303133;
}

.vue-scan-err {
  color: #f56c6c;
  word-break: break-all;
}

.result-toolbar {
  margin-bottom: 10px;
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

.log-container {
  padding: 10px;
  display: flex;
  flex-direction: column;
}

.log-toolbar {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.log-empty {
  text-align: center;
  color: #909399;
  padding: 20px;
}

.log-entry {
  padding: 5px 0;
  font-family: monospace;
  font-size: 12px;
}

.log-time {
  color: #909399;
  margin-right: 10px;
}

.log-level {
  margin-right: 10px;
  font-weight: bold;
}

.log-entry.info .log-level {
  color: #409eff;
}

.log-entry.warning .log-level {
  color: #e6a23c;
}

.log-entry.error .log-level {
  color: #f56c6c;
}

.log-message {
  color: #303133;
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

// 提取规则管理样式
.extract-rules-config {
  .rules-toolbar {
    margin-bottom: 15px;
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  .rules-table {
    margin-bottom: 15px;
  }

  .rules-pagination {
    display: flex;
    justify-content: flex-end;
    padding-top: 10px;
    border-top: 1px solid #ebeef5;
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
