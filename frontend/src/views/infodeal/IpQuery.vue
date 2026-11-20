<template>
  <div class="ip-query-wrapper">
    <!-- 左右布局 -->
    <el-row :gutter="10" class="ip-query-content">
      <!-- 左侧输入区域 -->
      <el-col :span="8">
        <el-card shadow="hover" class="input-card">
          <template #header>
            <div class="card-header">
              <span>IP地址/域名输入</span>
              <div class="header-actions">
                <!-- <el-checkbox v-model="useAuthorizedApi" @change="handleApiCheckboxChange">
                  <span style="color: chocolate; padding-right: 10px;">高精查询</span>
                </el-checkbox> -->
                <el-button type="primary" @click="startIpQuery" :loading="ipQueryLoading" size="small">
                  {{ ipQueryLoading ? '查询中...' : '查询' }}
                </el-button>
                <el-button @click="clearIpInput" size="small">清空</el-button>
              </div>
            </div>
          </template>

          <div class="input-section">
            <el-input
                v-model="ipQueryInput"
                type="textarea"
                placeholder="请输入IP地址或域名，每行一个。支持以下格式：
- 纯IP: 1.1.1.1
- 带协议: http://1.1.1.1, https://example.com
- 域名: example.com, www.google.com"
                :rows="29"
                class="ip-textarea"
                resize="none"
            />
          </div>

          <div class="input-stats">
            <div class="stat-item total">
              <span class="label">输入数量</span>
              <span class="value">{{ ipInputCount }}</span>
            </div>
            <div class="stat-item completed">
              <span class="label">已完成</span>
              <span class="value">{{ ipCompletedCount }}</span>
            </div>
            <div class="stat-item success">
              <span class="label">成功</span>
              <span class="value">{{ ipSuccessCount }}</span>
            </div>
            <div class="stat-item error">
              <span class="label">失败</span>
              <span class="value">{{ ipFailedCount }}</span>
            </div>
          </div>

          <!-- 查询进度 -->
          <div class="query-progress" v-if="ipQueryLoading">
            <el-progress
                :percentage="ipQueryProgress"
                :status="ipQueryProgress === 100 ? 'success' : ''"
                :show-text="true"
            />
            <p class="progress-text">查询进度: {{ ipQueryProgress }}% ({{ ipCompletedCount }}/{{ ipInputCount }})</p>
          </div>
        </el-card>
      </el-col>

      <!-- 右侧结果表格 -->
      <el-col :span="16">
        <el-card shadow="hover" class="result-card">
          <template #header>
            <div class="card-header">
              <span>查询结果 (共 {{ ipQueryResults.length }} 条)</span>
              <div class="result-actions">
                <el-button type="success" @click="exportIpResults" size="small" :loading="ipExportLoading">
                  {{ ipExportLoading ? '导出中...' : '导出XLSX' }}
                </el-button>
                <el-button type="primary" @click="copyIpResults" size="small">
                  复制结果
                </el-button>
                <el-button @click="clearIpResults" size="small">
                  清空结果
                </el-button>
              </div>
            </div>
          </template>

          <div class="table-container">
            <el-table
                :data="ipQueryResults"
                border
                stripe
                style="width: 100%"
                height="650"
                v-loading="ipQueryLoading"
                empty-text="暂无查询结果"
            >
              <el-table-column type="index" label="序号" width="55" align="center" />
              <el-table-column prop="original" label="URL" min-width="190" show-overflow-tooltip>
                <template #default="{ row }">
                  <span 
                    class="copyable-cell" 
                    @click.stop="copyToClipboard(row.original, 'URL')"
                    :title="row.original ? '点击复制' : ''"
                  >
                    {{ row.original }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column prop="ip" label="IP地址" width="160" align="center">
                <template #default="{ row }">
                  <span 
                    class="copyable-cell" 
                    @click.stop="row.ip ? copyToClipboard(row.ip, 'IP地址') : null"
                    :title="row.ip ? '点击复制' : ''"
                  >
                    {{ row.ip || '-' }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column prop="location" label="IP归属地" min-width="200" show-overflow-tooltip>
                <template #default="{ row }">
                  <span 
                    class="copyable-cell" 
                    @click.stop="row.location && row.location !== '查询中...' ? copyToClipboard(row.location, '归属地') : null"
                    :title="row.location && row.location !== '查询中...' ? '点击复制' : ''"
                  >
                    {{ row.location || '查询中...' }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column prop="coordinates" label="经纬度" width="160" align="center">
                <template #default="{ row }">
                  <span 
                    class="copyable-cell" 
                    @click.stop="row.coordinates && row.coordinates !== '-' ? copyToClipboard(row.coordinates, '经纬度') : null"
                    :title="row.coordinates && row.coordinates !== '-' ? '点击复制' : ''"
                  >
                    {{ row.coordinates || '-' }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column prop="status" label="状态" width="80" align="center">
                <template #default="{ row }">
                  <el-tag
                      :type="row.status === 'success' ? 'success' : row.status === 'error' ? 'danger' : 'warning'"
                      size="small"
                      class="copyable-cell"
                      @click.stop="copyToClipboard(getIpStatusText(row.status), '状态')"
                      title="点击复制"
                  >
                    {{ getIpStatusText(row.status) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="120" align="center">
                <template #default="{ row }">
                  <el-button
                      v-if="row.ip && !row.useAuthorizedApi"
                      link
                      size="small"
                      @click="openIpInfo(row.ip)"
                  >
                    详情
                  </el-button>
                  <el-button
                      link
                      size="small"
                      @click="retryQuery(row)"
                      :loading="row.retrying"
                  >
                    {{ row.retrying ? '重试中' : '重试' }}
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script>
import {ExportIPResults, QueryIPLocation, QueryIPLocationWithAuthorized} from "../../../wailsjs/go/infoDeal/QueryIp";
import {BrowserOpenURL} from "../../../wailsjs/runtime";
import {ElMessage} from "element-plus";

export default {
  name: "IpQuery",
  data() {
    return {
      // IP查询相关数据
      ipQueryInput: '',
      ipQueryLoading: false,
      ipQueryResults: [],
      ipQueryProgress: 0,
      ipExportLoading: false,
      useAuthorizedApi: false,

      // IP查询统计
      ipCompletedCount: 0,
      ipSuccessCount: 0,
      ipFailedCount: 0,
    }
  },
  computed:{
    // IP输入数量
    ipInputCount() {
      if (!this.ipQueryInput.trim()) return 0
      return this.ipQueryInput.split('\n')
          .filter(line => line.trim())
          .length
    },
  },
  methods: {
    // 处理 API 复选框变化
    async handleApiCheckboxChange(value) {
      if (value) {
        this.useAuthorizedApi = true;
      }
    },
    // IP查询相关方法
    clearIpInput() {
      this.ipQueryInput = ''
    },

    clearIpResults() {
      this.ipQueryResults = []
      this.ipCompletedCount = 0
      this.ipSuccessCount = 0
      this.ipFailedCount = 0
      this.ipQueryProgress = 0
    },

    copyIpResults() {
      if (this.ipQueryResults.length === 0) {
        this.$message.warning('没有可复制的数据')
        return
      }

      const text = this.ipQueryResults.map((item, index) =>
          `${index + 1}\t${item.original}\t${item.ip || ''}\t${item.location || ''}\t${item.coordinates || ''}`
      ).join('\n')

      this.copyToClipboard(text, 'IP查询结果')
    },

    getIpStatusText(status) {
      const statusMap = {
        'pending': '待查询',
        'success': '成功',
        'error': '失败'
      }
      return statusMap[status] || status
    },

    openIpInfo(ip) {
      if (!ip) return
      // 在新标签页打开IP信息查询网站
      BrowserOpenURL(`https://www.ipip.net/ip/${ip}.html`)
    },
    async startIpQuery() {
      if (!this.ipQueryInput.trim()) {
        this.$message.warning('请输入要查询的IP地址或域名')
        return
      }

      const inputs = this.ipQueryInput.split('\n')
          .filter(line => line.trim())
          .map(line => line.trim())

      if (inputs.length === 0) {
        this.$message.warning('请输入有效的IP地址或域名')
        return
      }

      // this.ipQueryLoading = true
      this.ipQueryResults = []
      this.ipCompletedCount = 0
      this.ipSuccessCount = 0
      this.ipFailedCount = 0
      this.ipQueryProgress = 0

      try {
        // 初始化结果数组
        this.ipQueryResults = inputs.map(input => ({
          original: input,
          ip: '',
          location: '',
          coordinates: '',
          status: 'pending',
          retrying: false,
          useAuthorizedApi: this.useAuthorizedApi
        }))

        // 使用 Promise.all 进行并发查询，但控制并发数
        const concurrency = 5 // 控制并发数
        const total = inputs.length

        // 分批处理
        for (let i = 0; i < inputs.length; i += concurrency) {
          const batch = inputs.slice(i, i + concurrency)
          const promises = batch.map((input, index) =>
              this.querySingleIpWithProgress(input, i + index, total)
          )

          await Promise.all(promises)

          // 更新进度
          this.ipCompletedCount = Math.min(i + concurrency, total)
          this.ipQueryProgress = Math.round((this.ipCompletedCount / total) * 100)

          // 添加小延迟，让UI有机会更新
          await new Promise(resolve => setTimeout(resolve, 100))
        }

        this.$message.success(`查询完成！成功: ${this.ipSuccessCount}, 失败: ${this.ipFailedCount}`)
      } catch (error) {
        console.error('IP查询错误:', error)
        this.$message.error('查询过程中发生错误: ' + error.message)
      } finally {
        this.ipQueryLoading = false
      }
    },

    // 重试单个查询
    async retryQuery(row) {
      const index = this.ipQueryResults.findIndex(item => item.original === row.original)
      if (index === -1) return

      row.retrying = true
      try {
        let response;
        const useAuthApi = row.useAuthorizedApi || this.useAuthorizedApi;
        
        if (useAuthApi) {
          response = await this.queryWithAuthorizedApi(row.original);
        } else {
          response = await QueryIPLocation({
            targets: [row.original]
          });
        }

        if (response.success) {
          // 构建归属地字符串
          let locationText = response.location || '';
          if (response.detail && response.detail !== locationText) {
            locationText = locationText ? `${locationText} - ${response.detail}` : response.detail;
          }

          // 构建经纬度字符串
          let coordinatesText = '';
          if (response.lat !== undefined && response.lng !== undefined) {
            coordinatesText = `${response.lat}, ${response.lng}`;
          }

          this.ipQueryResults[index] = {
            ...this.ipQueryResults[index],
            ip: response.ip,
            location: locationText,
            coordinates: coordinatesText,
            status: 'success',
            useAuthorizedApi: useAuthApi
          }
          // 更新统计
          if (row.status === 'error') {
            this.ipFailedCount--
          }
          this.ipSuccessCount++
          this.$message.success('重试成功')
        } else {
          this.ipQueryResults[index] = {
            ...this.ipQueryResults[index],
            ip: '',
            location: response.error,
            coordinates: '',
            status: 'error',
            useAuthorizedApi: useAuthApi
          }
          // 更新统计
          if (row.status === 'success') {
            this.ipSuccessCount--
          }
          this.ipFailedCount++
          this.$message.error('重试失败: ' + response.error)
        }
      } catch (error) {
        console.error('重试失败:', error)
        this.$message.error('重试失败: ' + error.message)
      } finally {
        row.retrying = false
      }
    },
    async querySingleIpWithProgress(input, index, total) {
      try {
        let response;
        if (this.useAuthorizedApi) {
          // 使用授权接口
          response = await this.queryWithAuthorizedApi(input);
        } else {
          // 使用原有接口
          response = await QueryIPLocation({
            targets: [input]
          });
        }

        if (response.success) {
          // 构建归属地字符串
          let locationText = response.location || '';
          if (response.detail && response.detail !== locationText) {
            locationText = locationText ? `${locationText} - ${response.detail}` : response.detail;
          }

          // 构建经纬度字符串
          let coordinatesText = '';
          if (response.lat !== undefined && response.lng !== undefined) {
            coordinatesText = `${response.lat}, ${response.lng}`;
          }

          this.ipQueryResults[index] = {
            ...this.ipQueryResults[index],
            ip: response.ip,
            location: locationText,
            coordinates: coordinatesText,
            status: 'success',
            useAuthorizedApi: this.useAuthorizedApi
          }
          this.ipSuccessCount++
        } else {
          throw new Error(response.error || '查询失败')
        }
      } catch (error) {
        console.error(`查询失败 [${input}]:`, error)
        this.ipQueryResults[index] = {
          ...this.ipQueryResults[index],
          ip: '',
          location: error.message,
          coordinates: '',
          status: 'error',
          useAuthorizedApi: this.useAuthorizedApi
        }
        this.ipFailedCount++
      }

      // 强制更新视图
      this.$forceUpdate()
    },
    // 使用授权接口查询
    async queryWithAuthorizedApi(input) {
      // 先提取 IP
      let ip = input;
      if (!this.isIPAddress(input)) {
        // 如果是域名或 URL,需要先解析
        try {
          const response = await QueryIPLocation({
            targets: [input]
          });
          if (response.success && response.ip) {
            ip = response.ip;
          } else {
            throw new Error('无法解析 IP 地址');
          }
        } catch (error) {
          return {
            success: false,
            error: `解析 IP 失败: ${error.message}`
          };
        }
      }

      // 使用后端 API 调用授权接口（后端会处理 CORS 问题）
      try {
        const response = await QueryIPLocationWithAuthorized({
          ip: ip,
          useAuthorized: true
        });
        
        // 如果后端返回成功，直接返回结果
        if (response.success) {
          return response;
        }
        
        // 如果授权接口失败，后端会自动回退到原有接口
        return response;
      } catch (error) {
        console.error('授权接口查询失败:', error);
        // 如果调用失败，回退到原有接口
        return await QueryIPLocation({
          targets: [input]
        });
      }
    },
    // 检查是否为 IP 地址
    isIPAddress(ip) {
      const ipRegex = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
      return ipRegex.test(ip);
    },
    async exportIpResults() {
      if (this.ipQueryResults.length === 0) {
        this.$message.warning('没有可导出的数据')
        return
      }

      this.ipExportLoading = true
      try {
        const timestamp = new Date().getTime()
        const fileName = `IP查询结果_${timestamp}`

        const result = await ExportIPResults({
          results: this.ipQueryResults,
          fileName: fileName,
          timestamp: new Date().toLocaleString('zh-CN')
        })

        if (result.success && result.data) {
          // 处理二进制数据 - 使用正确的方式
          try {
            // 将 base64 字符串转换为 ArrayBuffer
            const binaryString = atob(result.data)
            const bytes = new Uint8Array(binaryString.length)
            for (let i = 0; i < binaryString.length; i++) {
              bytes[i] = binaryString.charCodeAt(i)
            }

            // 创建 Blob
            const blob = new Blob([bytes], {
              type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
            })

            // 创建下载链接
            const url = window.URL.createObjectURL(blob)
            const link = document.createElement('a')
            link.href = url
            link.setAttribute('download', `${fileName}.xlsx`)
            document.body.appendChild(link)
            link.click()
            document.body.removeChild(link)
            window.URL.revokeObjectURL(url)

            this.$message.success('Excel导出成功')
          } catch (blobError) {
            console.error('Blob创建失败:', blobError)
            // 如果 Blob 方式失败，尝试直接下载
            this.downloadBase64File(result.data, `${fileName}.xlsx`, 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')
          }
        } else {
          this.$message.error('导出失败：' + (result.error || '未知错误'))
        }
      } catch (error) {
        console.error('导出失败:', error)
        this.$message.error('导出失败: ' + error.message)
      } finally {
        this.ipExportLoading = false
      }
    },
    // 备用的 Base64 文件下载方法
    downloadBase64File(base64Data, fileName, mimeType) {
      try {
        // 创建下载链接
        const link = document.createElement('a')
        link.href = `data:${mimeType};base64,${base64Data}`
        link.setAttribute('download', fileName)
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        this.$message.success('文件下载成功')
      } catch (error) {
        console.error('Base64下载失败:', error)
        this.$message.error('下载失败，请检查浏览器设置')
      }
    },
    // 复制到剪贴板
    copyToClipboard(text, type = '内容') {
      if (!text || text.trim() === '') {
        ElMessage.warning(`没有可复制的${type}内容`);
        return;
      }

      navigator.clipboard.writeText(text).then(() => {
        ElMessage.success(`${type}已复制到剪贴板`);
      }).catch(err => {
        console.error('复制失败:', err);
        // 备用方案
        const textArea = document.createElement('textarea');
        textArea.value = text;
        document.body.appendChild(textArea);
        textArea.select();
        try {
          document.execCommand('copy');
          ElMessage.success(`${type}已复制到剪贴板`);
        } catch (e) {
          ElMessage.error('复制失败，请手动复制');
        }
        document.body.removeChild(textArea);
      });
    },
  }
}
</script>

<style scoped>
.ip-query-wrapper {
  overflow-x: hidden;
  width: 100%;
  box-sizing: border-box;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: bold;
  color: #303133;
}

.input-stats {
  margin-top: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.stat-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  background-color: #f5f7fa;
  color: #606266;
}

.stat-item .label {
  color: #909399;
}

.stat-item .value {
  font-weight: 600;
  color: #303133;
}

.stat-item.total {
  background-color: #ecf5ff;
  color: #409eff;
}

.stat-item.completed {
  background-color: #f4f4f5;
  color: #909399;
}

.stat-item.success {
  background-color: #f0f9eb;
  color: #67c23a;
}

.stat-item.error {
  background-color: #fef0f0;
  color: #f56c6c;
}

/* 可点击复制的单元格样式 */
.copyable-cell {
  cursor: pointer;
  user-select: none;
  transition: all 0.2s;
  display: inline-block;
  padding: 2px 4px;
  border-radius: 2px;
}

.copyable-cell:hover {
  background-color: #f0f9ff;
  color: #409eff;
}

.copyable-cell:active {
  background-color: #e0f2fe;
}

/* 状态标签的悬停效果 */
.el-tag.copyable-cell:hover {
  opacity: 0.8;
  transform: scale(1.05);
}

</style>