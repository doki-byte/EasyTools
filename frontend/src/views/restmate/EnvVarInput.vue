<template>
  <div
    class="env-var-input"
    contenteditable
    @input="handleInput"
    @blur="handleBlur"
    @focus="handleFocus"
    @keydown="handleKeydown"
    @wheel="handleWheel"
    :class="{ 'is-focused': focused }"
    :data-placeholder="placeholder"
    :style="inputStyle"
    ref="inputRef"
  ></div>
</template>

<script>
export default {
  name: 'EnvVarInput',
  props: {
    modelValue: {
      type: String,
      default: '',
    },
    envVars: {
      type: Object,
      default: () => ({}),
    },
    placeholder: {
      type: String,
      default: '',
    },
    size: {
      type: String,
      default: 'default',
    },
    rows: {
      type: Number,
      default: null,
    },
  },
  emits: ['update:modelValue', 'keydown'],
  data() {
    return {
      focused: false,
    };
  },
  computed: {
    inputStyle() {
      if (this.rows) {
        const lineHeight = 1.5;
        const padding = 10; // 上下各 5px
        const minHeight = this.rows * lineHeight * 14 + padding; // 14px 是字体大小
        return {
          minHeight: `${minHeight}px`,
          height: 'auto',
        };
      }
      return {};
    },
  },
  watch: {
    modelValue: {
      handler(newVal, oldVal) {
        const newValStr = String(newVal || '');
        const oldValStr = String(oldVal || '');
        if (newValStr !== oldValStr) {
          this.queueContentUpdate();
        }
      },
      immediate: true,
    },
    envVars: {
      handler() {
        this.queueContentUpdate();
      },
      immediate: true,
      deep: true,
    },
  },
  mounted() {
    this.queueContentUpdate();
  },
  updated() {
    if (!this.focused) {
      this.queueContentUpdate();
    }
  },
  methods: {
    getInputElement() {
      if (this._inputElement && this._inputElement.isConnected) {
        return this._inputElement;
      }
      const el = this.$refs.inputRef || this.$el || null;
      if (el) {
        this._inputElement = el;
      }
      return el;
    },
    queueContentUpdate() {
      this.$nextTick(() => {
        const schedule =
          (typeof window !== 'undefined' && window.requestAnimationFrame) ||
          (fn => setTimeout(fn, 16));
        schedule(() => {
          const el = this.getInputElement();
          if (!el) return;
          this.updateContent();
        });
      });
    },
    handleInput(e) {
      const text = e.target.textContent || '';
      this.$emit('update:modelValue', text);
    },
    handleKeydown(e) {
      // 传递所有键盘事件给父组件
      this.$emit('keydown', e);
      // 如果是回车键且不是多行输入，阻止默认行为
      if (e.key === 'Enter' && !e.shiftKey) {
        const el = this.getInputElement();
        if (el) {
          const style = window.getComputedStyle(el);
          if (style.whiteSpace === 'nowrap' || style.overflow === 'hidden') {
            e.preventDefault();
          }
        }
      }
    },
    handleWheel(e) {
      const el = this.getInputElement();
      if (!el) return;
      const delta = Math.abs(e.deltaX) > Math.abs(e.deltaY) ? e.deltaX : e.deltaY;
      if (!delta) return;
      const prev = el.scrollLeft;
      el.scrollLeft += delta;
      if (el.scrollLeft !== prev) {
        e.stopPropagation();
      }
    },
    handleFocus() {
      this.focused = true;
      // 获得焦点时显示纯文本
      const el = this.getInputElement();
      if (el) {
        el.textContent = this.modelValue || '';
      }
    },
    handleBlur() {
      this.focused = false;
      this.updateContent();
    },
    updateContent() {
      const el = this.getInputElement();
      if (!el) {
        return;
      }
      
      const text = String(this.modelValue || '');
      const currentText = el.textContent || '';
      
      // 如果内容相同且不在焦点状态，不需要更新
      if (currentText === text && !this.focused) {
        return;
      }
      
      if (!text) {
        // 如果值为空，清空内容
        el.innerHTML = '';
        el.textContent = '';
        return;
      }
      
      // 如果不在焦点状态，更新高亮内容
      if (!this.focused) {
        const highlighted = this.highlightEnvVars(this.escapeHtml(text));
        el.innerHTML = highlighted;
      } else {
        // 如果正在编辑，只更新文本内容（如果不同）
        if (currentText !== text) {
          el.textContent = text;
        }
      }
    },
    escapeHtml(str) {
      return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
    },
    highlightEnvVars(text) {
      const envVarKeys = Object.keys(this.envVars || {});
      return text.replace(
        /\{\{([^}]+)\}\}/g,
        (match, varName) => {
          const varKey = varName.trim();
          const exists = envVarKeys.includes(varKey);
          const className = exists ? 'env-var-exists' : 'env-var-missing';
          return `<span class="${className}">${match}</span>`;
        },
      );
    },
  },
};
</script>

<style scoped>
.env-var-input {
  width: 100%;
  min-height: 32px;
  padding: 5px 11px;
  line-height: 1.5;
  font-size: 14px;
  color: var(--el-input-text-color, #606266);
  background-color: var(--el-input-bg-color, #fff);
  border: 1px solid var(--el-input-border-color, #dcdfe6);
  border-radius: 4px;
  outline: none;
  box-sizing: border-box;
  white-space: nowrap;
  word-break: normal;
  overflow-x: auto;
  overflow-y: hidden;
  font-family: Menlo, Monaco, Consolas, 'Courier New', monospace;
  user-select: text;
  scrollbar-width: thin;
  -ms-overflow-style: auto;
}

.env-var-input::-webkit-scrollbar {
  height: 6px;
}

.env-var-input::-webkit-scrollbar-thumb {
  background-color: rgba(0, 0, 0, 0.25);
  border-radius: 3px;
}

.env-var-input:empty::before {
  content: attr(data-placeholder);
  color: var(--el-input-placeholder-color, #c0c4cc);
}

.env-var-input.is-focused {
  border-color: var(--el-color-primary, #409eff);
  outline: none;
}

.env-var-input:focus {
  border-color: var(--el-color-primary, #409eff);
}
</style>

<style>
/* 环境变量高亮样式（全局） */
.env-var-exists {
  color: #67c23a !important;
  background-color: rgba(103, 194, 58, 0.15);
  padding: 1px 3px;
  border-radius: 2px;
  font-weight: 600;
}

.env-var-missing {
  color: #f56c6c !important;
  background-color: rgba(245, 108, 108, 0.15);
  padding: 1px 3px;
  border-radius: 2px;
  font-weight: 600;
}
</style>
