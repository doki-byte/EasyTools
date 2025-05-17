import axios from 'axios'
import { getToken } from './token'
import { Notification } from 'element-plus'
import { useRouter } from 'vue-router'

// 创建 axios 实例
const service = axios.create({
  baseURL: process.env.VUE_APP_API_URL, // 基础 URL + 请求 URL
  timeout: 5000 // 请求超时时间
})

// 请求拦截器
service.interceptors.request.use(
  config => {
    // 请求发送之前做一些处理
    config.headers['token'] = getToken()
    return config
  },
  error => {
    // 请求错误时做一些处理
    console.log(error) // 调试
    return Promise.reject(error)
  }
)

// 响应拦截器
service.interceptors.response.use(
  /**
   * 如果你想获取 http 信息，如 headers 或 status
   * 可以直接返回 response => response
   */
  response => {
    const res = response.data

    // 自定义错误码不为 200 时，视为错误
    if (res.code !== 200) {
      Notification({
        type: 'error',
        title: '温馨提示',
        message: res['message']
      })

      // 错误码 501: 非法 token；502: 其他客户端登录；503: token 过期
      if (res.code === 501 || res.code === 502 || res.code === 503) {
        // 需要登录，跳转到登录页
        const router = useRouter() // 使用 Vue Router 4，必须在组件内使用
        router.push({
          name: 'login'
        })
      }
      return Promise.reject(new Error(res.message || 'Error'))
    }
    return res
  },
  error => {
    console.log('err' + error) // 调试
    return Promise.reject(error)
  }
)

export default service
