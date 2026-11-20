<template>
  <div class="space-mapping-container">
    <!-- 顶部工具栏 -->
    <div class="top-toolbar">
      <div class="module-title">
        <el-icon class="title-icon"><Search /></el-icon>
        <span class="title-text">空间测绘</span>
      </div>
      <div class="toolbar-actions">
        <el-button 
          :icon="Document" 
          type="info" 
          plain
          size="default"
          @click="showSyntaxDialog = true"
        >
          查询语法
        </el-button>
        <el-button 
          :icon="Setting" 
          type="info" 
          plain
          size="default"
          @click="showSettingsDialog = true"
        >
          查询设置
        </el-button>
        <el-button 
          :icon="Setting" 
          type="info" 
          plain
          size="default"
          @click="showApiConfigDialog = true"
        >
          配置Api
        </el-button>
      </div>
    </div>

    <!-- 搜索栏区域 -->
    <div class="search-bar-card">
      <div class="search-bar">
        <div class="search-left">
          <el-select 
            v-model="selectedPlatform" 
            placeholder="选择平台" 
            class="platform-select"
            size="default"
            @change="handlePlatformChange"
          >
            <el-option 
              v-for="platform in enabledPlatformList" 
              :key="platform.key"
              :label="platform.label" 
              :value="platform.key"
            />
            <template #empty>
              <span>暂无可用平台</span>
            </template>
          </el-select>

          <el-input
            v-model="queryText"
            placeholder='输入查询内容，如: domain:"baidu.com"'
            class="query-input"
            clearable
            @keyup.enter="handleQuery"
          >
            <template #append>
              <el-button 
                :icon="Document" 
                @click="showMultiLineInput = true"
                title="多行输入"
              />
            </template>
          </el-input>
        </div>

        <div class="search-right">
          <div v-if="currentQueryTab && currentQueryTab.totalResults" class="count-display">
            <span class="count-label">结果数：</span>
            <span class="count-value">{{ currentQueryTab.totalResults }}</span>
          </div>

          <el-checkbox 
            v-model="latestData" 
            class="latest-data-checkbox"
            size="default"
          >
            最新数据
          </el-checkbox>

          <el-button 
            type="primary" 
            :icon="Search" 
            @click="handleQuery"
            :loading="queryLoading"
            :disabled="queryLoading"
            size="default"
            class="search-btn"
          >
            搜索
          </el-button>
          <el-button 
            v-if="queryLoading"
            type="danger" 
            :icon="Delete" 
            @click="handleCancelQuery"
            size="default"
            class="cancel-btn"
          >
            取消查询
          </el-button>
          <el-dropdown 
            @command="handleExportCommand"
            :disabled="!currentQueryTab || !currentQueryTab.totalResults"
          >
            <el-button 
              type="success" 
              :icon="Download" 
              size="default"
            >
              导出
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="excel">导出为 Excel</el-dropdown-item>
                <el-dropdown-item command="json">导出为 JSON</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>

    <!-- 主要内容区域 -->
    <div class="main-content">
      <CustomTabs>
        <!-- 空状态 -->
        <div v-if="queryTabs.length === 0" class="empty-tabs-container">
          <el-empty description="暂无查询任务，开始您的第一次查询吧！">
            <el-button type="primary" :icon="Document" @click="showSyntaxDialog = true" size="large">
              查看查询语法
            </el-button>
          </el-empty>
        </div>

        <!-- 标签页 -->
        <el-tabs 
          v-else
          v-model="activeTab" 
          type="border-card" 
          class="main-tabs"
          closable
          @tab-remove="handleTabRemove"
          @contextmenu.prevent="handleTabsContextMenu"
        >
          <!-- 查询结果标签页 -->
          <el-tab-pane
            v-for="tab in queryTabs"
            :key="tab.id"
            :name="tab.id"
          >
            <template #label>
              <span class="tab-label">
                <span class="tab-name">{{ tab.name }}</span>
                <el-badge
                  v-if="tab.totalResults > 0"
                  :value="tab.totalResults"
                  :max="9999"
                  class="tab-badge"
                />
              </span>
            </template>

            <!-- 查询中状态 -->
            <div v-if="tab.querying" class="querying-container">
              <el-icon class="is-loading querying-icon"><Loading /></el-icon>
              <span class="querying-text">正在查询中...</span>
            </div>

            <!-- 查询结果表格（单平台，不再显示平台标签） -->
            <div v-else class="results-container">
              <el-table 
                :data="getPagedResults(tab, Object.keys(tab.queryResults)[0] || '')" 
                border 
                stripe
                size="default"
                class="results-table"
                :height="tableHeight"
                :key="`table-${tab.id}-${tab.totalResults}`"
                @row-contextmenu="handleRowContextMenu"
                @selection-change="handleSelectionChange"
              >
                    <el-table-column type="selection" width="55" align="center" />
                    <el-table-column type="index" label="ID" width="60" align="center" />
                    <el-table-column prop="url" label="URL" min-width="200">
                      <template #default="{ row }">
                        <el-tooltip
                          v-if="row.url && row.url.length > 40"
                          :content="row.url"
                          placement="top"
                          effect="dark"
                          :popper-options="{ modifiers: [{ name: 'offset', options: { offset: [0, 8] } }] }"
                        >
                          <el-link 
                            type="primary" 
                            @click="openURL(row.url)"
                            class="url-link"
                          >
                            {{ row.url }}
                          </el-link>
                        </el-tooltip>
                        <el-link 
                          v-else-if="row.url"
                          type="primary" 
                          @click="openURL(row.url)"
                          class="url-link"
                        >
                          {{ row.url }}
                        </el-link>
                        <span v-else>-</span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="ip" label="IP" width="150">
                      <template #default="{ row }">
                        <el-tooltip
                          v-if="row.ip && row.ip.length > 15"
                          :content="row.ip"
                          placement="top"
                          effect="dark"
                          :popper-options="{ modifiers: [{ name: 'offset', options: { offset: [0, 8] } }] }"
                        >
                          <span>{{ row.ip }}</span>
                        </el-tooltip>
                        <span v-else>{{ row.ip || '-' }}</span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="port" label="端口" width="80" align="center" />
                    <el-table-column prop="domain" label="域名" min-width="150">
                      <template #default="{ row }">
                        <el-tooltip
                          v-if="row.domain && row.domain.length > 20"
                          :content="row.domain"
                          placement="top"
                          effect="dark"
                          :popper-options="{ modifiers: [{ name: 'offset', options: { offset: [0, 8] } }] }"
                        >
                          <span>{{ row.domain }}</span>
                        </el-tooltip>
                        <span v-else>{{ row.domain || '-' }}</span>
                      </template>
                    </el-table-column>
                    <el-table-column prop="protocol" label="协议" width="80" align="center">
                      <template #default="{ row }">
                        <el-tag size="small" :type="getProtocolTagType(row.url)">
                          {{ getProtocol(row.url) }}
                        </el-tag>
                      </template>
                    </el-table-column>
                    <el-table-column prop="title" label="标题" min-width="200">
                      <template #default="{ row }">
                        <el-tooltip
                          v-if="row.title && row.title.length > 30"
                          :content="row.title"
                          placement="top"
                          effect="dark"
                          :popper-options="{ modifiers: [{ name: 'offset', options: { offset: [0, 8] } }] }"
                        >
                          <span>{{ row.title }}</span>
                        </el-tooltip>
                        <span v-else>{{ row.title || '-' }}</span>
                      </template>
                    </el-table-column>
              </el-table>

              <!-- 分页 -->
              <div class="pagination-wrapper" v-if="Object.keys(tab.queryResults).length > 0">
                <div class="pagination-info">
                  <span>共 {{ (tab.queryResults[Object.keys(tab.queryResults)[0]] || []).length }} 条数据</span>
                </div>
                <el-pagination
                  v-model:current-page="tab.platformPagination[Object.keys(tab.queryResults)[0] || ''].currentPage"
                  v-model:page-size="tab.platformPagination[Object.keys(tab.queryResults)[0] || ''].pageSize"
                  :page-sizes="[10, 20, 50, 100, 200]"
                  :total="(tab.queryResults[Object.keys(tab.queryResults)[0]] || []).length"
                  layout="prev, pager, next, sizes, jumper"
                  background
                  @size-change="(size) => handlePlatformSizeChange(size, Object.keys(tab.queryResults)[0] || '', tab)"
                  @current-change="(page) => handlePlatformCurrentChange(page, Object.keys(tab.queryResults)[0] || '', tab)"
                />
                <div class="pagination-page-info">
                  <span>第 {{ tab.platformPagination[Object.keys(tab.queryResults)[0] || ''].currentPage }} 页 / 共 {{ Math.ceil((tab.queryResults[Object.keys(tab.queryResults)[0]] || []).length / tab.platformPagination[Object.keys(tab.queryResults)[0] || ''].pageSize) }} 页</span>
                </div>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </CustomTabs>
    </div>

    <!-- API Key配置对话框 -->
    <el-dialog
      v-model="showApiConfigDialog"
      title="API Key配置 (与域名查询接口共用所有key)"
      width="800px"
      :close-on-click-modal="false"
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

    <!-- 查询设置对话框 -->
    <el-dialog
      v-model="showSettingsDialog"
      title="查询设置"
      width="500px"
      :close-on-click-modal="false"
    >
      <div class="settings-content">
        <el-form :model="apiSettings" label-width="100px">
          <el-form-item label="页数上限">
            <el-input-number v-model="apiSettings.maxPages" :min="1" :max="100" />
            <span class="form-item-tip">每次查询的最大页数</span>
          </el-form-item>
          <el-form-item label="请求延迟">
            <el-input-number v-model="apiSettings.delay" :min="0" :max="10" :step="0.1" />
            <span class="form-item-tip">秒（每次请求之间的延迟时间）</span>
          </el-form-item>
          <el-form-item label="超时">
            <el-input-number v-model="apiSettings.timeout" :min="5" :max="300" />
            <span class="form-item-tip">秒（请求超时时间）</span>
          </el-form-item>
          <el-form-item label="开启代理">
            <el-switch v-model="apiSettings.enableProxy" />
            <span class="form-item-tip">使用系统全局代理设置</span>
          </el-form-item>
          <el-form-item label="每页条数">
            <el-input-number v-model="apiSettings.pageSize" :min="10" :max="10000" :step="10" />
            <span class="form-item-tip">每次查询返回的最大条数</span>
            <el-alert
              type="info"
              :closable="false"
              show-icon
              style="margin-top: 8px;"
            >
              <template #title>
                <div style="font-size: 12px; line-height: 1.5;">
                  <div>平台限制说明：</div>
                  <div>• FOFA官方/第三方：最多 10000 条/页</div>
                  <div>• Hunter：最多 100 条/页（超过将自动限制）</div>
                  <div>• Quake：最多 100 条/页（超过将自动限制）</div>
                </div>
              </template>
            </el-alert>
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showSettingsDialog = false">取消</el-button>
          <el-button type="primary" @click="handleSaveSettings">保存</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 多行输入对话框 -->
    <el-dialog
      v-model="showMultiLineInput"
      title="多行输入"
      width="600px"
    >
      <el-input
        v-model="multiLineText"
        type="textarea"
        :rows="10"
        placeholder="每行一个查询项"
      />
      <template #footer>
        <el-button @click="showMultiLineInput = false">取消</el-button>
        <el-button type="primary" @click="handleMultiLineConfirm">确定</el-button>
      </template>
    </el-dialog>

    <!-- 查询语法对话框 -->
    <el-dialog
      v-model="showSyntaxDialog"
      title="查询语法参考"
      width="900px"
      class="syntax-dialog"
    >
      <el-table :data="syntaxTable" border size="default" class="syntax-table" max-height="500">
        <el-table-column prop="name" label="名称" width="120" align="center" />
        <el-table-column prop="0zone" label="0.zone" min-width="200">
          <template #default="{ row }">
            <el-tooltip
              :content="row['0zone']"
              placement="top"
              effect="dark"
              :disabled="!row['0zone'] || row['0zone'].length <= 30"
            >
              <span class="syntax-cell" @click="copySyntax(row['0zone'])">
                {{ row['0zone'] }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="fofa" label="FOFA官方" min-width="200">
          <template #default="{ row }">
            <el-tooltip
              :content="row.fofa"
              placement="top"
              effect="dark"
              :disabled="!row.fofa || row.fofa.length <= 30"
            >
              <span class="syntax-cell" @click="copySyntax(row.fofa)">
                {{ row.fofa }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="fofa.red" label="FOFA.RED" min-width="200">
          <template #default="{ row }">
            <el-tooltip
              :content="row['fofa.red']"
              placement="top"
              effect="dark"
              :disabled="!row['fofa.red'] || row['fofa.red'].length <= 30"
            >
              <span class="syntax-cell" @click="copySyntax(row['fofa.red'])">
                {{ row['fofa.red'] }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="hunter" label="Hunter" min-width="200">
          <template #default="{ row }">
            <el-tooltip
              :content="row.hunter"
              placement="top"
              effect="dark"
              :disabled="!row.hunter || row.hunter.length <= 30"
            >
              <span class="syntax-cell" @click="copySyntax(row.hunter)">
                {{ row.hunter }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="quake" label="Quake" min-width="200">
          <template #default="{ row }">
            <el-tooltip
              :content="row.quake"
              placement="top"
              effect="dark"
              :disabled="!row.quake || row.quake.length <= 30"
            >
              <span class="syntax-cell" @click="copySyntax(row.quake)">
                {{ row.quake }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 标签页右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
    <Teleport to="body">
      <ul 
        class="tabs-context-menu" 
        v-if="tabsContextMenuVisible" 
        :style="tabsContextMenuStyle"
      >
        <li @click="handleCloseCurrentTab">关闭当前</li>
        <li @click="handleCloseAllTabs">全部关闭</li>
      </ul>
    </Teleport>

    <!-- 漏洞扫描对话框 -->
    <el-dialog
      v-model="vulnScanDialogVisible"
      title="创建漏洞扫描任务"
      width="600px"
      destroy-on-close
    >
      <el-form label-width="120px">
        <el-form-item label="任务名称:">
          <el-input v-model="vulnScanTaskName" placeholder="可选，默认自动生成" />
        </el-form-item>
        <el-form-item label="目标数量:">
          <span>{{ vulnScanTargets.length }} 个URL</span>
        </el-form-item>
        <el-form-item label="扫描模式:">
          <el-radio-group v-model="vulnScanMode">
            <el-radio label="full">全量漏洞扫描</el-radio>
            <el-radio label="fingerprint">指纹漏洞扫描</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="指定指纹:" v-if="vulnScanMode === 'fingerprint'">
          <el-select
            v-model="vulnScanCustomTags"
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
          <el-checkbox v-model="vulnScanSkipNucleiWithoutTags">无指纹跳过</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="vulnScanDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmVulnScan">创建任务</el-button>
      </template>
    </el-dialog>

    <!-- 表格行右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
    <Teleport to="body">
      <div
        v-if="rowContextMenuVisible"
        class="context-menu"
        :style="rowContextMenuStyle"
        @click.stop
      >
        <div v-if="isWebService(selectedRow)" class="context-menu-item" @click="handleRunFingerprint">
          <el-icon><View /></el-icon>
          <span>指纹识别</span>
        </div>
        <div v-if="isWebService(selectedRow)" class="context-menu-item" @click="handleRunDirsearch">
          <el-icon><View /></el-icon>
          <span>目录扫描</span>
        </div>
        <div v-if="isWebService(selectedRow)" class="context-menu-item" @click="handleRunVulnScan">
          <el-icon><View /></el-icon>
          <span>漏洞扫描</span>
        </div>
        <div v-if="isWebService(selectedRow)" class="context-menu-item" @click="handleRunAPIDetectionSingle">
          <el-icon><View /></el-icon>
          <span>接口探测（当前）</span>
        </div>
        <div v-if="isCrackableProtocol(selectedRow)" class="context-menu-item" @click="handleRunCrack">
          <el-icon><View /></el-icon>
          <span>密码爆破</span>
        </div>
        <div class="context-menu-divider" v-if="isWebService(selectedRow) || isCrackableProtocol(selectedRow)"></div>
        <template v-if="selectedRows.length > 0">
          <div v-if="selectedRows.some(row => isWebService(row))" class="context-menu-item" @click="handleRunFingerprintMulti">
            <el-icon><View /></el-icon>
            <span>批量指纹识别 ({{ selectedRows.filter(row => isWebService(row)).length }})</span>
          </div>
          <div v-if="selectedRows.some(row => isWebService(row))" class="context-menu-item" @click="handleRunDirsearchMulti">
            <el-icon><View /></el-icon>
            <span>批量目录扫描 ({{ selectedRows.filter(row => isWebService(row)).length }})</span>
          </div>
          <div v-if="selectedRows.some(row => isWebService(row))" class="context-menu-item" @click="handleRunVulnScanMulti">
            <el-icon><View /></el-icon>
            <span>批量漏洞扫描 ({{ selectedRows.filter(row => isWebService(row)).length }})</span>
          </div>
          <div v-if="selectedRows.some(row => isWebService(row))" class="context-menu-item" @click="handleRunAPIDetectionMulti">
            <el-icon><View /></el-icon>
            <span>批量接口探测 ({{ selectedRows.filter(row => isWebService(row)).length }})</span>
          </div>
          <div v-if="selectedRows.some(row => isCrackableProtocol(row))" class="context-menu-item" @click="handleRunCrackMulti">
            <el-icon><View /></el-icon>
            <span>批量密码爆破 ({{ selectedRows.filter(row => isCrackableProtocol(row)).length }})</span>
          </div>
          <div class="context-menu-divider" v-if="selectedRows.some(row => isWebService(row)) || selectedRows.some(row => isCrackableProtocol(row))"></div>
        </template>
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
import { markRaw } from 'vue';
import { Search, Setting, Document, Download, View, Hide, Loading, ArrowDown, CopyDocument, Delete } from '@element-plus/icons-vue';
import { ElMessage } from 'element-plus';
import axios from 'axios';
import { GetPortConfig } from '../../../wailsjs/go/controller/Server';
import { BrowserOpenURL } from '../../../wailsjs/runtime';
import { SaveFile, GetAppPath, ReadFile } from '../../../wailsjs/go/controller/Note';
import CustomTabs from '../../components/CustomTabs.vue';
import * as XLSX from 'xlsx';
import { StartFingerprintScan } from '../../../wailsjs/go/finger/FingerprintController';
import { StartDirectoryScan } from '../../../wailsjs/go/dirsearch/DirsearchController';
import { StartPocScan } from '../../../wailsjs/go/webscan/PocController';
import { GetPocConfig } from '../../../wailsjs/go/webscan/PocConfigController';
import { GetGlobalProxy } from '../../../wailsjs/go/proxy/ProxyManager';
import { NewCrackScanenr } from '../../../wailsjs/go/crack/CrackController';
import { CreateTask as CreateBrowserScanTask } from '../../../wailsjs/go/browserScan/BrowserScan';

export default {
  name: 'SpaceMappingView',
  components: {
    Search,
    Setting,
    Document,
    Download,
    View,
    Hide,
    CustomTabs,
    CopyDocument,
    Delete,
  },
  data() {
    return {
      // 图标组件
      Search: markRaw(Search),
      Setting: markRaw(Setting),
      Document: markRaw(Document),
      Download: markRaw(Download),
      View: markRaw(View),
      Hide: markRaw(Hide),
      Loading: markRaw(Loading),
      ArrowDown: markRaw(ArrowDown),
      
      // 查询相关
      queryText: '',
      queryLoading: false,
      selectedPlatform: '', // 当前选择的平台（单选）
      latestData: false, // 最新数据选项
      currentQueryId: null, // 当前查询的ID，用于取消查询
      isQueryCanceled: false, // 标记查询是否被取消
      
      // 平台列表
      platformList: [
        { key: '0zone', label: '0.zone' },
        { key: 'fofa', label: 'FOFA官方' },
        { key: 'otherfofa', label: 'FOFA第三方' },
        { key: 'hunter', label: 'Hunter' },
        { key: 'quake', label: 'Quake' },
      ],
      
      // 查询结果标签页
      queryTabs: [],
      activeTab: '',
      tabIdCounter: 0,
      
      // 表格高度
      tableHeight: 600,
      
      // API配置
      showApiConfigDialog: false,
      showSettingsDialog: false,
      showSyntaxDialog: false,
      apiConfigs: {
        '0zone': { key: '', enabled: false, showKey: false, status: 'unknown' },
        fofa: { email: '', key: '', enabled: false, showKey: false, status: 'unknown' },
        otherfofa: { key: '', enabled: false, showKey: false, status: 'unknown' },
        hunter: { key: '', enabled: false, showKey: false, status: 'unknown' },
        quake: { key: '', enabled: false, showKey: false, status: 'unknown' },
      },
      apiSettings: {
        maxPages: 5,
        delay: 2,
        timeout: 30,
        enableProxy: false,
        pageSize: 100,
        // FOFA 第三方接口（默认使用 FOFA.RED）
        fofaThirdBaseURL: 'https://fofa.red',
      },
      
      // 查询语法表
      syntaxTable: [
        {
          name: 'IP',
          '0zone': 'ip=1.1.1.1',
          fofa: 'ip="1.1.1.1"',
          'fofa.red': 'ip="1.1.1.1"',
          hunter: 'ip="1.1.1.1"',
          quake: 'ip:"1.1.1.1"',
        },
        {
          name: 'Domain',
          '0zone': 'url=https://baidu.com',
          fofa: 'domain="baidu.com"',
          'fofa.red': 'domain="baidu.com"',
          hunter: 'domain="baidu.com"',
          quake: 'domain:"baidu.com"',
        },
        {
          name: 'Port',
          '0zone': 'port=80',
          fofa: 'port="80"',
          'fofa.red': 'port="80"',
          hunter: 'ip.port="80"',
          quake: 'port:"80"',
        },
        {
          name: 'Title',
          '0zone': 'title=后台管理',
          fofa: 'title="标题"',
          'fofa.red': 'title="标题"',
          hunter: 'web.title="标题"',
          quake: 'title:"标题"',
        },
        {
          name: 'Body',
          '0zone': 'banner==HTTP/1.1 302 Found',
          fofa: 'body="内容"',
          'fofa.red': 'body="内容"',
          hunter: 'web.body="内容"',
          quake: 'response:"内容"',
        },
        {
          name: 'OS',
          '0zone': 'os=Linux',
          fofa: 'os="centos"',
          'fofa.red': 'os="centos"',
          hunter: 'ip.os="centos"',
          quake: 'os:"centos"',
        },
        {
          name: 'App',
          '0zone': 'component=nginx',
          fofa: 'app="Nginx"',
          'fofa.red': 'app="Nginx"',
          hunter: 'app.name="Nginx"',
          quake: 'app:"Nginx"',
        },
        {
          name: 'Protocol',
          '0zone': 'service=http',
          fofa: 'protocol="http"',
          'fofa.red': 'protocol="http"',
          hunter: 'protocol="http"',
          quake: 'service:"ssh"',
        },
        {
          name: 'ICP',
          '0zone': '<不支持>',
          fofa: 'icp="备案号"',
          'fofa.red': 'icp="备案号"',
          hunter: 'icp.number="备案号"',
          quake: 'icp:"备案号"',
        },
        {
          name: 'Country',
          '0zone': 'country=中国',
          fofa: 'country="中国"',
          'fofa.red': 'country="中国"',
          hunter: 'ip.country="中国"',
          quake: 'country_cn:"中国"',
        },
        {
          name: 'Region',
          '0zone': 'province=山东',
          fofa: 'region="山东"',
          'fofa.red': 'region="山东"',
          hunter: 'ip.province="山东"',
          quake: 'province:"山东"',
        },
        {
          name: 'City',
          '0zone': 'city=济南',
          fofa: 'city="济南"',
          'fofa.red': 'city="济南"',
          hunter: 'ip.city="济南"',
          quake: 'city:"济南"',
        },
        {
          name: 'Icon',
          '0zone': '<不支持>',
          fofa: 'icon_hash="mmh3"',
          'fofa.red': 'icon_hash="mmh3"',
          hunter: 'web.icon="MD5"',
          quake: 'favicon:"MD5"',
        },
      ],
      
      // 多行输入
      showMultiLineInput: false,
      multiLineText: '',
      
      // API客户端
      apiClient: null,
      portConfig: null,
      
      // 右键菜单相关
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: '',
      
      // 表格行右键菜单相关
      rowContextMenuVisible: false,
      rowContextMenuStyle: { left: '0px', top: '0px' },
      selectedRow: null,
      selectedRows: [], // 多选的行数据
      
      // 漏洞扫描对话框相关
      vulnScanDialogVisible: false,
      vulnScanMode: 'full', // 'full' 全量扫描, 'fingerprint' 指纹扫描
      vulnScanTargets: [],
      vulnScanTaskName: '',
      vulnScanCustomTags: [],
      vulnScanCustomTemplate: [],
      vulnScanSkipNucleiWithoutTags: false,
      
      // 数据文件路径
      tabsDataFilePath: null,
    };
  },
  computed: {
    currentQueryTab() {
      return this.queryTabs.find(tab => tab.id === this.activeTab) || null;
    },
    platformEnabledMap() {
      const map = {};
      Object.keys(this.apiConfigs).forEach(key => {
        const config = this.apiConfigs[key];
        map[key] = config && config.key && config.key.trim() !== '';
      });
      return map;
    },
    enabledPlatformList() {
      return this.platformList.filter(p => this.platformEnabledMap[p.key]);
    },
    tabsContextMenuStyle() {
      const { x, y } = this.tabsContextMenuPosition;
      return {
        left: `${x}px`,
        top: `${y}px`,
      };
    },
  },
  async mounted() {
    await this.initApiClient();
    await this.loadApiConfig();
    this.calculateTableHeight();
    window.addEventListener('resize', this.calculateTableHeight);
    // 从本地存储加载标签页数据
    this.loadTabsFromStorage();
  },
  // keep-alive 激活时
  activated() {
    // 从本地存储恢复数据
    this.loadTabsFromStorage();
    // 重新计算表格高度
    this.$nextTick(() => {
      this.calculateTableHeight();
    });
  },
  // keep-alive 失活时
  deactivated() {
    // 保存数据到本地存储
    this.saveTabsToStorage();
  },
  beforeUnmount() {
    window.removeEventListener('resize', this.calculateTableHeight);
    // 保存数据到本地存储
    this.saveTabsToStorage();
  },
  methods: {
    calculateTableHeight() {
      // 计算表格高度，留出分页和其他元素的空间
      const container = this.$el;
      if (container) {
        const height = container.clientHeight;
        const searchHeight = container.querySelector('.search-row')?.clientHeight || 0;
        const toolbarHeight = container.querySelector('.top-toolbar')?.clientHeight || 0;
        const queryTagHeight = container.querySelector('.query-tag-row')?.clientHeight || 0;
        const tabsHeight = 50; // 标签页高度
        const paginationHeight = 60; // 分页高度
        const padding = 40; // 其他padding和margin
        
        this.tableHeight = height - searchHeight - toolbarHeight - queryTagHeight - tabsHeight - paginationHeight - padding;
        if (this.tableHeight < 300) {
          this.tableHeight = 300;
        }
      }
    },
    async initApiClient() {
      try {
        this.portConfig = await GetPortConfig();
        const port = this.portConfig?.mainPort || 52867;
        const baseURL = `http://127.0.0.1:${port}`;
        // 使用配置的超时时间，默认30秒，转换为毫秒
        const timeout = (this.apiSettings?.timeout || 30) * 1000;
        this.apiClient = axios.create({
          baseURL: baseURL,
          timeout: timeout,
        });
        console.log('API客户端初始化成功，端口:', port, '超时:', timeout, 'ms');
      } catch (error) {
        console.error('初始化API客户端失败:', error);
        ElMessage.error('初始化API客户端失败');
      }
    },
    
    // 更新API客户端的超时时间
    updateApiClientTimeout() {
      if (this.apiClient) {
        const timeout = (this.apiSettings?.timeout || 30) * 1000;
        this.apiClient.defaults.timeout = timeout;
        console.log('API客户端超时时间已更新为:', timeout, 'ms');
      }
    },
    
    async loadApiConfig() {
      try {
        if (!this.apiClient) return;
        const response = await this.apiClient.get('/api/info-collection/config');
        if (response.data && response.data.success) {
          const config = response.data.data;
          if (config.apiKeys) {
            Object.keys(config.apiKeys).forEach(key => {
              // 兼容旧键名映射
              const keyMap = {
                'fofa.red': 'otherfofa',
                '0.zone': '0zone',
                zone0: '0zone',
              };
              const localKey = keyMap[key] || key;
              if (this.apiConfigs[localKey]) {
                const apiKeyValue = config.apiKeys[key] || '';
                // FOFA官方需要解析email:key格式
                if (localKey === 'fofa' && apiKeyValue.includes(':')) {
                  const parts = apiKeyValue.split(':', 2);
                  this.apiConfigs[localKey].email = parts[0] || '';
                  this.apiConfigs[localKey].key = parts[1] || '';
                } else {
                  this.apiConfigs[localKey].key = apiKeyValue;
                }
                this.apiConfigs[localKey].enabled = config.enabled?.[key] || false;
              }
            });
          }
          if (config.settings) {
            this.apiSettings = { ...this.apiSettings, ...config.settings };
            // 将 FOFA 第三方 baseURL 映射到本地默认值（用于前端展示）
            if (!this.apiSettings.fofaThirdBaseURL) {
              this.apiSettings.fofaThirdBaseURL = 'https://fofa.red';
            }
            // 更新API客户端的超时时间
            this.updateApiClientTimeout();
          }
          this.updatePlatformSelection();
        }
      } catch (error) {
        console.error('加载API配置失败:', error);
      }
    },
    
    handlePlatformChange() {
      // 平台选择改变时的处理（单选模式）
      // 确保至少选择一个平台
      if (!this.selectedPlatform && this.enabledPlatformList.length > 0) {
        this.selectedPlatform = this.enabledPlatformList[0].key;
      }
    },
    
    updatePlatformSelection() {
      // 初始化时，默认选择第一个启用的平台
      if (this.enabledPlatformList.length > 0 && !this.selectedPlatform) {
        this.selectedPlatform = this.enabledPlatformList[0].key;
      }
    },
    
    clearQuery() {
      if (this.currentQueryTab) {
        this.currentQueryTab.queryText = '';
      }
      this.queryText = '';
    },
    
    // 处理标签页关闭事件（点击关闭按钮）
    async handleTabRemove(tabId) {
      const targetTab = this.queryTabs.find(tab => tab.id === tabId);
      if (!targetTab) {
        return;
      }
      
      // 保存标签页数据的副本，以便在用户取消时恢复
      const tabBackup = JSON.parse(JSON.stringify(targetTab));
      const tabIndex = this.queryTabs.findIndex(tab => tab.id === tabId);
      
      // 检查是否有数据且未保存
      if (targetTab.totalResults > 0) {
        try {
          await this.$confirm(
            `标签页 "${targetTab.name}" 包含 ${targetTab.totalResults} 条查询结果，是否已保存数据？`,
            '提示',
            {
              confirmButtonText: '已保存',
              cancelButtonText: '取消',
              type: 'warning',
              distinguishCancelAndClose: true,
            }
          );
        } catch (error) {
          // 用户点击取消或关闭，需要恢复标签页
          // 检查标签页是否还在
          const currentIndex = this.queryTabs.findIndex(tab => tab.id === tabId);
          if (currentIndex === -1) {
            // 标签页已被移除，需要恢复
            // 在原来的位置插入标签页
            if (tabIndex >= 0 && tabIndex <= this.queryTabs.length) {
              this.queryTabs.splice(tabIndex, 0, tabBackup);
              // 恢复 activeTab
              this.activeTab = tabId;
              // 强制更新视图
              this.$nextTick(() => {
                this.$forceUpdate();
              });
            } else {
              // 如果索引超出范围，直接添加到末尾
              this.queryTabs.push(tabBackup);
              this.activeTab = tabId;
              this.$nextTick(() => {
                this.$forceUpdate();
              });
            }
          }
          return;
        }
      }
      
      // 执行关闭操作（如果标签页还在）
      const stillExists = this.queryTabs.findIndex(tab => tab.id === tabId) !== -1;
      if (stillExists) {
        this.removeTab(tabId);
      }
    },
    
    // 实际执行关闭标签页的方法
    removeTab(tabId) {
      const index = this.queryTabs.findIndex(tab => tab.id === tabId);
      if (index !== -1) {
        this.queryTabs.splice(index, 1);
        if (this.queryTabs.length > 0) {
          this.activeTab = this.queryTabs[this.queryTabs.length - 1].id;
        } else {
          this.activeTab = '';
        }
        // 保存到本地存储
        this.saveTabsToStorage();
      }
    },
    
    // 获取数据文件路径
    async getTabsDataFilePath() {
      const appPath = await GetAppPath();
      const pathSeparator = appPath.includes('\\') ? '\\' : '/';
      const dataDir = `${appPath}${pathSeparator}logs${pathSeparator}spaceMapping`;
      const fileName = 'spaceMapping_tabs_latest.json';
      return `${dataDir}${pathSeparator}${fileName}`;
    },
    
    // 保存标签页数据到本地 JSON 文件
    async saveTabsToStorage() {
      try {
        // 如果没有数据，不保存
        if (this.queryTabs.length === 0) {
          return;
        }
        
        const filePath = await this.getTabsDataFilePath();
        
        const dataToSave = {
          queryTabs: this.queryTabs.map(tab => ({
            id: tab.id,
            name: tab.name,
            queryText: tab.queryText,
            platforms: tab.platforms,
            queryResults: tab.queryResults,
            aggregatedData: tab.aggregatedData,
            totalResults: tab.totalResults,
            querying: tab.querying,
            activeView: tab.activeView,
            platformPagination: tab.platformPagination,
          })),
          activeTab: this.activeTab,
          tabIdCounter: this.tabIdCounter,
          savedAt: new Date().toISOString(),
        };
        
        // SaveFile 会自动创建目录（如果不存在）
        await SaveFile(filePath, JSON.stringify(dataToSave, null, 2));
        this.tabsDataFilePath = filePath;
      } catch (error) {
        console.error('保存标签页数据失败:', error);
        // 不显示错误提示，避免频繁提示影响用户体验
      }
    },
    
    // 从本地 JSON 文件加载标签页数据
    async loadTabsFromStorage() {
      try {
        const filePath = await this.getTabsDataFilePath();
        
        try {
          const content = await ReadFile(filePath);
          const data = JSON.parse(content);
          
          if (data.queryTabs && Array.isArray(data.queryTabs) && data.queryTabs.length > 0) {
            this.queryTabs = data.queryTabs;
            this.activeTab = data.activeTab || '';
            this.tabIdCounter = data.tabIdCounter || 0;
            this.tabsDataFilePath = filePath;
            
            // 确保 activeTab 存在
            if (this.activeTab && !this.queryTabs.find(tab => tab.id === this.activeTab)) {
              this.activeTab = this.queryTabs.length > 0 ? this.queryTabs[0].id : '';
            }
            
            // 确保所有标签页的 querying 状态为 false（避免加载后显示查询中状态）
            this.queryTabs.forEach(tab => {
              tab.querying = false;
            });
          }
        } catch (readError) {
          // 文件不存在或读取失败，忽略错误（首次使用时文件不存在是正常的）
          console.log('未找到已保存的数据文件，将创建新文件');
        }
      } catch (error) {
        console.error('加载标签页数据失败:', error);
      }
    },
    
    // 标签页右键菜单处理
    handleTabsContextMenu(event) {
      event.preventDefault();
      event.stopPropagation();
      
      // 检查是否点击在标签页上
      const tabItem = event.target.closest('.el-tabs__item');
      if (!tabItem) {
        this.hideTabsContextMenu();
        return;
      }
      
      const tabIdAttr = tabItem.id || tabItem.getAttribute('aria-controls');
      this.tabsContextMenuTabId = tabIdAttr
        ? tabIdAttr.replace(/^tab-/, '').replace(/^pane-/, '')
        : this.activeTab;

      // 直接从事件获取坐标，优先使用 clientX/clientY（相对于视口，适用于 position: fixed）
      const x = event.clientX || 0;
      const y = event.clientY || 0;
      
      // 直接设置坐标，确保菜单显示在鼠标点击位置
      this.tabsContextMenuPosition = { x, y };
      this.tabsContextMenuVisible = true;
      
      // 点击其他地方时隐藏菜单
      const hideMenu = (e) => {
        if (!e.target.closest('.tabs-context-menu')) {
          this.hideTabsContextMenu();
          document.removeEventListener('click', hideMenu);
          document.removeEventListener('contextmenu', hideMenu);
        }
      };
      setTimeout(() => {
        document.addEventListener('click', hideMenu);
        document.addEventListener('contextmenu', hideMenu);
      }, 100);
    },
    
    hideTabsContextMenu() {
      this.tabsContextMenuVisible = false;
    },
    
    // 关闭当前标签页（右键菜单）
    async handleCloseCurrentTab() {
      this.hideTabsContextMenu();
      
      const targetId = this.tabsContextMenuTabId || this.activeTab;
      if (!targetId) {
        return;
      }
      
      const targetTab = this.queryTabs.find(tab => tab.id === targetId);
      if (!targetTab) {
        return;
      }
      
      // 检查是否有数据且未保存
      if (targetTab.totalResults > 0) {
        try {
          await this.$confirm(
            `标签页 "${targetTab.name}" 包含 ${targetTab.totalResults} 条查询结果，是否已保存数据？`,
            '提示',
            {
              confirmButtonText: '已保存',
              cancelButtonText: '取消',
              type: 'warning',
              distinguishCancelAndClose: true,
            }
          );
        } catch (error) {
          // 用户点击取消，不关闭标签页
          return;
        }
      }
      
      // 关闭标签页
      this.removeTab(targetId);
    },
    
    // 关闭所有标签页（右键菜单）
    async handleCloseAllTabs() {
      this.hideTabsContextMenu();
      
      // 检查是否有未保存的数据
      const tabsWithData = this.queryTabs.filter(tab => tab.totalResults > 0);
      if (tabsWithData.length > 0) {
        const totalResults = tabsWithData.reduce((sum, tab) => sum + tab.totalResults, 0);
        try {
          await this.$confirm(
            `共有 ${tabsWithData.length} 个标签页包含查询结果（共 ${totalResults} 条），是否已保存数据？`,
            '提示',
            {
              confirmButtonText: '已保存',
              cancelButtonText: '取消',
              type: 'warning',
              distinguishCancelAndClose: true,
            }
          );
        } catch (error) {
          // 用户点击取消，不关闭标签页
          return;
        }
      }
      
      // 关闭所有标签页
      this.queryTabs = [];
      this.activeTab = '';
      // 保存到本地存储
      this.saveTabsToStorage();
    },
    
    createNewTab(queryText, platform) {
      const tabId = `query-${++this.tabIdCounter}`;
      const timeStr = new Date().toLocaleTimeString();
      const name = `${queryText.substring(0, 30)}${queryText.length > 30 ? '...' : ''} ${timeStr}`;
      
      const tab = {
        id: tabId,
        name: name,
        queryText: queryText,
        platforms: [platform], // 保持为数组格式以兼容后端
        queryResults: {},
        aggregatedData: null,
        totalResults: 0,
        querying: true,
        activeView: '',
        pagination: {
          currentPage: 1,
          pageSize: 20,
          total: 0,
        },
        platformPagination: {},
      };
      
      this.queryTabs.push(tab);
      this.activeTab = tabId;
      // 保存到本地存储
      this.saveTabsToStorage();
      return tab;
    },
    
    getPagedResults(tab, platform) {
      const results = tab.queryResults[platform] || [];
      if (!tab.platformPagination[platform]) {
        tab.platformPagination[platform] = {
          currentPage: 1,
          pageSize: 20,
        };
      }
      const start = (tab.platformPagination[platform].currentPage - 1) * tab.platformPagination[platform].pageSize;
      const end = start + tab.platformPagination[platform].pageSize;
      return results.slice(start, end);
    },
    
    handleSizeChange(size, tab) {
      tab.pagination.pageSize = size;
      tab.pagination.currentPage = 1;
    },
    
    handleCurrentChange(page, tab) {
      tab.pagination.currentPage = page;
    },
    
    handlePlatformSizeChange(size, platform, tab) {
      if (!tab.platformPagination[platform]) {
        tab.platformPagination[platform] = { currentPage: 1, pageSize: 20 };
      }
      tab.platformPagination[platform].pageSize = size;
      tab.platformPagination[platform].currentPage = 1;
    },
    
    handlePlatformCurrentChange(page, platform, tab) {
      if (!tab.platformPagination[platform]) {
        tab.platformPagination[platform] = { currentPage: 1, pageSize: 20 };
      }
      tab.platformPagination[platform].currentPage = page;
    },
    
    getProtocol(url) {
      if (!url) return '-';
      return url.startsWith('https') ? 'https' : 'http';
    },
    
    getProtocolTagType(url) {
      if (!url) return 'info';
      return url.startsWith('https') ? 'success' : 'primary';
    },
    
    openURL(url) {
      if (!url) return;
      try {
        BrowserOpenURL(url);
        ElMessage.success(`正在打开: ${url}`);
      } catch (error) {
        console.error('打开URL失败:', error);
        ElMessage.error(`打开URL失败: ${error?.message || error}`);
      }
    },
    
    // 表格行右键菜单处理
    handleRowContextMenu(row, column, event) {
      event.preventDefault();
      event.stopPropagation();
      
      this.selectedRow = row;
      
      // 设置菜单位置：先按鼠标位置显示，再基于实际菜单高度微调，防止刚好溢出底部
      const x = event.clientX || 0;
      const y = event.clientY || 0;
      
      this.rowContextMenuStyle.left = `${x}px`;
      this.rowContextMenuStyle.top = `${y}px`;
      this.rowContextMenuVisible = true;
      
      this.$nextTick(() => {
        const menuEl = document.querySelector('.context-menu');
        if (!menuEl) return;
        
        const menuHeight = menuEl.offsetHeight || 0;
        const windowHeight = window.innerHeight;
        if (y + menuHeight > windowHeight - 8) {
          let adjustedY = windowHeight - menuHeight - 8;
          if (adjustedY < 8) adjustedY = 8;
          this.rowContextMenuStyle.top = `${adjustedY}px`;
        }
      });
      
      // 点击其他地方时隐藏菜单
      const hideMenu = (e) => {
        if (!e.target.closest('.context-menu')) {
          this.rowContextMenuVisible = false;
          document.removeEventListener('click', hideMenu);
          document.removeEventListener('contextmenu', hideMenu);
        }
      };
      setTimeout(() => {
        document.addEventListener('click', hideMenu);
        document.addEventListener('contextmenu', hideMenu);
      }, 100);
    },
    
    // 表格多选变化处理
    handleSelectionChange(selection) {
      this.selectedRows = selection || [];
    },
    
    // 判断是否为Web服务
    isWebService(row) {
      if (!row || !row.url) return false;
      const url = row.url.toLowerCase();
      return url.startsWith('http://') || url.startsWith('https://');
    },
    
    // 判断是否可爆破协议
    isCrackableProtocol(row) {
      if (!row) return false;
      const url = (row.url || '').toLowerCase();
      const protocol = (row.protocol || '').toLowerCase();
      const port = Number(row.port) || 0;
      
      // 支持的协议列表
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
      ];
      
      // 检查是否匹配支持的协议
      for (const item of supportedProtocols) {
        if (
          protocol === item.name ||
          url.includes(`://${item.name}`) ||
          port === item.defaultPort
        ) {
          return true;
        }
      }
      
      return false;
    },
    
    // 获取Web服务的URL
    getWebURL(row) {
      if (!row || !row.url) return null;
      let url = row.url.trim();
      // 确保URL以http或https开头
      if (!url.startsWith('http://') && !url.startsWith('https://')) {
        // 根据端口判断协议
        const port = Number(row.port) || 80;
        const protocol = port === 443 ? 'https' : 'http';
        const host = row.domain || row.ip || '';
        if (host) {
          url = `${protocol}://${host}${port !== 80 && port !== 443 ? `:${port}` : ''}`;
        } else {
          return null;
        }
      }
      // 确保URL以/结尾（用于目录扫描和指纹识别）
      return url.endsWith('/') ? url : `${url}/`;
    },
    
    // 单个URL发送到指纹识别
    async handleRunFingerprint() {
      if (!this.selectedRow) return;
      this.rowContextMenuVisible = false;
      
      const url = this.getWebURL(this.selectedRow);
      if (!url) {
        ElMessage.warning('无法获取有效的URL');
        return;
      }
      
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
      };
      
      // 通过事件联动创建标签页并启动扫描
      window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'fingerprint' }));
      setTimeout(() => {
        window.dispatchEvent(new CustomEvent('cyberSabre-fingerprint-fill', { 
          detail: { 
            targets: [url], 
            config: cfg
          } 
        }));
      }, 100);
      ElMessage.success(`已启动指纹识别: ${url}`);
    },
    
    // 批量URL发送到指纹识别
    async handleRunFingerprintMulti() {
      if (this.selectedRows.length === 0) {
        ElMessage.warning('请先选择要发送的行');
        return;
      }
      this.rowContextMenuVisible = false;
      
      const urls = [];
      for (const row of this.selectedRows) {
        const url = this.getWebURL(row);
        if (url && this.isWebService(row)) {
          urls.push(url);
        }
      }
      
      if (urls.length === 0) {
        ElMessage.warning('选中的行中没有有效的Web服务URL');
        return;
      }
      
      const cfg = {
        targets: urls,
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
      };
      
      // 通过事件联动创建标签页并启动扫描
      window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'fingerprint' }));
      setTimeout(() => {
        window.dispatchEvent(new CustomEvent('cyberSabre-fingerprint-fill', { 
          detail: { 
            targets: urls, 
            config: cfg
          } 
        }));
      }, 100);
      ElMessage.success(`已启动批量指纹识别，共 ${urls.length} 个目标`);
    },
    
    // 单个URL发送到目录扫描
    async handleRunDirsearch() {
      if (!this.selectedRow) return;
      this.rowContextMenuVisible = false;
      
      const url = this.getWebURL(this.selectedRow);
      if (!url) {
        ElMessage.warning('无法获取有效的URL');
        return;
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
      };
      
      // 通过事件联动创建标签页并启动扫描
      window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'dirsearch' }));
      setTimeout(() => {
        window.dispatchEvent(new CustomEvent('cyberSabre-dirscan-fill', { detail: scanConfig }));
        window.dispatchEvent(new CustomEvent('cyberSabre-dirscan-request', { detail: scanConfig }));
      }, 100);
      ElMessage.success(`已启动目录扫描: ${url}`);
    },
    
    // 批量URL发送到目录扫描
    async handleRunDirsearchMulti() {
      if (this.selectedRows.length === 0) {
        ElMessage.warning('请先选择要发送的行');
        return;
      }
      this.rowContextMenuVisible = false;
      
      const urls = [];
      for (const row of this.selectedRows) {
        const url = this.getWebURL(row);
        if (url && this.isWebService(row)) {
          urls.push(url);
        }
      }
      
      if (urls.length === 0) {
        ElMessage.warning('选中的行中没有有效的Web服务URL');
        return;
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
      };
      
      // 通过事件联动创建标签页并启动扫描
      window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'dirsearch' }));
      setTimeout(() => {
        window.dispatchEvent(new CustomEvent('cyberSabre-dirscan-fill', { detail: scanConfig }));
        window.dispatchEvent(new CustomEvent('cyberSabre-dirscan-request', { detail: scanConfig }));
      }, 100);
      ElMessage.success(`已启动批量目录扫描，共 ${urls.length} 个目标`);
    },
    
    // 单个URL发送到漏洞扫描
    handleRunVulnScan() {
      if (!this.selectedRow) return;
      this.rowContextMenuVisible = false;
      
      const url = this.getWebURL(this.selectedRow);
      if (!url) {
        ElMessage.warning('无法获取有效的URL');
        return;
      }
      
      // 打开对话框
      this.vulnScanTargets = [url];
      this.vulnScanTaskName = `空间测绘-漏洞扫描-${new Date().toLocaleTimeString()}`;
      this.vulnScanMode = 'full';
      this.vulnScanCustomTags = [];
      this.vulnScanCustomTemplate = [];
      this.vulnScanSkipNucleiWithoutTags = false;
      this.vulnScanDialogVisible = true;
    },
    
    // 批量URL发送到漏洞扫描
    handleRunVulnScanMulti() {
      if (this.selectedRows.length === 0) {
        ElMessage.warning('请先选择要发送的行');
        return;
      }
      this.rowContextMenuVisible = false;
      
      const urls = [];
      for (const row of this.selectedRows) {
        const url = this.getWebURL(row);
        if (url && this.isWebService(row)) {
          urls.push(url);
        }
      }
      
      if (urls.length === 0) {
        ElMessage.warning('选中的行中没有有效的Web服务URL');
        return;
      }
      
      // 打开对话框
      this.vulnScanTargets = urls;
      this.vulnScanTaskName = `空间测绘-批量漏洞扫描-${new Date().toLocaleTimeString()}`;
      this.vulnScanMode = 'full';
      this.vulnScanCustomTags = [];
      this.vulnScanCustomTemplate = [];
      this.vulnScanSkipNucleiWithoutTags = false;
      this.vulnScanDialogVisible = true;
    },
    
    // 调用 JSFinder 模块（接口探测）
    async callBrowserScanModule(urls) {
      try {
        const uniqueUrls = [...new Set(urls)];
        const httpsUrl = uniqueUrls.find(url => url.startsWith('https://'));
        const targetUrl = httpsUrl || uniqueUrls[0];
        if (!targetUrl) {
          return;
        }
        
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
        };
        
        await CreateBrowserScanTask(targetUrl, config);
        window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'browser-scan' }));
        ElMessage.success(`已启动 JSFinder 接口探测，目标: ${targetUrl}`);
      } catch (error) {
        ElMessage.error(`启动 JSFinder 失败: ${error}`);
      }
    },
    
    // 单个URL发送到接口探测（JSFinder）
    handleRunAPIDetectionSingle() {
      if (!this.selectedRow) return;
      this.rowContextMenuVisible = false;
      
      const url = this.getWebURL(this.selectedRow);
      if (!url) {
        ElMessage.warning('无法获取有效的URL');
        return;
      }
      this.callBrowserScanModule([url]);
    },
    
    // 批量URL发送到接口探测（JSFinder）
    handleRunAPIDetectionMulti() {
      if (this.selectedRows.length === 0) {
        ElMessage.warning('请先选择要发送的行');
        return;
      }
      this.rowContextMenuVisible = false;
      
      const urls = [];
      for (const row of this.selectedRows) {
        const url = this.getWebURL(row);
        if (url && this.isWebService(row)) {
          urls.push(url);
        }
      }
      
      if (urls.length === 0) {
        ElMessage.warning('选中的行中没有有效的Web服务URL');
        return;
      }
      this.callBrowserScanModule(urls);
    },
    
    // 确认创建漏洞扫描任务
    async confirmVulnScan() {
      if (this.vulnScanTargets.length === 0) {
        ElMessage.warning('没有有效的目标URL');
        return;
      }
      
      let proxyString = '';
      try {
        const globalProxy = await GetGlobalProxy();
        if (globalProxy?.enabled && globalProxy.config) {
          const proxyConfig = globalProxy.config;
          if (proxyConfig.type && proxyConfig.host && proxyConfig.port) {
            const scheme = proxyConfig.type.toLowerCase();
            const host = proxyConfig.host;
            const port = proxyConfig.port;
            if (proxyConfig.auth && proxyConfig.auth.username) {
              const username = encodeURIComponent(proxyConfig.auth.username);
              const password = encodeURIComponent(proxyConfig.auth.password || '');
              proxyString = `${scheme}://${username}:${password}@${host}:${port}`;
            } else {
              proxyString = `${scheme}://${host}:${port}`;
            }
          }
        }
      } catch (e) {
        console.error('获取全局代理配置失败:', e);
      }
      
      let scanConfig = { threads: 5, timeout: 10, rate_limit: 150, retries: 1 };
      let templateFolders = [];
      try {
        const configData = await GetPocConfig();
        if (configData) {
          scanConfig = {
            threads: configData.threads || 5,
            timeout: configData.timeout || 10,
            rate_limit: configData.rate_limit || 150,
            retries: configData.retries || 1
          };
          templateFolders = (configData.template_folders || []).filter(Boolean);
        }
      } catch (e) {
        console.error('获取漏洞扫描配置失败:', e);
      }
      
      // 根据扫描模式设置参数
      let tagsParam = [];
      let templateFileParam = [];
      
      if (this.vulnScanMode === 'fingerprint') {
        // 指纹扫描模式
        tagsParam = [...this.vulnScanCustomTags];
      } else {
        // 全量扫描模式，tags 和 template_file 都为空
        tagsParam = [];
        templateFileParam = [];
      }
      
      const taskName = this.vulnScanTaskName || `空间测绘-漏洞扫描-${new Date().toLocaleTimeString()}`;
      const payload = {
        targets: this.vulnScanTargets,
        task_name: taskName,
        append_poc: '',
        tags: tagsParam,
        custom_tags: [],
        template_file: templateFileParam,
        template_folders: templateFolders,
        custom_headers: '',
        proxy: proxyString,
        skip_nuclei_without_tags: this.vulnScanSkipNucleiWithoutTags,
        thread_safe: true,
        threads: scanConfig.threads,
        timeout: scanConfig.timeout,
        rate_limit: scanConfig.rate_limit,
        retries: scanConfig.retries
      };
      
      try {
        const taskId = await StartPocScan(JSON.stringify(payload));
        this.vulnScanDialogVisible = false;
        
        // 先切换到漏洞扫描标签页
        window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'webscan' }));
        
        // 等待一小段时间确保标签页切换完成，然后刷新任务列表并打开任务
        setTimeout(() => {
          // 先刷新任务列表，确保任务数据是最新的
          window.dispatchEvent(new CustomEvent('webscan-refresh-tasks'));
          // 然后打开任务详情
          setTimeout(() => {
            window.dispatchEvent(
              new CustomEvent('webscan-open-task', {
                detail: { taskId }
              })
            );
          }, 200); // 再等待200ms确保任务列表已刷新
        }, 300); // 等待300ms确保标签页切换完成
        
        ElMessage.success(`已启动漏洞扫描任务，共 ${this.vulnScanTargets.length} 个目标`);
      } catch (e) {
        console.error('启动漏洞扫描任务失败:', e);
        ElMessage.error(`启动漏洞扫描任务失败: ${e?.message || e}`);
      }
    },
    
    // 单个URL发送到密码爆破
    async handleRunCrack() {
      if (!this.selectedRow) return;
      this.rowContextMenuVisible = false;
      
      if (!this.isCrackableProtocol(this.selectedRow)) {
        ElMessage.warning('当前行不支持密码爆破');
        return;
      }
      
      // 这里需要用户配置用户名和密码字典，暂时提示
      ElMessage.info('密码爆破功能需要配置用户名和密码字典，请在"渗透测试-密码爆破"模块中配置');
      window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'passwordcrack' }));
    },
    
    // 批量URL发送到密码爆破
    async handleRunCrackMulti() {
      if (this.selectedRows.length === 0) {
        ElMessage.warning('请先选择要发送的行');
        return;
      }
      this.rowContextMenuVisible = false;
      
      const crackableRows = this.selectedRows.filter(row => this.isCrackableProtocol(row));
      if (crackableRows.length === 0) {
        ElMessage.warning('选中的行中没有支持密码爆破的协议');
        return;
      }
      
      ElMessage.info(`已选中 ${crackableRows.length} 个可爆破目标，请在"渗透测试-密码爆破"模块中配置并启动`);
      window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'passwordcrack' }));
    },
    
    // 复制当前行
    handleCopyRow() {
      if (!this.selectedRow) return;
      this.rowContextMenuVisible = false;
      
      const text = JSON.stringify(this.selectedRow, null, 2);
      navigator.clipboard.writeText(text).then(() => {
        ElMessage.success('已复制到剪贴板');
      }).catch(() => {
        ElMessage.error('复制失败');
      });
    },
    
    // 复制选中行
    handleCopySelected() {
      if (this.selectedRows.length === 0) {
        ElMessage.warning('请先选择要复制的行');
        return;
      }
      this.rowContextMenuVisible = false;
      
      const text = JSON.stringify(this.selectedRows, null, 2);
      navigator.clipboard.writeText(text).then(() => {
        ElMessage.success(`已复制 ${this.selectedRows.length} 行到剪贴板`);
      }).catch(() => {
        ElMessage.error('复制失败');
      });
    },
    
    copySyntax(text) {
      if (!text || text === '<不支持>') {
        return;
      }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => {
          ElMessage.success('已复制到剪贴板');
        }).catch(() => {
          this.fallbackCopyText(text);
        });
      } else {
        this.fallbackCopyText(text);
      }
    },
    
    fallbackCopyText(text) {
      const textArea = document.createElement('textarea');
      textArea.value = text;
      textArea.style.position = 'fixed';
      textArea.style.left = '-999999px';
      textArea.style.top = '-999999px';
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();
      try {
        document.execCommand('copy');
        ElMessage.success('已复制到剪贴板');
      } catch (err) {
        ElMessage.error('复制失败');
      }
      document.body.removeChild(textArea);
    },
    
    async handleSaveApiConfig() {
      try {
        if (!this.apiClient) {
          ElMessage.error('API客户端未初始化');
          return;
        }
        
        const apiKeys = {};
        const enabled = {};
        Object.keys(this.apiConfigs).forEach(key => {
          // FOFA官方需要组合email:key格式
          if (key === 'fofa') {
            const email = this.apiConfigs[key].email || '';
            const keyValue = this.apiConfigs[key].key || '';
            if (email && keyValue) {
              apiKeys[key] = `${email}:${keyValue}`;
            } else if (keyValue) {
              // 如果只有key，也保存（后端会处理）
              apiKeys[key] = keyValue;
            } else {
              apiKeys[key] = '';
            }
          } else {
            apiKeys[key] = this.apiConfigs[key].key || '';
          }
          enabled[key] = this.apiConfigs[key].enabled;
        });

        // 确保 FOFA 第三方 baseURL 写入全局设置
        if (!this.apiSettings.fofaThirdBaseURL || this.apiSettings.fofaThirdBaseURL.trim() === '') {
          this.apiSettings.fofaThirdBaseURL = 'https://fofa.red';
        }
        
        const response = await this.apiClient.post('/api/info-collection/config', {
          apiKeys,
          enabled,
          settings: this.apiSettings,
        });
        
        if (response.data && response.data.success) {
          ElMessage.success('配置保存成功');
          this.updatePlatformSelection();
          this.showApiConfigDialog = false;
        } else {
          ElMessage.error(response.data?.message || '保存失败');
        }
      } catch (error) {
        console.error('保存API配置失败:', error);
        ElMessage.error('保存配置失败: ' + (error.response?.data?.message || error.message));
      }
    },
    
    async handleVerifyApiKeys() {
      if (!this.apiClient) {
        ElMessage.error('API客户端未初始化');
        return;
      }
      
      const verifyPromises = [];
      const verifyResults = {};
      
      for (const [platform, config] of Object.entries(this.apiConfigs)) {
        // FOFA官方需要email和key，其他平台只需要key
        const hasKey = platform === 'fofa' 
          ? (config.email && config.email.trim() !== '' && config.key && config.key.trim() !== '')
          : (config.key && config.key.trim() !== '');
        
        if (hasKey) {
          verifyResults[platform] = { status: 'verifying' };
          this.apiConfigs[platform].status = 'unknown';
          // FOFA官方需要组合email:key格式
          const apiKey = platform === 'fofa' 
            ? `${config.email}:${config.key}`
            : config.key;
          verifyPromises.push(
            this.verifySingleKey(platform, apiKey).then(result => {
              if (result.valid) {
                verifyResults[platform] = { status: 'valid', valid: true };
                this.apiConfigs[platform].status = 'valid';
              } else if (result.retryable) {
                verifyResults[platform] = { status: 'unknown', valid: false, retryable: true, message: result.message };
                this.apiConfigs[platform].status = 'unknown';
              } else {
                verifyResults[platform] = { status: 'invalid', valid: false, message: result.message };
                this.apiConfigs[platform].status = 'invalid';
              }
            }).catch(err => {
              verifyResults[platform] = { status: 'invalid', error: err.message };
              this.apiConfigs[platform].status = 'invalid';
            })
          );
        }
      }
      
      if (verifyPromises.length === 0) {
        ElMessage.warning('没有配置任何API Key');
        return;
      }
      
      ElMessage.info(`正在验证 ${verifyPromises.length} 个API Key...`);
      
      await Promise.all(verifyPromises);
      
      const validCount = Object.values(verifyResults).filter(r => r.status === 'valid').length;
      const invalidCount = Object.values(verifyResults).filter(r => r.status === 'invalid').length;
      const unknownCount = Object.values(verifyResults).filter(r => r.status === 'unknown').length;
      
      if (unknownCount > 0) {
        ElMessage.warning(`验证完成：有效 ${validCount} 个，无效 ${invalidCount} 个，待重试 ${unknownCount} 个（网络或超时）`);
      } else {
        ElMessage.success(`验证完成：有效 ${validCount} 个，无效 ${invalidCount} 个`);
      }
    },
    
    async verifySingleKey(platform, apiKey) {
      const requestPayload = {
        platform: platform,
        apiKey: apiKey,
        // 直接携带当前前端设置，避免用户必须先点击“保存”才能使用最新配置进行验证
        settings: this.apiSettings,
      };
      // 校验请求使用更大的超时，降低首轮连接建立导致的误判
      const requestTimeout = Math.max((this.apiSettings?.timeout || 30) * 1000, 20000);
      
      const doVerify = async () => {
        const response = await this.apiClient.post('/api/info-collection/space-mapping/verify-key', requestPayload, {
          timeout: requestTimeout,
        });
        if (response.data && response.data.success) {
          return {
            valid: response.data.valid === true,
            message: response.data.message || '',
          };
        }
        return {
          valid: false,
          message: response.data?.message || '验证失败',
        };
      };
      
      try {
        const firstResult = await doVerify();
        if (firstResult.valid) {
          return { valid: true, retryable: false, message: firstResult.message };
        }
        const msg = (firstResult.message || '').toLowerCase();
        const isRetryable = msg.includes('超时') || msg.includes('network') || msg.includes('timeout') || msg.includes('重试');
        if (!isRetryable) {
          return { valid: false, retryable: false, message: firstResult.message };
        }
        // 对临时错误自动重试一次，避免第一次冷启动误判
        await new Promise(resolve => setTimeout(resolve, 600));
        const retryResult = await doVerify();
        return {
          valid: retryResult.valid === true,
          retryable: retryResult.valid ? false : true,
          message: retryResult.message,
        };
      } catch (error) {
        console.error(`验证 ${platform} Key失败:`, error);
        const msg = (error?.response?.data?.message || error?.message || '').toLowerCase();
        const isRetryable = msg.includes('超时') || msg.includes('network') || msg.includes('timeout') || msg.includes('重试');
        return {
          valid: false,
          retryable: isRetryable,
          message: error?.response?.data?.message || error?.message || '验证请求失败',
        };
      }
    },
    
    async handleSaveSettings() {
      try {
        if (!this.apiClient) {
          ElMessage.error('API客户端未初始化');
          return;
        }

        // 确保 FOFA 第三方 baseURL 有默认值
        if (!this.apiSettings.fofaThirdBaseURL || this.apiSettings.fofaThirdBaseURL.trim() === '') {
          this.apiSettings.fofaThirdBaseURL = 'https://fofa.red';
        }
        
        // 验证pageSize，给出提示
        const pageSize = this.apiSettings.pageSize || 100;
        if (pageSize > 100) {
          const warnings = [];
          if (pageSize > 10000) {
            warnings.push('FOFA平台最多支持10000条/页，超过部分将被限制');
            warnings.push('Hunter和Quake平台最多支持100条/页，超过部分将被自动限制');
          } else {
            warnings.push('Hunter和Quake平台最多支持100条/页，超过部分将被自动限制');
          }
          if (warnings.length > 0) {
            ElMessage.warning({
              message: warnings.join('；'),
              duration: 5000,
            });
          }
        }
        
        // 保存设置时，同时保存当前的API Keys和启用状态，确保修改配置后无需重新点击配置Api保存
        const apiKeys = {};
        const enabled = {};
        Object.keys(this.apiConfigs).forEach(key => {
          // FOFA官方需要组合email:key格式
          if (key === 'fofa') {
            const email = this.apiConfigs[key].email || '';
            const keyValue = this.apiConfigs[key].key || '';
            if (email && keyValue) {
              apiKeys[key] = `${email}:${keyValue}`;
            } else if (keyValue) {
              // 如果只有key，也保存（后端会处理）
              apiKeys[key] = keyValue;
            } else {
              apiKeys[key] = '';
            }
          } else {
            apiKeys[key] = this.apiConfigs[key].key || '';
          }
          enabled[key] = this.apiConfigs[key].enabled;
        });
        
        const response = await this.apiClient.post('/api/info-collection/config', {
          apiKeys: apiKeys,
          enabled: enabled,
          settings: this.apiSettings,
        });
        
        if (response.data && response.data.success) {
          // 保存成功后重新加载配置，确保显示正确的值
          await this.loadApiConfig();
          // 更新API客户端的超时时间（loadApiConfig中已经更新，这里再确保一次）
          this.updateApiClientTimeout();
          ElMessage.success('设置保存成功');
          this.showSettingsDialog = false;
        } else {
          ElMessage.error(response.data?.message || '保存失败');
        }
      } catch (error) {
        console.error('保存设置失败:', error);
        ElMessage.error('保存设置失败: ' + (error.response?.data?.message || error.message));
      }
    },
    
    openApiApplyPage(platform) {
      let url = '';
      if (platform === '0zone') {
        url = 'https://0.zone/apply';
      } else if (platform === 'fofa') {
        url = 'https://fofa.info/';
      } else if (platform === 'otherfofa') {
        // 使用配置的 FOFA 第三方 BaseURL，默认为 https://fofa.red/
        url = (this.apiSettings.fofaThirdBaseURL || 'https://fofa.red/').trim();
      } else if (platform === 'hunter') {
        url = 'https://hunter.qianxin.com/';
      } else if (platform === 'quake') {
        url = 'https://quake.360.net/';
      }
      if (url) {
        BrowserOpenURL(url);
      }
    },
    
    getPlatformLabel(platform) {
      const labels = {
        '0zone': '0.zone',
        fofa: 'FOFA官方',
        otherfofa: 'FOFA第三方',
        hunter: 'Hunter',
        quake: 'Quake',
      };
      return labels[platform] || platform;
    },
    
    getItemValue(item) {
      // Element Plus 的 tag slot 中，item 可能是值本身，也可能是包含 value 的对象
      if (!item) return '';
      if (typeof item === 'string') {
        return item;
      }
      return item?.value || item;
    },
    
    async handleQuery() {
      if (!this.queryText.trim()) {
        ElMessage.warning('请输入查询内容');
        return;
      }
      
      // 使用选中的平台（单选模式）
      if (!this.selectedPlatform) {
        ElMessage.warning('请选择一个平台');
        return;
      }
      
      this.queryLoading = true;
      
      // 生成查询ID
      const queryId = `query_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      this.currentQueryId = queryId;
      this.isQueryCanceled = false;
      
      // 创建新标签页
      const tab = this.createNewTab(this.queryText, this.selectedPlatform);
      
      try {
        if (!this.apiClient) {
          await this.initApiClient();
        }
        
        // 准备API Keys和启用状态，优先使用当前前端配置，避免需要先保存配置
        const apiKeys = {};
        const enabled = {};
        Object.keys(this.apiConfigs).forEach(key => {
          // FOFA官方需要组合email:key格式
          if (key === 'fofa') {
            const email = this.apiConfigs[key].email || '';
            const keyValue = this.apiConfigs[key].key || '';
            if (email && keyValue) {
              apiKeys[key] = `${email}:${keyValue}`;
            } else if (keyValue) {
              apiKeys[key] = keyValue;
            } else {
              apiKeys[key] = '';
            }
          } else {
            apiKeys[key] = this.apiConfigs[key].key || '';
          }
          enabled[key] = this.apiConfigs[key].enabled;
        });
        
        // 注意：不要使用AbortController，让后端取消context并返回已查询到的结果
        // 这样前端的axios请求就能正常接收到响应
        const response = await this.apiClient.post('/api/info-collection/space-mapping/query', {
          query: this.queryText,
          platforms: [this.selectedPlatform], // 后端需要数组格式
          settings: this.apiSettings, // 传入当前设置，避免需要先保存配置
          apiKeys: apiKeys, // 传入当前API Keys，避免需要先保存配置
          enabled: enabled, // 传入当前启用状态，避免需要先保存配置
          queryId: queryId, // 传入查询ID，用于取消查询
        });
        
        if (response.data && response.data.success) {
          tab.queryResults = response.data.data || {};
          tab.aggregatedData = response.data.aggregated || null;
          tab.totalResults = Object.values(tab.queryResults)
            .reduce((sum, results) => sum + (results?.length || 0), 0);
          
          // 为各平台数据添加协议字段
          Object.keys(tab.queryResults).forEach(platform => {
            tab.queryResults[platform].forEach(item => {
              if (!item.protocol) {
                item.protocol = this.getProtocol(item.url);
              }
            });
            // 初始化平台分页
            if (!tab.platformPagination[platform]) {
              tab.platformPagination[platform] = {
                currentPage: 1,
                pageSize: 20,
              };
            }
          });
          
          // 设置默认激活第一个平台标签页
          const platforms = Object.keys(tab.queryResults);
          if (platforms.length > 0) {
            tab.activeView = platforms[0];
          }
          
          // 确保查询状态被设置为false
          tab.querying = false;
          
          // 保存数据到本地存储
          this.saveTabsToStorage();
          
          // 强制更新视图 - 使用多种方式确保刷新
          this.$nextTick(() => {
            // 方法1: 强制更新整个组件
            this.$forceUpdate();
            // 方法2: 通过改变 activeTab 触发重新渲染
            const currentActiveTab = this.activeTab;
            this.activeTab = '';
            this.$nextTick(() => {
              this.activeTab = currentActiveTab;
              // 方法3: 再次强制更新确保表格刷新
              this.$forceUpdate();
            });
          });
          
          // 检查是否有错误信息（message中包含"查询失败"或"平台"等关键词，说明是错误信息而不是queryID）
          const message = response.data?.message || '';
          const isErrorMessage = message && (
            message.includes('查询失败') || 
            message.includes('平台') || 
            message.includes('错误') ||
            message.includes('失败') ||
            message.includes('不足') ||
            message.includes('额度') ||
            message.includes('配置')
          );
          
          if (tab.totalResults > 0) {
            const uniqueCount = tab.aggregatedData?.unique || tab.totalResults;
            ElMessage.success(`查询完成，共找到 ${tab.totalResults} 条结果，去重后 ${uniqueCount} 条`);
            // 如果有错误信息，也显示警告
            if (isErrorMessage) {
              ElMessage.warning(message);
            }
          } else {
            // 未找到结果时，检查是否有错误信息
            if (isErrorMessage) {
              // 有错误信息，显示错误而不是"未找到相关结果"
              ElMessage.error(message);
            } else {
              // 没有错误信息，才显示"未找到相关结果"
              ElMessage.info('未找到相关结果');
            }
          }
        } else {
          // 查询失败，优先显示错误信息
          const errorMessage = response.data?.message || '查询失败';
          ElMessage.error(errorMessage);
          
          // 查询失败，检查是否有已查询到的数据
          if (tab) {
            tab.querying = false;
            // 检查是否有已查询到的数据
            const hasData = response.data?.data && Object.keys(response.data.data).length > 0 && 
              Object.values(response.data.data).some(results => results && results.length > 0);
            if (hasData) {
              // 有数据，保留标签页并显示已有数据
              tab.queryResults = response.data.data || {};
              tab.aggregatedData = response.data.aggregated || null;
              tab.totalResults = Object.values(tab.queryResults)
                .reduce((sum, results) => sum + (results?.length || 0), 0);
              
              // 为各平台数据添加协议字段
              Object.keys(tab.queryResults).forEach(platform => {
                tab.queryResults[platform].forEach(item => {
                  if (!item.protocol) {
                    item.protocol = this.getProtocol(item.url);
                  }
                });
                // 初始化平台分页
                if (!tab.platformPagination[platform]) {
                  tab.platformPagination[platform] = {
                    currentPage: 1,
                    pageSize: 20,
                  };
                }
              });
              
              // 设置默认激活第一个平台标签页
              const platforms = Object.keys(tab.queryResults);
              if (platforms.length > 0) {
                tab.activeView = platforms[0];
              }
              
              // 保存数据到本地存储
              this.saveTabsToStorage();
              
              // 强制更新视图
              this.$nextTick(() => {
                this.$forceUpdate();
              });
              
              ElMessage.warning(`查询部分失败，已返回 ${tab.totalResults} 条已查询到的结果`);
            } else {
              // 没有数据，显示错误信息并移除标签页
              // 错误信息已经在上面显示了，这里直接移除标签页
              this.removeTab(tab.id);
            }
          }
        }
      } catch (error) {
        console.error('查询失败:', error);
        // 检查是否是取消查询导致的错误
        const isCanceled = this.isQueryCanceled || 
          (error.name === 'CanceledError' || error.name === 'AbortError') ||
          (error.message && (
            error.message.includes('canceled') || 
            error.message.includes('取消') ||
            error.message.includes('aborted')
          )) ||
          error.code === 'ECONNABORTED';
        
        if (isCanceled) {
          // 取消查询，尝试等待一小段时间让后端返回已查询到的结果
          // 注意：当context被取消时，后端会立即返回已查询到的结果
          // 但由于axios请求被取消，我们无法直接接收响应
          // 所以我们需要等待一小段时间，然后尝试重新获取结果
          if (tab) {
            // 等待一小段时间，让后端有时间返回已查询到的结果
            await new Promise(resolve => setTimeout(resolve, 500));
            
            // 检查响应中是否有数据（无论success是否为true）
            let hasData = false;
            let responseData = null;
            
            // 首先检查error.response中是否有数据
            if (error.response && error.response.data) {
              // 即使success不是true，也检查是否有数据
              if (error.response.data.data) {
                const data = error.response.data.data;
                hasData = Object.keys(data).length > 0 && 
                  Object.values(data).some(results => results && results.length > 0);
                if (hasData) {
                  responseData = error.response.data;
                }
              } else if (error.response.data.success && error.response.data.data) {
                // 如果success是true，直接使用
                responseData = error.response.data;
                hasData = true;
              }
            }
            
            // 如果响应中有数据，就使用它
            if (hasData && responseData) {
              // 后端返回了已查询到的结果
              tab.querying = false;
              tab.queryResults = responseData.data || {};
              tab.aggregatedData = responseData.aggregated || null;
              tab.totalResults = Object.values(tab.queryResults)
                .reduce((sum, results) => sum + (results?.length || 0), 0);
              
              // 为各平台数据添加协议字段
              Object.keys(tab.queryResults).forEach(platform => {
                tab.queryResults[platform].forEach(item => {
                  if (!item.protocol) {
                    item.protocol = this.getProtocol(item.url);
                  }
                });
                // 初始化平台分页
                if (!tab.platformPagination[platform]) {
                  tab.platformPagination[platform] = {
                    currentPage: 1,
                    pageSize: 20,
                  };
                }
              });
              
              // 设置默认激活第一个平台标签页
              const platforms = Object.keys(tab.queryResults);
              if (platforms.length > 0) {
                tab.activeView = platforms[0];
              }
              
              // 保存数据到本地存储
              this.saveTabsToStorage();
              
              // 强制更新视图
              this.$nextTick(() => {
                this.$forceUpdate();
              });
              
              if (tab.totalResults > 0) {
                ElMessage.info(`查询已取消，已返回 ${tab.totalResults} 条已查询到的结果`);
              } else {
                ElMessage.info('查询已取消');
                this.removeTab(tab.id);
              }
            } else {
              // 取消查询但没有返回数据，可能是因为请求被立即中断
              // 在这种情况下，我们无法获取已查询到的结果
              // 但我们可以提示用户
              tab.querying = false;
              this.removeTab(tab.id);
              ElMessage.warning('查询已取消，但无法获取已查询到的结果（请求被中断）');
            }
          }
        } else {
          // 真正的查询失败
          // 优先提取错误信息
          let errorMessage = '查询失败';
          if (error.response && error.response.data) {
            // 优先使用后端返回的错误信息
            errorMessage = error.response.data.message || error.response.data.error || errorMessage;
          } else if (error.message) {
            errorMessage = error.message;
          }
          
          // 检查响应中是否有已查询到的数据（无论success是否为true）
          let hasData = false;
          let responseData = null;
          
          if (tab && error.response && error.response.data) {
            // 检查是否有数据字段
            if (error.response.data.data) {
              const data = error.response.data.data;
              hasData = Object.keys(data).length > 0 && 
                Object.values(data).some(results => results && results.length > 0);
              if (hasData) {
                responseData = error.response.data;
              }
            }
          }
          
          if (hasData && responseData) {
            // 有数据，保留标签页并显示已有数据
            // 但也要显示错误信息，让用户知道查询过程中出现了问题
            ElMessage.warning(errorMessage);
            
            tab.querying = false;
            tab.queryResults = responseData.data || {};
            tab.aggregatedData = responseData.aggregated || null;
            tab.totalResults = Object.values(tab.queryResults)
              .reduce((sum, results) => sum + (results?.length || 0), 0);
            
            // 为各平台数据添加协议字段
            Object.keys(tab.queryResults).forEach(platform => {
              tab.queryResults[platform].forEach(item => {
                if (!item.protocol) {
                  item.protocol = this.getProtocol(item.url);
                }
              });
              // 初始化平台分页
              if (!tab.platformPagination[platform]) {
                tab.platformPagination[platform] = {
                  currentPage: 1,
                  pageSize: 20,
                };
              }
            });
            
            // 设置默认激活第一个平台标签页
            const platforms = Object.keys(tab.queryResults);
            if (platforms.length > 0) {
              tab.activeView = platforms[0];
            }
            
            // 保存数据到本地存储
            this.saveTabsToStorage();
            
            // 强制更新视图
            this.$nextTick(() => {
              this.$forceUpdate();
            });
            
            ElMessage.warning(`查询部分失败，已返回 ${tab.totalResults} 条已查询到的结果`);
          } else {
            // 没有数据，显示错误信息并移除标签页
            ElMessage.error(errorMessage);
            if (tab) {
              tab.querying = false;
              this.removeTab(tab.id);
            }
          }
        }
      } finally {
        this.queryLoading = false;
        this.currentQueryId = null;
        this.isQueryCanceled = false;
      }
    },
    
    async handleCancelQuery() {
      if (!this.currentQueryId) {
        ElMessage.warning('没有正在进行的查询');
        return;
      }
      
      // 标记查询已被取消
      this.isQueryCanceled = true;
      
      try {
        if (!this.apiClient) {
          await this.initApiClient();
        }
        
        // 发送取消请求到后端，后端会取消context并返回已查询到的结果
        // 注意：我们不要取消axios请求，让后端正常返回已查询到的结果
        const response = await this.apiClient.post('/api/info-collection/space-mapping/cancel', {
          queryId: this.currentQueryId,
        });
        
        if (response.data && response.data.success) {
          // 取消成功，等待查询请求完成（即使被取消，也会返回已查询到的结果）
          // 注意：查询请求可能还在进行中，我们需要等待它完成
          // 但由于我们已经取消了查询，后端会立即返回已查询到的结果
          ElMessage.info('正在取消查询，将返回已查询到的结果...');
          // 不立即重置状态，让查询请求完成并返回已查询到的结果
          // 状态会在查询请求的finally中重置
        } else {
          ElMessage.error(response.data?.message || '取消查询失败');
          // 取消失败，重置状态
          this.queryLoading = false;
          this.currentQueryId = null;
          this.isQueryCanceled = false;
        }
      } catch (error) {
        console.error('取消查询失败:', error);
        ElMessage.error('取消查询失败: ' + (error.response?.data?.message || error.message));
        // 即使取消失败，也重置状态
        this.queryLoading = false;
        this.currentQueryId = null;
        this.isQueryCanceled = false;
      }
    },
    
    async handleExportCommand(command) {
      if (command === 'excel') {
        await this.exportToExcel();
      } else if (command === 'json') {
        await this.exportToJSON();
      }
    },
    
    async exportToExcel() {
      if (!this.currentQueryTab || !this.currentQueryTab.totalResults) {
        ElMessage.warning('没有可导出的数据');
        return;
      }
      
      const tab = this.currentQueryTab;
      
      try {
        // 创建工作簿
        const wb = XLSX.utils.book_new();
        
        // 为每个平台创建单独的工作表
        Object.keys(tab.queryResults).forEach(platform => {
          const platformResults = tab.queryResults[platform] || [];
          if (platformResults.length === 0) {
            return;
          }
          
          const data = platformResults.map((item, index) => ({
            '序号': index + 1,
            'URL': item.url || '-',
            'IP': item.ip || '-',
            '端口': item.port || '-',
            '域名': item.domain || '-',
            '协议': item.protocol || '-',
            '标题': item.title || '-',
          }));
          
          const ws = XLSX.utils.json_to_sheet(data);
          
          // 设置列宽
          ws['!cols'] = [
            { wch: 8 },  // 序号
            { wch: 60 }, // URL
            { wch: 15 }, // IP
            { wch: 8 },  // 端口
            { wch: 30 }, // 域名
            { wch: 10 }, // 协议
            { wch: 40 }, // 标题
          ];
          
          // 添加工作表
          const sheetName = this.getPlatformLabel(platform);
          XLSX.utils.book_append_sheet(wb, ws, sheetName);
        });
        
        // 检查是否有数据
        if (wb.SheetNames.length === 0) {
          ElMessage.warning('没有可导出的数据');
          return;
        }
        
        // 导出文件
        const fileName = `空间测绘_${tab.queryText.substring(0, 20)}_${new Date().getTime()}.xlsx`;
        XLSX.writeFile(wb, fileName);
        
        ElMessage.success('Excel 导出成功');
      } catch (error) {
        console.error('导出失败:', error);
        ElMessage.error(`导出失败: ${error.message}`);
      }
    },
    
    async exportToJSON() {
      if (!this.currentQueryTab || !this.currentQueryTab.totalResults) {
        ElMessage.warning('没有可导出的数据');
        return;
      }
      
      const tab = this.currentQueryTab;
      
      try {
        const appPath = await GetAppPath();
        const pathSeparator = appPath.includes('\\') ? '\\' : '/';
        const exportDir = `${appPath}${pathSeparator}logs${pathSeparator}spaceMapping${pathSeparator}exports`;
        
        // 准备导出数据
        const exportData = {
          queryText: tab.queryText,
          platforms: tab.platforms,
          queryResults: tab.queryResults,
          aggregatedData: tab.aggregatedData,
          totalResults: tab.totalResults,
          exportedAt: new Date().toISOString(),
        };
        
        // 为每个平台创建单独的数据结构
        const platformData = {};
        Object.keys(tab.queryResults).forEach(platform => {
          const platformResults = tab.queryResults[platform] || [];
          if (platformResults.length > 0) {
            platformData[platform] = platformResults.map((item, index) => ({
              序号: index + 1,
              URL: item.url || '-',
              IP: item.ip || '-',
              端口: item.port || '-',
              域名: item.domain || '-',
              协议: item.protocol || '-',
              标题: item.title || '-',
            }));
          }
        });
        
        exportData.platformData = platformData;
        
        // 生成文件名
        const timestamp = Date.now();
        const fileName = `空间测绘_${tab.queryText.substring(0, 20).replace(/[^\w\s-]/g, '_')}_${timestamp}.json`;
        const filePath = `${exportDir}${pathSeparator}${fileName}`;
        
        // SaveFile 会自动创建目录（如果不存在）
        await SaveFile(filePath, JSON.stringify(exportData, null, 2));
        
        ElMessage.success(`JSON 导出成功: ${fileName}`);
      } catch (error) {
        console.error('导出失败:', error);
        ElMessage.error(`导出失败: ${error.message}`);
      }
    },
    
    handleMultiLineConfirm() {
      this.queryText = this.multiLineText;
      this.showMultiLineInput = false;
    },
  },
};
</script>

<style scoped>
.space-mapping-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: #fafbfc;
  min-height: 89vh;
}

/* 顶部工具栏 */
.top-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 24px;
  background: #ffffff;
  border-radius: 10px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  border: 1px solid #f0f0f0;
}

.module-title {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 20px;
  font-weight: 500;
  color: #2c3e50;
  letter-spacing: 0.3px;
}

.title-icon {
  font-size: 22px;
  color: #67C23A;
}

.toolbar-actions {
  display: flex;
  gap: 12px;
}

.toolbar-actions .el-button {
  border-radius: 6px;
  font-weight: 400;
  transition: all 0.2s ease;
  border: 1px solid #e4e7ed;
}

.toolbar-actions .el-button:hover {
  border-color: #67C23A;
  color: #67C23A;
  background: #f0f9ff;
}

/* 搜索栏样式 */
.search-bar-card {
  background: #ffffff;
  border-radius: 10px;
  padding: 5px 10px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  border: 1px solid #f0f0f0;
}

.search-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.search-left {
  display: flex;
  align-items: center;
  gap: 14px;
  flex: 1;
  min-width: 400px;
}

.search-right {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-shrink: 0;
}

.platform-select {
  width: 160px;
}

.platform-select :deep(.el-input__wrapper) {
  border-radius: 6px;
  border: 1px solid #e4e7ed;
  transition: all 0.2s ease;
}

.platform-select :deep(.el-input__wrapper:hover) {
  border-color: #67C23A;
}

.platform-select :deep(.el-input__wrapper.is-focus) {
  border-color: #67C23A;
  box-shadow: 0 0 0 2px rgba(103, 194, 58, 0.1);
}

/* 自定义多选标签显示 */
.platform-select :deep(.el-select__tags) {
  max-width: 100%;
}

.platform-select :deep(.el-tag) {
  background-color: #67C23A;
  border-color: #67C23A;
  color: #ffffff;
  font-weight: 500;
  margin-right: 6px;
}

.platform-select :deep(.el-tag__close) {
  color: #ffffff;
  background-color: rgba(255, 255, 255, 0.2);
}

.platform-select :deep(.el-tag__close:hover) {
  background-color: rgba(255, 255, 255, 0.3);
}

.count-display {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  background: #67C23A;
  border-radius: 6px;
  color: white;
  font-size: 13px;
  white-space: nowrap;
  box-shadow: 0 2px 4px rgba(103, 194, 58, 0.2);
}

.count-label {
  opacity: 0.9;
}

.count-value {
  font-weight: 700;
  font-size: 18px;
}

.query-input {
  flex: 1;
  min-width: 300px;
}

.query-input :deep(.el-input__wrapper) {
  border-radius: 6px;
  border: 1px solid #e4e7ed;
  transition: all 0.2s ease;
}

.query-input :deep(.el-input__wrapper:hover) {
  border-color: #67C23A;
}

.query-input :deep(.el-input__wrapper.is-focus) {
  border-color: #67C23A;
  box-shadow: 0 0 0 2px rgba(103, 194, 58, 0.1);
}

.latest-data-checkbox {
  white-space: nowrap;
}

.search-btn {
  border-radius: 6px;
  font-weight: 400;
  padding: 10px 24px;
  transition: all 0.2s ease;
  background: #67C23A;
  border: none;
}

.search-btn:hover {
  background: #85ce61;
  box-shadow: 0 2px 8px rgba(103, 194, 58, 0.3);
}

/* 查询标签行 */
.query-tag-row {
  margin-top: -4px;
  padding: 0 4px;
}

.query-tag-row :deep(.el-tag) {
  border-radius: 6px;
  padding: 6px 14px;
  font-size: 13px;
  border: 1px solid #e4e7ed;
  background: #f5f7fa;
}

/* 主内容区域 */
.main-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: #ffffff;
  border-radius: 10px;
  padding: 5px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  border: 1px solid #f0f0f0;
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
}

.main-tabs :deep(.el-tabs__item) {
  border-radius: 0;
  font-weight: 400;
  transition: all 0.2s ease;
  color: #606266;
  padding: 12px 20px;
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

.syntax-table {
  font-size: 13px;
  width: 100%;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #f0f0f0;
}

.syntax-table :deep(.el-table__header) {
  background: #f5f7fa;
}

.syntax-table :deep(.el-table__header th) {
  background: transparent;
  color: #2c3e50;
  font-weight: 500;
  border-bottom: 1px solid #e4e7ed;
  padding: 12px 0;
}

.syntax-table :deep(.el-table__body tr:hover) {
  background: #f0f9ff;
}

.syntax-cell {
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s ease;
  display: inline-block;
  width: 100%;
  font-family: 'Courier New', monospace;
  color: #67C23A;
}

.syntax-cell:hover {
  background: #f0f9ff;
  color: #67C23A;
}

.results-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  border-radius: 8px;
  overflow: hidden;
}

.results-tabs :deep(.el-tabs__header) {
  margin: 0;
  background: rgba(255, 255, 255, 0.5);
  border-radius: 8px 8px 0 0;
}

.results-tabs :deep(.el-tabs__item) {
  border-radius: 8px 8px 0 0;
  font-weight: 500;
}

.results-tabs :deep(.el-tabs__content) {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding: 0;
  background: transparent;
}

.results-tabs :deep(.el-tab-pane) {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.results-table {
  width: 100%;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #f0f0f0;
}

.results-table :deep(.el-table__header) {
  background: #f5f7fa;
}

.results-table :deep(.el-table__header th) {
  background: transparent;
  color: #2c3e50;
  font-weight: 500;
  border-bottom: 1px solid #e4e7ed;
  padding: 12px 0;
}

.results-table :deep(.el-table__body tr:hover) {
  background: #f0f9ff;
}

.url-link {
  cursor: pointer;
}

.platform-tag {
  margin-right: 5px;
}

/* 分页样式 */
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

/* API配置对话框 */
.api-config-content {
  /* 限制对话框内容高度，避免撑开页面导致外层出现滚动条 */
  max-height: min(500px, calc(100vh - 320px));
  overflow-y: auto;
}

.api-config-item {
  margin-bottom: 16px;
  padding: 18px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #ffffff;
  transition: all 0.2s ease;
}

.api-config-item:hover {
  border-color: #67C23A;
  box-shadow: 0 2px 8px rgba(103, 194, 58, 0.1);
}

.api-config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.api-name {
  font-weight: bold;
}

.api-config-body {
  display: flex;
  align-items: center;
  gap: 10px;
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

.api-key-input {
  flex: 1;
}

.eye-icon {
  cursor: pointer;
}

.api-status {
  min-width: 60px;
  text-align: center;
}

.form-item-tip {
  margin-left: 10px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

/* Tooltip样式修复 */
:deep(.el-tooltip__popper) {
  max-width: 500px;
  word-break: break-all;
}

/* 空状态样式 */
.empty-tabs-container {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 70vh;
  background: #fafbfc;
  border: 1px dashed #e4e7ed;
  border-radius: 8px;
}

/* 查询语法对话框 */
.syntax-dialog :deep(.el-dialog) {
  border-radius: 12px;
  overflow: hidden;
}

.syntax-dialog :deep(.el-dialog__header) {
  background: #ffffff;
  color: #2c3e50;
  padding: 5px 10px;
  margin: 0;
  border-bottom: 1px solid #f0f0f0;
}

.syntax-dialog :deep(.el-dialog__title) {
  color: #2c3e50;
  font-weight: 500;
  font-size: 16px;
}

.syntax-dialog :deep(.el-dialog__headerbtn .el-dialog__close) {
  color: #909399;
  font-size: 18px;
}

.syntax-dialog :deep(.el-dialog__body) {
  padding: 20px;
  background: rgba(255, 255, 255, 0.95);
}

/* 标签页样式 */
.tab-label {
  display: flex;
  align-items: center;
  gap: 6px;
}

.tab-name {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tab-badge {
  margin-left: 4px;
}

.glass-card {
  background: var(--el-bg-color);
  border-radius: 4px;
}

/* 按钮样式优化 */
.el-button {
  border-radius: 8px;
  transition: all 0.3s ease;
}

.el-button:hover {
  transform: translateY(-2px);
}

.el-button--primary {
  background: #67C23A;
  border: none;
}

.el-button--primary:hover {
  background: #85ce61;
  box-shadow: 0 2px 8px rgba(103, 194, 58, 0.3);
}

.el-button--success {
  background: #67C23A;
  border: none;
}

.el-button--success:hover {
  background: #85ce61;
  box-shadow: 0 2px 8px rgba(103, 194, 58, 0.3);
}

/* 对话框样式优化 */
:deep(.el-dialog) {
  border-radius: 12px;
  overflow: hidden;
}

:deep(.el-dialog__header) {
  padding: 20px 24px;
  background: #ffffff;
  color: #2c3e50;
  margin: 0;
  border-bottom: 1px solid #f0f0f0;
}

:deep(.el-dialog__title) {
  color: #2c3e50;
  font-weight: 500;
  font-size: 16px;
}

:deep(.el-dialog__headerbtn .el-dialog__close) {
  color: #909399;
  font-size: 18px;
}

:deep(.el-dialog__headerbtn:hover .el-dialog__close) {
  color: rgba(255, 255, 255, 0.8);
}

:deep(.el-dialog__body) {
  padding: 5px;
  background: rgba(255, 255, 255, 0.95);
}

/* 输入框和选择器优化 */
:deep(.el-input__wrapper) {
  border-radius: 8px;
  transition: all 0.3s ease;
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

:deep(.el-select .el-input__wrapper) {
  border-radius: 8px;
}

/* 表格优化 */
:deep(.el-table) {
  border-radius: 8px;
  overflow: hidden;
}

:deep(.el-table th.el-table__cell) {
  background: #f5f7fa;
  color: #2c3e50;
  font-weight: 500;
  border-bottom: 1px solid #e4e7ed;
}

:deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background: #fafbfc;
}

:deep(.el-table__body tr:hover > td) {
  background: #f0f9ff !important;
}

/* 标签页优化 */
:deep(.el-tabs__item) {
  font-weight: 500;
  transition: all 0.3s ease;
}

:deep(.el-tabs__item.is-active) {
  color: var(--el-color-primary);
  font-weight: 600;
}

:deep(.el-tabs__active-bar) {
  background: #67C23A;
  height: 2px;
}

/* 复选框和开关优化 */
:deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background: #67C23A;
  border-color: #67C23A;
}

/* 分页器优化 */
:deep(.el-pagination .btn-next),
:deep(.el-pagination .btn-prev),
:deep(.el-pagination .number) {
  border-radius: 6px;
  transition: all 0.3s ease;
}

:deep(.el-pagination .number.is-active) {
  background: #67C23A;
  color: white;
  border-color: transparent;
}

:deep(.el-pagination .number:hover) {
  color: var(--el-color-primary);
  transform: translateY(-2px);
}

/* 查询中状态样式 */
.querying-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 400px;
  gap: 20px;
}

.querying-icon {
  font-size: 48px;
  color: #67C23A;
  animation: rotating 2s linear infinite;
}

.querying-text {
  font-size: 16px;
  color: #606266;
  font-weight: 500;
}

@keyframes rotating {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
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

.context-menu-item.danger {
  color: #f56c6c;
}

.context-menu-item.danger:hover {
  background-color: #fef0f0;
}

.context-menu-item .el-icon {
  font-size: 16px;
}

.context-menu-divider {
  height: 1px;
  background: #e4e7ed;
  margin: 4px 0;
}
</style>
