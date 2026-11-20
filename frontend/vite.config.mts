import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import path from 'path';

export default defineConfig({
  base: './',
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
    dedupe: ['vue'],
  },
  build: {
    chunkSizeWarningLimit: 1200,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return;
          if (id.includes('monaco-editor')) return 'monaco';
          if (id.includes('@arco-design')) return 'arco';
          if (id.includes('element-plus')) return 'element-plus';
          if (id.includes('@element-plus/icons-vue')) return 'element-icons';
          if (id.includes('vue-router')) return 'vue-router';
          if (id.includes('pinia')) return 'pinia';
        },
      },
    },
  },
  optimizeDeps: {
    include: ['vue', 'element-plus', '@arco-design/web-vue'],
  },
  css: {
    postcss: path.resolve(__dirname, 'postcss.config.cjs')
  },
  define: {
    global: 'globalThis'
  },
  server: {
    proxy: {
      // 百度地图 API 代理，解决浏览器环境下的 CORS 问题，仅用于前端调试发包
      '/baidu-map-api': {
        target: 'https://api.map.baidu.com',
        changeOrigin: true,
        rewrite: (p: string) => p.replace(/^\/baidu-map-api/, ''),
      },
    },
  },
});
