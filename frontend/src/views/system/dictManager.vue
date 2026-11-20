<template>
  <div class="dict-manager">
    <el-card class="dict-card">
      <template #header>
        <div class="card-header">
          <span>字典管理</span>
          <el-tag type="info" size="small">用于主机扫描的口令暴破功能</el-tag>
        </div>
      </template>

      <el-tabs v-model="activeTab" type="border-card">
        <!-- 用户名字典 -->
        <el-tab-pane label="用户名字典" name="username">
          <div class="dict-content">
            <div class="dict-header">
              <el-select
                v-model="selectedProtocol"
                placeholder="选择协议"
                style="width: 200px; margin-bottom: 16px;"
                @change="loadUsernameDict"
              >
                <el-option
                  v-for="protocol in availableProtocols"
                  :key="protocol"
                  :label="protocol"
                  :value="protocol"
                />
              </el-select>
              <el-button
                type="primary"
                :icon="Check"
                @click="saveUsernameDict"
                :loading="saving"
                :disabled="!selectedProtocol"
              >
                保存
              </el-button>
            </div>
            <el-input
              v-model="usernameDictText"
              type="textarea"
              :rows="15"
              placeholder="每行一个用户名"
              style="width: 100%;"
            />
            <div class="dict-info">
              <el-text type="info" size="small">
                当前协议: {{ selectedProtocol || '未选择' }} | 
                条目数: {{ usernameDictText ? usernameDictText.split('\n').filter(line => line.trim()).length : 0 }}
              </el-text>
            </div>
          </div>
        </el-tab-pane>

        <!-- 密码字典 -->
        <el-tab-pane label="密码字典" name="password">
          <div class="dict-content">
            <div class="dict-header">
              <el-button
                type="primary"
                :icon="Check"
                @click="savePasswordDict"
                :loading="saving"
              >
                保存
              </el-button>
            </div>
            <el-input
              v-model="passwordDictText"
              type="textarea"
              :rows="15"
              placeholder="每行一个密码"
              style="width: 100%;"
            />
            <div class="dict-info">
              <el-text type="info" size="small">
                条目数: {{ passwordDictText ? passwordDictText.split('\n').filter(line => line.trim()).length : 0 }}
              </el-text>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Check } from '@element-plus/icons-vue'
import {
  ListAvailableUsernameDicts,
  LoadUsernameDictFromFile,
  LoadPasswordDictFromFile,
  SaveUsernameDictToFile,
  SavePasswordDictToFile
} from '../../../wailsjs/go/crack/CrackController'

export default {
  name: 'DictManager',
  components: {
    Check
  },
  setup() {
    const activeTab = ref('username')
    const selectedProtocol = ref('')
    const availableProtocols = ref([])
    const usernameDictText = ref('')
    const passwordDictText = ref('')
    const saving = ref(false)

    // 加载可用的协议列表
    const loadProtocols = async () => {
      try {
        const protocols = await ListAvailableUsernameDicts()
        availableProtocols.value = protocols || []
        if (protocols && protocols.length > 0) {
          selectedProtocol.value = protocols[0]
          await loadUsernameDict()
        }
      } catch (error) {
        console.error('加载协议列表失败:', error)
        ElMessage.error('加载协议列表失败')
      }
    }

    // 加载用户名字典
    const loadUsernameDict = async () => {
      if (!selectedProtocol.value) {
        usernameDictText.value = ''
        return
      }
      try {
        const usernames = await LoadUsernameDictFromFile(selectedProtocol.value)
        usernameDictText.value = usernames ? usernames.join('\n') : ''
      } catch (error) {
        console.error('加载用户名字典失败:', error)
        ElMessage.error('加载用户名字典失败')
        usernameDictText.value = ''
      }
    }

    // 加载密码字典
    const loadPasswordDict = async () => {
      try {
        const passwords = await LoadPasswordDictFromFile()
        passwordDictText.value = passwords ? passwords.join('\n') : ''
      } catch (error) {
        console.error('加载密码字典失败:', error)
        ElMessage.error('加载密码字典失败')
        passwordDictText.value = ''
      }
    }

    // 保存用户名字典
    const saveUsernameDict = async () => {
      if (!selectedProtocol.value) {
        ElMessage.warning('请先选择协议')
        return
      }
      saving.value = true
      try {
        const usernames = usernameDictText.value
          .split('\n')
          .map(line => line.trim())
          .filter(line => line.length > 0)
        await SaveUsernameDictToFile(selectedProtocol.value, usernames)
        ElMessage.success('用户名字典保存成功')
      } catch (error) {
        console.error('保存用户名字典失败:', error)
        ElMessage.error('保存用户名字典失败: ' + (error.message || error))
      } finally {
        saving.value = false
      }
    }

    // 保存密码字典
    const savePasswordDict = async () => {
      saving.value = true
      try {
        const passwords = passwordDictText.value
          .split('\n')
          .map(line => line.trim())
          .filter(line => line.length > 0)
        await SavePasswordDictToFile(passwords)
        ElMessage.success('密码字典保存成功')
      } catch (error) {
        console.error('保存密码字典失败:', error)
        ElMessage.error('保存密码字典失败: ' + (error.message || error))
      } finally {
        saving.value = false
      }
    }

    onMounted(async () => {
      await loadProtocols()
      await loadPasswordDict()
    })

    return {
      activeTab,
      selectedProtocol,
      availableProtocols,
      usernameDictText,
      passwordDictText,
      saving,
      Check,
      loadUsernameDict,
      saveUsernameDict,
      savePasswordDict
    }
  }
}
</script>

<style scoped>
.dict-manager {
  padding: 20px;
}

.dict-card {
  background-color: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.dict-content {
  padding: 20px 0;
}

.dict-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.dict-info {
  margin-top: 12px;
  text-align: right;
}

:deep(.el-tabs__content) {
  padding: 20px;
}
</style>

