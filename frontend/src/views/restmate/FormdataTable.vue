<template>
  <div class="kv-table">
    <el-table :data="list" size="small" border>
      <el-table-column label="启用" width="50">
        <template #default="scope">
          <el-switch
            :model-value="scope.row.active"
            @update:model-value="val => handleChange(scope.$index, 'active', val)"
          />
        </template>
      </el-table-column>
      <el-table-column label="类型" width="75" v-if="allowFiles">
        <template #default="scope">
          <el-select
            v-model="scope.row.type"
            size="small"
            @change="val => handleChange(scope.$index, 'type', val)"
          >
            <el-option label="文本" value="text" />
            <el-option label="文件" value="file" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="键" width="120">
        <template #default="scope">
          <el-input
            v-model="scope.row.key"
            size="small"
            placeholder="Key"
            @input="val => handleChange(scope.$index, 'key', val)"
          />
        </template>
      </el-table-column>
      <el-table-column :label="allowFiles ? '值 / 文件' : '值'">
        <template #default="scope">
          <template v-if="!allowFiles || scope.row.type === 'text'">
            <el-input
              v-model="scope.row.value"
              size="small"
              placeholder="Value"
              @input="val => handleChange(scope.$index, 'value', val)"
            />
          </template>
          <template v-else>
            <div class="file-cell">
              <span v-if="scope.row.files && scope.row.files.length" class="file-paths">
                {{ scope.row.files.join(', ') }}
              </span>
              <el-button
                size="small"
                type="primary"
                link
                @click="$emit('choose-file', scope.$index)"
              >
                选择文件
              </el-button>
            </div>
          </template>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="80">
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
    <div class="mt-8">
      <el-button type="primary" link size="small" @click="handleAdd('text')">
        新增文本字段
      </el-button>
      <el-button v-if="allowFiles" type="primary" link size="small" @click="handleAdd('file')">
        新增文件字段
      </el-button>
    </div>
  </div>
</template>

<script>
import { nanoid } from 'nanoid';

export default {
  name: 'FormdataTable',
  props: {
    list: {
      type: Array,
      required: true,
    },
    allowFiles: {
      type: Boolean,
      default: true,
    },
  },
  emits: ['update:list', 'choose-file'],
  methods: {
    handleAdd(type = 'text') {
      const next = [
        ...this.list,
        { id: nanoid(), key: '', value: '', files: [], type, active: true },
      ];
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

.file-cell {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
}

.file-paths {
  font-size: 12px;
  color: #606266;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
</style>


