<script setup lang="ts">
import {ref} from "vue";
import {FetchProxies, UseFetchedDatasets} from "../../../wailsjs/go/proxy/Proxy";
import {Notification} from '@arco-design/web-vue';
import {useConfigStore} from "./store/types";

interface ProxyInfo {
  key: string;
  source: string;
  kind: string;
  address: string;
}

const datasets = ref<ProxyInfo[]>([
  {
    key: '',
    source: '',
    kind: '',
    address: '',
  },
])

const configStore = useConfigStore()
const loading = ref(false)
const pagination = {
  pageSize: 10,
  showPageSize: true,
  showJumper: true,
  showTotal: true
}
const columns = [
  {
    title: '序号',
    dataIndex: 'key',
  },
  {
    title: '类型',
    dataIndex: 'kind',
  },
  {
    title: '来源',
    dataIndex: 'source',
    //   filterable: {
    //     filters: [{
    //       text: '89代理',
    //       value: '89代理',
    //     }, {
    //       text: '开心代理',
    //       value: '开心代理',
    //     }, {
    //       text: '齐云代理',
    //       value: '齐云代理',
    //     }],
    //     filter: (value: string, row: any) => row.address.includes(value),
    //   }
  },
  {
    title: 'IP',
    dataIndex: 'address',
  },
]

function useFetchedDatasets() {
  Notification.info({
    title: '任务开始',
    content: '请手动转至运行标签页查看',
    duration: 1500,
    closable: true,
  });
  UseFetchedDatasets().then(res => {
    if (res.Code !== 200) {
      Notification.error({
        title: '错误',
        content: res.Message,
        duration: 2000,
        closable: true,
      });

      configStore.setStatus(3)
      return;
    }

    Notification.success({
      title: '任务完成',
      content: res.Message,
      duration: 2000,
    });
  })
}

function getProxies() {
  loading.value = true
  FetchProxies().then(res => {
    if (res.Code !== 200) {
      Notification.error({
        title: '错误',
        content: res.Message,
        duration: 0,
        closable: true,
      });

      loading.value = false
      return;
    }

    Notification.success({
      title: '任务完成',
      content: res.Message,
      duration: 2000,
    });

    datasets.value = JSON.parse(res.Data) as ProxyInfo[]
    configStore.setStatus(1)
    loading.value = false
  })
}
</script>

<template>
  <a-row :gutter="12">
    <a-col :span="18">
      <a-alert type='info'>如欲使用Hunter等进行搜索，请先配置好Key哦</a-alert>

<!--      <a-alert type='info'>公开免费的代理稳定性较差，请谨慎使用。</a-alert>-->
    </a-col>
    <a-col :span="6">
      <a-button-group>
        <a-button type="outline" size="large" :disabled="loading" @click="getProxies">获取</a-button>
        <a-button type="outline" status="success" size="large" :disabled="loading" @click="useFetchedDatasets">使用</a-button>
      </a-button-group>
    </a-col>
  </a-row>
  <br/>
  <a-table
      :columns="columns"
      :loading="loading"
      :data="datasets"
      :pagination="pagination"
      :scroll="{ y: '400px' }"
  />
</template>

<style scoped>

</style>