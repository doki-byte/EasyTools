// 样式工具函数

// 获取风险等级对应的 Tag 类型
export function getTagTypeBySeverity(severity) {
  const s = (severity || '').toUpperCase()
  switch (s) {
    case 'HIGH':
      return 'danger'
    case 'MEDIUM':
      return 'warning'
    case 'LOW':
      return 'primary'
    case 'CRITICAL':
      return 'success' // CRITICAL 使用 success 类型，但会通过 CSS 类覆盖
    default:
      return 'info'
  }
}

// 获取风险等级对应的 Badge 类名
export function getBadgeClass(level) {
  const l = (level || '').toUpperCase()
  switch (l) {
    case 'CRITICAL':
      return 'risk-badge-critical'
    case 'HIGH':
      return 'risk-badge-red'
    case 'MEDIUM':
      return 'risk-badge-yellow'
    case 'LOW':
      return 'risk-badge-green'
    case 'INFO':
    default:
      return 'risk-badge-gray'
  }
}

// 高亮指纹标签
export function highlightFingerprints(fingerprint, highlightFingerprints = []) {
  if (fingerprint === '疑似蜜罐') {
    return 'warning'
  }
  if (highlightFingerprints.includes(fingerprint)) {
    return 'danger'
  }
  return 'primary'
}

// 活动类型对应的图标
export const typeToIcon = {
  primary: 'MoreFilled',
  success: 'CircleCheck',
  warning: 'Warning',
  danger: 'CloseBold',
  info: 'Notification'
}

