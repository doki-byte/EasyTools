import { createApp } from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';
import { createPinia } from 'pinia'
import '@arco-design/web-vue/dist/arco.css';
import ArcoVue from '@arco-design/web-vue';
import ArcoVueIcon from '@arco-design/web-vue/es/icon';

import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';

import * as ElementPlusIconsVue from '@element-plus/icons-vue';

const app = createApp(App);

const pinia = createPinia()


for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component);
}

app.use(router);
app.use(store);
app.use(ElementPlus);
app.use(pinia)
app.use(ArcoVue);
app.use(ArcoVueIcon);
Notification._context = app._context;


app.mount('#app');
