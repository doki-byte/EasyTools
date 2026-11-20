<template>
  <div id="app" class="pcap-app system">
    <header class="app-header">
      <div class="header-content">
        <div class="title-section">
          <h1>PCAP数据包分析工具</h1>
        </div>
        <div class="status-bar">
          <div class="status-info">
            <span :class="['status-indicator', statusClass]"></span>
            <span :class="['status', statusClass]">{{ statusMessage }}</span>
            <span v-if="isProcessing" class="processing-stats">
              📊 已处理: {{ processingStats.packets }} 包 |
              匹配: {{ processingStats.matches }} 项
            </span>
          </div>
          <div class="stats">
            <span v-if="analysisResults.length > 0" class="stat-item">
              匹配: {{ analysisResults.length }}
            </span>
            <span v-if="extractedFiles.length > 0" class="stat-item">
              文件: {{ extractedFiles.length }}
            </span>
          </div>
          <button @click="checkEnvironment" class="btn-secondary" :disabled="isProcessing">
            <span class="btn-icon">🔍</span>
            检查环境
          </button>
          <button v-if="isProcessing" @click="cancelAnalysis" class="btn-warning">
            <span class="btn-icon">⏹️</span>
            取消
          </button>
        </div>
      </div>
    </header>

    <div class="main-layout">
      <!-- 左侧控制面板 -->
      <div class="left-panel">
        <div class="panel-section glass-card">
          <h3>文件选择</h3>
          <div class="file-selector">
            <div class="file-dropzone"
                 @click="selectFile"
                 @drop.prevent="handleFileDrop"
                 @dragover.prevent>
              <div class="dropzone-content">
                <span class="dropzone-icon">📁</span>
                <p>点击或拖拽选择PCAP文件</p>
                <small>支持 .pcap, .pcapng 格式</small>
              </div>
            </div>
            <input
                v-model="selectedFilePath"
                class="file-path-input"
                type="text"
                placeholder="可拖入/粘贴 PCAP 绝对路径，或手动输入"
                @drop.prevent="handleFilePathInputDrop"
                @dragover.prevent
            />
            <div v-if="selectedFilePath" class="file-info">
              <div class="file-info-header">
                <span class="file-icon">📄</span>
                <span class="file-name">{{ getFileName(selectedFilePath) }}</span>
              </div>
              <button @click="clearFile" class="btn-clear" title="清除文件">×</button>
            </div>
          </div>
        </div>

        <div class="panel-section glass-card">
          <h3>正则配置</h3>
          <div class="config-preview">
            <div class="config-item">
              <span class="config-label">Flag正则:</span>
              <span class="config-value" :title="currentRegexConfig.flagRegex">{{ truncateRegex(currentRegexConfig.flagRegex) }}</span>
            </div>
            <div class="config-item">
              <span class="config-label">SQL正则:</span>
              <span class="config-value" :title="currentRegexConfig.sqlRegex">{{ truncateRegex(currentRegexConfig.sqlRegex) }}</span>
            </div>
            <div class="config-item">
              <span class="config-label">凭据正则:</span>
              <span class="config-value" :title="currentRegexConfig.credentialRegex">{{ truncateRegex(currentRegexConfig.credentialRegex) }}</span>
            </div>
            <div class="config-item">
              <span class="config-label">密钥正则:</span>
              <span class="config-value" :title="currentRegexConfig.keyRegex">{{ truncateRegex(currentRegexConfig.keyRegex) }}</span>
            </div>
            <button @click="showRegexConfig" class="btn-small btn-outline config-edit-btn">
              <span class="btn-icon">✏️</span>
              编辑配置
            </button>
          </div>
        </div>

        <div class="panel-section glass-card">
          <h3>检测功能</h3>
          <div class="switch-grid">
            <div class="switch-item" v-for="option in detectionOptions" :key="option.key">
              <label class="switch">
                <input type="checkbox" v-model="option.enabled" :disabled="isProcessing" />
                <span class="slider"></span>
                <span class="switch-label">{{ option.label }}</span>
              </label>
              <span class="switch-description">{{ option.description }}</span>
            </div>
          </div>
        </div>

        <div class="panel-section glass-card">
          <h3>分析控制</h3>
          <div class="action-grid">
            <button @click="analyzePcap" :disabled="!selectedFilePath || isProcessing" class="btn-primary">
              <span class="btn-icon" v-if="isProcessing">⏳</span>
              <span class="btn-icon" v-else>🔍</span>
              {{ isProcessing ? '分析中...' : '开始分析' }}
            </button>
            <button @click="extractFilesHandler" :disabled="!selectedFilePath || isProcessing" class="btn-secondary">
              <span class="btn-icon">📂</span>
              提取文件
            </button>
            <!-- 添加正则配置按钮 -->
            <button @click="showRegexConfig" class="btn-outline" :disabled="isProcessing">
              <span class="btn-icon">⚙️</span>
              正则配置
            </button>
            <button @click="clearResults" class="btn-outline" :disabled="isProcessing">
              <span class="btn-icon">🗑️</span>
              清空结果
            </button>
          </div>
        </div>
      </div>

      <!-- 右侧结果展示 -->
      <div class="right-panel">
        <div class="results-container glass-card">
          <div class="tabs-header">
            <button
                v-for="tab in tabs"
                :key="tab.id"
                @click="activeTab = tab.id"
                :class="['tab-btn', { active: activeTab === tab.id, 'has-data': hasTabData(tab.id) }]"
            >
              <span class="tab-icon">{{ tab.icon }}</span>
              {{ tab.label }}
              <span v-if="getTabCount(tab.id) > 0" class="tab-badge">
                {{ getTabCount(tab.id) }}
              </span>
            </button>
          </div>

          <div class="tabs-content">
            <!-- 分析结果标签页 -->
            <div v-show="activeTab === 'results'" class="tab-panel">
              <div class="panel-header">
                <div class="panel-title-with-stats">
                  <h4>分析结果 ({{ analysisResults.length }})</h4>
                  <div class="type-stats">
                    <span v-for="stat in typeStats" :key="stat.type"
                          :class="['type-stat', `type-${stat.type}`]">
                      {{ stat.label }}: {{ stat.count }}
                    </span>
                  </div>
                </div>
                <div class="panel-actions">
                  <div class="search-box">
                    <input type="text" v-model="searchQuery" placeholder="搜索分析结果..."
                           @input="onSearchInput" class="search-input" />
                    <span v-if="searchQuery" class="search-clear" @click="clearSearch">×</span>
                  </div>
                  <button @click="toggleRealtimeUpdate" :class="['btn-small', { 'btn-primary': realtimeUpdate }]">
                    <span class="btn-icon">{{ realtimeUpdate ? '🔴' : '⚪' }}</span>
                    {{ realtimeUpdate ? '实时更新' : '手动更新' }}
                  </button>
                  <button @click="exportResults" class="btn-small" :disabled="analysisResults.length === 0">
                    <span class="btn-icon">📤</span>
                    导出结果
                  </button>
                  <button @click="clearResults" class="btn-small">
                    <span class="btn-icon">🗑️</span>
                    清空
                  </button>
                </div>
              </div>

              <div v-if="filteredResults.length === 0 && !isProcessing" class="empty-state">
                <div class="empty-icon">📊</div>
                <p v-if="searchQuery">没有找到匹配的搜索结果</p>
                <p v-else>暂无分析结果</p>
                <small>选择PCAP文件并开始分析以查看结果</small>
              </div>

              <div v-else class="results-list">
                <div
                    v-for="(result, index) in filteredResults"
                    :key="`${result.type}-${result.packetNum || result.fileName}-${index}`"
                    :class="['result-card', getResultClass(result.type)]"
                >
                  <div class="result-header">
                    <span class="result-type-badge">{{ getResultTypeLabel(result.type) }}</span>

                    <!-- 根据类型显示不同的信息 -->
                    <span v-if="isFileResult(result)" class="packet-info">
            文件: {{ result.fileName }} ({{ result.protocol }})
          </span>
                    <span v-else class="packet-info">数据包 #{{ result.packetNum }}</span>

                    <span class="result-time">{{ result.timestamp || getResultTime(index) }}</span>
                  </div>
                  <div class="result-content">
                    <pre>{{ result.content }}</pre>
                  </div>
                  <div class="result-actions">
                    <button @click="copyToClipboard(result.content)" class="btn-tiny" title="复制内容">
                      📋
                    </button>
                    <button @click="highlightResult(result)" class="btn-tiny" title="高亮显示">
                      🔍
                    </button>
                    <!-- 如果是文件结果，添加文件预览按钮 -->
                    <button v-if="isFileResult(result) && result.filePath"
                            @click="previewFileFromResult(result)"
                            class="btn-tiny"
                            title="预览文件">
                      👁️
                    </button>
                  </div>
                </div>

                <!-- 分析中的提示 -->
                <div v-if="isProcessing" class="analyzing-indicator">
                  <div class="loading-spinner"></div>
                  <span>分析进行中... 已找到 {{ analysisResults.length }} 个匹配项</span>
                </div>

                <!-- 搜索提示 -->
                <div v-if="searchQuery && filteredResults.length > 0" class="search-info">
                  找到 {{ filteredResults.length }} 个匹配 "{{ searchQuery }}" 的结果
                </div>
              </div>
            </div>

            <!-- 提取文件标签页 -->
            <div v-show="activeTab === 'files'" class="tab-panel">
              <div class="panel-header">
                <div class="panel-title-with-stats">
                  <h4>提取文件 ({{ extractedFiles.length }})</h4>
                  <div class="file-stats">
                    <span class="stat-item">总大小: {{ formatTotalSize }}</span>
                    <span class="stat-item">协议: {{ protocolStats }}</span>
                  </div>
                </div>
                <div class="panel-actions">
                  <div class="search-box">
                    <input type="text" v-model="fileSearchQuery" placeholder="搜索文件名、类型、协议..."
                           class="search-input" />
                    <span v-if="fileSearchQuery" class="search-clear" @click="fileSearchQuery = ''">×</span>
                  </div>
                  <div class="pagination-controls">
                    <span class="pagination-info">
                      第 {{ currentFilePage }} / {{ totalFilePages }} 页
                      (共 {{ filteredFiles.length }} 个文件)
                    </span>
                    <button @click="prevFilePage" :disabled="currentFilePage <= 1" class="btn-small" title="上一页">
                      <span class="btn-icon">◀</span>
                    </button>
                    <button @click="nextFilePage" :disabled="currentFilePage >= totalFilePages" class="btn-small" title="下一页">
                      <span class="btn-icon">▶</span>
                    </button>
                    <select v-model="filePageSize" @change="onFilePageSizeChange" class="page-size-select">
                      <option value="12">12/页</option>
                      <option value="24">24/页</option>
                      <option value="48">48/页</option>
                      <option value="96">96/页</option>
                    </select>
                  </div>
                </div>
              </div>

              <div v-if="paginatedFiles.length === 0" class="empty-state">
                <div class="empty-icon">📁</div>
                <p v-if="fileSearchQuery">没有找到匹配的文件</p>
                <p v-else>暂无提取文件</p>
                <small>启用文件提取功能并分析PCAP文件</small>
              </div>

              <!-- 列表视图 -->
              <div class="files-table-container">
                <table class="files-table">
                  <thead>
                  <tr>
                    <th class="col-name" @click="sortFiles('fileName')">
                      文件名
                      <span class="sort-indicator" v-if="sortBy === 'fileName'">
              {{ sortOrder === 'asc' ? '↑' : '↓' }}
            </span>
                    </th>
                    <th class="col-type" @click="sortFiles('fileType')">
                      类型
                      <span class="sort-indicator" v-if="sortBy === 'fileType'">
              {{ sortOrder === 'asc' ? '↑' : '↓' }}
            </span>
                    </th>
                    <th class="col-size" @click="sortFiles('size')">
                      大小
                      <span class="sort-indicator" v-if="sortBy === 'size'">
              {{ sortOrder === 'asc' ? '↑' : '↓' }}
            </span>
                    </th>
                    <th class="col-protocol" @click="sortFiles('protocol')">
                      协议
                      <span class="sort-indicator" v-if="sortBy === 'protocol'">
              {{ sortOrder === 'asc' ? '↑' : '↓' }}
            </span>
                    </th>
                    <th class="col-actions">操作</th>
                  </tr>
                  </thead>
                  <tbody>
                  <tr v-for="(file, index) in paginatedFiles" :key="`${file.filePath}-${index}`" class="file-row">
                    <td class="col-name">
                      <div class="file-name-cell">
              <span class="file-icon-small" :class="getFileIconClass(file.fileType)">
                {{ getFileIcon(file.fileType) }}
              </span>
                        <span class="file-name-text" :title="file.fileName">{{ file.fileName }}</span>
                      </div>
                    </td>
                    <td class="col-type">
                      <span class="file-type-badge">{{ file.fileType }}</span>
                    </td>
                    <td class="col-size">
                      <span class="file-size">{{ formatFileSize(file.size) }}</span>
                    </td>
                    <td class="col-protocol">
                      <span class="protocol-badge">{{ file.protocol }}</span>
                    </td>
                    <td class="col-actions">
                      <div class="table-actions">
                        <button @click="previewFile(file)" class="btn-icon-small" title="预览">
                          👁️
                        </button>
                        <button @click="copyToClipboard(file.filePath)" class="btn-icon-small" title="复制路径">
                          📋
                        </button>
                        <button @click="showFileInfo(file)" class="btn-icon-small" title="详细信息">
                          ℹ️
                        </button>
                      </div>
                    </td>
                  </tr>
                  </tbody>
                </table>
              </div>

              <!-- 文件信息模态框 -->
              <div v-if="showFileInfoModal && currentFileInfo" class="modal-overlay" @click="closeFileInfoModal">
                <div class="modal-content file-info-modal" @click.stop>
                  <div class="modal-header">
                    <div class="modal-title-section">
                      <div class="file-info-icon">
                        {{ getFileIcon(currentFileInfo.fileType) }}
                      </div>
                      <div class="modal-title">
                        <h3>文件详细信息</h3>
                        <p class="file-name-title">{{ currentFileInfo.fileName }}</p>
                      </div>
                    </div>
                    <div class="modal-actions">
                      <button @click="closeFileInfoModal" class="close-btn" title="关闭">
                        &times;
                      </button>
                    </div>
                  </div>

                  <div class="modal-body">
                    <div class="file-info-content">
                      <!-- 基本信息卡片 -->
                      <div class="info-section">
                        <h4 class="section-title">基本信息</h4>
                        <div class="info-grid">
                          <div class="info-item">
                            <span class="info-label">📄 文件名</span>
                            <div class="info-value">
                              <span class="value-text">{{ currentFileInfo.fileName }}</span>
                              <button @click="copyFileInfo('文件名', currentFileInfo.fileName)"
                                      class="btn-copy" title="复制文件名">
                                📋
                              </button>
                            </div>
                          </div>

                          <div class="info-item">
                            <span class="info-label">📊 文件类型</span>
                            <div class="info-value">
                              <span class="value-text">{{ currentFileInfo.fileType }}</span>
                              <span class="type-badge">{{ currentFileInfo.fileType }}</span>
                            </div>
                          </div>

                          <div class="info-item">
                            <span class="info-label">📏 文件大小</span>
                            <div class="info-value">
                              <span class="value-text">{{ formatFileSize(currentFileInfo.size) }}</span>
                              <span class="size-badge">{{ currentFileInfo.size }} 字节</span>
                            </div>
                          </div>

                          <div class="info-item">
                            <span class="info-label">🌐 传输协议</span>
                            <div class="info-value">
                              <span class="value-text">{{ currentFileInfo.protocol }}</span>
                              <span class="protocol-badge">{{ currentFileInfo.protocol }}</span>
                            </div>
                          </div>
                        </div>
                      </div>

                      <!-- 路径信息卡片 -->
                      <div class="info-section">
                        <h4 class="section-title">存储信息</h4>
                        <div class="info-item full-width">
                          <span class="info-label">📁 文件路径</span>
                          <div class="info-value">
              <span class="value-text path-text" :title="currentFileInfo.filePath">
                {{ currentFileInfo.filePath }}
              </span>
                            <button @click="copyFileInfo('文件路径', currentFileInfo.filePath)"
                                    class="btn-copy" title="复制文件路径">
                              📋
                            </button>
                          </div>
                        </div>
                      </div>

                      <!-- 技术信息卡片 -->
                      <div class="info-section">
                        <h4 class="section-title">技术信息</h4>
                        <div class="info-grid">
                          <div class="info-item" v-if="currentFileInfo.contentType">
                            <span class="info-label">📋 内容类型</span>
                            <div class="info-value">
                              <span class="value-text">{{ currentFileInfo.contentType }}</span>
                              <button @click="copyFileInfo('内容类型', currentFileInfo.contentType)"
                                      class="btn-copy" title="复制内容类型">
                                📋
                              </button>
                            </div>
                          </div>

                          <div class="info-item full-width" v-if="currentFileInfo.md5">
                            <span class="info-label">🔒 MD5 校验和</span>
                            <div class="info-value">
                <span class="value-text hash-text" :title="currentFileInfo.md5">
                  {{ currentFileInfo.md5 }}
                </span>
                              <button @click="copyFileInfo('MD5', currentFileInfo.md5)"
                                      class="btn-copy" title="复制MD5">
                                📋
                              </button>
                            </div>
                          </div>
                        </div>
                      </div>

                      <!-- 快速操作 -->
                      <div class="action-section">
                        <h4 class="section-title">快速操作</h4>
                        <div class="action-buttons">
                          <button @click="previewFile(currentFileInfo)" class="btn-action primary">
                            <span class="btn-icon">👁️</span>
                            预览文件内容
                          </button>
                          <button @click="copyToClipboard(currentFileInfo.filePath)" class="btn-action">
                            <span class="btn-icon">📋</span>
                            复制文件路径
                          </button>
                          <button @click="showInFolder(currentFileInfo)" class="btn-action" v-if="isElectron">
                            <span class="btn-icon">📁</span>
                            在文件夹中显示
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="modal-footer">
                    <button @click="closeFileInfoModal" class="btn-primary">
                      关闭
                    </button>
                  </div>
                </div>
              </div>

              <!-- 分页底部控件 -->
              <div v-if="paginatedFiles.length > 0" class="pagination-footer">
                <div class="pagination-stats">
                  显示 {{ startFileIndex + 1 }}-{{ endFileIndex }}，共 {{ filteredFiles.length }} 个文件
                  <span class="total-size">(总大小: {{ formatTotalSize }})</span>
                </div>
                <div class="pagination-navigation">
                  <button @click="goToFirstFilePage" :disabled="currentFilePage <= 1" class="btn-small">
                    首页
                  </button>
                  <button @click="prevFilePage" :disabled="currentFilePage <= 1" class="btn-small">
                    上一页
                  </button>
                  <span class="page-numbers">
                    <button
                        v-for="page in visiblePageNumbers"
                        :key="page"
                        @click="goToFilePage(page)"
                        :class="['page-btn', { active: page === currentFilePage }]"
                    >
                      {{ page }}
                    </button>
                    <span v-if="showPageEllipsis" class="page-ellipsis">...</span>
                  </span>
                  <button @click="nextFilePage" :disabled="currentFilePage >= totalFilePages" class="btn-small">
                    下一页
                  </button>
                  <button @click="goToLastFilePage" :disabled="currentFilePage >= totalFilePages" class="btn-small">
                    末页
                  </button>
                </div>
              </div>
            </div>

            <!-- 日志输出标签页 -->
            <div v-show="activeTab === 'logs'" class="tab-panel">
              <div class="panel-header">
                <h4>日志输出 ({{ logEntries.length }})</h4>
                <div class="panel-actions">
                  <div class="log-controls">
                    <label class="log-filter">
                      <input type="checkbox" v-model="logFilters.info" /> 信息
                    </label>
                    <label class="log-filter">
                      <input type="checkbox" v-model="logFilters.success" /> 成功
                    </label>
                    <label class="log-filter">
                      <input type="checkbox" v-model="logFilters.warning" /> 警告
                    </label>
                    <label class="log-filter">
                      <input type="checkbox" v-model="logFilters.error" /> 错误
                    </label>
                    <label class="log-filter">
                      <input type="checkbox" v-model="logFilters.match" /> 匹配
                    </label>
                  </div>
                  <button @click="clearLogs" class="btn-small">
                    <span class="btn-icon">🗑️</span>
                    清空日志
                  </button>
                  <button @click="exportLogs" class="btn-small" :disabled="!logs">
                    <span class="btn-icon">📤</span>
                    导出日志
                  </button>
                </div>
              </div>

              <div class="logs-container">
                <div class="logs-content" ref="logsContent">
                  <div
                      v-for="(log, index) in filteredLogs"
                      :key="index"
                      :class="['log-entry', getLogLevelClass(log)]"
                  >
                    <span class="log-time">{{ log.timestamp }}</span>
                    <span class="log-message">{{ log.message }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 文件预览模态框 -->
    <div v-if="showPreview" class="modal-overlay" @click="showPreview = false">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>文件预览</h3>
          <div class="modal-actions">
            <button @click="showPreview = false" class="close-btn">&times;</button>
          </div>
        </div>
        <div class="modal-body">
          <div v-if="previewError" class="preview-error">
            <span class="error-icon">⚠️</span>
            <p>{{ previewError }}</p>
          </div>
          <pre v-else-if="filePreview" class="file-preview">{{ filePreview }}</pre>
          <div v-else class="preview-loading">
            <span class="loading-spinner"></span>
            加载中...
          </div>
        </div>
      </div>
    </div>

    <!-- 正则配置对话框 -->
    <div v-if="showRegexDialog" class="modal-overlay" @click="closeRegexDialog">
      <div class="modal-content large-modal" @click.stop>
        <div class="modal-header">
          <h3>正则表达式配置</h3>
          <div class="modal-actions">
            <button @click="resetRegexConfig" class="btn-small btn-outline" title="重置为默认值">
              <span class="btn-icon">🔄</span>
              重置
            </button>
            <button @click="closeRegexDialog" class="close-btn">&times;</button>
          </div>
        </div>
        <div class="modal-body">
          <div class="regex-config-container">
            <!-- 内置正则配置 -->
            <div class="regex-section">
              <div class="section-header">
                <h4>内置正则表达式</h4>
                <div class="section-header-actions">
                  <span class="custom-regex-count">4 个规则</span>
                </div>
              </div>

              <div class="builtin-regex-container">
                <div class="builtin-regex-list">
                  <div class="builtin-regex-item">
                    <div class="regex-input-container">
                      <label class="regex-input-label">
                        <span class="regex-type-icon">🚩</span>
                        Flag检测正则:
                      </label>
                      <textarea
                          v-model="regexConfig.flagRegex"
                          placeholder="例如: flag{.*?}|ctf{.*?}|FLAG{.*?}"
                          rows="3"
                          class="regex-pattern-input"
                          @blur="validateBuiltinRegex('flagRegex')"
                          :class="{ 'input-error': hasBuiltinError('flagRegex') }"
                      ></textarea>
                      <div class="regex-stats">
                        <span class="regex-length">{{ regexConfig.flagRegex?.length || 0 }} 字符</span>
                        <span v-if="hasBuiltinError('flagRegex')" class="regex-error-msg">
                      {{ getBuiltinError('flagRegex') }}
                    </span>
                      </div>
                    </div>
                    <div class="regex-hint">
                      <span class="hint-icon">💡</span>
                      用于匹配CTF比赛中的Flag，通常以flag{开头，例如: flag{this_is_a_flag}
                    </div>
                  </div>

                  <div class="builtin-regex-item">
                    <div class="regex-input-container">
                      <label class="regex-input-label">
                        <span class="regex-type-icon">🗃️</span>
                        SQL语句检测正则:
                      </label>
                      <textarea
                          v-model="regexConfig.sqlRegex"
                          placeholder="例如: (SELECT|INSERT|UPDATE|DELETE).*FROM"
                          rows="3"
                          class="regex-pattern-input"
                          @blur="validateBuiltinRegex('sqlRegex')"
                          :class="{ 'input-error': hasBuiltinError('sqlRegex') }"
                      ></textarea>
                      <div class="regex-stats">
                        <span class="regex-length">{{ regexConfig.sqlRegex?.length || 0 }} 字符</span>
                        <span v-if="hasBuiltinError('sqlRegex')" class="regex-error-msg">
                      {{ getBuiltinError('sqlRegex') }}
                    </span>
                      </div>
                    </div>
                    <div class="regex-hint">
                      <span class="hint-icon">💡</span>
                      用于匹配SQL语句，常见于SQL注入攻击
                    </div>
                  </div>

                  <div class="builtin-regex-item">
                    <div class="regex-input-container">
                      <label class="regex-input-label">
                        <span class="regex-type-icon">🔑</span>
                        登录凭据检测正则:
                      </label>
                      <textarea
                          v-model="regexConfig.credentialRegex"
                          placeholder="例如: (username|password|user|pass)=[^&]+"
                          rows="3"
                          class="regex-pattern-input"
                          @blur="validateBuiltinRegex('credentialRegex')"
                          :class="{ 'input-error': hasBuiltinError('credentialRegex') }"
                      ></textarea>
                      <div class="regex-stats">
                        <span class="regex-length">{{ regexConfig.credentialRegex?.length || 0 }} 字符</span>
                        <span v-if="hasBuiltinError('credentialRegex')" class="regex-error-msg">
                      {{ getBuiltinError('credentialRegex') }}
                    </span>
                      </div>
                    </div>
                    <div class="regex-hint">
                      <span class="hint-icon">💡</span>
                      用于匹配HTTP请求中的用户名和密码等凭据信息
                    </div>
                  </div>

                  <div class="builtin-regex-item">
                    <div class="regex-input-container">
                      <label class="regex-input-label">
                        <span class="regex-type-icon">🔐</span>
                        各类秘钥检测正则:
                      </label>
                      <textarea
                          v-model="regexConfig.keyRegex"
                          placeholder="例如: (api[_-]?key|secret[_-]?key)=[^&]+"
                          rows="3"
                          class="regex-pattern-input"
                          @blur="validateBuiltinRegex('keyRegex')"
                          :class="{ 'input-error': hasBuiltinError('keyRegex') }"
                      ></textarea>
                      <div class="regex-stats">
                        <span class="regex-length">{{ regexConfig.keyRegex?.length || 0 }} 字符</span>
                        <span v-if="hasBuiltinError('keyRegex')" class="regex-error-msg">
                      {{ getBuiltinError('keyRegex') }}
                    </span>
                      </div>
                    </div>
                    <div class="regex-hint">
                      <span class="hint-icon">💡</span>
                      用于匹配API密钥、访问令牌等敏感信息
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 自定义正则配置 -->
            <div class="regex-section">
              <div class="section-header">
                <h4>自定义正则表达式</h4>
                <div class="section-header-actions">
              <span class="custom-regex-count">
                {{ Object.keys(customRegexDisplay).length }} 个规则
              </span>
                  <button @click="addCustomRegex" class="btn-small btn-primary add-regex-btn">
                    <span class="btn-icon">➕</span>
                    新增规则
                  </button>
                </div>
              </div>

              <div class="custom-regex-container">
                <div v-if="Object.keys(customRegexDisplay).length === 0" class="empty-custom-regex">
                  <div class="empty-icon">🔍</div>
                  <h5>暂无自定义正则表达式</h5>
                  <p>点击"新增规则"按钮创建自定义匹配规则</p>
                  <small>自定义正则可以帮助你匹配特定的数据模式</small>
                </div>

                <div v-else class="custom-regex-list">
                  <div
                      v-for="(regexData, name) in customRegexDisplay"
                      :key="name"
                      class="custom-regex-item"
                      :class="{
                  'regex-item-error': hasRegexError(name),
                  'regex-item-expanded': regexData.expanded
                }"
                  >
                    <div class="custom-regex-header" @click="toggleCustomRegex(name)">
                      <div class="regex-title-section">
                        <div class="regex-drag-handle" title="拖拽排序">⋮⋮</div>
                        <div class="regex-name-display">
                          <span class="regex-name-text">{{ regexData.displayName || name }}</span>
                          <span v-if="!regexData.expanded" class="regex-preview">
                        {{ truncateRegex(regexData.pattern) }}
                      </span>
                        </div>
                        <span v-if="hasRegexError(name)" class="regex-error-indicator" title="正则表达式语法错误">⚠️</span>
                        <span class="regex-expand-icon">
                      {{ regexData.expanded ? '▼' : '▶' }}
                    </span>
                      </div>
                      <div class="regex-actions" @click.stop>
                        <button
                            @click="testSingleRegex(name)"
                            class="btn-tiny btn-test"
                            title="测试此规则"
                            :disabled="!regexData.pattern?.trim()"
                        >
                          <span class="btn-icon">🧪</span>
                          测试
                        </button>
                        <button
                            @click="removeCustomRegex(name)"
                            class="btn-tiny btn-danger"
                            title="删除规则"
                        >
                          <span class="btn-icon">🗑️</span>
                        </button>
                      </div>
                    </div>

                    <div v-if="regexData.expanded" class="regex-content">
                      <div class="regex-input-container">
                        <label class="regex-input-label">规则名称:</label>
                        <input
                            type="text"
                            v-model="regexData.displayName"
                            placeholder="请输入规则名称"
                            class="regex-name-input"
                            @blur="validateRegexName(name)"
                        />
                      </div>

                      <div class="regex-input-container">
                        <label class="regex-input-label">正则表达式:</label>
                        <textarea
                            v-model="regexData.pattern"
                            placeholder="例如: example.*pattern"
                            rows="3"
                            class="regex-pattern-input"
                            :class="{ 'input-error': hasRegexError(name) }"
                            @blur="validateSingleRegex(name)"
                        ></textarea>
                        <div class="regex-stats">
                          <span class="regex-length">{{ regexData.pattern?.length || 0 }} 字符</span>
                          <span v-if="hasRegexError(name)" class="regex-error-msg">{{ getRegexError(name) }}</span>
                        </div>
                      </div>

                      <div class="regex-examples">
                        <div class="examples-header">
                          <span class="examples-label">匹配示例:</span>
                          <button @click="addExample(name)" class="btn-tiny btn-outline" title="添加测试用例">
                            <span class="btn-icon">+</span>
                          </button>
                        </div>
                        <div class="examples-list">
                          <div
                              v-for="(example, exIndex) in regexExamples[name] || []"
                              :key="exIndex"
                              class="example-item"
                              :class="{ 'example-matched': testExampleMatch(name, example) }"
                          >
                            <input
                                type="text"
                                v-model="regexExamples[name][exIndex]"
                                placeholder="输入测试文本"
                                class="example-input"
                            />
                            <div class="example-status">
                              <span v-if="testExampleMatch(name, example)" class="match-success">✓ 匹配</span>
                              <span v-else class="match-fail">✗ 不匹配</span>
                            </div>
                            <button
                                @click="removeExample(name, exIndex)"
                                class="btn-tiny btn-clear-example"
                                title="删除用例"
                            >
                              ×
                            </button>
                          </div>
                          <div v-if="!regexExamples[name]?.length" class="no-examples">
                            <small>暂无测试用例，点击 + 添加</small>
                          </div>
                        </div>
                      </div>
                    </div>

                    <div v-if="!regexData.expanded" class="regex-hint-collapsed">
                      <span class="hint-icon">💡</span>
                      点击展开编辑，支持自定义正则表达式匹配特定模式
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- 配置操作 -->
            <div class="config-actions">
              <button @click="testRegexConfig" class="btn-secondary">
                <span class="btn-icon">🧪</span>
                测试验证
              </button>
              <div class="action-spacer"></div>
              <button @click="closeRegexDialog" class="btn-outline">
                取消
              </button>
              <button @click="saveRegexConfig" class="btn-primary">
                <span class="btn-icon">💾</span>
                保存配置
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import {
  AnalyzePcap,
  CheckTshark,
  ExtractFiles,
  PreviewFile,
  CancelAnalysis,
  OpenFile,
  GetRegexConfig,
  UpdateRegexConfig,
  ResetRegexConfig
} from "../../../wailsjs/go/pcap/PcapAnalyzer";
import {EventsOn, OnFileDrop} from "../../../wailsjs/runtime";

const PCAP_LOG_SOURCE = 'CTF/PCAP分析';
import {ElMessage, ElMessageBox} from 'element-plus';
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';

export default {
  name: 'PcapAnalyzer',
  data() {
    return {
      eventUnsubscribers: [],
      wailsDropPaths: [],
      // 文件分页相关数据
      currentFilePage: 1,
      filePageSize: 20,
      sortBy: 'fileName',
      sortOrder: 'asc',
      showFileInfoModal: false,
      currentFileInfo: null,

      statusMessage: '准备就绪',
      statusClass: 'status-ready',
      selectedFilePath: '',
      currentRegexConfig: {
        flagRegex: '',
        sqlRegex: '',
        credentialRegex: '',
        keyRegex: '',
        customRegex: {}
      },
      // 正则管理
      showRegexDialog: false,
      regexConfig: {
        flagRegex: '',
        sqlRegex: '',
        credentialRegex: '',
        keyRegex: '',
        customRegex: {}
      },
      regexExamples: {}, // 存储每个正则的测试用例
      regexErrors: {},   // 存储每个正则的错误信息
      builtinRegexErrors: {}, // 存储内置正则的错误信息
      customRegexDisplay: {}, // 自定义正则的显示状态
      nextCustomId: 1,

      // 检测选项配置
      detectionOptions: [
        {
          key: 'extractFiles',
          label: '文件提取',
          description: '从流量中提取传输的文件',
          enabled: false
        },
        {
          key: 'detectFlags',
          label: 'Flag检测',
          description: '检测CTF比赛中的flag',
          enabled: true
        },
        {
          key: 'detectSQL',
          label: 'SQL语句检测',
          description: '检测SQL注入攻击特征',
          enabled: true
        },
        {
          key: 'detectCredentials',
          label: '登录凭据检测',
          description: '检测用户名密码等凭据',
          enabled: true
        },
        {
          key: 'detectKeys',
          label: '各类秘钥检测',
          description: '检测API密钥、令牌等',
          enabled: true
        }
      ],

      analysisResults: [],
      extractedFiles: [],
      logEntries: [],
      activeTab: 'logs',
      showPreview: false,
      filePreview: '',
      previewError: '',
      currentPreviewFile: '',
      isProcessing: false,

      // 处理统计
      processingStats: {
        packets: 0,
        matches: 0
      },

      // 搜索功能
      searchQuery: '',
      fileSearchQuery: '',

      // 日志过滤
      logFilters: {
        info: true,
        success: true,
        warning: true,
        error: true,
        match: true,
        progress: true,
        file: true
      },

      // 实时更新控制
      realtimeUpdate: true,

      // 状态轮询
      statusTimer: null,

      tabs: [
        { id: 'results', label: '分析结果', icon: '🔍' },
        { id: 'files', label: '提取文件', icon: '📁' },
        { id: 'logs', label: '日志输出', icon: '📝' }
      ],
    }
  },
  computed: {
    // 添加文件统计相关计算属性
    formatTotalSize() {
      const totalBytes = this.filteredFiles.reduce((sum, file) => sum + file.size, 0);
      return this.formatFileSize(totalBytes);
    },

    protocolStats() {
      const stats = {};
      this.filteredFiles.forEach(file => {
        stats[file.protocol] = (stats[file.protocol] || 0) + 1;
      });
      return Object.entries(stats).map(([protocol, count]) => `${protocol}:${count}`).join(', ');
    },

    // 添加排序后的文件列表
    sortedFiles() {
      const files = [...this.filteredFiles];
      return files.sort((a, b) => {
        let aVal = a[this.sortBy];
        let bVal = b[this.sortBy];

        if (this.sortBy === 'size') {
          aVal = Number(aVal);
          bVal = Number(bVal);
        } else {
          aVal = String(aVal).toLowerCase();
          bVal = String(bVal).toLowerCase();
        }

        if (aVal < bVal) return this.sortOrder === 'asc' ? -1 : 1;
        if (aVal > bVal) return this.sortOrder === 'asc' ? 1 : -1;
        return 0;
      });
    },
    // 分页相关计算属性
    filteredFiles() {
      if (!this.fileSearchQuery) {
        return this.extractedFiles;
      }

      const query = this.fileSearchQuery.toLowerCase();
      return this.extractedFiles.filter(file =>
          file.fileName.toLowerCase().includes(query) ||
          file.fileType.toLowerCase().includes(query) ||
          file.protocol.toLowerCase().includes(query) ||
          (file.filePath && file.filePath.toLowerCase().includes(query))
      );
    },

    totalFilePages() {
      return Math.ceil(this.filteredFiles.length / this.filePageSize);
    },

    // 更新分页计算属性使用排序后的文件
    paginatedFiles() {
      const start = (this.currentFilePage - 1) * this.filePageSize;
      const end = start + this.filePageSize;
      return this.sortedFiles.slice(start, end);
    },

    startFileIndex() {
      return (this.currentFilePage - 1) * this.filePageSize;
    },

    endFileIndex() {
      return Math.min(this.startFileIndex + this.filePageSize, this.filteredFiles.length);
    },

    // 可见页码计算
    visiblePageNumbers() {
      const current = this.currentFilePage;
      const total = this.totalFilePages;
      const pages = [];

      // 显示最多7个页码
      let start = Math.max(1, current - 3);
      let end = Math.min(total, start + 6);

      // 调整起始位置以确保显示7个页码（如果可能）
      if (end - start < 6) {
        start = Math.max(1, end - 6);
      }

      for (let i = start; i <= end; i++) {
        pages.push(i);
      }

      return pages;
    },

    showPageEllipsis() {
      return this.totalFilePages > this.visiblePageNumbers[this.visiblePageNumbers.length - 1];
    },

    logs() {
      return this.logEntries.map(entry => `[${entry.timestamp}] ${entry.message}`).join('\n');
    },

    // 类型统计
    typeStats() {
      const stats = {};
      this.analysisResults.forEach(result => {
        // 将文件类型映射到基础类型进行统计
        let statType = result.type;
        if (statType.startsWith('file_')) {
          statType = statType.replace('file_', '');
        }
        stats[statType] = (stats[statType] || 0) + 1;
      });

      return Object.keys(stats).map(type => ({
        type,
        label: this.getResultTypeLabel(type),
        count: stats[type]
      }));
    },

    // 过滤后的分析结果
    filteredResults() {
      if (!this.searchQuery) {
        return this.analysisResults;
      }

      const query = this.searchQuery.toLowerCase();
      return this.analysisResults.filter(result =>
          result.content.toLowerCase().includes(query) ||
          this.getResultTypeLabel(result.type).toLowerCase().includes(query) ||
          result.packetNum.toString().includes(query)
      );
    },

    // 过滤后的日志
    filteredLogs() {
      return this.logEntries.filter(log => {
        switch (log.level) {
          case 'info': return this.logFilters.info;
          case 'success': return this.logFilters.success;
          case 'warning': return this.logFilters.warning;
          case 'error': return this.logFilters.error;
          case 'match': return this.logFilters.match;
          case 'progress': return this.logFilters.info; // 进度日志作为信息显示
          case 'file': return this.logFilters.info; // 文件日志作为信息显示
          default: return true;
        }
      });
    }
  },
  watch: {
    // 当搜索条件改变时重置到第一页
    fileSearchQuery() {
      this.currentFilePage = 1;
    },

    // 当文件列表变化时检查当前页是否有效
    extractedFiles() {
      if (this.currentFilePage > this.totalFilePages && this.totalFilePages > 0) {
        this.currentFilePage = this.totalFilePages;
      }
    },

    logEntries: {
      handler() {
        this.$nextTick(() => {
          const container = this.$refs.logsContent;
          if (container) {
            container.scrollTop = container.scrollHeight;
          }
        });
      },
      deep: true
    }
  },
  mounted() {
    // console.log('🚀 应用程序启动');
    this.checkEnvironment();
    this.addLog('应用程序已启动', 'info');

    // 注册后端事件监听
    this.setupEventListeners();

    // 启动状态轮询
    this.startStatusPolling();

    // 加载正则配置
    this.loadRegexConfig();

    // 启用拖拽选取文件
    this.setupFileDropListener();
  },
  beforeUnmount() {
    this.cleanupEventListeners();
  },
  methods: {
    // 加载正则配置
    async loadRegexConfig() {
      try {
        const configJson = await GetRegexConfig();
        const config = JSON.parse(configJson);
        this.currentRegexConfig = config;
        this.addLog('正则表达式配置加载完成', 'success');
      } catch (error) {
        this.addLog(`加载正则配置失败: ${error}`, 'error');
        ElMessage.error(`加载正则配置失败: ${error}`);
      }
    },

    sortFiles(field) {
      if (this.sortBy === field) {
        this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc';
      } else {
        this.sortBy = field;
        this.sortOrder = 'asc';
      }
    },

    getFileIconClass(fileType) {
      const typeMap = {
        'PNG图像': 'icon-image',
        'JPEG图像': 'icon-image',
        'GIF图像': 'icon-image',
        'BMP图像': 'icon-image',
        'PDF文档': 'icon-pdf',
        'ZIP压缩包': 'icon-archive',
        'RAR压缩包': 'icon-archive',
        '文本文件': 'icon-text',
        'HTML文档': 'icon-web',
        'JSON文档': 'icon-data',
        'XML文档': 'icon-data',
        '可执行文件': 'icon-executable',
        'Windows可执行文件': 'icon-executable',
        '音频文件': 'icon-audio',
        '视频文件': 'icon-video'
      };
      return typeMap[fileType] || 'icon-default';
    },

    truncateFileName(name, maxLength = 25) {
      if (name.length <= maxLength) return name;
      return name.substring(0, maxLength - 3) + '...';
    },

    truncateFilePath(path, maxLength = 40) {
      if (path.length <= maxLength) return path;
      const parts = path.split(/[\\/]/);
      if (parts.length <= 2) return path;

      const fileName = parts.pop();
      const remainingLength = maxLength - fileName.length - 5; // 5 for ".../"
      if (remainingLength <= 0) return '.../' + fileName;

      const folderPath = parts.join('/');
      if (folderPath.length <= remainingLength) return folderPath + '/' + fileName;

      return '...' + folderPath.substring(folderPath.length - remainingLength) + '/' + fileName;
    },

    truncateHash(hash, maxLength = 12) {
      if (hash.length <= maxLength) return hash;
      return hash.substring(0, maxLength - 3) + '...';
    },

    showFileInfo(file) {
      this.currentFileInfo = file;
      this.showFileInfoModal = true;
      this.addLog(`查看文件信息: ${file.fileName}`, 'info');
    },

    closeFileInfoModal() {
      this.showFileInfoModal = false;
      this.currentFileInfo = null;
    },

    copyFileInfo(field, value) {
      this.copyToClipboard(value);
      this.addLog(`已复制${field}: ${value}`, 'success');
    },

    // 文件分页方法
    nextFilePage() {
      if (this.currentFilePage < this.totalFilePages) {
        this.currentFilePage++;
      }
    },

    prevFilePage() {
      if (this.currentFilePage > 1) {
        this.currentFilePage--;
      }
    },

    goToFilePage(page) {
      if (page >= 1 && page <= this.totalFilePages) {
        this.currentFilePage = page;
      }
    },

    goToFirstFilePage() {
      this.currentFilePage = 1;
    },

    goToLastFilePage() {
      this.currentFilePage = this.totalFilePages;
    },

    onFilePageSizeChange() {
      this.currentFilePage = 1; // 重置到第一页
    },
    // 设置事件监听器
    setupEventListeners() {
      // console.log('🔧 注册事件监听器...');

      // 仅接收 PCAP 模块日志，避免其他模块的全局 log_event 混入
      this.eventUnsubscribers.push(
        EventsOn('log_event', (eventData) => {
          if (eventData?.source && eventData.source !== PCAP_LOG_SOURCE) {
            return;
          }

          this.addLog(eventData.message, eventData.level);

          if (eventData.level === 'progress') {
            this.updateStatsFromLog(eventData.message);
          }
        })
      );

      // 监听匹配结果事件
      this.eventUnsubscribers.push(
        EventsOn('match_event', (result) => {
        // console.log('🎯 收到匹配事件:', result);

        // 检查是否已存在相同的结果（避免重复）
        const exists = this.analysisResults.some(r =>
            r.packetNum === result.packetNum &&
            r.type === result.type &&
            r.content === result.content
        );

        if (!exists) {
          // 实时添加匹配结果
          this.analysisResults.push({
            ...result,
            timestamp: result.timestamp || new Date().toLocaleTimeString()
          });

          // 如果有新匹配，自动切换到结果标签页
          // if (this.activeTab !== 'results' && this.realtimeUpdate) {
          //   this.activeTab = 'results';
          // }

          // 更新统计
          this.processingStats.matches = this.analysisResults.length;
          this.processingStats.packets = Math.max(this.processingStats.packets, result.packetNum);
        }
        })
      );

      // 监听文件提取事件
      this.eventUnsubscribers.push(
        EventsOn('file_extracted', (file) => {
        // console.log('📁 收到文件事件:', file);

        // 检查是否已存在相同的文件
        const exists = this.extractedFiles.some(f =>
            f.filePath === file.filePath &&
            f.fileName === file.fileName
        );

        if (!exists) {
          // 实时添加提取的文件
          this.extractedFiles.push(file);

          // 如果有新文件，自动切换到文件标签页
          // if (this.activeTab !== 'files' && this.realtimeUpdate) {
          //   this.activeTab = 'files';
          // }

          // 发送日志通知
          this.addLog(`✅ 实时提取文件: ${file.fileName}, 类型: ${file.fileType}, 大小: ${this.formatFileSize(file.size)}`, 'success');
        }
        })
      );

      // 监听分析状态事件
      this.eventUnsubscribers.push(
        EventsOn('analysis_status', (status) => {
          this.processingStats.packets = status.packets || 0;
          this.processingStats.matches = status.matches || this.analysisResults.length;
        })
      );
    },

    // 清理事件监听（使用 EventsOn 返回的取消函数，避免影响全局 log_event 监听）
    cleanupEventListeners() {
      this.eventUnsubscribers.forEach(unsub => unsub?.());
      this.eventUnsubscribers = [];
    },

    // 从日志中更新统计信息
    updateStatsFromLog(logMessage) {
      // 从日志中提取数据包数量
      const packetMatch = logMessage.match(/已处理\s+(\d+)\s+个数据包/);
      if (packetMatch) {
        this.processingStats.packets = parseInt(packetMatch[1]);
      }

      // 从日志中提取匹配项数量
      const matchMatch = logMessage.match(/找到\s+(\d+)\s+个匹配项/);
      if (matchMatch) {
        this.processingStats.matches = parseInt(matchMatch[1]);
      }
    },

    // 启动状态轮询
    startStatusPolling() {
      this.statusTimer = setInterval(() => {
        // 这里可以添加定期状态检查的逻辑
        // 目前主要依赖事件机制
      }, 2000);
    },

    // 切换实时更新模式
    toggleRealtimeUpdate() {
      this.realtimeUpdate = !this.realtimeUpdate;
      this.addLog(`实时更新模式: ${this.realtimeUpdate ? '开启' : '关闭'}`, 'info');
    },

    // 搜索输入处理
    onSearchInput() {
      // 搜索时自动切换到结果标签页
      if (this.searchQuery && this.activeTab !== 'results') {
        this.activeTab = 'results';
      }
    },

    // 清除搜索
    clearSearch() {
      this.searchQuery = '';
    },

    // 高亮显示结果
    highlightResult(result) {
      let displayInfo = '';
      if (this.isFileResult(result)) {
        displayInfo = `文件: ${result.fileName} (${result.protocol})`;
      } else {
        displayInfo = `数据包 #${result.packetNum}`;
      }

      this.addLog(`🔍 高亮显示: ${this.getResultTypeLabel(result.type)} - ${result.content} [${displayInfo}]`, 'info');

      // 这里可以添加实际的高亮逻辑，比如滚动到对应元素
      this.highlightResultElement(result);
    },

    // 添加高亮元素的方法
    highlightResultElement(result) {
      this.$nextTick(() => {
        // 找到对应的结果卡片并添加高亮样式
        const cards = document.querySelectorAll('.result-card');
        cards.forEach(card => {
          card.classList.remove('highlighted');
        });

        // 这里需要根据result找到对应的DOM元素
        // 由于Vue的虚拟DOM，我们可以通过设置一个ref或者使用自定义属性来定位
        const resultId = `${result.type}-${result.packetNum || result.fileName}`;
        const targetCard = document.querySelector(`[data-result-id="${resultId}"]`);
        if (targetCard) {
          targetCard.classList.add('highlighted');
          targetCard.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
      });
    },

    getDetectionOption(key) {
      return this.detectionOptions.find(opt => opt.key === key)?.enabled || false;
    },

    // 注册全局拖拽监听（支持桌面拖拽）
    setupFileDropListener() {
      // Wails 全局文件拖拽（绝对路径）
      OnFileDrop((x, y, paths) => {
        if (paths && paths.length > 0) {
          this.wailsDropPaths = paths;
          this.applyDroppedPcapPath(paths[0]);
          return;
        }

        // 兜底：复用本地处理逻辑
        this.handleFileDrop({
          preventDefault: () => {},
          paths: paths || []
        });
      });

      // 全局事件广播（后端 runtime.EventsEmit）
      EventsOn('global-file-drop', (paths) => {
        if (paths && paths.length > 0) {
          this.wailsDropPaths = paths;
          this.applyDroppedPcapPath(paths[0]);
        }
      });

      // 防止浏览器默认打开文件，确保拖拽事件可用
      window.addEventListener('dragover', e => e.preventDefault());
      window.addEventListener('drop', e => e.preventDefault());
    },

    // 统一提取拖拽的文件路径，兼容浏览器与 Wails 事件
    extractDroppedFilePaths(event) {
      let filePaths = [];

      // 优先使用 Wails 级别的路径（绝对路径）
      if (this.wailsDropPaths && this.wailsDropPaths.length) {
        filePaths = this.wailsDropPaths;
        this.wailsDropPaths = [];
      }

      if (event?.dataTransfer?.files?.length) {
        filePaths = Array.from(event.dataTransfer.files)
            .map(file => file.path || file.webkitRelativePath || file.name);
      } else if (event?.detail?.paths) {
        filePaths = event.detail.paths;
      } else if (event?.paths) {
        filePaths = event.paths;
      }

      // 进一步尝试从 dataTransfer 的文本数据中解析 file:// URI
      if (filePaths.length === 0 && event?.dataTransfer?.getData) {
        const uriList = event.dataTransfer.getData('text/uri-list') || event.dataTransfer.getData('text/plain');
        if (uriList) {
          const candidate = uriList.split('\n').find(line => line.trim().startsWith('file://') || /^[a-zA-Z]:\\/.test(line.trim()));
          if (candidate) {
            const normalized = candidate.trim().startsWith('file://')
                ? decodeURI(candidate.trim().replace('file://', '').replace(/\//g, '\\'))
                : candidate.trim();
            filePaths = [normalized];
          }
        }
      }

      return filePaths;
    },

    // 根据拖拽的路径处理 PCAP 选择
    applyDroppedPcapPath(filePath) {
      if (!filePath) return;

      if (filePath.match(/\.(pcap|pcapng)$/i)) {
        this.selectedFilePath = filePath;
        this.addLog(`通过拖拽选择文件: ${this.getFileName(filePath)}`, 'success');
      } else {
        this.addLog('请选择有效的PCAP文件', 'error');
      }
    },

    async selectFile() {
      try {
        const result = await OpenFile({
          Title: "选择PCAP文件",
          Filters: [
            { DisplayName: "PCAP Files", Pattern: "*.pcap;*.pcapng" },
            { DisplayName: "All Files", Pattern: "*" }
          ]
        });

        if (result) {
          this.selectedFilePath = result;
          this.addLog(`已选择文件: ${this.getFileName(result)}`, 'success');
        }
      } catch (error) {
        this.addLog(`文件选择错误: ${error}`, 'error');
      }
    },

    handleFileDrop(event) {
      event?.preventDefault?.();

      let filePaths = this.extractDroppedFilePaths(event);
      if (!filePaths.length) return;

      this.applyDroppedPcapPath(filePaths[0]);
    },

    // 输入框拖拽填充
    handleFilePathInputDrop(event) {
      event?.preventDefault?.();

      let filePaths = this.extractDroppedFilePaths(event);
      if (!filePaths.length) return;

      this.applyDroppedPcapPath(filePaths[0]);
    },

    clearFile() {
      this.selectedFilePath = '';
      this.addLog('已清除文件选择', 'info');
    },

    getFileName(path) {
      return path.split(/[\\/]/).pop();
    },

    async analyzePcap() {
      if (!this.selectedFilePath) {
        this.addLog("请先选择PCAP文件", 'error');
        return;
      }

      this.isProcessing = true;
      this.processingStats = { packets: 0, matches: 0 };
      this.statusMessage = '分析中...';
      this.statusClass = 'status-working';

      // 清空之前的结果
      this.analysisResults = [];
      this.extractedFiles = [];
      this.searchQuery = '';
      this.fileSearchQuery = '';

      this.addLog(`开始分析文件: ${this.getFileName(this.selectedFilePath)}`, 'info');

      try {
        const config = {
          filePath: this.selectedFilePath,
          // 不再传递正则表达式，后端会使用配置文件中的正则
          flagRegex: '',
          sqlRegex: '',
          credentialRegex: '',
          keyRegex: '',
          extractFiles: this.getDetectionOption('extractFiles'),
          detectFlags: this.getDetectionOption('detectFlags'),
          detectSQL: this.getDetectionOption('detectSQL'),
          detectCredentials: this.getDetectionOption('detectCredentials'),
          detectKeys: this.getDetectionOption('detectKeys')
        };

        // 不等待分析完成，让事件系统处理实时更新
        const analysisPromise = AnalyzePcap(config);

        // 设置超时检查
        const timeoutPromise = new Promise((resolve, reject) => {
          setTimeout(() => {
            if (this.isProcessing) {
              this.addLog('分析仍在进行中，请稍候...', 'info');
            }
          }, 5000);
        });

        // 等待分析完成或超时
        const result = await Promise.race([analysisPromise, timeoutPromise.then(() => null)]);

        if (result) {
          // 检查是否有错误
          if (result.startsWith("❌") && !result.includes("tshark命令执行出错")) {
            throw new Error(result);
          }

          // 解析最终结果
          try {
            const parsedResult = JSON.parse(result);

            // 处理分析结果
            if (parsedResult.analysisResults && parsedResult.analysisResults.length > 0) {
              // 使用更精确的重复检查
              const existingResults = new Set(
                  this.analysisResults.map(r =>
                      `${r.type}-${r.packetNum || 'file'}-${r.content}-${r.fileName || ''}`
                  )
              );

              const newResults = parsedResult.analysisResults.filter(r => {
                const key = `${r.type}-${r.packetNum || 'file'}-${r.content}-${r.fileName || ''}`;
                return !existingResults.has(key);
              });

              if (newResults.length > 0) {
                this.analysisResults.push(...newResults);
                this.addLog(`最终合并 ${newResults.length} 个匹配结果`, 'info');
              }
            }

            // 处理提取的文件
            if (parsedResult.extractedFiles && parsedResult.extractedFiles.length > 0) {
              const existingFilePaths = new Set(this.extractedFiles.map(f => f.filePath));
              const newFiles = parsedResult.extractedFiles.filter(f => !existingFilePaths.has(f.filePath));
              if (newFiles.length > 0) {
                this.extractedFiles.push(...newFiles);
                this.addLog(`最终合并 ${newFiles.length} 个提取文件`, 'info');
              }
            }
          } catch (parseError) {
            this.addLog(`解析最终结果时出错: ${parseError}`, 'warning');
          }

          this.statusMessage = '分析完成';
          this.statusClass = 'status-ready';

          // 更新统计信息
          const fileResults = this.analysisResults.filter(r => this.isFileResult(r)).length;
          const packetResults = this.analysisResults.filter(r => !this.isFileResult(r)).length;

          this.addLog(`分析完成，数据包匹配 ${packetResults} 个结果，文件内容匹配 ${fileResults} 个结果，提取 ${this.extractedFiles.length} 个文件`, 'success');
        }

      } catch (error) {
        this.statusMessage = '分析失败';
        this.statusClass = 'status-error';
        const errorMsg = error.message || error;
        this.addLog(`分析错误: ${errorMsg}`, 'error');

        if (errorMsg.includes('取消')) {
          this.statusMessage = '分析已取消';
          this.statusClass = 'status-ready';
        }
      } finally {
        this.isProcessing = false;
      }
    },

    async extractFilesHandler() {
      if (!this.selectedFilePath) {
        this.addLog("请先选择PCAP文件", 'error');
        return;
      }

      this.isProcessing = true;
      this.statusMessage = '提取文件中...';
      this.statusClass = 'status-working';

      this.addLog(`开始提取文件: ${this.getFileName(this.selectedFilePath)}`, 'info');

      try {
        const result = await ExtractFiles(this.selectedFilePath);

        if (result.startsWith("❌") || result.includes("错误")) {
          throw new Error(result);
        }

        this.extractedFiles = JSON.parse(result);
        this.activeTab = 'files';
        this.statusMessage = '文件提取完成';
        this.statusClass = 'status-ready';
        this.addLog(`文件提取完成，共提取 ${this.extractedFiles.length} 个文件`, 'success');

      } catch (error) {
        this.statusMessage = '文件提取失败';
        this.statusClass = 'status-error';
        this.addLog(`文件提取错误: ${error.message || error}`, 'error');
      } finally {
        this.isProcessing = false;
      }
    },

    // 添加从结果预览文件的方法
    previewFileFromResult(result) {
      if (result.filePath) {
        this.previewFile({
          filePath: result.filePath,
          fileName: result.fileName
        });
      }
    },

    async cancelAnalysis() {
      try {
        const result = await CancelAnalysis();
        this.addLog(result, 'info');
        this.statusMessage = '分析已取消';
        this.statusClass = 'status-ready';
      } catch (error) {
        this.addLog(`取消分析失败: ${error}`, 'error');
      }
    },

    clearResults() {
      this.analysisResults = [];
      this.extractedFiles = [];
      this.addLog('已清空所有结果', 'info');
    },

    clearLogs() {
      this.logEntries = [];
      this.addLog('日志已清空', 'info');
    },



    addLog(message, level = 'info') {
      const timestamp = new Date().toLocaleTimeString();
      this.logEntries.push({
        timestamp,
        message,
        level
      });

      // 限制日志数量，避免内存泄漏
      if (this.logEntries.length > 2000) {
        this.logEntries = this.logEntries.slice(-1000);
      }
    },

    getLogLevelClass(log) {
      return `log-${log.level}`;
    },

    getResultClass(type) {
      // 移除file_前缀，使用基础类型样式
      const baseType = type.replace('file_', '');
      return `result-${baseType}`;
    },


    // 判断是否为文件结果
    isFileResult(result) {
      return result.type && result.type.startsWith('file_');
    },

    getResultTypeLabel(type) {
      const labels = {
        flag: 'Flag',
        sql: 'SQL语句',
        credential: '凭据',
        key: '密钥',
        file_flag: '文件Flag',
        file_sql: '文件SQL',
        file_credential: '文件凭据',
        file_key: '文件密钥'
      };
      return labels[type] || type;
    },

    getResultTime(index) {
      return new Date().toLocaleTimeString();
    },

    // 获取文件图标
    getFileIcon(fileType) {
      const icons = {
        'PNG图像': '🖼️',
        'JPEG图像': '🖼️',
        'GIF图像': '🖼️',
        'BMP图像': '🖼️',
        'PDF文档': '📄',
        'ZIP压缩包': '📦',
        'RAR压缩包': '📦',
        '文本文件': '📃',
        'HTML文档': '🌐',
        'JSON文档': '📊',
        'XML文档': '📊',
        '可执行文件': '⚙️',
        'Windows可执行文件': '⚙️',
        '音频文件': '🎵',
        '视频文件': '🎬',
        'Word文档': '📝',
        'Excel文档': '📊',
        'PowerPoint文档': '📑',
        '数据文件': '📊',
        '网页文件': '🌐'
      };
      return icons[fileType] || '📄';
    },


    // 预览文件
    async previewFile(file) {
      try {
        this.showPreview = true;
        this.filePreview = '';
        this.previewError = '';
        this.currentPreviewFile = file.filePath;

        this.addLog(`正在预览文件: ${file.fileName}`, 'info');

        const result = await PreviewFile(file.filePath);
        this.filePreview = result;
      } catch (error) {
        this.previewError = `预览失败: ${error}`;
        this.addLog(`文件预览错误: ${error}`, 'error');
      }
    },

    // 格式化文件大小
    formatFileSize(bytes) {
      if (bytes === 0) return '0 B';
      const k = 1024;
      const sizes = ['B', 'KB', 'MB', 'GB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    },

    hasTabData(tabId) {
      switch (tabId) {
        case 'results': return this.analysisResults.length > 0;
        case 'files': return this.extractedFiles.length > 0;
        case 'logs': return this.logEntries.length > 0;
        default: return false;
      }
    },

    getTabCount(tabId) {
      switch (tabId) {
        case 'results': return this.analysisResults.length;
        case 'files': return this.extractedFiles.length;
        case 'logs': return this.logEntries.length;
        default: return 0;
      }
    },

    async copyToClipboard(text) {
      try {
        await navigator.clipboard.writeText(text);
        this.addLog('内容已复制到剪贴板', 'success');
      } catch (error) {
        this.addLog('复制失败', 'error');
      }
    },

    exportResults() {
      const data = {
        analysisResults: this.analysisResults,
        exportTime: new Date().toISOString(),
        file: this.selectedFilePath
      };
      this.downloadJSON(data, 'pcap-analysis-results.json');
      this.addLog('分析结果已导出', 'success');
    },

    exportLogs() {
      this.downloadText(this.logs, 'pcap-analysis-logs.txt');
      this.addLog('日志已导出', 'success');
    },

    downloadJSON(data, filename) {
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
      this.downloadBlob(blob, filename);
    },

    downloadText(text, filename) {
      const blob = new Blob([text], { type: 'text/plain' });
      this.downloadBlob(blob, filename);
    },

    downloadBlob(blob, filename) {
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = filename;
      link.click();
      URL.revokeObjectURL(url);
    },

    async checkEnvironment() {
      this.statusMessage = '检查环境中...';
      this.statusClass = 'status-checking';
      this.addLog('开始检查tshark环境...', 'info');

      try {
        const result = await CheckTshark();
        // 更灵活的环境检查判断
        if (result.includes('正常') || result.includes('✅') || result.includes('tshark环境检查通过')) {
          this.statusMessage = '环境正常';
          this.statusClass = 'status-ready';
          this.addLog('环境检查通过', 'success');
        } else {
          this.statusMessage = '环境异常';
          this.statusClass = 'status-error';
          this.addLog('环境检查失败: ' + result, 'error');
        }
      } catch (error) {
        this.statusMessage = '检查失败';
        this.statusClass = 'status-error';
        this.addLog('环境检查错误: ' + error, 'error');
      }
    },

    // 正则配置相关方法
    async showRegexConfig() {
      try {
        this.addLog('正在加载正则表达式配置...', 'info');
        const configJson = await GetRegexConfig();
        const config = JSON.parse(configJson);

        this.regexConfig = {
          flagRegex: config.flagRegex || '',
          sqlRegex: config.sqlRegex || '',
          credentialRegex: config.credentialRegex || '',
          keyRegex: config.keyRegex || '',
          customRegex: config.customRegex || {}
        };

        // 初始化自定义正则显示状态
        this.customRegexDisplay = {};
        Object.keys(this.regexConfig.customRegex).forEach(key => {
          this.customRegexDisplay[key] = {
            pattern: this.regexConfig.customRegex[key],
            displayName: key,
            expanded: false // 保存后默认折叠
          };
        });

        this.showRegexDialog = true;
        this.addLog('正则表达式配置加载完成', 'success');
      } catch (error) {
        this.addLog(`加载正则配置失败: ${error}`, 'error');
        ElMessage.error(`加载正则配置失败: ${error}`);
      }
    },

    closeRegexDialog() {
      this.showRegexDialog = false;
    },

    // 添加自定义正则
    addCustomRegex() {
      const newId = `custom_${Date.now()}_${this.nextCustomId++}`;
      const newRegex = {
        pattern: '',
        displayName: `规则${this.nextCustomId}`,
        expanded: true // 新增时展开
      };

      this.regexConfig.customRegex = {
        ...this.regexConfig.customRegex,
        [newId]: ''
      };

      this.customRegexDisplay = {
        ...this.customRegexDisplay,
        [newId]: newRegex
      };

      // 初始化测试用例数组
      this.regexExamples = {
        ...this.regexExamples,
        [newId]: []
      };

      ElMessage.success('已添加新的自定义规则');
    },

    // 切换自定义正则的展开状态
    toggleCustomRegex(name) {
      if (this.customRegexDisplay[name]) {
        this.customRegexDisplay[name].expanded = !this.customRegexDisplay[name].expanded;
      }
    },

    // 验证内置正则
    validateBuiltinRegex(field) {
      const pattern = this.regexConfig[field];
      if (!pattern?.trim()) {
        const newErrors = { ...this.builtinRegexErrors };
        delete newErrors[field];
        this.builtinRegexErrors = newErrors;
        return true;
      }

      try {
        new RegExp(pattern);
        const newErrors = { ...this.builtinRegexErrors };
        delete newErrors[field];
        this.builtinRegexErrors = newErrors;
        return true;
      } catch (error) {
        this.builtinRegexErrors = {
          ...this.builtinRegexErrors,
          [field]: error.message
        };
        return false;
      }
    },

    // 检查内置正则是否有错误
    hasBuiltinError(field) {
      return !!this.builtinRegexErrors[field];
    },

    // 获取内置正则错误信息
    getBuiltinError(field) {
      return this.builtinRegexErrors[field];
    },

    async removeCustomRegex(name) {
      try {
        await confirmDangerousAction({
          title: '确认删除',
          message: `确定要删除自定义正则规则「${name}」吗？`,
          expectedInput: DANGER_DELETE_TOKEN
        });
      } catch {
        return;
      }
      const newCustomRegex = { ...this.regexConfig.customRegex };
      delete newCustomRegex[name];
      this.regexConfig.customRegex = newCustomRegex;

      const newCustomDisplay = { ...this.customRegexDisplay };
      delete newCustomDisplay[name];
      this.customRegexDisplay = newCustomDisplay;

      const newExamples = { ...this.regexExamples };
      delete newExamples[name];
      this.regexExamples = newExamples;

      const newErrors = { ...this.regexErrors };
      delete newErrors[name];
      this.regexErrors = newErrors;

      ElMessage.success('自定义规则已删除');
    },


    // 添加测试用例
    addExample(regexName) {
      if (!this.regexExamples[regexName]) {
        this.regexExamples = {
          ...this.regexExamples,
          [regexName]: []
        };
      }

      this.regexExamples[regexName].push('');
    },

    // 删除测试用例
    removeExample(regexName, index) {
      if (this.regexExamples[regexName]) {
        const newExamples = [...this.regexExamples[regexName]];
        newExamples.splice(index, 1);
        this.regexExamples = {
          ...this.regexExamples,
          [regexName]: newExamples
        };
      }
    },

    // 验证单个正则表达式（自定义）
    validateSingleRegex(regexName) {
      const pattern = this.customRegexDisplay[regexName]?.pattern;
      if (!pattern?.trim()) {
        const newErrors = { ...this.regexErrors };
        delete newErrors[regexName];
        this.regexErrors = newErrors;
        return true;
      }

      try {
        new RegExp(pattern);
        const newErrors = { ...this.regexErrors };
        delete newErrors[regexName];
        this.regexErrors = newErrors;
        return true;
      } catch (error) {
        this.regexErrors = {
          ...this.regexErrors,
          [regexName]: error.message
        };
        return false;
      }
    },

    // 验证规则名称
    validateRegexName(regexName) {
      const regexData = this.customRegexDisplay[regexName];
      if (!regexData?.displayName?.trim()) {
        this.customRegexDisplay[regexName].displayName = `规则${regexName.split('_')[1]}`;
        ElMessage.warning('规则名称不能为空，已使用默认名称');
      }
    },

    // 检查正则是否有错误
    hasRegexError(regexName) {
      return !!this.regexErrors[regexName];
    },

    // 获取正则错误信息
    getRegexError(regexName) {
      return this.regexErrors[regexName];
    },

    // 测试单个正则
    testSingleRegex(regexName) {
      const pattern = this.regexConfig.customRegex[regexName];
      if (!pattern?.trim()) {
        ElMessage.warning('请输入正则表达式');
        return;
      }

      if (!this.validateSingleRegex(regexName)) {
        ElMessage.error(`正则表达式语法错误: ${this.regexErrors[regexName]}`);
        return;
      }

      // 如果有测试用例，显示匹配结果
      const examples = this.regexExamples[regexName] || [];
      if (examples.length > 0) {
        const matchedCount = examples.filter(example =>
            example && this.testExampleMatch(regexName, example)
        ).length;
        ElMessage.success(`测试完成: ${matchedCount}/${examples.length} 个用例匹配成功`);
      } else {
        ElMessage.success('正则表达式语法正确');
      }
    },

    // 测试用例匹配
    testExampleMatch(regexName, example) {
      if (!example?.trim()) return false;

      const pattern = this.regexConfig.customRegex[regexName];
      if (!pattern?.trim()) return false;

      try {
        const regex = new RegExp(pattern);
        return regex.test(example);
      } catch {
        return false;
      }
    },

    // 在保存配置时验证所有正则
    async saveRegexConfig() {
      // 验证所有内置正则
      let hasErrors = false;

      ['flagRegex', 'sqlRegex', 'credentialRegex', 'keyRegex'].forEach(field => {
        if (!this.validateBuiltinRegex(field)) {
          hasErrors = true;
        }
      });

      // 验证所有自定义正则
      Object.keys(this.customRegexDisplay).forEach(name => {
        if (!this.validateSingleRegex(name)) {
          hasErrors = true;
        }
      });

      if (hasErrors) {
        ElMessage.error('存在正则表达式语法错误，请检查后重新保存');
        return;
      }

      try {
        // 更新自定义正则的配置
        const updatedCustomRegex = {};
        Object.keys(this.customRegexDisplay).forEach(key => {
          const regexData = this.customRegexDisplay[key];
          updatedCustomRegex[regexData.displayName || key] = regexData.pattern;

          // 保存后折叠显示
          this.customRegexDisplay[key].expanded = false;
        });
        this.regexConfig.customRegex = updatedCustomRegex;

        const result = await UpdateRegexConfig(JSON.stringify(this.regexConfig));

        ElMessage.success('正则表达式配置保存成功！');
        this.addLog(result, 'success');

        // 更新前端当前配置显示
        this.loadRegexConfig();

      } catch (error) {
        const errorMsg = `保存正则配置失败: ${error}`;
        this.addLog(errorMsg, 'error');
        ElMessage.error(errorMsg);
      }
    },

    async resetRegexConfig() {
      try {
        const result = await ResetRegexConfig();

        // 重新加载配置
        const configJson = await GetRegexConfig();
        const config = JSON.parse(configJson);
        this.regexConfig = {
          flagRegex: config.flagRegex || '',
          sqlRegex: config.sqlRegex || '',
          credentialRegex: config.credentialRegex || '',
          keyRegex: config.keyRegex || '',
          customRegex: config.customRegex || {}
        };

        // 重新初始化自定义正则名称映射
        this.customRegexNames = {};
        Object.keys(this.regexConfig.customRegex).forEach(key => {
          this.customRegexNames[key] = key;
        });

        ElMessage.success('正则表达式配置已重置为默认值');
        this.addLog(result, 'success');
      } catch (error) {
        const errorMsg = `重置正则配置失败: ${error}`;
        this.addLog(errorMsg, 'error');
        ElMessage.error(errorMsg);
      }
    },

    // 测试正则配置
    testRegexConfig() {
      const issues = [];

      // 验证内置正则
      ['flagRegex', 'sqlRegex', 'credentialRegex', 'keyRegex'].forEach(field => {
        if (!this.regexConfig[field]?.trim()) {
          issues.push(`${this.getBuiltinFieldLabel(field)}不能为空`);
        } else if (this.hasBuiltinError(field)) {
          issues.push(`${this.getBuiltinFieldLabel(field)}语法错误: ${this.getBuiltinError(field)}`);
        }
      });

      // 测试自定义正则
      Object.entries(this.customRegexDisplay).forEach(([name, regexData]) => {
        if (regexData.pattern?.trim()) {
          if (this.hasRegexError(name)) {
            issues.push(`自定义正则 "${regexData.displayName || name}" 语法错误: ${this.getRegexError(name)}`);
          }
        }
      });

      if (issues.length === 0) {
        ElMessage.success('所有正则表达式语法验证通过');
        this.addLog('✅ 所有正则表达式语法验证通过', 'success');
      } else {
        issues.forEach(issue => {
          ElMessage.warning(issue);
          this.addLog(`⚠️ ${issue}`, 'warning');
        });
      }
    },

    // 辅助方法：截断显示长正则
    truncateRegex(regex) {
      if (!regex) return '未设置';
      if (regex.length > 30) {
        return regex.substring(0, 30) + '...';
      }
      return regex;
    },

    // 获取内置正则字段的标签
    getBuiltinFieldLabel(field) {
      const labels = {
        flagRegex: 'Flag检测正则',
        sqlRegex: 'SQL检测正则',
        credentialRegex: '凭据检测正则',
        keyRegex: '密钥检测正则'
      };
      return labels[field] || field;
    },
  }
}
</script>

<style scoped>
/* 优化后的样式 */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

#app {
  flex: 1;
  width: 100%;
  max-width: 100%;
  height: 100%;
  min-height: 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: white;
  overflow: hidden;
}

/* 头部样式优化 */
.app-header {
  background: white;
  border-bottom: 1px solid #e1e5e9;
  padding: 0;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  flex-shrink: 0;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  min-width: 0;
}

.title-section {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.title-section h1 {
  color: #2c3e50;
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
}

.status-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  min-width: 0;
  flex-shrink: 1;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}

.status {
  padding: 0.5rem 1rem;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-ready {
  background: #d4edda;
  color: #155724;
}

.status-ready .status-indicator {
  background: #28a745;
}

.status-checking {
  background: #fff3cd;
  color: #856404;
}

.status-checking .status-indicator {
  background: #ffc107;
}

.status-working {
  background: #cce7ff;
  color: #004085;
}

.status-working .status-indicator {
  background: #007bff;
}

.status-error {
  background: #f8d7da;
  color: #721c24;
}

.status-error .status-indicator {
  background: #dc3545;
}

.stats {
  display: flex;
  gap: 1rem;
}

.stat-item {
  font-size: 0.875rem;
  color: #6c757d;
  background: #f8f9fa;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
}

/* 主布局 */
.main-layout {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

/* 左侧面板优化 */
.left-panel {
  flex: 0 0 260px;
  width: 260px;
  max-width: 260px;
  min-width: 0;
  background: #f8f9fa;
  border-right: 1px solid #e1e5e9;
  padding: 0.75rem;
  overflow-y: auto;
  overflow-x: hidden;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.panel-section {
  background: white;
  border: 1px solid #e1e5e9;
  border-radius: 8px;
  padding: 0.75rem;
}

.pcap-app {
  --wails-drop-target: drop;
}

.panel-section h3 {
  color: #2c3e50;
  margin-bottom: 0.75rem;
  font-size: 0.9375rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

/* 文件选择器优化 */
.file-dropzone {
  border: 2px dashed #dee2e6;
  border-radius: 8px;
  padding: 1rem 0.75rem;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s ease;
  background: #fafbfc;
}

.file-dropzone:hover {
  border-color: #007bff;
  background: #f0f7ff;
}

.dropzone-content {
  color: #6c757d;
}

.dropzone-icon {
  font-size: 2rem;
  margin-bottom: 0.5rem;
  display: block;
}

.file-path-input {
  margin-top: 0.75rem;
  width: 100%;
  padding: 0.6rem 0.8rem;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  box-sizing: border-box;
}

.file-info {
  margin-top: 1rem;
  padding: 0.75rem;
  background: #e7f3ff;
  border-radius: 6px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.file-info-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex: 1;
  min-width: 0;
}

.file-name {
  font-weight: 500;
  color: #2c3e50;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-icon {
  flex-shrink: 0;
}

.btn-clear {
  background: none;
  border: none;
  font-size: 1.25rem;
  color: #adb5bd;
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 4px;
}

.btn-clear:hover {
  color: #dc3545;
  background: rgba(220, 53, 69, 0.1);
}

/* 配置组优化 */
.config-group {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.input-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.input-group label {
  font-weight: 500;
  color: #2c3e50;
  font-size: 0.875rem;
}

.input-group input {
  padding: 0.75rem;
  border: 1px solid #e1e5e9;
  border-radius: 6px;
  font-size: 0.875rem;
  transition: all 0.3s ease;
  background: white;
}

.input-group input:focus {
  outline: none;
  border-color: #007bff;
  box-shadow: 0 0 0 2px rgba(0,123,255,0.25);
}

.input-hint {
  font-size: 0.75rem;
  color: #adb5bd;
  line-height: 1.4;
}

/* 开关组优化 */
.switch-grid {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.switch-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.switch {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  cursor: pointer;
  font-weight: 500;
  color: #2c3e50;
}

.switch-label {
  font-size: 0.875rem;
}

.switch-description {
  font-size: 0.75rem;
  color: #adb5bd;
  margin-left: 2.75rem;
  line-height: 1.4;
}

.slider {
  width: 44px;
  height: 24px;
  background: #c4d5e1;
  border-radius: 12px;
  position: relative;
  transition: all 0.3s ease;
  flex-shrink: 0;
}

.slider:before {
  content: '';
  position: absolute;
  width: 20px;
  height: 20px;
  background: white;
  border-radius: 50%;
  top: 2px;
  left: 2px;
  transition: all 0.3s ease;
  box-shadow: 0 1px 3px rgba(0,0,0,0.2);
}

input:checked + .slider {
  background: #007bff;
}

input:checked + .slider:before {
  transform: translateX(20px);
}

/* 操作按钮优化 */
.action-grid {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

/* 按钮样式优化 */
.btn-primary, .btn-secondary, .btn-outline, .btn-small, .btn-tiny {
  padding: 0.5rem 0.75rem;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 0.8125rem;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  text-decoration: none;
}

.btn-primary {
  background: #007bff;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #0056b3;
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0,123,255,0.3);
}

.btn-secondary {
  background: #6c757d;
  color: white;
}

.btn-secondary:hover:not(:disabled) {
  background: #545b62;
  transform: translateY(-1px);
}

.btn-outline {
  background: transparent;
  border: 1px solid #e1e5e9;
  color: #6c757d;
}

.btn-outline:hover:not(:disabled) {
  background: #f8f9fa;
  border-color: #007bff;
  color: #007bff;
}

.btn-small {
  padding: 0.375rem 0.625rem;
  font-size: 0.75rem;
}

.btn-tiny {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  background: transparent;
  border: 1px solid #e1e5e9;
  color: #6c757d;
}

.btn-tiny:hover {
  background: #f8f9fa;
}

button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none !important;
}

.btn-icon {
  font-size: 0.875rem;
}

/* 右侧面板优化 */
.right-panel {
  flex: 1;
  min-width: 0;
  background: white;
  padding: 0.5rem;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.results-container {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: white;
  border: 1px solid #e1e5e9;
  border-radius: 8px;
  overflow: hidden;
}

/* 标签页头部优化 */
.tabs-header {
  display: flex;
  flex-wrap: wrap;
  background: #f8f9fa;
  border-bottom: 1px solid #e1e5e9;
  padding: 0 0.5rem;
  min-width: 0;
}

.tab-btn {
  padding: 0.625rem 0.875rem;
  border: none;
  background: none;
  cursor: pointer;
  font-weight: 500;
  font-size: 0.8125rem;
  color: #6c757d;
  border-bottom: 2px solid transparent;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 0.375rem;
  position: relative;
  flex-shrink: 0;
}

.tab-btn:hover {
  color: #007bff;
  background: rgba(0,123,255,0.05);
}

.tab-btn.active {
  color: #007bff;
  border-bottom-color: #007bff;
  background: white;
}

.tab-btn.has-data:not(.active) {
  color: #2c3e50;
  font-weight: 600;
}

.tab-badge {
  background: #007bff;
  color: white;
  border-radius: 10px;
  padding: 0.125rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  min-width: 1.5rem;
  text-align: center;
}

.tab-icon {
  font-size: 1rem;
}

/* 分类统计样式 */
.panel-title-with-stats {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  min-width: 0;
}

.type-stats {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.type-stat {
  font-size: 0.75rem;
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
  background: #f8f9fa;
  color: #6c757d;
}

.type-flag {
  background: #28a745;
  color: white;
}

.type-sql {
  background: #ffc107;
  color: white;
}

.type-credential {
  background: #dc3545;
  color: white;
}

.type-key {
  background: #6f42c1;
  color: white;
}

/* 搜索框样式 */
.search-box {
  position: relative;
  display: flex;
  align-items: center;
}

.search-input {
  padding: 0.5rem 2rem 0.5rem 0.75rem;
  border: 1px solid #e1e5e9;
  border-radius: 6px;
  font-size: 0.875rem;
  width: 200px;
  transition: all 0.3s ease;
}

.search-input:focus {
  outline: none;
  border-color: #007bff;
  box-shadow: 0 0 0 2px rgba(0,123,255,0.25);
}

.search-clear {
  position: absolute;
  right: 0.5rem;
  cursor: pointer;
  color: #adb5bd;
  font-size: 1.25rem;
  line-height: 1;
}

.search-clear:hover {
  color: #dc3545;
}

.search-info {
  text-align: center;
  padding: 1rem;
  color: #6c757d;
  font-size: 0.875rem;
  background: #f8f9fa;
  border-radius: 6px;
  margin-top: 1rem;
}

/* 日志过滤控件 */
.log-controls {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-right: 0.5rem;
}

.log-filter {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.6875rem;
  color: #6c757d;
  cursor: pointer;
  white-space: nowrap;
}

.log-filter input {
  margin: 0;
}

/* 实时更新按钮样式 */
.btn-primary {
  background: #007bff;
  color: white;
}

/* 分析中的指示器 */
.analyzing-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  color: #6c757d;
  gap: 1rem;
  background: #f8f9fa;
  border-radius: 8px;
  margin-top: 1rem;
}

.loading-spinner {
  width: 1.5rem;
  height: 1.5rem;
  border: 2px solid #f3f3f3;
  border-top: 2px solid #007bff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 文件哈希显示 */
.file-hash {
  font-size: 0.75rem;
  color: #adb5bd;
  margin-top: 0.25rem;
}

.hash-label {
  font-weight: 500;
  margin-right: 0.25rem;
}

.hash-value {
  font-family: 'Courier New', monospace;
  background: #f8f9fa;
  padding: 0.125rem 0.25rem;
  border-radius: 2px;
  word-break: break-all;
}

/* 处理统计样式 */
.processing-stats {
  font-size: 0.875rem;
  color: #6c757d;
  margin-left: 1rem;
  background: rgba(0, 123, 255, 0.1);
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
}

.btn-warning {
  background: #ffc107;
  color: #212529;
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 0.875rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-warning:hover:not(:disabled) {
  background: #e0a800;
  transform: translateY(-1px);
}

/* 标签页内容优化 */
.tabs-content {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.tab-panel {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding: 0.625rem 0.75rem;
  border-bottom: 1px solid #e1e5e9;
  background: #f8f9fa;
  min-width: 0;
}

.panel-header h4 {
  color: #2c3e50;
  font-size: 0.875rem;
  font-weight: 600;
  margin: 0;
  flex-shrink: 0;
}

.panel-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
  min-width: 0;
  align-items: center;
}

/* 空状态优化 */
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 3rem 2rem;
  color: #adb5bd;
  text-align: center;
}

.empty-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
  opacity: 0.5;
}

.empty-state p {
  font-size: 1.125rem;
  margin-bottom: 0.5rem;
  color: #6c757d;
}

.empty-state small {
  font-size: 0.875rem;
}

/* 结果列表优化 */
.results-list {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.result-card {
  background: white;
  border: 1px solid #e1e5e9;
  border-radius: 8px;
  padding: 1rem;
  transition: all 0.3s ease;
  position: relative;
}

.result-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

/* 添加高亮样式 */
.result-card.highlighted {
  animation: highlightPulse 2s ease-in-out;
  border: 2px solid #ffeb3b;
  box-shadow: 0 0 20px rgba(255, 235, 59, 0.5);
}

@keyframes highlightPulse {
  0%, 100% {
    box-shadow: 0 0 0 rgba(255, 235, 59, 0.5);
  }
  50% {
    box-shadow: 0 0 20px rgba(255, 235, 59, 0.8);
  }
}

/* 确保文件相关的结果显示正确的颜色 */
.result-file_flag {
  border-left: 4px solid #28a745;
}

.result-file_sql {
  border-left: 4px solid #ffc107;
}

.result-file_credential {
  border-left: 4px solid #dc3545;
}

.result-file_key {
  border-left: 4px solid #6f42c1;
}

/* 文件结果显示样式调整 */
.result-card .file-info {
  font-size: 0.875rem;
  color: #6c757d;
  margin-bottom: 0.5rem;
}

.result-card .file-name {
  font-weight: 600;
  color: #007bff;
}

.result-card .file-protocol {
  background: #f8f9fa;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  margin-left: 0.5rem;
}

.result-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 0.75rem;
  flex-wrap: wrap;
}

.result-type-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
  color: white;
}

.packet-info {
  font-size: 0.875rem;
  color: #6c757d;
  background: #f8f9fa;
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
}

.result-time {
  font-size: 0.75rem;
  color: #adb5bd;
  margin-left: auto;
}

.result-content {
  margin-bottom: 0.75rem;
}

.result-content pre {
  background: #519eec;
  padding: 0.75rem;
  border-radius: 6px;
  border: 1px solid #e1e5e9;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  line-height: 1.4;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}

.result-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

/* 结果类型颜色 */
.result-flag {
  border-left: 4px solid #28a745;
}

.result-flag .result-type-badge {
  background: #28a745;
}

.result-sql {
  border-left: 4px solid #ffc107;
}

.result-sql .result-type-badge {
  background: #ffc107;
}

.result-credential {
  border-left: 4px solid #dc3545;
}

.result-credential .result-type-badge {
  background: #dc3545;
}

.result-key {
  border-left: 4px solid #6f42c1;
}

.result-key .result-type-badge {
  background: #6f42c1;
}

/* 文件网格优化 */
.files-grid {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1rem;
  align-content: start;
}

.file-card {
  background: white;
  border: 1px solid #e1e5e9;
  border-radius: 8px;
  padding: 1.25rem;
  display: flex;
  gap: 1rem;
  transition: all 0.3s ease;
}

.file-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.file-icon {
  font-size: 2rem;
  flex-shrink: 0;
}

.file-info {
  flex: 1;
  min-width: 0;
}

.file-name {
  margin: 0 0 0.5rem 0;
  color: #2c3e50;
  font-size: 0.875rem;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-meta {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
  flex-wrap: wrap;
}

.file-type, .file-size, .file-protocol {
  font-size: 0.75rem;
  color: #6c757d;
  background: #f8f9fa;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.file-path {
  font-size: 0.75rem;
  color: #adb5bd;
  margin: 0 0 0.5rem 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-hash {
  font-size: 0.75rem;
  color: #adb5bd;
}

.hash-label {
  font-weight: 500;
  margin-right: 0.25rem;
}

.hash-value {
  font-family: 'Courier New', monospace;
  background: #f8f9fa;
  padding: 0.125rem 0.25rem;
  border-radius: 2px;
}

.file-actions {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  flex-shrink: 0;
}

/* 处理统计样式 */
.processing-stats {
  font-size: 0.875rem;
  color: #6c757d;
  margin-left: 1rem;
  background: rgba(0, 123, 255, 0.1);
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
}

.analyzing-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  color: #6c757d;
  gap: 1rem;
}

.btn-warning {
  background: #ffc107;
  color: #212529;
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 0.875rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-warning:hover:not(:disabled) {
  background: #e0a800;
  transform: translateY(-1px);
}

/* 日志容器优化 */
.logs-container {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.logs-content {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
  background: #221f1f;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  line-height: 1.4;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.log-entry {
  display: flex;
  gap: 1rem;
  padding: 0.25rem 0;
}

.log-time {
  color: #26679f;
  flex-shrink: 0;
  min-width: 80px;
}

.log-message {
  flex: 1;
  word-break: break-all;
}

.log-info .log-message {
  color: #00ff00;
}

.log-match .log-message {
  color: #ff0000;
}

.log-success .log-message {
  color: #28a745;
}

.log-warning .log-message {
  color: #ffc107;
}

.log-error .log-message {
  color: #dc3545;
}

.log-progress .log-message {
  color: #dda65d;
}


.log-file .log-message {
  color: #5f8fe4;
}

/* 模态框优化 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
  padding: 2rem;
}

.modal-content {
  background: white;
  border-radius: 12px;
  width: 90%;
  max-width: 900px;
  max-height: 80vh;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0,0,0,0.3);
  display: flex;
  flex-direction: column;
}

.modal-header {
  padding: 1.5rem;
  border-bottom: 1px solid #e1e5e9;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #f8f9fa;
}

.modal-header h3 {
  color: #2c3e50;
  margin: 0;
  font-size: 1.25rem;
}

.modal-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #adb5bd;
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
}

.close-btn:hover {
  color: #dc3545;
  background: rgba(220, 53, 69, 0.1);
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 1.5rem;
  background: white;
}

.file-preview {
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  line-height: 1.4;
  color: #333;
  background: #f8f9fa;
  padding: 1rem;
  border-radius: 6px;
  border: 1px solid #e1e5e9;
  max-height: 400px;
  overflow-y: auto;
}

.preview-error, .preview-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 2rem;
  color: #adb5bd;
  text-align: center;
}

.error-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
  opacity: 0.5;
}

.loading-spinner {
  width: 2rem;
  height: 2rem;
  border: 2px solid #f3f3f3;
  border-top: 2px solid #007bff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 进度条 */
.progress-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0,0,0,0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 2000;
}

.progress-content {
  background: white;
  padding: 2rem;
  border-radius: 12px;
  box-shadow: 0 10px 30px rgba(0,0,0,0.3);
  min-width: 300px;
  text-align: center;
}

.progress-bar {
  width: 100%;
  height: 8px;
  background: #e9ecef;
  border-radius: 4px;
  overflow: hidden;
  margin-bottom: 1rem;
}

.progress-fill {
  height: 100%;
  background: #007bff;
  transition: width 0.3s ease;
  border-radius: 4px;
}

.progress-text {
  color: #2c3e50;
  font-weight: 500;
  margin: 0;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .main-layout {
    flex-direction: column;
  }

  .left-panel {
    flex: 0 0 auto;
    width: 100%;
    max-width: 100%;
    max-height: 40vh;
  }

  .files-grid {
    grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  }
}

@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    gap: 1rem;
    align-items: flex-start;
  }

  .status-bar {
    width: 100%;
    justify-content: space-between;
  }

  .tabs-header {
    flex-wrap: wrap;
  }

  .tab-btn {
    flex: 1;
    min-width: 120px;
  }

  .files-grid {
    grid-template-columns: 1fr;
  }

  .modal-overlay {
    padding: 1rem;
  }
}
/* 添加分页相关样式 */
.pagination-controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-left: 1rem;
}

.pagination-info {
  font-size: 0.75rem;
  color: #6c757d;
  margin-right: 0.5rem;
}

.page-size-select {
  padding: 0.25rem 0.5rem;
  border: 1px solid #e1e5e9;
  border-radius: 4px;
  font-size: 0.75rem;
  background: white;
}

.pagination-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-top: 1px solid #e1e5e9;
  background: #f8f9fa;
}

.pagination-stats {
  font-size: 0.875rem;
  color: #6c757d;
}

.pagination-navigation {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.page-numbers {
  display: flex;
  gap: 0.25rem;
}

.page-btn {
  padding: 0.25rem 0.5rem;
  border: 1px solid #e1e5e9;
  background: white;
  border-radius: 4px;
  font-size: 0.75rem;
  min-width: 2rem;
  cursor: pointer;
  transition: all 0.3s ease;
}

.page-btn:hover:not(.active) {
  background: #f8f9fa;
  border-color: #007bff;
}

.page-btn.active {
  background: #007bff;
  color: white;
  border-color: #007bff;
}

.page-ellipsis {
  padding: 0.25rem 0.5rem;
  color: #adb5bd;
  font-size: 0.75rem;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .pagination-footer {
    flex-direction: column;
    gap: 1rem;
    align-items: stretch;
  }

  .pagination-navigation {
    justify-content: center;
  }

  .pagination-controls {
    flex-wrap: wrap;
    justify-content: center;
  }
}

/* 添加正则配置相关的样式 */
.large-modal {
  width: 90%;
  max-width: 800px;
  max-height: 90vh;
}

.regex-config-container {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.regex-section {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 1.5rem;
}

.regex-section h4 {
  color: #2c3e50;
  margin-bottom: 1rem;
  font-size: 1.1rem;
  font-weight: 600;
}

.regex-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
}

.regex-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.regex-item label {
  font-weight: 500;
  color: #2c3e50;
  font-size: 0.875rem;
}

.regex-item textarea {
  padding: 0.75rem;
  border: 1px solid #e1e5e9;
  border-radius: 6px;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  resize: vertical;
  min-height: 60px;
  background: white;
}

.regex-item textarea:focus {
  outline: none;
  border-color: #007bff;
  box-shadow: 0 0 0 2px rgba(0,123,255,0.25);
}

.regex-hint {
  font-size: 0.75rem;
  color: #adb5bd;
  line-height: 1.4;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.custom-regex-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.custom-regex-item {
  background: white;
  border: 1px solid #e1e5e9;
  border-radius: 6px;
  padding: 1rem;
}

.custom-regex-header {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 0.5rem;
}

.regex-name-input {
  flex: 1;
  padding: 0.5rem;
  border: 1px solid #e1e5e9;
  border-radius: 4px;
  font-size: 0.875rem;
}

.regex-name-input:focus {
  outline: none;
  border-color: #007bff;
}

.btn-danger {
  background: #dc3545;
  color: white;
  border: none;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  cursor: pointer;
}

.btn-danger:hover {
  background: #c82333;
}

.empty-custom-regex {
  text-align: center;
  padding: 2rem;
  color: #adb5bd;
}

.empty-custom-regex p {
  margin-bottom: 0.5rem;
}

.config-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #e1e5e9;
}

.action-spacer {
  flex: 1;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .large-modal {
    width: 95%;
    margin: 1rem;
  }

  .regex-grid {
    grid-template-columns: 1fr;
  }

  .config-actions {
    flex-direction: column;
  }

  .action-spacer {
    display: none;
  }
}

/* 添加配置预览样式 */
.config-preview {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.config-item {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.5rem;
  background: #f8f9fa;
  border-radius: 4px;
  font-size: 0.75rem;
}

.config-label {
  color: #6c757d;
  font-weight: 500;
  flex-shrink: 0;
  min-width: 60px;
}

.config-value {
  color: #2c3e50;
  font-family: 'Courier New', monospace;
  word-break: break-all;
  flex: 1;
  text-align: right;
  opacity: 0.8;
}

.config-edit-btn {
  align-self: flex-end;
  margin-top: 0.5rem;
}
/* 自定义正则容器 */
.custom-regex-container {
  margin-top: 1rem;
}

.section-header-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.custom-regex-count {
  font-size: 0.875rem;
  color: #adb5bd;
  background: #f8f9fa;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
}

.add-regex-btn {
  background: linear-gradient(135deg, #007bff, #4dabf7);
  border: none;
  box-shadow: 0 2px 8px rgba(0, 123, 255, 0.3);
  transition: all 0.3s ease;
}

.add-regex-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 123, 255, 0.4);
}

/* 空状态样式 */
.empty-custom-regex {
  text-align: center;
  padding: 3rem 2rem;
  background: linear-gradient(135deg, #f8f9fa, #e9ecef);
  border: 2px dashed #dee2e6;
  border-radius: 12px;
  color: #adb5bd;
}

.empty-custom-regex .empty-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
  opacity: 0.5;
}

.empty-custom-regex h5 {
  color: #6c757d;
  margin-bottom: 0.5rem;
  font-weight: 600;
}

.empty-custom-regex p {
  margin-bottom: 0.5rem;
  font-size: 0.95rem;
}

/* 自定义正则项样式 */
.custom-regex-item {
  background: white;
  border: 2px solid #f1f3f4;
  border-radius: 12px;
  padding: 1.5rem;
  margin-bottom: 1.5rem;
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.custom-regex-item::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 4px;
  height: 100%;
  background: linear-gradient(to bottom, #007bff, #4dabf7);
  opacity: 0;
  transition: opacity 0.3s ease;
}

.custom-regex-item:hover::before {
  opacity: 1;
}

.custom-regex-item:hover {
  border-color: #007bff;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.regex-item-error {
  border-color: #dc3545;
  background: #fff5f5;
}

.regex-item-error::before {
  background: linear-gradient(to bottom, #dc3545, #e03131);
  opacity: 1;
}

/* 正则头部样式 */
.custom-regex-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.25rem;
  gap: 1rem;
}

.regex-title-section {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex: 1;
}

.regex-drag-handle {
  color: #adb5bd;
  cursor: grab;
  font-size: 1.2rem;
  padding: 0.25rem;
  border-radius: 4px;
  transition: all 0.2s ease;
  user-select: none;
}

.regex-drag-handle:hover {
  background: #f8f9fa;
  color: #007bff;
}

.regex-name-input {
  flex: 1;
  padding: 0.75rem 1rem;
  border: 2px solid #e9ecef;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: 600;
  background: white;
  transition: all 0.3s ease;
  min-width: 200px;
}

.regex-name-input:focus {
  outline: none;
  border-color: #007bff;
  box-shadow: 0 0 0 3px rgba(0, 123, 255, 0.1);
  background: #fafbfc;
}

.regex-error-indicator {
  color: #dc3545;
  font-size: 1.2rem;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.regex-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.btn-test {
  background: linear-gradient(135deg, #20c997, #12b886);
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 6px;
  font-size: 0.75rem;
  transition: all 0.3s ease;
}

.btn-test:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(32, 201, 151, 0.3);
}

.btn-test:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

.btn-danger {
  background: linear-gradient(135deg, #dc3545, #e03131);
  color: white;
  border: none;
  padding: 0.5rem;
  border-radius: 6px;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s ease;
}

.btn-danger:hover {
  transform: scale(1.05);
  box-shadow: 0 4px 8px rgba(220, 53, 69, 0.3);
}

/* 正则内容区域 */
.regex-content {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.regex-input-container {
  position: relative;
}

.regex-input-label {
  display: block;
  font-weight: 600;
  color: #2c3e50;
  margin-bottom: 0.5rem;
  font-size: 0.9rem;
}

.regex-pattern-input {
  width: 100%;
  padding: 1rem;
  border: 2px solid #e9ecef;
  border-radius: 8px;
  font-family: 'Courier New', monospace;
  font-size: 0.9rem;
  resize: vertical;
  min-height: 80px;
  background: #fafbfc;
  transition: all 0.3s ease;
  line-height: 1.5;
}

.regex-pattern-input:focus {
  outline: none;
  border-color: #007bff;
  box-shadow: 0 0 0 3px rgba(0, 123, 255, 0.1);
  background: white;
}

.input-error {
  border-color: #dc3545 !important;
  background: #fff5f5 !important;
}

.input-error:focus {
  box-shadow: 0 0 0 3px rgba(220, 53, 69, 0.1) !important;
}

.regex-stats {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 0.5rem;
  font-size: 0.75rem;
}

.regex-length {
  color: #adb5bd;
}

.regex-error-msg {
  color: #dc3545;
  font-weight: 500;
}

/* 测试用例样式 */
.regex-examples {
  border: 1px solid #e9ecef;
  border-radius: 8px;
  padding: 1rem;
  background: #f8f9fa;
}

.examples-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.examples-label {
  font-weight: 600;
  color: #2c3e50;
  font-size: 0.9rem;
}

.examples-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.example-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem;
  background: white;
  border: 1px solid #e9ecef;
  border-radius: 6px;
  transition: all 0.3s ease;
}

.example-item:hover {
  border-color: #007bff;
}

.example-matched {
  border-color: #20c997;
  background: #f0fdf4;
}

.example-input {
  flex: 1;
  padding: 0.5rem;
  border: 1px solid #e9ecef;
  border-radius: 4px;
  font-size: 0.875rem;
  background: transparent;
}

.example-input:focus {
  outline: none;
  border-color: #007bff;
}

.example-status {
  min-width: 80px;
  text-align: center;
}

.match-success {
  color: #20c997;
  font-weight: 600;
  font-size: 0.75rem;
}

.match-fail {
  color: #adb5bd;
  font-size: 0.75rem;
}

.btn-clear-example {
  background: none;
  border: none;
  color: #adb5bd;
  font-size: 1.2rem;
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 4px;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.btn-clear-example:hover {
  background: #dc3545;
  color: white;
}

.no-examples {
  text-align: center;
  padding: 1rem;
  color: #adb5bd;
  font-style: italic;
}

/* 提示信息样式 */
.regex-hint {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.75rem;
  background: linear-gradient(135deg, #e7f5ff, #d0ebff);
  border-radius: 6px;
  margin-top: 1rem;
  font-size: 0.875rem;
  color: #6c757d;
  border-left: 4px solid #007bff;
}

.hint-icon {
  font-size: 1rem;
  flex-shrink: 0;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .custom-regex-header {
    flex-direction: column;
    align-items: stretch;
  }

  .regex-title-section {
    margin-bottom: 1rem;
  }

  .regex-actions {
    justify-content: flex-end;
  }

  .example-item {
    flex-direction: column;
    align-items: stretch;
  }

  .example-status {
    text-align: left;
  }
}


/* 内置正则样式 */
.builtin-regex-container {
  margin-top: 1rem;
}

.builtin-regex-list {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.builtin-regex-item {
  background: white;
  border: 2px solid #f1f3f4;
  border-radius: 12px;
  padding: 1.5rem;
  transition: all 0.3s ease;
}

.builtin-regex-item:hover {
  border-color: #007bff;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.regex-type-icon {
  font-size: 1.2rem;
  margin-right: 0.5rem;
}

/* 自定义正则折叠样式 */
.regex-name-display {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.regex-name-text {
  font-weight: 600;
  color: #2c3e50;
  font-size: 1rem;
}

.regex-preview {
  color: #adb5bd;
  font-size: 0.875rem;
  font-family: 'Courier New', monospace;
  background: #f8f9fa;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  word-break: break-all;
}

.regex-expand-icon {
  color: #adb5bd;
  font-size: 0.875rem;
  transition: transform 0.3s ease;
}

.regex-item-expanded .regex-expand-icon {
  transform: rotate(0deg);
}

.regex-item:not(.regex-item-expanded) .regex-expand-icon {
  transform: rotate(-90deg);
}

.regex-hint-collapsed {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem;
  background: linear-gradient(135deg, #f8f9fa, #e9ecef);
  border-radius: 6px;
  margin-top: 0.75rem;
  font-size: 0.875rem;
  color: #6c757d;
  border-left: 4px solid #007bff;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .builtin-regex-item {
    padding: 1rem;
  }

  .regex-name-display {
    min-width: 0;
  }
}

/* 内置正则错误样式 */
.builtin-regex-item .input-error {
  border-color: #dc3545 !important;
  background: #fff5f5 !important;
}

.builtin-regex-item .input-error:focus {
  box-shadow: 0 0 0 3px rgba(220, 53, 69, 0.1) !important;
}



/* 动画效果 */
.custom-regex-item {
  animation: slideInUp 0.3s ease;
}

@keyframes slideInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 滚动条美化 */
.custom-regex-container::-webkit-scrollbar {
  width: 6px;
}

.custom-regex-container::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.custom-regex-container::-webkit-scrollbar-thumb {
  background: #007bff;
  border-radius: 3px;
}

.custom-regex-container::-webkit-scrollbar-thumb:hover {
  background: #0056b3;
}

/* 文件显示美化样式 */
.file-stats {
  display: flex;
  gap: 1rem;
  font-size: 0.875rem;
}

.file-stats .stat-item {
  color: #6c757d;
  background: #f8f9fa;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
}

.view-controls {
  margin-right: 1rem;
}

/* 列表视图样式 */
.files-table-container {
  flex: 1;
  overflow: auto;
  padding: 0;
}

.files-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.files-table th {
  background: #f8f9fa;
  padding: 1rem;
  text-align: left;
  font-weight: 600;
  color: #2c3e50;
  border-bottom: 2px solid #e1e5e9;
  cursor: pointer;
  transition: background-color 0.3s ease;
  position: sticky;
  top: 0;
  z-index: 10;
}

.files-table th:hover {
  background: #e9ecef;
}

.sort-indicator {
  margin-left: 0.5rem;
  font-weight: bold;
  color: #007bff;
}

.files-table td {
  padding: 1rem;
  border-bottom: 1px solid #e1e5e9;
  vertical-align: middle;
}

.file-row {
  transition: background-color 0.3s ease;
}

.file-row:hover {
  background: #f8f9fa;
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.file-icon-small {
  font-size: 1.25rem;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  flex-shrink: 0;
}

.file-name-text {
  font-weight: 500;
  color: #2c3e50;
  word-break: break-word;
}

.file-type-badge, .protocol-badge {
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
}

.file-type-badge {
  background: #e7f3ff;
  color: #007bff;
}

.protocol-badge {
  background: #d4edda;
  color: #155724;
}

.table-actions {
  display: flex;
  gap: 0.25rem;
  justify-content: center;
}

/* 列宽调整 */
.col-name { width: 40%; }
.col-type { width: 15%; }
.col-size { width: 15%; }
.col-protocol { width: 15%; }
.col-actions { width: 15%; }

/* 文件图标类型颜色 */
.icon-image { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); }
.icon-pdf { background: linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%); }
.icon-archive { background: linear-gradient(135deg, #4ecdc4 0%, #44a08d 100%); }
.icon-text { background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%); }
.icon-web { background: linear-gradient(135deg, #ffd89b 0%, #19547b 100%); }
.icon-data { background: linear-gradient(135deg, #a3bded 0%, #6991c7 100%); }
.icon-executable { background: linear-gradient(135deg, #fad0c4 0%, #fad0c4 100%); }
.icon-audio { background: linear-gradient(135deg, #ffecd2 0%, #fcb69f 100%); }
.icon-video { background: linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%); }
.icon-default { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }

/* 响应式调整 */
@media (max-width: 1024px) {
  .files-grid {
    grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
    padding: 1rem;
    gap: 1rem;
  }

  .files-table {
    font-size: 0.8rem;
  }

  .files-table th,
  .files-table td {
    padding: 0.75rem 0.5rem;
  }
}

@media (max-width: 768px) {
  .files-grid {
    grid-template-columns: 1fr;
  }

  .panel-actions {
    flex-direction: column;
    gap: 1rem;
  }

  .view-controls {
    margin-right: 0;
    align-self: flex-start;
  }

  .files-table-container {
    overflow-x: auto;
  }

  .files-table {
    min-width: 600px;
  }
}

.total-size {
  color: #6c757d;
  font-size: 0.875rem;
  margin-left: 0.5rem;
}

/* 文件信息模态框样式 */
.file-info-modal {
  max-width: 600px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 1.5rem;
  border-bottom: 1px solid #e1e5e9;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.modal-title-section {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  flex: 1;
}

.file-info-icon {
  font-size: 2.5rem;
  width: 60px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 12px;
  backdrop-filter: blur(10px);
}

.modal-title h3 {
  margin: 0 0 0.25rem 0;
  color: white;
  font-size: 1.25rem;
}

.file-name-title {
  margin: 0;
  opacity: 0.9;
  font-size: 0.9rem;
  word-break: break-all;
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 0;
}

.file-info-content {
  padding: 1.5rem;
}

.info-section {
  margin-bottom: 2rem;
}

.section-title {
  color: #2c3e50;
  margin: 0 0 1rem 0;
  font-size: 1.1rem;
  font-weight: 600;
  padding-bottom: 0.5rem;
  border-bottom: 2px solid #e1e5e9;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e1e5e9;
  transition: all 0.3s ease;
}

.info-item:hover {
  border-color: #007bff;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.info-item.full-width {
  grid-column: 1 / -1;
}

.info-label {
  font-weight: 600;
  color: #6c757d;
  font-size: 0.875rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.info-value {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
}

.value-text {
  color: #2c3e50;
  font-weight: 500;
  word-break: break-all;
  flex: 1;
}

.path-text, .hash-text {
  font-family: 'Courier New', monospace;
  font-size: 0.8rem;
  background: rgba(0, 0, 0, 0.05);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.btn-copy {
  background: none;
  border: none;
  padding: 0.5rem;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.875rem;
  transition: all 0.3s ease;
  flex-shrink: 0;
}

.btn-copy:hover {
  background: #007bff;
  color: white;
  transform: scale(1.1);
}

.type-badge, .protocol-badge, .size-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
  flex-shrink: 0;
}

.type-badge {
  background: #e7f3ff;
  color: #007bff;
}

.protocol-badge {
  background: #d4edda;
  color: #155724;
}

.size-badge {
  background: #fff3cd;
  color: #856404;
}

.action-section {
  border-top: 2px solid #e1e5e9;
  padding-top: 1.5rem;
}

.action-buttons {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 0.75rem;
}

.btn-action {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border: 1px solid #e1e5e9;
  border-radius: 8px;
  background: white;
  color: #2c3e50;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 0.875rem;
  font-weight: 500;
}

.btn-action:hover {
  border-color: #007bff;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.btn-action.primary {
  background: #007bff;
  color: white;
  border-color: #007bff;
}

.btn-action.primary:hover {
  background: #0056b3;
}

.modal-footer {
  padding: 1rem 1.5rem;
  border-top: 1px solid #e1e5e9;
  display: flex;
  justify-content: flex-end;
  background: #f8f9fa;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .file-info-modal {
    margin: 1rem;
    max-height: 90vh;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }

  .modal-header {
    flex-direction: column;
    gap: 1rem;
  }

  .action-buttons {
    grid-template-columns: 1fr;
  }

  .file-info-icon {
    width: 50px;
    height: 50px;
    font-size: 2rem;
  }
}

/* 动画效果 */
.file-info-modal {
  animation: modalSlideIn 0.3s ease;
}

@keyframes modalSlideIn {
  from {
    opacity: 0;
    transform: scale(0.9) translateY(-20px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.info-item {
  animation: itemFadeIn 0.4s ease;
}

@keyframes itemFadeIn {
  from {
    opacity: 0;
    transform: translateX(-10px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

/* 交错动画延迟 */
.info-item:nth-child(1) { animation-delay: 0.1s; }
.info-item:nth-child(2) { animation-delay: 0.15s; }
.info-item:nth-child(3) { animation-delay: 0.2s; }
.info-item:nth-child(4) { animation-delay: 0.25s; }
.info-item:nth-child(5) { animation-delay: 0.3s; }



</style>