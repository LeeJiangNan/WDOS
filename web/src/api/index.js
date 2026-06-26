import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'

// API 在后端 9090 端口，前端在 5173，需要跨域
const api = axios.create({
  baseURL: window.location.protocol + '//' + window.location.hostname + ':9090/api/v1',
  timeout: 10000
})

// 请求拦截器 — 加 Token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('wdos_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器 — 统一错误处理
api.interceptors.response.use(
  res => {
    if (res.data.code !== 0) {
      ElMessage.error(res.data.message || '请求失败')
      return Promise.reject(res.data)
    }
    return res.data
  },
  err => {
    if (err.response?.status === 401) {
      localStorage.removeItem('wdos_token')
      router.push('/login')
    }
    ElMessage.error(err.response?.data?.message || '网络错误')
    return Promise.reject(err)
  }
)

export default api
