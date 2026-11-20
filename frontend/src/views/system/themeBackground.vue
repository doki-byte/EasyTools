<template>
  <div class="theme-background-container">
    <el-card class="config-card">
      <template #header>
        <div class="card-header">
          <el-icon><Picture /></el-icon>
          <span>主题背景设置</span>
        </div>
      </template>

      <!-- 默认主题快速选择 -->
      <div class="default-themes-section">
        <div class="section-title">默认主题</div>
        <div class="default-themes-list">
          <div
            v-for="(theme, index) in defaultThemes"
            :key="index"
            class="theme-card"
            :class="{ active: isThemeActive(theme) }"
            @click="applyDefaultTheme(theme)"
          >
            <div class="theme-preview" :style="getThemePreviewStyle(theme)">
              <div class="theme-preview-content">
                <el-icon><View /></el-icon>
              </div>
            </div>
            <div class="theme-info">
              <div class="theme-name">{{ theme.name }}</div>
              <div class="theme-desc">{{ theme.description }}</div>
            </div>
            <div class="theme-check" v-if="isThemeActive(theme)">
              <el-icon><Check /></el-icon>
            </div>
          </div>
        </div>
      </div>

      <el-divider />

      <el-form :model="form" label-width="140px" label-position="left">
        <el-form-item label="背景类型:">
          <el-radio-group v-model="form.type" @change="handleTypeChange">
            <el-radio label="color">纯色背景</el-radio>
            <el-radio label="image">图片/视频背景</el-radio>
          </el-radio-group>
        </el-form-item>

        <!-- 纯色背景设置 -->
        <template v-if="form.type === 'color'">
          <el-form-item label="背景颜色:">
            <div class="color-picker-wrapper">
              <el-color-picker v-model="form.color" @change="handleColorChange" />
              <el-input
                v-model="form.color"
                placeholder="#ffffff"
                style="width: 150px; margin-left: 10px;"
                @change="handleColorInputChange"
              >
                <template #prefix>
                  <el-icon><Brush /></el-icon>
                </template>
              </el-input>
            </div>
          </el-form-item>
        </template>

        <!-- 图片/视频背景设置 -->
        <template v-if="form.type === 'image'">
          <el-form-item label="选择背景:">
            <div class="image-selector-wrapper">
              <el-select 
                v-model="form.imageUrl" 
                placeholder="请选择背景图片" 
                style="width: 100%; max-width: 500px;"
                popper-class="image-selector-dropdown"
                @change="handleImageUrlChange"
              >
                <el-option
                  v-for="(url, index) in form.imageUrls"
                  :key="index"
                  :label="getImageLabel(url)"
                  :value="url"
                >
                  <div class="image-option-item">
                    <span class="image-option-label">{{ getImageLabel(url) }}</span>
                    <el-button
                      v-if="form.imageUrls.length > 1"
                      type="danger"
                      :icon="Delete"
                      size="small"
                      text
                      class="image-option-delete"
                      @click.stop="removeImageFromList(url, index)"
                    />
                  </div>
                </el-option>
              </el-select>
            </div>
          </el-form-item>
          <el-form-item label="背景文件:">
            <div class="file-upload-wrapper">
              <el-button type="primary" @click="selectMediaFile">
                <el-icon><Plus /></el-icon>
                添加图片/视频
              </el-button>
              <div v-if="form.imageUrl" class="file-preview-container">
                <div v-if="isVideoFile(form.imageUrl)" class="video-preview">
                  <video v-if="previewVideoUrl" :src="previewVideoUrl" class="preview-video" controls />
                  <div v-else class="loading-placeholder">加载中...</div>
                </div>
                <img v-else-if="previewFileUrl" :src="previewFileUrl" class="preview-image" />
                <div v-else class="loading-placeholder">加载中...</div>
                <div class="file-path-info">
                  <el-text type="info" size="small">{{ form.imageUrl }}</el-text>
                </div>
                <div class="file-actions">
                  <el-button 
                    size="small" 
                    type="danger" 
                    @click="removeMediaFile"
                    :disabled="form.imageUrls.length <= 1"
                  >
                    <el-icon><Delete /></el-icon>
                    移除文件
                  </el-button>
                </div>
              </div>
            </div>
            <div class="file-tips">
              <el-text type="info" size="small">
                支持 JPG、PNG、GIF 图片格式和 MP4、WebM 视频格式。可以添加多个背景图片并选择使用。
              </el-text>
            </div>
          </el-form-item>
        </template>

        <el-form-item label="背景透明度:">
          <div class="opacity-control">
            <el-slider
              v-model="form.opacity"
              :min="0"
              :max="100"
              :step="1"
              show-input
              :show-input-controls="false"
              @change="handleOpacityChange"
            />
            <span class="opacity-value">{{ form.opacity }}%</span>
          </div>
        </el-form-item>

        <el-divider />

        <el-form-item label="字体颜色:">
          <div class="color-picker-wrapper">
            <el-color-picker v-model="form.fontColor" @change="handleFontColorChange" />
            <el-input
              v-model="form.fontColor"
              placeholder="#ffffff"
              style="width: 150px; margin-left: 10px;"
              @change="handleFontColorInputChange"
            >
              <template #prefix>
                <el-icon><Brush /></el-icon>
              </template>
            </el-input>
          </div>
          <div class="font-color-tips">
            <el-text type="info" size="small">
              影响顶部标题栏、侧边栏、各模块顶层标签的字色（含全局搜索相关提示）
            </el-text>
          </div>
        </el-form-item>

        <el-form-item label="加粗程度:">
          <div class="opacity-control">
            <el-slider
              v-model="form.fontWeightRatio"
              :min="0"
              :max="200"
              :step="10"
              show-input
              :show-input-controls="false"
              @change="handleFontWeightRatioChange"
            />
            <span class="opacity-value">{{ form.fontWeightRatio }}%</span>
          </div>
          <div class="font-color-tips">
            <el-text type="info" size="small">
              控制标签页字体的加粗程度，100%为默认值
            </el-text>
          </div>
        </el-form-item>

        <el-form-item label="阴影程度:">
          <div class="opacity-control">
            <el-slider
              v-model="form.shadowRatio"
              :min="0"
              :max="200"
              :step="10"
              show-input
              :show-input-controls="false"
              @change="handleShadowRatioChange"
            />
            <span class="opacity-value">{{ form.shadowRatio }}%</span>
          </div>
          <div class="font-color-tips">
            <el-text type="info" size="small">
              控制标签页阴影的强度，100%为默认值
            </el-text>
          </div>
        </el-form-item>

        <el-form-item label="全局搜索快捷键:">
          <div class="global-search-hotkey-row">
            <el-input
              v-model="form.globalSearchHotkey"
              readonly
              placeholder="点击输入框后按下组合键"
              style="width: 300px;"
              class="hotkey-readonly-input"
              @keydown.prevent.stop="onGlobalSearchHotkeyKeydown"
            />
            <el-button link type="primary" @click="resetGlobalSearchHotkeyOnly">
              恢复默认 Ctrl+K
            </el-button>
          </div>
          <div class="font-color-tips">
            <el-text type="info" size="small">
              打开标题栏旁的「全局搜索」面板（与 Ctrl+K 行为相同）。建议在组合中包含 Ctrl/Cmd、Alt、Shift 之一以免与表单输入冲突。
            </el-text>
          </div>
        </el-form-item>

        <el-form-item label="预览:">
          <div class="preview-box">
            <video 
              v-if="form.type === 'image' && form.imageUrl && isVideoFile(form.imageUrl) && previewVideoUrl" 
              :src="previewVideoUrl" 
              class="preview-video" 
              autoplay 
              loop 
              muted 
              playsinline
            />
            <div v-else :style="previewStyle" class="preview-background">
              <div class="preview-content">
                <el-icon><View /></el-icon>
                <span>背景预览</span>
              </div>
            </div>
          </div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="saveConfig" :loading="saving">
            <el-icon><Check /></el-icon>
            保存设置
          </el-button>
          <el-button @click="resetConfig">
            <el-icon><RefreshLeft /></el-icon>
            重置为默认
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Picture, Brush, Plus, Delete, View, Check, RefreshLeft } from '@element-plus/icons-vue'
import { GetOpenFilePath, ReadFileAsDataURL } from '../../../wailsjs/go/system/System'
import { accelFromEvent } from '@/utils/hotkeyUtils'
import {
  DEFAULT_GLOBAL_SEARCH_HOTKEY,
  validateGlobalSearchHotkey,
  getGlobalSearchHotkeyFromConfig,
} from '@/utils/globalSearchHotkey'

const STORAGE_KEY = 'easytools:theme:background'

// 默认配置
const defaultConfig = {
  type: 'color', // 'color' | 'image'
  color: '#f5f7fa',
  imageUrls: ['./assets/system/EasyToolsBackGround.png', './assets/system/kali-linux.png'],
  imageUrl: './assets/system/kali-linux.png',
  opacity: 100,
  fontColor: '#ffffff',
  fontWeightRatio: 100, // 加粗程度比例 (0-200, 100为默认)
  shadowRatio: 100, // 阴影程度比例 (0-200, 100为默认)
  globalSearchHotkey: DEFAULT_GLOBAL_SEARCH_HOTKEY,
}

// 十六进制转 RGB
const hexToRgb = (hex) => {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex)
  return result ? {
    r: parseInt(result[1], 16),
    g: parseInt(result[2], 16),
    b: parseInt(result[3], 16)
  } : null
}

// 获取当前保存的配置
const getCurrentSavedConfig = () => {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) {
      const config = JSON.parse(saved)
      return {
        type: config.type || defaultConfig.type,
        color: config.color || defaultConfig.color,
        imageUrls: config.imageUrls || (config.imageUrl ? [config.imageUrl] : defaultConfig.imageUrls),
        imageUrl: config.imageUrl || defaultConfig.imageUrl,
        opacity: config.opacity !== undefined ? config.opacity : defaultConfig.opacity,
        fontColor: config.fontColor || defaultConfig.fontColor,
        fontWeightRatio: config.fontWeightRatio !== undefined ? config.fontWeightRatio : defaultConfig.fontWeightRatio,
        shadowRatio: config.shadowRatio !== undefined ? config.shadowRatio : defaultConfig.shadowRatio,
        globalSearchHotkey: getGlobalSearchHotkeyFromConfig(config),
      }
    }
  } catch (e) {
    console.error('读取当前配置失败:', e)
  }
  // 如果没有保存的配置，返回默认配置
  return { ...defaultConfig }
}

// 默认主题列表
const defaultThemes = computed(() => {
  // 获取当前保存的配置作为第一个主题
  const currentConfig = getCurrentSavedConfig()
  
  return [
    {
      id: 'current',
      name: '当前配置',
      description: '使用您已保存的配置',
      ...currentConfig
    },
    {
      id: 'pure-color',
      name: '纯色主题',
      description: '简洁的纯色背景',
      type: 'color',
      color: '#f5f7fa',
      imageUrls: [],
      imageUrl: '',
      opacity: 100,
      fontColor: '#333333',
      fontWeightRatio: 100,
      shadowRatio: 100,
      globalSearchHotkey: DEFAULT_GLOBAL_SEARCH_HOTKEY,
    },
    {
      id: 'image',
      name: '图片/视频主题',
      description: '使用图片或视频作为背景',
      type: 'image',
      color: '#f5f7fa',
      imageUrls: ['./assets/system/EasyToolsBackGround.png','./assets/system/kali-linux.png'],
      imageUrl: './assets/system/kali-linux.png',
      opacity: 85,
      fontColor: '#ffffff',
      fontWeightRatio: 100,
      shadowRatio: 100,
      globalSearchHotkey: DEFAULT_GLOBAL_SEARCH_HOTKEY,
    }
  ]
})

// 判断主题是否激活（与当前表单配置一致）
const isThemeActive = (theme) => {
  if (theme.id === 'current') {
    // 当前配置主题始终显示为激活状态（如果当前有保存的配置）
    const saved = localStorage.getItem(STORAGE_KEY)
    return !!saved
  }
  
  return (
    form.type === theme.type &&
    form.color === theme.color &&
    form.imageUrl === theme.imageUrl &&
    form.opacity === theme.opacity &&
    form.fontColor === theme.fontColor &&
    form.fontWeightRatio === theme.fontWeightRatio &&
    form.shadowRatio === theme.shadowRatio &&
    (theme.globalSearchHotkey ?? defaultConfig.globalSearchHotkey) === form.globalSearchHotkey
  )
}

// 获取主题预览样式（同步版本，用于主题卡片预览）
const getThemePreviewStyle = (theme) => {
  const opacity = (theme.opacity || 100) / 100
  let background = ''
  
  if (theme.type === 'color') {
    const rgb = hexToRgb(theme.color || '#f5f7fa')
    if (rgb) {
      background = `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${opacity})`
    } else {
      background = theme.color || '#f5f7fa'
    }
  } else if (theme.type === 'image' && theme.imageUrl) {
    // 对于主题预览，如果是 data URL、http URL 或 public 资源路径（/ 或 ./assets/ 开头），直接使用
    let imageUrl = theme.imageUrl
    if (imageUrl.startsWith('./assets/')) {
      imageUrl = imageUrl.replace(/^\.\//, '/')
    }
    if (imageUrl.startsWith('data:') || imageUrl.startsWith('http://') || imageUrl.startsWith('https://') || imageUrl.startsWith('/')) {
      background = `url(${imageUrl})`
    } else if (fileUrlCache.value[theme.imageUrl]) {
      background = `url(${fileUrlCache.value[theme.imageUrl]})`
    } else {
      // 异步加载（不影响当前显示）
      getFileUrl(theme.imageUrl).catch(() => {})
      background = '#f5f7fa'
    }
  } else {
    background = '#f5f7fa'
  }
  
  return {
    background,
    backgroundSize: theme.type === 'image' ? 'cover' : 'auto',
    backgroundPosition: 'center',
    backgroundRepeat: 'no-repeat'
  }
}

// 应用默认主题
const applyDefaultTheme = async (theme) => {
  // 如果是当前配置主题，加载已保存的配置并应用
  if (theme.id === 'current') {
    await loadConfig()
    // 触发保存事件，确保配置被应用
    saveConfig()
    ElMessage.success('已应用当前配置')
    return
  }
  
  // 兼容旧版本：如果主题没有 imageUrls，使用 imageUrl 创建数组
  const imageUrls = theme.imageUrls || (theme.imageUrl ? [theme.imageUrl] : defaultConfig.imageUrls)
  const imageUrl = theme.imageUrl || (imageUrls.length > 0 ? imageUrls[0] : defaultConfig.imageUrl)
  
  // 应用主题配置到表单
  Object.assign(form, {
    type: theme.type,
    color: theme.color,
    imageUrls: imageUrls,
    imageUrl: imageUrl,
    opacity: theme.opacity,
    fontColor: theme.fontColor,
    fontWeightRatio: theme.fontWeightRatio,
    shadowRatio: theme.shadowRatio,
    globalSearchHotkey:
      theme.globalSearchHotkey != null ? theme.globalSearchHotkey : defaultConfig.globalSearchHotkey,
  })
  
  // 如果是图片主题，清除旧的缓存并加载预览
  if (theme.type === 'image' && imageUrl) {
    // 清除旧的缓存，确保重新加载
    if (fileUrlCache.value[imageUrl]) {
      delete fileUrlCache.value[imageUrl]
    }
    // 加载预览 URL
    loadPreviewUrl()
  }
  
  // 自动保存并应用
  saveConfig()
  ElMessage.success(`已应用${theme.name}`)
}

// 表单数据
const form = reactive({
  type: 'color',
  color: '#f5f7fa',
  imageUrls: ['./assets/system/EasyToolsBackGround.png','./assets/system/kali-linux.png'],
  imageUrl: './assets/system/kali-linux.png',
  opacity: 100,
  fontColor: '#ffffff',
  fontWeightRatio: 100,
  shadowRatio: 100,
  globalSearchHotkey: DEFAULT_GLOBAL_SEARCH_HOTKEY,
})

const saving = ref(false)

// 判断是否为视频文件
const isVideoFile = (filePath) => {
  if (!filePath) return false
  const videoExtensions = ['.mp4', '.webm', '.ogg', '.mov', '.avi', '.mkv']
  const lowerPath = filePath.toLowerCase()
  return videoExtensions.some(ext => lowerPath.endsWith(ext))
}

// 文件 URL 缓存（用于存储转换后的 data URL）
const fileUrlCache = ref({})

// 获取文件 URL（将文件路径转换为 data URL）
const getFileUrl = async (filePath) => {
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
    if (!fileUrlCache.value[filePath]) {
      fileUrlCache.value[filePath] = normalizedPath
    }
    return normalizedPath
  }
  // 检查缓存
  if (fileUrlCache.value[filePath]) {
    return fileUrlCache.value[filePath]
  }
  // 如果是文件路径，通过后端 API 转换为 data URL
  try {
    const dataURL = await ReadFileAsDataURL(filePath)
    if (dataURL) {
      fileUrlCache.value[filePath] = dataURL
      return dataURL
    }
    return ''
  } catch (e) {
    console.error('读取文件失败:', e)
    return ''
  }
}

// 预览样式（使用缓存或同步检查）
const previewStyle = computed(() => {
  const opacity = form.opacity / 100
  let background = ''

  if (form.type === 'color') {
    const rgb = hexToRgb(form.color)
    if (rgb) {
      background = `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${opacity})`
    } else {
      background = form.color
    }
  } else if (form.type === 'image' && form.imageUrl) {
    // 检查是否是 data URL、http URL 或 public 资源路径（/ 或 ./assets/ 开头）
    let imageUrl = form.imageUrl
    if (imageUrl.startsWith('./assets/')) {
      imageUrl = imageUrl.replace(/^\.\//, '/')
    }
    if (imageUrl.startsWith('data:') || imageUrl.startsWith('http://') || imageUrl.startsWith('https://') || imageUrl.startsWith('/')) {
      // 对于 public 资源路径，直接使用，不需要转换为 data URL
      background = `linear-gradient(rgba(0, 0, 0, ${1 - opacity}), rgba(0, 0, 0, ${1 - opacity})), url(${imageUrl})`
    } else if (fileUrlCache.value[form.imageUrl]) {
      background = `linear-gradient(rgba(0, 0, 0, ${1 - opacity}), rgba(0, 0, 0, ${1 - opacity})), url(${fileUrlCache.value[form.imageUrl]})`
    } else {
      // 如果还没有缓存，先显示占位背景，异步加载后会自动更新
      background = '#f5f7fa'
      // 异步加载 data URL（仅对非 public 资源路径）
      getFileUrl(form.imageUrl).then(() => {
        // 触发响应式更新（通过访问 fileUrlCache 触发）
        fileUrlCache.value = { ...fileUrlCache.value }
      })
    }
  } else {
    background = '#f5f7fa'
  }

  return {
    background,
    backgroundSize: form.type === 'image' ? 'cover' : 'auto',
    backgroundPosition: 'center',
    backgroundRepeat: 'no-repeat'
  }
})

// 用于模板的文件 URL（支持异步加载）
const previewFileUrl = ref('')
const previewVideoUrl = ref('')

// 监听文件路径变化，异步加载 data URL
const loadPreviewUrl = async () => {
  if (form.type === 'image' && form.imageUrl) {
    const url = await getFileUrl(form.imageUrl)
    if (isVideoFile(form.imageUrl)) {
      previewVideoUrl.value = url
      previewFileUrl.value = ''
    } else {
      previewFileUrl.value = url
      previewVideoUrl.value = ''
    }
  } else {
    previewFileUrl.value = ''
    previewVideoUrl.value = ''
  }
}

// 监听 imageUrl 变化
watch(() => form.imageUrl, () => {
  loadPreviewUrl()
}, { immediate: true })

// 加载配置
const loadConfig = async () => {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) {
      const config = JSON.parse(saved)
      // 兼容旧版本：如果存在 imageUrl 但没有 imageUrls，创建数组
      let imageUrls = config.imageUrls || (config.imageUrl ? [config.imageUrl] : defaultConfig.imageUrls)
      let imageUrl = config.imageUrl || (imageUrls.length > 0 ? imageUrls[0] : defaultConfig.imageUrl)
      
      // 如果是图片类型，验证 imageUrl 是否可用
      if (config.type === 'image' && imageUrl) {
        const fileUrl = await getFileUrl(imageUrl)
        if (!fileUrl) {
          // 当前 imageUrl 不可用，尝试从 imageUrls 中找到可用的
          let foundValid = false
          for (const url of imageUrls) {
            if (url && url !== imageUrl) {
              const testUrl = await getFileUrl(url)
              if (testUrl) {
                imageUrl = url
                foundValid = true
                console.warn(`背景文件 ${config.imageUrl} 不可用，已切换到 ${url}`)
                break
              }
            }
          }
          
          // 如果所有背景都不可用，回退到默认配置
          if (!foundValid) {
            console.warn('所有背景文件都不可用，已回退到默认配置')
            imageUrls = [...defaultConfig.imageUrls]
            imageUrl = defaultConfig.imageUrl
          }
        }
      }
      
      Object.assign(form, {
        type: config.type || defaultConfig.type,
        color: config.color || defaultConfig.color,
        imageUrls: imageUrls,
        imageUrl: imageUrl,
        opacity: config.opacity !== undefined ? config.opacity : defaultConfig.opacity,
        fontColor: config.fontColor || defaultConfig.fontColor,
        fontWeightRatio: config.fontWeightRatio !== undefined ? config.fontWeightRatio : defaultConfig.fontWeightRatio,
        shadowRatio: config.shadowRatio !== undefined ? config.shadowRatio : defaultConfig.shadowRatio,
        globalSearchHotkey: getGlobalSearchHotkeyFromConfig(config),
      })
      
      // 如果配置被修复，保存修复后的配置
      if (config.type === 'image' && (imageUrls !== config.imageUrls || imageUrl !== config.imageUrl)) {
        const fixedConfig = {
          ...config,
          imageUrls: imageUrls,
          imageUrl: imageUrl
        }
        localStorage.setItem(STORAGE_KEY, JSON.stringify(fixedConfig))
      }
      
      // 加载配置后，触发预览 URL 加载
      loadPreviewUrl()
    }
  } catch (e) {
    console.error('加载主题配置失败:', e)
  }
}

// 捕获按键设置「全局搜索」快捷键（与原生程序风格一致）
const onGlobalSearchHotkeyKeydown = (e) => {
  if (e.key === 'Escape') {
    e.target?.blur?.()
    return
  }
  const acc = accelFromEvent(e)
  if (!acc) return
  const gv = validateGlobalSearchHotkey(acc)
  if (gv.ok) {
    form.globalSearchHotkey = gv.normalized
    ElMessage.success(`快捷键已设为：${gv.normalized}（点击保存设置生效）`)
  } else {
    ElMessage.warning(gv.message)
    form.globalSearchHotkey = gv.normalized
  }
}

const resetGlobalSearchHotkeyOnly = () => {
  form.globalSearchHotkey = DEFAULT_GLOBAL_SEARCH_HOTKEY
  ElMessage.success('已恢复为 Ctrl+K，请点击「保存设置」生效')
}

// 保存配置
const saveConfig = () => {
  saving.value = true
  try {
    const gv = validateGlobalSearchHotkey(form.globalSearchHotkey)
    const globalSearchHotkey = gv.ok ? gv.normalized : DEFAULT_GLOBAL_SEARCH_HOTKEY
    if (!gv.ok) {
      form.globalSearchHotkey = globalSearchHotkey
      ElMessage.warning(gv.message || '全局搜索快捷键无效，已恢复默认 Ctrl+K')
    }

    const config = {
      type: form.type,
      color: form.color,
      imageUrls: form.imageUrls || [],
      imageUrl: form.imageUrl,
      opacity: form.opacity,
      fontColor: form.fontColor,
      fontWeightRatio: form.fontWeightRatio,
      shadowRatio: form.shadowRatio,
      globalSearchHotkey,
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(config))
    
    // 触发全局事件，通知其他组件更新背景和字体颜色
    window.dispatchEvent(new CustomEvent('theme-background-changed', { detail: config }))
    
    // 强制刷新背景应用
    setTimeout(() => {
      window.dispatchEvent(new CustomEvent('theme-background-changed', { detail: config }))
    }, 300)
    
    ElMessage.success('主题背景设置已保存')
  } catch (e) {
    console.error('保存主题配置失败:', e)
    ElMessage.error('保存失败，请重试')
  } finally {
    saving.value = false
  }
}

// 重置配置（根据当前类型重置为对应默认主题）
const resetConfig = () => {
  try {
    if (form.type === 'color') {
      // 重置为默认纯色主题
      const defaultColorTheme = {
        type: 'color',
        color: '#f5f7fa',
        imageUrls: [],
        imageUrl: '',
        opacity: 100,
        fontColor: '#333333',
        fontWeightRatio: 100,
        shadowRatio: 100,
        globalSearchHotkey: DEFAULT_GLOBAL_SEARCH_HOTKEY,
      }
      
      // 更新表单
      Object.assign(form, defaultColorTheme)
      
      // 清除预览 URL
      previewFileUrl.value = ''
      previewVideoUrl.value = ''
      
      // 保存并应用
      saveConfig()
      ElMessage.success('已重置为默认纯色主题')
    } else if (form.type === 'image') {
      // 重置为默认图片/视频主题（使用默认 imageUrls 数组）
      const defaultImageUrls = [...defaultConfig.imageUrls] // 使用默认配置中的 imageUrls
      const defaultImageUrl = defaultImageUrls[0] || defaultConfig.imageUrl
      
      const defaultImageTheme = {
        type: 'image',
        color: '#f5f7fa',
        imageUrls: defaultImageUrls,
        imageUrl: defaultImageUrl,
        opacity: 85,
        fontColor: '#ffffff',
        fontWeightRatio: 100,
        shadowRatio: 100,
        globalSearchHotkey: DEFAULT_GLOBAL_SEARCH_HOTKEY,
      }
      
      // 更新表单
      Object.assign(form, defaultImageTheme)
      
      // 清除旧的缓存，确保重新加载
      defaultImageUrls.forEach(url => {
        if (fileUrlCache.value[url]) {
          delete fileUrlCache.value[url]
        }
      })
      
      // 加载预览 URL
      loadPreviewUrl()
      
      // 保存并应用
      saveConfig()
      ElMessage.success('已重置为默认图片/视频主题')
    }
  } catch (e) {
    console.error('重置主题配置失败:', e)
    ElMessage.error('重置失败，请重试')
  }
}

// 处理类型变化
const handleTypeChange = () => {
  // 切换类型时，如果切换到图片但还没有图片，保持当前设置
  if (form.type === 'image' && !form.imageUrl) {
    // 可以保持当前状态
  }
}

// 处理颜色变化
const handleColorChange = () => {
  // 颜色选择器变化时自动更新
}

// 处理颜色输入变化
const handleColorInputChange = (value) => {
  // 验证颜色格式
  if (/^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/.test(value)) {
    form.color = value
  }
}

// 处理字体颜色变化
const handleFontColorChange = () => {
  // 字体颜色选择器变化时自动更新
}

// 处理字体颜色输入变化
const handleFontColorInputChange = (value) => {
  // 验证颜色格式
  if (/^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$/.test(value)) {
    form.fontColor = value
  }
}

// 处理透明度变化
const handleOpacityChange = () => {
  // 透明度变化时自动更新
}

// 处理加粗程度变化
const handleFontWeightRatioChange = () => {
  // 加粗程度变化时自动更新
}

// 处理阴影程度变化
const handleShadowRatioChange = () => {
  // 阴影程度变化时自动更新
}

// 选择媒体文件（图片或视频）
const selectMediaFile = async () => {
  try {
    const filePath = await GetOpenFilePath()
    if (filePath) {
      // 验证文件类型
      const imageExtensions = ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp']
      const videoExtensions = ['.mp4', '.webm', '.ogg', '.mov', '.avi', '.mkv']
      const lowerPath = filePath.toLowerCase()
      const isValidImage = imageExtensions.some(ext => lowerPath.endsWith(ext))
      const isValidVideo = videoExtensions.some(ext => lowerPath.endsWith(ext))
      
      if (!isValidImage && !isValidVideo) {
        ElMessage.error('请选择有效的图片或视频文件')
        return
      }
      
      // 检查是否已存在
      if (form.imageUrls.includes(filePath)) {
        ElMessage.warning('该文件已存在')
        // 切换到该文件
        form.imageUrl = filePath
        await loadPreviewUrl()
        return
      }
      
      // 添加到数组
      form.imageUrls.push(filePath)
      // 设置为当前选中的图片
      form.imageUrl = filePath
      // 清除旧的缓存
      delete fileUrlCache.value[filePath]
      // 异步加载 data URL
      await loadPreviewUrl()
      ElMessage.success('文件已添加')
    }
  } catch (e) {
    console.error('选择文件失败:', e)
    ElMessage.error('选择文件失败，请重试')
  }
}

// 获取图片标签（用于下拉选择器显示）
const getImageLabel = (url) => {
  if (!url) return '未知'
  // 如果是相对路径，提取文件名
  if (url.includes('/')) {
    const parts = url.split('/')
    return parts[parts.length - 1]
  }
  // 如果是文件路径，提取文件名
  if (url.includes('\\')) {
    const parts = url.split('\\')
    return parts[parts.length - 1]
  }
  return url
}

// 处理图片URL变化
const handleImageUrlChange = () => {
  // 当选择的图片URL变化时，重新加载预览
  loadPreviewUrl()
}

// 从列表中移除指定图片
const removeImageFromList = (url, index) => {
  if (form.imageUrls.length <= 1) {
    ElMessage.warning('至少需要保留一个背景图片')
    return
  }
  
  const isCurrentSelected = form.imageUrl === url
  
  // 从数组中移除
  form.imageUrls.splice(index, 1)
  
  // 如果移除的是当前选中的，切换到第一个
  if (isCurrentSelected) {
    form.imageUrl = form.imageUrls[0] || ''
    loadPreviewUrl()
  }
  
  // 清除缓存
  if (url && fileUrlCache.value[url]) {
    delete fileUrlCache.value[url]
  }
  
  ElMessage.success('文件已移除')
}

// 移除媒体文件（移除当前选中的文件）
const removeMediaFile = () => {
  const currentUrl = form.imageUrl
  if (!currentUrl || form.imageUrls.length <= 1) {
    ElMessage.warning('至少需要保留一个背景图片')
    return
  }
  
  // 从数组中移除
  const index = form.imageUrls.indexOf(currentUrl)
  if (index > -1) {
    form.imageUrls.splice(index, 1)
    
    // 如果移除的是当前选中的，切换到第一个
    if (form.imageUrl === currentUrl) {
      form.imageUrl = form.imageUrls[0] || ''
      loadPreviewUrl()
    }
  }
  
  // 清除缓存
  if (currentUrl && fileUrlCache.value[currentUrl]) {
    delete fileUrlCache.value[currentUrl]
  }
  
  ElMessage.success('文件已移除')
}

onMounted(async () => {
  await loadConfig()
})
</script>

<style scoped lang="scss">
.theme-background-container {
  padding: 20px;
}

.config-card {
  max-width: 800px;
  margin: 0 auto;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 16px;
}

.color-picker-wrapper {
  display: flex;
  align-items: center;
}

.file-upload-wrapper {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.file-preview-container {
  margin-top: 10px;
  border: 2px solid #e4e7ed;
  border-radius: 8px;
  padding: 10px;
  background: #fafafa;
}

.preview-image {
  width: 100%;
  max-width: 500px;
  max-height: 300px;
  object-fit: contain;
  display: block;
  border-radius: 4px;
}

.preview-video {
  width: 100%;
  max-width: 500px;
  max-height: 300px;
  object-fit: contain;
  display: block;
  border-radius: 4px;
}

.video-preview {
  margin-bottom: 10px;
}

.file-path-info {
  margin-top: 10px;
  padding: 8px;
  background: #ffffff;
  border-radius: 4px;
  word-break: break-all;
}

.file-actions {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}

.file-tips {
  margin-top: 8px;
}

.image-selector-wrapper {
  width: 100%;
  margin-bottom: 16px;
}

.image-option-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 8px;
}

.image-option-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.image-option-delete {
  flex-shrink: 0;
  padding: 4px;
  margin-left: auto;
  
  &:hover {
    color: #f56c6c;
  }
}

.font-color-tips {
  margin-top: 8px;
}

.opacity-tips {
  margin-top: 8px;
}

.opacity-control {
  display: flex;
  align-items: center;
  gap: 15px;
  width: 100%;
  max-width: 400px;

  :deep(.el-slider) {
    flex: 1;
  }
}

.opacity-value {
  min-width: 50px;
  text-align: right;
  color: #606266;
  font-size: 14px;
}

.preview-box {
  width: 100%;
  height: 200px;
  border: 2px solid #e4e7ed;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  position: relative;
}

.preview-background {
  width: 100%;
  height: 100%;
}

.preview-box .preview-video {
  width: 100%;
  height: 100%;
  max-width: none;
  max-height: none;
  object-fit: cover;
}

.preview-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  color: #606266;
  font-size: 14px;
  z-index: 1;
  background: rgba(255, 255, 255, 0.8);
  padding: 20px;
  border-radius: 8px;
}

.loading-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: #909399;
  font-size: 14px;
}

:deep(.el-form-item) {
  margin-bottom: 24px;
}

:deep(.el-form-item__label) {
  font-weight: 500;
}

.default-themes-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
}

.default-themes-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.theme-card {
  position: relative;
  border: 2px solid #e4e7ed;
  border-radius: 12px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.3s;
  background: #ffffff;
  overflow: hidden;

  &:hover {
    border-color: #409eff;
    box-shadow: 0 4px 12px rgba(64, 158, 255, 0.15);
    transform: translateY(-2px);
  }

  &.active {
    border-color: #409eff;
    background: #ecf5ff;
    box-shadow: 0 4px 12px rgba(64, 158, 255, 0.2);
  }
}

.theme-preview {
  width: 100%;
  height: 120px;
  border-radius: 8px;
  margin-bottom: 12px;
  overflow: hidden;
  position: relative;
  border: 1px solid #e4e7ed;
}

.theme-preview-content {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #606266;
  font-size: 24px;
  z-index: 1;
  background: rgba(255, 255, 255, 0.6);
}

.theme-info {
  text-align: center;
}

.theme-name {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
}

.theme-desc {
  font-size: 12px;
  color: #909399;
}

.theme-check {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 24px;
  height: 24px;
  background: #409eff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  font-size: 14px;
}
</style>

<style>
/* 下拉框选项样式 */
.image-selector-dropdown .el-select-dropdown__item {
  padding-right: 8px;
}

.image-selector-dropdown .image-option-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 8px;
}

.image-selector-dropdown .image-option-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.image-selector-dropdown .image-option-delete {
  flex-shrink: 0;
  padding: 4px;
  margin-left: auto;
  opacity: 0.7;
  transition: opacity 0.2s;
}

.image-selector-dropdown .image-option-delete:hover {
  opacity: 1;
  color: #f56c6c;
}

.image-selector-dropdown .el-select-dropdown__item:hover .image-option-delete {
  opacity: 1;
}

.global-search-hotkey-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.hotkey-readonly-input :deep(.el-input__inner) {
  cursor: pointer;
}
</style>


