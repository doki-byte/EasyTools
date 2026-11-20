<template>
  <div class="dbmanager-container">
    <!-- 折叠/展开按钮，始终悬浮在左上角 -->
    <el-button
      class="dbmanager-toggle-sidebar-btn"
      size="small"
      circle
      @click="sidebarCollapsed = !sidebarCollapsed"
      :title="sidebarCollapsed ? '展开连接列表' : '收起连接列表'">
      <el-icon>
        <component :is="sidebarCollapsed ? 'ArrowRight' : 'ArrowLeft'" />
      </el-icon>
    </el-button>

    <!-- 左侧：连接管理区域 -->
    <div v-if="!sidebarCollapsed" class="dbmanager-sidebar">
      <div class="sidebar-header">
        <el-button type="primary" size="small" @click="showConfigDialog = true">新建连接</el-button>
      </div>
      <div class="sidebar-content">
        <el-scrollbar>
          <div class="config-list">
            <div v-for="config in configs" :key="config.id"
              :class="['config-item', { active: activeConfigId === String(config.id) }]"
              @click="handleConfigSelect(String(config.id))">
              <div class="config-info">
                <div class="config-name">{{ config.name }}</div>
                <div class="config-meta">
                  <el-tag size="small" :type="getTypeTagType(config.type)">{{ config.type }}</el-tag>
                  <span class="config-host">{{ getConfigHostDisplay(config) }}</span>
                </div>
              </div>
              <div class="config-actions" @click.stop>
                <el-button link type="primary" size="small" @click="editConfig(config)" title="编辑">
                  <el-icon>
                    <Edit />
                  </el-icon>
                </el-button>
                <!-- 使用默认 Teleport 到 body，避免被侧边栏/滚动容器裁剪 -->
                <el-button link type="danger" size="small" title="删除" @click="deleteConfig(config.id)">
                  <el-icon>
                    <Delete />
                  </el-icon>
                </el-button>
              </div>
            </div>
            <div v-if="configs.length === 0" class="empty-config">
              <el-empty description="暂无连接配置" :image-size="80" />
            </div>
          </div>
        </el-scrollbar>
      </div>
    </div>

    <!-- 右侧：数据库操作区域 -->
    <div class="dbmanager-main">
      <template v-if="tabs.length > 0">
        <el-tabs v-model="activeTabId" type="card" closable @tab-remove="handleTabRemove" @tab-click="handleTabClick"
          @contextmenu.prevent="handleTabsContextMenu" class="connection-tabs">
          <el-tab-pane v-for="tab in tabs" :key="tab.id" :label="tab.label" :name="tab.id">
            <template #label>
              <span class="tab-label">
                <el-icon class="tab-icon">
                  <Connection />
                </el-icon>
                {{ tab.label }}
              </span>
            </template>
            <div class="tab-content">
              <div class="toolbar">
                <el-button size="small" @click="refreshDatabases(tab.id)">刷新</el-button>
                <!-- <el-button size="small" @click="setTabActiveTab(tab.id, 'query')">SQL查询</el-button> -->
              </div>
              <el-tabs v-model="tab.activeTab" class="db-tabs">
                <el-tab-pane label="数据库" name="databases">
                  <div class="database-panel">
                    <div class="database-tree-wrapper">
                      <el-scrollbar class="database-tree-scrollbar">
                        <el-tree :data="tab.databaseTree" :props="{ label: 'name', children: 'children' }"
                          @node-click="(node) => handleNodeClick(node, tab.id)" class="database-tree"
                          :class="{ 'tree-with-data': tab.selectedTable }">
                          <template #default="{ node, data }">
                            <div class="tree-node">
                              <el-icon v-if="data.type === 'table'" class="table-icon">
                                <Document />
                              </el-icon>
                              <el-icon v-else-if="data.isPlaceholder" class="empty-icon">
                                <InfoFilled />
                              </el-icon>
                              <el-icon v-else class="database-icon">
                                <Folder />
                              </el-icon>
                              <span class="tree-node-label" :class="{ 'empty-hint': data.isPlaceholder }">{{ node.label
                                }}</span>
                            </div>
                          </template>
                        </el-tree>
                      </el-scrollbar>
                    </div>
                    <!-- 表数据查看区域 -->
                    <div v-if="tab.selectedTable" class="table-data-panel">
                      <div class="table-header">
                        <div class="table-header-left">
                          <h3>{{ tab.selectedTable.database }}.{{ tab.selectedTable.table }}</h3>
                          <span class="config-host-info">
                            {{ tab.config.name }} ({{ tab.config.host }}:{{ tab.config.port }})
                          </span>
                        </div>
                        <el-button size="small" @click="loadTableData(tab.id)">刷新数据</el-button>
                      </div>
                      <el-tabs v-model="tab.tableDataTab" class="table-data-tabs"
                        @tab-click="(pane) => handleTableDataTabClick(pane, tab.id)">
                        <el-tab-pane label="数据" name="data">
                          <div class="table-data-container">
                            <div class="table-toolbar">
                              <div class="toolbar-left">
                                <el-button size="small" type="primary" @click="handleAddRow(tab.id)"
                                  :icon="Plus">新增</el-button>
                                <el-button size="small" type="danger" @click="handleDeleteSelectedRows(tab.id)"
                                  :disabled="tab.selectedRows.length === 0" :icon="Delete">删除选中</el-button>
                                <el-button size="small" @click="loadTableData(tab.id)" :icon="Refresh">刷新</el-button>
                              </div>
                              <div class="toolbar-right">
                                <el-pagination v-if="tab.selectedTable"
                                  v-model:current-page="tab.tablePagination.currentPage"
                                  v-model:page-size="tab.tablePagination.pageSize" :page-sizes="[20, 50, 100, 200, 500]"
                                  :total="tab.tablePagination.total" :disabled="tab.tableDataLoading"
                                  layout="prev, pager, next, jumper, sizes, total" small
                                  @size-change="(size) => handlePageSizeChange(size, tab.id)"
                                  @current-change="(page) => handlePageChange(page, tab.id)" />
                              </div>
                            </div>
                            <div class="table-wrapper" v-loading="tab.tableDataLoading && tab.tableData.length === 0"
                              element-loading-text="加载中...">
                              <el-table
                                :key="`table-${tab.id}-${tab.selectedTable?.database}-${tab.selectedTable?.table}-${tab.tablePagination.currentPage}`"
                                :data="tab.tableData" border stripe style="width: 100%"
                                @selection-change="(selection) => handleSelectionChange(selection, tab.id)"
                                :row-key="getRowKey"
                                :class="{ 'table-loading': tab.tableDataLoading && tab.tableData.length > 0 }">
                                <el-table-column type="selection" width="55" />
                                <el-table-column v-for="col in tab.tableColumns" :key="col.field || col.name"
                                  :prop="col.field || col.name" :label="col.field || col.name" :min-width="150"
                                  :show-overflow-tooltip="{ popperClass: 'db-table-tooltip', placement: 'top', effect: 'dark' }">
                                  <template #default="scope">
                                    <span v-if="!scope.row._editing"
                                      class="db-table-cell-text"
                                      :class="{
                                        'db-table-cell-null':
                                          scope.row[col.field || col.name] === null ||
                                          scope.row[col.field || col.name] === undefined
                                      }"
                                      @dblclick="openCellViewer(scope.row, col.field || col.name, tab.selectedTable?.table, tab.id)">
                                      <template
                                        v-if="scope.row[col.field || col.name] === null || scope.row[col.field || col.name] === undefined">
                                        (null)
                                      </template>
                                      <template v-else>
                                        {{ formatCellValue(scope.row[col.field || col.name]) }}
                                      </template>
                                    </span>
                                    <el-input v-else v-model="scope.row[col.field || col.name]" size="small"
                                      @blur="handleCellBlur(scope.row, col.field || col.name)" />
                                  </template>
                                </el-table-column>
                                <el-table-column label="操作" width="170" fixed="right">
                                  <template #default="scope">
                                    <template v-if="scope.row._editing">
                                      <el-button link type="success" size="small"
                                        @click="handleSaveRow(scope.row, tab.id)">
                                        保存
                                      </el-button>
                                      <el-button link size="small"
                                        @click="handleCancelEditRow(scope.row, tab.id)">
                                        取消
                                      </el-button>
                                    </template>
                                    <template v-else>
                                      <el-button link type="primary" size="small"
                                        @click="handleEditRow(scope.row, tab.id)">
                                        编辑
                                      </el-button>
                                      <el-button link type="danger" size="small"
                                        @click="handleDeleteRow(scope.row, tab.id)">
                                        删除
                                      </el-button>
                                    </template>
                                  </template>
                                </el-table-column>
                              </el-table>
                            </div>
                          </div>
                        </el-tab-pane>
                        <el-tab-pane label="字段" name="columns">
                          <div class="columns-container">
                            <el-table :data="tab.tableColumns" border stripe style="width: 100%" empty-text="暂无字段信息"
                              v-loading="tab.tableDataLoading" element-loading-text="加载字段信息中...">
                              <el-table-column prop="field" label="字段名" min-width="150" />
                              <el-table-column prop="type" label="类型" min-width="120" />
                              <el-table-column prop="length" label="长度" min-width="100" />
                              <el-table-column prop="nullable" label="可空" width="80">
                                <template #default="{ row }">
                                  {{ row.nullable ? '是' : '否' }}
                                </template>
                              </el-table-column>
                              <el-table-column prop="key" label="键" width="80">
                                <template #default="{ row }">
                                  <el-tag v-if="row.key === 'primary' || row.key === 'PRI'" type="warning"
                                    size="small">主键</el-tag>
                                  <span v-else>{{ row.key || '-' }}</span>
                                </template>
                              </el-table-column>
                              <el-table-column prop="default" label="默认值" min-width="120">
                                <template #default="{ row }">
                                  {{ row.default || '-' }}
                                </template>
                              </el-table-column>
                              <el-table-column prop="comment" label="注释" min-width="200" show-overflow-tooltip>
                                <template #default="{ row }">
                                  {{ row.comment || '-' }}
                                </template>
                              </el-table-column>
                            </el-table>
                          </div>
                        </el-tab-pane>
                      </el-tabs>
                    </div>
                  </div>
                </el-tab-pane>
                <el-tab-pane label="查询" name="query">
                  <div class="query-panel">
                    <div class="query-toolbar">
                      <div class="query-selectors">
                        <el-select v-model="tab.querySelectedDatabase" placeholder="选择数据库" clearable filterable
                          size="small" style="width: 200px;" @change="(val) => handleQueryDatabaseChange(val, tab.id)">
                          <el-option v-for="db in tab.databaseTree" :key="db.name" :label="db.name" :value="db.name" />
                        </el-select>
                        <el-select v-model="tab.querySelectedTable" placeholder="选择表" clearable filterable size="small"
                          style="width: 200px;" :disabled="!tab.querySelectedDatabase"
                          @change="(val) => handleQueryTableChange(val, tab.id)">
                          <el-option v-for="table in tab.queryTableList" :key="table.name" :label="table.name"
                            :value="table.name" />
                        </el-select>
                        <el-button size="small" :disabled="!tab.querySelectedDatabase || !tab.querySelectedTable"
                          @click="insertTableToQuery(tab.id)" title="填充表名到SQL编辑器">
                          <el-icon>
                            <Plus />
                          </el-icon>
                          填充表名
                        </el-button>
                      </div>
                      <div class="query-actions">
                        <el-dropdown @command="(cmd) => insertTemplate(cmd, tab.id)" trigger="click"
                          popper-class="sql-command-dropdown">
                          <el-button size="small" type="primary">
                            常用命令 <el-icon>
                              <ArrowDown />
                            </el-icon>
                          </el-button>
                          <template #dropdown>
                            <el-dropdown-menu class="sql-command-menu">
                              <template v-for="category in sqlTemplateCategories" :key="category">
                                <el-dropdown-item divided>{{ category }}</el-dropdown-item>
                                <el-dropdown-item v-for="tpl in getTemplatesByCategory(category, tab.id)" :key="tpl.id"
                                  :command="tpl.command">
                                  {{ tpl.name }} ({{ tpl.command }})
                                </el-dropdown-item>
                              </template>
                            </el-dropdown-menu>
                          </template>
                        </el-dropdown>
                        <el-button size="small" @click="showSqlTemplateDialog = true" :icon="Setting">语句管理</el-button>
                        <el-button type="primary" @click="executeQuery(tab.id)"
                          :loading="tab.queryLoading">执行</el-button>
                        <el-button @click="clearQuery(tab.id)">清空</el-button>
                      </div>
                    </div>
                    <div class="query-editor-wrapper">
                      <VueMonacoEditor v-if="monacoReady" v-model:value="tab.querySql" language="sql" theme="vs"
                        :height="queryEditorHeight" :options="monacoOptions" class="query-editor" />
                    </div>
                    <div class="query-result-container">
                      <div v-if="tab.queryResult" class="query-result">
                        <div class="result-header">
                          <span>查询结果 ({{ tab.queryResult.rows?.length || 0 }} 行)</span>
                          <span v-if="tab.queryResult.columns" style="margin-left: 10px; font-size: 12px; color: #999;">
                            ({{ tab.queryResult.columns.length }} 列)
                          </span>
                        </div>
                        <el-table v-if="tab.queryResult.columns && tab.queryResult.columns.length > 0"
                          :data="tab.queryResult.rows || []" border stripe max-height="400" style="width: 100%"
                          empty-text="暂无数据" :default-empty-text="'暂无数据'">
                          <el-table-column v-for="(col, index) in tab.queryResult.columns"
                            :key="col.field || col.name || index" :prop="col.field || col.name"
                            :label="col.field || col.name || 'Unknown'" :min-width="150"
                            :show-overflow-tooltip="{ popperClass: 'db-table-tooltip', placement: 'top', effect: 'dark' }">
                            <template #default="{ row }">
                              <span
                                :class="{
                                  'db-table-cell-null':
                                    row[col.field || col.name] === null ||
                                    row[col.field || col.name] === undefined
                                }">
                                <template
                                  v-if="row[col.field || col.name] === null || row[col.field || col.name] === undefined">
                                  (null)
                                </template>
                                <template v-else>
                                  {{ row[col.field || col.name] }}
                                </template>
                              </span>
                            </template>
                          </el-table-column>
                        </el-table>
                        <div v-if="!tab.queryResult.columns || tab.queryResult.columns.length === 0"
                          class="no-columns-warning">
                          <el-alert type="warning" :closable="false">
                            <template #title>警告：未找到列信息</template>
                            <template #default>
                              查询返回了数据，但缺少列定义信息。这可能是后端数据格式问题。
                              <div v-if="tab.queryResult.rows && tab.queryResult.rows.length > 0">
                                数据行数：{{ tab.queryResult.rows.length }}
                              </div>
                              <div style="margin-top: 10px;">
                                调试信息：queryResult = {{ JSON.stringify(Object.keys(tab.queryResult || {})) }}
                              </div>
                            </template>
                          </el-alert>
                        </div>
                      </div>
                      <div v-if="tab.queryError" class="query-error">
                        <el-alert :title="tab.queryError" type="error" :closable="true" @close="tab.queryError = null"
                          show-icon />
                      </div>
                    </div>
                  </div>
                </el-tab-pane>
              </el-tabs>
            </div>
          </el-tab-pane>
        </el-tabs>
      </template>
      <div v-else class="dbmanager-empty-state">
        <div class="empty-content">
          <el-icon class="empty-icon">
            <Connection />
          </el-icon>
          <p class="empty-text">请先选择一个数据库连接</p>
        </div>
      </div>
    </div>

    <!-- 配置对话框 -->
    <el-dialog v-model="showConfigDialog" :title="configForm.id ? '编辑连接配置' : '新建连接配置'" width="600px"
      @close="handleConfigDialogClose" @open="handleConfigDialogOpen">
      <el-form :model="configForm" label-width="100px">
        <el-form-item label="类型">
          <el-select v-model="configForm.type" placeholder="选择数据库类型"
            @change="handleTypeChange">
            <el-option label="MySQL" value="mysql" />
            <el-option label="Doris (旧版兼容)" value="doris_legacy" />
            <el-option label="PostgreSQL" value="postgresql" />
            <el-option label="SQLite" value="sqlite" />
            <el-option label="Oracle" value="oracle" />
            <el-option label="SQL Server" value="sqlserver" />
            <el-option label="MariaDB" value="mariadb" />
            <el-option label="达梦 DM" value="dm" />
            <el-option label="人大金仓 Kingbase" value="kingbase" />
            <el-option label="南大通用 GBase" value="gbase" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="configForm.name" />
        </el-form-item>
        <el-form-item label="主机" v-if="configForm.type !== 'sqlite'">
          <el-input v-model="configForm.host" placeholder="例如: 127.0.0.1" />
        </el-form-item>
        <el-form-item label="端口" v-if="configForm.type !== 'sqlite'">
          <el-input v-model="configForm.port" placeholder="自动填充默认端口" />
        </el-form-item>
        <el-form-item label="用户名" v-if="configForm.type !== 'sqlite'">
          <el-input v-model="configForm.username" placeholder="自动填充默认用户名" />
        </el-form-item>
        <el-form-item label="密码" v-if="configForm.type !== 'sqlite'">
          <el-input v-model="configForm.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="数据库文件" v-if="configForm.type === 'sqlite'">
          <el-input v-model="configForm.db_file" placeholder="拖拽 SQLite 数据库文件到窗口，或直接输入绝对路径" />
        </el-form-item>
        <el-form-item label="连接方式" v-if="configForm.type !== 'sqlite'">
          <el-select v-model="configForm.connection_type" placeholder="请选择连接方式" clearable
            @change="handleConnectionTypeChange">
            <el-option value="direct" label="直连" />
            <el-option value="ssh" label="SSH隧道" />
          </el-select>
        </el-form-item>
        <div v-show="configForm.connection_type === 'ssh' && configForm.type !== 'sqlite'">
          <el-form-item label="SSH主机">
            <el-select v-model="configForm.ssh_host_id" placeholder="选择SSH主机或手动输入" filterable allow-create
              default-first-option @change="handleSshHostChange" clearable>
              <el-option v-for="host in sshHostList" :key="host.id"
                :label="`${host.name} (${host.address}:${host.port})`" :value="host.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="SSH 地址">
            <el-input v-model="configForm.ssh_addr" placeholder="SSH服务器地址，例如: 192.168.1.100" />
          </el-form-item>
          <el-form-item label="SSH 端口">
            <el-input v-model="configForm.ssh_port" placeholder="SSH端口，默认: 22" />
          </el-form-item>
          <el-form-item label="SSH 用户名">
            <el-input v-model="configForm.ssh_username" placeholder="SSH登录用户名" />
          </el-form-item>
          <el-form-item label="SSH 密码">
            <el-input v-model="configForm.ssh_password" type="password" show-password placeholder="SSH登录密码" />
          </el-form-item>
        </div>
        <el-form-item>
          <el-button @click="testConnection" :loading="testingConnection" type="success">
            <el-icon v-if="!testingConnection">
              <Connection />
            </el-icon>
            测试连接
          </el-button>
          <el-button v-if="!configForm.id" type="primary" @click="saveConfig">创建</el-button>
          <el-button v-else type="primary" @click="saveConfig">保存</el-button>
          <el-button @click="showConfigDialog = false">取消</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>

    <!-- 配置选择对话框 -->
    <el-dialog v-model="showConfigSelectDialog" title="选择数据库连接" width="500px">
      <el-radio-group :value="null" @change="handleConfigSelectFromDialog">
        <div class="config-select-list">
          <el-radio v-for="config in configs" :key="config.id" :label="String(config.id)" class="config-select-item">
            <div class="config-select-info">
              <div class="config-select-name">{{ config.name }}</div>
              <div class="config-select-meta">
                <el-tag size="small" :type="getTypeTagType(config.type)">{{ config.type }}</el-tag>
                <span>{{ getConfigHostDisplay(config) }}</span>
              </div>
            </div>
          </el-radio>
        </div>
      </el-radio-group>
      <template #footer>
        <el-button @click="showConfigSelectDialog = false">取消</el-button>
      </template>
    </el-dialog>
    <!-- 右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
    <Teleport to="body">
      <ul class="tabs-context-menu" v-if="tabsContextMenuVisible" :style="tabsContextMenuStyle">
        <li @click="handleCloseOtherTabs">关闭其他</li>
        <li @click="handleCloseAllTabs">全部关闭</li>
      </ul>
    </Teleport>

    <!-- SQL 模板管理对话框 -->
    <el-dialog v-model="showSqlTemplateDialog" title="SQL 语句管理" width="1000px"
      @open="async () => { await this.initClient(); await this.loadCategoriesOnInit(); await this.loadSqlTemplates(); }">
      <div class="config-toolbar" style="display: flex; align-items: center; gap: 8px; flex-wrap: wrap;">
        <el-button type="primary" @click="handleAddSqlTemplate" :icon="Plus">新增</el-button>
        <el-button @click="handleResetSqlTemplates" type="warning" :icon="Refresh">恢复初始值</el-button>
        <el-button @click="loadSqlTemplates" :icon="Refresh">刷新</el-button>
        <el-button @click="showCategoryManageDialog = true" :icon="Setting">分类管理</el-button>
        <el-select v-model="sqlTemplateFilter.dbType" placeholder="数据库类型" clearable style="width: 150px;"
          @change="loadSqlTemplates">
          <el-option label="MySQL" value="mysql" />
          <el-option label="Doris (旧版兼容)" value="doris_legacy" />
          <el-option label="PostgreSQL" value="postgresql" />
          <el-option label="SQLite" value="sqlite" />
          <el-option label="Oracle" value="oracle" />
          <el-option label="SQL Server" value="sqlserver" />
          <el-option label="MariaDB" value="mariadb" />
          <el-option label="达梦 DM" value="dm" />
          <el-option label="人大金仓" value="kingbase" />
          <el-option label="南大通用" value="gbase" />
        </el-select>
        <el-select v-model="sqlTemplateFilter.category" placeholder="分类" clearable style="width: 150px;"
          @change="loadSqlTemplates">
          <el-option v-for="category in sqlTemplateCategories" :key="category" :label="category" :value="category" />
        </el-select>
        <el-input v-model="sqlTemplateSearch" placeholder="搜索语句" clearable style="width: 200px;"
          @input="loadSqlTemplates">
          <template #prefix>
            <el-icon>
              <Search />
            </el-icon>
          </template>
        </el-input>
      </div>
      <el-table :data="sqlTemplates" border stripe style="width: 100%; margin-top: 12px;"
        v-loading="sqlTemplateLoading">
        <el-table-column type="index" width="60" label="#" />
        <el-table-column prop="dbType" label="数据库类型" width="120" />
        <el-table-column prop="category" label="分类" width="120" />
        <el-table-column prop="name" label="名称" min-width="150" />
        <el-table-column prop="command" label="命令" min-width="150" />
        <el-table-column prop="template" label="模板" min-width="300" show-overflow-tooltip />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleEditSqlTemplate(row)">编辑</el-button>
            <el-button link type="danger" size="small" @click="handleDeleteSqlTemplate(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination v-model:current-page="sqlTemplatePagination.page"
        v-model:page-size="sqlTemplatePagination.pageSize" :total="sqlTemplatePagination.total"
        :page-sizes="[10, 20, 50, 100]" layout="total, sizes, prev, pager, next"
        style="margin-top: 12px; justify-content: flex-end;" @size-change="loadSqlTemplates"
        @current-change="loadSqlTemplates" />
    </el-dialog>

    <!-- SQL 模板编辑对话框 -->
    <el-dialog v-model="showSqlTemplateEditDialog" :title="editingSqlTemplate.id ? '编辑 SQL 模板' : '新增 SQL 模板'"
      width="700px">
      <el-form :model="editingSqlTemplate" label-width="100px">
        <el-form-item label="数据库类型" required>
          <el-select v-model="editingSqlTemplate.dbType" placeholder="选择数据库类型">
            <el-option label="MySQL" value="mysql" />
            <el-option label="Doris (旧版兼容)" value="doris_legacy" />
            <el-option label="PostgreSQL" value="postgresql" />
            <el-option label="SQLite" value="sqlite" />
            <el-option label="Oracle" value="oracle" />
            <el-option label="SQL Server" value="sqlserver" />
            <el-option label="MariaDB" value="mariadb" />
            <el-option label="达梦 DM" value="dm" />
            <el-option label="人大金仓" value="kingbase" />
            <el-option label="南大通用" value="gbase" />
          </el-select>
        </el-form-item>
        <el-form-item label="分类" required>
          <el-select v-model="editingSqlTemplate.category" placeholder="选择或输入分类" filterable allow-create
            default-first-option style="width: 100%;">
            <el-option v-for="category in sqlTemplateCategories" :key="category" :label="category" :value="category" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="editingSqlTemplate.name" placeholder="请输入模板名称" />
        </el-form-item>
        <el-form-item label="命令" required>
          <el-input v-model="editingSqlTemplate.command" placeholder="请输入命令标识（如：select、insert等）" />
        </el-form-item>
        <el-form-item label="模板" required>
          <el-input v-model="editingSqlTemplate.template" type="textarea" :rows="5"
            placeholder="请输入 SQL 模板（可使用 table_name 作为占位符）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showSqlTemplateEditDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSaveSqlTemplate">保存</el-button>
      </template>
    </el-dialog>

    <!-- 分类管理对话框 -->
    <el-dialog v-model="showCategoryManageDialog" title="分类管理" width="600px" @open="loadCategories">
      <div class="category-toolbar" style="margin-bottom: 12px;">
        <el-button type="primary" @click="handleAddCategory" :icon="Plus">新增分类</el-button>
      </div>
      <el-table :data="categories" border stripe style="width: 100%;" v-loading="categoryLoading">
        <el-table-column type="index" width="60" label="#" />
        <el-table-column prop="name" label="分类名称" min-width="200">
          <template #default="{ row }">
            <span v-if="!row._editing">{{ row.name }}</span>
            <el-input v-else v-model="row._editName" size="small" @blur="handleSaveCategoryEdit(row)"
              @keyup.enter="handleSaveCategoryEdit(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button v-if="!row._editing" link type="primary" size="small" @click="handleEditCategory(row)">
              编辑
            </el-button>
            <el-button v-else link type="success" size="small" @click="handleSaveCategoryEdit(row)">
              保存
            </el-button>
            <el-button link type="danger" size="small" :disabled="row._editing" @click="handleDeleteCategory(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 新增分类对话框 -->
    <el-dialog v-model="showAddCategoryDialog" title="新增分类" width="400px">
      <el-form :model="newCategory" label-width="100px">
        <el-form-item label="分类名称" required>
          <el-input v-model="newCategory.name" placeholder="请输入分类名称" @keyup.enter="handleSaveNewCategory" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddCategoryDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSaveNewCategory">确定</el-button>
      </template>
    </el-dialog>

    <!-- 单元格内容查看对话框（类 Navicat + Monaco 编辑器效果） -->
    <el-dialog v-model="cellViewer.visible" :title="cellViewer.title" width="70%" top="15vh" append-to-body
      destroy-on-close class="cell-viewer-dialog">
      <div class="cell-viewer-toolbar">
        <div class="toolbar-left">
          <el-button size="small" @click="formatCellViewer" :disabled="!cellViewer.value">
            一键格式化
          </el-button>
        </div>
      </div>
      <VueMonacoEditor v-if="cellViewer.visible" v-model:value="cellViewer.value" :language="cellViewer.language"
        theme="vs-dark" height="56vh" class="cell-viewer-editor" :options="cellViewerMonacoOptions" />
      <template #footer>
        <el-button @click="copyCellViewer" :disabled="!cellViewer.value">复制内容</el-button>
        <el-button type="primary" @click="saveCellFromViewer">保存到当前单元格</el-button>
        <el-button @click="cellViewer.visible = false">关闭</el-button>
      </template>
    </el-dialog>

  </div>
</template>

<script>
import { GetPortConfig } from '../../../wailsjs/go/controller/Server';
import { OnFileDrop, EventsOn } from '../../../wailsjs/runtime';
import axios from 'axios';
import { VueMonacoEditor } from '@guolao/vue-monaco-editor';
import { ArrowDown, ArrowRight, ArrowLeft, Edit, Delete, Plus, Refresh, Document, Folder, Connection, InfoFilled, Setting, Search } from '@element-plus/icons-vue';
import { getToken } from '../../utils/token';
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';

export default {
  name: "DBManager",
  components: {
    VueMonacoEditor
  },
  setup() {
    return {
      Edit,
      Delete,
      Plus,
      Refresh,
      Document,
      Folder,
      Connection,
      ArrowRight,
      ArrowLeft,
      InfoFilled,
      Setting,
      Search
    };
  },
  data() {
    return {
      portConfig: null,
      apiClient: null,
      sshApiClient: null, // SSH模块的API客户端
      configs: [],
      sshHostList: [], // SSH主机列表
      showConfigDialog: false,
      showConfigSelectDialog: false,
      sidebarCollapsed: false,
      monacoReady: false,
      queryEditorHeight: '300px',
      // Wails 拖拽得到的绝对路径（优先使用）
      wailsDropPaths: [],
      // 标签页管理
      tabs: [],
      activeTabId: null,
      tabIdCounter: 0,
      activeConfigId: null, // 当前激活的连接ID
      // 标签页右键菜单
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: '',
      monacoOptions: {
        fontSize: 14,
        minimap: { enabled: true },
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        automaticLayout: true,
        tabSize: 2,
        formatOnPaste: true,
        formatOnType: true
      },
      configForm: {
        id: null,
        type: 'mysql',
        name: '',
        host: '',
        port: '',
        username: '',
        password: '',
        db_file: '',
        connection_type: '', // 连接方式：'direct' 或 'ssh'
        ssh_host_id: null, // 选择的SSH主机ID
        ssh_addr: '',
        ssh_port: '',
        ssh_username: '',
        ssh_password: ''
      },
      testingConnection: false,
      testResult: null,
      // SQL 模板管理
      showSqlTemplateDialog: false,
      sqlTemplates: [],
      sqlTemplateLoading: false,
      sqlTemplateSearch: '',
      sqlTemplatePagination: {
        page: 1,
        pageSize: 10,
        total: 0
      },
      sqlTemplateFilter: {
        dbType: '',
        category: ''
      },
      editingSqlTemplate: {
        id: '',
        dbType: 'mysql',
        command: '',
        name: '',
        category: '',
        template: ''
      },
      showSqlTemplateEditDialog: false,
      sqlTemplateCategories: ['查询语句', '聚合查询', '数据操作', '表结构', '性能分析'],
      sqlTemplatesCache: {}, // 缓存模板数据，key 为 tabId
      // 分类管理
      showCategoryManageDialog: false,
      categories: [],
      categoryLoading: false,
      showAddCategoryDialog: false,
      newCategory: {
        name: ''
      },
      // 单元格查看对话框
      cellViewer: {
        visible: false,
        title: '',
        value: '',
        language: 'plaintext',
        // 记录当前编辑的行和字段，方便保存
        row: null,
        field: '',
        tableName: '',
        tabId: null
      },
      cellViewerMonacoOptions: {
        minimap: { enabled: false },
        wordWrap: 'on',
        fontSize: 13,
        scrollbar: { verticalScrollbarSize: 8 },
        automaticLayout: true,
        scrollBeyondLastLine: false,
        lineNumbers: 'on'
      }
    };
  },
  computed: {
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
  async mounted() {
    // 获取端口配置并初始化客户端
    await this.initClient();
    // 加载配置列表
    await this.loadConfigs();
    // 加载SSH主机列表
    await this.loadSshHostList();
    // 加载分类列表
    await this.loadCategoriesOnInit();
    // 初始化Monaco编辑器
    this.initMonaco();
    // 监听填充连接表单事件
    window.addEventListener('fill-connect-form', this.handleFillConnectForm);

    // 仅在视图可见时处理 Wails 级别的拖拽，优先使用绝对路径
    OnFileDrop((x, y, paths) => {
      if (paths && paths.length && this.isViewActive()) {
        this.wailsDropPaths = paths;
        this.applySqliteDroppedPath(paths[0]);
      }
    });

    // 后端广播的全局拖拽事件（确保绝对路径）
    EventsOn('global-file-drop', (paths) => {
      if (paths && paths.length && this.isViewActive()) {
        this.wailsDropPaths = paths;
        this.applySqliteDroppedPath(paths[0]);
      }
    });
  },
  beforeUnmount() {
    // 清理事件监听器
    window.removeEventListener('fill-connect-form', this.handleFillConnectForm);
  },
  methods: {
    // 视图是否可见（限制跨页面填充）
    isViewActive() {
      return this.$el && document.body.contains(this.$el) && this.$el.offsetParent !== null;
    },
    // 将拖拽得到的路径应用到 SQLite 配置
    applySqliteDroppedPath(filePath) {
      if (!filePath) return;
      if (!this.isViewActive()) return;
      if (this.configForm.type === 'sqlite') {
        this.configForm.db_file = filePath;

        // 如果未填写名称，自动使用文件名（去掉扩展名）
        if (!this.configForm.name || this.configForm.name.trim() === '') {
          const fileName = (filePath.split(/[/\\]/).pop() || '').trim();
          const nameWithoutExt = fileName.replace(/\.[^/.]+$/, '');
          this.configForm.name = nameWithoutExt || fileName;
        }

        this.$message?.success?.('已通过拖拽选择 SQLite 数据库文件：' + this.configForm.name);
      }
    },

    async initClient() {
      if (this.apiClient) {
        return; // 已经初始化过了
      }
      await this.loadPortConfig();
      this.createHttpClient();
      this.createSshHttpClient();
      console.log('DBManager API 客户端初始化成功');
    },
    async loadPortConfig() {
      try {
        this.portConfig = await GetPortConfig();
        console.log('端口配置:', this.portConfig);
      } catch (error) {
        console.error('获取端口配置失败:', error);
        this.portConfig = {
          dbManagerPort: 52869
        };
      }
    },
    createHttpClient() {
      const port = this.portConfig?.dbManagerPort || 52869;
      const host = '127.0.0.1';
      const baseURL = `http://${host}:${port}`;
      const client = axios.create({
        baseURL: baseURL,
        timeout: 20000,
        // 在浏览器环境中，axios 会自动使用系统代理设置
        // 如果需要显式配置代理，可以通过代理服务器设置
        // 注意：浏览器环境中的代理由系统设置控制，这里不需要额外配置
      });
      this.apiClient = client;
    },
    createSshHttpClient() {
      const port = this.portConfig?.sshPort || 52868;
      const host = '127.0.0.1';
      const baseURL = `http://${host}:${port}`;
      const client = axios.create({
        baseURL: baseURL,
        timeout: 20000,
      });
      // 添加请求拦截器，添加token认证
      client.interceptors.request.use(
        (req) => {
          req.headers.Time = String(Date.now());
          const token = getToken();
          if (token) {
            req.headers.Authorization = token;
          }
          return req;
        },
        (err) => Promise.reject(err)
      );
      this.sshApiClient = client;
    },
    async loadConfigs() {
      if (!this.apiClient) {
        console.error('API客户端未初始化');
        return;
      }
      try {
        const response = await this.apiClient.get('/api/config');
        if (response.data.code === 0) {
          this.configs = response.data.data || [];
        }
      } catch (e) {
        console.error('加载配置失败:', e);
      }
    },
    async loadSshHostList() {
      if (!this.sshApiClient) {
        console.error('SSH API客户端未初始化');
        return;
      }
      try {
        const response = await this.sshApiClient.get('/api/conn_conf');
        if (response.data.code === 0) {
          this.sshHostList = response.data.data || [];
        }
      } catch (e) {
        console.error('加载SSH主机列表失败:', e);
        this.sshHostList = [];
      }
    },

    // 打开单元格内容查看对话框（双击单元格）
    // 打开“单元格编辑”模式的大编辑器
    openCellViewer(row, field, tableName, tabId) {
      const raw = row && field ? row[field] : null;
      let display = '';

      if (raw === null || raw === undefined) {
        display = '';
      } else if (typeof raw === 'object') {
        try {
          display = JSON.stringify(raw, null, 2);
        } catch (e) {
          display = String(raw);
        }
      } else {
        display = String(raw);
      }

      // 简单语言检测：JSON 优先，其次 SQL / XML / 纯文本
      const trimmed = display.trim();
      let language = 'plaintext';
      if ((trimmed.startsWith('{') && trimmed.endsWith('}')) ||
        (trimmed.startsWith('[') && trimmed.endsWith(']'))) {
        language = 'json';
      } else if (/^(select|update|insert|delete)\s/i.test(trimmed)) {
        language = 'sql';
      } else if (trimmed.startsWith('<') && trimmed.endsWith('>')) {
        language = 'xml';
      }

      // 确保 originalRowData 中有当前行的快照，供 handleSaveRow 使用
      const currentTabId = tabId ?? this.activeTabId;
      const tab = this.getTab(currentTabId);
      if (tab && tab.originalRowData && row && row._rowId != null && !tab.originalRowData.has(row._rowId)) {
        // 存一份当前值的浅拷贝作为“原始数据”快照
        tab.originalRowData.set(row._rowId, { ...row });
      }

      this.cellViewer.title = `${tableName || '数据表'} - ${field}`;
      this.cellViewer.value = display;
      this.cellViewer.language = language;
      this.cellViewer.row = row;
      this.cellViewer.field = field;
      this.cellViewer.tableName = tableName || '';
      this.cellViewer.tabId = currentTabId;
      this.cellViewer.visible = true;
    },

    // 复制查看对话框中的内容
    copyCellViewer() {
      if (!this.cellViewer.value) return;
      navigator.clipboard
        .writeText(this.cellViewer.value)
        .then(() => {
          this.$message?.success?.('已复制到剪贴板');
        })
        .catch((err) => {
          this.$message?.error?.(`复制失败: ${err.message}`);
        });
    },

    // 一键格式化当前单元格内容（优先按 JSON 处理，失败则做简单缩进处理）
    formatCellViewer() {
      if (!this.cellViewer.value) return;
      const original = this.cellViewer.value;

      // JSON 优先
      const trimmed = original.trim();
      if (
        (trimmed.startsWith('{') && trimmed.endsWith('}')) ||
        (trimmed.startsWith('[') && trimmed.endsWith(']'))
      ) {
        try {
          const obj = JSON.parse(trimmed);
          this.cellViewer.value = JSON.stringify(obj, null, 2);
          this.cellViewer.language = 'json';
          this.$message?.success?.('已按 JSON 进行格式化');
          return;
        } catch (e) {
          // JSON 解析失败则继续走普通格式化
        }
      }

      // 普通文本：按行去除多余空格，保持现有语言类型
      const lines = original.split('\n');
      const formatted = lines
        .map((line) => line.replace(/\s+$/g, '')) // 去掉行尾空格
        .join('\n')
        .trimEnd();
      this.cellViewer.value = formatted;
      this.$message?.success?.('已简单格式化文本');
    },

    // 从 Monaco 编辑器保存内容回当前行，并复用行保存逻辑
    async saveCellFromViewer() {
      const { row, field, tabId } = this.cellViewer;
      if (!row || !field || tabId == null) {
        this.$message?.warning?.('缺少上下文，无法保存当前单元格');
        return;
      }

      try {
        // 将编辑后的值写回当前行字段
        row[field] = this.cellViewer.value;
        await this.handleSaveRow(row, tabId);
        this.$message?.success?.('单元格已保存');
        this.cellViewer.visible = false;
      } catch (e) {
        console.error('保存单元格失败:', e);
        this.$message?.error?.('保存单元格失败');
      }
    },
    async testConnection() {
      if (!this.apiClient) {
        this.$message.error('API客户端未初始化');
        return;
      }

      // 验证必填字段
      if (this.configForm.type !== 'sqlite') {
        if (!this.configForm.host || this.configForm.host.trim() === '') {
          this.$message.warning('请填写主机地址');
          return;
        }
        if (!this.configForm.port || this.configForm.port.trim() === '') {
          this.$message.warning('请填写端口号');
          return;
        }
        if (!this.configForm.username || this.configForm.username.trim() === '') {
          this.$message.warning('请填写用户名');
          return;
        }
      } else {
        if (!this.configForm.db_file || this.configForm.db_file.trim() === '') {
          this.$message.warning('请填写数据库文件路径');
          return;
        }
      }

      // 如果选择了SSH隧道，必须验证SSH配置完整性
      if (this.configForm.connection_type === 'ssh') {
        if (!this.configForm.ssh_addr || this.configForm.ssh_addr.trim() === '') {
          this.$message.warning('SSH隧道模式下，SSH地址不能为空');
          return;
        }
        if (!this.configForm.ssh_username || this.configForm.ssh_username.trim() === '') {
          this.$message.warning('SSH隧道模式下，SSH用户名不能为空');
          return;
        }
        if (!this.configForm.ssh_password || this.configForm.ssh_password.trim() === '') {
          this.$message.warning('SSH隧道模式下，SSH密码不能为空');
          return;
        }
        // 如果SSH端口为空，设置默认值
        if (!this.configForm.ssh_port || this.configForm.ssh_port.trim() === '') {
          this.configForm.ssh_port = '22';
        }
      }

      this.testingConnection = true;
      this.testResult = null;

      try {
        // 构建测试连接的数据对象
        const testConfig = {
          id: this.configForm.id || null,
          type: this.configForm.type,
          name: this.configForm.name || '测试连接',
          host: this.configForm.type === 'sqlite' ? '' : (this.configForm.host || '').trim(),
          port: this.configForm.type === 'sqlite' ? '' : (this.configForm.port || '').trim(),
          username: this.configForm.type === 'sqlite' ? '' : (this.configForm.username || '').trim(),
          password: this.configForm.password || '',
          charset: this.configForm.charset || '',
          timeout: this.configForm.timeout || 10,
          read_timeout: this.configForm.read_timeout || 30,
          write_timeout: this.configForm.write_timeout || 30,
          db_file: this.configForm.type === 'sqlite' ? (this.configForm.db_file || '').trim() : '',
          connection_type: this.configForm.connection_type || '',
          ssh_addr: '',
          ssh_port: '',
          ssh_username: '',
          ssh_password: ''
        };

        // 如果连接方式是SSH，设置SSH配置
        if (testConfig.connection_type === 'ssh') {
          testConfig.ssh_addr = (this.configForm.ssh_addr || '').trim();
          testConfig.ssh_port = (this.configForm.ssh_port || '22').trim();
          testConfig.ssh_username = (this.configForm.ssh_username || '').trim();
          testConfig.ssh_password = this.configForm.ssh_password || '';
        }

        const response = await this.apiClient.post('/api/ping', testConfig);

        if (response.data.code === 0) {
          this.testResult = {
            success: true,
            message: '连接测试成功！'
          };
          this.$message.success('连接测试成功');
        } else {
          this.testResult = {
            success: false,
            message: response.data.msg || '连接测试失败'
          };
          this.$message.error(response.data.msg || '连接测试失败');
        }
      } catch (e) {
        const errorMsg = e.response?.data?.msg || e.message || '连接测试失败';
        this.testResult = {
          success: false,
          message: errorMsg
        };
        this.$message.error('连接测试失败: ' + errorMsg);
        console.error('连接测试失败:', e);
      } finally {
        this.testingConnection = false;
      }
    },
    async saveConfig() {
      if (!this.apiClient) {
        this.$message.error('API客户端未初始化');
        return;
      }
      try {
        // 基础验证
        if (!this.configForm.type) {
          this.$message.warning('请选择数据库类型');
          return;
        }
        if (!this.configForm.name || this.configForm.name.trim() === '') {
          this.$message.warning('请输入连接名称');
          return;
        }

        // 根据数据库类型验证必填字段
        if (this.configForm.type !== 'sqlite') {
          if (!this.configForm.host || this.configForm.host.trim() === '') {
            this.$message.warning('请输入主机地址');
            return;
          }
          if (!this.configForm.port || this.configForm.port.trim() === '') {
            this.$message.warning('请输入端口号');
            return;
          }
          if (!this.configForm.username || this.configForm.username.trim() === '') {
            this.$message.warning('请输入用户名');
            return;
          }
        } else {
          if (!this.configForm.db_file || this.configForm.db_file.trim() === '') {
            this.$message.warning('请选择SQLite数据库文件');
            return;
          }
        }

        // 如果连接方式是SSH，验证SSH配置完整性
        if (this.configForm.connection_type === 'ssh') {
          if (!this.configForm.ssh_addr || this.configForm.ssh_addr.trim() === '') {
            this.$message.warning('SSH隧道模式下，SSH地址不能为空');
            return;
          }
          if (!this.configForm.ssh_username || this.configForm.ssh_username.trim() === '') {
            this.$message.warning('SSH隧道模式下，SSH用户名不能为空');
            return;
          }
          if (!this.configForm.ssh_password || this.configForm.ssh_password.trim() === '') {
            this.$message.warning('SSH隧道模式下，SSH密码不能为空');
            return;
          }
          // 如果SSH端口为空，设置默认值
          if (!this.configForm.ssh_port || this.configForm.ssh_port.trim() === '') {
            this.configForm.ssh_port = '22';
          }
        }

        // 构建要保存的配置对象
        const configToSave = {
          id: this.configForm.id || null,
          type: this.configForm.type,
          name: this.configForm.name.trim(),
          host: this.configForm.type === 'sqlite' ? '' : (this.configForm.host || '').trim(),
          port: this.configForm.type === 'sqlite' ? '' : (this.configForm.port || '').trim(),
          username: this.configForm.type === 'sqlite' ? '' : (this.configForm.username || '').trim(),
          password: this.configForm.password || '',
          charset: this.configForm.charset || '',
          timeout: this.configForm.timeout || 10,
          read_timeout: this.configForm.read_timeout || 30,
          write_timeout: this.configForm.write_timeout || 30,
          db_file: this.configForm.type === 'sqlite' ? (this.configForm.db_file || '').trim() : '',
          connection_type: this.configForm.connection_type || '',
          ssh_addr: '',
          ssh_port: '',
          ssh_username: '',
          ssh_password: ''
        };

        // 如果连接方式是SSH，设置SSH配置
        if (configToSave.connection_type === 'ssh') {
          configToSave.ssh_addr = (this.configForm.ssh_addr || '').trim();
          configToSave.ssh_port = (this.configForm.ssh_port || '22').trim();
          configToSave.ssh_username = (this.configForm.ssh_username || '').trim();
          configToSave.ssh_password = this.configForm.ssh_password || '';
        }

        // 规范化 connection_type 的值
        if (configToSave.connection_type === null || configToSave.connection_type === undefined) {
          configToSave.connection_type = '';
        }

        // 如果是更新操作，确保ID存在
        if (this.configForm.id && !configToSave.id) {
          configToSave.id = this.configForm.id;
        }

        console.log('保存配置，完整数据:', configToSave);
        const method = this.configForm.id ? 'put' : 'post';
        const response = await this.apiClient[method]('/api/config', configToSave);

        if (response.data.code === 0) {
          this.$message.success(this.configForm.id ? '更新成功' : '创建成功');
          this.showConfigDialog = false;
          this.testResult = null;
          this.configForm = {
            id: null,
            type: 'mysql',
            name: '',
            host: '',
            port: '',
            username: '',
            password: '',
            db_file: '',
            connection_type: '',
            ssh_host_id: null,
            ssh_addr: '',
            ssh_port: '',
            ssh_username: '',
            ssh_password: ''
          };
          await this.loadConfigs();
        } else {
          this.$message.error(response.data.msg || '保存失败');
        }
      } catch (e) {
        const errorMsg = e.response?.data?.msg || e.message || '保存失败';
        this.$message.error('保存失败: ' + errorMsg);
        console.error('保存配置失败:', e);
      }
    },
    editConfig(config) {
      // 深拷贝配置，避免直接修改原对象
      this.configForm = {
        id: config.id || null,
        type: config.type || 'mysql',
        name: config.name || '',
        host: config.host || '',
        port: config.port || '',
        username: config.username || '',
        password: config.password || '', // 密码保持原样（后端已解码）
        charset: config.charset || '',
        timeout: config.timeout || 10,
        read_timeout: config.read_timeout || 30,
        write_timeout: config.write_timeout || 30,
        db_file: config.db_file || '',
        connection_type: config.connection_type || '',
        ssh_host_id: null, // 前端字段，需要根据SSH配置查找
        ssh_addr: config.ssh_addr || '',
        ssh_port: config.ssh_port || '',
        ssh_username: config.ssh_username || '',
        ssh_password: config.ssh_password || '' // SSH密码保持原样（后端已解码）
      };

      // 规范化 connection_type 的值
      if (this.configForm.connection_type === null || this.configForm.connection_type === undefined) {
        this.configForm.connection_type = '';
      }

      // 如果存在SSH配置，尝试从SSH主机列表中找到对应的ID
      if (this.configForm.connection_type === 'ssh' && this.configForm.ssh_addr) {
        const matchedHost = this.sshHostList.find(host =>
          host.address === this.configForm.ssh_addr &&
          String(host.port) === this.configForm.ssh_port &&
          host.user === this.configForm.ssh_username
        );
        if (matchedHost) {
          this.configForm.ssh_host_id = matchedHost.id;
        }
      }

      this.testResult = null;
      this.showConfigDialog = true;
    },
    async handleConfigDialogOpen() {
      // 加载SSH主机列表（编辑模式下需要用于匹配SSH主机ID）
      await this.loadSshHostList();

      // 对话框打开时，如果是新建模式，自动填充默认值
      if (!this.configForm.id && this.configForm.type) {
        this.$nextTick(() => {
          this.handleTypeChange(this.configForm.type);
        });
      } else if (this.configForm.id) {
        // 编辑模式下，如果存在SSH配置，尝试从SSH主机列表中找到对应的ID
        if (this.configForm.connection_type === 'ssh' && this.configForm.ssh_addr) {
          this.$nextTick(() => {
            const matchedHost = this.sshHostList.find(host =>
              host.address === this.configForm.ssh_addr &&
              String(host.port) === this.configForm.ssh_port &&
              host.user === this.configForm.ssh_username
            );
            if (matchedHost) {
              this.configForm.ssh_host_id = matchedHost.id;
            }
          });
        }
      }

      // 规范化 connection_type 的值，确保 null 或 undefined 转换为空字符串
      // 这样在 el-select 中会正确显示为空（因为 clearable 属性）
      if (this.configForm.connection_type === null || this.configForm.connection_type === undefined) {
        this.configForm.connection_type = '';
      }
    },
    // 处理填充连接表单事件
    handleFillConnectForm(event) {
      const { module, connectionInfo } = event.detail || {};
      // 处理数据库模块的请求（mysql, postgresql, mongodb, mssql, oracle）
      const dbTypes = ['mysql', 'postgresql', 'mongodb', 'mssql', 'oracle'];
      if (!dbTypes.includes(module) || !connectionInfo) {
        return;
      }

      // 先填充表单数据（在打开对话框之前）
      // 设置数据库类型（需要映射mssql到sqlserver）
      const dbType = module === 'mssql' ? 'sqlserver' : module;
      this.configForm.type = dbType;

      if (connectionInfo.host) {
        this.configForm.host = connectionInfo.host;
      }
      if (connectionInfo.port) {
        this.configForm.port = connectionInfo.port;
      }
      // 确保账号密码能够正确填充，即使为空也要设置
      if (connectionInfo.username !== undefined) {
        this.configForm.username = connectionInfo.username;
      }
      if (connectionInfo.password !== undefined) {
        this.configForm.password = connectionInfo.password;
      }
      if (connectionInfo.name) {
        this.configForm.name = connectionInfo.name;
      }

      // 根据数据库类型自动填充默认端口和用户名（如果未设置）
      // 注意：这里需要临时设置id为null，让handleTypeChange认为这是新建模式
      const originalId = this.configForm.id;
      this.configForm.id = null;
      this.handleTypeChange(dbType);
      this.configForm.id = originalId;

      // 打开对话框
      this.showConfigDialog = true;

      // 等待对话框完全打开后，再次填充数据（防止watch或其他逻辑覆盖）
      this.$nextTick(() => {
        // 加载SSH主机列表
        this.loadSshHostList();

        // 再次填充数据，确保不被默认值覆盖
        setTimeout(() => {
          if (connectionInfo.host) {
            this.configForm.host = connectionInfo.host;
          }
          if (connectionInfo.port) {
            this.configForm.port = connectionInfo.port;
          }
          // 重要：账号密码需要明确设置，即使为空值也要保留
          if (connectionInfo.username !== undefined) {
            this.configForm.username = connectionInfo.username;
          }
          if (connectionInfo.password !== undefined) {
            this.configForm.password = connectionInfo.password;
          }
          if (connectionInfo.name) {
            this.configForm.name = connectionInfo.name;
          }
        }, 150);
      });
    },
    handleConfigDialogClose() {
      // 关闭对话框时清除测试结果和表单
      this.testResult = null;
      this.configForm = {
        id: null,
        type: 'mysql',
        name: '',
        host: '',
        port: '',
        username: '',
        password: '',
        db_file: '',
        connection_type: '',
        ssh_host_id: null,
        ssh_addr: '',
        ssh_port: '',
        ssh_username: '',
        ssh_password: ''
      };
    },
    handleTypeChange(type) {
      // 根据数据库类型自动填充默认端口和用户名（新建与编辑均生效：仅当端口/用户名为空或为其它类型的默认值时才替换）
      const defaults = {
        mysql: { port: '3306', username: 'root' },
        doris_legacy: { port: '9030', username: 'root' },
        postgresql: { port: '5432', username: 'postgres' },
        oracle: { port: '1521', username: 'system' },
        sqlserver: { port: '1433', username: 'sa' },
        mariadb: { port: '3306', username: 'root' },
        dm: { port: '5236', username: 'SYSDBA' },
        kingbase: { port: '54321', username: 'SYSTEM' },
        gbase: { port: '5258', username: 'root' }
      };

      const defaultValues = defaults[type];
      if (defaultValues) {
        // 获取所有默认端口值和用户名，用于检测是否需要更新
        const allDefaultPorts = Object.values(defaults).map(d => d.port);
        const allDefaultUsernames = Object.values(defaults).map(d => d.username);

        // 如果端口为空，或者是其他类型的默认端口，则更新为新类型的默认端口
        if (!this.configForm.port || allDefaultPorts.includes(this.configForm.port)) {
          this.configForm.port = defaultValues.port;
        }
        // 如果用户名为空，或者是其他类型的默认用户名，则更新为新类型的默认用户名
        if (!this.configForm.username || allDefaultUsernames.includes(this.configForm.username)) {
          this.configForm.username = defaultValues.username;
        }
      }
    },
    handleConnectionTypeChange(connectionType) {
      // 当连接方式改变时，如果是直连，清空SSH配置
      if (connectionType !== 'ssh') {
        this.configForm.ssh_host_id = null;
        this.configForm.ssh_addr = '';
        this.configForm.ssh_port = '';
        this.configForm.ssh_username = '';
        this.configForm.ssh_password = '';
      }
    },
    handleSshHostChange(hostId) {
      // 当选择SSH主机时，自动填充SSH配置
      // hostId可能是数字（从列表选择）或字符串（手动输入）
      if (hostId) {
        // 尝试转换为数字，如果是数字则从列表查找
        const numericId = typeof hostId === 'string' ? parseInt(hostId, 10) : hostId;
        if (!isNaN(numericId) && typeof numericId === 'number') {
          const host = this.sshHostList.find(h => h.id === numericId);
          if (host) {
            this.configForm.ssh_addr = host.address || '';
            this.configForm.ssh_port = String(host.port || 22);
            this.configForm.ssh_username = host.user || '';
            this.configForm.ssh_password = host.pwd || '';
            return;
          }
        }
        // 如果是手动输入的字符串（不是有效的ID），不清空字段，让用户继续手动输入
        // 这种情况下，用户应该直接在SSH地址等字段中输入
      } else {
        // 清空选择时，不清空已输入的SSH配置，让用户保留手动输入的内容
      }
    },
    async deleteConfig(configId) {
      if (!this.apiClient) {
        this.$message.error('API客户端未初始化');
        return;
      }
      try {
        await confirmDangerousAction({
          title: '删除连接',
          message: '将删除此数据库连接配置，不可恢复。',
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      try {
        await this.apiClient.delete(`/api/config/${configId}`);
        this.$message.success('删除成功');
        if (this.activeConfigId === String(configId)) {
          this.activeConfigId = null;
          this.currentConfig = null;
          this.selectedTable = null;
          this.tableData = [];
          this.tableColumns = [];
          this.tablePagination = {
            currentPage: 1,
            pageSize: 50,
            total: 0
          };
        }
        await this.loadConfigs();
      } catch (e) {
        this.$message.error('删除失败: ' + (e.response?.data?.msg || e.message));
      }
    },
    getTypeTagType(type) {
      const typeMap = {
        mysql: 'success',
        doris_legacy: 'warning',
        postgresql: 'warning',
        sqlite: 'info',
        oracle: 'danger',
        sqlserver: 'success',
        mariadb: 'success',
        dm: 'warning',
        kingbase: 'warning',
        gbase: 'success'
      };
      return typeMap[type] || 'info';
    },
    // SQL 模板接口按 mysql 存储；Doris 旧版兼容与 MySQL 语法一致，请求模板时映射为 mysql
    sqlTemplateDbType(configType) {
      if (!configType) return 'mysql';
      const t = String(configType).toLowerCase();
      return t === 'doris_legacy' ? 'mysql' : t;
    },
    getConfigHostDisplay(config) {
      if (!config) return '';
      // SQLite 显示数据库文件路径，其它类型显示 host:port
      if (config.type === 'sqlite') {
        return config.db_file || '';
      }
      const host = config.host || '';
      const port = config.port || '';
      if (host && port) {
        return `${host}:${port}`;
      }
      return host || port || '';
    },
    async handleConfigSelect(configId) {
      const config = this.configs.find(c => c.id === parseInt(configId));
      if (!config) return;

      // 检查是否已存在该连接的标签页
      const existingTab = this.tabs.find(tab => tab.configId === String(configId));
      if (existingTab) {
        // 切换到已存在的标签页
        this.activeConfigId = String(configId);
        this.activeTabId = existingTab.id;
        this.showConfigSelectDialog = false;
        return;
      }

      // 先测试连接，连接失败则不创建标签页
      if (!this.apiClient) {
        this.$message.error('API客户端未初始化');
        return;
      }

      try {
        // 测试连接
        const pingResponse = await this.apiClient.post('/api/ping', config);
        if (pingResponse.data.code !== 0) {
          // 连接失败，不创建标签页，直接提示错误
          this.$message.error('连接失败: ' + (pingResponse.data.msg || '未知错误'));
          this.showConfigSelectDialog = false;
          return;
        }
      } catch (e) {
        // 连接失败，不创建标签页，直接提示错误
        const errorMsg = e.response?.data?.msg || e.message || '连接失败';
        this.$message.error('连接失败: ' + errorMsg);
        this.showConfigSelectDialog = false;
        return;
      }

      // 连接成功，创建新标签页
      this.activeConfigId = String(configId);
      const tabId = `tab_${++this.tabIdCounter}`;
      const newTab = {
        id: tabId,
        configId: String(configId),
        label: config.name,
        config: { ...config },
        activeTab: 'databases',
        databaseTree: [],
        selectedTable: null,
        tableDataTab: 'data',
        tableData: [],
        tableColumns: [],
        tableDataLoading: false,
        currentDatabase: null,
        selectedRows: [],
        editingRows: new Map(),
        originalRowData: new Map(),
        tablePagination: {
          currentPage: 1,
          pageSize: 50,
          total: 0
        },
        querySql: '',
        queryResult: null,
        queryError: null,
        queryLoading: false,
        querySelectedDatabase: null,
        querySelectedTable: null,
        queryTableList: []
      };

      this.tabs.push(newTab);
      this.activeTabId = tabId;
      this.showConfigSelectDialog = false;
      await this.refreshDatabases(tabId);
      // 加载 SQL 模板到下拉框
      await this.loadSqlTemplatesForDropdown(tabId);
    },
    handleConfigSelectFromDialog(configId) {
      this.handleConfigSelect(configId);
    },
    getTab(tabId) {
      return this.tabs.find(t => t.id === tabId);
    },
    setTabActiveTab(tabId, activeTab) {
      const tab = this.getTab(tabId);
      if (tab) {
        tab.activeTab = activeTab;
      }
    },
    async refreshDatabases(tabId) {
      const tab = this.getTab(tabId);
      if (!tab || !this.apiClient) return;

      try {
        const response = await this.apiClient.post('/api/databases', {
          cid: tab.config.id
        });

        if (response.data.code === 0) {
          const databases = response.data.data || [];
          tab.databaseTree = databases.map(db => ({
            name: db,
            children: []
          }));
        } else {
          // 加载数据库列表失败，提示错误但不关闭标签页（因为连接测试已通过）
          this.$message.warning('加载数据库列表失败: ' + (response.data.msg || '未知错误'));
          tab.databaseTree = [];
        }
      } catch (e) {
        // 加载数据库列表失败，提示错误但不关闭标签页（因为连接测试已通过）
        const errorMsg = e.response?.data?.msg || e.message || '加载失败';
        this.$message.warning('加载数据库列表失败: ' + errorMsg);
        tab.databaseTree = [];
      }
    },
    async handleNodeClick(node, tabId) {
      const tab = this.getTab(tabId);
      if (!tab || !this.apiClient) return;

      if (node.children && node.children.length === 0) {
        // 加载表
        try {
          const response = await this.apiClient.post('/api/tables', {
            cid: tab.config.id,
            database: node.name
          });

          if (response.data.code === 0) {
            const tables = response.data.data || [];
            if (tables.length === 0) {
              // 空数据库，显示提示信息
              node.children = [{
                name: '（无数据表）',
                type: 'empty',
                database: node.name,
                isPlaceholder: true
              }];
            } else {
              node.children = tables.map(table => ({
                name: table.name,
                type: 'table',
                database: node.name
              }));
            }
          } else {
            this.$message.warning('加载表失败: ' + (response.data.msg || '未知错误'));
          }
        } catch (e) {
          this.$message.error('加载表失败: ' + (e.response?.data?.msg || e.message));
        }
      } else if (node.type === 'table' && node.database && !node.isPlaceholder) {
        // 点击表，加载表数据和字段信息（排除占位符节点）
        tab.selectedTable = {
          table: node.name,
          database: node.database
        };
        tab.currentDatabase = node.database;
        tab.tablePagination = {
          currentPage: 1,
          pageSize: 50,
          total: 0
        };

        // 并行加载表数据和字段信息，即使一个失败也不影响另一个
        const [columnsResult, dataResult] = await Promise.allSettled([
          this.loadTableColumns(tabId),
          this.loadTableData(tabId)
        ]);

      }
    },
    async loadTableColumns(tabId) {
      const tab = this.getTab(tabId);
      if (!tab || !tab.selectedTable || !this.apiClient) return;

      tab.tableDataLoading = true;
      try {
        const response = await this.apiClient.post('/api/columns', {
          cid: tab.config.id,
          database: tab.selectedTable.database,
          table: tab.selectedTable.table
        });

        if (response.data.code === 0) {
          tab.tableColumns = response.data.data || [];
        } else {
          this.$message.error('加载字段失败: ' + (response.data.msg || 'Unknown error'));
        }
      } catch (e) {
        this.$message.error('加载字段失败: ' + (e.response?.data?.msg || e.message));
      } finally {
        tab.tableDataLoading = false;
      }
    },
    getTableNameQuote(config) {
      // 根据数据库类型返回合适的引号
      if (!config) return '`';
      const type = config.type?.toLowerCase();
      if (type === 'postgresql' || type === 'oracle') {
        return '"';
      }
      if (type === 'sqlserver') {
        return ''; // SQL Server使用方括号，不需要引号
      }
      return '`';
    },
    // 转义标识符中的特殊字符
    escapeIdentifier(identifier, quote) {
      if (!identifier) return '';
      // 对于反引号（MySQL/MariaDB），需要转义为两个反引号
      if (quote === '`') {
        return identifier.replace(/`/g, '``');
      }
      // 对于双引号（PostgreSQL/Oracle），需要转义为两个双引号
      if (quote === '"') {
        return identifier.replace(/"/g, '""');
      }
      // SQL Server使用方括号，需要转义右方括号
      if (quote === '') {
        return identifier.replace(/\]/g, ']]');
      }
      return identifier;
    },
    // 构建完整的表名（包含数据库名和表名）
    buildTableName(config, database, table) {
      if (!database || !table) return '';
      const quote = this.getTableNameQuote(config);
      const dbType = config?.type?.toLowerCase();
      const tableNameEscaped = this.escapeIdentifier(table, quote);

      if (dbType === 'sqlserver') {
        // SQL Server 需要三部分名称：[database].[schema].[table]
        // 即使后端执行了 USE [database]，在跨数据库查询时仍需要完全限定名
        // 为了兼容性，使用完全限定名：[database].[dbo].[table]
        const dbName = this.escapeIdentifier(database, '');
        return `[${dbName}].[dbo].[${tableNameEscaped}]`;
      } else if (dbType === 'postgresql' || dbType === 'kingbase') {
        // PostgreSQL 和 Kingbase 使用 schema.table 格式，默认 schema 是 public
        // 不需要包含数据库名，因为数据库在连接时已经指定
        return `${quote}public${quote}.${quote}${tableNameEscaped}${quote}`;
      } else {
        // MySQL, MariaDB, SQLite 等使用 database.table 格式
        const dbName = this.escapeIdentifier(database, quote);
        return `${quote}${dbName}${quote}.${quote}${tableNameEscaped}${quote}`;
      }
    },
    // 构建列名（带引号）
    buildColumnName(config, columnName) {
      if (!columnName) return '';
      const quote = this.getTableNameQuote(config);
      const dbType = config?.type?.toLowerCase();
      const escapedName = this.escapeIdentifier(columnName, quote);

      if (dbType === 'sqlserver') {
        return `[${escapedName}]`;
      } else {
        return `${quote}${escapedName}${quote}`;
      }
    },
    async loadTableData(tabId) {
      const tab = this.getTab(tabId);
      if (!tab || !tab.selectedTable || !this.apiClient) return;

      // 防止重复调用：如果正在加载，直接返回
      if (tab._loadingTableData) {
        return;
      }

      // 标记正在加载
      tab._loadingTableData = true;

      // 立即显示 loading（不再延迟，避免闪烁）
      tab.tableDataLoading = true;

      try {
        // 构建表名
        const tableName = this.buildTableName(tab.config, tab.selectedTable.database, tab.selectedTable.table);
        const dbType = tab.config?.type?.toLowerCase();

        // 先获取总数，所有数据库都支持标准的 COUNT(*) 语法
        const countSql = `SELECT COUNT(*) as total FROM ${tableName}`;

        let totalCount = 0;
        try {
          const countResponse = await this.apiClient.post('/api/query', {
            cid: tab.config.id,
            sql: countSql,
            database: tab.selectedTable.database
          });

          if (countResponse.data.code === 0) {
            const countResult = countResponse.data.data?.rows?.[0];
            // 确保 total 是数字类型
            totalCount = Number(countResult?.total) || 0;
            tab.tablePagination.total = totalCount;
          } else {
            console.warn('获取总数失败:', countResponse.data.msg);
            // 如果获取总数失败，仍然尝试加载数据
          }
        } catch (countError) {
          console.warn('获取总数异常:', countError);
          // 如果获取总数失败，仍然尝试加载数据
        }

        // 如果总数为0，直接返回空数据
        if (totalCount === 0) {
          tab.tableData = [];
          tab.tablePagination.total = 0;
          tab.tableDataLoading = false;
          tab._loadingTableData = false;
          return;
        }

        // 确保当前页不超过总页数
        const totalPages = Math.ceil(totalCount / tab.tablePagination.pageSize) || 1;
        if (tab.tablePagination.currentPage > totalPages) {
          tab.tablePagination.currentPage = totalPages;
        }

        // 获取分页数据
        const offset = (tab.tablePagination.currentPage - 1) * tab.tablePagination.pageSize;
        // 根据数据库类型构建不同的分页SQL
        let sql = '';
        if (dbType === 'oracle') {
          // Oracle使用ROWNUM或FETCH语法
          const startRow = offset + 1;
          const endRow = offset + tab.tablePagination.pageSize;
          sql = `SELECT * FROM (SELECT a.*, ROWNUM rnum FROM (SELECT * FROM ${tableName}) a WHERE ROWNUM <= ${endRow}) WHERE rnum >= ${startRow}`;
        } else if (dbType === 'sqlserver') {
          // SQL Server使用TOP和OFFSET FETCH语法
          sql = `SELECT * FROM ${tableName} ORDER BY (SELECT NULL) OFFSET ${offset} ROWS FETCH NEXT ${tab.tablePagination.pageSize} ROWS ONLY`;
        } else {
          // MySQL, PostgreSQL, SQLite 使用 LIMIT OFFSET
          sql = `SELECT * FROM ${tableName} LIMIT ${tab.tablePagination.pageSize} OFFSET ${offset}`;
        }

        const response = await this.apiClient.post('/api/query', {
          cid: tab.config.id,
          sql: sql,
          database: tab.selectedTable.database
        });

        if (response.data.code === 0) {
          const rows = response.data.data?.rows || [];
          // 为每行添加唯一标识
          const newData = rows.map((row, index) => ({
            ...row,
            _rowId: `row_${Date.now()}_${index}`,
            _editing: false
          }));

          // 直接更新数据（Vue 会处理响应式更新）
          tab.tableData = newData;

          // 等待 DOM 更新完成后再关闭 loading
          await this.$nextTick();
          tab.tableDataLoading = false;
        } else {
          this.$message.error(response.data.msg || '加载数据失败');
          tab.tableDataLoading = false;
        }
      } catch (e) {
        this.$message.error('加载数据失败: ' + (e.response?.data?.msg || e.message));
        tab.tableDataLoading = false;
      } finally {
        // 清除加载标记
        tab._loadingTableData = false;
      }
    },
    handlePageChange(page, tabId) {
      const tab = this.getTab(tabId);
      if (tab) {
        // 先更新分页状态
        tab.tablePagination.currentPage = page;
        // 然后加载数据（loadTableData 内部会设置 loading 状态）
        this.loadTableData(tabId);
      }
    },
    handlePageSizeChange(size, tabId) {
      const tab = this.getTab(tabId);
      if (tab) {
        tab.tablePagination.pageSize = size;
        tab.tablePagination.currentPage = 1;
        this.loadTableData(tabId);
      }
    },
    getRowKey(row) {
      return row._rowId || JSON.stringify(row);
    },
    formatCellValue(value) {
      if (value === null || value === undefined) {
        return '';
      }
      if (typeof value === 'object') {
        return JSON.stringify(value);
      }
      return String(value);
    },
    handleSelectionChange(selection, tabId) {
      const tab = this.getTab(tabId);
      if (tab) {
        tab.selectedRows = selection;
      }
    },
    handleEditRow(row, tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return;

      // 保存原始数据
      tab.originalRowData.set(row._rowId, JSON.parse(JSON.stringify(row)));
      row._editing = true;
      tab.editingRows.set(row._rowId, row);
    },
    handleCancelEditRow(row, tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return;

      // 如果是新行，直接删除
      if (row._isNew) {
        const index = tab.tableData.findIndex(r => r._rowId === row._rowId);
        if (index > -1) {
          tab.tableData.splice(index, 1);
        }
        tab.editingRows.delete(row._rowId);
        tab.originalRowData.delete(row._rowId);
        return;
      }

      // 恢复原始数据
      const original = tab.originalRowData.get(row._rowId);
      if (original) {
        Object.assign(row, original);
      }
      row._editing = false;
      tab.editingRows.delete(row._rowId);
      tab.originalRowData.delete(row._rowId);
    },
    escapeSqlValue(value) {
      if (value === null || value === undefined) {
        return 'NULL';
      }

      // 处理布尔值
      if (typeof value === 'boolean') {
        return value ? '1' : '0';
      }

      // 处理数字
      if (typeof value === 'number') {
        // 检查是否为NaN或Infinity
        if (!isFinite(value)) {
          return 'NULL';
        }
        return String(value);
      }

      // 处理字符串
      if (typeof value === 'string') {
        // 转义单引号（SQL标准转义方式）
        // 同时转义反斜杠（某些数据库需要）
        let escaped = value.replace(/\\/g, '\\\\').replace(/'/g, "''");
        return `'${escaped}'`;
      }

      // 处理日期对象
      if (value instanceof Date) {
        // 格式化为ISO字符串
        return `'${value.toISOString()}'`;
      }

      // 处理对象和数组（转换为JSON字符串）
      if (typeof value === 'object') {
        try {
          const jsonStr = JSON.stringify(value);
          return `'${jsonStr.replace(/\\/g, '\\\\').replace(/'/g, "''")}'`;
        } catch (e) {
          console.warn('无法序列化对象:', value, e);
          return 'NULL';
        }
      }

      // 其他类型转换为字符串并转义
      const strValue = String(value);
      return `'${strValue.replace(/\\/g, '\\\\').replace(/'/g, "''")}'`;
    },
    async handleSaveRow(row, tabId) {
      const tab = this.getTab(tabId);
      if (!tab || !tab.selectedTable || !this.apiClient) return;

      try {
        if (row._isNew) {
          await this.handleSaveNewRow(row, tabId);
          return;
        }

        // 获取原始数据，用于确定WHERE条件的主键值
        const originalRow = tab.originalRowData.get(row._rowId);
        if (!originalRow) {
          this.$message.error('无法获取原始数据，无法更新');
          return;
        }

        // 获取所有主键字段（支持复合主键）
        const primaryKeys = tab.tableColumns.filter(col => {
          const key = col.key || '';
          return key === 'PRI' || key === 'primary' || key === 'PRIMARY';
        });

        // 如果没有主键，尝试使用第一个字段作为唯一标识
        if (primaryKeys.length === 0) {
          if (tab.tableColumns.length === 0) {
            this.$message.error('无法确定主键，无法更新');
            return;
          }
          primaryKeys.push(tab.tableColumns[0]);
        }

        // 验证所有主键字段都有原始值
        const primaryKeyConditions = [];
        for (const pk of primaryKeys) {
          const pkName = pk.field || pk.name;
          if (!pkName) {
            this.$message.error('无法确定主键字段名，无法更新');
            return;
          }

          const originalPkValue = originalRow[pkName];
          if (originalPkValue === undefined || originalPkValue === null) {
            this.$message.error(`主键字段 ${pkName} 的原始值不能为空，无法更新`);
            return;
          }

          const pkQuoted = this.buildColumnName(tab.config, pkName);
          const pkValue = this.escapeSqlValue(originalPkValue);
          primaryKeyConditions.push(`${pkQuoted} = ${pkValue}`);
        }

        const tableName = this.buildTableName(tab.config, tab.selectedTable.database, tab.selectedTable.table);
        const setParts = [];
        const primaryKeyNames = new Set(primaryKeys.map(pk => pk.field || pk.name));

        // 只更新有变化的字段
        for (const col of tab.tableColumns) {
          const colName = col.field || col.name;
          if (!colName) {
            continue;
          }

          const currentValue = row[colName];
          const originalValue = originalRow[colName];

          // 比较值是否变化（考虑null和undefined的情况）
          const valueChanged = this.isValueChanged(currentValue, originalValue);

          if (!valueChanged) {
            // 值没有变化，跳过
            continue;
          }

          // 构建SET子句
          const value = this.escapeSqlValue(currentValue);
          const colNameQuoted = this.buildColumnName(tab.config, colName);
          setParts.push(`${colNameQuoted} = ${value}`);
        }

        if (setParts.length === 0) {
          this.$message.warning('没有可更新的字段（所有字段值都未变化）');
          return;
        }

        // 构建WHERE条件（支持复合主键）
        const whereClause = primaryKeyConditions.join(' AND ');
        const sql = `UPDATE ${tableName} SET ${setParts.join(', ')} WHERE ${whereClause}`;

        const response = await this.apiClient.post('/api/execute', {
          cid: tab.config.id,
          sql: sql,
          database: tab.selectedTable.database
        });

        if (response.data.code === 0) {
          this.$message.success('保存成功');
          row._editing = false;
          tab.editingRows.delete(row._rowId);
          tab.originalRowData.delete(row._rowId);
          // 重新加载数据，保持当前分页
          const currentPage = tab.tablePagination.currentPage;
          await this.loadTableData(tabId);
          // 恢复分页，确保能看到刚才编辑的数据
          tab.tablePagination.currentPage = currentPage;
        } else {
          this.$message.error(response.data.msg || '保存失败');
        }
      } catch (e) {
        this.$message.error('保存失败: ' + (e.response?.data?.msg || e.message));
        console.error('保存行失败:', e);
      }
    },
    // 比较两个值是否发生变化
    isValueChanged(currentValue, originalValue) {
      // 处理null和undefined
      if (currentValue === null || currentValue === undefined) {
        return originalValue !== null && originalValue !== undefined;
      }
      if (originalValue === null || originalValue === undefined) {
        return true;
      }

      // 对于字符串，去除首尾空格后比较
      if (typeof currentValue === 'string' && typeof originalValue === 'string') {
        return currentValue.trim() !== originalValue.trim();
      }

      // 对于数字，严格比较
      if (typeof currentValue === 'number' && typeof originalValue === 'number') {
        return currentValue !== originalValue;
      }

      // 其他类型转换为字符串比较
      return String(currentValue) !== String(originalValue);
    },
    handleCellBlur(row, colName) {
      // 可以在这里添加验证逻辑
    },
    handleAddRow(tabId) {
      const tab = this.getTab(tabId);
      if (!tab || !tab.selectedTable) {
        this.$message.warning('请先选择一个数据表');
        return;
      }

      const newRow = {
        _rowId: `new_${Date.now()}`,
        _editing: true,
        _isNew: true
      };

      // 为每个列设置默认值
      for (const col of tab.tableColumns) {
        const colName = col.field || col.name;
        if (!colName) continue;

        const isSequenceDef = this.isSequenceDefault(col.default);

        if (col.default !== null && col.default !== undefined) {
          // 如果是序列函数，不设置值（留空），让数据库在保存时使用默认值
          // 如果不是序列函数，设置默认值字符串
          if (!isSequenceDef) {
            newRow[colName] = col.default;
          } else {
            // 序列函数留空，用户不需要看到 nextval(...) 这样的字符串
            // 但我们需要保存原始默认值，以便在保存时判断是否使用了默认值
            newRow[colName] = '';
            // 在列信息中保存原始默认值，用于后续判断
            if (!col._originalDefault) {
              col._originalDefault = col.default;
            }
          }
        } else if (col.nullable === 'YES' || col.nullable === true) {
          newRow[colName] = null;
        } else {
          newRow[colName] = '';
        }
      }

      tab.tableData.unshift(newRow);
      tab.editingRows.set(newRow._rowId, newRow);
    },
    // 检查默认值是否是序列函数（PostgreSQL/SQL Server等）
    isSequenceDefault(defaultValue) {
      if (!defaultValue || typeof defaultValue !== 'string') {
        return false;
      }
      const normalized = String(defaultValue).toLowerCase().trim();
      // 检测常见的序列函数模式
      return normalized.includes('nextval') ||
        normalized.includes('sequence') ||
        normalized.includes('gen_random_uuid') ||
        normalized.includes('uuid_generate') ||
        /nextval\s*\(/i.test(normalized); // 匹配 nextval( 模式
    },
    // 检查值是否应该使用默认值（对于序列函数，空值应该使用默认值）
    shouldUseDefault(value, defaultValue) {
      const isSequenceDef = this.isSequenceDefault(defaultValue);
      if (isSequenceDef) {
        // 如果是序列函数，且值为空（空字符串、null、undefined），则使用默认值
        return !value || value === '' || value === null || value === undefined;
      }
      // 对于非序列默认值，只有值确实等于默认值时才使用默认值
      if (value === undefined || value === null) return false;
      if (defaultValue === null || defaultValue === undefined) return false;
      return String(value).trim() === String(defaultValue).trim();
    },
    async handleSaveNewRow(row, tabId) {
      const tab = this.getTab(tabId);
      if (!tab || !tab.selectedTable || !this.apiClient) return;

      try {
        const tableName = this.buildTableName(tab.config, tab.selectedTable.database, tab.selectedTable.table);
        const columns = [];
        const values = [];
        const missingRequiredFields = [];
        const dbType = tab.config?.type?.toLowerCase() || '';

        for (const col of tab.tableColumns) {
          const colName = col.field || col.name;
          if (!colName) continue;

          // 跳过 IDENTITY/AUTO_INCREMENT 列（SQL Server IDENTITY, MySQL AUTO_INCREMENT 等）
          // 这些列由数据库自动生成，不应该在 INSERT 语句中包含
          if (col.auto_increment === true || col.auto_increment === 'true' || col.auto_increment === 1) {
            continue;
          }

          const value = row[colName];
          const isNullable = col.nullable === 'YES' || col.nullable === true;
          const hasDefault = col.default !== null && col.default !== undefined && col.default !== '';
          const isSequenceDef = this.isSequenceDefault(col.default);

          // 如果应该使用默认值（序列函数且值为空，或值与默认值相同），跳过该字段
          if (hasDefault && this.shouldUseDefault(value, col.default)) {
            continue;
          }

          // 检查必填字段
          if (!isNullable && !hasDefault) {
            if (value === undefined || value === null || value === '') {
              missingRequiredFields.push(colName);
              continue;
            }
          }

          // 如果字段有值（非空），包含在INSERT中
          if (value !== undefined && value !== null && value !== '') {
            columns.push(this.buildColumnName(tab.config, colName));
            values.push(this.escapeSqlValue(value));
          } else if (isNullable) {
            // 可空字段且没有值，使用NULL
            columns.push(this.buildColumnName(tab.config, colName));
            values.push('NULL');
          } else if (hasDefault) {
            // 不可空但有默认值，不包含在INSERT中（已经在上面处理了shouldUseDefault的情况）
            // 这种情况不应该到达，但为了安全起见跳过
            continue;
          } else {
            // 不可空且无默认值的字段，如果值为空，已经在上面检查了
            // 这里不应该到达，但为了安全起见
            missingRequiredFields.push(colName);
          }
        }

        // 检查必填字段
        if (missingRequiredFields.length > 0) {
          this.$message.warning(`以下字段不能为空: ${missingRequiredFields.join(', ')}`);
          return;
        }

        if (columns.length === 0) {
          this.$message.warning('至少需要填写一个字段');
          return;
        }

        const sql = `INSERT INTO ${tableName} (${columns.join(', ')}) VALUES (${values.join(', ')})`;

        const response = await this.apiClient.post('/api/execute', {
          cid: tab.config.id,
          sql: sql,
          database: tab.selectedTable.database
        });

        if (response.data.code === 0) {
          this.$message.success('新增成功');
          row._editing = false;
          row._isNew = false;
          tab.editingRows.delete(row._rowId);
          await this.loadTableData(tabId);
        } else {
          this.$message.error(response.data.msg || '新增失败');
        }
      } catch (e) {
        this.$message.error('新增失败: ' + (e.response?.data?.msg || e.message));
      }
    },
    async handleDeleteRow(row, tabId, skipRowDeleteConfirm = false) {
      const tab = this.getTab(tabId);
      if (!tab || !tab.selectedTable || !this.apiClient) return;

      // 如果是新行（未保存），直接从数组中删除
      if (row._isNew) {
        const index = tab.tableData.findIndex(r => r._rowId === row._rowId);
        if (index > -1) {
          tab.tableData.splice(index, 1);
          tab.editingRows.delete(row._rowId);
          tab.originalRowData.delete(row._rowId);
        }
        return;
      }

      if (!skipRowDeleteConfirm) {
        try {
          await confirmDangerousAction({
            title: '删除行',
            message: `将删除表 ${tab.selectedTable.database}.${tab.selectedTable.table} 中的该行数据，不可恢复。`,
            expectedInput: DANGER_DELETE_TOKEN
          });
        } catch {
          return;
        }
      }

      try {
        // 获取所有主键字段（支持复合主键）
        const primaryKeys = tab.tableColumns.filter(col => {
          const key = col.key || '';
          return key === 'PRI' || key === 'primary' || key === 'PRIMARY';
        });

        // 如果没有主键，尝试使用第一个字段作为唯一标识
        if (primaryKeys.length === 0) {
          if (tab.tableColumns.length === 0) {
            this.$message.error('无法确定主键，无法删除');
            return;
          }
          primaryKeys.push(tab.tableColumns[0]);
        }

        // 构建WHERE条件（支持复合主键）
        const primaryKeyConditions = [];
        for (const pk of primaryKeys) {
          const pkName = pk.field || pk.name;
          if (!pkName) {
            this.$message.error('无法确定主键字段名，无法删除');
            return;
          }

          const pkValue = row[pkName];
          if (pkValue === undefined || pkValue === null) {
            this.$message.error(`主键字段 ${pkName} 的值不能为空，无法删除`);
            return;
          }

          const pkQuoted = this.buildColumnName(tab.config, pkName);
          const pkValueEscaped = this.escapeSqlValue(pkValue);
          primaryKeyConditions.push(`${pkQuoted} = ${pkValueEscaped}`);
        }

        const tableName = this.buildTableName(tab.config, tab.selectedTable.database, tab.selectedTable.table);
        const whereClause = primaryKeyConditions.join(' AND ');
        const sql = `DELETE FROM ${tableName} WHERE ${whereClause}`;

        const response = await this.apiClient.post('/api/execute', {
          cid: tab.config.id,
          sql: sql,
          database: tab.selectedTable.database
        });

        if (response.data.code === 0) {
          this.$message.success('删除成功');
          await this.loadTableData(tabId);
        } else {
          this.$message.error(response.data.msg || '删除失败');
        }
      } catch (e) {
        this.$message.error('删除失败: ' + (e.response?.data?.msg || e.message));
        console.error('删除行失败:', e);
      }
    },
    async handleDeleteSelectedRows(tabId) {
      const tab = this.getTab(tabId);
      if (!tab || tab.selectedRows.length === 0) {
        this.$message.warning('请先选择要删除的行');
        return;
      }

      try {
        await confirmDangerousAction({
          title: '批量删除',
          message: `确定要删除选中的 ${tab.selectedRows.length} 行数据吗？`,
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }

      try {
        const newRows = tab.selectedRows.filter((row) => row._isNew);
        const savedRows = tab.selectedRows.filter((row) => !row._isNew);

        for (const row of newRows) {
          const index = tab.tableData.findIndex((r) => r._rowId === row._rowId);
          if (index > -1) {
            tab.tableData.splice(index, 1);
            tab.editingRows.delete(row._rowId);
            tab.originalRowData.delete(row._rowId);
          }
        }

        if (savedRows.length > 0) {
          const primaryKeys = tab.tableColumns.filter((col) => {
            const key = col.key || '';
            return key === 'PRI' || key === 'primary' || key === 'PRIMARY';
          });

          if (primaryKeys.length === 0 && tab.tableColumns.length > 0) {
            primaryKeys.push(tab.tableColumns[0]);
          }

          if (primaryKeys.length > 0) {
            const tableName = this.buildTableName(tab.config, tab.selectedTable.database, tab.selectedTable.table);

            if (primaryKeys.length === 1) {
              const pkName = primaryKeys[0].field || primaryKeys[0].name;
              const pkValues = savedRows
                .map((row) => {
                  const val = row[pkName];
                  return val !== undefined && val !== null ? this.escapeSqlValue(val) : null;
                })
                .filter((val) => val !== null);

              if (pkValues.length > 0) {
                const pkQuoted = this.buildColumnName(tab.config, pkName);
                const sql = `DELETE FROM ${tableName} WHERE ${pkQuoted} IN (${pkValues.join(', ')})`;

                const response = await this.apiClient.post('/api/execute', {
                  cid: tab.config.id,
                  sql: sql,
                  database: tab.selectedTable.database
                });

                if (response.data.code === 0) {
                  this.$message.success(`已删除 ${newRows.length + savedRows.length} 行`);
                  await this.loadTableData(tabId);
                } else {
                  this.$message.error(response.data.msg || '批量删除失败');
                  for (const row of savedRows) {
                    await this.handleDeleteRow(row, tabId, true);
                  }
                }
              }
            } else {
              for (const row of savedRows) {
                await this.handleDeleteRow(row, tabId, true);
              }
              if (newRows.length > 0) {
                this.$message.success(`已删除 ${newRows.length + savedRows.length} 行`);
              }
            }
          } else {
            for (const row of savedRows) {
              await this.handleDeleteRow(row, tabId, true);
            }
          }
        } else {
          this.$message.success(`已删除 ${newRows.length} 行`);
        }

        tab.selectedRows = [];
      } catch (e) {
        this.$message.error('批量删除失败: ' + (e.response?.data?.msg || e.message));
        console.error('批量删除失败:', e);
      }
    },
    handleTabRemove(tabId) {
      const index = this.tabs.findIndex(t => t.id === tabId);
      if (index > -1) {
        const removedTab = this.tabs[index];
        this.tabs.splice(index, 1);
        if (this.tabs.length > 0) {
          const newActiveTab = this.tabs[Math.min(index, this.tabs.length - 1)];
          this.activeTabId = newActiveTab.id;
          this.activeConfigId = newActiveTab.configId;
        } else {
          this.activeTabId = null;
          this.activeConfigId = null;
        }
      }
    },
    handleTabClick(tab) {
      this.activeTabId = tab.paneName;
      // 根据当前激活的标签页更新 activeConfigId
      const activeTab = this.tabs.find(t => t.id === tab.paneName);
      if (activeTab) {
        this.activeConfigId = activeTab.configId;
      }
    },
    // 处理表数据标签页切换（数据/字段）
    handleTableDataTabClick(pane, tabId) {
      const tab = this.getTab(tabId);
      if (!tab || !tab.selectedTable) return;

      // 如果切换到"字段"标签页，且字段数据还没有加载，则加载字段信息
      if (pane.name === 'columns' && (!tab.tableColumns || tab.tableColumns.length === 0)) {
        this.loadTableColumns(tabId);
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
    // 关闭其他标签页（右键菜单）
    handleCloseOtherTabs() {
      this.hideTabsContextMenu()
      const targetId = this.tabsContextMenuTabId || this.activeTabId
      if (!targetId) return

      // 保留目标标签页，关闭其他标签页
      const targetTab = this.tabs.find(t => t.id === targetId)
      if (!targetTab) return

      // 关闭其他标签页
      this.tabs = this.tabs.filter(tab => tab.id === targetId)
      this.activeTabId = targetId
      this.activeConfigId = targetTab.configId
    },
    // 关闭所有标签页（右键菜单）
    handleCloseAllTabs() {
      this.hideTabsContextMenu()
      this.tabs = []
      this.activeTabId = null
      this.activeConfigId = null
    },
    initMonaco() {
      // 等待Monaco编辑器加载
      setTimeout(() => {
        this.monacoReady = true;
      }, 100);
    },
    async insertTemplate(command, tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return;

      const dbType = this.sqlTemplateDbType(tab.config?.type);
      const quote = this.getTableNameQuote(tab.config);

      // 从后端加载模板
      if (!this.apiClient) return;

      try {
        const response = await this.apiClient.get('/api/config/sql-templates', {
          params: {
            dbType: dbType,
            command: command,
            pageSize: 1
          }
        });

        if (response.data.code === 0 && response.data.data.list && response.data.data.list.length > 0) {
          let template = response.data.data.list[0].template;

          // 如果已选择数据库和表，自动替换 table_name
          if (tab.querySelectedDatabase && tab.querySelectedTable) {
            const tableName = `${quote}${tab.querySelectedDatabase}${quote}.${quote}${tab.querySelectedTable}${quote}`;
            template = template.replace(/table_name/g, tableName);
            // 对于 Oracle 的 DESCRIBE，需要替换大写表名
            if (dbType === 'oracle' && command === 'describe') {
              template = template.replace(/TABLE_NAME/g, tab.querySelectedTable.toUpperCase());
            }
          }

          tab.querySql = template;
          return;
        }
      } catch (error) {
        console.error('加载 SQL 模板失败:', error);
      }

      // 如果后端加载失败，使用默认模板（向后兼容）
      const defaultTemplates = {
        select: `SELECT * FROM table_name LIMIT 100;`,
        select_where: `SELECT * FROM table_name WHERE id = 1;`,
        select_limit: `SELECT * FROM table_name LIMIT 10 OFFSET 0;`,
        insert: `INSERT INTO table_name (col1, col2) VALUES (val1, val2);`,
        update: `UPDATE table_name SET col1 = val1 WHERE id = 1;`,
        delete: `DELETE FROM table_name WHERE id = 1;`,
        show_tables: `SHOW TABLES;`,
        describe: `DESCRIBE table_name;`,
        count: `SELECT COUNT(*) as total FROM table_name;`
      };

      if (defaultTemplates[command]) {
        let template = defaultTemplates[command];
        if (tab.querySelectedDatabase && tab.querySelectedTable) {
          const tableName = `${quote}${tab.querySelectedDatabase}${quote}.${quote}${tab.querySelectedTable}${quote}`;
          template = template.replace(/table_name/g, tableName);
        }
        tab.querySql = template;
      }
    },
    // 判断SQL语句是否是不需要数据库的操作
    isSqlRequiresDatabase(sql) {
      if (!sql || typeof sql !== 'string') return true;

      // 去除注释和多余空格，转换为大写进行匹配
      const normalizedSql = sql.trim().toUpperCase().replace(/--.*$/gm, '').replace(/\/\*[\s\S]*?\*\//g, '').trim();
      const originalSql = sql.trim();

      // 匹配不需要数据库的SQL语句（注意：normalizedSql已经是大写，所以不需要/i标志）
      const noDatabasePatterns = [
        /^CREATE\s+DATABASE/,
        /^DROP\s+DATABASE/,
        /^USE\s+/,
        /^SHOW\s+DATABASES/,
        /^SHOW\s+SCHEMAS/,
        /^SELECT\s+.*FROM\s+INFORMATION_SCHEMA\.SCHEMATA/,
        /^SELECT\s+.*FROM\s+SYS\.DATABASES/
      ];

      // 检查是否匹配不需要数据库的模式
      for (const pattern of noDatabasePatterns) {
        if (pattern.test(normalizedSql)) {
          return false;
        }
      }

      // 对于 DDL 语句（CREATE/DROP/ALTER TABLE/VIEW/INDEX/TRIGGER 等）
      // 如果SQL中已经包含了完整的数据库限定名，则不需要选择数据库
      const ddlPatterns = [
        /^(CREATE|DROP|ALTER)\s+(TABLE|VIEW|INDEX|TRIGGER|PROCEDURE|FUNCTION|SCHEMA)\s+/i,
        /^(CREATE|DROP|ALTER)\s+(OR\s+REPLACE\s+)?(TABLE|VIEW|INDEX|TRIGGER|PROCEDURE|FUNCTION)\s+/i,
        /^(CREATE|DROP|ALTER)\s+(TEMPORARY\s+)?(TABLE|VIEW|INDEX|TRIGGER|PROCEDURE|FUNCTION)\s+/i,
        /^(CREATE|DROP|ALTER)\s+(IF\s+NOT\s+EXISTS\s+)?(TABLE|VIEW|INDEX|TRIGGER|PROCEDURE|FUNCTION)\s+/i
      ];

      let isDdl = false;
      for (const pattern of ddlPatterns) {
        if (pattern.test(originalSql)) {
          isDdl = true;
          break;
        }
      }

      if (isDdl) {
        // 提取对象名（在 DDL 关键字之后，忽略 IF NOT EXISTS、TEMPORARY 等修饰符）
        // 匹配模式：CREATE/DROP/ALTER [修饰符] TABLE/VIEW/INDEX/TRIGGER 后面的名称
        const objectNameMatch = originalSql.match(/(?:CREATE|DROP|ALTER)\s+(?:OR\s+REPLACE\s+)?(?:TEMPORARY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?(?:TABLE|VIEW|INDEX|TRIGGER|PROCEDURE|FUNCTION|SCHEMA)\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s\(]+)/i);
        if (objectNameMatch && objectNameMatch[1]) {
          let objectName = objectNameMatch[1].trim();

          // 移除可能的尾部关键字（如 ON database）
          objectName = objectName.split(/\s+ON\s+/i)[0];

          // 检查对象名是否包含数据库限定名格式
          // SQL Server: [database].[schema].[table] 或 [database].schema.table 或 database.schema.table
          // MySQL: `database`.`table` 或 database.table
          // PostgreSQL: database.schema.table 或 "database"."schema"."table"

          // 1. 检查 SQL Server 格式：[database].[schema].[table] 或 [database].schema.table
          if (objectName.match(/\[[^\]]+\]\.\[[^\]]+\]/) || objectName.match(/\[[^\]]+\]\.\w+/)) {
            // 包含SQL Server格式的限定名，不需要选择数据库
            return false;
          }

          // 2. 检查是否包含点号（可能是限定名）
          if (objectName.includes('.')) {
            // 移除可能的引号（方括号、反引号、双引号）后检查是否包含点号
            const nameWithoutOuterQuotes = objectName.replace(/^\[|\]$/g, '').replace(/^`|`$/g, '').replace(/^"|"$/g, '');

            if (nameWithoutOuterQuotes.includes('.')) {
              // 分割点号，检查各部分
              const parts = nameWithoutOuterQuotes.split('.');
              const cleanParts = parts.map(part => {
                // 移除每个部分的引号
                return part.replace(/^\[|\]$/g, '').replace(/^`|`$/g, '').replace(/^"|"$/g, '').trim();
              }).filter(part => part.length > 0);

              // 如果至少有2部分，且第一部分不是SQL关键字，则认为是数据库限定名
              if (cleanParts.length >= 2) {
                const firstPart = cleanParts[0].toUpperCase();
                // 排除常见的SQL关键字
                const keywords = ['IF', 'NOT', 'EXISTS', 'OR', 'REPLACE', 'TEMPORARY', 'TEMP', 'UNIQUE', 'FULLTEXT', 'SPATIAL', 'ON'];
                if (!keywords.includes(firstPart) && /^[A-Z_][A-Z0-9_]*$/i.test(firstPart)) {
                  // 第一部分看起来像是数据库名（以字母或下划线开头），包含数据库限定名，不需要选择数据库
                  return false;
                }
              }
            }
          }
        }

        // 对于 DDL 语句，即使没有包含数据库限定名，也允许执行（使用默认数据库）
        // 这样可以支持 CREATE TABLE table_name 这样的语句
        return false;
      }

      // 对于其他SQL，需要数据库
      return true;
    },
    async executeQuery(tabId) {
      const tab = this.getTab(tabId);
      if (!tab || !tab.querySql || !this.apiClient) return;

      // 检查SQL是否为空
      if (!tab.querySql.trim()) {
        this.$message.warning('SQL语句不能为空');
        return;
      }

      // 使用选择的数据库，如果没有选择则使用当前数据库或配置的数据库，都没有则为空（让后端处理）
      const database = tab.querySelectedDatabase || tab.currentDatabase || tab.config.database || '';

      tab.queryLoading = true;
      tab.queryError = null;
      tab.queryResult = null;

      try {
        const response = await this.apiClient.post('/api/query', {
          cid: tab.config.id,
          sql: tab.querySql,
          database: database
        });

        if (response.data.code === 0) {
          tab.queryResult = response.data.data;

          // 确保 columns 存在且是数组
          if (tab.queryResult && !tab.queryResult.columns) {
            console.warn('Columns missing in query result!', tab.queryResult);
            tab.queryResult.columns = [];
          } else if (tab.queryResult && !Array.isArray(tab.queryResult.columns)) {
            console.warn('Columns is not an array!', tab.queryResult.columns);
            tab.queryResult.columns = [];
          }

          tab.queryError = null;
        } else {
          // 直接显示错误信息
          tab.queryError = response.data.msg || '查询失败';
          tab.queryResult = null;
        }
      } catch (e) {
        // 直接显示错误信息
        const errorMsg = e.response?.data?.msg || e.message || '查询失败';
        tab.queryError = errorMsg;
        tab.queryResult = null;
      } finally {
        tab.queryLoading = false;
      }
    },
    clearQuery(tabId) {
      const tab = this.getTab(tabId);
      if (tab) {
        tab.querySql = '';
        tab.queryResult = null;
        tab.queryError = null;
      }
    },
    async handleQueryDatabaseChange(database, tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return;

      tab.querySelectedTable = null;
      tab.queryTableList = [];

      if (!database || !this.apiClient) return;

      try {
        const response = await this.apiClient.post('/api/tables', {
          cid: tab.config.id,
          database: database
        });

        if (response.data.code === 0) {
          const tables = response.data.data || [];
          tab.queryTableList = tables.map(table => ({
            name: typeof table === 'string' ? table : table.name
          }));
        }
      } catch (e) {
        console.error('加载表列表失败:', e);
      }
    },
    handleQueryTableChange(table, tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return;

      // 表改变时自动填充表名到SQL编辑器（如果SQL为空）
      if (table && tab.querySelectedDatabase) {
        if (!tab.querySql || tab.querySql.trim() === '') {
          this.insertTableToQuery(tabId);
        }
      }
    },
    insertTableToQuery(tabId) {
      const tab = this.getTab(tabId);
      if (!tab || !tab.querySelectedDatabase || !tab.querySelectedTable) return;

      const quote = this.getTableNameQuote(tab.config);
      const tableName = `${quote}${tab.querySelectedDatabase}${quote}.${quote}${tab.querySelectedTable}${quote}`;

      // 如果SQL为空，插入SELECT语句模板
      if (!tab.querySql || tab.querySql.trim() === '') {
        tab.querySql = `SELECT * FROM ${tableName} LIMIT 100;`;
        return;
      }

      // 如果SQL中包含 table_name 占位符，替换它
      if (tab.querySql.includes('table_name')) {
        tab.querySql = tab.querySql.replace(/table_name/g, tableName);
        return;
      }

      // 否则在末尾追加表名
      if (tab.querySql.trim().endsWith(';')) {
        tab.querySql = tab.querySql.slice(0, -1) + ' ' + tableName + ';';
      } else {
        tab.querySql = tab.querySql + ' ' + tableName;
      }
    },
    // 检查服务器是否可用
    async checkServerAvailable(retries = 3, delay = 500) {
      if (!this.apiClient) return false;

      for (let i = 0; i < retries; i++) {
        try {
          const response = await this.apiClient.get('/api/port-config', { timeout: 2000 });
          if (response.status === 200) {
            return true;
          }
        } catch (error) {
          if (i < retries - 1) {
            await new Promise(resolve => setTimeout(resolve, delay));
          }
        }
      }
      return false;
    },
    // SQL 模板管理方法
    async loadSqlTemplates() {
      // 确保 API 客户端已初始化
      if (!this.apiClient) {
        await this.initClient();
      }
      if (!this.apiClient) {
        this.$message.error('API 客户端初始化失败，请检查服务是否启动');
        return;
      }

      // 检查服务器是否可用
      const serverAvailable = await this.checkServerAvailable();
      if (!serverAvailable) {
        this.$message.warning('服务器尚未启动，请稍候再试');
        return;
      }

      this.sqlTemplateLoading = true;
      try {
        const response = await this.apiClient.get('/api/config/sql-templates', {
          params: {
            dbType: this.sqlTemplateFilter.dbType || undefined,
            category: this.sqlTemplateFilter.category || undefined,
            search: this.sqlTemplateSearch || undefined,
            page: this.sqlTemplatePagination.page,
            pageSize: this.sqlTemplatePagination.pageSize
          }
        });

        if (response.data.code === 0) {
          const data = response.data.data;
          this.sqlTemplates = data.list || [];
          this.sqlTemplatePagination.total = data.total || 0;
        } else {
          this.$message.warning(response.data.msg || '加载 SQL 模板失败');
        }
      } catch (error) {
        console.error('加载 SQL 模板失败:', error);
        const errorMsg = error.response?.data?.msg || error.message || '加载 SQL 模板失败';

        // 如果是连接错误，提示用户等待服务器启动
        if (error.code === 'ERR_NETWORK' || error.code === 'ECONNREFUSED') {
          this.$message.warning('服务器连接失败，请确保后端服务已启动');
          // 重置 API 客户端，下次重试
          this.apiClient = null;
        } else {
          this.$message.error(errorMsg);
        }
      } finally {
        this.sqlTemplateLoading = false;
      }
    },
    async loadCategoriesOnInit() {
      // 初始化时加载分类列表，确保显示所有分类
      if (this.apiClient) {
        try {
          const response = await this.apiClient.get('/api/config/sql-template-categories');
          if (response.data.code === 0) {
            // 确保显示所有分类，不管有没有数据
            this.sqlTemplateCategories = (response.data.data || []).map(cat => cat.name);
          }
        } catch (error) {
          console.warn('加载分类列表失败:', error);
        }
      }
    },
    handleAddSqlTemplate() {
      this.editingSqlTemplate = {
        id: '',
        dbType: 'mysql',
        command: '',
        name: '',
        category: '',
        template: ''
      };
      this.showSqlTemplateEditDialog = true;
    },
    handleEditSqlTemplate(row) {
      this.editingSqlTemplate = {
        id: row.id,
        dbType: row.dbType,
        command: row.command,
        name: row.name,
        category: row.category,
        template: row.template
      };
      this.showSqlTemplateEditDialog = true;
    },
    async handleSaveSqlTemplate() {
      if (!this.editingSqlTemplate.name || !this.editingSqlTemplate.command || !this.editingSqlTemplate.template || !this.editingSqlTemplate.category) {
        this.$message.warning('请填写完整信息');
        return;
      }

      if (!this.apiClient) return;

      try {
        // 如果分类不在列表中，先创建分类
        if (!this.sqlTemplateCategories.includes(this.editingSqlTemplate.category)) {
          try {
            const categoryResponse = await this.apiClient.post('/api/config/sql-template-categories', {
              name: this.editingSqlTemplate.category.trim()
            });
            if (categoryResponse.data.code === 0) {
              // 更新分类列表
              await this.loadCategories();
            }
          } catch (catError) {
            // 如果创建分类失败，继续保存模板（可能分类已存在）
            console.warn('创建分类失败或分类已存在:', catError);
          }
        }

        if (this.editingSqlTemplate.id) {
          // 更新
          await this.apiClient.put(`/api/config/sql-templates/${this.editingSqlTemplate.id}`, {
            dbType: this.editingSqlTemplate.dbType,
            command: this.editingSqlTemplate.command,
            name: this.editingSqlTemplate.name,
            category: this.editingSqlTemplate.category,
            template: this.editingSqlTemplate.template
          });
          this.$message.success('更新成功');
        } else {
          // 新增
          await this.apiClient.post('/api/config/sql-templates', {
            dbType: this.editingSqlTemplate.dbType,
            command: this.editingSqlTemplate.command,
            name: this.editingSqlTemplate.name,
            category: this.editingSqlTemplate.category,
            template: this.editingSqlTemplate.template
          });
          this.$message.success('新增成功');
        }

        this.showSqlTemplateEditDialog = false;
        await this.loadSqlTemplates();
        // 重新加载模板到下拉框
        await this.loadSqlTemplatesForDropdown();
      } catch (error) {
        console.error('保存 SQL 模板失败:', error);
        this.$message.error('保存 SQL 模板失败: ' + (error.response?.data?.msg || error.message));
      }
    },
    async handleDeleteSqlTemplate(row) {
      if (!this.apiClient) return;

      try {
        await confirmDangerousAction({
          title: '删除模板',
          message: `将删除 SQL 模板「${row.name || row.id}」。`,
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }

      try {
        await this.apiClient.delete(`/api/config/sql-templates/${row.id}`);
        this.$message.success('删除成功');
        await this.loadSqlTemplates();
        // 重新加载模板到下拉框
        await this.loadSqlTemplatesForDropdown();
      } catch (error) {
        console.error('删除 SQL 模板失败:', error);
        this.$message.error('删除 SQL 模板失败: ' + (error.response?.data?.msg || error.message));
      }
    },
    async handleResetSqlTemplates() {
      if (!this.apiClient) return;

      try {
        await confirmDangerousAction({
          title: '恢复初始模板',
          message: '确定要恢复初始值吗？这将覆盖所有自定义 SQL 模板。',
          expectedInput: DANGER_DELETE_TOKEN
        });

        await this.apiClient.post('/api/config/sql-templates/reset');
        this.$message.success('恢复初始值成功');
        await this.loadSqlTemplates();
        // 重新加载模板到下拉框
        await this.loadSqlTemplatesForDropdown();
      } catch (error) {
        if (error === 'cancel' || error === 'close') return;
        console.error('恢复初始值失败:', error);
        this.$message.error('恢复初始值失败: ' + (error.response?.data?.msg || error.message));
      }
    },
    async loadSqlTemplatesForDropdown(tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return;

      // 确保 API 客户端已初始化
      if (!this.apiClient) {
        await this.initClient();
      }
      if (!this.apiClient) return;

      const dbType = this.sqlTemplateDbType(tab.config?.type);

      try {
        const response = await this.apiClient.get('/api/config/sql-templates', {
          params: {
            dbType: dbType,
            pageSize: 1000 // 获取所有模板
          }
        });

        if (response.data.code === 0 && response.data.data.list) {
          // 按分类缓存模板
          const templates = response.data.data.list;
          this.sqlTemplateCategories.forEach(category => {
            const cacheKey = `${dbType}_${category}`;
            this.sqlTemplatesCache[cacheKey] = templates.filter(t => t.category === category);
          });
        }
      } catch (error) {
        console.error('加载 SQL 模板下拉框失败:', error);
        // 如果连接失败，尝试重新初始化
        if (error.code === 'ERR_NETWORK' || error.code === 'ECONNREFUSED') {
          this.apiClient = null;
          await this.initClient();
        }
      }
    },
    getTemplatesByCategory(category, tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return [];
      const dbType = this.sqlTemplateDbType(tab.config?.type);
      const cacheKey = `${dbType}_${category}`;
      return this.sqlTemplatesCache[cacheKey] || [];
    },
    // 分类管理方法
    async loadCategories() {
      if (!this.apiClient) {
        await this.initClient();
      }
      if (!this.apiClient) return;

      this.categoryLoading = true;
      try {
        const response = await this.apiClient.get('/api/config/sql-template-categories');
        if (response.data.code === 0) {
          // 确保显示所有分类，不管有没有数据
          this.categories = (response.data.data || []).map(cat => ({
            ...cat,
            _editing: false,
            _editName: ''
          }));
          // 更新分类列表（用于下拉框和常用命令）
          this.sqlTemplateCategories = this.categories.map(cat => cat.name);
        } else {
          this.$message.warning(response.data.msg || '加载分类失败');
        }
      } catch (error) {
        console.error('加载分类失败:', error);
        this.$message.error('加载分类失败: ' + (error.response?.data?.msg || error.message));
      } finally {
        this.categoryLoading = false;
      }
    },
    handleAddCategory() {
      this.newCategory = { name: '' };
      this.showAddCategoryDialog = true;
    },
    async handleSaveNewCategory() {
      if (!this.newCategory.name || !this.newCategory.name.trim()) {
        this.$message.warning('请输入分类名称');
        return;
      }

      if (!this.apiClient) return;

      try {
        const response = await this.apiClient.post('/api/config/sql-template-categories', {
          name: this.newCategory.name.trim()
        });

        if (response.data.code === 0) {
          this.$message.success('新增分类成功');
          this.showAddCategoryDialog = false;
          // 重新加载分类列表，确保新增的分类显示出来
          await this.loadCategories();
          // 同时更新下拉框中的分类列表
          this.sqlTemplateCategories = this.categories.map(cat => cat.name);
        } else {
          this.$message.error(response.data.msg || '新增分类失败');
        }
      } catch (error) {
        console.error('新增分类失败:', error);
        this.$message.error('新增分类失败: ' + (error.response?.data?.msg || error.message));
      }
    },
    handleEditCategory(row) {
      row._editing = true;
      row._editName = row.name;
    },
    async handleSaveCategoryEdit(row) {
      if (!row._editName || !row._editName.trim()) {
        this.$message.warning('分类名称不能为空');
        return;
      }

      if (row._editName.trim() === row.name) {
        // 没有变化，直接取消编辑
        row._editing = false;
        row._editName = '';
        return;
      }

      if (!this.apiClient) return;

      try {
        const response = await this.apiClient.put(`/api/config/sql-template-categories/${row.id}`, {
          name: row._editName.trim()
        });

        if (response.data.code === 0) {
          this.$message.success('更新分类成功');
          row.name = row._editName.trim();
          row._editing = false;
          row._editName = '';
          await this.loadCategories();
        } else {
          this.$message.error(response.data.msg || '更新分类失败');
        }
      } catch (error) {
        console.error('更新分类失败:', error);
        this.$message.error('更新分类失败: ' + (error.response?.data?.msg || error.message));
      }
    },
    async handleDeleteCategory(row) {
      if (!this.apiClient) return;

      try {
        await confirmDangerousAction({
          title: '删除分类',
          message: `将删除分类「${row.name}」及其下全部 SQL 模板，不可恢复。`,
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }

      try {
        // 先删除该分类下的所有SQL模板
        let deletedTemplateCount = 0;
        try {
          // 获取该分类下的所有SQL模板
          const templatesResponse = await this.apiClient.get('/api/config/sql-templates', {
            params: {
              category: row.name,
              pageSize: 10000 // 获取所有该分类下的模板
            }
          });

          if (templatesResponse.data.code === 0 && templatesResponse.data.data.list) {
            const templates = templatesResponse.data.data.list;
            // 批量删除该分类下的所有SQL模板
            for (const template of templates) {
              try {
                await this.apiClient.delete(`/api/config/sql-templates/${template.id}`);
                deletedTemplateCount++;
              } catch (templateError) {
                console.warn(`删除SQL模板 ${template.id} 失败:`, templateError);
                // 继续删除其他模板，不中断流程
              }
            }
          }
        } catch (templateListError) {
          // 如果获取模板列表失败，记录警告但继续删除分类
          console.warn('获取SQL模板列表失败，继续删除分类:', templateListError);
        }

        // 然后删除分类本身
        const response = await this.apiClient.delete(`/api/config/sql-template-categories/${row.id}`);

        if (response.data.code === 0) {
          if (deletedTemplateCount > 0) {
            this.$message.success(`删除分类成功，已删除 ${deletedTemplateCount} 个SQL模板`);
          } else {
            this.$message.success('删除分类成功');
          }
          await this.loadCategories();
          // 重新加载SQL模板列表，因为删除了模板
          await this.loadSqlTemplates();
        } else {
          this.$message.error(response.data.msg || '删除分类失败');
        }
      } catch (error) {
        console.error('删除分类失败:', error);
        this.$message.error('删除分类失败: ' + (error.response?.data?.msg || error.message));
      }
    }
  }
};
</script>

<style scoped>
.dbmanager-container {
  height: 100%;
  display: flex;
  gap: 8px;
  padding: 8px;
  box-sizing: border-box;
  overflow: hidden;
  background: transparent;
  position: relative;
}

.dbmanager-sidebar {
  flex: 0 0 280px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid rgba(0, 0, 0, 0.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.dbmanager-toggle-sidebar-btn {
  position: fixed;
  top: 72px;
  left: 132px;
  z-index: 999;
}

.sidebar-header {
  padding: 12px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
  background: rgba(255, 255, 255, 1);
  display: flex;
  gap: 8px;
  align-items: center;
}

.sidebar-content {
  flex: 1;
  overflow: hidden;
  transition: all 0.3s ease;
}

.sidebar-content.collapsed {
  display: none;
}

.dbmanager-sidebar.collapsed {
  /* 保留类，方便以后扩展，如需要时可添加特定折叠样式 */
}

.config-menu {
  border: none;
  background: transparent;
}

.config-type {
  float: right;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.6);
}

.dbmanager-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  padding: 12px;
}

.toolbar {
  margin-bottom: 5px;
}

.connection-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.connection-tabs :deep(.el-tabs__content) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.connection-tabs :deep(.el-tab-pane) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 4px;
}

.tab-icon {
  font-size: 14px;
}

.tab-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.db-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.database-panel {
  display: flex;
  gap: 12px;
  height: 100%;
  overflow: hidden;
}

.database-tree-wrapper {
  flex: 0 0 300px;
  min-width: 300px;
  height: 100%;
  max-height: calc(100vh - 180px);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.database-tree {
  padding: 8px;
}

.table-data-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 1);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 4px;
  padding: 12px;
  overflow: hidden;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
}

.table-header-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.table-header h3 {
  margin: 0;
  font-size: 16px;
  color: #333;
}

.config-host-info {
  font-size: 12px;
  color: rgba(64, 158, 255, 0.8);
  cursor: pointer;
  transition: opacity 0.3s ease, transform 0.3s ease;
  opacity: 0;
  transform: translateY(-5px);
  white-space: nowrap;
  display: inline-block;
}

.config-host-info.show {
  opacity: 1;
  transform: translateY(0);
}

.config-host-info:hover {
  color: #409eff;
  text-decoration: underline;
}

.table-data-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.table-data-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
  height: 100%;
}

.table-wrapper {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.table-wrapper :deep(.el-table) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.table-wrapper :deep(.el-table__body-wrapper) {
  flex: 1;
  overflow-y: auto !important;
  min-height: 0;
}

/* 表格加载时的半透明效果，保持旧数据可见 */
.table-loading {
  opacity: 0.6;
  pointer-events: none;
  transition: opacity 0.3s ease;
}

.table-wrapper :deep(.table-loading) {
  opacity: 0.6;
}

.table-pagination {
  flex-shrink: 0;
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
  padding: 12px 0;
  border-top: 1px solid rgba(0, 0, 0, 0.1);
  background: #fff;
  z-index: 1;
}

.columns-container {
  height: 100%;
  overflow-y: auto;
}

.query-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.query-toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
  flex-wrap: nowrap;
}

.query-selectors {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: nowrap;
  flex-shrink: 0;
}

.query-actions {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: nowrap;
  flex-shrink: 0;
}

.query-editor-wrapper {
  flex: 1;
  min-height: 300px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 4px;
  overflow: hidden;
}

.query-editor {
  width: 100%;
  height: 100%;
}

.query-editor :deep(.monaco-editor) {
  background: #ffffff;
}

.query-result-container {
  flex: 1;
  min-height: 200px;
  max-height: 500px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: auto;
}

.query-result {
  flex: 1;
  min-height: 0;
  overflow: auto;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 4px;
  padding: 12px;
  background: rgba(255, 255, 255, 1);
}

.query-error {
  flex-shrink: 0;
}

.result-header {
  margin-bottom: 8px;
  font-weight: 500;
  color: #333;
}

.dbmanager-empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.empty-content {
  text-align: center;
  color: rgba(0, 0, 0, 0.6);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  color: #409eff;
}

.empty-text {
  font-size: 14px;
  margin: 0;
}

/* 对话框底部样式 */
.dialog-footer-right {
  display: flex;
  gap: 8px;
}

/* 主机列表样式 */
.config-list {
  padding: 8px;
}

.config-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  margin-bottom: 8px;
  border-radius: 6px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  background: rgba(255, 255, 255, 0.8);
  cursor: pointer;
  transition: all 0.2s ease;
}

.config-item:hover {
  background: rgba(64, 158, 255, 0.1);
  border-color: rgba(64, 158, 255, 0.3);
}

.config-item.active {
  background: rgba(64, 158, 255, 0.15);
  border-color: rgba(64, 158, 255, 0.5);
}

.config-info {
  flex: 1;
  min-width: 0;
}

.config-name {
  font-weight: 600;
  font-size: 14px;
  color: #333;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.config-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.config-host {
  color: rgba(0, 0, 0, 0.6);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.config-actions {
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s;
}

.config-item:hover .config-actions {
  opacity: 1;
}

.empty-config {
  padding: 40px 20px;
  text-align: center;
}

/* 数据库树样式优化 */
.database-tree-scrollbar {
  flex: 1;
  min-height: 0;
  height: 100%;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 4px;
  background: rgba(255, 255, 255, 1);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.database-tree-scrollbar :deep(.el-scrollbar) {
  flex: 1;
  height: 100%;
  min-height: 0;
}

.database-tree-scrollbar :deep(.el-scrollbar__wrap) {
  overflow-x: hidden !important;
  overflow-y: auto !important;
  height: 100%;
}

.database-tree-scrollbar :deep(.el-scrollbar__bar) {
  opacity: 0.6;
}

.database-tree-scrollbar :deep(.el-scrollbar__view) {
  height: 100%;
}

.database-tree {
  padding: 8px;
}

.database-tree :deep(.el-tree-node__content) {
  height: 32px;
  padding: 0 8px;
  border-radius: 4px;
  margin-bottom: 2px;
  transition: all 0.2s;
}

.database-tree :deep(.el-tree-node__content:hover) {
  background: rgba(64, 158, 255, 0.1);
}

.database-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background: rgba(64, 158, 255, 0.15);
  color: #409eff;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.tree-node-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.table-icon {
  color: #67c23a;
}

.database-icon {
  color: #409eff;
}

.empty-icon {
  color: #909399;
}

.empty-hint {
  color: #909399;
  font-style: italic;
}

/* 表格数据样式 */
.table-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px;
  background: rgba(245, 247, 250, 0.8);
  border-radius: 4px;
  flex-wrap: wrap;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: nowrap;
  flex: 1;
  justify-content: flex-end;
  min-width: 0;
  overflow: visible;
}

.toolbar-right :deep(.el-pagination) {
  margin: 0;
  flex-shrink: 0;
}

.toolbar-right :deep(.el-pagination__total) {
  margin-right: 8px;
  white-space: nowrap;
}

.toolbar-right :deep(.el-pagination__sizes) {
  margin: 0 8px 0 0;
}

.toolbar-right :deep(.el-pagination__jump) {
  margin-left: 8px;
}

.data-count {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.6);
  white-space: nowrap;
}

.page-info {
  color: rgba(0, 0, 0, 0.45);
  margin-left: 4px;
}

/* 配置选择对话框样式 */
.config-select-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 400px;
  overflow-y: auto;
}

.config-select-item {
  width: 100%;
  padding: 12px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 4px;
  transition: all 0.2s;
}

.config-select-item:hover {
  background: rgba(64, 158, 255, 0.05);
  border-color: rgba(64, 158, 255, 0.3);
}

.config-select-item.is-checked {
  background: rgba(64, 158, 255, 0.1);
  border-color: rgba(64, 158, 255, 0.5);
}

.config-select-info {
  flex: 1;
}

.config-select-name {
  font-weight: 600;
  font-size: 14px;
  color: #333;
  margin-bottom: 4px;
}

.config-select-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.6);
}

/* SQL 常用命令下拉菜单样式 - 支持滚动 */
.sql-command-dropdown {
  max-height: 500px;
  overflow-y: auto;
}

.sql-command-dropdown .el-dropdown-menu {
  max-height: 500px;
  overflow-y: auto;
}

.sql-command-dropdown .el-dropdown-menu__item {
  padding: 8px 20px;
  font-size: 13px;
  line-height: 1.5;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 500px;
}

.sql-command-dropdown .el-dropdown-menu__item.is-divided {
  padding: 6px 20px;
  font-weight: 600;
  color: #409eff;
  background-color: rgba(64, 158, 255, 0.08);
  cursor: default;
  font-size: 12px;
}

.sql-command-dropdown .el-dropdown-menu__item.is-divided:hover {
  background-color: rgba(64, 158, 255, 0.08);
}

/* 自定义滚动条样式 */
.sql-command-dropdown::-webkit-scrollbar {
  width: 6px;
}

.sql-command-dropdown::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.05);
  border-radius: 3px;
}

.sql-command-dropdown::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 3px;
}

.sql-command-dropdown::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.3);
}

/* 数据表单元格双击查看样式 */
.db-table-cell-text {
  display: inline-block;
  width: 100%;
  cursor: zoom-in;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.db-table-cell-text:hover {
  text-decoration: underline dotted;
}

.db-table-cell-null {
  opacity: 0.5;
  font-style: italic;
}

.cell-viewer-editor {
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  overflow: hidden;
}

.cell-viewer-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid #e4e7ed;
  background: #f5f7fa;
}

.cell-viewer-toolbar .toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cell-viewer-toolbar .toolbar-right {
  font-size: 12px;
  color: #909399;
}

.db-table-hint {
  margin-right: 12px;
  font-size: 12px;
  color: #909399;
}

/* 单元格编辑对话框美化 */
.cell-viewer-dialog :deep(.el-dialog__header) {
  padding: 10px 16px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
}

.cell-viewer-dialog :deep(.el-dialog__title) {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.cell-viewer-dialog :deep(.el-dialog__body) {
  padding: 0 0 12px 0;
  background: #1e1e1e;
}

.cell-viewer-dialog :deep(.el-dialog__footer) {
  padding: 10px 16px;
  border-top: 1px solid #e4e7ed;
  background: #f5f7fa;
}

/* 标签页右键菜单样式 */
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
</style>
