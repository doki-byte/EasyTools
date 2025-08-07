<template>
  <div class="iframe-container">
    <iframe :src="iframeSrc"></iframe>
  </div>
</template>

<script>
import { defineComponent } from 'vue';
import { ElMessageBox } from 'element-plus';
import { BrowserOpenURL } from '../../wailsjs/runtime';

export default defineComponent({
  name: 'CyberChefView',
  data() {
    return {
      // 本地 iframe 路径
      iframeSrc: 'http://127.0.0.1:52867/CyberChef/index.html',
    };
  },
  mounted() {
    // 只有第一次打开才弹窗：可以根据需要改成持久化判断（如 localStorage）
    ElMessageBox.confirm(
        '温馨提示，该功能还存在 bug，在您切换页面之后，无法保存已加载的编码选项，是否在浏览器中打开网页呢？',
        '提示',
        {
          confirmButtonText: '确认',
          cancelButtonText: '取消',
          type: 'warning',
          // 点击遮罩或按 ESC 不触发 confirm
          distinguishCancelAndClose: true,
        }
    )
        .then(() => {
          // 用户点击“确认”，在默认浏览器中打开
          const fullUrl = this.iframeSrc;
          BrowserOpenURL(fullUrl);
          // 本地 iframe 继续加载，不需要额外操作
        })
        .catch((action) => {
          // 用户点击“取消”或关闭，不做额外操作
          // 本地 iframe 会如常加载
        });
  },
});
</script>

<style scoped>
/* iframe 容器样式 */
.iframe-container {
    /* margin: 0 auto; */
    max-width: 100%;
    /* 控制 iframe 的最大宽度 */
    overflow: hidden;
    border-radius: 8px;
    /* 圆角处理 */
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    /* 增加阴影效果 */
}

/* iframe 样式 */
.iframe-container iframe {
    width: 100%;
    /* 保持 iframe 宽度填充父容器 */
    height: 98vh;
    /* 高度占视口的 98% */
    border: none;
    /* 去掉默认边框 */
    border-radius: 8px;
    /* 与父容器一致的圆角 */
}
</style>
