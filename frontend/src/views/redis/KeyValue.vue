<template>
  <main v-if="keyKey !== undefined" class="keyvalue-main">
    <el-form :model="form" class="keyvalue-form">
      <el-form-item label="键">
        <el-input type="textarea" autosize v-model="form.key" disabled placeholder="" />
      </el-form-item>
      <el-row :gutter="16" class="form-row">
        <el-col :span="12">
          <el-form-item label="过期时间">
            <el-input v-model="form.ttl" placeholder="" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="数据类型">
            <el-input v-model="form.type" disabled placeholder="" />
          </el-form-item>
        </el-col>
      </el-row>
      <div v-if="form.type === 'string'" class="value-section">
        <el-form-item label="值">
          <el-input type="textarea" :autosize="{ minRows: 6, maxRows: 20 }" v-model="form.value" placeholder="" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="updateKey">保存</el-button>
        </el-form-item>
      </div>
      <div v-else-if="form.type === 'hash'" class="value-section">
        <el-form-item>
          <el-button type="primary" @click="showHashChangeDialog('add', null)">新增一行</el-button>
        </el-form-item>
        <el-dialog
            v-model="hashDialogVisible"
            :title="hashDialogTitle"
            width="50%"
        >
          <el-form :model="hashForm" label-width="100px">
            <el-form-item label="字段名称">
              <el-input placeholder="请输入字段名称" v-model="hashForm.field" />
            </el-form-item>
            <el-form-item label="字段的值">
              <el-input placeholder="请输入字段的值" v-model="hashForm.value" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="hashFiledChange">创建</el-button>
              <el-button @click="hashDialogVisible = false">取消</el-button>
            </el-form-item>
          </el-form>
        </el-dialog>
        <div class="table-container">
          <el-table :data="form.value" border style="width: 100%">
            <el-table-column type="index" />
            <el-table-column prop="key" label="Key" />
            <el-table-column prop="value" label="Value" />
            <el-table-column label="操作">
              <template #default="scope">
                <el-button link type="primary" @click="showHashChangeDialog('edit', scope.row)">编辑</el-button>
                <el-popconfirm title="确认删除?" @confirm="deleteHashField([scope.row.key])">
                  <template #reference>
                    <el-button link type="danger">删除</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <div v-else-if="form.type === 'list'" class="value-section">
        <el-form-item>
          <el-button type="primary" @click="listDialogVisible = true">新增一行</el-button>
        </el-form-item>
        <el-dialog
            v-model="listDialogVisible"
            title="LIST 值新增"
            width="60%"
        >
          <el-form :model="listForm" label-width="100px">
            <el-form-item label="字段的值">
              <el-input type="textarea" :rows="6" placeholder="请输入字段的值" v-model="listForm.value" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="createListValue">创建</el-button>
              <el-button @click="listDialogVisible = false">取消</el-button>
            </el-form-item>
          </el-form>
        </el-dialog>
        <div class="table-container">
          <el-table :data="form.value" border style="width: 100%">
            <el-table-column type="index" />
            <el-table-column prop="value" label="Value" />
            <el-table-column label="操作">
              <template #default="scope">
                <el-popconfirm title="确认删除?" @confirm="deleteListItem(scope.row.value)">
                  <template #reference>
                    <el-button link type="danger">删除</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <div v-else-if="form.type === 'set'" class="value-section">
        <el-form-item>
          <el-button type="primary" @click="setDialogVisible = true">新增一行</el-button>
        </el-form-item>
        <el-dialog
            v-model="setDialogVisible"
            title="SET 值新增"
            width="60%"
        >
          <el-form :model="setForm" label-width="100px">
            <el-form-item label="字段的值">
              <el-input type="textarea" :rows="6" placeholder="请输入字段的值" v-model="setForm.value" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="createSetValue">创建</el-button>
              <el-button @click="setDialogVisible = false">取消</el-button>
            </el-form-item>
          </el-form>
        </el-dialog>
        <div class="table-container">
          <el-table :data="form.value" border style="width: 100%">
            <el-table-column type="index" />
            <el-table-column prop="value" label="Value" />
            <el-table-column label="操作">
              <template #default="scope">
                <el-popconfirm title="确认删除?"  @confirm="deleteSetItem(scope.row.value)">
                  <template #reference>
                    <el-button link type="danger">删除</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <div v-else-if="form.type === 'zset'" class="value-section">
        <el-form-item>
          <el-button type="primary" @click="zsetDialogVisible = true">新增一行</el-button>
        </el-form-item>
        <el-dialog
            v-model="zsetDialogVisible"
            title="ZSET 值新增"
            width="60%"
        >
          <el-form :model="zsetForm" label-width="100px">
            <el-form-item label="字段的分值">
              <el-input type="number" placeholder="请输入字段的分值" v-model="zsetForm.score" />
            </el-form-item>
            <el-form-item label="字段的值">
              <el-input type="textarea" :rows="6" placeholder="请输入字段的值" v-model="zsetForm.member" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="createZSetValue">创建</el-button>
              <el-button @click="zsetDialogVisible = false">取消</el-button>
            </el-form-item>
          </el-form>
        </el-dialog>
        <div class="table-container">
          <el-table :data="form.value" border style="width: 100%">
            <el-table-column type="index" />
            <el-table-column prop="Score" label="Score" />
            <el-table-column prop="Member" label="Member" />
            <el-table-column label="操作">
              <template #default="scope">
                <el-popconfirm title="确认删除?"  @confirm="deleteZSetMember(scope.row.Member)">
                  <template #reference>
                    <el-button link type="danger">删除</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </el-form>
  </main>
</template>

<script setup>
import {ref, watch} from 'vue'
import {
  GetKeyValue,
  HashAddOrUpdateField,
  HashFieldDelete, ListValueCreate,
  ListValueDelete, SetValueCreate, SetValueDelete,
  UpdateKeyValue, ZSetValueCreate, ZSetValueDelete
} from "../../../wailsjs/go/redis/Redis";
import {ElNotification} from "element-plus";

let props = defineProps(['keyDB', 'keyConnIdentity', 'keyKey'])
let form = ref({})
let hashDialogVisible = ref(false)
let hashDialogTitle = ref("")
let hashForm = ref({})
let listDialogVisible = ref(false)
let listForm = ref({})
let setDialogVisible = ref(false)
let setForm = ref({})
let zsetDialogVisible = ref(false)
let zsetForm = ref({})

watch(()=>props.keyKey, () => {
  getTheValue()
})

function getTheValue() {
  GetKeyValue({conn_identity:props.keyConnIdentity, db: props.keyDB, key: props.keyKey}).then(res => {
    if (res.code !== 200) {
      ElNotification({
        title:res.msg,
        type: "error",
      })
      return
    }
    form.value = res.data
    form.value.key = props.keyKey
  })
}

function updateKey() {
  UpdateKeyValue({conn_identity: props.keyConnIdentity, key: props.keyKey, db: props.keyDB, value: form.value.value, ttl: form.value.ttl}).then(res => {
    if (res.code !== 200) {
      ElNotification({
        title:res.msg,
        type: "error",
      })
      return
    }
    ElNotification({
      title:res.msg,
      type: "success",
    })
  })
}

function deleteHashField(fields) {
  HashFieldDelete({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, field:fields}).then(res => {
    if (res.code !== 200) {
      ElNotification({
        title:res.msg,
        type: "error",
      })
      return
    }
    ElNotification({
      title:res.msg,
      type: "success",
    })
    getTheValue()
  })
}

function showHashChangeDialog (type, hash) {
  hashDialogVisible.value = true
  if (type === "add") {
    hashDialogTitle.value = "HASH 字段新增"
    hashForm.value = {}
  } else {
    hashDialogTitle.value = "HASH 字段更新"
    hashForm.value = {
      field: hash.key,
      value: hash.value
    }
  }
}

function hashFiledChange () {
  HashAddOrUpdateField({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, field: hashForm.value.field, value: hashForm.value.value}).then(res => {
    hashDialogVisible.value = false
    if (res.code !== 200) {
      ElNotification({
        title:res.msg,
        type: "error",
      })
      return
    }
    ElNotification({
      title:res.msg,
      type: "success",
    })
    getTheValue()
  })
}

function deleteListItem(value) {
  ListValueDelete({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, value: value}).then(res => {
    if (res.code !== 200) {
      ElNotification({
        title:res.msg,
        type: "error",
      })
      return
    }
    ElNotification({
      title:res.msg,
      type: "success",
    })
    getTheValue()
  })
}

function createListValue() {
  ListValueCreate({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, value: listForm.value.value}).then(res => {
    listDialogVisible.value = false
    if (res.code !== 200) {
      ElNotification({
        title:res.msg,
        type: "error",
      })
      return
    }
    ElNotification({
      title:res.msg,
      type: "success",
    })
    getTheValue()
  })
}

function deleteSetItem(value) {
  SetValueDelete({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, value: value}).then(res => {
    if (res.code !== 200) {
      ElNotification({
        title:res.msg,
        type: "error",
      })
      return
    }
    ElNotification({
      title:res.msg,
      type: "success",
    })
    getTheValue()
  })
}

function createSetValue() {
  SetValueCreate({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, value: setForm.value.value}).then(res => {
    setDialogVisible.value = false
    if (res.code !== 200) {
      ElNotification({
        title:res.msg,
        type: "error",
      })
      return
    }
    ElNotification({
      title:res.msg,
      type: "success",
    })
    getTheValue()
  })
}

function deleteZSetMember(member) {
  ZSetValueDelete({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, member: member}).then(res => {
    if (res.code !== 200) {
      ElNotification({
        title:res.msg,
        type: "error",
      })
      return
    }
    ElNotification({
      title:res.msg,
      type: "success",
    })
    getTheValue()
  })
}

function createZSetValue() {
  ZSetValueCreate({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, score: +(zsetForm.value.score), member: zsetForm.value.member}).then(res => {
    zsetDialogVisible.value = false
    if (res.code !== 200) {
      ElNotification({
        title:res.msg,
        type: "error",
      })
      return
    }
    ElNotification({
      title:res.msg,
      type: "success",
    })
    getTheValue()
  })
}
</script>

<style scoped>
.form-row {
  display: flex;
  flex-wrap: nowrap;
}

.form-row :deep(.el-col) {
  display: flex;
  flex-direction: column;
}

.form-row :deep(.el-form-item) {
  margin-bottom: 20px;
  flex: 1;
}

.keyvalue-main {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.keyvalue-form {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.value-section {
  flex: 1;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.table-container {
  overflow: hidden;
  max-height: 75vh;
  border: 1px solid #ebeef5;
  border-radius: 4px;
}

/* 为 string 类型的文本区域也添加滚动限制 */
.value-section:has(.el-textarea) {
  overflow: hidden;
  max-height: 75vh;
}

/* 确保表格在容器内正确显示 */
:deep(.el-table) {
  height: 100%;
}

:deep(.el-table .el-table__body-wrapper) {
  overflow: hidden;
}

.custom-string-textarea :deep(.el-textarea__inner){
  min-height: 200px;
  height: 500px;
}
</style>