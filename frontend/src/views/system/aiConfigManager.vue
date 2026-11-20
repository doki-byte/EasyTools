<template>
  <div class="ai-config-manager">
    <el-card class="config-card">
      <template #header>
        <div class="card-header">
          <span>AI大模型配置</span>
          <el-button type="primary" :icon="Plus" @click="showAddDialog">
            添加配置
          </el-button>
        </div>
      </template>

      <el-table :data="configs" stripe border style="width: 100%">
        <el-table-column type="index" width="60" align="center" label="#" />
        <el-table-column prop="name" label="配置名称" min-width="120" />
        <el-table-column prop="provider" label="提供商" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getProviderTagType(row.provider)" size="small">
              {{ getProviderName(row.provider) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="model" label="模型" width="150" />
        <el-table-column prop="api_base" label="API地址" min-width="200">
          <template #default="{ row }">
            <span :title="row.api_base">{{ truncateText(row.api_base, 30) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="enabled" label="启用" width="80" align="center">
          <template #default="{ row }">
            <el-switch
              v-model="row.enabled"
              @change="handleEnableChange(row)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="is_default" label="默认" width="80" align="center">
          <template #default="{ row }">
            <el-radio
              v-model="defaultConfigId"
              :label="row.id"
              @change="handleSetDefault(row.id)"
            >
              {{ row.is_default ? '是' : '' }}
            </el-radio>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="editConfig(row)">
              编辑
            </el-button>
            <el-button type="success" size="small" @click="testConfig(row)">
              测试
            </el-button>
            <el-button type="danger" size="small" @click="deleteConfig(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="600px"
      destroy-on-close
    >
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="120px"
      >
        <el-form-item label="配置名称" prop="name">
          <el-input v-model="formData.name" placeholder="例如：OpenAI配置" />
        </el-form-item>

        <el-form-item label="提供商" prop="provider">
          <el-select v-model="formData.provider" placeholder="选择AI提供商" style="width: 100%">
            <el-option label="OpenAI" value="openai" />
            <el-option label="DeepSeek" value="deepseek" />
            <el-option label="阿里云千机百炼" value="qianwen" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>

        <el-form-item label="API密钥" prop="api_key">
          <el-input
            v-model="formData.api_key"
            type="password"
            show-password
            placeholder="输入API密钥"
          />
        </el-form-item>

        <el-form-item label="API地址" prop="api_base">
          <el-input
            v-model="formData.api_base"
            placeholder="API基础URL"
          />
          <div class="form-tip">
            <template v-if="formData.provider === 'openai'">
              默认: https://api.openai.com/v1
            </template>
            <template v-else-if="formData.provider === 'deepseek'">
              默认: https://api.deepseek.com/v1
            </template>
            <template v-else-if="formData.provider === 'qianwen'">
              默认: https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation
            </template>
            <template v-else>
              请输入自定义API地址
            </template>
          </div>
        </el-form-item>

        <el-form-item label="模型名称" prop="model">
          <el-input
            v-model="formData.model"
            placeholder="模型名称"
          />
          <div class="form-tip">
            <template v-if="formData.provider === 'openai'">
              例如: gpt-3.5-turbo, gpt-4
            </template>
            <template v-else-if="formData.provider === 'deepseek'">
              例如: deepseek-chat, deepseek-reasoner
            </template>
            <template v-else-if="formData.provider === 'qianwen'">
              例如: qwen-turbo, qwen-plus, qwen-max
            </template>
            <template v-else>
              请输入模型名称
            </template>
          </div>
        </el-form-item>

        <el-form-item label="描述">
          <el-input
            v-model="formData.description"
            type="textarea"
            :rows="2"
            placeholder="配置描述（可选）"
          />
        </el-form-item>

        <el-form-item>
          <el-checkbox v-model="formData.enabled">启用此配置</el-checkbox>
        </el-form-item>

        <el-form-item>
          <el-checkbox v-model="formData.is_default">设为默认配置</el-checkbox>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveConfig" :loading="saving">
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 使用说明 -->
    <el-card class="info-card" style="margin-top: 20px">
      <template #header>
        <span>使用说明</span>
      </template>
      <div class="info-content">
        <p><strong>AI配置说明：</strong></p>
        <ul>
          <li><strong>OpenAI</strong>：使用 OpenAI API，需要配置 API Key</li>
          <li><strong>DeepSeek</strong>：使用 DeepSeek API</li>
          <li><strong>阿里云千机百炼</strong>：使用通义千问等模型</li>
          <li><strong>自定义</strong>：支持兼容 OpenAI 格式的自定义 API</li>
        </ul>
        <p><strong>注意事项：</strong></p>
        <ul>
          <li>API密钥会加密存储，请妥善保管</li>
          <li>可以配置多个AI提供商，但只能设置一个为默认配置</li>
          <li>在域名收集功能中，可以选择使用系统配置的AI，也可以手动输入API密钥</li>
          <li>测试功能可以验证配置是否正确</li>
        </ul>
      </div>
    </el-card>
  </div>
</template>

<script>
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm'
import {
  GetAllConfigs,
  AddConfig,
  UpdateConfig,
  DeleteConfig,
  SetDefaultConfig,
  TestConfig
} from '../../../wailsjs/go/system/AIConfigManager'

export default {
  name: 'AIConfigManager',
  components: {
    Plus
  },
  data() {
    return {
      configs: [],
      defaultConfigId: '',
      dialogVisible: false,
      dialogTitle: '添加配置',
      saving: false,
      formData: {
        id: '',
        name: '',
        provider: 'openai',
        api_key: '',
        api_base: '',
        model: '',
        enabled: true,
        is_default: false,
        description: ''
      },
      formRules: {
        name: [
          { required: true, message: '请输入配置名称', trigger: 'blur' }
        ],
        provider: [
          { required: true, message: '请选择提供商', trigger: 'change' }
        ],
        api_key: [
          { required: true, message: '请输入API密钥', trigger: 'blur' }
        ],
        api_base: [
          { required: true, message: '请输入API地址', trigger: 'blur' }
        ],
        model: [
          { required: true, message: '请输入模型名称', trigger: 'blur' }
        ]
      }
    }
  },
  watch: {
    'formData.provider'(newVal, oldVal) {
      // 切换提供商时，自动更新默认值（仅在新增时或用户未修改时）
      const isNewConfig = !this.formData.id || oldVal === undefined
      const userModified = this.formData.api_base && this.formData.api_base !== this.getDefaultAPIBase(oldVal)
      
      if (isNewConfig || !userModified) {
        this.formData.api_base = this.getDefaultAPIBase(newVal)
        this.formData.model = this.getDefaultModel(newVal)
      }
    }
  },
  mounted() {
    this.loadConfigs()
  },
  methods: {
    async loadConfigs() {
      try {
        const configsJson = await GetAllConfigs()
        this.configs = JSON.parse(configsJson)
        // 设置默认配置ID
        const defaultConfig = this.configs.find(c => c.is_default)
        if (defaultConfig) {
          this.defaultConfigId = defaultConfig.id
        }
      } catch (error) {
        ElMessage.error(`加载配置失败: ${error}`)
      }
    },
    getDefaultAPIBase(provider) {
      const defaults = {
        openai: 'https://api.openai.com/v1',
        deepseek: 'https://api.deepseek.com/v1',
        qianwen: 'https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation',
        custom: ''
      }
      return defaults[provider] || ''
    },
    getDefaultModel(provider) {
      const defaults = {
        openai: 'gpt-3.5-turbo',
        deepseek: 'deepseek-chat',
        qianwen: 'qwen-turbo',
        custom: 'gpt-3.5-turbo'
      }
      return defaults[provider] || ''
    },
    showAddDialog() {
      this.dialogTitle = '添加配置'
      this.formData = {
        id: '',
        name: '',
        provider: 'openai',
        api_key: '',
        api_base: this.getDefaultAPIBase('openai'),
        model: this.getDefaultModel('openai'),
        enabled: true,
        is_default: false,
        description: ''
      }
      this.dialogVisible = true
    },
    editConfig(row) {
      this.dialogTitle = '编辑配置'
      this.formData = {
        id: row.id,
        name: row.name,
        provider: row.provider,
        api_key: row.api_key,
        api_base: row.api_base,
        model: row.model,
        enabled: row.enabled,
        is_default: row.is_default,
        description: row.description || ''
      }
      this.dialogVisible = true
    },
    async saveConfig() {
      try {
        await this.$refs.formRef.validate()
        this.saving = true

        const configJson = JSON.stringify(this.formData)
        
        if (this.formData.id) {
          await UpdateConfig(configJson)
          ElMessage.success('更新配置成功')
        } else {
          await AddConfig(configJson)
          ElMessage.success('添加配置成功')
        }

        this.dialogVisible = false
        await this.loadConfigs()
      } catch (error) {
        if (error !== false) { // 表单验证失败会返回false
          ElMessage.error(`保存失败: ${error}`)
        }
      } finally {
        this.saving = false
      }
    },
    async deleteConfig(row) {
      try {
        await confirmDangerousAction({
          title: '确认删除',
          message: `确定要删除配置「${row.name}」吗？`,
          expectedInput: DANGER_DELETE_TOKEN
        })

        await DeleteConfig(row.id)
        ElMessage.success('删除成功')
        await this.loadConfigs()
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(`删除失败: ${error}`)
        }
      }
    },
    async handleEnableChange(row) {
      try {
        const configJson = JSON.stringify(row)
        await UpdateConfig(configJson)
        ElMessage.success('更新成功')
      } catch (error) {
        ElMessage.error(`更新失败: ${error}`)
        // 恢复状态
        row.enabled = !row.enabled
      }
    },
    async handleSetDefault(configId) {
      try {
        await SetDefaultConfig(configId)
        ElMessage.success('设置默认配置成功')
        await this.loadConfigs()
      } catch (error) {
        ElMessage.error(`设置失败: ${error}`)
        await this.loadConfigs()
      }
    },
    async testConfig(row) {
      try {
        ElMessage.info('正在测试配置...')
        const configJson = JSON.stringify(row)
        const result = await TestConfig(configJson)
        ElMessage.success(result || '配置测试成功')
      } catch (error) {
        ElMessage.error(`测试失败: ${error}`)
      }
    },
    getProviderName(provider) {
      const names = {
        openai: 'OpenAI',
        deepseek: 'DeepSeek',
        qianwen: '阿里云千机百炼',
        custom: '自定义'
      }
      return names[provider] || provider
    },
    getProviderTagType(provider) {
      const types = {
        openai: 'primary',
        deepseek: 'success',
        qianwen: 'danger',
        custom: 'info'
      }
      return types[provider] || 'info'
    },
    truncateText(text, maxLength) {
      if (!text) return '-'
      if (text.length <= maxLength) return text
      return text.substring(0, maxLength) + '...'
    }
  }
}
</script>

<style scoped>
.ai-config-manager {
  padding: 20px;
}

.config-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.info-content {
  line-height: 1.8;
}

.info-content ul {
  margin: 10px 0;
  padding-left: 20px;
}

.info-content a {
  color: #409eff;
  text-decoration: none;
}

.info-content a:hover {
  text-decoration: underline;
}
</style>
