<template>
  <main>
    <el-button-group>
      <el-button type="primary" @click="getDbInfo" link>详情</el-button>
      <el-button type="primary" @click="openCommandDialog" link>执行命令</el-button>
    </el-button-group>

    <!-- 数据库详情对话框 -->
    <el-dialog
        v-model="dialogVisible"
        :title="`数据库详情(${ip})`"
        width="60%"
    >
      <el-table :data="info" border stripe style="width: 100%">
        <el-table-column type="index" width="65"/>
        <el-table-column prop="key" label="Key" />
        <el-table-column prop="value" label="Value" />
      </el-table>
    </el-dialog>

    <!-- 执行命令对话框 -->
    <el-dialog
        v-model="commandDialogVisible"
        title="执行Redis命令"
        width="70%"
    >
      <div class="command-container">
        <div class="command-input">
          <el-input
              v-model="command"
              placeholder="输入Redis命令，例如：GET key 或 HSET hash field value"
              @keyup.enter="executeCommand"
          >
            <template #prepend>
              <span style="color: #409EFF;">></span>
            </template>
            <template #append>
              <el-button @click="executeCommand" type="primary">执行</el-button>
            </template>
          </el-input>
        </div>

        <div class="common-commands" v-if="showCommonCommands">
          <el-divider content-position="left">常用命令</el-divider>
          <el-space wrap>
            <el-tag
                v-for="cmd in commonCommands"
                :key="cmd.command"
                type="info"
                style="cursor: pointer;"
                @click="fillCommand(cmd.command)"
            >
              {{ cmd.name }} ({{ cmd.command }})
            </el-tag>
          </el-space>
        </div>

        <div class="result-section" v-if="commandResult">
          <el-divider content-position="left">执行结果</el-divider>
          <div class="result-content">
            <el-alert
                :title="`执行${commandResult.success ? '成功' : '失败'} - 耗时: ${commandResult.duration}ms`"
                :type="commandResult.success ? 'success' : 'error'"
                :closable="false"
                :description="commandResult.error || ''"
            />

            <div
                class="result-data"
                v-if="commandResult.success && commandResult.data !== null && commandResult.data !== undefined"
            >
              <div class="result-header">
                <span class="result-type">数据类型: {{ getDataType(commandResult.data) }}</span>
                <span class="result-count" v-if="getDataCount(commandResult.data) > 0">
          数量: {{ getDataCount(commandResult.data) }}
        </span>
                <el-button
                    size="small"
                    link
                    @click="copyResult"
                    v-if="commandResult.data"
                    class="copy-btn"
                >
                  <el-icon><CopyDocument /></el-icon>
                  复制
                </el-button>
              </div>

              <div class="result-body">
                <!-- Redis INFO 命令结果美化 -->
                <div v-if="isRedisInfoResult(commandResult.data)" class="redis-info-result">
                  <div
                      v-for="(section, sectionName) in parseRedisInfo(commandResult.data)"
                      :key="sectionName"
                      class="info-section"
                  >
                    <h4 class="section-title">{{ sectionName }}</h4>
                    <div class="section-content">
                      <div
                          v-for="(value, key) in section"
                          :key="key"
                          class="info-item"
                      >
                        <span class="info-key">{{ key }}:</span>
                        <span class="info-value">{{ value }}</span>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 其他类型的结果 -->
                <div v-else>
                  <pre :class="getResultClass(commandResult.data)">{{ formatResult(commandResult.data) }}</pre>
                </div>
              </div>
            </div>

            <!-- 空结果提示 -->
            <div
                class="empty-result"
                v-else-if="commandResult.success && (commandResult.data === null || commandResult.data === undefined)"
            >
              <el-empty description="无返回数据" :image-size="80" />
            </div>
          </div>
        </div>

        <div class="command-history" v-if="commandHistory.length > 0">
          <el-divider content-position="left">历史命令</el-divider>
          <div class="history-list">
            <div
                v-for="(item, index) in commandHistory"
                :key="index"
                class="history-item"
                @click="fillCommand(item.command)"
            >
              <span class="command-text">{{ item.command }}</span>
              <span class="result-status" :class="item.success ? 'success' : 'error'">
                {{ item.success ? '✓' : '✗' }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </el-dialog>
  </main>
</template>

<script setup>
                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        
</script>

<style scoped>
.command-container {
  padding: 10px 0;
}

.command-input {
  margin-bottom: 20px;
}

.common-commands {
  margin-bottom: 20px;
}

.result-section {
  margin-top: 20px;
}

.result-content {
  margin-top: 10px;
}

.result-data {
  margin-top: 15px;
  border: 1px solid #e1e4e8;
  border-radius: 8px;
  overflow: hidden;
  background: #fafbfc;
}

.result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: #f6f8fa;
  border-bottom: 1px solid #e1e4e8;
  font-size: 12px;
  color: #586069;
}

.result-type {
  font-weight: 600;
  color: #24292e;
}

.result-count {
  margin-left: auto;
  margin-right: 10px;
  color: #0366d6;
}

.copy-btn {
  color: #0366d6;
  padding: 2px 6px;
}

.copy-btn:hover {
  background: #0366d6;
  color: white;
}

.result-body {
  padding: 0;
  max-height: 500px;
  overflow: auto;
}

/* Redis INFO 结果美化 */
.redis-info-result {
  padding: 16px;
}

.info-section {
  margin-bottom: 20px;
  border: 1px solid #e1e4e8;
  border-radius: 6px;
  overflow: hidden;
}

.section-title {
  margin: 0;
  padding: 8px 12px;
  background: #0366d6;
  color: white;
  font-size: 14px;
  font-weight: 600;
}

.section-content {
  padding: 0;
}

.info-item {
  display: flex;
  align-items: flex-start;
  padding: 8px 12px;
  border-bottom: 1px solid #f0f0f0;
  transition: background-color 0.2s;
}

.info-item:hover {
  background: #f8f9fa;
}

.info-item:last-child {
  border-bottom: none;
}

.info-key {
  flex: 0 0 200px;
  font-weight: 600;
  color: #24292e;
  font-size: 13px;
  word-break: break-word;
}

.info-value {
  flex: 1;
  color: #586069;
  font-size: 13px;
  word-break: break-word;
  padding-left: 8px;
}

/* 普通文本结果 */
.result-body pre {
  margin: 0;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.45;
  white-space: pre-wrap;
  word-wrap: break-word;
  background: transparent;
}

/* 不同类型的结果样式 */
.result-body pre.string {
  color: #032f62;
  background: #f1f8ff;
}

.result-body pre.array {
  color: #22863a;
}

.result-body pre.object {
  color: #6f42c1;
}

.result-body pre.number {
  color: #e36209;
}

.result-body pre.null {
  color: #6a737d;
  font-style: italic;
}

.result-body pre.error {
  color: #d73a49;
  background: #ffebee;
}

/* 滚动条样式 */
.result-body::-webkit-scrollbar {
  width: 8px;
}

.result-body::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 4px;
}

.result-body::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 4px;
}

.result-body::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

.empty-result {
  padding: 40px 0;
  text-align: center;
  color: #8c8c8c;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .result-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .result-count {
    margin-left: 0;
    margin-right: 0;
  }

  .info-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .info-key {
    flex: none;
    margin-bottom: 4px;
  }

  .info-value {
    padding-left: 0;
  }
}

.copy-btn {
  color: #0366d6;
  padding: 2px 6px;
}

.copy-btn:hover {
  background: #0366d6;
  color: white;
}

.empty-result {
  padding: 40px 0;
  text-align: center;
  color: #8c8c8c;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .result-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .result-count {
    margin-left: 0;
    margin-right: 0;
  }

  .result-body pre {
    padding: 12px;
    font-size: 12px;
  }
}

.command-history {
  margin-top: 20px;
}

.history-list {
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid #e9ecef;
  border-radius: 4px;
}

.history-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background-color 0.2s;
}

.history-item:hover {
  background-color: #f5f7fa;
}

.history-item:last-child {
  border-bottom: none;
}

.command-text {
  flex: 1;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  color: #606266;
}

.result-status {
  margin-left: 10px;
  font-weight: bold;
}

.result-status.success {
  color: #67c23a;
}

.result-status.error {
  color: #f56c6c;
}
</style>