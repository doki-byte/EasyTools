import {
    Suitcase,
    Link,
    Connection,
    Edit,
    SetUp,
    WindPower,
    DataAnalysis,
    Management,
    MagicStick,
    Promotion,
    Sugar,
    ElementPlus,
    Mouse
} from '@element-plus/icons-vue';

// 图标映射
export const iconMap = {
    Suitcase,
    Link,
    Connection,
    Edit,
    SetUp,
    WindPower,
    DataAnalysis,
    Management,
    MagicStick,
    Promotion,
    Sugar,
    ElementPlus,
    Mouse
};

// 默认一级菜单配置
export const defaultMenu = [
    { name: 'tool', icon: 'Suitcase', title: '工具仓库', defaultOrder: 0, visible: true },
    { name: 'website', icon: 'Link', title: '网址导航', defaultOrder: 1, visible: true },
    { name: 'infoDeal', icon: 'Edit', title: '信息处理', defaultOrder: 2, visible: true },
    { name: 'connect', icon: 'SetUp', title: '简连助手', defaultOrder: 3, visible: true },
    { name: 'cyberchef', icon: 'WindPower', title: '编码解码', defaultOrder: 4, visible: true },
    { name: 'randomInfo', icon: 'DataAnalysis', title: '随机生成', defaultOrder: 5, visible: true },
    { name: 'notes', icon: 'Management', title: '备忘笔记', defaultOrder: 6, visible: true },
    { name: 'proxy', icon: 'ElementPlus', title: '便携代理', defaultOrder: 7, visible: true },
    { name: 'fuzz', icon: 'MagicStick', title: '随心Fuzz', defaultOrder: 8, visible: false },
    { name: 'restmate', icon: 'Mouse', title: '便携发包', defaultOrder: 9, visible: true },
    { name: 'assistive', icon: 'Connection', title: '辅助工具', defaultOrder: 10, visible: true },
    { name: 'about', icon: 'Promotion', title: '关于软件', defaultOrder: 11, visible: true },
];

// 模块内部标签页配置
export const moduleTabsConfig = {
    // 辅助工具模块的标签页
    assistive: [
        { name: 'google-syntax', title: 'Google语法', defaultOrder: 0, visible: true },
        { name: 'password-query', title: '默认密码查询', defaultOrder: 1, visible: true },
        { name: 'shell-syntax', title: '反弹Shell', defaultOrder: 2, visible: true },
        { name: 'process-query', title: '杀软进程查询', defaultOrder: 3, visible: true },
        { name: 'ip-ban-deal', title: '蓝队大批量封禁IP处置', defaultOrder: 4, visible: true },
        { name: 'fscan-deal', title: 'Fscan结果处理', defaultOrder: 5, visible: true },
        { name: 'map-query', title: '地图测试', defaultOrder: 6, visible: true },
    ],
    // 信息处理模块的标签页
    infoDeal: [
        { name: 'unwxapp', title: 'WX小程序反编译', defaultOrder: 0, visible: true },
        { name: 'jwt_crack', title: 'JWT密钥破解', defaultOrder: 1, visible: true },
        { name: 'oss-list', title: 'OSS存储桶遍历', defaultOrder: 2, visible: true },
        { name: 'ip-query', title: 'IP归属地查询', defaultOrder: 3, visible: true },
    ],
    // 简练助手模块的标签页
    connect: [
        { name: 'ssh', title: 'SSH', defaultOrder: 0, visible: true },
        { name: 'ftp', title: 'FTP', defaultOrder: 1, visible: true },
        { name: 'redis', title: 'Redis', defaultOrder: 2, visible: true },
    ],

};

// 加载菜单顺序（包含一级和模块标签页）
export const loadMenuOrder = async () => {
    try {
        const savedOrder = localStorage.getItem('menuOrder');
        const savedTabsOrder = localStorage.getItem('moduleTabsOrder');

        if (savedOrder) {
            const mainOrder = JSON.parse(savedOrder);
            const tabsOrder = savedTabsOrder ? JSON.parse(savedTabsOrder) : {};

            return {
                main: mainOrder,
                tabs: tabsOrder
            };
        }

        // 未保存过则返回默认顺序
        const mainOrder = defaultMenu.map(item => ({
            name: item.name,
            order: item.defaultOrder,
            visible: item.visible
        }));

        const tabsOrder = {};
        Object.keys(moduleTabsConfig).forEach(moduleName => {
            tabsOrder[moduleName] = moduleTabsConfig[moduleName].map(item => ({
                name: item.name,
                order: item.defaultOrder,
                visible: item.visible
            }));
        });

        return { main: mainOrder, tabs: tabsOrder };
    } catch (error) {
        console.error('加载菜单顺序失败:', error);

        const mainOrder = defaultMenu.map(item => ({
            name: item.name,
            order: item.defaultOrder,
            visible: item.visible
        }));

        const tabsOrder = {};
        Object.keys(moduleTabsConfig).forEach(moduleName => {
            tabsOrder[moduleName] = moduleTabsConfig[moduleName].map(item => ({
                name: item.name,
                order: item.defaultOrder,
                visible: item.visible
            }));
        });

        return { main: mainOrder, tabs: tabsOrder };
    }
};

// 保存菜单顺序（包含一级和模块标签页）
export const saveMenuOrder = async (mainOrder, tabsOrder = null) => {
    try {
        localStorage.setItem('menuOrder', JSON.stringify(mainOrder));
        if (tabsOrder) {
            localStorage.setItem('moduleTabsOrder', JSON.stringify(tabsOrder));
        }
        return true;
    } catch (error) {
        console.error('保存菜单顺序失败:', error);
        return false;
    }
};