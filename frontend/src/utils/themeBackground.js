/**
 * 主题背景工具函数
 */

import { getItem, setItem } from './storageManager'

const STORAGE_KEY = 'easytools:theme:background'

// 当前已应用的主题配置缓存（用于避免重复应用相同配置）
let currentAppliedConfig = null

/**
 * 获取主题配置
 */
export function getThemeConfig() {
  return getItem(STORAGE_KEY, null)
}

/**
 * 比较两个配置是否相同
 */
function isConfigEqual(config1, config2) {
  if (!config1 || !config2) return config1 === config2
  return JSON.stringify(config1) === JSON.stringify(config2)
}


/**
 * 十六进制颜色转 RGB
 */
function hexToRgb(hex) {
  if (!hex) return null
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  return result ? {
    r: parseInt(result[1], 16),
    g: parseInt(result[2], 16),
    b: parseInt(result[3], 16)
  } : null
}

/**
 * 验证并修复配置中的 imageUrl
 * 如果 imageUrl 指向的文件不存在，尝试从 imageUrls 中选择可用的，或回退到默认配置
 */
async function validateAndFixConfig(config) {
  if (!config || config.type !== 'image' || !config.imageUrl) {
    return config
  }

  // 检查当前 imageUrl 是否可用
  const currentFileUrl = await getFileUrl(config.imageUrl)
  if (currentFileUrl) {
    // 当前 imageUrl 可用，直接返回
    return config
  }

  // 当前 imageUrl 不可用，尝试从 imageUrls 数组中找到可用的
  const imageUrls = config.imageUrls || []
  for (const url of imageUrls) {
    if (url && url !== config.imageUrl) {
      const fileUrl = await getFileUrl(url)
      if (fileUrl) {
        // 找到可用的背景，更新配置
        console.warn(`背景文件 ${config.imageUrl} 不可用，已切换到 ${url}`)
        const fixedConfig = {
          ...config,
          imageUrl: url
        }
        // 保存修复后的配置
        setItem(STORAGE_KEY, fixedConfig)
        return fixedConfig
      }
    }
  }

  // 所有背景都不可用，回退到默认配置
  console.warn('所有背景文件都不可用，已回退到默认纯色主题')
  const defaultColorConfig = {
    type: 'color',
    color: '#f5f7fa',
    imageUrls: [],
    imageUrl: '',
    opacity: 100,
    fontColor: '#333333',
    fontWeightRatio: 100,
    shadowRatio: 100,
    globalSearchHotkey: 'Ctrl+K',
  }
  // 保存默认配置
  setItem(STORAGE_KEY, defaultColorConfig)
  return defaultColorConfig
}

/**
 * 应用主题到整个应用
 * @param {boolean} force - 是否强制重新应用，即使配置相同
 */
export async function applyThemeToApp(force = false) {
  let config = getThemeConfig()
  if (!config) {
    // 如果没有配置，默认设置为纯色主题
    const defaultColorConfig = {
      type: 'color',
      color: '#f5f7fa',
      imageUrls: [],
      imageUrl: '',
      opacity: 100,
      fontColor: '#333333',
      fontWeightRatio: 100,
      shadowRatio: 100,
      globalSearchHotkey: 'Ctrl+K',
    }
    // 保存默认纯色主题配置
    setItem(STORAGE_KEY, defaultColorConfig)
    config = defaultColorConfig
  } else {
    // 验证并修复配置
    config = await validateAndFixConfig(config)
  }

  // 如果配置相同且不强制重新应用，跳过（避免不必要的重新应用导致闪烁）
  if (!force && currentAppliedConfig && isConfigEqual(config, currentAppliedConfig)) {
    return
  }

  // 更新缓存
  currentAppliedConfig = JSON.parse(JSON.stringify(config))

  // 直接应用，不使用延迟（延迟已在调用处处理）
  // 使用异步版本以支持文件路径转换
  _applyThemeToElementsImmediateAsync(config).catch(err => {
    console.error('应用主题失败:', err)
  })
}

/**
 * 立即应用主题（内部函数，异步版本）
 */
async function _applyThemeToElementsImmediateAsync(config) {
  // 如果是视频背景，在 applyThemeBackgroundAsync 中会智能处理视频切换
  // 这里只清除非视频的遮罩层，视频元素在 applyThemeBackgroundAsync 中处理
  const appElement = document.querySelector('.app')
  if (config && config.type === 'image' && config.imageUrl && isVideoFile(config.imageUrl)) {
    // 视频背景，不清除视频元素，让 applyThemeBackgroundAsync 智能处理
    // 只清除遮罩层
    const allOverlays = document.querySelectorAll('.theme-background-overlay')
    allOverlays.forEach(overlay => {
      try {
        overlay.remove()
      } catch (e) {
        console.warn('清除遮罩层失败:', e)
      }
    })
  } else {
    // 非视频背景，清除所有遮罩层和视频元素
    _clearAllOverlays()
  }
  
  // 只在最外层 .app 应用背景，让整个界面显示统一背景
  // 先应用 .app 的背景，确保背景已经设置好
  if (appElement) {
    await applyThemeBackgroundAsync(appElement, config)
    // 等待一小段时间，确保背景样式已经应用到 DOM
    await new Promise(resolve => setTimeout(resolve, 50))
  }

  // 标题栏使用半透明背景，保持可读性
  const titleBarElement = document.querySelector('.custom-titlebar')
  if (titleBarElement) {
    const titleBarConfig = { ...config, opacity: Math.min(config.opacity, 85) }
    await applyThemeBackgroundAsync(titleBarElement, titleBarConfig)
  }

  // 侧边栏保持透明，让主背景完全显示
  const menuElement = document.querySelector('.appMenu')
  if (menuElement) {
    menuElement.style.backgroundColor = 'transparent'
    menuElement.style.backgroundImage = 'none'
    menuElement.style.backgroundSize = ''
    menuElement.style.backgroundPosition = ''
    menuElement.style.backgroundRepeat = ''
    // 清除可能的渐变背景
    if (menuElement.style.background) {
      menuElement.style.background = ''
    }
    
    // 内部的 .menu 元素也保持透明
    const innerMenu = menuElement.querySelector('.menu')
    if (innerMenu) {
      innerMenu.style.backgroundColor = 'transparent'
      innerMenu.style.backgroundImage = 'none'
      if (innerMenu.style.background) {
        innerMenu.style.background = ''
      }
    }
  }

  // 主内容区域和页面容器保持透明，让主背景显示
  // 在 .app 背景应用完成后再设置透明，避免页面看起来透明
  const mainElement = document.querySelector('.appMain')
  if (mainElement) {
    mainElement.style.backgroundColor = 'transparent'
    mainElement.style.backgroundImage = 'none'
    if (mainElement.style.background) {
      mainElement.style.background = ''
    }
  }

  const mainContainer = document.querySelector('.main-container')
  if (mainContainer) {
    mainContainer.style.backgroundColor = 'transparent'
    mainContainer.style.backgroundImage = 'none'
    if (mainContainer.style.background) {
      mainContainer.style.background = ''
    }
  }

  // 所有功能页面的容器也保持透明，让主背景显示
  // 在 .app 背景应用完成后再清除页面容器背景
  clearPageContainersBackground()
  
  // 应用卡片等模块的半透明背景
  applyCardBackgrounds(config)
}

/**
 * 清除主题背景，恢复原有样式
 */
function _clearThemeBackground() {
  // 首先清除所有遮罩层
  _clearAllOverlays()
  
  const appElement = document.querySelector('.app')
  const titleBarElement = document.querySelector('.custom-titlebar')
  const menuElement = document.querySelector('.appMenu')
  const mainElement = document.querySelector('.appMain')
  const mainContainer = document.querySelector('.main-container')

  // 恢复原有样式
  const elementsToClear = [appElement, titleBarElement, menuElement, mainElement, mainContainer].filter(Boolean)
  
  elementsToClear.forEach(element => {
    element.style.backgroundColor = ''
    element.style.backgroundImage = ''
    element.style.backgroundSize = ''
    element.style.backgroundPosition = ''
    element.style.backgroundRepeat = ''
    // 恢复 position（如果有的话）
    if (element.style.position === 'relative') {
      element.style.position = ''
    }
  })

  // 恢复侧边栏内部的 .menu 元素样式
  if (menuElement) {
    const innerMenu = menuElement.querySelector('.menu')
    if (innerMenu) {
      innerMenu.style.backgroundColor = ''
      innerMenu.style.backgroundImage = ''
      if (innerMenu.style.background) {
        innerMenu.style.background = ''
      }
    }
  }

  // 清除所有页面容器的背景
  clearPageContainersBackground()

  // 清除卡片背景样式，但保留默认磨砂效果
  const styleElement = document.getElementById('theme-card-background-style')
  if (styleElement) {
    // 保留默认的磨砂效果（使用85%透明度，既能保持磨砂效果又能看清边框）
    const blurAmount = '8px'
    const defaultToolItemBg = 'rgb(255 255 255 / 85%)'
    styleElement.textContent = `
      .glass-card:not(.no-theme-bg),
      .el-card:not(.no-theme-bg),
      .el-tabs:not(.no-theme-bg),
      .tabs:not(.no-theme-bg) {
        backdrop-filter: blur(${blurAmount}) !important;
        -webkit-backdrop-filter: blur(${blurAmount}) !important;
      }
      .system .nav .tool-list .item:not(.no-theme-bg),
      .site .nav .site-list .item:not(.no-theme-bg),
      .system .nav .item:not(.no-theme-bg),
      .site .nav .item:not(.no-theme-bg) {
        background-color: ${defaultToolItemBg} !important;
        backdrop-filter: blur(${blurAmount}) !important;
        -webkit-backdrop-filter: blur(${blurAmount}) !important;
      }
      .el-input__wrapper:not(.no-theme-bg),
      .el-input__inner:not(.no-theme-bg),
      .el-textarea__inner:not(.no-theme-bg),
    `
  }
  
  // 额外清除一些通用容器（确保清除所有可能的背景）
  try {
    const allPossibleContainers = document.querySelectorAll('.system, .site')
    allPossibleContainers.forEach(element => {
      element.style.backgroundColor = ''
      element.style.backgroundImage = ''
      element.style.background = ''
      element.style.backgroundSize = ''
      element.style.backgroundPosition = ''
      element.style.backgroundRepeat = ''
    })
  } catch (e) {
    console.warn('清除通用容器背景时出错:', e)
  }

  // 恢复子元素的 z-index
  const allElements = [...elementsToClear]
  allElements.forEach(element => {
    const children = Array.from(element.children)
    children.forEach(child => {
      if (child.className !== 'theme-background-overlay') {
        const computedStyle = window.getComputedStyle(child)
        if (computedStyle.position === 'relative' && computedStyle.zIndex === '1') {
          child.style.position = ''
          child.style.zIndex = ''
        }
      }
    })
  })
}

/**
 * 清除所有遮罩层和视频元素（防止叠加）
 */
function _clearAllOverlays() {
  const allOverlays = document.querySelectorAll('.theme-background-overlay, .theme-background-video')
  allOverlays.forEach(overlay => {
    try {
      // 如果是视频元素，先停止播放
      if (overlay.tagName === 'VIDEO') {
        overlay.pause()
        overlay.src = ''
        overlay.load()
      }
      overlay.remove()
    } catch (e) {
      console.warn('清除遮罩层/视频失败:', e)
    }
  })
}

/**
 * 获取当前活动的视频元素
 */
function _getCurrentVideoElement(element) {
  if (!element) return null
  return element.querySelector('.theme-background-video')
}

/**
 * 获取当前视频的URL（用于判断是否需要重新加载）
 */
function _getCurrentVideoUrl(element) {
  const video = _getCurrentVideoElement(element)
  if (!video || !video.src) return null
  return video.src
}

/**
 * 判断是否为视频文件
 */
function isVideoFile(filePath) {
  if (!filePath) return false
  const videoExtensions = ['.mp4', '.webm', '.ogg', '.mov', '.avi', '.mkv']
  const lowerPath = filePath.toLowerCase()
  return videoExtensions.some(ext => lowerPath.endsWith(ext))
}

/**
 * 文件 URL 缓存（用于存储转换后的 data URL）
 */
const fileUrlCache = {}

/**
 * 获取文件 URL（将文件路径转换为 data URL，异步）
 */
async function getFileUrl(filePath) {
  if (!filePath) return ''
  // 如果已经是 URL（如 data: 或 http:），直接返回
  if (filePath.startsWith('data:') || filePath.startsWith('http://') || filePath.startsWith('https://')) {
    return filePath
  }
  // 如果是 public 目录下的资源路径（以 / 开头或 ./assets/ 开头），直接返回作为 URL，并加入缓存
  // 将 ./assets/ 转换为 /assets/ 的绝对路径
  let normalizedPath = filePath
  if (filePath.startsWith('./assets/')) {
    normalizedPath = filePath.replace(/^\.\//, '/')
  }
  
  if (normalizedPath.startsWith('/')) {
    // 将 public 资源路径加入缓存，避免重复处理
    if (!fileUrlCache[filePath]) {
      fileUrlCache[filePath] = normalizedPath
    }
    return normalizedPath
  }
  // 检查缓存
  if (fileUrlCache[filePath]) {
    return fileUrlCache[filePath]
  }
  // 如果是文件路径，通过后端 API 转换为 data URL
  try {
    // 使用全局的 window.go API
    const dataURL = await window['go']['system']['System']['ReadFileAsDataURL'](filePath)
    if (dataURL) {
      fileUrlCache[filePath] = dataURL
      return dataURL
    }
    return ''
  } catch (e) {
    console.error('读取文件失败:', e)
    return ''
  }
}

/**
 * 应用主题到各个元素（异步版本）
 */
async function applyThemeBackgroundAsync(element, config) {
  if (!element || !config) return

  const opacity = (config.opacity !== undefined ? config.opacity : 100) / 100

  if (config.type === 'color') {
    // 纯色背景，清除视频和遮罩层
    const existingOverlays = element.querySelectorAll('.theme-background-overlay, .theme-background-video')
    existingOverlays.forEach(overlay => {
      if (overlay.tagName === 'VIDEO') {
        overlay.pause()
        overlay.src = ''
        overlay.load()
      }
      overlay.remove()
    })
    // 纯色背景
    const rgb = hexToRgb(config.color || '#f5f7fa')
    if (rgb) {
      element.style.backgroundColor = `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${opacity})`
    } else {
      element.style.backgroundColor = config.color || '#f5f7fa'
    }
    element.style.backgroundImage = 'none'
    element.style.backgroundSize = 'auto'
    element.style.backgroundPosition = 'center'
    element.style.backgroundRepeat = 'no-repeat'
  } else if (config.type === 'image' && config.imageUrl) {
    let fileUrl = await getFileUrl(config.imageUrl)
    
    // 如果当前 imageUrl 加载失败，尝试从 imageUrls 数组中加载其他可用的背景
    if (!fileUrl && config.imageUrls && config.imageUrls.length > 0) {
      for (const url of config.imageUrls) {
        if (url && url !== config.imageUrl) {
          const testUrl = await getFileUrl(url)
          if (testUrl) {
            // 找到可用的背景，使用它
            fileUrl = testUrl
            // 更新配置中的 imageUrl（但不保存，因为 validateAndFixConfig 已经处理了）
            config.imageUrl = url
            console.warn(`背景文件加载失败，已切换到可用的背景: ${url}`)
            break
          }
        }
      }
    }
    
    if (!fileUrl) {
      // 如果所有背景都加载失败，使用默认背景
      element.style.backgroundColor = '#f5f7fa'
      element.style.backgroundImage = 'none'
      // 清除视频和遮罩层
      const existingOverlays = element.querySelectorAll('.theme-background-overlay, .theme-background-video')
      existingOverlays.forEach(overlay => {
        if (overlay.tagName === 'VIDEO') {
          overlay.pause()
          overlay.src = ''
          overlay.load()
        }
        overlay.remove()
      })
      return
    }
    const isVideo = isVideoFile(config.imageUrl)
    
    if (isVideo) {
      // 视频背景 - 使用 video 标签
      const currentVideoUrl = _getCurrentVideoUrl(element)
      const existingVideo = _getCurrentVideoElement(element)
      
      // 如果当前视频URL与新URL相同，只更新透明度和遮罩层，不重新创建视频
      if (existingVideo && currentVideoUrl === fileUrl) {
        // 只更新透明度和遮罩层
        existingVideo.style.opacity = opacity
        
        // 更新或创建透明度遮罩层
        const existingOverlay = element.querySelector('.theme-background-overlay')
        if (opacity < 1) {
          if (existingOverlay) {
            existingOverlay.style.backgroundColor = `rgba(255, 255, 255, ${1 - opacity})`
          } else {
            const overlay = document.createElement('div')
            overlay.className = 'theme-background-overlay'
            overlay.style.cssText = `
              position: absolute;
              top: 0;
              left: 0;
              right: 0;
              bottom: 0;
              background-color: rgba(255, 255, 255, ${1 - opacity});
              pointer-events: none;
              z-index: 1;
            `
            element.appendChild(overlay)
          }
        } else if (existingOverlay) {
          existingOverlay.remove()
        }
        
        element.style.backgroundColor = 'transparent'
        element.style.backgroundImage = 'none'
        return
      }
      
      // 如果视频不同，采用"先加载新视频，后移除旧视频"的策略，避免白色闪烁
      element.style.backgroundColor = 'transparent'
      element.style.backgroundImage = 'none'
      
      // 确保元素有相对定位
      const originalPosition = element.style.position
      if (!originalPosition || originalPosition === 'static') {
        element.style.position = 'relative'
      }
      
      // 创建新视频元素
      const newVideo = document.createElement('video')
      newVideo.className = 'theme-background-video'
      newVideo.autoplay = true
      newVideo.loop = true
      newVideo.muted = true
      newVideo.playsInline = true
      
      // 初始时，如果存在旧视频，新视频放在下方（较低z-index），等加载完成后再提升
      const initialZIndex = existingVideo ? -1 : 0
      newVideo.style.cssText = `
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        object-fit: cover;
        z-index: ${initialZIndex};
        opacity: ${opacity};
        pointer-events: none;
      `
      
      // 处理视频加载错误
      newVideo.onerror = () => {
        console.error('视频加载失败:', fileUrl)
        newVideo.remove()
        // 如果加载失败，保留旧视频（如果存在），否则设置默认背景
        if (!existingVideo) {
          element.style.backgroundColor = '#f5f7fa'
        }
        // 如果旧视频存在，保持原样，不改变背景
      }
      
      // 立即添加新视频到DOM（在旧视频下方加载）
      element.appendChild(newVideo)
      
      // 开始加载新视频
      newVideo.src = fileUrl
      newVideo.load()
      
      // 当新视频可以播放时，提升z-index并移除旧视频
      newVideo.addEventListener('canplay', () => {
        // 确保视频能播放
        newVideo.play().catch(err => {
          console.warn('视频自动播放失败:', err)
        })
        
        // 提升新视频的z-index到正常位置
        newVideo.style.zIndex = '0'
        
        // 移除旧视频和旧的遮罩层
        if (existingVideo) {
          existingVideo.pause()
          existingVideo.src = ''
          existingVideo.load()
          existingVideo.remove()
        }
        
        // 清除旧的遮罩层
        const existingOverlays = element.querySelectorAll('.theme-background-overlay')
        existingOverlays.forEach(overlay => overlay.remove())
        
        // 创建新的透明度遮罩层（如果需要）
        if (opacity < 1) {
          const overlay = document.createElement('div')
          overlay.className = 'theme-background-overlay'
          overlay.style.cssText = `
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background-color: rgba(255, 255, 255, ${1 - opacity});
            pointer-events: none;
            z-index: 1;
          `
          element.appendChild(overlay)
        }
      }, { once: true })
    } else {
      // 图片背景，清除视频元素
      const existingOverlays = element.querySelectorAll('.theme-background-overlay, .theme-background-video')
      existingOverlays.forEach(overlay => {
        if (overlay.tagName === 'VIDEO') {
          overlay.pause()
          overlay.src = ''
          overlay.load()
        }
        overlay.remove()
      })
      // 图片背景 - 使用遮罩层实现透明度
      element.style.backgroundImage = `url(${fileUrl})`
      element.style.backgroundSize = 'cover'
      element.style.backgroundPosition = 'center'
      element.style.backgroundRepeat = 'no-repeat'
      element.style.backgroundColor = 'transparent'
      
      // 确保元素有相对定位
      const originalPosition = element.style.position
      if (!originalPosition || originalPosition === 'static') {
        element.style.position = 'relative'
      }
      
      // 创建透明度遮罩层
      const overlay = document.createElement('div')
      overlay.className = 'theme-background-overlay'
      overlay.style.cssText = `
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background-color: rgba(255, 255, 255, ${1 - opacity});
        pointer-events: none;
        z-index: 0;
      `
      
      element.appendChild(overlay)
    }
    
    // 确保所有子元素在遮罩层/视频之上
    const children = Array.from(element.children)
    children.forEach(child => {
      if (child.className !== 'theme-background-overlay' && child.className !== 'theme-background-video') {
        const computedStyle = window.getComputedStyle(child)
        const childPosition = computedStyle.position
        const childZIndex = computedStyle.zIndex
        
        if (childPosition === 'static') {
          child.style.position = 'relative'
        }
        if (!childZIndex || childZIndex === 'auto' || childZIndex === '0') {
          child.style.zIndex = isVideo ? '2' : '1'
        }
      }
    })
  } else {
    // 默认背景
    element.style.backgroundColor = '#f5f7fa'
    element.style.backgroundImage = 'none'
  }
}

/**
 * 清除页面容器的背景，让主背景显示
 */
function clearPageContainersBackground() {
  // 扩展所有功能界面的容器类名
  const pageContainerSelectors = [
    // 通用容器
    '.app-container',
    '.system',
    '.site',
    // 扫描相关

    '.hostscan-container',
    '.dirsearch-container',
    '.fingerprint-container',
    // Restmate
    '.restmate-page',
    '.restmate-container',
    // CLI执行器
    '.cli-executor-container',
    '.tools-page',
    '.runner-page',
    '.history-page',
    // 绕过相关
    '.skyshadow-container',
    // SSH/连接
    '.ssh-page',
    // 备忘录
    '.note-views',
    // iframe容器
    '.iframe-container',
    // 辅助工具
    '.google-syntax-container',
    '.map-query-container',
    // 其他
    // 注意：.theme-background-container、.menu-manager 和 .sponsor-page 需要保留白色背景，不在此列表中
    // 添加更多可能的容器
    '.el-container',
    '.el-main',
    '.main-layout',
    '.left-panel',
    '.right-panel',
    '.content-panel',
    // 标签页容器（确保标签页背景也能应用主题）
    '.tabs',
    '.el-tabs',
    '.el-tabs__header',
    '.el-tabs__nav-wrap',
    '.el-tabs__content',
    // 更多页面容器
    '.filter-toolbar',
    '.toolbar-content',
    '.toolbar-left',
    '.toolbar-right',
    '.detail-wrapper',
    '.packet-inline-panels',
    '.packet-inline-panel',
    '.vuln-table',
    '.vuln-name',
    '.overview-header',
    '.overview-title-section',
    '.overview-content-section',
    '.overview-actions',
    '.title-with-progress',
    '.progress-inline',
    '.progress-content',
    '.progress-bar-wrapper',
    '.progress-meta',
    '.tab-indicator',
    '.empty-tabs-state',
    '.poc-info',
    '.risk-level-display',
    '.risk-badge',
    '.main-tabs',
    '.panel-card',
    '.empty-tabs-container',
    '.top-toolbar',
    '.overview-card',
    '.config-card',
    '.generator-card',
    '.poc-card',
    '.tool-card',
    '.cate-tree',
    '.upload-section',
    '.select-section',
    '.action-section',
    '.form-card',
    '.result-card',
    '.custom-table',
    '.coll-req-table',
    '.env-select',
    '.table-card',
    '.software-table',
    '.panel-section',
    '.results-container',
    '.hgdumper-header-card',
    '.batch-url-input-card',
    '.single-url-input-card',
    '.indexdumper-header-card',
    '.svndumper-header-card',
    '.dsstoredumper-header-card',
    '.gitdumper-header-card',
    '.oss-header-card',
    '.url-input-card',
    '.oss-result-card',
    '.applet-list-panel',
    '.detail-panel',
    // 注意：.menu-manager 需要保留白色背景，不在此列表中
    '.elevated-table',
    '.menu-section',
    '.menu-list',
    '.menu-item-container',
    '.menu-item',
    '.module-tabs-dialog',
    '.module-tabs-list',
    '.module-tab-item',
    '.action-buttons',
    '.custom-template-item',
    '.custom-template-select',
    '.custom-template-popper',
    '.color-picker-wrapper',
    '.image-upload-wrapper',
    '.image-uploader',
    '.preview-box',
    '.preview-content',
    '.opacity-control',
    '.image-actions',
    '.image-tips',
    '.opacity-tips',
    // OSS存储桶相关
    '.oss-content',
    '.oss-header-card',
    '.oss-url-section',
    '.url-input-card',
    '.url-input-header',
    '.url-input-group',
    '.url-input',
    '.detection-result',
    '.batch-scan-section',
    '.batch-url-input-card',
    '.batch-input-group',
    '.batch-textarea',
    '.batch-actions',
    '.oss-function-section',
    '.scan-config-section',
    '.config-section',
    '.config-grid',
    '.config-item',
    '.config-input',
    '.directory-input-group',
    '.slider-container',
    '.vuln-scan-config',
    '.scan-options',
    '.scan-option-item',
    // 文件泄露相关
    '.log-section',
    '.log-header',
    '.log-actions',
    '.log-content',
    '.log-item',
    // SSH和Redis相关（只添加容器，不添加内部元素，避免影响原有样式）
    '.redis-container',
    '.redis-connection',
    '.redis-keys',
    '.redis-db-info',
    // PCAP分析器相关
    '.pcap-app',
    '.app-header',
    '.header-content',
    '.title-section',
    '.status-bar',
    '.status-info',
    '.status-indicator',
    '.status',
    '.processing-stats',
    '.stats',
    '.stat-item',
    '.file-selector',
    '.file-dropzone',
    '.dropzone-content',
    '.dropzone-icon',
    '.file-path-input',
    '.file-info',
    '.file-info-header',
    '.file-icon',
    '.file-name',
    '.config-preview',
    '.config-label',
    '.config-value',
    '.config-edit-btn',
    '.switch-grid',
    '.switch-item',
    '.results-panel',
    '.results-header',
    '.results-content',
    '.result-item',
    '.result-header',
    '.result-body',
    '.extracted-files',
    '.file-list',
    '.file-item',
    // 便携发包相关
    '.restmate-page',
    '.restmate-container',
    '.restmate-aside',
    '.aside-header',
    '.aside-title',
    '.aside-actions',
    '.aside-toggle-btn',
    '.aside-scroll',
    '.collections-loading',
    '.loading-text',
    '.coll-title',
    '.coll-name',
    '.el-dropdown-link',
    '.req-method',
    '.req-name',
    '.restmate-main',
    '.main-header',
    '.main-content',
    // Google语法查询相关
    '.google-syntax-container',
    '.form-group',
    '.result-controls',
    '.search-section',
    '.result-pagination',
    '.query-results',
    '.command-cell',
    '.command-text',
    '.action-buttons',
    '.no-results',
    '.manage-toolbar',
    // 密码查询相关
    '.password-management',
    '.header',
    // .search-bar 不在此列表中，保留其黑色背景
    // 免杀生成相关
    '.skyshadow-selector',
    '.skyshadow-row',
    '.skyshadow-select',
    '.skyshadow-payload-dropdown',
    '.skyshadow-functions-dropdown',
    '.c2-module',
    '.bai-jia-hei-module',
    '.white-black-module',
    '.skyshadow-container',
    '.skyshadow-header',
    '.skyshadow-layout',
    '.left-panel',
    '.right-panel',
    '.form-card',
    '.result-card',
    '.table-card',
    '.log-card',
    '.table-toolbar',
    '.home',
    '.upload-container',
    '.upload-section',
    '.select-section',
    '.action-section',
    '.bypass-container',
    // 软件中心相关
    '.software-center',
    '.table-header',
    '.title-group',
    '.title-wrapper',
    '.title-icon',
    '.title',
    '.subtitle',
    '.search-box',
    '.search-icon',
    '.table-wrapper',
    '.name-cell',
    '.name-text',
    '.name-new-sup',
    '.tag-list',
    '.more-tag',
    '.table-card',
    '.table-footer',
    '.software-table',
    '.action-cell',
    '.last-updated-cell',
    '.version-text',
    // POC管理相关
    '.elevated-table',
    '.hero-title',
    '.request-card',
    '.selected-poc-list',
    // 随机生成相关（使用更具体的选择器避免冲突）
    '.id-generator',
    '.id-generator .header',
    '.id-generator .main-content',
    '.id-generator .form-section',
    '.id-generator .form-card',
    '.id-generator .section-title',
    '.id-generator .generate-button',
    '.id-generator .result-section',
    '.id-generator .result-card',
    // 关于页面相关
    '.sponsor-page',
    '.header',
    '.header-left',
    '.header-text',
    '.header-right',
    '.qr-card',
    '.qr-wrapper',
    '.content-area',
    '.main-column',
    '.side-column',
    '.update-card',
    '.card-header',
    '.card-subtitle',
    '.list',
    '.scrollable',
    '.tall-list',
    '.update-item',
    '.timeline-dot',
    '.date',
    '.text',
    '.large-text',
    '.new-badge',
    '.compact-card',
    '.thanks-grid',
    '.small-max',
    '.thanks-item',
    '.avatar-container',
    '.avatar-large',
    '.name',
    '.sponsor-grid',
    '.sponsor-card',
    '.sponsor-img-wrap',
    '.sponsor-img',
    '.glass-effect',
    '.chip',
    '.chip-primary',
    '.cta-group',
    '.btn',
    '.row-inline',
    '.qr-image'
  ]

  // 需要保留白色背景的界面（不设置为透明）
  const preserveWhiteBackgroundSelectors = [
    '.theme-background-container',
    '.menu-manager',
    '.sponsor-page'  // 关于界面
  ]

  pageContainerSelectors.forEach(selector => {
    try {
      // 跳过需要保留白色背景的界面
      if (preserveWhiteBackgroundSelectors.includes(selector)) {
        return
      }
      
      const elements = document.querySelectorAll(selector)
      elements.forEach(element => {
        // 清除背景，让主背景透过来
        element.style.backgroundColor = 'transparent'
        element.style.backgroundImage = 'none'
        element.style.backgroundSize = ''
        element.style.backgroundPosition = ''
        element.style.backgroundRepeat = ''
        // 清除可能的渐变背景
        if (element.style.background) {
          element.style.background = ''
        }
      })
    } catch (e) {
      // 忽略选择器错误
      console.warn(`清除背景时出错 (${selector}):`, e)
    }
  })
  
  // 确保主题管理、菜单管理和关于界面保持白色背景
  preserveWhiteBackgroundSelectors.forEach(selector => {
    try {
      const elements = document.querySelectorAll(selector)
      elements.forEach(element => {
        element.style.backgroundColor = '#ffffff'
        element.style.backgroundImage = 'none'
      })
    } catch (e) {
      console.warn(`设置白色背景时出错 (${selector}):`, e)
    }
  })
}

/**
 * 应用卡片等模块的半透明背景（磨砂效果）
 */
function applyCardBackgrounds(config) {
  const moduleOpacity = 0.55
  
  // 添加全局样式来设置卡片背景
  let styleElement = document.getElementById('theme-card-background-style')
  if (!styleElement) {
    styleElement = document.createElement('style')
    styleElement.id = 'theme-card-background-style'
    // 插入到 body 末尾，确保在 Element Plus 样式之后加载
    document.body.appendChild(styleElement)
  }

  // 磨砂效果的模糊度
  const blurAmount = '8px'

  // 通用模块选择器列表（包含所有常见的模块容器）
  const moduleSelectors = `
    .el-card:not(.no-theme-bg),
    .glass-card:not(.no-theme-bg),
    .el-dialog:not(.no-theme-bg),
    .el-drawer:not(.no-theme-bg),
    .el-table:not(.no-theme-bg),
    .el-tabs:not(.no-theme-bg),
    .el-tabs__content:not(.no-theme-bg),
    .el-tabs__header:not(.no-theme-bg),
    .el-tabs__nav-wrap:not(.no-theme-bg),
    .el-tabs__item:not(.no-theme-bg),
    .el-card__body:not(.no-theme-bg),
    .el-dialog__body:not(.no-theme-bg),
    .el-dialog__header:not(.no-theme-bg),
    .el-dialog__footer:not(.no-theme-bg),
    .el-table__body-wrapper:not(.no-theme-bg),
    .el-table__header-wrapper:not(.no-theme-bg),
    .el-table__header:not(.no-theme-bg),
    .el-table__body:not(.no-theme-bg),
    .el-table__row:not(.no-theme-bg),
    .config-card:not(.no-theme-bg),
    .overview-card:not(.no-theme-bg),
    .panel-card:not(.no-theme-bg),
    .poc-card:not(.no-theme-bg),
    .generator-card:not(.no-theme-bg),
    .top-toolbar:not(.no-theme-bg),
    .empty-tabs-container:not(.no-theme-bg),
    .tabs:not(.no-theme-bg),
    .tool-card:not(.no-theme-bg),
    .app-container:not(.no-theme-bg),
    .cate-tree:not(.no-theme-bg),
    .el-descriptions:not(.no-theme-bg),
    .el-descriptions__body:not(.no-theme-bg),
    .el-descriptions__table:not(.no-theme-bg),
    .el-pagination:not(.no-theme-bg),
    .el-pagination__wrapper:not(.no-theme-bg),
    .el-badge:not(.no-theme-bg),
    .el-tooltip__popper:not(.no-theme-bg),
    .el-popover:not(.no-theme-bg),
    .el-popover__content:not(.no-theme-bg),
    .el-dropdown-menu:not(.no-theme-bg),
    .el-menu:not(.no-theme-bg),
    .el-menu-item:not(.no-theme-bg),
    .el-sub-menu:not(.no-theme-bg),
    .el-collapse:not(.no-theme-bg),
    .el-collapse-item:not(.no-theme-bg),
    .el-collapse-item__header:not(.no-theme-bg),
    .el-collapse-item__content:not(.no-theme-bg),
    .el-steps:not(.no-theme-bg),
    .el-step:not(.no-theme-bg),
    .el-timeline:not(.no-theme-bg),
    .el-timeline-item:not(.no-theme-bg),
    .el-alert:not(.no-theme-bg),
    .el-notification:not(.no-theme-bg),
    .el-loading-mask:not(.no-theme-bg),
    .el-loading-spinner:not(.no-theme-bg),
    .el-skeleton:not(.no-theme-bg),
    .el-skeleton__item:not(.no-theme-bg),
    .el-empty:not(.no-theme-bg),
    .el-result:not(.no-theme-bg),
    .el-divider:not(.no-theme-bg),
    .el-backtop:not(.no-theme-bg),
    .el-affix:not(.no-theme-bg),
    .el-anchor:not(.no-theme-bg),
    .el-anchor-link:not(.no-theme-bg),
    .el-breadcrumb:not(.no-theme-bg),
    .el-breadcrumb-item:not(.no-theme-bg),
    .el-page-header:not(.no-theme-bg),
    .el-tag:not(.no-theme-bg),
    .el-link:not(.no-theme-bg),
    .el-text:not(.no-theme-bg),
    .el-progress:not(.no-theme-bg),
    .el-progress-bar:not(.no-theme-bg),
    .el-progress-bar__outer:not(.no-theme-bg),
    .el-progress-bar__inner:not(.no-theme-bg),
    .el-rate:not(.no-theme-bg),
    .el-slider:not(.no-theme-bg),
    .el-slider__runway:not(.no-theme-bg),
    .el-slider__bar:not(.no-theme-bg),
    .el-slider__button:not(.no-theme-bg),
    .el-switch:not(.no-theme-bg),
    .el-checkbox:not(.no-theme-bg),
    .el-checkbox-group:not(.no-theme-bg),
    .el-radio:not(.no-theme-bg),
    .el-radio-group:not(.no-theme-bg),
    /* 按钮不应用主题背景，保持原有样式 */
    /*     /* 按钮不应用主题背景，保持原有样式 */
    /* .el-button:not(.no-theme-bg),
    .el-button-group:not(.no-theme-bg), */ */
    .el-color-picker:not(.no-theme-bg),
    .el-color-picker__trigger:not(.no-theme-bg),
    .el-date-picker:not(.no-theme-bg),
    .el-time-picker:not(.no-theme-bg),
    .el-cascader:not(.no-theme-bg),
    .el-transfer:not(.no-theme-bg),
    .el-upload:not(.no-theme-bg),
    .el-upload-list:not(.no-theme-bg),
    .el-upload-list__item:not(.no-theme-bg),
    .el-image:not(.no-theme-bg),
    .el-image-viewer:not(.no-theme-bg),
    .el-avatar:not(.no-theme-bg),
    .el-badge__content:not(.no-theme-bg),
    .el-carousel:not(.no-theme-bg),
    .el-carousel__item:not(.no-theme-bg),
    .el-carousel__arrow:not(.no-theme-bg),
    .el-carousel__indicators:not(.no-theme-bg),
    .el-carousel__indicator:not(.no-theme-bg),
    .el-scrollbar:not(.no-theme-bg),
    .el-scrollbar__wrap:not(.no-theme-bg),
    .el-scrollbar__bar:not(.no-theme-bg),
    .el-scrollbar__thumb:not(.no-theme-bg),
    .el-infinite-scroll:not(.no-theme-bg),
    .el-watermark:not(.no-theme-bg),
    .el-statistic:not(.no-theme-bg),
    .el-statistic__head:not(.no-theme-bg),
    .el-statistic__number:not(.no-theme-bg),
    .el-statistic__content:not(.no-theme-bg)
  `
  
  // 表格和输入框选择器（使用稍微不同的透明度，保证可读性）
  const formSelectors = `
    .el-input__wrapper:not(.no-theme-bg),
    .el-input__inner:not(.no-theme-bg),
    .el-textarea__inner:not(.no-theme-bg),
    .el-form-item__label:not(.no-theme-bg),
    .el-form-item:not(.no-theme-bg),
    .el-form:not(.no-theme-bg),
    .el-table__header:not(.no-theme-bg),
    .el-table__header th:not(.no-theme-bg),
    .el-table__body:not(.no-theme-bg),
    .el-table__body td:not(.no-theme-bg),
    .el-table__body tr:not(.no-theme-bg),
    .el-table__cell:not(.no-theme-bg),
    .el-table__row:not(.no-theme-bg),
    .el-table th:not(.no-theme-bg),
    .el-table td:not(.no-theme-bg),
    .el-table tr:not(.no-theme-bg),
    .el-table__empty-block:not(.no-theme-bg),
    .el-table__empty-text:not(.no-theme-bg),
    .custom-table:not(.no-theme-bg),
    .query-results .el-table:not(.no-theme-bg),
    .inner-table:not(.no-theme-bg),
    .el-radio__input:not(.no-theme-bg),
    .el-radio__label:not(.no-theme-bg),
    .el-checkbox__input:not(.no-theme-bg),
    .el-checkbox__label:not(.no-theme-bg),
    .el-switch__core:not(.no-theme-bg),
    .el-slider__button-wrapper:not(.no-theme-bg),
    .el-date-editor:not(.no-theme-bg),
    .el-time-picker:not(.no-theme-bg),
    .el-cascader__tags:not(.no-theme-bg),
    .el-transfer-panel:not(.no-theme-bg),
    .el-transfer-panel__body:not(.no-theme-bg),
    .el-transfer-panel__list:not(.no-theme-bg),
    .el-upload-dragger:not(.no-theme-bg),
    .el-upload-list__item-name:not(.no-theme-bg),
    .el-color-picker__trigger:not(.no-theme-bg),
    .el-color-picker__color:not(.no-theme-bg),
    .el-color-picker__alpha:not(.no-theme-bg),
    .el-color-picker__preview:not(.no-theme-bg)
  `
  
  // 工具项选择器（工具仓库和网址导航中的工具项）- 使用更具体的选择器提高优先级
  const toolItemSelectors = `
    .system .nav .tool-list .item:not(.no-theme-bg),
    .site .nav .site-list .item:not(.no-theme-bg),
    .system .nav .item:not(.no-theme-bg),
    .site .nav .item:not(.no-theme-bg)
  `
  
  // 其他模块容器选择器
  const moduleContainerSelectors = `
    .pcap-app:not(.no-theme-bg),
    .restmate-page:not(.no-theme-bg),
    .restmate-container:not(.no-theme-bg),
    .cli-executor-container:not(.no-theme-bg),
    .tools-page:not(.no-theme-bg),
    .cyberSabre-container:not(.no-theme-bg),
    .webscan-page:not(.no-theme-bg),
    .hostscan-container:not(.no-theme-bg),
    .dirsearch-container:not(.no-theme-bg),
    .fingerprint-container:not(.no-theme-bg),
    .poc-page:not(.no-theme-bg),
    .ssh-page:not(.no-theme-bg),
    .note-views:not(.no-theme-bg),
    .iframe-container:not(.no-theme-bg),
    .google-syntax-container:not(.no-theme-bg),
    .map-query-container:not(.no-theme-bg),
    .sponsor-page:not(.no-theme-bg),
    .theme-background-container:not(.no-theme-bg),
    .main-layout:not(.no-theme-bg),
    .left-panel:not(.no-theme-bg),
    .right-panel:not(.no-theme-bg),
    .content-panel:not(.no-theme-bg),
    .filter-toolbar:not(.no-theme-bg),
    .toolbar-content:not(.no-theme-bg),
    .toolbar-left:not(.no-theme-bg),
    .toolbar-right:not(.no-theme-bg),
    .detail-wrapper:not(.no-theme-bg),
    .packet-inline-panels:not(.no-theme-bg),
    .packet-inline-panel:not(.no-theme-bg),
    .packet-inline-header:not(.no-theme-bg),
    .vuln-table:not(.no-theme-bg),
    .vuln-name:not(.no-theme-bg),
    .vuln-name-label:not(.no-theme-bg),
    .vuln-template:not(.no-theme-bg),
    .severity-pill:not(.no-theme-bg),
    .protocol-chip:not(.no-theme-bg),
    .tab-label:not(.no-theme-bg),
    .tab-name:not(.no-theme-bg),
    .tab-status-icon:not(.no-theme-bg),
    .tab-badge:not(.no-theme-bg),
    .tabs-context-menu:not(.no-theme-bg),
    .poc-info:not(.no-theme-bg),
    .info-label:not(.no-theme-bg),
    .info-value:not(.no-theme-bg),
    .risk-level-display:not(.no-theme-bg),
    .risk-badge:not(.no-theme-bg),
    .overview-header:not(.no-theme-bg),
    .overview-title-section:not(.no-theme-bg),
    .overview-content-section:not(.no-theme-bg),
    .overview-actions:not(.no-theme-bg),
    .title-with-progress:not(.no-theme-bg),
    .progress-inline:not(.no-theme-bg),
    .progress-content:not(.no-theme-bg),
    .progress-bar-wrapper:not(.no-theme-bg),
    .progress-meta:not(.no-theme-bg),
    .tab-indicator:not(.no-theme-bg),
    .empty-tabs-state:not(.no-theme-bg),
    .dialog-title:not(.no-theme-bg),
    .color-picker-wrapper:not(.no-theme-bg),
    .image-upload-wrapper:not(.no-theme-bg),
    .image-uploader:not(.no-theme-bg),
    .preview-image:not(.no-theme-bg),
    .uploader-icon:not(.no-theme-bg),
    .image-actions:not(.no-theme-bg),
    .image-tips:not(.no-theme-bg),
    .opacity-tips:not(.no-theme-bg),
    .opacity-control:not(.no-theme-bg),
    .opacity-value:not(.no-theme-bg),
    .preview-box:not(.no-theme-bg),
    .preview-content:not(.no-theme-bg)
  `

  // 确定背景颜色 - 所有模块统一使用白色半透明背景，避免颜色叠加
  // 这样无论主题背景是什么颜色，模块都保持统一的白色半透明效果
  const cardBg = `rgba(255, 255, 255, ${moduleOpacity})`
  
  // 对话框使用完全不透明背景，保持原有样式
  const dialogBg = `#ffffff`
  
  // 工具项使用半透明背景（55%透明度，既能保持磨砂效果又能看清边框）
  const toolItemOpacity = 0.55
  const toolItemBg = `rgb(255 255 255 / ${toolItemOpacity * 100}%)`
  
  // 表格和输入框使用更高的透明度（更不透明，保证可读性和输入体验）
  const formOpacity = Math.min(moduleOpacity + 0.15, 0.98)
  const formBg = `rgba(255, 255, 255, ${formOpacity})`
  
  // 按钮使用更高的透明度，确保文字可见
  const buttonOpacity = 0.95
  const buttonBg = `rgba(255, 255, 255, ${buttonOpacity})`
  
  // 对话框选择器（使用半透明55%磨砂背景）
  const dialogSelectors = `
    .el-dialog:not(.no-theme-bg),
    .el-dialog__header:not(.no-theme-bg),
    .el-dialog__body:not(.no-theme-bg),
    .el-dialog__footer:not(.no-theme-bg),
    .el-dialog__title:not(.no-theme-bg),
    .el-dialog__close:not(.no-theme-bg),
    .el-drawer:not(.no-theme-bg),
    .el-drawer__header:not(.no-theme-bg),
    .el-drawer__body:not(.no-theme-bg),
    .el-drawer__footer:not(.no-theme-bg),
    .el-drawer__title:not(.no-theme-bg),
    .el-drawer__close:not(.no-theme-bg)
  `
  
  // 按钮选择器 - 已移除，所有按钮保持原有样式，不受主题背景影响
  const buttonSelectors = ``
  
  // OSS存储桶相关选择器
  const ossSelectors = `
    .oss-content:not(.no-theme-bg),
    .oss-header-card:not(.no-theme-bg),
    .oss-url-section:not(.no-theme-bg),
    .url-input-card:not(.no-theme-bg),
    .url-input-header:not(.no-theme-bg),
    .url-input-group:not(.no-theme-bg),
    .url-input:not(.no-theme-bg),
    .detection-result:not(.no-theme-bg),
    .batch-scan-section:not(.no-theme-bg),
    .batch-url-input-card:not(.no-theme-bg),
    .batch-input-group:not(.no-theme-bg),
    .batch-textarea:not(.no-theme-bg),
    .batch-actions:not(.no-theme-bg),
    .oss-function-section:not(.no-theme-bg),
    .scan-config-section:not(.no-theme-bg),
    .config-section:not(.no-theme-bg),
    .config-grid:not(.no-theme-bg),
    .config-item:not(.no-theme-bg),
    .config-input:not(.no-theme-bg),
    .directory-input-group:not(.no-theme-bg),
    .slider-container:not(.no-theme-bg),
    .vuln-scan-config:not(.no-theme-bg),
    .scan-options:not(.no-theme-bg),
    .scan-option-item:not(.no-theme-bg)
  `
  
  // 文件泄露相关选择器
  const fileDumperSelectors = `
    .scan-config-section:not(.no-theme-bg),
    .config-section:not(.no-theme-bg),
    .config-grid:not(.no-theme-bg),
    .config-item:not(.no-theme-bg),
    .log-section:not(.no-theme-bg),
    .log-header:not(.no-theme-bg),
    .log-actions:not(.no-theme-bg),
    .log-content:not(.no-theme-bg),
    .log-item:not(.no-theme-bg)
  `
  
  // SSH和Redis相关选择器（只针对容器，不影响内部元素样式）
  const sshRedisSelectors = `
    .redis-container:not(.no-theme-bg),
    .redis-connection:not(.no-theme-bg),
    .redis-keys:not(.no-theme-bg),
    .redis-db-info:not(.no-theme-bg)
  `
  
  // PCAP分析器相关选择器
  const pcapSelectors = `
    .pcap-app:not(.no-theme-bg),
    .app-header:not(.no-theme-bg),
    .header-content:not(.no-theme-bg),
    .title-section:not(.no-theme-bg),
    .status-bar:not(.no-theme-bg),
    .status-info:not(.no-theme-bg),
    .status-indicator:not(.no-theme-bg),
    .status:not(.no-theme-bg),
    .processing-stats:not(.no-theme-bg),
    .stats:not(.no-theme-bg),
    .stat-item:not(.no-theme-bg),
    .main-layout:not(.no-theme-bg),
    .left-panel:not(.no-theme-bg),
    .file-selector:not(.no-theme-bg),
    .file-dropzone:not(.no-theme-bg),
    .dropzone-content:not(.no-theme-bg),
    .dropzone-icon:not(.no-theme-bg),
    .file-path-input:not(.no-theme-bg),
    .file-info:not(.no-theme-bg),
    .file-info-header:not(.no-theme-bg),
    .file-icon:not(.no-theme-bg),
    .file-name:not(.no-theme-bg),
    .config-preview:not(.no-theme-bg),
    .config-label:not(.no-theme-bg),
    .config-value:not(.no-theme-bg),
    .config-edit-btn:not(.no-theme-bg),
    .switch-grid:not(.no-theme-bg),
    .switch-item:not(.no-theme-bg),
    .right-panel:not(.no-theme-bg),
    .results-panel:not(.no-theme-bg),
    .results-header:not(.no-theme-bg),
    .results-content:not(.no-theme-bg),
    .result-item:not(.no-theme-bg),
    .result-header:not(.no-theme-bg),
    .result-body:not(.no-theme-bg),
    .extracted-files:not(.no-theme-bg),
    .file-list:not(.no-theme-bg),
    .file-item:not(.no-theme-bg)
  `
  
  // 便携发包（Restmate）：不再施加半透明底色 + 磨砂，避免整页发灰发雾；卡片仍可走通用 el-card / glass-card 规则
  
  // Google语法查询相关选择器
  const googleSelectors = `
    .google-syntax-container:not(.no-theme-bg),
    .form-group:not(.no-theme-bg),
    .result-controls:not(.no-theme-bg),
    .search-section:not(.no-theme-bg),
    .result-pagination:not(.no-theme-bg),
    .query-results:not(.no-theme-bg),
    .command-cell:not(.no-theme-bg),
    .command-text:not(.no-theme-bg),
    .action-buttons:not(.no-theme-bg),
    .no-results:not(.no-theme-bg),
    .manage-toolbar:not(.no-theme-bg)
  `
  
  // 密码查询相关选择器
  const passwordSelectors = `
    .password-management:not(.no-theme-bg),
    .header:not(.no-theme-bg),
    .search-bar:not(.no-theme-bg)
  `
  
  // 免杀生成相关选择器
  const bypassSelectors = `
    .skyshadow-selector:not(.no-theme-bg),
    .skyshadow-row:not(.no-theme-bg),
    .skyshadow-select:not(.no-theme-bg),
    .skyshadow-payload-dropdown:not(.no-theme-bg),
    .skyshadow-functions-dropdown:not(.no-theme-bg),
    .c2-module:not(.no-theme-bg),
    .bai-jia-hei-module:not(.no-theme-bg),
    .white-black-module:not(.no-theme-bg),
    .skyshadow-container:not(.no-theme-bg),
    .skyshadow-header:not(.no-theme-bg),
    .skyshadow-layout:not(.no-theme-bg),
    .home:not(.no-theme-bg),
    .upload-container:not(.no-theme-bg),
    .upload-section:not(.no-theme-bg),
    .select-section:not(.no-theme-bg),
    .action-section:not(.no-theme-bg),
    .bypass-container:not(.no-theme-bg)
  `
  
  // POC管理相关选择器
  const pocManagementSelectors = `
    .poc-page:not(.no-theme-bg),
    .poc-card:not(.no-theme-bg),
    .elevated-table:not(.no-theme-bg),
    .hero-title:not(.no-theme-bg),
    .request-card:not(.no-theme-bg),
    .selected-poc-list:not(.no-theme-bg)
  `
  
  // 随机生成相关选择器（调整背景，使用更具体的选择器避免冲突）
  const randomInfoSelectors = `
    .id-generator:not(.no-theme-bg),
    .id-generator .header:not(.no-theme-bg),
    .id-generator .main-content:not(.no-theme-bg),
    .id-generator .form-section:not(.no-theme-bg),
    .id-generator .form-card:not(.no-theme-bg),
    .id-generator .section-title:not(.no-theme-bg),
    .id-generator .generate-button:not(.no-theme-bg),
    .id-generator .result-section:not(.no-theme-bg),
    .id-generator .result-card:not(.no-theme-bg)
  `
  
  // 关于页面相关选择器
  const aboutSelectors = `
    .sponsor-page:not(.no-theme-bg),
    .header:not(.no-theme-bg),
    .header-left:not(.no-theme-bg),
    .header-text:not(.no-theme-bg),
    .header-right:not(.no-theme-bg),
    .qr-card:not(.no-theme-bg),
    .qr-wrapper:not(.no-theme-bg),
    .content-area:not(.no-theme-bg),
    .main-column:not(.no-theme-bg),
    .side-column:not(.no-theme-bg),
    .update-card:not(.no-theme-bg),
    .card-header:not(.no-theme-bg),
    .card-subtitle:not(.no-theme-bg),
    .list:not(.no-theme-bg),
    .scrollable:not(.no-theme-bg),
    .tall-list:not(.no-theme-bg),
    .update-item:not(.no-theme-bg),
    .timeline-dot:not(.no-theme-bg),
    .date:not(.no-theme-bg),
    .text:not(.no-theme-bg),
    .large-text:not(.no-theme-bg),
    .new-badge:not(.no-theme-bg),
    .compact-card:not(.no-theme-bg),
    .subtitle:not(.no-theme-bg),
    .thanks-grid:not(.no-theme-bg),
    .small-max:not(.no-theme-bg),
    .thanks-item:not(.no-theme-bg),
    .avatar-container:not(.no-theme-bg),
    .avatar-large:not(.no-theme-bg),
    .name:not(.no-theme-bg),
    .sponsor-grid:not(.no-theme-bg),
    .sponsor-card:not(.no-theme-bg),
    .sponsor-img-wrap:not(.no-theme-bg),
    .sponsor-img:not(.no-theme-bg),
    .glass-effect:not(.no-theme-bg),
    .chip:not(.no-theme-bg),
    .chip-primary:not(.no-theme-bg),
    .cta-group:not(.no-theme-bg),
    .btn:not(.no-theme-bg),
    .btn.primary:not(.no-theme-bg),
    .btn.ghost:not(.no-theme-bg),
    .row-inline:not(.no-theme-bg),
    .qr-image:not(.no-theme-bg)
  `
  
  // 软件中心相关选择器
  const softwareCenterSelectors = `
    .software-center:not(.no-theme-bg),
    .table-header:not(.no-theme-bg),
    .title-group:not(.no-theme-bg),
    .title-wrapper:not(.no-theme-bg),
    .title-icon:not(.no-theme-bg),
    .title:not(.no-theme-bg),
    .subtitle:not(.no-theme-bg),
    .search-box:not(.no-theme-bg),
    .search-icon:not(.no-theme-bg),
    .table-wrapper:not(.no-theme-bg),
    .name-cell:not(.no-theme-bg),
    .name-text:not(.no-theme-bg),
    .name-new-sup:not(.no-theme-bg),
    .tag-list:not(.no-theme-bg),
    .more-tag:not(.no-theme-bg),
    .table-card:not(.no-theme-bg),
    .table-footer:not(.no-theme-bg),
    .software-table:not(.no-theme-bg),
    .action-cell:not(.no-theme-bg),
    .last-updated-cell:not(.no-theme-bg),
    .version-text:not(.no-theme-bg)
  `
  
  // 应用样式
  styleElement.textContent = `
    ${moduleSelectors} {
      /* background-color: ${cardBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${dialogSelectors} {
      /* 对话框不使用磨砂效果，保持原有样式 */
      /* 确保对话框本身不透明 */
      background-color: #ffffff !important;
      backdrop-filter: none !important;
      -webkit-backdrop-filter: none !important;
    }
    /* 确保对话框内容区域可见 */
    .el-dialog__body:not(.no-theme-bg) {
      /* background-color: ${dialogBg} !important; */
      color: #303133 !important;
    }
    .el-drawer__body:not(.no-theme-bg) {
      /* background-color: ${dialogBg} !important; */
      color: #303133 !important;
    }
    /* 按钮保持原有样式，不应用主题背景 */
    /* ${buttonSelectors} {
      color: #303133 !important;
    } */
    /* 确保对话框的z-index足够高，不被遮罩层覆盖 */
    .el-dialog:not(.no-theme-bg),
    .el-drawer:not(.no-theme-bg) {
      z-index: 2000 !important;
    }
    .el-dialog__wrapper:not(.no-theme-bg),
    .el-overlay:not(.no-theme-bg) {
      z-index: 1999 !important;
      /* 确保对话框遮罩层和wrapper保持不透明，不使用半透明背景 */
      background-color: rgba(0, 0, 0, 0.5) !important;
      backdrop-filter: none !important;
      -webkit-backdrop-filter: none !important;
    }
    /* SSH对话框特殊处理 */
    .ssh-dialog:not(.no-theme-bg),
    .cmd-dialog:not(.no-theme-bg),
    .file-dialog:not(.no-theme-bg) {
      z-index: 2001 !important;
    }
    .ssh-dialog:not(.no-theme-bg) .el-dialog__wrapper,
    .cmd-dialog:not(.no-theme-bg) .el-dialog__wrapper,
    .file-dialog:not(.no-theme-bg) .el-dialog__wrapper {
      z-index: 2000 !important;
    }
    ${toolItemSelectors} {
      background-color: ${toolItemBg} !important;
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${formSelectors} {
      /* background-color: ${formBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    /* el-textarea__inner 特殊样式 */
    .el-form:not(.no-theme-bg), .el-input__wrapper:not(.no-theme-bg), .el-textarea__inner:not(.no-theme-bg), .el-upload-dragger:not(.no-theme-bg) {
      background-color: rgba(255, 255, 255, 0.5000000000000001) !important;
      /* backdrop-filter: blur(8px) !important; */
      -webkit-backdrop-filter: blur(8px) !important;
    }
    ${moduleContainerSelectors} {
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${ossSelectors} {
      /* background-color: ${cardBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${fileDumperSelectors} {
      /* background-color: ${cardBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${sshRedisSelectors} {
      /* background-color: ${cardBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${pcapSelectors} {
      /* background-color: ${cardBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${googleSelectors} {
      /* background-color: ${cardBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${passwordSelectors} {
      /* background-color: ${cardBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${bypassSelectors} {
      /* background-color: ${cardBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    .skyshadow-container:not(.no-theme-bg) {
      background-color: ${cardBg} !important;
      /* backdrop-filter: blur(${blurAmount}) !important; */
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    /* 菜单管理相关选择器 */
    .menu-manager:not(.no-theme-bg),
    .menu-section:not(.no-theme-bg),
    .menu-list:not(.no-theme-bg),
    .menu-item-container:not(.no-theme-bg),
    .menu-item:not(.no-theme-bg),
    .module-tabs-dialog:not(.no-theme-bg),
    .module-tabs-list:not(.no-theme-bg),
    .module-tab-item:not(.no-theme-bg),
    .action-buttons:not(.no-theme-bg) {
      /* background-color: ${cardBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${softwareCenterSelectors} {
      /* 软件中心相关元素不使用背景色和磨砂效果 */
    }
    ${pocManagementSelectors} {
      /* background-color: ${cardBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${randomInfoSelectors} {
      /* background-color: ${cardBg} !important; */
      backdrop-filter: blur(${blurAmount}) !important;
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
    ${aboutSelectors} {
      /* background-color: ${cardBg} !important; */
      /* backdrop-filter: blur(${blurAmount}) !important; */
      -webkit-backdrop-filter: blur(${blurAmount}) !important;
    }
  `
  
  // 如果是纯色主题，设置工具仓库和网址导航的 .nav .cate 以及标题栏为黑色
  if (config && config.type === 'color') {
    // 添加或更新专门的样式元素
    let cateStyleElement = document.getElementById('theme-cate-color-style')
    if (!cateStyleElement) {
      cateStyleElement = document.createElement('style')
      cateStyleElement.id = 'theme-cate-color-style'
      document.body.appendChild(cateStyleElement)
    }
    cateStyleElement.textContent = `
      .system .nav .cate,
      .site .nav .cate {
        color: #000000 !important;
      }
      .custom-titlebar,
      .custom-titlebar *,
      .custom-titlebar .app-title,
      .custom-titlebar .window-title,
      .custom-titlebar .titlebar-btn {
        color: #000000 !important;
        --titlebar-font-color: #000000 !important;
      }
    `
  } else {
    // 如果不是纯色主题，移除样式元素（恢复默认白色）
    const cateStyleElement = document.getElementById('theme-cate-color-style')
    if (cateStyleElement) {
      cateStyleElement.remove()
    }
  }
}


