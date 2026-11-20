import { defaultMenu, moduleTabsConfig } from '@/utils/menuConfig'

/** 一级菜单 name → 路由 path（与 router/index.js 一致） */
const MENU_PATH = {
  tool: '/',
  website: '/website',
  assistive: '/assistive',
  infoDeal: '/infoDeal',
  cloudSecurity: '/cloudSecurity',
  infoCollection: '/infoCollection',
  cyberSabre: '/cyberSabre',
  connect: '/Connect',
  ctfTools: '/ctfTools',
  notes: '/notes',
  cliExecutor: '/cliExecutor',
  restmate: '/restmate',
  about: '/About',
  bypassTools: '/bypassAv',
  systemManage: '/systemManage',
  games: '/games',
}

let cached = null

/**
 * @returns {{ id: string, title: string, path: string, keywords: string[], moduleKey: string|null, subTab: string|null }[]}
 */
export function getGlobalSearchIndex() {
  if (cached) return cached
  const items = []

  for (const m of defaultMenu) {
    const path = MENU_PATH[m.name]
    if (!path) continue

    items.push({
      id: `m-${m.name}`,
      title: m.title,
      path,
      keywords: [m.title, m.name, path],
      moduleKey: null,
      subTab: null,
    })

    const tabs = moduleTabsConfig[m.name]
    if (tabs && tabs.length) {
      for (const t of tabs) {
        items.push({
          id: `m-${m.name}-${t.name}`,
          title: `${m.title} · ${t.title}`,
          path,
          keywords: [m.title, t.title, t.name, m.name],
          moduleKey: m.name,
          subTab: t.name,
        })
      }
    }
  }

  cached = items
  return items
}

export function clearSearchIndexCache() {
  cached = null
}

/**
 * 模糊分数：子序列匹配 + 连续片段奖励，低端机可承受
 */
export function fuzzyScore(query, text) {
  const q = (query || '').trim().toLowerCase()
  const s = (text || '').toLowerCase()
  if (!q) return 0
  if (s.includes(q)) return 1000 + q.length * 2

  let qi = 0
  let gaps = 0
  let streak = 0
  let score = 0
  for (let i = 0; i < s.length && qi < q.length; i++) {
    if (s[i] === q[qi]) {
      score += 10 + streak * 3
      streak++
      qi++
    } else {
      streak = 0
      gaps++
    }
  }
  if (qi < q.length) return 0
  return score - gaps * 2
}

export function searchGlobalItems(query, limit = 80) {
  const idx = getGlobalSearchIndex()
  const q = (query || '').trim()
  if (!q) return idx.slice(0, limit)

  const scored = []
  for (const item of idx) {
    const blob = [item.title, ...item.keywords].join(' ')
    const sc = fuzzyScore(q, blob)
    if (sc > 0) scored.push({ item, sc })
  }
  scored.sort((a, b) => b.sc - a.sc)
  return scored.slice(0, limit).map((x) => x.item)
}
