<template>
  <main v-if="keyDB !== undefined" class="keys-main">
    <el-dialog
        v-model="keyDialogVisible"
        title="创建键"
        width="60%"
        append-to-body
        destroy-on-close
    >
      <el-form :model="keyForm" label-width="120px">
        <el-form-item label="键的名称">
          <el-input placeholder="请输入键的名称" v-model="keyForm.key" />
        </el-form-item>
        <el-form-item label="键的类型">
          <el-select v-model="keyForm.type" placeholder="请选择键的类型">
            <el-option value="string" label="string"></el-option>
            <el-option value="hash" label="hash"></el-option>
            <el-option value="list" label="list"></el-option>
            <el-option value="set" label="set"></el-option>
            <el-option value="zset" label="zset"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="createKey">创建</el-button>
          <el-button @click="keyDialogVisible = false">取消</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>
    
    <!-- 搜索和操作栏 -->
    <div class="keys-header">
      <div class="search-bar">
        <el-input 
          v-model="form.keyword" 
          placeholder="搜索键..." 
          clearable
          @keyup.enter="getKeyList"
          size="small"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button 
          type="primary" 
          @click="getKeyList" 
          size="small"
          :icon="Search"
        >
          查询
        </el-button>
      </div>
      <el-button 
        type="primary" 
        @click="keyDialogVisible = true" 
        size="small"
        class="create-btn"
        :icon="Plus"
      >
        新建键
      </el-button>
    </div>
    
    <!-- 键列表 -->
    <div class="keys-list" id="keys">
      <div 
        v-for="item in keys" 
        :key="item"
        @click="selectKeyKey(item)"
        class="key-item-wrapper"
      >
        <div v-if="item === selectKey" class="key-item key-item-selected">
          <div class="key-item-name">{{item}}</div>
          <el-button text type="danger" @click.stop="deleteKey(item)" size="small" class="delete-btn">
            删除
          </el-button>
        </div>
        <div v-else class="key-item">
          <div class="key-item-name">{{item}}</div>
          <el-button text type="danger" @click.stop="deleteKey(item)" size="small" class="delete-btn">
            删除
          </el-button>
        </div>
      </div>
      <div v-if="!keys || keys.length === 0" class="empty-keys">
        <el-empty description="暂无键" :image-size="60" />
      </div>
    </div>
  </main>
</template>

<script setup>
import {ConnectionDelete, CreateKeyValue, DeleteKeyValue, KeyList} from "../../../wailsjs/go/nosql/NoSQL";
import {reactive, ref, watch} from 'vue'
import {ElMessage} from "element-plus";
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';
import { Search, Plus } from "@element-plus/icons-vue";

const form = reactive({
  keyword: '',
})

let props = defineProps(['keyDB', 'keyConnIdentity'])
let emits = defineEmits(['emit-select-key'])
let keys = ref()
let selectKey = ref()
let keyDialogVisible = ref(false)
let keyForm = ref({})

watch(props, ()=> {
  getKeyList()
})

function getKeyList() {
  let data = {
    conn_identity: props.keyConnIdentity,
    db: props.keyDB,
    keyword: form.keyword.trim()
  };
  KeyList(data).then(res => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return;
    }
    // 重置选中的 key，保证后续点击一定会触发更新
    emits("emit-select-key", null);
    // 更新列表
    keys.value = res.data;
  });
}


function selectKeyKey(key) {
  selectKey.value = key
  emits('emit-select-key', key)
}

async function deleteKey(key) {
  try {
    await confirmDangerousAction({
      title: '删除键',
      message: `将删除键「${key}」及其全部数据，不可恢复。`,
      expectedInput: DANGER_DELETE_TOKEN
    })
  } catch {
    return
  }
  DeleteKeyValue({conn_identity: props.keyConnIdentity, db: props.keyDB, key: key}).then(res => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success(`${res.msg}`)
    getKeyList()
  })
}

function createKey() {
  CreateKeyValue({conn_identity: props.keyConnIdentity, db: props.keyDB, key: keyForm.value.key, type: keyForm.value.type}).then(res => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success(`${res.msg}`)
    keyDialogVisible.value = false
    getKeyList()
  })
}
</script>

<style scoped>
.keys-main {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 12px;
  overflow: hidden;
  box-sizing: border-box;
}

.keys-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
  flex-shrink: 0;
  /* 参考键值详情头部的浅色卡片背景 */
  padding: 10px 12px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.15);
  border: 1px solid rgba(0, 0, 0, 0.1);
}

.search-bar {
  display: flex;
  gap: 8px;
}

.search-bar :deep(.el-input) {
  flex: 1;
}

.create-btn {
  width: 100%;
}

.keys-list {
  flex: 1;
  overflow: auto;
  min-height: 0;
  margin-top: 4px;
  padding: 8px 10px;
  border-radius: 6px;
  /* 参考 KeyValue 中 table-container 的背景设计 */
  border: 1px solid rgba(0, 0, 0, 0.15);
  background: rgba(255, 255, 255, 0.45);
}

/* 优化滚动条 */
.keys-list::-webkit-scrollbar {
  width: 6px;
}

.keys-list::-webkit-scrollbar-track {
  background: transparent;
}

.keys-list::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.keys-list::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

.key-item-wrapper {
  margin-bottom: 4px;
  cursor: pointer;
}

.key-item {
  display: flex;
  width: 100%;
  justify-content: space-between;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-radius: 5px;
  transition: all 0.2s ease;
  border: 1px solid rgba(144, 147, 153, 0.3);
  background: rgba(144, 147, 153, 0.1);
  color: rgba(0, 0, 0, 0.75);
  font-weight: 600;
}

.key-item:hover {
  background: rgba(144, 147, 153, 0.2);
  border-color: rgba(144, 147, 153, 0.5);
  transform: translateX(2px);
  color: rgba(0, 0, 0, 0.85);
}

.key-item-selected {
  color: #E6A23C;
  background: rgba(230, 162, 60, 0.25);
  border-color: rgba(230, 162, 60, 0.6);
  font-weight: 700;
  box-shadow: 0 0 8px rgba(230, 162, 60, 0.4);
}

.key-item-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  padding-right: 8px;
}

.delete-btn {
  opacity: 0.6;
  transition: opacity 0.2s;
  flex-shrink: 0;
  padding: 4px 6px;
  font-size: 11px;
}

.key-item:hover .delete-btn {
  opacity: 1;
}

.empty-keys {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: rgba(0, 0, 0, 0.5);
}

/* 确保输入框文字清晰，按钮保持默认颜色 */
:deep(.el-input__inner) {
  color: rgba(0, 0, 0, 0.85);
  background: rgba(255, 255, 255, 0.6);
  border-color: rgba(0, 0, 0, 0.15);
}

:deep(.el-input__inner::placeholder) {
  color: rgba(0, 0, 0, 0.4);
}

:deep(.el-input__inner:focus) {
  border-color: #409EFF;
  background: rgba(255, 255, 255, 0.8);
}

/* 按钮保持Element Plus默认颜色，不覆盖 */
</style>