<template>
  <div class="cyber-col-header" :class="{ 'cyber-col-header--right': alignRight }">
    <div class="cyber-col-header__main">
      <span class="cyber-col-header__label" :class="{ 'has-filter': hasFilter }">{{ label }}</span>
      <el-popover :placement="placement" :width="popoverWidth" trigger="click">
        <template #reference>
          <el-button type="primary" link class="cyber-col-header__search-btn" @click.stop :title="searchTitle">
            <el-icon><Search /></el-icon>
          </el-button>
        </template>
        <el-input
          :model-value="modelValue"
          size="small"
          clearable
          :placeholder="placeholder"
          @update:model-value="onInput"
        />
      </el-popover>
    </div>
    <span v-if="sortable" class="cyber-col-header__sort">
      <el-icon
        class="cyber-col-header__sort-ico"
        :class="{ active: sortAscActive }"
        title="升序"
        @click.stop="$emit('sort-asc')"
      ><ArrowUp /></el-icon>
      <el-icon
        class="cyber-col-header__sort-ico"
        :class="{ active: sortDescActive }"
        title="降序"
        @click.stop="$emit('sort-desc')"
      ><ArrowDown /></el-icon>
    </span>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Search, ArrowUp, ArrowDown } from '@element-plus/icons-vue'

const props = defineProps({
  label: { type: String, required: true },
  modelValue: { type: String, default: '' },
  /** 筛选框占位：默认留空 */
  placeholder: { type: String, default: '' },
  popoverWidth: { type: Number, default: 220 },
  placement: { type: String, default: 'bottom-start' },
  alignRight: Boolean,
  sortable: Boolean,
  sortAscActive: Boolean,
  sortDescActive: Boolean,
  searchTitle: { type: String, default: '列筛选' }
})

const emit = defineEmits(['update:modelValue', 'filter-change', 'sort-asc', 'sort-desc'])

const hasFilter = computed(() => !!(props.modelValue || '').trim())

function onInput(v) {
  emit('update:modelValue', v)
  emit('filter-change')
}
</script>

<style scoped>
.cyber-col-header {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: center;
  gap: 2px;
  width: 100%;
  min-width: 0;
  min-height: 30px;
  box-sizing: border-box;
}

.cyber-col-header__main {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: center;
  gap: 4px;
  flex: 1;
  min-width: 0;
}

.cyber-col-header--right {
  justify-content: flex-end;
}

.cyber-col-header--right .cyber-col-header__main {
  justify-content: flex-end;
}

.cyber-col-header__label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: inherit;
}

.cyber-col-header__label.has-filter {
  color: var(--el-color-primary);
  font-weight: 600;
}

.cyber-col-header__search-btn {
  flex-shrink: 0;
  padding: 2px 4px;
  margin: 0;
}

.cyber-col-header__sort {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0;
  line-height: 1;
  flex-shrink: 0;
  width: 14px;
}

.cyber-col-header__sort-ico {
  font-size: 12px;
  cursor: pointer;
  color: var(--el-text-color-secondary);
  transition: color 0.15s;
}

.cyber-col-header__sort-ico:hover {
  color: var(--el-color-primary);
}

.cyber-col-header__sort-ico.active {
  color: var(--el-color-primary);
}
</style>
