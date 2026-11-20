/**
 * localStorage 统一管理工具
 * 提供容量管理、自动清理、错误处理等功能
 */

// localStorage 容量限制（通常为 5-10MB，这里设置为 4MB 作为安全阈值）
const STORAGE_LIMIT = 4 * 1024 * 1024 // 4MB
const WARNING_THRESHOLD = 3 * 1024 * 1024 // 3MB 警告阈值
const CLEANUP_THRESHOLD = 3.5 * 1024 * 1024 // 3.5MB 开始清理

// 存储优先级：数字越大优先级越高，低优先级的数据会被优先清理
const STORAGE_PRIORITY = {
  // 高优先级：关键配置和认证信息
  'EasyTools-Token': 100,
  'Admin-Token': 100,
  'isAuthorized': 100,
  'authExpire': 100,
  'cybersecurity_agreement': 90,
  'cybersecurity_agreement_at': 90,
  'EasyTools-SkipUpdateToday': 90,
  
  // 中高优先级：重要配置
  'easytools:theme:background': 80,
  'menuOrder': 75,
  'moduleTabsOrder': 75,
  'cli-tools': 70,
  'dirScannerConfig': 70,
  'hotkey-config': 70,
  
  // 中优先级：历史记录
  'cli-execution-history': 50,
  'cli-history-column-widths': 60,
  
  // 低优先级：任务和历史数据（会被优先清理）
  'hostscanTasks': 20,
  'hostscanHistory': 20,
  'dirsearchTasks': 20,
  'dirsearchHistory': 20,
  'webscanTasks': 20,
  'fingerprintTasks': 20,
  'fingerprint-history': 20,
}

// 默认优先级
const DEFAULT_PRIORITY = 30

/**
 * 获取 localStorage 当前使用的总大小（字节）
 */
function getStorageSize() {
  let total = 0
  for (let key in localStorage) {
    if (localStorage.hasOwnProperty(key)) {
      const value = localStorage.getItem(key)
      // key 和 value 的大小，每个字符 2 字节（UTF-16）
      total += key.length * 2 + (value ? value.length * 2 : 0)
    }
  }
  return total
}

/**
 * 获取指定 key 的优先级
 */
function getPriority(key) {
  return STORAGE_PRIORITY[key] || DEFAULT_PRIORITY
}

/**
 * 清理低优先级的数据
 */
function cleanupLowPriorityData(targetSize) {
  const currentSize = getStorageSize()
  if (currentSize <= targetSize) {
    return { cleaned: 0, removedKeys: [] }
  }
  
  // 收集所有键及其信息
  const keys = []
  for (let key in localStorage) {
    if (localStorage.hasOwnProperty(key)) {
      const value = localStorage.getItem(key)
      const size = (key.length + (value ? value.length : 0)) * 2
      const priority = getPriority(key)
      keys.push({ key, size, priority })
    }
  }
  
  // 按优先级排序（低优先级在前）
  keys.sort((a, b) => a.priority - b.priority)
  
  // 清理数据
  let cleaned = 0
  const removedKeys = []
  
  for (const item of keys) {
    if (currentSize - cleaned <= targetSize) {
      break
    }
    
    // 不清理高优先级数据
    if (item.priority >= 70) {
      continue
    }
    
    try {
      localStorage.removeItem(item.key)
      cleaned += item.size
      removedKeys.push(item.key)
      console.log(`清理低优先级数据: ${item.key} (${(item.size / 1024).toFixed(2)}KB, 优先级: ${item.priority})`)
    } catch (e) {
      console.warn(`清理数据失败: ${item.key}`, e)
    }
  }
  
  return { cleaned, removedKeys }
}

/**
 * 限制数组类型的存储数据大小
 * @param {Array} data - 要存储的数组数据
 * @param {number} maxItems - 最大保留项数
 * @param {Function} sortFn - 排序函数（可选，用于确定保留哪些项）
 */
function limitArraySize(data, maxItems = 50, sortFn = null) {
  if (!Array.isArray(data) || data.length <= maxItems) {
    return data
  }
  
  // 如果有排序函数，先排序
  if (sortFn && typeof sortFn === 'function') {
    data.sort(sortFn)
  }
  
  // 保留最新的 maxItems 项（通常排序后是最后 N 项）
  return data.slice(-maxItems)
}

/**
 * 限制对象类型的存储数据大小
 * @param {Object} data - 要存储的对象数据
 * @param {number} maxItems - 最大保留项数
 * @param {Function} sortFn - 排序函数，返回排序后的键值对数组
 */
function limitObjectSize(data, maxItems = 100, sortFn = null) {
  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    return data
  }
  
  const entries = Object.entries(data)
  if (entries.length <= maxItems) {
    return data
  }
  
  // 如果有排序函数，先排序
  let sortedEntries = entries
  if (sortFn && typeof sortFn === 'function') {
    sortedEntries = entries.sort((a, b) => sortFn(a[1], b[1]))
  } else {
    // 默认按创建时间排序（如果有 createdAt 字段）
    sortedEntries = entries.sort((a, b) => {
      const timeA = a[1]?.createdAt || 0
      const timeB = b[1]?.createdAt || 0
      return timeB - timeA // 最新的在前
    })
  }
  
  // 保留最新的 maxItems 项
  const limited = {}
  sortedEntries.slice(0, maxItems).forEach(([key, value]) => {
    limited[key] = value
  })
  
  return limited
}

/**
 * 安全地设置 localStorage
 * @param {string} key - 存储键
 * @param {any} value - 存储值（会自动 JSON.stringify）
 * @param {Object} options - 选项
 * @param {number} options.maxArrayItems - 如果是数组，最大保留项数
 * @param {number} options.maxObjectItems - 如果是对象，最大保留项数
 * @param {Function} options.sortFn - 排序函数
 * @returns {boolean} 是否成功
 */
export function setItem(key, value, options = {}) {
  try {
    // 如果当前存储已满，先清理
    const currentSize = getStorageSize()
    if (currentSize >= CLEANUP_THRESHOLD) {
      console.warn('localStorage 容量接近限制，开始清理低优先级数据')
      cleanupLowPriorityData(WARNING_THRESHOLD)
    }
    
    // 处理数组和对象的限制
    let processedValue = value
    if (Array.isArray(value) && options.maxArrayItems) {
      processedValue = limitArraySize(value, options.maxArrayItems, options.sortFn)
      if (processedValue.length < value.length) {
        console.log(`限制数组大小: ${key} ${value.length} -> ${processedValue.length}`)
      }
    } else if (value && typeof value === 'object' && !Array.isArray(value) && options.maxObjectItems) {
      processedValue = limitObjectSize(value, options.maxObjectItems, options.sortFn)
      const originalKeys = Object.keys(value).length
      const newKeys = Object.keys(processedValue).length
      if (newKeys < originalKeys) {
        console.log(`限制对象大小: ${key} ${originalKeys} -> ${newKeys}`)
      }
    }
    
    // 序列化
    const serialized = typeof processedValue === 'string' 
      ? processedValue 
      : JSON.stringify(processedValue)
    
    // 检查序列化后的大小
    const itemSize = (key.length + serialized.length) * 2
    if (itemSize > STORAGE_LIMIT) {
      console.error(`存储项过大: ${key} (${(itemSize / 1024 / 1024).toFixed(2)}MB)`)
      return false
    }
    
    // 尝试保存
    localStorage.setItem(key, serialized)
    
    // 检查是否真的保存成功
    const saved = localStorage.getItem(key)
    if (!saved || saved !== serialized) {
      console.error(`保存验证失败: ${key}`)
      // 如果保存失败且是低优先级数据，尝试清理后再保存
      if (getPriority(key) < 70) {
        cleanupLowPriorityData(WARNING_THRESHOLD)
        localStorage.setItem(key, serialized)
        const saved2 = localStorage.getItem(key)
        if (!saved2 || saved2 !== serialized) {
          return false
        }
      } else {
        return false
      }
    }
    
    return true
  } catch (error) {
    // QuotaExceededError 或其他错误
    if (error.name === 'QuotaExceededError' || error.code === 22) {
      console.warn('localStorage 超出限制，尝试清理:', error)
      
      // 清理低优先级数据
      cleanupLowPriorityData(WARNING_THRESHOLD)
      
      // 再次尝试保存
      try {
        const serialized = typeof value === 'string' 
          ? value 
          : JSON.stringify(value)
        
        // 如果还是失败，且是低优先级数据，直接返回失败
        if (getPriority(key) < 70) {
          console.error(`低优先级数据保存失败: ${key}`)
          return false
        }
        
        // 高优先级数据，尝试更激进的清理
        cleanupLowPriorityData(getStorageSize() * 0.7) // 清理到70%
        localStorage.setItem(key, serialized)
        return true
      } catch (e) {
        console.error(`清理后保存仍然失败: ${key}`, e)
        return false
      }
    } else {
      console.error(`保存 localStorage 失败: ${key}`, error)
      return false
    }
  }
}

/**
 * 安全地获取 localStorage
 * @param {string} key - 存储键
 * @param {any} defaultValue - 默认值
 * @returns {any} 存储的值或默认值
 */
export function getItem(key, defaultValue = null) {
  try {
    const value = localStorage.getItem(key)
    if (value === null) {
      return defaultValue
    }
    
    // 尝试解析 JSON
    try {
      return JSON.parse(value)
    } catch (e) {
      // 如果不是 JSON，返回原始字符串
      return value
    }
  } catch (error) {
    console.error(`读取 localStorage 失败: ${key}`, error)
    return defaultValue
  }
}

/**
 * 安全地移除 localStorage
 * @param {string} key - 存储键
 */
export function removeItem(key) {
  try {
    localStorage.removeItem(key)
  } catch (error) {
    console.error(`删除 localStorage 失败: ${key}`, error)
  }
}

/**
 * 清空所有低优先级数据
 */
export function clearLowPriorityData() {
  const removedKeys = []
  for (let key in localStorage) {
    if (localStorage.hasOwnProperty(key) && getPriority(key) < 70) {
      try {
        localStorage.removeItem(key)
        removedKeys.push(key)
      } catch (e) {
        console.warn(`清理失败: ${key}`, e)
      }
    }
  }
  return removedKeys
}

/**
 * 获取存储使用情况
 */
export function getStorageInfo() {
  const size = getStorageSize()
  const keys = []
  for (let key in localStorage) {
    if (localStorage.hasOwnProperty(key)) {
      const value = localStorage.getItem(key)
      const itemSize = (key.length + (value ? value.length : 0)) * 2
      keys.push({
        key,
        size: itemSize,
        priority: getPriority(key),
        sizeKB: (itemSize / 1024).toFixed(2)
      })
    }
  }
  
  keys.sort((a, b) => b.size - a.size)
  
  return {
    totalSize: size,
    totalSizeMB: (size / 1024 / 1024).toFixed(2),
    usagePercent: ((size / STORAGE_LIMIT) * 100).toFixed(2),
    items: keys,
    itemCount: keys.length
  }
}

/**
 * 检查存储容量
 */
export function checkStorageCapacity() {
  const size = getStorageSize()
  const percent = (size / STORAGE_LIMIT) * 100
  
  return {
    size,
    sizeMB: (size / 1024 / 1024).toFixed(2),
    limit: STORAGE_LIMIT,
    limitMB: (STORAGE_LIMIT / 1024 / 1024).toFixed(2),
    percent: percent.toFixed(2),
    available: STORAGE_LIMIT - size,
    availableMB: ((STORAGE_LIMIT - size) / 1024 / 1024).toFixed(2),
    isNearLimit: size >= WARNING_THRESHOLD,
    needsCleanup: size >= CLEANUP_THRESHOLD
  }
}
