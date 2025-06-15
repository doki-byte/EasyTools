import { createStore } from 'vuex'

export default createStore({
  state: {
    // 在这里定义你的状态
    token: localStorage.getItem('Admin-Token') || '',
  },
  getters: {
    // 在这里定义你的 getters
    getToken: state => state.token,
  },
  mutations: {
    // 在这里定义你的 mutations
    setToken(state, token) {
      state.token = token
      localStorage.setItem('Admin-Token', token)
    },
    removeToken(state) {
      state.token = ''
      localStorage.removeItem('Admin-Token')
    },
  },
  actions: {
    // 在这里定义你的 actions
    login({ commit }, token) {
      commit('setToken', token)
    },
    logout({ commit }) {
      commit('removeToken')
    },
  },
  modules: {
    // 在这里定义你的模块
  }
})
