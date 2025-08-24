<template>
  <div class="app">
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
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import Menu from './components/Menu.vue'

const route = useRoute()
const routeName = computed(() => route.name)

// 缓存页面列表 - 这些名称必须与页面组件的name属性匹配
const cachedPages = ref(['ToolsView', 'SiteView', 'InfoSearchView', 'InfoDealView', 'ConnectView', 'CyberChefView', 'RandomInfoView', 'BypassAvView', 'NoteViews', 'ProxyView']);

</script>

<style scoped lang="scss">
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