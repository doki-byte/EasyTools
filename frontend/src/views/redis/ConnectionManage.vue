<template>
  <main>
    <el-button :type="btnType" @click="dialogVisible = true">{{title}}</el-button>
    <el-dialog
        v-model="dialogVisible"
        :title="title"
        width="60%"
    >
      <el-form :model="form" label-width="120px">
        <el-form-item label="连接地址">
          <el-input placeholder="请输入连接地址" v-model="form.addr" />
        </el-form-item>
        <el-form-item label="连接名称">
          <el-input placeholder="请输入连接名称" v-model="form.name" />
        </el-form-item>
        <el-form-item label="端口号">
          <el-input placeholder="请输入端口号" v-model="form.port" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input placeholder="请输入用户名" v-model="form.username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input placeholder="请输入密码" type="password" v-model="form.password" />
        </el-form-item>
        <el-form-item label="连接方式">
          <el-select v-model="form.type" placeholder="请选连接方式" clearable>
            <el-option value="ssh" label="ssh"></el-option>
          </el-select>
        </el-form-item>
        <div v-show="form.type === 'ssh'">
          <el-form-item label="SSH 地址">
            <el-input placeholder="请输入 SSH 地址" v-model="form.ssh_addr" />
          </el-form-item>
          <el-form-item label="SSH 端口号">
            <el-input placeholder="请输入 SSH 端口号" v-model="form.ssh_port" />
          </el-form-item>
          <el-form-item label="SSH 用户名">
            <el-input placeholder="请输入 SSH 用户名" v-model="form.ssh_username" />
          </el-form-item>
          <el-form-item label="SSH 密码">
            <el-input placeholder="请输入 SSH 密码" type="password" v-model="form.ssh_password" />
          </el-form-item>
        </div>
        <el-form-item>
          <el-button type="success" @click="testConnection" :loading="testing">测试连接</el-button>
          <el-button v-if="data === undefined" type="primary" @click="createConnection">创建</el-button>
          <el-button v-else type="primary" @click="editConnection">保存</el-button>
          <el-button @click="dialogVisible = false">取消</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>

  </main>
</template>

<script setup>
import {ref, reactive, watch, watchEffect} from "vue";
import {ConnectionCreate, ConnectionEdit, ConnectionTest} from "../../../wailsjs/go/redis/Redis";
import {ElNotification, ElMessage} from "element-plus";

let props = defineProps(['title', 'btnType', 'data'])
const dialogVisible = ref(false)
const testing = ref(false)
const emits = defineEmits(['emit-connection-list'])
let form = reactive({})
if (props.data !== undefined) {
  form = props.data
}

// 测试连接功能
function testConnection() {
  // 基础验证
  if (!form.addr) {
    ElMessage.error("连接地址不能为空")
    return
  }

  testing.value = true

  ConnectionTest(form).then(res => {
    testing.value = false
    if (res.code !== 200) {
      ElNotification({
        title: '连接测试失败',
        message: res.msg,
        type: "error",
        duration: 5000
      })
      return
    }
    ElNotification({
      title: '连接测试成功',
      message: 'Redis 连接正常',
      type: "success",
    })
  }).catch(err => {
    testing.value = false
    ElNotification({
      title: '连接测试失败',
      message: err.message || '未知错误',
      type: "error",
      duration: 5000
    })
  })
}

function createConnection() {
  ConnectionCreate(form).then(res => {
    dialogVisible.value = false
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
    // 获取新的连接地址
    emits('emit-connection-list')
  })
}

function editConnection() {
  ConnectionEdit(form).then(res => {
    dialogVisible.value = false
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
    // 获取新的连接地址
    emits('emit-connection-list')
  })
}
</script>

<style scoped>

</style>