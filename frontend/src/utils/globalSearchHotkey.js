import { normalizeAccelerator, accelFromEvent } from '@/utils/hotkeyUtils'

/** 存于 theme 配置 JSON（easytools:theme:background）中的字段名 */
export const GLOBAL_SEARCH_THEME_FIELD = 'globalSearchHotkey'

/** 默认与历史行为一致：Ctrl+K（macOS 上用户可改成 Cmd+K） */
export const DEFAULT_GLOBAL_SEARCH_HOTKEY = 'Ctrl+K'

/**
 * 从完整主题配置中取全局搜索快捷键（归一化后字符串）
 */
export function getGlobalSearchHotkeyFromConfig(config) {
  if (!config || typeof config[GLOBAL_SEARCH_THEME_FIELD] !== 'string') {
    return DEFAULT_GLOBAL_SEARCH_HOTKEY
  }
  const raw = config[GLOBAL_SEARCH_THEME_FIELD].trim()
  if (!raw) return DEFAULT_GLOBAL_SEARCH_HOTKEY
  const n = normalizeAccelerator(raw)
  return n || DEFAULT_GLOBAL_SEARCH_HOTKEY
}

/**
 * 从 localStorage 当前主题条目读取快捷键
 */
export function getStoredGlobalSearchHotkey() {
  try {
    const raw = localStorage.getItem('easytools:theme:background')
    if (!raw) return DEFAULT_GLOBAL_SEARCH_HOTKEY
    const config = JSON.parse(raw)
    return getGlobalSearchHotkeyFromConfig(config)
  } catch {
    return DEFAULT_GLOBAL_SEARCH_HOTKEY
  }
}

export function validateGlobalSearchHotkey(accRaw) {
  const n = normalizeAccelerator((accRaw || '').trim())
  if (!n) {
    return { ok: false, message: '请按下快捷键，或点击下方恢复默认', normalized: DEFAULT_GLOBAL_SEARCH_HOTKEY }
  }
  const parts = n.split('+').map((p) => p.trim())
  const hasMod = parts.some((p) => ['Ctrl', 'Alt', 'Shift', 'Cmd'].includes(p))
  if (!hasMod || parts.length < 2) {
    return {
      ok: false,
      message: '需在组合键中包含 Ctrl / Cmd / Alt / Shift，避免与普通输入冲突',
      normalized: DEFAULT_GLOBAL_SEARCH_HOTKEY,
    }
  }
  return { ok: true, normalized: n }
}

/**
 * 判断是否应打开/关闭全局搜索面板
 */
export function eventMatchesGlobalSearchHotkey(event, accelStr) {
  const acc = accelStr || getStoredGlobalSearchHotkey()
  const fromEv = accelFromEvent(event)
  if (!fromEv) return false
  return normalizeAccelerator(fromEv) === normalizeAccelerator(acc)
}
