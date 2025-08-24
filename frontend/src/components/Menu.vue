<template>
  <div class="menu">
    <!-- 版本更新通知 -->
    <transition name="slide-fade">
      <div v-if="showUpdate" class="update-notice">
        <div class="notice-content">
          <div class="version-info">
            <span class="badge">New</span>
            <h3>发现新版本 {{ latestVersion }}!</h3>
          </div>
          <div class="action-buttons">
            <button @click="openBrowerToDownload" class="download-btn">立即下载</button>
            <button @click="closeNotice" class="close-btn">稍后提醒</button>
          </div>
        </div>
      </div>
    </transition>
    <div class="list">
      <div class="item" v-for="(item, index) in list" :key="index" @click="toPage(item.name)"
        :class="{ active: routeName === item.name }">
        <!-- 动态渲染图标 -->
        <el-icon>
          <component :is="item.icon" />
        </el-icon>
        <span>{{ item.title }}</span>
      </div>
    </div>
    <div class="logout-btn" @click="checkUpdate()">
      <span style="display: flex; align-items: center; white-space: nowrap;">
        <el-icon>
          <Promotion />
        </el-icon>
        <span :style="{ color: latestVersion ? '#0062bc' : 'inherit', marginLeft: '5px' }">
          {{ latestVersion ? `New 最新版${latestVersion}` : 'v1.8.4' }}
        </span>
      </span>
    </div>
    <div class="logout-btn" @click="updateUser()">
      <!-- 使用图标 -->
      <span><el-icon>
          <UserFilled />
        </el-icon>&nbsp;修改密码</span>
    </div>
    <div class="logout-btn" @click="logout()">
      <!-- 使用图标 -->
      <span><el-icon>
          <WarningFilled />
        </el-icon>&nbsp;退出登录</span>
    </div>
  </div>

  <!-- 修改密码对话框 -->
  <el-dialog
      v-model="showPasswordDialog"
      title="修改密码"
      width="400px"
      :close-on-click-modal="false"
  >
    <el-form
        ref="pwdFormRef"
        :rules="pwdRules"
        :model="pwdForm"
        label-width="80px"
        status-icon
    >
      <el-form-item label="账号" prop="username">
        <el-input
            v-model="pwdForm.username"
            placeholder="自动获取当前账号"
        />
      </el-form-item>
      <el-form-item label="原密码" prop="oldPassword">
        <el-input
            v-model="pwdForm.oldPassword"
            type="password"
            show-password
            placeholder="请输入原密码"
        />
      </el-form-item>
      <el-form-item label="新密码" prop="newPassword">
        <el-input
            v-model="pwdForm.newPassword"
            type="password"
            show-password
            placeholder="6-20位字母/数字组合"
        />
      </el-form-item>
      <el-form-item label="确认密码" prop="confirmPassword">
        <el-input
            v-model="pwdForm.confirmPassword"
            type="password"
            show-password
            placeholder="请再次输入新密码"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleCancelPassword">取 消</el-button>
        <el-button
            type="primary"
            @click="handleChangePassword"
            :loading="isUpdatingPassword"
        >
          确 定
        </el-button>
      </div>
    </template>
  </el-dialog>

</template>

<script setup>
import {ref, onMounted, markRaw, reactive} from 'vue';
import { useRouter } from 'vue-router';
import {ElMessageBox, ElNotification} from 'element-plus';
import { removeToken } from '@/utils/token';
import {
  Link,
  Connection,
  DataAnalysis,
  UserFilled,
  WarningFilled,
  Promotion,
  WindPower,
  Suitcase,
  Edit,
  Sugar,
  SetUp,
  Management, MoreFilled, MagicStick
} from '@element-plus/icons-vue'; // 引入所有图标组件
import {UpdateUser} from "../../wailsjs/go/controller/User";
import { GetLatestRelease } from '../../wailsjs/go/controller/Update'

// 当前路由名称，默认设置为 'tool'
const routeName = ref('tool');
const router = useRouter();

// 修改密码相关状态
const pwdFormRef = ref(null);
const showPasswordDialog = ref(false);
const isUpdatingPassword = ref(false);
const pwdForm = reactive({
  username: '',
  oldPassword: "",
  newPassword: "",
  confirmPassword: "",
});

// 初始化检查
onMounted(() => {
  AutoCheckUpdate()
})

// 打开修改密码弹窗
const updateUser = () => {
  // 自动填充当前登录用户（根据你的实际登录信息调整）
  pwdForm.username = localStorage.getItem('EasyTools-Token') || 'EasyTools';
  showPasswordDialog.value = true;
};

const pwdRules = {
  oldPassword: [{ required: true, message: "请输入原密码", trigger: "blur" }],
  newPassword: [
    { required: true, message: "请输入新密码", trigger: "blur" },
    { min: 6, message: "密码长度至少6位", trigger: "blur" }
  ],
  confirmPassword: [
    { validator: (rule, value, callback) => {
        if (value !== pwdForm.newPassword) {
          callback(new Error("两次输入密码不一致"));
        } else {
          callback();
        }
      }, trigger: "blur" }
  ]
};

// 响应式菜单项
const list = ref([
  {
    name: 'tool',
    icon: markRaw(Suitcase),
    title: '工具仓库',
  },
  {
    name: 'website',
    icon: markRaw(Link), // 使用图标组件
    title: '网址导航',
  },
  {
    name: 'infoSearch',
    icon: markRaw(Connection),
    title: '信息查询',
  },
  {
    name: 'infoDeal',
    icon: markRaw(Edit),
    title: '信息处理',
  },
  {
    name: 'connect',
    icon: markRaw(SetUp),
    title: '简连助手',
  },
  {
    name: 'cyberchef',
    icon: markRaw(WindPower),
    title: '编码解码',
  },
  {
    name: 'randomInfo',
    icon: markRaw(DataAnalysis),
    title: '随机生成',
  },
  {
    name: 'notes',
    icon: markRaw(Management),
    title: '备忘笔记',
  },
  {
    name: 'proxy',
    icon: markRaw(MagicStick),
    title: '便携代理',
  },
  {
    name: 'fuzz',
    icon: markRaw(MagicStick),
    title: '随心FUZZ',
  },
  {
    name: 'about',
    icon: markRaw(Promotion),
    title: '关于软件',
  }
]);



// 跳转到指定页面
function toPage(name) {
  if (routeName.value === name) {
    return;
  }
  routeName.value = name; // 更新路由名称以激活对应菜单样式
  router.push({ name }).catch(() => { });
}

// 修改密码处理逻辑
const handleChangePassword = async () => {
  try {
    // 执行表单验证
    await pwdFormRef.value.validate();

    // 显示加载状态
    isUpdatingPassword.value = true;

    // 调用后端接口
    const error = await UpdateUser(
        pwdForm.username,
        {
          UserName: pwdForm.username,
          PassWord: pwdForm.newPassword,
          OldPassword: pwdForm.oldPassword // 添加原密码字段
        }
    );

    if (error) throw new Error(error);

    ElNotification({
      title: "修改成功",
      message: "请使用新密码重新登录",
      type: "success",
      duration: 2000
    });

    // 关闭弹窗并跳转
    showPasswordDialog.value = false;
    removeToken();
    await router.replace({ name: "login" });

  } catch (err) {
    ElNotification({
      title: "修改失败",
      message: "主人忘记您设置的原密码了吗 o(╥﹏╥)o",
      type: "error",
      duration: 2500
    });
  } finally {
    isUpdatingPassword.value = false;
  }
}

// 修改密码取消处理
const handleCancelPassword = () => {
  showPasswordDialog.value = false

  // 延迟执行确保动画完成
  setTimeout(() => {
    // 重置表单验证状态
    pwdFormRef.value?.resetFields()

    // 清空敏感字段（保留用户名）
    pwdForm.oldPassword = ""
    pwdForm.newPassword = ""
    pwdForm.confirmPassword = ""
  }, 300)
}

// 退出登录
function logout() {
  ElMessageBox.confirm('是否退出登录？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(() => {
      removeToken();
      router.push({ name: 'login' }).catch(() => { });
    })
    .catch(() => {
      console.log('取消退出');
    });
}

// 版本更新状态
const showUpdate = ref(false)
const latestVersion = ref('')
const releaseUrl = ref('')

// 获取版本信息
const checkUpdate = async () => {
  try {
    const result = await GetLatestRelease()
    if (result.hasUpdate) {
      showUpdate.value = true
      latestVersion.value = result.latestRelease.tag_name
      releaseUrl.value = result.latestRelease.html_url
    } else{
      ElNotification({
        title: "温馨提示",
        message: "当前已经是最新版啦(*^▽^*)",
        type: "success",
        duration: 2500
      });
    }
  } catch (error) {
    console.error('版本检查失败:', error)
  }
}

// 获取版本信息
const AutoCheckUpdate = async () => {
  try {
    const result = await GetLatestRelease()
    if (result.hasUpdate) {
      showUpdate.value = true
      latestVersion.value = result.latestRelease.tag_name
      releaseUrl.value = result.latestRelease.html_url
      console.log(result)
    }
  } catch (error) {
    console.error('版本检查失败:', error)
  }
}

// 关闭通知
const closeNotice = () => {
  showUpdate.value = false
}

const openBrowerToDownload = () => {
  window.open(releaseUrl.value, "_blank");
  console.log(releaseUrl)
}

// 页面加载时设置默认路由
onMounted(() => {
  if (!routeName.value) {
    routeName.value = 'tool';
    router.push({ name: 'tool' }).catch(() => { });
  }
});

</script>


<style scoped lang="scss">
/* 样式保持不变 */
.update-notice {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 9999;
  background: #fff;
  border-radius: 8px;
  border-left: 4px solid #4ee439;
  animation: slideIn 0.3s ease-out;
  max-width: 320px;

  .notice-content {
    padding: 16px;
    position: relative;
  }

  .version-info {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;

    .badge {
      background: #409eff;
      color: white;
      padding: 4px 8px;
      border-radius: 4px;
      font-size: 12px;
      font-weight: bold;
    }

    h3 {
      margin: 0;
      font-size: 16px;
    }
  }

  .action-buttons {
    display: flex;
    gap: 12px;
    margin-top: 8px;

    .download-btn {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 8px 16px;
      background: #409eff;
      color: white;
      border-radius: 6px;
      text-decoration: none;
      transition: all 0.2s;
      border: none;

      &:hover {
        background: #66b1ff;
        transform: translateY(-1px);
      }

      .svg-icon {
        width: 16px;
        height: 16px;
      }
    }

    .close-btn {
      padding: 8px 16px;
      background: #f0f2f5;
      border: none;
      border-radius: 6px;
      color: #606266;
      cursor: pointer;
      transition: all 0.2s;

      &:hover {
        background: #e6e8eb;
      }
    }
  }
}

/* 入场动画 */
.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.3s cubic-bezier(1, 0.5, 0.8, 1);
}

.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translateX(20px);
  opacity: 0;
}

@keyframes slideIn {
  from {
    transform: translateX(100%);
  }
  to {
    transform: translateX(0);
  }
}

.menu {
  height: 100vh;
  // padding: 0 5px;
  background: #f0f5f6;
  color: #26384e;
  display: flex;
  flex-direction: column;
  align-items: center;

  .list {
    display: flex;
    flex-direction: column;
    align-items: center;
    flex-grow: 1;
    /* 使导航菜单占据剩余空间 */
  }

  .item,
  .logout-btn {
    padding: 0 10px;
    margin-top: 15px;
    cursor: pointer;
    width: 100px;
    height: 40px;
    line-height: 40px;
    text-align: center;
    font-size: 14px;
    border-radius: 10px;
    display: flex;
    justify-content: center;
    /* 水平居中 */
    align-items: center;
    /* 垂直居中 */

    /* 调整图标和文字间距 */
    span {
      margin-left: 5px;
      /* 文字与图标间距 */
    }

    el-icon {
      display: inline-block;
      margin-right: 10px;
      /* 图标与文字的间距 */
    }

    &:hover {
      background: #e4e8ec;
      color: #4f5a6b;
    }

    &.active {
      background: #fc3d48;
      color: #ffffff;
    }
  }

  .logout-btn {
    margin-top: -10px !important;
    bottom: 10px;
    /* 固定到底部 */
    color: #888;
    /* 灰色文字 */
    font-size: 12px;
    /* 字体略小 */
    display: flex;
    justify-content: center;
    align-items: center;

    span {
      display: flex;
      align-items: center;
    }

    el-icon {
      color: #888;
      /* 图标灰色 */
      margin-right: 5px;
      /* 图标与文字间隔 */
    }

    &:hover {
      color: #555;
      /* 悬停时颜色稍微加深 */
    }
  }
}
</style>
