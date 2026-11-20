/**
 * 统一的 ElMessage 工具函数
 * 统一配置消息提示的位置、大小等样式
 */
import { ElMessage } from 'element-plus'

// 配置全局 ElMessage 默认选项
const defaultOptions = {
  duration: 3000, // 默认显示时长 3 秒
  showClose: false, // 默认不显示关闭按钮
  offset: 20, // 距离顶部的偏移量
  grouping: true, // 合并相同内容的消息
}

/**
 * 成功消息
 * @param {string|object} message - 消息内容或配置对象
 */
export const messageSuccess = (message) => {
  if (typeof message === 'string') {
    return ElMessage.success({
      message,
      ...defaultOptions,
    })
  }
  return ElMessage.success({
    ...defaultOptions,
    ...message,
  })
}

/**
 * 错误消息
 * @param {string|object} message - 消息内容或配置对象
 */
export const messageError = (message) => {
  if (typeof message === 'string') {
    return ElMessage.error({
      message,
      ...defaultOptions,
    })
  }
  return ElMessage.error({
    ...defaultOptions,
    ...message,
  })
}

/**
 * 警告消息
 * @param {string|object} message - 消息内容或配置对象
 */
export const messageWarning = (message) => {
  if (typeof message === 'string') {
    return ElMessage.warning({
      message,
      ...defaultOptions,
    })
  }
  return ElMessage.warning({
    ...defaultOptions,
    ...message,
  })
}

/**
 * 信息消息
 * @param {string|object} message - 消息内容或配置对象
 */
export const messageInfo = (message) => {
  if (typeof message === 'string') {
    return ElMessage.info({
      message,
      ...defaultOptions,
    })
  }
  return ElMessage.info({
    ...defaultOptions,
    ...message,
  })
}

/**
 * 默认导出，提供类似 this.$message 的 API
 */
export default {
  success: messageSuccess,
  error: messageError,
  warning: messageWarning,
  info: messageInfo,
}
