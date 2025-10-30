<template>
  <main class="connection-main">
    <div class="demo-collapse">
      <el-collapse accordion>
        <el-collapse-item v-for="item in list" :name="item.identity" @click="getInfo(item.identity)">
          <template #title>
            <div class="item">
              <div class="truncate-name" style="
                  flex: 1;
                  overflow: hidden;
                  text-overflow: ellipsis;
                  white-space: nowrap;
                  padding-right: 8px;
                ">
                {{ item.name }}
              </div>
              <div style="display: flex">
                <DbInfo @click.stop :data="item"></DbInfo>
                <ConnectionManage @click.stop title="编辑" btn-type="text" :data="item" @emit-connection-list="connectionList"/>
                <el-popconfirm title="确认删除?" @confirm="connectionDelete(item.identity)">
                  <template #reference>
                    <el-button link type="danger" @click.stop>删除</el-button>
                  </template>
                </el-popconfirm>
              </div>
            </div>
          </template>
          <div class="dbs-container">
            <div v-for="db in infoDbList" @click.stop="selectDB(db.key, item.identity)">
              <div v-if="db.key !== selectDbKey" class="my-item">{{db.key}} ( {{db.number}} )</div>
              <div v-else class="my-select-item">{{db.key}} ( {{db.number}} )</div>
            </div>
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </main>
</template>

<script setup>
import {ref, watch} from "vue";
import {ConnectionDelete, ConnectionList, DbList} from "../../../wailsjs/go/redis/Redis";
import {ElNotification} from "element-plus";
import ConnectionManage from "./ConnectionManage.vue";
import DbInfo from "./DbInfo.vue";

let list = ref();
let props = defineProps(["flush"]);
let emits = defineEmits(["emit-select-db"]);
let infoDbList = ref();
let selectDbKey = ref();

watch(props, (newFlush) => {
  connectionList();
});

// 连接列表
function connectionList() {
  ConnectionList().then((res) => {
    if (res.code !== 200) {
      ElNotification({
        title: res.msg,
        type: "error",
      });
    }
    list.value = res.data;
  });
}

connectionList();

// 删除连接
function connectionDelete(identity) {
  ConnectionDelete(identity).then((res) => {
    if (res.code !== 200) {
      ElNotification({
        title: res.msg,
        type: "error",
      });
    }
    ElNotification({
      title: res.msg,
      type: "success",
    });

    // 删除连接后，重新获取连接列表，并重置选中的数据库信息
    connectionList();
    resetSelectedDbInfo();
  });
}

// 获取基本信息
function getInfo(identity) {
  // 获取数据库列表
  infoDbList.value = [];
  DbList(identity).then((res) => {
    if (res.code !== 200) {
      ElNotification({
        title: res.msg,
        type: "error",
      });
    }
    infoDbList.value = res.data;
  });
}

// 选中数据库
function selectDB(db, connIdentity) {
  selectDbKey.value = db;
  // db0 db1
  // 0 1
  emits("emit-select-db", Number(db.substring(2)), connIdentity);
}

// 重置选中的数据库信息
function resetSelectedDbInfo() {
  selectDbKey.value = null;
  infoDbList.value = [];
  emits("emit-select-db", null, null);
}
</script>

<style scoped>
.connection-main {
  height: 100%;
  overflow: hidden;
}

.demo-collapse {
  height: 100%;
  overflow: auto;
}

:deep(.el-collapse) {
  height: 80vh;

}


:deep(.el-collapse-item__content) {
  max-height: 200px; /* 限制内容区域的最大高度 */
  overflow-y: auto; /* 允许垂直滚动 */
  padding-bottom: 0;
}


.item {
  display: flex;
  width: 100%;
  justify-content: space-between;
}

.my-item {
  color: #409eff;
  background-color: #ecf5ff;
  padding: 5px 12px;
  margin-bottom: 5px;
  cursor: pointer;
}

.my-select-item {
  color: #67c23a;
  background-color: #f0f9eb;
  padding: 5px 12px;
  margin-bottom: 5px;
  cursor: pointer;
}
</style>