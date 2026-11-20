<template>
  <div class="kv-table">
    <el-table :data="list" size="small" border>
      <el-table-column label="启用" width="60">
        <template #default="scope">
          <el-switch
            :model-value="scope.row.active"
            @update:model-value="val => handleChange(scope.$index, 'active', val)"
          />
        </template>
      </el-table-column>
      <el-table-column label="键" width="120">
        <template #default="scope">
          <env-var-input
            :model-value="scope.row.key"
            @update:model-value="val => handleChange(scope.$index, 'key', val)"
            :env-vars="envVars"
            placeholder="Key"
            size="small"
          />
        </template>
      </el-table-column>
      <el-table-column label="值">
        <template #default="scope">
          <env-var-input
            :model-value="scope.row.value"
            @update:model-value="val => handleChange(scope.$index, 'value', val)"
            :env-vars="envVars"
            placeholder="Value"
            size="small"
            class="url-input"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="50">
        <template #default="scope">
          <el-button
            type="text"
            size="small"
            @click="handleDelete(scope.$index)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-button
      class="mt-8"
      type="primary"
      link
      size="small"
      @click="handleAdd"
    >
      {{ addLabel }}
    </el-button>
  </div>
</template>

<script>
import { nanoid } from 'nanoid';
import EnvVarInput from './EnvVarInput.vue';

export default {
  name: 'KvTable',
  components: {
    EnvVarInput,
  },
  props: {
    list: {
      type: Array,
      required: true,
    },
    addLabel: {
      type: String,
      default: '新增',
    },
    envVars: {
      type: Object,
      default: () => ({}),
    },
  },
  emits: ['update:list'],
  methods: {
    handleAdd() {
      const next = [...this.list, { id: nanoid(), key: '', value: '', active: true }];
      this.$emit('update:list', next);
    },
    handleDelete(index) {
      const next = this.list.slice();
      next.splice(index, 1);
      this.$emit('update:list', next);
    },
    handleChange(index, field, value) {
      const next = this.list.slice();
      next[index] = { ...next[index], [field]: value };
      this.$emit('update:list', next);
    },
  },
};
</script>

<style scoped>
.mt-8 {
  margin-top: 8px;
}

.kv-table :deep(.env-var-input) {
  height: 34px;
  min-height: 34px;
  line-height: 20px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: clip;
}

/* 确保开关按钮正常显示 */
.kv-table :deep(.el-switch) {
  display: inline-block !important;
  visibility: visible !important;
}

.kv-table :deep(.el-switch__core) {
  display: inline-block !important;
  visibility: visible !important;
}

/* 确保删除按钮正常显示 */
.kv-table :deep(.el-button--text) {
  color: #409eff !important;
  display: inline-block !important;
  visibility: visible !important;
}

.kv-table :deep(.el-button--text:hover) {
  color: #66b1ff !important;
}

/* 确保新增按钮正常显示 */
.kv-table :deep(.el-button--primary) {
  display: inline-block !important;
  visibility: visible !important;
}

.kv-table .url-input {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  line-height: 20px;
  height: 32px;
  min-height: 32px;
  max-height: 32px;
  line-height: 20px;
  white-space: nowrap;      /* 不自动换行 */
  word-break: normal;       /* 避免强制断行 */
  overflow-x: auto;         /* 水平方向可滚动，支持鼠标选择时自动滚动 */
  overflow-y: hidden;       /* 垂直方向隐藏，保持单行输入 */
  user-select: text;       /* 支持鼠标选择文本 */
  scrollbar-width: none;   /* Firefox 隐藏滚动条 */
  -ms-overflow-style: none; /* IE/Edge 隐藏滚动条 */
}

/* Chrome/Safari 隐藏滚动条 */
.kv-table .url-input::-webkit-scrollbar {
  display: none;
}
</style>


