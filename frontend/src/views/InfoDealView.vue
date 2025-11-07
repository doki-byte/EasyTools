<template>
  <el-container class="container">
    <!-- 标签栏 -->
    <el-tabs v-model="activeTab" class="tabs" :key="tabsKey">
      <el-tab-pane
          v-for="tab in visibleTabs"
          :key="tab.name"
          :label="tab.title"
          :name="tab.name"
      />
    </el-tabs>

    <!-- 内容区域 -->
    <el-main>
      <div v-if="activeTab === 'fscan-deal'" class="tab-content">
        <div class="fscan-content">
          <div class="fscan-header-card">
            <h4>选择 Fscan 结果文件</h4>
            <el-upload class="upload-demo" drag action="" :before-upload="beforeUpload" :file-list="fileList"
                       :show-file-list="false" :on-change="handleFileChange">
              <i class="el-icon-upload"></i>
              <div class="el-upload__text">
                拖拽文件到此处，或<em>点击上传</em>
              </div>
              <div class="el-upload__tip">仅支持 .txt 文件</div>
            </el-upload>
          </div>

          <!-- 操作按钮 -->
          <div class="actions" v-if="fileName">
            <el-button type="primary" @click="processFile">处理文件</el-button>
            <el-button type="success" @click="openExcelPreview" v-if="isDealFile">
              预览 Excel 文件
            </el-button>
            <p v-if="isDealFile">文件保存位置：{{ excelFilePath }}</p>
          </div>

          <!-- Excel 预览 -->
          <div v-if="sheetsData.length" class="fscan-preview-card">
            <!-- Tabs for sheet navigation -->
            <el-tabs v-model="activeSheet" class="tabs">
              <el-tab-pane v-for="(sheet, index) in sheetsData" :key="index" :label="sheet.sheetName"
                           :name="sheet.sheetName">
              </el-tab-pane>
            </el-tabs>

            <!-- 当前选中的表格 -->
            <div v-if="activeSheet">
              <table class="excel-preview-table">
                <thead>
                <tr>
                  <th v-for="(col, index) in getActiveSheetData()[0]" :key="index">{{ col }}</th>
                </tr>
                </thead>
                <tbody>
                <tr v-for="(row, rowIndex) in paginatedData.slice((currentPage-1)*pageSize, currentPage*pageSize)" :key="rowIndex">
                  <td v-for="(cell, colIndex) in row" :key="colIndex">{{ cell }}</td>
                </tr>
                </tbody>
              </table>

              <!-- 分页组件 -->
              <el-pagination
                  v-if="paginatedData.length > pageSize"
                  :current-page="currentPage"
                  :page-size="pageSize"
                  :total="paginatedData.length"
                  @current-change="handlePageChange">
              </el-pagination>
            </div>
          </div>
        </div>
      </div>

      <!-- 蓝队大批量IP封禁处置 -->
      <div v-if="activeTab === 'ip-ban-deal'" class="tab-content">
        <div class="ip-ban-content">
          <div class="ip-ban-info-card">
            <p class="info">
              请填写威胁情报 & 恶意IP列表（每行一个IP），以及选填白名单IP列表。系统将自动去重，并排除白名单内容，避免误封。
            </p>
          </div>

          <div class="ip-ban-input-card">
            <!-- 上半部分：输入框 -->
            <div class="row upper-row">
              <div class="column">
                <h4>威胁情报 & 恶意IP</h4>
                <el-input type="textarea" v-model="maliciousIPInput" placeholder="请输入威胁情报 & 恶意IP (每行一个)"
                          :rows="10" class="input-box" />
              </div>
              <div class="column">
                <h4>IP白名单</h4>
                <el-input type="textarea" v-model="whiteListIPInput" placeholder="请输入IP白名单 (每行一个)" :rows="10"
                          class="input-box" />
              </div>
            </div>

            <!-- 下半部分：结果框 -->
            <div class="row lower-row">
              <div class="column">
                <h4>去重后IP (排除白名单)</h4>
                <div class="ip-input-container">
                  <el-input type="textarea" :value="uniqueIPs.join('\n')" readonly placeholder="去重后IP (排除白名单)"
                            :rows="8" class="input-box readonly" />
                  <el-button type="success" class="ip-copy-button" @click="IpCopyToClipboard('uniqueIPs')">
                    复制
                  </el-button>
                </div>
              </div>
              <div class="column">
                <h4>重复IP</h4>
                <div class="ip-input-container">
                  <el-input type="textarea" :value="duplicateIPs.join('\n')" readonly placeholder="重复的IP"
                            :rows="8" class="input-box readonly" />
                  <el-button type="success" class="ip-copy-button" @click="IpCopyToClipboard('duplicateIPs')">
                    复制
                  </el-button>
                </div>
              </div>
            </div>

            <!-- 去重按钮 -->
            <el-button type="primary" @click="processIPs" class="process-button">
              去重
            </el-button>
          </div>
        </div>
      </div>

      <!-- OSS存储桶功能部分 -->
      <div v-if="activeTab === 'oss-list'" class="tab-content">
        <div class="oss-content">
          <!-- 功能选择 -->
          <div class="oss-header-card">
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
            <div class="url-input-card">
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
            <div class="batch-url-input-card">
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
                      :loading="batchScanLoading"
                      size="small"
                  >
                    {{ batchScanLoading ? '批量扫描中...' : '开始批量扫描' }}
                  </el-button>
                </div>
              </div>

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
                <p class="progress-text">批量扫描进度: {{ batchProgress }}% ({{ completedCount }}/{{ urlCount }})</p>
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
            <div class="oss-result-card" v-if="OssListSuccess">
              <div class="result-content">
                <div class="result-info">
                  <span class="result-label">文件保存位置：</span>
                  <span class="result-path">{{ OssListSavePath }}</span>
                </div>
                <el-button type="success" @click="openXlsxFileDir" class="open-folder-btn">
                  打开文件夹
                </el-button>
              </div>
            </div>
          </div>

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
                        <el-checkbox label="scan_delete_perm">DELETE权限漏洞</el-checkbox>
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
      </div>

      <!--wx小程序反编译-->
      <div v-if="activeTab === 'unwxapp'" class="tab-content">
        <div class="unwxapp-wrapper">
          <!-- 配置区域 -->
          <el-card shadow="hover" class="config-card">
            <template #header>
              <div class="card-header">
                <span>微信小程序反编译配置</span>
              </div>
            </template>

            <!-- 第一行：小程序路径选择 -->
            <el-row :gutter="20" class="config-row">
              <el-col :span="24">
                <el-form label-width="120px" label-position="left">
                  <el-form-item label="微信Applet路径">
                    <div class="path-selector">
                      <el-input
                          v-model="appletPath"
                          readonly
                          placeholder="请选择微信小程序存储路径"
                          class="path-input"
                      />
                      <el-button
                          type="primary"
                          @click="selectAppletPath"
                          class="select-btn"
                      >
                        选择路径
                      </el-button>
                      <el-tooltip
                          effect="dark"
                          placement="top"
                      >
                        <template #content>
                          <div style="text-align: left; line-height: 1.5;">
                            <div>• Win v3: C:\Users\用户名\Documents\WeChat Files\Applet (手工定位：PC微信->设置->文件管理)</div>
                            <div>• Win v4: C:\Users\用户名\AppData\Roaming\Tencent\xwechat\radium\Applet\packages</div>
                            <div>• Mac v3: /Users/用户名/Library/Containers/com.tencent.xinWeChat/Data/.wxapplet/packages</div>
                            <div>• Mac v4: /Users/用户名/Library/Containers/com.tencent.xinWeChat/Data/Documents/app_data/radium/Applet/packages</div>
                          </div>
                        </template>
                        <el-icon class="help-icon"><QuestionFilled /></el-icon>
                      </el-tooltip>
                    </div>
                  </el-form-item>
                </el-form>
              </el-col>
            </el-row>

            <!-- 第二行：其他所有配置 -->
            <el-row :gutter="20" class="config-row">
              <el-col :span="12">
                <el-form label-width="120px" label-position="left">
                  <el-form-item label="操作按钮">
                    <el-space>
                      <el-button
                          :type="autoDecompile ? 'success' : 'primary'"
                          @click="toggleAutoDecompile"
                          size="small"
                      >
                        {{ autoDecompile ? '停用自动反编译' : '启用自动反编译' }}
                      </el-button>

                      <el-popconfirm
                          title="确认删除"
                          description="确认删除Applet目录下的所有文件吗？(会同时删除反编译后的所有文件)"
                          @confirm="clearApplet"
                      >
                        <template #reference>
                          <el-button size="small">清空Applet目录</el-button>
                        </template>
                      </el-popconfirm>

                      <el-popconfirm
                          title="确认删除"
                          description="确认删除所有反编译后的文件吗？"
                          @confirm="clearDecompiled"
                      >
                        <template #reference>
                          <el-button size="small">清空反编译文件</el-button>
                        </template>
                      </el-popconfirm>
                    </el-space>
                  </el-form-item>
                </el-form>
              </el-col>

              <el-col :span="12">
                <el-form label-width="120px" label-position="left">
                  <el-row :gutter="10">
                    <el-col :span="12">
                      <el-form-item label="信息提取规则">
                        <el-button
                            size="small"
                            type="success"
                            @click="showRulesDialog = true"
                        >
                          编辑规则
                        </el-button>
                      </el-form-item>
                    </el-col>
                    <el-col :span="12">
                      <el-form-item label="格式化代码">
                        <el-switch v-model="wxformat" />
                      </el-form-item>
                    </el-col>
                  </el-row>
                </el-form>
              </el-col>
            </el-row>
          </el-card>

          <!-- 主内容区域 -->
          <el-row :gutter="10" class="main-content">
            <!-- 左侧小程序列表 -->
            <el-col :span="11">
              <el-card class="applet-list-panel">
                <template #header>
                  <div class="panel-header">
                    <span>
                      小程序列表
                      <el-button
                          type="primary"
                          @click="loadAppletList"
                          size="small"
                          :loading="refreshing"
                      >
                        <el-icon><Refresh /></el-icon>
                        {{ refreshing ? '刷新中...' : '刷新列表' }}
                      </el-button>
                    </span>

                    <span class="total-count">共 {{ appletList.length }} 个</span>
                  </div>
                </template>

                <div class="applet-list-container">
                  <div
                      v-for="app in appletList"
                      :key="app.AppID"
                      class="applet-item"
                  >
                    <!-- 小程序头部信息 -->
                    <div class="applet-header">
                      <div class="applet-basic">
                        <el-tag type="primary" size="small" class="appid-tag">
                          {{ app.AppID }}
                        </el-tag>
                        <el-tag type="warning" size="small" class="date-tag">
                          {{ app.UpdateDate }}
                        </el-tag>
                      </div>

                      <!-- 在小程序列表显示部分-->
                      <div class="applet-info">
                        <el-tag
                            v-if="app.Info && app.Info.nickname && app.Info.nickname !== ''"
                            type="danger"
                            size="small"
                            class="nickname-tag"
                            :title="app.Info.nickname"
                        >
                          {{ app.Info.nickname }}
                        </el-tag>
                        <el-tag
                            v-if="app.Info && app.Info.username && app.Info.username !== ''"
                            type="info"
                            size="small"
                        >
                          {{ app.Info.username }}
                        </el-tag>
                        <!-- 如果没有信息，显示AppID -->
                        <el-tag v-else type="warning" size="small">
                          {{ app.AppID }}
                        </el-tag>
                      </div>
                    </div>

                    <!-- 版本列表 -->
                    <div class="version-list">
                      <div
                          v-for="version in app.Versions"
                          :key="`${app.AppID}-${version.Number}`"
                          class="version-item"
                      >
                        <div class="version-info">
                          <el-tag
                              :type="getVersionTagType(version)"
                              class="version-tag"
                          >
                            <el-icon class="version-icon"><Files /></el-icon>
                            {{ version.Number }}
                          </el-tag>

                          <el-space class="version-actions" :size="4">
                            <!-- 反编译按钮 -->
                            <el-tooltip content="反编译">
                              <el-button
                                  :disabled="version.DecompileStatus === 'Running'"
                                  :loading="version.DecompileStatus === 'Running'"
                                  size="small"
                                  @click="decompileApplet(app, version)"
                              >
                                <template #icon>
                                  <el-icon><MagicStick /></el-icon>
                                </template>
                              </el-button>
                            </el-tooltip>

                            <!-- 提取敏感信息按钮 -->
                            <el-tooltip content="提取敏感信息">
                              <el-button
                                  :disabled="version.MatchStatus === 'Running'"
                                  :loading="version.MatchStatus === 'Running'"
                                  size="small"
                                  @click="extractSensitiveInfo(app, version)"
                              >
                                <template #icon>
                                  <el-icon><Search /></el-icon>
                                </template>
                              </el-button>
                            </el-tooltip>

                            <el-tooltip content="打开文件夹">
                              <el-button
                                  size="small"
                                  @click="openAppletFolder(app, version)"
                              >
                                <template #icon>
                                  <el-icon><FolderOpened /></el-icon>
                                </template>
                              </el-button>
                            </el-tooltip>

                            <el-tooltip content="敏感信息">
                              <el-button
                                  size="small"
                                  @click="showMatchedInfo(app, version)"
                              >
                                <template #icon>
                                  <el-icon><Document /></el-icon>
                                </template>
                              </el-button>
                            </el-tooltip>
                          </el-space>
                        </div>

                        <!-- 版本状态信息 -->
                        <div v-if="version.Message" class="version-message">
                          <el-text type="info" size="small">{{ version.Message }}</el-text>
                        </div>
                      </div>

                      <!-- 如果没有版本，显示提示 -->
                      <div v-if="app.Versions.length === 0" class="no-version">
                        <el-text type="info" size="small">暂无版本信息</el-text>
                      </div>

                    </div>
                  </div>

                  <!-- 空状态 -->
                  <div v-if="!appletList || appletList.length === 0" class="empty-state">
                    <el-empty description="暂无小程序数据" />
                  </div>
                </div>
              </el-card>
            </el-col>

            <!-- 右侧详情面板 -->
            <el-col :span="12">
              <el-card class="detail-panel">
                <template #header>
                  <div class="panel-header">
                    <span>
                      敏感信息提取结果
                      <el-button type="success" size="small" @click="MingGanInfoCopyToClipboard('matchedResult')">
                        复制
                      </el-button>
                    </span>
                    <div class="selected-info">
                      <el-tag v-if="selectedApplet" type="warning">
                        {{ selectedApplet.appid }} {{ selectedApplet.version }}
                      </el-tag>
                      <el-tag v-if="selectedApplet?.nickname" type="danger">
                        {{ selectedApplet.nickname }}
                      </el-tag>
                    </div>
                  </div>
                </template>

                <el-input
                    v-model="matchedResult"
                    type="textarea"
                    placeholder="选中版本的反编译结果和敏感信息将显示在这里"
                    :rows="25"
                    class="result-textarea"
                    resize="none"
                />
              </el-card>
            </el-col>
          </el-row>

          <!-- 规则编辑对话框 -->
          <el-dialog
              v-model="showRulesDialog"
              title="信息提取规则"
              width="700px"
              destroy-on-close
          >
            <div class="rules-dialog">
              <div class="rules-toolbar">
                <el-button type="primary" @click="saveRules" size="small">
                  保存规则
                </el-button>
                <el-button @click="showRulesDialog = false" size="small">
                  取消
                </el-button>
                <span style="color: #0a6ebd">正则测试地址: https://regex101.com/</span>
              </div>

              <el-input
                  v-model="rulesText"
                  type="textarea"
                  placeholder="每行一条正则表达式规则"
                  :rows="15"
                  class="rules-textarea"
                  resize="none"
              />
            </div>
          </el-dialog>
        </div>
      </div>

      <!--jwt密钥爆破-->
      <div v-if="activeTab === 'jwt_crack'" class="jwt-content">
        <el-row :gutter="20">
          <!-- 左侧：JWT Token 输入和解析结果 -->
          <el-col :span="12">
            <!-- JWT Token 输入卡片 -->
            <el-card class="jwt-card" shadow="hover">
              <template #header>
                <div class="card-header">
                  <span>
                    JWT Token
                    <el-button type="success" size="small" @click="copyToClipboard(jwtInput, 'Token')">
                      <el-icon><CopyDocument /></el-icon>
                    </el-button>
                    <el-button type="danger" size="small" @click="jwtInput = ''">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </span>
                  <el-tag
                      v-if="jwtInput"
                      :type="jwtverify ? 'success' : 'danger'"
                      size="small"
                  >
                    {{ getVerifyStatusText() }}
                  </el-tag>
                </div>
              </template>
              <div class="input-with-actions">
                <el-input
                    type="textarea"
                    :rows="6"
                    placeholder="请输入JWT Token..."
                    v-model="jwtInput"
                    class="jwt-input large-font"
                    resize="none"
                >
                </el-input>
              </div>
            </el-card>

            <!-- JWT 结构解析卡片 -->
            <el-card class="jwt-card" shadow="hover">
              <template #header>
                <div class="card-header">
                  <span>JWT 结构解析</span>
                </div>
              </template>

              <!-- Header -->
              <div class="jwt-section">
                <div class="section-header">
                  <h5 class="section-title">
                    Header
                      <el-button type="success" size="small" @click="copyToClipboard(jwtheader, 'Header')">
                        <el-icon><CopyDocument /></el-icon>
                      </el-button>
                      <el-button type="danger" size="small" @click="jwtheader = ''">
                        <el-icon><Delete /></el-icon>
                      </el-button>
                  </h5>
                </div>
                <el-input
                    type="textarea"
                    :rows="3"
                    placeholder="JWT Header 内容"
                    v-model="jwtheader"
                    class="section-input large-font"
                    resize="none"
                >
                </el-input>
              </div>

              <!-- Payload -->
              <div class="jwt-section">
                <div class="section-header">
                  <h5 class="section-title">
                    Payload
                    <el-button type="success" size="small" @click="copyToClipboard(jwtpayload, 'Payload')">
                      <el-icon><CopyDocument /></el-icon></el-button>
                    <el-button type="danger" size="small" @click="jwtpayload = ''">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </h5>
                </div>
                <el-input
                    type="textarea"
                    :rows="5"
                    placeholder="JWT Payload 内容（可编辑）"
                    v-model="jwtpayload"
                    class="section-input large-font editable"
                    resize="none"
                >
                </el-input>
              </div>

              <!-- Signature -->
              <div class="jwt-section">
                <div class="section-header">
                  <h5 class="section-title">
                    Signature
                    <el-button type="success" size="small" @click="copyToClipboard(jwtsignature, 'Signature')">
                      <el-icon><CopyDocument /></el-icon>
                    </el-button>
                    <el-button type="danger" size="small" @click="jwtsignature = ''">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </h5>
                </div>
                <el-input
                    type="textarea"
                    :rows="2"
                    placeholder="JWT Signature 内容"
                    v-model="jwtsignature"
                    readonly
                    class="section-input large-font"
                    resize="none"
                >
                </el-input>
              </div>
            </el-card>
          </el-col>

          <!-- 右侧：操作和配置区域 -->
          <el-col :span="12">
            <!-- JWT 操作卡片 -->
            <el-card class="jwt-card" shadow="hover">
              <template #header>
                <div class="card-header">
                  <span>JWT 操作</span>
                </div>
              </template>

              <!-- 算法选择 -->
              <div class="config-item">
                <h5 class="config-label">算法选择</h5>
                <el-select v-model="value" placeholder="请选择JWT算法" class="full-width large-font">
                  <el-option
                      v-for="item in options"
                      :key="item.value"
                      :label="item.label"
                      :value="item.value">
                  </el-option>
                </el-select>
              </div>

              <!-- 密钥输入 -->
              <div class="config-item">
                <div class="config-header">
                  <h5 class="config-label">
                    JWT 密钥
                    <el-button type="success" size="small" @click="copyToClipboard(jwtsecret, 'JWT密钥')">
                      <el-icon><CopyDocument /></el-icon>
                    </el-button>
                    <el-button type="danger" size="small" @click="jwtsecret = ''">
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </h5>
                </div>
                <el-input
                    type="textarea"
                    :rows="2"
                    placeholder="请输入JWT密钥"
                    v-model="jwtsecret"
                    class="full-width large-font"
                    resize="none"
                >
                </el-input>
              </div>

              <!-- 操作按钮 -->
              <div class="action-buttons">
                <el-button type="primary" @click="startdecode" class="action-btn">
                  <el-icon><Search /></el-icon>
                  解析 JWT
                </el-button>
                <el-button type="success" @click="startencode" class="action-btn">
                  <el-icon><Edit /></el-icon>
                  编码 JWT
                </el-button>
                <el-button type="warning" @click="formatJson" class="action-btn">
                  <el-icon><MagicStick /></el-icon>
                  格式化 JSON
                </el-button>
              </div>
            </el-card>

            <!-- JWT 破解卡片 -->
            <el-card class="jwt-card" shadow="hover">
              <template #header>
                <div class="card-header">
                  <span>JWT 密钥破解</span>
                </div>
              </template>

              <!-- 字典选择 -->
              <div class="config-item">
                <h5 class="config-label">JWT 字典</h5>
                <div class="file-selector">
                  <el-input v-model="dictPath" placeholder="请选择字典文件路径" readonly class="file-input large-font" />
                  <el-button type="primary" @click="chooseFile" class="select-btn">选择</el-button>
                  <el-button type="success" @click="useDefaultDict" class="select-btn">使用默认字典</el-button>
                </div>
              </div>

              <!-- 破解按钮和进度 -->
              <div class="crack-section">
                <el-button
                    type="warning"
                    @click="startCrack"
                    class="crack-btn"
                    :loading="cracking"
                >
                  <el-icon><Unlock /></el-icon>
                  {{ cracking ? '破解中...' : '开始破解' }}
                </el-button>

                <!-- 进度条 -->
                <div class="progress-section" v-if="progress > 0">
                  <el-progress
                      :percentage="progress"
                      :text-inside="true"
                      :stroke-width="20"
                      :status="progress === 100 ? 'success' : ''"
                  />
                  <p class="progress-text">破解进度: {{ progress }}%</p>
                </div>
              </div>
            </el-card>

            <!-- 快速操作卡片 -->
            <el-card class="jwt-card" shadow="hover">
              <template #header>
                <div class="card-header">
                  <span>快速操作</span>
                </div>
              </template>
              <div class="quick-actions">
                <el-button type="info" @click="clearAll" class="quick-btn">
                  <el-icon><Close /></el-icon>
                  清空所有
                </el-button>
                <el-button type="primary" @click="loadExample" class="quick-btn">
                  <el-icon><Collection /></el-icon>
                  加载示例
                </el-button>
              </div>
            </el-card>
          </el-col>
        </el-row>
      </div>
    </el-main>
  </el-container>
</template>

<script>
import {
  DealOssList,
  DetectCloudProvider,
  FscanResultDeal,
  GetExcelContent,
  StartVulnScan,
  UploadFile
} from "../../wailsjs/go/controller/InfoDeal";
import {
  AutoDecompile,
  ClearApplet,
  ClearDecompiled,
  Decompile,
  ExtractSensitiveInfo,
  GetAllMiniApp,
  GetAppletPath,
  GetMatchedString,
  GetWechatRules,
  InitCheck,
  SaveWechatRules,
  SelectDirectory,
  SetAppletPath
} from "../../wailsjs/go/unwxapp/UnWxapp";
import {ElMessage, ElMessageBox} from "element-plus";
import * as XLSX from "xlsx";
import {GetConfigDir, GetOs, OpenPath} from "../../wailsjs/go/system/System";
import {
  BruteForceJWT,
  ChooseJwtFile,
  DecodeJWTWithAlg,
  EncodeJWTWithAlg,
  GetDefaultDictPath
} from "../../wailsjs/go/controller/JwtCrackController";
import {BrowserOpenURL} from "../../wailsjs/runtime";
import {loadMenuOrder, moduleTabsConfig} from '@/utils/menuConfig';
import {
  Close,
  Collection,
  CopyDocument,
  Delete,
  Document,
  Edit,
  Files,
  FolderOpened,
  MagicStick,
  QuestionFilled,
  Refresh,
  Search,
  Unlock
} from '@element-plus/icons-vue'


export default {
  name: "InfoDealView",
  components: {
    Edit,
    Unlock,
    Collection,
    Close,
    Delete,
    CopyDocument,
    Search,
    Refresh,
    QuestionFilled,
    Files,
    MagicStick,
    FolderOpened,
    Document
  },
  data() {
    return {
      activeTab: "", // 初始为空，等配置加载后设置
      moduleTabs: [], // 模块标签页配置
      tabsKey: Date.now(),

      fileName: "", // 当前上传的文件名
      fileList: [], // 上传的文件列表
      excelData: [], // Excel 数据，用于渲染预览
      isDealFile: false,
      excelFilePath: "", // 后端生成的 Excel 文件路径
      sheetsData: [], // 存储所有表的数据
      activeSheet: "", // 当前显示的表
      maliciousIPInput: "",
      whiteListIPInput: "",
      uniqueIPs: [],
      duplicateIPs: [],
      currentPage: 1, // 当前页
      pageSize: 10, // 每页显示的行数
      paginatedData: [], // 分页后的数据

      // OSS存储桶相关数据
      ossFunction: 'vuln-scan',
      ossBucketURL: '', // 共用的URL输入
      fileListLoading: false, // 文件遍历加载状态

      // 文件遍历结果
      OssListSuccess: false,
      OssListSavePath: "",

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
          'scan_delete_perm',
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


      // 小程序反编译配置相关
      appletPath: "",
      autoDecompile: false,
      wxformat: true,
      rulesText: "",
      showRulesDialog: false,

      // 数据相关
      appletList: [],
      selectedApplet: null,
      matchedResult: "",

      // 状态相关
      refreshing: false,
      dataCachePath: "",
      pollInterval: null,
      pollingTasks: new Set(), // 正在轮询的任务集合

      //jwt相关
      jwtInput: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiRWFzeVRvb2xzIiwidXJsIjoiaHR0cHM6Ly9naXRodWIuY29tL2Rva2ktYnl0ZS9FYXN5VG9vbHMifQ.s0yuzN2oyhbUQl4Zrgg7vk9tvJB5hsDt0EBEfqoOeog",
      jwtheader: "",
      jwtpayload: "",
      jwtsecret: "",
      jwtsignature: "",
      jwtverify: true,
      progress: 0,
      cracking: false,
      dictPath: '选择jwt字典路径',

      // 算法选项
      options: [
        { value: 'HS256', label: 'HS256' },
        { value: 'HS384', label: 'HS384' },
        { value: 'HS512', label: 'HS512' },
        { value: 'RS256', label: 'RS256' },
        { value: 'RS384', label: 'RS384' },
        { value: 'RS512', label: 'RS512' },
        { value: 'ES256', label: 'ES256' },
        { value: 'ES384', label: 'ES384' },
        { value: 'ES512', label: 'ES512' },
        { value: 'PS256', label: 'PS256' },
        { value: 'PS384', label: 'PS384' },
        { value: 'PS512', label: 'PS512' },
        { value: 'EdDSA', label: 'EdDSA' },
        { value: 'None', label: 'None' }
      ],
      value: 'HS256',
    };
  },

  computed: {
    // 计算可见的标签页
    visibleTabs() {
      if (!this.moduleTabs || this.moduleTabs.length === 0) {
        return [];
      }

      // 先过滤可见的，再排序
      const visible = this.moduleTabs.filter(tab => tab.visible !== false);
      return visible.sort((a, b) => (a.order || 0) - (b.order || 0));
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
    }
  },

  watch: {
    // 监听 visibleTabs 变化，确保 activeTab 始终有效
    visibleTabs: {
      handler(newTabs) {
        if (newTabs.length > 0 && this.configLoaded) {
          this.ensureActiveTabValid();
        }
      },
      immediate: false
    },

    activeSheet() {
      this.paginatedData = this.getActiveSheetData(); // 更新分页数据
      this.currentPage = 1; // 每次切换表格时重置为第一页
    },

    ossBucketURL() {
      this.saveOssCache();
    },
    scanConfig: {
      handler() {
        this.saveOssCache();
      },
      deep: true
    }
  },

  async mounted() {
    // 加载小程序反编译配置
    await this.initUnwxAppData();

    // 加载标签页配置
    await this.loadTabsConfig();
    this.configLoaded = true;

    // 恢复OSS缓存
    this.restoreOssCache();

    // 监听菜单更新事件
    window.addEventListener('menu-order-updated', this.handleMenuOrderUpdated);
  },
  beforeUnmount() {
    // 清理所有轮询
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
    }
    this.pollingTasks.clear();

    // 在组件卸载时移除事件监听器
    window.removeEventListener('menu-order-updated', this.handleMenuOrderUpdated);
  },

  methods: {
    // 处理菜单顺序更新事件
    async handleMenuOrderUpdated(event) {
      // console.log('收到菜单配置更新事件:', event.detail);

      // 重新加载标签页配置
      await this.loadTabsConfig();

      // 强制重新渲染 el-tabs
      this.forceTabsRerender();
    },

    // 强制重新渲染 el-tabs 组件
    forceTabsRerender() {
      // 方法1: 使用 key 强制重新渲染
      this.tabsKey = Date.now();

      // 方法2: 临时改变 activeTab 再改回来
      const currentActiveTab = this.activeTab;
      this.activeTab = '';

      this.$nextTick(() => {
        this.activeTab = currentActiveTab;
      });
    },

    // 加载标签页配置
    async loadTabsConfig() {
      try {
        const savedData = await loadMenuOrder();
        // console.log('加载的标签页数据:', savedData);

        const savedTabsOrder = savedData.tabs || {};
        const infoDealTabs = savedTabsOrder.infoDeal || [];

        // 获取默认配置
        const defaultTabs = moduleTabsConfig.infoDeal || [];

        // 创建标签页映射
        const tabMap = {};
        defaultTabs.forEach(tab => {
          tabMap[tab.name] = {
            ...tab,
            order: tab.defaultOrder,
            visible: tab.visible
          };
        });

        // 应用保存的配置
        infoDealTabs.forEach(savedTab => {
          if (tabMap[savedTab.name]) {
            tabMap[savedTab.name].order = savedTab.order;
            tabMap[savedTab.name].visible = savedTab.visible;
          }
        });

        // 转换为数组并排序
        this.moduleTabs = Object.values(tabMap)
            .sort((a, b) => (a.order || 0) - (b.order || 0));

        console.log('处理后的标签页:', this.moduleTabs);

        // 设置默认激活的标签页
        this.setDefaultActiveTab();

      } catch (error) {
        console.error('加载标签页配置失败:', error);
        this.setDefaultTabs();
      }
    },

    // 确保当前激活的标签页有效
    ensureActiveTabValid() {
      const visibleTabs = this.visibleTabs;
      if (visibleTabs.length === 0) {
        this.activeTab = '';
        return;
      }

      // 检查当前激活的标签页是否在可见标签页中
      const currentTabExists = visibleTabs.some(tab => tab.name === this.activeTab);
      if (!currentTabExists) {
        // 如果当前激活标签不存在，设置为第一个可见标签
        this.activeTab = visibleTabs[0].name;
      }
    },

    setDefaultActiveTab() {
      const visibleTabs = this.visibleTabs;
      if (visibleTabs.length > 0) {
        // 如果当前没有激活标签或者当前激活标签不可见，则设置为第一个可见标签
        if (!this.activeTab || !visibleTabs.some(tab => tab.name === this.activeTab)) {
          this.activeTab = visibleTabs[0].name;
        }
      } else {
        // 如果没有可见标签页，设置一个默认值
        this.activeTab = '';
      }
    },

    setDefaultTabs() {
      // 直接使用默认配置
      this.moduleTabs = moduleTabsConfig.infoDeal.map(tab => ({
        ...tab,
        order: tab.defaultOrder
      }));
      this.setDefaultActiveTab();
    },


    // fscan解析
    beforeUpload(file) {
      const isTxt = file.type === "text/plain";
      if (!isTxt) {
        ElMessage.error("仅支持上传 .txt 文件！");
        return false; // 阻止上传
      }
      this.fileName = file.name; // 保存文件名
      return true; // 允许上传
    },
    async handleFileChange(file) {
      if (file.raw.type !== "text/plain") return;
      const reader = new FileReader();
      reader.onload = async (event) => {
        this.fileContent = event.target.result;

        try {
          await UploadFile(this.fileName, this.fileContent);
          ElMessage.success("文件上传成功！");
        } catch (error) {
          console.error("文件上传失败:", error);
          ElMessage.error(error.message || "文件上传失败！");
        }
      };
      reader.onerror = () => ElMessage.error("文件读取失败");
      reader.readAsText(file.raw);
    },
    async processFile() {
      try {
        const result = await FscanResultDeal(this.fileName);
        this.excelFilePath = result; // 保存路径
        this.isDealFile = true;
        ElMessage.success(`文件处理成功！生成的文件: ${result}`);
      } catch (error) {
        console.error("文件处理失败:", error);
        ElMessage.error(error.message || "处理文件失败！");
      }
    },
    async openExcelPreview() {
      if (!this.excelFilePath) {
        ElMessage.error("请先处理文件！");
        return;
      }

      try {
        const encodedContent = await GetExcelContent(this.excelFilePath);
        if (!encodedContent) {
          throw new Error("未能获取到有效的 Base64 数据！");
        }

        const binary = atob(encodedContent);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
          bytes[i] = binary.charCodeAt(i);
        }
        const workbook = XLSX.read(bytes, { type: "array" });
        this.sheetsData = workbook.SheetNames.map(sheetName => ({
          sheetName,
          data: XLSX.utils.sheet_to_json(workbook.Sheets[sheetName])
        }));
        this.activeSheet = this.sheetsData[0]?.sheetName;
        this.paginatedData = this.getActiveSheetData(); // 初始化分页数据
      } catch (error) {
        console.error("Excel 文件预览失败:", error);
        ElMessage.error(error.message || "Excel 文件预览失败！");
      }
    },
    handlePageChange(page) {
      this.currentPage = page;
    },
    getActiveSheetData() {
      const activeSheetData = this.sheetsData.find(sheet => sheet.sheetName === this.activeSheet);
      return activeSheetData ? activeSheetData.data : [];
    },

    // 蓝队封禁IP
    processIPs() {
      const maliciousIPs = this.maliciousIPInput
          .split("\n")
          .map(ip => ip.trim())
          .filter(ip => ip);
      const whiteListIPs = new Set(
          this.whiteListIPInput
              .split("\n")
              .map(ip => ip.trim())
              .filter(ip => ip)
      );

      const uniqueIPs = new Set();
      const duplicateIPs = new Set();

      maliciousIPs.forEach(ip => {
        if (whiteListIPs.has(ip)) return;
        if (uniqueIPs.has(ip)) {
          duplicateIPs.add(ip);
        } else {
          uniqueIPs.add(ip);
        }
      });

      this.uniqueIPs = Array.from(uniqueIPs);
      this.duplicateIPs = Array.from(duplicateIPs);
    },
    IpCopyToClipboard(type) {
      const list = this[type];
      const text = list.join("\n");
      navigator.clipboard.writeText(text).then(() => {
        this.$message.success("IP 列表已复制到剪贴板！");
      });
    },

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
          'scan_delete_perm',
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

    // 风险相关方法保持不变
    riskSummary() {
      const summary = {
        CRITICAL: 0,
        HIGH: 0,
        MEDIUM: 0,
        LOW: 0,
        ERROR: 0
      };

      this.scanResults.forEach(result => {
        const level = result.risk.split('_')[0];
        if (summary.hasOwnProperty(level)) {
          summary[level]++;
        }
      });

      return summary;
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

    // 批量扫描风险统计
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
    },

    // 打开URL的函数
    openResultURL(url) {
      if (url && url.startsWith('http')) {
        BrowserOpenURL(url);
      } else {
        this.$message.warning('URL格式不正确');
      }
    },

    // 批量扫描方法
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
      this.batchResults = [];
      this.urlCount = urls.length;
      this.completedCount = 0;
      this.successCount = 0;
      this.failedCount = 0;
      this.batchProgress = 0;

      // 使用Promise.all进行并发扫描，限制并发数为3
      const concurrency = 3;
      const batches = [];

      for (let i = 0; i < urls.length; i += concurrency) {
        batches.push(urls.slice(i, i + concurrency));
      }

      for (const batch of batches) {
        const promises = batch.map(url => this.scanSingleURL(url));
        await Promise.all(promises);
      }

      this.batchScanLoading = false;
      this.$message.success(`批量扫描完成！成功: ${this.successCount}, 失败: ${this.failedCount}`);
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




    // 小程序反编译
    async initUnwxAppData() {
      try {
        // 初始化时检查Node.js环境
        const hasNode = await InitCheck();
        if (!hasNode) {
          ElMessageBox.confirm(
              "检测到未安装 Node.js 环境，是否现在前往官网下载？",
              "温馨提示",
              {
                confirmButtonText: "马上前往",
                cancelButtonText: "稍后再说",
                type: "warning",
              }
          ).then(() => {
            BrowserOpenURL("https://nodejs.org/");
          });
        }

        // 加载规则
        const rules = await GetWechatRules();
        this.rulesText = rules ? rules.join("\n") : "";

        this.appletPath = await GetAppletPath();

        // 加载小程序列表
        await this.loadAppletList();
      } catch (error) {
        console.error("初始化数据失败:", error);
        ElMessage.error("初始化失败");
      }
    },

    // 修复小程序列表数据处理
    async loadAppletList() {
      try {
        this.refreshing = true;
        const oldList = JSON.parse(JSON.stringify(this.appletList)); // 深拷贝旧列表

        const data = await GetAllMiniApp();

        // 处理数据格式，确保字段名一致
        this.appletList = (data || []).map(app => {
          return {
            AppID: app.AppID || app.app_id || app.AppId || '',
            UpdateDate: app.UpdateDate || app.update_date || app.updateDate || '',
            Info: app.Info || app.info || {},
            Versions: (app.Versions || app.versions || app.Version || []).map(version => ({
              Number: version.Number || version.number || version.Version || version.version || '',
              DecompileStatus: version.DecompileStatus || version.decompileStatus || version.status || 'Waiting',
              MatchStatus: version.MatchStatus || version.matchStatus || 'Waiting',
              Message: version.Message || version.message || ''
            }))
          };
        });

        // 检查状态变化
        this.checkStatusChanges(oldList, this.appletList);

      } catch (error) {
        console.error("加载小程序列表失败:", error);
        this.$message.error("加载小程序列表失败: " + error);
        this.appletList = [];
      } finally {
        this.refreshing = false;
      }
    },

    // 检查状态变化
    checkStatusChanges(oldList, newList) {
      for (const newApp of newList) {
        const oldApp = oldList.find(app => app.AppID === newApp.AppID);
        if (!oldApp) continue;

        for (const newVersion of newApp.Versions) {
          const oldVersion = oldApp.Versions.find(v => v.Number === newVersion.Number);
          if (!oldVersion) continue;

          // 检查反编译状态变化
          if (oldVersion.DecompileStatus === 'Running' &&
              newVersion.DecompileStatus === 'Stopped') {
            this.$message.success(`小程序 ${newApp.AppID} 反编译完成`);
          }

          // 检查匹配状态变化
          if (oldVersion.MatchStatus === 'Running' &&
              newVersion.MatchStatus === 'Stopped') {
            this.$message.success(`小程序 ${newApp.AppID} 敏感信息提取完成`);
          }
        }
      }
    },

    async selectAppletPath() {
      try {
        const path = await SelectDirectory();
        if (path) {
          await SetAppletPath(path);
          this.appletPath = path;
          ElMessage.success("路径设置成功");

          // 立即刷新小程序列表
          await this.loadAppletList();
        }
      } catch (error) {
        ElMessage.error("选择路径失败");
      }
    },

    async toggleAutoDecompile() {
      try {
        await AutoDecompile(!this.autoDecompile);
        this.autoDecompile = !this.autoDecompile;
        ElMessage.success(
            this.autoDecompile ? "已启用自动反编译" : "已停用自动反编译"
        );

        // 如果启用自动反编译，立即开始轮询状态
        if (this.autoDecompile) {
          this.startAutoDecompilePolling();
        }
      } catch (error) {
        ElMessage.error("操作失败: " + error);
      }
    },

    // 自动反编译状态轮询
    startAutoDecompilePolling() {
      // 每30秒刷新一次列表，检查新任务
      setInterval(async () => {
        if (this.autoDecompile) {
          await this.loadAppletList();
        }
      }, 30000);
    },

    async clearApplet() {
      try {
        await ClearApplet();
        ElMessage.success("Applet目录已清空");
        // 立即刷新列表
        await this.loadAppletList();
      } catch (error) {
        ElMessage.error("清空失败: " + error);
      }
    },

    async clearDecompiled() {
      try {
        await ClearDecompiled();
        ElMessage.success("反编译文件已清空");
        // 立即刷新列表
        await this.loadAppletList();
      } catch (error) {
        ElMessage.error("清空失败: " + error);
      }
    },

    async saveRules() {
      try {
        const rules = this.rulesText.split('\n').filter(rule => rule.trim());
        await SaveWechatRules(rules);
        this.showRulesDialog = false;
        ElMessage.success("规则保存成功");
      } catch (error) {
        ElMessage.error("保存规则失败: " + error);
      }
    },

    // 开始轮询任务状态
    startPollingTask(appID, versionNumber) {
      const taskKey = `${appID}-${versionNumber}`;

      // 如果已经在轮询，则跳过
      if (this.pollingTasks.has(taskKey)) {
        return;
      }

      this.pollingTasks.add(taskKey);

      // 设置轮询间隔
      const pollInterval = setInterval(async () => {
        try {
          await this.loadAppletList(); // 刷新列表

          // 检查任务是否完成
          const task = this.findTask(appID, versionNumber);
          if (task &&
              task.DecompileStatus !== 'Running' &&
              task.MatchStatus !== 'Running') {
            // 任务完成，停止轮询
            clearInterval(pollInterval);
            this.pollingTasks.delete(taskKey);
            console.log(`停止轮询任务: ${taskKey}`);
          }
        } catch (error) {
          console.error('轮询任务状态失败:', error);
        }
      }, 3000); // 每3秒轮询一次

      // 10分钟后自动停止轮询（安全机制）
      setTimeout(() => {
        if (this.pollingTasks.has(taskKey)) {
          clearInterval(pollInterval);
          this.pollingTasks.delete(taskKey);
          console.log(`轮询超时，停止任务: ${taskKey}`);
        }
      }, 10 * 60 * 1000);
    },

    // 查找特定任务
    findTask(appID, versionNumber) {
      for (const app of this.appletList) {
        if (app.AppID === appID) {
          const version = app.Versions.find(v => v.Number === versionNumber);
          if (version) return version;
        }
      }
      return null;
    },

    // 修改 decompileApplet 方法中的数据结构
    async decompileApplet(app, version) {
      try {
        console.log("开始反编译:", app.AppID, version.Number);

        // 检查必要参数
        if (!app.AppID || !version.Number) {
          this.$message.error("小程序ID或版本号不能为空");
          return;
        }

        // 检查是否设置了小程序路径
        if (!this.appletPath) {
          this.$message.error("请先设置微信小程序路径");
          return;
        }

        // 更新前端状态
        version.DecompileStatus = 'Running';
        version.Message = '反编译中...';

        const item = {
          AppID: app.AppID,
          UpdateDate: app.UpdateDate || '',
          Info: app.Info || {},
          Versions: [{
            Number: version.Number,
            DecompileStatus: 'Running',
            MatchStatus: version.MatchStatus || 'Waiting',
            Message: '反编译中...'
          }]
        };

        console.log("调用 Decompile 方法进行反编译:", JSON.stringify(item, null, 2));

        // 调用 Decompile 方法
        await Decompile(item);

        this.$message.success("开始反编译，请稍后查看结果");

        // 启动轮询
        this.startPollingTask(app.AppID, version.Number);

      } catch (error) {
        console.error("反编译失败:", error);
        if (version) {
          version.DecompileStatus = 'Error';
          version.Message = '反编译失败: ' + (error.message || "未知错误");
        }
        this.$message.error("反编译失败: " + error.message);
      }
    },

    async extractSensitiveInfo(app, version) {
      try {
        console.log("开始提取敏感信息:", app.AppID, version.Number);

        // 检查必要参数
        if (!app.AppID || !version.Number) {
          this.$message.error("小程序ID或版本号不能为空");
          return;
        }

        // 更新前端状态
        version.MatchStatus = 'Running';
        version.Message = '正在提取敏感信息...';

        const item = {
          AppID: app.AppID,
          UpdateDate: app.UpdateDate || '',
          Info: app.Info || {},
          Versions: [{
            Number: version.Number,
            DecompileStatus: version.DecompileStatus || 'Waiting',
            MatchStatus: 'Running',
            Message: '正在提取敏感信息...'
          }]
        };

        console.log("调用 ExtractSensitiveInfo 进行敏感信息提取:", item);

        // 调用提取方法
        await ExtractSensitiveInfo(app.AppID, version.Number);

        this.$message.success("开始提取敏感信息，请稍后查看结果");

        // 启动轮询
        this.startPollingTask(app.AppID, version.Number);

      } catch (error) {
        console.error("提取敏感信息失败:", error);
        if (version) {
          version.MatchStatus = 'Error';
          version.Message = '提取失败: ' + (error.message || "未知错误");
        }
        this.$message.error("提取敏感信息失败: " + error.message);
      }
    },

    // 添加打开文件夹功能
    async openAppletFolder(app, version) {
      try {
        if (!app.AppID || !version.Number) {
          this.$message.error("小程序ID或版本号不能为空");
          return;
        }

        if (!this.appletPath) {
          this.$message.error("请先设置微信小程序路径");
          return;
        }
        let OS = await GetOs();
        let outputPath;
        let packagePath;

        if (OS === "windows"){
            // 构建包路径
            packagePath = `${this.appletPath}\\${app.AppID}\\${version.Number}`;
            outputPath = `${packagePath}`;
        } else{
          // 构建包路径
          packagePath = `${this.appletPath}/${app.AppID}/${version.Number}`;
          outputPath = `${packagePath}`;
        }

        // console.log("打开文件夹:", outputPath);

        // 调用打开路径的方法
        await OpenPath(outputPath);

      } catch (error) {
        console.error("打开文件夹失败:", error);
        this.$message.error("打开文件夹失败: " + (error.message || "未知错误"));
      }
    },

    // 在显示敏感信息的方法中，可以添加格式处理
    async showMatchedInfo(app, version) {
      try {
        // 检查必要参数
        if (!app.AppID || !version.Number) {
          this.$message.error("小程序ID或版本号不能为空");
          return;
        }

        this.selectedApplet = {
          appid: app.AppID,
          version: version.Number,
          nickname: (app.Info && app.Info.nickname) || (app.Info && app.Info.Nickname) || ""
        };

        console.log("获取匹配信息:", app.AppID, version.Number);

        const result = await GetMatchedString(app.AppID, version.Number);

        if (result && result.length > 0) {
          // 如果有统计信息，可以特殊显示
          this.matchedResult = result.map(line => {
            if (line.includes(" -> ")) {
              const [file, content] = line.split(" -> ");
              return `📄 ${file}\n   🔍 ${content}`;
            }
            return line;
          }).join("\n\n");
        } else {
          this.matchedResult = "未找到匹配的敏感信息";
        }
      } catch (error) {
        console.error("获取敏感信息失败:", error);
        ElMessage.error("获取敏感信息失败: " + error);
        this.matchedResult = "获取敏感信息失败: " + error;
      }
    },

    MingGanInfoCopyToClipboard(type) {
      const text = this[type];
      navigator.clipboard.writeText(text).then(() => {
        this.$message.success("敏感信息提取结果已复制到剪贴板！");
      });
    },


    getVersionTagType(version) {
      if (!version) return 'info';

      const status = version.DecompileStatus || 'Waiting';
      switch (status) {
        case 'Stopped':
          return 'success';
        case 'Running':
          return 'warning';
        case 'Error':
          return 'danger';
        default:
          return 'info';
      }
    },


    // jwt解析
    async useDefaultDict() {
      try {
        const defaultPath = await GetDefaultDictPath();
        if (defaultPath) {
          this.dictPath = defaultPath;
          ElMessage.success("已使用默认字典路径");
        } else {
          ElMessage.warning("未找到默认字典文件");
        }
      } catch (error) {
        console.error("获取默认字典路径失败:", error);
        ElMessage.error("获取默认字典路径失败");
      }
    },

    // 修改 chooseFile 方法，添加自动解析
    chooseFile() {
      ChooseJwtFile()
          .then((path) => {
            if (path) {
              this.dictPath = path;
              ElMessage.success("字典文件选择成功");

              // 如果已经有JWT输入，自动开始破解
              if (this.jwtInput && this.jwtInput.trim()) {
                this.$confirm('是否立即开始JWT破解？', '提示', {
                  confirmButtonText: '开始破解',
                  cancelButtonText: '稍后',
                  type: 'info'
                }).then(() => {
                  this.startCrack();
                });
              }
            } else {
              ElMessage.error("未选择任何文件");
            }
          })
          .catch((error) => {
            console.error("选择文件失败:", error);
            ElMessage.error("选择文件失败，请重试");
          });
    },

    // 修改 startCrack 方法，添加自动使用默认字典
    async startCrack() {
      // 如果没有选择字典，尝试使用默认字典
      if (!this.dictPath || this.dictPath === '选择jwt字典路径') {
        try {
          const defaultPath = await GetDefaultDictPath();
          if (defaultPath) {
            this.dictPath = defaultPath;
            ElMessage.info("已自动使用默认字典");
          } else {
            ElMessage.warning("请先选择字典文件或确保默认字典存在");
            return;
          }
        } catch (error) {
          ElMessage.error("无法找到默认字典，请手动选择字典文件");
          return;
        }
      }

      this.cracking = true;
      this.progress = 0;

      BruteForceJWT(this.jwtInput, this.value, this.dictPath)
          .then((result) => {
            this.cracking = false;
            if (!result.error) {
              this.jwtsecret = result.secret || "";
              this.jwtsignature = result.signature || "";
              this.jwtheader = JSON.stringify(result.header, null, 2);
              this.jwtpayload = JSON.stringify(result.payload, null, 2);
              ElMessage.success("JWT 密钥破解成功");
              this.progress = 100;
              this.jwtverify = true;
              return;
            }
            this.progress = 100;
            this.jwtverify = false;
            ElMessage.error(result.error);
          })
          .catch((error) => {
            this.cracking = false;
            console.error("JWT 密钥破解失败:", error);
            ElMessage.error(error.message || "JWT 密钥破解失败");
          });
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

    // 格式化JSON
    formatJson() {
      try {
        if (this.jwtheader) {
          const headerObj = JSON.parse(this.jwtheader);
          this.jwtheader = JSON.stringify(headerObj, null, 2);
        }
        if (this.jwtpayload) {
          const payloadObj = JSON.parse(this.jwtpayload);
          this.jwtpayload = JSON.stringify(payloadObj, null, 2);
        }
        ElMessage.success('JSON格式化完成');
      } catch (error) {
        ElMessage.error('JSON格式错误，无法格式化');
      }
    },

    // 清空所有内容
    clearAll() {
      this.$confirm('确定要清空所有JWT相关内容吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        this.jwtInput = '';
        this.jwtheader = '';
        this.jwtpayload = '';
        this.jwtsecret = ''; // 保留默认密钥
        this.jwtsignature = '';
        this.dictPath = '选择jwt字典路径';
        this.progress = 0;
        ElMessage.success('已清空所有内容');
      });
    },

    // 加载示例数据
    loadExample() {
      this.jwtInput = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiRWFzeVRvb2xzIiwidXJsIjoiaHR0cHM6Ly9naXRodWIuY29tL2Rva2ktYnl0ZS9FYXN5VG9vbHMifQ.s0yuzN2oyhbUQl4Zrgg7vk9tvJB5hsDt0EBEfqoOeog';
      this.jwtheader = JSON.stringify({
        alg: "HS256",
        typ: "JWT"
      }, null, 2);
      this.jwtpayload = JSON.stringify({
        name: "EasyTools",
        url: "https://github.com/doki-byte/EasyTools"
      }, null, 2);
      this.jwtsecret = "";
      ElMessage.info('已加载示例数据');
    },

    // 编码JWT（必须先验证秘钥）
    async startencode() {
      // 检查必要条件
      if (!this.jwtInput || !this.jwtInput.trim()) {
        ElMessage.warning('请先输入要验证的JWT Token');
        return;
      }

      if (!this.jwtsecret || this.jwtsecret.trim() === '') {
        ElMessage.warning('请先设置JWT秘钥');
        return;
      }

      // 如果还没有验证过秘钥，先验证
      if (!this.jwtverify) {
        this.$confirm('尚未验证秘钥的正确性，是否先测试秘钥？', '提示', {
          confirmButtonText: '先测试秘钥',
          cancelButtonText: '直接编码',
          type: 'warning'
        }).then(() => {
          this.testSecret().then(() => {
            if (this.jwtverify) {
              this.doEncode();
            }
          });
        }).catch(() => {
          // 用户选择直接编码，但仍然给出警告
          this.$confirm('直接编码可能使用错误的秘钥，是否继续？', '警告', {
            confirmButtonText: '继续',
            cancelButtonText: '取消',
            type: 'error'
          }).then(() => {
            this.doEncode();
          });
        });
        return;
      }

      // 已经验证过秘钥，直接编码
      this.doEncode();
    },

    // 执行编码
    async doEncode() {
      let headerObj = {};
      let payloadObj = {};

      try {
        // 解析header
        if (this.jwtheader && this.jwtheader.trim()) {
          headerObj = JSON.parse(this.jwtheader);
        } else {
          ElMessage.warning('请提供有效的Header');
          return;
        }

        // 解析payload
        if (this.jwtpayload && this.jwtpayload.trim()) {
          payloadObj = JSON.parse(this.jwtpayload);
        } else {
          ElMessage.warning('请提供有效的Payload');
          return;
        }

      } catch (error) {
        ElMessage.error('Header或Payload不是有效的JSON格式');
        return;
      }

      try {
        const result = await EncodeJWTWithAlg(this.value, this.jwtsecret, payloadObj, headerObj);
        if (!result.error) {
          this.jwtInput = result.jwt_token || "";
          this.jwtsignature = result.signature || "";
          ElMessage.success("JWT编码成功");
        } else {
          if (result.error === 'token signature is invalid: signature is invalid') {
            ElMessage.error("签名无效");
          }else {
            ElMessage.error(result.error);
          }
        }
      } catch (error) {
        ElMessage.error("JWT编码失败");
      }
    },

    // 修改解析方法，默认使用空秘钥进行测试
    async startdecode() {
      if (!this.jwtInput || !this.jwtInput.trim()) {
        ElMessage.warning('请先输入JWT Token');
        return;
      }

      try {
        const testSecret = this.jwtsecret;

        const result = await DecodeJWTWithAlg(this.jwtInput, this.value, testSecret);

        if (!result.error) {
          this.jwtheader = JSON.stringify(result.header, null, 2);
          this.jwtpayload = JSON.stringify(result.payload, null, 2);
          this.jwtsignature = result.signature || "";

          // 根据验证结果设置状态
          this.jwtverify = result.valid;

          if (result.valid) {
            ElMessage.success("JWT解析成功 - 无签名或签名验证通过");
            // 如果是无签名验证通过，可能是None算法
            if (this.value === 'None' || (result.header && result.header.alg === 'none')) {
              this.jwtsecret = ''; // 清空秘钥
              ElMessage.info("检测到无签名JWT (None算法)");
            }
          } else {
            ElMessage.warning("JWT解析完成，但签名验证失败 - 需要正确秘钥");
            this.jwtsecret = ''; // 清空当前秘钥，提示用户需要输入正确秘钥
          }
        } else {
          // 解析出错，可能是格式错误
          ElMessage.error("JWT解析失败: " + result.error);
          this.jwtverify = false;
        }
      } catch (error) {
        ElMessage.error(error.message || "JWT 解析失败");
        this.jwtverify = false;
      }
    },

    // 添加获取校验状态文本的方法
    getVerifyStatusText() {
      if (!this.jwtInput) return '';

      if (this.jwtverify) {
        return this.jwtsecret && this.jwtsecret.trim() ? '校验通过' : '无签名验证';
      } else {
        return this.jwtsecret && this.jwtsecret.trim() ? '校验失败' : '需要秘钥';
      }
    },
  }
};
</script>

<style scoped>
/* 页面容器 - 修复撑满屏幕问题 */
.container {
  min-height: 96vh;
  display: flex;
  flex-direction: column;
  background-color: #f8f9fb;
  padding: 0 10px;
  box-sizing: border-box;
}

/* 顶部 Tabs 样式 */
.tabs {
  background-color: #ffffff;
  box-shadow: 0 2px 2px rgba(0, 0, 0, 0.1);
  margin-bottom: 5px;
  border-radius: 10px 10px 10px 10px;
  padding: 0 10px;
  flex-shrink: 0;
}

/* 主内容区域 */
.el-main {
  padding: 0;
  flex: 1;
  min-height: 0;
  overflow: auto;
}

:deep(.el-main) {
  padding: 0 !important;
}

/* 标签页内容通用样式 */
.tab-content {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
}

/* 通用卡片样式 */
.tab-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 200px;
}

.tab-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.tab-card-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  min-height: 0;
}

/* 通用滚动条样式 */
.tab-card-content::-webkit-scrollbar {
  width: 6px;
}

.tab-card-content::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.tab-card-content::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.tab-card-content::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

/* 文本样式 */
:deep(h4){
  display: block;
  margin-block-start: 1.33em;
  margin-block-end: 1.33em;
  margin-inline-start: 0px;
  margin-inline-end: 0px;
  font-weight: bold;
  unicode-bidi: isolate;
}

:deep(h5){
  display: block;
  font-size: 0.83em;
  margin-block-start: 1.67em;
  margin-block-end: 1.67em;
  margin-inline-start: 0;
  margin-inline-end: 0;
  font-weight: bold;
  unicode-bidi: isolate;
}

:deep(p){
  display: block;
  margin-block-start: 1em;
  margin-block-end: 1em;
  margin-inline-start: 0;
  margin-inline-end: 0;
  unicode-bidi: isolate;
}

/* 使用 flex 布局确保按钮和文件路径显示在同一行 */
.actions {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}

/* 表格样式 */
.excel-preview-table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 20px;
}

.excel-preview-table th, .excel-preview-table td {
  border: 1px solid #ccc;
  padding: 8px;
  text-align: left;
  font-size: 14px;
}

.excel-preview-table th {
  background-color: #f0f0f0;
  font-weight: bold;
}

.excel-preview-table tr:nth-child(even) {
  background-color: #f9f9f9;
}

.excel-preview-table tr:hover {
  background-color: #f1f1f1;
}

.excel-preview-table td {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 蓝队封禁IP工具 */
.tab-content .info {
  color: #409eff;
}

.row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 10px;
  flex-wrap: wrap;
}

.column {
  width: 48%;
  box-sizing: border-box;
  min-width: 300px;
  margin-bottom: 10px;
}

.upper-row .column {
  padding: 0 1px;
}

.lower-row .column {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  padding: 0 10px;
}

/* 包裹 el-input 和按钮的容器 */
.ip-input-container {
  position: relative;
}

/* 输入框 */
.input-box {
  width: 100%;
}

/* 按钮定位到输入框右上角 */
.ip-copy-button {
  position: absolute;
  top: 5px;
  right: 5px;
  z-index: 10;
  padding: 5px 10px;
  font-size: 12px;
  background-color: #00aaff;
  color: #ffffff;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.ip-copy-button:hover {
  background-color: #176bca;
}

/* 按钮优化 */
.process-button {
  display: block;
  margin: 10px auto;
  padding: 8px 16px;
  margin-top: 25px;
  border-radius: 5px;
  width: 100%;
  font-size: 18px;
  text-align: center;
  background-color: #5cb85c;
}


/* ============ OSS存储桶样式 ============ */
/* OSS存储桶功能样式 */
.oss-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
}

.oss-header-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
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

/* 文件遍历结果 */
.oss-result-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  border-left: 4px solid #67c23a;
}

.result-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.result-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.result-label {
  color: #606266;
  font-weight: 500;
}

.result-path {
  color: #67c23a;
  font-weight: 600;
}

.open-folder-btn {
  flex-shrink: 0;
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
  gap: 8px;
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

/* ============ 小程序反编译样式 ============ */
.unwxapp-wrapper {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 8px 0;
  min-height: 0;
}

.config-card {
  flex-shrink: 0;
}

.path-selector {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.path-input {
  flex: 1;
  min-width: 750px;
}

.select-btn {
  flex-shrink: 0;
}

.help-icon {
  color: #909399;
  cursor: help;
}

.rules-tip {
  margin-top: 8px;
}

/* 确保操作按钮区域对齐 */
.config-row .el-form-item {
  margin-bottom: 15px;
}

.main-content {
  flex: 1;
  display: flex;
  gap: 5px;
}

.applet-list-panel,
.detail-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 300px;
  margin-left: 5px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}

.total-count {
  color: #909399;
  font-size: 12px;
}

.selected-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

/* 小程序列表容器 - 添加高度限制和滚动 */
.applet-list-container {
  flex: 1;
  overflow-y: auto;
  padding: 5px;
  max-height: 450px;
}

.applet-item {
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: white;
  transition: all 0.3s ease;
}

.applet-item:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.applet-header {
  margin-bottom: 8px;
}

.applet-basic {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.applet-info {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.appid-tag,
.date-tag {
  font-weight: bold;
}

.nickname-tag {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.version-list {
  margin-top: 8px;
  max-height: 200px;
  overflow-y: auto;
}

.version-item {
  padding: 8px;
  border: 1px solid #f0f0f0;
  border-radius: 4px;
  margin-bottom: 6px;
  background: #fafafa;
}

.version-item:last-child {
  margin-bottom: 0;
}

.version-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
  flex-wrap: wrap;
  gap: 8px;
}

.version-tag {
  display: flex;
  align-items: center;
  gap: 4px;
}

.version-icon {
  font-size: 12px;
}

.version-actions {
  flex-shrink: 0;
}

.version-message {
  margin-top: 4px;
}

.empty-state {
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.result-textarea {
  flex: 1;
  border: none;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  min-height: 200px;
  resize: vertical;
}

.rules-dialog {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.rules-toolbar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}

.rules-textarea {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
}

/* ============ 其他界面统一应用卡片样式 ============ */

/* Fscan结果处理 */
.fscan-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
}

.fscan-header-card {
  background: white;
  border-radius: 6px;
  padding: 10px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.fscan-preview-card {
  background: white;
  border-radius: 6px;
  padding: 10px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

/* 蓝队IP封禁 */
.ip-ban-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
}

.ip-ban-info-card {
  background: white;
  border-radius: 6px;
  padding: 10px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.ip-ban-input-card {
  background: white;
  border-radius: 6px;
  padding: 5px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

/* OSS存储桶遍历 */
.oss-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  min-height: 0;
}

.oss-header-card {
  background: white;
  border-radius: 6px;
  padding: 10px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.oss-result-card {
  background: white;
  border-radius: 6px;
  padding: 20px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  flex: 1;
}

/* JWT 专用样式 */
.jwt-content {
  flex: 1;
  padding: 10px 0;
  min-height: 0;
}

.jwt-card {
  margin-bottom: 20px;
  border-radius: 8px;
  border: 1px solid #e6e8eb;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: bold;
  color: #303133;
}

.jwt-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
}

.jwt-section {
  margin-bottom: 16px;
}

.jwt-section:last-child {
  margin-bottom: 0;
}

.section-title {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: #606266;
  font-weight: 600;
}

.section-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
}

.config-item {
  margin-bottom: 16px;
}

.config-item:last-child {
  margin-bottom: 0;
}

.config-label {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: #606266;
  font-weight: 600;
}

.full-width {
  width: 100%;
}

.file-selector {
  display: flex;
  gap: 10px;
}

.file-input {
  flex: 1;
}

.select-btn {
  flex-shrink: 0;
}

.action-buttons {
  display: flex;
  gap: 12px;
  margin-top: 20px;
}

.action-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.crack-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.crack-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.progress-section {
  text-align: center;
}

.progress-text {
  margin: 8px 0 0 0;
  font-size: 12px;
  color: #909399;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .jwt-content {
    padding: 5px 0;
  }

  .action-buttons {
    flex-direction: column;
  }

  .file-selector {
    flex-direction: column;
  }
}

@media (max-width: 480px) {
  .jwt-card {
    margin-bottom: 15px;
  }
}


/* 添加大字体样式类 */
.large-font {
  font-size: 14px !important;
}

.large-font :deep(.el-textarea__inner) {
  font-size: 14px !important;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  color: #155adf;
}

.large-font :deep(.el-input__inner) {
  font-size: 14px !important;
}

/* 修改现有的JWT相关样式，确保字体大小生效 */
.jwt-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px !important;
}

.section-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px !important;
}

.full-width :deep(.el-textarea__inner) {
  font-size: 14px !important;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.file-input :deep(.el-input__inner) {
  font-size: 14px !important;
}

/* 修改文件选择器布局，适应新增按钮 */
.file-selector {
  display: flex;
  gap: 8px;
  align-items: center;
}

.file-input {
  flex: 1;
}

.select-btn {
  flex-shrink: 0;
  white-space: nowrap;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .file-selector {
    flex-direction: column;
    align-items: stretch;
  }

  .select-btn {
    width: 100%;
    margin-bottom: 8px;
  }
}

/* 响应式设计 */


@media (max-width: 768px) {
  .container {
    padding: 0 5px;
  }

  .row {
    flex-direction: column;
  }

  .column {
    width: 100%;
    min-width: unset;
    padding: 0 5px;
  }

  .main-content {
    flex-direction: column;
  }

  .applet-list-panel,
  .detail-panel {
    height: 350px;
  }

  /* 在小屏幕上调整按钮布局 */
  .actions {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .jwt-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .jwt-actions .el-button {
    margin-bottom: 8px;
  }

  .version-info {
    flex-direction: column;
    align-items: flex-start;
  }

  .panel-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .fscan-header-card,
  .ip-ban-info-card,
  .oss-header-card,
  .jwt-card {
    padding: 15px;
  }
}

@media (max-width: 480px) {
  .tabs {
    padding: 0 5px;
  }

  .tab-content {
    gap: 8px;
    padding: 4px 0;
  }

  .config-row .el-col {
    margin-bottom: 10px;
  }

  .path-selector {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }

  .path-input {
    width: 100% !important;
  }

  .form-group {
    flex-direction: column;
    align-items: stretch;
  }

  .form-group .el-input {
    margin-bottom: 10px;
  }

  .applet-basic {
    flex-direction: column;
    align-items: flex-start;
  }

  .applet-info {
    flex-direction: column;
    align-items: flex-start;
  }

  .rules-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .rules-toolbar .el-button {
    margin-bottom: 8px;
  }
}

/* 确保所有flex容器都能正确收缩 */
.flex-container {
  min-height: 0;
}

/* 修复版本列表的滚动 */
.version-list {
  margin-top: 8px;
  max-height: 150px;
  overflow-y: auto;
  flex-shrink: 0;
}

/* 修复上传区域 */
.upload-demo {
  width: 100%;
}

/* 确保文本区域可以自适应 */
.el-textarea__inner {
  resize: vertical;
  min-height: 80px;
}

/* 修复进度条布局 */
.jwt-progress {
  margin-top: 20px;
  width: 100%;
}

/* 修复操作按钮在小屏幕上的布局 */
@media (max-width: 576px) {
  .version-actions {
    flex-wrap: wrap;
    justify-content: flex-start;
    width: 100%;
  }

  .version-actions .el-button {
    margin: 2px;
    flex: 1;
    min-width: 40px;
  }

  .applet-basic {
    flex-direction: column;
    align-items: flex-start;
  }

  .applet-info {
    flex-direction: column;
    align-items: flex-start;
  }

  .ip-copy-button {
    position: relative;
    top: 0;
    right: 0;
    margin-top: 5px;
    width: 100%;
  }

  .ip-input-container {
    display: flex;
    flex-direction: column;
  }
}

/* 分页组件响应式 */
@media (max-width: 768px) {
  .el-pagination {
    justify-content: center;
  }

  .el-pagination .btn-prev,
  .el-pagination .btn-next,
  .el-pagination .number {
    margin: 0 2px;
    min-width: 32px;
  }
}

/* 表格在小屏幕上的响应式 */
@media (max-width: 768px) {
  .excel-preview-table {
    font-size: 12px;
  }

  .excel-preview-table th,
  .excel-preview-table td {
    padding: 4px 6px;
  }
}

/* 确保标签在小屏幕上不换行 */
.el-tag {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}
</style>
