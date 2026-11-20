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
    Mouse, Compass,
    Menu,
    Search,
    Lock
} from '@element-plus/icons-vue';

// 图标映射
export const iconMap = {
    Suitcase,
    Link,
    Compass,
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
    Mouse,
    Menu,
    Search,
    Lock
};

// 默认一级菜单配置
export const defaultMenu = [
    { name: 'tool', icon: 'Suitcase', title: '工具仓库', defaultOrder: 0, visible: true },
    { name: 'website', icon: 'Link', title: '网址导航', defaultOrder: 1, visible: false },
    { name: 'cyberSabre', icon: 'Compass', title: '渗透测试', defaultOrder: 2, visible: true },
    { name: 'infoCollection', icon: 'Search', title: '信息收集', defaultOrder: 3, visible: true },
    { name: 'infoDeal', icon: 'Edit', title: '信息处理', defaultOrder: 4, visible: true },
    { name: 'cloudSecurity', icon: 'Lock', title: '云上安全', defaultOrder: 5, visible: true },
    { name: 'connect', icon: 'SetUp', title: '简连助手', defaultOrder: 6, visible: true },
    { name: 'ctfTools', icon: 'WindPower', title: 'CTF工具', defaultOrder: 7, visible: true },
    { name: 'notes', icon: 'Management', title: '备忘笔记', defaultOrder: 8, visible: true },
    { name: 'cliExecutor', icon: 'MagicStick', title: 'CLI执行', defaultOrder: 9, visible: true },
    { name: 'restmate', icon: 'Mouse', title: '便携发包', defaultOrder: 10, visible: true },
    { name: 'assistive', icon: 'Connection', title: '辅助工具', defaultOrder: 11, visible: true },
    { name: 'about', icon: 'Promotion', title: '关于软件', defaultOrder: 12, visible: true },
    { name: 'bypassTools', icon: 'Sugar', title: '免杀生成', defaultOrder: 13, visible: true },
    { name: 'systemManage', icon: 'Menu', title: '系统管理', defaultOrder: 14, visible: false },
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
        { name: 'MapCheck-query', title: '地图测试', defaultOrder: 6, visible: true },
        { name: 'randomInfo', title: '随机生成', defaultOrder: 7, visible: true },
        { name: 'portable-proxy', title: '便携代理', defaultOrder: 8, visible: true },
    ],
    // 信息处理模块的标签页
    infoDeal: [
        { name: 'unwxapp', title: 'WX小程序反编译', defaultOrder: 0, visible: true },
        { name: 'jwt_crack', title: 'JWT密钥破解', defaultOrder: 1, visible: true },
        { name: 'file-dumper', title: '文件泄露提取', defaultOrder: 2, visible: true },
        { name: 'ip-query', title: 'IP归属地查询', defaultOrder: 3, visible: true },
    ],
    // 简练助手模块的标签页
    connect: [
        { name: 'ssh', title: 'SSH', defaultOrder: 0, visible: true },
        { name: 'ftp', title: 'FTP', defaultOrder: 1, visible: true },
        { name: 'nosql', title: 'NoSQL数据库', defaultOrder: 2, visible: true },
        { name: 'dbmanager', title: '数据库管理', defaultOrder: 3, visible: true },
    ],
    // Ctf工具模块的标签页
    ctfTools: [
        { name: 'cyberchef', title: '编码解码', defaultOrder: 0, visible: true },
        { name: 'pcap', title: 'pcap分析器', defaultOrder: 1, visible: true },
    ],
    // Bypass模块的标签页
    bypassTools: [
        { name: 'bypassAv', title: 'C2免杀', defaultOrder: 0, visible: true },
        { name: 'skyshadow', title: '白加黑挖掘', defaultOrder: 1, visible: true },
    ],
    // 渗透测试模块的标签页
    cyberSabre: [
        { name: 'hostscan', title: '主机扫描', defaultOrder: 0, visible: true },
        { name: 'dirsearch', title: '目录扫描', defaultOrder: 1, visible: true },
        { name: 'fingerprint', title: '指纹识别', defaultOrder: 2, visible: true },
        { name: 'webscan', title: '漏洞扫描', defaultOrder: 3, visible: true },
        { name: 'pocManagement', title: 'POC管理', defaultOrder: 4, visible: true },
        { name: 'browser-scan', title: 'JSFinder', defaultOrder: 5, visible: true },
        { name: 'password-crack', title: '密码爆破', defaultOrder: 6, visible: true },
        { name: 'druid', title: 'Druid利用', defaultOrder: 7, visible: true },
    ],
    // 信息收集模块的标签页
    infoCollection: [
        { name: 'space-mapping', title: '空间测绘', defaultOrder: 0, visible: true },
        { name: 'domain-collection', title: '域名收集', defaultOrder: 1, visible: true },
    ],
    // 云上安全模块的标签页
    cloudSecurity: [
        { name: 'ossBucketPanel', title: 'OSS桶扫描', defaultOrder: 0, visible: true },
        { name: 'objectStoragePanel', title: '云储存工具', defaultOrder: 1, visible: true },
        { name: 'cloudServerPanel', title: '云服务器工具', defaultOrder: 2, visible: true },
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

// 获取第一个激活的菜单项名称
export const getFirstActiveMenuName = async () => {
    try {
        const savedData = await loadMenuOrder();
        const savedOrder = savedData.main || [];
        
        // 合并默认菜单和保存的配置
        const menuItems = defaultMenu.map(item => {
            const savedItem = savedOrder.find(i => i.name === item.name);
            return {
                ...item,
                order: savedItem ? savedItem.order : item.defaultOrder,
                visible: savedItem 
                    ? (typeof savedItem.visible === 'boolean' ? savedItem.visible : item.visible)
                    : item.visible
            };
        });
        
        // 过滤出可见的菜单项，并按顺序排序
        const visibleMenus = menuItems
            .filter(item => item.visible !== false)
            .sort((a, b) => a.order - b.order);
        
        // 返回第一个可见菜单项的名称，如果没有则返回默认的 'tool'
        return visibleMenus.length > 0 ? visibleMenus[0].name : 'tool';
    } catch (error) {
        console.error('获取第一个激活菜单失败:', error);
        // 出错时返回默认值
        return 'tool';
    }
};