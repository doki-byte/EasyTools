<template>
  <div class="ftp-container">
    <!-- 折叠/展开按钮，始终悬浮在左上角 -->
    <el-button
      class="ftp-toggle-sidebar-btn"
      size="small"
      circle
      @click="sidebarCollapsed = !sidebarCollapsed"
      :title="sidebarCollapsed ? '展开连接列表' : '收起连接列表'">
      <el-icon>
        <component :is="sidebarCollapsed ? 'ArrowRight' : 'ArrowLeft'" />
      </el-icon>
    </el-button>

    <!-- 左侧：连接管理区域 -->
    <div v-if="!sidebarCollapsed" class="ftp-sidebar">
      <div class="sidebar-header">
        <el-button type="primary" size="small" @click="showConfigDialog = true">新建连接</el-button>
      </div>
      <div class="sidebar-content">
        <el-scrollbar>
          <div class="config-list">
            <div
              v-for="config in configs"
              :key="config.id"
              :class="['config-item', { active: isConfigActive(config.id) }]"
              @click="handleConfigSelect(config.id)"
            >
              <div class="config-info">
                <div class="config-name">{{ config.name }}</div>
                <div class="config-meta">
                  <span class="config-host">{{ config.host }}:{{ config.port }}</span>
                </div>
              </div>
              <div class="config-actions" @click.stop>
                <el-button
                  link
                  type="primary"
                  size="small"
                  @click="editConfig(config)"
                  title="编辑"
                >
                  <el-icon><Edit /></el-icon>
                </el-button>
                <el-button link type="danger" size="small" title="删除" @click="deleteConfig(config.id)">
                  <el-icon><Delete /></el-icon>
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

    <!-- 右侧：FTP操作区域 -->
    <div class="ftp-main">
      <template v-if="tabs.length > 0">
        <el-tabs 
          v-model="activeTabId" 
          type="card" 
          closable
          @tab-remove="handleTabRemove"
          @tab-click="handleTabClick"
          @contextmenu.prevent="handleTabsContextMenu"
          class="ftp-tabs"
        >
          <el-tab-pane
            v-for="tab in tabs"
            :key="tab.id"
            :label="tab.label"
            :name="tab.id"
          >
            <template #label>
              <span class="tab-label">
                <el-icon class="tab-icon"><Connection /></el-icon>
                {{ tab.label }}
              </span>
            </template>
            <div class="tab-content">
              <el-card class="file-manager-card">
                <template #header>
                  <div class="file-manager-header">
                    <el-breadcrumb separator="/">
                      <el-breadcrumb-item
                          v-for="(label, index) in getTabPathItems(tab.id)"
                          :key="index"
                          @click="ftpNavigateTo(index, tab.id)">
                        {{ index === 0 ? '根目录' : label }}
                      </el-breadcrumb-item>
                    </el-breadcrumb>

                    <el-button type="warning" @click="showNewFolderDialog(tab.id)">
                      <el-icon><Folder /></el-icon> 新建文件夹
                    </el-button>

                    <input
                        :ref="el => setFileInputRef(el, tab.id)"
                        type="file"
                        multiple
                        style="display: none"
                        @change="(e) => onFileSelected(e, tab.id)"
                    />
                    <input
                        :ref="el => setFolderInputRef(el, tab.id)"
                        type="file"
                        webkitdirectory
                        multiple
                        style="display: none"
                        @change="(e) => onFolderSelected(e, tab.id)"
                    />

                    <!-- 上传按钮，点击时触发 file input -->
                    <el-button type="primary" @click="triggerFileInput(tab.id)">
                      <el-icon><Upload /></el-icon> 上传文件
                    </el-button>
                    <!-- 记住覆盖/跳过策略：放在上传按钮右侧，更显眼 -->
                    <el-checkbox
                      v-model="rememberUploadStrategy"
                      style="margin-left: 12px; font-weight: 500;"
                    >
                      对后续同名文件使用相同处理
                    </el-checkbox>
                    
                    <!-- 上传文件夹按钮 -->
                    <!-- <el-button type="primary" @click="triggerFolderInput(tab.id)">
                      <el-icon><Folder /></el-icon> 上传文件夹
                    </el-button> -->

                    <!-- 上传进度与取消 -->
                    <div
                      v-if="tab.uploading"
                      style="display: inline-flex; align-items: center; gap: 6px; margin-left: 5px;"
                    >
                      <el-progress
                        :percentage="tab.uploadProgress"
                        :status="tab.uploadProgress === 100 ? 'success' : ''"
                        :style="{ width: '80px' }"
                      />
                      <el-button
                        type="danger"
                        text
                        size="small"
                        @click="cancelFtpUpload(tab.id)"
                      >
                        取消上传
                      </el-button>
                    </div>
                  </div>
                </template>

                <div class="ftp-file-content" :class="{ 'has-preview': filePreview.file && filePreview.tabId === tab.id }">
                  <div
                      class="ftp-file-table-wrapper"
                      @drop.prevent="handleFtpFileAreaDrop($event, tab.id)"
                      @dragover.prevent
                      @dragenter.prevent
                  >
                    <el-table
                        :data="tab.fileList"
                        stripe
                        style="width: 100%"
                        class="ftp-file-table"
                        :height="fileTableHeight"
                        @row-contextmenu="handleFtpRowContextMenu"
                        @row-dblclick="(row) => handleFtpFileNameDblClick(row, tab.id)"
                    >
                      <el-table-column prop="name" label="名称" width="280">
                        <template #default="{ row }">
                          <div class="file-ftp-item">
                            <el-icon v-if="row.type === 'folder'"><Folder /></el-icon>
                            <el-icon v-else><Document /></el-icon>
                            <span
                                class="file-ftp-name"
                                @click="row.type === 'folder' ? enterFtpFolder(row.name, tab.id) : null"
                                :style="{ cursor: row.type === 'folder' ? 'pointer' : 'default' }">
                              {{ row.name }}
                            </span>
                          </div>
                        </template>
                      </el-table-column>
                      <el-table-column prop="type" label="类型" width="100"></el-table-column>
                      <el-table-column prop="size" label="大小" width="120">
                        <template #default="{ row }">
                          {{ row.type === 'folder' ? '-' : formatFtpFileSize(row.size) }}
                        </template>
                      </el-table-column>
                      <el-table-column prop="time" label="修改时间" width="180"></el-table-column>
                    </el-table>
                  </div>

                  <!-- 文件预览面板 -->
                  <div class="ftp-preview-pane" v-if="filePreview.file && filePreview.tabId === tab.id">
                    <div class="preview-header">
                      <div class="preview-title">
                        <span>预览：{{ filePreview.file.name }}</span>
                      </div>
                      <div class="preview-actions">
                        <el-tag v-if="filePreview.isText" size="small" type="info" effect="plain">
                          {{ formatFtpFileSize(filePreview.file.size) }}
                        </el-tag>
                        <el-button v-if="filePreview.isText" size="small" type="primary"
                          :disabled="!filePreview.dirty" @click="saveFtpFileEdit">
                          保存修改
                        </el-button>
                        <el-button size="small" text type="danger" class="preview-close-btn" @click="closeFtpFilePreview">
                          关闭预览
                        </el-button>
                      </div>
                    </div>

                    <div class="preview-body">
                      <el-scrollbar class="preview-scroll">
                        <div v-if="filePreview.loading" class="preview-empty">
                          <el-icon class="is-loading" style="margin-right: 6px">
                            <Refresh />
                          </el-icon>
                          正在加载文件内容...
                        </div>
                        <div v-else-if="filePreview.error" class="preview-empty">
                          <el-empty :description="filePreview.error" />
                        </div>
                        <div v-else-if="filePreview.isText" class="preview-editor-wrapper">
                          <VueMonacoEditor v-model:value="filePreview.editContent" :language="filePreviewLanguage"
                            theme="vs-dark" :height="'250px'" class="code-editor" :options="filePreviewMonacoOptions" />
                        </div>
                        <div v-else class="preview-empty">
                          <el-empty description="当前文件类型暂不支持在线预览，请使用下载操作" />
                        </div>
                      </el-scrollbar>
                    </div>
                  </div>
                </div>

                <el-pagination
                    v-model:current-page="tab.currentPage"
                    v-model:page-size="tab.pageSize"
                    :page-sizes="[10, 20, 50, 100]"
                    :total="tab.totalFiles"
                    layout="sizes, prev, pager, next, jumper"
                    @current-change="(page) => onPageChange(page, tab.id)"
                    @size-change="(size) => onSizeChange(size, tab.id)"
                />
              </el-card>
            </div>
          </el-tab-pane>
        </el-tabs>
      </template>
      <div v-else class="ftp-empty-state">
        <div class="empty-content">
          <el-icon class="empty-icon"><Connection /></el-icon>
          <p class="empty-text">请先选择一个FTP连接</p>
        </div>
      </div>
    </div>

    <!-- 配置对话框 -->
    <el-dialog
      v-model="showConfigDialog"
      :title="configForm.id ? '编辑连接配置' : '新建连接配置'"
      width="500px"
      @close="handleConfigDialogClose"
    >
      <el-form :model="configForm" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="configForm.name" placeholder="连接名称" />
        </el-form-item>
        <el-form-item label="主机地址">
          <el-input v-model="configForm.host" placeholder="ftp.example.com" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input v-model.number="configForm.port" type="number" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-select v-model="configForm.username" placeholder="请选择或输入用户名" filterable allow-create default-first-option>
            <el-option label="anonymous" value="anonymous" />
          </el-select>
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="configForm.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="连接方式">
          <el-select v-model="configForm.connection_type" placeholder="请选择连接方式" clearable @change="handleConnectionTypeChange">
            <el-option value="direct" label="直连" />
            <el-option value="ssh" label="SSH隧道" />
          </el-select>
        </el-form-item>
        <div v-show="configForm.connection_type === 'ssh'">
          <el-form-item label="SSH主机">
            <el-select
              v-model="configForm.ssh_host_id"
              placeholder="选择SSH主机或手动输入"
              filterable
              allow-create
              default-first-option
              @change="handleSshHostChange"
              clearable
            >
              <el-option
                v-for="host in sshHostList"
                :key="host.id"
                :label="`${host.name} (${host.address}:${host.port})`"
                :value="host.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="SSH 地址">
            <el-input v-model="configForm.ssh_addr" placeholder="SSH服务器地址，例如: 192.168.1.100" />
          </el-form-item>
          <el-form-item label="SSH 端口">
            <el-input v-model.number="configForm.ssh_port" type="number" placeholder="SSH端口，默认: 22" />
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
            <el-icon v-if="!testingConnection"><Connection /></el-icon>
            测试连接
          </el-button>
          <el-button v-if="!configForm.id" type="primary" @click="saveConfig">创建</el-button>
          <el-button v-else type="primary" @click="saveConfig">保存</el-button>
          <el-button @click="showConfigDialog = false">取消</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>

    <!-- 新建文件夹对话框 -->
    <el-dialog
        title="新建文件夹"
        v-model="showNewFolderDialogVisible"
        @close="currentNewFolderTabId = null"
    >
      <el-input v-model="newFolderName" placeholder="请输入文件夹名称" />
      <template #footer>
        <el-button @click="showNewFolderDialogVisible = false">取 消</el-button>
        <el-button type="primary" @click="createFolder">确 认</el-button>
      </template>
    </el-dialog>
    <!-- 右键菜单 - 使用 Teleport 渲染到 body，避免容器影响 -->
    <Teleport to="body">
      <ul class="tabs-context-menu" v-if="tabsContextMenuVisible" :style="tabsContextMenuStyle">
        <li @click="handleCloseOtherTabs">关闭其他</li>
        <li @click="handleCloseAllTabs">全部关闭</li>
      </ul>
    </Teleport>
    <!-- FTP 文件右键菜单 -->
    <Teleport to="body">
      <ul
          v-if="ftpFileContextMenuVisible"
          class="ftp-file-context-menu"
          :style="{
            top: ftpFileContextMenuPosition.y + 'px',
            left: ftpFileContextMenuPosition.x + 'px',
            position: 'fixed'
          }"
      >
        <li @click="handleFtpFileMenuOpen">
          {{ ftpFileContextMenuRow && ftpFileContextMenuRow.type === 'folder' ? '打开目录' : '预览 / 打开' }}
        </li>
        <li @click="handleFtpFileMenuDownload">
          {{ ftpFileContextMenuRow && ftpFileContextMenuRow.type === 'folder' ? '下载文件夹' : '下载' }}
        </li>
        <li class="danger" @click="handleFtpFileMenuDelete">
          删除
        </li>
      </ul>
    </Teleport>
  </div>
</template>

<script>
import {GetPortConfig} from "../../../wailsjs/go/controller/Server";
import axios from "axios";
import {BrowserOpenURL, OnFileDrop} from "../../../wailsjs/runtime";
import { UploadLocalPaths as FtpUploadLocalPaths } from '../../../wailsjs/go/service/FtpDesktopBridge';
import {ElMessage, ElMessageBox} from "element-plus";
import {Document, Folder, Upload, Edit, Delete, Connection, ArrowRight, ArrowLeft, Refresh} from "@element-plus/icons-vue";
import { getToken } from "../../utils/token";
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';
import { VueMonacoEditor } from '@guolao/vue-monaco-editor';

export default {
  name: "FTP",
  components: {
    Upload, 
    Folder, 
    Document, 
    Edit, 
    Delete, 
    Connection, 
    ArrowRight, 
    ArrowLeft,
    Refresh,
    VueMonacoEditor
  },
  setup() {
    return {
      Edit,
      Delete,
      Connection,
      ArrowRight,
      ArrowLeft
    };
  },
  data() {
    return {
      portConfig: null,
      apiClient: null, // FTP配置的API客户端
      configs: [],
      showConfigDialog: false,
      sidebarCollapsed: false,
      tabs: [],
      activeTabId: null,
      tabIdCounter: 0,
      fileInputRefs: {},
      folderInputRefs: {},
      // 标签页右键菜单
      tabsContextMenuVisible: false,
      tabsContextMenuPosition: { x: 0, y: 0 },
      tabsContextMenuTabId: '',
      // FTP 文件右键菜单
      ftpFileContextMenuVisible: false,
      ftpFileContextMenuPosition: { x: 0, y: 0 },
      ftpFileContextMenuRow: null,
      ftpFileContextMenuTabId: null,
      configForm: {
        id: null,
        name: '',
        host: '',
        port: 21,
        username: '',
        password: '',
        connection_type: '', // 连接方式：'direct' 或 'ssh'
        ssh_host_id: null, // 选择的SSH主机ID
        ssh_addr: '',
        ssh_port: 22,
        ssh_username: '',
        ssh_password: ''
      },
      sshApiClient: null, // SSH模块的API客户端
      sshHostList: [], // SSH主机列表
      testingConnection: false,
      testResult: null,
      showNewFolderDialogVisible: false,
      currentNewFolderTabId: null,
      newFolderName: '',
      // Wails 原生拖拽得到的本地绝对路径（用于 FTP 上传）
      wailsDropFtpPaths: [],
      // 文件上传策略：'ask' 询问, 'overwrite' 覆盖, 'skip' 跳过
      uploadStrategy: 'ask',
      // 是否对后续同名文件使用相同处理（覆盖/跳过）
      rememberUploadStrategy: false,
      // 文件预览相关
      filePreview: {
        file: null,
        loading: false,
        isText: false,
        error: '',
        content: '',
        editContent: '',
        dirty: false,
        tabId: null
      },
      filePreviewMonacoOptions: {
        minimap: { enabled: false },
        wordWrap: 'on',
        fontSize: 13,
        scrollbar: { verticalScrollbarSize: 8 },
        automaticLayout: true,
        scrollBeyondLastLine: false,
        lineNumbers: 'on',
        readOnly: false,
        tabSize: 2,
        insertSpaces: true
      },
    }
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
    },
    filePreviewLanguage() {
      const file = this.filePreview.file;
      if (!file || !file.name) return 'plaintext';
      const name = String(file.name).toLowerCase();
      if (name.endsWith('.sh')) return 'shell';
      if (name.endsWith('.js')) return 'javascript';
      if (name.endsWith('.ts')) return 'typescript';
      if (name.endsWith('.json')) return 'json';
      if (name.endsWith('.yml') || name.endsWith('.yaml')) return 'yaml';
      if (name.endsWith('.xml')) return 'xml';
      if (name.endsWith('.py')) return 'python';
      if (name.endsWith('.go')) return 'go';
      if (name.endsWith('.java')) return 'java';
      if (name.endsWith('.php')) return 'php';
      if (name.endsWith('.html') || name.endsWith('.htm')) return 'html';
      if (name.endsWith('.css') || name.endsWith('.scss') || name.endsWith('.less')) return 'css';
      if (name.endsWith('.md')) return 'markdown';
      return 'plaintext';
    },
    fileTableHeight() {
      // 如果没有预览文件，表格占满可用空间；如果有预览，使用固定高度
      if (!this.filePreview.file) {
        return null;
      }
      return 350;
    },
  },
  async mounted() {
    await this.initClient();
    await this.loadConfigs();
    await this.loadSshHostList();
    // 监听填充连接表单事件
    window.addEventListener('fill-connect-form', this.handleFillConnectForm);
    // 注册 FTP 全局拖拽监听（支持桌面拖拽绝对路径）
    this.setupFtpFileDropListener();
  },
  beforeUnmount() {
    window.removeEventListener('fill-connect-form', this.handleFillConnectForm);
  },
  methods: {
    // 注册 FTP 全局拖拽监听（支持桌面拖拽绝对路径）
    setupFtpFileDropListener() {
      OnFileDrop((x, y, paths) => {
        if (paths && paths.length > 0) {
          this.wailsDropFtpPaths = paths;
          // 找到当前活动的标签页
          const activeTab = this.tabs.find(tab => tab.id === this.activeTabId);
          if (activeTab) {
            this.startNativeFtpUpload(paths, activeTab.id);
          } else if (this.tabs.length > 0) {
            // 如果没有活动标签页，使用第一个
            this.startNativeFtpUpload(paths, this.tabs[0].id);
          } else {
            ElMessage.warning('请先选择一个 FTP 连接');
          }
        }
      });
    },
    // 检查 FTP 文件是否存在
    async checkFtpFileExists(tab, remotePath) {
      try {
        const port = parseInt(tab.config.port) || 21;
        // 获取文件所在目录
        const lastSlashIndex = remotePath.lastIndexOf('/');
        const dirPath = lastSlashIndex > 0 ? remotePath.substring(0, lastSlashIndex) : '/';
        const fileName = lastSlashIndex >= 0 ? remotePath.substring(lastSlashIndex + 1) : remotePath;
        
        const { data } = await axios.post(`http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/list`, {
          host: tab.config.host,
          port: port,
          username: tab.config.username,
          password: tab.config.password,
          path: dirPath,
          page: 1,
          pageSize: 1000,
          ssh_addr: tab.config.ssh_addr || '',
          ssh_port: parseInt(tab.config.ssh_port) || 0,
          ssh_username: tab.config.ssh_username || '',
          ssh_password: tab.config.ssh_password || ''
        });
        return data.files && data.files.some(f => f.name === fileName && f.type !== 'folder');
      } catch (err) {
        console.error('检查文件是否存在失败:', err);
        return false;
      }
    },
    // 处理文件存在的情况
    async handleFileExists(fileName, remotePath, tabId) {
      if (this.uploadStrategy === 'overwrite') {
        return 'overwrite';
      }
      if (this.uploadStrategy === 'skip') {
        return 'skip';
      }
      
      // 询问用户
      try {
        await ElMessageBox.confirm(
          `文件 "${fileName}" 已存在，请选择操作：`,
          '文件已存在',
          {
            distinguishCancelAndClose: true,
            confirmButtonText: '覆盖',
            cancelButtonText: '跳过',
            type: 'warning',
            autofocus: false
          }
        );
        // 用户点击“覆盖”
        if (this.rememberUploadStrategy) {
          // 记住本次选择，后续同名文件自动覆盖
          this.uploadStrategy = 'overwrite';
        }
        return 'overwrite';
      } catch (action) {
        if (action === 'cancel') {
          // 用户点击“跳过”
          if (this.rememberUploadStrategy) {
            // 记住本次选择，后续同名文件自动跳过
            this.uploadStrategy = 'skip';
          }
          return 'skip';
        }
        return 'cancel'; // 用户关闭对话框，取消上传
      }
    },
    // 使用 Wails 原生路径上传本地文件/文件夹到当前 FTP 目录
    async startNativeFtpUpload(paths, tabId) {
      const tab = this.getTab(tabId);
      if (!tab) {
        ElMessage.error('当前没有有效的 FTP 连接');
        return;
      }

      if (!paths || paths.length === 0) {
        return;
      }

      const targetDir = tab.currentPath || '/';
      this.uploadStrategy = 'ask'; // 重置策略

      try {
        tab.uploading = true;
        tab.uploadProgress = 0;

        // 使用 FTP DesktopBridge 服务上传
        const configId = String(tab.configId || tab.config.id || '');
        if (!configId) {
          ElMessage.error('无法获取 FTP 配置 ID');
          return;
        }

        const msg = await FtpUploadLocalPaths(configId, targetDir, paths);
        ElMessage.success(msg || '本地文件上传完成');

        // 刷新目录
        await this.loadFtpFileList(tabId);
      } catch (err) {
        const errMsg = err?.message || String(err || '');
        ElMessage.error(errMsg || '本地路径上传失败');
      } finally {
        tab.uploading = false;
        tab.uploadProgress = 0;
        this.wailsDropFtpPaths = [];
      }
    },
    normalizeConfig(config) {
      if (!config) return null;
      const normalized = {
        id: String(config.id ?? Date.now()),
        name: config.name || '',
        host: config.host || '',
        port: parseInt(config.port) || 21,
        username: config.username || '',
        password: config.password || '',
        connection_type: config.connection_type || '',
        ssh_host_id: config.ssh_host_id || null,
        ssh_addr: config.ssh_addr || '',
        ssh_port: parseInt(config.ssh_port) || 22,
        ssh_username: config.ssh_username || '',
        ssh_password: config.ssh_password || ''
      };
      // 不要自动设置connection_type，保持用户选择的值（或空值）
      // 如果connection_type未定义，保持为空字符串
      if (normalized.connection_type === undefined || normalized.connection_type === null) {
        normalized.connection_type = '';
      }
      return normalized;
    },
    async initClient() {
      await this.loadPortConfig();
      this.createHttpClient();
      this.createSshHttpClient();
    },
    async loadPortConfig() {
      try {
        this.portConfig = await GetPortConfig();
        console.log('端口配置:', this.portConfig);
      } catch (error) {
        console.error('获取端口配置失败:', error);
        this.portConfig = {
          sshPort: 52868,
          ftpPort: 52869,
        };
      }
    },
    createHttpClient() {
      const port = this.portConfig?.ftpPort || 52869;
      const host = '127.0.0.1';
      const baseURL = `http://${host}:${port}`;
      const client = axios.create({
        baseURL: baseURL,
        timeout: 20000,
      });
      this.apiClient = client;
    },
    createSshHttpClient() {
      const port = this.portConfig?.sshPort || 52868;
      const host = '127.0.0.1';
      const baseURL = `http://${host}:${port}`;
      const client = axios.create({
        baseURL: baseURL,
        // 通过 SSH 端口转发访问远端主机，涉及文件操作时也可能较耗时
        timeout: 600000,
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
    handleConnectionTypeChange(connectionType) {
      // 当连接方式改变时，如果是直连，清空SSH配置
      if (connectionType !== 'ssh') {
        this.configForm.ssh_host_id = null;
        this.configForm.ssh_addr = '';
        this.configForm.ssh_port = 22;
        this.configForm.ssh_username = '';
        this.configForm.ssh_password = '';
      }
    },
    handleSshHostChange(hostId) {
      // 当选择SSH主机时，自动填充SSH配置
      if (hostId) {
        // 尝试转换为数字，如果是数字则从列表查找
        const numericId = typeof hostId === 'string' ? parseInt(hostId, 10) : hostId;
        if (!isNaN(numericId) && typeof numericId === 'number') {
          const host = this.sshHostList.find(h => h.id === numericId);
          if (host) {
            this.configForm.ssh_addr = host.address || '';
            this.configForm.ssh_port = parseInt(host.port) || 22;
            this.configForm.ssh_username = host.user || '';
            this.configForm.ssh_password = host.pwd || '';
            return;
          }
        }
      }
    },
    async loadConfigs() {
      if (!this.apiClient) {
        console.error('API客户端未初始化');
        return;
      }
      try {
        const response = await this.apiClient.get('/api/config');
        if (response.data.code === 0) {
          this.configs = (response.data.data || []).map(this.normalizeConfig);
        }
      } catch (e) {
        console.error('加载配置失败:', e);
        this.configs = [];
      }
    },
    async testConnection() {
      if (!this.configForm.host) {
        ElMessage.warning('请输入主机地址');
        return;
      }
      
      // 如果选择了SSH隧道，必须验证SSH配置完整性
      if (this.configForm.connection_type === 'ssh') {
        if (!this.configForm.ssh_addr || this.configForm.ssh_addr.trim() === '') {
          ElMessage.warning('SSH隧道模式下，SSH地址不能为空');
          return;
        }
        if (!this.configForm.ssh_username || this.configForm.ssh_username.trim() === '') {
          ElMessage.warning('SSH隧道模式下，SSH用户名不能为空');
          return;
        }
        if (!this.configForm.ssh_password || this.configForm.ssh_password.trim() === '') {
          ElMessage.warning('SSH隧道模式下，SSH密码不能为空');
          return;
        }
        // 如果SSH端口为空，设置默认值
        if (!this.configForm.ssh_port || this.configForm.ssh_port === 0) {
          this.configForm.ssh_port = 22;
        }
      }
      
      this.testingConnection = true;
      this.testResult = null;
      
      try {
        // 确保port是整数类型
        const port = parseInt(this.configForm.port) || 21;
        await axios.post(`http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/connect`, {
          host: this.configForm.host,
          port: port,
          username: this.configForm.username,
          password: this.configForm.password,
          ssh_addr: this.configForm.ssh_addr || '',
          ssh_port: parseInt(this.configForm.ssh_port) || 0,
          ssh_username: this.configForm.ssh_username || '',
          ssh_password: this.configForm.ssh_password || ''
        });
        
        this.testResult = {
          success: true,
          message: '连接测试成功！'
        };
        ElMessage.success('连接测试成功');
      } catch (e) {
        const errorMsg = e.response?.data?.error || e.message || '连接测试失败';
        this.testResult = {
          success: false,
          message: errorMsg
        };
        ElMessage.error('连接测试失败: ' + errorMsg);
      } finally {
        this.testingConnection = false;
      }
    },
    async saveConfig() {
      if (!this.apiClient) {
        ElMessage.error('API客户端未初始化');
        return;
      }
      
      if (!this.configForm.name || !this.configForm.host) {
        ElMessage.warning('请填写名称和主机地址');
        return;
      }
      
      try {
        // 构建要保存的配置对象
        const configToSave = {
          name: (this.configForm.name || '').trim(),
          host: (this.configForm.host || '').trim(),
          port: String(parseInt(this.configForm.port) || 21),
          username: (this.configForm.username || '').trim(),
          password: this.configForm.password || '',
          connection_type: this.configForm.connection_type || '',
          ssh_host_id: this.configForm.ssh_host_id || null,
          ssh_addr: '',
          ssh_port: '',
          ssh_username: '',
          ssh_password: ''
        };

        // 如果连接方式是SSH，设置SSH配置
        if (configToSave.connection_type === 'ssh') {
          configToSave.ssh_addr = (this.configForm.ssh_addr || '').trim();
          configToSave.ssh_port = String(parseInt(this.configForm.ssh_port) || 22);
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

        const method = this.configForm.id ? 'put' : 'post';
        const response = await this.apiClient[method]('/api/config', configToSave);
        
        if (response.data.code === 0) {
          ElMessage.success(this.configForm.id ? '更新成功' : '创建成功');
          this.showConfigDialog = false;
          this.testResult = null;
          this.configForm = {
            id: null,
            name: '',
            host: '',
            port: 21,
            username: '',
            password: '',
            connection_type: '',
            ssh_host_id: null,
            ssh_addr: '',
            ssh_port: 22,
            ssh_username: '',
            ssh_password: ''
          };
          await this.loadConfigs();
        } else {
          ElMessage.error(response.data.msg || '保存失败');
        }
      } catch (e) {
        const errorMsg = e.response?.data?.msg || e.message || '保存失败';
        ElMessage.error('保存失败: ' + errorMsg);
        console.error('保存配置失败:', e);
      }
    },
    editConfig(config) {
      this.configForm = this.normalizeConfig(config);
      // 保持用户选择的值（或空值），不要自动设置connection_type
      // 如果connection_type未定义，保持为空字符串
      if (this.configForm.connection_type === undefined || this.configForm.connection_type === null) {
        this.configForm.connection_type = '';
      }
      this.testResult = null;
      this.showConfigDialog = true;
      // 加载SSH主机列表
      this.loadSshHostList();
    },
    // 处理填充连接表单事件
    handleFillConnectForm(event) {
      const { module, connectionInfo } = event.detail || {};
      // 只处理FTP模块的请求
      if (module !== 'ftp' || !connectionInfo) {
        return;
      }
      
      // 先填充表单数据（在打开对话框之前）
      if (connectionInfo.host) {
        this.configForm.host = connectionInfo.host;
      }
      if (connectionInfo.port) {
        this.configForm.port = parseInt(connectionInfo.port) || 21;
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
      } else if (connectionInfo.host && connectionInfo.port) {
        // 如果没有名称，自动生成
        this.configForm.name = `FTP - ${connectionInfo.host}:${connectionInfo.port}`;
      }
      
      // 打开对话框
      this.showConfigDialog = true;
      
      // 等待对话框完全打开后，再次确认数据
      this.$nextTick(() => {
        // 加载SSH主机列表
        this.loadSshHostList();
        
        // 再次填充数据，确保不被其他逻辑覆盖
        setTimeout(() => {
          if (connectionInfo.host) {
            this.configForm.host = connectionInfo.host;
          }
          if (connectionInfo.port) {
            this.configForm.port = parseInt(connectionInfo.port) || 21;
          }
          // 重要：账号密码需要明确设置
          if (connectionInfo.username !== undefined) {
            this.configForm.username = connectionInfo.username;
          }
          if (connectionInfo.password !== undefined) {
            this.configForm.password = connectionInfo.password;
          }
          if (connectionInfo.name) {
            this.configForm.name = connectionInfo.name;
          } else if (connectionInfo.host && connectionInfo.port) {
            this.configForm.name = `FTP - ${connectionInfo.host}:${connectionInfo.port}`;
          }
        }, 150);
      });
    },
    handleConfigDialogClose() {
      this.testResult = null;
      this.configForm = {
        id: null,
        name: '',
        host: '',
        port: 21,
        username: '',
        password: '',
        connection_type: '',
        ssh_host_id: null,
        ssh_addr: '',
        ssh_port: 22,
        ssh_username: '',
        ssh_password: ''
      };
    },
    async deleteConfig(configId) {
      if (!this.apiClient) {
        ElMessage.error('API客户端未初始化');
        return;
      }

      try {
        await confirmDangerousAction({
          title: '删除连接',
          message: '将删除该 FTP 连接配置，不可恢复。',
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }

      try {
        await this.apiClient.delete(`/api/config/${configId}`);
        ElMessage.success('删除成功');
        
        // 删除相关的标签页
        const tabsToRemove = this.tabs.filter(tab => tab.configId === configId);
        tabsToRemove.forEach(tab => {
          const idx = this.tabs.findIndex(t => t.id === tab.id);
          if (idx > -1) {
            this.tabs.splice(idx, 1);
          }
        });
        if (this.tabs.length > 0) {
          this.activeTabId = this.tabs[0].id;
        } else {
          this.activeTabId = null;
        }
        
        await this.loadConfigs();
      } catch (e) {
        const errorMsg = e.response?.data?.msg || e.message || '删除失败';
        ElMessage.error('删除失败: ' + errorMsg);
        console.error('删除配置失败:', e);
      }
    },
    isConfigActive(configId) {
      return this.tabs.some(tab => tab.configId === configId && tab.id === this.activeTabId);
    },
    async handleConfigSelect(configId) {
      const config = this.configs.find(c => c.id === configId);
      if (!config) return;
      
      // 检查是否已存在该名称的标签页
      const existingTab = this.tabs.find(tab => tab.label === config.name);
      if (existingTab) {
        // 切换到已存在的标签页
        this.activeTabId = existingTab.id;
        return;
      }
      
      // 测试连接
      const msg = ElMessage.info('正在连接中，请稍等...');
      try {
        // 确保port是整数类型
        const port = parseInt(config.port) || 21;
        await axios.post(`http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/connect`, {
          host: config.host,
          port: port,
          username: config.username,
          password: config.password,
          ssh_addr: config.ssh_addr || '',
          ssh_port: parseInt(config.ssh_port) || 0,
          ssh_username: config.ssh_username || '',
          ssh_password: config.ssh_password || ''
        });
        
        // 创建新标签页，使用名称作为标签
        const tabId = `tab_${++this.tabIdCounter}`;
        const newTab = {
          id: tabId,
          configId: configId,
          label: config.name,
          config: { 
            ...config,
            port: parseInt(config.port) || 21  // 确保port是整数
          },
          currentPath: '',
          fileList: [],
          currentPage: 1,
          pageSize: 10,
          totalFiles: 0,
          uploading: false,
          uploadProgress: 0,
          // 是否已请求取消当前上传任务（仅影响批量上传）
          cancelUploadRequested: false,
          downloadingFile: '',
          downloadProgress: 0
        };
        
        this.tabs.push(newTab);
        this.activeTabId = tabId;
        await this.loadFtpFileList(tabId);
        msg.close();
        ElMessage.success('连接成功');
      } catch (err) {
        msg.close();
        ElMessage.error('连接失败: ' + (err.response?.data?.error || err.message));
      }
    },
    getTab(tabId) {
      return this.tabs.find(t => t.id === tabId);
    },
    async loadFtpFileList(tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return;
      
      try {
        // 确保port是整数类型
        const port = parseInt(tab.config.port) || 21;
        const { data } = await axios.post(`http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/list`, {
          host: tab.config.host,
          port: port,
          username: tab.config.username,
          password: tab.config.password,
          path: tab.currentPath,
          page: tab.currentPage,
          pageSize: tab.pageSize,
          ssh_addr: tab.config.ssh_addr || '',
          ssh_port: parseInt(tab.config.ssh_port) || 0,
          ssh_username: tab.config.ssh_username || '',
          ssh_password: tab.config.ssh_password || ''
        });
        tab.fileList = data.files.map(f => ({ ...f, type: f.type === 'folder' ? 'folder' : 'file' }));
        tab.totalFiles = data.pagination.total;
      } catch (err) {
        ElMessage.error('获取列表失败: ' + err.message);
      }
    },
    onPageChange(page, tabId) {
      const tab = this.getTab(tabId);
      if (tab) {
        tab.currentPage = page;
        this.loadFtpFileList(tabId);
      }
    },
    onSizeChange(size, tabId) {
      const tab = this.getTab(tabId);
      if (tab) {
        tab.pageSize = size;
        tab.currentPage = 1;
        this.loadFtpFileList(tabId);
      }
    },
    enterFtpFolder(name, tabId) {
      const tab = this.getTab(tabId);
      if (tab) {
        tab.currentPath += '/' + name;
        this.loadFtpFileList(tabId);
      }
    },
    getTabPathItems(tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return ['根目录'];
      const parts = tab.currentPath.split('/').filter(p => p);
      return ['根目录', ...parts];
    },
    ftpNavigateTo(idx, tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return;
      
      if (idx === 0) {
        tab.currentPath = '';
      } else {
        const parts = this.getTabPathItems(tabId).slice(1, idx + 1);
        tab.currentPath = '/' + parts.join('/');
      }
      this.loadFtpFileList(tabId);
    },
    deleteFtpFile(name, tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return;
      
      // 确保port是整数类型
      const port = parseInt(tab.config.port) || 21;
      axios.post(`http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/delete`, {
        host: tab.config.host,
        port: port,
        username: tab.config.username,
        password: tab.config.password,
        path: `${tab.currentPath}/${name}`,
        ssh_addr: tab.config.ssh_addr || '',
        ssh_port: parseInt(tab.config.ssh_port) || 0,
        ssh_username: tab.config.ssh_username || '',
        ssh_password: tab.config.ssh_password || ''
      })
      .then(() => {
        ElMessage.success('删除成功');
        this.loadFtpFileList(tabId);
      })
      .catch(err => ElMessage.error('删除失败: ' + err.message));
    },
    async FtpDownloadFile(name, tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return;
      
      tab.downloadingFile = name;
      try {
        // 确保port是整数类型
        const port = parseInt(tab.config.port) || 21;
        const res = await axios.post(
            `http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/generate-download-url`,
            {
              host: tab.config.host,
              port: port,
              username: tab.config.username,
              password: tab.config.password,
              path: `${tab.currentPath}/${name}`,
              ssh_addr: tab.config.ssh_addr || '',
              ssh_port: parseInt(tab.config.ssh_port) || 0,
              ssh_username: tab.config.ssh_username || '',
              ssh_password: tab.config.ssh_password || ''
            }
        );

        const downloadUrl = res.data.downloadUrl;
        BrowserOpenURL(downloadUrl);

        ElMessage.success('已在浏览器中开始下载');
      } catch (err) {
        ElMessage.error('下载失败: ' + err.message);
      } finally {
        tab.downloadingFile = '';
      }
    },
    setFileInputRef(el, tabId) {
      if (el) {
        this.fileInputRefs[tabId] = el;
      }
    },
    setFolderInputRef(el, tabId) {
      if (el) {
        this.folderInputRefs[tabId] = el;
      }
    },
    triggerFileInput(tabId) {
      const input = this.fileInputRefs[tabId];
      if (input) {
        input.click();
      }
    },
    triggerFolderInput(tabId) {
      const input = this.folderInputRefs[tabId];
      if (input) {
        input.click();
      }
    },
    onFileSelected(event, tabId) {
      const file = event.target.files[0];
      if (!file) return;
      this.uploadFile(file, tabId);
      event.target.value = null;
    },
    onFolderSelected(event, tabId) {
      const files = Array.from(event.target.files || []);
      if (files.length === 0) return;
      
      // 处理文件夹上传，保持文件夹结构
      const collectedFiles = files.map(file => {
        // 使用 webkitRelativePath 保持文件夹结构
        const relativePath = file.webkitRelativePath || file.name;
        // eslint-disable-next-line no-param-reassign
        file.relativePath = relativePath;
        return file;
      });
      
      // 检查是否包含目录结构
      const hasDirectory = collectedFiles.some(f => f.relativePath && f.relativePath.includes('/'));
      
      this.uploadFtpFilesWithStructure(collectedFiles, tabId, hasDirectory);
      event.target.value = null;
    },
    async uploadFile(file, tabId) {
      const tab = this.getTab(tabId);
      if (!tab) return;
      
      // 确保port是整数类型
      const port = parseInt(tab.config.port) || 21;
      const form = new FormData();
      // 为本次上传生成一个进度 ID，用于后端统计
      const uploadId = `ftp-${Date.now()}-${Math.random().toString(16).slice(2)}`;
      form.append('file', file);
      form.append('host', tab.config.host);
      form.append('port', String(port));
      form.append('username', tab.config.username);
      form.append('password', tab.config.password);
      form.append('remotePath', `${tab.currentPath}/${file.name}`);
      form.append('ssh_addr', tab.config.ssh_addr || '');
      form.append('ssh_port', String(parseInt(tab.config.ssh_port) || 0));
      form.append('ssh_username', tab.config.ssh_username || '');
      form.append('ssh_password', tab.config.ssh_password || '');
      form.append('uploadId', uploadId);

      tab.uploading = true;
      tab.uploadProgress = 0;
      let pollingTimer = null;

      try {
        // 启动轮询后端真实进度
        pollingTimer = window.setInterval(async () => {
          try {
            const res = await axios.get(
              `http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/upload/progress`,
              { params: { id: uploadId } }
            );
            const data = res.data || {};
            if (!data.exists) {
              return;
            }
            const total = Number(data.total || 0);
            const uploaded = Number(data.uploaded || 0);
            let percent = 0;
            if (total > 0) {
              percent = Math.floor((uploaded / total) * 100);
            }
            if (percent < 0) percent = 0;
            if (percent > 100) percent = 100;
            tab.uploadProgress = percent;
            if (data.done) {
              if (pollingTimer) {
                window.clearInterval(pollingTimer);
                pollingTimer = null;
              }
            }
          } catch (e) {
            // 忽略单次进度查询错误，不影响整体上传
          }
        }, 500);

        await axios.post(
          `http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/upload`,
          form,
          {
            headers: { 'Content-Type': 'multipart/form-data' }
          }
        );
        // 请求真正成功返回后，如果进度还没到 100，则强制置为 100
        if (tab.uploadProgress < 100) {
          tab.uploadProgress = 100;
        }
        ElMessage.success('上传成功');
        await this.loadFtpFileList(tabId);
      } catch (err) {
        ElMessage.error('上传失败：' + err.message);
      } finally {
        if (pollingTimer) {
          window.clearInterval(pollingTimer);
          pollingTimer = null;
        }
        tab.uploading = false;
      }
    },
    showNewFolderDialog(tabId) {
      this.currentNewFolderTabId = tabId;
      this.newFolderName = '';
      this.showNewFolderDialogVisible = true;
    },
    async createFolder() {
      if (!this.newFolderName.trim()) {
        return ElMessage.warning('请输入文件夹名称');
      }
      
      const tab = this.getTab(this.currentNewFolderTabId);
      if (!tab) return;
      
      try {
        // 确保port是整数类型
        const port = parseInt(tab.config.port) || 21;
        await axios.post(`http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/mkdir`, {
          host: tab.config.host,
          port: port,
          username: tab.config.username,
          password: tab.config.password,
          ssh_addr: tab.config.ssh_addr || '',
          ssh_port: parseInt(tab.config.ssh_port) || 0,
          ssh_username: tab.config.ssh_username || '',
          ssh_password: tab.config.ssh_password || '',
          path: tab.currentPath,
          folderName: this.newFolderName
        });
        ElMessage.success('创建成功');
        this.showNewFolderDialogVisible = false;
        this.newFolderName = '';
        this.currentNewFolderTabId = null;
        await this.loadFtpFileList(tab.id);
      } catch (err) {
        ElMessage.error('创建失败：' + err.message);
      }
    },
    /**
     * 处理 FTP 文件区域的拖拽上传（支持文件和文件夹）
     */
    async handleFtpFileAreaDrop(event, tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      const items = event.dataTransfer?.items
      const fileList = event.dataTransfer?.files
      if ((!items || items.length === 0) && (!fileList || fileList.length === 0)) {
        return
      }

      // 收集文件（包含相对路径，用于还原文件夹结构）
      const collectedFiles = []
      // 额外收集目录路径，便于处理“空目录”场景
      const collectedDirs = new Set()

      const processEntry = (entry, parentPath = '') => {
        return new Promise((resolve) => {
          if (entry.isFile) {
            entry.file((file) => {
              const relativePath = parentPath ? `${parentPath}/${file.name}` : file.name
              // 自定义相对路径字段
              // eslint-disable-next-line no-param-reassign
              file.relativePath = relativePath
              collectedFiles.push(file)
              resolve()
            })
          } else if (entry.isDirectory) {
            const reader = entry.createReader()
            const dirPath = parentPath ? `${parentPath}/${entry.name}` : entry.name
            // 记录目录路径，即使该目录最终没有任何文件（空目录）
            if (dirPath) {
              collectedDirs.add(dirPath)
            }
            const readEntries = () => {
              reader.readEntries((entries) => {
                if (!entries || entries.length === 0) {
                  resolve()
                } else {
                  Promise.all(entries.map((e) => processEntry(e, dirPath))).then(() => {
                    readEntries()
                  })
                }
              })
            }
            readEntries()
          } else {
            resolve()
          }
        })
      }

      const tasks = []
      if (items && items.length > 0 && items[0].webkitGetAsEntry) {
        for (let i = 0; i < items.length; i += 1) {
          const item = items[i]
          if (item.kind === 'file') {
            const entry = item.webkitGetAsEntry && item.webkitGetAsEntry()
            if (entry) {
              tasks.push(processEntry(entry))
            } else {
              const file = item.getAsFile && item.getAsFile()
              if (file) {
                collectedFiles.push(file)
              }
            }
          }
        }
      } else if (fileList && fileList.length > 0) {
        const files = Array.from(fileList)
        const allMaybeFolderPlaceholder =
            files.length === 1 &&
            files[0].size === 0 &&
            !files[0].type &&
            !files[0].name.includes('.')

        if (allMaybeFolderPlaceholder) {
          // 某些环境下，拖拽空文件夹会表现为一个 size=0、无类型的占位“文件”
          // 此时我们将其视为一个空目录，在当前路径下直接创建同名文件夹
          const folderName = files[0].name
          const basePath = (tab.currentPath || '').replace(/\/+/g, '/').replace(/\/+$/, '')
          const fullDir = basePath ? `${basePath}/${folderName}` : folderName
          try {
            await this.ensureFtpDirectoryExists(tab, fullDir)
            ElMessage.success('已创建空文件夹')
            await this.loadFtpFileList(tabId)
          } catch (e) {
            ElMessage.error('创建空文件夹失败：' + (e.message || e))
          }
          return
        }

        files.forEach(f => collectedFiles.push(f))
      }

      if (tasks.length > 0) {
        await Promise.all(tasks)
      }

      // 如果没有实际文件但包含目录，则认为是拖拽“空文件夹”，直接在当前目录创建这些空目录
      if (collectedFiles.length === 0 && collectedDirs.size > 0) {
        try {
          const basePath = (tab.currentPath || '').replace(/\/+/g, '/').replace(/\/+$/, '')
          const tasksCreate = Array.from(collectedDirs).map(dir => {
            const fullDir = basePath ? `${basePath}/${dir}` : dir
            return this.ensureFtpDirectoryExists(tab, fullDir)
          })
          await Promise.all(tasksCreate)
          ElMessage.success('已创建空文件夹')
          await this.loadFtpFileList(tabId)
        } catch (e) {
          ElMessage.error('创建空文件夹失败：' + (e.message || e))
        }
        return
      }

      if (collectedFiles.length === 0) {
        return
      }

      // 标记是否包含目录结构
      const hasDirectory = collectedFiles.some(f => f.relativePath && f.relativePath.includes('/'))

      try {
        await this.uploadFtpFilesWithStructure(collectedFiles, tabId, hasDirectory)
      } catch (err) {
        ElMessage.error('拖拽上传失败：' + (err.message || err))
      }
    },
    // 取消当前标签页的批量上传
    cancelFtpUpload(tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return
      if (tab.uploading && !tab.cancelUploadRequested) {
        tab.cancelUploadRequested = true
        ElMessage.info('已请求取消上传，将在当前文件上传完成后停止剩余文件')
      }
    },
    /**
     * 确保 FTP 目录存在（递归创建多级目录）
     * fullDirPath 形如："", "/a", "a/b", "/a/b"
     */
    async ensureFtpDirectoryExists(tab, fullDirPath) {
      // 统一去掉前后多余的 /
      const cleaned = (fullDirPath || '').replace(/\/+/g, '/').replace(/^\/+/, '').replace(/\/+$/, '')
      if (!cleaned) return

      const segments = cleaned.split('/').filter(p => p)
      let parent = ''

      // 确保端口为整数
      const port = parseInt(tab.config.port) || 21

      for (let i = 0; i < segments.length; i += 1) {
        const folderName = segments[i]
        try {
          await axios.post(`http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/mkdir`, {
            host: tab.config.host,
            port: port,
            username: tab.config.username,
            password: tab.config.password,
            ssh_addr: tab.config.ssh_addr || '',
            ssh_port: parseInt(tab.config.ssh_port) || 0,
            ssh_username: tab.config.ssh_username || '',
            ssh_password: tab.config.ssh_password || '',
            path: parent,
            folderName
          })
        } catch (e) {
          // 目录可能已存在，忽略错误
        }
        parent = parent ? `${parent}/${folderName}` : folderName
      }
    },
    /**
     * 批量上传 FTP 文件，支持还原文件夹结构
     */
    async uploadFtpFilesWithStructure(files, tabId, hasDirectory) {
      const tab = this.getTab(tabId)
      if (!tab) return

      // 构造带相对路径的列表
      const fileItems = files.map(f => ({
        file: f,
        relativePath: f.relativePath || f.webkitRelativePath || f.name
      }))

      if (!fileItems.length) {
        ElMessage.warning('没有可上传的文件')
        return
      }

      // 计算总大小
      let totalSize = 0
      fileItems.forEach(item => {
        totalSize += item.file.size || 0
      })

      // 先收集需要创建的目录（相对于 FTP 根目录）
      const dirsToCreate = new Set()
      fileItems.forEach(item => {
        const parts = String(item.relativePath).split('/').filter(p => p)
        if (parts.length > 1) {
          const subDirs = parts.slice(0, -1)
          let current = ''
          subDirs.forEach(seg => {
            current = current ? `${current}/${seg}` : seg
            dirsToCreate.add(current)
          })
        }
      })

      // 把当前路径也加进去作为前缀
      const basePath = (tab.currentPath || '').replace(/\/+/g, '/').replace(/\/+$/, '')

      // 先创建目录
      if (hasDirectory && dirsToCreate.size > 0) {
        const tasks = Array.from(dirsToCreate).sort().map(dir => {
          const fullDir = basePath ? `${basePath}/${dir}` : dir
          return this.ensureFtpDirectoryExists(tab, fullDir)
        })
        await Promise.all(tasks)
      }

      // 按顺序上传文件
      tab.uploading = true
      tab.uploadProgress = 0
      tab.cancelUploadRequested = false

      let uploadedBytes = 0
      let uploaded = 0
      let skipped = 0
      let failed = 0
      const totalFiles = fileItems.length
      const port = parseInt(tab.config.port) || 21
      this.uploadStrategy = 'ask' // 重置策略

      for (let i = 0; i < fileItems.length; i += 1) {
        // 如果用户请求取消上传，则中断后续文件
        if (tab.cancelUploadRequested) {
          console.warn('用户取消上传，停止剩余文件上传')
          break
        }
        const item = fileItems[i]
        const rel = String(item.relativePath || item.file.name)
        const parts = rel.split('/').filter(p => p)
        const fileName = parts.pop() || item.file.name
        const subDir = parts.length ? parts.join('/') : ''

        const dirPath = basePath
            ? (subDir ? `${basePath}/${subDir}` : basePath)
            : subDir

        const cleanedDir = (dirPath || '').replace(/\/+/g, '/').replace(/\/+$/, '')
        const remotePath = cleanedDir ? `${cleanedDir}/${fileName}` : `/${fileName}`

        // 检查文件是否存在
        const exists = await this.checkFtpFileExists(tab, remotePath)
        if (exists) {
          const action = await this.handleFileExists(fileName, remotePath, tabId)
          if (action === 'cancel') {
            // 用户取消，停止上传
            break
          } else if (action === 'skip') {
            skipped++
            continue
          }
          // action === 'overwrite' 继续上传
        }

        const form = new FormData()
        form.append('file', item.file)
        form.append('host', tab.config.host)
        form.append('port', String(port))
        form.append('username', tab.config.username)
        form.append('password', tab.config.password)
        form.append('remotePath', remotePath)
        form.append('ssh_addr', tab.config.ssh_addr || '')
        form.append('ssh_port', String(parseInt(tab.config.ssh_port) || 0))
        form.append('ssh_username', tab.config.ssh_username || '')
        form.append('ssh_password', tab.config.ssh_password || '')

        let lastLoaded = 0

        try {
          // eslint-disable-next-line no-await-in-loop
          await axios.post(
              `http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/upload`,
              form,
              {
                headers: { 'Content-Type': 'multipart/form-data' },
                onUploadProgress: e => {
                  const { loaded, total } = e
                  if (!total || !totalSize) return
                  uploadedBytes += (loaded - lastLoaded)
                  lastLoaded = loaded
                  let percent = Math.floor((uploadedBytes / totalSize) * 100)
                  // 批量上传过程中，进度最多显示到 99%，只在所有文件上传请求成功后再显示 100%
                  if (percent >= 100) {
                    percent = 99
                  }
                  if (percent < 0) percent = 0
                  if (percent > 99) percent = 99
                  tab.uploadProgress = percent || Math.floor(((i + loaded / total) / totalFiles) * 100)
                }
              }
          )
          uploaded++
        } catch (err) {
          console.error(`上传文件失败: ${fileName}`, err)
          failed++
        }
      }

      tab.uploadProgress = 100
      // 显示上传结果
      if (failed === 0 && skipped === 0) {
        ElMessage.success(`成功上传 ${uploaded} 个文件`)
      } else {
        ElMessage.warning(`上传完成：成功 ${uploaded} 个，跳过 ${skipped} 个，失败 ${failed} 个`)
      }
      await this.loadFtpFileList(tabId)
      setTimeout(() => {
        tab.uploadProgress = 0
      }, 1500)
      tab.uploading = false
    },
    // FTP 文件行右键菜单
    handleFtpRowContextMenu(row, column, event) {
      event.preventDefault()
      event.stopPropagation()
      this.ftpFileContextMenuRow = row
      this.ftpFileContextMenuTabId = this.activeTabId

      const x = event.clientX || 0
      const y = event.clientY || 0
      this.ftpFileContextMenuPosition = { x, y }
      this.ftpFileContextMenuVisible = true

      const hideMenu = (e) => {
        if (!e.target.closest('.ftp-file-context-menu')) {
          this.hideFtpFileContextMenu()
          document.removeEventListener('click', hideMenu)
          document.removeEventListener('contextmenu', hideMenu)
        }
      }
      setTimeout(() => {
        document.addEventListener('click', hideMenu)
        document.addEventListener('contextmenu', hideMenu)
      }, 50)
    },
    hideFtpFileContextMenu() {
      this.ftpFileContextMenuVisible = false
      this.ftpFileContextMenuRow = null
      this.ftpFileContextMenuTabId = null
    },
    // 右键菜单：打开 / 进入目录
    handleFtpFileMenuOpen() {
      const row = this.ftpFileContextMenuRow
      const tabId = this.ftpFileContextMenuTabId
      if (!row || !tabId) return
      if (row.type === 'folder') {
        this.enterFtpFolder(row.name, tabId)
      } else {
        this.openFtpFilePreview(row, tabId)
      }
      this.hideFtpFileContextMenu()
    },
    // 右键菜单：下载
    handleFtpFileMenuDownload() {
      const row = this.ftpFileContextMenuRow
      const tabId = this.ftpFileContextMenuTabId
      if (!row || !tabId) return
      if (row.type === 'folder') {
        this.FtpDownloadDirectory(row.name, tabId)
      } else {
        this.FtpDownloadFile(row.name, tabId)
      }
      this.hideFtpFileContextMenu()
    },
    // 右键菜单：删除
    handleFtpFileMenuDelete() {
      const row = this.ftpFileContextMenuRow
      const tabId = this.ftpFileContextMenuTabId
      if (!row || !tabId) return
      this.deleteFtpFile(row.name, tabId)
      this.hideFtpFileContextMenu()
    },
    /**
     * 处理 FTP 文件区域的拖拽上传（支持文件和文件夹）
     */
    async handleFtpFileAreaDrop(event, tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      const items = event.dataTransfer?.items
      const fileList = event.dataTransfer?.files
      if ((!items || items.length === 0) && (!fileList || fileList.length === 0)) {
        return
      }

      // 收集文件（包含相对路径，用于还原文件夹结构）
      const collectedFiles = []

      const processEntry = (entry, parentPath = '') => {
        return new Promise((resolve) => {
          if (entry.isFile) {
            entry.file((file) => {
              const relativePath = parentPath ? `${parentPath}/${file.name}` : file.name
              // 自定义相对路径字段
              // eslint-disable-next-line no-param-reassign
              file.relativePath = relativePath
              collectedFiles.push(file)
              resolve()
            })
          } else if (entry.isDirectory) {
            const reader = entry.createReader()
            const dirPath = parentPath ? `${parentPath}/${entry.name}` : entry.name
            const readEntries = () => {
              reader.readEntries((entries) => {
                if (!entries || entries.length === 0) {
                  resolve()
                } else {
                  Promise.all(entries.map((e) => processEntry(e, dirPath))).then(() => {
                    readEntries()
                  })
                }
              })
            }
            readEntries()
          } else {
            resolve()
          }
        })
      }

      const tasks = []
      if (items && items.length > 0 && items[0].webkitGetAsEntry) {
        for (let i = 0; i < items.length; i += 1) {
          const item = items[i]
          if (item.kind === 'file') {
            const entry = item.webkitGetAsEntry && item.webkitGetAsEntry()
            if (entry) {
              tasks.push(processEntry(entry))
            } else {
              const file = item.getAsFile && item.getAsFile()
              if (file) {
                collectedFiles.push(file)
              }
            }
          }
        }
      } else if (fileList && fileList.length > 0) {
        const files = Array.from(fileList)
        const allMaybeFolderPlaceholder =
            files.length === 1 &&
            files[0].size === 0 &&
            !files[0].type &&
            !files[0].name.includes('.')

        if (allMaybeFolderPlaceholder) {
          ElMessage.warning('当前浏览器对“直接拖拽整个文件夹”支持不佳，请尝试在支持的浏览器中使用')
          return
        }

        files.forEach(f => collectedFiles.push(f))
      }

      if (tasks.length > 0) {
        await Promise.all(tasks)
      }

      if (collectedFiles.length === 0) {
        return
      }

      // 标记是否包含目录结构
      const hasDirectory = collectedFiles.some(f => f.relativePath && f.relativePath.includes('/'))

      try {
        await this.uploadFtpFilesWithStructure(collectedFiles, tabId, hasDirectory)
      } catch (err) {
        ElMessage.error('拖拽上传失败：' + (err.message || err))
      }
    },
    /**
     * 确保 FTP 目录存在（递归创建多级目录）
     * fullDirPath 形如："", "/a", "a/b", "/a/b"
     */
    async ensureFtpDirectoryExists(tab, fullDirPath) {
      // 统一去掉前后多余的 /
      const cleaned = (fullDirPath || '').replace(/\/+/g, '/').replace(/^\/+/, '').replace(/\/+$/, '')
      if (!cleaned) return

      const segments = cleaned.split('/').filter(p => p)
      let parent = ''

      // 确保端口为整数
      const port = parseInt(tab.config.port) || 21

      for (let i = 0; i < segments.length; i += 1) {
        const folderName = segments[i]
        try {
          await axios.post(`http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/mkdir`, {
            host: tab.config.host,
            port: port,
            username: tab.config.username,
            password: tab.config.password,
            ssh_addr: tab.config.ssh_addr || '',
            ssh_port: parseInt(tab.config.ssh_port) || 0,
            ssh_username: tab.config.ssh_username || '',
            ssh_password: tab.config.ssh_password || '',
            path: parent,
            folderName
          })
        } catch (e) {
          // 目录可能已存在，忽略错误
        }
        parent = parent ? `${parent}/${folderName}` : folderName
      }
    },
    formatFtpFileSize(bytes) {
      if (bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    },
    // 文件预览相关方法
    handleFtpFileNameDblClick(row, tabId) {
      if (!row) return
      if (row.type === 'folder') {
        this.enterFtpFolder(row.name, tabId)
      } else if (row.type === 'file') {
        this.openFtpFilePreview(row, tabId)
      }
    },
    async openFtpFilePreview(file, tabId) {
      const tab = this.getTab(tabId)
      if (!tab || !file || file.type === 'folder') {
        return
      }

      // 文件过大直接建议下载
      const size = typeof file.size === 'string' ? parseFloat(file.size) : file.size
      if (size && size > 5 * 1024 * 1024) {
        this.filePreview = {
          ...this.filePreview,
          file,
          tabId,
          loading: false,
          isText: false,
          error: '文件大于 5MB，暂不支持在线预览，请使用下载功能',
          content: '',
          editContent: '',
          dirty: false
        }
        return
      }

      this.filePreview.file = file
      this.filePreview.tabId = tabId
      this.filePreview.loading = true
      this.filePreview.error = ''
      this.filePreview.isText = false
      this.filePreview.content = ''
      this.filePreview.editContent = ''
      this.filePreview.dirty = false

      try {
        // 确保port是整数类型
        const port = parseInt(tab.config.port) || 21
        // 构建文件路径，处理根目录情况
        let filePath = ''
        if (tab.currentPath) {
          filePath = `${tab.currentPath}/${file.name}`.replace(/\/+/g, '/')
        } else {
          filePath = `/${file.name}`.replace(/\/+/g, '/')
        }
        
        const res = await axios.post(`http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/download`, {
          host: tab.config.host,
          port: port,
          username: tab.config.username,
          password: tab.config.password,
          path: filePath,
          ssh_addr: tab.config.ssh_addr || '',
          ssh_port: parseInt(tab.config.ssh_port) || 0,
          ssh_username: tab.config.ssh_username || '',
          ssh_password: tab.config.ssh_password || ''
        }, {
          responseType: 'arraybuffer'
        })

        const buffer = res.data instanceof ArrayBuffer ? res.data : res.data?.data
        if (!buffer) {
          this.filePreview.error = '无法读取文件内容'
          return
        }

        const bytes = new Uint8Array(buffer)
        const decoder = new TextDecoder('utf-8')
        const text = decoder.decode(bytes)

        // 简单判断是否为文本：控制字符比例过高则认为是二进制
        let controlCount = 0
        const sampleLen = Math.min(bytes.length, 2048)
        for (let i = 0; i < sampleLen; i += 1) {
          const ch = bytes[i]
          if (ch === 9 || ch === 10 || ch === 13) continue
          if (ch < 32 || ch === 127) controlCount += 1
        }
        const isBinary = sampleLen > 0 && controlCount / sampleLen > 0.1

        if (isBinary) {
          this.filePreview.isText = false
          this.filePreview.error = '当前文件看起来是二进制文件，暂不支持在线预览'
          this.filePreview.content = ''
          this.filePreview.editContent = ''
          this.filePreview.dirty = false
        } else {
          this.filePreview.isText = true
          this.filePreview.content = text
          this.filePreview.editContent = text
          this.filePreview.dirty = false
        }
      } catch (err) {
        console.error('预览文件失败', err)
        this.filePreview.error = '预览文件失败，请稍后重试或使用下载功能'
      } finally {
        this.filePreview.loading = false
      }
    },
    closeFtpFilePreview() {
      this.filePreview = {
        file: null,
        loading: false,
        isText: false,
        error: '',
        content: '',
        editContent: '',
        dirty: false,
        tabId: null
      }
    },
    async saveFtpFileEdit() {
      if (!this.filePreview.file || !this.filePreview.isText) return
      if (!this.filePreview.dirty) return

      const tab = this.getTab(this.filePreview.tabId)
      if (!tab) return

      try {
        const port = parseInt(tab.config.port) || 21
        // 构建文件路径，处理根目录情况
        let filePath = ''
        if (tab.currentPath) {
          filePath = `${tab.currentPath}/${this.filePreview.file.name}`.replace(/\/+/g, '/')
        } else {
          filePath = `/${this.filePreview.file.name}`.replace(/\/+/g, '/')
        }
        
        // 将编辑内容转换为 Blob
        const blob = new Blob([this.filePreview.editContent], {
          type: 'text/plain;charset=utf-8'
        })
        const uploadFile = new File([blob], this.filePreview.file.name, {
          type: 'text/plain;charset=utf-8'
        })

        const form = new FormData()
        form.append('file', uploadFile)
        form.append('host', tab.config.host)
        form.append('port', String(port))
        form.append('username', tab.config.username)
        form.append('password', tab.config.password)
        form.append('remotePath', filePath)
        form.append('ssh_addr', tab.config.ssh_addr || '')
        form.append('ssh_port', String(parseInt(tab.config.ssh_port) || 0))
        form.append('ssh_username', tab.config.ssh_username || '')
        form.append('ssh_password', tab.config.ssh_password || '')

        await axios.post(
          `http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/upload`,
          form,
          {
            headers: { 'Content-Type': 'multipart/form-data' }
          }
        )

        ElMessage.success('保存成功')
        this.filePreview.dirty = false
        // 重新刷新当前目录
        await this.loadFtpFileList(this.filePreview.tabId)
      } catch (err) {
        console.error('保存文件失败', err)
        ElMessage.error('保存文件失败，请稍后重试')
      }
    },
    // 文件夹下载
    async FtpDownloadDirectory(name, tabId) {
      const tab = this.getTab(tabId)
      if (!tab) return

      try {
        const port = parseInt(tab.config.port) || 21
        // 构建文件夹路径，处理根目录情况
        let folderPath = ''
        if (tab.currentPath) {
          folderPath = `${tab.currentPath}/${name}`.replace(/\/+/g, '/')
        } else {
          folderPath = `/${name}`.replace(/\/+/g, '/')
        }
        
        // 构建查询参数
        const params = new URLSearchParams()
        params.append('host', tab.config.host)
        params.append('port', String(port))
        params.append('username', tab.config.username)
        params.append('password', tab.config.password)
        params.append('path', folderPath)
        
        if (tab.config.ssh_addr) {
          params.append('ssh_addr', tab.config.ssh_addr)
          params.append('ssh_port', String(parseInt(tab.config.ssh_port) || 0))
          params.append('ssh_username', tab.config.ssh_username)
          params.append('ssh_password', tab.config.ssh_password)
        }
        
        const reqUrl = `http://127.0.0.1:${this.portConfig.ftpPort}/api/ftp/download_dir?${params.toString()}`

        BrowserOpenURL(reqUrl)
        ElMessage.success('已在浏览器中开始下载文件夹')
      } catch (err) {
        ElMessage.error('下载失败: ' + err.message)
      }
    },
    handleTabRemove(tabId) {
      const index = this.tabs.findIndex(t => t.id === tabId);
      if (index > -1) {
        this.tabs.splice(index, 1);
        if (this.tabs.length > 0) {
          this.activeTabId = this.tabs[Math.min(index, this.tabs.length - 1)].id;
        } else {
          this.activeTabId = null;
        }
      }
    },
    handleTabClick(tab) {
      this.activeTabId = tab.paneName;
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
    },
    // 关闭所有标签页（右键菜单）
    handleCloseAllTabs() {
      this.hideTabsContextMenu()
      this.tabs = []
      this.activeTabId = null
    },
  },
  watch: {
    'filePreview.editContent'(val, oldVal) {
      if (val !== oldVal && this.filePreview.isText) {
        this.filePreview.dirty = true
      }
    }
  }
}
</script>

<style scoped>
.ftp-container {
  height: 100%;
  display: flex;
  gap: 8px;
  padding: 8px;
  box-sizing: border-box;
  overflow: hidden;
  background: transparent;
}

.ftp-sidebar {
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

.ftp-toggle-sidebar-btn {
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

.ftp-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  padding: 12px;
  overflow: hidden;
}

.ftp-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.ftp-tabs :deep(.el-tabs__content) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.ftp-tabs :deep(.el-tab-pane) {
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

.file-manager-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.file-manager-card :deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.ftp-file-content {
  flex: 1;
  display: flex;
  gap: 10px;
  min-height: 0;
  overflow: hidden;
}

.ftp-file-content.has-preview {
  flex-direction: row;
}

.ftp-file-table-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.ftp-file-table {
  flex: 1;
  min-height: 260px; /* 保证在无数据时也有合理的表格高度 */
}

/* 文件预览面板样式 */
.ftp-preview-pane {
  flex: 1.2;
  display: flex;
  flex-direction: column;
  background: #0f172a;
  border-radius: 8px;
  box-shadow: 0 4px 10px rgba(15, 23, 42, 0.14);
  color: #e5e7eb;
  overflow: hidden;
  min-width: 0;
}

.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.4);
  background: radial-gradient(circle at top left, #1e293b, #020617);
}

.preview-title {
  font-size: 13px;
  color: #e5e7eb;
}

.preview-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.preview-close-btn {
  color: #fca5a5;
}

.preview-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #1f1f21;
  min-height: 0;
  overflow: hidden;
}

.preview-scroll {
  height: 100%;
  max-height: 100%;
  overflow: hidden;
}

.preview-editor-wrapper {
  height: 100%;
  padding: 0px;
  box-sizing: border-box;
}

.preview-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 230px;
  padding: 20px;
  box-sizing: border-box;
  color: #e5e7eb;
}

/* 窗口较窄时，改为上下布局 */
@media (max-width: 1400px) {
  .ftp-file-content.has-preview {
    flex-direction: column;
  }

  .ftp-preview-pane {
    margin-top: 8px;
  }
}

.file-manager-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.file-ftp-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-ftp-name {
  margin-left: 8px;
}

.el-breadcrumb {
  flex-grow: 1;
}

.el-breadcrumb-item {
  cursor: pointer;
}

.el-breadcrumb-item:hover {
  color: var(--el-color-primary);
}

.ftp-empty-state {
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

.dialog-footer-right {
  display: flex;
  gap: 8px;
  align-items: center;
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

.ftp-file-context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  padding: 2px 0;
  margin: 0;
  list-style: none;
  z-index: 3000;
  min-width: 120px;
}

.ftp-file-context-menu li {
  padding: 6px 14px;
  cursor: pointer;
  font-size: 13px;
  color: #606266;
  transition: background-color 0.2s;
}

.ftp-file-context-menu li:hover {
  background-color: #f5f7fa;
  color: #409eff;
}

.ftp-file-context-menu li.danger {
  color: #f56c6c;
}

</style>
