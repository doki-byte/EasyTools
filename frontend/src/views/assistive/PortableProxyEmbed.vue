<template>
  <div class="portable-proxy-embed">
    <Panel />
  </div>
</template>

<script setup>
defineOptions({
  name: 'PortableProxyAssistive'
})
import Panel from '@/views/proxy/Panel.vue'
</script>

<style scoped>
.portable-proxy-embed {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 0 4px 8px;
}
/* 让代理面板占满嵌入区域高度（flex 撑开，内部各页签自行滚动） */
.portable-proxy-embed :deep(.proxy-panel-layout) {
  flex: 1 1 auto;
  height: auto;
  min-height: 0;
  max-height: none;
}
</style>
