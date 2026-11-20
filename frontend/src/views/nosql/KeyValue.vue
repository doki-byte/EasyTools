<template>
  <main v-if="keyKey !== undefined" class="keyvalue-main">
    <div class="keyvalue-header">
      <div class="key-info">
        <div class="key-label">键名</div>
        <div class="key-value">{{ form.key }}</div>
      </div>
      <div class="key-meta">
        <div class="meta-item">
          <span class="meta-label">类型:</span>
          <el-tag :type="getTypeTagType(form.type)" size="small">{{ form.type || '-' }}</el-tag>
        </div>
        <div class="meta-item">
          <span class="meta-label">TTL:</span>
          <el-input 
            v-model="form.ttl" 
            placeholder="永久" 
            size="small"
            style="width: 100px;"
          />
        </div>
      </div>
    </div>
    
    <el-form :model="form" class="keyvalue-form">
      <div v-if="form.type === 'string'" class="value-section">
        <div class="section-header">
          <span class="section-title">值</span>
          <el-button type="primary" size="small" @click="updateKey">保存</el-button>
        </div>
        <el-input 
          type="textarea" 
          :autosize="{ minRows: 8, maxRows: 20 }" 
          v-model="form.value" 
          placeholder="请输入值..."
          class="value-textarea"
        />
      </div>
      <div v-else-if="form.type === 'hash'" class="value-section">
        <div class="section-header">
          <span class="section-title">Hash 字段</span>
          <el-button type="primary" size="small" @click="addHashRow">新增字段</el-button>
        </div>
        <div class="table-container">
          <el-table :data="displayHashData" border style="width: 100%" size="small">
            <el-table-column type="index" width="50" />
            <el-table-column prop="key" label="Key">
              <template #default="scope">
                <el-input 
                  v-if="scope.row.editing || scope.row.isNew" 
                  v-model="scope.row.key" 
                  size="small"
                  placeholder="字段名称"
                />
                <span v-else>{{ scope.row.key }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="value" label="Value">
              <template #default="scope">
                <el-input 
                  v-if="scope.row.editing || scope.row.isNew" 
                  v-model="scope.row.value" 
                  size="small"
                  placeholder="字段的值"
                />
                <span v-else>{{ scope.row.value }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="150">
              <template #default="scope">
                <template v-if="scope.row.editing || scope.row.isNew">
                  <el-button link type="primary" size="small" @click="saveHashRow(scope.row)">保存</el-button>
                  <el-button link size="small" @click="cancelHashRow(scope.row)">取消</el-button>
                </template>
                <template v-else>
                  <el-button link type="primary" size="small" @click="editHashRow(scope.row)">编辑</el-button>
                  <el-button link type="danger" size="small" @click="deleteHashField([scope.row.key])">删除</el-button>
                </template>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <div v-else-if="form.type === 'list'" class="value-section">
        <div class="section-header">
          <span class="section-title">List 元素</span>
          <el-button type="primary" size="small" @click="addListRow">新增元素</el-button>
        </div>
        <div class="table-container">
          <el-table :data="displayListData" border style="width: 100%" size="small">
            <el-table-column type="index" width="50" />
            <el-table-column prop="value" label="Value">
              <template #default="scope">
                <el-input 
                  v-if="scope.row.editing || scope.row.isNew" 
                  v-model="scope.row.value" 
                  type="textarea"
                  :autosize="{ minRows: 2, maxRows: 4 }"
                  size="small"
                  placeholder="请输入值"
                />
                <span v-else>{{ scope.row.value }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120">
              <template #default="scope">
                <template v-if="scope.row.editing || scope.row.isNew">
                  <el-button link type="primary" size="small" @click="saveListRow(scope.row)">保存</el-button>
                  <el-button link size="small" @click="cancelListRow(scope.row)">取消</el-button>
                </template>
                <template v-else>
                  <el-button link type="primary" size="small" @click="editListRow(scope.row)">编辑</el-button>
                  <el-button link type="danger" size="small" @click="deleteListItem(scope.row.value)">删除</el-button>
                </template>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <div v-else-if="form.type === 'set'" class="value-section">
        <div class="section-header">
          <span class="section-title">Set 元素</span>
          <el-button type="primary" size="small" @click="addSetRow">新增元素</el-button>
        </div>
        <div class="table-container">
          <el-table :data="displaySetData" border style="width: 100%" size="small">
            <el-table-column type="index" width="50" />
            <el-table-column prop="value" label="Value">
              <template #default="scope">
                <el-input 
                  v-if="scope.row.editing || scope.row.isNew" 
                  v-model="scope.row.value" 
                  type="textarea"
                  :autosize="{ minRows: 2, maxRows: 4 }"
                  size="small"
                  placeholder="请输入值"
                />
                <span v-else>{{ scope.row.value }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120">
              <template #default="scope">
                <template v-if="scope.row.editing || scope.row.isNew">
                  <el-button link type="primary" size="small" @click="saveSetRow(scope.row)">保存</el-button>
                  <el-button link size="small" @click="cancelSetRow(scope.row)">取消</el-button>
                </template>
                <template v-else>
                  <el-button link type="primary" size="small" @click="editSetRow(scope.row)">编辑</el-button>
                  <el-button link type="danger" size="small" @click="deleteSetItem(scope.row.value)">删除</el-button>
                </template>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <div v-else-if="form.type === 'zset'" class="value-section">
        <div class="section-header">
          <span class="section-title">ZSet 元素</span>
          <el-button type="primary" size="small" @click="addZSetRow">新增元素</el-button>
        </div>
        <div class="table-container">
          <el-table :data="displayZSetData" border style="width: 100%" size="small">
            <el-table-column type="index" width="50" />
            <el-table-column prop="Score" label="Score" width="150">
              <template #default="scope">
                <el-input 
                  v-if="scope.row.editing || scope.row.isNew" 
                  v-model.number="scope.row.Score" 
                  type="number"
                  size="small"
                  placeholder="分值"
                />
                <span v-else>{{ scope.row.Score }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="Member" label="Member">
              <template #default="scope">
                <el-input 
                  v-if="scope.row.editing || scope.row.isNew" 
                  v-model="scope.row.Member" 
                  type="textarea"
                  :autosize="{ minRows: 2, maxRows: 4 }"
                  size="small"
                  placeholder="请输入值"
                />
                <span v-else>{{ scope.row.Member }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120">
              <template #default="scope">
                <template v-if="scope.row.editing || scope.row.isNew">
                  <el-button link type="primary" size="small" @click="saveZSetRow(scope.row)">保存</el-button>
                  <el-button link size="small" @click="cancelZSetRow(scope.row)">取消</el-button>
                </template>
                <template v-else>
                  <el-button link type="primary" size="small" @click="editZSetRow(scope.row)">编辑</el-button>
                  <el-button link type="danger" size="small" @click="deleteZSetMember(scope.row.Member)">删除</el-button>
                </template>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <div v-else-if="form.type === 'collection'" class="value-section">
        <div class="section-header">
          <span class="section-title">文档列表</span>
          <el-button type="primary" size="small" @click="addMongoDocRow">新增文档</el-button>
        </div>
        <div class="table-container">
          <el-table :data="displayCollectionData" border style="width: 100%" size="small">
            <el-table-column type="index" width="50" />
            <el-table-column prop="key" label="_id" width="200">
              <template #default="scope">
                <span>{{ scope.row.key }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="value" label="文档内容">
              <template #default="scope">
                <el-input 
                  v-if="scope.row.editing || scope.row.isNew" 
                  v-model="scope.row.value" 
                  type="textarea"
                  :autosize="{ minRows: 4, maxRows: 10 }"
                  size="small"
                  placeholder="输入 JSON 格式的文档内容"
                />
                <pre v-else class="json-content">{{ formatJSON(scope.row.value) }}</pre>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="150">
              <template #default="scope">
                <template v-if="scope.row.editing || scope.row.isNew">
                  <el-button link type="primary" size="small" @click="saveMongoDocRow(scope.row)">保存</el-button>
                  <el-button link size="small" @click="cancelMongoDocRow(scope.row)">取消</el-button>
                </template>
                <template v-else>
                  <el-button link type="primary" size="small" @click="editMongoDocRow(scope.row)">编辑</el-button>
                  <el-button link type="danger" size="small" @click="deleteMongoDoc(scope.row.key)">删除</el-button>
                </template>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </el-form>
  </main>
</template>

<script setup>
import {ref, watch, computed} from 'vue'
import {
  GetKeyValue,
  HashAddOrUpdateField,
  HashFieldDelete, ListValueCreate,
  ListValueDelete, SetValueCreate, SetValueDelete,
  UpdateKeyValue, ZSetValueCreate, ZSetValueDelete,
  CreateKeyValue, MongoDocumentDelete
} from "../../../wailsjs/go/nosql/NoSQL";
import {ElMessage} from "element-plus";
import { confirmDangerousAction, DANGER_DELETE_TOKEN } from '@/utils/dangerConfirm';

let props = defineProps(['keyDB', 'keyConnIdentity', 'keyKey'])
let form = ref({})
let hashEditIndex = ref(-1)
let listEditIndex = ref(-1)
let setEditIndex = ref(-1)
let zsetEditIndex = ref(-1)
let mongoDocEditIndex = ref(-1)

watch(()=>props.keyKey, (newKey) => {
  // 如果keyKey为null或undefined，不调用getTheValue
  if (newKey !== null && newKey !== undefined) {
    getTheValue()
  }
})

// 计算属性：Hash 显示数据
const displayHashData = computed(() => {
  if (!form.value.value || !Array.isArray(form.value.value)) return []
  return form.value.value.map((item, index) => ({
    ...item,
    editing: index === hashEditIndex.value,
    isNew: false,
    originalIndex: index
  }))
})

// 计算属性：List 显示数据
const displayListData = computed(() => {
  if (!form.value.value || !Array.isArray(form.value.value)) return []
  return form.value.value.map((item, index) => ({
    ...item,
    editing: index === listEditIndex.value,
    isNew: false,
    originalIndex: index
  }))
})

// 计算属性：Set 显示数据
const displaySetData = computed(() => {
  if (!form.value.value || !Array.isArray(form.value.value)) return []
  return form.value.value.map((item, index) => ({
    ...item,
    editing: index === setEditIndex.value,
    isNew: false,
    originalIndex: index
  }))
})

// 计算属性：ZSet 显示数据
const displayZSetData = computed(() => {
  if (!form.value.value || !Array.isArray(form.value.value)) return []
  return form.value.value.map((item, index) => ({
    ...item,
    editing: index === zsetEditIndex.value,
    isNew: false,
    originalIndex: index
  }))
})

// 计算属性：Collection (MongoDB) 显示数据
const displayCollectionData = computed(() => {
  if (!form.value.value || !Array.isArray(form.value.value)) return []
  return form.value.value.map((item, index) => ({
    key: item.key || item.Key || '-',
    value: item.value || item.Value || '',
    editing: index === mongoDocEditIndex.value,
    isNew: item.isNew || false, // 保留原始的 isNew 属性
    originalIndex: index,
    originalKey: item.originalKey !== undefined ? item.originalKey : (item.key || item.Key),
    originalValue: item.originalValue !== undefined ? item.originalValue : (item.value || item.Value)
  }))
})

function getTheValue() {
  // 如果keyKey为空，直接返回，不调用API
  if (!props.keyKey || props.keyKey === null || props.keyKey === undefined) {
    return
  }
  
  GetKeyValue({conn_identity:props.keyConnIdentity, db: props.keyDB, key: props.keyKey}).then(res => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    form.value = res.data
    form.value.key = props.keyKey
    // 重置编辑索引
    hashEditIndex.value = -1
    listEditIndex.value = -1
    setEditIndex.value = -1
    zsetEditIndex.value = -1
    mongoDocEditIndex.value = -1
  })
}

function updateKey() {
  UpdateKeyValue({conn_identity: props.keyConnIdentity, key: props.keyKey, db: props.keyDB, value: form.value.value, ttl: form.value.ttl}).then(res => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success(`${res.msg}`)
  })
}

async function deleteHashField(fields) {
  try {
    await confirmDangerousAction({
      title: '删除字段',
      message: `将从键「${props.keyKey}」中删除所选 Hash 字段，不可恢复。`,
      expectedInput: DANGER_DELETE_TOKEN
    })
  } catch {
    return
  }
  HashFieldDelete({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, field:fields}).then(res => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success(`${res.msg}`)
    getTheValue()
  })
}

// Hash 相关函数
function addHashRow() {
  if (!form.value.value) form.value.value = []
  const newRow = {
    key: '',
    value: '',
    isNew: true
  }
  form.value.value.push(newRow)
  hashEditIndex.value = form.value.value.length - 1
}

function editHashRow(row) {
  hashEditIndex.value = row.originalIndex
}

function cancelHashRow(row) {
  if (row.isNew) {
    // 找到新添加的行并删除
    const index = form.value.value.findIndex((item, idx) => idx === row.originalIndex && item.isNew)
    if (index !== -1) {
      form.value.value.splice(index, 1)
    }
  }
  hashEditIndex.value = -1
  // 如果有新增行被取消，重新加载数据
  if (row.isNew) {
    getTheValue()
  }
}

function saveHashRow(row) {
  if (!row.key || !row.value) {
    ElMessage.warning('字段名称和值不能为空')
    return
  }
  
  // 如果是编辑模式（不是新增），需要检查字段名是否改变
  if (!row.isNew && row.originalIndex !== undefined) {
    const oldField = form.value.value[row.originalIndex]
    if (oldField && oldField.key !== row.key) {
      // 字段名改变了，需要先删除旧字段再添加新字段
      HashFieldDelete({
        conn_identity: props.keyConnIdentity,
        db: props.keyDB,
        key: props.keyKey,
        field: [oldField.key]
      }).then(deleteRes => {
        if (deleteRes.code !== 200) {
          ElMessage.error(`删除旧字段失败: ${deleteRes.msg}`)
          return
        }
        // 删除成功后，添加新字段
        HashAddOrUpdateField({
          conn_identity: props.keyConnIdentity, 
          db: props.keyDB, 
          key: props.keyKey, 
          field: row.key, 
          value: row.value
        }).then(res => {
          hashEditIndex.value = -1
          if (res.code !== 200) {
            ElMessage.error(`${res.msg}`)
            return
          }
          ElMessage.success('更新成功')
          getTheValue()
        })
      })
      return
    }
  }
  
  // 新增或字段名未改变的情况，直接添加或更新
  HashAddOrUpdateField({
    conn_identity: props.keyConnIdentity, 
    db: props.keyDB, 
    key: props.keyKey, 
    field: row.key, 
    value: row.value
  }).then(res => {
    hashEditIndex.value = -1
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success(`${res.msg}`)
    getTheValue()
  })
}

async function deleteListItem(value) {
  try {
    await confirmDangerousAction({
      title: '删除元素',
      message: `将从键「${props.keyKey}」的 List 中删除该元素，不可恢复。`,
      expectedInput: DANGER_DELETE_TOKEN
    })
  } catch {
    return
  }
  ListValueDelete({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, value: value}).then(res => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success(`${res.msg}`)
    getTheValue()
  })
}

// List 相关函数
function addListRow() {
  if (!form.value.value) form.value.value = []
  const newRow = {
    value: '',
    isNew: true
  }
  form.value.value.push(newRow)
  listEditIndex.value = form.value.value.length - 1
}

function editListRow(row) {
  listEditIndex.value = row.originalIndex
}

function cancelListRow(row) {
  if (row.isNew) {
    const index = form.value.value.findIndex((item, idx) => idx === row.originalIndex && item.isNew)
    if (index !== -1) {
      form.value.value.splice(index, 1)
    }
  }
  listEditIndex.value = -1
  if (row.isNew) {
    getTheValue()
  }
}

function saveListRow(row) {
  if (!row.value) {
    ElMessage.warning('值不能为空')
    return
  }
  if (row.isNew) {
    // 新增
    ListValueCreate({
      conn_identity: props.keyConnIdentity, 
      db: props.keyDB, 
      key: props.keyKey, 
      value: row.value
    }).then(res => {
      listEditIndex.value = -1
      if (res.code !== 200) {
        ElMessage.error(`${res.msg}`)
        return
      }
      ElMessage.success(`${res.msg}`)
      getTheValue()
    })
  } else {
    // 编辑需要先删除旧值再添加新值（List不支持直接更新）
    const oldValue = form.value.value[row.originalIndex].value
    ListValueDelete({
      conn_identity: props.keyConnIdentity, 
      db: props.keyDB, 
      key: props.keyKey, 
      value: oldValue
    }).then(deleteRes => {
      if (deleteRes.code !== 200) {
        ElMessage.error(`删除旧值失败: ${deleteRes.msg}`)
        return
      }
      ListValueCreate({
        conn_identity: props.keyConnIdentity, 
        db: props.keyDB, 
        key: props.keyKey, 
        value: row.value
      }).then(createRes => {
        listEditIndex.value = -1
        if (createRes.code !== 200) {
          ElMessage.error(`${createRes.msg}`)
          return
        }
        ElMessage.success('更新成功')
        getTheValue()
      })
    })
  }
}

async function deleteSetItem(value) {
  try {
    await confirmDangerousAction({
      title: '删除元素',
      message: `将从键「${props.keyKey}」的 Set 中删除该元素，不可恢复。`,
      expectedInput: DANGER_DELETE_TOKEN
    })
  } catch {
    return
  }
  SetValueDelete({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, value: value}).then(res => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success(`${res.msg}`)
    getTheValue()
  })
}

// Set 相关函数
function addSetRow() {
  if (!form.value.value) form.value.value = []
  const newRow = {
    value: '',
    isNew: true
  }
  form.value.value.push(newRow)
  setEditIndex.value = form.value.value.length - 1
}

function editSetRow(row) {
  setEditIndex.value = row.originalIndex
}

function cancelSetRow(row) {
  if (row.isNew) {
    const index = form.value.value.findIndex((item, idx) => idx === row.originalIndex && item.isNew)
    if (index !== -1) {
      form.value.value.splice(index, 1)
    }
  }
  setEditIndex.value = -1
  if (row.isNew) {
    getTheValue()
  }
}

function saveSetRow(row) {
  if (!row.value) {
    ElMessage.warning('值不能为空')
    return
  }
  if (row.isNew) {
    // 新增
    SetValueCreate({
      conn_identity: props.keyConnIdentity, 
      db: props.keyDB, 
      key: props.keyKey, 
      value: row.value
    }).then(res => {
      setEditIndex.value = -1
      if (res.code !== 200) {
        ElMessage.error(`${res.msg}`)
        return
      }
      ElMessage.success(`${res.msg}`)
      getTheValue()
    })
  } else {
    // 编辑需要先删除旧值再添加新值（Set不支持直接更新）
    const oldValue = form.value.value[row.originalIndex].value
    SetValueDelete({
      conn_identity: props.keyConnIdentity, 
      db: props.keyDB, 
      key: props.keyKey, 
      value: oldValue
    }).then(deleteRes => {
      if (deleteRes.code !== 200) {
        ElMessage.error(`删除旧值失败: ${deleteRes.msg}`)
        return
      }
      SetValueCreate({
        conn_identity: props.keyConnIdentity, 
        db: props.keyDB, 
        key: props.keyKey, 
        value: row.value
      }).then(createRes => {
        setEditIndex.value = -1
        if (createRes.code !== 200) {
          ElMessage.error(`${createRes.msg}`)
          return
        }
        ElMessage.success('更新成功')
        getTheValue()
      })
    })
  }
}

async function deleteZSetMember(member) {
  try {
    await confirmDangerousAction({
      title: '删除成员',
      message: `将从键「${props.keyKey}」的 ZSet 中删除该成员，不可恢复。`,
      expectedInput: DANGER_DELETE_TOKEN
    })
  } catch {
    return
  }
  ZSetValueDelete({conn_identity: props.keyConnIdentity, db: props.keyDB, key: props.keyKey, member: member}).then(res => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success(`${res.msg}`)
    getTheValue()
  })
}

// ZSet 相关函数
function addZSetRow() {
  if (!form.value.value) form.value.value = []
  const newRow = {
    Score: 0,
    Member: '',
    isNew: true
  }
  form.value.value.push(newRow)
  zsetEditIndex.value = form.value.value.length - 1
}

function editZSetRow(row) {
  zsetEditIndex.value = row.originalIndex
}

function cancelZSetRow(row) {
  if (row.isNew) {
    const index = form.value.value.findIndex((item, idx) => idx === row.originalIndex && item.isNew)
    if (index !== -1) {
      form.value.value.splice(index, 1)
    }
  }
  zsetEditIndex.value = -1
  if (row.isNew) {
    getTheValue()
  }
}

function saveZSetRow(row) {
  if (!row.Member || row.Score === undefined || row.Score === null) {
    ElMessage.warning('分值和值不能为空')
    return
  }
  if (row.isNew) {
    // 新增
    ZSetValueCreate({
      conn_identity: props.keyConnIdentity, 
      db: props.keyDB, 
      key: props.keyKey, 
      score: +row.Score, 
      member: row.Member
    }).then(res => {
      zsetEditIndex.value = -1
      if (res.code !== 200) {
        ElMessage.error(`${res.msg}`)
        return
      }
      ElMessage.success(`${res.msg}`)
      getTheValue()
    })
  } else {
    // 编辑需要先删除旧值再添加新值（ZSet不支持直接更新）
    const oldMember = form.value.value[row.originalIndex].Member
    ZSetValueDelete({
      conn_identity: props.keyConnIdentity, 
      db: props.keyDB, 
      key: props.keyKey, 
      member: oldMember
    }).then(deleteRes => {
      if (deleteRes.code !== 200) {
        ElMessage.error(`删除旧值失败: ${deleteRes.msg}`)
        return
      }
      ZSetValueCreate({
        conn_identity: props.keyConnIdentity, 
        db: props.keyDB, 
        key: props.keyKey, 
        score: +row.Score, 
        member: row.Member
      }).then(createRes => {
        zsetEditIndex.value = -1
        if (createRes.code !== 200) {
          ElMessage.error(`${createRes.msg}`)
          return
        }
        ElMessage.success('更新成功')
        getTheValue()
      })
    })
  }
}

function getTypeTagType(type) {
  if (!type) return undefined; // 如果没有类型，返回 undefined（不设置 type prop）
  const typeMap = {
    'string': 'success',
    'hash': 'warning',
    'list': 'info',
    'set': 'info', // 将空字符串改为 'info'
    'zset': 'danger',
    'collection': 'success' // MongoDB 集合类型
  }
  const result = typeMap[type];
  return result !== undefined ? result : undefined; // 如果类型不存在，返回 undefined
}

// 格式化 JSON 内容
function formatJSON(jsonStr) {
  if (!jsonStr) return ''
  try {
    const parsed = typeof jsonStr === 'string' ? JSON.parse(jsonStr) : jsonStr
    return JSON.stringify(parsed, null, 2)
  } catch (e) {
    return jsonStr
  }
}

// MongoDB 文档操作
function addMongoDocRow() {
  // 在数组开头添加新文档行
  const newRow = {
    key: '',
    value: '{}',
    editing: true,
    isNew: true,
    originalIndex: -1
  }
  form.value.value = [newRow, ...form.value.value]
  mongoDocEditIndex.value = 0
}

function editMongoDocRow(row) {
  mongoDocEditIndex.value = row.originalIndex
}

function cancelMongoDocRow(row) {
  if (row.isNew) {
    // 删除新增的行
    form.value.value = form.value.value.filter((item, index) => index !== 0)
  } else {
    // 恢复原始值
    form.value.value[row.originalIndex].value = row.originalValue
  }
  mongoDocEditIndex.value = -1
}

function saveMongoDocRow(row) {
  if (!row.value || row.value.trim() === '') {
    ElMessage.error('文档内容不能为空')
    return
  }
  
  // 验证 JSON 格式
  try {
    JSON.parse(row.value)
  } catch (e) {
    ElMessage.error('无效的 JSON 格式')
    return
  }
  
  if (row.isNew) {
    // 创建新文档
    CreateKeyValue({
      conn_identity: props.keyConnIdentity,
      db: props.keyDB,
      key: props.keyKey, // 集合名称
      value: row.value,
      type: 'collection'
    }).then(res => {
      mongoDocEditIndex.value = -1
      if (res.code !== 200) {
        ElMessage.error(`${res.msg}`)
        return
      }
      ElMessage.success('创建成功')
      getTheValue()
    })
  } else {
    // 更新文档
    UpdateKeyValue({
      conn_identity: props.keyConnIdentity,
      db: props.keyDB,
      key: props.keyKey, // 集合名称
      old_value: row.originalKey, // 文档 _id
      value: row.value,
      ttl: 0
    }).then(res => {
      mongoDocEditIndex.value = -1
      if (res.code !== 200) {
        ElMessage.error(`${res.msg}`)
        return
      }
      ElMessage.success('更新成功')
      getTheValue()
    })
  }
}

async function deleteMongoDoc(docId) {
  if (!docId) {
    ElMessage.error('文档 ID 不能为空')
    return
  }
  try {
    await confirmDangerousAction({
      title: '删除文档',
      message: `将从集合「${props.keyKey}」删除该 MongoDB 文档，不可恢复。`,
      expectedInput: DANGER_DELETE_TOKEN
    })
  } catch {
    return
  }

  MongoDocumentDelete({
    conn_identity: props.keyConnIdentity,
    db: props.keyDB,
    key: props.keyKey, // 集合名称
    doc_id: docId
  }).then(res => {
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success('删除成功')
    getTheValue()
  }).catch(err => {
    ElMessage.error(`删除失败: ${err}`)
  })
}
</script>

<style scoped>
.keyvalue-main {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 12px;
  overflow: hidden;
}

.keyvalue-header {
  padding: 12px;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 6px;
  margin-bottom: 12px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  flex-shrink: 0;
}

.key-info {
  margin-bottom: 10px;
}

.key-label {
  font-size: 11px;
  color: rgba(0, 0, 0, 0.65);
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-weight: 600;
}

.key-value {
  font-size: 14px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
  word-break: break-all;
  padding: 6px 10px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  border: 1px solid rgba(0, 0, 0, 0.1);
}

.key-meta {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}

.meta-label {
  color: rgba(0, 0, 0, 0.7);
  font-size: 12px;
  font-weight: 600;
}

.keyvalue-form {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.value-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 2px solid rgba(0, 0, 0, 0.15);
  flex-shrink: 0;
}

.section-title {
  font-size: 14px;
  font-weight: 700;
  color: rgba(0, 0, 0, 0.85);
}

.value-textarea {
  flex: 1;
  min-height: 0;
}

.value-textarea :deep(.el-textarea__inner) {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.7;
  background: rgba(255, 255, 255, 0.7);
  border: 1px solid rgba(0, 0, 0, 0.15);
  color: rgba(0, 0, 0, 0.9);
  font-weight: 500;
}

.value-textarea :deep(.el-textarea__inner::placeholder) {
  color: rgba(0, 0, 0, 0.4);
}

.value-textarea :deep(.el-textarea__inner):focus {
  border-color: #409EFF;
  background: rgba(255, 255, 255, 0.85);
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.2);
}

.table-container {
  flex: 1;
  overflow: auto;
  border: 1px solid rgba(0, 0, 0, 0.15);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.2);
  min-height: 0;
}

/* 优化表格样式 */
:deep(.el-table) {
  background: transparent;
}

:deep(.el-table th) {
  background: rgba(0, 0, 0, 0.08);
  color: rgba(0, 0, 0, 0.85);
  border-color: rgba(0, 0, 0, 0.15);
  font-weight: 700;
  font-size: 13px;
}

:deep(.el-table td) {
  background: rgba(255, 255, 255, 0.3);
  color: rgba(0, 0, 0, 0.8);
  border-color: rgba(0, 0, 0, 0.1);
  font-size: 13px;
  font-weight: 500;
}

:deep(.el-table tr:hover > td) {
  background: rgba(255, 255, 255, 0.5);
}

:deep(.el-table tr:nth-child(even) > td) {
  background: rgba(255, 255, 255, 0.25);
}

:deep(.el-table tr:nth-child(even):hover > td) {
  background: rgba(255, 255, 255, 0.55);
}

:deep(.el-table--border) {
  border-color: rgba(0, 0, 0, 0.15);
}

:deep(.el-table--border::after) {
  background-color: rgba(0, 0, 0, 0.15);
}

:deep(.el-table--border::before) {
  background-color: rgba(0, 0, 0, 0.15);
}

/* 确保按钮保持默认颜色，不覆盖 */
:deep(.el-button) {
  /* 不设置颜色，使用Element Plus默认样式 */
}

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

/* 优化滚动条 */
.table-container::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.table-container::-webkit-scrollbar-track {
  background: transparent;
}

.table-container::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.table-container::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

/* MongoDB 集合文档 JSON 内容样式 */
.json-content {
  margin: 0;
  padding: 8px;
  background: rgba(0, 0, 0, 0.02);
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 200px;
  overflow-y: auto;
  color: rgba(0, 0, 0, 0.85);
}
</style>
