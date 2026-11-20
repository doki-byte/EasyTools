<template>
  <div class="restmate-page no-theme-bg restmate-page--light-surface">
    <div class="restmate-top-shell no-theme-bg">
      <el-tabs
        v-model="restmateWorkspace"
        class="et-module-tabs et-module-tabs--header-only restmate-workspace-tabs no-theme-bg"
        :style="tabsStyle"
      >
        <el-tab-pane label="HTTP 工作台" name="composer" />
        <el-tab-pane label="数据包发包" name="rawPacket" />
      </el-tabs>
      <p class="restmate-workspace-hint">{{ restmateWorkspaceHint }}</p>
    </div>

    <el-container v-show="restmateWorkspace === 'composer'" class="restmate-container restmate-workbench no-theme-bg">
      <!-- 左侧集合列表 -->
      <el-aside :width="asideCollapsed ? '42px' : '260px'" class="restmate-aside">
        <div class="aside-header">
          <span class="aside-title" v-if="!asideCollapsed">请求集合</span>
          <div class="aside-actions">
            <el-button size="small" class="aside-toggle-btn" @click="asideCollapsed = !asideCollapsed">
              <el-icon v-if="asideCollapsed">
                <DArrowRight />
              </el-icon>
              <el-icon v-else>
                <DArrowLeft />
              </el-icon>
            </el-button>
            <el-tooltip v-if="!asideCollapsed" content="新增集合" placement="top">
              <el-button circle size="small" type="primary" @click="handleAddCollection">
                <el-icon>
                  <Plus />
                </el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip v-if="!asideCollapsed" content="导入集合" placement="top">
              <el-button circle size="small" @click="handleImportCollection">
                <el-icon>
                  <Upload />
                </el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip v-if="!asideCollapsed" content="从URL导入Swagger" placement="top">
              <el-button circle size="small" @click="handleImportSwaggerFromUrl">
                <el-icon>
                  <Link />
                </el-icon>
              </el-button>
            </el-tooltip>
            <el-tooltip v-if="!asideCollapsed" content="刷新集合列表" placement="top">
              <el-button circle size="small" @click="handleRefreshCollections" :loading="collectionsLoading">
                <el-icon>
                  <Refresh />
                </el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </div>

        <el-scrollbar v-if="!asideCollapsed" class="aside-scroll">
          <!-- 加载中界面 -->
          <div v-if="collectionsLoading" class="collections-loading">
            <el-icon class="is-loading" :size="24">
              <Loading />
            </el-icon>
            <p class="loading-text">正在加载接口集合...</p>
          </div>
          <!-- 正常显示集合列表 -->
          <el-collapse v-else v-model="activeCollectionNames">
            <el-collapse-item v-for="coll in collections" :key="coll.id" :name="coll.id">
              <template #title>
                <div class="coll-title">
                  <span class="coll-name">
                    {{ coll.name }}
                    <span v-if="coll.requestsCount !== undefined && coll.requestsCount > 0" class="coll-request-count">
                      ({{ coll.requestsCount }})
                    </span>
                  </span>
                  <el-dropdown trigger="click" @command="cmd => handleCollectionCommand(cmd, coll)">
                    <span class="el-dropdown-link" @click.stop>
                      <el-icon>
                        <MoreFilled />
                      </el-icon>
                    </span>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="batchTest">批量测试</el-dropdown-item>
                        <el-dropdown-item command="rename">重命名</el-dropdown-item>
                        <el-dropdown-item command="export">导出</el-dropdown-item>
                        <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </template>

              <!-- 使用 v-if 优化性能，只在展开时渲染表格 -->
              <!-- 优化：限制表格高度，使用滚动，避免一次性渲染过多DOM导致卡顿 -->
              <!-- 显示加载状态 -->
              <div v-if="isCollectionLoading(coll.id)" class="collection-requests-loading">
                <el-icon class="is-loading" :size="16">
                  <Loading />
                </el-icon>
                <span style="margin-left: 8px; font-size: 12px; color: #909399;">正在加载请求列表...</span>
              </div>
              <div v-else-if="activeCollectionNames && activeCollectionNames.includes(coll.id)" class="collection-requests-wrapper">
                <el-table 
                  :data="getDisplayedRequests(coll)" 
                  size="small" 
                  :border="false" 
                  :show-header="false" 
                  :row-class-name="({ row }) => requestRowClass(coll, row)" 
                  class="coll-req-table glass-card"
                  :max-height="400"
                  @row-click="row => handleRequestRowClick(coll, row)">
                  <el-table-column width="70">
                    <template #default="{ row }">
                      <span class="req-method" :class="`m-${(row.method || '').toLowerCase()}`">
                        {{ (row.method || 'GET').toUpperCase() }}
                      </span>
                    </template>
                  </el-table-column>
                  <el-table-column>
                    <template #default="{ row }">
                      <span class="req-name" :title="row.name || row.url">
                        {{ row.name || row.url || '未命名请求' }}
                      </span>
                    </template>
                  </el-table-column>
                  <el-table-column width="40" align="right">
                    <template #default="{ row }">
                      <el-dropdown trigger="click" @command="cmd => handleRequestCommand(cmd, coll, row)">
                        <span class="el-dropdown-link" @click.stop>
                          <el-icon>
                            <MoreFilled />
                          </el-icon>
                        </span>
                        <template #dropdown>
                          <el-dropdown-menu>
                            <el-dropdown-item command="rename">重命名</el-dropdown-item>
                            <el-dropdown-item command="duplicate">复制</el-dropdown-item>
                            <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                          </el-dropdown-menu>
                        </template>
                      </el-dropdown>
                    </template>
                  </el-table-column>
                </el-table>
                <!-- 加载更多按钮 -->
                <div v-if="hasMoreRequests(coll)" class="load-more-wrapper">
                  <el-button 
                    size="small" 
                    text 
                    type="primary" 
                    @click="loadMoreRequests(coll)"
                    class="load-more-btn"
                  >
                    加载更多 ({{ getRemainingCount(coll) }} 个)
                  </el-button>
                </div>
              </div>
            </el-collapse-item>
          </el-collapse>
        </el-scrollbar>
      </el-aside>

      <!-- 右侧主区域 -->
      <el-container>
        <!-- 顶部：标签页 + 环境变量 -->
        <div class="top-tabs-bar">
          <div class="tabs-left">
            <el-select v-model="selectedEnvId" placeholder="无环境变量" size="small" class="env-select glass-card"
              @change="handleEnvChange">
              <el-option label="无环境变量" :value="''" />
              <el-option v-for="env in envs" :key="env.id" :label="env.name + (env.selected ? '（当前）' : '')"
                :value="env.id" />
            </el-select>
            <el-tooltip content="管理环境变量" placement="bottom">
              <el-button circle size="small" class="env-setting-btn" @click="envDrawerVisible = true">
                <el-icon>
                  <Setting />
                </el-icon>
              </el-button>
            </el-tooltip>
          </div>

          <div class="tabs-right">
            <el-tabs
              ref="requestTabs"
              v-model="activeTabId"
              type="card"
              class="request-tabs"
              @tab-remove="handleTabRemove"
              @tab-click="handleTabClick" @tab-add="handleCreateRequest" addable @contextmenu.prevent="handleTabsContextMenu">
              <el-tab-pane v-for="tab in tabs" :key="tab.id" :name="tab.id" :label="tab.title"
                :closable="tabs.length > 1" />
            </el-tabs>
            <!-- 右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
            <Teleport to="body">
              <ul class="tabs-context-menu" v-if="tabsContextMenuVisible" :style="tabsContextMenuStyle">
                <li @click="handleCloseOtherTabs">关闭其他</li>
                <li @click="handleCloseAllTabs">全部关闭</li>
              </ul>
            </Teleport>
          </div>
        </div>

        <!-- 请求编辑区：仅保留方式 / URL / 发送 / 保存 -->
        <el-header class="restmate-header" height="50px">
          <div class="header-row">
            <el-select 
              v-model="currentRequest.method" 
              size="small" 
              :class="['method-select', `method-${(currentRequest.method || 'get').toLowerCase()}`]"
            >
              <el-option v-for="m in httpMethods" :key="m" :label="m" :value="m.toLowerCase()" />
            </el-select>

            <env-var-input :key="`url-${currentRequest.id || 'new'}-${activeTabId}`" v-model="currentRequest.url"
              placeholder="输入请求 URL，例如：https://example.com/api" size="small" class="url-input"
              :env-vars="currentEnvVars" @keydown="handleUrlKeydown" />

            <el-button
              :type="sending ? 'danger' : 'primary'"
              size="small"
              class="send-btn"
              :loading="cancelingRequest"
              :disabled="sending && cancelingRequest"
              @click="sending ? handleCancelRequest() : handleSend()"
            >
              {{ sending ? '取消请求' : '发送请求' }}
            </el-button>
            <el-button size="small" :loading="saving" class="save-btn-inline" @click="openSaveDialog">
              保存
            </el-button>
            <!-- <span class="header-inline-hint" v-if="restmateWorkspace === 'composer'">更多发包方式请切到上方「数据包发包」</span> -->
          </div>
        </el-header>

        <!-- 主体 Tabs：请求编辑 + 响应 -->
        <el-main class="restmate-main">
          <!-- 文件上传 Bypass 测试结果视图 - 全屏显示 -->
          <el-card v-if="currentTab && currentTab.isBypassTest" class="full-card batch-test-full-card" shadow="never">
            <template #header>
              <div class="card-header space-between">
                <span>文件上传 Bypass 测试结果</span>
                <div style="display: flex; align-items: center; gap: 12px;">
                  <div class="resp-meta" v-if="currentTab.bypassTestResults">
                    <el-tag size="small" type="info">
                      总计: {{ currentTab.bypassTestResults.total }}
                    </el-tag>
                    <el-tag size="small" type="success" class="ml-4">
                      成功: {{ currentTab.bypassTestResults.success }}
                    </el-tag>
                    <el-tag size="small" type="danger" class="ml-4">
                      失败: {{ currentTab.bypassTestResults.failed }}
                    </el-tag>
                    <el-tag v-if="currentTab.bypassTesting" size="small" type="warning" class="ml-4">
                      测试中...
                    </el-tag>
                  </div>
                  <el-button 
                    v-if="currentTab.bypassTesting" 
                    size="small" 
                    type="danger" 
                    @click="cancelBypassTest"
                  >
                    取消测试
                  </el-button>
                </div>
              </div>
            </template>
            <div v-if="currentTab.bypassTestResults && currentTab.bypassTestResults.results && currentTab.bypassTestResults.results.length > 0" class="batch-test-results-wrapper">
              <el-scrollbar class="batch-test-scrollbar">
                <el-table :data="currentTab.bypassTestResults.results" size="small" border height="100%" class="glass-card">
                  <el-table-column prop="index" label="ID" width="60" align="center" />
                  <el-table-column prop="status" label="状态码" width="60" align="center">
                    <template #default="{ row }">
                      <el-tag v-if="row.status >= 200 && row.status < 300" type="success" size="small">
                        {{ row.status }}
                      </el-tag>
                      <el-tag v-else-if="row.status === 0" type="danger" size="small">
                        错误
                      </el-tag>
                      <el-tag v-else type="warning" size="small">
                        {{ row.status }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="payload" label="Payload" min-width="300" show-overflow-tooltip>
                    <template #default="{ row }">
                      <span style="font-family: monospace; font-size: 12px;">
                        {{ row.payload.substring(0, 100) }}{{ row.payload.length > 100 ? '...' : '' }}
                      </span>
                    </template>
                  </el-table-column>
                  <el-table-column prop="responseSize" label="响应大小" width="100" align="center">
                    <template #default="{ row }">
                      {{ (row.responseSize / 1024).toFixed(2) }} KB
                    </template>
                  </el-table-column>
                  <el-table-column prop="response" label="响应预览" min-width="200" show-overflow-tooltip>
                    <template #default="{ row }">
                      <span style="font-family: monospace; font-size: 12px;">
                        {{ row.response.substring(0, 50) }}{{ row.response.length > 50 ? '...' : '' }}
                      </span>
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="120" align="center" fixed="right">
                    <template #default="{ row }">
                      <el-button 
                        size="small" 
                        text 
                        type="primary" 
                        @click="viewBypassResult(row)" 
                        v-if="row.requestPacket || row.responsePacket"
                      >
                        查看详情
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </el-scrollbar>
            </div>
            <div v-else-if="currentTab.bypassTesting" class="empty-tip">
              <el-icon class="is-loading" :size="32">
                <Loading />
              </el-icon>
              <p style="margin-top: 16px;">正在测试中...</p>
            </div>
            <div v-else class="empty-tip">
              暂无测试结果
            </div>
          </el-card>

          <!-- 批量测试结果视图 - 全屏显示 -->
          <el-card v-else-if="currentTab && currentTab.isBatchTest" class="full-card batch-test-full-card" shadow="never">
            <template #header>
              <div class="card-header space-between">
                <span>批量测试结果</span>
                <div style="display: flex; align-items: center; gap: 12px;">
                  <div class="resp-meta" v-if="currentTab.batchTestResults">
                    <el-tag size="small" type="info">
                      总计: {{ currentTab.batchTestResults.total }}
                    </el-tag>
                    <el-tag size="small" type="success" class="ml-4">
                      成功: {{ currentTab.batchTestResults.success }}
                    </el-tag>
                    <el-tag size="small" type="danger" class="ml-4">
                      失败: {{ currentTab.batchTestResults.failed }}
                    </el-tag>
                  </div>
                  <el-button 
                    v-if="currentTab.batchTesting" 
                    size="small" 
                    type="danger" 
                    @click="cancelBatchTest"
                  >
                    取消测试
                  </el-button>
                </div>
              </div>
            </template>
            <div v-if="currentTab.batchTestRequests && currentTab.batchTestRequests.length > 0" class="batch-test-results-wrapper">
              <el-scrollbar class="batch-test-scrollbar">
                <el-table :data="currentTab.batchTestRequests" size="small" border height="100%" class="glass-card">
                  <el-table-column type="index" label="#" width="50" align="center" />
                  <el-table-column prop="method" label="方法" width="80" align="center">
                    <template #default="{ row }">
                      <el-tag :type="getMethodTagType(row.method)" size="small">
                        {{ (row.method || 'GET').toUpperCase() }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="name" label="请求名称" min-width="200" show-overflow-tooltip />
                  <el-table-column prop="url" label="URL" min-width="280" show-overflow-tooltip>
                    <template #default="{ row }">
                      <span style="cursor: pointer; color: #409eff;" @click.stop="viewBatchTestPacket(row)">
                        {{ row.url }}
                      </span>
                    </template>
                  </el-table-column>
                  <el-table-column label="状态" width="100" align="center">
                    <template #default="{ row }">
                      <el-tag 
                        v-if="row.status === 'success'" 
                        type="success" 
                        size="small"
                      >
                        成功
                      </el-tag>
                      <el-tag 
                        v-else-if="row.status === 'failed'" 
                        type="danger" 
                        size="small"
                      >
                        失败
                      </el-tag>
                      <el-tag 
                        v-else-if="row.status === 'testing'" 
                        type="warning" 
                        size="small"
                      >
                        测试中
                      </el-tag>
                      <el-tag 
                        v-else 
                        type="info" 
                        size="small"
                      >
                        等待中
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="状态码" width="100" align="center">
                    <template #default="{ row }">
                      <span v-if="row.result && row.result.statusCode">{{ row.result.statusCode }}</span>
                      <span v-else>-</span>
                    </template>
                  </el-table-column>
                  <el-table-column label="耗时" width="100" align="center">
                    <template #default="{ row }">
                      <span v-if="row.result && row.result.duration">{{ row.result.duration }}</span>
                      <span v-else>-</span>
                    </template>
                  </el-table-column>
                  <el-table-column label="错误信息" min-width="200">
                    <template #default="{ row }">
                      <el-tooltip
                        v-if="row.error || (row.result && row.result.errorContent)"
                        :content="row.error || (row.result && row.result.errorContent) || ''"
                        placement="top"
                        effect="dark"
                      >
                        <span style="color: #f56c6c; display: inline-block; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ row.error || (row.result && row.result.errorContent) || '-' }}</span>
                      </el-tooltip>
                      <span v-else>-</span>
                    </template>
                  </el-table-column>
                  <el-table-column label="操作" width="120" align="center" fixed="right">
                    <template #default="{ row }">
                      <el-button 
                        size="small" 
                        text 
                        type="primary" 
                        @click="viewBatchTestPacket(row)" 
                        v-if="row.requestPacket || row.responsePacket || row.result"
                      >
                        查看详情
                      </el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </el-scrollbar>
            </div>
            <div v-else class="empty-tip">
              暂无测试请求
            </div>
          </el-card>
          
          <!-- 普通请求编辑 + 响应视图 -->
          <el-row v-else :gutter="8" class="full-height-row">
            <!-- 请求编辑 -->
            <el-col :span="12" class="full-height-col">
              <!-- 普通请求编辑视图 -->
              <el-card class="full-card" shadow="never">
                <template #header>
                  <div class="card-header">
                    <span>请求详情</span>
                  </div>
                </template>

                <el-tabs v-model="activeReqTab" class="req-tabs">
                  <el-tab-pane label="查询参数" name="params">
                    <kv-table
                      :key="`params-${currentRequest.id || 'new'}-${activeTabId}-${currentRequest.params?.length || 0}`"
                      v-model:list="currentRequest.params" add-label="新增查询参数" :env-vars="currentEnvVars" />
                  </el-tab-pane>

                  <el-tab-pane label="请求头" name="headers">
                    <kv-table
                      :key="`headers-${currentRequest.id || 'new'}-${activeTabId}-${currentRequest.headers?.length || 0}`"
                      v-model:list="currentRequest.headers" add-label="新增请求头" :env-vars="currentEnvVars" />
                  </el-tab-pane>

                  <el-tab-pane label="请求体" name="body">
                    <div class="body-toolbar">
                      <el-radio-group v-model="currentRequest.body.bodyType" size="small">
                        <el-radio-button label="json">JSON</el-radio-button>
                        <el-radio-button label="formdata">FormData</el-radio-button>
                        <el-radio-button label="urlencoded">x-www-form-urlencoded</el-radio-button>
                        <el-radio-button label="raw">RAW</el-radio-button>
                      </el-radio-group>
                      <div v-if="currentRequest.body.bodyType === 'json'" class="body-toolbar-actions">
                        <el-button size="small" text @click="formatRequestBodyJson">
                          格式化 JSON
                        </el-button>
                      </div>
                    </div>

                    <div class="mt-8 body-json-wrapper body-json-editor" v-if="currentRequest.body.bodyType === 'json'">
                      <VueMonacoEditor
                        ref="jsonBodyEditorRef"
                        :key="`body-${currentRequest.id || 'new'}-${activeTabId}`"
                        v-model:value="currentRequest.body.bodyRaw"
                        language="json"
                        theme="http-dark-theme"
                        :height="responseEditorHeight"
                        class="code-editor"
                        :options="monacoEditOptions"
                        placeholder="输入 JSON 请求体，支持自动格式化"
                      />
                    </div>

                    <div class="mt-8" v-else-if="currentRequest.body.bodyType === 'formdata'">
                      <formdata-table :key="`formdata-${currentRequest.id || 'new'}-${activeTabId}`" v-model:list="currentRequest.body.formData" @choose-file="handleChooseFile" />
                    </div>

                    <div class="mt-8" v-else-if="currentRequest.body.bodyType === 'urlencoded'">
                      <formdata-table :key="`urlencoded-${currentRequest.id || 'new'}-${activeTabId}`" v-model:list="currentRequest.body.formData" :allow-files="false" />
                    </div>

                    <div class="mt-8 body-json-wrapper" v-else-if="currentRequest.body.bodyType === 'raw'">
                      <VueMonacoEditor
                        ref="rawBodyEditorRef"
                        :key="`raw-${currentRequest.id || 'new'}-${activeTabId}`"
                        v-model:value="currentRequest.body.bodyRaw"
                        :language="requestRawLanguage"
                        theme="http-dark-theme"
                        :height="responseEditorHeight"
                        class="code-editor"
                        :options="monacoEditOptions"
                        placeholder="保持原始请求体，不改写 Content-Type"
                      />
                    </div>
                  </el-tab-pane>
                </el-tabs>
              </el-card>
            </el-col>

            <!-- 响应展示 -->
            <el-col :span="12" class="full-height-col">
              <!-- 普通响应视图 -->
              <el-card class="full-card" shadow="never">
                <template #header>
                  <div class="card-header space-between">
                    <span>响应结果</span>
                    <div class="resp-actions" v-if="hasPacketForCurrentTab">
                      <el-button size="small" text type="primary" @click="openSendPacketViewer">
                        查看数据包
                      </el-button>
                    </div>
                    <div class="resp-meta" v-if="response">
                      <el-tag size="small"
                        :type="response.statusCode >= 200 && response.statusCode < 300 ? 'success' : 'danger'">
                        {{ response.httpStatus || response.statusCode }}
                      </el-tag>
                      <el-tag size="small" type="info" class="ml-4">
                        {{ response.duration }} ms
                      </el-tag>
                      <el-tag size="small" class="ml-4">
                        {{ response.contentType }}
                      </el-tag>
                    </div>
                  </div>
                </template>

                <div v-if="!response && !responseError" class="resp-empty-placeholder">
                  <el-icon class="resp-empty-icon" :size="52">
                    <Promotion />
                  </el-icon>
                  <p class="resp-empty-title">请先发送请求</p>
                  <p class="resp-empty-desc">
                    在上方的 URL 栏填写地址并点击左侧醒目的「发送请求」，响应内容将出现在这里。
                  </p>
                </div>

                <div v-else class="resp-wrapper">
                  <el-tabs v-model="activeRespTab" class="resp-tabs">
                    <el-tab-pane label="响应体" name="body">
                      <div class="resp-toolbar" v-if="response && !responseError">
                        <el-button size="small" text :type="respViewMode === 'pretty' ? 'primary' : 'default'"
                          @click="respViewMode = 'pretty'">
                          高亮+格式化
                        </el-button>
                        <el-button size="small" text :type="respViewMode === 'raw' ? 'primary' : 'default'"
                          @click="respViewMode = 'raw'">
                          原始内容
                        </el-button>
                        <el-button size="small" text @click="copyResponseBody">
                          复制响应
                        </el-button>
                        <el-button 
                          v-if="response.isBinary && response.binaryData" 
                          size="small" 
                          text 
                          type="primary"
                          @click="saveBinaryResponse"
                        >
                          保存文件
                        </el-button>
                      </div>
                      <div class="resp-body" v-if="!responseError">
                        <VueMonacoEditor
                          ref="responseBodyEditorRef"
                          :key="`resp-${respViewMode}-${activeTabId}`"
                          :value="responseEditorValue"
                          :language="responseBodyLanguage"
                          theme="http-dark-theme"
                          :height="responseEditorHeight"
                          class="code-editor"
                          :options="monacoReadOnlyOptions"
                        />
                      </div>
                      <el-alert v-else type="error" :closable="false" show-icon :description="responseError">
                        请求失败
                      </el-alert>
                    </el-tab-pane>

                    <el-tab-pane label="响应头" name="headers">
                      <el-table v-if="response" :data="response.headers || []" size="small" border height="100%">
                        <el-table-column prop="key" label="Header" width="180" />
                        <el-table-column prop="value" label="Value" />
                      </el-table>
                    </el-tab-pane>

                    <el-tab-pane label="网页预览" name="preview">
                      <div v-if="canPreviewHtml" class="preview-wrapper">
                        <iframe 
                          class="preview-frame" 
                          :srcdoc="response.bodyContent"
                          sandbox="allow-same-origin allow-scripts"
                        />
                      </div>
                      <div v-else-if="response && response.statusCode >= 300 && response.statusCode < 400" class="empty-tip">
                        重定向响应（{{ response.statusCode }}），无法在iframe中预览。请查看响应头中的 Location 字段。
                      </div>
                      <div v-else class="empty-tip">
                        当前响应不是 HTML，无法预览。
                      </div>
                    </el-tab-pane>

                    <!-- <el-tab-pane label="数据包" name="packets">
                      <div v-if="hasPacketForCurrentTab" class="packet-pane">
                        <div class="packet-toolbar">
                          <el-button size="small" text @click="copyPacket('request')">复制请求包</el-button>
                          <el-button size="small" text @click="copyPacket('response')">复制响应包</el-button>
                        </div>
                        <el-row :gutter="8" class="packet-row">
                          <el-col :span="12" class="packet-col">
                            <div class="packet-title">请求包</div>
                            <VueMonacoEditor
                              :key="`req-packet-${activeTabId}`"
                              :value="currentRequestPacket || '暂无请求数据包'"
                              language="http"
                              theme="http-dark-theme"
                              :height="responseEditorHeight"
                              class="code-editor"
                              :options="monacoReadOnlyOptions"
                            />
                          </el-col>
                          <el-col :span="12" class="packet-col">
                            <div class="packet-title">响应包</div>
                            <VueMonacoEditor
                              :key="`resp-packet-${activeTabId}`"
                              :value="currentResponsePacket || '暂无响应数据包'"
                              language="http"
                              theme="http-dark-theme"
                              :height="responseEditorHeight"
                              class="code-editor"
                              :options="monacoReadOnlyOptions"
                            />
                          </el-col>
                        </el-row>
                      </div>
                      <div v-else class="empty-tip">
                        暂无数据包，请先发送请求。
                      </div>
                    </el-tab-pane> -->
                  </el-tabs>
                </div>
              </el-card>
            </el-col>
          </el-row>
        </el-main>
      </el-container>
    </el-container>

      <!-- 数据包发包：与查看器一致的并排编辑器（高亮 / 预览 / 复制） -->
      <div
        v-show="restmateWorkspace === 'rawPacket'"
        class="restmate-workspace restmate-workspace-raw restmate-workspace-shell restmate-workspace-shell--raw no-theme-bg"
      >
        <div class="restmate-embedded-toolbar">
          <el-button size="small" type="primary" plain @click="syncComposerPacketToRawWorkspace">
            代入 HTTP 工作台当前请求
          </el-button>
          <el-button size="small" plain @click="resetEmbeddedRawPacket">清空</el-button>
          <!-- <el-button text type="primary" size="small" @click="restmateWorkspace = 'composer'">
            HTTP 工作台
          </el-button> -->
        </div>
        <PacketViewer
          embedded
          :model-value="true"
          v-model:request-packet="packetInputForm.requestPacket"
          v-model:response-packet="packetInputForm.responsePacket"
          title="数据包发包"
          :editable="true"
          :enable-save="false"
          :loading="packetViewerLoading"
          :canceling="packetCancelingRequest"
          :show-send-actions="true"
          @send="handleRawWorkspaceSend"
          @cancel="handlePacketCancel"
          @upload-bypass="handleUploadBypass"
        />
      </div>

      <!-- 环境变量管理对话框 -->
      <el-dialog v-model="envDrawerVisible" title="环境变量管理" width="600px" :close-on-click-modal="false">
        <div class="env-manage">
          <div class="env-list-header">
            <span class="aside-title">环境列表</span>
            <div>
              <el-button size="small" type="primary" @click="handleAddEnv">新增环境</el-button>
            </div>
          </div>

          <el-scrollbar class="env-list-scroll">
            <el-collapse v-model="editingEnvId">
              <el-collapse-item v-for="env in envs" :key="env.id" :name="env.id">
                <template #title>
                  <div class="env-item-title">
                    <span>{{ env.name }} <span v-if="env.selected" class="env-badge">当前</span></span>
                    <div class="env-actions">
                      <el-button size="small" link type="primary" @click.stop="setCurrentEnv(env)">
                        设为当前
                      </el-button>
                      <el-button size="small" link type="primary" @click.stop="duplicateEnv(env)">
                        复制
                      </el-button>
                      <el-button size="small" link type="primary" @click.stop="renameEnv(env)">
                        重命名
                      </el-button>
                      <el-button size="small" link type="danger" @click.stop="deleteEnv(env)">
                        删除
                      </el-button>
                    </div>
                  </div>
                </template>

                <kv-table :list="envVariablePairsWithTemp(env)" add-label="新增变量" :env-vars="{}"
                  @update:list="vals => handleEnvVarListChange(env, vals)" />
              </el-collapse-item>
            </el-collapse>
          </el-scrollbar>
        </div>
        <template #footer>
          <span class="dialog-footer">
            <el-button @click="envDrawerVisible = false">关 闭</el-button>
          </span>
        </template>
      </el-dialog>

      <!-- 保存请求对话框 -->
      <el-dialog v-model="saveDialogVisible" title="保存请求" width="420px">
        <el-form label-width="80px">
          <el-form-item label="请求名称">
            <el-input v-model="saveForm.name" placeholder="例如：用户登录接口" />
          </el-form-item>
          <el-form-item label="保存到">
            <el-select 
              v-model="saveForm.collId" 
              placeholder="选择集合或输入新集合名称" 
              style="width: 100%"
              filterable
              allow-create
              default-first-option
              :reserve-keyword="false"
            >
              <el-option v-for="coll in collections" :key="coll.id" :label="coll.name" :value="coll.id" />
            </el-select>
          </el-form-item>
        </el-form>
        <template #footer>
          <span class="dialog-footer">
            <el-button @click="saveDialogVisible = false">取 消</el-button>
            <el-button type="primary" :loading="saving" @click="handleSaveRequest">
              保 存
            </el-button>
          </span>
        </template>
      </el-dialog>


      <!-- 数据包发包输入对话框 -->
      <el-dialog v-model="packetInputDialogVisible" title="粘贴数据包" width="640px">
        <el-form label-position="top">
          <el-form-item label="请求数据包（必填）">
            <el-input
              v-model="packetInputForm.requestPacket"
              type="textarea"
              :rows="10"
              placeholder="粘贴原始 HTTP 请求包"
            />
          </el-form-item>
          <el-form-item label="响应数据包（可选，便于预览）">
            <el-input
              v-model="packetInputForm.responsePacket"
              type="textarea"
              :rows="6"
              placeholder="可选：粘贴响应包"
            />
          </el-form-item>
        </el-form>
        <template #footer>
          <span class="dialog-footer">
            <el-button @click="packetInputDialogVisible = false">取 消</el-button>
            <el-button type="primary" @click="openPacketViewerFromInput">查看并发包</el-button>
          </span>
        </template>
      </el-dialog>

      <!-- 批量测试配置对话框 -->
      <el-dialog v-model="batchTestConfigDialogVisible" title="批量测试配置" width="480px">
        <el-form label-width="110px">
          <el-form-item label="发包速率（并发数）">
            <el-input-number 
              v-model="batchTestConfig.concurrency" 
              :min="1" 
              :max="1000" 
              :step="1"
              style="width: 100%"
            />
            <div class="form-tip">每秒发送的请求数（1-1000），建议值：10-50</div>
          </el-form-item>
          <el-form-item label="重试次数">
            <el-input-number 
              v-model="batchTestConfig.retryCount" 
              :min="0" 
              :max="10" 
              :step="1"
              style="width: 100%"
            />
            <div class="form-tip">请求失败时的重试次数（0-10），建议值：1-3</div>
          </el-form-item>
          <el-form-item label="过滤敏感接口">
            <el-switch v-model="batchTestConfig.filterSensitive" />
            <div class="form-tip">自动过滤DELETE协议及包含delete、insert等敏感关键词的接口</div>
          </el-form-item>
        </el-form>
        <template #footer>
          <span class="dialog-footer">
            <el-button @click="batchTestConfigDialogVisible = false">取 消</el-button>
            <el-button type="primary" @click="confirmBatchTestConfig">
              开始测试
            </el-button>
          </span>
        </template>
      </el-dialog>

      <!-- Bypass 测试配置对话框 -->
      <el-dialog v-model="bypassTestConfigDialogVisible" title="Bypass 测试配置" width="480px">
        <el-form label-width="110px">
          <el-form-item label="发包速率（并发数）">
            <el-input-number 
              v-model="bypassTestConfig.concurrency" 
              :min="1" 
              :max="1000" 
              :step="1"
              style="width: 100%"
            />
            <div class="form-tip">每秒发送的请求数（1-1000），建议值：10-50</div>
          </el-form-item>
          <el-form-item label="重试次数">
            <el-input-number 
              v-model="bypassTestConfig.retryCount" 
              :min="0" 
              :max="10" 
              :step="1"
              style="width: 100%"
            />
            <div class="form-tip">请求失败时的重试次数（0-10），建议值：0-3</div>
          </el-form-item>
        </el-form>
        <template #footer>
          <span class="dialog-footer">
            <el-button @click="bypassTestConfigDialogVisible = false">取 消</el-button>
            <el-button type="primary" @click="confirmBypassTestConfig">
              开始测试
            </el-button>
          </span>
        </template>
      </el-dialog>

      <!-- 数据包查看对话框 -->
      <PacketViewer
        v-model="packetViewerVisible"
        :request-packet="selectedPacket?.requestPacket || ''"
        :response-packet="selectedPacket?.responsePacket || ''"
        :title="`数据包查看 - ${selectedPacket ? selectedPacket.requestName : ''}`"
        :editable="packetViewerEditable"
        :loading="packetViewerLoading"
        :canceling="packetCancelingRequest"
        :enable-send="true"
        :enable-save="true"
        @send="handlePacketSend"
        @cancel="handlePacketCancel"
        @save="handlePacketSave"
        @upload-bypass="handleUploadBypass"
      />

  </div>
</template>

<script>
import { ElMessage, ElMessageBox } from 'element-plus';
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';
import { nanoid } from 'nanoid';
import { VueMonacoEditor, useMonaco } from '@guolao/vue-monaco-editor';
import 'monaco-editor/esm/vs/basic-languages/monaco.contribution';
import 'monaco-editor/esm/vs/language/json/monaco.contribution';
import 'monaco-editor/esm/vs/language/html/monaco.contribution';
import { registerHTTPLanguage, detectPacketLanguage } from '../utils/http-language';
import { EventsOn, EventsOff, EventsEmit } from '../../wailsjs/runtime';
import KvTable from './restmate/KvTable.vue';
import FormdataTable from './restmate/FormdataTable.vue';
import EnvVarInput from './restmate/EnvVarInput.vue';
import PacketViewer from '../components/PacketViewer.vue';
import {
  GetCollections,
  AddCollection,
  RenameCollection,
  DeleteCollection,
  GetRequest,
  DuplicateRequest,
  RenameRequest,
  DeleteRequest,
  ExportCollection,
  ImportCollection,
  GetEnvs,
  SelectEnv,
  InvokeRequest,
  ChoseFile,
  AddEnv,
  RenameEnv,
  DeleteEnv,
  AddVar,
  DeleteVar,
  DuplicateEnv,
  UpsertRequest,
  BatchInvokeRequests,
  ImportSwaggerFromUrl,
  UploadBypassTest,
} from '../../wailsjs/go/restmate/RestMate';
import {DArrowLeft, DArrowRight, MoreFilled, Plus, Setting, Upload, Loading, Link, Refresh, Promotion} from "@element-plus/icons-vue";

export default {
  name: 'RestMateView',
  components: {
    Setting,
    MoreFilled,
    Upload,
    Plus,
    DArrowLeft,
    DArrowRight,
    Loading,
    Link,
    Refresh,
    Promotion,
    KvTable,
    FormdataTable,
    EnvVarInput,
    PacketViewer,
    VueMonacoEditor,
  },
  data() {
    const WS_KEYS = ['composer', 'rawPacket'];
    let ws =
      typeof localStorage !== 'undefined'
        ? localStorage.getItem('easytools:restmate:workspace')
        : null;
    if (!WS_KEYS.includes(ws)) ws = 'composer';
    return {
      restmateWorkspace: ws,
      fontColor: '#ffffff',
      fontWeightRatio: 100,
      shadowRatio: 100,
      STORAGE_KEY: 'easytools:theme:background',
      httpMethods: ['GET', 'POST', 'PUT', 'DELETE'],
      collections: [],
      collectionsLoading: true, // 集合加载状态
      activeRequestKey: '',
      currentRequest: this.createEmptyRequest(),
      tabs: [],
      activeTabId: '',
      sending: false,
      cancelingRequest: false,
      cancelledByUser: false,
      saving: false,
      response: null,
      responseError: '',
      activeReqTab: 'params',
      activeRespTab: 'body',
      envs: [],
      selectedEnvId: '',
      envDrawerVisible: false,
      editingEnvId: '',
      activeRespPreviewType: 'auto',
      saveDialogVisible: false,
      saveForm: {
        name: '',
        collId: '',
      },
      packetInputDialogVisible: false,
      packetInputForm: {
        requestPacket: '',
        responsePacket: '',
      },
      batchTestConfigDialogVisible: false,
      batchTestConfig: {
        concurrency: 5,
        retryCount: 0,
        filterSensitive: true,
      },
      pendingBatchTest: null, // 待执行的批量测试
      bypassTestConfigDialogVisible: false,
      bypassTestConfig: {
        concurrency: 5,
        retryCount: 0,
      },
      pendingBypassPayload: null, // 待执行的 bypass 测试 payload
      packetViewerEditable: false,
      packetViewerLoading: false,
      packetCancelingRequest: false,
      packetCancelledByUser: false,
      packetCheckInterval: null,
      respViewMode: 'pretty',
      monacoEditOptions: {
        minimap: { enabled: false },
        wordWrap: 'on',
        fontSize: 13,
        scrollbar: { verticalScrollbarSize: 8 },
        automaticLayout: true,
        scrollBeyondLastLine: false,
        lineNumbers: 'on',
        formatOnPaste: true,
        formatOnType: false,
      },
      monacoReadOnlyOptions: {
        minimap: { enabled: false },
        wordWrap: 'on',
        fontSize: 13,
        scrollbar: { verticalScrollbarSize: 8 },
        automaticLayout: true,
        scrollBeyondLastLine: false,
        readOnly: true,
        lineNumbers: 'on',
      },
      responseEditorHeight: 'calc(90vh - 260px)',
      tabsNavScrollEl: null,
      // 左侧请求集合面板展开/收起
      asideCollapsed: false,
      // 展开的集合ID列表
      activeCollectionNames: [],
      // 上一次展开的集合ID列表（用于比较变化）
      previousActiveCollectionNames: [],
      // 环境变量临时数据（用于存储新添加但未保存的变量）
      envTempVars: {},
      // 环境变量保存定时器
      envVarSaveTimer: null,
      // 数据包查看
      packetViewerVisible: false,
      selectedPacket: null,
      // 批量测试事件处理器
      batchTestEventHandlers: null,
      // 标签页右键菜单
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      // 文件上传 bypass
      bypassConfig: {
        language: 'all',
        forceProtocol: '自动检测',
        proxyHost: '',
        proxyPort: '',
      },
      tabsContextMenuTabId: '',
      currentBatchTestTabId: null,
      lastRequestPacket: '',
      lastResponsePacket: '',
      lastPacketTitle: '',
      // 集合请求缓存：存储已加载的集合请求数据
      collectionRequestsCache: {},
      // 集合加载状态：记录哪些集合正在加载请求
      collectionLoadingRequests: {},
      // 集合显示状态：记录每个集合已显示的请求数量（用于分页加载）
      collectionDisplayCount: {},
      // 每次加载的请求数量
      requestsPerPage: 50,
    };
  },
  computed: {
    currentTab() {
      return this.tabs.find(t => t.id === this.activeTabId);
    },
    hasPacketForCurrentTab() {
      return !!(this.currentRequestPacket || this.currentResponsePacket);
    },
    currentRequestPacket() {
      const tab = this.currentTab;
      if (tab && tab.requestPacket) return tab.requestPacket;
      return this.lastRequestPacket || '';
    },
    currentResponsePacket() {
      const tab = this.currentTab;
      if (tab && tab.responsePacket) return tab.responsePacket;
      return this.lastResponsePacket || '';
    },
    currentPacketTitle() {
      const tab = this.currentTab;
      if (tab && tab.packetTitle) return tab.packetTitle;
      return this.lastPacketTitle || '发送请求';
    },
    responseEditorValue() {
      if (this.responseError) return this.responseError;
      if (!this.response) return '';
      if (this.response.isBinary && !this.response.bodyContent) {
        return '当前响应为二进制数据，请使用“保存文件”下载';
      }
      if (!this.response.bodyContent) return '';
      const raw = this.response.bodyContent;
      if (this.respViewMode === 'raw') return raw;
      return this.formatResponseBody(raw, this.response.contentType);
    },
    responseBodyLanguage() {
      if (this.responseError) return 'plaintext';
      const raw = this.response?.bodyContent || '';
      const ct = (this.response?.contentType || '').toLowerCase();
      if (ct.includes('json')) return 'json';
      if (ct.includes('html')) return 'html';
      if (ct.includes('xml')) return 'xml';
      if (ct.includes('javascript')) return 'javascript';
      return this.detectTextLanguage(raw);
    },
    requestRawLanguage() {
      return this.detectTextLanguage(this.currentRequest?.body?.bodyRaw || '');
    },
    canPreviewHtml() {
      if (!this.response || !this.response.bodyContent) return false;
      
      // 排除重定向状态码（3xx），避免在iframe中加载导致about:srcdoc错误
      const statusCode = this.response.statusCode || this.response.StatusCode;
      if (statusCode && statusCode >= 300 && statusCode < 400) {
        return false;
      }
      
      const ct = (this.response.contentType || '').toUpperCase();
      return ct === 'HTML' || /<!doctype html>|<html[\s>]/i.test(this.response.bodyContent);
    },
    currentEnvVars() {
      if (!this.selectedEnvId) return {};
      const env = this.envs.find(e => e.id === this.selectedEnvId);
      return env?.variable || {};
    },
    tabsContextMenuStyle() {
      // 直接从事件获取坐标，优先使用 clientX/clientY（相对于视口，适用于 position: fixed）
      const { x, y } = this.tabsContextMenuPosition;
      
      // 直接设置坐标，确保菜单显示在鼠标点击位置
      return {
        top: `${y}px`,
        left: `${x}px`,
        position: 'fixed'
      };
    },
    tabsStyle() {
      const fontWeight = 400 + (this.fontWeightRatio / 100) * 500;
      const shadowOpacity = (this.shadowRatio / 100) * 0.06;
      return {
        '--tabs-font-color': '#303133',
        '--tabs-font-weight': Math.min(600, fontWeight),
        '--tabs-shadow': `0px 1px 3px rgba(0, 0, 0, ${shadowOpacity})`,
      };
    },
    restmateWorkspaceHint() {
      const hints = {
        composer: '集合与环境变量；左侧选请求，右侧编辑与响应同屏。',
        rawPacket:
          '并排编辑 Raw；工具栏可代入 HTTP 工作台请求；发送 / 格式化 / 复制；响应支持网页预览；工具栏「文件上传 Bypass」发起变种测试。',
      };
      return hints[this.restmateWorkspace] || '';
    },
  },
  watch: {
    currentRequest: {
      deep: true,
      handler() {
        const id = this.activeTabId;
        if (!id) return;
        const tab = this.tabs.find(t => t.id === id);
        if (!tab) return;
        tab.request = JSON.parse(JSON.stringify(this.currentRequest));
      },
    },
    activeRespTab(val) {
      const id = this.activeTabId;
      if (!id) return;
      const tab = this.tabs.find(t => t.id === id);
      if (!tab) return;
      tab.activeRespTab = val;
    },
    respViewMode(val) {
      const id = this.activeTabId;
      if (!id) return;
      const tab = this.tabs.find(t => t.id === id);
      if (!tab) return;
      tab.respViewMode = val;
    },
    activeTabId() {
      this.$nextTick(() => {
        this.scrollActiveTabIntoView();
      });
    },
    tabs: {
      handler() {
        this.$nextTick(() => {
          this.initTabsScroll();
          this.scrollActiveTabIntoView();
        });
      },
      deep: true,
    },
    activeCollectionNames: {
      handler(newVal, oldVal) {
        // 使用 watch 监听变化，避免干扰 v-model
        this.handleCollectionChange(newVal, oldVal);
      },
      immediate: false,
    },
    restmateWorkspace(val) {
      if (typeof localStorage === 'undefined') return;
      try {
        localStorage.setItem('easytools:restmate:workspace', val || 'composer');
      } catch (e) {
        /* ignore */
      }
    },
  },
  created() {
    this.loadCollections();
    this.loadEnvs();
  },
  mounted() {
    // 配置 Monaco Editor Worker（修复 Worker 错误）
    this.configureMonacoWorker();
    this.loadFontColor();
    window.addEventListener('theme-background-changed', this.handleThemeChange);

    this.registerHttpLanguage();
    this.$nextTick(() => {
      this.initTabsScroll();
      this.scrollActiveTabIntoView();
    });
    
    // 检查是否从漏洞扫描页面传递了数据包
    this.checkAndLoadPacketFromStorage();

    
    // 监听路由变化，确保每次跳转都能加载数据包
    if (this.$route) {
      this.$watch('$route.query', () => {
        this.checkAndLoadPacketFromStorage();
      }, { immediate: false });
    }
    
    // 添加定时检查机制，防止路由监听失效
    this.packetCheckInterval = setInterval(() => {
      const urlParams = new URLSearchParams(window.location.search);
      if (urlParams.get('openPacket') === 'true') {
        this.checkAndLoadPacketFromStorage();
      }
    }, 500);
  },
  // 使用 keep-alive 时，组件会被缓存，使用 activated/deactivated 控制全局快捷键
  activated() {
    // 仅在 Restmate 页面激活时监听全局 Ctrl+S
    window.addEventListener('keydown', this.handleKeydown);
  },
  deactivated() {
    // 切换到其他模块（如备忘笔记）时，取消监听，避免误触发“请先输入 URL”
    window.removeEventListener('keydown', this.handleKeydown);
  },
  beforeDestroy() {
    window.removeEventListener('theme-background-changed', this.handleThemeChange);
    window.removeEventListener('keydown', this.handleKeydown);
    if (this.envVarSaveTimer) {
      clearTimeout(this.envVarSaveTimer);
    }
    if (this.packetCheckInterval) {
      clearInterval(this.packetCheckInterval);
    }
    this.cleanupBatchTestListeners();
    if (this.tabsNavScrollEl) {
      this.tabsNavScrollEl.removeEventListener('wheel', this.handleTabsWheel);
    }
  },
  methods: {
    loadFontColor() {
      try {
        const saved = localStorage.getItem(this.STORAGE_KEY);
        if (saved) {
          const config = JSON.parse(saved);
          if (config.fontColor) this.fontColor = config.fontColor;
          if (config.fontWeightRatio !== undefined) this.fontWeightRatio = config.fontWeightRatio;
          if (config.shadowRatio !== undefined) this.shadowRatio = config.shadowRatio;
        }
      } catch (e) {
        console.error('加载字体颜色配置失败:', e);
      }
    },
    handleThemeChange(event) {
      if (event.detail && event.detail !== null) {
        if (event.detail.fontColor) this.fontColor = event.detail.fontColor;
        if (event.detail.fontWeightRatio !== undefined) this.fontWeightRatio = event.detail.fontWeightRatio;
        if (event.detail.shadowRatio !== undefined) this.shadowRatio = event.detail.shadowRatio;
      } else if (event.detail === null) {
        this.fontColor = '#ffffff';
        this.fontWeightRatio = 100;
        this.shadowRatio = 100;
      } else {
        this.loadFontColor();
      }
    },
    isCollectionLoading(collId) {
      return !!(this.collectionLoadingRequests && this.collectionLoadingRequests[collId]);
    },
    // 获取当前集合已显示的请求列表（分页）
    getDisplayedRequests(coll) {
      if (!coll || !coll.requests || coll.requests.length === 0) {
        return [];
      }
      const collId = coll.id;
      const displayCount = this.collectionDisplayCount[collId] || this.requestsPerPage;
      return coll.requests.slice(0, displayCount);
    },
    // 检查是否还有更多请求未显示
    hasMoreRequests(coll) {
      if (!coll || !coll.requests || coll.requests.length === 0) {
        return false;
      }
      const collId = coll.id;
      const displayCount = this.collectionDisplayCount[collId] || this.requestsPerPage;
      return coll.requests.length > displayCount;
    },
    // 获取剩余未显示的请求数量
    getRemainingCount(coll) {
      if (!coll || !coll.requests || coll.requests.length === 0) {
        return 0;
      }
      const collId = coll.id;
      const displayCount = this.collectionDisplayCount[collId] || this.requestsPerPage;
      return coll.requests.length - displayCount;
    },
    // 加载更多请求
    loadMoreRequests(coll) {
      const collId = coll.id;
      const currentCount = this.collectionDisplayCount[collId] || this.requestsPerPage;
      // 每次增加 requestsPerPage 个（Vue 3 直接赋值即可触发响应式）
      this.collectionDisplayCount[collId] = currentCount + this.requestsPerPage;
    },
    configureMonacoWorker() {
      // 配置 Monaco Editor Worker，修复 Worker 错误
      if (typeof window !== 'undefined' && window.MonacoEnvironment) {
        // 如果已经配置过，跳过
        return;
      }
      
      // 在 Wails 环境中，Monaco Editor 需要配置 Worker
      // 使用内联 Worker 或禁用 Worker 功能
      if (typeof window !== 'undefined') {
        window.MonacoEnvironment = {
          getWorkerUrl: function (moduleId, label) {
            // 返回空字符串，使用内联 Worker
            // 这样可以避免文件系统访问错误
            return '';
          },
          getWorker: function (moduleId, label) {
            // 返回 null，禁用 Worker，使用主线程
            // 对于简单的编辑场景，这样可以避免 Worker 相关的错误
            return null;
          }
        };
      }
    },
    createEmptyRequest() {
      return {
        id: nanoid(),
        name: '',
        url: '',
        method: 'get',
        headers: [],
        params: [],
        body: {
          bodyType: 'json',
          bodyRaw: '',
          formData: [],
        },
        coll_id: '',
      };
    },
    buildUrlWithParams(url, params = [], envVars = this.currentEnvVars || {}) {
      const active = (params || []).filter(p => p && p.key && p.active !== false);
      if (!active.length) return url || '';

      const interpolate = (val) => {
        const text = String(val ?? '');
        if (!text.includes('{{')) return text;
        return text.replace(/\{\{([^}]+)\}\}/g, (match, varName) => {
          const key = (varName || '').trim();
          if (Object.prototype.hasOwnProperty.call(envVars, key)) {
            return envVars[key];
          }
          return match;
        });
      };

      try {
        const rawUrl = url || '';
        const hasProtocol = /^https?:\/\//i.test(rawUrl);
        const helperUrl = hasProtocol
          ? rawUrl
          : `http://placeholder${rawUrl.startsWith('/') ? '' : '/'}${rawUrl}`;
        const u = new URL(helperUrl);
        active.forEach(p => u.searchParams.append(p.key, interpolate(p.value)));
        if (hasProtocol) {
          return u.toString();
        }
        return `${u.pathname}${u.search}`;
      } catch {
        const qs = active
          .map(p => `${encodeURIComponent(p.key)}=${encodeURIComponent(interpolate(p.value))}`)
          .join('&');
        if (!qs) return url || '';
        const joiner = (url || '').includes('?') ? '&' : '?';
        return `${url || ''}${joiner}${qs}`;
      }
    },
    initTabsScroll() {
      const navScroll = this.$refs.requestTabs?.$el?.querySelector('.el-tabs__nav-scroll');
      if (!navScroll) return;
      if (this.tabsNavScrollEl !== navScroll) {
        if (this.tabsNavScrollEl) {
          this.tabsNavScrollEl.removeEventListener('wheel', this.handleTabsWheel);
        }
        this.tabsNavScrollEl = navScroll;
        this.tabsNavScrollEl.addEventListener('wheel', this.handleTabsWheel, { passive: false });
      }
      this.scrollActiveTabIntoView();
    },
    handleTabsWheel(e) {
      if (!this.tabsNavScrollEl) return;
      const delta = e.deltaY || e.deltaX || 0;
      if (delta !== 0) {
        e.preventDefault();
        this.tabsNavScrollEl.scrollLeft += delta;
      }
    },
    scrollActiveTabIntoView() {
      const navScroll = this.tabsNavScrollEl || this.$refs.requestTabs?.$el?.querySelector('.el-tabs__nav-scroll');
      if (!navScroll) return;
      const activeEl = navScroll.querySelector('.el-tabs__item.is-active');
      if (activeEl && activeEl.scrollIntoView) {
        activeEl.scrollIntoView({
          behavior: 'smooth',
          block: 'nearest',
          inline: 'center',
        });
      }
    },
    registerHttpLanguage() {
      const { monacoRef } = useMonaco();
      if (!monacoRef?.value) {
        setTimeout(() => this.registerHttpLanguage(), 120);
        return;
      }
      const monaco = monacoRef.value;
      if (monaco.languages.getLanguages().some(lang => lang.id === 'http')) {
        return;
      }
      registerHTTPLanguage(monaco);
    },
    detectTextLanguage(text) {
      const content = (text || '').trim();
      if (!content) return 'plaintext';
      // HTTP/JSON/XML 判定复用已有工具
      const detected = detectPacketLanguage(content);
      if (detected && detected !== 'plaintext') return detected;
      if (/^<(!doctype html|html[\s>])/i.test(content)) return 'html';
      if (/^<\?xml/i.test(content)) return 'xml';
      if (/^\s*function\s|\s*=>/.test(content)) return 'javascript';
      return 'plaintext';
    },
    formatResponseBody(text, contentType = '') {
      if (!text) return '';
      const ct = (contentType || '').toLowerCase();
      const looksJson = ct.includes('json') || (/^[\[{]/.test(text.trim()) && text.trim().endsWith('}') || text.trim().endsWith(']'));
      if (looksJson) {
        try {
          return JSON.stringify(JSON.parse(text), null, 2);
        } catch {
          // fall through
        }
      }
      const looksXml = ct.includes('xml');
      const looksHtml = ct.includes('html') || /^<(!doctype html|html[\s>])/i.test(text);
      if (looksHtml || looksXml) {
        return this.prettifyMarkup(text);
      }
      return text;
    },
    prettifyMarkup(source) {
      const cleaned = String(source || '')
        .replace(/\r\n/g, '\n')
        .replace(/\r/g, '\n')
        .replace(/>\s+</g, '><');
      const tokens = cleaned.split(/(?=<)/g);
      let indent = 0;
      const lines = [];
      tokens.forEach(token => {
        const t = token.trim();
        if (!t) return;
        const isClosing = /^<\/\w/i.test(t);
        const isComment = /^<!--/.test(t);
        const isSelfClosing = /\/>$/.test(t) || /^<!/.test(t);
        if (isClosing && !isComment) {
          indent = Math.max(indent - 1, 0);
        }
        const pad = '  '.repeat(indent);
        lines.push(pad + t);
        if (!isClosing && !isSelfClosing && !isComment && /^<\w/i.test(t)) {
          indent += 1;
        }
      });
      return lines.join('\n');
    },
    async runMonacoFormat(editorRef) {
      if (!editorRef?.getEditor) return;
      await this.$nextTick();
      const editor = editorRef.getEditor();
      if (editor) {
        await editor.getAction('editor.action.formatDocument').run().catch(() => {});
      }
    },
    async ensureTabForRequest({ collId, reqId, name, url, method }, requestData = null) {
      if (!collId || !reqId) return;

      // 检查是否已存在相同请求的标签页
      const existingTab = this.tabs.find(t =>
        t.collId === collId && t.reqId === reqId
      );

      if (existingTab) {
        // 如果标签页已存在，激活它
        this.activeTabId = existingTab.id;
        this.loadRequestFromTab(existingTab);
        this.loadResponseFromTab(existingTab);
        return;
      }

      const id = nanoid();
      const methodText = (method || 'GET').toUpperCase();
      const base = name || url || '未命名请求';
      const title = `${methodText} ${base}`;

      // 深拷贝请求数据并确保格式正确
      let requestToSave = requestData
        ? JSON.parse(JSON.stringify(requestData))
        : JSON.parse(JSON.stringify(this.currentRequest));

      // 确保数据格式正确（同时保留 form-data 的 type / files 等扩展字段）
      const ensureArrayWithIds = (arr) => {
        if (!Array.isArray(arr)) return [];
        return arr.map(item => ({
          ...(item || {}),
          id: (item && item.id) || nanoid(),
          key: (item && item.key) || '',
          value: (item && item.value) || '',
          active: item && item.active !== undefined ? item.active : true,
          // form-data 相关字段，给默认值，避免丢失
          type: item && item.type ? item.type : 'text',
          files: Array.isArray(item && item.files) ? item.files : [],
        }));
      };

      // 规范化请求数据，处理可能的字段名差异（Url vs url）
      requestToSave = {
        id: requestToSave.id || requestToSave.ID || nanoid(),
        name: requestToSave.name || requestToSave.Name || '',
        url: requestToSave.url || requestToSave.Url || url || '',
        method: (requestToSave.method || requestToSave.Method || 'get').toLowerCase(),
        headers: ensureArrayWithIds(requestToSave.headers || requestToSave.Headers || []),
        params: ensureArrayWithIds(requestToSave.params || requestToSave.Params || []),
        body: {
          bodyType: requestToSave.body?.bodyType || requestToSave.Body?.bodyType || requestToSave.Body?.BodyType || 'json',
          bodyRaw: requestToSave.body?.bodyRaw || requestToSave.body?.bodyRaw || requestToSave.Body?.bodyRaw || requestToSave.Body?.BodyRaw || '',
          formData: ensureArrayWithIds(requestToSave.body?.formData || requestToSave.body?.FormData || requestToSave.Body?.formData || requestToSave.Body?.FormData || []),
        },
        coll_id: requestToSave.coll_id || requestToSave.collId || requestToSave.CollId || collId,
      };

      console.log('规范化后的请求数据:', {
        url: requestToSave.url,
        method: requestToSave.method,
        paramsCount: requestToSave.params.length,
        headersCount: requestToSave.headers.length,
        originalData: requestData
      });

      const tab = {
        id,
        title,
        collId,
        reqId,
        method: methodText,
        isNew: false,
        request: requestToSave,
        response: null,
        responseError: '',
        activeRespTab: 'body',
        respViewMode: 'pretty',
      };

      this.tabs.push(tab);
      this.activeTabId = id;

      // 立即加载数据到当前视图，Vue 3 直接赋值即可触发响应式
      const newRequest = JSON.parse(JSON.stringify(requestToSave));
      console.log('准备设置 currentRequest:', newRequest);
      console.log('newRequest.url:', newRequest.url);
      
      // Vue 3 中直接替换整个对象即可触发响应式更新
      // 确保所有字段都有值，避免 undefined
      this.currentRequest = {
        id: newRequest.id || nanoid(),
        name: newRequest.name || '',
        url: String(newRequest.url || ''),
        method: newRequest.method || 'get',
        headers: Array.isArray(newRequest.headers) ? newRequest.headers : [],
        params: Array.isArray(newRequest.params) ? newRequest.params : [],
        body: {
          bodyType: newRequest.body?.bodyType || 'json',
          bodyRaw: String(newRequest.body?.bodyRaw || ''),
          formData: Array.isArray(newRequest.body?.formData) ? newRequest.body.formData : [],
        },
        coll_id: newRequest.coll_id || collId,
      };
      
      this.response = null;
      this.responseError = '';

      console.log('设置后的 currentRequest:', this.currentRequest);
      console.log('currentRequest.url:', this.currentRequest.url);
      console.log('currentRequest.params:', this.currentRequest.params);
      console.log('currentRequest.headers:', this.currentRequest.headers);

      // 确保DOM更新，让视图正确渲染
      await this.$nextTick();
      console.log('nextTick 后的 currentRequest:', this.currentRequest);
      console.log('nextTick 后的 currentRequest.url:', this.currentRequest.url);
      
      // 再次等待一个 tick，确保所有子组件都已更新
      await this.$nextTick();
      
      console.log('标签页创建完成:', {
        tabId: id,
        requestId: reqId,
        paramsCount: requestToSave.params?.length,
        headersCount: requestToSave.headers?.length,
        url: requestToSave.url,
        currentRequestUrl: this.currentRequest.url
      });
    },
    persistCurrentToActiveTab() {
      const id = this.activeTabId;
      if (!id) return;
      const tab = this.tabs.find(t => t.id === id);
      if (!tab) return;
      tab.request = JSON.parse(JSON.stringify(this.currentRequest));
    },
    loadRequestFromTab(tab) {
      if (!tab) return;

      if (tab.request) {
        const request = JSON.parse(JSON.stringify(tab.request));

        // 确保数据格式正确（同时保留 form-data 的 type / files 等扩展字段）
        const ensureArrayWithIds = (arr) => {
          if (!Array.isArray(arr)) return [];
          return arr.map(item => ({
            ...(item || {}),
            id: (item && item.id) || nanoid(),
            key: (item && item.key) || '',
            value: (item && item.value) || '',
            active: item && item.active !== undefined ? item.active : true,
            // form-data 相关字段，给默认值，避免丢失
            type: item && item.type ? item.type : 'text',
            files: Array.isArray(item && item.files) ? item.files : [],
          }));
        };

        // 重新构建请求对象，确保触发响应式更新
        const newRequest = {
          id: request.id,
          name: request.name || '',
          url: request.url || '',
          method: (request.method || 'get').toLowerCase(),
          headers: ensureArrayWithIds(request.headers),
          params: ensureArrayWithIds(request.params),
          body: {
            bodyType: request.body?.bodyType || 'json',
            bodyRaw: request.body?.bodyRaw || '',
            formData: ensureArrayWithIds(request.body?.formData || []),
          },
          coll_id: request.coll_id || request.collId || tab.collId,
        };

        // Vue 3 中直接替换整个对象即可触发响应式更新
        // 确保所有字段都有值，避免 undefined
        this.currentRequest = {
          id: newRequest.id || nanoid(),
          name: newRequest.name || '',
          url: String(newRequest.url || ''),
          method: newRequest.method || 'get',
          headers: Array.isArray(newRequest.headers) ? newRequest.headers : [],
          params: Array.isArray(newRequest.params) ? newRequest.params : [],
          body: {
            bodyType: newRequest.body?.bodyType || 'json',
            bodyRaw: String(newRequest.body?.bodyRaw || ''),
            formData: Array.isArray(newRequest.body?.formData) ? newRequest.body.formData : [],
          },
          coll_id: newRequest.coll_id || tab.collId || '',
        };

        console.log('加载请求数据:', {
          url: newRequest.url,
          method: newRequest.method,
          params: newRequest.params.length,
          headers: newRequest.headers.length,
          bodyType: newRequest.body.bodyType
        });
      } else if (tab.collId && tab.reqId && !tab.isNew) {
        // 如果没有缓存数据，重新获取
        this.fetchRequest(tab.collId, tab.reqId);
      } else {
        // 创建空白请求
        const emptyRequest = this.createEmptyRequest();
        emptyRequest.coll_id = tab.collId || '';
        this.currentRequest = JSON.parse(JSON.stringify(emptyRequest));
      }
    },
    loadResponseFromTab(tab) {
      this.response = tab.response || null;
      this.responseError = tab.responseError || '';
      this.activeRespTab = tab.activeRespTab || 'body';
      this.respViewMode = tab.respViewMode || 'pretty';
    },
    async copyResponseBody() {
      if (!this.response || !this.response.bodyContent) {
        ElMessage.warning('暂无可复制的响应内容');
        return;
      }
      try {
        await navigator.clipboard.writeText(this.response.bodyContent);
        ElMessage.success('响应内容已复制到剪贴板');
      } catch {
        ElMessage.error('复制失败，请手动选择复制');
      }
    },
    async copyPacket(type) {
      const text = type === 'request' ? this.currentRequestPacket : this.currentResponsePacket;
      if (!text) {
        ElMessage.warning(`暂无可复制的${type === 'request' ? '请求' : '响应'}数据包`);
        return;
      }
      try {
        await navigator.clipboard.writeText(text);
        ElMessage.success(`${type === 'request' ? '请求' : '响应'}数据包已复制`);
      } catch {
        ElMessage.error('复制失败，请手动选择复制');
      }
    },
    async saveBinaryResponse() {
      if (!this.response || !this.response.binaryData) {
        ElMessage.warning('暂无可保存的二进制数据');
        return;
      }
      try {
        // 解码 Base64 数据
        const binaryString = atob(this.response.binaryData);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
          bytes[i] = binaryString.charCodeAt(i);
        }
        const blob = new Blob([bytes], { type: 'application/octet-stream' });
        
        // 生成文件名
        const timestamp = new Date().getTime();
        const ext = this.response.suggestedExt || 'bin';
        const filename = `response_${timestamp}.${ext}`;
        
        // 创建下载链接
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
        
        ElMessage.success('文件已保存');
      } catch (error) {
        console.error('保存文件失败:', error);
        ElMessage.error('保存文件失败：' + (error?.message || '未知错误'));
      }
    },
    async loadCollections() {
      this.collectionsLoading = true;
      // 确保 activeCollectionNames 始终是数组
      if (!Array.isArray(this.activeCollectionNames)) {
        this.activeCollectionNames = [];
      }
      try {
        const res = await GetCollections();
        if (res.success) {
          // 性能优化：初始只加载集合基本信息，不加载请求列表
          // 请求列表将在展开集合时按需加载
          this.collections = (res.data || []).map(c => ({
            id: c.id,
            name: c.name,
            requests: [], // 初始为空，展开时再加载
            requestsCount: (c.requests || []).length, // 保存请求数量用于显示
          }));

          // 将请求数据缓存起来，避免重复请求后端
          (res.data || []).forEach(c => {
            if (c.requests && c.requests.length > 0) {
              this.collectionRequestsCache[c.id] = c.requests || [];
            }
          });

          // 初始化展开状态记录，使用当前 activeCollectionNames 的值
          // 这样第一次点击时就不会被误判为"首次加载"
          this.previousActiveCollectionNames = Array.isArray(this.activeCollectionNames) 
            ? [...this.activeCollectionNames] 
            : [];

          // 自动选中第一个请求
          // 默认打开空白标签页
          if (!this.tabs.length) {
            this.handleCreateRequest();
          }
        } else {
          ElMessage.error(res.msg || '加载集合失败');
        }
      } catch (e) {
        ElMessage.error('加载集合失败');
      } finally {
        this.collectionsLoading = false;
      }
    },
    async handleRefreshCollections() {
      // 清空缓存，强制重新加载
      this.collectionRequestsCache = {};
      this.collectionLoadingRequests = {};
      await this.loadCollections();
      ElMessage.success('集合列表已刷新');
    },
    async loadEnvs() {
      try {
        const res = await GetEnvs();
        if (res.success) {
          this.envs = res.data || [];
          const current = this.envs.find(e => e.selected);
          this.selectedEnvId = current ? current.id : '';
        }
      } catch {
        // 忽略错误
      }
    },
    async handleEnvChange(id) {
      if (!id) {
        this.selectedEnvId = '';
        return;
      }
      try {
        const res = await SelectEnv(id);
        if (res.success) {
          ElMessage.success(res.msg || '环境切换成功');
          this.loadEnvs();
        } else {
          ElMessage.error(res.msg || '环境切换失败');
        }
      } catch {
        ElMessage.error('环境切换失败');
      }
    },
    async handleAddEnv() {
      const resPrompt = await ElMessageBox.prompt('请输入新环境名称', '新增环境', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputPlaceholder: '例如：开发环境',
      }).catch(() => null);
      if (!resPrompt || !resPrompt.value) return;
      try {
        const res = await AddEnv(resPrompt.value);
        if (res.success) {
          ElMessage.success(res.msg || '新增环境成功');
          await this.loadEnvs();
        } else {
          ElMessage.error(res.msg || '新增环境失败');
        }
      } catch {
        ElMessage.error('新增环境失败');
      }
    },
    async renameEnv(env) {
      const resPrompt = await ElMessageBox.prompt('请输入新的环境名称', '重命名环境', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputValue: env.name,
      }).catch(() => null);
      if (!resPrompt || !resPrompt.value) return;
      try {
        const res = await RenameEnv(env.id, resPrompt.value);
        if (res.success) {
          ElMessage.success(res.msg || '重命名成功');
          await this.loadEnvs();
        } else {
          ElMessage.error(res.msg || '重命名失败');
        }
      } catch {
        ElMessage.error('重命名失败');
      }
    },
    async deleteEnv(env) {
      try {
        await confirmDangerousAction({
          title: '删除环境',
          message: `确定删除环境「${env.name}」吗？`,
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      try {
        const res = await DeleteEnv(env.id);
        if (res.success) {
          ElMessage.success(res.msg || '删除成功');
          await this.loadEnvs();
        } else {
          ElMessage.error(res.msg || '删除失败');
        }
      } catch {
        ElMessage.error('删除失败');
      }
    },
    async duplicateEnv(env) {
      try {
        const res = await DuplicateEnv(env.id);
        if (res.success) {
          ElMessage.success(res.msg || '复制成功');
          await this.loadEnvs();
        } else {
          ElMessage.error(res.msg || '复制失败');
        }
      } catch {
        ElMessage.error('复制失败');
      }
    },
    async setCurrentEnv(env) {
      try {
        const res = await SelectEnv(env.id);
        if (res.success) {
          ElMessage.success(res.msg || '环境已切换');
          await this.loadEnvs();
          this.selectedEnvId = env.id;
        } else {
          ElMessage.error(res.msg || '切换失败');
        }
      } catch {
        ElMessage.error('切换失败');
      }
    },
    envVariablePairs(env) {
      const vars = env.variable || {};
      return Object.keys(vars).map(k => ({
        id: k,
        key: k,
        value: vars[k],
        active: true,
      }));
    },
    envVariablePairsWithTemp(env) {
      const vars = env.variable || {};
      const tempVars = this.envTempVars[env.id] || [];

      // 合并实际变量和临时变量
      const result = Object.keys(vars).map(k => ({
        id: k,
        key: k,
        value: vars[k],
        active: true,
      }));

      // 添加临时变量（新添加但未保存的）
      // 使用 Map 来去重，优先使用临时变量的值（因为可能正在编辑）
      const varMap = new Map();
      // 先添加实际变量
      result.forEach(item => {
        varMap.set(item.id, item);
      });
      // 再添加临时变量，覆盖已存在的
      tempVars.forEach(item => {
        varMap.set(item.id, item);
      });

      return Array.from(varMap.values());
    },
    handleEnvVarListChange(env, kvList) {
      // 保存临时变量（包括空行和正在编辑的行）
      this.envTempVars[env.id] = kvList;

      // 延迟保存到后端（防抖），只保存有 key 的变量
      if (this.envVarSaveTimer) {
        clearTimeout(this.envVarSaveTimer);
      }
      this.envVarSaveTimer = setTimeout(() => {
        // 只保存有 key 的变量，保留空行和正在编辑的行
        const varsToSave = kvList.filter(item => {
          const key = (item.key || '').trim();
          return key.length > 0;
        });

        // 如果有需要保存的变量，才调用保存
        if (varsToSave.length > 0) {
          this.saveEnvVariables(env, varsToSave, kvList);
        }
      }, 1500); // 增加防抖时间到 1.5 秒，给用户更多输入时间
    },
    async saveEnvVariables(env, kvList, allKvList) {
      // 在保存前，保存当前的完整临时变量列表
      const currentTempVars = this.envTempVars[env.id] || [];

      const oldVars = env.variable || {};
      const oldKeys = new Set(Object.keys(oldVars));
      const newKeys = new Set();

      // 添加或更新（只保存有 key 的）
      for (const item of kvList) {
        const key = (item.key || '').trim();
        if (!key) continue;
        newKeys.add(key);
        if (!oldKeys.has(key) || oldVars[key] !== item.value) {
          try {
            await AddVar(env.id, key, item.value || '');
          } catch {
            // 忽略单个错误，统一刷新
          }
        }
      }

      // 删除不存在的
      for (const k of oldKeys) {
        if (!newKeys.has(k)) {
          try {
            await DeleteVar(env.id, k);
          } catch {
            // 忽略单个错误
          }
        }
      }

      await this.loadEnvs();

      // 重新加载后，保留所有临时变量（包括空行和正在编辑的行）
      // 使用保存前的完整列表，但要去除已完全同步的项
      const envAfterLoad = this.envs.find(e => e.id === env.id);
      const savedVars = envAfterLoad?.variable || {};

      // 保留所有临时变量，但去除已完全同步的（key和value都匹配的）
      const remainingTempVars = (allKvList || currentTempVars).filter(item => {
        const key = (item.key || '').trim();
        // 空行保留
        if (!key) return true;
        // 如果已保存且value相同，说明已同步，不保留
        if (savedVars[key] === item.value) {
          return false;
        }
        // 其他情况保留（包括正在编辑的）
        return true;
      });

      this.envTempVars[env.id] = remainingTempVars;
    },
    async updateEnvVariables(env, kvList) {
      const oldVars = env.variable || {};
      const oldKeys = new Set(Object.keys(oldVars));
      const newKeys = new Set();

      // 添加或更新
      for (const item of kvList) {
        const key = (item.key || '').trim();
        if (!key) continue;
        newKeys.add(key);
        if (!oldKeys.has(key) || oldVars[key] !== item.value) {
          try {
            await AddVar(env.id, key, item.value || '');
          } catch {
            // 忽略单个错误，统一刷新
          }
        }
      }

      // 删除不存在的
      for (const k of oldKeys) {
        if (!newKeys.has(k)) {
          try {
            await DeleteVar(env.id, k);
          } catch {
            // 忽略单个错误
          }
        }
      }

      await this.loadEnvs();
      ElMessage.success('环境变量已更新');
    },
    async handleAddCollection() {
      const name = await ElMessageBox.prompt('请输入集合名称', '新增集合', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputPlaceholder: '例如：渗透测试接口',
      }).catch(() => null);
      if (!name || !name.value) return;
      const id = nanoid();
      try {
        const res = await AddCollection(id, name.value);
        if (res.success) {
          ElMessage.success(res.msg || '集合创建成功');
          // 更新集合列表，同时更新缓存
          this.collections = (res.data || []).map(c => ({
            id: c.id,
            name: c.name,
            requests: [],
            requestsCount: (c.requests || []).length,
          }));
          // 更新缓存
          (res.data || []).forEach(c => {
            if (c.requests && c.requests.length > 0) {
              this.collectionRequestsCache[c.id] = c.requests || [];
            }
          });
        } else {
          ElMessage.error(res.msg || '集合创建失败');
        }
      } catch {
        ElMessage.error('集合创建失败');
      }
    },
    handleCollectionChange(newVal, oldVal) {
      // 确保 newVal 始终是数组
      const nowActive = Array.isArray(newVal) ? newVal : [];
      const previousActive = Array.isArray(oldVal) ? oldVal : [];
      
      // 找出新展开的集合（需要加载请求列表）
      const newlyOpened = nowActive.filter(id => !previousActive.includes(id));
      
      // 找出新关闭的集合（重置显示数量）
      const newlyClosed = previousActive.filter(id => !nowActive.includes(id));
      newlyClosed.forEach(collId => {
        // 关闭集合时，可以选择保留显示数量或重置
        // 这里选择重置，下次打开时重新从第一页开始（Vue 3 使用 delete 操作符）
        delete this.collectionDisplayCount[collId];
      });
      
      // 为新展开的集合加载请求列表
      newlyOpened.forEach(collId => {
        this.loadCollectionRequests(collId);
      });
      
      // 如果是首次变化（oldVal 为 undefined 或 null），不显示提示，只保存状态
      if (!oldVal || !Array.isArray(oldVal) || oldVal.length === 0) {
        // 如果 previousActiveCollectionNames 也为空，说明是首次加载
        if (!this.previousActiveCollectionNames || this.previousActiveCollectionNames.length === 0) {
          this.previousActiveCollectionNames = [...nowActive];
          return;
        }
        // 否则使用 previousActiveCollectionNames 作为 oldVal
        const prevActive = Array.isArray(this.previousActiveCollectionNames) ? this.previousActiveCollectionNames : [];
        this.handleCollectionChangeWithPrevious(nowActive, prevActive);
        return;
      }
      
      // 正常处理变化
      this.handleCollectionChangeWithPrevious(nowActive, previousActive);
    },
    // 按需加载集合的请求列表
    async loadCollectionRequests(collId) {
      if (!collId) return;
      
      // 如果已经在加载中，跳过
      if (this.collectionLoadingRequests[collId]) {
        return;
      }
      
      // 如果已经加载过，直接从缓存中获取
      if (this.collectionRequestsCache[collId]) {
        const coll = this.collections.find(c => c.id === collId);
        if (coll && (!coll.requests || coll.requests.length === 0)) {
          coll.requests = this.collectionRequestsCache[collId];
          // 重置显示数量为初始值（Vue 3 直接赋值即可触发响应式）
          this.collectionDisplayCount[collId] = this.requestsPerPage;
        }
        return;
      }
      
      // 标记为加载中
      this.collectionLoadingRequests[collId] = true;
      
      try {
        // 重新获取所有集合（因为后端接口一次性返回所有数据）
        // 但我们可以只更新当前集合的请求列表
        const res = await GetCollections();
        if (res.success && res.data) {
          const targetColl = res.data.find(c => c.id === collId);
          if (targetColl) {
            // 更新缓存
            this.collectionRequestsCache[collId] = targetColl.requests || [];
            
            // 更新当前集合的请求列表
            const coll = this.collections.find(c => c.id === collId);
            if (coll) {
              coll.requests = targetColl.requests || [];
              // 重置显示数量为初始值（Vue 3 直接赋值即可触发响应式）
              this.collectionDisplayCount[collId] = this.requestsPerPage;
            }
          }
        }
      } catch (e) {
        console.error('加载集合请求失败:', e);
      } finally {
        // 清除加载状态
        delete this.collectionLoadingRequests[collId];
      }
    },
    handleCollectionChangeWithPrevious(nowActive, previousActive) {
      // 找出新展开的集合和关闭的集合
      const opened = nowActive.filter(id => !previousActive.includes(id));
      const closed = previousActive.filter(id => !nowActive.includes(id));
      
      // 显示提示
      if (opened.length > 0) {
        const coll = this.collections.find(c => c.id === opened[0]);
        if (coll) {
          // ElMessage.success(`已展开集合「${coll.name}」`);
        }
      }
      if (closed.length > 0) {
        const coll = this.collections.find(c => c.id === closed[0]);
        if (coll) {
          // ElMessage.info(`已收起集合「${coll.name}」`);
        }
      }
      
      // 保存当前状态，用于下次比较
      this.previousActiveCollectionNames = [...nowActive];
    },
    handleCollectionCommand(cmd, coll) {
      if (cmd === 'rename') {
        this.renameCollection(coll);
      } else if (cmd === 'delete') {
        this.deleteCollection(coll);
      } else if (cmd === 'export') {
        this.exportCollection(coll);
      } else if (cmd === 'batchTest') {
        this.handleBatchTest(coll);
      }
    },
    async handleImportCollection() {
      try {
        const res = await ImportCollection();
        if (res.success) {
          // 更新集合列表，同时更新缓存
          this.collections = (res.data || []).map(c => ({
            id: c.id,
            name: c.name,
            requests: [],
            requestsCount: (c.requests || []).length,
          }));
          // 更新缓存
          (res.data || []).forEach(c => {
            if (c.requests && c.requests.length > 0) {
              this.collectionRequestsCache[c.id] = c.requests || [];
            }
          });
          ElMessage.success(res.msg || '导入成功');
        } else {
          ElMessage.error(res.msg || '导入失败');
        }
      } catch {
        ElMessage.error('导入失败');
      }
    },
    async handleImportSwaggerFromUrl() {
      try {
        const resPrompt = await ElMessageBox.prompt('请输入 Swagger/OpenAPI 文档的 URL', '从URL导入Swagger', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          inputPlaceholder: 'https://example.com/swagger.json',
          inputValidator: (value) => {
            if (!value) {
              return '请输入URL';
            }
            if (!value.startsWith('http://') && !value.startsWith('https://')) {
              return '请输入有效的URL（以http://或https://开头）';
            }
            return true;
          },
        }).catch(() => null);
        if (!resPrompt || !resPrompt.value) return;
        
        ElMessage.info('正在从URL导入Swagger文档...');
        const res = await ImportSwaggerFromUrl(resPrompt.value);
        if (res.success) {
          // 找到新导入的集合（通常是最后一个）
          const newCollections = (res.data || []).map(c => ({
            id: c.id,
            name: c.name,
            requests: c.requests || [],
          }));
          const newCollection = newCollections[newCollections.length - 1];
          
          // 弹出重命名对话框
          const renamePrompt = await ElMessageBox.prompt('请输入集合名称', '重命名集合', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            inputValue: newCollection?.name || '',
            inputPlaceholder: '例如：API接口集合',
          }).catch(() => null);
          
          if (renamePrompt && renamePrompt.value) {
            // 调用重命名接口
            try {
              const renameRes = await RenameCollection(newCollection.id, renamePrompt.value);
              if (renameRes.success) {
                // 更新集合列表，同时更新缓存
                this.collections = (renameRes.data || []).map(c => ({
                  id: c.id,
                  name: c.name,
                  requests: [],
                  requestsCount: (c.requests || []).length,
                }));
                // 更新缓存
                (renameRes.data || []).forEach(c => {
                  if (c.requests && c.requests.length > 0) {
                    this.collectionRequestsCache[c.id] = c.requests || [];
                  }
                });
                ElMessage.success('导入并重命名成功');
              } else {
                this.collections = newCollections.map(c => ({
                  id: c.id,
                  name: c.name,
                  requests: [],
                  requestsCount: (c.requests || []).length,
                }));
                // 更新缓存
                newCollections.forEach(c => {
                  if (c.requests && c.requests.length > 0) {
                    this.collectionRequestsCache[c.id] = c.requests || [];
                  }
                });
                ElMessage.warning('导入成功，但重命名失败：' + (renameRes.msg || '未知错误'));
              }
            } catch (e) {
              this.collections = newCollections.map(c => ({
                id: c.id,
                name: c.name,
                requests: [],
                requestsCount: (c.requests || []).length,
              }));
              // 更新缓存
              newCollections.forEach(c => {
                if (c.requests && c.requests.length > 0) {
                  this.collectionRequestsCache[c.id] = c.requests || [];
                }
              });
              ElMessage.warning('导入成功，但重命名失败：' + (e?.message || '未知错误'));
            }
          } else {
            this.collections = newCollections.map(c => ({
              id: c.id,
              name: c.name,
              requests: [],
              requestsCount: (c.requests || []).length,
            }));
            // 更新缓存
            newCollections.forEach(c => {
              if (c.requests && c.requests.length > 0) {
                this.collectionRequestsCache[c.id] = c.requests || [];
              }
            });
            ElMessage.success(res.msg || '导入成功');
          }
        } else {
          ElMessage.error(res.msg || '导入失败');
        }
      } catch (e) {
        ElMessage.error('导入失败：' + (e?.message || '未知错误'));
      }
    },
    async renameCollection(coll) {
      const resPrompt = await ElMessageBox.prompt('请输入新的集合名称', '重命名集合', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputValue: coll.name,
      }).catch(() => null);
      if (!resPrompt || !resPrompt.value) return;
      try {
        const res = await RenameCollection(coll.id, resPrompt.value);
        if (res.success) {
          ElMessage.success(res.msg || '重命名成功');
          // 更新集合列表，保持已加载的请求列表
          this.collections = (res.data || []).map(c => {
            const existingColl = this.collections.find(ex => ex.id === c.id);
            return {
              id: c.id,
              name: c.name,
              requests: existingColl?.requests || this.collectionRequestsCache[c.id] || [],
              requestsCount: (c.requests || []).length,
            };
          });
          // 更新缓存
          (res.data || []).forEach(c => {
            if (c.requests && c.requests.length > 0) {
              this.collectionRequestsCache[c.id] = c.requests || [];
            }
          });
        } else {
          ElMessage.error(res.msg || '重命名失败');
        }
      } catch {
        ElMessage.error('重命名失败');
      }
    },
    async deleteCollection(coll) {
      try {
        await confirmDangerousAction({
          title: '删除集合',
          message: `确定删除集合「${coll.name}」及其所有请求吗？`,
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      try {
        const res = await DeleteCollection(coll.id);
        if (res.success) {
          ElMessage.success(res.msg || '删除成功');
          this.collections = (res.data || []).map(c => ({
            id: c.id,
            name: c.name,
            requests: c.requests || [],
          }));
        } else {
          ElMessage.error(res.msg || '删除失败');
        }
      } catch {
        ElMessage.error('删除失败');
      }
    },
    async exportCollection(coll) {
      try {
        const res = await ExportCollection(coll.id);
        if (res.success) {
          ElMessage.success(res.msg || '导出成功');
        } else {
          ElMessage.error(res.msg || '导出失败');
        }
      } catch {
        ElMessage.error('导出失败');
      }
    },
    async handleBatchTest(coll) {
      // 确保请求列表已加载
      if (!coll.requests || coll.requests.length === 0) {
        // 尝试从缓存加载
        if (this.collectionRequestsCache[coll.id]) {
          coll.requests = this.collectionRequestsCache[coll.id];
        } else {
          // 如果缓存也没有，先加载
          await this.loadCollectionRequests(coll.id);
        }
      }
      
      if (!coll.requests || coll.requests.length === 0) {
        ElMessage.warning('该集合中没有请求');
        return;
      }
      
      // 重置配置为默认值
      this.batchTestConfig = {
        concurrency: 10,
        retryCount: 0,
        filterSensitive: true,
      };
      
      // 保存待执行的批量测试
      this.pendingBatchTest = coll;
      
      // 显示配置对话框（使用 nextTick 确保 DOM 更新）
      this.$nextTick(() => {
        this.batchTestConfigDialogVisible = true;
      });
    },
    confirmBatchTestConfig() {
      // 验证配置
      if (this.batchTestConfig.concurrency < 1 || this.batchTestConfig.concurrency > 1000) {
        ElMessage.warning('并发数必须在1-1000之间');
        return;
      }
      if (this.batchTestConfig.retryCount < 0 || this.batchTestConfig.retryCount > 10) {
        ElMessage.warning('重试次数必须在0-10之间');
        return;
      }
      
      if (!this.pendingBatchTest) {
        ElMessage.error('批量测试数据丢失，请重新操作');
        this.batchTestConfigDialogVisible = false;
        return;
      }
      
      // 关闭对话框
      this.batchTestConfigDialogVisible = false;
      
      // 开始批量测试
      this.startBatchTest(this.pendingBatchTest, this.batchTestConfig);
      this.pendingBatchTest = null;
    },
    // 检查请求是否为敏感接口
    isSensitiveRequest(req) {
      if (!req) return false;
      
      // 检查是否为DELETE协议
      const method = (req.method || '').toLowerCase();
      if (method === 'delete') {
        return true;
      }
      
      // 检查URL中是否包含敏感关键词
      const url = (req.url || '').toLowerCase();
      const sensitiveKeywords = ['delete', 'insert', 'update', 'remove', 'drop', 'truncate', 'clear', 'destroy'];
      
      for (const keyword of sensitiveKeywords) {
        if (url.includes(keyword)) {
          return true;
        }
      }
      
      return false;
    },
    startBatchTest(coll, batchConfig) {
      // 根据过滤配置过滤请求
      let requestsToTest = coll.requests || [];
      
      if (batchConfig.filterSensitive) {
        const beforeFilter = requestsToTest.length;
        requestsToTest = requestsToTest.filter(req => !this.isSensitiveRequest(req));
        const afterFilter = requestsToTest.length;
        
        if (beforeFilter !== afterFilter) {
          ElMessage.info(`已过滤 ${beforeFilter - afterFilter} 个敏感接口，将测试 ${afterFilter} 个请求`);
        }
      }
      
      if (requestsToTest.length === 0) {
        ElMessage.warning('过滤后没有可测试的请求');
        return;
      }
      
      // 创建批量测试标签页
      const newTabId = nanoid();
      const title = `批量测试: ${coll.name}`;
      const batchTestTab = {
        id: newTabId,
        title,
        collId: coll.id,
        reqId: '',
        request: this.createEmptyRequest(),
        isNew: false,
        isBatchTest: true,
        batchTesting: true,
        batchTestResults: null,
        batchTestRequests: requestsToTest.map((req, index) => ({
          requestId: req.id,
          name: req.name || '未命名请求',
          method: req.method || 'get',
          url: req.url || '',
          status: 'pending',
          index: index,
          result: null,
          error: null,
          requestPacket: '',
          responsePacket: '',
        })),
        response: null,
        responseError: '',
        activeRespTab: 'body',
        respViewMode: 'pretty',
      };
      this.tabs.push(batchTestTab);
      this.activeTabId = newTabId;
      
      // 监听批量测试事件
      this.setupBatchTestListeners(newTabId);
      
      // 执行批量测试（异步执行，通过事件实时更新）
      BatchInvokeRequests(coll.id, batchConfig.concurrency, batchConfig.retryCount, batchConfig.filterSensitive || false).then((res) => {
        // 批量测试完成后，确保最终结果已更新
        const tabIndex = this.tabs.findIndex(t => t.id === newTabId);
        if (tabIndex !== -1) {
          if (res.success && res.data) {
            // 最终结果已经在事件中更新，这里只更新测试状态
            this.tabs[tabIndex].batchTesting = false;
            // 确保最终结果完整
            if (!this.tabs[tabIndex].batchTestResults) {
              this.tabs[tabIndex].batchTestResults = res.data;
            }
          } else {
            this.tabs[tabIndex].batchTesting = false;
            ElMessage.error(res.msg || '批量测试失败');
          }
        }
      }).catch((e) => {
        const tabIndex = this.tabs.findIndex(t => t.id === newTabId);
        if (tabIndex !== -1) {
          this.tabs[tabIndex].batchTesting = false;
        }
        ElMessage.error('批量测试异常：' + (e?.message || '未知错误'));
      });
    },
    setupBatchTestListeners(tabId) {
      this.currentBatchTestTabId = tabId;
      
      // 监听单个请求状态更新
      EventsOn('batchRequestResult', (data) => {
        const tab = this.tabs.find(t => t.id === tabId);
        if (!tab || !tab.isBatchTest) return;
        
        const requestIndex = data.index;
        if (requestIndex !== undefined && tab.batchTestRequests && tab.batchTestRequests[requestIndex]) {
          // 使用 Vue 3 的响应式更新方式：使用 splice 替换整个对象以确保触发响应式
          const currentRequest = tab.batchTestRequests[requestIndex];
          
          // 创建更新后的请求对象
          let updatedRequest = { ...currentRequest };
          
          // 如果有结果，更新为最终状态
          if (data.result) {
            updatedRequest.status = data.result.success ? 'success' : 'failed';
            updatedRequest.result = data.result.result || currentRequest.result;
            updatedRequest.error = data.result.error || currentRequest.error;
            updatedRequest.requestPacket = data.result.requestPacket || currentRequest.requestPacket;
            updatedRequest.responsePacket = data.result.responsePacket || currentRequest.responsePacket;
          } else if (currentRequest.status === 'pending') {
            // 如果没有结果但状态是 pending，更新为 testing（请求开始）
            updatedRequest.status = 'testing';
          }
          
          // 使用 splice 替换整个对象以确保响应式更新（Vue 3 推荐方式）
          // 这样可以确保 Vue 的响应式系统能够检测到变化
          tab.batchTestRequests.splice(requestIndex, 1, updatedRequest);
        }
        
        // 实时更新批量测试结果表格
        if (!tab.batchTestResults) {
          tab.batchTestResults = {
            total: data.total || 0,
            success: data.success || 0,
            failed: data.failed || 0,
            results: [],
          };
        }
        
        // 更新统计信息
        tab.batchTestResults.total = data.total || tab.batchTestResults.total;
        tab.batchTestResults.success = data.success || tab.batchTestResults.success;
        tab.batchTestResults.failed = data.failed || tab.batchTestResults.failed;
        
        // 更新或添加结果 - 实时更新表格
        if (data.result) {
          // 使用 requestId 或 requestID 来匹配（兼容不同的大小写）
          const resultId = data.result.requestId || data.result.requestID;
          const existingIndex = tab.batchTestResults.results.findIndex(r => 
            (r.requestId || r.requestID) === resultId
          );
          if (existingIndex >= 0) {
            // 更新现有结果 - Vue 3 中直接替换数组元素即可触发响应式
            tab.batchTestResults.results.splice(existingIndex, 1, data.result);
          } else {
            // 添加新结果
            tab.batchTestResults.results.push(data.result);
          }
        }
      });
      
      // 监听批量测试完成
      EventsOn('batchTestComplete', (data) => {
        const tab = this.tabs.find(t => t.id === tabId);
        if (!tab || !tab.isBatchTest) return;
        
        tab.batchTestResults = data;
        tab.batchTesting = false;
        ElMessage.success(`批量测试完成：成功 ${data.success}，失败 ${data.failed}`);
        
        // 清理事件监听
        this.cleanupBatchTestListeners();
      });
      
      // 监听批量测试取消
      EventsOn('batchTestCancelled', (data) => {
        const tab = this.tabs.find(t => t.id === tabId);
        if (!tab || !tab.isBatchTest) return;
        
        tab.batchTesting = false;
        ElMessage.warning('批量测试已取消');
        
        // 清理事件监听
        this.cleanupBatchTestListeners();
      });
    },
    cleanupBatchTestListeners() {
      if (this.currentBatchTestTabId) {
        EventsOff('batchRequestResult');
        EventsOff('batchTestComplete');
        EventsOff('batchTestCancelled');
        this.currentBatchTestTabId = null;
      }
    },
    cancelBatchTest() {
      // 发送取消事件到后端
      EventsEmit('cancelBatchTest');
      // 更新当前标签页状态
      const tab = this.tabs.find(t => t.id === this.activeTabId);
      if (tab && tab.isBatchTest) {
        tab.batchTesting = false;
      }
    },
    parseRequestPacket(packet, options = {}) {
      const forceHttps = !!options.forceHttps;
      if (!packet || typeof packet !== 'string') {
        ElMessage.warning('暂无可解析的请求包');
        return null;
      }
      // 保留原始换行（尤其是 multipart 边界依赖 CRLF），仅在解析头部时做宽松拆分
      const rawPacket = packet;
      // 定位首个空行，空行之前是请求行+头，之后是原始 body（保持原样）
      const sepCRLF = '\r\n\r\n';
      const sepLF = '\n\n';
      let headerPart = rawPacket;
      let bodyPart = '';
      let sepLen = 0;
      const idxCRLF = rawPacket.indexOf(sepCRLF);
      if (idxCRLF >= 0) {
        headerPart = rawPacket.slice(0, idxCRLF);
        bodyPart = rawPacket.slice(idxCRLF + sepCRLF.length);
        sepLen = sepCRLF.length;
      } else {
        const idxLF = rawPacket.indexOf(sepLF);
        if (idxLF >= 0) {
          headerPart = rawPacket.slice(0, idxLF);
          bodyPart = rawPacket.slice(idxLF + sepLF.length);
          sepLen = sepLF.length;
        }
      }

      const lines = headerPart.split(/\r?\n/).filter(l => l.trim() !== '');
      if (!lines.length) {
        ElMessage.error('请求包内容为空，无法解析');
        return null;
      }
      const requestLine = lines.shift().trim();
      const [methodRaw, pathRaw] = requestLine.split(/\s+/);
      if (!methodRaw || !pathRaw) {
        ElMessage.error('无法解析请求行');
        return null;
      }

      const headers = [];
      let host = '';
      let contentType = '';

      for (const line of lines) {
        const [k, ...rest] = line.split(':');
        if (!k) continue;
        const key = k.trim();
        const value = rest.join(':').trim();
        if (!key) continue;
        const lower = key.toLowerCase();
        if (lower === 'host') {
          host = value;
          continue; // Host 由 URL 自动生成，避免导入后重复
        }
        if (lower === 'content-type') contentType = value.toLowerCase();
        headers.push({ id: nanoid(), key, value, active: true });
      }

      // 保持原始 body，避免 CRLF 被破坏
      const rawBody = bodyPart;
      let url = pathRaw;
      const targetScheme = forceHttps ? 'https' : 'http';

      const isAbsolute = /^https?:\/\//i.test(pathRaw);
      if (!isAbsolute) {
        if (host) {
          const path = pathRaw.startsWith('/') ? pathRaw : `/${pathRaw}`;
          url = `${targetScheme}://${host}${path}`;
        } else {
          url = pathRaw;
        }
      } else if (forceHttps && /^http:\/\//i.test(pathRaw)) {
        url = pathRaw.replace(/^http:\/\//i, 'https://');
      }

      let params = [];
      try {
        const helperUrl = /^https?:\/\//i.test(url)
          ? url
          : `${targetScheme}://placeholder${url.startsWith('/') ? '' : '/'}${url}`;
        const u = new URL(helperUrl);
        u.searchParams.forEach((value, key) => {
          params.push({ id: nanoid(), key, value, active: true });
        });
        const pathnameOnly = u.pathname || '/';
        if (!/^https?:\/\//i.test(url) && host) {
          url = `${targetScheme}://${host}${pathnameOnly}`;
        } else if (/^https?:\/\//i.test(url)) {
          const normalizedUrl = `${u.protocol}//${u.host}${pathnameOnly}`;
          url = forceHttps && normalizedUrl.toLowerCase().startsWith('http://')
            ? normalizedUrl.replace(/^http:\/\//i, 'https://')
            : normalizedUrl;
        } else {
          url = pathnameOnly;
        }
      } catch {
        // 忽略 URL 解析错误
      }

      const ct = (contentType || '').toLowerCase();
      let bodyType = 'raw';
      let bodyRaw = rawBody;
      let formData = [];

      if (ct.includes('application/json')) {
        bodyType = 'json';
      } else if (ct.includes('application/x-www-form-urlencoded')) {
        bodyType = 'urlencoded';
        const sp = new URLSearchParams(rawBody);
        const items = [];
        sp.forEach((value, key) => {
          items.push({ id: nanoid(), key, value, type: 'text', files: [], active: true });
        });
        formData = items;
        bodyRaw = '';
      } else if (ct.includes('multipart/form-data')) {
        // 对 multipart 默认走 RAW，保持边界和原始内容，避免改写
        bodyType = 'raw';
      }

      return {
        id: nanoid(),
        name: url || pathRaw,
        url,
        method: (methodRaw || 'get').toLowerCase(),
        headers,
        params,
        body: {
          bodyType,
          bodyRaw,
          formData,
        },
        rawPacket: packet,
        coll_id: this.currentRequest.coll_id || (this.collections[0]?.id || ''),
      };
    },
    applyParsedPacketRequest(parsedRequest, requestName = '', options = {}) {
      if (!parsedRequest) return;
      const base = this.createEmptyRequest();
      const merged = {
        ...base,
        ...parsedRequest,
        name: requestName || parsedRequest.name || base.name,
        coll_id: parsedRequest.coll_id || base.coll_id,
      };

      this.currentRequest = merged;
      this.response = null;
      this.responseError = '';
      this.activeReqTab = 'params';
      this.activeRespTab = 'body';
      this.respViewMode = 'pretty';

      const title = `${(merged.method || 'get').toUpperCase()} ${merged.name || merged.url || '数据包请求'}`;
      let tab = this.tabs.find(t => t.id === this.activeTabId && !t.isBatchTest);

      if (!options.keepCurrentTab) {
        if (!tab) {
          const tabId = `pkt-${merged.id}`;
          tab = {
            id: tabId,
            title,
            collId: merged.coll_id,
            reqId: merged.id,
            request: JSON.parse(JSON.stringify(merged)),
            isNew: true,
            response: null,
            responseError: '',
            activeRespTab: 'body',
            respViewMode: 'pretty',
          };
          this.tabs.push(tab);
          this.activeTabId = tabId;
        } else {
          tab.request = JSON.parse(JSON.stringify(merged));
          tab.title = title;
          tab.collId = merged.coll_id;
          tab.reqId = merged.id;
          tab.isNew = true;
          tab.response = null;
          tab.responseError = '';
        }
      } else if (tab) {
        tab.request = JSON.parse(JSON.stringify(merged));
        tab.title = title;
        tab.collId = merged.coll_id;
        tab.reqId = merged.id;
        tab.isNew = true;
        tab.response = null;
        tab.responseError = '';
      }
    },
    handlePacketSend(payload) {
      this.packetCancelledByUser = false;
      this.packetCancelingRequest = false;
      const parsed = this.parseRequestPacket(payload?.requestPacket, { forceHttps: payload?.forceHttps });
      if (!parsed) return;
      const req = {
        id: parsed.id || nanoid(),
        name: parsed.name || '',
        url: parsed.url || '',
        method: parsed.method || 'get',
        headers: (parsed.headers || []).filter(h => h.key),
        params: (parsed.params || []).filter(p => p.key),
        body: parsed.body || { bodyType: 'json', bodyRaw: '', formData: [] },
        coll_id: parsed.coll_id || '',
      };

      const syncResponseTo = payload?.syncResponseTo;

      this.packetViewerLoading = true;
      InvokeRequest(req)
        .then((res) => {
          const userCancelled = this.packetCancelledByUser;
          let responsePacket = '';
          let requestPacket = '';

          if (userCancelled) {
            responsePacket = '请求已取消';
            requestPacket = payload?.requestPacket || parsed.rawPacket || '';
          } else if (res.success && res.data && !res.data?.errorContent) {
            const resp = res.data;
            responsePacket =
              resp.responsePacket ||
              (() => {
                const statusLine = `HTTP/${resp.httpVersion || '1.1'} ${resp.httpStatus || resp.statusCode || ''}`;
                const headerLines = (resp.headers || []).map(h => `${h.key}: ${h.value}`);
                const body = resp.bodyContent || '';
                return `${statusLine}\r\n${headerLines.join('\r\n')}\r\n\r\n${body}`;
              })();
            requestPacket = payload?.requestPacket || parsed.rawPacket || resp.requestPacket || '';
            ElMessage.success('请求完成');
          } else {
            responsePacket = userCancelled ? '请求已取消' : res.msg || res.data?.errorContent || '请求失败';
            requestPacket = payload?.requestPacket || parsed.rawPacket || '';
            ElMessage.error(responsePacket);
          }

          this.selectedPacket = {
            ...(this.selectedPacket || {}),
            requestName: this.selectedPacket?.requestName || payload?.title || '数据包请求',
            requestPacket,
            responsePacket,
          };

          if (syncResponseTo === 'raw') {
            this.packetInputForm.responsePacket = responsePacket;
          }
        })
        .catch((e) => {
          const errText = this.packetCancelledByUser ? '请求已取消' : e?.message || '请求异常';
          this.selectedPacket = {
            ...(this.selectedPacket || {}),
            requestName: this.selectedPacket?.requestName || payload?.title || '数据包请求',
            requestPacket: payload?.requestPacket || parsed.rawPacket || '',
            responsePacket: errText,
          };
          if (syncResponseTo === 'raw') {
            this.packetInputForm.responsePacket = errText;
          }
          if (!this.packetCancelledByUser) {
            ElMessage.error(errText);
          }
        })
        .finally(() => {
          this.packetViewerLoading = false;
          this.packetCancelingRequest = false;
          this.packetCancelledByUser = false;
        });
    },
    handlePacketCancel() {
      if (!this.packetViewerLoading || this.packetCancelingRequest) return;
      this.packetCancelledByUser = true;
      this.packetCancelingRequest = true;
      try {
        EventsEmit('cancelRequest');
        ElMessage.info('正在取消请求...');
      } catch (e) {
        console.error('取消数据包请求失败', e);
        ElMessage.error('取消数据包请求失败');
        this.packetCancelingRequest = false;
      }
    },
    handlePacketSave(payload) {
      const parsed = this.parseRequestPacket(payload?.requestPacket, { forceHttps: payload?.forceHttps });
      if (!parsed) return;
      const name = this.selectedPacket?.requestName || payload?.title || parsed.name;
      this.applyParsedPacketRequest(parsed, name);
      this.saveForm.name = name || parsed.url || '导入请求';
      this.saveForm.collId = this.currentRequest.coll_id || this.saveForm.collId || (this.collections[0]?.id || '');
      this.packetViewerVisible = false;
      this.openSaveDialog();
    },
    async handleUploadBypass(payload) {
      // 自动关闭数据包详情对话框
      this.packetViewerVisible = false
      
      // 保存 payload 信息
      this.pendingBypassPayload = payload
      
      // 重置 bypass 配置为默认值
      this.bypassTestConfig = {
        concurrency: 10,
        retryCount: 0,
      }
      
      // 显示配置对话框
      this.$nextTick(() => {
        this.bypassTestConfigDialogVisible = true
      })
    },
    confirmBypassTestConfig() {
      // 验证配置
      if (this.bypassTestConfig.concurrency < 1 || this.bypassTestConfig.concurrency > 1000) {
        ElMessage.warning('并发数必须在1-1000之间')
        return
      }
      if (this.bypassTestConfig.retryCount < 0 || this.bypassTestConfig.retryCount > 10) {
        ElMessage.warning('重试次数必须在0-10之间')
        return
      }
      
      if (!this.pendingBypassPayload) {
        ElMessage.error('Bypass测试数据丢失，请重新操作')
        this.bypassTestConfigDialogVisible = false
        return
      }
      
      // 关闭对话框
      this.bypassTestConfigDialogVisible = false
      
      // 开始 bypass 测试
      this.startBypassTest(this.pendingBypassPayload, this.bypassTestConfig)
      this.pendingBypassPayload = null
    },
    async startBypassTest(payload, config) {
      // 执行 bypass 测试
      this.bypassConfig.requestPacket = payload.requestPacket
      this.bypassConfig.forceProtocol = payload.forceHttps ? 'HTTPS' : (this.bypassConfig.forceProtocol || '自动检测')
      
      // 创建 bypass 测试标签页
      const tabId = `bypass-${nanoid()}`
      const title = '文件上传 Bypass 测试'
      
      const bypassTestTab = {
        id: tabId,
        title: title,
        isBypassTest: true,
        bypassTesting: true,
        bypassTestResults: {
          total: 0,
          success: 0,
          failed: 0,
          results: []
        },
        requestPacket: payload.requestPacket,
      }
      
      this.tabs.push(bypassTestTab)
      this.activeTabId = tabId

      // 清理之前的事件监听
      this.cleanupBypassTestListeners()

      // 监听 bypass 测试结果事件
      EventsOn('bypassTestResult', (data) => {
        const tab = this.tabs.find(t => t.id === tabId)
        if (!tab || !tab.isBypassTest) return
        
        if (data.index !== undefined) {
          // 更新或添加结果
          if (data.result) {
            // 有结果，更新或添加
            const existingIndex = tab.bypassTestResults.results.findIndex(r => r.index === data.result.index)
            if (existingIndex >= 0) {
              // 更新现有结果
              tab.bypassTestResults.results.splice(existingIndex, 1, data.result)
            } else {
              // 添加新结果，按索引排序插入
              let insertIndex = tab.bypassTestResults.results.length
              for (let i = 0; i < tab.bypassTestResults.results.length; i++) {
                if (tab.bypassTestResults.results[i].index > data.result.index) {
                  insertIndex = i
                  break
                }
              }
              tab.bypassTestResults.results.splice(insertIndex, 0, data.result)
            }
          }
          // 更新统计信息
          tab.bypassTestResults.total = data.total || tab.bypassTestResults.total
          tab.bypassTestResults.success = data.success || tab.bypassTestResults.success
          tab.bypassTestResults.failed = data.failed || tab.bypassTestResults.failed
        }
      })

      // 监听 bypass 测试完成
      EventsOn('bypassTestComplete', (data) => {
        const tab = this.tabs.find(t => t.id === tabId)
        if (!tab || !tab.isBypassTest) return
        
        tab.bypassTestResults = {
          ...tab.bypassTestResults,
          total: data.total || tab.bypassTestResults.total,
          success: data.success || tab.bypassTestResults.success,
          failed: data.failed || tab.bypassTestResults.failed,
        }
        tab.bypassTesting = false
        ElMessage.success(`测试完成：成功 ${data.success}，失败 ${data.failed}`)
        this.cleanupBypassTestListeners()
      })

      // 监听 bypass 测试取消
      EventsOn('bypassTestCancelled', (data) => {
        const tab = this.tabs.find(t => t.id === tabId)
        if (!tab || !tab.isBypassTest) return
        
        tab.bypassTesting = false
        ElMessage.warning('测试已取消')
        this.cleanupBypassTestListeners()
      })

      try {
        const res = await UploadBypassTest({
          requestPacket: this.bypassConfig.requestPacket,
          language: this.bypassConfig.language || 'all',
          forceProtocol: this.bypassConfig.forceProtocol || '自动检测',
          proxyHost: this.bypassConfig.proxyHost || '',
          proxyPort: this.bypassConfig.proxyPort || '',
          payloads: [],
          concurrency: config.concurrency,
          retryCount: config.retryCount,
        })

        if (!res.success) {
          ElMessage.error(res.msg || '测试启动失败')
          const tab = this.tabs.find(t => t.id === tabId)
          if (tab) {
            tab.bypassTesting = false
          }
          this.cleanupBypassTestListeners()
        }
      } catch (error) {
        ElMessage.error('测试异常: ' + (error.message || error))
        const tab = this.tabs.find(t => t.id === tabId)
        if (tab) {
          tab.bypassTesting = false
        }
        this.cleanupBypassTestListeners()
      }
    },
    cleanupBypassTestListeners() {
      EventsOff('bypassTestResult')
      EventsOff('bypassTestComplete')
      EventsOff('bypassTestCancelled')
    },
    cancelBypassTest() {
      // 发送取消事件到后端
      EventsEmit('cancelBypassTest')
    },
    viewBypassResult(row) {
      // 查看 bypass 测试结果的详细信息
      if (!row.requestPacket && !row.responsePacket) {
        ElMessage.warning('该测试结果还没有数据包信息')
        return
      }
      this.selectedPacket = {
        requestName: `Bypass Payload #${row.index}`,
        requestPacket: row.requestPacket || '',
        responsePacket: row.responsePacket || '',
      }
      this.packetViewerEditable = false
      this.packetViewerVisible = true
    },
    viewBatchTestPacket(row) {
      if (!row.requestPacket && !row.responsePacket) {
        ElMessage.warning('该请求还没有数据包信息');
        return;
      }
      this.selectedPacket = {
        requestName: row.name,
        requestPacket: row.requestPacket || '',
        responsePacket: row.responsePacket || '',
      };
      this.packetViewerEditable = false;
      this.packetViewerVisible = true;
    },
    getMethodTagType(method) {
      const m = (method || 'get').toLowerCase();
      switch (m) {
        case 'get':
          return 'success';
        case 'post':
          return 'warning';
        case 'put':
          return 'primary';
        case 'delete':
          return 'danger';
        default:
          return 'info';
      }
    },
    handleRequestCommand(cmd, coll, req) {
      if (cmd === 'rename') {
        this.renameRequest(coll, req);
      } else if (cmd === 'duplicate') {
        this.duplicateRequest(coll, req);
      } else if (cmd === 'delete') {
        this.deleteRequest(coll, req);
      }
    },
    async renameRequest(coll, req) {
      const resPrompt = await ElMessageBox.prompt('请输入新的请求名称', '重命名请求', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputValue: req.name || '',
      }).catch(() => null);
      if (!resPrompt || !resPrompt.value) return;
      try {
        const res = await RenameRequest(coll.id, req.id, resPrompt.value);
        if (res.success) {
          ElMessage.success(res.msg || '重命名成功');
          // 更新集合列表，保持已加载的请求列表
          const existingColl = this.collections.find(c => c.id === coll.id);
          this.collections = (res.data || []).map(c => ({
            id: c.id,
            name: c.name,
            requests: existingColl?.requests || this.collectionRequestsCache[c.id] || [],
            requestsCount: (c.requests || []).length,
          }));
          // 更新缓存
          (res.data || []).forEach(c => {
            if (c.requests && c.requests.length > 0) {
              this.collectionRequestsCache[c.id] = c.requests || [];
            }
          });
        } else {
          ElMessage.error(res.msg || '重命名失败');
        }
      } catch {
        ElMessage.error('重命名失败');
      }
    },
    async duplicateRequest(coll, req) {
      try {
        const res = await DuplicateRequest(coll.id, req.id);
        if (res.success) {
          ElMessage.success(res.msg || '复制成功');
          // 更新集合列表，保持已加载的请求列表
          const existingColl = this.collections.find(c => c.id === coll.id);
          this.collections = (res.data || []).map(c => ({
            id: c.id,
            name: c.name,
            requests: c.id === coll.id ? (existingColl?.requests || this.collectionRequestsCache[c.id] || []) : (this.collectionRequestsCache[c.id] || []),
            requestsCount: (c.requests || []).length,
          }));
          // 更新缓存
          (res.data || []).forEach(c => {
            if (c.requests && c.requests.length > 0) {
              this.collectionRequestsCache[c.id] = c.requests || [];
            }
          });
          // 如果当前集合已展开，刷新请求列表
          if (this.activeCollectionNames.includes(coll.id)) {
            const updatedColl = this.collections.find(c => c.id === coll.id);
            if (updatedColl && this.collectionRequestsCache[coll.id]) {
              updatedColl.requests = this.collectionRequestsCache[coll.id];
            }
          }
        } else {
          ElMessage.error(res.msg || '复制失败');
        }
      } catch {
        ElMessage.error('复制失败');
      }
    },
    async deleteRequest(coll, req) {
      try {
        await confirmDangerousAction({
          title: '删除请求',
          message: `确定删除请求「${req.name || req.url || req.id}」吗？`,
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      try {
        const res = await DeleteRequest(coll.id, req.id);
        if (res.success) {
          ElMessage.success(res.msg || '删除成功');
          // 更新集合列表，保持已加载的请求列表
          const existingColl = this.collections.find(c => c.id === coll.id);
          this.collections = (res.data || []).map(c => ({
            id: c.id,
            name: c.name,
            requests: c.id === coll.id ? (existingColl?.requests || this.collectionRequestsCache[c.id] || []) : (this.collectionRequestsCache[c.id] || []),
            requestsCount: (c.requests || []).length,
          }));
          // 更新缓存
          (res.data || []).forEach(c => {
            if (c.requests && c.requests.length > 0) {
              this.collectionRequestsCache[c.id] = c.requests || [];
            }
          });
          // 如果当前集合已展开，刷新请求列表
          if (this.activeCollectionNames.includes(coll.id)) {
            const updatedColl = this.collections.find(c => c.id === coll.id);
            if (updatedColl && this.collectionRequestsCache[coll.id]) {
              updatedColl.requests = this.collectionRequestsCache[coll.id];
            }
          }
        } else {
          ElMessage.error(res.msg || '删除失败');
        }
      } catch {
        ElMessage.error('删除失败');
      }
    },
    handleRequestSelect(key) {
      const [collId, reqId] = key.split('::');
      this.activeRequestKey = key;
      this.fetchRequest(collId, reqId);
    },
    async fetchRequest(collId, reqId) {
      try {
        const res = await GetRequest(reqId, collId);
        console.log('GetRequest 返回结果:', res);
        if (res.success && res.data) {
          const r = res.data;
          console.log('获取到的请求数据:', r);

          // 先创建标签页，再加载数据
          const tabInfo = {
            collId: collId,
            reqId: r.id || r.ID,
            name: r.name || r.Name,
            url: r.url || r.Url,
            method: r.method || r.Method,
          };

          console.log('准备创建标签页，数据:', tabInfo);

          // ensureTabForRequest 内部已经会加载数据到 currentRequest
          // 所以这里只需要确保标签页存在并激活即可
          await this.ensureTabForRequest(tabInfo, r);
          
          console.log('加载后的 currentRequest:', this.currentRequest);
        }
      } catch (error) {
        console.error('加载请求失败:', error);
        ElMessage.error('加载请求失败');
      }
    },
    async handleChangeRequestCollection(newCollId) {
      // 仅前端切换集合，实际保存时会写入 coll_id
      this.currentRequest.coll_id = newCollId;
    },
    handleCreateRequest() {
      const baseCollId = this.currentRequest.coll_id || (this.collections[0] && this.collections[0].id) || '';
      const next = this.createEmptyRequest();
      next.coll_id = baseCollId;
      this.currentRequest = next;
      this.activeReqTab = 'params';
      this.response = null;
      this.responseError = '';
      this.activeRespTab = 'body';
      this.respViewMode = 'pretty';
      this.activeRequestKey = '';
      const tabId = `new-${next.id}`;
      this.tabs.push({
        id: tabId,
        title: '新请求',
        collId: baseCollId,
        reqId: next.id,
        method: (next.method || 'get').toUpperCase(),
        isNew: true,
        request: JSON.parse(JSON.stringify(next)),
        response: null,
        responseError: '',
        activeRespTab: 'body',
        respViewMode: 'pretty',
      });
      this.activeTabId = tabId;
    },
    handleRequestRowClick(coll, row) {
      if (!row || !row.id) return;
      const key = `${coll.id}::${row.id}`;
      this.activeRequestKey = key;
      this.fetchRequest(coll.id, row.id);
    },
    requestRowClass(coll, row) {
      const key = `${coll.id}::${row.id}`;
      return key === this.activeRequestKey ? 'is-active-req-row' : '';
    },
    async handleChooseFile(rowIndex) {
      try {
        const res = await ChoseFile();
        if (res.success && Array.isArray(res.data) && res.data.length) {
          const next = this.currentRequest.body.formData.slice();
          if (!next[rowIndex]) return;
          next[rowIndex] = {
            ...next[rowIndex],
            files: res.data,
            type: 'file',
          };
          this.currentRequest.body.formData = next;
        } else if (!res.success) {
          ElMessage.error(res.msg || '选择文件失败');
        }
      } catch {
        ElMessage.error('选择文件失败');
      }
    },
    normalizeRequestForSend() {
      const r = this.currentRequest || {};

      // 只保留有 key 的请求参数
      const activeParams = (r.params || []).filter(p => p && p.key);

      // 先拿到用户自己配置的 header
      const rawHeaders = (r.headers || []).filter(h => h && h.key);
      const headerMap = new Map();
      rawHeaders.forEach(h => {
        const key = String(h.key || '').trim();
        if (!key) return;
        headerMap.set(key.toLowerCase(), h);
      });

      const bodyType = r.body?.bodyType || 'json';
      const bodyRaw = r.body?.bodyRaw || '';
      const formData = r.body?.formData || [];

      // 根据当前请求情况自动补充一些常用 Header（仅当用户未手动设置时）
      const ensureHeader = (key, value) => {
        const lower = key.toLowerCase();
        if (headerMap.has(lower)) return;
        const headerItem = {
          id: nanoid(),
          key,
          value,
          active: true,
        };
        headerMap.set(lower, headerItem);
      };

      // User-Agent
      ensureHeader('User-Agent', 'EasyTools-Restmate/1.0');

      // Accept
      ensureHeader('Accept', 'application/json, text/plain, */*');

      // Connection
      ensureHeader('Connection', 'keep-alive');

      // Content-Type：根据 body 类型自动补充（用户已经设置的情况下绝不覆盖）
      const hasContentType = headerMap.has('content-type');
      if (!hasContentType) {
        if (bodyType === 'json' && bodyRaw) {
          ensureHeader('Content-Type', 'application/json; charset=utf-8');
        } else if (bodyType === 'urlencoded' && Array.isArray(formData) && formData.some(i => i && i.key && i.active !== false)) {
          ensureHeader('Content-Type', 'application/x-www-form-urlencoded');
        }
        // formdata / raw 这里不自动加：
        // - formdata 让底层自己带 boundary
        // - raw 完全由用户自己控制 Content-Type
      }

      // 组装为数组（保持原有顺序，自动补的附在后面）
      const finalHeaders = [];
      headerMap.forEach(h => finalHeaders.push(h));

      // 同步回 UI，让用户能看到自动补的 Header，且不会重复添加
      this.currentRequest.headers = finalHeaders;

      return {
        id: r.id || nanoid(),
        name: r.name || '',
        url: r.url || '',
        method: (r.method || 'get').toLowerCase(),
        headers: finalHeaders,
        params: activeParams,
        body: {
          bodyType,
          bodyRaw,
          formData,
        },
        coll_id: r.coll_id || '',
      };
    },
    buildRequestPacket(req) {
      const method = (req.method || 'get').toUpperCase();
      const urlWithParams = this.buildUrlWithParams(req.url || '', req.params || []);
      let path = urlWithParams || '/';
      let host = '';
      try {
        const u = new URL(urlWithParams);
        path = `${u.pathname || '/'}${u.search || ''}`;
        host = u.host;
      } catch {
        // ignore parse errors
      }
      const lines = [`${method} ${path || '/'} HTTP/1.1`];
      const headers = (req.headers || []).filter(h => h.key);
      const hasHostHeader = headers.some(h => (h.key || '').toLowerCase() === 'host');
      if (host && !hasHostHeader) {
        lines.push(`Host: ${host}`);
      }
      headers.forEach(h => {
        lines.push(`${h.key}: ${h.value}`);
      });

      let body = '';
      const bodyType = req.body?.bodyType || 'json';
      if (bodyType === 'json') {
        body = req.body?.bodyRaw || '';
      } else if (bodyType === 'urlencoded') {
        const sp = new URLSearchParams();
        (req.body?.formData || []).filter(i => i.key && i.active !== false).forEach(i => {
          sp.append(i.key, i.value || '');
        });
        body = sp.toString();
        if (body) {
          lines.push('Content-Type: application/x-www-form-urlencoded');
        }
      } else if (bodyType === 'formdata') {
        // 仅序列化文本字段，保留文件提示
        const boundary = '----EasyToolsBoundary';
        lines.push(`Content-Type: multipart/form-data; boundary=${boundary}`);
        const parts = [];
        (req.body?.formData || []).filter(i => i.key && i.active !== false).forEach(i => {
          if (i.type === 'file' && Array.isArray(i.files) && i.files.length) {
            const firstFile = i.files[0];
            let filename = 'file';
            let contentType = 'application/octet-stream';

            if (typeof firstFile === 'string') {
              // Wails 返回的是文件路径字符串，这里从路径中提取文件名和扩展名
              const segments = firstFile.split(/[\\/]/);
              filename = segments[segments.length - 1] || 'file';
              const dotIdx = filename.lastIndexOf('.');
              const ext = dotIdx >= 0 ? filename.slice(dotIdx + 1).toLowerCase() : '';
              if (ext) {
                const extMap = {
                  'txt': 'text/plain',
                  'html': 'text/html',
                  'htm': 'text/html',
                  'json': 'application/json',
                  'js': 'application/javascript',
                  'css': 'text/css',
                  'png': 'image/png',
                  'jpg': 'image/jpeg',
                  'jpeg': 'image/jpeg',
                  'gif': 'image/gif',
                  'bmp': 'image/bmp',
                  'webp': 'image/webp',
                  'svg': 'image/svg+xml',
                  'pdf': 'application/pdf',
                  'zip': 'application/zip',
                  'rar': 'application/x-rar-compressed',
                  '7z': 'application/x-7z-compressed',
                  'xml': 'application/xml',
                  'csv': 'text/csv',
                  'mp3': 'audio/mpeg',
                  'mp4': 'video/mp4',
                };
                if (extMap[ext]) {
                  contentType = extMap[ext];
                }
              }
            } else if (firstFile && typeof firstFile === 'object') {
              filename = firstFile.Name || firstFile.name || 'file';
            }

            parts.push(
              `--${boundary}\r\n` +
              `Content-Disposition: form-data; name="${i.key}"; filename="${filename}"\r\n` +
              `Content-Type: ${contentType}\r\n\r\n` +
              `<文件内容略>`
            );
          } else {
            parts.push(`--${boundary}\r\nContent-Disposition: form-data; name="${i.key}"\r\n\r\n${i.value || ''}`);
          }
        });
        parts.push(`--${boundary}--`);
        body = parts.join('\r\n');
      } else {
        body = req.body?.bodyRaw || '';
      }

      return `${lines.join('\r\n')}\r\n\r\n${body}`;
    },
    buildResponsePacket(resData, fallback = '') {
      if (!resData) return fallback || '';
      if (resData.responsePacket) return resData.responsePacket;
      const status = resData.httpStatus || resData.statusCode || '';
      const version = resData.httpVersion || '1.1';
      const lines = [`HTTP/${version} ${status}`];
      (resData.headers || []).forEach(h => {
        if (h.key) lines.push(`${h.key}: ${h.value}`);
      });
      if (resData.isBinary && resData.binaryData) {
        return `${lines.join('\r\n')}\r\n\r\n<二进制响应，已省略，建议使用“保存文件”>`;
      }
      const body = resData.bodyContent || '';
      return `${lines.join('\r\n')}\r\n\r\n${body}`;
    },
    handleCancelRequest() {
      if (!this.sending || this.cancelingRequest) return;
      this.cancelledByUser = true;
      this.cancelingRequest = true;
      try {
        EventsEmit('cancelRequest');
        ElMessage.info('正在取消请求...');
      } catch (e) {
        console.error('取消请求失败', e);
        ElMessage.error('取消请求失败');
        this.cancelingRequest = false;
      }
    },
    async handleSend() {
      // 检查 URL 是否为空或只包含空格
      const url = (this.currentRequest.url || '').trim();
      if (!url) {
        ElMessage.warning('请先输入 URL');
        return;
      }
      this.cancelledByUser = false;
      this.sending = true;
      this.response = null;
      this.responseError = '';
      this.activeRespTab = 'body';
      this.respViewMode = 'pretty';
      const id = this.activeTabId;
      const tab = this.tabs.find(t => t.id === id);
      if (tab) {
        tab.response = null;
        tab.responseError = '';
        tab.activeRespTab = 'body';
        tab.respViewMode = 'pretty';
      }
      try {
        const req = this.normalizeRequestForSend();
        const packetTitle = req.name || req.url || '发送请求';
        const requestPacket = this.buildRequestPacket(req);
        this.lastPacketTitle = packetTitle;
        this.lastRequestPacket = requestPacket;
        if (tab) {
          tab.packetTitle = packetTitle;
          tab.requestPacket = requestPacket;
          tab.responsePacket = '';
        }
        const res = await InvokeRequest(req);
        const hasErrorContent = !!res?.data?.errorContent;
        const userCancelled = this.cancelledByUser;
        if (res.success && res.data && !hasErrorContent && !userCancelled) {
          this.response = res.data;
          const respPacket = this.buildResponsePacket(res.data);
          this.lastResponsePacket = respPacket;
          if (tab) {
            tab.response = res.data;
            tab.responsePacket = respPacket;
          }
        } else {
          this.responseError = userCancelled
            ? '请求已取消'
            : res?.data?.errorContent || res?.msg || '请求失败';
          this.lastResponsePacket = this.buildResponsePacket(res.data, this.responseError);
          if (tab) {
            tab.responseError = this.responseError;
            tab.responsePacket = this.lastResponsePacket;
          }
        }
      } catch (e) {
        const userCancelled = this.cancelledByUser;
        this.responseError = userCancelled ? '请求已取消' : e?.message || '请求异常';
        this.lastResponsePacket = this.responseError;
        const id2 = this.activeTabId;
        const tab2 = this.tabs.find(t => t.id === id2);
        if (tab2) {
          tab2.responseError = this.responseError;
          tab2.responsePacket = this.lastResponsePacket;
        }
      } finally {
        this.sending = false;
        this.cancelingRequest = false;
        this.cancelledByUser = false;
      }
    },
    openSendPacketViewer() {
      if (!this.hasPacketForCurrentTab) {
        ElMessage.warning('暂无可查看的数据包');
        return;
      }
      const requestPacket = this.currentRequestPacket || '';
      const responsePacket = this.currentResponsePacket || '';
      this.selectedPacket = {
        requestName: this.currentPacketTitle || '发送数据包',
        requestPacket,
        responsePacket,
      };
      this.packetViewerEditable = false;
      this.packetViewerLoading = false;
      this.packetViewerVisible = true;
    },
    async handleSaveRequest() {
      // 等待 DOM 更新
      await this.$nextTick();
      
      // 尝试从 DOM 中获取 URL 输入框的实际值（contenteditable 元素）
      let url = (this.currentRequest.url || '').trim();
      
      // 如果 currentRequest.url 为空，尝试从 DOM 中获取
      if (!url) {
        const urlInputElement = this.$el?.querySelector('.url-input.env-var-input') ||
                                this.$el?.querySelector('.url-input') ||
                                document.querySelector('.url-input.env-var-input') ||
                                document.querySelector('.url-input');
        
        if (urlInputElement) {
          // contenteditable 元素使用 textContent 或 innerText
          const domValue = (urlInputElement.textContent || urlInputElement.innerText || '').trim();
          if (domValue) {
            url = domValue;
            // 同步到 currentRequest.url
            this.currentRequest.url = domValue;
          }
        }
      }
      
      // 如果还是为空，尝试触发 blur 事件确保值同步
      if (!url) {
        const urlInputElement = this.$el?.querySelector('.url-input.env-var-input') ||
                                this.$el?.querySelector('.url-input') ||
                                document.querySelector('.url-input.env-var-input') ||
                                document.querySelector('.url-input');
        if (urlInputElement && document.activeElement === urlInputElement) {
          // 如果输入框有焦点，触发 blur 事件
          urlInputElement.blur();
          // 等待 blur 事件处理完成
          await this.$nextTick();
          url = (this.currentRequest.url || '').trim();
          // 如果还是为空，再次从 DOM 获取
          if (!url) {
            url = (urlInputElement.textContent || urlInputElement.innerText || '').trim();
            if (url) {
              this.currentRequest.url = url;
            }
          }
        }
      }
      
      // 检查 URL 是否为空或只包含空格
      if (!url) {
        ElMessage.warning('请先输入 URL');
        return;
      }
      
      let targetCollId = this.saveForm.collId;
      
      // 如果用户输入了集合名称或ID
      if (targetCollId) {
        // 检查输入的是集合ID还是集合名称
        const existingCollById = this.collections.find(c => c.id === targetCollId);
        const existingCollByName = this.collections.find(c => c.name === targetCollId);
        
        if (existingCollById) {
          // 输入的是ID，直接使用
          targetCollId = existingCollById.id;
        } else if (existingCollByName) {
          // 输入的是名称，使用对应的ID
          targetCollId = existingCollByName.id;
        } else {
          // 输入的是新集合名称，创建新集合
          const newCollectionName = targetCollId.trim();
          if (!newCollectionName) {
            ElMessage.warning('请输入有效的集合名称');
            return;
          }
          const id = nanoid();
          try {
            const createRes = await AddCollection(id, newCollectionName);
            if (createRes.success) {
              this.collections = (createRes.data || []).map(c => ({
                id: c.id,
                name: c.name,
                requests: c.requests || [],
              }));
              targetCollId = id;
              ElMessage.success(`已创建新集合「${newCollectionName}」`);
            } else {
              ElMessage.error('创建集合失败：' + (createRes.msg || '未知错误'));
              return;
            }
          } catch (e) {
            ElMessage.error('创建集合失败：' + (e?.message || '未知错误'));
            return;
          }
        }
      } else {
        // 如果没有输入，检查是否有现有集合
        if (!this.collections.length) {
          // 如果集合不存在，自动创建一个默认集合
          const defaultCollectionName = '默认集合';
          const id = nanoid();
          try {
            const createRes = await AddCollection(id, defaultCollectionName);
            if (createRes.success) {
              this.collections = (createRes.data || []).map(c => ({
                id: c.id,
                name: c.name,
                requests: c.requests || [],
              }));
              targetCollId = id;
              ElMessage.success('已自动创建默认集合');
            } else {
              ElMessage.error('自动创建集合失败：' + (createRes.msg || '未知错误'));
              return;
            }
          } catch (e) {
            ElMessage.error('自动创建集合失败：' + (e?.message || '未知错误'));
            return;
          }
        } else {
          // 使用第一个集合
          targetCollId = this.collections[0].id;
        }
      }

      this.currentRequest.name = this.saveForm.name || this.currentRequest.name || '';
      this.currentRequest.coll_id = targetCollId;

      const req = this.normalizeRequestForSend();
      this.saving = true;
      try {
        const res = await UpsertRequest(req);
        if (res.success) {
          ElMessage.success(res.msg || '请求已保存');
          if (res.data) {
            // 更新集合列表，保持已加载的请求列表
            this.collections = (res.data || []).map(c => {
              const existingColl = this.collections.find(ex => ex.id === c.id);
              return {
                id: c.id,
                name: c.name,
                requests: existingColl?.requests || this.collectionRequestsCache[c.id] || [],
                requestsCount: (c.requests || []).length,
              };
            });
            // 更新缓存
            (res.data || []).forEach(c => {
              if (c.requests && c.requests.length > 0) {
                this.collectionRequestsCache[c.id] = c.requests || [];
              }
            });
            const coll = this.collections.find(c => c.id === req.coll_id);
            const savedReq = coll && (coll.requests || []).find(r => r.id === req.id);
            if (coll && savedReq) {
              const newKey = `${coll.id}::${savedReq.id}`;
              this.activeRequestKey = newKey;
              const oldTabId = this.activeTabId || '';
              const title = `${(savedReq.method || 'GET').toUpperCase()} ${savedReq.name || savedReq.url || '未命名请求'}`;
              let tab = this.tabs.find(t => t.id === oldTabId);
              if (!tab) {
                tab = this.tabs.find(t => t.reqId === savedReq.id && t.collId === coll.id);
              }
              const requestSnapshot = JSON.parse(JSON.stringify({
                ...req,
                id: savedReq.id,
                name: savedReq.name || req.name,
                url: savedReq.url || req.url,
                method: (savedReq.method || req.method || 'get').toLowerCase(),
                coll_id: coll.id,
              }));

              if (tab) {
                tab.id = newKey;
                tab.title = title;
                tab.collId = coll.id;
                tab.reqId = savedReq.id;
                tab.request = requestSnapshot;
                tab.isNew = false;
              } else {
                this.tabs.push({
                  id: newKey,
                  title,
                  collId: coll.id,
                  reqId: savedReq.id,
                  request: requestSnapshot,
                  isNew: false,
                  response: null,
                  responseError: '',
                  activeRespTab: 'body',
                  respViewMode: 'pretty',
                });
              }
              this.activeTabId = newKey;
              // 同步当前请求的 ID 与集合信息，避免后续保存无法匹配
              this.currentRequest = {
                ...this.currentRequest,
                id: savedReq.id,
                coll_id: coll.id,
                name: savedReq.name || this.currentRequest.name,
              };
              // 更新 saveForm.collId 为集合ID，确保下次打开对话框时正确显示
              this.saveForm.collId = coll.id;
            }
          }
        } else {
          ElMessage.error(res.msg || '保存失败');
        }
      } catch {
        ElMessage.error('保存失败');
      } finally {
        this.saving = false;
        this.saveDialogVisible = false;
      }
    },
    async openSaveDialog() {
      // 等待 DOM 更新
      await this.$nextTick();
      
      // 尝试从 DOM 中获取 URL 输入框的实际值（contenteditable 元素）
      // 因为 EnvVarInput 是 contenteditable，值可能还没有同步到 v-model
      let url = (this.currentRequest.url || '').trim();
      
      // 如果 currentRequest.url 为空，尝试从 DOM 中获取
      if (!url) {
        // EnvVarInput 是 contenteditable div，类名是 env-var-input
        const urlInputElement = this.$el?.querySelector('.url-input.env-var-input') ||
                                this.$el?.querySelector('.url-input') ||
                                document.querySelector('.url-input.env-var-input') ||
                                document.querySelector('.url-input');
        
        if (urlInputElement) {
          // contenteditable 元素使用 textContent 或 innerText
          const domValue = (urlInputElement.textContent || urlInputElement.innerText || '').trim();
          if (domValue) {
            url = domValue;
            // 同步到 currentRequest.url
            this.currentRequest.url = domValue;
          }
        }
      }
      
      // 如果还是为空，尝试触发 blur 事件确保值同步
      if (!url) {
        const urlInputElement = this.$el?.querySelector('.url-input.env-var-input') ||
                                this.$el?.querySelector('.url-input') ||
                                document.querySelector('.url-input.env-var-input') ||
                                document.querySelector('.url-input');
        if (urlInputElement && document.activeElement === urlInputElement) {
          // 如果输入框有焦点，触发 blur 事件
          urlInputElement.blur();
          // 等待 blur 事件处理完成
          await this.$nextTick();
          url = (this.currentRequest.url || '').trim();
          // 如果还是为空，再次从 DOM 获取
          if (!url) {
            url = (urlInputElement.textContent || urlInputElement.innerText || '').trim();
            if (url) {
              this.currentRequest.url = url;
            }
          }
        }
      }
      
      // 检查 URL 是否为空或只包含空格
      if (!url) {
        ElMessage.warning('请先输入 URL');
        return;
      }
      if (this.saveDialogVisible) return;
      this.saveForm.name = this.currentRequest.name || '';
      // 如果有集合ID，尝试找到对应的集合名称，如果没有找到则使用ID
      const currentColl = this.currentRequest.coll_id 
        ? this.collections.find(c => c.id === this.currentRequest.coll_id)
        : null;
      this.saveForm.collId =
        (currentColl && currentColl.id) ||
        (this.collections[0] && this.collections[0].id) ||
        '';
      this.saveDialogVisible = true;
    },

    openPacketInputDialog() {
      this.selectedPacket = {
        requestName: '自定义数据包',
        requestPacket: '',
        responsePacket: '',
      };
      this.packetViewerEditable = true;
      this.packetViewerVisible = true;
    },
    openPacketViewerFromInput() {
      const requestPacket = (this.packetInputForm.requestPacket || '').trim();
      if (!requestPacket) {
        ElMessage.warning('请先粘贴请求数据包');
        return;
      }
      this.selectedPacket = {
        requestName: '自定义数据包',
        requestPacket,
        responsePacket: this.packetInputForm.responsePacket || '',
      };
      this.packetInputDialogVisible = false;
      this.packetViewerEditable = true;
      this.packetViewerVisible = true;
    },
    handleRawWorkspaceSend(payload) {
      this.handlePacketSend({
        ...payload,
        title: payload?.title || 'Raw 发包',
        syncResponseTo: 'raw',
      });
    },
    resetEmbeddedRawPacket() {
      this.packetInputForm.requestPacket = '';
      this.packetInputForm.responsePacket = '';
    },
    /** 将 HTTP 工作台当前请求序列化为 Raw 填入「数据包发包」并排编辑器 */
    syncComposerPacketToRawWorkspace() {
      const url = (this.currentRequest?.url || '').trim();
      if (!url) {
        ElMessage.warning('请先在 HTTP 工作台填写 URL，并配置 multipart（含本地文件路径）');
        this.restmateWorkspace = 'composer';
        return;
      }
      try {
        const req = this.normalizeRequestForSend();
        const requestPacket = this.buildRequestPacket(req);
        if (!requestPacket.trim()) {
          ElMessage.warning('无法根据当前请求生成数据包，请检查配置');
          return;
        }
        this.packetInputForm.requestPacket = requestPacket;
        this.packetInputForm.responsePacket = this.currentResponsePacket || '';
        this.restmateWorkspace = 'rawPacket';
      } catch (e) {
        ElMessage.error(e?.message || '生成请求包失败');
      }
    },
    checkAndLoadPacketFromStorage() {
      // 检查 URL 参数
      const urlParams = new URLSearchParams(window.location.search);
      const openPacket = urlParams.get('openPacket');
      
      // 也检查 sessionStorage，即使没有 URL 参数（支持同页面多次跳转）
      const storageKey = 'easytools:restmate:packet';
      const packetDataStr = sessionStorage.getItem(storageKey);
      
      if (openPacket === 'true' || packetDataStr) {
        // 立即清除 URL 参数，避免刷新页面时重复触发
        if (openPacket === 'true' && window.history.replaceState) {
          const cleanUrl = window.location.pathname;
          window.history.replaceState({}, document.title, cleanUrl);
        }
        
        try {
          if (packetDataStr) {
            const packetData = JSON.parse(packetDataStr);
            
            // 验证数据有效性
            if (packetData.fromWebscan && packetData.requestPacket) {
              // 检查时间戳，避免加载过期的数据（5分钟内有效）
              const now = Date.now();
              const dataAge = now - (packetData.timestamp || 0);
              if (dataAge > 5 * 60 * 1000) {
                // 数据过期，清除
                sessionStorage.removeItem(storageKey);
                return;
              }
              
              // 设置数据包内容
              this.selectedPacket = {
                requestName: packetData.requestName || '漏洞请求',
                requestPacket: packetData.requestPacket || '',
                responsePacket: packetData.responsePacket || '',
              };
              
              // 清除 sessionStorage 中的数据，避免重复加载
              sessionStorage.removeItem(storageKey);
              
              // 延迟打开 PacketViewer，确保页面已完全加载
              this.$nextTick(() => {
                setTimeout(() => {
                  this.packetViewerEditable = true;
                  this.packetViewerVisible = true;
                  ElMessage.success('已自动加载漏洞扫描数据包');
                }, 300);
              });
            }
          }
        } catch (e) {
          console.error('加载数据包失败:', e);
          ElMessage.error('数据包加载失败');
        }
      }
    },
    handleTabRemove(targetId) {
      this.persistCurrentToActiveTab();
      const idx = this.tabs.findIndex(t => t.id === targetId);
      if (idx === -1) return;
      this.tabs.splice(idx, 1);
      if (this.activeTabId === targetId) {
        const next = this.tabs[idx] || this.tabs[idx - 1];
        if (next) {
          this.activeTabId = next.id;
          this.loadRequestFromTab(next);
          this.loadResponseFromTab(next);
        } else {
          // 所有标签关闭后，新建一个空的
          this.handleCreateRequest();
        }
      }
    },
    handleTabClick(tabPane) {
      this.persistCurrentToActiveTab();
      const id = tabPane.paneName || tabPane.name || tabPane;
      const tab = this.tabs.find(t => t.id === id);
      if (!tab) return;
      this.activeTabId = id;

      // 立即加载数据，然后等待DOM更新
      this.loadRequestFromTab(tab);
      this.loadResponseFromTab(tab);
      
      // 确保DOM更新
      this.$nextTick(() => {
        // 强制更新以确保所有组件正确渲染
        this.$forceUpdate();
      });
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
        : this.activeTabId;
      
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
    // 关闭其他标签页
    handleCloseOtherTabs() {
      this.hideTabsContextMenu();
      if (this.tabs.length <= 1) return;

      const targetId = this.tabsContextMenuTabId || this.activeTabId;
      const targetTab = this.tabs.find(t => t.id === targetId);
      if (!targetTab) return;

      // 保存当前标签页的数据
      this.persistCurrentToActiveTab();

      // 仅保留目标标签页
      this.tabs = [targetTab];
      this.activeTabId = targetId;
      this.loadRequestFromTab(targetTab);
      this.loadResponseFromTab(targetTab);
    },
    // 关闭所有标签页
    handleCloseAllTabs() {
      this.hideTabsContextMenu();
      if (this.tabs.length === 0) return;
      
      // 保存当前标签页的数据
      this.persistCurrentToActiveTab();
      
      // 关闭所有标签页
      this.tabs = [];
      
      // 创建一个新的空标签页
      this.handleCreateRequest();
    },
    async formatRequestBodyJson() {
      const raw = this.currentRequest.body?.bodyRaw || '';
      if (!raw.trim()) {
        return;
      }
      try {
        const parsed = JSON.parse(raw);
        this.currentRequest.body.bodyRaw = JSON.stringify(parsed, null, 2);
        await this.runMonacoFormat(this.jsonBodyEditorRef);
        // 再次读取编辑器内容，确保格式与 Monaco 保持一致
        if (this.jsonBodyEditorRef?.getEditor) {
          const editor = this.jsonBodyEditorRef.getEditor();
          if (editor) {
            this.currentRequest.body.bodyRaw = editor.getValue();
          }
        }
        ElMessage.success('JSON 已格式化并高亮');
      } catch {
        ElMessage.error('当前内容不是有效的 JSON');
      }
    },
    handleUrlKeydown(e) {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        this.handleSend();
      }
    },
    async handleKeydown(e) {
      if ((e.ctrlKey || e.metaKey) && String(e.key).toLowerCase() === 's') {
        e.preventDefault();
        
        // 如果 URL 输入框有焦点，先触发 blur 确保值已同步
        const urlInputElement = this.$el?.querySelector('.url-input.env-var-input') ||
                                this.$el?.querySelector('.url-input') ||
                                document.querySelector('.url-input.env-var-input') ||
                                document.querySelector('.url-input');
        if (urlInputElement && document.activeElement === urlInputElement) {
          urlInputElement.blur();
          // 等待 blur 事件处理完成
          await this.$nextTick();
        }
        
        // 如果对话框已经打开，直接保存；否则打开对话框
        if (this.saveDialogVisible) {
          // 对话框已打开，直接触发保存（handleSaveRequest 内部会再次验证 URL）
          await this.handleSaveRequest();
        } else {
          // 对话框未打开，先打开对话框（openSaveDialog 内部会验证 URL）
          await this.openSaveDialog();
        }
      }
    },
  },
};
</script>

<style scoped>
.restmate-page {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 8px 10px;
  box-sizing: border-box;
  min-height: 0;
  /* 工作台强调色（参考专业发包工具的视觉焦点，略作收敛以免刺眼） */
  --restmate-accent: #ff8c00;
  --restmate-accent-strong: #e67300;
  --restmate-accent-hover: #ffa033;
  --restmate-accent-soft: rgba(255, 140, 0, 0.14);
  --restmate-accent-softer: rgba(255, 140, 0, 0.06);
}

.restmate-page--light-surface {
  background: #ffffff !important;
  color: #303133;
}

.restmate-page--light-surface .restmate-workspace-hint {
  color: #909399;
}

.restmate-embedded-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 4px 0 10px;
  flex-shrink: 0;
}

.restmate-workspace-shell--raw {
  background: #ffffff !important;
}

.restmate-page--light-surface .restmate-aside {
  background: #fafafa !important;
}

.restmate-top-shell {
  flex-shrink: 0;
}

.restmate-workspace-hint {
  margin: 4px 0 8px 10px;
  font-size: 12px;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}

/* 与全局 et-module-tabs 一致：不参与主题磨砂卡片层，去掉导航条底部分隔线 */
.restmate-workspace-tabs.no-theme-bg :deep(.el-tabs__header),
.restmate-workspace-tabs.no-theme-bg :deep(.el-tabs__nav-wrap) {
  background: transparent !important;
}

.restmate-workspace-tabs.no-theme-bg :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.restmate-workspace-tabs.no-theme-bg :deep(.el-tabs__active-bar) {
  height: 3px;
  border-radius: 3px 3px 0 0;
  background-color: var(--restmate-accent);
}

.restmate-workspace-tabs.no-theme-bg :deep(.el-tabs__item.is-active) {
  font-weight: 600;
  opacity: 1;
}

.restmate-workspace-tabs.no-theme-bg :deep(.el-tabs__item:not(.is-active)) {
  opacity: 0.72;
}

.restmate-panel {
  border-radius: 8px;
  overflow: hidden;
}

.restmate-panel :deep(.el-card__header) {
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color-lighter);
  padding: 12px 14px;
}

.restmate-panel--raw :deep(.el-card__body) {
  padding: 14px 14px 18px;
}

.restmate-panel--bypass :deep(.el-card__body) {
  padding: 14px 16px 18px;
}

.restmate-panel-lead {
  margin: 0 0 14px;
  font-size: 12px;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}

.bypass-intro-card .bypass-steps strong {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.restmate-workspace-shell .bypass-entry-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 14px;
}

.restmate-workspace-shell .bypass-entry-actions :deep(.el-button) {
  margin: 0;
}

.restmate-container {
  width: 100%;
  flex: 1;
  min-height: 0;
  border-radius: 12px;
  overflow: hidden;
}

/* HTTP 工作台：三栏分区 + 扁平边框（功能保持现有，仅强化层级与强调色） */
.restmate-workbench.restmate-container {
  border: 1px solid var(--el-border-color-light);
  background: #ffffff !important;
}

.restmate-workbench.restmate-container > .restmate-aside {
  background: transparent;
}

.restmate-workbench .coll-title,
.restmate-workbench .coll-name,
.restmate-workbench .req-name {
  color: var(--el-text-color-primary);
}

.restmate-workbench .top-tabs-bar {
  background: #fafafa !important;
}

.restmate-workbench .full-card {
  box-shadow: none;
  border: 1px solid var(--el-border-color-lighter);
  background: #ffffff !important;
}

.restmate-workbench .card-header {
  color: var(--el-text-color-primary);
}

.restmate-workbench .request-tabs :deep(.el-tabs__item.is-active) {
  color: var(--restmate-accent);
  font-weight: 600;
}

.restmate-workbench .request-tabs :deep(.el-tabs__item:hover) {
  color: var(--restmate-accent-hover);
}

.restmate-workbench .req-tabs :deep(.el-tabs__active-bar),
.restmate-workbench .resp-tabs :deep(.el-tabs__active-bar) {
  background-color: var(--restmate-accent);
  height: 2px;
  border-radius: 999px;
}

.restmate-workbench .req-tabs :deep(.el-tabs__item.is-active),
.restmate-workbench .resp-tabs :deep(.el-tabs__item.is-active) {
  font-weight: 600;
}

.restmate-workbench .resp-toolbar :deep(.el-button.el-button--primary.is-link),
.restmate-workbench .resp-toolbar :deep(.el-button.el-button--primary.is-text) {
  color: var(--restmate-accent);
}

.restmate-workbench .resp-toolbar :deep(.el-button.el-button--primary.is-link:hover),
.restmate-workbench .resp-toolbar :deep(.el-button.el-button--primary.is-text:hover) {
  color: var(--restmate-accent-strong);
}

.restmate-workbench .resp-actions :deep(.el-button.el-button--primary.is-text) {
  color: var(--restmate-accent);
}

.restmate-workbench .method-select.method-post :deep(.el-input__inner) {
  color: var(--restmate-accent);
}

.restmate-workbench .method-select :deep(.el-select-dropdown__item[data-value='post']) {
  color: var(--restmate-accent);
}

.restmate-workbench .req-method.m-post {
  background: linear-gradient(135deg, var(--restmate-accent), var(--restmate-accent-strong));
}

.resp-empty-placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 220px;
  padding: 28px 20px;
  text-align: center;
  box-sizing: border-box;
  border-radius: 10px;
  border: 1px dashed var(--el-border-color);
  background: transparent;
}

.resp-empty-icon {
  color: var(--restmate-accent);
  opacity: 0.92;
  margin-bottom: 14px;
}

.resp-empty-title {
  margin: 0 0 10px;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  letter-spacing: 0.02em;
}

.resp-empty-desc {
  margin: 0;
  font-size: 12px;
  line-height: 1.65;
  color: var(--el-text-color-secondary);
  max-width: 320px;
}

.restmate-workspace {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.restmate-workspace-scroll {
  height: 100%;
}

/* 数据包发包：与 HTTP 工作台统一的容器与橙色强调 */
.restmate-workspace-shell {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border-radius: 12px;
  border: 1px solid var(--el-border-color-light);
  box-sizing: border-box;
  padding: 12px 14px 14px;
  background: #ffffff !important;
}

.restmate-workspace-shell .workspace-surface-card.el-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  box-shadow: none;
  background: var(--el-bg-color);
  max-width: 960px;
  margin-left: auto;
  margin-right: auto;
}

.restmate-workspace-shell .workspace-card.workspace-card-surface.restmate-panel {
  border-radius: 12px;
}

.restmate-workspace-shell .workspace-surface-card :deep(.el-card__header) {
  padding: 14px 18px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: linear-gradient(105deg, var(--restmate-accent-soft) 0%, transparent 55%);
}

.restmate-workspace-shell .workspace-card-title {
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.03em;
  color: var(--el-text-color-primary);
  display: flex;
  align-items: center;
  gap: 10px;
}

.restmate-workspace-shell .workspace-card-title::before {
  content: '';
  width: 4px;
  height: 18px;
  border-radius: 2px;
  background: linear-gradient(180deg, var(--restmate-accent), var(--restmate-accent-strong));
  flex-shrink: 0;
}

.restmate-workspace-shell .workspace-card-actions {
  gap: 8px;
}

.restmate-workspace-shell :deep(.el-button--primary:not(.is-plain):not(.is-link)) {
  background: linear-gradient(135deg, var(--restmate-accent), var(--restmate-accent-strong));
  border-color: transparent;
  box-shadow: 0 6px 16px rgba(255, 140, 0, 0.3);
}

.restmate-workspace-shell :deep(.el-button--primary:not(.is-plain):not(.is-link):hover) {
  filter: brightness(1.06);
  border-color: transparent;
}

.restmate-workspace-shell :deep(.el-button.is-text.el-button--primary) {
  color: var(--restmate-accent);
}

.restmate-workspace-shell :deep(.el-button.is-text.el-button--primary:hover) {
  color: var(--restmate-accent-strong);
}

.restmate-workspace-shell .restmate-panel-lead {
  padding: 12px 14px;
  margin: 0 0 16px;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  background: transparent;
  color: var(--el-text-color-regular);
}

.restmate-workspace-shell .workspace-form :deep(.el-form-item__label) {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.restmate-workspace-shell .workspace-textarea :deep(.el-textarea__inner) {
  font-family: ui-monospace, 'Cascadia Code', monospace;
  font-size: 12px;
  line-height: 1.45;
  border-radius: 8px;
  border: 1px solid var(--el-border-color);
  background: var(--el-fill-color-light, rgba(127, 127, 127, 0.08));
  box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.06);
  transition:
    border-color 0.15s ease,
    box-shadow 0.15s ease;
}

.restmate-workspace-shell .workspace-textarea :deep(.el-textarea__inner:hover) {
  border-color: var(--restmate-accent-hover);
}

.restmate-workspace-shell .workspace-textarea :deep(.el-textarea__inner:focus) {
  border-color: var(--restmate-accent);
  box-shadow:
    inset 0 1px 2px rgba(0, 0, 0, 0.06),
    0 0 0 1px var(--restmate-accent-soft);
}

.restmate-workspace-shell .bypass-intro-card .bypass-steps {
  list-style: none;
  margin: 0 0 18px;
  padding: 0;
  counter-reset: bypass-step;
}

.restmate-workspace-shell .bypass-intro-card .bypass-steps li {
  counter-increment: bypass-step;
  position: relative;
  padding: 12px 14px 12px 52px;
  margin-bottom: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-fill-color-blank, rgba(255, 255, 255, 0.02));
  font-size: 13px;
  line-height: 1.65;
  color: var(--el-text-color-regular);
}

.restmate-workspace-shell .bypass-intro-card .bypass-steps li:last-child {
  margin-bottom: 0;
}

.restmate-workspace-shell .bypass-intro-card .bypass-steps li::before {
  content: counter(bypass-step);
  position: absolute;
  left: 12px;
  top: 12px;
  width: 28px;
  height: 28px;
  line-height: 28px;
  text-align: center;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 700;
  color: #fff;
  background: linear-gradient(135deg, var(--restmate-accent), var(--restmate-accent-strong));
  box-shadow: 0 4px 10px rgba(255, 140, 0, 0.25);
}

.restmate-workspace-shell .bypass-intro-card .bypass-steps li + li {
  margin-top: 0;
}

.restmate-workspace-shell .bypass-quick-actions {
  margin-top: 4px;
  padding-top: 16px;
  border-top: 1px dashed var(--el-border-color);
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.restmate-workspace-shell .bypass-quick-actions :deep(.el-button--primary.is-plain) {
  --el-button-bg-color: transparent;
  --el-button-border-color: var(--restmate-accent);
  --el-button-text-color: var(--restmate-accent);
  --el-button-hover-bg-color: var(--restmate-accent-soft);
  --el-button-hover-border-color: var(--restmate-accent-hover);
  --el-button-hover-text-color: var(--restmate-accent-strong);
}

.restmate-workspace-shell .bypass-quick-actions :deep(.el-button:not(.el-button--primary):hover) {
  border-color: var(--restmate-accent-hover);
  color: var(--restmate-accent);
}

/* —— 数据包发包：左说明与操作 / 右编辑区（参考三栏工具中的「配置 | 编辑器」分区）—— */
.restmate-workspace-shell--raw {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.restmate-workspace-shell--raw .raw-packet-fill-card.el-card {
  flex: 1;
  min-height: 0;
  max-width: none;
  margin: 0;
  display: flex;
  flex-direction: column;
}

.restmate-workspace-shell--raw .raw-packet-fill-card :deep(.el-card__body) {
  flex: 1;
  min-height: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
}

.raw-packet-card-header .raw-packet-header-meta {
  margin-left: auto;
  font-size: 12px;
  font-weight: 400;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
  text-align: right;
  max-width: min(440px, 58%);
}

.raw-packet-split {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: row;
  align-items: stretch;
}

.raw-packet-aside {
  width: 280px;
  flex-shrink: 0;
  box-sizing: border-box;
  padding: 12px 12px 14px;
  border-right: 1px solid var(--el-border-color-lighter);
  background: transparent;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: auto;
}

.raw-packet-collapse {
  flex: 1;
  min-height: 0;
  border: none;
  --el-collapse-border-color: transparent;
}

.raw-packet-collapse :deep(.el-collapse-item__header) {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  padding: 10px 4px;
}

.raw-packet-collapse :deep(.el-collapse-item__wrap) {
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.raw-packet-collapse :deep(.el-collapse-item__content) {
  padding: 8px 6px 12px;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.raw-packet-tip-list {
  margin: 0;
  padding-left: 1.1rem;
  line-height: 1.65;
}

.raw-packet-tip-list li + li {
  margin-top: 8px;
}

.raw-packet-collapse-p {
  margin: 0;
  line-height: 1.65;
}

.raw-packet-aside-actions {
  margin-top: auto;
  padding-top: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  flex-shrink: 0;
}

.raw-packet-aside-actions :deep(.el-button) {
  width: 100%;
  margin: 0;
}

.raw-packet-action-primary {
  font-weight: 600;
}

.raw-packet-main {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 12px 14px 10px;
  background: var(--el-fill-color-extra-light, rgba(127, 127, 127, 0.04));
}

.raw-packet-pane {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.raw-packet-pane--request {
  flex: 1;
}

.raw-packet-pane--response {
  flex-shrink: 0;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.raw-packet-pane-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.raw-packet-pane-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  padding-left: 8px;
  border-left: 3px solid var(--restmate-accent);
}

.raw-packet-textarea--request :deep(.el-textarea__inner) {
  min-height: 260px;
}

.raw-packet-textarea--response :deep(.el-textarea__inner) {
  min-height: 140px;
}

.raw-packet-main-footer-hint {
  flex-shrink: 0;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 10px;
  padding: 10px 12px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
  border-radius: 8px;
  border: 1px dashed var(--el-border-color);
  background: transparent;
}

.raw-packet-footer-icon {
  color: var(--restmate-accent);
  flex-shrink: 0;
  margin-top: 1px;
}

@media (max-width: 960px) {
  .raw-packet-split {
    flex-direction: column;
  }

  .raw-packet-aside {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .raw-packet-aside-actions {
    margin-top: 12px;
    flex-direction: row;
    flex-wrap: wrap;
  }

  .raw-packet-aside-actions :deep(.el-button) {
    flex: 1;
    min-width: 132px;
  }

  .raw-packet-card-header .raw-packet-header-meta {
    margin-left: 0;
    max-width: none;
    text-align: left;
    width: 100%;
  }
}

.workspace-card.workspace-card-surface {
  margin: 0;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
}

.workspace-card.workspace-card-surface.restmate-panel {
  border-radius: 8px;
  border-color: var(--el-border-color-lighter);
}

.workspace-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.workspace-card-title {
  font-size: 14px;
  font-weight: 600;
}

.workspace-card-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.workspace-form :deep(.el-form-item__label) {
  font-size: 12px;
}

.workspace-textarea :deep(textarea) {
  font-family: ui-monospace, 'Cascadia Code', monospace;
  font-size: 12px;
  line-height: 1.45;
}

.bypass-intro-card .bypass-steps {
  margin: 0 0 16px 1rem;
  padding: 0;
  font-size: 13px;
  line-height: 1.65;
  color: var(--el-text-color-regular);
}

.bypass-intro-card .bypass-steps li + li {
  margin-top: 8px;
}

.bypass-quick-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.header-inline-hint {
  flex: 1 1 auto;
  min-width: 0;
  margin-left: 4px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

@media (max-width: 900px) {
  .header-inline-hint {
    display: none;
  }
}

.restmate-aside {
  border-right: 1px solid var(--el-border-color-light);
  padding: 10px 8px;
  box-sizing: border-box;
  color: #374151;
}

.aside-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 4px 8px;
}

.aside-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.aside-toggle-btn {
  padding: 2px;
}

.aside-title {
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.03em;
}

.aside-scroll {
  height: calc(100% - 36px);
}

.collections-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 200px;
  color: #909399;
}

.collections-loading .is-loading {
  animation: rotating 2s linear infinite;
  margin-bottom: 12px;
}

.collections-loading .loading-text {
  font-size: 13px;
  color: #909399;
  margin: 0;
}

@keyframes rotating {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}

.collection-menu {
  border-right: none;
  background: transparent;
  color: inherit;
  --el-menu-bg-color: transparent;
}

.coll-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  color: #111827;
}

.coll-name {
  max-width: 180px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.coll-request-count {
  font-size: 11px;
  color: #909399;
  font-weight: normal;
  margin-left: 4px;
}

.collection-requests-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  color: #909399;
}

.collection-requests-wrapper {
  display: flex;
  flex-direction: column;
}

.load-more-wrapper {
  display: flex;
  justify-content: center;
  padding: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.load-more-btn {
  font-size: 12px;
}

.req-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 4px;
  padding: 4px 6px;
  border-radius: 6px;
  transition: background-color 0.16s ease, transform 0.12s ease;
}

.req-item:hover {
  background: rgba(148, 163, 184, 0.18);
  transform: translateX(1px);
}

.req-method {
  font-size: 12px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 8px;
  color: #fff;
  box-shadow: 0 0 0 1px rgba(15, 23, 42, 0.45);
  display: inline-block !important;
  visibility: visible !important;
}

.req-method.m-get {
  background: #67c23a;
}

.req-method.m-post {
  background: #409eff;
}

.req-method.m-put {
  background: #e6a23c;
}

.req-method.m-delete {
  background: #f56c6c;
}

.req-name {
  flex: 1;
  font-size: 13px;
  margin: 0 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #111827;
}

/* 顶部标签栏 + 环境变量 */
.top-tabs-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 14px 6px;
  border-bottom: 1px solid var(--el-border-color-light);
}

.tabs-left,
.tabs-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.tabs-right {
  flex: 1;
  min-width: 0;
}

.toolbar-btn {
  padding: 0 10px;
}

.restmate-header {
  padding: 8px 14px 10px;
  border-bottom: 1px solid var(--el-border-color-light);
  box-sizing: border-box;
}

.header-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mt-8 {
  margin-top: 8px;
}

.name-input {
  flex: 1.2;
  max-width: 340px;
}

.coll-select {
  width: 190px;
}

.env-select {
  width: 190px;
}

.env-select-wrapper {
  display: flex;
  align-items: center;
  gap: 6px;
}

.env-setting-btn {
  border-radius: 50%;
}

.method-select {
  width: 76px;
}

/* 为 method-select 的输入框添加颜色样式 */
.method-select :deep(.el-input__inner) {
  font-weight: 600;
  text-transform: uppercase;
}

/* GET 方法样式 */
.method-select.method-get :deep(.el-input__inner) {
  color: #67c23a;
}

/* POST 方法样式 */
.method-select.method-post :deep(.el-input__inner) {
  color: #409eff;
}

/* PUT 方法样式 */
.method-select.method-put :deep(.el-input__inner) {
  color: #e6a23c;
}

/* DELETE 方法样式 */
.method-select.method-delete :deep(.el-input__inner) {
  color: #f56c6c;
}

/* 下拉选项样式 */
.method-select :deep(.el-select-dropdown__item) {
  font-weight: 600;
  text-transform: uppercase;
}

.method-select :deep(.el-select-dropdown__item[data-value="get"]) {
  color: #67c23a;
}

.method-select :deep(.el-select-dropdown__item[data-value="post"]) {
  color: #409eff;
}

.method-select :deep(.el-select-dropdown__item[data-value="put"]) {
  color: #e6a23c;
}

.method-select :deep(.el-select-dropdown__item[data-value="delete"]) {
  color: #f56c6c;
}

.url-input {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  line-height: 20px;
  height: 32px;
  min-height: 32px;
  max-height: 32px;
  line-height: 20px;
  white-space: nowrap;      /* 不自动换行 */
  word-break: normal;       /* 避免强制断行 */
  overflow-x: auto;         /* 水平方向可滚动，支持鼠标选择时自动滚动 */
  overflow-y: hidden;       /* 垂直方向隐藏，保持单行输入 */
  user-select: text;       /* 支持鼠标选择文本 */
  scrollbar-width: none;   /* Firefox 隐藏滚动条 */
  -ms-overflow-style: none; /* IE/Edge 隐藏滚动条 */
}

/* Chrome/Safari 隐藏滚动条 */
.url-input::-webkit-scrollbar {
  display: none;
}


.send-btn {
  width: 128px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.send-btn[type='primary'] {
  background: linear-gradient(135deg, var(--restmate-accent), var(--restmate-accent-strong));
  border: none;
  box-shadow: 0 8px 18px rgba(255, 140, 0, 0.35);
}

.send-btn[type='primary']:hover {
  filter: brightness(1.06);
  box-shadow: 0 12px 26px rgba(255, 140, 0, 0.42);
}

.send-btn[type='danger'] {
  box-shadow: 0 10px 20px rgba(239, 68, 68, 0.25);
}

.send-btn[type='danger']:hover {
  filter: brightness(1.02);
  box-shadow: 0 14px 28px rgba(239, 68, 68, 0.35);
}

.save-btn-inline {
  width: 50px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #fff;
  background: linear-gradient(135deg, #ffa726, var(--restmate-accent-strong));
  border: none;
}

.save-btn-inline:hover {
  filter: brightness(1.05);
  box-shadow: 0 10px 22px rgba(255, 140, 0, 0.38);
}

.packet-btn-inline {
  width: 70px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #fff;
  background: #ff6a00;
  border: none;
}

.packet-btn-inline:hover,
.packet-btn-inline:focus,
.packet-btn-inline:active,
.packet-btn-inline:focus-visible {
  background: #ff6a00 !important;
  color: #fff !important;
  box-shadow: none;
  filter: none;
}

.restmate-main {
  padding: 10px;
  box-sizing: border-box;
}

.full-height-row,
.full-height-col {
  height: 100%;
}

:deep(.el-card){
  padding: 0;
}

.full-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 14px 32px rgba(15, 23, 42, 0.08);
}

.full-card :deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 10px 14px;
  min-height: 0;
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  font-weight: 600;
  font-size: 13px;
  color: #111827;
}

.tabs-header {
  padding: 0;
}

.request-tabs {
  width: 100%;
  flex: 1;
  min-width: 0;
}

.request-tabs :deep(.el-tabs__header) {
  margin: 0;
}

.request-tabs :deep(.el-tabs__nav-scroll) {
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
  scroll-behavior: smooth;
}

.request-tabs :deep(.el-tabs__nav-scroll::-webkit-scrollbar) {
  display: none;
}

.request-tabs :deep(.el-tabs__nav) {
  min-width: max-content;
}

.request-tabs :deep(.el-tabs__item) {
  font-size: 12px;
  padding: 0 12px;
  border-radius: 6px 6px 0 0;
  white-space: nowrap;
}

.tabs-right {
  position: relative;
}

.tabs-context-menu {
  position: fixed;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  padding: 2px 0;
  margin: 0;
  list-style: none;
  z-index: 3000;
  min-width: 85px;
}

.tabs-context-menu li {
  padding: 6px 12px;
  cursor: pointer;
  font-size: 14px;
  color: #606266;
  transition: background-color 0.2s;
}

.tabs-context-menu li:hover {
  color: var(--restmate-accent);
}

.card-header.space-between {
  justify-content: space-between;
}

.req-tabs,
.req-tabs :deep(.el-tabs__content) {
  height: 100%;
}

.req-tabs :deep(.el-tab-pane) {
  height: calc(100% - -4px);
  overflow: auto;
}

.req-tabs :deep(.el-tabs__header) {
  margin-bottom: 8px;
}

.req-tabs :deep(.el-tabs__item) {
  font-size: 12px;
  padding: 0 16px;
}

.req-tabs :deep(.el-tabs__active-bar) {
  height: 2px;
  border-radius: 999px;
}

.kv-table {
  margin-top: 4px;
}

.resp-meta {
  display: flex;
  align-items: center;
}

.ml-4 {
  margin-left: 4px;
}

.empty-tip {
  text-align: center;
  color: #909399;
  padding: 16px 0;
  font-size: 13px;
}

.resp-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.resp-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.resp-tabs :deep(.el-tabs__header) {
  flex-shrink: 0;
}

.resp-tabs :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.resp-tabs :deep(.el-tab-pane) {
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0;
}

.packet-pane {
  display: flex;
  flex-direction: column;
  gap: 8px;
  height: 100%;
}

.packet-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.packet-row {
  flex: 1;
}

.packet-col {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.packet-title {
  font-weight: 600;
  font-size: 12px;
  color: #cfd8e3;
}

.packet-col :deep(.code-editor) {
  flex: 1;
}

.resp-body {
  background: #1e1e1e;
  border-radius: 4px;
  padding: 4px;
  min-height: 200px;
  box-shadow: inset 0 0 0 1px rgba(55, 65, 81, 0.7);
}

.resp-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.cookie-toolbar {
  margin-bottom: 8px;
  display: flex;
  gap: 8px;
}

.file-cell {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
}

.file-paths {
  font-size: 12px;
  color: #606266;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-wrapper {
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
  overflow: hidden;
  height: 90%;
}

.preview-frame {
  width: 100%;
  height: 100%;
  border: none;
}

.env-manage {
  display: flex;
  flex-direction: column;
  height: 60vh;
  max-height: 600px;
}

.env-list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  flex-shrink: 0;
}

.env-list-scroll {
  flex: 1;
  overflow: hidden;
  min-height: 0;
  height: 100%;
}

.env-list-scroll :deep(.el-scrollbar__wrap) {
  overflow-x: hidden;
  overflow-y: auto;
}

.env-item-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.env-actions {
  display: flex;
  gap: 4px;
}

.env-manage :deep(.el-input__inner) {
  white-space: nowrap;
  word-break: normal;
  overflow-x: auto;
  overflow-y: hidden;
  user-select: text;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.env-manage :deep(.el-input__inner::-webkit-scrollbar) {
  display: none;
}

.env-badge {
  font-size: 11px;
  color: #67c23a;
  margin-left: 6px;
}

/* 细节：菜单与滚动条美化 */
.aside-scroll :deep(.el-menu-item),
.aside-scroll :deep(.el-sub-menu__title) {
  border-radius: 6px;
  height: 32px;
  line-height: 32px;
  padding-left: 6px;
}

/* 折叠面板内容区域样式 */
.aside-scroll :deep(.el-collapse-item__content) {
  padding-bottom: 0px;
}

.coll-req-table :deep(.el-table__body-wrapper) {
  border-radius: 6px;
}

.coll-req-table :deep(.el-table__row) {
  cursor: pointer;
}

.coll-req-table :deep(.el-table__row.is-active-req-row) {
  background-color: rgba(59, 130, 246, 0.10);
}

.coll-req-table :deep(.el-table__row:hover)>td {
  background-color: rgba(156, 163, 175, 0.15);
}

.coll-req-table :deep(.el-table__cell) {
  border-bottom: none;
  padding: 4px 4px;
}

.coll-req-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.body-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.body-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.body-json-wrapper {
  border-radius: 6px;
  overflow: hidden;
}

.body-json-editor :deep(.code-editor) {
  max-height: 700px;
}

.code-editor {
  border-radius: 4px;
  overflow: hidden;
  border: 1px solid #1f2a44;
}

.code-editor :deep(.monaco-editor),
.code-editor :deep(.monaco-editor-background) {
  background: #1e1e1e;
}

.code-editor :deep(.monaco-editor) {
  font-family: "JetBrains Mono", "Fira Code", monospace;
}

.aside-scroll :deep(.is-active > .el-sub-menu__title),
.aside-scroll :deep(.el-menu-item.is-active) {
  background: rgba(59, 130, 246, 0.2);
}

.aside-scroll :deep(.el-scrollbar__bar.is-vertical) {
  width: 6px;
}

.aside-scroll :deep(.el-scrollbar__thumb) {
  background-color: rgba(148, 163, 184, 0.8);
}

/* 批量测试全屏卡片样式 */
.batch-test-full-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.batch-test-full-card :deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

/* 批量测试滚动样式 */
.batch-test-requests-wrapper,
.batch-test-results-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.batch-test-scrollbar {
  flex: 1;
  min-height: 0;
  height: 100%;
}

.batch-test-scrollbar :deep(.el-scrollbar__wrap) {
  overflow-x: hidden;
}

.batch-test-scrollbar :deep(.el-scrollbar__view) {
  height: 100%;
}

.batch-test-results-wrapper :deep(.el-table) {
  height: 100%;
}

.batch-test-results-wrapper :deep(.el-table__row) {
  max-height: 48px;
}

.batch-test-results-wrapper :deep(.el-table__row td) {
  max-height: 48px;
  padding: 8px 0;
}

.batch-test-results-wrapper :deep(.el-table__cell) {
  max-height: 48px;
  overflow: hidden;
}

.batch-test-results-wrapper :deep(.el-table__cell .cell) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.batch-test-results-wrapper :deep(.el-table__body-wrapper) {
  max-height: calc(100% - 48px);
  overflow-y: auto;
}
</style>

<!-- 高亮样式需要作用在 v-html 动态插入的内容上，不能使用 scoped，否则无法命中 -->
<style>
/* JSON 高亮颜色（全局） */
.json-key {
  color: #93c5fd;
}

.json-string {
  color: #f97373;
}

.json-number {
  color: #facc15;
}

.json-boolean {
  color: #4ade80;
}

.json-null {
  color: #9ca3af;
}

/* HTML 高亮颜色（全局作用于 v-html 内容） */
.html-tag {
  color: #60a5fa;
}

.html-string {
  color: #f97373;
}

.html-comment {
  color: #9ca3af;
  font-style: italic;
}

/* 环境变量高亮颜色（全局作用于 v-html 内容） */
.env-var {
  color: #fbbf24 !important;
  background-color: rgba(251, 191, 36, 0.15);
  padding: 2px 4px;
  border-radius: 3px;
  font-weight: 600;
  font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;
  display: inline-block;
}
</style>