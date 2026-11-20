<template>
  <div class="gitdumper-content">
    <!-- 功能选择 -->
    <div class="gitdumper-header-card glass-card">
      <h4>Git Dumper 功能(提取稍慢，请耐心等待)</h4>
      <div class="function-select">
        <el-radio-group v-model="gitdumperFunction" @change="onGitDumperFunctionChange">
          <el-radio value="single-scan">单URL扫描</el-radio>
          <el-radio value="batch-scan">批量扫描</el-radio>
        </el-radio-group>
      </div>
    </div>

    <!-- 批量扫描功能 -->
    <div v-if="gitdumperFunction === 'batch-scan'" class="batch-scan-section">
      <div class="batch-url-input-card glass-card">
        <div class="url-input-header">
          <h5>批量URL扫描</h5>
          <div class="batch-actions">
            <el-button
                @click="clearBatchUrls"
                size="small"
            >
              清空
            </el-button>
            <el-button
                type="primary"
                @click="startBatchScan"
                :loading="scanning"
                size="small"
            >
              {{ scanning ? '批量扫描中...' : '开始批量扫描' }}
            </el-button>
            <el-button
                v-if="scanning"
                type="danger"
                @click="cancelBatchScan"
                size="small"
            >
              取消扫描
            </el-button>
          </div>
        </div>

        <div class="batch-input-group">
          <el-input
              v-model="form.urlsText"
              type="textarea"
              placeholder="请输入多个目标URL，每行一个URL&#10;例如：&#10;https://example.com&#10;https://test.com/.git"
              :rows="8"
              class="batch-textarea"
              resize="none"
              @input="handleUrlsInput"
          />
        </div>

        <div class="batch-stats">
          <span class="stat-item">URL数量: {{ urlCount }}</span>
          <span class="stat-item">已完成: {{ completedCount }}</span>
          <span class="stat-item">成功: {{ successCount }}</span>
          <span class="stat-item">失败: {{ failedCount }}</span>
        </div>

        <!-- 批量扫描进度 -->
        <div class="batch-progress" v-if="scanning">
          <el-progress
              :percentage="batchProgress"
              :status="batchProgress === 100 ? 'success' : ''"
              :show-text="true"
          />
          <p class="progress-text">批量扫描进度: {{ batchProgress }}% ({{ completedCount }}/{{ urlCount }})</p>
        </div>
      </div>

      <!-- 批量扫描结果 - 表格展示 -->
      <div class="batch-results" v-if="showResults && tableData.length > 0">
        <div class="results-header">
          <h5>批量扫描结果 (共 {{ results.length }} 个URL)</h5>
          <div class="batch-result-actions">
            <el-button type="success" @click="exportResults" size="small" :loading="exportLoading">
              {{ exportLoading ? '导出中...' : '导出结果' }}
            </el-button>
            <el-button @click="clearResults" size="small">
              清空结果
            </el-button>
          </div>
        </div>

        <!-- 批量结果摘要 -->
        <div class="batch-summary">
          <div class="summary-card summary-total">
            <div class="summary-count">{{ results.length }}</div>
            <div class="summary-label">总任务数</div>
          </div>
          <div class="summary-card summary-success">
            <div class="summary-count">{{ successCount }}</div>
            <div class="summary-label">成功</div>
          </div>
          <div class="summary-card summary-failed">
            <div class="summary-count">{{ failedCount }}</div>
            <div class="summary-label">失败</div>
          </div>
          <div class="summary-card summary-rate">
            <div class="summary-count">{{ successRate }}%</div>
            <div class="summary-label">成功率</div>
          </div>
        </div>

        <!-- 结果表格 -->
        <div class="results-table-container">
          <el-table
              :data="tableData"
              stripe
              style="width: 100%"
              class="batch-results-table glass-card"
              v-loading="scanning"
          >
            <el-table-column prop="index" label="序号" width="80" align="center">
              <template #default="scope">
                {{ (currentPage - 1) * pageSize + scope.$index + 1 }}
              </template>
            </el-table-column>

            <el-table-column prop="url" label="目标URL" min-width="300">
              <template #default="scope">
                <span class="url-text">{{ scope.row.url }}</span>
              </template>
            </el-table-column>

            <el-table-column prop="status" label="状态" width="100" align="center">
              <template #default="scope">
                <el-tag :type="scope.row.success ? 'success' : 'danger'" size="small">
                  {{ scope.row.success ? '成功' : '失败' }}
                </el-tag>
              </template>
            </el-table-column>

            <el-table-column prop="message" label="状态信息" min-width="200">
              <template #default="scope">
                <span class="message-text">{{ scope.row.message }}</span>
              </template>
            </el-table-column>

            <el-table-column prop="outputDir" label="输出目录" min-width="200">
              <template #default="scope">
                <div v-if="scope.row.outputDir" class="output-dir-cell">
                  <span class="dir-path">{{ truncatePath(scope.row.outputDir) }}</span>
                  <el-button
                      v-if="scope.row.success"
                      type="text"
                      size="small"
                      @click="openDirectory(scope.row.outputDir)"
                      class="open-dir-btn"
                  >
                    <i class="el-icon-folder-opened"></i>
                    打开
                  </el-button>
                </div>
                <span v-else class="no-dir">-</span>
              </template>
            </el-table-column>

            <el-table-column prop="timestamp" label="扫描时间" width="180" align="center">
              <template #default="scope">
                {{ formatTime(scope.row.timestamp) }}
              </template>
            </el-table-column>

            <el-table-column label="操作" width="120" align="center" fixed="right">
              <template #default="scope">
                <el-button
                    size="small"
                    type="primary"
                    @click="retryScan(scope.row)"
                    :loading="scope.row.retrying"
                >
                  {{ scope.row.retrying ? '扫描中' : '重新扫描' }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <!-- 分页 -->
          <div class="pagination-container">
            <el-pagination
                v-model:current-page="currentPage"
                v-model:page-size="pageSize"
                :page-sizes="[10, 20, 50, 100]"
                :total="results.length"
                layout="total, sizes, prev, pager, next, jumper"
                @size-change="handleSizeChange"
                @current-change="handleCurrentChange"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 单URL扫描功能 -->
    <div v-if="gitdumperFunction === 'single-scan'" class="single-scan-section">
      <div class="single-url-input-card glass-card">
        <div class="url-input-header">
          <h5>单URL扫描</h5>
          <div class="single-actions">
            <el-button
                type="primary"
                @click="startSingleScan"
                :loading="scanning"
                :disabled="!canStartSingleScan"
            >
              {{ scanning ? '扫描中...' : '开始扫描' }}
            </el-button>
            <el-button
                v-if="scanning"
                type="danger"
                @click="cancelSingleScan"
            >
              取消扫描
            </el-button>
            <el-button
                @click="clearLogs"
                :disabled="scanning"
            >
              清空日志
            </el-button>
          </div>
        </div>

        <div class="single-input-group">
          <el-input
              v-model="singleUrl"
              placeholder="请输入目标URL，例如：https://example.com/.git"
              class="single-url-input"
              clearable
          />
        </div>
      </div>

      <!-- 扫描配置 -->
      <div class="scan-config-section">
        <div class="config-section">
          <h5>扫描配置</h5>
          <div class="config-grid">
            <div class="config-item">
              <label>输出目录：</label>
              <div class="directory-input-group">
                <el-input v-model="form.outputDir" placeholder="请选择输出目录" readonly>
                  <template #append>
                    <el-button @click="selectOutputDir">选择目录</el-button>
                  </template>
                </el-input>
              </div>
            </div>
            <div class="config-item">
              <label>并发数：</label>
              <div class="slider-container">
                <el-slider
                    v-model="form.jobs"
                    :min="1"
                    :max="20"
                    :step="1"
                    show-stops
                    show-input
                />
              </div>
            </div>
            <div class="config-item">
              <label>超时时间(秒)：</label>
              <div class="slider-container">
                <el-slider
                    v-model="form.timeout"
                    :min="10"
                    :max="120"
                    :step="5"
                    show-stops
                    show-input
                />
              </div>
            </div>
            <div class="config-item">
              <label>重试次数：</label>
              <div class="slider-container">
                <el-slider
                    v-model="form.retry"
                    :min="1"
                    :max="5"
                    :step="1"
                    show-stops
                    show-input
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 实时日志展示 -->
      <div class="log-section" v-if="gitdumperFunction === 'single-scan'">
        <div class="log-header">
          <h5>扫描日志</h5>
          <div class="log-actions">
            <el-button
                type="text"
                @click="toggleAutoScroll"
                :class="{ 'active': autoScroll }"
            >
              {{ autoScroll ? '暂停滚动' : '自动滚动' }}
            </el-button>
            <el-button
                type="text"
                @click="copyLogs"
            >
              复制日志
            </el-button>
            <el-button
                type="text"
                @click="clearLogs"
            >
              清空日志
            </el-button>
          </div>
        </div>

        <div class="log-container" ref="logContainer">
          <div
              v-for="(log, index) in scanLogs"
              :key="index"
              class="log-item"
              :class="getLogLevelClass(log)"
          >
            <span class="log-time">{{ formatLogTime(log.timestamp) }}</span>
            <span class="log-content">{{ log.message }}</span>
          </div>
          <div v-if="scanLogs.length === 0" class="log-empty">
            暂无日志，开始扫描后这里将显示详细的操作日志...
          </div>
        </div>

        <div class="log-stats" v-if="scanLogs.length > 0">
          <span class="stat-item">日志数量: {{ scanLogs.length }}</span>
          <span class="stat-item">最新: {{ formatLogTime(latestLogTime) }}</span>
        </div>
      </div>

      <!-- 单URL扫描结果 -->
      <div class="single-results" v-if="singleResult">
        <div class="results-header">
          <h5>扫描结果</h5>
          <div class="single-result-actions">
            <el-button
                type="success"
                @click="exportSingleResult"
                :disabled="!singleResult"
            >
              导出结果
            </el-button>
            <el-button
                type="primary"
                @click="retrySingleScan"
                :loading="singleResult.retrying"
            >
              {{ singleResult.retrying ? '重新扫描中...' : '重新扫描' }}
            </el-button>
          </div>
        </div>

        <div class="single-result-card" :class="singleResult.success ? 'result-success' : 'result-failed'">
          <div class="result-status-line">
            <el-tag :type="singleResult.success ? 'success' : 'danger'" size="large">
              {{ singleResult.success ? '扫描成功' : '扫描失败' }}
            </el-tag>
            <span class="result-time">{{ formatTime(singleResult.timestamp) }}</span>
          </div>

          <div class="result-details">
            <div class="detail-row">
              <span class="detail-label">目标URL:</span>
              <span class="detail-value url-value">{{ singleResult.url }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">状态信息:</span>
              <span class="detail-value">{{ singleResult.message }}</span>
            </div>
            <div class="detail-row" v-if="singleResult.outputDir">
              <span class="detail-label">输出目录:</span>
              <span class="detail-value output-dir">
                {{ singleResult.outputDir }}
                <el-button
                    type="text"
                    @click="openDirectory(singleResult.outputDir)"
                    v-if="singleResult.success"
                    class="open-dir-btn"
                >
                  <i class="el-icon-folder-opened"></i>
                  打开目录
                </el-button>
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 详情对话框 -->
    <el-dialog
        v-model="detailDialogVisible"
        :title="detailDialogTitle"
        width="600px"
    >
      <div class="detail-content">
        <div class="detail-item" v-if="currentDetail">
          <div class="detail-row">
            <span class="detail-label">URL:</span>
            <span class="detail-value">{{ currentDetail.url }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">状态:</span>
            <el-tag :type="currentDetail.success ? 'success' : 'danger'">
              {{ currentDetail.success ? '成功' : '失败' }}
            </el-tag>
          </div>
          <div class="detail-row">
            <span class="detail-label">状态信息:</span>
            <span class="detail-value">{{ currentDetail.message }}</span>
          </div>
          <div class="detail-row" v-if="currentDetail.outputDir">
            <span class="detail-label">输出目录:</span>
            <span class="detail-value">{{ currentDetail.outputDir }}</span>
          </div>
          <div class="detail-row" v-if="currentDetail.timestamp">
            <span class="detail-label">扫描时间:</span>
            <span class="detail-value">{{ formatTime(currentDetail.timestamp) }}</span>
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="detailDialogVisible = false">关闭</el-button>
        <el-button
            type="primary"
            @click="openDirectory(currentDetail.outputDir)"
            v-if="currentDetail && currentDetail.success"
        >
          打开目录
        </el-button>
        <el-button
            type="primary"
            @click="retryCurrentDetail"
            v-if="currentDetail && !currentDetail.success"
        >
          重新扫描
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { OpenPath, GetOpenDir } from "../../../../wailsjs/go/system/System";
import { BatchScan, GetDefaultOutputDir, RetryScan } from "../../../../wailsjs/go/fileDumper/GitDumperApp";
import {CancelGitScan} from "../../../../wailsjs/go/fileDumper/GitDumperApp";

export default {
  name: 'GitDumper',
  data() {
    return {
      gitdumperFunction: 'single-scan',
      scanning: false,
      showResults: false,
      selectingDir: false,
      exportLoading: false,
      detailDialogVisible: false,
      currentDetail: null,
      form: {
        urlsText: '',
        outputDir: '',
        jobs: 5,
        timeout: 30,
        retry: 3
      },
      singleUrl: '',
      singleResult: null,
      results: [],
      completedCount: 0,
      batchProgress: 0,
      urlCount: 0,
      // 分页相关数据
      currentPage: 1,
      pageSize: 10,
      // 日志相关数据
      scanLogs: [],
      autoScroll: true,
      logListener: null,
      cancelToken: null
    }
  },
  computed: {
    canStartSingleScan() {
      return this.singleUrl.trim() &&
          this.form.outputDir &&
          !this.scanning
    },
    successCount() {
      return this.results.filter(r => r.success).length
    },
    failedCount() {
      return this.results.filter(r => !r.success).length
    },
    successRate() {
      if (this.results.length === 0) return 0
      return ((this.successCount / this.results.length) * 100).toFixed(1)
    },
    detailDialogTitle() {
      return this.currentDetail ? `扫描详情 - ${this.currentDetail.success ? '成功' : '失败'}` : '扫描详情'
    },
    // 表格数据 - 根据分页计算当前显示的数据
    tableData() {
      const start = (this.currentPage - 1) * this.pageSize
      const end = start + this.pageSize
      return this.results.slice(start, end)
    },
    // 最新日志时间
    latestLogTime() {
      if (this.scanLogs.length === 0) return null
      return this.scanLogs[this.scanLogs.length - 1].timestamp
    }
  },
  async mounted() {
    // 获取默认输出目录
    try {
      const defaultDir = await GetDefaultOutputDir()
      if (defaultDir) {
        this.form.outputDir = defaultDir
      }
    } catch (error) {
      console.error('Failed to get default directory:', error)
      this.$message.error('获取默认目录失败')
    }

    // 设置日志监听
    this.setupLogListener()
  },
  beforeUnmount() {
    // 清理日志监听
    if (this.logListener) {
      this.logListener()
    }
  },
  watch: {
    scanLogs() {
      if (this.autoScroll) {
        this.$nextTick(() => {
          this.scrollToBottom()
        })
      }
    }
  },
  methods: {
    // 设置日志监听
    setupLogListener() {
      if (window.runtime && window.runtime.EventsOn) {
        this.logListener = window.runtime.EventsOn('git-dumper-log', (data) => {
          this.addLog(data.message, data.level || 'info')
        })
      }
    },

    // 添加日志
    addLog(message, level = 'info') {
      const logEntry = {
        message: message,
        level: level,
        timestamp: new Date().toISOString()
      }
      this.scanLogs.push(logEntry)

      // 限制日志数量，避免内存占用过多
      if (this.scanLogs.length > 1000) {
        this.scanLogs = this.scanLogs.slice(-500)
      }
    },

    // 清空日志
    clearLogs() {
      this.scanLogs = []
    },

    // 切换自动滚动
    toggleAutoScroll() {
      this.autoScroll = !this.autoScroll
      if (this.autoScroll) {
        this.scrollToBottom()
      }
    },

    // 滚动到底部
    scrollToBottom() {
      const container = this.$refs.logContainer
      if (container) {
        container.scrollTop = container.scrollHeight
      }
    },

    // 复制日志
    copyLogs() {
      if (this.scanLogs.length === 0) {
        this.$message.warning('没有日志内容可复制')
        return
      }

      const logText = this.scanLogs.map(log =>
          `[${this.formatLogTime(log.timestamp)}] ${log.message}`
      ).join('\n')

      navigator.clipboard.writeText(logText).then(() => {
        this.$message.success('日志已复制到剪贴板')
      }).catch(err => {
        console.error('复制失败:', err)
        this.$message.error('复制失败')
      })
    },

    // 获取日志级别对应的CSS类
    getLogLevelClass(log) {
      return `log-level-${log.level}`
    },

    // 格式化日志时间
    formatLogTime(timestamp) {
      if (!timestamp) return ''
      const date = new Date(timestamp)
      return date.toLocaleTimeString('zh-CN', {
        hour12: false,
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      })
    },

    onGitDumperFunctionChange() {
      this.results = []
      this.singleResult = null
      this.showResults = false
      // 重置分页
      this.currentPage = 1
    },

    handleUrlsInput() {
      const text = this.form.urlsText.trim()
      if (!text) {
        this.urlCount = 0
        return
      }

      const lines = text.split('\n')
          .map(line => line.trim())
          .filter(line => line.length > 0)
          .filter(line => line.startsWith('http://') || line.startsWith('https://'))

      this.urlCount = lines.length
    },

    isValidUrl(string) {
      try {
        const url = new URL(string)
        return url.protocol === 'http:' || url.protocol === 'https:'
      } catch (_) {
        return false
      }
    },

    parseUrlsFromText(text) {
      return text.split('\n')
          .map(line => line.trim())
          .filter(line => line.length > 0)
          .filter(line => this.isValidUrl(line))
    },

    async selectOutputDir() {
      try {
        const selectedDir = await GetOpenDir()
        if (selectedDir) {
          this.form.outputDir = selectedDir
        }
      } catch (error) {
        console.error('Failed to select directory:', error)
        this.$message.error('选择目录失败')
      }
    },

    async startBatchScan() {
      const urls = this.parseUrlsFromText(this.form.urlsText)
      if (urls.length === 0) {
        this.$message.warning('请输入至少一个有效的URL')
        return
      }

      if (!this.form.outputDir) {
        this.$message.warning('请选择输出目录')
        return
      }

      this.scanning = true
      this.showResults = true
      this.completedCount = 0
      this.batchProgress = 0
      this.cancelToken = Date.now().toString()

      this.$message.info(`开始批量扫描 ${urls.length} 个URL...`)

      try {
        const request = {
          urls: urls,
          outputDir: this.form.outputDir,
          jobs: this.form.jobs,
          timeout: this.form.timeout,
          retry: this.form.retry,
          cancelToken: this.cancelToken
        }

        const results = await BatchScan(request)

        if (Array.isArray(results)) {
          this.results = results.map(result => ({
            ...result,
            timestamp: result.timestamp || new Date().toISOString(),
            retrying: false
          }))

          this.completedCount = this.results.length
          this.batchProgress = 100

          if (this.successCount > 0) {
            this.$message.success(`扫描完成！成功 ${this.successCount} 个，失败 ${this.failedCount} 个`)
          } else {
            this.$message.warning('扫描完成，但没有成功下载任何仓库')
          }
        } else {
          throw new Error('返回结果格式错误')
        }
      } catch (error) {
        console.error('Scan failed:', error)
        this.$message.error('扫描失败: ' + error.message)
      } finally {
        this.scanning = false
        this.cancelToken = null
      }
    },

    async startSingleScan() {
      if (!this.singleUrl.trim()) {
        this.$message.warning('请输入目标URL')
        return
      }

      if (!this.form.outputDir) {
        this.$message.warning('请选择输出目录')
        return
      }

      this.scanning = true
      // 清空之前的日志
      this.clearLogs()
      this.cancelToken = Date.now().toString()

      // 添加开始扫描的日志
      this.addLog(`开始扫描: ${this.singleUrl}`, 'info')

      this.$message.info(`开始扫描: ${this.singleUrl}`)

      try {
        const request = {
          urls: [this.singleUrl],
          outputDir: this.form.outputDir,
          jobs: this.form.jobs,
          timeout: this.form.timeout,
          retry: this.form.retry,
          cancelToken: this.cancelToken
        }

        // 在扫描开始前添加模拟日志（实际应该由后端发送）
        this.addLog('正在初始化下载器...', 'info')
        this.addLog(`目标URL: ${this.singleUrl}`, 'info')
        this.addLog(`输出目录: ${this.form.outputDir}`, 'info')
        this.addLog(`并发数: ${this.form.jobs}, 超时: ${this.form.timeout}s, 重试: ${this.form.retry}次`, 'info')

        const results = await BatchScan(request)

        if (results && Array.isArray(results) && results.length > 0) {
          this.singleResult = {
            ...results[0],
            timestamp: results[0].timestamp || new Date().toISOString(),
            retrying: false
          }

          if (this.singleResult.success) {
            this.addLog('扫描成功完成！', 'success')
            this.$message.success('扫描成功！')
          } else {
            this.addLog(`扫描失败: ${this.singleResult.message}`, 'error')
            this.$message.warning('扫描失败: ' + this.singleResult.message)
          }
        } else {
          throw new Error('未返回扫描结果')
        }
      } catch (error) {
        console.error('Single scan failed:', error)
        this.addLog(`扫描异常: ${error.message}`, 'error')
        this.$message.error('扫描失败: ' + error.message)
      } finally {
        this.scanning = false
        this.cancelToken = null
      }
    },

    async cancelBatchScan() {
      if (this.cancelToken) {
        try {
          await CancelGitScan(this.cancelToken); // 使用正确的函数名
          this.scanning = false;
          this.cancelToken = null;
          this.$message.info('批量扫描已取消');
        } catch (error) {
          console.error('Cancel failed:', error);
          // 即使取消失败也重置状态
          this.scanning = false;
          this.cancelToken = null;
          this.$message.info('扫描已停止');
        }
      }
    },

    async cancelSingleScan() {
      if (this.cancelToken) {
        try {
          await CancelGitScan(this.cancelToken); // 使用正确的函数名
          this.scanning = false;
          this.cancelToken = null;
          this.$message.info('扫描已取消');
        } catch (error) {
          console.error('Cancel failed:', error);
          // 即使取消失败也重置状态
          this.scanning = false;
          this.cancelToken = null;
          this.$message.info('扫描已停止');
        }
      }
    },

    async retryScan(result) {
      result.retrying = true

      this.$message.info(`重新扫描: ${result.url}`)

      try {
        const newResult = await RetryScan({
          url: result.url,
          outputDir: this.form.outputDir,
          jobs: this.form.jobs,
          timeout: this.form.timeout,
          retry: this.form.retry
        })

        Object.assign(result, newResult, {
          timestamp: new Date().toISOString(),
          retrying: false
        })

        if (result.success) {
          this.$message.success('重新扫描成功！')
        } else {
          this.$message.warning('重新扫描失败: ' + result.message)
        }
      } catch (error) {
        console.error('Retry scan failed:', error)
        result.retrying = false
        this.$message.error('重新扫描失败: ' + error.message)
      } finally {
        this.$forceUpdate()
      }
    },

    async retrySingleScan() {
      if (!this.singleUrl) {
        this.$message.warning('请输入目标URL')
        return
      }

      this.singleResult.retrying = true
      this.$message.info(`重新扫描: ${this.singleUrl}`)

      try {
        const newResult = await RetryScan({
          url: this.singleUrl,
          outputDir: this.form.outputDir,
          jobs: this.form.jobs,
          timeout: this.form.timeout,
          retry: this.form.retry
        })

        this.singleResult = {
          ...newResult,
          timestamp: new Date().toISOString(),
          retrying: false
        }

        if (this.singleResult.success) {
          this.$message.success('重新扫描成功！')
        } else {
          this.$message.warning('重新扫描失败: ' + this.singleResult.message)
        }
      } catch (error) {
        console.error('Retry single scan failed:', error)
        this.singleResult.retrying = false
        this.$message.error('重新扫描失败: ' + error.message)
      }
    },

    async retryCurrentDetail() {
      if (this.currentDetail) {
        await this.retryScan(this.currentDetail)
        this.detailDialogVisible = false
      }
    },

    clearBatchUrls() {
      this.form.urlsText = ''
      this.urlCount = 0
    },

    clearResults() {
      this.results = []
      this.currentPage = 1
      this.$message.info('已清空扫描结果')
    },

    openDirectory(dirPath) {
      if (dirPath) {
        OpenPath(dirPath)
      } else {
        this.$message.warning('输出目录不存在')
      }
    },

    truncatePath(path) {
      if (!path) return ''
      if (path.length > 40) {
        return '...' + path.slice(-37)
      }
      return path
    },

    formatTime(timestamp) {
      if (!timestamp) return ''
      return new Date(timestamp).toLocaleString('zh-CN')
    },

    // 分页相关方法
    handleSizeChange(val) {
      this.pageSize = val
      this.currentPage = 1
    },

    handleCurrentChange(val) {
      this.currentPage = val
    },

    exportResults() {
      if (this.results.length === 0) {
        this.$message.warning('没有可导出的结果数据')
        return
      }

      this.exportLoading = true

      try {
        const data = {
          scanTime: new Date().toISOString(),
          config: { ...this.form },
          summary: {
            total: this.results.length,
            success: this.successCount,
            failed: this.failedCount,
            successRate: this.successRate
          },
          results: this.results
        }

        const blob = new Blob([JSON.stringify(data, null, 2)], {
          type: 'application/json'
        })

        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `git-dumper-results-${new Date().getTime()}.json`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(url)

        this.$message.success('结果导出成功')
      } catch (error) {
        console.error('Export failed:', error)
        this.$message.error('导出失败: ' + error.message)
      } finally {
        this.exportLoading = false
      }
    },

    exportSingleResult() {
      if (!this.singleResult) {
        this.$message.warning('没有可导出的结果数据')
        return
      }

      this.exportLoading = true

      try {
        const data = {
          scanTime: new Date().toISOString(),
          config: { ...this.form, singleUrl: this.singleUrl },
          result: this.singleResult
        }

        const blob = new Blob([JSON.stringify(data, null, 2)], {
          type: 'application/json'
        })

        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `git-dumper-single-result-${new Date().getTime()}.json`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(url)

        this.$message.success('结果导出成功')
      } catch (error) {
        console.error('Export failed:', error)
        this.$message.error('导出失败: ' + error.message)
      } finally {
        this.exportLoading = false
      }
    }
  }
}
</script>

<style scoped>
.gitdumper-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
}

.gitdumper-header-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.function-select {
  margin-bottom: 16px;
}

/* 批量扫描区域 */
.batch-scan-section,
.single-scan-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.batch-url-input-card,
.single-url-input-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.url-input-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.url-input-header h5 {
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.batch-actions {
  display: flex;
  gap: 8px;
}

.batch-input-group,
.single-input-group {
  margin-bottom: 16px;
}

.batch-textarea {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
}

.single-url-input {
  width: 100%;
}

:deep(.single-url-input .el-input-group__append) {
  background-color: #409eff;
  border-color: #409eff;
  color: white;
}

:deep(.single-url-input .el-input-group__append .el-button) {
  color: white;
  border: none;
}

.batch-stats {
  display: flex;
  gap: 20px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 6px;
  flex-wrap: wrap;
}

.stat-item {
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

.batch-progress {
  margin-top: 16px;
}

.progress-text {
  text-align: center;
  margin-top: 8px;
  color: #909399;
  font-size: 14px;
}

/* 扫描配置区域 */
.scan-config-section {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.config-section {
  margin-bottom: 0;
}

.config-section h5 {
  margin: 0 0 16px 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
  padding-bottom: 8px;
  border-bottom: 1px solid #e4e7ed;
}


.config-input {
  width: 100%;
}

.directory-input-group {
  width: 100%;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}

.config-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.config-item label {
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

.slider-container {
  padding: 5px 0;
}

:deep(.slider-container .el-slider) {
  margin-top: 5px;
}

/* 结果区域 */
.batch-results,
.single-results {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.results-header h5 {
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.batch-result-actions {
  display: flex;
  gap: 8px;
}

/* 批量结果摘要 */
.batch-summary {
  display: flex;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.summary-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px;
  border-radius: 8px;
  min-width: 120px;
  text-align: center;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.summary-card .summary-count {
  font-size: 28px;
  font-weight: bold;
  margin-bottom: 8px;
}

.summary-card .summary-label {
  font-size: 14px;
  font-weight: 500;
}

.summary-total {
  background: #f0f9ff;
  color: #409eff;
  border: 1px solid #b3d8ff;
}

.summary-success {
  background: #f0f9f4;
  color: #67c23a;
  border: 1px solid #c2e7b0;
}

.summary-failed {
  background: #fef0f0;
  color: #f56c6c;
  border: 1px solid #fbc4c4;
}

.summary-rate {
  background: #fdf4ff;
  color: #c456d6;
  border: 1px solid #e8c5f0;
}

/* 表格样式 */
.results-table-container {
  margin-top: 16px;
}

.batch-results-table {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  overflow: hidden;
}

:deep(.batch-results-table .el-table__header-wrapper) {
  background: #f5f7fa;
}

:deep(.batch-results-table .el-table th) {
  background: #f5f7fa;
  color: #606266;
  font-weight: 600;
}

.url-text {
  color: #409eff;
  word-break: break-all;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
}

.message-text {
  word-break: break-all;
  font-size: 12px;
}

.output-dir-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.dir-path {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  color: #909399;
  flex: 1;
  min-width: 0;
  word-break: break-all;
}

.open-dir-btn {
  padding: 2px 6px;
  font-size: 12px;
  flex-shrink: 0;
}

.no-dir {
  color: #c0c4cc;
  font-style: italic;
}

/* 分页样式 */
.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
  padding: 16px 0;
}

/* 单URL结果卡片 */
.single-result-card {
  padding: 24px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: white;
  transition: all 0.3s ease;
}

.single-result-card.result-success {
  border-left: 4px solid #67c23a;
}

.single-result-card.result-failed {
  border-left: 4px solid #f56c6c;
}

.result-status-line {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e4e7ed;
}

.result-details {
  margin-bottom: 20px;
}

.detail-row {
  display: flex;
  margin-bottom: 12px;
  align-items: flex-start;
}

.detail-row .detail-label {
  min-width: 100px;
}

.url-value {
  color: #409eff;
  font-weight: 500;
}

.result-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

/* 详情对话框 */
.detail-content {
  padding: 16px 0;
}

.detail-content .detail-row {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
  padding: 8px 0;
}

.detail-content .detail-label {
  min-width: 100px;
  font-weight: 600;
  color: #606266;
}

.detail-content .detail-value {
  flex: 1;
  color: #303133;
  word-break: break-all;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .batch-stats {
    flex-direction: column;
    gap: 8px;
  }

  .batch-summary {
    justify-content: center;
  }

  .summary-card {
    min-width: 100px;
    padding: 16px;
  }

  .config-grid {
    grid-template-columns: 1fr;
  }

  .url-input-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .results-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .pagination-container {
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .gitdumper-header-card,
  .batch-url-input-card,
  .single-url-input-card,
  .scan-config-section,
  .batch-results,
  .single-results {
    padding: 16px;
  }

  .detail-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .detail-label {
    min-width: auto;
  }
}

.single-actions {
  display: flex;
  gap: 8px;
}

.single-result-actions {
  display: flex;
  gap: 8px;
}
/* 日志区域样式 */
.log-section {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  margin-bottom: 16px;
}

.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.log-header h5 {
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.log-actions {
  display: flex;
  gap: 8px;
}

.log-actions .el-button {
  padding: 4px 8px;
  font-size: 12px;
}

.log-actions .el-button.active {
  color: #409eff;
  font-weight: 500;
}

.log-container {
  height: 300px;
  overflow-y: auto;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 12px;
  background: #fafafa;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  line-height: 1.4;
}

.log-item {
  margin-bottom: 4px;
  padding: 2px 0;
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.log-time {
  color: #909399;
  font-size: 11px;
  min-width: 80px;
  flex-shrink: 0;
}

.log-content {
  color: #606266;
  word-break: break-all;
  flex: 1;
}

/* 日志级别样式 */
.log-level-info .log-content {
  color: #606266;
}

.log-level-success .log-content {
  color: #67c23a;
  font-weight: 500;
}

.log-level-warning .log-content {
  color: #e6a23c;
  font-weight: 500;
}

.log-level-error .log-content {
  color: #f56c6c;
  font-weight: 500;
}

.log-level-debug .log-content {
  color: #909399;
}

.log-empty {
  text-align: center;
  color: #c0c4cc;
  font-style: italic;
  padding: 40px 0;
}

.log-stats {
  display: flex;
  gap: 16px;
  margin-top: 12px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
  color: #909399;
}

.stat-item {
  display: flex;
  align-items: center;
}

/* 滚动条样式 */
.log-container::-webkit-scrollbar {
  width: 6px;
}

.log-container::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.log-container::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.log-container::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .log-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .log-actions {
    align-self: flex-end;
  }

  .log-item {
    flex-direction: column;
    gap: 2px;
  }

  .log-time {
    min-width: auto;
  }
}
</style>