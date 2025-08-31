import { createApp } from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';
import { createPinia } from 'pinia';

import '@arco-design/web-vue/dist/arco.css';
import ArcoVue from '@arco-design/web-vue';
import ArcoVueIcon from '@arco-design/web-vue/es/icon';

// Element Plus
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import zhCn from 'element-plus/es/locale/lang/zh-cn';
import * as ElementPlusIconsVue from '@element-plus/icons-vue';

// dayjs 中文 locale（Element Plus 的日期组件依赖 dayjs）
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
dayjs.locale('zh-cn');

const app = createApp(App);

const pinia = createPinia();

// 注册 Element Plus 图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component);
}

app.use(router);
app.use(store);
app.use(pinia);

app.use(ArcoVue);
app.use(ArcoVueIcon);

app.use(ElementPlus, { locale: zhCn });


app.mount('#app');
