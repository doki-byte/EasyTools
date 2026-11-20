<template>
  <div class="domain-collection-container">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar">
      <div class="toolbar-left">
        <div class="module-title">
          <el-icon class="title-icon">
            <Link />
          </el-icon>
          <span class="title-text">域名收集</span>
        </div>
      </div>
      <div class="toolbar-right">
        <el-button :icon="Setting" type="info" plain size="default" @click="showApiConfigDialog = true">
          配置API
        </el-button>
        <el-button :icon="Plus" type="primary" size="default" @click="createNewTab">
          新建任务
        </el-button>
        <el-button :icon="Clock" type="info" plain size="default" @click="historyDialog = true">
          任务管理
        </el-button>
      </div>
    </div>

    <!-- 主要内容区域 -->
    <div class="main-content">
      <CustomTabs>
        <!-- 空状态 -->
        <div v-if="collectionTabs.length === 0" class="empty-state">
          <el-empty description="暂无收集任务">
            <el-button type="primary" :icon="Plus" @click="createNewTab">
              新建任务
            </el-button>
          </el-empty>
        </div>

        <!-- 标签页 -->
        <el-tabs v-else v-model="activeTabId" type="border-card" class="main-tabs" closable @tab-remove="handleTabRemove"
          @contextmenu.prevent="handleTabsContextMenu">
          <el-tab-pane v-for="tab in collectionTabs" :key="tab.id" :name="tab.id">
            <template #label>
              <span class="tab-label">
                <span class="tab-name">{{ tab.name }}{{ tab.rootDomain ? ` - ${tab.rootDomain}` : '' }}</span>
                <el-badge
                  v-if="tab.results && (tab.results.validDomains || tab.results.valid_domains || tab.results.ValidDomains) && (tab.results.validDomains?.length || tab.results.valid_domains?.length || tab.results.ValidDomains?.length || 0) > 0"
                  :value="tab.results.validDomains?.length || tab.results.valid_domains?.length || tab.results.ValidDomains?.length || 0"
                  :max="9999"
                  class="tab-badge"
                />
              </span>
            </template>

            <!-- 标签页内容 -->
            <div class="tab-content">
              <!-- 输入和配置横向布局 -->
              <div class="top-section">
                <!-- 输入和操作卡片 -->
                <div class="card input-card">

                  <div class="card-body">
                    <div class="input-wrapper">
                      <el-input v-model="tab.rootDomainsText" type="textarea" :rows="5"
                        placeholder="每行一个域名，不要添加http前缀，例如：example.com" class="domain-textarea" resize="none" />
                      <div class="action-bar">
                        <div class="progress-section">
                          <span class="progress-label">进度</span>
                          <el-progress :percentage="tab.currentDomainProgress || 0"
                            :status="tab.currentDomainProgress >= 100 ? 'success' : undefined" :stroke-width="6"
                            :show-text="true" />
                        </div>
                        <div class="action-buttons">
                          <el-button type="primary" :icon="VideoPlay" @click="startCollection(tab)"
                            :loading="tab.loading && tab.status !== 'running'" :disabled="tab.status === 'running'"
                            size="small">
                            开始收集
                          </el-button>
                          <el-button type="warning" :icon="VideoPause" @click="stopCollection(tab)"
                            :disabled="tab.status !== 'running'" size="small">
                            停止
                          </el-button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 基础配置卡片 -->
                <div class="card config-card">

                  <div class="card-body">
                    <div class="config-content">
                      <div class="config-grid">
                        <div class="config-item">
                          <label>线程数</label>
                          <el-input-number v-model="tab.config.threads" :min="1" :max="200" controls-position="right"
                            size="small" style="width: 100%" />
                        </div>
                        <div class="config-item">
                          <label>DNS服务器</label>
                          <el-select v-model="tab.config.dnsServer" size="small" style="width: 100%">
                            <el-option label="阿里DNS-223.5.5.5" value="223.5.5.5" />
                            <el-option label="腾讯DNS-119.29.29.29" value="119.29.29.29" />
                            <el-option label="114DNS-114.114.114.114" value="114.114.114.114" />
                            <el-option label="Google DNS-8.8.8.8" value="8.8.8.8" />
                            <el-option label="Cloudflare DNS-1.1.1.1" value="1.1.1.1" />
                          </el-select>
                        </div>
                        <div class="config-item">
                          <label>探测端口</label>
                          <el-input v-model="tab.config.ports" placeholder="80,443" size="small" />
                        </div>
                        <div class="config-item">
                          <label>过滤IP</label>
                          <el-input v-model="tab.config.filterIP" placeholder="127.0.0.1" size="small" />
                        </div>
                        <div class="config-item full-width">
                          <label>字典文件</label>
                          <el-select v-model="tab.config.selectedDicts" multiple collapse-tags collapse-tags-tooltip
                            placeholder="选择字典文件" size="small" style="width: 100%" @visible-change="loadDictList">
                            <el-option v-for="dict in dicts" :key="dict.name"
                              :label="`${dict.name} (${formatSize(dict.size)})`" :value="dict.name" />
                          </el-select>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 高级配置卡片 -->
                <div class="card advanced-config-card">

                  <div class="card-body">
                    <div class="config-content">
                      <div class="config-item full-width">
                        <label>收集模式</label>
                        <el-radio-group v-model="tab.mode" size="small" @change="handleModeChange(tab)">
                          <el-radio-button label="dict">字典模式</el-radio-button>
                          <el-radio-button label="ai">AI模式</el-radio-button>
                          <el-radio-button label="hybrid">混合模式</el-radio-button>
                        </el-radio-group>
                      </div>
                      <template v-if="tab.mode === 'ai' || tab.mode === 'hybrid'">
                        <div class="config-item full-width">
                          <div class="switch-with-button">
                            <label>使用系统配置</label>
                            <el-switch v-model="tab.config.useSystemConfig" @change="handleSystemConfigChange(tab)" />
                            <el-button type="text" size="small"
                              @click="$router.push({ name: 'systemManage', query: { tab: 'ai-config' } })">
                              管理AI配置
                            </el-button>
                          </div>
                        </div>
                        <div v-if="tab.mode === 'ai' && tab.config.useSystemConfig" class="config-item full-width">
                          <label>选择AI配置</label>
                          <el-select v-model="tab.config.aiConfigId" placeholder="选择系统配置的AI" size="small"
                            style="width: 100%" filterable @visible-change="loadAIConfigs">
                            <el-option v-for="config in aiConfigs" :key="config.id"
                              :label="`${config.name} (${getProviderName(config.provider)})`" :value="config.id"
                              :disabled="!config.enabled" />
                          </el-select>
                        </div>
                        <!-- AI模式：生成域名数量 -->
                        <div v-if="tab.mode === 'ai'" class="config-item full-width">
                          <label>生成域名数量</label>
                          <el-input-number
                            v-model="tab.config.generateCount"
                            :min="10"
                            :max="10000"
                            :step="10"
                            size="small"
                            style="width: 100%"
                            placeholder="请输入要生成的域名数量"
                          />
                          <div class="config-hint">
                            <el-text type="info" size="small">
                              提示：AI将生成指定数量的域名前缀，默认100个
                            </el-text>
                          </div>
                        </div>
                        <!-- AI模式：样本域名输入（当未启用空间测绘时） -->
                        <div v-if="tab.mode === 'ai' && !tab.config.enableSpaceMapping" class="config-item full-width">
                          <label>样本域名（可选）</label>
                          <el-input
                            v-model="tab.config.sampleDomainsText"
                            type="textarea"
                            :rows="3"
                            placeholder="请输入已有的子域名，每行一个，例如：&#10;www.example.com&#10;api.example.com&#10;admin.example.com&#10;&#10;AI将根据这些样本域名推断生成类似的域名"
                            size="small"
                          />
                          <div class="config-hint">
                            <el-text type="info" size="small">
                              提示：如果未启用空间测绘，请在此输入已有的子域名，AI将根据这些样本推断生成类似的域名
                            </el-text>
                          </div>
                        </div>
                      </template>
                      <div class="config-switches">
                        <div class="switch-item">
                          <label>指纹信息</label>
                          <el-switch v-model="tab.config.fingerprint" />
                        </div>
                        <div class="switch-item">
                          <label>泛解析</label>
                          <el-switch v-model="tab.config.wildcard" />
                        </div>
                        <!-- <div class="switch-item">
                          <label>POC检测</label>
                          <el-switch v-model="tab.config.pocDetection" />
                        </div>
                        <div class="switch-item">
                          <label>接口探测</label>
                          <el-switch v-model="tab.config.apiDetection" />
                        </div> -->
                        <div class="switch-item">
                          <label>空间测绘</label>
                          <el-switch v-model="tab.config.enableSpaceMapping" @change="handleSpaceMappingChange(tab)" />
                        </div>
                      </div>
                      <div v-if="tab.config.enableSpaceMapping" class="config-item full-width">
                        <label>空间测绘平台</label>
                        <el-select v-model="tab.config.spaceMappingPlatforms" multiple collapse-tags
                          collapse-tags-tooltip placeholder="选择空间测绘平台" size="small" style="width: 100%">
                          <el-option label="0.zone" value="0zone" />
                          <el-option label="FOFA官方" value="fofa" />
                          <el-option label="FOFA第三方" value="otherfofa" />
                          <el-option label="Hunter" value="hunter" />
                          <el-option label="Quake" value="quake" />
                        </el-select>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 结果卡片 -->
              <div class="card results-card">
                <div class="card-header">
                  <div class="card-title-group">
                    <span class="card-title">收集结果</span>
                    <div class="card-badges">
                      <el-tag type="success" size="small">
                        有效: {{ (tab.results?.validDomains || tab.results?.valid_domains || tab.results?.ValidDomains || []).length }}
                      </el-tag>
                      <el-tag type="info" size="small">
                        总计: {{ tab.results?.totalGenerated || tab.results?.total_generated || tab.results?.TotalGenerated || 0 }}
                      </el-tag>
                    </div>
                  </div>
              <div class="card-actions">
                    <el-button
                      size="default"
                      :type="tab.showAllResults ? 'primary' : 'default'"
                      @click="toggleResultFilter(tab)"
                    >
                      {{ tab.showAllResults ? '只显示有效' : '显示全部' }}
                    </el-button>
                    <el-button :icon="Download"
                      :disabled="!tab.results || !(tab.results.validDomains || tab.results.valid_domains || tab.results.ValidDomains) || (tab.results.validDomains?.length || tab.results.valid_domains?.length || tab.results.ValidDomains?.length || 0) === 0"
                      @click="exportToExcel(tab)" size="default" plain>
                      导出Excel
                    </el-button>
                    <el-button :icon="Delete"
                      :disabled="!tab.results || !(tab.results.validDomains || tab.results.valid_domains || tab.results.ValidDomains) || (tab.results.validDomains?.length || tab.results.valid_domains?.length || tab.results.ValidDomains?.length || 0) === 0"
                      @click="clearResults(tab)" size="default" plain>
                      清空
                    </el-button>
                  </div>
                </div>
                <div class="card-body">
                  <el-table 
                  :data="getPagedResults(tab)" 
                  :height="tableHeight" 
                  stripe 
                  border 
                  size="default"
                  class="results-table" 
                  :empty-text="'暂无数据'"
                  :key="`table-${tab.id}-${(tab.results?.validDomains || tab.results?.valid_domains || tab.results?.ValidDomains || []).length}-${tableHeight}`"
                  style="width: 100%"
                  @row-contextmenu="handleRowContextMenu"
                  @selection-change="handleSelectionChange"
                  >
                    <el-table-column type="selection" width="55" align="center" />
                    <el-table-column type="index" width="50" align="center" label="#" />
                    <el-table-column prop="domain" label="域名" min-width="200" sortable>
                      <template #default="{ row }">
                        <el-tooltip :content="row.domain" placement="top" :disabled="!row.domain || row.domain.length <= 30" effect="dark">
                          <span class="domain-text clickable" @click.stop="handleDomainClick(row.domain)" :title="`点击打开: ${row.domain}`">
                            {{ row.domain }}
                          </span>
                        </el-tooltip>
                      </template>
                    </el-table-column>
                    <el-table-column prop="source" label="来源" min-width="150" align="center">
                      <template #default="{ row }">
                        <div class="source-tags">
                          <template v-for="(sourceItem, index) in getDomainSources(row)" :key="index">
                            <el-tag 
                              size="small" 
                              :type="getDomainSourceTagType(sourceItem.source) || undefined"
                              style="margin-right: 4px; margin-bottom: 2px;"
                            >
                              {{ sourceItem.text }}
                            </el-tag>
                          </template>
                        </div>
                      </template>
                    </el-table-column>
                    <el-table-column prop="ips" label="IP地址" width="150">
                      <template #default="{ row }">
                        <div v-if="row.ips && row.ips.length > 0" class="ips-cell">
                          <el-tooltip v-if="row.ips.length > 2" :content="row.ips.join(', ')" placement="top" effect="dark">
                            <div>
                              <el-tag v-for="(ip, idx) in row.ips.slice(0, 2)" :key="idx" size="small"
                                style="margin-right: 4px">
                                {{ ip }}
                              </el-tag>
                              <el-tag size="small" type="info">
                                +{{ row.ips.length - 2 }}
                              </el-tag>
                            </div>
                          </el-tooltip>
                          <div v-else>
                            <el-tag v-for="(ip, idx) in row.ips" :key="idx" size="small"
                              style="margin-right: 4px">
                              {{ ip }}
                            </el-tag>
                          </div>
                        </div>
                        <span v-else class="empty-text">-</span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="status" label="状态" width="100" align="center">
                      <template #default="{ row }">
                        <el-tag :type="row.status === 'valid' ? 'success' : 'danger'" size="default" effect="dark">
                          {{ row.status === 'valid' ? '有效' : '无效' }}
                        </el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="httpStatus" label="HTTP状态" width="100" align="center">
                      <template #default="{ row }">
                        <el-tag v-if="row.httpStatus > 0" :type="getHTTPStatusTagType(row.httpStatus)" size="small">
                          {{ row.httpStatus }}
                        </el-tag>
                        <span v-else class="empty-text">-</span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="httpTitle" label="HTTP标题" width="200">
                      <template #default="{ row }">
                        <el-tooltip :content="row.httpTitle || ''" placement="top"
                          :disabled="!row.httpTitle || row.httpTitle.length <= 20" effect="dark">
                          <div class="table-cell-text">
                            {{ row.httpTitle || '-' }}
                          </div>
                        </el-tooltip>
                      </template>
                    </el-table-column>
                  </el-table>
                  <div class="pagination-wrapper">
                    <el-pagination v-model:current-page="tab.currentPage" v-model:page-size="tab.pageSize"
                      :page-sizes="[10, 20, 50, 100, 200]"
                      :total="getTotalResults(tab)"
                      layout="total, sizes, prev, pager, next, jumper" background
                      @size-change="(size) => handleSizeChange(size, tab)"
                      @current-change="(page) => handleCurrentChange(page, tab)" />
                  </div>
                </div>
              </div>

            </div>
          </el-tab-pane>
        </el-tabs>
      </CustomTabs>
    </div>

    <!-- API配置对话框 -->
    <el-dialog
      v-model="showApiConfigDialog"
      title="API Key配置 (与空间测绘接口共用所有key)"
      width="800px"
      :close-on-click-modal="false"
      @open="loadApiConfig"
    >
      <div class="api-config-content">
        <div 
          v-for="(config, key) in apiConfigs" 
          :key="key"
          class="api-config-item"
        >
          <div class="api-config-header">
            <span class="api-name">{{ getPlatformLabel(key) }}</span>
            <el-checkbox v-model="config.enabled" label="启用" />
          </div>
          <div class="api-config-body">
            <!-- FOFA官方需要邮箱和key，放在同一行 -->
            <template v-if="key === 'fofa'">
              <div class="fofa-input-row">
                <el-input
                  v-model="config.email"
                  type="text"
                  placeholder="请输入FOFA官方邮箱"
                  class="api-key-input fofa-email-input"
                />
                <el-input
                  v-model="config.key"
                  :type="config.showKey ? 'text' : 'password'"
                  placeholder="请输入FOFA官方 API Key"
                  class="api-key-input fofa-key-input"
                >
                  <template #suffix>
                    <el-icon 
                      class="eye-icon"
                      @click="config.showKey = !config.showKey"
                    >
                      <View v-if="!config.showKey" />
                      <Hide v-else />
                    </el-icon>
                  </template>
                </el-input>
              </div>
            </template>
            <!-- 其他平台只需要key，FOFA第三方额外支持 BaseURL -->
            <template v-else>
              <!-- FOFA第三方 BaseURL + Key 一行显示 -->
              <template v-if="key === 'otherfofa'">
                <div class="fofa-input-row">
                  <el-input
                    v-model="apiSettings.fofaThirdBaseURL"
                    type="text"
                    placeholder="FOFA第三方 BaseURL，例如：https://fofa.red"
                    class="api-key-input fofa-baseurl-input"
                  />
                  <el-input
                    v-model="config.key"
                    :type="config.showKey ? 'text' : 'password'"
                    :placeholder="`请输入${getPlatformLabel(key)} API Key`"
                    class="api-key-input"
                  >
                    <template #suffix>
                      <el-icon 
                        class="eye-icon"
                        @click="config.showKey = !config.showKey"
                      >
                        <View v-if="!config.showKey" />
                        <Hide v-else />
                      </el-icon>
                    </template>
                  </el-input>
                </div>
              </template>
              <template v-else>
                <el-input
                  v-model="config.key"
                  :type="config.showKey ? 'text' : 'password'"
                  :placeholder="`请输入${getPlatformLabel(key)} API Key`"
                  class="api-key-input"
                >
                  <template #suffix>
                    <el-icon 
                      class="eye-icon"
                      @click="config.showKey = !config.showKey"
                    >
                      <View v-if="!config.showKey" />
                      <Hide v-else />
                    </el-icon>
                  </template>
                </el-input>
              </template>
            </template>
            <el-tag 
              :type="config.status === 'valid' ? 'success' : config.status === 'invalid' ? 'danger' : 'info'"
              size="small"
              class="api-status"
            >
              {{ config.status === 'valid' ? '有效' : config.status === 'invalid' ? '无效' : '未知' }}
            </el-tag>
            <el-button 
              type="primary" 
              size="small" 
              link
              @click="openApiApplyPage(key)"
            >
              API申请
            </el-button>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showApiConfigDialog = false">取消</el-button>
          <el-button type="primary" @click="handleSaveApiConfig">保存</el-button>
          <el-button type="warning" @click="handleVerifyApiKeys">Key可用验证</el-button>
        </div>
      </template>
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
          v-loading="historyLoading"
        >
          <el-table-column type="index" width="60" align="center" label="#" />
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
          <el-table-column prop="rootDomain" label="根域名" min-width="150" />
          <el-table-column prop="mode" label="模式" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.mode === 'dict' ? 'primary' : row.mode === 'ai' ? 'success' : 'warning'" size="small">
                {{ row.mode === 'dict' ? '字典' : row.mode === 'ai' ? 'AI' : row.mode === 'hybrid' ? '混合' : row.mode }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="getStatusTagType(row.status)" size="small" effect="dark">
                {{ getStatusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="结果" width="200" align="center">
            <template #default="{ row }">
              <div v-if="row.results" style="display: flex; flex-direction: column; gap: 4px; align-items: center;">
                <span>
                  有效: {{ (row.results.validDomains || row.results.valid_domains || row.results.ValidDomains || []).length }} /
                  总计: {{ row.results.totalGenerated || row.results.total_generated || row.results.TotalGenerated || 0 }}
                </span>
              </div>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column prop="startTime" label="开始时间" width="180" align="center">
            <template #default="{ row }">
              {{ formatTime(row.startTime) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200" align="center" fixed="right">
            <template #default="{ row }">
              <el-button-group>
                <el-tooltip content="加载任务">
                  <el-button :icon="View" link @click="loadTaskToTab(row)" />
                </el-tooltip>
                <el-tooltip content="保存日志">
                  <el-button :icon="Download" link @click="saveTaskLog(row.id)" />
                </el-tooltip>
                <el-tooltip content="删除">
                  <el-button :icon="Delete" link @click="deleteTask(row.id)" />
                </el-tooltip>
              </el-button-group>
            </template>
          </el-table-column>
        </el-table>
        <div class="history-pagination">
          <el-pagination
            v-model:current-page="taskPagination.currentPage"
            v-model:page-size="taskPagination.pageSize"
            :total="historyList ? historyList.length : 0"
            :page-sizes="[5,10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next"
            background
          />
        </div>
      </div>
    </el-dialog>

    <!-- POC检测对话框 -->
    <el-dialog
      v-model="pocScanDialogVisible"
      title="创建POC检测任务"
      width="600px"
      destroy-on-close
    >
      <el-form label-width="120px">
        <el-form-item label="任务名称:">
          <el-input v-model="pocScanTaskName" placeholder="可选，默认自动生成" />
        </el-form-item>
        <el-form-item label="目标数量:">
          <span>{{ pocScanTargets.length }} 个URL</span>
        </el-form-item>
        <el-form-item label="扫描模式:">
          <el-radio-group v-model="pocScanMode">
            <el-radio label="full">全量漏洞扫描</el-radio>
            <el-radio label="fingerprint">指纹漏洞扫描</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="指定指纹:" v-if="pocScanMode === 'fingerprint'">
          <el-select
            v-model="pocScanCustomTags"
            multiple
            filterable
            placeholder="请选择指纹（可选）"
            style="width: 100%"
          >
            <!-- 这里可以添加指纹选项，暂时留空，用户可以在漏洞扫描模块中选择 -->
          </el-select>
          <div style="color: #909399; font-size: 12px; margin-top: 5px;">
            提示：可在漏洞扫描模块中选择具体指纹
          </div>
        </el-form-item>
        <el-form-item label="高级配置:">
          <el-checkbox v-model="pocScanSkipNucleiWithoutTags">无指纹跳过</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pocScanDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmPOCScan">创建任务</el-button>
      </template>
    </el-dialog>

    <!-- 标签页右键菜单 -->
    <Teleport to="body">
      <ul 
        class="tabs-context-menu" 
        v-if="tabsContextMenuVisible" 
        :style="tabsContextMenuStyle"
      >
        <li @click="handleCloseCurrentTab">关闭当前</li>
        <li @click="handleCloseOtherTabs">关闭其他</li>
        <li @click="handleCloseAllTabs">全部关闭</li>
      </ul>
    </Teleport>
    <!-- 表格行右键菜单 -->
    <Teleport to="body">
      <div
        v-if="rowContextMenuVisible"
        class="context-menu"
        :style="rowContextMenuStyle"
        @click.stop
      >
        <div
          class="context-menu-item"
          @click="handleRunFingerprintSingle"
        >
          <el-icon><View /></el-icon>
          <span>指纹识别（当前）</span>
        </div>
        <div
          class="context-menu-item"
          @click="handleRunFingerprintMulti"
        >
          <el-icon><View /></el-icon>
          <span>批量指纹识别 ({{ selectedRows.length }})</span>
        </div>
        <div class="context-menu-divider"></div>
        <div
          class="context-menu-item"
          @click="handleRunAPIDetectionSingle"
        >
          <el-icon><Tools /></el-icon>
          <span>接口探测（当前）</span>
        </div>
        <div
          class="context-menu-item"
          @click="handleRunAPIDetectionMulti"
        >
          <el-icon><Tools /></el-icon>
          <span>批量接口探测 ({{ selectedRows.length }})</span>
        </div>
        <div class="context-menu-divider"></div>
        <div
          class="context-menu-item"
          @click="handleRunPOCSingle"
        >
          <el-icon><Document /></el-icon>
          <span>POC检测（当前）</span>
        </div>
        <div
          class="context-menu-item"
          @click="handleRunPOCMulti"
        >
          <el-icon><Document /></el-icon>
          <span>批量POC检测 ({{ selectedRows.length }})</span>
        </div>
        <div class="context-menu-divider"></div>
        <div
          v-if="isWebServiceFromDomainRow(selectedRow)"
          class="context-menu-item"
          @click="handleRunDirsearchSingle"
        >
          <el-icon><FolderOpened /></el-icon>
          <span>目录扫描（当前）</span>
        </div>
        <div
          v-if="selectedRows.some(row => isWebServiceFromDomainRow(row))"
          class="context-menu-item"
          @click="handleRunDirsearchMulti"
        >
          <el-icon><FolderOpened /></el-icon>
          <span>批量目录扫描 ({{ selectedRows.filter(row => isWebServiceFromDomainRow(row)).length }})</span>
        </div>
        <div class="context-menu-divider"></div>
        <div class="context-menu-item" @click="handleCopyRow">
          <el-icon><CopyDocument /></el-icon>
          <span>复制当前行</span>
        </div>
        <div class="context-menu-item" @click="handleCopySelected">
          <el-icon><CopyDocument /></el-icon>
          <span>复制选中行 ({{ selectedRows.length }})</span>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script>
import {
  Link,
  VideoPlay,
  VideoPause,
  Download,
  Delete,
  CopyDocument,
  Plus,
  Clock,
  CircleCheck,
  CircleClose,
  Setting,
  Tools,
  View,
  Hide,
  Document,
  Refresh,
  InfoFilled,
  FolderOpened
} from '@element-plus/icons-vue'
import { markRaw } from 'vue'
import { EventsOn, EventsOff, BrowserOpenURL } from '../../../wailsjs/runtime'
import {
  CreateCollectionTask,
  StartCollection,
  StopCollection,
  GetTask,
  GetAllTasks,
  DeleteTask,
  GetDictList,
  ExportResults,
  ClearAllTasks,
  SaveTaskLog
} from '../../../wailsjs/go/domainCollection/DomainCollectionController'
import { GetAllConfigs } from '../../../wailsjs/go/system/AIConfigManager'
import { GetPortConfig } from '../../../wailsjs/go/controller/Server'
import { StartPocScan } from '../../../wailsjs/go/webscan/PocController'
import { GetPocConfig } from '../../../wailsjs/go/webscan/PocConfigController'
import { GetGlobalProxy } from '../../../wailsjs/go/proxy/ProxyManager'
import { CreateTask } from '../../../wailsjs/go/browserScan/BrowserScan'
import { StartFingerprintScan } from '../../../wailsjs/go/finger/FingerprintController'
import { StartDirectoryScan } from '../../../wailsjs/go/dirsearch/DirsearchController'
import CustomTabs from '../../components/CustomTabs.vue'
import axios from 'axios'
export default {
  name: 'DomainCollection',
  components: {
    Link,
    VideoPlay,
    VideoPause,
    Download,
    Delete,
    CopyDocument,
    Plus,
    Clock,
    CircleCheck,
    CircleClose,
    Setting,
      Tools,
      View,
      Hide,
      FolderOpened,
      CustomTabs
    },
  data() {
    return {
      Link: markRaw(Link),
      VideoPlay: markRaw(VideoPlay),
      VideoPause: markRaw(VideoPause),
      Download: markRaw(Download),
      Delete: markRaw(Delete),
      Plus: markRaw(Plus),
      Clock: markRaw(Clock),
      CircleCheck: markRaw(CircleCheck),
      CircleClose: markRaw(CircleClose),
      Setting: markRaw(Setting),
      Tools: markRaw(Tools),
      View: markRaw(View),
      Hide: markRaw(Hide),
      Document: markRaw(Document),
      Refresh: markRaw(Refresh),
      InfoFilled: markRaw(InfoFilled),
      FolderOpened: markRaw(FolderOpened),
      collectionTabs: [],
      activeTabId: '',
      tabIdCounter: 0,
      STORAGE_KEY: 'easytools:domain-collection:data', // 数据持久化存储键
      saveDataTimer: null, // 防抖定时器
      configExpandedDefault: [],
      historyDialog: false,
      historyList: [],
      historyLoading: false,
      taskPagination: {
        currentPage: 1,
        pageSize: 5
      },
      dicts: [],
      aiConfigs: [],
      tableHeight: 400,
      showApiConfigDialog: false,
      apiConfigs: {
        '0zone': { key: '', enabled: false, showKey: false, status: 'unknown' },
        fofa: { email: '', key: '', enabled: false, showKey: false, status: 'unknown' },
        otherfofa: { key: '', enabled: false, showKey: false, status: 'unknown' },
        hunter: { key: '', enabled: false, showKey: false, status: 'unknown' },
        quake: { key: '', enabled: false, showKey: false, status: 'unknown' }
      },
      apiSettings: {
        maxPages: 5,
        delay: 1,
        timeout: 30,
        enableProxy: false,
        pageSize: 100,
        // FOFA 第三方接口（默认使用 FOFA.RED）
        fofaThirdBaseURL: 'https://fofa.red'
      },
      apiClient: null,
      portConfig: null,
      // 右键菜单相关
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: '',
      // POC检测对话框相关
      pocScanDialogVisible: false,
      pocScanTaskName: '',
      pocScanTargets: [],
      pocScanMode: 'full',
      pocScanCustomTags: [],
      pocScanSkipNucleiWithoutTags: false,
      // 浏览器扫描对话框相关
      browserScanDialogVisible: false,
      browserScanTaskName: '',
      browserScanTargets: [],
      // 表格行右键菜单相关
      rowContextMenuVisible: false,
      rowContextMenuStyle: { left: '0px', top: '0px' },
      selectedRow: null,
      selectedRows: []
    }
  },
  computed: {
    tabsContextMenuStyle() {
      const { x, y } = this.tabsContextMenuPosition
      return {
        left: `${x}px`,
        top: `${y}px`
      }
    },
    currentTab() {
      return this.collectionTabs.find(tab => tab.id === this.activeTabId) || null
    },
    paginatedTasks() {
      const start = (this.taskPagination.currentPage - 1) * this.taskPagination.pageSize
      const end = start + this.taskPagination.pageSize
      return this.historyList.slice(start, end)
    }
  },
  mounted() {
    this.restoreData() // 恢复保存的数据
    this.setupEventListeners()
    this.loadDictList()
    this.updateTableHeight()
    this.initApiClient()
    this.loadApiConfig()
    window.addEventListener('resize', this.updateTableHeight)
  },
  activated() {
    // keep-alive 激活时恢复数据
    this.restoreData()
  },
  deactivated() {
    // keep-alive 停用时立即保存数据
    if (this.saveDataTimer) {
      clearTimeout(this.saveDataTimer)
      this.saveDataTimer = null
    }
    this.saveData(true)
  },
  beforeUnmount() {
    // 清除防抖定时器，立即保存数据
    if (this.saveDataTimer) {
      clearTimeout(this.saveDataTimer)
      this.saveDataTimer = null
    }
    this.saveData(true) // 组件销毁前立即保存数据
    this.removeEventListeners()
    window.removeEventListener('resize', this.updateTableHeight)
  },
  methods: {
    // 保存数据到 localStorage（带防抖）
    saveData(immediate = false) {
      // 清除之前的定时器
      if (this.saveDataTimer) {
        clearTimeout(this.saveDataTimer)
        this.saveDataTimer = null
      }
      
      const doSave = () => {
        try {
          // 保存所有数据，包括运行状态
          // 注意：对于正在运行的任务，恢复时会从后端检查最新状态
          const tabsToSave = this.collectionTabs.map(tab => {
            // 创建一个副本，但清除定时器引用（定时器不能序列化）
            const tabCopy = { ...tab }
            // 清除不能序列化的属性
            delete tabCopy.updateTimer
            return tabCopy
          })
          
          const dataToSave = {
            collectionTabs: tabsToSave,
            activeTabId: this.activeTabId,
            tabIdCounter: this.tabIdCounter,
            timestamp: Date.now()
          }
          localStorage.setItem(this.STORAGE_KEY, JSON.stringify(dataToSave))
        } catch (error) {
          console.error('保存域名收集数据失败:', error)
        }
      }
      
      if (immediate) {
        doSave()
      } else {
        // 防抖：500ms 后保存
        this.saveDataTimer = setTimeout(doSave, 500)
      }
    },
    // 从 localStorage 恢复数据
    async restoreData() {
      try {
        const savedData = localStorage.getItem(this.STORAGE_KEY)
        if (savedData) {
          const data = JSON.parse(savedData)
          // 恢复标签页数据
          if (data.collectionTabs && Array.isArray(data.collectionTabs) && data.collectionTabs.length > 0) {
            this.collectionTabs = data.collectionTabs
            // 每次启动时重新计算计数器，从现有标签页中找出最大值，确保新创建的标签页ID不会冲突
            const maxId = this.collectionTabs.reduce((max, tab) => {
              const match = tab.id.match(/tab_(\d+)/)
              if (match) {
                const idNum = parseInt(match[1], 10)
                return Math.max(max, idNum)
              }
              return max
            }, 0)
            this.tabIdCounter = maxId
            // 恢复激活的标签页
            if (data.activeTabId && this.collectionTabs.some(tab => tab.id === data.activeTabId)) {
              this.activeTabId = data.activeTabId
            } else if (this.collectionTabs.length > 0) {
              this.activeTabId = this.collectionTabs[0].id
            }
            
            // 异步检查有 taskId 的任务状态（如果任务还在运行，需要从后端获取最新状态）
            this.checkTaskStatuses()
          }
        }
      } catch (error) {
        console.error('恢复域名收集数据失败:', error)
      }
    },
    // 检查任务状态（对于有 taskId 的任务）
    async checkTaskStatuses() {
      // 检查所有有 taskId 的标签页
      const tabsWithTaskId = this.collectionTabs.filter(tab => tab.taskId)
      if (tabsWithTaskId.length === 0) {
        return
      }
      
      // 异步检查每个任务的状态
      tabsWithTaskId.forEach(async (tab) => {
        try {
          const taskJson = await GetTask(tab.taskId)
          const task = JSON.parse(taskJson)
          
          // 如果任务状态发生变化，更新标签页状态
          if (task.status && task.status !== tab.status) {
            tab.status = task.status
            
            // 如果任务已完成或失败，更新结果
            if (task.status === 'completed' || task.status === 'failed') {
              if (task.results) {
                // 统一处理字段名
                if (task.results.valid_domains && !task.results.validDomains) {
                  task.results.validDomains = task.results.valid_domains
                }
                if (task.results.invalid_domains && !task.results.invalidDomains) {
                  task.results.invalidDomains = task.results.invalid_domains
                }
                tab.results = task.results
              }
              if (task.logs) {
                tab.logs = task.logs
              }
            }
            
            // 如果任务正在运行，启动更新定时器
            if (task.status === 'running' && !tab.updateTimer) {
              this.startTaskUpdateTimer(tab)
            } else if (task.status !== 'running' && tab.updateTimer) {
              // 如果任务不再运行，清除定时器
              clearInterval(tab.updateTimer)
              tab.updateTimer = null
            }
          }
        } catch (error) {
          // 如果任务不存在或获取失败，可能是任务已被删除，保持当前状态
        }
      })
    },
    async initApiClient() {
      try {
        this.portConfig = await GetPortConfig()
        const port = this.portConfig?.mainPort || 52867
        const baseURL = `http://127.0.0.1:${port}`
        this.apiClient = axios.create({
          baseURL: baseURL,
          timeout: 30000,
        })
      } catch (error) {
        console.error('初始化API客户端失败:', error)
        this.$message.error('初始化API客户端失败')
      }
    },
    async loadApiConfig() {
      try {
        if (!this.apiClient) return
        const response = await this.apiClient.get('/api/info-collection/config')
        if (response.data && response.data.success) {
          const config = response.data.data
          if (config.apiKeys) {
            Object.keys(config.apiKeys).forEach(key => {
              // 将旧的 'fofa.red' 映射为新的 'otherfofa'，保持与空间测绘模块兼容
              const keyMap = {
                'fofa.red': 'otherfofa',
                '0.zone': '0zone',
                zone0: '0zone',
              }
              const localKey = keyMap[key] || key
              if (this.apiConfigs[localKey]) {
                const apiKeyValue = config.apiKeys[key] || ''
                // FOFA官方需要解析email:key格式
                if (localKey === 'fofa' && apiKeyValue.includes(':')) {
                  const parts = apiKeyValue.split(':', 2)
                  this.apiConfigs[localKey].email = parts[0] || ''
                  this.apiConfigs[localKey].key = parts[1] || ''
                } else {
                  this.apiConfigs[localKey].key = apiKeyValue
                }
                this.apiConfigs[localKey].enabled = config.enabled?.[key] || false
              }
            })
          }
          if (config.settings) {
            this.apiSettings = { ...this.apiSettings, ...config.settings }
            // 将 FOFA 第三方 baseURL 映射到本地默认值（用于前端展示）
            if (!this.apiSettings.fofaThirdBaseURL) {
              this.apiSettings.fofaThirdBaseURL = 'https://fofa.red'
            }
          }
        }
      } catch (error) {
        console.error('加载API配置失败:', error)
      }
    },
    async handleSaveApiConfig() {
      try {
        if (!this.apiClient) {
          this.$message.error('API客户端未初始化')
          return
        }

        const apiKeys = {}
        const enabled = {}
        Object.keys(this.apiConfigs).forEach(key => {
          // FOFA官方需要组合email:key格式
          if (key === 'fofa') {
            const email = this.apiConfigs[key].email || ''
            const keyValue = this.apiConfigs[key].key || ''
            if (email && keyValue) {
              apiKeys[key] = `${email}:${keyValue}`
            } else if (keyValue) {
              // 如果只有key，也保存（后端会处理）
              apiKeys[key] = keyValue
            } else {
              apiKeys[key] = ''
            }
          } else {
            apiKeys[key] = this.apiConfigs[key].key || ''
          }
          enabled[key] = this.apiConfigs[key].enabled
        })
        // 确保 FOFA 第三方 baseURL 写入全局设置，保持与空间测绘模块一致
        if (!this.apiSettings.fofaThirdBaseURL || this.apiSettings.fofaThirdBaseURL.trim() === '') {
          this.apiSettings.fofaThirdBaseURL = 'https://fofa.red'
        }

        const response = await this.apiClient.post('/api/info-collection/config', {
          apiKeys,
          enabled,
          settings: this.apiSettings
        })

        if (response.data && response.data.success) {
          this.$message.success('配置保存成功')
          this.showApiConfigDialog = false
        } else {
          this.$message.error(response.data?.message || '保存失败')
        }
      } catch (error) {
        console.error('保存API配置失败:', error)
        this.$message.error('保存配置失败: ' + (error.response?.data?.message || error.message))
      }
    },
    async handleVerifyApiKeys() {
      if (!this.apiClient) {
        this.$message.error('API客户端未初始化')
        return
      }

      const verifyPromises = []
      const verifyResults = {}

      for (const [platform, config] of Object.entries(this.apiConfigs)) {
        // FOFA官方需要email和key，其他平台只需要key
        const hasKey = platform === 'fofa' 
          ? (config.email && config.email.trim() !== '' && config.key && config.key.trim() !== '')
          : (config.key && config.key.trim() !== '')
        
        if (hasKey) {
          verifyResults[platform] = { status: 'verifying' }
          // FOFA官方需要组合email:key格式
          const apiKey = platform === 'fofa' 
            ? `${config.email}:${config.key}`
            : config.key
          verifyPromises.push(
            this.verifySingleKey(platform, apiKey).then(valid => {
              verifyResults[platform] = { status: valid ? 'valid' : 'invalid', valid }
              this.apiConfigs[platform].status = valid ? 'valid' : 'invalid'
            }).catch(err => {
              verifyResults[platform] = { status: 'invalid', error: err.message }
              this.apiConfigs[platform].status = 'invalid'
            })
          )
        }
      }

      if (verifyPromises.length === 0) {
        this.$message.warning('没有配置任何API Key')
        return
      }

      this.$message.info(`正在验证 ${verifyPromises.length} 个API Key...`)

      await Promise.all(verifyPromises)

      const validCount = Object.values(verifyResults).filter(r => r.status === 'valid').length
      const invalidCount = Object.values(verifyResults).filter(r => r.status === 'invalid').length

      this.$message.success(`验证完成：有效 ${validCount} 个，无效 ${invalidCount} 个`)
    },
    async verifySingleKey(platform, apiKey) {
      try {
        const response = await this.apiClient.post('/api/info-collection/space-mapping/verify-key', {
          platform: platform,
          apiKey: apiKey,
          // 携带当前域名收集模块中的空间测绘相关配置（尤其是 FOFA 第三方 BaseURL），
          // 这样用户修改输入框后无需先保存即可使用最新配置进行 Key 验证
          settings: this.apiSettings,
        })

        if (response.data && response.data.success) {
          return response.data.valid === true
        }
        return false
      } catch (error) {
        console.error(`验证 ${platform} Key失败:`, error)
        return false
      }
    },
    getPlatformLabel(platform) {
      const labels = {
        '0zone': '0.zone',
        fofa: 'FOFA官方',
        otherfofa: 'FOFA第三方',
        hunter: 'Hunter',
        quake: 'Quake',
      }
      return labels[platform] || platform
    },
    openApiApplyPage(platform) {
      let url = ''
      if (platform === '0zone') {
        url = 'https://0.zone/apply'
      } else if (platform === 'fofa') {
        url = 'https://fofa.info'
      } else if (platform === 'otherfofa') {
        // 使用配置的 FOFA 第三方 BaseURL，默认为 https://fofa.red/
        url = (this.apiSettings.fofaThirdBaseURL || 'https://fofa.red/').trim()
      } else if (platform === 'hunter') {
        url = 'https://hunter.qianxin.com/'
      } else if (platform === 'quake') {
        url = 'https://quake.360.net/'
      }
      if (url) {
        BrowserOpenURL(url)
      }
    },
    setupEventListeners() {
      EventsOn('domain_collection_log', (data) => {
        const tab = this.collectionTabs.find(t => t.id === data.task_id)
        if (tab) {
          tab.logs.push({
            time: data.time,
            level: data.level,
            message: data.message
          })
          // 使用全局消息展示日志
          const messageType = data.level === 'error' ? 'error' : data.level === 'warning' ? 'warning' : 'info'
          this.$message({
            type: messageType,
            message: `[${data.time}] ${data.message}`,
            duration: data.level === 'error' ? 5000 : 3000,
            showClose: true
          })
        }
      })
    },
    removeEventListeners() {
      EventsOff('domain_collection_log')
    },
    updateTableHeight() {
      // 使用 $nextTick 确保DOM已渲染
      this.$nextTick(() => {
        const container = this.$el
        if (!container) {
          this.tableHeight = 400
          return
        }
        
        // 获取当前激活的标签页内容区域
        const activeTabPane = container.querySelector('.el-tab-pane.is-active')
        if (!activeTabPane) {
          this.tableHeight = 400
          return
        }
        
        // 获取结果卡片
        const resultsCard = activeTabPane.querySelector('.results-card')
        if (!resultsCard) {
          this.tableHeight = 400
          return
        }
        
        // 获取结果卡片的实际可用高度
        const resultsCardHeight = resultsCard.clientHeight
        const resultsCardHeader = resultsCard.querySelector('.card-header')
        const resultsCardHeaderHeight = resultsCardHeader?.clientHeight || 60
        const paginationWrapper = resultsCard.querySelector('.pagination-wrapper')
        const paginationHeight = paginationWrapper?.clientHeight || 60
        
        // 计算表格高度：卡片高度 - 头部高度 - 分页高度 - 内边距
        const padding = 20
        this.tableHeight = resultsCardHeight - resultsCardHeaderHeight - paginationHeight - padding
        
        // 确保最小高度
        if (this.tableHeight < 300) {
          this.tableHeight = 300
        }
      })
    },
    createNewTab() {
      const tabId = `tab_${++this.tabIdCounter}`
      const tab = {
        id: tabId,
        name: `域名收集任务 ${this.tabIdCounter}`,
        rootDomainsText: '',
        rootDomains: [],
        currentDomainIndex: 0,
        currentDomainProgress: 0,
        mode: 'dict',
        status: 'pending',
        loading: false,
        configExpanded: [],
        config: {
          threads: 50,
          fingerprint: true,
          selectedDicts: ['domain.txt'],
          dnsServer: '223.5.5.5',
          wildcard: true,
          ports: '80,443',
          filterIP: '',
          pocDetection: false,
          apiDetection: false,
          advancedExpanded: [],
          dictFile: '',
          dictContentText: '',
          useSystemConfig: false,
          aiConfigId: '',
          aiProvider: 'openai',
          aiAPIKey: '',
          aiAPIBase: '',
          sampleDomainsText: '',
          generateCount: 100,
          timeout: 5,
          verifyDNS: true,
          verifyHTTP: true, // HTTP验证始终开启，用于获取HTTP状态码和标题
          enableSpaceMapping: false,
          spaceMappingPlatforms: []
        },
        results: {
          totalGenerated: 0,
          validDomains: [],
          invalidDomains: []
        },
        showAllResults: false,
        logs: [],
        currentPage: 1,
        pageSize: 20
      }
      this.collectionTabs.push(tab)
      this.activeTabId = tabId
    },
    async handleTabRemove(tabId) {
      const targetTab = this.collectionTabs.find(t => t.id === tabId)
      if (!targetTab) {
        return
      }

      const hasResults = targetTab.results &&
        ((targetTab.results.validDomains?.length || targetTab.results.valid_domains?.length || targetTab.results.ValidDomains?.length || 0) > 0 ||
          (targetTab.results.invalidDomains?.length || targetTab.results.invalid_domains?.length || targetTab.results.InvalidDomains?.length || 0) > 0)

      if (hasResults) {
        try {
          await this.$confirm(
            `标签页 "${targetTab.name}" 包含查询结果，是否已保存数据？`,
            '提示',
            {
              confirmButtonText: '已保存',
              cancelButtonText: '取消',
              type: 'warning',
              distinguishCancelAndClose: true
            }
          )
        } catch (error) {
          // 用户取消
          return
        }
      }

      this.removeTab(tabId)
    },
    removeTab(tabId) {
      const index = this.collectionTabs.findIndex(t => t.id === tabId)
      if (index !== -1) {
        if (this.collectionTabs[index].status === 'running') {
          this.$message.warning('请先停止正在运行的任务')
          return
        }
        this.collectionTabs.splice(index, 1)
        if (this.activeTabId === tabId) {
          this.activeTabId = this.collectionTabs.length > 0 ? this.collectionTabs[0].id : ''
        }
      }
    },
    handleTabClick(tab) {
      // 可以在这里添加切换标签页的逻辑
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

      // 直接从事件获取坐标
      const x = event.clientX || 0
      const y = event.clientY || 0
      
      // 设置菜单位置
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
    // 关闭当前标签页（右键菜单）
    async handleCloseCurrentTab() {
      this.hideTabsContextMenu()
      
      const targetId = this.tabsContextMenuTabId || this.activeTabId
      if (!targetId) {
        return
      }
      
      const targetTab = this.collectionTabs.find(tab => tab.id === targetId)
      if (!targetTab) {
        return
      }
      
      // 检查是否有数据且未保存
      const hasResults = targetTab.results && 
        (targetTab.results.validDomains?.length > 0 || targetTab.results.ValidDomains?.length > 0)
      if (hasResults) {
        try {
          await this.$confirm(
            `标签页 "${targetTab.name}" 包含查询结果，是否已保存数据？`,
            '提示',
            {
              confirmButtonText: '已保存',
              cancelButtonText: '取消',
              type: 'warning',
              distinguishCancelAndClose: true
            }
          )
        } catch (error) {
          // 用户点击取消，不关闭标签页
          return
        }
      }
      
      // 关闭标签页
      this.removeTab(targetId)
    },
    // 关闭其他标签页（右键菜单）
    async handleCloseOtherTabs() {
      this.hideTabsContextMenu()
      
      const targetId = this.tabsContextMenuTabId || this.activeTabId
      if (!targetId) {
        return
      }
      
      // 检查其他标签页是否有数据
      const otherTabs = this.collectionTabs.filter(tab => tab.id !== targetId)
      const tabsWithData = otherTabs.filter(tab => {
        return tab.results && 
          (tab.results.validDomains?.length > 0 || tab.results.ValidDomains?.length > 0)
      })
      
      if (tabsWithData.length > 0) {
        try {
          await this.$confirm(
            `共有 ${tabsWithData.length} 个标签页包含查询结果，是否已保存数据？`,
            '提示',
            {
              confirmButtonText: '已保存',
              cancelButtonText: '取消',
              type: 'warning',
              distinguishCancelAndClose: true
            }
          )
        } catch (error) {
          // 用户点击取消，不关闭标签页
          return
        }
      }
      
      // 关闭其他标签页
      const tabsToRemove = otherTabs.map(tab => tab.id)
      for (const tabId of tabsToRemove) {
        const tab = this.collectionTabs.find(t => t.id === tabId)
        if (tab && tab.status !== 'running') {
          this.removeTab(tabId)
        } else if (tab && tab.status === 'running') {
          // 如果任务正在运行，先停止
          try {
            await this.stopCollection(tab)
            this.removeTab(tabId)
          } catch (error) {
            console.error('停止任务失败:', error)
          }
        }
      }
      
      // 确保目标标签页被激活
      this.activeTabId = targetId
    },
    // 关闭所有标签页（右键菜单）
    async handleCloseAllTabs() {
      this.hideTabsContextMenu()
      
      // 检查是否有未保存的数据
      const tabsWithData = this.collectionTabs.filter(tab => {
        return tab.results && 
          (tab.results.validDomains?.length > 0 || tab.results.ValidDomains?.length > 0)
      })
      
      if (tabsWithData.length > 0) {
        const totalResults = tabsWithData.reduce((sum, tab) => {
          const validCount = tab.results.validDomains?.length || tab.results.ValidDomains?.length || 0
          return sum + validCount
        }, 0)
        try {
          await this.$confirm(
            `共有 ${tabsWithData.length} 个标签页包含查询结果（共 ${totalResults} 条），是否已保存数据？`,
            '提示',
            {
              confirmButtonText: '已保存',
              cancelButtonText: '取消',
              type: 'warning',
              distinguishCancelAndClose: true
            }
          )
        } catch (error) {
          // 用户点击取消，不关闭标签页
          return
        }
      }
      
      // 关闭所有标签页（先停止运行中的任务）
      const tabsToRemove = [...this.collectionTabs]
      for (const tab of tabsToRemove) {
        if (tab.status === 'running') {
          try {
            await this.stopCollection(tab)
          } catch (error) {
            console.error('停止任务失败:', error)
          }
        }
        this.removeTab(tab.id)
      }
    },
    async handleModeChange(tab) {
      // 模式切换时的处理
      // 如果是AI模式，默认启用空间测绘并检查key
      if (tab.mode === 'ai') {
        // 默认启用空间测绘
        if (!tab.config.enableSpaceMapping) {
          tab.config.enableSpaceMapping = true
          await this.handleSpaceMappingChange(tab)
        }
        await this.checkSpaceMappingKeysForAI(tab)
      }
    },
    async checkSpaceMappingKeysForAI(tab) {
      // 检查空间测绘key配置
      try {
        if (!this.apiClient) return
        
        const response = await this.apiClient.get('/api/info-collection/config')
        if (response.data && response.data.success) {
          const config = response.data.data
          if (config.apiKeys) {
            // 检查是否有有效的key
            let hasValidKey = false
            for (const [platform, key] of Object.entries(config.apiKeys)) {
              if (key && config.enabled?.[platform]) {
                hasValidKey = true
                break
              }
            }
            
            if (!hasValidKey) {
              this.$message.warning('空间测绘Key未配置或未启用，AI模式将无法自动调用空间测绘功能。请在配置API中设置。')
            }
          } else {
            this.$message.warning('空间测绘Key未配置，AI模式将无法自动调用空间测绘功能。请在配置API中设置。')
          }
        }
      } catch (error) {
        console.error('检查空间测绘配置失败:', error)
      }
    },
    async handleSpaceMappingChange(tab) {
      // 当开启空间测绘时，检查key并自动选择已启用的平台
      if (tab.config.enableSpaceMapping) {
        try {
          if (!this.apiClient) return
          
          const response = await this.apiClient.get('/api/info-collection/config')
          if (response.data && response.data.success) {
            const config = response.data.data
            if (config.apiKeys && config.enabled) {
              // 自动选择已启用且有key的平台
              const enabledPlatforms = []
              const allPlatforms = ['0zone', 'fofa', 'otherfofa', 'hunter', 'quake']
              
              for (const platform of allPlatforms) {
                const key = config.apiKeys[platform]
                const enabled = config.enabled[platform]
                // FOFA官方需要检查email和key
                if (platform === 'fofa') {
                  const hasKey = key && key.includes(':') && key.split(':').length === 2
                  if (hasKey && enabled) {
                    enabledPlatforms.push(platform)
                  }
                } else {
                  if (key && key.trim() !== '' && enabled) {
                    enabledPlatforms.push(platform)
                  }
                }
              }
              
              if (enabledPlatforms.length > 0) {
                // 自动选择已启用的平台
                tab.config.spaceMappingPlatforms = enabledPlatforms
                this.$message.success(`已自动选择 ${enabledPlatforms.length} 个已启用的空间测绘平台`)
              } else {
                // 所有平台都未配置，提示用户
                this.$message.warning('空间测绘Key未配置或未启用，请在配置API中设置。')
                tab.config.enableSpaceMapping = false // 关闭开关
              }
            } else {
              this.$message.warning('空间测绘Key未配置，请在配置API中设置。')
              tab.config.enableSpaceMapping = false // 关闭开关
            }
          }
        } catch (error) {
          console.error('检查空间测绘配置失败:', error)
          this.$message.error('检查空间测绘配置失败，请确保已正确配置空间测绘Key。')
          tab.config.enableSpaceMapping = false // 关闭开关
        }
      } else {
        // 关闭空间测绘时，清空已选择的平台
        tab.config.spaceMappingPlatforms = []
      }
    },
    async startCollection(tab) {
      const domainsText = tab.rootDomainsText || tab.rootDomain || ''
      const domains = domainsText.split('\n')
        .map(line => line.trim())
        .filter(line => line && !line.startsWith('http://') && !line.startsWith('https://'))

      if (domains.length === 0) {
        this.$message.warning('请输入域名，每行一个')
        return
      }
      
      // AI模式验证：必须启用空间测绘或提供样本域名
      if (tab.mode === 'ai') {
        const hasSpaceMapping = tab.config.enableSpaceMapping && tab.config.spaceMappingPlatforms && tab.config.spaceMappingPlatforms.length > 0
        const hasSampleDomains = tab.config.sampleDomainsText && tab.config.sampleDomainsText.trim().split('\n').filter(line => line.trim()).length > 0
        
        if (!hasSpaceMapping && !hasSampleDomains) {
          this.$message.error('AI模式需要至少以下之一：1) 启用空间测绘并选择平台，或 2) 提供样本域名')
          return
        }
      }

      tab.rootDomains = domains
      tab.currentDomainIndex = 0
      tab.currentDomainProgress = 0

      await this.processNextDomain(tab)
    },
    async processNextDomain(tab) {
      if (tab.currentDomainIndex >= tab.rootDomains.length) {
        tab.status = 'completed'
        this.$message.success('所有域名处理完成')
        return
      }

      const rootDomain = tab.rootDomains[tab.currentDomainIndex]
      tab.rootDomain = rootDomain
      tab.status = 'running'
      tab.currentDomainProgress = 0

      tab.loading = true
      try {
        // 处理字典文件路径
        // 处理字典文件：优先使用selectedDicts，如果没有则使用dictFile
        let dictFile = tab.config.dictFile
        if (!dictFile && tab.config.selectedDicts && tab.config.selectedDicts.length > 0) {
          // 使用第一个选中的字典文件
          dictFile = tab.config.selectedDicts[0]
          // 如果selectedDicts是文件名，后端会自动从domain目录查找
        }
        
        // 如果还是没有字典文件，使用默认的domain.txt
        if (!dictFile) {
          dictFile = 'domain.txt'
        }
        
        const config = {
          dict_file: dictFile,
          dict_content: tab.config.dictContentText
            ? tab.config.dictContentText.split('\n').filter(line => line.trim())
            : [],
          ai_config_id: tab.config.useSystemConfig ? tab.config.aiConfigId : '',
          ai_provider: tab.config.aiProvider,
          ai_api_key: tab.config.useSystemConfig ? '' : tab.config.aiAPIKey,
          ai_api_base: tab.config.useSystemConfig ? '' : (tab.config.aiAPIBase || ''),
          sample_domains: tab.config.sampleDomainsText
            ? tab.config.sampleDomainsText.split('\n').filter(line => line.trim())
            : [],
          generate_count: tab.config.generateCount,
          threads: tab.config.threads,
          timeout: tab.config.timeout,
          verify_dns: tab.config.verifyDNS,
          verify_http: true, // HTTP验证始终开启，用于获取HTTP状态码和标题
          dns_server: tab.config.dnsServer,
          wildcard: tab.config.wildcard,
          ports: tab.config.ports,
          filter_ip: tab.config.filterIP,
          fingerprint: tab.config.fingerprint,
          poc_detection: tab.config.pocDetection,
          api_detection: tab.config.apiDetection,
          enable_space_mapping: tab.config.enableSpaceMapping,
          space_mapping_platforms: tab.config.spaceMappingPlatforms || []
        }

        // 构建任务名称，包含根域名信息
        const taskName = rootDomain ? `${tab.name} - ${rootDomain}` : tab.name
        
        const taskId = await CreateCollectionTask(
          taskName,
          rootDomain.trim(),
          tab.mode,
          JSON.stringify(config)
        )

        tab.taskId = taskId
        tab.logs = []

        await StartCollection(taskId)
        this.startTaskUpdateTimer(tab)
      } catch (error) {
        this.$message.error(`启动收集失败: ${error}`)
        tab.status = 'error'
        tab.loading = false
      }
    },
    async stopCollection(tab) {
      if (!tab.taskId) {
        this.$message.warning('没有正在运行的任务')
        return
      }

      if (tab.status !== 'running') {
        this.$message.warning('任务未在运行中')
        return
      }

      try {
        await StopCollection(tab.taskId)
        tab.status = 'stopped'
        tab.loading = false
        // 清除定时器
        if (tab.updateTimer) {
          clearInterval(tab.updateTimer)
          tab.updateTimer = null
        }
        this.$message.success('收集已停止')
      } catch (error) {
        this.$message.error(`停止收集失败: ${error}`)
      }
    },
    startTaskUpdateTimer(tab) {
      if (tab.updateTimer) {
        clearInterval(tab.updateTimer)
      }

      // 使用更短的更新间隔（500ms），提高响应速度
      tab.updateTimer = setInterval(async () => {
        if (!tab.taskId || tab.status !== 'running') {
          if (tab.updateTimer) {
            clearInterval(tab.updateTimer)
            tab.updateTimer = null
          }
          return
        }

        try {
          const taskJson = await GetTask(tab.taskId)
          const task = JSON.parse(taskJson)

          tab.status = task.status
          
          // 统一处理字段名（后端可能返回 valid_domains 或 ValidDomains）
          if (task.results) {
            // 统一转换为 validDomains 格式
            if (task.results.valid_domains && !task.results.validDomains) {
              task.results.validDomains = task.results.valid_domains
            } else if (task.results.ValidDomains && !task.results.validDomains) {
              task.results.validDomains = task.results.ValidDomains
            }
            if (task.results.invalid_domains && !task.results.invalidDomains) {
              task.results.invalidDomains = task.results.invalid_domains
            } else if (task.results.InvalidDomains && !task.results.invalidDomains) {
              task.results.invalidDomains = task.results.InvalidDomains
            }
            if (task.results.total_generated !== undefined && task.results.totalGenerated === undefined) {
              task.results.totalGenerated = task.results.total_generated
            } else if (task.results.TotalGenerated !== undefined && task.results.totalGenerated === undefined) {
              task.results.totalGenerated = task.results.TotalGenerated
            }
            
            // 统一处理域名信息中的HTTP状态和标题字段
            if (task.results.validDomains) {
              task.results.validDomains = task.results.validDomains.map(domain => {
                const mapped = { ...domain }
                // 处理 http_status 字段
                if (mapped.http_status !== undefined) {
                  mapped.httpStatus = mapped.http_status
                }
                // 处理 http_title 字段
                if (mapped.http_title !== undefined) {
                  mapped.httpTitle = mapped.http_title
                }
                // 处理 ips 字段（可能是 IPs）
                if (mapped.IPs && !mapped.ips) {
                  mapped.ips = mapped.IPs
                }
                // 处理 source 字段（可能是 Source）
                if (!mapped.source && mapped.Source) {
                  mapped.source = mapped.Source
                }
                // 处理 platform 字段（可能是 Platform）
                if (!mapped.platform && mapped.Platform) {
                  mapped.platform = mapped.Platform
                }
                return mapped
              })
            }
          }
          
          tab.results = task.results || tab.results
          tab.logs = task.logs || tab.logs

          // 检查是否有新验证的域名需要调用相应模块
          if (task.results && task.results.validDomains) {
            this.handleNewValidDomains(tab, task.results.validDomains)
          }

          // 更新进度 - 基于实际验证的域名数量
          if (task.results) {
            // 优先使用 total_generated（后端返回的字段名）
            const totalGenerated = task.results.total_generated || task.results.TotalGenerated || 0
            const validCount = (task.results.valid_domains?.length || task.results.ValidDomains?.length || task.results.validDomains?.length || 0)
            const invalidCount = (task.results.invalid_domains?.length || task.results.InvalidDomains?.length || task.results.invalidDomains?.length || 0)
            const verifiedCount = validCount + invalidCount
            
            if (totalGenerated > 0) {
              // 如果有总生成数，基于已验证的域名计算进度
              const progress = (verifiedCount / totalGenerated) * 100
              // 修复浮点数精度问题，保留2位小数
              tab.currentDomainProgress = Math.min(Math.max(Math.round(progress * 100) / 100, 0), 100)
            } else if (tab.rootDomains && tab.rootDomains.length > 0) {
              // 如果没有生成数，基于根域名进度计算
              const progress = ((tab.currentDomainIndex + 1) / tab.rootDomains.length) * 100
              // 修复浮点数精度问题
              tab.currentDomainProgress = Math.min(Math.max(Math.round(progress * 100) / 100, 0), 100)
            } else if (verifiedCount > 0) {
              // 如果已经开始验证，显示一个基础进度
              tab.currentDomainProgress = Math.min(Math.round(verifiedCount * 2 * 100) / 100, 95) // 最多显示95%，避免过早显示100%
            }
            
            // 如果任务已完成，立即刷新一次结果
            if (task.status === 'completed' && tab.status !== 'completed') {
              tab.status = 'completed'
              tab.loading = false
              // 立即再获取一次最新结果，确保数据完整
              setTimeout(async () => {
                try {
                  const latestTaskJson = await GetTask(tab.taskId)
                  const latestTask = JSON.parse(latestTaskJson)
                  if (latestTask.results) {
                    // 统一处理字段名
                    if (latestTask.results.valid_domains && !latestTask.results.validDomains) {
                      latestTask.results.validDomains = latestTask.results.valid_domains
                    } else if (latestTask.results.ValidDomains && !latestTask.results.validDomains) {
                      latestTask.results.validDomains = latestTask.results.ValidDomains
                    }
                    if (latestTask.results.validDomains) {
                      latestTask.results.validDomains = latestTask.results.validDomains.map(domain => {
                        const mapped = { ...domain }
                        if (mapped.http_status !== undefined) mapped.httpStatus = mapped.http_status
                        if (mapped.http_title !== undefined) mapped.httpTitle = mapped.http_title
                        if (mapped.IPs && !mapped.ips) mapped.ips = mapped.IPs
                        if (!mapped.source && mapped.Source) mapped.source = mapped.Source
                        if (!mapped.platform && mapped.Platform) mapped.platform = mapped.Platform
                        return mapped
                      })
                    }
                    tab.results = latestTask.results
                    tab.currentDomainProgress = 100
                  }
                } catch (error) {
                  console.error('获取最新结果失败:', error)
                }
              }, 200)
            }
          } else if (tab.rootDomains && tab.rootDomains.length > 0) {
            // 如果没有结果，基于根域名进度计算
            const progress = ((tab.currentDomainIndex + 1) / tab.rootDomains.length) * 100
            // 修复浮点数精度问题
            tab.currentDomainProgress = Math.min(Math.max(Math.round(progress * 100) / 100, 0), 100)
          }

          if (task.status === 'completed' || task.status === 'error' || task.status === 'stopped') {
            if (tab.updateTimer) {
              clearInterval(tab.updateTimer)
              tab.updateTimer = null
            }
            tab.loading = false

            // 如果当前域名完成，继续下一个
            if (task.status === 'completed' && tab.currentDomainIndex < tab.rootDomains.length - 1) {
              tab.currentDomainIndex++
              tab.currentDomainProgress = 0
              setTimeout(() => {
                this.processNextDomain(tab)
              }, 1000)
            }
          }

        } catch (error) {
          console.error('更新任务状态失败:', error)
        }
      }, 500) // 使用500ms间隔，提高响应速度
    },
    async loadDictList() {
      try {
        const dictListJson = await GetDictList()
        const dictList = JSON.parse(dictListJson)
        this.dicts = dictList || []
        if (this.dicts.length > 0 && this.currentTab && (!this.currentTab.config.selectedDicts || this.currentTab.config.selectedDicts.length === 0)) {
          const domainDict = this.dicts.find(d => d.name === 'domain.txt')
          if (domainDict) {
            this.currentTab.config.selectedDicts = ['domain.txt']
          }
        }
      } catch (error) {
        console.error('加载字典列表失败:', error)
      }
    },
    formatSize(bytes) {
      if (bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    },
    async loadAIConfigs() {
      try {
        const configsJson = await GetAllConfigs()
        this.aiConfigs = JSON.parse(configsJson).filter(c => c.enabled)
      } catch (error) {
        console.error('加载AI配置列表失败:', error)
      }
    },
    handleSystemConfigChange(tab) {
      if (!tab.config.useSystemConfig) {
        tab.config.aiConfigId = ''
      } else {
        this.loadAIConfigs()
      }
    },
    getProviderName(provider) {
      const names = {
        openai: 'OpenAI',
        deepseek: 'DeepSeek',
        qianwen: '阿里云千机百炼',
        custom: '自定义'
      }
      return names[provider] || provider
    },
    getPagedResults(tab) {
      if (!tab.results) {
        return []
      }

      // 统一处理字段名
      let validDomains = tab.results.validDomains || tab.results.valid_domains || tab.results.ValidDomains || []
      let invalidDomains = tab.results.invalidDomains || tab.results.invalid_domains || tab.results.InvalidDomains || []

      validDomains = validDomains || []
      invalidDomains = invalidDomains || []

      let allDomains = tab.showAllResults ? [...validDomains, ...invalidDomains] : validDomains

      if (!allDomains || allDomains.length === 0) {
        return []
      }

      // 确保字段映射正确（处理 http_status -> httpStatus, http_title -> httpTitle）
      allDomains = allDomains.map(domain => {
        const mapped = { ...domain }
        // 处理 http_status 字段
        if (mapped.http_status !== undefined && mapped.httpStatus === undefined) {
          mapped.httpStatus = mapped.http_status
        }
        // 处理 http_title 字段
        if (mapped.http_title !== undefined && mapped.httpTitle === undefined) {
          mapped.httpTitle = mapped.http_title
        }
        // 处理 ips 字段（可能是 IPs）
        if (mapped.IPs && !mapped.ips) {
          mapped.ips = mapped.IPs
        }
        // 确保 source 和 platform 字段存在
        if (!mapped.source && mapped.Source) {
          mapped.source = mapped.Source
        }
        if (!mapped.platform && mapped.Platform) {
          mapped.platform = mapped.Platform
        }
        return mapped
      })

      const start = (tab.currentPage - 1) * tab.pageSize
      const end = start + tab.pageSize
      return allDomains.slice(start, end)
    },
    getTotalResults(tab) {
      if (!tab.results) {
        return 0
      }
      const valid = tab.results.validDomains || tab.results.valid_domains || tab.results.ValidDomains || []
      const invalid = tab.results.invalidDomains || tab.results.invalid_domains || tab.results.InvalidDomains || []
      const validLen = valid ? valid.length : 0
      const invalidLen = invalid ? invalid.length : 0
      return tab.showAllResults ? validLen + invalidLen : validLen
    },
    handleSizeChange(size, tab) {
      tab.pageSize = size
      tab.currentPage = 1
    },
    handleCurrentChange(page, tab) {
      tab.currentPage = page
    },
    async exportToExcel(tab) {
      if (!tab.taskId) {
        this.$message.warning('请先运行收集任务')
        return
      }

      try {
        const filePath = await ExportResults(tab.taskId)
        this.$message.success(`导出成功: ${filePath}`)
      } catch (error) {
        this.$message.error(`导出失败: ${error}`)
      }
    },
    clearResults(tab) {
      if (tab.status === 'running') {
        this.$message.warning('请先停止正在运行的任务')
        return
      }
      tab.results = {
        totalGenerated: 0,
        validDomains: [],
        invalidDomains: []
      }
    },
    clearLogs(tab) {
      tab.logs = []
    },
    scrollLogsToBottom(tab) {
      // 不再需要滚动，因为只显示最新日志
    },
    getDisplayLogs(tab) {
      if (!tab.logs || tab.logs.length === 0) {
        return []
      }
      const maxLogs = 30
      return tab.logs.slice(-maxLogs)
    },
    copyToClipboard(text) {
      navigator.clipboard.writeText(text).then(() => {
        this.$message.success('已复制到剪贴板')
      }).catch(() => {
        this.$message.error('复制失败')
      })
    },
    openUrl(url) {
      BrowserOpenURL(url)
    },
    handleDomainClick(domain) {
      // 点击域名直接打开浏览器
      const url = domain.startsWith('http://') || domain.startsWith('https://') 
        ? domain 
        : `http://${domain}`
      BrowserOpenURL(url)
    },
    getHTTPStatusTagType(status) {
      if (status >= 200 && status < 300) return 'success'
      if (status >= 300 && status < 400) return 'warning'
      if (status >= 400) return 'danger'
      return 'info'
    },
    getStatusTagType(status) {
      const statusMap = {
        running: 'warning',
        completed: 'success',
        error: 'danger',
        stopped: 'info',
        pending: 'info'
      }
      return statusMap[status] || 'info'
    },
    getStatusText(status) {
      const statusMap = {
        running: '运行中',
        completed: '已完成',
        error: '错误',
        stopped: '已停止',
        pending: '待开始'
      }
      return statusMap[status] || status
    },
    getDomainSourceText(source, platform) {
      switch (source) {
        case 'ai':
          return 'AI'
        case 'space-mapping':
          // 如果有平台信息，显示具体平台名称
          if (platform && platform.trim() !== '') {
            return platform
          }
          return '测绘平台'
        case 'hybrid':
          return '混合'
        case 'dict':
        default:
          return '字典'
      }
    },
    // 解析多个来源，返回标签数组
    getDomainSources(row) {
      const sources = []
      const sourceStr = row.source || this.currentTab?.mode || ''
      const platform = row.platform || ''
      
      // 如果source包含逗号，说明有多个来源
      if (sourceStr.includes(',')) {
        const sourceList = sourceStr.split(',').map(s => s.trim()).filter(s => s)
        sourceList.forEach(source => {
          if (source === 'space-mapping') {
            // 空间测绘来源，显示具体平台（如果有）
            if (platform && platform.trim() !== '') {
              sources.push({
                source: 'space-mapping',
                text: platform
              })
            } else {
              sources.push({
                source: 'space-mapping',
                text: '测绘平台'
              })
            }
          } else {
            sources.push({
              source: source,
              text: this.getDomainSourceText(source, '')
            })
          }
        })
      } else {
        // 单个来源
        const source = sourceStr || 'dict'
        if (source === 'space-mapping' && platform && platform.trim() !== '') {
          sources.push({
            source: 'space-mapping',
            text: platform
          })
        } else {
          sources.push({
            source: source,
            text: this.getDomainSourceText(source, platform)
          })
        }
      }
      
      // 去重（避免重复显示相同的来源）
      const uniqueSources = []
      const seen = new Set()
      sources.forEach(item => {
        const key = `${item.source}:${item.text}`
        if (!seen.has(key)) {
          seen.add(key)
          uniqueSources.push(item)
        }
      })
      
      return uniqueSources
    },
    getDomainSourceTagType(source) {
      switch (source) {
        case 'ai':
          return 'success'
        case 'space-mapping':
          return 'warning'
        case 'hybrid':
          return 'info'
        case 'dict':
        default:
          return undefined // 返回 undefined 而不是空字符串，这样 el-tag 不会传递 type 属性
      }
    },
    formatTime(timeStr) {
      if (!timeStr) return '-'
      try {
        const date = new Date(timeStr)
        if (isNaN(date.getTime())) return timeStr
        return date.toLocaleString('zh-CN', {
          year: 'numeric',
          month: '2-digit',
          day: '2-digit',
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit'
        })
      } catch (e) {
        return timeStr
      }
    },
    handleHistoryDialogOpened() {
      this.loadHistory()
    },
    async refreshTasks() {
      await this.loadHistory()
    },
    async clearAllTasks() {
      try {
        await this.$confirm('确定清除所有任务？此操作不可恢复。', '清除全部任务', { type: 'warning' })
        await ClearAllTasks()
        this.historyList = []
        // 清空所有标签页的任务关联
        this.collectionTabs.forEach(tab => {
          tab.taskId = ''
          tab.status = 'pending'
        })
        this.$message.success('已清除全部任务')
      } catch (error) {
        if (error !== 'cancel') {
          this.$message.error(`清除失败: ${error}`)
        }
      }
    },
    async saveTaskLog(taskId) {
      try {
        await SaveTaskLog(taskId)
        this.$message.success('日志保存成功')
      } catch (error) {
        this.$message.error(`保存日志失败: ${error}`)
      }
    },
    async loadHistory() {
      this.historyLoading = true
      try {
        const tasksJson = await GetAllTasks()
        let tasks = JSON.parse(tasksJson)
        
        // 统一处理字段名映射，确保所有字段都能正确显示
        tasks = tasks.map(task => {
          // 处理任务基本字段的命名（后端可能返回下划线风格）
          if (task.root_domain && !task.rootDomain) {
            task.rootDomain = task.root_domain
          }
          if (task.start_time && !task.startTime) {
            task.startTime = task.start_time
          }
          if (task.end_time && !task.endTime) {
            task.endTime = task.end_time
          }
          // 确保 id 字段存在（可能从 ID 或 id 字段获取）
          if (!task.id) {
            task.id = task.ID || task.id
          }
          
          // 处理 results 字段
          if (task.results) {
            // 统一转换为 validDomains 格式
            if (task.results.valid_domains && !task.results.validDomains) {
              task.results.validDomains = task.results.valid_domains
            } else if (task.results.ValidDomains && !task.results.validDomains) {
              task.results.validDomains = task.results.ValidDomains
            }
            // 统一处理 totalGenerated
            if (task.results.total_generated !== undefined && task.results.totalGenerated === undefined) {
              task.results.totalGenerated = task.results.total_generated
            } else if (task.results.TotalGenerated !== undefined && task.results.totalGenerated === undefined) {
              task.results.totalGenerated = task.results.TotalGenerated
            }
          }
          return task
        })
        
        this.historyList = tasks
      } catch (error) {
        this.$message.error(`加载任务列表失败: ${error}`)
      } finally {
        this.historyLoading = false
      }
    },
    toggleResultFilter(tab) {
      tab.showAllResults = !tab.showAllResults
      tab.currentPage = 1
    },
    async deleteTask(taskId) {
      try {
        await this.$confirm('确定删除该任务及其记录？', '删除任务', { type: 'warning' })
        await DeleteTask(taskId)
        this.$message.success('删除成功')
        this.loadHistory()
      } catch (error) {
        if (error !== 'cancel' && error !== 'close') {
          this.$message.error(`删除失败: ${error}`)
        }
      }
    },
    async loadTaskToTab(task) {
      let tab = this.collectionTabs.find(t => t.taskId === task.id)

      if (!tab) {
        const tabId = `tab_${++this.tabIdCounter}`
        // 从任务名称中提取根域名（如果任务名称包含根域名）
        let rootDomain = task.rootDomain || ''
        let taskName = task.name || ''
        
        // 如果任务名称包含 " - "，提取根域名部分
        if (taskName.includes(' - ') && !rootDomain) {
          const parts = taskName.split(' - ')
          if (parts.length > 1) {
            rootDomain = parts[parts.length - 1]
            taskName = parts.slice(0, -1).join(' - ')
          }
        }
        
        // 兼容不同字段命名（后端可能返回下划线风格）
        const cfg = task.config || {}
        const getCfg = (camelKey, snakeKey, pascalKey, defaultVal) => {
          if (cfg[camelKey] !== undefined) return cfg[camelKey]
          if (cfg[snakeKey] !== undefined) return cfg[snakeKey]
          if (cfg[pascalKey] !== undefined) return cfg[pascalKey]
          return defaultVal
        }

        tab = {
          id: tabId,
          name: taskName || `域名收集任务 ${this.tabIdCounter}`,
          rootDomain: rootDomain,
          rootDomainsText: rootDomain ? rootDomain : '',
          rootDomains: rootDomain ? [rootDomain] : [],
          mode: task.mode,
          status: task.status,
          loading: false,
          taskId: task.id,
          config: {
            // 字典 & AI 相关
            dictFile: getCfg('dictFile', 'dict_file', 'DictFile', ''),
            dictContentText: Array.isArray(getCfg('dictContent', 'dict_content', 'DictContent', null))
              ? getCfg('dictContent', 'dict_content', 'DictContent', []).join('\n')
              : '',
            selectedDicts: getCfg('selectedDicts', 'selected_dicts', 'SelectedDicts', []),
            useSystemConfig: !!getCfg('useSystemConfig', 'use_system_config', 'UseSystemConfig', false),
            aiConfigId: getCfg('aiConfigId', 'ai_config_id', 'AIConfigID', ''),
            aiProvider: getCfg('aiProvider', 'ai_provider', 'AIProvider', 'openai'),
            aiAPIKey: getCfg('aiAPIKey', 'ai_api_key', 'AIAPIKey', ''),
            aiAPIBase: getCfg('aiAPIBase', 'ai_api_base', 'AIAPIBase', ''),
            sampleDomainsText: Array.isArray(getCfg('sampleDomains', 'sample_domains', 'SampleDomains', null))
              ? getCfg('sampleDomains', 'sample_domains', 'SampleDomains', []).join('\n')
              : '',

            // 通用探测参数
            generateCount: getCfg('generateCount', 'generate_count', 'GenerateCount', 300),
            threads: getCfg('threads', 'threads', 'Threads', 50),
            timeout: getCfg('timeout', 'timeout', 'Timeout', 5),
            dnsServer: getCfg('dnsServer', 'dns_server', 'DNSServer', '223.5.5.5'),
            verifyDNS: getCfg('verifyDNS', 'verify_dns', 'VerifyDNS', true) !== false,
            verifyHTTP: true, // HTTP验证始终开启，用于获取HTTP状态码和标题
            wildcard: !!getCfg('wildcard', 'wildcard', 'Wildcard', true),
            ports: getCfg('ports', 'ports', 'Ports', '80,443'),
            filterIP: getCfg('filterIP', 'filter_ip', 'FilterIP', ''),

            // 功能开关（历史任务保留原有勾选）
            fingerprint: getCfg('fingerprint', 'fingerprint', 'Fingerprint', true),
            pocDetection: !!getCfg('pocDetection', 'poc_detection', 'POCDetection', false),
            apiDetection: !!getCfg('apiDetection', 'api_detection', 'APIDetection', false),
            enableSpaceMapping: !!getCfg('enableSpaceMapping', 'enable_space_mapping', 'EnableSpaceMapping', false),
            spaceMappingPlatforms: getCfg('spaceMappingPlatforms', 'space_mapping_platforms', 'SpaceMappingPlatforms', []) || [],

            // 高级面板展开状态
            advancedExpanded: getCfg('advancedExpanded', 'advanced_expanded', 'AdvancedExpanded', [])
          },
          results: task.results || {
            totalGenerated: 0,
            validDomains: [],
            invalidDomains: []
          },
          showAllResults: false,
          logs: task.logs || [],
          currentPage: 1,
          pageSize: 20
        }
        this.collectionTabs.push(tab)
      }

      this.activeTabId = tab.id
      this.historyDialog = false
    },
    async resetOptions(tab) {
      try {
        await this.$confirm(
          '确定要重置所有配置选项吗？当前标签内的自定义项将丢失。',
          '重置配置',
          { type: 'warning' }
        )
      } catch {
        return
      }
      tab.config = {
        threads: 50,
        fingerprint: true,
        selectedDicts: [],
        dnsServer: '223.5.5.5',
        wildcard: true,
        ports: '80,443',
        filterIP: '',
        pocDetection: false,
        apiDetection: true,
        advancedExpanded: [],
        dictFile: '',
        dictContentText: '',
        useSystemConfig: false,
        aiConfigId: '',
        aiProvider: 'openai',
        aiAPIKey: '',
        aiAPIBase: '',
        sampleDomainsText: '',
        generateCount: 300,
        timeout: 5,
        verifyDNS: true,
        verifyHTTP: true,
        enableSpaceMapping: false,
        spaceMappingPlatforms: []
      }
      this.$message.success('已重置所有选项')
    },
    // 处理新验证的域名，调用相应模块
    handleNewValidDomains(tab, validDomains) {
      if (!validDomains || validDomains.length === 0) return
      
      // 获取已处理的域名ID集合（使用domain作为唯一标识）
      const processedDomains = new Set(tab.processedDomainsForModules || [])
      
      // 筛选出需要处理的新域名
      const newDomains = validDomains.filter(domain => {
        const domainKey = domain.domain || domain
        return !processedDomains.has(domainKey) && domain.status === 'valid' && domain.httpStatus > 0
      })
      
      if (newDomains.length === 0) return
      
      // 构建URL列表
      const urls = []
      for (const domainInfo of newDomains) {
        const domain = domainInfo.domain || domainInfo
        if (domainInfo.httpStatus > 0) {
          // 根据HTTP状态码判断协议（简化处理，实际应该从响应中获取）
          const protocol = domainInfo.httpStatus === 200 ? 'https' : 'http'
          urls.push(`${protocol}://${domain}`)
        }
      }
      
      if (urls.length === 0) return
      
      // 根据配置调用相应模块（只调用一次，避免重复调用）
      if (tab.config.fingerprint && urls.length > 0 && !tab.fingerprintCalled) {
        this.callFingerprintModule(urls, tab)
        tab.fingerprintCalled = true
      }
      
      if (tab.config.apiDetection && urls.length > 0 && !tab.apiDetectionCalled) {
        this.callBrowserScanModule(urls, tab)
        tab.apiDetectionCalled = true
      }
      
      if (tab.config.pocDetection && urls.length > 0 && !tab.pocDetectionCalled) {
        this.preparePOCScan(urls, tab)
        tab.pocDetectionCalled = true
      }
      
      // 标记已处理的域名
      if (!tab.processedDomainsForModules) {
        tab.processedDomainsForModules = []
      }
      newDomains.forEach(domainInfo => {
        const domainKey = domainInfo.domain || domainInfo
        if (!tab.processedDomainsForModules.includes(domainKey)) {
          tab.processedDomainsForModules.push(domainKey)
        }
      })
    },
    // 调用指纹识别模块
    async callFingerprintModule(urls, tab) {
      try {
        const scanConfig = {
          targets: urls,
          timeout: tab.config.timeout || 8,
          threads: Math.min(tab.config.threads || 4, 10), // 限制最大线程数
          dirThreads: 4,
          maxDirChecks: 400,
          dirMode: false,
          useWappalyzer: true,
          useTideRules: true,
          useCmsDict: true,
          usePoweredBy: true,
          checkAlive: false,
          followRedirect: true,
          headers: {},
          tabId: tab.id
        }
        
        await StartFingerprintScan(JSON.stringify(scanConfig))
        this.$message.success(`已启动指纹识别，目标数量: ${urls.length}`)
      } catch (error) {
        console.error('启动指纹识别失败:', error)
        this.$message.error(`启动指纹识别失败: ${error}`)
      }
    },
    // 调用 JSFinder 模块（接口探测）
    // 接口探测功能：自动发送到 JSFinder 模块进行扫描，JSFinder 会自动发现和测试API端点
    async callBrowserScanModule(urls, tab) {
      try {
        // 去重URL（避免重复扫描）
        const uniqueUrls = [...new Set(urls)]
        
        // 只对第一个可访问的URL创建扫描任务（避免创建过多任务）
        // 优先使用https协议的URL
        const httpsUrl = uniqueUrls.find(url => url.startsWith('https://'))
        const targetUrl = httpsUrl || uniqueUrls[0]
        
        if (!targetUrl) {
          return
        }
        
        const config = {
          wait_time: 5,
          extract_rules: [],
          api_routes: ['/api', '/api/v1', '/api/v2', '/rest', '/graphql'],
          global_api_routes: ['/api', '/api/v1', '/api/v2'],
          allow_list: [],
          auto_test_api: false, // 自动测试API
          test_methods: ['GET', 'POST'],
          worker_num: 10
        }
        
        await CreateTask(targetUrl, config)
        window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'browser-scan' }))
        this.$message.success(`已启动 JSFinder 接口探测，目标: ${targetUrl}`)
      } catch (error) {
        this.$message.error(`启动 JSFinder 失败: ${error}`)
      }
    },
    // 准备POC扫描（弹出对话框）
    preparePOCScan(urls, tab) {
      // POC检测：自动发送至漏洞扫描模块，然后弹出对话框创建扫描任务
      this.pocScanTargets = urls
      this.pocScanTaskName = `域名收集-POC检测-${new Date().toLocaleTimeString()}`
      this.pocScanMode = 'full'
      this.pocScanCustomTags = []
      this.pocScanSkipNucleiWithoutTags = false
      this.pocScanDialogVisible = true
    },
    // 确认创建POC扫描任务
    async confirmPOCScan() {
      if (this.pocScanTargets.length === 0) {
        this.$message.warning('没有有效的目标URL')
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
      
      // 根据扫描模式设置参数
      let tagsParam = []
      let templateFileParam = []
      
      if (this.pocScanMode === 'fingerprint') {
        // 指纹扫描模式
        tagsParam = [...this.pocScanCustomTags]
      } else {
        // 全量扫描模式
        tagsParam = []
      }
      
      const payload = {
        targets: this.pocScanTargets,
        task_name: this.pocScanTaskName,
        append_poc: '',
        tags: [],
        custom_tags: tagsParam,
        template_file: templateFileParam,
        template_folders: templateFolders,
        custom_headers: '',
        proxy: proxyString,
        skip_nuclei_without_tags: this.pocScanSkipNucleiWithoutTags,
        thread_safe: true,
        threads: scanConfig.threads,
        timeout: scanConfig.timeout,
        rate_limit: scanConfig.rate_limit,
        retries: scanConfig.retries
      }
      
      try {
        const taskId = await StartPocScan(JSON.stringify(payload))
        this.pocScanDialogVisible = false
        this.$message.success({
          message: `已启动POC检测任务，目标数量 ${this.pocScanTargets.length}。请在"漏洞扫描"模块查看进度。`,
          duration: 3000
        })
      } catch (error) {
        console.error('启动POC检测任务失败:', error)
        this.$message.error(`启动POC检测任务失败: ${error?.message || error}`)
      }
    },
    // 表格行右键菜单和多选相关方法
    handleRowContextMenu(row, column, event) {
      event.preventDefault()
      event.stopPropagation()

      this.selectedRow = row

      const x = event.clientX || 0
      const y = event.clientY || 0
      
      // 先用点击位置显示菜单
      this.rowContextMenuStyle.left = `${x}px`
      this.rowContextMenuStyle.top = `${y}px`
      this.rowContextMenuVisible = true

      // 基于实际菜单高度做一次轻微修正，防止刚好溢出窗口底部
      this.$nextTick(() => {
        const menuEl = document.querySelector('.context-menu')
        if (!menuEl) return

        const menuHeight = menuEl.offsetHeight || 0
        const windowHeight = window.innerHeight
        if (y + menuHeight > windowHeight - 8) {
          let adjustedY = windowHeight - menuHeight - 8
          if (adjustedY < 8) adjustedY = 8
          this.rowContextMenuStyle.top = `${adjustedY}px`
        }
      })

      const hideMenu = (e) => {
        if (!e.target.closest('.context-menu')) {
          this.rowContextMenuVisible = false
          document.removeEventListener('click', hideMenu)
          document.removeEventListener('contextmenu', hideMenu)
        }
      }
      setTimeout(() => {
        document.addEventListener('click', hideMenu)
        document.addEventListener('contextmenu', hideMenu)
      }, 100)
    },
    handleSelectionChange(selection) {
      this.selectedRows = selection || []
    },
    getWebURLFromDomainRow(row) {
      if (!row || !row.domain) return null
      const domain = row.domain.trim()
      if (!domain) return null
      // 简单根据 HTTP 状态码判断协议，200 优先 https，其它用 http
      const protocol = row.httpStatus === 200 ? 'https' : 'http'
      return `${protocol}://${domain}`
    },
    // 单个发送到指纹识别
    handleRunFingerprintSingle() {
      if (!this.selectedRow || !this.currentTab) return
      this.rowContextMenuVisible = false
      const url = this.getWebURLFromDomainRow(this.selectedRow)
      if (!url) {
        this.$message.warning('无法获取有效的URL')
        return
      }
      this.callFingerprintModule([url], this.currentTab)
    },
    // 批量发送到指纹识别
    handleRunFingerprintMulti() {
      if (!this.selectedRows.length || !this.currentTab) {
        this.$message.warning('请先选择要发送的行')
        return
      }
      this.rowContextMenuVisible = false
      const urls = []
      for (const row of this.selectedRows) {
        const url = this.getWebURLFromDomainRow(row)
        if (url && row.status === 'valid' && row.httpStatus > 0) {
          urls.push(url)
        }
      }
      if (!urls.length) {
        this.$message.warning('选中的行中没有有效的 Web 服务 URL')
        return
      }
      this.callFingerprintModule(urls, this.currentTab)
    },
    // 单个发送到接口探测（浏览器扫描）
    handleRunAPIDetectionSingle() {
      if (!this.selectedRow || !this.currentTab) return
      this.rowContextMenuVisible = false
      const url = this.getWebURLFromDomainRow(this.selectedRow)
      if (!url) {
        this.$message.warning('无法获取有效的URL')
        return
      }
      this.callBrowserScanModule([url], this.currentTab)
    },
    // 批量发送到接口探测
    handleRunAPIDetectionMulti() {
      if (!this.selectedRows.length || !this.currentTab) {
        this.$message.warning('请先选择要发送的行')
        return
      }
      this.rowContextMenuVisible = false
      const urls = []
      for (const row of this.selectedRows) {
        const url = this.getWebURLFromDomainRow(row)
        if (url && row.status === 'valid' && row.httpStatus > 0) {
          urls.push(url)
        }
      }
      if (!urls.length) {
        this.$message.warning('选中的行中没有有效的 Web 服务 URL')
        return
      }
      this.callBrowserScanModule(urls, this.currentTab)
    },
    // 单个发送到 POC 检测
    handleRunPOCSingle() {
      if (!this.selectedRow || !this.currentTab) return
      this.rowContextMenuVisible = false
      const url = this.getWebURLFromDomainRow(this.selectedRow)
      if (!url) {
        this.$message.warning('无法获取有效的URL')
        return
      }
      this.preparePOCScan([url], this.currentTab)
    },
    // 批量发送到 POC 检测
    handleRunPOCMulti() {
      if (!this.selectedRows.length || !this.currentTab) {
        this.$message.warning('请先选择要发送的行')
        return
      }
      this.rowContextMenuVisible = false
      const urls = []
      for (const row of this.selectedRows) {
        const url = this.getWebURLFromDomainRow(row)
        if (url && row.status === 'valid' && row.httpStatus > 0) {
          urls.push(url)
        }
      }
      if (!urls.length) {
        this.$message.warning('选中的行中没有有效的 Web 服务 URL')
        return
      }
      this.preparePOCScan(urls, this.currentTab)
    },
    // 判断是否是 Web 服务
    isWebServiceFromDomainRow(row) {
      if (!row) return false
      // 如果有 HTTP 状态码且大于 0，说明是 Web 服务
      return row.status === 'valid' && row.httpStatus > 0
    },
    // 获取完整的 Web URL（用于目录扫描）
    getWebURLForDirsearch(row) {
      const url = this.getWebURLFromDomainRow(row)
      if (!url) return null
      // 确保URL以/结尾（用于目录扫描）
      return url.endsWith('/') ? url : `${url}/`
    },
    // 单个发送到目录扫描
    async handleRunDirsearchSingle() {
      if (!this.selectedRow || !this.currentTab) return
      this.rowContextMenuVisible = false
      
      const url = this.getWebURLForDirsearch(this.selectedRow)
      if (!url) {
        this.$message.warning('无法获取有效的URL')
        return
      }
      
      if (!this.isWebServiceFromDomainRow(this.selectedRow)) {
        this.$message.warning('当前域名不是有效的 Web 服务')
        return
      }
      
      const scanConfig = {
        targets: [url],
        method: 'GET',
        threads: 20,
        timeout: 10,
        recursionDepth: 0,
        followRedirects: true,
        customHeaders: {},
        builtinDicts: ['dicc.txt'],
        customDict: '',
        extensions: [],
        fuzz: {
          enabled: false,
          position: 'suffix',
          length: 3,
          charset: ['lower', 'digits'],
          customChars: '',
          generateCount: 1000
        },
        filter: {
          statusCodes: ['404'],
          sizeRange: [0, 1000000],
          keywords: [],
          regex: ''
        }
      }
      
      // 通过事件联动创建标签页并启动扫描
      window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'dirsearch' }))
      setTimeout(() => {
        window.dispatchEvent(new CustomEvent('cyberSabre-dirscan-fill', { detail: scanConfig }))
        window.dispatchEvent(new CustomEvent('cyberSabre-dirscan-request', { detail: scanConfig }))
      }, 100)
      this.$message.success(`已启动目录扫描: ${url}`)
    },
    // 批量发送到目录扫描
    async handleRunDirsearchMulti() {
      if (!this.selectedRows.length || !this.currentTab) {
        this.$message.warning('请先选择要发送的行')
        return
      }
      this.rowContextMenuVisible = false
      
      const urls = []
      for (const row of this.selectedRows) {
        if (this.isWebServiceFromDomainRow(row)) {
          const url = this.getWebURLForDirsearch(row)
          if (url) {
            urls.push(url)
          }
        }
      }
      
      if (urls.length === 0) {
        this.$message.warning('选中的行中没有有效的 Web 服务 URL')
        return
      }
      
      const scanConfig = {
        targets: urls,
        method: 'GET',
        threads: 20,
        timeout: 10,
        recursionDepth: 0,
        followRedirects: true,
        customHeaders: {},
        builtinDicts: ['dicc.txt'],
        customDict: '',
        extensions: [],
        fuzz: {
          enabled: false,
          position: 'suffix',
          length: 3,
          charset: ['lower', 'digits'],
          customChars: '',
          generateCount: 1000
        },
        filter: {
          statusCodes: ['404'],
          sizeRange: [0, 1000000],
          keywords: [],
          regex: ''
        }
      }
      
      // 通过事件联动创建标签页并启动扫描
      window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'dirsearch' }))
      setTimeout(() => {
        window.dispatchEvent(new CustomEvent('cyberSabre-dirscan-fill', { detail: scanConfig }))
        window.dispatchEvent(new CustomEvent('cyberSabre-dirscan-request', { detail: scanConfig }))
      }, 100)
      this.$message.success(`已启动批量目录扫描，共 ${urls.length} 个目标`)
    },
    // 复制当前行
    handleCopyRow() {
      if (!this.selectedRow) return
      this.rowContextMenuVisible = false
      const text = JSON.stringify(this.selectedRow, null, 2)
      navigator.clipboard.writeText(text).then(() => {
        this.$message.success('已复制当前行到剪贴板')
      }).catch(() => {
        this.$message.error('复制失败')
      })
    },
    // 复制选中行
    handleCopySelected() {
      if (!this.selectedRows.length) {
        this.$message.warning('请先选择要复制的行')
        return
      }
      this.rowContextMenuVisible = false
      const text = JSON.stringify(this.selectedRows, null, 2)
      navigator.clipboard.writeText(text).then(() => {
        this.$message.success(`已复制 ${this.selectedRows.length} 行到剪贴板`)
      }).catch(() => {
        this.$message.error('复制失败')
      })
    }
  },
  watch: {
    historyDialog(newVal) {
      if (newVal) {
        this.loadHistory()
      }
    },
    activeTabId() {
      // 标签页切换时重新计算表格高度
      this.$nextTick(() => {
        this.updateTableHeight()
      })
      // 保存数据
      this.saveData()
    },
    collectionTabs: {
      handler() {
        // 深度监听 collectionTabs 变化，自动保存
        this.saveData()
      },
      deep: true
    }
  }
}
</script>

<style lang="scss" scoped>
.domain-collection-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 6px;
  overflow: hidden;
}

// 顶部工具栏
.top-toolbar {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background: #ffffff;
  border-radius: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);

  .toolbar-left {
    display: flex;
    align-items: center;

    .module-title {
      display: flex;
      align-items: center;
      gap: 12px;

      .title-icon {
        font-size: 22px;
        color: #409eff;
      }

      .title-text {
        font-size: 20px;
        font-weight: 600;
        color: #303133;
        letter-spacing: 0.5px;
      }
    }
  }

  .toolbar-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }
}

// 主内容区域
.main-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: transparent;
  border-radius: 6px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 80vh;
}

.main-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  border-radius: 8px;
  overflow: hidden;
}

.main-tabs :deep(.el-tabs__header) {
  margin: 0;
  background: transparent;
  border-radius: 0;
  padding: 0;
  border-bottom: 1px solid #f0f0f0;
  overflow: hidden !important;
  flex-shrink: 0;
}

.main-tabs :deep(.el-tabs__nav-wrap) {
  overflow: hidden !important;
  position: relative;
}

.main-tabs :deep(.el-tabs__nav-wrap::after),
.main-tabs :deep(.el-tabs__nav-wrap::before) {
  display: none !important;
}

.main-tabs :deep(.el-tabs__nav-scroll) {
  overflow-x: auto !important;
  overflow-y: hidden !important;
  scrollbar-width: thin !important;
  scrollbar-color: rgba(0, 0, 0, 0.2) transparent !important;
  width: 100%;
}

.main-tabs :deep(.el-tabs__nav-scroll::-webkit-scrollbar) {
  height: 4px !important;
  width: 100% !important;
}

.main-tabs :deep(.el-tabs__nav-scroll::-webkit-scrollbar-track) {
  background: transparent !important;
}

.main-tabs :deep(.el-tabs__nav-scroll::-webkit-scrollbar-thumb) {
  background: rgba(0, 0, 0, 0.2) !important;
  border-radius: 2px !important;
}

.main-tabs :deep(.el-tabs__nav-scroll::-webkit-scrollbar-thumb:hover) {
  background: rgba(0, 0, 0, 0.3) !important;
}

.main-tabs :deep(.el-tabs__nav) {
  white-space: nowrap;
  display: flex;
  flex-wrap: nowrap;
  transform: translateX(0);
}

.main-tabs :deep(.el-tabs__item) {
  border-radius: 0;
  font-weight: 400;
  transition: all 0.2s ease;
  color: #606266;
  padding: 12px 20px;
  white-space: nowrap;
}

.main-tabs :deep(.el-tabs__item:hover) {
  color: #67C23A;
}

.main-tabs :deep(.el-tabs__item.is-active) {
  color: #67C23A;
  font-weight: 500;
}

.main-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding: 2px;
  background: transparent;
  border-radius: 0;
}

.main-tabs :deep(.el-tab-pane) {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 8px;

  .tab-name {
    font-size: 14px;
    font-weight: 400;
  }

  .tab-badge {
    margin-left: 4px;
  }

  .tab-name {
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

// 标签页内容
.tab-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
  height: 100%;
}

// 顶部区域（输入和配置横向布局）
.top-section {
  flex-shrink: 0;
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

// 卡片样式
.card {
  background: #ffffff;
  border-radius: 12px;
  border: 1px solid #e4e7ed;
  overflow: auto;
  transition: all 0.3s ease;

  &:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    background: #fafbfc;
    border-bottom: 1px solid #e4e7ed;

    .card-title-group {
      display: flex;
      align-items: center;
      gap: 12px;

      .card-title {
        font-size: 16px;
        font-weight: 600;
        color: #303133;
      }

      .card-badges {
        display: flex;
        gap: 8px;
      }
    }

    .card-actions {
      display: flex;
      gap: 8px;
    }
  }

  .card-body {
    padding: 20px;
  }
}

// 输入卡片
.input-card {
  flex: 1;
  min-width: 0;
  height: 180px;
  display: flex;
  flex-direction: column;

  .card-header {
    padding: 12px 16px;
    flex-shrink: 0;
  }

  .card-body {
    padding: 12px 16px;
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .input-wrapper {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .action-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;

    .progress-section {
      flex: 1;
      display: flex;
      flex-direction: column;
      gap: 4px;

      .progress-label {
        font-size: 12px;
        color: #606266;
        font-weight: 500;
      }
    }

    .action-buttons {
      display: flex;
      gap: 8px;
    }
  }
}

.domain-textarea {
  :deep(.el-textarea__inner) {
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 13px;
    line-height: 1.6;
    border-radius: 8px;
  }
}

// 配置卡片
.config-card {
  flex: 1;
  min-width: 0;
  height: 180px;
  display: flex;
  flex-direction: column;

  .card-header {
    padding: 8px 12px;
    flex-shrink: 0;
  }

  .card-body {
    padding: 8px 12px;
    flex: 1;
    min-height: 0;
    overflow: auto;
  }

  // 减小控件尺寸
  :deep(.el-input-number) {
    .el-input__inner {
      padding: 0 6px;
      height: 28px;
      line-height: 28px;
    }
  }

  :deep(.el-select) {
    .el-input__inner {
      padding: 0 6px;
      height: 28px;
      line-height: 28px;
    }
  }

  :deep(.el-input) {
    .el-input__inner {
      padding: 0 6px;
      height: 28px;
      line-height: 28px;
    }
  }
}

// 高级配置卡片
.advanced-config-card {
  flex: 1;
  min-width: 0;
  height: 180px;
  display: flex;
  flex-direction: column;

  .card-header {
    padding: 8px 12px;
    flex-shrink: 0;
  }

  .card-body {
    padding: 8px 12px;
    flex: 1;
    min-height: 0;
    overflow: auto;
  }

  // 减小按钮padding
  :deep(.el-button) {
    padding: 4px 8px;
    height: 28px;
    font-size: 12px;
  }

  :deep(.el-radio-button) {
    .el-radio-button__inner {
      padding: 6px 10px;
      height: 28px;
      line-height: 16px;
      font-size: 12px;
    }
  }

  :deep(.el-select) {
    .el-input__inner {
      padding: 0 6px;
      height: 28px;
      line-height: 28px;
    }
  }
}

.collapse-title {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;

  .el-icon {
    font-size: 18px;
    color: #409eff;
  }
}

.config-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 6px;
}

.config-item {
  display: flex;
  flex-direction: column;
  gap: 4px;

  label {
    font-size: 12px;
    color: #606266;
    font-weight: 500;
    line-height: 1.2;
  }

  &.full-width {
    grid-column: 1 / -1;
  }
}

.config-switches {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 6px;
  padding: 6px;
  background: #fafbfc;
  border-radius: 6px;

  .switch-item {
    display: flex;
    justify-content: space-between;
    align-items: center;

    label {
      font-size: 12px;
      color: #606266;
      font-weight: 500;
      line-height: 1.2;
    }
  }
}

.advanced-config {
  :deep(.el-collapse-item__header) {
    font-size: 14px;
    font-weight: 600;
    padding: 12px 0;
  }

  :deep(.el-collapse-item__content) {
    padding: 16px 0;
  }
}

.advanced-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.switch-with-button {
  display: flex;
  align-items: center;
  gap: 8px;

  label {
    flex-shrink: 0;
    font-size: 12px;
  }

  // 减小按钮padding
  :deep(.el-button) {
    padding: 4px 8px;
    font-size: 12px;
    height: 28px;
  }
}

.config-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 8px;
  margin-top: 4px;
}

// 结果卡片
.results-card {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  .card-header {
    flex-shrink: 0;
  }

  .card-body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: 0;
    overflow: hidden;
    
    .results-table {
      flex: 1;
      min-height: 0;
    }
    
    .pagination-wrapper {
      flex-shrink: 0;
    }
  }
}

.results-table {
  flex: 1;
  min-height: 0;
  width: 100%;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #f0f0f0;

  :deep(.el-table__header) {
    background: #f5f7fa;
  }

  :deep(.el-table__header th) {
    background: transparent;
    color: #2c3e50;
    font-weight: 500;
    border-bottom: 1px solid #e4e7ed;
    padding: 12px 0;
  }
  
  :deep(.el-table__body-wrapper) {
    min-height: 300px;
    max-height: none;
    overflow-y: auto;
  }

  :deep(.el-table__body tr:hover > td) {
    background: #f0f9ff !important;
  }

  :deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
    background: #fafbfc;
  }
  
  :deep(.el-table__empty-block) {
    min-height: 300px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
  }
  
  :deep(.el-table__body) {
    width: 100% !important;
  }
  
  :deep(.el-table__header-wrapper) {
    width: 100% !important;
  }
  
  :deep(.el-table) {
    width: 100% !important;
    border-radius: 6px;
    overflow: hidden;
  }
  
  :deep(.el-table__inner-wrapper) {
    width: 100% !important;
  }
}

.pagination-wrapper {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 5px 10px;
  margin-top: 2px;
  border-top: 1px solid #f0f0f0;
  background: transparent;
  padding-left: 0;
  padding-right: 0;
}

.pagination-info {
  font-size: 14px;
  color: var(--el-text-color-regular);
}

.pagination-page-info {
  font-size: 14px;
  color: var(--el-text-color-regular);
}

.pagination-wrapper :deep(.el-pagination) {
  flex: 1;
  justify-content: center;
}

.pagination-wrapper :deep(.el-pagination .el-pagination__sizes),
.pagination-wrapper :deep(.el-pagination .el-pagination__total),
.pagination-wrapper :deep(.el-pagination .el-pagination__jump) {
  font-size: 12px;
}

.pagination-wrapper :deep(.el-pagination .btn-prev),
.pagination-wrapper :deep(.el-pagination .btn-next),
.pagination-wrapper :deep(.el-pagination .number) {
  min-width: 32px;
  height: 32px;
  line-height: 32px;
}

.domain-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  
  &.clickable {
    cursor: pointer;
    color: #409eff;
    transition: color 0.2s;
    
    &:hover {
      color: #66b1ff;
      text-decoration: underline;
    }
  }
}

.ips-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.source-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  justify-content: center;
  align-items: center;
}

.empty-text {
  color: #c0c4cc;
}

.table-cell-text {
  overflow: auto;
  text-overflow: ellipsis;
  white-space: nowrap;
}


// API配置对话框
.api-config-content {
  max-height: 500px;
  overflow-y: auto;
}

.api-config-item {
  margin-bottom: 20px;
  padding: 20px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #fafbfc;
  transition: all 0.3s ease;

  &:last-child {
    margin-bottom: 0;
  }

  &:hover {
    border-color: #409eff;
    box-shadow: 0 2px 8px rgba(64, 158, 255, 0.1);
  }
}

.api-config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;

  .api-name {
    font-weight: 600;
    color: #303133;
    font-size: 15px;
  }
}

.api-config-body {
  display: flex;
  align-items: center;
  gap: 12px;

  .api-key-input {
    flex: 1;
  }

  .fofa-input-row {
    display: flex;
    align-items: center;
    gap: 10px;
    flex: 1;
    white-space: nowrap;
  }

  .fofa-email-input,
  .fofa-key-input,
  .fofa-baseurl-input {
    min-width: 0;
  }

  .eye-icon {
    cursor: pointer;
    color: #909399;
    transition: color 0.3s;

    &:hover {
      color: #409eff;
    }
  }

  .api-status {
    min-width: 60px;
    text-align: center;
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

// 任务管理对话框样式
:deep(.history-dialog .el-overlay-dialog) {
  display: flex !important;
  align-items: flex-start !important;
  justify-content: center !important;
  width: 100% !important;
}

:deep(.history-dialog .el-dialog) {
  max-width: 60vw !important;
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
}

.history-pagination {
  display: flex;
  justify-content: center;
  padding: 16px 0;
  border-top: 1px solid #e4e7ed;
}

.task-name-cell {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 标签页右键菜单样式 */
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
  color: #67C23A;
}

/* 表格行右键菜单样式 */
.context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  z-index: 3000;
  min-width: 180px;
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
}

.context-menu-item .el-icon {
  font-size: 16px;
}

.context-menu-divider {
  height: 1px;
  background: #e4e7ed;
  margin: 4px 0;
}

.mr-5px {
  margin-right: 5px;
}

// 响应式
@media (max-width: 1200px) {
  .config-grid {
    grid-template-columns: 1fr;
  }

  .config-switches {
    grid-template-columns: repeat(2, 1fr);
  }
}

/* 标签页右键菜单样式 */
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
  color: #67C23A;
}
</style>
