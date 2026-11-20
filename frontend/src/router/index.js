import { createRouter, createWebHistory } from 'vue-router';
import NProgress from 'nprogress';
import 'nprogress/nprogress.css';
import { getFirstActiveMenuName, loadMenuOrder, defaultMenu } from '@/utils/menuConfig';
import { isCybersecurityAgreementValid } from '@/utils/cybersecurityAgreement';

const checkAgreementStatus = () => isCybersecurityAgreementValid();

/** 视图懒加载：降低首屏 JS 体积与低端机解析耗时（原先路由静态导入会把几乎所有页面打进入口 chunk） */
const lazyView = (loader) => () =>
  loader().catch((err) => {
    console.error('路由视图加载失败:', err);
    throw err;
  });

// 路由配置
const routes = [
  {
    path: '/',
    name: 'tool',
    component: lazyView(() => import('@/views/ToolsView.vue')),
  },
  {
    path: '/website',
    name: 'website',
    component: lazyView(() => import('@/views/SiteView.vue')),
  },
  {
    path: '/assistive',
    name: 'assistive',
    component: lazyView(() => import('@/views/AssistiveView.vue')),
  },
  {
    path: '/infoDeal',
    name: 'infoDeal',
    component: lazyView(() => import('@/views/InfoDealView.vue')),
  },
  {
    path: '/cloudSecurity/scanner',
    redirect: '/cloudSecurity',
  },
  {
    path: '/cloudSecurity/cos',
    redirect: '/cloudSecurity',
  },
  {
    path: '/cloudSecurity/server',
    redirect: '/cloudSecurity',
  },
  {
    path: '/cloudSecurity',
    name: 'cloudSecurity',
    component: lazyView(() => import('@/views/CloudSecurityLayoutView.vue')),
  },
  {
    path: '/infoCollection',
    name: 'infoCollection',
    component: lazyView(() => import('@/views/InfoCollectionView.vue')),
  },
  {
    path: '/cyberSabre',
    name: 'cyberSabre',
    component: lazyView(() => import('@/views/CyberSabreView.vue')),
  },
  {
    path: '/pocManagement',
    name: 'pocManagement',
    component: lazyView(() => import('@/views/cyberSabre/PocManagementView.vue')),
  },
  {
    path: '/cyberSabre/pocManagement',
    name: 'cyberSabrePocManagement',
    component: lazyView(() => import('@/views/cyberSabre/PocManagementView.vue')),
  },
  {
    path: '/Connect',
    name: 'connect',
    component: lazyView(() => import('@/views/ConnectView.vue')),
  },
  {
    path: '/ctfTools',
    name: 'ctfTools',
    component: lazyView(() => import('@/views/CtfToolsView.vue')),
  },
  {
    path: '/randomInfo',
    name: 'randomInfo',
    component: lazyView(() => import('@/views/assistive/RandomInfo.vue')),
  },
  {
    path: '/bypassAv',
    // 与菜单配置中的 name 保持一致，避免路由 name 不匹配
    name: 'bypassTools',
    component: lazyView(() => import('@/views/BypassView.vue')),
  },
  {
    path: '/notes',
    name: 'notes',
    component: lazyView(() => import('@/views/NoteView.vue')),
  },
  {
    path: '/proxy',
    redirect(to) {
      return { path: '/assistive', query: { ...to.query, tab: 'portable-proxy' } };
    },
  },
  {
    path: '/cliExecutor',
    name: 'cliExecutor',
    component: lazyView(() => import('@/views/CLIExecutorView.vue')),
  },
  {
    path: '/restmate',
    name: 'restmate',
    component: lazyView(() => import('@/views/Restmate.vue')),
  },
  {
    path: '/About',
    name: 'about',
    component: lazyView(() => import('@/views/About.vue')),
  },
  {
    path: '/login',
    name: 'login',
    component: lazyView(() => import('@/views/LoginView.vue')),
  },
  {
    path: '/systemManage',
    name: 'systemManage',
    component: lazyView(() => import('@/views/SystemView.vue')),
  },
  {
    path: '/games',
    name: 'games',
    component: lazyView(() => import('@/views/GamesView.vue')),
  },
  // 默认重定向路由 - 将在路由守卫中动态处理
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

// 检查路由是否被隐藏
const isRouteHidden = async (routeName) => {
  // 登录页面和系统管理页面不检查
  if (routeName === 'login' || routeName === 'systemManage' || routeName === 'games') {
    return false;
  }

  try {
    const savedData = await loadMenuOrder();
    const savedOrder = savedData.main || [];

    // 查找该路由对应的菜单项
    const menuItem = defaultMenu.find((item) => item.name === routeName);
    if (!menuItem) {
      // 如果不在菜单配置中，允许访问（可能是子路由）
      return false;
    }

    // 查找保存的配置
    const savedItem = savedOrder.find((i) => i.name === routeName);
    const isVisible = savedItem
      ? typeof savedItem.visible === 'boolean'
        ? savedItem.visible
        : menuItem.visible
      : menuItem.visible;

    return !isVisible;
  } catch (error) {
    console.error('检查路由可见性失败:', error);
    return false; // 出错时允许访问
  }
};

// 路由守卫
router.beforeEach(async (to, from, next) => {
  const agreed = checkAgreementStatus();
  // 如果没有同意协议且目标路由不是 login，则重定向到协议页面
  if (!agreed && to.name !== 'login') {
    next({ name: 'login' }); // 重定向到协议页面
    return;
  }

  // 如果已同意协议，检查路由是否被隐藏
  if (agreed && to.name && to.name !== 'login') {
    const hidden = await isRouteHidden(to.name);
    if (hidden) {
      // 如果路由被隐藏，重定向到第一个激活的路由
      const firstActiveMenu = await getFirstActiveMenuName();
      next({ name: firstActiveMenu, replace: true });
      return;
    }
  }

  // 处理根路径重定向（当访问 '/' 时，如果 tool 被隐藏，重定向到第一个激活的路由）
  if (to.path === '/' && to.name === 'tool') {
    const hidden = await isRouteHidden('tool');
    if (hidden) {
      const firstActiveMenu = await getFirstActiveMenuName();
      next({ name: firstActiveMenu, replace: true });
      return;
    }
  }

  NProgress.start(); // 开始进度条
  next(); // 继续路由
});

// 结束进度条
router.afterEach(() => {
  NProgress.done(); // 结束进度条
});

export default router;
