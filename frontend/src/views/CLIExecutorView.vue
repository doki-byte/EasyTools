<template>
  <div class="cli-executor-container system">
    <!-- 顶部选项卡 -->
    <el-tabs v-model="activeTab" class="main-tabs" type="border-card" closable @tab-remove="removeToolTab" @contextmenu.prevent="handleTabsContextMenu">
      <!-- 工具列表页 -->
      <el-tab-pane label="工具列表" name="tools">
        <div class="tools-page">
          <div class="page-header">
            <h1>CLI 工具执行器</h1>
            <div class="el-upload__tip">注：使用前建议查看配置说明，要不然很容易配置错误！！！
              <el-tooltip
                  effect="dark"
                  content="工具配置教程：https://www.yuque.com/doki-byte/fdzks6/rcim7hylfoh1gvg6?singleDoc#"
                  placement="top"
              >
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
            </div>
            <p>管理和执行命令行工具</p>
          </div>

          <!-- 搜索和操作栏 -->
          <div class="tools-action-bar">
            <div class="search-box">
              <el-input
                  v-model="searchKeyword"
                  placeholder="搜索工具..."
                  :prefix-icon="Search"
                  clearable
              />
            </div>
            <div class="action-buttons">
              <el-button type="primary" :icon="Plus" @click="newTool">
                新增工具
              </el-button>
              <el-button :icon="CopyDocument" @click="exportTools">
                导出配置
              </el-button>
              <el-button :icon="Folder" class="import-button">
                导入配置
                <input
                    type="file"
                    accept=".json"
                    @change="importTools"
                    class="file-input-hidden"
                />
              </el-button>
            </div>
          </div>

          <!-- 工具网格容器 -->
          <div class="tools-content">
            <!-- 工具网格 -->
            <div class="tools-grid">
              <el-card
                  v-for="tool in paginatedTools"
                  :key="tool.id"
                  class="tool-card glass-card"
                  shadow="hover"
              >
                <div class="tool-card-header">
                  <h3>{{ tool.name }}</h3>
                  <div class="tool-actions">
                    <el-tooltip content="编辑">
                      <el-button
                          size="small"
                          type="primary"
                          :icon="Edit"
                          @click="editTool(tool)"
                      />
                    </el-tooltip>
                    <el-tooltip content="复制">
                      <el-button
                          size="small"
                          type="warning"
                          :icon="CopyDocument"
                          @click="duplicateTool(tool)"
                      />
                    </el-tooltip>
                    <el-tooltip content="删除">
                      <el-button
                          size="small"
                          type="danger"
                          :icon="Delete"
                          @click="deleteTool(tool)"
                      />
                    </el-tooltip>
                  </div>
                </div>
                <p class="tool-description">{{ tool.description }}</p>
                <div class="tool-command">
                  <el-tag type="info" size="small">{{ tool.command }}</el-tag>
                </div>
                <div class="tool-parameters">
                  <el-tag
                      v-for="param in tool.parameters"
                      :key="param.id"
                      size="small"
                      class="param-tag"
                  >
                    {{ param.name }}
                  </el-tag>
                </div>
                <div class="tool-footer">
                  <el-button type="success" @click="openTool(tool)">
                    运行工具
                  </el-button>
                </div>
              </el-card>
            </div>

            <!-- 工具列表分页 -->
            <div class="tools-pagination" v-if="filteredTools.length > 0">
              <el-pagination
                  v-model:current-page="toolPagination.currentPage"
                  v-model:page-size="toolPagination.pageSize"
                  :page-sizes="[6, 12, 24, 48]"
                  :total="toolPagination.total"
                  layout="total, sizes, prev, pager, next, jumper"
                  @size-change="toolPagination.currentPage = 1"
              />
            </div>

            <!-- 空状态 -->
            <div v-if="filteredTools.length === 0" class="empty-state">
              <p>没有找到匹配的工具</p>
              <el-button type="primary" @click="newTool">
                创建第一个工具
              </el-button>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- AI 大模型：紧凑布局 + 终端式交互 + MCP 开关 -->
      <el-tab-pane label="AI 大模型" name="ai-assistant">
        <div class="cli-ai-compact">
          <div class="cli-ai-toolbar">
            <span class="cli-ai-title">AI / 终端</span>
            <el-tag size="small" :type="aiCliStatus.ok ? 'success' : 'warning'">
              {{ aiCliStatus.ok ? '模型就绪' : '未配置' }}
            </el-tag>
            <el-radio-group v-model="aiViewMode" size="small">
              <el-radio-button label="terminal">终端</el-radio-button>
              <el-radio-button label="chat">对话</el-radio-button>
            </el-radio-group>
            <div class="cli-ai-toolbar-spacer" />
            <span class="cli-ai-mcp-label">MCP HTTP</span>
            <el-switch
              v-model="mcpHttpEnabled"
              size="small"
              @change="persistCLIMCPPref"
            />
            <el-button-group class="cli-ai-btngrp">
              <el-button size="small" :loading="aiCliStatus.refreshing" @click="refreshAIDefault" :icon="Refresh">刷新</el-button>
              <el-button size="small" type="primary" :disabled="!aiCliStatus.ok" :loading="aiCliStatus.testing" @click="testAIDefault" :icon="Select">测通</el-button>
              <el-button size="small" @click="openAIConfigInSystem" :icon="Key">AI配置</el-button>
            </el-button-group>
          </div>

          <div v-if="aiCliStatus.ok" class="cli-ai-meta-line">
            {{ aiCliStatus.name }} · {{ aiProviderLabel }} · {{ aiCliStatus.model || '—' }}
            <span class="cli-ai-meta-dim"> · {{ aiCliStatus.api_key_hint ? '密钥 ' + aiCliStatus.api_key_hint : '未填密钥' }}</span>
          </div>
          <el-alert
            v-if="aiCliStatus.bannerMessage"
            class="cli-ai-alert-tight"
            :title="aiCliStatus.bannerMessage"
            :type="aiCliStatus.ok ? 'info' : 'warning'"
            :closable="false"
            show-icon
          />

          <div class="cli-ai-body">
            <div v-show="aiViewMode === 'terminal'" class="cli-ai-console" ref="aiConsoleScroll">
              <div v-for="(s, si) in aiSysLines" :key="'sys'+si" class="cli-line cli-line-sys">{{ s }}</div>
              <div
                v-for="(m, mi) in aiMessages"
                :key="'m'+mi"
                :class="['cli-line', m.role === 'user' ? 'cli-line-in' : 'cli-line-out']"
              >
                {{ m.role === 'user' ? '> ' : 'ai> ' }}{{ m.content }}
              </div>
            </div>

            <div v-show="aiViewMode === 'chat'" class="cli-ai-chat-wrap" ref="aiChatScroll">
              <div
                v-for="(m, idx) in aiMessages"
                :key="idx"
                :class="['cli-chat-bubble', m.role === 'user' ? 'is-user' : 'is-assistant']"
              >
                <span class="cli-chat-role">{{ m.role === 'user' ? '你' : '助手' }}</span>
                <div class="cli-chat-text">{{ m.content }}</div>
              </div>
            </div>

            <div class="cli-ai-prompt-row">
              <span class="cli-prompt-glyph">›</span>
              <el-input
                v-model="aiInputLine"
                class="cli-ai-input"
                size="small"
                placeholder="输入问题；或 help / clear / mcp list / mcp call 工具名 {JSON}"
                :disabled="aiSending"
                @keydown.enter.prevent="submitAIPrompt"
              />
              <el-button type="primary" size="small" :loading="aiSending" :disabled="!aiCliStatus.ok && !isAICommandLine(aiInputLine)" @click="submitAIPrompt">
                执行
              </el-button>
              <el-button size="small" @click="clearAISession" :disabled="aiSending">清空</el-button>
            </div>
          </div>

          <el-collapse class="cli-ai-collapse">
            <el-collapse-item name="mcp">
              <template #title>
                <span class="collapse-mcp-title"><el-icon><Connection /></el-icon> MCP：stdio 子进程与 JSON 示例</span>
              </template>
              <p class="cli-ai-hint">编辑器连接仍使用独立进程：<code>EasyTools mcp --stdio</code>（与上方 HTTP 开关无关）。</p>
              <div class="code-block cli-code-tight">
                <pre>{{ mcpStdioExampleText }}</pre>
              </div>
              <el-button size="small" :icon="DocumentCopy" @click="copyMcpStdioExample">复制</el-button>
              <el-button size="small" link @click="openEasyToolsDoc" :icon="Document">教程</el-button>
            </el-collapse-item>
          </el-collapse>
        </div>
      </el-tab-pane>

      <!-- 动态工具运行页 -->
      <el-tab-pane
          v-for="tab in toolTabs"
          :key="tab.id"
          :label="tab.label"
          :name="tab.id"
      >
        <div class="runner-page" :data-tab="tab.id">
          <div class="runner-header">
            <div class="runner-header-title">
              <h2>{{ tab.tool.name }}</h2>
              <p class="tool-description">{{ tab.tool.description }}</p>
            </div>
          </div>

          <!-- 参数配置和输出区域容器 -->
          <div class="runner-content">
            <!-- 参数配置 -->
            <el-card class="config-card">
              <template #header>
                <div class="config-card-header" @click="toggleConfigCollapse(tab.id)">
                  <div class="header-left">
                    <el-icon class="collapse-icon" :class="{ 'collapsed': tab.configCollapsed }">
                      <ArrowDown v-if="!tab.configCollapsed" />
                      <ArrowUp v-else />
                    </el-icon>
                    <h3>工具参数</h3>
                    <el-tag v-if="tab.tool.parameters && tab.tool.parameters.length > 0" size="small" type="info" class="param-count-tag">
                      {{ tab.tool.parameters.length }} 个参数
                    </el-tag>
                  </div>
                </div>
              </template>
              <el-collapse-transition>
                <div v-show="!tab.configCollapsed" class="config-wrapper">
                <div class="config-content">
                  <el-form label-width="100px" size="default">
                  <el-row :gutter="12">
                    <el-col :span="12" v-for="param in tab.tool.parameters" :key="param.id">
                      <el-form-item
                          :label="param.label"
                          :required="param.required"
                      >
                        <!-- 输入框 -->
                        <el-input
                            v-if="param.type === 'input'"
                            v-model="tab.parameterValues[param.id]"
                            :placeholder="param.placeholder"
                            clearable
                        />

                        <!-- 文本域 -->
                        <el-input
                            v-else-if="param.type === 'text'"
                            v-model="tab.parameterValues[param.id]"
                            :placeholder="param.placeholder"
                            type="textarea"
                            :rows="3"
                        />

                        <!-- 文件选择 -->
                        <div
                            v-else-if="param.type === 'file'"
                            class="file-input"
                        >
                          <el-input
                              v-model="tab.parameterValues[param.id]"
                              :placeholder="param.placeholder"
                              readonly
                          />
                          <el-button type="primary" :icon="Folder" @click="handleFileSelect(tab.id, param.id)">
                            选择文件
                          </el-button>
                          <input
                              type="file"
                              :ref="`fileInput-${tab.id}-${param.id}`"
                              class="file-input-hidden"
                              @change="handleFileChange($event, tab.id, param.id)"
                          />
                        </div>

                        <!-- 数字输入 -->
                        <el-input-number
                            v-else-if="param.type === 'number'"
                            v-model="tab.parameterValues[param.id]"
                            :min="1"
                        />

                        <!-- 下拉选择 -->
                        <el-select
                            v-else-if="param.type === 'select'"
                            v-model="tab.parameterValues[param.id]"
                            :placeholder="param.placeholder"
                            clearable
                        >
                          <el-option
                              v-for="option in param.options"
                              :key="option"
                              :label="option"
                              :value="option"
                          />
                        </el-select>

                        <!-- 开关 -->
                        <el-switch
                            v-else-if="param.type === 'switch'"
                            v-model="tab.parameterValues[param.id]"
                        />

                        <div v-if="param.description" class="param-description">
                          {{ param.description }}
                        </div>
                      </el-form-item>
                    </el-col>
                  </el-row>

                  <!-- 固定配置项 -->
                  <el-row :gutter="12">
                    <el-col :span="12">
                      <el-form-item label="输出编码">
                        <el-select v-model="tab.outputEncoding" placeholder="选择输出编码" style="width: 100%">
                          <el-option
                              v-for="option in encodingOptions"
                              :key="option.value"
                              :label="option.label"
                              :value="option.value"
                          />
                        </el-select>
                      </el-form-item>
                    </el-col>
                    <el-col :span="12">
                      <el-form-item label="执行模式">
                        <el-select v-model="tab.executionMode" placeholder="选择执行模式" style="width: 100%">
                          <el-option label="普通模式" value="normal" />
                          <el-option label="交互模式(调用原生终端)" value="interactive" />
                          <el-option label="Web终端模式(浏览器内交互式)" value="web" />
                        </el-select>
                        <div class="param-description">
                          交互模式适用于需要用户输入的命令（如python、mysql等）<br>
                          Web终端模式提供浏览器内终端体验，支持完整的终端交互功能
                        </div>
                      </el-form-item>
                    </el-col>
                  </el-row>

                  <el-form-item label="定时执行">
                    <div class="schedule-config">
                      <el-switch
                          v-model="tab.enableSchedule"
                          @change="clearSchedule(tab.id)"
                      />
                      <div v-if="tab.enableSchedule" class="schedule-options">
                        <el-row :gutter="10">
                          <el-col :span="8">
                            <div class="schedule-row">
                              <span class="schedule-label">首次延迟:</span>
                              <el-input-number
                                  v-model="tab.scheduleConfig.delaySeconds"
                                  :min="0"
                                  :max="3600"
                                  controls-position="right"
                                  size="small"
                                  style="width: 100px"
                              />
                              <span class="schedule-unit">秒</span>
                            </div>
                          </el-col>
                          <el-col :span="8">
                            <div class="schedule-row">
                              <span class="schedule-label">执行次数:</span>
                              <el-input-number
                                  v-model="tab.scheduleConfig.executeTimes"
                                  :min="1"
                                  :max="100"
                                  controls-position="right"
                                  size="small"
                                  style="width: 100px"
                              />
                              <span class="schedule-unit">次</span>
                            </div>
                          </el-col>
                          <el-col :span="8">
                            <div class="schedule-row">
                              <span class="schedule-label">执行间隔:</span>
                              <el-input-number
                                  v-model="tab.scheduleConfig.intervalSeconds"
                                  :min="1"
                                  :max="3600"
                                  controls-position="right"
                                  size="small"
                                  style="width: 100px"
                              />
                              <span class="schedule-unit">秒</span>
                            </div>
                          </el-col>
                        </el-row>
                      </div>
                    </div>
                  </el-form-item>
                  </el-form>
                </div>

                <!-- 操作按钮 -->
                <div class="execution-buttons">
                  <el-button
                      type="success"
                      :icon="Plus"
                      @click="startScheduledExecution(tab.id)"
                      :loading="tab.isExecuting"
                      :disabled="tab.isExecuting"
                  >
                    {{ tab.enableSchedule ? (tab.isScheduling ? '定时执行中...' : '开始定时执行') : '执行命令' }}
                  </el-button>
                  <el-button
                      type="danger"
                      @click="cancelExecution(tab.id)"
                      :disabled="!tab.isExecuting && !tab.isScheduling"
                  >
                    {{ tab.isScheduling ? '停止定时执行' : '取消执行' }}
                  </el-button>
                  <el-button @click="activeTab = 'tools'">
                    返回列表
                  </el-button>
                </div>
              </div>
              </el-collapse-transition>
            </el-card>

            <!-- 执行输出 -->
            <el-card class="output-card">
              <template #header>
                <div class="output-header">
                  <h3>执行输出</h3>
                  <div>
                    <el-button
                        v-if="tab.executionOutput"
                        type="success"
                        link
                        @click="copyOutput(tab.id)"
                    >
                      复制输出
                    </el-button>
                    <el-button
                        v-if="tab.executionOutput"
                        type="primary"
                        link
                        @click="clearOutput(tab.id)"
                    >
                      清空输出
                    </el-button>
                    <el-button
                        v-if="tab.executionOutput"
                        type="warning"
                        link
                        @click="clearRecentOutput(tab.id)"
                    >
                      仅清空本次输出
                    </el-button>
                  </div>
                </div>
              </template>
              <div class="output-content">
                <!-- Web 终端模式 -->
                <div 
                  v-if="tab.executionMode === 'web'" 
                  :data-terminal-id="tab.id"
                  class="web-terminal-container"
                ></div>
                <!-- 普通输出模式 -->
                <pre v-else class="output-pre fixed-height-output">{{ tab.executionOutput }}</pre>
              </div>
            </el-card>
          </div>
        </div>
      </el-tab-pane>

      <!-- 执行历史页 -->
            <el-tab-pane label="执行历史" name="history">
        <div class="history-page">
          <div class="page-header">
            <h2>执行历史记录</h2>
            <el-button
                type="danger"
                :icon="Delete"
                @click="clearExecutionHistory"
                :disabled="getAllExecutionHistory().length === 0"
            >
              清空历史
            </el-button>
          </div>

          <div class="history-content">
            <!-- 修改：为表格添加列宽拖拽功能 -->
            <el-table
                :data="paginatedHistory"
                class="glass-card"
                empty-text="暂无执行记录"
                border
                style="width: 100%"
                @header-dragend="handleHeaderDragEnd"
            >
              <el-table-column
                  prop="id"
                  label="ID"
                  width="120"
                  resizable
              >
                <template #default="{ row }">
                  <el-tag size="small">{{ row.id.slice(0, 8) }}</el-tag>
                </template>
              </el-table-column>

              <el-table-column
                  prop="toolName"
                  label="工具名称"
                  width="120"
                  resizable
                  show-overflow-tooltip
              />

              <el-table-column
                  prop="command"
                  label="命令"
                  width="120"
                  resizable
                  show-overflow-tooltip
              />

              <el-table-column
                  label="参数"
                  resizable
                  min-width="150"
                  show-overflow-tooltip
              >
                <template #default="{ row }">
                  <span class="parameters-display">
                    {{ formatParameters(row.parameters) }}
                  </span>
                </template>
              </el-table-column>

              <el-table-column
                  prop="status"
                  label="状态"
                  width="100"
                  resizable
              >
                <template #default="{ row }">
                  <el-tag
                      :type="row.status === 'success' ? 'success' :
                           row.status === 'error' ? 'danger' :
                           row.status === 'running' ? 'warning' : 'info'"
                  >
                    {{ getStatusText(row.status) }}
                  </el-tag>
                </template>
              </el-table-column>

              <el-table-column
                  prop="startTime"
                  label="开始时间"
                  width="180"
                  resizable
                  sortable
              >
                <template #default="{ row }">
                  {{ formatDateTime(row.startTime) }}
                </template>
              </el-table-column>

              <el-table-column
                  prop="endTime"
                  label="结束时间"
                  width="180"
                  resizable
                  sortable
              >
                <template #default="{ row }">
                  {{ row.endTime ? formatDateTime(row.endTime) : '-' }}
                </template>
              </el-table-column>

              <el-table-column
                  label="操作"
                  width="120"
                  resizable
                  fixed="right"
              >
                <template #default="{ row }">
                  <el-button
                      type="primary"
                      link
                      @click="viewExecutionHistory(row)"
                      :disabled="!row.output"
                  >
                    查看输出
                  </el-button>
                </template>
              </el-table-column>
            </el-table>

            <!-- 执行历史分页 -->
            <div class="history-pagination" v-if="getAllExecutionHistory().length > 0">
              <el-pagination
                  v-model:current-page="historyPagination.currentPage"
                  v-model:page-size="historyPagination.pageSize"
                  :page-sizes="[5, 10, 20, 50]"
                  :total="historyPagination.total"
                  layout="total, sizes, prev, pager, next, jumper"
                  @size-change="historyPagination.currentPage = 1"
              />
            </div>
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>
    <!-- 右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
    <Teleport to="body">
      <ul class="tabs-context-menu" v-if="tabsContextMenuVisible" :style="tabsContextMenuStyle">
        <li @click="handleCloseOtherTabs">关闭其他</li>
        <li @click="handleCloseAllTabs">全部关闭</li>
      </ul>
    </Teleport>

    <!-- 工具编辑对话框 -->
    <el-dialog
        v-model="showToolDialog"
        :title="editingTool && editingTool.id ? '编辑工具' : '新增工具'"
        width="900px"
        class="tool-edit-dialog"
    >
      <el-form v-if="editingTool" label-width="100px">
        <el-form-item label="工具名称" required>
          <el-input v-model="editingTool.name" placeholder="输入工具名称" />
        </el-form-item>

        <el-form-item label="命令" required>
          <el-input v-model="editingTool.command" placeholder="输入命令名称" />
        </el-form-item>

        <el-form-item label="描述">
          <el-input
              v-model="editingTool.description"
              type="textarea"
              placeholder="输入工具描述"
              :rows="2"
          />
        </el-form-item>

        <el-form-item label="工作目录">
          <el-input
              v-model="editingTool.workingDir"
              placeholder="可选，命令执行的工作目录"
          />
        </el-form-item>

        <el-form-item label="参数配置">
          <div class="parameters-list">
            <div
                v-for="(param, index) in editingTool.parameters"
                :key="param.id"
                class="parameter-item"
            >
              <div class="parameter-header">
                <el-input
                    v-model="param.name"
                    placeholder="参数名"
                    style="width: 120px;"
                />
                <el-input
                    v-model="param.label"
                    placeholder="显示标签"
                    style="width: 120px;"
                />
                <el-select v-model="param.type" placeholder="类型" style="width: 120px;">
                  <el-option label="输入框" value="input" />
                  <el-option label="文本域" value="text" />
                  <el-option label="文件" value="file" />
                  <el-option label="数字" value="number" />
                  <el-option label="下拉" value="select" />
                  <el-option label="开关" value="switch" />
                </el-select>
                <!-- 添加参数格式选择 -->
                <el-select v-model="param.argType" placeholder="参数格式" style="width: 130px;">
                  <el-option label="键值对 (-name value)" value="key-value" />
                  <el-option label="位置参数 (value)" value="positional" />
                  <el-option label="开关参数 (-flag)" value="flag" />
                  <el-option label="组合参数 (-name=value)" value="combined" />
                </el-select>
                <el-switch v-model="param.required" active-text="必填" />
                <div class="parameter-actions">
                  <el-button
                      type="primary"
                      link
                      :icon="Top"
                      @click="moveParameter(index, 'up')"
                      :disabled="index === 0"
                  />
                  <el-button
                      type="primary"
                      link
                      :icon="Bottom"
                      @click="moveParameter(index, 'down')"
                      :disabled="index === editingTool.parameters.length - 1"
                  />
                  <el-button
                      type="danger"
                      :icon="Delete"
                      @click="removeParameter(index)"
                  />
                </div>
              </div>
              <div class="parameter-options">
                <el-input
                    v-model="param.placeholder"
                    placeholder="命令参数占位符"
                    style="width: 200px;"
                />
                <el-input
                    v-model="param.description"
                    placeholder="参数描述"
                    style="flex: 1;"
                />
              </div>
              <div v-if="param.type === 'select'" class="select-options">
                <el-tag
                    v-for="(option, optIndex) in param.options || []"
                    :key="optIndex"
                    closable
                    @close="removeOption(param, optIndex)"
                >
                  {{ option }}
                </el-tag>
                <el-input
                    v-model="newOptionValues[param.id]"
                    placeholder="添加选项"
                    style="width: 120px;"
                    @keyup.enter="addOption(param)"
                />
                <el-button @click="addOption(param)">添加</el-button>
              </div>
              <div v-if="param.type === 'number'" class="default-value">
                <el-input-number
                    v-model="param.defaultValue"
                    placeholder="默认值"
                    :min="0"
                />
              </div>
              <div v-else-if="param.type === 'switch'" class="default-value">
                <el-switch v-model="param.defaultValue" />
              </div>
              <div v-else class="default-value">
                <el-input
                    v-model="param.defaultValue"
                    placeholder="默认值"
                />
              </div>
            </div>
            <el-button type="primary" @click="addParameter" :icon="Plus">
              添加参数
            </el-button>
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="showToolDialog = false">取消</el-button>
        <el-button type="primary" @click="saveTool">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ElMessage, ElMessageBox } from 'element-plus'
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm'
import {
  Plus, Delete, Edit, Search, Folder, CopyDocument, Top, Bottom, QuestionFilled,
  Connection, Key, Setting, Document, Tools, Refresh, Select, InfoFilled, Warning,
  DocumentCopy, ArrowDown, ArrowUp
} from '@element-plus/icons-vue'
import { GetCLIMCPEnabled, SetCLIMCPEnabled } from "../../wailsjs/go/controller/Server"
import { discoverLocalPortConfig } from '@/utils/localServicePorts'
import { OnFileDrop } from "../../wailsjs/runtime"
import { EventsOn } from "../../wailsjs/runtime"
import { BrowserOpenURL } from '../../wailsjs/runtime';
import axios from 'axios'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

export default {
  name: 'CLIExecutorView',
  components: {QuestionFilled},

  data() {
    return {
      // 响应式数据
      wailsDropPaths: [],
      activeTab: 'tools',
      toolTabs: [], // 动态工具运行页
      tools: [],
      showToolDialog: false,
      editingTool: null,
      searchKeyword: '',
      executionHistory: [], // 全局执行历史记录
      portConfig: null,
      newOptionValues: {},
      aiCliStatus: {
        ok: false,
        bannerMessage: '',
        name: '',
        provider: '',
        model: '',
        api_base: '',
        api_key_hint: '',
        has_api_key: false,
        refreshing: false,
        testing: false
      },
      aiViewMode: 'terminal',
      aiInputLine: '',
      aiMessages: [],
      aiSending: false,
      aiSysLines: [],
      mcpHttpEnabled: true,

      // 工具列表分页
      toolPagination: {
        currentPage: 1,
        pageSize: 6,
        total: 0
      },

      // 执行历史分页
      historyPagination: {
        currentPage: 1,
        pageSize: 10,
        total: 0
      },

      // 标签页右键菜单
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: '',

      // 新增：保存列宽设置
      columnWidths: {
        id: 120,
        toolName: 120,
        command: 120,
        parameters: 150,
        status: 100,
        startTime: 180,
        endTime: 180,
        actions: 120
      },

      // Web 终端实例存储
      terminalInstances: {},
      terminalWebSockets: {}

    }
  },

  computed: {
    Folder() {
      return Folder
    },
    Plus() {
      return Plus
    },
    Search() {
      return Search
    },
    Delete() {
      return Delete
    },
    CopyDocument() {
      return CopyDocument
    },
    Edit() {
      return Edit
    },
    Top() {
      return Top
    },
    Bottom() {
      return Bottom
    },
    Connection() {
      return Connection
    },
    Key() {
      return Key
    },
    Setting() {
      return Setting
    },
    Document() {
      return Document
    },
    Tools() {
      return Tools
    },
    Refresh() {
      return Refresh
    },
    Select() {
      return Select
    },
    InfoFilled() {
      return InfoFilled
    },
    Warning() {
      return Warning
    },
    DocumentCopy() {
      return DocumentCopy
    },
    // API 基础URL计算属性
    API_BASE() {
      if (!this.portConfig) return ''
      return `http://127.0.0.1:${this.portConfig.fuzzPort}/cli`
    },

    // 过滤工具列表
    filteredTools() {
      if (!this.searchKeyword) return this.tools
      const keyword = this.searchKeyword.toLowerCase()
      return this.tools.filter(tool =>
          tool.name.toLowerCase().includes(keyword) ||
          tool.description.toLowerCase().includes(keyword) ||
          tool.command.toLowerCase().includes(keyword)
      )
    },

    encodingOptions() {
      return [
        { label: 'UTF-8', value: 'utf-8' },
        { label: 'GBK', value: 'gbk' },
        { label: 'GB2312', value: 'gb2312' },
        { label: 'ISO-8859-1', value: 'iso-8859-1' }
      ]
    },

    // 分页后的工具列表
    paginatedTools() {
      const filtered = this.filteredTools
      this.toolPagination.total = filtered.length
      const start = (this.toolPagination.currentPage - 1) * this.toolPagination.pageSize
      const end = start + this.toolPagination.pageSize
      return filtered.slice(start, end)
    },

    // 分页后的执行历史
    paginatedHistory() {
      const allHistory = this.getAllExecutionHistory()
      this.historyPagination.total = allHistory.length
      const start = (this.historyPagination.currentPage - 1) * this.historyPagination.pageSize
      const end = start + this.historyPagination.pageSize
      return allHistory.slice(start, end)
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
    },

    aiProviderLabel() {
      const p = this.aiCliStatus.provider
      const map = {
        openai: 'OpenAI',
        deepseek: 'DeepSeek',
        qianwen: '通义千问',
        custom: '自定义'
      }
      return map[p] || p || '—'
    },

    mcpStdioExampleText() {
      return [
        '{',
        '  "type": "stdio",',
        '  "command": "<EasyTools可执行文件完整路径>",',
        '  "args": ["mcp", "--stdio"],',
        '  "env": {',
        '    "MCP_AUTO_REGISTER": "true"',
        '  }',
        '}'
      ].join('\n')
    }
  },

  async mounted() {
    await this.loadPortConfig()
    await this.loadTools()
    this.loadExecutionHistory()
    this.loadColumnWidths()
    this.refreshAIDefault().catch(() => {})
    this.loadCLIMCPPref().catch(() => {})

    // 捕获 Wails 级别的拖拽，优先使用绝对路径
    OnFileDrop((x, y, paths) => {
      if (paths && paths.length && this.isViewActive()) {
        this.wailsDropPaths = paths
        this.applyDroppedParamPath(paths[0])
      }
    })

    // 后端广播的全局拖拽事件（确保绝对路径）
    EventsOn('global-file-drop', (paths) => {
      if (paths && paths.length && this.isViewActive()) {
        this.wailsDropPaths = paths
        this.applyDroppedParamPath(paths[0])
      }
    })
  },

  methods: {
    // 视图是否可见（限制跨页面填充）
    isViewActive() {
      return this.$el && document.body.contains(this.$el) && this.$el.offsetParent !== null
    },

    applyDroppedParamPath(filePath) {
      if (!filePath) return
      if (!this.isViewActive()) return

      // 使用当前激活的 tab 的第一个文件/输入参数
      const activeTabObj = this.getTab(this.activeTab)
      if (activeTabObj && activeTabObj.tool && activeTabObj.tool.parameters?.length) {
        const targetParam = activeTabObj.tool.parameters.find(p =>
            ['file', 'input', 'text'].includes(p.type))
        if (targetParam) {
          activeTabObj.parameterValues[targetParam.id] = filePath
          ElMessage.success('已通过拖拽填充路径')
        }
      }
    },

    // 获取所有工具的执行历史
    getAllExecutionHistory() {
      // 按时间倒序排序
      return this.executionHistory.sort((a, b) => new Date(b.startTime) - new Date(a.startTime))
    },

    // 根据工具ID获取执行历史
    getExecutionHistoryByToolId(toolId) {
      return this.executionHistory
          .filter(history => history.toolId === toolId)
          .sort((a, b) => new Date(b.startTime) - new Date(a.startTime))
    },

    // 创建axios实例
    createAxiosInstance() {
      return axios.create({
        baseURL: this.API_BASE,
        timeout: 30000,
        headers: {
          'Content-Type': 'application/json',
        }
      })
    },


    // 加载端口配置
    async loadPortConfig() {
      try {
        // 依次尝试 Wails 绑定 → 本地缓存 → 本机主程序 /api/port-config 探测，
        // 保证「网页预览 + 桌面主程序在运行」时也能命中真实端口（端口是动态分配的）。
        const config = await discoverLocalPortConfig()
        this.portConfig = config || null
        if (config) {
          console.log('端口配置:', this.portConfig)
        } else {
          console.warn('未检测到本地 EasyTools 服务（Wails 绑定不可用且未探测到主程序）')
        }
      } catch (error) {
        console.error('获取端口配置失败:', error)
        this.portConfig = null
      }
    },

    // 无后端服务时仅使用本地存储的工具配置
    useLocalToolsFallback() {
      const savedTools = localStorage.getItem('cli-tools')
      if (savedTools) {
        this.tools = JSON.parse(savedTools)
        this.removeLegacyIflowTool()
      } else {
        this.tools = []
      }
    },

    // 加载工具配置
    async loadTools() {
      if (!this.API_BASE) {
        // 后端不可用时使用本地存储
        this.useLocalToolsFallback()
        return
      }
      try {
        const axiosInstance = this.createAxiosInstance()
        const response = await axiosInstance.get('/tools')
        this.tools = response.data
        this.removeLegacyIflowTool()
      } catch (error) {
        console.error('Failed to load tools from backend:', error)
        // 后端不可用时使用本地存储
        this.useLocalToolsFallback()
      }
    },
    
    // 移除历史遗留的 CLI 预设工具 ID，避免误执行已删除的命令
    removeLegacyIflowTool() {
      const idx = this.tools.findIndex(t => t.id === 'tool-iflow-cli')
      if (idx !== -1) {
        this.tools.splice(idx, 1)
        this.saveTools()
      }
    },

    // 保存工具配置
    async saveTools() {
      // 先保存到本地存储
      localStorage.setItem('cli-tools', JSON.stringify(this.tools))

      if (!this.API_BASE) {
        ElMessage.warning('未检测到本地 EasyTools 服务，配置仅保存在本地存储（执行命令需桌面主程序后端）')
        return
      }

      try {
        const axiosInstance = this.createAxiosInstance()
        await axiosInstance.post('/tools', this.tools)
        // ElMessage.success('工具配置保存成功')
      } catch (error) {
        console.error('Failed to save tools to backend:', error)
        ElMessage.warning('同步到本地服务失败，配置已保存到本地存储')
      }
    },

    // 加载执行历史
    loadExecutionHistory() {
      const savedHistory = localStorage.getItem('cli-execution-history')
      if (savedHistory) {
        this.executionHistory = JSON.parse(savedHistory)
      }
    },

    // 保存执行历史
    saveExecutionHistory() {
      localStorage.setItem('cli-execution-history', JSON.stringify(this.executionHistory))
    },

    // 添加执行历史记录
    addExecutionHistory(history) {
      this.executionHistory.unshift(history)
      // 限制历史记录数量，避免存储过大
      if (this.executionHistory.length > 1000) {
        this.executionHistory = this.executionHistory.slice(0, 1000)
      }
      this.saveExecutionHistory()
    },

    // 更新执行历史记录
    updateExecutionHistory(executionId, updates) {
      const index = this.executionHistory.findIndex(h => h.id === executionId)
      if (index !== -1) {
        this.executionHistory[index] = { ...this.executionHistory[index], ...updates }
        this.saveExecutionHistory()
      }
    },

    // 切换参数配置折叠状态
    toggleConfigCollapse(tabId) {
      const tab = this.toolTabs.find(t => t.id === tabId)
      if (tab) {
        tab.configCollapsed = !tab.configCollapsed
      }
    },

    // 打开工具运行页 - 修改为支持同一工具打开多个窗口
    openTool(tool) {
      // 计算当前工具已打开的窗口数量，用于生成标签页名称
      const sameToolCount = this.toolTabs.filter(tab => tab.tool.id === tool.id).length
      const tabLabel = sameToolCount > 0 ? `${tool.name} (${sameToolCount + 1})` : tool.name

      // 创建新的工具标签页
      const tabId = `tool-${tool.id}-${Date.now()}`
      const newTab = {
        id: tabId,
        tool: { ...tool },
        label: tabLabel,
        parameterValues: {},
        outputEncoding: 'utf-8',
        executionMode: 'normal',
        enableSchedule: false,
        scheduleConfig: {
          delaySeconds: 5,
          executeTimes: 1,
          intervalSeconds: 10
        },
        isExecuting: false,
        isScheduling: false,
        currentScheduleCount: 0,
        scheduleTimer: null,
        currentExecutionId: '',
        executionOutput: '',
        eventSource: null,
        configCollapsed: false // 参数配置折叠状态，默认展开
      }

      // 初始化参数值
      if (tool.parameters && Array.isArray(tool.parameters)) {
        tool.parameters.forEach(param => {
          if (param && param.id) {
            if (param.defaultValue !== undefined) {
              newTab.parameterValues[param.id] = param.defaultValue
            } else {
              newTab.parameterValues[param.id] = ''
            }
          }
        })
      }

      this.toolTabs.push(newTab)
      this.activeTab = tabId
    },

    // 删除工具标签页 - 修复：不删除执行历史记录
    removeToolTab(tabName) {
      if (tabName === 'tools' || tabName === 'history') {
        return
      }

      const tabIndex = this.toolTabs.findIndex(tab => tab.id === tabName)
      if (tabIndex !== -1) {
        const tab = this.toolTabs[tabIndex]

        // 清理资源
        if (tab.eventSource) {
          tab.eventSource.close()
        }
        if (tab.scheduleTimer) {
          clearTimeout(tab.scheduleTimer)
        }
        if (tab.isExecuting) {
          this.cancelExecution(tab.id)
        }

        this.toolTabs.splice(tabIndex, 1)

        // 如果删除的是当前激活的标签页，切换到工具列表
        if (this.activeTab === tabName) {
          this.activeTab = 'tools'
        }
      }
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
          : this.activeTab

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

    // 关闭其他标签页
    handleCloseOtherTabs() {
      this.hideTabsContextMenu()

      const targetId = this.tabsContextMenuTabId || this.activeTab

      // 如果点击的是固定页（工具列表或历史），则仅关闭所有工具标签页
      if (targetId === 'tools' || targetId === 'history') {
        this.toolTabs.forEach(tab => {
          if (tab.eventSource) {
            tab.eventSource.close()
          }
          if (tab.scheduleTimer) {
            clearTimeout(tab.scheduleTimer)
          }
          if (tab.isExecuting) {
            this.cancelExecution(tab.id)
          }
        })
        this.toolTabs = []
        this.activeTab = targetId
        return
      }

      const targetTab = this.toolTabs.find(tab => tab.id === targetId)
      if (!targetTab) {
        return
      }

      // 关闭其他工具标签页
      this.toolTabs.forEach(tab => {
        if (tab.id !== targetId) {
          if (tab.eventSource) {
            tab.eventSource.close()
          }
          if (tab.scheduleTimer) {
            clearTimeout(tab.scheduleTimer)
          }
          if (tab.isExecuting) {
            this.cancelExecution(tab.id)
          }
        }
      })

      this.toolTabs = [targetTab]
      this.activeTab = targetId
    },

    hideTabsContextMenu() {
      this.tabsContextMenuVisible = false
    },

    // 关闭所有标签页
    handleCloseAllTabs() {
      this.hideTabsContextMenu()
      
      // 关闭所有工具标签页（保留工具列表和历史记录标签页）
      this.toolTabs.forEach(tab => {
        // 清理资源
        if (tab.eventSource) {
          tab.eventSource.close()
        }
        if (tab.scheduleTimer) {
          clearTimeout(tab.scheduleTimer)
        }
        if (tab.isExecuting) {
          this.cancelExecution(tab.id)
        }
      })
      
      this.toolTabs = []
      
      // 切换到工具列表
      this.activeTab = 'tools'
    },

    // 获取标签页
    getTab(tabId) {
      return this.toolTabs.find(tab => tab.id === tabId)
    },

    // 新增工具
    newTool() {
      this.editingTool = {
        id: 'tool-' + Date.now(),
        name: '',
        command: '',
        description: '',
        parameters: [],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString()
      }
      this.showToolDialog = true
    },

    // 编辑工具
    editTool(tool) {
      this.editingTool = JSON.parse(JSON.stringify(tool))
      this.showToolDialog = true
    },

    // 复制工具
    duplicateTool(tool) {
      const duplicated = JSON.parse(JSON.stringify(tool))
      duplicated.id = 'tool-' + Date.now()
      duplicated.name = `${tool.name} - 副本`
      duplicated.createdAt = new Date().toISOString()
      duplicated.updatedAt = new Date().toISOString()
      this.tools.push(duplicated)
      this.saveTools()
      ElMessage.success('工具复制成功')
    },

    // 删除工具
    async deleteTool(tool) {
      try {
        await confirmDangerousAction({
          title: '删除确认',
          message: `确定要删除工具「${tool.name}」吗？此操作不可恢复。`,
          expectedInput: DANGER_DELETE_TOKEN
        })
        const index = this.tools.findIndex(t => t.id === tool.id)
        if (index !== -1) {
          this.tools.splice(index, 1)
          await this.saveTools()
          ElMessage.success('工具删除成功')
        }
      } catch {
        // 用户取消删除
      }
    },

    // 保存工具
    saveTool() {
      if (!this.editingTool) return

      if (!this.editingTool.name || !this.editingTool.command) {
        ElMessage.error('请填写工具名称和命令')
        return
      }

      // 验证参数名称唯一性
      const paramNames = new Set()
      for (const param of this.editingTool.parameters) {
        // if (paramNames.has(param.name)) {
        //   ElMessage.error(`参数名称 "${param.name}" 重复`)
        //   return
        // }
        paramNames.add(param.name)
      }

      const index = this.tools.findIndex(t => t.id === this.editingTool.id)
      if (index !== -1) {
        this.editingTool.updatedAt = new Date().toISOString()
        this.tools[index] = this.editingTool
      } else {
        this.tools.push(this.editingTool)
      }

      this.saveTools()
      this.showToolDialog = false
      ElMessage.success('工具保存成功')
    },

    // 添加参数
    addParameter() {
      if (!this.editingTool) return
      const newParam = {
        id: 'param-' + Date.now() + '-' + Math.random().toString(36).substr(2, 5),
        name: '',
        label: '',
        type: 'input',
        required: false,
        argType: 'key-value'
      }
      this.editingTool.parameters.push(newParam)
    },

    // 删除参数
    removeParameter(index) {
      if (!this.editingTool) return
      this.editingTool.parameters.splice(index, 1)
    },

    // 移动参数位置
    moveParameter(index, direction) {
      if (!this.editingTool) return
      const newIndex = direction === 'up' ? index - 1 : index + 1
      if (newIndex >= 0 && newIndex < this.editingTool.parameters.length) {
        const temp = this.editingTool.parameters[newIndex]
        this.editingTool.parameters[newIndex] = this.editingTool.parameters[index]
        this.editingTool.parameters[index] = temp
      }
    },

    // 添加选项
    addOption(param) {
      if (!this.editingTool) return

      const optionValue = this.newOptionValues[param.id]
      if (optionValue && optionValue.trim()) {
        if (!param.options) {
          param.options = []
        }
        if (!param.options.includes(optionValue)) {
          param.options.push(optionValue)
          this.newOptionValues[param.id] = ''
        } else {
          ElMessage.warning('选项已存在')
        }
      }
    },

    // 删除选项
    removeOption(param, index) {
      param.options.splice(index, 1)
    },

    // 文件选择
    handleFileSelect(tabId, paramId) {
      const fileInput = this.$refs[`fileInput-${tabId}-${paramId}`]
      if (fileInput && fileInput[0]) {
        fileInput[0].click()
      }
    },


    // 文件变更
    handleFileChange(event, tabId, paramId) {
      const input = event.target
      const tab = this.getTab(tabId)
      if (input.files && input.files[0] && tab) {
        const file = input.files[0]
        tab.parameterValues[paramId] = file.path || file.name
      }
    },

    // 在 executeTool 方法中添加交互式命令的特殊处理
    async executeTool(tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      // 检查是否为交互式命令
      const interactiveCommands = ['python', 'mysql', 'sqlite3', 'node', 'irb', 'php', 'rails', 'bash', 'sh', 'zsh', 'fish', 'cmd', 'powershell']
      const isInteractiveCommand = interactiveCommands.some(cmd =>
          tab.tool.command.toLowerCase().includes(cmd)
      )

      // 修复：如果检测到交互式命令但执行模式不是交互式，自动切换
      // if (isInteractiveCommand && tab.executionMode !== 'interactive') {
      //   tab.executionMode = 'interactive'
      //   ElMessage.info('检测到交互式命令，已自动切换到交互模式')
      // }

      // 如果是定时执行且不是第一次执行，添加分隔符
      if (tab.isScheduling && tab.currentScheduleCount > 0) {
        const separator = '\n' + '='.repeat(80) + '\n'
        const marker = `定时执行 ${tab.currentScheduleCount}/${tab.scheduleConfig.executeTimes} (${new Date().toLocaleString()})\n`
        tab.executionOutput += separator + marker
      }

      const executionId = 'exec-' + Date.now()
      tab.currentExecutionId = executionId
      tab.isExecuting = true

      // 第一次执行时添加开始标记
      if (!tab.isScheduling || tab.currentScheduleCount === 0) {
        const separator = tab.executionOutput ? '\n' + '='.repeat(80) + '\n' : ''
        let modeInfo = ''
        if (tab.executionMode === 'interactive') {
          modeInfo = ' [交互模式]'
        } else if (tab.executionMode === 'web') {
          modeInfo = ' [Web终端模式]'
        }
        const startMarker = `开始执行: ${tab.tool.command}${modeInfo} (${new Date().toLocaleString()})\n`
        tab.executionOutput += separator + startMarker
      }

      const log = {
        id: executionId,
        toolId: tab.tool.id,
        toolName: tab.tool.name,
        command: tab.tool.command,
        parameters: { ...tab.parameterValues },
        executionMode: tab.executionMode,
        status: 'running',
        startTime: new Date().toISOString(),
        output: ''
      }

      // 添加到全局执行历史记录
      this.addExecutionHistory(log)

      if (!this.API_BASE) {
        tab.executionOutput += '错误: 未检测到本地 EasyTools 服务，无法执行命令\n'
        tab.executionOutput += '提示: CLI 命令由桌面主程序在本机执行；请确认 EasyTools 主程序正在运行，或重启后再试。\n'
        tab.isExecuting = false

        this.updateExecutionHistory(executionId, {
          status: 'error',
          endTime: new Date().toISOString(),
          output: tab.executionOutput
        })

        ElMessage.error('未检测到本地 EasyTools 服务，无法执行命令')
        return
      }

      try {
        const executionRequest = {
          toolId: tab.tool.id,
          command: tab.tool.command,
          params: tab.parameterValues,
          workingDir: tab.tool.workingDir,
          encoding: tab.outputEncoding,
          toolParams: tab.tool.parameters,
          // web模式不使用terminal，避免弹出cmd窗口
          terminal: tab.executionMode === 'interactive' ? 1 : 0,
          executionMode: tab.executionMode
        }

        const axiosInstance = this.createAxiosInstance()
        const response = await axiosInstance.post('/execute', executionRequest)
        const result = response.data
        const backendExecutionId = result.id

        // Web 终端模式：初始化终端并连接 WebSocket
        if (tab.executionMode === 'web') {
          // 保存执行ID到tab，用于后续连接
          tab.currentExecutionId = backendExecutionId
          // 延迟一下，确保后端会话已创建（增加延迟时间）
          setTimeout(() => {
            this.initWebTerminalWithRetry(tabId, backendExecutionId, 0)
          }, 1000)
        } else {
          // 启动实时输出监听
          this.startRealtimeOutput(tabId, backendExecutionId)

          // 启动状态轮询
          this.pollExecutionStatus(tabId, backendExecutionId, executionId)
        }

      } catch (error) {
        console.error('Execution error:', error)
        tab.executionOutput += `\n执行错误: ${this.humanizeServiceError(error)}\n`

        this.updateExecutionHistory(executionId, {
          status: 'error',
          endTime: new Date().toLocaleString(),
          output: tab.executionOutput
        })

        tab.isExecuting = false
        // 网络类失败时刷新一次端口配置（主程序可能重启导致动态端口变化），不自动重试避免重复执行
        if (!error || !error.response) {
          this.refreshPortConfigAfterNetworkError()
        }
        ElMessage.error('命令执行失败')
      }
    },

    // 把底层 HTTP/网络报错转成用户可读的中文提示
    humanizeServiceError(error) {
      const raw = (error && (error.message || String(error))) || '未知错误'
      const lower = String(raw).toLowerCase()
      // 无响应/连接失败（CORS 拦截或端口未监听都会表现为 Network Error）
      if (!error || !error.response) {
        if (lower.includes('network') || lower.includes('econnrefused') || lower.includes('timeout')) {
          return '无法连接本地执行服务（网络错误）：请确认 EasyTools 主程序正在运行且本地服务端口正常'
        }
      } else if (error.response && error.response.status === 403) {
        return '本地服务拒绝访问(403)：请从 EasyTools 应用窗口内使用本页面'
      }
      return raw
    },

    // 执行失败(网络原因)后刷新一次端口配置，方便用户再次点击即可命中最新端口
    async refreshPortConfigAfterNetworkError() {
      try {
        const cfg = await discoverLocalPortConfig({ force: true })
        if (cfg) {
          this.portConfig = cfg
          return true
        }
        this.portConfig = null
        return false
      } catch (e) {
        return false
      }
    },

    // 修复：改进轮询执行状态
    async pollExecutionStatus(tabId, backendExecutionId, executionId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      let isCompleted = false
      const maxPollTime = 300000 // 5分钟超时
      const startTime = Date.now()

      const pollInterval = setInterval(async () => {
        if (!tab.isExecuting || isCompleted || Date.now() - startTime > maxPollTime) {
          clearInterval(pollInterval)
          return
        }

        try {
          const axiosInstance = this.createAxiosInstance()
          const statusResponse = await axiosInstance.get(`/status?id=${backendExecutionId}`)
          const statusResult = statusResponse.data

          // 修复：对于交互式命令和Web终端模式，用户手动关闭窗口不视为错误
          const isInteractiveMode = tab.executionMode === 'interactive' || tab.executionMode === 'web'

          if (statusResult.status === 'completed' || statusResult.status === 'failed' || statusResult.status === 'cancelled') {
            isCompleted = true
            tab.isExecuting = false
            clearInterval(pollInterval)

            // 修复：交互式命令被手动关闭时视为成功
            let finalStatus = statusResult.status
            let statusText = '成功'
            let messageType = 'success'

            if (statusResult.status === 'completed') {
              finalStatus = 'completed'
              statusText = '成功'
              messageType = 'success'
            } else if (statusResult.status === 'failed' && isInteractiveMode) {
              // 交互式命令失败（如手动关闭窗口）视为成功
              finalStatus = 'completed'
              statusText = '已结束'
              messageType = 'info'
            } else if (statusResult.status === 'failed') {
              finalStatus = 'failed'
              statusText = '失败'
              messageType = 'error'
            } else if (statusResult.status === 'cancelled' && isInteractiveMode) {
              // 交互式命令取消视为正常结束
              finalStatus = 'completed'
              statusText = '已结束'
              messageType = 'info'
            } else {
              finalStatus = 'cancelled'
              statusText = '已取消'
              messageType = 'info'
            }

            // 处理定时执行逻辑
            if (tab.isScheduling) {
              this.handleScheduleCompletion(tabId)
            } else {
              const endMarker = `\n执行完成: ${statusText} (${new Date().toLocaleString()})\n`
              tab.executionOutput += endMarker
            }

            this.updateExecutionHistory(executionId, {
              status: finalStatus === 'completed' ? 'success' :
                  finalStatus === 'failed' ? 'error' : 'cancelled',
              endTime: new Date().toISOString(),
              output: tab.executionOutput
            })

            if (messageType === 'success') {
              ElMessage.success('命令执行完成')
            } else if (messageType === 'error') {
              ElMessage.error('命令执行失败')
            } else if (messageType === 'info') {
              ElMessage.info('命令执行已结束')
            }
          }
        } catch (error) {
          console.error('Error polling execution status:', error)
          // 修复：对于交互式命令和Web终端模式，轮询错误也不视为失败
          if (tab.executionMode === 'interactive' || tab.executionMode === 'web') {
            // 交互式命令可能因为各种原因轮询失败，但不影响命令本身
            console.log('交互式命令轮询状态失败，继续等待...')
          } else {
            // 非交互式命令的轮询错误需要处理
            if (Date.now() - startTime > 10000) { // 10秒后如果还在出错，认为命令执行失败
              isCompleted = true
              tab.isExecuting = false
              clearInterval(pollInterval)

              const errorMarker = `\n状态检查失败: ${this.humanizeServiceError(error)} (${new Date().toLocaleString()})\n`
              tab.executionOutput += errorMarker

              this.updateExecutionHistory(executionId, {
                status: 'error',
                endTime: new Date().toISOString(),
                output: tab.executionOutput
              })

              ElMessage.error('命令状态检查失败')
            }
          }
        }
      }, 1000)

      // 超时处理
      setTimeout(() => {
        if (!isCompleted) {
          clearInterval(pollInterval)
          tab.isExecuting = false

          // 修复：交互式命令和Web终端模式超时也不视为错误
          if (tab.executionMode === 'interactive' || tab.executionMode === 'web') {
            ElMessage.info('命令执行时间较长，仍在后台运行')
          } else {
            ElMessage.warning('命令执行超时')

            this.updateExecutionHistory(executionId, {
              status: 'error',
              endTime: new Date().toISOString(),
              output: tab.executionOutput + '\n\n执行超时'
            })
          }
        }
      }, maxPollTime)
    },

    // 修复：改进实时输出处理
    startRealtimeOutput(tabId, executionId) {
      if (!this.API_BASE) return

      const tab = this.getTab(tabId)
      if (!tab) return

      // 关闭现有的事件源
      if (tab.eventSource) {
        tab.eventSource.close()
      }

      const eventSource = new EventSource(
        `${this.API_BASE}/realtime-output?id=${executionId}&encoding=${tab.outputEncoding}` +
        `&token=${encodeURIComponent(window.__LOCAL_SERVICE_TOKEN__ || '')}`
      )

      eventSource.onmessage = (event) => {
        if (event.data === '[执行完成]') {
          eventSource.close()
          tab.eventSource = null
          this.scrollToBottom(tabId)
          return
        }

        // 修复：直接追加数据，不添加额外换行符
        tab.executionOutput += event.data + '\n'
        this.scrollToBottom(tabId)
      }

      eventSource.onerror = (error) => {
        console.error('EventSource error:', error)
        eventSource.close()
        tab.eventSource = null
      }

      eventSource.onopen = () => {
        console.log('EventSource connected for execution:', executionId)
      }

      tab.eventSource = eventSource
    },

    // 滚动到底部
    scrollToBottom(tabId) {
      this.$nextTick(() => {
        const outputElement = document.querySelector(`[data-tab="${tabId}"] .output-pre`)
        if (outputElement) {
          outputElement.scrollTop = outputElement.scrollHeight
        }
      })
    },

    // 处理定时执行完成
    handleScheduleCompletion(tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      const endMarker = `\n✅ 第 ${tab.currentScheduleCount}/${tab.scheduleConfig.executeTimes} 次执行完成 (${new Date().toLocaleString()})\n`
      tab.executionOutput += endMarker

      // 检查是否还需要继续执行
      if (tab.currentScheduleCount < tab.scheduleConfig.executeTimes) {
        tab.executionOutput += `⏰ 等待 ${tab.scheduleConfig.intervalSeconds} 秒后执行下一次...\n`

        tab.scheduleTimer = setTimeout(() => {
          this.executeScheduledRun(tabId)
        }, tab.scheduleConfig.intervalSeconds * 1000)
      } else {
        // 所有执行完成
        tab.executionOutput += `\n🎉 定时执行计划完成！共执行 ${tab.scheduleConfig.executeTimes} 次\n`
        this.clearSchedule(tabId)
        ElMessage.success(`定时执行完成，共执行 ${tab.scheduleConfig.executeTimes} 次`)
      }
    },

    // 执行定时运行
    executeScheduledRun(tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      tab.currentScheduleCount++
      tab.executionOutput += `\n🔄 开始第 ${tab.currentScheduleCount}/${tab.scheduleConfig.executeTimes} 次执行...\n`

      // 执行工具
      this.executeTool(tabId)
    },

    // 开始定时执行
    startScheduledExecution(tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      if (!tab.enableSchedule) {
        // 非定时执行模式
        this.executeTool(tabId)
        return
      }

      // 验证定时执行配置
      if (tab.scheduleConfig.delaySeconds < 0) {
        ElMessage.error('延迟时间不能为负数')
        return
      }

      if (tab.scheduleConfig.executeTimes <= 0) {
        ElMessage.error('执行次数必须大于0')
        return
      }

      if (tab.scheduleConfig.intervalSeconds <= 0) {
        ElMessage.error('执行间隔必须大于0')
        return
      }

      // 初始化定时执行状态
      tab.isScheduling = true
      tab.currentScheduleCount = 0

      tab.executionOutput += `\n🚀 开始定时执行计划:\n`
      tab.executionOutput += `   执行次数: ${tab.scheduleConfig.executeTimes} 次\n`
      tab.executionOutput += `   首次延迟: ${tab.scheduleConfig.delaySeconds} 秒\n`
      tab.executionOutput += `   执行间隔: ${tab.scheduleConfig.intervalSeconds} 秒\n`
      tab.executionOutput += `   开始时间: ${new Date().toLocaleString()}\n\n`

      // 设置首次延迟执行
      if (tab.scheduleConfig.delaySeconds > 0) {
        tab.executionOutput += `⏰ 等待 ${tab.scheduleConfig.delaySeconds} 秒后开始第一次执行...\n`
        tab.scheduleTimer = setTimeout(() => {
          this.executeScheduledRun(tabId)
        }, tab.scheduleConfig.delaySeconds * 1000)
      } else {
        // 立即开始第一次执行
        this.executeScheduledRun(tabId)
      }
    },

    // 清除定时执行
    clearSchedule(tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      if (tab.scheduleTimer) {
        clearTimeout(tab.scheduleTimer)
        tab.scheduleTimer = null
      }
      tab.isScheduling = false
      tab.currentScheduleCount = 0
    },

    // 修复：改进取消执行方法
    async cancelExecution(tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      // 清除定时执行
      this.clearSchedule(tabId)

      // 关闭实时输出事件源
      if (tab.eventSource) {
        tab.eventSource.close()
        tab.eventSource = null
      }

      // Web终端模式：清理终端并停止后端会话
      if (tab.executionMode === 'web') {
        // 清理前端终端
        this.cleanupWebTerminal(tabId)
        
        // 停止后端会话
        if (this.API_BASE && tab.currentExecutionId) {
          try {
            const axiosInstance = this.createAxiosInstance()
            await axiosInstance.post(`/stop?id=${tab.currentExecutionId}`)
          } catch (error) {
            console.error('Failed to stop web terminal session:', error)
          }
        }
      } else if (tab.executionMode !== 'interactive') {
        // 非交互式模式：向后端发送停止请求
        if (this.API_BASE && tab.currentExecutionId) {
          try {
            const axiosInstance = this.createAxiosInstance()
            await axiosInstance.post(`/stop?id=${tab.currentExecutionId}`)
          } catch (error) {
            console.error('Failed to cancel execution:', error)
          }
        }
      }

      tab.isExecuting = false

      // 修复：交互式命令和Web终端模式取消时添加友好的结束消息
      if (tab.executionMode === 'interactive' || tab.executionMode === 'web') {
        const endMarker = `\n命令执行已结束 (${new Date().toLocaleString()})\n`
        tab.executionOutput += endMarker
      }

      this.updateExecutionHistory(tab.currentExecutionId, {
        status: (tab.executionMode === 'interactive' || tab.executionMode === 'web') ? 'success' : 'cancelled',
        endTime: new Date().toISOString()
      })

      const message = (tab.executionMode === 'interactive' || tab.executionMode === 'web') ? '命令执行已结束' : '命令执行已取消'
      ElMessage.info(message)
    },

    // 查看执行历史
    viewExecutionHistory(log) {
      // 查找对应的工具标签页
      const toolTab = this.toolTabs.find(tab => tab.tool.id === log.toolId)
      if (toolTab) {
        toolTab.executionOutput = log.output || '无输出记录'
        this.activeTab = toolTab.id
        ElMessage.info('已加载历史执行记录')
      } else {
        // 如果没有对应的标签页，创建一个新的
        const tool = this.tools.find(t => t.id === log.toolId)
        if (tool) {
          this.openTool(tool)
          // 等待标签页创建完成后设置输出
          this.$nextTick(() => {
            const newTab = this.toolTabs.find(tab => tab.tool.id === log.toolId)
            if (newTab) {
              newTab.executionOutput = log.output || '无输出记录'
            }
          })
        } else {
          ElMessage.warning('未找到对应的工具')
        }
      }
    },

    // 清空执行历史
    clearExecutionHistory() {
      this.executionHistory = []
      this.saveExecutionHistory()
      ElMessage.success('执行记录已清空')
    },

    // 清空输出
    clearOutput(tabId) {
      const tab = this.getTab(tabId)
      if (tab) {
        tab.executionOutput = ''
      }
    },

    // 清空最近输出
    clearRecentOutput(tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      const lines = tab.executionOutput.split('\n')
      const lastSeparatorIndex = lines.lastIndexOf('='.repeat(80))

      if (lastSeparatorIndex !== -1) {
        // 保留分隔符之前的内容
        tab.executionOutput = lines.slice(0, lastSeparatorIndex).join('\n')
        if (tab.executionOutput) {
          tab.executionOutput += '\n'
        }
      } else {
        // 如果没有找到分隔符，清空所有输出
        tab.executionOutput = ''
      }
    },

    // 初始化 Web 终端（带重试）
    initWebTerminalWithRetry(tabId, executionId, retryCount) {
      const maxRetries = 5
      const retryDelay = 1000
      
      if (retryCount >= maxRetries) {
        const tab = this.getTab(tabId)
        if (tab) {
          tab.executionOutput += `\n错误: 无法连接到Web终端，已重试${maxRetries}次\n`
        }
        return
      }
      
      this.initWebTerminal(tabId, executionId, retryCount, maxRetries, retryDelay)
    },

    // 初始化 Web 终端
    initWebTerminal(tabId, executionId, retryCount = 0, maxRetries = 5, retryDelay = 1000) {
      const tab = this.getTab(tabId)
      if (!tab || tab.executionMode !== 'web') return

      // 如果已经初始化，不重复初始化
      if (this.terminalInstances[tabId]) {
        return
      }

      // 获取终端容器
      const container = document.querySelector(`[data-terminal-id="${tabId}"]`)
      
      if (!container) {
        // 如果容器不存在，等待 DOM 更新
        this.$nextTick(() => {
          this.initWebTerminal(tabId, executionId, retryCount, maxRetries, retryDelay)
        })
        return
      }

      // 创建终端实例
      const terminal = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: 'Consolas, "Courier New", monospace',
        theme: {
          background: '#1e1e1e',
          foreground: '#d4d4d4',
          cursor: '#aeafad',
          selection: '#264f78'
        }
      })

      const fitAddon = new FitAddon()
      terminal.loadAddon(fitAddon)

      // 挂载终端
      terminal.open(container)
      fitAddon.fit()

      // 保存终端实例
      this.terminalInstances[tabId] = { terminal, fitAddon }

      // 连接 WebSocket（带重试逻辑）
      this.connectWebTerminal(tabId, executionId, terminal, retryCount, maxRetries, retryDelay)
    },

    // 连接 Web 终端 WebSocket
    connectWebTerminal(tabId, executionId, terminal, retryCount = 0, maxRetries = 5, retryDelay = 1000) {
      if (!this.API_BASE) {
        terminal.writeln('错误: 端口配置未加载，无法连接终端')
        return
      }

      // 构建 WebSocket URL
      // API_BASE 格式是 http://127.0.0.1:端口号/cli
      // WebSocket URL 应该是 ws://127.0.0.1:端口号/cli/web-terminal
      // 本地防护要求携带进程级 Token（Wails 页面 Origin 为 null 时校验）
      const wsToken = (window.__LOCAL_SERVICE_TOKEN__ || '')
      let wsURL
      if (this.API_BASE.startsWith('http://')) {
        wsURL = this.API_BASE.replace('http://', 'ws://') + '/web-terminal?executionId=' + executionId + '&token=' + encodeURIComponent(wsToken)
      } else if (this.API_BASE.startsWith('https://')) {
        wsURL = this.API_BASE.replace('https://', 'wss://') + '/web-terminal?executionId=' + executionId + '&token=' + encodeURIComponent(wsToken)
      } else {
        // 如果没有协议，假设是 http
        wsURL = 'ws://' + this.API_BASE + '/web-terminal?executionId=' + executionId + '&token=' + encodeURIComponent(wsToken)
      }
      
      console.log('WebSocket URL:', wsURL, 'ExecutionId:', executionId)

      let ws
      try {
        ws = new WebSocket(wsURL)
      } catch (error) {
        terminal.writeln(`WebSocket 连接失败: ${error.message}`)
        // 重试
        if (retryCount < maxRetries) {
          setTimeout(() => {
            this.connectWebTerminal(tabId, executionId, terminal, retryCount + 1, maxRetries, retryDelay)
          }, retryDelay)
        }
        return
      }

      // 保存 WebSocket 连接
      this.terminalWebSockets[tabId] = ws

      // 连接打开
      ws.onopen = () => {
        terminal.writeln('Web 终端已连接')
        
        // 发送初始大小
        const size = { cols: terminal.cols, rows: terminal.rows }
        ws.send(JSON.stringify({
          type: 'resize',
          width: size.cols,
          height: size.rows
        }))
      }

      // 接收消息
      ws.onmessage = (event) => {
        if (typeof event.data === 'string') {
          // 尝试解析JSON消息（错误消息）
          try {
            const msg = JSON.parse(event.data)
            if (msg.type === 'error') {
              terminal.writeln(`\r\n错误: ${msg.data}`)
              // 如果是session not found，尝试重连（但限制重试次数）
              if (msg.data.includes('Session not found') && retryCount < maxRetries) {
                setTimeout(() => {
                  terminal.writeln(`正在重连... (${retryCount + 1}/${maxRetries})`)
                  this.connectWebTerminal(tabId, executionId, terminal, retryCount + 1, maxRetries, retryDelay)
                }, retryDelay)
                return
              } else if (msg.data.includes('Session not found')) {
                terminal.writeln('\r\n无法连接到终端会话，请检查后端日志')
                return
              }
            } else if (msg.type === 'pong') {
              // ping响应，忽略
              return
            }
          } catch (e) {
            // 不是JSON，是普通文本输出，直接写入终端
            terminal.write(event.data)
          }
        } else {
          // 二进制数据（直接写入终端，保持原始格式）
          // 需要将ArrayBuffer转换为字符串
          if (event.data instanceof ArrayBuffer) {
            const decoder = new TextDecoder('utf-8')
            const text = decoder.decode(event.data)
            terminal.write(text)
          } else if (event.data instanceof Blob) {
            event.data.text().then(text => {
              terminal.write(text)
            })
          } else {
            terminal.write(event.data)
          }
        }
      }

      // 连接错误
      ws.onerror = (error) => {
        terminal.writeln(`WebSocket 错误: ${error.message || '未知错误'}`)
      }

      // 连接关闭
      ws.onclose = (event) => {
        if (event.code !== 1000) { // 正常关闭
          terminal.writeln(`\r\n终端连接已关闭 (代码: ${event.code})`)
        }
        delete this.terminalWebSockets[tabId]
      }

      // 终端输入
      terminal.onData((data) => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({
            type: 'input',
            data: data
          }))
        }
      })

      // 终端大小变化
      terminal.onResize((size) => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({
            type: 'resize',
            width: size.cols,
            height: size.rows
          }))
        }
      })

      // 窗口大小变化时调整终端
      const container = document.querySelector(`[data-terminal-id="${tabId}"]`)
      if (container) {
        const resizeObserver = new ResizeObserver(() => {
          if (this.terminalInstances[tabId] && this.terminalInstances[tabId].fitAddon) {
            this.terminalInstances[tabId].fitAddon.fit()
          }
        })
        resizeObserver.observe(container)
        
        // 保存 resizeObserver 以便清理
        if (this.terminalInstances[tabId]) {
          this.terminalInstances[tabId].resizeObserver = resizeObserver
        }
      }
    },

    // 清理 Web 终端
    cleanupWebTerminal(tabId) {
      // 关闭 WebSocket
      if (this.terminalWebSockets[tabId]) {
        this.terminalWebSockets[tabId].close()
        delete this.terminalWebSockets[tabId]
      }

      // 销毁终端实例
      if (this.terminalInstances[tabId]) {
        if (this.terminalInstances[tabId].resizeObserver) {
          this.terminalInstances[tabId].resizeObserver.disconnect()
        }
        if (this.terminalInstances[tabId].terminal) {
          this.terminalInstances[tabId].terminal.dispose()
        }
        delete this.terminalInstances[tabId]
      }
    },

    // 复制输出
    copyOutput(tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      let contentToCopy = ''

      // 如果是 Web 终端模式，从终端实例获取内容
      if (tab.executionMode === 'web' && this.terminalInstances[tabId]) {
        const terminal = this.terminalInstances[tabId].terminal
        if (terminal) {
          // 获取终端缓冲区的所有内容
          const buffer = terminal.buffer.active
          const lines = []
          
          // 遍历所有行，获取文本内容
          for (let i = 0; i < buffer.length; i++) {
            const line = buffer.getLine(i)
            if (line) {
              const lineText = line.translateToString(true) // true 表示包含尾随空格
              if (lineText.trim() || i < buffer.length - 1) { // 保留非空行或非最后一行
                lines.push(lineText)
              }
            }
          }
          
          contentToCopy = lines.join('\n')
        }
      } else if (tab.executionOutput) {
        // 普通模式，使用 executionOutput
        // 过滤掉执行标记信息，只保留实际输出
        const outputLines = tab.executionOutput.split('\n')
        const filteredLines = outputLines.filter(line => {
          // 过滤掉执行开始和结束的标记行
          return !line.match(/^开始执行:|^命令执行已结束|^命令执行已取消/)
        })
        contentToCopy = filteredLines.join('\n').trim()
      }

      if (!contentToCopy) {
        ElMessage.warning('没有可复制的内容')
        return
      }

      // 复制到剪贴板
      navigator.clipboard.writeText(contentToCopy).then(() => {
        ElMessage.success('输出已复制到剪贴板')
      }).catch(err => {
        console.error('复制失败:', err)
        // 降级方案
        const textArea = document.createElement('textarea')
        textArea.value = contentToCopy
        document.body.appendChild(textArea)
        textArea.select()
        try {
          document.execCommand('copy')
          ElMessage.success('输出已复制到剪贴板')
        } catch (e) {
          ElMessage.error('复制失败，请手动复制')
        }
        document.body.removeChild(textArea)
      })
    },

    // 导出工具配置
    exportTools() {
      const dataStr = JSON.stringify(this.tools, null, 2)
      const dataUri = 'data:application/json;charset=utf-8,'+ encodeURIComponent(dataStr)
      const exportFileDefaultName = 'cli-tools-config.json'

      const linkElement = document.createElement('a')
      linkElement.setAttribute('href', dataUri)
      linkElement.setAttribute('download', exportFileDefaultName)
      linkElement.click()
    },

    // 导入工具配置
    importTools(event) {
      const input = event.target
      if (!input.files?.length) return

      const file = input.files[0]
      const reader = new FileReader()

      reader.onload = (e) => {
        try {
          const importedTools = JSON.parse(e.target?.result)
          if (Array.isArray(importedTools)) {
            this.tools = importedTools
            this.saveTools()
            ElMessage.success('工具配置导入成功')
          } else {
            ElMessage.error('无效的配置文件格式')
          }
        } catch (error) {
          ElMessage.error('配置文件解析失败')
        }
        input.value = ''
      }

      reader.readAsText(file)
    },

    // 新增：处理表头拖拽结束事件
    handleHeaderDragEnd(newWidth, oldWidth, column, event) {
      const columnProperty = column.property || 'parameters' // 对于自定义列，默认使用parameters
      this.columnWidths[columnProperty] = newWidth
      this.saveColumnWidths()
    },

    // 新增：保存列宽设置到本地存储
    saveColumnWidths() {
      localStorage.setItem('cli-history-column-widths', JSON.stringify(this.columnWidths))
    },

    // 新增：从本地存储加载列宽设置
    loadColumnWidths() {
      const savedWidths = localStorage.getItem('cli-history-column-widths')
      if (savedWidths) {
        this.columnWidths = { ...this.columnWidths, ...JSON.parse(savedWidths) }
      }
    },

    // 新增：格式化参数显示
    formatParameters(parameters) {
      if (!parameters) return '-'
      if (typeof parameters === 'string') return parameters

      try {
        const params = typeof parameters === 'string' ? JSON.parse(parameters) : parameters
        const paramArray = []
        for (const key in params) {
          if (params[key] !== undefined && params[key] !== null && params[key] !== '') {
            paramArray.push(`${key}: ${params[key]}`)
          }
        }
        return paramArray.length > 0 ? paramArray.join(', ') : '-'
      } catch (e) {
        return String(parameters)
      }
    },

    // 新增：获取状态文本
    getStatusText(status) {
      const statusMap = {
        'success': '成功',
        'error': '失败',
        'running': '运行中',
        'cancelled': '已取消'
      }
      return statusMap[status] || status
    },

    // 新增：格式化日期时间
    formatDateTime(dateTime) {
      if (!dateTime) return '-'
      return new Date(dateTime).toLocaleString()
    },

    // 打开工具并指定执行模式
    openToolWithMode(tool, mode = 'interactive') {
      const sameToolCount = this.toolTabs.filter(tab => tab.tool.id === tool.id).length
      const tabLabel = sameToolCount > 0 ? `${tool.name} (${sameToolCount + 1})` : tool.name

      const tabId = `tool-${tool.id}-${Date.now()}`
      const newTab = {
        id: tabId,
        tool: { ...tool },
        label: tabLabel,
        parameterValues: {},
        outputEncoding: 'utf-8',
        executionMode: mode,
        enableSchedule: false,
        scheduleConfig: {
          delaySeconds: 5,
          executeTimes: 1,
          intervalSeconds: 10
        },
        isExecuting: false,
        isScheduling: false,
        currentScheduleCount: 0,
        scheduleTimer: null,
        currentExecutionId: '',
        executionOutput: '',
        eventSource: null,
        configCollapsed: false // 参数配置折叠状态，默认展开
      }

      // 初始化参数值（与 openTool 中保持一致）
      if (tool.parameters && Array.isArray(tool.parameters)) {
        tool.parameters.forEach(param => {
          if (param && param.id) {
            if (param.defaultValue !== undefined) {
              newTab.parameterValues[param.id] = param.defaultValue
            } else {
              newTab.parameterValues[param.id] = ''
            }
          }
        })
      }

      this.toolTabs.push(newTab)
      this.activeTab = tabId
    },

    async refreshAIDefault() {
      if (!this.API_BASE) {
        this.aiCliStatus.ok = false
        this.aiCliStatus.bannerMessage = '端口配置未加载，无法读取 AI 状态'
        return
      }
      this.aiCliStatus.refreshing = true
      try {
        const axiosInstance = this.createAxiosInstance()
        const res = await axiosInstance.get('/ai/default', { timeout: 15000 })
        const d = res.data || {}
        this.aiCliStatus.ok = !!d.ok
        this.aiCliStatus.bannerMessage = d.ok ? '' : (d.message || '未找到默认且已启用的 AI 配置')
        if (d.ok) {
          this.aiCliStatus.name = d.name || ''
          this.aiCliStatus.provider = d.provider || ''
          this.aiCliStatus.model = d.model || ''
          this.aiCliStatus.api_base = d.api_base || ''
          this.aiCliStatus.api_key_hint = d.api_key_hint || ''
          this.aiCliStatus.has_api_key = !!d.has_api_key
        } else {
          this.aiCliStatus.name = ''
          this.aiCliStatus.provider = ''
          this.aiCliStatus.model = ''
          this.aiCliStatus.api_base = ''
          this.aiCliStatus.api_key_hint = ''
          this.aiCliStatus.has_api_key = false
        }
      } catch (error) {
        this.aiCliStatus.ok = false
        this.aiCliStatus.bannerMessage = error?.response?.data?.message || error.message || '获取 AI 配置失败'
      } finally {
        this.aiCliStatus.refreshing = false
      }
    },

    async testAIDefault() {
      if (!this.API_BASE) {
        ElMessage.warning('端口配置未加载')
        return
      }
      this.aiCliStatus.testing = true
      try {
        const axiosInstance = this.createAxiosInstance()
        const res = await axiosInstance.post('/ai/test', {}, { timeout: 45000 })
        if (res.data?.ok) {
          ElMessage.success(res.data.message || '测试成功')
        } else {
          ElMessage.error(res.data?.message || '测试失败')
        }
      } catch (error) {
        const msg = error?.response?.data?.message || error?.response?.data || error.message || '测试失败'
        ElMessage.error(String(msg))
      } finally {
        this.aiCliStatus.testing = false
      }
    },

    openAIConfigInSystem() {
      try {
        this.$router.push({ name: 'systemManage', query: { tab: 'ai-config' } })
      } catch (e) {
        console.error(e)
        ElMessage.warning('无法跳转，请手动打开系统管理 → AI配置')
      }
    },

    async loadCLIMCPPref() {
      try {
        this.mcpHttpEnabled = await GetCLIMCPEnabled()
      } catch (e) {
        this.mcpHttpEnabled = true
      }
    },

    async persistCLIMCPPref(val) {
      try {
        await SetCLIMCPEnabled(!!val)
        ElMessage.success(val ? '已开启 MCP HTTP（本进程 Fuzz 端口 /cli/mcp）' : '已关闭 MCP HTTP')
      } catch (e) {
        this.mcpHttpEnabled = !val
        ElMessage.error(`保存失败: ${e}`)
      }
    },

    isAICommandLine(line) {
      const t = (line || '').trim()
      if (!t) return false
      const lo = t.toLowerCase()
      if (lo === 'help' || lo === '?' || lo === 'clear') return true
      if (lo.startsWith('mcp ')) return true
      return false
    },

    buildAiHelpText() {
      return [
        'help | ?     本说明',
        'clear        清空会话',
        'mcp list     列出 CLI 工具（需开启「MCP HTTP」）',
        'mcp call <工具名或ID> {"参数":"值"}  调用工具；可无 JSON 表示 {}',
        '其他输入     发给默认大模型（多轮上下文）',
        'stdio 子进程：EasyTools mcp --stdio（供 Cursor 等，不受 HTTP 开关影响）。'
      ].join('\n')
    },

    clearAISession() {
      this.aiMessages = []
      this.aiSysLines = []
      this.aiInputLine = ''
    },

    appendAiSys(text) {
      this.aiSysLines.push(text)
      this.$nextTick(() => this.scrollAiPanels())
    },

    scrollAiPanels() {
      const c = this.$refs.aiConsoleScroll
      const k = this.$refs.aiChatScroll
      if (c) c.scrollTop = c.scrollHeight
      if (k) k.scrollTop = k.scrollHeight
    },

    parseMcpCallLine(line) {
      const m = line.match(/^mcp\s+call\s+(.+)$/i)
      if (!m) {
        throw new Error('格式: mcp call <工具名> {JSON可选}')
      }
      let rest = m[1].trim()
      const idx = rest.indexOf('{')
      let name
      let jsonStr = '{}'
      if (idx < 0) {
        name = rest
      } else {
        name = rest.slice(0, idx).trim()
        jsonStr = rest.slice(idx).trim() || '{}'
      }
      if (!name) {
        throw new Error('缺少工具名')
      }
      let args = {}
      try {
        args = JSON.parse(jsonStr)
      } catch (e) {
        throw new Error(`JSON 无效: ${e.message}`)
      }
      return { name, args }
    },

    async runMcpList() {
      if (!this.API_BASE) {
        throw new Error('端口未加载')
      }
      if (!this.mcpHttpEnabled) {
        throw new Error('请先在工具栏开启「MCP HTTP」')
      }
      const axiosInstance = this.createAxiosInstance()
      const res = await axiosInstance.get('/mcp/tools', { timeout: 30000 })
      const tools = res.data?.tools || []
      if (!tools.length) return '(无工具)'
      return tools.map(t => `${t.name}: ${t.description || ''}`).join('\n')
    },

    async runMcpCall(name, args) {
      if (!this.API_BASE) {
        throw new Error('端口未加载')
      }
      if (!this.mcpHttpEnabled) {
        throw new Error('请先在工具栏开启「MCP HTTP」')
      }
      const axiosInstance = this.createAxiosInstance()
      const res = await axiosInstance.post('/mcp/call', { name, arguments: args || {} }, { timeout: 600000 })
      const blocks = res.data?.content
      if (Array.isArray(blocks) && blocks[0]?.text) {
        return blocks[0].text
      }
      return JSON.stringify(res.data)
    },

    async submitAIPrompt() {
      const raw = (this.aiInputLine || '').trim()
      if (!raw) return

      const lo = raw.toLowerCase()

      if (lo === 'help' || raw === '?') {
        this.appendAiSys(this.buildAiHelpText())
        this.aiInputLine = ''
        return
      }
      if (lo === 'clear') {
        this.clearAISession()
        return
      }

      if (lo === 'mcp list') {
        this.aiInputLine = ''
        this.appendAiSys('>> mcp list')
        try {
          const txt = await this.runMcpList()
          this.appendAiSys(txt)
        } catch (e) {
          const msg = e?.response?.data?.error || e?.response?.data?.message || e?.response?.data || e.message
          this.appendAiSys(`错误: ${msg}`)
        }
        return
      }

      if (/^mcp\s+call\s+/i.test(raw)) {
        this.aiInputLine = ''
        this.appendAiSys(`>> ${raw}`)
        try {
          const { name, args } = this.parseMcpCallLine(raw)
          const out = await this.runMcpCall(name, args)
          this.appendAiSys(out)
        } catch (e) {
          const msg = e?.response?.data?.error || e?.response?.data?.message || e?.response?.data || e.message
          this.appendAiSys(`错误: ${msg}`)
        }
        return
      }

      if (!this.aiCliStatus.ok) {
        ElMessage.warning('请先在系统管理中配置并启用默认大模型')
        return
      }
      if (!this.API_BASE) {
        ElMessage.warning('端口配置未加载')
        return
      }

      const messages = [...this.aiMessages, { role: 'user', content: raw }]
      this.aiInputLine = ''
      this.aiSending = true
      try {
        const axiosInstance = this.createAxiosInstance()
        const res = await axiosInstance.post('/ai/chat', { messages }, { timeout: 130000 })
        if (res.data?.ok) {
          this.aiMessages = [...messages, { role: 'assistant', content: res.data.content || '' }]
        } else {
          this.aiMessages = messages
          ElMessage.error(res.data?.message || '调用失败')
        }
        this.$nextTick(() => this.scrollAiPanels())
      } catch (error) {
        this.aiMessages = messages
        const msg = error?.response?.data?.message || error?.response?.data?.error || error.message || '调用失败'
        ElMessage.error(String(msg))
      } finally {
        this.aiSending = false
      }
    },

    copyMcpStdioExample() {
      const text = this.mcpStdioExampleText
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => {
          ElMessage.success('已复制到剪贴板')
        }).catch(() => {
          ElMessage.warning('复制失败，请手动选择文本复制')
        })
      } else {
        ElMessage.warning('当前环境不支持一键复制')
      }
    },

    openEasyToolsDoc() {
      try {
        BrowserOpenURL('https://www.yuque.com/doki-byte/fdzks6/rcim7hylfoh1gvg6?singleDoc#')
      } catch (error) {
        console.error('打开链接失败:', error)
      }
    },

  }
}
</script>

<style scoped>
/* 样式保持不变，与之前相同 */
.cli-executor-container {
  height: 96vh;
  display: flex;
  flex-direction: column;
  --wails-drop-target: drop;
}

/* 覆盖 Element Plus tabs 的默认 padding */
:deep(.el-tabs--border-card > .el-tabs__content) {
  padding: 2px;
}

.main-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  /* 与其他页面一致的玻璃卡片风格（原来是透明背景，很难看） */
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(3px);
  -webkit-backdrop-filter: blur(3px);
  border: 1px solid rgba(255, 255, 255, 0.55);
  border-radius: 12px;
  margin: 10px;
  box-shadow: 0 10px 28px rgba(69, 107, 183, 0.18);
  position: relative;
}

.main-tabs :deep(.el-tabs__header) {
  background: transparent;
  border-radius: 12px 12px 0 0;
}

.main-tabs :deep(.el-tabs__content) {
  background: transparent;
}

.main-tabs :deep(.el-tabs__item) {
  color: var(--tabs-font-color, #303133);
  font-weight: var(--tabs-font-weight, bold);
}

.main-tabs :deep(.el-tabs__item.is-active) {
  color: var(--tabs-font-color, #303133);
  font-weight: var(--tabs-font-weight, bold);
}

.tabs-context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
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
  background-color: #f5f7fa;
  color: #409eff;
}

:deep(.el-tabs__content) {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 0;
}

:deep(.el-tab-pane) {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* 工具列表页样式 */
.tools-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.page-header {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 10px 20px 8px;
  border-bottom: 1px solid #e5e7eb;
}

.page-header h1 {
  color: #1f2937;
  margin: 0;
  font-size: 28px;
}

.page-header p {
  color: #6b7280;
  margin: 0;
  font-size: 16px;
}


.tools-action-bar {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 0 20px;
  gap: 16px;
}

.search-box {
  width: 300px;
}

.action-buttons {
  display: flex;
  gap: 8px;
  position: relative;
}

.import-button {
  position: relative;
  overflow: hidden;
}

.file-input-hidden {
  position: absolute;
  opacity: 0;
  width: 100%;
  height: 100%;
  left: 0;
  top: 0;
  cursor: pointer;
}

.tools-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.tools-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 10px;
  overflow-y: auto;
  padding-right: 15px;
  padding-left: 15px;
  margin-bottom: 20px;
  min-height: 0;
}

.tool-card {
  transition: all 0.3s ease;
  border: 1px solid #e5e7eb;
  height: 250px; /* 固定高度，确保所有卡片大小一致 */
  display: flex;
  flex-direction: column;
}

.tool-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(0,0,0,0.1);
}

/* 确保 el-card 的 body 也使用 flex 布局 */
:deep(.tool-card .el-card__body) {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px;
}

.tool-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  flex-shrink: 0; /* 不收缩 */
}

.tool-card-header h3 {
  margin: 0;
  color: #1f2937;
  font-size: 18px;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-shrink: 0;
}

.tool-description {
  color: #6b7280;
  font-size: 14px;
  margin-bottom: 12px;
  line-height: 1.4;
  flex-shrink: 0;
  max-height: 42px; /* 限制描述高度，约3行 */
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 3; /* 最多显示3行 */
  line-clamp: 3; /* 标准属性，用于兼容性 */
  -webkit-box-orient: vertical;
}

.tool-command {
  margin-bottom: 12px;
  flex-shrink: 0;
}

.tool-parameters {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 16px;
  flex: 1; /* 占据剩余空间 */
  min-height: 0; /* 允许收缩 */
  overflow-y: auto; /* 如果参数太多，允许滚动 */
  max-height: 80px; /* 限制参数区域最大高度 */
}

.tool-parameters::-webkit-scrollbar {
  width: 4px;
}

.tool-parameters::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 2px;
}

.tool-parameters::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 2px;
}

.param-tag {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 11px;
}

.tool-footer {
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0; /* 不收缩，始终在底部 */
  margin-top: auto; /* 推到底部 */
}

/* 运行页样式 */
.runner-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
}

.runner-header {
  flex-shrink: 0;
  margin-bottom: 12px;
  padding: 20px 16px;
  border-bottom: 1px solid #e5e7eb;
  background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
  border-radius: 0;
}

.runner-header-title {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.runner-header h2 {
  color: #1f2937;
  margin: 0;
  font-size: 20px;
  flex-shrink: 0;
}

.runner-header .tool-description {
  margin: 0;
  font-size: 12px;
  color: #6b7280;
  flex: 1;
  min-width: 0;
}

.tool-command-display {
  margin-top: 6px;
}

.runner-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0 12px 12px;
  gap: 12px;
  min-height: 0;
}

.config-card {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-radius: 6px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  transition: all 0.3s ease;
}

.config-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
}

:deep(.config-card .el-card__header) {
  padding: 10px 14px;
  background: linear-gradient(135deg, #f5f7fa 0%, #ffffff 100%);
  border-bottom: 1px solid #e4e7ed;
  border-radius: 6px 6px 0 0;
}

.config-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  user-select: none;
  transition: all 0.2s ease;
}

.config-card-header:hover {
  opacity: 0.8;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.collapse-icon {
  font-size: 16px;
  color: #409eff;
  transition: transform 0.3s ease;
  flex-shrink: 0;
}

.collapse-icon.collapsed {
  transform: rotate(180deg);
}

.config-card-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  flex: 1;
}

.param-count-tag {
  margin-left: 6px;
  font-size: 11px;
}

:deep(.config-card .el-card__body) {
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow: hidden;
  padding: 12px 14px;
}

.config-wrapper {
  display: flex;
  flex-direction: column;
  max-height: 500px;
  animation: slideDown 0.3s ease;
}

/* 优化表单项间距 */
:deep(.config-content .el-form-item) {
  margin-bottom: 14px;
}

:deep(.config-content .el-form-item:last-child) {
  margin-bottom: 5px;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.config-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 0;
  padding-right: 6px;
  padding-top: 4px;
}

.config-content::-webkit-scrollbar {
  width: 6px;
}

.config-content::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.config-content::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.config-content::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

.file-input {
  display: flex;
  gap: 8px;
}

.param-description {
  font-size: 11px;
  color: #6b7280;
  margin-top: 3px;
  line-height: 1.4;
}

.execution-buttons {
  display: flex;
  gap: 10px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #f0f2f5;
  flex-shrink: 0;
  background: linear-gradient(to bottom, transparent, #fafbfc);
  border-radius: 0 0 6px 6px;
  padding-left: 2px;
  padding-right: 2px;
}

.output-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 300px;
  border-radius: 6px;
}

:deep(.output-card .el-card__header) {
  padding: 10px 14px;
  border-bottom: 1px solid #e4e7ed;
}

:deep(.output-card .el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 12px 14px;
}

.output-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.output-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.output-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 修改：添加固定高度输出样式 */
.fixed-height-output {
  flex: 1;
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 10px 12px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  line-height: 1.4;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  word-wrap: break-word;
  min-height: 200px;
  /* 设置固定最大高度，可根据需要调整 */
  max-height: 500px;
  height: 100%;
}

/* 历史页样式 */
.history-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
}

.history-page .page-header {
  padding: 20px 20px 0;
}

.history-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0 20px 20px;
}

:deep(.history-page .el-table) {
  flex: 1;
  overflow: auto;
}

/* CLI — AI 大模型（紧凑 + 终端） */
.cli-ai-compact {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 6px 10px 8px;
  background: #fff;
  overflow: hidden;
}

.cli-ai-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 10px;
  padding: 4px 0 6px;
  border-bottom: 1px solid #ebeef5;
}

.cli-ai-title {
  font-weight: 600;
  font-size: 14px;
  color: #303133;
  margin-right: 4px;
}

.cli-ai-toolbar-spacer {
  flex: 1;
  min-width: 8px;
}

.cli-ai-mcp-label {
  font-size: 12px;
  color: #606266;
}

.cli-ai-btngrp {
  flex-shrink: 0;
}

.cli-ai-meta-line {
  font-size: 12px;
  color: #303133;
  padding: 4px 0 2px;
  line-height: 1.4;
  word-break: break-all;
}

.cli-ai-meta-dim {
  color: #909399;
}

.cli-ai-alert-tight {
  margin-top: 4px;
  padding: 4px 8px;
}

.cli-ai-alert-tight :deep(.el-alert__title) {
  font-size: 12px;
  line-height: 1.35;
}

.cli-ai-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  margin-top: 6px;
  gap: 6px;
}

.cli-ai-console {
  flex: 1;
  min-height: 160px;
  max-height: min(42vh, 360px);
  overflow: auto;
  padding: 8px 10px;
  border-radius: 6px;
  background: #0d1117;
  color: #c9d1d9;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
  border: 1px solid #30363d;
}

.cli-line {
  white-space: pre-wrap;
  word-break: break-word;
  margin-bottom: 2px;
}

.cli-line-sys {
  color: #d29922;
}

.cli-line-in {
  color: #58a6ff;
}

.cli-line-out {
  color: #7ee787;
}

.cli-ai-chat-wrap {
  flex: 1;
  min-height: 120px;
  max-height: min(38vh, 320px);
  overflow: auto;
  padding: 6px 8px;
  background: #f6f8fa;
  border-radius: 6px;
  border: 1px solid #e1e4e8;
}

.cli-chat-bubble {
  margin-bottom: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.45;
}

.cli-chat-bubble.is-user {
  background: #e7f3ff;
  border: 1px solid #b3d8ff;
}

.cli-chat-bubble.is-assistant {
  background: #fff;
  border: 1px solid #e1e4e8;
}

.cli-chat-role {
  display: block;
  font-size: 11px;
  color: #909399;
  margin-bottom: 2px;
}

.cli-chat-text {
  white-space: pre-wrap;
  word-break: break-word;
}

.cli-ai-prompt-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.cli-prompt-glyph {
  color: #409eff;
  font-weight: 700;
  font-family: ui-monospace, monospace;
  font-size: 14px;
}

.cli-ai-input {
  flex: 1;
  min-width: 0;
}

.cli-ai-collapse {
  margin-top: 4px;
  border: none;
}

.cli-ai-collapse :deep(.el-collapse-item__header) {
  height: 32px;
  line-height: 32px;
  font-size: 12px;
  padding: 0 4px;
  background: transparent;
}

.cli-ai-collapse :deep(.el-collapse-item__wrap) {
  border: none;
}

.cli-ai-collapse :deep(.el-collapse-item__content) {
  padding-bottom: 8px;
  font-size: 12px;
}

.collapse-mcp-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.cli-ai-hint {
  margin: 0 0 6px;
  color: #606266;
  font-size: 12px;
  line-height: 1.4;
}

.cli-code-tight pre {
  margin: 0;
  font-size: 11px;
  max-height: 140px;
  overflow: auto;
}

.action-card,
.guide-card {
  border-radius: 12px;
  transition: all 0.3s ease;
  border: 1px solid #e5e7eb;
}

.action-card:hover,
.guide-card:hover {
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.header-icon {
  font-size: 20px;
  color: #409eff;
}

.action-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}

.status-message {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #ebeef5;
}

.guide-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.guide-section h3 {
  margin: 0 0 12px 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.guide-section p {
  margin: 0 0 12px 0;
  color: #606266;
  line-height: 1.6;
}

.guide-section ol,
.guide-section ul {
  margin: 0;
  padding-left: 24px;
  color: #606266;
  line-height: 1.8;
}

.guide-section li {
  margin-bottom: 8px;
}

.guide-section li strong {
  color: #303133;
}

.code-block {
  background: #1e1e1e;
  border-radius: 8px;
  padding: 16px;
  margin-top: 12px;
  overflow-x: auto;
}

.code-block pre {
  margin: 0;
  color: #d4d4d4;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.code-block code {
  color: #d4d4d4;
}

.guide-section code {
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Monaco', 'Menlo', monospace;
  color: #409eff;
  font-size: 12px;
}

.link-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: flex-start;
}

.parameters-display {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #666;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 参数编辑样式 */
.parameters-list {
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 16px;
  background: #f9fafb;
}

.parameter-item {
  padding: 16px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  margin-bottom: 16px;
  background: white;
}

.parameter-header {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.parameter-actions {
  display: flex;
  gap: 4px;
  margin-left: auto;
}

.parameter-options {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.select-options {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
  margin-top: 8px;
}

.default-value {
  margin-top: 8px;
}

/* 空状态和禁用状态 */
.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #6b7280;
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}

.empty-state p {
  margin-bottom: 16px;
  font-size: 16px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .tools-action-bar {
    flex-direction: column;
    gap: 12px;
  }

  .search-box, .action-buttons {
    width: 100%;
  }

  .tools-grid {
    grid-template-columns: 1fr;
  }

  .parameter-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .parameter-actions {
    margin-left: 0;
    margin-top: 8px;
  }
}

.tool-edit-dialog .el-dialog__body {
  max-height: 70vh;
  overflow-y: auto;
}

.schedule-config {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.schedule-options {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  background: #f8f9fa;
  border-radius: 6px;
  border: 1px solid #e9ecef;
}

.schedule-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.schedule-label {
  color: #606266;
  font-size: 14px;
  min-width: 70px;
}

.schedule-unit {
  color: #909399;
  font-size: 12px;
}

/* 执行状态指示 */
:deep(.el-button--success.is-disabled) {
  background-color: #f4f4f5;
  border-color: #e4e7ed;
  color: #c0c4cc;
}

/* Web 终端样式 */
.web-terminal-container {
  width: 100%;
  height: 600px;
  min-height: 400px;
  background: #1e1e1e;
  padding: 10px;
  border-radius: 4px;
}

.web-terminal-container :deep(.xterm) {
  height: 100%;
}

.web-terminal-container :deep(.xterm-viewport) {
  background: #1e1e1e !important;
}

/* 确保中文字体显示 */
.cli-executor-container {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'PingFang SC', 'Microsoft YaHei', sans-serif;
}

/* 分页样式 */
.tools-pagination, .history-pagination {
  flex-shrink: 0;
  display: flex;
  justify-content: center;
  margin-top: 20px;
  padding: 16px 0;
  border-top: 1px solid #e5e7eb;
}

/* 滚动条优化 */
:deep(.el-tabs__content) {
  scrollbar-width: thin;
  scrollbar-color: #c1c1c1 #f1f1f1;
  padding: 5px;
}

:deep(.el-tabs__content::-webkit-scrollbar) {
  width: 6px;
}

:deep(.el-tabs__content::-webkit-scrollbar-track) {
  background: #f1f1f1;
  border-radius: 3px;
}

:deep(.el-tabs__content::-webkit-scrollbar-thumb) {
  background: #c1c1c1;
  border-radius: 3px;
}

:deep(.el-tabs__content::-webkit-scrollbar-thumb:hover) {
  background: #a8a8a8;
}

/* 输出区域滚动优化 - 修改为固定高度输出样式 */
.fixed-height-output::-webkit-scrollbar {
  width: 8px;
}

.fixed-height-output::-webkit-scrollbar-track {
  background: #2d2d2d;
  border-radius: 4px;
}

.fixed-height-output::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 4px;
}

.fixed-height-output::-webkit-scrollbar-thumb:hover {
  background: #777;
}

/* 工具网格滚动优化 */
.tools-grid::-webkit-scrollbar {
  width: 6px;
}

.tools-grid::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.tools-grid::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.tools-grid::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

/* 配置内容滚动优化 */
.config-content::-webkit-scrollbar {
  width: 6px;
}

.config-content::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.config-content::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.config-content::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

/* 新增：历史记录表格样式优化 */
.history-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0 10px 10px;
}

:deep(.history-page .el-table) {
  flex: 1;
  overflow: auto;
}

:deep(.history-page .el-table th) {
  background-color: #f5f7fa;
  color: #606266;
  font-weight: 600;
}

:deep(.history-page .el-table .cell) {
  word-break: break-word;
  line-height: 1.4;
}

.parameters-display {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #666;
  line-height: 1.3;
}

/* 表格列宽拖拽样式优化 */
:deep(.history-page .el-table__header-wrapper .el-table__header th) {
  position: relative;
}

:deep(.history-page .el-table__header-wrapper .el-table__header th .cell) {
  display: flex;
  align-items: center;
}

/* 分页样式保持不变 */
.history-pagination {
  flex-shrink: 0;
  display: flex;
  justify-content: center;
  margin-top: 10px;
  padding: 16px 0;
  border-top: 1px solid #e5e7eb;
}

/* 响应式设计 */
@media (max-width: 768px) {
  :deep(.history-page .el-table) {
    font-size: 12px;
  }

  :deep(.history-page .el-table .el-tag) {
    font-size: 10px;
    padding: 0 4px;
    height: 20px;
    line-height: 20px;
  }

  .parameters-display {
    font-size: 10px;
  }
}
</style>

<!-- 全局作用域，确保 Wails 拖拽生效覆盖 -->
<style>
:global(.system) {
  --wails-drop-target: drop;
  min-height: 100vh;
}
</style>