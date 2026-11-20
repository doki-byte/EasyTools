<template>
  <div class="oss-content">
    <!-- 功能选择 -->
    <div class="oss-header-card glass-card">
      <h4>OSS 存储桶功能</h4>
      <div class="function-select">
        <el-radio-group v-model="ossFunction" @change="onOssFunctionChange">
          <el-radio label="vuln-scan">漏洞扫描</el-radio>
          <el-radio label="file-list">文件遍历</el-radio>
          <el-radio label="batch-scan">批量扫描</el-radio>
        </el-radio-group>
      </div>
    </div>

    <!-- 公共URL输入区域 -->
    <div class="oss-url-section" v-if="ossFunction !== 'batch-scan'">
      <div class="url-input-card glass-card">
        <div class="url-input-header">
          <h5>存储桶URL配置</h5>
          <el-button
              v-if="ossFunction === 'vuln-scan'"
              @click="autoDetectCloudProvider"
              :loading="detecting"
              size="small"
              type="primary"
          >
            {{ detecting ? '识别中...' : '智能识别' }}
          </el-button>
        </div>

        <div class="url-input-group">
          <el-input
              v-model="ossBucketURL"
              placeholder="请输入OSS存储桶链接，系统将自动识别云厂商和区域"
              class="url-input"
              clearable
          >
            <template #append>
              <el-button
                  v-if="ossFunction === 'file-list'"
                  type="primary"
                  @click="generateOssListQueries"
                  :loading="fileListLoading"
              >
                {{ fileListLoading ? '获取中...' : '获取数据' }}
              </el-button>
              <el-button
                  v-else
                  type="primary"
                  @click="startVulnScan"
                  :loading="scanLoading"
              >
                {{ scanLoading ? '扫描中...' : '开始扫描' }}
              </el-button>
            </template>
          </el-input>
        </div>

        <!-- 识别结果展示 -->
        <div class="detection-result" v-if="detectionResult.identified && ossFunction === 'vuln-scan'">
          <el-alert
              :title="`已识别为: ${detectionResult.cloudName} | 区域: ${detectionResult.region} | Bucket: ${detectionResult.bucket}`"
              type="success"
              :closable="false"
              show-icon
          />
        </div>

      </div>
    </div>

    <!-- 批量扫描功能 -->
    <div v-if="ossFunction === 'batch-scan'" class="batch-scan-section">
      <div class="batch-url-input-card glass-card">
        <div class="url-input-header">
          <h5>批量URL扫描</h5>
          <div class="batch-actions">
            <el-button
                @click="clearBatchUrls"
                size="small"
                :disabled="batchScanLoading"
            >
              清空
            </el-button>
            <el-button
                v-if="batchScanLoading"
                size="small"
                @click="toggleBatchScanPause"
            >
              {{ batchScanPaused ? '继续' : '暂停' }}
            </el-button>
            <el-button
                v-if="batchScanLoading"
                type="danger"
                size="small"
                plain
                @click="stopBatchScan"
            >
              停止
            </el-button>
            <el-button
                type="primary"
                @click="startBatchScan"
                :loading="batchScanLoading"
                size="small"
            >
              {{ batchScanLoading ? (batchScanPaused ? '已暂停…' : '批量扫描中...') : '开始批量扫描' }}
            </el-button>
          </div>
        </div>

        <el-alert
            v-if="scanConfig.scanOptions.includes('scan_delete_perm')"
            type="warning"
            :closable="false"
            show-icon
            class="batch-delete-warning"
            title="当前扫描选项包含 DELETE 权限测试，批量扫描将对每个 URL 尝试删除本工具创建的临时对象；请务必确认已获授权且勿对生产桶滥用。"
        />

        <div class="batch-input-group">
          <el-input
              v-model="batchUrls"
              type="textarea"
              placeholder="请输入多个OSS存储桶链接，每行一个URL"
              :rows="8"
              class="batch-textarea"
              resize="none"
          />
        </div>

        <div class="batch-stats">
          <span class="stat-item">URL数量: {{ urlCount }}</span>
          <span class="stat-item">已完成: {{ completedCount }}</span>
          <span class="stat-item">成功: {{ successCount }}</span>
          <span class="stat-item">失败: {{ failedCount }}</span>
        </div>

        <!-- 批量扫描进度 -->
        <div class="batch-progress" v-if="batchScanLoading">
          <el-progress
              :percentage="batchProgress"
              :status="batchProgress === 100 ? 'success' : ''"
              :show-text="true"
          />
          <p class="progress-text">
            {{ batchScanPaused ? '已暂停 — ' : '' }}批量扫描进度: {{ batchProgress }}% ({{ completedCount }}/{{ urlCount }})
          </p>
        </div>
      </div>

      <!-- 批量扫描结果 -->
      <div class="batch-results" v-if="batchResults.length > 0">
        <div class="results-header">
          <h5>批量扫描结果 (共 {{ batchResults.length }} 个URL)</h5>
          <div class="batch-result-actions">
            <el-button type="success" @click="exportBatchResults" size="small" :loading="exportLoading">
              {{ exportLoading ? '导出中...' : '导出结果' }}
            </el-button>
            <el-button @click="clearBatchResults" size="small">
              清空结果
            </el-button>
          </div>
        </div>

        <!-- 添加导出配置对话框 -->
        <el-dialog
            v-model="exportDialogVisible"
            title="导出配置"
            width="500px"
            :close-on-click-modal="false"
        >
          <div class="export-config">
            <el-form label-width="120px">
              <el-form-item label="导出格式">
                <el-radio-group v-model="exportConfig.format">
                  <el-radio label="excel">Excel文件 (.xlsx)</el-radio>
                  <el-radio label="csv">CSV文件 (.csv)</el-radio>
                </el-radio-group>
              </el-form-item>

              <el-form-item label="文件名">
                <el-input
                    v-model="exportConfig.filename"
                    placeholder="请输入文件名"
                    clearable
                >
                  <template #append>
                    <span class="file-extension">.{{ exportConfig.format }}</span>
                  </template>
                </el-input>
              </el-form-item>

              <el-form-item label="包含内容">
                <el-checkbox-group v-model="exportConfig.include">
                  <el-checkbox label="summary">风险摘要</el-checkbox>
                  <el-checkbox label="details">详细结果</el-checkbox>
                  <el-checkbox label="failed">失败记录</el-checkbox>
                </el-checkbox-group>
              </el-form-item>

              <el-form-item label="时间范围">
                <el-date-picker
                    v-model="exportConfig.dateRange"
                    type="daterange"
                    range-separator="至"
                    start-placeholder="开始日期"
                    end-placeholder="结束日期"
                    value-format="YYYY-MM-DD"
                />
              </el-form-item>
            </el-form>

            <div class="export-preview">
              <h5>导出预览 (共 {{ exportDataCount }} 条记录)</h5>
              <div class="preview-stats">
                <span>成功扫描: {{ exportSuccessCount }} 个URL</span>
                <span>风险项: {{ exportRiskCount }} 个</span>
                <span>失败扫描: {{ exportFailedCount }} 个</span>
              </div>
            </div>
          </div>

          <template #footer>
            <el-button @click="exportDialogVisible = false">取消</el-button>
            <el-button type="primary" @click="confirmExport" :loading="exportLoading">
              确认导出
            </el-button>
          </template>
        </el-dialog>

        <!-- 批量结果摘要 -->
        <div class="batch-summary">
          <div class="summary-card" v-for="(count, level) in batchRiskSummary" :key="level"
               :class="['summary-item', getRiskClass(level)]">
            <div class="summary-count">{{ count }}</div>
            <div class="summary-label">{{ getRiskLabel(level) }}</div>
          </div>
        </div>

        <!-- 批量结果列表 -->
        <div class="batch-results-list">
          <div v-for="(result, index) in batchResults" :key="index"
               class="batch-result-item" :class="getRiskClass(result.highestRisk)">
            <div class="batch-result-header">
              <div class="result-url">
                <span class="url-label">URL:</span>
                <span class="url-text" @click="openResultURL(result.url)">{{ result.url }}</span>
              </div>
              <div class="result-status">
                <el-tag :type="getStatusType(result.status)" size="small">
                  {{ result.status }}
                </el-tag>
                <span class="result-counts">
                  风险: {{ result.riskCount }} 项
                </span>
              </div>
            </div>

            <div class="batch-result-details" v-if="result.results && result.results.length > 0">
              <div v-for="(item, itemIndex) in result.results.slice(0, 3)" :key="itemIndex"
                   class="detail-item" :class="getRiskClass(item.risk)">
                <span class="risk-badge">{{ getRiskLabel(item.risk) }}</span>
                <span class="risk-msg">{{ item.msg }}</span>
              </div>
              <div v-if="result.results.length > 3" class="more-results">
                还有 {{ result.results.length - 3 }} 个风险项...
              </div>
            </div>

            <div class="batch-result-details" v-else-if="result.error">
              <div class="error-item">
                <span class="error-msg">{{ result.error }}</span>
              </div>
            </div>

            <div class="result-actions">
              <el-button size="small" @click="viewDetailedResult(result)">
                查看详情
              </el-button>
              <el-button size="small" type="primary" @click="openResultURL(result.url)">
                打开URL
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 文件遍历功能结果 -->
    <div v-if="ossFunction === 'file-list'" class="oss-function-section">
      <div class="oss-result-card glass-card" v-if="OssListSuccess">
        <div class="result-content-compact">
          <div class="result-info-compact">
            <span class="result-label-compact">文件保存位置：</span>
            <span class="result-path-compact">{{ OssListSavePath }}</span>
          </div>
          <div class="result-actions-compact">
            <el-button type="success" @click="openXlsxFileDir" size="small" class="open-folder-btn">
              打开文件夹
            </el-button>
            <el-button type="primary" @click="loadCSVData" size="small" :loading="csvLoading">
              {{ csvLoading ? '加载中...' : '查看CSV数据' }}
            </el-button>
            <el-button type="info" @click="showKkfileConfig" size="small">
              预览配置
            </el-button>
          </div>
        </div>
      </div>

      <!-- CSV数据表格 -->
      <div class="csv-table-section glass-card" v-if="csvData && csvData.length > 0">
        <div class="csv-table-header">
          <h5>CSV数据预览 (共 {{ csvTotalCount }} 条记录{{ (csvSearchText || csvFileTypeFilter) ? '（已过滤）' : '' }})</h5>
          <div class="csv-table-actions">
            <el-select
                v-model="csvFileTypeFilter"
                placeholder="文件类型"
                size="small"
                style="width: 150px; margin-right: 8px;"
                clearable
                filterable
            >
              <el-option
                  v-for="fileType in csvFileTypes"
                  :key="fileType"
                  :label="fileType"
                  :value="fileType"
              />
            </el-select>
            <el-input
                v-model="csvSearchText"
                placeholder="搜索..."
                size="small"
                style="width: 200px; margin-right: 8px;"
                clearable
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
            <el-button @click="clearCSVData" size="small">清空</el-button>
          </div>
        </div>

        <div class="csv-table-container">
          <el-table
              :data="filteredCSVData"
              stripe
              border
              :max-height="csvTableHeight"
              style="width: 100%"
          >
            <el-table-column label="序号" width="60" fixed="left">
              <template #default="{ $index }">
                {{ (csvCurrentPage - 1) * csvPageSize + $index + 1 }}
              </template>
            </el-table-column>
            <el-table-column
                v-for="(header, index) in csvHeaders"
                :key="index"
                :prop="`col_${index}`"
                :label="header"
                :min-width="header === 'url' ? 300 : 120"
            >
              <template #default="{ row }">
                <div v-if="header === 'url'" class="url-cell-clickable" @click="previewFile(row[`col_${index}`])">
                  {{ row[`col_${index}`] }}
                </div>
                <div v-else class="csv-cell-content">{{ row[`col_${index}`] }}</div>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="csv-pagination" v-if="csvData.length > 0">
          <el-pagination
              v-model:current-page="csvCurrentPage"
              v-model:page-size="csvPageSize"
              :page-sizes="[20, 50, 100, 200]"
              :total="csvTotalCount"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="handleCSVSizeChange"
              @current-change="handleCSVPageChange"
          />
        </div>
      </div>
    </div>

    <!-- kkfile预览配置对话框 -->
    <el-dialog
        v-model="kkfileConfigVisible"
        title="kkfile预览配置"
        width="600px"
        :close-on-click-modal="false"
    >
      <el-form label-width="140px">
        <div class="config-tip">
            <el-alert
                type="info"
                :closable="false"
                show-icon
            >
              <template #default>
                <p>占位符 <code>{base64(url)}</code> 会替换为 <code>encodeURIComponent(Base64(文件URL))</code>（kkFileView 3.x）。</p>
                <p>默认（含官方演示参数 <code>key=000</code>）：<code>https://file.kkview.cn/onlinePreview?url={base64(url)}&amp;key=000</code></p>
              </template>
            </el-alert>
          </div>
        <el-form-item label="预览接口地址">
          <el-input
              v-model="kkfilePreviewURL"
              placeholder="请输入kkfile预览接口地址"
              clearable
          >
            <template #append>
              <el-button @click="resetKkfileConfig" size="small">重置默认</el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="kkfileConfigVisible = false">取消</el-button>
        <el-button type="primary" @click="saveKkfileConfig">保存</el-button>
      </template>
    </el-dialog>

    <!-- 文件预览对话框 -->
    <el-dialog
        v-model="previewDialogVisible"
        width="75%"
        :close-on-click-modal="false"
        top="5vh"
        class="preview-dialog"
    >
      <template #header>
        <div class="preview-header">
          <span>文件预览</span>
          <el-button
              v-if="currentPreviewURL"
              type="primary"
              size="small"
              @click="openPreviewURL"
          >
            在新窗口打开
          </el-button>
        </div>
      </template>
      <div class="preview-container" v-if="previewLoading">
        <el-skeleton :rows="10" animated />
      </div>
      <div class="preview-container" v-else-if="previewError">
        <el-alert
            :title="previewError"
            type="error"
            :closable="false"
            show-icon
        />
      </div>
      <div class="preview-container" v-else-if="currentPreviewURL">
        <iframe
            :src="currentPreviewURL"
            frameborder="0"
            class="preview-iframe"
            allowfullscreen
        ></iframe>
      </div>
    </el-dialog>

    <!-- 漏洞扫描功能配置 -->
    <div v-if="ossFunction === 'vuln-scan'" class="oss-function-section">
      <div class="vuln-scan-config">
        <!-- 自动填充的基础配置 -->
        <div class="config-section">
          <h5>基础配置</h5>
          <div class="config-grid">
            <div class="config-item">
              <label>云厂商：</label>
              <el-select v-model="scanConfig.cloud" placeholder="自动识别或手动选择" class="config-input">
                <el-option label="阿里云" value="aliyun"></el-option>
                <el-option label="腾讯云" value="tencent"></el-option>
                <el-option label="华为云" value="huawei"></el-option>
                <el-option label="AWS" value="aws"></el-option>
                <el-option label="谷歌云" value="gcp"></el-option>
                <el-option label="Azure" value="azure"></el-option>
              </el-select>
            </div>
            <div class="config-item">
              <label>区域：</label>
              <el-input v-model="scanConfig.region" placeholder="自动识别区域" class="config-input"></el-input>
            </div>
            <div class="config-item">
              <label>Bucket名称：</label>
              <el-input v-model="scanConfig.bucket" placeholder="自动识别存储桶名称" class="config-input"></el-input>
            </div>
            <div class="config-item" v-if="scanConfig.cloud === 'tencent'">
              <label>腾讯云APPID：</label>
              <el-input v-model="scanConfig.tencentAppid" placeholder="腾讯云需要APPID" class="config-input"></el-input>
            </div>
            <div class="config-item" v-if="scanConfig.cloud === 'azure'">
              <label>Azure账户名：</label>
              <el-input v-model="scanConfig.azureAccount" placeholder="Azure存储账户名" class="config-input"></el-input>
            </div>
            <div class="config-item">
              <label>线程数：</label>
              <el-input-number v-model="scanConfig.threads" :min="1" :max="20" class="config-input"></el-input-number>
            </div>
          </div>
        </div>

        <!-- 扫描选项 - 修改为横向排列 -->
        <div class="config-section">
          <h5>扫描选项</h5>
          <el-alert
              type="warning"
              :closable="false"
              show-icon
              class="scan-mutation-hint"
              title="涉及 PUT/POST/DELETE 的项会对桶内写入或删除由本工具生成的临时对象（固定随机文件名）；请在合法授权场景下使用，生产桶请谨慎勾选。"
          />
          <div class="scan-options-horizontal">
            <el-checkbox-group v-model="scanConfig.scanOptions">
              <div class="scan-options-row">
                <div class="scan-option-item">
                  <el-checkbox label="scan_put_upload">PUT上传漏洞</el-checkbox>
                </div>
                <div class="scan-option-item">
                  <el-checkbox label="scan_post_upload">POST上传漏洞</el-checkbox>
                </div>
                <div class="scan-option-item">
                  <el-checkbox label="scan_delete_perm">DELETE权限漏洞（默认关闭，请谨慎启用）</el-checkbox>
                </div>
                <div class="scan-option-item">
                  <el-checkbox label="scan_cors">CORS配置漏洞</el-checkbox>
                </div>
                <div class="scan-option-item">
                  <el-checkbox label="scan_logs">访问日志泄露</el-checkbox>
                </div>
              </div>
              <div class="scan-options-row">
                <div class="scan-option-item">
                  <el-checkbox label="scan_directory_traversal">目录遍历漏洞</el-checkbox>
                </div>
                <div class="scan-option-item">
                  <el-checkbox label="scan_sensitive_headers">敏感头泄露</el-checkbox>
                </div>
                <div class="scan-option-item">
                  <el-checkbox label="scan_bucket_policy">Bucket策略漏洞</el-checkbox>
                </div>
                <div class="scan-option-item">
                  <el-checkbox label="scan_kms_encryption">KMS加密配置</el-checkbox>
                </div>
              </div>
            </el-checkbox-group>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="action-buttons">
          <el-button type="primary" @click="startVulnScan" :loading="scanLoading" class="scan-btn">
            {{ scanLoading ? '扫描中...' : '开始扫描' }}
          </el-button>
          <el-button @click="resetScanConfig" class="reset-btn">重置配置</el-button>
        </div>
      </div>
    </div>

    <!-- 漏洞扫描功能配置 -->
    <div v-if="ossFunction === 'vuln-scan'" class="oss-function-section">

      <!-- 扫描结果显示部分 -->
      <div class="scan-results-section" v-if="scanResults && scanResults.length > 0">
        <div class="results-header">
          <h5>扫描结果 (共 {{ scanResults.length }} 项) &nbsp;
            <el-button type="success" @click="exportSingleResult" size="small">
              导出结果
            </el-button>
          </h5>
          <!-- 风险摘要 -->
          <div class="results-summary">
            <div class="summary-item" v-for="(count, level) in riskSummary" :key="level"
                 :class="['risk-item', getRiskClass(level)]">
              <span class="summary-label">{{ getRiskLabel(level) }}:</span>
              <span class="summary-count">{{ count }} 项</span>
            </div>
          </div>
        </div>

        <!-- 详细结果列表 -->
        <div class="results-list">
          <div v-for="(result, index) in scanResults" :key="index"
               :class="['result-item', getRiskClass(result.risk)]">
            <div class="result-header">
              <div class="risk-info">
                <span class="risk-level">{{ getRiskLabel(result.risk) }}</span>
                <span class="risk-code">{{ result.risk }}</span>
              </div>
              <span class="result-time">{{ formatTime(result.timestamp) }}</span>
            </div>
            <div class="risk-msg">{{ result.msg }}</div>
            <div class="risk-url" v-if="result.url">
              <span class="url-label">URL: </span>
              <span class="url-link" @click="openResultURL(result.url)">{{ result.url }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 无结果提示 -->
      <div class="no-results" v-else-if="scanCompleted && scanResults.length === 0">
        <el-alert
            title="未发现风险项"
            type="success"
            description="扫描完成，未发现安全风险。"
            show-icon
            :closable="false"
        />
      </div>

      <!-- 扫描中状态 -->
      <div class="scanning" v-else-if="scanLoading">
        <div class="scan-progress">
          <el-progress :percentage="scanProgress" :status="scanProgress === 100 ? 'success' : ''" />
          <p class="progress-text">正在扫描中... {{ scanProgress }}%</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import {ElMessageBox} from "element-plus";
import {Search} from "@element-plus/icons-vue";
import { DealOssList, DetectCloudProvider, StartVulnScan, ReadOssListCSV } from '../../../wailsjs/go/cloudSecurity/OssBucker';
import {BrowserOpenURL, OnFileDrop} from "../../../wailsjs/runtime";
import {GetConfigDir, OpenPath, GetOpenFilePath} from "../../../wailsjs/go/system/System";

export default {
  name: "OssBucketPanel",
  data() {
    return {
      // OSS存储桶相关数据
      ossFunction: 'vuln-scan',
      ossBucketURL: '', // 共用的URL输入
      fileListLoading: false, // 文件遍历加载状态

      // 文件遍历结果
      OssListSuccess: false,
      OssListSavePath: "",
      // CSV数据相关
      csvLoading: false,
      csvData: [],
      csvHeaders: [],
      csvSearchText: '',
      csvFileTypeFilter: '',
      csvCurrentPage: 1,
      csvPageSize: 20,
      // kkfile预览相关
      kkfileConfigVisible: false,
      kkfilePreviewURL: 'https://file.kkview.cn/onlinePreview?url={base64(url)}&key=000',
      previewDialogVisible: false,
      previewLoading: false,
      previewContent: '',
      previewError: '',
      currentPreviewURL: '',
      // 拖拽相关
      wailsDropPaths: [],

      // 漏洞扫描相关
      scanLoading: false,
      scanCompleted: false,
      scanProgress: 0,
      scanResults: [],
      detecting: false,
      detectionResult: {
        identified: false,
        cloudName: '',
        region: '',
        bucket: '',
        cloudProvider: ''
      },
      scanConfig: {
        cloud: '',
        region: '',
        bucket: '',
        tencentAppid: '',
        azureAccount: '',
        threads: 5,
        scanOptions: [
          'scan_put_upload',
          'scan_post_upload',
          'scan_cors',
          'scan_logs',
          'scan_directory_traversal',
          'scan_sensitive_headers',
          'scan_bucket_policy',
          'scan_kms_encryption'
        ]
      },
      // OSS存储桶缓存
      ossCache: {
        bucketURL: '',
        function: 'file-list',
        scanConfig: null,
        detectionResult: null
      },
      // 批量扫描相关
      batchUrls: '',
      batchScanLoading: false,
      batchScanPaused: false,
      batchScanStopped: false,
      batchScanCompletedUrls: null,
      batchProgress: 0,
      batchResults: [],
      urlCount: 0,
      completedCount: 0,
      successCount: 0,
      failedCount: 0,

      // 详细结果对话框
      detailDialogVisible: false,
      currentDetailResult: null,
      // 导出相关
      exportLoading: false,
      exportDialogVisible: false,
      exportConfig: {
        format: 'excel',
        filename: '',
        include: ['summary', 'details', 'failed'],
        dateRange: []
      },
    }
  },
  watch:{
    ossBucketURL() {
      this.saveOssCache();
    },
    scanConfig: {
      handler() {
        this.saveOssCache();
      },
      deep: true
    },
    csvSearchText() {
      // 搜索时重置到第一页
      this.csvCurrentPage = 1;
    },
    csvFileTypeFilter() {
      // 文件类型过滤时重置到第一页
      this.csvCurrentPage = 1;
    }
  },
  mounted() {
    // 恢复OSS缓存
    this.restoreOssCache();
    // 设置文件拖拽监听
    this.setupCSVFileDropListener();
  },
  beforeUnmount() {
    // 清理所有轮询
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
    }
  },
  computed: {
    // 获取filetype列的索引
    csvFileTypeIndex() {
      return this.csvHeaders.findIndex(header => 
        header && (header.toLowerCase() === 'filetype' || header.toLowerCase() === 'file_type' || header.toLowerCase() === 'type')
      );
    },
    
    // 获取所有唯一的文件类型
    csvFileTypes() {
      if (this.csvFileTypeIndex === -1 || !this.csvData.length) {
        return [];
      }
      const fileTypes = new Set();
      this.csvData.forEach(row => {
        const fileType = row[`col_${this.csvFileTypeIndex}`];
        if (fileType && String(fileType).trim()) {
          fileTypes.add(String(fileType).trim());
        }
      });
      return Array.from(fileTypes).sort();
    },
    
    // CSV过滤后的数据（包含分页）
    filteredCSVData() {
      let data = this.csvData;
      
      // 文件类型过滤
      if (this.csvFileTypeFilter && this.csvFileTypeIndex !== -1) {
        data = data.filter(row => {
          const fileType = row[`col_${this.csvFileTypeIndex}`];
          return fileType && String(fileType).trim() === this.csvFileTypeFilter;
        });
      }
      
      // 搜索过滤
      if (this.csvSearchText) {
        const searchText = this.csvSearchText.toLowerCase();
        data = data.filter(row => {
          return Object.values(row).some(value => 
            String(value).toLowerCase().includes(searchText)
          );
        });
      }
      
      // 分页处理
      const start = (this.csvCurrentPage - 1) * this.csvPageSize;
      const end = start + this.csvPageSize;
      return data.slice(start, end);
    },
    
    // CSV总数据量（用于分页器）
    csvTotalCount() {
      let data = this.csvData;
      
      // 文件类型过滤
      if (this.csvFileTypeFilter && this.csvFileTypeIndex !== -1) {
        data = data.filter(row => {
          const fileType = row[`col_${this.csvFileTypeIndex}`];
          return fileType && String(fileType).trim() === this.csvFileTypeFilter;
        });
      }
      
      // 搜索过滤
      if (this.csvSearchText) {
        const searchText = this.csvSearchText.toLowerCase();
        data = data.filter(row => {
          return Object.values(row).some(value => 
            String(value).toLowerCase().includes(searchText)
          );
        });
      }
      
      return data.length;
    },
    
    // 动态计算表格最大高度
    csvTableHeight() {
      // 设置最大高度，表格会根据内容自适应，不超过这个高度
      // 如果数据少，表格会自适应显示所有内容，不会出现滚动条
      // 如果数据多，超过最大高度时才会显示滚动条
      return 600;
    },
    // OSS存储桶
    riskSummary() {
      const summary = {
        CRITICAL: 0,
        HIGH: 0,
        MEDIUM: 0,
        LOW: 0,
        ERROR: 0,
        UNKNOWN: 0 // 添加未知类型
      };

      if (!this.scanResults || !Array.isArray(this.scanResults)) {
        return summary;
      }

      this.scanResults.forEach(result => {
        if (!result || !result.risk) {
          summary.UNKNOWN++;
          return;
        }

        const riskStr = String(result.risk).toUpperCase();

        // 更灵活的风险等级匹配
        if (riskStr.includes('CRITICAL')) {
          summary.CRITICAL++;
        } else if (riskStr.includes('HIGH')) {
          summary.HIGH++;
        } else if (riskStr.includes('MEDIUM')) {
          summary.MEDIUM++;
        } else if (riskStr.includes('LOW')) {
          summary.LOW++;
        } else if (riskStr.includes('ERROR')) {
          summary.ERROR++;
        } else {
          summary.UNKNOWN++;
          console.warn('未知风险等级:', result.risk, '完整结果:', result);
        }
      });

      return summary;
    },
    // 导出数据统计
    exportDataCount() {
      let count = 0;
      this.batchResults.forEach(result => {
        if (this.shouldIncludeResult(result)) {
          if (result.results && result.results.length > 0) {
            count += result.results.length;
          } else {
            count += 1; // 失败记录算一条
          }
        }
      });
      return count;
    },

    exportSuccessCount() {
      return this.batchResults.filter(result =>
          result.status === '成功' && this.shouldIncludeResult(result)
      ).length;
    },

    exportFailedCount() {
      return this.batchResults.filter(result =>
          result.status === '失败' && this.shouldIncludeResult(result)
      ).length;
    },

    exportRiskCount() {
      let count = 0;
      this.batchResults.forEach(result => {
        if (result.status === '成功' && this.shouldIncludeResult(result)) {
          count += result.results?.length || 0;
        }
      });
      return count;
    },

    batchRiskSummary() {
      const summary = {
        CRITICAL: 0,
        HIGH: 0,
        MEDIUM: 0,
        LOW: 0,
        ERROR: 0,
        UNKNOWN: 0
      };

      this.batchResults.forEach(result => {
        if (result.results && Array.isArray(result.results)) {
          result.results.forEach(item => {
            const riskStr = String(item.risk).toUpperCase();
            if (riskStr.includes('CRITICAL')) summary.CRITICAL++;
            else if (riskStr.includes('HIGH')) summary.HIGH++;
            else if (riskStr.includes('MEDIUM')) summary.MEDIUM++;
            else if (riskStr.includes('LOW')) summary.LOW++;
            else if (riskStr.includes('ERROR')) summary.ERROR++;
            else summary.UNKNOWN++;
          });
        }
      });

      return summary;
    }
  },
  methods: {
// 恢复OSS缓存
    restoreOssCache() {
      const cached = sessionStorage.getItem('ossStorageCache');
      if (cached) {
        try {
          const cacheData = JSON.parse(cached);
          this.ossBucketURL = cacheData.bucketURL || '';
          this.ossFunction = cacheData.function || 'file-list';
          if (cacheData.scanConfig) {
            this.scanConfig = { ...this.scanConfig, ...cacheData.scanConfig };
          }
          if (cacheData.detectionResult) {
            this.detectionResult = cacheData.detectionResult;
          }
        } catch (e) {
          console.error('恢复OSS缓存失败:', e);
        }
      }
    },

    // 保存OSS缓存
    saveOssCache() {
      const cacheData = {
        bucketURL: this.ossBucketURL,
        function: this.ossFunction,
        scanConfig: this.scanConfig,
        detectionResult: this.detectionResult
      };
      sessionStorage.setItem('ossStorageCache', JSON.stringify(cacheData));
    },

    // OSS功能切换
    onOssFunctionChange() {
      this.scanResults = [];
      this.OssListSuccess = false;
      // 切换功能时不清空URL，保持共用
    },

    // 文件遍历功能
    async generateOssListQueries() {
      if (!this.ossBucketURL.trim()) {
        this.$message.warning('请输入OSS存储桶链接！');
        return;
      }

      this.fileListLoading = true;

      try {
        ElMessageBox({
          title: "提示",
          message: "正在请求数据哦",
          type: "info",
          showConfirmButton: false,
          closeOnClickModal: false,
          closeOnPressEscape: false,
        });

        this.OssListSavePath = await DealOssList(this.ossBucketURL.trim());
        this.OssListSuccess = true;
        this.$message.success('文件生成成功：' + this.OssListSavePath);
        ElMessageBox.close();
      } catch (err) {
        ElMessageBox.close();
        this.$message.warning("请检查链接，只有存在信息泄露的链接才可以爬取哦");
      } finally {
        this.fileListLoading = false;
      }
    },

    // 漏洞扫描方法
    async startVulnScan() {
      if (!this.ossBucketURL.trim()) {
        this.$message.warning('请输入存储桶URL');
        return;
      }

      this.scanLoading = true;
      this.scanCompleted = false;
      this.scanProgress = 0;
      this.scanResults = [];

      try {
        // 构建扫描配置
        const scanConfig = {
          scanOptions: this.scanConfig.scanOptions,
          threads: this.scanConfig.threads || 5,
          timeout: 10,
          cloud: this.scanConfig.cloud,
          region: this.scanConfig.region,
          bucket: this.scanConfig.bucket
        };

        // 模拟进度更新
        const progressInterval = setInterval(() => {
          if (this.scanProgress < 90) {
            this.scanProgress += 10;
          }
        }, 500);

        // 调用后端扫描接口
        const result = await StartVulnScan(
            this.ossBucketURL.trim(),
            JSON.stringify(scanConfig)
        );

        clearInterval(progressInterval);
        this.scanProgress = 100;

        if (result && result.success) {
          let results = result.results || [];
          if (!Array.isArray(results)) {
            console.warn('扫描结果不是数组格式:', results);
            results = [];
          }

          // 为每个结果添加时间戳
          results = results.map(item => ({
            ...item,
            timestamp: new Date().toISOString()
          }));

          this.scanResults = results;
          this.$message.success(`扫描完成，发现 ${this.scanResults.length} 个风险项`);
        } else {
          const errorMsg = result?.error || '扫描失败';
          this.$message.error(`扫描失败: ${errorMsg}`);
        }
      } catch (error) {
        console.error('扫描出错:', error);
        this.$message.error(`扫描出错: ${error.message}`);
      } finally {
        this.scanLoading = false;
        this.scanCompleted = true;

        setTimeout(() => {
          this.scanProgress = 0;
        }, 1000);
      }
    },

    // 智能识别云厂商
    async autoDetectCloudProvider() {
      const url = this.ossBucketURL?.trim();
      if (!url) {
        this.$message.warning('请输入存储桶URL');
        return;
      }

      this.detecting = true;

      try {
        const result = await DetectCloudProvider(url);

        if (result.success) {
          this.detectionResult = {
            identified: true,
            cloudName: this.getCloudName(result.cloudProvider),
            region: result.region || '未知',
            bucket: result.bucket || '未知',
            cloudProvider: result.cloudProvider
          };

          // 自动填充表单
          this.scanConfig.cloud = result.cloudProvider;
          this.scanConfig.region = result.region || '';
          this.scanConfig.bucket = result.bucket || '';

          if (result.cloudProvider === 'tencent' && result.appid) {
            this.scanConfig.tencentAppid = result.appid;
          }

          if (result.cloudProvider === 'azure' && result.account) {
            this.scanConfig.azureAccount = result.account;
          }

          this.$message.success('云厂商识别成功！');
          this.saveOssCache(); // 保存缓存
        } else {
          this.$message.warning('无法自动识别云厂商，请手动选择');
        }
      } catch (error) {
        console.error('识别云厂商失败:', error);
        this.$message.error('识别失败，请检查URL格式');
      } finally {
        this.detecting = false;
      }
    },

    getCloudName(provider) {
      const names = {
        aliyun: '阿里云',
        tencent: '腾讯云',
        huawei: '华为云',
        aws: 'AWS',
        gcp: '谷歌云',
        azure: 'Azure'
      };
      return names[provider] || '未知';
    },

    // 重置配置
    resetScanConfig() {
      this.scanConfig = {
        cloud: '',
        region: '',
        bucket: '',
        tencentAppid: '',
        azureAccount: '',
        threads: 5,
        scanOptions: [
          'scan_put_upload',
          'scan_post_upload',
          'scan_cors',
          'scan_logs',
          'scan_directory_traversal',
          'scan_sensitive_headers',
          'scan_bucket_policy',
          'scan_kms_encryption'
        ]
      };
      this.detectionResult = { identified: false };
      this.scanResults = [];

      // 清除缓存
      sessionStorage.removeItem('ossStorageCache');
      this.$message.success('配置已重置');
    },

    getRiskLabel(riskLevel) {
      if (!riskLevel) return '未知';
      const level = String(riskLevel).toUpperCase();
      const labels = {
        'CRITICAL1': '严重 - 密钥泄露',
        'CRITICAL2': '严重 - 数据库泄露',
        'CRITICAL3': '严重 - 上传漏洞',
        'CRITICAL4': '严重 - 删除漏洞',
        'HIGH1': '高危 - 目录遍历',
        'HIGH2': '高危 - 日志泄露',
        'MEDIUM1': '中危 - 配置泄露',
        'MEDIUM2': '中危 - CORS漏洞',
        'MEDIUM3': '中危 - 遍历漏洞',
        'LOW1': '低危 - 文件泄露',
        'LOW2': '低危 - 访问异常',
        'LOW3': '低危 - 头信息泄露',
        'ERROR': '错误'
      };
      return labels[level] || level;
    },

    getRiskClass(riskLevel) {
      if (!riskLevel) return 'risk-unknown';
      const level = String(riskLevel).toUpperCase();
      if (level.includes('CRITICAL')) return 'risk-critical';
      if (level.includes('HIGH')) return 'risk-high';
      if (level.includes('MEDIUM')) return 'risk-medium';
      if (level.includes('LOW')) return 'risk-low';
      if (level.includes('ERROR')) return 'risk-error';
      return 'risk-unknown';
    },

    formatTime(timestamp) {
      if (!timestamp) return '';
      return new Date(timestamp).toLocaleString();
    },

    // 打开URL的函数
    openResultURL(url) {
      if (url && url.startsWith('http')) {
        BrowserOpenURL(url);
      } else {
        this.$message.warning('URL格式不正确');
      }
    },

    toggleBatchScanPause() {
      this.batchScanPaused = !this.batchScanPaused;
    },

    stopBatchScan() {
      this.batchScanStopped = true;
      this.batchScanPaused = false;
      this.$message.info('正在停止，当前进行中的任务结束后将不再排队新 URL');
    },

    // 批量扫描方法（支持暂停 / 停止；并发消费队列）
    async startBatchScan() {
      const urls = this.batchUrls.split('\n')
          .map(url => url.trim())
          .filter(url => url && url.startsWith('http'));

      if (urls.length === 0) {
        this.$message.warning('请输入有效的URL，每行一个');
        return;
      }

      if (urls.length > 50) {
        this.$message.warning('单次批量扫描最多支持50个URL');
        return;
      }

      this.batchScanLoading = true;
      this.batchScanPaused = false;
      this.batchScanStopped = false;
      this.batchScanCompletedUrls = new Set();
      this.batchResults = [];
      this.urlCount = urls.length;
      this.completedCount = 0;
      this.successCount = 0;
      this.failedCount = 0;
      this.batchProgress = 0;

      const concurrency = 3;
      let nextIndex = 0;

      const waitWhilePaused = async () => {
        while (this.batchScanPaused && !this.batchScanStopped) {
          await new Promise(r => setTimeout(r, 150));
        }
      };

      const worker = async () => {
        while (!this.batchScanStopped) {
          await waitWhilePaused();
          if (this.batchScanStopped) break;
          const i = nextIndex++;
          if (i >= urls.length) break;
          await this.scanSingleURL(urls[i]);
        }
      };

      await Promise.all(Array.from({ length: concurrency }, () => worker()));

      if (this.batchScanStopped) {
        const done = new Set(this.batchScanCompletedUrls);
        for (const u of urls) {
          if (!done.has(u)) {
            this.batchResults.push({
              url: u,
              results: [],
              status: '已取消',
              highestRisk: 'UNKNOWN',
              riskCount: 0,
              error: '用户停止扫描',
              timestamp: new Date().toISOString()
            });
            this.completedCount++;
          }
        }
        this.batchProgress = 100;
      }

      this.batchScanLoading = false;
      this.batchScanPaused = false;

      if (this.batchScanStopped) {
        const cancelled = urls.length - this.successCount - this.failedCount;
        this.$message.info(
            `已停止：成功 ${this.successCount}，失败 ${this.failedCount}，未执行 ${Math.max(0, cancelled)}`);
      } else {
        this.$message.success(`批量扫描完成！成功: ${this.successCount}, 失败: ${this.failedCount}`);
      }
    },

    // 单个URL扫描
    async scanSingleURL(url) {
      const scanConfig = {
        scanOptions: this.scanConfig.scanOptions,
        threads: this.scanConfig.threads || 5,
        timeout: 10,
        cloud: '',
        region: '',
        bucket: ''
      };

      try {
        const result = await StartVulnScan(url, JSON.stringify(scanConfig));

        let results = [];
        let highestRisk = 'UNKNOWN';
        let riskCount = 0;

        if (result && result.success) {
          results = result.results || [];
          if (!Array.isArray(results)) {
            results = [];
          }

          // 计算最高风险等级和风险数量
          if (results.length > 0) {
            riskCount = results.length;
            const riskLevels = results.map(item => {
              const riskStr = String(item.risk).toUpperCase();
              if (riskStr.includes('CRITICAL')) return 5;
              if (riskStr.includes('HIGH')) return 4;
              if (riskStr.includes('MEDIUM')) return 3;
              if (riskStr.includes('LOW')) return 2;
              if (riskStr.includes('ERROR')) return 1;
              return 0;
            });
            const maxRiskLevel = Math.max(...riskLevels);
            highestRisk = this.getRiskLevelFromScore(maxRiskLevel);
          }

          this.successCount++;
        } else {
          throw new Error(result?.error || '扫描失败');
        }

        this.batchResults.push({
          url,
          results,
          status: '成功',
          highestRisk,
          riskCount,
          timestamp: new Date().toISOString()
        });

      } catch (error) {
        console.error(`扫描URL失败: ${url}`, error);
        this.failedCount++;

        this.batchResults.push({
          url,
          results: [],
          status: '失败',
          highestRisk: 'ERROR',
          riskCount: 0,
          error: error.message,
          timestamp: new Date().toISOString()
        });
      } finally {
        this.batchScanCompletedUrls.add(url);
      }

      this.completedCount++;
      this.batchProgress = Math.round((this.completedCount / this.urlCount) * 100);
    },

    // 根据风险分数获取风险等级
    getRiskLevelFromScore(score) {
      switch (score) {
        case 5: return 'CRITICAL';
        case 4: return 'HIGH';
        case 3: return 'MEDIUM';
        case 2: return 'LOW';
        case 1: return 'ERROR';
        default: return 'UNKNOWN';
      }
    },

    // 获取状态类型
    getStatusType(status) {
      switch (status) {
        case '成功': return 'success';
        case '失败': return 'danger';
        case '已取消': return 'info';
        default: return 'info';
      }
    },

    // 查看详细结果
    viewDetailedResult(result) {
      this.currentDetailResult = result;
      this.detailDialogVisible = true;
    },

    // 清空批量URL
    clearBatchUrls() {
      this.batchUrls = '';
    },

    // 清空批量结果
    clearBatchResults() {
      this.batchResults = [];
      this.$message.info('已清空批量扫描结果');
    },

    // 打开文件夹
    async openXlsxFileDir() {
      const baseDir = await GetConfigDir()
      const fileDir = baseDir + "/file"; // 拼接file子目录
      await OpenPath(fileDir)
    },

    // 导出批量结果
    async exportBatchResults() {
      if (this.batchResults.length === 0) {
        this.$message.warning('没有可导出的结果数据');
        return;
      }

      // 设置默认文件名
      if (!this.exportConfig.filename) {
        const timestamp = new Date().toISOString().slice(0, 19).replace(/[:.]/g, '-');
        this.exportConfig.filename = `OSS扫描结果_${timestamp}`;
      }

      this.exportDialogVisible = true;
    },

    // 确认导出
    async confirmExport() {
      this.exportLoading = true;

      try {
        const exportData = this.prepareExportData();

        if (this.exportConfig.format === 'excel') {
          await this.exportToExcel(exportData);
        } else {
          await this.exportToCSV(exportData);
        }

        this.exportDialogVisible = false;
        this.$message.success('导出成功！');
      } catch (error) {
        console.error('导出失败:', error);
        this.$message.error('导出失败: ' + error.message);
      } finally {
        this.exportLoading = false;
      }
    },

    // 准备导出数据
    prepareExportData() {
      const data = {
        summary: this.getExportSummary(),
        results: [],
        timestamp: new Date().toISOString(),
        config: { ...this.exportConfig }
      };

      // 过滤要导出的结果
      this.batchResults.forEach(result => {
        if (this.shouldIncludeResult(result)) {
          if (result.status === '成功' && result.results && result.results.length > 0) {
            result.results.forEach(item => {
              data.results.push({
                url: result.url,
                status: result.status,
                scanTime: result.timestamp,
                riskLevel: item.risk,
                riskDescription: this.getRiskLabel(item.risk),
                message: item.msg,
                riskUrl: item.url || '',
                riskType: this.getRiskType(item.risk)
              });
            });
          } else {
            // 失败记录或无风险记录
            data.results.push({
              url: result.url,
              status: result.status,
              scanTime: result.timestamp,
              riskLevel: result.highestRisk,
              riskDescription: result.status === '失败' ? '扫描失败' : '无风险',
              message: result.error || '未发现安全风险',
              riskUrl: '',
              riskType: result.status === '失败' ? 'ERROR' : 'SAFE'
            });
          }
        }
      });

      return data;
    },

    // 判断是否包含该结果
    shouldIncludeResult(result) {
      // 日期过滤
      if (this.exportConfig.dateRange && this.exportConfig.dateRange.length === 2) {
        const [start, end] = this.exportConfig.dateRange;
        const scanDate = result.timestamp.split('T')[0];
        if (scanDate < start || scanDate > end) {
          return false;
        }
      }

      // 内容过滤
      if (result.status === '失败' && !this.exportConfig.include.includes('failed')) {
        return false;
      }

      return true;
    },

    // 获取导出摘要
    getExportSummary() {
      return {
        totalUrls: this.batchResults.length,
        successUrls: this.batchResults.filter(r => r.status === '成功').length,
        failedUrls: this.batchResults.filter(r => r.status === '失败').length,
        totalRisks: this.batchResults.reduce((sum, r) => sum + (r.results?.length || 0), 0),
        riskDistribution: {...this.batchRiskSummary},
        exportTime: new Date().toLocaleString('zh-CN'),
        scanConfig: {
          threads: this.scanConfig.threads,
          scanOptions: this.scanConfig.scanOptions
        }
      };
    },

    // 获取风险类型
    getRiskType(riskLevel) {
      const level = String(riskLevel).toUpperCase();
      if (level.includes('CRITICAL')) return '严重风险';
      if (level.includes('HIGH')) return '高危风险';
      if (level.includes('MEDIUM')) return '中危风险';
      if (level.includes('LOW')) return '低危风险';
      if (level.includes('ERROR')) return '错误信息';
      return '未知风险';
    },

    // 导出到Excel
    async exportToExcel(exportData) {
      const XLSX = await import('xlsx');

      // 创建工作簿
      const wb = XLSX.utils.book_new();

      // 创建摘要工作表
      const summaryData = [
        ['OSS存储桶安全扫描报告'],
        ['导出时间', exportData.summary.exportTime],
        ['扫描URL总数', exportData.summary.totalUrls],
        ['成功扫描', exportData.summary.successUrls],
        ['失败扫描', exportData.summary.failedUrls],
        ['发现风险总数', exportData.summary.totalRisks],
        [],
        ['风险分布'],
        ['严重风险', exportData.summary.riskDistribution.CRITICAL || 0],
        ['高危风险', exportData.summary.riskDistribution.HIGH || 0],
        ['中危风险', exportData.summary.riskDistribution.MEDIUM || 0],
        ['低危风险', exportData.summary.riskDistribution.LOW || 0],
        ['错误信息', exportData.summary.riskDistribution.ERROR || 0],
        [],
        ['扫描配置'],
        ['线程数', exportData.summary.scanConfig.threads],
        ['扫描选项', exportData.summary.scanConfig.scanOptions.join(', ')]
      ];

      const summaryWs = XLSX.utils.aoa_to_sheet(summaryData);
      XLSX.utils.book_append_sheet(wb, summaryWs, '扫描摘要');

      // 创建详细结果工作表
      const resultsData = [
        ['URL', '状态', '扫描时间', '风险等级', '风险描述', '风险类型', '详细信息', '风险URL']
      ];

      exportData.results.forEach(item => {
        resultsData.push([
          item.url,
          item.status,
          new Date(item.scanTime).toLocaleString('zh-CN'),
          item.riskLevel,
          item.riskDescription,
          item.riskType,
          item.message,
          item.riskUrl
        ]);
      });

      const resultsWs = XLSX.utils.aoa_to_sheet(resultsData);
      XLSX.utils.book_append_sheet(wb, resultsWs, '详细结果');

      // 生成文件并下载
      const filename = `${this.exportConfig.filename}.xlsx`;
      XLSX.writeFile(wb, filename);
    },

    // 导出到CSV
    async exportToCSV(exportData) {
      let csvContent = 'OSS存储桶安全扫描报告\n';
      csvContent += `导出时间,${exportData.summary.exportTime}\n`;
      csvContent += `扫描URL总数,${exportData.summary.totalUrls}\n`;
      csvContent += `成功扫描,${exportData.summary.successUrls}\n`;
      csvContent += `失败扫描,${exportData.summary.failedUrls}\n`;
      csvContent += `发现风险总数,${exportData.summary.totalRisks}\n\n`;

      csvContent += '风险分布\n';
      csvContent += `严重风险,${exportData.summary.riskDistribution.CRITICAL || 0}\n`;
      csvContent += `高危风险,${exportData.summary.riskDistribution.HIGH || 0}\n`;
      csvContent += `中危风险,${exportData.summary.riskDistribution.MEDIUM || 0}\n`;
      csvContent += `低危风险,${exportData.summary.riskDistribution.LOW || 0}\n`;
      csvContent += `错误信息,${exportData.summary.riskDistribution.ERROR || 0}\n\n`;

      csvContent += 'URL,状态,扫描时间,风险等级,风险描述,风险类型,详细信息,风险URL\n';

      exportData.results.forEach(item => {
        const row = [
          `"${item.url}"`,
          item.status,
          new Date(item.scanTime).toLocaleString('zh-CN'),
          item.riskLevel,
          item.riskDescription,
          item.riskType,
          `"${item.message.replace(/"/g, '""')}"`,
          item.riskUrl
        ].join(',');
        csvContent += row + '\n';
      });

      // 创建Blob并下载
      const blob = new Blob(['\uFEFF' + csvContent], {
        type: 'text/csv;charset=utf-8;'
      });
      const link = document.createElement('a');
      const url = URL.createObjectURL(blob);
      link.setAttribute('href', url);
      link.setAttribute('download', `${this.exportConfig.filename}.csv`);
      link.style.visibility = 'hidden';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    },

    // 导出单个扫描结果
    async exportSingleResult() {
      if (this.scanResults.length === 0) {
        this.$message.warning('没有可导出的扫描结果');
        return;
      }

      this.exportLoading = true;

      try {
        const XLSX = await import('xlsx');
        const wb = XLSX.utils.book_new();

        // 单个扫描的摘要
        const summaryData = [
          ['单个OSS存储桶安全扫描报告'],
          ['目标URL', this.ossBucketURL],
          ['扫描时间', new Date().toLocaleString('zh-CN')],
          ['发现风险项', this.scanResults.length],
          [],
          ['风险项详情']
        ];

        const summaryWs = XLSX.utils.aoa_to_sheet(summaryData);
        XLSX.utils.book_append_sheet(wb, summaryWs, '扫描摘要');

        // 详细结果
        const resultsData = [
          ['风险等级', '风险描述', '详细信息', '相关URL', '发现时间']
        ];

        this.scanResults.forEach(item => {
          resultsData.push([
            item.risk,
            this.getRiskLabel(item.risk),
            item.msg,
            item.url || '',
            new Date(item.timestamp).toLocaleString('zh-CN')
          ]);
        });

        const resultsWs = XLSX.utils.aoa_to_sheet(resultsData);
        XLSX.utils.book_append_sheet(wb, resultsWs, '风险详情');

        // 生成文件名并下载
        const timestamp = new Date().toISOString().slice(0, 19).replace(/[:.]/g, '-');
        const filename = `单个OSS扫描_${timestamp}.xlsx`;
        XLSX.writeFile(wb, filename);

        this.$message.success('导出成功！');
      } catch (error) {
        console.error('导出失败:', error);
        this.$message.error('导出失败: ' + error.message);
      } finally {
        this.exportLoading = false;
      }
    },

    // 重置导出配置
    resetExportConfig() {
      this.exportConfig = {
        format: 'excel',
        filename: '',
        include: ['summary', 'details', 'failed'],
        dateRange: []
      };
    },

    // 加载CSV数据
    async loadCSVData() {
      if (!this.OssListSavePath) {
        this.$message.warning('请先获取文件列表');
        return;
      }

      this.csvLoading = true;
      try {
        const result = await ReadOssListCSV(this.OssListSavePath);
        if (result.success) {
          this.csvHeaders = result.headers || [];
          const rows = result.rows || [];
          
          // 将CSV行数据转换为对象数组
          this.csvData = rows.map(row => {
            const obj = {};
            this.csvHeaders.forEach((header, index) => {
              obj[`col_${index}`] = row[index] || '';
            });
            return obj;
          });

          // 重置过滤条件
          this.csvSearchText = '';
          this.csvFileTypeFilter = '';
          this.csvCurrentPage = 1;

          this.$message.success(`成功加载 ${this.csvData.length} 条记录`);
        } else {
          this.$message.error(result.error || '加载CSV数据失败');
        }
      } catch (error) {
        console.error('加载CSV数据失败:', error);
        this.$message.error('加载CSV数据失败: ' + error.message);
      } finally {
        this.csvLoading = false;
      }
    },

    // 清空CSV数据
    clearCSVData() {
      this.csvData = [];
      this.csvHeaders = [];
      this.csvSearchText = '';
      this.csvFileTypeFilter = '';
      this.csvCurrentPage = 1;
    },

    // CSV分页处理
    handleCSVSizeChange(val) {
      this.csvPageSize = val;
      this.csvCurrentPage = 1;
    },

    handleCSVPageChange(val) {
      this.csvCurrentPage = val;
    },

    // 显示kkfile配置
    showKkfileConfig() {
      // 从localStorage恢复配置
      const savedConfig = localStorage.getItem('kkfilePreviewURL');
      if (savedConfig) {
        this.kkfilePreviewURL = savedConfig;
      }
      this.kkfileConfigVisible = true;
    },

    // 保存kkfile配置
    saveKkfileConfig() {
      if (!this.kkfilePreviewURL.includes('{base64(url)}')) {
        this.$message.warning('预览接口地址必须包含 {base64(url)} 占位符');
        return;
      }
      localStorage.setItem('kkfilePreviewURL', this.kkfilePreviewURL);
      this.kkfileConfigVisible = false;
      this.$message.success('配置已保存');
    },

    // 重置kkfile配置
    resetKkfileConfig() {
      this.kkfilePreviewURL = 'https://file.kkview.cn/onlinePreview?url={base64(url)}&key=000';
    },

    // 预览文件
    async previewFile(fileURL) {
      if (!fileURL) {
        this.$message.warning('无效的文件URL');
        return;
      }
      
      // 支持http、https和file://协议
      if (!fileURL.startsWith('http') && !fileURL.startsWith('file://') && !fileURL.startsWith('data:')) {
        this.$message.warning('无效的文件URL，仅支持http、https、file://或data:协议');
        return;
      }

      this.previewDialogVisible = true;
      this.previewLoading = true;
      this.previewContent = '';
      this.previewError = '';
      this.currentPreviewURL = '';

      try {
        // 获取配置的预览URL
        let previewURL = this.kkfilePreviewURL;
        const savedConfig = localStorage.getItem('kkfilePreviewURL');
        if (savedConfig) {
          previewURL = savedConfig;
        }

        // 直接构建预览URL，不使用后端获取内容
        // 将文件URL进行base64编码（与后端保持一致：直接对UTF-8字符串进行base64编码）
        let encodedURL;
        try {
          // 使用TextEncoder将字符串转换为UTF-8字节，然后进行base64编码
          const encoder = new TextEncoder();
          const bytes = encoder.encode(fileURL);
          // 将Uint8Array转换为字符串，然后进行base64编码
          let binaryString = '';
          for (let i = 0; i < bytes.length; i++) {
            binaryString += String.fromCharCode(bytes[i]);
          }
          encodedURL = btoa(binaryString);
        } catch (e) {
          // 如果编码失败，使用URI编码后再base64编码（兼容方案）
          encodedURL = btoa(unescape(encodeURIComponent(fileURL)));
        }
        // kkFileView 3.x：url = encodeURIComponent(Base64(文件地址))
        const urlParam = encodeURIComponent(encodedURL);
        const finalPreviewURL = previewURL.replace('{base64(url)}', urlParam);
        
        this.currentPreviewURL = finalPreviewURL;
        this.previewLoading = false;
      } catch (error) {
        console.error('预览文件失败:', error);
        this.previewError = '预览失败: ' + error.message;
        this.previewLoading = false;
      }
    },

    // 在新窗口打开预览URL
    openPreviewURL() {
      if (this.currentPreviewURL) {
        BrowserOpenURL(this.currentPreviewURL);
      }
    },


    // 通过文件路径预览（使用绝对路径）
    async previewFileByPath(filePath) {
      if (!filePath) {
        this.$message.warning('文件路径为空');
        return;
      }

      // 构建file:// URL（Windows路径需要特殊处理）
      let fileURL;
      if (filePath.startsWith('http://') || filePath.startsWith('https://')) {
        fileURL = filePath;
      } else {
        // 转换为file:// URL
        let normalizedPath = filePath.replace(/\\/g, '/');
        // Windows路径处理：C:/path/to/file.csv -> file:///C:/path/to/file.csv
        if (/^[A-Za-z]:/.test(normalizedPath)) {
          fileURL = 'file:///' + normalizedPath;
        } else {
          // Unix路径
          if (!normalizedPath.startsWith('/')) {
            normalizedPath = '/' + normalizedPath;
          }
          fileURL = 'file://' + normalizedPath;
        }
      }

      // 调用预览功能
      await this.previewFile(fileURL);
    },

    // 从文件路径加载CSV数据到表格
    async loadCSVDataFromPath(filePath) {
      if (!filePath) {
        this.$message.warning('文件路径为空');
        return;
      }

      this.csvLoading = true;
      try {
        const result = await ReadOssListCSV(filePath);
        if (result.success) {
          this.csvHeaders = result.headers || [];
          const rows = result.rows || [];
          
          // 将CSV行数据转换为对象数组
          this.csvData = rows.map(row => {
            const obj = {};
            this.csvHeaders.forEach((header, index) => {
              obj[`col_${index}`] = row[index] || '';
            });
            return obj;
          });

          // 重置分页、搜索和文件类型过滤
          this.csvCurrentPage = 1;
          this.csvSearchText = '';
          this.csvFileTypeFilter = '';

          this.$message.success(`成功加载 ${this.csvData.length} 条记录`);
        } else {
          this.$message.error(result.error || '加载CSV数据失败');
        }
      } catch (error) {
        console.error('加载CSV数据失败:', error);
        this.$message.error('加载CSV数据失败: ' + error.message);
      } finally {
        this.csvLoading = false;
      }
    },

    // 设置CSV文件拖拽监听
    setupCSVFileDropListener() {
      // Wails 全局文件拖拽（绝对路径）
      OnFileDrop((x, y, paths) => {
        if (paths && paths.length > 0) {
          this.wailsDropPaths = paths;
          const filePath = paths[0];
          // 检查是否为CSV文件
          if (filePath.toLowerCase().endsWith('.csv')) {
            // 延迟一下，确保处理完成
            this.$nextTick(() => {
              this.loadCSVDataFromPath(filePath);
            });
          }
        }
      });
    },
  }
}
</script>

<style scoped>
/* OSS存储桶功能样式 */
.oss-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
}

.oss-header-card {
  background: white;
  border-radius: 8px;
  padding: 5px 15px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.function-select {
  margin-bottom: 16px;
}

/* 公共URL输入区域 */
.oss-url-section {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.url-input-card {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.url-input-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.url-input-header h5 {
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.url-input-group {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.url-input {
  flex: 1;
}

:deep(.url-input .el-input-group__append) {
  background-color: #409eff;
  border-color: #409eff;
  color: white;
}

:deep(.url-input .el-input-group__append .el-button) {
  color: white;
  border: none;
}

.detection-result {
  margin-top: 8px;
}

/* 文件遍历结果 - 缩小版 */
.oss-result-card {
  background: white;
  border-radius: 8px;
  padding: 12px 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  border-left: 4px solid #67c23a;
  margin-bottom: 16px;
}

.result-content-compact {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.result-info-compact {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 200px;
}

.result-label-compact {
  color: #606266;
  font-weight: 500;
  font-size: 13px;
}

.result-path-compact {
  color: #67c23a;
  font-weight: 600;
  font-size: 13px;
  word-break: break-all;
}

.result-actions-compact {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.open-folder-btn {
  flex-shrink: 0;
}

/* CSV表格区域 */
.csv-table-section {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  margin-top: 16px;
  flex: 0 0 auto;
}

.csv-table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 12px;
}

.csv-table-header h5 {
  margin: 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.csv-table-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.csv-table-container {
  margin-bottom: 16px;
  position: relative;
}

/* CSV表格单元格样式 - 支持换行显示 */
:deep(.el-table) {
  .el-table__cell {
    .cell {
      white-space: normal;
      word-break: break-all;
      word-wrap: break-word;
      line-height: 1.5;
      padding-top: 8px;
      padding-bottom: 8px;
    }
  }
  
  /* 表格行高度自适应 */
  .el-table__row {
    td {
      padding-top: 8px;
      padding-bottom: 8px;
    }
  }
}

/* CSV单元格内容样式 */
.csv-cell-content {
  word-break: break-all;
  word-wrap: break-word;
  white-space: normal;
  line-height: 1.5;
}

.url-cell-clickable {
  color: #409eff;
  cursor: pointer;
  word-break: break-all;
  word-wrap: break-word;
  white-space: normal;
  text-decoration: underline;
  transition: color 0.3s;
  line-height: 1.5;
}

.url-cell-clickable:hover {
  color: #66b1ff;
}

.csv-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

/* kkfile配置提示 */
.config-tip {
  margin-top: 12px;
  margin-bottom: 12px;
}

.config-tip code {
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  color: #e6a23c;
}

/* 预览对话框 */
:deep(.preview-dialog) {
  .el-dialog__body {
    padding: 1px;
  }
}

.preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #e4e7ed;
}

/* 预览容器 */
.preview-container {
  min-height: 500px;
  width: 100%;
  display: flex;
  flex-direction: column;
}

.preview-iframe {
  width: 100%;
  height: 70vh;
  min-height: 500px;
  max-height: 800px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .result-content-compact {
    flex-direction: column;
    align-items: flex-start;
  }

  .result-actions-compact {
    width: 100%;
    justify-content: flex-start;
  }

  .csv-table-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .csv-table-actions {
    width: 100%;
  }
}

/* 漏洞扫描配置 */
.vuln-scan-config {
  background: white;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  margin-bottom: 16px;
}

.config-section {
  margin-bottom: 24px;
}

.config-section h5 {
  margin: 0 0 16px 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
  padding-bottom: 8px;
  border-bottom: 1px solid #e4e7ed;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 6px;
}

.config-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 0;
}

.config-item label {
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

.config-input {
  width: 100%;
}

.scan-options-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.scan-option-item {
  display: flex;
  align-items: center;
}

:deep(.scan-option-item .el-checkbox) {
  width: 100%;
}

.action-buttons {
  display: flex;
  gap: 12px;
  justify-content: center;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid #e4e7ed;
}

.scan-btn {
  min-width: 120px;
}

.reset-btn {
  min-width: 100px;
}

/* 扫描结果区域 */
.scan-results-section {
  background: white;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.results-header {
  margin-bottom: 20px;
}

.results-header h5 {
  margin: 0 0 16px 0;
  color: #303133;
  font-size: 16px;
  font-weight: 600;
}

.results-summary {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.summary-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
}

.summary-label {
  font-weight: 600;
}

.summary-count {
  font-weight: 700;
}

/* 扫描选项横向排列样式 */
.scan-options-horizontal {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.scan-options-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: center;
}

.scan-option-item {
  flex: 0 0 auto;
  margin-bottom: 8px;
}

:deep(.scan-option-item .el-checkbox) {
  margin-right: 0;
}

/* 响应式调整 */
@media (max-width: 1200px) {
  .scan-options-row {
    gap: 12px;
  }
}

@media (max-width: 768px) {
  .scan-options-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .scan-option-item {
    width: 100%;
  }
}

/* 风险项样式 */
.risk-critical {
  background-color: #fef0f0;
  color: #f56c6c;
  border: 1px solid #fbc4c4;
}

.risk-high {
  background-color: #fdf6ec;
  color: #e6a23c;
  border: 1px solid #f5dab1;
}

.risk-medium {
  background-color: #f4f4f5;
  color: #909399;
  border: 1px solid #d3d4d6;
}

.risk-low {
  background-color: #f0f9ff;
  color: #409eff;
  border: 1px solid #b3d8ff;
}

.risk-error {
  background-color: #fdf4ff;
  color: #c456d6;
  border: 1px solid #e8c5f0;
}

.risk-unknown {
  background-color: #f4f4f5;
  color: #909399;
  border: 1px solid #d3d4d6;
}

/* 结果列表 */
.results-list {
  max-height: 500px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.result-item {
  padding: 16px;
  border-radius: 8px;
  border-left: 4px solid;
  background: white;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
}

.result-item:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  transform: translateY(-1px);
}

.result-item.risk-critical {
  border-left-color: #f56c6c;
}

.result-item.risk-high {
  border-left-color: #e6a23c;
}

.result-item.risk-medium {
  border-left-color: #909399;
}

.result-item.risk-low {
  border-left-color: #409eff;
}

.result-item.risk-error {
  border-left-color: #c456d6;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.risk-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.risk-level {
  font-weight: 600;
  font-size: 14px;
}

.risk-code {
  font-size: 12px;
  color: #909399;
  background: #f4f4f5;
  padding: 2px 6px;
  border-radius: 4px;
}

.result-time {
  font-size: 12px;
  color: #c0c4cc;
}

.risk-msg {
  margin-bottom: 8px;
  line-height: 1.5;
  color: #606266;
}

.risk-url {
  margin-bottom: 8px;
}

.url-label {
  font-weight: 500;
  color: #909399;
}

.url-link {
  color: #409eff;
  text-decoration: none;
  word-break: break-all;
}

.url-link:hover {
  text-decoration: underline;
}

/* 无结果状态 */
.no-results {
  margin-top: 16px;
}

/* 扫描进度 */
.scanning {
  background: white;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  text-align: center;
}

.scan-progress {
  max-width: 400px;
  margin: 0 auto;
}

.progress-text {
  margin-top: 12px;
  color: #909399;
  font-size: 14px;
  font-weight: 500;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .url-input-group {
    flex-direction: column;
  }

  .config-grid {
    grid-template-columns: 1fr;
  }

  .scan-options-grid {
    grid-template-columns: 1fr;
  }

  .action-buttons {
    flex-direction: column;
  }

  .result-content {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .results-summary {
    flex-direction: column;
  }
}

@media (max-width: 480px) {
  .oss-header-card,
  .oss-url-section,
  .vuln-scan-config,
  .scan-results-section {
    padding: 16px;
  }

  .url-input-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .result-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}

/* 批量扫描样式 */
.batch-scan-section {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.batch-url-input-card {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.batch-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.scan-mutation-hint {
  margin-bottom: 12px;
}

.batch-delete-warning {
  margin-bottom: 12px;
}

.batch-input-group {
  margin-bottom: 16px;
}

.batch-textarea {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
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

/* 批量结果样式 */
.batch-results {
  margin-top: 24px;
}

.batch-result-actions {
  display: flex;
  gap: 8px;
}

.batch-summary {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.summary-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  border-radius: 8px;
  min-width: 100px;
  text-align: center;
}

.summary-card .summary-count {
  font-size: 24px;
  font-weight: bold;
  margin-bottom: 4px;
}

.summary-card .summary-label {
  font-size: 12px;
  font-weight: 500;
}

/* 批量结果列表 */
.batch-results-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 600px;
  overflow-y: auto;
}

.batch-result-item {
  padding: 16px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: white;
  transition: all 0.3s ease;
}

.batch-result-item:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transform: translateY(-1px);
}

.batch-result-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  flex-wrap: wrap;
  gap: 8px;
}

.result-url {
  flex: 1;
  min-width: 300px;
}

.url-label {
  font-weight: 600;
  color: #606266;
  margin-right: 8px;
}

.url-text {
  color: #409eff;
  cursor: pointer;
  word-break: break-all;
}

.url-text:hover {
  text-decoration: underline;
}

.result-status {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}

.result-counts {
  font-size: 12px;
  color: #909399;
}

/* 批量结果详情 */
.batch-result-details {
  margin-bottom: 12px;
}

.detail-item {
  padding: 8px 12px;
  margin-bottom: 6px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.risk-badge {
  font-size: 12px;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 600;
  min-width: 60px;
  text-align: center;
  flex-shrink: 0;
}

.risk-msg {
  flex: 1;
  font-size: 13px;
  line-height: 1.4;
}

.error-item {
  padding: 8px 12px;
  background: #fef0f0;
  border: 1px solid #fbc4c4;
  border-radius: 4px;
  color: #f56c6c;
}

.error-msg {
  font-size: 13px;
}

.more-results {
  text-align: center;
  color: #909399;
  font-size: 12px;
  padding: 4px;
  background: #f5f7fa;
  border-radius: 4px;
}

.result-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .batch-result-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .result-status {
    flex-direction: row;
    align-items: center;
    width: 100%;
    justify-content: space-between;
  }

  .batch-stats {
    flex-direction: column;
    gap: 8px;
  }

  .batch-summary {
    justify-content: center;
  }

  .summary-card {
    min-width: 80px;
    padding: 12px;
  }
}

</style>