<template>
  <div class="fingerprint-manage-panel">
    <div class="panel-header">
      <div>
        <div class="title">自定义指纹管理</div>
        <div class="subtitle">管理手动添加的指纹规则（支持导入EHole指纹库）</div>
      </div>
      <div class="header-actions">
        <el-button :icon="Upload" @click="handleImportClick">导入JSON</el-button>
        <el-button type="primary" :icon="Plus" @click="showAddDialog">添加指纹</el-button>
      </div>
      <input 
        ref="fileInputRef" 
        type="file" 
        accept=".json" 
        style="display: none" 
        @change="handleFileImport"
      />
    </div>

    <div class="table-wrapper" ref="tableWrapperRef">
      <el-table :data="paginatedFingerprints" stripe border v-loading="loading" :height="tableHeight">
      <el-table-column 
        type="index" 
        label="#" 
        width="60"
        :index="getTableIndex"
      />
      <el-table-column prop="cms" min-width="150">
        <template #header>
          <TableColumnFilterHeader v-model="columnFilters.cms" label="识别名称" @filter-change="handleFingerColFilter" />
        </template>
      </el-table-column>
      <el-table-column prop="method" width="120">
        <template #header>
          <TableColumnFilterHeader v-model="columnFilters.method" label="匹配方式" @filter-change="handleFingerColFilter" />
        </template>
        <template #default="scope">
          <el-tag size="small">{{ getMethodLabel(scope.row.method) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="location" width="120">
        <template #header>
          <TableColumnFilterHeader v-model="columnFilters.location" label="匹配位置" @filter-change="handleFingerColFilter" />
        </template>
        <template #default="scope">
          <el-tag size="small" type="info">{{ scope.row.location }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="keyword" min-width="200">
        <template #header>
          <TableColumnFilterHeader v-model="columnFilters.keyword" label="关键词/哈希" @filter-change="handleFingerColFilter" />
        </template>
        <template #default="scope">
          <div class="keyword-list">
            <el-tag v-for="(kw, idx) in scope.row.keyword" :key="idx" size="small" class="keyword-tag">
              {{ kw }}
            </el-tag>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150" align="center" fixed="right">
        <template #default="scope">
          <el-button size="small" :icon="Edit" type="primary" link @click="editFingerprint(scope.row)">编辑</el-button>
          <el-button size="small" :icon="Delete" type="danger" link @click="deleteFingerprint(scope.row)">删除</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="暂无自定义指纹，点击上方按钮添加" />
      </template>
      </el-table>
    </div>

    <!-- 分页组件 -->
    <div class="pagination-wrapper" v-if="filteredFingerprints.length > 0">
      <el-pagination small
        v-model:current-page="pagination.currentPage"
        v-model:page-size="pagination.pageSize"
        :total="filteredFingerprints.length"
        :page-sizes="[10, 20, 50, 100, 200]"
        layout="total, sizes, prev, pager, next, jumper"
        background
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>

    <!-- 添加/编辑指纹对话框 -->
    <el-dialog 
      v-model="dialogVisible" 
      :title="editingItem ? '编辑指纹规则' : '新增指纹规则'" 
      width="520px"
      :close-on-click-modal="false"
      class="fingerprint-dialog"
      @closed="handleDialogClosed"
    >
      <el-form :model="fingerForm" label-width="100px" size="default" ref="fingerFormRef">
        <el-form-item label="识别名称" prop="cms" :rules="[{ required: true, message: '请输入识别名称', trigger: 'blur' }]">
          <el-input v-model="fingerForm.cms" placeholder="如：自定义系统名称" />
        </el-form-item>
        <el-form-item label="匹配方式" prop="method">
          <el-select v-model="fingerForm.method" style="width: 180px">
            <el-option label="关键字" value="keyword" />
            <el-option label="正则" value="regex" />
            <el-option label="Favicon Hash" value="faviconhash" />
          </el-select>
        </el-form-item>
        <el-form-item label="匹配位置" prop="location">
          <el-select v-model="fingerForm.location" style="width: 180px">
            <el-option label="body" value="body" />
            <el-option label="header" value="header" />
            <el-option label="title" value="title" />
            <el-option label="favicon" value="favicon" />
          </el-select>
        </el-form-item>
        <el-form-item 
          label="关键词/哈希" 
          prop="keywords"
          :rules="[{ required: true, message: '请至少填写一个关键词或哈希', trigger: 'blur', validator: validateKeywords }]"
        >
          <el-input
            v-model="fingerForm.keywords"
            type="textarea"
            :rows="4"
            placeholder="每行一个关键词或 favicon 哈希"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveFingerprint">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted, nextTick, reactive } from 'vue'
import { Plus, Edit, Delete, Upload } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ListCustomFingerRules, AddCustomFingerRule, ClearCustomFingerRules } from '../../../wailsjs/go/finger/FingerprintController'
import TableColumnFilterHeader from '@/components/TableColumnFilterHeader.vue'

export default {
  name: 'FingerprintManagePanel',
  components: { TableColumnFilterHeader },
  emits: ['close', 'refresh'],
  setup(props, { emit }) {
    const allFingerprints = ref([]) // 存储所有指纹数据
    const loading = ref(false)
    const pagination = ref({
      currentPage: 1,
      pageSize: 10
    })
    const dialogVisible = ref(false)
    const fingerForm = ref({
      cms: '',
      method: 'keyword',
      location: 'body',
      keywords: ''
    })
    const editingItem = ref(null)
    const saving = ref(false)
    const fingerFormRef = ref(null)
    const fileInputRef = ref(null)
    const tableWrapperRef = ref(null)
    const tableHeight = ref('400px')
    const columnFilters = reactive({
      cms: '',
      method: '',
      location: '',
      keyword: ''
    })

    const getMethodLabel = (method) => {
      const map = {
        keyword: '关键字',
        regex: '正则',
        faviconhash: 'Favicon Hash'
      }
      return map[method] || method
    }

    const validateKeywords = (rule, value, callback) => {
      const keywordList = (value || '')
        .split('\n')
        .map(k => k.trim())
        .filter(Boolean)
      if (keywordList.length === 0) {
        callback(new Error('请至少填写一个关键词或哈希'))
      } else {
        callback()
      }
    }

    const filteredFingerprints = computed(() => {
      const rows = allFingerprints.value || []
      const c = (columnFilters.cms || '').trim().toLowerCase()
      const m = (columnFilters.method || '').trim().toLowerCase()
      const l = (columnFilters.location || '').trim().toLowerCase()
      const k = (columnFilters.keyword || '').trim().toLowerCase()
      if (!c && !m && !l && !k) return rows
      return rows.filter((row) => {
        if (c && !String(row.cms || '').toLowerCase().includes(c)) return false
        if (m) {
          const label = getMethodLabel(row.method).toLowerCase()
          const raw = String(row.method || '').toLowerCase()
          if (!label.includes(m) && !raw.includes(m)) return false
        }
        if (l && !String(row.location || '').toLowerCase().includes(l)) return false
        if (k) {
          const joined = (Array.isArray(row.keyword) ? row.keyword : []).join(' ').toLowerCase()
          if (!joined.includes(k)) return false
        }
        return true
      })
    })

    const handleFingerColFilter = () => {
      pagination.value.currentPage = 1
    }

    // 计算属性：获取当前页的数据
    const paginatedFingerprints = computed(() => {
      const list = filteredFingerprints.value
      const start = (pagination.value.currentPage - 1) * pagination.value.pageSize
      const end = start + pagination.value.pageSize
      return list.slice(start, end)
    })

    const loadFingerprints = async () => {
      loading.value = true
      try {
        const list = await ListCustomFingerRules()
        allFingerprints.value = list || []
        // 如果当前页没有数据，跳转到第一页
        const maxPage = Math.max(1, Math.ceil(filteredFingerprints.value.length / pagination.value.pageSize) || 1)
        if (pagination.value.currentPage > maxPage) {
          pagination.value.currentPage = 1
        }
      } catch (e) {
        console.error(e)
        ElMessage.error('加载指纹列表失败: ' + (e?.message || e))
      } finally {
        loading.value = false
        // 数据加载完成后更新表格高度
        setTimeout(() => {
          updateTableHeight()
        }, 100)
      }
    }

    const handleSizeChange = (size) => {
      pagination.value.pageSize = size
      pagination.value.currentPage = 1
      // 分页变化后更新表格高度
      setTimeout(() => {
        updateTableHeight()
      }, 50)
    }

    const handleCurrentChange = (page) => {
      pagination.value.currentPage = page
      // 分页变化后更新表格高度
      setTimeout(() => {
        updateTableHeight()
      }, 50)
    }

    const getTableIndex = (index) => {
      return (pagination.value.currentPage - 1) * pagination.value.pageSize + index + 1
    }

    const resetForm = () => {
      fingerForm.value = {
        cms: '',
        method: 'keyword',
        location: 'body',
        keywords: ''
      }
      editingItem.value = null
      if (fingerFormRef.value) {
        fingerFormRef.value.clearValidate()
      }
    }

    const showAddDialog = () => {
      resetForm()
      dialogVisible.value = true
    }

    const editFingerprint = (fingerprint) => {
      editingItem.value = fingerprint
      fingerForm.value = {
        cms: fingerprint.cms || '',
        method: fingerprint.method || 'keyword',
        location: fingerprint.location || 'body',
        keywords: (fingerprint.keyword || []).join('\n')
      }
      dialogVisible.value = true
    }

    const handleDialogClosed = () => {
      resetForm()
    }

    const deleteFingerprint = async (fingerprint) => {
      try {
        await ElMessageBox.confirm(
          `将删除指纹规则「${fingerprint.cms || ''}」，不可恢复。`,
          '删除指纹',
          { type: 'warning' }
        )
      } catch {
        return
      }
      const deletedItem = { ...fingerprint }
      const newList = allFingerprints.value.filter(item => {
        return !(item.cms === fingerprint.cms && 
                 item.method === fingerprint.method && 
                 item.location === fingerprint.location)
      })
      
      // 立即更新前端数据
      allFingerprints.value = newList
      
      // 如果当前页没有数据了，跳转到上一页
      if (paginatedFingerprints.value.length === 0 && pagination.value.currentPage > 1) {
        pagination.value.currentPage--
      }
      
      ElMessage.success('指纹已删除')
      
      // 异步执行后端操作，不阻塞UI
      try {
        // 由于后端没有批量更新接口，我们使用ClearCustomFingerRules + 重新添加的方式
        // 先清空
        await ClearCustomFingerRules()
        
        // 重新添加除删除项外的所有项
        for (const item of newList) {
          await AddCustomFingerRule({
            CMS: item.cms,
            Method: item.method,
            Location: item.location,
            Keyword: item.keyword || []
          })
        }
        
        // 触发父组件刷新
        emit('refresh')
      } catch (e) {
        console.error(e)
        // 如果后端操作失败，重新加载列表以恢复状态
        await loadFingerprints()
        ElMessage.error('删除失败: ' + (e?.message || e) + '，已恢复数据')
      }
    }

    const saveFingerprint = async () => {
      if (!fingerFormRef.value) return
      
      try {
        await fingerFormRef.value.validate()
      } catch (e) {
        return
      }

      const keywordList = fingerForm.value.keywords
        .split('\n')
        .map(k => k.trim())
        .filter(Boolean)
      
      if (keywordList.length === 0) {
        ElMessage.warning('请至少填写一个关键词/哈希')
        return
      }

      saving.value = true
      try {
        const rule = {
          CMS: fingerForm.value.cms.trim(),
          Method: fingerForm.value.method,
          Location: fingerForm.value.location,
          Keyword: keywordList
        }

        if (editingItem.value) {
          // 编辑：先删除旧项，再添加新项
          // 由于后端限制，需要清空后重新添加所有项（除了被编辑的项）
          const oldList = allFingerprints.value.filter(item => {
            return !(item.cms === editingItem.value.cms && 
                     item.method === editingItem.value.method && 
                     item.location === editingItem.value.location)
          })
          
          // 清空后重新添加
          await ClearCustomFingerRules()
          for (const item of oldList) {
            await AddCustomFingerRule({
              CMS: item.cms,
              Method: item.method,
              Location: item.location,
              Keyword: item.keyword || []
            })
          }
        }

        await AddCustomFingerRule(rule)
        
        // 重新加载列表以获取最新数据
        await loadFingerprints()
        
        ElMessage.success(editingItem.value ? '指纹已更新' : '指纹已添加')
        dialogVisible.value = false
        
        // 触发父组件刷新
        emit('refresh')
      } catch (e) {
        console.error(e)
        ElMessage.error(e?.message || (editingItem.value ? '更新失败' : '添加失败'))
      } finally {
        saving.value = false
      }
    }

    const handleImportClick = () => {
      if (fileInputRef.value) {
        fileInputRef.value.click()
      }
    }

    const handleFileImport = async (event) => {
      const file = event.target.files?.[0]
      if (!file) {
        return
      }

      // 重置文件选择器
      if (fileInputRef.value) {
        fileInputRef.value.value = ''
      }

      // 检查文件类型
      if (!file.name.toLowerCase().endsWith('.json')) {
        ElMessage.error('请选择JSON格式的文件')
        return
      }

      try {
        loading.value = true
        const text = await file.text()
        let jsonData
        
        try {
          jsonData = JSON.parse(text)
        } catch (e) {
          ElMessage.error('JSON文件格式错误: ' + (e?.message || '无法解析JSON'))
          return
        }

        // 检查JSON结构
        let fingerprintArray = []
        if (jsonData.fingerprint && Array.isArray(jsonData.fingerprint)) {
          fingerprintArray = jsonData.fingerprint
        } else if (Array.isArray(jsonData)) {
          fingerprintArray = jsonData
        } else {
          ElMessage.error('JSON格式不正确，应包含 fingerprint 数组或直接为数组格式')
          return
        }

        if (fingerprintArray.length === 0) {
          ElMessage.warning('JSON文件中没有指纹规则')
          return
        }

        // 确认导入
        const result = await ElMessageBox.confirm(
          `检测到 ${fingerprintArray.length} 条指纹规则，是否导入？\n注意：这将追加到现有指纹列表中。`,
          '确认导入',
          {
            confirmButtonText: '确定导入',
            cancelButtonText: '取消',
            type: 'info'
          }
        ).catch(() => false)

        if (!result) {
          return
        }

        // 批量导入指纹
        let successCount = 0
        let failCount = 0
        const errors = []

        for (const item of fingerprintArray) {
          try {
            // 验证必要字段
            if (!item.cms || !item.method || !item.location) {
              errors.push(`跳过无效规则: ${JSON.stringify(item)} - 缺少必要字段`)
              failCount++
              continue
            }

            // 处理keyword字段：可能是数组或单个值
            let keywords = []
            if (item.keyword) {
              if (Array.isArray(item.keyword)) {
                keywords = item.keyword.filter(k => k && k.toString().trim())
              } else {
                keywords = [item.keyword.toString().trim()].filter(Boolean)
              }
            }

            if (keywords.length === 0) {
              errors.push(`跳过无效规则: ${item.cms} - 关键词为空`)
              failCount++
              continue
            }

            // 添加指纹规则
            await AddCustomFingerRule({
              CMS: item.cms.toString().trim(),
              Method: item.method.toString().trim(),
              Location: item.location.toString().trim(),
              Keyword: keywords
            })

            successCount++
          } catch (e) {
            failCount++
            errors.push(`导入失败: ${item.cms || '未知'} - ${e?.message || e}`)
            console.error('导入指纹失败:', item, e)
          }
        }

        // 重新加载列表
        await loadFingerprints()

        // 显示导入结果
        if (errors.length > 0 && errors.length <= 10) {
          ElMessageBox.alert(
            `导入完成！\n成功: ${successCount} 条\n失败: ${failCount} 条\n\n失败详情：\n${errors.join('\n')}`,
            '导入结果',
            { type: failCount === 0 ? 'success' : 'warning' }
          )
        } else if (errors.length > 10) {
          ElMessageBox.alert(
            `导入完成！\n成功: ${successCount} 条\n失败: ${failCount} 条\n\n失败项过多，请查看控制台。`,
            '导入结果',
            { type: failCount === 0 ? 'success' : 'warning' }
          )
          console.error('导入失败的详细信息:', errors)
        } else {
          ElMessage.success(`成功导入 ${successCount} 条指纹规则`)
        }

        // 触发父组件刷新
        emit('refresh')
      } catch (e) {
        console.error('导入失败:', e)
        ElMessage.error('导入失败: ' + (e?.message || e))
      } finally {
        loading.value = false
      }
    }

    const updateTableHeight = () => {
      nextTick(() => {
        if (tableWrapperRef.value) {
          const height = tableWrapperRef.value.clientHeight
          if (height > 0) {
            // 减去一些边距，确保表格不会溢出
            tableHeight.value = `${Math.max(height - 2, 200)}px`
          }
        }
      })
    }

    onMounted(() => {
      loadFingerprints()
      // 延迟更新表格高度，确保 DOM 已渲染
      setTimeout(() => {
        updateTableHeight()
      }, 300)
      // 监听窗口大小变化
      window.addEventListener('resize', updateTableHeight)
    })

    onUnmounted(() => {
      window.removeEventListener('resize', updateTableHeight)
    })

    return {
      allFingerprints,
      columnFilters,
      filteredFingerprints,
      handleFingerColFilter,
      paginatedFingerprints,
      pagination,
      loading,
      dialogVisible,
      fingerForm,
      editingItem,
      saving,
      fingerFormRef,
      fileInputRef,
      tableWrapperRef,
      tableHeight,
      Plus,
      Edit,
      Delete,
      Upload,
      getMethodLabel,
      validateKeywords,
      showAddDialog,
      editFingerprint,
      deleteFingerprint,
      saveFingerprint,
      handleDialogClosed,
      handleImportClick,
      handleFileImport,
      handleSizeChange,
      handleCurrentChange,
      getTableIndex
    }
  }
}
</script>

<style scoped>
.fingerprint-manage-panel {
  padding: 0;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 0 10px;
  flex-shrink: 0;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.title {
  font-size: 16px;
  font-weight: 600;
  color: #1f2d3d;
}

.subtitle {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.keyword-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.keyword-tag {
  margin: 2px 0;
}

.table-wrapper {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  padding: 0 10px;
}

.fingerprint-manage-panel :deep(.el-table) {
  font-size: 12px;
}
.fingerprint-manage-panel :deep(.el-table th.el-table__cell) {
  background-color: #f5f7fa;
  padding: 8px 0;
  font-weight: 600;
}
.fingerprint-manage-panel :deep(.el-table td.el-table__cell) {
  padding: 6px 0;
}
.fingerprint-manage-panel :deep(.el-table__empty-block) {
  padding: 60px 0;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 8px 10px;
  margin-top: 16px;
  border-top: 1px solid #e4e7ed;
  flex-shrink: 0;
}
</style>

<style>
/* 指纹管理对话框样式，限制高度避免多个滚动条，基于父容器（标签页对话框） */
.fingerprint-manage-panel .fingerprint-dialog .el-dialog {
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.fingerprint-manage-panel .fingerprint-dialog .el-dialog__header {
  flex-shrink: 0;
  padding: 20px 20px 10px;
  border-bottom: 1px solid #e4e7ed;
}

.fingerprint-manage-panel .fingerprint-dialog .el-dialog__body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  min-height: 0;
  max-height: calc(80vh - 130px);
}

.fingerprint-manage-panel .fingerprint-dialog .el-dialog__footer {
  flex-shrink: 0;
  padding: 10px 20px 20px;
  border-top: 1px solid #e4e7ed;
}

.fingerprint-manage-panel .fingerprint-dialog .el-form {
  max-height: 100%;
  overflow: visible;
}

/* 当对话框在父对话框内时，限制其高度不超过父容器 */
.el-dialog__body .fingerprint-manage-panel .fingerprint-dialog .el-dialog {
  max-height: 70vh;
}

.el-dialog__body .fingerprint-manage-panel .fingerprint-dialog .el-dialog__body {
  max-height: calc(70vh - 130px);
}
</style>

