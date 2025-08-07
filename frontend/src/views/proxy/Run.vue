<template>
  <a-row :gutter="12">
    <a-col :span="18">
      <a-input
          :readonly="true"
          :model-value="configState.getFilePath()"
          placeholder="请选择包含代理URL的文件"
      >
        <template #suffix>
          <!-- <a-tooltip content="可以从此处获取, 点击即可打开">
          <icon-question-circle @click="BrowserOpenURL('http://proxycompass.com/cn/free-proxies/asia/china/')"/>
          </a-tooltip> -->
        </template>
      </a-input>
    </a-col>
    <a-col :span="6">
      <a-button
          :disabled="disabled || configState.getStatus() === 2"
          v-show="configState.getStatus() !== 2"
          @click="openFile"
          size="medium"
          type='outline' long
      >导入
      </a-button>
      <a-button
          v-show="configState.getStatus() === 2"
          type='outline' long
          size="medium"
          @click="stopTask"
      >停止
      </a-button>
    </a-col>
  </a-row>

  <br>

  <a-row :gutter="12">
    <a-col :span="5">
      <a-card hoverable :bordered="false" class="card progress">
        <a-progress
            :percent="percent"
            :type="'circle'"
            :status='configState.getStatus() === 0 || configState.getStatus() === 3 ? "danger" : (configState.getStatus() === 2 ? "success" : "normal")'
            size="medium"
            style="margin-bottom: 10px;"
        />
        <a-tag class="RunGetStatus" :color='configState.getStatus() === 0 || configState.getStatus() === 3 ? "red" : (configState.getStatus() === 2 ? "green" : "blue")'>
          <span v-if="configState.getStatus() === 0">尚未测试</span>
          <span v-else-if="configState.getStatus() === 1">正在测试</span>
          <span v-else-if="configState.getStatus() === 2">测试完成</span>
          <span v-else>任务取消</span>
        </a-tag>
      </a-card>
    </a-col>

    <a-col :span="19">
      <a-card hoverable :bordered="false" class="card">
        <a-descriptions
            :column="4"
            v-model:data="details"
            title='当前配置'
            :align="{ label: 'right' }"
        />
      </a-card>
    </a-col>
  </a-row>

  <br>

  <div class="log-viewer" ref="logContainer">
    <p class="log-viewer-title">运行日志</p>
    <div v-for="(log, index) in logs" :key="index">
      <span :style="logStyle(log)">{{ log }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue";
import { ChooseFile } from "../../../wailsjs/go/proxy/Proxy";
import { Notification } from '@arco-design/web-vue';
import {BrowserOpenURL, EventsOff, EventsOn} from "../../../wailsjs/runtime";
import {Config, useConfigStore} from "./store/types";

const logs = ref<string[]>([]);  // 日志数组
const logContainer = ref<HTMLElement | null>(null);  // 日志容器

const percent = ref(0.00);  // 进度条百分比
const disabled = ref(false);  // 禁用按钮

const started = ref(false);  // 任务是否已启动

const configState = useConfigStore();  // 获取配置状态
const data = ref<Config>();  // 配置信息
const details = ref([  // 配置详情
  {
    label: '监听绑定',
    value: configState.getSocksAddress(),
  },
  {
    label: '超时时间',
    value: `${configState.getTimeout()} s`,
  },
  {
    label: '协程数',
    value: `${configState.getCoroutineCount()}`,
  },
  // {
  //   label: '代理总数',
  //   value: `${configState.getAllProxies()} 条`,
  // },
  {
    label: '有效代理',
    value: `${configState.getLiveProxies()} 条`,
  },
]);

// 打开文件选择框
function openFile() {
  disabled.value = true;
  ChooseFile().then((res) => {
    data.value = res as unknown as Config;
    if (data.value.Code !== 200) {
      configState.setStatus(3)
      Notification.error({
        title: '任务失败',
        content: data.value.Error,
      });
      return;
    }
  }).catch((err) => {
    configState.setStatus(3)
    Notification.error({
      title: '导入失败',
      content: err,
    });
  }).finally(() => {
    disabled.value = false;
  });
}

// 处理日志输出
const handleLogEmits = (log: string) => {
  logs.value.push(log);
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight;
  }
};

// 日志样式
const logStyle = (log: string) => {
  if (log.includes('[ERR]')) {
    return { color: '#b35351' };
  } else if (log.includes('[INF]')) {
    return { color: '#29b445' };
  } else if (log.includes('[WAR]')) {
    return { color: '#b16f34' };
  }
  return {};
};

const stopTask = () => {
  console.log(configState.getStatus())
  // 检查当前状态，确保任务正在运行
  if (configState.getStatus() !== 2) {
    Notification.error({
      title: '停止失败',
      content: '当前没有正在运行的任务。',
    });
    return; // 任务状态不是运行中，直接返回
  }

  // 向后端发送停止任务请求
  configState.setStatus(0);
  disabled.value = true;  // 禁用按钮防止重复点击

  // 调用 stopListening 停止任务
  configState.stopTask().then(() => {
    configState.setStatus(0);  // 更新任务状态为已取消
    disabled.value = false;  // 启用按钮
    Notification.success({
      title: '任务已停止',
      content: '任务已经成功停止。',
    });
  }).catch((err: any) => {
    Notification.error({
      title: '停止失败',
      content: err,
    });
    disabled.value = false;  // 启用按钮
  });
};


onMounted(() => {
  configState.getProfile()
  EventsOn('log_update', handleLogEmits);
  EventsOn('task_progress', (p: number) => {
    percent.value = p;
  });
  EventsOn('start_task', (profile: Config) => {
    started.value = true;
    configState.setFilePath(profile.FilePath);
    configState.setStatus(1)
    details.value = [
      { label: '监听绑定', value: profile.SocksAddress },
      { label: '协程数',   value: `${profile.CoroutineCount}` },
      { label: '超时时间', value: `${profile.Timeout}s` },
      // 读 profile.AllProxies 而不是 getAllProxies()
      // { label: '代理总数', value: `${profile.AllProxies} 条` },
    ]
  });
  EventsOn('is_ready', (callback: string) => {
    started.value = false;
    configState.setStatus(2)
    details.value.push({
      label: '有效代理',
      value: `${callback} 条`,
    })
    Notification.success({
      title: "任务完成",
      content: `共有 ${callback} 条有效数据`,
      duration: 3000,
      closable: true,
    });
  });
});

onUnmounted(() => {
  EventsOff('log_update');
  EventsOff('task_progress');
  EventsOff('start_task');
  EventsOff('is_ready');
});
</script>

<style scoped>
.log-viewer {
  height: 53vh;
  border-radius: 8px;
  overflow-y: auto;
  background-color: rgba(255, 255, 255, 0.4);
  color: #e88500;
  scrollbar-width: thin;
  border: 1px solid rgba(197, 186, 186, 0.4);
}

.log-viewer-title{
  text-align: left;
  padding-left: 16px;
  color: #000000;
  font-weight: bold;
}

.RunGetStatus{
  border-radius: 5px;
}

.log-viewer::-webkit-scrollbar {
  width: 8px;
}

.log-viewer::-webkit-scrollbar-track {
  background: #1e1e1e;
}

.log-viewer::-webkit-scrollbar-thumb {
  background: #6b7280;
  border-radius: 4px;
}

.log-viewer::-webkit-scrollbar-thumb:hover {
  background: #4b5563;
}

.log-viewer > div {
  margin-left: 10px;
  padding: 2px;
}

.card {
  background-color: rgba(255, 255, 255, 0.4);
  border: 1px solid rgba(197, 186, 186, 0.4);
  border-radius: 8px;
}

.progress {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-left: 15px;
}
</style>
