<template>
  <div class="hostscan-container">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar">
      <div class="toolbar-content">
        <div class="toolbar-left">
          <div class="module-title">
            <el-icon class="title-icon"><Connection /></el-icon>
            <span class="title-text">主机扫描（代理扫描暂时不准确，建议使用直连模式）</span>
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

          <!-- 配置区域 -->
          <div class="config-section">
            <el-card shadow="never">
              <div class="config-layout">
                <!-- 左侧：输入框和按钮 -->
                <div class="target-input-section">
                  <div class="target-label">目标主机 / IP</div>
                  <IsolatedTextarea
                    :ref="(el) => setIsolatedRef('targets-' + tab.id, el)"
                    v-model="tab.targetsText"
                    :rows="4"
                    placeholder="每行一个目标，例如：&#10;192.168.1.1&#10;192.168.1.1/24&#10;example.com"
                    size="default"
                    class="target-textarea"
                  />
                  <div class="target-actions">
                    <el-button
                      :type="tab.scanning ? 'danger' : 'primary'"
                      :icon="tab.scanning ? VideoPause : VideoPlay"
                      @click="toggleScan(tab)"
                      :loading="tab.loading && !tab.scanning"
                      size="default"
                      style="flex: 1"
                    >
                      {{ tab.scanning ? '停止扫描' : '开始扫描' }}
                    </el-button>
                  </div>
                  <div class="quick-stats">
                    <div class="quick-stat-item">
                      <span class="label">{{ tab.stats?.scanPhase === 'alive_probe' ? '存活探测' : '已扫端口' }}</span>
                      <span class="value">{{ (tab.stats && tab.stats.checked) || 0 }}/{{ (tab.stats && tab.stats.totalPorts) || 0 }}</span>
                    </div>
                    <div class="quick-stat-item success">
                      <span class="label">开放</span>
                      <span class="value">{{ (tab.stats && tab.stats.openCount) || 0 }}</span>
                    </div>
                    <div class="quick-stat-item">
                      <span class="label">速度</span>
                      <span class="value">{{ (tab.stats && tab.stats.portsPerSecond) ? tab.stats.portsPerSecond.toFixed(1) : '0.0' }}{{ tab.stats?.scanPhase === 'alive_probe' ? ' IP' : ' 端口' }}/秒</span>
                    </div>
                  </div>
                  <div class="alive-stats" v-if="tab.aliveCheckResult">
                    <span class="alive-label">存活数量:</span>
                    <span class="alive-value">
                      {{ tab.aliveCheckResult.aliveIPs }}/{{ tab.aliveCheckResult.totalIPs }}
                      <span v-if="tab.aliveCheckResult.domainCount > 0">(域名: {{ tab.aliveCheckResult.domainCount }})</span>
                    </span>
                  </div>
                </div>

                <!-- 中间：端口配置 -->
                <div class="port-config-section">
                  <div class="config-group">
                    <div class="config-row">
                      <div class="config-item full-width">
                        <label class="config-label">端口预设</label>
                        <div class="port-preset-options">
                          <el-radio-group v-model="tab.config.portPreset" size="default" @change="handlePortPresetChange(tab)">
                            <el-radio label="custom">自定义端口</el-radio>
                            <el-radio label="all">全端口 (1-65535)</el-radio>
                            <el-radio label="top100">Top 100 端口</el-radio>
                            <el-radio label="top1000">Top 1000 端口</el-radio>
                          </el-radio-group>
                        </div>
                      </div>
                      <div class="config-item full-width">
                        <label class="config-label">过滤端口（多选）</label>
                        <el-checkbox-group v-model="tab.config.excludePorts" size="default">
                          <el-checkbox label="printer">打印机端口 (515,631,9100-9109)</el-checkbox>
                        </el-checkbox-group>
                      </div>
                      <div class="config-item full-width">
                        <label class="config-label">端口范围</label>
                        <el-input
                          v-model="tab.config.ports"
                          placeholder="例如：80,443,8080-8090（可与预设合并）"
                          size="default"
                        />
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 右侧：配置选项 -->
                <div class="config-section-right">
                  <div class="config-group">
                    <div class="config-row">
                      <div class="config-item">
                        <label class="config-label">线程数</label>
                        <el-input-number
                          v-model="tab.config.threads"
                          :min="1"
                          :max="2000"
                          controls-position="right"
                          size="default"
                          style="width: 100%"
                        />
                      </div>
                      <div class="config-item">
                        <label class="config-label">超时(秒)</label>
                        <el-input-number
                          v-model="tab.config.timeout"
                          :min="1"
                          :max="10"
                          controls-position="right"
                          size="default"
                          style="width: 100%"
                        />
                      </div>
                    </div>

                    <div class="config-row switch-row">
                      <div class="config-item">
                        <el-checkbox v-model="tab.config.checkAlive" @change="handleAliveCheckChange(tab)">IP存活检测</el-checkbox>
                      </div>
                      <div class="config-item" v-if="tab.config.checkAlive">
                        <el-button size="small" @click="showAliveConfigDialog = true; currentAliveTab = tab" plain>
                          配置存活检测
                        </el-button>
                      </div>
                      <div class="config-item">
                        <el-checkbox v-model="tab.config.enablePortFingerprint">端口指纹识别</el-checkbox>
                      </div>
                      <div class="config-item">
                        <el-checkbox v-model="tab.config.enableFingerprint">指纹识别联动</el-checkbox>
                      </div>
                      <div class="config-item">
                        <el-checkbox v-model="tab.config.enableDirsearch">目录扫描联动</el-checkbox>
                      </div>
                      <div class="config-item">
                        <el-checkbox v-model="tab.config.enableVulnScan">漏洞指纹扫描</el-checkbox>
                      </div>
                      <div class="config-item">
                        <el-checkbox v-model="tab.config.enableCrack">密码爆破</el-checkbox>
                      </div>
                      <div class="config-item">
                        <el-button
                          :icon="Setting"
                          @click="showAdvancedDialog = true; currentAdvancedTab = tab"
                          size="small"
                          plain
                        >
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
                  <div class="progress-head">
                    <span class="progress-title">扫描进度</span>
                    <el-tag size="small" :type="tab.stats?.scanPhase === 'alive_probe' ? 'warning' : 'primary'">
                      {{ tab.stats?.scanPhase === 'alive_probe' ? 'IP存活探测' : '端口扫描' }}
                    </el-tag>
                  </div>
                  <div class="progress-bar-row">
                    <el-progress
                      :percentage="Math.round(tab.stats?.progressPercent || 0)"
                      :stroke-width="10"
                      :show-text="false"
                      :color="(p) => p >= 100 ? '#67c23a' : '#409eff'"
                    />
                    <span class="progress-percent">{{ Math.round(tab.stats?.progressPercent || 0) }}%</span>
                  </div>
                  <div class="progress-grid">
                    <div class="pg-item">
                      <span class="pg-label">已用时</span>
                      <span class="pg-value">{{ formatDuration(tab.stats?.elapsedSeconds || 0) }}</span>
                    </div>
                    <div class="pg-item" v-if="tab.stats?.estimatedSeconds > 0">
                      <span class="pg-label">预计剩余</span>
                      <span class="pg-value">{{ formatDuration(tab.stats?.estimatedSeconds || 0) }}</span>
                    </div>
                    <div class="pg-item">
                      <span class="pg-label">速度</span>
                      <span class="pg-value">{{ formatNumber(tab.stats?.portsPerSecond || 0) }} {{ tab.stats?.scanPhase === 'alive_probe' ? 'IP' : '端口' }}/秒</span>
                    </div>
                    <div class="pg-item">
                      <span class="pg-label">已检查</span>
                      <span class="pg-value">{{ formatNumber(tab.stats?.checked || 0) }} / {{ formatNumber(tab.stats?.totalPorts || 0) }}</span>
                    </div>
                    <div class="pg-item">
                      <span class="pg-label">开放</span>
                      <span class="pg-value">{{ tab.stats?.openCount || 0 }}</span>
                    </div>
                    <div class="pg-item">
                      <span class="pg-label">关闭/过滤</span>
                      <span class="pg-value">{{ tab.stats?.closedOrFiltered || 0 }}</span>
                    </div>
                    <div class="pg-item" v-if="tab.stats?.scanPhase === 'alive_probe' && tab.stats?.portScanTotalPorts">
                      <span class="pg-label">端口任务数</span>
                      <span class="pg-value">{{ formatNumber(tab.stats.portScanTotalPorts) }}</span>
                    </div>
                  </div>
                  <div v-if="tab.currentTarget" class="progress-target">
                    <el-icon><Location /></el-icon>
                    <span>当前目标：{{ tab.currentTarget }}</span>
                  </div>
                </el-card>
              </div>
            </transition>
          </div>

          <!-- 结果区域 -->
          <div class="results-section">
            <el-card shadow="never" class="results-card">
              <template #header>
                <div class="results-header">
                  <div class="results-title-group">
                    <span class="results-title">主机扫描结果（仅开放端口）</span>
                    <el-tag type="success" size="small">{{ tab.results.length }} 条</el-tag>
                  </div>
                  <div class="results-actions">
                    <el-button :icon="Download" :disabled="tab.results.length === 0" @click="exportToExcel(tab)" size="default">
                      导出Excel
                    </el-button>
                    <el-button :icon="Delete" :disabled="tab.results.length === 0" @click="clearResults(tab)" size="default" plain>
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
                  class="results-table"
                  @row-contextmenu="(row, column, event) => handleRowContextMenu(row, column, event, tab)"
                >
            <el-table-column type="index" width="60" label="#" align="center" />
            <el-table-column prop="target" min-width="150" align="left">
              <template #header>
                <TableColumnFilterHeader
                  label="Host"
                  v-model="tab.columnFilters.target"
                  :popover-width="240"
                  sortable
                  :sort-asc-active="tab.sortField === 'target' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'target' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'target', 'asc')"
                  @sort-desc="setTabSort(tab, 'target', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">
                <el-tooltip
                    :content="row.target || ''"
                    placement="top"
                    :disabled="!row.target || row.target.length <= 20"
                    effect="dark"
                >
                  <div class="table-cell-text">
                    {{ row.target || '-' }}
                  </div>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column prop="port" width="92" align="center">
              <template #header>
                <TableColumnFilterHeader
                  label="Port"
                  v-model="tab.columnFilters.port"
                  :popover-width="200"
                  sortable
                  :sort-asc-active="tab.sortField === 'port' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'port' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'port', 'asc')"
                  @sort-desc="setTabSort(tab, 'port', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">{{ row.port }}</template>
            </el-table-column>
            <el-table-column prop="service" min-width="100" align="center">
              <template #header>
                <TableColumnFilterHeader
                  label="Proto"
                  v-model="tab.columnFilters.service"
                  :popover-width="200"
                  sortable
                  :sort-asc-active="tab.sortField === 'service' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'service' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'service', 'asc')"
                  @sort-desc="setTabSort(tab, 'service', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">
                <el-tooltip
                    :content="getServiceDisplay(row)"
                    placement="top"
                    :disabled="!getServiceDisplay(row) || getServiceDisplay(row).length <= 15"
                    effect="dark"
                >
                  <div class="table-cell-text">
                    {{ getServiceDisplay(row) }}
                  </div>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column min-width="210" align="left">
              <template #header>
                <TableColumnFilterHeader
                  label="Target"
                  v-model="tab.columnFilters.targetUrl"
                  :popover-width="280"
                  sortable
                  :sort-asc-active="tab.sortField === 'targetUrl' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'targetUrl' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'targetUrl', 'asc')"
                  @sort-desc="setTabSort(tab, 'targetUrl', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">
                <el-tooltip
                    :content="row.protocol === 'http' || row.protocol === 'https' ? getTargetURL(row) : `${row.target}:${row.port}`"
                    placement="top"
                    effect="dark"
                >
                  <div class="url-cell">
                    <span 
                      v-if="row.protocol === 'http' || row.protocol === 'https'"
                      class="url-text clickable-url"
                      @click="openURL(row)"
                      :title="`点击打开: ${getTargetURL(row)}`"
                    >
                      {{ getTargetURL(row) }}
                    </span>
                    <span v-else class="url-text">{{ row.target }}:{{ row.port }}</span>
                  </div>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column min-width="300" align="left">
              <template #header>
                <TableColumnFilterHeader
                  label="Banner"
                  v-model="tab.columnFilters.banner"
                  :popover-width="280"
                  sortable
                  :sort-asc-active="tab.sortField === 'banner' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'banner' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'banner', 'asc')"
                  @sort-desc="setTabSort(tab, 'banner', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">
                <div class="banner-cell">
                  <div class="banner-tags">
                    <!-- 显示指纹识别的技术栈标签 -->
                    <el-tag
                      v-for="(tech, index) in (row.technologies || [])"
                      :key="'tech-' + index"
                      size="small"
                      type="success"
                      effect="plain"
                      class="banner-tag"
                    >
                      {{ tech }}
                    </el-tag>
                    <!-- 显示CMS信息 -->
                    <el-tag
                      v-if="row.cms"
                      size="small"
                      type="warning"
                      effect="plain"
                      class="banner-tag"
                    >
                      {{ row.cms }}
                    </el-tag>
                    <!-- 显示banner内容（用标签包裹，显示约50个字符） -->
                    <template v-if="row.banner && row.banner.trim()">
                      <el-tag
                        size="small"
                        type="info"
                        effect="plain"
                        class="banner-tag"
                      >
                        {{ truncateBanner(cleanBanner(row.banner, row.protocol), 35) }}
                      </el-tag>
                    </template>
                    <!-- 如果没有banner，尝试从service解析标签 -->
                    <template v-else-if="(!row.technologies || row.technologies.length === 0) && !row.cms">
                      <el-tag
                        v-for="(tag, index) in parseBannerTags(row.banner, row.service)"
                        :key="'banner-' + index"
                        size="small"
                        type="info"
                        effect="plain"
                        class="banner-tag"
                      >
                        {{ tag }}
                      </el-tag>
                      <span v-if="parseBannerTags(row.banner, row.service).length === 0" class="empty-banner">-</span>
                    </template>
                    <span v-else-if="(!row.technologies || row.technologies.length === 0) && !row.cms" class="empty-banner">-</span>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="title" min-width="480" align="left">
              <template #header>
                <TableColumnFilterHeader
                  label="Title"
                  v-model="tab.columnFilters.title"
                  :popover-width="280"
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
                    :disabled="!row.title || row.title.length <= 50"
                    effect="dark"
                >
                  <div class="table-cell-text">
                    <span v-if="row.title">{{ row.title }}</span>
                    <span v-else-if="row.protocol === 'http' || row.protocol === 'https'" class="title-none">None</span>
                    <span v-else>-</span>
                  </div>
                </el-tooltip>
              </template>
            </el-table-column>
            <el-table-column width="100" align="center">
              <template #header>
                <TableColumnFilterHeader
                  label="Code"
                  v-model="tab.columnFilters.code"
                  :popover-width="200"
                  sortable
                  :sort-asc-active="tab.sortField === 'code' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'code' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'code', 'asc')"
                  @sort-desc="setTabSort(tab, 'code', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">
                <span>{{ getStatusCode(row) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="location" min-width="96" align="left">
              <template #header>
                <TableColumnFilterHeader
                  label="地区"
                  v-model="tab.columnFilters.location"
                  :popover-width="220"
                  sortable
                  :sort-asc-active="tab.sortField === 'location' && tab.sortOrder === 'asc'"
                  :sort-desc-active="tab.sortField === 'location' && tab.sortOrder === 'desc'"
                  @sort-asc="setTabSort(tab, 'location', 'asc')"
                  @sort-desc="setTabSort(tab, 'location', 'desc')"
                  @filter-change="handleColumnFilterInput(tab)"
                />
              </template>
              <template #default="{ row }">
                <el-tooltip
                  v-if="row.location"
                  effect="dark"
                  placement="top"
                  :content="row.location"
                >
                  <span>{{ getCityFromLocation(row.location) }}</span>
                </el-tooltip>
                <span v-else>-</span>
              </template>
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
                />
              </div>
            </el-card>
          </div>
        </el-tab-pane>
      </el-tabs>
    </CustomTabs>

    <!-- 高级配置对话框 -->
    <el-dialog
      v-model="showAdvancedDialog"
      title="高级配置"
      width="700px"
      destroy-on-close
      @closed="handleHostscanAdvancedDialogClosed"
    >
      <div class="advanced-dialog-content" v-if="currentAdvancedTab">
        <div class="advanced-section">
          <h4 class="section-title">端口扫描配置</h4>
          <div class="advanced-row">
            <div class="advanced-item full-width">
              <el-checkbox v-model="currentAdvancedTab.config.enableFirewallDetection">
                启用防火墙检测（全端口疑似全存活时自动降级 Top100）
              </el-checkbox>
            </div>
          </div>
          <div class="advanced-row">
            <div class="advanced-item">
              <label class="config-label">检测采样端口数</label>
              <el-input-number
                v-model="currentAdvancedTab.config.firewallDetectSampleSize"
                :min="6"
                :max="14"
                :step="1"
                controls-position="right"
                :disabled="!currentAdvancedTab.config.enableFirewallDetection"
              />
            </div>
            <div class="advanced-item">
              <label class="config-label">触发阈值(%)</label>
              <el-input-number
                v-model="currentAdvancedTab.config.firewallDetectThresholdPercent"
                :min="60"
                :max="100"
                :step="5"
                controls-position="right"
                :disabled="!currentAdvancedTab.config.enableFirewallDetection"
              />
            </div>
          </div>
          <div class="advanced-row">
            <div class="advanced-item full-width">
              <el-checkbox
                v-model="autoFullPortRescanRemember"
                @change="saveAutoFullPortRescanPreference"
              >
                之后同类场景默认自动全端口重扫（不再弹窗）
              </el-checkbox>
            </div>
          </div>
          <div class="advanced-row">
            <div class="advanced-item full-width">
              <el-checkbox v-model="currentAdvancedTab.config.emitClosedPorts">
                将关闭端口实时推送到 UI（大网段/全端口扫描时勿开，易导致卡顿）
              </el-checkbox>
            </div>
          </div>
        </div>

        <div class="advanced-section">
          <h4 class="section-title">漏洞扫描配置</h4>
          <div class="advanced-row">
            <div class="advanced-item full-width">
              <el-checkbox v-model="currentAdvancedTab.config.vulnSkipWithoutTags">
                无指纹目标跳过漏洞扫描
              </el-checkbox>
            </div>
          </div>
        </div>

        <div class="advanced-section">
          <h4 class="section-title">密码爆破配置</h4>
          <div class="advanced-row">
            <div class="advanced-item">
              <el-checkbox v-model="currentAdvancedTab.config.crackBuiltInUsername">
                使用内置用户名字典
              </el-checkbox>
            </div>
            <div class="advanced-item">
              <el-checkbox v-model="currentAdvancedTab.config.crackBuiltInPassword">
                使用内置密码字典
              </el-checkbox>
            </div>
          </div>
          <div class="advanced-row">
            <div class="advanced-item">
              <label class="config-label">自定义用户名（每行一个，可选）</label>
              <IsolatedTextarea
                v-if="currentAdvancedTab"
                :ref="(el) => setIsolatedRef('hs-crack-user-' + currentAdvancedTab.id, el)"
                v-model="currentAdvancedTab.config.crackUsernames"
                :rows="4"
                placeholder="例如：&#10;admin&#10;root&#10;test"
              />
            </div>
            <div class="advanced-item">
              <label class="config-label">自定义密码（每行一个，可选）</label>
              <IsolatedTextarea
                v-if="currentAdvancedTab"
                :ref="(el) => setIsolatedRef('hs-crack-pwd-' + currentAdvancedTab.id, el)"
                v-model="currentAdvancedTab.config.crackPasswords"
                :rows="4"
                placeholder="例如：&#10;123456&#10;admin123&#10;P@ssw0rd"
              />
            </div>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- 任务管理对话框 -->
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
        <el-table
          :data="paginatedTasks"
          stripe
          border
          class="history-table"
          :height="400"
        >
          <el-table-column type="index" width="60" label="#" align="center" />
          <el-table-column prop="name" label="任务名称" min-width="200">
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
          <el-table-column prop="targetsText" label="目标" min-width="250">
            <template #default="{ row }">
              <el-tooltip 
                v-if="row.targetsText && row.targetsText.split('\n').length > 2" 
                :content="row.targetsText" 
                placement="top"
                :show-after="300"
                effect="dark"
              >
                <div class="targets-text-truncated">
                  {{ row.targetsText.split('\n').slice(0, 2).join(', ') }}...
                </div>
              </el-tooltip>
              <span v-else>{{ row.targetsText || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="getStatusTagType(row.status)">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <!-- <el-table-column label="结果数" width="100" align="center">
            <template #default="{ row }">
              {{ row.resultsCount !== undefined ? row.resultsCount : (Array.isArray(row.results) ? row.results.length : row.results || 0) }}
            </template>
          </el-table-column> -->
          <el-table-column prop="createdAt" label="创建时间" width="180" align="center">
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
        <div v-if="isWebService(selectedRow)" class="context-menu-item" @click="handleRunFingerprint">
          <el-icon><View /></el-icon>
          <span>指纹识别</span>
        </div>
        <div v-if="isWebService(selectedRow)" class="context-menu-item" @click="handleRunDirsearch">
          <el-icon><View /></el-icon>
          <span>目录扫描</span>
        </div>
        <div v-if="isWebService(selectedRow)" class="context-menu-item" @click="handleRunJsfinder">
          <el-icon><View /></el-icon>
          <span>JSFinder 接口探测</span>
        </div>
        <div v-if="isWebService(selectedRow)" class="context-menu-item" @click="handleRunVulnScan">
          <el-icon><View /></el-icon>
          <span>漏洞扫描</span>
        </div>
        <div v-if="isCrackableProtocol(selectedRow)" class="context-menu-item" @click="handleRunCrack">
          <el-icon><View /></el-icon>
          <span>密码爆破</span>
        </div>
        <div class="context-menu-divider" v-if="isWebService(selectedRow) || isCrackableProtocol(selectedRow)"></div>
        <div class="context-menu-item" @click="handleCopyRow">
          <el-icon><CopyDocument /></el-icon>
          <span>复制当前行</span>
        </div>
        <div class="context-menu-item" @click="handleCopyAll">
          <el-icon><CopyDocument /></el-icon>
          <span>复制所有内容</span>
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
      :title="`数据包查看 - ${selectedRow ? selectedRow.target + ':' + selectedRow.port : ''}`"
    />

    <!-- 存活检测配置对话框 -->
    <el-dialog
      v-model="showAliveConfigDialog"
      title="存活检测配置"
      width="600px"
      destroy-on-close
    >
      <div class="alive-config-dialog">
        <el-form :model="aliveConfig" label-width="120px" size="default">
          <el-form-item label="检测类型">
            <el-select v-model="aliveConfig.type" style="width: 100%">
              <el-option label="TCP端口检测" value="tcp_port" />
              <el-option label="ICMP检测" value="icmp" />
              <el-option label="HTTP检测" value="http" />
            </el-select>
          </el-form-item>
          
          <el-form-item v-if="aliveConfig.type === 'tcp_port'" label="检测端口">
            <el-input
              v-model="aliveConfig.portsText"
              placeholder="例如：80,443,22（多个端口用逗号分隔）"
            />
            <div class="form-tip">将尝试连接这些端口来检测主机是否存活</div>
          </el-form-item>
          
          <el-form-item v-if="aliveConfig.type === 'http'" label="检测URL">
            <el-input
              v-model="aliveConfig.url"
              placeholder="例如：http://{host}/ 或 http://{host}:8080/"
            />
            <div class="form-tip">{host} 将被替换为实际的主机地址</div>
          </el-form-item>
          
          <el-form-item label="超时时间(秒)">
            <el-input-number
              v-model="aliveConfig.timeout"
              :min="1"
              :max="30"
              controls-position="right"
              style="width: 100%"
            />
          </el-form-item>
        </el-form>
        
        <div class="dialog-footer">
          <el-button @click="showAliveConfigDialog = false">取消</el-button>
          <el-button type="primary" @click="saveAliveConfig">保存</el-button>
        </div>
      </div>
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
  VideoPlay,
  VideoPause,
  Download,
  Delete,
  View,
  CopyDocument,
  Link,
  Plus,
  Clock,
  Refresh,
  Setting,
  CircleCheck,
  Connection,
  Timer,
  Location
} from '@element-plus/icons-vue'
import { markRaw } from 'vue'
import { EventsOn, EventsOff, BrowserOpenURL } from '../../../wailsjs/runtime';
import { StartHostScan, StopHostScan } from '../../../wailsjs/go/hostscan/HostscanController'
import { SaveFile, GetAppPath, ReadFile } from '../../../wailsjs/go/controller/Note'
import { setItem, getItem, removeItem } from '../../utils/storageManager'
import { CreateTask as CreateBrowserScanTask } from '../../../wailsjs/go/browserScan/BrowserScan'
import {
  StartPocScan
} from '../../../wailsjs/go/webscan/PocController'
import { GetPocConfig } from '../../../wailsjs/go/webscan/PocConfigController'
import { GetGlobalProxy } from '../../../wailsjs/go/proxy/ProxyManager'
import { NewCrackScanenr, LoadUsernameDictFromFile, LoadPasswordDictFromFile } from '../../../wailsjs/go/crack/CrackController'
import { StartFingerprintScan } from '../../../wailsjs/go/finger/FingerprintController'
import * as XLSX from 'xlsx'
import PacketViewer from '../../components/PacketViewer.vue'
import CustomTabs from '../../components/CustomTabs.vue'
import TableColumnFilterHeader from '../../components/TableColumnFilterHeader.vue'
import IsolatedTextarea from '../../components/IsolatedTextarea.vue'

export default {
  name: 'HostscanView',
  components: {
    VideoPlay,
    VideoPause,
    Download,
    Delete,
    View,
    CopyDocument,
    PacketViewer,
    CustomTabs,
    TableColumnFilterHeader,
    IsolatedTextarea,
    Plus,
    Clock,
    Refresh,
    Setting,
    CircleCheck,
    Connection
  },
  data() {
    return {
      VideoPlay: markRaw(VideoPlay),
      VideoPause: markRaw(VideoPause),
      Download: markRaw(Download),
      Delete: markRaw(Delete),
      View: markRaw(View),
      CopyDocument: markRaw(CopyDocument),
      Plus: markRaw(Plus),
      Clock: markRaw(Clock),
      Refresh: markRaw(Refresh),
      Setting: markRaw(Setting),
      CircleCheck: markRaw(CircleCheck),
      Connection: markRaw(Connection),
      Timer: markRaw(Timer),
      Location: markRaw(Location),
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
      currentAliveTab: null,
      tableHeight: 420,
      contextMenuVisible: false,
      contextMenuPos: { x: 0, y: 0 },
      selectedRow: null,
      selectedTab: null,
      packetViewerVisible: false,
      isolatedTextareaRefs: {},
      showAliveConfigDialog: false,
      saveHistoryTimer: null, // 防抖定时器
      saveTasksTimer: null, // 防抖定时器
      saveResultsTimer: null, // 保存结果到文件的防抖定时器
      aliveConfig: {
        type: 'icmp',
        portsText: '80,443,22',
        url: 'http://{host}/',
        timeout: 3,
        enabled: true
      },
      // 标签页右键菜单
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: '',
      firewallNoticePrompted: new Set(),
      autoFullPortRescanRemember: false
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
    activeTabId(newVal) {
      if (newVal) {
        this.$nextTick(() => {
          this.calcTableHeight()
        })
      }
    }
  },
  mounted() {
    this.bindEvents()
    this.loadAutoFullPortRescanPreference()
    this.loadTasks()
    this.calcTableHeight()
    window.addEventListener('resize', this.calcTableHeight)
    window.addEventListener('cyberSabre-hostscan-fill', this.handleExternalHostscanFill)
    // 初始化时创建一个默认标签页
    if (this.scanTabs.length === 0) {
      this.createNewTab()
    }
    this.scanTabs.forEach(t => this.ensureTabColumnFilters(t))
  },
  beforeUnmount() {
    this.unbindEvents()
    window.removeEventListener('resize', this.calcTableHeight)
    window.removeEventListener('cyberSabre-hostscan-fill', this.handleExternalHostscanFill)
    // 清理防抖定时器
    if (this.saveHistoryTimer) {
      clearTimeout(this.saveHistoryTimer)
      this.saveHistoryTimer = null
    }
    if (this.saveTasksTimer) {
      clearTimeout(this.saveTasksTimer)
      this.saveTasksTimer = null
    }
    if (this.saveResultsTimer) {
      clearTimeout(this.saveResultsTimer)
      this.saveResultsTimer = null
    }
    // 确保在卸载前保存一次
    this._saveHistory()
    this._saveTasks()
  },
  methods: {
    setIsolatedRef(key, el) {
      if (el) {
        this.isolatedTextareaRefs[key] = el
      } else {
        delete this.isolatedTextareaRefs[key]
      }
    },
    flushHostscanAdvancedTextareas(tabId) {
      ;[`hs-crack-user-${tabId}`, `hs-crack-pwd-${tabId}`].forEach((k) => {
        this.isolatedTextareaRefs[k]?.flush?.()
      })
    },
    flushPendingHostscanTextareas(tab) {
      if (!tab) return
      this.isolatedTextareaRefs[`targets-${tab.id}`]?.flush?.()
      if (this.currentAdvancedTab && this.currentAdvancedTab.id === tab.id) {
        this.flushHostscanAdvancedTextareas(tab.id)
      }
    },
    handleHostscanAdvancedDialogClosed() {
      if (this.currentAdvancedTab) {
        this.flushHostscanAdvancedTextareas(this.currentAdvancedTab.id)
      }
    },
    // 标签页管理
    createNewTab() {
      const tabId = `tab-${++this.tabIdCounter}`
      const tab = {
        id: tabId,
        name: `主机扫描-${new Date().toLocaleTimeString()}`,
        targetsText: '',
        config: {
          ports: '80,443,8080-8090',
          portPreset: 'custom',
          checkAlive: true,
          aliveCheck: {
            type: 'icmp',
            ports: [80, 443, 22],
            url: 'http://{host}/',
            timeout: 3,
            enabled: true
          },
          randomizePorts: true,
          threads: 200,
          timeout: 3,
          enablePortFingerprint: true,
          emitClosedPorts: false,
          enableFingerprint: false,
          enableDirsearch: false,
          enableVulnScan: false,
          enableCrack: false,
          vulnSkipWithoutTags: false,
          crackBuiltInUsername: true,
          crackBuiltInPassword: true,
          crackUsernames: '',
          crackPasswords: '',
          excludePorts: [],
          enableFirewallDetection: true,
          disableFirewallDetection: false,
          firewallDetectSampleSize: 14,
          firewallDetectThresholdPercent: 90
        },
        scanning: false,
        loading: false,
        completed: false,
        results: [],
        stats: {
          totalPorts: 0,
          checked: 0,
          openCount: 0,
          closedOrFiltered: 0,
          portsPerSecond: 0
        },
        aliveCheckResult: null,
        currentTarget: '',
        currentPage: 1,
        pageSize: 10,
        createTime: Date.now(),
        columnFilters: {
          target: '',
          port: '',
          service: '',
          targetUrl: '',
          banner: '',
          title: '',
          code: '',
          location: ''
        },
        sortField: '',
        sortOrder: 'asc'
      }
      this.scanTabs.push(tab)
      this.activeTabId = tabId
      this.saveHistory()
      // 返回新建的标签对象，便于联动场景中进一步配置
      return tab
    },
    async removeTab(tabId) {
      const index = this.scanTabs.findIndex(tab => tab.id === tabId)
      if (index !== -1) {
        const tab = this.scanTabs[index]
        // 关闭标签页时，不停止扫描，允许后台继续运行
        // 重要：关闭标签页时，任务数据必须保留在任务管理中，不能删除
        if (tab.taskId) {
          const task = this.tasksLocal[tab.taskId]
          if (task) {
            // 保存完整的结果数组到任务中
            task.results = JSON.parse(JSON.stringify(tab.results || []))
            task.resultsCount = tab.results ? tab.results.length : 0
            task.stats = JSON.parse(JSON.stringify(tab.stats || {}))
            task.aliveCheckResult = tab.aliveCheckResult ? JSON.parse(JSON.stringify(tab.aliveCheckResult)) : null
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
        // 保存当前任务数据，但保持任务状态（如果正在扫描则保持为 running）
        // 重要：关闭标签页时，任务数据必须保留在任务管理中，不能删除
        if (tab.taskId) {
          const task = this.tasksLocal[tab.taskId]
          if (task) {
            // 保存完整的结果数组到任务中
            task.results = JSON.parse(JSON.stringify(tab.results || []))
            task.resultsCount = tab.results ? tab.results.length : 0
            task.stats = JSON.parse(JSON.stringify(tab.stats || {}))
            task.aliveCheckResult = tab.aliveCheckResult ? JSON.parse(JSON.stringify(tab.aliveCheckResult)) : null
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
        // 保存当前任务数据，但保持任务状态（如果正在扫描则保持为 running）
        // 重要：关闭标签页时，任务数据必须保留在任务管理中，不能删除
        if (tab.taskId) {
          const task = this.tasksLocal[tab.taskId]
          if (task) {
            // 保存完整的结果数组到任务中
            task.results = JSON.parse(JSON.stringify(tab.results || []))
            task.resultsCount = tab.results ? tab.results.length : 0
            task.stats = JSON.parse(JSON.stringify(tab.stats || {}))
            task.aliveCheckResult = tab.aliveCheckResult ? JSON.parse(JSON.stringify(tab.aliveCheckResult)) : null
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
        target: '',
        port: '',
        service: '',
        targetUrl: '',
        banner: '',
        title: '',
        code: '',
        location: ''
      }
    },
    resetColumnFilters(tab) {
      this.ensureTabColumnFilters(tab)
      Object.assign(tab.columnFilters, {
        target: '',
        port: '',
        service: '',
        targetUrl: '',
        banner: '',
        title: '',
        code: '',
        location: ''
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
    getHostscanTargetCellText(row) {
      if (!row) return ''
      if (row.protocol === 'http' || row.protocol === 'https') {
        return this.getTargetURL(row) || ''
      }
      return `${row.target || ''}:${row.port != null ? row.port : ''}`
    },
    getHostscanRowBannerText(row) {
      if (!row) return ''
      const parts = []
      ;(row.technologies || []).forEach(t => parts.push(String(t)))
      if (row.cms) parts.push(String(row.cms))
      if (row.banner && row.banner.trim()) parts.push(String(row.banner))
      return parts.join(' ').toLowerCase()
    },
    getFilteredResults(tab) {
      this.ensureTabColumnFilters(tab)
      let filtered = [...(tab.results || [])]
      const cf = tab.columnFilters
      const tq = (cf.target || '').trim().toLowerCase()
      if (tq) {
        filtered = filtered.filter(r => (r.target || '').toLowerCase().includes(tq))
      }
      const pq = (cf.port || '').trim().toLowerCase()
      if (pq) {
        filtered = filtered.filter(r => String(r.port ?? '').toLowerCase().includes(pq))
      }
      const sq = (cf.service || '').trim().toLowerCase()
      if (sq) {
        filtered = filtered.filter(r =>
          String(this.getServiceDisplay(r) || '').toLowerCase().includes(sq)
        )
      }
      const uq = (cf.targetUrl || '').trim().toLowerCase()
      if (uq) {
        filtered = filtered.filter(r =>
          String(this.getHostscanTargetCellText(r)).toLowerCase().includes(uq)
        )
      }
      const bq = (cf.banner || '').trim().toLowerCase()
      if (bq) {
        filtered = filtered.filter(r => this.getHostscanRowBannerText(r).includes(bq))
      }
      const titleQ = (cf.title || '').trim().toLowerCase()
      if (titleQ) {
        filtered = filtered.filter(r => (r.title || '').toLowerCase().includes(titleQ))
      }
      const cq = (cf.code || '').trim().toLowerCase()
      if (cq) {
        filtered = filtered.filter(r =>
          String(this.getStatusCode(r) ?? '').toLowerCase().includes(cq)
        )
      }
      const lq = (cf.location || '').trim().toLowerCase()
      if (lq) {
        filtered = filtered.filter(r => (r.location || '').toLowerCase().includes(lq))
      }
      return filtered
    },
    getPagedResults(tab) {
      const filtered = this.getFilteredResults(tab)
      const sorted = [...filtered]
      const prop = tab.sortField
      const order = tab.sortOrder
      if (prop && order) {
        sorted.sort((a, b) => {
          let aVal
          let bVal
          if (prop === 'port') {
            aVal = Number(a.port) || 0
            bVal = Number(b.port) || 0
          } else if (prop === 'code') {
            aVal = Number(this.getStatusCode(a)) || 0
            bVal = Number(this.getStatusCode(b)) || 0
          } else if (prop === 'target') {
            aVal = String(a.target || '').toLowerCase()
            bVal = String(b.target || '').toLowerCase()
          } else if (prop === 'service') {
            aVal = String(this.getServiceDisplay(a) || '').toLowerCase()
            bVal = String(this.getServiceDisplay(b) || '').toLowerCase()
          } else if (prop === 'targetUrl') {
            aVal = String(this.getHostscanTargetCellText(a)).toLowerCase()
            bVal = String(this.getHostscanTargetCellText(b)).toLowerCase()
          } else if (prop === 'banner') {
            aVal = this.getHostscanRowBannerText(a)
            bVal = this.getHostscanRowBannerText(b)
          } else if (prop === 'title') {
            aVal = String(a.title || '').toLowerCase()
            bVal = String(b.title || '').toLowerCase()
          } else if (prop === 'location') {
            aVal = String(a.location || '').toLowerCase()
            bVal = String(b.location || '').toLowerCase()
          } else {
            return 0
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
    // 历史记录管理
    saveHistory() {
      // 使用防抖，减少保存频率
      if (this.saveHistoryTimer) {
        clearTimeout(this.saveHistoryTimer)
      }
      this.saveHistoryTimer = setTimeout(() => {
        this._saveHistory()
      }, 1000) // 1秒防抖
    },
    _saveHistory() {
      try {
        // 不保存完整结果数组，只保存摘要信息
        const history = this.scanTabs.map(tab => ({
          id: tab.id,
          name: tab.name,
          targetsText: tab.targetsText,
          config: JSON.parse(JSON.stringify(tab.config)),
          // 只保存结果数量，不保存完整结果数组
          resultsCount: tab.results ? tab.results.length : 0,
          stats: tab.stats,
          createTime: tab.createTime
        }))
        
        // 使用存储管理器保存历史记录，自动限制大小和清理
        const success = setItem('hostscanHistory', history, {
          maxArrayItems: 50, // 最多保留50条历史记录
          sortFn: (a, b) => (a.createTime || 0) - (b.createTime || 0) // 按创建时间排序
        })
        if (!success) {
          console.warn('保存历史记录失败，可能存储空间不足')
        }
      } catch (error) {
        console.error('保存历史记录失败:', error)
      }
    },
    loadHistory() {
      try {
        const history = getItem('hostscanHistory', [])
        if (Array.isArray(history) && history.length > 0) {
          this.historyList = history.map(item => ({
            ...item,
            scanning: false,
            loading: false,
            completed: true,
            currentTarget: '',
            currentPage: 1,
            pageSize: 10
          }))
        }
      } catch (error) {
        console.error('加载历史记录失败:', error)
      }
    },
    handleHistoryDialogOpened() {
      // 先确保 tasks 数组是最新的（基于当前的 tasksLocal）
      // 这样即使 localStorage 还没有更新，任务管理对话框也能看到新任务
      this.tasks = Object.values(this.tasksLocal)
      
      // 然后保存当前内存中的任务（确保新创建的任务被保存）
      // 由于创建任务时已经立即保存，这里主要是为了确保所有任务都被保存
      this._saveTasks().then(() => {
        // 保存完成后，重新加载任务列表（确保显示最新的任务）
        this.loadTasks()
      }).catch(() => {
        // 如果保存失败，仍然加载 localStorage 中的任务
        this.loadTasks()
        // 然后合并内存中的新任务（确保新创建的任务不会丢失）
        const memoryTasks = { ...this.tasksLocal }
        for (const [taskId, task] of Object.entries(memoryTasks)) {
          // 如果内存中的任务不存在于 localStorage 中，或者内存中的任务更新，则使用内存中的
          if (!this.tasksLocal[taskId] || (task.updatedAt && task.updatedAt > (this.tasksLocal[taskId].updatedAt || 0))) {
            this.tasksLocal[taskId] = task
          }
        }
        this.tasks = Object.values(this.tasksLocal)
      })
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
          if (task.targetsText && task.targetsText !== tab.targetsText) {
            tab.targetsText = task.targetsText
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
                      if (fileData.aliveCheckResult) {
                        tab.aliveCheckResult = JSON.parse(JSON.stringify(fileData.aliveCheckResult))
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
          
          // 同步存活检测结果
          if (task.aliveCheckResult && !tab.aliveCheckResult) {
            tab.aliveCheckResult = JSON.parse(JSON.stringify(task.aliveCheckResult))
          }
        }
      }
    },
    // 保存扫描结果到本地JSON文件
    async saveResultsToFile(taskId, results, stats, aliveCheckResult) {
      try {
        const appPath = await GetAppPath()
        // 使用路径分隔符兼容不同系统
        const pathSeparator = appPath.includes('\\') ? '\\' : '/'
        const resultsDir = `${appPath}${pathSeparator}logs${pathSeparator}hostscan`
        
        // 生成文件名（使用任务ID和时间戳）
        const timestamp = Date.now()
        const fileName = `hostscan_${taskId}_${timestamp}.json`
        const filePath = `${resultsDir}${pathSeparator}${fileName}`
        
        const dataToSave = {
          taskId,
          timestamp: new Date().toISOString(),
          results: results || [],
          stats: stats || {},
          aliveCheckResult: aliveCheckResult || null
        }
        
        // SaveFile 会自动创建目录（如果不存在）
        await SaveFile(filePath, JSON.stringify(dataToSave, null, 2))
        return filePath
      } catch (error) {
        console.error('保存结果到文件失败:', error)
        // 如果保存失败，返回 null，但不影响其他功能
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
    // 防抖保存结果到文件
    saveResultsToFileDebounced(tab) {
      if (this.saveResultsTimer) {
        clearTimeout(this.saveResultsTimer)
      }
      // 30秒防抖，避免频繁保存
      this.saveResultsTimer = setTimeout(async () => {
        if (tab.taskId && this.tasksLocal[tab.taskId] && tab.results && tab.results.length > 0) {
          const filePath = await this.saveResultsToFile(
            tab.taskId,
            tab.results,
            tab.stats,
            tab.aliveCheckResult
          )
          if (filePath) {
            this.tasksLocal[tab.taskId].resultsFilePath = filePath
            // 保存任务元数据
            this.saveTasks()
          }
        }
      }, 30000) // 30秒
    },
    // 任务管理方法
    saveTasks() {
      // 使用防抖，减少保存频率
      if (this.saveTasksTimer) {
        clearTimeout(this.saveTasksTimer)
      }
      this.saveTasksTimer = setTimeout(() => {
        this._saveTasks()
      }, 2000) // 2秒防抖，任务数据更重要，但可以稍微延迟
    },
    async _saveTasks() {
      try {
        // 压缩任务数据：完整结果保存到文件，localStorage只保存元数据
        const compressedTasks = {}
        for (const [taskId, task] of Object.entries(this.tasksLocal)) {
          // 如果任务有完整结果且没有保存到文件，则保存到文件
          if (task.results && task.results.length > 0 && !task.resultsFilePath) {
            const filePath = await this.saveResultsToFile(
              taskId,
              task.results,
              task.stats,
              task.aliveCheckResult
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
        
        // 使用存储管理器保存任务，自动限制大小和清理
        const success = setItem('hostscanTasks', compressedTasks, {
          maxObjectItems: 100, // 最多保留100个任务
          sortFn: (a, b) => (b.createdAt || 0) - (a.createdAt || 0) // 按创建时间排序，最新的在前
        })
        
        if (success) {
          // 如果保存成功且被限制了大小，重新加载以同步内存中的任务
          const saved = getItem('hostscanTasks', {})
          if (Object.keys(saved).length < Object.keys(compressedTasks).length) {
            this.tasksLocal = saved
            this.tasks = Object.values(this.tasksLocal)
          }
        } else {
          console.warn('保存任务失败，可能存储空间不足')
        }
      } catch (error) {
        console.error('保存任务失败:', error)
      }
    },
    loadTasks() {
      try {
        const tasks = getItem('hostscanTasks', {})
        if (tasks && typeof tasks === 'object') {
          this.tasksLocal = tasks
          // 根据创建时间倒序排序（最新的在前）
          this.tasks = Object.values(this.tasksLocal).sort((a, b) => {
            const ta = a.createdAt || a.createTime || 0
            const tb = b.createdAt || b.createTime || 0
            return tb - ta
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
    async loadTask(taskId) {
      const task = this.tasksLocal[taskId]
      if (!task) {
        this.$message.error('任务不存在')
        return
      }
      
      // 尝试从文件加载完整结果
      let loadedResults = null
      let loadedStats = null
      let loadedAliveCheckResult = null
      
      if (task.resultsFilePath) {
        try {
          const fileData = await this.loadResultsFromFile(task.resultsFilePath)
          if (fileData) {
            loadedResults = fileData.results || null
            loadedStats = fileData.stats || null
            loadedAliveCheckResult = fileData.aliveCheckResult || null
          }
        } catch (error) {
          console.warn('从文件加载结果失败，尝试从内存加载:', error)
        }
      }
      
      // 如果文件加载失败，尝试从内存加载（兼容旧数据）
      if (!loadedResults) {
        if (task.results && Array.isArray(task.results) && task.results.length > 0) {
          loadedResults = JSON.parse(JSON.stringify(task.results))
        }
      }
      if (!loadedStats && task.stats) {
        loadedStats = JSON.parse(JSON.stringify(task.stats))
      }
      if (!loadedAliveCheckResult && task.aliveCheckResult) {
        loadedAliveCheckResult = JSON.parse(JSON.stringify(task.aliveCheckResult))
      }
      
      // 查找是否已有该任务的标签页
      let tab = this.scanTabs.find(t => t.taskId === taskId)
      if (!tab) {
        // 创建新标签页
        const tabId = `tab-${++this.tabIdCounter}`
        tab = {
          id: tabId,
          name: task.name,
          targetsText: task.targetsText || '',
          config: task.config || {
            portPreset: 'top100',
            ports: '',
            checkAlive: false,
            aliveCheck: null,
            randomizePorts: false,
            threads: 100,
            timeout: 3,
            excludePorts: [],
            enablePortFingerprint: false,
            emitClosedPorts: false,
            enableFirewallDetection: true,
            disableFirewallDetection: false,
            firewallDetectSampleSize: 14,
            firewallDetectThresholdPercent: 90
          },
          scanning: task.status === 'running',
          loading: false,
          completed: task.status === 'completed',
          results: loadedResults || [],
          stats: loadedStats || {
            totalPorts: 0,
            checked: 0,
            openCount: loadedResults ? loadedResults.length : (task.resultsCount || 0),
            closedOrFiltered: 0,
            portsPerSecond: 0
          },
          aliveCheckResult: loadedAliveCheckResult || null,
          currentTarget: '',
          currentPage: 1,
          pageSize: 10,
          createTime: task.createdAt || Date.now(),
          taskId: taskId,
          columnFilters: {
            target: '',
            port: '',
            service: '',
            targetUrl: '',
            banner: '',
            title: '',
            code: '',
            location: ''
          },
          sortField: '',
          sortOrder: 'asc'
        }
        this.scanTabs.push(tab)
      } else {
        // 如果标签页已存在，也更新数据
        if (loadedResults) {
          tab.results = loadedResults
        } else if (Array.isArray(task.results)) {
          tab.results = JSON.parse(JSON.stringify(task.results))
        } else if (typeof task.results === 'number') {
          // 旧格式：只有结果数量，保持现有结果
        }
        
        if (loadedStats) {
          tab.stats = loadedStats
        } else if (task.stats) {
          tab.stats = JSON.parse(JSON.stringify(task.stats))
        } else if (typeof task.results === 'number') {
          // 旧格式：只有结果数量，更新openCount
          tab.stats.openCount = task.results
        }
        
        if (loadedAliveCheckResult) {
          tab.aliveCheckResult = loadedAliveCheckResult
        } else {
          tab.aliveCheckResult = task.aliveCheckResult ? JSON.parse(JSON.stringify(task.aliveCheckResult)) : null
        }
        
        tab.completed = task.status === 'completed'
        tab.scanning = task.status === 'running'
        tab.scanProgress = task.progress || 0
        
        // 同步配置
        if (task.config) {
          tab.config = JSON.parse(JSON.stringify(task.config))
        }
        if (task.targetsText) {
          tab.targetsText = task.targetsText
        }
        if (!tab.columnFilters) {
          tab.columnFilters = {
            target: '',
            port: '',
            service: '',
            targetUrl: '',
            banner: '',
            title: '',
            code: '',
            location: ''
          }
        }
        if (tab.sortField === undefined) {
          tab.sortField = ''
          tab.sortOrder = 'asc'
        }
      }
      
      // 切换到该标签页
      this.activeTabId = tab.id
      this.historyDialog = false
      
      const resultsCount = tab.results ? tab.results.length : 0
      if (resultsCount > 0) {
        this.$message.success(`任务已加载到标签页，共 ${resultsCount} 条结果`)
      } else {
        this.$message.success('任务已加载到标签页')
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
      removeItem('hostscanTasks')
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
        targetsText: historyItem.targetsText || '',
        config: historyItem.config || {
          ports: '80,443,8080-8090',
          portPreset: 'custom',
          checkAlive: true,
          aliveCheck: {
            type: 'icmp',
            ports: [80, 443, 22],
            url: 'http://{host}/',
            timeout: 3,
            enabled: true
          },
          randomizePorts: true,
          threads: 300,
          timeout: 3,
          enablePortFingerprint: true,
          emitClosedPorts: false,
          enableFingerprint: true,
          enableDirsearch: true,
          enableVulnScan: false,
          enableCrack: false,
          vulnSkipWithoutTags: false,
          crackBuiltInUsername: true,
          crackBuiltInPassword: true,
          crackUsernames: '',
          crackPasswords: '',
          excludePorts: [],
          enableFirewallDetection: true,
          disableFirewallDetection: false,
          firewallDetectSampleSize: 14,
          firewallDetectThresholdPercent: 90
        },
        scanning: false,
        loading: false,
        completed: true,
        results: historyItem.results || [],
        stats: historyItem.stats || {
          totalPorts: 0,
          checked: 0,
          openCount: 0,
          closedOrFiltered: 0,
          portsPerSecond: 0
        },
        aliveCheckResult: null,
        currentTarget: '',
        currentPage: 1,
        pageSize: 10,
        createTime: historyItem.createTime || Date.now(),
        columnFilters: {
          target: '',
          port: '',
          service: '',
          targetUrl: '',
          banner: '',
          title: '',
          code: '',
          location: ''
        },
        sortField: '',
        sortOrder: 'asc'
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
        setItem('hostscanHistory', this.historyList, { maxArrayItems: 50 })
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
      removeItem('hostscanHistory')
      this.$message.success('清除成功')
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
    bindEvents() {
      EventsOn('hostscanResult', this.handleResult)
      EventsOn('hostscanProgress', this.handleProgress)
      EventsOn('hostscanError', this.handleError)
      EventsOn('hostscanComplete', this.handleComplete)
      EventsOn('hostscanAliveCheck', this.handleAliveCheck)
      EventsOn('hostscanNotice', this.handleNotice)
    },
    unbindEvents() {
      EventsOff('hostscanResult')
      EventsOff('hostscanProgress')
      EventsOff('hostscanError')
      EventsOff('hostscanComplete')
      EventsOff('hostscanAliveCheck')
      EventsOff('hostscanNotice')
    },
    calcTableHeight() {
      this.$nextTick(() => {
        const container = document.querySelector('.hostscan-container')
        const config = document.querySelector('.config-card')
        const resultsCard = document.querySelector('.results-card')
        const header = document.querySelector('.results-header')
        const pagination = document.querySelector('.pagination-wrapper')
        if (container && config && resultsCard && header && pagination) {
          const height =
            resultsCard.clientHeight -
            header.clientHeight -
            pagination.clientHeight -
            60 // 增加padding确保最后一条完整显示
          this.tableHeight = Math.max(height, 260)
        }
      })
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
    async startScan(tab) {
      // 手动触发时，不跳过确认对话框
      await this.startScanInternal(tab, false)
    },
    resetScanState(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      tab.results = []
      tab.stats = {
        totalPorts: 0,
        checked: 0,
        openCount: 0,
        closedOrFiltered: 0,
        portsPerSecond: 0
      }
      tab.currentTarget = ''
      tab.aliveCheckResult = null
    },
    handleExternalHostscanFill(event) {
      const payload = event?.detail
      const targets = payload?.targets
      const sourceTaskId = payload?.sourceTaskId // 来源任务ID（用于关联）

      if (!Array.isArray(targets) || targets.length === 0) {
        return
      }

      // 如果提供了 sourceTaskId，查找是否已存在相同来源任务的标签页
      let targetTab = null
      if (sourceTaskId) {
        const sourceTaskType = payload?.sourceTaskType || 'unknown'
        targetTab = this.scanTabs.find(tab => 
          tab.sourceTaskId === sourceTaskId && 
          tab.sourceTaskType === sourceTaskType
        )
        // 如果找到了标签页，确保sourceTaskId和sourceTaskType已设置（防止后续查找失败）
        if (targetTab && !targetTab.sourceTaskId) {
          targetTab.sourceTaskId = sourceTaskId
          targetTab.sourceTaskType = sourceTaskType
        }
      }

      // 1）如果还没匹配到标签，并且当前只有一个初始化时创建的默认标签页，
      //    优先复用这个默认页（不论里面有没有旧数据），直接作为联动目标页使用
      if (!targetTab && this.scanTabs.length === 1) {
        const onlyTab = this.scanTabs[0]
        // 没有关联来源任务视为“默认页”，可以被联动覆盖使用
        if (!onlyTab.sourceTaskId) {
          // 如果默认页里有旧结果/状态，这里主动清空，专门留给目录扫描 / 指纹识别联动使用
          this.resetScanState(onlyTab)
          onlyTab.targetsText = ''
          targetTab = onlyTab
          if (sourceTaskId) {
            onlyTab.sourceTaskId = sourceTaskId
            onlyTab.sourceTaskType = payload?.sourceTaskType || 'unknown'
          }
        }
      }

      // 2）如果仍未匹配到标签，再尝试复用一个“空闲标签页”（无目标、无结果、未在扫描）
      //    这样在存在多个标签但有空闲页时，联动会优先占用空闲页而不是新建
      if (!targetTab) {
        targetTab = this.scanTabs.find(tab => {
          const noTargets = !tab.targetsText || tab.targetsText.trim().length === 0
          const noResults = !tab.results || tab.results.length === 0
          const notScanning = !tab.scanning
          return noTargets && noResults && notScanning
        }) || null

        if (targetTab && sourceTaskId) {
          targetTab.sourceTaskId = sourceTaskId
          targetTab.sourceTaskType = payload?.sourceTaskType || 'unknown'
        }
      }

      if (targetTab) {
        // 如果已存在相同来源任务的标签页，追加目标而不是创建新标签页
        const existingTargets = targetTab.targetsText.split('\n').filter(t => t.trim())
        const newTargets = targets.filter(t => !existingTargets.includes(t.trim()))
        if (newTargets.length > 0) {
          targetTab.targetsText = [...existingTargets, ...newTargets].join('\n')
        }
        // 切换到现有标签页
        this.activeTabId = targetTab.id
        // 应用配置
        this.applyExternalScanConfigToTab(payload, targetTab)
        // 如果标签页未在扫描中，自动开始扫描
        if (!targetTab.scanning) {
          this.$nextTick(() => {
            this.startScanInternal(targetTab, true) // 传入true表示自动联动，跳过覆盖提示
          })
        }
      } else {
        // 联动时自动创建新标签页
        const newTab = this.createNewTab()
        newTab.name = `主机扫描-联动-${new Date().toLocaleTimeString()}`
        
        // 记录来源任务ID（用于任务管理中的关联显示）
        if (sourceTaskId) {
          newTab.sourceTaskId = sourceTaskId
          newTab.sourceTaskType = payload?.sourceTaskType || 'unknown' // 来源任务类型
        }

        // 切换到新标签页
        this.activeTabId = newTab.id

        // 先填充目标（确保新标签页有目标）
        if (targets.length > 0) {
          newTab.targetsText = targets.join('\n')
        }

        // 应用配置
        this.applyExternalScanConfigToTab(payload, newTab)
        
        // 自动联动时，直接保存记录并开始扫描，不触发覆盖提示
        this.$nextTick(() => {
          this.startScanInternal(newTab, true) // 传入true表示自动联动，跳过覆盖提示
        })
      }
    },
    applyExternalScanConfigToTab(payload, tab) {
      if (!tab) {
        tab = this.currentTab || this.scanTabs[0]
      }
      if (!tab) return
      
      // 应用配置（如果提供了）
      if (payload?.config) {
        const cfg = payload.config
        if (cfg.ports !== undefined) tab.config.ports = cfg.ports
        if (cfg.portPreset !== undefined) tab.config.portPreset = cfg.portPreset
        if (cfg.threads !== undefined) tab.config.threads = cfg.threads
        if (cfg.timeout !== undefined) tab.config.timeout = cfg.timeout
        if (cfg.checkAlive !== undefined) tab.config.checkAlive = cfg.checkAlive
        if (cfg.randomizePorts !== undefined) tab.config.randomizePorts = cfg.randomizePorts
        if (cfg.enablePortFingerprint !== undefined) tab.config.enablePortFingerprint = cfg.enablePortFingerprint
        if (cfg.emitClosedPorts !== undefined) tab.config.emitClosedPorts = !!cfg.emitClosedPorts
        if (cfg.enableFingerprint !== undefined) tab.config.enableFingerprint = cfg.enableFingerprint
        if (cfg.enableDirsearch !== undefined) tab.config.enableDirsearch = cfg.enableDirsearch
        if (cfg.excludePorts !== undefined) tab.config.excludePorts = cfg.excludePorts
        if (cfg.aliveCheck !== undefined) tab.config.aliveCheck = cfg.aliveCheck
        if (cfg.disableFirewallDetection !== undefined) tab.config.disableFirewallDetection = cfg.disableFirewallDetection
        if (cfg.enableFirewallDetection !== undefined) tab.config.enableFirewallDetection = !!cfg.enableFirewallDetection
        if (cfg.disableFirewallDetection !== undefined && cfg.enableFirewallDetection === undefined) {
          tab.config.enableFirewallDetection = !cfg.disableFirewallDetection
        }
        if (cfg.firewallDetectSampleSize !== undefined) tab.config.firewallDetectSampleSize = cfg.firewallDetectSampleSize
        if (cfg.firewallDetectThreshold !== undefined) {
          tab.config.firewallDetectThresholdPercent = Math.round(Number(cfg.firewallDetectThreshold) * 100)
        } else if (cfg.firewallDetectThresholdPercent !== undefined) {
          tab.config.firewallDetectThresholdPercent = cfg.firewallDetectThresholdPercent
        }
      }
    },
    async startScanInternal(tab, skipConfirm = false) {
      if (!tab) tab = this.currentTab
      if (!tab) {
        this.$message.warning('请先创建或选择一个任务')
        return
      }
      this.flushPendingHostscanTextareas(tab)
      const targets = tab.targetsText
        .split('\n')
        .map(t => t.trim())
        .filter(t => t.length > 0)
      if (targets.length === 0) {
        this.$message.warning({
          message: '请输入至少一个目标',
          duration: 2000
        })
        return
      }
      if (tab.config.portPreset === 'custom' && !tab.config.ports.trim()) {
        this.$message.warning({
          message: '请选择端口预设或输入端口范围',
          duration: 2000
        })
        return
      }

      let shouldClearResults = tab.results.length === 0
      let shouldKeepResults = false

      // 自动联动时跳过确认对话框
      if (!skipConfirm && tab.results.length > 0) {
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
          shouldClearResults = false
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
              return
            }
          } else {
            return
          }
        }
      } else if (skipConfirm) {
        // 自动联动时，保留现有结果
        shouldKeepResults = true
        shouldClearResults = false
      }

      if (shouldClearResults) {
        this.resetScanState(tab)
      }
      try {
        this.clearFirewallNoticePrompted(tab.id)
        tab.scanning = true
        tab.loading = true
        if (!shouldKeepResults) {
          tab.currentTarget = ''
        }
        // 确保存活检测配置正确
        let aliveCheck = null
        if (tab.config.checkAlive) {
          if (tab.config.aliveCheck && tab.config.aliveCheck.enabled !== false) {
            aliveCheck = {
              type: tab.config.aliveCheck.type || 'tcp_port',
              ports: tab.config.aliveCheck.ports || [80, 443, 22],
              url: tab.config.aliveCheck.url || 'http://{host}/',
              timeout: tab.config.aliveCheck.timeout || 3,
              enabled: true
            }
          } else {
            // 如果没有配置，使用默认ICMP配置
            aliveCheck = {
              type: 'icmp',
              ports: [80, 443, 22],
              url: 'http://{host}/',
              timeout: 3,
              enabled: true
            }
          }
        }
        
        // 处理端口过滤
        let ports = tab.config.ports
        if (tab.config.portPreset === 'all') {
          ports = '1-65535'
        }
        
        // 处理排除端口
        let excludePorts = []
        if (tab.config.excludePorts && tab.config.excludePorts.includes('printer')) {
          excludePorts = [515, 631, 9100, 9101, 9102, 9103, 9104, 9105, 9106, 9107, 9108, 9109]
        }

        const firewallDetectSampleSize = Number(tab.config.firewallDetectSampleSize) || 14
        const firewallDetectThresholdPercent = Number(tab.config.firewallDetectThresholdPercent) || 90
        const firewallDetectThreshold = Math.min(1, Math.max(0.6, firewallDetectThresholdPercent / 100))
        const enableFirewallDetection = tab.config.enableFirewallDetection !== false
        
        const cfg = {
          targets,
          ports: ports,
          portPreset: tab.config.portPreset === 'all' ? 'custom' : tab.config.portPreset,
          checkAlive: tab.config.checkAlive,
          aliveCheck: aliveCheck,
          randomizePorts: tab.config.randomizePorts,
          threads: tab.config.threads,
          timeout: tab.config.timeout,
          excludePorts: excludePorts,
          enablePortFingerprint: tab.config.enablePortFingerprint,
          emitClosedPorts: tab.config.emitClosedPorts === true,
          enableFirewallDetection: enableFirewallDetection,
          disableFirewallDetection: !enableFirewallDetection,
          firewallDetectSampleSize: firewallDetectSampleSize,
          firewallDetectThreshold: firewallDetectThreshold,
          tabId: tab.id // 传递tabId用于事件处理
        }
        
        // 生成任务ID
        const taskId = `hostscan-${Date.now()}-${++this.taskIdCounter}`
        tab.taskId = taskId
        
        // 创建任务记录（保存完整数据）
        // 合并后端扫描配置和前端标签配置，确保联动配置（enableCrack、enableVulnScan等）也被保存
        const fullConfig = {
          ...JSON.parse(JSON.stringify(cfg)), // 后端扫描配置
          ...JSON.parse(JSON.stringify(tab.config)) // 前端标签配置（包括联动配置）
        }
        this.tasksLocal[taskId] = {
          taskId,
          name: tab.name || `主机扫描-${new Date().toLocaleTimeString()}`,
          targetsText: tab.targetsText,
          config: fullConfig, // 保存完整的配置（包括联动配置）
          // 保存完整结果到内存（用于实时显示），完整结果会保存到文件
          results: JSON.parse(JSON.stringify(tab.results)),
          resultsCount: tab.results.length,
          stats: JSON.parse(JSON.stringify(tab.stats)),
          aliveCheckResult: tab.aliveCheckResult ? JSON.parse(JSON.stringify(tab.aliveCheckResult)) : null,
          status: 'running',
          progress: 0,
          completed: 0,
          total: 0,
          createdAt: Date.now(),
          updatedAt: Date.now(),
          resultsFilePath: null // 结果文件路径，会在保存时设置
        }
        this.tasks = Object.values(this.tasksLocal)
        // 立即保存任务到 localStorage，确保任务管理对话框能实时看到新任务
        await this._saveTasks()
        
        await StartHostScan(JSON.stringify(cfg))
        tab.loading = false
        this.saveHistory()
        this.$message.success({
          message: '主机扫描已启动',
          duration: 2000
        })
      } catch (e) {
        this.$message.error({
          message: `启动失败: ${e?.message || e}`,
          duration: 2000
        })
        tab.scanning = false
        tab.loading = false
      }
    },
    async stopScan(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      try {
        // 先设置状态，避免UI卡顿
        tab.scanning = false
        tab.loading = true
        await StopHostScan()
        // 更新任务状态（保存完整数据）
        if (tab.taskId && this.tasksLocal[tab.taskId]) {
          this.tasksLocal[tab.taskId].status = 'stopped'
          this.tasksLocal[tab.taskId].results = JSON.parse(JSON.stringify(tab.results))
          this.tasksLocal[tab.taskId].resultsCount = tab.results.length
          this.tasksLocal[tab.taskId].stats = JSON.parse(JSON.stringify(tab.stats))
          this.tasksLocal[tab.taskId].aliveCheckResult = tab.aliveCheckResult ? JSON.parse(JSON.stringify(tab.aliveCheckResult)) : null
          this.tasksLocal[tab.taskId].progress = tab.scanProgress || 0
          
          // 同步标签页的基本信息到任务（确保任务管理和标签页显示一致）
          if (tab.name) {
            this.tasksLocal[tab.taskId].name = tab.name
          }
          if (tab.targetsText) {
            this.tasksLocal[tab.taskId].targetsText = tab.targetsText
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
      } catch (e) {
        this.$message.error({
          message: `停止失败: ${e?.message || e}`,
          duration: 2000
        })
      } finally {
        tab.scanning = false
        tab.loading = false
        tab.currentTarget = ''
        // 清理批量更新定时器
        if (tab.updateResultsTimer) {
          cancelAnimationFrame(tab.updateResultsTimer)
          tab.updateResultsTimer = null
        }
        // 处理剩余待更新的结果
        if (tab.pendingResults && tab.pendingResults.length > 0) {
          tab.results.push(...tab.pendingResults)
          tab.stats.openCount = tab.results.length
          tab.pendingResults = []
        }
      }
    },
    handleResult(result) {
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
      
      // 只添加开放端口到结果列表；关闭端口默认由后端不再推送，此处仅兼容「高级配置：推送关闭端口」
      if (!result.open) {
        // 如果标签存在，更新标签统计信息
        if (tab) {
          tab.stats.closedOrFiltered = tab.stats.closedOrFiltered + 1
        }
        // 更新任务数据
        if (taskId && this.tasksLocal[taskId]) {
          if (!this.tasksLocal[taskId].stats) {
            this.tasksLocal[taskId].stats = {}
          }
          this.tasksLocal[taskId].stats.closedOrFiltered = (this.tasksLocal[taskId].stats.closedOrFiltered || 0) + 1
          this.tasksLocal[taskId].updatedAt = Date.now()
          this.saveTasks()
        }
        return
      }
      
      // 如果标签存在，添加到标签结果列表
      if (tab) {
        // 直接添加结果，Vue的响应式系统会自动处理更新
        tab.results.push(result)
        // 立即更新统计信息，确保实时显示
        tab.stats.openCount = tab.results.length
      }
      
      // 同步更新任务数据（无论标签是否存在）
      if (taskId && this.tasksLocal[taskId]) {
        // 初始化任务结果数组（如果不存在）
        if (!this.tasksLocal[taskId].results) {
          this.tasksLocal[taskId].results = []
        }
        // 保存完整结果到内存
        this.tasksLocal[taskId].results.push(JSON.parse(JSON.stringify(result)))
        this.tasksLocal[taskId].resultsCount = this.tasksLocal[taskId].results.length
        
        // 更新统计信息
        if (!this.tasksLocal[taskId].stats) {
          this.tasksLocal[taskId].stats = {}
        }
        if (tab) {
          this.tasksLocal[taskId].stats = JSON.parse(JSON.stringify(tab.stats))
          // 同步标签页的基本信息到任务（确保任务管理和标签页显示一致）
          if (tab.name) {
            this.tasksLocal[taskId].name = tab.name
          }
          if (tab.targetsText) {
            this.tasksLocal[taskId].targetsText = tab.targetsText
          }
          if (tab.config) {
            this.tasksLocal[taskId].config = JSON.parse(JSON.stringify(tab.config))
          }
        } else {
          this.tasksLocal[taskId].stats.openCount = this.tasksLocal[taskId].results.length
        }
        
        this.tasksLocal[taskId].updatedAt = Date.now()
        // 使用防抖保存，减少保存频率（完整结果会保存到文件）
        this.saveTasks()
        // 定期保存完整结果到文件（每30秒或扫描完成时）
        if (tab) {
          this.saveResultsToFileDebounced(tab)
        } else {
          // 如果标签不存在，也保存到文件
          this.saveResultsToFileDebounced({ taskId, results: this.tasksLocal[taskId].results, stats: this.tasksLocal[taskId].stats })
        }
      }
      
      // 获取联动配置（优先从标签配置中获取，如果标签不存在则从任务配置中获取）
      let enableFingerprint = false
      let enableDirsearch = false
      if (tab && tab.config) {
        // 标签存在时，优先使用标签配置（配置可能已被用户修改）
        enableFingerprint = tab.config.enableFingerprint === true || tab.config.enableFingerprint === 'true'
        enableDirsearch = tab.config.enableDirsearch === true || tab.config.enableDirsearch === 'true'
      } else if (taskId && this.tasksLocal[taskId] && this.tasksLocal[taskId].config) {
        // 标签不存在时，从任务配置获取（标签已关闭的情况）
        enableFingerprint = this.tasksLocal[taskId].config.enableFingerprint === true || this.tasksLocal[taskId].config.enableFingerprint === 'true'
        enableDirsearch = this.tasksLocal[taskId].config.enableDirsearch === true || this.tasksLocal[taskId].config.enableDirsearch === 'true'
      }
      
      // 如果标签存在，执行UI相关操作
      if (tab) {
        // 自动滚动到最新结果
        this.$nextTick(() => {
          // 计算总页数并跳转到最后一页
          const filtered = this.getFilteredResults(tab)
          const totalPages = Math.ceil(filtered.length / tab.pageSize)
          if (totalPages > 0) {
            tab.currentPage = totalPages
          }
        })
        this.saveHistory()
      }
      
      // 检查是否需要联动指纹识别或目录扫描（无论标签是否存在都要执行）
      // 使用防抖机制，避免频繁触发联动导致消息冲突
      if (this.isWebService(result)) {
          if (enableFingerprint) {
            // 延迟执行，避免阻塞扫描结果更新，使用 $nextTick 确保事件能够正确触发
            // 使用更长的延迟，避免与其他操作冲突
            // 传递 result 对象（包含 tabId），即使 tab 为 null 也能通过 tabId 查找任务
            this.$nextTick(() => {
              setTimeout(() => {
                this.runFingerprint(result, false, tab)
              }, 300)
            })
          }
          if (enableDirsearch) {
            // 延迟执行，避免阻塞扫描结果更新，使用 $nextTick 确保事件能够正确触发
            // 使用更长的延迟，避免与其他操作冲突
            // 传递 result 对象（包含 tabId），即使 tab 为 null 也能通过 tabId 查找任务
            this.$nextTick(() => {
              setTimeout(() => {
                this.runDirsearch(result, false, tab)
              }, 400)
            })
          }
      }
    },
    handleProgress(payload) {
      // 根据后端携带的 tabId 精确找到对应的标签页
      const tabId = payload?.tabId
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
      
      if (payload?.stats) {
        // 如果标签存在，更新标签状态
        if (tab) {
          tab.stats = { ...tab.stats, ...payload.stats }
          tab.currentTarget = payload.stats.currentTarget || ''
          
          // 如果进度达到100%，立即刷新表格显示
          if (payload.stats.progressPercent >= 100) {
            this.$nextTick(() => {
              // 强制更新表格，确保显示最新数据
              this.$forceUpdate()
            })
          }
        }
        
        // 同步更新任务数据（无论标签是否存在）
        if (taskId && this.tasksLocal[taskId]) {
          if (!this.tasksLocal[taskId].stats) {
            this.tasksLocal[taskId].stats = {}
          }
          this.tasksLocal[taskId].stats = { ...this.tasksLocal[taskId].stats, ...payload.stats }
          
          // 如果标签存在，同步标签页的基本信息到任务（确保任务管理和标签页显示一致）
          if (tab) {
            if (tab.name) {
              this.tasksLocal[taskId].name = tab.name
            }
            if (tab.targetsText) {
              this.tasksLocal[taskId].targetsText = tab.targetsText
            }
            if (tab.config) {
              this.tasksLocal[taskId].config = JSON.parse(JSON.stringify(tab.config))
            }
            // 同步进度
            if (payload.progress !== undefined) {
              this.tasksLocal[taskId].progress = payload.progress
            }
          }
          
          this.tasksLocal[taskId].updatedAt = Date.now()
          // 使用防抖保存，减少保存频率
          this.saveTasks()
        }
      }
    },
    handleError(err) {
      if (err?.message) {
        this.$message.warning({
          message: err.message,
          duration: 2000
        })
      }
      // 根据后端携带的 tabId 精确找到对应的标签页并更新状态
      const tabId = err?.tabId
      const tab = tabId ? this.scanTabs.find(t => t.id === tabId) : null
      if (tab) {
        tab.scanning = false
        tab.loading = false
      }
    },
    async handleComplete(payload) {
      // 根据后端携带的 tabId 精确找到对应的标签页
      const tabId = payload?.tabId
      const tab = tabId ? this.scanTabs.find(t => t.id === tabId) : null
      
      // 如果标签页存在，立即更新标签页状态（不等待文件保存）
      if (tab) {
        tab.scanning = false
        tab.loading = false
        tab.completed = true
        tab.currentTarget = ''
        // 清理批量更新定时器
        if (tab.updateResultsTimer) {
          cancelAnimationFrame(tab.updateResultsTimer)
          tab.updateResultsTimer = null
        }
        // 处理剩余待更新的结果
        if (tab.pendingResults && tab.pendingResults.length > 0) {
          tab.results.push(...tab.pendingResults)
          tab.stats.openCount = tab.results.length
          tab.pendingResults = []
        }
        
        // 立即更新最终统计信息（从payload获取最新数据）
        if (payload?.stats) {
          tab.stats = { ...tab.stats, ...payload.stats }
          // 确保进度为100%
          tab.stats.progressPercent = 100
        }
        
        // 强制刷新表格显示
        this.$nextTick(() => {
          // 触发表格重新渲染
          this.$forceUpdate()
        })
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
          this.tasksLocal[taskId].aliveCheckResult = tab.aliveCheckResult ? JSON.parse(JSON.stringify(tab.aliveCheckResult)) : null
          
          // 同步标签页的基本信息到任务（确保任务管理和标签页显示一致）
          if (tab.name) {
            this.tasksLocal[taskId].name = tab.name
          }
          if (tab.targetsText) {
            this.tasksLocal[taskId].targetsText = tab.targetsText
          }
          if (tab.config) {
            this.tasksLocal[taskId].config = JSON.parse(JSON.stringify(tab.config))
          }
        } else if (payload?.results) {
          // 如果标签页不存在，尝试使用 payload 中的数据
          this.tasksLocal[taskId].results = JSON.parse(JSON.stringify(payload.results))
          this.tasksLocal[taskId].resultsCount = payload.results?.length || 0
          this.tasksLocal[taskId].stats = payload.stats || this.tasksLocal[taskId].stats
        }
        
        this.tasksLocal[taskId].updatedAt = Date.now()
        // 立即保存任务元数据（不等待文件保存）
        this.saveTasks()
        
        // 异步保存完整结果到文件（不阻塞UI更新）
        const results = tab ? tab.results : (payload?.results || [])
        const stats = tab ? tab.stats : (payload?.stats || {})
        const aliveCheckResult = tab ? tab.aliveCheckResult : (payload?.aliveCheckResult || null)
        // 使用setTimeout延迟执行文件保存，避免阻塞UI
        setTimeout(async () => {
          try {
            const filePath = await this.saveResultsToFile(taskId, results, stats, aliveCheckResult)
            if (filePath && this.tasksLocal[taskId]) {
              this.tasksLocal[taskId].resultsFilePath = filePath
              this.saveTasks()
            }
          } catch (error) {
            console.error('保存结果文件失败:', error)
          }
        }, 100) // 延迟100ms执行，确保UI先更新
      }
      
      if (tab) {
        // 异步保存历史记录
        this.$nextTick(() => {
          this.saveHistory()
        })
      }
      // 移除主机扫描完成的提示消息，避免与其他提示冲突
      // this.$message.success({
      //   message: '主机扫描完成',
      //   duration: 2000
      // })

      // 扫描完成后，如有勾选联动任务则执行（无论标签是否存在）
      // 获取联动配置（优先从任务配置中获取，如果任务不存在则从标签配置中获取）
      let enableVulnScan = false
      let enableCrack = false
      let taskResults = []
      
      if (taskId && this.tasksLocal[taskId]) {
        const task = this.tasksLocal[taskId]
        if (task.config) {
          enableVulnScan = task.config.enableVulnScan === true || task.config.enableVulnScan === 'true'
          enableCrack = task.config.enableCrack === true || task.config.enableCrack === 'true'
        }
        taskResults = task.results || []
      } else if (tab && tab.config) {
        enableVulnScan = tab.config.enableVulnScan === true || tab.config.enableVulnScan === 'true'
        enableCrack = tab.config.enableCrack === true || tab.config.enableCrack === 'true'
        taskResults = tab.results || []
      }
      
      // 如果标签不存在，创建一个临时对象用于传递数据（包含 config）
      const tabOrTask = tab || (taskId && this.tasksLocal[taskId] ? {
        results: taskResults,
        taskId: taskId,
        config: this.tasksLocal[taskId].config || {}
      } : null)
      
      if (enableVulnScan && tabOrTask) {
        this.startVulnScanFromResults(tabOrTask).catch(e => {
          console.error('从主机扫描结果启动漏洞扫描失败:', e)
        })
      }
      if (enableCrack && tabOrTask) {
        this.startCrackFromResults(tabOrTask).catch(e => {
          console.error('从主机扫描结果启动密码爆破失败:', e)
        })
      }
    },
    handleAliveCheck(result) {
      // 根据后端携带的 tabId 精确找到对应的标签页
      const tabId = result.tabId
      const tab = tabId ? this.scanTabs.find(t => t.id === tabId) : null
      if (!tab) return
      
      tab.aliveCheckResult = result

      // 同步更新任务数据
      if (tab.taskId && this.tasksLocal[tab.taskId]) {
        this.tasksLocal[tab.taskId].aliveCheckResult = JSON.parse(JSON.stringify(result))
        this.saveTasks()
      }

      if (result && result.totalIPs > 0 && result.aliveIPs === 0) {
        this.$message.warning({
          message:
            '未检测到存活主机：目标可能禁 ping，可关闭「IP存活检测」直接扫端口，或打开「配置存活检测」改用 TCP/HTTP。',
          duration: 8000,
          showClose: true
        })
      }
    },
    clearFirewallNoticePrompted(tabId) {
      if (!tabId) return
      const nextSet = new Set()
      this.firewallNoticePrompted.forEach(key => {
        if (!String(key).startsWith(`${tabId}:`)) {
          nextSet.add(key)
        }
      })
      this.firewallNoticePrompted = nextSet
    },
    loadAutoFullPortRescanPreference() {
      const remembered = getItem('hostscanAutoFullPortRescanRemember', false)
      this.autoFullPortRescanRemember = remembered === true || remembered === 'true'
    },
    saveAutoFullPortRescanPreference() {
      setItem('hostscanAutoFullPortRescanRemember', !!this.autoFullPortRescanRemember)
    },
    async restartFullPortScanFromNotice(payload) {
      const tabId = payload?.tabId
      if (!tabId) return
      const tab = this.scanTabs.find(t => t.id === tabId)
      if (!tab) return

      tab.config.enableFirewallDetection = false
      tab.config.disableFirewallDetection = true
      try {
        if (tab.scanning) {
          await this.stopScan(tab)
        }
        this.resetScanState(tab)
        await this.startScanInternal(tab, true)
        this.$message.success({
          message: '已关闭防火墙检测，开始全端口重扫',
          duration: 2500
        })
      } catch (e) {
        this.$message.error({
          message: `全端口重扫启动失败: ${e?.message || e}`,
          duration: 2500
        })
      }
    },
    handleNotice(payload) {
      if (!payload?.message) return
      const tabId = payload.tabId
      if (tabId && !this.scanTabs.find(t => t.id === tabId)) {
        return
      }
      if (payload.type === 'firewall_auto_downgrade') {
        const key = `${tabId || 'global'}:${payload.host || ''}`
        if (this.firewallNoticePrompted.has(key)) {
          return
        }
        this.firewallNoticePrompted.add(key)
        if (this.autoFullPortRescanRemember) {
          this.restartFullPortScanFromNotice(payload)
          return
        }
        this.$confirm(
          payload.message,
          '检测到疑似防火墙策略',
          {
            type: 'warning',
            distinguishCancelAndClose: true,
            confirmButtonText: '继续全端口重扫',
            cancelButtonText: '保持自动降级'
          }
        ).then(() => {
          this.restartFullPortScanFromNotice(payload)
        }).catch(() => {})
        return
      }
      this.$message.warning({
        message: payload.message,
        duration: 5000
      })
    },
    isWebService(row) {
      if (!row || !row.open) return false
      const port = Number(row.port)
      const service = (row.service || '').toLowerCase()
      const protocol = (row.protocol || '').toLowerCase()
      const banner = (row.banner || '').toLowerCase()

      // 协议是HTTP或HTTPS
      if (protocol === 'http' || protocol === 'https') {
        return true
      }

      // 常见 Web 端口
      const webPorts = [80, 443, 8080, 8081, 8000, 8888, 8443, 3000, 7001, 4222]
      if (webPorts.includes(port)) {
        return true
      }

      // 服务名中包含典型 Web 关键字
      if (
        service.includes('http') ||
        service.includes('nginx') ||
        service.includes('apache') ||
        service.includes('tomcat') ||
        service.includes('iis') ||
        service.includes('vrml')
      ) {
        return true
      }

      // Banner中包含HTTP特征或JSON格式（可能是API服务）
      if (
        banner.includes('http/') ||
        banner.includes('content-type') ||
        banner.includes('server:') ||
        (banner.includes('{') && banner.includes('"server_id"')) ||
        (banner.includes('{') && banner.includes('"server"'))
      ) {
        return true
      }

      return false
    },
    getWebScheme(row) {
      const service = (row.service || '').toLowerCase()
      const port = Number(row.port)
      if (port === 443 || service.includes('https')) {
        return 'https'
      }
      return 'http'
    },
    getTargetURL(row) {
      if (!row || !row.target || !row.port) {
        return '-'
      }
      
      const protocol = row.protocol || 'tcp'
      const target = row.target
      const port = row.port
      
      // 如果是HTTP/HTTPS协议，拼接完整URL
      if (protocol === 'http' || protocol === 'https') {
        // 标准端口不需要显示端口号
        if ((protocol === 'http' && port === 80) || (protocol === 'https' && port === 443)) {
          return `${protocol}://${target}`
        }
        return `${protocol}://${target}:${port}`
      }
      
      // 其他协议显示为 protocol://target:port
      return `${protocol}://${target}:${port}`
    },
    openURL(row) {
      const url = this.getTargetURL(row)
      if (url && url.startsWith('http')) {
        try {
          BrowserOpenURL(url)
          this.$message.success({
            message: `正在打开: ${url}`,
            duration: 2000
          })
        } catch (error) {
          this.$message.error({
            message: `打开URL失败: ${error?.message || error}`,
            duration: 2000
          })
        }
      } else {
        this.$message.warning({
          message: '仅支持HTTP/HTTPS协议',
          duration: 2000
        })
      }
    },
    runFingerprint(row, autoSwitch = true, tab = null) {
      // 如果 tab 为 null 且 autoSwitch 为 true（手动触发），尝试从当前标签获取
      // 如果 autoSwitch 为 false（自动联动），即使 tab 为 null 也继续执行（标签可能已关闭）
      if (!tab && autoSwitch) {
        tab = this.currentTab
      }
      
      if (!this.isWebService(row)) {
        this.$message.warning({
          message: '仅支持 Web 服务联动指纹识别',
          duration: 2000
        })
        return
      }
      const scheme = this.getWebScheme(row)
      const url = `${scheme}://${row.target}:${row.port}`
      const cfg = {
        targets: [url],
        timeout: 8,
        threads: 4,
        dirThreads: 4,
        maxDirChecks: 400,
        dirMode: false,
        useWappalyzer: true,
        useTideRules: true,
        useCmsDict: true,
        usePoweredBy: true,
        followRedirect: true,
        headers: {}
      }
      
      // 获取 sourceTaskId（优先从 tab 获取，如果 tab 不存在则尝试通过 row 的 tabId 查找任务）
      let sourceTaskId = null
      if (tab && tab.taskId) {
        sourceTaskId = tab.taskId
      } else if (row && row.tabId) {
        // 如果标签不存在，通过 row.tabId 查找任务
        const task = Object.values(this.tasksLocal).find(t => t.config?.tabId === row.tabId)
        if (task) {
          sourceTaskId = task.taskId || task.id
        }
      }
      
      // 自动填充URL到指纹识别模块，传递来源任务信息
      window.dispatchEvent(new CustomEvent('cyberSabre-fingerprint-fill', { 
        detail: { 
          targets: [url], 
          config: cfg,
          sourceTaskId: sourceTaskId,
          sourceTaskType: 'hostscan'
        } 
      }))
      
      if (autoSwitch && tab) {
        // 手动触发时才跳转（需要标签存在）
        StartFingerprintScan(JSON.stringify(cfg))
        window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'fingerprint' }))
        this.$message.info(`已触发指纹识别: ${url}`)
      } else if (!autoSwitch) {
        // 自动联动时，如果组件已挂载但未激活，直接启动扫描以确保任务能够执行
        // 事件会被暂存，等用户切换到该标签页时也能看到数据
        if (window.__fingerprintReady && !window.__fingerprintActivated) {
          StartFingerprintScan(JSON.stringify(cfg))
        }
      }
      // 自动联动时静默处理，不显示提示消息（实际扫描会在handleExternalFingerprintFill中启动）
    },
    runDirsearch(row, autoSwitch = true, tab = null) {
      // 如果 tab 为 null 且 autoSwitch 为 true（手动触发），尝试从当前标签获取
      // 如果 autoSwitch 为 false（自动联动），即使 tab 为 null 也继续执行（标签可能已关闭）
      if (!tab && autoSwitch) {
        tab = this.currentTab
      }
      
      if (!this.isWebService(row)) {
        this.$message.warning({
          message: '仅支持 Web 服务联动目录扫描',
          duration: 2000
        })
        return
      }
      const scheme = this.getWebScheme(row)
      const url = `${scheme}://${row.target}:${row.port}`
      const cfg = {
        targets: [url],
        method: 'GET',
        threads: 10,
        timeout: 10,
        recursionDepth: 0,
        followRedirects: true,
        // 传递字符串形式，交由 Dirsearch 视图统一解析
        customHeaders: '',
        builtinDicts: ['dicc.txt'], // 使用默认字典，避免无字典导致立即完成
        customDict: '',
        fuzz: {
          enabled: false,
          position: 'suffix',
          length: 3,
          charset: ['lower', 'digits'],
          customChars: '',
          generateCount: 1000
        },
        filter: {
          // 使用字符串数组，Dirsearch 会在界面层做转换
          statusCodes: ['404'], // 默认过滤404状态码
          sizeRange: [0, 1000000],
          // 关键字用数组传入，Dirsearch 会 join 成 textarea 文本
          keywords: [],
          regex: ''
        }
      }
      
      // 获取 sourceTaskId（优先从 tab 获取，如果 tab 不存在则尝试通过 row 的 tabId 查找任务）
      let sourceTaskId = null
      if (tab && tab.taskId) {
        sourceTaskId = tab.taskId
      } else if (row && row.tabId) {
        // 如果标签不存在，通过 row.tabId 查找任务
        const task = Object.values(this.tasksLocal).find(t => t.config?.tabId === row.tabId)
        if (task) {
          sourceTaskId = task.taskId || task.id
        }
      }
      
      // 自动填充URL到目录扫描模块（将配置展开到顶层，兼容applyExternalScanConfig）
      const fillPayload = {
        targets: [url],
        ...cfg,
        sourceTaskId: sourceTaskId,
        sourceTaskType: 'hostscan'
      }
      window.dispatchEvent(new CustomEvent('cyberSabre-dirscan-fill', { detail: fillPayload }))
      
      if (autoSwitch && tab) {
        // 手动触发时才跳转（需要标签存在）
        window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'dirsearch' }))
        setTimeout(() => {
          window.dispatchEvent(new CustomEvent('cyberSabre-dirscan-request', { detail: fillPayload }))
        }, 0)
        this.$message.info({
          message: `已触发目录扫描: ${url}`,
          duration: 2000
        })
      } else if (!autoSwitch) {
        // 自动联动时，如果组件已挂载但未激活，发送请求事件以确保任务能够执行
        // 事件会被暂存，等用户切换到该标签页时也能看到数据
        if (window.__dirsearchReady && !window.__dirsearchActivated) {
          setTimeout(() => {
            window.dispatchEvent(new CustomEvent('cyberSabre-dirscan-request', { detail: fillPayload }))
          }, 100)
        }
      }
      // 自动联动时静默处理，不显示提示消息（实际扫描会在handleExternalDirscanFill中启动）
    },
    async runJsfinder(row, autoSwitch = true, tab = null) {
      if (!this.isWebService(row)) {
        this.$message.warning({
          message: '仅支持 Web 服务联动 JSFinder 接口探测',
          duration: 2000
        })
        return
      }

      const scheme = this.getWebScheme(row)
      const url = `${scheme}://${row.target}:${row.port}`

      try {
        const config = {
          wait_time: 5,
          extract_rules: [],
          api_routes: ['/api', '/api/v1', '/api/v2', '/rest', '/graphql'],
          global_api_routes: ['/api', '/api/v1', '/api/v2'],
          allow_list: [],
          // 联动场景默认只做 JS 解析和接口发现，不自动发包测试，避免触发授权
          auto_test_api: false,
          test_methods: ['GET', 'POST'],
          worker_num: 10
        }

        await CreateBrowserScanTask(url, config)

        if (autoSwitch && tab) {
          window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'browser-scan' }))
          this.$message.success({
            message: `已触发 JSFinder 接口探测: ${url}`,
            duration: 2000
          })
        }
      } catch (error) {
        this.$message.error({
          message: `JSFinder 接口探测失败: ${error?.message || error}`,
          duration: 2000
        })
      }
    },
    async startVulnScanFromResults(tab) {
      // 如果 tab 为 null，尝试从当前标签获取；如果都没有，可能标签已关闭，但可以尝试从任务数据获取
      if (!tab) tab = this.currentTab
      if (!tab) return
      
      // 获取结果数据（可能是标签结果或任务结果）
      let results = tab.results || []
      if (!results.length) {
        // 如果标签没有结果，尝试从任务数据获取（标签可能已关闭）
        if (tab.taskId && this.tasksLocal[tab.taskId] && this.tasksLocal[tab.taskId].results) {
          results = this.tasksLocal[tab.taskId].results
        }
      }
      
      if (!results.length) {
        this.$message.warning({
          message: '当前没有可用于漏洞扫描的端口结果',
          duration: 2000
        })
        return
      }

      const webUrlsSet = new Set()
      results.forEach(r => {
        if (!r.open || !this.isWebService(r)) return
        const scheme = this.getWebScheme(r)
        const url = this.getTargetURL({ ...r, protocol: scheme })
        if (url && url.startsWith('http')) {
          const normalized = url.endsWith('/') ? url : `${url}/`
          webUrlsSet.add(normalized)
        }
      })

      const webUrls = Array.from(webUrlsSet)
      if (!webUrls.length) {
        this.$message.info({
          message: '未发现可用于漏洞扫描的 Web 服务',
          duration: 2000
        })
        return
      }

      let proxyString = ''
      try {
        const globalProxy = await GetGlobalProxy()
        if (globalProxy?.enabled && globalProxy.config) {
          const proxyConfig = globalProxy.config
          if (proxyConfig.type && proxyConfig.host && proxyConfig.port) {
            const scheme = proxyConfig.type.toLowerCase()
            const host = proxyConfig.host
            const port = proxyConfig.port
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
          templateFolders = (configData.template_folders || []).filter(Boolean)
        }
      } catch (e) {
        console.error('获取漏洞扫描配置失败:', e)
      }

      // 获取配置（优先从任务配置中获取，如果任务不存在则从标签配置中获取）
      let vulnSkipWithoutTags = false
      if (tab.taskId && this.tasksLocal[tab.taskId] && this.tasksLocal[tab.taskId].config) {
        vulnSkipWithoutTags = this.tasksLocal[tab.taskId].config.vulnSkipWithoutTags === true || this.tasksLocal[tab.taskId].config.vulnSkipWithoutTags === 'true'
      } else if (tab.config) {
        vulnSkipWithoutTags = tab.config.vulnSkipWithoutTags === true || tab.config.vulnSkipWithoutTags === 'true'
      }
      
      const taskName = `主机扫描-漏洞扫描-${new Date().toLocaleTimeString()}`
      const payload = {
        targets: webUrls,
        task_name: taskName,
        append_poc: '',
        tags: [],
        custom_tags: [],
        template_file: [],
        template_folders: templateFolders,
        custom_headers: '',
        proxy: proxyString,
        skip_nuclei_without_tags: vulnSkipWithoutTags,
        thread_safe: true,
        threads: scanConfig.threads,
        timeout: scanConfig.timeout,
        rate_limit: scanConfig.rate_limit,
        retries: scanConfig.retries
      }

      try {
        const taskId = await StartPocScan(JSON.stringify(payload))
        // 减少提示消息时长，避免与其他消息冲突
        this.$message.success({
          message: `已启动漏洞扫描任务，目标数量 ${webUrls.length}。请在"漏洞扫描"模块查看进度。`,
          duration: 2000
        })
        // 不再自动跳转到漏洞扫描标签页
      } catch (e) {
        console.error('启动漏洞扫描任务失败:', e)
        this.$message.error({
          message: `启动漏洞扫描任务失败: ${e?.message || e}`,
          duration: 3000
        })
      }
    },
    async startCrackFromResults(tab) {
      // 如果 tab 为 null，尝试从当前标签获取；如果都没有，可能标签已关闭，但可以尝试从任务数据获取
      if (!tab) tab = this.currentTab
      
      // 获取结果数据（可能是标签结果或任务结果）
      let results = []
      let taskId = null
      
      if (tab) {
        // 如果 tab 有 taskId，说明可能是临时对象或真实标签页
        if (tab.taskId) {
          taskId = tab.taskId
          // 优先从任务数据获取结果（最完整）
          if (this.tasksLocal[taskId] && this.tasksLocal[taskId].results) {
            results = this.tasksLocal[taskId].results
          } else if (tab.results && tab.results.length > 0) {
            // 如果任务数据没有结果，从 tab 获取
            results = tab.results
          }
        } else if (tab.results && tab.results.length > 0) {
          // 如果是真实标签页但没有 taskId（不应该发生），从 tab 获取
          results = tab.results
        }
      }
      
      if (!taskId && tab && tab.taskId) {
        taskId = tab.taskId
      }
      
      if (!results.length) {
        this.$message.warning({
          message: '当前没有可用于密码爆破的端口结果',
          duration: 2000
        })
        return
      }

      // 支持的协议及其端口与别名，保持与密码爆破模块的服务列表一致
      const protocolPorts = [
        { name: 'ssh', defaultPort: 22, aliases: ['ssh'] },
        { name: 'rdp', defaultPort: 3389, aliases: ['rdp', 'ms-wbt-server'] },
        { name: 'smb', defaultPort: 445, aliases: ['smb', 'microsoft-ds'] },
        { name: 'mysql', defaultPort: 3306, aliases: ['mysql'] },
        { name: 'mssql', defaultPort: 1433, aliases: ['mssql', 'sqlserver', 'microsoft-sql-server'] },
        { name: 'oracle', defaultPort: 1521, aliases: ['oracle'] },
        { name: 'postgresql', defaultPort: 5432, aliases: ['postgresql', 'postgres'] },
        { name: 'redis', defaultPort: 6379, aliases: ['redis'] },
        { name: 'mongodb', defaultPort: 27017, aliases: ['mongodb', 'mongo'] },
        { name: 'dm', defaultPort: 5236, aliases: ['dm', 'dmsql'] },
        { name: 'memcached', defaultPort: 11211, aliases: ['memcached'] },
        { name: 'elasticsearch', defaultPort: 9200, aliases: ['elasticsearch', 'es'] },
        { name: 'ftp', defaultPort: 21, aliases: ['ftp'] },
        { name: 'telnet', defaultPort: 23, aliases: ['telnet'] },
        { name: 'winrm', defaultPort: 5985, aliases: ['winrm', 'wsman'] },
        { name: 'vnc', defaultPort: 5900, aliases: ['vnc', 'rfb'] },
        { name: 'svn', defaultPort: 3690, aliases: ['svn', 'subversion'] },
        { name: 'tomcat', defaultPort: 8080, aliases: ['tomcat'] },
        { name: 'weblogic', defaultPort: 7001, aliases: ['weblogic'] },
        { name: 'jboss', defaultPort: 8081, aliases: ['jboss', 'wildfly'] },
        { name: 'zookeeper', defaultPort: 2181, aliases: ['zookeeper'] },
        { name: 'activemq', defaultPort: 61613, aliases: ['activemq', 'stomp'] },
        { name: 'rabbitmq', defaultPort: 5672, aliases: ['rabbitmq', 'amqp'] },
        { name: 'ldap', defaultPort: 389, aliases: ['ldap'] },
        { name: 'socks5', defaultPort: 1080, aliases: ['socks5'] }
      ]

      const targets = []
      const protocolMapForEvent = {}
      results.forEach(r => {
        if (!r.open) return
        const port = Number(r.port)
        const service = (r.service || '').toLowerCase()
        const product = (r.product || '').toLowerCase()
        const protocol = (r.protocol || '').toLowerCase()

        let proto = null
        for (const item of protocolPorts) {
          const matchedByAlias = item.aliases?.some(alias =>
            service.includes(alias) ||
            product.includes(alias) ||
            protocol === alias
          )
          if (
            protocol === item.name ||
            service.includes(item.name) ||
            matchedByAlias ||
            port === item.defaultPort
          ) {
            proto = item.name
            break
          }
        }
        if (!proto) return
        const url = `${proto}://${r.target}:${r.port}`
        targets.push({ url, protocol: proto, port })
        if (!protocolMapForEvent[proto.toUpperCase()]) {
          protocolMapForEvent[proto.toUpperCase()] = new Set()
        }
        protocolMapForEvent[proto.toUpperCase()].add(port)
      })

      if (!targets.length) {
        this.$message.info({
          message: '未发现可进行密码爆破的服务',
          duration: 2000
        })
        return
      }

      // 获取配置（优先从任务配置中获取，如果任务不存在则从标签配置中获取）
      let crackBuiltInPassword = false
      let crackPasswords = ''
      let crackBuiltInUsername = false
      let crackUsernames = ''
      
      if (taskId && this.tasksLocal[taskId] && this.tasksLocal[taskId].config) {
        // 优先从任务配置获取（标签可能已关闭）
        const taskConfig = this.tasksLocal[taskId].config
        crackBuiltInPassword = taskConfig.crackBuiltInPassword === true || taskConfig.crackBuiltInPassword === 'true'
        crackPasswords = taskConfig.crackPasswords || ''
        crackBuiltInUsername = taskConfig.crackBuiltInUsername === true || taskConfig.crackBuiltInUsername === 'true'
        crackUsernames = taskConfig.crackUsernames || ''
      } else if (tab && tab.config) {
        // 如果任务配置不存在，从标签配置获取
        crackBuiltInPassword = tab.config.crackBuiltInPassword === true || tab.config.crackBuiltInPassword === 'true'
        crackPasswords = tab.config.crackPasswords || ''
        crackBuiltInUsername = tab.config.crackBuiltInUsername === true || tab.config.crackBuiltInUsername === 'true'
        crackUsernames = tab.config.crackUsernames || ''
      }
      
      try {
        // 构建密码字典：内置 + 自定义
        let passwords = []
        if (crackBuiltInPassword) {
          const builtinPasswords = await LoadPasswordDictFromFile()
          if (Array.isArray(builtinPasswords)) {
            passwords.push(...builtinPasswords)
          }
        }
        if (crackPasswords && crackPasswords.trim()) {
          const extraPass = crackPasswords
            .split('\n')
            .map(p => p.trim())
            .filter(p => p.length > 0)
          passwords.push(...extraPass)
        }
        // 去重
        passwords = Array.from(new Set(passwords))
        if (!passwords.length) {
          this.$message.warning({
            message: '当前密码字典为空，请勾选使用内置密码字典或填写自定义密码后再尝试',
            duration: 3000
          })
          return
        }

        const usernameCache = {}
        let started = 0

        for (const target of targets) {
          const proto = target.protocol
          if (!usernameCache[proto]) {
            let users = []
            if (crackBuiltInUsername) {
              try {
                const builtinUsers = await LoadUsernameDictFromFile(proto)
                if (Array.isArray(builtinUsers)) {
                  users.push(...builtinUsers)
                }
              } catch (e) {
                console.error(`加载协议 ${proto} 用户名字典失败:`, e)
              }
            }
            if (crackUsernames && crackUsernames.trim()) {
              const extraUsers = crackUsernames
                .split('\n')
                .map(u => u.trim())
                .filter(u => u.length > 0)
              users.push(...extraUsers)
            }
            // 去重
            usernameCache[proto] = Array.from(new Set(users))
          }
          const usernames = usernameCache[proto]
          if (!usernames.length) {
            console.warn(`协议 ${proto} 没有可用的用户名字典，跳过 ${target.url}`)
            continue
          }
          try {
            // 使用固定的taskId格式，确保所有密码爆破结果追加到同一个标签页
            const crackTaskId = `hostscan-crack-${taskId || 'default'}`
            // SSH隧道配置（暂时为空）
            const sshConfigJSON = ''
            // hostscan 联动使用默认配置
            const defaultConfigJSON = JSON.stringify({
              threads: 5,
              timeout: 3000,
              only_one_account: true
            })
            await NewCrackScanenr(crackTaskId, target.url, usernames, passwords, sshConfigJSON, defaultConfigJSON)
            started++
          } catch (e) {
            console.error(`启动对 ${target.url} 的密码爆破失败:`, e)
          }
        }

        if (started > 0) {
          // 发送事件到密码爆破模块，自动勾选协议和端口
          const targetLines = []
          targets.forEach(target => {
            const match = target.url.match(/^(\w+):\/\/([^:]+):(\d+)$/)
            if (match) {
              const [, , host] = match
              if (!targetLines.includes(host)) {
                targetLines.push(host)
              }
            }
          })

          // 将 Set 转为数组，便于前端直接使用
          const protocolMapPayload = {}
          Object.keys(protocolMapForEvent).forEach(key => {
            protocolMapPayload[key] = Array.from(protocolMapForEvent[key])
          })
          
          // 触发事件，通知密码爆破模块
          window.dispatchEvent(new CustomEvent('send-to-crack', {
            detail: {
              targets: targets.map(t => t.url).filter(Boolean),
              hosts: targetLines,
              protocols: protocolMapPayload
            }
          }))
          
          // 通知密码爆破模块刷新数据（延迟一下，确保后端已经保存了数据）
          setTimeout(() => {
            window.dispatchEvent(new CustomEvent('refresh-crack-results', {}))
          }, 1000)
          
          // 减少提示消息的显示时间，避免与其他消息冲突
          this.$message.success({
            message: `已启动 ${started} 个密码爆破任务。结果将在"密码爆破"模块中显示`,
            duration: 3000,
            showClose: true
          })
        } else {
          this.$message.warning({
            message: '未成功启动任何密码爆破任务，请检查服务类型与字典配置',
            duration: 2000
          })
        }
      } catch (e) {
        console.error('从主机扫描结果启动密码爆破失败:', e)
        this.$message.error({
          message: `启动密码爆破失败: ${e?.message || e}`,
          duration: 3000
        })
      }
    },
    runVulnScan(row, tab) {
      if (!tab) tab = this.currentTab
      if (!this.isWebService(row)) {
        this.$message.warning({
          message: '仅支持 Web 服务联动漏洞扫描',
          duration: 2000
        })
        return
      }
      // 根据行数据构造 URL，优先使用 Web 协议
      const scheme = this.getWebScheme(row)
      const url = `${scheme}://${row.target}:${row.port}`
      
      // 构造传递给漏洞扫描模块的任务名称
      const taskName = `主机扫描-${row.target}:${row.port}`
      
      try {
        // 先切换到漏洞扫描标签页
        window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'webscan' }))
        // 再弹出“新建网站扫描”对话框，并预填目标和任务名
        setTimeout(() => {
          try {
            window.dispatchEvent(
              new CustomEvent('webscan-open-new-task', {
                detail: {
                  targets: [url],
                  taskName
                }
              })
            )
          } catch (e) {
            console.error('发送 webscan-open-new-task 事件失败:', e)
          }
        }, 300)
      } catch (e) {
        console.error('联动漏洞扫描失败:', e)
        this.$message.error({
          message: `联动漏洞扫描失败: ${e?.message || e}`,
          duration: 3000
        })
      }
    },
    clearResults(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      tab.results = []
      this.resetColumnFilters(tab)
      tab.sortField = ''
      tab.currentPage = 1
      tab.stats = {
        totalPorts: 0,
        checked: 0,
        openCount: 0,
        closedOrFiltered: 0,
        portsPerSecond: 0
      }
      this.saveHistory()
      this.$message.success({
        message: '已清空结果',
        duration: 2000
      })
    },
    exportToExcel(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      if (tab.results.length === 0) {
        this.$message.warning('没有可导出的数据')
        return
      }
      const data = tab.results.map((r, idx) => ({
        序号: idx + 1,
        目标: r.target,
        端口: r.port,
        状态: r.open ? '开放' : '关闭/过滤',
        服务: r.service || '',
        Banner: r.banner || '',
        标题: r.title || '',
        地区: r.location || '',
        错误: r.error || ''
      }))
      const wb = XLSX.utils.book_new()
      const ws = XLSX.utils.json_to_sheet(data)
      XLSX.utils.book_append_sheet(wb, ws, '主机扫描')
      XLSX.writeFile(wb, `主机扫描_${Date.now()}.xlsx`)
      this.$message.success('导出成功')
    },
    handleAliveCheckChange(tab) {
      if (!tab) tab = this.currentTab
      if (!tab) return
      const val = tab.config.checkAlive
      if (val && !tab.config.aliveCheck) {
        // 初始化默认配置，使用ICMP
        tab.config.aliveCheck = {
          type: 'icmp',
          ports: [80, 443, 22],
          url: 'http://{host}/',
          timeout: 3,
          enabled: true
        }
      }
      // 同步到aliveConfig用于对话框显示
      if (tab.config.aliveCheck) {
        this.aliveConfig = {
          type: tab.config.aliveCheck.type || 'icmp',
          portsText: Array.isArray(tab.config.aliveCheck.ports) 
            ? tab.config.aliveCheck.ports.join(',') 
            : '80,443,22',
          url: tab.config.aliveCheck.url || 'http://{host}/',
          timeout: tab.config.aliveCheck.timeout || 3,
          enabled: tab.config.aliveCheck.enabled !== false
        }
      }
    },
    saveAliveConfig() {
      if (!this.currentAliveTab) return
      const tab = this.currentAliveTab
      // 解析端口列表
      let ports = []
      if (this.aliveConfig.type === 'tcp_port' && this.aliveConfig.portsText) {
        ports = this.aliveConfig.portsText
          .split(',')
          .map(p => parseInt(p.trim()))
          .filter(p => !isNaN(p) && p > 0 && p <= 65535)
      }
      
      // 如果没有端口，使用默认端口
      if (ports.length === 0 && this.aliveConfig.type === 'tcp_port') {
        ports = [80, 443, 22]
      }
      
      tab.config.aliveCheck = {
        type: this.aliveConfig.type,
        ports: ports,
        url: this.aliveConfig.url || 'http://{host}/',
        timeout: this.aliveConfig.timeout || 3,
        enabled: true
      }
      
      this.showAliveConfigDialog = false
      this.$message.success('存活检测配置已保存')
    },
    handleRowContextMenu(row, column, event, tab) {
      event.preventDefault()
      event.stopPropagation()
      this.selectedRow = row
      this.selectedTab = tab || this.currentTab
      
      // 优先使用 clientX/clientY（相对于视口，适用于 position: fixed）
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
        
        // 只在会溢出时才向上微调，尽量保持靠近鼠标
        if (y + menuHeight > windowHeight - 8) {
          let adjustedY = windowHeight - menuHeight - 8
          if (adjustedY < 8) adjustedY = 8
          this.contextMenuPos = { x, y: adjustedY }
        }
      })
      
      // 点击其他地方关闭菜单
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
    handleCopyRow() {
      if (!this.selectedRow) return
      const rowText = `目标: ${this.selectedRow.target}\n端口: ${this.selectedRow.port}\n状态: ${this.selectedRow.open ? '开放' : '关闭/过滤'}\n服务: ${this.selectedRow.service || ''}\nBanner: ${this.selectedRow.banner || ''}\n标题: ${this.selectedRow.title || ''}\n地区: ${this.selectedRow.location || ''}`
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
        return `${idx + 1}. ${r.target}:${r.port} - ${r.open ? '开放' : '关闭/过滤'} - ${r.service || ''}`
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
      const index = this.selectedTab.results.findIndex(
        (r) => r.target === this.selectedRow.target && r.port === this.selectedRow.port
      )
      if (index !== -1) {
        this.selectedTab.results.splice(index, 1)
        this.selectedTab.stats.openCount = this.selectedTab.results.length
        this.saveHistory()
        this.$message.success('已删除')
      }
      this.contextMenuVisible = false
    },
    handleRunFingerprint() {
      if (!this.selectedRow || !this.selectedTab) return
      this.contextMenuVisible = false
      this.runFingerprint(this.selectedRow, true, this.selectedTab)
    },
    handleRunDirsearch() {
      if (!this.selectedRow || !this.selectedTab) return
      this.contextMenuVisible = false
      this.runDirsearch(this.selectedRow, true, this.selectedTab)
    },
    handleRunVulnScan() {
      if (!this.selectedRow || !this.selectedTab) return
      this.contextMenuVisible = false
      this.runVulnScan(this.selectedRow, this.selectedTab)
    },
    handleRunCrack() {
      if (!this.selectedRow || !this.selectedTab) return
      this.contextMenuVisible = false
      this.runCrack(this.selectedRow, this.selectedTab)
    },
    handleRunJsfinder() {
      if (!this.selectedRow || !this.selectedTab) return
      this.contextMenuVisible = false
      this.runJsfinder(this.selectedRow, true, this.selectedTab)
    },
    isCrackableProtocol(row) {
      if (!row || !row.open) return false
      const port = Number(row.port)
      const service = (row.service || '').toLowerCase()
      const protocol = (row.protocol || '').toLowerCase()
      
      // 支持的协议列表（与crackController.go中的crackScanners保持一致）
      const supportedProtocols = [
        { name: 'ftp', defaultPort: 21 },
        { name: 'ssh', defaultPort: 22 },
        { name: 'telnet', defaultPort: 23 },
        { name: 'ldap', defaultPort: 389 },
        { name: 'smb', defaultPort: 445 },
        { name: 'rdp', defaultPort: 3389 },
        { name: 'mysql', defaultPort: 3306 },
        { name: 'mssql', defaultPort: 1433 },
        { name: 'postgresql', defaultPort: 5432 },
        { name: 'oracle', defaultPort: 1521 },
        { name: 'redis', defaultPort: 6379 },
        { name: 'mongodb', defaultPort: 27017 },
        { name: 'vnc', defaultPort: 5900 },
        { name: 'socks5', defaultPort: 1080 },
        { name: 'mqtt', defaultPort: 1883 },
        { name: 'memcached', defaultPort: 11211 },
        { name: 'activemq', defaultPort: 61616 },
        { name: 'rsync', defaultPort: 873 },
        { name: 'kafka', defaultPort: 9092 },
        { name: 'rmi', defaultPort: 1099 },
        { name: 'jdwp', defaultPort: 5005 },
        { name: 'adb', defaultPort: 5555 }
      ]
      
      // 检查是否匹配支持的协议
      for (const item of supportedProtocols) {
        if (
          protocol === item.name ||
          service.includes(item.name) ||
          port === item.defaultPort
        ) {
          return true
        }
      }
      
      return false
    },
    async runCrack(row, tab) {
      if (!tab) tab = this.currentTab
      if (!this.isCrackableProtocol(row)) {
        this.$message.warning({
          message: '当前端口不支持密码爆破',
          duration: 2000
        })
        return
      }
      
      const protocolPorts = [
        { name: 'ftp', defaultPort: 21 },
        { name: 'ssh', defaultPort: 22 },
        { name: 'telnet', defaultPort: 23 },
        { name: 'ldap', defaultPort: 389 },
        { name: 'smb', defaultPort: 445 },
        { name: 'rdp', defaultPort: 3389 },
        { name: 'mysql', defaultPort: 3306 },
        { name: 'mssql', defaultPort: 1433 },
        { name: 'postgresql', defaultPort: 5432 },
        { name: 'oracle', defaultPort: 1521 },
        { name: 'redis', defaultPort: 6379 },
        { name: 'mongodb', defaultPort: 27017 },
        { name: 'vnc', defaultPort: 5900 },
        { name: 'socks5', defaultPort: 1080 },
        { name: 'mqtt', defaultPort: 1883 },
        { name: 'memcached', defaultPort: 11211 },
        { name: 'activemq', defaultPort: 61616 },
        { name: 'rsync', defaultPort: 873 },
        { name: 'kafka', defaultPort: 9092 },
        { name: 'rmi', defaultPort: 1099 },
        { name: 'jdwp', defaultPort: 5005 },
        { name: 'adb', defaultPort: 5555 }
      ]
      
      const port = Number(row.port)
      const service = (row.service || '').toLowerCase()
      const protocol = (row.protocol || '').toLowerCase()
      
      let proto = null
      for (const item of protocolPorts) {
        if (
          protocol === item.name ||
          service.includes(item.name) ||
          port === item.defaultPort
        ) {
          proto = item.name
          break
        }
      }
      
      if (!proto) {
        this.$message.warning({
          message: '无法识别协议类型',
          duration: 2000
        })
        return
      }
      
      const target = `${proto}://${row.target}:${row.port}`
      
      try {
        // 构建密码字典：内置 + 自定义
        let passwords = []
        if (tab.config.crackBuiltInPassword) {
          const builtinPasswords = await LoadPasswordDictFromFile()
          if (Array.isArray(builtinPasswords)) {
            passwords.push(...builtinPasswords)
          }
        }
        if (tab.config.crackPasswords && tab.config.crackPasswords.trim()) {
          const extraPass = tab.config.crackPasswords
            .split('\n')
            .map(p => p.trim())
            .filter(p => p.length > 0)
          passwords.push(...extraPass)
        }
        // 去重
        passwords = Array.from(new Set(passwords))
        if (!passwords.length) {
          this.$message.warning({
            message: '当前密码字典为空，请勾选使用内置密码字典或填写自定义密码后再尝试',
            duration: 3000
          })
          return
        }
        
        // 构建用户名字典
        let usernames = []
        if (tab.config.crackBuiltInUsername) {
          try {
            const builtinUsers = await LoadUsernameDictFromFile(proto)
            if (Array.isArray(builtinUsers)) {
              usernames.push(...builtinUsers)
            }
          } catch (e) {
            console.error(`加载协议 ${proto} 用户名字典失败:`, e)
          }
        }
        if (tab.config.crackUsernames && tab.config.crackUsernames.trim()) {
          const extraUsers = tab.config.crackUsernames
            .split('\n')
            .map(u => u.trim())
            .filter(u => u.length > 0)
          usernames.push(...extraUsers)
        }
        // 去重
        usernames = Array.from(new Set(usernames))
        if (!usernames.length) {
          this.$message.warning({
            message: '当前用户名字典为空，请勾选使用内置用户名字典或填写自定义用户名后再尝试',
            duration: 3000
          })
          return
        }
        
        const taskId = `hostscan-crack-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
        // SSH隧道配置（暂时为空）
        const sshConfigJSON = ''
        // hostscan 联动使用默认配置
        const defaultConfigJSON = JSON.stringify({
          threads: 5,
          timeout: 3000,
          only_one_account: true
        })
        await NewCrackScanenr(taskId, target, usernames, passwords, sshConfigJSON, defaultConfigJSON)
        this.$message.success({
          message: `已启动密码爆破任务：${target}。爆破结果将在"渗透测试-漏洞扫描"模块中显示（以漏洞形式展示，如${proto.toUpperCase()}弱口令等）`,
          duration: 5000,
          showClose: true
        })
      } catch (e) {
        console.error('启动密码爆破失败:', e)
        this.$message.error({
          message: `启动密码爆破失败: ${e?.message || e}`,
          duration: 3000
        })
      }
    },
    getPresetPortsList(preset) {
      // 根据预设返回端口列表
      switch (preset) {
        case 'top100':
          return [
          80,23,443,21,22,25,3389,110,445,139,143,53,135,3306,8080,1723,111,5900,8888,81,10000,514,5060,2000,8443,8000,32768,554,1433,49152,2001,8008,49154,5666,5000,5631,49153,8081,2049,88,79,5800,2121,1110,49155,6000,513,990,5357,543,544,5101,389,8009,444,9999,5009,7070,5190,3000,5432,3986,1900,6646,5051,49157,873,1755,2717,4899,9100,82,83,85,888,1521,3443,4430,4433,4443,5443,5985,6379,7001,8001,8002,8003,8010,8082,8086,8088,8089,8090,9000,9043,9200,9443,10443,11211,27017,8848,7001
          ]
        case 'top1000':
          const commonPorts = [
            7,9,11,13,17,19,21,22,23,26,37,38,43,49,51,53,67,69,70,79,80,81,82,83,84,85,86,88,89,102,104,111,113,119,121,123,135,137,138,139,143,161,175,179,199,211,264,311,389,443,444,445,465,500,502,503,512,515,520,548,554,564,587,623,631,636,646,666,771,777,789,800,801,808,853,873,880,888,902,992,993,995,999,1000,1022,1023,1024,1025,1026,1027,1080,1099,1177,1194,1200,1201,1234,1241,1248,1260,1290,1311,1344,1400,1433,1443,1471,1494,1515,1521,1554,1588,1701,1720,1723,1741,1777,1863,1883,1900,1911,1935,1962,1967,1991,2000,2001,2002,2020,2022,2030,2031,2049,2052,2053,2077,2080,2082,2083,2086,2087,2095,2096,2121,2123,2152,2181,2222,2223,2252,2323,2332,2375,2376,2379,2401,2404,2424,2427,2443,2455,2480,2501,2601,2628,3000,3001,3002,3006,3128,3260,3283,3288,3299,3306,3307,3310,3333,3388,3389,3390,3443,3460,3541,3542,3689,3690,3749,3780,4000,4022,4040,4063,4064,4111,4343,4369,4430,4433,4443,4444,4500,4505,4567,4664,4712,4730,4782,4786,4840,4848,4880,4911,4949,5000,5001,5002,5004,5005,5006,5007,5009,5050,5060,5084,5222,5258,5269,5351,5353,5357,5400,5432,5443,5500,5554,5555,5556,5560,5577,5601,5631,5672,5678,5683,5800,5801,5900,5901,5902,5903,5938,5984,5985,5986,6000,6001,6002,6003,6005,6006,6050,6060,6068,6110,6363,6379,6443,6488,6560,6565,6581,6588,6590,6600,6664,6665,6666,6667,6668,6669,6699,6881,6998,7000,7001,7002,7003,7005,7006,7010,7014,7025,7070,7071,7077,7080,7170,7288,7306,7307,7312,7401,7443,7474,7493,7537,7547,7548,7634,7657,7777,7779,7788,7911,8000,8001,8002,8003,8004,8006,8008,8009,8010,8015,8020,8025,8030,8040,8060,8069,8080,8081,8082,8083,8084,8085,8086,8087,8088,8089,8090,8091,8092,8093,8094,8095,8096,8097,8098,8099,8111,8112,8118,8123,8125,8126,8139,8159,8161,8180,8181,8182,8200,8222,8291,8333,8334,8377,8378,8388,8443,8500,8545,8546,8554,8649,8686,8800,8834,8880,8883,8887,8888,8889,8899,8983,8999,9000,9001,9002,9003,9009,9010,9030,9042,9050,9051,9080,9083,9090,9091,9100,9151,9191,9200,9295,9333,9418,9443,9444,9527,9530,9595,9600,9653,9700,9711,9869,9944,9981,9997,9999,10000,10001,10003,10162,10243,10333,10443,10554,11001,11211,11300,11310,12300,12345,13579,14000,14147,14265,16010,16030,16992,16993,17000,18000,18001,18080,18081,18245,18246,19999,20000,20547,20880,22105,22222,23023,23424,25000,25105,25565,27015,27017,28017,28080,29876,30001,32400,33338,33890,37215,37777,41795,45554,49151,49152,49153,49154,49155,50000,50050,50070,50100,51106,52869,55442,55553,55555,60001,60010,60030,60443,61613,61616,62078,64738,8848
          ]
          return commonPorts
      }
    },
    isJSONBanner(banner) {
      if (!banner) return false
      const trimmed = banner.trim()
      return trimmed.startsWith('{') && trimmed.endsWith('}') || 
             trimmed.startsWith('[') && trimmed.endsWith(']') ||
             (trimmed.includes('"server_id"') || trimmed.includes('"server"') || trimmed.includes('"version"'))
    },
    formatJSONBanner(banner) {
      if (!banner) return ''
      try {
        // 尝试解析JSON
        let jsonStr = banner.trim()
        // 如果banner以INFO开头，提取JSON部分
        if (jsonStr.startsWith('INFO ')) {
          jsonStr = jsonStr.substring(5).trim()
        }
        const obj = JSON.parse(jsonStr)
        // 格式化JSON为可读的HTML
        const formatted = JSON.stringify(obj, null, 2)
        // 将换行符转换为<br>
        return formatted.replace(/\n/g, '<br>').replace(/ /g, '&nbsp;')
      } catch (e) {
        // 如果解析失败，返回原始内容
        return banner.replace(/\n/g, '<br>')
      }
    },
    formatJSONBannerShort(banner) {
      if (!banner) return ''
      try {
        let jsonStr = banner.trim()
        // 如果banner以INFO开头，提取JSON部分
        if (jsonStr.startsWith('INFO ')) {
          jsonStr = jsonStr.substring(5).trim()
        }
        const obj = JSON.parse(jsonStr)
        // 提取关键信息
        const parts = []
        if (obj.server_name) parts.push(`Server: ${obj.server_name}`)
        if (obj.version) parts.push(`Version: ${obj.version}`)
        if (obj.server_id) parts.push(`ID: ${obj.server_id.substring(0, 16)}...`)
        if (obj.go) parts.push(`Go: ${obj.go}`)
        if (obj.cluster) parts.push(`Cluster: ${obj.cluster}`)
        // 如果没有提取到关键信息，显示前100个字符
        if (parts.length === 0) {
          return truncateBanner(banner, 100)
        }
        return parts.join(' | ')
      } catch (e) {
        // 如果解析失败，返回截断的原始内容
        return truncateBanner(banner, 100)
      }
    },
    truncateBanner(text, maxLength = 150) {
      if (!text) return ''
      if (text.length <= maxLength) return text
      return text.substring(0, maxLength) + '...'
    },
    getServiceDisplay(row) {
      // 优先显示service字段（具体服务名：redis、mysql、ssh等）
      if (row.service && row.service !== 'tcp' && row.service !== 'unknown' && row.service !== '') {
        // 如果是http-alt，显示为http
        if (row.service === 'http-alt') {
          return 'http'
        }
        return row.service
      }
      
      // 如果service为空或是tcp/unknown，根据banner内容推断服务名
      if (row.banner) {
        const bannerLower = row.banner.toLowerCase()
        
        // SSH服务识别
        if (bannerLower.includes('ssh-') || bannerLower.startsWith('ssh-')) {
          return 'ssh'
        }
        // MySQL服务识别
        if (bannerLower.includes('mysql')) {
          return 'mysql'
        }
        // Redis服务识别
        if (bannerLower.includes('redis') || 
            bannerLower.includes('+pong') || 
            bannerLower.includes('+ok') ||
            bannerLower.startsWith('+') ||
            bannerLower.includes('redis_version')) {
          return 'redis'
        }
        // FTP服务识别
        if (bannerLower.startsWith('220') || bannerLower.includes('ftp')) {
          return 'ftp'
        }
        // Telnet服务识别
        if (bannerLower.includes('telnet') || bannerLower.includes('login:')) {
          return 'telnet'
        }
        // SMTP服务识别
        if (bannerLower.startsWith('220') && bannerLower.includes('smtp')) {
          return 'smtp'
        }
        // PostgreSQL服务识别
        if (bannerLower.includes('postgresql') || bannerLower.includes('postgres')) {
          return 'postgresql'
        }
        // MongoDB服务识别
        if (bannerLower.includes('mongodb')) {
          return 'mongodb'
        }
        // MSSQL服务识别
        if (bannerLower.includes('sql server') || bannerLower.includes('mssql') || bannerLower.includes('microsoft sql')) {
          return 'mssql'
        }
        // SMB服务识别
        if (bannerLower.includes('smb') || bannerLower.includes('samba')) {
          return 'smb'
        }
        // RDP服务识别
        if (bannerLower.includes('rdp') || bannerLower.includes('microsoft remote desktop')) {
          return 'rdp'
        }
        // VNC服务识别
        if (bannerLower.includes('rfb') || bannerLower.includes('vnc')) {
          return 'vnc'
        }
        // VRML-MULTI-USE服务识别（NATS服务器等）
        // 检查是否以"INFO "开头且包含server_id的JSON格式
        if (bannerLower.startsWith('info ') || bannerLower.startsWith('info{')) {
          // 检查是否包含server_id、server_name等特征
          if (bannerLower.includes('server_id') || 
              bannerLower.includes('server_name') || 
              bannerLower.includes('"server"') ||
              (bannerLower.includes('"version"') && bannerLower.includes('"go"'))) {
            // 检查是否是NATS服务器（通常包含go版本信息）
            if (bannerLower.includes('"go"') || bannerLower.includes('"proto"')) {
              return 'nats'
            }
            // 其他VRML服务
            if (bannerLower.includes('vrml')) {
              return 'vrml-multi-use'
            }
            // 如果包含server_id但没有vrml，可能是其他JSON API服务
            return 'api-json'
          }
        }
        // 检查JSON格式的响应（即使不以INFO开头）
        if (bannerLower.trim().startsWith('{') && 
            (bannerLower.includes('server_id') || bannerLower.includes('"server"'))) {
          if (bannerLower.includes('vrml')) {
            return 'vrml-multi-use'
          }
          return 'api-json'
        }
        // HTTP/HTTPS服务识别
        if (bannerLower.includes('http/') || bannerLower.includes('server:') || bannerLower.includes('content-type:')) {
          return row.protocol === 'https' ? 'https' : 'http'
        }
      }
      
      // 如果protocol是http/https，显示protocol
      if (row.protocol === 'http' || row.protocol === 'https') {
        return row.protocol
      }
      
      // 其他情况显示service或protocol，都没有则显示-
      return row.service || row.protocol || '-'
    },
    getStatusCode(row) {
      // 优先使用statusCode字段
      if (row.statusCode !== undefined && row.statusCode !== null) {
        return row.statusCode.toString()
      }
      // 从响应包中提取HTTP状态码
      if (row.responsePacket) {
        const match = row.responsePacket.match(/HTTP\/[\d.]+\s+(\d+)/i)
        if (match && match[1]) {
          return match[1]
        }
      }
      // 非HTTP服务返回0
      if (row.protocol !== 'http' && row.protocol !== 'https') {
        return '0'
      }
      return '-'
    },
    getCityFromLocation(location) {
      if (!location) return '-'
      // 地区格式通常是：国家, 省份, 城市, 其他信息
      // 例如：中国, 广东, 广州市, Hangzhou Alibaba Advertising Co., Ltd.
      const parts = location.split(',').map(p => p.trim())
      if (parts.length >= 3) {
        // 返回城市部分（去掉"市"、"县"等后缀）
        let city = parts[2]
        // 去掉常见的后缀
        city = city.replace(/市$|县$|区$|州$/, '')
        return city || parts[2]
      }
      // 如果格式不符合预期，返回最后一部分或原字符串
      return parts[parts.length - 1] || location
    },
    cleanBanner(banner, protocol) {
      if (!banner) return ''
      let cleaned = banner
      
      // 如果是HTTP/HTTPS协议，移除HTML内容，只保留响应头信息
      if (protocol === 'http' || protocol === 'https') {
        // 移除HTTP响应行（HTTP/1.1 200 OK 或 HTTP/1.1 200 200 OK）
        cleaned = cleaned.replace(/^HTTP\/[\d.]+\s+\d+\s+(\d+\s+)?[^\r\n]*[\r\n]*/i, '')
        // 移除常见的HTTP响应头字段
        cleaned = cleaned.replace(/^(Content-Type|Content-Length|Server|Date|Connection|Cache-Control|Expires|Last-Modified|ETag|Accept-Ranges|Vary|X-Powered-By|Set-Cookie|Location|WWW-Authenticate|Proxy-Authenticate):[^\r\n]*[\r\n]*/gmi, '')
        // 移除多个连续的空行
        cleaned = cleaned.replace(/\r\n\r\n+/g, '\r\n')
        // 移除开头的空行和空白字符
        cleaned = cleaned.replace(/^[\r\n\s]+/, '')
        
        // 检测并移除HTML内容（从第一个<开始到结尾的所有内容）
        const htmlStartIndex = cleaned.indexOf('<')
        if (htmlStartIndex !== -1) {
          cleaned = cleaned.substring(0, htmlStartIndex)
        }
        
        // 移除末尾的空行和空白字符
        cleaned = cleaned.replace(/[\r\n\s]+$/, '')
      } else {
        // 对于非HTTP协议（TCP获取的信息），只移除HTTP响应头（如果有）
        // 移除HTTP响应行（HTTP/1.1 200 OK 或 HTTP/1.1 200 200 OK）
        cleaned = cleaned.replace(/^HTTP\/[\d.]+\s+\d+\s+(\d+\s+)?[^\r\n]*[\r\n]*/i, '')
        // 移除常见的HTTP响应头字段
        cleaned = cleaned.replace(/^(Content-Type|Content-Length|Server|Date|Connection|Cache-Control|Expires|Last-Modified|ETag|Accept-Ranges|Vary|X-Powered-By|Set-Cookie|Location|WWW-Authenticate|Proxy-Authenticate):[^\r\n]*[\r\n]*/gmi, '')
        // 移除多个连续的空行
        cleaned = cleaned.replace(/\r\n\r\n+/g, '\r\n')
        // 移除开头的空行和空白字符
        cleaned = cleaned.replace(/^[\r\n\s]+/, '')
      }
      
      return cleaned.trim()
    },
    handlePortPresetChange(tab) {
      if (!tab || !tab.config) return
      this.updatePortsFromPreset(tab)
    },
    updatePortsFromPreset(tab) {
      if (!tab || !tab.config) return
      const preset = tab.config.portPreset
      
      switch (preset) {
        case 'all':
          tab.config.ports = '1-65535'
          break
        case 'top100':
          const top100Ports = this.getPresetPortsList('top100')
          tab.config.ports = top100Ports.join(',')
          break
        case 'top1000':
          const top1000Ports = this.getPresetPortsList('top1000')
          tab.config.ports = top1000Ports.join(',')
          break
        case 'custom':
          // 自定义端口时，不自动修改端口范围，保持用户输入
          break
        default:
          break
      }
    },
    parseBannerTags(banner, service) {
      if (!banner && !service) {
        return []
      }
      
      const tags = []
      const text = (banner || service || '').toLowerCase()
      
      // 常见服务和技术栈关键词
      const keywords = [
        'nginx', 'apache', 'iis', 'tomcat', 'jetty', 'undertow',
        'springboot', 'spring-boot', 'spring framework', 'spring',
        'vmware', 'vsphere', 'esxi',
        'mysql', 'mariadb', 'postgresql', 'mongodb', 'redis', 'elasticsearch',
        'ssh', 'openssh', 'dropbear',
        'smb', 'samba', 'cifs',
        'ftp', 'vsftpd', 'proftpd',
        'docker', 'kubernetes', 'k8s',
        'jenkins', 'gitlab', 'jira', 'confluence',
        'wordpress', 'drupal', 'joomla',
        'php', 'python', 'nodejs', 'ruby',
        'java', 'dotnet', '.net',
        'golang', 'go',
        'react', 'vue', 'angular',
        'express', 'django', 'flask', 'laravel',
        'oracle', 'sql server', 'mssql',
        'rabbitmq', 'kafka', 'activemq',
        'zookeeper', 'consul', 'etcd',
        'prometheus', 'grafana', 'kibana',
        'hadoop', 'spark',
        'cassandra', 'couchdb',
        'memcached',
        'solr', 'lucene',
        'git', 'svn',
        'ldap', 'active directory',
        'kerberos', 'ntlm',
        'tls', 'ssl', 'openssl',
        'http', 'https', 'http/2', 'http2',
        'websocket', 'ws', 'wss',
        'grpc', 'thrift',
        'rest', 'soap', 'graphql',
        'oauth', 'jwt', 'saml',
        'cloudflare', 'aws', 'azure', 'gcp',
        'akamai', 'fastly',
        'waf', 'firewall',
        'vpn', 'ipsec',
        'radius', 'tacacs',
        'snmp', 'syslog', 'ntp',
        'dhcp', 'dns', 'bind',
        'smtp', 'pop3', 'imap',
        'exchange', 'outlook',
        'sharepoint', 'teams',
        'citrix', 'vmware horizon',
        'rdp', 'vnc', 'teamviewer',
        'splunk', 'solarwinds',
        'nagios', 'zabbix', 'prtg',
        'ansible', 'puppet', 'chef',
        'terraform',
        'istio', 'linkerd',
        'envoy', 'traefik',
        'kong', 'tyk',
        'helm',
        'argo',
        'spinnaker',
        'github actions', 'gitlab ci',
        'circleci', 'travis ci',
        'azure devops',
        'bamboo', 'teamcity',
        'maven', 'gradle',
        'npm', 'yarn',
        'pip', 'conda',
        'composer',
        'cargo',
        'nuget',
        'webpack', 'rollup',
        'babel', 'typescript',
        'eslint',
        'jest', 'mocha',
        'pytest',
        'rspec',
        'junit',
        'selenium', 'cypress',
        'playwright', 'puppeteer',
        'postman',
        'swagger', 'openapi',
        'graphql', 'apollo',
        'redux', 'mobx',
        'vuex',
        'rxjs',
        'lodash',
        'moment', 'date-fns',
        'axios',
        'socket.io',
        'express', 'koa',
        'django', 'flask', 'fastapi',
        'rails',
        'spring', 'struts',
        'hibernate',
        'sequelize', 'typeorm',
        'prisma',
        'sqlalchemy',
        'mongodb driver', 'mongoose',
        'redis client',
        'elasticsearch client',
        'kafka client',
        'rabbitmq client',
        'grpc client',
        'protobuf',
        'tensorflow', 'pytorch',
        'keras',
        'xgboost',
        'mlflow',
        'tensorboard',
        'kubeflow',
        'sagemaker',
        'snowflake', 'bigquery',
        'redshift',
        'databricks',
        'hive',
        'presto',
        'druid',
        'clickhouse',
        'influxdb', 'prometheus',
        'grafana', 'kibana',
        'elasticsearch', 'opensearch',
        'splunk', 'datadog',
        'new relic', 'dynatrace',
        'appdynamics',
        'jaeger', 'zipkin',
        'opentelemetry',
        'sentry', 'rollbar',
        'bugsnag',
        'loggly',
        'sumologic',
        'fluentd',
        'logstash', 'filebeat',
        'apm',
        'segment',
        'amplitude', 'mixpanel',
        'heap',
        'fullstory',
        'hotjar',
        'microsoft clarity'
      ]
      
      // 检查每个关键词
      for (const keyword of keywords) {
        if (text.includes(keyword)) {
          // 格式化标签名称（首字母大写，处理连字符等）
          const tagName = keyword.split(/[-_\s]+/)
            .map(word => word.charAt(0).toUpperCase() + word.slice(1))
            .join(' ')
            .replace(/\s+/g, ' ')
          if (!tags.includes(tagName)) {
            tags.push(tagName)
          }
        }
      }
      
      // 如果没有匹配到任何标签，尝试从service字段提取
      if (tags.length === 0 && service) {
        const serviceName = service.split(/[\s\/\-_]/)[0]
        if (serviceName && serviceName.length > 1) {
          tags.push(serviceName.charAt(0).toUpperCase() + serviceName.slice(1))
        }
      }
      
      return tags
    }
  }
}
</script>

<style lang="scss" scoped>
.hostscan-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: transparent;
  padding: 6px;
  overflow: hidden;
  box-sizing: border-box;
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

.target-input-section {
  width: 260px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.target-label {
  font-weight: 600;
  color: #303133;
  font-size: 13px;
}

.target-actions {
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
}

.alive-stats {
  margin-top: 4px;
  padding: 4px 6px;
  background: #e6f7ff;
  border-radius: 4px;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 6px;

  .alive-label {
    color: #606266;
  }

  .alive-value {
    color: #409eff;
    font-weight: 600;
  }
}

.port-config-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.config-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.config-section-right {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
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

.port-input-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}
}

.switch-row {
  align-items: center;
}

.config-label {
  font-size: 12px;
  color: #606266;
  font-weight: 500;
}

.progress-info {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  background: #f0f2f5;
  border-radius: 4px;
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

.progress-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.progress-title {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}

.progress-bar-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.progress-bar-row .el-progress {
  flex: 1;
}

.progress-percent {
  flex-shrink: 0;
  min-width: 44px;
  text-align: right;
  font-size: 13px;
  font-weight: 600;
  color: #409eff;
  font-variant-numeric: tabular-nums;
}

.progress-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px 16px;
  margin-top: 12px;
}

.pg-item {
  display: flex;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
}

.pg-label {
  flex-shrink: 0;
  font-size: 12px;
  color: #909399;
}

.pg-value {
  font-size: 12px;
  color: #303133;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.progress-target {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 10px;
  padding-top: 8px;
  border-top: 1px dashed #e4e7ed;
  font-size: 12px;
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  padding: 12px;
  overflow: hidden;
}

.results-table {
  width: 100%;
  height: 100%;

  :deep(.el-table__header-wrapper) {
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
  }
}

.status-tag {
  min-width: 80px;
  font-weight: 600;
  font-size: 13px;
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

.port-preset-options {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  padding: 4px 0;
}

.port-preset-options :deep(.el-radio) {
  margin-right: 0;
  margin-bottom: 0;
}

@media (max-width: 1200px) {
  .config-layout {
    flex-direction: column;
  }
  .target-input-section {
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

.banner-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.banner-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.banner-tag {
  margin: 0;
}


.empty-banner {
  color: #c0c4cc;
  font-style: italic;
}

:deep(.banner-tooltip) {
  max-width: 600px;
  word-break: break-all;
  white-space: pre-wrap;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.6;
}

.url-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.url-text {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.clickable-url {
  color: #409eff;
  cursor: pointer;
  text-decoration: underline;
  transition: color 0.2s;
}

.clickable-url:hover {
  color: #66b1ff;
}

.alive-config-dialog {
  .form-tip {
    font-size: 12px;
    color: #909399;
    margin-top: 4px;
  }
  
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 20px;
  }
}

// 顶部工具栏
.top-toolbar {
  flex-shrink: 0;
  margin-bottom: 16px;
  padding: 16px 20px;
  background: #fff !important;
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
.advanced-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
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

// 任务管理对话框样式
:deep(.history-dialog) {
  --el-dialog-width: 75%;
  --el-dialog-margin-top: 8vh;
  max-width: 90vw;
  
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

.mr-5px {
  margin-right: 5px;
}

.mb-10px {
  margin-bottom: 10px;
}
</style>
