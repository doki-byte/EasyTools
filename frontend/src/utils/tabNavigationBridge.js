/**
 * 全局搜索等场景：路由到达后通过 window 事件切换到模块内子标签
 * detail: { module: menuConfig 中的一级 name, tab: moduleTabsConfig 中的 name }
 */
export function tabNavigationBridge(moduleKey) {
  return {
    mounted() {
      this._onEasyToolsTab = (e) => {
        const d = e.detail
        if (!d || d.module !== moduleKey || !d.tab) return
        const exists = this.visibleTabs?.some((tab) => tab.name === d.tab)
        if (exists) this.activeTab = d.tab
      }
      window.addEventListener('easytools-set-tab', this._onEasyToolsTab)
    },
    beforeUnmount() {
      window.removeEventListener('easytools-set-tab', this._onEasyToolsTab)
    },
  }
}
