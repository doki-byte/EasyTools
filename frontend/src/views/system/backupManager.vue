<template>
  <div class="backup-manager">
    <el-card class="backup-card">
      <template #header>
        <div class="card-header">
          <span>配置信息备份</span>
          <el-tag type="info" size="small">备份EasyTools配置文件和数据</el-tag>
        </div>
      </template>

      <el-form :model="backupConfig" label-width="180px" class="backup-form">
        <!-- 自动备份开关 -->
        <el-form-item label="自动备份">
          <el-switch
            v-model="backupConfig.autoBackup"
            active-text="开启"
            inactive-text="关闭"
            @change="saveConfig"
          />
          <el-text type="info" size="small" style="margin-left: 10px;">
            程序退出时自动执行备份
          </el-text>
        </el-form-item>

        <!-- 备份周期 -->
        <el-form-item label="备份周期" v-if="backupConfig.autoBackup">
          <el-select
            v-model="backupConfig.backupCycle"
            @change="saveConfig"
            style="width: 200px;"
          >
            <el-option label="每次退出" value="every_exit" />
            <el-option label="每天一次" value="daily" />
            <el-option label="每周一次" value="weekly" />
            <el-option label="每月一次" value="monthly" />
          </el-select>
          <el-text type="info" size="small" style="margin-left: 10px;">
            选择自动备份的执行频率
          </el-text>
        </el-form-item>

        <!-- 备份模式 -->
        <el-form-item label="备份模式">
          <el-radio-group v-model="backupConfig.backupMode" @change="saveConfig">
            <el-radio label="overwrite">覆盖模式</el-radio>
            <el-radio label="copy">创建副本</el-radio>
          </el-radio-group>
          <el-text type="info" size="small" style="margin-left: 10px; display: block; margin-top: 5px;">
            <span v-if="backupConfig.backupMode === 'overwrite'">直接覆盖备份目录，不保留历史版本</span>
            <span v-else>每次备份创建带时间戳的新目录，保留历史版本</span>
          </el-text>
        </el-form-item>

        <!-- 备份保存路径 -->
        <el-form-item label="备份保存路径">
          <el-input
            v-model="backupConfig.backupPath"
            placeholder="请选择备份保存路径"
            readonly
            style="flex: 1;"
          />
          <el-button
            type="primary"
            :icon="FolderOpened"
            @click="selectBackupPath"
            style="margin-left: 10px;"
          >
            选择路径
          </el-button>
        </el-form-item>

        <!-- 要备份的文件和文件夹 -->
        <el-form-item label="选择备份项">
          <div class="backup-items-container">
            <el-table
              ref="fileTableRef"
              :data="fileList"
              style="width: 100%"
              max-height="400"
              @selection-change="handleSelectionChange"
            >
              <el-table-column type="selection" width="55" :selectable="() => true" />
              <el-table-column prop="name" label="名称" width="200">
                <template #default="{ row }">
                  <el-icon v-if="row.isDir" style="margin-right: 5px;">
                    <Folder />
                  </el-icon>
                  <el-icon v-else style="margin-right: 5px;">
                    <Document />
                  </el-icon>
                  {{ row.name }}
                </template>
              </el-table-column>
              <el-table-column prop="isDir" label="类型" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.isDir ? 'success' : 'info'" size="small">
                    {{ row.isDir ? '目录' : '文件' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="size" label="大小" width="120">
                <template #default="{ row }">
                  {{ formatSize(row.size) }}
                </template>
              </el-table-column>
              <el-table-column prop="modified" label="修改时间" />
            </el-table>
          </div>
        </el-form-item>

        <!-- 操作按钮 -->
        <el-form-item>
          <el-button
            type="primary"
            :icon="Download"
            @click="performBackup"
            :loading="backingUp"
          >
            立即备份
          </el-button>
          <el-button
            :icon="Refresh"
            @click="loadFileList"
            :loading="loading"
          >
            刷新列表
          </el-button>
          <el-button
            :icon="Check"
            @click="setDefaultSelection"
            :loading="loading"
          >
            恢复默认选择
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  FolderOpened,
  Download,
  Refresh,
  Check,
  Folder,
  Document
} from '@element-plus/icons-vue'
import {
  GetConfigDir,
  GetUserHomeDir,
  GetDefaultBackupPath,
  ListConfigFiles,
  BackupConfigFiles,
  GetOpenDir
} from '../../../wailsjs/go/system/System'

const STORAGE_KEY = 'easytools:backup:config'
const DEFAULT_ITEMS = ['config.db', 'config', 'notes']

export default {
  name: 'BackupManager',
  components: {
    FolderOpened,
    Download,
    Refresh,
    Check,
    Folder,
    Document
  },
  setup() {
    const backupConfig = ref({
      autoBackup: false,
      backupCycle: 'every_exit', // 备份周期：every_exit(每次退出), daily(每天), weekly(每周), monthly(每月)
      backupMode: 'copy', // 备份模式：overwrite(覆盖), copy(创建副本)
      backupPath: '',
      selectedItems: [...DEFAULT_ITEMS]
    })

    const fileList = ref([])
    const selectedFiles = ref([])
    const loading = ref(false)
    const backingUp = ref(false)
    const fileTableRef = ref(null)
    const isInitializing = ref(false)

    // 格式化文件大小
    const formatSize = (bytes) => {
      if (!bytes || bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
    }

    // 加载配置
    const loadConfig = async () => {
      try {
        const saved = localStorage.getItem(STORAGE_KEY)
        if (saved) {
          const config = JSON.parse(saved)
          backupConfig.value = {
            ...backupConfig.value,
            ...config
          }
        } else {
          // 如果没有保存的配置，设置默认备份路径
          try {
            const defaultPath = await GetDefaultBackupPath()
            backupConfig.value.backupPath = defaultPath
          } catch (error) {
            console.error('获取默认备份路径失败:', error)
          }
        }
      } catch (e) {
        console.error('加载备份配置失败:', e)
      }
    }

    // 保存配置
    const saveConfig = () => {
      try {
        const configToSave = {
          autoBackup: backupConfig.value.autoBackup,
          backupCycle: backupConfig.value.backupCycle || 'every_exit',
          backupMode: backupConfig.value.backupMode || 'copy',
          backupPath: backupConfig.value.backupPath,
          selectedItems: backupConfig.value.selectedItems,
          lastBackupTime: backupConfig.value.lastBackupTime || null
        }
        localStorage.setItem(STORAGE_KEY, JSON.stringify(configToSave))
        
        // 发送事件通知App.vue
        window.dispatchEvent(new CustomEvent('backup-config-changed', {
          detail: configToSave
        }))
      } catch (e) {
        console.error('保存备份配置失败:', e)
      }
    }

    // 加载文件列表
    const loadFileList = async () => {
      loading.value = true
      try {
        const files = await ListConfigFiles()
        fileList.value = files || []
        
        // 等待DOM更新
        await new Promise(resolve => setTimeout(resolve, 100))
        
        // 设置默认选中项
        await setDefaultSelection()
      } catch (error) {
        console.error('加载文件列表失败:', error)
        ElMessage.error('加载文件列表失败: ' + (error.message || error))
      } finally {
        loading.value = false
      }
    }

    // 设置默认选择
    const setDefaultSelection = async () => {
      if (!fileTableRef.value || fileList.value.length === 0) {
        return
      }

      isInitializing.value = true
      
      try {
        // 清空当前选择
        fileTableRef.value.clearSelection()
        
        // 获取要选中的项
        const itemsToSelect = backupConfig.value.selectedItems && backupConfig.value.selectedItems.length > 0
          ? backupConfig.value.selectedItems
          : DEFAULT_ITEMS
        
        // 找到对应的行并选中
        itemsToSelect.forEach(itemName => {
          const row = fileList.value.find(f => f.name === itemName)
          if (row) {
            fileTableRef.value.toggleRowSelection(row, true)
          }
        })
        
        // 如果使用的是默认项，更新配置
        if (!backupConfig.value.selectedItems || backupConfig.value.selectedItems.length === 0) {
          backupConfig.value.selectedItems = [...DEFAULT_ITEMS]
          saveConfig()
        }
      } finally {
        isInitializing.value = false
      }
    }

    // 处理选择变化
    const handleSelectionChange = (selection) => {
      if (isInitializing.value) {
        return
      }
      
      selectedFiles.value = selection
      backupConfig.value.selectedItems = selection.map(item => item.name)
      saveConfig()
    }

    // 选择备份路径
    const selectBackupPath = async () => {
      try {
        const path = await GetOpenDir()
        if (path) {
          backupConfig.value.backupPath = path
          saveConfig()
          ElMessage.success('备份路径已更新')
        }
      } catch (error) {
        console.error('选择备份路径失败:', error)
        ElMessage.error('选择备份路径失败: ' + (error.message || error))
      }
    }

    // 执行备份
    const performBackup = async () => {
      if (!backupConfig.value.backupPath) {
        ElMessage.warning('请先选择备份保存路径')
        return
      }

      const itemsToBackup = selectedFiles.value.length > 0
        ? selectedFiles.value.map(item => item.name)
        : backupConfig.value.selectedItems

      if (!itemsToBackup || itemsToBackup.length === 0) {
        ElMessage.warning('请至少选择一个要备份的文件或文件夹')
        return
      }

      // 是否为覆盖模式
      const isOverwrite = backupConfig.value.backupMode === 'overwrite'
      
      try {
        // 第一次确认：普通确认
        await ElMessageBox.confirm(
          `确定要备份以下 ${itemsToBackup.length} 项到 ${backupConfig.value.backupPath} 吗？`,
          '确认备份',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'info'
          }
        )

        // 覆盖模式下进行手动输入的二次确认
        if (isOverwrite) {
          await ElMessageBox.prompt(
            '当前为【覆盖模式】，将直接覆盖备份目录中的同名文件和文件夹，可能导致历史备份无法恢复。\n\n如需继续，请在下方输入框中输入「覆盖」二字进行确认：',
            '危险操作确认',
            {
              confirmButtonText: '确认覆盖',
              cancelButtonText: '取消',
              inputPlaceholder: '请输入：覆盖',
              inputPattern: /^覆盖$/,
              inputErrorMessage: '请输入正确的确认文本：覆盖'
            }
          )
        }
      } catch {
        // 用户取消任意一步确认则终止备份
        return
      }

      backingUp.value = true
      try {
        await BackupConfigFiles(backupConfig.value.backupPath, itemsToBackup, isOverwrite)
        
        // 更新上次备份时间
        backupConfig.value.lastBackupTime = new Date().toISOString()
        saveConfig()
        
        ElMessage.success('备份完成！')
      } catch (error) {
        console.error('备份失败:', error)
        ElMessage.error('备份失败: ' + (error.message || error))
      } finally {
        backingUp.value = false
      }
    }

    onMounted(async () => {
      await loadConfig()
      await loadFileList()
    })

    return {
      backupConfig,
      fileList,
      selectedFiles,
      loading,
      backingUp,
      fileTableRef,
      formatSize,
      loadFileList,
      setDefaultSelection,
      handleSelectionChange,
      selectBackupPath,
      performBackup,
      saveConfig,
      FolderOpened,
      Download,
      Refresh,
      Check,
      Folder,
      Document
    }
  }
}
</script>

<style scoped>
.backup-manager {
  padding: 20px;
}

.backup-card {
  border: 1px solid rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.backup-form {
  padding: 20px 0;
}

.backup-items-container {
  width: 100%;
}

:deep(.el-table) {
  background-color: transparent;
}

:deep(.el-table th) {
  background-color: rgba(255, 255, 255, 0.05);
}

:deep(.el-table td) {
  background-color: transparent;
}

:deep(.el-table tr) {
  background-color: transparent;
}

:deep(.el-table--enable-row-hover .el-table__body tr:hover > td) {
  background-color: rgba(255, 255, 255, 0.1);
}
</style>

