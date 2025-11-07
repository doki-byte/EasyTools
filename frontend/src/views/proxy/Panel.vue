<template>
  <a-layout style="height: 95.6vh;">
    <a-layout-header class="header">
      <a-alert center :type="alertType">
        <p>{{ statusMessage }}</p>
      </a-alert>
    </a-layout-header>

    <a-layout-content class="content">
      <a-tabs v-model:active-key="activeTab">
        <a-tab-pane key="1">
          <template #title>
            <icon-poweroff/> 运行
          </template>
          <Run/>
        </a-tab-pane>
        <a-tab-pane key="2">
          <template #title>
            <icon-clock-circle/> 获取
          </template>
          <Fetch @switch-tab="handleSwitchTab"/>
        </a-tab-pane>
        <a-tab-pane key="3">
          <template #title>
            <icon-settings/> 配置
          </template>
          <Config/>
        </a-tab-pane>
      </a-tabs>
    </a-layout-content>
  </a-layout>
</template>


<script setup lang="ts">
defineOptions({
  name: 'ProxyView'
})

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
.header {
  margin: 4px;
  background-color: rgb(255, 255, 255);
}
.content {
  margin: 4px;
}
</style>