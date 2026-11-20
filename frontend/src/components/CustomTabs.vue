<template>
  <div class="custom-tabs-container">
    <slot></slot>
    <div class="filter-container" :style="{ left: typeof filterLeftPosition === 'number' ? `${filterLeftPosition}px` : filterLeftPosition }">
      <slot name="filter"></slot>
    </div>
    <div class="custom-tabs-titlebar">
      <slot name="ctrl"></slot>
    </div>
  </div>
</template>

<script>
export default {
  name: 'CustomTabs',
  props: {
    filterLeftPosition: {
      type: [Number, String],
      default: 120
    }
  }
}
</script>

<style scoped>
.custom-tabs-container {
  position: relative;
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

/* 确保 slot 中的直接子元素能够填充容器 */
.custom-tabs-container > :deep(> *) {
  flex: 1;
  min-height: 0;
  height: 100%;
}

.filter-container {
  position: absolute;
  top: 4px;
  z-index: 10;
  flex: none !important;
  height: auto !important;
}

.custom-tabs-titlebar {
  position: absolute;
  right: 5px;
  top: 4px;
  z-index: 10;
  flex: none !important;
  height: auto !important;
}
</style>

