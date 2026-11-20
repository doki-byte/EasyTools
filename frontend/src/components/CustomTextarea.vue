<template>
  <div class="textarea-container">
    <el-input
      v-model="localValue"
      type="textarea"
      :rows="rows"
      :placeholder="placeholder"
      :readonly="readonly"
      :resize="resize"
      :minRows="2"
      @input="handleInput"
    ></el-input>

    <el-button-group v-if="showActions && !readonly" class="action-area">
      <el-button :icon="CloseBold" size="small" @click="clearInput"></el-button>
    </el-button-group>
    <el-button v-else :icon="DocumentCopy" size="small" @click="handleCopy" class="action-area">
      Copy
    </el-button>
  </div>
</template>

<script>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { CloseBold, DocumentCopy } from '@element-plus/icons-vue'

export default {
  name: 'CustomTextarea',
  props: {
    modelValue: {
      type: String,
      required: true
    },
    rows: {
      type: Number,
      default: 6
    },
    showActions: {
      type: Boolean,
      default: true
    },
    readonly: {
      type: Boolean,
      default: false
    },
    resize: {
      type: String,
      default: 'vertical'
    },
    placeholder: {
      type: String,
      default: ''
    }
  },
  emits: ['update:modelValue', 'continue'],
  setup(props, { emit }) {
    const localValue = ref(props.modelValue)

    watch(
      () => props.modelValue,
      (newValue) => {
        localValue.value = newValue
      }
    )

    const handleInput = (value) => {
      emit('update:modelValue', value)
    }

    const clearInput = () => {
      localValue.value = ''
      emit('update:modelValue', '')
    }

    const handleCopy = () => {
      if (navigator.clipboard) {
        navigator.clipboard.writeText(props.modelValue).then(() => {
          ElMessage.success('已复制到剪贴板')
        })
      } else {
        ElMessage.warning('复制功能不可用')
      }
    }

    return {
      localValue,
      handleInput,
      clearInput,
      handleCopy,
      CloseBold,
      DocumentCopy
    }
  }
}
</script>

<style scoped>
.textarea-container {
  position: relative;
}

.action-area {
  position: absolute;
  bottom: 8px;
  right: 8px;
}
</style>

