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
      <div v-if="activeTab === 'ssh'" class="tab-content iframe-container">
        <iframe src="http://127.0.0.1:10088/"></iframe>
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
              <el-input v-model="ftpConfig.host" placeholder="ftp.example.com" style="width: 180px"></el-input>
            </el-form-item>
            <el-form-item label="端口">
              <el-input v-model.number="ftpConfig.port" type="number" style="width: 55px"></el-input>
            </el-form-item>
            <el-form-item label="用户名">
              <el-input v-model="ftpConfig.username" placeholder="匿名登录：anonymous" style="width: 180px"></el-input>
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
                    v-for="(ftpPathItems, index) in ftpPathItems"
                    :key="index"
                    @click.native="ftpNnavigateTo(index)">
                  {{ getBreadcrumbName(ftpPathItems, index) }}
                </el-breadcrumb-item>
              </el-breadcrumb>

              <el-upload
                  class="upload-ftp-btn"
                  :show-file-list="false"
                  :before-upload="handleFtpUpload">
                <el-button type="primary">
                  <el-icon><Upload /></el-icon>
                  上传文件
                </el-button>
              </el-upload>
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
            <el-table-column label="操作" width="150">
              <template #default="{ row }">
                <el-button
                    v-if="row.type !== 'folder'"
                    size="small"
                    @click="FtpDownloadFile(row.name)">
                  下载
                </el-button>
                <el-button
                    size="small"
                    type="danger"
                    @click="deleteFtpFile(row.name)">
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </div>

      <!-- redis -->
      <div v-if="activeTab === 'redis'" class="tab-content">
        <el-row>
          <el-col :span="5" style="height: 100vh; padding: 12px">
            <div style="margin-bottom: 12px">
              <ConnectionManage title="新建连接" btn-type="primary" @emit-connection-list="flushConnectionList"/>
            </div>
            <ConnectionList @emit-select-db="selectRedisDB" :flush="flushFlag"/>
          </el-col>
          <el-col :span="7" style="padding: 12px">
            <Keys :keyDB="keyDB" :keyConnIdentity="keyConnIdentity" @emit-select-key="selectKey"/>
          </el-col>
          <el-col :span="12" style="padding: 12px">
            <KeyValue :keyDB="keyDB" :keyConnIdentity="keyConnIdentity" :keyKey="keyKey" />
          </el-col>
        </el-row>
      </div>
    </el-main>
  </el-container>
</template>

<script>
import ConnectionList from "./redis/ConnectionList.vue";
import ConnectionManage from "./redis/ConnectionManage.vue";
import Keys from "./redis/Keys.vue";
import KeyValue from "./redis/KeyValue.vue";
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { Folder, Document, Upload } from '@element-plus/icons-vue'

export default {
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
      fileList: []
    };
  },
  computed: {
    ftpPathItems() {
      // 确保始终包含根目录项
      const parts = this.currentPath.split('/').filter(item => item)
      return ['', ...parts] // 第一个元素代表根目录
    }
  },
  mounted() {
    // 禁用右键菜单
    document.addEventListener('contextmenu', (event) => {
      event.preventDefault();
    });
  },
  beforeUnmount() {
    // 在组件卸载时移除事件监听器，避免潜在的内存泄漏
    document.removeEventListener('contextmenu', (event) => {
      event.preventDefault();
    });
  },
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
      try {
        await axios.post('http://127.0.0.1:10089/api/ftp/connect', this.ftpConfig)
        this.ftp_connected = true
        this.currentPath = ''
        this.loadFtpFileList()
        ElMessage.success('连接成功')
      } catch (error) {
        ElMessage.error('连接失败: ' + (error.response?.data?.error || error.message))
      }
    },

    async loadFtpFileList(path = '') {
      try {
        const response = await axios.post('http://127.0.0.1:10089/api/ftp/list', {
          ...this.ftpConfig,
          path: this.currentPath + path
        })
        this.fileList = response.data.files.map(file => ({
          ...file,
          type: file.type === 'folder' ? 'folder' : 'file'
        }))
      } catch (error) {
        ElMessage.error('获取文件列表失败: ' + error.message)
      }
    },

    enterFtpFolder(folderName) {
      this.currentPath += `/${folderName}`
      this.loadFtpFileList()
    },

    getBreadcrumbName(ftpPathItems, index) {
      return index === 0 ? '根目录' : ftpPathItems
    },

    ftpNnavigateTo(index) {
      if (index === 0) {
        // 点击根目录时重置路径
        this.currentPath = ''
      } else {
        // 构建新路径时去掉根目录占位符
        const selectedParts = this.ftpPathItems.slice(1, index + 1)
        this.currentPath = '/' + selectedParts.join('/')
      }
      this.loadFtpFileList()
    },

    async deleteFtpFile(fileName) {
      try {
        await axios.post('http://127.0.0.1:10089/api/ftp/delete', {
          ...this.ftpConfig,
          path: `${this.currentPath}/${fileName}`
        })
        ElMessage.success('删除成功')
        this.loadFtpFileList()
      } catch (error) {
        ElMessage.error('删除失败: ' + error.message)
      }
    },

    async FtpDownloadFile(fileName) {
      try {
        const response = await axios.post(
            'http://127.0.0.1:10089/api/ftp/download',
            {
              ...this.ftpConfig,
              path: `${this.currentPath}/${fileName}`
            },
            { responseType: 'blob' }
        )

        const url = window.URL.createObjectURL(new Blob([response.data]))
        const link = document.createElement('a')
        link.href = url
        link.setAttribute('download', fileName)
        document.body.appendChild(link)
        link.click()
        link.remove()
      } catch (error) {
        ElMessage.error('下载失败: ' + error.message)
      }
    },

    async handleFtpUpload(file) {
      const formData = new FormData()
      formData.append('file', file)
      formData.append('host', this.ftpConfig.host)
      formData.append('port', this.ftpConfig.port)
      formData.append('username', this.ftpConfig.username)
      formData.append('password', this.ftpConfig.password)
      formData.append('remotePath', `${this.currentPath}/${file.name}`)

      try {
        await axios.post('http://127.0.0.1:10089/api/ftp/upload', formData, {
          headers: { 'Content-Type': 'multipart/form-data' }
        })
        ElMessage.success('上传成功')
        this.loadFtpFileList()
      } catch (error) {
        ElMessage.error('上传失败: ' + error.message)
      }
    },

    formatFtpFileSize(bytes) {
      if (bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    }
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

:deep(.el-tabs__item) {
  font-weight: 700;
  /* 加粗字体 */
  font-size: 16px;
  /* 设置字体大小 */
  transition: all 0.3s;
}
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
