<template>
  <div class="software-center">
    <!-- 软件列表 -->
    <el-card class="table-card glass-card" shadow="hover">
      <div class="table-header">
        <div class="title-group">
          <div class="title-wrapper">
            <el-icon class="title-icon"><List /></el-icon>
            <span class="title">在线软件列表</span>
            <el-button
                type="primary"
                size="small"
                :icon="Refresh"
                @click="handleRefresh"
                :loading="loading.table"
                circle
            />
            <el-button
                type="info"
                size="small"
                :icon="Setting"
                @click="handleOpenSettings"
                circle
            />
          </div>
          <span class="subtitle">浏览和查看可用软件列表，发现更多实用工具</span>
        </div>
        <div class="search-box">
          <el-input
              v-model="searchKeyword"
              placeholder="搜索软件名称、描述、作者..."
              clearable
              style="width: 300px;"
              @input="handleSearch"
          >
            <template #prefix>
              <el-icon class="search-icon"><Search /></el-icon>
            </template>
          </el-input>
        </div>
      </div>
      <div class="table-wrapper">
        <el-table
            :data="paginatedSoftwareList"
            border
            stripe
            v-loading="loading.table"
            :height="tableHeight"
            class="software-table glass-card"
            :row-class-name="getRowClassName"
        >
        <el-table-column type="index" width="60" label="#" />
        <el-table-column prop="name" label="软件" min-width="120">
          <template #default="{ row }">
            <div class="name-cell">
              <span class="name-text">
                {{ row.name || row.id }}
                <sup
                    v-if="row.isNew"
                    class="name-new-sup"
                >
                  New
                </sup>
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="工具简介" min-width="400" show-overflow-tooltip />
        <el-table-column prop="toolType" label="工具类型" min-width="200">
          <template #default="{ row }">
            <div class="tag-list">
              <template v-if="getTagDisplayInfo(row.toolType).allTags.length">
                <el-tag
                    v-for="(item, index) in getTagDisplayInfo(row.toolType).display"
                    :key="`type-${index}-${item}`"
                    size="small"
                    type="warning"
                    effect="light"
                >
                  {{ item }}
                </el-tag>
                <el-tooltip
                    v-if="getTagDisplayInfo(row.toolType).remaining.length > 0"
                    :content="getTagDisplayInfo(row.toolType).remaining.join(', ')"
                    placement="top"
                >
                  <el-tag
                      size="small"
                      type="info"
                      effect="light"
                      class="more-tag"
                  >
                    +{{ getTagDisplayInfo(row.toolType).remaining.length }}
                  </el-tag>
                </el-tooltip>
              </template>
              <span v-else>-</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="toolLanguage" label="工具语言" min-width="180">
          <template #default="{ row }">
            <div class="tag-list">
              <template v-if="getTagDisplayInfo(row.toolLanguage).allTags.length">
                <el-tag
                    v-for="(item, index) in getTagDisplayInfo(row.toolLanguage).display"
                    :key="`lang-${index}-${item}`"
                    size="small"
                    type="success"
                    effect="light"
                >
                  {{ item }}
                </el-tag>
                <el-tooltip
                    v-if="getTagDisplayInfo(row.toolLanguage).remaining.length > 0"
                    :content="getTagDisplayInfo(row.toolLanguage).remaining.join(', ')"
                    placement="top"
                >
                  <el-tag
                      size="small"
                      type="success"
                      effect="light"
                      class="more-tag"
                  >
                    +{{ getTagDisplayInfo(row.toolLanguage).remaining.length }}
                  </el-tag>
                </el-tooltip>
              </template>
              <span v-else>-</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="版本" min-width="100">
          <template #default="{ row }">
            <span class="version-text">
              {{ row.latestVersion || '-' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="author" label="工具作者" min-width="100">
          <template #default="{ row }">
            <span>{{ row.author || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="lastUpdated" label="最近更新" min-width="100">
          <template #default="{ row }">
            <div class="last-updated-cell">
              <div class="last-updated-main">
                <span class="date-part">
                  {{ parseDateTime(row.lastUpdated).date }}
                </span>
              </div>
              <span
                  v-if="parseDateTime(row.lastUpdated).time"
                  class="time-part"
              >
                {{ parseDateTime(row.lastUpdated).time }}
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <div class="action-cell">
              <el-button 
                size="small" 
                type="primary" 
                :icon="Link"
                @click="openLink(row.homepage)"
              >
                地址
              </el-button>
            </div>
          </template>
        </el-table-column>
        </el-table>
      </div>
      <div class="table-footer">
        <el-pagination
            v-model:current-page="pagination.currentPage"
            v-model:page-size="pagination.pageSize"
            :total="filteredTotal"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            small
            background
            @current-change="handlePageChange"
            @size-change="handlePageSizeChange"
        />
      </div>
    </el-card>

    <!-- 设置软件清单地址对话框 -->
    <el-dialog
        v-model="dialogVisible"
        title="设置软件清单地址"
        width="600px"
        :close-on-click-modal="false"
    >
      <el-form :model="settingsForm" label-width="120px">
        <el-form-item label="清单地址">
          <el-input
              v-model="settingsForm.catalogURL"
              placeholder="请输入软件清单地址（JSON格式）"
              clearable
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveSettings" :loading="loading.save">
            保存并刷新
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, onUnmounted, reactive, ref, computed, nextTick } from 'vue';
import { Search, List, Link, Refresh, Setting } from '@element-plus/icons-vue';
import { ElMessage } from 'element-plus';
import { BrowserOpenURL } from '../../../wailsjs/runtime';
import {
  CheckSoftwareUpdates,
  GetSoftwareSettings,
  SaveSoftwareSettings
} from '../../../wailsjs/go/system/SoftwareCenter';

const allSoftwareList = ref([]); // 存储所有软件列表用于搜索
const searchKeyword = ref('');
const tableHeight = ref('100%');

const loading = reactive({
  table: false,
  save: false
});

const dialogVisible = ref(false);
const currentCatalogURL = ref('');
const settingsForm = reactive({
  catalogURL: ''
});

const pagination = reactive({
  pageSize: 10,
  currentPage: 1
});

// 计算过滤后的软件列表
const filteredSoftwareList = computed(() => {
  if (!searchKeyword.value.trim()) {
    return allSoftwareList.value;
  }
  
  const keyword = searchKeyword.value.toLowerCase().trim();
  return allSoftwareList.value.filter(item => {
    return (
      (item.name && item.name.toLowerCase().includes(keyword)) ||
      (item.description && item.description.toLowerCase().includes(keyword)) ||
      (item.author && item.author.toLowerCase().includes(keyword)) ||
      (item.toolType && item.toolType.toLowerCase().includes(keyword)) ||
      (item.toolLanguage && item.toolLanguage.toLowerCase().includes(keyword)) ||
      (item.id && item.id.toLowerCase().includes(keyword))
    );
  });
});

// 计算分页后的软件列表
const paginatedSoftwareList = computed(() => {
  const start = (pagination.currentPage - 1) * pagination.pageSize;
  const end = start + pagination.pageSize;
  return filteredSoftwareList.value.slice(start, end);
});

// 计算总数
const filteredTotal = computed(() => {
  return filteredSoftwareList.value.length;
});

const parseTags = (val) => {
  if (!val) return [];
  return String(val)
      .split(',')
      .map((v) => v.trim())
      .filter(Boolean);
};

// 获取显示的标签和剩余的标签（最多显示2个）
const getDisplayTags = (tags) => {
  if (!tags || tags.length === 0) {
    return { display: [], remaining: [] };
  }
  const maxDisplay = 2; // 最多显示2个标签
  if (tags.length <= maxDisplay) {
    return { display: tags, remaining: [] };
  }
  return {
    display: tags.slice(0, maxDisplay),
    remaining: tags.slice(maxDisplay)
  };
};

// 获取标签显示信息（包含所有标签、显示标签和剩余标签）
const getTagDisplayInfo = (val) => {
  if (!val) {
    return {
      allTags: [],
      display: [],
      remaining: []
    };
  }
  const allTags = parseTags(val);
  const result = getDisplayTags(allTags);
  return {
    allTags,
    display: result.display,
    remaining: result.remaining
  };
};

const loadAllSoftware = async () => {
  loading.table = true;
  try {
    // 直接获取远端软件列表，返回的就是纯数组，格式严格遵守 software.json 中定义的字段
    const list = await CheckSoftwareUpdates();
    allSoftwareList.value = Array.isArray(list) ? list : [];
    
    // 检查是否有新增软件，如果有则提示用户
    const newSoftwareCount = allSoftwareList.value.filter(item => item.isNew).length;
    if (newSoftwareCount > 0) {
      ElMessage({
        message: `发现 ${newSoftwareCount} 个新增软件，已高亮显示`,
        type: 'info',
        duration: 4000,
        showClose: true
      });
    }
    
    // 应用搜索和分页
    handleSearch();
  } catch (error) {
    ElMessage.error('读取软件列表失败');
    console.error(error);
  } finally {
    loading.table = false;
  }
};

const handleSearch = () => {
  // 搜索时重置到第一页
  pagination.currentPage = 1;
};

const handleRefresh = async () => {
  await loadAllSoftware();
  ElMessage.success('刷新成功');
};

const handleOpenSettings = async () => {
  try {
    const settings = await GetSoftwareSettings();
    currentCatalogURL.value = settings.catalogURL || '';
    settingsForm.catalogURL = settings.catalogURL || '';
    dialogVisible.value = true;
  } catch (error) {
    ElMessage.error('获取配置失败');
    console.error(error);
  }
};

const handleSaveSettings = async () => {
  loading.save = true;
  try {
    // 获取当前配置
    const currentSettings = await GetSoftwareSettings();
    
    // 更新配置
    const newSettings = {
      ...currentSettings,
      catalogURL: settingsForm.catalogURL.trim()
    };
    
    // 保存配置
    await SaveSoftwareSettings(newSettings);
    
    ElMessage.success('配置保存成功，正在刷新软件列表...');
    dialogVisible.value = false;
    
    // 重新加载软件列表
    await loadAllSoftware();
    ElMessage.success('刷新成功');
  } catch (error) {
    ElMessage.error('保存配置失败');
    console.error(error);
  } finally {
    loading.save = false;
  }
};

const openLink = (url) => {
  if (url) {
    BrowserOpenURL(url);
  }
};

const parseDateTime = (val) => {
  if (!val) {
    return {
      date: '-',
      time: ''
    };
  }

  let s = String(val).trim();

  // 兼容 ISO 格式：2025-01-01T12:00:00Z
  s = s.replace('T', ' ');

  const [datePart, timePart = ''] = s.split(' ');

  return {
    date: datePart || '-',
    time: timePart.slice(0, 8) // 只保留到时分秒
  };
};

const handlePageChange = (page) => {
  pagination.currentPage = page;
};

const handlePageSizeChange = (size) => {
  pagination.pageSize = size;
  pagination.currentPage = 1;
};

const calcTableHeight = () => {
  nextTick(() => {
    const wrapper = document.querySelector('.table-wrapper');
    if (wrapper) {
      tableHeight.value = wrapper.clientHeight + 'px';
    }
  });
};

// 为表格行添加类名，用于高亮显示新增软件
const getRowClassName = ({ row }) => {
  if (row.isNew) {
    return 'row-is-new';
  }
  return '';
};

onMounted(async () => {
  // 加载配置以显示当前清单地址
  try {
    const settings = await GetSoftwareSettings();
    currentCatalogURL.value = settings.catalogURL || '';
  } catch (error) {
    console.error('加载配置失败:', error);
  }
  
  await loadAllSoftware();
  calcTableHeight();
  window.addEventListener('resize', calcTableHeight);
});

onUnmounted(() => {
  window.removeEventListener('resize', calcTableHeight);
});
</script>

<style scoped>
.software-center {
  display: flex;
  flex-direction: column;
  height: 90vh;
  padding: 2px;
  background: #f5f7fa;
  box-sizing: border-box;
  overflow: hidden;
}

/* 页面头部样式 */
.page-header {
  background: #ffffff;
  border-radius: 8px;
  padding: 20px 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.header-text {
  flex: 1;
}

/* 卡片样式 */
.table-card {
  background: #ffffff;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.table-card :deep(.el-card__body) {
  padding: 20px;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

/* 表格头部样式 */
.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e4e7ed;
}

.title-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.title-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
}

.title-icon {
  color: #409eff;
  font-size: 18px;
}

.title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.subtitle {
  color: #909399;
  font-size: 12px;
  margin-left: 26px;
}

/* 搜索框样式 */
.search-box {
  display: flex;
  align-items: center;
  gap: 10px;
}

.search-icon {
  color: #909399;
}

/* 表格容器 */
.table-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
  height: 100%;
}

/* 表格样式 */
.software-table {
  border-radius: 6px;
  overflow: hidden;
  flex: 1;
  height: 100%;
}

.software-table :deep(.el-table) {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.software-table :deep(.el-table__body-wrapper) {
  flex: 1;
  overflow-y: auto;
}

.software-table :deep(.el-table__header) {
  background: #fafafa;
}

.software-table :deep(.el-table__header th) {
  background: #fafafa;
  color: #303133;
  font-weight: 600;
  font-size: 14px;
  border-bottom: 1px solid #e4e7ed;
  padding: 12px 0;
}

.software-table :deep(.el-table__body tr:hover) {
  background: #f5f7fa;
}

.software-table :deep(.el-table__body td) {
  padding: 12px 0;
  border-bottom: 1px solid #f0f2f5;
}

.software-table :deep(.el-table--border) {
  border: 1px solid #e4e7ed;
}

.software-table :deep(.el-table--border::after) {
  background-color: #e4e7ed;
}

.software-table :deep(.el-table--border::before) {
  background-color: #e4e7ed;
}

/* 名称单元格 */
.name-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  color: #303133;
}

.name-text {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.name-new-sup {
  margin-left: 4px;
  font-size: 10px;
  color: #f56c6c;
  font-weight: 600;
}

.name-cell :deep(.el-tag) {
  border-radius: 4px;
  font-weight: 500;
  padding: 2px 8px;
}

/* 版本单元格 */
.version-text {
  color: #67c23a;
  font-weight: 500;
  font-size: 13px;
}

.version-text.highlight {
  color: #f56c6c;
  font-weight: 600;
}

/* 标签样式 */
.software-table :deep(.el-tag) {
  border-radius: 4px;
  font-weight: 500;
  padding: 4px 10px;
}

/* 标签列表容器 - 限制行高，不换行 */
.tag-list {
  display: flex;
  align-items: center;
  gap: 6px;
  max-height: 32px;
  overflow: hidden;
  white-space: nowrap;
  flex-wrap: nowrap;
}

/* 更多标签样式 */
.more-tag {
  cursor: pointer;
  user-select: none;
  color: #409eff;
  font-size: 12px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 4px;
  background-color: #f0f2f5;
  border: 1px solid #e4e7ed;
  border-color: #409eff;
}

/* 操作按钮 */
.action-cell {
  display: flex;
  justify-content: center;
}

.action-cell :deep(.el-button) {
  border-radius: 4px;
  font-weight: 500;
  padding: 6px 14px;
}

/* 链接样式 */
.software-table :deep(.el-link) {
  font-weight: 500;
}

/* 分页样式 */
.table-footer {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
  padding-top: 12px;
  border-top: 1px solid #e4e7ed;
}

.table-footer :deep(.el-pagination) {
  --el-pagination-button-color: #606266;
  --el-pagination-hover-color: #409eff;
  --el-pagination-bg-color: #fff;
}

.table-footer :deep(.el-pagination .btn-next),
.table-footer :deep(.el-pagination .btn-prev) {
  border-radius: 4px;
}

.table-footer :deep(.el-pagination .btn-next:hover),
.table-footer :deep(.el-pagination .btn-prev:hover) {
  background: #409eff;
  color: white;
}

.table-footer :deep(.el-pagination .number) {
  border-radius: 4px;
}

.table-footer :deep(.el-pagination .number:hover) {
  background: #409eff;
  color: white;
}

.table-footer :deep(.el-pagination .number.is-active) {
  background: #409eff;
  color: white;
  font-weight: 600;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .table-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .page-header {
    padding: 16px;
  }

  .header-content {
    flex-direction: column;
    align-items: flex-start;
  }
}

/* 加载动画优化 */
.software-table :deep(.el-loading-mask) {
  background-color: rgba(255, 255, 255, 0.9);
}

.software-table :deep(.el-loading-spinner) {
  margin-top: -25px;
}

.software-table :deep(.el-loading-spinner .circular) {
  width: 50px;
  height: 50px;
}

/* 新增软件行高亮样式 */
.software-table :deep(.row-is-new) {
  background-color: #f0f9ff !important;
  border-left: 3px solid #409eff;
}

.software-table :deep(.row-is-new:hover) {
  background-color: #e0f2fe !important;
}

.software-table :deep(.row-is-new td) {
  background-color: transparent !important;
}

/* 最近更新时间样式 */
.last-updated-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
}

.last-updated-main {
  display: flex;
  align-items: center;
  gap: 6px;
}

.new-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 6px;
  height: 18px;
  border-radius: 10px;
  background-color: #f56c6c;
  color: #fff;
  font-size: 11px;
  font-weight: 600;
}

.date-part {
  color: #606266;
  font-size: 13px;
}

.time-part {
  color: #409eff;
  font-size: 12px;
}
</style>

