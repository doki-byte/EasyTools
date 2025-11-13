<template>
  <div class="iframe-container">
    <iframe :src="iframeSrc"></iframe>
  </div>
</template>

<script>
import { defineComponent } from 'vue';

export default defineComponent({
  name: 'CyberChefView',
  data() {
    return {
      // 本地 iframe 路径
      iframeSrc: 'http://127.0.0.1:52867/CyberChef/index.html',
    };
  },
});
</script>

<style scoped>
/* 保持原有样式不变 */
.iframe-container {
  max-width: 100%;
  overflow: hidden;
  border-radius: 8px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.iframe-container iframe {
  width: 100%;
  height: 98vh;
  border: none;
  border-radius: 8px;
}
</style>