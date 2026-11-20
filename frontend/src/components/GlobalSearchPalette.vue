<template>
  <teleport to="body">
    <transition name="et-search-fade">
      <div v-if="visible" class="et-search-overlay" @click.self="close">
        <div class="et-search-panel glass-surface" @keydown.stop="onKeydown">
          <div class="et-search-head">
            <el-icon class="et-search-icon"><Search /></el-icon>
            <input
              ref="inputRef"
              v-model="query"
              type="text"
              class="et-search-input"
              placeholder="搜索功能、模块…（模糊匹配）"
              autocomplete="off"
              spellcheck="false"
              @keydown="onInputKey"
            />
            <span class="et-search-hint">{{ hintText }}</span>
          </div>
          <ul class="et-search-list" role="listbox">
            <li
              v-for="(row, i) in results"
              :key="row.id"
              :class="['et-search-item', { active: i === activeIndex }]"
              role="option"
              @click="choose(row)"
              @mouseenter="activeIndex = i"
            >
              <span class="et-search-title">{{ row.title }}</span>
              <span class="et-search-path">{{ row.path }}</span>
            </li>
            <li v-if="results.length === 0" class="et-search-empty">无匹配项，换个关键词试试</li>
          </ul>
        </div>
      </div>
    </transition>
  </teleport>
</template>

<script setup>
import { ref, watch, nextTick, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search } from '@element-plus/icons-vue'
import { searchGlobalItems } from '@/utils/globalSearchIndex'
import {
  getStoredGlobalSearchHotkey,
  eventMatchesGlobalSearchHotkey,
} from '@/utils/globalSearchHotkey'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const router = useRouter()
const query = ref('')
const results = ref([])
const activeIndex = ref(0)
const inputRef = ref(null)
/** 当前绑定的快捷键（从主题本地配置读取），保存后 theme-background-changed 会刷新 */
const boundGlobalSearchAccelerator = ref(getStoredGlobalSearchHotkey())

const refreshGlobalSearchAcceleratorFromStorage = () => {
  boundGlobalSearchAccelerator.value = getStoredGlobalSearchHotkey()
}

const hintText = 'Enter 打开 · Esc 关闭'

function close() {
  visible.value = false
  query.value = ''
}

function refreshResults() {
  results.value = searchGlobalItems(query.value, 60)
  activeIndex.value = 0
}

watch(
  () => props.modelValue,
  (v) => {
    if (v) {
      refreshResults()
      nextTick(() => {
        inputRef.value?.focus?.()
        inputRef.value?.select?.()
      })
    } else {
      query.value = ''
    }
  }
)

watch(query, () => {
  refreshResults()
})

function navigate(row) {
  if (!row?.path) return
  router.push(row.path).then(() => {
    if (row.moduleKey && row.subTab) {
      setTimeout(() => {
        window.dispatchEvent(
          new CustomEvent('easytools-set-tab', {
            detail: { module: row.moduleKey, tab: row.subTab },
          })
        )
      }, 200)
    }
    close()
  })
}

function choose(row) {
  navigate(row)
}

function onInputKey(e) {
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeIndex.value = Math.min(activeIndex.value + 1, Math.max(0, results.value.length - 1))
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeIndex.value = Math.max(activeIndex.value - 1, 0)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const row = results.value[activeIndex.value]
    if (row) navigate(row)
  } else if (e.key === 'Escape') {
    e.preventDefault()
    close()
  }
}

function onKeydown(e) {
  if (e.key === 'Escape') close()
}

function onDocKey(e) {
  if (eventMatchesGlobalSearchHotkey(e, boundGlobalSearchAccelerator.value)) {
    e.preventDefault()
    visible.value = !visible.value
  }
}

onMounted(() => {
  document.addEventListener('keydown', onDocKey)
  window.addEventListener('theme-background-changed', refreshGlobalSearchAcceleratorFromStorage)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onDocKey)
  window.removeEventListener('theme-background-changed', refreshGlobalSearchAcceleratorFromStorage)
})
</script>

<style scoped>
.et-search-overlay {
  position: fixed;
  inset: 0;
  z-index: 10000;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 12vh;
  background: rgba(15, 23, 42, 0.35);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}

.et-search-panel {
  width: min(520px, 92vw);
  max-height: min(70vh, 560px);
  border-radius: var(--et-radius-lg, 14px);
  border: 1px solid var(--et-glass-border, rgba(255, 255, 255, 0.55));
  box-shadow: var(--et-shadow-lg, 0 20px 50px rgba(15, 23, 42, 0.18));
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.glass-surface {
  background: var(--et-glass-bg, rgba(255, 255, 255, 0.82));
}

.et-search-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--et-line, rgba(15, 23, 42, 0.08));
}

.et-search-icon {
  color: var(--et-muted, #64748b);
  flex-shrink: 0;
}

.et-search-input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font-size: 15px;
  color: var(--et-fg, #0f172a);
}

.et-search-hint {
  font-size: 11px;
  color: var(--et-muted, #94a3b8);
  white-space: nowrap;
}

.et-search-list {
  list-style: none;
  margin: 0;
  padding: 6px;
  overflow-y: auto;
  max-height: 52vh;
}

.et-search-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border-radius: var(--et-radius-md, 10px);
  cursor: pointer;
  transition: background 0.12s ease;
}

.et-search-item:hover,
.et-search-item.active {
  background: rgba(59, 130, 246, 0.08);
}

.et-search-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--et-fg, #0f172a);
}

.et-search-path {
  font-size: 11px;
  color: var(--et-muted, #94a3b8);
  font-family: ui-monospace, monospace;
}

.et-search-empty {
  padding: 24px;
  text-align: center;
  color: var(--et-muted, #94a3b8);
  font-size: 13px;
}

.et-search-fade-enter-active,
.et-search-fade-leave-active {
  transition: opacity 0.15s ease;
}

.et-search-fade-enter-from,
.et-search-fade-leave-to {
  opacity: 0;
}
</style>
