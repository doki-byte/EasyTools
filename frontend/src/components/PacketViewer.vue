<template>
  <!-- 嵌入页面（便携发包）：与弹窗内布局一致 -->
  <div v-if="embedded" class="packet-viewer-embedded">
    <div class="packet-viewer-container packet-viewer-container--embedded">
      <div class="packet-viewer-toolbar">
        <div class="toolbar-left">
          <template v-if="showSendActions">
            <el-button
              v-if="!loading"
              type="primary"
              size="small"
              @click="handleSendClick"
            >
              发送请求
            </el-button>
            <el-button
              v-else
              type="danger"
              size="small"
              :loading="canceling"
              :disabled="canceling"
              @click="handleCancelClick"
            >
              取消请求
            </el-button>
            <el-checkbox v-model="forceHttps" size="small">
              强制HTTPS
            </el-checkbox>
          </template>
          <el-button 
            v-if="enableSave" 
            type="success" 
            size="small" 
            plain 
            @click="handleSaveClick"
          >
            导入并保存
          </el-button>
          <el-button 
            v-if="showSendActions"
            type="warning" 
            size="small" 
            plain 
            @click="handleUploadBypassClick"
            :disabled="loading"
          >
            文件上传Bypass
          </el-button>
        </div>
        <div class="toolbar-right">
          <el-button-group>
            <el-button
              size="small"
              @click="formatRequest"
              title="格式化请求包"
            >
              格式化请求
            </el-button>
            <el-button
              size="small"
              @click="formatResponse"
              title="格式化响应包"
            >
              格式化响应
            </el-button>
          </el-button-group>
        </div>
      </div>

      <div class="packet-viewer-content">
        <div class="packet-panel request-panel">
          <div class="panel-header">
            <div class="panel-title-row">
              <span class="panel-title">请求包</span>
              <el-button
                :icon="CopyDocument"
                size="small"
                text
                @click="copyRequest"
              >
                复制请求包
              </el-button>
            </div>
          </div>
          <div class="panel-content" ref="requestPanelRef">
            <VueMonacoEditor
              v-if="editable"
              ref="requestEditorRef"
              v-model:value="editableRequestPacket"
              :language="requestLanguage"
              theme="http-dark-theme"
              :height="panelEditorHeight"
              class="code-editor"
              :options="monacoOptions"
              placeholder="在此粘贴/编辑请求数据包"
            />
            <VueMonacoEditor
              v-else
              :value="displayRequestPacket"
              :language="requestLanguage"
              theme="http-dark-theme"
              :height="panelEditorHeight"
              class="code-editor"
              :options="previewMonacoOptions"
            />
          </div>
        </div>

        <div class="packet-panel response-panel">
          <div class="panel-header">
            <div class="panel-title-row">
              <span class="panel-title">响应包</span>
              <el-button
                text
                size="small"
                :disabled="!responsePreviewSrcdoc"
                @click="openResponseHtmlPreview"
              >
                网页预览
              </el-button>
              <el-button
                :icon="CopyDocument"
                size="small"
                text
                @click="copyResponse"
              >
                复制响应包
              </el-button>
            </div>
          </div>
          <div class="panel-content" ref="responsePanelRef">
            <VueMonacoEditor
              v-if="editable"
              ref="responseEditorRef"
              v-model:value="editableResponsePacket"
              :language="responseLanguage"
              theme="http-dark-theme"
              :height="panelEditorHeight"
              class="code-editor"
              :options="monacoOptions"
              placeholder="可选：在此粘贴响应数据包"
            />
            <VueMonacoEditor
              v-else
              :value="displayResponsePacket"
              :language="responseLanguage"
              theme="http-dark-theme"
              :height="panelEditorHeight"
              class="code-editor"
              :options="previewMonacoOptions"
            />
          </div>
        </div>
      </div>
    </div>
  </div>

  <el-dialog
    v-else
    v-model="visible"
    :title="title"
    width="90%"
    top="5vh"
    destroy-on-close
    class="packet-viewer-dialog"
    @close="handleClose"
  >
    <div class="packet-viewer-container">
      <div class="packet-viewer-toolbar">
        <div class="toolbar-left">
          <template v-if="showSendActions">
            <el-button
              v-if="!loading"
              type="primary"
              size="small"
              @click="handleSendClick"
            >
              发送请求
            </el-button>
            <el-button
              v-else
              type="danger"
              size="small"
              :loading="canceling"
              :disabled="canceling"
              @click="handleCancelClick"
            >
              取消请求
            </el-button>
            <el-checkbox v-model="forceHttps" size="small">
              强制HTTPS
            </el-checkbox>
          </template>
          <el-button
            v-if="enableSave"
            type="success"
            size="small"
            plain
            @click="handleSaveClick"
          >
            导入并保存
          </el-button>
          <el-button
            v-if="showSendActions"
            type="warning"
            size="small"
            plain
            @click="handleUploadBypassClick"
            :disabled="loading"
          >
            文件上传Bypass
          </el-button>
        </div>
        <div class="toolbar-right">
          <el-button-group>
            <el-button
              size="small"
              @click="formatRequest"
              title="格式化请求包"
            >
              格式化请求
            </el-button>
            <el-button
              size="small"
              @click="formatResponse"
              title="格式化响应包"
            >
              格式化响应
            </el-button>
          </el-button-group>
        </div>
      </div>

      <div class="packet-viewer-content">
        <div class="packet-panel request-panel">
          <div class="panel-header">
            <div class="panel-title-row">
              <span class="panel-title">请求包</span>
              <el-button
                :icon="CopyDocument"
                size="small"
                text
                @click="copyRequest"
              >
                复制请求包
              </el-button>
            </div>
          </div>
          <div class="panel-content" ref="requestPanelRef">
            <VueMonacoEditor
              v-if="editable"
              ref="requestEditorRef"
              v-model:value="editableRequestPacket"
              :language="requestLanguage"
              theme="http-dark-theme"
              :height="panelEditorHeight"
              class="code-editor"
              :options="monacoOptions"
              placeholder="在此粘贴/编辑请求数据包"
            />
            <VueMonacoEditor
              v-else
              :value="displayRequestPacket"
              :language="requestLanguage"
              theme="http-dark-theme"
              :height="panelEditorHeight"
              class="code-editor"
              :options="previewMonacoOptions"
            />
          </div>
        </div>

        <div class="packet-panel response-panel">
          <div class="panel-header">
            <div class="panel-title-row">
              <span class="panel-title">响应包</span>
              <el-button
                text
                size="small"
                :disabled="!responsePreviewSrcdoc"
                @click="openResponseHtmlPreview"
              >
                网页预览
              </el-button>
              <el-button
                :icon="CopyDocument"
                size="small"
                text
                @click="copyResponse"
              >
                复制响应包
              </el-button>
            </div>
          </div>
          <div class="panel-content" ref="responsePanelRef">
            <VueMonacoEditor
              v-if="editable"
              ref="responseEditorRef"
              v-model:value="editableResponsePacket"
              :language="responseLanguage"
              theme="http-dark-theme"
              :height="panelEditorHeight"
              class="code-editor"
              :options="monacoOptions"
              placeholder="可选：在此粘贴响应数据包"
            />
            <VueMonacoEditor
              v-else
              :value="displayResponsePacket"
              :language="responseLanguage"
              theme="http-dark-theme"
              :height="panelEditorHeight"
              class="code-editor"
              :options="previewMonacoOptions"
            />
          </div>
        </div>
      </div>
    </div>
  </el-dialog>

  <el-dialog
    v-model="responsePreviewVisible"
    title="响应网页预览"
    width="72%"
    top="6vh"
    append-to-body
    destroy-on-close
    class="packet-response-preview-dialog"
  >
    <iframe
      class="packet-response-preview-frame"
      :srcdoc="responsePreviewSrcdoc"
      sandbox="allow-same-origin allow-scripts"
    />
  </el-dialog>
</template>

<script>
import { CopyDocument } from '@element-plus/icons-vue'
import { markRaw, ref, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { VueMonacoEditor } from '@guolao/vue-monaco-editor'
import { useMonaco } from '@guolao/vue-monaco-editor'
// 预注册基础语言，HTTP 在下方手动注册
import 'monaco-editor/esm/vs/basic-languages/monaco.contribution'
import { registerHTTPLanguage, formatHTTPResponse, formatHTTPRequest, detectPacketLanguage } from '../utils/http-language';

export default {
  name: 'PacketViewer',
  components: {
    CopyDocument: markRaw(CopyDocument),
    VueMonacoEditor
  },
  props: {
    modelValue: {
      type: Boolean,
      default: false
    },
    requestPacket: {
      type: String,
      default: ''
    },
    responsePacket: {
      type: String,
      default: ''
    },
    title: {
      type: String,
      default: '数据包查看'
    },
    editable: {
      type: Boolean,
      default: false
    },
    enableSave: {
      type: Boolean,
      default: false
    },
    loading: {
      type: Boolean,
      default: false
    },
    canceling: {
      type: Boolean,
      default: false
    },
    showSendActions: {
      type: Boolean,
      default: true
    },
    // 纯文本模式：强制使用 plaintext 语法高亮（关闭自动检测）
    plainText: {
      type: Boolean,
      default: false
    },
    embedded: {
      type: Boolean,
      default: false
    }
  },
  emits: ['update:modelValue', 'send', 'save', 'cancel', 'upload-bypass', 'update:requestPacket', 'update:responsePacket'],
  mounted() {
    this.registerHttpLanguage()
  },
  setup() {
    const requestEditorRef = ref(null)
    const responseEditorRef = ref(null)
    
    return {
      requestEditorRef,
      responseEditorRef
    }
  },
  data() {
    return {
      CopyDocument: markRaw(CopyDocument),
      requestPanelRef: null,
      responsePanelRef: null,
      editableRequestPacket: '',
      editableResponsePacket: '',
      displayRequestPacket: '',
      displayResponsePacket: '',
      forceHttps: false,
      responsePreviewVisible: false,
      monacoOptions: {
        minimap: { enabled: false },
        wordWrap: 'on',
        fontSize: 13,
        scrollbar: { verticalScrollbarSize: 8 },
        automaticLayout: true,
        scrollBeyondLastLine: false,
        lineNumbers: 'on',
        formatOnPaste: true,
        formatOnType: false
      },
      previewMonacoOptions: {
        minimap: { enabled: false },
        wordWrap: 'on',
        fontSize: 13,
        scrollbar: { verticalScrollbarSize: 8 },
        automaticLayout: true,
        scrollBeyondLastLine: false,
        readOnly: true,
        lineNumbers: 'on'
      }
    }
  },
  computed: {
    currentRequestPacket() {
      return this.editable ? this.editableRequestPacket : this.displayRequestPacket
    },
    currentResponsePacket() {
      return this.editable ? this.editableResponsePacket : this.displayResponsePacket
    },
    requestLanguage() {
      return this.plainText ? 'plaintext' : detectPacketLanguage(this.currentRequestPacket)
    },
    responseLanguage() {
      return this.plainText ? 'plaintext' : detectPacketLanguage(this.currentResponsePacket)
    },
    panelEditorHeight() {
      return this.embedded ? 'calc(100vh - 230px)' : 'calc(80vh - 150px)';
    },
    responsePreviewSrcdoc() {
      return this.extractHtmlBodyFromRaw(this.currentResponsePacket || '');
    },
    visible: {
      get() {
        return this.modelValue
      },
      set(val) {
        this.$emit('update:modelValue', val)
      }
    }
  },
  watch: {
    modelValue(val) {
      if (this.embedded) return;
      if (val) {
        this.editableRequestPacket = this.requestPacket || ''
        this.editableResponsePacket = this.responsePacket || ''
        this.displayRequestPacket = this.requestPacket || ''
        this.displayResponsePacket = this.responsePacket || ''
        this.forceHttps = false

        // 自动格式化：响应包总是自动，请求包仅在只读时自动
        if (!this.editable) {
          this.formatRequest({ auto: true })
        }
        this.formatResponse({ auto: true })
      }
    },
    requestPacket(val) {
      this.editableRequestPacket = val || ''
      this.displayRequestPacket = val || ''
      // 只读模式下自动格式化请求包
      if (!this.editable) {
        this.formatRequest({ auto: true })
      }
    },
    responsePacket(val) {
      this.editableResponsePacket = val || ''
      this.displayResponsePacket = val || ''
      // 响应包始终自动格式化
      this.formatResponse({ auto: true })
    },
    editableRequestPacket(val) {
      if (!this.embedded) return;
      if ((val || '') === (this.requestPacket || '')) return;
      this.$emit('update:requestPacket', val);
    },
    editableResponsePacket(val) {
      if (!this.embedded) return;
      if ((val || '') === (this.responsePacket || '')) return;
      this.$emit('update:responsePacket', val);
    }
  },
  methods: {
    extractHtmlBodyFromRaw(raw) {
      const text = (raw || '').replace(/\r\n/g, '\n');
      const sep = text.indexOf('\n\n');
      if (sep < 0) return '';
      const headerBlock = text.slice(0, sep);
      const headerLower = headerBlock.toLowerCase();
      const firstLine = (headerBlock.split('\n')[0] || '').trim();
      if (!/^http\/\d/i.test(firstLine)) return '';
      const ctLine = headerLower.split('\n').find(l => l.trimStart().startsWith('content-type:'));
      if (!ctLine || !ctLine.includes('html')) return '';
      return text.slice(sep + 2).trim();
    },
    openResponseHtmlPreview() {
      if (!this.responsePreviewSrcdoc) {
        ElMessage.warning('当前响应不是可预览的 HTML');
        return;
      }
      this.responsePreviewVisible = true;
    },
    registerHttpLanguage() {
      const { monacoRef } = useMonaco()
      if (!monacoRef.value) {
        // 如果 Monaco 还没加载，稍后重试
        setTimeout(() => this.registerHttpLanguage(), 100)
        return
      }

      const monaco = monacoRef.value

      // 若已注册则跳过
      if (monaco.languages.getLanguages().some(lang => lang.id === 'http')) {
        return
      }

      // 使用导入的registerHTTPLanguage函数
      registerHTTPLanguage(monaco)
    },
    
    async formatRequest({ auto = false } = {}) {
      const targetKey = this.editable ? 'editableRequestPacket' : 'displayRequestPacket'
      const content = this[targetKey] || ''
      
      if (!content.trim()) {
        if (!auto) {
          ElMessage.warning('请求包为空')
        }
        return content
      }
      
      try {
        const formatted = formatHTTPRequest(content)
        this[targetKey] = formatted
        
        // 如果编辑器已加载，触发Monaco的格式化
        if (this.requestEditorRef?.getEditor) {
          await nextTick()
          const editor = this.requestEditorRef.getEditor()
          if (editor) {
            await editor.getAction('editor.action.formatDocument').run().catch(() => {})
            this[targetKey] = editor.getValue()
          }
        }
        
        if (!auto) {
          ElMessage.success('请求包格式化完成')
        }
        return this[targetKey]
      } catch (error) {
        console.error('请求格式化失败:', error)
        this.simpleFormat('request', auto)
        return this[targetKey]
      }
    },
    
    async formatResponse({ auto = false } = {}) {
      const targetKey = this.editable ? 'editableResponsePacket' : 'displayResponsePacket'
      const content = this[targetKey] || ''
      
      if (!content.trim()) {
        if (!auto) {
          ElMessage.warning('响应包为空')
        }
        return content
      }
      
      try {
        const formatted = formatHTTPResponse(content)
        this[targetKey] = formatted
        
        // 如果编辑器已加载，触发Monaco的格式化
        if (this.responseEditorRef?.getEditor) {
          await nextTick()
          const editor = this.responseEditorRef.getEditor()
          if (editor) {
            await editor.getAction('editor.action.formatDocument').run().catch(() => {})
            this[targetKey] = editor.getValue()
          }
        }
        
        if (!auto) {
          ElMessage.success('响应包格式化完成')
        }
        return this[targetKey]
      } catch (error) {
        console.error('响应格式化失败:', error)
        this.simpleFormat('response', auto)
        return this[targetKey]
      }
    },
    
    simpleFormat(type, auto = false) {
      try {
        const key = type === 'request'
          ? (this.editable ? 'editableRequestPacket' : 'displayRequestPacket')
          : (this.editable ? 'editableResponsePacket' : 'displayResponsePacket')
        const content = this[key]
        const lines = content.split('\n')
        let inHeaders = true
        let result = []
        
        // 处理头部
        for (let i = 0; i < lines.length; i++) {
          const line = lines[i]
          if (inHeaders) {
            if (line.trim() === '') {
              inHeaders = false
              result.push('')
              break
            } else {
              result.push(line.trimEnd())
            }
          }
        }
        
        // 处理body
        if (!inHeaders) {
          const bodyStart = result.length
          const bodyLines = lines.slice(bodyStart)
          const bodyText = bodyLines.join('\n').trim()
          
          if (bodyText) {
            const trimmedBody = bodyText.trim()
            if ((trimmedBody.startsWith('{') && trimmedBody.endsWith('}')) ||
                (trimmedBody.startsWith('[') && trimmedBody.endsWith(']'))) {
              try {
                const parsed = JSON.parse(trimmedBody)
                const formattedJSON = JSON.stringify(parsed, null, 2)
                result.push(formattedJSON)
              } catch (e) {
                result.push(...bodyLines)
              }
            } else {
              result.push(...bodyLines)
            }
          }
        }
        
        this[key] = result.join('\n')
        
        if (!auto) {
          ElMessage.success(`${type === 'request' ? '请求包' : '响应包'}格式化完成`)
        }
      } catch (error) {
        console.error('简单格式化失败:', error)
        ElMessage.error('格式化失败')
      }
    },
    
    copyRequest() {
      const text = this.currentRequestPacket || ''
      if (!text) {
        ElMessage.warning('无请求包数据')
        return
      }
      navigator.clipboard.writeText(text).then(() => {
        ElMessage.success('请求包已复制到剪贴板')
      }).catch(() => {
        ElMessage.error('复制失败')
      })
    },
    copyResponse() {
      const text = this.currentResponsePacket || ''
      if (!text) {
        ElMessage.warning('无响应包数据')
        return
      }
      navigator.clipboard.writeText(text).then(() => {
        ElMessage.success('响应包已复制到剪贴板')
      }).catch(() => {
        ElMessage.error('复制失败')
      })
    },
    copyAll() {
      const all = `=== 请求包 ===\r\n${this.currentRequestPacket}\r\n\r\n=== 响应包 ===\r\n${this.currentResponsePacket}`
      navigator.clipboard.writeText(all).then(() => {
        ElMessage.success('全部内容已复制到剪贴板')
      }).catch(() => {
        ElMessage.error('复制失败')
      })
    },
    async handleSendClick() {
      // 发送前自动格式化请求包
      await this.formatRequest({ auto: true })
      const packet = this.currentRequestPacket
      const resp = this.currentResponsePacket
      if (!packet) {
        ElMessage.warning('无请求包数据，无法发包');
        return;
      }
      this.$emit('send', {
        requestPacket: packet,
        responsePacket: resp,
        title: this.title,
        forceHttps: this.forceHttps,
      });
    },
    handleSaveClick() {
      const packet = this.currentRequestPacket
      const resp = this.currentResponsePacket
      if (!packet) {
        ElMessage.warning('无请求包数据，无法保存');
        return;
      }
      this.$emit('save', {
        requestPacket: packet,
        responsePacket: resp,
        title: this.title,
        forceHttps: this.forceHttps,
      });
    },
    handleCancelClick() {
      this.$emit('cancel');
    },
    handleClose() {
      this.$emit('update:modelValue', false)
    },
    handleUploadBypassClick() {
      const packet = this.currentRequestPacket
      if (!packet) {
        ElMessage.warning('无请求包数据，无法进行文件上传 bypass');
        return;
      }
      // 检查是否是文件上传请求
      if (!packet.includes('multipart/form-data') && !packet.includes('filename=')) {
        ElMessage.warning('当前请求不是文件上传请求');
        return;
      }
      this.$emit('upload-bypass', {
        requestPacket: packet,
        forceHttps: this.forceHttps,
      });
    }
  }
}
</script>

<style lang="scss" scoped>
.packet-viewer-dialog {
  :deep(.el-dialog__body) {
    padding: 0;
  }
}

.packet-viewer-embedded {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: #fff;
}

.packet-viewer-container--embedded {
  height: calc(100vh - 120px);
  min-height: 360px;
}

.packet-response-preview-dialog {
  :deep(.el-dialog__body) {
    padding: 8px 12px 12px;
  }
}

.packet-response-preview-frame {
  width: 100%;
  height: 68vh;
  border: none;
  border-radius: 6px;
  background: #fff;
}

.packet-viewer-container {
  display: flex;
  flex-direction: column;
  height: 80vh;
}

.packet-viewer-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  background: #f5f7fa;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.packet-viewer-content {
  flex: 1;
  display: flex;
  gap: 1px;
  overflow: hidden;
  background: #e4e7ed;
}

.packet-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #fff;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
}

.panel-title {
  font-weight: 600;
  color: #303133;
  font-size: 14px;
}

.panel-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.panel-content {
  flex: 1;
  overflow: hidden;
  padding: 0;
  background: #1e1e1e;
  display: flex;
  flex-direction: column;
}

.code-editor {
  border-radius: 4px;
  overflow: hidden;
  border: 1px solid #1f2a44;
}

.code-editor :deep(.monaco-editor),
.code-editor :deep(.monaco-editor-background) {
  background: #1e1e1e;
}

.code-editor :deep(.monaco-editor) {
  font-family: "JetBrains Mono", "Fira Code", monospace;
}

// 自定义滚动条
.panel-content::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.panel-content::-webkit-scrollbar-track {
  background: #1e1e1e;
}

.panel-content::-webkit-scrollbar-thumb {
  background: #424242;
  border-radius: 4px;
  
  &:hover {
    background: #4e4e4e;
  }
}
</style>