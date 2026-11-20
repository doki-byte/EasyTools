<template>
  <main>
    <el-button :type="btnType" @click="dialogVisible = true">{{title}}</el-button>
    <el-dialog
        v-model="dialogVisible"
        :title="form.identity ? '编辑连接配置' : title"
        width="60%"
        append-to-body
        destroy-on-close
        @close="handleDialogClose"
    >
      <el-form :model="form" label-width="120px">
        <el-form-item label="数据库类型">
          <el-select v-model="form.db_type" placeholder="选择数据库类型" @change="handleDbTypeChange" :disabled="!!data">
            <el-option label="Redis" value="redis" />
            <el-option label="MongoDB" value="mongodb" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称">
          <el-input placeholder="请输入连接名称" v-model="form.name" />
        </el-form-item>
        <el-form-item label="主机">
          <el-input placeholder="请输入主机" v-model="form.addr" />
        </el-form-item>
        <el-form-item label="端口号">
          <el-input placeholder="自动填充默认端口" v-model="form.port" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input 
            placeholder="请输入用户名" 
            v-model="form.username" 
          />
        </el-form-item>
        <el-form-item label="密码">
          <el-input 
            placeholder="请输入密码" 
            type="password" 
            show-password 
            v-model="form.password"
          />
        </el-form-item>
        <el-form-item label="连接方式">
          <el-select v-model="form.type" placeholder="请选择连接方式" clearable @change="handleConnectionTypeChange">
            <el-option value="direct" label="直连" />
            <el-option value="ssh" label="SSH隧道" />
          </el-select>
        </el-form-item>
        <div v-show="form.type === 'ssh'">
          <el-form-item label="SSH主机">
            <el-select
              v-model="form.ssh_host_id"
              placeholder="选择SSH主机或手动输入"
              filterable
              allow-create
              default-first-option
              @change="handleSshHostChange"
              clearable
            >
              <el-option
                v-for="host in sshHostList"
                :key="host.id"
                :label="`${host.name} (${host.address}:${host.port})`"
                :value="host.id"
              />
            </el-select>
          </el-form-item>
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
            <el-input placeholder="请输入 SSH 密码" type="password" show-password v-model="form.ssh_password" />
          </el-form-item>
        </div>
        <el-form-item>
          <el-button type="success" @click="testConnection" :loading="testing">测试连接</el-button>
          <el-button v-if="!form.identity" type="primary" @click="createConnection">创建</el-button>
          <el-button v-else type="primary" @click="editConnection">保存</el-button>
          <el-button @click="handleDialogClose">取消</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>

  </main>
</template>

<script setup>
import {ref, reactive, watch, onMounted} from "vue";
import {ConnectionCreate, ConnectionEdit, ConnectionTest} from "../../../wailsjs/go/nosql/NoSQL";
import {GetPortConfig} from "../../../wailsjs/go/controller/Server";
import {ElMessage} from "element-plus";
import axios from "axios";
import { getToken } from "../../utils/token";

let props = defineProps(['title', 'btnType', 'data'])
const dialogVisible = ref(false)
const testing = ref(false)
const emits = defineEmits(['emit-connection-list'])
let form = reactive({
  identity: '', // 连接唯一标识，用于区分编辑/创建模式
  db_type: 'redis', // 默认类型为 Redis
  addr: '',
  port: '',
  username: '',
  password: '',
  name: '',
  type: '',
  ssh_host_id: null,
  ssh_addr: '',
  ssh_port: '',
  ssh_username: '',
  ssh_password: ''
})
if (props.data !== undefined) {
  Object.assign(form, {
    ...props.data,
    db_type: props.data.db_type || 'redis' // 如果没有 db_type，默认为 redis
  })
}


const portConfig = ref(null)
const sshApiClient = ref(null)
const sshHostList = ref([])

// 初始化SSH API客户端
const createSshHttpClient = async () => {
  try {
    if (!portConfig.value) {
      portConfig.value = await GetPortConfig();
    }
    const port = portConfig.value?.sshPort || 52868;
    const host = '127.0.0.1';
    const baseURL = `http://${host}:${port}`;
    const client = axios.create({
      baseURL: baseURL,
      timeout: 20000,
    });
    // 添加请求拦截器，添加token认证
    client.interceptors.request.use(
      (req) => {
        req.headers.Time = String(Date.now());
        const token = getToken();
        if (token) {
          req.headers.Authorization = token;
        }
        return req;
      },
      (err) => Promise.reject(err)
    );
    sshApiClient.value = client;
  } catch (error) {
    console.error('创建SSH API客户端失败:', error);
  }
}

// 加载SSH主机列表
const loadSshHostList = async () => {
  if (!sshApiClient.value) {
    await createSshHttpClient();
  }
  if (!sshApiClient.value) {
    console.error('SSH API客户端未初始化');
    return;
  }
  try {
    const response = await sshApiClient.value.get('/api/conn_conf');
    if (response.data.code === 0) {
      sshHostList.value = response.data.data || [];
    }
  } catch (e) {
    console.error('加载SSH主机列表失败:', e);
    sshHostList.value = [];
  }
}

// 处理数据库类型变化
const handleDbTypeChange = (dbType) => {
  if (!dbType) return;
  
  const defaults = {
    redis: { port: '6379', username: '' },
    mongodb: { port: '27017', username: '' }
  };
  
  // 获取所有默认端口值，用于检测是否需要更新
  const allDefaultPorts = Object.values(defaults).map(d => d.port);
  
  const defaultValues = defaults[dbType];
  if (defaultValues) {
    // 如果端口为空，或者是其他类型的默认端口，则更新为新类型的默认端口
    if (!form.port || allDefaultPorts.includes(form.port)) {
      form.port = defaultValues.port;
    }
    
    // Redis 的用户名是可选的，不需要强制设置
  }
}

// 处理连接方式变化
const handleConnectionTypeChange = (connectionType) => {
  if (connectionType !== 'ssh') {
    form.ssh_host_id = null;
    form.ssh_addr = '';
    form.ssh_port = '';
    form.ssh_username = '';
    form.ssh_password = '';
  }
}

// 处理SSH主机选择变化
const handleSshHostChange = (hostId) => {
  if (hostId) {
    const numericId = typeof hostId === 'string' ? parseInt(hostId, 10) : hostId;
    if (!isNaN(numericId) && typeof numericId === 'number') {
      const host = sshHostList.value.find(h => h.id === numericId);
      if (host) {
        form.ssh_addr = host.address || '';
        form.ssh_port = String(host.port || 22);
        form.ssh_username = host.user || '';
        form.ssh_password = host.pwd || '';
      }
    }
  }
}

// 监听对话框打开，加载SSH主机列表
watch(dialogVisible, (newVal) => {
  if (newVal) {
    loadSshHostList();
    // 如果没有设置数据库类型，默认设为 Redis
    if (!form.db_type) {
      form.db_type = 'redis';
    }
    // 如果已有SSH配置，设置连接方式
    if (form.ssh_addr || form.ssh_port || form.ssh_username || form.ssh_password) {
      if (!form.type) {
        form.type = 'ssh';
      }
    }
    // 根据数据库类型设置默认端口（如果没有端口或者是默认端口）
    // 注意：只有在没有明确设置端口时才设置默认端口，避免覆盖已填充的数据
    if (form.db_type && !form.port) {
      const defaultPorts = {
        redis: '6379',
        mongodb: '27017'
      };
      form.port = defaultPorts[form.db_type] || '6379';
    }
  }
})

// 重置表单
const resetForm = () => {
  form.identity = '';
  form.db_type = 'redis';
  form.addr = '';
  form.port = '';
  form.username = '';
  form.password = '';
  form.name = '';
  form.type = '';
  form.ssh_host_id = null;
  form.ssh_addr = '';
  form.ssh_port = '';
  form.ssh_username = '';
  form.ssh_password = '';
}

// 对话框关闭处理
const handleDialogClose = () => {
  resetForm();
  dialogVisible.value = false;
}

// 暴露填充表单的方法给父组件
const fillForm = (connectionInfo) => {
  if (!connectionInfo) return;
  
  console.log('[ConnectionManage] fillForm 接收到连接信息:', connectionInfo)
  
  // 设置identity，用于区分编辑/创建模式
  if (connectionInfo.identity) {
    form.identity = connectionInfo.identity;
  }
  
  if (connectionInfo.host) {
    form.addr = connectionInfo.host;
  } else if (connectionInfo.addr) {
    form.addr = connectionInfo.addr;
  }
  if (connectionInfo.port) {
    form.port = connectionInfo.port;
  }
  if (connectionInfo.username !== undefined) {
    form.username = connectionInfo.username;
  }
  // 对于Redis，密码字段必须正确填充（包括空字符串）
  if (connectionInfo.password !== undefined && connectionInfo.password !== null) {
    form.password = connectionInfo.password;
    console.log('[ConnectionManage] 填充密码字段:', connectionInfo.password ? '***' : '(空)')
  } else if (connectionInfo.password === '') {
    // 允许空字符串（表示无密码）
    form.password = '';
    console.log('[ConnectionManage] 密码为空字符串')
  }
  if (connectionInfo.name) {
    form.name = connectionInfo.name;
  }
  if (connectionInfo.db_type) {
    form.db_type = connectionInfo.db_type;
    // 根据数据库类型设置默认端口（如果端口为空）
    handleDbTypeChange(connectionInfo.db_type);
  }
  if (connectionInfo.type !== undefined) {
    form.type = connectionInfo.type;
  }
  if (connectionInfo.ssh_addr) {
    form.ssh_addr = connectionInfo.ssh_addr;
  }
  if (connectionInfo.ssh_port) {
    form.ssh_port = connectionInfo.ssh_port;
  }
  if (connectionInfo.ssh_username) {
    form.ssh_username = connectionInfo.ssh_username;
  }
  if (connectionInfo.ssh_password !== undefined) {
    form.ssh_password = connectionInfo.ssh_password;
  }
  // 加载SSH主机列表（如果需要）
  loadSshHostList();
}

// 暴露方法给父组件
defineExpose({
  form,
  dialogVisible,
  fillForm
})

onMounted(async () => {
  await createSshHttpClient();
})

// 测试连接功能
function testConnection() {
  // 基础验证
  if (!form.addr) {
    ElMessage.error("主机不能为空")
    return
  }
  
  if (!form.port || form.port.trim() === '') {
    ElMessage.error("端口号不能为空")
    return
  }

  // Redis 不需要用户名（只有密码），所以这里不需要验证用户名

  // 如果选择了SSH隧道，验证SSH配置完整性
  if (form.type === 'ssh') {
    if (!form.ssh_addr || form.ssh_addr.trim() === '') {
      ElMessage.warning("SSH隧道模式下，SSH地址不能为空")
      return
    }
    if (!form.ssh_username || form.ssh_username.trim() === '') {
      ElMessage.warning("SSH隧道模式下，SSH用户名不能为空")
      return
    }
    if (!form.ssh_password || form.ssh_password.trim() === '') {
      ElMessage.warning("SSH隧道模式下，SSH密码不能为空")
      return
    }
    // 如果SSH端口为空，设置默认值
    if (!form.ssh_port || form.ssh_port.trim() === '') {
      form.ssh_port = '22'
    }
  }

  testing.value = true

  ConnectionTest(form).then(res => {
    testing.value = false
    if (res.code !== 200) {
      // 显示具体的错误信息
      const errorMsg = res.msg || '连接测试失败'
      ElMessage.error(errorMsg)
      return
    }
    // 根据数据库类型显示不同的成功消息
    const dbTypeName = form.db_type === 'mongodb' || form.db_type === 'mongo' ? 'MongoDB' : 'Redis'
    ElMessage.success(`${dbTypeName} 连接正常`)
  }).catch(err => {
    testing.value = false
    // 显示详细的错误信息
    const errorMsg = err.message || err.toString() || '未知错误'
    ElMessage.error(`连接测试失败: ${errorMsg}`)
  })
}

function createConnection() {
  ConnectionCreate(form).then(res => {
    handleDialogClose()
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success(`${res.msg}`)
    // 获取新的连接地址
    emits('emit-connection-list')
  })
}

function editConnection() {
  ConnectionEdit(form).then(res => {
    handleDialogClose()
    if (res.code !== 200) {
      ElMessage.error(`${res.msg}`)
      return
    }
    ElMessage.success(`${res.msg}`)
    // 获取新的连接地址
    emits('emit-connection-list')
  })
}
</script>

<style scoped>

</style>