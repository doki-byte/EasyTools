<template>
  <el-container class="app-container"  direction="vertical">
    <!-- 标签栏 -->
    <el-tabs v-model="activeTab" class="et-module-tabs" :style="tabsStyle">
      <el-tab-pane label="git目录遍历提取" name="git-dumper" />
      <el-tab-pane label="index目录遍历提取" name="index-dumper" />
      <el-tab-pane label="svn目录遍历提取" name="svn-dumper" />
      <el-tab-pane label="Ds_Store目录遍历提取" name="dsStore-dumper" />
<!--      <el-tab-pane label="hg目录遍历提取" name="hg-dumper" />-->
    </el-tabs>

    <!-- 内容区域 -->
    <el-main>
      <keep-alive>
        <component :is="currentComponent" />
      </keep-alive>
    </el-main>
  </el-container>
</template>

<script>
import { defineAsyncComponent } from 'vue'

const GitDumper = defineAsyncComponent(() => import('@/views/infodeal/fileDumper/gitDumper.vue'));
const SvnDumper = defineAsyncComponent(() => import('@/views/infodeal/fileDumper/svnDumper.vue'));
const DsStoreDumper = defineAsyncComponent(() => import('@/views/infodeal/fileDumper/dsStoreDumper.vue'));
const HgDumper = defineAsyncComponent(() => import('@/views/infodeal/fileDumper/hgDumper.vue'));
const IndexDumper = defineAsyncComponent(() => import('@/views/infodeal/fileDumper/indexDumper.vue'));

export default {
  name: "FileDumper",
  components: { IndexDumper, HgDumper, DsStoreDumper, SvnDumper, GitDumper },
  data: function () {
    return {
      activeTab: "git-dumper",
      
      // 字体颜色配置
      fontColor: '#ffffff',
      fontWeightRatio: 100,
      shadowRatio: 100,
      STORAGE_KEY: 'easytools:theme:background'
    }
  },
  computed: {
    currentComponent() {
      const map = {
        'git-dumper': 'GitDumper',
        'index-dumper': 'IndexDumper',
        'svn-dumper': 'SvnDumper',
        'dsStore-dumper': 'DsStoreDumper',
        'hg-dumper': 'HgDumper'
      }
      return map[this.activeTab] || null
    },
    
    // 计算标签页样式
    tabsStyle() {
      // 计算加粗程度：默认 bold (700)，100% 对应 700，0% 对应 400，200% 对应 900
      const fontWeight = 400 + (this.fontWeightRatio / 100) * 500
      // 计算阴影程度：默认 0px 2px 2px rgba(0, 0, 0, 0.1)，根据比例调整透明度
      const shadowOpacity = (this.shadowRatio / 100) * 0.1
      return {
        '--tabs-font-color': this.fontColor,
        '--tabs-font-weight': fontWeight,
        '--tabs-shadow': `0px 2px 2px rgba(0, 0, 0, ${shadowOpacity})`
      }
    }
  },
  mounted() {
    // 加载字体颜色配置
    this.loadFontColor();
    
    // 监听主题配置变化
    window.addEventListener('theme-background-changed', this.handleThemeChange);
  },
  beforeUnmount() {
    window.removeEventListener('theme-background-changed', this.handleThemeChange);
  },
  methods: {
    // 加载字体颜色配置
    loadFontColor() {
      try {
        const saved = localStorage.getItem(this.STORAGE_KEY);
        if (saved) {
          const config = JSON.parse(saved);
          if (config.fontColor) {
            this.fontColor = config.fontColor;
          }
          if (config.fontWeightRatio !== undefined) {
            this.fontWeightRatio = config.fontWeightRatio;
          }
          if (config.shadowRatio !== undefined) {
            this.shadowRatio = config.shadowRatio;
          }
        }
      } catch (e) {
        console.error('加载字体颜色配置失败:', e);
      }
    },
    // 监听主题配置变化
    handleThemeChange(event) {
      if (event.detail && event.detail !== null) {
        if (event.detail.fontColor) {
          this.fontColor = event.detail.fontColor;
        }
        if (event.detail.fontWeightRatio !== undefined) {
          this.fontWeightRatio = event.detail.fontWeightRatio;
        }
        if (event.detail.shadowRatio !== undefined) {
          this.shadowRatio = event.detail.shadowRatio;
        }
      } else if (event.detail === null) {
        // 如果配置被清除，恢复默认值
        this.fontColor = '#ffffff';
        this.fontWeightRatio = 100;
        this.shadowRatio = 100;
      } else {
        // 如果没有配置，加载本地存储的配置
        this.loadFontColor();
      }
    }
  }
}
</script>

<style scoped>
/* 保持原有样式不变 */
.app-container {
  min-height: 96vh;
  display: flex;
  flex-direction: column;
  background-color: #f8f9fb;
  padding: 0 10px;
  box-sizing: border-box;
}



:deep(.el-main) {
  --el-main-padding: 0px;
}
:deep(h2) {
  display: block;
  font-size: 1.5em;
  margin-block-start: 0.83em;
  margin-block-end: 0.83em;
  margin-inline-start: 0;
  margin-inline-end: 0;
  font-weight: bold;
  unicode-bidi: isolate;
}
</style>