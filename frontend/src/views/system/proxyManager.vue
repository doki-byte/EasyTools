<template>
  <el-card class="proxy-card">
    <template #header>
      <div class="card-header">
        <span>全局代理配置</span>
        <div style="display: flex; gap: 20px; align-items: center;">
          <el-switch
              v-model="globalProxyEnabled"
              active-text="启用代理"
              inactive-text="禁用代理"
              @change="handleGlobalProxyToggle"
          />
          <template v-if="isWindows">
            <el-divider direction="vertical" />
            <el-switch
                v-model="systemProxyEnabled"
                active-text="系统代理"
                inactive-text="程序代理"
                :disabled="!systemProxySupported || !globalProxyEnabled"
                @change="handleSystemProxyToggle"
            />
            <el-tooltip
                v-if="!systemProxySupported"
                effect="dark"
                content="系统代理功能在当前平台不可用"
                placement="top"
            >
              <el-icon style="color: #909399;"><QuestionFilled /></el-icon>
            </el-tooltip>
            <el-tooltip
                v-else-if="!globalProxyEnabled"
                effect="dark"
                content="请先启用程序代理，才能使用系统代理功能"
                placement="top"
            >
              <el-icon style="color: #909399;"><QuestionFilled /></el-icon>
            </el-tooltip>
          </template>
        </div>
      </div>
    </template>

    <el-form :model="proxyConfig" :rules="proxyRules" ref="proxyFormRef" label-width="100px">
      <el-form-item label="代理类型" prop="type">
        <el-radio-group v-model="proxyConfig.type">
          <el-radio value="http">HTTP</el-radio>
          <el-radio value="https">HTTPS</el-radio>
          <el-radio value="socks5">SOCKS5</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="服务器地址" prop="host">
        <el-input
            v-model="proxyConfig.host"
            placeholder="例如: 127.0.0.1"
            :disabled="!globalProxyEnabled"
            style="width: 200px"
        />
      </el-form-item>

      <el-form-item label="端口" prop="port">
        <el-input
            v-model="proxyConfig.port"
            placeholder="例如: 8080"
            :disabled="!globalProxyEnabled"
            style="width: 120px"
        />
      </el-form-item>

      <el-form-item label="超时时间" prop="timeout">
        <div style="display: flex; align-items: center; gap: 10px;">
          <el-input
              v-model.number="proxyConfig.timeout"
              placeholder="例如: 30"
              :disabled="!globalProxyEnabled"
              style="width: 120px"
              type="number"
              min="1"
              max="300"
          >
            <template #append>秒</template>
          </el-input>
          <el-tooltip
              effect="dark"
              content="设置代理连接和请求的超时时间（1-300秒）"
              placement="right"
          >
            <el-icon><QuestionFilled /></el-icon>
          </el-tooltip>
        </div>
      </el-form-item>

      <el-form-item label="认证信息">
        <div class="auth-section">
          <el-checkbox v-model="useAuth" :disabled="!globalProxyEnabled">
            需要认证
          </el-checkbox>

          <div v-if="useAuth" class="auth-fields">
            <el-input
                v-model="proxyConfig.auth.username"
                placeholder="用户名"
                :disabled="!globalProxyEnabled"
                style="width: 150px; margin-right: 10px"
            />
            <el-input
                v-model="proxyConfig.auth.password"
                placeholder="密码"
                type="password"
                :disabled="!globalProxyEnabled"
                style="width: 150px"
                show-password
            />
          </div>
        </div>
      </el-form-item>

      <el-form-item>
        <el-button
            type="primary"
            :loading="testing"
            :disabled="!globalProxyEnabled"
            @click="testProxy"
        >
          测试连接
        </el-button>

        <el-button
            type="success"
            :loading="saving"
            @click="saveProxyConfig"
        >
          保存配置变更
        </el-button>
      </el-form-item>
    </el-form>

    <!-- 测试结果 -->
    <div v-if="testResult" class="test-result">
      <el-alert
          :title="testResult.message"
          :type="testResult.type"
          :closable="false"
          show-icon
      />
    </div>
  </el-card>

  <!-- 使用说明 -->
  <el-card class="info-card">
    <template #header>
      <span>使用说明</span>
    </template>

    <div class="info-content">
      <p><strong>代理模式说明：</strong></p>
      <ul>
        <li><strong>程序代理模式</strong>：仅当前程序的网络请求通过代理服务器</li>
        <li><strong>系统代理模式</strong>（仅 Windows 平台）：系统级别的代理设置，所有应用程序的网络请求都会通过代理</li>
      </ul>

      <p><strong>注意事项：</strong></p>
      <ul>
        <li><b>开启/关闭代理开关会立即生效，无需点击保存按钮</b></li>
        <li><b>修改代理地址、端口等配置后，需要点击"保存配置变更"按钮</b></li>
        <li>启用程序代理后，仅当前程序的网络请求会通过配置的代理服务器</li>
        <li>启用系统代理后，整个系统的网络请求都会通过代理（仅 Windows 平台支持）</li>
        <li>请确保代理服务器配置正确，否则可能导致网络连接失败</li>
        <li>测试连接功能可以帮助验证代理配置是否有效</li>
        <li>系统代理需要管理员权限，如果设置失败，请尝试以管理员身份运行程序</li>
      </ul>
    </div>
  </el-card>
</template>

<script>
import {QuestionFilled} from "@element-plus/icons-vue";
import {ElMessage, ElMessageBox} from "element-plus";
import {GetGlobalProxy, SetGlobalProxy, TestProxyConnection, SetSystemProxy, GetSystemProxy, IsSystemProxySupported} from "../../../wailsjs/go/proxy/ProxyManager";
import {GetOs} from "../../../wailsjs/go/system/System";

export default {
  name: 'ProxyManager',
  components: {QuestionFilled},
  data() {
    return {
      // 代理配置相关数据
      globalProxyEnabled: false,
      systemProxyEnabled: false,
      systemProxySupported: false,
      isWindows: false,
      proxyConfig: {
        type: 'http',
        host: '127.0.0.1',
        port: '8080',
        timeout: 10,
        auth: {
          username: '',
          password: ''
        }
      },
      useAuth: false,
      testing: false,
      saving: false,
      testResult: null,

      proxyRules: {
        host: [
          { required: true, message: '请输入代理服务器地址', trigger: 'blur' }
        ],
        port: [
          { required: true, message: '请输入代理端口', trigger: 'blur' },
          { pattern: /^[0-9]+$/, message: '端口必须是数字', trigger: 'blur' }
        ],
        timeout: [ // 新增超时验证规则
          { required: true, message: '请输入超时时间', trigger: 'blur' },
          { pattern: /^[0-9]+$/, message: '超时时间必须是数字', trigger: 'blur' },
          { validator: this.validateTimeout, trigger: 'blur' }
        ]
      }
    }
  },
  async created() {
    await this.checkOS()
    await this.checkSystemProxySupport()
    await this.loadProxyConfig()
    if (this.isWindows) {
      await this.loadSystemProxyConfig()
    }
  },
  methods: {
    // 检测操作系统
    async checkOS() {
      try {
        const os = await GetOs()
        this.isWindows = os === 'windows'
      } catch (error) {
        console.error('检测操作系统失败:', error)
        this.isWindows = false
      }
    },
    // 代理配置方法 - 简化为全局模式
    async loadProxyConfig() {
      try {
        const result = await GetGlobalProxy()

        if (result && typeof result === 'object') {
          this.globalProxyEnabled = result.enabled || false

          if (result.config) {
            this.proxyConfig = {
              type: result.config.type || 'http',
              host: result.config.host || '127.0.0.1',
              port: result.config.port || '8080',
              timeout: result.config.timeout || 10, // 加载超时配置
              auth: result.config.auth || { username: '', password: '' }
            }
            this.useAuth = !!(result.config.auth && result.config.auth.username)
          }
          
          // 如果程序代理未启用，确保系统代理也被禁用
          if (!this.globalProxyEnabled && this.systemProxyEnabled && this.isWindows) {
            this.systemProxyEnabled = false
            // 如果系统代理之前是启用的，需要禁用系统代理
            if (this.systemProxySupported) {
              await this.saveSystemProxyConfig(false)
            }
          }
        }
      } catch (error) {
        ElMessage.error('加载代理配置失败: ' + error.message)
      }
    },

    async handleGlobalProxyToggle(enabled) {
      console.log('代理开关状态改变:', enabled)

      // 如果启用代理，需要验证配置
      if (enabled) {
        if (!this.proxyConfig.host || !this.proxyConfig.port) {
          ElMessage.warning('请先配置代理服务器地址和端口')
          this.globalProxyEnabled = false
          return
        }

        // 验证表单
        if (this.$refs.proxyFormRef) {
          try {
            await this.$refs.proxyFormRef.validate()
          } catch (error) {
            ElMessage.warning('请完善代理配置信息')
            this.globalProxyEnabled = false
            return
          }
        }
      } else {
        // 禁用代理时确认
        try {
          await ElMessageBox.confirm(
              '确定要禁用程序代理吗？禁用后系统代理也会自动关闭。',
              '确认操作',
              { type: 'warning' }
          )
        } catch (error) {
          console.log('用户取消禁用代理')
          this.globalProxyEnabled = true
          return
        }
        
        // 如果系统代理已启用，先禁用系统代理
        if (this.isWindows && this.systemProxyEnabled) {
          await this.saveSystemProxyConfig(false)
        }
      }

      // 直接保存并生效
      await this.saveProxyConfigImmediate(enabled)
    },

    // 立即保存代理配置（用于开关切换时）
    async saveProxyConfigImmediate(enabled) {
      this.saving = true
      this.testResult = null

      try {
        const config = {
          ...this.proxyConfig,
          auth: this.useAuth ? this.proxyConfig.auth : null
        }

        // 添加超时处理
        const savePromise = SetGlobalProxy(config, enabled)
        const timeoutPromise = new Promise((_, reject) => {
          setTimeout(() => reject(new Error('保存超时')), 10000)
        });

        const success = await Promise.race([savePromise, timeoutPromise])

        if (success) {
          ElMessage.success(enabled ? '程序代理已启用' : '程序代理已禁用')
          window.dispatchEvent(new CustomEvent('proxy-config-saved'))
        } else {
          ElMessage.error('保存代理配置失败')
          // 恢复开关状态
          this.globalProxyEnabled = !enabled
        }
      } catch (error) {
        ElMessage.error('保存代理配置失败: ' + (error.message || error))
        // 恢复开关状态
        this.globalProxyEnabled = !enabled
      } finally {
        this.saving = false
      }
    },

    // 保存代理配置（用于手动保存配置变更）
    async saveProxyConfig() {
      // 如果启用了代理，需要验证表单
      if (this.globalProxyEnabled) {
        if (!this.$refs.proxyFormRef) {
          ElMessage.warning('表单引用未初始化')
          return
        }

        try {
          await this.$refs.proxyFormRef.validate()
        } catch (error) {
          ElMessage.warning('请完善代理配置信息')
          return
        }
      }

      this.saving = true
      this.testResult = null

      try {
        const config = {
          ...this.proxyConfig,
          auth: this.useAuth ? this.proxyConfig.auth : null
        }

        // 添加超时处理
        const savePromise = SetGlobalProxy(config, this.globalProxyEnabled)
        const timeoutPromise = new Promise((_, reject) => {
          setTimeout(() => reject(new Error('保存超时')), 10000)
        });

        const success = await Promise.race([savePromise, timeoutPromise])

        if (success) {
          ElMessage.success('代理配置已保存')
          window.dispatchEvent(new CustomEvent('proxy-config-saved'))
        } else {
          ElMessage.error('保存代理配置失败')
        }
      } catch (error) {
        ElMessage.error('保存代理配置失败: ' + (error.message || error))
      } finally {
        this.saving = false
      }
    },

    async testProxy() {
      if (!this.$refs.proxyFormRef) return

      // 验证表单
      try {
        await this.$refs.proxyFormRef.validate()
      } catch (error) {
        ElMessage.warning('请完善代理配置信息')
        return
      }

      this.testing = true
      this.testResult = null

      try {
        const config = {
          ...this.proxyConfig,
          auth: this.useAuth ? this.proxyConfig.auth : null
        }

        const success = await TestProxyConnection(config)

        if (success) {
          this.testResult = {
            type: 'success',
            message: '代理连接测试成功！'
          }
          ElMessage.success('代理连接测试成功！')
        } else {
          this.testResult = {
            type: 'error',
            message: '代理连接测试失败'
          }
          ElMessage.error('代理连接测试失败')
        }
      } catch (error) {
        console.error('代理测试错误:', error)
        this.testResult = {
          type: 'error',
          message: `测试失败: 代理无法访问`
        }
        ElMessage.error(`测试失败: 代理无法访问`)
      } finally {
        this.testing = false
      }
    },
    // 超时时间验证
    validateTimeout(rule, value, callback) {
      if (!value) {
        callback(new Error('请输入超时时间'));
        return;
      }

      const timeout = parseInt(value);
      if (isNaN(timeout) || timeout <= 0) {
        callback(new Error('超时时间必须大于0'));
      } else if (timeout > 300) {
        callback(new Error('超时时间不能超过300秒'));
      } else {
        callback();
      }
    },

    // 检查系统代理是否支持
    async checkSystemProxySupport() {
      if (!this.isWindows) {
        this.systemProxySupported = false
        return
      }
      try {
        this.systemProxySupported = await IsSystemProxySupported()
      } catch (error) {
        console.error('检查系统代理支持失败:', error)
        this.systemProxySupported = false
      }
    },

    // 加载系统代理配置
    async loadSystemProxyConfig() {
      if (!this.isWindows || !this.systemProxySupported) {
        return
      }

      try {
        const systemConfig = await GetSystemProxy()
        // 只有在程序代理已启用时，才允许系统代理启用
        if (systemConfig.enabled && this.globalProxyEnabled) {
          this.systemProxyEnabled = true
        } else {
          this.systemProxyEnabled = false
          // 如果系统代理在系统中是启用的，但程序代理未启用，需要禁用系统代理
          if (this.isWindows && systemConfig.enabled && !this.globalProxyEnabled) {
            await this.saveSystemProxyConfig(false)
          }
        }
      } catch (error) {
        console.error('加载系统代理配置失败:', error)
      }
    },

    // 处理系统代理开关切换
    async handleSystemProxyToggle(enabled) {
      if (!this.isWindows || !this.systemProxySupported) {
        ElMessage.warning('系统代理功能仅在 Windows 平台可用')
        this.systemProxyEnabled = false
        return
      }

      // 检查程序代理是否已启用
      if (enabled && !this.globalProxyEnabled) {
        ElMessage.warning('请先启用程序代理，才能使用系统代理功能')
        this.systemProxyEnabled = false
        return
      }

      if (enabled) {
        // 验证配置
        if (!this.proxyConfig.host || !this.proxyConfig.port) {
          ElMessage.warning('请先配置代理服务器地址和端口')
          this.systemProxyEnabled = false
          return
        }

        try {
          await ElMessageBox.confirm(
            '启用系统代理将修改系统网络设置，所有应用程序的网络请求都会通过代理。是否继续？',
            '确认启用系统代理',
            {
              type: 'warning',
              confirmButtonText: '确定',
              cancelButtonText: '取消'
            }
          )
        } catch (error) {
          this.systemProxyEnabled = false
          return
        }
      } else {
        try {
          await ElMessageBox.confirm(
            '确定要禁用系统代理吗？',
            '确认禁用系统代理',
            {
              type: 'warning',
              confirmButtonText: '确定',
              cancelButtonText: '取消'
            }
          )
        } catch (error) {
          this.systemProxyEnabled = true
          return
        }
      }

      // 设置系统代理
      await this.saveSystemProxyConfig(enabled)
    },

    // 保存系统代理配置
    async saveSystemProxyConfig(enabled) {
      if (!this.isWindows) {
        return
      }
      this.saving = true
      try {
        const config = {
          ...this.proxyConfig,
          auth: this.useAuth ? this.proxyConfig.auth : null
        }

        const success = await SetSystemProxy(config, enabled)
        if (success) {
          ElMessage.success(enabled ? '系统代理已启用' : '系统代理已禁用')
          this.systemProxyEnabled = enabled
          // 触发状态更新事件
          window.dispatchEvent(new CustomEvent('proxy-config-saved'))
        } else {
          ElMessage.error('设置系统代理失败')
          this.systemProxyEnabled = !enabled
        }
      } catch (error) {
        ElMessage.error('设置系统代理失败: ' + (error.message || error))
        this.systemProxyEnabled = !enabled
      } finally {
        this.saving = false
      }
    },
  }
}
</script>

<style scoped>


.proxy-config-content {
  padding: 20px;
  max-width: 600px;
  margin: 0 auto;
}

.proxy-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.auth-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.auth-fields {
  display: flex;
  gap: 8px;
}

.test-result {
  margin-top: 16px;
}

.info-card {
  margin-top: 20px;
}

.info-content ul {
  margin: 10px 0;
  padding-left: 20px;
}

.info-content li {
  margin-bottom: 5px;
  color: #606266;
}

.code-example {
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 12px;
  font-family: 'Monaco', 'Consolas', monospace;
  font-size: 12px;
  color: #303133;
  overflow-x: auto;
  margin: 10px 0;
  line-height: 1.4;
}
</style>