import { createStore } from 'vuex'
import { getItem, setItem, removeItem } from '../utils/storageManager'

export default createStore({
  state: {
    // 在这里定义你的状态
    token: getItem('Admin-Token', '') || '',
  },
  getters: {
    // 在这里定义你的 getters
    getToken: state => state.token,
  },
  mutations: {
    // 在这里定义你的 mutations
    setToken(state, token) {
      state.token = token
      setItem('Admin-Token', token)
    },
    removeToken(state) {
      state.token = ''
      removeItem('Admin-Token')
    },
  },
  actions: {
    // 在这里定义你的 actions
  },
  modules: {
    // 在这里定义你的模块
  }
})
