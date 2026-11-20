<template>
  <div class="poc-page">
    <div class="page-hero">
      <div>
        <div class="hero-title">POC 管理中心</div>
        <div class="hero-subtitle">支持子目录，直接编辑 / 删除本地模板 ^_^漏洞仓库推荐
          <el-tooltip
                    effect="dark"
                    placement="top"
                >
                  <template #content>
                    <div style="text-align: left; line-height: 1.5;">
                      <div>• 仓库1: https://github.com/doki-byte/NucleiPocGather</div>
                      <div>• 仓库2: https://github.com/eeeeeeeeee-code/POC</div>
                      <div>• 仓库3: https://github.com/Threekiii/Awesome-POC</div>
                    </div>
                  </template>
                  <el-icon class="help-icon"><QuestionFilled /></el-icon>
                </el-tooltip>
        </div>
      </div>
    </div>
    <CustomTabs>
      <el-tabs v-model="activeTabs" type="card">
        <el-tab-pane name="poc" label="POC管理">
          <el-card shadow="never" class="poc-card glass-card">
            <template #header>
              <div class="card-header">
                <div>
                  <div class="title">
                    POC 列表
                    <el-button type="primary" :icon="Refresh" :loading="tableLoading" size="small"
                      @click="handleRefreshList">
                      刷新
                    </el-button>
                  </div>
                  <div class="subtitle">若“查看”提示未找到，请在“漏洞扫描-扫描配置”中确认模板目录包含该文件</div>
                </div>
                <div class="flex-center" style="gap: 12px;">
                  <div class="stat">
                    <span class="stat-label">总数</span>
                    <span class="stat-value">{{ pocDisplayRows.length }}</span>
                  </div>
                  <el-button type="primary" size="small" :disabled="!multipleSelection.length"
                    @click="openMultiScanDialog">
                    使用所选生成扫描任务
                  </el-button>
                </div>
              </div>
            </template>

            <div class="flex-between mb-10px" style="padding-bottom: 8px;">
              <el-input v-model="filter" :suffix-icon="Search" placeholder="根据名称或指纹过滤" class="w-1/3" clearable
                @input="filterList">
                <template #prepend>
                  <el-select v-model="defaultFilter" style="width: 80px" @change="filterList">
                    <el-option label="名称" value="Name" />
                    <el-option label="指纹" value="Fingerprint" />
                  </el-select>
                </template>
              </el-input>
            </div>

            <el-table :data="slicedPocTableRows" height="60vh" stripe border class="elevated-table glass-card"
              v-loading="tableLoading" :header-cell-style="{ background: '#f8fafc', color: '#3b4255', fontWeight: 600 }"
              @row-contextmenu="handleRowContextMenu" @selection-change="handleSelectionChange">
              <el-table-column type="selection" width="50" align="center" fixed="left" />
              <el-table-column prop="Name" min-width="200">
                <template #header>
                  <TableColumnFilterHeader v-model="pocColumnFilters.Name" label="名称" @filter-change="handlePocColumnFilter" />
                </template>
                <template #default="{ row }">
                  <el-tooltip
                      :content="row.Name || ''"
                      placement="top"
                      :disabled="!row.Name || row.Name.length <= 25"
                      effect="dark"
                  >
                    <div class="table-cell-text">
                      {{ row.Name || '-' }}
                    </div>
                  </el-tooltip>
                </template>
              </el-table-column>
              <el-table-column min-width="260">
                <template #header>
                  <TableColumnFilterHeader v-model="pocColumnFilters.Path" label="路径" @filter-change="handlePocColumnFilter" />
                </template>
                <template #default="scope">
                  <el-tooltip :content="scope.row.Path || '未找到文件'" placement="top" :offset="4" :show-after="200"
                    :hide-after="0" effect="dark">
                    <div class="path-cell">
                      <el-text type="info" class="path-text">{{ scope.row.Path || '未找到文件' }}</el-text>
                    </div>
                  </el-tooltip>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="150" align="center" fixed="right">
                <template #default="scope">
                  <el-button size="small" :icon="Reading" @click="viewPoc(scope.row.OriginalName || scope.row.Name)"
                    title="查看"></el-button>
                  <el-button size="small" :icon="Edit" type="primary" link
                    @click="editPoc(scope.row.OriginalName || scope.row.Name, scope.row.Path)" title="编辑"></el-button>
                  <el-button type="danger" plain :icon="Delete" size="small" title="删除"
                    @click="deletePoc(scope.row.OriginalName || scope.row.Name)"></el-button>
                </template>
              </el-table-column>
              <template #empty>
                <el-empty description="未加载到 POC，请检查 POC 配置中的模板目录" />
              </template>
            </el-table>

            <Teleport to="body">
              <div v-if="contextMenu.visible" class="context-menu"
                :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }" ref="contextMenuRef">
                <div class="context-menu-item" @click="openScanDialog('jump')">
                  使用该 POC 创建扫描任务（创建后跳转并展示）
                </div>
                <div class="context-menu-item" @click="openScanDialog('background')">
                  使用该 POC 创建扫描任务（仅后台扫描）
                </div>
              </div>
            </Teleport>

            <div class="pagination-wrapper">
              <el-pagination small
                v-model:current-page="pagination.table.currentPage"
                v-model:page-size="pagination.table.pageSize"
                :page-sizes="[10, 20, 50, 100, 200]"
                :total="pocDisplayRows.length"
                layout="total, sizes, prev, pager, next, jumper"
                background
                @size-change="handleSizeChange"
                @current-change="handleCurrentChange"
              />
            </div>
          </el-card>
        </el-tab-pane>

        <el-tab-pane name="generator" label="POC生成器">
          <el-card class="glass-card generator-card">
            <div class="generator-header">
              <div>
                <div class="font-bold">Nuclei POC 生成（增强版）</div>
                <div class="generator-subtitle">左侧填写表单实时生成，右侧高亮预览，随时复制/保存</div>
              </div>
              <el-space>
                <el-button :icon="DocumentCopy" size="small" @click="copyGenerated">复制</el-button>
                <el-button :icon="VideoPlay" size="small" type="success" @click="testPocDialog = true">测试</el-button>
                <el-button :icon="DocumentAdd" size="small" type="primary" @click="saveGenerated">保存</el-button>
              </el-space>
            </div>

            <el-row :gutter="12" class="generator-grid">
              <el-col :span="11">
                <el-card shadow="never" class="basic-info-card">
                  <div class="card-title-header">
                    <div class="title-left">
                      <el-icon class="title-icon">
                        <Document />
                      </el-icon>
                      <span class="title-text">基础信息</span>
                    </div>
                    <el-tag size="small" type="success" effect="dark">实时生成</el-tag>
                  </div>
                  <div class="card-content">
                    <el-form :model="pocForm" label-width="100px" size="small">
                      <el-form-item label="模板ID">
                        <el-input v-model="pocForm.id" placeholder="唯一模板 ID" />
                      </el-form-item>
                      <el-form-item label="漏洞名称">
                        <el-input v-model="pocForm.name" placeholder="如: example-cve-2025-xxxx" />
                      </el-form-item>
                      <el-row :gutter="12">
                        <el-col :span="12">
                          <el-form-item label="作者">
                            <el-input v-model="pocForm.author" placeholder="author" />
                          </el-form-item>
                        </el-col>
                        <el-col :span="12">
                          <el-form-item label="风险等级">
                            <el-select v-model="pocForm.severity" style="width: 100%">
                              <el-option v-for="s in severityOptions" :key="s" :label="s" :value="s.toLowerCase()" />
                            </el-select>
                          </el-form-item>
                        </el-col>
                      </el-row>
                      <el-form-item label="关联指纹">
                        <el-select-v2 v-model="pocForm.tags" filterable multiple clearable :options="fingerOptions"
                          placeholder="选择或搜索指纹标签" allow-create default-first-option :reserve-keyword="false" />
                      </el-form-item>
                      <el-form-item label="描述">
                        <el-input v-model="pocForm.description" type="textarea" :rows="4" placeholder="简要描述漏洞或利用条件" />
                      </el-form-item>
                    </el-form>
                  </div>
                </el-card>

                <el-card shadow="never" class="extended-fields-card mt-12px">
                  <div class="card-title-header">
                    <div class="title-left">
                      <el-icon class="title-icon">
                        <Setting />
                      </el-icon>
                      <span class="title-text">扩展字段</span>
                    </div>
                    <el-tag size="small" type="info" effect="dark">可选</el-tag>
                  </div>
                  <div class="card-content">
                    <el-form :model="pocForm" label-width="100px" size="small">
                      <el-form-item label="metadata">
                        <div class="kv-wrapper">
                          <div class="kv-list-container">
                            <div v-for="(item, idx) in pocForm.metadata" :key="idx" class="kv-item">
                              <el-input v-model="item.key" placeholder="key" class="kv-key" />
                              <span class="kv-separator">:</span>
                              <el-input v-model="item.value" placeholder="value" class="kv-value" />
                              <el-button circle type="danger" :icon="Delete" size="small" @click="removeMeta(idx)"
                                class="kv-delete-btn" />
                            </div>
                          </div>
                          <el-button type="primary" :icon="CirclePlusFilled" text @click="addMeta" class="add-kv-btn">
                            添加 metadata
                          </el-button>
                        </div>
                      </el-form-item>
                      <el-form-item label="variables">
                        <div class="kv-wrapper">
                          <div class="kv-list-container">
                            <div v-for="(item, idx) in pocForm.variables" :key="idx" class="kv-item">
                              <el-input v-model="item.key" placeholder="key" class="kv-key" />
                              <span class="kv-separator">:</span>
                              <el-input v-model="item.value" placeholder="value" class="kv-value" />
                              <el-button circle type="danger" :icon="Delete" size="small" @click="removeVar(idx)"
                                class="kv-delete-btn" />
                            </div>
                          </div>
                          <el-button type="primary" :icon="CirclePlusFilled" text @click="addVar" class="add-kv-btn">
                            添加变量
                          </el-button>
                        </div>
                      </el-form-item>
                      <el-form-item v-if="pocForm.requests.length > 1" label="flow">
                        <el-input v-model="pocForm.flow" placeholder="flow: r0 && r1" />
                      </el-form-item>
                    </el-form>
                  </div>
                </el-card>
              </el-col>

              <el-col :span="13">
                <el-card shadow="never" class="sub-card">
                  <div class="sub-card-header">
                    <span>POC 预览</span>
                    <el-space size="small">
                      <el-tag size="small" type="warning">YAML</el-tag>
                      <el-tag size="small" type="success">高亮</el-tag>
                    </el-space>
                  </div>
                  <VueMonacoEditor :value="generatedPoc" language="yaml" theme="vs-dark" height="70vh"
                    class="code-editor" :options="previewMonacoOptions" />
                </el-card>
              </el-col>
            </el-row>

            <el-divider class="mt-14px mb-14px" />

            <div class="flex-between mb-10px">
              <div class="font-bold">请求列表</div>
              <el-button type="primary" :icon="CirclePlusFilled" @click="addRequest">添加请求</el-button>
            </div>
            <el-row :gutter="12">
              <el-col :span="24">
                <el-card v-for="(req, rIdx) in pocForm.requests" :key="rIdx" class="mb-12px request-card"
                  shadow="never">
                  <template #header>
                    <div class="flex-between request-header">
                      <div class="flex-center">
                        <el-tag size="small" type="primary" class="mr-8px request-tag">请求 #{{ rIdx + 1 }}</el-tag>
                        <span class="text-muted">配置请求并设置匹配/提取</span>
                      </div>
                      <el-button text type="danger" :icon="Delete" @click="removeRequest(rIdx)"
                        v-if="pocForm.requests.length > 1" />
                    </div>
                  </template>

                  <div class="request-content">
                    <!-- 基础配置区域 -->
                    <div class="config-section">
                      <div class="section-title">
                        <el-icon>
                          <Setting />
                        </el-icon>
                        <span>基础配置</span>
                      </div>
                      <div class="basic-config-row">
                        <el-form-item label="Raw模式" label-width="90px" size="small">
                          <el-switch v-model="req.raw" />
                        </el-form-item>
                        <el-form-item label="首次匹配停止" label-width="110px" size="small">
                          <el-switch v-model="req.stopAtFirstMatcher" />
                        </el-form-item>
                        <el-form-item label="Cookie复用" label-width="90px" size="small">
                          <el-switch v-model="req.cookieReuse" />
                        </el-form-item>
                      </div>
                    </div>

                    <el-divider />

                    <!-- 请求配置区域 -->
                    <div class="config-section">
                      <div class="section-title">
                        <el-icon>
                          <Document />
                        </el-icon>
                        <span>请求配置</span>
                      </div>
                      <div class="request-config-container">
                        <el-form label-width="90px" size="small">
                          <el-form-item v-if="req.raw" label="Raw数据包">
                            <el-input v-model="req.rawBody" type="textarea" :rows="10" placeholder="Raw HTTP 包"
                              class="raw-input" />
                          </el-form-item>
                          <template v-else>
                            <!-- 请求方法和路径 -->
                            <div class="request-method-path">
                              <el-form-item label="请求方式" class="method-item">
                                <el-select v-model="req.method" style="width: 120px">
                                  <el-option label="GET" value="GET" />
                                  <el-option label="POST" value="POST" />
                                  <el-option label="PUT" value="PUT" />
                                  <el-option label="DELETE" value="DELETE" />
                                  <el-option label="HEAD" value="HEAD" />
                                </el-select>
                              </el-form-item>
                              <el-form-item label="请求路径" class="path-item">
                                <el-input v-model="req.pathText" type="textarea" :rows="1"
                                  placeholder="/login/api/v1" />
                              </el-form-item>
                            </div>

                            <!-- 请求头和请求体 -->
                            <div class="request-headers-body">
                              <!-- 请求头区域 -->
                              <div class="request-headers-section">
                                <div class="section-subtitle">
                                  <span>请求头</span>
                                  <el-button type="primary" :icon="CirclePlusFilled" text size="small"
                                    @click="addHeader(rIdx)" class="add-header-btn-small">
                                    添加
                                  </el-button>
                                </div>
                                <div class="header-list-compact">
                                  <div v-for="(item, idx) in req.headers" :key="idx" class="header-item-compact">
                                    <el-input v-model="item.key" placeholder="Header" size="small"
                                      class="header-key-compact" />
                                    <span class="header-separator">:</span>
                                    <el-input v-model="item.value" placeholder="Value" size="small"
                                      class="header-value-compact" />
                                    <el-button circle type="danger" :icon="Delete" size="small"
                                      @click="removeHeader(rIdx, idx)" class="header-delete-btn" />
                                  </div>
                                  <div v-if="req.headers.length === 0" class="header-empty">
                                    <span class="empty-hint">暂无请求头，点击"添加"按钮添加</span>
                                  </div>
                                </div>
                              </div>

                              <!-- 请求体区域 -->
                              <div class="request-body-section">
                                <div class="section-subtitle">
                                  <span>请求体</span>
                                </div>
                                <el-input v-model="req.body" type="textarea" :rows="8" placeholder="可为空，支持JSON、XML等格式"
                                  class="body-input" />
                              </div>
                            </div>
                          </template>
                        </el-form>
                      </div>
                    </div>

                    <el-divider />

                    <!-- 匹配器区域 -->
                    <div class="config-section">
                      <div class="section-title">
                        <el-icon>
                          <Search />
                        </el-icon>
                        <span>匹配器</span>
                        <el-tag size="small" type="info" class="ml-8px">{{ req.matchers.length }}</el-tag>
                      </div>
                      <div class="matcher-list">
                        <div v-for="(m, idx) in req.matchers" :key="idx" class="matcher-item">
                          <div class="matcher-header">
                            <div class="flex-center">
                              <el-tag size="small" type="warning" class="mr-8px">#{{ idx + 1 }}</el-tag>
                              <el-select v-model="m.type" size="small" style="width: 120px">
                                <el-option label="word" value="word" />
                                <el-option label="regex" value="regex" />
                                <el-option label="status" value="status" />
                                <el-option label="size" value="size" />
                                <el-option label="dsl" value="dsl" />
                              </el-select>
                              <el-select v-if="m.type === 'word' || m.type === 'regex'" v-model="m.part" size="small"
                                style="width: 100px; margin-left: 8px">
                                <el-option label="body" value="body" />
                                <el-option label="header" value="header" />
                                <el-option label="all" value="all" />
                              </el-select>
                              <template v-if="req.matchers.length > 1 && idx < req.matchers.length - 1">
                                <el-select v-model="req.matchersCondition" size="small"
                                  style="width: 100px; margin-left: 8px">
                                  <el-option label="AND" value="and" />
                                  <el-option label="OR" value="or" />
                                </el-select>
                                <span class="matcher-condition-label">条件</span>
                              </template>
                            </div>
                            <el-button circle type="danger" :icon="Delete" size="small"
                              @click="removeMatcher(rIdx, idx)" />
                          </div>
                          <el-input v-model="m.wordsText" type="textarea" :rows="3" placeholder="每行一项"
                            class="matcher-content" />
                        </div>
                        <el-button type="primary" :icon="CirclePlusFilled" text @click="addMatcher(rIdx)"
                          class="add-btn">
                          添加匹配器
                        </el-button>
                      </div>
                    </div>

                    <el-divider />

                    <!-- 提取器区域 -->
                    <div class="config-section">
                      <div class="section-title">
                        <el-icon>
                          <Box />
                        </el-icon>
                        <span>提取器</span>
                        <el-tag size="small" type="info" class="ml-8px">{{ req.extractors.length }}</el-tag>
                      </div>
                      <div class="extractor-list">
                        <div v-for="(ex, idx) in req.extractors" :key="idx" class="extractor-item">
                          <div class="extractor-header">
                            <div class="flex-center">
                              <el-tag size="small" type="success" class="mr-8px">#{{ idx + 1 }}</el-tag>
                              <el-select v-model="ex.type" size="small" style="width: 120px">
                                <el-option label="regex" value="regex" />
                                <el-option label="xpath" value="xpath" />
                                <el-option label="json" value="json" />
                                <el-option label="dsl" value="dsl" />
                              </el-select>
                              <el-select v-model="ex.part" size="small" style="width: 100px; margin-left: 8px">
                                <el-option label="body" value="body" />
                                <el-option label="header" value="header" />
                                <el-option label="all" value="all" />
                              </el-select>
                              <el-switch v-model="ex.internal" size="small" style="margin-left: 12px" />
                              <span class="switch-label">内部提取</span>
                            </div>
                            <el-button circle type="danger" :icon="Delete" size="small"
                              @click="removeExtractor(rIdx, idx)" />
                          </div>
                          <el-input v-model="ex.name" placeholder="名称(可选)" class="extractor-name" />
                          <el-input v-model="ex.typeText" type="textarea" :rows="3" placeholder="每行一个表达式"
                            class="extractor-content" />
                        </div>
                        <el-button type="primary" :icon="CirclePlusFilled" text @click="addExtractor(rIdx)"
                          class="add-btn">
                          添加提取器
                        </el-button>
                      </div>
                    </div>
                  </div>
                </el-card>
              </el-col>
            </el-row>
          </el-card>
        </el-tab-pane>
      </el-tabs>
    </CustomTabs>

    <!-- 查看POC内容 -->
    <el-dialog v-model="detailDialog" title="POC详情" width="70%">
      <VueMonacoEditor v-model:value="pocContent" language="yaml" theme="vs-dark" height="60vh" class="code-editor"
        :options="previewMonacoOptions" />
    </el-dialog>

    <!-- 编辑POC -->
    <el-dialog v-model="editDialog" width="78%" class="editor-dialog" destroy-on-close>
      <template #header>
        <div class="editor-header">
          <div>
            <div class="editor-title">编辑 POC</div>
            <div class="editor-subtitle">
              <el-tag type="success" effect="dark" size="small">直接修改保存</el-tag>
              <span class="ml-8px" v-if="editPath">文件：{{ editPath }}</span>
            </div>
          </div>
          <el-button size="small" :icon="DocumentCopy" @click="copyEditContent">复制</el-button>
        </div>
      </template>
      <VueMonacoEditor v-model:value="editContent" language="yaml" theme="vs-dark" height="65vh" class="code-editor"
        :options="monacoOptions" />
      <template #footer>
        <el-button @click="editDialog = false">取消</el-button>
        <el-button type="primary" :loading="savingEdit" @click="saveEdit">保存</el-button>
      </template>
    </el-dialog>

    <!-- 测试POC对话框 -->
    <el-dialog v-model="testPocDialog" title="测试POC" width="80%" destroy-on-close>
      <el-form label-width="100px">
        <el-form-item label="测试URL">
          <el-input v-model="testUrl" type="textarea" :rows="6"
            placeholder="每行一个URL，例如:&#10;http://example.com&#10;https://test.com/api" clearable />
          <div class="form-tip">支持多个URL，每行一个</div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="testLoading" @click="testGeneratedPoc">开始测试</el-button>
          <el-button v-if="testResults.length > 0" @click="testResults = []">清空结果</el-button>
        </el-form-item>
      </el-form>

      <!-- 测试进度 -->
      <div v-if="testLoading && testProgress.total > 0" class="test-progress mb-10px">
        <el-progress :percentage="Math.round((testProgress.current / testProgress.total) * 100)"
          :status="testProgress.current === testProgress.total ? 'success' : null" />
        <div class="progress-text">测试进度: {{ testProgress.current }} / {{ testProgress.total }}</div>
      </div>

      <!-- 测试结果列表 -->
      <el-divider v-if="testResults.length > 0" />

      <div v-if="testResults.length > 0" class="test-results">
        <div class="test-summary mb-10px">
          <el-tag type="success">成功: {{testResults.filter(r => r.matched).length}}</el-tag>
          <el-tag type="info" class="ml-10px">失败: {{testResults.filter(r => !r.matched).length}}</el-tag>
          <el-tag type="warning" class="ml-10px">总计: {{ testResults.length }}</el-tag>
        </div>

        <el-table :data="filteredTestResults" stripe border max-height="500px" v-loading="testLoading">
          <el-table-column prop="url" min-width="200">
            <template #header>
              <TableColumnFilterHeader v-model="testColFilters.url" label="URL" />
            </template>
            <template #default="{ row }">
              <el-tooltip
                  :content="row?.url || ''"
                  placement="top"
                  :disabled="!row?.url || row.url.length <= 25"
                  effect="dark"
              >
                <div class="table-cell-text">
                  {{ row?.url || '-' }}
                </div>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column width="100" align="center">
            <template #header>
              <TableColumnFilterHeader v-model="testColFilters.status" label="状态" />
            </template>
            <template #default="{ row }">
              <el-tag :type="row?.matched ? 'success' : 'info'">
                {{ row?.matched ? '匹配' : '未匹配' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="message" min-width="150">
            <template #header>
              <TableColumnFilterHeader v-model="testColFilters.message" label="消息" />
            </template>
            <template #default="{ row }">
              <el-tooltip
                  :content="row?.message || ''"
                  placement="top"
                  :disabled="!row?.message || row.message.length <= 20"
                  effect="dark"
              >
                <div class="table-cell-text">
                  {{ row?.message || '-' }}
                </div>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column prop="responseTime" width="100" align="center">
            <template #header>
              <TableColumnFilterHeader v-model="testColFilters.responseTime" label="响应时间" />
            </template>
            <template #default="{ row }">
              <span v-if="row?.responseTime">{{ row.responseTime }}s</span>
              <span v-else class="empty-text">-</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" align="center" fixed="right">
            <template #default="{ row }">
              <el-button size="small" type="primary" link @click="showTestDetail(row)" :disabled="!row?.matched">
                详情
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <!-- 测试详情对话框 -->
    <el-dialog v-model="testDetailDialog" title="测试详情" width="70%" destroy-on-close @closed="selectedTestResult = null">
      <div v-if="selectedTestResult" class="test-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="URL">
            <el-link :href="selectedTestResult?.url || ''" target="_blank" type="primary">
              {{ selectedTestResult?.url || '' }}
            </el-link>
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="selectedTestResult?.matched ? 'success' : 'info'">
              {{ selectedTestResult?.matched ? '匹配成功' : '未匹配' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="响应时间" v-if="selectedTestResult?.responseTime">
            {{ selectedTestResult.responseTime }}s
          </el-descriptions-item>
          <el-descriptions-item label="提取结果"
            v-if="selectedTestResult?.extracted && selectedTestResult.extracted.length > 0">
            <el-tag v-for="(item, idx) in selectedTestResult.extracted" :key="idx" class="mr-5px">{{ item }}</el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <el-divider>数据包</el-divider>
        <div class="packet-inline-panels">
          <div class="packet-inline-panel">
            <div class="packet-inline-header">请求</div>
            <VueMonacoEditor v-if="monacoReady" :value="selectedTestResult?.request || ''"
              :language="detectPacketLanguage(selectedTestResult?.request || '')" theme="http-dark-theme" height="300px"
              :options="{
                readOnly: true,
                lineNumbers: 'on',
                minimap: { enabled: false },
                automaticLayout: true,
                wordWrap: 'on',
              }" />
            <pre v-else class="packet-fallback">{{ selectedTestResult?.request || '' }}</pre>
          </div>
          <div class="packet-inline-panel">
            <div class="packet-inline-header">响应</div>
            <VueMonacoEditor v-if="monacoReady" :value="selectedTestResult?.response || ''"
              :language="detectPacketLanguage(selectedTestResult?.response || '')" theme="http-dark-theme"
              height="300px" :options="{
                readOnly: true,
                lineNumbers: 'on',
                minimap: { enabled: false },
                automaticLayout: true,
                wordWrap: 'on',
              }" />
            <pre v-else class="packet-fallback">{{ selectedTestResult?.response || '' }}</pre>
          </div>
        </div>
      </div>
      <div v-else class="empty-state">
        <el-empty description="暂无测试结果" />
      </div>
    </el-dialog>

    <el-dialog v-model="scanDialogVisible" title="创建扫描任务" width="650px">
      <el-form label-width="90px">
        <el-form-item label="POC文件">
          <el-tag type="info">{{ scanForm.pocName || '-' }}</el-tag>
        </el-form-item>
        <el-form-item label="任务名称">
          <el-input v-model="scanForm.taskName" placeholder="可选，默认根据POC生成" />
        </el-form-item>
        <el-form-item label="目标地址">
          <el-input v-model="scanForm.targets" type="textarea" :rows="5" placeholder="每行一个URL或IP" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="scanDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="scanLoading" @click="createScanTask">创建任务</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="multiScanDialogVisible" title="使用所选POC创建扫描任务" width="700px">
      <el-form label-width="90px">
        <el-form-item label="已选POC">
          <div class="selected-poc-list">
            <el-tag v-for="item in multipleSelection" :key="item.OriginalName || item.Name" class="mr-5px mb-5px"
              type="info">
              {{ item.OriginalName || item.Name }}
            </el-tag>
            <div v-if="!multipleSelection.length" class="hint">当前未选择任何POC</div>
          </div>
        </el-form-item>
        <el-form-item label="任务名称">
          <el-input v-model="multiScanForm.taskName" placeholder="可选，默认自动生成" />
        </el-form-item>
        <el-form-item label="目标地址">
          <el-input v-model="multiScanForm.targets" type="textarea" :rows="5" placeholder="每行一个URL或IP" />
        </el-form-item>
        <el-form-item label="扫描模式">
          <el-radio-group v-model="multiScanMode">
            <el-radio value="jump">创建后跳转并展示</el-radio>
            <el-radio value="background">仅后台扫描</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="multiScanDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="multiScanLoading" @click="createMultiScanTask">创建任务</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { onMounted, onBeforeUnmount, ref, reactive, computed, watch, nextTick, shallowRef } from 'vue'
import { Search, CirclePlusFilled, Delete, DocumentCopy, DocumentAdd, Reading, Edit, VideoPlay, Setting, Document, Box, Refresh, QuestionFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import CustomTabs from '../../components/CustomTabs.vue'
import TableColumnFilterHeader from '@/components/TableColumnFilterHeader.vue'
import usePagination from '../../utils/usePagination'
import { VueMonacoEditor, useMonaco } from '@guolao/vue-monaco-editor'
import {
  GetFingerPocMap,
  ListPocs,
  ReadPoc,
  SavePoc,
  DeletePoc,
  TestPoc,
  StartPocScan
} from '../../../wailsjs/go/webscan/PocController'
import { GetPocConfig} from '../../../wailsjs/go/webscan/PocConfigController'
import { GetGlobalProxy } from '../../../wailsjs/go/proxy/ProxyManager'
import { registerHTTPLanguage, detectPacketLanguage } from '../../utils/http-language'

export default {
  name: 'PocManagementView',
  computed: {
    CirclePlusFilled() {
      return CirclePlusFilled
    },
    DocumentAdd() {
      return DocumentAdd
    },
    DocumentCopy() {
      return DocumentCopy
    },
    Delete() {
      return Delete
    },
    Edit() {
      return Edit
    },
    Reading() {
      return Reading
    }
  },
  components: {
    CustomTabs,
    TableColumnFilterHeader,
    VueMonacoEditor,
    Search,
    CirclePlusFilled,
    Delete,
    DocumentCopy,
    DocumentAdd,
    Reading,
    Edit,
    VideoPlay,
    Setting,
    Document,
    Box
  },
  setup() {
    const activeTabs = ref('poc')
    const filter = ref('')
    const defaultFilter = ref('Name')
    const pagination = usePagination(20)
    const pocColumnFilters = reactive({ Name: '', Path: '' })
    const pocDisplayRows = computed(() => {
      const rows = pagination.table.result || []
      const n = (pocColumnFilters.Name || '').trim().toLowerCase()
      const p = (pocColumnFilters.Path || '').trim().toLowerCase()
      if (!n && !p) return rows
      return rows.filter((row) => {
        if (n && !String(row.Name || '').toLowerCase().includes(n)) return false
        if (p && !String(row.Path || '').toLowerCase().includes(p)) return false
        return true
      })
    })
    const slicedPocTableRows = computed(() => {
      const start = (pagination.table.currentPage - 1) * pagination.table.pageSize
      const end = start + pagination.table.pageSize
      return pocDisplayRows.value.slice(start, end)
    })
    const handlePocColumnFilter = () => {
      pagination.table.currentPage = 1
    }
    let filterTimer = null
    const tableLoading = ref(false)
    const hasLoaded = ref(false) // 标记是否已经加载过数据
    const testPocDialog = ref(false) // 测试POC对话框
    const testUrl = ref('') // 测试URL（支持多行，每行一个）
    const testLoading = ref(false) // 测试加载状态
    const testResults = ref([]) // 测试结果列表
    const testColFilters = reactive({ url: '', status: '', message: '', responseTime: '' })
    const filteredTestResults = computed(() => {
      const rows = testResults.value || []
      const u = (testColFilters.url || '').trim().toLowerCase()
      const s = (testColFilters.status || '').trim().toLowerCase()
      const m = (testColFilters.message || '').trim().toLowerCase()
      const rt = (testColFilters.responseTime || '').trim().toLowerCase()
      if (!u && !s && !m && !rt) return rows
      return rows.filter((row) => {
        if (u && !String(row.url || '').toLowerCase().includes(u)) return false
        if (s) {
          const lab = row.matched ? '匹配' : '未匹配'
          if (!lab.toLowerCase().includes(s)) return false
        }
        if (m && !String(row.message || '').toLowerCase().includes(m)) return false
        if (rt) {
          const t = row.responseTime != null && row.responseTime !== '' ? `${row.responseTime}s` : '-'
          if (!String(t).toLowerCase().includes(rt)) return false
        }
        return true
      })
    })
    const testProgress = ref({ current: 0, total: 0 }) // 测试进度
    const testDetailDialog = ref(false) // 测试详情对话框
    const selectedTestResult = ref(null) // 选中的测试结果
    const monacoReady = ref(false) // Monaco编辑器就绪状态
    const detailDialog = ref(false)
    const pocContent = ref('')
    const fingerOptions = ref([])
    const pocMapCache = ref(null) // 缓存指纹映射，避免重复请求
    const loadedTagPages = new Set() // 记录已加载过指纹的分页页码
    const multipleSelection = ref([]) // 多选的POC行
    const editDialog = ref(false)
    const editContent = ref('')
    const editName = ref('')
    const editPath = ref('')
    const savingEdit = ref(false)
    const contextMenu = reactive({
      visible: false,
      x: 0,
      y: 0,
      row: null
    })
    const contextMenuRef = ref(null)
    const scanDialogVisible = ref(false)
    const scanLoading = ref(false)
    // 扫描模式：jump = 创建后跳转到漏洞扫描并展示；background = 仅后台扫描
    const scanMode = ref('background')
    const scanForm = reactive({
      pocName: '',
      pocPath: '',
      taskName: '',
      targets: ''
    })
    const multiScanDialogVisible = ref(false)
    const multiScanLoading = ref(false)
    // 多选扫描模式：jump = 创建后跳转到漏洞扫描并展示；background = 仅后台扫描
    const multiScanMode = ref('background')
    const multiScanForm = reactive({
      taskName: '',
      targets: ''
    })
    const monacoOptions = {
      minimap: { enabled: false },
      wordWrap: 'on',
      fontSize: 13,
      scrollbar: { verticalScrollbarSize: 8 },
      automaticLayout: true,
      scrollBeyondLastLine: false
    }
    const previewMonacoOptions = {
      ...monacoOptions,
      readOnly: true,
      lineNumbers: 'on'
    }

    const severityOptions = ['critical', 'high', 'medium', 'low', 'info']
    const pocForm = ref({
      id: 'nuclei-template',
      name: '',
      author: 'EasyTools',
      severity: 'high',
      tags: [],
      description: '',
      metadata: [],
      variables: [],
      flow: '',
      requests: [
        {
          raw: false,
          rawBody: '',
          method: 'GET',
          pathText: '',
          headers: [],
          body: '',
          stopAtFirstMatcher: false,
          cookieReuse: false,
          matchersCondition: 'and',
          matchers: [],
          extractors: []
        }
      ]
    })

    const indent = (level, content) => `${'  '.repeat(level)}${content}`

    const newRequest = () => ({
      raw: false,
      rawBody: '',
      method: 'GET',
      pathText: '',
      headers: [],
      body: '',
      stopAtFirstMatcher: false,
      cookieReuse: false,
      matchersCondition: 'and',
      matchers: [],
      extractors: []
    })

    const generatedPoc = computed(() => {
      const form = pocForm.value
      const lines = []

      // info
      lines.push(`id: ${form.id}`)
      lines.push('info:')
      lines.push(indent(1, `name: ${form.name || form.id}`))
      lines.push(indent(1, `author: ${form.author}`))
      lines.push(indent(1, `severity: ${form.severity}`))
      lines.push(indent(1, 'description: |'))
      lines.push(indent(2, `${form.description || 'auto generated by easytools'}`))

      // metadata
      if (form.metadata.length) {
        lines.push(indent(1, 'metadata:'))
        form.metadata.forEach(m => {
          if (m.key && m.value) lines.push(indent(2, `${m.key}: ${m.value}`))
        })
      }

      // tags
      if (form.tags.length) {
        lines.push(indent(1, `tags: ${form.tags.join(',')}`))
      }

      // variables
      if (form.variables.length) {
        lines.push('variables:')
        form.variables.forEach(v => {
          if (v.key && v.value) lines.push(indent(1, `${v.key}: ${v.value}`))
        })
      }

      if (form.flow && form.requests.length > 1) {
        lines.push('')
        lines.push(`flow: ${form.flow}`)
      }

      lines.push('')
      lines.push('http:')
      form.requests.forEach((req, rIdx) => {
        lines.push(indent(1, '-'))

        if (req.raw) {
          lines.push(indent(2, 'raw:'))
          lines.push(indent(3, '- |'))
          req.rawBody.split('\n').forEach(line => lines.push(indent(4, line)))
        } else {
          lines.push(indent(2, `method: ${req.method}`))
          lines.push(indent(2, 'path:'))
          const paths = req.pathText.split('\n').map(p => p.trim()).filter(Boolean)
          if (paths.length === 0) {
            lines.push(indent(3, '- "{{BaseURL}}"'))
          } else {
            paths.forEach(p => lines.push(indent(3, `- "{{BaseURL}}${p}"`)))
          }

          if (req.headers.length) {
            lines.push(indent(2, 'headers:'))
            req.headers.forEach(h => {
              if (h.key && h.value) lines.push(indent(3, `${h.key}: ${h.value}`))
            })
          }
          if (req.body) {
            lines.push(indent(2, 'body: |'))
            req.body.split('\n').forEach(line => lines.push(indent(3, line)))
          }
        }

        if (req.stopAtFirstMatcher) {
          lines.push(indent(2, 'stop-at-first-matcher: true'))
        }
        if (req.cookieReuse) {
          lines.push(indent(2, 'cookie-reuse: true'))
        }

        // matchers
        if (req.matchers.length) {
          if (req.matchersCondition && req.matchers.length > 1) {
            lines.push(indent(2, `matchers-condition: ${req.matchersCondition}`))
          }
          lines.push(indent(2, 'matchers:'))
          req.matchers.forEach(m => {
            lines.push(indent(3, `- type: ${m.type || 'word'}`))
            if ((m.type === 'word' || m.type === 'regex') && m.part && m.part !== 'all') {
              lines.push(indent(4, `part: ${m.part}`))
            }
            const values = (m.wordsText || '').split('\n').map(x => x.trim()).filter(Boolean)
            if (m.type === 'status' || m.type === 'size') {
              lines.push(indent(4, `${m.type === 'status' ? 'status' : 'size'}:`))
              values.forEach(v => lines.push(indent(5, `- ${v}`)))
            } else {
              const key = m.type === 'regex' ? 'regex' : m.type === 'dsl' ? 'dsl' : 'words'
              lines.push(indent(4, `${key}:`))
              values.forEach(v => lines.push(indent(5, `- '${v}'`)))
            }
          })
        }

        // extractors
        if (req.extractors.length) {
          lines.push(indent(2, 'extractors:'))
          req.extractors.forEach(ex => {
            lines.push(indent(3, `- type: ${ex.type || 'regex'}`))
            if (ex.name) lines.push(indent(4, `name: ${ex.name}`))
            if (ex.part) lines.push(indent(4, `part: ${ex.part}`))
            if (ex.internal) lines.push(indent(4, 'internal: true'))
            const values = (ex.typeText || '').split('\n').map(x => x.trim()).filter(Boolean)
            lines.push(indent(4, `${ex.type || 'regex'}:`))
            values.forEach(v => lines.push(indent(5, `- ${v}`)))
          })
        }
      })

      return lines.join('\n')
    })

    // 获取并缓存 POC->指纹 映射
    const fetchPocMap = async () => {
      if (pocMapCache.value) return pocMapCache.value
      const map = await GetFingerPocMap()
      const pocMap = map || {}
      pocMapCache.value = pocMap

      // 基于完整映射构建全局指纹选项与 POC 计数
      const fingerSet = new Set()
      Object.values(pocMap).forEach(tags => {
        ; (tags || []).forEach(t => fingerSet.add(t))
      })
      fingerOptions.value = Array.from(fingerSet).map(f => ({ label: f, value: f }))
      return pocMap
    }

    // 将指纹应用到指定行（懒加载），不对未展示的行做无谓计算
    const applyTagsForRows = (rows, pocMap) => {
      if (!rows || !rows.length || !pocMap) return
      rows.forEach(row => {
        const key = row.OriginalName || row.Name
        let tags = pocMap[key] || []
        if ((!tags || !tags.length) && row.Name) {
          // 兼容旧索引，仅使用纯文件名作为 key
          tags = pocMap[row.Name] || []
        }
        row.AssociatedFingerprint = Array.from(new Set(Array.isArray(tags) ? tags : []))
      })
    }

    // 懒加载指定页的指纹信息
    const loadTagsForPage = async (page) => {
      if (!page || loadedTagPages.has(page)) return
      const pageSize = pagination.table.pageSize
      const start = (page - 1) * pageSize
      const end = page * pageSize
      const rows = pocDisplayRows.value.slice(start, end)
      if (!rows.length) return
      try {
        const pocMap = await fetchPocMap()
        applyTagsForRows(rows, pocMap)
        loadedTagPages.add(page)
      } catch (e) {
        console.error('懒加载指纹失败:', e)
      }
    }

    const loadPocMap = async (force = false) => {
      // 防止重复加载列表，除非强制刷新
      if (!force && tableLoading.value) {
        return
      }
      try {
        tableLoading.value = true
        const list = await ListPocs()
        const result = []
          ; (list || []).forEach(item => {
            // 只显示文件名，不显示目录路径，但保存原始名称用于 API 调用及指纹映射
            const originalName = item.name || ''
            const fileName = originalName ? originalName.split(/[/\\]/).pop() : ''
            result.push({
              Name: fileName, // 列表展示用文件名
              OriginalName: originalName, // 相对路径名称（不含扩展名）
              AssociatedFingerprint: [], // 关联指纹后续懒加载
              Path: item.path
            })
          })

        // 先快速渲染完整列表数据，但只展示当前页
        pagination.table.result = result
        pagination.table.filterTemp = [...result]
        pagination.table.currentPage = 1
        pagination.ctrl.watchResultChange(pagination.table)
        hasLoaded.value = true
      } catch (e) {
        ElMessage.error(`加载 POC 列表失败: ${e.message || e}`)
      } finally {
        tableLoading.value = false
      }

      // 默认懒加载前三页的指纹信息（如果存在这些页）
      await loadTagsForPage(1)
      await loadTagsForPage(2)
      await loadTagsForPage(3)
    }

    const filterList = () => {
      // 清除之前的定时器
      if (filterTimer) {
        clearTimeout(filterTimer)
      }

      const val = filter.value.trim()

      // 如果搜索框为空，立即显示所有数据，无需防抖
      if (!val) {
        pagination.table.result = [...pagination.table.filterTemp]
        pagination.table.currentPage = 1
        pagination.ctrl.watchResultChange(pagination.table)
        return
      }

      // 对于有输入的情况，使用较短的防抖延迟（100ms），提供更实时的反馈
      tableLoading.value = true
      filterTimer = setTimeout(async () => {
        const searchVal = val.toLowerCase()
        let filtered = []

        // 使用更高效的搜索方式，对于大数据量使用for循环优化性能
        const filterTemp = pagination.table.filterTemp
        const tempLength = filterTemp.length

        if (defaultFilter.value === 'Name') {
          // 名称搜索：使用for循环比filter更快（大数据量时）
          filtered = []
          for (let i = 0; i < tempLength; i++) {
            const item = filterTemp[i]
            if (item.Name && item.Name.toLowerCase().includes(searchVal)) {
              filtered.push(item)
            }
          }
        } else {
          // 指纹搜索：在本地搜索前，确保所有行已补齐关联指纹（一次性懒加载所有）
          try {
            const pocMap = await fetchPocMap()
            applyTagsForRows(filterTemp, pocMap)
          } catch (e) {
            console.error('搜索前加载指纹失败:', e)
          }

          // 指纹搜索：使用for循环优化性能
          filtered = []
          for (let i = 0; i < tempLength; i++) {
            const item = filterTemp[i]
            const fingerprints = item.AssociatedFingerprint
            if (fingerprints && fingerprints.length > 0) {
              // 使用for循环比some更快（大数据量时）
              for (let j = 0; j < fingerprints.length; j++) {
                if (fingerprints[j].toLowerCase().includes(searchVal)) {
                  filtered.push(item)
                  break // 找到一个匹配就跳出内层循环
                }
              }
            }
          }
        }

        // 直接赋值
        pagination.table.result = filtered
        pagination.table.currentPage = 1

        // 使用nextTick确保DOM更新后再执行分页，避免卡顿
        nextTick(() => {
          pagination.ctrl.watchResultChange(pagination.table)
          tableLoading.value = false
        })
      }, 100) // 减少到100ms，提供更实时的反馈
    }

    const hideContextMenu = () => {
      contextMenu.visible = false
      contextMenu.row = null
      document.removeEventListener('click', hideContextMenu)
    }

    const adjustContextMenuPosition = async (x, y) => {
      await nextTick()
      const menuEl = contextMenuRef.value
      const menuWidth = (menuEl?.offsetWidth) || 180
      const menuHeight = (menuEl?.offsetHeight) || 50
      const maxX = window.innerWidth - menuWidth - 8
      const maxY = window.innerHeight - menuHeight - 8
      contextMenu.x = Math.min(Math.max(8, x), maxX)
      contextMenu.y = Math.min(Math.max(8, y), maxY)
    }

    const handleRowContextMenu = (row, _column, event) => {
      event.preventDefault()
      event.stopPropagation()
      contextMenu.row = row
      contextMenu.visible = true
      const x = event.pageX || (event.clientX + window.scrollX)
      const y = event.pageY || (event.clientY + window.scrollY)
      adjustContextMenuPosition(x + 2, y + 2)
      document.addEventListener('click', hideContextMenu)
    }

    const handleSelectionChange = (rows) => {
      multipleSelection.value = rows || []
    }

    const openScanDialog = (mode = 'background') => {
      if (!contextMenu.row) return
      scanMode.value = mode
      scanForm.pocName = contextMenu.row.OriginalName || contextMenu.row.Name
      scanForm.pocPath = contextMenu.row.Path || ''
      // 每次打开都根据当前选中的 POC 重新生成默认任务名称，避免沿用上次记录
      scanForm.taskName = scanForm.pocName ? `POC-${scanForm.pocName.split('/').pop()}` : ''
      scanDialogVisible.value = true
      hideContextMenu()
    }

    const openMultiScanDialog = () => {
      if (!multipleSelection.value.length) {
        ElMessage.warning('请先选择至少一个POC')
        return
      }
      const names = multipleSelection.value
        .map(item => item.OriginalName || item.Name)
        .filter(Boolean)
      // 每次打开都根据当前多选的 POC 重新生成默认任务名称，避免沿用上次记录
      multiScanForm.taskName = names.length
        ? `POC-MULTI-${names[0].split('/').pop()}`
        : 'POC-多模板任务'
      multiScanDialogVisible.value = true
    }

    const createScanTask = async () => {
      const targets = scanForm.targets
        .split('\n')
        .map(t => t.trim())
        .filter(Boolean)

      if (!scanForm.pocName) {
        ElMessage.warning('未选择POC文件')
        return
      }
      if (targets.length === 0) {
        ElMessage.warning('请输入目标地址')
        return
      }

      scanLoading.value = true
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
        console.error('获取全局代理失败:', e)
      }

      let scanConfig = { threads: 5, timeout: 10, rate_limit: 150, retries: 1 }
      try {
        const cfg = await GetPocConfig()
        if (cfg) {
          scanConfig = {
            threads: cfg.threads || 5,
            timeout: cfg.timeout || 10,
            rate_limit: cfg.rate_limit || 150,
            retries: cfg.retries || 1
          }
        }
      } catch (e) {
        console.error('获取扫描配置失败:', e)
      }

      const payload = {
        targets,
        task_name: scanForm.taskName || `POC-${scanForm.pocName}`,
        append_poc: '',
        tags: [],
        custom_tags: [],
        template_file: [scanForm.pocPath || scanForm.pocName],
        template_folders: [],
        custom_headers: '',
        proxy: proxyString,
        skip_nuclei_without_tags: false,
        thread_safe: true,
        threads: scanConfig.threads,
        timeout: scanConfig.timeout,
        rate_limit: scanConfig.rate_limit,
        retries: scanConfig.retries
      }

      try {
        const taskId = await StartPocScan(JSON.stringify(payload))
        // 根据模式决定后续行为：跳转展示 or 仅后台
        if (scanMode.value === 'jump') {
          ElMessage.success('扫描任务已创建，正在跳转到漏洞扫描界面')
          // 先切换到漏洞扫描标签页
          window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'webscan' }))
          // 等待一小段时间确保标签页切换完成，然后刷新任务列表并打开任务
          setTimeout(() => {
            // 先刷新任务列表，确保任务数据是最新的
            window.dispatchEvent(new CustomEvent('webscan-refresh-tasks'))
            // 然后打开任务详情
            setTimeout(() => {
              window.dispatchEvent(
                new CustomEvent('webscan-open-task', {
                  detail: { taskId }
                })
              )
            }, 200) // 再等待200ms确保任务列表已刷新
          }, 300) // 等待300ms确保标签页切换完成
        } else {
          ElMessage.success('扫描任务已创建，已在后台运行，可在"漏洞扫描 - 任务管理"中查看进度')
          // 通知 WebscanView 刷新任务列表（不强制切换标签）
          try {
            window.dispatchEvent(new CustomEvent('webscan-refresh-tasks'))
          } catch (e) {
            console.error('发送 Webscan 刷新事件失败:', e)
          }
        }
        scanDialogVisible.value = false
        // 重置表单字段，下次打开时重新生成默认任务名
        scanForm.targets = ''
        scanForm.taskName = ''
      } catch (e) {
        console.error('创建扫描任务失败:', e)
        ElMessage.error('创建扫描任务失败')
      } finally {
        scanLoading.value = false
      }
    }

    const createMultiScanTask = async () => {
      const targets = multiScanForm.targets
        .split('\n')
        .map(t => t.trim())
        .filter(Boolean)

      if (!multipleSelection.value.length) {
        ElMessage.warning('未选择任何POC')
        return
      }
      if (targets.length === 0) {
        ElMessage.warning('请输入目标地址')
        return
      }

      // 从多选中提取模板文件路径（优先使用物理路径，其次使用名称），并保存到本地存储供 Webscan 读取
      const templateFiles = multipleSelection.value
        .map(item => item.Path || item.OriginalName || item.Name)
        .filter(Boolean)

      if (!templateFiles.length) {
        ElMessage.warning('所选POC缺少有效路径或名称')
        return
      }

      // 写入最近一次从 POC 管理选择的模板列表，供漏洞扫描页面“指定漏洞”默认选中
      try {
        const key = 'easytools:webscan:selectedTemplatesFromPoc'
        window.localStorage.setItem(key, JSON.stringify(templateFiles))
      } catch (e) {
        console.error('写入本地模板联动缓存失败:', e)
      }

      multiScanLoading.value = true
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
        console.error('获取全局代理失败:', e)
      }

      let scanConfig = { threads: 5, timeout: 10, rate_limit: 150, retries: 1 }
      try {
        const cfg = await GetPocConfig()
        if (cfg) {
          scanConfig = {
            threads: cfg.threads || 5,
            timeout: cfg.timeout || 10,
            rate_limit: cfg.rate_limit || 150,
            retries: cfg.retries || 1
          }
        }
      } catch (e) {
        console.error('获取扫描配置失败:', e)
      }

      const payload = {
        targets,
        task_name: multiScanForm.taskName || `POC-MULTI-${templateFiles.length}`,
        append_poc: '',
        tags: [],
        custom_tags: [],
        template_file: templateFiles,
        template_folders: [],
        custom_headers: '',
        proxy: proxyString,
        skip_nuclei_without_tags: false,
        thread_safe: true,
        threads: scanConfig.threads,
        timeout: scanConfig.timeout,
        rate_limit: scanConfig.rate_limit,
        retries: scanConfig.retries
      }

      try {
        const taskId = await StartPocScan(JSON.stringify(payload))
        // 根据模式决定后续行为：跳转展示 or 仅后台
        if (multiScanMode.value === 'jump') {
          ElMessage.success('扫描任务已创建，正在跳转到漏洞扫描界面')
          // 先切换到漏洞扫描标签页
          window.dispatchEvent(new CustomEvent('cyberSabre-switch-tab', { detail: 'webscan' }))
          // 等待一小段时间确保标签页切换完成，然后刷新任务列表并打开任务
          setTimeout(() => {
            // 先刷新任务列表，确保任务数据是最新的
            window.dispatchEvent(new CustomEvent('webscan-refresh-tasks'))
            // 然后打开任务详情
            setTimeout(() => {
              window.dispatchEvent(
                new CustomEvent('webscan-open-task', {
                  detail: { taskId }
                })
              )
            }, 200) // 再等待200ms确保任务列表已刷新
          }, 300) // 等待300ms确保标签页切换完成
        } else {
          ElMessage.success('扫描任务已创建，已在后台运行，可在"漏洞扫描 - 任务管理"中查看进度')
          // 通知 WebscanView 刷新任务列表（不强制切换标签）
          try {
            window.dispatchEvent(new CustomEvent('webscan-refresh-tasks'))
          } catch (e) {
            console.error('发送 Webscan 刷新事件失败:', e)
          }
        }
        multiScanDialogVisible.value = false
        // 重置表单字段，下次打开时重新生成默认任务名
        multiScanForm.targets = ''
        multiScanForm.taskName = ''
      } catch (e) {
        console.error('创建多POC扫描任务失败:', e)
        ElMessage.error('创建多POC扫描任务失败')
      } finally {
        multiScanLoading.value = false
      }
    }

    const viewPoc = async (name) => {
      try {
        pocContent.value = await ReadPoc(name)
        detailDialog.value = true
      } catch (e) {
        ElMessage.error(`读取失败: ${e.message || e}`)
      }
    }

    const editPoc = async (name, path = '') => {
      try {
        editContent.value = await ReadPoc(name)
        editName.value = name
        editPath.value = path || ''
        editDialog.value = true
      } catch (e) {
        ElMessage.error(`读取失败: ${e.message || e}`)
      }
    }

    const saveEdit = async () => {
      if (!editName.value.trim()) {
        ElMessage.warning('名称为空')
        return
      }
      savingEdit.value = true
      try {
        const savedPath = await SavePoc(editName.value.trim(), editContent.value)
        // 异步加载列表，不阻塞UI
        loadPocMap().catch(err => {
          console.error('加载POC列表失败:', err)
        })
        ElMessage.success({
          message: savedPath ? `已保存到: ${savedPath}` : '已保存',
          duration: 4000,
          showClose: true
        })
        editDialog.value = false
      } catch (e) {
        ElMessage.error(`保存失败: ${e.message || e}`)
      } finally {
        savingEdit.value = false
      }
    }

    const copyEditContent = async () => {
      try {
        await navigator.clipboard.writeText(editContent.value)
        ElMessage.success('内容已复制')
      } catch {
        ElMessage.error('复制失败')
      }
    }

    const deletePoc = async (name) => {
      try {
        await ElMessageBox.confirm(`将删除 POC「${name}」，文件不可恢复。`, '删除 POC', { type: 'warning' })
      } catch {
        return
      }
      try {
        const fileName = name.split(/[/\\]/).pop()
        const index = pagination.table.result.findIndex(item =>
          (item.OriginalName === name || item.Name === fileName)
        )
        if (index !== -1) {
          pagination.table.result.splice(index, 1)
          pagination.table.filterTemp = pagination.table.filterTemp.filter(item =>
            !(item.OriginalName === name || item.Name === fileName)
          )
          pagination.ctrl.watchResultChange(pagination.table)
        }

        await DeletePoc(name)

        loadPocMap().catch(err => {
          console.error('重新加载POC列表失败:', err)
        })

        ElMessage.success('已删除')
      } catch (e) {
        await loadPocMap()
        ElMessage.error(`删除失败: ${e.message || e}`)
      }
    }

    const copyGenerated = async () => {
      try {
        await navigator.clipboard.writeText(generatedPoc.value)
        ElMessage.success('已复制')
      } catch {
        ElMessage.error('复制失败')
      }
    }

    const showTestDetail = (result) => {
      selectedTestResult.value = result
      testDetailDialog.value = true
    }

    const testGeneratedPoc = async () => {
      if (!pocForm.value.id.trim()) {
        ElMessage.warning('请填写模板ID')
        return
      }
      if (!pocForm.value.requests.length) {
        ElMessage.warning('至少需要一个请求')
        return
      }
      if (!testUrl.value.trim()) {
        ElMessage.warning('请输入测试URL')
        return
      }

      // 解析多个URL
      const urls = testUrl.value
        .split('\n')
        .map(url => url.trim())
        .filter(url => url && (url.startsWith('http://') || url.startsWith('https://')))

      if (urls.length === 0) {
        ElMessage.warning('请输入有效的URL（每行一个，以http://或https://开头）')
        return
      }

      testLoading.value = true
      testResults.value = []
      testProgress.value = { current: 0, total: urls.length }

      // 逐个测试URL
      for (let i = 0; i < urls.length; i++) {
        const url = urls[i]
        try {
          // 获取扫描配置
          let testConfig = { threads: 5, timeout: 10, rate_limit: 150, retries: 1 }
          try {
            const configData = await GetPocConfig()
            if (configData) {
              testConfig = {
                threads: configData.threads || 5,
                timeout: configData.timeout || 10,
                rate_limit: configData.rate_limit || 150,
                retries: configData.retries || 1
              }
            }
          } catch (e) {
            console.error('获取扫描配置失败:', e)
          }
          const result = await TestPoc(url, generatedPoc.value, JSON.stringify(testConfig))
          testResults.value.push({
            url: url,
            ...result
          })
        } catch (e) {
          testResults.value.push({
            url: url,
            matched: false,
            message: `测试失败: ${e.message || e}`,
            request: '',
            response: '',
            responseTime: '',
            extracted: []
          })
        }
        testProgress.value.current = i + 1

        // 添加小延迟，避免请求过快
        if (i < urls.length - 1) {
          await new Promise(resolve => setTimeout(resolve, 100))
        }
      }

      const successCount = testResults.value.filter(r => r.matched).length
      if (successCount > 0) {
        ElMessage.success(`测试完成：${successCount}/${urls.length} 个URL匹配成功`)
      } else {
        ElMessage.info(`测试完成：${urls.length} 个URL均未匹配`)
      }

      testLoading.value = false
    }

    const saveGenerated = async () => {
      if (!pocForm.value.id.trim()) {
        ElMessage.warning('请填写模板ID')
        return
      }
      if (!pocForm.value.requests.length) {
        ElMessage.warning('至少需要一个请求')
        return
      }
      try {
        const savedPath = await SavePoc(pocForm.value.id.trim(), generatedPoc.value)

        // 等待文件写入和标签重新加载完成（SavePoc会异步重新加载标签）
        await new Promise(resolve => setTimeout(resolve, 1000))

        ElMessage.success({
          message: '已保存，正在刷新列表',
          duration: 3000,
          showClose: true
        })

        // 强制刷新列表（不设置tableLoading，让loadPocMap内部管理）
        await loadPocMap(true)

        ElMessage.success({
          message: savedPath ? `已保存到: ${savedPath}\nPOC列表已刷新` : '已保存并刷新列表',
          duration: 3000,
          showClose: true
        })
      } catch (e) {
        ElMessage.error(`保存失败: ${e.message || e}`)
      }
    }

    // helpers
    const addRequest = () => {
      pocForm.value.requests.push(newRequest())
    }
    const removeRequest = (idx) => {
      if (pocForm.value.requests.length <= 1) {
        ElMessage.warning('至少保留一个请求')
        return
      }
      pocForm.value.requests.splice(idx, 1)
    }

    const addHeader = (rIdx) => {
      pocForm.value.requests[rIdx].headers.push({ key: '', value: '' })
    }
    const removeHeader = (rIdx, idx) => {
      pocForm.value.requests[rIdx].headers.splice(idx, 1)
    }

    const addMatcher = (rIdx) => {
      pocForm.value.requests[rIdx].matchers.push({
        type: 'word',
        part: 'body',
        wordsText: ''
      })
    }
    const removeMatcher = (rIdx, idx) => {
      pocForm.value.requests[rIdx].matchers.splice(idx, 1)
    }

    const addExtractor = (rIdx) => {
      pocForm.value.requests[rIdx].extractors.push({
        type: 'regex',
        name: '',
        part: 'body',
        internal: false,
        typeText: ''
      })
    }
    const removeExtractor = (rIdx, idx) => {
      pocForm.value.requests[rIdx].extractors.splice(idx, 1)
    }

    const addMeta = () => {
      pocForm.value.metadata.push({ key: '', value: '' })
    }
    const removeMeta = (idx) => {
      pocForm.value.metadata.splice(idx, 1)
    }
    const addVar = () => {
      pocForm.value.variables.push({ key: '', value: '' })
    }
    const removeVar = (idx) => {
      pocForm.value.variables.splice(idx, 1)
    }

    // 监听tab切换，但不自动加载（只在首次进入时加载）
    watch(activeTabs, (newVal) => {
      // 移除自动加载逻辑，只在首次进入时加载
    })

    onMounted(async () => {
      // 确保首次加载，使用nextTick确保组件完全挂载后再加载数据
      await nextTick()
      // 确保在组件挂载时加载数据
      if (!hasLoaded.value) {
        await loadPocMap()
      }

      // 初始化Monaco编辑器并注册HTTP语言
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
    })

    onBeforeUnmount(() => {
      hideContextMenu()
    })

    // 包装分页切换函数，直接更新但添加loading状态
    const handleSizeChange = (val) => {
      tableLoading.value = true
      pagination.ctrl.handleSizeChange(val)
      // 使用nextTick确保DOM更新完成后再隐藏loading
      nextTick(() => {
        tableLoading.value = false
      })
    }

    const handleCurrentChange = (val) => {
      tableLoading.value = true
      pagination.ctrl.handleCurrentChange(val)
      // 使用nextTick确保DOM更新完成后再隐藏loading
      nextTick(() => {
        tableLoading.value = false
      })
    }

    const handleRefreshList = async () => {
      try {
        await loadPocMap(true)
        ElMessage.success('列表已刷新')
      } catch (e) {
        ElMessage.error(`刷新失败: ${e.message || e}`)
      }
    }

    return {
      activeTabs,
      filter,
      defaultFilter,
      pagination,
      pocColumnFilters,
      pocDisplayRows,
      slicedPocTableRows,
      handlePocColumnFilter,
      tableLoading,
      detailDialog,
      pocContent,
      editDialog,
      editContent,
      editName,
      editPath,
      savingEdit,
      monacoOptions,
      previewMonacoOptions,
      pocForm,
      generatedPoc,
      severityOptions,
      fingerOptions,
      Search,
      CirclePlusFilled,
      Refresh,
      Delete,
      DocumentCopy,
      DocumentAdd,
      Reading,
      filterList,
      viewPoc,
      deletePoc,
      editPoc,
      saveEdit,
      copyEditContent,
      copyGenerated,
      saveGenerated,
      testPocDialog,
      testUrl,
      testLoading,
      testResults,
      testColFilters,
      filteredTestResults,
      testProgress,
      testGeneratedPoc,
      testDetailDialog,
      selectedTestResult,
      showTestDetail,
      monacoReady,
      detectPacketLanguage,
      addRequest,
      removeRequest,
      addHeader,
      removeHeader,
      addMatcher,
      removeMatcher,
      addExtractor,
      removeExtractor,
      addMeta,
      removeMeta,
      addVar,
      removeVar,
      handleSizeChange,
      handleCurrentChange,
      handleRefreshList,
      handleRowContextMenu,
      contextMenu,
      scanDialogVisible,
      scanForm,
      scanLoading,
      openScanDialog,
      createScanTask,
      contextMenuRef,
      // 多选 & 多POC扫描任务
      multipleSelection,
      handleSelectionChange,
      multiScanDialogVisible,
      multiScanForm,
      multiScanLoading,
      multiScanMode,
      openMultiScanDialog,
      createMultiScanTask
    }
  }
}
</script>

<style scoped>
.poc-page {
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 5px;
  background: radial-gradient(circle at 20% 20%, #f4f7ffcc, #eef2f73b);
  height: 100%;
}

.page-hero {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 16px;
  border-radius: 12px;
  background: linear-gradient(135deg, #6ba8ff, #8ec5ff 45%, #e8f1ff);
  color: #0a1f44;
  box-shadow: 0 10px 30px rgba(36, 99, 235, 0.16);
}

.hero-title {
  font-size: 20px;
  font-weight: 700;
}

.hero-subtitle {
  opacity: 0.8;
  margin-top: 2px;
}

.hero-actions {
  display: flex;
  gap: 10px;
}

.glass-card {
  border: 1px solid rgba(255, 255, 255, 0.45);
  box-shadow: 0 12px 30px rgba(35, 71, 138, 0.1);
  backdrop-filter: blur(4px);
  border-radius: 12px;
}

.title {
  font-weight: 600;
  font-size: 15px;
  color: #1f2d3d;
}

.subtitle {
  font-size: 12px;
  color: #909399;
}

.elevated-table :deep(.el-table__row) {
  transition: background-color 0.2s ease, transform 0.2s ease;
}

.elevated-table :deep(.el-table__row:hover) {
  background: #f7faff;
  transform: translateY(-1px);
}

.context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.12);
  border-radius: 8px;
  min-width: 180px;
  z-index: 3000;
  padding: 4px 0;
}

.context-menu-item {
  padding: 10px 14px;
  cursor: pointer;
  font-size: 13px;
  color: #303133;
}

.context-menu-item:hover {
  background: #f5f7fa;
  color: #409eff;
}

.stat {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.stat-label {
  color: #909399;
  font-size: 12px;
}

.stat-value {
  font-weight: 700;
  color: #1f2d3d;
  font-size: 16px;
}

.finger-container {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.hint {
  color: #909399;
  font-size: 12px;
}

.el-tag {
  border-radius: 10px;
}

.editor-dialog :deep(.el-dialog__header) {
  padding: 14px 20px 6px;
}

.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.editor-title {
  font-weight: 700;
  font-size: 16px;
}

.editor-subtitle {
  color: #5b6b7a;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.ml-8px {
  margin-left: 8px;
}

.code-editor {
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid #1f2a44;
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.2);
}

.code-editor :deep(.monaco-editor),
.code-editor :deep(.monaco-editor-background) {
  background: #0f172a;
}

.code-editor :deep(.monaco-editor) {
  font-family: "JetBrains Mono", "Fira Code", monospace;
}

.generator-card {
  padding: 18px;
}

.generator-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 12px;
}

.generator-subtitle {
  color: #6b7280;
  font-size: 12px;
  margin-top: 4px;
}

.generator-grid {
  margin-top: 4px;
}

.sub-card {
  border-radius: 10px;
  border: 1px solid #eef1f6;
}

.sub-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-weight: 600;
  color: #1f2d3d;
}

/* 基础信息卡片样式 */
.basic-info-card {
  border-radius: 12px;
  border: 1px solid #e4e7ed;
  background: linear-gradient(135deg, #ffffff 0%, #f8fafc 100%);
  transition: all 0.3s ease;
}

.basic-info-card:hover {
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.1);
  border-color: #c0d9ff;
}

.extended-fields-card {
  border-radius: 12px;
  border: 1px solid #e4e7ed;
  background: linear-gradient(135deg, #ffffff 0%, #f8fafc 100%);
  transition: all 0.3s ease;
}

.extended-fields-card:hover {
  box-shadow: 0 4px 12px rgba(103, 194, 58, 0.1);
  border-color: #b3e19d;
}

.mt-12px {
  margin-top: 12px;
}

.card-title-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 12px;
  margin-bottom: 16px;
  border-bottom: 2px solid #f0f2f5;
}

.title-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.title-icon {
  font-size: 18px;
  color: #409eff;
}

.title-text {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.card-content {
  padding: 4px 0;
}

.card-content :deep(.el-form-item) {
  margin-bottom: 16px;
}

.card-content :deep(.el-form-item__label) {
  font-weight: 500;
  color: #606266;
}

.card-content :deep(.el-input__wrapper) {
  border-radius: 6px;
  transition: all 0.2s ease;
}

.card-content :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #c0d9ff inset;
}

.card-content :deep(.el-textarea__inner) {
  border-radius: 6px;
  transition: all 0.2s ease;
}

.card-content :deep(.el-textarea__inner:hover) {
  border-color: #c0d9ff;
}

.card-content :deep(.el-select) {
  width: 100%;
}

.card-content :deep(.el-select-v2) {
  width: 100%;
}

.card-content :deep(.el-row) {
  margin-left: 0 !important;
  margin-right: 0 !important;
}

.card-content :deep(.el-col) {
  padding-left: 6px;
  padding-right: 6px;
}

/* 键值对列表样式 */
.kv-wrapper {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.kv-list-container {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 140px;
  /* 最多显示3个标签，每个约52px + gap 10px，3个 = 52*3 + 10*2 = 176px，留10px余量 */
  overflow-y: auto;
  overflow-x: hidden;
  padding-right: 4px;
  flex-shrink: 0;
}

.kv-list-container::-webkit-scrollbar {
  width: 6px;
}

.kv-list-container::-webkit-scrollbar-thumb {
  background: #c0c4cc;
  border-radius: 3px;
}

.kv-list-container::-webkit-scrollbar-thumb:hover {
  background: #a8abb2;
}

.kv-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
  border-radius: 8px;
  border: 1px solid #e9ecef;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.kv-item:hover {
  background: linear-gradient(135deg, #f0f2f5 0%, #f8f9fa 100%);
  border-color: #409eff;
  box-shadow: 0 2px 6px rgba(64, 158, 255, 0.1);
}

.kv-key {
  flex: 1;
  min-width: 0;
}

.kv-separator {
  color: #909399;
  font-weight: 600;
  flex-shrink: 0;
  font-size: 14px;
}

.kv-value {
  flex: 1.5;
  min-width: 0;
}

.kv-delete-btn {
  flex-shrink: 0;
}

.add-kv-btn {
  width: 100%;
  margin-top: 0;
  border: 1px dashed #d0d7de;
  border-radius: 6px;
  padding: 10px;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.add-kv-btn:hover {
  border-color: #409eff;
  background: #ecf5ff;
  color: #409eff;
}

.flex-center {
  display: flex;
  align-items: center;
}

.flex-between {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.text-muted {
  color: #8b97a8;
  font-size: 12px;
}

.mr-8px {
  margin-right: 8px;
}

.mt-10px {
  margin-top: 10px;
}

.mt-14px {
  margin-top: 14px;
}

.mb-14px {
  margin-bottom: 14px;
}

.ml-10px {
  margin-left: 10px;
}

.mb-10px {
  margin-bottom: 10px;
}

.test-progress {
  margin-top: 15px;
}

.progress-text {
  text-align: center;
  margin-top: 8px;
  color: #606266;
  font-size: 12px;
}

/* 多选 POC 对话框中，已选 POC 标签区域高度限制，超出滚动 */
.selected-poc-list {
  max-height: 180px;
  overflow-y: auto;
  padding: 4px 0;
}

.test-results {
  margin-top: 15px;
}

.test-summary {
  display: flex;
  align-items: center;
}

.test-detail {
  max-height: 70vh;
  overflow-y: auto;
}

.code-textarea {
  font-family: 'Courier New', monospace;
  font-size: 12px;
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

.empty-text {
  color: #c0c4cc;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

/* 请求配置卡片样式 */
.request-card {
  border-radius: 12px;
  border: 1px solid #e4e7ed;
  transition: all 0.3s ease;
}

.request-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.request-header {
  padding: 4px 0;
}

.request-tag {
  font-weight: 600;
}

.request-content {
  padding: 8px 0;
}

/* 配置区域样式 */
.config-section {
  margin-bottom: 20px;
}

.config-section:last-child {
  margin-bottom: 0;
}

/* 基础配置行样式 */
.basic-config-row {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}

.basic-config-row :deep(.el-form-item) {
  margin-bottom: 0;
}

.section-title {
  display: flex;
  align-items: center;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 2px solid #f0f2f5;
}

.section-title .el-icon {
  margin-right: 8px;
  color: #409eff;
  font-size: 18px;
}

/* 请求配置容器样式 */
.request-config-container {
  background: #fafbfc;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #e4e7ed;
}

/* 请求方法和路径区域 */
.request-method-path {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.method-item {
  flex-shrink: 0;
  width: 180px;
}

.path-item {
  flex: 1;
}

/* 请求头和请求体区域 */
.request-headers-body {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.request-headers-section,
.request-body-section {
  background: #fff;
  border-radius: 8px;
  padding: 12px;
  border: 1px solid #e4e7ed;
  transition: all 0.2s ease;
}

.request-headers-section:hover,
.request-body-section:hover {
  border-color: #c0c4cc;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.section-subtitle {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
  font-weight: 600;
  color: #606266;
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f2f5;
}

.add-header-btn-small {
  padding: 4px 8px;
  font-size: 12px;
}

/* 紧凑型请求头列表 */
.header-list-compact {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 140px;
  overflow-y: auto;
  padding-right: 4px;
}

.header-list-compact::-webkit-scrollbar {
  width: 6px;
}

.header-list-compact::-webkit-scrollbar-thumb {
  background: #c0c4cc;
  border-radius: 3px;
}

.header-list-compact::-webkit-scrollbar-thumb:hover {
  background: #a8abb2;
}

.header-item-compact {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
  border-radius: 6px;
  border: 1px solid #e9ecef;
  transition: all 0.2s ease;
}

.header-item-compact:hover {
  background: linear-gradient(135deg, #f0f2f5 0%, #f8f9fa 100%);
  border-color: #409eff;
  box-shadow: 0 2px 4px rgba(64, 158, 255, 0.1);
}

.header-key-compact {
  flex: 1;
  min-width: 0;
}

.header-separator {
  color: #909399;
  font-weight: 600;
  flex-shrink: 0;
}

.header-value-compact {
  flex: 1.5;
  min-width: 0;
}

.header-delete-btn {
  flex-shrink: 0;
}

.header-empty {
  padding: 20px;
  text-align: center;
  color: #c0c4cc;
}

.empty-hint {
  font-size: 12px;
}

.body-input {
  width: 100%;
}

.body-input :deep(.el-textarea__inner) {
  font-family: 'Courier New', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.raw-input {
  width: 100%;
}

.raw-input :deep(.el-textarea__inner) {
  font-family: 'Courier New', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
}

/* 请求头列表样式（保留旧样式以兼容） */
.header-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 300px;
  overflow-y: auto;
  padding-right: 4px;
}

.header-list::-webkit-scrollbar {
  width: 6px;
}

.header-list::-webkit-scrollbar-thumb {
  background: #c0c4cc;
  border-radius: 3px;
}

.header-list::-webkit-scrollbar-thumb:hover {
  background: #a8abb2;
}

.header-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e9ecef;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.header-item:hover {
  background: #f0f2f5;
  border-color: #d0d7de;
}

.header-key {
  flex: 1;
}

.header-value {
  flex: 2;
}

.add-header-btn {
  width: 100%;
  margin-top: 4px;
  border: 1px dashed #d0d7de;
  border-radius: 6px;
  padding: 8px;
}

.add-header-btn:hover {
  border-color: #409eff;
  background: #ecf5ff;
}

/* 匹配器列表样式 */
.matcher-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.matcher-item {
  padding: 14px;
  background: linear-gradient(135deg, #fff9e6 0%, #fffbf0 100%);
  border: 1px solid #ffeaa7;
  border-radius: 8px;
  transition: all 0.2s ease;
}

.matcher-item:hover {
  box-shadow: 0 2px 8px rgba(255, 193, 7, 0.15);
  border-color: #ffc107;
}

.matcher-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.matcher-content {
  margin-top: 8px;
}

.matcher-content :deep(.el-textarea__inner) {
  background: #fff;
  border-color: #ffeaa7;
}

.matcher-condition-label {
  margin-left: 4px;
  font-size: 12px;
  color: #606266;
  white-space: nowrap;
}

/* 提取器列表样式 */
.extractor-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.extractor-item {
  padding: 14px;
  background: linear-gradient(135deg, #e8f5e9 0%, #f1f8f4 100%);
  border: 1px solid #a5d6a7;
  border-radius: 8px;
  transition: all 0.2s ease;
}

.extractor-item:hover {
  box-shadow: 0 2px 8px rgba(76, 175, 80, 0.15);
  border-color: #4caf50;
}

.extractor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.switch-label {
  margin-left: 6px;
  font-size: 12px;
  color: #606266;
}

.extractor-name {
  margin-bottom: 8px;
}

.extractor-name :deep(.el-input__inner) {
  background: #fff;
  border-color: #a5d6a7;
}

.extractor-content {
  margin-top: 8px;
}

.extractor-content :deep(.el-textarea__inner) {
  background: #fff;
  border-color: #a5d6a7;
}

/* 添加按钮样式 */
.add-btn {
  width: 100%;
  margin-top: 4px;
  border: 1px dashed #d0d7de;
  border-radius: 6px;
  padding: 10px;
  font-weight: 500;
}

.add-btn:hover {
  border-color: #409eff;
  background: #ecf5ff;
  color: #409eff;
}

/* 分隔线样式优化 */
.request-content :deep(.el-divider) {
  margin: 20px 0;
  border-color: #e4e7ed;
}

.request-content :deep(.el-divider__text) {
  background: #fff;
  color: #909399;
  font-size: 12px;
}

/* 表单项样式优化 */
.request-content :deep(.el-form-item) {
  margin-bottom: 16px;
}

.request-content :deep(.el-form-item__label) {
  font-weight: 500;
  color: #606266;
}

/* 请求配置区域优化 */
.request-content :deep(.el-row) {
  margin-bottom: 0;
}

.request-content :deep(.el-col) {
  display: flex;
  flex-direction: column;
}

.request-content :deep(.el-col .el-form-item) {
  flex: 1;
  display: flex;
  flex-direction: column;
  margin-bottom: 16px;
}

.request-content :deep(.el-col .el-form-item__content) {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.request-content :deep(.el-col .el-textarea) {
  flex: 1;
}

.request-content :deep(.el-col .el-textarea .el-textarea__inner) {
  height: 100%;
  min-height: 120px;
}

/* 响应式优化 */
@media (max-width: 1200px) {
  .request-headers-body {
    grid-template-columns: 1fr;
  }

  .request-method-path {
    flex-direction: column;
  }

  .method-item {
    width: 100%;
  }

  .header-item {
    flex-direction: column;
    align-items: stretch;
  }

  .header-key,
  .header-value {
    width: 100% !important;
  }

  .basic-config-row {
    flex-direction: column;
    align-items: flex-start;
  }

  .kv-item {
    flex-wrap: wrap;
  }

  .kv-key,
  .kv-value {
    flex: 1 1 100%;
    margin-bottom: 8px;
  }

  .kv-separator {
    display: none;
  }

  .kv-delete-btn {
    margin-left: auto;
  }
}

.path-cell {
  width: 100%;
  overflow: hidden;
}

.path-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  width: 100%;
}

.table-cell-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  width: 100%;
}

/* POC 列表表：密度与爆破页对齐（保留 header-cell-style 的总体配色方向） */
.poc-card :deep(.el-table) {
  font-size: 12px;
}
.poc-card :deep(.el-table td.el-table__cell) {
  padding: 6px 0;
}
.poc-card :deep(.el-table__empty-block) {
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
