<template>
  <main class="connection-main">
    <el-scrollbar>
      <div class="config-list">
        <el-collapse accordion>
          <template v-for="item in list" :key="item?.identity || `item-${item?.name || 'unknown'}`">
            <el-collapse-item v-if="item && item.identity" :name="item.identity" @click="getInfo(item.identity)">
              <template #title>
                <div class="config-item">
                  <div class="config-info">
                    <div class="config-name">{{ item.name }}</div>
                    <div class="config-meta">
                      <el-tag size="small" :type="getTypeTagType(item.db_type)">{{ getTypeDisplayName(item.db_type) }}</el-tag>
                      <span class="config-host">{{ getItemHostDisplay(item) }}</span>
                    </div>
                  </div>
                  <div class="config-actions" @click.stop>
                    <el-button
                      link
                      type="primary"
                      size="small"
                      @click.stop="editConnection(item)"
                      title="编辑"
                    >
                      <el-icon><Edit /></el-icon>
                    </el-button>
                    <el-button
                      link
                      type="danger"
                      size="small"
                      title="删除"
                      @click.stop="connectionDelete(item.identity)"
                    >
                      <el-icon><Delete /></el-icon>
                    </el-button>
                  </div>
                </div>
              </template>
              <div class="dbs-container">
                <div v-for="db in getDbListForItem(item.identity)" @click.stop="selectDB(db.key, item.identity)">
                  <div v-if="db.key !== selectDbKey || currentConnIdentity !== item.identity" class="my-item">{{db.key}} ( {{db.number}} )</div>
                  <div v-else class="my-select-item">{{db.key}} ( {{db.number}} )</div>
                </div>
              </div>
            </el-collapse-item>
          </template>
        </el-collapse>
        <div v-if="list.length === 0" class="empty-config">
          <el-empty description="暂无连接配置" :image-size="80" />
        </div>
      </div>
    </el-scrollbar>
  </main>
</template>

<script setup>
import {ref, watch, reactive} from "vue";
import {ConnectionDelete, ConnectionList, DbList} from "../../../wailsjs/go/nosql/NoSQL";
import {ElMessage} from "element-plus";
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';
import {Edit, Delete} from "@element-plus/icons-vue";
import ConnectionManage from "./ConnectionManage.vue";

let list = ref([]);
let props = defineProps(["flush"]);
let emits = defineEmits(["emit-select-db"]);
let selectDbKey = ref();
let currentConnIdentity = ref();
let infoDbList = ref();
// 存储每个连接展开的数据库列表（使用对象模拟Map）
let dbListMap = reactive({});
let connectionManageRef = ref(null);

watch(() => props.flush, (newVal, oldVal) => {
  // 当 flush 属性改变时，重新加载连接列表
  if (newVal !== oldVal) {
    connectionList();
  }
}, { immediate: false });

// 连接列表
function connectionList() {
  ConnectionList().then((res) => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    // 确保数据是数组且每个项都有必要的属性
    const data = (res.data || []).filter(item => item && item.identity);
    // 使用 nextTick 确保数据更新后再更新列表
    list.value = data || [];
  }).catch(err => {
    console.error('加载连接列表失败:', err);
    ElMessage.error('加载连接列表失败');
    list.value = [];
  });
}

connectionList();

// 编辑连接
function editConnection(item) {
  emits("emit-edit-connection", item);
}

// 删除连接
async function connectionDelete(identity) {
  try {
    await confirmDangerousAction({
      title: '删除连接',
      message: '将删除该 NoSQL 连接配置，不可恢复。',
      expectedInput: DANGER_DELETE_TOKEN
    })
  } catch {
    return
  }
  ConnectionDelete(identity).then((res) => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success(`${res.msg}`)

    // 清理该连接的数据
    if (dbListMap[identity]) {
      delete dbListMap[identity];
    }
    
    // 如果删除的是当前选中的连接，重置选中状态
    if (currentConnIdentity.value === identity) {
      resetSelectedDbInfo();
    }
    
    // 重新获取连接列表
    connectionList();
  });
}

// 获取基本信息
function getInfo(identity) {
  // 获取数据库列表
  if (!dbListMap[identity]) {
    DbList(identity).then((res) => {
      if (res.code !== 200) {
        ElMessage.error(`${res.msg}`)
        return
      }
      dbListMap[identity] = res.data || [];
    });
  }
}

// 获取指定连接的数据库列表
function getDbListForItem(identity) {
  return dbListMap[identity] || [];
}

// 选中数据库
function selectDB(db, connIdentity) {
  selectDbKey.value = db;
  currentConnIdentity.value = connIdentity;
  
  // 从连接列表中查找对应的连接，判断数据库类型
  const connection = list.value.find(conn => (conn.identity === connIdentity || conn.Identity === connIdentity));
  const dbType = connection?.db_type || connection?.DbType || 'redis';
  const dbTypeLower = dbType.toLowerCase();
  
  // Redis 数据库名格式: db0, db1, ... 需要提取数字
  // MongoDB 数据库名是字符串: admin, config, test, ...
  if (dbTypeLower === 'mongodb' || dbTypeLower === 'mongo') {
    // MongoDB: 直接使用数据库名字符串
    emits("emit-select-db", db, connIdentity);
  } else {
    // Redis: 从 db0, db1 格式中提取数字
    // db0 db1 -> 0 1
    const dbNumber = db.startsWith('db') ? Number(db.substring(2)) : Number(db);
    if (isNaN(dbNumber)) {
      console.error('无效的数据库名:', db, '数据库类型:', dbType);
      ElMessage.error('无效的数据库名');
      return;
    }
    emits("emit-select-db", dbNumber, connIdentity);
  }
}

// 重置选中的数据库信息
function resetSelectedDbInfo() {
  selectDbKey.value = null;
  currentConnIdentity.value = null;
  emits("emit-select-db", null, null);
}

// 获取类型标签类型（参考数据库管理器）
function getTypeTagType(dbType) {
  if (!dbType) return 'info';
  const typeMap = {
    redis: 'danger',
    mongodb: 'success',
    mongo: 'success'
  };
  return typeMap[dbType.toLowerCase()] || 'info';
}

// 获取类型显示名称
function getTypeDisplayName(dbType) {
  if (!dbType) return 'Unknown';
  const typeMap = {
    redis: 'Redis',
    mongodb: 'MongoDB',
    mongo: 'MongoDB'
  };
  return typeMap[dbType.toLowerCase()] || dbType;
}

// 获取主机显示信息
function getItemHostDisplay(item) {
  if (!item) return '';
  const addr = item.addr || '';
  const port = item.port || '';
  if (addr && port) {
    return `${addr}:${port}`;
  }
  return addr || port || '';
}

</script>

<style scoped>
.connection-main {
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

:deep(.el-scrollbar) {
  flex: 1;
  height: 100%;
}

:deep(.el-scrollbar__wrap) {
  overflow-x: hidden !important;
}

.config-list {
  padding: 8px;
}

.config-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  width: 100%;
  box-sizing: border-box;
}

.config-info {
  flex: 1;
  min-width: 0;
}

.config-name {
  font-weight: 600;
  font-size: 14px;
  color: #333;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.config-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  flex: 1;
  min-width: 0;
}

.config-host {
  color: rgba(0, 0, 0, 0.6);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}

.config-actions {
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s;
  flex-shrink: 0;
}

.config-item:hover .config-actions {
  opacity: 1;
}

.empty-config {
  padding: 40px 20px;
  text-align: center;
}

/* 折叠面板样式 */
:deep(.el-collapse) {
  border: none;
  background: transparent;
}

:deep(.el-collapse-item) {
  margin-bottom: 8px;
  border-radius: 6px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  background: rgba(255, 255, 255, 0.8);
  overflow: hidden;
  transition: all 0.2s ease;
}

:deep(.el-collapse-item:hover) {
  background: rgba(64, 158, 255, 0.1);
  border-color: rgba(64, 158, 255, 0.3);
}

:deep(.el-collapse-item__header:hover .config-item) {
  background: transparent;
}

:deep(.el-collapse-item__header) {
  padding: 0;
  background: transparent;
  border: none;
  height: auto;
  line-height: 1.5;
}

:deep(.el-collapse-item__content) {
  padding: 8px 12px;
  background: rgba(255, 255, 255, 1);
  border-top: 1px solid rgba(0, 0, 0, 0.1);
}

.dbs-container {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.my-item {
  color: #E6A23C;
  background: rgba(230, 162, 60, 0.15);
  padding: 6px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  transition: all 0.2s ease;
  border: 1px solid rgba(230, 162, 60, 0.3);
}

.my-item:hover {
  background: rgba(230, 162, 60, 0.25);
  border-color: rgba(230, 162, 60, 0.5);
  transform: translateX(2px);
  color: #E6A23C;
}

.my-select-item {
  color: #E6A23C;
  background: rgba(230, 162, 60, 0.3);
  padding: 6px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
  border: 1px solid rgba(230, 162, 60, 0.6);
  box-shadow: 0 0 8px rgba(230, 162, 60, 0.4);
}
</style>