<template>
  <div ref="reactRootEl" class="react-root" style="width:100%; height:100%"></div>
</template>

<script setup>
defineOptions({ name: 'RestMateView' })

import { ref, onMounted, onBeforeUnmount } from 'vue'
import * as React from 'react'
import { createRoot } from 'react-dom/client'

// React App
import ReactApp from '@/views/restmate/App.jsx'

// React 全局样式
import '@/views/restmate/index.css'
import '@szhsin/react-menu/dist/index.css'
import 'tippy.js/dist/tippy.css'

const reactRootEl = ref(null)
let reactRoot = null

onMounted(() => {
  if (!reactRootEl.value) return
  // 延迟挂载React应用，确保Wails桥接已经准备好
  setTimeout(() => {
    reactRoot = createRoot(reactRootEl.value)
    reactRoot.render(React.createElement(ReactApp, null))
  }, 100)
})

onBeforeUnmount(() => {
  if (reactRoot) {
    reactRoot.unmount()
    reactRoot = null
  }
})
</script>

<style scoped>
:deep(.react-root) {
  width: 100%;
  height: 100%;
}

:deep(.react-root .some-react-class) {
}
</style>