<template>
  <div class="poc-config-panel">
    <el-tabs v-model="activeTab">
      <el-tab-pane label="POC配置" name="poc">
        <el-form :model="config" label-width="140px" label-position="left">
          <el-form-item label="POC模板文件夹">
            <el-input
              v-model="config.templateFoldersText"
              type="textarea"
              :rows="4"
              placeholder="每行一个路径，例如:&#10;D:\nuclei-templates&#10;C:\custom-pocs"
            />
            <div class="form-tip">留空则使用默认路径</div>
          </el-form-item>

          <el-form-item label="默认请求头">
            <el-input
              v-model="config.defaultHeaders"
              type="textarea"
              :rows="4"
              placeholder="每行一个请求头，例如:&#10;User-Agent: Mozilla/5.0&#10;X-Forwarded-For: 127.0.0.1"
            />
          </el-form-item>

          <el-form-item label="自动重新加载">
            <el-switch v-model="config.autoReload" />
            <div class="form-tip">保存配置后自动重新加载POC标签</div>
          </el-form-item>

          <el-form-item>
            <el-button type="primary" @click="saveConfig" :loading="saving">保存配置</el-button>
            <el-button @click="reloadTags" :loading="reloading">重新加载标签</el-button>
            <el-button @click="validateFolder" :loading="validating">验证文件夹</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>
      <el-tab-pane label="扫描配置" name="scan">
        <el-form :model="config" label-width="140px" label-position="left">
          <el-form-item label="扫描线程数">
            <el-input-number v-model="config.threads" :min="1" :max="50" />
            <div class="form-tip">并发扫描的线程数量，建议值：5-10</div>
          </el-form-item>

          <el-form-item label="超时时间（秒）">
            <el-input-number v-model="config.timeout" :min="1" :max="300" />
            <div class="form-tip">单个请求的超时时间，建议值：10-30秒</div>
          </el-form-item>

          <el-form-item label="速率限制（/秒）">
            <el-input-number v-model="config.rateLimit" :min="1" :max="1000" />
            <div class="form-tip">每秒最大请求数，建议值：100-200</div>
          </el-form-item>

          <el-form-item label="重试次数">
            <el-input-number v-model="config.retries" :min="0" :max="5" />
            <div class="form-tip">请求失败时的重试次数，建议值：1-2</div>
          </el-form-item>

          <el-form-item>
            <el-button type="primary" @click="saveConfig" :loading="saving">保存配置</el-button>
            <el-button @click="resetScanConfig">重置为默认值</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>

    <el-divider>验证文件夹</el-divider>
    <el-form :inline="true">
      <el-form-item label="文件夹路径">
        <el-input
          v-model="validatePath"
          placeholder="输入要验证的文件夹路径"
          style="width: 400px"
        />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" @click="doValidateFolder">验证</el-button>
      </el-form-item>
    </el-form>

    <div v-if="validateResult" class="validate-result">
      <el-alert
        :type="validateResult.valid ? 'success' : 'error'"
        :title="validateResult.message"
        :description="validateResult.valid ? `找到 ${validateResult.count} 个YAML文件` : ''"
        show-icon
        :closable="false"
      />
    </div>
  </div>
</template>

<script>
import { GetPocConfig, SavePocConfig, ReloadPocTags, ValidateTemplateFolder } from '../../../wailsjs/go/webscan/PocConfigController'

export default {
  name: 'PocConfigPanel',
  emits: ['close'],
  data() {
    return {
      activeTab: 'poc',
      config: {
        templateFoldersText: '',
        defaultHeaders: '',
        autoReload: true,
        threads: 5,
        timeout: 10,
        rateLimit: 150,
        retries: 1
      },
      saving: false,
      reloading: false,
      validating: false,
      validatePath: '',
      validateResult: null
    }
  },
  async mounted() {
    await this.loadConfig()
  },
  methods: {
    async loadConfig() {
      try {
        const config = await GetPocConfig()
        // 后端返回的字段名是下划线命名，需要正确映射
        this.config.templateFoldersText = (config.template_folders || []).join('\n')
        this.config.defaultHeaders = config.default_headers || ''
        this.config.autoReload = config.auto_reload !== false
        this.config.threads = config.threads || 5
        this.config.timeout = config.timeout || 10
        this.config.rateLimit = config.rate_limit || 150
        this.config.retries = config.retries || 1
      } catch (error) {
        console.error('加载配置失败:', error)
        this.$message.error('加载配置失败: ' + (error.message || error))
      }
    },
    async saveConfig() {
      this.saving = true
      try {
        const config = {
          template_folders: this.config.templateFoldersText
            .split('\n')
            .map(p => p.trim())
            .filter(p => p),
          default_headers: this.config.defaultHeaders,
          auto_reload: this.config.autoReload,
          threads: this.config.threads || 5,
          timeout: this.config.timeout || 10,
          rate_limit: this.config.rateLimit || 150,
          retries: this.config.retries || 1
        }

        await SavePocConfig(JSON.stringify(config))

        // 保存配置后，始终主动触发一次标签重载，确保新添加的 POC 模板文件夹能立即生效
        // 这样就不需要退出重进才能在「POC 管理」页面看到最新列表/指纹
        this.reloading = true
        try {
          await ReloadPocTags()
          this.$message.success('配置已保存并重新加载标签')
        } catch (reloadError) {
          console.error('重新加载标签失败:', reloadError)
          this.$message.error('配置已保存，但重新加载标签失败: ' + (reloadError.message || reloadError))
        } finally {
          this.reloading = false
        }
      } catch (error) {
        console.error('保存配置失败:', error)
        this.$message.error('保存配置失败: ' + (error.message || error))
      } finally {
        this.saving = false
      }
    },
    resetScanConfig() {
      this.config.threads = 5
      this.config.timeout = 10
      this.config.rateLimit = 150
      this.config.retries = 1
      this.$message.success('已重置为默认值')
    },
    async reloadTags() {
      this.reloading = true
      try {
        await ReloadPocTags()
        this.$message.success('标签已重新加载')
      } catch (error) {
        console.error('重新加载标签失败:', error)
        this.$message.error('重新加载标签失败: ' + (error.message || error))
      } finally {
        this.reloading = false
      }
    },
    async validateFolder() {
      if (!this.validatePath.trim()) {
        this.$message.warning('请输入文件夹路径')
        return
      }
      this.validating = true
      this.validateResult = null
      try {
        const result = await ValidateTemplateFolder(this.validatePath)
        this.validateResult = result
        if (result.valid) {
          this.$message.success(`验证成功，找到 ${result.count} 个YAML文件`)
        } else {
          this.$message.warning(result.message)
        }
      } catch (error) {
        console.error('验证文件夹失败:', error)
        this.$message.error('验证文件夹失败: ' + (error.message || error))
      } finally {
        this.validating = false
      }
    },
    doValidateFolder() {
      this.validateFolder()
    }
  }
}
</script>

<style scoped>
.poc-config-panel {
  padding: 10px;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.validate-result {
  margin-top: 15px;
}
</style>

