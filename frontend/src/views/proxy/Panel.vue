<template>
  <el-container class="proxy-panel-layout">
    <el-header class="header">
      <el-alert :type="alertType" :closable="false" show-icon>
        <template #title>{{ statusMessage }}</template>
      </el-alert>
    </el-header>

    <el-main class="content">
      <el-tabs v-model="activeTab">
        <el-tab-pane name="1">
          <template #label>
            <span class="proxy-tab-label"><el-icon><SwitchButton /></el-icon> 运行</span>
          </template>
          <Run/>
        </el-tab-pane>
        <el-tab-pane name="2">
          <template #label>
            <span class="proxy-tab-label"><el-icon><Download /></el-icon> 获取</span>
          </template>
          <Fetch @switch-tab="handleSwitchTab"/>
        </el-tab-pane>
        <el-tab-pane name="3">
          <template #label>
            <span class="proxy-tab-label"><el-icon><Setting /></el-icon> 配置</span>
          </template>
          <Config/>
        </el-tab-pane>
      </el-tabs>
    </el-main>
  </el-container>
</template>


<script setup lang="ts">
defineOptions({
  name: 'ProxyView'
})

import { SwitchButton, Download, Setting } from '@element-plus/icons-vue'
import { ref, computed, onMounted, onUnmounted } from "vue";
import Run from "./Run.vue";
import Fetch from "./Fetch.vue";
import Config from "./Config.vue";
import {useConfigStore} from "./store/types";
import { EventsOn, EventsOff } from "../../../wailsjs/runtime";

const configStore = useConfigStore()

// 添加当前激活标签页的响应式变量
const activeTab = ref('1')

// 监听当前IP更新
const setupIPListener = () => {
  EventsOn('current_ip', (ip: string) => {
    console.log('收到当前IP更新:', ip);
    configStore.setCurrentIP(ip);
  });
};

// 处理标签页切换
const handleSwitchTab = (tabKey: string) => {
  console.log('切换到标签页:', tabKey)
  activeTab.value = tabKey
}

// 使用计算属性确保响应式
const current = computed(() => configStore.getCurrentIP());
const status = computed(() => configStore.getStatus());
const alertType = computed(() => {
  switch (configStore.getStatus()) {
    case 1: return "info"
    case 2: return "success"
    default: return "warning"
  }
});

const statusMessage = computed(() => {
  switch (configStore.getStatus()) {
    case 0: return "尚未指定数据源，请导入txt文件或在线获取。"
    case 1: return "已指定，等待可用性测试完成..."
    case 2: return `数据源已准备就绪, 当前IP: ${current.value}。`
    default: return "执行异常，请检查配置。"
  }
});

onMounted(() => {
  setupIPListener();
  // 初始化时获取当前状态
  configStore.getProfile();
});

onUnmounted(() => {
  EventsOff('current_ip');
});
</script>


<style scoped>
.proxy-panel-layout {
  /* 由外部容器决定高度：嵌在辅助工具/独立页时均铺满可用区域，
     不再使用 95.6vh 之类的视口硬编码，避免被模块页签/标题栏挤压后裁切 */
  height: 100%;
  min-height: 0;
  position: relative;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.header {
  flex: none;
  margin: 0;
  background-color: rgb(255, 255, 255);
  height: auto !important;
  padding: 0 8px;
}

.content {
  flex: 1 1 auto;
  min-height: 0;
  margin: 4px;
  background-color: #faf9f6;
  border-radius: 8px;
  padding: 10px 12px 12px;
  overflow-y: auto; /* 页面过长时由内容区整体滚动，保证底部内容可达 */
  overflow-x: hidden;
}

.proxy-tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

/* 取消 Element Plus 对内容区的内部隐藏裁切：让高度跟随内容，由外层滚动接管 */
.content :deep(.el-tabs__content) {
  overflow: visible;
  background-color: #faf9f6;
}

.content :deep(.el-tabs__header) {
  margin: 0 0 10px;
}
</style>
