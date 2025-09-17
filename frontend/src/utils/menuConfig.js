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
    ElementPlus, Mouse
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

// 默认菜单配置（增加 visible）
export const defaultMenu = [
    { name: 'tool', icon: 'Suitcase', title: '工具仓库', defaultOrder: 0, visible: true },
    { name: 'website', icon: 'Link', title: '网址导航', defaultOrder: 1, visible: true },
    { name: 'infoSearch', icon: 'Connection', title: '信息查询', defaultOrder: 2, visible: true },
    { name: 'infoDeal', icon: 'Edit', title: '信息处理', defaultOrder: 3, visible: true },
    { name: 'connect', icon: 'SetUp', title: '简连助手', defaultOrder: 4, visible: true },
    { name: 'cyberchef', icon: 'WindPower', title: '编码解码', defaultOrder: 5, visible: true },
    { name: 'randomInfo', icon: 'DataAnalysis', title: '随机生成', defaultOrder: 6, visible: true },
    { name: 'notes', icon: 'Management', title: '备忘笔记', defaultOrder: 7, visible: true },
    { name: 'proxy', icon: 'ElementPlus', title: '便携代理', defaultOrder: 8, visible: true },
    { name: 'fuzz', icon: 'MagicStick', title: '随心Fuzz', defaultOrder: 9, visible: false },
    { name: 'restmate', icon: 'Mouse', title: '便携发包', defaultOrder: 10, visible: true },
    { name: 'about', icon: 'Promotion', title: '关于软件', defaultOrder: 11, visible: true },
];

// 加载菜单顺序
export const loadMenuOrder = async () => {
    try {
        const savedOrder = localStorage.getItem('menuOrder');
        if (savedOrder) {
            return JSON.parse(savedOrder);
        }

        // 未保存过则返回默认顺序（包含 visible）
        return defaultMenu.map(item => ({ name: item.name, order: item.defaultOrder, visible: item.visible }));
    } catch (error) {
        console.error('加载菜单顺序失败:', error);
        return defaultMenu.map(item => ({ name: item.name, order: item.defaultOrder, visible: item.visible }));
    }
};

// 保存菜单顺序（并包含 visible）
// order 参数应为 [{ name, order, visible }]
export const saveMenuOrder = async (order) => {
    try {
        localStorage.setItem('menuOrder', JSON.stringify(order));
        // 若有后端持久化可在此处同时保存到后端
        return true;
    } catch (error) {
        console.error('保存菜单顺序失败:', error);
        return false;
    }
};
