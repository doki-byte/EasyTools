<script setup lang="ts">
defineOptions({
  name: 'ProxyView'
})

import { ref} from "vue";
import Run from "./Run.vue";
import Fetch from "./Fetch.vue";
import Config from "./Config.vue";
import {useConfigStore} from "./store/types";

const current = ref("N/A")
const configStore = useConfigStore()
</script>


<template>
  <a-layout style="height: 95.6vh;">
    <a-layout-header class="header">
      <a-alert center :type='configStore.getStatus() == 1 ? "info" : (configStore.getStatus() == 2 ? "success" : "warning")'>
        <p v-if="configStore.getStatus() == 0">尚未指定数据源，请导入txt文件或在线获取。</p>
        <p v-else-if="configStore.getStatus() == 1">已指定，等待可用性测试完成...</p>
        <p v-else-if="configStore.getStatus() == 2">数据源已准备就绪, 当前IP: {{current}}。</p>
        <p v-else>执行异常，请检查配置。</p>
      </a-alert>
    </a-layout-header>
    
    <a-layout-content class="content">
      <a-tabs>
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
            <Fetch/>
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

<style scoped>

.header {
  margin: 4px;
  background-color: rgb(255, 255, 255);
}
.content {
  margin: 4px;
}
</style>