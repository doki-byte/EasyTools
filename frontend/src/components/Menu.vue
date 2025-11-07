<template>
  <div class="menu">
    <div class="list">
      <div
          class="item"
          v-for="(item, index) in visibleMenu"
          :key="item.name"
          @click="toPage(item.name)"
          :class="{ active: routeName === item.name }"
      >
        <el-icon>
          <component :is="item.icon" />
        </el-icon>
        <span>{{ item.title }}</span>
      </div>
    </div>

    <!-- 检查更新按钮 -->
    <div class="logout-btn" @click="handleCheckUpdate">
      <span style="display: flex; align-items: center; white-space: nowrap;">
        <el-icon>
          <Promotion />
        </el-icon>
        <el-badge :is-dot="hasUpdate" style="margin-left: 5px;">
          <span :style="{ color: hasUpdate ? '#0062bc' : 'inherit' }">
            {{ hasUpdate ? `New ${latestVersion}` : 'v1.9.4' }}
          </span>
        </el-badge>
      </span>
    </div>

    <div class="logout-btn" @click="goToSystemManager()">
      <span><el-icon><Menu /></el-icon>&nbsp;系统管理</span>
    </div>
    <div class="logout-btn" @click="updateUser()">
      <span><el-icon><UserFilled /></el-icon>&nbsp;修改密码</span>
    </div>
    <div class="logout-btn" @click="logout()">
      <span><el-icon><WarningFilled /></el-icon>&nbsp;退出登录</span>
    </div>

    <!-- 更新对话框 -->
    <el-dialog
        v-model="updateDialogVisible"
        title="检查更新"
        width="500px"
        :close-on-click-modal="false"
        :show-close="false"
        class="glass-update-dialog"
        :before-close="handleDialogClose"
    >
      <div class="glass-dialog-content">
        <div class="dialog-header">
          <div class="update-icon">
            <el-icon size="24"><Promotion /></el-icon>
          </div>
          <h3 class="dialog-title">发现新版本</h3>
        </div>

        <div class="update-content">
          <div class="version-section">
            <span class="section-label">最新版本:</span>
            <el-tag class="version-tag" effect="dark">{{ latestVersion }}</el-tag>
          </div>

          <div class="description-section">
            <span class="section-label">更新内容:</span>
            <div class="description-text">
              <p
                  v-for="(line, index) in formattedDescription"
                  :key="index"
                  class="description-line"
              >
                {{ line }}
              </p>
            </div>
          </div>

          <!-- 下载进度 -->
          <div v-if="isDownloading" class="progress-section">
            <span class="section-label">下载进度:</span>
            <div class="progress-content">
              <el-progress
                  :percentage="downloadProgress"
                  :stroke-width="8"
                  :show-text="true"
                  status="success"
              />
              <div class="progress-text">{{ downloadStatus }}</div>
            </div>
          </div>

          <div class="url-section">
            <span class="section-label">下载地址:</span>
            <div class="url-text">
              <el-tag
                  v-if="releaseUrl"
                  class="url-tag"
                  effect="plain"
                  @click="handleUrlClick"
              >
                {{ releaseUrl }}
              </el-tag>
              <span v-else class="no-url">暂无下载地址</span>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="glass-dialog-footer">
          <el-button
              class="footer-btn skip-btn"
              size="small"
              @click="handleSkipToday"
              :disabled="isDownloading"
          >
            今日不再提示
          </el-button>
          <el-button
              class="footer-btn cancel-btn"
              size="small"
              @click="updateDialogVisible = false"
              :disabled="isDownloading"
          >
            取消
          </el-button>
          <el-button
              @click="handleDownloadUpdate"
              :disabled="isDownloading"
              :loading="isDownloading"
              type="success"
          >
            <el-icon style="margin-right: 4px;"><Download /></el-icon>
            {{ isDownloading ? `下载中...` : '自动下载更新' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 毛玻璃效果修改密码对话框 -->
    <el-dialog
        v-model="showPasswordDialog"
        title="修改密码"
        width="400px"
        :close-on-click-modal="false"
        class="glass-update-dialog"
    >
      <div class="glass-dialog-content">
        <div class="dialog-header">
          <div class="update-icon">
            <el-icon size="24"><UserFilled /></el-icon>
          </div>
          <h3 class="dialog-title">修改密码</h3>
        </div>

        <el-form
            ref="pwdFormRef"
            :rules="pwdRules"
            :model="pwdForm"
            label-width="80px"
            status-icon
            class="password-form"
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
      </div>

      <template #footer>
        <div class="glass-dialog-footer">
          <el-button
              class="footer-btn cancel-btn"
              @click="handleCancelPassword"
          >
            取 消
          </el-button>
          <el-button
              class="footer-btn confirm-btn"
              type="primary"
              @click="handleChangePassword"
              :loading="isUpdatingPassword"
          >
            确 定
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import {computed, markRaw, nextTick, onMounted, onUnmounted, reactive, ref, watch} from 'vue';
import { useRouter } from 'vue-router';
import { ElMessageBox, ElNotification } from 'element-plus';
import { removeToken } from '@/utils/token';
import { defaultMenu, iconMap, loadMenuOrder } from '@/utils/menuConfig';
import { Menu, Promotion, UserFilled, WarningFilled, Download } from '@element-plus/icons-vue';
import { UpdateUser } from "../../wailsjs/go/controller/User";
import { GetLatestRelease, DownloadAndUpdate, RestartApplication } from '../../wailsjs/go/system/Update';
import { BrowserOpenURL } from "../../wailsjs/runtime";
import { EventsOn, EventsOff } from '../../wailsjs/runtime';

// 当前路由名称
const routeName = ref('tool');
const router = useRouter();

// 版本更新相关状态
const updateDialogVisible = ref(false);
const hasUpdate = ref(false);
const latestVersion = ref('');
const releaseUrl = ref('');
const releaseDescription = ref('');
const loading = ref(false);

// 下载状态
const isDownloading = ref(false);
const downloadProgress = ref(0);
const downloadStatus = ref('');

// 响应式菜单项
const menuList = ref([]);

onMounted(() => {
  if (!routeName.value) {
    routeName.value = 'tool'
    router.push({ name: 'tool' }).catch(() => {})
  }
  loadMenu()
  autoCheckUpdate()

  // 注册下载进度监听器
  EventsOn('downloadProgress', (progress) => {
    downloadProgress.value = Math.round(progress)
    // console.log('下载进度:', progress, '四舍五入后:', downloadProgress.value)

    // 如果正在下载且对话框未打开，自动打开对话框显示进度
    if (progress > 0 && progress < 100 && !updateDialogVisible.value) {
      updateDialogVisible.value = true
    }
  })

  window.addEventListener('menu-order-updated', handleMenuUpdated)
})

onUnmounted(() => {
  window.removeEventListener('menu-order-updated', handleMenuUpdated);
  // 组件卸载时移除事件监听器
  EventsOff('downloadProgress');
});

// 计算属性
const sortedMenuList = computed(() => {
  return menuList.value.slice().sort((a, b) => a.order - b.order);
});

const visibleMenu = computed(() => {
  const list = Array.isArray(sortedMenuList.value) ? sortedMenuList.value : [];
  return list.filter(i => i && (i.visible === undefined ? true : !!i.visible));
});

const formattedDescription = computed(() => {
  if (!releaseDescription.value) return [];
  return releaseDescription.value
      .replace(/\n(?!\n?$)/g, '\n')
      .split('\n')
      .filter(line => line.trim() !== '');
});

// 加载菜单配置
const loadMenu = async () => {
  const savedData = await loadMenuOrder();
  const savedOrder = savedData.main || []; // 取 main 数组，如果不存在则用空数组

  menuList.value = defaultMenu.map(item => {
    const savedItem = savedOrder.find(i => i.name === item.name);
    return {
      ...item,
      order: savedItem ? savedItem.order : item.defaultOrder,
      visible: savedItem ? (typeof savedItem.visible === 'boolean' ? savedItem.visible : item.visible) : item.visible,
      icon: markRaw(iconMap[item.icon])
    };
  });
};

const goToSystemManager = () => {
  router.push({ name: 'systemManage' });
};

// 修改密码相关
const pwdFormRef = ref(null);
const showPasswordDialog = ref(false);
const isUpdatingPassword = ref(false);
const pwdForm = reactive({
  username: '',
  oldPassword: "",
  newPassword: "",
  confirmPassword: "",
});

// 事件处理
const handleMenuUpdated = (ev) => {
  loadMenu();
};

// 生命周期
onMounted(() => {
  if (!routeName.value) {
    routeName.value = 'tool';
    router.push({ name: 'tool' }).catch(() => { });
  }
  loadMenu();
  autoCheckUpdate();

  window.addEventListener('menu-order-updated', handleMenuUpdated);
});

onUnmounted(() => {
  window.removeEventListener('menu-order-updated', handleMenuUpdated);
  // 确保移除事件监听器
  EventsOff('downloadProgress')
});

// 页面导航
function toPage(name) {
  if (routeName.value === name) {
    return;
  }
  routeName.value = name;
  router.push({ name }).catch(() => { });
}

// 密码修改逻辑
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

const updateUser = () => {
  pwdForm.username = localStorage.getItem('EasyTools-Token') || 'EasyTools';
  showPasswordDialog.value = true;
};

const handleChangePassword = async () => {
  try {
    await pwdFormRef.value.validate();
    isUpdatingPassword.value = true;
    const error = await UpdateUser(
        pwdForm.username,
        {
          UserName: pwdForm.username,
          PassWord: pwdForm.newPassword,
          OldPassword: pwdForm.oldPassword
        }
    );

    if (error) throw new Error(error);

    ElNotification({
      title: "修改成功",
      message: "请使用新密码重新登录",
      type: "success",
      duration: 2000
    });

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

const handleCancelPassword = () => {
  showPasswordDialog.value = false;
  setTimeout(() => {
    pwdFormRef.value?.resetFields();
    pwdForm.oldPassword = "";
    pwdForm.newPassword = "";
    pwdForm.confirmPassword = "";
  }, 300);
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

// 版本更新逻辑
const checkUpdate = async () => {
  loading.value = true;
  try {
    const result = await GetLatestRelease();
    if (result.hasUpdate) {
      hasUpdate.value = true;
      latestVersion.value = result.latestRelease.tag_name;
      releaseUrl.value = result.latestRelease.html_url;
      releaseDescription.value = result.latestRelease.body || '暂无更新说明';
      return true;
    } else {
      ElNotification({
        title: "温馨提示",
        message: "当前已经是最新版啦(*^▽^*)",
        type: "success",
        duration: 2500
      });
      return false;
    }
  } catch (error) {
    console.error('版本检查失败:', error);
    ElNotification({
      title: "检查失败",
      message: "无法连接到更新服务器",
      type: "error",
      duration: 2500
    });
    return false;
  } finally {
    loading.value = false;
  }
}

// 自动检查更新
const autoCheckUpdate = async () => {
  const skipTodayKey = 'EasyTools-SkipUpdateToday';
  const skipDate = localStorage.getItem(skipTodayKey);
  const today = new Date().toDateString();

  try {
    const result = await GetLatestRelease();
    if (result.hasUpdate) {
      hasUpdate.value = true;
      latestVersion.value = result.latestRelease.tag_name;
      releaseUrl.value = result.latestRelease.html_url;
      releaseDescription.value = result.latestRelease.body || '暂无更新说明';

      if (skipDate === today) {
          updateDialogVisible.value = false;
          return;
      }
      updateDialogVisible.value = true;
    }
  } catch (error) {
    console.error('自动版本检查失败:', error);
  }
}

// 处理下载更新
const handleDownloadUpdate = async () => {
  isDownloading.value = true
  downloadProgress.value = 0
  downloadStatus.value = '开始下载...'

  // 强制更新一次，确保状态显示
  await nextTick()

  try {
    const result = await DownloadAndUpdate()

    if (result.error) {
      downloadStatus.value = '下载失败'
      ElNotification.error('下载更新失败: ' + result.msg)
    } else {
      downloadStatus.value = '下载完成'
      ElNotification.success('更新成功: ' + result.msg)

      // 根据系统提示重启
      if (result.msg.includes('重启') || result.msg.includes('重启应用')) {
        ElMessageBox.confirm('更新成功，是否立即重启应用？', '提示', {
          confirmButtonText: '重启',
          cancelButtonText: '稍后',
          type: 'success'
        }).then(async () => {
          await RestartApplication()
        }).catch(() => {
          updateDialogVisible.value = false
        })
      } else {
        updateDialogVisible.value = false
      }
    }
  } catch (error) {
    console.error('下载更新失败:', error)
    downloadStatus.value = '下载出错'
    ElNotification.error('下载更新失败: ' + error.message)
  } finally {
    isDownloading.value = false
    // 不清空状态，让用户看到最终状态
    // downloadProgress.value = 0
    // downloadStatus.value = ''
  }
}

// 修改进度监听器，同时更新状态
EventsOn('downloadProgress', (progress) => {
  downloadProgress.value = Math.round(progress)
  // 同时更新状态文本
  downloadStatus.value = `下载中... ${downloadProgress.value}%`
  console.log('下载进度:', progress, '四舍五入后:', downloadProgress.value)

  // 如果正在下载且对话框未打开，自动打开对话框显示进度
  if (progress > 0 && progress < 100 && !updateDialogVisible.value) {
    updateDialogVisible.value = true
  }
})

const handleCheckUpdate = async () => {
  const hasNewVersion = await checkUpdate();
  if (hasNewVersion) {
    updateDialogVisible.value = true;
  }
}

const handleSkipToday = () => {
  const today = new Date().toDateString();
  localStorage.setItem('EasyTools-SkipUpdateToday', today);
  updateDialogVisible.value = false;
  ElNotification({
    title: "提示",
    message: "今日将不再提示更新",
    type: "info",
    duration: 2000
  });
}

const handleDialogClose = (done) => {
  updateDialogVisible.value = false;
  done();
}

const handleUrlClick = () => {
  if (releaseUrl.value) {
    BrowserOpenURL(releaseUrl.value);
  }
}
</script>

<style scoped lang="scss">
.menu {
  height: 96vh;
  padding: 0 10px;
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
  }

  .item,
  .logout-btn {
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
    align-items: center;

    span {
      margin-left: 5px;
    }

    el-icon {
      display: inline-block;
      margin-right: 10px;
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
    color: #888;
    font-size: 12px;
    display: flex;
    justify-content: center;
    align-items: center;

    span {
      display: flex;
      align-items: center;
    }

    el-icon {
      color: #888;
      margin-right: 5px;
    }

    &:hover {
      color: #555;
    }
  }
}

/* 毛玻璃对话框样式 */
.glass-dialog-content {
  background: rgba(255, 255, 255, 0.85);
  backdrop-filter: blur(20px);
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  padding: 0;
}

.dialog-header {
  display: flex;
  align-items: center;
  padding: 20px 20px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 12px 12px 0 0;
}

.update-icon {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
}

.update-icon .el-icon {
  color: white;
  font-size: 20px;
}

.dialog-title {
  margin: 0;
  color: white;
  font-size: 16px;
  font-weight: 600;
}

.update-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px;
}

.version-section,
.description-section,
.url-section {
  display: flex;
  align-items: flex-start;
}

.section-label {
  min-width: 80px;
  font-weight: 500;
  color: #333;
  margin-right: 12px;
}

.version-tag {
  background: linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%);
  color: white;
  border: none;
  font-weight: 600;
}

.description-section {
  flex-direction: column;

  .description-text {
    margin-left: 0;
    margin-top: 8px;
  }

  .description-line {
    white-space: pre;
    margin: 4px 0;
    line-height: 1.4;
    color: #666;
    padding-left: 8px;
    border-left: 2px solid #667eea;
  }
}

.url-section {
  flex-direction: column;

  .url-text {
    margin-left: 0;
    margin-top: 8px;
  }

  .url-tag {
    word-break: break-all;
    cursor: pointer;
    transition: all 0.3s ease;
    border: 1px solid #667eea;
    color: #667eea;

    &:hover {
      background: #667eea;
      color: white;
      transform: translateY(-1px);
      box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
    }
  }

  .no-url {
    color: #999;
    font-style: italic;
  }
}

.glass-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 16px 20px;
  background: rgba(248, 248, 248, 0.8);
  border-radius: 0 0 12px 12px;
}

.footer-btn {
  border-radius: 6px;
  min-width: 80px;
  transition: all 0.3s ease;
}

.skip-btn {
  border: 1px solid #d9d9d9;
  color: #666;

  &:hover {
    border-color: #4096ff;
    color: #4096ff;
  }
}

.cancel-btn {
  border: 1px solid #d9d9d9;
  color: #666;

  &:hover {
    border-color: #4096ff;
    color: #4096ff;
  }
}

.download-btn {
  background: linear-gradient(135deg, #ff6b6b 0%, #ee5a24 100%);
  border: none;
  color: white;

  &:hover:not(:disabled) {
    background: linear-gradient(135deg, #ff8787 0%, #f76707 100%);
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(255, 107, 107, 0.3);
  }
}

.confirm-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  color: white;

  &:hover {
    background: linear-gradient(135deg, #7c8fee 0%, #8a68b5 100%);
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
  }
}

.password-form {
  padding: 20px;
}

.password-form :deep(.el-form-item__label) {
  color: #333;
  font-weight: 500;
}
</style>

<style>
/* 全局样式，美化 Element Plus Dialog */
.glass-update-dialog .el-dialog {
  background: transparent;
  box-shadow: none;
  border-radius: 16px;
}

.glass-update-dialog .el-dialog__header {
  display: none;
}

.glass-update-dialog .el-dialog__body {
  padding: 0 !important;
  background: transparent;
}

.glass-update-dialog .el-dialog__footer {
  padding: 0 !important;
  background: transparent;
}

.glass-update-dialog .el-overlay {
  backdrop-filter: blur(4px);
}

.glass-update-dialog .el-overlay-dialog {
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 新增进度条样式 */
.progress-section {
  display: flex;
  flex-direction: column;

  .progress-content {
    margin-left: 0;
    margin-top: 8px;

    .progress-text {
      margin-top: 8px;
      font-size: 12px;
      color: #666;
      text-align: center;
    }
  }
}

/* 调整下载按钮样式 */
.download-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>