<template>
  <div class="iframe-container">
    <iframe :src="cyberChefUrl"></iframe>
  </div>
</template>

<script>
import { defineComponent } from 'vue';
import {GetPortConfig} from "../../../wailsjs/go/controller/Server";

export default defineComponent({
  name: 'CyberChefView',
  data() {
    return {
      portConfig: null,
    };
  },
  computed: {
    cyberChefUrl() {
      if (!this.portConfig) return 'about:blank';
      return `http://127.0.0.1:${this.portConfig.mainPort}/CyberChef/index.html`;
    },
  },
  async mounted() {
    await this.loadPortConfig();
  },
  methods: {
    async loadPortConfig() {
      try {
        this.portConfig = await GetPortConfig();
        console.log('端口配置:', this.portConfig);
      } catch (error) {
        console.error('获取端口配置失败:', error);
        // 使用默认端口作为后备
        this.portConfig = {
          mainPort: 52867,
        };
      }
    },
  }
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
  height: 90vh;
  border: none;
  border-radius: 8px;
}
</style>