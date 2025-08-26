<template>
  <div class="menu-manager">
    <h2>菜单顺序管理</h2>
    <p>拖拽调整菜单项顺序，点击保存应用更改（可切换显示/隐藏）</p>

    <div class="menu-list">
      <draggable
          v-model="menuItems"
          item-key="name"
          @end="onDragEnd"
          handle=".drag-handle"
      >
        <template #item="{ element, index }">
          <div class="menu-item">
            <el-icon class="drag-handle"><MoreFilled /></el-icon>
            <el-icon class="menu-icon">
              <component :is="element.icon" />
            </el-icon>
            <span class="menu-title">{{ element.title }}</span>
            <el-tag type="info" size="small">{{ element.name }}</el-tag>

            <!-- 可见性开关 -->
            <div class="visibility-switch">
              <el-switch
                  v-model="element.visible"
                  active-text="显示"
                  inactive-text="隐藏"
                  size="small"
              />
            </div>
          </div>
        </template>
      </draggable>
    </div>

    <div class="action-buttons">
      <el-button type="primary" @click="saveOrder">保存顺序</el-button>
      <el-button @click="resetOrder">重置为默认顺序</el-button>
      <el-button @click="goBack">返回</el-button>
    </div>
  </div>
</template>

<script setup>
import {markRaw, onMounted, ref} from 'vue';
import {useRouter} from 'vue-router';
import {ElMessage} from 'element-plus';
import draggable from 'vuedraggable';
import {MoreFilled} from '@element-plus/icons-vue';
import {defaultMenu, iconMap, loadMenuOrder, saveMenuOrder} from '@/utils/menuConfig';

const router = useRouter();
const menuItems = ref([]);

// 加载菜单数据（包含 visible）
const loadMenuData = async () => {
  const savedOrder = await loadMenuOrder();
  const os = localStorage.getItem('os');

  // 创建完整菜单项列表（把 savedOrder 的 order/visible 合并进来）
  let fullMenu = defaultMenu.map(item => {
    const savedItem = savedOrder.find(i => i.name === item.name);
    return {
      ...item,
      order: savedItem ? savedItem.order : item.defaultOrder,
      visible: savedItem ? (typeof savedItem.visible === 'boolean' ? savedItem.visible : item.visible) : item.visible,
      icon: markRaw(iconMap[item.icon])
    };
  });

  // 按当前顺序排序
  menuItems.value = fullMenu.sort((a, b) => a.order - b.order);
};

// 拖拽结束事件
const onDragEnd = () => {
  // 更新每个项目的顺序值
  menuItems.value.forEach((item, index) => {
    item.order = index;
  });
};

// 保存顺序（包含 visible）
const saveOrder = async () => {
  const orderToSave = menuItems.value.map(item => ({
    name: item.name,
    order: item.order,
    visible: typeof item.visible === 'boolean' ? item.visible : true
  }));

  const success = await saveMenuOrder(orderToSave);
  if (success) {
    ElMessage.success('菜单顺序与可见性已保存');

    // 派发事件，通知其它组件（比如 menu.vue）重新加载菜单
    window.dispatchEvent(new CustomEvent('menu-order-updated', { detail: { time: Date.now() } }));
  } else {
    ElMessage.error('保存失败，请重试');
  }
};

// 重置为默认顺序（包含默认 visible）
const resetOrder = () => {
  menuItems.value = menuItems.value
      .map(item => {
        const def = defaultMenu.find(i => i.name === item.name);
        return {
          ...item,
          order: def.defaultOrder,
          visible: def.visible
        };
      })
      .sort((a, b) => a.order - b.order);

  // 重置后也触发一次保存/通知（可选：这里仅触发事件，不自动写入 localStorage；如果想同时保存可以调用 saveMenuOrder）
  window.dispatchEvent(new CustomEvent('menu-order-updated', { detail: { time: Date.now(), reset: true } }));
};

// 返回上一页
const goBack = () => {
  router.go(-1);
};

onMounted(() => {
  loadMenuData();
});
</script>

<style scoped>
.menu-manager {
  padding: 5px;
  max-width: 700px;
  margin: 0 auto;
}

.menu-list {
  margin: 20px 0;
  border: 1px solid #e6e8eb;
  border-radius: 8px;
  overflow: hidden;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #fff;
  border-bottom: 1px solid #e6e8eb;
  cursor: move;
}

.menu-item:last-child {
  border-bottom: none;
}

.drag-handle {
  margin-right: 12px;
  color: #909399;
  cursor: move;
}

.menu-icon {
  margin-right: 12px;
  color: #606266;
}

.menu-title {
  flex: 1;
  font-size: 14px;
}

.visibility-switch {
  margin-left: 12px;
  display: flex;
  align-items: center;
}

/* 按钮区域 */
.action-buttons {
  display: flex;
  gap: 12px;
  justify-content: center;
  margin-top: 12px;
}
</style>
