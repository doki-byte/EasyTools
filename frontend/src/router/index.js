// router/index.js
import { createRouter, createWebHistory } from 'vue-router';
import WebsiteView from '@/views/SiteView.vue';
import { getToken } from '@/utils/token';
import NProgress from 'nprogress';
import 'nprogress/nprogress.css';

import ToolView from '@/views/ToolsView.vue';
import RandomInfoView from '@/views/RandomInfoView.vue';
import InfoSearchView from '@/views/InfoSearchView.vue';
import InfoDealView from '@/views/InfoDealView.vue';
import CyberChefView from '@/views/CyberChefView.vue';
import ConnectView from "@/views/ConnectView.vue";
import AboutView from "@/views/About.vue";
import noteView from "@/views/NoteView.vue";
import FuzzView from "@/views/FuzzView.vue";

// 路由配置
const routes = [
  {
    path: '/',
    name: 'tool',
    component: ToolView,
  },
  {
    path: '/',
    name: 'website',
    component: WebsiteView,
  },
  {
    path: '/infoSearch',
    name: 'infoSearch',
    component: InfoSearchView,
  },
  {
    path: '/infoDeal',
    name: 'infoDeal',
    component: InfoDealView,
  },
  {
    path: '/Connect',
    name: 'connect',
    component: ConnectView,
  },
  {
    path: '/cyberchef',
    name: 'cyberchef',
    component: CyberChefView,
  },
  {
    path: '/randomInfo',
    name: 'randomInfo',
    component: RandomInfoView,
  },
  {
    path: '/notes',
    name: 'notes',
    component: noteView,
  },
  {
    path: '/proxy',
    name: 'proxy',
    component: () =>
        import('@/views/proxy/Panel.vue'),  // 确保路径正确
  },
  {
    path: '/fuzz',
    name: 'fuzz',
    component: FuzzView,
  },
  {
    path: '/About',
    name: 'about',
    component: AboutView,
  },
  {
    path: '/login',
    name: 'login',
    component: () =>
      import('@/views/LoginView.vue'),  // 确保路径正确
  },
  // 默认重定向路由
  {
    path: '/',
    redirect: { name: 'tool' }
  }
];

// 创建路由
const router = createRouter({
  history: createWebHistory(), // 使用 HTML5 history 模式
  routes,
});

// 配置 NProgress
NProgress.configure({
  easing: 'ease',
  speed: 500,
  showSpinner: false, // 可选择是否显示旋转图标
});

// 路由守卫
router.beforeEach((to, from, next) => {
  const token = getToken();
  // 如果没有 token 且目标路由不是 login，则重定向到登录页面
  if (!token && to.name !== 'login') {
    next({ name: 'login' }); // 重定向到登录页面
  } else {
    NProgress.start(); // 开始进度条
    next(); // 继续路由
  }
});

// 结束进度条
router.afterEach(() => {
  NProgress.done(); // 结束进度条
});

export default router;
