<template>
  <!-- 嵌入 public/shell/index.html -->
  <div class="iframe-container">
    <iframe src="/Shell/index.html"></iframe>
  </div>
</template>

<script>
export default{
  name: "shell",
}

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
  height: 90vh;
  /* 高度占视口的 80% */
  border: none;
  /* 去掉默认边框 */
  border-radius: 8px;
  /* 与父容器一致的圆角 */
}

:deep(.el-textarea__inner) {
  padding-top: 15px;
  /* 增加顶部内边距，将文字往下移 */
  padding-bottom: 10px;
  /* 保持上下内边距平衡 */
  line-height: 1.5;
  /* 设置合适的行高 */
}


</style>