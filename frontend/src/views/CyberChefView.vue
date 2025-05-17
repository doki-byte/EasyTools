<template>
    <!-- 嵌入 public/CyberChef/index.html -->
    <div class="iframe-container">
        <iframe src="/CyberChef/index.html"></iframe>
    </div>
</template>


<script>
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
