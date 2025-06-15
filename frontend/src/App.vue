<template>
  <div class="app">
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

    <!-- 原有布局 -->
    <div class="appMenu" v-if="routeName !== 'login'">
      <Menu :routeName="routeName" />
    </div>
    <div class="appMain">
      <!-- 添加缓存功能 -->
      <router-view v-slot="{ Component }">
        <keep-alive :include="cachedPages">
          <component :is="Component" :key="routeName" />
        </keep-alive>
      </router-view>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import Menu from './components/Menu.vue'
import { GetLatestRelease } from '../wailsjs/go/controller/Update'
import { BrowserOpenURL } from "../wailsjs/runtime";

const route = useRoute()
const routeName = computed(() => route.name)

// 缓存页面列表 - 这些名称必须与页面组件的name属性匹配
const cachedPages = ref(['ToolsView', 'SiteView', 'InfoSearchView', 'InfoDealView', 'ConnectView', 'CyberChefView', 'RandomInfoView', 'BypassAvView']);

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
  BrowserOpenURL(releaseUrl.value)
  console.log(releaseUrl)
}

// 初始化检查
onMounted(() => {
  checkUpdate()
})
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

/* 原有样式保持不动 */
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

html,
body {
  margin: 0;
  padding: 0;
}

body {
  width: 100vw;
  height: 100vh;
  background: #ffffff;
}

.app {
  display: flex;
  align-items: center;

  .appMenu {
    flex-shrink: 0;
  }

  .appMain {
    height: 100vh;
    overflow: auto;
    flex: 1;
  }
}
</style>