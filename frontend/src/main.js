import { createApp, nextTick } from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';
import { createPinia } from 'pinia';
import axios from 'axios';
import { GetLocalServiceToken } from '../wailsjs/go/controller/Server';

import './styles/tokens.css';
import './styles/module-top-tabs.css';
import './styles/tabs-nav.css';
import './styles/mac-platform.css';

// Element Plus
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import zhCn from 'element-plus/es/locale/lang/zh-cn';
import * as ElementPlusIconsVue from '@element-plus/icons-vue';
import { ElMessage } from 'element-plus';

// dayjs 中文 locale（Element Plus 的日期组件依赖 dayjs）
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';

// 导入主题应用函数
import { applyThemeToApp } from '@/utils/themeBackground';
import './styles/db-tooltips.css';
import CustomTabs from '@/components/CustomTabs.vue';
import TableColumnFilterHeader from '@/components/TableColumnFilterHeader.vue';

/** Monaco：Worker + 动态导入 core，拆分 chunk；需在挂载前完成 loader.config */
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker';
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker';
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker';
import { loader, install as VueMonacoEditorPlugin } from '@guolao/vue-monaco-editor';

globalThis.MonacoEnvironment = {
  getWorker(_workerId, label) {
    if (label === 'json') {
      return new jsonWorker();
    }
    if (label === 'css' || label === 'scss' || label === 'less') {
      return new cssWorker();
    }
    if (label === 'html' || label === 'handlebars' || label === 'razor') {
      return new htmlWorker();
    }
    if (label === 'typescript' || label === 'javascript') {
      return new tsWorker();
    }
    return new editorWorker();
  },
};

const monacoModule = await import('monaco-editor');
const monacoApi = monacoModule.default ?? monacoModule;
loader.config({ monaco: monacoApi });

dayjs.locale('zh-cn');

// ---------------------------------------------------------------------------
// 本地服务访问令牌：Wails 协议页面（wails://wails）下浏览器把 Origin 序列化为
// "null"，本地服务防护要求此类请求携带进程级 Token（恶意网页无法获取）。
// 1) 默认 axios 全局注入请求头（axios.create 实例共享 defaults.headers.common）；
// 2) 请求拦截器兜底：每次请求前确保 Token 已就绪（避免页面首屏请求早于 Token 获取）；
// 3) 挂到 window 供 WebSocket URL 拼接使用。
// ---------------------------------------------------------------------------
let localServiceToken = '';
let localTokenPromise = null;

function getLocalToken() {
  if (localTokenPromise) return localTokenPromise;
  localTokenPromise = GetLocalServiceToken()
    .then((t) => {
      localServiceToken = t || '';
      return localServiceToken;
    })
    .catch((e) => {
      console.warn('获取本地服务令牌失败:', e);
      // 失败后允许重试（绑定可能尚未就绪）
      localTokenPromise = null;
      return '';
    });
  return localTokenPromise;
}

try {
  localServiceToken = await getLocalToken();
} catch (e) {
  console.warn('获取本地服务令牌失败:', e);
}
window.__LOCAL_SERVICE_TOKEN__ = localServiceToken || '';
if (localServiceToken) {
  axios.defaults.headers.common['X-Auth-Token'] = localServiceToken;
}

// 拦截器兜底：请求发出前确保 Token 就绪（对本地服务请求生效）
axios.interceptors.request.use(async (config) => {
  const url = config.url || '';
  if (/^https?:\/\/(127\.0\.0\.1|localhost|\[::1\])/.test(url) && !config.headers['X-Auth-Token']) {
    try {
      const token = await getLocalToken();
      if (token) config.headers['X-Auth-Token'] = token;
    } catch (e) {
      console.warn('请求前获取本地服务令牌失败:', e);
    }
  }
  return config;
});

const app = createApp(App);

const pinia = createPinia();

// 全局注册自定义组件
app.component('CustomTabs', CustomTabs);
app.component('TableColumnFilterHeader', TableColumnFilterHeader);

// 注册 Element Plus 图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component);
}

app.use(router);
app.use(store);
app.use(pinia);

app.use(ElementPlus, { locale: zhCn });
app.use(VueMonacoEditorPlugin);

// 配置全局 $message 属性，统一使用 ElMessage，并包装默认选项
app.config.globalProperties.$message = {
  success: (options) => {
    if (typeof options === 'string') {
      return ElMessage.success({
        message: options,
        duration: 3000,
        showClose: false,
        offset: 20,
        grouping: true,
      });
    }
    return ElMessage.success({
      duration: 3000,
      showClose: false,
      offset: 20,
      grouping: true,
      ...options,
    });
  },
  error: (options) => {
    if (typeof options === 'string') {
      return ElMessage.error({
        message: options,
        duration: 3000,
        showClose: false,
        offset: 20,
        grouping: true,
      });
    }
    return ElMessage.error({
      duration: 3000,
      showClose: false,
      offset: 20,
      grouping: true,
      ...options,
    });
  },
  warning: (options) => {
    if (typeof options === 'string') {
      return ElMessage.warning({
        message: options,
        duration: 3000,
        showClose: false,
        offset: 20,
        grouping: true,
      });
    }
    return ElMessage.warning({
      duration: 3000,
      showClose: false,
      offset: 20,
      grouping: true,
      ...options,
    });
  },
  info: (options) => {
    if (typeof options === 'string') {
      return ElMessage.info({
        message: options,
        duration: 3000,
        showClose: false,
        offset: 20,
        grouping: true,
      });
    }
    return ElMessage.info({
      duration: 3000,
      showClose: false,
      offset: 20,
      grouping: true,
      ...options,
    });
  },
};

// 挂载应用
app.mount('#app');

// 在应用挂载后立即应用主题，确保主题在第一次渲染时就生效
// 使用 nextTick 确保 DOM 已经渲染完成
nextTick(async () => {
  await applyThemeToApp();
});
