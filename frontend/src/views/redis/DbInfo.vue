<template>
  <main>
    <el-button type="primary" @click="getDbInfo" link>详情</el-button>
    <el-dialog
        v-model="dialogVisible"
        :title="`数据库详情(${ip})`"
        width="60%"
    >
      <el-table :data="info" border stripe style="width: 100%">
        <el-table-column type="index" width="65"/>
        <el-table-column prop="key" label="Key" />
        <el-table-column prop="value" label="Value" />
      </el-table>
    </el-dialog>
  </main>
</template>

<script setup>
import { ref, computed } from 'vue';
import { DbInfo } from '../../../wailsjs/go/redis/Redis';
import { ElNotification } from 'element-plus';

let dialogVisible = ref(false);
let props = defineProps(['data']);
let info = ref([]);

// 从 props 中获取 IP（identity）
const ip = computed(() => props.data.addr || '');

function getDbInfo() {
  dialogVisible.value = true;
  DbInfo(props.data.identity).then(res => {
    if (res.code !== 200) {
      ElNotification({
        title: res.msg,
        type: 'error',
      });
      return;
    }
    info.value = res.data;
  });
}
</script>

<style scoped>
</style>
