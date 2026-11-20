<template>
  <el-input
    v-model="localText"
    type="textarea"
    :class="inputClass"
    v-bind="inputAttrs"
    @blur="commit"
  />
</template>

<script>
/**
 * 输入与父级响应式树隔离：打字时只重渲染本组件，避免巨型页面每键全量 diff。
 * 在 blur / 卸载 / flush() 时再把内容同步给父组件。
 */
export default {
  name: 'IsolatedTextarea',
  inheritAttrs: false,
  props: {
    modelValue: {
      type: String,
      default: '',
    },
  },
  emits: ['update:modelValue'],
  data() {
    return {
      localText: this.modelValue || '',
    };
  },
  computed: {
    inputClass() {
      return ['isolated-textarea', this.$attrs.class].filter(Boolean);
    },
    inputAttrs() {
      const { class: _cls, ...rest } = this.$attrs;
      return rest;
    },
  },
  watch: {
    modelValue(val) {
      const next = val || '';
      if (next !== this.localText) {
        this.localText = next;
      }
    },
  },
  beforeUnmount() {
    this.commit();
  },
  methods: {
    commit() {
      const next = this.localText ?? '';
      if (next !== (this.modelValue ?? '')) {
        this.$emit('update:modelValue', next);
      }
    },
    flush() {
      this.commit();
    },
    getValue() {
      return this.localText ?? '';
    },
  },
};
</script>

<style scoped>
.isolated-textarea {
  width: 100%;
}
</style>
