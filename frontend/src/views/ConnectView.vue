<template>
  <el-container class="container">
    <!-- 标签栏 -->
    <el-tabs v-model="activeTab" class="tabs">
      <el-tab-pane label="SSH" name="ssh" />
      <el-tab-pane label="FTP" name="ftp" />
      <el-tab-pane label="Redis" name="redis" />
    </el-tabs>

    <!-- 内容区域 -->
    <el-main>
      <!-- ssh -->
      <div v-show="activeTab === 'ssh'" class="tab-content iframe-container">
        <iframe src="http://127.0.0.1:52868/"></iframe>
      </div>

      <!-- ftp -->
      <div v-if="activeTab === 'ftp'" class="tab-content iframe-container">
        <el-card class="connection-card">
          <template #header>
            <div class="card-header">
              <span>FTP服务器连接</span>
            </div>
          </template>

          <el-form :inline="true">
            <el-form-item label="主机地址">
              <el-input v-model="ftpConfig.host" placeholder="ftp.example.com" style="width: 165px"></el-input>
            </el-form-item>
            <el-form-item label="端口">
              <el-input v-model.number="ftpConfig.port" type="number" style="width: 75px"></el-input>
            </el-form-item>
            <el-form-item label="用户名">
              <el-input v-model="ftpConfig.username" placeholder="匿名登录：anonymous" style="width: 165px"></el-input>
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="ftpConfig.password" type="password" style="width: 150px"></el-input>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="connect">连接</el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <el-card v-if="ftp_connected" class="file-manager-card">
          <template #header>
            <div class="file-manager-header">
              <el-breadcrumb separator="/">
                <el-breadcrumb-item
                    v-for="(label, index) in ftpPathItems"
                    :key="index"
                    @click="ftpNavigateTo(index)">
                  {{ index === 0 ? '根目录' : label }}
                </el-breadcrumb-item>
              </el-breadcrumb>

              <el-button type="warning" @click="showNewFolderDialog = true">
                <el-icon><Folder /></el-icon> 新建文件夹
              </el-button>

              <input
                  ref="ftpFileInput"
                  type="file"
                  style="display: none"
                  @change="onFileSelected"
              />

              <!-- 上传按钮，点击时触发 file input -->
              <el-button type="primary" @click="$refs.ftpFileInput.click()">
                <el-icon><Upload /></el-icon> 上传文件
              </el-button>

              <!-- 上传进度条 -->
              <el-progress
                  v-if="uploading"
                  :percentage="uploadProgress"
                  :status="uploadProgress === 100 ? 'success' : ''"
                  :style="{ width: '50px', display: 'inline-block', marginLeft: '5px' }"
              />

              <!-- 新建文件夹对话框 -->
              <el-dialog
                  title="新建文件夹"
                  v-model="showNewFolderDialog"
              >
                <el-input v-model="newFolderName" placeholder="请输入文件夹名称" />
                <span slot="footer" class="dialog-footer">
                  <el-button @click="showNewFolderDialog = false">取 消</el-button>
                  <el-button type="primary" @click="createFolder">确 认</el-button>
                </span>
              </el-dialog>
            </div>
          </template>

          <el-table :data="fileList" stripe style="width: 100%">
            <el-table-column prop="name" label="名称">
              <template #default="{ row }">
                <div class="file-ftp-item">
                  <el-icon v-if="row.type === 'folder'"><Folder /></el-icon>
                  <el-icon v-else><Document /></el-icon>
                  <span
                      class="file-ftp-name"
                      @click="row.type === 'folder' ? enterFtpFolder(row.name) : null"
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
            <el-table-column label="操作" width="200">
              <template #default="{ row }">
                <el-button
                    v-if="row.type !== 'folder'"
                    size="small"
                    @click="FtpDownloadFile(row.name)">
                  下载
                </el-button>
                <el-progress
                    v-if="downloadingFile === row.name"
                    :percentage="downloadProgress"
                    :style="{ width: '50px', display: 'inline-block', marginLeft: '5px' }"
                />
                <el-button
                    size="small"
                    type="danger"
                    @click="deleteFtpFile(row.name)">
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-pagination
              v-model:current-page="currentPage"
              v-model:page-size="pageSize"
              :page-sizes="[10, 20, 50, 100]"
              :total="totalFiles"
              layout="sizes, prev, pager, next, jumper"
              @current-change="onPageChange"
              @size-change="onSizeChange"
          />
        </el-card>
      </div>

      <!-- redis -->
      <div v-if="activeTab === 'redis'" class="tab-content">
        <el-row>
          <el-col :span="7" style="height: 100vh; padding: 5px">
            <div style="margin-bottom: 12px">
              <ConnectionManage title="新建连接" btn-type="primary" @emit-connection-list="flushConnectionList"/>
            </div>
            <ConnectionList @emit-select-db="selectRedisDB" :flush="flushFlag"/>
          </el-col>

          <template v-if="isConnected">
            <el-col :span="5" style="padding: 12px">
              <Keys
                  :keyDB="keyDB"
                  :keyConnIdentity="keyConnIdentity"
                  @emit-select-key="selectKey"
              />
            </el-col>
            <el-col :span="12" style="padding: 12px">
              <KeyValue
                  :keyDB="keyDB"
                  :keyConnIdentity="keyConnIdentity"
                  :keyKey="keyKey"
              />
            </el-col>
          </template>

        </el-row>
      </div>
    </el-main>
  </el-container>
</template>

<script>
import ConnectionList from "./redis/ConnectionList.vue";
import ConnectionManage from "./redis/ConnectionManage.vue";
import { BrowserOpenURL } from '../../wailsjs/runtime';
import Keys from "./redis/Keys.vue";
import KeyValue from "./redis/KeyValue.vue";
import axios from 'axios'
import { ElMessage,ElMessageBox } from 'element-plus'
import { Folder, Document, Upload } from '@element-plus/icons-vue'

export default {
  name: "ConnectView",
  components: {
    ConnectionList,
    ConnectionManage,
    Keys,
    KeyValue,

    Folder,
    Document,
    Upload
  },
  data() {
    return {
      activeTab: "ssh",  // 默认选中的 Tab
      flushFlag: true,
      keyDB: null,
      keyConnIdentity: null,
      keyKey: null,

      ftpConfig: {
        host: '',
        port: 21,
        username: '',
        password: ''
      },
      ftp_connected: false,
      currentPath: '',
      fileList: [],

      currentPage: 1,
      pageSize: 20,
      totalFiles: 0,
      uploading: false,
      uploadProgress: 0,
      downloading: false,
      downloadingFile: '',    // 正在下载的文件名
      downloadProgress: 0,

      showNewFolderDialog: false,
      newFolderName: '',
    };
  },
  computed: {
    ftpPathItems() {
      const parts = this.currentPath.split('/').filter(p => p);
      return ['根目录', ...parts];
    },
    // 只要有 keyConnIdentity，就认为连接成功
    isConnected() {
      return this.keyConnIdentity !== null;
    },
  },
  mounted() {
    // 禁用右键菜单
    document.addEventListener('contextmenu', (event) => {
      event.preventDefault();
    });
    // 只有第一次打开才弹窗：可以根据需要改成持久化判断（如 localStorage）
    ElMessageBox.confirm(
        '温馨提示，该功能还存在 bug，在您切换页面之后，无法保存已加载的编码选项，是否在浏览器中打开网页呢？',
        '提示',
        {
          confirmButtonText: '确认',
          cancelButtonText: '取消',
          type: 'warning',
          // 点击遮罩或按 ESC 不触发 confirm
          distinguishCancelAndClose: true,
        }
    )
        .then(() => {
          // 用户点击“确认”，在默认浏览器中打开
          const fullUrl = "http://127.0.0.1:52868/";
          BrowserOpenURL(fullUrl);
          // 本地 iframe 继续加载，不需要额外操作
        })
        .catch((action) => {
          // 用户点击“取消”或关闭，不做额外操作
          // 本地 iframe 会如常加载
        });
  },
  // beforeUnmount() {
  //   // 在组件卸载时移除事件监听器，避免潜在的内存泄漏
  //   document.removeEventListener('contextmenu', (event) => {
  //     event.preventDefault();
  //   });
  // },
  methods: {
    flushConnectionList() {
      this.flushFlag = !this.flushFlag;
    },
    selectRedisDB(db, connIdentity) {
      this.keyDB = db;
      this.keyConnIdentity = connIdentity;
    },
    selectKey(key) {
      this.keyKey = key;
    },

    // ftp
    async connect() {
      const msg = ElMessage.info('正在连接中，请稍等...');
      try {
        await axios.post('http://127.0.0.1:52869/api/ftp/connect', this.ftpConfig);
        this.ftp_connected = true;
        this.currentPath = '';
        await this.loadFtpFileList();
        msg.close();
        ElMessage.success('连接成功');
      } catch (err) {
        msg.close();
        ElMessage.error('连接失败: ' + (err.response?.data?.error || err.message));
      }
    },
    async loadFtpFileList() {
      try {
        const { data } = await axios.post('http://127.0.0.1:52869/api/ftp/list', {
          ...this.ftpConfig,
          path: this.currentPath,
          page: this.currentPage,
          pageSize: this.pageSize,
        });
        this.fileList = data.files.map(f => ({ ...f, type: f.type === 'folder' ? 'folder' : 'file' }));
        this.totalFiles = data.pagination.total;
      } catch (err) {
        ElMessage.error('获取列表失败: ' + err.message);
      }
    },
    onPageChange(page) { this.currentPage = page; this.loadFtpFileList(); },
    onSizeChange(size) { this.pageSize = size; this.currentPage = 1; this.loadFtpFileList(); },
    enterFtpFolder(name) { this.currentPath += '/' + name; this.loadFtpFileList(); },
    ftpNavigateTo(idx) {
      if (idx === 0) this.currentPath = '';
      else {
        const parts = this.ftpPathItems.slice(1, idx + 1);
        this.currentPath = '/' + parts.join('/');
      }
      this.loadFtpFileList();
    },
    deleteFtpFile(name) {
      axios.post('http://127.0.0.1:52869/api/ftp/delete', { ...this.ftpConfig, path: `${this.currentPath}/${name}` })
          .then(() => { ElMessage.success('删除成功'); this.loadFtpFileList(); })
          .catch(err => ElMessage.error('删除失败: ' + err.message));
    },
    async FtpDownloadFile(name) {
      this.downloadingFile = name;
      this.downloadProgress = 0;
      try {
        const res = await axios.post(
            'http://127.0.0.1:52869/api/ftp/download',
            { ...this.ftpConfig, path: `${this.currentPath}/${name}` },
            { responseType: 'blob', onDownloadProgress: e => {
                this.downloadProgress = Math.floor(e.loaded / e.total * 100);
              }}
        );
        const url = URL.createObjectURL(new Blob([res.data]));
        const link = document.createElement('a'); link.href = url; link.download = name;
        document.body.appendChild(link); link.click(); link.remove();
        ElMessage.success('下载完成');
      } catch (err) {
        ElMessage.error('下载失败: ' + err.message);
      } finally {
        this.downloadingFile = '';
        this.downloadProgress = 0;
      }
    },
    // 文件选中后触发
    onFileSelected(event) {
      const file = event.target.files[0];
      if (!file) return;
      // 调用真正的上传逻辑
      this.uploadFile(file);
      // 清空一下，以便同一个文件二次选择也能触发
      this.$refs.ftpFileInput.value = null;
    },

    // 真正的上传逻辑
    async uploadFile(file) {
      const form = new FormData();
      form.append('file', file);
      form.append('host', this.ftpConfig.host);
      form.append('port', this.ftpConfig.port);
      form.append('username', this.ftpConfig.username);
      form.append('password', this.ftpConfig.password);
      form.append('remotePath', `${this.currentPath}/${file.name}`);

      this.uploading = true;
      this.uploadProgress = 0;

      try {
        await axios.post(
            'http://127.0.0.1:52869/api/ftp/upload',
            form,
            {
              headers: { 'Content-Type': 'multipart/form-data' },
              onUploadProgress: e => {
                this.uploadProgress = Math.floor((e.loaded / e.total) * 100);
              }
            }
        );
        this.$message.success('上传成功');
        // 刷新列表
        await this.loadFtpFileList();
      } catch (err) {
        this.$message.error('上传失败：' + err.message);
      } finally {
        this.uploading = false;
      }
    },

    async createFolder() {
      console.log('createFolder invoked:', this.newFolderName);
      if (!this.newFolderName.trim()) {
        return this.$message.warning('请输入文件夹名称');
      }
      try {
        await axios.post('http://127.0.0.1:52869/api/ftp/mkdir', {
          ...this.ftpConfig,
          path: this.currentPath,
          folderName: this.newFolderName
        });
        this.$message.success('创建成功');
        this.showNewFolderDialog = false;
        this.newFolderName = '';
        await this.loadFtpFileList();
      } catch (err) {
        this.$message.error('创建失败：' + err.message);
      }
    },

    formatFtpFileSize(bytes) {
      if (bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    },

  },
};
</script>

<style scoped>
.el-main {
  padding: 0;
}

/* 页面容器 */
.container {
  height: 100vh;
  display: flex;

  flex-direction: column;
  background-color: #f8f9fb;
}

/* 顶部 Tabs 样式 */
.tabs {
  background-color: #ffffff;
  box-shadow: 0px 2px 2px rgba(0, 0, 0, 0.1);
  margin-bottom: 5px;
  /* border-bottom: 2px solid #ebeef5; */
  padding-left: 10px;
  /* 增加左边距 */
  border-radius: 10px 10px 10px 10px;
}

/*   顶部样式 注释  */
/* :deep(.el-tabs__item) { */
/*   font-weight: 700; */
  /* 加粗字体 */
/*   font-size: 16px; */
  /* 设置字体大小 */
/*   transition: all 0.3s; */
/* } */

/* iframe 容器样式 */
.iframe-container {
  /* margin: 0 auto; */
  max-width: 100%;
  /* 控制 iframe 的最大宽度 */
  overflow: hidden;
  border-radius: 8px;
  /* 圆角处理 */
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  /* 增加阴影效果 */
}

/* iframe 样式 */
.iframe-container iframe {
  width: 100%;
  /* 保持 iframe 宽度填充父容器 */
  height: 90vh;
  /* 高度占视口的 98% */
  border: none;
  /* 去掉默认边框 */
  border-radius: 8px;
  /* 与父容器一致的圆角 */
}


/* FTP 样式 */
.connection-card {
  margin-bottom: 20px;
}

.file-manager-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.upload-ftp-btn {
  margin-left: auto;
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
  &:hover {
    color: var(--el-color-primary);
  }
}
</style>
